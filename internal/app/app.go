package app

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/ramisoul84/kfc-crm/internal/client"
	"github.com/ramisoul84/kfc-crm/internal/config"
	"github.com/ramisoul84/kfc-crm/internal/events"
	"github.com/ramisoul84/kfc-crm/internal/repository"
	"github.com/ramisoul84/kfc-crm/internal/service"
	grpcTransport "github.com/ramisoul84/kfc-crm/internal/transport/grpc"
	httpTransport "github.com/ramisoul84/kfc-crm/internal/transport/http"
	"github.com/ramisoul84/kfc-crm/internal/transport/http/handler"
	"github.com/ramisoul84/kfc-crm/migrations"
	"github.com/ramisoul84/kfc-crm/pkg/database"
	"github.com/ramisoul84/kfc-crm/pkg/jwt"
	"github.com/ramisoul84/kfc-crm/pkg/kafka"
	"github.com/ramisoul84/kfc-crm/pkg/logger"
	"github.com/ramisoul84/kfc-crm/pkg/migrate"
	"github.com/ramisoul84/kfc-crm/pkg/outbox"
	"github.com/ramisoul84/kfc-crm/pkg/storage"
	"github.com/ramisoul84/kfc-crm/pkg/validator"
)

// App wires together all application components.
type App struct {
	config             *config.Config
	logger             *logger.Logger
	db                 *database.Postgres
	redis              *database.Redis
	notificationClient client.NotificationClient
	orderClient        client.OrderClient
	loyaltyClient      client.LoyaltyClient
	server             *httpTransport.Server
	grpcServer         *grpcTransport.Transport
	kafkaProducer      *kafka.Producer
	relay              *outbox.Relay
}

// New creates and wires the application.
func New(cfg *config.Config) (*App, error) {
	log := logger.New(&logger.Config{
		Level:    cfg.Logger.Level,
		Format:   cfg.Logger.Format,
		Output:   cfg.Logger.Output,
		FilePath: cfg.Logger.FilePath,
		Service:  cfg.Logger.Service,
	})

	log.Info("starting application",
		"name", cfg.App.Name,
		"version", cfg.App.Version,
		"environment", cfg.App.Environment,
	)

	// Postgres
	db, err := database.NewPostgres(&cfg.DB, cfg.DSN())
	if err != nil {
		return nil, fmt.Errorf("database: %w", err)
	}
	log.Info("postgres connected")

	if cfg.DB.AutoMigrate {
		applied, err := migrate.Up(context.Background(), db.DB, migrations.FS)
		if err != nil {
			db.Close()
			return nil, fmt.Errorf("migrate: %w", err)
		}
		log.Info("migrations up to date", "applied", applied)
	}

	photoStorage, err := storage.NewMinIOStorage(cfg.MinIO)
	if err != nil {
		return nil, fmt.Errorf("minio storage: %w", err)
	}
	log.Info("minio storage ready",
		"endpoint", cfg.MinIO.Endpoint,
		"bucket", cfg.MinIO.Bucket,
	)

	// Kafka via the outbox: menu events are recorded in Postgres and
	// relayed, so a Kafka outage delays cache invalidation instead of
	// losing it.
	var kafkaProducer *kafka.Producer
	var relay *outbox.Relay
	notify := func() {}
	if len(cfg.Kafka.Brokers) > 0 {
		kafkaProducer = kafka.NewProducer(cfg.Kafka.Brokers, log)
		relay = outbox.NewRelay(db.DB, kafkaProducer, outboxLockID, log)
		notify = relay.Notify
		log.Info("outbox relay initialized", "brokers", cfg.Kafka.Brokers)
	} else {
		log.Warn("kafka disabled; menu events accumulate in the outbox until it is enabled")
	}
	eventsPublisher := events.NewPublisher(db.DB, notify, log)

	// Redis
	redisClient, err := database.NewRedis(&cfg.Redis)
	if err != nil {
		db.Close()
		return nil, fmt.Errorf("redis: %w", err)
	}
	log.Info("redis connected")

	// JWT
	tokenManager := jwt.NewTokenManager(
		cfg.JWT.Secret,
		cfg.JWT.AccessDuration,
		cfg.JWT.RefreshDuration,
	)

	// ─────────────────────────────────────────────────────────────────
	// gRPC clients to peer services
	// ─────────────────────────────────────────────────────────────────

	// Notification client
	notificationClient, err := buildNotificationClient(cfg, log)
	if err != nil {
		return nil, err
	}

	// UserAPI client
	userAPIClient, err := buildUserAPIClient(cfg, log)
	if err != nil {
		return nil, err
	}

	// Order client (lazy: kfc-order may start after CRM)
	orderClient, err := client.NewOrderClient(&cfg.Order)
	if err != nil {
		return nil, fmt.Errorf("order client: %w", err)
	}
	log.Info("order client initialized", "address", cfg.Order.Address)

	loyaltyClient, err := client.NewLoyaltyClient(&cfg.Loyalty)
	if err != nil {
		return nil, fmt.Errorf("loyalty client: %w", err)
	}
	log.Info("loyalty client initialized", "address", cfg.Loyalty.Address)

	// Repositories
	userRepo := repository.NewUserRepository(db.DB)
	tokenRepo := repository.NewTokenRepository(redisClient.Client)
	regionRepo := repository.NewRegionRepository(db.DB)
	restaurantRepo := repository.NewRestaurantRepository(db.DB)
	deviceRepo := repository.NewDeviceRepository(db.DB)
	menuCategoryRepo := repository.NewMenuCategoryRepository(db.DB)
	menuItemRepo := repository.NewMenuItemRepository(db.DB)
	menuVariationRepo := repository.NewMenuItemVariationRepository(db.DB)
	menuOverrideRepo := repository.NewMenuItemOverrideRepository(db.DB)
	menuPromotionRepo := repository.NewMenuPromotionRepository(db.DB)
	effectiveMenuRepo := repository.NewEffectiveMenuRepository(db.DB)

	// RBAC service
	rbacService := service.NewRBACService(regionRepo, restaurantRepo, deviceRepo)

	// Services
	authService := service.NewAuthService(userRepo, tokenRepo, tokenManager, log)
	regionService := service.NewRegionService(regionRepo, rbacService, log)
	restaurantService := service.NewRestaurantService(restaurantRepo, rbacService, log)
	userService := service.NewUserService(userRepo, rbacService, notificationClient, log)
	deviceService := service.NewDeviceService(deviceRepo, restaurantRepo, rbacService, notificationClient, userAPIClient, log)
	menuCategoryService := service.NewMenuCategoryService(
		menuCategoryRepo,
		photoStorage,
		rbacService,
		eventsPublisher,
		log,
	)
	menuItemService := service.NewMenuItemService(
		menuItemRepo,
		menuCategoryRepo,
		photoStorage,
		rbacService,
		eventsPublisher,
		log,
	)
	menuItemVariationService := service.NewMenuItemVariationService(
		menuVariationRepo,
		menuItemRepo,
		rbacService,
		eventsPublisher,
		log,
	)
	menuItemOverrideService := service.NewMenuItemOverrideService(
		menuOverrideRepo,
		menuItemRepo,
		rbacService,
		eventsPublisher,
		log,
	)
	menuPromotionService := service.NewMenuPromotionService(menuPromotionRepo, rbacService, eventsPublisher, log)
	effectiveMenuService := service.NewEffectiveMenuService(
		effectiveMenuRepo,
		restaurantRepo,
		rbacService,
		log,
	)

	orderService := service.NewOrderService(orderClient, restaurantRepo, rbacService, log)
	customerService := service.NewCustomerService(loyaltyClient, rbacService, log)

	// Validator
	v := validator.New()

	// Handlers
	authHandler := handler.NewAuthHandler(authService, v, cfg.IsProduction())
	regionHandler := handler.NewRegionHandler(regionService, v)
	restaurantHandler := handler.NewRestaurantHandler(restaurantService, v)
	userHandler := handler.NewUserHandler(userService, v)
	deviceHandler := handler.NewDeviceHandler(deviceService, v)
	menuCategoryHandler := handler.NewMenuCategoryHandler(menuCategoryService, v)
	menuItemHandler := handler.NewMenuItemHandler(menuItemService, v)
	menuItemVariationHandler := handler.NewMenuItemVariationHandler(menuItemVariationService, v)
	menuItemOverrideHandler := handler.NewMenuItemOverrideHandler(menuItemOverrideService, v)
	menuPromotionHandler := handler.NewMenuPromotionHandler(menuPromotionService, v)
	effectiveMenuHandler := handler.NewEffectiveMenuHandler(effectiveMenuService)

	businessLoc, err := time.LoadLocation(cfg.DB.Timezone)
	if err != nil {
		return nil, fmt.Errorf("load timezone %q: %w", cfg.DB.Timezone, err)
	}
	orderHandler := handler.NewOrderHandler(orderService, v, businessLoc)
	customerHandler := handler.NewCustomerHandler(customerService, v)

	// Server
	server := httpTransport.NewServer(
		cfg,
		log,
		authHandler,
		regionHandler,
		restaurantHandler,
		userHandler,
		deviceHandler,
		menuCategoryHandler,
		menuItemHandler,
		menuItemVariationHandler,
		menuItemOverrideHandler,
		menuPromotionHandler,
		effectiveMenuHandler,
		orderHandler,
		customerHandler,
		tokenManager,
		tokenRepo,
		userRepo,
	)

	// gRPC server (NEW)
	menuServer := grpcTransport.NewMenuServer(effectiveMenuService, log)
	restaurantServer := grpcTransport.NewRestaurantServer(restaurantRepo, regionRepo, log)
	grpcServer := grpcTransport.NewTransport(cfg, log, menuServer, restaurantServer)

	return &App{
		config:             cfg,
		logger:             log,
		db:                 db,
		redis:              redisClient,
		notificationClient: notificationClient,
		orderClient:        orderClient,
		loyaltyClient:      loyaltyClient,
		server:             server,
		grpcServer:         grpcServer,
		kafkaProducer:      kafkaProducer,
		relay:              relay,
	}, nil
}

// // Start begins serving both HTTP and gRPC.
func (a *App) Start(ctx context.Context) error {
	a.logger.Info("starting transports")

	errCh := make(chan error, 2)

	if a.relay != nil {
		go a.relay.Run(ctx)
	}

	go func() {
		if err := a.server.Start(); err != nil {
			errCh <- fmt.Errorf("http: %w", err)
		}
	}()
	go func() {
		if err := a.grpcServer.Start(); err != nil {
			errCh <- fmt.Errorf("grpc: %w", err)
		}
	}()

	select {
	case <-ctx.Done():
		a.logger.Info("shutdown signal received")
		return nil
	case err := <-errCh:
		a.logger.Error("transport failed", "error", err)
		return err
	}
}

// Shutdown gracefully stops all transports and releases resources.
func (a *App) Shutdown(ctx context.Context) error {
	a.logger.Info("shutting down application")

	var errs []error

	if err := a.server.Shutdown(ctx); err != nil {
		a.logger.Error("http shutdown failed", "error", err)
		errs = append(errs, fmt.Errorf("http shutdown: %w", err))
	}
	if err := a.grpcServer.Shutdown(ctx); err != nil {
		a.logger.Error("grpc shutdown failed", "error", err)
		errs = append(errs, fmt.Errorf("grpc shutdown: %w", err))
	}

	if a.kafkaProducer != nil {
		if err := a.kafkaProducer.Close(); err != nil {
			a.logger.Error("kafka producer close failed", "error", err)
			errs = append(errs, err)
		}
	}

	if a.notificationClient != nil {
		_ = a.notificationClient.Close()
	}
	if a.orderClient != nil {
		_ = a.orderClient.Close()
	}
	if a.loyaltyClient != nil {
		_ = a.loyaltyClient.Close()
	}
	if a.redis != nil {
		_ = a.redis.Close()
	}
	if a.db != nil {
		_ = a.db.Close()
	}

	a.logger.Info("shutdown complete")
	return errors.Join(errs...)
}

// buildNotificationClient constructs the notification client, or a no-op
// when disabled. In production, construction failures are fatal; elsewhere
// they fall back to a no-op so the service still starts.
func buildNotificationClient(cfg *config.Config, log *logger.Logger) (client.NotificationClient, error) {
	if !cfg.Notification.Enabled {
		log.Info("notification client disabled, using no-op")
		return client.NewNoopNotificationClient(log), nil
	}

	nc, err := client.NewNotificationClient(&cfg.Notification)
	if err != nil {
		if cfg.IsProduction() {
			return nil, fmt.Errorf("notification client: %w", err)
		}
		log.Warn("notification client unavailable, using no-op",
			"address", cfg.Notification.Address,
			"error", err,
		)
		return client.NewNoopNotificationClient(log), nil
	}

	log.Info("notification client initialized", "address", cfg.Notification.Address)
	return nc, nil
}

// buildUserAPIClient constructs the userapi client, or a no-op when
// disabled. Same fail-fast rules as buildNotificationClient.
func buildUserAPIClient(cfg *config.Config, log *logger.Logger) (client.UserAPIClient, error) {
	if !cfg.UserAPI.Enabled {
		log.Info("userapi client disabled, using no-op")
		return client.NewNoopUserAPIClient(log), nil
	}

	uc, err := client.NewUserAPIClient(&cfg.UserAPI)
	if err != nil {
		if cfg.IsProduction() {
			return nil, fmt.Errorf("userapi client: %w", err)
		}
		log.Warn("userapi client unavailable, using no-op",
			"address", cfg.UserAPI.Address,
			"error", err,
		)
		return client.NewNoopUserAPIClient(log), nil
	}

	log.Info("userapi client initialized", "address", cfg.UserAPI.Address)
	return uc, nil
}

// outboxLockID elects one outbox relay among CRM replicas.
const outboxLockID = 7_310_455_200

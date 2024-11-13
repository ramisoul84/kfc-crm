package app

import (
	"context"
	"fmt"

	"github.com/ramisoul84/kfc-crm/internal/client"
	"github.com/ramisoul84/kfc-crm/internal/config"
	"github.com/ramisoul84/kfc-crm/internal/repository"
	"github.com/ramisoul84/kfc-crm/internal/service"
	httpTransport "github.com/ramisoul84/kfc-crm/internal/transport/http"
	"github.com/ramisoul84/kfc-crm/internal/transport/http/handler"
	"github.com/ramisoul84/kfc-crm/pkg/database"
	"github.com/ramisoul84/kfc-crm/pkg/jwt"
	"github.com/ramisoul84/kfc-crm/pkg/logger"
	"github.com/ramisoul84/kfc-crm/pkg/validator"
)

// App wires together all application components.
type App struct {
	config             *config.Config
	logger             *logger.Logger
	db                 *database.Postgres
	redis              *database.Redis
	notificationClient client.NotificationClient
	server             *httpTransport.Server
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

	// Notification client
	var notificationClient client.NotificationClient

	if cfg.Notification.Enabled {
		nc, err := client.NewNotificationClient(&cfg.Notification)
		if err != nil {
			if cfg.IsProduction() {
				return nil, fmt.Errorf("notification client: %w", err)
			}
			log.Warn("notification client unavailable, using no-op",
				"address", cfg.Notification.Address,
				"error", err,
			)
			notificationClient = client.NewNoopNotificationClient(log)
		} else {
			notificationClient = nc
			log.Info("notification client initialized", "address", cfg.Notification.Address)
		}
	} else {
		log.Info("notification client disabled, using no-op")
		notificationClient = client.NewNoopNotificationClient(log)
	}

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

	// RBAC service
	rbacService := service.NewRBACService(regionRepo, restaurantRepo, deviceRepo)

	// Services
	authService := service.NewAuthService(userRepo, tokenRepo, tokenManager, log)
	regionService := service.NewRegionService(regionRepo, rbacService, log)
	restaurantService := service.NewRestaurantService(restaurantRepo, rbacService, log)
	userService := service.NewUserService(userRepo, rbacService, notificationClient, log)
	deviceService := service.NewDeviceService(deviceRepo, rbacService, log)
	menuCategoryService := service.NewMenuCategoryService(menuCategoryRepo, rbacService, log)
	menuItemService := service.NewMenuItemService(menuItemRepo, menuCategoryRepo, rbacService, log)
	menuItemVariationService := service.NewMenuItemVariationService(
		menuVariationRepo,
		menuItemRepo,
		rbacService,
		log,
	)
	menuItemOverrideService := service.NewMenuItemOverrideService(
		menuOverrideRepo,
		menuItemRepo,
		rbacService,
		log,
	)
	menuPromotionService := service.NewMenuPromotionService(menuPromotionRepo, rbacService, log)

	// Validator
	v := validator.New()

	// Handlers
	authHandler := handler.NewAuthHandler(authService, v)
	regionHandler := handler.NewRegionHandler(regionService, v)
	restaurantHandler := handler.NewRestaurantHandler(restaurantService, v)
	userHandler := handler.NewUserHandler(userService, v)
	deviceHandler := handler.NewDeviceHandler(deviceService, v)
	menuCategoryHandler := handler.NewMenuCategoryHandler(menuCategoryService, v)
	menuItemHandler := handler.NewMenuItemHandler(menuItemService, v)
	menuItemVariationHandler := handler.NewMenuItemVariationHandler(menuItemVariationService, v)
	menuItemOverrideHandler := handler.NewMenuItemOverrideHandler(menuItemOverrideService, v)
	menuPromotionHandler := handler.NewMenuPromotionHandler(menuPromotionService, v)

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
		tokenManager,
		tokenRepo,
		userRepo,
	)

	return &App{
		config:             cfg,
		logger:             log,
		db:                 db,
		redis:              redisClient,
		notificationClient: notificationClient,
		server:             server,
	}, nil
}

// Start begins serving HTTP requests.
func (a *App) Start() error {
	return a.server.Start()
}

// Shutdown gracefully stops the application.
func (a *App) Shutdown(ctx context.Context) error {
	a.logger.Info("shutting down application")

	if err := a.server.Shutdown(ctx); err != nil {
		a.logger.Error("server shutdown failed", "error", err)
	}
	if a.redis != nil {
		_ = a.redis.Close()
	}
	if a.db != nil {
		_ = a.db.Close()
	}

	if a.notificationClient != nil {
		_ = a.notificationClient.Close()
	}

	a.logger.Info("shutdown complete")
	return nil
}

package http

import (
	"context"
	"fmt"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/recover"

	"github.com/ramisoul84/kfc-crm/internal/config"
	"github.com/ramisoul84/kfc-crm/internal/repository"
	"github.com/ramisoul84/kfc-crm/internal/transport/http/handler"
	"github.com/ramisoul84/kfc-crm/internal/transport/http/middleware"
	"github.com/ramisoul84/kfc-crm/pkg/ctxutil"
	"github.com/ramisoul84/kfc-crm/pkg/jwt"
	"github.com/ramisoul84/kfc-crm/pkg/logger"
)

// Server wraps the Fiber app with lifecycle methods.
type Server struct {
	app    *fiber.App
	cfg    *config.Config
	logger *logger.Logger

	authHandler       *handler.AuthHandler
	regionHandler     *handler.RegionHandler
	restaurantHandler *handler.RestaurantHandler
	userHandler       *handler.UserHandler
	deviceHandler     *handler.DeviceHandler

	menuCategoryHandler      *handler.MenuCategoryHandler
	menuItemHandler          *handler.MenuItemHandler
	menuItemVariationHandler *handler.MenuItemVariationHandler

	tokenManager *jwt.TokenManager
	tokenRepo    repository.TokenRepository
	userRepo     repository.UserRepository
}

// NewServer creates a Fiber app configured from cfg.
func NewServer(
	cfg *config.Config,
	log *logger.Logger,
	authHandler *handler.AuthHandler,
	regionHandler *handler.RegionHandler,
	restaurantHandler *handler.RestaurantHandler,
	userHandler *handler.UserHandler,
	deviceHandler *handler.DeviceHandler,
	menuCategoryHandler *handler.MenuCategoryHandler,
	menuItemHandler *handler.MenuItemHandler,
	menuItemVariationHandler *handler.MenuItemVariationHandler,
	tokenManager *jwt.TokenManager,
	tokenRepo repository.TokenRepository,
	userRepo repository.UserRepository,
) *Server {
	app := fiber.New(fiber.Config{
		AppName:               cfg.App.Name,
		ReadTimeout:           cfg.HTTP.ReadTimeout,
		WriteTimeout:          cfg.HTTP.WriteTimeout,
		IdleTimeout:           cfg.HTTP.IdleTimeout,
		DisableStartupMessage: true,
		ErrorHandler:          errorHandler(log),
	})

	s := &Server{
		app:                      app,
		cfg:                      cfg,
		logger:                   log,
		authHandler:              authHandler,
		regionHandler:            regionHandler,
		restaurantHandler:        restaurantHandler,
		userHandler:              userHandler,
		deviceHandler:            deviceHandler,
		menuCategoryHandler:      menuCategoryHandler,
		menuItemHandler:          menuItemHandler,
		menuItemVariationHandler: menuItemVariationHandler,
		tokenManager:             tokenManager,
		tokenRepo:                tokenRepo,
		userRepo:                 userRepo,
	}

	s.registerMiddleware()
	s.registerRoutes()

	return s
}

// App returns the underlying Fiber app.
func (s *Server) App() *fiber.App {
	return s.app
}

// Start begins listening for HTTP requests. Blocks until shutdown.
func (s *Server) Start() error {
	addr := ":" + s.cfg.HTTP.Port
	s.logger.Info("http server listening", "addr", addr)
	if err := s.app.Listen(addr); err != nil {
		return fmt.Errorf("http server failed: %w", err)
	}
	return nil
}

// Shutdown gracefully shuts down the server.
func (s *Server) Shutdown(ctx context.Context) error {
	return s.app.ShutdownWithContext(ctx)
}

// ═══════════════════════════════════════════════════════════════════
// MIDDLEWARE
// ═══════════════════════════════════════════════════════════════════

func (s *Server) registerMiddleware() {
	s.app.Use(middleware.RequestID())
	s.app.Use(middleware.Logging(s.logger))
	s.app.Use(recover.New())
	s.app.Use(cors.New(cors.Config{
		AllowOrigins: "*",
		AllowMethods: "GET,POST,PUT,DELETE,PATCH,OPTIONS",
		AllowHeaders: "Origin,Content-Type,Accept,Authorization,X-Request-ID",
	}))
}

// ═══════════════════════════════════════════════════════════════════
// ROUTES
// ═══════════════════════════════════════════════════════════════════

func (s *Server) registerRoutes() {
	// Health check (public)
	s.app.Get("/health", s.healthCheck)

	// API v1
	api := s.app.Group("/api/v1")

	// ── Auth (public) ──
	auth := api.Group("/auth")
	auth.Post("/login", s.authHandler.Login)
	auth.Post("/refresh", s.authHandler.RefreshToken)

	// Logout requires a valid access token
	auth.Post("/logout",
		middleware.Auth(s.tokenManager, s.tokenRepo, s.userRepo),
		s.authHandler.Logout,
	)

	// ── Protected routes ──
	protected := api.Group("", middleware.Auth(s.tokenManager, s.tokenRepo, s.userRepo))

	// Regions
	regions := protected.Group("/regions")
	regions.Post("/", s.regionHandler.Create)
	regions.Get("/", s.regionHandler.List)
	regions.Get("/:id", s.regionHandler.GetByID)
	regions.Put("/:id", s.regionHandler.Update)
	regions.Delete("/:id", s.regionHandler.Delete)

	// Restaurants
	restaurants := protected.Group("/restaurants")
	restaurants.Post("/", s.restaurantHandler.Create)
	restaurants.Get("/", s.restaurantHandler.List)
	restaurants.Get("/:id", s.restaurantHandler.GetByID)
	restaurants.Put("/:id", s.restaurantHandler.Update)
	restaurants.Delete("/:id", s.restaurantHandler.Delete)

	// Self-service FIRST so "me" doesn't match ":id"
	me := protected.Group("/users/me")
	me.Post("/setup", s.userHandler.Setup)
	me.Put("/password", s.userHandler.ChangePassword)

	// Then dynamic routes
	users := protected.Group("/users")
	users.Post("/", s.userHandler.Create)
	users.Get("/", s.userHandler.List)
	users.Get("/:id", s.userHandler.GetByID)
	users.Put("/:id", s.userHandler.Update)
	users.Delete("/:id", s.userHandler.Delete)

	// Devices
	devices := protected.Group("/devices")
	devices.Post("/", s.deviceHandler.Create)
	devices.Get("/", s.deviceHandler.List)
	devices.Get("/:id", s.deviceHandler.GetByID)
	devices.Put("/:id", s.deviceHandler.Update)
	devices.Delete("/:id", s.deviceHandler.Delete)

	// Menu
	menu := protected.Group("/menu")

	categories := menu.Group("/categories")
	categories.Post("/", s.menuCategoryHandler.Create)
	categories.Get("/", s.menuCategoryHandler.List)
	categories.Get("/:id", s.menuCategoryHandler.GetByID)
	categories.Put("/:id", s.menuCategoryHandler.Update)
	categories.Delete("/:id", s.menuCategoryHandler.Delete)

	// Items
	items := menu.Group("/items")
	items.Post("/", s.menuItemHandler.Create)
	items.Get("/", s.menuItemHandler.List)
	items.Get("/:id", s.menuItemHandler.GetByID)
	items.Put("/:id", s.menuItemHandler.Update)
	items.Delete("/:id", s.menuItemHandler.Delete)

	// Variations
	variations := menu.Group("/variations")
	variations.Post("/", s.menuItemVariationHandler.Create)
	variations.Get("/:id", s.menuItemVariationHandler.GetByID)
	variations.Put("/:id", s.menuItemVariationHandler.Update)
	variations.Delete("/:id", s.menuItemVariationHandler.Delete)

	// Variations for a specific item
	menu.Get("/items/:item_id/variations", s.menuItemVariationHandler.ListByItem)

}

// ═══════════════════════════════════════════════════════════════════
// HANDLERS
// ═══════════════════════════════════════════════════════════════════

func (s *Server) healthCheck(c *fiber.Ctx) error {
	return c.JSON(fiber.Map{
		"status": "ok",
		"time":   time.Now().Format(time.RFC3339),
	})
}

// ═══════════════════════════════════════════════════════════════════
// ERROR HANDLER
// ═══════════════════════════════════════════════════════════════════

func errorHandler(log *logger.Logger) fiber.ErrorHandler {
	return func(c *fiber.Ctx, err error) error {
		code := fiber.StatusInternalServerError
		if e, ok := err.(*fiber.Error); ok {
			code = e.Code
		}

		requestID := ctxutil.GetRequestIDFromFiber(c)

		log.WithRequestID(requestID).Error("unhandled error",
			"method", c.Method(),
			"path", c.Path(),
			"status", code,
			"error", err.Error(),
		)

		return c.Status(code).JSON(fiber.Map{
			"success": false,
			"error": fiber.Map{
				"type":    "internal",
				"message": err.Error(),
			},
			"request_id": requestID,
			"timestamp":  time.Now().Unix(),
		})
	}
}

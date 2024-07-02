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
	app          *fiber.App
	cfg          *config.Config
	logger       *logger.Logger
	authHandler  *handler.AuthHandler
	tokenManager *jwt.TokenManager
	tokenRepo    repository.TokenRepository
}

// NewServer creates a Fiber app configured from cfg.
func NewServer(
	cfg *config.Config,
	log *logger.Logger,
	authHandler *handler.AuthHandler,
	tokenManager *jwt.TokenManager,
	tokenRepo repository.TokenRepository,
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
		app:          app,
		cfg:          cfg,
		logger:       log,
		authHandler:  authHandler,
		tokenManager: tokenManager,
		tokenRepo:    tokenRepo,
	}

	// Register middleware first, then routes
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
		middleware.Auth(s.tokenManager, s.tokenRepo),
		s.authHandler.Logout,
	)
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

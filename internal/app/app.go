package app

import (
	"context"

	"github.com/ramisoul84/kfc-crm/internal/config"
	httpTransport "github.com/ramisoul84/kfc-crm/internal/transport/http"
	"github.com/ramisoul84/kfc-crm/pkg/logger"
)

// App wires together all application components.
type App struct {
	config *config.Config
	logger *logger.Logger
	server *httpTransport.Server
}

// New creates and wires the application.
func New(cfg *config.Config) (*App, error) {
	log := logger.New(&logger.Config{
		Level:    cfg.Logger.Level,
		Format:   cfg.Logger.Format,
		Output:   cfg.Logger.Output,
		FilePath: "logs/app.log",
		Service:  cfg.Logger.Service,
	})

	log.Info("starting application",
		"name", cfg.App.Name,
		"version", cfg.App.Version,
		"environment", cfg.App.Environment,
	)

	// Create HTTP server
	server := httpTransport.NewServer(cfg, log)

	return &App{
		config: cfg,
		logger: log,
		server: server,
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
		a.logger.Error("http shutdown failed", "error", err)
		return err
	}

	a.logger.Info("shutdown complete")
	return nil
}

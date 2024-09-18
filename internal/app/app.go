package app

import (
	"context"

	"github.com/ramisoul84/kfc-crm/internal/config"
	httpTransport "github.com/ramisoul84/kfc-crm/internal/transport/http"
)

// App wires together all application components.
type App struct {
	config *config.Config
	server *httpTransport.Server
}

// New creates and wires the application.
func New(cfg *config.Config) (*App, error) {
	// Create HTTP server
	server := httpTransport.NewServer(cfg)

	return &App{
		config: cfg,
		server: server,
	}, nil
}

// Start begins serving HTTP requests.
func (a *App) Start() error {
	return a.server.Start()
}

// Shutdown gracefully stops the application.
func (a *App) Shutdown(ctx context.Context) error {
	return a.server.Shutdown(ctx)
}

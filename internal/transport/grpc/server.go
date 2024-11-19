package grpc

import (
	"context"
	"fmt"
	"net"

	"google.golang.org/grpc"

	menuv1 "github.com/ramisoul84/kfc-crm/gen/menu/v1"
	"github.com/ramisoul84/kfc-crm/internal/config"
	"github.com/ramisoul84/kfc-crm/pkg/logger"
)

// Transport wraps the gRPC server with lifecycle methods.
// It is a peer of the HTTP server; the App owns both.
type Transport struct {
	srv    *grpc.Server
	cfg    *config.Config
	logger *logger.Logger
}

// NewTransport creates a gRPC server and registers all gRPC services.
func NewTransport(
	cfg *config.Config,
	log *logger.Logger,
	menuServer *MenuServer,
) *Transport {
	srv := grpc.NewServer(
	// Interceptors (service-token auth, request-id, logging, recovery)
	// will be added here in a follow-up step.
	)

	// Register all gRPC services here.
	menuv1.RegisterMenuServiceServer(srv, menuServer)

	return &Transport{
		srv:    srv,
		cfg:    cfg,
		logger: log,
	}
}

// Start begins serving gRPC. Blocks until Serve returns.
func (t *Transport) Start() error {
	addr := ":" + t.cfg.GRPC.Port
	lis, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("grpc listen %s: %w", addr, err)
	}

	t.logger.Info("grpc server listening", "addr", addr)
	if err := t.srv.Serve(lis); err != nil {
		return fmt.Errorf("grpc serve: %w", err)
	}
	return nil
}

// Shutdown gracefully stops the gRPC server, respecting ctx.
func (t *Transport) Shutdown(ctx context.Context) error {
	done := make(chan struct{})
	go func() {
		t.srv.GracefulStop()
		close(done)
	}()

	select {
	case <-done:
		return nil
	case <-ctx.Done():
		t.srv.Stop() // hard stop after timeout
		return ctx.Err()
	}
}

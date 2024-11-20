package client

import (
	"context"
	"fmt"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	notificationv1 "github.com/ramisoul84/kfc-crm/gen/notification/v1"
	"github.com/ramisoul84/kfc-crm/internal/config"
)

// NotificationClient sends notifications to the notification service.
type NotificationClient interface {
	SendWelcomeEmail(ctx context.Context, req *WelcomeEmail) error
	Close() error
}

// ─────────────────────────────────────────────────────────────────
// Request DTOs
// ─────────────────────────────────────────────────────────────────

type WelcomeEmail struct {
	To              string
	Email           string
	RoleName        string
	InitialPassword string
	RestaurantID    string
	RegionID        string
}

// ─────────────────────────────────────────────────────────────────
// Implementation
// ─────────────────────────────────────────────────────────────────

type notificationClient struct {
	client  notificationv1.NotificationServiceClient
	conn    *grpc.ClientConn
	timeout time.Duration
}

// NewNotificationClient creates a gRPC client for the notification service.
func NewNotificationClient(cfg *config.NotificationConfig) (NotificationClient, error) {
	conn, err := grpc.NewClient(
		cfg.Address,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		return nil, fmt.Errorf("create notification grpc client: %w", err)
	}

	return &notificationClient{
		client:  notificationv1.NewNotificationServiceClient(conn),
		conn:    conn,
		timeout: cfg.Timeout,
	}, nil
}

func (c *notificationClient) SendWelcomeEmail(ctx context.Context, req *WelcomeEmail) error {
	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	resp, err := c.client.SendWelcomeEmail(ctx, &notificationv1.SendWelcomeEmailRequest{
		To:              req.To,
		Email:           req.Email,
		RoleName:        req.RoleName,
		InitialPassword: req.InitialPassword,
		RestaurantId:    req.RestaurantID,
		RegionId:        req.RegionID,
	})
	if err != nil {
		return fmt.Errorf("send welcome email: %w", err)
	}
	if !resp.Success {
		return fmt.Errorf("notification service rejected welcome email: %s", resp.Message)
	}
	return nil
}

func (c *notificationClient) Close() error {
	if c.conn != nil {
		return c.conn.Close()
	}
	return nil
}

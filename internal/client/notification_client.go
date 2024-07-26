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
	SendPasswordResetEmail(ctx context.Context, req *PasswordResetEmail) error
	SendEmail(ctx context.Context, req *GenericEmail) error
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

type PasswordResetEmail struct {
	To          string
	Email       string
	NewPassword string
}

type GenericEmail struct {
	To      string
	Subject string
	Body    string
	IsHTML  bool
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

func (c *notificationClient) SendPasswordResetEmail(ctx context.Context, req *PasswordResetEmail) error {
	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	resp, err := c.client.SendPasswordResetEmail(ctx, &notificationv1.SendPasswordResetEmailRequest{
		To:          req.To,
		Email:       req.Email,
		NewPassword: req.NewPassword,
	})
	if err != nil {
		return fmt.Errorf("send password reset email: %w", err)
	}
	if !resp.Success {
		return fmt.Errorf("notification service rejected password reset email: %s", resp.Message)
	}
	return nil
}

func (c *notificationClient) SendEmail(ctx context.Context, req *GenericEmail) error {
	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	resp, err := c.client.SendEmail(ctx, &notificationv1.SendEmailRequest{
		To:      req.To,
		Subject: req.Subject,
		Body:    req.Body,
		IsHtml:  req.IsHTML,
	})
	if err != nil {
		return fmt.Errorf("send email: %w", err)
	}
	if !resp.Success {
		return fmt.Errorf("notification service rejected email: %s", resp.Message)
	}
	return nil
}

func (c *notificationClient) Close() error {
	if c.conn != nil {
		return c.conn.Close()
	}
	return nil
}

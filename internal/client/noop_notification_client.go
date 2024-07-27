package client

import (
	"context"

	"github.com/ramisoul84/kfc-crm/pkg/logger"
)

// NoopNotificationClient logs notifications instead of sending them.
// Used in development when the notification service is not running.
type NoopNotificationClient struct {
	logger *logger.Logger
}

// NewNoopNotificationClient creates a NoopNotificationClient.
func NewNoopNotificationClient(log *logger.Logger) *NoopNotificationClient {
	return &NoopNotificationClient{logger: log}
}

func (c *NoopNotificationClient) SendWelcomeEmail(ctx context.Context, req *WelcomeEmail) error {
	c.logger.Warn("notification (no-op): welcome email not sent",
		"email", req.Email,
		"role", req.RoleName,
		"initial_password", req.InitialPassword,
	)
	return nil
}

func (c *NoopNotificationClient) SendPasswordResetEmail(ctx context.Context, req *PasswordResetEmail) error {
	c.logger.Warn("notification (no-op): password reset email not sent",
		"email", req.Email,
		"new_password", req.NewPassword,
	)
	return nil
}

func (c *NoopNotificationClient) SendEmail(ctx context.Context, req *GenericEmail) error {
	c.logger.Warn("notification (no-op): generic email not sent",
		"to", req.To,
		"subject", req.Subject,
	)
	return nil
}

func (c *NoopNotificationClient) Close() error {
	return nil
}

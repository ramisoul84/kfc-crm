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
		"to", req.To,
		"email", req.Email,
		"role", req.RoleName,
		"initial_password", req.InitialPassword,
	)
	return nil
}

func (c *NoopNotificationClient) SendDevicePairingEmail(ctx context.Context, req *DevicePairingEmail) error {
	c.logger.Warn("notification (no-op): device pairing email not sent",
		"to", req.To,
		"restaurant_id", req.RestaurantID,
		"restaurant_name", req.RestaurantName,
		"expires_at", req.ExpiresAt,
		"device_count", len(req.Devices),
	)

	// Log each code so you can pair devices by hand in dev.
	for _, d := range req.Devices {
		c.logger.Warn("notification (no-op): device pairing code",
			"serial_number", d.SerialNumber,
			"device_type", d.DeviceType,
			"code", d.Code,
		)
	}

	return nil
}

func (c *NoopNotificationClient) Close() error {
	return nil
}

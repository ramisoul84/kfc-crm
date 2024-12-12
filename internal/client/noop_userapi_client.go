package client

import (
	"context"

	"github.com/ramisoul84/kfc-crm/pkg/logger"
)

// NoopUserAPIClient logs pairing code operations instead of calling userapi.
// Used in development when the userapi service is not running.
type NoopUserAPIClient struct {
	logger *logger.Logger
}

func NewNoopUserAPIClient(log *logger.Logger) *NoopUserAPIClient {
	return &NoopUserAPIClient{logger: log}
}

func (c *NoopUserAPIClient) CachePairingCodes(ctx context.Context, req *CachePairingCodesRequest) error {
	c.logger.Warn("userapi (no-op): pairing codes not cached",
		"restaurant_id", req.RestaurantID,
		"expires_at", req.ExpiresAt,
		"count", len(req.Codes),
	)
	for _, e := range req.Codes {
		c.logger.Warn("userapi (no-op): would cache code",
			"serial_number", e.SerialNumber,
			"device_id", e.DeviceID,
			"code", e.Code,
		)
	}
	return nil
}

func (c *NoopUserAPIClient) CachePairingCode(ctx context.Context, req *CachePairingCodeRequest) error {
	c.logger.Warn("userapi (no-op): pairing code not cached",
		"restaurant_id", req.RestaurantID,
		"serial_number", req.SerialNumber,
		"device_id", req.DeviceID,
		"code", req.Code,
		"expires_at", req.ExpiresAt,
	)
	return nil
}

func (c *NoopUserAPIClient) RevokePairingCode(ctx context.Context, serialNumber string) error {
	c.logger.Warn("userapi (no-op): pairing code not revoked",
		"serial_number", serialNumber,
	)
	return nil
}

func (c *NoopUserAPIClient) RevokeDevice(ctx context.Context, deviceID string) error {
	c.logger.Warn("userapi (no-op): device not revoked",
		"device_id", deviceID,
	)
	return nil
}

func (c *NoopUserAPIClient) Close() error {
	return nil
}

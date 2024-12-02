package client

import (
	"context"
	"fmt"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	userapiv1 "github.com/ramisoul84/kfc-crm/gen/userapi/v1"
	"github.com/ramisoul84/kfc-crm/internal/config"
)

// UserAPIClient sends pairing codes to the userapi service.
type UserAPIClient interface {
	CachePairingCodes(ctx context.Context, req *CachePairingCodesRequest) error
	CachePairingCode(ctx context.Context, req *CachePairingCodeRequest) error
	RevokePairingCode(ctx context.Context, serialNumber string) error
	Close() error
}

// ─────────────────────────────────────────────────────────────────
// DTOs
// ─────────────────────────────────────────────────────────────────

type CachePairingCodesRequest struct {
	RestaurantID string
	ExpiresAt    time.Time
	Codes        []PairingCodeEntry
}

type PairingCodeEntry struct {
	SerialNumber string
	DeviceID     string
	Code         string
}

type CachePairingCodeRequest struct {
	RestaurantID string
	SerialNumber string
	DeviceID     string
	Code         string
	ExpiresAt    time.Time
}

// ─────────────────────────────────────────────────────────────────
// Implementation
// ─────────────────────────────────────────────────────────────────

type userAPIClient struct {
	client  userapiv1.UserAPIServiceClient
	conn    *grpc.ClientConn
	timeout time.Duration
}

func NewUserAPIClient(cfg *config.UserAPIConfig) (UserAPIClient, error) {
	conn, err := grpc.NewClient(
		cfg.Address,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		return nil, fmt.Errorf("create userapi grpc client: %w", err)
	}

	return &userAPIClient{
		client:  userapiv1.NewUserAPIServiceClient(conn),
		conn:    conn,
		timeout: cfg.Timeout,
	}, nil
}

func (c *userAPIClient) CachePairingCodes(ctx context.Context, req *CachePairingCodesRequest) error {
	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	codes := make([]*userapiv1.PairingCode, 0, len(req.Codes))
	for _, e := range req.Codes {
		codes = append(codes, &userapiv1.PairingCode{
			SerialNumber: e.SerialNumber,
			DeviceId:     e.DeviceID,
			Code:         e.Code,
		})
	}

	resp, err := c.client.CachePairingCodes(ctx, &userapiv1.CachePairingCodesRequest{
		RestaurantId: req.RestaurantID,
		ExpiresAt:    req.ExpiresAt.Unix(),
		Codes:        codes,
	})
	if err != nil {
		return fmt.Errorf("cache pairing codes: %w", err)
	}
	if !resp.Success {
		return fmt.Errorf("userapi rejected pairing codes: %s", resp.Message)
	}
	return nil
}

func (c *userAPIClient) CachePairingCode(ctx context.Context, req *CachePairingCodeRequest) error {
	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	resp, err := c.client.CachePairingCode(ctx, &userapiv1.CachePairingCodeRequest{
		RestaurantId: req.RestaurantID,
		SerialNumber: req.SerialNumber,
		DeviceId:     req.DeviceID,
		Code:         req.Code,
		ExpiresAt:    req.ExpiresAt.Unix(),
	})
	if err != nil {
		return fmt.Errorf("cache pairing code: %w", err)
	}
	if !resp.Success {
		return fmt.Errorf("userapi rejected pairing code: %s", resp.Message)
	}
	return nil
}

func (c *userAPIClient) RevokePairingCode(ctx context.Context, serialNumber string) error {
	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	resp, err := c.client.RevokePairingCode(ctx, &userapiv1.RevokePairingCodeRequest{
		SerialNumber: serialNumber,
	})
	if err != nil {
		return fmt.Errorf("revoke pairing code: %w", err)
	}
	if !resp.Success {
		return fmt.Errorf("userapi rejected revoke: %s", resp.Message)
	}
	return nil
}

func (c *userAPIClient) Close() error {
	if c.conn != nil {
		return c.conn.Close()
	}
	return nil
}

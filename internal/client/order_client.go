package client

import (
	"context"
	"fmt"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	orderv1 "github.com/ramisoul84/kfc-crm/gen/order/v1"
	"github.com/ramisoul84/kfc-crm/internal/config"
)

// OrderClient is the subset of kfc-order that CRM uses: history,
// cancellation and dashboard stats. Creating orders is not CRM's job.
type OrderClient interface {
	ListOrders(ctx context.Context, req *orderv1.ListOrdersRequest) (*orderv1.ListOrdersResponse, error)
	GetOrder(ctx context.Context, id string) (*orderv1.Order, error)
	UpdateOrderStatus(ctx context.Context, req *orderv1.UpdateOrderStatusRequest) (*orderv1.Order, error)
	GetOrderStats(ctx context.Context, req *orderv1.GetOrderStatsRequest) (*orderv1.OrderStats, error)
	Close() error
}

type orderClient struct {
	client  orderv1.OrderServiceClient
	conn    *grpc.ClientConn
	timeout time.Duration
}

// NewOrderClient creates a lazily-connecting client; the order service
// does not need to be up when CRM starts.
func NewOrderClient(cfg *config.OrderConfig) (OrderClient, error) {
	conn, err := grpc.NewClient(
		cfg.Address,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		return nil, fmt.Errorf("create order grpc client: %w", err)
	}
	return &orderClient{
		client:  orderv1.NewOrderServiceClient(conn),
		conn:    conn,
		timeout: cfg.Timeout,
	}, nil
}

func (c *orderClient) ListOrders(ctx context.Context, req *orderv1.ListOrdersRequest) (*orderv1.ListOrdersResponse, error) {
	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()
	return c.client.ListOrders(ctx, req)
}

func (c *orderClient) GetOrder(ctx context.Context, id string) (*orderv1.Order, error) {
	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()
	return c.client.GetOrder(ctx, &orderv1.GetOrderRequest{Id: id})
}

func (c *orderClient) UpdateOrderStatus(ctx context.Context, req *orderv1.UpdateOrderStatusRequest) (*orderv1.Order, error) {
	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()
	return c.client.UpdateOrderStatus(ctx, req)
}

func (c *orderClient) GetOrderStats(ctx context.Context, req *orderv1.GetOrderStatsRequest) (*orderv1.OrderStats, error) {
	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()
	return c.client.GetOrderStats(ctx, req)
}

func (c *orderClient) Close() error {
	if c.conn == nil {
		return nil
	}
	return c.conn.Close()
}

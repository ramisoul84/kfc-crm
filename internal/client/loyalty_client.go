package client

import (
	"context"
	"fmt"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	loyaltyv1 "github.com/ramisoul84/kfc-crm/gen/loyalty/v1"
	"github.com/ramisoul84/kfc-crm/internal/config"
)

// LoyaltyClient is the back-office side of kfc-loyalty: finding
// customers, reading their points history and adjusting balances.
type LoyaltyClient interface {
	ListMembers(ctx context.Context, search string, limit, offset int) (*loyaltyv1.ListMembersResponse, error)
	GetMember(ctx context.Context, id string) (*loyaltyv1.Member, error)
	ListTransactions(ctx context.Context, memberID string, limit, offset int) (*loyaltyv1.ListTransactionsResponse, error)
	AdjustPoints(ctx context.Context, req *loyaltyv1.AdjustPointsRequest) (*loyaltyv1.Member, error)
	GetProgram(ctx context.Context) (*loyaltyv1.Program, error)
	Close() error
}

type loyaltyClient struct {
	client  loyaltyv1.LoyaltyServiceClient
	conn    *grpc.ClientConn
	timeout time.Duration
}

func NewLoyaltyClient(cfg *config.LoyaltyConfig) (LoyaltyClient, error) {
	conn, err := grpc.NewClient(cfg.Address, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, fmt.Errorf("create loyalty grpc client: %w", err)
	}
	return &loyaltyClient{client: loyaltyv1.NewLoyaltyServiceClient(conn), conn: conn, timeout: cfg.Timeout}, nil
}

func (c *loyaltyClient) ListMembers(ctx context.Context, search string, limit, offset int) (*loyaltyv1.ListMembersResponse, error) {
	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()
	return c.client.ListMembers(ctx, &loyaltyv1.ListMembersRequest{Search: search, Limit: int32(limit), Offset: int32(offset)})
}

func (c *loyaltyClient) GetMember(ctx context.Context, id string) (*loyaltyv1.Member, error) {
	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()
	return c.client.GetMember(ctx, &loyaltyv1.GetMemberRequest{Id: id})
}

func (c *loyaltyClient) ListTransactions(ctx context.Context, memberID string, limit, offset int) (*loyaltyv1.ListTransactionsResponse, error) {
	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()
	return c.client.ListTransactions(ctx, &loyaltyv1.ListTransactionsRequest{MemberId: memberID, Limit: int32(limit), Offset: int32(offset)})
}

func (c *loyaltyClient) AdjustPoints(ctx context.Context, req *loyaltyv1.AdjustPointsRequest) (*loyaltyv1.Member, error) {
	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()
	return c.client.AdjustPoints(ctx, req)
}

func (c *loyaltyClient) GetProgram(ctx context.Context) (*loyaltyv1.Program, error) {
	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()
	return c.client.GetProgram(ctx, &loyaltyv1.GetProgramRequest{})
}

func (c *loyaltyClient) Close() error {
	if c.conn == nil {
		return nil
	}
	return c.conn.Close()
}

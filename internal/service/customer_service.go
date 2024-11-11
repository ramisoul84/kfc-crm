package service

import (
	"context"
	"time"

	"github.com/google/uuid"

	loyaltyv1 "github.com/ramisoul84/kfc-crm/gen/loyalty/v1"
	"github.com/ramisoul84/kfc-crm/internal/client"
	"github.com/ramisoul84/kfc-crm/internal/domain"
	"github.com/ramisoul84/kfc-crm/pkg/ctxutil"
	"github.com/ramisoul84/kfc-crm/pkg/logger"
)

// CustomerService gives staff a view of loyalty members.
//
// Members are chain-wide, not tied to a restaurant: a customer may order
// anywhere, and any restaurant's staff may need to help them.
type CustomerService interface {
	List(ctx context.Context, actor *domain.User, search string, limit, offset int) ([]*domain.CustomerResponse, int, error)
	GetByID(ctx context.Context, actor *domain.User, id uuid.UUID) (*domain.CustomerResponse, error)
	Transactions(ctx context.Context, actor *domain.User, id uuid.UUID, limit, offset int) ([]*domain.PointsTransactionResponse, int, error)
	AdjustPoints(ctx context.Context, actor *domain.User, id uuid.UUID, req *domain.AdjustPointsRequest) (*domain.CustomerResponse, error)
	Program(ctx context.Context, actor *domain.User) (*domain.LoyaltyProgramResponse, error)
}

type customerService struct {
	loyalty     client.LoyaltyClient
	rbacService RBACService
	logger      *logger.Logger
}

func NewCustomerService(loyalty client.LoyaltyClient, rbac RBACService, log *logger.Logger) CustomerService {
	return &customerService{loyalty: loyalty, rbacService: rbac, logger: log}
}

func (s *customerService) List(
	ctx context.Context, actor *domain.User, search string, limit, offset int,
) ([]*domain.CustomerResponse, int, error) {
	if err := s.rbacService.CheckPermission(actor, domain.PermCustomerRead); err != nil {
		return nil, 0, err
	}
	resp, err := s.loyalty.ListMembers(ctx, search, limit, offset)
	if err != nil {
		return nil, 0, s.mapError(ctx, "list customers", err)
	}
	out := make([]*domain.CustomerResponse, 0, len(resp.GetMembers()))
	for _, m := range resp.GetMembers() {
		out = append(out, toCustomer(m))
	}
	return out, int(resp.GetTotal()), nil
}

func (s *customerService) GetByID(ctx context.Context, actor *domain.User, id uuid.UUID) (*domain.CustomerResponse, error) {
	if err := s.rbacService.CheckPermission(actor, domain.PermCustomerRead); err != nil {
		return nil, err
	}
	m, err := s.loyalty.GetMember(ctx, id.String())
	if err != nil {
		return nil, s.mapError(ctx, "get customer", err)
	}
	return toCustomer(m), nil
}

func (s *customerService) Transactions(
	ctx context.Context, actor *domain.User, id uuid.UUID, limit, offset int,
) ([]*domain.PointsTransactionResponse, int, error) {
	if err := s.rbacService.CheckPermission(actor, domain.PermCustomerRead); err != nil {
		return nil, 0, err
	}
	resp, err := s.loyalty.ListTransactions(ctx, id.String(), limit, offset)
	if err != nil {
		return nil, 0, s.mapError(ctx, "customer transactions", err)
	}
	out := make([]*domain.PointsTransactionResponse, 0, len(resp.GetTransactions()))
	for _, t := range resp.GetTransactions() {
		out = append(out, &domain.PointsTransactionResponse{
			ID:           t.GetId(),
			Type:         t.GetType(),
			Points:       int(t.GetPoints()),
			BalanceAfter: int(t.GetBalanceAfter()),
			OrderID:      t.GetOrderId(),
			OrderNumber:  int(t.GetOrderNumber()),
			RestaurantID: t.GetRestaurantId(),
			Description:  t.GetDescription(),
			CreatedAt:    time.UnixMilli(t.GetCreatedAt()),
		})
	}
	return out, int(resp.GetTotal()), nil
}

func (s *customerService) AdjustPoints(
	ctx context.Context, actor *domain.User, id uuid.UUID, req *domain.AdjustPointsRequest,
) (*domain.CustomerResponse, error) {
	log := s.logger.WithRequestID(ctxutil.GetRequestID(ctx))
	if err := s.rbacService.CheckPermission(actor, domain.PermCustomerAdjust); err != nil {
		log.Warn("adjust points denied", "actor_id", actor.ID, "customer_id", id)
		return nil, err
	}
	m, err := s.loyalty.AdjustPoints(ctx, &loyaltyv1.AdjustPointsRequest{
		MemberId: id.String(),
		Points:   int32(req.Points),
		Reason:   req.Reason,
		Actor:    "user:" + actor.ID.String(),
	})
	if err != nil {
		return nil, s.mapError(ctx, "adjust points", err)
	}
	log.Info("customer points adjusted", "customer_id", id, "points", req.Points, "actor_id", actor.ID)
	return toCustomer(m), nil
}

func (s *customerService) Program(ctx context.Context, actor *domain.User) (*domain.LoyaltyProgramResponse, error) {
	if err := s.rbacService.CheckPermission(actor, domain.PermCustomerRead); err != nil {
		return nil, err
	}
	p, err := s.loyalty.GetProgram(ctx)
	if err != nil {
		return nil, s.mapError(ctx, "loyalty program", err)
	}
	return &domain.LoyaltyProgramResponse{
		EarnPercent:      int(p.GetEarnPercent()),
		MaxRedeemPercent: int(p.GetMaxRedeemPercent()),
		WelcomeBonus:     int(p.GetWelcomeBonus()),
		PointValue:       rubles(p.GetPointValueKopecks()),
	}, nil
}

// mapError reuses the gRPC → domain mapping of the order service.
func (s *customerService) mapError(ctx context.Context, op string, err error) error {
	return (&orderService{logger: s.logger}).mapError(ctx, op, err)
}

func toCustomer(m *loyaltyv1.Member) *domain.CustomerResponse {
	return &domain.CustomerResponse{
		ID:              m.GetId(),
		Phone:           m.GetPhone(),
		FirstName:       m.GetFirstName(),
		Email:           m.GetEmail(),
		PointsBalance:   int(m.GetPointsBalance()),
		PointsAvailable: int(m.GetPointsAvailable()),
		LifetimePoints:  int(m.GetLifetimePoints()),
		CreatedAt:       time.UnixMilli(m.GetCreatedAt()),
	}
}

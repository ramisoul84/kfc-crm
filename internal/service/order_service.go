package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	orderv1 "github.com/ramisoul84/kfc-crm/gen/order/v1"
	"github.com/ramisoul84/kfc-crm/internal/client"
	"github.com/ramisoul84/kfc-crm/internal/domain"
	"github.com/ramisoul84/kfc-crm/internal/repository"
	"github.com/ramisoul84/kfc-crm/pkg/ctxutil"
	"github.com/ramisoul84/kfc-crm/pkg/logger"
)

// OrderService gives CRM users a scoped view of kfc-order.
type OrderService interface {
	List(ctx context.Context, actor *domain.User, filter domain.OrderFilter) ([]*domain.OrderResponse, int, error)
	GetByID(ctx context.Context, actor *domain.User, id uuid.UUID) (*domain.OrderResponse, error)
	Cancel(ctx context.Context, actor *domain.User, id uuid.UUID, reason string) (*domain.OrderResponse, error)
	Stats(ctx context.Context, actor *domain.User, filter domain.OrderStatsFilter) (*domain.OrderStatsResponse, error)
}

type orderService struct {
	orders         client.OrderClient
	restaurantRepo repository.RestaurantRepository
	rbacService    RBACService
	logger         *logger.Logger
}

func NewOrderService(
	orders client.OrderClient,
	restaurantRepo repository.RestaurantRepository,
	rbacService RBACService,
	log *logger.Logger,
) OrderService {
	return &orderService{
		orders:         orders,
		restaurantRepo: restaurantRepo,
		rbacService:    rbacService,
		logger:         log,
	}
}

// scope is the set of restaurants an actor may see, plus their names for
// display. all=true means "every restaurant" (super admin, no filter).
type scope struct {
	all         bool
	ids         []uuid.UUID
	restaurants map[uuid.UUID]*domain.Restaurant
}

// empty reports whether the actor can see no restaurants at all. Callers
// must short-circuit: an empty ID list means "all" to kfc-order.
func (sc *scope) empty() bool { return !sc.all && len(sc.ids) == 0 }

func (sc *scope) idStrings() []string {
	out := make([]string, len(sc.ids))
	for i, id := range sc.ids {
		out[i] = id.String()
	}
	return out
}

// ═══════════════════════════════════════════════════════════════════
// LIST / GET
// ═══════════════════════════════════════════════════════════════════

func (s *orderService) List(
	ctx context.Context,
	actor *domain.User,
	filter domain.OrderFilter,
) ([]*domain.OrderResponse, int, error) {
	if err := s.rbacService.CheckPermission(actor, domain.PermOrderRead); err != nil {
		return nil, 0, err
	}

	sc, err := s.resolveScope(ctx, actor, filter.RestaurantID)
	if err != nil {
		return nil, 0, err
	}
	if sc.empty() {
		return []*domain.OrderResponse{}, 0, nil
	}

	req := &orderv1.ListOrdersRequest{
		RestaurantIds: sc.idStrings(),
		Status:        filter.Status,
		Channel:       filter.Channel,
		Number:        int32(filter.Number),
		MemberId:      memberIDString(filter.MemberID),
		Limit:         int32(filter.Limit),
		Offset:        int32(filter.Offset),
	}
	if filter.From != nil {
		req.From = filter.From.UnixMilli()
	}
	if filter.To != nil {
		req.To = filter.To.UnixMilli()
	}

	resp, err := s.orders.ListOrders(ctx, req)
	if err != nil {
		return nil, 0, s.mapError(ctx, "list orders", err)
	}

	out := make([]*domain.OrderResponse, 0, len(resp.GetOrders()))
	for _, o := range resp.GetOrders() {
		out = append(out, toOrderResponse(o, sc.restaurants))
	}
	return out, int(resp.GetTotal()), nil
}

func (s *orderService) GetByID(
	ctx context.Context,
	actor *domain.User,
	id uuid.UUID,
) (*domain.OrderResponse, error) {
	if err := s.rbacService.CheckPermission(actor, domain.PermOrderRead); err != nil {
		return nil, err
	}

	o, restaurant, err := s.loadAuthorized(ctx, actor, id)
	if err != nil {
		return nil, err
	}
	return toOrderResponse(o, map[uuid.UUID]*domain.Restaurant{restaurant.ID: restaurant}), nil
}

// ═══════════════════════════════════════════════════════════════════
// CANCEL
// ═══════════════════════════════════════════════════════════════════

func (s *orderService) Cancel(
	ctx context.Context,
	actor *domain.User,
	id uuid.UUID,
	reason string,
) (*domain.OrderResponse, error) {
	log := s.logger.WithRequestID(ctxutil.GetRequestID(ctx))

	if err := s.rbacService.CheckPermission(actor, domain.PermOrderCancel); err != nil {
		log.Warn("cancel order denied", "actor_id", actor.ID, "order_id", id, "error", err)
		return nil, err
	}

	_, restaurant, err := s.loadAuthorized(ctx, actor, id)
	if err != nil {
		return nil, err
	}

	updated, err := s.orders.UpdateOrderStatus(ctx, &orderv1.UpdateOrderStatusRequest{
		Id:     id.String(),
		Status: "cancelled",
		Reason: reason,
		Actor:  "user:" + actor.ID.String(),
	})
	if err != nil {
		return nil, s.mapError(ctx, "cancel order", err)
	}

	log.Info("order cancelled",
		"order_id", id,
		"restaurant_id", restaurant.ID,
		"number", updated.GetNumber(),
		"actor_id", actor.ID,
		"reason", reason,
	)
	return toOrderResponse(updated, map[uuid.UUID]*domain.Restaurant{restaurant.ID: restaurant}), nil
}

// ═══════════════════════════════════════════════════════════════════
// STATS
// ═══════════════════════════════════════════════════════════════════

func (s *orderService) Stats(
	ctx context.Context,
	actor *domain.User,
	filter domain.OrderStatsFilter,
) (*domain.OrderStatsResponse, error) {
	if err := s.rbacService.CheckPermission(actor, domain.PermOrderRead); err != nil {
		return nil, err
	}

	sc, err := s.resolveScope(ctx, actor, filter.RestaurantID)
	if err != nil {
		return nil, err
	}

	out := &domain.OrderStatsResponse{
		From:      filter.From,
		To:        filter.To,
		ByChannel: []domain.ChannelStatResp{},
		ByHour:    []domain.HourStatResp{},
		TopItems:  []domain.ItemStatResp{},
		ByStatus:  []domain.StatusStatResp{},
	}
	if sc.empty() {
		return out, nil
	}

	st, err := s.orders.GetOrderStats(ctx, &orderv1.GetOrderStatsRequest{
		RestaurantIds: sc.idStrings(),
		From:          filter.From.UnixMilli(),
		To:            filter.To.UnixMilli(),
	})
	if err != nil {
		return nil, s.mapError(ctx, "order stats", err)
	}

	out.TotalOrders = int(st.GetTotalOrders())
	out.Revenue = rubles(st.GetRevenue())
	out.AverageTicket = rubles(st.GetAverageTicket())
	out.AvgPrepSeconds = int(st.GetAvgPrepSeconds())
	out.CancelledOrders = int(st.GetCancelledOrders())
	out.ActiveOrders = int(st.GetActiveOrders())
	for _, c := range st.GetByChannel() {
		out.ByChannel = append(out.ByChannel, domain.ChannelStatResp{
			Channel: c.GetChannel(), Orders: int(c.GetOrders()), Revenue: rubles(c.GetRevenue()),
		})
	}
	for _, h := range st.GetByHour() {
		out.ByHour = append(out.ByHour, domain.HourStatResp{
			Hour: int(h.GetHour()), Orders: int(h.GetOrders()), Revenue: rubles(h.GetRevenue()),
		})
	}
	for _, i := range st.GetTopItems() {
		out.TopItems = append(out.TopItems, domain.ItemStatResp{
			ItemID: i.GetItemId(), Name: i.GetName(), Quantity: int(i.GetQuantity()), Revenue: rubles(i.GetRevenue()),
		})
	}
	for _, b := range st.GetByStatus() {
		out.ByStatus = append(out.ByStatus, domain.StatusStatResp{Status: b.GetStatus(), Orders: int(b.GetOrders())})
	}
	return out, nil
}

// ═══════════════════════════════════════════════════════════════════
// SCOPE
// ═══════════════════════════════════════════════════════════════════

// resolveScope turns the actor's role (and an optional restaurant filter)
// into the restaurants they may see.
func (s *orderService) resolveScope(
	ctx context.Context,
	actor *domain.User,
	requested *uuid.UUID,
) (*scope, error) {
	// A specific restaurant: allowed only if it's inside the actor's scope.
	if requested != nil {
		if err := s.rbacService.CanAccessRestaurant(ctx, actor, *requested); err != nil {
			return nil, err
		}
		r, err := s.restaurantRepo.GetByID(ctx, *requested)
		if err != nil {
			return nil, err
		}
		return &scope{
			ids:         []uuid.UUID{r.ID},
			restaurants: map[uuid.UUID]*domain.Restaurant{r.ID: r},
		}, nil
	}

	switch actor.Role {
	case domain.RoleSuperAdmin:
		all, err := s.listRestaurants(ctx, nil)
		if err != nil {
			return nil, err
		}
		return &scope{all: true, restaurants: indexRestaurants(all)}, nil

	case domain.RoleRegionalManager:
		if actor.RegionID == nil {
			return nil, domain.NewAuthorizationError("no region assigned")
		}
		inRegion, err := s.listRestaurants(ctx, actor.RegionID)
		if err != nil {
			return nil, err
		}
		sc := &scope{restaurants: indexRestaurants(inRegion)}
		for _, r := range inRegion {
			sc.ids = append(sc.ids, r.ID)
		}
		return sc, nil

	default:
		if actor.RestaurantID == nil {
			return nil, domain.NewAuthorizationError("no restaurant assigned")
		}
		return s.resolveScope(ctx, actor, actor.RestaurantID)
	}
}

// loadAuthorized fetches an order and checks the actor may see its
// restaurant. Out-of-scope orders are reported as not found so their
// existence isn't leaked.
func (s *orderService) loadAuthorized(
	ctx context.Context,
	actor *domain.User,
	id uuid.UUID,
) (*orderv1.Order, *domain.Restaurant, error) {
	o, err := s.orders.GetOrder(ctx, id.String())
	if err != nil {
		return nil, nil, s.mapError(ctx, "get order", err)
	}
	restaurantID, err := uuid.Parse(o.GetRestaurantId())
	if err != nil {
		return nil, nil, domain.NewInternalError(fmt.Errorf("order has bad restaurant id: %w", err))
	}
	if err := s.rbacService.CanAccessRestaurant(ctx, actor, restaurantID); err != nil {
		return nil, nil, domain.NewNotFoundError("order not found")
	}
	restaurant, err := s.restaurantRepo.GetByID(ctx, restaurantID)
	if err != nil {
		return nil, nil, err
	}
	return o, restaurant, nil
}

// listRestaurants pages through restaurants (the repo caps pages at 100).
func (s *orderService) listRestaurants(ctx context.Context, regionID *uuid.UUID) ([]*domain.Restaurant, error) {
	const page = 100
	var all []*domain.Restaurant
	for offset := 0; ; offset += page {
		rs, total, err := s.restaurantRepo.List(ctx, domain.RestaurantFilter{
			RegionID: regionID,
			Limit:    page,
			Offset:   offset,
		})
		if err != nil {
			return nil, err
		}
		all = append(all, rs...)
		if len(rs) == 0 || len(all) >= total {
			return all, nil
		}
	}
}

func indexRestaurants(rs []*domain.Restaurant) map[uuid.UUID]*domain.Restaurant {
	m := make(map[uuid.UUID]*domain.Restaurant, len(rs))
	for _, r := range rs {
		m[r.ID] = r
	}
	return m
}

// mapError converts a kfc-order gRPC error into a domain error.
func (s *orderService) mapError(ctx context.Context, op string, err error) error {
	st, ok := status.FromError(err)
	if !ok {
		return domain.NewInternalError(fmt.Errorf("%s: %w", op, err))
	}
	switch st.Code() {
	case codes.InvalidArgument:
		return domain.NewValidationError(st.Message())
	case codes.NotFound:
		return domain.NewNotFoundError(st.Message())
	case codes.FailedPrecondition, codes.AlreadyExists:
		return domain.NewConflictError(st.Message())
	case codes.Unavailable, codes.DeadlineExceeded:
		s.logger.WithRequestID(ctxutil.GetRequestID(ctx)).Error("order service unavailable", "op", op, "error", err)
		return domain.NewInternalError(errors.New("order service unavailable"))
	default:
		return domain.NewInternalError(fmt.Errorf("%s: %w", op, err))
	}
}

// ═══════════════════════════════════════════════════════════════════
// CONVERSION
// ═══════════════════════════════════════════════════════════════════

func toOrderResponse(o *orderv1.Order, restaurants map[uuid.UUID]*domain.Restaurant) *domain.OrderResponse {
	r := &domain.OrderResponse{
		ID:             o.GetId(),
		RestaurantID:   o.GetRestaurantId(),
		Number:         int(o.GetNumber()),
		BusinessDate:   o.GetBusinessDate(),
		Channel:        o.GetChannel(),
		OrderType:      o.GetOrderType(),
		Status:         o.GetStatus(),
		PaymentMethod:  o.GetPaymentMethod(),
		PaymentStatus:  o.GetPaymentStatus(),
		CustomerName:   o.GetCustomerName(),
		CustomerPhone:  o.GetCustomerPhone(),
		Note:           o.GetNote(),
		DeviceID:       o.GetDeviceId(),
		Subtotal:       rubles(o.GetSubtotal()),
		Discount:       rubles(o.GetDiscount()),
		PointsRedeemed: int(o.GetPointsRedeemed()),
		MemberID:       o.GetMemberId(),
		Total:          rubles(o.GetTotal()),
		Currency:       o.GetCurrency(),
		CancelReason:   o.GetCancelReason(),
		Lines:          make([]domain.OrderLineResponse, 0, len(o.GetLines())),
		CreatedAt:      time.UnixMilli(o.GetCreatedAt()),
		UpdatedAt:      time.UnixMilli(o.GetUpdatedAt()),
		PreparingAt:    optionalMillis(o.GetPreparingAt()),
		ReadyAt:        optionalMillis(o.GetReadyAt()),
		CompletedAt:    optionalMillis(o.GetCompletedAt()),
		CancelledAt:    optionalMillis(o.GetCancelledAt()),
	}
	if id, err := uuid.Parse(o.GetRestaurantId()); err == nil {
		if rest, ok := restaurants[id]; ok {
			r.RestaurantName = rest.Name
			r.RestaurantCode = rest.Code
		}
	}
	for _, l := range o.GetLines() {
		r.Lines = append(r.Lines, domain.OrderLineResponse{
			ItemID:        l.GetItemId(),
			ProductCode:   l.GetProductCode(),
			Name:          l.GetName(),
			VariationName: l.GetVariationName(),
			Quantity:      int(l.GetQuantity()),
			UnitPrice:     rubles(l.GetUnitPrice()),
			LineTotal:     rubles(l.GetLineTotal()),
		})
	}
	return r
}

func rubles(kopecks int64) float64 { return float64(kopecks) / 100 }

func memberIDString(id *uuid.UUID) string {
	if id == nil {
		return ""
	}
	return id.String()
}

func optionalMillis(ms int64) *time.Time {
	if ms == 0 {
		return nil
	}
	t := time.UnixMilli(ms)
	return &t
}

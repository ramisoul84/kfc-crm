package handler

import (
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"

	"github.com/ramisoul84/kfc-crm/internal/domain"
	"github.com/ramisoul84/kfc-crm/internal/service"
	"github.com/ramisoul84/kfc-crm/internal/transport/http/response"
	"github.com/ramisoul84/kfc-crm/pkg/ctxutil"
	"github.com/ramisoul84/kfc-crm/pkg/validator"
)

// OrderHandler handles order history, cancellation and dashboard stats.
type OrderHandler struct {
	orderService service.OrderService
	validator    *validator.Validator
	// loc interprets date-only query params (YYYY-MM-DD) as local days.
	loc *time.Location
}

func NewOrderHandler(orderService service.OrderService, v *validator.Validator, loc *time.Location) *OrderHandler {
	return &OrderHandler{orderService: orderService, validator: v, loc: loc}
}

// List handles GET /api/v1/orders.
//
// Query: restaurant_id, status, channel, number, from, to, limit, offset.
// from/to accept RFC 3339 timestamps or YYYY-MM-DD dates; a date-only
// `to` is inclusive of that whole day.
func (h *OrderHandler) List(c *fiber.Ctx) error {
	actor := ctxutil.GetUserFromFiber(c)
	if actor == nil {
		return response.Error(c, domain.NewAuthenticationError("not authenticated"))
	}

	filter := domain.OrderFilter{
		Status:  c.Query("status"),
		Channel: c.Query("channel"),
		Number:  c.QueryInt("number", 0),
		Limit:   c.QueryInt("limit", 25),
		Offset:  c.QueryInt("offset", 0),
	}

	var err error
	if filter.RestaurantID, err = optionalUUIDQuery(c, "restaurant_id"); err != nil {
		return response.Error(c, err)
	}
	if filter.From, err = h.timeQuery(c, "from", false); err != nil {
		return response.Error(c, err)
	}
	if filter.To, err = h.timeQuery(c, "to", true); err != nil {
		return response.Error(c, err)
	}

	orders, total, err := h.orderService.List(c.UserContext(), actor, filter)
	if err != nil {
		return response.Error(c, err)
	}

	return response.Success(c, fiber.Map{
		"items":  orders,
		"total":  total,
		"limit":  filter.Limit,
		"offset": filter.Offset,
	})
}

// GetByID handles GET /api/v1/orders/:id.
func (h *OrderHandler) GetByID(c *fiber.Ctx) error {
	actor := ctxutil.GetUserFromFiber(c)
	if actor == nil {
		return response.Error(c, domain.NewAuthenticationError("not authenticated"))
	}

	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return response.Error(c, domain.NewValidationError("invalid order ID"))
	}

	order, err := h.orderService.GetByID(c.UserContext(), actor, id)
	if err != nil {
		return response.Error(c, err)
	}
	return response.Success(c, order)
}

// Cancel handles POST /api/v1/orders/:id/cancel.
func (h *OrderHandler) Cancel(c *fiber.Ctx) error {
	actor := ctxutil.GetUserFromFiber(c)
	if actor == nil {
		return response.Error(c, domain.NewAuthenticationError("not authenticated"))
	}

	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return response.Error(c, domain.NewValidationError("invalid order ID"))
	}

	var req domain.CancelOrderRequest
	if err := c.BodyParser(&req); err != nil {
		return response.Error(c, domain.NewValidationError("invalid request body"))
	}
	if err := h.validator.Validate(&req); err != nil {
		return response.Error(c, err)
	}

	order, err := h.orderService.Cancel(c.UserContext(), actor, id, req.Reason)
	if err != nil {
		return response.Error(c, err)
	}
	return response.Success(c, order)
}

// Stats handles GET /api/v1/orders/stats.
//
// Query: restaurant_id, from, to. Defaults to today (local time).
func (h *OrderHandler) Stats(c *fiber.Ctx) error {
	actor := ctxutil.GetUserFromFiber(c)
	if actor == nil {
		return response.Error(c, domain.NewAuthenticationError("not authenticated"))
	}

	now := time.Now().In(h.loc)
	startOfDay := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, h.loc)
	filter := domain.OrderStatsFilter{From: startOfDay, To: startOfDay.AddDate(0, 0, 1)}

	restaurantID, err := optionalUUIDQuery(c, "restaurant_id")
	if err != nil {
		return response.Error(c, err)
	}
	filter.RestaurantID = restaurantID

	if from, err := h.timeQuery(c, "from", false); err != nil {
		return response.Error(c, err)
	} else if from != nil {
		filter.From = *from
	}
	if to, err := h.timeQuery(c, "to", true); err != nil {
		return response.Error(c, err)
	} else if to != nil {
		filter.To = *to
	}

	stats, err := h.orderService.Stats(c.UserContext(), actor, filter)
	if err != nil {
		return response.Error(c, err)
	}
	return response.Success(c, stats)
}

// timeQuery parses an RFC 3339 timestamp or a YYYY-MM-DD date. For an
// end-of-range date (endOfDay=true) it returns the start of the next day,
// so "to=2026-10-10" includes all of the 10th.
func (h *OrderHandler) timeQuery(c *fiber.Ctx, key string, endOfDay bool) (*time.Time, error) {
	raw := c.Query(key)
	if raw == "" {
		return nil, nil
	}
	if t, err := time.Parse(time.RFC3339, raw); err == nil {
		return &t, nil
	}
	d, err := time.ParseInLocation(time.DateOnly, raw, h.loc)
	if err != nil {
		return nil, domain.NewValidationError("invalid " + key + ": use YYYY-MM-DD or RFC 3339")
	}
	if endOfDay {
		d = d.AddDate(0, 0, 1)
	}
	return &d, nil
}

func optionalUUIDQuery(c *fiber.Ctx, key string) (*uuid.UUID, error) {
	raw := c.Query(key)
	if raw == "" {
		return nil, nil
	}
	id, err := uuid.Parse(raw)
	if err != nil {
		return nil, domain.NewValidationError("invalid " + key)
	}
	return &id, nil
}

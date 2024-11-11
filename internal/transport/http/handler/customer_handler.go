package handler

import (
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"

	"github.com/ramisoul84/kfc-crm/internal/domain"
	"github.com/ramisoul84/kfc-crm/internal/service"
	"github.com/ramisoul84/kfc-crm/internal/transport/http/response"
	"github.com/ramisoul84/kfc-crm/pkg/ctxutil"
	"github.com/ramisoul84/kfc-crm/pkg/validator"
)

// CustomerHandler handles loyalty customer endpoints.
type CustomerHandler struct {
	customers service.CustomerService
	validator *validator.Validator
}

func NewCustomerHandler(customers service.CustomerService, v *validator.Validator) *CustomerHandler {
	return &CustomerHandler{customers: customers, validator: v}
}

// List handles GET /api/v1/customers?search=&limit=&offset=.
func (h *CustomerHandler) List(c *fiber.Ctx) error {
	actor := ctxutil.GetUserFromFiber(c)
	if actor == nil {
		return response.Error(c, domain.NewAuthenticationError("not authenticated"))
	}
	limit, offset := c.QueryInt("limit", 25), c.QueryInt("offset", 0)
	items, total, err := h.customers.List(c.UserContext(), actor, c.Query("search"), limit, offset)
	if err != nil {
		return response.Error(c, err)
	}
	return response.Success(c, fiber.Map{"items": items, "total": total, "limit": limit, "offset": offset})
}

// GetByID handles GET /api/v1/customers/:id.
func (h *CustomerHandler) GetByID(c *fiber.Ctx) error {
	actor := ctxutil.GetUserFromFiber(c)
	if actor == nil {
		return response.Error(c, domain.NewAuthenticationError("not authenticated"))
	}
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return response.Error(c, domain.NewValidationError("invalid customer ID"))
	}
	cust, err := h.customers.GetByID(c.UserContext(), actor, id)
	if err != nil {
		return response.Error(c, err)
	}
	return response.Success(c, cust)
}

// Transactions handles GET /api/v1/customers/:id/points.
func (h *CustomerHandler) Transactions(c *fiber.Ctx) error {
	actor := ctxutil.GetUserFromFiber(c)
	if actor == nil {
		return response.Error(c, domain.NewAuthenticationError("not authenticated"))
	}
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return response.Error(c, domain.NewValidationError("invalid customer ID"))
	}
	limit, offset := c.QueryInt("limit", 25), c.QueryInt("offset", 0)
	items, total, err := h.customers.Transactions(c.UserContext(), actor, id, limit, offset)
	if err != nil {
		return response.Error(c, err)
	}
	return response.Success(c, fiber.Map{"items": items, "total": total, "limit": limit, "offset": offset})
}

// Adjust handles POST /api/v1/customers/:id/points {points, reason}.
func (h *CustomerHandler) Adjust(c *fiber.Ctx) error {
	actor := ctxutil.GetUserFromFiber(c)
	if actor == nil {
		return response.Error(c, domain.NewAuthenticationError("not authenticated"))
	}
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return response.Error(c, domain.NewValidationError("invalid customer ID"))
	}
	var req domain.AdjustPointsRequest
	if err := c.BodyParser(&req); err != nil {
		return response.Error(c, domain.NewValidationError("invalid request body"))
	}
	if err := h.validator.Validate(&req); err != nil {
		return response.Error(c, err)
	}
	cust, err := h.customers.AdjustPoints(c.UserContext(), actor, id, &req)
	if err != nil {
		return response.Error(c, err)
	}
	return response.Success(c, cust)
}

// Program handles GET /api/v1/loyalty/program.
func (h *CustomerHandler) Program(c *fiber.Ctx) error {
	actor := ctxutil.GetUserFromFiber(c)
	if actor == nil {
		return response.Error(c, domain.NewAuthenticationError("not authenticated"))
	}
	p, err := h.customers.Program(c.UserContext(), actor)
	if err != nil {
		return response.Error(c, err)
	}
	return response.Success(c, p)
}

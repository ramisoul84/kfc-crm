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

// MenuPromotionHandler handles promotion endpoints.
type MenuPromotionHandler struct {
	promotionService service.MenuPromotionService
	validator        *validator.Validator
}

// NewMenuPromotionHandler creates a MenuPromotionHandler.
func NewMenuPromotionHandler(
	promotionService service.MenuPromotionService,
	v *validator.Validator,
) *MenuPromotionHandler {
	return &MenuPromotionHandler{
		promotionService: promotionService,
		validator:        v,
	}
}

// Create handles POST /api/v1/menu/promotions.
func (h *MenuPromotionHandler) Create(c *fiber.Ctx) error {
	actor := ctxutil.GetUserFromFiber(c)
	if actor == nil {
		return response.Error(c, domain.NewAuthenticationError("not authenticated"))
	}

	var req domain.CreatePromotionRequest
	if err := c.BodyParser(&req); err != nil {
		return response.Error(c, domain.NewValidationError("invalid request body"))
	}
	if err := h.validator.Validate(&req); err != nil {
		return response.Error(c, err)
	}

	promotion, err := h.promotionService.Create(c.UserContext(), actor, &req)
	if err != nil {
		return response.Error(c, err)
	}

	return response.Created(c, promotion)
}

// GetByID handles GET /api/v1/menu/promotions/:id.
func (h *MenuPromotionHandler) GetByID(c *fiber.Ctx) error {
	actor := ctxutil.GetUserFromFiber(c)
	if actor == nil {
		return response.Error(c, domain.NewAuthenticationError("not authenticated"))
	}

	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return response.Error(c, domain.NewValidationError("invalid promotion ID"))
	}

	promotion, err := h.promotionService.GetByID(c.UserContext(), actor, id)
	if err != nil {
		return response.Error(c, err)
	}

	return response.Success(c, promotion)
}

// List handles GET /api/v1/menu/promotions.
func (h *MenuPromotionHandler) List(c *fiber.Ctx) error {
	actor := ctxutil.GetUserFromFiber(c)
	if actor == nil {
		return response.Error(c, domain.NewAuthenticationError("not authenticated"))
	}

	filter := domain.PromotionFilter{
		Scope:     domain.PromotionScope(c.Query("scope")),
		ActiveNow: c.Query("active_now") == "true",
		Limit:     c.QueryInt("limit", 10),
		Offset:    c.QueryInt("offset", 0),
	}

	if v := c.Query("region_id"); v != "" {
		id, err := uuid.Parse(v)
		if err != nil {
			return response.Error(c, domain.NewValidationError("invalid region_id"))
		}
		filter.RegionID = &id
	}

	if v := c.Query("restaurant_id"); v != "" {
		id, err := uuid.Parse(v)
		if err != nil {
			return response.Error(c, domain.NewValidationError("invalid restaurant_id"))
		}
		filter.RestaurantID = &id
	}

	promotions, total, err := h.promotionService.List(c.UserContext(), actor, filter)
	if err != nil {
		return response.Error(c, err)
	}

	return response.Success(c, fiber.Map{
		"items":  promotions,
		"total":  total,
		"limit":  filter.Limit,
		"offset": filter.Offset,
	})
}

// Update handles PUT /api/v1/menu/promotions/:id.
func (h *MenuPromotionHandler) Update(c *fiber.Ctx) error {
	actor := ctxutil.GetUserFromFiber(c)
	if actor == nil {
		return response.Error(c, domain.NewAuthenticationError("not authenticated"))
	}

	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return response.Error(c, domain.NewValidationError("invalid promotion ID"))
	}

	var req domain.UpdatePromotionRequest
	if err := c.BodyParser(&req); err != nil {
		return response.Error(c, domain.NewValidationError("invalid request body"))
	}
	if err := h.validator.Validate(&req); err != nil {
		return response.Error(c, err)
	}

	promotion, err := h.promotionService.Update(c.UserContext(), actor, id, &req)
	if err != nil {
		return response.Error(c, err)
	}

	return response.Success(c, promotion)
}

// Delete handles DELETE /api/v1/menu/promotions/:id.
func (h *MenuPromotionHandler) Delete(c *fiber.Ctx) error {
	actor := ctxutil.GetUserFromFiber(c)
	if actor == nil {
		return response.Error(c, domain.NewAuthenticationError("not authenticated"))
	}

	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return response.Error(c, domain.NewValidationError("invalid promotion ID"))
	}

	if err := h.promotionService.Delete(c.UserContext(), actor, id); err != nil {
		return response.Error(c, err)
	}

	return response.Success(c, fiber.Map{
		"message": "promotion deleted successfully",
	})
}

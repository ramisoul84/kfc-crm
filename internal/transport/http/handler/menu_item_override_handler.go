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

// MenuItemOverrideHandler handles menu item override endpoints.
type MenuItemOverrideHandler struct {
	overrideService service.MenuItemOverrideService
	validator       *validator.Validator
}

// NewMenuItemOverrideHandler creates a MenuItemOverrideHandler.
func NewMenuItemOverrideHandler(
	overrideService service.MenuItemOverrideService,
	v *validator.Validator,
) *MenuItemOverrideHandler {
	return &MenuItemOverrideHandler{
		overrideService: overrideService,
		validator:       v,
	}
}

// Set handles POST /api/v1/menu/overrides.
func (h *MenuItemOverrideHandler) Set(c *fiber.Ctx) error {
	actor := ctxutil.GetUserFromFiber(c)
	if actor == nil {
		return response.Error(c, domain.NewAuthenticationError("not authenticated"))
	}

	var req domain.CreateMenuItemOverrideRequest
	if err := c.BodyParser(&req); err != nil {
		return response.Error(c, domain.NewValidationError("invalid request body"))
	}
	if err := h.validator.Validate(&req); err != nil {
		return response.Error(c, err)
	}

	override, err := h.overrideService.Set(c.UserContext(), actor, &req)
	if err != nil {
		return response.Error(c, err)
	}

	return response.Success(c, override)
}

// GetByID handles GET /api/v1/menu/overrides/:id.
func (h *MenuItemOverrideHandler) GetByID(c *fiber.Ctx) error {
	actor := ctxutil.GetUserFromFiber(c)
	if actor == nil {
		return response.Error(c, domain.NewAuthenticationError("not authenticated"))
	}

	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return response.Error(c, domain.NewValidationError("invalid override ID"))
	}

	override, err := h.overrideService.GetByID(c.UserContext(), actor, id)
	if err != nil {
		return response.Error(c, err)
	}

	return response.Success(c, override)
}

// ListByRestaurant handles GET /api/v1/restaurants/:restaurant_id/menu-overrides.
func (h *MenuItemOverrideHandler) ListByRestaurant(c *fiber.Ctx) error {
	actor := ctxutil.GetUserFromFiber(c)
	if actor == nil {
		return response.Error(c, domain.NewAuthenticationError("not authenticated"))
	}

	restaurantID, err := uuid.Parse(c.Params("restaurant_id"))
	if err != nil {
		return response.Error(c, domain.NewValidationError("invalid restaurant ID"))
	}

	overrides, err := h.overrideService.ListByRestaurant(c.UserContext(), actor, restaurantID)
	if err != nil {
		return response.Error(c, err)
	}

	return response.Success(c, overrides)
}

// Update handles PUT /api/v1/menu/overrides/:id.
func (h *MenuItemOverrideHandler) Update(c *fiber.Ctx) error {
	actor := ctxutil.GetUserFromFiber(c)
	if actor == nil {
		return response.Error(c, domain.NewAuthenticationError("not authenticated"))
	}

	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return response.Error(c, domain.NewValidationError("invalid override ID"))
	}

	var req domain.UpdateMenuItemOverrideRequest
	if err := c.BodyParser(&req); err != nil {
		return response.Error(c, domain.NewValidationError("invalid request body"))
	}
	if err := h.validator.Validate(&req); err != nil {
		return response.Error(c, err)
	}

	override, err := h.overrideService.Update(c.UserContext(), actor, id, &req)
	if err != nil {
		return response.Error(c, err)
	}

	return response.Success(c, override)
}

// Delete handles DELETE /api/v1/menu/overrides/:id.
func (h *MenuItemOverrideHandler) Delete(c *fiber.Ctx) error {
	actor := ctxutil.GetUserFromFiber(c)
	if actor == nil {
		return response.Error(c, domain.NewAuthenticationError("not authenticated"))
	}

	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return response.Error(c, domain.NewValidationError("invalid override ID"))
	}

	if err := h.overrideService.Delete(c.UserContext(), actor, id); err != nil {
		return response.Error(c, err)
	}

	return response.Success(c, fiber.Map{
		"message": "menu item override removed successfully",
	})
}

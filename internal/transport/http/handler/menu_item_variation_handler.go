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

// MenuItemVariationHandler handles menu item variation endpoints.
type MenuItemVariationHandler struct {
	variationService service.MenuItemVariationService
	validator        *validator.Validator
}

// NewMenuItemVariationHandler creates a MenuItemVariationHandler.
func NewMenuItemVariationHandler(
	variationService service.MenuItemVariationService,
	v *validator.Validator,
) *MenuItemVariationHandler {
	return &MenuItemVariationHandler{
		variationService: variationService,
		validator:        v,
	}
}

// Create handles POST /api/v1/menu/variations.
func (h *MenuItemVariationHandler) Create(c *fiber.Ctx) error {
	actor := ctxutil.GetUserFromFiber(c)
	if actor == nil {
		return response.Error(c, domain.NewAuthenticationError("not authenticated"))
	}

	var req domain.CreateMenuItemVariationRequest
	if err := c.BodyParser(&req); err != nil {
		return response.Error(c, domain.NewValidationError("invalid request body"))
	}
	if err := h.validator.Validate(&req); err != nil {
		return response.Error(c, err)
	}

	variation, err := h.variationService.Create(c.UserContext(), actor, &req)
	if err != nil {
		return response.Error(c, err)
	}

	return response.Created(c, variation)
}

// GetByID handles GET /api/v1/menu/variations/:id.
func (h *MenuItemVariationHandler) GetByID(c *fiber.Ctx) error {
	actor := ctxutil.GetUserFromFiber(c)
	if actor == nil {
		return response.Error(c, domain.NewAuthenticationError("not authenticated"))
	}

	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return response.Error(c, domain.NewValidationError("invalid variation ID"))
	}

	variation, err := h.variationService.GetByID(c.UserContext(), actor, id)
	if err != nil {
		return response.Error(c, err)
	}

	return response.Success(c, variation)
}

// ListByItem handles GET /api/v1/menu/items/:item_id/variations.
func (h *MenuItemVariationHandler) ListByItem(c *fiber.Ctx) error {
	actor := ctxutil.GetUserFromFiber(c)
	if actor == nil {
		return response.Error(c, domain.NewAuthenticationError("not authenticated"))
	}

	itemID, err := uuid.Parse(c.Params("item_id"))
	if err != nil {
		return response.Error(c, domain.NewValidationError("invalid menu item ID"))
	}

	variations, err := h.variationService.ListByItem(c.UserContext(), actor, itemID)
	if err != nil {
		return response.Error(c, err)
	}

	return response.Success(c, variations)
}

// Update handles PUT /api/v1/menu/variations/:id.
func (h *MenuItemVariationHandler) Update(c *fiber.Ctx) error {
	actor := ctxutil.GetUserFromFiber(c)
	if actor == nil {
		return response.Error(c, domain.NewAuthenticationError("not authenticated"))
	}

	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return response.Error(c, domain.NewValidationError("invalid variation ID"))
	}

	var req domain.UpdateMenuItemVariationRequest
	if err := c.BodyParser(&req); err != nil {
		return response.Error(c, domain.NewValidationError("invalid request body"))
	}
	if err := h.validator.Validate(&req); err != nil {
		return response.Error(c, err)
	}

	variation, err := h.variationService.Update(c.UserContext(), actor, id, &req)
	if err != nil {
		return response.Error(c, err)
	}

	return response.Success(c, variation)
}

// Delete handles DELETE /api/v1/menu/variations/:id.
func (h *MenuItemVariationHandler) Delete(c *fiber.Ctx) error {
	actor := ctxutil.GetUserFromFiber(c)
	if actor == nil {
		return response.Error(c, domain.NewAuthenticationError("not authenticated"))
	}

	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return response.Error(c, domain.NewValidationError("invalid variation ID"))
	}

	if err := h.variationService.Delete(c.UserContext(), actor, id); err != nil {
		return response.Error(c, err)
	}

	return response.Success(c, fiber.Map{
		"message": "menu item variation deleted successfully",
	})
}

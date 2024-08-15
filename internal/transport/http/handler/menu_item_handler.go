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

// MenuItemHandler handles menu item endpoints.
type MenuItemHandler struct {
	itemService service.MenuItemService
	validator   *validator.Validator
}

// NewMenuItemHandler creates a MenuItemHandler.
func NewMenuItemHandler(itemService service.MenuItemService, v *validator.Validator) *MenuItemHandler {
	return &MenuItemHandler{
		itemService: itemService,
		validator:   v,
	}
}

// Create handles POST /api/v1/menu/items.
func (h *MenuItemHandler) Create(c *fiber.Ctx) error {
	actor := ctxutil.GetUserFromFiber(c)
	if actor == nil {
		return response.Error(c, domain.NewAuthenticationError("not authenticated"))
	}

	var req domain.CreateMenuItemRequest
	if err := c.BodyParser(&req); err != nil {
		return response.Error(c, domain.NewValidationError("invalid request body"))
	}
	if err := h.validator.Validate(&req); err != nil {
		return response.Error(c, err)
	}

	item, err := h.itemService.Create(c.UserContext(), actor, &req)
	if err != nil {
		return response.Error(c, err)
	}

	return response.Created(c, item)
}

// GetByID handles GET /api/v1/menu/items/:id.
func (h *MenuItemHandler) GetByID(c *fiber.Ctx) error {
	actor := ctxutil.GetUserFromFiber(c)
	if actor == nil {
		return response.Error(c, domain.NewAuthenticationError("not authenticated"))
	}

	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return response.Error(c, domain.NewValidationError("invalid item ID"))
	}

	item, err := h.itemService.GetByID(c.UserContext(), actor, id)
	if err != nil {
		return response.Error(c, err)
	}

	return response.Success(c, item)
}

// List handles GET /api/v1/menu/items.
func (h *MenuItemHandler) List(c *fiber.Ctx) error {
	actor := ctxutil.GetUserFromFiber(c)
	if actor == nil {
		return response.Error(c, domain.NewAuthenticationError("not authenticated"))
	}

	filter := domain.MenuItemFilter{
		Search: c.Query("search"),
		Limit:  c.QueryInt("limit", 10),
		Offset: c.QueryInt("offset", 0),
	}

	if v := c.Query("category_id"); v != "" {
		id, err := uuid.Parse(v)
		if err != nil {
			return response.Error(c, domain.NewValidationError("invalid category_id"))
		}
		filter.CategoryID = &id
	}

	if v := c.Query("is_active"); v != "" {
		active := v == "true"
		filter.IsActive = &active
	}

	items, total, err := h.itemService.List(c.UserContext(), actor, filter)
	if err != nil {
		return response.Error(c, err)
	}

	return response.Success(c, fiber.Map{
		"items":  items,
		"total":  total,
		"limit":  filter.Limit,
		"offset": filter.Offset,
	})
}

// Update handles PUT /api/v1/menu/items/:id.
func (h *MenuItemHandler) Update(c *fiber.Ctx) error {
	actor := ctxutil.GetUserFromFiber(c)
	if actor == nil {
		return response.Error(c, domain.NewAuthenticationError("not authenticated"))
	}

	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return response.Error(c, domain.NewValidationError("invalid item ID"))
	}

	var req domain.UpdateMenuItemRequest
	if err := c.BodyParser(&req); err != nil {
		return response.Error(c, domain.NewValidationError("invalid request body"))
	}
	if err := h.validator.Validate(&req); err != nil {
		return response.Error(c, err)
	}

	item, err := h.itemService.Update(c.UserContext(), actor, id, &req)
	if err != nil {
		return response.Error(c, err)
	}

	return response.Success(c, item)
}

// Delete handles DELETE /api/v1/menu/items/:id.
func (h *MenuItemHandler) Delete(c *fiber.Ctx) error {
	actor := ctxutil.GetUserFromFiber(c)
	if actor == nil {
		return response.Error(c, domain.NewAuthenticationError("not authenticated"))
	}

	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return response.Error(c, domain.NewValidationError("invalid item ID"))
	}

	if err := h.itemService.Delete(c.UserContext(), actor, id); err != nil {
		return response.Error(c, err)
	}

	return response.Success(c, fiber.Map{
		"message": "menu item deleted successfully",
	})
}

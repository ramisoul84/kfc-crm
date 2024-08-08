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

// MenuCategoryHandler handles menu category endpoints.
type MenuCategoryHandler struct {
	categoryService service.MenuCategoryService
	validator       *validator.Validator
}

// NewMenuCategoryHandler creates a MenuCategoryHandler.
func NewMenuCategoryHandler(
	categoryService service.MenuCategoryService,
	v *validator.Validator,
) *MenuCategoryHandler {
	return &MenuCategoryHandler{
		categoryService: categoryService,
		validator:       v,
	}
}

// Create handles POST /api/v1/menu/categories.
func (h *MenuCategoryHandler) Create(c *fiber.Ctx) error {
	actor := ctxutil.GetUserFromFiber(c)
	if actor == nil {
		return response.Error(c, domain.NewAuthenticationError("not authenticated"))
	}

	var req domain.CreateMenuCategoryRequest
	if err := c.BodyParser(&req); err != nil {
		return response.Error(c, domain.NewValidationError("invalid request body"))
	}
	if err := h.validator.Validate(&req); err != nil {
		return response.Error(c, err)
	}

	category, err := h.categoryService.Create(c.UserContext(), actor, &req)
	if err != nil {
		return response.Error(c, err)
	}

	return response.Created(c, category)
}

// GetByID handles GET /api/v1/menu/categories/:id.
func (h *MenuCategoryHandler) GetByID(c *fiber.Ctx) error {
	actor := ctxutil.GetUserFromFiber(c)
	if actor == nil {
		return response.Error(c, domain.NewAuthenticationError("not authenticated"))
	}

	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return response.Error(c, domain.NewValidationError("invalid category ID"))
	}

	category, err := h.categoryService.GetByID(c.UserContext(), actor, id)
	if err != nil {
		return response.Error(c, err)
	}

	return response.Success(c, category)
}

// List handles GET /api/v1/menu/categories.
func (h *MenuCategoryHandler) List(c *fiber.Ctx) error {
	actor := ctxutil.GetUserFromFiber(c)
	if actor == nil {
		return response.Error(c, domain.NewAuthenticationError("not authenticated"))
	}

	filter := domain.MenuCategoryFilter{
		Search: c.Query("search"),
		Limit:  c.QueryInt("limit", 10),
		Offset: c.QueryInt("offset", 0),
	}

	if v := c.Query("is_active"); v != "" {
		active := v == "true"
		filter.IsActive = &active
	}

	categories, total, err := h.categoryService.List(c.UserContext(), actor, filter)
	if err != nil {
		return response.Error(c, err)
	}

	return response.Success(c, fiber.Map{
		"items":  categories,
		"total":  total,
		"limit":  filter.Limit,
		"offset": filter.Offset,
	})
}

// Update handles PUT /api/v1/menu/categories/:id.
func (h *MenuCategoryHandler) Update(c *fiber.Ctx) error {
	actor := ctxutil.GetUserFromFiber(c)
	if actor == nil {
		return response.Error(c, domain.NewAuthenticationError("not authenticated"))
	}

	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return response.Error(c, domain.NewValidationError("invalid category ID"))
	}

	var req domain.UpdateMenuCategoryRequest
	if err := c.BodyParser(&req); err != nil {
		return response.Error(c, domain.NewValidationError("invalid request body"))
	}
	if err := h.validator.Validate(&req); err != nil {
		return response.Error(c, err)
	}

	category, err := h.categoryService.Update(c.UserContext(), actor, id, &req)
	if err != nil {
		return response.Error(c, err)
	}

	return response.Success(c, category)
}

// Delete handles DELETE /api/v1/menu/categories/:id.
func (h *MenuCategoryHandler) Delete(c *fiber.Ctx) error {
	actor := ctxutil.GetUserFromFiber(c)
	if actor == nil {
		return response.Error(c, domain.NewAuthenticationError("not authenticated"))
	}

	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return response.Error(c, domain.NewValidationError("invalid category ID"))
	}

	if err := h.categoryService.Delete(c.UserContext(), actor, id); err != nil {
		return response.Error(c, err)
	}

	return response.Success(c, fiber.Map{
		"message": "menu category deleted successfully",
	})
}

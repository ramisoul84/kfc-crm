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

// RestaurantHandler handles restaurant endpoints.
type RestaurantHandler struct {
	restaurantService service.RestaurantService
	validator         *validator.Validator
}

// NewRestaurantHandler creates a RestaurantHandler.
func NewRestaurantHandler(
	restaurantService service.RestaurantService,
	v *validator.Validator,
) *RestaurantHandler {
	return &RestaurantHandler{
		restaurantService: restaurantService,
		validator:         v,
	}
}

// ═══════════════════════════════════════════════════════════════════
// CREATE
// ═══════════════════════════════════════════════════════════════════

// Create handles POST /api/v1/restaurants.
func (h *RestaurantHandler) Create(c *fiber.Ctx) error {
	actor := ctxutil.GetUserFromFiber(c)
	if actor == nil {
		return response.Error(c, domain.NewAuthenticationError("not authenticated"))
	}

	var req domain.CreateRestaurantRequest
	if err := c.BodyParser(&req); err != nil {
		return response.Error(c, domain.NewValidationError("invalid request body"))
	}
	if err := h.validator.Validate(&req); err != nil {
		return response.Error(c, err)
	}

	restaurant, err := h.restaurantService.Create(c.UserContext(), actor, &req)
	if err != nil {
		return response.Error(c, err)
	}

	return response.Created(c, restaurant)
}

// ═══════════════════════════════════════════════════════════════════
// READ
// ═══════════════════════════════════════════════════════════════════

// GetByID handles GET /api/v1/restaurants/:id.
func (h *RestaurantHandler) GetByID(c *fiber.Ctx) error {
	actor := ctxutil.GetUserFromFiber(c)
	if actor == nil {
		return response.Error(c, domain.NewAuthenticationError("not authenticated"))
	}

	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return response.Error(c, domain.NewValidationError("invalid restaurant ID"))
	}

	restaurant, err := h.restaurantService.GetByID(c.UserContext(), actor, id)
	if err != nil {
		return response.Error(c, err)
	}

	return response.Success(c, restaurant)
}

// List handles GET /api/v1/restaurants.
func (h *RestaurantHandler) List(c *fiber.Ctx) error {
	actor := ctxutil.GetUserFromFiber(c)
	if actor == nil {
		return response.Error(c, domain.NewAuthenticationError("not authenticated"))
	}

	filter := domain.RestaurantFilter{
		Search: c.Query("search"),
		Limit:  c.QueryInt("limit", 10),
		Offset: c.QueryInt("offset", 0),
	}

	if v := c.Query("region_id"); v != "" {
		id, err := uuid.Parse(v)
		if err != nil {
			return response.Error(c, domain.NewValidationError("invalid region_id"))
		}
		filter.RegionID = &id
	}

	if v := c.Query("is_active"); v != "" {
		active := v == "true"
		filter.IsActive = &active
	}

	restaurants, total, err := h.restaurantService.List(c.UserContext(), actor, filter)
	if err != nil {
		return response.Error(c, err)
	}

	return response.Success(c, fiber.Map{
		"items":  restaurants,
		"total":  total,
		"limit":  filter.Limit,
		"offset": filter.Offset,
	})
}

// ═══════════════════════════════════════════════════════════════════
// UPDATE
// ═══════════════════════════════════════════════════════════════════

// Update handles PUT /api/v1/restaurants/:id.
func (h *RestaurantHandler) Update(c *fiber.Ctx) error {
	actor := ctxutil.GetUserFromFiber(c)
	if actor == nil {
		return response.Error(c, domain.NewAuthenticationError("not authenticated"))
	}

	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return response.Error(c, domain.NewValidationError("invalid restaurant ID"))
	}

	var req domain.UpdateRestaurantRequest
	if err := c.BodyParser(&req); err != nil {
		return response.Error(c, domain.NewValidationError("invalid request body"))
	}
	if err := h.validator.Validate(&req); err != nil {
		return response.Error(c, err)
	}

	restaurant, err := h.restaurantService.Update(c.UserContext(), actor, id, &req)
	if err != nil {
		return response.Error(c, err)
	}

	return response.Success(c, restaurant)
}

// ═══════════════════════════════════════════════════════════════════
// DELETE
// ═══════════════════════════════════════════════════════════════════

// Delete handles DELETE /api/v1/restaurants/:id.
func (h *RestaurantHandler) Delete(c *fiber.Ctx) error {
	actor := ctxutil.GetUserFromFiber(c)
	if actor == nil {
		return response.Error(c, domain.NewAuthenticationError("not authenticated"))
	}

	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return response.Error(c, domain.NewValidationError("invalid restaurant ID"))
	}

	if err := h.restaurantService.Delete(c.UserContext(), actor, id); err != nil {
		return response.Error(c, err)
	}

	return response.Success(c, fiber.Map{
		"message": "restaurant deleted successfully",
	})
}

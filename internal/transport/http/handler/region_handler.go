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

// RegionHandler handles region endpoints.
type RegionHandler struct {
	regionService service.RegionService
	validator     *validator.Validator
}

// NewRegionHandler creates a RegionHandler.
func NewRegionHandler(regionService service.RegionService, v *validator.Validator) *RegionHandler {
	return &RegionHandler{
		regionService: regionService,
		validator:     v,
	}
}

// ═══════════════════════════════════════════════════════════════════
// CREATE
// ═══════════════════════════════════════════════════════════════════

// Create handles POST /api/v1/regions.
func (h *RegionHandler) Create(c *fiber.Ctx) error {
	actor := ctxutil.GetUserFromFiber(c)
	if actor == nil {
		return response.Error(c, domain.NewAuthenticationError("not authenticated"))
	}

	var req domain.CreateRegionRequest
	if err := c.BodyParser(&req); err != nil {
		return response.Error(c, domain.NewValidationError("invalid request body"))
	}
	if err := h.validator.Validate(&req); err != nil {
		return response.Error(c, err)
	}

	region, err := h.regionService.Create(c.UserContext(), actor, &req)
	if err != nil {
		return response.Error(c, err)
	}

	return response.Created(c, region)
}

// ═══════════════════════════════════════════════════════════════════
// READ
// ═══════════════════════════════════════════════════════════════════

// GetByID handles GET /api/v1/regions/:id.
func (h *RegionHandler) GetByID(c *fiber.Ctx) error {
	actor := ctxutil.GetUserFromFiber(c)
	if actor == nil {
		return response.Error(c, domain.NewAuthenticationError("not authenticated"))
	}

	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return response.Error(c, domain.NewValidationError("invalid region ID"))
	}

	region, err := h.regionService.GetByID(c.UserContext(), actor, id)
	if err != nil {
		return response.Error(c, err)
	}

	return response.Success(c, region)
}

// List handles GET /api/v1/regions.
func (h *RegionHandler) List(c *fiber.Ctx) error {
	actor := ctxutil.GetUserFromFiber(c)
	if actor == nil {
		return response.Error(c, domain.NewAuthenticationError("not authenticated"))
	}

	filter := domain.RegionFilter{
		Search: c.Query("search"),
		Limit:  c.QueryInt("limit", 10),
		Offset: c.QueryInt("offset", 0),
	}

	if v := c.Query("is_active"); v != "" {
		active := v == "true"
		filter.IsActive = &active
	}

	regions, total, err := h.regionService.List(c.UserContext(), actor, filter)
	if err != nil {
		return response.Error(c, err)
	}

	return response.Success(c, fiber.Map{
		"items":  regions,
		"total":  total,
		"limit":  filter.Limit,
		"offset": filter.Offset,
	})
}

// ═══════════════════════════════════════════════════════════════════
// UPDATE
// ═══════════════════════════════════════════════════════════════════

// Update handles PUT /api/v1/regions/:id.
func (h *RegionHandler) Update(c *fiber.Ctx) error {
	actor := ctxutil.GetUserFromFiber(c)
	if actor == nil {
		return response.Error(c, domain.NewAuthenticationError("not authenticated"))
	}

	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return response.Error(c, domain.NewValidationError("invalid region ID"))
	}

	var req domain.UpdateRegionRequest
	if err := c.BodyParser(&req); err != nil {
		return response.Error(c, domain.NewValidationError("invalid request body"))
	}
	if err := h.validator.Validate(&req); err != nil {
		return response.Error(c, err)
	}

	region, err := h.regionService.Update(c.UserContext(), actor, id, &req)
	if err != nil {
		return response.Error(c, err)
	}

	return response.Success(c, region)
}

// ═══════════════════════════════════════════════════════════════════
// DELETE
// ═══════════════════════════════════════════════════════════════════

// Delete handles DELETE /api/v1/regions/:id.
func (h *RegionHandler) Delete(c *fiber.Ctx) error {
	actor := ctxutil.GetUserFromFiber(c)
	if actor == nil {
		return response.Error(c, domain.NewAuthenticationError("not authenticated"))
	}

	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return response.Error(c, domain.NewValidationError("invalid region ID"))
	}

	if err := h.regionService.Delete(c.UserContext(), actor, id); err != nil {
		return response.Error(c, err)
	}

	return response.Success(c, fiber.Map{
		"message": "region deleted successfully",
	})
}

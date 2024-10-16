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

// UserHandler handles user endpoints.
type UserHandler struct {
	userService service.UserService
	validator   *validator.Validator
}

// NewUserHandler creates a UserHandler.
func NewUserHandler(userService service.UserService, v *validator.Validator) *UserHandler {
	return &UserHandler{
		userService: userService,
		validator:   v,
	}
}

// ═══════════════════════════════════════════════════════════════════
// CREATE
// ═══════════════════════════════════════════════════════════════════

// Create handles POST /api/v1/users.
func (h *UserHandler) Create(c *fiber.Ctx) error {
	actor := ctxutil.GetUserFromFiber(c)
	if actor == nil {
		return response.Error(c, domain.NewAuthenticationError("not authenticated"))
	}

	var req domain.CreateUserRequest
	if err := c.BodyParser(&req); err != nil {
		return response.Error(c, domain.NewValidationError("invalid request body"))
	}
	if err := h.validator.Validate(&req); err != nil {
		return response.Error(c, err)
	}

	user, err := h.userService.CreateUser(c.UserContext(), actor, &req)
	if err != nil {
		return response.Error(c, err)
	}

	return response.Created(c, user)
}

// ═══════════════════════════════════════════════════════════════════
// FIRST LOGIN SETUP (self)
// ═══════════════════════════════════════════════════════════════════

// Setup handles POST /api/v1/users/me/setup.
// The authenticated user completes their profile and changes their password.
func (h *UserHandler) Setup(c *fiber.Ctx) error {
	actor := ctxutil.GetUserFromFiber(c)
	if actor == nil {
		return response.Error(c, domain.NewAuthenticationError("not authenticated"))
	}

	var req domain.FirstLoginSetupRequest
	if err := c.BodyParser(&req); err != nil {
		return response.Error(c, domain.NewValidationError("invalid request body"))
	}
	if err := h.validator.Validate(&req); err != nil {
		return response.Error(c, err)
	}

	if err := h.userService.FirstLoginSetup(c.UserContext(), actor.ID, &req); err != nil {
		return response.Error(c, err)
	}

	return response.Success(c, fiber.Map{
		"message": "profile completed successfully",
	})
}

// ═══════════════════════════════════════════════════════════════════
// READ
// ═══════════════════════════════════════════════════════════════════

// GetByID handles GET /api/v1/users/:id.
func (h *UserHandler) GetByID(c *fiber.Ctx) error {
	actor := ctxutil.GetUserFromFiber(c)
	if actor == nil {
		return response.Error(c, domain.NewAuthenticationError("not authenticated"))
	}

	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return response.Error(c, domain.NewValidationError("invalid user ID"))
	}

	user, err := h.userService.GetByID(c.UserContext(), actor, id)
	if err != nil {
		return response.Error(c, err)
	}

	return response.Success(c, user)
}

// List handles GET /api/v1/users.
func (h *UserHandler) List(c *fiber.Ctx) error {
	actor := ctxutil.GetUserFromFiber(c)
	if actor == nil {
		return response.Error(c, domain.NewAuthenticationError("not authenticated"))
	}

	filter := domain.UserFilter{
		Search: c.Query("search"),
		Role:   domain.Role(c.Query("role")),
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

	if v := c.Query("restaurant_id"); v != "" {
		id, err := uuid.Parse(v)
		if err != nil {
			return response.Error(c, domain.NewValidationError("invalid restaurant_id"))
		}
		filter.RestaurantID = &id
	}

	if v := c.Query("is_active"); v != "" {
		active := v == "true"
		filter.IsActive = &active
	}

	users, total, err := h.userService.List(c.UserContext(), actor, filter)
	if err != nil {
		return response.Error(c, err)
	}

	return response.Success(c, fiber.Map{
		"items":  users,
		"total":  total,
		"limit":  filter.Limit,
		"offset": filter.Offset,
	})
}

// ═══════════════════════════════════════════════════════════════════
// UPDATE
// ═══════════════════════════════════════════════════════════════════

// Update handles PUT /api/v1/users/:id.
func (h *UserHandler) Update(c *fiber.Ctx) error {
	actor := ctxutil.GetUserFromFiber(c)
	if actor == nil {
		return response.Error(c, domain.NewAuthenticationError("not authenticated"))
	}

	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return response.Error(c, domain.NewValidationError("invalid user ID"))
	}

	var req domain.UpdateUserRequest
	if err := c.BodyParser(&req); err != nil {
		return response.Error(c, domain.NewValidationError("invalid request body"))
	}
	if err := h.validator.Validate(&req); err != nil {
		return response.Error(c, err)
	}

	user, err := h.userService.Update(c.UserContext(), actor, id, &req)
	if err != nil {
		return response.Error(c, err)
	}

	return response.Success(c, user)
}

// ═══════════════════════════════════════════════════════════════════
// DELETE
// ═══════════════════════════════════════════════════════════════════

// Delete handles DELETE /api/v1/users/:id.
func (h *UserHandler) Delete(c *fiber.Ctx) error {
	actor := ctxutil.GetUserFromFiber(c)
	if actor == nil {
		return response.Error(c, domain.NewAuthenticationError("not authenticated"))
	}

	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return response.Error(c, domain.NewValidationError("invalid user ID"))
	}

	if err := h.userService.Delete(c.UserContext(), actor, id); err != nil {
		return response.Error(c, err)
	}

	return response.Success(c, fiber.Map{
		"message": "user deleted successfully",
	})
}

// ═══════════════════════════════════════════════════════════════════
// CHANGE PASSWORD (self)
// ═══════════════════════════════════════════════════════════════════

// ChangePasswordRequest is the payload for changing a password.
type ChangePasswordRequest struct {
	OldPassword string `json:"old_password" validate:"required,min=8,max=72"`
	NewPassword string `json:"new_password" validate:"required,min=8,max=72"`
}

// ChangePassword handles PUT /api/v1/users/me/password.
func (h *UserHandler) ChangePassword(c *fiber.Ctx) error {
	actor := ctxutil.GetUserFromFiber(c)
	if actor == nil {
		return response.Error(c, domain.NewAuthenticationError("not authenticated"))
	}

	var req ChangePasswordRequest
	if err := c.BodyParser(&req); err != nil {
		return response.Error(c, domain.NewValidationError("invalid request body"))
	}
	if err := h.validator.Validate(&req); err != nil {
		return response.Error(c, err)
	}

	if err := h.userService.ChangePassword(c.UserContext(), actor.ID, req.OldPassword, req.NewPassword); err != nil {
		return response.Error(c, err)
	}

	return response.Success(c, fiber.Map{
		"message": "password changed successfully",
	})
}

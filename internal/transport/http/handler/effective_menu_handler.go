package handler

import (
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"

	"github.com/ramisoul84/kfc-crm/internal/domain"
	"github.com/ramisoul84/kfc-crm/internal/service"
	"github.com/ramisoul84/kfc-crm/internal/transport/http/response"
	"github.com/ramisoul84/kfc-crm/pkg/ctxutil"
)

// EffectiveMenuHandler handles the effective menu endpoint.
type EffectiveMenuHandler struct {
	effectiveMenuService service.EffectiveMenuService
}

// NewEffectiveMenuHandler creates an EffectiveMenuHandler.
func NewEffectiveMenuHandler(effectiveMenuService service.EffectiveMenuService) *EffectiveMenuHandler {
	return &EffectiveMenuHandler{
		effectiveMenuService: effectiveMenuService,
	}
}

// Get handles GET /api/v1/restaurants/:restaurant_id/menu.
func (h *EffectiveMenuHandler) Get(c *fiber.Ctx) error {
	actor := ctxutil.GetUserFromFiber(c)
	if actor == nil {
		return response.Error(c, domain.NewAuthenticationError("not authenticated"))
	}

	restaurantID, err := uuid.Parse(c.Params("restaurant_id"))
	if err != nil {
		return response.Error(c, domain.NewValidationError("invalid restaurant ID"))
	}

	menu, err := h.effectiveMenuService.GetEffectiveMenu(c.UserContext(), actor, restaurantID)
	if err != nil {
		return response.Error(c, err)
	}

	return response.Success(c, menu)
}

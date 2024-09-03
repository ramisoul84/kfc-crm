package service

import (
	"context"

	"github.com/google/uuid"

	"github.com/ramisoul84/kfc-crm/internal/domain"
	"github.com/ramisoul84/kfc-crm/internal/repository"
	"github.com/ramisoul84/kfc-crm/pkg/ctxutil"
	"github.com/ramisoul84/kfc-crm/pkg/logger"
)

// MenuItemOverrideService handles restaurant menu overrides.
type MenuItemOverrideService interface {
	Set(ctx context.Context, actor *domain.User, req *domain.CreateMenuItemOverrideRequest) (*domain.MenuItemOverrideResponse, error)
	GetByID(ctx context.Context, actor *domain.User, id uuid.UUID) (*domain.MenuItemOverrideResponse, error)
	ListByRestaurant(ctx context.Context, actor *domain.User, restaurantID uuid.UUID) ([]*domain.MenuItemOverrideResponse, error)
	Update(ctx context.Context, actor *domain.User, id uuid.UUID, req *domain.UpdateMenuItemOverrideRequest) (*domain.MenuItemOverrideResponse, error)
	Delete(ctx context.Context, actor *domain.User, id uuid.UUID) error
}

type menuItemOverrideService struct {
	overrideRepo repository.MenuItemOverrideRepository
	itemRepo     repository.MenuItemRepository
	rbacService  RBACService
	logger       *logger.Logger
}

// NewMenuItemOverrideService creates a MenuItemOverrideService.
func NewMenuItemOverrideService(
	overrideRepo repository.MenuItemOverrideRepository,
	itemRepo repository.MenuItemRepository,
	rbacService RBACService,
	log *logger.Logger,
) MenuItemOverrideService {
	return &menuItemOverrideService{
		overrideRepo: overrideRepo,
		itemRepo:     itemRepo,
		rbacService:  rbacService,
		logger:       log,
	}
}

// ═══════════════════════════════════════════════════════════════════
// SET (create or update)
// ═══════════════════════════════════════════════════════════════════

func (s *menuItemOverrideService) Set(
	ctx context.Context,
	actor *domain.User,
	req *domain.CreateMenuItemOverrideRequest,
) (*domain.MenuItemOverrideResponse, error) {
	log := s.logger.WithRequestID(ctxutil.GetRequestID(ctx))

	log.Debug("setting menu item override",
		"actor_id", actor.ID,
		"restaurant_id", req.RestaurantID,
		"menu_item_id", req.MenuItemID,
	)

	// 1. Permission
	if err := s.rbacService.CheckPermission(actor, domain.PermMenuOverrideWrite); err != nil {
		log.Warn("set override denied: permission",
			"actor_id", actor.ID,
			"role", actor.Role,
			"error", err,
		)
		return nil, err
	}

	// 2. Scope — can the actor act on this restaurant?
	if err := s.rbacService.CanAccessRestaurant(ctx, actor, req.RestaurantID); err != nil {
		log.Warn("set override denied: scope",
			"actor_id", actor.ID,
			"restaurant_id", req.RestaurantID,
			"error", err,
		)
		return nil, err
	}

	// 3. Menu item must exist
	if _, err := s.itemRepo.GetByID(ctx, req.MenuItemID); err != nil {
		log.Warn("set override: menu item not found",
			"menu_item_id", req.MenuItemID,
			"error", err,
		)
		return nil, err
	}

	// 4. At least one field must be set
	if req.PriceOverride == nil && req.IsAvailableOverride == nil {
		return nil, domain.NewValidationError(
			"override must set price_override or is_available_override",
		)
	}

	// 5. Build and upsert
	override := &domain.RestaurantMenuItemOverride{
		ID:                  uuid.New(),
		RestaurantID:        req.RestaurantID,
		MenuItemID:          req.MenuItemID,
		PriceOverride:       req.PriceOverride,
		IsAvailableOverride: req.IsAvailableOverride,
		Reason:              req.Reason,
	}

	if err := s.overrideRepo.Upsert(ctx, override); err != nil {
		log.Error("set override failed",
			"restaurant_id", req.RestaurantID,
			"menu_item_id", req.MenuItemID,
			"error", err,
		)
		return nil, err
	}

	log.Info("menu item override set",
		"override_id", override.ID,
		"restaurant_id", override.RestaurantID,
		"menu_item_id", override.MenuItemID,
		"actor_id", actor.ID,
	)

	return override.ToResponse(), nil
}

// ═══════════════════════════════════════════════════════════════════
// READ
// ═══════════════════════════════════════════════════════════════════

func (s *menuItemOverrideService) GetByID(
	ctx context.Context,
	actor *domain.User,
	id uuid.UUID,
) (*domain.MenuItemOverrideResponse, error) {
	log := s.logger.WithRequestID(ctxutil.GetRequestID(ctx))

	// Permission
	if err := s.rbacService.CheckPermission(actor, domain.PermMenuOverrideRead); err != nil {
		log.Warn("get override denied", "actor_id", actor.ID, "error", err)
		return nil, err
	}

	override, err := s.overrideRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	// Scope
	if err := s.rbacService.CanAccessRestaurant(ctx, actor, override.RestaurantID); err != nil {
		log.Warn("get override denied: scope",
			"actor_id", actor.ID,
			"override_id", id,
			"error", err,
		)
		return nil, err
	}

	return override.ToResponse(), nil
}

func (s *menuItemOverrideService) ListByRestaurant(
	ctx context.Context,
	actor *domain.User,
	restaurantID uuid.UUID,
) ([]*domain.MenuItemOverrideResponse, error) {
	log := s.logger.WithRequestID(ctxutil.GetRequestID(ctx))

	// Permission
	if err := s.rbacService.CheckPermission(actor, domain.PermMenuOverrideRead); err != nil {
		log.Warn("list overrides denied", "actor_id", actor.ID, "error", err)
		return nil, err
	}

	// Scope
	if err := s.rbacService.CanAccessRestaurant(ctx, actor, restaurantID); err != nil {
		log.Warn("list overrides denied: scope",
			"actor_id", actor.ID,
			"restaurant_id", restaurantID,
			"error", err,
		)
		return nil, err
	}

	overrides, err := s.overrideRepo.ListByRestaurant(ctx, restaurantID)
	if err != nil {
		log.Error("list overrides failed", "error", err)
		return nil, err
	}

	out := make([]*domain.MenuItemOverrideResponse, 0, len(overrides))
	for _, o := range overrides {
		out = append(out, o.ToResponse())
	}
	return out, nil
}

// ═══════════════════════════════════════════════════════════════════
// UPDATE
// ═══════════════════════════════════════════════════════════════════

func (s *menuItemOverrideService) Update(
	ctx context.Context,
	actor *domain.User,
	id uuid.UUID,
	req *domain.UpdateMenuItemOverrideRequest,
) (*domain.MenuItemOverrideResponse, error) {
	log := s.logger.WithRequestID(ctxutil.GetRequestID(ctx))

	// 1. Permission
	if err := s.rbacService.CheckPermission(actor, domain.PermMenuOverrideWrite); err != nil {
		log.Warn("update override denied: permission", "actor_id", actor.ID, "error", err)
		return nil, err
	}

	// 2. Load
	override, err := s.overrideRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	// 3. Scope
	if err := s.rbacService.CanAccessRestaurant(ctx, actor, override.RestaurantID); err != nil {
		log.Warn("update override denied: scope",
			"actor_id", actor.ID,
			"override_id", id,
			"error", err,
		)
		return nil, err
	}

	// 4. At least one field must be set
	if req.PriceOverride == nil && req.IsAvailableOverride == nil {
		return nil, domain.NewValidationError(
			"override must set price_override or is_available_override",
		)
	}

	// 5. Apply
	override.PriceOverride = req.PriceOverride
	override.IsAvailableOverride = req.IsAvailableOverride
	override.Reason = req.Reason

	// 6. Persist
	if err := s.overrideRepo.Update(ctx, override); err != nil {
		log.Error("update override failed", "override_id", id, "error", err)
		return nil, err
	}

	log.Info("menu item override updated", "override_id", id, "actor_id", actor.ID)

	return override.ToResponse(), nil
}

// ═══════════════════════════════════════════════════════════════════
// DELETE
// ═══════════════════════════════════════════════════════════════════

func (s *menuItemOverrideService) Delete(
	ctx context.Context,
	actor *domain.User,
	id uuid.UUID,
) error {
	log := s.logger.WithRequestID(ctxutil.GetRequestID(ctx))

	// 1. Permission
	if err := s.rbacService.CheckPermission(actor, domain.PermMenuOverrideDelete); err != nil {
		log.Warn("delete override denied: permission", "actor_id", actor.ID, "error", err)
		return err
	}

	// 2. Load
	override, err := s.overrideRepo.GetByID(ctx, id)
	if err != nil {
		return err
	}

	// 3. Scope
	if err := s.rbacService.CanAccessRestaurant(ctx, actor, override.RestaurantID); err != nil {
		log.Warn("delete override denied: scope",
			"actor_id", actor.ID,
			"override_id", id,
			"error", err,
		)
		return err
	}

	// 4. Delete
	if err := s.overrideRepo.Delete(ctx, id); err != nil {
		log.Error("delete override failed", "override_id", id, "error", err)
		return err
	}

	log.Info("menu item override deleted", "override_id", id, "actor_id", actor.ID)

	return nil
}

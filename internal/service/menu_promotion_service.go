package service

import (
	"context"

	"github.com/google/uuid"

	"github.com/ramisoul84/kfc-crm/internal/domain"
	"github.com/ramisoul84/kfc-crm/internal/repository"
	"github.com/ramisoul84/kfc-crm/pkg/ctxutil"
	"github.com/ramisoul84/kfc-crm/pkg/logger"
)

// MenuPromotionService handles promotion business logic.
type MenuPromotionService interface {
	Create(ctx context.Context, actor *domain.User, req *domain.CreatePromotionRequest) (*domain.PromotionResponse, error)
	GetByID(ctx context.Context, actor *domain.User, id uuid.UUID) (*domain.PromotionResponse, error)
	List(ctx context.Context, actor *domain.User, filter domain.PromotionFilter) ([]*domain.PromotionResponse, int, error)
	Update(ctx context.Context, actor *domain.User, id uuid.UUID, req *domain.UpdatePromotionRequest) (*domain.PromotionResponse, error)
	Delete(ctx context.Context, actor *domain.User, id uuid.UUID) error
}

type menuPromotionService struct {
	promotionRepo repository.MenuPromotionRepository
	rbacService   RBACService
	logger        *logger.Logger
}

// NewMenuPromotionService creates a MenuPromotionService.
func NewMenuPromotionService(
	promotionRepo repository.MenuPromotionRepository,
	rbacService RBACService,
	log *logger.Logger,
) MenuPromotionService {
	return &menuPromotionService{
		promotionRepo: promotionRepo,
		rbacService:   rbacService,
		logger:        log,
	}
}

// ═══════════════════════════════════════════════════════════════════
// CREATE
// ═══════════════════════════════════════════════════════════════════

func (s *menuPromotionService) Create(
	ctx context.Context,
	actor *domain.User,
	req *domain.CreatePromotionRequest,
) (*domain.PromotionResponse, error) {
	log := s.logger.WithRequestID(ctxutil.GetRequestID(ctx))

	log.Debug("creating promotion",
		"actor_id", actor.ID,
		"name", req.Name,
		"scope", req.Scope,
	)

	// 1. Permission
	if err := s.rbacService.CheckPermission(actor, domain.PermMenuPromotionWrite); err != nil {
		log.Warn("create promotion denied: permission",
			"actor_id", actor.ID,
			"role", actor.Role,
			"error", err,
		)
		return nil, err
	}

	// 2. Validate scope-specific fields
	if err := validatePromotionScopeFields(req); err != nil {
		return nil, err
	}

	// 3. Scope check — RM can only create promotions in their scope
	if err := s.checkPromotionScope(ctx, actor, req.Scope, req.RegionID, req.RestaurantID); err != nil {
		log.Warn("create promotion denied: scope",
			"actor_id", actor.ID,
			"scope", req.Scope,
			"error", err,
		)
		return nil, err
	}

	// 4. Build entity
	isActive := true
	if req.IsActive != nil {
		isActive = *req.IsActive
	}

	promotion := &domain.Promotion{
		ID:            uuid.New(),
		Name:          req.Name,
		Description:   req.Description,
		Scope:         req.Scope,
		RegionID:      req.RegionID,
		RestaurantID:  req.RestaurantID,
		DiscountType:  req.DiscountType,
		DiscountValue: req.DiscountValue,
		StartsAt:      req.StartsAt,
		EndsAt:        req.EndsAt,
		IsActive:      isActive,
		Priority:      req.Priority,
		MenuItemIDs:   req.MenuItemIDs,
	}

	// 5. Persist
	if err := s.promotionRepo.Create(ctx, promotion); err != nil {
		log.Error("create promotion failed", "error", err)
		return nil, err
	}

	log.Info("promotion created",
		"promotion_id", promotion.ID,
		"name", promotion.Name,
		"scope", promotion.Scope,
		"actor_id", actor.ID,
	)

	return promotion.ToResponse(), nil
}

// ═══════════════════════════════════════════════════════════════════
// READ
// ═══════════════════════════════════════════════════════════════════

func (s *menuPromotionService) GetByID(
	ctx context.Context,
	actor *domain.User,
	id uuid.UUID,
) (*domain.PromotionResponse, error) {
	log := s.logger.WithRequestID(ctxutil.GetRequestID(ctx))

	if err := s.rbacService.CheckPermission(actor, domain.PermMenuPromotionRead); err != nil {
		log.Warn("get promotion denied", "actor_id", actor.ID, "error", err)
		return nil, err
	}

	promotion, err := s.promotionRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	// Scope check on read
	if err := s.checkPromotionScope(ctx, actor, promotion.Scope, promotion.RegionID, promotion.RestaurantID); err != nil {
		log.Warn("get promotion denied: scope", "actor_id", actor.ID, "promotion_id", id, "error", err)
		return nil, err
	}

	return promotion.ToResponse(), nil
}

func (s *menuPromotionService) List(
	ctx context.Context,
	actor *domain.User,
	filter domain.PromotionFilter,
) ([]*domain.PromotionResponse, int, error) {
	log := s.logger.WithRequestID(ctxutil.GetRequestID(ctx))

	if err := s.rbacService.CheckPermission(actor, domain.PermMenuPromotionRead); err != nil {
		log.Warn("list promotions denied", "actor_id", actor.ID, "error", err)
		return nil, 0, err
	}

	// Apply scope filter based on role
	switch actor.Role {
	case domain.RoleSuperAdmin:
		// no filter
	case domain.RoleRegionalManager:
		// RM sees promotions in their region or their restaurants
		filter.RegionID = actor.RegionID
	case domain.RoleRestaurantManager, domain.RoleShiftManager, domain.RoleCashier:
		// See only their restaurant's promotions (or global)
		filter.RestaurantID = actor.RestaurantID
	}

	promotions, total, err := s.promotionRepo.List(ctx, filter)
	if err != nil {
		log.Error("list promotions failed", "error", err)
		return nil, 0, err
	}

	out := make([]*domain.PromotionResponse, 0, len(promotions))
	for _, p := range promotions {
		out = append(out, p.ToResponse())
	}
	return out, total, nil
}

// ═══════════════════════════════════════════════════════════════════
// UPDATE
// ═══════════════════════════════════════════════════════════════════

func (s *menuPromotionService) Update(
	ctx context.Context,
	actor *domain.User,
	id uuid.UUID,
	req *domain.UpdatePromotionRequest,
) (*domain.PromotionResponse, error) {
	log := s.logger.WithRequestID(ctxutil.GetRequestID(ctx))

	// 1. Permission
	if err := s.rbacService.CheckPermission(actor, domain.PermMenuPromotionWrite); err != nil {
		log.Warn("update promotion denied: permission", "actor_id", actor.ID, "error", err)
		return nil, err
	}

	// 2. Load
	promotion, err := s.promotionRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	// 3. Scope
	if err := s.checkPromotionScope(ctx, actor, promotion.Scope, promotion.RegionID, promotion.RestaurantID); err != nil {
		log.Warn("update promotion denied: scope", "actor_id", actor.ID, "promotion_id", id, "error", err)
		return nil, err
	}

	// 4. Apply updates
	promotion.Name = req.Name
	promotion.Description = req.Description
	promotion.DiscountType = req.DiscountType
	promotion.DiscountValue = req.DiscountValue
	promotion.StartsAt = req.StartsAt
	promotion.EndsAt = req.EndsAt
	if req.IsActive != nil {
		promotion.IsActive = *req.IsActive
	}
	if req.Priority != nil {
		promotion.Priority = *req.Priority
	}
	promotion.MenuItemIDs = req.MenuItemIDs

	// 5. Persist
	if err := s.promotionRepo.Update(ctx, promotion); err != nil {
		log.Error("update promotion failed", "promotion_id", id, "error", err)
		return nil, err
	}

	log.Info("promotion updated", "promotion_id", id, "actor_id", actor.ID)

	return promotion.ToResponse(), nil
}

// ═══════════════════════════════════════════════════════════════════
// DELETE
// ═══════════════════════════════════════════════════════════════════

func (s *menuPromotionService) Delete(
	ctx context.Context,
	actor *domain.User,
	id uuid.UUID,
) error {
	log := s.logger.WithRequestID(ctxutil.GetRequestID(ctx))

	// 1. Permission
	if err := s.rbacService.CheckPermission(actor, domain.PermMenuPromotionDelete); err != nil {
		log.Warn("delete promotion denied: permission", "actor_id", actor.ID, "error", err)
		return err
	}

	// 2. Load
	promotion, err := s.promotionRepo.GetByID(ctx, id)
	if err != nil {
		return err
	}

	// 3. Scope
	if err := s.checkPromotionScope(ctx, actor, promotion.Scope, promotion.RegionID, promotion.RestaurantID); err != nil {
		log.Warn("delete promotion denied: scope", "actor_id", actor.ID, "promotion_id", id, "error", err)
		return err
	}

	// 4. Delete
	if err := s.promotionRepo.Delete(ctx, id); err != nil {
		log.Error("delete promotion failed", "promotion_id", id, "error", err)
		return err
	}

	log.Info("promotion deleted", "promotion_id", id, "actor_id", actor.ID)

	return nil
}

// ═══════════════════════════════════════════════════════════════════
// HELPERS
// ═══════════════════════════════════════════════════════════════════

// validatePromotionScopeFields ensures the scope-specific fields are set correctly.
func validatePromotionScopeFields(req *domain.CreatePromotionRequest) error {
	switch req.Scope {
	case domain.PromotionScopeGlobal:
		if req.RegionID != nil || req.RestaurantID != nil {
			return domain.NewValidationError("global promotions cannot have region_id or restaurant_id")
		}
	case domain.PromotionScopeRegion:
		if req.RegionID == nil {
			return domain.NewValidationError("region_id is required for regional promotions")
		}
		if req.RestaurantID != nil {
			return domain.NewValidationError("restaurant_id should not be set for regional promotions")
		}
	case domain.PromotionScopeRestaurant:
		if req.RestaurantID == nil {
			return domain.NewValidationError("restaurant_id is required for restaurant promotions")
		}
		if req.RegionID != nil {
			return domain.NewValidationError("region_id should not be set for restaurant promotions")
		}
	default:
		return domain.NewValidationError("invalid promotion scope")
	}
	return nil
}

// checkPromotionScope ensures the actor can manage promotions in the given scope.
func (s *menuPromotionService) checkPromotionScope(
	ctx context.Context,
	actor *domain.User,
	scope domain.PromotionScope,
	regionID, restaurantID *uuid.UUID,
) error {
	switch actor.Role {
	case domain.RoleSuperAdmin:
		return nil
	case domain.RoleRegionalManager:
		// RM can create/manage global and regional promotions, or restaurant-level in their region
		if scope == domain.PromotionScopeGlobal {
			return nil
		}
		if scope == domain.PromotionScopeRegion {
			if actor.RegionID == nil || regionID == nil || *actor.RegionID != *regionID {
				return domain.NewAuthorizationError("cannot manage promotions outside your region")
			}
			return nil
		}
		if scope == domain.PromotionScopeRestaurant {
			if restaurantID == nil {
				return domain.NewValidationError("restaurant_id is required")
			}
			return s.rbacService.CanAccessRestaurant(ctx, actor, *restaurantID)
		}
		return nil
	default:
		return domain.NewAuthorizationError("insufficient permissions to manage promotions")
	}
}

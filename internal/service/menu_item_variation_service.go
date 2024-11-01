package service

import (
	"context"

	"github.com/google/uuid"

	"github.com/ramisoul84/kfc-crm/internal/domain"
	"github.com/ramisoul84/kfc-crm/internal/repository"
	"github.com/ramisoul84/kfc-crm/pkg/ctxutil"
	"github.com/ramisoul84/kfc-crm/pkg/logger"
)

// MenuItemVariationService handles menu item variation business logic.
type MenuItemVariationService interface {
	Create(ctx context.Context, actor *domain.User, req *domain.CreateMenuItemVariationRequest) (*domain.MenuItemVariationResponse, error)
	GetByID(ctx context.Context, actor *domain.User, id uuid.UUID) (*domain.MenuItemVariationResponse, error)
	ListByItem(ctx context.Context, actor *domain.User, itemID uuid.UUID) ([]*domain.MenuItemVariationResponse, error)
	Update(ctx context.Context, actor *domain.User, id uuid.UUID, req *domain.UpdateMenuItemVariationRequest) (*domain.MenuItemVariationResponse, error)
	Delete(ctx context.Context, actor *domain.User, id uuid.UUID) error
}

type menuItemVariationService struct {
	variationRepo repository.MenuItemVariationRepository
	itemRepo      repository.MenuItemRepository
	rbacService   RBACService
	logger        *logger.Logger
}

// NewMenuItemVariationService creates a MenuItemVariationService.
func NewMenuItemVariationService(
	variationRepo repository.MenuItemVariationRepository,
	itemRepo repository.MenuItemRepository,
	rbacService RBACService,
	log *logger.Logger,
) MenuItemVariationService {
	return &menuItemVariationService{
		variationRepo: variationRepo,
		itemRepo:      itemRepo,
		rbacService:   rbacService,
		logger:        log,
	}
}

// ═══════════════════════════════════════════════════════════════════
// CREATE
// ═══════════════════════════════════════════════════════════════════

func (s *menuItemVariationService) Create(
	ctx context.Context,
	actor *domain.User,
	req *domain.CreateMenuItemVariationRequest,
) (*domain.MenuItemVariationResponse, error) {
	log := s.logger.WithRequestID(ctxutil.GetRequestID(ctx))

	log.Debug("creating menu item variation",
		"actor_id", actor.ID,
		"menu_item_id", req.MenuItemID,
		"name", req.Name,
	)

	// 1. Permission — super admin only
	if err := s.rbacService.CheckPermission(actor, domain.PermMenuVariationWrite); err != nil {
		log.Warn("create variation denied",
			"actor_id", actor.ID,
			"role", actor.Role,
			"error", err,
		)
		return nil, err
	}

	// 2. Parent item must exist
	if _, err := s.itemRepo.GetByID(ctx, req.MenuItemID); err != nil {
		log.Warn("create variation: item not found", "menu_item_id", req.MenuItemID, "error", err)
		return nil, err
	}

	// 3. Build entity
	variation := &domain.MenuItemVariation{
		ID:         uuid.New(),
		MenuItemID: req.MenuItemID,
		Name:       req.Name,
		PriceDelta: req.PriceDelta,
		IsDefault:  req.IsDefault,
		SortOrder:  req.SortOrder,
		IsActive:   true,
	}

	// 4. Persist
	if err := s.variationRepo.Create(ctx, variation); err != nil {
		log.Error("create variation failed",
			"variation_id", variation.ID,
			"menu_item_id", req.MenuItemID,
			"error", err,
		)
		return nil, err
	}

	log.Info("menu item variation created",
		"variation_id", variation.ID,
		"menu_item_id", req.MenuItemID,
		"name", variation.Name,
		"actor_id", actor.ID,
	)

	return variation.ToResponse(), nil
}

// ═══════════════════════════════════════════════════════════════════
// READ
// ═══════════════════════════════════════════════════════════════════

func (s *menuItemVariationService) GetByID(
	ctx context.Context,
	actor *domain.User,
	id uuid.UUID,
) (*domain.MenuItemVariationResponse, error) {
	log := s.logger.WithRequestID(ctxutil.GetRequestID(ctx))

	if err := s.rbacService.CheckPermission(actor, domain.PermMenuVariationRead); err != nil {
		log.Warn("get variation denied", "actor_id", actor.ID, "error", err)
		return nil, err
	}

	v, err := s.variationRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	return v.ToResponse(), nil
}

func (s *menuItemVariationService) ListByItem(
	ctx context.Context,
	actor *domain.User,
	itemID uuid.UUID,
) ([]*domain.MenuItemVariationResponse, error) {
	log := s.logger.WithRequestID(ctxutil.GetRequestID(ctx))

	if err := s.rbacService.CheckPermission(actor, domain.PermMenuVariationRead); err != nil {
		log.Warn("list variations denied", "actor_id", actor.ID, "error", err)
		return nil, err
	}

	variations, err := s.variationRepo.ListByItem(ctx, itemID)
	if err != nil {
		log.Error("list variations failed", "error", err)
		return nil, err
	}

	out := make([]*domain.MenuItemVariationResponse, 0, len(variations))
	for _, v := range variations {
		out = append(out, v.ToResponse())
	}
	return out, nil
}

// ═══════════════════════════════════════════════════════════════════
// UPDATE
// ═══════════════════════════════════════════════════════════════════

func (s *menuItemVariationService) Update(
	ctx context.Context,
	actor *domain.User,
	id uuid.UUID,
	req *domain.UpdateMenuItemVariationRequest,
) (*domain.MenuItemVariationResponse, error) {
	log := s.logger.WithRequestID(ctxutil.GetRequestID(ctx))

	if err := s.rbacService.CheckPermission(actor, domain.PermMenuVariationWrite); err != nil {
		log.Warn("update variation denied", "actor_id", actor.ID, "error", err)
		return nil, err
	}

	v, err := s.variationRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	v.Name = req.Name
	v.PriceDelta = req.PriceDelta
	v.IsDefault = req.IsDefault
	v.SortOrder = req.SortOrder
	if req.IsActive != nil {
		v.IsActive = *req.IsActive
	}

	if err := s.variationRepo.Update(ctx, v); err != nil {
		log.Error("update variation failed", "variation_id", id, "error", err)
		return nil, err
	}

	log.Info("menu item variation updated", "variation_id", id, "actor_id", actor.ID)

	return v.ToResponse(), nil
}

// ═══════════════════════════════════════════════════════════════════
// DELETE
// ═══════════════════════════════════════════════════════════════════

func (s *menuItemVariationService) Delete(
	ctx context.Context,
	actor *domain.User,
	id uuid.UUID,
) error {
	log := s.logger.WithRequestID(ctxutil.GetRequestID(ctx))

	if err := s.rbacService.CheckPermission(actor, domain.PermMenuVariationDelete); err != nil {
		log.Warn("delete variation denied", "actor_id", actor.ID, "error", err)
		return err
	}

	if err := s.variationRepo.Delete(ctx, id); err != nil {
		log.Error("delete variation failed", "variation_id", id, "error", err)
		return err
	}

	log.Info("menu item variation deleted", "variation_id", id, "actor_id", actor.ID)

	return nil
}

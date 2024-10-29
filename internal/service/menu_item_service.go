package service

import (
	"context"

	"github.com/google/uuid"

	"github.com/ramisoul84/kfc-crm/internal/domain"
	"github.com/ramisoul84/kfc-crm/internal/repository"
	"github.com/ramisoul84/kfc-crm/pkg/ctxutil"
	"github.com/ramisoul84/kfc-crm/pkg/logger"
)

// MenuItemService handles menu item business logic.
type MenuItemService interface {
	Create(ctx context.Context, actor *domain.User, req *domain.CreateMenuItemRequest) (*domain.MenuItemResponse, error)
	GetByID(ctx context.Context, actor *domain.User, id uuid.UUID) (*domain.MenuItemResponse, error)
	List(ctx context.Context, actor *domain.User, filter domain.MenuItemFilter) ([]*domain.MenuItemResponse, int, error)
	Update(ctx context.Context, actor *domain.User, id uuid.UUID, req *domain.UpdateMenuItemRequest) (*domain.MenuItemResponse, error)
	Delete(ctx context.Context, actor *domain.User, id uuid.UUID) error
}

type menuItemService struct {
	itemRepo     repository.MenuItemRepository
	categoryRepo repository.MenuCategoryRepository
	rbacService  RBACService
	logger       *logger.Logger
}

// NewMenuItemService creates a MenuItemService.
func NewMenuItemService(
	itemRepo repository.MenuItemRepository,
	categoryRepo repository.MenuCategoryRepository,
	rbacService RBACService,
	log *logger.Logger,
) MenuItemService {
	return &menuItemService{
		itemRepo:     itemRepo,
		categoryRepo: categoryRepo,
		rbacService:  rbacService,
		logger:       log,
	}
}

// ═══════════════════════════════════════════════════════════════════
// CREATE
// ═══════════════════════════════════════════════════════════════════

func (s *menuItemService) Create(
	ctx context.Context,
	actor *domain.User,
	req *domain.CreateMenuItemRequest,
) (*domain.MenuItemResponse, error) {
	log := s.logger.WithRequestID(ctxutil.GetRequestID(ctx))

	log.Debug("creating menu item",
		"actor_id", actor.ID,
		"product_code", req.ProductCode,
		"category_id", req.CategoryID,
	)

	// 1. Permission — super admin only
	if err := s.rbacService.CheckPermission(actor, domain.PermMenuItemWrite); err != nil {
		log.Warn("create menu item denied",
			"actor_id", actor.ID,
			"role", actor.Role,
			"error", err,
		)
		return nil, err
	}

	// 2. Category must exist
	if _, err := s.categoryRepo.GetByID(ctx, req.CategoryID); err != nil {
		log.Warn("create menu item: category not found",
			"category_id", req.CategoryID,
			"error", err,
		)
		return nil, err
	}

	// 3. Build entity
	item := &domain.MenuItem{
		ID:          uuid.New(),
		CategoryID:  req.CategoryID,
		ProductCode: req.ProductCode,
		Name:        req.Name,
		Description: req.Description,
		ImageURL:    req.ImageURL,
		BasePrice:   req.BasePrice,
		SortOrder:   req.SortOrder,
		IsActive:    true,
	}

	// 4. Persist
	if err := s.itemRepo.Create(ctx, item); err != nil {
		log.Error("create menu item failed",
			"item_id", item.ID,
			"product_code", item.ProductCode,
			"error", err,
		)
		return nil, err
	}

	log.Info("menu item created",
		"item_id", item.ID,
		"product_code", item.ProductCode,
		"actor_id", actor.ID,
	)

	return item.ToResponse(), nil
}

// ═══════════════════════════════════════════════════════════════════
// READ
// ═══════════════════════════════════════════════════════════════════

func (s *menuItemService) GetByID(
	ctx context.Context,
	actor *domain.User,
	id uuid.UUID,
) (*domain.MenuItemResponse, error) {
	log := s.logger.WithRequestID(ctxutil.GetRequestID(ctx))

	if err := s.rbacService.CheckPermission(actor, domain.PermMenuItemRead); err != nil {
		log.Warn("get menu item denied", "actor_id", actor.ID, "error", err)
		return nil, err
	}

	item, err := s.itemRepo.GetByIDWithCategory(ctx, id)
	if err != nil {
		return nil, err
	}

	return item.ToResponse(), nil
}

func (s *menuItemService) List(
	ctx context.Context,
	actor *domain.User,
	filter domain.MenuItemFilter,
) ([]*domain.MenuItemResponse, int, error) {
	log := s.logger.WithRequestID(ctxutil.GetRequestID(ctx))

	if err := s.rbacService.CheckPermission(actor, domain.PermMenuItemRead); err != nil {
		log.Warn("list menu items denied", "actor_id", actor.ID, "error", err)
		return nil, 0, err
	}

	items, total, err := s.itemRepo.List(ctx, filter)
	if err != nil {
		log.Error("list menu items failed", "error", err)
		return nil, 0, err
	}

	out := make([]*domain.MenuItemResponse, 0, len(items))
	for _, i := range items {
		out = append(out, i.ToResponse())
	}

	return out, total, nil
}

// ═══════════════════════════════════════════════════════════════════
// UPDATE
// ═══════════════════════════════════════════════════════════════════

func (s *menuItemService) Update(
	ctx context.Context,
	actor *domain.User,
	id uuid.UUID,
	req *domain.UpdateMenuItemRequest,
) (*domain.MenuItemResponse, error) {
	log := s.logger.WithRequestID(ctxutil.GetRequestID(ctx))

	if err := s.rbacService.CheckPermission(actor, domain.PermMenuItemWrite); err != nil {
		log.Warn("update menu item denied", "actor_id", actor.ID, "error", err)
		return nil, err
	}

	// Load
	item, err := s.itemRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	// If category is changing, verify the new one exists
	if req.CategoryID != item.CategoryID {
		if _, err := s.categoryRepo.GetByID(ctx, req.CategoryID); err != nil {
			return nil, err
		}
	}

	// Apply
	item.CategoryID = req.CategoryID
	item.Name = req.Name
	item.Description = req.Description
	item.ImageURL = req.ImageURL
	item.BasePrice = req.BasePrice
	item.SortOrder = req.SortOrder
	if req.IsActive != nil {
		item.IsActive = *req.IsActive
	}

	if err := s.itemRepo.Update(ctx, item); err != nil {
		log.Error("update menu item failed", "item_id", id, "error", err)
		return nil, err
	}

	log.Info("menu item updated", "item_id", id, "actor_id", actor.ID)

	return item.ToResponse(), nil
}

// ═══════════════════════════════════════════════════════════════════
// DELETE
// ═══════════════════════════════════════════════════════════════════

func (s *menuItemService) Delete(
	ctx context.Context,
	actor *domain.User,
	id uuid.UUID,
) error {
	log := s.logger.WithRequestID(ctxutil.GetRequestID(ctx))

	if err := s.rbacService.CheckPermission(actor, domain.PermMenuItemDelete); err != nil {
		log.Warn("delete menu item denied", "actor_id", actor.ID, "error", err)
		return err
	}

	if err := s.itemRepo.Delete(ctx, id); err != nil {
		log.Error("delete menu item failed", "item_id", id, "error", err)
		return err
	}

	log.Info("menu item deleted", "item_id", id, "actor_id", actor.ID)

	return nil
}

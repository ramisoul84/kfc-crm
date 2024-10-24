package service

import (
	"context"

	"github.com/google/uuid"

	"github.com/ramisoul84/kfc-crm/internal/domain"
	"github.com/ramisoul84/kfc-crm/internal/repository"
	"github.com/ramisoul84/kfc-crm/pkg/ctxutil"
	"github.com/ramisoul84/kfc-crm/pkg/logger"
)

// MenuCategoryService handles menu category business logic.
type MenuCategoryService interface {
	Create(ctx context.Context, actor *domain.User, req *domain.CreateMenuCategoryRequest) (*domain.MenuCategoryResponse, error)
	GetByID(ctx context.Context, actor *domain.User, id uuid.UUID) (*domain.MenuCategoryResponse, error)
	List(ctx context.Context, actor *domain.User, filter domain.MenuCategoryFilter) ([]*domain.MenuCategoryResponse, int, error)
	Update(ctx context.Context, actor *domain.User, id uuid.UUID, req *domain.UpdateMenuCategoryRequest) (*domain.MenuCategoryResponse, error)
	Delete(ctx context.Context, actor *domain.User, id uuid.UUID) error
}

type menuCategoryService struct {
	categoryRepo repository.MenuCategoryRepository
	rbacService  RBACService
	logger       *logger.Logger
}

// NewMenuCategoryService creates a MenuCategoryService.
func NewMenuCategoryService(
	categoryRepo repository.MenuCategoryRepository,
	rbacService RBACService,
	log *logger.Logger,
) MenuCategoryService {
	return &menuCategoryService{
		categoryRepo: categoryRepo,
		rbacService:  rbacService,
		logger:       log,
	}
}

// ═══════════════════════════════════════════════════════════════════
// CREATE
// ═══════════════════════════════════════════════════════════════════

func (s *menuCategoryService) Create(
	ctx context.Context,
	actor *domain.User,
	req *domain.CreateMenuCategoryRequest,
) (*domain.MenuCategoryResponse, error) {
	log := s.logger.WithRequestID(ctxutil.GetRequestID(ctx))

	log.Debug("creating menu category",
		"actor_id", actor.ID,
		"name", req.Name,
		"code", req.Code,
	)

	// 1. Permission
	if err := s.rbacService.CheckPermission(actor, domain.PermMenuWrite); err != nil {
		log.Warn("create menu category denied",
			"actor_id", actor.ID,
			"role", actor.Role,
			"error", err,
		)
		return nil, err
	}

	// 2. Build entity
	category := &domain.MenuCategory{
		ID:          uuid.New(),
		Name:        req.Name,
		Code:        req.Code,
		Description: req.Description,
		ImageURL:    req.ImageURL,
		SortOrder:   req.SortOrder,
		IsActive:    true,
	}

	// 3. Persist
	if err := s.categoryRepo.Create(ctx, category); err != nil {
		log.Error("create menu category failed",
			"category_id", category.ID,
			"code", category.Code,
			"error", err,
		)
		return nil, err
	}

	log.Info("menu category created",
		"category_id", category.ID,
		"code", category.Code,
		"actor_id", actor.ID,
	)

	return category.ToResponse(), nil
}

// ═══════════════════════════════════════════════════════════════════
// READ
// ═══════════════════════════════════════════════════════════════════

func (s *menuCategoryService) GetByID(
	ctx context.Context,
	actor *domain.User,
	id uuid.UUID,
) (*domain.MenuCategoryResponse, error) {
	log := s.logger.WithRequestID(ctxutil.GetRequestID(ctx))

	// Permission
	if err := s.rbacService.CheckPermission(actor, domain.PermMenuRead); err != nil {
		log.Warn("get menu category denied", "actor_id", actor.ID, "error", err)
		return nil, err
	}

	// Load
	category, err := s.categoryRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	return category.ToResponse(), nil
}

func (s *menuCategoryService) List(
	ctx context.Context,
	actor *domain.User,
	filter domain.MenuCategoryFilter,
) ([]*domain.MenuCategoryResponse, int, error) {
	log := s.logger.WithRequestID(ctxutil.GetRequestID(ctx))

	// Permission
	if err := s.rbacService.CheckPermission(actor, domain.PermMenuRead); err != nil {
		log.Warn("list menu categories denied", "actor_id", actor.ID, "error", err)
		return nil, 0, err
	}

	// Load
	categories, total, err := s.categoryRepo.List(ctx, filter)
	if err != nil {
		log.Error("list menu categories failed", "error", err)
		return nil, 0, err
	}

	// Convert
	out := make([]*domain.MenuCategoryResponse, 0, len(categories))
	for _, c := range categories {
		out = append(out, c.ToResponse())
	}

	return out, total, nil
}

// ═══════════════════════════════════════════════════════════════════
// UPDATE
// ═══════════════════════════════════════════════════════════════════

func (s *menuCategoryService) Update(
	ctx context.Context,
	actor *domain.User,
	id uuid.UUID,
	req *domain.UpdateMenuCategoryRequest,
) (*domain.MenuCategoryResponse, error) {
	log := s.logger.WithRequestID(ctxutil.GetRequestID(ctx))

	// 1. Permission
	if err := s.rbacService.CheckPermission(actor, domain.PermMenuWrite); err != nil {
		log.Warn("update menu category denied", "actor_id", actor.ID, "error", err)
		return nil, err
	}

	// 2. Load
	category, err := s.categoryRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	// 3. Apply updates
	category.Name = req.Name
	category.Description = req.Description
	category.ImageURL = req.ImageURL
	category.SortOrder = req.SortOrder
	if req.IsActive != nil {
		category.IsActive = *req.IsActive
	}

	// 4. Persist
	if err := s.categoryRepo.Update(ctx, category); err != nil {
		log.Error("update menu category failed", "category_id", id, "error", err)
		return nil, err
	}

	log.Info("menu category updated", "category_id", id, "actor_id", actor.ID)

	return category.ToResponse(), nil
}

// ═══════════════════════════════════════════════════════════════════
// DELETE
// ═══════════════════════════════════════════════════════════════════

func (s *menuCategoryService) Delete(
	ctx context.Context,
	actor *domain.User,
	id uuid.UUID,
) error {
	log := s.logger.WithRequestID(ctxutil.GetRequestID(ctx))

	// 1. Permission
	if err := s.rbacService.CheckPermission(actor, domain.PermMenuDelete); err != nil {
		log.Warn("delete menu category denied", "actor_id", actor.ID, "error", err)
		return err
	}

	// 2. Delete
	if err := s.categoryRepo.Delete(ctx, id); err != nil {
		log.Error("delete menu category failed", "category_id", id, "error", err)
		return err
	}

	log.Info("menu category deleted", "category_id", id, "actor_id", actor.ID)

	return nil
}

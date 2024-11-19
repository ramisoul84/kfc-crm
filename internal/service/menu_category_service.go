package service

import (
	"context"

	"github.com/google/uuid"

	"github.com/ramisoul84/kfc-crm/internal/domain"
	"github.com/ramisoul84/kfc-crm/internal/events"
	"github.com/ramisoul84/kfc-crm/internal/repository"
	"github.com/ramisoul84/kfc-crm/pkg/ctxutil"
	"github.com/ramisoul84/kfc-crm/pkg/logger"
	"github.com/ramisoul84/kfc-crm/pkg/storage"
)

// MenuCategoryService handles menu category business logic.
type MenuCategoryService interface {
	Create(ctx context.Context, actor *domain.User, req *domain.CreateMenuCategoryRequest) (*domain.MenuCategoryResponse, error)
	GetByID(ctx context.Context, actor *domain.User, id uuid.UUID) (*domain.MenuCategoryResponse, error)
	List(ctx context.Context, actor *domain.User, filter domain.MenuCategoryFilter) ([]*domain.MenuCategoryResponse, int, error)
	Update(ctx context.Context, actor *domain.User, id uuid.UUID, req *domain.UpdateMenuCategoryRequest) (*domain.MenuCategoryResponse, error)
	Delete(ctx context.Context, actor *domain.User, id uuid.UUID) error
	// SetPhoto uploads an image for the category and stores its URL.
	SetPhoto(ctx context.Context, actor *domain.User, id uuid.UUID, data []byte, mimeType string) (*domain.MenuCategoryResponse, error)

	// RemovePhoto deletes the category's image from storage and clears its URL.
	RemovePhoto(ctx context.Context, actor *domain.User, id uuid.UUID) (*domain.MenuCategoryResponse, error)
}

type menuCategoryService struct {
	categoryRepo repository.MenuCategoryRepository
	storage      storage.PhotoStorage
	rbacService  RBACService
	publisher    *events.Publisher
	logger       *logger.Logger
}

// NewMenuCategoryService creates a MenuCategoryService.
func NewMenuCategoryService(
	categoryRepo repository.MenuCategoryRepository,
	storage storage.PhotoStorage,
	rbacService RBACService,
	publisher *events.Publisher,
	log *logger.Logger,
) MenuCategoryService {
	return &menuCategoryService{
		categoryRepo: categoryRepo,
		storage:      storage,
		rbacService:  rbacService,
		publisher:    publisher,
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
	if err := s.rbacService.CheckPermission(actor, domain.PermMenuCategoryWrite); err != nil {
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

	s.publishMenuChanged(ctx, "category.created")

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
	if err := s.rbacService.CheckPermission(actor, domain.PermMenuCategoryRead); err != nil {
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
	if err := s.rbacService.CheckPermission(actor, domain.PermMenuCategoryRead); err != nil {
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
	if err := s.rbacService.CheckPermission(actor, domain.PermMenuCategoryWrite); err != nil {
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
	category.SortOrder = req.SortOrder
	if req.IsActive != nil {
		category.IsActive = *req.IsActive
	}

	// 4. Persist
	if err := s.categoryRepo.Update(ctx, category); err != nil {
		log.Error("update menu category failed", "category_id", id, "error", err)
		return nil, err
	}

	s.publishMenuChanged(ctx, "category.updated")

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
	if err := s.rbacService.CheckPermission(actor, domain.PermMenuCategoryDelete); err != nil {
		log.Warn("delete menu category denied", "actor_id", actor.ID, "error", err)
		return err
	}

	// 2. Delete
	if err := s.categoryRepo.Delete(ctx, id); err != nil {
		log.Error("delete menu category failed", "category_id", id, "error", err)
		return err
	}

	s.publishMenuChanged(ctx, "category.deleted")

	log.Info("menu category deleted", "category_id", id, "actor_id", actor.ID)

	return nil
}

// ═══════════════════════════════════════════════════════════════════
// PHOTO
// ═══════════════════════════════════════════════════════════════════

// SetPhoto uploads an image, stores it under the category's owner prefix
// in MinIO, and updates the category's image_url + image_key.
//
// Upload happens first: if MinIO is unavailable, the DB is untouched.
// If the DB update fails after a successful upload, the newly uploaded
// object is removed so we don't leave an orphan.
func (s *menuCategoryService) SetPhoto(
	ctx context.Context,
	actor *domain.User,
	id uuid.UUID,
	data []byte,
	mimeType string,
) (*domain.MenuCategoryResponse, error) {
	log := s.logger.WithRequestID(ctxutil.GetRequestID(ctx))

	// 1. Authorization — same permission as any other category write.
	if err := s.rbacService.CheckPermission(actor, domain.PermMenuCategoryWrite); err != nil {
		log.Warn("set category photo denied", "actor_id", actor.ID, "error", err)
		return nil, err
	}

	// 2. Load the category — needed for the old image key cleanup.
	category, err := s.categoryRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	// 3. Upload to storage.
	url, key, err := s.storage.Upload(ctx, id.String(), data, mimeType)
	if err != nil {
		log.Error("category photo upload failed",
			"category_id", id,
			"error", err,
		)
		return nil, domain.NewInternalError(err)
	}

	// 4. Persist the new URL and key before touching the old object, so
	//    a failed update never leaves the category pointing at nothing.
	if err := s.categoryRepo.SetImage(ctx, id, url, key); err != nil {
		// Roll back the upload to avoid an orphaned object.
		if delErr := s.storage.Delete(ctx, key); delErr != nil {
			log.Warn("failed to roll back uploaded photo",
				"category_id", id,
				"key", key,
				"error", delErr,
			)
		}
		log.Warn("failed to set category photo",
			"category_id", id,
			"key", key,
			"error", err,
		)
		return nil, err
	}

	// 5. The previous photo is now unreferenced. Best-effort delete — a
	//    failure leaves an orphaned object, not a broken menu.
	if category.ImageKey != "" && category.ImageKey != key {
		if err := s.storage.Delete(ctx, category.ImageKey); err != nil {
			log.Warn("failed to delete old category photo",
				"category_id", id,
				"key", category.ImageKey,
				"error", err,
			)
		}
	}

	// 6. Update the in-memory entity so ToResponse returns the new URL.
	category.ImageURL = url
	category.ImageKey = key

	log.Info("menu category photo updated",
		"category_id", id,
		"actor_id", actor.ID,
	)

	// 7. Invalidate the effective menu cache — the image is part of the
	//    menu payload that kiosks render.
	s.publishMenuChanged(ctx, "category.photo")

	return category.ToResponse(), nil
}

// RemovePhoto deletes the category's image from storage and clears the
// image_url + image_key columns. Idempotent: calling it when there is
// no photo is a no-op that returns the current state.
func (s *menuCategoryService) RemovePhoto(
	ctx context.Context,
	actor *domain.User,
	id uuid.UUID,
) (*domain.MenuCategoryResponse, error) {
	log := s.logger.WithRequestID(ctxutil.GetRequestID(ctx))

	if err := s.rbacService.CheckPermission(actor, domain.PermMenuCategoryWrite); err != nil {
		log.Warn("remove category photo denied", "actor_id", actor.ID, "error", err)
		return nil, err
	}

	category, err := s.categoryRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	// Clear the reference first, then delete the object (best-effort).
	if err := s.categoryRepo.ClearImage(ctx, id); err != nil {
		return nil, err
	}
	if category.ImageKey != "" {
		if err := s.storage.Delete(ctx, category.ImageKey); err != nil {
			log.Warn("failed to delete category photo",
				"category_id", id,
				"key", category.ImageKey,
				"error", err,
			)
		}
	}

	category.ImageURL = ""
	category.ImageKey = ""

	log.Info("menu category photo removed",
		"category_id", id,
		"actor_id", actor.ID,
	)

	// Invalidate the cache so kiosks drop the image.
	s.publishMenuChanged(ctx, "category.photo")

	return category.ToResponse(), nil
}

// ═══════════════════════════════════════════════════════════════════
// HELPERS
// ═══════════════════════════════════════════════════════════════════

// publishMenuChanged notifies userapi that the master menu changed.
// Categories are global, so this affects every restaurant.
func (s *menuCategoryService) publishMenuChanged(ctx context.Context, reason string) {
	if s.publisher == nil {
		return
	}
	s.publisher.PublishMenuUpdated(ctx, "", "", reason)
}

package service

import (
	"context"

	"github.com/google/uuid"

	"github.com/ramisoul84/kfc-crm/internal/domain"
	"github.com/ramisoul84/kfc-crm/internal/repository"
	"github.com/ramisoul84/kfc-crm/pkg/ctxutil"
	"github.com/ramisoul84/kfc-crm/pkg/logger"
)

// RegionService handles region business logic.
type RegionService interface {
	Create(ctx context.Context, actor *domain.User, req *domain.CreateRegionRequest) (*domain.RegionResponse, error)
	GetByID(ctx context.Context, actor *domain.User, id uuid.UUID) (*domain.RegionResponse, error)
	List(ctx context.Context, actor *domain.User, filter domain.RegionFilter) ([]*domain.RegionResponse, int, error)
	Update(ctx context.Context, actor *domain.User, id uuid.UUID, req *domain.UpdateRegionRequest) (*domain.RegionResponse, error)
	Delete(ctx context.Context, actor *domain.User, id uuid.UUID) error
}

type regionService struct {
	regionRepo  repository.RegionRepository
	rbacService RBACService
	logger      *logger.Logger
}

// NewRegionService creates a RegionService.
func NewRegionService(
	regionRepo repository.RegionRepository,
	rbacService RBACService,
	log *logger.Logger,
) RegionService {
	return &regionService{
		regionRepo:  regionRepo,
		rbacService: rbacService,
		logger:      log,
	}
}

// ═══════════════════════════════════════════════════════════════════
// CREATE
// ═══════════════════════════════════════════════════════════════════

func (s *regionService) Create(
	ctx context.Context,
	actor *domain.User,
	req *domain.CreateRegionRequest,
) (*domain.RegionResponse, error) {
	log := s.logger.WithRequestID(ctxutil.GetRequestID(ctx))

	log.Debug("creating region",
		"actor_id", actor.ID,
		"name", req.Name,
		"code", req.Code,
	)

	// 1. Authorization
	if err := s.rbacService.CanCreateRegion(actor); err != nil {
		log.Warn("create region denied",
			"actor_id", actor.ID,
			"actor_role", actor.Role,
			"error", err,
		)
		return nil, err
	}

	// 2. Build entity
	region := &domain.Region{
		ID:       uuid.New(),
		Name:     req.Name,
		Code:     req.Code,
		Timezone: req.Timezone,
		IsActive: true,
	}

	// 3. Persist
	if err := s.regionRepo.Create(ctx, region); err != nil {
		log.Error("create region failed",
			"region_id", region.ID,
			"code", region.Code,
			"error", err,
		)
		return nil, err
	}

	log.Info("region created",
		"region_id", region.ID,
		"code", region.Code,
		"actor_id", actor.ID,
	)

	return region.ToResponse(), nil
}

// ═══════════════════════════════════════════════════════════════════
// READ
// ═══════════════════════════════════════════════════════════════════

func (s *regionService) GetByID(
	ctx context.Context,
	actor *domain.User,
	id uuid.UUID,
) (*domain.RegionResponse, error) {
	log := s.logger.WithRequestID(ctxutil.GetRequestID(ctx))

	log.Debug("getting region",
		"actor_id", actor.ID,
		"region_id", id,
	)

	// 1. Permission
	if err := s.rbacService.CheckPermission(actor, domain.PermRegionRead); err != nil {
		log.Warn("get region denied",
			"actor_id", actor.ID,
			"region_id", id,
			"error", err,
		)
		return nil, err
	}

	// 2. Scope (non-super-admin can only see their own region)
	if err := s.rbacService.CanAccessRegion(actor, id); err != nil {
		log.Warn("get region denied: scope",
			"actor_id", actor.ID,
			"region_id", id,
			"error", err,
		)
		return nil, err
	}

	// 3. Load
	region, err := s.regionRepo.GetByID(ctx, id)
	if err != nil {
		log.Error("get region failed", "region_id", id, "error", err)
		return nil, err
	}

	return region.ToResponse(), nil
}

func (s *regionService) List(
	ctx context.Context,
	actor *domain.User,
	filter domain.RegionFilter,
) ([]*domain.RegionResponse, int, error) {
	log := s.logger.WithRequestID(ctxutil.GetRequestID(ctx))

	log.Debug("listing regions", "actor_id", actor.ID, "role", actor.Role)

	// 1. Permission
	if err := s.rbacService.CheckPermission(actor, domain.PermRegionRead); err != nil {
		log.Warn("list regions denied",
			"actor_id", actor.ID,
			"error", err,
		)
		return nil, 0, err
	}

	// 2. Load regions
	regions, total, err := s.regionRepo.List(ctx, filter)
	if err != nil {
		log.Error("list regions failed", "error", err)
		return nil, 0, err
	}

	// 3. Filter by scope for non-super-admin
	if actor.Role != domain.RoleSuperAdmin {
		filtered := make([]*domain.Region, 0, len(regions))
		for _, r := range regions {
			if actor.RegionID != nil && r.ID == *actor.RegionID {
				filtered = append(filtered, r)
			}
		}
		regions = filtered
		total = len(filtered)
	}

	// 4. Convert to responses
	out := make([]*domain.RegionResponse, 0, len(regions))
	for _, r := range regions {
		out = append(out, r.ToResponse())
	}

	log.Debug("regions listed", "count", len(out))

	return out, total, nil
}

// ═══════════════════════════════════════════════════════════════════
// UPDATE
// ═══════════════════════════════════════════════════════════════════

func (s *regionService) Update(
	ctx context.Context,
	actor *domain.User,
	id uuid.UUID,
	req *domain.UpdateRegionRequest,
) (*domain.RegionResponse, error) {
	log := s.logger.WithRequestID(ctxutil.GetRequestID(ctx))

	log.Debug("updating region",
		"actor_id", actor.ID,
		"region_id", id,
	)

	// 1. Authorization
	if err := s.rbacService.CanUpdateRegion(actor); err != nil {
		log.Warn("update region denied",
			"actor_id", actor.ID,
			"region_id", id,
			"error", err,
		)
		return nil, err
	}

	// 2. Load current
	region, err := s.regionRepo.GetByID(ctx, id)
	if err != nil {
		log.Error("get region failed", "region_id", id, "error", err)
		return nil, err
	}

	// 3. Apply updates
	region.Name = req.Name
	region.Code = req.Code
	region.Timezone = req.Timezone
	if req.IsActive != nil {
		region.IsActive = *req.IsActive
	}

	// 4. Persist
	if err := s.regionRepo.Update(ctx, region); err != nil {
		log.Error("update region failed",
			"region_id", id,
			"error", err,
		)
		return nil, err
	}

	log.Info("region updated",
		"region_id", id,
		"code", region.Code,
		"actor_id", actor.ID,
	)

	return region.ToResponse(), nil
}

// ═══════════════════════════════════════════════════════════════════
// DELETE
// ═══════════════════════════════════════════════════════════════════

func (s *regionService) Delete(
	ctx context.Context,
	actor *domain.User,
	id uuid.UUID,
) error {
	log := s.logger.WithRequestID(ctxutil.GetRequestID(ctx))

	log.Debug("deleting region",
		"actor_id", actor.ID,
		"region_id", id,
	)

	// 1. Authorization
	if err := s.rbacService.CanDeleteRegion(actor); err != nil {
		log.Warn("delete region denied",
			"actor_id", actor.ID,
			"region_id", id,
			"error", err,
		)
		return err
	}

	// 2. Delete
	if err := s.regionRepo.Delete(ctx, id); err != nil {
		log.Error("delete region failed", "region_id", id, "error", err)
		return err
	}

	log.Info("region deleted",
		"region_id", id,
		"actor_id", actor.ID,
	)

	return nil
}

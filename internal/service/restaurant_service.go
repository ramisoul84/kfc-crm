package service

import (
	"context"

	"github.com/google/uuid"

	"github.com/ramisoul84/kfc-crm/internal/domain"
	"github.com/ramisoul84/kfc-crm/internal/repository"
	"github.com/ramisoul84/kfc-crm/pkg/ctxutil"
	"github.com/ramisoul84/kfc-crm/pkg/logger"
)

// RestaurantService handles restaurant business logic.
type RestaurantService interface {
	Create(ctx context.Context, actor *domain.User, req *domain.CreateRestaurantRequest) (*domain.RestaurantResponse, error)
	GetByID(ctx context.Context, actor *domain.User, id uuid.UUID) (*domain.RestaurantResponse, error)
	List(ctx context.Context, actor *domain.User, filter domain.RestaurantFilter) ([]*domain.RestaurantResponse, int, error)
	Update(ctx context.Context, actor *domain.User, id uuid.UUID, req *domain.UpdateRestaurantRequest) (*domain.RestaurantResponse, error)
	Delete(ctx context.Context, actor *domain.User, id uuid.UUID) error
}

type restaurantService struct {
	restaurantRepo repository.RestaurantRepository
	rbacService    RBACService
	logger         *logger.Logger
}

// NewRestaurantService creates a RestaurantService.
func NewRestaurantService(
	restaurantRepo repository.RestaurantRepository,
	rbacService RBACService,
	log *logger.Logger,
) RestaurantService {
	return &restaurantService{
		restaurantRepo: restaurantRepo,
		rbacService:    rbacService,
		logger:         log,
	}
}

// ═══════════════════════════════════════════════════════════════════
// CREATE
// ═══════════════════════════════════════════════════════════════════

func (s *restaurantService) Create(
	ctx context.Context,
	actor *domain.User,
	req *domain.CreateRestaurantRequest,
) (*domain.RestaurantResponse, error) {
	log := s.logger.WithRequestID(ctxutil.GetRequestID(ctx))

	log.Debug("creating restaurant",
		"actor_id", actor.ID,
		"name", req.Name,
		"code", req.Code,
		"region_id", req.RegionID,
	)

	// 1. Authorization — permission + region existence + scope
	if err := s.rbacService.CanCreateRestaurant(actor, req.RegionID); err != nil {
		log.Warn("create restaurant denied",
			"actor_id", actor.ID,
			"actor_role", actor.Role,
			"region_id", req.RegionID,
			"error", err,
		)
		return nil, err
	}

	// 2. Explicit existence check for a cleaner error message
	if err := s.rbacService.RegionExists(ctx, req.RegionID); err != nil {
		log.Warn("create restaurant: region not found",
			"region_id", req.RegionID,
			"error", err,
		)
		return nil, err
	}

	// 3. Build entity
	restaurant := &domain.Restaurant{
		ID:         uuid.New(),
		Name:       req.Name,
		Code:       req.Code,
		RegionID:   req.RegionID,
		Address:    req.Address,
		City:       req.City,
		PostalCode: req.PostalCode,
		Phone:      req.Phone,
		IsActive:   true,
	}

	// 4. Persist
	if err := s.restaurantRepo.Create(ctx, restaurant); err != nil {
		log.Error("create restaurant failed",
			"restaurant_id", restaurant.ID,
			"code", restaurant.Code,
			"error", err,
		)
		return nil, err
	}

	log.Info("restaurant created",
		"restaurant_id", restaurant.ID,
		"code", restaurant.Code,
		"region_id", restaurant.RegionID,
		"actor_id", actor.ID,
	)

	return restaurant.ToResponse(), nil
}

// ═══════════════════════════════════════════════════════════════════
// READ
// ═══════════════════════════════════════════════════════════════════

func (s *restaurantService) GetByID(
	ctx context.Context,
	actor *domain.User,
	id uuid.UUID,
) (*domain.RestaurantResponse, error) {
	log := s.logger.WithRequestID(ctxutil.GetRequestID(ctx))

	log.Debug("getting restaurant",
		"actor_id", actor.ID,
		"restaurant_id", id,
	)

	// 1. Permission + scope
	if err := s.rbacService.CanReadRestaurant(ctx, actor, id); err != nil {
		log.Warn("get restaurant denied",
			"actor_id", actor.ID,
			"restaurant_id", id,
			"error", err,
		)
		return nil, err
	}

	// 2. Load with region for the response
	restaurant, err := s.restaurantRepo.GetByIDWithRegion(ctx, id)
	if err != nil {
		log.Error("get restaurant failed", "restaurant_id", id, "error", err)
		return nil, err
	}

	return restaurant.ToResponse(), nil
}

func (s *restaurantService) List(
	ctx context.Context,
	actor *domain.User,
	filter domain.RestaurantFilter,
) ([]*domain.RestaurantResponse, int, error) {
	log := s.logger.WithRequestID(ctxutil.GetRequestID(ctx))

	log.Debug("listing restaurants",
		"actor_id", actor.ID,
		"role", actor.Role,
	)

	// 1. Permission
	if err := s.rbacService.CheckPermission(actor, domain.PermRestaurantRead); err != nil {
		log.Warn("list restaurants denied",
			"actor_id", actor.ID,
			"error", err,
		)
		return nil, 0, err
	}

	// 2. Apply scope filter
	switch actor.Role {
	case domain.RoleSuperAdmin:
		// no filter
	case domain.RoleRegionalManager:
		if actor.RegionID != nil {
			filter.RegionID = actor.RegionID
		}
	case domain.RoleRestaurantManager, domain.RoleShiftManager, domain.RoleCashier:
		if actor.RestaurantID != nil {
			// These roles see only one restaurant — return early
			restaurant, err := s.restaurantRepo.GetByIDWithRegion(ctx, *actor.RestaurantID)
			if err != nil {
				log.Error("list restaurants failed", "error", err)
				return nil, 0, err
			}
			return []*domain.RestaurantResponse{restaurant.ToResponse()}, 1, nil
		}
	}

	// 3. Load
	restaurants, total, err := s.restaurantRepo.List(ctx, filter)
	if err != nil {
		log.Error("list restaurants failed", "error", err)
		return nil, 0, err
	}

	// 4. Convert
	out := make([]*domain.RestaurantResponse, 0, len(restaurants))
	for _, r := range restaurants {
		out = append(out, r.ToResponse())
	}

	log.Debug("restaurants listed", "count", len(out))

	return out, total, nil
}

// ═══════════════════════════════════════════════════════════════════
// UPDATE
// ═══════════════════════════════════════════════════════════════════

func (s *restaurantService) Update(
	ctx context.Context,
	actor *domain.User,
	id uuid.UUID,
	req *domain.UpdateRestaurantRequest,
) (*domain.RestaurantResponse, error) {
	log := s.logger.WithRequestID(ctxutil.GetRequestID(ctx))

	log.Debug("updating restaurant",
		"actor_id", actor.ID,
		"restaurant_id", id,
	)

	// 1. Load current
	restaurant, err := s.restaurantRepo.GetByID(ctx, id)
	if err != nil {
		log.Error("get restaurant failed", "restaurant_id", id, "error", err)
		return nil, err
	}

	// 2. Authorization
	if err := s.rbacService.CanUpdateRestaurant(ctx, actor, restaurant); err != nil {
		log.Warn("update restaurant denied",
			"actor_id", actor.ID,
			"restaurant_id", id,
			"error", err,
		)
		return nil, err
	}

	// 3. Apply updates
	restaurant.Name = req.Name
	restaurant.Code = req.Code
	restaurant.Address = req.Address
	restaurant.City = req.City
	restaurant.PostalCode = req.PostalCode
	restaurant.Phone = req.Phone
	if req.IsActive != nil {
		restaurant.IsActive = *req.IsActive
	}

	// 4. Persist
	if err := s.restaurantRepo.Update(ctx, restaurant); err != nil {
		log.Error("update restaurant failed",
			"restaurant_id", id,
			"error", err,
		)
		return nil, err
	}

	// 5. Reload with region for the response
	restaurant, err = s.restaurantRepo.GetByIDWithRegion(ctx, id)
	if err != nil {
		return nil, err
	}

	log.Info("restaurant updated",
		"restaurant_id", id,
		"code", restaurant.Code,
		"actor_id", actor.ID,
	)

	return restaurant.ToResponse(), nil
}

// ═══════════════════════════════════════════════════════════════════
// DELETE
// ═══════════════════════════════════════════════════════════════════

func (s *restaurantService) Delete(
	ctx context.Context,
	actor *domain.User,
	id uuid.UUID,
) error {
	log := s.logger.WithRequestID(ctxutil.GetRequestID(ctx))

	log.Debug("deleting restaurant",
		"actor_id", actor.ID,
		"restaurant_id", id,
	)

	// 1. Load current
	restaurant, err := s.restaurantRepo.GetByID(ctx, id)
	if err != nil {
		log.Error("get restaurant failed", "restaurant_id", id, "error", err)
		return err
	}

	// 2. Authorization
	if err := s.rbacService.CanDeleteRestaurant(ctx, actor, restaurant); err != nil {
		log.Warn("delete restaurant denied",
			"actor_id", actor.ID,
			"restaurant_id", id,
			"error", err,
		)
		return err
	}

	// 3. Delete
	if err := s.restaurantRepo.Delete(ctx, id); err != nil {
		log.Error("delete restaurant failed",
			"restaurant_id", id,
			"error", err,
		)
		return err
	}

	log.Info("restaurant deleted",
		"restaurant_id", id,
		"actor_id", actor.ID,
	)

	return nil
}

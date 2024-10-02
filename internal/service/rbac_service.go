package service

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"github.com/ramisoul84/kfc-crm/internal/domain"
	"github.com/ramisoul84/kfc-crm/internal/repository"
)

// RBACService handles all authorization checks.
// It has no side effects and no logging — callers log denials with context.
type RBACService interface {
	// ═══════════════════════════════════════════════════════════════
	// Permission
	// ═══════════════════════════════════════════════════════════════
	CheckPermission(user *domain.User, permission domain.Permission) error

	// ═══════════════════════════════════════════════════════════════
	// Existence
	// ═══════════════════════════════════════════════════════════════
	RegionExists(ctx context.Context, regionID uuid.UUID) error
	RestaurantExists(ctx context.Context, restaurantID uuid.UUID) error
	DeviceExists(ctx context.Context, deviceID uuid.UUID) error

	// ═══════════════════════════════════════════════════════════════
	// Scope
	// ═══════════════════════════════════════════════════════════════
	CanAccessRegion(user *domain.User, regionID uuid.UUID) error
	CanAccessRestaurant(ctx context.Context, user *domain.User, restaurantID uuid.UUID) error

	// ═══════════════════════════════════════════════════════════════
	// User
	// ═══════════════════════════════════════════════════════════════
	CanCreateUser(ctx context.Context, actor *domain.User, req *domain.CreateUserRequest) error
	CanReadUser(actor, target *domain.User) error
	CanUpdateUser(actor, target *domain.User, req *domain.UpdateUserRequest) error
	CanDeleteUser(actor, target *domain.User) error

	// ═══════════════════════════════════════════════════════════════
	// Restaurant
	// ═══════════════════════════════════════════════════════════════
	CanCreateRestaurant(actor *domain.User, regionID uuid.UUID) error
	CanReadRestaurant(ctx context.Context, actor *domain.User, restaurantID uuid.UUID) error
	CanUpdateRestaurant(ctx context.Context, actor *domain.User, restaurant *domain.Restaurant) error
	CanDeleteRestaurant(ctx context.Context, actor *domain.User, restaurant *domain.Restaurant) error

	// ═══════════════════════════════════════════════════════════════
	// Device
	// ═══════════════════════════════════════════════════════════════
	CanCreateDevice(ctx context.Context, actor *domain.User, restaurantID uuid.UUID) error
	CanReadDevice(ctx context.Context, actor *domain.User, device *domain.Device) error
	CanUpdateDevice(ctx context.Context, actor *domain.User, device *domain.Device) error
	CanDeleteDevice(ctx context.Context, actor *domain.User, device *domain.Device) error

	// ═══════════════════════════════════════════════════════════════
	// Region
	// ═══════════════════════════════════════════════════════════════
	CanCreateRegion(actor *domain.User) error
	CanUpdateRegion(actor *domain.User) error
	CanDeleteRegion(actor *domain.User) error
}

type rbacService struct {
	regionRepo     repository.RegionRepository
	restaurantRepo repository.RestaurantRepository
	deviceRepo     repository.DeviceRepository
}

func NewRBACService(
	regionRepo repository.RegionRepository,
	restaurantRepo repository.RestaurantRepository,
	deviceRepo repository.DeviceRepository,
) RBACService {
	return &rbacService{
		regionRepo:     regionRepo,
		restaurantRepo: restaurantRepo,
		deviceRepo:     deviceRepo,
	}
}

// ═══════════════════════════════════════════════════════════════════
// PERMISSION
// ═══════════════════════════════════════════════════════════════════

func (s *rbacService) CheckPermission(user *domain.User, permission domain.Permission) error {
	if user == nil {
		return domain.NewAuthorizationError("user not found")
	}
	if !user.Role.IsValid() {
		return domain.NewAuthorizationError("user has invalid role")
	}
	if !user.IsActive {
		return domain.NewAuthorizationError("user account is inactive")
	}
	if !user.Role.HasPermission(permission) {
		return domain.NewAuthorizationError(
			fmt.Sprintf("missing permission: %s", permission),
		)
	}
	return nil
}

// ═══════════════════════════════════════════════════════════════════
// EXISTENCE
// ═══════════════════════════════════════════════════════════════════

func (s *rbacService) RegionExists(ctx context.Context, regionID uuid.UUID) error {
	if _, err := s.regionRepo.GetByID(ctx, regionID); err != nil {
		if domain.IsNotFoundError(err) {
			return domain.NewValidationError("region does not exist")
		}
		return err
	}
	return nil
}

func (s *rbacService) RestaurantExists(ctx context.Context, restaurantID uuid.UUID) error {
	if _, err := s.restaurantRepo.GetByID(ctx, restaurantID); err != nil {
		if domain.IsNotFoundError(err) {
			return domain.NewValidationError("restaurant does not exist")
		}
		return err
	}
	return nil
}

func (s *rbacService) DeviceExists(ctx context.Context, deviceID uuid.UUID) error {
	if _, err := s.deviceRepo.GetByID(ctx, deviceID); err != nil {
		if domain.IsNotFoundError(err) {
			return domain.NewValidationError("device does not exist")
		}
		return err
	}
	return nil
}

// ═══════════════════════════════════════════════════════════════════
// SCOPE
// ═══════════════════════════════════════════════════════════════════

func (s *rbacService) CanAccessRegion(user *domain.User, regionID uuid.UUID) error {
	if user == nil {
		return domain.NewAuthorizationError("user not found")
	}

	switch user.Role {
	case domain.RoleSuperAdmin:
		return nil

	case domain.RoleRegionalManager:
		if user.RegionID == nil {
			return domain.NewAuthorizationError("no region assigned")
		}
		if *user.RegionID != regionID {
			return domain.NewAuthorizationError("cannot access this region")
		}
		return nil

	default:
		return domain.NewAuthorizationError("insufficient permissions")
	}
}

func (s *rbacService) CanAccessRestaurant(
	ctx context.Context,
	user *domain.User,
	restaurantID uuid.UUID,
) error {
	if user == nil {
		return domain.NewAuthorizationError("user not found")
	}

	switch user.Role {
	case domain.RoleSuperAdmin:
		return nil

	case domain.RoleRegionalManager:
		if user.RegionID == nil {
			return domain.NewAuthorizationError("no region assigned")
		}
		restaurant, err := s.restaurantRepo.GetByID(ctx, restaurantID)
		if err != nil {
			return err
		}
		if restaurant.RegionID != *user.RegionID {
			return domain.NewAuthorizationError("restaurant is outside your region")
		}
		return nil

	case domain.RoleRestaurantManager, domain.RoleShiftManager, domain.RoleCashier:
		if user.RestaurantID == nil {
			return domain.NewAuthorizationError("no restaurant assigned")
		}
		if *user.RestaurantID != restaurantID {
			return domain.NewAuthorizationError("cannot access this restaurant")
		}
		return nil

	default:
		return domain.NewAuthorizationError("insufficient permissions")
	}
}

// ═══════════════════════════════════════════════════════════════════
// USER
// ═══════════════════════════════════════════════════════════════════

func (s *rbacService) CanCreateUser(
	ctx context.Context,
	actor *domain.User,
	req *domain.CreateUserRequest,
) error {
	// 1. Permission
	if err := s.CheckPermission(actor, domain.PermUserCreate); err != nil {
		return err
	}

	// 2. Role hierarchy
	if !actor.Role.CanCreateRole(req.Role) {
		return domain.NewAuthorizationError(
			fmt.Sprintf("%s cannot create %s", actor.Role, req.Role),
		)
	}

	// 3. Existence — load referenced entities once
	var (
		targetRegion     *domain.Region
		targetRestaurant *domain.Restaurant
	)

	if req.RegionID != nil {
		r, err := s.regionRepo.GetByID(ctx, *req.RegionID)
		if err != nil {
			if domain.IsNotFoundError(err) {
				return domain.NewValidationError("region does not exist")
			}
			return err
		}
		targetRegion = r
	}

	if req.RestaurantID != nil {
		r, err := s.restaurantRepo.GetByID(ctx, *req.RestaurantID)
		if err != nil {
			if domain.IsNotFoundError(err) {
				return domain.NewValidationError("restaurant does not exist")
			}
			return err
		}
		targetRestaurant = r
	}

	// 4. Consistency — if both provided, they must match
	if targetRegion != nil && targetRestaurant != nil {
		if targetRestaurant.RegionID != targetRegion.ID {
			return domain.NewValidationError(
				"restaurant does not belong to the specified region",
			)
		}
	}

	// 5. Scope
	switch actor.Role {
	case domain.RoleSuperAdmin:
		return nil

	case domain.RoleRegionalManager:
		if actor.RegionID == nil {
			return domain.NewAuthorizationError("no region assigned")
		}

		// If a restaurant is targeted, its region must equal actor's region
		if targetRestaurant != nil && targetRestaurant.RegionID != *actor.RegionID {
			return domain.NewAuthorizationError("cannot create users outside your region")
		}

		// If a region is targeted, it must equal actor's region
		if targetRegion != nil && targetRegion.ID != *actor.RegionID {
			return domain.NewAuthorizationError("cannot create users outside your region")
		}

		return nil

	case domain.RoleRestaurantManager:
		if actor.RestaurantID == nil {
			return domain.NewAuthorizationError("no restaurant assigned")
		}
		if targetRestaurant == nil {
			return domain.NewValidationError("restaurant is required")
		}
		if targetRestaurant.ID != *actor.RestaurantID {
			return domain.NewAuthorizationError("cannot create users outside your restaurant")
		}
		return nil

	default:
		return domain.NewAuthorizationError("insufficient permissions")
	}
}

// CanReadUser is the base "can this actor see this target user?" check.
// Used by GetByID and List.
func (s *rbacService) CanReadUser(actor, target *domain.User) error {
	if err := s.CheckPermission(actor, domain.PermUserRead); err != nil {
		return err
	}
	return s.canManageUser(actor, target)
}

func (s *rbacService) CanUpdateUser(
	actor, target *domain.User,
	req *domain.UpdateUserRequest,
) error {
	// 1. Permission
	if err := s.CheckPermission(actor, domain.PermUserUpdate); err != nil {
		return err
	}

	// 2. Self-deactivation guard
	if req.IsActive != nil && !*req.IsActive && actor.ID == target.ID {
		return domain.NewAuthorizationError("cannot deactivate yourself")
	}

	// 3. Base rule
	return s.canManageUser(actor, target)
}

func (s *rbacService) CanDeleteUser(actor, target *domain.User) error {
	// 1. Permission
	if err := s.CheckPermission(actor, domain.PermUserDelete); err != nil {
		return err
	}

	// 2. Cannot delete self
	if actor.ID == target.ID {
		return domain.NewAuthorizationError("cannot delete yourself")
	}

	// 3. Base rule
	return s.canManageUser(actor, target)
}

// canManageUser is the internal base rule: self, hierarchy, scope.
// Not exported — callers use CanUpdateUser/CanDeleteUser/CanReadUser.
func (s *rbacService) canManageUser(actor, target *domain.User) error {
	if actor == nil || target == nil {
		return domain.NewAuthorizationError("user not found")
	}

	// 1. Cannot manage yourself
	if actor.ID == target.ID {
		return domain.NewAuthorizationError("cannot manage yourself")
	}

	// 2. Hierarchy — target must be strictly lower
	if target.Role.IsAtLeast(actor.Role) {
		return domain.NewAuthorizationError("cannot manage user with equal or higher role")
	}

	// 3. Scope
	switch actor.Role {
	case domain.RoleSuperAdmin:
		return nil

	case domain.RoleRegionalManager:
		if actor.RegionID == nil {
			return domain.NewAuthorizationError("no region assigned")
		}
		// Target must belong to the same region.
		// Prefer RegionID; fall back to the target's restaurant if set.
		if target.RegionID != nil {
			if *target.RegionID != *actor.RegionID {
				return domain.NewAuthorizationError("cannot manage users outside your region")
			}
			return nil
		}
		// Target has no region — deny
		return domain.NewAuthorizationError("cannot manage users outside your region")

	case domain.RoleRestaurantManager:
		if actor.RestaurantID == nil {
			return domain.NewAuthorizationError("no restaurant assigned")
		}
		if target.RestaurantID == nil || *target.RestaurantID != *actor.RestaurantID {
			return domain.NewAuthorizationError("cannot manage users outside your restaurant")
		}
		return nil

	default:
		return domain.NewAuthorizationError("insufficient permissions")
	}
}

// ═══════════════════════════════════════════════════════════════════
// RESTAURANT
// ═══════════════════════════════════════════════════════════════════

func (s *rbacService) CanCreateRestaurant(actor *domain.User, regionID uuid.UUID) error {
	if err := s.CheckPermission(actor, domain.PermRestaurantCreate); err != nil {
		return err
	}

	switch actor.Role {
	case domain.RoleSuperAdmin:
		return nil

	case domain.RoleRegionalManager:
		if actor.RegionID == nil {
			return domain.NewAuthorizationError("no region assigned")
		}
		if *actor.RegionID != regionID {
			return domain.NewAuthorizationError("can only create restaurants in your region")
		}
		return nil

	default:
		return domain.NewAuthorizationError("insufficient permissions")
	}
}

func (s *rbacService) CanReadRestaurant(
	ctx context.Context,
	actor *domain.User,
	restaurantID uuid.UUID,
) error {
	if err := s.CheckPermission(actor, domain.PermRestaurantRead); err != nil {
		return err
	}
	return s.CanAccessRestaurant(ctx, actor, restaurantID)
}

func (s *rbacService) CanUpdateRestaurant(
	ctx context.Context,
	actor *domain.User,
	restaurant *domain.Restaurant,
) error {
	if err := s.CheckPermission(actor, domain.PermRestaurantUpdate); err != nil {
		return err
	}
	if restaurant == nil {
		return domain.NewAuthorizationError("restaurant not found")
	}
	return s.CanAccessRestaurant(ctx, actor, restaurant.ID)
}

func (s *rbacService) CanDeleteRestaurant(
	ctx context.Context,
	actor *domain.User,
	restaurant *domain.Restaurant,
) error {
	if err := s.CheckPermission(actor, domain.PermRestaurantDelete); err != nil {
		return err
	}
	if restaurant == nil {
		return domain.NewAuthorizationError("restaurant not found")
	}
	return s.CanAccessRestaurant(ctx, actor, restaurant.ID)
}

// ═══════════════════════════════════════════════════════════════════
// DEVICE
// ═══════════════════════════════════════════════════════════════════

func (s *rbacService) CanCreateDevice(
	ctx context.Context,
	actor *domain.User,
	restaurantID uuid.UUID,
) error {
	if err := s.CheckPermission(actor, domain.PermDeviceCreate); err != nil {
		return err
	}
	// Existence check gives a cleaner error than scope check
	if err := s.RestaurantExists(ctx, restaurantID); err != nil {
		return err
	}
	return s.CanAccessRestaurant(ctx, actor, restaurantID)
}

func (s *rbacService) CanReadDevice(
	ctx context.Context,
	actor *domain.User,
	device *domain.Device,
) error {
	if err := s.CheckPermission(actor, domain.PermDeviceRead); err != nil {
		return err
	}
	if device == nil {
		return domain.NewAuthorizationError("device not found")
	}
	return s.CanAccessRestaurant(ctx, actor, device.RestaurantID)
}

func (s *rbacService) CanUpdateDevice(
	ctx context.Context,
	actor *domain.User,
	device *domain.Device,
) error {
	if err := s.CheckPermission(actor, domain.PermDeviceUpdate); err != nil {
		return err
	}
	if device == nil {
		return domain.NewAuthorizationError("device not found")
	}
	return s.CanAccessRestaurant(ctx, actor, device.RestaurantID)
}

func (s *rbacService) CanDeleteDevice(
	ctx context.Context,
	actor *domain.User,
	device *domain.Device,
) error {
	if err := s.CheckPermission(actor, domain.PermDeviceDelete); err != nil {
		return err
	}
	if device == nil {
		return domain.NewAuthorizationError("device not found")
	}
	return s.CanAccessRestaurant(ctx, actor, device.RestaurantID)
}

// ═══════════════════════════════════════════════════════════════════
// REGION
// ═══════════════════════════════════════════════════════════════════

func (s *rbacService) CanCreateRegion(actor *domain.User) error {
	if err := s.CheckPermission(actor, domain.PermRegionCreate); err != nil {
		return err
	}
	if actor.Role != domain.RoleSuperAdmin {
		return domain.NewAuthorizationError("only super admin can create regions")
	}
	return nil
}

func (s *rbacService) CanUpdateRegion(actor *domain.User) error {
	if err := s.CheckPermission(actor, domain.PermRegionUpdate); err != nil {
		return err
	}
	if actor.Role != domain.RoleSuperAdmin {
		return domain.NewAuthorizationError("only super admin can update regions")
	}
	return nil
}

func (s *rbacService) CanDeleteRegion(actor *domain.User) error {
	if err := s.CheckPermission(actor, domain.PermRegionDelete); err != nil {
		return err
	}
	if actor.Role != domain.RoleSuperAdmin {
		return domain.NewAuthorizationError("only super admin can delete regions")
	}
	return nil
}

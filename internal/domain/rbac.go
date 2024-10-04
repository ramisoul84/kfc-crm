package domain

import "github.com/google/uuid"

// ═══════════════════════════════════════════════════════════════════
// ROLE
// ═══════════════════════════════════════════════════════════════════

// Role represents a user role.
type Role string

const (
	RoleSuperAdmin        Role = "super_admin"
	RoleRegionalManager   Role = "regional_manager"
	RoleRestaurantManager Role = "restaurant_manager"
	RoleShiftManager      Role = "shift_manager"
	RoleCashier           Role = "cashier"
)

// AllRoles is the exhaustive list of valid roles.
var AllRoles = []Role{
	RoleSuperAdmin,
	RoleRegionalManager,
	RoleRestaurantManager,
	RoleShiftManager,
	RoleCashier,
}

// Level returns the hierarchical level of the role.
// Higher level = more authority.
func (r Role) Level() int {
	switch r {
	case RoleSuperAdmin:
		return 5
	case RoleRegionalManager:
		return 4
	case RoleRestaurantManager:
		return 3
	case RoleShiftManager:
		return 2
	case RoleCashier:
		return 1
	default:
		return 0
	}
}

// IsAtLeast reports whether r has at least the given level.
func (r Role) IsAtLeast(other Role) bool {
	return r.Level() >= other.Level()
}

// IsValid reports whether the role is a known role.
func (r Role) IsValid() bool {
	return r.Level() > 0
}

// IsValidRole reports whether the given string is a known role.
func IsValidRole(name string) bool {
	return Role(name).IsValid()
}

// ═══════════════════════════════════════════════════════════════════
// PERMISSION
// ═══════════════════════════════════════════════════════════════════

// Permission represents a granular permission.
type Permission string

const (
	// Region
	PermRegionCreate Permission = "region.create"
	PermRegionRead   Permission = "region.read"
	PermRegionUpdate Permission = "region.update"
	PermRegionDelete Permission = "region.delete"

	// Restaurant
	PermRestaurantCreate Permission = "restaurant.create"
	PermRestaurantRead   Permission = "restaurant.read"
	PermRestaurantUpdate Permission = "restaurant.update"
	PermRestaurantDelete Permission = "restaurant.delete"

	// User
	PermUserCreate Permission = "user.create"
	PermUserRead   Permission = "user.read"
	PermUserUpdate Permission = "user.update"
	PermUserDelete Permission = "user.delete"

	// Device
	PermDeviceCreate Permission = "device.create"
	PermDeviceRead   Permission = "device.read"
	PermDeviceUpdate Permission = "device.update"
	PermDeviceDelete Permission = "device.delete"
	PermDevicePair   Permission = "device.pair"

	// Menu Category permissions
	PermMenuCategoryRead   Permission = "menu.category.read"
	PermMenuCategoryWrite  Permission = "menu.category.write"
	PermMenuCategoryDelete Permission = "menu.category.delete"

	// Menu Item permissions
	PermMenuItemRead   Permission = "menu.item.read"
	PermMenuItemWrite  Permission = "menu.item.write"
	PermMenuItemDelete Permission = "menu.item.delete"

	// Menu Item Variation permissions
	PermMenuVariationRead   Permission = "menu.variation.read"
	PermMenuVariationWrite  Permission = "menu.variation.write"
	PermMenuVariationDelete Permission = "menu.variation.delete"

	// Menu Override permissions
	PermMenuOverrideRead   Permission = "menu.override.read"
	PermMenuOverrideWrite  Permission = "menu.override.write"
	PermMenuOverrideDelete Permission = "menu.override.delete"

	// Menu Promotion permissions
	PermMenuPromotionRead   Permission = "menu.promotion.read"
	PermMenuPromotionWrite  Permission = "menu.promotion.write"
	PermMenuPromotionDelete Permission = "menu.promotion.delete"
)

// ═══════════════════════════════════════════════════════════════════
// ROLE → PERMISSIONS
// ═══════════════════════════════════════════════════════════════════

// RolePermissions maps each role to its permissions.
var RolePermissions = map[Role][]Permission{
	RoleSuperAdmin: {
		PermRegionCreate, PermRegionRead, PermRegionUpdate, PermRegionDelete,
		PermRestaurantCreate, PermRestaurantRead, PermRestaurantUpdate, PermRestaurantDelete,
		PermUserCreate, PermUserRead, PermUserUpdate, PermUserDelete,
		PermDeviceCreate, PermDeviceRead, PermDeviceUpdate, PermDeviceDelete, PermDevicePair,
		PermMenuCategoryRead, PermMenuCategoryWrite, PermMenuCategoryDelete,
		PermMenuItemRead, PermMenuItemWrite, PermMenuItemDelete,
		PermMenuVariationRead, PermMenuVariationWrite, PermMenuVariationDelete,
		PermMenuOverrideRead, PermMenuOverrideWrite, PermMenuOverrideDelete,
		PermMenuPromotionRead, PermMenuPromotionWrite, PermMenuPromotionDelete,
	},
	RoleRegionalManager: {
		PermRegionRead,
		PermRestaurantCreate, PermRestaurantRead, PermRestaurantUpdate, PermRestaurantDelete,
		PermUserCreate, PermUserRead, PermUserUpdate, PermUserDelete,
		PermDeviceCreate, PermDeviceRead, PermDeviceUpdate, PermDeviceDelete, PermDevicePair,
		PermMenuCategoryRead,
		PermMenuItemRead,
		PermMenuVariationRead,
		PermMenuOverrideRead, PermMenuOverrideWrite, PermMenuOverrideDelete,
		PermMenuPromotionRead, PermMenuPromotionWrite, PermMenuPromotionDelete,
	},
	RoleRestaurantManager: {
		PermRestaurantRead,
		PermUserCreate, PermUserRead, PermUserUpdate,
		PermDeviceCreate, PermDeviceRead, PermDeviceUpdate, PermDevicePair,
		PermMenuCategoryRead,
		PermMenuItemRead,
		PermMenuVariationRead,
		PermMenuOverrideRead, PermMenuOverrideWrite, PermMenuOverrideDelete,
		PermMenuPromotionRead,
	},
	RoleShiftManager: {
		PermRestaurantRead,
		PermUserRead,
		PermDeviceRead,
		PermMenuCategoryRead,
		PermMenuItemRead,
		PermMenuVariationRead,
		PermMenuOverrideRead,
		PermMenuPromotionRead,
	},
	RoleCashier: {
		PermRestaurantRead,
		PermMenuCategoryRead,
		PermMenuItemRead,
		PermMenuVariationRead,
		PermMenuOverrideRead,
		PermMenuPromotionRead,
	},
}

// HasPermission reports whether the role has the given permission.
func (r Role) HasPermission(perm Permission) bool {
	for _, p := range RolePermissions[r] {
		if p == perm {
			return true
		}
	}
	return false
}

// ═══════════════════════════════════════════════════════════════════
// ROLE → CREATABLE ROLES
// ═══════════════════════════════════════════════════════════════════

// CreatableRoles maps each role to the roles it may create.
// Derived from Level() — a role can only create strictly lower roles.
var CreatableRoles = map[Role][]Role{
	RoleSuperAdmin: {
		RoleRegionalManager, RoleRestaurantManager, RoleShiftManager, RoleCashier,
	},
	RoleRegionalManager: {
		RoleRestaurantManager, RoleShiftManager, RoleCashier,
	},
	RoleRestaurantManager: {
		RoleShiftManager, RoleCashier,
	},
}

// CanCreateRole reports whether r may create the target role.
func (r Role) CanCreateRole(target Role) bool {
	for _, allowed := range CreatableRoles[r] {
		if allowed == target {
			return true
		}
	}
	return false
}

// ═══════════════════════════════════════════════════════════════════
// SCOPE
// ═══════════════════════════════════════════════════════════════════

// Scope defines what data a user can access.
type Scope struct {
	RegionIDs      []uuid.UUID
	RestaurantIDs  []uuid.UUID
	AllRegions     bool
	AllRestaurants bool
}

// CanAccessRegion reports whether the scope includes the region.
func (s Scope) CanAccessRegion(regionID uuid.UUID) bool {
	if s.AllRegions {
		return true
	}
	for _, id := range s.RegionIDs {
		if id == regionID {
			return true
		}
	}
	return false
}

// CanAccessRestaurant reports whether the scope includes the restaurant.
func (s Scope) CanAccessRestaurant(restaurantID uuid.UUID) bool {
	if s.AllRestaurants {
		return true
	}
	for _, id := range s.RestaurantIDs {
		if id == restaurantID {
			return true
		}
	}
	return false
}

// ═══════════════════════════════════════════════════════════════════
// USER CONTEXT
// ═══════════════════════════════════════════════════════════════════

// UserContext represents an authenticated user with role and scope.
type UserContext struct {
	UserID       uuid.UUID
	Email        string
	Role         Role
	RegionID     *uuid.UUID
	RestaurantID *uuid.UUID
	Scope        Scope
}

// HasPermission reports whether the user has the permission.
func (uc *UserContext) HasPermission(perm Permission) bool {
	return uc.Role.HasPermission(perm)
}

// CanAccessRegion reports whether the user can access the region.
func (uc *UserContext) CanAccessRegion(regionID uuid.UUID) bool {
	return uc.Scope.CanAccessRegion(regionID)
}

// CanAccessRestaurant reports whether the user can access the restaurant.
func (uc *UserContext) CanAccessRestaurant(restaurantID uuid.UUID) bool {
	return uc.Scope.CanAccessRestaurant(restaurantID)
}

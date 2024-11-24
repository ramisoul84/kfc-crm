package domain

import "testing"

func TestRoleHierarchy(t *testing.T) {
	order := []Role{RoleCashier, RoleShiftManager, RoleRestaurantManager, RoleRegionalManager, RoleSuperAdmin}
	for i := 1; i < len(order); i++ {
		if !order[i].IsAtLeast(order[i-1]) || order[i-1].IsAtLeast(order[i]) {
			t.Fatalf("%s should outrank %s", order[i], order[i-1])
		}
	}
	if Role("chef").IsValid() {
		t.Fatal("unknown role accepted")
	}
}

// Nobody may create a peer or a superior.
func TestCanOnlyCreateLowerRoles(t *testing.T) {
	for _, actor := range AllRoles {
		for _, target := range AllRoles {
			if actor.CanCreateRole(target) && actor.Level() <= target.Level() {
				t.Errorf("%s may create %s", actor, target)
			}
		}
	}
	if RoleCashier.CanCreateRole(RoleCashier) || RoleShiftManager.CanCreateRole(RoleCashier) {
		t.Fatal("cashiers and shift managers must not create users")
	}
}

// Permissions that move money or points are restricted on purpose.
func TestSensitivePermissions(t *testing.T) {
	cases := []struct {
		role Role
		perm Permission
		want bool
	}{
		{RoleCashier, PermOrderRead, true},
		{RoleCashier, PermOrderCancel, false},
		{RoleShiftManager, PermOrderCancel, true},
		{RoleShiftManager, PermCustomerRead, true},
		{RoleShiftManager, PermCustomerAdjust, false},
		{RoleRestaurantManager, PermCustomerAdjust, true},
		{RoleRestaurantManager, PermMenuItemWrite, false}, // master menu is HQ-only
		{RoleRegionalManager, PermRegionCreate, false},
		{RoleSuperAdmin, PermRegionCreate, true},
	}
	for _, c := range cases {
		if got := c.role.HasPermission(c.perm); got != c.want {
			t.Errorf("%s / %s = %v, want %v", c.role, c.perm, got, c.want)
		}
	}
}

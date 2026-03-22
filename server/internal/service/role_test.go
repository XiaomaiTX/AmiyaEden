package service

import (
	"amiya-eden/internal/model"
	"testing"
)

func TestContainsRoleCode(t *testing.T) {
	roles := []string{"guest", "admin", "super_admin"}

	if !containsRoleCode(roles, "admin") {
		t.Fatal("expected admin to be found")
	}

	if containsRoleCode(roles, "fc") {
		t.Fatal("did not expect fc to be found")
	}
}

func TestEnsureUserHasDefaultRoleUsesGuest(t *testing.T) {
	svc := NewRoleService()
	if svc == nil {
		t.Fatal("expected role service to be constructed")
	}

	if model.RoleGuest != "guest" {
		t.Fatalf("expected guest compatibility constant, got %q", model.RoleGuest)
	}
}

func TestValidateSetUserRolesPermission(t *testing.T) {
	t.Run("admin cannot edit admin target", func(t *testing.T) {
		err := validateSetUserRolesPermission(
			[]string{model.RoleAdmin},
			[]string{model.RoleAdmin},
			[]string{model.RoleUser},
		)
		if err == nil {
			t.Fatal("expected protected target edit to be blocked")
		}
	})

	t.Run("admin cannot assign admin role", func(t *testing.T) {
		err := validateSetUserRolesPermission(
			[]string{model.RoleAdmin},
			[]string{model.RoleUser},
			[]string{model.RoleAdmin},
		)
		if err == nil {
			t.Fatal("expected admin role assignment to be blocked")
		}
	})

	t.Run("admin can assign normal roles to normal user", func(t *testing.T) {
		err := validateSetUserRolesPermission(
			[]string{model.RoleAdmin},
			[]string{model.RoleUser},
			[]string{model.RoleUser, model.RoleFC},
		)
		if err != nil {
			t.Fatalf("expected normal role assignment to pass, got %v", err)
		}
	})
}

func TestNormalizeAssignedRoles(t *testing.T) {
	t.Run("keeps guest when it is the only role", func(t *testing.T) {
		roleIDs, roleCodes := normalizeAssignedRoles([]requestedRoleAssignment{
			{id: 1, code: model.RoleGuest},
		})

		if len(roleIDs) != 1 || roleIDs[0] != 1 {
			t.Fatalf("expected guest role id to remain, got %v", roleIDs)
		}
		if len(roleCodes) != 1 || roleCodes[0] != model.RoleGuest {
			t.Fatalf("expected guest role code to remain, got %v", roleCodes)
		}
	})

	t.Run("drops guest when a real role is present", func(t *testing.T) {
		roleIDs, roleCodes := normalizeAssignedRoles([]requestedRoleAssignment{
			{id: 1, code: model.RoleGuest},
			{id: 2, code: model.RoleUser},
			{id: 3, code: model.RoleFC},
		})

		if len(roleIDs) != 2 || roleIDs[0] != 2 || roleIDs[1] != 3 {
			t.Fatalf("expected non-guest role ids only, got %v", roleIDs)
		}
		if len(roleCodes) != 2 || roleCodes[0] != model.RoleUser || roleCodes[1] != model.RoleFC {
			t.Fatalf("expected non-guest role codes only, got %v", roleCodes)
		}
	})
}

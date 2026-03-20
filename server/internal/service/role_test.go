package service

import "testing"

func TestContainsRoleCode(t *testing.T) {
	roles := []string{"guest", "admin", "super_admin"}

	if !containsRoleCode(roles, "admin") {
		t.Fatal("expected admin to be found")
	}

	if containsRoleCode(roles, "fc") {
		t.Fatal("did not expect fc to be found")
	}
}

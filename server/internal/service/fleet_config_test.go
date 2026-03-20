package service

import (
	"amiya-eden/internal/model"
	"testing"
)

func TestResolveTypeIDFromName(t *testing.T) {
	tests := []struct {
		name         string
		input        string
		nameToTypeID map[string]int64
		wantID       int64
		wantOK       bool
	}{
		{
			name:         "named type",
			input:        "Ferox Navy Issue",
			nameToTypeID: map[string]int64{"Ferox Navy Issue": 72812},
			wantID:       72812,
			wantOK:       true,
		},
		{
			name:         "type id placeholder",
			input:        "TypeID:72812",
			nameToTypeID: map[string]int64{},
			wantID:       72812,
			wantOK:       true,
		},
		{
			name:         "unknown type",
			input:        "Missing Ship",
			nameToTypeID: map[string]int64{},
			wantID:       0,
			wantOK:       false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotID, gotOK := resolveTypeIDFromName(tt.input, tt.nameToTypeID)
			if gotID != tt.wantID || gotOK != tt.wantOK {
				t.Fatalf("resolveTypeIDFromName(%q) = (%d, %v), want (%d, %v)", tt.input, gotID, gotOK, tt.wantID, tt.wantOK)
			}
		})
	}
}

func TestFleetConfigCanManage(t *testing.T) {
	svc := &FleetConfigService{}
	config := &model.FleetConfig{CreatedBy: 42}

	tests := []struct {
		name   string
		userID uint
		roles  []string
		want   bool
	}{
		{name: "super admin", userID: 7, roles: []string{model.RoleSuperAdmin}, want: true},
		{name: "admin", userID: 7, roles: []string{model.RoleAdmin}, want: true},
		{name: "fc", userID: 7, roles: []string{model.RoleFC}, want: true},
		{name: "owner user", userID: 42, roles: []string{model.RoleUser}, want: true},
		{name: "srp non owner", userID: 7, roles: []string{model.RoleSRP}, want: false},
		{name: "user non owner", userID: 7, roles: []string{model.RoleUser}, want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := svc.canManage(config, tt.userID, tt.roles); got != tt.want {
				t.Fatalf("canManage() = %v, want %v", got, tt.want)
			}
		})
	}
}

package service

import (
	"amiya-eden/global"
	"amiya-eden/internal/model"
	"testing"
)

func TestAutoRoleMappingAuditEvents(t *testing.T) {
	db := newServiceTestDB(t, "auto_role_audit", &model.EsiRoleMapping{}, &model.EsiTitleMapping{}, &model.AuditEvent{})
	previous := global.DB
	global.DB = db
	t.Cleanup(func() { global.DB = previous })

	svc := NewAutoRoleService()

	roleMapping, err := svc.CreateEsiRoleMappingByOperator("Director", model.RoleAdmin, 101)
	if err != nil {
		t.Fatalf("CreateEsiRoleMappingByOperator() error = %v", err)
	}
	if err := svc.DeleteEsiRoleMappingByOperator(roleMapping.ID, 101); err != nil {
		t.Fatalf("DeleteEsiRoleMappingByOperator() error = %v", err)
	}

	titleMapping, err := svc.CreateEsiTitleMappingByOperator(99000001, 7, "Director", model.RoleFC, 102)
	if err != nil {
		t.Fatalf("CreateEsiTitleMappingByOperator() error = %v", err)
	}
	if err := svc.DeleteEsiTitleMappingByOperator(titleMapping.ID, 102); err != nil {
		t.Fatalf("DeleteEsiTitleMappingByOperator() error = %v", err)
	}

	var events []model.AuditEvent
	if err := db.Order("occurred_at ASC").Find(&events).Error; err != nil {
		t.Fatalf("load audit events: %v", err)
	}
	if len(events) != 4 {
		t.Fatalf("expected 4 audit events, got %d", len(events))
	}

	wantActions := []string{
		"esi_role_mapping_create",
		"esi_role_mapping_delete",
		"esi_title_mapping_create",
		"esi_title_mapping_delete",
	}
	for i, want := range wantActions {
		if events[i].Category != "permission" {
			t.Fatalf("event %d category = %q, want permission", i, events[i].Category)
		}
		if events[i].Action != want {
			t.Fatalf("event %d action = %q, want %q", i, events[i].Action, want)
		}
		if events[i].Result != model.AuditResultSuccess {
			t.Fatalf("event %d result = %q, want %q", i, events[i].Result, model.AuditResultSuccess)
		}
	}
}

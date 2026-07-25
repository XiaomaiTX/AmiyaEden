package service

import (
	"amiya-eden/global"
	"amiya-eden/internal/model"
	"amiya-eden/internal/repository"
	"context"
	"encoding/json"
	"testing"
)

func TestQQGovernanceServiceSavePolicy(t *testing.T) {
	db := newServiceTestDB(t, "qq_governance_policy", &model.QQGroupGovernancePolicy{}, &model.QQGovernanceActionTask{}, &model.EveEntityNameCache{})
	origDB := global.DB
	global.DB = db
	defer func() { global.DB = origDB }()
	svc := NewQQGovernanceServiceWithRepository(repository.NewQQGovernanceRepositoryWithDB(db))
	policy, err := svc.SavePolicy(context.Background(), QQGovernancePolicyInput{GroupID: 100, Enabled: true, AllowedCorporationIDs: []int64{200}, AllowedRoleCodes: []string{model.RoleAdmin}, MemberViolationPolicy: model.QQGovernanceViolationAutoKick, CardTemplate: "{nickname}", CardSyncEnabled: true}, 1)
	if err != nil {
		t.Fatalf("save valid policy: %v", err)
	}
	if policy.GroupID != 100 || policy.MemberViolationPolicy != model.QQGovernanceViolationAutoKick || !policy.CardSyncEnabled {
		t.Fatalf("policy = %#v", policy)
	}
}

func TestIsQQGovernanceAdminRole(t *testing.T) {
	tests := []struct {
		role string
		want bool
	}{
		{role: "owner", want: true},
		{role: "admin", want: true},
		{role: "member", want: false},
		{role: "", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.role, func(t *testing.T) {
			if got := isQQGovernanceAdminRole(tt.role); got != tt.want {
				t.Fatalf("isQQGovernanceAdminRole(%q) = %t, want %t", tt.role, got, tt.want)
			}
		})
	}
}

type qqGovernanceAdminTestExecutor struct{ calls int }

func (*qqGovernanceAdminTestExecutor) OneBotConnected() bool { return true }

func (e *qqGovernanceAdminTestExecutor) CallOneBot(_ context.Context, _ string, _ map[string]any) (json.RawMessage, error) {
	e.calls++
	return json.RawMessage(`{}`), nil
}

func TestListGroupStatusesUsesPersistedSnapshotOnly(t *testing.T) {
	db := newServiceTestDB(t, "qq_governance_group_status",
		&model.SystemConfig{}, &model.QQGroupGovernancePolicy{}, &model.QQGroupRuntimeSnapshot{},
		&model.QQGroupMemberState{}, &model.QQGovernanceReconcileRun{},
	)
	oldDB := global.DB
	global.DB = db
	defer func() {
		global.DB = oldDB
	}()
	botIsAdmin := false
	if err := db.Create(&model.QQGroupGovernancePolicy{GroupID: 123, Enabled: true}).Error; err != nil {
		t.Fatalf("seed policy: %v", err)
	}
	if err := db.Create(&model.QQGroupRuntimeSnapshot{GroupID: 123, GroupName: "saved-group", MemberCount: 8, MaxMemberCount: 500, BotIsAdmin: &botIsAdmin}).Error; err != nil {
		t.Fatalf("seed snapshot: %v", err)
	}

	svc := NewQQGovernanceServiceWithRepository(repository.NewQQGovernanceRepositoryWithDB(db))
	executor := &qqGovernanceAdminTestExecutor{}
	svc.SetOneBotActionExecutor(executor)
	rows, err := svc.ListGroupStatuses(context.Background())
	if err != nil {
		t.Fatalf("list group statuses: %v", err)
	}
	if len(rows) != 1 || rows[0].GroupName != "saved-group" || rows[0].BotIsAdmin == nil || *rows[0].BotIsAdmin {
		t.Fatalf("group statuses = %#v", rows)
	}
	if executor.calls != 0 {
		t.Fatalf("OneBot calls = %d, want 0", executor.calls)
	}
}

package service

import (
	"amiya-eden/global"
	"amiya-eden/internal/model"
	"amiya-eden/internal/repository"
	"context"
	"encoding/json"
	"testing"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
)

func TestQQGovernanceServiceSavePolicy(t *testing.T) {
	db := newServiceTestDB(t, "qq_governance_policy", &model.QQGroupGovernancePolicy{}, &model.EveEntityNameCache{})
	origDB := global.DB
	global.DB = db
	defer func() { global.DB = origDB }()
	svc := NewQQGovernanceServiceWithRepository(repository.NewQQGovernanceRepositoryWithDB(db))
	policy, err := svc.SavePolicy(context.Background(), QQGovernancePolicyInput{GroupID: 100, Enabled: true, AllowedCorporationIDs: []int64{200}, AllowedRoleCodes: []string{model.RoleAdmin}, MemberViolationPolicy: model.QQGovernanceViolationAutoKick, CardTemplate: "{nickname}"}, 1)
	if err != nil {
		t.Fatalf("save valid policy: %v", err)
	}
	if policy.GroupID != 100 || policy.MemberViolationPolicy != model.QQGovernanceViolationAutoKick {
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

type qqGovernanceAdminTestExecutor struct{}

func (qqGovernanceAdminTestExecutor) OneBotConnected() bool { return true }

func (qqGovernanceAdminTestExecutor) CallOneBot(_ context.Context, action string, _ map[string]any) (json.RawMessage, error) {
	switch action {
	case "get_group_info":
		return json.RawMessage(`{"group_name":"test-group","member_count":8,"max_member_count":500}`), nil
	case "get_group_member_info":
		return json.RawMessage(`{"role":"member"}`), nil
	default:
		return json.RawMessage(`{}`), nil
	}
}

func TestEnrichGroupStatusFromOneBotUsesGroupInfoWhenBotIsNotAdmin(t *testing.T) {
	db := newServiceTestDB(t, "qq_governance_group_status", &model.SystemConfig{})
	if err := db.Create(&model.SystemConfig{Key: model.SysConfigOneBotBotQQ, Value: "9001"}).Error; err != nil {
		t.Fatalf("seed bot qq: %v", err)
	}

	oldDB, oldRedis := global.DB, global.Redis
	global.DB = db
	mini := miniredis.RunT(t)
	global.Redis = redis.NewClient(&redis.Options{Addr: mini.Addr()})
	defer func() {
		_ = global.Redis.Close()
		global.DB, global.Redis = oldDB, oldRedis
	}()

	svc := NewQQGovernanceServiceWithRepository(repository.NewQQGovernanceRepositoryWithDB(db))
	svc.SetOneBotActionExecutor(qqGovernanceAdminTestExecutor{})
	item := &QQGovernanceGroupStatus{GroupID: 123, GroupName: "-"}

	svc.enrichGroupStatusFromOneBot(context.Background(), item)

	if item.GroupName != "test-group" || item.MemberCount != 8 || item.MaxMemberCount != 500 {
		t.Fatalf("group status = %#v", item)
	}
	if item.BotIsAdmin == nil || *item.BotIsAdmin {
		t.Fatalf("bot admin status = %#v, want false", item.BotIsAdmin)
	}
}

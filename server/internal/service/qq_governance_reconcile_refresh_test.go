package service

import (
	"amiya-eden/global"
	"amiya-eden/internal/model"
	"amiya-eden/internal/repository"
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

type qqGovernanceReconcileTestExecutor struct {
	calls      []string
	membersRaw json.RawMessage
	groupRaw   json.RawMessage
}

func (*qqGovernanceReconcileTestExecutor) OneBotConnected() bool { return true }

func (e *qqGovernanceReconcileTestExecutor) CallOneBot(_ context.Context, action string, _ map[string]any) (json.RawMessage, error) {
	e.calls = append(e.calls, action)
	if action == "get_group_member_list" {
		return e.membersRaw, nil
	}
	if action == "get_group_info" {
		return e.groupRaw, nil
	}
	return nil, nil
}

func setupQQGovernanceRedis(t *testing.T, db *gorm.DB) func() {
	t.Helper()
	oldDB, oldRedis := global.DB, global.Redis
	global.DB = db
	mini := miniredis.RunT(t)
	global.Redis = redis.NewClient(&redis.Options{Addr: mini.Addr()})
	return func() {
		_ = global.Redis.Close()
		global.DB, global.Redis = oldDB, oldRedis
	}
}

func TestQQGovernanceBotIsAdminFromSnapshotMembers(t *testing.T) {
	tests := []struct {
		name    string
		members []oneBotGroupMember
		want    *bool
	}{
		{name: "owner", members: []oneBotGroupMember{{UserID: 9001, Role: "owner"}}, want: boolPtr(true)},
		{name: "admin", members: []oneBotGroupMember{{UserID: 9001, Role: "admin"}}, want: boolPtr(true)},
		{name: "member", members: []oneBotGroupMember{{UserID: 9001, Role: "member"}}, want: boolPtr(false)},
		{name: "absent", members: []oneBotGroupMember{{UserID: 100}}, want: nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := qqGovernanceBotIsAdmin(tt.members, 9001)
			if (got == nil) != (tt.want == nil) || got != nil && *got != *tt.want {
				t.Fatalf("qqGovernanceBotIsAdmin() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCaptureReconcileSnapshotQueuesGroupInfoWithoutSecondOneBotCall(t *testing.T) {
	db := newServiceTestDB(t, "qq_governance_snapshot_group_info",
		&model.SystemConfig{}, &model.QQGroupGovernancePolicy{}, &model.QQGovernanceReconcileRun{},
		&model.QQGovernanceReconcileMember{}, &model.QQGovernanceActionTask{}, &model.QQGovernanceActionLog{},
		&model.QQGroupRuntimeSnapshot{},
	)
	cleanup := setupQQGovernanceRedis(t, db)
	defer cleanup()
	const groupID int64 = 778899
	now := time.Date(2026, 7, 25, 12, 0, 0, 0, time.UTC)
	if err := db.Create(&model.SystemConfig{Key: model.SysConfigOneBotBotQQ, Value: "9001"}).Error; err != nil {
		t.Fatalf("seed bot QQ: %v", err)
	}
	if err := db.Create(&model.QQGroupGovernancePolicy{GroupID: groupID, Enabled: true}).Error; err != nil {
		t.Fatalf("seed policy: %v", err)
	}
	run := &model.QQGovernanceReconcileRun{GroupID: groupID, ActiveKey: "group:778899", Status: model.QQGovernanceRunPending, StartedAt: &now}
	if err := db.Create(run).Error; err != nil {
		t.Fatalf("seed run: %v", err)
	}
	task := &model.QQGovernanceActionTask{ActionType: model.QQGovernanceActionSnapshot, RunID: run.ID, GroupID: groupID, IdempotencyKey: "snapshot-test", Status: model.QQGovernanceActionRunning, Priority: 20, RunAfter: now, LeaseToken: "snapshot-lease"}
	if err := db.Create(task).Error; err != nil {
		t.Fatalf("seed task: %v", err)
	}
	executor := &qqGovernanceReconcileTestExecutor{membersRaw: json.RawMessage(`[{"user_id":9001,"role":"admin"},{"user_id":100}]`)}
	svc := NewQQGovernanceServiceWithRepository(repository.NewQQGovernanceRepositoryWithDB(db))
	svc.now = func() time.Time { return now }
	svc.SetOneBotActionExecutor(executor)

	if err := svc.captureReconcileSnapshot(t.Context(), task, &model.QQGroupGovernancePolicy{GroupID: groupID, Enabled: true}, run.ID); err != nil {
		t.Fatalf("capture snapshot: %v", err)
	}
	if len(executor.calls) != 1 || executor.calls[0] != "get_group_member_list" {
		t.Fatalf("OneBot calls = %#v", executor.calls)
	}
	var snapshot model.QQGroupRuntimeSnapshot
	if err := db.Where("group_id = ?", groupID).First(&snapshot).Error; err != nil {
		t.Fatalf("load snapshot: %v", err)
	}
	if snapshot.MemberCount != 2 || snapshot.BotIsAdmin == nil || !*snapshot.BotIsAdmin {
		t.Fatalf("snapshot = %#v", snapshot)
	}
	var tasks []model.QQGovernanceActionTask
	if err := db.Order("priority ASC").Find(&tasks).Error; err != nil {
		t.Fatalf("list tasks: %v", err)
	}
	if len(tasks) != 3 || tasks[1].ActionType != model.QQGovernanceActionRefreshGroupInfo || tasks[1].RunID != run.ID {
		t.Fatalf("tasks = %#v", tasks)
	}
}

func TestRefreshGroupInfoRespectsRateLimitAndUpdatesSnapshot(t *testing.T) {
	db := newServiceTestDB(t, "qq_governance_refresh_group_info",
		&model.QQGroupGovernancePolicy{}, &model.QQGovernanceActionTask{}, &model.QQGovernanceActionLog{},
		&model.QQGroupRuntimeSnapshot{},
	)
	cleanup := setupQQGovernanceRedis(t, db)
	defer cleanup()
	const groupID int64 = 778899
	now := time.Date(2026, 7, 25, 12, 0, 0, 0, time.UTC)
	if err := db.Create(&model.QQGroupGovernancePolicy{GroupID: groupID, Enabled: true}).Error; err != nil {
		t.Fatalf("seed policy: %v", err)
	}
	svc := NewQQGovernanceServiceWithRepository(repository.NewQQGovernanceRepositoryWithDB(db))
	svc.now = func() time.Time { return now }
	executor := &qqGovernanceReconcileTestExecutor{groupRaw: json.RawMessage(`{"group_name":"test-group","member_count":8,"max_member_count":500}`)}
	svc.SetOneBotActionExecutor(executor)
	if _, err := svc.acquireQQGovernanceRateLimit(t.Context(), &model.QQGovernanceActionTask{ActionType: model.QQGovernanceActionRefreshGroupInfo, GroupID: groupID}); err != nil {
		t.Fatalf("consume rate-limit token: %v", err)
	}
	task := &model.QQGovernanceActionTask{ActionType: model.QQGovernanceActionRefreshGroupInfo, GroupID: groupID, IdempotencyKey: "group-info-test", Status: model.QQGovernanceActionRunning, Priority: 30, RunAfter: now, LeaseToken: "group-info-lease"}
	if err := db.Create(task).Error; err != nil {
		t.Fatalf("seed task: %v", err)
	}
	if err := svc.refreshQQGovernanceGroupInfo(t.Context(), task); err != nil {
		t.Fatalf("refresh rate-limited group info: %v", err)
	}
	if len(executor.calls) != 0 {
		t.Fatalf("OneBot calls while rate-limited = %#v", executor.calls)
	}
	var rateLimited model.QQGovernanceActionTask
	if err := db.First(&rateLimited, task.ID).Error; err != nil {
		t.Fatalf("reload rate-limited task: %v", err)
	}
	if rateLimited.Status != model.QQGovernanceActionRetryWait {
		t.Fatalf("task status = %q, want %q", rateLimited.Status, model.QQGovernanceActionRetryWait)
	}

	freshTask := &model.QQGovernanceActionTask{ActionType: model.QQGovernanceActionRefreshGroupInfo, GroupID: groupID + 1, IdempotencyKey: "group-info-success", Status: model.QQGovernanceActionRunning, Priority: 30, RunAfter: now, LeaseToken: "group-info-success-lease"}
	if err := db.Create(&model.QQGroupGovernancePolicy{GroupID: groupID + 1, Enabled: true}).Error; err != nil {
		t.Fatalf("seed second policy: %v", err)
	}
	if err := db.Create(freshTask).Error; err != nil {
		t.Fatalf("seed success task: %v", err)
	}
	if err := svc.refreshQQGovernanceGroupInfo(t.Context(), freshTask); err != nil {
		t.Fatalf("refresh group info: %v", err)
	}
	if len(executor.calls) != 1 || executor.calls[0] != "get_group_info" {
		t.Fatalf("OneBot calls = %#v", executor.calls)
	}
	var snapshot model.QQGroupRuntimeSnapshot
	if err := db.Where("group_id = ?", groupID+1).First(&snapshot).Error; err != nil {
		t.Fatalf("load refreshed snapshot: %v", err)
	}
	if snapshot.GroupName != "test-group" || snapshot.MemberCount != 8 || snapshot.MaxMemberCount != 500 {
		t.Fatalf("snapshot = %#v", snapshot)
	}
}

func boolPtr(value bool) *bool { return &value }

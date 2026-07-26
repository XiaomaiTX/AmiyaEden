package service

import (
	"amiya-eden/global"
	"amiya-eden/internal/model"
	"amiya-eden/internal/repository"
	"context"
	"encoding/json"
	"testing"
	"time"
)

type qqGovernanceConnectionTestExecutor struct{ connected bool }

func (e qqGovernanceConnectionTestExecutor) OneBotConnected() bool { return e.connected }
func (qqGovernanceConnectionTestExecutor) CallOneBot(context.Context, string, map[string]any) (json.RawMessage, error) {
	return nil, nil
}

func TestQQGovernanceReconnectRecoversOnlyDisconnectedRetryWaitTasks(t *testing.T) {
	db := newQQGovernanceCardSyncDB(t, "qq_governance_reconnect_recovery")
	now := time.Date(2026, 7, 27, 10, 0, 0, 0, time.UTC)
	tasks := []model.QQGovernanceActionTask{
		{ActionType: model.QQGovernanceActionSnapshot, IdempotencyKey: "retry-disconnected", GroupID: 100, Status: model.QQGovernanceActionRetryWait, RetryCount: 4, RetryCause: model.QQGovernanceRetryCauseDisconnected, LastError: oneBotDisconnectedMessage, RunAfter: now.Add(time.Hour)},
		{ActionType: model.QQGovernanceActionSnapshot, IdempotencyKey: "retry-other", GroupID: 101, Status: model.QQGovernanceActionRetryWait, RetryCount: 4, RetryCause: model.QQGovernanceRetryCauseRetryable, LastError: "temporary error", RunAfter: now.Add(time.Hour)},
		{ActionType: model.QQGovernanceActionSnapshot, IdempotencyKey: "dead-disconnected", GroupID: 102, Status: model.QQGovernanceActionDead, RetryCount: 5, RetryCause: model.QQGovernanceRetryCauseDisconnected, LastError: oneBotDisconnectedMessage, RunAfter: now.Add(time.Hour)},
	}
	if err := db.Create(&tasks).Error; err != nil {
		t.Fatalf("create tasks: %v", err)
	}
	svc := NewQQGovernanceServiceWithRepository(repository.NewQQGovernanceRepositoryWithDB(db))
	svc.now = func() time.Time { return now }
	svc.OnOneBotConnected()

	var rows []model.QQGovernanceActionTask
	if err := db.Order("id ASC").Find(&rows).Error; err != nil {
		t.Fatalf("reload tasks: %v", err)
	}
	if rows[0].Status != model.QQGovernanceActionPending || rows[0].RetryCount != 0 || rows[0].RetryCause != "" || !rows[0].RunAfter.Equal(now) {
		t.Fatalf("disconnected retry task was not recovered: %#v", rows[0])
	}
	if rows[1].Status != model.QQGovernanceActionRetryWait || rows[1].RetryCount != 4 {
		t.Fatalf("non-disconnected task should remain untouched: %#v", rows[1])
	}
	if rows[2].Status != model.QQGovernanceActionDead {
		t.Fatalf("dead task must not be automatically recovered: %#v", rows[2])
	}
}

func TestQQGovernanceManualRecoveryRequiresConnectionAndRecoversOnlyDisconnected(t *testing.T) {
	db := newQQGovernanceCardSyncDB(t, "qq_governance_manual_recovery")
	now := time.Date(2026, 7, 27, 10, 0, 0, 0, time.UTC)
	tasks := []model.QQGovernanceActionTask{
		{ActionType: model.QQGovernanceActionSnapshot, IdempotencyKey: "dead-disconnected", GroupID: 100, Status: model.QQGovernanceActionDead, RetryCount: 5, RetryCause: model.QQGovernanceRetryCauseDisconnected, RunAfter: now},
		{ActionType: model.QQGovernanceActionSnapshot, IdempotencyKey: "dead-other", GroupID: 101, Status: model.QQGovernanceActionDead, RetryCount: 5, RetryCause: model.QQGovernanceRetryCauseRetryable, RunAfter: now},
	}
	if err := db.Create(&tasks).Error; err != nil {
		t.Fatalf("create tasks: %v", err)
	}
	svc := NewQQGovernanceServiceWithRepository(repository.NewQQGovernanceRepositoryWithDB(db))
	svc.now = func() time.Time { return now }
	svc.SetOneBotActionExecutor(qqGovernanceConnectionTestExecutor{})
	if _, err := svc.RecoverDisconnectedTasks(1); err == nil {
		t.Fatal("expected disconnected manual recovery to fail")
	}
	svc.SetOneBotActionExecutor(qqGovernanceConnectionTestExecutor{connected: true})
	result, err := svc.RecoverDisconnectedTasks(1)
	if err != nil || result.RecoveredTasks != 1 {
		t.Fatalf("manual recovery result = %#v, err = %v", result, err)
	}
	var rows []model.QQGovernanceActionTask
	if err := db.Order("id ASC").Find(&rows).Error; err != nil {
		t.Fatalf("reload tasks: %v", err)
	}
	if rows[0].Status != model.QQGovernanceActionPending || rows[1].Status != model.QQGovernanceActionDead {
		t.Fatalf("manual recovery touched unexpected tasks: %#v", rows)
	}
}

func TestQQGovernanceTriggerReconcileResumesOnlyDisconnectedSnapshot(t *testing.T) {
	db := newQQGovernanceCardSyncDB(t, "qq_governance_reconcile_recovery")
	now := time.Date(2026, 7, 27, 10, 0, 0, 0, time.UTC)
	policy := &model.QQGroupGovernancePolicy{GroupID: 778899, Enabled: true, AllowedCorporationIDsJSON: "[]", AllowedRoleCodesJSON: "[]"}
	run := &model.QQGovernanceReconcileRun{GroupID: policy.GroupID, ActiveKey: "group:778899", Status: model.QQGovernanceRunPending}
	if err := db.Create(policy).Error; err != nil {
		t.Fatalf("create policy: %v", err)
	}
	if err := db.Create(run).Error; err != nil {
		t.Fatalf("create run: %v", err)
	}
	task := &model.QQGovernanceActionTask{ActionType: model.QQGovernanceActionSnapshot, RunID: run.ID, IdempotencyKey: "snapshot-recovery", GroupID: policy.GroupID, Status: model.QQGovernanceActionDead, RetryCount: 5, RetryCause: model.QQGovernanceRetryCauseDisconnected, RunAfter: now}
	if err := db.Create(task).Error; err != nil {
		t.Fatalf("create task: %v", err)
	}
	svc := NewQQGovernanceServiceWithRepository(repository.NewQQGovernanceRepositoryWithDB(db))
	svc.now = func() time.Time { return now }
	svc.SetOneBotActionExecutor(qqGovernanceConnectionTestExecutor{connected: true})
	result, err := svc.TriggerReconcile(t.Context(), policy.GroupID, 1)
	if err != nil || result.Status != "resumed" || result.RecoveredTasks != 1 {
		t.Fatalf("trigger result = %#v, err = %v", result, err)
	}
	if err := db.First(task, task.ID).Error; err != nil {
		t.Fatalf("reload task: %v", err)
	}
	if task.Status != model.QQGovernanceActionPending || task.RetryCount != 0 {
		t.Fatalf("snapshot task was not resumed: %#v", task)
	}
	task.RetryCause, task.Status = model.QQGovernanceRetryCauseRetryable, model.QQGovernanceActionDead
	if err := db.Save(task).Error; err != nil {
		t.Fatalf("block task: %v", err)
	}
	result, err = svc.TriggerReconcile(t.Context(), policy.GroupID, 1)
	if err != nil || result.Status != "blocked" {
		t.Fatalf("blocked trigger result = %#v, err = %v", result, err)
	}
}

func TestQQGovernanceTriggerReconcileCreatesSnapshotWithoutActiveRun(t *testing.T) {
	db := newQQGovernanceCardSyncDB(t, "qq_governance_reconcile_create")
	policy := &model.QQGroupGovernancePolicy{GroupID: 778900, Enabled: true, AllowedCorporationIDsJSON: "[]", AllowedRoleCodesJSON: "[]"}
	if err := db.Create(policy).Error; err != nil {
		t.Fatalf("create policy: %v", err)
	}
	svc := NewQQGovernanceServiceWithRepository(repository.NewQQGovernanceRepositoryWithDB(db))
	result, err := svc.TriggerReconcile(t.Context(), policy.GroupID, 1)
	if err != nil || result.Status != "created" {
		t.Fatalf("trigger result = %#v, err = %v", result, err)
	}
	var task model.QQGovernanceActionTask
	if err := db.Where("group_id = ? AND action_type = ?", policy.GroupID, model.QQGovernanceActionSnapshot).First(&task).Error; err != nil {
		t.Fatalf("expected snapshot task: %v", err)
	}
	if task.Status != model.QQGovernanceActionPending || task.RunID == 0 {
		t.Fatalf("unexpected snapshot task: %#v", task)
	}
}

func TestQQGovernanceRateLimitStatusReportsBucketsAndRedisUnavailability(t *testing.T) {
	db := newQQGovernanceCardSyncDB(t, "qq_governance_rate_status")
	cleanup := setupQQGovernanceRedis(t, db)
	defer cleanup()
	now := time.Date(2026, 7, 27, 10, 0, 0, 0, time.UTC)
	if err := db.Create(&model.QQGroupGovernancePolicy{GroupID: 778899, AllowedCorporationIDsJSON: "[]", AllowedRoleCodesJSON: "[]"}).Error; err != nil {
		t.Fatalf("create policy: %v", err)
	}
	if err := global.Redis.HSet(t.Context(), "qq_governance:rate:global", "tokens", 0, "last", now.UnixMilli()).Err(); err != nil {
		t.Fatalf("seed global bucket: %v", err)
	}
	if err := global.Redis.HSet(t.Context(), "qq_governance:rate:group:778899", "tokens", 0, "last", now.UnixMilli()).Err(); err != nil {
		t.Fatalf("seed group bucket: %v", err)
	}
	svc := NewQQGovernanceServiceWithRepository(repository.NewQQGovernanceRepositoryWithDB(db))
	svc.now = func() time.Time { return now }
	status := svc.rateLimitStatus()
	if !status.Available || status.Global.WaitMS != qqGovernanceGlobalRefillInterval.Milliseconds() || len(status.Groups) != 1 || status.Groups[0].Bucket.WaitMS != qqGovernanceGroupActionInterval.Milliseconds() {
		t.Fatalf("unexpected rate status: %#v", status)
	}
	oldRedis := global.Redis
	global.Redis = nil
	if status = svc.rateLimitStatus(); status.Available {
		t.Fatalf("expected unavailable status without redis: %#v", status)
	}
	global.Redis = oldRedis
}

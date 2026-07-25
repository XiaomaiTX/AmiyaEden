package service

import (
	"amiya-eden/internal/model"
	"amiya-eden/internal/repository"
	"encoding/json"
	"errors"
	"fmt"
	"testing"
	"time"

	"gorm.io/gorm"
)

func TestQQGovernanceServiceGroupRequestCreatesOneApprovalTask(t *testing.T) {
	db := newServiceTestDB(t, "qq_governance",
		&model.User{}, &model.EveCharacter{}, &model.UserRole{},
		&model.QQGroupGovernancePolicy{}, &model.QQGovernanceEvent{},
		&model.QQGroupMemberState{}, &model.QQGovernanceReview{},
		&model.QQGovernanceActionTask{}, &model.QQGovernanceActionLog{},
	)
	if err := db.Create(&model.User{BaseModel: model.BaseModel{ID: 9}, QQ: "123456", Nickname: "Amiya", PrimaryCharacterID: 42, Role: model.RoleGuest}).Error; err != nil {
		t.Fatalf("create user: %v", err)
	}
	if err := db.Create(&model.EveCharacter{CharacterID: 42, CharacterName: "Amiya Prime", UserID: 9, CorporationID: 1001}).Error; err != nil {
		t.Fatalf("create character: %v", err)
	}
	if err := db.Create(&model.UserRole{UserID: 9, RoleCode: model.RoleAdmin}).Error; err != nil {
		t.Fatalf("create role: %v", err)
	}
	corps, _ := json.Marshal([]int64{1001})
	roles, _ := json.Marshal([]string{model.RoleAdmin})
	if err := db.Create(&model.QQGroupGovernancePolicy{GroupID: 778899, Enabled: true, AllowedCorporationIDsJSON: string(corps), AllowedRoleCodesJSON: string(roles), CardTemplate: "{nickname}-{primary_character_name}"}).Error; err != nil {
		t.Fatalf("create policy: %v", err)
	}

	svc := NewQQGovernanceServiceWithRepository(repository.NewQQGovernanceRepositoryWithDB(db))
	now := time.Date(2026, 7, 11, 12, 0, 0, 0, time.UTC)
	svc.now = func() time.Time { return now }
	event := QQGovernanceInboundEvent{EventKey: "request/group_add:778899:123456:flag", EventType: "request/group_add", GroupID: 778899, QQ: 123456, RequestFlag: "flag"}
	if err := svc.HandleOneBotEvent(t.Context(), event); err != nil {
		t.Fatalf("handle event: %v", err)
	}
	if err := svc.HandleOneBotEvent(t.Context(), event); err != nil {
		t.Fatalf("handle duplicate event: %v", err)
	}

	var tasks []model.QQGovernanceActionTask
	if err := db.Find(&tasks).Error; err != nil {
		t.Fatalf("list tasks: %v", err)
	}
	if len(tasks) != 1 {
		t.Fatalf("task count = %d, want 1", len(tasks))
	}
	if tasks[0].ActionType != model.QQGovernanceActionApprove || tasks[0].Priority != 10 || tasks[0].TargetVersion != 1 {
		t.Fatalf("approval task = %#v", tasks[0])
	}
	var review model.QQGovernanceReview
	if err := db.First(&review).Error; err != nil {
		t.Fatalf("load review: %v", err)
	}
	if review.Decision != model.QQGovernanceReviewMatched || review.UserID != 9 {
		t.Fatalf("review = %#v", review)
	}
}

func TestQQGovernanceEvaluateMissingSeatDataIsUnmatched(t *testing.T) {
	db := newServiceTestDB(t, "qq_governance_missing_data",
		&model.User{}, &model.EveCharacter{}, &model.UserRole{},
		&model.QQGroupGovernancePolicy{},
	)
	policy := &model.QQGroupGovernancePolicy{
		GroupID:                   778899,
		Enabled:                   true,
		AllowedCorporationIDsJSON: `[1001]`,
		AllowedRoleCodesJSON:      `["admin"]`,
	}
	svc := NewQQGovernanceServiceWithRepository(repository.NewQQGovernanceRepositoryWithDB(db))
	var decision qqEligibility
	if err := db.Transaction(func(tx *gorm.DB) error {
		var err error
		decision, err = svc.evaluate(tx, policy, 123456)
		return err
	}); err != nil {
		t.Fatalf("evaluate: %v", err)
	}
	if decision.Decision != model.QQGovernanceReviewUnmatched {
		t.Fatalf("decision = %q, want unmatched", decision.Decision)
	}
	if decision.Reason != "QQ 未绑定 Seat 用户" {
		t.Fatalf("reason = %q", decision.Reason)
	}
}

func TestQQGovernanceReconcileRunIsReused(t *testing.T) {
	db := newServiceTestDB(t, "qq_governance_reconcile_run",
		&model.QQGroupGovernancePolicy{}, &model.QQGovernanceActionTask{},
		&model.QQGovernanceReconcileRun{}, &model.QQGovernanceReconcileMember{},
	)
	if err := db.Create(&model.QQGroupGovernancePolicy{GroupID: 778899, Enabled: true}).Error; err != nil {
		t.Fatalf("create policy: %v", err)
	}
	svc := NewQQGovernanceServiceWithRepository(repository.NewQQGovernanceRepositoryWithDB(db))
	if err := svc.startReconcileRun(t.Context(), 778899); err != nil {
		t.Fatalf("start first run: %v", err)
	}
	if err := svc.startReconcileRun(t.Context(), 778899); err != nil {
		t.Fatalf("reuse active run: %v", err)
	}
	var runs []model.QQGovernanceReconcileRun
	var tasks []model.QQGovernanceActionTask
	if err := db.Find(&runs).Error; err != nil {
		t.Fatalf("list runs: %v", err)
	}
	if err := db.Find(&tasks).Error; err != nil {
		t.Fatalf("list tasks: %v", err)
	}
	if len(runs) != 1 || len(tasks) != 1 || tasks[0].ActionType != model.QQGovernanceActionSnapshot {
		t.Fatalf("runs=%d tasks=%#v", len(runs), tasks)
	}
}

// applyQQGovernanceReconcileRunActiveIndex mirrors the partial unique index
// bootstrap/db.go installs for qq_governance_reconcile_run so the regression
// tests below exercise the same shape on SQLite.
func applyQQGovernanceReconcileRunActiveIndex(t *testing.T, db *gorm.DB) {
	t.Helper()
	stmts := []string{
		`DROP INDEX IF EXISTS idx_qq_governance_reconcile_run_active_key`,
		`CREATE UNIQUE INDEX IF NOT EXISTS idx_qq_governance_reconcile_run_active_group ON qq_governance_reconcile_run (group_id) WHERE active_key <> '' AND deleted_at IS NULL`,
	}
	for _, stmt := range stmts {
		if err := db.Exec(stmt).Error; err != nil {
			t.Fatalf("apply reconcile run index: %v", err)
		}
	}
}

func TestQQGovernanceReconcileRunTerminalRunsCoexist(t *testing.T) {
	db := newServiceTestDB(t, "qq_governance_reconcile_run_terminal",
		&model.QQGroupGovernancePolicy{}, &model.QQGovernanceActionTask{},
		&model.QQGovernanceReconcileRun{}, &model.QQGovernanceReconcileMember{},
	)
	applyQQGovernanceReconcileRunActiveIndex(t, db)

	now := time.Date(2026, 7, 24, 12, 0, 0, 0, time.UTC)
	for i := range 3 {
		run := &model.QQGovernanceReconcileRun{
			GroupID:        778899,
			ActiveKey:      "",
			Status:         model.QQGovernanceRunCompleted,
			ExpectedCount:  10,
			ProcessedCount: 10,
			StartedAt:      &now,
			CompletedAt:    &now,
		}
		run.ID = uint(i + 1)
		if err := db.Create(run).Error; err != nil {
			t.Fatalf("create terminal run %d: %v", i+1, err)
		}
	}
	var count int64
	if err := db.Model(&model.QQGovernanceReconcileRun{}).Count(&count).Error; err != nil {
		t.Fatalf("count runs: %v", err)
	}
	if count != 3 {
		t.Fatalf("terminal run count = %d, want 3", count)
	}
}

func TestQQGovernanceReconcileRunActiveKeyIsUniquePerGroup(t *testing.T) {
	db := newServiceTestDB(t, "qq_governance_reconcile_run_active_unique",
		&model.QQGroupGovernancePolicy{}, &model.QQGovernanceActionTask{},
		&model.QQGovernanceReconcileRun{}, &model.QQGovernanceReconcileMember{},
	)
	applyQQGovernanceReconcileRunActiveIndex(t, db)

	now := time.Date(2026, 7, 24, 12, 0, 0, 0, time.UTC)
	first := &model.QQGovernanceReconcileRun{
		GroupID: 778899, ActiveKey: "group:778899", Status: model.QQGovernanceRunRunning,
		StartedAt: &now,
	}
	if err := db.Create(first).Error; err != nil {
		t.Fatalf("create first active run: %v", err)
	}
	duplicate := &model.QQGovernanceReconcileRun{
		GroupID: 778899, ActiveKey: "group:778899", Status: model.QQGovernanceRunRunning,
		StartedAt: &now,
	}
	if err := db.Create(duplicate).Error; err == nil {
		t.Fatalf("expected duplicate active run to be rejected")
	}

	// A second group can still run concurrently.
	other := &model.QQGovernanceReconcileRun{
		GroupID: 778900, ActiveKey: "group:778900", Status: model.QQGovernanceRunRunning,
		StartedAt: &now,
	}
	if err := db.Create(other).Error; err != nil {
		t.Fatalf("create active run for second group: %v", err)
	}
}

func TestQQGovernanceComputeReconcileBatchCompletesStuckRun(t *testing.T) {
	db := newServiceTestDB(t, "qq_governance_reconcile_compute_batch",
		&model.QQGroupGovernancePolicy{}, &model.QQGovernanceActionTask{},
		&model.QQGovernanceReconcileRun{}, &model.QQGovernanceReconcileMember{},
		&model.QQGroupMemberState{}, &model.QQGroupRuntimeSnapshot{},
		&model.QQGovernanceReview{},
	)
	const groupID int64 = 778899
	const total = 215
	if err := db.Create(&model.QQGroupGovernancePolicy{GroupID: groupID, Enabled: true}).Error; err != nil {
		t.Fatalf("create policy: %v", err)
	}
	startedAt := time.Date(2026, 7, 24, 12, 0, 0, 0, time.UTC)
	run := &model.QQGovernanceReconcileRun{
		GroupID:        groupID,
		ActiveKey:      fmt.Sprintf("group:%d", groupID),
		Status:         model.QQGovernanceRunRunning,
		ExpectedCount:  total,
		ProcessedCount: 200,
		StartedAt:      &startedAt,
	}
	if err := db.Create(run).Error; err != nil {
		t.Fatalf("create run: %v", err)
	}
	members := make([]model.QQGovernanceReconcileMember, 0, total)
	for i := range total {
		members = append(members, model.QQGovernanceReconcileMember{
			RunID:   run.ID,
			GroupID: groupID,
			QQ:      int64(100000 + i),
			Status:  model.QQGovernanceRunMemberDone,
		})
	}
	if err := db.CreateInBatches(members, 100).Error; err != nil {
		t.Fatalf("seed members: %v", err)
	}
	now := time.Date(2026, 7, 24, 12, 30, 0, 0, time.UTC)
	payload, _ := json.Marshal(qqGovernanceActionPayload{RunID: run.ID, Batch: 4})
	task := &model.QQGovernanceActionTask{
		ActionType: model.QQGovernanceActionComputeBatch, RunID: run.ID,
		IdempotencyKey: fmt.Sprintf("compute:%d:%d", run.ID, 4), GroupID: groupID,
		PayloadJSON: string(payload), Status: model.QQGovernanceActionRunning,
		Priority: 40, RunAfter: now, Source: "reconcile", LeaseToken: "lease-token",
		ClaimedAt:      &now,
		LeaseExpiresAt: &now,
	}
	if err := db.Create(task).Error; err != nil {
		t.Fatalf("create task: %v", err)
	}

	svc := NewQQGovernanceServiceWithRepository(repository.NewQQGovernanceRepositoryWithDB(db))
	svc.now = func() time.Time { return now }

	if err := svc.computeReconcileBatch(t.Context(), task, &model.QQGroupGovernancePolicy{GroupID: groupID, Enabled: true}, run.ID, 4); err != nil {
		t.Fatalf("compute batch: %v", err)
	}

	var refreshed model.QQGovernanceReconcileRun
	if err := db.First(&refreshed, run.ID).Error; err != nil {
		t.Fatalf("reload run: %v", err)
	}
	if refreshed.Status != model.QQGovernanceRunCompleted {
		t.Fatalf("run status = %q, want completed", refreshed.Status)
	}
	if refreshed.ActiveKey != "" {
		t.Fatalf("run active_key = %q, want empty", refreshed.ActiveKey)
	}
	if refreshed.ProcessedCount != total {
		t.Fatalf("run processed_count = %d, want %d", refreshed.ProcessedCount, total)
	}
	if refreshed.CompletedAt == nil || !refreshed.CompletedAt.Equal(now) {
		t.Fatalf("run completed_at = %v, want %v", refreshed.CompletedAt, now)
	}

	var refreshedTask model.QQGovernanceActionTask
	if err := db.First(&refreshedTask, task.ID).Error; err != nil {
		t.Fatalf("reload task: %v", err)
	}
	if refreshedTask.Status != model.QQGovernanceActionSucceeded {
		t.Fatalf("task status = %q, want succeeded", refreshedTask.Status)
	}
}

func TestQQGovernanceEnqueueGroupNotificationsCreatesOneTaskPerGroup(t *testing.T) {
	db := newServiceTestDB(t, "qq_governance_enqueue_notify",
		&model.QQGovernanceActionTask{}, &model.QQGovernanceActionLog{},
	)
	svc := NewQQGovernanceServiceWithRepository(repository.NewQQGovernanceRepositoryWithDB(db))
	now := time.Date(2026, 7, 26, 12, 0, 0, 0, time.UTC)
	svc.now = func() time.Time { return now }

	if err := svc.EnqueueGroupNotifications([]int64{111, 222, 333}, "hello"); err != nil {
		t.Fatalf("enqueue: %v", err)
	}

	var tasks []model.QQGovernanceActionTask
	if err := db.Order("group_id ASC").Find(&tasks).Error; err != nil {
		t.Fatalf("list tasks: %v", err)
	}
	if len(tasks) != 3 {
		t.Fatalf("task count = %d, want 3", len(tasks))
	}
	for i, want := range []int64{111, 222, 333} {
		if tasks[i].GroupID != want {
			t.Fatalf("tasks[%d].GroupID = %d, want %d", i, tasks[i].GroupID, want)
		}
		if tasks[i].ActionType != model.QQGovernanceActionNotify {
			t.Fatalf("tasks[%d].ActionType = %q, want notify", i, tasks[i].ActionType)
		}
		if tasks[i].QQ != 0 {
			t.Fatalf("tasks[%d].QQ = %d, want 0", i, tasks[i].QQ)
		}
		if tasks[i].Source != "webhook" {
			t.Fatalf("tasks[%d].Source = %q, want webhook", i, tasks[i].Source)
		}
		var payload qqGovernanceActionPayload
		if err := json.Unmarshal([]byte(tasks[i].PayloadJSON), &payload); err != nil {
			t.Fatalf("tasks[%d] payload: %v", i, err)
		}
		if payload.Message != "hello" {
			t.Fatalf("tasks[%d].Message = %q, want hello", i, payload.Message)
		}
	}
}

func TestQQGovernanceEnqueueGroupNotificationsRejectsInvalidInput(t *testing.T) {
	db := newServiceTestDB(t, "qq_governance_enqueue_notify_invalid",
		&model.QQGovernanceActionTask{},
	)
	svc := NewQQGovernanceServiceWithRepository(repository.NewQQGovernanceRepositoryWithDB(db))

	cases := []struct {
		name     string
		ids      []int64
		content  string
		wantSub  string
	}{
		{name: "empty_content", ids: []int64{111}, content: "  ", wantSub: "通知内容"},
		{name: "empty_ids", ids: nil, content: "hi", wantSub: "目标群号"},
		{name: "zero_id", ids: []int64{0}, content: "hi", wantSub: "正数"},
		{name: "duplicate_id", ids: []int64{111, 111}, content: "hi", wantSub: "重复"},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			err := svc.EnqueueGroupNotifications(tc.ids, tc.content)
			if err == nil || !containsSubstr(err.Error(), tc.wantSub) {
				t.Fatalf("expected error containing %q, got %v", tc.wantSub, err)
			}
		})
	}
}

func containsSubstr(s, sub string) bool {
	return len(s) >= len(sub) && (s == sub || indexOf(s, sub) >= 0)
}

func indexOf(s, sub string) int {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}

func TestQQGovernanceNotifyParamsMapToSendGroupMsg(t *testing.T) {
	payload, _ := json.Marshal(qqGovernanceActionPayload{Message: "hello"})
	task := &model.QQGovernanceActionTask{
		ActionType:  model.QQGovernanceActionNotify,
		GroupID:     123456,
		PayloadJSON: string(payload),
	}
	params, err := paramsForQQGovernanceAction(task)
	if err != nil {
		t.Fatalf("params: %v", err)
	}
	if got := params["group_id"]; got != int64(123456) {
		t.Fatalf("group_id = %v, want 123456", got)
	}
	if got := params["message"]; got != "hello" {
		t.Fatalf("message = %v, want hello", got)
	}
	if name := oneBotActionName(model.QQGovernanceActionNotify); name != "send_group_msg" {
		t.Fatalf("oneBotActionName = %q, want send_group_msg", name)
	}
}

func TestQQGovernanceNotifyActionSkipsPolicyCheck(t *testing.T) {
	db := newServiceTestDB(t, "qq_governance_notify_valid",
		&model.QQGroupGovernancePolicy{}, &model.QQGroupMemberState{},
		&model.QQGovernanceActionTask{},
	)
	svc := NewQQGovernanceServiceWithRepository(repository.NewQQGovernanceRepositoryWithDB(db))

	// No policy, no member state — should still be valid for notify.
	task := &model.QQGovernanceActionTask{
		ActionType: model.QQGovernanceActionNotify,
		GroupID:    123456,
	}
	valid, reason, err := svc.actionStillValid(task)
	if err != nil {
		t.Fatalf("actionStillValid: %v", err)
	}
	if !valid {
		t.Fatalf("notify task should remain valid without policy, reason = %q", reason)
	}

	// Non-notify task should be cancelled without policy.
	other := &model.QQGovernanceActionTask{
		ActionType: model.QQGovernanceActionSetCard,
		GroupID:    123456,
		QQ:         999,
	}
	valid, _, err = svc.actionStillValid(other)
	if err != nil {
		t.Fatalf("actionStillValid(set_card): %v", err)
	}
	if valid {
		t.Fatalf("set_card task should be invalid without policy")
	}

	// Notify with GroupID=0 is invalid.
	invalid := &model.QQGovernanceActionTask{ActionType: model.QQGovernanceActionNotify}
	if valid, _, err := svc.actionStillValid(invalid); err != nil || valid {
		t.Fatalf("notify without group id should be invalid (valid=%v err=%v)", valid, err)
	}

	// Ensure other errors package is still used elsewhere — keep import valid.
	_ = errors.New("noop")
}

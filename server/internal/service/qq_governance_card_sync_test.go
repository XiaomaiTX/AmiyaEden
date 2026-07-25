package service

import (
	"amiya-eden/global"
	"amiya-eden/internal/model"
	"amiya-eden/internal/repository"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func newQQGovernanceCardSyncDB(t *testing.T, prefix string) *gorm.DB {
	t.Helper()
	dsn := fmt.Sprintf("file:%s_%s_%d_%d?mode=memory&cache=shared", prefix, t.Name(), time.Now().UnixNano(), time.Now().UnixMicro())
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(
		&model.SystemConfig{},
		&model.User{}, &model.EveCharacter{}, &model.UserRole{},
		&model.QQGroupGovernancePolicy{}, &model.QQGovernanceEvent{},
		&model.QQGroupMemberState{},
		&model.QQGovernanceActionTask{}, &model.QQGovernanceActionLog{},
		&model.QQGovernanceReconcileRun{}, &model.QQGovernanceReconcileMember{},
		&model.QQGovernanceReview{}, &model.QQGovernanceAlert{},
		&model.QQGroupRuntimeSnapshot{},
	); err != nil {
		t.Fatalf("auto migrate: %v", err)
	}
	prevDB := global.DB
	global.DB = db
	t.Cleanup(func() { global.DB = prevDB })
	return db
}

// seedCardSyncUserAndPolicy creates a Seat user, character, role, and a group
// policy with card_sync enabled, returning the policy and matching QQ.
func seedCardSyncUserAndPolicy(t *testing.T, db *gorm.DB, cardSyncEnabled bool) (*model.QQGroupGovernancePolicy, int64) {
	t.Helper()
	const groupID int64 = 778899
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
	policy := &model.QQGroupGovernancePolicy{GroupID: groupID, Enabled: true, AllowedCorporationIDsJSON: string(corps), AllowedRoleCodesJSON: string(roles), CardTemplate: "{nickname}-{primary_character_name}", CardSyncEnabled: cardSyncEnabled}
	if err := db.Create(policy).Error; err != nil {
		t.Fatalf("create policy: %v", err)
	}
	return policy, 123456
}

func TestQQGovernanceReconcileSnapshotCapturesCurrentCard(t *testing.T) {
	db := newQQGovernanceCardSyncDB(t, "qq_governance_card_snapshot")
	cleanup := setupQQGovernanceRedis(t, db)
	defer cleanup()
	policy, _ := seedCardSyncUserAndPolicy(t, db, true)
	svc := NewQQGovernanceServiceWithRepository(repository.NewQQGovernanceRepositoryWithDB(db))
	runStart := time.Date(2026, 7, 26, 12, 0, 0, 0, time.UTC)
	run := &model.QQGovernanceReconcileRun{GroupID: policy.GroupID, ActiveKey: fmt.Sprintf("group:%d", policy.GroupID), Status: model.QQGovernanceRunRunning, ExpectedCount: 1, StartedAt: &runStart}
	if err := db.Create(run).Error; err != nil {
		t.Fatalf("create run: %v", err)
	}
	task := &model.QQGovernanceActionTask{ActionType: model.QQGovernanceActionSnapshot, RunID: run.ID, GroupID: policy.GroupID, IdempotencyKey: "snapshot-card", Status: model.QQGovernanceActionRunning, Priority: 20, RunAfter: runStart, LeaseToken: "lease"}
	if err := db.Create(task).Error; err != nil {
		t.Fatalf("create task: %v", err)
	}
	executor := &qqGovernanceReconcileTestExecutor{membersRaw: json.RawMessage(`[{"user_id":123456,"role":"member","card":"旧名片"}]`)}
	svc.now = func() time.Time { return runStart }
	svc.SetOneBotActionExecutor(executor)

	if err := svc.captureReconcileSnapshot(t.Context(), task, policy, run.ID); err != nil {
		t.Fatalf("capture snapshot: %v", err)
	}

	var member model.QQGovernanceReconcileMember
	if err := db.Where("qq = ?", 123456).First(&member).Error; err != nil {
		t.Fatalf("load member: %v", err)
	}
	if member.Card != "旧名片" {
		t.Fatalf("member.Card = %q, want 旧名片", member.Card)
	}
}

func TestQQGovernanceReconcileEnqueuesSetCardWhenCardDiffers(t *testing.T) {
	db := newQQGovernanceCardSyncDB(t, "qq_governance_card_diff")
	policy, qq := seedCardSyncUserAndPolicy(t, db, true)
	svc := NewQQGovernanceServiceWithRepository(repository.NewQQGovernanceRepositoryWithDB(db))
	now := time.Date(2026, 7, 26, 12, 0, 0, 0, time.UTC)
	svc.now = func() time.Time { return now }
	if err := svc.reconcileMember(t.Context(), policy, policy.GroupID, qq, "scan", "旧的群名片"); err != nil {
		t.Fatalf("reconcile: %v", err)
	}
	var tasks []model.QQGovernanceActionTask
	if err := db.Where("action_type = ?", model.QQGovernanceActionSetCard).Find(&tasks).Error; err != nil {
		t.Fatalf("list tasks: %v", err)
	}
	if len(tasks) != 1 {
		t.Fatalf("set_card task count = %d, want 1", len(tasks))
	}
	var payload qqGovernanceActionPayload
	if err := json.Unmarshal([]byte(tasks[0].PayloadJSON), &payload); err != nil {
		t.Fatalf("decode payload: %v", err)
	}
	if payload.Card != "Amiya-Amiya Prime" {
		t.Fatalf("payload.Card = %q, want Amiya-Amiya Prime", payload.Card)
	}
	expectedKey := fmt.Sprintf("card:%d:%d:%d", policy.GroupID, qq, tasks[0].TargetVersion)
	if tasks[0].IdempotencyKey != expectedKey {
		t.Fatalf("idempotency key = %q, want %q", tasks[0].IdempotencyKey, expectedKey)
	}
}

func TestQQGovernanceReconcileSkipsSetCardWhenCardMatches(t *testing.T) {
	db := newQQGovernanceCardSyncDB(t, "qq_governance_card_match")
	policy, qq := seedCardSyncUserAndPolicy(t, db, true)
	svc := NewQQGovernanceServiceWithRepository(repository.NewQQGovernanceRepositoryWithDB(db))
	svc.now = func() time.Time { return time.Date(2026, 7, 26, 12, 0, 0, 0, time.UTC) }
	if err := svc.reconcileMember(t.Context(), policy, policy.GroupID, qq, "scan", "Amiya-Amiya Prime"); err != nil {
		t.Fatalf("reconcile: %v", err)
	}
	var count int64
	if err := db.Model(&model.QQGovernanceActionTask{}).Where("action_type = ?", model.QQGovernanceActionSetCard).Count(&count).Error; err != nil {
		t.Fatalf("count tasks: %v", err)
	}
	if count != 0 {
		t.Fatalf("set_card task count = %d, want 0", count)
	}
}

func TestQQGovernanceReconcileSkipsSetCardWhenToggleOff(t *testing.T) {
	db := newQQGovernanceCardSyncDB(t, "qq_governance_card_disabled")
	policy, qq := seedCardSyncUserAndPolicy(t, db, false)
	svc := NewQQGovernanceServiceWithRepository(repository.NewQQGovernanceRepositoryWithDB(db))
	svc.now = func() time.Time { return time.Date(2026, 7, 26, 12, 0, 0, 0, time.UTC) }
	if err := svc.reconcileMember(t.Context(), policy, policy.GroupID, qq, "scan", "旧的群名片"); err != nil {
		t.Fatalf("reconcile: %v", err)
	}
	var count int64
	if err := db.Model(&model.QQGovernanceActionTask{}).Where("action_type = ?", model.QQGovernanceActionSetCard).Count(&count).Error; err != nil {
		t.Fatalf("count tasks: %v", err)
	}
	if count != 0 {
		t.Fatalf("set_card task count = %d, want 0", count)
	}
}

func TestQQGovernanceReconcileSkipsSetCardForUnmatchedMember(t *testing.T) {
	db := newQQGovernanceCardSyncDB(t, "qq_governance_card_unmatched")
	// Policy with role/corp restrictions but no matching Seat user — unmatched.
	policy := &model.QQGroupGovernancePolicy{GroupID: 778899, Enabled: true, AllowedCorporationIDsJSON: "[1001]", AllowedRoleCodesJSON: "[\"admin\"]", CardTemplate: "{nickname}", CardSyncEnabled: true}
	if err := db.Create(policy).Error; err != nil {
		t.Fatalf("create policy: %v", err)
	}
	svc := NewQQGovernanceServiceWithRepository(repository.NewQQGovernanceRepositoryWithDB(db))
	svc.now = func() time.Time { return time.Date(2026, 7, 26, 12, 0, 0, 0, time.UTC) }
	if err := svc.reconcileMember(t.Context(), policy, policy.GroupID, 999999, "scan", "irrelevant"); err != nil {
		t.Fatalf("reconcile: %v", err)
	}
	var count int64
	if err := db.Model(&model.QQGovernanceActionTask{}).Where("action_type = ?", model.QQGovernanceActionSetCard).Count(&count).Error; err != nil {
		t.Fatalf("count tasks: %v", err)
	}
	if count != 0 {
		t.Fatalf("set_card task count = %d, want 0", count)
	}
}

func TestQQGovernanceReconcileSameVersionDeduplicatesSetCard(t *testing.T) {
	db := newQQGovernanceCardSyncDB(t, "qq_governance_card_dedupe")
	policy, qq := seedCardSyncUserAndPolicy(t, db, true)
	now := time.Date(2026, 7, 26, 12, 0, 0, 0, time.UTC)
	// Seed the member state at version 3; reconcileMember would normally bump it
	// on every call. To exercise the idempotency-key dedupe we instead call the
	// underlying enqueue path directly with the same version twice, mirroring how
	// a retry inside a single batch would attempt to re-insert the same task.
	state := &model.QQGroupMemberState{GroupID: policy.GroupID, QQ: qq, Status: model.QQGovernanceMemberValid, TargetCard: "Amiya-Amiya Prime", Version: 3, LastCheckedAt: now}
	if err := db.Create(state).Error; err != nil {
		t.Fatalf("seed state: %v", err)
	}
	repo := repository.NewQQGovernanceRepositoryWithDB(db)
	payload, _ := json.Marshal(qqGovernanceActionPayload{Card: "Amiya-Amiya Prime"})
	first := &model.QQGovernanceActionTask{ActionType: model.QQGovernanceActionSetCard, IdempotencyKey: fmt.Sprintf("card:%d:%d:%d", policy.GroupID, qq, state.Version), GroupID: policy.GroupID, QQ: qq, TargetVersion: state.Version, PayloadJSON: string(payload), Status: model.QQGovernanceActionPending, Priority: 30, RunAfter: now, Source: "scan"}
	second := &model.QQGovernanceActionTask{ActionType: model.QQGovernanceActionSetCard, IdempotencyKey: fmt.Sprintf("card:%d:%d:%d", policy.GroupID, qq, state.Version), GroupID: policy.GroupID, QQ: qq, TargetVersion: state.Version, PayloadJSON: string(payload), Status: model.QQGovernanceActionPending, Priority: 30, RunAfter: now, Source: "scan"}
	if _, err := repo.CreateActionTaskIfAbsent(first); err != nil {
		t.Fatalf("insert first: %v", err)
	}
	if _, err := repo.CreateActionTaskIfAbsent(second); err != nil {
		t.Fatalf("insert second: %v", err)
	}
	var count int64
	if err := db.Model(&model.QQGovernanceActionTask{}).Where("action_type = ?", model.QQGovernanceActionSetCard).Count(&count).Error; err != nil {
		t.Fatalf("count tasks: %v", err)
	}
	if count != 1 {
		t.Fatalf("set_card task count = %d, want 1", count)
	}
}

func TestQQGovernanceSavePolicyDisablingCardSyncCancelsPending(t *testing.T) {
	db := newQQGovernanceCardSyncDB(t, "qq_governance_card_save_cancel")
	policy, _ := seedCardSyncUserAndPolicy(t, db, true)
	now := time.Date(2026, 7, 26, 12, 0, 0, 0, time.UTC)
	queued := &model.QQGovernanceActionTask{ActionType: model.QQGovernanceActionSetCard, IdempotencyKey: fmt.Sprintf("card:%d:%d:%d", policy.GroupID, 123456, 1), GroupID: policy.GroupID, QQ: 123456, TargetVersion: 1, Status: model.QQGovernanceActionPending, Priority: 30, RunAfter: now, Source: "scan"}
	if err := db.Create(queued).Error; err != nil {
		t.Fatalf("seed queued task: %v", err)
	}
	svc := NewQQGovernanceServiceWithRepository(repository.NewQQGovernanceRepositoryWithDB(db))
	if _, err := svc.SavePolicy(t.Context(), QQGovernancePolicyInput{GroupID: policy.GroupID, Enabled: true, AllowedCorporationIDs: []int64{1001}, AllowedRoleCodes: []string{model.RoleAdmin}, MemberViolationPolicy: model.QQGovernanceViolationReview, CardTemplate: "{nickname}", CardSyncEnabled: false}, 1); err != nil {
		t.Fatalf("save policy: %v", err)
	}
	var refreshed model.QQGovernanceActionTask
	if err := db.First(&refreshed, queued.ID).Error; err != nil {
		t.Fatalf("reload task: %v", err)
	}
	if refreshed.Status != model.QQGovernanceActionCancelled {
		t.Fatalf("task status = %q, want %q", refreshed.Status, model.QQGovernanceActionCancelled)
	}
}

func TestQQGovernanceActionStillValidRejectsSetCardWhenToggleOff(t *testing.T) {
	db := newQQGovernanceCardSyncDB(t, "qq_governance_card_valid_toggle")
	policy, qq := seedCardSyncUserAndPolicy(t, db, false)
	state := &model.QQGroupMemberState{GroupID: policy.GroupID, QQ: qq, Status: model.QQGovernanceMemberValid, TargetCard: "Amiya-Amiya Prime", Version: 1, LastCheckedAt: time.Now()}
	if err := db.Create(state).Error; err != nil {
		t.Fatalf("create state: %v", err)
	}
	svc := NewQQGovernanceServiceWithRepository(repository.NewQQGovernanceRepositoryWithDB(db))
	task := &model.QQGovernanceActionTask{ActionType: model.QQGovernanceActionSetCard, GroupID: policy.GroupID, QQ: qq, TargetVersion: 1}
	valid, reason, err := svc.actionStillValid(task)
	if err != nil {
		t.Fatalf("actionStillValid: %v", err)
	}
	if valid {
		t.Fatalf("expected set_card to be invalid when toggle is off")
	}
	if reason != "群名片同步已关闭" {
		t.Fatalf("reason = %q", reason)
	}
}

func TestQQGovernanceWorkerUpdatesLastCardUpdatedAtOnSuccess(t *testing.T) {
	db := newQQGovernanceCardSyncDB(t, "qq_governance_card_mark_updated")
	cleanup := setupQQGovernanceRedis(t, db)
	defer cleanup()
	policy, qq := seedCardSyncUserAndPolicy(t, db, true)
	past := time.Date(2026, 7, 26, 11, 0, 0, 0, time.UTC)
	state := &model.QQGroupMemberState{GroupID: policy.GroupID, QQ: qq, Status: model.QQGovernanceMemberValid, TargetCard: "Amiya-Amiya Prime", Version: 5, LastCheckedAt: past}
	if err := db.Create(state).Error; err != nil {
		t.Fatalf("create state: %v", err)
	}
	payload, _ := json.Marshal(qqGovernanceActionPayload{Card: "Amiya-Amiya Prime"})
	task := &model.QQGovernanceActionTask{ActionType: model.QQGovernanceActionSetCard, IdempotencyKey: "card-success", GroupID: policy.GroupID, QQ: qq, TargetVersion: 5, PayloadJSON: string(payload), Status: model.QQGovernanceActionPending, Priority: 30, RunAfter: past}
	if err := db.Create(task).Error; err != nil {
		t.Fatalf("create task: %v", err)
	}
	svc := NewQQGovernanceServiceWithRepository(repository.NewQQGovernanceRepositoryWithDB(db))
	now := time.Date(2026, 7, 26, 12, 0, 0, 0, time.UTC)
	svc.now = func() time.Time { return now }
	svc.SetOneBotActionExecutor(&qqGovernanceReconcileTestExecutor{})

	if err := svc.RunActionWorkerOnce(t.Context()); err != nil {
		t.Fatalf("run worker: %v", err)
	}

	var refreshed model.QQGroupMemberState
	if err := db.Where("group_id = ? AND qq = ?", policy.GroupID, qq).First(&refreshed).Error; err != nil {
		t.Fatalf("reload state: %v", err)
	}
	if refreshed.LastCardUpdatedAt == nil {
		t.Fatalf("last_card_updated_at was not updated")
	}
	if !refreshed.LastCardUpdatedAt.Equal(now) {
		t.Fatalf("last_card_updated_at = %v, want %v", *refreshed.LastCardUpdatedAt, now)
	}
}

func TestQQGovernanceGroupIncreaseRespectsCardSyncToggle(t *testing.T) {
	cases := []struct {
		name      string
		cardSync  bool
		wantTask  bool
	}{
		{name: "off", cardSync: false, wantTask: false},
		{name: "on", cardSync: true, wantTask: true},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			db := newQQGovernanceCardSyncDB(t, "qq_governance_increase_"+tc.name)
			policy, qq := seedCardSyncUserAndPolicy(t, db, tc.cardSync)
			svc := NewQQGovernanceServiceWithRepository(repository.NewQQGovernanceRepositoryWithDB(db))
			now := time.Date(2026, 7, 26, 12, 0, 0, 0, time.UTC)
			svc.now = func() time.Time { return now }
			event := QQGovernanceInboundEvent{EventKey: fmt.Sprintf("notice/group_increase:%d:%d:%d:%d", policy.GroupID, qq, 9999, 1), EventType: "notice/group_increase", GroupID: policy.GroupID, QQ: qq}
			if err := svc.HandleOneBotEvent(t.Context(), event); err != nil {
				t.Fatalf("handle event: %v", err)
			}
			var count int64
			if err := db.Model(&model.QQGovernanceActionTask{}).Where("action_type = ?", model.QQGovernanceActionSetCard).Count(&count).Error; err != nil {
				t.Fatalf("count tasks: %v", err)
			}
			if tc.wantTask && count != 1 {
				t.Fatalf("set_card task count = %d, want 1", count)
			}
			if !tc.wantTask && count != 0 {
				t.Fatalf("set_card task count = %d, want 0", count)
			}
		})
	}
}

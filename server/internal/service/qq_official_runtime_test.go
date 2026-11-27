package service

import (
	"amiya-eden/internal/model"
	"amiya-eden/internal/service/internal/qqapi"
	"bytes"
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

const qqDeploymentKeyFixture = "test-deployment-secret-not-for-production"

func qqActive(t *testing.T, rt *QQOfficialRuntime) *QQOfficialService {
	t.Helper()
	s, err := rt.Active(t.Context(), nil)
	if err != nil || s == nil {
		t.Fatalf("no active runtime: %v", err)
	}
	return s
}

func TestQQOfficialRuntimeEncryptsAndKeepsSecretsAcrossSettingsChanges(t *testing.T) {
	_, db, _ := qqFixture(t)
	ctx := t.Context()
	rt := NewQQOfficialRuntime(db, qqDeploymentKeyFixture)
	if view, err := rt.Status(ctx); err != nil || view.Configured || view.HasAppSecret || view.Enabled || view.ApprovalWritesEnabled {
		t.Fatalf("unexpected unconfigured view: %+v %v", view, err)
	}
	id, secret := "4123", "test-qq-secret-12345"
	if _, err := rt.UpdateSettings(ctx, 7, QQOfficialSettingsUpdate{AppID: &id}); !errors.Is(err, ErrQQOfficialConflict) {
		t.Fatalf("incomplete setup accepted: %v", err)
	}
	if _, err := NewQQOfficialRuntime(db, "change_me_in_production").UpdateSettings(ctx, 7, QQOfficialSettingsUpdate{AppID: &id, AppSecret: &secret}); !errors.Is(err, ErrQQOfficialUnavailable) {
		t.Fatalf("public example key accepted: %v", err)
	}
	view, err := rt.UpdateSettings(ctx, 7, QQOfficialSettingsUpdate{AppID: &id, AppSecret: &secret})
	if err != nil || !view.Configured || !view.HasAppSecret || view.Enabled || view.ApprovalWritesEnabled || view.AppID != id {
		t.Fatalf("unexpected bootstrap view: %+v %v", view, err)
	}
	row, err := rt.installation(ctx, nil)
	if err != nil || len(row.SecretCiphertext) == 0 || bytes.Contains(row.SecretCiphertext, []byte(secret)) {
		t.Fatalf("credential storage is not encrypted: %v", err)
	}
	copied := row
	copied.AppID = "89765"
	if _, err := rt.decrypt(copied); !errors.Is(err, ErrQQOfficialUnavailable) {
		t.Fatal("ciphertext was not authenticated to the AppID")
	}
	wrongKey := NewQQOfficialRuntime(db, "another-test-key")
	if view, err := wrongKey.Status(ctx); err != nil || view.Configured || !view.HasAppSecret || view.TokenState != "credentials_unavailable" || view.AppID != id {
		t.Fatalf("wrong-key status: %+v %v", view, err)
	}
	if s, err := wrongKey.Active(ctx, nil); s != nil || !errors.Is(err, ErrQQOfficialUnavailable) {
		t.Fatalf("wrong key resolved active: %t %v", s != nil, err)
	}
	on := true
	if _, err := rt.UpdateSettings(ctx, 7, QQOfficialSettingsUpdate{Enabled: &on, ApprovalWritesEnabled: &on}); err != nil {
		t.Fatal(err)
	}
	if _, err := rt.UpdateSettings(ctx, 7, QQOfficialSettingsUpdate{Paused: &on}); err != nil {
		t.Fatal(err)
	}
	if _, err := rt.UpdateSettings(ctx, 7, QQOfficialSettingsUpdate{AppID: &id, AppSecret: &secret}); err != nil {
		t.Fatal(err)
	}
	peer := NewQQOfficialRuntime(db, qqDeploymentKeyFixture)
	view, err = peer.Status(ctx)
	if err != nil || !view.Paused || !view.Enabled || !view.ApprovalWritesEnabled {
		t.Fatalf("partial settings were overwritten: %+v %v", view, err)
	}
	fresh, err := peer.installation(ctx, nil)
	if err != nil || !bytes.Equal(fresh.SecretCiphertext, row.SecretCiphertext) {
		t.Fatal("omitting or resubmitting same secret changed ciphertext")
	}
	if view, err := wrongKey.Status(ctx); err != nil || view.Configured || !view.ApprovalWritesEnabled || !view.Enabled || !view.Paused {
		t.Fatalf("cannot see persisted switches for recovery: %+v %v", view, err)
	}
	recovered, err := wrongKey.UpdateSettings(ctx, 7, QQOfficialSettingsUpdate{AppSecret: &secret})
	if err != nil || !recovered.Configured || recovered.Enabled || recovered.ApprovalWritesEnabled {
		t.Fatalf("re-entering credentials after key rotation was unsafe: %+v %v", recovered, err)
	}
	if s, err := peer.Active(ctx, nil); s != nil || !errors.Is(err, ErrQQOfficialUnavailable) {
		t.Fatalf("old deployment key still resolves cached credentials: %t %v", s != nil, err)
	}
}

func TestQQOfficialRuntimeRotatesAndFencesOtherInstances(t *testing.T) {
	_, db, _ := qqFixture(t)
	checkQQRotation(t, db)
}

func checkQQRotation(t *testing.T, db *gorm.DB) {
	t.Helper()
	ctx := t.Context()
	a, b := NewQQOfficialRuntime(db, qqDeploymentKeyFixture), NewQQOfficialRuntime(db, qqDeploymentKeyFixture)
	id, otherID, secret, newSecret, on := "7890", "7891", "original-test-secret", "rotated-test-secret", true
	if _, err := a.UpdateSettings(ctx, 7, QQOfficialSettingsUpdate{AppID: &id, AppSecret: &secret, Enabled: &on}); !errors.Is(err, ErrQQOfficialConflict) {
		t.Fatalf("credential update can enable direct sends: %v", err)
	}
	if _, err := a.UpdateSettings(ctx, 7, QQOfficialSettingsUpdate{AppID: &id, AppSecret: &secret}); err != nil {
		t.Fatal(err)
	}
	if _, err := a.UpdateSettings(ctx, 7, QQOfficialSettingsUpdate{Enabled: &on, ApprovalWritesEnabled: &on}); err != nil {
		t.Fatal(err)
	}
	stale := qqActive(t, b)
	group := qqGroupFixture(t, stale, 1)
	first, queued := qqSubmitTest(t, stale, group), qqSubmitTest(t, stale, group)
	permit, err := stale.claim(ctx, qqToken(time.Now()), 1)
	if err != nil || permit == nil || permit.attempt.TaskID == "" {
		t.Fatalf("no real lease: %+v %v", permit, err)
	}
	if _, err := a.UpdateSettings(ctx, 7, QQOfficialSettingsUpdate{AppID: &otherID, AppSecret: &newSecret}); !errors.Is(err, ErrQQOfficialConflict) {
		t.Fatalf("rotation did not respect active lease: %v", err)
	}
	if ok, err := stale.finish(ctx, permit.attempt.ID, qqapi.Result{MessageID: "confirmed"}, qqapi.Token{}, nil); err != nil || !ok {
		t.Fatalf("finish %t %v", ok, err)
	}
	rotated, err := a.UpdateSettings(ctx, 7, QQOfficialSettingsUpdate{AppID: &otherID, AppSecret: &newSecret})
	if err != nil || rotated.AppID != otherID || rotated.Enabled || rotated.ApprovalWritesEnabled || rotated.QueueDepth != 0 {
		t.Fatalf("rotation unsafe: %+v %v", rotated, err)
	}
	if got, err := stale.claim(ctx, qqToken(time.Now()), 1); got != nil || !errors.Is(err, ErrQQOfficialUnavailable) {
		t.Fatalf("stale worker still has access: %+v %v", got, err)
	}
	if got, err := stale.TestGroup(ctx, 7, group.ID, uuid.NewString()); got.ID != "" || !errors.Is(err, ErrQQOfficialUnavailable) {
		t.Fatalf("stale API can create work: %+v %v", got, err)
	}
	if first.ID != permit.attempt.TaskID {
		first, queued = queued, first
	}
	for _, expected := range []struct{ id, status string }{{first.ID, model.QQSucceeded}, {queued.ID, model.QQCancelled}} {
		row, _, err := stale.Task(ctx, expected.id)
		if err != nil || row.Status != expected.status || row.Text != "" {
			t.Fatalf("inconsistent old task %s: %s %v", expected.id, row.Status, err)
		}
	}
	current := qqActive(t, b)
	if current.appID != otherID || current == stale {
		t.Fatal("peer did not resolve new AppID")
	}
	if _, err := a.UpdateSettings(ctx, 7, QQOfficialSettingsUpdate{AppID: &id, AppSecret: &newSecret}); err != nil {
		t.Fatal(err)
	}
	view, err := b.Status(ctx)
	if err != nil || view.QueueDepth != 0 || view.Enabled {
		t.Fatalf("switching back can replay prior work: %+v %v", view, err)
	}
}

func TestQQOfficialRuntimeDisablingApprovalCancelsQueuedAndFencesRetries(t *testing.T) {
	_, db, _ := qqFixture(t)
	ctx := t.Context()
	rt := NewQQOfficialRuntime(db, qqDeploymentKeyFixture)
	id, secret, on, off := "4555", "approval-test-secret", true, false
	if _, err := rt.UpdateSettings(ctx, 7, QQOfficialSettingsUpdate{AppID: &id, AppSecret: &secret}); err != nil {
		t.Fatal(err)
	}
	if _, err := rt.UpdateSettings(ctx, 7, QQOfficialSettingsUpdate{Enabled: &on, ApprovalWritesEnabled: &on}); err != nil {
		t.Fatal(err)
	}
	s := qqActive(t, rt)
	group := qqGroupFixture(t, s, 2)
	group.ApprovalCapability, group.ApprovalEnabled = qqAvailable, true
	if err := s.repo.SaveGroup(ctx, nil, &group); err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&model.User{BaseModel: model.BaseModel{ID: 7}, Nickname: "operator", Status: 1}).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&model.UserRole{UserID: 7, RoleCode: model.RoleSuperAdmin}).Error; err != nil {
		t.Fatal(err)
	}
	ids := make([]string, 0, 2)
	for range 2 {
		key := uuid.NewString()
		request := model.QQOfficialJoinRequest{ID: key, AppID: id, GroupID: group.ID, RequestKey: key, RequestID: key, MemberOpenID: key,
			Status: "pending", ExpiresAt: time.Now().Add(time.Hour)}
		if err := s.repo.SaveJoinRequest(ctx, nil, &request); err != nil {
			t.Fatal(err)
		}
		task, err := s.DecideJoinRequest(ctx, 7, key, "approve", uuid.NewString())
		if err != nil {
			t.Fatal(err)
		}
		ids = append(ids, task.ID)
	}
	permit, err := s.claim(ctx, qqToken(time.Now()), 1)
	if err != nil || permit == nil || permit.attempt.Kind != qqapi.JoinDecision {
		t.Fatalf("decision not admitted: %+v %v", permit, err)
	}
	for _, enabled := range []*bool{&off, &on} {
		if _, err := rt.UpdateSettings(ctx, 7, QQOfficialSettingsUpdate{ApprovalWritesEnabled: enabled}); err != nil {
			t.Fatal(err)
		}
	}
	if ok, err := s.finish(ctx, permit.attempt.ID, qqapi.Result{}, qqapi.Token{}, &qqapi.OperationError{Kind: qqapi.TemporaryNotApplied}); err != nil || !ok {
		t.Fatalf("old approved lease result not recorded: %t %v", ok, err)
	}
	for _, taskID := range ids {
		row, _, err := s.Task(ctx, taskID)
		if err != nil || row.Status != model.QQCancelled || row.Reason != "approval_disabled" {
			t.Fatalf("decision can be retried: %+v %v", row, err)
		}
	}
}

func TestQQOfficialPostgresManagedRuntime(t *testing.T) {
	dsn := os.Getenv("QQ_TEST_POSTGRES_DSN")
	if dsn == "" {
		t.Skip("set QQ_TEST_POSTGRES_DSN to a disposable test database")
	}
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatal(err)
	}
	var name string
	if err = db.Raw("SELECT current_database()").Scan(&name).Error; err != nil || name != "qq_official_test" {
		t.Fatalf("not the disposable test database: %v", err)
	}
	conn, err := db.DB()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = conn.Close() })
	qqMigrateFixture(t, db)
	checkQQRotation(t, db)

	rt := NewQQOfficialRuntime(db, qqDeploymentKeyFixture)
	s := qqActive(t, rt)
	entered, release, done := make(chan struct{}), make(chan struct{}), make(chan error, 1)
	go func() {
		done <- s.locked(t.Context(), nil, func(*gorm.DB, *model.QQOfficialApp, time.Time) error {
			close(entered)
			<-release
			return nil
		})
	}()
	select {
	case <-entered:
	case <-time.After(3 * time.Second):
		close(release)
		t.Fatalf("did not acquire shared configuration row: %v", <-done)
	}
	deadline, cancel := context.WithTimeout(t.Context(), 250*time.Millisecond)
	secret := "candidate-secret"
	_, err = rt.UpdateSettings(deadline, 7, QQOfficialSettingsUpdate{AppSecret: &secret})
	cancel()
	close(release)
	if waitErr := <-done; waitErr != nil {
		t.Fatal(waitErr)
	}
	if !errors.Is(err, ErrQQOfficialUnavailable) {
		t.Fatalf("config mutation did not wait for admission transaction: %v", err)
	}
	if _, err = rt.UpdateSettings(t.Context(), 7, QQOfficialSettingsUpdate{AppSecret: &secret}); err != nil {
		t.Fatalf("config mutation did not recover: %v", err)
	}
}

package service

import (
	"amiya-eden/internal/model"
	"amiya-eden/internal/service/internal/qqapi"
	"context"
	"errors"
	"fmt"
	"os"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"github.com/google/uuid"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type qqFakeClient struct {
	execute func(context.Context, string, qqapi.Request) (qqapi.Result, error)
	refresh func(context.Context) (qqapi.Token, error)
}

func (f qqFakeClient) Refresh(ctx context.Context) (qqapi.Token, error) { return f.refresh(ctx) }
func (f qqFakeClient) Execute(ctx context.Context, token string, req qqapi.Request) (qqapi.Result, error) {
	return f.execute(ctx, token, req)
}

func qqFixture(t *testing.T) (*QQOfficialService, *gorm.DB, *time.Time) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file:"+uuid.NewString()+"?mode=memory&cache=shared"), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatal(err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatal(err)
	}
	sqlDB.SetMaxOpenConns(1)
	t.Cleanup(func() { _ = sqlDB.Close() })
	qqMigrateFixture(t, db)
	now := time.Date(2026, 11, 26, 14, 0, 0, 0, time.UTC)
	svc := NewQQOfficialService(db, "1234", "sample-secret", true)
	svc.now = func() time.Time { return now }
	return svc, db, &now
}

func qqMigrateFixture(t *testing.T, db *gorm.DB) {
	t.Helper()
	models := append(model.QQOfficialModels(), &model.AuditEvent{}, &model.User{}, &model.UserRole{})
	if err := db.AutoMigrate(models...); err != nil {
		t.Fatal(err)
	}
}

func qqEnable(t *testing.T, s *QQOfficialService) {
	t.Helper()
	enabled := true
	if _, err := s.UpdateSettings(t.Context(), 7, QQOfficialSettingsUpdate{Enabled: &enabled}); err != nil {
		t.Fatal(err)
	}
}

func qqGroupFixture(t *testing.T, s *QQOfficialService, idx int) model.QQOfficialGroup {
	t.Helper()
	var group model.QQOfficialGroup
	ctx := t.Context()
	if err := s.locked(ctx, nil, func(tx *gorm.DB, _ *model.QQOfficialApp, now time.Time) error {
		group = qqNewGroup(s.appID, fmt.Sprintf("QQGroup%024d", idx), now)
		group.Present, group.Authorized, group.Alias = true, true, fmt.Sprintf("Group %d", idx)
		return s.repo.SaveGroup(ctx, tx, &group)
	}); err != nil {
		t.Fatal(err)
	}
	return group
}

func qqSubmitTest(t *testing.T, s *QQOfficialService, group model.QQOfficialGroup) model.QQOfficialTask {
	t.Helper()
	task, err := s.TestGroup(t.Context(), 7, group.ID, uuid.NewString())
	if err != nil {
		t.Fatal(err)
	}
	return task
}

func qqToken(now time.Time) qqapi.Token {
	return qqapi.Token{AccessToken: "in-memory", ExpiresAt: now.Add(time.Hour)}
}

func TestQQOfficialTestQueueIdempotencyAndOptOut(t *testing.T) {
	s, _, now := qqFixture(t)
	primary := qqGroupFixture(t, s, 1)
	other := qqGroupFixture(t, s, 2)
	key := uuid.NewString()
	if _, err := s.TestGroup(t.Context(), 7, primary.ID, key); !errors.Is(err, ErrQQOfficialDisabled) {
		t.Fatalf("initial state %v", err)
	}
	qqEnable(t, s)
	one, err := s.TestGroup(t.Context(), 7, primary.ID, key)
	if err != nil {
		t.Fatal(err)
	}
	*now = now.Add(time.Second)
	two, err := s.TestGroup(t.Context(), 7, primary.ID, key)
	if err != nil || two.ID != one.ID || !two.ExpiresAt.Equal(one.ExpiresAt) {
		t.Fatalf("replay %+v %v", two, err)
	}
	if _, err = s.TestGroup(t.Context(), 7, other.ID, key); !errors.Is(err, ErrQQOfficialConflict) {
		t.Fatalf("different target %v", err)
	}
	ctx := t.Context()
	if err = s.locked(ctx, nil, func(tx *gorm.DB, _ *model.QQOfficialApp, at time.Time) error {
		return s.applyEvent(ctx, tx, model.QQOfficialEvent{Kind: "GROUP_MSG_REJECT", GroupOpenID: primary.OpenID, EventAt: at}, at)
	}); err != nil {
		t.Fatal(err)
	}
	row, _, err := s.Task(ctx, one.ID)
	if err != nil || row.Status != model.QQCancelled || row.Text != "" {
		t.Fatalf("cancelled %+v %v", row, err)
	}
	if _, err = s.TestGroup(ctx, 7, primary.ID, uuid.NewString()); !errors.Is(err, ErrQQOfficialNotAuthorized) {
		t.Fatalf("opt-out not honored %v", err)
	}
}

func TestQQOfficialUncertainWriteIsNeverReplayed(t *testing.T) {
	s, _, now := qqFixture(t)
	qqEnable(t, s)
	g := qqGroupFixture(t, s, 1)
	task := qqSubmitTest(t, s, g)
	var sent int
	s.setToken(qqToken(*now), 0)
	s.client = qqFakeClient{execute: func(_ context.Context, token string, in qqapi.Request) (qqapi.Result, error) {
		sent++
		if in.Kind != qqapi.SendText || in.Text != qqTestText || in.GroupOpenID != g.OpenID || token != "in-memory" {
			t.Fatalf("wrong request %+v", in)
		}
		return qqapi.Result{}, &qqapi.OperationError{Kind: qqapi.Ambiguous, TraceID: "trace-123"}
	}}
	if did, err := s.dispatchOnce(t.Context()); !did || err != nil {
		t.Fatalf("dispatch %t %v", did, err)
	}
	row, attempts, err := s.Task(t.Context(), task.ID)
	if err != nil || row.Status != model.QQUnknown || row.Text != "" || len(attempts) != 1 || attempts[0].TraceID != "trace-123" {
		t.Fatalf("state %+v %+v %v", row, attempts, err)
	}
	*now = now.Add(time.Minute)
	if did, err := s.dispatchOnce(t.Context()); did || err != nil || sent != 1 {
		t.Fatalf("duplicate %d dispatch=%t error=%v", sent, did, err)
	}
	if _, _, err = s.Cancel(t.Context(), 7, task.ID); !errors.Is(err, ErrQQOfficialConflict) {
		t.Fatalf("terminal cancellation %v", err)
	}
}

func TestQQOfficialTokenSharesAdmissionAndThrottleSurvivesWorker(t *testing.T) {
	s, db, now := qqFixture(t)
	qqEnable(t, s)
	task := qqSubmitTest(t, s, qqGroupFixture(t, s, 1))
	s.client = qqFakeClient{refresh: func(context.Context) (qqapi.Token, error) { return qqToken(*now), nil }, execute: func(context.Context, string, qqapi.Request) (qqapi.Result, error) {
		return qqapi.Result{}, &qqapi.OperationError{Kind: qqapi.Throttle, HTTPStatus: 429, RetryAfter: 90 * time.Second}
	}}
	if got, err := s.dispatchOnce(t.Context()); !got || err != nil {
		t.Fatalf("token %t %v", got, err)
	}
	if got, err := s.dispatchOnce(t.Context()); got || err != nil {
		t.Fatalf("slot gap %t %v", got, err)
	}
	*now = now.Add(time.Second)
	if got, err := s.dispatchOnce(t.Context()); !got || err != nil {
		t.Fatalf("send %t %v", got, err)
	}
	second := NewQQOfficialService(db, "1234", "sample-secret", true)
	second.now = s.now
	second.setToken(s.currentToken())
	token, epoch := second.currentToken()
	*now = now.Add(30 * time.Second)
	if p, err := second.claim(t.Context(), token, epoch); err != nil || p != nil {
		t.Fatalf("cold worker bypassed cooldown %+v %v", p, err)
	}
	view, err := s.Status(t.Context())
	if err != nil || len(view.Cooldowns) != 2 || view.AttemptsLastMinute != 2 {
		t.Fatalf("status %+v %v", view, err)
	}
	row, _, err := s.Task(t.Context(), task.ID)
	if err != nil || row.Status != model.QQRetryWait || row.RetryCount != 0 {
		t.Fatalf("status %+v %v", row, err)
	}
	*now = now.Add(61 * time.Second)
	if p, err := second.claim(t.Context(), token, epoch); err != nil || p == nil || p.attempt.TaskID != task.ID {
		t.Fatalf("no recovery %+v %v", p, err)
	}
}

func TestQQOfficialExpiredLeaseFencesLateResult(t *testing.T) {
	s, _, now := qqFixture(t)
	qqEnable(t, s)
	task := qqSubmitTest(t, s, qqGroupFixture(t, s, 1))
	first, err := s.claim(t.Context(), qqToken(*now), 0)
	if err != nil || first == nil {
		t.Fatalf("claim %+v %v", first, err)
	}
	*now = now.Add(31 * time.Second)
	if p, err := s.claim(t.Context(), qqToken(*now), 0); err != nil || p != nil {
		t.Fatalf("incorrect reclaimed write %+v %v", p, err)
	}
	accepted, err := s.finish(t.Context(), first.attempt.ID, qqapi.Result{MessageID: "late"}, qqapi.Token{}, nil)
	if err != nil || accepted {
		t.Fatalf("late result accepted %t %v", accepted, err)
	}
	row, _, err := s.Task(t.Context(), task.ID)
	if err != nil || row.Status != model.QQUnknown || row.Reason != "lease_expired" {
		t.Fatalf("state %+v %v", row, err)
	}
}

func TestQQOfficialReadMergeAndApprovalCannotReverse(t *testing.T) {
	s, db, now := qqFixture(t)
	qqEnable(t, s)
	g := qqGroupFixture(t, s, 1)
	key := uuid.NewString()
	one, err := s.RefreshJoinRequests(t.Context(), 7, g.ID, key)
	if err != nil {
		t.Fatal(err)
	}
	two, err := s.RefreshJoinRequests(t.Context(), 7, g.ID, uuid.NewString())
	if err != nil || two.ID != one.ID {
		t.Fatalf("read did not coalesce %+v %v", two, err)
	}
	s.setToken(qqToken(*now), 0)
	s.client = qqFakeClient{execute: func(_ context.Context, _ string, in qqapi.Request) (qqapi.Result, error) {
		if in.Kind == qqapi.JoinDecision {
			return qqapi.Result{}, nil
		}
		return qqapi.Result{JoinRequests: []qqapi.JoinRequest{{ID: "external-request", MemberOpenID: "external-member", Username: "Pilot", AppliedAt: *now}}}, nil
	}}
	if did, err := s.dispatchOnce(t.Context()); !did || err != nil {
		t.Fatalf("refresh %t %v", did, err)
	}
	rows, _, err := s.JoinRequests(t.Context(), g.ID, "pending", 0, 20)
	if err != nil || len(rows) != 1 {
		t.Fatalf("requests %+v %v", rows, err)
	}
	active := true
	if _, err = s.UpdateGroup(t.Context(), 7, g.ID, QQOfficialGroupUpdate{ApprovalEnabled: &active}); err != nil {
		t.Fatal(err)
	}
	operator := model.User{Nickname: "operator", Status: 1}
	operator.ID = 7
	if err = db.Create(&operator).Error; err != nil {
		t.Fatal(err)
	}
	if err = db.Create(&model.UserRole{UserID: 7, RoleCode: model.RoleSuperAdmin}).Error; err != nil {
		t.Fatal(err)
	}
	key = uuid.NewString()
	decision, err := s.DecideJoinRequest(t.Context(), 7, rows[0].ID, "approve", key)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = s.DecideJoinRequest(t.Context(), 7, rows[0].ID, "decline", uuid.NewString()); !errors.Is(err, ErrQQOfficialConflict) {
		t.Fatalf("opposite action accepted: %v", err)
	}
	if replay, err := s.DecideJoinRequest(t.Context(), 7, rows[0].ID, "approve", key); err != nil || replay.ID != decision.ID {
		t.Fatalf("idempotent decision %+v %v", replay, err)
	}
	*now = now.Add(time.Second)
	if got, err := s.dispatchOnce(t.Context()); err != nil || !got {
		t.Fatalf("decision %t %v", got, err)
	}
	rows, _, err = s.JoinRequests(t.Context(), g.ID, "approved", 0, 20)
	if err != nil || len(rows) != 1 {
		t.Fatalf("local projection %+v %v", rows, err)
	}
}

func TestQQOfficialGatewayEventCommitsAndDeduplicates(t *testing.T) {
	s, db, now := qqFixture(t)
	groupID := "QQGroup000000000000000000000001"
	event := qqapi.Event{ID: "evt-1", Type: qqapi.GroupAddRobot, GroupOpenID: groupID, At: *now}
	owner := uuid.NewString()
	if acquired, err := s.acquireGateway(t.Context(), owner); err != nil || !acquired {
		t.Fatalf("gateway lease: %t %v", acquired, err)
	}
	for range 2 {
		if err := s.storeGatewayEvent(t.Context(), owner, event); err != nil {
			t.Fatal(err)
		}
	}
	if status, err := s.Status(t.Context()); err != nil || status.LastEventAt == nil || !status.LastEventAt.Equal(*now) {
		t.Fatalf("last event status: %+v %v", status, err)
	}
	if n, err := s.repo.EventBacklog(t.Context(), nil, s.appID); err != nil || n != 1 {
		t.Fatalf("backlog %d %v", n, err)
	}
	if _, err := s.claim(t.Context(), qqapi.Token{}, 0); err != nil {
		t.Fatal(err)
	}
	g, err := s.repo.GroupByOpenID(t.Context(), nil, s.appID, groupID)
	if err != nil || !g.Present || g.Authorized || g.NotificationEnabled {
		t.Fatalf("unsafe discovered group %+v %v", g, err)
	}
	if err = db.Migrator().DropTable(&model.QQOfficialEvent{}); err != nil {
		t.Fatal(err)
	}
	if err = s.storeGatewayEvent(t.Context(), owner, event); err == nil {
		t.Fatal("event accepted without persistence")
	}
}

// Set QQ_TEST_POSTGRES_DSN only to a disposable PostgreSQL database created for
// this package. This test writes a unique application and requires a clean schema.
func TestQQOfficialPostgresParallelLoad(t *testing.T) {
	dsn := os.Getenv("QQ_TEST_POSTGRES_DSN")
	if dsn == "" {
		t.Skip("set QQ_TEST_POSTGRES_DSN to a disposable test database")
	}
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatal(err)
	}
	var database string
	if err = db.Raw("SELECT current_database()").Scan(&database).Error; err != nil || database != "qq_official_test" {
		t.Fatalf("not the disposable test database: %v", err)
	}
	qqMigrateFixture(t, db)
	s := NewQQOfficialService(db, "12345", "sample-secret", true)
	other := NewQQOfficialService(db, "12345", "sample-secret", true)
	qqEnable(t, s)
	groups := make([]model.QQOfficialGroup, 20)
	for i := range groups {
		groups[i] = qqGroupFixture(t, s, i+1)
	}
	var wg sync.WaitGroup
	errs := make(chan error, 1000)
	for i := range groups {
		wg.Go(func() {
			for j := 0; j < 50; j++ {
				worker := s
				if j%2 == 0 {
					worker = other
				}
				var submitErr error
				switch {
				case j == 0 && i%2 == 0:
					_, submitErr = worker.VerifyGroup(t.Context(), 7, groups[i].OpenID, uuid.NewString())
				case j == 0:
					_, submitErr = worker.RefreshJoinRequests(t.Context(), 7, groups[i].ID, uuid.NewString())
				default:
					_, submitErr = worker.TestGroup(t.Context(), 7, groups[i].ID, uuid.NewString())
				}
				if submitErr != nil {
					errs <- submitErr
				}
			}
		})
	}
	wg.Wait()
	close(errs)
	for err = range errs {
		t.Error(err)
	}
	if t.Failed() {
		return
	}
	state, err := s.Status(t.Context())
	if err != nil || state.QueueDepth != 1000 {
		t.Fatalf("depth %+v %v", state, err)
	}
	for range 30 {
		qqSubmitTest(t, s, groups[0])
	}
	if _, err = s.TestGroup(t.Context(), 7, groups[0].ID, uuid.NewString()); !errors.Is(err, ErrQQOfficialCapacity) {
		t.Fatalf("queue capacity %v", err)
	}

	var calls, inFlight atomic.Int32
	entered, release := make(chan struct{}), make(chan struct{})
	f := qqFakeClient{execute: func(ctx context.Context, _ string, _ qqapi.Request) (qqapi.Result, error) {
		n := inFlight.Add(1)
		defer inFlight.Add(-1)
		if n != 1 {
			return qqapi.Result{}, errors.New("unexpected concurrent provider requests")
		}
		if calls.Add(1) == 1 {
			close(entered)
		}
		select {
		case <-release:
			return qqapi.Result{MessageID: "verified", AllowProactive: true, MemberRole: "admin"}, nil
		case <-ctx.Done():
			return qqapi.Result{}, ctx.Err()
		}
	}}
	s.client, other.client = f, f
	s.setToken(qqToken(time.Now()), 0)
	other.setToken(s.currentToken())
	done := make(chan error, 1)
	go func() { _, err := s.dispatchOnce(t.Context()); done <- err }()
	select {
	case <-entered:
	case err = <-done:
		t.Fatalf("dispatch didn't enter provider: %v", err)
	case <-time.After(5 * time.Second):
		t.Fatal("provider not reached")
	}
	for range 12 {
		if got, err := other.dispatchOnce(t.Context()); got || err != nil {
			t.Errorf("parallel gate: %t %v", got, err)
		}
	}
	close(release)
	if err = <-done; err != nil {
		t.Fatal(err)
	}
	if calls.Load() != 1 {
		t.Fatalf("provider calls %d", calls.Load())
	}
}

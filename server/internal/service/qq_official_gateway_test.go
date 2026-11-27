package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"amiya-eden/internal/model"
	"amiya-eden/internal/service/internal/qqapi"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type qqFakeGateway struct {
	discover func(context.Context, string) (string, error)
	connect  func(context.Context, string, string, bool, *qqapi.GatewaySession, func(context.Context) error, func(context.Context, qqapi.Event) error) error
}

func (f qqFakeGateway) DiscoverGateway(ctx context.Context, token string) (string, error) {
	return f.discover(ctx, token)
}

func (f qqFakeGateway) RunGateway(ctx context.Context, addr, token string, joinRequests bool, session *qqapi.GatewaySession, ready func(context.Context) error, event func(context.Context, qqapi.Event) error) error {
	return f.connect(ctx, addr, token, joinRequests, session, ready, event)
}

func TestQQOfficialGatewayLeaseFencesOldEventsAndRelease(t *testing.T) {
	s, db, now := qqFixture(t)
	other := NewQQOfficialService(db, s.appID, "other-secret", false)
	other.now = s.now
	ctx := t.Context()
	first, second := uuid.NewString(), uuid.NewString()
	if got, err := s.acquireGateway(ctx, first); err != nil || !got {
		t.Fatalf("first lease = %t, %v", got, err)
	}
	if got, err := other.acquireGateway(ctx, second); err != nil || got {
		t.Fatalf("concurrent lease = %t, %v", got, err)
	}
	*now = now.Add(qqGatewayLeaseTTL)
	if status, err := s.Status(ctx); err != nil || status.GatewayState != qqGatewayRetrying {
		t.Fatalf("stale lease status = %q, %v", status.GatewayState, err)
	}
	if got, err := other.acquireGateway(ctx, second); err != nil || !got {
		t.Fatalf("takeover lease = %t, %v", got, err)
	}
	event := qqapi.Event{ID: "fenced", Type: qqapi.GroupAddRobot, GroupOpenID: "QQGroup000000000000000000000001", At: *now}
	if err := s.storeGatewayEvent(ctx, first, event); !errors.Is(err, ErrQQOfficialUnavailable) {
		t.Fatalf("expired owner event = %v", err)
	}
	if err := s.releaseGateway(ctx, first, qqGatewayBlocked); err != nil {
		t.Fatal(err)
	}
	if err := other.storeGatewayEvent(ctx, second, event); err != nil {
		t.Fatalf("replacement owner event = %v", err)
	}
}

func TestQQOfficialGatewaySharesHTTPBudgetWhileDispatchDisabled(t *testing.T) {
	s, _, now := qqFixture(t)
	ctx, owner := t.Context(), uuid.NewString()
	if acquired, err := s.acquireGateway(ctx, owner); err != nil || !acquired {
		t.Fatalf("gateway lease: %t %v", acquired, err)
	}
	refreshes, discoveries := 0, 0
	s.client = qqFakeClient{refresh: func(context.Context) (qqapi.Token, error) {
		refreshes++
		return qqToken(*now), nil
	}}
	s.gateway = qqFakeGateway{discover: func(_ context.Context, token string) (string, error) {
		discoveries++
		if token != "in-memory" {
			t.Errorf("discovery token wasn't refreshed")
		}
		return "", &qqapi.OperationError{Kind: qqapi.Throttle, HTTPStatus: 429, RetryAfter: time.Minute}
	}}
	if _, _, err := s.gatewayBootstrapStep(ctx, owner); err != nil || refreshes != 1 {
		t.Fatalf("initial refresh = %d, %v", refreshes, err)
	}
	if _, _, err := s.gatewayBootstrapStep(ctx, owner); err != nil || discoveries != 0 {
		t.Fatalf("discovery ignored HTTP pacing = %d, %v", discoveries, err)
	}
	*now = now.Add(time.Second)
	if _, _, err := s.gatewayBootstrapStep(ctx, owner); err == nil || discoveries != 1 {
		t.Fatalf("discovery throttle = %d, %v", discoveries, err)
	}
	qqEnable(t, s)
	group := qqGroupFixture(t, s, 1)
	qqSubmitTest(t, s, group)
	*now = now.Add(time.Second)
	if dispatched, err := s.dispatchOnce(ctx); err != nil || dispatched {
		t.Fatalf("dispatch ignored discovery cooldown = %t, %v", dispatched, err)
	}
	if status, err := s.Status(ctx); err != nil || status.AttemptsLastMinute != 2 || status.TokenState != "ready" {
		t.Fatalf("shared attempt ledger: %+v, %v", status, err)
	}
}

func TestQQOfficialGatewayRuntimeReadyWhileDisabledAndStopsOnCancellation(t *testing.T) {
	s, db, _ := qqFixture(t)
	ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
	defer cancel()
	r := NewQQOfficialRuntime(db, "deployment-test-secret")
	secret := "sample-secret"
	if _, err := r.UpdateSettings(ctx, 7, QQOfficialSettingsUpdate{AppID: &s.appID, AppSecret: &secret}); err != nil {
		t.Fatal(err)
	}
	active, err := r.Active(ctx, nil)
	if err != nil || active == nil {
		t.Fatalf("missing active configuration: %v", err)
	}
	// Read the generation initialized by saving the credentials rather than
	// relying on the migration-time token generation value.
	app, err := active.repo.App(ctx, nil, active.appID, false)
	if err != nil {
		t.Fatal(err)
	}
	active.setToken(qqToken(time.Now()), app.TokenGeneration)
	ready, done := make(chan struct{}), make(chan error, 1)
	active.gateway = qqFakeGateway{
		discover: func(context.Context, string) (string, error) { return "wss://mock", nil },
		connect: func(conn context.Context, addr, token string, joinRequests bool, _ *qqapi.GatewaySession, onReady func(context.Context) error, onEvent func(context.Context, qqapi.Event) error) error {
			if addr != "wss://mock" || token != "in-memory" || joinRequests {
				return errors.New("wrong handshake arguments")
			}
			if err := onReady(conn); err != nil {
				return err
			}
			if err := onEvent(conn, qqapi.Event{ID: "gateway-ready", Type: qqapi.GroupAddRobot, GroupOpenID: "QQGroup000000000000000000000001", At: time.Now()}); err != nil {
				return err
			}
			close(ready)
			<-conn.Done()
			return conn.Err()
		},
	}
	go func() { done <- r.RunGateway(ctx) }()
	select {
	case <-ready:
	case <-ctx.Done():
		t.Fatal("gateway did not start with dispatch disabled")
	}
	if status, err := r.Status(ctx); err != nil || status.Enabled || status.GatewayState != qqGatewayConnected || status.LastEventAt == nil {
		t.Fatalf("status after Ready: %+v %v", status, err)
	}
	cancel()
	select {
	case err := <-done:
		if !errors.Is(err, context.Canceled) {
			t.Fatal(err)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("gateway did not stop on shutdown")
	}
	app, err = active.repo.App(t.Context(), nil, active.appID, false)
	if err != nil || app.GatewayLeaseID != "" {
		t.Fatalf("receiver did not release lease: %v", err)
	}
}

func TestQQOfficialGatewayFallsBackWhenOptionalIntentIsForbidden(t *testing.T) {
	s, _, now := qqFixture(t) // approval writes are enabled on the fixture
	ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
	defer cancel()
	owner := uuid.NewString()
	if acquired, err := s.acquireGateway(ctx, owner); err != nil || !acquired {
		t.Fatalf("gateway lease: %t %v", acquired, err)
	}
	s.setToken(qqToken(*now), 0)
	var attempted []bool
	s.gateway = qqFakeGateway{
		discover: func(context.Context, string) (string, error) { return "wss://mock", nil },
		connect: func(_ context.Context, _, _ string, joinRequests bool, _ *qqapi.GatewaySession, _ func(context.Context) error, _ func(context.Context, qqapi.Event) error) error {
			attempted = append(attempted, joinRequests)
			if len(attempted) == 1 {
				// Move the fixture's database clock past the shared discovery
				// window without waiting or letting the receiver lease expire.
				*now = now.Add(6 * time.Second)
				if err := s.withGateway(ctx, owner, func(app *model.QQOfficialApp, now time.Time) {
					until := now.Add(qqGatewayLeaseTTL)
					app.GatewayLeaseUntil = &until
				}); err != nil {
					t.Fatal(err)
				}
				*now = now.Add(9 * time.Second)
			}
			return qqapi.ErrGatewayIntentForbidden
		},
	}
	if err := s.connectGateway(ctx, owner); !errors.Is(err, qqapi.ErrGatewayBlocked) {
		t.Fatalf("base permission rejection = %v", err)
	}
	if len(attempted) != 2 || !attempted[0] || attempted[1] {
		t.Fatalf("intent attempts: %v", attempted)
	}
}

func TestQQOfficialGatewayBlockedStateRequiresExplicitRecovery(t *testing.T) {
	s, db, _ := qqFixture(t)
	r := NewQQOfficialRuntime(db, "deployment-test-secret")
	ctx := t.Context()
	secret := "sample-secret"
	if _, err := r.UpdateSettings(ctx, 7, QQOfficialSettingsUpdate{AppID: &s.appID, AppSecret: &secret}); err != nil {
		t.Fatal(err)
	}
	active, err := r.Active(ctx, nil)
	if err != nil || active == nil {
		t.Fatalf("missing active configuration: %v", err)
	}
	owner := uuid.NewString()
	if acquired, err := active.acquireGateway(ctx, owner); err != nil || !acquired {
		t.Fatalf("gateway lease: %t %v", acquired, err)
	}
	if err := active.locked(ctx, nil, func(tx *gorm.DB, app *model.QQOfficialApp, _ time.Time) error {
		app.GlobalHold = "auth_blocked"
		return active.repo.SaveApp(ctx, tx, app)
	}); err != nil {
		t.Fatal(err)
	}
	active.serveGateway(ctx, owner)
	if status, err := r.Status(ctx); err != nil || status.GatewayState != qqGatewayBlocked {
		t.Fatalf("blocked state: %+v %v", status, err)
	}
	if acquired, err := active.acquireGateway(ctx, uuid.NewString()); err != nil || acquired {
		t.Fatalf("blocked lease was reacquired: %t %v", acquired, err)
	}
	if status, err := r.UpdateSettings(ctx, 7, QQOfficialSettingsUpdate{AppID: &s.appID}); err != nil || status.GatewayState != qqGatewayIdle || status.TokenState != "idle" {
		t.Fatalf("saving AppID did not clear the block: %+v %v", status, err)
	}
	if acquired, err := active.acquireGateway(ctx, uuid.NewString()); err != nil || !acquired {
		t.Fatalf("reset lease was not acquired: %t %v", acquired, err)
	}
}

package service

import (
	"context"
	"errors"
	"math/rand/v2"
	"time"

	"amiya-eden/internal/model"
	"amiya-eden/internal/service/internal/qqapi"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

const (
	qqGatewayIdle       = "idle"
	qqGatewayConnecting = "connecting"
	qqGatewayConnected  = "connected"
	qqGatewayRetrying   = "retrying"
	qqGatewayBlocked    = "blocked"
	qqGatewayLeaseTTL   = 15 * time.Second
	qqGatewayPoll       = 3 * time.Second
	qqGatewayDBBudget   = 2 * time.Second
	qqGatewayRetryMin   = 2 * time.Second
	qqGatewayRetryMax   = time.Minute
)

func qqOwnsGateway(app *model.QQOfficialApp, owner string, now time.Time) bool {
	return owner != "" && app.GatewayLeaseID == owner && app.GatewayLeaseUntil != nil && app.GatewayLeaseUntil.After(now)
}

// RunGateway is independent of outbound dispatch: a newly configured or
// paused installation still discovers groups and observes membership changes.
func (r *QQOfficialRuntime) RunGateway(ctx context.Context) error {
	if r == nil {
		return nil
	}
	for ctx.Err() == nil {
		cycle, cancel := context.WithTimeout(ctx, qqGatewayDBBudget)
		s, err := r.Active(cycle, nil)
		owner := uuid.NewString()
		var acquired bool
		if err == nil && s != nil {
			acquired, err = s.acquireGateway(cycle, owner)
		}
		cancel()
		if err == nil && acquired {
			s.serveGateway(ctx, owner)
		}
		if !qqWaitGateway(ctx, qqGatewayPoll) {
			break
		}
	}
	return ctx.Err()
}

func (s *QQOfficialService) acquireGateway(ctx context.Context, owner string) (bool, error) {
	acquired := false
	err := s.locked(ctx, nil, func(tx *gorm.DB, app *model.QQOfficialApp, now time.Time) error {
		// Even if settings changed, wait for the old owner to close or expire.
		if qqOwnsGateway(app, app.GatewayLeaseID, now) || app.GatewayRevision == s.configRevision && app.GatewayState == qqGatewayBlocked {
			return nil
		}
		until := now.Add(qqGatewayLeaseTTL)
		app.GatewayLeaseID, app.GatewayLeaseUntil = owner, &until
		app.GatewayRevision, app.GatewayState = s.configRevision, qqGatewayConnecting
		acquired = true
		return s.repo.SaveApp(ctx, tx, app)
	})
	return acquired && err == nil, err
}

// The configuration and lease are rechecked on every event and on renewal.
// The receiver is cancelled well before this database-clock lease can expire.
func (s *QQOfficialService) withGateway(ctx context.Context, owner string, update func(*model.QQOfficialApp, time.Time)) error {
	cycle, cancel := context.WithTimeout(ctx, qqGatewayDBBudget)
	defer cancel()
	return s.locked(cycle, nil, func(tx *gorm.DB, app *model.QQOfficialApp, now time.Time) error {
		if !qqOwnsGateway(app, owner, now) {
			return ErrQQOfficialUnavailable
		}
		if app.GlobalHold != "" {
			return qqapi.ErrGatewayBlocked
		}
		update(app, now)
		return s.repo.SaveApp(cycle, tx, app)
	})
}

func (s *QQOfficialService) releaseGateway(ctx context.Context, owner, state string) error {
	// This runs after closing the socket and only releases our own lease, even
	// if the credential revision changed in the meantime.
	return s.repo.Transaction(ctx, nil, func(tx *gorm.DB) error {
		return s.withAppLock(ctx, tx, func(tx *gorm.DB, app *model.QQOfficialApp, _ time.Time) error {
			if app.GatewayLeaseID != owner {
				return nil
			}
			app.GatewayLeaseID, app.GatewayLeaseUntil, app.GatewayState = "", nil, state
			return s.repo.SaveApp(ctx, tx, app)
		})
	})
}

func (s *QQOfficialService) serveGateway(ctx context.Context, owner string) {
	connection, stop := context.WithCancel(ctx)
	defer stop()
	done := make(chan error, 1)
	go func() { done <- s.connectGateway(connection, owner) }()
	ticker := time.NewTicker(qqGatewayPoll)
	defer ticker.Stop()
	var err error
	defer func() {
		state := qqGatewayRetrying
		if errors.Is(err, qqapi.ErrGatewayBlocked) {
			state = qqGatewayBlocked
		}
		cleanup, cancel := context.WithTimeout(context.WithoutCancel(ctx), qqGatewayDBBudget)
		defer cancel()
		_ = s.releaseGateway(cleanup, owner, state)
	}()
	for {
		select {
		case err = <-done:
			return
		case <-ctx.Done():
			stop()
			<-done
			return
		case <-ticker.C:
			err = s.withGateway(connection, owner, func(app *model.QQOfficialApp, now time.Time) {
				until := now.Add(qqGatewayLeaseTTL)
				app.GatewayLeaseUntil = &until
			})
			if err != nil {
				stop()
				<-done
				return
			}
		}
	}
}

func (s *QQOfficialService) connectGateway(ctx context.Context, owner string) error {
	var session qqapi.GatewaySession
	joinRequests := s.approvalWrites
	backoff := qqGatewayRetryMin
	for ctx.Err() == nil {
		addr, token, err := s.gatewayBootstrap(ctx, owner)
		if err == nil {
			err = s.gateway.RunGateway(ctx, addr, token, joinRequests, &session, func(eventCtx context.Context) error {
				backoff = qqGatewayRetryMin
				return s.withGateway(eventCtx, owner, func(app *model.QQOfficialApp, _ time.Time) {
					app.GatewayState = qqGatewayConnected
				})
			}, func(eventCtx context.Context, event qqapi.Event) error {
				persist, cancel := context.WithTimeout(eventCtx, qqGatewayDBBudget)
				defer cancel()
				return s.storeGatewayEvent(persist, owner, event)
			})
		}
		if ctx.Err() != nil {
			return ctx.Err()
		}
		if joinRequests && errors.Is(err, qqapi.ErrGatewayIntentForbidden) {
			// Do not retry a forbidden subscription. On accounts without the
			// optional permission, use basic group events and manual refresh.
			joinRequests = false
			session = qqapi.GatewaySession{}
		} else if errors.Is(err, qqapi.ErrGatewayBlocked) {
			return err
		}
		if errors.Is(err, qqapi.ErrGatewayInvalidToken) && token != "" {
			s.forgetGatewayToken(token)
		}
		if err = s.withGateway(ctx, owner, func(app *model.QQOfficialApp, _ time.Time) {
			app.GatewayState = qqGatewayRetrying
		}); err != nil {
			return err
		}
		if !qqWaitGateway(ctx, backoff+time.Duration(rand.Int64N(int64(time.Second)))) {
			break
		}
		backoff = min(2*backoff, qqGatewayRetryMax)
	}
	return ctx.Err()
}

func qqWaitGateway(ctx context.Context, delay time.Duration) bool {
	timer := time.NewTimer(delay)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return false
	case <-timer.C:
		return true
	}
}

func (s *QQOfficialService) forgetGatewayToken(token string) {
	s.tokenMu.Lock()
	defer s.tokenMu.Unlock()
	if s.token.AccessToken == token {
		s.token, s.tokenEpoch = qqapi.Token{}, 0
	}
}

func (s *QQOfficialService) gatewayBootstrap(ctx context.Context, owner string) (string, string, error) {
	for ctx.Err() == nil {
		addr, token, err := s.gatewayBootstrapStep(ctx, owner)
		if err != nil || addr != "" {
			return addr, token, err
		}
		if !qqWaitGateway(ctx, qqPollInterval) {
			break
		}
	}
	return "", "", ctx.Err()
}

// Allow only the receiver to discover the WSS endpoint. Token refresh and
// discovery reserve and settle ordinary attempts, including shared 429 cooldown.
func (s *QQOfficialService) gatewayBootstrapStep(ctx context.Context, owner string) (string, string, error) {
	call, cancel := context.WithTimeout(ctx, qqHTTPBudget)
	defer cancel()
	token, epoch := s.currentToken()
	var attempt model.QQOfficialAttempt
	var refreshEpoch int64
	err := s.locked(call, nil, func(tx *gorm.DB, app *model.QQOfficialApp, now time.Time) error {
		if !qqOwnsGateway(app, owner, now) {
			return ErrQQOfficialUnavailable
		}
		if app.GlobalHold != "" {
			return qqapi.ErrGatewayBlocked
		}
		cooldowns, err := s.requestCooldowns(call, tx, app, now, app.GatewayLeaseUntil.Sub(now) > qqHTTPBudget)
		if err != nil || cooldowns == nil {
			return err
		}
		kind := qqGatewayKind
		if epoch != app.TokenGeneration || token.AccessToken == "" || !token.ExpiresAt.After(s.now().Add(time.Minute)) {
			kind = qqTokenKind
		}
		wait, _, err := s.nextEligible(call, tx, app, cooldowns, kind, "", now)
		if err != nil || wait.After(now) {
			return err
		}
		refreshEpoch = app.TokenGeneration
		attempt, err = s.reserve(call, tx, app, kind, nil, now)
		return err
	})
	if err != nil || attempt.ID == "" {
		return "", "", err
	}
	var addr string
	var refreshed qqapi.Token
	if call.Err() != nil {
		err = &qqapi.OperationError{Kind: qqapi.TemporaryNotApplied}
	} else if attempt.Kind == qqTokenKind {
		refreshed, err = s.client.Refresh(call)
	} else {
		addr, err = s.gateway.DiscoverGateway(call, token.AccessToken)
	}
	finish, end := context.WithTimeout(context.WithoutCancel(ctx), 3*time.Second)
	defer end()
	accepted, finishErr := s.finish(finish, attempt.ID, qqapi.Result{}, refreshed, err)
	if finishErr != nil {
		return "", "", finishErr
	}
	if !accepted {
		return "", "", ErrQQOfficialUnavailable
	}
	if err == nil && attempt.Kind == qqTokenKind {
		s.setToken(refreshed, refreshEpoch)
	}
	var upstream *qqapi.OperationError
	if errors.As(err, &upstream) {
		if upstream.Kind == qqapi.AuthBlocked {
			return "", "", qqapi.ErrGatewayBlocked
		}
		if upstream.Kind == qqapi.InvalidToken && attempt.Kind == qqGatewayKind {
			s.forgetGatewayToken(token.AccessToken)
		}
	}
	if ctx.Err() != nil {
		return "", "", ctx.Err()
	}
	return addr, token.AccessToken, err
}

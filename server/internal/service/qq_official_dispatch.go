package service

import (
	"amiya-eden/internal/model"
	"amiya-eden/internal/repository"
	"amiya-eden/internal/service/internal/qqapi"
	"context"
	"errors"
	"math/rand/v2"
	"sort"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

const (
	qqTokenKind     = "token"
	qqGatewayKind   = "gateway"
	qqHTTPBudget    = 5 * time.Second
	qqLeaseDuration = 30 * time.Second
	qqPollInterval  = time.Second
	qqGroupWriteGap = 10 * time.Second
	qqPageCap       = 20
)

func qqMessageBudget(certified bool) int {
	if certified {
		return 48
	}
	return 24
}

// window pads the reservation timestamp by the maximum dispatch delay. Even
// when a process stalls after commit, requests still cannot bunch at a boundary.
func (s *QQOfficialService) window(ctx context.Context, tx *gorm.DB, group, kind string, now time.Time, limit int, duration time.Duration) (time.Time, error) {
	span := duration + qqHTTPBudget
	stamp, err := s.repo.AttemptAtLimit(ctx, tx, s.appID, group, kind, now.Add(-span), limit)
	if err != nil || stamp == nil {
		return time.Time{}, err
	}
	return stamp.Add(span), nil
}

func qqScope(kind string) string {
	switch kind {
	case qqTokenKind, qqGatewayKind:
		return kind
	case qqapi.SendText:
		return "message"
	case qqapi.JoinDecision:
		return "management_write"
	default:
		return "read"
	}
}

type qqAdmission struct {
	attempt    model.QQOfficialAttempt
	request    qqapi.Request
	lifetime   time.Duration // remaining at admission, bounded from before claiming
	tokenEpoch int64
}

type qqLimit struct {
	group, kind string
	count       int
	duration    time.Duration
}

func (s *QQOfficialService) nextEligible(ctx context.Context, tx *gorm.DB, app *model.QQOfficialApp, cooldowns map[string]model.QQOfficialCooldown, kind, group string, now time.Time) (time.Time, string, error) {
	until, reason := now, ""
	for _, scope := range []string{qqScope(kind), "group:" + group + ":" + qqScope(kind)} {
		if row := cooldowns[scope]; row.Until != nil && row.Until.After(until) {
			until, reason = *row.Until, "cooldown"
		}
	}
	limits := []qqLimit{}
	switch kind {
	case qqapi.SendText:
		limits = append(limits, qqLimit{"", kind, qqMessageBudget(app.Certified), time.Minute}, qqLimit{group, kind, 12, time.Minute}, qqLimit{group, kind, 800, 24 * time.Hour})
	case qqapi.GroupState, qqapi.JoinList:
		limits = append(limits, qqLimit{"", kind, 20, time.Minute})
	case qqapi.JoinDecision:
		limits = append(limits, qqLimit{"", kind, 40, time.Minute}, qqLimit{group, kind, 1, qqGroupWriteGap})
	case qqTokenKind, qqGatewayKind:
		limits = append(limits, qqLimit{"", kind, 1, 10 * time.Second})
	}
	for _, limit := range limits {
		t, err := s.window(ctx, tx, limit.group, limit.kind, now, limit.count, limit.duration)
		if err != nil {
			return now, "", err
		}
		if t.After(until) {
			until, reason = t, "rate_limited"
		}
	}
	return until, reason, nil
}

func qqOrderCandidates(rows []model.QQOfficialTask, app *model.QQOfficialApp) {
	wantRegular := app.UrgentStreak >= 3
	sort.Slice(rows, func(i, j int) bool {
		a, b := rows[i], rows[j]
		if a.Urgent != b.Urgent {
			return a.Urgent != wantRegular
		}
		last := app.LastRegularGroup
		if a.Urgent {
			last = app.LastUrgentGroup
		}
		aAfter, bAfter := a.GroupID > last, b.GroupID > last
		if aAfter != bAfter {
			return aAfter
		}
		return a.GroupID < b.GroupID
	})
}

func (s *QQOfficialService) currentToken() (qqapi.Token, int64) {
	s.tokenMu.Lock()
	defer s.tokenMu.Unlock()
	return s.token, s.tokenEpoch
}
func (s *QQOfficialService) setToken(token qqapi.Token, epoch int64) {
	s.tokenMu.Lock()
	defer s.tokenMu.Unlock()
	s.token, s.tokenEpoch = token, epoch
}

func (s *QQOfficialService) dispatchOnce(ctx context.Context) (bool, error) {
	// The deadline runs before attempting the SQL lock, not from an arbitrarily
	// delayed point after COMMIT. It is independent of clock synchronization.
	started := time.Now()
	callCtx, cancel := context.WithDeadline(ctx, started.Add(qqHTTPBudget))
	defer cancel()
	token, epoch := s.currentToken()
	permit, err := s.claim(callCtx, token, epoch)
	if err != nil || permit == nil {
		return false, err
	}
	if permit.attempt.Kind != qqTokenKind && permit.lifetime < qqHTTPBudget {
		var deadlineCancel context.CancelFunc
		callCtx, deadlineCancel = context.WithDeadline(callCtx, started.Add(permit.lifetime))
		defer deadlineCancel()
	}
	var result qqapi.Result
	var refreshed qqapi.Token
	if callCtx.Err() != nil {
		err = &qqapi.OperationError{Kind: qqapi.TemporaryNotApplied}
	} else if permit.attempt.Kind == qqTokenKind {
		refreshed, err = s.client.Refresh(callCtx)
	} else {
		result, err = s.client.Execute(callCtx, token.AccessToken, permit.request)
	}
	finishCtx, end := context.WithTimeout(context.WithoutCancel(ctx), 3*time.Second)
	defer end()
	accepted, finishErr := s.finish(finishCtx, permit.attempt.ID, result, refreshed, err)
	if finishErr != nil {
		return true, finishErr
	}
	var operationError *qqapi.OperationError
	if accepted && permit.attempt.Kind == qqTokenKind && err == nil {
		s.setToken(refreshed, permit.tokenEpoch)
	}
	if accepted && errors.As(err, &operationError) && (operationError.Kind == qqapi.InvalidToken || operationError.Kind == qqapi.AuthBlocked) {
		s.setToken(qqapi.Token{}, 0)
	}
	return true, nil
}

func (s *QQOfficialService) claim(ctx context.Context, token qqapi.Token, epoch int64) (*qqAdmission, error) {
	var permit *qqAdmission
	err := s.locked(ctx, nil, func(tx *gorm.DB, app *model.QQOfficialApp, now time.Time) error {
		if epoch != app.TokenGeneration {
			token = qqapi.Token{}
		}
		processed, err := s.processEvents(ctx, tx, now, 32)
		if err != nil {
			return err
		}
		if err = s.repo.Expire(ctx, tx, s.appID, now); err != nil {
			return err
		}
		if processed == 32 {
			s.signal()
		}
		cooldowns, err := s.requestCooldowns(ctx, tx, app, now, processed < 32 && app.Enabled && !app.Paused)
		if err != nil || cooldowns == nil {
			return err
		}
		rows, err := s.repo.ReadyHeads(ctx, tx, s.appID, now)
		if err != nil {
			return err
		}
		qqOrderCandidates(rows, app)
		for i := 0; i < min(len(rows), 32); i++ {
			task := &rows[i]
			request, invalid, err := s.checkTask(ctx, tx, app, *task, now)
			if err != nil {
				return err
			}
			if invalid != "" {
				qqTerminal(task, model.QQCancelled, invalid, now)
				if err = s.repo.SaveTask(ctx, tx, task); err != nil {
					return err
				}
				if err = s.projectDecision(ctx, tx, *task); err != nil {
					return err
				}
				continue
			}
			until, reason, err := s.nextEligible(ctx, tx, app, cooldowns, task.Kind, task.GroupID, now)
			if err != nil {
				return err
			}
			if until.After(now) {
				task.NextRunAt, task.Reason = until, reason
				if err = s.repo.SaveTask(ctx, tx, task); err != nil {
					return err
				}
				continue
			}
			if token.AccessToken == "" || !token.ExpiresAt.After(s.now().Add(time.Minute)) {
				wait, _, err := s.nextEligible(ctx, tx, app, cooldowns, qqTokenKind, "", now)
				if err != nil {
					return err
				}
				if !wait.After(now) {
					attempt, err := s.reserve(ctx, tx, app, qqTokenKind, nil, now)
					permit = &qqAdmission{attempt: attempt, tokenEpoch: app.TokenGeneration}
					return err
				}
				if token.AccessToken == "" || !token.ExpiresAt.After(s.now().Add(qqHTTPBudget)) {
					// Do not delay this task globally: another worker can already
					// hold a valid credential and take it as soon as the slot opens.
					return nil
				}
			}
			attempt, err := s.reserve(ctx, tx, app, task.Kind, task, now)
			permit = &qqAdmission{attempt: attempt, request: request, lifetime: task.ExpiresAt.Sub(now)}
			return err
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return permit, nil
}

// Both the gateway bootstrap and the dispatch worker use the same single HTTP
// slot and app-wide rate limits; the long-lived WebSocket does not hold that slot.
func (s *QQOfficialService) requestCooldowns(ctx context.Context, tx *gorm.DB, app *model.QQOfficialApp, now time.Time, allowed bool) (map[string]model.QQOfficialCooldown, error) {
	if app.LeaseID != "" {
		if app.LeaseUntil == nil || app.LeaseUntil.After(now) {
			return nil, nil
		}
		if err := s.recoverAttempt(ctx, tx, app, now); err != nil {
			return nil, err
		}
	}
	if !allowed || app.GlobalHold != "" || app.NextAdmissionAt != nil && app.NextAdmissionAt.After(now) {
		return nil, nil
	}
	global, err := s.window(ctx, tx, "", "", now, 48, time.Minute)
	if err != nil || global.After(now) {
		return nil, err
	}
	rows, err := s.repo.Cooldowns(ctx, tx, s.appID)
	if err != nil {
		return nil, err
	}
	cooldowns := make(map[string]model.QQOfficialCooldown, len(rows))
	for _, row := range rows {
		cooldowns[row.Scope] = row
	}
	if row := cooldowns["app"]; row.Until != nil && row.Until.After(now) {
		return nil, nil
	}
	return cooldowns, nil
}

func (s *QQOfficialService) checkTask(ctx context.Context, tx *gorm.DB, app *model.QQOfficialApp, task model.QQOfficialTask, now time.Time) (qqapi.Request, string, error) {
	req := qqapi.Request{Kind: task.Kind, Text: task.Text, Cursor: task.Cursor, Decision: task.Decision}
	if app.Generation != task.AppGeneration {
		return req, "configuration_changed", nil
	}
	group, err := s.repo.Group(ctx, tx, s.appID, task.GroupID)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return req, "group_missing", nil
	}
	if err != nil {
		return req, "", err
	}
	req.GroupOpenID = group.OpenID
	if group.Generation != task.GroupGeneration {
		return req, "group_changed", nil
	}
	if task.Kind == qqapi.GroupState {
		return req, "", nil
	}
	if !group.Present || !group.Authorized {
		return req, "group_not_authorized", nil
	}
	switch task.Kind {
	case qqapi.SendText:
		if group.NotifyGeneration != task.ModeGeneration || group.PushState == qqBlockedPush {
			return req, "push_rejected", nil
		}
		if task.Source != qqSourceTest && (!group.NotificationEnabled || !group.TestVerified || group.PushState != qqAllowedPush) {
			return req, "notification_disabled", nil
		}
		if task.Source == QQSourceFleet || task.Source == QQSourceStructure {
			valid, err := s.businessStillValid(ctx, tx, task, group, now)
			if err != nil {
				return req, "", err
			}
			if !valid {
				return req, "business_changed", nil
			}
		}
	case qqapi.JoinList:
		if group.ApprovalGeneration != task.ModeGeneration {
			return req, "approval_changed", nil
		}
	case qqapi.JoinDecision:
		if !s.approvalWrites || !group.ApprovalEnabled || group.ApprovalCapability != qqAvailable || group.ApprovalGeneration != task.ModeGeneration {
			return req, "approval_disabled", nil
		}
		roles, err := repository.NewRoleRepository().GetUserRoleCodesTx(tx, task.OperatorID)
		if err != nil {
			return req, "", err
		}
		if !model.IsSuperAdmin(roles) {
			return req, "operator_no_longer_authorized", nil
		}
		operator, err := repository.NewUserRepository().GetByIDTx(tx, task.OperatorID)
		if errors.Is(err, gorm.ErrRecordNotFound) || err == nil && operator.Status != 1 {
			return req, "operator_no_longer_authorized", nil
		}
		if err != nil {
			return req, "", err
		}
		application, err := s.repo.JoinRequest(ctx, tx, s.appID, task.ApplicationID)
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return req, "application_missing", nil
		}
		if err != nil {
			return req, "", err
		}
		if application.GroupID != group.ID || application.Status != "queued" || application.TaskID != task.ID || !application.ExpiresAt.After(now) {
			return req, "application_changed", nil
		}
		req.MemberOpenID, req.JoinRequestID = application.MemberOpenID, application.RequestID
	default:
		return req, "unknown_operation", nil
	}
	return req, "", nil
}

func (s *QQOfficialService) reserve(ctx context.Context, tx *gorm.DB, app *model.QQOfficialApp, kind string, task *model.QQOfficialTask, now time.Time) (model.QQOfficialAttempt, error) {
	row := model.QQOfficialAttempt{ID: uuid.NewString(), AppID: s.appID, Kind: kind, AdmittedAt: now}
	if task != nil {
		row.TaskID, row.GroupID = task.ID, task.GroupID
		task.Status, task.Reason = model.QQExecuting, ""
		if err := s.repo.SaveTask(ctx, tx, task); err != nil {
			return row, err
		}
		if task.Urgent {
			app.LastUrgentGroup = task.GroupID
			app.UrgentStreak = min(app.UrgentStreak+1, 3)
		} else {
			app.LastRegularGroup = task.GroupID
			app.UrgentStreak = 0
		}
	} else if kind == qqTokenKind {
		app.TokenState = "refreshing"
	}
	app.LeaseID, app.LeaseTaskID = row.ID, row.TaskID
	deadline := now.Add(qqLeaseDuration)
	app.LeaseUntil = &deadline
	if err := s.repo.SaveAttempt(ctx, tx, &row); err != nil {
		return row, err
	}
	return row, s.repo.SaveApp(ctx, tx, app)
}

func qqRelease(app *model.QQOfficialApp, now time.Time) {
	app.LeaseID, app.LeaseTaskID, app.LeaseUntil = "", "", nil
	next := now.Add(time.Second)
	app.NextAdmissionAt = &next
}

func qqSafeRead(kind string) bool { return kind == qqapi.GroupState || kind == qqapi.JoinList }

func (s *QQOfficialService) recoverAttempt(ctx context.Context, tx *gorm.DB, app *model.QQOfficialApp, now time.Time) error {
	row, err := s.repo.Attempt(ctx, tx, app.LeaseID)
	if err != nil {
		return err
	}
	row.Outcome, row.FinishedAt = "lease_expired", &now
	if err = s.repo.SaveAttempt(ctx, tx, &row); err != nil {
		return err
	}
	if row.Kind == qqTokenKind {
		app.TokenState = "unknown"
	}
	if app.LeaseTaskID != "" {
		task, err := s.repo.Task(ctx, tx, s.appID, app.LeaseTaskID)
		if err != nil {
			return err
		}
		if task.Status != model.QQExecuting {
			return ErrQQOfficialConflict
		}
		if qqSafeRead(task.Kind) {
			qqRetry(&task, now, "lease_expired")
		} else {
			qqTerminal(&task, model.QQUnknown, "lease_expired", now)
		}
		if err = s.repo.SaveTask(ctx, tx, &task); err != nil {
			return err
		}
		if err = s.projectDecision(ctx, tx, task); err != nil {
			return err
		}
	}
	if err = s.updateCooldown(ctx, tx, row, &qqapi.OperationError{Kind: qqapi.Ambiguous}, now); err != nil {
		return err
	}
	qqRelease(app, now)
	return s.repo.SaveApp(ctx, tx, app)
}

func qqRetry(task *model.QQOfficialTask, now time.Time, reason string) {
	steps := [...]time.Duration{5 * time.Second, 15 * time.Second, 45 * time.Second}
	if task.RetryCount >= len(steps) {
		qqTerminal(task, model.QQFailed, reason, now)
		return
	}
	wait := steps[task.RetryCount] + time.Duration(rand.Int64N(int64(time.Second)))
	task.RetryCount++
	if !now.Add(wait).Before(task.ExpiresAt) {
		qqTerminal(task, model.QQExpired, "expired", now)
		return
	}
	task.Status, task.Reason, task.NextRunAt = model.QQRetryWait, reason, now.Add(wait)
}

func (s *QQOfficialService) finish(ctx context.Context, id string, result qqapi.Result, token qqapi.Token, callErr error) (bool, error) {
	accepted := false
	err := s.repo.Transaction(ctx, nil, func(tx *gorm.DB) error {
		var changed bool
		if s.configRevision != 0 {
			row, err := s.repo.Installation(ctx, tx, "SHARE")
			if err != nil {
				return err
			}
			changed = row.AppID != s.appID || row.Revision != s.configRevision
		}
		return s.finishLocked(ctx, tx, id, result, token, callErr, changed, &accepted)
	})
	return accepted, err
}

func (s *QQOfficialService) finishLocked(ctx context.Context, tx *gorm.DB, id string, result qqapi.Result, token qqapi.Token, callErr error, changed bool, accepted *bool) error {
	return s.withAppLock(ctx, tx, func(tx *gorm.DB, app *model.QQOfficialApp, now time.Time) error {
		if app.LeaseID != id || app.LeaseUntil == nil || !app.LeaseUntil.After(now) {
			return nil
		}
		attempt, err := s.repo.Attempt(ctx, tx, id)
		if err != nil {
			return err
		}
		var upstream *qqapi.OperationError
		if callErr != nil && !errors.As(callErr, &upstream) {
			upstream = &qqapi.OperationError{Kind: qqapi.Ambiguous}
		}
		attempt.Outcome, attempt.TraceID, attempt.FinishedAt = "succeeded", result.TraceID, &now
		if upstream != nil {
			attempt.Outcome, attempt.TraceID, attempt.PlatformCode, attempt.HTTPStatus = string(upstream.Kind), upstream.TraceID, upstream.Code, upstream.HTTPStatus
		}
		if result.AwaitingReview && upstream == nil {
			attempt.Outcome = "awaiting_review"
		}
		if err = s.repo.SaveAttempt(ctx, tx, &attempt); err != nil {
			return err
		}
		if err = s.updateCooldown(ctx, tx, attempt, upstream, now); err != nil {
			return err
		}
		if upstream != nil && upstream.Kind == qqapi.AuthBlocked {
			app.GlobalHold = "auth_blocked"
		}
		if attempt.Kind == qqTokenKind {
			if upstream == nil {
				app.TokenState, app.TokenExpiresAt = "ready", &token.ExpiresAt
			} else {
				app.TokenState = string(upstream.Kind)
			}
		} else if attempt.TaskID != "" {
			task, err := s.repo.Task(ctx, tx, s.appID, attempt.TaskID)
			if err != nil {
				return err
			}
			if task.Status != model.QQExecuting {
				return ErrQQOfficialConflict
			}
			task.TraceID, task.UpstreamCode = attempt.TraceID, attempt.PlatformCode
			if err = s.applyOutcome(ctx, tx, app, &task, result, upstream, now, attempt.AdmittedAt.Truncate(time.Second)); err != nil {
				return err
			}
			if changed && task.Kind == qqapi.JoinDecision && task.Status == model.QQRetryWait {
				qqTerminal(&task, model.QQCancelled, "approval_disabled", now)
			}
			if err = s.repo.SaveTask(ctx, tx, &task); err != nil {
				return err
			}
			if err = s.projectDecision(ctx, tx, task); err != nil {
				return err
			}
		}
		qqRelease(app, now)
		// A timeout may still be executing at QQ; even other task types wait
		// one lease horizon to reduce the chance of overlap upstream.
		if upstream != nil && upstream.Kind == qqapi.Ambiguous && attempt.TaskID != "" && !qqSafeRead(attempt.Kind) {
			until := attempt.AdmittedAt.Add(qqLeaseDuration)
			if until.After(*app.NextAdmissionAt) {
				app.NextAdmissionAt = &until
			}
		}
		if err = s.repo.SaveApp(ctx, tx, app); err != nil {
			return err
		}
		*accepted = true
		return nil
	})
}

func (s *QQOfficialService) applyOutcome(ctx context.Context, tx *gorm.DB, app *model.QQOfficialApp, task *model.QQOfficialTask, result qqapi.Result, e *qqapi.OperationError, now, observedAt time.Time) error {
	group, err := s.repo.Group(ctx, tx, s.appID, task.GroupID)
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}
	validGroup := err == nil && app.Generation == task.AppGeneration && group.Generation == task.GroupGeneration
	validNotify := validGroup && group.NotifyGeneration == task.ModeGeneration
	validApproval := validGroup && group.ApprovalGeneration == task.ModeGeneration
	if e != nil {
		reason := string(e.Kind)
		switch e.Kind {
		case qqapi.Throttle:
			task.Status, task.Reason, task.NextRunAt = model.QQRetryWait, reason, now.Add(time.Second)
		case qqapi.InvalidToken:
			app.TokenState, app.TokenExpiresAt = "expired", nil
			app.TokenGeneration++
			if task.CredentialRetried {
				qqTerminal(task, model.QQFailed, reason, now)
			} else {
				task.CredentialRetried = true
				task.Status, task.Reason, task.NextRunAt = model.QQRetryWait, reason, now.Add(time.Second)
			}
		case qqapi.TemporaryNotApplied:
			qqRetry(task, now, reason)
		case qqapi.Ambiguous:
			if qqSafeRead(task.Kind) {
				qqRetry(task, now, reason)
			} else {
				qqTerminal(task, model.QQUnknown, reason, now)
			}
		default:
			qqTerminal(task, model.QQFailed, reason, now)
		}
		if validGroup && e.Kind == qqapi.GroupNotFound {
			group.Present, group.Authorized, group.TestVerified, group.NotificationEnabled, group.ApprovalEnabled = false, false, false, false, false
			group.MembershipAt = &now
			group.Generation++
			if err = s.repo.InvalidateQueued(ctx, tx, s.appID, group.ID, "", "bot_not_in_group", now); err != nil {
				return err
			}
			return s.repo.SaveGroup(ctx, tx, &group)
		}
		if validNotify && task.Kind == qqapi.SendText && e.Kind == qqapi.PushRejected {
			group.PushState, group.PushAt, group.TestVerified, group.NotificationEnabled = qqBlockedPush, &now, false, false
			group.NotifyGeneration++
			if err = s.repo.InvalidateQueued(ctx, tx, s.appID, group.ID, "notification", "push_rejected", now); err != nil {
				return err
			}
			return s.repo.SaveGroup(ctx, tx, &group)
		}
		if validApproval && (task.Kind == qqapi.JoinList || task.Kind == qqapi.JoinDecision) && e.Kind == qqapi.Forbidden {
			group.ApprovalCapability, group.ApprovalEnabled = qqForbidden, false
			group.ApprovalGeneration++
			if err = s.repo.InvalidateQueued(ctx, tx, s.appID, group.ID, "approval", "approval_disabled", now); err != nil {
				return err
			}
			return s.repo.SaveGroup(ctx, tx, &group)
		}
		return nil
	}
	if result.AwaitingReview && task.Kind == qqapi.SendText {
		task.Status, task.Reason, task.NextRunAt, task.Text = model.QQAwaitingReview, "awaiting_review", now.Add(15*time.Minute), ""
		return nil
	}
	switch task.Kind {
	case qqapi.SendText:
		if result.MessageID == "" {
			qqTerminal(task, model.QQUnknown, "ambiguous", now)
			return nil
		}
		task.ReceiptDigest = qqHash(result.MessageID)
		if validNotify && task.Source == qqSourceTest && (group.PushAt == nil || !group.PushAt.After(observedAt)) {
			group.PushState, group.PushAt, group.TestVerified = qqAllowedPush, &observedAt, true
			if err = s.repo.SaveGroup(ctx, tx, &group); err != nil {
				return err
			}
		}
	case qqapi.GroupState:
		if validGroup && (group.MembershipAt == nil || !group.MembershipAt.After(observedAt)) {
			group.Present, group.MembershipAt = true, &observedAt
			if group.PushAt == nil || group.PushAt.Before(observedAt) {
				if result.AllowProactive {
					group.PushState, group.PushAt = qqAllowedPush, &observedAt
				} else {
					group.PushState, group.PushAt, group.NotificationEnabled, group.TestVerified = qqBlockedPush, &observedAt, false, false
					group.NotifyGeneration++
					if err = s.repo.InvalidateQueued(ctx, tx, s.appID, group.ID, "notification", "push_rejected", now); err != nil {
						return err
					}
				}
			}
			if result.MemberRole == "member" && group.ApprovalCapability == qqAvailable {
				group.ApprovalCapability, group.ApprovalEnabled = qqForbidden, false
				group.ApprovalGeneration++
				if err = s.repo.InvalidateQueued(ctx, tx, s.appID, group.ID, "approval", "approval_disabled", now); err != nil {
					return err
				}
			}
			if err = s.repo.SaveGroup(ctx, tx, &group); err != nil {
				return err
			}
		}
	case qqapi.JoinList:
		if !validApproval || !group.Authorized {
			qqTerminal(task, model.QQCancelled, "group_changed", now)
			return nil
		}
		group.ApprovalCapability = qqAvailable
		if err = s.repo.SaveGroup(ctx, tx, &group); err != nil {
			return err
		}
		for _, row := range result.JoinRequests {
			if err = s.mergeJoinRequest(ctx, tx, group, row, task.ID, now); err != nil {
				return err
			}
		}
		task.PageCount++
		if result.NextCursor != "" {
			if task.PageCount >= qqPageCap || qqSeenCursor(task.VisitedCursors, result.NextCursor) {
				qqTerminal(task, model.QQFailed, "incomplete_list", now)
				return nil
			}
			task.VisitedCursors = append(task.VisitedCursors, qqHash(result.NextCursor))
			task.Cursor, task.NextRunAt, task.Status, task.Reason = result.NextCursor, now, model.QQQueued, ""
			return nil
		}
		if err = s.repo.CloseMissingRequests(ctx, tx, s.appID, group.ID, task.ID, task.CreatedAt); err != nil {
			return err
		}
	}
	qqTerminal(task, model.QQSucceeded, "", now)
	return nil
}

func qqSeenCursor(hashes []string, cursor string) bool {
	hash := qqHash(cursor)
	for _, previous := range hashes {
		if previous == hash {
			return true
		}
	}
	return false
}

func (s *QQOfficialService) updateCooldown(ctx context.Context, tx *gorm.DB, attempt model.QQOfficialAttempt, e *qqapi.OperationError, now time.Time) error {
	scope := qqScope(attempt.Kind)
	rows, err := s.repo.Cooldowns(ctx, tx, s.appID)
	if err != nil {
		return err
	}
	m := make(map[string]model.QQOfficialCooldown, len(rows))
	for _, row := range rows {
		m[row.Scope] = row
	}
	row := m[scope]
	row.AppID, row.Scope = s.appID, scope
	if e == nil {
		if row.Failures == 0 && row.ThrottleRun == 0 && row.Until == nil {
			return nil
		}
		row.Failures, row.ThrottleRun, row.Until = 0, 0, nil
		return s.repo.SaveCooldown(ctx, tx, &row)
	}
	if e.Kind == qqapi.Throttle {
		row.ThrottleRun = min(row.ThrottleRun+1, 3)
		steps := [...]time.Duration{time.Minute, 5 * time.Minute, 15 * time.Minute}
		until := now.Add(max(e.RetryAfter, steps[row.ThrottleRun-1]))
		if row.Until == nil || until.After(*row.Until) {
			row.Until = &until
		}
		if err = s.repo.SaveCooldown(ctx, tx, &row); err != nil {
			return err
		}
		if e.HTTPStatus == 429 && attempt.Kind != qqTokenKind {
			global := m["app"]
			global.AppID, global.Scope = s.appID, "app"
			if global.Until == nil || until.After(*global.Until) {
				global.Until = &until
			}
			return s.repo.SaveCooldown(ctx, tx, &global)
		}
		return nil
	}
	if e.Kind == qqapi.TemporaryNotApplied || e.Kind == qqapi.Ambiguous {
		row.Failures = min(row.Failures+1, 3)
		if row.Failures >= 3 {
			until := now.Add(time.Minute)
			if row.Until == nil || until.After(*row.Until) {
				row.Until = &until
			}
		}
		return s.repo.SaveCooldown(ctx, tx, &row)
	}
	return nil
}

// CleanupQQOfficial needs no credentials; switching or removing an AppID must
// not strand applicant data or expired queue payloads in the database.
func CleanupQQOfficial(ctx context.Context, db *gorm.DB) error {
	if db == nil {
		return ErrQQOfficialUnavailable
	}
	repo := repository.NewQQOfficialRepository(db)
	ids, err := repo.AppIDs(ctx)
	if err != nil {
		return QQOfficialPublicError(err)
	}
	for _, id := range ids {
		s := &QQOfficialService{appID: id, repo: repo, now: time.Now}
		err = s.locked(ctx, nil, func(tx *gorm.DB, app *model.QQOfficialApp, now time.Time) error {
			if app.LeaseID != "" && app.LeaseUntil != nil && !app.LeaseUntil.After(now) {
				if err := s.recoverAttempt(ctx, tx, app, now); err != nil {
					return err
				}
			}
			if err := repo.Expire(ctx, tx, id, now); err != nil {
				return err
			}
			return repo.Cleanup(ctx, tx, id, now)
		})
		if err != nil {
			return QQOfficialPublicError(err)
		}
	}
	return nil
}

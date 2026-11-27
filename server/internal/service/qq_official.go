package service

import (
	"amiya-eden/internal/model"
	"amiya-eden/internal/repository"
	"amiya-eden/internal/service/internal/qqapi"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

const (
	qqSourceTest        = "test"
	qqSourceVerify      = "verification"
	qqSourceRefresh     = "join_refresh"
	qqSourceDecision    = "join_decision"
	QQSourceFleet       = "fleet"
	QQSourceStructure   = "structure"
	qqUnknownCapability = "unknown"
	qqAllowedPush       = "allowed"
	qqBlockedPush       = "blocked"
	qqAvailable         = "available"
	qqForbidden         = "forbidden"
	qqMaxGroups         = 2500
	qqQueueMax          = 2000
	qqGroupQueueMax     = 100
	qqEventMaxPending   = 4000
	qqTestText          = "AmiyaEden：QQ 官方机器人测试消息。"
)

var (
	ErrQQOfficialUnavailable   = errors.New("unavailable")
	ErrQQOfficialDisabled      = errors.New("disabled")
	ErrQQOfficialInvalid       = errors.New("invalid_input")
	ErrQQOfficialNotFound      = errors.New("not_found")
	ErrQQOfficialConflict      = errors.New("conflict")
	ErrQQOfficialCapacity      = errors.New("queue_full")
	ErrQQOfficialNotAuthorized = errors.New("group_not_authorized")
	ErrQQOfficialNoTargets     = errors.New("no_targets")
	ErrQQOfficialExpired       = errors.New("expired")
)

type qqOfficialClient interface {
	Refresh(context.Context) (qqapi.Token, error)
	Execute(context.Context, string, qqapi.Request) (qqapi.Result, error)
}

type qqOfficialGateway interface {
	DiscoverGateway(context.Context, string) (string, error)
	RunGateway(context.Context, string, string, bool, *qqapi.GatewaySession, func(context.Context) error, func(context.Context, qqapi.Event) error) error
}

type QQOfficialService struct {
	appID          string
	approvalWrites bool
	configRevision int64 // nonzero for a database-managed credential snapshot
	repo           *repository.QQOfficialRepository
	client         qqOfficialClient
	gateway        qqOfficialGateway
	audit          *AuditService
	now            func() time.Time
	tokenMu        sync.Mutex
	token          qqapi.Token
	tokenEpoch     int64
	wake           chan struct{}
}

func NewQQOfficialService(db *gorm.DB, appID, secret string, approvalWrites bool) *QQOfficialService {
	appID, secret = strings.TrimSpace(appID), strings.TrimSpace(secret)
	if db == nil || !qqDecimalAppID(appID) || len(secret) == 0 || len(secret) > 4096 {
		return nil
	}
	client := qqapi.New(appID, secret)
	return &QQOfficialService{appID: appID, approvalWrites: approvalWrites, repo: repository.NewQQOfficialRepository(db),
		client: client, gateway: client, audit: NewAuditService(), now: time.Now, wake: make(chan struct{}, 1)}
}

func qqDecimalAppID(value string) bool {
	if value == "" || len(value) > 32 {
		return false
	}
	for _, c := range value {
		if c < '0' || c > '9' {
			return false
		}
	}
	return true
}

func (s *QQOfficialService) signal() {
	select {
	case s.wake <- struct{}{}:
	default:
	}
}

func (s *QQOfficialService) locked(ctx context.Context, txn *gorm.DB, run func(*gorm.DB, *model.QQOfficialApp, time.Time) error) error {
	if s == nil {
		return ErrQQOfficialUnavailable
	}
	return s.repo.Transaction(ctx, txn, func(tx *gorm.DB) error {
		if err := s.checkInstallation(ctx, tx); err != nil {
			return err
		}
		return s.withAppLock(ctx, tx, run)
	})
}

func (s *QQOfficialService) checkInstallation(ctx context.Context, tx *gorm.DB) error {
	if s.configRevision == 0 { // isolated fixtures and credential-independent cleanup
		return nil
	}
	row, err := s.repo.Installation(ctx, tx, "SHARE")
	if err != nil || row.AppID != s.appID || row.Revision != s.configRevision {
		return ErrQQOfficialUnavailable
	}
	return nil
}

func (s *QQOfficialService) withAppLock(ctx context.Context, tx *gorm.DB, run func(*gorm.DB, *model.QQOfficialApp, time.Time) error) error {
	app, err := s.repo.App(ctx, tx, s.appID, true)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		if err = s.repo.EnsureApp(ctx, tx, s.appID); err != nil {
			return err
		}
		app, err = s.repo.App(ctx, tx, s.appID, true)
	}
	if err != nil {
		return err
	}
	now, err := s.repo.Clock(ctx, tx, s.now())
	if err != nil {
		return err
	}
	return run(tx, &app, now)
}

func qqOfficialReadError(err error) error {
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return ErrQQOfficialNotFound
	}
	return err
}

func qqHash(raw string) string { sum := sha256.Sum256([]byte(raw)); return hex.EncodeToString(sum[:]) }

func qqOperationKey(id string) (string, error) {
	u, err := uuid.Parse(id)
	if err != nil || u == uuid.Nil {
		return "", ErrQQOfficialInvalid
	}
	return u.String(), nil
}

func qqValidOpenID(value string) bool {
	if len(value) < 16 || len(value) > 128 {
		return false
	}
	for _, r := range value {
		if r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || r == '-' || r == '_' {
			continue
		}
		return false
	}
	return true
}

func qqNewGroup(app, openID string, now time.Time) model.QQOfficialGroup {
	return model.QQOfficialGroup{ID: uuid.NewString(), AppID: app, OpenID: openID, PushState: qqUnknownCapability, ApprovalCapability: qqUnknownCapability,
		Generation: 1, NotifyGeneration: 1, ApprovalGeneration: 1, CreatedAt: now}
}

func (s *QQOfficialService) record(ctx context.Context, tx *gorm.DB, actor uint, action, resource, id string, details map[string]any) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	return s.audit.RecordRequiredEventTx(tx, AuditRecordInput{Category: "qq_official", Action: action,
		ActorUserID: actor, ResourceType: resource, ResourceID: id, Result: model.AuditResultSuccess, Details: details})
}

type QQOfficialSettingsUpdate struct {
	Enabled, Paused, Certified, ApprovalWritesEnabled *bool
	AppID, AppSecret                                  *string
}

type QQOfficialGroupUpdate struct {
	Alias                                            *string
	Authorized, NotificationEnabled, ApprovalEnabled *bool
	FleetCorporationID, StructureCorporationID       *int64
}

type QQOfficialStatus struct {
	Configured, Enabled, Paused, Certified, ApprovalWritesEnabled bool
	AppID                                                         string
	HasAppSecret                                                  bool
	TokenState, GatewayState                                      string
	LastEventAt, NextRequestAt, OldestQueuedAt                    *time.Time
	QueueDepth, AttemptsLastMinute, MessagesLastMinute            int64
	MessageBudget                                                 int
	Cooldowns                                                     []model.QQOfficialCooldown
}

func qqIdleStatus() QQOfficialStatus {
	return QQOfficialStatus{TokenState: "idle", GatewayState: qqGatewayIdle, MessageBudget: 24, Cooldowns: []model.QQOfficialCooldown{}}
}

func (s *QQOfficialService) Status(ctx context.Context) (QQOfficialStatus, error) {
	view := qqIdleStatus()
	if s == nil {
		return view, nil
	}
	view.Configured, view.ApprovalWritesEnabled = true, s.approvalWrites
	view.AppID, view.HasAppSecret = s.appID, true
	app, err := s.repo.App(ctx, nil, s.appID, false)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return view, nil
	}
	if err != nil {
		return view, err
	}
	now, err := s.repo.Clock(ctx, nil, s.now())
	if err != nil {
		return view, err
	}
	view.Enabled, view.Paused, view.Certified = app.Enabled, app.Paused, app.Certified
	view.LastEventAt, view.TokenState = app.LastEventAt, app.TokenState
	if app.GatewayRevision == s.configRevision {
		switch {
		case app.GatewayState == qqGatewayBlocked:
			view.GatewayState = qqGatewayBlocked
		case qqOwnsGateway(&app, app.GatewayLeaseID, now):
			view.GatewayState = app.GatewayState
		case app.GatewayState != "":
			view.GatewayState = qqGatewayRetrying
		}
	}
	if app.GlobalHold != "" {
		view.TokenState = app.GlobalHold
	}
	if view.TokenState == "ready" && (app.TokenExpiresAt == nil || !app.TokenExpiresAt.After(now)) {
		view.TokenState = "expired"
	}
	view.MessageBudget = qqMessageBudget(app.Certified)
	if view.QueueDepth, err = s.repo.QueueDepth(ctx, nil, s.appID, "", now); err != nil {
		return view, err
	}
	if view.OldestQueuedAt, err = s.repo.OldestQueued(ctx, s.appID, now); err != nil {
		return view, err
	}
	if view.AttemptsLastMinute, err = s.repo.CountAttempts(ctx, s.appID, "", now.Add(-time.Minute)); err != nil {
		return view, err
	}
	if view.MessagesLastMinute, err = s.repo.CountAttempts(ctx, s.appID, qqapi.SendText, now.Add(-time.Minute)); err != nil {
		return view, err
	}
	for _, t := range []*time.Time{app.NextAdmissionAt, app.LeaseUntil} {
		if t != nil && t.After(now) && (view.NextRequestAt == nil || t.After(*view.NextRequestAt)) {
			view.NextRequestAt = t
		}
	}
	limitUntil, err := s.window(ctx, nil, "", "", now, 48, time.Minute)
	if err != nil {
		return view, err
	}
	if limitUntil.After(now) && (view.NextRequestAt == nil || limitUntil.After(*view.NextRequestAt)) {
		view.NextRequestAt = &limitUntil
	}
	rows, err := s.repo.Cooldowns(ctx, nil, s.appID)
	if err != nil {
		return view, err
	}
	for _, row := range rows {
		if row.Until != nil && row.Until.After(now) {
			view.Cooldowns = append(view.Cooldowns, row)
		}
		if row.Scope == "app" && row.Until != nil && row.Until.After(now) && (view.NextRequestAt == nil || row.Until.After(*view.NextRequestAt)) {
			view.NextRequestAt = row.Until
		}
	}
	return view, nil
}

func (s *QQOfficialService) UpdateSettings(ctx context.Context, actor uint, in QQOfficialSettingsUpdate) (QQOfficialStatus, error) {
	err := s.locked(ctx, nil, func(tx *gorm.DB, app *model.QQOfficialApp, now time.Time) error {
		changed, err := s.applySettings(ctx, tx, app, now, in)
		if err != nil || !changed {
			return err
		}
		return s.record(ctx, tx, actor, "qq_official_settings", "qq_official_app", "", qqSettingsAudit(app))
	})
	if err != nil {
		return QQOfficialStatus{}, err
	}
	s.signal()
	return s.Status(ctx)
}

func qqSettingsAudit(app *model.QQOfficialApp) map[string]any {
	return map[string]any{"enabled": app.Enabled, "paused": app.Paused, "certified": app.Certified}
}

func (s *QQOfficialService) applySettings(ctx context.Context, tx *gorm.DB, app *model.QQOfficialApp, now time.Time, in QQOfficialSettingsUpdate) (bool, error) {
	changed := false
	if in.Enabled != nil && *in.Enabled != app.Enabled {
		app.Enabled, changed = *in.Enabled, true
		if !app.Enabled {
			app.Generation++
			if err := s.repo.InvalidateQueued(ctx, tx, app.AppID, "", "", "disabled", now); err != nil {
				return false, err
			}
		} else if app.GlobalHold != "" {
			app.GlobalHold, app.TokenState = "", "idle"
			app.TokenGeneration++
		}
	}
	if in.Paused != nil && *in.Paused != app.Paused {
		app.Paused, changed = *in.Paused, true
	}
	if in.Certified != nil && *in.Certified != app.Certified {
		app.Certified, changed = *in.Certified, true
	}
	if !changed {
		return false, nil
	}
	return true, s.repo.SaveApp(ctx, tx, app)
}

func (s *QQOfficialService) Groups(ctx context.Context, offset, limit int) ([]model.QQOfficialGroup, int64, error) {
	if s == nil {
		return []model.QQOfficialGroup{}, 0, nil
	}
	return s.repo.Groups(ctx, s.appID, offset, limit)
}

func (s *QQOfficialService) UpdateGroup(ctx context.Context, actor uint, id string, in QQOfficialGroupUpdate) (model.QQOfficialGroup, error) {
	var result model.QQOfficialGroup
	err := s.locked(ctx, nil, func(tx *gorm.DB, _ *model.QQOfficialApp, now time.Time) error {
		group, err := s.repo.Group(ctx, tx, s.appID, id)
		if err != nil {
			return qqOfficialReadError(err)
		}
		before := group
		if in.Alias != nil {
			label := strings.TrimSpace(*in.Alias)
			if !utf8.ValidString(label) || utf8.RuneCountInString(label) > 80 || strings.ContainsFunc(label, unicode.IsControl) {
				return ErrQQOfficialInvalid
			}
			group.Alias = label
		}
		if in.Authorized != nil {
			group.Authorized = *in.Authorized
		}
		if in.NotificationEnabled != nil {
			group.NotificationEnabled = *in.NotificationEnabled
		}
		if in.ApprovalEnabled != nil {
			group.ApprovalEnabled = *in.ApprovalEnabled
		}
		if in.FleetCorporationID != nil {
			if *in.FleetCorporationID < 0 {
				return ErrQQOfficialInvalid
			}
			group.FleetCorporation = *in.FleetCorporationID
		}
		if in.StructureCorporationID != nil {
			if *in.StructureCorporationID < 0 {
				return ErrQQOfficialInvalid
			}
			group.StructureCorporation = *in.StructureCorporationID
		}
		if !group.Authorized {
			if in.NotificationEnabled != nil && *in.NotificationEnabled || in.ApprovalEnabled != nil && *in.ApprovalEnabled {
				return ErrQQOfficialNotAuthorized
			}
			group.NotificationEnabled, group.ApprovalEnabled, group.TestVerified = false, false, false
		}
		if group.Authorized && (!group.Present || group.Alias == "") {
			return ErrQQOfficialNotAuthorized
		}
		if group.NotificationEnabled && (!group.TestVerified || group.PushState != qqAllowedPush) {
			return ErrQQOfficialNotAuthorized
		}
		if in.ApprovalEnabled != nil && *in.ApprovalEnabled && (!s.approvalWrites || group.ApprovalCapability != qqAvailable) {
			return ErrQQOfficialNotAuthorized
		}
		if group == before {
			result = group
			return nil
		}
		if before.Authorized && !group.Authorized {
			group.Generation++
			if err := s.repo.InvalidateQueued(ctx, tx, s.appID, id, "", "group_disabled", now); err != nil {
				return err
			}
		} else {
			if before.NotificationEnabled && !group.NotificationEnabled || group.FleetCorporation != before.FleetCorporation || group.StructureCorporation != before.StructureCorporation {
				group.NotifyGeneration++
				if err := s.repo.InvalidateQueued(ctx, tx, s.appID, id, "notification", "group_changed", now); err != nil {
					return err
				}
			}
			if before.ApprovalEnabled && !group.ApprovalEnabled {
				group.ApprovalGeneration++
				if err := s.repo.InvalidateQueued(ctx, tx, s.appID, id, "approval", "group_changed", now); err != nil {
					return err
				}
			}
		}
		if err := s.repo.SaveGroup(ctx, tx, &group); err != nil {
			return err
		}
		result = group
		return s.record(ctx, tx, actor, "qq_official_group_update", "qq_official_group", id, map[string]any{"authorized": group.Authorized, "notification_enabled": group.NotificationEnabled, "approval_enabled": group.ApprovalEnabled, "fleet_corporation_id": group.FleetCorporation, "structure_corporation_id": group.StructureCorporation})
	})
	return result, err
}

type qqIntentInput struct {
	key, digest, kind, source, reference, text, decision, application string
	operator                                                          uint
	expires                                                           time.Time
	urgent                                                            bool
}

func (s *QQOfficialService) previous(ctx context.Context, tx *gorm.DB, in qqIntentInput) ([]model.QQOfficialTask, error) {
	intent, err := s.repo.Intent(ctx, tx, s.appID, qqHash(in.key))
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if intent.Digest != in.digest {
		return nil, ErrQQOfficialConflict
	}
	if intent.MergedTaskID != "" {
		row, err := s.repo.Task(ctx, tx, s.appID, intent.MergedTaskID)
		if err != nil {
			return nil, qqOfficialReadError(err)
		}
		return []model.QQOfficialTask{row}, nil
	}
	return s.repo.IntentTasks(ctx, tx, s.appID, intent.ID)
}

func (s *QQOfficialService) insert(ctx context.Context, tx *gorm.DB, app *model.QQOfficialApp, now time.Time, in qqIntentInput, groups []model.QQOfficialGroup) ([]model.QQOfficialTask, error) {
	if !app.Enabled {
		return nil, ErrQQOfficialDisabled
	}
	if app.GlobalHold != "" {
		return nil, ErrQQOfficialUnavailable
	}
	if !in.expires.After(now) {
		return nil, ErrQQOfficialExpired
	}
	if len(groups) == 0 {
		return nil, ErrQQOfficialNoTargets
	}
	maxApp, maxGroup := qqQueueMax, qqGroupQueueMax
	if !in.urgent {
		maxApp, maxGroup = maxApp*4/5, maxGroup*4/5
	}
	count, err := s.repo.QueueDepth(ctx, tx, s.appID, "", now)
	if err != nil {
		return nil, err
	}
	if count+int64(len(groups)) > int64(maxApp) {
		return nil, ErrQQOfficialCapacity
	}
	intent := model.QQOfficialIntent{ID: uuid.NewString(), AppID: s.appID, Key: qqHash(in.key), Digest: in.digest, CreatedAt: now}
	tasks := make([]model.QQOfficialTask, 0, len(groups))
	for _, group := range groups {
		count, err = s.repo.QueueDepth(ctx, tx, s.appID, group.ID, now)
		if err != nil {
			return nil, err
		}
		if count >= int64(maxGroup) {
			return nil, ErrQQOfficialCapacity
		}
		modeGen := group.NotifyGeneration
		if in.kind == qqapi.JoinList || in.kind == qqapi.JoinDecision {
			modeGen = group.ApprovalGeneration
		}
		tasks = append(tasks, model.QQOfficialTask{ID: uuid.NewString(), AppID: s.appID, IntentID: intent.ID, GroupID: group.ID,
			AppGeneration: app.Generation, GroupGeneration: group.Generation, ModeGeneration: modeGen,
			Kind: in.kind, Source: in.source, Reference: in.reference, Text: in.text, Decision: in.decision, ApplicationID: in.application, OperatorID: in.operator,
			Urgent: in.urgent, Status: model.QQQueued, CreatedAt: now, NextRunAt: now, ExpiresAt: in.expires})
	}
	if err := s.repo.InsertIntent(ctx, tx, &intent, tasks); err != nil {
		return nil, err
	}
	return tasks, nil
}

// QQOfficialNotification is an internal business capability, not an API proxy.
type QQOfficialNotification struct {
	Source, StableKey, Reference, Text string
	CorporationID                      int64
	ExpiresAt                          time.Time
}

func (s *QQOfficialService) SubmitNotification(ctx context.Context, txn *gorm.DB, in QQOfficialNotification) ([]model.QQOfficialTask, error) {
	if (in.Source != QQSourceFleet && in.Source != QQSourceStructure) || in.StableKey == "" || len(in.StableKey) > 256 || in.Reference == "" || len(in.Reference) > 128 || in.CorporationID <= 0 || !utf8.ValidString(in.Text) {
		return nil, ErrQQOfficialInvalid
	}
	text := strings.TrimSpace(strings.NewReplacer("<", "＜", ">", "＞").Replace(in.Text))
	if text == "" || len(text) > 1024 {
		return nil, ErrQQOfficialInvalid
	}
	// The caller's absolute deadline is immutable. Normalize the PostgreSQL
	// precision so a business timestamp read back from storage hashes equally.
	metadata, _ := json.Marshal([]string{in.Reference, text, strconv.FormatInt(in.CorporationID, 10), in.ExpiresAt.UTC().Truncate(time.Microsecond).Format(time.RFC3339Nano)})
	input := qqIntentInput{key: in.Source + ":" + in.StableKey, digest: qqHash(string(metadata)), kind: qqapi.SendText, source: in.Source, reference: in.Reference, text: text, expires: in.ExpiresAt.UTC(), urgent: in.Source == QQSourceFleet}
	var result []model.QQOfficialTask
	err := s.locked(ctx, txn, func(tx *gorm.DB, app *model.QQOfficialApp, now time.Time) error {
		rows, err := s.previous(ctx, tx, input)
		if err != nil || rows != nil {
			result = rows
			return err
		}
		if input.expires.After(now.Add(6 * time.Hour)) {
			return ErrQQOfficialInvalid
		}
		groups, err := s.repo.TargetGroups(ctx, tx, s.appID, in.Source, in.CorporationID)
		if err != nil {
			return err
		}
		result, err = s.insert(ctx, tx, app, now, input, groups)
		return err
	})
	if err == nil && txn == nil {
		s.signal()
	}
	return result, err
}

type QQOfficialTaskFilter = repository.QQOfficialTaskFilter

func (s *QQOfficialService) Tasks(ctx context.Context, filter QQOfficialTaskFilter, offset, limit int) ([]model.QQOfficialTask, int64, error) {
	if s == nil {
		return []model.QQOfficialTask{}, 0, nil
	}
	return s.repo.Tasks(ctx, s.appID, filter, offset, limit)
}

func (s *QQOfficialService) FleetTasks(ctx context.Context, fleetID string, offset, limit int) ([]model.QQOfficialTask, int64, error) {
	if s == nil {
		return []model.QQOfficialTask{}, 0, nil
	}
	return s.repo.Tasks(ctx, s.appID, QQOfficialTaskFilter{Source: QQSourceFleet, Reference: fleetID}, offset, limit)
}

func (s *QQOfficialService) Task(ctx context.Context, id string) (model.QQOfficialTask, []model.QQOfficialAttempt, error) {
	if s == nil {
		return model.QQOfficialTask{}, nil, ErrQQOfficialNotFound
	}
	row, err := s.repo.Task(ctx, nil, s.appID, id)
	if err != nil {
		return row, nil, qqOfficialReadError(err)
	}
	attempts, err := s.repo.Attempts(ctx, s.appID, id)
	return row, attempts, err
}

func (s *QQOfficialService) Cancel(ctx context.Context, actor uint, id string) (model.QQOfficialTask, []model.QQOfficialAttempt, error) {
	err := s.locked(ctx, nil, func(tx *gorm.DB, _ *model.QQOfficialApp, now time.Time) error {
		row, err := s.repo.Task(ctx, tx, s.appID, id)
		if err != nil {
			return qqOfficialReadError(err)
		}
		if row.Status == model.QQCancelled {
			return nil
		}
		if row.Status != model.QQQueued && row.Status != model.QQRetryWait {
			return ErrQQOfficialConflict
		}
		qqTerminal(&row, model.QQCancelled, "operator_cancelled", now)
		if err := s.repo.SaveTask(ctx, tx, &row); err != nil {
			return err
		}
		if err := s.projectDecision(ctx, tx, row); err != nil {
			return err
		}
		return s.record(ctx, tx, actor, "qq_official_task_cancel", "qq_official_task", id, nil)
	})
	if err != nil {
		return model.QQOfficialTask{}, nil, err
	}
	return s.Task(ctx, id)
}

func qqTerminal(row *model.QQOfficialTask, state, reason string, now time.Time) {
	row.Status, row.Reason, row.FinishedAt = state, reason, &now
	row.Text, row.Cursor, row.VisitedCursors = "", "", nil
}

func qqOfficialUnavailableOnError(err error) error {
	if err == nil {
		return nil
	}
	for _, typed := range []error{ErrQQOfficialUnavailable, ErrQQOfficialDisabled, ErrQQOfficialInvalid, ErrQQOfficialNotFound, ErrQQOfficialConflict, ErrQQOfficialCapacity, ErrQQOfficialNotAuthorized, ErrQQOfficialNoTargets, ErrQQOfficialExpired} {
		if errors.Is(err, typed) {
			return err
		}
	}
	return fmt.Errorf("%w: operation could not be persisted", ErrQQOfficialUnavailable)
}

// QQOfficialPublicError masks SQL and provider values before exposing errors to callers.
func QQOfficialPublicError(err error) error { return qqOfficialUnavailableOnError(err) }

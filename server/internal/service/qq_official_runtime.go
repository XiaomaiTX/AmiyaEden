package service

import (
	"amiya-eden/internal/model"
	"amiya-eden/internal/repository"
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/hkdf"
	"crypto/sha256"
	"crypto/subtle"
	"errors"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"go.uber.org/zap"
	"gorm.io/gorm"
)

const qqCredentialDomain = "amiya-eden/qq-official/app-secret/aes256-gcm/v1"

// QQOfficialRuntime is the stable entry point, even when an installation has
// no credentials. Each entry reads PostgreSQL; the cache only avoids rebuilding
// the platform client and losing its in-memory access token every poll.
type QQOfficialRuntime struct {
	db    *gorm.DB
	repo  *repository.QQOfficialRepository
	seal  cipher.AEAD
	wake  chan struct{}
	mu    sync.Mutex
	cache *QQOfficialService
}

var defaultQQOfficialRuntime atomic.Pointer[QQOfficialRuntime]

func SetDefaultQQOfficialRuntime(r *QQOfficialRuntime) { defaultQQOfficialRuntime.Store(r) }
func DefaultQQOfficialRuntime() *QQOfficialRuntime     { return defaultQQOfficialRuntime.Load() }

func NewQQOfficialRuntime(db *gorm.DB, deploymentJWTSecret string) *QQOfficialRuntime {
	if db == nil {
		return nil
	}
	r := &QQOfficialRuntime{db: db, repo: repository.NewQQOfficialRepository(db), wake: make(chan struct{}, 1)}
	if keyInput := strings.TrimSpace(deploymentJWTSecret); keyInput != "" && keyInput != "change_me_in_production" {
		key, err := hkdf.Key(sha256.New, []byte(deploymentJWTSecret), nil, qqCredentialDomain, 32)
		if err == nil {
			if block, blockErr := aes.NewCipher(key); blockErr == nil {
				r.seal, _ = cipher.NewGCMWithRandomNonce(block)
			}
			clear(key)
		}
	}
	return r
}

func qqCredentialAAD(appID string) []byte { return []byte(qqCredentialDomain + ":" + appID) }

func (r *QQOfficialRuntime) decrypt(row model.QQOfficialInstallation) ([]byte, error) {
	if r.seal == nil || !qqDecimalAppID(row.AppID) || len(row.SecretCiphertext) == 0 {
		return nil, ErrQQOfficialUnavailable
	}
	secret, err := r.seal.Open(nil, nil, row.SecretCiphertext, qqCredentialAAD(row.AppID))
	if err != nil {
		return nil, ErrQQOfficialUnavailable
	}
	return secret, nil
}

func (r *QQOfficialRuntime) clientFor(row model.QQOfficialInstallation) (*QQOfficialService, error) {
	if row.AppID == "" && len(row.SecretCiphertext) == 0 {
		return nil, nil
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.cache != nil && r.cache.appID == row.AppID && r.cache.configRevision == row.Revision {
		return r.cache, nil
	}
	secret, err := r.decrypt(row)
	if err != nil {
		return nil, err
	}
	defer clear(secret)
	s := NewQQOfficialService(r.db, row.AppID, string(secret), row.ApprovalWritesEnabled)
	if s == nil || row.Revision <= 0 {
		return nil, ErrQQOfficialUnavailable
	}
	s.configRevision, s.wake = row.Revision, r.wake
	if r.cache == nil || r.cache.configRevision < row.Revision {
		r.cache = s
	}
	return s, nil
}

func (r *QQOfficialRuntime) installation(ctx context.Context, tx *gorm.DB) (model.QQOfficialInstallation, error) {
	row, err := r.repo.Installation(ctx, tx, "")
	if errors.Is(err, gorm.ErrRecordNotFound) {
		err = nil
	}
	return row, err
}

func (r *QQOfficialRuntime) Active(ctx context.Context, tx *gorm.DB) (*QQOfficialService, error) {
	if r == nil {
		return nil, nil
	}
	row, err := r.installation(ctx, tx)
	if err != nil {
		return nil, ErrQQOfficialUnavailable
	}
	return r.clientFor(row)
}

func (r *QQOfficialRuntime) Status(ctx context.Context) (QQOfficialStatus, error) {
	if r == nil {
		return qqIdleStatus(), nil
	}
	row, err := r.installation(ctx, nil)
	if err != nil {
		return qqIdleStatus(), ErrQQOfficialUnavailable
	}
	s, err := r.clientFor(row)
	if err != nil {
		view := qqIdleStatus()
		view.AppID, view.HasAppSecret = row.AppID, len(row.SecretCiphertext) != 0
		view.TokenState, view.ApprovalWritesEnabled = "credentials_unavailable", row.ApprovalWritesEnabled
		// Show the persisted switches so operators can still turn them off
		// while fixing inconsistent deployment keys across replicas.
		app, err := r.repo.App(ctx, nil, row.AppID, false)
		if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			return view, ErrQQOfficialUnavailable
		}
		view.Enabled, view.Paused, view.Certified = app.Enabled, app.Paused, app.Certified
		return view, nil
	}
	view, err := s.Status(ctx)
	return view, QQOfficialPublicError(err)
}

func (r *QQOfficialRuntime) Run(ctx context.Context, logger *zap.Logger) error {
	if r == nil {
		return nil
	}
	if logger == nil {
		logger = zap.NewNop()
	}
	ticker := time.NewTicker(qqPollInterval)
	defer ticker.Stop()
	var lastError time.Time
	for ctx.Err() == nil {
		cycle, cancel := context.WithTimeout(ctx, qqHTTPBudget)
		s, err := r.Active(cycle, nil)
		if err == nil && s != nil {
			_, err = s.dispatchOnce(cycle)
		}
		cancel()
		if err != nil && ctx.Err() == nil && time.Since(lastError) > 30*time.Second {
			logger.Warn("QQ official dispatch paused: configuration or state unavailable")
			lastError = time.Now()
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
		case <-r.wake:
		}
	}
	return ctx.Err()
}

func (r *QQOfficialRuntime) UpdateSettings(ctx context.Context, actor uint, in QQOfficialSettingsUpdate) (QQOfficialStatus, error) {
	if r == nil {
		return qqIdleStatus(), ErrQQOfficialUnavailable
	}
	if !qqValidSettings(in) {
		return qqIdleStatus(), ErrQQOfficialInvalid
	}
	err := r.repo.Transaction(ctx, nil, func(tx *gorm.DB) error {
		row, err := r.repo.Installation(ctx, tx, "UPDATE")
		if errors.Is(err, gorm.ErrRecordNotFound) {
			if err = r.repo.EnsureInstallation(ctx, tx); err != nil {
				return err
			}
			row, err = r.repo.Installation(ctx, tx, "UPDATE")
		}
		if err != nil {
			return err
		}
		return r.updateInstallation(ctx, tx, &row, actor, in)
	})
	if err != nil {
		return qqIdleStatus(), QQOfficialPublicError(err)
	}
	select {
	case r.wake <- struct{}{}:
	default:
	}
	return r.Status(ctx)
}

func qqValidSettings(in QQOfficialSettingsUpdate) bool {
	if in.AppID != nil && !qqDecimalAppID(*in.AppID) {
		return false
	}
	if in.AppSecret != nil {
		secret := *in.AppSecret
		// Official app secrets are visible ASCII. Reject whitespace rather than
		// silently storing a different credential than the operator provided.
		if len(secret) == 0 || len(secret) > 4096 {
			return false
		}
		for i := range len(secret) {
			if secret[i] < '!' || secret[i] > '~' {
				return false
			}
		}
	}
	return in.AppID != nil || in.AppSecret != nil || in.Enabled != nil || in.Paused != nil || in.Certified != nil || in.ApprovalWritesEnabled != nil
}

func qqTrue(value *bool) bool { return value != nil && *value }

func (r *QQOfficialRuntime) updateInstallation(ctx context.Context, tx *gorm.DB, row *model.QQOfficialInstallation, actor uint, in QQOfficialSettingsUpdate) error {
	id := row.AppID
	if in.AppID != nil {
		id = *in.AppID
	}
	if id == "" || (row.AppID != id && in.AppSecret == nil) {
		return ErrQQOfficialConflict
	}
	previous, decryptErr := r.decrypt(*row)
	defer clear(previous)
	rotating := in.AppSecret != nil && (row.AppID != id || decryptErr != nil || subtle.ConstantTimeCompare(previous, []byte(*in.AppSecret)) != 1)
	if rotating {
		if qqTrue(in.Enabled) || qqTrue(in.ApprovalWritesEnabled) {
			return ErrQQOfficialConflict
		}
		if r.seal == nil {
			return ErrQQOfficialUnavailable
		}
	} else if decryptErr != nil && (qqTrue(in.Enabled) || qqTrue(in.ApprovalWritesEnabled)) {
		return ErrQQOfficialUnavailable
	}
	s := &QQOfficialService{appID: id, repo: r.repo, audit: NewAuditService(), now: time.Now}
	if rotating && row.AppID != "" && row.AppID != id {
		old := &QQOfficialService{appID: row.AppID, repo: r.repo, now: time.Now}
		if err := old.withAppLock(ctx, tx, old.revokeCredentials(ctx, false)); err != nil {
			return err
		}
	}
	if rotating {
		if err := s.withAppLock(ctx, tx, s.revokeCredentials(ctx, true)); err != nil {
			return err
		}
		secret := []byte(*in.AppSecret)
		defer clear(secret)
		row.SecretCiphertext = r.seal.Seal(nil, nil, secret, qqCredentialAAD(id))
		row.AppID, row.ApprovalWritesEnabled = id, false
	}
	approvalChanged := in.ApprovalWritesEnabled != nil && *in.ApprovalWritesEnabled != row.ApprovalWritesEnabled
	if approvalChanged {
		row.ApprovalWritesEnabled = *in.ApprovalWritesEnabled
	}
	if rotating || approvalChanged {
		row.Revision++
		if err := r.repo.SaveInstallation(ctx, tx, row); err != nil {
			return err
		}
	}
	return s.withAppLock(ctx, tx, func(tx *gorm.DB, app *model.QQOfficialApp, now time.Time) error {
		gatewayReset := !rotating && decryptErr == nil && (in.AppID != nil || in.AppSecret != nil) && app.GatewayState == qqGatewayBlocked
		if gatewayReset {
			app.GatewayState = ""
			if app.GlobalHold != "" {
				app.GlobalHold, app.TokenState = "", "idle"
				app.TokenGeneration++
			}
			if err := r.repo.SaveApp(ctx, tx, app); err != nil {
				return err
			}
		}
		if approvalChanged && !row.ApprovalWritesEnabled {
			if err := r.repo.InvalidateQueued(ctx, tx, id, "", "approval_write", "approval_disabled", now); err != nil {
				return err
			}
		}
		changed, err := s.applySettings(ctx, tx, app, now, in)
		if err != nil || (!changed && !rotating && !approvalChanged && !gatewayReset) {
			return err
		}
		details := qqSettingsAudit(app)
		details["approval_writes_enabled"] = row.ApprovalWritesEnabled
		details["credentials_changed"] = rotating
		details["gateway_reset"] = gatewayReset
		return s.record(ctx, tx, actor, "qq_official_settings", "qq_official_app", "", details)
	})
}

func (s *QQOfficialService) revokeCredentials(ctx context.Context, entering bool) func(*gorm.DB, *model.QQOfficialApp, time.Time) error {
	return func(tx *gorm.DB, app *model.QQOfficialApp, now time.Time) error {
		if app.LeaseID != "" {
			if app.LeaseUntil == nil || app.LeaseUntil.After(now) {
				return ErrQQOfficialConflict
			}
			if err := s.recoverAttempt(ctx, tx, app, now); err != nil {
				return err
			}
		}
		app.Enabled = false
		app.Generation++
		app.TokenGeneration++
		app.TokenState, app.TokenExpiresAt = "idle", nil
		if entering {
			app.LastEventAt, app.GlobalHold = nil, ""
		}
		if err := s.repo.InvalidateQueued(ctx, tx, app.AppID, "", "", "configuration_changed", now); err != nil {
			return err
		}
		return s.repo.SaveApp(ctx, tx, app)
	}
}

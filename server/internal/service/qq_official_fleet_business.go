package service

import (
	"amiya-eden/internal/model"
	"context"
	"errors"
	"time"

	"gorm.io/gorm"
)

// ErrFleetQQForbidden is distinct from an unavailable QQ group: the caller
// cannot view the fleet ledger or submit on behalf of this fleet.
var ErrFleetQQForbidden = errors.New("forbidden")

func (s *FleetService) persistFleetWithOptionalQQ(ctx context.Context, fleet *model.Fleet, sendQQ bool) (int, error) {
	if !sendQQ {
		return 0, s.repo.Create(fleet)
	}
	var qq *QQOfficialService
	var tasks []model.QQOfficialTask
	err := s.repo.Transaction(ctx, func(tx *gorm.DB) error {
		var err error
		qq, err = DefaultQQOfficialRuntime().Active(ctx, tx)
		if err != nil {
			return err
		}
		if qq == nil {
			return ErrQQOfficialUnavailable
		}
		return qq.locked(ctx, tx, func(tx *gorm.DB, _ *model.QQOfficialApp, now time.Time) error {
			if err := s.repo.CreateTx(tx, fleet); err != nil {
				return err
			}
			var err error
			tasks, err = qq.queueFleetInTransaction(ctx, tx, *fleet, "created:"+fleet.ID, now)
			return err
		})
	})
	if err != nil {
		return 0, QQOfficialPublicError(err)
	}
	qq.signal()
	return len(tasks), nil
}

func (s *QQOfficialService) queueFleetInTransaction(ctx context.Context, tx *gorm.DB, fleet model.Fleet, key string, now time.Time) ([]model.QQOfficialTask, error) {
	expires := qqFleetExpiry(now, fleet.StartAt)
	if !expires.After(now) {
		return nil, ErrQQOfficialExpired
	}
	corp, err := s.repo.FleetCharacterCorporation(ctx, tx, fleet.FCCharacterID)
	if errors.Is(err, gorm.ErrRecordNotFound) || err == nil && corp <= 0 {
		return nil, ErrQQOfficialNoTargets
	}
	if err != nil {
		return nil, err
	}
	return s.SubmitNotification(ctx, tx, QQOfficialNotification{
		Source: QQSourceFleet, StableKey: key, Reference: fleet.ID, Text: qqFleetText(fleet), CorporationID: corp, ExpiresAt: expires,
	})
}

func (s *FleetService) manageQQFleet(fleet *model.Fleet, err error, userID uint, roles []string) error {
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return ErrQQOfficialNotFound
	}
	if err != nil {
		return QQOfficialPublicError(err)
	}
	if !s.canManageFleet(fleet, userID, roles) {
		return ErrFleetQQForbidden
	}
	return nil
}

func (s *FleetService) SendFleetQQPing(ctx context.Context, fleetID, idempotencyKey string, userID uint, roles []string) ([]model.QQOfficialTask, error) {
	fleetID, err := qqOperationKey(fleetID)
	if err != nil {
		return nil, err
	}
	key, err := qqOperationKey(idempotencyKey)
	if err != nil {
		return nil, err
	}
	key = "manual:" + key
	var qq *QQOfficialService
	var tasks []model.QQOfficialTask
	err = s.repo.Transaction(ctx, func(tx *gorm.DB) error {
		var err error
		qq, err = DefaultQQOfficialRuntime().Active(ctx, tx)
		if err != nil {
			return err
		}
		if qq == nil {
			return ErrQQOfficialUnavailable
		}
		return qq.locked(ctx, tx, func(tx *gorm.DB, _ *model.QQOfficialApp, now time.Time) error {
			fleet, err := qq.repo.NotificationFleet(ctx, tx, fleetID)
			if err = s.manageQQFleet(&fleet, err, userID, roles); err != nil {
				return err
			}
			// Replays are tied to the requested fleet and return the original
			// receipts even when its schedule or target groups change later.
			intent, err := qq.repo.Intent(ctx, tx, qq.appID, qqHash(QQSourceFleet+":"+key))
			if err == nil {
				tasks, err = qq.repo.IntentTasks(ctx, tx, qq.appID, intent.ID)
				if err != nil {
					return err
				}
				if len(tasks) == 0 || tasks[0].Source != QQSourceFleet || tasks[0].Reference != fleet.ID {
					return ErrQQOfficialConflict
				}
				return nil
			}
			if !errors.Is(err, gorm.ErrRecordNotFound) {
				return err
			}
			tasks, err = qq.queueFleetInTransaction(ctx, tx, fleet, key, now)
			return err
		})
	})
	if err != nil {
		if qq == nil {
			// A missing or unreadable runtime must not bypass fleet authorization.
			fleet, lookupErr := s.repo.GetByID(fleetID)
			if authErr := s.manageQQFleet(fleet, lookupErr, userID, roles); authErr != nil {
				return nil, authErr
			}
		}
		if errors.Is(err, ErrFleetQQForbidden) {
			return nil, err
		}
		return nil, QQOfficialPublicError(err)
	}
	qq.signal()
	return tasks, nil
}

func (s *FleetService) ListFleetQQPingResults(ctx context.Context, fleetID string, userID uint, roles []string, offset, limit int) ([]model.QQOfficialTask, int64, error) {
	fleetID, err := qqOperationKey(fleetID)
	if err != nil {
		return nil, 0, err
	}
	fleet, err := s.repo.GetByID(fleetID)
	if err = s.manageQQFleet(fleet, err, userID, roles); err != nil {
		return nil, 0, err
	}
	qq, err := DefaultQQOfficialRuntime().Active(ctx, nil)
	if err != nil {
		return nil, 0, QQOfficialPublicError(err)
	}
	if qq == nil {
		return []model.QQOfficialTask{}, 0, nil
	}
	rows, total, err := qq.FleetTasks(ctx, fleetID, offset, limit)
	return rows, total, QQOfficialPublicError(err)
}

package service

import (
	"amiya-eden/internal/model"
	"context"
	"errors"
	"fmt"
	"time"

	"gorm.io/gorm"
)

const qqStructureScanBatch = 25

// ScanStructureAlerts observes persisted snapshots only. Each batch takes the
// QQ app lock before reading or changing alert state and committing queue tasks.
func (s *QQOfficialService) ScanStructureAlerts(ctx context.Context) error {
	if s == nil {
		return nil
	}
	corps, err := s.repo.StructureTargetCorporations(ctx, s.appID)
	if err != nil {
		return QQOfficialPublicError(err)
	}
	for _, corp := range corps {
		var after int64
		for {
			next, queued, err := s.scanStructureAlertBatch(ctx, corp, after)
			if err != nil {
				return QQOfficialPublicError(err)
			}
			if queued {
				s.signal()
			}
			if next == 0 {
				break
			}
			after = next
		}
	}
	return nil
}

func (s *QQOfficialService) scanStructureAlertBatch(ctx context.Context, corp, after int64) (next int64, queued bool, err error) {
	err = s.locked(ctx, nil, func(tx *gorm.DB, app *model.QQOfficialApp, now time.Time) error {
		if !app.Enabled {
			return nil
		}
		settings, authorized, err := s.qqStructureSettings(ctx, tx, corp)
		if err != nil || !authorized {
			return err
		}
		groups, err := s.repo.TargetGroups(ctx, tx, s.appID, QQSourceStructure, corp)
		if err != nil || len(groups) == 0 {
			return err
		}
		fence := qqStructureTargetFence(app.Generation, groups)
		snapshots, err := s.repo.StructureAlertSnapshots(ctx, tx, corp, after, now.Add(-qqStructureFreshness).Unix(), now.Unix(), qqStructureScanBatch)
		if err != nil || len(snapshots) == 0 {
			return err
		}
		ids := make([]int64, 0, len(snapshots))
		for _, st := range snapshots {
			ids = append(ids, st.StructureID)
		}
		stored, err := s.repo.StructureAlerts(ctx, tx, s.appID, corp, ids)
		if err != nil {
			return err
		}
		type alertKey struct {
			id   int64
			kind string
		}
		current := make(map[alertKey]model.QQOfficialStructureAlert, len(stored))
		for _, row := range stored {
			current[alertKey{row.StructureID, row.Kind}] = row
		}
		for _, st := range snapshots {
			for _, kind := range []string{qqStructureFuel, qqStructureTimer} {
				condition := qqStructureCondition(st, settings, kind, now)
				if !condition.known {
					continue
				}
				alert, found := current[alertKey{st.StructureID, kind}]
				if !found {
					alert = model.QQOfficialStructureAlert{AppID: s.appID, CorporationID: corp, StructureID: st.StructureID, Kind: kind}
				}
				submitted, err := s.observeStructureAlert(ctx, tx, &alert, condition, fence, now)
				if err != nil {
					return err
				}
				queued = queued || submitted
			}
		}
		if len(snapshots) == qqStructureScanBatch {
			next = snapshots[len(snapshots)-1].StructureID
		}
		return nil
	})
	return next, queued, err
}

// TargetGroups supplies rows in ID order. A pending version has not yet
// captured durable task generations, so fence it against opt-out and rebinding.
func qqStructureTargetFence(appGeneration int64, groups []model.QQOfficialGroup) string {
	data := fmt.Appendf(nil, "%d", appGeneration)
	for _, group := range groups {
		data = fmt.Appendf(data, ":%q:%d:%d", group.ID, group.Generation, group.NotifyGeneration)
	}
	return qqHash(string(data))
}

func (s *QQOfficialService) observeStructureAlert(ctx context.Context, tx *gorm.DB, row *model.QQOfficialStructureAlert, condition qqAlertCondition, fence string, now time.Time) (bool, error) {
	if !condition.active {
		if !row.Active {
			return false, nil
		}
		row.Active, row.Pending = false, false
		return false, s.repo.SaveStructureAlert(ctx, tx, row)
	}

	changed := !row.Active || !qqSameTimerDeadline(row.TimerDeadline, condition.deadline)
	if changed {
		row.Version++
		row.Active, row.Pending = true, true
		row.TargetFence = fence
		row.TimerDeadline = nil
		row.ExpiresAt = now.Add(qqStructureAlertLifetime)
		if !condition.deadline.IsZero() {
			row.TimerDeadline = &condition.deadline
			if row.ExpiresAt.After(condition.deadline) {
				row.ExpiresAt = condition.deadline
			}
		}
	}
	if !row.Pending {
		return false, nil
	}
	if row.TargetFence != fence || !row.ExpiresAt.After(now) {
		row.Pending = false
		return false, s.repo.SaveStructureAlert(ctx, tx, row)
	}

	reference := qqStructureReference(row.CorporationID, row.StructureID, row.Kind, row.Version)
	_, err := s.SubmitNotification(ctx, tx, QQOfficialNotification{
		Source: QQSourceStructure, StableKey: reference, Reference: reference, CorporationID: row.CorporationID,
		Text: qqStructureText(row.Kind, condition.deadline), ExpiresAt: row.ExpiresAt,
	})
	switch {
	case err == nil:
		row.Pending = false
		return true, s.repo.SaveStructureAlert(ctx, tx, row)
	case errors.Is(err, ErrQQOfficialNoTargets), errors.Is(err, ErrQQOfficialDisabled), errors.Is(err, ErrQQOfficialExpired):
		row.Pending = false
		return false, s.repo.SaveStructureAlert(ctx, tx, row)
	case errors.Is(err, ErrQQOfficialCapacity), errors.Is(err, ErrQQOfficialUnavailable):
		// Keep the same version and absolute expiry, never create a fresh task
		// just because the provider outcome or old snapshot is unknown.
		if changed {
			return false, s.repo.SaveStructureAlert(ctx, tx, row)
		}
		return false, nil
	default:
		return false, err
	}
}

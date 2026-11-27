package service

import (
	"amiya-eden/internal/model"
	"amiya-eden/internal/utils"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"strconv"
	"strings"
	"time"

	"gorm.io/gorm"
)

const (
	qqFleetLifetime          = 10 * time.Minute
	qqStructureFreshness     = 24 * time.Hour
	qqStructureAlertLifetime = 3 * time.Hour
	qqStructureFuel          = "fuel"
	qqStructureTimer         = "timer"
)

// Business messages contain only fixed labels and typed schedule data. Never
// copy a title, description, name, mention, link, or user-authored markup.
func qqFleetText(fleet model.Fleet) string {
	importance := "舰队行动"
	switch fleet.Importance {
	case model.FleetImportanceStratOp:
		importance = "战略行动"
	case model.FleetImportanceCTA:
		importance = "全面集结"
	}
	return "AmiyaEden 舰队提醒：" + importance + "，开始时间 " + fleet.StartAt.UTC().Format(time.RFC3339) + "。请在舰队列表查看详情。"
}

func qqFleetExpiry(now, start time.Time) time.Time {
	if deadline := now.Add(qqFleetLifetime); deadline.Before(start) {
		return deadline
	}
	return start
}

func qqStructureText(kind string, deadline time.Time) string {
	if kind == qqStructureTimer {
		return "AmiyaEden 建筑提醒：有建筑的计时器将在 " + deadline.UTC().Format(time.RFC3339) + " 结束。请在建筑管理查看详情。"
	}
	return "AmiyaEden 建筑提醒：有建筑的燃料已达到提醒阈值。请在建筑管理查看详情。"
}

// businessStillValid runs with the app lock held by the dispatcher. Missing,
// stale or superseded records are permanent skips; infrastructure errors are
// propagated so the dispatcher does not mistake them for business recovery.
func (s *QQOfficialService) businessStillValid(ctx context.Context, tx *gorm.DB, task model.QQOfficialTask, group model.QQOfficialGroup, now time.Time) (bool, error) {
	if !task.ExpiresAt.After(now) {
		return false, nil
	}
	switch task.Source {
	case QQSourceFleet:
		if group.FleetCorporation <= 0 {
			return false, nil
		}
		fleet, err := s.repo.NotificationFleet(ctx, tx, task.Reference)
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return false, nil
		}
		if err != nil {
			return false, err
		}
		if !fleet.StartAt.After(now) || task.Text != qqFleetText(fleet) {
			return false, nil
		}
		corp, err := s.repo.FleetCharacterCorporation(ctx, tx, fleet.FCCharacterID)
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return false, nil
		}
		return corp == group.FleetCorporation && err == nil, err
	case QQSourceStructure:
		corp, structure, kind, version, ok := parseQQStructureReference(task.Reference)
		if !ok || corp != group.StructureCorporation {
			return false, nil
		}
		state, err := s.repo.StructureAlert(ctx, tx, s.appID, corp, structure, kind)
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return false, nil
		}
		if err != nil {
			return false, err
		}
		if !state.Active || state.Pending || state.Version != version || !state.ExpiresAt.After(now) {
			return false, nil
		}
		settings, authorized, err := s.qqStructureSettings(ctx, tx, corp)
		if err != nil || !authorized {
			return false, err
		}
		snapshot, err := s.repo.StructureAlertSnapshot(ctx, tx, corp, structure)
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return false, nil
		}
		if err != nil {
			return false, err
		}
		condition := qqStructureCondition(snapshot, settings, kind, now)
		return condition.active && qqSameTimerDeadline(state.TimerDeadline, condition.deadline) && task.Text == qqStructureText(kind, condition.deadline), nil
	default:
		return false, nil
	}
}

func qqStructureReference(corp, id int64, kind string, version int64) string {
	return fmt.Sprintf("%d:%d:%s:%d", corp, id, kind, version)
}

func parseQQStructureReference(in string) (corp, id int64, kind string, version int64, ok bool) {
	parts := strings.Split(in, ":")
	if len(parts) != 4 || (parts[2] != qqStructureFuel && parts[2] != qqStructureTimer) {
		return 0, 0, "", 0, false
	}
	corp, _ = strconv.ParseInt(parts[0], 10, 64)
	id, _ = strconv.ParseInt(parts[1], 10, 64)
	version, _ = strconv.ParseInt(parts[3], 10, 64)
	return corp, id, parts[2], version, corp > 0 && id > 0 && version > 0 && in == qqStructureReference(corp, id, parts[2], version)
}

func (s *QQOfficialService) qqStructureSettings(ctx context.Context, tx *gorm.DB, corp int64) (settings corporationStructureNoticeThresholdSettings, authorized bool, err error) {
	rows, err := s.repo.StructureAlertSettings(ctx, tx)
	if err != nil {
		return settings, false, err
	}
	var bindings map[string]int64
	if raw := rows[model.SysConfigDashboardCorpStructuresAuth]; raw == "" || json.Unmarshal([]byte(raw), &bindings) != nil {
		return settings, false, nil
	}
	if character := bindings[strconv.FormatInt(corp, 10)]; character <= 0 {
		return settings, false, nil
	}
	var allowed []int64
	if raw := rows[model.SysConfigAllowCorporations]; raw != "" && json.Unmarshal([]byte(raw), &allowed) != nil {
		return settings, false, nil
	}
	if !slices.Contains(utils.NormalizeAllowCorporations(allowed), corp) {
		return settings, false, nil
	}
	threshold := func(key string, fallback int) int {
		if value, err := strconv.Atoi(rows[key]); err == nil {
			return normalizeNoticeThresholdDays(value)
		}
		return fallback
	}
	settings.FuelNoticeThresholdDays = threshold(model.SysConfigDashboardCorpStructuresFuelNoticeThresholdDays, model.SysConfigDefaultDashboardCorpStructuresFuelNoticeThresholdDays)
	settings.TimerNoticeThresholdDays = threshold(model.SysConfigDashboardCorpStructuresTimerNoticeThresholdDays, model.SysConfigDefaultDashboardCorpStructuresTimerNoticeThresholdDays)
	authorized, err = s.repo.StructureAlertDirector(ctx, tx, corp, bindings[strconv.FormatInt(corp, 10)])
	return settings, authorized, err
}

type qqAlertCondition struct {
	known, active bool
	deadline      time.Time
}

func qqStructureCondition(st model.CorpStructureInfo, settings corporationStructureNoticeThresholdSettings, kind string, now time.Time) qqAlertCondition {
	// The ESI sync writes Unix seconds, not milliseconds, when it observes
	// the snapshot. Neither missing nor implausibly future values are fresh.
	if st.UpdateAt <= 0 || st.UpdateAt < now.Add(-qqStructureFreshness).Unix() || st.UpdateAt > now.Unix() {
		return qqAlertCondition{}
	}
	switch kind {
	case qqStructureFuel:
		if settings.FuelNoticeThresholdDays == 0 {
			return qqAlertCondition{known: true}
		}
		remaining, _ := calculateFuelRemaining(st.FuelExpires, now)
		if remaining == nil {
			return qqAlertCondition{}
		}
		return qqAlertCondition{known: true, active: (*remaining+hoursPerDay-1)/hoursPerDay <= settings.FuelNoticeThresholdDays}
	case qqStructureTimer:
		if settings.TimerNoticeThresholdDays == 0 {
			return qqAlertCondition{known: true}
		}
		deadline, ok := parseFlexibleTime(st.StateTimerEnd)
		if !ok {
			return qqAlertCondition{}
		}
		deadline = deadline.UTC().Truncate(time.Microsecond) // PostgreSQL timestamp precision
		remain := deadline.Sub(now)
		return qqAlertCondition{known: true, active: remain > 0 && (remain-1)/(hoursPerDay*time.Hour)+1 <= time.Duration(settings.TimerNoticeThresholdDays), deadline: deadline}
	}
	return qqAlertCondition{}
}

func qqSameTimerDeadline(previous *time.Time, current time.Time) bool {
	if previous == nil {
		return current.IsZero()
	}
	return previous.Equal(current)
}

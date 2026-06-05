package esi

import (
	"amiya-eden/global"
	"amiya-eden/internal/model"
	"encoding/json"
	"strings"
	"sync"
	"time"
)

var (
	intervalOverrides   map[string]RefreshInterval
	intervalOverridesMu sync.RWMutex
)

// SetIntervalOverrides replaces the in-memory override cache.
// overrides maps task_name -> RefreshInterval.
func SetIntervalOverrides(overrides map[string]RefreshInterval) {
	intervalOverridesMu.Lock()
	defer intervalOverridesMu.Unlock()
	intervalOverrides = overrides
}

// GetIntervalOverride returns the override for a task if one exists.
func GetIntervalOverride(taskName string) (RefreshInterval, bool) {
	intervalOverridesMu.RLock()
	defer intervalOverridesMu.RUnlock()
	override, ok := intervalOverrides[taskName]
	return override, ok
}

// ResolveInterval returns the effective RefreshInterval for a task.
// Override takes precedence; falls back to the task's default Interval().
// If the task is not registered, returns a sensible default.
func ResolveInterval(taskName string) RefreshInterval {
	intervalOverridesMu.RLock()
	if override, ok := intervalOverrides[taskName]; ok {
		intervalOverridesMu.RUnlock()
		return override
	}
	intervalOverridesMu.RUnlock()

	if task, ok := GetTask(taskName); ok {
		return task.Interval()
	}

	return RefreshInterval{Active: 6 * time.Hour, Inactive: 24 * time.Hour}
}

// RefreshIntervalMinutes is the JSON-serializable form used in system_config.
type RefreshIntervalMinutes struct {
	ActiveMinutes   int `json:"active_minutes"`
	InactiveMinutes int `json:"inactive_minutes"`
}

// ToRefreshInterval converts minutes-based config to RefreshInterval.
func (m RefreshIntervalMinutes) ToRefreshInterval() RefreshInterval {
	return RefreshInterval{
		Active:   time.Duration(m.ActiveMinutes) * time.Minute,
		Inactive: time.Duration(m.InactiveMinutes) * time.Minute,
	}
}

// FromRefreshInterval converts a RefreshInterval to minutes-based config.
func FromRefreshInterval(ri RefreshInterval) RefreshIntervalMinutes {
	return RefreshIntervalMinutes{
		ActiveMinutes:   int(ri.Active.Minutes()),
		InactiveMinutes: int(ri.Inactive.Minutes()),
	}
}

// LoadIntervalOverrides 从 system_config 加载 ESI 间隔覆盖并写入内存缓存。
func LoadIntervalOverrides() error {
	if global.DB == nil {
		SetIntervalOverrides(nil)
		return nil
	}

	var raw string
	if err := global.DB.Model(&model.SystemConfig{}).
		Where("key = ?", model.SysConfigESITaskIntervals).
		Pluck("value", &raw).Error; err != nil {
		return err
	}

	overrides := make(map[string]RefreshIntervalMinutes)
	if strings.TrimSpace(raw) != "" {
		if err := json.Unmarshal([]byte(raw), &overrides); err != nil {
			// 解析失败时使用空覆盖，不阻塞启动
			overrides = make(map[string]RefreshIntervalMinutes)
		}
	}

	refreshOverrides := make(map[string]RefreshInterval, len(overrides))
	for name, ov := range overrides {
		refreshOverrides[name] = ov.ToRefreshInterval()
	}
	SetIntervalOverrides(refreshOverrides)
	return nil
}

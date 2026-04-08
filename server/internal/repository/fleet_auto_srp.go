package repository

import (
	"amiya-eden/global"
	"amiya-eden/internal/model"
	"time"
)

func (r *FleetRepository) SetAutoSrpScheduledFor(fleetID string, scheduledFor *time.Time) error {
	return global.DB.Model(&model.Fleet{}).
		Where("id = ? AND deleted_at IS NULL", fleetID).
		Update("auto_srp_scheduled_for", scheduledFor).Error
}

func (r *FleetRepository) ClearAutoSrpScheduledForIfMatch(fleetID string, scheduledFor time.Time) error {
	return global.DB.Model(&model.Fleet{}).
		Where("id = ? AND auto_srp_scheduled_for = ? AND deleted_at IS NULL", fleetID, scheduledFor).
		Update("auto_srp_scheduled_for", nil).Error
}

func (r *FleetRepository) ClaimAutoSrpScheduledForIfMatch(fleetID string, scheduledFor time.Time) (bool, error) {
	result := global.DB.Model(&model.Fleet{}).
		Where("id = ? AND auto_srp_scheduled_for = ? AND deleted_at IS NULL", fleetID, scheduledFor).
		Update("auto_srp_scheduled_for", nil)
	if result.Error != nil {
		return false, result.Error
	}
	return result.RowsAffected > 0, nil
}

func (r *FleetRepository) ListWithAutoSrpScheduled() ([]model.Fleet, error) {
	var fleets []model.Fleet
	err := global.DB.Where(
		"auto_srp_mode != ? AND auto_srp_scheduled_for IS NOT NULL AND deleted_at IS NULL",
		model.FleetAutoSrpDisabled,
	).Find(&fleets).Error
	return fleets, err
}

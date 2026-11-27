package repository

import (
	"amiya-eden/internal/model"
	"context"

	"gorm.io/gorm"
)

func (r *QQOfficialRepository) NotificationFleet(ctx context.Context, tx *gorm.DB, id string) (model.Fleet, error) {
	var fleet model.Fleet
	err := r.dbFor(ctx, tx).Where("id = ? AND deleted_at IS NULL", id).Take(&fleet).Error
	return fleet, err
}

func (r *QQOfficialRepository) FleetCharacterCorporation(ctx context.Context, tx *gorm.DB, characterID int64) (int64, error) {
	var char model.EveCharacter
	err := r.dbFor(ctx, tx).Select("corporation_id").Where("character_id = ?", characterID).Take(&char).Error
	return char.CorporationID, err
}

func (r *QQOfficialRepository) StructureAlertSettings(ctx context.Context, tx *gorm.DB) (map[string]string, error) {
	var rows []model.SystemConfig
	err := r.dbFor(ctx, tx).Select("key", "value").Where("key IN ?", []string{
		model.SysConfigAllowCorporations,
		model.SysConfigDashboardCorpStructuresAuth,
		model.SysConfigDashboardCorpStructuresFuelNoticeThresholdDays,
		model.SysConfigDashboardCorpStructuresTimerNoticeThresholdDays,
	}).Find(&rows).Error
	out := make(map[string]string, len(rows))
	for _, row := range rows {
		out[row.Key] = row.Value
	}
	return out, err
}

// StructureAlertDirector checks the same local Director and platform role
// requirements as the corporation structure authorization settings.
func (r *QQOfficialRepository) StructureAlertDirector(ctx context.Context, tx *gorm.DB, corporation, character int64) (bool, error) {
	var count int64
	err := r.dbFor(ctx, tx).Table("eve_character AS ec").
		Joins("JOIN user_role AS ur ON ur.user_id = ec.user_id").
		Joins("JOIN eve_character_corp_role AS ecr ON ecr.character_id = ec.character_id AND ecr.corporation_id = ec.corporation_id").
		Where("ur.role_code IN ?", []string{model.RoleAdmin, model.RoleSuperAdmin}).
		Where("ec.corporation_id = ? AND ec.character_id = ? AND LOWER(ecr.corp_role) = ?", corporation, character, "director").
		Limit(1).Count(&count).Error
	return count > 0, err
}

func (r *QQOfficialRepository) StructureTargetCorporations(ctx context.Context, appID string) ([]int64, error) {
	ids := []int64{}
	err := r.dbFor(ctx, nil).Model(&model.QQOfficialGroup{}).
		Where("app_id = ? AND present AND authorized AND notification_enabled AND test_verified AND push_state = ? AND structure_corporation > 0", appID, "allowed").
		Distinct().Order("structure_corporation").Pluck("structure_corporation", &ids).Error
	return ids, err
}

func (r *QQOfficialRepository) StructureAlertSnapshot(ctx context.Context, tx *gorm.DB, corpID, structureID int64) (model.CorpStructureInfo, error) {
	var row model.CorpStructureInfo
	err := r.dbFor(ctx, tx).Select("corporation_id", "structure_id", "fuel_expires", "state_timer_end", "update_at").
		Where("corporation_id = ? AND structure_id = ?", corpID, structureID).Take(&row).Error
	return row, err
}

func (r *QQOfficialRepository) StructureAlertSnapshots(ctx context.Context, tx *gorm.DB, corpID, afterID, earliest, latest int64, limit int) ([]model.CorpStructureInfo, error) {
	var rows []model.CorpStructureInfo
	err := r.dbFor(ctx, tx).Select("corporation_id", "structure_id", "fuel_expires", "state_timer_end", "update_at").
		Where("corporation_id = ? AND structure_id > ? AND update_at BETWEEN ? AND ?", corpID, afterID, earliest, latest).
		Order("structure_id").Limit(limit).Find(&rows).Error
	return rows, err
}

func (r *QQOfficialRepository) StructureAlert(ctx context.Context, tx *gorm.DB, appID string, corpID, structureID int64, kind string) (model.QQOfficialStructureAlert, error) {
	var row model.QQOfficialStructureAlert
	err := r.dbFor(ctx, tx).Where("app_id = ? AND corporation_id = ? AND structure_id = ? AND kind = ?", appID, corpID, structureID, kind).Take(&row).Error
	return row, err
}

func (r *QQOfficialRepository) StructureAlerts(ctx context.Context, tx *gorm.DB, appID string, corpID int64, ids []int64) ([]model.QQOfficialStructureAlert, error) {
	var rows []model.QQOfficialStructureAlert
	err := r.dbFor(ctx, tx).Where("app_id = ? AND corporation_id = ? AND structure_id IN ?", appID, corpID, ids).Find(&rows).Error
	return rows, err
}

func (r *QQOfficialRepository) SaveStructureAlert(ctx context.Context, tx *gorm.DB, row *model.QQOfficialStructureAlert) error {
	return r.dbFor(ctx, tx).Save(row).Error
}

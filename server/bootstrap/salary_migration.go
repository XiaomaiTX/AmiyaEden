package bootstrap

import (
	"amiya-eden/internal/model"
	"errors"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"strconv"
)

// Split the former shared admin award once. Never replace an independently
// configured category or change FC settings/history during startup.
func migrateSalarySettings(db *gorm.DB) error {
	return db.Transaction(func(tx *gorm.DB) error {
		legacy := strconv.Itoa(model.SysConfigDefaultPAPAdminAward)
		var old model.SystemConfig
		if err := tx.Where("key = ?", model.SysConfigPAPAdminAward).First(&old).Error; err == nil {
			legacy = old.Value
		} else if !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
		rows := []model.SystemConfig{
			{Key: model.SysConfigSalaryRecruitment, Value: "10", Desc: "招新工资"},
			{Key: model.SysConfigSalaryWelfare, Value: legacy, Desc: "福利发放工资"},
			{Key: model.SysConfigSalaryShop, Value: legacy, Desc: "商城发货工资"},
			{Key: model.SysConfigSalarySRP, Value: "10", Desc: "补损发放工资"},
		}
		return tx.Clauses(clause.OnConflict{Columns: []clause.Column{{Name: "key"}}, DoNothing: true}).Create(&rows).Error
	})
}

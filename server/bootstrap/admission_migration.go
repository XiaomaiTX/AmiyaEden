package bootstrap

import (
	"amiya-eden/internal/model"

	"gorm.io/gorm"
)

func migrateAdmissions(db *gorm.DB) error {
	if err := db.AutoMigrate(
		&model.AdmissionApplication{}, &model.AdmissionCharacter{},
		&model.AdmissionCharacterClaim{}, &model.AdmissionEvidence{},
		&model.AdmissionProcessingEvent{}, &model.AdmissionSalary{},
	); err != nil {
		return err
	}
	if err := db.AutoMigrate(&model.TicketCategory{}); err != nil {
		return err
	}
	return db.Model(&model.TicketCategory{}).Where("name IN ?", []string{"加团申请", "加团", "小号加团"}).Updates(map[string]interface{}{"enabled": false, "retired_admission": true}).Error
}

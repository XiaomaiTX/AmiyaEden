package repository

import (
	"amiya-eden/global"
	"amiya-eden/internal/model"

	"gorm.io/gorm/clause"
)

type StructureServiceActivityRepository struct{}

func NewStructureServiceActivityRepository() *StructureServiceActivityRepository {
	return &StructureServiceActivityRepository{}
}

func (r *StructureServiceActivityRepository) ListAll() ([]model.StructureServiceActivity, error) {
	var rows []model.StructureServiceActivity
	err := global.DB.Order("activity_name ASC").Find(&rows).Error
	return rows, err
}

func (r *StructureServiceActivityRepository) Upsert(rows []model.StructureServiceActivity) error {
	if len(rows) == 0 {
		return nil
	}
	return global.DB.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "activity_name"}},
		DoUpdates: clause.AssignmentColumns([]string{"type_id", "updated_at"}),
	}).Create(&rows).Error
}

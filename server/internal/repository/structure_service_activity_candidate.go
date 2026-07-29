package repository

import (
	"amiya-eden/global"
	"amiya-eden/internal/model"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type StructureServiceActivityCandidateRepository struct{}

func NewStructureServiceActivityCandidateRepository() *StructureServiceActivityCandidateRepository {
	return &StructureServiceActivityCandidateRepository{}
}

func (r *StructureServiceActivityCandidateRepository) ListAll() ([]model.StructureServiceActivityCandidate, error) {
	var rows []model.StructureServiceActivityCandidate
	err := global.DB.Order("activity_name ASC, type_id ASC").Find(&rows).Error
	return rows, err
}

func (r *StructureServiceActivityCandidateRepository) Count() (int64, error) {
	var count int64
	err := global.DB.Model(&model.StructureServiceActivityCandidate{}).Count(&count).Error
	return count, err
}

func (r *StructureServiceActivityCandidateRepository) Upsert(rows []model.StructureServiceActivityCandidate) error {
	if len(rows) == 0 {
		return nil
	}
	return global.DB.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "activity_name"}, {Name: "type_id"}},
		DoUpdates: clause.AssignmentColumns([]string{"system_managed", "updated_at"}),
	}).Create(&rows).Error
}

func (r *StructureServiceActivityCandidateRepository) ReplaceCustom(activityName string, typeIDs []int) error {
	return global.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("activity_name = ? AND system_managed = ?", activityName, false).
			Delete(&model.StructureServiceActivityCandidate{}).Error; err != nil {
			return err
		}
		rows := make([]model.StructureServiceActivityCandidate, 0, len(typeIDs))
		for _, typeID := range typeIDs {
			rows = append(rows, model.StructureServiceActivityCandidate{ActivityName: activityName, TypeID: typeID})
		}
		if len(rows) == 0 {
			return nil
		}
		return tx.Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "activity_name"}, {Name: "type_id"}},
			DoUpdates: clause.AssignmentColumns([]string{"system_managed", "updated_at"}),
		}).Create(&rows).Error
	})
}

func (r *StructureServiceActivityCandidateRepository) ReplaceSystem(rows []model.StructureServiceActivityCandidate) error {
	if len(rows) == 0 {
		return nil
	}
	names := make([]string, 0, len(rows))
	seen := make(map[string]struct{})
	for _, row := range rows {
		if _, ok := seen[row.ActivityName]; !ok {
			names = append(names, row.ActivityName)
			seen[row.ActivityName] = struct{}{}
		}
	}
	return global.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("activity_name IN ?", names).Delete(&model.StructureServiceActivityCandidate{}).Error; err != nil {
			return err
		}
		return tx.Create(&rows).Error
	})
}

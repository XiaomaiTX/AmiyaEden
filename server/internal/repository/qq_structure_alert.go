package repository

import (
	"amiya-eden/internal/model"
	"context"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type QQStructureAlertRepository struct{ db *gorm.DB }

func NewQQStructureAlertRepository(db *gorm.DB) *QQStructureAlertRepository {
	return &QQStructureAlertRepository{db}
}
func (r *QQStructureAlertRepository) Transaction(ctx context.Context, fn func(*gorm.DB) error) error {
	return r.db.WithContext(ctx).Transaction(fn)
}
func (r *QQStructureAlertRepository) Structures(ctx context.Context) ([]model.CorpStructureInfo, error) {
	rows := []model.CorpStructureInfo{}
	err := r.db.WithContext(ctx).Order("structure_id").Find(&rows).Error
	return rows, err
}
func (r *QQStructureAlertRepository) Threshold(ctx context.Context, key string) (model.SystemConfig, error) {
	var row model.SystemConfig
	err := r.db.WithContext(ctx).First(&row, "key = ?", key).Error
	return row, err
}
func (r *QQStructureAlertRepository) State(tx *gorm.DB, id int64, kind string) (model.QQStructureAlert, error) {
	row := model.QQStructureAlert{StructureID: id, Kind: kind}
	if err := tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&row).Error; err != nil {
		return row, err
	}
	err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&row, "structure_id = ? AND kind = ?", id, kind).Error
	return row, err
}
func (r *QQStructureAlertRepository) Save(tx *gorm.DB, row *model.QQStructureAlert) error {
	return tx.Save(row).Error
}

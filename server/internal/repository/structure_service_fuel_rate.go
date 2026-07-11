package repository

import (
	"amiya-eden/global"
	"amiya-eden/internal/model"

	"gorm.io/gorm/clause"
)

// StructureServiceFuelRateRepository 建筑「服务名 → 每小时燃料块消耗」映射表
type StructureServiceFuelRateRepository struct{}

func NewStructureServiceFuelRateRepository() *StructureServiceFuelRateRepository {
	return &StructureServiceFuelRateRepository{}
}

// ListAll 返回全部映射记录
func (r *StructureServiceFuelRateRepository) ListAll() ([]model.StructureServiceFuelRate, error) {
	var rows []model.StructureServiceFuelRate
	err := global.DB.
		Model(&model.StructureServiceFuelRate{}).
		Order("service_name ASC").
		Find(&rows).Error
	return rows, err
}

// Count 返回记录数（用于启动时判断是否需要补跑同步）
func (r *StructureServiceFuelRateRepository) Count() (int64, error) {
	var count int64
	err := global.DB.Model(&model.StructureServiceFuelRate{}).Count(&count).Error
	return count, err
}

// UpsertBatch 批量写入（冲突时全字段更新）
func (r *StructureServiceFuelRateRepository) UpsertBatch(rows []model.StructureServiceFuelRate) error {
	if len(rows) == 0 {
		return nil
	}
	return global.DB.
		Clauses(clause.OnConflict{UpdateAll: true}).
		Create(&rows).Error
}

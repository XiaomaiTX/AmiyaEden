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

// UpsertBatch 批量写入。
// 注意：本表主键是自增 id，而冲突判定键是 service_name 的 unique index，
// 因此必须显式指定 Columns 为 service_name，否则 OnConflict 会按主键 id 判断冲突
// （Create 时 id 为 0/自增，永远不命中已有主键 → unique index 冲突抛 23505）。
// 更新时刷新 type_id / type_name / fuel_per_hour / updated_at（created_at 保留首次值）。
func (r *StructureServiceFuelRateRepository) UpsertBatch(rows []model.StructureServiceFuelRate) error {
	if len(rows) == 0 {
		return nil
	}
	return global.DB.
		Clauses(clause.OnConflict{
			Columns: []clause.Column{{Name: "service_name"}},
			DoUpdates: clause.AssignmentColumns([]string{
				"type_id",
				"type_name",
				"fuel_per_hour",
				"fuel_category",
				"updated_at",
			}),
		}).
		Create(&rows).Error
}

func (r *StructureServiceFuelRateRepository) FindByTypeID(typeID int) (model.StructureServiceFuelRate, error) {
	var row model.StructureServiceFuelRate
	err := global.DB.Where("type_id = ?", typeID).First(&row).Error
	return row, err
}

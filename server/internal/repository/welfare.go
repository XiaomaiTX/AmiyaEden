package repository

import (
	"amiya-eden/global"
	"amiya-eden/internal/model"
)

// WelfareRepository 福利数据访问层
type WelfareRepository struct{}

func NewWelfareRepository() *WelfareRepository {
	return &WelfareRepository{}
}

// ─────────────────────────────────────────────
//  福利定义
// ─────────────────────────────────────────────

// CreateWelfare 创建福利
func (r *WelfareRepository) CreateWelfare(w *model.Welfare) error {
	return global.DB.Create(w).Error
}

// UpdateWelfare 更新福利
func (r *WelfareRepository) UpdateWelfare(w *model.Welfare) error {
	return global.DB.Save(w).Error
}

// DeleteWelfare 删除福利（软删除）
func (r *WelfareRepository) DeleteWelfare(id uint) error {
	return global.DB.Delete(&model.Welfare{}, id).Error
}

// GetWelfareByID 根据 ID 获取福利
func (r *WelfareRepository) GetWelfareByID(id uint) (*model.Welfare, error) {
	var w model.Welfare
	if err := global.DB.First(&w, id).Error; err != nil {
		return nil, err
	}
	return &w, nil
}

// WelfareFilter 福利查询筛选
type WelfareFilter struct {
	Status *int8
	Name   string
}

// ListWelfares 分页查询福利
func (r *WelfareRepository) ListWelfares(page, pageSize int, filter WelfareFilter) ([]model.Welfare, int64, error) {
	var list []model.Welfare
	var total int64
	offset := (page - 1) * pageSize

	db := global.DB.Model(&model.Welfare{})
	if filter.Status != nil {
		db = db.Where("status = ?", *filter.Status)
	}
	if filter.Name != "" {
		db = db.Where("name ILIKE ?", "%"+filter.Name+"%")
	}

	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	if err := db.Order("id DESC").Offset(offset).Limit(pageSize).Find(&list).Error; err != nil {
		return nil, 0, err
	}
	return list, total, nil
}

// ─────────────────────────────────────────────
//  发放记录
// ─────────────────────────────────────────────

// CountDistributionsByWelfareID 统计福利的发放记录数
func (r *WelfareRepository) CountDistributionsByWelfareID(welfareID uint) (int64, error) {
	var count int64
	err := global.DB.Model(&model.WelfareDistribution{}).Where("welfare_id = ?", welfareID).Count(&count).Error
	return count, err
}

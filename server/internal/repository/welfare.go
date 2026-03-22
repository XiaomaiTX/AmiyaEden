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

	// 填充 SkillPlanIDs
	if err := r.fillSkillPlanIDs(list); err != nil {
		return nil, 0, err
	}
	return list, total, nil
}

// ─────────────────────────────────────────────
//  福利-技能计划关联
// ─────────────────────────────────────────────

// ReplaceWelfareSkillPlans 替换福利的技能计划关联
func (r *WelfareRepository) ReplaceWelfareSkillPlans(welfareID uint, skillPlanIDs []uint) error {
	tx := global.DB.Begin()
	if err := tx.Where("welfare_id = ?", welfareID).Delete(&model.WelfareSkillPlan{}).Error; err != nil {
		tx.Rollback()
		return err
	}
	if len(skillPlanIDs) > 0 {
		rows := make([]model.WelfareSkillPlan, 0, len(skillPlanIDs))
		for _, spID := range skillPlanIDs {
			rows = append(rows, model.WelfareSkillPlan{WelfareID: welfareID, SkillPlanID: spID})
		}
		if err := tx.Create(&rows).Error; err != nil {
			tx.Rollback()
			return err
		}
	}
	return tx.Commit().Error
}

// GetSkillPlanIDsByWelfareID 获取福利关联的技能计划 ID 列表
func (r *WelfareRepository) GetSkillPlanIDsByWelfareID(welfareID uint) ([]uint, error) {
	var rows []model.WelfareSkillPlan
	if err := global.DB.Where("welfare_id = ?", welfareID).Find(&rows).Error; err != nil {
		return nil, err
	}
	ids := make([]uint, len(rows))
	for i, row := range rows {
		ids[i] = row.SkillPlanID
	}
	return ids, nil
}

// fillSkillPlanIDs 批量填充 Welfare.SkillPlanIDs
func (r *WelfareRepository) fillSkillPlanIDs(list []model.Welfare) error {
	if len(list) == 0 {
		return nil
	}
	welfareIDs := make([]uint, len(list))
	for i, w := range list {
		welfareIDs[i] = w.ID
	}
	var rows []model.WelfareSkillPlan
	if err := global.DB.Where("welfare_id IN ?", welfareIDs).Find(&rows).Error; err != nil {
		return err
	}

	m := make(map[uint][]uint)
	for _, row := range rows {
		m[row.WelfareID] = append(m[row.WelfareID], row.SkillPlanID)
	}
	for i := range list {
		list[i].SkillPlanIDs = m[list[i].ID]
		if list[i].SkillPlanIDs == nil {
			list[i].SkillPlanIDs = []uint{}
		}
	}
	return nil
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

package repository

import (
	"amiya-eden/global"
	"amiya-eden/internal/model"
	"errors"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// FuxiAdminRepository 伏羲管理人员数据访问层
type FuxiAdminRepository struct{}

func NewFuxiAdminRepository() *FuxiAdminRepository {
	return &FuxiAdminRepository{}
}

// ─── Config (singleton) ───

func (r *FuxiAdminRepository) GetConfig() (*model.FuxiAdminConfig, error) {
	var cfg model.FuxiAdminConfig
	if err := global.DB.First(&cfg, 1).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &cfg, nil
}

func (r *FuxiAdminRepository) UpsertConfig(cfg *model.FuxiAdminConfig) error {
	cfg.ID = 1
	return global.DB.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "id"}},
		DoUpdates: clause.AssignmentColumns([]string{"base_font_size", "updated_at"}),
	}).Create(cfg).Error
}

// ─── Tiers ───

func (r *FuxiAdminRepository) ListTiers() ([]model.FuxiAdminTier, error) {
	var tiers []model.FuxiAdminTier
	if err := global.DB.Order("sort_order ASC, id ASC").Find(&tiers).Error; err != nil {
		return nil, err
	}
	return tiers, nil
}

func (r *FuxiAdminRepository) GetTierByID(id uint) (*model.FuxiAdminTier, error) {
	var tier model.FuxiAdminTier
	if err := global.DB.First(&tier, id).Error; err != nil {
		return nil, err
	}
	return &tier, nil
}

func (r *FuxiAdminRepository) CreateTier(tier *model.FuxiAdminTier) error {
	return global.DB.Create(tier).Error
}

func (r *FuxiAdminRepository) UpdateTier(tier *model.FuxiAdminTier) error {
	return global.DB.Save(tier).Error
}

func (r *FuxiAdminRepository) DeleteTier(id uint) error {
	return global.DB.Delete(&model.FuxiAdminTier{}, id).Error
}

func (r *FuxiAdminRepository) MaxTierSortOrder() (int, error) {
	var max int
	row := global.DB.Model(&model.FuxiAdminTier{}).Select("COALESCE(MAX(sort_order), -1)").Row()
	if err := row.Scan(&max); err != nil {
		return -1, err
	}
	return max, nil
}

// ─── Admins ───

func (r *FuxiAdminRepository) ListAdminsByTierIDs(tierIDs []uint) ([]model.FuxiAdmin, error) {
	if len(tierIDs) == 0 {
		return nil, nil
	}
	var admins []model.FuxiAdmin
	if err := global.DB.Where("tier_id IN ?", tierIDs).Order("id ASC").Find(&admins).Error; err != nil {
		return nil, err
	}
	return admins, nil
}

func (r *FuxiAdminRepository) GetAdminByID(id uint) (*model.FuxiAdmin, error) {
	var admin model.FuxiAdmin
	if err := global.DB.First(&admin, id).Error; err != nil {
		return nil, err
	}
	return &admin, nil
}

func (r *FuxiAdminRepository) CreateAdmin(admin *model.FuxiAdmin) error {
	return global.DB.Create(admin).Error
}

func (r *FuxiAdminRepository) UpdateAdmin(admin *model.FuxiAdmin) error {
	return global.DB.Save(admin).Error
}

func (r *FuxiAdminRepository) DeleteAdmin(id uint) error {
	return global.DB.Delete(&model.FuxiAdmin{}, id).Error
}

func (r *FuxiAdminRepository) DeleteAdminsByTierID(tierID uint) error {
	return global.DB.Where("tier_id = ?", tierID).Delete(&model.FuxiAdmin{}).Error
}

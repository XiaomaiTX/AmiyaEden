package repository

import (
	"amiya-eden/global"
	"amiya-eden/internal/model"

	"gorm.io/gorm"
)

type ToolBookmarkRepository struct{}

func NewToolBookmarkRepository() *ToolBookmarkRepository {
	return &ToolBookmarkRepository{}
}

func (r *ToolBookmarkRepository) ListVisible() ([]model.ToolBookmark, error) {
	var rows []model.ToolBookmark
	err := global.DB.
		Model(&model.ToolBookmark{}).
		Where("is_enabled = ?", true).
		Order("sort_order ASC, id ASC").
		Find(&rows).Error
	return rows, err
}

func (r *ToolBookmarkRepository) ListAdmin() ([]model.ToolBookmark, error) {
	var rows []model.ToolBookmark
	err := global.DB.
		Model(&model.ToolBookmark{}).
		Order("sort_order ASC, id ASC").
		Find(&rows).Error
	return rows, err
}

func (r *ToolBookmarkRepository) Create(row *model.ToolBookmark) error {
	return global.DB.Create(row).Error
}

func (r *ToolBookmarkRepository) GetByID(id uint) (*model.ToolBookmark, error) {
	var row model.ToolBookmark
	if err := global.DB.First(&row, id).Error; err != nil {
		return nil, err
	}
	return &row, nil
}

func (r *ToolBookmarkRepository) UpdateFields(id uint, fields map[string]interface{}) error {
	result := global.DB.Model(&model.ToolBookmark{}).Where("id = ?", id).Updates(fields)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

func (r *ToolBookmarkRepository) Delete(id uint) error {
	result := global.DB.Delete(&model.ToolBookmark{}, id)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

func (r *ToolBookmarkRepository) MaxSortOrder() (int, error) {
	var max int
	row := global.DB.Model(&model.ToolBookmark{}).Select("COALESCE(MAX(sort_order), 0)").Row()
	if err := row.Scan(&max); err != nil {
		return 0, err
	}
	return max, nil
}

package repository

import (
	"amiya-eden/global"
	"amiya-eden/internal/model"
	"errors"
	"fmt"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// FuxiHallRepository 伏羲大厅数据访问层
type FuxiHallRepository struct {
	db *gorm.DB
}

func NewFuxiHallRepository() *FuxiHallRepository {
	return NewFuxiHallRepositoryWithDB(global.DB)
}

func NewFuxiHallRepositoryWithDB(db *gorm.DB) *FuxiHallRepository {
	return &FuxiHallRepository{db: db}
}

func (r *FuxiHallRepository) Transaction(fn func(repo *FuxiHallRepository) error) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		return fn(NewFuxiHallRepositoryWithDB(tx))
	})
}

func (r *FuxiHallRepository) GetPageByKey(pageKey string) (*model.FuxiHallPage, error) {
	var page model.FuxiHallPage
	if err := r.db.Where("page_key = ?", pageKey).First(&page).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &page, nil
}

func (r *FuxiHallRepository) UpsertPage(page *model.FuxiHallPage) error {
	return r.db.Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "page_key"}},
		DoUpdates: clause.AssignmentColumns([]string{
			"title",
			"subtitle",
			"description_html",
		}),
	}).Create(page).Error
}

func (r *FuxiHallRepository) ListCardsByPage(pageKey string, visibleOnly bool) ([]model.FuxiHallCard, error) {
	var cards []model.FuxiHallCard
	query := r.db.Model(&model.FuxiHallCard{}).Where("page_key = ?", pageKey)
	if visibleOnly {
		query = query.Where("visible = ?", true)
	}
	if err := query.Order("sort_order ASC, id ASC").Find(&cards).Error; err != nil {
		return nil, err
	}
	return cards, nil
}

func (r *FuxiHallRepository) GetCardByID(id uint) (*model.FuxiHallCard, error) {
	var card model.FuxiHallCard
	if err := r.db.First(&card, id).Error; err != nil {
		return nil, err
	}
	return &card, nil
}

func (r *FuxiHallRepository) CreateCard(card *model.FuxiHallCard) error {
	return r.db.Create(card).Error
}

func (r *FuxiHallRepository) UpdateCardFields(id uint, fields map[string]interface{}) error {
	result := r.db.Model(&model.FuxiHallCard{}).Where("id = ?", id).Updates(fields)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

func (r *FuxiHallRepository) DeleteCard(id uint) error {
	result := r.db.Delete(&model.FuxiHallCard{}, id)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

func (r *FuxiHallRepository) MaxCardSortOrder(pageKey string) (int, error) {
	var max int
	row := r.db.Model(&model.FuxiHallCard{}).Where("page_key = ?", pageKey).Select("COALESCE(MAX(sort_order), 0)").Row()
	if err := row.Scan(&max); err != nil {
		return 0, err
	}
	return max, nil
}

func (r *FuxiHallRepository) CountCardsByPageAndIDs(pageKey string, ids []uint) (int64, error) {
	var count int64
	if err := r.db.Model(&model.FuxiHallCard{}).
		Where("page_key = ? AND id IN ?", pageKey, ids).
		Count(&count).Error; err != nil {
		return 0, err
	}
	return count, nil
}

func (r *FuxiHallRepository) ReorderCards(pageKey string, orderedIDs []uint) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		for i, id := range orderedIDs {
			result := tx.Model(&model.FuxiHallCard{}).
				Where("id = ? AND page_key = ?", id, pageKey).
				Update("sort_order", i+1)
			if result.Error != nil {
				return result.Error
			}
			if result.RowsAffected == 0 {
				return fmt.Errorf("卡片 %d 不存在或不属于页面 %s", id, pageKey)
			}
		}
		return nil
	})
}

package repository

import (
	"amiya-eden/global"
	"amiya-eden/internal/model"
	"time"

	"gorm.io/gorm"
)

type EntityTickerCacheRepository struct{}

func NewEntityTickerCacheRepository() *EntityTickerCacheRepository {
	return &EntityTickerCacheRepository{}
}

func (r *EntityTickerCacheRepository) GetFreshTicker(entityType string, entityID int64, now time.Time) (string, bool, error) {
	var row model.EveEntityTickerCache
	err := global.DB.
		Where("entity_type = ? AND entity_id = ? AND expires_at > ?", entityType, entityID, now).
		First(&row).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return "", false, nil
		}
		return "", false, err
	}
	return row.Ticker, true, nil
}

func (r *EntityTickerCacheRepository) ListFreshEntityIDs(entityType string, entityIDs []int64, now time.Time) (map[int64]struct{}, error) {
	result := make(map[int64]struct{}, len(entityIDs))
	if len(entityIDs) == 0 {
		return result, nil
	}
	var ids []int64
	if err := global.DB.Model(&model.EveEntityTickerCache{}).
		Where("entity_type = ? AND entity_id IN ? AND expires_at > ?", entityType, entityIDs, now).
		Pluck("entity_id", &ids).Error; err != nil {
		return nil, err
	}
	for _, id := range ids {
		result[id] = struct{}{}
	}
	return result, nil
}

func (r *EntityTickerCacheRepository) Upsert(entries []model.EveEntityTickerCache) error {
	for i := range entries {
		if err := global.DB.
			Where("entity_type = ? AND entity_id = ?", entries[i].EntityType, entries[i].EntityID).
			Assign(entries[i]).
			FirstOrCreate(&entries[i]).Error; err != nil {
			return err
		}
	}
	return nil
}

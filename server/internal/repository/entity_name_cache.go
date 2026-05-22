package repository

import (
	"amiya-eden/global"
	"amiya-eden/internal/model"
)

type EntityNameCacheRepository struct{}

func NewEntityNameCacheRepository() *EntityNameCacheRepository {
	return &EntityNameCacheRepository{}
}

func (r *EntityNameCacheRepository) ListByEntityIDs(entityIDs []int64) ([]model.EveEntityNameCache, error) {
	result := make([]model.EveEntityNameCache, 0, len(entityIDs))
	if len(entityIDs) == 0 {
		return result, nil
	}

	err := global.DB.
		Where("entity_id IN ?", entityIDs).
		Find(&result).Error
	return result, err
}

func (r *EntityNameCacheRepository) Upsert(entries []model.EveEntityNameCache) error {
	if len(entries) == 0 {
		return nil
	}

	for i := range entries {
		if err := global.DB.
			Where("entity_id = ?", entries[i].EntityID).
			Assign(entries[i]).
			FirstOrCreate(&entries[i]).Error; err != nil {
			return err
		}
	}

	return nil
}

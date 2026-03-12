package repository

import (
	"amiya-eden/global"
	"amiya-eden/internal/model"
	"time"
)

// NpcKillRepository NPC 刷怪数据访问层
type NpcKillRepository struct{}

func NewNpcKillRepository() *NpcKillRepository {
	return &NpcKillRepository{}
}

// GetBountyJournals 获取指定角色的刷怪流水（bounty_prizes + ess_escrow_transfer）
// 支持时间范围过滤
func (r *NpcKillRepository) GetBountyJournals(characterID int64, startDate, endDate *time.Time) ([]model.EVECharacterWalletJournal, error) {
	var journals []model.EVECharacterWalletJournal

	db := global.DB.Model(&model.EVECharacterWalletJournal{}).
		Where("character_id = ?", characterID).
		Where("ref_type IN ?", []string{"bounty_prizes", "ess_escrow_transfer"})

	if startDate != nil {
		db = db.Where("date >= ?", *startDate)
	}
	if endDate != nil {
		db = db.Where("date <= ?", *endDate)
	}

	err := db.Order("date DESC").Find(&journals).Error
	if err != nil {
		return nil, err
	}
	return journals, nil
}

// GetBountyJournalsPaged 分页获取指定角色的刷怪流水
func (r *NpcKillRepository) GetBountyJournalsPaged(characterID int64, startDate, endDate *time.Time, page, pageSize int) ([]model.EVECharacterWalletJournal, int64, error) {
	var journals []model.EVECharacterWalletJournal
	var total int64

	db := global.DB.Model(&model.EVECharacterWalletJournal{}).
		Where("character_id = ?", characterID).
		Where("ref_type IN ?", []string{"bounty_prizes", "ess_escrow_transfer"})

	if startDate != nil {
		db = db.Where("date >= ?", *startDate)
	}
	if endDate != nil {
		db = db.Where("date <= ?", *endDate)
	}

	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	err := db.Order("date DESC").
		Offset((page - 1) * pageSize).
		Limit(pageSize).
		Find(&journals).Error
	if err != nil {
		return nil, 0, err
	}
	return journals, total, nil
}

// GetBountyJournalsByCharacterIDs 获取多个角色的刷怪流水（admin 用）
func (r *NpcKillRepository) GetBountyJournalsByCharacterIDs(characterIDs []int64, startDate, endDate *time.Time) ([]model.EVECharacterWalletJournal, error) {
	if len(characterIDs) == 0 {
		return nil, nil
	}
	var journals []model.EVECharacterWalletJournal

	db := global.DB.Model(&model.EVECharacterWalletJournal{}).
		Where("character_id IN ?", characterIDs).
		Where("ref_type IN ?", []string{"bounty_prizes", "ess_escrow_transfer"})

	if startDate != nil {
		db = db.Where("date >= ?", *startDate)
	}
	if endDate != nil {
		db = db.Where("date <= ?", *endDate)
	}

	err := db.Order("date DESC").Find(&journals).Error
	if err != nil {
		return nil, err
	}
	return journals, nil
}

// GetBountyJournalsByCharacterIDsPaged 分页获取多个角色的刷怪流水（admin 用）
func (r *NpcKillRepository) GetBountyJournalsByCharacterIDsPaged(characterIDs []int64, startDate, endDate *time.Time, page, pageSize int) ([]model.EVECharacterWalletJournal, int64, error) {
	if len(characterIDs) == 0 {
		return nil, 0, nil
	}
	var journals []model.EVECharacterWalletJournal
	var total int64

	db := global.DB.Model(&model.EVECharacterWalletJournal{}).
		Where("character_id IN ?", characterIDs).
		Where("ref_type IN ?", []string{"bounty_prizes", "ess_escrow_transfer"})

	if startDate != nil {
		db = db.Where("date >= ?", *startDate)
	}
	if endDate != nil {
		db = db.Where("date <= ?", *endDate)
	}

	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	err := db.Order("date DESC").
		Offset((page - 1) * pageSize).
		Limit(pageSize).
		Find(&journals).Error
	if err != nil {
		return nil, 0, err
	}
	return journals, total, nil
}

// GetSolarSystemNames 批量查询星系名称
func (r *NpcKillRepository) GetSolarSystemNames(solarSystemIDs []int) (map[int]string, error) {
	if len(solarSystemIDs) == 0 {
		return map[int]string{}, nil
	}
	var systems []model.MapSolarSystem
	err := global.DB.Where(`"solarSystemID" IN ?`, solarSystemIDs).Find(&systems).Error
	if err != nil {
		return nil, err
	}
	result := make(map[int]string, len(systems))
	for _, s := range systems {
		result[s.SolarSystemID] = s.SolarSystemName
	}
	return result, nil
}

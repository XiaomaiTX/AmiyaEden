package repository

import (
	"amiya-eden/global"
	"amiya-eden/internal/model"
	"time"

	"gorm.io/gorm"
)

// NpcKillRepository NPC 刷怪数据访问层
type NpcKillRepository struct{}

func NewNpcKillRepository() *NpcKillRepository {
	return &NpcKillRepository{}
}

// npcIncomeRefTypes 包含所有 NPC 来源收入的 ref_type
var npcIncomeRefTypes = []string{"bounty_prizes", "ess_escrow_transfer", "incursion_payout", "agent_mission_reward"}

// NpcKillJournalQuery NPC 收入流水查询条件
type NpcKillJournalQuery struct {
	CharacterIDs   []int64
	RefTypes       []string
	SolarSystemIDs []int
	StartDate      *time.Time
	EndDate        *time.Time
	MinAmount      *float64
	MaxAmount      *float64
	Page           int
	PageSize       int
}

// GetBountyJournals 获取指定人物的 NPC 收入流水
// 支持时间范围过滤
func (r *NpcKillRepository) GetBountyJournals(characterID int64, startDate, endDate *time.Time) ([]model.EVECharacterWalletJournal, error) {
	return r.ListJournals(NpcKillJournalQuery{
		CharacterIDs: []int64{characterID},
		StartDate:    startDate,
		EndDate:      endDate,
	})
}

// GetBountyJournalsPaged 分页获取指定人物的 NPC 收入流水
func (r *NpcKillRepository) GetBountyJournalsPaged(characterID int64, startDate, endDate *time.Time, page, pageSize int) ([]model.EVECharacterWalletJournal, int64, error) {
	return r.ListJournalsPaged(NpcKillJournalQuery{
		CharacterIDs: []int64{characterID},
		StartDate:    startDate,
		EndDate:      endDate,
		Page:         page,
		PageSize:     pageSize,
	})
}

// GetBountyJournalsByCharacterIDs 获取多个人物的 NPC 收入流水（admin 用）
func (r *NpcKillRepository) GetBountyJournalsByCharacterIDs(characterIDs []int64, startDate, endDate *time.Time) ([]model.EVECharacterWalletJournal, error) {
	return r.ListJournals(NpcKillJournalQuery{
		CharacterIDs: characterIDs,
		StartDate:    startDate,
		EndDate:      endDate,
	})
}

// GetBountyJournalsByCharacterIDsPaged 分页获取多个人物的 NPC 收入流水（admin 用）
func (r *NpcKillRepository) GetBountyJournalsByCharacterIDsPaged(characterIDs []int64, startDate, endDate *time.Time, page, pageSize int) ([]model.EVECharacterWalletJournal, int64, error) {
	return r.ListJournalsPaged(NpcKillJournalQuery{
		CharacterIDs: characterIDs,
		StartDate:    startDate,
		EndDate:      endDate,
		Page:         page,
		PageSize:     pageSize,
	})
}

// ListJournals 查询 NPC 收入流水
func (r *NpcKillRepository) ListJournals(query NpcKillJournalQuery) ([]model.EVECharacterWalletJournal, error) {
	if len(query.CharacterIDs) == 0 {
		return nil, nil
	}
	var journals []model.EVECharacterWalletJournal

	db := r.getJournalQuery(query)

	err := db.Order("date DESC").Find(&journals).Error
	if err != nil {
		return nil, err
	}
	return journals, nil
}

// ListJournalsPaged 分页查询 NPC 收入流水
func (r *NpcKillRepository) ListJournalsPaged(query NpcKillJournalQuery) ([]model.EVECharacterWalletJournal, int64, error) {
	if len(query.CharacterIDs) == 0 {
		return nil, 0, nil
	}
	var journals []model.EVECharacterWalletJournal
	var total int64

	db := r.getJournalQuery(query)

	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	err := db.Order("date DESC").
		Offset((query.Page - 1) * query.PageSize).
		Limit(query.PageSize).
		Find(&journals).Error
	if err != nil {
		return nil, 0, err
	}
	return journals, total, nil
}

func (r *NpcKillRepository) getJournalQuery(query NpcKillJournalQuery) *gorm.DB {
	refTypes := query.RefTypes
	if len(refTypes) == 0 {
		refTypes = npcIncomeRefTypes
	}

	db := global.DB.Model(&model.EVECharacterWalletJournal{}).
		Where("character_id IN ?", query.CharacterIDs).
		Where("ref_type IN ?", refTypes)

	if query.StartDate != nil {
		db = db.Where("date >= ?", *query.StartDate)
	}
	if query.EndDate != nil {
		db = db.Where("date <= ?", *query.EndDate)
	}
	if len(query.SolarSystemIDs) > 0 {
		db = db.Where("ref_type = ? AND context_id IN ?", "bounty_prizes", query.SolarSystemIDs)
	}
	if query.MinAmount != nil {
		db = db.Where("amount >= ?", *query.MinAmount)
	}
	if query.MaxAmount != nil {
		db = db.Where("amount <= ?", *query.MaxAmount)
	}

	return db
}

// GetSolarSystemNames 批量查询星系名称
func (r *NpcKillRepository) GetSolarSystemNames(solarSystemIDs []int) (map[int]string, error) {
	if len(solarSystemIDs) == 0 {
		return map[int]string{}, nil
	}
	type solarSystemNameRow struct {
		SolarSystemID   int    `gorm:"column:solarSystemID"`
		SolarSystemName string `gorm:"column:solarSystemName"`
	}
	var systems []solarSystemNameRow
	err := global.DB.Model(&model.MapSolarSystem{}).
		Select(`"solarSystemID", "solarSystemName"`).
		Where(`"solarSystemID" IN ?`, solarSystemIDs).
		Find(&systems).Error
	if err != nil {
		return nil, err
	}
	result := make(map[int]string, len(systems))
	for _, s := range systems {
		result[s.SolarSystemID] = s.SolarSystemName
	}
	return result, nil
}

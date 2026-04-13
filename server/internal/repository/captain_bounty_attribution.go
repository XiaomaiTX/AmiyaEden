package repository

import (
	"amiya-eden/global"
	"amiya-eden/internal/model"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type CaptainBountyAttributionRepository struct{}

func NewCaptainBountyAttributionRepository() *CaptainBountyAttributionRepository {
	return &CaptainBountyAttributionRepository{}
}

func (r *CaptainBountyAttributionRepository) CreateIgnoreDuplicate(row *model.CaptainBountyAttribution) error {
	return global.DB.Clauses(clause.OnConflict{DoNothing: true}).Create(row).Error
}

func (r *CaptainBountyAttributionRepository) ExistsByWalletJournalID(walletJournalID int64) (bool, error) {
	var count int64
	err := global.DB.Model(&model.CaptainBountyAttribution{}).
		Where("wallet_journal_id = ?", walletJournalID).
		Count(&count).Error
	return count > 0, err
}

func buildCaptainAttributionAggregateQuery(
	db *gorm.DB,
	supportedRefTypes []string,
	includeRecordCount bool,
) *gorm.DB {
	selectExpr := "0 AS bounty_total"
	selectArgs := make([]any, 0, 1)
	if len(supportedRefTypes) > 0 {
		selectExpr = "COALESCE(SUM(CASE WHEN ref_type IN ? THEN amount ELSE 0 END), 0) AS bounty_total"
		selectArgs = append(selectArgs, supportedRefTypes)
	}
	if includeRecordCount {
		selectExpr += ", COUNT(*) AS record_count"
	}
	return db.Select(selectExpr, selectArgs...)
}

func (r *CaptainBountyAttributionRepository) SumByCaptainUserID(
	captainUserID uint,
	supportedRefTypes []string,
) (bountyTotal float64, recordCount int64, err error) {
	type row struct {
		BountyTotal float64
		RecordCount int64
	}
	var result row
	err = buildCaptainAttributionAggregateQuery(
		global.DB.Model(&model.CaptainBountyAttribution{}),
		supportedRefTypes,
		true,
	).
		Where("captain_user_id = ?", captainUserID).
		Scan(&result).Error
	return result.BountyTotal, result.RecordCount, err
}

func (r *CaptainBountyAttributionRepository) SumByCaptainUserIDs(
	captainUserIDs []uint,
	supportedRefTypes []string,
) (map[uint]float64, map[uint]int64, error) {
	bountyTotals := make(map[uint]float64, len(captainUserIDs))
	recordCounts := make(map[uint]int64, len(captainUserIDs))
	if len(captainUserIDs) == 0 {
		return bountyTotals, recordCounts, nil
	}

	type row struct {
		CaptainUserID uint
		BountyTotal   float64
		RecordCount   int64
	}

	query := global.DB.Model(&model.CaptainBountyAttribution{}).
		Where("captain_user_id IN ?", captainUserIDs)
	if len(supportedRefTypes) > 0 {
		query = query.Select(
			"captain_user_id, COALESCE(SUM(CASE WHEN ref_type IN ? THEN amount ELSE 0 END), 0) AS bounty_total, COUNT(*) AS record_count",
			supportedRefTypes,
		)
	} else {
		query = query.Select("captain_user_id, 0 AS bounty_total, COUNT(*) AS record_count")
	}

	var rows []row
	err := query.Group("captain_user_id").Scan(&rows).Error
	if err != nil {
		return nil, nil, err
	}
	for _, row := range rows {
		bountyTotals[row.CaptainUserID] = row.BountyTotal
		recordCounts[row.CaptainUserID] = row.RecordCount
	}
	return bountyTotals, recordCounts, nil
}

func (r *CaptainBountyAttributionRepository) SumByCaptainAndPlayerUserID(
	captainUserID,
	playerUserID uint,
	supportedRefTypes []string,
) (bountyTotal float64, err error) {
	type row struct {
		BountyTotal float64
	}
	var result row
	err = buildCaptainAttributionAggregateQuery(
		global.DB.Model(&model.CaptainBountyAttribution{}),
		supportedRefTypes,
		false,
	).
		Where("captain_user_id = ? AND player_user_id = ?", captainUserID, playerUserID).
		Scan(&result).Error
	return result.BountyTotal, err
}

func (r *CaptainBountyAttributionRepository) ListByCaptainUserIDFiltered(
	captainUserID uint,
	page,
	pageSize int,
	playerUserID *uint,
	refType string,
	startDate,
	endDate *time.Time,
) ([]model.CaptainBountyAttribution, int64, error) {
	var rows []model.CaptainBountyAttribution
	var total int64
	db := global.DB.Model(&model.CaptainBountyAttribution{}).
		Where("captain_user_id = ?", captainUserID)
	if playerUserID != nil && *playerUserID > 0 {
		db = db.Where("player_user_id = ?", *playerUserID)
	}
	if refType != "" {
		db = db.Where("ref_type = ?", refType)
	}
	if startDate != nil {
		db = db.Where("journal_at >= ?", *startDate)
	}
	if endDate != nil {
		db = db.Where("journal_at <= ?", *endDate)
	}
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	err := db.Order("journal_at DESC, id DESC").
		Offset((page - 1) * pageSize).
		Limit(pageSize).
		Find(&rows).Error
	return rows, total, err
}

func (r *CaptainBountyAttributionRepository) SummarizeByCaptainUserIDFiltered(
	captainUserID uint,
	playerUserID *uint,
	refType string,
	startDate,
	endDate *time.Time,
	supportedRefTypes []string,
) (bountyTotal float64, recordCount int64, err error) {
	type row struct {
		BountyTotal float64
		RecordCount int64
	}
	var result row
	db := buildCaptainAttributionAggregateQuery(
		global.DB.Model(&model.CaptainBountyAttribution{}),
		supportedRefTypes,
		true,
	).Where("captain_user_id = ?", captainUserID)
	if playerUserID != nil && *playerUserID > 0 {
		db = db.Where("player_user_id = ?", *playerUserID)
	}
	if refType != "" {
		db = db.Where("ref_type = ?", refType)
	}
	if startDate != nil {
		db = db.Where("journal_at >= ?", *startDate)
	}
	if endDate != nil {
		db = db.Where("journal_at <= ?", *endDate)
	}
	err = db.Scan(&result).Error
	return result.BountyTotal, result.RecordCount, err
}

func buildUnattributedPlayerJournalQuery(
	db *gorm.DB,
	lastWalletJournalID int64,
	lookbackStart time.Time,
	refTypes []string,
	limit int,
) *gorm.DB {
	query := db.Model(&model.EVECharacterWalletJournal{}).
		Joins(`LEFT JOIN captain_bounty_attribution ON captain_bounty_attribution.wallet_journal_id = eve_character_wallet_journal.id`).
		Where("eve_character_wallet_journal.id > ?", lastWalletJournalID).
		Where("eve_character_wallet_journal.date >= ?", lookbackStart).
		Where("captain_bounty_attribution.wallet_journal_id IS NULL")
	if len(refTypes) > 0 {
		query = query.Where("eve_character_wallet_journal.ref_type IN ?", refTypes)
	}
	query = query.Order("eve_character_wallet_journal.id ASC")
	if limit > 0 {
		query = query.Limit(limit)
	}
	return query
}

func (r *CaptainBountyAttributionRepository) ListUnattributedPlayerJournalsFromLookback(
	lastWalletJournalID int64,
	lookbackStart time.Time,
	refTypes []string,
	limit int,
) ([]model.EVECharacterWalletJournal, error) {
	var rows []model.EVECharacterWalletJournal
	err := buildUnattributedPlayerJournalQuery(global.DB, lastWalletJournalID, lookbackStart, refTypes, limit).
		Find(&rows).Error
	return rows, err
}

func buildCaptainCandidateJournalQuery(
	db *gorm.DB,
	characterID int64,
	systemID int64,
	start time.Time,
	end time.Time,
	refTypes []string,
) *gorm.DB {
	query := db.Model(&model.EVECharacterWalletJournal{}).
		Where("character_id = ?", characterID).
		Where("context_id = ?", systemID).
		Where("date >= ? AND date <= ?", start, end)
	if len(refTypes) > 0 {
		query = query.Where("ref_type IN ?", refTypes)
	}
	return query.Order("date ASC, id ASC")
}

func (r *CaptainBountyAttributionRepository) ListCaptainCandidateJournals(
	characterID int64,
	systemID int64,
	start time.Time,
	end time.Time,
	refTypes []string,
) ([]model.EVECharacterWalletJournal, error) {
	var rows []model.EVECharacterWalletJournal
	err := buildCaptainCandidateJournalQuery(global.DB, characterID, systemID, start, end, refTypes).
		Find(&rows).Error
	return rows, err
}

func (r *CaptainBountyAttributionRepository) GetSyncState(syncKey string) (*model.CaptainBountySyncState, error) {
	var state model.CaptainBountySyncState
	err := global.DB.Where("sync_key = ?", syncKey).First(&state).Error
	if err != nil {
		return nil, err
	}
	return &state, nil
}

func (r *CaptainBountyAttributionRepository) SaveSyncState(state *model.CaptainBountySyncState) error {
	return global.DB.Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "sync_key"}},
		DoUpdates: clause.Assignments(map[string]any{
			"last_wallet_journal_id": gorm.Expr("EXCLUDED.last_wallet_journal_id"),
			"last_journal_at":        gorm.Expr("EXCLUDED.last_journal_at"),
			"updated_at":             gorm.Expr("EXCLUDED.updated_at"),
			"deleted_at":             nil,
		}),
	}).Create(state).Error
}

func (r *CaptainBountyAttributionRepository) ListUnprocessed(limit int) ([]model.CaptainBountyAttribution, error) {
	var rows []model.CaptainBountyAttribution
	query := global.DB.Model(&model.CaptainBountyAttribution{}).
		Where("processed_at IS NULL").
		Order("captain_user_id ASC, journal_at ASC, id ASC")
	if limit > 0 {
		query = query.Limit(limit)
	}
	err := query.Find(&rows).Error
	return rows, err
}

func (r *CaptainBountyAttributionRepository) MarkProcessedTx(
	tx *gorm.DB,
	ids []uint,
	processedAt time.Time,
) (int64, error) {
	if len(ids) == 0 {
		return 0, nil
	}
	result := tx.Model(&model.CaptainBountyAttribution{}).
		Where("id IN ? AND processed_at IS NULL", ids).
		Update("processed_at", processedAt)
	return result.RowsAffected, result.Error
}

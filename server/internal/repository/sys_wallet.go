package repository

import (
	"amiya-eden/global"
	"amiya-eden/internal/model"
	"errors"
	"strings"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// SysWalletRepository 伏羲币数据访问层
type SysWalletRepository struct{}

func NewSysWalletRepository() *SysWalletRepository {
	return &SysWalletRepository{}
}

// ─────────────────────────────────────────────
//  钱包 CRUD
// ─────────────────────────────────────────────

func buildGetOrCreateWalletForUpdateQuery(db *gorm.DB, userID uint) *gorm.DB {
	return db.
		Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("user_id = ?", userID)
}

// GetOrCreateWalletTx 在事务内获取或创建用户钱包（使用 FOR UPDATE 行锁防止并发竞态）
func (r *SysWalletRepository) GetOrCreateWalletTx(tx *gorm.DB, userID uint) (*model.SystemWallet, error) {
	var wallet model.SystemWallet
	err := buildGetOrCreateWalletForUpdateQuery(tx, userID).First(&wallet).Error
	if err == nil {
		return &wallet, nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}

	if err := tx.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "user_id"}},
		DoNothing: true,
	}).Create(&model.SystemWallet{UserID: userID, Balance: 0}).Error; err != nil {
		return nil, err
	}

	if err := buildGetOrCreateWalletForUpdateQuery(tx, userID).First(&wallet).Error; err != nil {
		return nil, err
	}

	return &wallet, nil
}

// GetOrCreateWallet 获取或创建用户钱包
func (r *SysWalletRepository) GetOrCreateWallet(userID uint) (*model.SystemWallet, error) {
	var wallet model.SystemWallet
	err := global.DB.Where("user_id = ?", userID).First(&wallet).Error
	if err == nil {
		return &wallet, nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}

	if err := global.DB.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "user_id"}},
		DoNothing: true,
	}).Create(&model.SystemWallet{UserID: userID, Balance: 0}).Error; err != nil {
		return nil, err
	}

	if err := global.DB.Where("user_id = ?", userID).First(&wallet).Error; err != nil {
		return nil, err
	}

	return &wallet, nil
}

// GetWalletByUserID 根据用户 ID 获取钱包（不自动创建）
func (r *SysWalletRepository) GetWalletByUserID(userID uint) (*model.SystemWallet, error) {
	var wallet model.SystemWallet
	err := global.DB.Where("user_id = ?", userID).First(&wallet).Error
	if err != nil {
		return nil, err
	}
	return &wallet, nil
}

// UpdateBalance 更新钱包余额
func (r *SysWalletRepository) UpdateBalance(userID uint, newBalance float64) error {
	return global.DB.Model(&model.SystemWallet{}).Where("user_id = ?", userID).
		Update("balance", newBalance).Error
}

// UpdateBalanceTx 在事务中更新钱包余额
func (r *SysWalletRepository) UpdateBalanceTx(tx *gorm.DB, userID uint, newBalance float64) error {
	return tx.Model(&model.SystemWallet{}).Where("user_id = ?", userID).
		Update("balance", newBalance).Error
}

// ─────────────────────────────────────────────
//  钱包流水
// ─────────────────────────────────────────────

// CreateTransaction 创建钱包流水
func (r *SysWalletRepository) CreateTransaction(tx *model.WalletTransaction) error {
	return global.DB.Create(tx).Error
}

// CreateTransactionTx 在事务中创建钱包流水
func (r *SysWalletRepository) CreateTransactionTx(dbTx *gorm.DB, tx *model.WalletTransaction) error {
	return dbTx.Create(tx).Error
}

// ExistsTransactionByRefID 检查是否已存在指定 RefID 的流水记录
func (r *SysWalletRepository) ExistsTransactionByRefID(refID string) (bool, error) {
	var count int64
	err := global.DB.Model(&model.WalletTransaction{}).Where("ref_id = ?", refID).Count(&count).Error
	return count > 0, err
}

// GetTransactionByUserRefTypeRefIDInRange 根据用户、流水类型、关联 ID 和时间范围获取单条钱包流水
func (r *SysWalletRepository) GetTransactionByUserRefTypeRefIDInRange(userID uint, refType, refID string, startAt, endAt time.Time) (*model.WalletTransaction, error) {
	var tx model.WalletTransaction
	err := global.DB.Where(
		"user_id = ? AND ref_type = ? AND ref_id = ? AND created_at >= ? AND created_at < ?",
		userID, refType, refID, startAt, endAt,
	).First(&tx).Error
	if err != nil {
		return nil, err
	}
	return &tx, nil
}

// CountTransactionsByUserRefTypeInRange 统计某用户在指定时间范围内的指定流水类型数量
func (r *SysWalletRepository) CountTransactionsByUserRefTypeInRange(userID uint, refType string, startAt, endAt time.Time) (int64, error) {
	var count int64
	err := global.DB.Model(&model.WalletTransaction{}).
		Where("user_id = ? AND ref_type = ? AND created_at >= ? AND created_at < ?", userID, refType, startAt, endAt).
		Count(&count).Error
	return count, err
}

// WalletTransactionFilter 流水查询筛选条件
type WalletTransactionFilter struct {
	UserID      *uint
	UserKeyword string
	RefType     string
}

type WalletListFilter struct {
	UserKeyword string
}

type WalletAnalyticsFilter struct {
	StartAt     time.Time
	EndAt       time.Time
	RefTypes    []string
	UserKeyword string
}

type WalletAnalyticsSummaryAgg struct {
	WalletCount       int64
	ActiveWalletCount int64
	TotalBalance      float64
	IncomeTotal       float64
	ExpenseTotal      float64
}

type WalletAnalyticsDailyAgg struct {
	Date    string
	Income  float64
	Expense float64
}

type WalletAnalyticsRefTypeAgg struct {
	RefType string
	Income  float64
	Expense float64
	Count   int64
}

type WalletAnalyticsTopUserAgg struct {
	UserID        uint
	CharacterName string
	Amount        float64
}

type WalletAnalyticsOperatorAgg struct {
	OperatorID   uint
	OperatorName string
	Count        int64
	AmountTotal  float64
}

type WalletAnalyticsAdminAdjustStatsAgg struct {
	Count       int64
	AmountTotal float64
	ByOperator  []WalletAnalyticsOperatorAgg
}

type WalletAnalyticsLargeTransactionAgg struct {
	ID            uint
	UserID        uint
	CharacterName string
	Amount        float64
	RefType       string
	CreatedAt     time.Time
}

type WalletAnalyticsFrequentAdjustmentAgg struct {
	TargetUID          uint
	CharacterName      string
	AdjustCount        int64
	AmountTotal        float64
	LastAdjustmentTime time.Time
}

func applyWalletUserKeywordFilter(db *gorm.DB, userIDColumn string, userKeyword string) *gorm.DB {
	if strings.TrimSpace(userKeyword) == "" {
		return db
	}

	pattern := "%" + strings.ToLower(strings.TrimSpace(userKeyword)) + "%"
	return db.Where(
		userIDColumn+` IN (
			SELECT DISTINCT search_u.id
			FROM "user" search_u
			LEFT JOIN eve_character search_ec ON search_ec.user_id = search_u.id
			WHERE LOWER(search_u.nickname) LIKE ? OR LOWER(search_ec.character_name) LIKE ?
		)`,
		pattern, pattern,
	)
}

func applyWalletTransactionUserFilter(db *gorm.DB, userIDColumn string, refTypeColumn string, filter WalletTransactionFilter) *gorm.DB {
	if filter.UserID != nil {
		db = db.Where(userIDColumn+" = ?", *filter.UserID)
	}
	db = applyWalletUserKeywordFilter(db, userIDColumn, filter.UserKeyword)
	if filter.RefType != "" {
		db = db.Where(refTypeColumn+" = ?", filter.RefType)
	}
	return db
}

func applyWalletAnalyticsFilter(db *gorm.DB, userIDColumn string, refTypeColumn string, createdAtColumn string, filter WalletAnalyticsFilter) *gorm.DB {
	db = db.Where(createdAtColumn+" >= ? AND "+createdAtColumn+" < ?", filter.StartAt, filter.EndAt)
	db = applyWalletUserKeywordFilter(db, userIDColumn, filter.UserKeyword)
	if len(filter.RefTypes) > 0 {
		db = db.Where(refTypeColumn+" IN ?", filter.RefTypes)
	}
	return db
}

// ListTransactions 分页查询钱包流水
func (r *SysWalletRepository) ListTransactions(page, pageSize int, filter WalletTransactionFilter) ([]model.WalletTransaction, int64, error) {
	var txs []model.WalletTransaction
	var total int64
	offset := (page - 1) * pageSize

	db := applyWalletTransactionUserFilter(global.DB.Model(&model.WalletTransaction{}), "user_id", "ref_type", filter)

	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	if err := db.Order("created_at DESC").Offset(offset).Limit(pageSize).Find(&txs).Error; err != nil {
		return nil, 0, err
	}
	return txs, total, nil
}

// ListTransactionsWithCharacter 分页查询钱包流水（附带用户主人物名）
func (r *SysWalletRepository) ListTransactionsWithCharacter(page, pageSize int, filter WalletTransactionFilter) ([]model.TransactionWithCharacter, int64, error) {
	var results []model.TransactionWithCharacter
	var total int64
	offset := (page - 1) * pageSize

	countDB := applyWalletTransactionUserFilter(global.DB.Model(&model.WalletTransaction{}), "user_id", "ref_type", filter)
	if err := countDB.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	queryDB := global.DB.Table("wallet_transaction wt").
		Select(`wt.*,
			COALESCE(ec.character_name, '') AS character_name,
			COALESCE(NULLIF(u.nickname, ''), ec.character_name, '') AS nickname,
			CASE
				WHEN wt.operator_id = 0 THEN ''
				ELSE COALESCE(NULLIF(operator_u.nickname, ''), operator_ec.character_name, '')
			END AS operator_name`).
		Joins(`LEFT JOIN "user" u ON wt.user_id = u.id`).
		Joins("LEFT JOIN eve_character ec ON u.primary_character_id = ec.character_id").
		Joins(`LEFT JOIN "user" operator_u ON wt.operator_id = operator_u.id`).
		Joins("LEFT JOIN eve_character operator_ec ON operator_u.primary_character_id = operator_ec.character_id")
	queryDB = applyWalletTransactionUserFilter(queryDB, "wt.user_id", "wt.ref_type", filter)
	if err := queryDB.Order("wt.created_at DESC").Offset(offset).Limit(pageSize).Scan(&results).Error; err != nil {
		return nil, 0, err
	}
	return results, total, nil
}

// ─────────────────────────────────────────────
//  操作日志
// ─────────────────────────────────────────────

// CreateLog 创建操作日志
func (r *SysWalletRepository) CreateLog(log *model.WalletLog) error {
	return global.DB.Create(log).Error
}

// CreateLogTx 在事务中创建操作日志
func (r *SysWalletRepository) CreateLogTx(dbTx *gorm.DB, log *model.WalletLog) error {
	return dbTx.Create(log).Error
}

// WalletLogFilter 日志查询筛选条件
type WalletLogFilter struct {
	OperatorID *uint
	TargetUID  *uint
	Action     string
}

// ListLogs 分页查询操作日志
func (r *SysWalletRepository) ListLogs(page, pageSize int, filter WalletLogFilter) ([]model.WalletLog, int64, error) {
	var logs []model.WalletLog
	var total int64
	offset := (page - 1) * pageSize

	db := global.DB.Model(&model.WalletLog{})
	if filter.OperatorID != nil {
		db = db.Where("operator_id = ?", *filter.OperatorID)
	}
	if filter.TargetUID != nil {
		db = db.Where("target_uid = ?", *filter.TargetUID)
	}
	if filter.Action != "" {
		db = db.Where("action = ?", filter.Action)
	}

	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	if err := db.Order("created_at DESC").Offset(offset).Limit(pageSize).Find(&logs).Error; err != nil {
		return nil, 0, err
	}
	return logs, total, nil
}

// ListLogsWithCharacter 分页查询操作日志（附带操作人和目标用户主人物名）
func (r *SysWalletRepository) ListLogsWithCharacter(page, pageSize int, filter WalletLogFilter) ([]model.LogWithCharacter, int64, error) {
	var results []model.LogWithCharacter
	var total int64
	offset := (page - 1) * pageSize

	countDB := global.DB.Model(&model.WalletLog{})
	if filter.OperatorID != nil {
		countDB = countDB.Where("operator_id = ?", *filter.OperatorID)
	}
	if filter.TargetUID != nil {
		countDB = countDB.Where("target_uid = ?", *filter.TargetUID)
	}
	if filter.Action != "" {
		countDB = countDB.Where("action = ?", filter.Action)
	}
	if err := countDB.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	queryDB := global.DB.Table("wallet_log wl").
		Select(`wl.*,
			COALESCE(t_ec.character_name, '') AS target_character_name,
			COALESCE(o_ec.character_name, '') AS operator_character_name`).
		Joins(`LEFT JOIN "user" t_u ON wl.target_uid = t_u.id`).
		Joins("LEFT JOIN eve_character t_ec ON t_u.primary_character_id = t_ec.character_id").
		Joins(`LEFT JOIN "user" o_u ON wl.operator_id = o_u.id`).
		Joins("LEFT JOIN eve_character o_ec ON o_u.primary_character_id = o_ec.character_id")
	if filter.OperatorID != nil {
		queryDB = queryDB.Where("wl.operator_id = ?", *filter.OperatorID)
	}
	if filter.TargetUID != nil {
		queryDB = queryDB.Where("wl.target_uid = ?", *filter.TargetUID)
	}
	if filter.Action != "" {
		queryDB = queryDB.Where("wl.action = ?", filter.Action)
	}
	if err := queryDB.Order("wl.created_at DESC").Offset(offset).Limit(pageSize).Scan(&results).Error; err != nil {
		return nil, 0, err
	}
	return results, total, nil
}

// ─────────────────────────────────────────────
//  管理员：批量查询钱包
// ─────────────────────────────────────────────

// ListWallets 分页查询所有用户钱包
func (r *SysWalletRepository) ListWallets(page, pageSize int) ([]model.SystemWallet, int64, error) {
	var wallets []model.SystemWallet
	var total int64
	offset := (page - 1) * pageSize

	db := global.DB.Model(&model.SystemWallet{})
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	if err := db.Order("updated_at DESC").Offset(offset).Limit(pageSize).Find(&wallets).Error; err != nil {
		return nil, 0, err
	}
	return wallets, total, nil
}

// ListWalletsWithCharacter 分页查询所有用户钱包（附带主人物名）
func buildWalletListWithCharacterQuery(db *gorm.DB, filter WalletListFilter) *gorm.DB {
	query := db.Table("system_wallet sw").
		Select("sw.*, COALESCE(ec.character_name, '') AS character_name").
		Joins(`LEFT JOIN "user" u ON sw.user_id = u.id`).
		Joins("LEFT JOIN eve_character ec ON u.primary_character_id = ec.character_id")

	return applyWalletUserKeywordFilter(query, "sw.user_id", filter.UserKeyword)
}

// ListWalletsWithCharacter 分页查询所有用户钱包（附带主人物名）
func (r *SysWalletRepository) ListWalletsWithCharacter(page, pageSize int, filter WalletListFilter) ([]model.WalletWithCharacter, int64, error) {
	var results []model.WalletWithCharacter
	var total int64
	offset := (page - 1) * pageSize

	db := applyWalletUserKeywordFilter(global.DB.Model(&model.SystemWallet{}), "user_id", filter.UserKeyword)
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	err := buildWalletListWithCharacterQuery(global.DB, filter).
		Order("sw.updated_at DESC").
		Offset(offset).Limit(pageSize).
		Scan(&results).Error
	if err != nil {
		return nil, 0, err
	}
	return results, total, nil
}

func (r *SysWalletRepository) GetWalletAnalyticsSummary(filter WalletAnalyticsFilter) (WalletAnalyticsSummaryAgg, error) {
	var result WalletAnalyticsSummaryAgg

	activeUsersQuery := applyWalletAnalyticsFilter(
		global.DB.Model(&model.WalletTransaction{}).Select("DISTINCT user_id"),
		"user_id",
		"ref_type",
		"created_at",
		filter,
	)

	if err := global.DB.Model(&model.SystemWallet{}).
		Where("user_id IN (?)", activeUsersQuery).
		Count(&result.WalletCount).Error; err != nil {
		return result, err
	}
	result.ActiveWalletCount = result.WalletCount

	if err := global.DB.Model(&model.SystemWallet{}).
		Select("COALESCE(SUM(balance), 0)").
		Where("user_id IN (?)", activeUsersQuery).
		Scan(&result.TotalBalance).Error; err != nil {
		return result, err
	}

	type txTotals struct {
		Income  float64
		Expense float64
	}
	var totals txTotals
	txQuery := applyWalletAnalyticsFilter(global.DB.Model(&model.WalletTransaction{}), "user_id", "ref_type", "created_at", filter)
	if err := txQuery.Select(`
		COALESCE(SUM(CASE WHEN amount > 0 THEN amount ELSE 0 END), 0) AS income,
		COALESCE(SUM(CASE WHEN amount < 0 THEN -amount ELSE 0 END), 0) AS expense
	`).Scan(&totals).Error; err != nil {
		return result, err
	}

	result.IncomeTotal = totals.Income
	result.ExpenseTotal = totals.Expense
	return result, nil
}

func (r *SysWalletRepository) ListWalletAnalyticsDailySeries(filter WalletAnalyticsFilter) ([]WalletAnalyticsDailyAgg, error) {
	var rows []WalletAnalyticsDailyAgg
	query := applyWalletAnalyticsFilter(
		global.DB.Model(&model.WalletTransaction{}),
		"user_id",
		"ref_type",
		"created_at",
		filter,
	)

	err := query.Select(`
		DATE(created_at) AS date,
		COALESCE(SUM(CASE WHEN amount > 0 THEN amount ELSE 0 END), 0) AS income,
		COALESCE(SUM(CASE WHEN amount < 0 THEN -amount ELSE 0 END), 0) AS expense
	`).
		Group("DATE(created_at)").
		Order("date ASC").
		Scan(&rows).Error
	return rows, err
}

func (r *SysWalletRepository) ListWalletAnalyticsRefTypeBreakdown(filter WalletAnalyticsFilter) ([]WalletAnalyticsRefTypeAgg, error) {
	var rows []WalletAnalyticsRefTypeAgg
	query := applyWalletAnalyticsFilter(
		global.DB.Model(&model.WalletTransaction{}),
		"user_id",
		"ref_type",
		"created_at",
		filter,
	)

	err := query.Select(`
		ref_type,
		COALESCE(SUM(CASE WHEN amount > 0 THEN amount ELSE 0 END), 0) AS income,
		COALESCE(SUM(CASE WHEN amount < 0 THEN -amount ELSE 0 END), 0) AS expense,
		COUNT(*) AS count
	`).
		Group("ref_type").
		Order("COUNT(*) DESC, ref_type ASC").
		Scan(&rows).Error
	return rows, err
}

func (r *SysWalletRepository) ListWalletAnalyticsTopUsers(filter WalletAnalyticsFilter, inflow bool, topN int) ([]WalletAnalyticsTopUserAgg, error) {
	var rows []WalletAnalyticsTopUserAgg
	query := global.DB.Table("wallet_transaction wt").
		Joins(`LEFT JOIN "user" u ON wt.user_id = u.id`).
		Joins("LEFT JOIN eve_character ec ON u.primary_character_id = ec.character_id")
	query = applyWalletAnalyticsFilter(query, "wt.user_id", "wt.ref_type", "wt.created_at", filter)
	if inflow {
		query = query.Where("wt.amount > 0")
	} else {
		query = query.Where("wt.amount < 0")
	}

	err := query.Select(`
		wt.user_id,
		COALESCE(ec.character_name, '') AS character_name,
		COALESCE(SUM(ABS(wt.amount)), 0) AS amount
	`).
		Group("wt.user_id, ec.character_name").
		Order("amount DESC, wt.user_id ASC").
		Limit(topN).
		Scan(&rows).Error
	return rows, err
}

func (r *SysWalletRepository) GetWalletAnalyticsAdminAdjustStats(filter WalletAnalyticsFilter, topN int) (WalletAnalyticsAdminAdjustStatsAgg, error) {
	result := WalletAnalyticsAdminAdjustStatsAgg{ByOperator: make([]WalletAnalyticsOperatorAgg, 0)}
	query := global.DB.Table("wallet_log wl")
	query = query.Where("wl.created_at >= ? AND wl.created_at < ?", filter.StartAt, filter.EndAt)
	query = applyWalletUserKeywordFilter(query, "wl.target_uid", filter.UserKeyword)

	type totalRow struct {
		Count       int64
		AmountTotal float64
	}
	var total totalRow
	if err := query.Select("COUNT(*) AS count, COALESCE(SUM(wl.amount), 0) AS amount_total").Scan(&total).Error; err != nil {
		return result, err
	}
	result.Count = total.Count
	result.AmountTotal = total.AmountTotal

	var byOperator []WalletAnalyticsOperatorAgg
	err := query.
		Select(`
			wl.operator_id,
			CASE
				WHEN wl.operator_id = 0 THEN ''
				ELSE COALESCE(NULLIF(operator_u.nickname, ''), operator_ec.character_name, '')
			END AS operator_name,
			COUNT(*) AS count,
			COALESCE(SUM(wl.amount), 0) AS amount_total
		`).
		Joins(`LEFT JOIN "user" operator_u ON wl.operator_id = operator_u.id`).
		Joins("LEFT JOIN eve_character operator_ec ON operator_u.primary_character_id = operator_ec.character_id").
		Group("wl.operator_id, operator_u.nickname, operator_ec.character_name").
		Order("amount_total DESC, wl.operator_id ASC").
		Limit(topN).
		Scan(&byOperator).Error
	if err != nil {
		return result, err
	}
	result.ByOperator = byOperator
	return result, nil
}

func (r *SysWalletRepository) ListWalletAnalyticsAbsoluteAmounts(filter WalletAnalyticsFilter) ([]float64, error) {
	var values []float64
	query := applyWalletAnalyticsFilter(global.DB.Model(&model.WalletTransaction{}), "user_id", "ref_type", "created_at", filter)
	err := query.Select("ABS(amount) AS amount").Where("amount <> 0").Pluck("ABS(amount)", &values).Error
	return values, err
}

func (r *SysWalletRepository) ListWalletAnalyticsLargeTransactions(filter WalletAnalyticsFilter, minAbsAmount float64, topN int) ([]WalletAnalyticsLargeTransactionAgg, error) {
	var rows []WalletAnalyticsLargeTransactionAgg
	query := global.DB.Table("wallet_transaction wt").
		Select(`
			wt.id,
			wt.user_id,
			COALESCE(ec.character_name, '') AS character_name,
			wt.amount,
			wt.ref_type,
			wt.created_at
		`).
		Joins(`LEFT JOIN "user" u ON wt.user_id = u.id`).
		Joins("LEFT JOIN eve_character ec ON u.primary_character_id = ec.character_id")
	query = applyWalletAnalyticsFilter(query, "wt.user_id", "wt.ref_type", "wt.created_at", filter).
		Where("ABS(wt.amount) >= ?", minAbsAmount).
		Order("ABS(wt.amount) DESC, wt.created_at DESC").
		Limit(topN)
	err := query.Scan(&rows).Error
	return rows, err
}

func (r *SysWalletRepository) ListWalletAnalyticsFrequentAdjustments(filter WalletAnalyticsFilter, topN int) ([]WalletAnalyticsFrequentAdjustmentAgg, error) {
	var rows []WalletAnalyticsFrequentAdjustmentAgg
	query := global.DB.Table("wallet_log wl").
		Select(`
			wl.target_uid,
			COALESCE(ec.character_name, '') AS character_name,
			COUNT(*) AS adjust_count,
			COALESCE(SUM(wl.amount), 0) AS amount_total,
			MAX(wl.created_at) AS last_adjustment_time
		`).
		Joins(`LEFT JOIN "user" u ON wl.target_uid = u.id`).
		Joins("LEFT JOIN eve_character ec ON u.primary_character_id = ec.character_id").
		Where("wl.created_at >= ? AND wl.created_at < ?", filter.StartAt, filter.EndAt)
	query = applyWalletUserKeywordFilter(query, "wl.target_uid", filter.UserKeyword)
	err := query.Group("wl.target_uid, ec.character_name, DATE(wl.created_at)").
		Having("COUNT(*) >= 3").
		Order("adjust_count DESC, amount_total DESC").
		Limit(topN).
		Scan(&rows).Error
	return rows, err
}

func (r *SysWalletRepository) ListWalletAnalyticsOperatorConcentration(filter WalletAnalyticsFilter, topN int) ([]WalletAnalyticsOperatorAgg, error) {
	var rows []WalletAnalyticsOperatorAgg
	query := global.DB.Table("wallet_log wl").
		Select(`
			wl.operator_id,
			CASE
				WHEN wl.operator_id = 0 THEN ''
				ELSE COALESCE(NULLIF(operator_u.nickname, ''), operator_ec.character_name, '')
			END AS operator_name,
			COUNT(*) AS count,
			COALESCE(SUM(wl.amount), 0) AS amount_total
		`).
		Joins(`LEFT JOIN "user" operator_u ON wl.operator_id = operator_u.id`).
		Joins("LEFT JOIN eve_character operator_ec ON operator_u.primary_character_id = operator_ec.character_id").
		Where("wl.created_at >= ? AND wl.created_at < ?", filter.StartAt, filter.EndAt)
	query = applyWalletUserKeywordFilter(query, "wl.target_uid", filter.UserKeyword)
	err := query.Group("wl.operator_id, operator_u.nickname, operator_ec.character_name").
		Order("amount_total DESC, wl.operator_id ASC").
		Limit(topN).
		Scan(&rows).Error
	return rows, err
}

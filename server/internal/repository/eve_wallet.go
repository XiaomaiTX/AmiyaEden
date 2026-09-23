package repository

import (
	"time"

	"amiya-eden/global"
	"amiya-eden/internal/model"

	"gorm.io/gorm"
)

type EveWalletRepository struct{}

func NewEveWalletRepository() *EveWalletRepository {
	return &EveWalletRepository{}
}

func (r *EveWalletRepository) GetWallet(characterID int) (*model.EVECharacterWallet, error) {
	var wallet model.EVECharacterWallet
	err := global.DB.Where("character_id = ?", characterID).First(&wallet).Error
	if err != nil {
		return nil, err
	}
	return &wallet, nil
}

func (r *EveWalletRepository) getWalletJournalsQuery(db *gorm.DB, characterID int64, refTypes []string) *gorm.DB {
	query := db.Model(&model.EVECharacterWalletJournal{}).Where("character_id = ?", characterID)
	if len(refTypes) > 0 {
		query = query.Where("ref_type IN ?", refTypes)
	}
	return query
}

func (r *EveWalletRepository) getWalletJournalRefTypesQuery(db *gorm.DB, characterID int64) *gorm.DB {
	return db.Model(&model.EVECharacterWalletJournal{}).
		Where("character_id = ?", characterID).
		Distinct().
		Order("ref_type ASC")
}

// GetWalletJournals 分页获取人物钱包流水
func (r *EveWalletRepository) GetWalletJournals(characterID int64, page, pageSize int, refTypes []string) ([]model.EVECharacterWalletJournal, int64, error) {
	var journals []model.EVECharacterWalletJournal
	var total int64

	db := r.getWalletJournalsQuery(global.DB, characterID, refTypes)

	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	err := db.Order("date DESC").Offset((page - 1) * pageSize).Limit(pageSize).Find(&journals).Error
	if err != nil {
		return nil, 0, err
	}
	return journals, total, nil
}

// ListWalletJournalRefTypes 获取人物钱包流水中出现过的所有交易类型
func (r *EveWalletRepository) ListWalletJournalRefTypes(characterID int64) ([]string, error) {
	var refTypes []string
	err := r.getWalletJournalRefTypesQuery(global.DB, characterID).Pluck("ref_type", &refTypes).Error
	if err != nil {
		return nil, err
	}
	return refTypes, nil
}

// SumBalanceByCharacterIDs 汇总多个人物的钱包余额
func (r *EveWalletRepository) SumBalanceByCharacterIDs(characterIDs []int64) (float64, error) {
	if len(characterIDs) == 0 {
		return 0, nil
	}
	var total float64
	err := global.DB.Model(&model.EVECharacterWallet{}).
		Where("character_id IN ?", characterIDs).
		Select("COALESCE(SUM(balance), 0)").
		Scan(&total).Error
	return total, err
}

// WalletJournalFlowEntry 钱包分析用的瘦流水条目。
// 只取分析需要的列，避免为聚合把整行（含 description/reason 等大字段）搬进内存。
type WalletJournalFlowEntry struct {
	Date    time.Time
	Amount  float64
	Tax     float64
	Balance float64
	RefType string
}

// listWalletJournalFlowEntriesQuery 构造流水条目查询：左闭右开区间 [from, to)，按入账顺序升序。
// 升序 + (date, id) 二级排序是"当日最后一笔的 balance 即日终余额"的前提。
func (r *EveWalletRepository) listWalletJournalFlowEntriesQuery(db *gorm.DB, characterID int64, from, to time.Time) *gorm.DB {
	return db.Model(&model.EVECharacterWalletJournal{}).
		Where("character_id = ? AND date >= ? AND date < ?", characterID, from, to).
		Select("date, amount, tax, balance, ref_type").
		Order("date ASC").
		Order("id ASC")
}

// ListWalletJournalFlowEntries 按入账顺序取回 [from, to) 区间内的钱包流水条目。
// 区间与 UTC 自然日的换算由调用方负责（to 取"最后一天 + 1 天"）。
func (r *EveWalletRepository) ListWalletJournalFlowEntries(characterID int64, from, to time.Time) ([]WalletJournalFlowEntry, error) {
	rows := make([]WalletJournalFlowEntry, 0)
	if err := r.listWalletJournalFlowEntriesQuery(global.DB, characterID, from, to).Find(&rows).Error; err != nil {
		return nil, err
	}
	return rows, nil
}

// getWalletJournalLastBalanceBeforeQuery 构造"某一时刻之前最后一笔流水"查询：倒序取一条。
func (r *EveWalletRepository) getWalletJournalLastBalanceBeforeQuery(db *gorm.DB, characterID int64, before time.Time) *gorm.DB {
	return db.Model(&model.EVECharacterWalletJournal{}).
		Where("character_id = ? AND date < ?", characterID, before).
		Order("date DESC").
		Order("id DESC").
		Limit(1)
}

// GetWalletJournalLastBalanceBefore 取 before 之前最后一笔流水的余额，即该时刻的账户余额。
// 返回 nil 表示此前没有任何流水（调用方需退化为"区间首笔的 balance - amount"）。
func (r *EveWalletRepository) GetWalletJournalLastBalanceBefore(characterID int64, before time.Time) (*float64, error) {
	balances := make([]float64, 0, 1)
	if err := r.getWalletJournalLastBalanceBeforeQuery(global.DB, characterID, before).
		Pluck("balance", &balances).Error; err != nil {
		return nil, err
	}
	if len(balances) == 0 {
		return nil, nil
	}
	return &balances[0], nil
}

// getWalletJournalFirstDateQuery 构造"库内最早一笔流水日期"查询（排序 + LIMIT 1）。
func (r *EveWalletRepository) getWalletJournalFirstDateQuery(db *gorm.DB, characterID int64) *gorm.DB {
	return db.Model(&model.EVECharacterWalletJournal{}).
		Where("character_id = ?", characterID).
		Order("date ASC").
		Limit(1)
}

// getWalletJournalLastDateQuery 构造"库内最晚一笔流水日期"查询（排序 + LIMIT 1）。
func (r *EveWalletRepository) getWalletJournalLastDateQuery(db *gorm.DB, characterID int64) *gorm.DB {
	return db.Model(&model.EVECharacterWalletJournal{}).
		Where("character_id = ?", characterID).
		Order("date DESC").
		Limit(1)
}

// GetWalletJournalDateBounds 返回该人物钱包流水的可得时间范围（库内最早 / 最晚一笔）。
// 两者均为 nil 表示库内没有任何流水。
// 前端用它约束日期选择器下限，而不是写死 ESI 的 30 天窗口：本地表只增不删，可得范围会随运行时长增长。
//
// 取"排序 + LIMIT 1"而不是 MIN/MAX 聚合：聚合结果丢失列类型，sqlite（测试库）会把日期
// 当裸字符串还回来，扫不进 time.Time；取具体行也与全库其他读时间戳的写法一致。
func (r *EveWalletRepository) GetWalletJournalDateBounds(characterID int64) (*time.Time, *time.Time, error) {
	firstDates := make([]time.Time, 0, 1)
	if err := r.getWalletJournalFirstDateQuery(global.DB, characterID).
		Pluck("date", &firstDates).Error; err != nil {
		return nil, nil, err
	}

	lastDates := make([]time.Time, 0, 1)
	if err := r.getWalletJournalLastDateQuery(global.DB, characterID).
		Pluck("date", &lastDates).Error; err != nil {
		return nil, nil, err
	}

	var minDate *time.Time
	var maxDate *time.Time
	if len(firstDates) > 0 {
		minDate = &firstDates[0]
	}
	if len(lastDates) > 0 {
		maxDate = &lastDates[0]
	}
	return minDate, maxDate, nil
}

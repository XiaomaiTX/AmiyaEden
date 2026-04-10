package esi

import (
	"amiya-eden/global"
	"amiya-eden/internal/model"

	"context"
	"fmt"
	"time"

	"gorm.io/gorm/clause"
)

func init() {
	Register(&WalletTask{})
}

// WalletTask 人物钱包刷新任务
type WalletTask struct{}

func (t *WalletTask) Name() string        { return "character_wallet" }
func (t *WalletTask) Description() string { return "人物钱包信息" }
func (t *WalletTask) Priority() Priority  { return PriorityNormal }

func (t *WalletTask) Interval() RefreshInterval {
	return RefreshInterval{
		Active:   12 * time.Hour,
		Inactive: 7 * 24 * time.Hour,
	}
}

func (t *WalletTask) RequiredScopes() []TaskScope {
	return []TaskScope{
		{Scope: "esi-wallet.read_character_wallet.v1", Description: "读取人物钱包信息"},
	}
}

// WalletJournalEntry 单条钱包交易记录
type WalletJournalEntry struct {
	Amount        float64   `json:"amount"`
	Balance       float64   `json:"balance"`
	ContextID     int64     `json:"context_id"`
	ContextIDType string    `json:"context_id_type"`
	Date          time.Time `json:"date"`
	Description   string    `json:"description"`
	FirstPartyID  int64     `json:"first_party_id"`
	ID            int64     `json:"id"`
	Reason        string    `json:"reason"`
	RefType       string    `json:"ref_type"`
	SecondPartyID int64     `json:"second_party_id"`
	Tax           float64   `json:"tax"`
	TaxReceiverID int64     `json:"tax_receiver_id"`
}

// WalletJournalResult 钱包交易记录查询结果
type WalletTransaction struct {
	ClientID      int64     `json:"client_id"`
	Date          time.Time `json:"date"`
	IsBuy         bool      `json:"is_buy"`
	IsPersonal    bool      `json:"is_personal"`
	JournalRefID  int64     `json:"journal_ref_id"`
	LocationID    int64     `json:"location_id"`
	Quantity      int       `json:"quantity"`
	TransactionID int64     `json:"transaction_id"`
	TypeID        int       `json:"type_id"`
	UnitPrice     float64   `json:"unit_price"`
}

// WalletJournalResult 钱包交易记录查询结果
type WalletJournalResult []WalletJournalEntry

// Execute 执行钱包数据刷新
func (t *WalletTask) Execute(ctx *TaskContext) error {
	bgCtx := context.Background()

	// 1. 获取钱包余额
	var balance float64
	path := fmt.Sprintf("/characters/%d/wallet/", ctx.CharacterID)
	if err := ctx.Client.Get(bgCtx, path, ctx.AccessToken, &balance); err != nil {
		return fmt.Errorf("fetch wallet balance: %w", err)
	}

	// 2. 获取钱包记录
	var walletJournal WalletJournalResult
	path = fmt.Sprintf("/characters/%d/wallet/journal", ctx.CharacterID)
	if _, err := ctx.Client.GetPaginated(bgCtx, path, ctx.AccessToken, &walletJournal); err != nil {
		return fmt.Errorf("fetch wallet journal: %w", err)
	}

	// 3. 获取钱包市场交易
	walletTransactions, err := t.fetchWalletTransactions(bgCtx, ctx)
	if err != nil {
		return err
	}

	// 入库
	tx := global.DB.Begin()
	if tx.Error != nil {
		return fmt.Errorf("begin wallet transaction db tx: %w", tx.Error)
	}
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
			panic(r)
		}
	}()

	if err := tx.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "character_id"}},
		DoUpdates: clause.AssignmentColumns([]string{"balance", "update_time"}),
	}).Create(&model.EVECharacterWallet{
		CharacterID: ctx.CharacterID,
		Balance:     balance,
	}).Error; err != nil {
		tx.Rollback()
		return fmt.Errorf("upsert wallet balance: %w", err)
	}

	var waitingEntries []model.EVECharacterWalletJournal
	seenJournalIDs := make(map[int64]struct{}, len(walletJournal))
	for _, entry := range walletJournal {
		if _, ok := seenJournalIDs[entry.ID]; ok {
			continue
		}
		seenJournalIDs[entry.ID] = struct{}{}

		waitingEntries = append(waitingEntries, model.EVECharacterWalletJournal{
			ID:            entry.ID,
			CharacterID:   ctx.CharacterID,
			Amount:        entry.Amount,
			Balance:       entry.Balance,
			ContextID:     entry.ContextID,
			ContextIDType: entry.ContextIDType,
			Date:          entry.Date,
			Description:   entry.Description,
			FirstPartyID:  entry.FirstPartyID,
			Reason:        entry.Reason,
			RefType:       entry.RefType,
			SecondPartyID: entry.SecondPartyID,
			Tax:           entry.Tax,
			TaxReceiverID: entry.TaxReceiverID,
		})
	}
	if len(waitingEntries) > 0 {
		if err := tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&waitingEntries).Error; err != nil {
			tx.Rollback()
			return fmt.Errorf("insert wallet journal: %w", err)
		}
	}

	var waitingTransactions []model.EVECharacterWalletTransaction
	seenTransactionIDs := make(map[int64]struct{}, len(walletTransactions))
	for _, txEntry := range walletTransactions {
		if _, ok := seenTransactionIDs[txEntry.TransactionID]; ok {
			continue
		}
		seenTransactionIDs[txEntry.TransactionID] = struct{}{}

		waitingTransactions = append(waitingTransactions, model.EVECharacterWalletTransaction{
			TransactionID: txEntry.TransactionID,
			CharacterID:   ctx.CharacterID,
			ClientID:      txEntry.ClientID,
			Date:          txEntry.Date,
			IsBuy:         txEntry.IsBuy,
			IsPersonal:    txEntry.IsPersonal,
			JournalRefID:  txEntry.JournalRefID,
			LocationID:    txEntry.LocationID,
			Quantity:      txEntry.Quantity,
			TypeID:        txEntry.TypeID,
			UnitPrice:     txEntry.UnitPrice,
		})
	}
	if len(waitingTransactions) > 0 {
		if err := tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&waitingTransactions).Error; err != nil {
			tx.Rollback()
			return fmt.Errorf("insert wallet transactions: %w", err)
		}
	}
	if err := tx.Commit().Error; err != nil {
		return fmt.Errorf("commit wallet transaction db tx: %w", err)
	}

	return nil
}

func (t *WalletTask) fetchWalletTransactions(ctx context.Context, taskCtx *TaskContext) ([]WalletTransaction, error) {
	basePath := fmt.Sprintf("/characters/%d/wallet/transactions", taskCtx.CharacterID)
	var (
		results []WalletTransaction
		fromID  int64
	)
	seen := make(map[int64]struct{})

	for {
		path := basePath
		if fromID != 0 {
			path = fmt.Sprintf("%s?from_id=%d", basePath, fromID)
		}

		var batch []WalletTransaction
		if err := taskCtx.Client.Get(ctx, path, taskCtx.AccessToken, &batch); err != nil {
			return nil, fmt.Errorf("fetch wallet transactions: %w", err)
		}
		if len(batch) == 0 {
			return results, nil
		}

		nextFromID := batch[0].TransactionID
		for _, entry := range batch {
			if entry.TransactionID < nextFromID {
				nextFromID = entry.TransactionID
			}
			if _, ok := seen[entry.TransactionID]; ok {
				continue
			}
			seen[entry.TransactionID] = struct{}{}
			results = append(results, entry)
		}

		if fromID != 0 && nextFromID >= fromID {
			return nil, fmt.Errorf("fetch wallet transactions: pagination did not advance from_id=%d next_from_id=%d", fromID, nextFromID)
		}
		fromID = nextFromID
	}
}

package service

import (
	"amiya-eden/global"
	"amiya-eden/internal/model"
	"fmt"
	"testing"
	"time"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestCreditUserStoresSystemOperatorOnWalletTransaction(t *testing.T) {
	db := newSysWalletServiceTestDB(t)
	originalDB := global.DB
	global.DB = db
	defer func() { global.DB = originalDB }()

	svc := NewSysWalletService()
	if err := svc.CreditUser(42, 15.5, "shop order", "shop", "order:1"); err != nil {
		t.Fatalf("CreditUser() error = %v", err)
	}

	var txs []model.WalletTransaction
	if err := db.Order("id ASC").Find(&txs).Error; err != nil {
		t.Fatalf("load wallet transactions: %v", err)
	}
	if len(txs) != 1 {
		t.Fatalf("wallet transaction count = %d, want 1", len(txs))
	}

	tx := txs[0]
	if tx.UserID != 42 {
		t.Fatalf("wallet transaction user_id = %d, want 42", tx.UserID)
	}
	if tx.Amount != 15.5 {
		t.Fatalf("wallet transaction amount = %f, want 15.5", tx.Amount)
	}
	if tx.BalanceAfter != 15.5 {
		t.Fatalf("wallet transaction balance_after = %f, want 15.5", tx.BalanceAfter)
	}
	if tx.OperatorID != 0 {
		t.Fatalf("wallet transaction operator_id = %d, want 0", tx.OperatorID)
	}
	if tx.RefType != "shop" || tx.RefID != "order:1" || tx.Reason != "shop order" {
		t.Fatalf("unexpected transaction metadata: %+v", tx)
	}
}

func TestNormalizeLedgerPageSizeUsesLedgerStandardBounds(t *testing.T) {
	tests := []struct {
		name string
		size int
		want int
	}{
		{name: "defaults when zero", size: 0, want: 200},
		{name: "preserves smaller valid page", size: 20, want: 20},
		{name: "keeps ledger default", size: 200, want: 200},
		{name: "allows larger ledger page", size: 500, want: 500},
		{name: "caps at thousand", size: 5000, want: 1000},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := normalizeLedgerPageSize(tt.size); got != tt.want {
				t.Fatalf("normalizeLedgerPageSize(%d) = %d, want %d", tt.size, got, tt.want)
			}
		})
	}
}

func newSysWalletServiceTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	dsn := fmt.Sprintf("file:sys_wallet_service_test_%d?mode=memory&cache=shared", time.Now().UnixNano())
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&model.SystemWallet{}, &model.WalletTransaction{}); err != nil {
		t.Fatalf("auto migrate: %v", err)
	}
	return db
}

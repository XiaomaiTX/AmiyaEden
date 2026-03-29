package service

import "testing"

func TestBuildSystemWalletTransactionUsesDeltaAndSystemOperator(t *testing.T) {
	tx := buildSystemWalletTransaction(42, -15.5, 84.5, "shop order", "shop", "order:1")

	if tx.UserID != 42 {
		t.Fatalf("expected user id 42, got %d", tx.UserID)
	}
	if tx.Amount != -15.5 {
		t.Fatalf("expected delta amount to be preserved, got %f", tx.Amount)
	}
	if tx.BalanceAfter != 84.5 {
		t.Fatalf("expected balance after 84.5, got %f", tx.BalanceAfter)
	}
	if tx.OperatorID != 0 {
		t.Fatalf("expected system operator id 0, got %d", tx.OperatorID)
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

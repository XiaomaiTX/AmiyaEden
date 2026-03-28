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

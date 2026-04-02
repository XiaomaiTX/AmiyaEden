package service

import "testing"

func TestExclusiveRunGuardRejectsConcurrentRunsUntilFinished(t *testing.T) {
	var guard exclusiveRunGuard

	if err := guard.Start("奖励结算"); err != nil {
		t.Fatalf("expected first run to start, got %v", err)
	}
	if err := guard.Start("奖励结算"); err == nil {
		t.Fatal("expected second concurrent run to be rejected")
	}

	guard.Finish()

	if err := guard.Start("奖励结算"); err != nil {
		t.Fatalf("expected run to start again after Finish, got %v", err)
	}
}

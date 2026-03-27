package service

import (
	"testing"
	"time"
)

func TestSortCaptainCandidatesByLastOnlineDesc(t *testing.T) {
	now := time.Date(2026, 3, 27, 10, 0, 0, 0, time.UTC)
	older := now.Add(-2 * time.Hour)

	candidates := []NewbroCaptainCandidate{
		{CaptainUserID: 3, LastOnlineAt: nil},
		{CaptainUserID: 2, LastOnlineAt: &older},
		{CaptainUserID: 1, LastOnlineAt: &now},
		{CaptainUserID: 4, LastOnlineAt: &now},
	}

	sortCaptainCandidatesByLastOnline(candidates)

	got := []uint{
		candidates[0].CaptainUserID,
		candidates[1].CaptainUserID,
		candidates[2].CaptainUserID,
		candidates[3].CaptainUserID,
	}
	want := []uint{1, 4, 2, 3}

	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("unexpected order at index %d: got %v want %v", i, got, want)
		}
	}
}

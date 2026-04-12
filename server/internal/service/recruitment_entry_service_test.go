package service

import (
	"amiya-eden/internal/model"
	"testing"
	"time"
)

func TestResolveEntryStatus_ValidWhenUserCreatedAfterEntry(t *testing.T) {
	entryTime := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	userCreatedAt := entryTime.Add(24 * time.Hour) // user created 1 day after entry
	now := entryTime.Add(30 * 24 * time.Hour)

	result := resolveEntryStatus(entryTime, userCreatedAt, true, now, 90)
	if result != model.RecruitEntryStatusValid {
		t.Fatalf("resolveEntryStatus = %q, want %q", result, model.RecruitEntryStatusValid)
	}
}

func TestResolveEntryStatus_StalledWhenUserCreatedBeforeEntry(t *testing.T) {
	entryTime := time.Date(2026, 1, 10, 0, 0, 0, 0, time.UTC)
	userCreatedAt := entryTime.Add(-24 * time.Hour) // user created before entry
	now := entryTime.Add(5 * 24 * time.Hour)

	result := resolveEntryStatus(entryTime, userCreatedAt, true, now, 90)
	if result != model.RecruitEntryStatusStalled {
		t.Fatalf("resolveEntryStatus = %q, want %q", result, model.RecruitEntryStatusStalled)
	}
}

func TestResolveEntryStatus_StalledWhenExpired(t *testing.T) {
	entryTime := time.Date(2025, 10, 1, 0, 0, 0, 0, time.UTC)
	now := entryTime.Add(91 * 24 * time.Hour) // 91 days later

	result := resolveEntryStatus(entryTime, time.Time{}, false, now, 90)
	if result != model.RecruitEntryStatusStalled {
		t.Fatalf("resolveEntryStatus = %q, want %q", result, model.RecruitEntryStatusStalled)
	}
}

func TestResolveEntryStatus_OngoingWhenNoMatchYet(t *testing.T) {
	entryTime := time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)
	now := entryTime.Add(10 * 24 * time.Hour)

	result := resolveEntryStatus(entryTime, time.Time{}, false, now, 90)
	if result != model.RecruitEntryStatusOngoing {
		t.Fatalf("resolveEntryStatus = %q, want %q", result, model.RecruitEntryStatusOngoing)
	}
}

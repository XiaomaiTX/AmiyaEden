package service

import (
	"amiya-eden/internal/model"
	"testing"
	"time"
)

func TestBuildCaptainPlayerListItemsUsesCurrentPrimaryCharacterAndNickname(t *testing.T) {
	startedAt := time.Date(2026, 3, 27, 12, 0, 0, 0, time.UTC)
	rows := []model.NewbroCaptainAffiliation{
		{
			PlayerUserID:                    2001,
			PlayerPrimaryCharacterIDAtStart: 9001,
			CaptainUserID:                   3001,
			StartedAt:                       startedAt,
		},
	}
	users := map[uint]model.User{
		2001: {
			BaseModel:          model.BaseModel{ID: 2001},
			Nickname:           "Little Bee",
			PrimaryCharacterID: 9002,
		},
	}
	chars := map[int64]model.EveCharacter{
		9001: {CharacterID: 9001, CharacterName: "Old Main", PortraitURL: "old.png"},
		9002: {CharacterID: 9002, CharacterName: "Current Main", PortraitURL: "current.png"},
	}

	items, err := buildCaptainPlayerListItems(rows, users, chars, 3001, func(captainUserID, playerUserID uint) (float64, error) {
		if captainUserID != 3001 {
			t.Fatalf("expected captain user ID 3001, got %d", captainUserID)
		}
		if playerUserID != 2001 {
			t.Fatalf("expected player user ID 2001, got %d", playerUserID)
		}
		return 123.45, nil
	})
	if err != nil {
		t.Fatalf("build captain player list items: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(items))
	}

	item := items[0]
	if item.PlayerCharacterID != 9002 {
		t.Fatalf("expected current primary character ID 9002, got %d", item.PlayerCharacterID)
	}
	if item.PlayerCharacterName != "Current Main" {
		t.Fatalf("expected current primary character name, got %q", item.PlayerCharacterName)
	}
	if item.PlayerPortraitURL != "current.png" {
		t.Fatalf("expected current primary portrait URL, got %q", item.PlayerPortraitURL)
	}
	if item.PlayerNickname != "Little Bee" {
		t.Fatalf("expected nickname to be returned, got %q", item.PlayerNickname)
	}
	if item.AttributedBountyTotal != 123.45 {
		t.Fatalf("expected bounty total 123.45, got %v", item.AttributedBountyTotal)
	}
}

func TestBuildCaptainOverviewIncludesNicknameAndPrimaryCharacter(t *testing.T) {
	profile := captainProfile{
		Nickname:             "Bee Keeper",
		PrimaryCharacterID:   9002,
		PrimaryCharacterName: "Current Main",
	}

	overview := buildCaptainOverview(3001, profile, 7, 12, 123.45, 3)

	if overview.CaptainUserID != 3001 {
		t.Fatalf("expected captain user ID 3001, got %d", overview.CaptainUserID)
	}
	if overview.CaptainNickname != "Bee Keeper" {
		t.Fatalf("expected nickname to be returned, got %q", overview.CaptainNickname)
	}
	if overview.CaptainCharacterID != 9002 {
		t.Fatalf("expected primary character ID 9002, got %d", overview.CaptainCharacterID)
	}
	if overview.CaptainCharacterName != "Current Main" {
		t.Fatalf("expected primary character name to be returned, got %q", overview.CaptainCharacterName)
	}
	if overview.ActivePlayerCount != 7 || overview.HistoricalPlayerCount != 12 {
		t.Fatalf("unexpected player counts: %+v", overview)
	}
}

func TestBuildAdminAffiliationHistoryItemsUsesCurrentCaptainAndHistoricalPlayerCharacter(t *testing.T) {
	startedAt := time.Date(2026, 3, 25, 12, 0, 0, 0, time.UTC)
	endedAt := time.Date(2026, 3, 26, 12, 0, 0, 0, time.UTC)
	rows := []model.NewbroCaptainAffiliation{
		{
			BaseModel:                       model.BaseModel{ID: 1, CreatedAt: startedAt.Add(-time.Hour)},
			PlayerUserID:                    2001,
			PlayerPrimaryCharacterIDAtStart: 9001,
			CaptainUserID:                   3001,
			CreatedBy:                       3001,
			StartedAt:                       startedAt,
			EndedAt:                         &endedAt,
		},
	}
	captains := map[uint]captainProfile{
		3001: {
			Nickname:             "Captain Bee",
			PrimaryCharacterID:   8001,
			PrimaryCharacterName: "Captain Current Main",
		},
	}
	users := map[uint]model.User{
		2001: {
			BaseModel: model.BaseModel{ID: 2001},
			Nickname:  "Newbro One",
		},
	}
	chars := map[int64]model.EveCharacter{
		9001: {CharacterID: 9001, CharacterName: "Newbro Start Main"},
		8001: {CharacterID: 8001, CharacterName: "Captain Current Main"},
	}

	actors := map[uint]captainProfile{
		3001: {
			PrimaryCharacterID:   7001,
			PrimaryCharacterName: "Affiliation Actor",
		},
	}
	chars[7001] = model.EveCharacter{CharacterID: 7001, CharacterName: "Affiliation Actor"}

	items := buildAdminAffiliationHistoryItems(rows, captains, actors, users, chars)
	if len(items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(items))
	}

	item := items[0]
	if item.CaptainUserID != 3001 {
		t.Fatalf("expected captain user ID 3001, got %d", item.CaptainUserID)
	}
	if item.CaptainCharacterID != 8001 {
		t.Fatalf("expected current captain character ID 8001, got %d", item.CaptainCharacterID)
	}
	if item.CaptainCharacterName != "Captain Current Main" {
		t.Fatalf("expected current captain character name, got %q", item.CaptainCharacterName)
	}
	if item.CaptainNickname != "Captain Bee" {
		t.Fatalf("expected captain nickname, got %q", item.CaptainNickname)
	}
	if item.PlayerUserID != 2001 {
		t.Fatalf("expected player user ID 2001, got %d", item.PlayerUserID)
	}
	if item.PlayerCharacterID != 9001 {
		t.Fatalf("expected stored player character ID 9001, got %d", item.PlayerCharacterID)
	}
	if item.PlayerCharacterName != "Newbro Start Main" {
		t.Fatalf("expected stored player character name, got %q", item.PlayerCharacterName)
	}
	if item.PlayerNickname != "Newbro One" {
		t.Fatalf("expected player nickname, got %q", item.PlayerNickname)
	}
	if item.ChangedByCharacterName != "Affiliation Actor" {
		t.Fatalf("expected changed-by character name, got %q", item.ChangedByCharacterName)
	}
	if !item.StartedAt.Equal(startedAt) {
		t.Fatalf("expected started_at %v, got %v", startedAt, item.StartedAt)
	}
	if item.EndedAt == nil || !item.EndedAt.Equal(endedAt) {
		t.Fatalf("expected ended_at %v, got %v", endedAt, item.EndedAt)
	}
}

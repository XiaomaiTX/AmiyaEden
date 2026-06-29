package repository

import (
	"amiya-eden/global"
	"amiya-eden/internal/model"
	"database/sql"
	"fmt"
	"testing"
	"time"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func newNpcKillRepoTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := fmt.Sprintf("file:npc_kill_repo_%d?mode=memory&cache=shared", time.Now().UnixNano())
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&model.MapSolarSystem{}); err != nil {
		t.Fatalf("auto migrate mapSolarSystems: %v", err)
	}
	if err := db.AutoMigrate(&model.EVECharacterWalletJournal{}); err != nil {
		t.Fatalf("auto migrate wallet journals: %v", err)
	}
	return db
}

func TestNpcKillRepositoryGetSolarSystemNamesSelectsOnlyRequiredColumns(t *testing.T) {
	db := newNpcKillRepoTestDB(t)
	oldDB := global.DB
	global.DB = db
	t.Cleanup(func() { global.DB = oldDB })

	rows := []model.MapSolarSystem{
		{
			SolarSystemID:   30000142,
			SolarSystemName: "Jita",
			Border:          sql.NullBool{Bool: true, Valid: true},
			Fringe:          sql.NullBool{Bool: false, Valid: true},
		},
		{
			SolarSystemID:   30002187,
			SolarSystemName: "Amarr",
			Border:          sql.NullBool{Bool: false, Valid: true},
			Fringe:          sql.NullBool{Bool: true, Valid: true},
		},
	}
	if err := db.Create(&rows).Error; err != nil {
		t.Fatalf("seed mapSolarSystems: %v", err)
	}

	repo := NewNpcKillRepository()
	got, err := repo.GetSolarSystemNames([]int{30000142, 30002187, 99999999})
	if err != nil {
		t.Fatalf("GetSolarSystemNames returned error: %v", err)
	}

	if len(got) != 2 {
		t.Fatalf("name map size = %d, want 2", len(got))
	}
	if got[30000142] != "Jita" {
		t.Fatalf("solarSystem 30000142 = %q, want %q", got[30000142], "Jita")
	}
	if got[30002187] != "Amarr" {
		t.Fatalf("solarSystem 30002187 = %q, want %q", got[30002187], "Amarr")
	}
}

func TestNpcKillRepositoryListJournalsAppliesFilters(t *testing.T) {
	db := newNpcKillRepoTestDB(t)
	oldDB := global.DB
	global.DB = db
	t.Cleanup(func() { global.DB = oldDB })

	base := time.Date(2026, 6, 24, 12, 0, 0, 0, time.UTC)
	rows := []model.EVECharacterWalletJournal{
		{ID: 1, CharacterID: 1001, RefType: "bounty_prizes", ContextID: 30000142, Date: base, Amount: 100, Tax: -10},
		{ID: 2, CharacterID: 1001, RefType: "incursion_payout", ContextID: 0, Date: base.Add(time.Minute), Amount: 40000000, Tax: 0},
		{ID: 3, CharacterID: 1002, RefType: "bounty_prizes", ContextID: 30000143, Date: base.Add(2 * time.Minute), Amount: 200, Tax: -20},
		{ID: 4, CharacterID: 1001, RefType: "agent_mission_reward", ContextID: 0, Date: base.Add(3 * time.Minute), Amount: 50, Tax: 0},
	}
	if err := db.Create(&rows).Error; err != nil {
		t.Fatalf("seed wallet journals: %v", err)
	}

	repo := NewNpcKillRepository()
	got, err := repo.ListJournals(NpcKillJournalQuery{
		CharacterIDs: []int64{1001, 1002},
		RefTypes:     []string{"incursion_payout"},
		MinAmount:    floatPtr(1000000),
		MaxAmount:    floatPtr(50000000),
	})
	if err != nil {
		t.Fatalf("ListJournals returned error: %v", err)
	}
	if len(got) != 1 || got[0].ID != 2 {
		t.Fatalf("expected only incursion payout row, got %+v", got)
	}

	got, err = repo.ListJournals(NpcKillJournalQuery{
		CharacterIDs:   []int64{1001, 1002},
		SolarSystemIDs: []int{30000142},
	})
	if err != nil {
		t.Fatalf("ListJournals with system filter returned error: %v", err)
	}
	if len(got) != 1 || got[0].ID != 1 {
		t.Fatalf("expected only bounty row in requested system, got %+v", got)
	}
}

func floatPtr(value float64) *float64 {
	return &value
}

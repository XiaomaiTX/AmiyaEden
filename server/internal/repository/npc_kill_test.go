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

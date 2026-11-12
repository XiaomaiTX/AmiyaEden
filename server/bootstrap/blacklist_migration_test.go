package bootstrap

import (
	"amiya-eden/internal/model"
	"amiya-eden/internal/testutil"
	"fmt"
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func TestBlacklistMigrationSQLite(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(fmt.Sprintf("file:blacklist_migration_%d?mode=memory&cache=shared", time.Now().UnixNano())), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { sqlDB, _ := db.DB(); _ = sqlDB.Close() })
	checkBlacklistMigration(t, db)
}

func TestBlacklistMigrationPostgres(t *testing.T) {
	checkBlacklistMigration(t, testutil.Postgres(t))
}

func checkBlacklistMigration(t *testing.T, db *gorm.DB) {
	t.Helper()
	if err := migrateBlacklist(db); err != nil {
		t.Fatal(err)
	}
	entry := model.BlacklistEntry{Kind: model.BlacklistKindQQ, Value: "12345678", Display: "12345678", Reason: "scam"}
	if err := db.Create(&entry).Error; err != nil {
		t.Fatal(err)
	}
	unresolved := model.BlacklistUnresolvedName{NameKey: "ghost pilot", Name: "Ghost Pilot", Kind: model.BlacklistUnresolvedNotFound}
	if err := db.Create(&unresolved).Error; err != nil {
		t.Fatal(err)
	}
	for range 2 {
		if err := migrateBlacklist(db); err != nil {
			t.Fatal(err)
		}
	}
	var kept model.BlacklistEntry
	if err := db.First(&kept, entry.ID).Error; err != nil || kept.Reason != "scam" {
		t.Fatalf("migration altered existing entry: %+v, %v", kept, err)
	}

	duplicate := model.BlacklistEntry{Kind: model.BlacklistKindQQ, Value: "12345678", Display: "12345678"}
	if err := db.Create(&duplicate).Error; err == nil {
		t.Fatal("duplicate (kind, value) accepted")
	}
	otherKind := model.BlacklistEntry{Kind: model.BlacklistKindCharacter, Value: "12345678", Display: "Some Pilot"}
	if err := db.Create(&otherKind).Error; err != nil {
		t.Fatal("same value under another kind rejected:", err)
	}
	if err := db.Create(&model.BlacklistUnresolvedName{NameKey: "ghost pilot", Name: "GHOST PILOT", Kind: model.BlacklistUnresolvedInvalidFormat}).Error; err == nil {
		t.Fatal("duplicate unresolved name key accepted")
	}
	run := model.BlacklistRefreshRun{OperatorUserID: 1, FinishedAt: time.Now(), QQs: 3}
	if err := db.Create(&run).Error; err != nil {
		t.Fatal(err)
	}
	for _, table := range []string{"blacklist_entry", "blacklist_unresolved_name", "blacklist_refresh_run"} {
		if !db.Migrator().HasTable(table) {
			t.Fatalf("missing table %s", table)
		}
	}
	if !db.Migrator().HasColumn(&model.BlacklistRefreshRun{}, "qqs") {
		t.Fatal("refresh run is missing the qqs column")
	}
	// Simulate upgrading a database that already has refresh history but
	// predates the lookup_failed counter.
	if err := db.Migrator().DropColumn(&model.BlacklistRefreshRun{}, "lookup_failed"); err != nil {
		t.Fatal(err)
	}
	if err := migrateBlacklist(db); err != nil {
		t.Fatal(err)
	}
	var oldRun model.BlacklistRefreshRun
	if err := db.First(&oldRun, run.ID).Error; err != nil || oldRun.QQs != 3 || oldRun.LookupFailed != 0 {
		t.Fatalf("old refresh history after upgrade: %+v, %v", oldRun, err)
	}
	run = model.BlacklistRefreshRun{OperatorUserID: 1, FinishedAt: time.Now(), LookupFailed: 7}
	if err := db.Create(&run).Error; err != nil {
		t.Fatal(err)
	}
	var saved model.BlacklistRefreshRun
	if err := db.First(&saved, run.ID).Error; err != nil || saved.LookupFailed != 7 {
		t.Fatalf("lookup_failed not persisted: %+v, %v", saved, err)
	}
}

package bootstrap

import (
	"amiya-eden/internal/model"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func contactMigrationTestDB(t *testing.T, postgresDSN string) *gorm.DB {
	t.Helper()
	cfg := &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)}
	var db *gorm.DB
	var err error
	if postgresDSN != "" {
		base, openErr := gorm.Open(postgres.Open(postgresDSN), cfg)
		if openErr != nil {
			t.Fatal(openErr)
		}
		name := fmt.Sprintf("contact_migration_test_%d", time.Now().UnixNano())
		if err := base.Exec("CREATE SCHEMA " + name).Error; err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { base.Exec("DROP SCHEMA " + name + " CASCADE"); sqlDB, _ := base.DB(); _ = sqlDB.Close() })
		db, err = gorm.Open(postgres.Open(postgresDSN+" search_path="+name), cfg)
	} else {
		db, err = gorm.Open(sqlite.Open(fmt.Sprintf("file:contact_migration_%d?mode=memory&cache=shared", time.Now().UnixNano())), cfg)
	}
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { sqlDB, _ := db.DB(); _ = sqlDB.Close() })
	if err := db.AutoMigrate(&model.User{}, &model.EveCharacter{}, &model.UserRole{}); err != nil {
		t.Fatal(err)
	}
	for _, table := range contactCleanupCharacterTables {
		if err := db.Exec("CREATE TABLE " + table + " (character_id bigint)").Error; err != nil {
			t.Fatal(err)
		}
	}
	for _, stmt := range []string{"CREATE TABLE mumble_identity (seat_user_id bigint)", "CREATE TABLE shop_order (user_id bigint)"} {
		if err := db.Exec(stmt).Error; err != nil {
			t.Fatal(err)
		}
	}
	return db
}

func testContactCleanup(t *testing.T, db *gorm.DB) {
	t.Helper()
	for _, user := range []model.User{
		{BaseModel: model.BaseModel{ID: 1}, QQ: "123456"},
		{BaseModel: model.BaseModel{ID: 2}, QQ: " \t\n", DiscordID: "\u3000"},
		{BaseModel: model.BaseModel{ID: 3}, DiscordID: "discord-only"},
		{BaseModel: model.BaseModel{ID: 4}},
	} {
		if err := db.Create(&user).Error; err != nil {
			t.Fatal(err)
		}
	}
	db.Delete(&model.User{}, 4)
	for _, id := range []uint{1, 2, 4} {
		if err := db.Create(&model.EveCharacter{UserID: id, CharacterID: int64(9000 + id), CharacterName: "Test"}).Error; err != nil {
			t.Fatal(err)
		}
		if err := db.Create(&model.UserRole{UserID: id, RoleCode: model.RoleUser}).Error; err != nil {
			t.Fatal(err)
		}
		if err := db.Exec("INSERT INTO mumble_identity VALUES (?)", id).Error; err != nil {
			t.Fatal(err)
		}
		if err := db.Exec("INSERT INTO shop_order VALUES (?)", id).Error; err != nil {
			t.Fatal(err)
		}
		for _, table := range contactCleanupCharacterTables {
			if err := db.Exec("INSERT INTO "+table+" VALUES (?)", 9000+id).Error; err != nil {
				t.Fatal(err)
			}
		}
	}
	for i := 0; i < 2; i++ {
		if err := migrateRequiredUserContact(db); err != nil {
			t.Fatal(err)
		}
	}
	for _, check := range []struct {
		table string
		want  int64
	}{{"user", 2}, {"eve_character", 1}, {"user_role", 1}, {"mumble_identity", 1}, {"shop_order", 3}, {"schema_migration", 1}} {
		var got int64
		if err := db.Table(check.table).Count(&got).Error; err != nil || got != check.want {
			t.Fatalf("%s count=%d want=%d err=%v", check.table, got, check.want, err)
		}
	}
	for _, table := range contactCleanupCharacterTables {
		var got int64
		db.Table(table).Count(&got)
		if got != 1 {
			t.Fatalf("%s count=%d", table, got)
		}
	}
}

func TestRequiredUserContactCleanup(t *testing.T) {
	testContactCleanup(t, contactMigrationTestDB(t, ""))
}

func TestRequiredUserContactRollback(t *testing.T) {
	db := contactMigrationTestDB(t, "")
	for _, user := range []model.User{{BaseModel: model.BaseModel{ID: 1}}, {DiscordID: "duplicate"}, {DiscordID: "duplicate"}} {
		if err := db.Create(&user).Error; err != nil {
			t.Fatal(err)
		}
	}
	if err := migrateRequiredUserContact(db); err == nil {
		t.Fatal("expected duplicate index failure")
	}
	var count int64
	db.Unscoped().Model(&model.User{}).Count(&count)
	if count != 3 {
		t.Fatal("cleanup survived failed migration")
	}
}

func TestRequiredUserContactPostgres(t *testing.T) {
	dsn := os.Getenv("AMIYA_CONTACT_TEST_DSN")
	if strings.TrimSpace(dsn) == "" {
		t.Skip("set AMIYA_CONTACT_TEST_DSN for isolated PostgreSQL schema integration")
	}
	db := contactMigrationTestDB(t, dsn)
	testContactCleanup(t, db)
	for _, value := range []string{"", " \t\r\n", "\u3000"} {
		if err := db.Create(&model.User{QQ: value}).Error; err == nil {
			t.Fatalf("contactless insert accepted: %q", value)
		}
	}
	if err := db.Model(&model.User{}).Where("id = ?", 1).Update("qq", "").Error; err == nil {
		t.Fatal("last contact removal accepted")
	}
	if err := db.Create(&model.User{DiscordID: "discord-only"}).Error; err == nil {
		t.Fatal("duplicate Discord accepted")
	}
}

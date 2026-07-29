package repository

import (
	"amiya-eden/global"
	"amiya-eden/internal/model"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func TestBuildFuelOfficerUsersByCorporationsQueryQuotesUserTable(t *testing.T) {
	db := newDryRunPostgresDB(t)

	sql := db.ToSQL(func(tx *gorm.DB) *gorm.DB {
		return buildFuelOfficerUsersByCorporationsQuery(tx, []int64{9001}).Scan(&[]FuelOfficerUserOption{})
	})

	if !strings.Contains(sql, `FROM "user" AS u`) {
		t.Fatalf("expected quoted user table in fuel officer query, got SQL: %s", sql)
	}
	if strings.Contains(sql, `FROM user AS u`) {
		t.Fatalf("expected fuel officer query to avoid bare user table name, got SQL: %s", sql)
	}
	if !strings.Contains(sql, `JOIN user_role AS ur ON ur.user_id = u.id`) {
		t.Fatalf("expected user_role join in fuel officer query, got SQL: %s", sql)
	}
	if !strings.Contains(sql, `JOIN eve_character AS ec ON ec.user_id = u.id`) {
		t.Fatalf("expected eve_character join in fuel officer query, got SQL: %s", sql)
	}
	if !strings.Contains(sql, `COALESCE(NULLIF(TRIM(u.nickname), ''), MIN(ec.character_name), 'User-' || u.id) AS display_name`) {
		t.Fatalf("expected display_name fallback expression in fuel officer query, got SQL: %s", sql)
	}
}

func newCorporationStructureTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := fmt.Sprintf("file:corporation_structure_test_%d?mode=memory&cache=shared", time.Now().UnixNano())
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&model.CorpStructureInfo{}); err != nil {
		t.Fatalf("auto migrate: %v", err)
	}
	return db
}

func TestCorporationStructureRepository_ListAllCorpStructures(t *testing.T) {
	db := newCorporationStructureTestDB(t)
	oldDB := global.DB
	global.DB = db
	t.Cleanup(func() { global.DB = oldDB })

	rows := []model.CorpStructureInfo{
		{CorporationID: 2, StructureID: 22, Name: "Second"},
		{CorporationID: 1, StructureID: 11, Name: "First"},
	}
	if err := db.Create(&rows).Error; err != nil {
		t.Fatalf("seed structures: %v", err)
	}

	got, err := NewCorporationStructureRepository().ListAllCorpStructures()
	if err != nil {
		t.Fatalf("ListAllCorpStructures() error = %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("ListAllCorpStructures() returned %d rows, want 2", len(got))
	}
	if got[0].CorporationID != 1 || got[0].StructureID != 11 {
		t.Fatalf("ListAllCorpStructures() first row = %+v, want corporation 1 structure 11", got[0])
	}
}

package repository

import (
	"strings"
	"testing"

	"gorm.io/gorm"
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

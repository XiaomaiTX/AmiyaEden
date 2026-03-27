package repository

import (
	"strings"
	"testing"

	"gorm.io/gorm"
)

func TestGetByIDForUpdateTxUsesRowLock(t *testing.T) {
	db := newDryRunPostgresDB(t)

	sql := db.ToSQL(func(tx *gorm.DB) *gorm.DB {
		return buildGetByIDForUpdateQuery(tx, 42)
	})

	if !strings.Contains(sql, `FOR UPDATE`) {
		t.Fatalf("expected row lock query to use FOR UPDATE, got SQL: %s", sql)
	}
	if !strings.Contains(sql, `FROM "user"`) {
		t.Fatalf("expected user table query, got SQL: %s", sql)
	}
	if !strings.Contains(sql, `"user"."id" = 42`) && !strings.Contains(sql, `"id" = 42`) {
		t.Fatalf("expected user id predicate, got SQL: %s", sql)
	}
}

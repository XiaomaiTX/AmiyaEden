package repository

import (
	"amiya-eden/internal/model"
	"fmt"
	"strings"
	"testing"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func TestBuildGetOrCreateWalletForUpdateQueryUsesRowLock(t *testing.T) {
	db := newDryRunPostgresDB(t)

	query := buildGetOrCreateWalletForUpdateQuery(db, 42).
		Session(&gorm.Session{DryRun: true}).
		First(&model.SystemWallet{})
	sql := query.Statement.SQL.String()

	if !strings.Contains(sql, `FROM "system_wallet"`) {
		t.Fatalf("expected system_wallet select, got SQL: %s", sql)
	}
	if !strings.Contains(sql, `WHERE user_id =`) && !strings.Contains(sql, `WHERE "user_id" =`) {
		t.Fatalf("expected user-scoped filter, got SQL: %s", sql)
	}
	if len(query.Statement.Vars) == 0 || fmt.Sprint(query.Statement.Vars[0]) != "42" {
		t.Fatalf("expected first bound variable to be user ID 42, got vars: %#v", query.Statement.Vars)
	}
	if !strings.Contains(sql, `FOR UPDATE`) {
		t.Fatalf("expected row lock query to use FOR UPDATE, got SQL: %s", sql)
	}
	lockingClause, ok := query.Statement.Clauses["FOR"]
	if !ok {
		t.Fatalf("expected FOR locking clause to be present, got clauses: %#v", query.Statement.Clauses)
	}
	lockingExpr, ok := lockingClause.Expression.(clause.Locking)
	if !ok {
		t.Fatalf("expected FOR clause to use clause.Locking, got %T", lockingClause.Expression)
	}
	if lockingExpr.Strength != "UPDATE" {
		t.Fatalf("expected UPDATE row lock strength, got %q", lockingExpr.Strength)
	}
	if !strings.Contains(sql, `ORDER BY`) {
		t.Fatalf("expected first query ordering, got SQL: %s", sql)
	}
}

func TestSysWalletCountTransactionsByUserRefTypeInRangeUsesTimeBounds(t *testing.T) {
	db := newDryRunPostgresDB(t)
	startAt := time.Date(2026, time.March, 1, 0, 0, 0, 0, time.UTC)
	endAt := startAt.AddDate(0, 1, 0)

	sql := db.ToSQL(func(tx *gorm.DB) *gorm.DB {
		return tx.Model(&struct{}{}).
			Table("wallet_transaction").
			Where("user_id = ? AND ref_type = ? AND created_at >= ? AND created_at < ?", 42, "pap_fc_salary", startAt, endAt).
			Count(new(int64))
	})

	if !strings.Contains(sql, `ref_type = 'pap_fc_salary'`) {
		t.Fatalf("expected ref_type filter in SQL, got %s", sql)
	}
	if !strings.Contains(sql, `created_at >=`) || !strings.Contains(sql, `created_at <`) {
		t.Fatalf("expected month bounds in SQL, got %s", sql)
	}
}

func TestSysWalletTransactionLookupByUserRefTypeRefIDUsesAllFilters(t *testing.T) {
	db := newDryRunPostgresDB(t)
	startAt := time.Date(2026, time.March, 1, 0, 0, 0, 0, time.UTC)
	endAt := startAt.AddDate(0, 1, 0)

	sql := db.ToSQL(func(tx *gorm.DB) *gorm.DB {
		return tx.Where(
			"user_id = ? AND ref_type = ? AND ref_id = ? AND created_at >= ? AND created_at < ?",
			42, "pap_fc_salary", "fleet-1", startAt, endAt,
		).
			First(&model.WalletTransaction{})
	})

	if !strings.Contains(sql, `ref_type = 'pap_fc_salary'`) {
		t.Fatalf("expected ref_type filter in SQL, got %s", sql)
	}
	if !strings.Contains(sql, `ref_id = 'fleet-1'`) {
		t.Fatalf("expected ref_id filter in SQL, got %s", sql)
	}
	if !strings.Contains(sql, `created_at >=`) || !strings.Contains(sql, `created_at <`) {
		t.Fatalf("expected month bounds in SQL, got %s", sql)
	}
}

func TestListTransactionsWithCharacterAppliesUserKeywordAcrossNicknameAndCharacterName(t *testing.T) {
	db := newDryRunPostgresDB(t)
	filter := WalletTransactionFilter{UserKeyword: "bee"}

	sql := db.ToSQL(func(tx *gorm.DB) *gorm.DB {
		return applyWalletTransactionUserFilter(
			tx.Table("wallet_transaction wt").
				Joins(`LEFT JOIN "user" u ON wt.user_id = u.id`).
				Joins("LEFT JOIN eve_character ec ON u.primary_character_id = ec.character_id"),
			"wt.user_id",
			"wt.ref_type",
			filter,
		).Find(&[]struct{}{})
	})

	if !strings.Contains(sql, `LEFT JOIN "user" u ON wt.user_id = u.id`) {
		t.Fatalf("expected wallet transaction query to join user table, got SQL: %s", sql)
	}
	if !strings.Contains(sql, `LEFT JOIN eve_character ec ON u.primary_character_id = ec.character_id`) {
		t.Fatalf("expected wallet transaction query to join primary character, got SQL: %s", sql)
	}
	if !strings.Contains(sql, `LOWER(u.nickname) LIKE`) || !strings.Contains(sql, `LOWER(ec.character_name) LIKE`) {
		t.Fatalf("expected nickname and character name keyword search, got SQL: %s", sql)
	}
}

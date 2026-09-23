package repository

import (
	"amiya-eden/internal/model"
	"strings"
	"testing"
	"time"

	"gorm.io/gorm"
)

func TestGetWalletJournalsAppliesRefTypeFilterWhenProvided(t *testing.T) {
	db := newDryRunPostgresDB(t)
	repo := &EveWalletRepository{}

	sql := db.ToSQL(func(tx *gorm.DB) *gorm.DB {
		return repo.getWalletJournalsQuery(tx, 90000001, []string{"bounty_prizes", "ess_escrow_transfer"}).
			Order("date DESC").
			Offset(0).
			Limit(20).
			Find(&[]model.EVECharacterWalletJournal{})
	})

	if !strings.Contains(sql, `WHERE character_id =`) {
		t.Fatalf("expected character filter, got SQL: %s", sql)
	}
	if !strings.Contains(sql, `ref_type IN (`) {
		t.Fatalf("expected ref_type filter, got SQL: %s", sql)
	}
	if !strings.Contains(sql, `'bounty_prizes'`) || !strings.Contains(sql, `'ess_escrow_transfer'`) {
		t.Fatalf("expected caller supplied ref types, got SQL: %s", sql)
	}
}

func TestListWalletJournalRefTypesUsesDistinctRefTypeQuery(t *testing.T) {
	db := newDryRunPostgresDB(t)
	repo := &EveWalletRepository{}

	sql := db.ToSQL(func(tx *gorm.DB) *gorm.DB {
		return repo.getWalletJournalRefTypesQuery(tx, 90000001).
			Pluck("ref_type", &[]string{})
	})

	if !strings.Contains(sql, `SELECT DISTINCT`) {
		t.Fatalf("expected distinct query, got SQL: %s", sql)
	}
	if !strings.Contains(sql, `WHERE character_id =`) {
		t.Fatalf("expected character filter, got SQL: %s", sql)
	}
	if !strings.Contains(sql, `ORDER BY ref_type ASC`) {
		t.Fatalf("expected ref_type ordering, got SQL: %s", sql)
	}
}

func TestListWalletJournalFlowEntriesQueryUsesHalfOpenRangeAndAscendingOrder(t *testing.T) {
	db := newDryRunPostgresDB(t)
	repo := &EveWalletRepository{}

	from := time.Date(2026, time.September, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2026, time.September, 2, 0, 0, 0, 0, time.UTC)
	sql := db.ToSQL(func(tx *gorm.DB) *gorm.DB {
		return repo.listWalletJournalFlowEntriesQuery(tx, 90000001, from, to).
			Find(&[]WalletJournalFlowEntry{})
	})

	if !strings.Contains(sql, `WHERE character_id =`) {
		t.Fatalf("expected character filter, got SQL: %s", sql)
	}
	if !strings.Contains(sql, `date >=`) || !strings.Contains(sql, `date <`) {
		t.Fatalf("expected half-open date range [from, to), got SQL: %s", sql)
	}
	if !strings.Contains(sql, `date, amount, tax, balance, ref_type`) {
		t.Fatalf("expected slim projection for aggregation, got SQL: %s", sql)
	}
	if !strings.Contains(sql, `ORDER BY date ASC`) || !strings.Contains(sql, `id ASC`) {
		t.Fatalf("expected ascending date/id ordering, got SQL: %s", sql)
	}
}

func TestGetWalletJournalLastBalanceBeforeQueryTakesLatestEntryBeforeCutoff(t *testing.T) {
	db := newDryRunPostgresDB(t)
	repo := &EveWalletRepository{}

	before := time.Date(2026, time.September, 1, 0, 0, 0, 0, time.UTC)
	sql := db.ToSQL(func(tx *gorm.DB) *gorm.DB {
		return repo.getWalletJournalLastBalanceBeforeQuery(tx, 90000001, before).
			Pluck("balance", &[]float64{})
	})

	if !strings.Contains(sql, `WHERE character_id =`) {
		t.Fatalf("expected character filter, got SQL: %s", sql)
	}
	if !strings.Contains(sql, `date <`) {
		t.Fatalf("expected exclusive upper bound, got SQL: %s", sql)
	}
	if !strings.Contains(sql, `ORDER BY date DESC`) || !strings.Contains(sql, `id DESC`) {
		t.Fatalf("expected descending date/id ordering, got SQL: %s", sql)
	}
	if !strings.Contains(sql, `LIMIT 1`) {
		t.Fatalf("expected single row limit, got SQL: %s", sql)
	}
}

func TestGetWalletJournalDateBoundsQueriesTakeSingleOrderedRow(t *testing.T) {
	db := newDryRunPostgresDB(t)
	repo := &EveWalletRepository{}

	firstSQL := db.ToSQL(func(tx *gorm.DB) *gorm.DB {
		return repo.getWalletJournalFirstDateQuery(tx, 90000001).
			Pluck("date", &[]time.Time{})
	})
	if !strings.Contains(firstSQL, `WHERE character_id =`) {
		t.Fatalf("expected character filter, got SQL: %s", firstSQL)
	}
	if !strings.Contains(firstSQL, `ORDER BY date ASC`) || !strings.Contains(firstSQL, `LIMIT 1`) {
		t.Fatalf("expected a single earliest row, got SQL: %s", firstSQL)
	}

	lastSQL := db.ToSQL(func(tx *gorm.DB) *gorm.DB {
		return repo.getWalletJournalLastDateQuery(tx, 90000001).
			Pluck("date", &[]time.Time{})
	})
	if !strings.Contains(lastSQL, `WHERE character_id =`) {
		t.Fatalf("expected character filter, got SQL: %s", lastSQL)
	}
	if !strings.Contains(lastSQL, `ORDER BY date DESC`) || !strings.Contains(lastSQL, `LIMIT 1`) {
		t.Fatalf("expected a single latest row, got SQL: %s", lastSQL)
	}
}

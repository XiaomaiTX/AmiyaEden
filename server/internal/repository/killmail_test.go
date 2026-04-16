package repository

import (
	"amiya-eden/internal/model"
	"strings"
	"testing"
	"time"

	"gorm.io/gorm"
)

func TestBuildVictimKillmailListQueryAppliesSinceOrderAndLimit(t *testing.T) {
	db := newDryRunPostgresDB(t)
	since := time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)

	sql := db.ToSQL(func(tx *gorm.DB) *gorm.DB {
		var list []model.EveKillmailList
		return buildVictimKillmailListQuery(tx, VictimKillmailListFilter{
			CharacterIDs: []int64{100, 200},
			Since:        &since,
			Limit:        50,
		}).Find(&list)
	})

	if !strings.Contains(sql, `FROM "eve_killmail_list"`) {
		t.Fatalf("expected eve_killmail_list table, got SQL: %s", sql)
	}
	if !strings.Contains(sql, `character_id IN (`) {
		t.Fatalf("expected character_id IN filter, got SQL: %s", sql)
	}
	if !strings.Contains(sql, `kill_mail_time >= `) {
		t.Fatalf("expected kill_mail_time >= filter, got SQL: %s", sql)
	}
	if !strings.Contains(sql, `ORDER BY kill_mail_time DESC`) {
		t.Fatalf("expected descending time order, got SQL: %s", sql)
	}
	if !strings.Contains(sql, `LIMIT `) {
		t.Fatalf("expected LIMIT clause, got SQL: %s", sql)
	}
}

func TestBuildVictimKillmailListQueryAppliesTimeRangeAndExcludeSubmitted(t *testing.T) {
	db := newDryRunPostgresDB(t)
	start := time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(2026, 3, 2, 0, 0, 0, 0, time.UTC)
	userID := uint(88)

	sql := db.ToSQL(func(tx *gorm.DB) *gorm.DB {
		var list []model.EveKillmailList
		return buildVictimKillmailListQuery(tx, VictimKillmailListFilter{
			CharacterIDs:             []int64{100},
			StartAt:                  &start,
			EndAt:                    &end,
			ExcludeSubmittedByUserID: &userID,
		}).Find(&list)
	})

	if !strings.Contains(sql, `FROM "eve_killmail_list"`) {
		t.Fatalf("expected eve_killmail_list table, got SQL: %s", sql)
	}
	if !strings.Contains(sql, `character_id IN (`) {
		t.Fatalf("expected character_id IN filter, got SQL: %s", sql)
	}
	if !strings.Contains(sql, `BETWEEN`) {
		t.Fatalf("expected BETWEEN clause for time range, got SQL: %s", sql)
	}
	if !strings.Contains(sql, `NOT EXISTS`) {
		t.Fatalf("expected NOT EXISTS clause for submitted exclusion, got SQL: %s", sql)
	}
}

func TestListKillmailItemsQueryFiltersByKillmailID(t *testing.T) {
	db := newDryRunPostgresDB(t)

	sql := db.ToSQL(func(tx *gorm.DB) *gorm.DB {
		var list []model.EveKillmailItem
		return tx.Where("kill_mail_id = ?", int64(99999)).Find(&list)
	})

	if !strings.Contains(sql, `FROM "eve_killmail_item"`) {
		t.Fatalf("expected eve_killmail_item table, got SQL: %s", sql)
	}
	if !strings.Contains(sql, `kill_mail_id = `) {
		t.Fatalf("expected kill_mail_id filter, got SQL: %s", sql)
	}
}

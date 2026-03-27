package repository

import (
	"amiya-eden/internal/model"
	"strings"
	"testing"
	"time"

	"gorm.io/gorm"
)

func TestBuildUnattributedPlayerJournalQueryUsesJoinAndCallerSuppliedFilters(t *testing.T) {
	db := newDryRunPostgresDB(t)
	start := time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)

	sql := db.ToSQL(func(tx *gorm.DB) *gorm.DB {
		return buildUnattributedPlayerJournalQuery(tx, 100, start, []string{"bounty_prizes", "ess_escrow_transfer"}, 250).
			Find(&[]model.EVECharacterWalletJournal{})
	})

	if !strings.Contains(sql, `FROM "eve_character_wallet_journal"`) {
		t.Fatalf("expected wallet journal source table, got SQL: %s", sql)
	}
	if !strings.Contains(sql, `LEFT JOIN captain_bounty_attribution`) {
		t.Fatalf("expected left join to captain_bounty_attribution, got SQL: %s", sql)
	}
	if !strings.Contains(sql, `captain_bounty_attribution.wallet_journal_id IS NULL`) {
		t.Fatalf("expected unattributed join filter, got SQL: %s", sql)
	}
	if !strings.Contains(sql, `id >`) {
		t.Fatalf("expected last wallet journal id filter, got SQL: %s", sql)
	}
	if !strings.Contains(sql, `date >=`) {
		t.Fatalf("expected lookback start filter, got SQL: %s", sql)
	}
	if !strings.Contains(sql, `ref_type IN (`) {
		t.Fatalf("expected caller-supplied ref_type filter, got SQL: %s", sql)
	}
	if !strings.Contains(sql, `ORDER BY eve_character_wallet_journal.id ASC`) {
		t.Fatalf("expected stable ascending id order, got SQL: %s", sql)
	}
	if !strings.Contains(sql, `LIMIT 250`) {
		t.Fatalf("expected explicit processing limit, got SQL: %s", sql)
	}
}

func TestBuildCaptainCandidateJournalQueryUsesExplicitWindowAndRefTypes(t *testing.T) {
	db := newDryRunPostgresDB(t)
	start := time.Date(2026, 3, 27, 19, 45, 0, 0, time.UTC)
	end := time.Date(2026, 3, 27, 20, 15, 0, 0, time.UTC)

	sql := db.ToSQL(func(tx *gorm.DB) *gorm.DB {
		return buildCaptainCandidateJournalQuery(tx, 90000001, 30000142, start, end, []string{"ess_escrow_transfer"}).
			Find(&[]model.EVECharacterWalletJournal{})
	})

	if !strings.Contains(sql, `character_id =`) {
		t.Fatalf("expected character_id filter, got SQL: %s", sql)
	}
	if !strings.Contains(sql, `context_id =`) {
		t.Fatalf("expected context_id filter, got SQL: %s", sql)
	}
	if !strings.Contains(sql, `date >=`) || !strings.Contains(sql, `date <=`) {
		t.Fatalf("expected explicit time window bounds, got SQL: %s", sql)
	}
	if !strings.Contains(sql, `ref_type IN (`) {
		t.Fatalf("expected caller-supplied candidate ref_type filter, got SQL: %s", sql)
	}
	if strings.Contains(sql, `'bounty_prizes'`) {
		t.Fatalf("did not expect repository to hardcode bounty_prizes, got SQL: %s", sql)
	}
}

func TestBuildCaptainAttributionAggregateQueryUsesCallerSuppliedSupportedRefTypes(t *testing.T) {
	db := newDryRunPostgresDB(t)

	sql := db.ToSQL(func(tx *gorm.DB) *gorm.DB {
		return buildCaptainAttributionAggregateQuery(
			tx.Model(&model.CaptainBountyAttribution{}),
			[]string{"ess_escrow_transfer"},
			true,
		).Where("captain_user_id = ?", 42).Scan(&struct{}{})
	})

	if !strings.Contains(sql, `ref_type IN (`) {
		t.Fatalf("expected aggregate query to filter by caller-supplied ref types, got SQL: %s", sql)
	}
	if !strings.Contains(sql, `'ess_escrow_transfer'`) {
		t.Fatalf("expected aggregate query to include caller-supplied ref type, got SQL: %s", sql)
	}
	if strings.Contains(sql, `'bounty_prizes'`) {
		t.Fatalf("did not expect aggregate query to hardcode bounty_prizes, got SQL: %s", sql)
	}
	if !strings.Contains(sql, `COUNT(*) AS record_count`) {
		t.Fatalf("expected aggregate query to project record_count when requested, got SQL: %s", sql)
	}
}

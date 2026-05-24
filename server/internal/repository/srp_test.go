package repository

import (
	"amiya-eden/internal/model"
	"strings"
	"testing"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func TestBuildApprovedUnpaidBatchPayoutApplicationsQueryUsesUserScopedLocking(t *testing.T) {
	db := newDryRunPostgresDB(t)

	tx := buildApprovedUnpaidBatchPayoutApplicationsQuery(db, 42).
		Session(&gorm.Session{DryRun: true}).
		Find(&[]model.SrpApplication{})
	sql := tx.Statement.SQL.String()

	if !strings.Contains(sql, `FROM "srp_application"`) {
		t.Fatalf("expected srp_application select, got SQL: %s", sql)
	}
	if !strings.Contains(sql, `user_id = $1`) {
		t.Fatalf("expected user-scoped filter, got SQL: %s", sql)
	}
	if !strings.Contains(sql, `payout_status = $2`) || !strings.Contains(sql, `review_status = $3`) {
		t.Fatalf("expected payout/review status filters, got SQL: %s", sql)
	}
	if !strings.Contains(sql, `FOR UPDATE`) {
		t.Fatalf("expected row locking for batch payout selection, got SQL: %s", sql)
	}
}

func TestBuildPendingBadgeSrpCountQueryUsesPendingApprovalStatuses(t *testing.T) {
	db := newDryRunPostgresDB(t)

	sql := db.ToSQL(func(tx *gorm.DB) *gorm.DB {
		return buildPendingBadgeSrpCountQuery(tx).Count(new(int64))
	})

	if !strings.Contains(sql, `FROM "srp_application"`) {
		t.Fatalf("expected srp_application count query, got SQL: %s", sql)
	}
	if !strings.Contains(sql, `review_status IN (`) {
		t.Fatalf("expected pending review scope on badge count query, got SQL: %s", sql)
	}
	if !strings.Contains(sql, `payout_status =`) {
		t.Fatalf("expected unpaid scope on badge count query, got SQL: %s", sql)
	}
}

func TestBuildBatchPayoutApplicationsUpdateTargetsSelectedApplicationIDs(t *testing.T) {
	db := newDryRunPostgresDB(t)
	paidAt := time.Unix(1_700_000_000, 0).UTC()

	sql := db.ToSQL(func(tx *gorm.DB) *gorm.DB {
		return buildBatchPayoutApplicationsUpdateQuery(tx, []uint{7, 9}, 99, paidAt)
	})

	if !strings.Contains(sql, `UPDATE "srp_application"`) {
		t.Fatalf("expected srp_application update, got SQL: %s", sql)
	}
	if !strings.Contains(sql, `id IN (`) {
		t.Fatalf("expected ID-scoped update, got SQL: %s", sql)
	}
	if strings.Contains(sql, `user_id =`) {
		t.Fatalf("expected update to avoid broad user-scoped predicate, got SQL: %s", sql)
	}
	if !strings.Contains(sql, `payout_status =`) || !strings.Contains(sql, `review_status =`) {
		t.Fatalf("expected payout/review status guard on update, got SQL: %s", sql)
	}
}

func TestSummarizeBatchPayoutApplicationsUsesExactSelectedRows(t *testing.T) {
	summary, ids := summarizeBatchPayoutApplications(42, []model.SrpApplication{
		{ID: 7, FinalAmount: 12.5},
		{ID: 9, FinalAmount: 30},
	})

	if summary.UserID != 42 {
		t.Fatalf("expected user ID 42, got %d", summary.UserID)
	}
	if summary.ApplicationCount != 2 {
		t.Fatalf("expected 2 applications, got %d", summary.ApplicationCount)
	}
	if summary.TotalAmount != 42.5 {
		t.Fatalf("expected total amount 42.5, got %v", summary.TotalAmount)
	}
	if len(ids) != 2 || ids[0] != 7 || ids[1] != 9 {
		t.Fatalf("expected selected IDs [7 9], got %v", ids)
	}
}

func TestBuildSrpApplicationListQueryAppliesPendingTabScope(t *testing.T) {
	db := newDryRunPostgresDB(t)

	sql := db.ToSQL(func(tx *gorm.DB) *gorm.DB {
		return buildSrpApplicationListQuery(tx, SrpApplicationFilter{Tab: SrpTabPending}).
			Find(&[]model.SrpApplication{})
	})

	if !strings.Contains(sql, `FROM "srp_application"`) {
		t.Fatalf("expected srp_application select, got SQL: %s", sql)
	}
	if !strings.Contains(sql, `review_status IN (`) {
		t.Fatalf("expected pending tab review scope, got SQL: %s", sql)
	}
	if !strings.Contains(sql, `payout_status =`) {
		t.Fatalf("expected pending tab payout scope, got SQL: %s", sql)
	}
}

func TestBuildSrpApplicationListQueryAppliesHistoryTabScope(t *testing.T) {
	db := newDryRunPostgresDB(t)

	sql := db.ToSQL(func(tx *gorm.DB) *gorm.DB {
		return buildSrpApplicationListQuery(tx, SrpApplicationFilter{Tab: SrpTabHistory}).
			Find(&[]model.SrpApplication{})
	})

	if !strings.Contains(sql, `FROM "srp_application"`) {
		t.Fatalf("expected srp_application select, got SQL: %s", sql)
	}
	if !strings.Contains(sql, `payout_status =`) || !strings.Contains(sql, `OR review_status =`) {
		t.Fatalf("expected history tab to include paid or rejected scope, got SQL: %s", sql)
	}
}

func TestBuildSrpApplicationListQueryAppliesCharacterAndNicknameKeywordFilter(t *testing.T) {
	db := newDryRunPostgresDB(t)

	sql := db.ToSQL(func(tx *gorm.DB) *gorm.DB {
		return buildSrpApplicationListQuery(tx, SrpApplicationFilter{Keyword: "bee"}).Find(&[]model.SrpApplication{})
	})

	if !strings.Contains(sql, `FROM "user" AS applicant_user`) {
		t.Fatalf("expected keyword filter to query current applicant nickname, got SQL: %s", sql)
	}
	if !strings.Contains(sql, `LOWER(applicant_user.nickname) LIKE`) {
		t.Fatalf("expected applicant nickname keyword predicate, got SQL: %s", sql)
	}
	if !strings.Contains(sql, `LOWER(character_name) LIKE`) {
		t.Fatalf("expected application character keyword predicate, got SQL: %s", sql)
	}
}

func TestBuildSrpApplicationListQueryAppliesExtendedFilters(t *testing.T) {
	db := newDryRunPostgresDB(t)
	fleetID := "fleet-1"
	corporationID := int64(1234)
	shipTypeID := int64(587)
	solarSystemID := int64(30000142)
	hasRecommendedMatch := true

	sql := db.ToSQL(func(tx *gorm.DB) *gorm.DB {
		return buildSrpApplicationListQuery(tx, SrpApplicationFilter{
			FleetID:             &fleetID,
			CorporationID:       &corporationID,
			ShipTypeID:          &shipTypeID,
			SolarSystemID:       &solarSystemID,
			HasRecommendedMatch: &hasRecommendedMatch,
		}).Find(&[]model.SrpApplication{})
	})

	if !strings.Contains(sql, `fleet_id =`) {
		t.Fatalf("expected fleet filter, got SQL: %s", sql)
	}
	if !strings.Contains(sql, `corporation_id =`) {
		t.Fatalf("expected corporation filter, got SQL: %s", sql)
	}
	if !strings.Contains(sql, `ship_type_id =`) {
		t.Fatalf("expected ship type filter, got SQL: %s", sql)
	}
	if !strings.Contains(sql, `solar_system_id =`) {
		t.Fatalf("expected solar system filter, got SQL: %s", sql)
	}
	if !strings.Contains(sql, `recommended_amount > 0`) {
		t.Fatalf("expected has recommended match predicate, got SQL: %s", sql)
	}
}

func TestBuildSrpApplicationListQueryAppliesUnmatchedRecommendedFilter(t *testing.T) {
	db := newDryRunPostgresDB(t)
	hasRecommendedMatch := false

	sql := db.ToSQL(func(tx *gorm.DB) *gorm.DB {
		return buildSrpApplicationListQuery(tx, SrpApplicationFilter{
			HasRecommendedMatch: &hasRecommendedMatch,
		}).Find(&[]model.SrpApplication{})
	})

	if !strings.Contains(sql, `recommended_amount <= 0`) {
		t.Fatalf("expected unmatched recommended predicate, got SQL: %s", sql)
	}
}

func TestBuildSrpApplicationOrderClauseUsesWhitelistAndDefault(t *testing.T) {
	assertOrder := func(got clause.OrderByColumn, wantColumn string, wantDesc bool) {
		t.Helper()
		if got.Column.Name != wantColumn || got.Desc != wantDesc {
			t.Fatalf("expected order %s desc=%t, got column=%s desc=%t", wantColumn, wantDesc, got.Column.Name, got.Desc)
		}
	}

	assertOrder(buildSrpApplicationOrderClause(SrpApplicationFilter{
		SortBy:    "final_amount",
		SortOrder: "asc",
	}), "final_amount", false)

	assertOrder(buildSrpApplicationOrderClause(SrpApplicationFilter{
		SortBy:    "created_at;drop table srp_application",
		SortOrder: "asc",
	}), "created_at", false)

	assertOrder(buildSrpApplicationOrderClause(SrpApplicationFilter{
		SortBy:    "ship_type_id",
		SortOrder: "invalid",
	}), "ship_type_id", true)
}

func TestGetApplicationByKillmailAndCharacterQueryUsesCompositeKey(t *testing.T) {
	db := newDryRunPostgresDB(t)

	sql := db.ToSQL(func(tx *gorm.DB) *gorm.DB {
		var app model.SrpApplication
		return tx.Where("killmail_id = ? AND character_id = ?", int64(880001), int64(90001001)).First(&app)
	})

	if !strings.Contains(sql, `FROM "srp_application"`) {
		t.Fatalf("expected srp_application table, got SQL: %s", sql)
	}
	if !strings.Contains(sql, `killmail_id =`) || !strings.Contains(sql, `character_id =`) {
		t.Fatalf("expected composite killmail+character filter, got SQL: %s", sql)
	}
}

func TestBuildSrpFleetOptionsQueryUsesApplicationAggregation(t *testing.T) {
	db := newDryRunPostgresDB(t)

	sql := db.ToSQL(func(tx *gorm.DB) *gorm.DB {
		return buildSrpFleetOptionsQuery(tx).Find(&[]SrpFleetOptionRow{})
	})

	if !strings.Contains(sql, `FROM srp_application AS app`) {
		t.Fatalf("expected srp_application source, got SQL: %s", sql)
	}
	if !strings.Contains(sql, `LEFT JOIN fleet AS f ON f.id = app.fleet_id`) {
		t.Fatalf("expected fleet left join for metadata enrichment, got SQL: %s", sql)
	}
	if !strings.Contains(sql, `app.fleet_id IS NOT NULL`) || !strings.Contains(sql, `app.fleet_id <> ''`) {
		t.Fatalf("expected non-empty fleet_id filter, got SQL: %s", sql)
	}
	if !strings.Contains(sql, `GROUP BY app.fleet_id, f.title, f.fc_character_name`) {
		t.Fatalf("expected fleet-based grouping, got SQL: %s", sql)
	}
	if !strings.Contains(sql, `ORDER BY MAX(app.created_at) DESC`) {
		t.Fatalf("expected recent-application ordering, got SQL: %s", sql)
	}
}

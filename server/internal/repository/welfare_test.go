package repository

import (
	"amiya-eden/internal/model"
	"strings"
	"testing"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func TestBuildGetApplicationByIDForUpdateQueryUsesRowLock(t *testing.T) {
	db := newDryRunPostgresDB(t)

	query := buildGetApplicationByIDForUpdateQuery(db, 42).
		Session(&gorm.Session{DryRun: true}).
		First(&model.WelfareApplication{})
	sql := query.Statement.SQL.String()

	if !strings.Contains(sql, `FROM "welfare_application"`) {
		t.Fatalf("expected welfare_application select, got SQL: %s", sql)
	}
	if !strings.Contains(sql, `WHERE welfare_application.id =`) && !strings.Contains(sql, `WHERE "welfare_application"."id" =`) && !strings.Contains(sql, `WHERE id =`) {
		t.Fatalf("expected application id filter, got SQL: %s", sql)
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
}

func TestBuildApplicationsByUserIDQueryAppliesUserStatusAndPagination(t *testing.T) {
	db := newDryRunPostgresDB(t)

	sql := db.ToSQL(func(tx *gorm.DB) *gorm.DB {
		return buildApplicationsByUserIDQuery(tx.Model(&model.WelfareApplication{}), 42, "delivered").
			Order("id DESC").
			Offset(20).
			Limit(10).
			Find(&[]model.WelfareApplication{})
	})

	if !strings.Contains(sql, `FROM "welfare_application"`) {
		t.Fatalf("expected welfare_application select, got SQL: %s", sql)
	}
	if !strings.Contains(sql, `user_id =`) {
		t.Fatalf("expected user-scoped filter, got SQL: %s", sql)
	}
	if !strings.Contains(sql, `status =`) {
		t.Fatalf("expected status filter, got SQL: %s", sql)
	}
	if !strings.Contains(sql, `ORDER BY id DESC`) {
		t.Fatalf("expected newest-first ordering, got SQL: %s", sql)
	}
	if !strings.Contains(sql, `LIMIT 10`) {
		t.Fatalf("expected page size limit, got SQL: %s", sql)
	}
	if !strings.Contains(sql, `OFFSET 20`) {
		t.Fatalf("expected page offset, got SQL: %s", sql)
	}
}

func TestListApplicationsPaginatedAppliesApplicantIdentityKeywordFilter(t *testing.T) {
	db := newDryRunPostgresDB(t)

	sql := db.ToSQL(func(tx *gorm.DB) *gorm.DB {
		query := tx.Model(&model.WelfareApplication{}).
			Joins(`LEFT JOIN "user" AS applicant_user ON applicant_user.id = welfare_application.user_id`)
		query = applyKeywordLikeFilter(
			query,
			"bee",
			`LOWER(applicant_user.nickname) LIKE ?`,
			`LOWER(welfare_application.character_name) LIKE ?`,
			`LOWER(welfare_application.qq) LIKE ?`,
		)
		return query.Order("id DESC").Offset(0).Limit(20).Find(&[]model.WelfareApplication{})
	})

	if !strings.Contains(sql, `LEFT JOIN "user" AS applicant_user`) {
		t.Fatalf("expected applicant user join for nickname search, got SQL: %s", sql)
	}
	if !strings.Contains(sql, `LOWER(applicant_user.nickname) LIKE`) {
		t.Fatalf("expected applicant nickname keyword predicate, got SQL: %s", sql)
	}
	if !strings.Contains(sql, `LOWER(welfare_application.character_name) LIKE`) {
		t.Fatalf("expected character keyword predicate, got SQL: %s", sql)
	}
	if !strings.Contains(sql, `LOWER(welfare_application.qq) LIKE`) {
		t.Fatalf("expected QQ keyword predicate, got SQL: %s", sql)
	}
}

func TestListApplicationsPaginatedQualifiesStatusAndOrderColumnsWhenJoined(t *testing.T) {
	db := newDryRunPostgresDB(t)

	sql := db.ToSQL(func(tx *gorm.DB) *gorm.DB {
		query := tx.Model(&model.WelfareApplication{}).
			Where("welfare_application.status IN ?", []string{"delivered", "rejected"}).
			Joins(`LEFT JOIN "user" AS applicant_user ON applicant_user.id = welfare_application.user_id`)
		query = applyKeywordLikeFilter(
			query,
			"bee",
			`LOWER(applicant_user.nickname) LIKE ?`,
			`LOWER(welfare_application.character_name) LIKE ?`,
			`LOWER(welfare_application.qq) LIKE ?`,
		)
		return query.Order("welfare_application.id DESC").Offset(0).Limit(20).Find(&[]model.WelfareApplication{})
	})

	if !strings.Contains(sql, `welfare_application.status IN`) {
		t.Fatalf("expected joined welfare query to qualify status column, got SQL: %s", sql)
	}
	if !strings.Contains(sql, `ORDER BY welfare_application.id DESC`) {
		t.Fatalf("expected joined welfare query to qualify order column, got SQL: %s", sql)
	}
}

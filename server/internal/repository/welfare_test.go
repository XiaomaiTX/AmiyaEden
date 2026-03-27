package repository

import (
	"amiya-eden/internal/model"
	"strings"
	"testing"

	"gorm.io/gorm"
)

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

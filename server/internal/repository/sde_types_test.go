package repository

import (
	"strings"
	"testing"

	"gorm.io/gorm"
)

func TestGetTypesByCategoryIDQueryUsesBooleanPublishedFilter(t *testing.T) {
	db := newDryRunPostgresDB(t)

	sql := db.ToSQL(func(tx *gorm.DB) *gorm.DB {
		var rows []TypeInfo
		return tx.Table(`"invTypes" t`).
			Where(`"c"."categoryID" = ? AND "t"."published" = ?`, 16, true).
			Find(&rows)
	})

	if !strings.Contains(sql, `"t"."published" = true`) {
		t.Fatalf("expected boolean published filter, got SQL: %s", sql)
	}
}

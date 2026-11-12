package bootstrap

import (
	"amiya-eden/internal/model"
	"amiya-eden/internal/testutil"
	"testing"
)

func TestAdmissionTicketRetirementPostgres(t *testing.T) {
	db := testutil.Postgres(t)
	if err := migrateAdmissions(db); err != nil {
		t.Fatal(err)
	}
	var count int64
	db.Model(&model.TicketCategory{}).Count(&count)
	if count != 0 {
		t.Fatal("migration created category")
	}
	for _, name := range []string{"加团申请", "加团", "小号加团", "Other"} {
		if err := db.Create(&model.TicketCategory{Name: name, NameEN: name, Enabled: true}).Error; err != nil {
			t.Fatal(err)
		}
	}
	for range 2 {
		if err := migrateAdmissions(db); err != nil {
			t.Fatal(err)
		}
	}
	var categories []model.TicketCategory
	db.Order("id").Find(&categories)
	if len(categories) != 4 {
		t.Fatal("categories lost")
	}
	for i, category := range categories {
		if category.ID != uint(i+1) || (i < 3 && (!category.RetiredAdmission || category.Enabled)) || (i == 3 && (category.RetiredAdmission || !category.Enabled)) {
			t.Fatalf("migration category altered incorrectly: %+v", category)
		}
	}
}

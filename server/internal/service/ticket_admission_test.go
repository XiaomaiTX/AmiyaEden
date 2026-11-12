package service

import (
	"amiya-eden/internal/model"
	"errors"
	"testing"
)

func TestTicketAdmissionRetirementPreservesHistory(t *testing.T) {
	db := setupTicketServiceTestDB(t)
	svc := NewTicketService()
	for index, name := range []string{"加团申请", "加团", "小号加团", "Unrelated disabled"} {
		category := model.TicketCategory{Name: name, NameEN: name, Enabled: true, RetiredAdmission: index < 3}
		if err := db.Create(&category).Error; err != nil {
			t.Fatal(err)
		}
		if index == 3 {
			db.Model(&category).Update("enabled", false)
		}
		var before int64
		db.Model(&model.TicketStatusHistory{}).Count(&before)
		if _, err := svc.CreateTicket(1, category.ID, "Title", "Description"); !errors.Is(err, errTicketCategoryDisabled) {
			t.Fatalf("retired/disabled accepted %s: %v", name, err)
		}
		var count int64
		db.Model(&model.TicketStatusHistory{}).Count(&count)
		if count != before {
			t.Fatal("initial history written for denied request")
		}
		if index < 3 {
			history := model.Ticket{UserID: 1, CategoryID: category.ID, Title: "Existing admission", Description: "Keep", Status: model.TicketStatusPending}
			if err := db.Create(&history).Error; err != nil {
				t.Fatal(err)
			}
			if _, err := svc.GetMyTicket(1, history.ID); err != nil {
				t.Fatal(err)
			}
			if _, err := svc.AddReplyAsUser(1, history.ID, "Still readable"); err != nil {
				t.Fatal(err)
			}
			if _, err := svc.UpdateStatusAsAdmin(2, history.ID, model.TicketStatusCompleted); err != nil {
				t.Fatal(err)
			}
			if _, err := svc.UpdateCategory(2, category.ID, &model.TicketCategory{Name: "Renamed", NameEN: "Renamed", Enabled: true}); !errors.Is(err, errTicketCategoryDisabled) {
				t.Fatalf("retired category renamed/re-enabled: %v", err)
			}
			if err := svc.CreateCategory(2, &model.TicketCategory{Name: name, NameEN: "Recreate", Enabled: true}); !errors.Is(err, errTicketCategoryDisabled) {
				t.Fatal("old category recreated")
			}
		}
	}
	category := seedTicketCategory(t)
	if _, err := svc.CreateTicket(1, category.ID, "Normal", "Normal"); err != nil {
		t.Fatal(err)
	}
}

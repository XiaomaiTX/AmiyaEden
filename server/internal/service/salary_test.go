package service

import (
	"amiya-eden/global"
	"amiya-eden/internal/model"
	"amiya-eden/internal/repository"
	"fmt"
	"math"
	"strconv"
	"testing"
)

func TestSalarySettingsRoundTripZeroAndIndependentAmounts(t *testing.T) {
	admissionApplicationDB(t, false)
	svc := NewSalaryService()
	initial, err := svc.Settings()
	if err != nil || initial.Recruitment != 10 || initial.WelfareDelivery != 10 || initial.ShopDelivery != 10 || initial.SRPPayout != 10 {
		t.Fatalf("defaults: %+v %v", initial, err)
	}
	want := SalarySettings{Recruitment: 0, WelfareDelivery: 1.25, ShopDelivery: 2.5, SRPPayout: 3.75, FCSalary: 0, FCSalaryMonthlyLimit: 0}
	if _, err = svc.Update(want, 1); err != nil {
		t.Fatal(err)
	}
	got, err := svc.Settings()
	if err != nil || *got != want {
		t.Fatalf("round trip: %+v %v", got, err)
	}
	rates, err := svc.OperationRates()
	if err != nil || rates.Recruitment != 0 || rates.WelfareDelivery != 1.25 || rates.ShopDelivery != 2.5 || rates.SRPPayout != 3.75 {
		t.Fatalf("operation rates: %+v %v", rates, err)
	}
	for _, bad := range []float64{-1, 0.001, math.NaN(), math.Inf(1), 1e13} {
		invalid := want
		invalid.Recruitment = bad
		if _, err = svc.Update(invalid, 1); err == nil {
			t.Fatalf("accepted %v", bad)
		}
	}
	got, err = svc.Settings()
	if err != nil || *got != want {
		t.Fatal("invalid write modified settings")
	}
	var audit model.AuditEvent
	if err = global.DB.Where("action = ?", "salary_settings_update").First(&audit).Error; err != nil {
		t.Fatal(err)
	}
}

func TestAdmissionUsesConfiguredSalaryAndZero(t *testing.T) {
	for _, amount := range []float64{0, 4.25} {
		t.Run(fmt.Sprintf("amount_%g", amount), func(t *testing.T) {
			db := admissionApplicationDB(t, false)
			char := seedAdmissionUser(t, db, 1, 1001, model.RoleGuest)
			seedAdmissionUser(t, db, 2, model.SystemCorporationID, model.RoleAdmin)
			if err := repository.NewSysConfigRepository().Set(model.SysConfigSalaryRecruitment, strconv.FormatFloat(amount, 'f', -1, 64), "salary"); err != nil {
				t.Fatal(err)
			}
			svc := NewAdmissionService()
			app, err := svc.Create(1, newAdmissionInput(char.CharacterID), admissionImages(t))
			if err != nil {
				t.Fatal(err)
			}
			if _, err = svc.Process(2, app.ID, app.Version, true, ""); err != nil {
				t.Fatal(err)
			}
			var rows []model.WalletTransaction
			if err = db.Find(&rows).Error; err != nil {
				t.Fatal(err)
			}
			if amount == 0 {
				if len(rows) != 0 {
					t.Fatal("disabled salary created transaction")
				}
			} else if len(rows) != 1 || rows[0].Amount != amount {
				t.Fatalf("salary: %+v", rows)
			}
			if _, err = svc.Process(2, app.ID, app.Version, true, ""); err == nil {
				t.Fatal("duplicate process accepted")
			}
		})
	}
}

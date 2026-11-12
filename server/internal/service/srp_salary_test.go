package service

import (
	"amiya-eden/global"
	"amiya-eden/internal/model"
	"amiya-eden/internal/testutil"
	"fmt"
	"sync"
	"testing"
	"time"
)

func TestSRPBatchSalaryCountsOperationsNotApplications(t *testing.T) {
	for _, mode := range []string{"batch_manual", "batch_fuxi"} {
		for _, amount := range []float64{0, 2.5} {
			t.Run(fmt.Sprintf("%s_%g", mode, amount), func(t *testing.T) {
				db := newSrpServiceTestDB(t)
				previous := global.DB
				global.DB = db
				t.Cleanup(func() { global.DB = previous; clearCorpPolicyCache() })
				if err := db.Create(&model.SystemConfig{Key: model.SysConfigSalarySRP, Value: fmt.Sprint(amount)}).Error; err != nil {
					t.Fatal(err)
				}
				svc := newSrpServiceForTests()
				payout := func() error {
					if mode == "batch_manual" {
						_, _, err := svc.BatchPayoutByUser(77, []string{model.RoleSRP}, 42)
						return err
					}
					_, _, err := svc.BatchPayoutAsFuxiCoin(77, []string{model.RoleSRP})
					return err
				}
				if err := payout(); err == nil {
					t.Fatal("empty batch succeeded")
				}
				for batch := 0; batch < 2; batch++ {
					for i := 0; i < 3; i++ {
						app := model.SrpApplication{UserID: 42, CharacterID: 90000042, KillmailID: int64(912300 + batch*3 + i), ShipTypeID: 1, SolarSystemID: 30000142, KillmailTime: time.Now(), RecommendedAmount: 1000000, FinalAmount: 1000000, ReviewStatus: model.SrpReviewApproved, PayoutStatus: model.SrpPayoutNotPaid}
						if err := db.Create(&app).Error; err != nil {
							t.Fatal(err)
						}
					}
					if err := payout(); err != nil {
						t.Fatal(err)
					}
					if err := payout(); err == nil {
						t.Fatal("repeat batch succeeded")
					}
					var salaries []model.WalletTransaction
					if err := db.Where("ref_type = ?", model.WalletRefSRPSalary).Find(&salaries).Error; err != nil {
						t.Fatal(err)
					}
					wantCount := batch + 1
					if amount == 0 {
						wantCount = 0
					}
					if len(salaries) != wantCount {
						t.Fatalf("batch salary count = %d, want %d", len(salaries), wantCount)
					}
					for _, salary := range salaries {
						if salary.UserID != 77 || salary.Amount != amount {
							t.Fatalf("unexpected salary: %+v", salary)
						}
					}
					var paid int64
					if err := db.Model(&model.SrpApplication{}).Where("payout_status = ?", model.SrpPayoutPaid).Count(&paid).Error; err != nil {
						t.Fatal(err)
					}
					if paid != int64((batch+1)*3) {
						t.Fatalf("paid count = %d", paid)
					}
				}
			})
		}
	}
}

func TestSRPSalaryFailureRollsBackPayout(t *testing.T) {
	for _, mode := range []string{SrpPayoutModeManualTransfer, SrpPayoutModeFuxiCoin, "batch_manual", "batch_fuxi"} {
		t.Run(mode, func(t *testing.T) {
			db := newSrpServiceTestDB(t)
			previous := global.DB
			global.DB = db
			t.Cleanup(func() { global.DB = previous; clearCorpPolicyCache() })
			// The applicant has a wallet; the human payer's corporation does not.
			if err := db.Model(&model.EveCharacter{}).Where("character_id = ?", 90000077).Update("corporation_id", 2002).Error; err != nil {
				t.Fatal(err)
			}
			for i := 0; i < 2; i++ {
				app := model.SrpApplication{UserID: 42, CharacterID: 90000042, KillmailID: int64(812300 + i), ShipTypeID: 1, SolarSystemID: 30000142, KillmailTime: time.Now(), RecommendedAmount: 1000000, FinalAmount: 1000000, ReviewStatus: model.SrpReviewApproved, PayoutStatus: model.SrpPayoutNotPaid}
				if err := db.Create(&app).Error; err != nil {
					t.Fatal(err)
				}
			}
			svc := newSrpServiceForTests()
			var err error
			switch mode {
			case "batch_manual":
				_, _, err = svc.BatchPayoutByUser(77, []string{model.RoleAdmin}, 42)
			case "batch_fuxi":
				_, _, err = svc.BatchPayoutAsFuxiCoin(77, []string{model.RoleAdmin})
			default:
				_, _, err = svc.Payout(77, []string{model.RoleAdmin}, 1, &SrpPayoutRequest{Mode: mode})
			}
			if err == nil {
				t.Fatal("disabled salary wallet allowed payout")
			}
			var paid, transactions int64
			db.Model(&model.SrpApplication{}).Where("payout_status = ?", model.SrpPayoutPaid).Count(&paid)
			db.Model(&model.WalletTransaction{}).Count(&transactions)
			if paid != 0 || transactions != 0 {
				t.Fatalf("partial payout: %d paid, %d transactions", paid, transactions)
			}
		})
	}
}

func TestSRPPayoutSalaryConcurrentPostgres(t *testing.T) {
	db := testutil.Postgres(t)
	if err := db.AutoMigrate(&model.SystemConfig{}, &model.User{}, &model.EveCharacter{}, &model.SystemWallet{}, &model.WalletTransaction{}, &model.SrpApplication{}, &model.AuditEvent{}); err != nil {
		t.Fatal(err)
	}
	previous := global.DB
	global.DB = db
	t.Cleanup(func() { global.DB = previous; clearCorpPolicyCache() })
	seedWalletCapabilityEnabledUserForTests(t, db, 42, 90000042, 1001)
	seedWalletCapabilityEnabledUserForTests(t, db, 77, 90000077, 1001)
	setWalletCapabilityPolicyForTests(t, db, 1001, true)
	app := model.SrpApplication{UserID: 42, CharacterID: 90000042, KillmailID: 812301, ShipTypeID: 1, SolarSystemID: 30000142, KillmailTime: time.Now(), RecommendedAmount: 1000000, FinalAmount: 1000000, ReviewStatus: model.SrpReviewApproved, PayoutStatus: model.SrpPayoutNotPaid}
	if err := db.Create(&app).Error; err != nil {
		t.Fatal(err)
	}
	svc := newSrpServiceForTests()
	start := make(chan struct{})
	results := make(chan error, 2)
	var wg sync.WaitGroup
	for range 2 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			_, _, err := svc.Payout(77, []string{model.RoleSRP}, app.ID, &SrpPayoutRequest{Mode: SrpPayoutModeFuxiCoin})
			results <- err
		}()
	}
	close(start)
	wg.Wait()
	close(results)
	success := 0
	for err := range results {
		if err == nil {
			success++
		}
	}
	var rows []model.WalletTransaction
	if err := db.Where("ref_type = ?", model.WalletRefSRPSalary).Find(&rows).Error; err != nil {
		t.Fatal(err)
	}
	if success != 1 || len(rows) != 1 || rows[0].Amount != 10 {
		t.Fatalf("duplicate payout: %d successes, %+v", success, rows)
	}
}

package service

import (
	"amiya-eden/global"
	"amiya-eden/internal/model"
	"fmt"
	"testing"
	"time"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func newFuxiAdminServiceTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := fmt.Sprintf("file:fuxi_admin_svc_%d?mode=memory&cache=shared", time.Now().UnixNano())
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&model.FuxiAdminConfig{}, &model.FuxiAdminTier{}, &model.FuxiAdmin{}); err != nil {
		t.Fatalf("auto migrate: %v", err)
	}
	return db
}

func TestFuxiAdminGetConfigReturnsDefaultWhenAbsent(t *testing.T) {
	global.DB = newFuxiAdminServiceTestDB(t)
	svc := NewFuxiAdminService()

	cfg, err := svc.GetConfig()
	if err != nil {
		t.Fatalf("GetConfig: %v", err)
	}
	if cfg.BaseFontSize != 14 {
		t.Fatalf("expected default BaseFontSize 14, got %d", cfg.BaseFontSize)
	}
}

func TestFuxiAdminUpdateConfigValidatesRange(t *testing.T) {
	global.DB = newFuxiAdminServiceTestDB(t)
	svc := NewFuxiAdminService()

	tooSmall := 5
	if _, err := svc.UpdateConfig(&FuxiAdminUpdateConfigRequest{BaseFontSize: &tooSmall}); err == nil {
		t.Fatal("expected error for font size 5")
	}

	tooBig := 40
	if _, err := svc.UpdateConfig(&FuxiAdminUpdateConfigRequest{BaseFontSize: &tooBig}); err == nil {
		t.Fatal("expected error for font size 40")
	}

	valid := 16
	cfg, err := svc.UpdateConfig(&FuxiAdminUpdateConfigRequest{BaseFontSize: &valid})
	if err != nil {
		t.Fatalf("expected success for font size 16, got: %v", err)
	}
	if cfg.BaseFontSize != 16 {
		t.Fatalf("expected BaseFontSize 16, got %d", cfg.BaseFontSize)
	}
}

func TestFuxiAdminCreateTierRejectsEmptyName(t *testing.T) {
	global.DB = newFuxiAdminServiceTestDB(t)
	svc := NewFuxiAdminService()

	_, err := svc.CreateTier(&FuxiAdminCreateTierRequest{Name: ""})
	if err == nil {
		t.Fatal("expected error for empty tier name")
	}
}

func TestFuxiAdminCreateAdminRejectsInvalidTierID(t *testing.T) {
	global.DB = newFuxiAdminServiceTestDB(t)
	svc := NewFuxiAdminService()

	_, err := svc.CreateAdmin(&FuxiAdminCreateAdminRequest{TierID: 999, Name: "Ghost"})
	if err == nil {
		t.Fatal("expected error for non-existent tier")
	}
}

func TestFuxiAdminGetDirectoryGroupsAdminsByTier(t *testing.T) {
	global.DB = newFuxiAdminServiceTestDB(t)
	svc := NewFuxiAdminService()

	_, _ = svc.CreateTier(&FuxiAdminCreateTierRequest{Name: "High"})
	tiers, _ := svc.ListTiers()
	highTierID := tiers[0].ID

	_, _ = svc.CreateAdmin(&FuxiAdminCreateAdminRequest{TierID: highTierID, Name: "Alpha"})
	_, _ = svc.CreateAdmin(&FuxiAdminCreateAdminRequest{TierID: highTierID, Name: "Beta"})

	dir, err := svc.GetDirectory()
	if err != nil {
		t.Fatalf("GetDirectory: %v", err)
	}
	if len(dir.Tiers) != 1 {
		t.Fatalf("expected 1 tier, got %d", len(dir.Tiers))
	}
	if len(dir.Tiers[0].Admins) != 2 {
		t.Fatalf("expected 2 admins in tier, got %d", len(dir.Tiers[0].Admins))
	}
}

func TestFuxiAdminDeleteTierCascadesToAdmins(t *testing.T) {
	global.DB = newFuxiAdminServiceTestDB(t)
	svc := NewFuxiAdminService()

	_, _ = svc.CreateTier(&FuxiAdminCreateTierRequest{Name: "Mid"})
	tiers, _ := svc.ListTiers()
	tierID := tiers[0].ID
	_, _ = svc.CreateAdmin(&FuxiAdminCreateAdminRequest{TierID: tierID, Name: "Member"})

	if err := svc.DeleteTier(tierID); err != nil {
		t.Fatalf("DeleteTier: %v", err)
	}

	dir, _ := svc.GetDirectory()
	if len(dir.Tiers) != 0 {
		t.Fatalf("expected 0 tiers after delete, got %d", len(dir.Tiers))
	}
}

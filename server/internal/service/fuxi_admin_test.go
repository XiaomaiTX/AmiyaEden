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

	tooNarrowCard := 120
	if _, err := svc.UpdateConfig(&FuxiAdminUpdateConfigRequest{CardWidth: &tooNarrowCard}); err == nil {
		t.Fatal("expected error for card width 120")
	}
}

func TestFuxiAdminUpdateConfigStoresStyleOptions(t *testing.T) {
	global.DB = newFuxiAdminServiceTestDB(t)
	svc := NewFuxiAdminService()

	pageBackgroundColor := "#10243a"
	cardBackgroundColor := "#1e3650"
	cardBorderColor := "#d9a441"
	tierTitleColor := "#ffd56a"
	nameTextColor := "#fff7d6"
	bodyTextColor := "#d7dfef"
	cardWidth := 280

	cfg, err := svc.UpdateConfig(&FuxiAdminUpdateConfigRequest{
		PageBackgroundColor: &pageBackgroundColor,
		CardBackgroundColor: &cardBackgroundColor,
		CardBorderColor:     &cardBorderColor,
		TierTitleColor:      &tierTitleColor,
		NameTextColor:       &nameTextColor,
		BodyTextColor:       &bodyTextColor,
		CardWidth:           &cardWidth,
	})
	if err != nil {
		t.Fatalf("expected config update to succeed, got: %v", err)
	}
	if cfg.PageBackgroundColor != pageBackgroundColor {
		t.Fatalf("expected page background color %q, got %q", pageBackgroundColor, cfg.PageBackgroundColor)
	}
	if cfg.CardBackgroundColor != cardBackgroundColor {
		t.Fatalf("expected card background color %q, got %q", cardBackgroundColor, cfg.CardBackgroundColor)
	}
	if cfg.CardBorderColor != cardBorderColor {
		t.Fatalf("expected card border color %q, got %q", cardBorderColor, cfg.CardBorderColor)
	}
	if cfg.TierTitleColor != tierTitleColor {
		t.Fatalf("expected tier title color %q, got %q", tierTitleColor, cfg.TierTitleColor)
	}
	if cfg.NameTextColor != nameTextColor {
		t.Fatalf("expected name text color %q, got %q", nameTextColor, cfg.NameTextColor)
	}
	if cfg.BodyTextColor != bodyTextColor {
		t.Fatalf("expected body text color %q, got %q", bodyTextColor, cfg.BodyTextColor)
	}
	if cfg.CardWidth != cardWidth {
		t.Fatalf("expected card width %d, got %d", cardWidth, cfg.CardWidth)
	}

	reloaded, err := svc.GetConfig()
	if err != nil {
		t.Fatalf("GetConfig: %v", err)
	}
	if reloaded.PageBackgroundColor != pageBackgroundColor || reloaded.CardBackgroundColor != cardBackgroundColor {
		t.Fatalf("expected colors to persist, got %+v", reloaded)
	}
	if reloaded.CardBorderColor != cardBorderColor || reloaded.CardWidth != cardWidth {
		t.Fatalf("expected style options to persist, got %+v", reloaded)
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

func TestFuxiAdminCreateAdminStoresDescription(t *testing.T) {
	global.DB = newFuxiAdminServiceTestDB(t)
	svc := NewFuxiAdminService()

	_, _ = svc.CreateTier(&FuxiAdminCreateTierRequest{Name: "Ops"})
	tiers, _ := svc.ListTiers()
	tierID := tiers[0].ID

	admin, err := svc.CreateAdmin(&FuxiAdminCreateAdminRequest{
		TierID:      tierID,
		Name:        "Alpha",
		Title:       "Coordinator",
		Description: "Handles alliance operations and escalation",
	})
	if err != nil {
		t.Fatalf("CreateAdmin: %v", err)
	}
	if admin.Description != "Handles alliance operations and escalation" {
		t.Fatalf("expected description to be stored, got %q", admin.Description)
	}

	dir, err := svc.GetDirectory()
	if err != nil {
		t.Fatalf("GetDirectory: %v", err)
	}
	if len(dir.Tiers) != 1 || len(dir.Tiers[0].Admins) != 1 {
		t.Fatalf("expected one admin in one tier, got %+v", dir)
	}
	if dir.Tiers[0].Admins[0].Description != "Handles alliance operations and escalation" {
		t.Fatalf("expected description in directory response, got %q", dir.Tiers[0].Admins[0].Description)
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

package service

import (
	"amiya-eden/global"
	"amiya-eden/internal/model"
	"errors"
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
	if err := db.AutoMigrate(
		&model.FuxiAdminConfig{},
		&model.FuxiAdminTier{},
		&model.FuxiAdmin{},
		&model.User{},
		&model.EveCharacter{},
		&model.Fleet{},
		&model.WelfareApplication{},
		&model.ShopOrder{},
	); err != nil {
		t.Fatalf("auto migrate: %v", err)
	}
	return db
}

func useFuxiAdminServiceTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	db := newFuxiAdminServiceTestDB(t)
	oldDB := global.DB
	global.DB = db
	t.Cleanup(func() { global.DB = oldDB })

	return db
}

func TestFuxiAdminGetConfigReturnsDefaultWhenAbsent(t *testing.T) {
	useFuxiAdminServiceTestDB(t)
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
	useFuxiAdminServiceTestDB(t)
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
	useFuxiAdminServiceTestDB(t)
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
	useFuxiAdminServiceTestDB(t)
	svc := NewFuxiAdminService()

	_, err := svc.CreateTier(&FuxiAdminCreateTierRequest{Name: ""})
	if err == nil {
		t.Fatal("expected error for empty tier name")
	}
}

func TestFuxiAdminCreateTierRejectsWhitespaceOnlyName(t *testing.T) {
	useFuxiAdminServiceTestDB(t)
	svc := NewFuxiAdminService()

	_, err := svc.CreateTier(&FuxiAdminCreateTierRequest{Name: "   "})
	if err == nil {
		t.Fatal("expected error for whitespace-only tier name")
	}
}

func TestFuxiAdminCreateTierTrimsWhitespace(t *testing.T) {
	useFuxiAdminServiceTestDB(t)
	svc := NewFuxiAdminService()

	tier, err := svc.CreateTier(&FuxiAdminCreateTierRequest{Name: "  Ops  "})
	if err != nil {
		t.Fatalf("CreateTier: %v", err)
	}
	if tier.Name != "Ops" {
		t.Fatalf("expected trimmed tier name, got %q", tier.Name)
	}
}

func TestFuxiAdminCreateAdminRejectsInvalidTierID(t *testing.T) {
	useFuxiAdminServiceTestDB(t)
	svc := NewFuxiAdminService()

	_, err := svc.CreateAdmin(&FuxiAdminCreateAdminRequest{TierID: 999, Nickname: "Ghost"})
	if err == nil {
		t.Fatal("expected error for non-existent tier")
	}
}

func TestFuxiAdminUpdateTierRejectsWhitespaceOnlyName(t *testing.T) {
	useFuxiAdminServiceTestDB(t)
	svc := NewFuxiAdminService()

	tier, err := svc.CreateTier(&FuxiAdminCreateTierRequest{Name: "Ops"})
	if err != nil {
		t.Fatalf("CreateTier: %v", err)
	}
	blank := "   "

	_, err = svc.UpdateTier(tier.ID, &FuxiAdminUpdateTierRequest{Name: &blank})
	if err == nil {
		t.Fatal("expected error for whitespace-only tier name")
	}
}

func TestFuxiAdminUpdateTierTrimsWhitespace(t *testing.T) {
	useFuxiAdminServiceTestDB(t)
	svc := NewFuxiAdminService()

	tier, err := svc.CreateTier(&FuxiAdminCreateTierRequest{Name: "Ops"})
	if err != nil {
		t.Fatalf("CreateTier: %v", err)
	}
	name := "  Command  "

	updatedTier, err := svc.UpdateTier(tier.ID, &FuxiAdminUpdateTierRequest{Name: &name})
	if err != nil {
		t.Fatalf("UpdateTier: %v", err)
	}
	if updatedTier.Name != "Command" {
		t.Fatalf("expected trimmed tier name, got %q", updatedTier.Name)
	}
}

func TestFuxiAdminCreateAdminStoresDescription(t *testing.T) {
	useFuxiAdminServiceTestDB(t)
	svc := NewFuxiAdminService()

	_, _ = svc.CreateTier(&FuxiAdminCreateTierRequest{Name: "Ops"})
	tiers, _ := svc.ListTiers()
	tierID := tiers[0].ID

	admin, err := svc.CreateAdmin(&FuxiAdminCreateAdminRequest{
		TierID:        tierID,
		Nickname:      "Alpha",
		CharacterName: "Coordinator",
		Description:   "Handles alliance operations and escalation",
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
	useFuxiAdminServiceTestDB(t)
	svc := NewFuxiAdminService()

	_, _ = svc.CreateTier(&FuxiAdminCreateTierRequest{Name: "High"})
	tiers, _ := svc.ListTiers()
	highTierID := tiers[0].ID

	_, _ = svc.CreateAdmin(&FuxiAdminCreateAdminRequest{TierID: highTierID, Nickname: "Alpha"})
	_, _ = svc.CreateAdmin(&FuxiAdminCreateAdminRequest{TierID: highTierID, Nickname: "Beta"})

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
	useFuxiAdminServiceTestDB(t)
	svc := NewFuxiAdminService()

	_, _ = svc.CreateTier(&FuxiAdminCreateTierRequest{Name: "Mid"})
	tiers, _ := svc.ListTiers()
	tierID := tiers[0].ID
	_, _ = svc.CreateAdmin(&FuxiAdminCreateAdminRequest{TierID: tierID, Nickname: "Member"})

	if err := svc.DeleteTier(tierID); err != nil {
		t.Fatalf("DeleteTier: %v", err)
	}

	dir, _ := svc.GetDirectory()
	if len(dir.Tiers) != 0 {
		t.Fatalf("expected 0 tiers after delete, got %d", len(dir.Tiers))
	}
}

func TestFuxiAdminDeleteTierReturnsNotFoundWhenTierMissing(t *testing.T) {
	useFuxiAdminServiceTestDB(t)
	svc := NewFuxiAdminService()

	err := svc.DeleteTier(999)
	if err == nil {
		t.Fatal("expected not-found error")
	}
	if !IsUserVisibleError(err) {
		t.Fatalf("expected not-found error to be user-visible, got %v", err)
	}
	if err.Error() != "层级不存在" {
		t.Fatalf("expected 层级不存在, got %v", err)
	}
}

func TestFuxiAdminDeleteTierRollsBackWhenTierDeleteFails(t *testing.T) {
	useFuxiAdminServiceTestDB(t)
	svc := NewFuxiAdminService()

	tier, err := svc.CreateTier(&FuxiAdminCreateTierRequest{Name: "Mid"})
	if err != nil {
		t.Fatalf("CreateTier: %v", err)
	}
	if _, err := svc.CreateAdmin(&FuxiAdminCreateAdminRequest{TierID: tier.ID, Nickname: "Member"}); err != nil {
		t.Fatalf("CreateAdmin: %v", err)
	}

	callbackName := fmt.Sprintf("fuxi_admin_fail_delete_%d", time.Now().UnixNano())
	if err := global.DB.Callback().Delete().Before("gorm:delete").Register(callbackName, func(tx *gorm.DB) {
		if tx.Statement != nil && tx.Statement.Table == (model.FuxiAdminTier{}).TableName() {
			_ = tx.AddError(errors.New("forced tier delete failure"))
		}
	}); err != nil {
		t.Fatalf("register delete callback: %v", err)
	}
	t.Cleanup(func() {
		if err := global.DB.Callback().Delete().Remove(callbackName); err != nil {
			t.Errorf("remove delete callback: %v", err)
		}
	})

	if err := svc.DeleteTier(tier.ID); err == nil {
		t.Fatal("expected DeleteTier to fail")
	}

	dir, err := svc.GetDirectory()
	if err != nil {
		t.Fatalf("GetDirectory: %v", err)
	}
	if len(dir.Tiers) != 1 {
		t.Fatalf("expected tier delete rollback to keep 1 tier, got %d", len(dir.Tiers))
	}
	if len(dir.Tiers[0].Admins) != 1 {
		t.Fatalf("expected tier delete rollback to keep 1 admin, got %d", len(dir.Tiers[0].Admins))
	}
}

func TestFuxiAdminUpdateTierPreservesInfrastructureErrors(t *testing.T) {
	useFuxiAdminServiceTestDB(t)
	svc := NewFuxiAdminService()

	tier, err := svc.CreateTier(&FuxiAdminCreateTierRequest{Name: "Ops"})
	if err != nil {
		t.Fatalf("CreateTier: %v", err)
	}
	renamed := "Operations"

	sqlDB, err := global.DB.DB()
	if err != nil {
		t.Fatalf("open sql db: %v", err)
	}
	if err := sqlDB.Close(); err != nil {
		t.Fatalf("close sql db: %v", err)
	}

	_, err = svc.UpdateTier(tier.ID, &FuxiAdminUpdateTierRequest{Name: &renamed})
	if err == nil {
		t.Fatal("expected infrastructure error")
	}
	if IsUserVisibleError(err) {
		t.Fatalf("expected infrastructure error to remain non-user-visible, got %v", err)
	}
	if err.Error() == "层级不存在" {
		t.Fatalf("expected infrastructure error instead of not-found message, got %v", err)
	}
}

func TestFuxiAdminUpdateAdminPreservesInfrastructureErrors(t *testing.T) {
	useFuxiAdminServiceTestDB(t)
	svc := NewFuxiAdminService()

	tier, err := svc.CreateTier(&FuxiAdminCreateTierRequest{Name: "Ops"})
	if err != nil {
		t.Fatalf("CreateTier: %v", err)
	}
	admin, err := svc.CreateAdmin(&FuxiAdminCreateAdminRequest{TierID: tier.ID, Nickname: "Alpha"})
	if err != nil {
		t.Fatalf("CreateAdmin: %v", err)
	}
	renamed := "Bravo"

	sqlDB, err := global.DB.DB()
	if err != nil {
		t.Fatalf("open sql db: %v", err)
	}
	if err := sqlDB.Close(); err != nil {
		t.Fatalf("close sql db: %v", err)
	}

	_, err = svc.UpdateAdmin(admin.ID, &FuxiAdminUpdateAdminRequest{Nickname: &renamed})
	if err == nil {
		t.Fatal("expected infrastructure error")
	}
	if IsUserVisibleError(err) {
		t.Fatalf("expected infrastructure error to remain non-user-visible, got %v", err)
	}
	if err.Error() == "管理员不存在" {
		t.Fatalf("expected infrastructure error instead of not-found message, got %v", err)
	}
}

func TestFuxiAdminGetManageDirectoryIncludesCountsAndOffset(t *testing.T) {
	useFuxiAdminServiceTestDB(t)
	svc := NewFuxiAdminService()

	tier, err := svc.CreateTier(&FuxiAdminCreateTierRequest{Name: "Ops"})
	if err != nil {
		t.Fatalf("CreateTier: %v", err)
	}

	linkedUser := model.User{Nickname: "Linked User", PrimaryCharacterID: 9999}
	if err := global.DB.Create(&linkedUser).Error; err != nil {
		t.Fatalf("create linked user: %v", err)
	}
	linkedCharacter := model.EveCharacter{CharacterID: 1001, CharacterName: "Linked Alt", UserID: linkedUser.ID}
	if err := global.DB.Create(&linkedCharacter).Error; err != nil {
		t.Fatalf("create linked character: %v", err)
	}

	admins := []model.FuxiAdmin{
		{TierID: tier.ID, Nickname: "Linked", CharacterID: 1001, WelfareDeliveryOffset: 4},
		{TierID: tier.ID, Nickname: "Unlinked", CharacterID: 2002},
	}
	if err := global.DB.Create(&admins).Error; err != nil {
		t.Fatalf("create admins: %v", err)
	}

	deletedAt := time.Now()
	fleets := []model.Fleet{
		{ID: "fleet-1", Title: "Fleet One", StartAt: time.Now(), EndAt: time.Now().Add(time.Hour), Importance: model.FleetImportanceOther, FCUserID: linkedUser.ID, FCCharacterID: 1001, FCCharacterName: "Linked"},
		{ID: "fleet-2", Title: "Fleet Two", StartAt: time.Now(), EndAt: time.Now().Add(time.Hour), Importance: model.FleetImportanceOther, FCUserID: linkedUser.ID, FCCharacterID: 1001, FCCharacterName: "Linked"},
		{ID: "fleet-3", Title: "Deleted Fleet", StartAt: time.Now(), EndAt: time.Now().Add(time.Hour), Importance: model.FleetImportanceOther, FCUserID: linkedUser.ID, FCCharacterID: 1001, FCCharacterName: "Linked", DeletedAt: &deletedAt},
	}
	if err := global.DB.Create(&fleets).Error; err != nil {
		t.Fatalf("create fleets: %v", err)
	}

	welfareApps := []model.WelfareApplication{
		{WelfareID: 1, CharacterID: 1001, CharacterName: "One", Status: model.WelfareAppStatusDelivered, ReviewedBy: linkedUser.ID},
		{WelfareID: 1, CharacterID: 1002, CharacterName: "Two", Status: model.WelfareAppStatusDelivered, ReviewedBy: linkedUser.ID},
		{WelfareID: 1, CharacterID: 1003, CharacterName: "Three", Status: model.WelfareAppStatusRequested, ReviewedBy: linkedUser.ID},
	}
	if err := global.DB.Create(&welfareApps).Error; err != nil {
		t.Fatalf("create welfare applications: %v", err)
	}

	reviewedBy := linkedUser.ID
	shopOrders := []model.ShopOrder{
		{OrderNo: "SO-1", UserID: linkedUser.ID, ProductID: 1, ProductName: "One", ProductType: model.ProductTypeNormal, UnitPrice: 1, TotalPrice: 1, Status: model.OrderStatusDelivered, ReviewedBy: &reviewedBy},
		{OrderNo: "SO-2", UserID: linkedUser.ID, ProductID: 1, ProductName: "Two", ProductType: model.ProductTypeNormal, UnitPrice: 1, TotalPrice: 1, Status: model.OrderStatusDelivered, ReviewedBy: &reviewedBy},
		{OrderNo: "SO-3", UserID: linkedUser.ID, ProductID: 1, ProductName: "Three", ProductType: model.ProductTypeNormal, UnitPrice: 1, TotalPrice: 1, Status: model.OrderStatusDelivered, ReviewedBy: &reviewedBy},
		{OrderNo: "SO-4", UserID: linkedUser.ID, ProductID: 1, ProductName: "Four", ProductType: model.ProductTypeNormal, UnitPrice: 1, TotalPrice: 1, Status: model.OrderStatusRequested, ReviewedBy: &reviewedBy},
	}
	if err := global.DB.Create(&shopOrders).Error; err != nil {
		t.Fatalf("create shop orders: %v", err)
	}

	dir, err := svc.GetManageDirectory()
	if err != nil {
		t.Fatalf("GetManageDirectory: %v", err)
	}
	if len(dir.Tiers) != 1 {
		t.Fatalf("expected 1 tier, got %d", len(dir.Tiers))
	}
	if len(dir.Tiers[0].Admins) != 2 {
		t.Fatalf("expected 2 admins, got %d", len(dir.Tiers[0].Admins))
	}

	gotByNickname := make(map[string]FuxiAdminManageAdmin, len(dir.Tiers[0].Admins))
	for _, admin := range dir.Tiers[0].Admins {
		gotByNickname[admin.Nickname] = admin
	}

	linked := gotByNickname["Linked"]
	if linked.FleetLedCount != 2 {
		t.Fatalf("expected linked admin fleet count 2, got %d", linked.FleetLedCount)
	}
	if linked.WelfareDeliveryCount != 9 {
		t.Fatalf("expected linked admin welfare delivery count 9, got %d", linked.WelfareDeliveryCount)
	}
	if linked.WelfareDeliveryOffset != 4 {
		t.Fatalf("expected linked admin offset 4, got %d", linked.WelfareDeliveryOffset)
	}

	unlinked := gotByNickname["Unlinked"]
	if unlinked.FleetLedCount != 0 {
		t.Fatalf("expected unlinked admin fleet count 0, got %d", unlinked.FleetLedCount)
	}
	if unlinked.WelfareDeliveryCount != 0 {
		t.Fatalf("expected unlinked admin welfare delivery count 0, got %d", unlinked.WelfareDeliveryCount)
	}
}

func TestFuxiAdminUpdateAdminRejectsNegativeWelfareDeliveryOffset(t *testing.T) {
	useFuxiAdminServiceTestDB(t)
	svc := NewFuxiAdminService()

	tier, err := svc.CreateTier(&FuxiAdminCreateTierRequest{Name: "Ops"})
	if err != nil {
		t.Fatalf("CreateTier: %v", err)
	}
	admin, err := svc.CreateAdmin(&FuxiAdminCreateAdminRequest{TierID: tier.ID, Nickname: "Alpha"})
	if err != nil {
		t.Fatalf("CreateAdmin: %v", err)
	}

	negativeOffset := -1
	_, err = svc.UpdateAdmin(admin.ID, &FuxiAdminUpdateAdminRequest{WelfareDeliveryOffset: &negativeOffset})
	if err == nil {
		t.Fatal("expected negative welfare delivery offset to be rejected")
	}
	if !IsUserVisibleError(err) {
		t.Fatalf("expected user-visible error, got %v", err)
	}
	if err.Error() != "福利发放次数偏移不能为负数" {
		t.Fatalf("unexpected error message: %v", err)
	}
}

func TestFuxiAdminGetManageDirectoryFallsBackToPrimaryCharacterLookup(t *testing.T) {
	useFuxiAdminServiceTestDB(t)
	svc := NewFuxiAdminService()

	tier, err := svc.CreateTier(&FuxiAdminCreateTierRequest{Name: "Ops"})
	if err != nil {
		t.Fatalf("CreateTier: %v", err)
	}

	linkedUser := model.User{Nickname: "Primary Linked User", PrimaryCharacterID: 3003}
	if err := global.DB.Create(&linkedUser).Error; err != nil {
		t.Fatalf("create linked user: %v", err)
	}

	admins := []model.FuxiAdmin{{TierID: tier.ID, Nickname: "Primary Linked", CharacterID: 3003}}
	if err := global.DB.Create(&admins).Error; err != nil {
		t.Fatalf("create admins: %v", err)
	}

	fleets := []model.Fleet{
		{ID: "fleet-primary-1", Title: "Fleet One", StartAt: time.Now(), EndAt: time.Now().Add(time.Hour), Importance: model.FleetImportanceOther, FCUserID: linkedUser.ID, FCCharacterID: 3003, FCCharacterName: "Primary Linked"},
	}
	if err := global.DB.Create(&fleets).Error; err != nil {
		t.Fatalf("create fleets: %v", err)
	}

	dir, err := svc.GetManageDirectory()
	if err != nil {
		t.Fatalf("GetManageDirectory: %v", err)
	}
	if len(dir.Tiers) != 1 || len(dir.Tiers[0].Admins) != 1 {
		t.Fatalf("unexpected directory shape: %+v", dir.Tiers)
	}
	if dir.Tiers[0].Admins[0].FleetLedCount != 1 {
		t.Fatalf("expected primary-character fallback fleet count 1, got %d", dir.Tiers[0].Admins[0].FleetLedCount)
	}
}

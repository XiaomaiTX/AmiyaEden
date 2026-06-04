package service

import (
	"amiya-eden/global"
	"amiya-eden/internal/model"
	"testing"
	"time"

	"gorm.io/gorm"
)

func newGalaxyRegistryServiceTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	return newServiceTestDB(
		t,
		"galaxy_registry",
		&model.User{},
		&model.EveCharacter{},
		&model.GalaxyRegistrySystem{},
		&model.GalaxyRegistryEntry{},
		&model.EVECharacterWalletJournal{},
	)
}

func TestGalaxyRegistryStartEntryRejectsUsersWithoutCharacters(t *testing.T) {
	db := newGalaxyRegistryServiceTestDB(t)
	previous := global.DB
	global.DB = db
	t.Cleanup(func() { global.DB = previous })

	if err := db.Create(&model.User{
		BaseModel: model.BaseModel{ID: 1},
		Nickname:  "captain-one",
		Role:      model.RoleCaptain,
	}).Error; err != nil {
		t.Fatalf("create user: %v", err)
	}
	if err := db.Create(&model.GalaxyRegistrySystem{
		SolarSystemID:     30000142,
		SolarSystemName:   "Jita",
		RegionID:          10000002,
		RegionName:        "The Forge",
		ConstellationID:   20000020,
		ConstellationName: "Kimotoro",
		Security:          0.9,
		MinBountyAmount:   model.GalaxyRegistryDefaultMinBountyAmount,
		IsEnabled:         true,
	}).Error; err != nil {
		t.Fatalf("create system: %v", err)
	}

	svc := NewGalaxyRegistryService()
	_, err := svc.StartEntry(1, GalaxyRegistryCreateEntryRequest{
		SystemConfigID: 1,
		ExpectedEndAt:  time.Now().Add(2 * time.Hour),
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !IsUserVisibleError(err) || err.Error() != "当前队长未绑定任何角色" {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestGalaxyRegistryStartAndEndEntryFlow(t *testing.T) {
	db := newGalaxyRegistryServiceTestDB(t)
	previous := global.DB
	global.DB = db
	t.Cleanup(func() { global.DB = previous })

	if err := db.Create(&model.User{
		BaseModel:          model.BaseModel{ID: 2},
		Nickname:           "captain-two",
		PrimaryCharacterID: 90000002,
		Role:               model.RoleCaptain,
	}).Error; err != nil {
		t.Fatalf("create user: %v", err)
	}
	if err := db.Create(&model.EveCharacter{
		CharacterID:   90000002,
		CharacterName: "Captain Main",
		UserID:        2,
	}).Error; err != nil {
		t.Fatalf("create character: %v", err)
	}
	if err := db.Create(&model.GalaxyRegistrySystem{
		SolarSystemID:     30002187,
		SolarSystemName:   "MJ-5F9",
		RegionID:          10000060,
		RegionName:        "Delve",
		ConstellationID:   20000690,
		ConstellationName: "8WA-Z6",
		Security:          -0.1,
		MinBountyAmount:   12000000,
		IsEnabled:         true,
	}).Error; err != nil {
		t.Fatalf("create system: %v", err)
	}

	svc := NewGalaxyRegistryService()
	entry, err := svc.StartEntry(2, GalaxyRegistryCreateEntryRequest{
		SystemConfigID: 1,
		ExpectedEndAt:  time.Now().Add(90 * time.Minute),
	})
	if err != nil {
		t.Fatalf("StartEntry() error = %v", err)
	}
	if entry.Status != model.GalaxyRegistryEntryStatusActive {
		t.Fatalf("status = %q, want %q", entry.Status, model.GalaxyRegistryEntryStatusActive)
	}
	if entry.CaptainCharacterID != 90000002 {
		t.Fatalf("captain_character_id = %d, want 90000002", entry.CaptainCharacterID)
	}

	_, err = svc.StartEntry(2, GalaxyRegistryCreateEntryRequest{
		SystemConfigID: 1,
		ExpectedEndAt:  time.Now().Add(2 * time.Hour),
	})
	if err == nil {
		t.Fatal("expected duplicate active entry error")
	}
	if !IsUserVisibleError(err) || err.Error() != "该星系当前已有生产登记" {
		t.Fatalf("unexpected duplicate error: %v", err)
	}

	ended, err := svc.EndMyEntry(2, entry.ID)
	if err != nil {
		t.Fatalf("EndMyEntry() error = %v", err)
	}
	if ended.Status != model.GalaxyRegistryEntryStatusCompleted {
		t.Fatalf("status = %q, want %q", ended.Status, model.GalaxyRegistryEntryStatusCompleted)
	}
	if ended.ActualEndAt == nil {
		t.Fatal("actual_end_at should be set")
	}
}

func TestGalaxyRegistryValidateCompletedEntriesMarksValidAndViolation(t *testing.T) {
	db := newGalaxyRegistryServiceTestDB(t)
	previous := global.DB
	global.DB = db
	t.Cleanup(func() { global.DB = previous })

	user := &model.User{BaseModel: model.BaseModel{ID: 3}, Nickname: "captain-three", Role: model.RoleCaptain}
	if err := db.Create(user).Error; err != nil {
		t.Fatalf("create user: %v", err)
	}
	characters := []model.EveCharacter{
		{CharacterID: 91000001, CharacterName: "Alt One", UserID: 3},
		{CharacterID: 91000002, CharacterName: "Alt Two", UserID: 3},
	}
	if err := db.Create(&characters).Error; err != nil {
		t.Fatalf("create characters: %v", err)
	}

	validEndAt := time.Now().Add(-2 * time.Hour)
	violationEndAt := time.Now().Add(-26 * time.Hour)
	entries := []model.GalaxyRegistryEntry{
		{
			SystemConfigID:        1,
			SolarSystemID:         30004759,
			SolarSystemName:       "1DQ1-A",
			CaptainUserID:         3,
			CaptainCharacterID:    91000001,
			CaptainCharacterName:  "Alt One",
			Status:                model.GalaxyRegistryEntryStatusCompleted,
			ValidationStatus:      model.GalaxyRegistryValidationPending,
			ExpectedEndAt:         validEndAt.Add(-1 * time.Hour),
			ActualStartAt:         validEndAt.Add(-30 * time.Minute),
			ActualEndAt:           &validEndAt,
			FrozenMinBountyAmount: 10000000,
		},
		{
			SystemConfigID:        2,
			SolarSystemID:         30004760,
			SolarSystemName:       "T5ZI-S",
			CaptainUserID:         3,
			CaptainCharacterID:    91000002,
			CaptainCharacterName:  "Alt Two",
			Status:                model.GalaxyRegistryEntryStatusCompleted,
			ValidationStatus:      model.GalaxyRegistryValidationPending,
			ExpectedEndAt:         violationEndAt.Add(-1 * time.Hour),
			ActualStartAt:         violationEndAt.Add(-30 * time.Minute),
			ActualEndAt:           &violationEndAt,
			FrozenMinBountyAmount: 10000000,
		},
	}
	if err := db.Create(&entries).Error; err != nil {
		t.Fatalf("create entries: %v", err)
	}

	if err := db.Create(&model.EVECharacterWalletJournal{
		ID:            8001,
		CharacterID:   91000002,
		Amount:        12000000,
		Balance:       12000000,
		ContextID:     30004759,
		ContextIDType: "solar_system_id",
		Date:          validEndAt.Add(-15 * time.Minute),
		Description:   "valid bounty",
		FirstPartyID:  1,
		Reason:        "",
		RefType:       "bounty_prizes",
		SecondPartyID: 2,
		Tax:           0,
		TaxReceiverID: 0,
	}).Error; err != nil {
		t.Fatalf("create journal: %v", err)
	}

	svc := NewGalaxyRegistryService()
	count, err := svc.ValidateCompletedEntries(10)
	if err != nil {
		t.Fatalf("ValidateCompletedEntries() error = %v", err)
	}
	if count != 2 {
		t.Fatalf("validated count = %d, want 2", count)
	}

	validRow, err := svc.repo.GetEntryByID(entries[0].ID)
	if err != nil {
		t.Fatalf("reload valid row: %v", err)
	}
	if validRow.ValidationStatus != model.GalaxyRegistryValidationValid {
		t.Fatalf("validation_status = %q, want %q", validRow.ValidationStatus, model.GalaxyRegistryValidationValid)
	}
	if validRow.ValidatedBountyAmount != 12000000 {
		t.Fatalf("validated_bounty_amount = %v, want 12000000", validRow.ValidatedBountyAmount)
	}

	violationRow, err := svc.repo.GetEntryByID(entries[1].ID)
	if err != nil {
		t.Fatalf("reload violation row: %v", err)
	}
	if violationRow.ValidationStatus != model.GalaxyRegistryValidationViolation {
		t.Fatalf("validation_status = %q, want %q", violationRow.ValidationStatus, model.GalaxyRegistryValidationViolation)
	}
	if violationRow.ViolationReason != model.GalaxyRegistryViolationNoBountyInWindow {
		t.Fatalf("violation_reason = %q, want %q", violationRow.ViolationReason, model.GalaxyRegistryViolationNoBountyInWindow)
	}
}

func TestGalaxyRegistryDeleteAdminSystemRejectsActiveEntries(t *testing.T) {
	db := newGalaxyRegistryServiceTestDB(t)
	previous := global.DB
	global.DB = db
	t.Cleanup(func() { global.DB = previous })

	if err := db.Create(&model.GalaxyRegistrySystem{
		SolarSystemID:     30000142,
		SolarSystemName:   "Jita",
		RegionID:          10000002,
		RegionName:        "The Forge",
		ConstellationID:   20000020,
		ConstellationName: "Kimotoro",
		Security:          0.9,
		MinBountyAmount:   model.GalaxyRegistryDefaultMinBountyAmount,
		IsEnabled:         true,
	}).Error; err != nil {
		t.Fatalf("create system: %v", err)
	}
	if err := db.Create(&model.GalaxyRegistryEntry{
		SystemConfigID:        1,
		SolarSystemID:         30000142,
		SolarSystemName:       "Jita",
		CaptainUserID:         9,
		CaptainCharacterID:    9009,
		CaptainCharacterName:  "Captain",
		Status:                model.GalaxyRegistryEntryStatusActive,
		ValidationStatus:      model.GalaxyRegistryValidationPending,
		ExpectedEndAt:         time.Now().Add(1 * time.Hour),
		ActualStartAt:         time.Now(),
		FrozenMinBountyAmount: 10000000,
	}).Error; err != nil {
		t.Fatalf("create entry: %v", err)
	}

	svc := NewGalaxyRegistryService()
	err := svc.DeleteAdminSystem(1)
	if err == nil {
		t.Fatal("expected active-entry rejection")
	}
	if !IsUserVisibleError(err) || err.Error() != "该星系存在进行中的登记，不能删除" {
		t.Fatalf("unexpected error: %v", err)
	}
}

package service

import (
	"amiya-eden/global"
	"amiya-eden/internal/model"
	"amiya-eden/pkg/eve/esi"
	"testing"
	"time"

	"gorm.io/gorm"
)

type fakeGalaxyRegistryWalletExecutor struct {
	execute func(ctx *esi.TaskContext) error
}

func (f fakeGalaxyRegistryWalletExecutor) Execute(ctx *esi.TaskContext) error {
	if f.execute != nil {
		return f.execute(ctx)
	}
	return nil
}

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

func TestGalaxyRegistryEndEntryValidatesImmediately(t *testing.T) {
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
		AccessToken:   "token",
		TokenExpiry:   time.Now().Add(2 * time.Hour),
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
	svc.wallet = fakeGalaxyRegistryWalletExecutor{}
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

	if err := db.Create(&model.EVECharacterWalletJournal{
		ID:            8101,
		CharacterID:   90000002,
		Amount:        15000000,
		Balance:       15000000,
		ContextID:     30002187,
		ContextIDType: "solar_system_id",
		Date:          time.Now(),
		Description:   "valid bounty",
		FirstPartyID:  1,
		RefType:       "bounty_prizes",
		SecondPartyID: 2,
	}).Error; err != nil {
		t.Fatalf("create journal: %v", err)
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
	if ended.ValidationStatus != model.GalaxyRegistryValidationValid {
		t.Fatalf("validation_status = %q, want %q", ended.ValidationStatus, model.GalaxyRegistryValidationValid)
	}
	if ended.ValidatedBountyAmount != 15000000 {
		t.Fatalf("validated_bounty_amount = %v, want 15000000", ended.ValidatedBountyAmount)
	}
}

func TestGalaxyRegistryEndEntryMarksViolationImmediately(t *testing.T) {
	db := newGalaxyRegistryServiceTestDB(t)
	previous := global.DB
	global.DB = db
	t.Cleanup(func() { global.DB = previous })

	if err := db.Create(&model.User{
		BaseModel:          model.BaseModel{ID: 12},
		Nickname:           "captain-twelve",
		PrimaryCharacterID: 90000012,
		Role:               model.RoleCaptain,
	}).Error; err != nil {
		t.Fatalf("create user: %v", err)
	}
	if err := db.Create(&model.EveCharacter{
		CharacterID:   90000012,
		CharacterName: "Captain Alt",
		UserID:        12,
		AccessToken:   "token",
		TokenExpiry:   time.Now().Add(2 * time.Hour),
	}).Error; err != nil {
		t.Fatalf("create character: %v", err)
	}
	if err := db.Create(&model.GalaxyRegistrySystem{
		SolarSystemID:     30004759,
		SolarSystemName:   "1DQ1-A",
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
	svc.wallet = fakeGalaxyRegistryWalletExecutor{}
	entry, err := svc.StartEntry(12, GalaxyRegistryCreateEntryRequest{
		SystemConfigID: 1,
		ExpectedEndAt:  time.Now().Add(90 * time.Minute),
	})
	if err != nil {
		t.Fatalf("StartEntry() error = %v", err)
	}

	if err := db.Create(&model.EVECharacterWalletJournal{
		ID:            8102,
		CharacterID:   90000012,
		Amount:        5000000,
		Balance:       5000000,
		ContextID:     30004759,
		ContextIDType: "solar_system_id",
		Date:          time.Now(),
		Description:   "partial bounty",
		FirstPartyID:  1,
		RefType:       "bounty_prizes",
		SecondPartyID: 2,
	}).Error; err != nil {
		t.Fatalf("create journal: %v", err)
	}

	ended, err := svc.EndMyEntry(12, entry.ID)
	if err != nil {
		t.Fatalf("EndMyEntry() error = %v", err)
	}
	if ended.ValidationStatus != model.GalaxyRegistryValidationViolation {
		t.Fatalf("validation_status = %q, want %q", ended.ValidationStatus, model.GalaxyRegistryValidationViolation)
	}
	if ended.ViolationReason != model.GalaxyRegistryViolationBountyBelowThreshold {
		t.Fatalf("violation_reason = %q, want %q", ended.ViolationReason, model.GalaxyRegistryViolationBountyBelowThreshold)
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

func TestGalaxyRegistryUpdateMyExpectedEndAt(t *testing.T) {
	db := newGalaxyRegistryServiceTestDB(t)
	previous := global.DB
	global.DB = db
	t.Cleanup(func() { global.DB = previous })

	if err := db.Create(&model.User{
		BaseModel:          model.BaseModel{ID: 21},
		Nickname:           "captain-twenty-one",
		PrimaryCharacterID: 92000021,
		Role:               model.RoleCaptain,
	}).Error; err != nil {
		t.Fatalf("create user: %v", err)
	}
	if err := db.Create(&model.EveCharacter{
		CharacterID:   92000021,
		CharacterName: "Captain 21",
		UserID:        21,
	}).Error; err != nil {
		t.Fatalf("create character: %v", err)
	}
	startAt := time.Now().Add(-30 * time.Minute)
	entry := &model.GalaxyRegistryEntry{
		SystemConfigID:        1,
		SolarSystemID:         30000142,
		SolarSystemName:       "Jita",
		CaptainUserID:         21,
		CaptainCharacterID:    92000021,
		CaptainCharacterName:  "Captain 21",
		Status:                model.GalaxyRegistryEntryStatusActive,
		ValidationStatus:      model.GalaxyRegistryValidationPending,
		ExpectedEndAt:         startAt.Add(1 * time.Hour),
		ActualStartAt:         startAt,
		FrozenMinBountyAmount: 10000000,
	}
	if err := db.Create(entry).Error; err != nil {
		t.Fatalf("create entry: %v", err)
	}

	svc := NewGalaxyRegistryService()
	newExpectedEndAt := startAt.Add(2 * time.Hour)
	updated, err := svc.UpdateMyExpectedEndAt(21, entry.ID, GalaxyRegistryUpdateExpectedEndAtRequest{
		ExpectedEndAt: newExpectedEndAt,
	})
	if err != nil {
		t.Fatalf("UpdateMyExpectedEndAt() error = %v", err)
	}
	if updated.ExpectedEndAt.IsZero() {
		t.Fatal("expected_end_at should be set")
	}

	_, err = svc.UpdateMyExpectedEndAt(22, entry.ID, GalaxyRegistryUpdateExpectedEndAtRequest{
		ExpectedEndAt: newExpectedEndAt.Add(1 * time.Hour),
	})
	if err == nil || !IsUserVisibleError(err) || err.Error() != "只能修改自己的登记" {
		t.Fatalf("unexpected non-owner error: %v", err)
	}

	if err := db.Model(&model.GalaxyRegistryEntry{}).
		Where("id = ?", entry.ID).
		Update("status", model.GalaxyRegistryEntryStatusCompleted).Error; err != nil {
		t.Fatalf("mark completed: %v", err)
	}

	_, err = svc.UpdateMyExpectedEndAt(21, entry.ID, GalaxyRegistryUpdateExpectedEndAtRequest{
		ExpectedEndAt: newExpectedEndAt.Add(2 * time.Hour),
	})
	if err == nil || !IsUserVisibleError(err) || err.Error() != "只能修改进行中的登记" {
		t.Fatalf("unexpected completed-entry error: %v", err)
	}
}

func TestGalaxyRegistryOverrideEntryValidation(t *testing.T) {
	db := newGalaxyRegistryServiceTestDB(t)
	previous := global.DB
	global.DB = db
	t.Cleanup(func() { global.DB = previous })

	if err := db.Create(&model.User{
		BaseModel: model.BaseModel{ID: 31},
		Nickname:  "captain-thirty-one",
		Role:      model.RoleCaptain,
	}).Error; err != nil {
		t.Fatalf("create user: %v", err)
	}
	endAt := time.Now().Add(-10 * time.Minute)
	entry := &model.GalaxyRegistryEntry{
		SystemConfigID:        1,
		SolarSystemID:         30004760,
		SolarSystemName:       "T5ZI-S",
		CaptainUserID:         31,
		CaptainCharacterID:    93000031,
		CaptainCharacterName:  "Captain 31",
		Status:                model.GalaxyRegistryEntryStatusCompleted,
		ValidationStatus:      model.GalaxyRegistryValidationPending,
		ExpectedEndAt:         endAt.Add(-1 * time.Hour),
		ActualStartAt:         endAt.Add(-2 * time.Hour),
		ActualEndAt:           &endAt,
		FrozenMinBountyAmount: 10000000,
	}
	if err := db.Create(entry).Error; err != nil {
		t.Fatalf("create entry: %v", err)
	}

	svc := NewGalaxyRegistryService()
	reason := model.GalaxyRegistryViolationNoBountyInWindow
	updated, err := svc.OverrideEntryValidation(entry.ID, model.GalaxyRegistryValidationViolation, &reason)
	if err != nil {
		t.Fatalf("OverrideEntryValidation() error = %v", err)
	}
	if updated.ValidationStatus != model.GalaxyRegistryValidationViolation {
		t.Fatalf("validation_status = %q, want %q", updated.ValidationStatus, model.GalaxyRegistryValidationViolation)
	}
	if updated.ViolationReason != model.GalaxyRegistryViolationNoBountyInWindow {
		t.Fatalf("violation_reason = %q, want %q", updated.ViolationReason, model.GalaxyRegistryViolationNoBountyInWindow)
	}
	if updated.ValidatedAt == nil {
		t.Fatal("validated_at should be set")
	}
}

func TestGalaxyRegistryRevalidateEntryWithContext(t *testing.T) {
	db := newGalaxyRegistryServiceTestDB(t)
	previous := global.DB
	global.DB = db
	t.Cleanup(func() { global.DB = previous })

	if err := db.Create(&model.User{
		BaseModel: model.BaseModel{ID: 41},
		Nickname:  "captain-forty-one",
		Role:      model.RoleCaptain,
	}).Error; err != nil {
		t.Fatalf("create user: %v", err)
	}
	if err := db.Create(&model.EveCharacter{
		CharacterID:   94000041,
		CharacterName: "Captain 41",
		UserID:        41,
		AccessToken:   "token",
		TokenExpiry:   time.Now().Add(2 * time.Hour),
	}).Error; err != nil {
		t.Fatalf("create character: %v", err)
	}
	endAt := time.Now().Add(-10 * time.Minute)
	entry := &model.GalaxyRegistryEntry{
		SystemConfigID:        1,
		SolarSystemID:         30004761,
		SolarSystemName:       "ZXIC-7",
		CaptainUserID:         41,
		CaptainCharacterID:    94000041,
		CaptainCharacterName:  "Captain 41",
		Status:                model.GalaxyRegistryEntryStatusCompleted,
		ValidationStatus:      model.GalaxyRegistryValidationViolation,
		ExpectedEndAt:         endAt.Add(-1 * time.Hour),
		ActualStartAt:         endAt.Add(-2 * time.Hour),
		ActualEndAt:           &endAt,
		FrozenMinBountyAmount: 10000000,
		ViolationReason:       model.GalaxyRegistryViolationNoBountyInWindow,
	}
	if err := db.Create(entry).Error; err != nil {
		t.Fatalf("create entry: %v", err)
	}
	if err := db.Create(&model.EVECharacterWalletJournal{
		ID:            8201,
		CharacterID:   94000041,
		Amount:        15000000,
		Balance:       15000000,
		ContextID:     30004761,
		ContextIDType: "solar_system_id",
		Date:          endAt.Add(-30 * time.Minute),
		Description:   "valid bounty",
		FirstPartyID:  1,
		RefType:       "bounty_prizes",
		SecondPartyID: 2,
	}).Error; err != nil {
		t.Fatalf("create journal: %v", err)
	}

	svc := NewGalaxyRegistryService()
	svc.wallet = fakeGalaxyRegistryWalletExecutor{}
	row, err := svc.RevalidateEntryWithContext(t.Context(), entry.ID)
	if err != nil {
		t.Fatalf("RevalidateEntryWithContext() error = %v", err)
	}
	if row.ValidationStatus != model.GalaxyRegistryValidationValid {
		t.Fatalf("validation_status = %q, want %q", row.ValidationStatus, model.GalaxyRegistryValidationValid)
	}
	if row.ValidatedBountyAmount != 15000000 {
		t.Fatalf("validated_bounty_amount = %v, want 15000000", row.ValidatedBountyAmount)
	}
	if row.ViolationReason != "" {
		t.Fatalf("violation_reason = %q, want empty", row.ViolationReason)
	}
}

func TestGalaxyRegistryRevalidateEntryWithContextRejectsActiveEntries(t *testing.T) {
	db := newGalaxyRegistryServiceTestDB(t)
	previous := global.DB
	global.DB = db
	t.Cleanup(func() { global.DB = previous })

	entry := &model.GalaxyRegistryEntry{
		SystemConfigID:        1,
		SolarSystemID:         30000142,
		SolarSystemName:       "Jita",
		CaptainUserID:         51,
		CaptainCharacterID:    95000051,
		CaptainCharacterName:  "Captain 51",
		Status:                model.GalaxyRegistryEntryStatusActive,
		ValidationStatus:      model.GalaxyRegistryValidationPending,
		ExpectedEndAt:         time.Now().Add(1 * time.Hour),
		ActualStartAt:         time.Now().Add(-1 * time.Hour),
		FrozenMinBountyAmount: 10000000,
	}
	if err := db.Create(entry).Error; err != nil {
		t.Fatalf("create entry: %v", err)
	}

	svc := NewGalaxyRegistryService()
	_, err := svc.RevalidateEntryWithContext(t.Context(), entry.ID)
	if err == nil || !IsUserVisibleError(err) || err.Error() != "只能重新校验已结束的登记" {
		t.Fatalf("unexpected error: %v", err)
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

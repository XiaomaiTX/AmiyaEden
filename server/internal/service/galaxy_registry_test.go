package service

import (
	"amiya-eden/global"
	"amiya-eden/internal/model"
	"context"
	"errors"
	"testing"
	"time"

	"go.uber.org/zap"
	"gorm.io/gorm"
)

type fakeGalaxyRegistryWalletTaskRunner struct {
	runTask func(ctx context.Context, taskName string, characterID int64) error
	calls   []int64
}

func (f *fakeGalaxyRegistryWalletTaskRunner) RunTask(ctx context.Context, taskName string, characterID int64) error {
	f.calls = append(f.calls, characterID)
	if f.runTask != nil {
		return f.runTask(ctx, taskName, characterID)
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

func TestGalaxyRegistryStartEntryRejectsExpectedEndOverMaxDuration(t *testing.T) {
	db := newGalaxyRegistryServiceTestDB(t)
	previous := global.DB
	global.DB = db
	t.Cleanup(func() { global.DB = previous })

	if err := db.Create(&model.User{
		BaseModel:          model.BaseModel{ID: 11},
		Nickname:           "captain-eleven",
		PrimaryCharacterID: 90000011,
		Role:               model.RoleCaptain,
	}).Error; err != nil {
		t.Fatalf("create user: %v", err)
	}
	if err := db.Create(&model.EveCharacter{
		CharacterID:   90000011,
		CharacterName: "Captain Eleven",
		UserID:        11,
	}).Error; err != nil {
		t.Fatalf("create character: %v", err)
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
	_, err := svc.StartEntry(11, GalaxyRegistryCreateEntryRequest{
		SystemConfigID: 1,
		ExpectedEndAt:  time.Now().Add(GalaxyRegistryMaxEntryDuration + time.Minute),
	})
	if err == nil || !IsUserVisibleError(err) || err.Error() != "单次登记最长不能超过 2 小时" {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestGalaxyRegistryEndEntryLeavesPendingThenSettlementMarksValid(t *testing.T) {
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
	if ended.ValidationStatus != model.GalaxyRegistryValidationPending {
		t.Fatalf("validation_status after end = %q, want %q", ended.ValidationStatus, model.GalaxyRegistryValidationPending)
	}

	runner := &fakeGalaxyRegistryWalletTaskRunner{}
	settled, err := svc.SettlePendingEntriesWithWalletRefresh(t.Context(), runner, 10)
	if err != nil {
		t.Fatalf("SettlePendingEntriesWithWalletRefresh() error = %v", err)
	}
	if settled != 1 {
		t.Fatalf("settled count = %d, want 1", settled)
	}
	reloaded, err := svc.repo.GetEntryByID(entry.ID)
	if err != nil {
		t.Fatalf("reload entry: %v", err)
	}
	if reloaded.ValidationStatus != model.GalaxyRegistryValidationValid {
		t.Fatalf("validation_status = %q, want %q", reloaded.ValidationStatus, model.GalaxyRegistryValidationValid)
	}
	if reloaded.ValidatedBountyAmount != 15000000 {
		t.Fatalf("validated_bounty_amount = %v, want 15000000", reloaded.ValidatedBountyAmount)
	}
}

func TestGalaxyRegistrySettlementMarksViolation(t *testing.T) {
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

	_, err = svc.EndMyEntry(12, entry.ID)
	if err != nil {
		t.Fatalf("EndMyEntry() error = %v", err)
	}

	runner := &fakeGalaxyRegistryWalletTaskRunner{}
	settled, err := svc.SettlePendingEntriesWithWalletRefresh(t.Context(), runner, 10)
	if err != nil {
		t.Fatalf("SettlePendingEntriesWithWalletRefresh() error = %v", err)
	}
	if settled != 1 {
		t.Fatalf("settled count = %d, want 1", settled)
	}
	reloaded, err := svc.repo.GetEntryByID(entry.ID)
	if err != nil {
		t.Fatalf("reload entry: %v", err)
	}
	if reloaded.ValidationStatus != model.GalaxyRegistryValidationViolation {
		t.Fatalf("validation_status = %q, want %q", reloaded.ValidationStatus, model.GalaxyRegistryValidationViolation)
	}
	if reloaded.ViolationReason != model.GalaxyRegistryViolationBountyBelowThreshold {
		t.Fatalf("violation_reason = %q, want %q", reloaded.ViolationReason, model.GalaxyRegistryViolationBountyBelowThreshold)
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

func TestGalaxyRegistryUpdateMyExpectedEndAtRejectsOverMaxDuration(t *testing.T) {
	db := newGalaxyRegistryServiceTestDB(t)
	previous := global.DB
	global.DB = db
	t.Cleanup(func() { global.DB = previous })

	startAt := time.Now().Add(-30 * time.Minute)
	entry := &model.GalaxyRegistryEntry{
		SystemConfigID:        1,
		SolarSystemID:         30000142,
		SolarSystemName:       "Jita",
		CaptainUserID:         22,
		CaptainCharacterID:    92000022,
		CaptainCharacterName:  "Captain 22",
		Status:                model.GalaxyRegistryEntryStatusActive,
		ValidationStatus:      model.GalaxyRegistryValidationPending,
		ExpectedEndAt:         startAt.Add(time.Hour),
		ActualStartAt:         startAt,
		FrozenMinBountyAmount: 10000000,
	}
	if err := db.Create(entry).Error; err != nil {
		t.Fatalf("create entry: %v", err)
	}

	svc := NewGalaxyRegistryService()
	_, err := svc.UpdateMyExpectedEndAt(22, entry.ID, GalaxyRegistryUpdateExpectedEndAtRequest{
		ExpectedEndAt: startAt.Add(GalaxyRegistryMaxEntryDuration + time.Minute),
	})
	if err == nil || !IsUserVisibleError(err) || err.Error() != "单次登记最长不能超过 2 小时" {
		t.Fatalf("unexpected error: %v", err)
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

func TestGalaxyRegistryRevalidateEntryWithContextResetsPendingThenSettlementMarksValid(t *testing.T) {
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
	row, err := svc.RevalidateEntryWithContext(t.Context(), entry.ID)
	if err != nil {
		t.Fatalf("RevalidateEntryWithContext() error = %v", err)
	}
	if row.ValidationStatus != model.GalaxyRegistryValidationPending {
		t.Fatalf("validation_status after reset = %q, want %q", row.ValidationStatus, model.GalaxyRegistryValidationPending)
	}
	if row.ValidatedBountyAmount != 0 {
		t.Fatalf("validated_bounty_amount after reset = %v, want 0", row.ValidatedBountyAmount)
	}

	runner := &fakeGalaxyRegistryWalletTaskRunner{}
	settled, err := svc.SettlePendingEntriesWithWalletRefresh(t.Context(), runner, 10)
	if err != nil {
		t.Fatalf("SettlePendingEntriesWithWalletRefresh() error = %v", err)
	}
	if settled != 1 {
		t.Fatalf("settled count = %d, want 1", settled)
	}
	reloaded, err := svc.repo.GetEntryByID(entry.ID)
	if err != nil {
		t.Fatalf("reload entry: %v", err)
	}
	if reloaded.ValidationStatus != model.GalaxyRegistryValidationValid {
		t.Fatalf("validation_status = %q, want %q", reloaded.ValidationStatus, model.GalaxyRegistryValidationValid)
	}
	if reloaded.ValidatedBountyAmount != 15000000 {
		t.Fatalf("validated_bounty_amount = %v, want 15000000", reloaded.ValidatedBountyAmount)
	}
	if reloaded.ViolationReason != "" {
		t.Fatalf("violation_reason = %q, want empty", reloaded.ViolationReason)
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

func TestGalaxyRegistryAutoEndOverdueEntriesUsesMaxDurationEndAt(t *testing.T) {
	db := newGalaxyRegistryServiceTestDB(t)
	previous := global.DB
	global.DB = db
	t.Cleanup(func() { global.DB = previous })

	startAt := time.Now().Add(-3 * time.Hour)
	if err := db.Create(&model.User{
		BaseModel: model.BaseModel{ID: 61},
		Nickname:  "captain-sixty-one",
		Role:      model.RoleCaptain,
	}).Error; err != nil {
		t.Fatalf("create user: %v", err)
	}
	entry := &model.GalaxyRegistryEntry{
		SystemConfigID:        1,
		SolarSystemID:         30000142,
		SolarSystemName:       "Jita",
		CaptainUserID:         61,
		CaptainCharacterID:    96000061,
		CaptainCharacterName:  "Captain 61",
		Status:                model.GalaxyRegistryEntryStatusActive,
		ValidationStatus:      model.GalaxyRegistryValidationPending,
		ExpectedEndAt:         startAt.Add(GalaxyRegistryMaxEntryDuration),
		ActualStartAt:         startAt,
		FrozenMinBountyAmount: 10000000,
	}
	if err := db.Create(entry).Error; err != nil {
		t.Fatalf("create entry: %v", err)
	}

	svc := NewGalaxyRegistryService()
	ended, err := svc.AutoEndOverdueEntries(10)
	if err != nil {
		t.Fatalf("AutoEndOverdueEntries() error = %v", err)
	}
	if ended != 1 {
		t.Fatalf("ended count = %d, want 1", ended)
	}
	reloaded, err := svc.repo.GetEntryByID(entry.ID)
	if err != nil {
		t.Fatalf("reload entry: %v", err)
	}
	if reloaded.Status != model.GalaxyRegistryEntryStatusCompleted {
		t.Fatalf("status = %q, want %q", reloaded.Status, model.GalaxyRegistryEntryStatusCompleted)
	}
	if reloaded.ActualEndAt == nil {
		t.Fatal("actual_end_at should be set")
	}
	if !reloaded.ActualEndAt.Equal(startAt.Add(GalaxyRegistryMaxEntryDuration)) {
		t.Fatalf("actual_end_at = %s, want %s", reloaded.ActualEndAt, startAt.Add(GalaxyRegistryMaxEntryDuration))
	}
	if reloaded.EndedByUserID != 0 {
		t.Fatalf("ended_by_user_id = %d, want 0", reloaded.EndedByUserID)
	}
	if reloaded.ForceEndedByAdmin {
		t.Fatal("force_ended_by_admin should be false for system auto-end")
	}
}

// newGalaxyRegistryOverwriteTestDB seeds an enabled system and a first captain
// (with a bound character) that holds the active entry described by startAt /
// expectedEndAt. The second captain is also created with a bound character.
func newGalaxyRegistryOverwriteTestDB(t *testing.T, startAt, expectedEndAt time.Time) (*gorm.DB, *model.GalaxyRegistryEntry) {
	t.Helper()
	db := newGalaxyRegistryServiceTestDB(t)

	system := &model.GalaxyRegistrySystem{
		SolarSystemID:     30002187,
		SolarSystemName:   "Amamake",
		RegionID:          10000054,
		RegionName:        "Heimatar",
		ConstellationID:   20000605,
		ConstellationName: "Hed-Belt",
		Security:          -0.4,
		MinBountyAmount:   model.GalaxyRegistryDefaultMinBountyAmount,
		IsEnabled:         true,
	}
	if err := db.Create(system).Error; err != nil {
		t.Fatalf("create system: %v", err)
	}
	if err := db.Create(&model.User{
		BaseModel:          model.BaseModel{ID: 70},
		Nickname:           "captain-seventy",
		PrimaryCharacterID: 96000070,
		Role:               model.RoleCaptain,
	}).Error; err != nil {
		t.Fatalf("create first captain: %v", err)
	}
	if err := db.Create(&model.EveCharacter{
		CharacterID:   96000070,
		CharacterName: "Captain Seventy",
		UserID:        70,
	}).Error; err != nil {
		t.Fatalf("create first character: %v", err)
	}
	if err := db.Create(&model.User{
		BaseModel:          model.BaseModel{ID: 71},
		Nickname:           "captain-seventy-one",
		PrimaryCharacterID: 96000071,
		Role:               model.RoleCaptain,
	}).Error; err != nil {
		t.Fatalf("create second captain: %v", err)
	}
	if err := db.Create(&model.EveCharacter{
		CharacterID:   96000071,
		CharacterName: "Captain Seventy-One",
		UserID:        71,
	}).Error; err != nil {
		t.Fatalf("create second character: %v", err)
	}
	existing := &model.GalaxyRegistryEntry{
		SystemConfigID:        system.ID,
		SolarSystemID:         system.SolarSystemID,
		SolarSystemName:       system.SolarSystemName,
		CaptainUserID:         70,
		CaptainCharacterID:    96000070,
		CaptainCharacterName:  "Captain Seventy",
		Status:                model.GalaxyRegistryEntryStatusActive,
		ValidationStatus:      model.GalaxyRegistryValidationPending,
		ExpectedEndAt:         expectedEndAt,
		ActualStartAt:         startAt,
		FrozenMinBountyAmount: system.MinBountyAmount,
	}
	if err := db.Create(existing).Error; err != nil {
		t.Fatalf("create existing entry: %v", err)
	}
	return db, existing
}

func TestGalaxyRegistryStartEntryOverwritesOverdueEntryByExpectedEnd(t *testing.T) {
	now := time.Now()
	startAt := now.Add(-90 * time.Minute)
	expectedEndAt := now.Add(-30 * time.Minute) // expected end already passed
	db, existing := newGalaxyRegistryOverwriteTestDB(t, startAt, expectedEndAt)
	previous := global.DB
	global.DB = db
	t.Cleanup(func() { global.DB = previous })

	svc := NewGalaxyRegistryService()
	newEntry, err := svc.StartEntry(71, GalaxyRegistryCreateEntryRequest{
		SystemConfigID: existing.SystemConfigID,
		ExpectedEndAt:  now.Add(30 * time.Minute),
	})
	if err != nil {
		t.Fatalf("StartEntry() overwrite error = %v", err)
	}
	if newEntry.CaptainUserID != 71 {
		t.Fatalf("new entry captain = %d, want 71", newEntry.CaptainUserID)
	}
	if newEntry.Status != model.GalaxyRegistryEntryStatusActive {
		t.Fatalf("new entry status = %q, want active", newEntry.Status)
	}

	// The old entry must be completed and credited to the overwriting captain.
	reloaded, err := svc.repo.GetEntryByID(existing.ID)
	if err != nil {
		t.Fatalf("reload old entry: %v", err)
	}
	if reloaded.Status != model.GalaxyRegistryEntryStatusCompleted {
		t.Fatalf("old entry status = %q, want completed", reloaded.Status)
	}
	if reloaded.ActualEndAt == nil {
		t.Fatal("old entry actual_end_at should be set")
	}
	if reloaded.EndedByUserID != 71 {
		t.Fatalf("old entry ended_by_user_id = %d, want 71", reloaded.EndedByUserID)
	}
}

func TestGalaxyRegistryStartEntryOverwritesOverdueEntryByHardCap(t *testing.T) {
	// expected_end_at is still in the future, but actual_start_at + 2h passed.
	now := time.Now()
	startAt := now.Add(-(GalaxyRegistryMaxEntryDuration + time.Minute))
	expectedEndAt := now.Add(30 * time.Minute)
	db, existing := newGalaxyRegistryOverwriteTestDB(t, startAt, expectedEndAt)
	previous := global.DB
	global.DB = db
	t.Cleanup(func() { global.DB = previous })

	svc := NewGalaxyRegistryService()
	newEntry, err := svc.StartEntry(71, GalaxyRegistryCreateEntryRequest{
		SystemConfigID: existing.SystemConfigID,
		ExpectedEndAt:  now.Add(30 * time.Minute),
	})
	if err != nil {
		t.Fatalf("StartEntry() overwrite error = %v", err)
	}
	if newEntry.CaptainUserID != 71 {
		t.Fatalf("new entry captain = %d, want 71", newEntry.CaptainUserID)
	}
	reloaded, err := svc.repo.GetEntryByID(existing.ID)
	if err != nil {
		t.Fatalf("reload old entry: %v", err)
	}
	if reloaded.Status != model.GalaxyRegistryEntryStatusCompleted {
		t.Fatalf("old entry status = %q, want completed", reloaded.Status)
	}
}

func TestGalaxyRegistryStartEntryRejectsOverwritingOwnOverdueEntry(t *testing.T) {
	now := time.Now()
	startAt := now.Add(-90 * time.Minute)
	expectedEndAt := now.Add(-30 * time.Minute)
	db, existing := newGalaxyRegistryOverwriteTestDB(t, startAt, expectedEndAt)
	previous := global.DB
	global.DB = db
	t.Cleanup(func() { global.DB = previous })

	svc := NewGalaxyRegistryService()
	_, err := svc.StartEntry(70, GalaxyRegistryCreateEntryRequest{ // same captain as existing
		SystemConfigID: existing.SystemConfigID,
		ExpectedEndAt:  now.Add(30 * time.Minute),
	})
	if err == nil || !IsUserVisibleError(err) || err.Error() != "请先结束自己的超时登记" {
		t.Fatalf("unexpected error: %v", err)
	}

	// The original entry must remain active (nothing changed).
	reloaded, err := svc.repo.GetEntryByID(existing.ID)
	if err != nil {
		t.Fatalf("reload old entry: %v", err)
	}
	if reloaded.Status != model.GalaxyRegistryEntryStatusActive {
		t.Fatalf("old entry status = %q, want still active", reloaded.Status)
	}
}

func TestGalaxyRegistryStartEntryStillRejectsNonOverdueActiveEntry(t *testing.T) {
	now := time.Now()
	startAt := now.Add(-10 * time.Minute)
	expectedEndAt := now.Add(80 * time.Minute)
	db, existing := newGalaxyRegistryOverwriteTestDB(t, startAt, expectedEndAt)
	previous := global.DB
	global.DB = db
	t.Cleanup(func() { global.DB = previous })

	svc := NewGalaxyRegistryService()
	_, err := svc.StartEntry(71, GalaxyRegistryCreateEntryRequest{
		SystemConfigID: existing.SystemConfigID,
		ExpectedEndAt:  now.Add(30 * time.Minute),
	})
	if err == nil || !IsUserVisibleError(err) || err.Error() != "该星系当前已有生产登记" {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestGalaxyRegistrySettlementContinuesWhenOneWalletRefreshFails(t *testing.T) {
	db := newGalaxyRegistryServiceTestDB(t)
	previous := global.DB
	global.DB = db
	global.Logger = zap.NewNop()
	t.Cleanup(func() { global.DB = previous; global.Logger = nil })

	if err := db.Create(&model.User{
		BaseModel:          model.BaseModel{ID: 101},
		Nickname:           "captain-token-skip",
		PrimaryCharacterID: 90000101,
		Role:               model.RoleCaptain,
	}).Error; err != nil {
		t.Fatalf("create user: %v", err)
	}
	if err := db.Create(&model.EveCharacter{
		CharacterID:   90000101,
		CharacterName: "Char Invalid Token",
		UserID:        101,
		AccessToken:   "expired",
		TokenExpiry:   time.Now().Add(-1 * time.Hour),
		TokenInvalid:  true,
	}).Error; err != nil {
		t.Fatalf("create character A: %v", err)
	}
	if err := db.Create(&model.EveCharacter{
		CharacterID:   90000102,
		CharacterName: "Char Valid Token",
		UserID:        101,
		AccessToken:   "valid-token",
		TokenExpiry:   time.Now().Add(2 * time.Hour),
	}).Error; err != nil {
		t.Fatalf("create character B: %v", err)
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
	entry, err := svc.StartEntry(101, GalaxyRegistryCreateEntryRequest{
		SystemConfigID: 1,
		ExpectedEndAt:  time.Now().Add(90 * time.Minute),
	})
	if err != nil {
		t.Fatalf("StartEntry() error = %v", err)
	}

	if err := db.Create(&model.EVECharacterWalletJournal{
		ID:            9101,
		CharacterID:   90000102,
		Amount:        15000000,
		Balance:       15000000,
		ContextID:     30002187,
		ContextIDType: "solar_system_id",
		Date:          time.Now(),
		Description:   "valid bounty from char B",
		FirstPartyID:  1,
		RefType:       "bounty_prizes",
		SecondPartyID: 2,
	}).Error; err != nil {
		t.Fatalf("create journal: %v", err)
	}

	ended, err := svc.EndMyEntry(101, entry.ID)
	if err != nil {
		t.Fatalf("EndMyEntry() error = %v", err)
	}
	if ended.Status != model.GalaxyRegistryEntryStatusCompleted {
		t.Fatalf("status = %q, want %q", ended.Status, model.GalaxyRegistryEntryStatusCompleted)
	}
	if ended.ValidationStatus != model.GalaxyRegistryValidationPending {
		t.Fatalf("validation_status after end = %q, want %q", ended.ValidationStatus, model.GalaxyRegistryValidationPending)
	}

	runner := &fakeGalaxyRegistryWalletTaskRunner{
		runTask: func(_ context.Context, _ string, characterID int64) error {
			if characterID == 90000101 {
				return errors.New("invalid_token")
			}
			return nil
		},
	}
	settled, err := svc.SettlePendingEntriesWithWalletRefresh(t.Context(), runner, 10)
	if err != nil {
		t.Fatalf("SettlePendingEntriesWithWalletRefresh() error = %v", err)
	}
	if settled != 1 {
		t.Fatalf("settled count = %d, want 1", settled)
	}
	reloaded, err := svc.repo.GetEntryByID(entry.ID)
	if err != nil {
		t.Fatalf("reload entry: %v", err)
	}
	if reloaded.ValidationStatus != model.GalaxyRegistryValidationValid {
		t.Fatalf("validation_status = %q, want %q", reloaded.ValidationStatus, model.GalaxyRegistryValidationValid)
	}
	if reloaded.ValidatedBountyAmount != 15000000 {
		t.Fatalf("validated_bounty_amount = %v, want 15000000", reloaded.ValidatedBountyAmount)
	}
}

func TestGalaxyRegistrySettlementKeepsPendingWhenAllWalletRefreshesFail(t *testing.T) {
	db := newGalaxyRegistryServiceTestDB(t)
	previous := global.DB
	global.DB = db
	global.Logger = zap.NewNop()
	t.Cleanup(func() { global.DB = previous; global.Logger = nil })

	if err := db.Create(&model.User{
		BaseModel:          model.BaseModel{ID: 103},
		Nickname:           "captain-all-invalid",
		PrimaryCharacterID: 90000105,
		Role:               model.RoleCaptain,
	}).Error; err != nil {
		t.Fatalf("create user: %v", err)
	}
	if err := db.Create(&model.EveCharacter{
		CharacterID:   90000105,
		CharacterName: "Char A Invalid",
		UserID:        103,
		AccessToken:   "expired",
		TokenExpiry:   time.Now().Add(-1 * time.Hour),
		TokenInvalid:  true,
	}).Error; err != nil {
		t.Fatalf("create character A: %v", err)
	}
	if err := db.Create(&model.EveCharacter{
		CharacterID:   90000106,
		CharacterName: "Char B Invalid",
		UserID:        103,
		AccessToken:   "expired",
		TokenExpiry:   time.Now().Add(-1 * time.Hour),
		TokenInvalid:  true,
	}).Error; err != nil {
		t.Fatalf("create character B: %v", err)
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
	entry, err := svc.StartEntry(103, GalaxyRegistryCreateEntryRequest{
		SystemConfigID: 1,
		ExpectedEndAt:  time.Now().Add(90 * time.Minute),
	})
	if err != nil {
		t.Fatalf("StartEntry() error = %v", err)
	}

	_, err = svc.EndMyEntry(103, entry.ID)
	if err != nil {
		t.Fatalf("EndMyEntry() error = %v", err)
	}

	runner := &fakeGalaxyRegistryWalletTaskRunner{
		runTask: func(_ context.Context, _ string, _ int64) error {
			return errors.New("database timeout")
		},
	}
	settled, err := svc.SettlePendingEntriesWithWalletRefresh(t.Context(), runner, 10)
	if err != nil {
		t.Fatalf("SettlePendingEntriesWithWalletRefresh() error = %v", err)
	}
	if settled != 0 {
		t.Fatalf("settled count = %d, want 0", settled)
	}

	reloaded, reloadErr := svc.repo.GetEntryByID(entry.ID)
	if reloadErr != nil {
		t.Fatalf("reload entry: %v", reloadErr)
	}
	if reloaded.Status != model.GalaxyRegistryEntryStatusCompleted {
		t.Fatalf("status = %q, want %q", reloaded.Status, model.GalaxyRegistryEntryStatusCompleted)
	}
	if reloaded.ValidationStatus != model.GalaxyRegistryValidationPending {
		t.Fatalf("validation_status = %q, want %q", reloaded.ValidationStatus, model.GalaxyRegistryValidationPending)
	}
}

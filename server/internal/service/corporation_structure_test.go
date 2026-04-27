package service

import (
	"amiya-eden/global"
	"amiya-eden/internal/model"
	"amiya-eden/internal/repository"
	"amiya-eden/internal/utils"
	"amiya-eden/pkg/eve/esi"
	"context"
	"fmt"
	"testing"
	"time"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestDeduplicateManagedCorporationIDs(t *testing.T) {
	chars := []model.EveCharacter{
		{CharacterID: 1, CorporationID: 100},
		{CharacterID: 2, CorporationID: 200},
		{CharacterID: 3, CorporationID: 100},
		{CharacterID: 4, CorporationID: 0},
		{CharacterID: 5, CorporationID: 300},
	}

	got := deduplicateManagedCorporationIDs(chars, []int64{100, 300, 400})
	want := []int64{100, 300}

	if len(got) != len(want) {
		t.Fatalf("len = %d, want %d", len(got), len(want))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("index %d = %d, want %d", i, got[i], want[i])
		}
	}
}

func TestValidateAuthorizationBindings(t *testing.T) {
	managed := map[int64]struct{}{100: {}, 200: {}}
	directors := map[int64]map[int64]struct{}{
		100: {10: {}, 11: {}},
		200: {20: {}},
	}

	t.Run("accepts valid bindings", func(t *testing.T) {
		err := validateAuthorizationBindings(
			[]CorporationStructureAuthorizationBinding{
				{CorporationID: 100, CharacterID: 10},
				{CorporationID: 200, CharacterID: 0},
			},
			managed,
			directors,
		)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("rejects duplicate corporation binding", func(t *testing.T) {
		err := validateAuthorizationBindings(
			[]CorporationStructureAuthorizationBinding{
				{CorporationID: 100, CharacterID: 10},
				{CorporationID: 100, CharacterID: 11},
			},
			managed,
			directors,
		)
		if err == nil {
			t.Fatal("expected duplicate corporation to be rejected")
		}
	})

	t.Run("rejects unmanaged corporation", func(t *testing.T) {
		err := validateAuthorizationBindings(
			[]CorporationStructureAuthorizationBinding{
				{CorporationID: 300, CharacterID: 10},
			},
			managed,
			directors,
		)
		if err == nil {
			t.Fatal("expected unmanaged corporation to be rejected")
		}
	})

	t.Run("rejects non director character", func(t *testing.T) {
		err := validateAuthorizationBindings(
			[]CorporationStructureAuthorizationBinding{
				{CorporationID: 200, CharacterID: 10},
			},
			managed,
			directors,
		)
		if err == nil {
			t.Fatal("expected non-director character to be rejected")
		}
	})
}

func TestCorporationStructureListUsesSnapshotFieldsAndPlaceholders(t *testing.T) {
	db := newCorporationStructureServiceTestDB(t)
	oldDB := global.DB
	global.DB = db
	utils.InvalidateAllowCorporationsCache()
	t.Cleanup(func() {
		global.DB = oldDB
		utils.InvalidateAllowCorporationsCache()
	})

	seedCorporationStructureManageScope(t, db, 9001)
	if err := db.Create(&model.CorpStructureInfo{
		CorporationID:   9001,
		CorporationName: "Snapshot Corp",
		StructureID:     111,
		Name:            "Alpha Structure",
		TypeID:          35832,
		TypeName:        "Astrahus",
		SystemID:        30000142,
		SystemName:      "Jita",
		Security:        0.9,
		State:           "shield_vulnerable",
		UpdateAt:        time.Now().Unix(),
	}).Error; err != nil {
		t.Fatalf("seed snapshot row #1: %v", err)
	}
	if err := db.Create(&model.CorpStructureInfo{
		CorporationID: 9001,
		StructureID:   222,
		Name:          "",
		TypeID:        35833,
		TypeName:      "",
		SystemID:      30002187,
		SystemName:    "",
		Security:      0,
		State:         "low_power",
		UpdateAt:      time.Now().Unix(),
	}).Error; err != nil {
		t.Fatalf("seed snapshot row #2: %v", err)
	}

	svc := newCorporationStructureServiceForTest()
	resp, err := svc.ListStructures(context.Background(), CorporationStructureListRequest{CorporationID: 9001})
	if err != nil {
		t.Fatalf("ListStructures returned error: %v", err)
	}
	if len(resp.Items) != 2 {
		t.Fatalf("expected 2 items, got %d", len(resp.Items))
	}

	byID := make(map[int64]CorporationStructureRow, len(resp.Items))
	for _, item := range resp.Items {
		byID[item.StructureID] = item
	}

	first := byID[111]
	if first.CorporationName != "Snapshot Corp" {
		t.Fatalf("expected snapshot corporation name, got %q", first.CorporationName)
	}
	if first.TypeName != "Astrahus" {
		t.Fatalf("expected snapshot type name, got %q", first.TypeName)
	}
	if first.SystemName != "Jita" {
		t.Fatalf("expected snapshot system name, got %q", first.SystemName)
	}
	if first.Security != 0.9 {
		t.Fatalf("expected snapshot security 0.9, got %v", first.Security)
	}

	second := byID[222]
	if second.CorporationName != "Corporation-9001" {
		t.Fatalf("expected fallback corporation placeholder, got %q", second.CorporationName)
	}
	if second.TypeName != "Type-35833" {
		t.Fatalf("expected fallback type placeholder, got %q", second.TypeName)
	}
	if second.SystemName != "System-30002187" {
		t.Fatalf("expected fallback system placeholder, got %q", second.SystemName)
	}
}

func TestCorporationStructureListFiltersAndSorts(t *testing.T) {
	db := newCorporationStructureServiceTestDB(t)
	oldDB := global.DB
	global.DB = db
	utils.InvalidateAllowCorporationsCache()
	t.Cleanup(func() {
		global.DB = oldDB
		utils.InvalidateAllowCorporationsCache()
	})

	seedCorporationStructureManageScope(t, db, 9001)
	now := time.Now().UTC()
	seedRows := []model.CorpStructureInfo{
		{
			CorporationID:   9001,
			CorporationName: "Snapshot Corp",
			StructureID:     1,
			Name:            "Alpha",
			TypeID:          35832,
			TypeName:        "Astrahus",
			SystemID:        30000142,
			SystemName:      "Jita",
			Security:        0.9,
			State:           "shield_vulnerable",
			Services:        `[{"name":"market","state":"online"},{"name":"industry","state":"online"}]`,
			FuelExpires:     now.Add(10 * time.Hour).Format(time.RFC3339),
			StateTimerEnd:   now.Add(90 * time.Minute).Format(time.RFC3339),
			UpdateAt:        now.Unix(),
		},
		{
			CorporationID:   9001,
			CorporationName: "Snapshot Corp",
			StructureID:     2,
			Name:            "Beta",
			TypeID:          35833,
			TypeName:        "Fortizar",
			SystemID:        30002187,
			SystemName:      "Otsasai",
			Security:        0.3,
			State:           "low_power",
			Services:        `[{"name":"market","state":"online"}]`,
			FuelExpires:     now.Add(60 * time.Hour).Format(time.RFC3339),
			StateTimerEnd:   now.Add(6 * time.Hour).Format(time.RFC3339),
			UpdateAt:        now.Unix(),
		},
		{
			CorporationID:   9001,
			CorporationName: "Snapshot Corp",
			StructureID:     3,
			Name:            "Gamma",
			TypeID:          35834,
			TypeName:        "Keepstar",
			SystemID:        30002510,
			SystemName:      "MJ-13",
			Security:        -0.1,
			State:           "abandoned",
			Services:        `[{"name":"reaction","state":"online"}]`,
			FuelExpires:     "",
			StateTimerEnd:   "",
			UpdateAt:        now.Unix(),
		},
	}
	for _, row := range seedRows {
		if err := db.Create(&row).Error; err != nil {
			t.Fatalf("seed row failed: %v", err)
		}
	}

	svc := newCorporationStructureServiceForTest()

	resp, err := svc.ListStructures(context.Background(), CorporationStructureListRequest{
		CorporationID: 9001,
		FuelBucket:    "lt_24h",
		Page:          1,
		PageSize:      20,
	})
	if err != nil {
		t.Fatalf("ListStructures fuel filter returned error: %v", err)
	}
	if resp.Total != 1 || resp.Items[0].StructureID != 1 {
		t.Fatalf("fuel lt_24h expected structure #1 only, got total=%d", resp.Total)
	}

	resp, err = svc.ListStructures(context.Background(), CorporationStructureListRequest{
		CorporationID:    9001,
		ServiceNames:     []string{"market", "industry"},
		ServiceMatchMode: "and",
		Page:             1,
		PageSize:         20,
	})
	if err != nil {
		t.Fatalf("ListStructures service and returned error: %v", err)
	}
	if resp.Total != 1 || resp.Items[0].StructureID != 1 {
		t.Fatalf("service and expected structure #1 only, got total=%d", resp.Total)
	}

	resp, err = svc.ListStructures(context.Background(), CorporationStructureListRequest{
		CorporationID:    9001,
		ServiceNames:     []string{"market", "reaction"},
		ServiceMatchMode: "or",
		Page:             1,
		PageSize:         20,
	})
	if err != nil {
		t.Fatalf("ListStructures service or returned error: %v", err)
	}
	if resp.Total != 3 {
		t.Fatalf("service or expected total 3, got %d", resp.Total)
	}

	resp, err = svc.ListStructures(context.Background(), CorporationStructureListRequest{
		CorporationID: 9001,
		TimerBucket:   "next_2_hours",
		Page:          1,
		PageSize:      20,
	})
	if err != nil {
		t.Fatalf("ListStructures timer bucket returned error: %v", err)
	}
	if resp.Total != 1 || resp.Items[0].StructureID != 1 {
		t.Fatalf("next_2_hours expected structure #1 only, got total=%d", resp.Total)
	}

	resp, err = svc.ListStructures(context.Background(), CorporationStructureListRequest{
		CorporationID: 9001,
		Page:          1,
		PageSize:      20,
	})
	if err != nil {
		t.Fatalf("ListStructures default sort returned error: %v", err)
	}
	if len(resp.Items) < 3 {
		t.Fatalf("expected 3 rows for default sort, got %d", len(resp.Items))
	}
	if resp.Items[0].StructureID != 1 || resp.Items[1].StructureID != 2 || resp.Items[2].StructureID != 3 {
		t.Fatalf("default sort expected 1,2,3 by fuel asc with nil last, got %d,%d,%d",
			resp.Items[0].StructureID, resp.Items[1].StructureID, resp.Items[2].StructureID)
	}
}

func TestCorporationStructureFilterOptions(t *testing.T) {
	db := newCorporationStructureServiceTestDB(t)
	oldDB := global.DB
	global.DB = db
	utils.InvalidateAllowCorporationsCache()
	t.Cleanup(func() {
		global.DB = oldDB
		utils.InvalidateAllowCorporationsCache()
	})

	seedCorporationStructureManageScope(t, db, 9001)
	if err := db.Create(&model.CorpStructureInfo{
		CorporationID:   9001,
		CorporationName: "Snapshot Corp",
		StructureID:     111,
		Name:            "Alpha Structure",
		TypeID:          35832,
		TypeName:        "Astrahus",
		SystemID:        30000142,
		SystemName:      "Jita",
		Security:        0.9,
		State:           "shield_vulnerable",
		Services:        `[{"name":"market","state":"online"},{"name":"industry","state":"online"}]`,
		UpdateAt:        time.Now().Unix(),
	}).Error; err != nil {
		t.Fatalf("seed snapshot row: %v", err)
	}

	svc := newCorporationStructureServiceForTest()
	resp, err := svc.GetFilterOptions(context.Background(), CorporationStructureFilterOptionsRequest{
		CorporationID: 9001,
	})
	if err != nil {
		t.Fatalf("GetFilterOptions returned error: %v", err)
	}
	if len(resp.Systems) != 1 || resp.Systems[0].SystemID != 30000142 {
		t.Fatalf("expected single system option, got %+v", resp.Systems)
	}
	if len(resp.Types) != 1 || resp.Types[0].TypeID != 35832 {
		t.Fatalf("expected single type option, got %+v", resp.Types)
	}
	if len(resp.Services) != 2 {
		t.Fatalf("expected two service options, got %+v", resp.Services)
	}
}

func TestCorporationStructureSettingsThresholds(t *testing.T) {
	db := newCorporationStructureServiceTestDB(t)
	oldDB := global.DB
	global.DB = db
	utils.InvalidateAllowCorporationsCache()
	t.Cleanup(func() {
		global.DB = oldDB
		utils.InvalidateAllowCorporationsCache()
	})

	seedCorporationStructureManageScope(t, db, 9001)

	svc := newCorporationStructureServiceForTest()

	settings, err := svc.GetSettings(context.Background())
	if err != nil {
		t.Fatalf("GetSettings returned error: %v", err)
	}
	if settings.FuelNoticeThresholdDays != model.SysConfigDefaultDashboardCorpStructuresFuelNoticeThresholdDays {
		t.Fatalf(
			"expected default fuel threshold %d, got %d",
			model.SysConfigDefaultDashboardCorpStructuresFuelNoticeThresholdDays,
			settings.FuelNoticeThresholdDays,
		)
	}
	if settings.TimerNoticeThresholdDays != model.SysConfigDefaultDashboardCorpStructuresTimerNoticeThresholdDays {
		t.Fatalf(
			"expected default timer threshold %d, got %d",
			model.SysConfigDefaultDashboardCorpStructuresTimerNoticeThresholdDays,
			settings.TimerNoticeThresholdDays,
		)
	}

	fuelThreshold := 3
	timerThreshold := 5
	err = svc.UpdateAuthorizations(context.Background(), CorporationStructureAuthorizationUpdate{
		Authorizations:           []CorporationStructureAuthorizationBinding{},
		FuelNoticeThresholdDays:  &fuelThreshold,
		TimerNoticeThresholdDays: &timerThreshold,
	})
	if err != nil {
		t.Fatalf("UpdateAuthorizations returned error: %v", err)
	}

	updated, err := svc.GetSettings(context.Background())
	if err != nil {
		t.Fatalf("GetSettings after update returned error: %v", err)
	}
	if updated.FuelNoticeThresholdDays != fuelThreshold {
		t.Fatalf("expected fuel threshold %d, got %d", fuelThreshold, updated.FuelNoticeThresholdDays)
	}
	if updated.TimerNoticeThresholdDays != timerThreshold {
		t.Fatalf("expected timer threshold %d, got %d", timerThreshold, updated.TimerNoticeThresholdDays)
	}
}

func TestCorporationStructureSettingsRejectsNegativeThresholds(t *testing.T) {
	db := newCorporationStructureServiceTestDB(t)
	oldDB := global.DB
	global.DB = db
	utils.InvalidateAllowCorporationsCache()
	t.Cleanup(func() {
		global.DB = oldDB
		utils.InvalidateAllowCorporationsCache()
	})

	seedCorporationStructureManageScope(t, db, 9001)

	svc := newCorporationStructureServiceForTest()
	negative := -1

	err := svc.UpdateAuthorizations(context.Background(), CorporationStructureAuthorizationUpdate{
		Authorizations:          []CorporationStructureAuthorizationBinding{},
		FuelNoticeThresholdDays: &negative,
	})
	if err == nil {
		t.Fatal("expected negative fuel threshold to be rejected")
	}
}

func TestCorporationStructureCountAttentionStructures(t *testing.T) {
	db := newCorporationStructureServiceTestDB(t)
	oldDB := global.DB
	global.DB = db
	utils.InvalidateAllowCorporationsCache()
	t.Cleanup(func() {
		global.DB = oldDB
		utils.InvalidateAllowCorporationsCache()
	})

	seedCorporationStructureManageScope(t, db, 9001)

	now := time.Now().UTC()
	seedRows := []model.CorpStructureInfo{
		{
			CorporationID: 9001,
			StructureID:   1,
			FuelExpires:   now.Add(6 * time.Hour).Format(time.RFC3339),
		},
		{
			CorporationID: 9001,
			StructureID:   2,
			StateTimerEnd: now.Add(12 * time.Hour).Format(time.RFC3339),
		},
		{
			CorporationID: 9001,
			StructureID:   3,
			FuelExpires:   now.Add(6 * time.Hour).Format(time.RFC3339),
			StateTimerEnd: now.Add(6 * time.Hour).Format(time.RFC3339),
		},
		{
			CorporationID: 9001,
			StructureID:   4,
			FuelExpires:   now.Add(9 * 24 * time.Hour).Format(time.RFC3339),
			StateTimerEnd: now.Add(9 * 24 * time.Hour).Format(time.RFC3339),
		},
	}
	for _, row := range seedRows {
		if err := db.Create(&row).Error; err != nil {
			t.Fatalf("seed row failed: %v", err)
		}
	}

	svc := newCorporationStructureServiceForTest()
	if err := svc.sysConfigRepo.SetMany([]repository.SysConfigUpsertItem{
		{
			Key:   model.SysConfigDashboardCorpStructuresFuelNoticeThresholdDays,
			Value: "2",
			Desc:  "test",
		},
		{
			Key:   model.SysConfigDashboardCorpStructuresTimerNoticeThresholdDays,
			Value: "2",
			Desc:  "test",
		},
	}); err != nil {
		t.Fatalf("set thresholds: %v", err)
	}

	count, err := svc.CountAttentionStructures(context.Background())
	if err != nil {
		t.Fatalf("CountAttentionStructures returned error: %v", err)
	}
	if count != 3 {
		t.Fatalf("expected 3 attention structures, got %d", count)
	}

	if err := svc.sysConfigRepo.SetMany([]repository.SysConfigUpsertItem{
		{
			Key:   model.SysConfigDashboardCorpStructuresFuelNoticeThresholdDays,
			Value: "0",
			Desc:  "test",
		},
		{
			Key:   model.SysConfigDashboardCorpStructuresTimerNoticeThresholdDays,
			Value: "0",
			Desc:  "test",
		},
	}); err != nil {
		t.Fatalf("set zero thresholds: %v", err)
	}
	count, err = svc.CountAttentionStructures(context.Background())
	if err != nil {
		t.Fatalf("CountAttentionStructures with zero thresholds returned error: %v", err)
	}
	if count != 0 {
		t.Fatalf("expected 0 attention structures with zero thresholds, got %d", count)
	}
}

func newCorporationStructureServiceForTest() *CorporationStructureService {
	return &CorporationStructureService{
		roleRepo:      repository.NewRoleRepository(),
		charRepo:      repository.NewEveCharacterRepository(),
		sysConfigRepo: repository.NewSysConfigRepository(),
		sdeRepo:       repository.NewSdeRepository(),
		repo:          repository.NewCorporationStructureRepository(),
		esiClient:     esi.NewClientWithConfig("http://127.0.0.1:1", ""),
	}
}

func newCorporationStructureServiceTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	dsn := fmt.Sprintf("file:corp_structure_service_test_%d?mode=memory&cache=shared", time.Now().UnixNano())
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(
		&model.User{},
		&model.UserRole{},
		&model.SystemConfig{},
		&model.EveCharacter{},
		&model.EveCharacterCorpRole{},
		&model.CorpStructureInfo{},
		&model.MapSolarSystem{},
		&model.MapRegion{},
	); err != nil {
		t.Fatalf("auto migrate: %v", err)
	}
	return db
}

func seedCorporationStructureManageScope(t *testing.T, db *gorm.DB, corpID int64) {
	t.Helper()

	admin := &model.User{BaseModel: model.BaseModel{ID: 1}, Nickname: "admin", Role: model.RoleAdmin}
	if err := db.Create(admin).Error; err != nil {
		t.Fatalf("create admin user: %v", err)
	}
	if err := db.Create(&model.UserRole{UserID: 1, RoleCode: model.RoleAdmin}).Error; err != nil {
		t.Fatalf("create admin role: %v", err)
	}
	if err := db.Create(&model.EveCharacter{
		CharacterID:   91000001,
		CharacterName: "Admin Character",
		UserID:        1,
		CorporationID: corpID,
	}).Error; err != nil {
		t.Fatalf("create admin character: %v", err)
	}
	if err := db.Create(&model.SystemConfig{
		Key:   model.SysConfigAllowCorporations,
		Value: fmt.Sprintf("[%d]", corpID),
	}).Error; err != nil {
		t.Fatalf("create allow corporations config: %v", err)
	}
}

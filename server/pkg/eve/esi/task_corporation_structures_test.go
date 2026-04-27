package esi

import (
	"amiya-eden/global"
	"amiya-eden/internal/model"
	"amiya-eden/internal/utils"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestCorporationStructuresTaskExecutePersistsEnrichedSnapshotFields(t *testing.T) {
	db := newCorporationStructuresTaskTestDB(t)
	oldDB := global.DB
	global.DB = db
	utils.InvalidateAllowCorporationsCache()
	t.Cleanup(func() {
		global.DB = oldDB
		utils.InvalidateAllowCorporationsCache()
	})

	const (
		characterID   = int64(90010001)
		corporationID = int64(555001)
		structureID   = int64(1020000000001)
		systemID      = int64(30000142)
		typeID        = int64(35832)
	)
	seedCorporationStructuresTaskScope(t, db, characterID, corporationID)
	if err := db.Create(&model.MapSolarSystem{
		SolarSystemID:   int(systemID),
		SolarSystemName: "Jita",
		Security:        0.9,
	}).Error; err != nil {
		t.Fatalf("seed mapSolarSystems: %v", err)
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch req.URL.Path {
		case fmt.Sprintf("/corporations/%d/structures/", corporationID):
			w.Header().Set("X-Pages", "1")
			_, _ = w.Write([]byte(fmt.Sprintf(`[
{"corporation_id":%d,"structure_id":%d,"system_id":%d,"type_id":%d,"state":"shield_vulnerable","name":"Alpha","services":[]}
]`, corporationID, structureID, systemID, typeID)))
		case "/universe/names":
			_, _ = w.Write([]byte(fmt.Sprintf(`[{"id":%d,"name":"Test Corp"}]`, corporationID)))
		case fmt.Sprintf("/universe/structures/%d/", structureID):
			_, _ = w.Write([]byte(fmt.Sprintf(`{"name":"Alpha","owner_id":%d,"solar_system_id":%d,"type_id":%d,"position":{"x":1,"y":2,"z":3}}`, corporationID, systemID, typeID)))
		default:
			t.Fatalf("unexpected request path: %s", req.URL.Path)
		}
	}))
	t.Cleanup(server.Close)

	task := &CorporationStructuresTask{}
	if err := task.Execute(&TaskContext{
		CharacterID: characterID,
		AccessToken: "token",
		Client:      NewClientWithConfig(server.URL, ""),
	}); err != nil {
		t.Fatalf("execute task: %v", err)
	}

	var row model.CorpStructureInfo
	if err := db.Where("structure_id = ?", structureID).First(&row).Error; err != nil {
		t.Fatalf("load corp snapshot: %v", err)
	}
	if row.CorporationName != "Test Corp" {
		t.Fatalf("corporation_name = %q, want %q", row.CorporationName, "Test Corp")
	}
	if row.SystemName != "Jita" {
		t.Fatalf("system_name = %q, want %q", row.SystemName, "Jita")
	}
	if row.Security != 0.9 {
		t.Fatalf("security = %v, want 0.9", row.Security)
	}
	if row.TypeName != fmt.Sprintf("Type-%d", typeID) {
		t.Fatalf("type_name = %q, want placeholder", row.TypeName)
	}
}

func TestCorporationStructuresTaskExecuteFallsBackToExistingSnapshotValues(t *testing.T) {
	db := newCorporationStructuresTaskTestDB(t)
	oldDB := global.DB
	global.DB = db
	utils.InvalidateAllowCorporationsCache()
	t.Cleanup(func() {
		global.DB = oldDB
		utils.InvalidateAllowCorporationsCache()
	})

	const (
		characterID   = int64(90010002)
		corporationID = int64(555002)
		structureID   = int64(1020000000002)
		systemID      = int64(30002187)
		typeID        = int64(35833)
	)
	seedCorporationStructuresTaskScope(t, db, characterID, corporationID)
	if err := db.Create(&model.CorpStructureInfo{
		CorporationID:   corporationID,
		CorporationName: "Old Corp Name",
		StructureID:     structureID,
		TypeID:          typeID,
		TypeName:        "Old Type Name",
		SystemID:        systemID,
		SystemName:      "Old System Name",
		Security:        0.6,
	}).Error; err != nil {
		t.Fatalf("seed existing snapshot: %v", err)
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch req.URL.Path {
		case fmt.Sprintf("/corporations/%d/structures/", corporationID):
			w.Header().Set("X-Pages", "1")
			_, _ = w.Write([]byte(fmt.Sprintf(`[
{"corporation_id":%d,"structure_id":%d,"system_id":%d,"type_id":%d,"state":"low_power","name":"Beta","services":[]}
]`, corporationID, structureID, systemID, typeID)))
		case "/universe/names":
			http.Error(w, `{"error":"boom"}`, http.StatusInternalServerError)
		case fmt.Sprintf("/universe/structures/%d/", structureID):
			_, _ = w.Write([]byte(fmt.Sprintf(`{"name":"Beta","owner_id":%d,"solar_system_id":%d,"type_id":%d,"position":{"x":1,"y":2,"z":3}}`, corporationID, systemID, typeID)))
		default:
			t.Fatalf("unexpected request path: %s", req.URL.Path)
		}
	}))
	t.Cleanup(server.Close)

	task := &CorporationStructuresTask{}
	if err := task.Execute(&TaskContext{
		CharacterID: characterID,
		AccessToken: "token",
		Client:      NewClientWithConfig(server.URL, ""),
	}); err != nil {
		t.Fatalf("execute task: %v", err)
	}

	var row model.CorpStructureInfo
	if err := db.Where("structure_id = ?", structureID).First(&row).Error; err != nil {
		t.Fatalf("load corp snapshot: %v", err)
	}
	if row.CorporationName != "Old Corp Name" {
		t.Fatalf("corporation_name = %q, want old value", row.CorporationName)
	}
	if row.TypeName != "Old Type Name" {
		t.Fatalf("type_name = %q, want old value", row.TypeName)
	}
	if row.SystemName != "Old System Name" {
		t.Fatalf("system_name = %q, want old value", row.SystemName)
	}
	if row.Security != 0.6 {
		t.Fatalf("security = %v, want old value 0.6", row.Security)
	}
}

func TestCorporationStructuresTaskExecuteUsesPlaceholdersWhenNoSnapshotAndNoLookup(t *testing.T) {
	db := newCorporationStructuresTaskTestDB(t)
	oldDB := global.DB
	global.DB = db
	utils.InvalidateAllowCorporationsCache()
	t.Cleanup(func() {
		global.DB = oldDB
		utils.InvalidateAllowCorporationsCache()
	})

	const (
		characterID   = int64(90010003)
		corporationID = int64(555003)
		structureID   = int64(1020000000003)
		systemID      = int64(30002188)
		typeID        = int64(35834)
	)
	seedCorporationStructuresTaskScope(t, db, characterID, corporationID)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch req.URL.Path {
		case fmt.Sprintf("/corporations/%d/structures/", corporationID):
			w.Header().Set("X-Pages", "1")
			_, _ = w.Write([]byte(fmt.Sprintf(`[
{"corporation_id":%d,"structure_id":%d,"system_id":%d,"type_id":%d,"state":"shield_vulnerable","name":"Gamma","services":[]}
]`, corporationID, structureID, systemID, typeID)))
		case "/universe/names":
			http.Error(w, `{"error":"boom"}`, http.StatusInternalServerError)
		case fmt.Sprintf("/universe/structures/%d/", structureID):
			_, _ = w.Write([]byte(fmt.Sprintf(`{"name":"Gamma","owner_id":%d,"solar_system_id":%d,"type_id":%d,"position":{"x":1,"y":2,"z":3}}`, corporationID, systemID, typeID)))
		default:
			t.Fatalf("unexpected request path: %s", req.URL.Path)
		}
	}))
	t.Cleanup(server.Close)

	task := &CorporationStructuresTask{}
	if err := task.Execute(&TaskContext{
		CharacterID: characterID,
		AccessToken: "token",
		Client:      NewClientWithConfig(server.URL, ""),
	}); err != nil {
		t.Fatalf("execute task: %v", err)
	}

	var row model.CorpStructureInfo
	if err := db.Where("structure_id = ?", structureID).First(&row).Error; err != nil {
		t.Fatalf("load corp snapshot: %v", err)
	}
	if row.CorporationName != fmt.Sprintf("Corporation-%d", corporationID) {
		t.Fatalf("corporation_name = %q, want placeholder", row.CorporationName)
	}
	if row.TypeName != fmt.Sprintf("Type-%d", typeID) {
		t.Fatalf("type_name = %q, want placeholder", row.TypeName)
	}
	if row.SystemName != fmt.Sprintf("System-%d", systemID) {
		t.Fatalf("system_name = %q, want placeholder", row.SystemName)
	}
}

func TestCorporationStructuresTaskExecuteDeletesMissingStructures(t *testing.T) {
	db := newCorporationStructuresTaskTestDB(t)
	oldDB := global.DB
	global.DB = db
	utils.InvalidateAllowCorporationsCache()
	t.Cleanup(func() {
		global.DB = oldDB
		utils.InvalidateAllowCorporationsCache()
	})

	const (
		characterID   = int64(90010011)
		corporationID = int64(555011)
		systemID      = int64(30000142)
		typeID        = int64(35832)
	)
	seedCorporationStructuresTaskScope(t, db, characterID, corporationID)

	existingIDs := []int64{1020000000111, 1020000000112, 1020000000113}
	for _, structureID := range existingIDs {
		if err := db.Create(&model.CorpStructureInfo{
			CorporationID: corporationID,
			StructureID:   structureID,
			SystemID:      systemID,
			TypeID:        typeID,
			Name:          fmt.Sprintf("Existing-%d", structureID),
		}).Error; err != nil {
			t.Fatalf("seed existing structure %d: %v", structureID, err)
		}
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch req.URL.Path {
		case fmt.Sprintf("/corporations/%d/structures/", corporationID):
			w.Header().Set("X-Pages", "1")
			_, _ = w.Write([]byte(fmt.Sprintf(`[
{"corporation_id":%d,"structure_id":%d,"system_id":%d,"type_id":%d,"state":"shield_vulnerable","name":"A","services":[]},
{"corporation_id":%d,"structure_id":%d,"system_id":%d,"type_id":%d,"state":"low_power","name":"B","services":[]}
]`, corporationID, existingIDs[0], systemID, typeID, corporationID, existingIDs[1], systemID, typeID)))
		case "/universe/names":
			_, _ = w.Write([]byte(fmt.Sprintf(`[{"id":%d,"name":"Test Corp"}]`, corporationID)))
		case fmt.Sprintf("/universe/structures/%d/", existingIDs[0]):
			_, _ = w.Write([]byte(fmt.Sprintf(`{"name":"A","owner_id":%d,"solar_system_id":%d,"type_id":%d,"position":{"x":1,"y":2,"z":3}}`, corporationID, systemID, typeID)))
		case fmt.Sprintf("/universe/structures/%d/", existingIDs[1]):
			_, _ = w.Write([]byte(fmt.Sprintf(`{"name":"B","owner_id":%d,"solar_system_id":%d,"type_id":%d,"position":{"x":4,"y":5,"z":6}}`, corporationID, systemID, typeID)))
		default:
			t.Fatalf("unexpected request path: %s", req.URL.Path)
		}
	}))
	t.Cleanup(server.Close)

	task := &CorporationStructuresTask{}
	if err := task.Execute(&TaskContext{
		CharacterID: characterID,
		AccessToken: "token",
		Client:      NewClientWithConfig(server.URL, ""),
	}); err != nil {
		t.Fatalf("execute task: %v", err)
	}

	var rows []model.CorpStructureInfo
	if err := db.Where("corporation_id = ?", corporationID).
		Order("structure_id ASC").
		Find(&rows).Error; err != nil {
		t.Fatalf("load corp snapshots: %v", err)
	}
	if len(rows) != 2 {
		t.Fatalf("row count = %d, want 2", len(rows))
	}
	if rows[0].StructureID != existingIDs[0] || rows[1].StructureID != existingIDs[1] {
		t.Fatalf("remaining structure ids = [%d, %d], want [%d, %d]",
			rows[0].StructureID, rows[1].StructureID, existingIDs[0], existingIDs[1])
	}
}

func TestCorporationStructuresTaskExecuteClearsCorporationWhenESIReturnsEmpty(t *testing.T) {
	db := newCorporationStructuresTaskTestDB(t)
	oldDB := global.DB
	global.DB = db
	utils.InvalidateAllowCorporationsCache()
	t.Cleanup(func() {
		global.DB = oldDB
		utils.InvalidateAllowCorporationsCache()
	})

	const (
		characterID    = int64(90010012)
		corporationID  = int64(555012)
		otherCorpID    = int64(555013)
		targetStructID = int64(1020000000121)
		otherStructID  = int64(1020000000131)
	)
	seedCorporationStructuresTaskScope(t, db, characterID, corporationID)
	if err := db.Create(&model.CorpStructureInfo{
		CorporationID: corporationID,
		StructureID:   targetStructID,
		Name:          "Will-Be-Deleted",
	}).Error; err != nil {
		t.Fatalf("seed target corp structure: %v", err)
	}
	if err := db.Create(&model.CorpStructureInfo{
		CorporationID: otherCorpID,
		StructureID:   otherStructID,
		Name:          "Should-Stay",
	}).Error; err != nil {
		t.Fatalf("seed other corp structure: %v", err)
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch req.URL.Path {
		case fmt.Sprintf("/corporations/%d/structures/", corporationID):
			w.Header().Set("X-Pages", "1")
			_, _ = w.Write([]byte(`[]`))
		default:
			t.Fatalf("unexpected request path: %s", req.URL.Path)
		}
	}))
	t.Cleanup(server.Close)

	task := &CorporationStructuresTask{}
	if err := task.Execute(&TaskContext{
		CharacterID: characterID,
		AccessToken: "token",
		Client:      NewClientWithConfig(server.URL, ""),
	}); err != nil {
		t.Fatalf("execute task: %v", err)
	}

	var targetCount int64
	if err := db.Model(&model.CorpStructureInfo{}).
		Where("corporation_id = ?", corporationID).
		Count(&targetCount).Error; err != nil {
		t.Fatalf("count target corp snapshots: %v", err)
	}
	if targetCount != 0 {
		t.Fatalf("target corp row count = %d, want 0", targetCount)
	}

	var otherCount int64
	if err := db.Model(&model.CorpStructureInfo{}).
		Where("corporation_id = ?", otherCorpID).
		Count(&otherCount).Error; err != nil {
		t.Fatalf("count other corp snapshots: %v", err)
	}
	if otherCount != 1 {
		t.Fatalf("other corp row count = %d, want 1", otherCount)
	}
}

func TestCorporationStructuresTaskExecuteDoesNotDeleteOnFetchFailure(t *testing.T) {
	db := newCorporationStructuresTaskTestDB(t)
	oldDB := global.DB
	global.DB = db
	utils.InvalidateAllowCorporationsCache()
	t.Cleanup(func() {
		global.DB = oldDB
		utils.InvalidateAllowCorporationsCache()
	})

	const (
		characterID   = int64(90010013)
		corporationID = int64(555014)
		structureID   = int64(1020000000141)
	)
	seedCorporationStructuresTaskScope(t, db, characterID, corporationID)
	if err := db.Create(&model.CorpStructureInfo{
		CorporationID: corporationID,
		StructureID:   structureID,
		Name:          "Persist-On-Error",
	}).Error; err != nil {
		t.Fatalf("seed existing structure: %v", err)
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch req.URL.Path {
		case fmt.Sprintf("/corporations/%d/structures/", corporationID):
			http.Error(w, `{"error":"boom"}`, http.StatusInternalServerError)
		default:
			t.Fatalf("unexpected request path: %s", req.URL.Path)
		}
	}))
	t.Cleanup(server.Close)

	task := &CorporationStructuresTask{}
	if err := task.Execute(&TaskContext{
		CharacterID: characterID,
		AccessToken: "token",
		Client:      NewClientWithConfig(server.URL, ""),
	}); err == nil {
		t.Fatal("expected execute error, got nil")
	}

	var count int64
	if err := db.Model(&model.CorpStructureInfo{}).
		Where("corporation_id = ?", corporationID).
		Count(&count).Error; err != nil {
		t.Fatalf("count corp snapshots: %v", err)
	}
	if count != 1 {
		t.Fatalf("row count = %d, want 1", count)
	}
}

func TestCorporationStructuresTaskExecuteOnlyDeletesTargetCorporation(t *testing.T) {
	db := newCorporationStructuresTaskTestDB(t)
	oldDB := global.DB
	global.DB = db
	utils.InvalidateAllowCorporationsCache()
	t.Cleanup(func() {
		global.DB = oldDB
		utils.InvalidateAllowCorporationsCache()
	})

	const (
		characterID   = int64(90010014)
		corporationID = int64(555015)
		otherCorpID   = int64(555016)
		systemID      = int64(30000142)
		typeID        = int64(35832)
	)
	seedCorporationStructuresTaskScope(t, db, characterID, corporationID)
	if err := db.Create(&model.CorpStructureInfo{
		CorporationID: corporationID,
		StructureID:   1020000000151,
		Name:          "Target-Old",
	}).Error; err != nil {
		t.Fatalf("seed target structure: %v", err)
	}
	if err := db.Create(&model.CorpStructureInfo{
		CorporationID: otherCorpID,
		StructureID:   1020000000161,
		Name:          "Other-Stay",
	}).Error; err != nil {
		t.Fatalf("seed other corp structure: %v", err)
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch req.URL.Path {
		case fmt.Sprintf("/corporations/%d/structures/", corporationID):
			w.Header().Set("X-Pages", "1")
			_, _ = w.Write([]byte(fmt.Sprintf(`[
{"corporation_id":%d,"structure_id":%d,"system_id":%d,"type_id":%d,"state":"shield_vulnerable","name":"Target-New","services":[]}
]`, corporationID, int64(1020000000152), systemID, typeID)))
		case "/universe/names":
			_, _ = w.Write([]byte(fmt.Sprintf(`[{"id":%d,"name":"Test Corp"}]`, corporationID)))
		case fmt.Sprintf("/universe/structures/%d/", int64(1020000000152)):
			_, _ = w.Write([]byte(fmt.Sprintf(`{"name":"Target-New","owner_id":%d,"solar_system_id":%d,"type_id":%d,"position":{"x":7,"y":8,"z":9}}`, corporationID, systemID, typeID)))
		default:
			t.Fatalf("unexpected request path: %s", req.URL.Path)
		}
	}))
	t.Cleanup(server.Close)

	task := &CorporationStructuresTask{}
	if err := task.Execute(&TaskContext{
		CharacterID: characterID,
		AccessToken: "token",
		Client:      NewClientWithConfig(server.URL, ""),
	}); err != nil {
		t.Fatalf("execute task: %v", err)
	}

	var targetRows []model.CorpStructureInfo
	if err := db.Where("corporation_id = ?", corporationID).Find(&targetRows).Error; err != nil {
		t.Fatalf("load target corp rows: %v", err)
	}
	if len(targetRows) != 1 || targetRows[0].StructureID != 1020000000152 {
		t.Fatalf("target corp rows = %+v, want one row with structure_id 1020000000152", targetRows)
	}

	var otherCount int64
	if err := db.Model(&model.CorpStructureInfo{}).
		Where("corporation_id = ?", otherCorpID).
		Count(&otherCount).Error; err != nil {
		t.Fatalf("count other corp rows: %v", err)
	}
	if otherCount != 1 {
		t.Fatalf("other corp row count = %d, want 1", otherCount)
	}
}

func seedCorporationStructuresTaskScope(t *testing.T, db *gorm.DB, characterID, corporationID int64) {
	t.Helper()
	if err := db.Create(&model.EveCharacter{
		CharacterID:   characterID,
		CharacterName: "Director",
		UserID:        1,
		CorporationID: corporationID,
	}).Error; err != nil {
		t.Fatalf("seed eve_character: %v", err)
	}
	if err := db.Create(&model.EveCharacterCorpRole{
		CharacterID: characterID,
		CorpRole:    "Director",
	}).Error; err != nil {
		t.Fatalf("seed eve_character_corp_role: %v", err)
	}
	if err := db.Create(&model.SystemConfig{
		Key:   model.SysConfigAllowCorporations,
		Value: fmt.Sprintf("[%d]", corporationID),
	}).Error; err != nil {
		t.Fatalf("seed allow corporations config: %v", err)
	}

	authPayload, err := json.Marshal(map[string]int64{
		fmt.Sprintf("%d", corporationID): characterID,
	})
	if err != nil {
		t.Fatalf("marshal auth payload: %v", err)
	}
	if err := db.Create(&model.SystemConfig{
		Key:   model.SysConfigDashboardCorpStructuresAuth,
		Value: string(authPayload),
	}).Error; err != nil {
		t.Fatalf("seed dashboard auth config: %v", err)
	}
}

func newCorporationStructuresTaskTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	dsn := fmt.Sprintf("file:corp_structures_task_test_%d?mode=memory&cache=shared", time.Now().UnixNano())
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(
		&model.SystemConfig{},
		&model.EveCharacter{},
		&model.EveCharacterCorpRole{},
		&model.CorpStructureInfo{},
		&model.EveStructure{},
		&model.MapSolarSystem{},
	); err != nil {
		t.Fatalf("auto migrate: %v", err)
	}
	return db
}

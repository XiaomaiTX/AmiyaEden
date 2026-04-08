package esi

import (
	"amiya-eden/global"
	"amiya-eden/internal/model"
	"fmt"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
	"time"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestCharacterCorporationHistoryTaskSyncStoresFullHistoryAndComputesTenure(t *testing.T) {
	db := newCharacterCorporationHistoryTaskTestDB(t)
	oldDB := global.DB
	global.DB = db
	t.Cleanup(func() { global.DB = oldDB })

	oldTenure := 99
	character := &model.EveCharacter{
		CharacterID:          91000001,
		CharacterName:        "Pilot One",
		UserID:               1,
		CorporationID:        model.SystemCorporationID,
		FuxiLegionTenureDays: &oldTenure,
	}
	if err := db.Create(character).Error; err != nil {
		t.Fatalf("create character: %v", err)
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = fmt.Fprintf(w, `[
			{"record_id":1001,"corporation_id":%d,"is_deleted":false,"start_date":"2026-01-01T00:00:00Z"},
			{"record_id":1002,"corporation_id":12345,"is_deleted":true,"start_date":"2026-01-11T00:00:00Z"},
			{"record_id":1003,"corporation_id":%d,"is_deleted":false,"start_date":"2026-01-21T00:00:00Z"},
			{"record_id":1004,"corporation_id":54321,"is_deleted":false,"start_date":"2026-02-05T00:00:00Z"}
		]`, model.SystemCorporationID, model.SystemCorporationID)
	}))
	t.Cleanup(server.Close)

	task := &CorporationHistoryTask{}
	if err := task.Execute(&TaskContext{
		CharacterID: character.CharacterID,
		Client:      NewClientWithConfig(server.URL, ""),
	}); err != nil {
		t.Fatalf("execute task: %v", err)
	}

	var historyRows []model.CharacterCorporationHistory
	if err := db.Order("start_date asc").Find(&historyRows).Error; err != nil {
		t.Fatalf("list history rows: %v", err)
	}
	if len(historyRows) != 4 {
		t.Fatalf("expected 4 history rows, got %d", len(historyRows))
	}

	recordIDs := make([]int64, 0, len(historyRows))
	for _, row := range historyRows {
		recordIDs = append(recordIDs, row.RecordID)
	}
	if !reflect.DeepEqual(recordIDs, []int64{1001, 1002, 1003, 1004}) {
		t.Fatalf("expected persisted ESI record ids, got %v", recordIDs)
	}
	if !historyRows[1].IsDeleted {
		t.Fatal("expected is_deleted to be stored from the ESI payload")
	}

	var persisted model.EveCharacter
	if err := db.Where("character_id = ?", character.CharacterID).First(&persisted).Error; err != nil {
		t.Fatalf("reload character: %v", err)
	}
	if persisted.FuxiLegionTenureDays == nil || *persisted.FuxiLegionTenureDays != 25 {
		t.Fatalf("expected 25 tenure days from stored history, got %+v", persisted.FuxiLegionTenureDays)
	}
}

func TestNormalizeCorporationHistoryRowsOrdersByStartDateThenRecordID(t *testing.T) {
	rows := normalizeCorporationHistoryRows(91000003, []corporationHistoryResponse{
		{RecordID: 30, CorporationID: 12345, StartDate: "2026-03-01T00:00:00Z"},
		{RecordID: 10, CorporationID: model.SystemCorporationID, IsDeleted: true, StartDate: "2026-03-01T00:00:00Z"},
		{RecordID: 20, CorporationID: 54321, StartDate: "2026-02-01T00:00:00Z"},
	})

	recordIDs := make([]int64, 0, len(rows))
	for _, row := range rows {
		recordIDs = append(recordIDs, row.RecordID)
	}
	if !reflect.DeepEqual(recordIDs, []int64{20, 10, 30}) {
		t.Fatalf("expected rows to sort by start date then record id, got %v", recordIDs)
	}
	if !rows[1].IsDeleted {
		t.Fatal("expected normalized rows to preserve is_deleted")
	}
}

func TestCharacterCorporationHistoryTaskSyncReplacesExistingRowsOnRepeatedSync(t *testing.T) {
	db := newCharacterCorporationHistoryTaskTestDB(t)
	oldDB := global.DB
	global.DB = db
	t.Cleanup(func() { global.DB = oldDB })

	character := &model.EveCharacter{
		CharacterID:   91000002,
		CharacterName: "Pilot Two",
		UserID:        2,
		CorporationID: model.SystemCorporationID,
	}
	if err := db.Create(character).Error; err != nil {
		t.Fatalf("create character: %v", err)
	}
	if err := db.Create(&model.CharacterCorporationHistory{
		CharacterID:   character.CharacterID,
		RecordID:      9,
		CorporationID: 99999,
		StartDate:     time.Date(2025, time.December, 1, 0, 0, 0, 0, time.UTC),
	}).Error; err != nil {
		t.Fatalf("create stale history row: %v", err)
	}

	responses := []string{
		fmt.Sprintf(`[
			{"record_id":2001,"corporation_id":%d,"start_date":"2026-03-01T00:00:00Z"},
			{"record_id":2002,"corporation_id":12345,"start_date":"2026-03-03T00:00:00Z"}
		]`, model.SystemCorporationID),
		fmt.Sprintf(`[
			{"record_id":3001,"corporation_id":%d,"start_date":"2026-03-10T00:00:00Z"},
			{"record_id":3002,"corporation_id":12345,"start_date":"2026-03-15T00:00:00Z"}
		]`, model.SystemCorporationID),
	}
	requestCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(responses[requestCount]))
		requestCount++
	}))
	t.Cleanup(server.Close)

	task := &CorporationHistoryTask{}
	ctx := &TaskContext{CharacterID: character.CharacterID, Client: NewClientWithConfig(server.URL, "")}
	if err := task.Execute(ctx); err != nil {
		t.Fatalf("first execute: %v", err)
	}
	if err := task.Execute(ctx); err != nil {
		t.Fatalf("second execute: %v", err)
	}

	var historyRows []model.CharacterCorporationHistory
	if err := db.Order("record_id asc").Find(&historyRows).Error; err != nil {
		t.Fatalf("list history rows: %v", err)
	}
	if len(historyRows) != 2 {
		t.Fatalf("expected repeated sync to leave exactly 2 current rows, got %d", len(historyRows))
	}

	recordIDs := []int64{historyRows[0].RecordID, historyRows[1].RecordID}
	if !reflect.DeepEqual(recordIDs, []int64{3001, 3002}) {
		t.Fatalf("expected stale and previous rows to be replaced, got %v", recordIDs)
	}

	var staleCount int64
	if err := db.Model(&model.CharacterCorporationHistory{}).Where("record_id IN ?", []int64{9, 2001, 2002}).Count(&staleCount).Error; err != nil {
		t.Fatalf("count stale rows: %v", err)
	}
	if staleCount != 0 {
		t.Fatalf("expected stale rows to be removed, found %d", staleCount)
	}

	var persisted model.EveCharacter
	if err := db.Where("character_id = ?", character.CharacterID).First(&persisted).Error; err != nil {
		t.Fatalf("reload character: %v", err)
	}
	if persisted.FuxiLegionTenureDays == nil || *persisted.FuxiLegionTenureDays != 5 {
		t.Fatalf("expected repeated sync to recompute tenure to 5 days, got %+v", persisted.FuxiLegionTenureDays)
	}
}

func TestTaskIntervalsMatchConfiguredCadence(t *testing.T) {
	affiliation := (&AffiliationTask{}).Interval()
	if affiliation.Active != 6*time.Hour || affiliation.Inactive != 6*time.Hour {
		t.Fatalf("unexpected affiliation intervals: %+v", affiliation)
	}

	corpRoles := (&CorpRolesTask{}).Interval()
	if corpRoles.Active != 24*time.Hour || corpRoles.Inactive != 24*time.Hour {
		t.Fatalf("unexpected corp role intervals: %+v", corpRoles)
	}

	titles := (&TitlesTask{}).Interval()
	if titles.Active != 24*time.Hour || titles.Inactive != 24*time.Hour {
		t.Fatalf("unexpected title intervals: %+v", titles)
	}

	clones := (&ClonesTask{}).Interval()
	if clones.Active != 24*time.Hour || clones.Inactive != 24*time.Hour {
		t.Fatalf("unexpected clone intervals: %+v", clones)
	}

	history := (&CorporationHistoryTask{}).Interval()
	if history.Active != 7*24*time.Hour || history.Inactive != 7*24*time.Hour {
		t.Fatalf("unexpected corporation history intervals: %+v", history)
	}
}

func newCharacterCorporationHistoryTaskTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	dsn := fmt.Sprintf("file:corp_history_task_test_%d?mode=memory&cache=shared", time.Now().UnixNano())
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&model.EveCharacter{}, &model.CharacterCorporationHistory{}); err != nil {
		t.Fatalf("auto migrate: %v", err)
	}
	return db
}

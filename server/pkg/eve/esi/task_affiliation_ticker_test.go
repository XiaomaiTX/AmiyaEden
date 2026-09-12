package esi

import (
	"amiya-eden/global"
	"amiya-eden/internal/model"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	"github.com/glebarez/sqlite"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

func TestAffiliationTaskPersistsTickerSnapshots(t *testing.T) {
	oldDB := global.DB
	oldLogger := global.Logger
	global.SetLogger(zap.NewNop())
	t.Cleanup(func() {
		global.DB = oldDB
		global.SetLogger(oldLogger)
	})
	db, err := gorm.Open(sqlite.Open("file:affiliation_ticker_cache?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&model.EveCharacter{}, &model.EveEntityTickerCache{}); err != nil {
		t.Fatal(err)
	}
	global.DB = db
	if err := db.Create(&model.EveCharacter{CharacterID: 900001, CharacterName: "Pilot"}).Error; err != nil {
		t.Fatal(err)
	}

	var tickerRequests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		switch request.URL.Path {
		case "/characters/affiliation/":
			_ = json.NewEncoder(writer).Encode([]AffiliationResult{{CharacterID: 900001, CorporationID: 98000001, AllianceID: int64Pointer(99000006)}})
		case "/corporations/98000001/":
			tickerRequests.Add(1)
			_, _ = writer.Write([]byte(`{"ticker":"FUXI"}`))
		case "/alliances/99000006/":
			tickerRequests.Add(1)
			_, _ = writer.Write([]byte(`{"ticker":"FRT"}`))
		default:
			http.NotFound(writer, request)
		}
	}))
	defer server.Close()

	task := &AffiliationTask{}
	client := NewClientWithConfig(server.URL, "")
	if err := task.fetchAffiliation(t.Context(), client, []int64{900001}); err != nil {
		t.Fatalf("fetch affiliation: %v", err)
	}
	assertAffiliationTicker(t, db, model.EntityTickerTypeCorporation, 98000001, "FUXI")
	assertAffiliationTicker(t, db, model.EntityTickerTypeAlliance, 99000006, "FRT")

	if err := task.fetchAffiliation(t.Context(), client, []int64{900001}); err != nil {
		t.Fatalf("refresh cached affiliation: %v", err)
	}
	if got := tickerRequests.Load(); got != 2 {
		t.Fatalf("ticker request count = %d, want 2 after cached refresh", got)
	}
}

func assertAffiliationTicker(t *testing.T, db *gorm.DB, entityType string, entityID int64, want string) {
	t.Helper()
	var row model.EveEntityTickerCache
	if err := db.Where("entity_type = ? AND entity_id = ?", entityType, entityID).First(&row).Error; err != nil {
		t.Fatal(err)
	}
	if row.Ticker != want {
		t.Fatalf("%s ticker = %q, want %q", entityType, row.Ticker, want)
	}
}

func int64Pointer(value int64) *int64 { return &value }

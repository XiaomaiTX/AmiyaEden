package service

import (
	"amiya-eden/config"
	"amiya-eden/global"
	"amiya-eden/internal/model"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

func setupEntityNameResolverTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file:entity_name_resolver_test?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&model.EveEntityNameCache{}); err != nil {
		t.Fatalf("auto migrate: %v", err)
	}
	return db
}

func TestEntityNameResolverCacheHitWithoutESI(t *testing.T) {
	db := setupEntityNameResolverTestDB(t)
	origDB, origCfg, origLogger := global.DB, global.Config, global.Logger
	global.DB = db
	global.Logger = zap.NewNop()
	defer func() {
		global.DB, global.Config, global.Logger = origDB, origCfg, origLogger
	}()

	now := time.Now()
	if err := db.Create(&model.EveEntityNameCache{
		EntityID:       1001,
		EntityType:     model.EntityNameTypeCharacter,
		Name:           "Cached Pilot",
		Source:         model.EntityNameSourceESI,
		LastResolvedAt: now,
		ExpiresAt:      now.Add(time.Hour),
	}).Error; err != nil {
		t.Fatalf("seed cache: %v", err)
	}

	global.Config = &config.Config{}
	config.ApplyDefaults(global.Config)
	global.Config.EveSSO.ESIBaseURL = "http://127.0.0.1:0"
	global.Config.EveSSO.ESIAPIPrefix = ""

	resolver := NewEntityNameResolver()
	result := resolver.Resolve(t.Context(), []int64{1001})
	if got := result.Names[1001]; got != "Cached Pilot" {
		t.Fatalf("cached name = %q, want %q", got, "Cached Pilot")
	}
	if len(result.Miss) != 0 {
		t.Fatalf("misses = %v, want empty", result.Miss)
	}
}

func TestEntityNameResolverPartialHitAndBackfill(t *testing.T) {
	db := setupEntityNameResolverTestDB(t)
	origDB, origCfg, origLogger := global.DB, global.Config, global.Logger
	global.DB = db
	global.Logger = zap.NewNop()
	defer func() {
		global.DB, global.Config, global.Logger = origDB, origCfg, origLogger
	}()

	now := time.Now()
	if err := db.Create(&model.EveEntityNameCache{
		EntityID:       2001,
		EntityType:     model.EntityNameTypeCorporation,
		Name:           "Cached Corp",
		Source:         model.EntityNameSourceESI,
		LastResolvedAt: now,
		ExpiresAt:      now.Add(time.Hour),
	}).Error; err != nil {
		t.Fatalf("seed cache: %v", err)
	}

	esiServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/universe/names" {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`[{"id":2002,"name":"Fresh Alliance","category":"alliance"}]`))
	}))
	defer esiServer.Close()

	global.Config = &config.Config{}
	config.ApplyDefaults(global.Config)
	global.Config.EveSSO.ESIBaseURL = esiServer.URL
	global.Config.EveSSO.ESIAPIPrefix = ""

	resolver := NewEntityNameResolver()
	result := resolver.Resolve(t.Context(), []int64{2001, 2002})
	if got := result.Names[2001]; got != "Cached Corp" {
		t.Fatalf("name[2001] = %q, want %q", got, "Cached Corp")
	}
	if got := result.Names[2002]; got != "Fresh Alliance" {
		t.Fatalf("name[2002] = %q, want %q", got, "Fresh Alliance")
	}
	if len(result.Miss) != 0 {
		t.Fatalf("misses = %v, want empty", result.Miss)
	}

	var backfilled model.EveEntityNameCache
	if err := db.Where("entity_id = ?", 2002).First(&backfilled).Error; err != nil {
		t.Fatalf("load backfilled cache: %v", err)
	}
	if backfilled.Name != "Fresh Alliance" {
		t.Fatalf("backfilled name = %q, want %q", backfilled.Name, "Fresh Alliance")
	}
}

func TestEntityNameResolverESIFailureReturnsMisses(t *testing.T) {
	db := setupEntityNameResolverTestDB(t)
	origDB, origCfg, origLogger := global.DB, global.Config, global.Logger
	global.DB = db
	global.Logger = zap.NewNop()
	defer func() {
		global.DB, global.Config, global.Logger = origDB, origCfg, origLogger
	}()

	esiServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTooManyRequests)
		_, _ = w.Write([]byte(`{"error":"Rate limit exceeded"}`))
	}))
	defer esiServer.Close()

	global.Config = &config.Config{}
	config.ApplyDefaults(global.Config)
	global.Config.EveSSO.ESIBaseURL = esiServer.URL
	global.Config.EveSSO.ESIAPIPrefix = ""

	resolver := NewEntityNameResolver()
	result := resolver.Resolve(t.Context(), []int64{3001, 3002})
	if len(result.Names) != 0 {
		t.Fatalf("names = %v, want empty", result.Names)
	}
	if len(result.Miss) != 2 || result.Miss[0] != 3001 || result.Miss[1] != 3002 {
		t.Fatalf("misses = %v, want [3001 3002]", result.Miss)
	}
}

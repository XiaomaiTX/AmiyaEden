package service

import (
	"amiya-eden/config"
	"amiya-eden/global"
	"amiya-eden/internal/model"
	"amiya-eden/internal/repository"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"go.uber.org/zap"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func setupQQGovernancePolicyTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := fmt.Sprintf("file:%s_%d?mode=memory&cache=shared", strings.ReplaceAll(t.Name(), "/", "_"), time.Now().UnixNano())
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&model.QQGroupGovernancePolicy{}, &model.EveEntityNameCache{}); err != nil {
		t.Fatalf("auto migrate: %v", err)
	}
	return db
}

func TestSysConfigSearchCorporationsPostsToUniverseIDs(t *testing.T) {
	db := setupQQGovernancePolicyTestDB(t)
	origDB, origCfg, origLogger := global.DB, global.Config, global.Logger
	global.DB = db
	global.Logger = zap.NewNop()
	defer func() {
		global.DB, global.Config, global.Logger = origDB, origCfg, origLogger
	}()

	var (
		gotMethod string
		gotPath   string
		gotBody   string
	)
	esiServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path
		bodyBytes, _ := io.ReadAll(r.Body)
		gotBody = string(bodyBytes)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"corporations":[{"id":1234567,"name":"Pivot Quantum"}]}`))
	}))
	defer esiServer.Close()

	global.Config = &config.Config{}
	config.ApplyDefaults(global.Config)
	global.Config.EveSSO.ESIBaseURL = esiServer.URL
	global.Config.EveSSO.ESIAPIPrefix = ""

	svc := NewSysConfigService()
	result, err := svc.SearchCorporations(t.Context(), "Pivot Quantum")
	if err != nil {
		t.Fatalf("search corporations: %v", err)
	}
	if gotMethod != http.MethodPost {
		t.Fatalf("method = %s, want POST", gotMethod)
	}
	if gotPath != "/universe/ids/" {
		t.Fatalf("path = %s, want /universe/ids/", gotPath)
	}
	var posted []string
	if err := json.Unmarshal([]byte(gotBody), &posted); err != nil {
		t.Fatalf("unmarshal posted body: %v (body=%q)", err, gotBody)
	}
	if len(posted) != 1 || posted[0] != "Pivot Quantum" {
		t.Fatalf("posted = %v, want [\"Pivot Quantum\"]", posted)
	}
	if len(result) != 1 || result[0].CorporationID != 1234567 || result[0].CorporationName != "Pivot Quantum" {
		t.Fatalf("result = %+v", result)
	}
}

func TestSysConfigSearchCorporationsEmptyAndNoMatch(t *testing.T) {
	origCfg, origLogger := global.Config, global.Logger
	global.Logger = zap.NewNop()
	defer func() {
		global.Config, global.Logger = origCfg, origLogger
	}()

	var requests int
	esiServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"corporations":[]}`))
	}))
	defer esiServer.Close()

	global.Config = &config.Config{}
	config.ApplyDefaults(global.Config)
	global.Config.EveSSO.ESIBaseURL = esiServer.URL
	global.Config.EveSSO.ESIAPIPrefix = ""

	svc := NewSysConfigService()
	empty, err := svc.SearchCorporations(t.Context(), "   ")
	if err != nil {
		t.Fatalf("empty query: %v", err)
	}
	if len(empty) != 0 || requests != 0 {
		t.Fatalf("empty query should skip ESI; got result=%v requests=%d", empty, requests)
	}
	noMatch, err := svc.SearchCorporations(context.Background(), "Does Not Exist Corp")
	if err != nil {
		t.Fatalf("no-match query: %v", err)
	}
	if len(noMatch) != 0 {
		t.Fatalf("no-match result = %v, want empty", noMatch)
	}
	if requests != 1 {
		t.Fatalf("expected single ESI request for no-match query, got %d", requests)
	}
}

func TestSysConfigSearchCorporationsESIFailureBubblesUp(t *testing.T) {
	origCfg, origLogger := global.Config, global.Logger
	global.Logger = zap.NewNop()
	defer func() {
		global.Config, global.Logger = origCfg, origLogger
	}()

	esiServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTooManyRequests)
		_, _ = w.Write([]byte(`{"error":"rate limited"}`))
	}))
	defer esiServer.Close()

	global.Config = &config.Config{}
	config.ApplyDefaults(global.Config)
	global.Config.EveSSO.ESIBaseURL = esiServer.URL
	global.Config.EveSSO.ESIAPIPrefix = ""

	svc := NewSysConfigService()
	_, err := svc.SearchCorporations(context.Background(), "Pivot Quantum")
	if err == nil {
		t.Fatal("expected error from ESI failure")
	}
}

func TestQQGovernanceListPoliciesResolvesAllowedCorporationNames(t *testing.T) {
	db := setupQQGovernancePolicyTestDB(t)
	origDB, origCfg, origLogger := global.DB, global.Config, global.Logger
	global.DB = db
	global.Logger = zap.NewNop()
	defer func() {
		global.DB, global.Config, global.Logger = origDB, origCfg, origLogger
	}()

	cachedNameRow := model.EveEntityNameCache{
		EntityID:       98_001,
		EntityType:     model.EntityNameTypeCorporation,
		Name:           "Cached Corp",
		Source:         model.EntityNameSourceESI,
		LastResolvedAt: time.Now(),
		ExpiresAt:      time.Now().Add(time.Hour),
	}
	if err := db.Create(&cachedNameRow).Error; err != nil {
		t.Fatalf("seed cached name: %v", err)
	}
	policyJSON := model.QQGroupGovernancePolicy{
		GroupID:                   42,
		Enabled:                   true,
		AllowedCorporationIDsJSON: "[98001,98002]",
		AllowedRoleCodesJSON:      "[]",
		MemberViolationPolicy:     model.QQGovernanceViolationReview,
		UpdatedBy:                 1,
	}
	if err := db.Create(&policyJSON).Error; err != nil {
		t.Fatalf("seed policy: %v", err)
	}

	var requests int
	esiServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`[{"id":98002,"name":"Fresh Corp","category":"corporation"}]`))
	}))
	defer esiServer.Close()

	global.Config = &config.Config{}
	config.ApplyDefaults(global.Config)
	global.Config.EveSSO.ESIBaseURL = esiServer.URL
	global.Config.EveSSO.ESIAPIPrefix = ""

	svc := NewQQGovernanceServiceWithRepository(repository.NewQQGovernanceRepositoryWithDB(db))
	policies, err := svc.ListPolicies(context.Background())
	if err != nil {
		t.Fatalf("list policies: %v", err)
	}
	if len(policies) != 1 {
		t.Fatalf("want 1 policy, got %d", len(policies))
	}
	view := policies[0]
	if len(view.AllowedCorporations) != 2 {
		t.Fatalf("allowed corporations = %d, want 2", len(view.AllowedCorporations))
	}
	byID := make(map[int64]string, len(view.AllowedCorporations))
	for _, corp := range view.AllowedCorporations {
		byID[corp.CorporationID] = corp.CorporationName
	}
	if byID[98001] != "Cached Corp" {
		t.Fatalf("98001 name = %q, want %q", byID[98001], "Cached Corp")
	}
	if byID[98002] != "Fresh Corp" {
		t.Fatalf("98002 name = %q, want %q", byID[98002], "Fresh Corp")
	}
}

func TestQQGovernanceListPoliciesFallsBackWhenESIFails(t *testing.T) {
	db := setupQQGovernancePolicyTestDB(t)
	origDB, origCfg, origLogger := global.DB, global.Config, global.Logger
	global.DB = db
	global.Logger = zap.NewNop()
	defer func() {
		global.DB, global.Config, global.Logger = origDB, origCfg, origLogger
	}()

	policyJSON := model.QQGroupGovernancePolicy{
		GroupID:                   43,
		Enabled:                   true,
		AllowedCorporationIDsJSON: "[98003]",
		AllowedRoleCodesJSON:      "[]",
		MemberViolationPolicy:     model.QQGovernanceViolationReview,
		UpdatedBy:                 1,
	}
	if err := db.Create(&policyJSON).Error; err != nil {
		t.Fatalf("seed policy: %v", err)
	}

	esiServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer esiServer.Close()

	global.Config = &config.Config{}
	config.ApplyDefaults(global.Config)
	global.Config.EveSSO.ESIBaseURL = esiServer.URL
	global.Config.EveSSO.ESIAPIPrefix = ""

	svc := NewQQGovernanceServiceWithRepository(repository.NewQQGovernanceRepositoryWithDB(db))
	policies, err := svc.ListPolicies(context.Background())
	if err != nil {
		t.Fatalf("list policies with failing ESI: %v", err)
	}
	if len(policies) != 1 || len(policies[0].AllowedCorporations) != 1 {
		t.Fatalf("policies = %+v", policies)
	}
	corp := policies[0].AllowedCorporations[0]
	if corp.CorporationID != 98003 {
		t.Fatalf("corp id = %d, want 98003", corp.CorporationID)
	}
	if corp.CorporationName != "" {
		t.Fatalf("expected empty fallback name, got %q", corp.CorporationName)
	}
}

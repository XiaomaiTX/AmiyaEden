package repository

import (
	"amiya-eden/global"
	"amiya-eden/internal/model"
	"encoding/json"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"go.uber.org/zap"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func newSDERepoTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := fmt.Sprintf("file:sde_repo_%d?mode=memory&cache=shared", time.Now().UnixNano())
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&model.SystemConfig{}); err != nil {
		t.Fatalf("auto migrate system config: %v", err)
	}
	return db
}

func resetSDEQueryErrorStateForTest() {
	sdeQueryErrorState.mu.Lock()
	defer sdeQueryErrorState.mu.Unlock()
	sdeQueryErrorState.lastSeen = make(map[string]time.Time)
}

func fetchSDEStatusFromDB(t *testing.T, db *gorm.DB) map[string]interface{} {
	t.Helper()
	var cfg model.SystemConfig
	if err := db.Where("key = ?", model.SysConfigSDEStatus).First(&cfg).Error; err != nil {
		t.Fatalf("fetch sde status: %v", err)
	}
	result := map[string]interface{}{}
	if err := json.Unmarshal([]byte(cfg.Value), &result); err != nil {
		t.Fatalf("unmarshal sde status: %v", err)
	}
	return result
}

func TestReportSDEQueryErrorWritesStatusSnapshot(t *testing.T) {
	global.SetLogger(zap.NewNop())
	resetSDEQueryErrorStateForTest()

	db := newSDERepoTestDB(t)
	oldDB := global.DB
	global.DB = db
	t.Cleanup(func() { global.DB = oldDB })

	reportSDEQueryError("GetTypes", errors.New("boom"))

	status := fetchSDEStatusFromDB(t, db)
	if status["last_query_error"] != "boom" {
		t.Fatalf("last_query_error = %#v, want boom", status["last_query_error"])
	}
	if status["last_query_error_source"] != "sde_repository.GetTypes" {
		t.Fatalf("last_query_error_source = %#v", status["last_query_error_source"])
	}
	if status["last_query_error_at"] == nil {
		t.Fatalf("last_query_error_at missing")
	}
}

func TestReportSDEQueryErrorDoesNotOverwriteExistingCheckAndUpdateFields(t *testing.T) {
	global.SetLogger(zap.NewNop())
	resetSDEQueryErrorStateForTest()

	db := newSDERepoTestDB(t)
	oldDB := global.DB
	global.DB = db
	t.Cleanup(func() { global.DB = oldDB })

	seed := map[string]interface{}{
		"last_check_at":       float64(10),
		"last_check_error":    "check_err",
		"last_update_at":      float64(20),
		"last_update_error":   "update_err",
		"last_update_success": false,
	}
	seedBytes, _ := json.Marshal(seed)
	if err := db.Create(&model.SystemConfig{
		Key:   model.SysConfigSDEStatus,
		Value: string(seedBytes),
		Desc:  "seed",
	}).Error; err != nil {
		t.Fatalf("seed status: %v", err)
	}

	reportSDEQueryError("GetNames", errors.New("lookup failed"))

	status := fetchSDEStatusFromDB(t, db)
	if status["last_check_error"] != "check_err" || int(status["last_check_at"].(float64)) != 10 {
		t.Fatalf("check fields changed unexpectedly: %#v", status)
	}
	if status["last_update_error"] != "update_err" || int(status["last_update_at"].(float64)) != 20 {
		t.Fatalf("update fields changed unexpectedly: %#v", status)
	}
	if status["last_query_error"] != "lookup failed" {
		t.Fatalf("last_query_error = %#v", status["last_query_error"])
	}
}

func TestReportSDEQueryErrorThrottle(t *testing.T) {
	global.SetLogger(zap.NewNop())
	resetSDEQueryErrorStateForTest()

	db := newSDERepoTestDB(t)
	oldDB := global.DB
	global.DB = db
	t.Cleanup(func() { global.DB = oldDB })

	reportSDEQueryError("GetFlags", errors.New("same error"))
	first := fetchSDEStatusFromDB(t, db)
	firstAt := int64(first["last_query_error_at"].(float64))

	time.Sleep(10 * time.Millisecond)
	reportSDEQueryError("GetFlags", errors.New("same error"))
	second := fetchSDEStatusFromDB(t, db)
	secondAt := int64(second["last_query_error_at"].(float64))

	if secondAt != firstAt {
		t.Fatalf("expected throttled timestamp unchanged, got %d -> %d", firstAt, secondAt)
	}
}

func TestGetTypesFinalErrorReportsThroughRepositoryHook(t *testing.T) {
	global.SetLogger(zap.NewNop())
	resetSDEQueryErrorStateForTest()

	db := newSDERepoTestDB(t)
	oldDB := global.DB
	global.DB = db
	t.Cleanup(func() { global.DB = oldDB })

	repo := NewSdeRepository()
	if _, err := repo.GetTypes([]int{1}, nil, "en"); err == nil {
		t.Fatalf("expected GetTypes error on sqlite without SDE tables")
	}

	status := fetchSDEStatusFromDB(t, db)
	if status["last_query_error_source"] != "sde_repository.GetTypes" {
		t.Fatalf("last_query_error_source = %#v", status["last_query_error_source"])
	}
}

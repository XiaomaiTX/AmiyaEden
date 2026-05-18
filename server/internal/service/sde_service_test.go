package service

import (
	"amiya-eden/global"
	"amiya-eden/internal/model"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"go.uber.org/zap"
)

func TestImportSQLHandlesMultilineTranslation(t *testing.T) {
	global.SetLogger(zap.NewNop())

	db := newServiceTestDB(t, "sde-import", &model.SystemConfig{})
	raw, err := db.DB()
	if err != nil {
		t.Fatalf("db handle: %v", err)
	}
	if _, err := raw.Exec(`CREATE TABLE "trnTranslations" ("keyID" integer, "languageID" text, "tcID" integer, "text" text)`); err != nil {
		t.Fatalf("create trnTranslations: %v", err)
	}

	oldDB := global.DB
	global.DB = db
	t.Cleanup(func() { global.DB = oldDB })

	tmpDir := t.TempDir()
	sqlPath := filepath.Join(tmpDir, "sde.sql")
	sql := strings.Join([]string{
		`INSERT INTO "trnTranslations" ("keyID", "languageID", "tcID", "text") VALUES`,
		`(3411, 'en', 8, 'Cybernetics'),`,
		`(3411, 'ru', 8, 'Навык эксплуатации разведсистем.`,
		`За каждую степень освоения навыка: на 5% повышается чувствительность бортовой аппаратуры разведзондов;'),`,
		`(3412, 'en', 8, 'O''Brien');`,
	}, "\n")
	if err := os.WriteFile(sqlPath, []byte(sql), 0644); err != nil {
		t.Fatalf("write sql file: %v", err)
	}

	if err := importSQL(sqlPath); err != nil {
		t.Fatalf("importSQL: %v", err)
	}

	type row struct {
		KeyID      int
		LanguageID string
		Text       string
	}
	var rows []row
	if err := db.Raw(`SELECT "keyID", "languageID", "text" FROM "trnTranslations" ORDER BY "languageID"`).Scan(&rows).Error; err != nil {
		t.Fatalf("query translations: %v", err)
	}
	if len(rows) != 3 {
		t.Fatalf("row count = %d, want 3", len(rows))
	}
	if !strings.Contains(rows[2].Text, "\n") {
		t.Fatalf("expected multiline translation text, got %q", rows[2].Text)
	}
}

func TestExtractSQLWithDepthLimit(t *testing.T) {
	global.SetLogger(zap.NewNop())

	tmpDir := t.TempDir()
	sqlPath := filepath.Join(tmpDir, "a.sql")
	if err := os.WriteFile(sqlPath, []byte("select 1;"), 0644); err != nil {
		t.Fatalf("write sql file: %v", err)
	}

	_, err := extractSQLWithDepth(sqlPath, tmpDir, maxExtractDepth+1)
	if err == nil {
		t.Fatalf("expected depth limit error")
	}
}

func TestGetStatusSnapshotInvalidJSONReturnsDefault(t *testing.T) {
	global.SetLogger(zap.NewNop())

	db := newServiceTestDB(t, "sde-status", &model.SystemConfig{})
	oldDB := global.DB
	global.DB = db
	t.Cleanup(func() { global.DB = oldDB })

	if err := db.Create(&model.SystemConfig{
		Key:   model.SysConfigSDEStatus,
		Value: "{",
		Desc:  "broken",
	}).Error; err != nil {
		t.Fatalf("seed system config: %v", err)
	}

	svc := NewSdeService()
	status, err := svc.getStatusSnapshot()
	if err != nil {
		t.Fatalf("getStatusSnapshot error: %v", err)
	}
	if status != (SDEStatus{}) {
		t.Fatalf("status = %#v, want zero value", status)
	}
}

func TestVerifySDEDataAvailabilitySuccess(t *testing.T) {
	global.SetLogger(zap.NewNop())

	db := newServiceTestDB(t, "sde-health")
	raw, err := db.DB()
	if err != nil {
		t.Fatalf("db handle: %v", err)
	}
	for _, ddl := range []string{
		`CREATE TABLE "invTypes" ("typeID" integer, "published" boolean)`,
		`CREATE TABLE "invGroups" ("groupID" integer)`,
		`CREATE TABLE "trnTranslations" ("keyID" integer)`,
		`INSERT INTO "invTypes" ("typeID", "published") VALUES (1, true)`,
	} {
		if _, err := raw.Exec(ddl); err != nil {
			t.Fatalf("exec ddl: %v", err)
		}
	}

	oldDB := global.DB
	global.DB = db
	t.Cleanup(func() { global.DB = oldDB })

	svc := NewSdeService()
	if err := svc.verifySDEDataAvailability(); err != nil {
		t.Fatalf("verifySDEDataAvailability: %v", err)
	}
}

func TestVerifySDEDataAvailabilityFailsWhenCoreTableMissing(t *testing.T) {
	global.SetLogger(zap.NewNop())

	db := newServiceTestDB(t, "sde-health-missing")
	raw, err := db.DB()
	if err != nil {
		t.Fatalf("db handle: %v", err)
	}
	for _, ddl := range []string{
		`CREATE TABLE "invTypes" ("typeID" integer, "published" boolean)`,
		`CREATE TABLE "invGroups" ("groupID" integer)`,
		`INSERT INTO "invTypes" ("typeID", "published") VALUES (1, true)`,
	} {
		if _, err := raw.Exec(ddl); err != nil {
			t.Fatalf("exec ddl: %v", err)
		}
	}

	oldDB := global.DB
	global.DB = db
	t.Cleanup(func() { global.DB = oldDB })

	svc := NewSdeService()
	if err := svc.verifySDEDataAvailability(); err == nil {
		t.Fatalf("expected verifySDEDataAvailability to fail when trnTranslations is missing")
	}
}

func TestReportQueryErrorUpdatesSnapshotWithoutOverwritingCheckAndUpdate(t *testing.T) {
	global.SetLogger(zap.NewNop())

	db := newServiceTestDB(t, "sde-query-error", &model.SystemConfig{}, &model.SdeVersion{})
	oldDB := global.DB
	global.DB = db
	t.Cleanup(func() { global.DB = oldDB })

	seed := SDEStatus{
		CurrentVersion:    "v1",
		LatestVersion:     "v2",
		HasUpdate:         true,
		LastCheckAt:       10,
		LastCheckSuccess:  true,
		LastCheckError:    "check_err",
		LastUpdateAt:      20,
		LastUpdateSuccess: false,
		LastUpdateError:   "update_err",
	}
	svc := NewSdeService()
	if err := svc.setStatusSnapshot(seed); err != nil {
		t.Fatalf("setStatusSnapshot: %v", err)
	}

	svc.ReportQueryError("npc_kill", "GetSolarSystemNames", testErr("query failed"))

	got, err := svc.getStatusSnapshot()
	if err != nil {
		t.Fatalf("getStatusSnapshot: %v", err)
	}
	if got.LastQueryError != "query failed" {
		t.Fatalf("LastQueryError = %q, want %q", got.LastQueryError, "query failed")
	}
	if got.LastQuerySource != "npc_kill.GetSolarSystemNames" {
		t.Fatalf("LastQuerySource = %q", got.LastQuerySource)
	}
	if got.LastQueryErrorAt <= 0 {
		t.Fatalf("LastQueryErrorAt = %d, want > 0", got.LastQueryErrorAt)
	}
	if got.LastCheckAt != seed.LastCheckAt || got.LastCheckError != seed.LastCheckError {
		t.Fatalf("check snapshot changed unexpectedly: %#v", got)
	}
	if got.LastUpdateAt != seed.LastUpdateAt || got.LastUpdateError != seed.LastUpdateError {
		t.Fatalf("update snapshot changed unexpectedly: %#v", got)
	}
}

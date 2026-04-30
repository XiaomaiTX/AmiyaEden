package service

import (
	"amiya-eden/global"
	"amiya-eden/internal/model"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestAuditArchiveServiceRunDailyArchiveArchivesAndPurges(t *testing.T) {
	db := newServiceTestDB(t, "audit_archive", &model.AuditEvent{})
	prevDB := global.DB
	global.DB = db
	t.Cleanup(func() { global.DB = prevDB })

	oldEvent := model.AuditEvent{
		EventID:    "evt-old-1",
		OccurredAt: time.Now().AddDate(0, 0, -120),
		Category:   "config",
		Action:     "x",
		Result:     model.AuditResultSuccess,
	}
	newEvent := model.AuditEvent{
		EventID:    "evt-new-1",
		OccurredAt: time.Now().AddDate(0, 0, -1),
		Category:   "config",
		Action:     "y",
		Result:     model.AuditResultSuccess,
	}
	if err := db.Create(&oldEvent).Error; err != nil {
		t.Fatalf("create old event: %v", err)
	}
	if err := db.Create(&newEvent).Error; err != nil {
		t.Fatalf("create new event: %v", err)
	}

	tmpDir := t.TempDir()
	oldDir := auditArchiveStorageDir
	auditArchiveStorageDir = filepath.Join(tmpDir, "archive")
	t.Cleanup(func() { auditArchiveStorageDir = oldDir })

	svc := NewAuditArchiveService()
	summary, err := svc.RunDailyArchive(t.Context())
	if err != nil {
		t.Fatalf("RunDailyArchive() error = %v", err)
	}
	if summary.ArchivedRows != 1 || summary.PurgedRows != 1 {
		t.Fatalf("unexpected summary: %+v", summary)
	}
	if summary.FilePath == "" {
		t.Fatalf("expected archive file path, got empty")
	}
	if _, err := os.Stat(summary.FilePath); err != nil {
		t.Fatalf("archive file not found: %v", err)
	}

	var remaining int64
	if err := db.Model(&model.AuditEvent{}).Where("event_id = ?", "evt-old-1").Count(&remaining).Error; err != nil {
		t.Fatalf("count old rows: %v", err)
	}
	if remaining != 0 {
		t.Fatalf("expected old rows to be purged, got %d", remaining)
	}
}

func TestAuditArchiveServiceRunDailyArchiveIsIdempotentOnEmptySet(t *testing.T) {
	db := newServiceTestDB(t, "audit_archive_empty", &model.AuditEvent{})
	prevDB := global.DB
	global.DB = db
	t.Cleanup(func() { global.DB = prevDB })

	tmpDir := t.TempDir()
	oldDir := auditArchiveStorageDir
	auditArchiveStorageDir = filepath.Join(tmpDir, "archive")
	t.Cleanup(func() { auditArchiveStorageDir = oldDir })

	svc := NewAuditArchiveService()
	summary, err := svc.RunDailyArchive(t.Context())
	if err != nil {
		t.Fatalf("RunDailyArchive() error = %v", err)
	}
	if summary.ArchivedRows != 0 || summary.PurgedRows != 0 || summary.FilePath != "" {
		t.Fatalf("unexpected summary for empty archive: %+v", summary)
	}
}

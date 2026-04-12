package repository

import (
	"amiya-eden/global"
	"amiya-eden/internal/model"
	"fmt"
	"testing"
	"time"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func newRecruitTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := fmt.Sprintf("file:recruit_test_%d?mode=memory&cache=shared", time.Now().UnixNano())
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&model.NewbroRecruitment{}, &model.NewbroRecruitmentEntry{}); err != nil {
		t.Fatalf("auto migrate: %v", err)
	}
	return db
}

func TestNewbroRecruitmentRepository_GetLatestByUserID_ReturnsNilWhenNone(t *testing.T) {
	db := newRecruitTestDB(t)
	oldDB := global.DB
	global.DB = db
	t.Cleanup(func() { global.DB = oldDB })

	repo := NewNewbroRecruitmentRepository()
	rec, err := repo.GetLatestByUserID(1)
	if err != nil {
		t.Fatalf("GetLatestByUserID() error = %v", err)
	}
	if rec != nil {
		t.Fatalf("expected nil, got %+v", rec)
	}
}

func TestNewbroRecruitmentRepository_CreateAndGetByCode(t *testing.T) {
	db := newRecruitTestDB(t)
	oldDB := global.DB
	global.DB = db
	t.Cleanup(func() { global.DB = oldDB })

	repo := NewNewbroRecruitmentRepository()
	rec := &model.NewbroRecruitment{
		UserID:      10,
		Code:        "abc",
		GeneratedAt: time.Now(),
	}
	if err := repo.Create(rec); err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if rec.ID == 0 {
		t.Fatal("expected ID to be set after Create")
	}

	found, err := repo.GetByCode("abc")
	if err != nil {
		t.Fatalf("GetByCode() error = %v", err)
	}
	if found.UserID != 10 {
		t.Fatalf("GetByCode().UserID = %d, want 10", found.UserID)
	}
}

func TestNewbroRecruitmentRepository_CreateEntry(t *testing.T) {
	db := newRecruitTestDB(t)
	oldDB := global.DB
	global.DB = db
	t.Cleanup(func() { global.DB = oldDB })

	repo := NewNewbroRecruitmentRepository()
	rec := &model.NewbroRecruitment{UserID: 5, Code: "xyz", GeneratedAt: time.Now()}
	if err := repo.Create(rec); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	entry := &model.NewbroRecruitmentEntry{
		RecruitmentID: rec.ID,
		QQ:            "12345",
		EnteredAt:     time.Now(),
		Status:        model.RecruitEntryStatusOngoing,
	}
	if err := repo.CreateEntry(entry); err != nil {
		t.Fatalf("CreateEntry() error = %v", err)
	}
	if entry.ID == 0 {
		t.Fatal("expected entry ID to be set")
	}
}

func TestNewbroRecruitmentRepository_CreateEntryRejectsDuplicateRecruitmentQQ(t *testing.T) {
	db := newRecruitTestDB(t)
	oldDB := global.DB
	global.DB = db
	t.Cleanup(func() { global.DB = oldDB })

	repo := NewNewbroRecruitmentRepository()
	rec := &model.NewbroRecruitment{UserID: 5, Code: "dup", GeneratedAt: time.Now()}
	if err := repo.Create(rec); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	first := &model.NewbroRecruitmentEntry{
		RecruitmentID: rec.ID,
		QQ:            "12345",
		EnteredAt:     time.Now(),
		Status:        model.RecruitEntryStatusOngoing,
	}
	if err := repo.CreateEntry(first); err != nil {
		t.Fatalf("CreateEntry(first) error = %v", err)
	}

	duplicate := &model.NewbroRecruitmentEntry{
		RecruitmentID: rec.ID,
		QQ:            "12345",
		EnteredAt:     time.Now().Add(time.Minute),
		Status:        model.RecruitEntryStatusOngoing,
	}
	if err := repo.CreateEntry(duplicate); err == nil {
		t.Fatal("expected duplicate recruitment QQ entry to fail")
	}
}


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

func newFuxiAdminRepoTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := fmt.Sprintf("file:fuxi_admin_repo_%d?mode=memory&cache=shared", time.Now().UnixNano())
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&model.FuxiAdminConfig{}, &model.FuxiAdminTier{}, &model.FuxiAdmin{}); err != nil {
		t.Fatalf("auto migrate: %v", err)
	}
	return db
}

func TestFuxiAdminRepoUpsertConfigCreatesDefault(t *testing.T) {
	db := newFuxiAdminRepoTestDB(t)
	global.DB = db
	repo := NewFuxiAdminRepository()

	cfg := model.DefaultFuxiAdminConfig()
	if err := repo.UpsertConfig(&cfg); err != nil {
		t.Fatalf("upsert config: %v", err)
	}

	got, err := repo.GetConfig()
	if err != nil {
		t.Fatalf("get config: %v", err)
	}
	if got == nil {
		t.Fatal("expected config, got nil")
	}
	if got.BaseFontSize != 14 {
		t.Fatalf("expected base_font_size 14, got %d", got.BaseFontSize)
	}
}

func TestFuxiAdminRepoListAdminsByTierReturnsOnlyMatchingTier(t *testing.T) {
	db := newFuxiAdminRepoTestDB(t)
	global.DB = db
	repo := NewFuxiAdminRepository()

	tier1 := model.FuxiAdminTier{Name: "High"}
	tier2 := model.FuxiAdminTier{Name: "Low"}
	db.Create(&tier1)
	db.Create(&tier2)

	admin1 := model.FuxiAdmin{TierID: tier1.ID, Name: "Alpha"}
	admin2 := model.FuxiAdmin{TierID: tier2.ID, Name: "Beta"}
	db.Create(&admin1)
	db.Create(&admin2)

	got, err := repo.ListAdminsByTierIDs([]uint{tier1.ID})
	if err != nil {
		t.Fatalf("list admins: %v", err)
	}
	if len(got) != 1 || got[0].Name != "Alpha" {
		t.Fatalf("expected 1 admin Alpha, got %+v", got)
	}
}

func TestFuxiAdminRepoDeleteAdminsByTierIDRemovesAll(t *testing.T) {
	db := newFuxiAdminRepoTestDB(t)
	global.DB = db
	repo := NewFuxiAdminRepository()

	tier := model.FuxiAdminTier{Name: "Mid"}
	db.Create(&tier)
	db.Create(&model.FuxiAdmin{TierID: tier.ID, Name: "X"})
	db.Create(&model.FuxiAdmin{TierID: tier.ID, Name: "Y"})

	if err := repo.DeleteAdminsByTierID(tier.ID); err != nil {
		t.Fatalf("delete by tier: %v", err)
	}

	got, _ := repo.ListAdminsByTierIDs([]uint{tier.ID})
	if len(got) != 0 {
		t.Fatalf("expected 0 admins after delete, got %d", len(got))
	}
}

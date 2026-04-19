package repository

import (
	"amiya-eden/global"
	"amiya-eden/internal/model"
	"fmt"
	"reflect"
	"testing"
	"time"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func useFuxiAdminAuditCountRepoTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	dsn := fmt.Sprintf("file:fuxi_admin_audit_counts_repo_%d?mode=memory&cache=shared", time.Now().UnixNano())
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(
		&model.User{},
		&model.EveCharacter{},
		&model.Fleet{},
		&model.WelfareApplication{},
		&model.ShopOrder{},
	); err != nil {
		t.Fatalf("auto migrate: %v", err)
	}

	oldDB := global.DB
	global.DB = db
	t.Cleanup(func() { global.DB = oldDB })

	return db
}

func TestFleetRepositoryCountByCreatorUserIDs(t *testing.T) {
	useFuxiAdminAuditCountRepoTestDB(t)
	repo := NewFleetRepository()
	deletedAt := time.Now()

	fleets := []model.Fleet{
		{ID: "fleet-1", Title: "One", StartAt: time.Now(), EndAt: time.Now().Add(time.Hour), Importance: model.FleetImportanceOther, FCUserID: 11, FCCharacterID: 1001, FCCharacterName: "Alpha"},
		{ID: "fleet-2", Title: "Two", StartAt: time.Now(), EndAt: time.Now().Add(time.Hour), Importance: model.FleetImportanceOther, FCUserID: 11, FCCharacterID: 1001, FCCharacterName: "Alpha"},
		{ID: "fleet-3", Title: "Three", StartAt: time.Now(), EndAt: time.Now().Add(time.Hour), Importance: model.FleetImportanceOther, FCUserID: 22, FCCharacterID: 2002, FCCharacterName: "Beta"},
		{ID: "fleet-4", Title: "Deleted", StartAt: time.Now(), EndAt: time.Now().Add(time.Hour), Importance: model.FleetImportanceOther, FCUserID: 11, FCCharacterID: 1001, FCCharacterName: "Alpha", DeletedAt: &deletedAt},
	}
	if err := global.DB.Create(&fleets).Error; err != nil {
		t.Fatalf("seed fleets: %v", err)
	}

	tests := []struct {
		name    string
		userIDs []uint
		want    map[uint]int64
	}{
		{
			name:    "counts active fleets by creator",
			userIDs: []uint{11, 22, 33},
			want:    map[uint]int64{11: 2, 22: 1},
		},
		{
			name:    "returns empty map for empty input",
			userIDs: nil,
			want:    map[uint]int64{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := repo.CountByCreatorUserIDs(tt.userIDs)
			if err != nil {
				t.Fatalf("CountByCreatorUserIDs: %v", err)
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("CountByCreatorUserIDs(%v) = %#v, want %#v", tt.userIDs, got, tt.want)
			}
			if tt.name == "counts active fleets by creator" {
				if _, ok := got[33]; ok {
					t.Fatalf("expected user 33 to be absent from result map, got %#v", got)
				}
			}
		})
	}
}

func TestWelfareRepositoryCountDeliveredByReviewers(t *testing.T) {
	useFuxiAdminAuditCountRepoTestDB(t)
	repo := NewWelfareRepository()

	apps := []model.WelfareApplication{
		{WelfareID: 1, CharacterID: 1001, CharacterName: "Alpha", Status: model.WelfareAppStatusDelivered, ReviewedBy: 11},
		{WelfareID: 1, CharacterID: 1002, CharacterName: "Bravo", Status: model.WelfareAppStatusDelivered, ReviewedBy: 11},
		{WelfareID: 1, CharacterID: 1003, CharacterName: "Charlie", Status: model.WelfareAppStatusRequested, ReviewedBy: 11},
		{WelfareID: 1, CharacterID: 1004, CharacterName: "Delta", Status: model.WelfareAppStatusDelivered, ReviewedBy: 22},
	}
	if err := global.DB.Create(&apps).Error; err != nil {
		t.Fatalf("seed welfare applications: %v", err)
	}

	tests := []struct {
		name        string
		reviewerIDs []uint
		want        map[uint]int64
	}{
		{
			name:        "counts delivered applications by reviewer",
			reviewerIDs: []uint{11, 22, 33},
			want:        map[uint]int64{11: 2, 22: 1},
		},
		{
			name:        "returns empty map for empty input",
			reviewerIDs: []uint{},
			want:        map[uint]int64{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := repo.CountDeliveredByReviewers(tt.reviewerIDs)
			if err != nil {
				t.Fatalf("CountDeliveredByReviewers: %v", err)
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("CountDeliveredByReviewers(%v) = %#v, want %#v", tt.reviewerIDs, got, tt.want)
			}
			if tt.name == "counts delivered applications by reviewer" {
				if _, ok := got[33]; ok {
					t.Fatalf("expected reviewer 33 to be absent from result map, got %#v", got)
				}
			}
		})
	}
}

func TestShopRepositoryCountDeliveredByReviewers(t *testing.T) {
	useFuxiAdminAuditCountRepoTestDB(t)
	repo := NewShopRepository()
	reviewer11 := uint(11)
	reviewer22 := uint(22)

	orders := []model.ShopOrder{
		{OrderNo: "SO-1", UserID: 1, ProductID: 1, ProductName: "One", ProductType: model.ProductTypeNormal, UnitPrice: 1, TotalPrice: 1, Status: model.OrderStatusDelivered, ReviewedBy: &reviewer11},
		{OrderNo: "SO-2", UserID: 1, ProductID: 1, ProductName: "Two", ProductType: model.ProductTypeNormal, UnitPrice: 1, TotalPrice: 1, Status: model.OrderStatusDelivered, ReviewedBy: &reviewer11},
		{OrderNo: "SO-3", UserID: 1, ProductID: 1, ProductName: "Three", ProductType: model.ProductTypeNormal, UnitPrice: 1, TotalPrice: 1, Status: model.OrderStatusRequested, ReviewedBy: &reviewer11},
		{OrderNo: "SO-4", UserID: 1, ProductID: 1, ProductName: "Four", ProductType: model.ProductTypeNormal, UnitPrice: 1, TotalPrice: 1, Status: model.OrderStatusDelivered, ReviewedBy: &reviewer22},
	}
	if err := global.DB.Create(&orders).Error; err != nil {
		t.Fatalf("seed shop orders: %v", err)
	}

	tests := []struct {
		name        string
		reviewerIDs []uint
		want        map[uint]int64
	}{
		{
			name:        "counts delivered orders by reviewer",
			reviewerIDs: []uint{11, 22, 33},
			want:        map[uint]int64{11: 2, 22: 1},
		},
		{
			name:        "returns empty map for empty input",
			reviewerIDs: nil,
			want:        map[uint]int64{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := repo.CountDeliveredByReviewers(tt.reviewerIDs)
			if err != nil {
				t.Fatalf("CountDeliveredByReviewers: %v", err)
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("CountDeliveredByReviewers(%v) = %#v, want %#v", tt.reviewerIDs, got, tt.want)
			}
			if tt.name == "counts delivered orders by reviewer" {
				if _, ok := got[33]; ok {
					t.Fatalf("expected reviewer 33 to be absent from result map, got %#v", got)
				}
			}
		})
	}
}

func TestUserRepositoryListByPrimaryCharacterIDs(t *testing.T) {
	useFuxiAdminAuditCountRepoTestDB(t)
	repo := NewUserRepository()

	users := []model.User{
		{Nickname: "Alpha", PrimaryCharacterID: 1001},
		{Nickname: "Bravo", PrimaryCharacterID: 2002},
	}
	if err := global.DB.Create(&users).Error; err != nil {
		t.Fatalf("seed users: %v", err)
	}

	tests := []struct {
		name         string
		characterIDs []int64
		want         map[int64]uint
	}{
		{
			name:         "returns character to user map",
			characterIDs: []int64{1001, 2002, 3003},
			want:         map[int64]uint{1001: users[0].ID, 2002: users[1].ID},
		},
		{
			name:         "returns empty map for empty input",
			characterIDs: nil,
			want:         map[int64]uint{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := repo.ListByPrimaryCharacterIDs(tt.characterIDs)
			if err != nil {
				t.Fatalf("ListByPrimaryCharacterIDs: %v", err)
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("ListByPrimaryCharacterIDs(%v) = %#v, want %#v", tt.characterIDs, got, tt.want)
			}
		})
	}
}

func TestEveCharacterRepositoryListUserIDsByCharacterIDs(t *testing.T) {
	useFuxiAdminAuditCountRepoTestDB(t)
	repo := NewEveCharacterRepository()

	chars := []model.EveCharacter{
		{CharacterID: 1001, CharacterName: "Alpha", UserID: 11},
		{CharacterID: 2002, CharacterName: "Bravo", UserID: 22},
	}
	if err := global.DB.Create(&chars).Error; err != nil {
		t.Fatalf("seed eve characters: %v", err)
	}

	tests := []struct {
		name         string
		characterIDs []int64
		want         map[int64]uint
	}{
		{
			name:         "returns character to user map for bound characters",
			characterIDs: []int64{1001, 2002, 3003},
			want:         map[int64]uint{1001: 11, 2002: 22},
		},
		{
			name:         "returns empty map for empty input",
			characterIDs: nil,
			want:         map[int64]uint{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := repo.ListUserIDsByCharacterIDs(tt.characterIDs)
			if err != nil {
				t.Fatalf("ListUserIDsByCharacterIDs: %v", err)
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("ListUserIDsByCharacterIDs(%v) = %#v, want %#v", tt.characterIDs, got, tt.want)
			}
		})
	}
}

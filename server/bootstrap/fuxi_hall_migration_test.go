package bootstrap

import (
	"amiya-eden/internal/model"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"testing"
)

func TestFuxiHallMigrationPreservesCardsAndCuration(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err = db.AutoMigrate(&model.FuxiHallPage{}, &model.FuxiHallCard{}); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 40; i++ {
		key := "contributors"
		if i == 0 {
			key = "leadership"
		}
		card := model.FuxiHallCard{PageKey: key, Nickname: "Legacy", MainCharacterID: int64(i + 1), TitleTags: []string{"Title"}, DescriptionHTML: "<p>Preserve</p>", SortOrder: i + 1}
		if err = db.Create(&card).Error; err != nil {
			t.Fatal(err)
		}
	}
	if err = migrateFuxiHall(db); err != nil {
		t.Fatal(err)
	}
	var page model.FuxiHallPage
	if err = db.Where("page_key = ?", "hall-of-fame").First(&page).Error; err != nil {
		t.Fatal(err)
	}
	if page.Floors[2].Capacity != 40 {
		t.Fatal("migration capacity lost legacy cards")
	}
	var cards []model.FuxiHallCard
	if err = db.Order("id").Find(&cards).Error; err != nil {
		t.Fatal(err)
	}
	for i, card := range cards {
		if card.ID != uint(i+1) || card.PageKey != "hall-of-fame" || card.DescriptionHTML != "<p>Preserve</p>" || card.IsCurrentManagement != (i == 0) || card.FloorKey != "first" {
			t.Fatalf("legacy card changed: %+v", card)
		}
	}
	page.Floors[0].Capacity = 10
	if err = db.Save(&page).Error; err != nil {
		t.Fatal(err)
	}
	if err = db.Model(&cards[0]).Updates(map[string]any{"floor_key": "top", "is_current_management": false}).Error; err != nil {
		t.Fatal(err)
	}
	if err = migrateFuxiHall(db); err != nil {
		t.Fatal(err)
	}
	if err = db.First(&cards[0], cards[0].ID).Error; err != nil {
		t.Fatal(err)
	}
	if cards[0].FloorKey != "top" || cards[0].IsCurrentManagement {
		t.Fatal("restart overwrote curation")
	}
	if err = db.First(&page, page.ID).Error; err != nil {
		t.Fatal(err)
	}
	if page.Floors[0].Capacity != 10 {
		t.Fatal("restart overwrote floor configuration")
	}
}

func TestFuxiHallMigrationAvoidsSortOrderCollisionsBetweenLegacyGroups(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err = db.AutoMigrate(&model.FuxiHallPage{}, &model.FuxiHallCard{}); err != nil {
		t.Fatal(err)
	}
	legacy := []model.FuxiHallCard{
		{PageKey: "contributors", Nickname: "Contrib-1", MainCharacterID: 101, SortOrder: 1},
		{PageKey: "leadership", Nickname: "Leader-1", MainCharacterID: 102, SortOrder: 1},
		{PageKey: "contributors", Nickname: "Contrib-2", MainCharacterID: 103, SortOrder: 2},
		{PageKey: "leadership", Nickname: "Leader-2", MainCharacterID: 104, SortOrder: 2},
	}
	for i := range legacy {
		if err = db.Create(&legacy[i]).Error; err != nil {
			t.Fatal(err)
		}
	}
	if err = migrateFuxiHall(db); err != nil {
		t.Fatal(err)
	}
	var migrated []model.FuxiHallCard
	if err = db.Where("page_key = ?", "hall-of-fame").Order("sort_order ASC, id ASC").Find(&migrated).Error; err != nil {
		t.Fatal(err)
	}
	wantNames := []string{"Leader-1", "Leader-2", "Contrib-1", "Contrib-2"}
	if len(migrated) != len(wantNames) {
		t.Fatalf("migrated len = %d, want %d", len(migrated), len(wantNames))
	}
	for i, card := range migrated {
		if card.Nickname != wantNames[i] || card.SortOrder != i+1 {
			t.Fatalf("card[%d] = (%q, sort_order=%d), want (%q, sort_order=%d)", i, card.Nickname, card.SortOrder, wantNames[i], i+1)
		}
	}
}

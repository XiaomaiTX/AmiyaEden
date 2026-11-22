package bootstrap

import (
	"amiya-eden/internal/model"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// Migrate legacy cards only once; subsequent startups preserve curation.
func migrateFuxiHall(db *gorm.DB) error {
	return db.Transaction(func(tx *gorm.DB) error {
		var count int64
		if err := tx.Model(&model.FuxiHallCard{}).Where("page_key IN ?", []string{"leadership", "contributors"}).Count(&count).Error; err != nil {
			return err
		}
		floors := model.DefaultFuxiHallFloors()
		if count > int64(floors[2].Capacity) {
			floors[2].Capacity = int(count)
		}
		page := model.FuxiHallPage{PageKey: string(model.FuxiHallPageFame), Title: "伏羲名人堂", Floors: floors}
		if err := tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&page).Error; err != nil {
			return err
		}
		if count == 0 {
			return nil
		}
		var baseSortOrder int
		if err := tx.Model(&model.FuxiHallCard{}).
			Where("page_key = ?", string(model.FuxiHallPageFame)).
			Select("COALESCE(MAX(sort_order), 0)").
			Row().Scan(&baseSortOrder); err != nil {
			return err
		}
		nextSortOrder := baseSortOrder + 1
		for _, key := range []string{"leadership", "contributors"} {
			var legacyCards []model.FuxiHallCard
			if err := tx.Where("page_key = ?", key).Order("sort_order ASC, id ASC").Find(&legacyCards).Error; err != nil {
				return err
			}
			for _, card := range legacyCards {
				if err := tx.Model(&model.FuxiHallCard{}).Where("id = ?", card.ID).
					Updates(map[string]any{
						"page_key":              string(model.FuxiHallPageFame),
						"floor_key":             "first",
						"is_current_management": key == "leadership",
						"sort_order":            nextSortOrder,
					}).Error; err != nil {
					return err
				}
				nextSortOrder++
			}
		}
		return nil
	})
}

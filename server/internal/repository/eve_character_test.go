package repository

import (
	"amiya-eden/global"
	"amiya-eden/internal/model"
	"fmt"
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func TestUpdateBirthdayOnlyWritesBirthdayColumn(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(fmt.Sprintf("file:eve_char_repo_test_%d?mode=memory&cache=shared", time.Now().UnixNano())), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&model.EveCharacter{}); err != nil {
		t.Fatalf("auto migrate: %v", err)
	}

	oldDB := global.DB
	global.DB = db
	t.Cleanup(func() { global.DB = oldDB })

	const characterID = int64(4210020001)
	seed := &model.EveCharacter{
		CharacterID:  characterID,
		CharacterName: "Birthday",
		UserID:       1,
		AccessToken:  "at-seed",
		RefreshToken: "rt-seed",
		TokenExpiry:  time.Now().Add(time.Hour),
		Scopes:       "publicData esi-corporations.read_structures.v1",
	}
	if err := db.Create(seed).Error; err != nil {
		t.Fatalf("seed character: %v", err)
	}

	birthday := time.Date(2015, time.April, 1, 0, 0, 0, 0, time.UTC)
	repo := NewEveCharacterRepository()
	if err := repo.UpdateBirthday(characterID, &birthday); err != nil {
		t.Fatalf("UpdateBirthday: %v", err)
	}

	var got model.EveCharacter
	if err := db.Where("character_id = ?", characterID).First(&got).Error; err != nil {
		t.Fatalf("reload character: %v", err)
	}
	if got.Birthday == nil || !got.Birthday.Equal(birthday) {
		t.Fatalf("birthday not updated: %+v", got.Birthday)
	}
	if got.AccessToken != seed.AccessToken || got.RefreshToken != seed.RefreshToken || got.Scopes != seed.Scopes {
		t.Fatalf("token/scopes columns must stay untouched by UpdateBirthday: %+v", got)
	}
}

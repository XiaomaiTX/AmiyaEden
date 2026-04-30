package service

import (
	"amiya-eden/global"
	"amiya-eden/internal/model"
	"strings"
	"testing"

	"gorm.io/gorm"
)

func TestImpersonateUserRejectsInvalidPrimaryCharacter(t *testing.T) {
	db := newUserServiceTestDB(t)
	seedImpersonationTargetUser(t, db)

	originalDB := global.DB
	global.DB = db
	defer func() { global.DB = originalDB }()

	_, _, err := NewUserService().ImpersonateUser(1)
	if err == nil || !strings.Contains(err.Error(), "无法模拟登录") {
		t.Fatalf("expected impersonation restriction error, got %v", err)
	}
}

func newUserServiceTestDB(t *testing.T) *gorm.DB {
	db := newServiceTestDB(t, "user_service_test", &model.User{}, &model.EveCharacter{})
	return db
}

func seedImpersonationTargetUser(t *testing.T, db *gorm.DB) {
	t.Helper()

	user := model.User{
		BaseModel:          model.BaseModel{ID: 1},
		Nickname:           "Amiya",
		Role:               model.RoleUser,
		PrimaryCharacterID: 9001,
	}
	if err := db.Create(&user).Error; err != nil {
		t.Fatalf("create user: %v", err)
	}

	if err := db.Create(&model.EveCharacter{
		CharacterID:   9001,
		CharacterName: "Amiya Prime",
		UserID:        user.ID,
		TokenInvalid:  true,
	}).Error; err != nil {
		t.Fatalf("create character: %v", err)
	}
}

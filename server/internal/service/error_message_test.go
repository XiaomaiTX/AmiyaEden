package service

import (
	"amiya-eden/global"
	"amiya-eden/internal/model"
	"amiya-eden/internal/repository"
	"fmt"
	"testing"
	"time"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestFleetConfigServiceGetFleetConfigNotFoundMessage(t *testing.T) {
	db := newServiceErrorMessageTestDB(t, &model.FleetConfig{})
	useServiceErrorMessageTestDB(t, db)

	svc := &FleetConfigService{repo: repository.NewFleetConfigRepository()}
	_, err := svc.GetFleetConfig(999)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if err.Error() != "舰队配置不存在" {
		t.Fatalf("error = %q, want %q", err.Error(), "舰队配置不存在")
	}
}

func TestSkillPlanServiceGetSkillPlanNotFoundMessage(t *testing.T) {
	db := newServiceErrorMessageTestDB(t, &model.SkillPlan{})
	useServiceErrorMessageTestDB(t, db)

	svc := &SkillPlanService{repo: repository.NewSkillPlanRepository()}
	_, err := svc.GetSkillPlan(999, "zh")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if err.Error() != "技能计划不存在" {
		t.Fatalf("error = %q, want %q", err.Error(), "技能计划不存在")
	}
}

func TestContractServiceGetContractDetailNotFoundMessage(t *testing.T) {
	db := newServiceErrorMessageTestDB(t, &model.EveCharacter{}, &model.EveCharacterContract{})
	useServiceErrorMessageTestDB(t, db)

	if err := db.Create(&model.EveCharacter{
		CharacterID:   90000001,
		CharacterName: "Test Pilot",
		UserID:        42,
	}).Error; err != nil {
		t.Fatalf("create character: %v", err)
	}

	svc := &ContractService{
		charRepo:     repository.NewEveCharacterRepository(),
		contractRepo: repository.NewContractRepository(),
		sdeRepo:      repository.NewSdeRepository(),
	}
	_, err := svc.GetContractDetail(42, &InfoContractDetailRequest{
		CharacterID: 90000001,
		ContractID:  123456,
		Language:    "zh",
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if err.Error() != "合同不存在" {
		t.Fatalf("error = %q, want %q", err.Error(), "合同不存在")
	}
}

func TestNotificationServiceGetUnreadCountDoesNotLeakDatabaseError(t *testing.T) {
	db := newServiceErrorMessageTestDB(t, &model.EveCharacter{})
	useServiceErrorMessageTestDB(t, db)

	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("get sql db: %v", err)
	}
	if err := sqlDB.Close(); err != nil {
		t.Fatalf("close sql db: %v", err)
	}

	svc := &NotificationService{charRepo: repository.NewEveCharacterRepository()}
	_, err = svc.GetUnreadCount(42)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if err.Error() != "获取人物列表失败" {
		t.Fatalf("error = %q, want %q", err.Error(), "获取人物列表失败")
	}
}

func newServiceErrorMessageTestDB(t *testing.T, models ...interface{}) *gorm.DB {
	t.Helper()

	dsn := fmt.Sprintf("file:error_message_test_%d?mode=memory&cache=shared", time.Now().UnixNano())
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if len(models) > 0 {
		if err := db.AutoMigrate(models...); err != nil {
			t.Fatalf("auto migrate: %v", err)
		}
	}
	return db
}

func useServiceErrorMessageTestDB(t *testing.T, db *gorm.DB) {
	t.Helper()

	oldDB := global.DB
	global.DB = db
	t.Cleanup(func() {
		global.DB = oldDB
	})
}

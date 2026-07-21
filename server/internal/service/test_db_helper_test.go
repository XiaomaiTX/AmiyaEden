package service

import (
	"amiya-eden/global"
	"amiya-eden/internal/model"
	"fmt"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var serviceTestDBSeq uint64

func newServiceTestDB(t *testing.T, prefix string, models ...interface{}) *gorm.DB {
	t.Helper()
	clearCorpPolicyCache()

	name := strings.NewReplacer("/", "_", "\\", "_", " ", "_", ":", "_").Replace(t.Name())
	dsn := fmt.Sprintf(
		"file:%s_%s_%d_%d?mode=memory&cache=shared",
		prefix,
		name,
		time.Now().UnixNano(),
		atomic.AddUint64(&serviceTestDBSeq, 1),
	)
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
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

func seedWalletCapabilityEnabledUserForTests(t *testing.T, db *gorm.DB, userID uint, characterID int64, corporationID int64) {
	t.Helper()
	if err := db.Where("id = ?", userID).
		Assign(model.User{
			BaseModel:          model.BaseModel{ID: userID},
			Nickname:           fmt.Sprintf("user_%d", userID),
			PrimaryCharacterID: characterID,
			Role:               model.RoleUser,
		}).
		FirstOrCreate(&model.User{}).Error; err != nil {
		t.Fatalf("seed user: %v", err)
	}
	if err := db.Where("character_id = ?", characterID).
		Assign(model.EveCharacter{
			CharacterID:   characterID,
			CharacterName: fmt.Sprintf("char_%d", characterID),
			UserID:        userID,
			CorporationID: corporationID,
		}).
		FirstOrCreate(&model.EveCharacter{}).Error; err != nil {
		t.Fatalf("seed character: %v", err)
	}
}

func setWalletCapabilityPolicyForTests(t *testing.T, db *gorm.DB, corporationID int64, enabled bool) {
	t.Helper()
	previous := global.DB
	global.DB = db
	defer func() { global.DB = previous }()

	capabilities := []string{}
	if enabled {
		capabilities = []string{model.CorpCapabilityWalletUserEnabled}
	}
	if err := NewCorporationPolicyService().UpdatePolicies(CorporationPolicyConfig{
		Version:     1,
		DefaultMode: "deny",
		Policies: []CorporationPolicy{
			{
				CorporationID: corporationID,
				FullAccess:    false,
				Capabilities:  capabilities,
				Rules:         map[string]any{},
			},
		},
	}); err != nil {
		t.Fatalf("update policy: %v", err)
	}
}

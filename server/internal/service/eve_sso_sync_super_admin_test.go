package service

import (
	"amiya-eden/config"
	"amiya-eden/global"
	"amiya-eden/internal/model"
	"context"
	"fmt"
	"testing"

	"github.com/alicebob/miniredis/v2"
	"github.com/glebarez/sqlite"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// TestSyncConfigSuperAdminsInvalidatesRoleCache verifies the Stage 0A fix:
// after SyncConfigSuperAdmins grants super_admin, the cached user_roles
// entry must be cleared so subsequent requests observe the new role without
// waiting up to 30 minutes for TTL expiry.
func TestSyncConfigSuperAdminsInvalidatesRoleCache(t *testing.T) {
	mr := miniredis.RunT(t)
	originalRedis := global.Redis
	originalDB := global.DB
	originalConfig := global.Config
	originalLogger := global.Logger
	global.Redis = redis.NewClient(&redis.Options{Addr: mr.Addr()})
	global.Logger = zap.NewNop()
	defer func() {
		global.Redis = originalRedis
		global.DB = originalDB
		global.Config = originalConfig
		global.Logger = originalLogger
	}()

	db, err := gorm.Open(sqlite.Open("file:sync_super_admin_test?mode=memory&cache=shared"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&model.User{}, &model.UserRole{}, &model.EveCharacter{}); err != nil {
		t.Fatalf("auto migrate: %v", err)
	}
	global.DB = db

	const (
		userID      uint = 9001
		adminCharID      = int64(98001)
	)
	if err := db.Create(&model.User{
		BaseModel:          model.BaseModel{ID: userID},
		PrimaryCharacterID: adminCharID,
		Role:               model.RoleUser,
	}).Error; err != nil {
		t.Fatalf("seed user: %v", err)
	}
	if err := db.Create(&model.EveCharacter{
		CharacterID:   adminCharID,
		CharacterName: "Admin",
		UserID:        userID,
	}).Error; err != nil {
		t.Fatalf("seed character: %v", err)
	}

	// Configure super-admin list to include the seeded character.
	global.Config = &config.Config{}
	global.Config.App.SuperAdmins = []int64{adminCharID}

	// Seed role cache with the pre-grant snapshot (no super_admin).
	roleSvc := NewRoleService()
	ctx := context.Background()
	if _, err := roleSvc.GetUserRoleNames(ctx, userID); err != nil {
		t.Fatalf("seed role cache: %v", err)
	}
	cacheKey := fmt.Sprintf("user_roles:%d", userID)
	if !mr.Exists(cacheKey) {
		t.Fatalf("expected cached user_roles key %q before sync", cacheKey)
	}

	SyncConfigSuperAdmins(ctx, userID)

	if mr.Exists(cacheKey) {
		t.Fatalf("expected user_roles cache cleared after SyncConfigSuperAdmins, still present")
	}

	// After re-read, the roles should include super_admin.
	roles, err := roleSvc.GetUserRoleNames(ctx, userID)
	if err != nil {
		t.Fatalf("re-read roles: %v", err)
	}
	if !model.ContainsRole(roles, model.RoleSuperAdmin) {
		t.Fatalf("roles after sync = %#v, want super_admin included", roles)
	}
}

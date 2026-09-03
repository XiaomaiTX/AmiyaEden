package service

import (
	"amiya-eden/global"
	"amiya-eden/internal/model"
	"amiya-eden/internal/repository"
	"amiya-eden/pkg/eve"
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/glebarez/sqlite"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func newTokenTestEnv(t *testing.T) (*EveSSOService, *gorm.DB) {
	t.Helper()

	mr := miniredis.RunT(t)
	originalRedis := global.Redis
	originalDB := global.DB
	originalLogger := global.Logger
	global.Redis = redis.NewClient(&redis.Options{Addr: mr.Addr()})
	global.Logger = zap.NewNop()
	t.Cleanup(func() {
		global.Redis = originalRedis
		global.DB = originalDB
		global.Logger = originalLogger
	})

	db, err := gorm.Open(sqlite.Open(fmt.Sprintf("file:eve_sso_token_test_%d?mode=memory&cache=shared", time.Now().UnixNano())), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&model.EveCharacter{}); err != nil {
		t.Fatalf("auto migrate: %v", err)
	}
	global.DB = db

	svc := &EveSSOService{
		charRepo: repository.NewEveCharacterRepository(),
		userRepo: repository.NewUserRepository(),
		roleSvc:  NewRoleService(),
	}
	return svc, db
}

func TestRefreshCharacterTokenRotatesTokenWithoutOverwritingScopes(t *testing.T) {
	svc, db := newTokenTestEnv(t)

	const characterID = int64(4210000001)
	seed := &model.EveCharacter{
		CharacterID:  characterID,
		CharacterName: "Director",
		UserID:       1,
		AccessToken:  "at-old",
		RefreshToken: "rt-old",
		TokenExpiry:  time.Now().Add(-time.Minute),
		Scopes:       "publicData esi-corporations.read_structures.v1 esi-assets.read_corporation_assets.v1",
	}
	if err := db.Create(seed).Error; err != nil {
		t.Fatalf("seed character: %v", err)
	}

	tokenServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		if err := req.ParseForm(); err != nil {
			t.Fatalf("parse form: %v", err)
		}
		if got := req.FormValue("grant_type"); got != "refresh_token" {
			t.Fatalf("grant_type = %q, want refresh_token", got)
		}
		if got := req.FormValue("refresh_token"); got != "rt-old" {
			t.Fatalf("refresh_token = %q, want rt-old", got)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"access_token":"at-new","token_type":"Bearer","expires_in":1199,"refresh_token":"rt-new"}`))
	}))
	t.Cleanup(tokenServer.Close)
	svc.eveClient = eve.NewClientWithEndpoints("cid", "csecret", "http://localhost/cb", tokenServer.URL, tokenServer.URL)

	if err := svc.refreshCharacterToken(context.Background(), characterID); err != nil {
		t.Fatalf("refreshCharacterToken: %v", err)
	}

	var got model.EveCharacter
	if err := db.Where("character_id = ?", characterID).First(&got).Error; err != nil {
		t.Fatalf("reload character: %v", err)
	}
	if got.AccessToken != "at-new" || got.RefreshToken != "rt-new" {
		t.Fatalf("tokens not rotated: access=%q refresh=%q", got.AccessToken, got.RefreshToken)
	}
	if got.TokenInvalid {
		t.Fatal("expected TokenInvalid to stay false")
	}
	if time.Until(got.TokenExpiry) <= 0 {
		t.Fatalf("expected refreshed expiry in the future, got %v", got.TokenExpiry)
	}
	if got.Scopes != seed.Scopes {
		t.Fatalf("scopes overwritten by refresh: got %q, want %q", got.Scopes, seed.Scopes)
	}
}

func TestUpdateExistingCharacterLockedWaitsForLock(t *testing.T) {
	svc, db := newTokenTestEnv(t)

	const characterID = int64(4210000002)
	if err := db.Create(&model.EveCharacter{
		CharacterID:  characterID,
		CharacterName: "Before",
		UserID:       1,
		AccessToken:  "at-old",
		RefreshToken: "rt-old",
	}).Error; err != nil {
		t.Fatalf("seed character: %v", err)
	}

	mu := getCharacterLock(characterID)
	mu.Lock()
	done := make(chan struct{})
	go func() {
		defer close(done)
		if _, err := svc.updateExistingCharacterLocked(characterID, func(c *model.EveCharacter) error {
			c.CharacterName = "After"
			c.RefreshToken = "rt-from-callback"
			return nil
		}); err != nil {
			t.Errorf("updateExistingCharacterLocked: %v", err)
		}
	}()

	select {
	case <-done:
		t.Fatal("expected updateExistingCharacterLocked to block while the character lock is held")
	case <-time.After(100 * time.Millisecond):
	}

	mu.Unlock()
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("updateExistingCharacterLocked did not finish after lock release")
	}

	var got model.EveCharacter
	if err := db.Where("character_id = ?", characterID).First(&got).Error; err != nil {
		t.Fatalf("reload character: %v", err)
	}
	if got.CharacterName != "After" || got.RefreshToken != "rt-from-callback" {
		t.Fatalf("locked write not persisted: %+v", got)
	}
}

func TestUpdateExistingCharacterLockedPropagatesMutateError(t *testing.T) {
	svc, db := newTokenTestEnv(t)

	const characterID = int64(4210000003)
	if err := db.Create(&model.EveCharacter{
		CharacterID:  characterID,
		CharacterName: "Keep",
		UserID:       1,
	}).Error; err != nil {
		t.Fatalf("seed character: %v", err)
	}

	if _, err := svc.updateExistingCharacterLocked(characterID, func(c *model.EveCharacter) error {
		c.CharacterName = "Discarded"
		return http.ErrAbortHandler
	}); err == nil {
		t.Fatal("expected mutate error to propagate")
	}

	var got model.EveCharacter
	if err := db.Where("character_id = ?", characterID).First(&got).Error; err != nil {
		t.Fatalf("reload character: %v", err)
	}
	if got.CharacterName != "Keep" {
		t.Fatalf("mutate should not persist on error, got %q", got.CharacterName)
	}
}

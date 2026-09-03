package service

import (
	"amiya-eden/config"
	"amiya-eden/global"
	"amiya-eden/internal/model"
	"amiya-eden/internal/repository"
	"amiya-eden/pkg/cache"
	"amiya-eden/pkg/eve"
	"amiya-eden/pkg/eve/esi"
	"amiya-eden/pkg/jwt"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/glebarez/sqlite"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

const (
	callbackTestRequiredScope = "esi-corporations.read_structures.v1"
	callbackTestAssetsScope   = "esi-assets.read_corporation_assets.v1"
	callbackTestCharacterID   = int64(4210010001)
	callbackTestUserID        = uint(42001)
)

// buildTestEVEJWT 构造未签名但结构合法的 EVE SSO access_token（ParseAccessToken 不验签）
func buildTestEVEJWT(t *testing.T, characterID int64, name string, scopes []string) string {
	t.Helper()
	header := base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"RS256","typ":"JWT"}`))
	payload, err := json.Marshal(map[string]any{
		"sub": fmt.Sprintf("CHARACTER:EVE:%d", characterID),
		"name": name,
		"scp":  scopes,
		"iss":  "https://login.eveonline.com",
		"exp":  time.Now().Add(20 * time.Minute).Unix(),
	})
	if err != nil {
		t.Fatalf("marshal claims: %v", err)
	}
	return header + "." + base64.RawURLEncoding.EncodeToString(payload) + ".test-signature"
}

type callbackTestEnv struct {
	svc    *EveSSOService
	db     *gorm.DB
	mr     *miniredis.Miniredis
	grants []string
}

func newCallbackTestEnv(t *testing.T, grantedScopes []string) *callbackTestEnv {
	t.Helper()

	mr := miniredis.RunT(t)
	originalRedis := global.Redis
	originalDB := global.DB
	originalConfig := global.Config
	originalLogger := global.Logger
	global.Redis = redis.NewClient(&redis.Options{Addr: mr.Addr()})
	global.Logger = zap.NewNop()
	global.Config = &config.Config{}
	global.Config.JWT.ExpireDay = 7
	jwt.Init("callback-test-secret")
	t.Cleanup(func() {
		global.Redis = originalRedis
		global.DB = originalDB
		global.Config = originalConfig
		global.Logger = originalLogger
	})

	db, err := gorm.Open(sqlite.Open(fmt.Sprintf("file:eve_sso_callback_test_%d?mode=memory&cache=shared", time.Now().UnixNano())), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&model.User{}, &model.UserRole{}, &model.EveCharacter{}); err != nil {
		t.Fatalf("auto migrate: %v", err)
	}
	global.DB = db

	env := &callbackTestEnv{mr: mr, db: db, grants: grantedScopes}

	accessToken := buildTestEVEJWT(t, callbackTestCharacterID, "Director", grantedScopes)
	tokenServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = fmt.Fprintf(w,
			`{"access_token":%q,"token_type":"Bearer","expires_in":1199,"refresh_token":"rt-callback-new"}`,
			accessToken,
		)
	}))
	t.Cleanup(tokenServer.Close)

	esiServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = fmt.Fprintf(w, `[{"character_id":%d,"corporation_id":98765}]`, callbackTestCharacterID)
	}))
	t.Cleanup(esiServer.Close)

	env.svc = &EveSSOService{
		charRepo: repository.NewEveCharacterRepository(),
		userRepo: repository.NewUserRepository(),
		roleSvc:  NewRoleService(),
		eveClient: eve.NewClientWithEndpoints(
			"cid", "csecret", "http://localhost/cb",
			tokenServer.URL, tokenServer.URL,
		),
		esiClient: esi.NewClientWithConfig(esiServer.URL, ""),
	}
	return env
}

func (e *callbackTestEnv) seedUserAndCharacter(t *testing.T, scopes string) {
	t.Helper()
	if err := e.db.Create(&model.User{
		BaseModel:          model.BaseModel{ID: callbackTestUserID},
		PrimaryCharacterID: callbackTestCharacterID,
		Role:               model.RoleUser,
	}).Error; err != nil {
		t.Fatalf("seed user: %v", err)
	}
	if err := e.db.Create(&model.EveCharacter{
		CharacterID:  callbackTestCharacterID,
		CharacterName: "Director",
		UserID:       callbackTestUserID,
		AccessToken:  "at-old",
		RefreshToken: "rt-old",
		TokenExpiry:  time.Now().Add(10 * time.Minute),
		Scopes:       scopes,
	}).Error; err != nil {
		t.Fatalf("seed character: %v", err)
	}
}

func (e *callbackTestEnv) storeState(t *testing.T, sd stateData) string {
	t.Helper()
	state := fmt.Sprintf("state-%d", time.Now().UnixNano())
	if err := cache.Set(context.Background(), stateCachePrefix+state, sd, stateCacheTTL); err != nil {
		t.Fatalf("store state: %v", err)
	}
	return state
}

func (e *callbackTestEnv) reloadCharacter(t *testing.T) model.EveCharacter {
	t.Helper()
	var got model.EveCharacter
	if err := e.db.Where("character_id = ?", callbackTestCharacterID).First(&got).Error; err != nil {
		t.Fatalf("reload character: %v", err)
	}
	return got
}

func TestHandleCallbackReLoginMergesOptionalScopes(t *testing.T) {
	replaceRegisteredScopes(t,
		RegisteredScope{Module: "structures", Scope: callbackTestRequiredScope, Required: true},
		RegisteredScope{Module: "structures", Scope: callbackTestAssetsScope, Required: false},
	)

	// 本次登录链不含可选 assets scope
	env := newCallbackTestEnv(t, []string{"publicData", callbackTestRequiredScope})
	env.seedUserAndCharacter(t, "publicData "+callbackTestRequiredScope+" "+callbackTestAssetsScope)

	state := env.storeState(t, stateData{ScopeRefreshAttempted: true})

	result, err := env.svc.HandleCallback(context.Background(), "code", state, "127.0.0.1")
	if err != nil {
		t.Fatalf("HandleCallback: %v", err)
	}
	if result.Token == "" {
		t.Fatal("expected platform JWT to be issued")
	}

	got := env.reloadCharacter(t)
	if got.RefreshToken != "rt-callback-new" || got.TokenInvalid {
		t.Fatalf("token fields not updated: %+v", got)
	}
	for _, want := range []string{"publicData", callbackTestRequiredScope, callbackTestAssetsScope} {
		if !strings.Contains(" "+got.Scopes+" ", " "+want+" ") {
			t.Fatalf("expected merged scopes to retain %q, got %q", want, got.Scopes)
		}
	}
}

func TestHandleCallbackReLoginTriggersTransparentScopeReauth(t *testing.T) {
	replaceRegisteredScopes(t,
		RegisteredScope{Module: "structures", Scope: callbackTestRequiredScope, Required: true},
		RegisteredScope{Module: "structures", Scope: callbackTestAssetsScope, Required: false},
	)

	env := newCallbackTestEnv(t, []string{"publicData", callbackTestRequiredScope})
	env.seedUserAndCharacter(t, "publicData "+callbackTestRequiredScope+" "+callbackTestAssetsScope)

	state := env.storeState(t, stateData{})

	result, err := env.svc.HandleCallback(context.Background(), "code", state, "127.0.0.1")
	if err != nil {
		t.Fatalf("HandleCallback: %v", err)
	}
	if result.ReauthURL == "" {
		t.Fatal("expected ReauthURL for missing optional scope")
	}
	if result.Token != "" {
		t.Fatal("expected no platform JWT while reauth is pending")
	}

	// 补授 URL 应请求 必需+缺失 scope，且 state 已写入防循环标记
	parsed, err := url.Parse(result.ReauthURL)
	if err != nil {
		t.Fatalf("parse reauth url: %v", err)
	}
	requestedScopes := strings.Fields(parsed.Query().Get("scope"))
	for _, want := range []string{"publicData", callbackTestRequiredScope, callbackTestAssetsScope} {
		found := false
		for _, s := range requestedScopes {
			if s == want {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("reauth url scope %v missing %q", requestedScopes, want)
		}
	}

	var next stateData
	if err := cache.Get(context.Background(), stateCachePrefix+parsed.Query().Get("state"), &next); err != nil {
		t.Fatalf("load reauth state: %v", err)
	}
	if !next.ScopeRefreshAttempted {
		t.Fatal("expected reauth state to carry ScopeRefreshAttempted flag")
	}
	if len(next.ExtraScopes) != 1 || next.ExtraScopes[0] != callbackTestAssetsScope {
		t.Fatalf("reauth state ExtraScopes = %v, want [%s]", next.ExtraScopes, callbackTestAssetsScope)
	}

	// DB 已按并集保存，补授回调将再次幂等合并
	got := env.reloadCharacter(t)
	if !strings.Contains(got.Scopes, callbackTestAssetsScope) {
		t.Fatalf("expected merged scopes to retain assets scope, got %q", got.Scopes)
	}

	// 已补授过（flag 置位）后不再二次跳转，正常颁发 JWT
	state2 := env.storeState(t, stateData{ScopeRefreshAttempted: true})
	result2, err := env.svc.HandleCallback(context.Background(), "code", state2, "127.0.0.1")
	if err != nil {
		t.Fatalf("HandleCallback second pass: %v", err)
	}
	if result2.ReauthURL != "" || result2.Token == "" {
		t.Fatalf("second pass should issue JWT without reauth, got reauth=%q token=%q", result2.ReauthURL, result2.Token)
	}
}

func TestHandleCallbackReLoginNoReauthWithoutOptionalHistory(t *testing.T) {
	replaceRegisteredScopes(t,
		RegisteredScope{Module: "structures", Scope: callbackTestRequiredScope, Required: true},
		RegisteredScope{Module: "structures", Scope: callbackTestAssetsScope, Required: false},
	)

	env := newCallbackTestEnv(t, []string{"publicData", callbackTestRequiredScope})
	env.seedUserAndCharacter(t, "publicData "+callbackTestRequiredScope)

	state := env.storeState(t, stateData{})

	result, err := env.svc.HandleCallback(context.Background(), "code", state, "127.0.0.1")
	if err != nil {
		t.Fatalf("HandleCallback: %v", err)
	}
	if result.ReauthURL != "" || result.Token == "" {
		t.Fatalf("plain user should log in directly, got reauth=%q token=%q", result.ReauthURL, result.Token)
	}
}

func TestHandleCallbackStripsAdminOnlyScopeAfterDemotion(t *testing.T) {
	replaceRegisteredScopes(t,
		RegisteredScope{Module: "structures", Scope: callbackTestRequiredScope, Required: true},
		RegisteredScope{Module: "killmail", Scope: corpKillmailScope, Required: false},
	)

	// 无 user_role 行 → guest，不再具备管理员职权
	env := newCallbackTestEnv(t, []string{"publicData", callbackTestRequiredScope})
	env.seedUserAndCharacter(t, "publicData "+callbackTestRequiredScope+" "+corpKillmailScope)

	state := env.storeState(t, stateData{ScopeRefreshAttempted: true})

	result, err := env.svc.HandleCallback(context.Background(), "code", state, "127.0.0.1")
	if err != nil {
		t.Fatalf("HandleCallback: %v", err)
	}
	if result.Token == "" {
		t.Fatal("expected platform JWT to be issued")
	}

	got := env.reloadCharacter(t)
	if strings.Contains(got.Scopes, corpKillmailScope) {
		t.Fatalf("expected admin-only scope stripped for demoted owner, got %q", got.Scopes)
	}
	if !strings.Contains(got.Scopes, callbackTestRequiredScope) {
		t.Fatalf("expected required scope retained, got %q", got.Scopes)
	}
}

func TestHandleCallbackRetainsAdminOnlyScopeForAdmin(t *testing.T) {
	replaceRegisteredScopes(t,
		RegisteredScope{Module: "structures", Scope: callbackTestRequiredScope, Required: true},
		RegisteredScope{Module: "killmail", Scope: corpKillmailScope, Required: false},
	)

	env := newCallbackTestEnv(t, []string{"publicData", callbackTestRequiredScope})
	env.seedUserAndCharacter(t, "publicData "+callbackTestRequiredScope+" "+corpKillmailScope)
	if err := env.db.Create(&model.UserRole{
		UserID:   callbackTestUserID,
		RoleCode: model.RoleAdmin,
	}).Error; err != nil {
		t.Fatalf("seed user role: %v", err)
	}

	state := env.storeState(t, stateData{ScopeRefreshAttempted: true})

	if _, err := env.svc.HandleCallback(context.Background(), "code", state, "127.0.0.1"); err != nil {
		t.Fatalf("HandleCallback: %v", err)
	}

	got := env.reloadCharacter(t)
	if !strings.Contains(got.Scopes, corpKillmailScope) {
		t.Fatalf("expected admin-only scope retained for admin owner, got %q", got.Scopes)
	}
}

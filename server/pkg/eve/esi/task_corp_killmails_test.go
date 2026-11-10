package esi

import (
	"amiya-eden/global"
	"amiya-eden/internal/model"
	"amiya-eden/internal/repository"
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

func TestCorpKillmailsTaskLoadExistingKillmailsReturnsExistingRows(t *testing.T) {
	db := newCorpKillmailTaskTestDB(t)
	oldDB := global.DB
	global.DB = db
	t.Cleanup(func() { global.DB = oldDB })

	if err := db.Create(&model.EveKillmailList{KillmailID: 9001, CharacterID: 1001}).Error; err != nil {
		t.Fatalf("create killmail: %v", err)
	}

	task := &CorpKillmailsTask{}
	existing, err := task.loadExistingKillmails([]int64{9001, 9002})
	if err != nil {
		t.Fatalf("load existing killmails: %v", err)
	}
	if _, ok := existing[9001]; !ok {
		t.Fatal("expected existing killmail 9001 to be returned")
	}
	if _, ok := existing[9002]; ok {
		t.Fatal("expected missing killmail 9002 to stay absent")
	}
}

func TestCorpKillmailsTaskEnsureVictimLinksCreatesMissingLinksIdempotently(t *testing.T) {
	db := newCorpKillmailTaskTestDB(t)
	oldDB := global.DB
	global.DB = db
	t.Cleanup(func() { global.DB = oldDB })

	killmail := model.EveKillmailList{KillmailID: 9001, CharacterID: 1001}
	if err := db.Create(&killmail).Error; err != nil {
		t.Fatalf("create killmail: %v", err)
	}

	task := &CorpKillmailsTask{}
	existing := map[int64]model.EveKillmailList{9001: killmail}
	knownChars := map[int64]bool{1001: true}

	if err := task.ensureVictimLinks(existing, knownChars); err != nil {
		t.Fatalf("ensure victim links: %v", err)
	}
	if err := task.ensureVictimLinks(existing, knownChars); err != nil {
		t.Fatalf("ensure victim links second call: %v", err)
	}

	var links []model.EveCharacterKillmail
	if err := db.Order("id asc").Find(&links).Error; err != nil {
		t.Fatalf("list links: %v", err)
	}
	if len(links) != 1 {
		t.Fatalf("expected exactly one victim link, got %d", len(links))
	}
	if links[0].CharacterID != 1001 || links[0].KillmailID != 9001 || !links[0].Victim {
		t.Fatalf("unexpected link: %+v", links[0])
	}
}

func newCorpKillmailTaskTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	dsn := fmt.Sprintf("file:corp_killmail_task_test_%d?mode=memory&cache=shared", time.Now().UnixNano())
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&model.EveKillmailList{}, &model.EveCharacterKillmail{}); err != nil {
		t.Fatalf("auto migrate: %v", err)
	}
	return db
}

func TestCorpKillmailsTaskSkipsOnForbidden(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(fmt.Sprintf("file:corp_killmails_forbidden_test_%d?mode=memory&cache=shared", time.Now().UnixNano())), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&model.EveCharacter{}, &model.EveCharacterCorpRole{}); err != nil {
		t.Fatalf("auto migrate: %v", err)
	}

	oldDB := global.DB
	oldLogger := global.Logger
	global.DB = db
	global.Logger = zap.NewNop()
	t.Cleanup(func() {
		global.DB = oldDB
		global.Logger = oldLogger
	})

	const (
		characterID   = int64(90020001)
		corporationID = int64(555021)
	)
	if err := db.Create(&model.EveCharacter{
		CharacterID:   characterID,
		CharacterName: "Director",
		UserID:        1,
		CorporationID: corporationID,
		AccessToken:   "token",
		Scopes:        "esi-killmails.read_corporation_killmails.v1",
	}).Error; err != nil {
		t.Fatalf("seed eve_character: %v", err)
	}
	if err := db.Create(&model.EveCharacterCorpRole{
		CharacterID: characterID,
		CorpRole:    "Director",
	}).Error; err != nil {
		t.Fatalf("seed corp role: %v", err)
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if req.URL.Path == fmt.Sprintf("/corporations/%d/killmails/recent/", corporationID) {
			http.Error(w, `{"error":"insufficient_scope"}`, http.StatusForbidden)
			return
		}
		t.Fatalf("unexpected request path: %s", req.URL.Path)
	}))
	t.Cleanup(server.Close)

	task := &CorpKillmailsTask{}
	err = task.Execute(&TaskContext{
		CharacterID: characterID,
		AccessToken: "token",
		Client:      NewClientWithConfig(server.URL, ""),
	})
	if err == nil {
		t.Fatal("expected ErrTaskSkipped on 403, got nil")
	}
	if err != ErrTaskSkipped {
		t.Fatalf("expected ErrTaskSkipped on 403, got %v", err)
	}
}

type revokedCorpKillmailTokenService struct{}

func (revokedCorpKillmailTokenService) GetValidToken(ctx context.Context, characterID int64) (string, error) {
	if characterID == 1001 {
		if err := global.DB.Model(&model.EveCharacter{}).Where("character_id = ?", characterID).Update("token_invalid", true).Error; err != nil {
			return "", err
		}
		return "", fmt.Errorf("invalid_grant")
	}
	return fakeQueueTokenService{}.GetValidToken(ctx, characterID)
}

func TestCorpKillmailsProvidersFallBackAndExcludeRejectedAuthorization(t *testing.T) {
	for _, status := range []int{0, http.StatusOK, http.StatusUnauthorized, http.StatusForbidden, http.StatusInternalServerError} {
		t.Run(fmt.Sprint(status), func(t *testing.T) {
			db := newCorpKillmailTaskTestDB(t)
			if err := db.AutoMigrate(&model.EveCharacter{}, &model.EveCharacterCorpRole{}); err != nil {
				t.Fatal(err)
			}
			oldDB, oldLogger, oldRedis := global.DB, global.Logger, global.Redis
			global.DB, global.Logger, global.Redis = db, zap.NewNop(), nil
			t.Cleanup(func() { global.DB, global.Logger, global.Redis = oldDB, oldLogger, oldRedis })
			chars := []model.EveCharacter{
				{CharacterID: 1001, CorporationID: 9901, AccessToken: "token-1001", Scopes: "publicData esi-killmails.read_corporation_killmails.v1"},
				{CharacterID: 1002, CorporationID: 9901, AccessToken: "token-1002", Scopes: "publicData esi-killmails.read_corporation_killmails.v1"},
			}
			for i := range chars {
				if err := db.Create(&chars[i]).Error; err != nil {
					t.Fatal(err)
				}
				if err := db.Create(&model.EveCharacterCorpRole{CharacterID: chars[i].CharacterID, CorpRole: "Director"}).Error; err != nil {
					t.Fatal(err)
				}
			}
			var tokens []string
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
				token := req.Header.Get("Authorization")
				tokens = append(tokens, token)
				if token == "Bearer token-1001" && status != http.StatusOK {
					http.Error(w, `{"error":"Unauthorized - Token is not valid for any required scope: esi-killmails.read_corporation_killmails.v1"}`, status)
					return
				}
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte("[]"))
			}))
			t.Cleanup(srv.Close)
			repo := repository.NewEveCharacterRepository()
			q := NewQueue(fakeQueueTokenService{}, repo)
			if status == 0 {
				q.ssoSvc = revokedCorpKillmailTokenService{}
			}
			q.client = NewClientWithConfig(srv.URL, "")
			providers, err := buildAuthorizedCorpKillmailProviders(chars)
			if err != nil {
				t.Fatal(err)
			}
			err = q.executeTaskWithProviders(context.Background(), &CorpKillmailsTask{}, chars[0], true, true, chars, providers)
			if status == http.StatusInternalServerError {
				if err == nil || errors.Is(err, ErrTaskSkipped) {
					t.Fatalf("expected transient failure, got %v", err)
				}
				current, lookupErr := repo.GetByCharacterID(1001)
				if lookupErr != nil || current.Scopes != chars[0].Scopes {
					t.Fatal("transient failure changed authorization")
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if status == http.StatusOK {
				if len(tokens) != 1 || tokens[0] != "Bearer token-1001" {
					t.Fatalf("successful provider should finish the corporation job, got %v", tokens)
				}
				return
			}
			if status == 0 {
				if len(tokens) != 1 || tokens[0] != "Bearer token-1002" {
					t.Fatalf("expected revoked refresh token to fall back without an ESI request, got %v", tokens)
				}
				current, err := repo.GetByCharacterID(1001)
				if err != nil || !current.TokenInvalid {
					t.Fatalf("expected revoked token to remain invalid: %v", err)
				}
				return
			}
			if len(tokens) != 2 || tokens[1] != "Bearer token-1002" {
				t.Fatalf("expected fallback to healthy provider, got %v", tokens)
			}
			current, err := repo.GetByCharacterID(1001)
			if err != nil {
				t.Fatal(err)
			}
			if current.Scopes != "publicData" || current.TokenInvalid {
				t.Fatalf("expected only rejected scope disabled, got scopes=%q invalid=%v", current.Scopes, current.TokenInvalid)
			}
			if err := (&CorpKillmailsTask{}).Execute(&TaskContext{CharacterID: 1001, AccessToken: "token-1001", Client: q.client}); !errors.Is(err, ErrTaskSkipped) {
				t.Fatalf("expected rejected authorization to remain skipped, got %v", err)
			}
			if len(tokens) != 2 {
				t.Fatal("rejected authorization sent another ESI request")
			}
			chars[0] = *current
			providers, err = buildAuthorizedCorpKillmailProviders(chars)
			if err != nil {
				t.Fatal(err)
			}
			if _, ok := providers[1001]; ok {
				t.Fatal("rejected provider was selected again")
			}
			if providers[1002] != 9901 {
				t.Fatal("healthy provider was removed")
			}
			// A subsequent SSO grant restores eligibility.
			if err := db.Model(current).Update("scopes", chars[1].Scopes).Error; err != nil {
				t.Fatal(err)
			}
			chars[0].Scopes = chars[1].Scopes
			providers, err = buildAuthorizedCorpKillmailProviders(chars)
			if err != nil || providers[1001] != 9901 {
				t.Fatalf("renewed authorization not eligible: %v", err)
			}
		})
	}
}

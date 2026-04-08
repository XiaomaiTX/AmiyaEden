package esi

import (
	"amiya-eden/global"
	"amiya-eden/internal/model"
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type fakeQueueTokenService struct{}

func (fakeQueueTokenService) GetValidToken(ctx context.Context, characterID int64) (string, error) {
	return fmt.Sprintf("token-%d", characterID), nil
}

type fakeQueueCharacterRepository struct {
	characters     map[int64]model.EveCharacter
	listCharacters []model.EveCharacter
}

func (f *fakeQueueCharacterRepository) ListAllWithToken() ([]model.EveCharacter, error) {
	return append([]model.EveCharacter(nil), f.listCharacters...), nil
}

func (f *fakeQueueCharacterRepository) GetByCharacterID(characterID int64) (*model.EveCharacter, error) {
	char, ok := f.characters[characterID]
	if !ok {
		return nil, fmt.Errorf("character %d not found", characterID)
	}
	copyChar := char
	return &copyChar, nil
}

func TestQueueShouldSkipAutomaticTaskSkipsCharacterKillmailsWhenCorporationCovered(t *testing.T) {
	repo := &fakeQueueCharacterRepository{}
	queue := NewQueue(fakeQueueTokenService{}, repo)
	char := model.EveCharacter{CharacterID: 1001, CorporationID: 9901}
	corpCoverage := map[int64]bool{9901: true}

	if !queue.shouldSkipAutomaticTask(char, &KillmailsTask{}, corpCoverage) {
		t.Fatalf("expected automatic character killmail task to be skipped when corp coverage exists")
	}
	if queue.shouldSkipAutomaticTask(char, &CorpKillmailsTask{}, corpCoverage) {
		t.Fatalf("expected corporation killmail task itself to remain runnable")
	}
}

func TestQueueTaskExecutionKeyUsesCorporationForCorporationKillmails(t *testing.T) {
	queue := NewQueue(fakeQueueTokenService{}, &fakeQueueCharacterRepository{})
	charA := model.EveCharacter{CharacterID: 1001, CorporationID: 9901}
	charB := model.EveCharacter{CharacterID: 1002, CorporationID: 9901}

	keyA := queue.taskExecutionKey(&CorpKillmailsTask{}, charA)
	keyB := queue.taskExecutionKey(&CorpKillmailsTask{}, charB)
	if keyA != keyB {
		t.Fatalf("expected corporation killmail executions to dedupe by corp, got %q and %q", keyA, keyB)
	}

	charKeyA := queue.taskExecutionKey(&KillmailsTask{}, charA)
	charKeyB := queue.taskExecutionKey(&KillmailsTask{}, charB)
	if charKeyA == charKeyB {
		t.Fatalf("expected personal killmail executions to stay per character, got %q", charKeyA)
	}
}

func TestQueueNeedsRefreshSharesCorporationKillmailLastRunAcrossProviders(t *testing.T) {
	mini := miniredis.RunT(t)
	oldRedis := global.Redis
	global.Redis = redis.NewClient(&redis.Options{Addr: mini.Addr()})
	t.Cleanup(func() {
		_ = global.Redis.Close()
		global.Redis = oldRedis
	})

	queue := NewQueue(fakeQueueTokenService{}, &fakeQueueCharacterRepository{})
	charA := model.EveCharacter{CharacterID: 1001, CorporationID: 9901}
	charB := model.EveCharacter{CharacterID: 1002, CorporationID: 9901}
	corpTask := &CorpKillmailsTask{}

	queue.setLastRun(corpTask, charA, time.Now())
	if queue.needsRefresh(corpTask, charB, true) {
		t.Fatalf("expected corporation killmail freshness to be shared across providers in the same corp")
	}
	if !queue.needsRefresh(&KillmailsTask{}, charB, true) {
		t.Fatalf("expected personal killmail freshness to remain character-scoped")
	}
}

func TestBuildAuthorizedCorpKillmailProvidersMarksOnlyDirectorBackedProviders(t *testing.T) {
	chars := []model.EveCharacter{
		{CharacterID: 1001, CorporationID: 9901, Scopes: "esi-killmails.read_corporation_killmails.v1 esi-location.read_location.v1"},
		{CharacterID: 1002, CorporationID: 9901, Scopes: "esi-location.read_location.v1"},
		{CharacterID: 1003, CorporationID: 9902, Scopes: "esi-location.read_location.v1"},
	}

	db := newQueueCoverageTestDB(t)
	oldDB := global.DB
	global.DB = db
	t.Cleanup(func() { global.DB = oldDB })
	if err := db.Create(&model.EveCharacterCorpRole{CharacterID: 1001, CorpRole: "Director"}).Error; err != nil {
		t.Fatalf("create corp role: %v", err)
	}

	providers, err := buildAuthorizedCorpKillmailProviders(chars)
	if err != nil {
		t.Fatalf("build corp coverage: %v", err)
	}
	if providers[1001] != 9901 {
		t.Fatalf("expected director-backed character to be an authorized provider")
	}
	if _, ok := providers[1002]; ok {
		t.Fatalf("expected non-director character to stay unauthorized")
	}
	if _, ok := providers[1003]; ok {
		t.Fatalf("expected character without scope to stay unauthorized")
	}
}

func TestBuildAuthorizedCorpKillmailProvidersRequiresDirectorRole(t *testing.T) {
	db := newQueueCoverageTestDB(t)
	oldDB := global.DB
	global.DB = db
	t.Cleanup(func() { global.DB = oldDB })

	chars := []model.EveCharacter{
		{CharacterID: 1001, CorporationID: 9901, Scopes: "esi-killmails.read_corporation_killmails.v1"},
	}
	providers, err := buildAuthorizedCorpKillmailProviders(chars)
	if err != nil {
		t.Fatalf("build corp coverage: %v", err)
	}
	if _, ok := providers[1001]; ok {
		t.Fatalf("expected provider selection to require Director role")
	}
}

func TestCorporationKillmailsFreshRequiresSuccessfulCorpRun(t *testing.T) {
	mini := miniredis.RunT(t)
	oldRedis := global.Redis
	global.Redis = redis.NewClient(&redis.Options{Addr: mini.Addr()})
	t.Cleanup(func() {
		_ = global.Redis.Close()
		global.Redis = oldRedis
	})

	queue := NewQueue(fakeQueueTokenService{}, &fakeQueueCharacterRepository{})
	if queue.corporationKillmailsFresh(9901, true) {
		t.Fatalf("expected corp coverage to stay false before any successful corp killmail refresh")
	}
	queue.setLastRun(&CorpKillmailsTask{}, model.EveCharacter{CorporationID: 9901}, time.Now())
	if !queue.corporationKillmailsFresh(9901, true) {
		t.Fatalf("expected corp coverage after a successful corp killmail refresh")
	}
}

func TestCorporationKillmailsFreshUsesInactiveInterval(t *testing.T) {
	mini := miniredis.RunT(t)
	oldRedis := global.Redis
	global.Redis = redis.NewClient(&redis.Options{Addr: mini.Addr()})
	t.Cleanup(func() {
		_ = global.Redis.Close()
		global.Redis = oldRedis
	})

	queue := NewQueue(fakeQueueTokenService{}, &fakeQueueCharacterRepository{})
	recent := time.Now().Add(-2 * time.Hour)
	queue.setLastRun(&CorpKillmailsTask{}, model.EveCharacter{CorporationID: 9901}, recent)
	if queue.corporationKillmailsFresh(9901, true) {
		t.Fatalf("expected active corp coverage to expire after 60 minutes")
	}
	if !queue.corporationKillmailsFresh(9901, false) {
		t.Fatalf("expected inactive corp coverage to remain fresh for the 1 day window")
	}
}

func newQueueCoverageTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	dsn := fmt.Sprintf("file:queue_coverage_test_%d?mode=memory&cache=shared", time.Now().UnixNano())
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&model.EveCharacterCorpRole{}); err != nil {
		t.Fatalf("auto migrate: %v", err)
	}
	return db
}

package esi

import (
	"amiya-eden/global"
	"amiya-eden/internal/model"
	"context"
	"errors"
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

func TestQueueRunHandlesNilGlobalLogger(t *testing.T) {
	oldLogger := global.CurrentLogger()
	global.SetLogger(nil)
	t.Cleanup(func() {
		global.SetLogger(oldLogger)
	})

	queue := NewQueue(fakeQueueTokenService{}, &fakeQueueCharacterRepository{})
	if err := queue.Run(context.Background()); err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
}

func TestQueueRunSkipsWhenGlobalDBIsNil(t *testing.T) {
	oldDB := global.DB
	oldLogger := global.CurrentLogger()
	global.DB = nil
	global.SetLogger(nil)
	t.Cleanup(func() {
		global.DB = oldDB
		global.SetLogger(oldLogger)
	})

	queue := NewQueue(fakeQueueTokenService{}, &fakeQueueCharacterRepository{
		listCharacters: []model.EveCharacter{
			{CharacterID: 1001, RefreshToken: "refresh-token"},
		},
	})

	if err := queue.Run(context.Background()); err != nil {
		t.Fatalf("Run returned error: %v", err)
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
	if queue.corporationKillmailsFresh(9901, nil, true) {
		t.Fatalf("expected corp coverage to stay false before any successful corp killmail refresh")
	}
	queue.setLastRun(&CorpKillmailsTask{}, model.EveCharacter{CorporationID: 9901}, time.Now())
	if !queue.corporationKillmailsFresh(9901, nil, true) {
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
	if queue.corporationKillmailsFresh(9901, nil, true) {
		t.Fatalf("expected active corp coverage to expire after 60 minutes")
	}
	if !queue.corporationKillmailsFresh(9901, nil, false) {
		t.Fatalf("expected inactive corp coverage to remain fresh for the 1 day window")
	}
}

func TestCharacterKillmailsFreshUsesDailyActiveInterval(t *testing.T) {
	mini := miniredis.RunT(t)
	oldRedis := global.Redis
	global.Redis = redis.NewClient(&redis.Options{Addr: mini.Addr()})
	t.Cleanup(func() {
		_ = global.Redis.Close()
		global.Redis = oldRedis
	})

	queue := NewQueue(fakeQueueTokenService{}, &fakeQueueCharacterRepository{})
	char := model.EveCharacter{CharacterID: 1001}

	queue.setLastRun(&KillmailsTask{}, char, time.Now().Add(-23*time.Hour))
	if queue.needsRefresh(&KillmailsTask{}, char, true) {
		t.Fatalf("expected personal killmail refresh to stay fresh within the 24 hour active window")
	}

	queue.setLastRun(&KillmailsTask{}, char, time.Now().Add(-25*time.Hour))
	if !queue.needsRefresh(&KillmailsTask{}, char, true) {
		t.Fatalf("expected personal killmail refresh to expire after the 24 hour active window")
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

type skippedQueueTask struct{}

func (skippedQueueTask) Name() string        { return "skipped_queue_task" }
func (skippedQueueTask) Description() string { return "skipped task" }
func (skippedQueueTask) Priority() Priority  { return PriorityLow }
func (skippedQueueTask) Interval() RefreshInterval {
	return RefreshInterval{Active: time.Hour, Inactive: 2 * time.Hour}
}
func (skippedQueueTask) RequiredScopes() []TaskScope { return nil }
func (skippedQueueTask) Execute(ctx *TaskContext) error {
	return ErrTaskSkipped
}

type legacyQueueTask struct{}

func (legacyQueueTask) Name() string        { return "legacy_queue_task" }
func (legacyQueueTask) Description() string { return "legacy task" }
func (legacyQueueTask) Priority() Priority  { return PriorityLow }
func (legacyQueueTask) Interval() RefreshInterval {
	return RefreshInterval{Active: time.Hour, Inactive: 2 * time.Hour}
}
func (legacyQueueTask) RequiredScopes() []TaskScope { return nil }
func (legacyQueueTask) Execute(ctx *TaskContext) error {
	return nil
}

type contextAwareQueueTask struct {
	name string
	seen chan context.Context
}

func (t *contextAwareQueueTask) Name() string        { return t.name }
func (t *contextAwareQueueTask) Description() string { return "context-aware task" }
func (t *contextAwareQueueTask) Priority() Priority  { return PriorityLow }
func (t *contextAwareQueueTask) Interval() RefreshInterval {
	return RefreshInterval{Active: time.Hour, Inactive: 2 * time.Hour}
}
func (t *contextAwareQueueTask) RequiredScopes() []TaskScope { return nil }
func (t *contextAwareQueueTask) Execute(ctx *TaskContext) error {
	t.seen <- ctx.Context
	return nil
}

func TestQueueRunTaskPropagatesContextToTask(t *testing.T) {
	type contextKey string
	const key contextKey = "queue-propagation"

	task := &contextAwareQueueTask{
		name: fmt.Sprintf("context_aware_queue_task_%d", time.Now().UnixNano()),
		seen: make(chan context.Context, 1),
	}
	Register(task)

	queue := NewQueue(fakeQueueTokenService{}, &fakeQueueCharacterRepository{
		characters: map[int64]model.EveCharacter{
			1001: {CharacterID: 1001},
		},
	})

	runCtx := context.WithValue(context.Background(), key, "expected")

	if err := queue.RunTask(runCtx, task.name, 1001); err != nil {
		t.Fatalf("RunTask returned error: %v", err)
	}

	select {
	case seenCtx := <-task.seen:
		if seenCtx == nil {
			t.Fatal("expected task to receive a context")
		}
		if got := seenCtx.Value(key); got != "expected" {
			t.Fatalf("task context value = %v, want %q", got, "expected")
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for task execution")
	}
}

func TestQueueExecuteTaskMarksSkippedStatusWithoutPersistingLastRun(t *testing.T) {
	mini := miniredis.RunT(t)
	oldRedis := global.Redis
	global.Redis = redis.NewClient(&redis.Options{Addr: mini.Addr()})
	t.Cleanup(func() {
		_ = global.Redis.Close()
		global.Redis = oldRedis
	})

	queue := NewQueue(fakeQueueTokenService{}, &fakeQueueCharacterRepository{})
	char := model.EveCharacter{CharacterID: 1001}
	task := skippedQueueTask{}

	if err := queue.executeTask(context.Background(), task, char, true, true); !errors.Is(err, ErrTaskSkipped) {
		t.Fatalf("executeTask() error = %v, want %v", err, ErrTaskSkipped)
	}

	status := queue.statuses[fmt.Sprintf("%s:%d", task.Name(), char.CharacterID)]
	if status == nil {
		t.Fatal("expected task status to be recorded")
	}
	if status.Status != "skipped" {
		t.Fatalf("expected skipped status, got %q", status.Status)
	}
	key := fmt.Sprintf("%s%s", lastRunKeyPrefix, queue.taskExecutionKey(task, char))
	if mini.Exists(key) {
		t.Fatalf("expected skipped task to avoid persisting last-run key %q", key)
	}
}

func TestQueueNeedsRefreshFallsBackToLegacyLastRunKey(t *testing.T) {
	mini := miniredis.RunT(t)
	oldRedis := global.Redis
	global.Redis = redis.NewClient(&redis.Options{Addr: mini.Addr()})
	t.Cleanup(func() {
		_ = global.Redis.Close()
		global.Redis = oldRedis
	})

	queue := NewQueue(fakeQueueTokenService{}, &fakeQueueCharacterRepository{})
	task := legacyQueueTask{}
	char := model.EveCharacter{CharacterID: 1001}
	legacyKey := fmt.Sprintf("%s%s:%d", lastRunKeyPrefix, task.Name(), char.CharacterID)
	legacyRun := time.Now().Add(-30 * time.Minute).Unix()
	if err := mini.Set(legacyKey, fmt.Sprintf("%d", legacyRun)); err != nil {
		t.Fatalf("set legacy key: %v", err)
	}

	if queue.needsRefresh(task, char, true) {
		t.Fatal("expected fresh legacy last-run key to suppress refresh")
	}
}

func TestCorporationKillmailsFreshFallsBackToLegacyProviderKeys(t *testing.T) {
	mini := miniredis.RunT(t)
	oldRedis := global.Redis
	global.Redis = redis.NewClient(&redis.Options{Addr: mini.Addr()})
	t.Cleanup(func() {
		_ = global.Redis.Close()
		global.Redis = oldRedis
	})

	queue := NewQueue(fakeQueueTokenService{}, &fakeQueueCharacterRepository{})
	legacyKey := fmt.Sprintf("%s%s:%d", lastRunKeyPrefix, (&CorpKillmailsTask{}).Name(), 1001)
	if err := mini.Set(legacyKey, fmt.Sprintf("%d", time.Now().Add(-30*time.Minute).Unix())); err != nil {
		t.Fatalf("set legacy provider key: %v", err)
	}

	if !queue.corporationKillmailsFresh(9901, []int64{1001}, true) {
		t.Fatal("expected corporation freshness to use legacy provider keys during migration")
	}
}

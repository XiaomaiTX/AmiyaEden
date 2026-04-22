package jobs

import (
	"amiya-eden/global"
	"amiya-eden/internal/model"
	"amiya-eden/internal/taskregistry"
	"amiya-eden/pkg/background"
	"amiya-eden/pkg/eve/esi"
	"context"
	"testing"
	"time"

	"go.uber.org/zap"
	"gorm.io/gorm"
)

type esiRefreshTestTokenService struct{}

func (esiRefreshTestTokenService) GetValidToken(context.Context, int64) (string, error) {
	return "", nil
}

type esiRefreshTestCharacterRepo struct {
	listed chan struct{}
}

func (r *esiRefreshTestCharacterRepo) ListAllWithToken() ([]model.EveCharacter, error) {
	close(r.listed)
	return nil, nil
}

func (r *esiRefreshTestCharacterRepo) GetByCharacterID(int64) (*model.EveCharacter, error) {
	return nil, gorm.ErrRecordNotFound
}

func TestRegisterESIRefreshTaskStartsInitialQueuePass(t *testing.T) {
	oldQueueFactory := newESIQueueForJobs
	oldStartupRun := startInitialESIQueueRun
	oldLogger := global.Logger
	oldQueue := esiQueue

	t.Cleanup(func() {
		newESIQueueForJobs = oldQueueFactory
		startInitialESIQueueRun = oldStartupRun
		global.SetLogger(oldLogger)
		esiQueue = oldQueue
	})

	global.SetLogger(zap.NewNop())
	newESIQueueForJobs = func() *esi.Queue {
		return &esi.Queue{}
	}

	var started bool
	startInitialESIQueueRun = func(queue *esi.Queue) {
		started = true
		if queue == nil {
			t.Fatal("expected startup queue pass to receive a queue instance")
		}
	}

	reg := taskregistry.New()
	registerESIRefreshTask(reg)

	if esiQueue == nil {
		t.Fatal("expected registerESIRefreshTask to initialize the global ESI queue")
	}
	if !started {
		t.Fatal("expected registerESIRefreshTask to trigger one immediate startup queue pass")
	}
	def, ok := reg.Get("esi_refresh")
	if !ok {
		t.Fatal("expected esi_refresh task definition to be registered")
	}
	if def.Category != taskregistry.TaskCategoryESI {
		t.Fatalf("category = %q, want %q", def.Category, taskregistry.TaskCategoryESI)
	}
	if def.Type != taskregistry.TaskTypeRecurring {
		t.Fatalf("type = %q, want %q", def.Type, taskregistry.TaskTypeRecurring)
	}
	if def.DefaultCron != "0 */5 * * * *" {
		t.Fatalf("default cron = %q, want %q", def.DefaultCron, "0 */5 * * * *")
	}
	if def.RunFunc == nil {
		t.Fatal("expected esi_refresh task to have a run function")
	}
}

func TestStartInitialESIQueueRunRunsInlineWhenManagerIsStopping(t *testing.T) {
	oldLogger := global.CurrentLogger()
	oldManager := global.BackgroundTaskManager()
	oldDB := global.DB
	global.SetLogger(zap.NewNop())
	global.DB = &gorm.DB{}
	mgr := background.New(context.Background(), func() *zap.Logger { return global.Logger })
	global.SetBackgroundTaskManager(mgr)
	t.Cleanup(func() {
		global.SetBackgroundTaskManager(oldManager)
		global.SetLogger(oldLogger)
		global.DB = oldDB
	})

	if err := mgr.Shutdown(time.Second); err != nil {
		t.Fatalf("Shutdown returned error: %v", err)
	}

	listed := make(chan struct{}, 1)
	queue := esi.NewQueue(
		esiRefreshTestTokenService{},
		&esiRefreshTestCharacterRepo{listed: listed},
	)

	startInitialESIQueueRun(queue)

	select {
	case <-listed:
	case <-time.After(time.Second):
		t.Fatal("expected initial ESI queue pass to run inline when background manager is stopping")
	}
}

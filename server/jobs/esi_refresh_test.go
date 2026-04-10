package jobs

import (
	"amiya-eden/global"
	"amiya-eden/internal/taskregistry"
	"amiya-eden/pkg/eve/esi"
	"testing"

	"go.uber.org/zap"
)

func TestRegisterESIRefreshTaskStartsInitialQueuePass(t *testing.T) {
	oldQueueFactory := newESIQueueForJobs
	oldStartupRun := startInitialESIQueueRun
	oldLogger := global.Logger
	oldQueue := esiQueue

	t.Cleanup(func() {
		newESIQueueForJobs = oldQueueFactory
		startInitialESIQueueRun = oldStartupRun
		global.Logger = oldLogger
		esiQueue = oldQueue
	})

	global.Logger = zap.NewNop()
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

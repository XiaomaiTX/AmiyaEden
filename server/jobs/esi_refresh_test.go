package jobs

import (
	"amiya-eden/global"
	"amiya-eden/pkg/eve/esi"
	"testing"

	"github.com/robfig/cron/v3"
	"go.uber.org/zap"
)

func TestRegisterESIRefreshJobStartsInitialQueuePass(t *testing.T) {
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

	c := cron.New(cron.WithSeconds())
	registerESIRefreshJob(c)

	if esiQueue == nil {
		t.Fatal("expected registerESIRefreshJob to initialize the global ESI queue")
	}
	if !started {
		t.Fatal("expected registerESIRefreshJob to trigger one immediate startup queue pass")
	}
	if len(c.Entries()) != 1 {
		t.Fatalf("expected exactly one cron entry, got %d", len(c.Entries()))
	}
}

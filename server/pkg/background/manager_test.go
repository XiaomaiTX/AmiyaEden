package background

import (
	"context"
	"testing"
	"time"

	"go.uber.org/zap"
)

func TestManagerShutdownCancelsAndWaitsForTrackedTasks(t *testing.T) {
	mgr := New(context.Background(), func() *zap.Logger { return zap.NewNop() })
	started := make(chan struct{})
	finished := make(chan struct{})

	if ok := mgr.Go("wait-for-cancel", func(ctx context.Context) error {
		close(started)
		<-ctx.Done()
		close(finished)
		return ctx.Err()
	}); !ok {
		t.Fatal("expected manager to accept task before shutdown")
	}

	select {
	case <-started:
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for background task to start")
	}

	if err := mgr.Shutdown(time.Second); err != nil {
		t.Fatalf("Shutdown returned error: %v", err)
	}

	select {
	case <-finished:
	case <-time.After(time.Second):
		t.Fatal("expected Shutdown to wait for tracked background tasks")
	}
}

func TestManagerGoSkipsTasksAfterShutdown(t *testing.T) {
	mgr := New(context.Background(), func() *zap.Logger { return zap.NewNop() })
	if err := mgr.Shutdown(time.Second); err != nil {
		t.Fatalf("Shutdown returned error: %v", err)
	}

	ran := make(chan struct{}, 1)
	if ok := mgr.Go("late-task", func(ctx context.Context) error {
		close(ran)
		return nil
	}); ok {
		t.Fatal("expected manager to reject new tasks after shutdown")
	}

	select {
	case <-ran:
		t.Fatal("task ran after shutdown")
	case <-time.After(100 * time.Millisecond):
	}
}

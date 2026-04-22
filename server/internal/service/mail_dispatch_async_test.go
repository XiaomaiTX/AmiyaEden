package service

import (
	"amiya-eden/global"
	"amiya-eden/pkg/background"
	"context"
	"errors"
	"testing"
	"time"

	"go.uber.org/zap"
)

func TestDispatchMailAttemptAsyncUsesBackgroundTaskManager(t *testing.T) {
	oldLogger := global.CurrentLogger()
	oldManager := global.BackgroundTaskManager()
	global.SetLogger(zap.NewNop())
	mgr := background.New(context.Background(), func() *zap.Logger { return global.Logger })
	global.SetBackgroundTaskManager(mgr)
	t.Cleanup(func() {
		global.SetBackgroundTaskManager(oldManager)
		global.SetLogger(oldLogger)
	})

	started := make(chan struct{})
	finished := make(chan struct{})
	onErrorCalled := make(chan struct{}, 1)

	dispatchMailAttemptAsync(func(ctx context.Context) (MailAttemptSummary, error) {
		close(started)
		<-ctx.Done()
		close(finished)
		return MailAttemptSummary{}, ctx.Err()
	}, func(summary MailAttemptSummary, err error) {
		onErrorCalled <- struct{}{}
	}, "mail dispatch panic")

	select {
	case <-started:
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for async mail dispatch to start")
	}

	if err := mgr.Shutdown(time.Second); err != nil {
		t.Fatalf("Shutdown returned error: %v", err)
	}

	select {
	case <-finished:
	case <-time.After(time.Second):
		t.Fatal("expected async mail dispatch to stop when background manager shuts down")
	}

	select {
	case <-onErrorCalled:
		t.Fatal("expected shutdown cancellation to skip async mail error callback")
	case <-time.After(100 * time.Millisecond):
	}
}

func TestDispatchMailAttemptAsyncInvokesOnErrorOnTimeout(t *testing.T) {
	oldLogger := global.CurrentLogger()
	oldManager := global.BackgroundTaskManager()
	global.SetLogger(zap.NewNop())
	mgr := background.New(context.Background(), func() *zap.Logger { return global.Logger })
	global.SetBackgroundTaskManager(mgr)
	t.Cleanup(func() {
		global.SetBackgroundTaskManager(oldManager)
		global.SetLogger(oldLogger)
	})

	onErrorCalled := make(chan error, 1)
	dispatchMailAttemptAsync(func(ctx context.Context) (MailAttemptSummary, error) {
		return MailAttemptSummary{}, context.DeadlineExceeded
	}, func(summary MailAttemptSummary, err error) {
		onErrorCalled <- err
	}, "mail dispatch panic")

	select {
	case err := <-onErrorCalled:
		if !errors.Is(err, context.DeadlineExceeded) {
			t.Fatalf("onError err = %v, want %v", err, context.DeadlineExceeded)
		}
	case <-time.After(time.Second):
		t.Fatal("expected timeout to trigger async mail error callback")
	}

	if err := mgr.Shutdown(time.Second); err != nil {
		t.Fatalf("Shutdown returned error: %v", err)
	}
}

func TestDispatchMailAttemptAsyncRunsInlineWhenManagerIsStopping(t *testing.T) {
	oldLogger := global.CurrentLogger()
	oldManager := global.BackgroundTaskManager()
	global.SetLogger(zap.NewNop())
	mgr := background.New(context.Background(), func() *zap.Logger { return global.Logger })
	global.SetBackgroundTaskManager(mgr)
	t.Cleanup(func() {
		global.SetBackgroundTaskManager(oldManager)
		global.SetLogger(oldLogger)
	})

	if err := mgr.Shutdown(time.Second); err != nil {
		t.Fatalf("Shutdown returned error: %v", err)
	}

	ran := make(chan struct{}, 1)
	dispatchMailAttemptAsync(func(ctx context.Context) (MailAttemptSummary, error) {
		ran <- struct{}{}
		return MailAttemptSummary{}, nil
	}, nil, "mail dispatch panic")

	select {
	case <-ran:
	case <-time.After(time.Second):
		t.Fatal("expected async mail dispatch to run inline when background manager is stopping")
	}
}

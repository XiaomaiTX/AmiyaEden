package main

import (
	"amiya-eden/global"
	"amiya-eden/pkg/background"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"go.uber.org/zap"
)

func TestShutdownServerWaitsForCronAndBackgroundTasks(t *testing.T) {
	mgr := background.New(context.Background(), func() *zap.Logger { return zap.NewNop() })
	started := make(chan struct{})
	finished := make(chan struct{})

	mgr.Go("shutdown-test", func(ctx context.Context) error {
		close(started)
		<-ctx.Done()
		close(finished)
		return ctx.Err()
	})

	select {
	case <-started:
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for background task to start")
	}

	cronStopCtx, cancelCron := context.WithCancel(context.Background())
	defer cancelCron()

	done := make(chan struct{})
	go func() {
		shutdownServer(&http.Server{}, cronStopCtx, mgr, 2*time.Second)
		close(done)
	}()

	select {
	case <-done:
		t.Fatal("shutdown returned before cron stop completed")
	case <-time.After(150 * time.Millisecond):
	}

	cancelCron()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for shutdown to finish")
	}

	select {
	case <-finished:
	case <-time.After(time.Second):
		t.Fatal("expected shutdown to cancel background tasks")
	}
}

func TestShutdownServerKeepsBackgroundManagerAliveWhileRequestsDrain(t *testing.T) {
	oldLogger := global.CurrentLogger()
	global.SetLogger(zap.NewNop())
	t.Cleanup(func() {
		global.SetLogger(oldLogger)
	})

	mgr := background.New(context.Background(), func() *zap.Logger { return zap.NewNop() })
	handlerStarted := make(chan struct{})
	releaseHandler := make(chan struct{})
	requestDone := make(chan error, 1)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		close(handlerStarted)
		<-releaseHandler
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	go func() {
		resp, err := http.Get(srv.URL)
		if err == nil {
			_ = resp.Body.Close()
		}
		requestDone <- err
	}()

	select {
	case <-handlerStarted:
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for request handler to start")
	}

	shutdownDone := make(chan struct{})
	go func() {
		shutdownServer(srv.Config, nil, mgr, time.Second)
		close(shutdownDone)
	}()

	select {
	case <-mgr.Context().Done():
		close(releaseHandler)
		<-requestDone
		<-shutdownDone
		t.Fatal("background manager shut down before in-flight request drained")
	case <-time.After(150 * time.Millisecond):
	}

	close(releaseHandler)

	select {
	case err := <-requestDone:
		if err != nil {
			t.Fatalf("request returned error: %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for in-flight request to finish")
	}

	select {
	case <-shutdownDone:
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for shutdown to finish")
	}

	select {
	case <-mgr.Context().Done():
	case <-time.After(time.Second):
		t.Fatal("expected background manager to shut down after requests drained")
	}
}

func TestShutdownServerKeepsBackgroundManagerAliveWhileCronDrains(t *testing.T) {
	oldLogger := global.CurrentLogger()
	global.SetLogger(zap.NewNop())
	t.Cleanup(func() {
		global.SetLogger(oldLogger)
	})

	mgr := background.New(context.Background(), func() *zap.Logger { return zap.NewNop() })
	cronStopCtx, cancelCron := context.WithCancel(context.Background())
	defer cancelCron()

	shutdownDone := make(chan struct{})
	go func() {
		shutdownServer(&http.Server{}, cronStopCtx, mgr, time.Second)
		close(shutdownDone)
	}()

	select {
	case <-mgr.Context().Done():
		<-shutdownDone
		t.Fatal("background manager shut down before cron drain completed")
	case <-time.After(150 * time.Millisecond):
	}

	cancelCron()

	select {
	case <-shutdownDone:
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for shutdown to finish after cron drain completed")
	}
}

func TestShutdownServerHonorsDeadlineAfterHTTPDrainTimeout(t *testing.T) {
	oldLogger := global.CurrentLogger()
	global.SetLogger(zap.NewNop())
	t.Cleanup(func() {
		global.SetLogger(oldLogger)
	})

	mgr := background.New(context.Background(), func() *zap.Logger { return zap.NewNop() })
	backgroundTaskStarted := make(chan struct{})
	releaseBackgroundTask := make(chan struct{})
	if ok := mgr.Go("deadline-test", func(ctx context.Context) error {
		close(backgroundTaskStarted)
		<-releaseBackgroundTask
		return nil
	}); !ok {
		t.Fatal("expected background task to start")
	}

	select {
	case <-backgroundTaskStarted:
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for background task to start")
	}

	handlerStarted := make(chan struct{})
	releaseHandler := make(chan struct{})
	requestDone := make(chan error, 1)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		close(handlerStarted)
		<-releaseHandler
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	go func() {
		resp, err := http.Get(srv.URL)
		if err == nil {
			_ = resp.Body.Close()
		}
		requestDone <- err
	}()

	select {
	case <-handlerStarted:
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for request handler to start")
	}

	shutdownDone := make(chan struct{})
	go func() {
		shutdownServer(srv.Config, nil, mgr, 100*time.Millisecond)
		close(shutdownDone)
	}()

	select {
	case <-shutdownDone:
	case <-time.After(300 * time.Millisecond):
		close(releaseHandler)
		close(releaseBackgroundTask)
		<-requestDone
		<-shutdownDone
		t.Fatal("shutdown exceeded its deadline after HTTP drain timed out")
	}

	close(releaseHandler)
	close(releaseBackgroundTask)

	select {
	case <-requestDone:
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for request cleanup")
	}

	select {
	case <-mgr.Context().Done():
	default:
		t.Fatal("expected background manager to be canceled when shutdown deadline is exhausted")
	}
}

package global

import (
	"amiya-eden/pkg/background"
	"context"
	"testing"
	"time"

	"go.uber.org/zap"
)

func TestSetLoggerUpdatesLogger(t *testing.T) {
	oldLogger := Logger
	SetLogger(nil)
	t.Cleanup(func() {
		SetLogger(oldLogger)
	})

	if got := Logger; got != nil {
		t.Fatalf("Logger = %v, want nil", got)
	}

	logger := zap.NewNop()
	SetLogger(logger)

	if got := Logger; got != logger {
		t.Fatalf("Logger = %v, want %v", got, logger)
	}
}

func TestEnsureBackgroundTaskManagerCreatesReusableManager(t *testing.T) {
	oldManager := BackgroundTaskManager()
	SetBackgroundTaskManager(nil)
	t.Cleanup(func() {
		SetBackgroundTaskManager(oldManager)
	})

	manager := EnsureBackgroundTaskManager()
	if manager == nil {
		t.Fatal("EnsureBackgroundTaskManager() returned nil")
	}
	if got := EnsureBackgroundTaskManager(); got != manager {
		t.Fatal("expected EnsureBackgroundTaskManager() to reuse the existing manager")
	}

	ctx := BackgroundContext()
	if ctx == nil {
		t.Fatal("BackgroundContext() returned nil")
	}
	if ctx.Done() != manager.Context().Done() {
		t.Fatal("expected BackgroundContext() to expose the manager context")
	}

	if err := manager.Shutdown(time.Second); err != nil {
		t.Fatalf("Shutdown() error = %v", err)
	}

	select {
	case <-ctx.Done():
		if err := ctx.Err(); err != context.Canceled {
			t.Fatalf("BackgroundContext err = %v, want %v", err, context.Canceled)
		}
	case <-time.After(time.Second):
		t.Fatal("expected BackgroundContext() to be canceled when manager shuts down")
	}

	replacement := background.New(context.Background(), func() *zap.Logger { return zap.NewNop() })
	SetBackgroundTaskManager(replacement)
	t.Cleanup(func() {
		_ = replacement.Shutdown(time.Second)
	})
}

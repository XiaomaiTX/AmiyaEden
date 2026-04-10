package taskregistry

import (
	"context"
	"errors"
	"testing"
)

func TestRegisterAndGet(t *testing.T) {
	r := New()
	runCount := 0
	wantErr := errors.New("run failed")
	task := TaskDefinition{
		Name:        "test_task",
		Description: "A test task",
		Category:    TaskCategorySystem,
		Type:        TaskTypeRecurring,
		DefaultCron: "0 */5 * * * *",
		RunFunc: func(ctx context.Context) error {
			runCount++
			return wantErr
		},
	}
	r.Register(task)

	got, ok := r.Get("test_task")
	if !ok {
		t.Fatal("expected registered task to be returned")
	}
	if got.Name != task.Name {
		t.Fatalf("expected name %q, got %q", task.Name, got.Name)
	}
	if got.Description != task.Description {
		t.Fatalf("expected description %q, got %q", task.Description, got.Description)
	}
	if got.Category != task.Category {
		t.Fatalf("expected category %q, got %q", task.Category, got.Category)
	}
	if got.Type != task.Type {
		t.Fatalf("expected type %q, got %q", task.Type, got.Type)
	}
	if got.DefaultCron != task.DefaultCron {
		t.Fatalf("expected cron %q, got %q", task.DefaultCron, got.DefaultCron)
	}
	if got.RunFunc == nil {
		t.Fatal("expected run function to be registered")
	}
	if err := got.RunFunc(context.Background()); !errors.Is(err, wantErr) {
		t.Fatalf("expected run function error %v, got %v", wantErr, err)
	}
	if runCount != 1 {
		t.Fatalf("expected run function to be invoked once, got %d", runCount)
	}
}

func TestGetNotFound(t *testing.T) {
	r := New()
	_, ok := r.Get("nonexistent")
	if ok {
		t.Fatal("expected missing task lookup to return false")
	}
}

func TestAll(t *testing.T) {
	r := New()
	runCountA := 0
	runCountB := 0
	taskA := TaskDefinition{
		Name:        "a",
		Description: "Task A",
		Category:    TaskCategoryESI,
		Type:        TaskTypeRecurring,
		DefaultCron: "0 */5 * * * *",
		RunFunc: func(ctx context.Context) error {
			runCountA++
			return nil
		},
	}
	taskB := TaskDefinition{
		Name:        "b",
		Description: "Task B",
		Category:    TaskCategoryOperation,
		Type:        TaskTypeTriggered,
		RunFunc: func(ctx context.Context) error {
			runCountB++
			return nil
		},
	}
	r.Register(taskA)
	r.Register(taskB)

	all := r.All()
	if len(all) != 2 {
		t.Fatalf("expected 2 tasks, got %d", len(all))
	}

	byName := make(map[string]TaskDefinition, len(all))
	for _, task := range all {
		if _, exists := byName[task.Name]; exists {
			t.Fatalf("duplicate task %q returned from All", task.Name)
		}
		byName[task.Name] = task
	}

	for _, want := range []TaskDefinition{taskA, taskB} {
		got, ok := byName[want.Name]
		if !ok {
			t.Fatalf("expected task %q to be returned from All", want.Name)
		}
		if got.Description != want.Description {
			t.Fatalf("expected description %q for task %q, got %q", want.Description, want.Name, got.Description)
		}
		if got.Category != want.Category {
			t.Fatalf("expected category %q for task %q, got %q", want.Category, want.Name, got.Category)
		}
		if got.Type != want.Type {
			t.Fatalf("expected type %q for task %q, got %q", want.Type, want.Name, got.Type)
		}
		if got.DefaultCron != want.DefaultCron {
			t.Fatalf("expected cron %q for task %q, got %q", want.DefaultCron, want.Name, got.DefaultCron)
		}
		if got.RunFunc == nil {
			t.Fatalf("expected run function for task %q", want.Name)
		}
	}

	if err := byName[taskA.Name].RunFunc(context.Background()); err != nil {
		t.Fatalf("expected task %q run function to succeed, got %v", taskA.Name, err)
	}
	if err := byName[taskB.Name].RunFunc(context.Background()); err != nil {
		t.Fatalf("expected task %q run function to succeed, got %v", taskB.Name, err)
	}
	if runCountA != 1 {
		t.Fatalf("expected task %q run function to be invoked once, got %d", taskA.Name, runCountA)
	}
	if runCountB != 1 {
		t.Fatalf("expected task %q run function to be invoked once, got %d", taskB.Name, runCountB)
	}
}

func TestDuplicateRegisterPanics(t *testing.T) {
	r := New()
	task := TaskDefinition{Name: "dup", Category: TaskCategorySystem, Type: TaskTypeRecurring, RunFunc: func(ctx context.Context) error { return nil }}
	r.Register(task)

	defer func() {
		recovered := recover()
		if recovered == nil {
			t.Fatal("expected duplicate registration to panic")
		}
		message, ok := recovered.(string)
		if !ok {
			t.Fatalf("expected panic message to be a string, got %T", recovered)
		}
		if message != "task \"dup\" already registered" {
			t.Fatalf("expected duplicate registration panic message, got %q", message)
		}
	}()

	r.Register(task)
}

func TestTryLock(t *testing.T) {
	r := New()
	r.Register(TaskDefinition{Name: "locked", Category: TaskCategorySystem, Type: TaskTypeRecurring, RunFunc: func(ctx context.Context) error { return nil }})

	handle, ok := r.TryLock("locked")
	if !ok {
		t.Fatal("expected first lock attempt to succeed")
	}
	if handle == nil {
		t.Fatal("expected successful lock attempt to return a release handle")
	}
	if handle, ok := r.TryLock("locked"); ok {
		t.Fatal("expected second lock attempt to fail while held")
	} else if handle != nil {
		t.Fatal("expected failed lock attempt to return a nil release handle")
	}
	handle.Release()

	handle, ok = r.TryLock("locked")
	if !ok {
		t.Fatal("expected lock attempt after unlock to succeed")
	}
	handle.Release()
}

func TestTryLockUnknownTask(t *testing.T) {
	r := New()
	handle, ok := r.TryLock("nonexistent")
	if ok {
		t.Fatal("expected unknown task lock attempt to fail")
	}
	if handle != nil {
		t.Fatal("expected unknown task lock attempt to return a nil release handle")
	}
}

func TestTryLockFailedCallerCannotReleaseActiveLock(t *testing.T) {
	r := New()
	r.Register(TaskDefinition{Name: "running", Category: TaskCategorySystem, Type: TaskTypeRecurring, RunFunc: func(ctx context.Context) error { return nil }})

	handleA, ok := r.TryLock("running")
	if !ok {
		t.Fatal("expected first lock attempt to succeed")
	}
	if handleA == nil {
		t.Fatal("expected successful lock attempt to return a release handle")
	}

	handleB, ok := r.TryLock("running")
	if ok {
		t.Fatal("expected second lock attempt to fail while task is running")
	}
	if handleB != nil {
		t.Fatal("expected failed lock attempt to return a nil release handle")
	}
	if !r.IsRunning("running") {
		t.Fatal("expected task to remain running after failed lock attempt")
	}

	if handleC, ok := r.TryLock("running"); ok {
		t.Fatal("expected task to remain locked until the acquired handle is released")
	} else if handleC != nil {
		t.Fatal("expected repeated failed lock attempts to return a nil release handle")
	}

	handleA.Release()

	handle, ok := r.TryLock("running")
	if !ok {
		t.Fatal("expected task to become lockable after the acquired handle is released")
	}
	handle.Release()
}

func TestReleaseHandleIsIdempotent(t *testing.T) {
	r := New()
	r.Register(TaskDefinition{Name: "double_release", Category: TaskCategorySystem, Type: TaskTypeRecurring, RunFunc: func(ctx context.Context) error { return nil }})

	handle, ok := r.TryLock("double_release")
	if !ok {
		t.Fatal("expected initial lock attempt to succeed")
	}

	handle.Release()
	handle.Release()

	if r.IsRunning("double_release") {
		t.Fatal("expected task to remain not running after repeated release calls")
	}

	handle, ok = r.TryLock("double_release")
	if !ok {
		t.Fatal("expected task to be lockable after repeated release calls")
	}
	handle.Release()
}

func TestReleaseHandleCanBeCalledFromAnotherGoroutine(t *testing.T) {
	r := New()
	r.Register(TaskDefinition{Name: "cross_goroutine", Category: TaskCategorySystem, Type: TaskTypeRecurring, RunFunc: func(ctx context.Context) error { return nil }})

	handle, ok := r.TryLock("cross_goroutine")
	if !ok {
		t.Fatal("expected initial lock attempt to succeed")
	}

	done := make(chan struct{})
	go func() {
		handle.Release()
		close(done)
	}()
	<-done

	if r.IsRunning("cross_goroutine") {
		t.Fatal("expected task to be cleared after cross-goroutine release")
	}

	handle, ok = r.TryLock("cross_goroutine")
	if !ok {
		t.Fatal("expected task to be lockable after cross-goroutine release")
	}
	handle.Release()
}

func TestIsRunning(t *testing.T) {
	r := New()
	r.Register(TaskDefinition{Name: "running", Category: TaskCategorySystem, Type: TaskTypeRecurring, RunFunc: func(ctx context.Context) error { return nil }})

	if r.IsRunning("running") {
		t.Fatal("expected task to start unlocked")
	}
	handle, ok := r.TryLock("running")
	if !ok {
		t.Fatal("expected lock attempt to succeed")
	}
	if !r.IsRunning("running") {
		t.Fatal("expected task to be marked running while lock is held")
	}
	handle.Release()
	if r.IsRunning("running") {
		t.Fatal("expected task to be cleared after unlock")
	}
	if r.IsRunning("nonexistent") {
		t.Fatal("expected unknown task to report not running")
	}
}

func TestHoldsLockRequiresMatchingHandle(t *testing.T) {
	r := New()
	r.Register(TaskDefinition{Name: "locked", Category: TaskCategorySystem, Type: TaskTypeRecurring, RunFunc: func(ctx context.Context) error { return nil }})
	r.Register(TaskDefinition{Name: "other", Category: TaskCategorySystem, Type: TaskTypeRecurring, RunFunc: func(ctx context.Context) error { return nil }})

	handle, ok := r.TryLock("locked")
	if !ok {
		t.Fatal("expected first lock attempt to succeed")
	}
	defer handle.Release()

	if !r.HoldsLock("locked", handle) {
		t.Fatal("expected handle to match its owned lock")
	}
	if r.HoldsLock("other", handle) {
		t.Fatal("expected handle not to match a different task")
	}
	if r.HoldsLock("locked", nil) {
		t.Fatal("expected nil handle not to match a lock")
	}
}

func TestClaimLockConsumesOwnership(t *testing.T) {
	r := New()
	r.Register(TaskDefinition{Name: "locked", Category: TaskCategorySystem, Type: TaskTypeRecurring, RunFunc: func(ctx context.Context) error { return nil }})

	handle, ok := r.TryLock("locked")
	if !ok {
		t.Fatal("expected first lock attempt to succeed")
	}
	defer handle.Release()

	if !r.ClaimLock("locked", handle) {
		t.Fatal("expected first claim to succeed")
	}
	if r.ClaimLock("locked", handle) {
		t.Fatal("expected second claim to fail after handle is consumed")
	}
}

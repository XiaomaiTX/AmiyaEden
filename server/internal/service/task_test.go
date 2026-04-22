package service

import (
	"amiya-eden/global"
	"amiya-eden/internal/model"
	"amiya-eden/internal/repository"
	"amiya-eden/internal/taskregistry"
	"amiya-eden/pkg/background"
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"go.uber.org/zap"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func newTaskServiceTestDepsWithDB(t *testing.T) (*taskregistry.Registry, *repository.TaskRepository, *TaskService, *gorm.DB) {
	t.Helper()

	originalLogger := global.CurrentLogger()
	global.SetLogger(zap.NewNop())
	t.Cleanup(func() {
		global.SetLogger(originalLogger)
	})

	dsn := fmt.Sprintf("file:%s_%d?mode=memory&cache=shared", t.Name(), time.Now().UnixNano())
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&model.TaskSchedule{}, &model.TaskExecution{}, &model.User{}); err != nil {
		t.Fatalf("auto migrate task models: %v", err)
	}

	registry := taskregistry.New()
	repo := repository.NewTaskRepositoryWithDB(db)
	svc := NewTaskService(registry, repo, nil)

	return registry, repo, svc, db
}

func newTaskServiceTestDeps(t *testing.T) (*taskregistry.Registry, *repository.TaskRepository, *TaskService) {
	t.Helper()
	registry, repo, svc, _ := newTaskServiceTestDepsWithDB(t)
	return registry, repo, svc
}

func TestTaskService_RunTaskSuccess(t *testing.T) {
	registry, repo, svc, db := newTaskServiceTestDepsWithDB(t)
	triggeredBy := uint(42)
	if err := db.Create(&model.User{BaseModel: model.BaseModel{ID: triggeredBy}, Nickname: "Trigger Nick"}).Error; err != nil {
		t.Fatalf("create user fixture: %v", err)
	}
	registry.Register(taskregistry.TaskDefinition{
		Name:        "ok_task",
		Description: "Runs successfully",
		Category:    taskregistry.TaskCategorySystem,
		Type:        taskregistry.TaskTypeRecurring,
		DefaultCron: "0 */5 * * * *",
		RunFunc: func(ctx context.Context) error {
			if ctx == nil {
				t.Fatal("expected context to be passed to RunFunc")
			}
			return nil
		},
	})

	if err := svc.RunTask(context.Background(), "ok_task", &triggeredBy); err != nil {
		t.Fatalf("RunTask returned error: %v", err)
	}

	execs, total, err := repo.ListExecutions("ok_task", "", 1, 10)
	if err != nil {
		t.Fatalf("ListExecutions returned error: %v", err)
	}
	if total != 1 {
		t.Fatalf("execution total = %d, want 1", total)
	}
	if len(execs) != 1 {
		t.Fatalf("execution count = %d, want 1", len(execs))
	}
	if execs[0].Trigger != "manual" {
		t.Fatalf("trigger = %q, want manual", execs[0].Trigger)
	}
	if execs[0].TriggeredBy == nil || *execs[0].TriggeredBy != triggeredBy {
		t.Fatalf("triggered_by = %v, want %d", execs[0].TriggeredBy, triggeredBy)
	}
	if execs[0].TriggeredByName != "Trigger Nick" {
		t.Fatalf("triggered_by_name = %q, want %q", execs[0].TriggeredByName, "Trigger Nick")
	}
	if execs[0].Status != "success" {
		t.Fatalf("status = %q, want success", execs[0].Status)
	}
	if execs[0].FinishedAt == nil {
		t.Fatal("expected finished_at to be set")
	}
	if execs[0].DurationMs == nil {
		t.Fatal("expected duration_ms to be set")
	}
}

func TestTaskService_RunTaskFailureFromRunFunc(t *testing.T) {
	registry, repo, svc := newTaskServiceTestDeps(t)
	runErr := errors.New("boom")
	registry.Register(taskregistry.TaskDefinition{
		Name:        "fail_task",
		Description: "Fails",
		Category:    taskregistry.TaskCategoryOperation,
		Type:        taskregistry.TaskTypeRecurring,
		DefaultCron: "0 */5 * * * *",
		RunFunc: func(ctx context.Context) error {
			return runErr
		},
	})

	err := svc.RunTask(context.Background(), "fail_task", nil)
	if !errors.Is(err, runErr) {
		t.Fatalf("RunTask error = %v, want %v", err, runErr)
	}

	last, err := repo.GetLastExecution("fail_task")
	if err != nil {
		t.Fatalf("GetLastExecution returned error: %v", err)
	}
	if last == nil {
		t.Fatal("expected persisted execution")
	}
	if last.Status != "failed" {
		t.Fatalf("status = %q, want failed", last.Status)
	}
	if last.Error != runErr.Error() {
		t.Fatalf("error = %q, want %q", last.Error, runErr.Error())
	}
}

func TestTaskService_RunTaskRecoversPanic(t *testing.T) {
	registry, repo, svc := newTaskServiceTestDeps(t)
	registry.Register(taskregistry.TaskDefinition{
		Name:        "panic_task",
		Description: "Panics",
		Category:    taskregistry.TaskCategorySystem,
		Type:        taskregistry.TaskTypeRecurring,
		DefaultCron: "0 */5 * * * *",
		RunFunc: func(ctx context.Context) error {
			panic("boom")
		},
	})

	err := svc.RunTask(context.Background(), "panic_task", nil)
	if err == nil {
		t.Fatal("expected RunTask to recover and return a panic error")
	}
	if !strings.Contains(err.Error(), "task panicked") {
		t.Fatalf("RunTask error = %v, want panic marker", err)
	}

	last, getErr := repo.GetLastExecution("panic_task")
	if getErr != nil {
		t.Fatalf("GetLastExecution returned error: %v", getErr)
	}
	if last == nil {
		t.Fatal("expected persisted execution after panic")
	}
	if last.Status != taskStatusFailed {
		t.Fatalf("status = %q, want %q", last.Status, taskStatusFailed)
	}

	if err := svc.RunTask(context.Background(), "panic_task", nil); err == nil {
		t.Fatal("expected lock to be released so second panic run also returns an error")
	}
}

func TestTaskService_RunTaskAlreadyRunning(t *testing.T) {
	registry, _, svc := newTaskServiceTestDeps(t)
	entered := make(chan struct{})
	release := make(chan struct{})
	firstDone := make(chan error, 1)

	registry.Register(taskregistry.TaskDefinition{
		Name:        "slow_task",
		Description: "Blocks until released",
		Category:    taskregistry.TaskCategorySystem,
		Type:        taskregistry.TaskTypeRecurring,
		DefaultCron: "0 */5 * * * *",
		RunFunc: func(ctx context.Context) error {
			close(entered)
			<-release
			return nil
		},
	})

	go func() {
		firstDone <- svc.RunTask(context.Background(), "slow_task", nil)
	}()

	select {
	case <-entered:
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for first run to enter RunFunc")
	}

	err := svc.RunTask(context.Background(), "slow_task", nil)
	if !errors.Is(err, ErrTaskAlreadyRunning) {
		t.Fatalf("RunTask error = %v, want ErrTaskAlreadyRunning", err)
	}

	close(release)
	select {
	case err := <-firstDone:
		if err != nil {
			t.Fatalf("first RunTask returned error: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for first run to finish")
	}
}

func TestTaskService_RunTaskUnknownTask(t *testing.T) {
	_, _, svc := newTaskServiceTestDeps(t)

	err := svc.RunTask(context.Background(), "missing_task", nil)
	if !errors.Is(err, ErrTaskNotFound) {
		t.Fatalf("RunTask error = %v, want ErrTaskNotFound", err)
	}
}

func TestTaskService_RunTaskNotRunnable(t *testing.T) {
	registry, _, svc := newTaskServiceTestDeps(t)
	registry.Register(taskregistry.TaskDefinition{
		Name:        "triggered_only",
		Description: "Has no run function",
		Category:    taskregistry.TaskCategoryOperation,
		Type:        taskregistry.TaskTypeTriggered,
	})

	err := svc.RunTask(context.Background(), "triggered_only", nil)
	if !errors.Is(err, ErrTaskNotRunnable) {
		t.Fatalf("RunTask error = %v, want ErrTaskNotRunnable", err)
	}
}

func TestTaskService_RunTaskFromCronUsesBackgroundTaskManagerContext(t *testing.T) {
	registry, _, svc := newTaskServiceTestDeps(t)
	oldManager := global.BackgroundTaskManager()
	mgr := background.New(context.Background(), func() *zap.Logger { return global.Logger })
	global.SetBackgroundTaskManager(mgr)
	t.Cleanup(func() {
		global.SetBackgroundTaskManager(oldManager)
	})

	if err := mgr.Shutdown(time.Second); err != nil {
		t.Fatalf("Shutdown returned error: %v", err)
	}

	seenCtxErr := make(chan error, 1)
	registry.Register(taskregistry.TaskDefinition{
		Name:        "cron_context_task",
		Description: "Observes cron context",
		Category:    taskregistry.TaskCategorySystem,
		Type:        taskregistry.TaskTypeRecurring,
		DefaultCron: "0 */5 * * * *",
		RunFunc: func(ctx context.Context) error {
			seenCtxErr <- ctx.Err()
			return nil
		},
	})

	svc.RunTaskFromCron("cron_context_task")

	select {
	case err := <-seenCtxErr:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("cron task context error = %v, want %v", err, context.Canceled)
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for cron task execution")
	}
}

func TestTaskService_UpdateScheduleValidRecurringTaskPersistsSchedule(t *testing.T) {
	registry, repo, svc := newTaskServiceTestDeps(t)
	registry.Register(taskregistry.TaskDefinition{
		Name:        "cron_task",
		Description: "Recurring task",
		Category:    taskregistry.TaskCategorySystem,
		Type:        taskregistry.TaskTypeRecurring,
		DefaultCron: "0 */5 * * * *",
		RunFunc: func(ctx context.Context) error {
			return nil
		},
	})

	rescheduleCalls := 0
	var rescheduledTask string
	var rescheduledCron string
	svc.reschedule = func(taskName, cronExpr string) error {
		rescheduleCalls++
		rescheduledTask = taskName
		rescheduledCron = cronExpr
		return nil
	}

	err := svc.UpdateSchedule("cron_task", "0 */10 * * * *", 7)
	if err != nil {
		t.Fatalf("UpdateSchedule returned error: %v", err)
	}

	schedule, err := repo.GetSchedule("cron_task")
	if err != nil {
		t.Fatalf("GetSchedule returned error: %v", err)
	}
	if schedule.CronExpr != "0 */10 * * * *" {
		t.Fatalf("cron_expr = %q, want %q", schedule.CronExpr, "0 */10 * * * *")
	}
	if schedule.UpdatedBy != 7 {
		t.Fatalf("updated_by = %d, want 7", schedule.UpdatedBy)
	}
	if rescheduleCalls != 1 {
		t.Fatalf("reschedule call count = %d, want 1", rescheduleCalls)
	}
	if rescheduledTask != "cron_task" || rescheduledCron != "0 */10 * * * *" {
		t.Fatalf("reschedule args = (%q, %q), want (%q, %q)", rescheduledTask, rescheduledCron, "cron_task", "0 */10 * * * *")
	}
}

func TestTaskService_UpdateScheduleInvalidCron(t *testing.T) {
	registry, _, svc := newTaskServiceTestDeps(t)
	registry.Register(taskregistry.TaskDefinition{
		Name:        "cron_task_invalid",
		Description: "Recurring task",
		Category:    taskregistry.TaskCategorySystem,
		Type:        taskregistry.TaskTypeRecurring,
		DefaultCron: "0 */5 * * * *",
		RunFunc: func(ctx context.Context) error {
			return nil
		},
	})

	err := svc.UpdateSchedule("cron_task_invalid", "not-a-cron", 1)
	if !errors.Is(err, ErrInvalidCronExpr) {
		t.Fatalf("UpdateSchedule error = %v, want ErrInvalidCronExpr", err)
	}
}

func TestTaskService_UpdateScheduleRequiresConfiguredRescheduler(t *testing.T) {
	registry, repo, svc := newTaskServiceTestDeps(t)
	registry.Register(taskregistry.TaskDefinition{
		Name:        "cron_task_no_rescheduler",
		Description: "Recurring task",
		Category:    taskregistry.TaskCategorySystem,
		Type:        taskregistry.TaskTypeRecurring,
		DefaultCron: "0 */5 * * * *",
		RunFunc: func(ctx context.Context) error {
			return nil
		},
	})

	err := svc.UpdateSchedule("cron_task_no_rescheduler", "0 */10 * * * *", 1)
	if err == nil {
		t.Fatal("expected UpdateSchedule without a configured rescheduler to return an error")
	}

	schedule, getErr := repo.GetSchedule("cron_task_no_rescheduler")
	if !errors.Is(getErr, gorm.ErrRecordNotFound) {
		t.Fatalf("GetSchedule error = %v, want gorm.ErrRecordNotFound", getErr)
	}
	if schedule != nil {
		t.Fatalf("expected no persisted schedule without a configured rescheduler, got %#v", schedule)
	}
}

func TestTaskService_UpdateScheduleReturnsRescheduleError(t *testing.T) {
	registry, repo, svc := newTaskServiceTestDeps(t)
	registry.Register(taskregistry.TaskDefinition{
		Name:        "cron_task_reschedule_error",
		Description: "Recurring task",
		Category:    taskregistry.TaskCategorySystem,
		Type:        taskregistry.TaskTypeRecurring,
		DefaultCron: "0 */5 * * * *",
		RunFunc: func(ctx context.Context) error {
			return nil
		},
	})

	wantErr := errors.New("reschedule failed")
	svc.reschedule = func(taskName, cronExpr string) error {
		return wantErr
	}

	err := svc.UpdateSchedule("cron_task_reschedule_error", "0 */10 * * * *", 1)
	if !errors.Is(err, wantErr) {
		t.Fatalf("UpdateSchedule error = %v, want %v", err, wantErr)
	}

	schedule, getErr := repo.GetSchedule("cron_task_reschedule_error")
	if !errors.Is(getErr, gorm.ErrRecordNotFound) {
		t.Fatalf("GetSchedule error = %v, want gorm.ErrRecordNotFound", getErr)
	}
	if schedule != nil {
		t.Fatalf("expected no persisted schedule after reschedule failure, got %#v", schedule)
	}
}

func TestTaskService_UpdateScheduleRevertsRuntimeWhenPersistenceFails(t *testing.T) {
	registry, repo, svc, db := newTaskServiceTestDepsWithDB(t)
	registry.Register(taskregistry.TaskDefinition{
		Name:        "cron_task_persist_error",
		Description: "Recurring task",
		Category:    taskregistry.TaskCategorySystem,
		Type:        taskregistry.TaskTypeRecurring,
		DefaultCron: "0 */5 * * * *",
		RunFunc: func(ctx context.Context) error {
			return nil
		},
	})

	if err := repo.UpsertSchedule(&model.TaskSchedule{
		TaskName:  "cron_task_persist_error",
		CronExpr:  "0 */15 * * * *",
		UpdatedBy: 1,
	}); err != nil {
		t.Fatalf("seed existing schedule: %v", err)
	}

	rescheduled := make([]string, 0, 2)
	svc.reschedule = func(taskName, cronExpr string) error {
		rescheduled = append(rescheduled, cronExpr)
		return nil
	}

	persistErr := errors.New("persist failed")
	const callbackName = "task_schedule_persist_fail"
	if err := db.Callback().Create().Before("gorm:create").Register(callbackName, func(tx *gorm.DB) {
		if tx.Statement != nil && tx.Statement.Table == "task_schedules" {
			if err := tx.AddError(persistErr); err != nil && !errors.Is(err, persistErr) {
				t.Fatalf("inject persistence error: %v", err)
			}
		}
	}); err != nil {
		t.Fatalf("register failing create callback: %v", err)
	}
	t.Cleanup(func() {
		_ = db.Callback().Create().Remove(callbackName)
	})

	err := svc.UpdateSchedule("cron_task_persist_error", "0 */10 * * * *", 2)
	if !errors.Is(err, persistErr) {
		t.Fatalf("UpdateSchedule error = %v, want %v", err, persistErr)
	}
	if len(rescheduled) != 2 {
		t.Fatalf("reschedule call count = %d, want 2", len(rescheduled))
	}
	if rescheduled[0] != "0 */10 * * * *" || rescheduled[1] != "0 */15 * * * *" {
		t.Fatalf("reschedule sequence = %#v, want [new, previous]", rescheduled)
	}

	schedule, getErr := repo.GetSchedule("cron_task_persist_error")
	if getErr != nil {
		t.Fatalf("GetSchedule returned error: %v", getErr)
	}
	if schedule == nil {
		t.Fatal("expected original schedule to remain after persistence failure")
	}
	if schedule.CronExpr != "0 */15 * * * *" {
		t.Fatalf("cron_expr = %q, want original %q", schedule.CronExpr, "0 */15 * * * *")
	}
}

func TestTaskService_UpdateScheduleSerializesConcurrentUpdates(t *testing.T) {
	registry, repo, svc := newTaskServiceTestDeps(t)
	registry.Register(taskregistry.TaskDefinition{
		Name:        "cron_task_serialized",
		Description: "Recurring task",
		Category:    taskregistry.TaskCategorySystem,
		Type:        taskregistry.TaskTypeRecurring,
		DefaultCron: "0 */5 * * * *",
		RunFunc: func(ctx context.Context) error {
			return nil
		},
	})

	firstEntered := make(chan struct{})
	secondEntered := make(chan struct{})
	releaseFirst := make(chan struct{})
	callCount := 0
	svc.reschedule = func(taskName, cronExpr string) error {
		callCount++
		switch callCount {
		case 1:
			close(firstEntered)
			<-releaseFirst
		case 2:
			close(secondEntered)
		}
		return nil
	}

	firstDone := make(chan error, 1)
	secondDone := make(chan error, 1)
	go func() {
		firstDone <- svc.UpdateSchedule("cron_task_serialized", "0 */10 * * * *", 1)
	}()

	select {
	case <-firstEntered:
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for first UpdateSchedule call to enter rescheduler")
	}

	go func() {
		secondDone <- svc.UpdateSchedule("cron_task_serialized", "0 */20 * * * *", 2)
	}()

	select {
	case <-secondEntered:
		t.Fatal("expected second UpdateSchedule call to wait for the first")
	case <-time.After(100 * time.Millisecond):
	}

	close(releaseFirst)

	select {
	case err := <-firstDone:
		if err != nil {
			t.Fatalf("first UpdateSchedule returned error: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for first UpdateSchedule call to finish")
	}

	select {
	case <-secondEntered:
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for second UpdateSchedule call to enter rescheduler")
	}

	select {
	case err := <-secondDone:
		if err != nil {
			t.Fatalf("second UpdateSchedule returned error: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for second UpdateSchedule call to finish")
	}

	schedule, err := repo.GetSchedule("cron_task_serialized")
	if err != nil {
		t.Fatalf("GetSchedule returned error: %v", err)
	}
	if schedule.CronExpr != "0 */20 * * * *" {
		t.Fatalf("cron_expr = %q, want second update value", schedule.CronExpr)
	}
}

func TestTaskService_UpdateScheduleSerializesAcrossServiceInstances(t *testing.T) {
	registry, repo, svc, _ := newTaskServiceTestDepsWithDB(t)
	svc2 := NewTaskService(registry, repo, nil)
	registry.Register(taskregistry.TaskDefinition{
		Name:        "cron_task_multi_service",
		Description: "Recurring task",
		Category:    taskregistry.TaskCategorySystem,
		Type:        taskregistry.TaskTypeRecurring,
		DefaultCron: "0 */5 * * * *",
		RunFunc: func(ctx context.Context) error {
			return nil
		},
	})

	firstEntered := make(chan struct{})
	secondEntered := make(chan struct{})
	releaseFirst := make(chan struct{})
	callCount := 0
	reschedule := func(taskName, cronExpr string) error {
		callCount++
		switch callCount {
		case 1:
			close(firstEntered)
			<-releaseFirst
		case 2:
			close(secondEntered)
		}
		return nil
	}
	svc.reschedule = reschedule
	svc2.reschedule = reschedule

	firstDone := make(chan error, 1)
	secondDone := make(chan error, 1)
	go func() {
		firstDone <- svc.UpdateSchedule("cron_task_multi_service", "0 */10 * * * *", 1)
	}()

	select {
	case <-firstEntered:
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for first UpdateSchedule call to enter rescheduler")
	}

	go func() {
		secondDone <- svc2.UpdateSchedule("cron_task_multi_service", "0 */20 * * * *", 2)
	}()

	select {
	case <-secondEntered:
		t.Fatal("expected second service instance to wait for the first update")
	case <-time.After(100 * time.Millisecond):
	}

	close(releaseFirst)

	select {
	case err := <-firstDone:
		if err != nil {
			t.Fatalf("first UpdateSchedule returned error: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for first UpdateSchedule call to finish")
	}

	select {
	case <-secondEntered:
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for second UpdateSchedule call to enter rescheduler")
	}

	select {
	case err := <-secondDone:
		if err != nil {
			t.Fatalf("second UpdateSchedule returned error: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for second UpdateSchedule call to finish")
	}
}

func TestTaskService_UpdateScheduleAcceptsEveryDescriptor(t *testing.T) {
	registry, repo, svc, _ := newTaskServiceTestDepsWithDB(t)
	registry.Register(taskregistry.TaskDefinition{
		Name:        "every_task",
		Description: "Recurring task",
		Category:    taskregistry.TaskCategoryOperation,
		Type:        taskregistry.TaskTypeRecurring,
		DefaultCron: "@every 13h",
		RunFunc: func(ctx context.Context) error {
			return nil
		},
	})

	var rescheduledTask string
	var rescheduledCron string
	svc.reschedule = func(taskName, cronExpr string) error {
		rescheduledTask = taskName
		rescheduledCron = cronExpr
		return nil
	}

	err := svc.UpdateSchedule("every_task", "@every 100h", 9)
	if err != nil {
		t.Fatalf("UpdateSchedule returned error: %v", err)
	}
	if rescheduledTask != "every_task" {
		t.Fatalf("rescheduled task = %q, want %q", rescheduledTask, "every_task")
	}
	if rescheduledCron != "@every 100h" {
		t.Fatalf("rescheduled cron = %q, want %q", rescheduledCron, "@every 100h")
	}

	schedule, err := repo.GetSchedule("every_task")
	if err != nil {
		t.Fatalf("GetSchedule returned error: %v", err)
	}
	if schedule.CronExpr != "@every 100h" {
		t.Fatalf("cron_expr = %q, want %q", schedule.CronExpr, "@every 100h")
	}
	if schedule.UpdatedBy != 9 {
		t.Fatalf("updated_by = %d, want 9", schedule.UpdatedBy)
	}
}

func TestTaskService_UpdateScheduleNonRecurringTask(t *testing.T) {
	registry, _, svc := newTaskServiceTestDeps(t)
	registry.Register(taskregistry.TaskDefinition{
		Name:        "triggered_task",
		Description: "Triggered only",
		Category:    taskregistry.TaskCategoryOperation,
		Type:        taskregistry.TaskTypeTriggered,
	})

	err := svc.UpdateSchedule("triggered_task", "0 */5 * * * *", 1)
	if !errors.Is(err, ErrTaskNotRecurring) {
		t.Fatalf("UpdateSchedule error = %v, want ErrTaskNotRecurring", err)
	}
}

func TestTaskService_GetTasksReturnsMergedDefinitions(t *testing.T) {
	registry, repo, svc := newTaskServiceTestDeps(t)
	registry.Register(taskregistry.TaskDefinition{
		Name:        "recurring_task",
		Description: "Recurring task",
		Category:    taskregistry.TaskCategoryESI,
		Type:        taskregistry.TaskTypeRecurring,
		DefaultCron: "0 */5 * * * *",
		RunFunc: func(ctx context.Context) error {
			return nil
		},
	})
	registry.Register(taskregistry.TaskDefinition{
		Name:        "triggered_task",
		Description: "Triggered task",
		Category:    taskregistry.TaskCategoryOperation,
		Type:        taskregistry.TaskTypeTriggered,
	})

	if err := repo.UpsertSchedule(&model.TaskSchedule{
		TaskName:  "recurring_task",
		CronExpr:  "0 */10 * * * *",
		UpdatedBy: 9,
	}); err != nil {
		t.Fatalf("UpsertSchedule returned error: %v", err)
	}

	startedAt := time.Date(2026, time.April, 10, 18, 30, 0, 0, time.UTC)
	finishedAt := startedAt.Add(3 * time.Second)
	durationMs := int64(3000)
	if err := repo.CreateExecution(&model.TaskExecution{
		TaskName:   "recurring_task",
		Trigger:    "cron",
		Status:     "success",
		StartedAt:  startedAt,
		FinishedAt: &finishedAt,
		DurationMs: &durationMs,
		Summary:    "done",
	}); err != nil {
		t.Fatalf("CreateExecution returned error: %v", err)
	}

	items, err := svc.GetTasks()
	if err != nil {
		t.Fatalf("GetTasks returned error: %v", err)
	}
	if len(items) != 2 {
		t.Fatalf("task item count = %d, want 2", len(items))
	}

	byName := make(map[string]TaskItem, len(items))
	for _, item := range items {
		byName[item.Name] = item
	}

	recurring, ok := byName["recurring_task"]
	if !ok {
		t.Fatal("expected recurring_task in response")
	}
	if recurring.CronExpr != "0 */10 * * * *" {
		t.Fatalf("effective cron = %q, want %q", recurring.CronExpr, "0 */10 * * * *")
	}
	if recurring.DefaultCron != "0 */5 * * * *" {
		t.Fatalf("default cron = %q, want %q", recurring.DefaultCron, "0 */5 * * * *")
	}
	if recurring.LastExecution == nil {
		t.Fatal("expected last_execution for recurring_task")
	}
	if recurring.LastExecution.Status != "success" {
		t.Fatalf("last_execution.status = %q, want success", recurring.LastExecution.Status)
	}
	if !recurring.LastExecution.StartedAt.Equal(startedAt) {
		t.Fatalf("last_execution.started_at = %v, want %v", recurring.LastExecution.StartedAt, startedAt)
	}

	triggered, ok := byName["triggered_task"]
	if !ok {
		t.Fatal("expected triggered_task in response")
	}
	if triggered.CronExpr != "" {
		t.Fatalf("triggered task cron = %q, want empty string", triggered.CronExpr)
	}
	if triggered.DefaultCron != "" {
		t.Fatalf("triggered task default cron = %q, want empty string", triggered.DefaultCron)
	}
	if triggered.LastExecution != nil {
		t.Fatalf("triggered task last_execution = %#v, want nil", triggered.LastExecution)
	}
}

func TestTaskService_GetExecutionHistoryReturnsPaginatedRepositoryData(t *testing.T) {
	_, repo, svc := newTaskServiceTestDeps(t)
	base := time.Date(2026, time.April, 10, 19, 0, 0, 0, time.UTC)
	fixtures := []model.TaskExecution{
		{TaskName: "task-a", Trigger: "cron", Status: "success", StartedAt: base.Add(1 * time.Minute)},
		{TaskName: "task-a", Trigger: "manual", Status: "failed", StartedAt: base.Add(2 * time.Minute)},
		{TaskName: "task-b", Trigger: "cron", Status: "success", StartedAt: base.Add(3 * time.Minute)},
	}
	for _, fixture := range fixtures {
		fixture := fixture
		if err := repo.CreateExecution(&fixture); err != nil {
			t.Fatalf("CreateExecution returned error: %v", err)
		}
	}

	history, total, err := svc.GetExecutionHistory("task-a", "success", 1, 10)
	if err != nil {
		t.Fatalf("GetExecutionHistory returned error: %v", err)
	}
	if total != 1 {
		t.Fatalf("history total = %d, want 1", total)
	}
	if len(history) != 1 {
		t.Fatalf("history count = %d, want 1", len(history))
	}
	if history[0].TaskName != "task-a" || history[0].Status != "success" {
		t.Fatalf("history item = %#v, want task-a success", history[0])
	}
}

func TestTaskService_RunTaskFromCronLogsExecutionWithCronTrigger(t *testing.T) {
	registry, repo, svc := newTaskServiceTestDeps(t)
	registry.Register(taskregistry.TaskDefinition{
		Name:        "cron_task",
		Description: "Runs from cron",
		Category:    taskregistry.TaskCategorySystem,
		Type:        taskregistry.TaskTypeRecurring,
		DefaultCron: "0 */5 * * * *",
		RunFunc: func(ctx context.Context) error {
			return nil
		},
	})

	svc.RunTaskFromCron("cron_task")

	last, err := repo.GetLastExecution("cron_task")
	if err != nil {
		t.Fatalf("GetLastExecution returned error: %v", err)
	}
	if last == nil {
		t.Fatal("expected persisted execution")
	}
	if last.Trigger != "cron" {
		t.Fatalf("trigger = %q, want cron", last.Trigger)
	}
	if last.Status != "success" {
		t.Fatalf("status = %q, want success", last.Status)
	}
	if last.TriggeredBy != nil {
		t.Fatalf("triggered_by = %v, want nil", last.TriggeredBy)
	}
}

func TestTaskService_RunTaskFromCronSkipsWhenTaskAlreadyRunning(t *testing.T) {
	registry, repo, svc := newTaskServiceTestDeps(t)
	runs := 0
	registry.Register(taskregistry.TaskDefinition{
		Name:        "cron_locked_task",
		Description: "Already running task",
		Category:    taskregistry.TaskCategorySystem,
		Type:        taskregistry.TaskTypeRecurring,
		DefaultCron: "0 */5 * * * *",
		RunFunc: func(ctx context.Context) error {
			runs++
			return nil
		},
	})

	handle, ok := registry.TryLock("cron_locked_task")
	if !ok {
		t.Fatal("expected to acquire registry lock in test")
	}
	defer handle.Release()

	svc.RunTaskFromCron("cron_locked_task")

	if runs != 0 {
		t.Fatalf("run count = %d, want 0", runs)
	}

	history, total, err := repo.ListExecutions("cron_locked_task", "", 1, 10)
	if err != nil {
		t.Fatalf("ListExecutions returned error: %v", err)
	}
	if total != 0 || len(history) != 0 {
		t.Fatalf("cron conflict should not create execution rows, got total=%d len=%d", total, len(history))
	}
}

func TestTaskService_RunTaskLockedExecutesWithoutReacquiringLock(t *testing.T) {
	registry, repo, svc := newTaskServiceTestDeps(t)
	runs := 0
	registry.Register(taskregistry.TaskDefinition{
		Name:        "locked_task",
		Description: "Requires external lock",
		Category:    taskregistry.TaskCategorySystem,
		Type:        taskregistry.TaskTypeRecurring,
		DefaultCron: "0 */5 * * * *",
		RunFunc: func(ctx context.Context) error {
			runs++
			return nil
		},
	})

	handle, ok := registry.TryLock("locked_task")
	if !ok {
		t.Fatal("expected to acquire registry lock in test")
	}
	defer handle.Release()

	triggeredBy := uint(99)
	ctx := taskregistry.ContextWithLockHandle(context.Background(), handle)
	if err := svc.RunTaskLocked(ctx, "locked_task", &triggeredBy); err != nil {
		t.Fatalf("RunTaskLocked returned error: %v", err)
	}
	if runs != 1 {
		t.Fatalf("run count = %d, want 1", runs)
	}

	last, err := repo.GetLastExecution("locked_task")
	if err != nil {
		t.Fatalf("GetLastExecution returned error: %v", err)
	}
	if last == nil {
		t.Fatal("expected persisted execution")
	}
	if last.Trigger != "manual" {
		t.Fatalf("trigger = %q, want manual", last.Trigger)
	}
	if last.TriggeredBy == nil || *last.TriggeredBy != triggeredBy {
		t.Fatalf("triggered_by = %v, want %d", last.TriggeredBy, triggeredBy)
	}
}

func TestTaskService_RunTaskLockedRequiresHeldLock(t *testing.T) {
	registry, repo, svc := newTaskServiceTestDeps(t)
	runs := 0
	registry.Register(taskregistry.TaskDefinition{
		Name:        "lock_required_task",
		Description: "Requires held lock",
		Category:    taskregistry.TaskCategorySystem,
		Type:        taskregistry.TaskTypeRecurring,
		DefaultCron: "0 */5 * * * *",
		RunFunc: func(ctx context.Context) error {
			runs++
			return nil
		},
	})

	err := svc.RunTaskLocked(context.Background(), "lock_required_task", nil)
	if err == nil {
		t.Fatal("expected RunTaskLocked without a held lock to return an error")
	}
	if runs != 0 {
		t.Fatalf("run count = %d, want 0", runs)
	}

	history, total, historyErr := repo.ListExecutions("lock_required_task", "", 1, 10)
	if historyErr != nil {
		t.Fatalf("ListExecutions returned error: %v", historyErr)
	}
	if total != 0 || len(history) != 0 {
		t.Fatalf("RunTaskLocked without lock should not create execution rows, got total=%d len=%d", total, len(history))
	}
}

func TestTaskService_RunTaskLockedRequiresOwnershipHandle(t *testing.T) {
	registry, repo, svc := newTaskServiceTestDeps(t)
	runs := 0
	registry.Register(taskregistry.TaskDefinition{
		Name:        "owned_task",
		Description: "Requires matching lock handle",
		Category:    taskregistry.TaskCategorySystem,
		Type:        taskregistry.TaskTypeRecurring,
		DefaultCron: "0 */5 * * * *",
		RunFunc: func(ctx context.Context) error {
			runs++
			return nil
		},
	})

	handle, ok := registry.TryLock("owned_task")
	if !ok {
		t.Fatal("expected to acquire registry lock in test")
	}
	defer handle.Release()

	err := svc.RunTaskLocked(context.Background(), "owned_task", nil)
	if !errors.Is(err, ErrTaskLockNotHeld) {
		t.Fatalf("RunTaskLocked error = %v, want ErrTaskLockNotHeld", err)
	}
	if runs != 0 {
		t.Fatalf("run count = %d, want 0", runs)
	}

	ctx := taskregistry.ContextWithLockHandle(context.Background(), handle)
	if err := svc.RunTaskLocked(ctx, "owned_task", nil); err != nil {
		t.Fatalf("RunTaskLocked with ownership handle returned error: %v", err)
	}
	if runs != 1 {
		t.Fatalf("run count = %d, want 1", runs)
	}

	history, total, historyErr := repo.ListExecutions("owned_task", "", 1, 10)
	if historyErr != nil {
		t.Fatalf("ListExecutions returned error: %v", historyErr)
	}
	if total != 1 || len(history) != 1 {
		t.Fatalf("RunTaskLocked with handle should create one execution row, got total=%d len=%d", total, len(history))
	}
}

func TestTaskService_RunTaskLockedRejectsReusedOwnershipHandle(t *testing.T) {
	registry, _, svc := newTaskServiceTestDeps(t)
	entered := make(chan struct{})
	finish := make(chan struct{})
	done := make(chan error, 1)
	registry.Register(taskregistry.TaskDefinition{
		Name:        "single_use_handle_task",
		Description: "Consumes its lock handle once",
		Category:    taskregistry.TaskCategorySystem,
		Type:        taskregistry.TaskTypeRecurring,
		DefaultCron: "0 */5 * * * *",
		RunFunc: func(ctx context.Context) error {
			close(entered)
			<-finish
			return nil
		},
	})

	handle, ok := registry.TryLock("single_use_handle_task")
	if !ok {
		t.Fatal("expected to acquire registry lock in test")
	}
	ctx := taskregistry.ContextWithLockHandle(context.Background(), handle)

	go func() {
		done <- svc.RunTaskLocked(ctx, "single_use_handle_task", nil)
	}()

	select {
	case <-entered:
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for first RunTaskLocked call to enter RunFunc")
	}

	err := svc.RunTaskLocked(ctx, "single_use_handle_task", nil)
	if !errors.Is(err, ErrTaskLockNotHeld) {
		t.Fatalf("RunTaskLocked error = %v, want ErrTaskLockNotHeld for reused handle", err)
	}

	close(finish)
	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("first RunTaskLocked returned error: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for first RunTaskLocked call to finish")
	}
}

func TestTaskService_RunTaskLockedRecoversPanic(t *testing.T) {
	registry, repo, svc := newTaskServiceTestDeps(t)
	registry.Register(taskregistry.TaskDefinition{
		Name:        "panic_locked_task",
		Description: "Panics under lock-aware execution",
		Category:    taskregistry.TaskCategorySystem,
		Type:        taskregistry.TaskTypeRecurring,
		DefaultCron: "0 */5 * * * *",
		RunFunc: func(ctx context.Context) error {
			panic("locked boom")
		},
	})

	handle, ok := registry.TryLock("panic_locked_task")
	if !ok {
		t.Fatal("expected to acquire registry lock in test")
	}
	ctx := taskregistry.ContextWithLockHandle(context.Background(), handle)

	err := svc.RunTaskLocked(ctx, "panic_locked_task", nil)
	if err == nil {
		t.Fatal("expected RunTaskLocked to recover and return a panic error")
	}
	if !strings.Contains(err.Error(), "task panicked") {
		t.Fatalf("RunTaskLocked error = %v, want panic marker", err)
	}

	last, getErr := repo.GetLastExecution("panic_locked_task")
	if getErr != nil {
		t.Fatalf("GetLastExecution returned error: %v", getErr)
	}
	if last == nil {
		t.Fatal("expected persisted execution after panic")
	}
	if last.Status != taskStatusFailed {
		t.Fatalf("status = %q, want %q", last.Status, taskStatusFailed)
	}
}

package service

import (
	"amiya-eden/global"
	"amiya-eden/internal/model"
	"amiya-eden/internal/repository"
	"amiya-eden/internal/taskregistry"
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/robfig/cron/v3"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

var (
	ErrTaskAlreadyRunning         = errors.New("task is already running")
	ErrTaskNotFound               = errors.New("task not found")
	ErrTaskNotRunnable            = errors.New("task is not runnable")
	ErrTaskNotRecurring           = errors.New("task is not recurring")
	ErrInvalidCronExpr            = errors.New("invalid cron expression")
	ErrTaskLockNotHeld            = errors.New("task lock is not held")
	errTaskReschedulerUnavailable = errors.New("task rescheduler is not configured")

	taskCronParser    = cron.NewParser(cron.Second | cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow | cron.Descriptor)
	nopTaskLogger     = zap.NewNop()
	taskScheduleLocks sync.Map
)

const (
	taskTriggerManual  = "manual"
	taskTriggerCron    = "cron"
	taskStatusRunning  = "running"
	taskStatusSuccess  = "success"
	taskStatusFailed   = "failed"
	taskHistoryMaxSize = 1000
)

type TaskService struct {
	registry   *taskregistry.Registry
	repo       *repository.TaskRepository
	reschedule func(taskName, cronExpr string) error
	auditSvc   *AuditService
}

type TaskItem struct {
	Name          string                    `json:"name"`
	Description   string                    `json:"description"`
	Category      taskregistry.TaskCategory `json:"category"`
	Type          taskregistry.TaskType     `json:"type"`
	Runnable      bool                      `json:"runnable"`
	CronExpr      string                    `json:"cron_expr"`
	DefaultCron   string                    `json:"default_cron"`
	LastExecution *TaskLastExecution        `json:"last_execution"`
}

type TaskLastExecution struct {
	Status     string     `json:"status"`
	StartedAt  time.Time  `json:"started_at"`
	FinishedAt *time.Time `json:"finished_at,omitempty"`
	DurationMs *int64     `json:"duration_ms,omitempty"`
	Error      string     `json:"error,omitempty"`
	Summary    string     `json:"summary,omitempty"`
}

func NewTaskService(registry *taskregistry.Registry, repo *repository.TaskRepository, reschedule func(taskName, cronExpr string) error) *TaskService {
	return &TaskService{
		registry:   registry,
		repo:       repo,
		reschedule: reschedule,
		auditSvc:   NewAuditService(),
	}
}

func (s *TaskService) GetTasks() ([]TaskItem, error) {
	definitions := s.registry.All()
	taskNames := make([]string, 0, len(definitions))
	for _, definition := range definitions {
		taskNames = append(taskNames, definition.Name)
	}

	schedules, err := s.repo.ListAllSchedules()
	if err != nil {
		return nil, err
	}
	scheduleByTask := make(map[string]string, len(schedules))
	for _, schedule := range schedules {
		scheduleByTask[schedule.TaskName] = schedule.CronExpr
	}

	lastExecutions, err := s.repo.GetLastExecutions(taskNames)
	if err != nil {
		return nil, err
	}

	items := make([]TaskItem, 0, len(definitions))
	for _, definition := range definitions {
		item := TaskItem{
			Name:        definition.Name,
			Description: definition.Description,
			Category:    definition.Category,
			Type:        definition.Type,
			Runnable:    definition.RunFunc != nil,
			DefaultCron: definition.DefaultCron,
		}

		if definition.Type == taskregistry.TaskTypeRecurring {
			item.CronExpr = definition.DefaultCron
			if schedule, ok := scheduleByTask[definition.Name]; ok {
				item.CronExpr = schedule
			}
		}

		if execution := lastExecutions[definition.Name]; execution != nil {
			item.LastExecution = &TaskLastExecution{
				Status:     execution.Status,
				StartedAt:  execution.StartedAt,
				FinishedAt: execution.FinishedAt,
				DurationMs: execution.DurationMs,
				Error:      execution.Error,
				Summary:    execution.Summary,
			}
		}

		items = append(items, item)
	}

	sort.Slice(items, func(i, j int) bool {
		return items[i].Name < items[j].Name
	})

	return items, nil
}

func (s *TaskService) RunTask(ctx context.Context, taskName string, triggeredBy *uint) error {
	definition, err := s.getRunnableTask(taskName)
	if err != nil {
		return err
	}

	release, ok := s.registry.TryLock(taskName)
	if !ok {
		return ErrTaskAlreadyRunning
	}
	defer release.Release()

	return s.executeAndLog(ctx, taskName, taskTriggerManual, triggeredBy, definition.RunFunc)
}

func (s *TaskService) RunTaskLocked(ctx context.Context, taskName string, triggeredBy *uint) error {
	definition, err := s.getRunnableTask(taskName)
	if err != nil {
		return err
	}
	handle := taskregistry.LockHandleFromContext(ctx)
	if !s.registry.ClaimLock(taskName, handle) {
		return ErrTaskLockNotHeld
	}
	defer handle.Release()

	return s.executeAndLog(ctx, taskName, taskTriggerManual, triggeredBy, definition.RunFunc)
}

func (s *TaskService) RunTaskFromCron(taskName string) {
	definition, err := s.getRunnableTask(taskName)
	if err != nil {
		return
	}

	release, ok := s.registry.TryLock(taskName)
	if !ok {
		s.logger().Debug("task already running; skipping cron trigger", zap.String("task_name", taskName))
		return
	}
	defer release.Release()

	if err := s.executeAndLog(global.BackgroundContext(), taskName, taskTriggerCron, nil, definition.RunFunc); err != nil {
		s.logger().Warn("task execution failed", zap.String("task_name", taskName), zap.String("trigger", taskTriggerCron), zap.Error(err))
	}
}

func (s *TaskService) UpdateSchedule(taskName, cronExpr string, updatedBy uint) error {
	scheduleLock := s.scheduleLock(taskName)
	scheduleLock.Lock()
	defer scheduleLock.Unlock()

	definition, ok := s.registry.Get(taskName)
	if !ok {
		return ErrTaskNotFound
	}
	if definition.Type != taskregistry.TaskTypeRecurring {
		return ErrTaskNotRecurring
	}

	normalizedCron := strings.TrimSpace(cronExpr)
	if _, err := taskCronParser.Parse(normalizedCron); err != nil {
		return fmt.Errorf("%w: %v", ErrInvalidCronExpr, err)
	}

	previousCron := definition.DefaultCron
	if existingSchedule, err := s.repo.GetSchedule(taskName); err == nil && existingSchedule != nil {
		previousCron = existingSchedule.CronExpr
	} else if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}
	if s.reschedule == nil {
		return errTaskReschedulerUnavailable
	}

	if err := s.reschedule(taskName, normalizedCron); err != nil {
		return err
	}

	if err := s.repo.UpsertSchedule(&model.TaskSchedule{
		TaskName:  taskName,
		CronExpr:  normalizedCron,
		UpdatedBy: updatedBy,
	}); err != nil {
		if revertErr := s.reschedule(taskName, previousCron); revertErr != nil {
			s.logger().Error("failed to revert runtime schedule after persistence error", zap.String("task_name", taskName), zap.String("revert_cron", previousCron), zap.Error(revertErr))
		}
		return err
	}
	if s.auditSvc != nil {
		_ = s.auditSvc.RecordEvent(context.Background(), AuditRecordInput{
			Category:     "task_ops",
			Action:       "task_schedule_update",
			ActorUserID:  updatedBy,
			ResourceType: "task_schedule",
			ResourceID:   taskName,
			Result:       model.AuditResultSuccess,
			Details: map[string]any{
				"before_cron": previousCron,
				"after_cron":  normalizedCron,
			},
		})
	}

	return nil
}

func (s *TaskService) GetExecutionHistory(taskName, status string, page, pageSize int) ([]model.TaskExecutionHistoryItem, int64, error) {
	normalizeLedgerPageRequest(&page, &pageSize)
	if pageSize > taskHistoryMaxSize {
		pageSize = taskHistoryMaxSize
	}
	return s.repo.ListExecutions(taskName, status, page, pageSize)
}

func (s *TaskService) Registry() *taskregistry.Registry {
	return s.registry
}

func (s *TaskService) getRunnableTask(taskName string) (taskregistry.TaskDefinition, error) {
	definition, ok := s.registry.Get(taskName)
	if !ok {
		return taskregistry.TaskDefinition{}, ErrTaskNotFound
	}
	if definition.RunFunc == nil {
		return taskregistry.TaskDefinition{}, ErrTaskNotRunnable
	}
	return definition, nil
}

func (s *TaskService) executeAndLog(ctx context.Context, taskName, trigger string, triggeredBy *uint, runFunc func(context.Context) error) error {
	startedAt := time.Now().UTC()
	execution := &model.TaskExecution{
		TaskName:    taskName,
		Trigger:     trigger,
		TriggeredBy: triggeredBy,
		Status:      taskStatusRunning,
		StartedAt:   startedAt,
	}
	if err := s.repo.CreateExecution(execution); err != nil {
		s.logger().Error("failed to create task execution record", zap.String("task_name", taskName), zap.String("trigger", trigger), zap.Error(err))
	}

	var runErr error
	func() {
		defer func() {
			if recovered := recover(); recovered != nil {
				runErr = fmt.Errorf("task panicked: %v", recovered)
				s.logger().Error("task execution panicked", zap.String("task_name", taskName), zap.String("trigger", trigger), zap.Any("panic", recovered))
			}
		}()
		runErr = runFunc(ctx)
	}()

	finishedAt := time.Now().UTC()
	durationMs := finishedAt.Sub(startedAt).Milliseconds()
	execution.FinishedAt = &finishedAt
	execution.DurationMs = &durationMs
	if runErr != nil {
		execution.Status = taskStatusFailed
		execution.Error = runErr.Error()
	} else {
		execution.Status = taskStatusSuccess
		execution.Error = ""
	}

	if err := s.repo.UpdateExecution(execution); err != nil {
		s.logger().Error("failed to update task execution record", zap.String("task_name", taskName), zap.String("trigger", trigger), zap.Error(err))
	}
	if trigger == taskTriggerManual && triggeredBy != nil && s.auditSvc != nil {
		result := model.AuditResultSuccess
		if runErr != nil {
			result = model.AuditResultFailed
		}
		details := map[string]any{
			"trigger": trigger,
			"status":  execution.Status,
		}
		if runErr != nil {
			details["error"] = runErr.Error()
		}
		_ = s.auditSvc.RecordEvent(ctx, AuditRecordInput{
			Category:     "task_ops",
			Action:       "task_manual_run",
			ActorUserID:  *triggeredBy,
			ResourceType: "task",
			ResourceID:   taskName,
			Result:       result,
			Details:      details,
		})
	}

	return runErr
}

func (s *TaskService) logger() *zap.Logger {
	if global.Logger != nil {
		return global.Logger
	}
	return nopTaskLogger
}

func (s *TaskService) scheduleLock(taskName string) *sync.Mutex {
	if lock, ok := taskScheduleLocks.Load(taskName); ok {
		return lock.(*sync.Mutex)
	}

	lock := &sync.Mutex{}
	actual, _ := taskScheduleLocks.LoadOrStore(taskName, lock)
	return actual.(*sync.Mutex)
}

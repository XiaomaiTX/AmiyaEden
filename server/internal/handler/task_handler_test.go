package handler

import (
	"amiya-eden/internal/model"
	"amiya-eden/internal/repository"
	"amiya-eden/internal/service"
	"amiya-eden/internal/taskregistry"
	"amiya-eden/pkg/response"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type taskHandlerResponse struct {
	Code int             `json:"code"`
	Msg  string          `json:"msg"`
	Data json.RawMessage `json:"data"`
}

type taskHandlerConflictResponse struct {
	Code int    `json:"code"`
	Msg  string `json:"msg"`
}

type taskHistoryPage struct {
	List     []model.TaskExecution `json:"list"`
	Total    int64                 `json:"total"`
	Page     int                   `json:"page"`
	PageSize int                   `json:"pageSize"`
}

func newTaskHandlerTestDeps(t *testing.T) (*taskregistry.Registry, *repository.TaskRepository, *service.TaskService, *TaskHandler) {
	t.Helper()

	dsn := fmt.Sprintf("file:%s_%d?mode=memory&cache=shared", t.Name(), time.Now().UnixNano())
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&model.TaskSchedule{}, &model.TaskExecution{}); err != nil {
		t.Fatalf("auto migrate task models: %v", err)
	}

	registry := taskregistry.New()
	repo := repository.NewTaskRepositoryWithDB(db)
	svc := service.NewTaskService(registry, repo)
	handler := NewTaskHandler(svc)

	t.Cleanup(func() {
		service.RescheduleFn = nil
	})

	return registry, repo, svc, handler
}

func setupTaskHandlerTestRouter(h *TaskHandler, userID uint) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set("userID", userID)
		c.Next()
	})
	g := r.Group("/api/v1")
	g.GET("/tasks", h.GetTasks)
	g.GET("/tasks/history", h.GetHistory)
	g.POST("/tasks/:name/run", h.RunTask)
	g.PUT("/tasks/:name/schedule", h.UpdateSchedule)
	return r
}

func decodeTaskHandlerResponse(t *testing.T, body []byte) taskHandlerResponse {
	t.Helper()

	var result taskHandlerResponse
	if err := json.Unmarshal(body, &result); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	return result
}

func waitForTaskExecution(t *testing.T, repo *repository.TaskRepository, taskName string) *model.TaskExecution {
	t.Helper()

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		execution, err := repo.GetLastExecution(taskName)
		if err != nil {
			t.Fatalf("GetLastExecution returned error: %v", err)
		}
		if execution != nil {
			return execution
		}
		time.Sleep(10 * time.Millisecond)
	}

	t.Fatalf("timed out waiting for task %q execution", taskName)
	return nil
}

func TestTaskHandler_GetTasksEmpty(t *testing.T) {
	_, _, _, h := newTaskHandlerTestDeps(t)
	r := setupTaskHandlerTestRouter(h, 1)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/tasks", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("HTTP status = %d, want %d", w.Code, http.StatusOK)
	}

	result := decodeTaskHandlerResponse(t, w.Body.Bytes())
	if result.Code != response.CodeOK {
		t.Fatalf("response code = %d, want %d", result.Code, response.CodeOK)
	}

	var items []service.TaskItem
	if err := json.Unmarshal(result.Data, &items); err != nil {
		t.Fatalf("decode task list: %v", err)
	}
	if len(items) != 0 {
		t.Fatalf("task count = %d, want 0", len(items))
	}
}

func TestTaskHandler_RunTaskOK(t *testing.T) {
	registry, repo, _, h := newTaskHandlerTestDeps(t)
	registry.Register(taskregistry.TaskDefinition{
		Name:        "test_task",
		Description: "Runs successfully",
		Category:    taskregistry.TaskCategorySystem,
		Type:        taskregistry.TaskTypeRecurring,
		DefaultCron: "0 */5 * * * *",
		RunFunc: func(ctx context.Context) error {
			return nil
		},
	})
	r := setupTaskHandlerTestRouter(h, 42)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/tasks/test_task/run", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("HTTP status = %d, want %d", w.Code, http.StatusOK)
	}

	result := decodeTaskHandlerResponse(t, w.Body.Bytes())
	if result.Code != response.CodeOK {
		t.Fatalf("response code = %d, want %d", result.Code, response.CodeOK)
	}

	execution := waitForTaskExecution(t, repo, "test_task")
	if execution.Status != "success" {
		t.Fatalf("status = %q, want %q", execution.Status, "success")
	}
	if execution.Trigger != "manual" {
		t.Fatalf("trigger = %q, want %q", execution.Trigger, "manual")
	}
	if execution.TriggeredBy == nil || *execution.TriggeredBy != 42 {
		t.Fatalf("triggered_by = %v, want 42", execution.TriggeredBy)
	}
}

func TestTaskHandler_RunTaskNotFound(t *testing.T) {
	_, _, _, h := newTaskHandlerTestDeps(t)
	r := setupTaskHandlerTestRouter(h, 1)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/tasks/missing/run", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("HTTP status = %d, want %d", w.Code, http.StatusOK)
	}

	result := decodeTaskHandlerResponse(t, w.Body.Bytes())
	if result.Code != response.CodeNotFound {
		t.Fatalf("response code = %d, want %d", result.Code, response.CodeNotFound)
	}
}

func TestTaskHandler_RunTaskAlreadyRunningReturns409(t *testing.T) {
	registry, _, _, h := newTaskHandlerTestDeps(t)
	registry.Register(taskregistry.TaskDefinition{
		Name:        "slow_task",
		Description: "Blocks until released",
		Category:    taskregistry.TaskCategorySystem,
		Type:        taskregistry.TaskTypeRecurring,
		DefaultCron: "0 */5 * * * *",
		RunFunc: func(ctx context.Context) error {
			return nil
		},
	})
	r := setupTaskHandlerTestRouter(h, 1)

	handle, ok := registry.TryLock("slow_task")
	if !ok || handle == nil {
		t.Fatal("expected test setup lock acquisition to succeed")
	}
	defer handle.Release()

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/tasks/slow_task/run", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusConflict {
		t.Fatalf("HTTP status = %d, want %d", w.Code, http.StatusConflict)
	}

	var result taskHandlerConflictResponse
	if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil {
		t.Fatalf("decode conflict response: %v", err)
	}
	if result.Code != http.StatusConflict {
		t.Fatalf("response code = %d, want %d", result.Code, http.StatusConflict)
	}
}

func TestTaskHandler_UpdateScheduleOK(t *testing.T) {
	registry, repo, _, h := newTaskHandlerTestDeps(t)
	registry.Register(taskregistry.TaskDefinition{
		Name:        "sched_task",
		Description: "Recurring task",
		Category:    taskregistry.TaskCategorySystem,
		Type:        taskregistry.TaskTypeRecurring,
		DefaultCron: "0 */5 * * * *",
		RunFunc: func(ctx context.Context) error {
			return nil
		},
	})
	service.RescheduleFn = func(taskName, cronExpr string) error {
		return nil
	}
	r := setupTaskHandlerTestRouter(h, 7)

	body, _ := json.Marshal(map[string]string{"cron_expr": "0 */10 * * * *"})
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPut, "/api/v1/tasks/sched_task/schedule", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("HTTP status = %d, want %d", w.Code, http.StatusOK)
	}

	result := decodeTaskHandlerResponse(t, w.Body.Bytes())
	if result.Code != response.CodeOK {
		t.Fatalf("response code = %d, want %d", result.Code, response.CodeOK)
	}

	schedule, err := repo.GetSchedule("sched_task")
	if err != nil {
		t.Fatalf("GetSchedule returned error: %v", err)
	}
	if schedule.CronExpr != "0 */10 * * * *" {
		t.Fatalf("cron_expr = %q, want %q", schedule.CronExpr, "0 */10 * * * *")
	}
	if schedule.UpdatedBy != 7 {
		t.Fatalf("updated_by = %d, want 7", schedule.UpdatedBy)
	}
}

func TestTaskHandler_UpdateScheduleInvalidCron(t *testing.T) {
	registry, _, _, h := newTaskHandlerTestDeps(t)
	registry.Register(taskregistry.TaskDefinition{
		Name:        "sched_task_invalid",
		Description: "Recurring task",
		Category:    taskregistry.TaskCategorySystem,
		Type:        taskregistry.TaskTypeRecurring,
		DefaultCron: "0 */5 * * * *",
		RunFunc: func(ctx context.Context) error {
			return nil
		},
	})
	r := setupTaskHandlerTestRouter(h, 1)

	body, _ := json.Marshal(map[string]string{"cron_expr": "not-a-cron"})
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPut, "/api/v1/tasks/sched_task_invalid/schedule", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("HTTP status = %d, want %d", w.Code, http.StatusOK)
	}

	result := decodeTaskHandlerResponse(t, w.Body.Bytes())
	if result.Code != response.CodeParamError {
		t.Fatalf("response code = %d, want %d", result.Code, response.CodeParamError)
	}
}

func TestTaskHandler_GetHistory(t *testing.T) {
	_, repo, _, h := newTaskHandlerTestDeps(t)
	startedAt := time.Now().UTC()
	finishedAt := startedAt.Add(2 * time.Second)
	durationMs := finishedAt.Sub(startedAt).Milliseconds()
	if err := repo.CreateExecution(&model.TaskExecution{
		TaskName:    "history_task",
		Trigger:     "manual",
		Status:      "success",
		StartedAt:   startedAt,
		FinishedAt:  &finishedAt,
		DurationMs:  &durationMs,
		Summary:     "done",
		TriggeredBy: nil,
	}); err != nil {
		t.Fatalf("CreateExecution returned error: %v", err)
	}
	r := setupTaskHandlerTestRouter(h, 1)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/tasks/history?current=1&size=20", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("HTTP status = %d, want %d", w.Code, http.StatusOK)
	}

	result := decodeTaskHandlerResponse(t, w.Body.Bytes())
	if result.Code != response.CodeOK {
		t.Fatalf("response code = %d, want %d", result.Code, response.CodeOK)
	}

	var page taskHistoryPage
	if err := json.Unmarshal(result.Data, &page); err != nil {
		t.Fatalf("decode history page: %v", err)
	}
	if page.Total != 1 {
		t.Fatalf("total = %d, want 1", page.Total)
	}
	if len(page.List) != 1 {
		t.Fatalf("list length = %d, want 1", len(page.List))
	}
	if page.List[0].TaskName != "history_task" {
		t.Fatalf("task_name = %q, want %q", page.List[0].TaskName, "history_task")
	}
}

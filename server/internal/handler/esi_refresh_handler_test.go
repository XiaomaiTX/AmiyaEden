package handler

import (
	"amiya-eden/global"
	"amiya-eden/internal/middleware"
	"amiya-eden/internal/model"
	"amiya-eden/internal/repository"
	"amiya-eden/jobs"
	"amiya-eden/pkg/background"
	"amiya-eden/pkg/eve/esi"
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
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type esiRefreshHandlerResponse struct {
	Code int             `json:"code"`
	Msg  string          `json:"msg"`
	Data json.RawMessage `json:"data"`
}

var (
	oldGetESIQueue    func() *esi.Queue
	getESIQueueWasSet bool
	originalGetUserID func(c *gin.Context) uint
	originalGetDB     func() *gorm.DB
)

func init() {
	oldGetESIQueue = jobs.GetESIQueue
	originalGetUserID = middleware.GetUserID
	originalGetDB = func() *gorm.DB { return global.DB }
}

func TestRunMyCharacterTask_AllowsUserToRefreshOwnCharacter(t *testing.T) {
	setupTest(t)
	defer teardownTest(t)

	db := newESIRefreshHandlerTestDB(t)
	seedUserWithCharacter(t, db, 1, 9001, "Amiya Main")

	setupGlobalDependencies(t, db)

	queue := setupMockESIQueue(t)
	getESIQueueWasSet = true
	jobs.SetTestESIQueue(queue)
	taskName := registerSuccessfulHandlerQueueTask(t)

	recorder := performRunMyCharacterTaskRequest(t, 1, 9001, taskName)
	if recorder.Code != http.StatusOK {
		t.Fatalf("expected http status 200, got %d", recorder.Code)
	}

	var result esiRefreshHandlerResponse
	if err := json.Unmarshal(recorder.Body.Bytes(), &result); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if result.Code != response.CodeOK {
		t.Fatalf("expected success code, got code=%d, msg=%s", result.Code, result.Msg)
	}
}

func TestRunMyCharacterTask_RejectsOtherUsersCharacter(t *testing.T) {
	setupTest(t)
	defer teardownTest(t)

	db := newESIRefreshHandlerTestDB(t)

	user1ID := uint(1)
	user2ID := uint(2)
	charID := int64(9001)

	seedUserWithCharacter(t, db, user1ID, charID, "User1 Character")
	seedUser(t, db, user2ID, "User2")

	setupGlobalDependencies(t, db)

	recorder := performRunMyCharacterTaskRequest(t, user2ID, charID, "character_skill")
	if recorder.Code != http.StatusForbidden {
		t.Fatalf("expected http status 403, got %d", recorder.Code)
	}

	var result esiRefreshHandlerResponse
	if err := json.Unmarshal(recorder.Body.Bytes(), &result); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if result.Code != response.CodeForbidden {
		t.Fatalf("expected forbidden code, got code=%d, msg=%s", result.Code, result.Msg)
	}
	if result.Msg != "无权操作此角色" {
		t.Fatalf("expected '无权操作此角色', got '%s'", result.Msg)
	}
}

func TestRunMyCharacterTask_RejectsNonExistentCharacter(t *testing.T) {
	setupTest(t)
	defer teardownTest(t)

	db := newESIRefreshHandlerTestDB(t)
	seedUser(t, db, 1, "Amiya")

	setupGlobalDependencies(t, db)

	recorder := performRunMyCharacterTaskRequest(t, 1, 99999, "character_skill")
	if recorder.Code != http.StatusOK {
		t.Fatalf("expected http status 200, got %d", recorder.Code)
	}

	var result esiRefreshHandlerResponse
	if err := json.Unmarshal(recorder.Body.Bytes(), &result); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if result.Code != response.CodeBizError {
		t.Fatalf("expected biz error code, got code=%d, msg=%s", result.Code, result.Msg)
	}
	if result.Msg != "角色不存在" {
		t.Fatalf("expected '角色不存在', got '%s'", result.Msg)
	}
}

func TestRunMyCharacterTask_RejectsInvalidTaskName(t *testing.T) {
	setupTest(t)
	defer teardownTest(t)

	db := newESIRefreshHandlerTestDB(t)
	seedUserWithCharacter(t, db, 1, 9001, "Amiya Main")

	setupGlobalDependencies(t, db)

	queue := setupMockESIQueue(t)
	getESIQueueWasSet = true
	jobs.SetTestESIQueue(queue)

	recorder := performRunMyCharacterTaskRequest(t, 1, 9001, "invalid_task")
	if recorder.Code != http.StatusOK {
		t.Fatalf("expected http status 200, got %d", recorder.Code)
	}

	var result esiRefreshHandlerResponse
	if err := json.Unmarshal(recorder.Body.Bytes(), &result); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if result.Code != response.CodeBizError {
		t.Fatalf("expected biz error code, got code=%d, msg=%s", result.Code, result.Msg)
	}
}

func TestRunMyCharacterTask_RejectsMissingParameters(t *testing.T) {
	setupTest(t)
	defer teardownTest(t)

	db := newESIRefreshHandlerTestDB(t)
	seedUserWithCharacter(t, db, 1, 9001, "Amiya Main")

	setupGlobalDependencies(t, db)

	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)

	reqBody := map[string]interface{}{"character_id": 9001}
	bodyBytes, _ := json.Marshal(reqBody)
	ctx.Request = httptest.NewRequest(http.MethodPost, "/api/v1/info/esi-refresh", bytes.NewBuffer(bodyBytes))
	ctx.Request.Header.Set("Content-Type", "application/json")
	ctx.Set("userID", uint(1))
	ctx.Set("roles", []string{model.RoleUser})

	NewESIRefreshHandler().RunMyCharacterTask(ctx)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected http status 200, got %d", recorder.Code)
	}

	var result esiRefreshHandlerResponse
	if err := json.Unmarshal(recorder.Body.Bytes(), &result); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if result.Code != response.CodeParamError {
		t.Fatalf("expected param error code, got code=%d, msg=%s", result.Code, result.Msg)
	}
}

func TestGetStatuses_FiltersByCharacterIDOrName(t *testing.T) {
	setupTest(t)
	defer teardownTest(t)

	db := newESIRefreshHandlerTestDB(t)
	seedUserWithCharacter(t, db, 1, 9001, "Amiya Main")
	seedUserWithCharacter(t, db, 2, 9002, "Kal'tsit Alt")

	setupGlobalDependencies(t, db)

	queue := setupMockESIQueue(t)
	setQueueStatusesForTest(queue, map[string]*esi.TaskStatus{
		"character_skill:9001": {
			TaskName:    "character_skill",
			Description: "Character Skill",
			CharacterID: 9001,
			Priority:    50,
			Status:      "success",
		},
		"character_wallet:9002": {
			TaskName:    "character_wallet",
			Description: "Character Wallet",
			CharacterID: 9002,
			Priority:    50,
			Status:      "pending",
		},
	})
	getESIQueueWasSet = true
	jobs.SetTestESIQueue(queue)

	recorder := performGetStatusesRequest(t, "/api/v1/tasks/esi/statuses?character=amiya")

	var result esiRefreshHandlerResponse
	if err := json.Unmarshal(recorder.Body.Bytes(), &result); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if result.Code != response.CodeOK {
		t.Fatalf("expected success code, got code=%d, msg=%s", result.Code, result.Msg)
	}

	var page struct {
		List []struct {
			CharacterID   int64  `json:"character_id"`
			CharacterName string `json:"character_name"`
		} `json:"list"`
		Total int64 `json:"total"`
	}
	if err := json.Unmarshal(result.Data, &page); err != nil {
		t.Fatalf("decode page data: %v", err)
	}
	if page.Total != 1 {
		t.Fatalf("expected total=1 for character-name filter, got %d", page.Total)
	}
	if len(page.List) != 1 || page.List[0].CharacterID != 9001 || page.List[0].CharacterName != "Amiya Main" {
		t.Fatalf("unexpected filtered result: %+v", page.List)
	}

	recorder = performGetStatusesRequest(t, "/api/v1/tasks/esi/statuses?character=9002")
	if err := json.Unmarshal(recorder.Body.Bytes(), &result); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if result.Code != response.CodeOK {
		t.Fatalf("expected success code, got code=%d, msg=%s", result.Code, result.Msg)
	}
	if err := json.Unmarshal(result.Data, &page); err != nil {
		t.Fatalf("decode page data: %v", err)
	}
	if page.Total != 1 {
		t.Fatalf("expected total=1 for character-id filter, got %d", page.Total)
	}
	if len(page.List) != 1 || page.List[0].CharacterID != 9002 || page.List[0].CharacterName != "Kal'tsit Alt" {
		t.Fatalf("unexpected filtered result: %+v", page.List)
	}
}

func TestRunTaskByName_RejectsSchedulingWhenBackgroundManagerIsStopping(t *testing.T) {
	setupTest(t)
	defer teardownTest(t)

	oldManager := global.BackgroundTaskManager()
	mgr := background.New(context.Background(), func() *zap.Logger { return zap.NewNop() })
	global.SetBackgroundTaskManager(mgr)
	t.Cleanup(func() {
		global.SetBackgroundTaskManager(oldManager)
	})
	if err := mgr.Shutdown(time.Second); err != nil {
		t.Fatalf("Shutdown returned error: %v", err)
	}

	getESIQueueWasSet = true
	jobs.SetTestESIQueue(setupMockESIQueue(t))

	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	bodyBytes, _ := json.Marshal(map[string]string{"task_name": "character_skill"})
	ctx.Request = httptest.NewRequest(http.MethodPost, "/api/v1/tasks/esi/run-task", bytes.NewBuffer(bodyBytes))
	ctx.Request.Header.Set("Content-Type", "application/json")

	NewESIRefreshHandler().RunTaskByName(ctx)

	var result esiRefreshHandlerResponse
	if err := json.Unmarshal(recorder.Body.Bytes(), &result); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if result.Code != response.CodeBizError {
		t.Fatalf("expected biz error code, got code=%d, msg=%s", result.Code, result.Msg)
	}
	if result.Msg != "服务正在关闭，任务未启动" {
		t.Fatalf("unexpected message: %s", result.Msg)
	}
}

func TestRunAll_RejectsSchedulingWhenBackgroundManagerIsStopping(t *testing.T) {
	setupTest(t)
	defer teardownTest(t)

	oldManager := global.BackgroundTaskManager()
	mgr := background.New(context.Background(), func() *zap.Logger { return zap.NewNop() })
	global.SetBackgroundTaskManager(mgr)
	t.Cleanup(func() {
		global.SetBackgroundTaskManager(oldManager)
	})
	if err := mgr.Shutdown(time.Second); err != nil {
		t.Fatalf("Shutdown returned error: %v", err)
	}

	getESIQueueWasSet = true
	jobs.SetTestESIQueue(setupMockESIQueue(t))

	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodPost, "/api/v1/tasks/esi/run-all", nil)

	NewESIRefreshHandler().RunAll(ctx)

	var result esiRefreshHandlerResponse
	if err := json.Unmarshal(recorder.Body.Bytes(), &result); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if result.Code != response.CodeBizError {
		t.Fatalf("expected biz error code, got code=%d, msg=%s", result.Code, result.Msg)
	}
	if result.Msg != "服务正在关闭，任务未启动" {
		t.Fatalf("unexpected message: %s", result.Msg)
	}
}

func performRunMyCharacterTaskRequest(t *testing.T, userID uint, characterID int64, taskName string) *httptest.ResponseRecorder {
	t.Helper()

	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)

	reqBody := map[string]interface{}{
		"task_name":    taskName,
		"character_id": characterID,
	}
	bodyBytes, _ := json.Marshal(reqBody)
	ctx.Request = httptest.NewRequest(http.MethodPost, "/api/v1/info/esi-refresh", bytes.NewBuffer(bodyBytes))
	ctx.Request.Header.Set("Content-Type", "application/json")
	ctx.Set("userID", userID)
	ctx.Set("roles", []string{model.RoleUser})

	NewESIRefreshHandler().RunMyCharacterTask(ctx)

	return recorder
}

func performGetStatusesRequest(t *testing.T, target string) *httptest.ResponseRecorder {
	t.Helper()

	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodGet, target, nil)

	NewESIRefreshHandler().GetStatuses(ctx)

	return recorder
}

func newESIRefreshHandlerTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	dsn := fmt.Sprintf("file:esi_refresh_handler_test_%d?mode=memory&cache=shared", time.Now().UnixNano())
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&model.User{}, &model.EveCharacter{}); err != nil {
		t.Fatalf("auto migrate: %v", err)
	}
	return db
}

func setupGlobalDependencies(t *testing.T, db *gorm.DB) {
	t.Helper()

	originalDB := global.DB
	originalRedis := global.Redis
	originalLogger := global.CurrentLogger()
	global.DB = db
	global.Redis = redis.NewClient(&redis.Options{Addr: "127.0.0.1:0"})
	global.SetLogger(zap.NewNop())

	t.Cleanup(func() {
		global.DB = originalDB
		if global.Redis != nil {
			_ = global.Redis.Close()
		}
		global.Redis = originalRedis
		global.SetLogger(originalLogger)
	})
}

func seedUser(t *testing.T, db *gorm.DB, userID uint, nickname string) {
	t.Helper()

	user := model.User{
		BaseModel: model.BaseModel{ID: userID},
		Nickname:  nickname,
		QQ:        "12345",
		Role:      model.RoleUser,
	}
	if err := db.Create(&user).Error; err != nil {
		t.Fatalf("create user: %v", err)
	}
}

func seedUserWithCharacter(t *testing.T, db *gorm.DB, userID uint, characterID int64, characterName string) {
	t.Helper()

	seedUser(t, db, userID, characterName)

	character := model.EveCharacter{
		CharacterID:   characterID,
		CharacterName: characterName,
		UserID:        userID,
		TokenInvalid:  false,
	}
	if err := db.Create(&character).Error; err != nil {
		t.Fatalf("create character: %v", err)
	}
}

func setupTest(t *testing.T) {
	t.Helper()
	getESIQueueWasSet = false
}

func teardownTest(t *testing.T) {
	t.Helper()
	if getESIQueueWasSet {
		jobs.SetTestESIQueue(nil)
	}
}

func setupMockESIQueue(t *testing.T) *esi.Queue {
	t.Helper()

	mockSSO := &mockTokenService{}
	mockCharRepo := repository.NewEveCharacterRepository()

	queue := esi.NewQueue(mockSSO, mockCharRepo)
	queue.SetConcurrency(1)
	return queue
}

type successfulHandlerQueueTask struct {
	name string
}

func (t *successfulHandlerQueueTask) Name() string        { return t.name }
func (t *successfulHandlerQueueTask) Description() string { return "successful handler task" }
func (t *successfulHandlerQueueTask) Priority() esi.Priority {
	return esi.PriorityLow
}
func (t *successfulHandlerQueueTask) Interval() esi.RefreshInterval {
	return esi.RefreshInterval{Active: time.Hour, Inactive: 2 * time.Hour}
}
func (t *successfulHandlerQueueTask) RequiredScopes() []esi.TaskScope { return nil }
func (t *successfulHandlerQueueTask) Execute(ctx *esi.TaskContext) error {
	return nil
}

func registerSuccessfulHandlerQueueTask(t *testing.T) string {
	t.Helper()

	taskName := fmt.Sprintf("successful_handler_queue_task_%d", time.Now().UnixNano())
	esi.Register(&successfulHandlerQueueTask{name: taskName})
	return taskName
}

func setQueueStatusesForTest(queue *esi.Queue, statuses map[string]*esi.TaskStatus) {
	queue.SetStatusesForTest(statuses)
}

type mockTokenService struct{}

func (m *mockTokenService) GetValidToken(ctx context.Context, characterID int64) (string, error) {
	return "test_access_token", nil
}

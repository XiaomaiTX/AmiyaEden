package handler

import (
	"amiya-eden/global"
	"amiya-eden/internal/apicontract"
	"amiya-eden/internal/model"
	"amiya-eden/pkg/response"
	"bytes"
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type ticketHandlerResp struct {
	Code int             `json:"code"`
	Msg  string          `json:"msg"`
	Data json.RawMessage `json:"data"`
}

func setupTicketHandlerTestDB(t *testing.T) {
	t.Helper()
	dsn := fmt.Sprintf("file:ticket_handler_test_%d?mode=memory&cache=shared", time.Now().UnixNano())
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&model.User{}, &model.EveCharacter{}, &model.Ticket{}, &model.TicketCategory{}, &model.TicketReply{}, &model.TicketStatusHistory{}); err != nil {
		t.Fatalf("auto migrate: %v", err)
	}
	oldDB := global.DB
	global.DB = db
	t.Cleanup(func() { global.DB = oldDB })
}

func seedTicketHandlerCategory(t *testing.T) model.TicketCategory {
	t.Helper()
	category := model.TicketCategory{Name: "账号问题", NameEN: "Account Issues", Enabled: true}
	if err := global.DB.Create(&category).Error; err != nil {
		t.Fatalf("create category: %v", err)
	}
	return category
}

func decodeTicketHandlerResp(t *testing.T, recorder *httptest.ResponseRecorder) ticketHandlerResp {
	t.Helper()
	var out ticketHandlerResp
	if err := json.Unmarshal(recorder.Body.Bytes(), &out); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	return out
}

func TestTicketHandlerCreateTicketSuccess(t *testing.T) {
	setupTicketHandlerTestDB(t)
	category := seedTicketHandlerCategory(t)
	r := newStrictTicketTestRouter(777)

	body, _ := json.Marshal(map[string]any{
		"category_id": category.ID,
		"title":       "无法登录",
		"description": "登录后立刻掉线",
	})
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/ticket/tickets", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("http status = %d, want %d", rec.Code, http.StatusOK)
	}
	resp := decodeTicketHandlerResp(t, rec)
	if resp.Code != response.CodeOK {
		t.Fatalf("response code = %d, want %d", resp.Code, response.CodeOK)
	}
}

func TestTicketHandlerCreateTicketRejectsInvalidPayload(t *testing.T) {
	setupTicketHandlerTestDB(t)
	r := newStrictTicketTestRouter(777)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/ticket/tickets", bytes.NewReader([]byte(`{"title":"x"}`)))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(rec, req)

	resp := decodeTicketHandlerResp(t, rec)
	if resp.Code != response.CodeParamError {
		t.Fatalf("response code = %d, want %d", resp.Code, response.CodeParamError)
	}
}

func TestTicketHandlerListMyTicketsReturnsPage(t *testing.T) {
	setupTicketHandlerTestDB(t)
	category := seedTicketHandlerCategory(t)

	ticket := model.Ticket{UserID: 8080, CategoryID: category.ID, Title: "页面测试", Description: "分页返回", Status: model.TicketStatusPending}
	if err := global.DB.Create(&ticket).Error; err != nil {
		t.Fatalf("create ticket: %v", err)
	}

	r := newStrictTicketTestRouter(8080)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/ticket/tickets/me?current=1&size=20", nil)
	r.ServeHTTP(rec, req)

	resp := decodeTicketHandlerResp(t, rec)
	if resp.Code != response.CodeOK {
		t.Fatalf("response code = %d, want %d", resp.Code, response.CodeOK)
	}
}

func TestTicketHandlerAdminListTicketsRejectsInvalidUserID(t *testing.T) {
	setupTicketHandlerTestDB(t)
	r := newStrictTicketTestRouter(1)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/system/ticket/tickets?user_id=abc", nil)
	r.ServeHTTP(rec, req)

	resp := decodeTicketHandlerResp(t, rec)
	if resp.Code != response.CodeParamError {
		t.Fatalf("response code = %d, want %d", resp.Code, response.CodeParamError)
	}
}

func TestTicketHandlerAdminListTicketsRejectsOverflowUserID(t *testing.T) {
	setupTicketHandlerTestDB(t)
	r := newStrictTicketTestRouter(1)

	overflow := strconv.FormatUint(uint64(math.MaxUint32)+1, 10)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/system/ticket/tickets?user_id="+overflow, nil)
	r.ServeHTTP(rec, req)

	resp := decodeTicketHandlerResp(t, rec)
	if resp.Code != response.CodeParamError {
		t.Fatalf("response code = %d, want %d", resp.Code, response.CodeParamError)
	}
}

func TestTicketHandlerAdminListTicketsRejectsOverflowCategoryID(t *testing.T) {
	setupTicketHandlerTestDB(t)
	r := newStrictTicketTestRouter(1)

	overflow := strconv.FormatUint(uint64(math.MaxUint32)+1, 10)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/system/ticket/tickets?category_id="+overflow, nil)
	r.ServeHTTP(rec, req)

	resp := decodeTicketHandlerResp(t, rec)
	if resp.Code != response.CodeParamError {
		t.Fatalf("response code = %d, want %d", resp.Code, response.CodeParamError)
	}
}

func TestTicketHandlerAdminListStatusHistoryReturnsOperatorNames(t *testing.T) {
	setupTicketHandlerTestDB(t)
	category := seedTicketHandlerCategory(t)
	operatorID := uint(9001)
	ticket := model.Ticket{UserID: 8080, CategoryID: category.ID, Title: "状态历史", Description: "显示操作人", Status: model.TicketStatusPending}
	if err := global.DB.Create(&ticket).Error; err != nil {
		t.Fatalf("create ticket: %v", err)
	}
	if err := global.DB.Create(&model.User{
		BaseModel:          model.BaseModel{ID: operatorID},
		Nickname:           "Operator Nick",
		PrimaryCharacterID: 991001,
	}).Error; err != nil {
		t.Fatalf("create operator user: %v", err)
	}
	if err := global.DB.Create(&model.EveCharacter{
		UserID:        operatorID,
		CharacterID:   991001,
		CharacterName: "Operator Character",
	}).Error; err != nil {
		t.Fatalf("create operator character: %v", err)
	}
	if err := global.DB.Create(&model.TicketStatusHistory{
		TicketID:   ticket.ID,
		FromStatus: model.TicketStatusPending,
		ToStatus:   model.TicketStatusInProgress,
		ChangedBy:  operatorID,
	}).Error; err != nil {
		t.Fatalf("create history: %v", err)
	}
	r := newStrictTicketTestRouter(operatorID)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/system/ticket/tickets/%d/status-history", ticket.ID), nil)
	r.ServeHTTP(rec, req)
	resp := decodeTicketHandlerResp(t, rec)
	if resp.Code != response.CodeOK {
		t.Fatalf("response code = %d, want %d", resp.Code, response.CodeOK)
	}
	var history []model.TicketStatusHistoryItem
	if err := json.Unmarshal(resp.Data, &history); err != nil {
		t.Fatalf("decode history: %v", err)
	}
	if len(history) != 1 {
		t.Fatalf("history len = %d, want 1", len(history))
	}
	if history[0].ChangedByNickname != "Operator Nick" {
		t.Fatalf("ChangedByNickname = %q, want %q", history[0].ChangedByNickname, "Operator Nick")
	}
	if history[0].ChangedByName != "Operator Nick" {
		t.Fatalf("ChangedByName = %q, want %q", history[0].ChangedByName, "Operator Nick")
	}
	if history[0].ChangedByCharacterName != "Operator Character" {
		t.Fatalf("ChangedByCharacterName = %q, want %q", history[0].ChangedByCharacterName, "Operator Character")
	}
}

// Embed the fallback one level deeper so the ticket methods win method lookup.
type ticketTestStrictFallback struct {
	apicontract.StrictServerInterface
}

func newStrictTicketTestRouter(userID uint) *gin.Engine {
	gin.SetMode(gin.TestMode)
	strict := apicontract.NewStrictHandler(&struct {
		*TicketHandler
		ticketTestStrictFallback
	}{TicketHandler: NewTicketHandler()})
	r := gin.New()
	apicontract.RegisterHandlersWithOptions(r, strict, apicontract.GinServerOptions{
		BaseURL:                  "/api/v1",
		ErrorHandlerForOperation: migratedOperationParamError,
		MiddlewaresForOperation: func(_ string, chain []string) []gin.HandlerFunc {
			if len(chain) == 0 {
				return nil
			}
			return []gin.HandlerFunc{func(c *gin.Context) { c.Set("userID", userID) }}
		},
	})
	return r
}

func TestStrictTicketHandlerPaginationDefaultsAndBounds(t *testing.T) {
	setupTicketHandlerTestDB(t)
	strict := newStrictTicketTestRouter(8080)
	for _, path := range []string{"/api/v1/ticket/tickets/me", "/api/v1/system/ticket/tickets"} {
		for _, tc := range []struct {
			name, query string
			page, size  int
		}{
			{name: "defaults", page: 1, size: 20},
			{name: "nonpositive", query: "?current=-1&size=0", page: 1, size: 20},
			{name: "max size", query: "?current=2&size=100", page: 2, size: 100},
			{name: "oversized falls back", query: "?current=2&size=150", page: 2, size: 20},
		} {
			t.Run(path+"/"+tc.name, func(t *testing.T) {
				rec := httptest.NewRecorder()
				strict.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, path+tc.query, nil))
				result := decodeTicketHandlerResp(t, rec)
				if rec.Code != http.StatusOK || result.Code != response.CodeOK {
					t.Fatalf("unexpected response: %d %s", rec.Code, rec.Body)
				}
				var page struct {
					Page     int `json:"page"`
					PageSize int `json:"pageSize"`
				}
				if err := json.Unmarshal(result.Data, &page); err != nil {
					t.Fatalf("decode page: %v", err)
				}
				if page.Page != tc.page || page.PageSize != tc.size {
					t.Fatalf("page = %d/%d, want %d/%d", page.Page, page.PageSize, tc.page, tc.size)
				}
			})
		}
	}
}

func TestStrictTicketHandlerCreateAndPreserveOwnershipError(t *testing.T) {
	setupTicketHandlerTestDB(t)
	category := seedTicketHandlerCategory(t)
	owner := newStrictTicketTestRouter(777)
	payload := fmt.Sprintf(`{"category_id":%d,"title":"无法登录","description":"登录后立刻掉线"}`, category.ID)
	created := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/ticket/tickets", strings.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	owner.ServeHTTP(created, req)
	result := decodeTicketHandlerResp(t, created)
	var record apicontract.TicketRecord
	if err := json.Unmarshal(result.Data, &record); err != nil {
		t.Fatalf("decode ticket: %v", err)
	}
	if created.Code != http.StatusOK || result.Code != response.CodeOK || record.Id == 0 || record.UserId != 777 || record.CategoryId != category.ID {
		t.Fatalf("unexpected response: %d %s", created.Code, created.Body)
	}

	other := newStrictTicketTestRouter(888)
	denied := httptest.NewRecorder()
	path := fmt.Sprintf("/api/v1/ticket/tickets/%d", record.Id)
	other.ServeHTTP(denied, httptest.NewRequest(http.MethodGet, path, nil))
	result = decodeTicketHandlerResp(t, denied)
	if denied.Code != http.StatusOK || result.Code != response.CodeBizError || string(result.Data) != "null" {
		t.Fatalf("unexpected response: %d %s", denied.Code, denied.Body)
	}
}

func TestStrictTicketHandlerRejectsInvalidAdminFilterBounds(t *testing.T) {
	r := newStrictTicketTestRouter(1)
	for _, query := range []string{"user_id=0", "user_id=4294967296", "category_id=0", "category_id=4294967296"} {
		t.Run(query, func(t *testing.T) {
			got := httptest.NewRecorder()
			r.ServeHTTP(got, httptest.NewRequest(http.MethodGet, "/api/v1/system/ticket/tickets?"+query, nil))
			result := decodeTicketHandlerResp(t, got)
			field, _, _ := strings.Cut(query, "=")
			if got.Code != http.StatusOK || result.Code != response.CodeParamError || result.Msg != "无效的 "+field {
				t.Fatalf("unexpected response: %d %s", got.Code, got.Body)
			}
		})
	}
}

package handler

import (
	"amiya-eden/internal/model"
	"amiya-eden/internal/service"
	"amiya-eden/pkg/response"
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

type stubFuxiHallService struct {
	getPublicPage   func(string) (*service.FuxiHallPublicPageResponse, error)
	getPageConfig   func(string) (*model.FuxiHallPage, error)
	updatePage      func(uint, string, *service.FuxiHallUpdatePageRequest) (*model.FuxiHallPage, error)
	listCards       func(string, bool) ([]model.FuxiHallCard, error)
	listManageCards func(string) ([]service.FuxiHallManageCard, error)
	createCard      func(uint, *service.FuxiHallCreateCardRequest) (*model.FuxiHallCard, error)
	updateCard      func(uint, uint, []string, *service.FuxiHallUpdateCardRequest) (*model.FuxiHallCard, error)
	reorderCards    func(uint, *service.FuxiHallReorderRequest) error
	deleteCard      func(uint, uint) error
}

func (s stubFuxiHallService) GetPublicPage(pageKey string) (*service.FuxiHallPublicPageResponse, error) {
	if s.getPublicPage != nil {
		return s.getPublicPage(pageKey)
	}
	return nil, nil
}

func (s stubFuxiHallService) GetPageConfig(pageKey string) (*model.FuxiHallPage, error) {
	if s.getPageConfig != nil {
		return s.getPageConfig(pageKey)
	}
	return nil, nil
}

func (s stubFuxiHallService) UpdatePageConfig(
	operatorID uint,
	pageKey string,
	req *service.FuxiHallUpdatePageRequest,
) (*model.FuxiHallPage, error) {
	if s.updatePage != nil {
		return s.updatePage(operatorID, pageKey, req)
	}
	return nil, nil
}

func (s stubFuxiHallService) ListCards(pageKey string, visibleOnly bool) ([]model.FuxiHallCard, error) {
	if s.listCards != nil {
		return s.listCards(pageKey, visibleOnly)
	}
	return nil, nil
}

func (s stubFuxiHallService) ListManageCards(pageKey string) ([]service.FuxiHallManageCard, error) {
	if s.listManageCards != nil {
		return s.listManageCards(pageKey)
	}
	return nil, nil
}

func (s stubFuxiHallService) CreateCard(operatorID uint, req *service.FuxiHallCreateCardRequest) (*model.FuxiHallCard, error) {
	if s.createCard != nil {
		return s.createCard(operatorID, req)
	}
	return nil, nil
}

func (s stubFuxiHallService) UpdateCard(operatorID, id uint, operatorRoles []string, req *service.FuxiHallUpdateCardRequest) (*model.FuxiHallCard, error) {
	if s.updateCard != nil {
		return s.updateCard(operatorID, id, operatorRoles, req)
	}
	return nil, nil
}

func (s stubFuxiHallService) ReorderCards(operatorID uint, req *service.FuxiHallReorderRequest) error {
	if s.reorderCards != nil {
		return s.reorderCards(operatorID, req)
	}
	return nil
}

func (s stubFuxiHallService) DeleteCard(operatorID, id uint) error {
	if s.deleteCard != nil {
		return s.deleteCard(operatorID, id)
	}
	return nil
}

func TestFuxiHallHandlerGetLeadership(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodGet, "/api/v1/fuxi-hall/leadership", nil)

	h := &FuxiHallHandler{svc: stubFuxiHallService{
		getPublicPage: func(pageKey string) (*service.FuxiHallPublicPageResponse, error) {
			if pageKey != "leadership" {
				t.Fatalf("unexpected page key: %s", pageKey)
			}
			return &service.FuxiHallPublicPageResponse{
				Page: model.FuxiHallPage{PageKey: "leadership", Title: "管理层"},
			}, nil
		},
	}}

	h.GetLeadership(ctx)
	resp := decodeFuxiHallResponse(t, recorder)
	if resp.Code != response.CodeOK {
		t.Fatalf("response code = %d, want %d", resp.Code, response.CodeOK)
	}
}

func TestFuxiHallHandlerGetPageConfigMasksInternalError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Params = gin.Params{{Key: "page_key", Value: "leadership"}}
	ctx.Request = httptest.NewRequest(http.MethodGet, "/api/v1/system/fuxi-hall/pages/leadership", nil)

	h := &FuxiHallHandler{svc: stubFuxiHallService{
		getPageConfig: func(string) (*model.FuxiHallPage, error) {
			return nil, errors.New("sql: broken pipe")
		},
	}}

	h.GetPageConfig(ctx)
	resp := decodeFuxiHallResponse(t, recorder)
	if resp.Msg != "获取页面配置失败" {
		t.Fatalf("response msg = %q, want fallback message", resp.Msg)
	}
}

func TestFuxiHallHandlerReorderCardsBindsJSON(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(
		http.MethodPut,
		"/api/v1/system/fuxi-hall/cards/reorder",
		bytes.NewBufferString(`{"page_key":"leadership","ordered_ids":[1,2,3]}`),
	)
	ctx.Request.Header.Set("Content-Type", "application/json")

	called := false
	h := &FuxiHallHandler{svc: stubFuxiHallService{
		reorderCards: func(_ uint, req *service.FuxiHallReorderRequest) error {
			called = true
			if req.PageKey != "leadership" || len(req.OrderedIDs) != 3 {
				t.Fatalf("unexpected reorder payload: %+v", req)
			}
			return nil
		},
	}}

	h.ReorderCards(ctx)

	resp := decodeFuxiHallResponse(t, recorder)
	if resp.Code != response.CodeOK {
		t.Fatalf("response code = %d, want %d", resp.Code, response.CodeOK)
	}
	if !called {
		t.Fatal("expected service.ReorderCards to be called")
	}
}

func TestFuxiHallHandlerListCardsUsesManageCardsEndpoint(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Params = gin.Params{{Key: "page_key", Value: "leadership"}}
	ctx.Request = httptest.NewRequest(http.MethodGet, "/api/v1/system/fuxi-hall/cards/leadership", nil)

	called := false
	h := &FuxiHallHandler{svc: stubFuxiHallService{
		listManageCards: func(pageKey string) ([]service.FuxiHallManageCard, error) {
			called = true
			if pageKey != "leadership" {
				t.Fatalf("unexpected page key: %s", pageKey)
			}
			return []service.FuxiHallManageCard{}, nil
		},
	}}

	h.ListCards(ctx)
	resp := decodeFuxiHallResponse(t, recorder)
	if resp.Code != response.CodeOK {
		t.Fatalf("response code = %d, want %d", resp.Code, response.CodeOK)
	}
	if !called {
		t.Fatal("expected ListManageCards to be called")
	}
}

func decodeFuxiHallResponse(t *testing.T, recorder *httptest.ResponseRecorder) response.Response {
	t.Helper()
	var resp response.Response
	if err := json.Unmarshal(recorder.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	return resp
}

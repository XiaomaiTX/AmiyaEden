package handler

import (
	"amiya-eden/internal/apicontract"
	"amiya-eden/internal/model"
	"amiya-eden/internal/service"
	"amiya-eden/pkg/response"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"math"
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

func (s stubFuxiHallService) PublicHall(viewerID uint) (*service.FuxiHallPublicPageResponse, error) {
	if s.getPublicPage != nil {
		return s.getPublicPage("hall-of-fame")
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

type fuxiHallTestFallback struct {
	apicontract.StrictServerInterface
}

func newFuxiHallTestWrapper(h *FuxiHallHandler) *apicontract.ServerInterfaceWrapper {
	adapter := apicontract.NewStrictHandler(struct {
		fuxiHallTestFallback
		*FuxiHallHandler
	}{FuxiHallHandler: h})
	return &apicontract.ServerInterfaceWrapper{Handler: adapter}
}

func TestFuxiHallHandlerGetHall(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodGet, "/api/v1/fuxi-hall/hall-of-fame", nil)

	h := &FuxiHallHandler{svc: stubFuxiHallService{
		getPublicPage: func(pageKey string) (*service.FuxiHallPublicPageResponse, error) {
			if pageKey != "hall-of-fame" {
				t.Fatalf("unexpected page key: %s", pageKey)
			}
			return &service.FuxiHallPublicPageResponse{
				Page: model.FuxiHallPage{PageKey: "hall-of-fame", Title: "管理层"},
			}, nil
		},
	}}

	newFuxiHallTestWrapper(h).GetHall(ctx)
	resp := decodeFuxiHallResponse(t, recorder)
	if resp.Code != response.CodeOK {
		t.Fatalf("response code = %d, want %d", resp.Code, response.CodeOK)
	}
}

func TestFuxiHallHandlerGetPageConfigMasksInternalError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Params = gin.Params{{Key: "page_key", Value: "hall-of-fame"}}
	ctx.Request = httptest.NewRequest(http.MethodGet, "/api/v1/system/fuxi-hall/pages/hall-of-fame", nil)

	h := &FuxiHallHandler{svc: stubFuxiHallService{
		getPageConfig: func(string) (*model.FuxiHallPage, error) {
			return nil, errors.New("sql: broken pipe")
		},
	}}

	newFuxiHallTestWrapper(h).GetHallPageConfig(ctx)
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
		bytes.NewBufferString(`{"page_key":"hall-of-fame","ordered_ids":[1,2,3]}`),
	)
	ctx.Request.Header.Set("Content-Type", "application/json")

	called := false
	h := &FuxiHallHandler{svc: stubFuxiHallService{
		reorderCards: func(_ uint, req *service.FuxiHallReorderRequest) error {
			called = true
			if req.PageKey != "hall-of-fame" || len(req.OrderedIDs) != 3 {
				t.Fatalf("unexpected reorder payload: %+v", req)
			}
			return nil
		},
	}}

	newFuxiHallTestWrapper(h).ReorderCards(ctx)

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
	ctx.Params = gin.Params{{Key: "card_key", Value: "hall-of-fame"}}
	ctx.Request = httptest.NewRequest(http.MethodGet, "/api/v1/system/fuxi-hall/cards/hall-of-fame", nil)

	called := false
	h := &FuxiHallHandler{svc: stubFuxiHallService{
		listManageCards: func(pageKey string) ([]service.FuxiHallManageCard, error) {
			called = true
			if pageKey != "hall-of-fame" {
				t.Fatalf("unexpected page key: %s", pageKey)
			}
			return []service.FuxiHallManageCard{}, nil
		},
	}}

	newFuxiHallTestWrapper(h).ListHallCards(ctx)
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

func (s stubFuxiHallService) UpdateOwnCard(userID, id uint, req *service.FuxiHallOwnCardRequest) (*model.FuxiHallCard, error) {
	return nil, nil
}

func (s stubFuxiHallService) RefreshContributions(ctx context.Context, operatorID uint) error {
	return nil
}

func TestFuxiHallOpenAPICardKeyDependsOnMethod(t *testing.T) {
	ctx, _ := gin.CreateTestContext(httptest.NewRecorder())
	var pageKey string
	var mutatedIDs []uint
	h := &FuxiHallHandler{svc: stubFuxiHallService{
		listManageCards: func(key string) ([]service.FuxiHallManageCard, error) {
			pageKey = key
			return nil, nil
		},
		updateCard: func(_ uint, id uint, _ []string, _ *service.FuxiHallUpdateCardRequest) (*model.FuxiHallCard, error) {
			mutatedIDs = append(mutatedIDs, id)
			return &model.FuxiHallCard{BaseModel: model.BaseModel{ID: id}}, nil
		},
		deleteCard: func(_ uint, id uint) error {
			mutatedIDs = append(mutatedIDs, id)
			return nil
		},
	}}

	list, err := h.ListHallCards(context.Background(), apicontract.ListHallCardsRequestObject{CardKey: "hall-of-fame"})
	if err != nil {
		t.Fatal(err)
	}
	result := recordStrictHallResponse(t, list.VisitListHallCardsResponse)
	if data, ok := result.Data.([]any); pageKey != "hall-of-fame" || !ok || len(data) != 0 {
		t.Fatalf("page key = %q, response = %#v; want hall-of-fame and an empty array", pageKey, result)
	}

	invalid, err := h.UpdateHallCard(ctx, apicontract.UpdateHallCardRequestObject{CardKey: 0})
	if err != nil {
		t.Fatal(err)
	}
	result = recordStrictHallResponse(t, invalid.VisitUpdateHallCardResponse)
	if result.Code != response.CodeParamError || result.Msg != "无效的卡片 ID" || len(mutatedIDs) != 0 {
		t.Fatalf("invalid ID response = %#v, mutations = %v", result, mutatedIDs)
	}

	update, err := h.UpdateHallCard(ctx, apicontract.UpdateHallCardRequestObject{CardKey: math.MaxUint32, Body: &apicontract.HallCardUpdate{}})
	if err != nil {
		t.Fatal(err)
	}
	result = recordStrictHallResponse(t, update.VisitUpdateHallCardResponse)
	if result.Code != response.CodeOK || len(mutatedIDs) != 1 || mutatedIDs[0] != math.MaxUint32 {
		t.Fatalf("update response = %#v, mutations = %v", result, mutatedIDs)
	}

	deleted, err := h.DeleteHallCard(ctx, apicontract.DeleteHallCardRequestObject{CardKey: 1})
	if err != nil {
		t.Fatal(err)
	}
	result = recordStrictHallResponse(t, deleted.VisitDeleteHallCardResponse)
	if result.Code != response.CodeOK || result.Data != nil || len(mutatedIDs) != 2 || mutatedIDs[1] != 1 {
		t.Fatalf("delete response = %#v, mutations = %v", result, mutatedIDs)
	}
}

func TestFuxiHallOpenAPIRejectsMissingPageResult(t *testing.T) {
	h := &FuxiHallHandler{svc: stubFuxiHallService{}}
	out, err := h.GetHallPageConfig(context.Background(), apicontract.GetHallPageConfigRequestObject{PageKey: "hall-of-fame"})
	if err != nil {
		t.Fatal(err)
	}
	result := recordStrictHallResponse(t, out.VisitGetHallPageConfigResponse)
	if result.Code != response.CodeBizError || result.Msg != "获取页面配置失败" || result.Data != nil {
		t.Fatalf("unexpected missing page result: %#v", result)
	}
}

func recordStrictHallResponse(t *testing.T, visit func(http.ResponseWriter) error) response.Response {
	t.Helper()
	recorder := httptest.NewRecorder()
	if err := visit(recorder); err != nil {
		t.Fatal(err)
	}
	if recorder.Code != http.StatusOK {
		t.Fatalf("HTTP status = %d, want 200", recorder.Code)
	}
	return decodeFuxiHallResponse(t, recorder)
}

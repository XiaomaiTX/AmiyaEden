package handler

import (
	"amiya-eden/internal/apicontract"
	"amiya-eden/internal/middleware"
	"amiya-eden/internal/model"
	"amiya-eden/internal/service"
	"amiya-eden/pkg/response"
	"context"
	"github.com/gin-gonic/gin"
	"math"
)

const (
	toolBookmarkReadMessage   = "获取工具书签失败"
	toolBookmarkCreateMessage = "创建工具书签失败"
	toolBookmarkUpdateMessage = "更新工具书签失败"
	toolBookmarkDeleteMessage = "删除工具书签失败"
)

type ToolBookmarkHandler struct {
	svc *service.ToolBookmarkService
}

func NewToolBookmarkHandler() *ToolBookmarkHandler {
	return &ToolBookmarkHandler{svc: service.NewToolBookmarkService()}
}

func (h *ToolBookmarkHandler) createToolBookmark(operatorID uint, req apicontract.ToolBookmarkCreateInput) (apicontract.ToolBookmark, error) {
	row, err := h.svc.AdminCreate(operatorID, service.ToolBookmarkUpsertRequest{
		GroupName:   req.GroupName,
		Name:        req.Name,
		URL:         req.URL,
		Description: req.Description,
		IsEnabled:   req.IsEnabled,
		SortOrder:   req.SortOrder,
	})
	if err != nil {
		return apicontract.ToolBookmark{}, err
	}
	return toolBookmark(*row), nil
}

func (h *ToolBookmarkHandler) updateToolBookmark(operatorID, id uint, req apicontract.ToolBookmarkUpdateInput) (apicontract.ToolBookmark, error) {
	row, err := h.svc.AdminUpdate(operatorID, id, service.ToolBookmarkUpsertRequest{
		GroupName:   req.GroupName,
		Name:        req.Name,
		URL:         req.URL,
		Description: req.Description,
		IsEnabled:   req.IsEnabled,
		SortOrder:   req.SortOrder,
	})
	if err != nil {
		return apicontract.ToolBookmark{}, err
	}
	return toolBookmark(*row), nil
}

func toolBookmarkErrorMessage(err error, fallback string) string {
	if service.IsUserVisibleError(err) {
		return err.Error()
	}
	return fallback
}

func toolBookmarkRows(rows []model.ToolBookmark) []apicontract.ToolBookmark {
	out := make([]apicontract.ToolBookmark, 0, len(rows))
	for _, row := range rows {
		out = append(out, toolBookmark(row))
	}
	return out
}

func toolBookmark(row model.ToolBookmark) apicontract.ToolBookmark {
	return apicontract.ToolBookmark{
		Id: row.ID, CreatedAt: row.CreatedAt, UpdatedAt: row.UpdatedAt,
		GroupName: row.GroupName, Name: row.Name, Url: row.URL, Description: row.Description,
		LogoUrl: row.LogoURL, LogoSource: row.LogoSource, IsEnabled: row.IsEnabled,
		SortOrder: row.SortOrder, CreatedBy: row.CreatedBy,
	}
}

func (h *ToolBookmarkHandler) ListVisible(_ context.Context, _ apicontract.ListVisibleRequestObject) (apicontract.ListVisibleResponseObject, error) {
	var out apicontract.ListVisible200JSONResponse
	rows, err := h.svc.ListVisibleBookmarks()
	if err != nil {
		return &out, out.FromBusinessError(contractBusinessError(response.CodeBizError, toolBookmarkReadMessage))
	}
	return &out, out.FromToolBookmarkListSuccess(apicontract.ToolBookmarkListSuccess{Code: response.CodeOK, Msg: "success", Data: toolBookmarkRows(rows)})
}

func (h *ToolBookmarkHandler) ListToolBookmarks(_ context.Context, _ apicontract.ListToolBookmarksRequestObject) (apicontract.ListToolBookmarksResponseObject, error) {
	var out apicontract.ListToolBookmarks200JSONResponse
	rows, err := h.svc.AdminList()
	if err != nil {
		return &out, out.FromBusinessError(contractBusinessError(response.CodeBizError, toolBookmarkReadMessage))
	}
	return &out, out.FromToolBookmarkListSuccess(apicontract.ToolBookmarkListSuccess{Code: response.CodeOK, Msg: "success", Data: toolBookmarkRows(rows)})
}

func (h *ToolBookmarkHandler) AdminCreate(ctx context.Context, request apicontract.AdminCreateRequestObject) (apicontract.AdminCreateResponseObject, error) {
	c, _ := ctx.(*gin.Context)
	if c == nil {
		return apicontract.AdminCreate401JSONResponse{UnauthorizedJSONResponse: apicontract.UnauthorizedJSONResponse{Code: response.CodeUnauthorized, Msg: "未登录"}}, nil
	}
	var out apicontract.AdminCreate200JSONResponse
	if request.Body == nil {
		return &out, out.FromBusinessError(contractBusinessError(response.CodeParamError, "请求参数错误"))
	}
	data, err := h.createToolBookmark(middleware.GetUserID(c), *request.Body)
	if err != nil {
		return &out, out.FromBusinessError(contractBusinessError(response.CodeBizError, toolBookmarkErrorMessage(err, toolBookmarkCreateMessage)))
	}
	return &out, out.FromToolBookmarkSuccess(apicontract.ToolBookmarkSuccess{Code: response.CodeOK, Msg: "success", Data: data})
}

func (h *ToolBookmarkHandler) UpdateToolBookmark(ctx context.Context, request apicontract.UpdateToolBookmarkRequestObject) (apicontract.UpdateToolBookmarkResponseObject, error) {
	c, _ := ctx.(*gin.Context)
	if c == nil {
		return apicontract.UpdateToolBookmark401JSONResponse{UnauthorizedJSONResponse: apicontract.UnauthorizedJSONResponse{Code: response.CodeUnauthorized, Msg: "未登录"}}, nil
	}
	var out apicontract.UpdateToolBookmark200JSONResponse
	if request.Id == 0 || request.Id > math.MaxUint32 {
		return &out, out.FromBusinessError(contractBusinessError(response.CodeParamError, "无效的书签 ID"))
	}
	if request.Body == nil {
		return &out, out.FromBusinessError(contractBusinessError(response.CodeParamError, "请求参数错误"))
	}
	data, err := h.updateToolBookmark(middleware.GetUserID(c), request.Id, *request.Body)
	if err != nil {
		return &out, out.FromBusinessError(contractBusinessError(response.CodeBizError, toolBookmarkErrorMessage(err, toolBookmarkUpdateMessage)))
	}
	return &out, out.FromToolBookmarkSuccess(apicontract.ToolBookmarkSuccess{Code: response.CodeOK, Msg: "success", Data: data})
}

func (h *ToolBookmarkHandler) DeleteToolBookmark(ctx context.Context, request apicontract.DeleteToolBookmarkRequestObject) (apicontract.DeleteToolBookmarkResponseObject, error) {
	c, _ := ctx.(*gin.Context)
	if c == nil {
		return apicontract.DeleteToolBookmark401JSONResponse{UnauthorizedJSONResponse: apicontract.UnauthorizedJSONResponse{Code: response.CodeUnauthorized, Msg: "未登录"}}, nil
	}
	var out apicontract.DeleteToolBookmark200JSONResponse
	if request.Id == 0 || request.Id > math.MaxUint32 {
		return &out, out.FromBusinessError(contractBusinessError(response.CodeParamError, "无效的书签 ID"))
	}
	if err := h.svc.AdminDelete(middleware.GetUserID(c), request.Id); err != nil {
		return &out, out.FromBusinessError(contractBusinessError(response.CodeBizError, toolBookmarkErrorMessage(err, toolBookmarkDeleteMessage)))
	}
	return &out, out.FromNullSuccess(apicontract.NullSuccess{Code: response.CodeOK, Msg: "success"})
}

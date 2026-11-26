package handler

import (
	"amiya-eden/internal/middleware"
	"amiya-eden/pkg/response"
	"context"
	"github.com/gin-gonic/gin"
	"math"

	"amiya-eden/internal/apicontract"
	"amiya-eden/internal/model"
	"amiya-eden/internal/service"
)

// AutoRoleHandler ESI 自动权限映射管理
type AutoRoleHandler struct {
	svc *service.AutoRoleService
}

func NewAutoRoleHandler() *AutoRoleHandler {
	return &AutoRoleHandler{svc: service.NewAutoRoleService()}
}

func (h *AutoRoleHandler) listEsiRoleMappings() ([]apicontract.AutoRoleEsiRoleMapping, error) {
	mappings, err := h.svc.ListEsiRoleMappings()
	if err != nil {
		return nil, err
	}
	rows := make([]apicontract.AutoRoleEsiRoleMapping, 0, len(mappings))
	for _, mapping := range mappings {
		rows = append(rows, autoRoleEsiRoleMapping(mapping))
	}
	return rows, nil
}

func (h *AutoRoleHandler) createEsiRoleMapping(operatorID uint, req apicontract.AutoRoleEsiRoleMappingInput) (apicontract.AutoRoleEsiRoleMapping, error) {
	mapping, err := h.svc.CreateEsiRoleMappingByOperator(req.EsiRole, req.RoleCode, operatorID)
	if err != nil {
		return apicontract.AutoRoleEsiRoleMapping{}, err
	}
	return autoRoleEsiRoleMapping(*mapping), nil
}

func (h *AutoRoleHandler) listEsiTitleMappings() ([]apicontract.AutoRoleEsiTitleMapping, error) {
	mappings, err := h.svc.ListEsiTitleMappings()
	if err != nil {
		return nil, err
	}
	rows := make([]apicontract.AutoRoleEsiTitleMapping, 0, len(mappings))
	for _, mapping := range mappings {
		rows = append(rows, autoRoleEsiTitleMapping(mapping))
	}
	return rows, nil
}

func (h *AutoRoleHandler) listCorpTitles(ctx context.Context) ([]apicontract.AutoRoleCorpTitle, error) {
	titles, err := h.svc.ListCorpTitles(ctx)
	if err != nil || titles == nil {
		return nil, err
	}
	out := make([]apicontract.AutoRoleCorpTitle, 0, len(titles))
	for _, title := range titles {
		out = append(out, apicontract.AutoRoleCorpTitle{
			CorporationId: title.CorporationID, CorporationName: title.CorporationName,
			TitleId: title.TitleID, TitleName: title.TitleName,
		})
	}
	return out, nil
}

func (h *AutoRoleHandler) createEsiTitleMapping(operatorID uint, req apicontract.AutoRoleEsiTitleMappingInput) (apicontract.AutoRoleEsiTitleMapping, error) {
	mapping, err := h.svc.CreateEsiTitleMappingByOperator(req.CorporationId, req.TitleId, req.TitleName, req.RoleCode, operatorID)
	if err != nil {
		return apicontract.AutoRoleEsiTitleMapping{}, err
	}
	return autoRoleEsiTitleMapping(*mapping), nil
}

func (h *AutoRoleHandler) triggerAutoRoleSync(ctx context.Context) string {
	go h.svc.SyncAllUsersAutoRoles(ctx)
	return "同步任务已触发"
}

func autoRoleEsiRoleMapping(mapping model.EsiRoleMapping) apicontract.AutoRoleEsiRoleMapping {
	return apicontract.AutoRoleEsiRoleMapping{
		Id: mapping.ID, CreatedAt: mapping.CreatedAt, UpdatedAt: mapping.UpdatedAt,
		EsiRole: mapping.EsiRole, RoleCode: mapping.RoleCode, RoleName: mapping.RoleName,
	}
}

func autoRoleEsiTitleMapping(mapping model.EsiTitleMapping) apicontract.AutoRoleEsiTitleMapping {
	return apicontract.AutoRoleEsiTitleMapping{
		Id: mapping.ID, CreatedAt: mapping.CreatedAt, UpdatedAt: mapping.UpdatedAt,
		CorporationId: mapping.CorporationID, TitleId: mapping.TitleID, TitleName: mapping.TitleName,
		RoleCode: mapping.RoleCode, RoleName: mapping.RoleName,
	}
}

func (h *AutoRoleHandler) GetAllEsiRoles(_ context.Context, _ apicontract.GetAllEsiRolesRequestObject) (apicontract.GetAllEsiRolesResponseObject, error) {
	return apicontract.GetAllEsiRoles200JSONResponse{Code: response.CodeOK, Msg: "success", Data: h.svc.GetAllEsiRoles()}, nil
}

func (h *AutoRoleHandler) ListEsiRoleMappings(_ context.Context, _ apicontract.ListEsiRoleMappingsRequestObject) (apicontract.ListEsiRoleMappingsResponseObject, error) {
	var out apicontract.ListEsiRoleMappings200JSONResponse
	data, err := h.listEsiRoleMappings()
	if err != nil {
		return &out, out.FromBusinessError(contractBusinessError(response.CodeBizError, err.Error()))
	}
	return &out, out.FromAutoRoleEsiRoleMappingListSuccess(apicontract.AutoRoleEsiRoleMappingListSuccess{Code: response.CodeOK, Msg: "success", Data: data})
}

func (h *AutoRoleHandler) CreateEsiRoleMapping(ctx context.Context, request apicontract.CreateEsiRoleMappingRequestObject) (apicontract.CreateEsiRoleMappingResponseObject, error) {
	c, _ := ctx.(*gin.Context)
	if c == nil {
		return apicontract.CreateEsiRoleMapping401JSONResponse{UnauthorizedJSONResponse: apicontract.UnauthorizedJSONResponse{Code: response.CodeUnauthorized, Msg: "未登录"}}, nil
	}
	var out apicontract.CreateEsiRoleMapping200JSONResponse
	if request.Body == nil {
		return &out, out.FromBusinessError(contractBusinessError(response.CodeParamError, "请求参数错误"))
	}
	data, err := h.createEsiRoleMapping(middleware.GetUserID(c), *request.Body)
	if err != nil {
		return &out, out.FromBusinessError(contractBusinessError(response.CodeBizError, err.Error()))
	}
	return &out, out.FromAutoRoleEsiRoleMappingSuccess(apicontract.AutoRoleEsiRoleMappingSuccess{Code: response.CodeOK, Msg: "success", Data: data})
}

func (h *AutoRoleHandler) DeleteAutoRoleEsiRoleMapping(ctx context.Context, request apicontract.DeleteAutoRoleEsiRoleMappingRequestObject) (apicontract.DeleteAutoRoleEsiRoleMappingResponseObject, error) {
	c, _ := ctx.(*gin.Context)
	if c == nil {
		return apicontract.DeleteAutoRoleEsiRoleMapping401JSONResponse{UnauthorizedJSONResponse: apicontract.UnauthorizedJSONResponse{Code: response.CodeUnauthorized, Msg: "未登录"}}, nil
	}
	var out apicontract.DeleteAutoRoleEsiRoleMapping200JSONResponse
	if request.Id == 0 || request.Id > math.MaxUint32 {
		return &out, out.FromBusinessError(contractBusinessError(response.CodeParamError, "无效的映射ID"))
	}
	if err := h.svc.DeleteEsiRoleMappingByOperator(request.Id, middleware.GetUserID(c)); err != nil {
		return &out, out.FromBusinessError(contractBusinessError(response.CodeBizError, err.Error()))
	}
	return &out, out.FromNullSuccess(apicontract.NullSuccess{Code: response.CodeOK, Msg: "success"})
}

func (h *AutoRoleHandler) ListCorpTitles(ctx context.Context, _ apicontract.ListCorpTitlesRequestObject) (apicontract.ListCorpTitlesResponseObject, error) {
	if c, ok := ctx.(*gin.Context); ok {
		ctx = c.Request.Context()
	}
	var out apicontract.ListCorpTitles200JSONResponse
	data, err := h.listCorpTitles(ctx)
	if err != nil {
		return &out, out.FromBusinessError(contractBusinessError(response.CodeBizError, err.Error()))
	}
	return &out, out.FromAutoRoleCorpTitleListSuccess(apicontract.AutoRoleCorpTitleListSuccess{Code: response.CodeOK, Msg: "success", Data: &data})
}

func (h *AutoRoleHandler) ListEsiTitleMappings(_ context.Context, _ apicontract.ListEsiTitleMappingsRequestObject) (apicontract.ListEsiTitleMappingsResponseObject, error) {
	var out apicontract.ListEsiTitleMappings200JSONResponse
	data, err := h.listEsiTitleMappings()
	if err != nil {
		return &out, out.FromBusinessError(contractBusinessError(response.CodeBizError, err.Error()))
	}
	return &out, out.FromAutoRoleEsiTitleMappingListSuccess(apicontract.AutoRoleEsiTitleMappingListSuccess{Code: response.CodeOK, Msg: "success", Data: data})
}

func (h *AutoRoleHandler) CreateEsiTitleMapping(ctx context.Context, request apicontract.CreateEsiTitleMappingRequestObject) (apicontract.CreateEsiTitleMappingResponseObject, error) {
	c, _ := ctx.(*gin.Context)
	if c == nil {
		return apicontract.CreateEsiTitleMapping401JSONResponse{UnauthorizedJSONResponse: apicontract.UnauthorizedJSONResponse{Code: response.CodeUnauthorized, Msg: "未登录"}}, nil
	}
	var out apicontract.CreateEsiTitleMapping200JSONResponse
	if request.Body == nil {
		return &out, out.FromBusinessError(contractBusinessError(response.CodeParamError, "请求参数错误"))
	}
	data, err := h.createEsiTitleMapping(middleware.GetUserID(c), *request.Body)
	if err != nil {
		return &out, out.FromBusinessError(contractBusinessError(response.CodeBizError, err.Error()))
	}
	return &out, out.FromAutoRoleEsiTitleMappingSuccess(apicontract.AutoRoleEsiTitleMappingSuccess{Code: response.CodeOK, Msg: "success", Data: data})
}

func (h *AutoRoleHandler) DeleteAutoRoleEsiTitleMapping(ctx context.Context, request apicontract.DeleteAutoRoleEsiTitleMappingRequestObject) (apicontract.DeleteAutoRoleEsiTitleMappingResponseObject, error) {
	c, _ := ctx.(*gin.Context)
	if c == nil {
		return apicontract.DeleteAutoRoleEsiTitleMapping401JSONResponse{UnauthorizedJSONResponse: apicontract.UnauthorizedJSONResponse{Code: response.CodeUnauthorized, Msg: "未登录"}}, nil
	}
	var out apicontract.DeleteAutoRoleEsiTitleMapping200JSONResponse
	if request.Id == 0 || request.Id > math.MaxUint32 {
		return &out, out.FromBusinessError(contractBusinessError(response.CodeParamError, "无效的映射ID"))
	}
	if err := h.svc.DeleteEsiTitleMappingByOperator(request.Id, middleware.GetUserID(c)); err != nil {
		return &out, out.FromBusinessError(contractBusinessError(response.CodeBizError, err.Error()))
	}
	return &out, out.FromNullSuccess(apicontract.NullSuccess{Code: response.CodeOK, Msg: "success"})
}

func (h *AutoRoleHandler) TriggerAutoRoleSync(ctx context.Context, _ apicontract.TriggerAutoRoleSyncRequestObject) (apicontract.TriggerAutoRoleSyncResponseObject, error) {
	if c, ok := ctx.(*gin.Context); ok {
		ctx = c.Request.Context()
	}
	return apicontract.TriggerAutoRoleSync200JSONResponse{Code: response.CodeOK, Msg: "success", Data: h.triggerAutoRoleSync(ctx)}, nil
}

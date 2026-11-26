package handler

import (
	"context"

	"amiya-eden/internal/apicontract"
	"amiya-eden/internal/middleware"
	"amiya-eden/internal/model"
	"amiya-eden/internal/service"
	"amiya-eden/pkg/response"

	"github.com/gin-gonic/gin"
)

type RoleHandler struct {
	svc *service.RoleService
}

func NewRoleHandler() *RoleHandler {
	return &RoleHandler{svc: service.NewRoleService()}
}

func (h *RoleHandler) listRoleDefinitions() []apicontract.SystemRoleDefinition {
	return systemRoleDefinitions(h.svc.ListRoleDefinitions())
}

// ─── 用户职权管理 ───

func (h *RoleHandler) getSystemUserRoles(id uint) ([]apicontract.SystemRoleDefinition, error) {
	if err := validateSystemUserID(id); err != nil {
		return nil, err
	}
	roles, err := h.svc.GetUserRoles(id)
	if err != nil {
		return nil, err
	}
	return systemRoleDefinitions(roles), nil
}

func (h *RoleHandler) setSystemUserRoles(ctx context.Context, userID, operatorID uint, operatorRoles []string, req *apicontract.SystemUserRolesInput) error {
	if err := validateSystemUserID(userID); err != nil {
		return err
	}
	if req == nil {
		return errInvalidSystemUserInput
	}
	return h.svc.SetUserRoles(ctx, operatorID, operatorRoles, userID, req.RoleCodes)
}

func systemRoleDefinitions(definitions []model.RoleDefinition) []apicontract.SystemRoleDefinition {
	rows := make([]apicontract.SystemRoleDefinition, 0, len(definitions))
	for _, role := range definitions {
		rows = append(rows, apicontract.SystemRoleDefinition{
			Code: role.Code, Name: role.Name, I18nKey: role.I18nKey, Description: role.Description, Sort: role.Sort,
		})
	}
	return rows
}

func (h *RoleHandler) ListRoleDefinitions(_ context.Context, _ apicontract.ListRoleDefinitionsRequestObject) (apicontract.ListRoleDefinitionsResponseObject, error) {
	return apicontract.ListRoleDefinitions200JSONResponse{
		Code: response.CodeOK, Msg: "success", Data: h.listRoleDefinitions(),
	}, nil
}

func (h *RoleHandler) GetSystemUserRoles(_ context.Context, request apicontract.GetSystemUserRolesRequestObject) (apicontract.GetSystemUserRolesResponseObject, error) {
	data, err := h.getSystemUserRoles(request.Id)
	var out apicontract.GetSystemUserRoles200JSONResponse
	if err != nil {
		return &out, out.FromBusinessError(systemUserBusinessError(err))
	}
	return &out, out.FromSystemRoleDefinitionListSuccess(apicontract.SystemRoleDefinitionListSuccess{Code: response.CodeOK, Msg: "success", Data: data})
}

func (h *RoleHandler) SetSystemUserRoles(ctx context.Context, request apicontract.SetSystemUserRolesRequestObject) (apicontract.SetSystemUserRolesResponseObject, error) {
	c := ctx.(*gin.Context)
	err := h.setSystemUserRoles(c.Request.Context(), request.Id, middleware.GetUserID(c), middleware.GetUserRoles(c), request.Body)
	var out apicontract.SetSystemUserRoles200JSONResponse
	if err != nil {
		return &out, out.FromBusinessError(systemUserBusinessError(err))
	}
	return &out, out.FromNullSuccess(apicontract.NullSuccess{Code: response.CodeOK, Msg: "success"})
}

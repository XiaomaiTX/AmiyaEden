package handler

import (
	"context"
	"errors"
	"math"

	"amiya-eden/internal/apicontract"
	"amiya-eden/internal/middleware"
	"amiya-eden/internal/model"
	"amiya-eden/internal/service"
	"amiya-eden/internal/utils"
	"amiya-eden/pkg/response"

	"github.com/gin-gonic/gin"
)

var (
	errInvalidSystemUserID    = errors.New("无效的用户ID")
	errInvalidSystemUserInput = errors.New("请求参数错误")
	errSystemUserNotFound     = errors.New("用户不存在")
)

func validateSystemUserID(id uint) error {
	if id == 0 || id > math.MaxUint32 {
		return errInvalidSystemUserID
	}
	return nil
}

func systemUserErrorCode(err error) int {
	switch {
	case errors.Is(err, errInvalidSystemUserID), errors.Is(err, errInvalidSystemUserInput):
		return response.CodeParamError
	case errors.Is(err, errSystemUserNotFound):
		return response.CodeNotFound
	default:
		return response.CodeBizError
	}
}

type UserHandler struct {
	svc *service.UserService
}

func NewUserHandler() *UserHandler {
	return &UserHandler{svc: service.NewUserService()}
}

func (h *UserHandler) listSystemUsers(roles []string, params apicontract.ListSystemUsersParams) (apicontract.SystemUserPage, error) {
	page, pageSize := 1, 20
	if params.Current != nil {
		page = normalizePage(*params.Current)
	}
	if params.Size != nil && *params.Size > 0 {
		pageSize = normalizeLedgerPageSize(*params.Size)
	}
	filter := service.UserFilter{Status: params.Status}
	if params.Keyword != nil {
		filter.Keyword = *params.Keyword
	}
	if params.Role != nil {
		filter.Role = *params.Role
	}

	// admin 只能看到 allow_corporations 下有人物的用户，super_admin 看全部
	if !model.IsSuperAdmin(roles) {
		filter.AllowCorporations = utils.GetAllowCorporations()
	}

	users, total, err := h.svc.ListUsers(page, pageSize, filter)
	if err != nil {
		return apicontract.SystemUserPage{}, err
	}
	rows := make([]apicontract.SystemUserListItem, 0, len(users))
	for _, user := range users {
		rows = append(rows, systemUserListItem(user))
	}
	return apicontract.SystemUserPage{List: rows, Total: total, Page: page, PageSize: pageSize}, nil
}

func (h *UserHandler) getSystemUser(id uint) (apicontract.CurrentUser, error) {
	if err := validateSystemUserID(id); err != nil {
		return apicontract.CurrentUser{}, err
	}
	user, err := h.svc.GetUserByID(id)
	if err != nil {
		return apicontract.CurrentUser{}, errSystemUserNotFound
	}
	return currentUserDTO(user), nil
}

func (h *UserHandler) updateSystemUser(id, operatorID uint, operatorRoles []string, req *apicontract.SystemUserUpdateInput) error {
	if err := validateSystemUserID(id); err != nil {
		return err
	}
	if req == nil {
		return errInvalidSystemUserInput
	}
	return h.svc.UpdateUserByAdmin(id, operatorID, operatorRoles, service.UserPatch{
		Nickname:  req.Nickname,
		QQ:        req.Qq,
		DiscordID: req.DiscordId,
		Status:    req.Status,
	})
}

func (h *UserHandler) deleteSystemUser(id, operatorID uint, operatorRoles []string) error {
	if err := validateSystemUserID(id); err != nil {
		return err
	}
	return h.svc.DeleteUser(id, operatorID, operatorRoles)
}

func (h *UserHandler) impersonateSystemUser(id, operatorID uint) (apicontract.SystemUserImpersonation, error) {
	if err := validateSystemUserID(id); err != nil {
		return apicontract.SystemUserImpersonation{}, err
	}
	token, user, err := h.svc.ImpersonateUser(operatorID, id)
	if err != nil {
		return apicontract.SystemUserImpersonation{}, err
	}
	return apicontract.SystemUserImpersonation{Token: token, User: currentUserDTO(user)}, nil
}

func systemUserListItem(user model.UserListItem) apicontract.SystemUserListItem {
	characters := make([]apicontract.SystemUserListCharacter, 0, len(user.Characters))
	for _, character := range user.Characters {
		characters = append(characters, apicontract.SystemUserListCharacter{
			CharacterId: character.CharacterID, CharacterName: character.CharacterName,
			TotalSp: character.TotalSP, TokenInvalid: character.TokenInvalid,
		})
	}
	return apicontract.SystemUserListItem{
		Id: user.ID, Nickname: user.Nickname, Qq: user.QQ, DiscordId: user.DiscordID,
		PrimaryCharacterId: user.PrimaryCharacterID, Status: user.Status, Roles: user.Roles,
		Characters: characters, LastLoginAt: user.LastLoginAt, LastLoginIp: user.LastLoginIP,
		CreatedAt: user.CreatedAt, UpdatedAt: user.UpdatedAt,
	}
}

func (h *UserHandler) ListSystemUsers(ctx context.Context, request apicontract.ListSystemUsersRequestObject) (apicontract.ListSystemUsersResponseObject, error) {
	data, err := h.listSystemUsers(middleware.GetUserRoles(ctx.(*gin.Context)), request.Params)
	var out apicontract.ListSystemUsers200JSONResponse
	if err != nil {
		return &out, out.FromBusinessError(systemUserBusinessError(err))
	}
	return &out, out.FromSystemUserPageSuccess(apicontract.SystemUserPageSuccess{Code: response.CodeOK, Msg: "success", Data: data})
}

func (h *UserHandler) GetSystemUser(_ context.Context, request apicontract.GetSystemUserRequestObject) (apicontract.GetSystemUserResponseObject, error) {
	data, err := h.getSystemUser(request.Id)
	var out apicontract.GetSystemUser200JSONResponse
	if err != nil {
		return &out, out.FromBusinessError(systemUserBusinessError(err))
	}
	return &out, out.FromSystemUserDetailSuccess(apicontract.SystemUserDetailSuccess{Code: response.CodeOK, Msg: "success", Data: data})
}

func (h *UserHandler) UpdateSystemUser(ctx context.Context, request apicontract.UpdateSystemUserRequestObject) (apicontract.UpdateSystemUserResponseObject, error) {
	c := ctx.(*gin.Context)
	err := h.updateSystemUser(request.Id, middleware.GetUserID(c), middleware.GetUserRoles(c), request.Body)
	var out apicontract.UpdateSystemUser200JSONResponse
	if err != nil {
		if conflict, ok := contactConflictPayload(err); ok {
			return &out, out.FromContactConflict(conflict)
		}
		return &out, out.FromBusinessError(systemUserBusinessError(err))
	}
	return &out, out.FromNullSuccess(apicontract.NullSuccess{Code: response.CodeOK, Msg: "success"})
}

func (h *UserHandler) DeleteSystemUser(ctx context.Context, request apicontract.DeleteSystemUserRequestObject) (apicontract.DeleteSystemUserResponseObject, error) {
	c := ctx.(*gin.Context)
	err := h.deleteSystemUser(request.Id, middleware.GetUserID(c), middleware.GetUserRoles(c))
	var out apicontract.DeleteSystemUser200JSONResponse
	if err != nil {
		return &out, out.FromBusinessError(systemUserBusinessError(err))
	}
	return &out, out.FromNullSuccess(apicontract.NullSuccess{Code: response.CodeOK, Msg: "success"})
}

func (h *UserHandler) ImpersonateSystemUser(ctx context.Context, request apicontract.ImpersonateSystemUserRequestObject) (apicontract.ImpersonateSystemUserResponseObject, error) {
	data, err := h.impersonateSystemUser(request.Id, middleware.GetUserID(ctx.(*gin.Context)))
	var out apicontract.ImpersonateSystemUser200JSONResponse
	if err != nil {
		return &out, out.FromBusinessError(systemUserBusinessError(err))
	}
	return &out, out.FromSystemUserImpersonationSuccess(apicontract.SystemUserImpersonationSuccess{Code: response.CodeOK, Msg: "success", Data: data})
}

func systemUserBusinessError(err error) apicontract.BusinessError {
	return contractBusinessError(systemUserErrorCode(err), err.Error())
}

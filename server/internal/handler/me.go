package handler

import (
	"amiya-eden/internal/apicontract"
	"amiya-eden/internal/middleware"
	"amiya-eden/internal/model"
	"amiya-eden/internal/service"
	"amiya-eden/pkg/response"
	"context"
	"errors"

	"github.com/gin-gonic/gin"
)

type MeHandler struct {
	userSvc              *service.UserService
	roleSvc              *service.RoleService
	cfgSvc               *service.SysConfigService
	mentorEligibilitySvc *service.MentorEligibilityService
}

func NewMeHandler() *MeHandler {
	return &MeHandler{
		userSvc:              service.NewUserService(),
		roleSvc:              service.NewRoleService(),
		cfgSvc:               service.NewSysConfigService(),
		mentorEligibilitySvc: service.NewMentorEligibilityService(),
	}
}

type meAuthContext struct {
	userID     uint
	user       *model.User
	roles      []string
	corpPolicy service.UserCorpPolicyContext
}

func meAuthFromGin(c *gin.Context) meAuthContext {
	return meAuthContext{
		userID: middleware.GetUserID(c),
		user:   middleware.GetAuthUser(c),
		roles:  middleware.GetUserRoles(c),
		corpPolicy: service.UserCorpPolicyContext{
			PrimaryCorporationID: middleware.GetPrimaryCorporationID(c),
			Capabilities:         middleware.GetCorpCapabilities(c),
			Rules:                middleware.GetCorpRules(c),
			FullAccess:           middleware.IsCorpFullAccess(c),
		},
	}
}

func (h *MeHandler) getMe(auth meAuthContext) (apicontract.Me, *contractOperationFailure) {
	user := auth.user
	if user == nil || user.ID != auth.userID {
		var err error
		user, err = h.userSvc.GetUserByID(auth.userID)
		if err != nil {
			return apicontract.Me{}, &contractOperationFailure{response.CodeUnauthorized, "用户不存在"}
		}
	}

	characters, err := h.userSvc.ListUserCharacters(auth.userID)
	if err != nil {
		return apicontract.Me{}, &contractOperationFailure{response.CodeBizError, "加载人物信息失败"}
	}
	if characters == nil {
		characters = []model.EveCharacter{}
	}

	roles := auth.roles
	if roles == nil {
		roles = []string{}
	}
	var isCurrentlyNewbro *bool
	var isMentorMenteeEligible *bool
	if result, err := h.mentorEligibilitySvc.EvaluateEligibilityWithCharacters(characters); err == nil && result != nil {
		value := result.IsEligible
		isCurrentlyNewbro = &value
		isMentorMenteeEligible = &value
	}
	enforceCharacterESIRestriction := h.cfgSvc.GetCharacterESIRestrictionConfig()

	corpCtx := auth.corpPolicy

	characterDTOs := make([]apicontract.EveCharacter, len(characters))
	for i, character := range characters {
		characterDTOs[i] = eveCharacterDTO(character)
	}
	return apicontract.Me{
		User: currentUserDTO(user), Characters: characterDTOs, Roles: roles,
		ProfileComplete: user.ProfileComplete(), PrimaryCorporationId: corpCtx.PrimaryCorporationID,
		CorpCapabilities: service.EffectiveCapabilities(corpCtx), CorpRules: corpCtx.Rules,
		EnforceCharacterESIRestriction: enforceCharacterESIRestriction,
		IsCurrentlyNewbro:              isCurrentlyNewbro, IsMentorMenteeEligible: isMentorMenteeEligible,
	}, nil
}

func currentUserDTO(user *model.User) apicontract.CurrentUser {
	return apicontract.CurrentUser{
		Id: user.ID, CreatedAt: user.CreatedAt, UpdatedAt: user.UpdatedAt,
		Nickname: user.Nickname, Qq: user.QQ, DiscordId: user.DiscordID, Status: user.Status,
		Role: user.Role, PrimaryCharacterId: user.PrimaryCharacterID,
		LastLoginAt: user.LastLoginAt, LastLoginIp: user.LastLoginIP,
	}
}

func (h *MeHandler) updateMe(userID uint, req apicontract.UpdateMeJSONRequestBody) (apicontract.MeUpdate, error) {
	user, err := h.userSvc.UpdateCurrentProfile(userID, service.UserPatch{
		Nickname:  req.Nickname,
		QQ:        req.Qq,
		DiscordID: req.DiscordId,
	})
	if err != nil {
		return apicontract.MeUpdate{}, err
	}
	return apicontract.MeUpdate{User: currentUserDTO(user), ProfileComplete: user.ProfileComplete()}, nil
}

func contactConflictPayload(err error) (apicontract.ContactConflict, bool) {
	var conflict *service.ContactConflictError
	if !errors.As(err, &conflict) {
		return apicontract.ContactConflict{}, false
	}
	return apicontract.ContactConflict{
		Code: 409,
		Msg:  apicontract.ContactAlreadyBound,
		Data: apicontract.ContactConflictDetail{
			Contact:            apicontract.ContactConflictDetailContact(conflict.Contact),
			PrimaryCharacterId: conflict.PrimaryCharacterID, PrimaryCharacterName: conflict.PrimaryCharacterName,
		},
	}, true
}

func (h *MeHandler) deleteMe(userID uint) *contractOperationFailure {
	if userID == 0 {
		return &contractOperationFailure{response.CodeUnauthorized, "未登录"}
	}
	if err := h.userSvc.DeleteCurrentUser(userID); err != nil {
		return &contractOperationFailure{response.CodeBizError, err.Error()}
	}
	return nil
}

func meUnauthorized(message string) apicontract.UnauthorizedJSONResponse {
	return apicontract.UnauthorizedJSONResponse{Code: response.CodeUnauthorized, Msg: message}
}

func (h *MeHandler) GetMe(ctx context.Context, _ apicontract.GetMeRequestObject) (apicontract.GetMeResponseObject, error) {
	c, _ := ctx.(*gin.Context)
	if c == nil {
		return apicontract.GetMe401JSONResponse{UnauthorizedJSONResponse: meUnauthorized("未登录")}, nil
	}
	data, failure := h.getMe(meAuthFromGin(c))
	out := &apicontract.GetMe200JSONResponse{}
	if failure != nil {
		if failure.code == response.CodeUnauthorized {
			return apicontract.GetMe401JSONResponse{UnauthorizedJSONResponse: meUnauthorized(failure.message)}, nil
		}
		return out, out.FromBusinessError(contractBusinessError(failure.code, failure.message))
	}
	return out, out.FromMeSuccess(apicontract.MeSuccess{Code: response.CodeOK, Msg: "success", Data: data})
}

func (h *MeHandler) UpdateMe(ctx context.Context, request apicontract.UpdateMeRequestObject) (apicontract.UpdateMeResponseObject, error) {
	c, _ := ctx.(*gin.Context)
	if c == nil {
		return apicontract.UpdateMe401JSONResponse{UnauthorizedJSONResponse: meUnauthorized("未登录")}, nil
	}
	out := &apicontract.UpdateMe200JSONResponse{}
	if request.Body == nil {
		return out, out.FromBusinessError(contractBusinessError(response.CodeParamError, "请求参数错误"))
	}
	data, err := h.updateMe(middleware.GetUserID(c), *request.Body)
	if err != nil {
		if conflict, ok := contactConflictPayload(err); ok {
			return out, out.FromContactConflict(conflict)
		}
		return out, out.FromBusinessError(contractBusinessError(response.CodeBizError, err.Error()))
	}
	return out, out.FromMeUpdateSuccess(apicontract.MeUpdateSuccess{Code: response.CodeOK, Msg: "success", Data: data})
}

func (h *MeHandler) DeleteMe(ctx context.Context, _ apicontract.DeleteMeRequestObject) (apicontract.DeleteMeResponseObject, error) {
	c, _ := ctx.(*gin.Context)
	if c == nil {
		return apicontract.DeleteMe401JSONResponse{UnauthorizedJSONResponse: meUnauthorized("未登录")}, nil
	}
	out := &apicontract.DeleteMe200JSONResponse{}
	if failure := h.deleteMe(middleware.GetUserID(c)); failure != nil {
		if failure.code == response.CodeUnauthorized {
			return apicontract.DeleteMe401JSONResponse{UnauthorizedJSONResponse: meUnauthorized(failure.message)}, nil
		}
		return out, out.FromBusinessError(contractBusinessError(failure.code, failure.message))
	}
	return out, out.FromNullSuccess(apicontract.NullSuccess{Code: response.CodeOK, Msg: "success"})
}

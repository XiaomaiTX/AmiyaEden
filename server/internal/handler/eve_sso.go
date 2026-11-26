package handler

import (
	"context"
	"net/http"
	"net/url"

	"amiya-eden/internal/apicontract"
	"amiya-eden/internal/middleware"
	"amiya-eden/internal/model"
	"amiya-eden/internal/service"
	"amiya-eden/pkg/response"

	"github.com/gin-gonic/gin"
)

// EveSSOHandler EVE SSO 登录处理器
type EveSSOHandler struct {
	svc      *service.EveSSOService
	auditSvc *service.AuditService
}

func NewEveSSOHandler() *EveSSOHandler {
	return &EveSSOHandler{
		svc:      service.NewEveSSOService(),
		auditSvc: service.NewAuditService(),
	}
}

func (h *EveSSOHandler) eveSsoAuthorization(request ssoRequestContext, redirectURL, scopesParam string, bind bool) (apicontract.EveSsoAuthorization, *contractOperationFailure) {
	action := "eve_sso_login"
	var scopeRoles []string
	if bind {
		if request.userID == 0 {
			return apicontract.EveSsoAuthorization{}, &contractOperationFailure{response.CodeUnauthorized, "未登录"}
		}
		action = "eve_sso_bind"
		scopeRoles = request.roles
	}

	extraScopes := splitCSV(scopesParam)
	if err := service.ValidateExtraScopes(extraScopes, scopeRoles); err != nil {
		h.recordSecurityEvent(request, action+"_scope_rejected", model.AuditResultFailed, map[string]any{
			"reason":       err.Error(),
			"scopes_param": scopesParam,
		})
		return apicontract.EveSsoAuthorization{}, &contractOperationFailure{response.CodeForbidden, err.Error()}
	}

	var authURL string
	var err error
	if bind {
		authURL, err = h.svc.GetBindAuthURL(request.ctx, request.userID, extraScopes, redirectURL)
	} else {
		authURL, err = h.svc.GetAuthURL(request.ctx, extraScopes, redirectURL)
	}
	if err != nil {
		h.recordSecurityEvent(request, action+"_url_failed", model.AuditResultFailed, map[string]any{
			"error":        err.Error(),
			"scopes_param": scopesParam,
		})
		return apicontract.EveSsoAuthorization{}, &contractOperationFailure{response.CodeBizError, "生成授权 URL 失败: " + err.Error()}
	}
	details := map[string]any{"scopes_param": scopesParam}
	if bind {
		details["user_id"] = request.userID
	}
	h.recordSecurityEvent(request, action+"_start", model.AuditResultSuccess, details)
	return apicontract.EveSsoAuthorization{Url: authURL}, nil
}

type eveSsoCallbackOutcome struct {
	redirect     string
	failure      *contractOperationFailure
	registration *apicontract.EveSsoRegistrationTicket
	login        apicontract.EveSsoCallbackData
}

func (h *EveSSOHandler) eveSsoCallback(request ssoRequestContext, params apicontract.EveSsoCallbackParams) eveSsoCallbackOutcome {
	code := ssoQueryValue(params.Code)
	state := ssoQueryValue(params.State)
	errParam := ssoQueryValue(params.Error)

	// Read the redirect before HandleCallback consumes the state.
	frontendRedirect := h.svc.GetRedirectURLFromState(request.ctx, state)
	callbackFailure := func(message string) eveSsoCallbackOutcome {
		if frontendRedirect != "" {
			return eveSsoCallbackOutcome{redirect: frontendRedirect + "?error=" + url.QueryEscape(message)}
		}
		return eveSsoCallbackOutcome{failure: &contractOperationFailure{response.CodeBizError, message}}
	}

	if errParam != "" {
		errDesc := errParam
		if params.ErrorDescription != nil {
			errDesc = *params.ErrorDescription
		}
		h.recordSecurityEvent(request, "eve_sso_callback_denied", model.AuditResultFailed, map[string]any{
			"error":             errParam,
			"error_description": errDesc,
		})
		return callbackFailure("EVE SSO 授权被拒绝: " + errDesc)
	}

	result, err := h.svc.HandleCallback(request.ctx, code, state, request.clientIP)
	if err != nil {
		h.recordSecurityEvent(request, "eve_sso_callback_failed", model.AuditResultFailed, map[string]any{"error": err.Error()})
		return callbackFailure("登录处理失败: " + err.Error())
	}

	// Missing optional scopes may require another authorization pass instead of issuing a JWT now.
	if result.ReauthURL != "" {
		h.recordSecurityEvent(request, "eve_sso_scope_reauth", model.AuditResultSuccess, map[string]any{})
		return eveSsoCallbackOutcome{redirect: result.ReauthURL}
	}

	if result.RegistrationTicket != "" {
		if result.RedirectURL != "" {
			return eveSsoCallbackOutcome{redirect: result.RedirectURL + "?registration=" + url.QueryEscape(result.RegistrationTicket)}
		}
		return eveSsoCallbackOutcome{registration: &apicontract.EveSsoRegistrationTicket{Registration: result.RegistrationTicket}}
	}
	h.recordSecurityEvent(request, "eve_sso_callback_success", model.AuditResultSuccess, map[string]any{
		"user_id":      result.User.ID,
		"character_id": result.Character.CharacterID,
	})

	if result.RedirectURL != "" {
		return eveSsoCallbackOutcome{redirect: result.RedirectURL + "?token=" + result.Token}
	}
	return eveSsoCallbackOutcome{login: apicontract.EveSsoCallbackData{
		Token: result.Token, User: currentUserDTO(result.User), Character: eveCharacterDTO(*result.Character),
	}}
}

func listEveSsoScopes() []apicontract.EveSsoScope {
	scopes := service.GetRegisteredScopes()
	dto := make([]apicontract.EveSsoScope, len(scopes))
	for i, scope := range scopes {
		dto[i] = apicontract.EveSsoScope{Module: scope.Module, Scope: scope.Scope, Description: scope.Description, Required: scope.Required}
	}
	return dto
}

func (h *EveSSOHandler) listMyEveCharacters(userID uint) ([]apicontract.EveCharacter, *contractOperationFailure) {
	if userID == 0 {
		return nil, &contractOperationFailure{response.CodeUnauthorized, "未登录"}
	}
	chars, err := h.svc.GetCharactersByUserID(userID)
	if err != nil {
		return nil, &contractOperationFailure{response.CodeBizError, err.Error()}
	}
	var dto []apicontract.EveCharacter
	if chars != nil {
		dto = make([]apicontract.EveCharacter, len(chars))
		for i, char := range chars {
			dto[i] = eveCharacterDTO(char)
		}
	}
	return dto, nil
}

func changeEveSsoCharacter(userID uint, characterID int64, change func(uint, int64) error) *contractOperationFailure {
	if userID == 0 {
		return &contractOperationFailure{response.CodeUnauthorized, "未登录"}
	}
	if characterID <= 0 {
		return &contractOperationFailure{response.CodeParamError, "无效的人物 ID"}
	}
	if err := change(userID, characterID); err != nil {
		return &contractOperationFailure{response.CodeBizError, err.Error()}
	}
	return nil
}

func ssoQueryValue(value *string) string {
	if value != nil {
		return *value
	}
	return ""
}

func eveCharacterDTO(character model.EveCharacter) apicontract.EveCharacter {
	return apicontract.EveCharacter{
		Id: character.ID, CreatedAt: character.CreatedAt, UpdatedAt: character.UpdatedAt,
		CharacterId: character.CharacterID, CharacterName: character.CharacterName, UserId: character.UserID,
		Scopes: character.Scopes, TokenExpiry: character.TokenExpiry, TokenInvalid: character.TokenInvalid,
		Birthday: character.Birthday, FuxiLegionTenureDays: character.FuxiLegionTenureDays,
		CorporationId: character.CorporationID, AllianceId: character.AllianceID, FactionId: character.FactionID,
	}
}

// splitCSV 按逗号或空格分割字符串
func splitCSV(s string) []string {
	var result []string
	for _, part := range splitAny(s, ",; ") {
		if part != "" {
			result = append(result, part)
		}
	}
	return result
}

func splitAny(s string, seps string) []string {
	splitter := func(r rune) bool {
		for _, sep := range seps {
			if r == sep {
				return true
			}
		}
		return false
	}
	result := []string{}
	start := 0
	for i, r := range s {
		if splitter(r) {
			if i > start {
				result = append(result, s[start:i])
			}
			start = i + 1
		}
	}
	if start < len(s) {
		result = append(result, s[start:])
	}
	return result
}

type ssoRequestContext struct {
	ctx        context.Context
	userID     uint
	roles      []string
	resourceID string
	requestID  string
	clientIP   string
	userAgent  string
}

func ssoRequestFromGin(c *gin.Context) ssoRequestContext {
	return ssoRequestContext{
		ctx:        c.Request.Context(),
		userID:     middleware.GetUserID(c),
		roles:      middleware.GetUserRoles(c),
		resourceID: c.FullPath(),
		requestID:  c.GetString("request_id"),
		clientIP:   c.ClientIP(),
		userAgent:  c.Request.UserAgent(),
	}
}

func (h *EveSSOHandler) recordSecurityEvent(request ssoRequestContext, action string, result string, details map[string]any) {
	if h.auditSvc == nil {
		return
	}
	_ = h.auditSvc.RecordEvent(request.ctx, service.AuditRecordInput{
		Category:     "security",
		Action:       action,
		ActorUserID:  request.userID,
		ResourceType: "eve_sso",
		ResourceID:   request.resourceID,
		Result:       result,
		RequestID:    request.requestID,
		IP:           request.clientIP,
		UserAgent:    request.userAgent,
		Details:      details,
	})
}

func ssoAuthError(failure *contractOperationFailure) apicontract.AuthError {
	return apicontract.AuthError{Code: apicontract.AuthErrorCode(failure.code), Msg: failure.message}
}

func (h *EveSSOHandler) GetEveSsoLogin(ctx context.Context, request apicontract.GetEveSsoLoginRequestObject) (apicontract.GetEveSsoLoginResponseObject, error) {
	c := ctx.(*gin.Context)
	data, failure := h.eveSsoAuthorization(ssoRequestFromGin(c), ssoQueryValue(request.Params.Redirect), ssoQueryValue(request.Params.Scopes), false)
	out := &apicontract.GetEveSsoLogin200JSONResponse{}
	if failure != nil {
		if failure.code == response.CodeForbidden {
			return apicontract.GetEveSsoLogin403JSONResponse{ForbiddenJSONResponse: apicontract.ForbiddenJSONResponse(ssoAuthError(failure))}, nil
		}
		return out, out.FromBusinessError(contractBusinessError(failure.code, failure.message))
	}
	return out, out.FromEveSsoAuthorizationSuccess(apicontract.EveSsoAuthorizationSuccess{Code: response.CodeOK, Msg: "success", Data: data})
}

func (h *EveSSOHandler) EveSsoCallback(ctx context.Context, request apicontract.EveSsoCallbackRequestObject) (apicontract.EveSsoCallbackResponseObject, error) {
	c := ctx.(*gin.Context)
	result := h.eveSsoCallback(ssoRequestFromGin(c), request.Params)
	if result.redirect != "" {
		return eveSsoRedirectResponse{
			request: c.Request,
			headers: apicontract.EveSsoCallback302ResponseHeaders{
				Location: result.redirect, CacheControl: "no-store", ReferrerPolicy: "no-referrer",
			},
		}, nil
	}
	out := &apicontract.EveSsoCallback200JSONResponse{
		Headers: apicontract.EveSsoCallback200ResponseHeaders{CacheControl: "no-store", ReferrerPolicy: "no-referrer"},
	}
	switch {
	case result.failure != nil:
		return out, out.FromBusinessError(contractBusinessError(result.failure.code, result.failure.message))
	case result.registration != nil:
		return out, out.FromEveSsoRegistrationTicketSuccess(apicontract.EveSsoRegistrationTicketSuccess{Code: response.CodeOK, Msg: "success", Data: *result.registration})
	default:
		return out, out.FromEveSsoCallbackSuccess(apicontract.EveSsoCallbackSuccess{Code: response.CodeOK, Msg: "success", Data: result.login})
	}
}

// Use the same redirect writer as the old Gin handler so relative paths, non-ASCII
// Location values, and the small HTML response retain net/http semantics.
type eveSsoRedirectResponse struct {
	request *http.Request
	headers apicontract.EveSsoCallback302ResponseHeaders
}

func (r eveSsoRedirectResponse) VisitEveSsoCallbackResponse(w http.ResponseWriter) error {
	w.Header().Set("Cache-Control", r.headers.CacheControl)
	w.Header().Set("Referrer-Policy", r.headers.ReferrerPolicy)
	http.Redirect(w, r.request, r.headers.Location, http.StatusFound)
	return nil
}

func (h *EveSSOHandler) ListEveSsoScopes(_ context.Context, _ apicontract.ListEveSsoScopesRequestObject) (apicontract.ListEveSsoScopesResponseObject, error) {
	return apicontract.ListEveSsoScopes200JSONResponse{Code: response.CodeOK, Msg: "success", Data: listEveSsoScopes()}, nil
}

func (h *EveSSOHandler) GetEveRegistration(ctx context.Context, request apicontract.GetEveRegistrationRequestObject) (apicontract.GetEveRegistrationResponseObject, error) {
	c := ctx.(*gin.Context)
	out := &apicontract.GetEveRegistration200JSONResponse{
		Headers: apicontract.GetEveRegistration200ResponseHeaders{CacheControl: "no-store"},
	}
	data, err := h.getEveRegistration(c.Request.Context(), request.Params.XRegistrationToken)
	if err != nil {
		return out, out.FromBusinessError(contractBusinessError(response.CodeParamError, err.Error()))
	}
	return out, out.FromEveRegistrationInfoSuccess(apicontract.EveRegistrationInfoSuccess{Code: response.CodeOK, Msg: "success", Data: data})
}

func (h *EveSSOHandler) CompleteEveRegistration(ctx context.Context, request apicontract.CompleteEveRegistrationRequestObject) (apicontract.CompleteEveRegistrationResponseObject, error) {
	c := ctx.(*gin.Context)
	out := &apicontract.CompleteEveRegistration200JSONResponse{
		Headers: apicontract.CompleteEveRegistration200ResponseHeaders{CacheControl: "no-store"},
	}
	if request.Body == nil {
		return out, out.FromBusinessError(contractBusinessError(response.CodeParamError, "请求参数错误"))
	}
	data, err := h.completeEveRegistration(ssoRequestFromGin(c), request.Params.XRegistrationToken, *request.Body)
	if err != nil {
		if conflict, ok := contactConflictPayload(err); ok {
			return out, out.FromContactConflict(conflict)
		}
		return out, out.FromBusinessError(contractBusinessError(response.CodeParamError, err.Error()))
	}
	return out, out.FromEveRegistrationSuccess(apicontract.EveRegistrationSuccess{Code: response.CodeOK, Msg: "success", Data: data})
}

func (h *EveSSOHandler) ListMyEveCharacters(ctx context.Context, _ apicontract.ListMyEveCharactersRequestObject) (apicontract.ListMyEveCharactersResponseObject, error) {
	c := ctx.(*gin.Context)
	data, failure := h.listMyEveCharacters(middleware.GetUserID(c))
	out := &apicontract.ListMyEveCharacters200JSONResponse{}
	if failure != nil {
		if failure.code == response.CodeUnauthorized {
			return apicontract.ListMyEveCharacters401JSONResponse{UnauthorizedJSONResponse: apicontract.UnauthorizedJSONResponse(ssoAuthError(failure))}, nil
		}
		return out, out.FromBusinessError(contractBusinessError(failure.code, failure.message))
	}
	return out, out.FromEveSsoCharactersSuccess(apicontract.EveSsoCharactersSuccess{Code: response.CodeOK, Msg: "success", Data: data})
}

func (h *EveSSOHandler) GetEveSsoBind(ctx context.Context, request apicontract.GetEveSsoBindRequestObject) (apicontract.GetEveSsoBindResponseObject, error) {
	c := ctx.(*gin.Context)
	data, failure := h.eveSsoAuthorization(ssoRequestFromGin(c), ssoQueryValue(request.Params.Redirect), ssoQueryValue(request.Params.Scopes), true)
	out := &apicontract.GetEveSsoBind200JSONResponse{}
	if failure != nil {
		switch failure.code {
		case response.CodeUnauthorized:
			return apicontract.GetEveSsoBind401JSONResponse{UnauthorizedJSONResponse: apicontract.UnauthorizedJSONResponse(ssoAuthError(failure))}, nil
		case response.CodeForbidden:
			return apicontract.GetEveSsoBind403JSONResponse{ForbiddenJSONResponse: apicontract.ForbiddenJSONResponse(ssoAuthError(failure))}, nil
		}
		return out, out.FromBusinessError(contractBusinessError(failure.code, failure.message))
	}
	return out, out.FromEveSsoAuthorizationSuccess(apicontract.EveSsoAuthorizationSuccess{Code: response.CodeOK, Msg: "success", Data: data})
}

func (h *EveSSOHandler) SetPrimaryEveCharacter(ctx context.Context, request apicontract.SetPrimaryEveCharacterRequestObject) (apicontract.SetPrimaryEveCharacterResponseObject, error) {
	c := ctx.(*gin.Context)
	out := &apicontract.SetPrimaryEveCharacter200JSONResponse{}
	if failure := changeEveSsoCharacter(middleware.GetUserID(c), request.CharacterId, h.svc.SetPrimaryCharacter); failure != nil {
		if failure.code == response.CodeUnauthorized {
			return apicontract.SetPrimaryEveCharacter401JSONResponse{UnauthorizedJSONResponse: apicontract.UnauthorizedJSONResponse(ssoAuthError(failure))}, nil
		}
		return out, out.FromBusinessError(contractBusinessError(failure.code, failure.message))
	}
	return out, out.FromNullSuccess(apicontract.NullSuccess{Code: response.CodeOK, Msg: "success"})
}

func (h *EveSSOHandler) UnbindEveCharacter(ctx context.Context, request apicontract.UnbindEveCharacterRequestObject) (apicontract.UnbindEveCharacterResponseObject, error) {
	c := ctx.(*gin.Context)
	out := &apicontract.UnbindEveCharacter200JSONResponse{}
	if failure := changeEveSsoCharacter(middleware.GetUserID(c), request.CharacterId, h.svc.UnbindCharacter); failure != nil {
		if failure.code == response.CodeUnauthorized {
			return apicontract.UnbindEveCharacter401JSONResponse{UnauthorizedJSONResponse: apicontract.UnauthorizedJSONResponse(ssoAuthError(failure))}, nil
		}
		return out, out.FromBusinessError(contractBusinessError(failure.code, failure.message))
	}
	return out, out.FromNullSuccess(apicontract.NullSuccess{Code: response.CodeOK, Msg: "success"})
}

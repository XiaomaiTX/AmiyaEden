package handler

import (
	"amiya-eden/internal/middleware"
	"context"
	"errors"
	"github.com/gin-gonic/gin"

	"amiya-eden/internal/apicontract"
	"amiya-eden/internal/model"
	"amiya-eden/internal/service"
	"amiya-eden/pkg/response"
)

const (
	sysConfigInvalidInputMessage = "请求参数错误"
	sysConfigSuperAdminMessage   = "仅超级管理员可修改该配置"
	sysConfigSDEStatusMessage    = "读取 SDE 状态失败"
	sysConfigSDECheckPrefix      = "检查 SDE 版本失败: "
	sysConfigSDETriggerPrefix    = "执行 SDE 更新失败: "
)

type SysConfigHandler struct {
	cfgSvc        *service.SysConfigService
	corpPolicySvc *service.CorporationPolicyService
	walletSvc     *service.SysWalletService
	sdeSvc        sdeSysConfigService
	auditSvc      *service.AuditService
}

type sdeSysConfigService interface {
	GetStatus() (service.SDEStatus, error)
	CheckLatestVersion() (service.SDEStatus, error)
	TriggerManualUpdateWithStatus() (service.SDEStatus, error)
}

func NewSysConfigHandler() *SysConfigHandler {
	return newSysConfigHandlerWithDeps(service.NewSysConfigService(), service.NewSdeService())
}

func newSysConfigHandlerWithDeps(
	cfgSvc *service.SysConfigService,
	sdeSvc sdeSysConfigService,
) *SysConfigHandler {
	return &SysConfigHandler{
		cfgSvc:        cfgSvc,
		corpPolicySvc: service.NewCorporationPolicyService(),
		walletSvc:     service.NewSysWalletService(),
		sdeSvc:        sdeSvc,
		auditSvc:      service.NewAuditService(),
	}
}

func systemBasicConfig() apicontract.SystemBasicConfig {
	identity := model.DefaultSystemIdentity()
	return apicontract.SystemBasicConfig{CorpId: identity.CorpID, SiteTitle: identity.SiteTitle}
}

func (h *SysConfigHandler) getSDEConfig() apicontract.SystemSDEConfig {
	config := h.cfgSvc.GetSDEConfig()
	return apicontract.SystemSDEConfig{
		APIKey:      config.APIKey,
		Proxy:       config.Proxy,
		DownloadURL: config.DownloadURL,
	}
}

func (h *SysConfigHandler) updateSDEConfig(ctx context.Context, actorID uint, req apicontract.SystemSDEConfigInput) error {
	if err := h.cfgSvc.UpdateSDEConfig(req.APIKey, req.Proxy, req.DownloadURL); err != nil {
		return err
	}
	h.recordConfigAudit(ctx, actorID, "sde_config_update", model.SysConfigSDEAPIKey, map[string]any{
		"proxy_updated": req.Proxy != nil, "download_url_updated": req.DownloadURL != nil, "api_key_updated": req.APIKey != nil,
	})
	return nil
}

func (h *SysConfigHandler) getAlliancePAPConfig() apicontract.SystemAlliancePAPConfig {
	cfg := h.cfgSvc.GetAlliancePAPConfig()
	return apicontract.SystemAlliancePAPConfig{BaseURL: cfg.BaseURL, APIKey: cfg.APIKey}
}

func (h *SysConfigHandler) updateAlliancePAPConfig(ctx context.Context, actorID uint, req apicontract.SystemAlliancePAPConfigInput) error {
	if err := h.cfgSvc.UpdateAlliancePAPConfig(req.BaseURL, req.APIKey); err != nil {
		return err
	}
	h.recordConfigAudit(ctx, actorID, "alliance_pap_config_update", model.SysConfigAlliancePAPBaseURL, map[string]any{
		"base_url_updated": req.BaseURL != nil, "api_key_updated": req.APIKey != nil,
	})
	return nil
}

func (h *SysConfigHandler) getMumbleConfig() apicontract.SystemMumbleConfig {
	cfg := h.cfgSvc.GetMumbleConfig()
	nodes := make([]apicontract.MumblePublicNode, 0, len(cfg.PublicNodes))
	for _, node := range cfg.PublicNodes {
		nodes = append(nodes, apicontract.MumblePublicNode{
			Name: node.Name, Description: node.Description, Address: node.Address, Port: node.Port,
		})
	}
	return apicontract.SystemMumbleConfig{
		ServiceToken: cfg.ServiceToken, ServerURL: cfg.ServerURL, RevalidateToken: cfg.RevalidateToken,
		RevalidateTimeoutMS: cfg.RevalidateTimeoutMS, PublicNodes: nodes,
		DisplayNameTemplate: cfg.DisplayNameTemplate,
	}
}

func (h *SysConfigHandler) updateMumbleConfig(ctx context.Context, actorID uint, req apicontract.SystemMumbleConfigInput) error {
	nodes := make([]service.MumblePublicNode, 0, len(req.PublicNodes))
	for _, node := range req.PublicNodes {
		nodes = append(nodes, service.MumblePublicNode{
			Name: node.Name, Description: node.Description, Address: node.Address, Port: node.Port,
		})
	}
	if err := h.cfgSvc.UpdateMumbleConfig(service.MumbleRuntimeConfig{
		ServiceToken: req.ServiceToken, ServerURL: req.ServerURL, RevalidateToken: req.RevalidateToken,
		RevalidateTimeoutMS: req.RevalidateTimeoutMS, PublicNodes: nodes,
		DisplayNameTemplate: req.DisplayNameTemplate,
	}); err != nil {
		return err
	}
	h.recordConfigAudit(ctx, actorID, "mumble_config_update", model.SysConfigMumbleServerURL, map[string]any{
		"server_url_updated": true, "service_token_updated": true, "revalidate_token_updated": true,
		"revalidate_timeout_ms": req.RevalidateTimeoutMS, "display_name_template": req.DisplayNameTemplate,
		"public_nodes": len(req.PublicNodes),
	})
	return nil
}

func (h *SysConfigHandler) recordConfigAudit(ctx context.Context, actorID uint, action, resourceID string, details map[string]any) {
	if h.auditSvc == nil {
		return
	}
	_ = h.auditSvc.RecordEvent(ctx, service.AuditRecordInput{
		Category: "config", Action: action, ActorUserID: actorID, ResourceType: "system_config",
		ResourceID: resourceID, Result: model.AuditResultSuccess, Details: details,
	})
}

func systemSDEStatus(status service.SDEStatus) apicontract.SystemSDEStatus {
	return apicontract.SystemSDEStatus{
		CurrentVersion: status.CurrentVersion, LatestVersion: status.LatestVersion, HasUpdate: status.HasUpdate,
		LastQueryError: status.LastQueryError, LastQueryErrorAt: status.LastQueryErrorAt, LastQuerySource: status.LastQuerySource,
		LastCheckAt: status.LastCheckAt, LastCheckSuccess: status.LastCheckSuccess, LastCheckError: status.LastCheckError,
		LastUpdateAt: status.LastUpdateAt, LastUpdateSuccess: status.LastUpdateSuccess, LastUpdateError: status.LastUpdateError,
		IsUpdating: status.IsUpdating, UpdateStage: apicontract.SystemSDEUpdateStage(status.UpdateStage),
	}
}

func (h *SysConfigHandler) getAllowCorporations(ctx context.Context) apicontract.SystemAllowedCorporations {
	allowed := h.cfgSvc.GetAllowCorporations()
	displays := h.cfgSvc.GetAllowCorporationDisplays(ctx)
	corporations := make([]apicontract.SystemConfigCorporationDisplay, 0, len(displays))
	for _, display := range displays {
		corporations = append(corporations, apicontract.SystemConfigCorporationDisplay{
			CorporationId: display.CorporationID, CorporationName: display.CorporationName,
		})
	}
	return apicontract.SystemAllowedCorporations{AllowCorporations: allowed, Corporations: corporations}
}

func (h *SysConfigHandler) updateAllowCorporations(req apicontract.SystemAllowedCorporationsInput) *contractOperationFailure {
	if err := h.cfgSvc.UpdateAllowCorporations(req.AllowCorporations); err != nil {
		if errors.Is(err, service.ErrInvalidAllowCorporations) {
			return &contractOperationFailure{code: response.CodeParamError, message: err.Error()}
		}
		return &contractOperationFailure{code: response.CodeBizError, message: err.Error()}
	}
	return nil
}

func (h *SysConfigHandler) getCharacterESIRestrictionConfig() apicontract.SystemCharacterESIRestriction {
	return apicontract.SystemCharacterESIRestriction{
		EnforceCharacterESIRestriction: h.cfgSvc.GetCharacterESIRestrictionConfig(),
	}
}

func (h *SysConfigHandler) getCorporationAccessPolicies() apicontract.SystemCorporationAccessPolicies {
	policies := h.corpPolicySvc.GetPolicies()
	rows := make([]apicontract.SystemConfigCorporationPolicy, 0, len(policies.Policies))
	for _, policy := range policies.Policies {
		rows = append(rows, apicontract.SystemConfigCorporationPolicy{
			CorporationId: policy.CorporationID, FullAccess: policy.FullAccess,
			Capabilities: policy.Capabilities, Rules: policy.Rules,
		})
	}
	return apicontract.SystemCorporationAccessPolicies{
		Version:              policies.Version,
		DefaultMode:          apicontract.SystemConfigPolicyMode(policies.DefaultMode),
		Policies:             rows,
		EnforcedCapabilities: model.EnforcedCorpCapabilities(),
		LegacyCapabilities:   h.corpPolicySvc.StoredLegacyCapabilities(),
	}
}

func (h *SysConfigHandler) updateCorporationAccessPolicies(ctx context.Context, actorID uint, req apicontract.SystemCorporationAccessPoliciesInput) *contractOperationFailure {
	before := h.corpPolicySvc.GetPolicies()
	policies := make([]service.CorporationPolicy, 0, len(req.Policies))
	for _, policy := range req.Policies {
		policies = append(policies, service.CorporationPolicy{
			CorporationID: policy.CorporationId, FullAccess: policy.FullAccess,
			Capabilities: policy.Capabilities, Rules: policy.Rules,
		})
	}
	config := service.CorporationPolicyConfig{
		Version:     req.Version,
		DefaultMode: string(req.DefaultMode),
		Policies:    policies,
	}
	if err := h.corpPolicySvc.UpdatePolicies(config); err != nil {
		if errors.Is(err, service.ErrInvalidCorporationAccessPolicy) {
			return &contractOperationFailure{code: response.CodeParamError, message: err.Error()}
		}
		return &contractOperationFailure{code: response.CodeBizError, message: err.Error()}
	}
	if err := h.walletSvc.ForceZeroBalancesForUsersWithoutWalletCapability(); err != nil {
		return &contractOperationFailure{code: response.CodeBizError, message: err.Error()}
	}
	after := h.corpPolicySvc.GetPolicies()
	h.recordConfigAudit(ctx, actorID, "corporation_access_policy_update", model.SysConfigCorporationAccessPolicies, map[string]any{
		"before": before,
		"after":  after,
	})
	return nil
}

func sysConfigUnauthorized() apicontract.UnauthorizedJSONResponse {
	return apicontract.UnauthorizedJSONResponse{Code: response.CodeUnauthorized, Msg: "未登录"}
}

func (h *SysConfigHandler) GetBasicConfig(_ context.Context, _ apicontract.GetBasicConfigRequestObject) (apicontract.GetBasicConfigResponseObject, error) {
	return apicontract.GetBasicConfig200JSONResponse{Code: response.CodeOK, Msg: "success", Data: systemBasicConfig()}, nil
}

func (h *SysConfigHandler) GetSDEConfig(_ context.Context, _ apicontract.GetSDEConfigRequestObject) (apicontract.GetSDEConfigResponseObject, error) {
	return apicontract.GetSDEConfig200JSONResponse{Code: response.CodeOK, Msg: "success", Data: h.getSDEConfig()}, nil
}

func (h *SysConfigHandler) UpdateSDEConfig(ctx context.Context, request apicontract.UpdateSDEConfigRequestObject) (apicontract.UpdateSDEConfigResponseObject, error) {
	c, _ := ctx.(*gin.Context)
	if c == nil {
		return apicontract.UpdateSDEConfig401JSONResponse{UnauthorizedJSONResponse: sysConfigUnauthorized()}, nil
	}
	var out apicontract.UpdateSDEConfig200JSONResponse
	if request.Body == nil {
		return &out, out.FromBusinessError(contractBusinessError(response.CodeParamError, sysConfigInvalidInputMessage))
	}
	if err := h.updateSDEConfig(c.Request.Context(), middleware.GetUserID(c), *request.Body); err != nil {
		return &out, out.FromBusinessError(contractBusinessError(response.CodeBizError, err.Error()))
	}
	return &out, out.FromNullSuccess(apicontract.NullSuccess{Code: response.CodeOK, Msg: "success"})
}

func (h *SysConfigHandler) GetAlliancePAPConfig(_ context.Context, _ apicontract.GetAlliancePAPConfigRequestObject) (apicontract.GetAlliancePAPConfigResponseObject, error) {
	return apicontract.GetAlliancePAPConfig200JSONResponse{Code: response.CodeOK, Msg: "success", Data: h.getAlliancePAPConfig()}, nil
}

func (h *SysConfigHandler) UpdateAlliancePAPConfig(ctx context.Context, request apicontract.UpdateAlliancePAPConfigRequestObject) (apicontract.UpdateAlliancePAPConfigResponseObject, error) {
	c, _ := ctx.(*gin.Context)
	if c == nil {
		return apicontract.UpdateAlliancePAPConfig401JSONResponse{UnauthorizedJSONResponse: sysConfigUnauthorized()}, nil
	}
	var out apicontract.UpdateAlliancePAPConfig200JSONResponse
	if request.Body == nil {
		return &out, out.FromBusinessError(contractBusinessError(response.CodeParamError, sysConfigInvalidInputMessage))
	}
	if err := h.updateAlliancePAPConfig(c.Request.Context(), middleware.GetUserID(c), *request.Body); err != nil {
		return &out, out.FromBusinessError(contractBusinessError(response.CodeBizError, err.Error()))
	}
	return &out, out.FromNullSuccess(apicontract.NullSuccess{Code: response.CodeOK, Msg: "success"})
}

func (h *SysConfigHandler) GetMumbleConfig(_ context.Context, _ apicontract.GetMumbleConfigRequestObject) (apicontract.GetMumbleConfigResponseObject, error) {
	return apicontract.GetMumbleConfig200JSONResponse{Code: response.CodeOK, Msg: "success", Data: h.getMumbleConfig()}, nil
}

func (h *SysConfigHandler) UpdateMumbleConfig(ctx context.Context, request apicontract.UpdateMumbleConfigRequestObject) (apicontract.UpdateMumbleConfigResponseObject, error) {
	c, _ := ctx.(*gin.Context)
	if c == nil {
		return apicontract.UpdateMumbleConfig401JSONResponse{UnauthorizedJSONResponse: sysConfigUnauthorized()}, nil
	}
	var out apicontract.UpdateMumbleConfig200JSONResponse
	if request.Body == nil {
		return &out, out.FromBusinessError(contractBusinessError(response.CodeParamError, sysConfigInvalidInputMessage))
	}
	if err := h.updateMumbleConfig(c.Request.Context(), middleware.GetUserID(c), *request.Body); err != nil {
		return &out, out.FromBusinessError(contractBusinessError(response.CodeParamError, err.Error()))
	}
	return &out, out.FromNullSuccess(apicontract.NullSuccess{Code: response.CodeOK, Msg: "success"})
}

func (h *SysConfigHandler) GetSDEStatus(_ context.Context, _ apicontract.GetSDEStatusRequestObject) (apicontract.GetSDEStatusResponseObject, error) {
	var out apicontract.GetSDEStatus200JSONResponse
	status, err := h.sdeSvc.GetStatus()
	if err != nil {
		return &out, out.FromBusinessError(contractBusinessError(response.CodeBizError, sysConfigSDEStatusMessage))
	}
	return &out, out.FromSystemSDEStatusSuccess(apicontract.SystemSDEStatusSuccess{Code: response.CodeOK, Msg: "success", Data: systemSDEStatus(status)})
}

func (h *SysConfigHandler) CheckSDEVersion(_ context.Context, _ apicontract.CheckSDEVersionRequestObject) (apicontract.CheckSDEVersionResponseObject, error) {
	var out apicontract.CheckSDEVersion200JSONResponse
	status, err := h.sdeSvc.CheckLatestVersion()
	if err != nil {
		return &out, out.FromBusinessError(contractBusinessError(response.CodeBizError, sysConfigSDECheckPrefix+err.Error()))
	}
	return &out, out.FromSystemSDEStatusSuccess(apicontract.SystemSDEStatusSuccess{Code: response.CodeOK, Msg: "success", Data: systemSDEStatus(status)})
}

func (h *SysConfigHandler) TriggerSDEUpdate(_ context.Context, _ apicontract.TriggerSDEUpdateRequestObject) (apicontract.TriggerSDEUpdateResponseObject, error) {
	var out apicontract.TriggerSDEUpdate200JSONResponse
	status, err := h.sdeSvc.TriggerManualUpdateWithStatus()
	if err != nil {
		return &out, out.FromBusinessError(contractBusinessError(response.CodeBizError, sysConfigSDETriggerPrefix+err.Error()))
	}
	return &out, out.FromSystemSDEStatusSuccess(apicontract.SystemSDEStatusSuccess{Code: response.CodeOK, Msg: "success", Data: systemSDEStatus(status)})
}

func (h *SysConfigHandler) GetAllowCorporations(ctx context.Context, _ apicontract.GetAllowCorporationsRequestObject) (apicontract.GetAllowCorporationsResponseObject, error) {
	if c, ok := ctx.(*gin.Context); ok && c != nil {
		ctx = c.Request.Context()
	}
	return apicontract.GetAllowCorporations200JSONResponse{Code: response.CodeOK, Msg: "success", Data: h.getAllowCorporations(ctx)}, nil
}

func (h *SysConfigHandler) UpdateAllowCorporations(_ context.Context, request apicontract.UpdateAllowCorporationsRequestObject) (apicontract.UpdateAllowCorporationsResponseObject, error) {
	var out apicontract.UpdateAllowCorporations200JSONResponse
	if request.Body == nil {
		return &out, out.FromBusinessError(contractBusinessError(response.CodeParamError, sysConfigInvalidInputMessage))
	}
	if failure := h.updateAllowCorporations(*request.Body); failure != nil {
		return &out, out.FromBusinessError(contractBusinessError(failure.code, failure.message))
	}
	return &out, out.FromNullSuccess(apicontract.NullSuccess{Code: response.CodeOK, Msg: "success"})
}

func (h *SysConfigHandler) GetCharacterESIRestrictionConfig(_ context.Context, _ apicontract.GetCharacterESIRestrictionConfigRequestObject) (apicontract.GetCharacterESIRestrictionConfigResponseObject, error) {
	return apicontract.GetCharacterESIRestrictionConfig200JSONResponse{Code: response.CodeOK, Msg: "success", Data: h.getCharacterESIRestrictionConfig()}, nil
}

func (h *SysConfigHandler) UpdateCharacterESIRestrictionConfig(ctx context.Context, request apicontract.UpdateCharacterESIRestrictionConfigRequestObject) (apicontract.UpdateCharacterESIRestrictionConfigResponseObject, error) {
	c, _ := ctx.(*gin.Context)
	if c == nil {
		return apicontract.UpdateCharacterESIRestrictionConfig401JSONResponse{UnauthorizedJSONResponse: sysConfigUnauthorized()}, nil
	}
	if !model.IsSuperAdmin(middleware.GetUserRoles(c)) {
		return apicontract.UpdateCharacterESIRestrictionConfig403JSONResponse{
			ForbiddenJSONResponse: apicontract.ForbiddenJSONResponse{Code: response.CodeForbidden, Msg: sysConfigSuperAdminMessage},
		}, nil
	}
	var out apicontract.UpdateCharacterESIRestrictionConfig200JSONResponse
	if request.Body == nil || request.Body.EnforceCharacterESIRestriction == nil {
		return &out, out.FromBusinessError(contractBusinessError(response.CodeParamError, sysConfigInvalidInputMessage))
	}
	if err := h.cfgSvc.UpdateCharacterESIRestrictionConfig(*request.Body.EnforceCharacterESIRestriction); err != nil {
		return &out, out.FromBusinessError(contractBusinessError(response.CodeBizError, err.Error()))
	}
	return &out, out.FromNullSuccess(apicontract.NullSuccess{Code: response.CodeOK, Msg: "success"})
}

func (h *SysConfigHandler) GetCorporationAccessPolicies(_ context.Context, _ apicontract.GetCorporationAccessPoliciesRequestObject) (apicontract.GetCorporationAccessPoliciesResponseObject, error) {
	return apicontract.GetCorporationAccessPolicies200JSONResponse{Code: response.CodeOK, Msg: "success", Data: h.getCorporationAccessPolicies()}, nil
}

func (h *SysConfigHandler) UpdateCorporationAccessPolicies(ctx context.Context, request apicontract.UpdateCorporationAccessPoliciesRequestObject) (apicontract.UpdateCorporationAccessPoliciesResponseObject, error) {
	c, _ := ctx.(*gin.Context)
	if c == nil {
		return apicontract.UpdateCorporationAccessPolicies401JSONResponse{UnauthorizedJSONResponse: sysConfigUnauthorized()}, nil
	}
	var out apicontract.UpdateCorporationAccessPolicies200JSONResponse
	if request.Body == nil {
		return &out, out.FromBusinessError(contractBusinessError(response.CodeParamError, sysConfigInvalidInputMessage))
	}
	if failure := h.updateCorporationAccessPolicies(c.Request.Context(), middleware.GetUserID(c), *request.Body); failure != nil {
		return &out, out.FromBusinessError(contractBusinessError(failure.code, failure.message))
	}
	return &out, out.FromNullSuccess(apicontract.NullSuccess{Code: response.CodeOK, Msg: "success"})
}

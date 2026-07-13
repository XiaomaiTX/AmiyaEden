package handler

import (
	"errors"

	"amiya-eden/internal/middleware"
	"amiya-eden/internal/model"
	"amiya-eden/internal/service"
	"amiya-eden/pkg/response"

	"github.com/gin-gonic/gin"
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

func (h *SysConfigHandler) GetBasicConfig(c *gin.Context) {
	response.OK(c, model.DefaultSystemIdentity())
}

// SDEConfigResponse SDE 配置响应
type SDEConfigResponse struct {
	APIKey      string `json:"api_key"`
	Proxy       string `json:"proxy"`
	DownloadURL string `json:"download_url"`
}

// UpdateSDEConfigRequest 更新 SDE 配置请求
type UpdateSDEConfigRequest struct {
	APIKey      *string `json:"api_key"`
	Proxy       *string `json:"proxy"`
	DownloadURL *string `json:"download_url"`
}

func (h *SysConfigHandler) GetSDEConfig(c *gin.Context) {
	config := h.cfgSvc.GetSDEConfig()

	response.OK(c, SDEConfigResponse{
		APIKey:      config.APIKey,
		Proxy:       config.Proxy,
		DownloadURL: config.DownloadURL,
	})
}

func (h *SysConfigHandler) UpdateSDEConfig(c *gin.Context) {
	var req UpdateSDEConfigRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, response.CodeParamError, "请求参数错误")
		return
	}

	if err := h.cfgSvc.UpdateSDEConfig(req.APIKey, req.Proxy, req.DownloadURL); err != nil {
		response.Fail(c, response.CodeBizError, err.Error())
		return
	}
	h.recordConfigAudit(c, "sde_config_update", model.SysConfigSDEAPIKey, map[string]any{
		"proxy_updated": req.Proxy != nil, "download_url_updated": req.DownloadURL != nil, "api_key_updated": req.APIKey != nil,
	})

	response.OK(c, nil)
}

type AlliancePAPConfigResponse struct {
	BaseURL string `json:"base_url"`
	APIKey  string `json:"api_key"`
}

type UpdateAlliancePAPConfigRequest struct {
	BaseURL *string `json:"base_url"`
	APIKey  *string `json:"api_key"`
}

func (h *SysConfigHandler) GetAlliancePAPConfig(c *gin.Context) {
	cfg := h.cfgSvc.GetAlliancePAPConfig()
	response.OK(c, AlliancePAPConfigResponse{BaseURL: cfg.BaseURL, APIKey: cfg.APIKey})
}

func (h *SysConfigHandler) UpdateAlliancePAPConfig(c *gin.Context) {
	var req UpdateAlliancePAPConfigRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, response.CodeParamError, "请求参数错误")
		return
	}
	if err := h.cfgSvc.UpdateAlliancePAPConfig(req.BaseURL, req.APIKey); err != nil {
		response.Fail(c, response.CodeBizError, err.Error())
		return
	}
	h.recordConfigAudit(c, "alliance_pap_config_update", model.SysConfigAlliancePAPBaseURL, map[string]any{
		"base_url_updated": req.BaseURL != nil, "api_key_updated": req.APIKey != nil,
	})
	response.OK(c, nil)
}

type OneBotConfigResponse struct {
	Enabled      bool     `json:"enabled"`
	AccessToken  string   `json:"access_token"`
	BotQQ        int64    `json:"bot_qq"`
	AllowedCIDRs []string `json:"allowed_cidrs"`
}

type UpdateOneBotConfigRequest struct {
	Enabled      *bool     `json:"enabled"`
	AccessToken  *string   `json:"access_token"`
	BotQQ        *int64    `json:"bot_qq"`
	AllowedCIDRs *[]string `json:"allowed_cidrs"`
}

func (h *SysConfigHandler) GetOneBotConfig(c *gin.Context) {
	cfg := h.cfgSvc.GetOneBotConfig()
	response.OK(c, OneBotConfigResponse{Enabled: cfg.Enabled, AccessToken: cfg.AccessToken, BotQQ: cfg.BotQQ, AllowedCIDRs: cfg.AllowedCIDRs})
}

func (h *SysConfigHandler) UpdateOneBotConfig(c *gin.Context) {
	var req UpdateOneBotConfigRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, response.CodeParamError, "请求参数错误")
		return
	}
	if err := h.cfgSvc.UpdateOneBotConfig(req.Enabled, req.AccessToken, req.BotQQ, req.AllowedCIDRs); err != nil {
		response.Fail(c, response.CodeParamError, err.Error())
		return
	}
	h.recordConfigAudit(c, "onebot_config_update", model.SysConfigOneBotEnabled, map[string]any{
		"enabled_updated": req.Enabled != nil, "bot_qq_updated": req.BotQQ != nil,
		"allowed_cidrs_updated": req.AllowedCIDRs != nil, "access_token_updated": req.AccessToken != nil,
	})
	response.OK(c, nil)
}

func (h *SysConfigHandler) recordConfigAudit(c *gin.Context, action, resourceID string, details map[string]any) {
	if h.auditSvc == nil {
		return
	}
	_ = h.auditSvc.RecordEvent(c.Request.Context(), service.AuditRecordInput{
		Category: "config", Action: action, ActorUserID: middleware.GetUserID(c), ResourceType: "system_config",
		ResourceID: resourceID, Result: model.AuditResultSuccess, Details: details,
	})
}

func (h *SysConfigHandler) GetSDEStatus(c *gin.Context) {
	status, err := h.sdeSvc.GetStatus()
	if err != nil {
		response.Fail(c, response.CodeBizError, "读取 SDE 状态失败")
		return
	}
	response.OK(c, status)
}

func (h *SysConfigHandler) CheckSDEVersion(c *gin.Context) {
	status, err := h.sdeSvc.CheckLatestVersion()
	if err != nil {
		response.Fail(c, response.CodeBizError, "检查 SDE 版本失败: "+err.Error())
		return
	}
	response.OK(c, status)
}

func (h *SysConfigHandler) TriggerSDEUpdate(c *gin.Context) {
	status, err := h.sdeSvc.TriggerManualUpdateWithStatus()
	if err != nil {
		response.Fail(c, response.CodeBizError, "执行 SDE 更新失败: "+err.Error())
		return
	}
	response.OK(c, status)
}

type AllowCorporationsResponse struct {
	AllowCorporations []int64                      `json:"allow_corporations"`
	Corporations      []service.CorporationDisplay `json:"corporations"`
}

type UpdateAllowCorporationsRequest struct {
	AllowCorporations []int64 `json:"allow_corporations"`
}

type CharacterESIRestrictionConfigResponse struct {
	EnforceCharacterESIRestriction bool `json:"enforce_character_esi_restriction"`
}

type UpdateCharacterESIRestrictionConfigRequest struct {
	EnforceCharacterESIRestriction *bool `json:"enforce_character_esi_restriction"`
}

type CorporationAccessPoliciesResponse struct {
	Version     int                         `json:"version"`
	DefaultMode string                      `json:"default_mode"`
	Policies    []service.CorporationPolicy `json:"policies"`
}

type UpdateCorporationAccessPoliciesRequest struct {
	Version     int                         `json:"version"`
	DefaultMode string                      `json:"default_mode"`
	Policies    []service.CorporationPolicy `json:"policies"`
}

func (h *SysConfigHandler) GetAllowCorporations(c *gin.Context) {
	response.OK(c, AllowCorporationsResponse{
		AllowCorporations: h.cfgSvc.GetAllowCorporations(),
		Corporations:      h.cfgSvc.GetAllowCorporationDisplays(c.Request.Context()),
	})
}

func (h *SysConfigHandler) UpdateAllowCorporations(c *gin.Context) {
	var req UpdateAllowCorporationsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, response.CodeParamError, "请求参数错误")
		return
	}
	if err := h.cfgSvc.UpdateAllowCorporations(req.AllowCorporations); err != nil {
		if errors.Is(err, service.ErrInvalidAllowCorporations) {
			response.Fail(c, response.CodeParamError, err.Error())
			return
		}
		response.Fail(c, response.CodeBizError, err.Error())
		return
	}

	response.OK(c, nil)
}

func (h *SysConfigHandler) GetCharacterESIRestrictionConfig(c *gin.Context) {
	response.OK(c, CharacterESIRestrictionConfigResponse{
		EnforceCharacterESIRestriction: h.cfgSvc.GetCharacterESIRestrictionConfig(),
	})
}

func (h *SysConfigHandler) UpdateCharacterESIRestrictionConfig(c *gin.Context) {
	if !model.IsSuperAdmin(middleware.GetUserRoles(c)) {
		response.Fail(c, response.CodeForbidden, "仅超级管理员可修改该配置")
		return
	}

	var req UpdateCharacterESIRestrictionConfigRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, response.CodeParamError, "请求参数错误")
		return
	}
	if req.EnforceCharacterESIRestriction == nil {
		response.Fail(c, response.CodeParamError, "请求参数错误")
		return
	}

	if err := h.cfgSvc.UpdateCharacterESIRestrictionConfig(*req.EnforceCharacterESIRestriction); err != nil {
		response.Fail(c, response.CodeBizError, err.Error())
		return
	}

	response.OK(c, nil)
}

func (h *SysConfigHandler) GetCorporationAccessPolicies(c *gin.Context) {
	policies := h.corpPolicySvc.GetPolicies()
	response.OK(c, CorporationAccessPoliciesResponse{
		Version:     policies.Version,
		DefaultMode: policies.DefaultMode,
		Policies:    policies.Policies,
	})
}

func (h *SysConfigHandler) UpdateCorporationAccessPolicies(c *gin.Context) {
	var req UpdateCorporationAccessPoliciesRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, response.CodeParamError, "请求参数错误")
		return
	}
	if err := h.corpPolicySvc.UpdatePolicies(service.CorporationPolicyConfig{
		Version:     req.Version,
		DefaultMode: req.DefaultMode,
		Policies:    req.Policies,
	}); err != nil {
		if errors.Is(err, service.ErrInvalidCorporationAccessPolicy) {
			response.Fail(c, response.CodeParamError, err.Error())
			return
		}
		response.Fail(c, response.CodeBizError, err.Error())
		return
	}
	if err := h.walletSvc.ForceZeroBalancesForUsersWithoutWalletCapability(); err != nil {
		response.Fail(c, response.CodeBizError, err.Error())
		return
	}
	response.OK(c, nil)
}

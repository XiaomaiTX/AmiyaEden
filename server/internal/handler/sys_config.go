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
	cfgSvc *service.SysConfigService
	sdeSvc sdeSysConfigService
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
		cfgSvc: cfgSvc,
		sdeSvc: sdeSvc,
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

	response.OK(c, nil)
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
	AllowCorporations []int64 `json:"allow_corporations"`
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

func (h *SysConfigHandler) GetAllowCorporations(c *gin.Context) {
	response.OK(c, AllowCorporationsResponse{
		AllowCorporations: h.cfgSvc.GetAllowCorporations(),
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

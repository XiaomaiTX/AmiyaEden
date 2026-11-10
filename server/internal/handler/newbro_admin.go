package handler

import (
	"amiya-eden/internal/middleware"
	"amiya-eden/internal/service"
	"amiya-eden/pkg/response"
	"github.com/gin-gonic/gin"
)

type NewbroAdminHandler struct {
	settingsSvc *service.NewbroSettingsService
}

func NewNewbroAdminHandler() *NewbroAdminHandler {
	return &NewbroAdminHandler{service.NewNewbroSettingsService()}
}

type UpdateNewbroRecruitSettingsRequest struct {
	RecruitQQURL        string   `json:"recruit_qq_url"`
	RecruitRewardAmount *float64 `json:"recruit_reward_amount" binding:"required,gte=0"`
	RecruitCooldownDays int      `json:"recruit_cooldown_days" binding:"required,gt=0"`
}

func (h *NewbroAdminHandler) GetRecruitSettings(c *gin.Context) {
	response.OK(c, h.settingsSvc.GetRecruitSettings())
}

func (h *NewbroAdminHandler) UpdateRecruitSettings(c *gin.Context) {
	var req UpdateNewbroRecruitSettingsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, response.CodeParamError, "invalid request: "+err.Error())
		return
	}

	updated, err := h.settingsSvc.UpdateRecruitSettingsByOperator(service.NewbroRecruitSettings{
		RecruitQQURL:        req.RecruitQQURL,
		RecruitRewardAmount: *req.RecruitRewardAmount,
		RecruitCooldownDays: req.RecruitCooldownDays,
	}, middleware.GetUserID(c))
	if err != nil {
		response.Fail(c, response.CodeBizError, err.Error())
		return
	}

	response.OK(c, updated)
}

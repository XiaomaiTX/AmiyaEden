package handler

import (
	"amiya-eden/internal/middleware"
	"amiya-eden/internal/service"
	"amiya-eden/pkg/response"

	"github.com/gin-gonic/gin"
)

type NewbroAdminHandler struct {
	reportSvc   *service.NewbroReportService
	settingsSvc newbroAdminSettingsService
}

type newbroAdminSettingsService interface {
	GetSupportSettings() service.NewbroSupportSettings
	UpdateSupportSettingsByOperator(cfg service.NewbroSupportSettings, operatorID uint) (service.NewbroSupportSettings, error)
	GetRecruitSettings() service.NewbroRecruitSettings
	UpdateRecruitSettingsByOperator(cfg service.NewbroRecruitSettings, operatorID uint) (service.NewbroRecruitSettings, error)
}

func NewNewbroAdminHandler() *NewbroAdminHandler {
	return &NewbroAdminHandler{
		reportSvc:   service.NewNewbroReportService(),
		settingsSvc: service.NewNewbroSettingsService(),
	}
}

func (h *NewbroAdminHandler) ListCaptains(c *gin.Context) {
	page, pageSize, err := parsePaginationQuery(c, 20, 100)
	if err != nil {
		response.Fail(c, response.CodeParamError, err.Error())
		return
	}
	result, total, err := h.reportSvc.ListAllCaptainOverviews(page, pageSize, c.Query("keyword"))
	if err != nil {
		response.Fail(c, response.CodeBizError, err.Error())
		return
	}
	response.OKWithPage(c, result, total, page, pageSize)
}

func (h *NewbroAdminHandler) GetCaptainDetail(c *gin.Context) {
	id, err := parseStrictUint(c.Param("user_id"))
	if err != nil {
		response.Fail(c, response.CodeParamError, "invalid user_id")
		return
	}
	result, err := h.reportSvc.GetAdminCaptainDetail(id)
	if err != nil {
		response.Fail(c, response.CodeBizError, err.Error())
		return
	}
	response.OK(c, result)
}

type UpdateNewbroSupportSettingsRequest struct {
	MaxCharacterSP          int64    `json:"max_character_sp" binding:"required,gt=0"`
	MultiCharacterSP        int64    `json:"multi_character_sp" binding:"required,gt=0"`
	MultiCharacterThreshold int      `json:"multi_character_threshold" binding:"required,gt=0"`
	RefreshIntervalDays     int      `json:"refresh_interval_days" binding:"required,gt=0"`
	BonusRate               *float64 `json:"bonus_rate" binding:"required,gte=0"`
}

type UpdateNewbroRecruitSettingsRequest struct {
	RecruitQQURL        string   `json:"recruit_qq_url"`
	RecruitRewardAmount *float64 `json:"recruit_reward_amount" binding:"required,gte=0"`
	RecruitCooldownDays int      `json:"recruit_cooldown_days" binding:"required,gt=0"`
}

func (h *NewbroAdminHandler) GetSupportSettings(c *gin.Context) {
	response.OK(c, h.settingsSvc.GetSupportSettings())
}

func (h *NewbroAdminHandler) UpdateSupportSettings(c *gin.Context) {
	var req UpdateNewbroSupportSettingsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, response.CodeParamError, "invalid request: "+err.Error())
		return
	}

	updated, err := h.settingsSvc.UpdateSupportSettingsByOperator(service.NewbroSupportSettings{
		MaxCharacterSP:          req.MaxCharacterSP,
		MultiCharacterSP:        req.MultiCharacterSP,
		MultiCharacterThreshold: req.MultiCharacterThreshold,
		RefreshIntervalDays:     req.RefreshIntervalDays,
		BonusRate:               *req.BonusRate,
	}, middleware.GetUserID(c))
	if err != nil {
		response.Fail(c, response.CodeBizError, err.Error())
		return
	}

	response.OK(c, updated)
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

func (h *NewbroAdminHandler) ListAffiliationHistory(c *gin.Context) {
	page, pageSize, err := parseLedgerPaginationQuery(c, 20)
	if err != nil {
		response.Fail(c, response.CodeParamError, err.Error())
		return
	}
	changeStartDate, err := parseOptionalNewbroDate(c.Query("change_start_date"), false)
	if err != nil {
		response.Fail(c, response.CodeParamError, err.Error())
		return
	}
	changeEndDate, err := parseOptionalNewbroDate(c.Query("change_end_date"), true)
	if err != nil {
		response.Fail(c, response.CodeParamError, err.Error())
		return
	}

	result, total, err := h.reportSvc.ListAdminAffiliationHistory(service.AdminAffiliationHistoryListRequest{
		Page:            page,
		PageSize:        pageSize,
		CaptainSearch:   c.Query("captain_search"),
		PlayerSearch:    c.Query("player_search"),
		ChangeStartDate: changeStartDate,
		ChangeEndDate:   changeEndDate,
	})
	if err != nil {
		response.Fail(c, response.CodeBizError, err.Error())
		return
	}
	response.OKWithPage(c, result, total, page, pageSize)
}

func (h *NewbroAdminHandler) ListRewardSettlements(c *gin.Context) {
	page, pageSize, err := parseLedgerPaginationQuery(c, 200)
	if err != nil {
		response.Fail(c, response.CodeParamError, err.Error())
		return
	}
	summary, result, total, err := h.reportSvc.ListAdminRewardSettlements(page, pageSize, c.Query("keyword"))
	if err != nil {
		response.Fail(c, response.CodeBizError, err.Error())
		return
	}
	response.OK(c, gin.H{
		"summary":   summary,
		"list":      result,
		"total":     total,
		"page":      page,
		"page_size": pageSize,
	})
}

package handler

import (
	"amiya-eden/internal/middleware"
	"amiya-eden/internal/service"
	"amiya-eden/pkg/response"
	"fmt"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

type GalaxyRegistryHandler struct {
	svc *service.GalaxyRegistryService
}

func NewGalaxyRegistryHandler() *GalaxyRegistryHandler {
	return &GalaxyRegistryHandler{svc: service.NewGalaxyRegistryService()}
}

type galaxyRegistryCreateEntryRequest struct {
	SystemConfigID uint   `json:"system_config_id" binding:"required"`
	ExpectedEndAt  string `json:"expected_end_at" binding:"required"`
}

type galaxyRegistryUpdateExpectedEndAtRequest struct {
	ExpectedEndAt string `json:"expected_end_at" binding:"required"`
}

type galaxyRegistryAdminCreateSystemRequest struct {
	SolarSystemID   int64    `json:"solar_system_id" binding:"required"`
	Note            string   `json:"note"`
	MinBountyAmount *float64 `json:"min_bounty_amount"`
	IsEnabled       *bool    `json:"is_enabled"`
}

type galaxyRegistryAdminUpdateSystemRequest struct {
	Note            *string  `json:"note"`
	MinBountyAmount *float64 `json:"min_bounty_amount"`
	IsEnabled       *bool    `json:"is_enabled"`
}

type galaxyRegistryAdminUpdateValidationRequest struct {
	ValidationStatus string  `json:"validation_status" binding:"required"`
	ViolationReason  *string `json:"violation_reason"`
}

func (h *GalaxyRegistryHandler) ListSystems(c *gin.Context) {
	rows, err := h.svc.ListSystems(middleware.GetUserID(c))
	if err != nil {
		response.Fail(c, response.CodeBizError, "获取星系列表失败")
		return
	}
	response.OK(c, rows)
}

func (h *GalaxyRegistryHandler) CreateEntry(c *gin.Context) {
	var req galaxyRegistryCreateEntryRequest
	if !bindJSON(c, &req) {
		return
	}
	expectedEndAt, err := parseGalaxyRegistryTime(req.ExpectedEndAt, false)
	if err != nil {
		response.Fail(c, response.CodeParamError, "无效的 expected_end_at")
		return
	}
	row, err := h.svc.StartEntry(middleware.GetUserID(c), service.GalaxyRegistryCreateEntryRequest{
		SystemConfigID: req.SystemConfigID,
		ExpectedEndAt:  expectedEndAt,
	})
	if err != nil {
		if service.IsUserVisibleError(err) {
			response.Fail(c, response.CodeBizError, err.Error())
			return
		}
		response.Fail(c, response.CodeBizError, "创建登记失败")
		return
	}
	response.OK(c, row)
}

func (h *GalaxyRegistryHandler) EndMyEntry(c *gin.Context) {
	entryID := requireUintID(c, "id", "登记 ID")
	if entryID == 0 {
		return
	}
	row, err := h.svc.EndMyEntryWithContext(c.Request.Context(), middleware.GetUserID(c), entryID)
	if err != nil {
		if service.IsUserVisibleError(err) {
			response.Fail(c, response.CodeBizError, err.Error())
			return
		}
		response.Fail(c, response.CodeBizError, "结束登记失败")
		return
	}
	response.OK(c, row)
}

func (h *GalaxyRegistryHandler) UpdateMyExpectedEndAt(c *gin.Context) {
	entryID := requireUintID(c, "id", "登记 ID")
	if entryID == 0 {
		return
	}
	var req galaxyRegistryUpdateExpectedEndAtRequest
	if !bindJSON(c, &req) {
		return
	}
	expectedEndAt, err := parseGalaxyRegistryTime(req.ExpectedEndAt, false)
	if err != nil {
		response.Fail(c, response.CodeParamError, "无效的 expected_end_at")
		return
	}
	row, err := h.svc.UpdateMyExpectedEndAt(
		middleware.GetUserID(c),
		entryID,
		service.GalaxyRegistryUpdateExpectedEndAtRequest{ExpectedEndAt: expectedEndAt},
	)
	if err != nil {
		if service.IsUserVisibleError(err) {
			response.Fail(c, response.CodeBizError, err.Error())
			return
		}
		response.Fail(c, response.CodeBizError, "更新预计结束时间失败")
		return
	}
	response.OK(c, row)
}

func (h *GalaxyRegistryHandler) ListMyEntries(c *gin.Context) {
	page, pageSize, err := parseLedgerPaginationQuery(c, 200)
	if err != nil {
		response.Fail(c, response.CodeParamError, err.Error())
		return
	}
	filter, err := parseGalaxyRegistryEntryFilter(c)
	if err != nil {
		response.Fail(c, response.CodeParamError, err.Error())
		return
	}
	rows, total, err := h.svc.ListMyEntries(middleware.GetUserID(c), filter, page, pageSize)
	if err != nil {
		response.Fail(c, response.CodeBizError, "获取我的登记失败")
		return
	}
	response.OKWithPage(c, rows, total, page, pageSize)
}

func (h *GalaxyRegistryHandler) SearchAdminSdeSystems(c *gin.Context) {
	limit, err := parseIntQuery(c, "limit", 20)
	if err != nil {
		response.Fail(c, response.CodeParamError, err.Error())
		return
	}
	rows, err := h.svc.SearchAdminSdeSystems(c.Query("keyword"), limit)
	if err != nil {
		response.Fail(c, response.CodeBizError, "搜索星系失败")
		return
	}
	response.OK(c, rows)
}

func (h *GalaxyRegistryHandler) ListAdminSystems(c *gin.Context) {
	rows, err := h.svc.ListAdminSystems()
	if err != nil {
		response.Fail(c, response.CodeBizError, "获取星系配置失败")
		return
	}
	response.OK(c, rows)
}

func (h *GalaxyRegistryHandler) CreateAdminSystem(c *gin.Context) {
	var req galaxyRegistryAdminCreateSystemRequest
	if !bindJSON(c, &req) {
		return
	}
	minBountyAmount := modelGalaxyRegistryDefaultMinBountyAmount()
	if req.MinBountyAmount != nil {
		minBountyAmount = *req.MinBountyAmount
	}
	isEnabled := true
	if req.IsEnabled != nil {
		isEnabled = *req.IsEnabled
	}
	row, err := h.svc.CreateAdminSystem(req.SolarSystemID, req.Note, minBountyAmount, isEnabled)
	if err != nil {
		if service.IsUserVisibleError(err) {
			response.Fail(c, response.CodeBizError, err.Error())
			return
		}
		response.Fail(c, response.CodeBizError, "创建星系配置失败")
		return
	}
	response.OK(c, row)
}

func (h *GalaxyRegistryHandler) UpdateAdminSystem(c *gin.Context) {
	id := requireUintID(c, "id", "星系配置 ID")
	if id == 0 {
		return
	}
	var req galaxyRegistryAdminUpdateSystemRequest
	if !bindJSON(c, &req) {
		return
	}
	row, err := h.svc.UpdateAdminSystem(id, req.Note, req.MinBountyAmount, req.IsEnabled)
	if err != nil {
		if service.IsUserVisibleError(err) {
			response.Fail(c, response.CodeBizError, err.Error())
			return
		}
		response.Fail(c, response.CodeBizError, "更新星系配置失败")
		return
	}
	response.OK(c, row)
}

func (h *GalaxyRegistryHandler) DeleteAdminSystem(c *gin.Context) {
	id := requireUintID(c, "id", "星系配置 ID")
	if id == 0 {
		return
	}
	if err := h.svc.DeleteAdminSystem(id); err != nil {
		if service.IsUserVisibleError(err) {
			response.Fail(c, response.CodeBizError, err.Error())
			return
		}
		response.Fail(c, response.CodeBizError, "删除星系配置失败")
		return
	}
	response.OK(c, nil)
}

func (h *GalaxyRegistryHandler) ListAdminEntries(c *gin.Context) {
	page, pageSize, err := parseLedgerPaginationQuery(c, 200)
	if err != nil {
		response.Fail(c, response.CodeParamError, err.Error())
		return
	}
	filter, err := parseGalaxyRegistryEntryFilter(c)
	if err != nil {
		response.Fail(c, response.CodeParamError, err.Error())
		return
	}
	if raw := strings.TrimSpace(c.Query("system_config_id")); raw != "" {
		value, convErr := parseRequiredUintQueryParam("system_config_id", raw)
		if convErr != nil {
			response.Fail(c, response.CodeParamError, "无效的 system_config_id")
			return
		}
		filter.SystemConfigID = &value
	}
	rows, total, err := h.svc.ListAdminEntries(filter, page, pageSize)
	if err != nil {
		response.Fail(c, response.CodeBizError, "获取登记列表失败")
		return
	}
	response.OKWithPage(c, rows, total, page, pageSize)
}

func (h *GalaxyRegistryHandler) ForceEndAdminEntry(c *gin.Context) {
	id := requireUintID(c, "id", "登记 ID")
	if id == 0 {
		return
	}
	row, err := h.svc.ForceEndEntryWithContext(c.Request.Context(), middleware.GetUserID(c), id)
	if err != nil {
		if service.IsUserVisibleError(err) {
			response.Fail(c, response.CodeBizError, err.Error())
			return
		}
		response.Fail(c, response.CodeBizError, "强制结束登记失败")
		return
	}
	response.OK(c, row)
}

func (h *GalaxyRegistryHandler) UpdateAdminEntryValidation(c *gin.Context) {
	id := requireUintID(c, "id", "登记 ID")
	if id == 0 {
		return
	}
	var req galaxyRegistryAdminUpdateValidationRequest
	if !bindJSON(c, &req) {
		return
	}
	row, err := h.svc.OverrideEntryValidation(id, strings.TrimSpace(req.ValidationStatus), req.ViolationReason)
	if err != nil {
		if service.IsUserVisibleError(err) {
			response.Fail(c, response.CodeBizError, err.Error())
			return
		}
		response.Fail(c, response.CodeBizError, "更新校验结果失败")
		return
	}
	response.OK(c, row)
}

func (h *GalaxyRegistryHandler) GetAdminAnalytics(c *gin.Context) {
	startDate, err := parseGalaxyRegistryOptionalDateParam(c.Query("start_date"), false)
	if err != nil {
		response.Fail(c, response.CodeParamError, "无效的 start_date")
		return
	}
	endDate, err := parseGalaxyRegistryOptionalDateParam(c.Query("end_date"), true)
	if err != nil {
		response.Fail(c, response.CodeParamError, "无效的 end_date")
		return
	}
	rows, err := h.svc.GetAdminAnalytics(startDate, endDate)
	if err != nil {
		response.Fail(c, response.CodeBizError, "获取分析面板失败")
		return
	}
	response.OK(c, rows)
}

func parseGalaxyRegistryEntryFilter(c *gin.Context) (service.GalaxyRegistryEntryListFilter, error) {
	startDateFrom, err := parseGalaxyRegistryOptionalDateParam(c.Query("start_date"), false)
	if err != nil {
		return service.GalaxyRegistryEntryListFilter{}, fmt.Errorf("无效的 start_date")
	}
	endDateTo, err := parseGalaxyRegistryOptionalDateParam(c.Query("end_date"), true)
	if err != nil {
		return service.GalaxyRegistryEntryListFilter{}, fmt.Errorf("无效的 end_date")
	}
	return service.GalaxyRegistryEntryListFilter{
		Keyword:          strings.TrimSpace(c.Query("keyword")),
		Status:           strings.TrimSpace(c.Query("status")),
		ValidationStatus: strings.TrimSpace(c.Query("validation_status")),
		StartDateFrom:    startDateFrom,
		EndDateTo:        endDateTo,
	}, nil
}

func parseGalaxyRegistryOptionalDateParam(raw string, endOfDay bool) (*time.Time, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, nil
	}
	parsed, err := parseGalaxyRegistryTime(raw, endOfDay)
	if err != nil {
		return nil, err
	}
	return &parsed, nil
}

func parseGalaxyRegistryTime(raw string, endOfDay bool) (time.Time, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return time.Time{}, fmt.Errorf("empty time")
	}
	layouts := []string{time.RFC3339, "2006-01-02 15:04:05", "2006-01-02"}
	for _, layout := range layouts {
		if parsed, err := time.ParseInLocation(layout, raw, time.Local); err == nil {
			if layout == "2006-01-02" && endOfDay {
				return parsed.Add(23*time.Hour + 59*time.Minute + 59*time.Second), nil
			}
			return parsed, nil
		}
	}
	return time.Time{}, fmt.Errorf("invalid time")
}

func modelGalaxyRegistryDefaultMinBountyAmount() float64 {
	return 10_000_000
}

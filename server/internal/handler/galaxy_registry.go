package handler

import (
	"context"
	"fmt"
	"math"
	"strings"
	"time"

	"amiya-eden/internal/apicontract"
	"amiya-eden/internal/middleware"
	"amiya-eden/internal/model"
	"amiya-eden/internal/service"
	"amiya-eden/pkg/response"

	"github.com/gin-gonic/gin"
)

type GalaxyRegistryHandler struct {
	svc *service.GalaxyRegistryService
}

func NewGalaxyRegistryHandler() *GalaxyRegistryHandler {
	return &GalaxyRegistryHandler{svc: service.NewGalaxyRegistryService()}
}

func galaxyRegistryMutationFailure(err error, fallback string) *contractOperationFailure {
	if err == nil {
		return nil
	}
	if service.IsUserVisibleError(err) {
		fallback = err.Error()
	}
	return &contractOperationFailure{response.CodeBizError, fallback}
}

func galaxyRegistryEntryResult(row *service.GalaxyRegistryEntryItem, err error, fallback string) (apicontract.GalaxyRegistryEntry, *contractOperationFailure) {
	if failure := galaxyRegistryMutationFailure(err, fallback); failure != nil {
		return apicontract.GalaxyRegistryEntry{}, failure
	}
	return galaxyRegistryEntryDTO(*row), nil
}

func (h *GalaxyRegistryHandler) createEntry(userID uint, req apicontract.GalaxyRegistryCreateEntryInput) (apicontract.GalaxyRegistryEntry, *contractOperationFailure) {
	expectedEndAt, err := parseGalaxyRegistryTime(req.ExpectedEndAt, false)
	if err != nil {
		return apicontract.GalaxyRegistryEntry{}, &contractOperationFailure{response.CodeParamError, "无效的 expected_end_at"}
	}
	row, err := h.svc.StartEntry(userID, service.GalaxyRegistryCreateEntryRequest{
		SystemConfigID: req.SystemConfigID,
		ExpectedEndAt:  expectedEndAt,
	})
	return galaxyRegistryEntryResult(row, err, "创建登记失败")
}

func (h *GalaxyRegistryHandler) updateMyExpectedEndAt(userID, entryID uint, req apicontract.GalaxyRegistryUpdateExpectedEndAtInput) (apicontract.GalaxyRegistryEntry, *contractOperationFailure) {
	expectedEndAt, err := parseGalaxyRegistryTime(req.ExpectedEndAt, false)
	if err != nil {
		return apicontract.GalaxyRegistryEntry{}, &contractOperationFailure{response.CodeParamError, "无效的 expected_end_at"}
	}
	row, err := h.svc.UpdateMyExpectedEndAt(userID, entryID, service.GalaxyRegistryUpdateExpectedEndAtRequest{ExpectedEndAt: expectedEndAt})
	return galaxyRegistryEntryResult(row, err, "更新预计结束时间失败")
}

func (h *GalaxyRegistryHandler) listMyEntries(userID uint, params apicontract.ListMyEntriesParams) (apicontract.GalaxyRegistryEntryPage, *contractOperationFailure) {
	page, pageSize := normalizeLedgerPagination(galaxyRegistryValue(params.Current), galaxyRegistryValue(params.Size))
	filter, err := parseGalaxyRegistryEntryFilter(params.Keyword, params.Status, params.ValidationStatus, params.StartDate, params.EndDate)
	if err != nil {
		return apicontract.GalaxyRegistryEntryPage{}, &contractOperationFailure{response.CodeParamError, err.Error()}
	}
	rows, total, err := h.svc.ListMyEntries(userID, filter, page, pageSize)
	if err != nil {
		return apicontract.GalaxyRegistryEntryPage{}, &contractOperationFailure{response.CodeBizError, "获取我的登记失败"}
	}
	return galaxyRegistryEntryPageDTO(rows, total, page, pageSize), nil
}

func (h *GalaxyRegistryHandler) createAdminSystem(req apicontract.GalaxyRegistryAdminCreateSystemInput) (apicontract.GalaxyRegistryAdminSystem, *contractOperationFailure) {
	minBountyAmount := modelGalaxyRegistryDefaultMinBountyAmount()
	if req.MinBountyAmount != nil {
		minBountyAmount = *req.MinBountyAmount
	}
	isEnabled := true
	if req.IsEnabled != nil {
		isEnabled = *req.IsEnabled
	}
	row, err := h.svc.CreateAdminSystem(req.SolarSystemID, req.Note, minBountyAmount, isEnabled)
	if failure := galaxyRegistryMutationFailure(err, "创建星系配置失败"); failure != nil {
		return apicontract.GalaxyRegistryAdminSystem{}, failure
	}
	return galaxyRegistryAdminSystemDTO(*row), nil
}

func (h *GalaxyRegistryHandler) updateAdminSystem(id uint, req apicontract.GalaxyRegistryAdminUpdateSystemInput) (apicontract.GalaxyRegistryAdminSystem, *contractOperationFailure) {
	row, err := h.svc.UpdateAdminSystem(id, req.Note, req.MinBountyAmount, req.IsEnabled)
	if failure := galaxyRegistryMutationFailure(err, "更新星系配置失败"); failure != nil {
		return apicontract.GalaxyRegistryAdminSystem{}, failure
	}
	return galaxyRegistryAdminSystemDTO(*row), nil
}

func (h *GalaxyRegistryHandler) listAdminEntries(params apicontract.ListAdminEntriesParams) (apicontract.GalaxyRegistryEntryPage, *contractOperationFailure) {
	page, pageSize := normalizeLedgerPagination(galaxyRegistryValue(params.Current), galaxyRegistryValue(params.Size))
	filter, err := parseGalaxyRegistryEntryFilter(params.Keyword, params.Status, params.ValidationStatus, params.StartDate, params.EndDate)
	if err != nil {
		return apicontract.GalaxyRegistryEntryPage{}, &contractOperationFailure{response.CodeParamError, err.Error()}
	}
	// A typed query parameter alone does not enforce the contract's uint32 range.
	if id := params.SystemConfigId; id != nil && (*id == 0 || *id > math.MaxUint32) {
		return apicontract.GalaxyRegistryEntryPage{}, &contractOperationFailure{response.CodeParamError, "无效的 system_config_id"}
	}
	filter.SystemConfigID = params.SystemConfigId
	rows, total, err := h.svc.ListAdminEntries(filter, page, pageSize)
	if err != nil {
		return apicontract.GalaxyRegistryEntryPage{}, &contractOperationFailure{response.CodeBizError, "获取登记列表失败"}
	}
	return galaxyRegistryEntryPageDTO(rows, total, page, pageSize), nil
}

func (h *GalaxyRegistryHandler) updateAdminEntryValidation(id uint, req apicontract.GalaxyRegistryAdminUpdateValidationInput) (apicontract.GalaxyRegistryEntry, *contractOperationFailure) {
	row, err := h.svc.OverrideEntryValidation(id, strings.TrimSpace(req.ValidationStatus), req.ViolationReason)
	return galaxyRegistryEntryResult(row, err, "更新校验结果失败")
}

func (h *GalaxyRegistryHandler) getAdminAnalytics(params apicontract.GetAdminAnalyticsParams) (apicontract.GalaxyRegistryAdminAnalytics, *contractOperationFailure) {
	startDate, err := parseGalaxyRegistryOptionalDateParam(galaxyRegistryValue(params.StartDate), false)
	if err != nil {
		return apicontract.GalaxyRegistryAdminAnalytics{}, &contractOperationFailure{response.CodeParamError, "无效的 start_date"}
	}
	endDate, err := parseGalaxyRegistryOptionalDateParam(galaxyRegistryValue(params.EndDate), true)
	if err != nil {
		return apicontract.GalaxyRegistryAdminAnalytics{}, &contractOperationFailure{response.CodeParamError, "无效的 end_date"}
	}
	rows, err := h.svc.GetAdminAnalytics(startDate, endDate)
	if err != nil {
		return apicontract.GalaxyRegistryAdminAnalytics{}, &contractOperationFailure{response.CodeBizError, "获取分析面板失败"}
	}
	return galaxyRegistryAnalyticsDTO(*rows), nil
}

func parseGalaxyRegistryEntryFilter(keyword, status, validationStatus, startDate, endDate *string) (service.GalaxyRegistryEntryListFilter, error) {
	startDateFrom, err := parseGalaxyRegistryOptionalDateParam(galaxyRegistryValue(startDate), false)
	if err != nil {
		return service.GalaxyRegistryEntryListFilter{}, fmt.Errorf("无效的 start_date")
	}
	endDateTo, err := parseGalaxyRegistryOptionalDateParam(galaxyRegistryValue(endDate), true)
	if err != nil {
		return service.GalaxyRegistryEntryListFilter{}, fmt.Errorf("无效的 end_date")
	}
	return service.GalaxyRegistryEntryListFilter{
		Keyword:          strings.TrimSpace(galaxyRegistryValue(keyword)),
		Status:           strings.TrimSpace(galaxyRegistryValue(status)),
		ValidationStatus: strings.TrimSpace(galaxyRegistryValue(validationStatus)),
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

func galaxyRegistryEncode[T any](success func(T) error, businessError func(apicontract.BusinessError) error, data T, failure *contractOperationFailure) error {
	if failure != nil {
		return businessError(contractBusinessError(failure.code, failure.message))
	}
	return success(data)
}

func galaxyRegistryMissingBody() apicontract.BusinessError {
	return contractBusinessError(response.CodeParamError, "请求参数错误")
}

func (h *GalaxyRegistryHandler) ListSystems(ctx context.Context, _ apicontract.ListSystemsRequestObject) (apicontract.ListSystemsResponseObject, error) {
	out := &apicontract.ListSystems200JSONResponse{}
	rows, err := h.svc.ListSystems(middleware.GetUserID(ctx.(*gin.Context)))
	if err != nil {
		return out, out.FromBusinessError(contractBusinessError(response.CodeBizError, "获取星系列表失败"))
	}
	return out, out.FromGalaxyRegistrySystemsSuccess(apicontract.GalaxyRegistrySystemsSuccess{Code: response.CodeOK, Msg: "success", Data: galaxyRegistrySystemsDTO(*rows)})
}

func (h *GalaxyRegistryHandler) CreateEntry(ctx context.Context, request apicontract.CreateEntryRequestObject) (apicontract.CreateEntryResponseObject, error) {
	out := &apicontract.CreateEntry200JSONResponse{}
	if request.Body == nil {
		return out, out.FromBusinessError(galaxyRegistryMissingBody())
	}
	data, failure := h.createEntry(middleware.GetUserID(ctx.(*gin.Context)), *request.Body)
	return out, galaxyRegistryEncode(out.FromGalaxyRegistryEntrySuccess, out.FromBusinessError, apicontract.GalaxyRegistryEntrySuccess{Code: response.CodeOK, Msg: "success", Data: data}, failure)
}

func (h *GalaxyRegistryHandler) EndMyEntry(ctx context.Context, request apicontract.EndMyEntryRequestObject) (apicontract.EndMyEntryResponseObject, error) {
	c := ctx.(*gin.Context)
	row, err := h.svc.EndMyEntryWithContext(c.Request.Context(), middleware.GetUserID(c), request.Id)
	data, failure := galaxyRegistryEntryResult(row, err, "结束登记失败")
	out := &apicontract.EndMyEntry200JSONResponse{}
	return out, galaxyRegistryEncode(out.FromGalaxyRegistryEntrySuccess, out.FromBusinessError, apicontract.GalaxyRegistryEntrySuccess{Code: response.CodeOK, Msg: "success", Data: data}, failure)
}

func (h *GalaxyRegistryHandler) UpdateMyExpectedEndAt(ctx context.Context, request apicontract.UpdateMyExpectedEndAtRequestObject) (apicontract.UpdateMyExpectedEndAtResponseObject, error) {
	out := &apicontract.UpdateMyExpectedEndAt200JSONResponse{}
	if request.Body == nil {
		return out, out.FromBusinessError(galaxyRegistryMissingBody())
	}
	data, failure := h.updateMyExpectedEndAt(middleware.GetUserID(ctx.(*gin.Context)), request.Id, *request.Body)
	return out, galaxyRegistryEncode(out.FromGalaxyRegistryEntrySuccess, out.FromBusinessError, apicontract.GalaxyRegistryEntrySuccess{Code: response.CodeOK, Msg: "success", Data: data}, failure)
}

func (h *GalaxyRegistryHandler) ListMyEntries(ctx context.Context, request apicontract.ListMyEntriesRequestObject) (apicontract.ListMyEntriesResponseObject, error) {
	data, failure := h.listMyEntries(middleware.GetUserID(ctx.(*gin.Context)), request.Params)
	out := &apicontract.ListMyEntries200JSONResponse{}
	return out, galaxyRegistryEncode(out.FromGalaxyRegistryEntryPageSuccess, out.FromBusinessError, apicontract.GalaxyRegistryEntryPageSuccess{Code: response.CodeOK, Msg: "success", Data: data}, failure)
}

func (h *GalaxyRegistryHandler) SearchAdminSdeSystems(_ context.Context, request apicontract.SearchAdminSdeSystemsRequestObject) (apicontract.SearchAdminSdeSystemsResponseObject, error) {
	out := &apicontract.SearchAdminSdeSystems200JSONResponse{}
	rows, err := h.svc.SearchAdminSdeSystems(galaxyRegistryValue(request.Params.Keyword), galaxyRegistryValue(request.Params.Limit))
	if err != nil {
		return out, out.FromBusinessError(contractBusinessError(response.CodeBizError, "搜索星系失败"))
	}
	return out, out.FromGalaxyRegistrySdeSystemsSuccess(apicontract.GalaxyRegistrySdeSystemsSuccess{Code: response.CodeOK, Msg: "success", Data: fleetDTOs(rows, galaxyRegistrySdeSystemDTO)})
}

func (h *GalaxyRegistryHandler) ListAdminSystems(_ context.Context, _ apicontract.ListAdminSystemsRequestObject) (apicontract.ListAdminSystemsResponseObject, error) {
	out := &apicontract.ListAdminSystems200JSONResponse{}
	rows, err := h.svc.ListAdminSystems()
	if err != nil {
		return out, out.FromBusinessError(contractBusinessError(response.CodeBizError, "获取星系配置失败"))
	}
	return out, out.FromGalaxyRegistryAdminSystemsSuccess(apicontract.GalaxyRegistryAdminSystemsSuccess{Code: response.CodeOK, Msg: "success", Data: fleetDTOs(rows, galaxyRegistryAdminSystemDTO)})
}

func (h *GalaxyRegistryHandler) CreateAdminSystem(_ context.Context, request apicontract.CreateAdminSystemRequestObject) (apicontract.CreateAdminSystemResponseObject, error) {
	out := &apicontract.CreateAdminSystem200JSONResponse{}
	if request.Body == nil {
		return out, out.FromBusinessError(galaxyRegistryMissingBody())
	}
	data, failure := h.createAdminSystem(*request.Body)
	return out, galaxyRegistryEncode(out.FromGalaxyRegistryAdminSystemSuccess, out.FromBusinessError, apicontract.GalaxyRegistryAdminSystemSuccess{Code: response.CodeOK, Msg: "success", Data: data}, failure)
}

func (h *GalaxyRegistryHandler) UpdateAdminSystem(_ context.Context, request apicontract.UpdateAdminSystemRequestObject) (apicontract.UpdateAdminSystemResponseObject, error) {
	out := &apicontract.UpdateAdminSystem200JSONResponse{}
	if request.Body == nil {
		return out, out.FromBusinessError(galaxyRegistryMissingBody())
	}
	data, failure := h.updateAdminSystem(request.Id, *request.Body)
	return out, galaxyRegistryEncode(out.FromGalaxyRegistryAdminSystemSuccess, out.FromBusinessError, apicontract.GalaxyRegistryAdminSystemSuccess{Code: response.CodeOK, Msg: "success", Data: data}, failure)
}

func (h *GalaxyRegistryHandler) DeleteAdminSystem(_ context.Context, request apicontract.DeleteAdminSystemRequestObject) (apicontract.DeleteAdminSystemResponseObject, error) {
	failure := galaxyRegistryMutationFailure(h.svc.DeleteAdminSystem(request.Id), "删除星系配置失败")
	out := &apicontract.DeleteAdminSystem200JSONResponse{}
	return out, galaxyRegistryEncode(out.FromNullSuccess, out.FromBusinessError, apicontract.NullSuccess{Code: response.CodeOK, Msg: "success"}, failure)
}

func (h *GalaxyRegistryHandler) ListAdminEntries(_ context.Context, request apicontract.ListAdminEntriesRequestObject) (apicontract.ListAdminEntriesResponseObject, error) {
	data, failure := h.listAdminEntries(request.Params)
	out := &apicontract.ListAdminEntries200JSONResponse{}
	return out, galaxyRegistryEncode(out.FromGalaxyRegistryEntryPageSuccess, out.FromBusinessError, apicontract.GalaxyRegistryEntryPageSuccess{Code: response.CodeOK, Msg: "success", Data: data}, failure)
}

func (h *GalaxyRegistryHandler) ForceEndAdminEntry(ctx context.Context, request apicontract.ForceEndAdminEntryRequestObject) (apicontract.ForceEndAdminEntryResponseObject, error) {
	c := ctx.(*gin.Context)
	row, err := h.svc.ForceEndEntryWithContext(c.Request.Context(), middleware.GetUserID(c), request.Id)
	data, failure := galaxyRegistryEntryResult(row, err, "强制结束登记失败")
	out := &apicontract.ForceEndAdminEntry200JSONResponse{}
	return out, galaxyRegistryEncode(out.FromGalaxyRegistryEntrySuccess, out.FromBusinessError, apicontract.GalaxyRegistryEntrySuccess{Code: response.CodeOK, Msg: "success", Data: data}, failure)
}

func (h *GalaxyRegistryHandler) RevalidateAdminEntry(ctx context.Context, request apicontract.RevalidateAdminEntryRequestObject) (apicontract.RevalidateAdminEntryResponseObject, error) {
	c := ctx.(*gin.Context)
	row, err := h.svc.RevalidateEntryWithContext(c.Request.Context(), request.Id)
	data, failure := galaxyRegistryEntryResult(row, err, "重新校验登记失败")
	out := &apicontract.RevalidateAdminEntry200JSONResponse{}
	return out, galaxyRegistryEncode(out.FromGalaxyRegistryEntrySuccess, out.FromBusinessError, apicontract.GalaxyRegistryEntrySuccess{Code: response.CodeOK, Msg: "success", Data: data}, failure)
}

func (h *GalaxyRegistryHandler) UpdateAdminEntryValidation(_ context.Context, request apicontract.UpdateAdminEntryValidationRequestObject) (apicontract.UpdateAdminEntryValidationResponseObject, error) {
	out := &apicontract.UpdateAdminEntryValidation200JSONResponse{}
	if request.Body == nil {
		return out, out.FromBusinessError(galaxyRegistryMissingBody())
	}
	data, failure := h.updateAdminEntryValidation(request.Id, *request.Body)
	return out, galaxyRegistryEncode(out.FromGalaxyRegistryEntrySuccess, out.FromBusinessError, apicontract.GalaxyRegistryEntrySuccess{Code: response.CodeOK, Msg: "success", Data: data}, failure)
}

func (h *GalaxyRegistryHandler) GetAdminAnalytics(_ context.Context, request apicontract.GetAdminAnalyticsRequestObject) (apicontract.GetAdminAnalyticsResponseObject, error) {
	data, failure := h.getAdminAnalytics(request.Params)
	out := &apicontract.GetAdminAnalytics200JSONResponse{}
	return out, galaxyRegistryEncode(out.FromGalaxyRegistryAdminAnalyticsSuccess, out.FromBusinessError, apicontract.GalaxyRegistryAdminAnalyticsSuccess{Code: response.CodeOK, Msg: "success", Data: data}, failure)
}

func galaxyRegistryValue[T any](p *T) T {
	if p != nil {
		return *p
	}
	var zero T
	return zero
}

func galaxyRegistrySystemsDTO(s service.GalaxyRegistrySystemsResponse) apicontract.GalaxyRegistrySystems {
	return apicontract.GalaxyRegistrySystems{
		Summary: apicontract.GalaxyRegistrySystemSummary(s.Summary),
		Items:   dashboardDTOs(s.Items, galaxyRegistrySystemDTO),
	}
}

func galaxyRegistrySystemDTO(s service.GalaxyRegistrySystemItem) apicontract.GalaxyRegistrySystemItem {
	item := apicontract.GalaxyRegistrySystemItem{
		ID: s.ID, SolarSystemID: s.SolarSystemID, SolarSystemName: s.SolarSystemName,
		RegionID: s.RegionID, RegionName: s.RegionName,
		ConstellationID: s.ConstellationID, ConstellationName: s.ConstellationName,
		Security: s.Security, Note: s.Note, MinBountyAmount: s.MinBountyAmount, IsEnabled: s.IsEnabled, Status: s.Status,
	}
	if s.ActiveEntry != nil {
		active := apicontract.GalaxyRegistrySystemActiveEntry(*s.ActiveEntry)
		item.ActiveEntry = &active
	}
	return item
}

func galaxyRegistryEntryDTO(e service.GalaxyRegistryEntryItem) apicontract.GalaxyRegistryEntry {
	return apicontract.GalaxyRegistryEntry(e)
}

func galaxyRegistryEntryPageDTO(rows []service.GalaxyRegistryEntryItem, total int64, page, pageSize int) apicontract.GalaxyRegistryEntryPage {
	return apicontract.GalaxyRegistryEntryPage{
		List: dashboardDTOs(rows, galaxyRegistryEntryDTO), Total: total, Page: page, PageSize: pageSize,
	}
}

func galaxyRegistrySdeSystemDTO(s model.GalaxyRegistrySdeSystem) apicontract.GalaxyRegistrySdeSystem {
	return apicontract.GalaxyRegistrySdeSystem(s)
}

func galaxyRegistryAdminSystemDTO(s model.GalaxyRegistrySystem) apicontract.GalaxyRegistryAdminSystem {
	return apicontract.GalaxyRegistryAdminSystem{
		ID: s.ID, CreatedAt: s.CreatedAt, UpdatedAt: s.UpdatedAt, SolarSystemID: s.SolarSystemID,
		SolarSystemName: s.SolarSystemName, RegionID: s.RegionID, RegionName: s.RegionName,
		ConstellationID: s.ConstellationID, ConstellationName: s.ConstellationName,
		Security: s.Security, Note: s.Note, MinBountyAmount: s.MinBountyAmount, IsEnabled: s.IsEnabled,
	}
}

func galaxyRegistryAnalyticsDTO(a service.GalaxyRegistryAdminAnalytics) apicontract.GalaxyRegistryAdminAnalytics {
	return apicontract.GalaxyRegistryAdminAnalytics{
		RangeStart: a.RangeStart, RangeEnd: a.RangeEnd,
		CurrentSnapshot: apicontract.GalaxyRegistryCurrentSnapshot(a.CurrentSnapshot),
		Recent7d:        apicontract.GalaxyRegistryPeriodSummary(a.Recent7D),
		Recent30d:       apicontract.GalaxyRegistryPeriodSummary(a.Recent30D),
		TopSystems: dashboardDTOs(a.TopSystems, func(row model.GalaxyRegistryTopSystemStat) apicontract.GalaxyRegistryTopSystemStat {
			return apicontract.GalaxyRegistryTopSystemStat(row)
		}),
		RecentViolations: dashboardDTOs(a.RecentViolations, galaxyRegistryEntryDTO),
	}
}

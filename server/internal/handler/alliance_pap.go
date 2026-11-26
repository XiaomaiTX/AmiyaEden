package handler

import (
	"amiya-eden/global"
	"amiya-eden/internal/apicontract"
	"amiya-eden/internal/model"
	"amiya-eden/internal/service"
	"amiya-eden/internal/utils"
	"amiya-eden/pkg/response"
	"context"
	"time"
)

// AlliancePAPHandler 联盟 PAP HTTP 处理器
type AlliancePAPHandler struct {
	svc *service.AlliancePAPService
}

func NewAlliancePAPHandler() *AlliancePAPHandler {
	return &AlliancePAPHandler{svc: service.NewAlliancePAPService()}
}

func (h *AlliancePAPHandler) GetMyAlliancePAP(ctx context.Context, request apicontract.GetMyAlliancePAPRequestObject) (apicontract.GetMyAlliancePAPResponseObject, error) {
	out := &apicontract.GetMyAlliancePAP200JSONResponse{}
	userID, _ := fleetStrictActor(ctx)
	data, failure := h.getMyAlliancePAP(userID, request.Params)
	if failure != nil {
		return out, out.FromBusinessError(failure.contract())
	}
	return out, out.FromFleetAlliancePAPDataSuccess(apicontract.FleetAlliancePAPDataSuccess{Code: response.CodeOK, Msg: "success", Data: data})
}

func (h *AlliancePAPHandler) GetAllAlliancePAP(ctx context.Context, request apicontract.GetAllAlliancePAPRequestObject) (apicontract.GetAllAlliancePAPResponseObject, error) {
	out := &apicontract.GetAllAlliancePAP200JSONResponse{}
	_, roles := fleetStrictActor(ctx)
	data, failure := h.getAllAlliancePAP(roles, request.Params)
	if failure != nil {
		return out, out.FromBusinessError(failure.contract())
	}
	return out, out.FromAdminAlliancePAPPageSuccess(apicontract.AdminAlliancePAPPageSuccess{Code: response.CodeOK, Msg: "success", Data: data})
}

func (h *AlliancePAPHandler) TriggerFetch(ctx context.Context, request apicontract.TriggerFetchRequestObject) (apicontract.TriggerFetchResponseObject, error) {
	out := &apicontract.TriggerFetch200JSONResponse{}
	data := h.triggerAlliancePAPFetch(request.Body, apicontract.StrictGinJSONDecodeError(ctx))
	return out, out.FromAdminAlliancePAPMessageSuccess(apicontract.AdminAlliancePAPMessageSuccess{Code: response.CodeOK, Msg: "success", Data: data})
}

func (h *AlliancePAPHandler) ImportAlliancePAP(_ context.Context, request apicontract.ImportAlliancePAPRequestObject) (apicontract.ImportAlliancePAPResponseObject, error) {
	out := &apicontract.ImportAlliancePAP200JSONResponse{}
	if request.Body == nil {
		return out, out.FromBusinessError(contractBusinessError(response.CodeParamError, "请求参数错误"))
	}
	data, failure := h.importAlliancePAP(*request.Body)
	if failure != nil {
		return out, out.FromBusinessError(failure.contract())
	}
	return out, out.FromAdminAlliancePAPMessageSuccess(apicontract.AdminAlliancePAPMessageSuccess{Code: response.CodeOK, Msg: "success", Data: data})
}

func (h *AlliancePAPHandler) SettleMonth(ctx context.Context, request apicontract.SettleMonthRequestObject) (apicontract.SettleMonthResponseObject, error) {
	out := &apicontract.SettleMonth200JSONResponse{}
	if request.Body == nil {
		return out, out.FromBusinessError(contractBusinessError(response.CodeParamError, "请求参数错误"))
	}
	_, roles := fleetStrictActor(ctx)
	data, failure := h.settleMonth(roles, *request.Body)
	if failure != nil {
		return out, out.FromBusinessError(failure.contract())
	}
	return out, out.FromAdminAlliancePAPSettledMonthSuccess(apicontract.AdminAlliancePAPSettledMonthSuccess{Code: response.CodeOK, Msg: "success", Data: data})
}

// A super admin can operate on all corporations; other roles are scoped to the allowlist.
func getAllowCorpFilter(roles []string) []int64 {
	if model.IsSuperAdmin(roles) {
		return nil
	}
	return utils.GetAllowCorporations()
}

func (h *AlliancePAPHandler) getMyAlliancePAP(userID uint, params apicontract.GetMyAlliancePAPParams) (apicontract.FleetAlliancePAPData, *fleetOperationFailure) {
	now := time.Now()
	year, month := fleetValueOr(params.Year, now.Year()), fleetValueOr(params.Month, int(now.Month()))
	result, err := h.svc.GetMyPAPByUserID(userID, year, month)
	if err != nil {
		return apicontract.FleetAlliancePAPData{}, &fleetOperationFailure{code: response.CodeBizError, message: err.Error()}
	}
	return fleetAlliancePAPDTO(*result), nil
}

func (h *AlliancePAPHandler) getAllAlliancePAP(roles []string, params apicontract.GetAllAlliancePAPParams) (apicontract.AdminAlliancePAPPage, *fleetOperationFailure) {
	now := time.Now()
	year, month := fleetValueOr(params.Year, now.Year()), fleetValueOr(params.Month, int(now.Month()))
	page, pageSize := normalizePage(fleetValue(params.Current)), fleetValue(params.Size)
	if pageSize < 1 {
		pageSize = 20
	}
	pageSize = normalizeLedgerPageSize(pageSize)
	list, total, err := h.svc.GetAllPAPPaged(year, month, page, pageSize, getAllowCorpFilter(roles))
	if err != nil {
		return apicontract.AdminAlliancePAPPage{}, &fleetOperationFailure{code: response.CodeBizError, message: err.Error()}
	}
	return apicontract.AdminAlliancePAPPage{
		List: fleetNullableDTOs(list, alliancePAPSummaryDTO), Page: page, PageSize: pageSize, Total: total,
	}, nil
}

func alliancePAPFetchMonth(now time.Time, req *apicontract.AdminAlliancePAPFetchInput, bindingErr error) (int, int) {
	if bindingErr == nil && req != nil && req.Year != 0 && req.Month >= 1 && req.Month <= 12 {
		return req.Year, req.Month
	}
	return now.Year(), int(now.Month())
}

func (h *AlliancePAPHandler) triggerAlliancePAPFetch(req *apicontract.AdminAlliancePAPFetchInput, bindingErr error) apicontract.AdminAlliancePAPMessage {
	year, month := alliancePAPFetchMonth(time.Now(), req, bindingErr)
	go func() {
		global.Logger.Info("手动触发联盟 PAP 拉取")
		h.svc.FetchAllUsers(year, month)
	}()
	return apicontract.AdminAlliancePAPMessage{Message: "已触发后台拉取任务"}
}

func (h *AlliancePAPHandler) importAlliancePAP(req apicontract.AdminAlliancePAPImportInput) (apicontract.AdminAlliancePAPMessage, *fleetOperationFailure) {
	if req.PAPImportInfo.CalculatedAt == "" {
		return apicontract.AdminAlliancePAPMessage{}, &fleetOperationFailure{code: response.CodeParamError, message: "请求参数错误: 缺少数据时间"}
	}
	if err := h.svc.ImportAlliancePAPByPrimaryCharacter(req.Year, req.Month, &service.PAPImportInfo{
		PrimaryCharacterName: req.PAPImportInfo.PrimaryCharacterName, MonthlyPAP: req.PAPImportInfo.MonthlyPAP,
		CalculatedAt: req.PAPImportInfo.CalculatedAt,
	}); err != nil {
		return apicontract.AdminAlliancePAPMessage{}, &fleetOperationFailure{code: response.CodeBizError, message: err.Error()}
	}
	return apicontract.AdminAlliancePAPMessage{Message: "导入成功"}, nil
}

func (h *AlliancePAPHandler) settleMonth(roles []string, req apicontract.AdminAlliancePAPSettleInput) (apicontract.AdminAlliancePAPSettledMonth, *fleetOperationFailure) {
	result, err := h.svc.SettleMonth(req.Year, req.Month, getAllowCorpFilter(roles))
	if err != nil {
		return apicontract.AdminAlliancePAPSettledMonth{}, &fleetOperationFailure{code: response.CodeBizError, message: err.Error()}
	}
	return apicontract.AdminAlliancePAPSettledMonth{Year: result.Year, Month: result.Month}, nil
}

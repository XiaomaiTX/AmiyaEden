package handler

import (
	"amiya-eden/internal/apicontract"
	"amiya-eden/internal/service"
	"amiya-eden/pkg/response"
	"context"
)

// FleetConfigHandler 舰队配置 HTTP 处理器
type FleetConfigHandler struct {
	svc *service.FleetConfigService
}

func NewFleetConfigHandler() *FleetConfigHandler {
	return &FleetConfigHandler{
		svc: service.NewFleetConfigService(),
	}
}

// Preserve the binding type names used by Gin's detailed parameter errors.
type (
	CreateFleetConfigRequest          apicontract.FleetConfigCreateInput
	UpdateFleetConfigRequest          apicontract.FleetConfigUpdateInput
	ImportFittingRequest              apicontract.FleetConfigImportFittingInput
	ExportToESIRequest                apicontract.FleetConfigExportESIInput
	UpdateFittingItemsSettingsRequest apicontract.FleetConfigUpdateItemSettingsInput
)

func (h *FleetConfigHandler) CreateFleetConfig(ctx context.Context, request apicontract.CreateFleetConfigRequestObject) (apicontract.CreateFleetConfigResponseObject, error) {
	out := &apicontract.CreateFleetConfig200JSONResponse{}
	if request.Body == nil {
		return out, out.FromBusinessError(contractBusinessError(response.CodeParamError, "请求参数错误"))
	}
	userID, _ := fleetStrictActor(ctx)
	data, failure := h.createFleetConfig(userID, *request.Body)
	if failure != nil {
		return out, out.FromBusinessError(failure.contract())
	}
	return out, out.FromFleetConfigRecordSuccess(apicontract.FleetConfigRecordSuccess{Code: response.CodeOK, Msg: "success", Data: data})
}

func (h *FleetConfigHandler) ListFleetConfigs(_ context.Context, request apicontract.ListFleetConfigsRequestObject) (apicontract.ListFleetConfigsResponseObject, error) {
	out := &apicontract.ListFleetConfigs200JSONResponse{}
	data, failure := h.listFleetConfigs(request.Params)
	if failure != nil {
		return out, out.FromBusinessError(failure.contract())
	}
	return out, out.FromFleetConfigPageSuccess(apicontract.FleetConfigPageSuccess{Code: response.CodeOK, Msg: "success", Data: data})
}

func (h *FleetConfigHandler) GetFleetConfig(_ context.Context, request apicontract.GetFleetConfigRequestObject) (apicontract.GetFleetConfigResponseObject, error) {
	out := &apicontract.GetFleetConfig200JSONResponse{}
	if !fleetValidUintID(request.Id) {
		return out, out.FromBusinessError(contractBusinessError(response.CodeParamError, "无效的配置ID"))
	}
	data, failure := h.getFleetConfig(request.Id)
	if failure != nil {
		return out, out.FromBusinessError(failure.contract())
	}
	return out, out.FromFleetConfigRecordSuccess(apicontract.FleetConfigRecordSuccess{Code: response.CodeOK, Msg: "success", Data: data})
}

func (h *FleetConfigHandler) UpdateFleetConfig(ctx context.Context, request apicontract.UpdateFleetConfigRequestObject) (apicontract.UpdateFleetConfigResponseObject, error) {
	out := &apicontract.UpdateFleetConfig200JSONResponse{}
	if !fleetValidUintID(request.Id) {
		return out, out.FromBusinessError(contractBusinessError(response.CodeParamError, "无效的配置ID"))
	}
	if request.Body == nil {
		return out, out.FromBusinessError(contractBusinessError(response.CodeParamError, "请求参数错误"))
	}
	userID, roles := fleetStrictActor(ctx)
	data, failure := h.updateFleetConfig(request.Id, userID, roles, *request.Body)
	if failure != nil {
		return out, out.FromBusinessError(failure.contract())
	}
	return out, out.FromFleetConfigRecordSuccess(apicontract.FleetConfigRecordSuccess{Code: response.CodeOK, Msg: "success", Data: data})
}

func (h *FleetConfigHandler) DeleteFleetConfig(ctx context.Context, request apicontract.DeleteFleetConfigRequestObject) (apicontract.DeleteFleetConfigResponseObject, error) {
	out := &apicontract.DeleteFleetConfig200JSONResponse{}
	if !fleetValidUintID(request.Id) {
		return out, out.FromBusinessError(contractBusinessError(response.CodeParamError, "无效的配置ID"))
	}
	userID, roles := fleetStrictActor(ctx)
	if failure := h.deleteFleetConfig(request.Id, userID, roles); failure != nil {
		return out, out.FromBusinessError(failure.contract())
	}
	return out, out.FromNullSuccess(apicontract.NullSuccess{Code: response.CodeOK, Msg: "success"})
}

func (h *FleetConfigHandler) ImportFromUserFitting(ctx context.Context, request apicontract.ImportFromUserFittingRequestObject) (apicontract.ImportFromUserFittingResponseObject, error) {
	out := &apicontract.ImportFromUserFitting200JSONResponse{}
	if request.Body == nil {
		return out, out.FromBusinessError(contractBusinessError(response.CodeParamError, "请求参数错误"))
	}
	userID, _ := fleetStrictActor(ctx)
	data, failure := h.importFromUserFitting(userID, *request.Body)
	if failure != nil {
		return out, out.FromBusinessError(failure.contract())
	}
	return out, out.FromFleetConfigImportedFittingSuccess(apicontract.FleetConfigImportedFittingSuccess{Code: response.CodeOK, Msg: "success", Data: data})
}

func (h *FleetConfigHandler) ExportToESI(ctx context.Context, request apicontract.ExportToESIRequestObject) (apicontract.ExportToESIResponseObject, error) {
	out := &apicontract.ExportToESI200JSONResponse{}
	if request.Body == nil {
		return out, out.FromBusinessError(contractBusinessError(response.CodeParamError, "请求参数错误"))
	}
	userID, _ := fleetStrictActor(ctx)
	if failure := h.exportToESI(userID, *request.Body); failure != nil {
		return out, out.FromBusinessError(failure.contract())
	}
	return out, out.FromNullSuccess(apicontract.NullSuccess{Code: response.CodeOK, Msg: "success"})
}

func (h *FleetConfigHandler) GetFittingEFT(_ context.Context, request apicontract.GetFittingEFTRequestObject) (apicontract.GetFittingEFTResponseObject, error) {
	out := &apicontract.GetFittingEFT200JSONResponse{}
	if !fleetValidUintID(request.Id) {
		return out, out.FromBusinessError(contractBusinessError(response.CodeParamError, "无效的配置ID"))
	}
	data, failure := h.getFittingEFT(request.Id, request.Params)
	if failure != nil {
		return out, out.FromBusinessError(failure.contract())
	}
	return out, out.FromFleetConfigEFTDataSuccess(apicontract.FleetConfigEFTDataSuccess{Code: response.CodeOK, Msg: "success", Data: data})
}

func (h *FleetConfigHandler) GetFittingItems(_ context.Context, request apicontract.GetFittingItemsRequestObject) (apicontract.GetFittingItemsResponseObject, error) {
	out := &apicontract.GetFittingItems200JSONResponse{}
	if !fleetValidUintID(request.Id) {
		return out, out.FromBusinessError(contractBusinessError(response.CodeParamError, "无效的配置ID"))
	}
	if !fleetValidUintID(request.FittingId) {
		return out, out.FromBusinessError(contractBusinessError(response.CodeParamError, "无效的装配ID"))
	}
	data, failure := h.getFittingItems(request.Id, request.FittingId, request.Params)
	if failure != nil {
		return out, out.FromBusinessError(failure.contract())
	}
	return out, out.FromFleetConfigFittingItemsSuccess(apicontract.FleetConfigFittingItemsSuccess{Code: response.CodeOK, Msg: "success", Data: data})
}

func (h *FleetConfigHandler) UpdateFittingItemsSettings(ctx context.Context, request apicontract.UpdateFittingItemsSettingsRequestObject) (apicontract.UpdateFittingItemsSettingsResponseObject, error) {
	out := &apicontract.UpdateFittingItemsSettings200JSONResponse{}
	if !fleetValidUintID(request.Id) {
		return out, out.FromBusinessError(contractBusinessError(response.CodeParamError, "无效的配置ID"))
	}
	if !fleetValidUintID(request.FittingId) {
		return out, out.FromBusinessError(contractBusinessError(response.CodeParamError, "无效的装配ID"))
	}
	if request.Body == nil {
		return out, out.FromBusinessError(contractBusinessError(response.CodeParamError, "请求参数错误"))
	}
	userID, roles := fleetStrictActor(ctx)
	if failure := h.updateFittingItemsSettings(request.Id, request.FittingId, userID, roles, *request.Body); failure != nil {
		return out, out.FromBusinessError(failure.contract())
	}
	return out, out.FromNullSuccess(apicontract.NullSuccess{Code: response.CodeOK, Msg: "success"})
}

func (h *FleetConfigHandler) createFleetConfig(userID uint, req apicontract.FleetConfigCreateInput) (apicontract.FleetConfigRecord, *fleetOperationFailure) {
	result, err := h.svc.CreateFleetConfig(userID, &service.CreateFleetConfigRequest{
		Name: req.Name, Description: req.Description, Fittings: fleetConfigFittingInputs(req.Fittings),
	})
	if err != nil {
		return apicontract.FleetConfigRecord{}, &fleetOperationFailure{code: response.CodeBizError, message: err.Error()}
	}
	return fleetConfigRecordDTO(*result), nil
}

func (h *FleetConfigHandler) listFleetConfigs(params apicontract.ListFleetConfigsParams) (apicontract.FleetConfigPage, *fleetOperationFailure) {
	page, pageSize := normalizePagination(fleetValue(params.Current), fleetValue(params.Size), 20, 100)
	records, total, err := h.svc.ListFleetConfigs(page, pageSize)
	if err != nil {
		return apicontract.FleetConfigPage{}, &fleetOperationFailure{code: response.CodeBizError, message: err.Error()}
	}
	return apicontract.FleetConfigPage{
		List: fleetDTOs(records, fleetConfigRecordDTO), Page: page, PageSize: pageSize, Total: total,
	}, nil
}

func (h *FleetConfigHandler) getFleetConfig(id uint) (apicontract.FleetConfigRecord, *fleetOperationFailure) {
	result, err := h.svc.GetFleetConfig(id)
	if err != nil {
		return apicontract.FleetConfigRecord{}, &fleetOperationFailure{code: response.CodeNotFound, message: err.Error()}
	}
	return fleetConfigRecordDTO(*result), nil
}

func (h *FleetConfigHandler) updateFleetConfig(id, userID uint, userRoles []string, req apicontract.FleetConfigUpdateInput) (apicontract.FleetConfigRecord, *fleetOperationFailure) {
	result, err := h.svc.UpdateFleetConfig(id, userID, userRoles, &service.UpdateFleetConfigRequest{
		Name: req.Name, Description: req.Description, Fittings: fleetConfigOptionalFittingInputs(req.Fittings),
	})
	if err != nil {
		return apicontract.FleetConfigRecord{}, &fleetOperationFailure{code: response.CodeBizError, message: err.Error()}
	}
	return fleetConfigRecordDTO(*result), nil
}

func (h *FleetConfigHandler) deleteFleetConfig(id, userID uint, userRoles []string) *fleetOperationFailure {
	if err := h.svc.DeleteFleetConfig(id, userID, userRoles); err != nil {
		return &fleetOperationFailure{code: response.CodeBizError, message: err.Error()}
	}
	return nil
}

func (h *FleetConfigHandler) importFromUserFitting(userID uint, req apicontract.FleetConfigImportFittingInput) (apicontract.FleetConfigImportedFitting, *fleetOperationFailure) {
	result, err := h.svc.ImportFromUserFitting(userID, &service.ImportFittingRequest{
		CharacterID: req.CharacterID, FittingID: req.FittingID,
	})
	if err != nil {
		return apicontract.FleetConfigImportedFitting{}, &fleetOperationFailure{code: response.CodeBizError, message: err.Error()}
	}
	return apicontract.FleetConfigImportedFitting{FittingName: result.FittingName, EFT: result.EFT, SrpAmount: result.SrpAmount}, nil
}

func (h *FleetConfigHandler) exportToESI(userID uint, req apicontract.FleetConfigExportESIInput) *fleetOperationFailure {
	if err := h.svc.ExportToESI(userID, &service.ExportToESIRequest{
		CharacterID: req.CharacterID, FleetConfigID: req.FleetConfigID, FittingItemID: req.FittingItemID,
	}); err != nil {
		return &fleetOperationFailure{code: response.CodeBizError, message: err.Error()}
	}
	return nil
}

func (h *FleetConfigHandler) getFittingEFT(id uint, params apicontract.GetFittingEFTParams) (apicontract.FleetConfigEFTData, *fleetOperationFailure) {
	result, err := h.svc.GetFittingEFT(id, fleetValueOr(params.Lang, "zh"))
	if err != nil {
		return apicontract.FleetConfigEFTData{}, &fleetOperationFailure{code: response.CodeBizError, message: err.Error()}
	}
	return fleetConfigEFTDTO(*result), nil
}

func (h *FleetConfigHandler) getFittingItems(configID, fittingID uint, params apicontract.GetFittingItemsParams) (apicontract.FleetConfigFittingItems, *fleetOperationFailure) {
	result, err := h.svc.GetFittingItems(configID, fittingID, fleetValueOr(params.Lang, "zh"))
	if err != nil {
		return apicontract.FleetConfigFittingItems{}, &fleetOperationFailure{code: response.CodeBizError, message: err.Error()}
	}
	return fleetConfigFittingItemsDTO(*result), nil
}

func (h *FleetConfigHandler) updateFittingItemsSettings(configID, fittingID, userID uint, userRoles []string, req apicontract.FleetConfigUpdateItemSettingsInput) *fleetOperationFailure {
	if err := h.svc.UpdateFittingItemsSettings(configID, fittingID, userID, userRoles, &service.UpdateFittingItemsSettingsRequest{
		Items: fleetDTOs(req.Items, func(item apicontract.UpdateItemSettingsReq) service.UpdateItemSettingsReq {
			return service.UpdateItemSettingsReq{
				ID: item.ID, Importance: item.Importance, Penalty: item.Penalty,
				ReplacementPenalty: item.ReplacementPenalty, Replacements: item.Replacements,
			}
		}),
	}); err != nil {
		return &fleetOperationFailure{code: response.CodeBizError, message: err.Error()}
	}
	return nil
}

func fleetConfigFittingInputs(items []apicontract.FleetConfigFittingReq) []service.FleetConfigFittingReq {
	return fleetDTOs(items, func(f apicontract.FleetConfigFittingReq) service.FleetConfigFittingReq {
		return service.FleetConfigFittingReq{ID: f.ID, FittingName: f.FittingName, EFT: f.EFT, SrpAmount: f.SrpAmount}
	})
}

func fleetConfigOptionalFittingInputs(items *[]apicontract.FleetConfigFittingReq) *[]service.FleetConfigFittingReq {
	if items == nil {
		return nil
	}
	converted := fleetConfigFittingInputs(*items)
	return &converted
}

func fleetConfigRecordDTO(config service.FleetConfigResp) apicontract.FleetConfigRecord {
	return apicontract.FleetConfigRecord{
		ID: config.ID, Name: config.Name, Description: config.Description,
		CreatedBy: config.CreatedBy, CreatedAt: config.CreatedAt, UpdatedAt: config.UpdatedAt,
		Fittings: fleetDTOs(config.Fittings, func(f service.FleetConfigFittingResp) apicontract.FleetConfigFitting {
			return apicontract.FleetConfigFitting{ID: f.ID, FleetConfigID: f.FleetConfigID, ShipTypeID: f.ShipTypeID, FittingName: f.FittingName, SrpAmount: f.SrpAmount}
		}),
	}
}

func fleetConfigEFTDTO(config service.FleetConfigEFTResponse) apicontract.FleetConfigEFTData {
	return apicontract.FleetConfigEFTData{
		Fittings: fleetDTOs(config.Fittings, func(f service.FleetConfigEFTFitting) apicontract.FleetConfigEFTFitting {
			return apicontract.FleetConfigEFTFitting{ID: f.ID, EFT: f.EFT}
		}),
	}
}

func fleetConfigFittingItemsDTO(f service.FittingItemsResponse) apicontract.FleetConfigFittingItems {
	return apicontract.FleetConfigFittingItems{
		FittingID: f.FittingID, FittingName: f.FittingName, ShipTypeID: f.ShipTypeID,
		Items: fleetDTOs(f.Items, func(item service.FittingItemResp) apicontract.FleetConfigFittingItem {
			return apicontract.FleetConfigFittingItem{
				ID: item.ID, TypeID: item.TypeID, TypeName: item.TypeName, Quantity: item.Quantity,
				Flag: item.Flag, FlagGroup: item.FlagGroup, Importance: item.Importance,
				Penalty: item.Penalty, ReplacementPenalty: item.ReplacementPenalty,
				Replacements: fleetDTOs(item.Replacements, func(rep service.FittingItemReplacementResp) apicontract.FleetConfigFittingReplacement {
					return apicontract.FleetConfigFittingReplacement{ID: rep.ID, TypeID: rep.TypeID, TypeName: rep.TypeName}
				}),
			}
		}),
	}
}

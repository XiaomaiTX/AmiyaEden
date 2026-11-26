package handler

import (
	"amiya-eden/internal/apicontract"
	"amiya-eden/internal/middleware"
	"amiya-eden/internal/service"
	"amiya-eden/pkg/response"
	"context"
	"github.com/gin-gonic/gin"
)

// EveInfoHandler EVE 人物信息处理器
type EveInfoHandler struct {
	svc         *service.EveInfoService
	cloneSvc    *service.CloneService
	assetSvc    *service.AssetService
	contractSvc *service.ContractService
}

func NewEveInfoHandler() *EveInfoHandler {
	return &EveInfoHandler{
		svc:         service.NewEveInfoService(),
		cloneSvc:    service.NewCloneService(),
		assetSvc:    service.NewAssetService(),
		contractSvc: service.NewContractService(),
	}
}

func (h *EveInfoHandler) getWalletJournal(userID uint, req apicontract.GetWalletJournalJSONRequestBody) (apicontract.InfoWalletResponse, error) {
	result, err := h.svc.GetWalletJournal(userID, (*service.InfoWalletRequest)(&req))
	if err != nil {
		return apicontract.InfoWalletResponse{}, err
	}
	return infoWalletDTO(result), nil
}

func (h *EveInfoHandler) getWalletAnalytics(userID uint, req apicontract.GetWalletAnalyticsJSONRequestBody) (apicontract.InfoWalletAnalyticsResponse, error) {
	result, err := h.svc.GetWalletAnalytics(userID, (*service.InfoWalletAnalyticsRequest)(&req))
	if err != nil {
		return apicontract.InfoWalletAnalyticsResponse{}, err
	}
	return infoWalletAnalyticsDTO(result), nil
}

func (h *EveInfoHandler) getCharacterSkills(userID uint, req apicontract.GetCharacterSkillsJSONRequestBody) (apicontract.InfoSkillResponse, error) {
	result, err := h.svc.GetCharacterSkills(userID, (*service.InfoSkillRequest)(&req))
	if err != nil {
		return apicontract.InfoSkillResponse{}, err
	}
	return infoSkillDTO(result), nil
}

func (h *EveInfoHandler) getCharacterShips(userID uint, req apicontract.GetCharacterShipsJSONRequestBody) (apicontract.InfoShipResponse, error) {
	result, err := h.svc.GetCharacterShips(userID, (*service.InfoShipRequest)(&req))
	if err != nil {
		return apicontract.InfoShipResponse{}, err
	}
	return infoShipDTO(result), nil
}

func (h *EveInfoHandler) getCharacterImplants(ctx context.Context, userID uint, req apicontract.GetCharacterImplantsJSONRequestBody) (apicontract.InfoImplantsResponse, error) {
	result, err := h.cloneSvc.GetCharacterImplants(ctx, userID, (*service.InfoImplantsRequest)(&req))
	if err != nil {
		return apicontract.InfoImplantsResponse{}, err
	}
	return infoImplantsDTO(result), nil
}

func (h *EveInfoHandler) getAssets(userID uint, req apicontract.GetAssetsJSONRequestBody) (apicontract.InfoAssets, error) {
	result, err := h.assetSvc.GetUserAssets(userID, (*service.InfoAssetsRequest)(&req))
	if err != nil {
		return apicontract.InfoAssets{}, err
	}
	return apicontract.InfoAssets{
		TotalItems: result.TotalItems,
		Locations:  mapEveInfoRows(result.Locations, infoAssetTreeLocationDTO),
	}, nil
}

func (h *EveInfoHandler) getAssetLocations(userID uint, req apicontract.GetAssetLocationsJSONRequestBody) (apicontract.InfoAssetLocations, error) {
	result, err := h.assetSvc.GetUserAssetLocations(userID, (*service.AssetLocationsRequest)(&req))
	if err != nil {
		return apicontract.InfoAssetLocations{}, err
	}
	return apicontract.InfoAssetLocations{
		TotalLocations: result.TotalLocations,
		TotalItems:     result.TotalItems,
		Locations: mapEveInfoRows(result.Locations, func(item service.AssetLocationSummary) apicontract.InfoAssetLocationSummary {
			return apicontract.InfoAssetLocationSummary(item)
		}),
	}, nil
}

func (h *EveInfoHandler) getAssetLocationItems(userID uint, req apicontract.GetAssetLocationItemsJSONRequestBody) (apicontract.InfoAssetLocationItems, error) {
	result, err := h.assetSvc.GetUserAssetLocationItems(userID, (*service.AssetLocationItemsRequest)(&req))
	if err != nil {
		return apicontract.InfoAssetLocationItems{}, err
	}
	return apicontract.InfoAssetLocationItems{
		LocationID: result.LocationID, LocationName: result.LocationName, TotalRootItems: result.TotalRootItems,
		Items: mapEveInfoRows(result.Items, infoAssetListItemDTO),
	}, nil
}

func (h *EveInfoHandler) getAssetChildren(userID uint, req apicontract.GetAssetChildrenJSONRequestBody) (apicontract.InfoAssetChildren, error) {
	result, err := h.assetSvc.GetUserAssetChildren(userID, (*service.AssetChildrenRequest)(&req))
	if err != nil {
		return apicontract.InfoAssetChildren{}, err
	}
	return apicontract.InfoAssetChildren{
		ParentItemID: result.ParentItemID,
		Items:        mapEveInfoRows(result.Items, infoAssetListItemDTO),
	}, nil
}

func (h *EveInfoHandler) getContracts(userID uint, req service.InfoContractsRequest) (apicontract.InfoContracts, error) {
	list, total, err := h.contractSvc.GetUserContracts(userID, &req)
	if err != nil {
		return apicontract.InfoContracts{}, err
	}
	return apicontract.InfoContracts{
		List: mapEveInfoRows(list, func(item service.ContractResponse) apicontract.InfoContractListItem {
			return apicontract.InfoContractListItem(item)
		}),
		Total: total, Page: req.Current, PageSize: req.Size,
	}, nil
}

func (h *EveInfoHandler) getContractDetail(userID uint, req apicontract.GetContractDetailJSONRequestBody) (apicontract.InfoContractDetail, error) {
	result, err := h.contractSvc.GetContractDetail(userID, (*service.InfoContractDetailRequest)(&req))
	if err != nil {
		return apicontract.InfoContractDetail{}, err
	}
	return apicontract.InfoContractDetail{
		Items: mapEveInfoRows(result.Items, func(item service.ContractItemDetail) apicontract.InfoContractItemDetail {
			return apicontract.InfoContractItemDetail(item)
		}),
		Bids: mapEveInfoRows(result.Bids, func(item service.ContractBidItem) apicontract.InfoContractBid {
			return apicontract.InfoContractBid(item)
		}),
	}, nil
}

func normalizeInfoContractsRequest(body *apicontract.GetContractsJSONRequestBody, bindErr error) service.InfoContractsRequest {
	var req apicontract.InfoContractsInput
	if body != nil {
		req = *body
	}
	if body == nil || bindErr != nil {
		// Keep decoded filters but fall back to 1/20 on binding errors. A bound
		// empty object or null instead uses the ledger defaults (1/200).
		req.Current = 1
		req.Size = 20
	}
	req.Current, req.Size = normalizeLedgerPagination(req.Current, req.Size)
	return service.InfoContractsRequest(req)
}

func infoAssetTreeLocationDTO(value service.AssetLocationNode) apicontract.InfoAssetTreeLocation {
	return apicontract.InfoAssetTreeLocation{
		LocationID: value.LocationID, LocationType: value.LocationType, LocationName: value.LocationName,
		Items: mapEveInfoRows(value.Items, infoAssetTreeItemDTO),
	}
}

func infoAssetTreeItemDTO(value service.AssetItemNode) apicontract.InfoAssetTreeItem {
	return apicontract.InfoAssetTreeItem{
		ItemID: value.ItemID, TypeID: value.TypeID, TypeName: value.TypeName, GroupName: value.GroupName,
		CategoryID: value.CategoryID, Quantity: value.Quantity, LocationFlag: value.LocationFlag,
		IsSingleton: value.IsSingleton, IsBlueprintCopy: value.IsBlueprintCopy, AssetName: value.AssetName,
		CharacterID: value.CharacterID, CharacterName: value.CharacterName,
		Children: mapEveInfoRows(value.Children, infoAssetTreeItemDTO),
	}
}

func infoAssetListItemDTO(value service.AssetListItemNode) apicontract.InfoAssetListItem {
	return apicontract.InfoAssetListItem(value)
}

// The strict Gin adapter supplies a Gin context. Preserve both its middleware
// identity and the HTTP request's cancellation and deadline for service calls.
func strictGinRequestDetails(ctx context.Context) (context.Context, uint) {
	if c, ok := ctx.(*gin.Context); ok && c != nil {
		if c.Request != nil {
			return c.Request.Context(), middleware.GetUserID(c)
		}
		return ctx, middleware.GetUserID(c)
	}
	return ctx, 0
}

func (h *EveInfoHandler) GetAssets(ctx context.Context, request apicontract.GetAssetsRequestObject) (apicontract.GetAssetsResponseObject, error) {
	out := &apicontract.GetAssets200JSONResponse{}
	if request.Body == nil {
		return out, out.FromBusinessError(contractBusinessError(response.CodeParamError, "参数错误"))
	}
	_, userID := strictGinRequestDetails(ctx)
	data, err := h.getAssets(userID, *request.Body)
	if err != nil {
		return out, out.FromBusinessError(contractBusinessError(response.CodeBizError, err.Error()))
	}
	return out, out.FromInfoAssetsSuccess(apicontract.InfoAssetsSuccess{Code: response.CodeOK, Msg: "success", Data: data})
}

func (h *EveInfoHandler) GetAssetLocations(ctx context.Context, request apicontract.GetAssetLocationsRequestObject) (apicontract.GetAssetLocationsResponseObject, error) {
	out := &apicontract.GetAssetLocations200JSONResponse{}
	if request.Body == nil {
		return out, out.FromBusinessError(contractBusinessError(response.CodeParamError, "参数错误"))
	}
	_, userID := strictGinRequestDetails(ctx)
	data, err := h.getAssetLocations(userID, *request.Body)
	if err != nil {
		return out, out.FromBusinessError(contractBusinessError(response.CodeBizError, err.Error()))
	}
	return out, out.FromInfoAssetLocationsSuccess(apicontract.InfoAssetLocationsSuccess{Code: response.CodeOK, Msg: "success", Data: data})
}

func (h *EveInfoHandler) GetAssetLocationItems(ctx context.Context, request apicontract.GetAssetLocationItemsRequestObject) (apicontract.GetAssetLocationItemsResponseObject, error) {
	out := &apicontract.GetAssetLocationItems200JSONResponse{}
	if request.Body == nil {
		return out, out.FromBusinessError(contractBusinessError(response.CodeParamError, "参数错误"))
	}
	_, userID := strictGinRequestDetails(ctx)
	data, err := h.getAssetLocationItems(userID, *request.Body)
	if err != nil {
		return out, out.FromBusinessError(contractBusinessError(response.CodeBizError, err.Error()))
	}
	return out, out.FromInfoAssetLocationItemsSuccess(apicontract.InfoAssetLocationItemsSuccess{Code: response.CodeOK, Msg: "success", Data: data})
}

func (h *EveInfoHandler) GetAssetChildren(ctx context.Context, request apicontract.GetAssetChildrenRequestObject) (apicontract.GetAssetChildrenResponseObject, error) {
	out := &apicontract.GetAssetChildren200JSONResponse{}
	if request.Body == nil {
		return out, out.FromBusinessError(contractBusinessError(response.CodeParamError, "参数错误"))
	}
	_, userID := strictGinRequestDetails(ctx)
	data, err := h.getAssetChildren(userID, *request.Body)
	if err != nil {
		return out, out.FromBusinessError(contractBusinessError(response.CodeBizError, err.Error()))
	}
	return out, out.FromInfoAssetChildrenSuccess(apicontract.InfoAssetChildrenSuccess{Code: response.CodeOK, Msg: "success", Data: data})
}

func (h *EveInfoHandler) GetContracts(ctx context.Context, request apicontract.GetContractsRequestObject) (apicontract.GetContractsResponseObject, error) {
	out := &apicontract.GetContracts200JSONResponse{}
	_, userID := strictGinRequestDetails(ctx)
	input := normalizeInfoContractsRequest(request.Body, apicontract.StrictGinJSONDecodeError(ctx))
	data, err := h.getContracts(userID, input)
	if err != nil {
		return out, out.FromBusinessError(contractBusinessError(response.CodeBizError, err.Error()))
	}
	return out, out.FromInfoContractsSuccess(apicontract.InfoContractsSuccess{Code: response.CodeOK, Msg: "success", Data: data})
}

func (h *EveInfoHandler) GetContractDetail(ctx context.Context, request apicontract.GetContractDetailRequestObject) (apicontract.GetContractDetailResponseObject, error) {
	out := &apicontract.GetContractDetail200JSONResponse{}
	if request.Body == nil {
		return out, out.FromBusinessError(contractBusinessError(response.CodeParamError, "参数错误"))
	}
	_, userID := strictGinRequestDetails(ctx)
	data, err := h.getContractDetail(userID, *request.Body)
	if err != nil {
		return out, out.FromBusinessError(contractBusinessError(response.CodeBizError, err.Error()))
	}
	return out, out.FromInfoContractDetailSuccess(apicontract.InfoContractDetailSuccess{Code: response.CodeOK, Msg: "success", Data: data})
}

func (h *EveInfoHandler) GetWalletJournal(ctx context.Context, request apicontract.GetWalletJournalRequestObject) (apicontract.GetWalletJournalResponseObject, error) {
	out := &apicontract.GetWalletJournal200JSONResponse{}
	if request.Body == nil {
		return out, out.FromBusinessError(contractBusinessError(response.CodeParamError, "参数错误"))
	}
	_, userID := strictGinRequestDetails(ctx)
	data, err := h.getWalletJournal(userID, *request.Body)
	if err != nil {
		return out, out.FromBusinessError(contractBusinessError(response.CodeBizError, err.Error()))
	}
	return out, out.FromInfoWalletSuccess(apicontract.InfoWalletSuccess{Code: response.CodeOK, Msg: "success", Data: data})
}

func (h *EveInfoHandler) GetWalletAnalytics(ctx context.Context, request apicontract.GetWalletAnalyticsRequestObject) (apicontract.GetWalletAnalyticsResponseObject, error) {
	out := &apicontract.GetWalletAnalytics200JSONResponse{}
	if request.Body == nil {
		return out, out.FromBusinessError(contractBusinessError(response.CodeParamError, "参数错误"))
	}
	_, userID := strictGinRequestDetails(ctx)
	data, err := h.getWalletAnalytics(userID, *request.Body)
	if err != nil {
		return out, out.FromBusinessError(contractBusinessError(response.CodeBizError, err.Error()))
	}
	return out, out.FromInfoWalletAnalyticsSuccess(apicontract.InfoWalletAnalyticsSuccess{Code: response.CodeOK, Msg: "success", Data: data})
}

func (h *EveInfoHandler) GetCharacterSkills(ctx context.Context, request apicontract.GetCharacterSkillsRequestObject) (apicontract.GetCharacterSkillsResponseObject, error) {
	out := &apicontract.GetCharacterSkills200JSONResponse{}
	if request.Body == nil {
		return out, out.FromBusinessError(contractBusinessError(response.CodeParamError, "参数错误"))
	}
	_, userID := strictGinRequestDetails(ctx)
	data, err := h.getCharacterSkills(userID, *request.Body)
	if err != nil {
		return out, out.FromBusinessError(contractBusinessError(response.CodeBizError, err.Error()))
	}
	return out, out.FromInfoSkillSuccess(apicontract.InfoSkillSuccess{Code: response.CodeOK, Msg: "success", Data: data})
}

func (h *EveInfoHandler) GetCharacterShips(ctx context.Context, request apicontract.GetCharacterShipsRequestObject) (apicontract.GetCharacterShipsResponseObject, error) {
	out := &apicontract.GetCharacterShips200JSONResponse{}
	if request.Body == nil {
		return out, out.FromBusinessError(contractBusinessError(response.CodeParamError, "参数错误"))
	}
	_, userID := strictGinRequestDetails(ctx)
	data, err := h.getCharacterShips(userID, *request.Body)
	if err != nil {
		return out, out.FromBusinessError(contractBusinessError(response.CodeBizError, err.Error()))
	}
	return out, out.FromInfoShipSuccess(apicontract.InfoShipSuccess{Code: response.CodeOK, Msg: "success", Data: data})
}

func (h *EveInfoHandler) GetCharacterImplants(ctx context.Context, request apicontract.GetCharacterImplantsRequestObject) (apicontract.GetCharacterImplantsResponseObject, error) {
	out := &apicontract.GetCharacterImplants200JSONResponse{}
	if request.Body == nil {
		return out, out.FromBusinessError(contractBusinessError(response.CodeParamError, "参数错误"))
	}
	requestCtx, userID := strictGinRequestDetails(ctx)
	data, err := h.getCharacterImplants(requestCtx, userID, *request.Body)
	if err != nil {
		return out, out.FromBusinessError(contractBusinessError(response.CodeBizError, err.Error()))
	}
	return out, out.FromInfoImplantsSuccess(apicontract.InfoImplantsSuccess{Code: response.CodeOK, Msg: "success", Data: data})
}

func infoWalletDTO(result *service.InfoWalletResponse) apicontract.InfoWalletResponse {
	return apicontract.InfoWalletResponse{
		Balance: result.Balance, RefTypes: result.RefTypes, Total: result.Total, Page: result.Page,
		PageSize: result.PageSize,
		Journals: mapEveInfoRows(result.Journals, func(row service.InfoWalletJournal) apicontract.InfoWalletJournal {
			return apicontract.InfoWalletJournal(row)
		}),
	}
}

func infoWalletAnalyticsDTO(result *service.InfoWalletAnalyticsResponse) apicontract.InfoWalletAnalyticsResponse {
	return apicontract.InfoWalletAnalyticsResponse{
		Summary: apicontract.InfoWalletAnalyticsSummary(result.Summary),
		DailySeries: mapEveInfoRows(result.DailySeries, func(row service.InfoWalletDailyPoint) apicontract.InfoWalletDailyPoint {
			return apicontract.InfoWalletDailyPoint(row)
		}),
		RefTypeBreakdown: mapEveInfoRows(result.RefTypeBreakdown, func(row service.InfoWalletRefTypeItem) apicontract.InfoWalletRefTypeItem {
			return apicontract.InfoWalletRefTypeItem(row)
		}),
	}
}

func infoSkillDTO(result *service.InfoSkillResponse) apicontract.InfoSkillResponse {
	queue := mapEveInfoRows(result.SkillQueue, func(row service.InfoSkillQueueItem) apicontract.InfoSkillQueueItem {
		return apicontract.InfoSkillQueueItem(row)
	})
	return apicontract.InfoSkillResponse{
		TotalSP: result.TotalSP, UnallocatedSP: result.UnallocatedSP, SkillCount: result.SkillCount,
		Skills: mapEveInfoRows(result.Skills, func(row service.InfoSkillItem) apicontract.InfoSkillItem {
			return apicontract.InfoSkillItem(row)
		}),
		SkillQueue: &queue,
	}
}

func infoShipDTO(result *service.InfoShipResponse) apicontract.InfoShipResponse {
	return apicontract.InfoShipResponse{
		TotalShips: result.TotalShips, FlyableShips: result.FlyableShips,
		Ships: mapEveInfoRows(result.Ships, func(row service.InfoShipItem) apicontract.InfoShipItem {
			return apicontract.InfoShipItem{
				TypeID: row.TypeID, TypeName: row.TypeName, GroupID: row.GroupID, GroupName: row.GroupName,
				MarketGroupID: row.MarketGroupID, MarketGroupName: row.MarketGroupName,
				RaceID: row.RaceID, RaceName: row.RaceName, CanFly: row.CanFly,
				SkillReqs: mapEveInfoRows(row.SkillReqs, func(req service.InfoShipSkillReq) apicontract.InfoShipSkillReq {
					return apicontract.InfoShipSkillReq(req)
				}),
			}
		}),
	}
}

func infoImplantsDTO(result *service.InfoImplantsResponse) apicontract.InfoImplantsResponse {
	mapItem := func(row service.ImplantItem) apicontract.ImplantItem { return apicontract.ImplantItem(row) }
	return apicontract.InfoImplantsResponse{
		HomeLocation:      (*apicontract.ImplantLocation)(result.HomeLocation),
		LastCloneJumpDate: result.LastCloneJumpDate, LastStationChangeDate: result.LastStationChangeDate,
		JumpFatigueExpire: result.JumpFatigueExpire, LastJumpDate: result.LastJumpDate,
		ActiveImplants: mapEveInfoRows(result.ActiveImplants, mapItem),
		JumpClones: mapEveInfoRows(result.JumpClones, func(row service.JumpCloneInfo) apicontract.JumpCloneInfo {
			return apicontract.JumpCloneInfo{
				JumpCloneID: row.JumpCloneID, Location: apicontract.ImplantLocation(row.Location),
				Implants: mapEveInfoRows(row.Implants, mapItem),
			}
		}),
	}
}

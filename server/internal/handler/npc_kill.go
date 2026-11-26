package handler

import (
	"amiya-eden/internal/apicontract"
	"amiya-eden/internal/service"
	"amiya-eden/pkg/response"
	"context"
)

// NpcKillHandler NPC 刷怪报表处理器
type NpcKillHandler struct {
	svc *service.NpcKillService
}

func NewNpcKillHandler() *NpcKillHandler {
	return &NpcKillHandler{
		svc: service.NewNpcKillService(),
	}
}

func (h *NpcKillHandler) getNpcKills(userID uint, req apicontract.GetNpcKillsJSONRequestBody) (apicontract.NpcKillData, error) {
	result, err := h.svc.GetNpcKills(userID, (*service.NpcKillRequest)(&req))
	if err != nil {
		return apicontract.NpcKillData{}, err
	}
	return npcKillDTO(result), nil
}

func (h *NpcKillHandler) getAllNpcKills(userID uint, req apicontract.GetAllNpcKillsJSONRequestBody) (apicontract.NpcKillData, error) {
	result, err := h.svc.GetAllNpcKills(userID, (*service.NpcKillAllRequest)(&req))
	if err != nil {
		return apicontract.NpcKillData{}, err
	}
	return npcKillDTO(result), nil
}

func (h *NpcKillHandler) getCorpNpcKills(req apicontract.GetCorpNpcKillsJSONRequestBody) (apicontract.NpcKillCorpData, error) {
	result, err := h.svc.GetCorpNpcKills((*service.NpcKillCorpRequest)(&req))
	if err != nil {
		return apicontract.NpcKillCorpData{}, err
	}
	return npcKillCorpDTO(result), nil
}

func npcKillDTO(value *service.NpcKillResponse) apicontract.NpcKillData {
	byNpc := mapEveInfoRows(value.ByNpc, func(item service.NpcKillByNpc) apicontract.NpcKillByNpc { return apicontract.NpcKillByNpc(item) })
	return apicontract.NpcKillData{
		Summary:  apicontract.NpcKillSummary(value.Summary),
		ByNpc:    &byNpc,
		BySystem: mapEveInfoRows(value.BySystem, npcKillSystemDTO),
		Trend:    mapEveInfoRows(value.Trend, npcKillTrendDTO),
		Journals: mapEveInfoRows(value.Journals, func(item service.NpcKillJournalItem) apicontract.NpcKillJournalItem {
			return apicontract.NpcKillJournalItem(item)
		}),
		Total: value.Total, Page: value.Page, PageSize: value.PageSize,
	}
}

func npcKillCorpDTO(value *service.NpcKillCorpResponse) apicontract.NpcKillCorpData {
	return apicontract.NpcKillCorpData{
		Summary: apicontract.NpcKillSummary(value.Summary),
		Members: mapEveInfoRows(value.Members, func(item service.NpcKillCorpMemberSummary) apicontract.NpcKillCorpMemberSummary {
			return apicontract.NpcKillCorpMemberSummary(item)
		}),
		BySystem: mapEveInfoRows(value.BySystem, npcKillSystemDTO),
		Trend:    mapEveInfoRows(value.Trend, npcKillTrendDTO),
	}
}

func npcKillSystemDTO(value service.NpcKillBySystem) apicontract.NpcKillBySystem {
	return apicontract.NpcKillBySystem(value)
}

func npcKillTrendDTO(value service.NpcKillTrend) apicontract.NpcKillTrend {
	return apicontract.NpcKillTrend(value)
}

func (h *NpcKillHandler) GetNpcKills(ctx context.Context, request apicontract.GetNpcKillsRequestObject) (apicontract.GetNpcKillsResponseObject, error) {
	out := &apicontract.GetNpcKills200JSONResponse{}
	if request.Body == nil {
		return out, out.FromBusinessError(contractBusinessError(response.CodeParamError, "参数错误"))
	}
	_, userID := strictGinRequestDetails(ctx)
	data, err := h.getNpcKills(userID, *request.Body)
	if err != nil {
		return out, out.FromBusinessError(contractBusinessError(response.CodeBizError, err.Error()))
	}
	return out, out.FromNpcKillDataSuccess(apicontract.NpcKillDataSuccess{Code: response.CodeOK, Msg: "success", Data: data})
}

func (h *NpcKillHandler) GetAllNpcKills(ctx context.Context, request apicontract.GetAllNpcKillsRequestObject) (apicontract.GetAllNpcKillsResponseObject, error) {
	out := &apicontract.GetAllNpcKills200JSONResponse{}
	if request.Body == nil {
		return out, out.FromBusinessError(contractBusinessError(response.CodeParamError, "参数错误"))
	}
	_, userID := strictGinRequestDetails(ctx)
	data, err := h.getAllNpcKills(userID, *request.Body)
	if err != nil {
		return out, out.FromBusinessError(contractBusinessError(response.CodeBizError, err.Error()))
	}
	return out, out.FromNpcKillDataSuccess(apicontract.NpcKillDataSuccess{Code: response.CodeOK, Msg: "success", Data: data})
}

func (h *NpcKillHandler) GetCorpNpcKills(_ context.Context, request apicontract.GetCorpNpcKillsRequestObject) (apicontract.GetCorpNpcKillsResponseObject, error) {
	out := &apicontract.GetCorpNpcKills200JSONResponse{}
	if request.Body == nil {
		return out, out.FromBusinessError(contractBusinessError(response.CodeParamError, "参数错误"))
	}
	data, err := h.getCorpNpcKills(*request.Body)
	if err != nil {
		return out, out.FromBusinessError(contractBusinessError(response.CodeBizError, err.Error()))
	}
	return out, out.FromNpcKillCorpDataSuccess(apicontract.NpcKillCorpDataSuccess{Code: response.CodeOK, Msg: "success", Data: data})
}

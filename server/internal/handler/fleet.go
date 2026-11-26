package handler

import (
	"amiya-eden/internal/apicontract"
	"amiya-eden/internal/middleware"
	"amiya-eden/internal/model"
	"amiya-eden/internal/service"
	"amiya-eden/pkg/response"
	"context"
	"github.com/gin-gonic/gin"
	"math"
	"strings"
)

// FleetHandler 舰队 HTTP 处理器
type FleetHandler struct {
	svc *service.FleetService
}

func NewFleetHandler() *FleetHandler {
	return &FleetHandler{
		svc: service.NewFleetService(),
	}
}

// The generated strict adapter supplies the Gin context populated by the existing
// authentication middleware. Only read identity here; request data is already typed.
func fleetStrictActor(ctx context.Context) (uint, []string) {
	if c, ok := ctx.(*gin.Context); ok && c != nil {
		return middleware.GetUserID(c), middleware.GetUserRoles(c)
	}
	return 0, nil
}

func fleetValidUintID(id uint) bool { return id != 0 && id <= math.MaxUint32 }

// fleetOperationFailure carries the legacy business-envelope error without
// tying the operation core to either HTTP adapter.
type fleetOperationFailure struct {
	code    int
	message string
}

func (failure *fleetOperationFailure) contract() apicontract.BusinessError {
	return contractBusinessError(failure.code, failure.message)
}

func (h *FleetHandler) CreateFleet(ctx context.Context, request apicontract.CreateFleetRequestObject) (apicontract.CreateFleetResponseObject, error) {
	out := &apicontract.CreateFleet200JSONResponse{}
	if request.Body == nil {
		return out, out.FromBusinessError(contractBusinessError(response.CodeParamError, "请求参数错误"))
	}
	userID, _ := fleetStrictActor(ctx)
	data, failure := h.createFleet(userID, *request.Body)
	if failure != nil {
		return out, out.FromBusinessError(failure.contract())
	}
	return out, out.FromFleetRecordSuccess(apicontract.FleetRecordSuccess{Code: response.CodeOK, Msg: "success", Data: data})
}

func (h *FleetHandler) ListFleets(_ context.Context, request apicontract.ListFleetsRequestObject) (apicontract.ListFleetsResponseObject, error) {
	out := &apicontract.ListFleets200JSONResponse{}
	data, failure := h.listFleets(request.Params)
	if failure != nil {
		return out, out.FromBusinessError(failure.contract())
	}
	return out, out.FromFleetPageSuccess(apicontract.FleetPageSuccess{Code: response.CodeOK, Msg: "success", Data: data})
}

func (h *FleetHandler) GetMyFleets(ctx context.Context, request apicontract.GetMyFleetsRequestObject) (apicontract.GetMyFleetsResponseObject, error) {
	out := &apicontract.GetMyFleets200JSONResponse{}
	userID, _ := fleetStrictActor(ctx)
	data, failure := h.getMyFleets(userID)
	if failure != nil {
		return out, out.FromBusinessError(failure.contract())
	}
	return out, out.FromMyFleetListSuccess(apicontract.MyFleetListSuccess{Code: response.CodeOK, Msg: "success", Data: &data})
}

func (h *FleetHandler) GetFleet(_ context.Context, request apicontract.GetFleetRequestObject) (apicontract.GetFleetResponseObject, error) {
	out := &apicontract.GetFleet200JSONResponse{}
	if request.Id == "" {
		return out, out.FromBusinessError(contractBusinessError(response.CodeParamError, "缺少舰队ID"))
	}
	data, failure := h.getFleet(request.Id)
	if failure != nil {
		return out, out.FromBusinessError(failure.contract())
	}
	return out, out.FromFleetRecordSuccess(apicontract.FleetRecordSuccess{Code: response.CodeOK, Msg: "success", Data: data})
}

func (h *FleetHandler) RefreshFleetESI(ctx context.Context, request apicontract.RefreshFleetESIRequestObject) (apicontract.RefreshFleetESIResponseObject, error) {
	out := &apicontract.RefreshFleetESI200JSONResponse{}
	if request.Id == "" {
		return out, out.FromBusinessError(contractBusinessError(response.CodeParamError, "缺少舰队ID"))
	}
	userID, roles := fleetStrictActor(ctx)
	data, failure := h.refreshFleetESI(request.Id, userID, roles)
	if failure != nil {
		return out, out.FromBusinessError(failure.contract())
	}
	return out, out.FromFleetRecordSuccess(apicontract.FleetRecordSuccess{Code: response.CodeOK, Msg: "success", Data: data})
}

func (h *FleetHandler) UpdateFleet(ctx context.Context, request apicontract.UpdateFleetRequestObject) (apicontract.UpdateFleetResponseObject, error) {
	out := &apicontract.UpdateFleet200JSONResponse{}
	if request.Id == "" {
		return out, out.FromBusinessError(contractBusinessError(response.CodeParamError, "缺少舰队ID"))
	}
	if request.Body == nil {
		return out, out.FromBusinessError(contractBusinessError(response.CodeParamError, "请求参数错误"))
	}
	userID, roles := fleetStrictActor(ctx)
	data, failure := h.updateFleet(request.Id, userID, roles, *request.Body)
	if failure != nil {
		return out, out.FromBusinessError(failure.contract())
	}
	return out, out.FromFleetRecordSuccess(apicontract.FleetRecordSuccess{Code: response.CodeOK, Msg: "success", Data: data})
}

func (h *FleetHandler) DeleteFleet(ctx context.Context, request apicontract.DeleteFleetRequestObject) (apicontract.DeleteFleetResponseObject, error) {
	out := &apicontract.DeleteFleet200JSONResponse{}
	if request.Id == "" {
		return out, out.FromBusinessError(contractBusinessError(response.CodeParamError, "缺少舰队ID"))
	}
	userID, roles := fleetStrictActor(ctx)
	if failure := h.deleteFleet(request.Id, userID, roles); failure != nil {
		return out, out.FromBusinessError(failure.contract())
	}
	return out, out.FromNullSuccess(apicontract.NullSuccess{Code: response.CodeOK, Msg: "success"})
}

func (h *FleetHandler) GetMembers(_ context.Context, request apicontract.GetMembersRequestObject) (apicontract.GetMembersResponseObject, error) {
	out := &apicontract.GetMembers200JSONResponse{}
	if request.Id == "" {
		return out, out.FromBusinessError(contractBusinessError(response.CodeParamError, "缺少舰队ID"))
	}
	data, failure := h.getMembers(request.Id)
	if failure != nil {
		return out, out.FromBusinessError(failure.contract())
	}
	return out, out.FromFleetMemberListSuccess(apicontract.FleetMemberListSuccess{Code: response.CodeOK, Msg: "success", Data: &data})
}

func (h *FleetHandler) GetMembersWithPap(_ context.Context, request apicontract.GetMembersWithPapRequestObject) (apicontract.GetMembersWithPapResponseObject, error) {
	out := &apicontract.GetMembersWithPap200JSONResponse{}
	if request.Id == "" {
		return out, out.FromBusinessError(contractBusinessError(response.CodeParamError, "缺少舰队ID"))
	}
	data, failure := h.getMembersWithPap(request.Id, request.Params)
	if failure != nil {
		return out, out.FromBusinessError(failure.contract())
	}
	return out, out.FromFleetMemberWithPapPageSuccess(apicontract.FleetMemberWithPapPageSuccess{Code: response.CodeOK, Msg: "success", Data: data})
}

func (h *FleetHandler) ManualAddMembers(ctx context.Context, request apicontract.ManualAddMembersRequestObject) (apicontract.ManualAddMembersResponseObject, error) {
	out := &apicontract.ManualAddMembers200JSONResponse{}
	if request.Id == "" {
		return out, out.FromBusinessError(contractBusinessError(response.CodeParamError, "缺少舰队ID"))
	}
	if request.Body == nil {
		return out, out.FromBusinessError(contractBusinessError(response.CodeParamError, "请求参数错误"))
	}
	userID, roles := fleetStrictActor(ctx)
	data, failure := h.manualAddMembers(request.Id, userID, roles, *request.Body)
	if failure != nil {
		return out, out.FromBusinessError(failure.contract())
	}
	return out, out.FromFleetManualAddResultDataSuccess(apicontract.FleetManualAddResultDataSuccess{Code: response.CodeOK, Msg: "success", Data: data})
}

func (h *FleetHandler) SyncESIMembers(ctx context.Context, request apicontract.SyncESIMembersRequestObject) (apicontract.SyncESIMembersResponseObject, error) {
	out := &apicontract.SyncESIMembers200JSONResponse{}
	if request.Id == "" {
		return out, out.FromBusinessError(contractBusinessError(response.CodeParamError, "缺少舰队ID"))
	}
	userID, roles := fleetStrictActor(ctx)
	data, failure := h.syncESIMembers(request.Id, userID, roles)
	if failure != nil {
		return out, out.FromBusinessError(failure.contract())
	}
	return out, out.FromFleetESIMemberListSuccess(apicontract.FleetESIMemberListSuccess{Code: response.CodeOK, Msg: "success", Data: &data})
}

func (h *FleetHandler) IssuePap(ctx context.Context, request apicontract.IssuePapRequestObject) (apicontract.IssuePapResponseObject, error) {
	out := &apicontract.IssuePap200JSONResponse{}
	if request.Id == "" {
		return out, out.FromBusinessError(contractBusinessError(response.CodeParamError, "缺少舰队ID"))
	}
	userID, roles := fleetStrictActor(ctx)
	if failure := h.issuePap(request.Id, userID, roles); failure != nil {
		return out, out.FromBusinessError(failure.contract())
	}
	return out, out.FromNullSuccess(apicontract.NullSuccess{Code: response.CodeOK, Msg: "success"})
}

func (h *FleetHandler) GetPapLogs(_ context.Context, request apicontract.GetPapLogsRequestObject) (apicontract.GetPapLogsResponseObject, error) {
	out := &apicontract.GetPapLogs200JSONResponse{}
	if request.Id == "" {
		return out, out.FromBusinessError(contractBusinessError(response.CodeParamError, "缺少舰队ID"))
	}
	data, failure := h.getPapLogs(request.Id)
	if failure != nil {
		return out, out.FromBusinessError(failure.contract())
	}
	return out, out.FromFleetPapLogListSuccess(apicontract.FleetPapLogListSuccess{Code: response.CodeOK, Msg: "success", Data: &data})
}

func (h *FleetHandler) GetMyPapLogs(ctx context.Context, request apicontract.GetMyPapLogsRequestObject) (apicontract.GetMyPapLogsResponseObject, error) {
	out := &apicontract.GetMyPapLogs200JSONResponse{}
	userID, _ := fleetStrictActor(ctx)
	data, failure := h.getMyPapLogs(userID)
	if failure != nil {
		return out, out.FromBusinessError(failure.contract())
	}
	return out, out.FromMyFleetPapLogListSuccess(apicontract.MyFleetPapLogListSuccess{Code: response.CodeOK, Msg: "success", Data: &data})
}

func (h *FleetHandler) GetCorporationPapSummary(_ context.Context, request apicontract.GetCorporationPapSummaryRequestObject) (apicontract.GetCorporationPapSummaryResponseObject, error) {
	out := &apicontract.GetCorporationPapSummary200JSONResponse{}
	data, failure := h.getCorporationPapSummary(request.Params)
	if failure != nil {
		return out, out.FromBusinessError(failure.contract())
	}
	return out, out.FromFleetCorporationPapPageSuccess(apicontract.FleetCorporationPapPageSuccess{Code: response.CodeOK, Msg: "success", Data: data})
}

func (h *FleetHandler) CreateInvite(ctx context.Context, request apicontract.CreateInviteRequestObject) (apicontract.CreateInviteResponseObject, error) {
	out := &apicontract.CreateInvite200JSONResponse{}
	if request.Id == "" {
		return out, out.FromBusinessError(contractBusinessError(response.CodeParamError, "缺少舰队ID"))
	}
	userID, roles := fleetStrictActor(ctx)
	data, failure := h.createInvite(request.Id, userID, roles)
	if failure != nil {
		return out, out.FromBusinessError(failure.contract())
	}
	return out, out.FromFleetInviteSuccess(apicontract.FleetInviteSuccess{Code: response.CodeOK, Msg: "success", Data: data})
}

func (h *FleetHandler) GetInvites(_ context.Context, request apicontract.GetInvitesRequestObject) (apicontract.GetInvitesResponseObject, error) {
	out := &apicontract.GetInvites200JSONResponse{}
	if request.Id == "" {
		return out, out.FromBusinessError(contractBusinessError(response.CodeParamError, "缺少舰队ID"))
	}
	data, failure := h.getInvites(request.Id)
	if failure != nil {
		return out, out.FromBusinessError(failure.contract())
	}
	return out, out.FromFleetInviteListSuccess(apicontract.FleetInviteListSuccess{Code: response.CodeOK, Msg: "success", Data: &data})
}

func (h *FleetHandler) DeactivateInvite(ctx context.Context, request apicontract.DeactivateInviteRequestObject) (apicontract.DeactivateInviteResponseObject, error) {
	out := &apicontract.DeactivateInvite200JSONResponse{}
	if !fleetValidUintID(request.InviteId) {
		return out, out.FromBusinessError(contractBusinessError(response.CodeParamError, "无效的邀请ID"))
	}
	userID, roles := fleetStrictActor(ctx)
	if failure := h.deactivateInvite(request.InviteId, userID, roles); failure != nil {
		return out, out.FromBusinessError(failure.contract())
	}
	return out, out.FromNullSuccess(apicontract.NullSuccess{Code: response.CodeOK, Msg: "success"})
}

func (h *FleetHandler) JoinFleet(ctx context.Context, request apicontract.JoinFleetRequestObject) (apicontract.JoinFleetResponseObject, error) {
	out := &apicontract.JoinFleet200JSONResponse{}
	if request.Body == nil {
		return out, out.FromBusinessError(contractBusinessError(response.CodeParamError, "请求参数错误"))
	}
	userID, _ := fleetStrictActor(ctx)
	if failure := h.joinFleet(userID, *request.Body); failure != nil {
		return out, out.FromBusinessError(failure.contract())
	}
	return out, out.FromNullSuccess(apicontract.NullSuccess{Code: response.CodeOK, Msg: "success"})
}

func (h *FleetHandler) GetCharacterFleetInfo(ctx context.Context, request apicontract.GetCharacterFleetInfoRequestObject) (apicontract.GetCharacterFleetInfoResponseObject, error) {
	out := &apicontract.GetCharacterFleetInfo200JSONResponse{}
	userID, _ := fleetStrictActor(ctx)
	data, failure := h.getCharacterFleetInfo(userID, request.CharacterId)
	if failure != nil {
		return out, out.FromBusinessError(failure.contract())
	}
	return out, out.FromFleetCharacterInfoSuccess(apicontract.FleetCharacterInfoSuccess{Code: response.CodeOK, Msg: "success", Data: data})
}

func (h *FleetHandler) PingFleet(ctx context.Context, request apicontract.PingFleetRequestObject) (apicontract.PingFleetResponseObject, error) {
	out := &apicontract.PingFleet200JSONResponse{}
	if request.Id == "" {
		return out, out.FromBusinessError(contractBusinessError(response.CodeParamError, "缺少舰队ID"))
	}
	userID, roles := fleetStrictActor(ctx)
	if failure := h.pingFleet(request.Id, userID, roles); failure != nil {
		return out, out.FromBusinessError(failure.contract())
	}
	return out, out.FromNullSuccess(apicontract.NullSuccess{Code: response.CodeOK, Msg: "success"})
}

func (h *FleetHandler) createFleet(userID uint, req apicontract.FleetCreateInput) (apicontract.FleetRecord, *fleetOperationFailure) {
	fleet, err := h.svc.CreateFleet(userID, &service.CreateFleetRequest{
		Title: req.Title, Description: req.Description, StartAt: req.StartAt, EndAt: req.EndAt,
		Importance: req.Importance, PapCount: req.PapCount, CharacterID: req.CharacterID,
		SendPing: req.SendPing, FleetConfigID: req.FleetConfigID, AutoSrpMode: req.AutoSrpMode,
	})
	if err != nil {
		return apicontract.FleetRecord{}, &fleetOperationFailure{code: response.CodeBizError, message: err.Error()}
	}
	return fleetRecordDTO(*fleet), nil
}

func (h *FleetHandler) listFleets(params apicontract.ListFleetsParams) (apicontract.FleetPage, *fleetOperationFailure) {
	page, pageSize := normalizePagination(fleetValue(params.Current), fleetValue(params.Size), 20, 100)
	if params.FcUserId != nil && !fleetValidUintID(*params.FcUserId) {
		params.FcUserId = nil
	}
	filter := service.FleetFilter{Importance: fleetValue(params.Importance), FCUserID: params.FcUserId}
	records, total, err := h.svc.ListFleets(page, pageSize, filter)
	if err != nil {
		return apicontract.FleetPage{}, &fleetOperationFailure{code: response.CodeBizError, message: err.Error()}
	}
	return apicontract.FleetPage{
		List: fleetNullableDTOs(records, fleetListItemDTO), Total: total, Page: page, PageSize: pageSize,
	}, nil
}

func (h *FleetHandler) getMyFleets(userID uint) ([]apicontract.FleetRecord, *fleetOperationFailure) {
	fleets, err := h.svc.GetMyFleets(userID)
	if err != nil {
		return nil, &fleetOperationFailure{code: response.CodeBizError, message: err.Error()}
	}
	return fleetDTOs(fleets, fleetRecordDTO), nil
}

func (h *FleetHandler) getFleet(fleetID string) (apicontract.FleetRecord, *fleetOperationFailure) {
	fleet, err := h.svc.GetFleet(fleetID)
	if err != nil {
		return apicontract.FleetRecord{}, &fleetOperationFailure{code: response.CodeNotFound, message: "舰队不存在"}
	}
	return fleetRecordDTO(*fleet), nil
}

func (h *FleetHandler) refreshFleetESI(fleetID string, userID uint, userRoles []string) (apicontract.FleetRecord, *fleetOperationFailure) {
	fleet, err := h.svc.RefreshESIFleetID(fleetID, userID, userRoles)
	if err != nil {
		return apicontract.FleetRecord{}, &fleetOperationFailure{code: response.CodeBizError, message: err.Error()}
	}
	return fleetRecordDTO(*fleet), nil
}

func (h *FleetHandler) updateFleet(fleetID string, userID uint, userRoles []string, req apicontract.FleetUpdateInput) (apicontract.FleetRecord, *fleetOperationFailure) {
	fleet, err := h.svc.UpdateFleet(fleetID, userID, userRoles, &service.UpdateFleetRequest{
		Title: req.Title, Description: req.Description, StartAt: req.StartAt, EndAt: req.EndAt,
		Importance: req.Importance, PapCount: req.PapCount, CharacterID: req.CharacterID,
		ESIFleetID: req.ESIFleetID, FleetConfigID: req.FleetConfigID, AutoSrpMode: req.AutoSrpMode,
	})
	if err != nil {
		return apicontract.FleetRecord{}, &fleetOperationFailure{code: response.CodeBizError, message: err.Error()}
	}
	return fleetRecordDTO(*fleet), nil
}

func (h *FleetHandler) deleteFleet(fleetID string, userID uint, userRoles []string) *fleetOperationFailure {
	if err := h.svc.DeleteFleet(fleetID, userID, userRoles); err != nil {
		return &fleetOperationFailure{code: response.CodeBizError, message: err.Error()}
	}
	return nil
}

func (h *FleetHandler) getMembers(fleetID string) ([]apicontract.FleetMember, *fleetOperationFailure) {
	members, err := h.svc.GetMembers(fleetID)
	if err != nil {
		return nil, &fleetOperationFailure{code: response.CodeBizError, message: err.Error()}
	}
	return fleetDTOs(members, fleetMemberDTO), nil
}

func (h *FleetHandler) getMembersWithPap(fleetID string, params apicontract.GetMembersWithPapParams) (apicontract.FleetMemberWithPapPage, *fleetOperationFailure) {
	page, pageSize := normalizePagination(fleetValue(params.Current), fleetValue(params.Size), 260, 260)
	list, total, err := h.svc.ListMembersWithPap(fleetID, page, pageSize)
	if err != nil {
		return apicontract.FleetMemberWithPapPage{}, &fleetOperationFailure{code: response.CodeBizError, message: err.Error()}
	}
	return apicontract.FleetMemberWithPapPage{
		List: fleetNullableDTOs(list, fleetMemberWithPapDTO), Total: total, Page: page, PageSize: pageSize,
	}, nil
}

func (h *FleetHandler) manualAddMembers(fleetID string, userID uint, userRoles []string, req apicontract.FleetManualAddInput) (apicontract.FleetManualAddResultData, *fleetOperationFailure) {

	result, err := h.svc.ManualAddMembers(fleetID, userID, userRoles, &service.ManualAddFleetMembersRequest{
		CharacterNames: req.CharacterNames,
	})
	if err != nil {
		return apicontract.FleetManualAddResultData{}, &fleetOperationFailure{code: response.CodeBizError, message: err.Error()}
	}
	return apicontract.FleetManualAddResultData{
		AddedCharacterNames: result.AddedCharacterNames, MissingCharacterNames: result.MissingCharacterNames,
	}, nil
}

func (h *FleetHandler) syncESIMembers(fleetID string, userID uint, userRoles []string) ([]apicontract.FleetESIMember, *fleetOperationFailure) {
	members, err := h.svc.SyncESIMembers(fleetID, userID, userRoles)
	if err != nil {
		return nil, &fleetOperationFailure{code: response.CodeBizError, message: err.Error()}
	}
	return fleetDTOs(members, func(member service.ESIFleetMember) apicontract.FleetESIMember {
		return apicontract.FleetESIMember(member)
	}), nil
}

func (h *FleetHandler) issuePap(fleetID string, userID uint, userRoles []string) *fleetOperationFailure {
	if err := h.svc.IssuePap(fleetID, userID, userRoles); err != nil {
		return &fleetOperationFailure{code: response.CodeBizError, message: err.Error()}
	}
	return nil
}

func (h *FleetHandler) getPapLogs(fleetID string) ([]apicontract.FleetPapLog, *fleetOperationFailure) {
	logs, err := h.svc.GetPapLogs(fleetID)
	if err != nil {
		return nil, &fleetOperationFailure{code: response.CodeBizError, message: err.Error()}
	}
	return fleetDTOs(logs, fleetPapLogDTO), nil
}

func (h *FleetHandler) getMyPapLogs(userID uint) ([]apicontract.FleetPapLogDetail, *fleetOperationFailure) {
	logs, err := h.svc.GetUserPapLogs(userID)
	if err != nil {
		return nil, &fleetOperationFailure{code: response.CodeBizError, message: err.Error()}
	}
	return fleetDTOs(logs, fleetPapLogDetailDTO), nil
}

func (h *FleetHandler) getCorporationPapSummary(params apicontract.GetCorporationPapSummaryParams) (apicontract.FleetCorporationPapPage, *fleetOperationFailure) {
	page, pageSize := normalizeLedgerPagination(fleetValue(params.Current), fleetValue(params.Size))
	corpTickerParam := fleetValue(params.CorpTickers)
	var corpTickers []string
	if corpTickerParam != "" {
		corpTickers = strings.Split(corpTickerParam, ",")
	}

	result, err := h.svc.GetCorporationPapSummary(page, pageSize, fleetValueOr(params.Period, service.CorporationPapPeriodLastMonth), fleetValue(params.Year), corpTickers)
	if err != nil {
		return apicontract.FleetCorporationPapPage{}, &fleetOperationFailure{code: response.CodeBizError, message: err.Error()}
	}

	return fleetCorporationPapPageDTO(*result), nil
}

func (h *FleetHandler) createInvite(fleetID string, userID uint, userRoles []string) (apicontract.FleetInvite, *fleetOperationFailure) {
	invite, err := h.svc.CreateInvite(fleetID, userID, userRoles)
	if err != nil {
		return apicontract.FleetInvite{}, &fleetOperationFailure{code: response.CodeBizError, message: err.Error()}
	}
	return fleetInviteDTO(*invite), nil
}

func (h *FleetHandler) getInvites(fleetID string) ([]apicontract.FleetInvite, *fleetOperationFailure) {
	invites, err := h.svc.GetInvites(fleetID)
	if err != nil {
		return nil, &fleetOperationFailure{code: response.CodeBizError, message: err.Error()}
	}
	return fleetDTOs(invites, fleetInviteDTO), nil
}

func (h *FleetHandler) deactivateInvite(inviteID, userID uint, userRoles []string) *fleetOperationFailure {
	if err := h.svc.DeactivateInvite(inviteID, userID, userRoles); err != nil {
		return &fleetOperationFailure{code: response.CodeBizError, message: err.Error()}
	}
	return nil
}

func (h *FleetHandler) joinFleet(userID uint, req apicontract.FleetJoinInput) *fleetOperationFailure {
	if err := h.svc.JoinFleet(req.Code, userID, req.CharacterID); err != nil {
		return &fleetOperationFailure{code: response.CodeBizError, message: err.Error()}
	}
	return nil
}

func (h *FleetHandler) getCharacterFleetInfo(userID uint, charID int64) (apicontract.FleetCharacterInfo, *fleetOperationFailure) {
	info, err := h.svc.GetCharacterFleetInfo(userID, charID)
	if err != nil {
		return apicontract.FleetCharacterInfo{}, &fleetOperationFailure{code: response.CodeBizError, message: err.Error()}
	}
	return apicontract.FleetCharacterInfo{
		FleetID: info.FleetID, FleetBossID: info.FleetBossID, Role: info.Role, SquadID: info.SquadID, WingID: info.WingID,
	}, nil
}

func (h *FleetHandler) pingFleet(fleetID string, userID uint, userRoles []string) *fleetOperationFailure {
	if err := h.svc.PingFleet(fleetID, userID, userRoles); err != nil {
		return &fleetOperationFailure{code: response.CodeBizError, message: err.Error()}
	}
	return nil
}

func fleetValue[T any](p *T) T {
	if p != nil {
		return *p
	}
	var zero T
	return zero
}

func fleetValueOr[T any](p *T, fallback T) T {
	if p == nil {
		return fallback
	}
	return *p
}

// Preserve nil lists as JSON null and non-nil empty lists as [].
func fleetDTOs[M, D any](items []M, convert func(M) D) []D {
	if items == nil {
		return nil
	}
	out := make([]D, len(items))
	for i, item := range items {
		out[i] = convert(item)
	}
	return out
}

func fleetNullableDTOs[M, D any](items []M, convert func(M) D) *[]D {
	out := fleetDTOs(items, convert)
	if out == nil {
		return nil
	}
	return &out
}

func fleetRecordDTO(f model.Fleet) apicontract.FleetRecord {
	return apicontract.FleetRecord{
		ID: f.ID, Title: f.Title, Description: f.Description, StartAt: f.StartAt, EndAt: f.EndAt,
		Importance: f.Importance, PapCount: f.PapCount, FCUserID: f.FCUserID,
		FCCharacterID: f.FCCharacterID, FCCharacterName: f.FCCharacterName,
		ESIFleetID: f.ESIFleetID, FleetConfigID: f.FleetConfigID, AutoSrpMode: apicontract.FleetRecordAutoSrpMode(f.AutoSrpMode),
		CreatedAt: f.CreatedAt, UpdatedAt: f.UpdatedAt, DeletedAt: f.DeletedAt,
	}
}

func fleetListItemDTO(f model.FleetListItem) apicontract.FleetListItem {
	var displayName *string
	if f.FCDisplayName != "" {
		displayName = &f.FCDisplayName
	}
	return apicontract.FleetListItem{
		ID: f.ID, Title: f.Title, Description: f.Description, StartAt: f.StartAt, EndAt: f.EndAt,
		Importance: f.Importance, PapCount: f.PapCount, FCUserID: f.FCUserID,
		FCCharacterID: f.FCCharacterID, FCCharacterName: f.FCCharacterName, FCDisplayName: displayName,
		ESIFleetID: f.ESIFleetID, FleetConfigID: f.FleetConfigID, AutoSrpMode: apicontract.FleetListItemAutoSrpMode(f.AutoSrpMode),
		CreatedAt: f.CreatedAt, UpdatedAt: f.UpdatedAt,
	}
}

func fleetMemberDTO(m model.FleetMember) apicontract.FleetMember {
	return apicontract.FleetMember{
		ID: m.ID, FleetID: m.FleetID, CharacterID: m.CharacterID, CharacterName: m.CharacterName, UserID: m.UserID,
		ShipTypeID: m.ShipTypeID, SolarSystemID: m.SolarSystemID, KMRefreshedAt: m.KMRefreshedAt, JoinedAt: m.JoinedAt,
	}
}

func fleetMemberWithPapDTO(m service.MemberWithPap) apicontract.FleetMemberWithPap {
	return apicontract.FleetMemberWithPap{
		ID: m.ID, FleetID: m.FleetID, CharacterID: m.CharacterID, CharacterName: m.CharacterName, UserID: m.UserID,
		ShipTypeID: m.ShipTypeID, SolarSystemID: m.SolarSystemID, KMRefreshedAt: m.KMRefreshedAt, JoinedAt: m.JoinedAt,
		PapCount: m.PapCount, IssuedAt: m.IssuedAt,
	}
}

func fleetPapLogDTO(p model.FleetPapLog) apicontract.FleetPapLog {
	return apicontract.FleetPapLog{
		ID: p.ID, FleetID: p.FleetID, CharacterID: p.CharacterID, UserID: p.UserID,
		PapCount: p.PapCount, IssuedBy: p.IssuedBy, IssuedAt: p.IssuedAt,
	}
}

func fleetPapLogDetailDTO(p service.PapLogDetail) apicontract.FleetPapLogDetail {
	return apicontract.FleetPapLogDetail{
		ID: p.ID, FleetID: p.FleetID, CharacterID: p.CharacterID, UserID: p.UserID,
		PapCount: p.PapCount, IssuedBy: p.IssuedBy, IssuedAt: p.IssuedAt,
		CharacterName: p.CharacterName, FleetTitle: p.FleetTitle, FleetStartAt: p.FleetStartAt,
		FCCharacterName: p.FCCharacterName, FleetImportance: p.FleetImportance, ShipTypeID: p.ShipTypeID,
	}
}

func fleetCorporationPapPageDTO(p service.CorporationPapSummaryResponse) apicontract.FleetCorporationPapPage {
	return apicontract.FleetCorporationPapPage{
		List: fleetDTOs(p.List, func(item service.CorporationPapSummaryItem) apicontract.FleetCorporationPapItem {
			return apicontract.FleetCorporationPapItem{
				UserID: item.UserID, Nickname: item.Nickname, CorpTicker: item.CorpTicker,
				MainCharacterName: item.MainCharacterName, CharacterCount: item.CharacterCount,
				StratOpPaps: item.StratOpPaps, SkirmishPaps: item.SkirmishPaps, AllianceStratPaps: item.AllianceStratPaps,
			}
		}),
		Total: p.Total, Page: p.Page, PageSize: p.PageSize,
		Overview: apicontract.FleetCorporationPapOverview{
			FilteredPapTotal: p.Overview.FilteredPapTotal, FilteredStratOpTotal: p.Overview.FilteredStratOpTotal,
			AllPapTotal: p.Overview.AllPapTotal, FilteredUserCount: p.Overview.FilteredUserCount,
			Period: apicontract.FleetCorporationPapOverviewPeriod(p.Overview.Period), Year: p.Overview.Year,
		},
	}
}

func fleetInviteDTO(i model.FleetInvite) apicontract.FleetInvite {
	return apicontract.FleetInvite{
		ID: i.ID, FleetID: i.FleetID, Code: i.Code, Active: i.Active, CreatedAt: i.CreatedAt, ExpiresAt: i.ExpiresAt,
	}
}

func fleetAlliancePAPDTO(result service.AlliancePAPResult) apicontract.FleetAlliancePAPData {
	data := apicontract.FleetAlliancePAPData{Fleets: fleetNullableDTOs(result.Fleets, fleetAlliancePAPRecordDTO)}
	if result.Summary != nil {
		summary := alliancePAPSummaryDTO(*result.Summary)
		data.Summary = &summary
	}
	return data
}

func alliancePAPSummaryDTO(s model.AlliancePAPSummary) apicontract.FleetAlliancePAPSummary {
	return apicontract.FleetAlliancePAPSummary{
		ID: s.ID, MainCharacter: s.MainCharacter, Year: s.Year, Month: s.Month, CorporationID: s.CorporationID,
		TotalPap: s.TotalPap, YearlyTotalPap: s.YearlyTotalPap, MonthlyRank: s.MonthlyRank, YearlyRank: s.YearlyRank,
		GlobalMonthlyRank: s.GlobalMonthlyRank, GlobalYearlyRank: s.GlobalYearlyRank, TotalInCorp: s.TotalInCorp,
		TotalGlobal: s.TotalGlobal, CalculatedAt: s.CalculatedAt, IsArchived: s.IsArchived, IsRedeemed: s.IsRedeemed,
		WalletIssued: s.WalletIssued, CreatedAt: s.CreatedAt, UpdatedAt: s.UpdatedAt,
	}
}

func fleetAlliancePAPRecordDTO(r model.AlliancePAPRecord) apicontract.FleetAlliancePAPRecord {
	return apicontract.FleetAlliancePAPRecord{
		ID: r.ID, MainCharacter: r.MainCharacter, CharacterID: r.CharacterID, CharacterName: r.CharacterName,
		FleetID: r.FleetID, Year: r.Year, Month: r.Month, StartAt: r.StartAt, EndAt: r.EndAt,
		Title: r.Title, Level: r.Level, Pap: r.Pap, ShipGroupID: r.ShipGroupID, ShipGroupName: r.ShipGroupName,
		ShipTypeID: r.ShipTypeID, ShipTypeName: r.ShipTypeName, IsArchived: r.IsArchived,
		CreatedAt: r.CreatedAt, UpdatedAt: r.UpdatedAt,
	}
}

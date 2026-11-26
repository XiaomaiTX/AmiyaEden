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
)

// SrpHandler 补损 HTTP 处理器
type SrpHandler struct {
	svc *service.SrpService
}

func NewSrpHandler() *SrpHandler {
	return &SrpHandler{
		svc: service.NewSrpService(),
	}
}

func srpValue[T any](value *T) T {
	if value != nil {
		return *value
	}
	var zero T
	return zero
}

func srpMap[T, U any](values []T, convert func(T) U) []U {
	if values == nil {
		return nil
	}
	result := make([]U, len(values))
	for i, value := range values {
		result[i] = convert(value)
	}
	return result
}

func srpShipPriceDTO(price model.SrpShipPrice) apicontract.SrpShipPrice {
	return apicontract.SrpShipPrice{
		Id: price.ID, ShipTypeId: price.ShipTypeID, ShipName: price.ShipName, Amount: price.Amount,
		CreatedBy: price.CreatedBy, UpdatedBy: price.UpdatedBy, CreatedAt: price.CreatedAt, UpdatedAt: price.UpdatedAt,
	}
}

func srpApplicationDTO(app model.SrpApplication) apicontract.SrpApplication {
	return apicontract.SrpApplication{
		Id: app.ID, UserId: app.UserID, CharacterId: app.CharacterID, CharacterName: app.CharacterName,
		KillmailId: app.KillmailID, FleetId: app.FleetID, Note: app.Note, ShipTypeId: app.ShipTypeID,
		ShipName: app.ShipName, SolarSystemId: app.SolarSystemID, SolarSystemName: app.SolarSystemName,
		KillmailTime: app.KillmailTime, CorporationId: app.CorporationID, CorporationName: app.CorporationName,
		AllianceId: app.AllianceID, AllianceName: app.AllianceName, RecommendedAmount: app.RecommendedAmount,
		FinalAmount: app.FinalAmount, ReviewStatus: apicontract.SrpReviewStatus(app.ReviewStatus),
		ReviewedBy: app.ReviewedBy, ReviewedAt: app.ReviewedAt, ReviewNote: app.ReviewNote,
		PayoutStatus: apicontract.SrpPayoutStatus(app.PayoutStatus), PaidBy: app.PaidBy, PaidAt: app.PaidAt,
		CreatedAt: app.CreatedAt, UpdatedAt: app.UpdatedAt,
	}
}

func srpApplicationResponseDTO(app service.SrpApplicationResponse) apicontract.SrpApplication {
	dto := srpApplicationDTO(app.SrpApplication)
	dto.FleetTitle = app.FleetTitle
	dto.FleetFcName = app.FleetFCName
	dto.Nickname = app.Nickname
	dto.LastActorNickname = app.LastActorNickname
	return dto
}

func srpApplicationPageDTO(list []service.SrpApplicationResponse, total int64, page, pageSize int) apicontract.SrpApplicationPage {
	return apicontract.SrpApplicationPage{
		List: srpMap(list, srpApplicationResponseDTO), Total: total, Page: page, PageSize: pageSize,
	}
}

func srpBatchPayoutSummaryDTO(summary service.SrpBatchPayoutSummaryResponse) apicontract.SrpBatchPayoutSummary {
	return apicontract.SrpBatchPayoutSummary{
		UserId: summary.UserID, Nickname: summary.Nickname, MainCharacterId: summary.MainCharacterID,
		MainCharacterName: summary.MainCharacterName, TotalAmount: summary.TotalAmount, ApplicationCount: summary.ApplicationCount,
	}
}

func srpMailAttemptDTO(summary service.MailAttemptSummary) apicontract.SrpMailAttempt {
	return apicontract.SrpMailAttempt{
		MailError: summary.MailError, MailId: summary.MailID,
		MailSenderCharacterId: summary.MailSenderCharacterID, MailSenderCharacterName: summary.MailSenderCharacterName,
		MailRecipientCharacterId: summary.MailRecipientCharacterID, MailRecipientCharacterName: summary.MailRecipientCharacterName,
	}
}

func srpKillmailDTO(km service.FleetKillmailItem) apicontract.SrpKillmail {
	return apicontract.SrpKillmail{
		KillmailId: km.KillmailID, KillmailTime: km.KillmailTime, ShipTypeId: km.ShipTypeID,
		SolarSystemId: km.SolarSystemID, CharacterId: km.CharacterID, VictimName: km.VictimName,
	}
}

func srpKillmailDetailDTO(km service.KillmailDetailResponse) apicontract.SrpKillmailDetail {
	return apicontract.SrpKillmailDetail{
		KillmailId: km.KillmailID, KillmailTime: km.KillmailTime, ShipTypeId: km.ShipTypeID, ShipName: km.ShipName,
		SolarSystemId: km.SolarSystemID, SystemName: km.SystemName, CharacterId: km.CharacterID,
		CharacterName: km.CharacterName, JaniceAmount: km.JaniceAmount,
		Slots: srpMap(km.Slots, func(slot service.KillmailSlotGroup) apicontract.SrpKillmailSlotGroup {
			return apicontract.SrpKillmailSlotGroup{
				FlagId: slot.FlagID, FlagName: slot.FlagName, FlagText: slot.FlagText, OrderId: slot.OrderID,
				Items: srpMap(slot.Items, func(item service.KillmailSlotItem) apicontract.SrpKillmailSlotItem {
					return apicontract.SrpKillmailSlotItem{ItemId: item.ItemID, ItemName: item.ItemName, Quantity: item.Quantity, Dropped: item.Dropped}
				}),
			}
		}),
	}
}

func strictSrpActor(ctx context.Context) (uint, []string) {
	// The generated adapter passes the Gin context resolved by authentication.
	if c, ok := ctx.(*gin.Context); ok && c != nil {
		return middleware.GetUserID(c), middleware.GetUserRoles(c)
	}
	return 0, nil
}

func validSrpPathID(id uint) bool { return id > 0 && id <= math.MaxUint32 }

func (h *SrpHandler) ListSrpShipPrices(_ context.Context, request apicontract.ListSrpShipPricesRequestObject) (apicontract.ListSrpShipPricesResponseObject, error) {
	out := &apicontract.ListSrpShipPrices200JSONResponse{}
	data, failure := h.listSrpShipPrices(request.Params)
	if failure != nil {
		return out, out.FromBusinessError(contractBusinessError(failure.code, failure.message))
	}
	return out, out.FromSrpShipPricesSuccess(apicontract.SrpShipPricesSuccess{Code: response.CodeOK, Msg: "success", Data: data})
}

func (h *SrpHandler) UpsertSrpShipPrice(ctx context.Context, request apicontract.UpsertSrpShipPriceRequestObject) (apicontract.UpsertSrpShipPriceResponseObject, error) {
	out := &apicontract.UpsertSrpShipPrice200JSONResponse{}
	if request.Body == nil {
		return out, out.FromBusinessError(contractBusinessError(response.CodeParamError, "请求参数错误"))
	}
	actorID, _ := strictSrpActor(ctx)
	data, failure := h.upsertSrpShipPrice(actorID, *request.Body)
	if failure != nil {
		return out, out.FromBusinessError(contractBusinessError(failure.code, failure.message))
	}
	return out, out.FromSrpShipPriceSuccess(apicontract.SrpShipPriceSuccess{Code: response.CodeOK, Msg: "success", Data: data})
}

func (h *SrpHandler) DeleteSrpShipPrice(_ context.Context, request apicontract.DeleteSrpShipPriceRequestObject) (apicontract.DeleteSrpShipPriceResponseObject, error) {
	out := &apicontract.DeleteSrpShipPrice200JSONResponse{}
	if !validSrpPathID(request.Id) {
		return out, out.FromBusinessError(contractBusinessError(response.CodeParamError, "无效的 ID"))
	}
	if failure := h.deleteSrpShipPrice(request.Id); failure != nil {
		return out, out.FromBusinessError(contractBusinessError(failure.code, failure.message))
	}
	return out, out.FromNullSuccess(apicontract.NullSuccess{Code: response.CodeOK, Msg: "success"})
}

func (h *SrpHandler) GetSrpConfig(_ context.Context, _ apicontract.GetSrpConfigRequestObject) (apicontract.GetSrpConfigResponseObject, error) {
	out := &apicontract.GetSrpConfig200JSONResponse{}
	data := h.getSrpConfig()
	return out, out.FromSrpConfigSuccess(apicontract.SrpConfigSuccess{Code: response.CodeOK, Msg: "success", Data: data})
}

func (h *SrpHandler) UpdateSrpConfig(_ context.Context, request apicontract.UpdateSrpConfigRequestObject) (apicontract.UpdateSrpConfigResponseObject, error) {
	out := &apicontract.UpdateSrpConfig200JSONResponse{}
	if request.Body == nil {
		return out, out.FromBusinessError(contractBusinessError(response.CodeParamError, "请求参数错误"))
	}
	if failure := h.updateSrpConfig(*request.Body); failure != nil {
		return out, out.FromBusinessError(contractBusinessError(failure.code, failure.message))
	}
	return out, out.FromNullSuccess(apicontract.NullSuccess{Code: response.CodeOK, Msg: "success"})
}

func (h *SrpHandler) SubmitSrpApplication(ctx context.Context, request apicontract.SubmitSrpApplicationRequestObject) (apicontract.SubmitSrpApplicationResponseObject, error) {
	out := &apicontract.SubmitSrpApplication200JSONResponse{}
	if request.Body == nil {
		return out, out.FromBusinessError(contractBusinessError(response.CodeParamError, "请求参数错误"))
	}
	actorID, _ := strictSrpActor(ctx)
	data, failure := h.submitSrpApplication(actorID, *request.Body)
	if failure != nil {
		return out, out.FromBusinessError(contractBusinessError(failure.code, failure.message))
	}
	return out, out.FromSrpApplicationSuccess(apicontract.SrpApplicationSuccess{Code: response.CodeOK, Msg: "success", Data: data})
}

func (h *SrpHandler) ListMySrpApplications(ctx context.Context, request apicontract.ListMySrpApplicationsRequestObject) (apicontract.ListMySrpApplicationsResponseObject, error) {
	out := &apicontract.ListMySrpApplications200JSONResponse{}
	actorID, _ := strictSrpActor(ctx)
	data, failure := h.listMySrpApplications(actorID, request.Params)
	if failure != nil {
		return out, out.FromBusinessError(contractBusinessError(failure.code, failure.message))
	}
	return out, out.FromSrpApplicationPageSuccess(apicontract.SrpApplicationPageSuccess{Code: response.CodeOK, Msg: "success", Data: data})
}

func (h *SrpHandler) GetMySrpKillmails(ctx context.Context, request apicontract.GetMySrpKillmailsRequestObject) (apicontract.GetMySrpKillmailsResponseObject, error) {
	out := &apicontract.GetMySrpKillmails200JSONResponse{}
	actorID, _ := strictSrpActor(ctx)
	data, failure := h.getMySrpKillmails(actorID, request.Params)
	if failure != nil {
		return out, out.FromBusinessError(contractBusinessError(failure.code, failure.message))
	}
	return out, out.FromSrpKillmailsSuccess(apicontract.SrpKillmailsSuccess{Code: response.CodeOK, Msg: "success", Data: data})
}

func (h *SrpHandler) GetSrpFleetKillmails(ctx context.Context, request apicontract.GetSrpFleetKillmailsRequestObject) (apicontract.GetSrpFleetKillmailsResponseObject, error) {
	out := &apicontract.GetSrpFleetKillmails200JSONResponse{}
	if request.FleetId == "" {
		return out, out.FromBusinessError(contractBusinessError(response.CodeParamError, "缺少 fleet_id 参数"))
	}
	actorID, _ := strictSrpActor(ctx)
	data, failure := h.getSrpFleetKillmails(actorID, request.FleetId, request.Params)
	if failure != nil {
		return out, out.FromBusinessError(contractBusinessError(failure.code, failure.message))
	}
	return out, out.FromSrpKillmailsSuccess(apicontract.SrpKillmailsSuccess{Code: response.CodeOK, Msg: "success", Data: data})
}

func (h *SrpHandler) ListSrpApplications(_ context.Context, request apicontract.ListSrpApplicationsRequestObject) (apicontract.ListSrpApplicationsResponseObject, error) {
	out := &apicontract.ListSrpApplications200JSONResponse{}
	data, failure := h.listSrpApplications(request.Params)
	if failure != nil {
		return out, out.FromBusinessError(contractBusinessError(failure.code, failure.message))
	}
	return out, out.FromSrpApplicationPageSuccess(apicontract.SrpApplicationPageSuccess{Code: response.CodeOK, Msg: "success", Data: data})
}

func (h *SrpHandler) ListSrpFleetOptions(_ context.Context, _ apicontract.ListSrpFleetOptionsRequestObject) (apicontract.ListSrpFleetOptionsResponseObject, error) {
	out := &apicontract.ListSrpFleetOptions200JSONResponse{}
	data, failure := h.listSrpFleetOptions()
	if failure != nil {
		return out, out.FromBusinessError(contractBusinessError(failure.code, failure.message))
	}
	return out, out.FromSrpFleetOptionsSuccess(apicontract.SrpFleetOptionsSuccess{Code: response.CodeOK, Msg: "success", Data: data})
}

func (h *SrpHandler) ListSrpBatchPayoutSummary(_ context.Context, _ apicontract.ListSrpBatchPayoutSummaryRequestObject) (apicontract.ListSrpBatchPayoutSummaryResponseObject, error) {
	out := &apicontract.ListSrpBatchPayoutSummary200JSONResponse{}
	data, failure := h.listSrpBatchPayoutSummary()
	if failure != nil {
		return out, out.FromBusinessError(contractBusinessError(failure.code, failure.message))
	}
	return out, out.FromSrpBatchPayoutSummariesSuccess(apicontract.SrpBatchPayoutSummariesSuccess{Code: response.CodeOK, Msg: "success", Data: data})
}

func (h *SrpHandler) RunSrpFleetAutoApproval(ctx context.Context, request apicontract.RunSrpFleetAutoApprovalRequestObject) (apicontract.RunSrpFleetAutoApprovalResponseObject, error) {
	out := &apicontract.RunSrpFleetAutoApproval200JSONResponse{}
	if request.Body == nil {
		return out, out.FromBusinessError(contractBusinessError(response.CodeParamError, "请求参数错误"))
	}
	actorID, roles := strictSrpActor(ctx)
	data, failure := h.runSrpFleetAutoApproval(actorID, roles, *request.Body)
	if failure != nil {
		return out, out.FromBusinessError(contractBusinessError(failure.code, failure.message))
	}
	return out, out.FromSrpAutoApproveSuccess(apicontract.SrpAutoApproveSuccess{Code: response.CodeOK, Msg: "success", Data: data})
}

func (h *SrpHandler) GetSrpApplication(_ context.Context, request apicontract.GetSrpApplicationRequestObject) (apicontract.GetSrpApplicationResponseObject, error) {
	out := &apicontract.GetSrpApplication200JSONResponse{}
	if !validSrpPathID(request.Id) {
		return out, out.FromBusinessError(contractBusinessError(response.CodeParamError, "无效的 ID"))
	}
	data, failure := h.getSrpApplication(request.Id)
	if failure != nil {
		return out, out.FromBusinessError(contractBusinessError(failure.code, failure.message))
	}
	return out, out.FromSrpApplicationSuccess(apicontract.SrpApplicationSuccess{Code: response.CodeOK, Msg: "success", Data: data})
}

func (h *SrpHandler) ReviewSrpApplication(ctx context.Context, request apicontract.ReviewSrpApplicationRequestObject) (apicontract.ReviewSrpApplicationResponseObject, error) {
	out := &apicontract.ReviewSrpApplication200JSONResponse{}
	if !validSrpPathID(request.Id) {
		return out, out.FromBusinessError(contractBusinessError(response.CodeParamError, "无效的 ID"))
	}
	if request.Body == nil {
		return out, out.FromBusinessError(contractBusinessError(response.CodeParamError, "请求参数错误"))
	}
	actorID, roles := strictSrpActor(ctx)
	data, failure := h.reviewSrpApplication(request.Id, actorID, roles, *request.Body)
	if failure != nil {
		return out, out.FromBusinessError(contractBusinessError(failure.code, failure.message))
	}
	return out, out.FromSrpApplicationSuccess(apicontract.SrpApplicationSuccess{Code: response.CodeOK, Msg: "success", Data: data})
}

func (h *SrpHandler) PayoutSrpApplication(ctx context.Context, request apicontract.PayoutSrpApplicationRequestObject) (apicontract.PayoutSrpApplicationResponseObject, error) {
	out := &apicontract.PayoutSrpApplication200JSONResponse{}
	if !validSrpPathID(request.Id) {
		return out, out.FromBusinessError(contractBusinessError(response.CodeParamError, "无效的 ID"))
	}
	if request.Body == nil {
		return out, out.FromBusinessError(contractBusinessError(response.CodeParamError, "请求参数错误"))
	}
	actorID, roles := strictSrpActor(ctx)
	data, failure := h.payoutSrpApplication(request.Id, actorID, roles, *request.Body)
	if failure != nil {
		return out, out.FromBusinessError(contractBusinessError(failure.code, failure.message))
	}
	return out, out.FromSrpPayoutSuccess(apicontract.SrpPayoutSuccess{Code: response.CodeOK, Msg: "success", Data: data})
}

func (h *SrpHandler) BatchPayoutSrpByUser(ctx context.Context, request apicontract.BatchPayoutSrpByUserRequestObject) (apicontract.BatchPayoutSrpByUserResponseObject, error) {
	out := &apicontract.BatchPayoutSrpByUser200JSONResponse{}
	if !validSrpPathID(request.UserId) {
		return out, out.FromBusinessError(contractBusinessError(response.CodeParamError, "无效的用户 ID"))
	}
	actorID, roles := strictSrpActor(ctx)
	data, failure := h.batchPayoutSrpByUser(request.UserId, actorID, roles)
	if failure != nil {
		return out, out.FromBusinessError(contractBusinessError(failure.code, failure.message))
	}
	return out, out.FromSrpBatchPayoutSuccess(apicontract.SrpBatchPayoutSuccess{Code: response.CodeOK, Msg: "success", Data: data})
}

func (h *SrpHandler) BatchPayoutSrpAsFuxiCoin(ctx context.Context, _ apicontract.BatchPayoutSrpAsFuxiCoinRequestObject) (apicontract.BatchPayoutSrpAsFuxiCoinResponseObject, error) {
	out := &apicontract.BatchPayoutSrpAsFuxiCoin200JSONResponse{}
	actorID, roles := strictSrpActor(ctx)
	data, failure := h.batchPayoutSrpAsFuxiCoin(actorID, roles)
	if failure != nil {
		return out, out.FromBusinessError(contractBusinessError(failure.code, failure.message))
	}
	return out, out.FromSrpFuxiPayoutSuccess(apicontract.SrpFuxiPayoutSuccess{Code: response.CodeOK, Msg: "success", Data: data})
}

func (h *SrpHandler) OpenSrpInfoWindow(ctx context.Context, request apicontract.OpenSrpInfoWindowRequestObject) (apicontract.OpenSrpInfoWindowResponseObject, error) {
	out := &apicontract.OpenSrpInfoWindow200JSONResponse{}
	if request.Body == nil {
		return out, out.FromBusinessError(contractBusinessError(response.CodeParamError, "请求参数错误"))
	}
	actorID, _ := strictSrpActor(ctx)
	if failure := h.openSrpInfoWindow(actorID, *request.Body); failure != nil {
		return out, out.FromBusinessError(contractBusinessError(failure.code, failure.message))
	}
	return out, out.FromNullSuccess(apicontract.NullSuccess{Code: response.CodeOK, Msg: "success"})
}

func (h *SrpHandler) GetSrpKillmailDetail(_ context.Context, request apicontract.GetSrpKillmailDetailRequestObject) (apicontract.GetSrpKillmailDetailResponseObject, error) {
	out := &apicontract.GetSrpKillmailDetail200JSONResponse{}
	if request.Body == nil {
		return out, out.FromBusinessError(contractBusinessError(response.CodeParamError, "请求参数错误"))
	}
	data, failure := h.getSrpKillmailDetail(*request.Body)
	if failure != nil {
		return out, out.FromBusinessError(contractBusinessError(failure.code, failure.message))
	}
	return out, out.FromSrpKillmailDetailSuccess(apicontract.SrpKillmailDetailSuccess{Code: response.CodeOK, Msg: "success", Data: data})
}

func (h *SrpHandler) listSrpShipPrices(params apicontract.ListSrpShipPricesParams) ([]apicontract.SrpShipPrice, *contractOperationFailure) {
	keyword := srpValue(params.Keyword)
	list, err := h.svc.ListShipPrices(keyword)
	if err != nil {
		return nil, &contractOperationFailure{code: response.CodeBizError, message: err.Error()}
	}
	return srpMap(list, srpShipPriceDTO), nil
}

func (h *SrpHandler) upsertSrpShipPrice(userID uint, req apicontract.SrpShipPriceInput) (apicontract.SrpShipPrice, *contractOperationFailure) {
	p, err := h.svc.UpsertShipPrice(userID, &service.UpsertShipPriceRequest{
		ID: req.Id, ShipTypeID: req.ShipTypeID, ShipName: req.ShipName, Amount: req.Amount,
	})
	if err != nil {
		return apicontract.SrpShipPrice{}, &contractOperationFailure{code: response.CodeBizError, message: err.Error()}
	}
	return srpShipPriceDTO(*p), nil
}

func (h *SrpHandler) deleteSrpShipPrice(id uint) *contractOperationFailure {
	if err := h.svc.DeleteShipPrice(id); err != nil {
		return &contractOperationFailure{code: response.CodeBizError, message: err.Error()}
	}
	return nil
}

func (h *SrpHandler) getSrpConfig() apicontract.SrpConfig {
	amountLimit := h.svc.GetSrpAmountLimit()
	return apicontract.SrpConfig{AmountLimit: amountLimit}
}

func (h *SrpHandler) updateSrpConfig(req apicontract.SrpConfigInput) *contractOperationFailure {
	if req.AmountLimit == nil {
		return &contractOperationFailure{code: response.CodeParamError, message: "请求参数错误"}
	}
	if err := h.svc.UpdateSrpAmountLimit(*req.AmountLimit); err != nil {
		return &contractOperationFailure{code: response.CodeBizError, message: "更新 SRP 配置失败"}
	}
	return nil
}

func (h *SrpHandler) submitSrpApplication(userID uint, req apicontract.SrpSubmitInput) (apicontract.SrpApplication, *contractOperationFailure) {
	app, err := h.svc.SubmitApplication(userID, &service.SubmitApplicationRequest{
		CharacterID: req.CharacterID, KillmailID: req.KillmailID, FleetID: req.FleetId, Note: req.Note,
	})
	if err != nil {
		return apicontract.SrpApplication{}, &contractOperationFailure{code: response.CodeBizError, message: err.Error()}
	}
	return srpApplicationDTO(*app), nil
}

func (h *SrpHandler) listMySrpApplications(userID uint, params apicontract.ListMySrpApplicationsParams) (apicontract.SrpApplicationPage, *contractOperationFailure) {
	page, pageSize := normalizePagination(srpValue(params.Current), srpValue(params.Size), 20, 100)

	list, total, err := h.svc.ListMyApplications(userID, page, pageSize)
	if err != nil {
		return apicontract.SrpApplicationPage{}, &contractOperationFailure{code: response.CodeBizError, message: err.Error()}
	}
	return srpApplicationPageDTO(list, total, page, pageSize), nil
}

func (h *SrpHandler) getMySrpKillmails(userID uint, params apicontract.GetMySrpKillmailsParams) ([]apicontract.SrpKillmail, *contractOperationFailure) {
	options := srpKillmailOptions(srpValue(params.Limit), srpValue(params.ExcludeSubmitted))
	kms, err := h.svc.GetMyKillmails(userID, srpValue(params.CharacterId), options)
	if err != nil {
		return nil, &contractOperationFailure{code: response.CodeBizError, message: err.Error()}
	}
	return srpMap(kms, srpKillmailDTO), nil
}

func (h *SrpHandler) getSrpFleetKillmails(userID uint, fleetID string, params apicontract.GetSrpFleetKillmailsParams) ([]apicontract.SrpKillmail, *contractOperationFailure) {
	options := srpKillmailOptions(srpValue(params.Limit), srpValue(params.ExcludeSubmitted))
	kms, err := h.svc.GetFleetKillmails(userID, fleetID, options)
	if err != nil {
		return nil, &contractOperationFailure{code: response.CodeBizError, message: err.Error()}
	}
	return srpMap(kms, srpKillmailDTO), nil
}

func (h *SrpHandler) listSrpApplications(params apicontract.ListSrpApplicationsParams) (apicontract.SrpApplicationPage, *contractOperationFailure) {
	page := normalizePage(srpValue(params.Current))
	pageSize := srpValue(params.Size)
	if pageSize < 1 {
		pageSize = 20
	} else {
		pageSize = normalizeLedgerPageSize(pageSize)
	}
	// The legacy binder treats an explicitly empty fleet filter as absent.
	if srpValue(params.FleetId) == "" {
		params.FleetId = nil
	}
	filter := service.SrpApplicationFilter{
		Tab: srpValue(params.Tab), ReviewStatus: srpValue(params.ReviewStatus), PayoutStatus: srpValue(params.PayoutStatus),
		Keyword: srpValue(params.Keyword), SortBy: srpValue(params.SortBy), SortOrder: srpValue(params.SortOrder),
		FleetID: params.FleetId, CharacterID: params.CharacterId, CorporationID: params.CorporationId,
		ShipTypeID: params.ShipTypeId, SolarSystemID: params.SolarSystemId, HasRecommendedMatch: params.HasRecommendedMatch,
	}
	list, total, err := h.svc.ListApplications(page, pageSize, filter)
	if err != nil {
		return apicontract.SrpApplicationPage{}, &contractOperationFailure{code: response.CodeBizError, message: err.Error()}
	}
	return srpApplicationPageDTO(list, total, page, pageSize), nil
}

func (h *SrpHandler) listSrpFleetOptions() ([]apicontract.SrpFleetOption, *contractOperationFailure) {
	list, err := h.svc.ListFleetOptions()
	if err != nil {
		return nil, &contractOperationFailure{code: response.CodeBizError, message: err.Error()}
	}
	return srpMap(list, func(option service.SrpFleetOptionResponse) apicontract.SrpFleetOption {
		return apicontract.SrpFleetOption{FleetId: option.FleetID, FleetTitle: option.FleetTitle, FleetFcName: option.FleetFCName}
	}), nil
}

func (h *SrpHandler) listSrpBatchPayoutSummary() ([]apicontract.SrpBatchPayoutSummary, *contractOperationFailure) {
	list, err := h.svc.ListBatchPayoutSummary()
	if err != nil {
		return nil, &contractOperationFailure{code: response.CodeBizError, message: err.Error()}
	}
	return srpMap(list, srpBatchPayoutSummaryDTO), nil
}

func (h *SrpHandler) runSrpFleetAutoApproval(reviewerID uint, callerRoles []string, req apicontract.SrpAutoApproveInput) (apicontract.SrpAutoApproveSummary, *contractOperationFailure) {
	result, err := h.svc.RunFleetAutoApproval(reviewerID, callerRoles, req.FleetID)
	if err != nil {
		return apicontract.SrpAutoApproveSummary{}, &contractOperationFailure{code: response.CodeBizError, message: err.Error()}
	}
	return apicontract.SrpAutoApproveSummary{
		CheckedCount: result.CheckedCount, ApprovedCount: result.ApprovedCount, SkippedCount: result.SkippedCount,
	}, nil
}

func (h *SrpHandler) getSrpApplication(id uint) (apicontract.SrpApplication, *contractOperationFailure) {
	app, err := h.svc.GetApplication(id)
	if err != nil {
		return apicontract.SrpApplication{}, &contractOperationFailure{code: response.CodeNotFound, message: "申请不存在"}
	}
	return srpApplicationResponseDTO(*app), nil
}

func (h *SrpHandler) reviewSrpApplication(id, reviewerID uint, callerRoles []string, req apicontract.SrpReviewInput) (apicontract.SrpApplication, *contractOperationFailure) {
	app, err := h.svc.ReviewApplication(reviewerID, callerRoles, id, &service.ReviewApplicationRequest{
		Action: string(req.Action), ReviewNote: req.ReviewNote, FinalAmount: req.FinalAmount,
	})
	if err != nil {
		return apicontract.SrpApplication{}, &contractOperationFailure{code: response.CodeBizError, message: err.Error()}
	}
	return srpApplicationDTO(*app), nil
}

func (h *SrpHandler) payoutSrpApplication(id, payerID uint, callerRoles []string, req apicontract.SrpPayoutInput) (apicontract.SrpPayoutResult, *contractOperationFailure) {
	app, mailSummary, err := h.svc.Payout(payerID, callerRoles, id, &service.SrpPayoutRequest{FinalAmount: req.FinalAmount, Mode: string(req.Mode)})
	if err != nil {
		return apicontract.SrpPayoutResult{}, &contractOperationFailure{code: response.CodeBizError, message: err.Error()}
	}
	return srpPayoutResultDTO(srpApplicationDTO(*app), srpMailAttemptDTO(mailSummary)), nil
}

func (h *SrpHandler) batchPayoutSrpByUser(userID, payerID uint, callerRoles []string) (apicontract.SrpBatchPayoutResult, *contractOperationFailure) {
	result, mailSummary, err := h.svc.BatchPayoutByUser(payerID, callerRoles, userID)
	if err != nil {
		return apicontract.SrpBatchPayoutResult{}, &contractOperationFailure{code: response.CodeBizError, message: err.Error()}
	}
	return srpBatchPayoutResultDTO(srpBatchPayoutSummaryDTO(*result), srpMailAttemptDTO(mailSummary)), nil
}

func (h *SrpHandler) batchPayoutSrpAsFuxiCoin(payerID uint, callerRoles []string) (apicontract.SrpFuxiPayoutResult, *contractOperationFailure) {
	result, mailSummary, err := h.svc.BatchPayoutAsFuxiCoin(payerID, callerRoles)
	if err != nil {
		return apicontract.SrpFuxiPayoutResult{}, &contractOperationFailure{code: response.CodeBizError, message: err.Error()}
	}
	return srpFuxiPayoutResultDTO(apicontract.SrpFuxiPayoutSummary{
		ApplicationCount: result.ApplicationCount, UserCount: result.UserCount,
		TotalIskAmount: result.TotalISKAmount, TotalFuxiCoin: result.TotalFuxiCoin,
	}, srpMailAttemptDTO(mailSummary)), nil
}

func (h *SrpHandler) openSrpInfoWindow(userID uint, req apicontract.SrpOpenInfoInput) *contractOperationFailure {
	if err := h.svc.OpenInfoWindow(userID, &service.OpenInfoWindowRequest{CharacterID: req.CharacterID, TargetID: req.TargetID}); err != nil {
		return &contractOperationFailure{code: response.CodeBizError, message: err.Error()}
	}
	return nil
}

func (h *SrpHandler) getSrpKillmailDetail(req apicontract.SrpKillmailDetailInput) (apicontract.SrpKillmailDetail, *contractOperationFailure) {
	detail, err := h.svc.GetKillmailDetail(&service.KillmailDetailRequest{KillmailID: req.KillmailID, Language: req.Language})
	if err != nil {
		return apicontract.SrpKillmailDetail{}, &contractOperationFailure{code: response.CodeBizError, message: err.Error()}
	}
	return srpKillmailDetailDTO(*detail), nil
}

func srpKillmailOptions(limit int, excludeSubmitted bool) service.KillmailListOptions {
	if limit < 1 || limit > 200 {
		limit = 200
	}
	return service.KillmailListOptions{Limit: limit, ExcludeSubmitted: excludeSubmitted}
}

func srpPayoutResultDTO(data apicontract.SrpApplication, mail apicontract.SrpMailAttempt) apicontract.SrpPayoutResult {
	return apicontract.SrpPayoutResult{
		AllianceId:                 data.AllianceId,
		AllianceName:               data.AllianceName,
		CharacterId:                data.CharacterId,
		CharacterName:              data.CharacterName,
		CorporationId:              data.CorporationId,
		CorporationName:            data.CorporationName,
		CreatedAt:                  data.CreatedAt,
		FinalAmount:                data.FinalAmount,
		FleetFcName:                data.FleetFcName,
		FleetId:                    data.FleetId,
		FleetTitle:                 data.FleetTitle,
		Id:                         data.Id,
		KillmailId:                 data.KillmailId,
		KillmailTime:               data.KillmailTime,
		LastActorNickname:          data.LastActorNickname,
		Nickname:                   data.Nickname,
		Note:                       data.Note,
		PaidAt:                     data.PaidAt,
		PaidBy:                     data.PaidBy,
		PayoutStatus:               data.PayoutStatus,
		RecommendedAmount:          data.RecommendedAmount,
		ReviewNote:                 data.ReviewNote,
		ReviewStatus:               data.ReviewStatus,
		ReviewedAt:                 data.ReviewedAt,
		ReviewedBy:                 data.ReviewedBy,
		ShipName:                   data.ShipName,
		ShipTypeId:                 data.ShipTypeId,
		SolarSystemId:              data.SolarSystemId,
		SolarSystemName:            data.SolarSystemName,
		UpdatedAt:                  data.UpdatedAt,
		UserId:                     data.UserId,
		MailError:                  mail.MailError,
		MailId:                     mail.MailId,
		MailRecipientCharacterId:   mail.MailRecipientCharacterId,
		MailRecipientCharacterName: mail.MailRecipientCharacterName,
		MailSenderCharacterId:      mail.MailSenderCharacterId,
		MailSenderCharacterName:    mail.MailSenderCharacterName,
	}
}

func srpBatchPayoutResultDTO(data apicontract.SrpBatchPayoutSummary, mail apicontract.SrpMailAttempt) apicontract.SrpBatchPayoutResult {
	return apicontract.SrpBatchPayoutResult{
		ApplicationCount:           data.ApplicationCount,
		MainCharacterId:            data.MainCharacterId,
		MainCharacterName:          data.MainCharacterName,
		Nickname:                   data.Nickname,
		TotalAmount:                data.TotalAmount,
		UserId:                     data.UserId,
		MailError:                  mail.MailError,
		MailId:                     mail.MailId,
		MailRecipientCharacterId:   mail.MailRecipientCharacterId,
		MailRecipientCharacterName: mail.MailRecipientCharacterName,
		MailSenderCharacterId:      mail.MailSenderCharacterId,
		MailSenderCharacterName:    mail.MailSenderCharacterName,
	}
}

func srpFuxiPayoutResultDTO(data apicontract.SrpFuxiPayoutSummary, mail apicontract.SrpMailAttempt) apicontract.SrpFuxiPayoutResult {
	return apicontract.SrpFuxiPayoutResult{
		ApplicationCount:           data.ApplicationCount,
		TotalFuxiCoin:              data.TotalFuxiCoin,
		TotalIskAmount:             data.TotalIskAmount,
		UserCount:                  data.UserCount,
		MailError:                  mail.MailError,
		MailId:                     mail.MailId,
		MailRecipientCharacterId:   mail.MailRecipientCharacterId,
		MailRecipientCharacterName: mail.MailRecipientCharacterName,
		MailSenderCharacterId:      mail.MailSenderCharacterId,
		MailSenderCharacterName:    mail.MailSenderCharacterName,
	}
}

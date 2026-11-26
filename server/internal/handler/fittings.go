package handler

import (
	"amiya-eden/global"
	"amiya-eden/internal/apicontract"
	"amiya-eden/internal/service"
	"amiya-eden/pkg/response"
	"context"
	"errors"

	"go.uber.org/zap"
)

// FittingsHandler 装配处理器
type FittingsHandler struct {
	svc *service.FittingsService
}

func NewFittingsHandler() *FittingsHandler {
	return &FittingsHandler{
		svc: service.NewFittingsService(),
	}
}

func infoFittingDTO(value service.FittingResponse) apicontract.InfoFitting {
	return apicontract.InfoFitting{
		FittingID: value.FittingID, CharacterID: value.CharacterID, Name: value.Name,
		Description: value.Description, ShipTypeID: value.ShipTypeID, ShipName: value.ShipName,
		GroupID: value.GroupID, GroupName: value.GroupName, RaceID: value.RaceID, RaceName: value.RaceName,
		Slots: mapEveInfoRows(value.Slots, func(slot service.FittingSlotGroup) apicontract.InfoFittingSlot {
			return apicontract.InfoFittingSlot{
				FlagName: slot.FlagName, FlagText: slot.FlagText, OrderID: slot.OrderID,
				Items: mapEveInfoRows(slot.Items, func(item service.FittingItemResponse) apicontract.InfoFittingItem {
					return apicontract.InfoFittingItem(item)
				}),
			}
		}),
	}
}

func mapEveInfoRows[S, T any](rows []S, mapper func(S) T) []T {
	if rows == nil {
		return nil
	}
	result := make([]T, len(rows))
	for i, row := range rows {
		result[i] = mapper(row)
	}
	return result
}

func (h *FittingsHandler) GetFittings(ctx context.Context, request apicontract.GetFittingsRequestObject) (apicontract.GetFittingsResponseObject, error) {
	out := &apicontract.GetFittings200JSONResponse{}
	if request.Body == nil {
		return out, out.FromBusinessError(contractBusinessError(response.CodeParamError, "参数错误"))
	}
	_, userID := strictGinRequestDetails(ctx)
	result, err := h.svc.GetFittings(userID, &service.FittingsRequest{Language: request.Body.Language})
	if err != nil {
		return out, out.FromBusinessError(contractBusinessError(response.CodeBizError, err.Error()))
	}
	data := apicontract.InfoFittings{Total: result.Total, Fittings: mapEveInfoRows(result.Fittings, infoFittingDTO)}
	return out, out.FromInfoFittingsSuccess(apicontract.InfoFittingsSuccess{Code: response.CodeOK, Msg: "success", Data: data})
}

func (h *FittingsHandler) SaveFitting(ctx context.Context, request apicontract.SaveFittingRequestObject) (apicontract.SaveFittingResponseObject, error) {
	out := &apicontract.SaveFitting200JSONResponse{}
	if request.Body == nil {
		return out, out.FromBusinessError(contractBusinessError(response.CodeParamError, "装配参数无效"))
	}
	requestCtx, userID := strictGinRequestDetails(ctx)
	body := request.Body
	input := service.SaveFittingRequest{
		CharacterID: body.CharacterID, FittingID: body.FittingID, Name: body.Name,
		Description: body.Description, ShipTypeID: body.ShipTypeID,
		Items: mapEveInfoRows(body.Items, func(item apicontract.SaveFittingItem) service.SaveFittingItem { return service.SaveFittingItem(item) }),
	}
	data, err := h.svc.SaveFitting(requestCtx, userID, &input)
	if err != nil {
		return out, out.FromBusinessError(contractBusinessError(response.CodeBizError, fittingSaveErrorMessage(err)))
	}
	return out, out.FromInfoFittingSuccess(apicontract.InfoFittingSuccess{Code: response.CodeOK, Msg: "success", Data: infoFittingDTO(*data)})
}

func fittingSaveErrorMessage(err error) string {
	if errors.Is(err, service.ErrFittingTokenUnavailable) {
		return service.ErrFittingTokenUnavailable.Error()
	}
	if logger := global.CurrentLogger(); logger != nil {
		logger.Error("[Fittings] 保存装配失败", zap.Error(err))
	}
	return "保存装配失败，请检查装配数据和人物授权后重试"
}

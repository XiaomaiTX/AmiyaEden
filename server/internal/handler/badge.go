package handler

import (
	"amiya-eden/internal/apicontract"
	"amiya-eden/internal/middleware"
	"amiya-eden/internal/service"
	"amiya-eden/pkg/response"
	"context"
	"github.com/gin-gonic/gin"
)

type BadgeHandler struct {
	svc *service.BadgeService
}

func NewBadgeHandler() *BadgeHandler {
	return &BadgeHandler{svc: service.NewBadgeService()}
}

func (h *BadgeHandler) GetBadgeCounts(ctx context.Context, _ apicontract.GetBadgeCountsRequestObject) (apicontract.GetBadgeCountsResponseObject, error) {
	c, _ := ctx.(*gin.Context)
	if c == nil {
		return apicontract.GetBadgeCounts401JSONResponse{UnauthorizedJSONResponse: apicontract.UnauthorizedJSONResponse{Code: response.CodeUnauthorized, Msg: "未登录"}}, nil
	}
	var out apicontract.GetBadgeCounts200JSONResponse
	counts, err := h.svc.GetBadgeCounts(middleware.GetUserID(c), middleware.GetUserRoles(c))
	if err != nil {
		return &out, out.FromBusinessError(contractBusinessError(response.CodeBizError, err.Error()))
	}
	return &out, out.FromBadgeCountsSuccess(apicontract.BadgeCountsSuccess{Code: response.CodeOK, Msg: "success", Data: apicontract.BadgeCounts(counts)})
}

package handler

import (
	"amiya-eden/internal/apicontract"
	"amiya-eden/internal/middleware"
	"amiya-eden/internal/service"
	"amiya-eden/pkg/response"
	"context"
	"errors"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

// blacklistRefreshTimeout bounds one synchronous refresh (download plus ESI
// lookups); the frontends wait up to 120 s for the response.
const blacklistRefreshTimeout = 90 * time.Second

type BlacklistHandler struct {
	svc *service.BlacklistService
}

func NewBlacklistHandler() *BlacklistHandler {
	return &BlacklistHandler{svc: service.NewBlacklistService()}
}

func (h *BlacklistHandler) listBlacklist(params apicontract.ListBlacklistParams) (apicontract.BlacklistPage, error) {
	page, size := 1, 20
	if params.Current != nil {
		page = *params.Current
	}
	if params.Size != nil {
		size = *params.Size
	}
	page, size = normalizePagination(page, size, 20, 100)
	var kind, keyword string
	if params.Kind != nil {
		kind = strings.TrimSpace(string(*params.Kind))
	}
	if params.Keyword != nil {
		keyword = *params.Keyword
	}
	rows, total, err := h.svc.ListEntries(page, size, kind, keyword)
	if err != nil {
		return apicontract.BlacklistPage{}, err
	}
	list := make([]apicontract.BlacklistEntry, 0, len(rows))
	for _, row := range rows {
		list = append(list, apicontract.BlacklistEntry{
			Kind: apicontract.BlacklistKind(row.Kind), Value: row.Value, Display: row.Display,
			Reason: row.Reason, UpdatedAt: row.UpdatedAt,
		})
	}
	return apicontract.BlacklistPage{List: list, Total: total, Page: page, PageSize: size}, nil
}

func (h *BlacklistHandler) refreshBlacklist(ctx context.Context, operatorID uint) (*apicontract.BlacklistRefreshSummary, error) {
	ctx, cancel := context.WithTimeout(ctx, blacklistRefreshTimeout)
	defer cancel()
	summary, err := h.svc.Refresh(ctx, operatorID)
	if err != nil {
		return nil, err
	}
	return blacklistSummary(summary), nil
}

func blacklistSummary(summary *service.BlacklistRefreshSummary) *apicontract.BlacklistRefreshSummary {
	if summary == nil {
		return nil
	}
	return &apicontract.BlacklistRefreshSummary{
		FinishedAt: summary.FinishedAt, OperatorUserId: summary.OperatorUserID, OperatorName: summary.OperatorName,
		Characters: summary.Characters, Qqs: summary.QQs, Discords: summary.Discords,
		Added: summary.Added, Updated: summary.Updated, Removed: summary.Removed,
		NewUnresolved: summary.NewUnresolved, SkippedUnresolved: summary.SkippedUnresolved,
		LookupFailed: summary.LookupFailed, NextRefreshAt: summary.NextRefreshAt,
	}
}

var blacklistUserErrors = []error{
	service.ErrBlacklistSourceUnavailable,
	service.ErrBlacklistSourceLayoutChanged,
	service.ErrBlacklistEmptyResult,
	service.ErrBlacklistRefreshInProgress,
	service.ErrBlacklistRefreshCooldown,
}

func blacklistError(err error) (int, string) {
	if errors.Is(err, service.ErrBlacklistInvalidQuery) {
		return response.CodeParamError, service.ErrBlacklistInvalidQuery.Error()
	}
	for _, known := range blacklistUserErrors {
		if errors.Is(err, known) {
			return response.CodeBizError, known.Error()
		}
	}
	return response.CodeBizError, service.ErrBlacklistInternal.Error()
}

func (h *BlacklistHandler) ListBlacklist(_ context.Context, request apicontract.ListBlacklistRequestObject) (apicontract.ListBlacklistResponseObject, error) {
	var out apicontract.ListBlacklist200JSONResponse
	data, err := h.listBlacklist(request.Params)
	if err != nil {
		code, message := blacklistError(err)
		return &out, out.FromBusinessError(contractBusinessError(code, message))
	}
	return &out, out.FromBlacklistPageSuccess(apicontract.BlacklistPageSuccess{Code: response.CodeOK, Msg: "success", Data: data})
}

func (h *BlacklistHandler) LatestRefresh(_ context.Context, _ apicontract.LatestRefreshRequestObject) (apicontract.LatestRefreshResponseObject, error) {
	var out apicontract.LatestRefresh200JSONResponse
	summary, err := h.svc.LatestSummary()
	if err != nil {
		code, message := blacklistError(err)
		return &out, out.FromBusinessError(contractBusinessError(code, message))
	}
	if summary == nil {
		return &out, out.FromNullSuccess(apicontract.NullSuccess{Code: response.CodeOK, Msg: "success"})
	}
	return &out, out.FromBlacklistRefreshSuccess(apicontract.BlacklistRefreshSuccess{Code: response.CodeOK, Msg: "success", Data: *blacklistSummary(summary)})
}

func (h *BlacklistHandler) Refresh(ctx context.Context, _ apicontract.RefreshRequestObject) (apicontract.RefreshResponseObject, error) {
	c, _ := ctx.(*gin.Context)
	if c == nil {
		return apicontract.Refresh401JSONResponse{UnauthorizedJSONResponse: apicontract.UnauthorizedJSONResponse{Code: response.CodeUnauthorized, Msg: "未登录"}}, nil
	}
	var out apicontract.Refresh200JSONResponse
	summary, err := h.refreshBlacklist(c.Request.Context(), middleware.GetUserID(c))
	if err != nil {
		code, message := blacklistError(err)
		return &out, out.FromBusinessError(contractBusinessError(code, message))
	}
	return &out, out.FromBlacklistRefreshSuccess(apicontract.BlacklistRefreshSuccess{Code: response.CodeOK, Msg: "success", Data: *summary})
}

package handler

import (
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

// List GET /system/blacklist
func (h *BlacklistHandler) List(c *gin.Context) {
	page, size, err := parsePaginationQuery(c, 20, 100)
	if err != nil {
		response.Fail(c, response.CodeParamError, service.ErrBlacklistInvalidQuery.Error())
		return
	}
	rows, total, err := h.svc.ListEntries(page, size, strings.TrimSpace(c.Query("kind")), c.Query("keyword"))
	if err != nil {
		blacklistFailure(c, err)
		return
	}
	response.OKWithPage(c, rows, total, page, size)
}

// LatestRefresh GET /system/blacklist/refresh; data is null before the first refresh.
func (h *BlacklistHandler) LatestRefresh(c *gin.Context) {
	summary, err := h.svc.LatestSummary()
	if err != nil {
		blacklistFailure(c, err)
		return
	}
	response.OK(c, summary)
}

// Refresh POST /system/blacklist/refresh
func (h *BlacklistHandler) Refresh(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), blacklistRefreshTimeout)
	defer cancel()
	summary, err := h.svc.Refresh(ctx, middleware.GetUserID(c))
	if err != nil {
		blacklistFailure(c, err)
		return
	}
	response.OK(c, summary)
}

var blacklistUserErrors = []error{
	service.ErrBlacklistSourceUnavailable,
	service.ErrBlacklistSourceLayoutChanged,
	service.ErrBlacklistEmptyResult,
	service.ErrBlacklistRefreshInProgress,
	service.ErrBlacklistRefreshCooldown,
}

// blacklistFailure answers with the sentinel's i18n key only; wrapped causes
// stay in the server log.
func blacklistFailure(c *gin.Context, err error) {
	if errors.Is(err, service.ErrBlacklistInvalidQuery) {
		response.Fail(c, response.CodeParamError, service.ErrBlacklistInvalidQuery.Error())
		return
	}
	for _, known := range blacklistUserErrors {
		if errors.Is(err, known) {
			response.Fail(c, response.CodeBizError, known.Error())
			return
		}
	}
	response.Fail(c, response.CodeBizError, service.ErrBlacklistInternal.Error())
}

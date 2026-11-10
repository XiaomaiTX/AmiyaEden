package handler

import (
	"amiya-eden/internal/middleware"
	"amiya-eden/internal/service"
	"amiya-eden/pkg/response"
	"github.com/gin-gonic/gin"
	"time"
)

type CaptainRefundHandler struct{ svc *service.CaptainRefundService }

func NewCaptainRefundHandler() *CaptainRefundHandler {
	return &CaptainRefundHandler{service.NewCaptainRefundService()}
}
func refundResponse(c *gin.Context, data interface{}, err error) {
	if err != nil {
		response.Fail(c, response.CodeBizError, err.Error())
		return
	}
	response.OK(c, data)
}
func (h *CaptainRefundHandler) Options(c *gin.Context) {
	v, e := h.svc.Options(middleware.GetUserID(c))
	refundResponse(c, v, e)
}
func (h *CaptainRefundHandler) Preview(c *gin.Context) {
	var req service.CaptainRefundRequest
	if e := c.ShouldBindJSON(&req); e != nil {
		response.Fail(c, response.CodeParamError, "invalid request")
		return
	}
	v, e := h.svc.Calculate(middleware.GetUserID(c), req, time.Now())
	refundResponse(c, v, e)
}
func (h *CaptainRefundHandler) Create(c *gin.Context) {
	var req service.CaptainRefundRequest
	if e := c.ShouldBindJSON(&req); e != nil {
		response.Fail(c, response.CodeParamError, "invalid request")
		return
	}
	v, e := h.svc.Create(middleware.GetUserID(c), req)
	refundResponse(c, v, e)
}
func (h *CaptainRefundHandler) list(c *gin.Context, admin bool) {
	page, size, e := parsePaginationQuery(c, 200, 200)
	if e != nil {
		response.Fail(c, response.CodeParamError, "invalid pagination")
		return
	}
	rows, total, e := h.svc.List(middleware.GetUserID(c), admin, page, size)
	if e != nil {
		refundResponse(c, nil, e)
		return
	}
	response.OKWithPage(c, rows, total, page, size)
}
func (h *CaptainRefundHandler) ListMine(c *gin.Context)  { h.list(c, false) }
func (h *CaptainRefundHandler) ListAdmin(c *gin.Context) { h.list(c, true) }
func (h *CaptainRefundHandler) detail(c *gin.Context, admin bool) {
	id, e := parseStrictUint(c.Param("id"))
	if e != nil {
		response.Fail(c, response.CodeParamError, "invalid id")
		return
	}
	v, e := h.svc.Detail(middleware.GetUserID(c), id, admin)
	refundResponse(c, v, e)
}
func (h *CaptainRefundHandler) DetailMine(c *gin.Context)  { h.detail(c, false) }
func (h *CaptainRefundHandler) DetailAdmin(c *gin.Context) { h.detail(c, true) }
func (h *CaptainRefundHandler) review(c *gin.Context, approve bool) {
	id, e := parseStrictUint(c.Param("id"))
	if e != nil {
		response.Fail(c, response.CodeParamError, "invalid id")
		return
	}
	var req struct {
		Reason string `json:"reason"`
	}
	if !approve {
		if e = c.ShouldBindJSON(&req); e != nil {
			response.Fail(c, response.CodeParamError, "invalid request")
			return
		}
	}
	e = h.svc.Review(middleware.GetUserID(c), id, approve, req.Reason)
	refundResponse(c, gin.H{}, e)
}
func (h *CaptainRefundHandler) Approve(c *gin.Context)     { h.review(c, true) }
func (h *CaptainRefundHandler) Reject(c *gin.Context)      { h.review(c, false) }
func (h *CaptainRefundHandler) GetSettings(c *gin.Context) { response.OK(c, h.svc.Settings()) }
func (h *CaptainRefundHandler) UpdateSettings(c *gin.Context) {
	var req service.CaptainRefundSettings
	if e := c.ShouldBindJSON(&req); e != nil {
		response.Fail(c, response.CodeParamError, "invalid settings")
		return
	}
	v, e := h.svc.UpdateSettings(middleware.GetUserID(c), req)
	refundResponse(c, v, e)
}

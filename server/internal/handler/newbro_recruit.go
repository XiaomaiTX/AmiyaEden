package handler

import (
	"amiya-eden/internal/middleware"
	"amiya-eden/internal/service"
	"amiya-eden/pkg/response"
	"time"

	"github.com/gin-gonic/gin"
)

type NewbroRecruitHandler struct {
	linkSvc  *service.RecruitmentLinkService
	entrySvc *service.RecruitmentEntryService
}

func NewNewbroRecruitHandler() *NewbroRecruitHandler {
	return &NewbroRecruitHandler{
		linkSvc:  service.NewRecruitmentLinkService(),
		entrySvc: service.NewRecruitmentEntryService(),
	}
}

// GenerateLink creates a new recruitment link for the current user.
// POST /api/v1/newbro/recruit/link
func (h *NewbroRecruitHandler) GenerateLink(c *gin.Context) {
	userID := middleware.GetUserID(c)
	rec, _, err := h.linkSvc.GenerateLink(userID, time.Now())
	if err != nil {
		response.Fail(c, response.CodeBizError, err.Error())
		return
	}
	response.OK(c, gin.H{
		"id":           rec.ID,
		"code":         rec.Code,
		"generated_at": rec.GeneratedAt,
	})
}

// GetMyLinks returns all recruitment links created by the current user.
// GET /api/v1/newbro/recruit/links
func (h *NewbroRecruitHandler) GetMyLinks(c *gin.Context) {
	userID := middleware.GetUserID(c)
	rows, err := h.linkSvc.GetMyLinks(userID)
	if err != nil {
		response.Fail(c, response.CodeBizError, err.Error())
		return
	}
	if rows == nil {
		rows = []service.RecruitLinkRow{}
	}
	response.OK(c, rows)
}

// GetDirectReferralStatus reports whether the current user can still submit a direct referrer.
// GET /api/v1/newbro/recruit/direct-referral
func (h *NewbroRecruitHandler) GetDirectReferralStatus(c *gin.Context) {
	status, err := h.entrySvc.GetDirectReferralStatus(middleware.GetUserID(c), time.Now())
	if err != nil {
		response.Fail(c, response.CodeBizError, err.Error())
		return
	}
	response.OK(c, status)
}

// CheckDirectReferrer looks up a direct referrer candidate by QQ.
// POST /api/v1/newbro/recruit/direct-referral/check
func (h *NewbroRecruitHandler) CheckDirectReferrer(c *gin.Context) {
	var req struct {
		QQ string `json:"qq" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, response.CodeParamError, "invalid request")
		return
	}

	candidate, err := h.entrySvc.LookupDirectReferrer(middleware.GetUserID(c), req.QQ, time.Now())
	if err != nil {
		response.Fail(c, response.CodeBizError, err.Error())
		return
	}
	response.OK(c, candidate)
}

// ConfirmDirectReferrer finalizes a direct referral and immediately rewards the referrer.
// POST /api/v1/newbro/recruit/direct-referral/confirm
func (h *NewbroRecruitHandler) ConfirmDirectReferrer(c *gin.Context) {
	var req struct {
		ReferrerUserID uint `json:"referrer_user_id" binding:"required,gt=0"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, response.CodeParamError, "invalid request")
		return
	}

	confirmed, err := h.entrySvc.ConfirmDirectReferral(middleware.GetUserID(c), req.ReferrerUserID, time.Now())
	if err != nil {
		response.Fail(c, response.CodeBizError, err.Error())
		return
	}
	response.OK(c, confirmed)
}

// GetAdminLinks returns paginated recruitment links for all users (admin only).
// GET /api/v1/system/recruit/links
func (h *NewbroRecruitHandler) GetAdminLinks(c *gin.Context) {
	page, pageSize, err := parsePaginationQuery(c, 20, 100)
	if err != nil {
		response.Fail(c, response.CodeParamError, err.Error())
		return
	}
	rows, total, err := h.linkSvc.ListAllLinks(page, pageSize)
	if err != nil {
		response.Fail(c, response.CodeBizError, err.Error())
		return
	}
	if rows == nil {
		rows = []service.AdminRecruitLinkRow{}
	}
	response.OKWithPage(c, rows, total, page, pageSize)
}

// SubmitQQ is the public (unauthenticated) handler for the recruitment landing page.
// POST /api/v1/recruit/:code/submit
func (h *NewbroRecruitHandler) SubmitQQ(c *gin.Context) {
	code := c.Param("code")

	var req struct {
		QQ string `json:"qq" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, response.CodeParamError, "invalid request")
		return
	}

	qqURL, err := h.entrySvc.SubmitQQ(code, req.QQ, time.Now())
	if err != nil {
		response.Fail(c, response.CodeBizError, err.Error())
		return
	}
	response.OK(c, gin.H{"qq_url": qqURL})
}

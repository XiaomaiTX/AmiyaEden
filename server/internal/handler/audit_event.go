package handler

import (
	"amiya-eden/internal/middleware"
	"amiya-eden/internal/repository"
	"amiya-eden/internal/service"
	"amiya-eden/pkg/response"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type AuditEventHandler struct {
	svc *service.AuditService
}

func NewAuditEventHandler() *AuditEventHandler {
	return &AuditEventHandler{svc: service.NewAuditService()}
}

type adminAuditEventListRequest struct {
	Current      int    `json:"current"`
	Size         int    `json:"size"`
	StartDate    string `json:"start_date"`
	EndDate      string `json:"end_date"`
	Category     string `json:"category"`
	Action       string `json:"action"`
	ActorUserID  *uint  `json:"actor_user_id"`
	TargetUserID *uint  `json:"target_user_id"`
	Result       string `json:"result"`
	RequestID    string `json:"request_id"`
	ResourceID   string `json:"resource_id"`
	Keyword      string `json:"keyword"`
}

type adminAuditExportRequest struct {
	Format string                          `json:"format"`
	Filter adminAuditEventListRequestInner `json:"filter"`
}

type adminAuditEventListRequestInner struct {
	StartDate    string `json:"start_date"`
	EndDate      string `json:"end_date"`
	Category     string `json:"category"`
	Action       string `json:"action"`
	ActorUserID  *uint  `json:"actor_user_id"`
	TargetUserID *uint  `json:"target_user_id"`
	Result       string `json:"result"`
	RequestID    string `json:"request_id"`
	ResourceID   string `json:"resource_id"`
	Keyword      string `json:"keyword"`
}

func parseAuditDate(raw string, field string) (*time.Time, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, nil
	}
	t, err := time.Parse("2006-01-02", raw)
	if err != nil {
		return nil, fmt.Errorf("%s format must be YYYY-MM-DD", field)
	}
	return &t, nil
}

func (h *AuditEventHandler) buildFilter(in adminAuditEventListRequestInner) (repository.AuditEventFilter, error) {
	start, err := parseAuditDate(in.StartDate, "start_date")
	if err != nil {
		return repository.AuditEventFilter{}, err
	}
	end, err := parseAuditDate(in.EndDate, "end_date")
	if err != nil {
		return repository.AuditEventFilter{}, err
	}
	if start != nil && end != nil && start.After(*end) {
		return repository.AuditEventFilter{}, fmt.Errorf("start_date cannot be later than end_date")
	}
	if end != nil {
		next := end.Add(24*time.Hour - time.Nanosecond)
		end = &next
	}
	return repository.AuditEventFilter{
		StartDate:    start,
		EndDate:      end,
		Category:     in.Category,
		Action:       in.Action,
		ActorUserID:  in.ActorUserID,
		TargetUserID: in.TargetUserID,
		Result:       in.Result,
		RequestID:    in.RequestID,
		ResourceID:   in.ResourceID,
		Keyword:      in.Keyword,
	}, nil
}

func (h *AuditEventHandler) AdminList(c *gin.Context) {
	var req adminAuditEventListRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		req.Current = 1
		req.Size = 20
	}
	req.Current, req.Size = normalizeLedgerPagination(req.Current, req.Size)

	filter, err := h.buildFilter(adminAuditEventListRequestInner{
		StartDate:    req.StartDate,
		EndDate:      req.EndDate,
		Category:     req.Category,
		Action:       req.Action,
		ActorUserID:  req.ActorUserID,
		TargetUserID: req.TargetUserID,
		Result:       req.Result,
		RequestID:    req.RequestID,
		ResourceID:   req.ResourceID,
		Keyword:      req.Keyword,
	})
	if err != nil {
		response.Fail(c, response.CodeParamError, err.Error())
		return
	}

	records, total, err := h.svc.AdminListAuditEvents(req.Current, req.Size, filter)
	if err != nil {
		response.Fail(c, response.CodeBizError, err.Error())
		return
	}
	response.OKWithPage(c, records, total, req.Current, req.Size)
}

func (h *AuditEventHandler) CreateExportTask(c *gin.Context) {
	var req adminAuditExportRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, response.CodeParamError, "参数错误")
		return
	}
	filter, err := h.buildFilter(req.Filter)
	if err != nil {
		response.Fail(c, response.CodeParamError, err.Error())
		return
	}
	operatorID := middleware.GetUserID(c)
	out, err := h.svc.CreateExportTask(c.Request.Context(), service.AuditExportTaskCreateInput{
		OperatorUserID: operatorID,
		Format:         req.Format,
		Filter:         filter,
		RequestID:      c.GetString("request-id"),
		IP:             c.ClientIP(),
		UserAgent:      c.Request.UserAgent(),
	})
	if err != nil {
		response.Fail(c, response.CodeBizError, err.Error())
		return
	}
	response.OK(c, out)
}

func (h *AuditEventHandler) GetExportTaskStatus(c *gin.Context) {
	taskID := strings.TrimSpace(c.Param("task_id"))
	if taskID == "" {
		response.Fail(c, response.CodeParamError, "task_id is required")
		return
	}
	out, err := h.svc.GetExportTaskStatus(taskID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			response.Fail(c, response.CodeParamError, "task not found")
			return
		}
		response.Fail(c, response.CodeBizError, err.Error())
		return
	}
	response.OK(c, out)
}

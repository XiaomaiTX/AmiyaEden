package handler

import (
	"amiya-eden/internal/apicontract"
	"amiya-eden/internal/middleware"
	"amiya-eden/internal/model"
	"amiya-eden/internal/service"
	"amiya-eden/pkg/response"
	"context"
	"errors"
	"fmt"
	"io"
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

func buildAuditFilter(in apicontract.AuditEventFilter) (service.AuditEventFilter, error) {
	start, err := parseAuditDate(in.StartDate, "start_date")
	if err != nil {
		return service.AuditEventFilter{}, err
	}
	end, err := parseAuditDate(in.EndDate, "end_date")
	if err != nil {
		return service.AuditEventFilter{}, err
	}
	if start != nil && end != nil && start.After(*end) {
		return service.AuditEventFilter{}, fmt.Errorf("start_date cannot be later than end_date")
	}
	if end != nil {
		next := end.Add(24*time.Hour - time.Nanosecond)
		end = &next
	}
	return service.AuditEventFilter{
		StartDate:    start,
		EndDate:      end,
		Category:     in.Category,
		Action:       in.Action,
		ActorUserID:  in.ActorUserId,
		TargetUserID: in.TargetUserId,
		Result:       string(in.Result),
		RequestID:    in.RequestId,
		ResourceID:   in.ResourceId,
		Keyword:      in.Keyword,
	}, nil
}

func (h *AuditEventHandler) listAuditEvents(req apicontract.AuditEventSearchInput, decodeErr error) (apicontract.AuditEventPage, *apicontract.BusinessError) {
	// A failed bind resets pagination but keeps any filters the decoder populated.
	if decodeErr != nil {
		req.Current, req.Size = 1, 20
	}
	req.Current, req.Size = normalizeLedgerPagination(req.Current, req.Size)

	filter, err := buildAuditFilter(apicontract.AuditEventFilter{
		StartDate:    req.StartDate,
		EndDate:      req.EndDate,
		Category:     req.Category,
		Action:       req.Action,
		ActorUserId:  req.ActorUserId,
		TargetUserId: req.TargetUserId,
		Result:       req.Result,
		RequestId:    req.RequestId,
		ResourceId:   req.ResourceId,
		Keyword:      req.Keyword,
	})
	if err != nil {
		failure := contractBusinessError(response.CodeParamError, err.Error())
		return apicontract.AuditEventPage{}, &failure
	}

	records, total, err := h.svc.AdminListAuditEvents(req.Current, req.Size, filter)
	if err != nil {
		failure := contractBusinessError(response.CodeBizError, err.Error())
		return apicontract.AuditEventPage{}, &failure
	}
	rows := make([]apicontract.AuditEvent, 0, len(records))
	for _, record := range records {
		rows = append(rows, auditEvent(record))
	}
	return apicontract.AuditEventPage{
		List: rows, Total: total, Page: req.Current, PageSize: req.Size,
	}, nil
}

func auditExportRequestMetadata(c *gin.Context) service.AuditExportTaskCreateInput {
	return service.AuditExportTaskCreateInput{
		OperatorUserID: middleware.GetUserID(c),
		RequestID:      c.GetString("request-id"),
		IP:             c.ClientIP(),
		UserAgent:      c.Request.UserAgent(),
	}
}

func (h *AuditEventHandler) createAuditExportTask(ctx context.Context, req apicontract.AuditExportCreateInput, input service.AuditExportTaskCreateInput) (apicontract.AuditExportTaskStatus, *apicontract.BusinessError) {
	var filterInput apicontract.AuditEventFilter
	if req.Filter != nil {
		filterInput = *req.Filter
	}
	filter, err := buildAuditFilter(filterInput)
	if err != nil {
		failure := contractBusinessError(response.CodeParamError, err.Error())
		return apicontract.AuditExportTaskStatus{}, &failure
	}
	input.Format, input.Filter = string(req.Format), filter
	out, err := h.svc.CreateExportTask(ctx, input)
	if err != nil {
		failure := contractBusinessError(response.CodeBizError, err.Error())
		return apicontract.AuditExportTaskStatus{}, &failure
	}
	return auditExportStatus(*out), nil
}

func (h *AuditEventHandler) getAuditExportTaskStatus(taskID string) (apicontract.AuditExportTaskStatus, *apicontract.BusinessError) {
	taskID = strings.TrimSpace(taskID)
	if taskID == "" {
		failure := contractBusinessError(response.CodeParamError, "task_id is required")
		return apicontract.AuditExportTaskStatus{}, &failure
	}
	out, err := h.svc.GetExportTaskStatus(taskID)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		failure := contractBusinessError(response.CodeParamError, "task not found")
		return apicontract.AuditExportTaskStatus{}, &failure
	}
	if err != nil {
		failure := contractBusinessError(response.CodeBizError, err.Error())
		return apicontract.AuditExportTaskStatus{}, &failure
	}
	return auditExportStatus(*out), nil
}

func (h *AuditEventHandler) listAuditExportTasks(operatorID uint, req apicontract.AuditExportListInput, decodeErr error) ([]apicontract.AuditExportTaskStatus, error) {
	if decodeErr != nil || req.Limit <= 0 {
		req.Limit = 20
	}
	if req.Limit > 200 {
		req.Limit = 200
	}
	out, err := h.svc.ListExportTaskStatuses(operatorID, req.Limit)
	if err != nil {
		return nil, err
	}
	tasks := make([]apicontract.AuditExportTaskStatus, 0, len(out))
	for _, task := range out {
		tasks = append(tasks, auditExportStatus(task))
	}
	return tasks, nil
}

func auditEvent(record model.AuditEvent) apicontract.AuditEvent {
	return apicontract.AuditEvent{
		Id: record.ID, EventId: record.EventID, OccurredAt: record.OccurredAt,
		Category: record.Category, Action: record.Action, ActorUserId: record.ActorUserID,
		TargetUserId: record.TargetUserID, ResourceType: record.ResourceType,
		ResourceId: record.ResourceID, Result: apicontract.AuditResult(record.Result),
		RequestId: record.RequestID, Ip: record.IP, UserAgent: record.UserAgent,
		DetailsJson: record.DetailsJSON, CreatedAt: record.CreatedAt,
	}
}

func auditExportStatus(task service.AuditExportTaskStatus) apicontract.AuditExportTaskStatus {
	return apicontract.AuditExportTaskStatus{
		TaskId: task.TaskID, Status: apicontract.AuditExportStatus(task.Status),
		Format: apicontract.AuditExportFormat(task.Format), DownloadUrl: task.DownloadURL,
		ErrorMessage: task.ErrorMessage, ExpireAt: task.ExpireAt, CreatedAt: task.CreatedAt,
		FinishedAt: task.FinishedAt,
	}
}

func (h *AuditEventHandler) ListAuditEvents(ctx context.Context, req apicontract.ListAuditEventsRequestObject) (apicontract.ListAuditEventsResponseObject, error) {
	var input apicontract.AuditEventSearchInput
	decodeErr := apicontract.StrictGinJSONDecodeError(ctx)
	if req.Body != nil {
		input = *req.Body
	} else {
		// The legacy decoder treated an absent body as EOF, unlike a JSON {}.
		decodeErr = io.EOF
	}
	var out apicontract.ListAuditEvents200JSONResponse
	data, failure := h.listAuditEvents(input, decodeErr)
	if failure != nil {
		return &out, out.FromBusinessError(*failure)
	}
	return &out, out.FromAuditEventPageSuccess(apicontract.AuditEventPageSuccess{Code: response.CodeOK, Msg: "success", Data: data})
}

func (h *AuditEventHandler) CreateAuditExportTask(ctx context.Context, req apicontract.CreateAuditExportTaskRequestObject) (apicontract.CreateAuditExportTaskResponseObject, error) {
	var out apicontract.CreateAuditExportTask200JSONResponse
	if req.Body == nil {
		return &out, out.FromBusinessError(contractBusinessError(response.CodeParamError, "参数错误"))
	}
	c, _ := ctx.(*gin.Context)
	if c == nil || c.Request == nil {
		return apicontract.CreateAuditExportTask401JSONResponse{UnauthorizedJSONResponse: apicontract.UnauthorizedJSONResponse{Code: response.CodeUnauthorized, Msg: "未登录"}}, nil
	}
	data, failure := h.createAuditExportTask(c.Request.Context(), *req.Body, auditExportRequestMetadata(c))
	if failure != nil {
		return &out, out.FromBusinessError(*failure)
	}
	return &out, out.FromAuditExportTaskSuccess(apicontract.AuditExportTaskSuccess{Code: response.CodeOK, Msg: "success", Data: data})
}

func (h *AuditEventHandler) GetAuditExportTaskStatus(_ context.Context, req apicontract.GetAuditExportTaskStatusRequestObject) (apicontract.GetAuditExportTaskStatusResponseObject, error) {
	var out apicontract.GetAuditExportTaskStatus200JSONResponse
	data, failure := h.getAuditExportTaskStatus(req.TaskId)
	if failure != nil {
		return &out, out.FromBusinessError(*failure)
	}
	return &out, out.FromAuditExportTaskSuccess(apicontract.AuditExportTaskSuccess{Code: response.CodeOK, Msg: "success", Data: data})
}

func (h *AuditEventHandler) ListAuditExportTasks(ctx context.Context, req apicontract.ListAuditExportTasksRequestObject) (apicontract.ListAuditExportTasksResponseObject, error) {
	c, _ := ctx.(*gin.Context)
	if c == nil {
		return apicontract.ListAuditExportTasks401JSONResponse{UnauthorizedJSONResponse: apicontract.UnauthorizedJSONResponse{Code: response.CodeUnauthorized, Msg: "未登录"}}, nil
	}
	var input apicontract.AuditExportListInput
	if req.Body != nil {
		input = *req.Body
	}
	var out apicontract.ListAuditExportTasks200JSONResponse
	data, err := h.listAuditExportTasks(middleware.GetUserID(c), input, apicontract.StrictGinJSONDecodeError(ctx))
	if err != nil {
		return &out, out.FromBusinessError(contractBusinessError(response.CodeBizError, err.Error()))
	}
	return &out, out.FromAuditExportListSuccess(apicontract.AuditExportListSuccess{Code: response.CodeOK, Msg: "success", Data: data})
}

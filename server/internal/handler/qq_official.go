package handler

import (
	"amiya-eden/internal/apicontract"
	"amiya-eden/internal/middleware"
	"amiya-eden/internal/model"
	"amiya-eden/internal/service"
	"amiya-eden/pkg/response"
	"context"
	"errors"
	"slices"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type QQOfficialHandler struct{}

func NewQQOfficialHandler() *QQOfficialHandler { return &QQOfficialHandler{} }

// Resolve credentials on each request. A genuinely unconfigured service keeps
// the existing empty read results; a failure to read usable credentials is a
// generic outage, never an empty result or a detailed credential error.
func officialActive(ctx context.Context) (*service.QQOfficialService, error) {
	svc, err := service.DefaultQQOfficialRuntime().Active(officialRequestContext(ctx), nil)
	if err != nil {
		return nil, service.ErrQQOfficialUnavailable
	}
	return svc, nil
}

// The generated strict interface provides a pooled Gin context. Database code
// must use the underlying request context, which is safe beyond the handler.
func officialRequestContext(ctx context.Context) context.Context {
	if c, ok := ctx.(*gin.Context); ok && c != nil && c.Request != nil {
		return c.Request.Context()
	}
	return ctx
}

func officialActor(ctx context.Context) uint {
	if c, ok := ctx.(*gin.Context); ok {
		return middleware.GetUserID(c)
	}
	return 0
}

func officialValue[T any](p *T) (v T) {
	if p != nil {
		return *p
	}
	return v
}
func officialOptional[T comparable](v T) *T {
	var zero T
	if v == zero {
		return nil
	}
	return &v
}
func officialID(s string) bool {
	id, err := uuid.Parse(s)
	return err == nil && id != uuid.Nil && id.String() == s
}

func officialPagination(page, size *int, defaultSize int) (int, int, bool) {
	p, n := 1, defaultSize
	if page != nil {
		p = *page
	}
	if size != nil {
		n = *size
	}
	return p, n, p >= 1 && p <= 10000 && n >= 1 && n <= 200
}

func officialError(status int, err error) apicontract.QQOfficialError {
	return apicontract.QQOfficialError{Code: apicontract.QQOfficialErrorCode(status), Msg: service.QQOfficialPublicError(err).Error()}
}

func officialErrorCode(err error) int {
	switch {
	case err == nil:
		return 0
	case errors.Is(err, service.ErrQQOfficialInvalid):
		return 400
	case errors.Is(err, service.ErrQQOfficialNotFound):
		return 404
	case errors.Is(err, service.ErrQQOfficialConflict), errors.Is(err, service.ErrQQOfficialNotAuthorized), errors.Is(err, service.ErrQQOfficialNoTargets), errors.Is(err, service.ErrQQOfficialExpired):
		return 409
	case errors.Is(err, service.ErrQQOfficialCapacity):
		return 429
	default:
		return 503
	}
}

func qqStatusDTO(view service.QQOfficialStatus) apicontract.QQOfficialStatusResultJSONResponse {
	entries := make([]apicontract.QQOfficialCooldown, 0, len(view.Cooldowns))
	for _, row := range view.Cooldowns {
		if row.Until != nil {
			entries = append(entries, apicontract.QQOfficialCooldown{Scope: row.Scope, Until: *row.Until})
		}
	}
	return apicontract.QQOfficialStatusResultJSONResponse{Code: response.CodeOK, Msg: "success", Data: apicontract.QQOfficialStatus{
		Configured: view.Configured, AppID: officialOptional(view.AppID), HasAppSecret: view.HasAppSecret,
		Enabled: view.Enabled, Paused: view.Paused, Certified: view.Certified, ApprovalWritesEnabled: view.ApprovalWritesEnabled,
		TokenState: view.TokenState, GatewayState: apicontract.QQOfficialStatusGatewayState(view.GatewayState), LastEventAt: view.LastEventAt,
		NextRequestAt: view.NextRequestAt, OldestQueuedAt: view.OldestQueuedAt,
		QueueDepth: int(view.QueueDepth), AttemptsLastMinute: int(view.AttemptsLastMinute), MessagesLastMinute: int(view.MessagesLastMinute), MessageBudget: view.MessageBudget, Cooldowns: entries}}
}

func qqGroupDTO(row model.QQOfficialGroup) apicontract.QQOfficialGroup {
	return apicontract.QQOfficialGroup{Id: row.ID, Alias: row.Alias, Present: row.Present, Authorized: row.Authorized, TestVerified: row.TestVerified,
		NotificationEnabled: row.NotificationEnabled, ApprovalEnabled: row.ApprovalEnabled, ApprovalCapability: apicontract.QQOfficialApprovalCapability(row.ApprovalCapability),
		PushState: apicontract.QQOfficialPushState(row.PushState), FleetCorporationID: officialOptional(row.FleetCorporation), StructureCorporationID: officialOptional(row.StructureCorporation)}
}

func qqTaskDTO(row model.QQOfficialTask) apicontract.QQOfficialTask {
	return apicontract.QQOfficialTask{Id: row.ID, GroupID: officialOptional(row.GroupID), Kind: row.Kind, Source: row.Source, Status: row.Status, Reason: officialOptional(row.Reason),
		TraceID: officialOptional(row.TraceID), UpstreamCode: officialOptional(row.UpstreamCode), CreatedAt: row.CreatedAt, ExpiresAt: row.ExpiresAt, NextRunAt: row.NextRunAt, FinishedAt: row.FinishedAt}
}

func qqTaskResult(row model.QQOfficialTask, attempts []model.QQOfficialAttempt) apicontract.QQOfficialTaskResultJSONResponse {
	entries := make([]apicontract.QQOfficialTaskAttempt, 0, len(attempts))
	for _, item := range attempts {
		entries = append(entries, apicontract.QQOfficialTaskAttempt{At: item.AdmittedAt, FinishedAt: item.FinishedAt, Outcome: officialOptional(item.Outcome),
			TraceID: officialOptional(item.TraceID), PlatformCode: officialOptional(item.PlatformCode), HTTPStatus: officialOptional(item.HTTPStatus)})
	}
	t := qqTaskDTO(row)
	return apicontract.QQOfficialTaskResultJSONResponse{Code: response.CodeOK, Msg: "success", Data: apicontract.QQOfficialTaskDetail{
		Id: t.Id, GroupID: t.GroupID, Kind: t.Kind, Source: t.Source, Status: t.Status, Reason: t.Reason, TraceID: t.TraceID, UpstreamCode: t.UpstreamCode,
		CreatedAt: t.CreatedAt, ExpiresAt: t.ExpiresAt, NextRunAt: t.NextRunAt, FinishedAt: t.FinishedAt, Attempts: entries}}
}

func qqAccepted(row model.QQOfficialTask) apicontract.QQOfficialAcceptedJSONResponse {
	return apicontract.QQOfficialAcceptedJSONResponse{Code: response.CodeOK, Msg: "success", Data: apicontract.QQOfficialTaskReceipt{
		TaskID: row.ID, Status: row.Status, ExpiresAt: row.ExpiresAt}}
}

func (h *QQOfficialHandler) GetQQOfficialStatus(ctx context.Context, _ apicontract.GetQQOfficialStatusRequestObject) (apicontract.GetQQOfficialStatusResponseObject, error) {
	if c, ok := ctx.(*gin.Context); ok {
		c.Header("Cache-Control", "no-store")
	}
	view, err := service.DefaultQQOfficialRuntime().Status(officialRequestContext(ctx))
	if err != nil {
		return apicontract.GetQQOfficialStatus503JSONResponse{QQOfficialUnavailableJSONResponse: apicontract.QQOfficialUnavailableJSONResponse(officialError(503, service.ErrQQOfficialUnavailable))}, nil
	}
	return apicontract.GetQQOfficialStatus200JSONResponse{QQOfficialStatusResultJSONResponse: qqStatusDTO(view)}, nil
}

func (h *QQOfficialHandler) UpdateQQOfficialSettings(ctx context.Context, req apicontract.UpdateQQOfficialSettingsRequestObject) (apicontract.UpdateQQOfficialSettingsResponseObject, error) {
	c, _ := ctx.(*gin.Context)
	if c != nil {
		c.Header("Cache-Control", "no-store")
	}
	if req.Body == nil || (c != nil && c.Request.URL.RawQuery != "") {
		return apicontract.UpdateQQOfficialSettings400JSONResponse{QQOfficialBadRequestJSONResponse: apicontract.QQOfficialBadRequestJSONResponse(officialError(400, service.ErrQQOfficialInvalid))}, nil
	}
	body := req.Body
	view, err := service.DefaultQQOfficialRuntime().UpdateSettings(officialRequestContext(ctx), officialActor(ctx), service.QQOfficialSettingsUpdate{
		AppID: body.AppID, AppSecret: body.AppSecret, ApprovalWritesEnabled: body.ApprovalWritesEnabled,
		Enabled: body.Enabled, Paused: body.Paused, Certified: body.Certified,
	})
	switch {
	case err == nil:
		return apicontract.UpdateQQOfficialSettings200JSONResponse{QQOfficialStatusResultJSONResponse: qqStatusDTO(view)}, nil
	case errors.Is(err, service.ErrQQOfficialInvalid):
		return apicontract.UpdateQQOfficialSettings400JSONResponse{QQOfficialBadRequestJSONResponse: apicontract.QQOfficialBadRequestJSONResponse(officialError(400, service.ErrQQOfficialInvalid))}, nil
	case errors.Is(err, service.ErrQQOfficialConflict):
		return apicontract.UpdateQQOfficialSettings409JSONResponse{QQOfficialConflictJSONResponse: apicontract.QQOfficialConflictJSONResponse(officialError(409, service.ErrQQOfficialConflict))}, nil
	default:
		return apicontract.UpdateQQOfficialSettings503JSONResponse{QQOfficialUnavailableJSONResponse: apicontract.QQOfficialUnavailableJSONResponse(officialError(503, service.ErrQQOfficialUnavailable))}, nil
	}
}

func (h *QQOfficialHandler) ListQQOfficialGroups(ctx context.Context, req apicontract.ListQQOfficialGroupsRequestObject) (apicontract.ListQQOfficialGroupsResponseObject, error) {
	page, size, valid := officialPagination(req.Params.Current, req.Params.Size, 20)
	if !valid {
		return apicontract.ListQQOfficialGroups400JSONResponse{QQOfficialBadRequestJSONResponse: apicontract.QQOfficialBadRequestJSONResponse(officialError(400, service.ErrQQOfficialInvalid))}, nil
	}
	svc, err := officialActive(ctx)
	if err != nil {
		return apicontract.ListQQOfficialGroups503JSONResponse{QQOfficialUnavailableJSONResponse: apicontract.QQOfficialUnavailableJSONResponse(officialError(503, err))}, nil
	}
	rows, total, err := svc.Groups(officialRequestContext(ctx), (page-1)*size, size)
	if err != nil {
		return apicontract.ListQQOfficialGroups503JSONResponse{QQOfficialUnavailableJSONResponse: apicontract.QQOfficialUnavailableJSONResponse(officialError(503, err))}, nil
	}
	list := make([]apicontract.QQOfficialGroup, 0, len(rows))
	for _, row := range rows {
		list = append(list, qqGroupDTO(row))
	}
	return apicontract.ListQQOfficialGroups200JSONResponse{Code: response.CodeOK, Msg: "success", Data: apicontract.QQOfficialGroupPage{List: list, Page: page, PageSize: size, Total: total}}, nil
}

func (h *QQOfficialHandler) UpdateQQOfficialGroup(ctx context.Context, req apicontract.UpdateQQOfficialGroupRequestObject) (apicontract.UpdateQQOfficialGroupResponseObject, error) {
	var row model.QQOfficialGroup
	err := service.ErrQQOfficialInvalid
	if req.Body != nil && officialID(req.Id) {
		svc, activeErr := officialActive(ctx)
		err = activeErr
		if err == nil {
			b := req.Body
			row, err = svc.UpdateGroup(officialRequestContext(ctx), officialActor(ctx), req.Id, service.QQOfficialGroupUpdate{Alias: b.Alias, Authorized: b.Authorized,
				NotificationEnabled: b.NotificationEnabled, ApprovalEnabled: b.ApprovalEnabled, FleetCorporationID: b.FleetCorporationID, StructureCorporationID: b.StructureCorporationID})
		}
	}
	if err == nil {
		return apicontract.UpdateQQOfficialGroup200JSONResponse{Code: response.CodeOK, Msg: "success", Data: qqGroupDTO(row)}, nil
	}
	code := officialErrorCode(err)
	failure := officialError(code, err)
	switch code {
	case 400:
		return apicontract.UpdateQQOfficialGroup400JSONResponse{QQOfficialBadRequestJSONResponse: apicontract.QQOfficialBadRequestJSONResponse(failure)}, nil
	case 404:
		return apicontract.UpdateQQOfficialGroup404JSONResponse{QQOfficialNotFoundJSONResponse: apicontract.QQOfficialNotFoundJSONResponse(failure)}, nil
	case 409:
		return apicontract.UpdateQQOfficialGroup409JSONResponse{QQOfficialConflictJSONResponse: apicontract.QQOfficialConflictJSONResponse(failure)}, nil
	default:
		return apicontract.UpdateQQOfficialGroup503JSONResponse{QQOfficialUnavailableJSONResponse: apicontract.QQOfficialUnavailableJSONResponse(failure)}, nil
	}
}

func (h *QQOfficialHandler) VerifyQQOfficialGroup(ctx context.Context, req apicontract.VerifyQQOfficialGroupRequestObject) (apicontract.VerifyQQOfficialGroupResponseObject, error) {
	var row model.QQOfficialTask
	err := service.ErrQQOfficialInvalid
	if req.Body != nil {
		svc, activeErr := officialActive(ctx)
		err = activeErr
		if err == nil {
			row, err = svc.VerifyGroup(officialRequestContext(ctx), officialActor(ctx), req.Body.GroupOpenID, req.Body.IdempotencyKey)
		}
	}
	if err == nil {
		return apicontract.VerifyQQOfficialGroup202JSONResponse{QQOfficialAcceptedJSONResponse: qqAccepted(row)}, nil
	}
	code := officialErrorCode(err)
	failure := officialError(code, err)
	switch code {
	case 400:
		return apicontract.VerifyQQOfficialGroup400JSONResponse{QQOfficialBadRequestJSONResponse: apicontract.QQOfficialBadRequestJSONResponse(failure)}, nil
	case 409:
		return apicontract.VerifyQQOfficialGroup409JSONResponse{QQOfficialConflictJSONResponse: apicontract.QQOfficialConflictJSONResponse(failure)}, nil
	case 429:
		return apicontract.VerifyQQOfficialGroup429JSONResponse{QQOfficialRateLimitedJSONResponse: apicontract.QQOfficialRateLimitedJSONResponse(failure)}, nil
	default:
		return apicontract.VerifyQQOfficialGroup503JSONResponse{QQOfficialUnavailableJSONResponse: apicontract.QQOfficialUnavailableJSONResponse(failure)}, nil
	}
}

func (h *QQOfficialHandler) TestQQOfficialGroup(ctx context.Context, req apicontract.TestQQOfficialGroupRequestObject) (apicontract.TestQQOfficialGroupResponseObject, error) {
	var row model.QQOfficialTask
	err := service.ErrQQOfficialInvalid
	if req.Body != nil && officialID(req.Id) {
		svc, activeErr := officialActive(ctx)
		err = activeErr
		if err == nil {
			row, err = svc.TestGroup(officialRequestContext(ctx), officialActor(ctx), req.Id, req.Body.IdempotencyKey)
		}
	}
	if err == nil {
		return apicontract.TestQQOfficialGroup202JSONResponse{QQOfficialAcceptedJSONResponse: qqAccepted(row)}, nil
	}
	code := officialErrorCode(err)
	failure := officialError(code, err)
	switch code {
	case 400:
		return apicontract.TestQQOfficialGroup400JSONResponse{QQOfficialBadRequestJSONResponse: apicontract.QQOfficialBadRequestJSONResponse(failure)}, nil
	case 404:
		return apicontract.TestQQOfficialGroup404JSONResponse{QQOfficialNotFoundJSONResponse: apicontract.QQOfficialNotFoundJSONResponse(failure)}, nil
	case 409:
		return apicontract.TestQQOfficialGroup409JSONResponse{QQOfficialConflictJSONResponse: apicontract.QQOfficialConflictJSONResponse(failure)}, nil
	case 429:
		return apicontract.TestQQOfficialGroup429JSONResponse{QQOfficialRateLimitedJSONResponse: apicontract.QQOfficialRateLimitedJSONResponse(failure)}, nil
	default:
		return apicontract.TestQQOfficialGroup503JSONResponse{QQOfficialUnavailableJSONResponse: apicontract.QQOfficialUnavailableJSONResponse(failure)}, nil
	}
}

func (h *QQOfficialHandler) ListQQOfficialTasks(ctx context.Context, req apicontract.ListQQOfficialTasksRequestObject) (apicontract.ListQQOfficialTasksResponseObject, error) {
	page, size, valid := officialPagination(req.Params.Current, req.Params.Size, 200)
	group := officialValue(req.Params.GroupId)
	filter := service.QQOfficialTaskFilter{State: officialValue(req.Params.Status), Kind: officialValue(req.Params.Kind), Source: officialValue(req.Params.Source), GroupID: group}
	valid = valid && (group == "" || officialID(group)) && len(filter.State) <= 24 && len(filter.Kind) <= 24 && len(filter.Source) <= 24
	if !valid {
		return apicontract.ListQQOfficialTasks400JSONResponse{QQOfficialBadRequestJSONResponse: apicontract.QQOfficialBadRequestJSONResponse(officialError(400, service.ErrQQOfficialInvalid))}, nil
	}
	svc, err := officialActive(ctx)
	if err != nil {
		return apicontract.ListQQOfficialTasks503JSONResponse{QQOfficialUnavailableJSONResponse: apicontract.QQOfficialUnavailableJSONResponse(officialError(503, err))}, nil
	}
	rows, total, err := svc.Tasks(officialRequestContext(ctx), filter, (page-1)*size, size)
	if err != nil {
		return apicontract.ListQQOfficialTasks503JSONResponse{QQOfficialUnavailableJSONResponse: apicontract.QQOfficialUnavailableJSONResponse(officialError(503, err))}, nil
	}
	list := make([]apicontract.QQOfficialTask, 0, len(rows))
	for _, row := range rows {
		list = append(list, qqTaskDTO(row))
	}
	return apicontract.ListQQOfficialTasks200JSONResponse{Code: response.CodeOK, Msg: "success", Data: apicontract.QQOfficialTaskPage{List: list, Page: page, PageSize: size, Total: total}}, nil
}

func (h *QQOfficialHandler) GetQQOfficialTask(ctx context.Context, req apicontract.GetQQOfficialTaskRequestObject) (apicontract.GetQQOfficialTaskResponseObject, error) {
	if !officialID(req.Id) {
		return apicontract.GetQQOfficialTask400JSONResponse{QQOfficialBadRequestJSONResponse: apicontract.QQOfficialBadRequestJSONResponse(officialError(400, service.ErrQQOfficialInvalid))}, nil
	}
	svc, err := officialActive(ctx)
	if err != nil {
		return apicontract.GetQQOfficialTask503JSONResponse{QQOfficialUnavailableJSONResponse: apicontract.QQOfficialUnavailableJSONResponse(officialError(503, err))}, nil
	}
	row, attempts, err := svc.Task(officialRequestContext(ctx), req.Id)
	if err == nil {
		return apicontract.GetQQOfficialTask200JSONResponse{QQOfficialTaskResultJSONResponse: qqTaskResult(row, attempts)}, nil
	}
	if errors.Is(err, service.ErrQQOfficialNotFound) {
		return apicontract.GetQQOfficialTask404JSONResponse{QQOfficialNotFoundJSONResponse: apicontract.QQOfficialNotFoundJSONResponse(officialError(404, err))}, nil
	}
	return apicontract.GetQQOfficialTask503JSONResponse{QQOfficialUnavailableJSONResponse: apicontract.QQOfficialUnavailableJSONResponse(officialError(503, err))}, nil
}

func (h *QQOfficialHandler) CancelQQOfficialTask(ctx context.Context, req apicontract.CancelQQOfficialTaskRequestObject) (apicontract.CancelQQOfficialTaskResponseObject, error) {
	if !officialID(req.Id) {
		return apicontract.CancelQQOfficialTask400JSONResponse{QQOfficialBadRequestJSONResponse: apicontract.QQOfficialBadRequestJSONResponse(officialError(400, service.ErrQQOfficialInvalid))}, nil
	}
	svc, err := officialActive(ctx)
	if err != nil {
		return apicontract.CancelQQOfficialTask503JSONResponse{QQOfficialUnavailableJSONResponse: apicontract.QQOfficialUnavailableJSONResponse(officialError(503, err))}, nil
	}
	row, attempts, err := svc.Cancel(officialRequestContext(ctx), officialActor(ctx), req.Id)
	if err == nil {
		return apicontract.CancelQQOfficialTask200JSONResponse{QQOfficialTaskResultJSONResponse: qqTaskResult(row, attempts)}, nil
	}
	code := officialErrorCode(err)
	failure := officialError(code, err)
	switch code {
	case 404:
		return apicontract.CancelQQOfficialTask404JSONResponse{QQOfficialNotFoundJSONResponse: apicontract.QQOfficialNotFoundJSONResponse(failure)}, nil
	case 409:
		return apicontract.CancelQQOfficialTask409JSONResponse{QQOfficialConflictJSONResponse: apicontract.QQOfficialConflictJSONResponse(failure)}, nil
	default:
		return apicontract.CancelQQOfficialTask503JSONResponse{QQOfficialUnavailableJSONResponse: apicontract.QQOfficialUnavailableJSONResponse(failure)}, nil
	}
}

func (h *QQOfficialHandler) ListQQOfficialJoinRequests(ctx context.Context, req apicontract.ListQQOfficialJoinRequestsRequestObject) (apicontract.ListQQOfficialJoinRequestsResponseObject, error) {
	page, size, valid := officialPagination(req.Params.Current, req.Params.Size, 200)
	state := string(officialValue(req.Params.Status))
	valid = valid && officialID(req.Id) && slices.Contains([]string{"", "pending", "queued", "approved", "declined", "closed_elsewhere", "unknown", "expired", "failed"}, state)
	if !valid {
		return apicontract.ListQQOfficialJoinRequests400JSONResponse{QQOfficialBadRequestJSONResponse: apicontract.QQOfficialBadRequestJSONResponse(officialError(400, service.ErrQQOfficialInvalid))}, nil
	}
	svc, err := officialActive(ctx)
	if err != nil {
		return apicontract.ListQQOfficialJoinRequests503JSONResponse{QQOfficialUnavailableJSONResponse: apicontract.QQOfficialUnavailableJSONResponse(officialError(503, err))}, nil
	}
	rows, total, err := svc.JoinRequests(officialRequestContext(ctx), req.Id, state, (page-1)*size, size)
	if errors.Is(err, service.ErrQQOfficialNotFound) || errors.Is(err, service.ErrQQOfficialNotAuthorized) {
		return apicontract.ListQQOfficialJoinRequests404JSONResponse{QQOfficialNotFoundJSONResponse: apicontract.QQOfficialNotFoundJSONResponse(officialError(404, err))}, nil
	}
	if err != nil {
		return apicontract.ListQQOfficialJoinRequests503JSONResponse{QQOfficialUnavailableJSONResponse: apicontract.QQOfficialUnavailableJSONResponse(officialError(503, err))}, nil
	}
	list := make([]apicontract.QQOfficialJoinRequest, 0, len(rows))
	for _, row := range rows {
		list = append(list, apicontract.QQOfficialJoinRequest{Id: row.ID, GroupID: row.GroupID, TaskID: officialOptional(row.TaskID), Username: row.Username,
			AppliedAt: row.AppliedAt, ExpiresAt: row.ExpiresAt, Status: apicontract.QQOfficialJoinRequestStatus(row.Status)})
	}
	return apicontract.ListQQOfficialJoinRequests200JSONResponse{Code: response.CodeOK, Msg: "success", Data: apicontract.QQOfficialJoinRequestPage{List: list, Page: page, PageSize: size, Total: total}}, nil
}

func (h *QQOfficialHandler) RefreshQQOfficialJoinRequests(ctx context.Context, req apicontract.RefreshQQOfficialJoinRequestsRequestObject) (apicontract.RefreshQQOfficialJoinRequestsResponseObject, error) {
	var row model.QQOfficialTask
	err := service.ErrQQOfficialInvalid
	if req.Body != nil && officialID(req.Id) {
		svc, activeErr := officialActive(ctx)
		err = activeErr
		if err == nil {
			row, err = svc.RefreshJoinRequests(officialRequestContext(ctx), officialActor(ctx), req.Id, req.Body.IdempotencyKey)
		}
	}
	if err == nil {
		return apicontract.RefreshQQOfficialJoinRequests202JSONResponse{QQOfficialAcceptedJSONResponse: qqAccepted(row)}, nil
	}
	code := officialErrorCode(err)
	failure := officialError(code, err)
	switch code {
	case 400:
		return apicontract.RefreshQQOfficialJoinRequests400JSONResponse{QQOfficialBadRequestJSONResponse: apicontract.QQOfficialBadRequestJSONResponse(failure)}, nil
	case 404:
		return apicontract.RefreshQQOfficialJoinRequests404JSONResponse{QQOfficialNotFoundJSONResponse: apicontract.QQOfficialNotFoundJSONResponse(failure)}, nil
	case 409:
		return apicontract.RefreshQQOfficialJoinRequests409JSONResponse{QQOfficialConflictJSONResponse: apicontract.QQOfficialConflictJSONResponse(failure)}, nil
	case 429:
		return apicontract.RefreshQQOfficialJoinRequests429JSONResponse{QQOfficialRateLimitedJSONResponse: apicontract.QQOfficialRateLimitedJSONResponse(failure)}, nil
	default:
		return apicontract.RefreshQQOfficialJoinRequests503JSONResponse{QQOfficialUnavailableJSONResponse: apicontract.QQOfficialUnavailableJSONResponse(failure)}, nil
	}
}

func (h *QQOfficialHandler) DecideQQOfficialJoinRequest(ctx context.Context, req apicontract.DecideQQOfficialJoinRequestRequestObject) (apicontract.DecideQQOfficialJoinRequestResponseObject, error) {
	var row model.QQOfficialTask
	err := service.ErrQQOfficialInvalid
	if req.Body != nil && officialID(req.Id) {
		svc, activeErr := officialActive(ctx)
		err = activeErr
		if err == nil {
			row, err = svc.DecideJoinRequest(officialRequestContext(ctx), officialActor(ctx), req.Id, string(req.Body.Decision), req.Body.IdempotencyKey)
		}
	}
	if err == nil {
		return apicontract.DecideQQOfficialJoinRequest202JSONResponse{QQOfficialAcceptedJSONResponse: qqAccepted(row)}, nil
	}
	code := officialErrorCode(err)
	failure := officialError(code, err)
	switch code {
	case 400:
		return apicontract.DecideQQOfficialJoinRequest400JSONResponse{QQOfficialBadRequestJSONResponse: apicontract.QQOfficialBadRequestJSONResponse(failure)}, nil
	case 404:
		return apicontract.DecideQQOfficialJoinRequest404JSONResponse{QQOfficialNotFoundJSONResponse: apicontract.QQOfficialNotFoundJSONResponse(failure)}, nil
	case 409:
		return apicontract.DecideQQOfficialJoinRequest409JSONResponse{QQOfficialConflictJSONResponse: apicontract.QQOfficialConflictJSONResponse(failure)}, nil
	case 429:
		return apicontract.DecideQQOfficialJoinRequest429JSONResponse{QQOfficialRateLimitedJSONResponse: apicontract.QQOfficialRateLimitedJSONResponse(failure)}, nil
	default:
		return apicontract.DecideQQOfficialJoinRequest503JSONResponse{QQOfficialUnavailableJSONResponse: apicontract.QQOfficialUnavailableJSONResponse(failure)}, nil
	}
}

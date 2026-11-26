package handler

import (
	"amiya-eden/global"
	"amiya-eden/internal/apicontract"
	"amiya-eden/internal/middleware"
	"amiya-eden/internal/service"
	"amiya-eden/internal/taskregistry"
	"amiya-eden/pkg/response"
	"context"
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
)

type TaskHandler struct {
	svc *service.TaskService
}

func NewTaskHandler(svc *service.TaskService) *TaskHandler {
	return &TaskHandler{svc: svc}
}

func (h *TaskHandler) getManagedTasks() ([]apicontract.ManagedTaskDefinition, *taskFailure) {
	items, err := h.svc.GetTasks()
	if err != nil {
		return nil, taskBizFailure("获取任务列表失败: " + err.Error())
	}

	dtos := make([]apicontract.ManagedTaskDefinition, len(items))
	for i, item := range items {
		dto := apicontract.ManagedTaskDefinition{
			Name: item.Name, Description: item.Description, Runnable: item.Runnable,
			Category: apicontract.ManagedTaskCategory(item.Category), Type: apicontract.ManagedTaskType(item.Type),
			CronExpr: item.CronExpr, DefaultCron: item.DefaultCron,
		}
		if last := item.LastExecution; last != nil {
			dto.LastExecution = &apicontract.ManagedTaskLastExecution{
				Status: apicontract.ManagedTaskRunStatus(last.Status), StartedAt: last.StartedAt,
				FinishedAt: last.FinishedAt, DurationMs: last.DurationMs, Error: last.Error, Summary: last.Summary,
			}
		}
		dtos[i] = dto
	}
	return dtos, nil
}

func (h *TaskHandler) listManagedTaskHistory(params apicontract.ListManagedTaskHistoryParams) (apicontract.ManagedTaskHistoryPage, *taskFailure) {
	page, pageSize := taskPage(params.Current, params.Size)
	executions, total, err := h.svc.GetExecutionHistory(taskQueryValue(params.TaskName), taskQueryValue(params.Status), page, pageSize)
	if err != nil {
		return apicontract.ManagedTaskHistoryPage{}, taskBizFailure("获取执行历史失败: " + err.Error())
	}

	var dtos []apicontract.ManagedTaskExecution
	if executions != nil {
		dtos = make([]apicontract.ManagedTaskExecution, len(executions))
	}
	for i, execution := range executions {
		dtos[i] = apicontract.ManagedTaskExecution{
			Id: execution.ID, TaskName: execution.TaskName, Trigger: apicontract.ManagedTaskTrigger(execution.Trigger),
			TriggeredBy: execution.TriggeredBy, TriggeredByName: execution.TriggeredByName,
			Status: apicontract.ManagedTaskRunStatus(execution.Status), StartedAt: execution.StartedAt,
			FinishedAt: execution.FinishedAt, DurationMs: execution.DurationMs, Error: execution.Error, Summary: execution.Summary,
		}
	}
	return apicontract.ManagedTaskHistoryPage{List: dtos, Total: total, Page: page, PageSize: pageSize}, nil
}

func (h *TaskHandler) runManagedTask(triggeredBy uint, taskName string) (apicontract.TaskCommandMessage, *taskFailure) {
	definition, ok := h.svc.Registry().Get(taskName)
	if !ok {
		return apicontract.TaskCommandMessage{}, &taskFailure{response.CodeNotFound, "任务不存在"}
	}
	if definition.RunFunc == nil {
		return apicontract.TaskCommandMessage{}, taskBizFailure("该任务不支持手动触发")
	}

	handle, ok := h.svc.Registry().TryLock(taskName)
	if !ok {
		return apicontract.TaskCommandMessage{}, &taskFailure{http.StatusConflict, "任务正在运行中，请稍后再试"}
	}

	if ok := global.EnsureBackgroundTaskManager().Go("task_manual_"+taskName, func(ctx context.Context) error {
		ctx = taskregistry.ContextWithLockHandle(ctx, handle)
		return h.svc.RunTaskLocked(ctx, taskName, &triggeredBy)
	}); !ok {
		handle.Release()
		return apicontract.TaskCommandMessage{}, taskBizFailure("服务正在关闭，任务未启动")
	}

	return apicontract.TaskCommandMessage{Message: "任务已触发"}, nil
}

func (h *TaskHandler) updateManagedTaskSchedule(userID uint, taskName string, req apicontract.ManagedTaskScheduleInput) (apicontract.TaskCommandMessage, *taskFailure) {
	err := h.svc.UpdateSchedule(taskName, req.CronExpr, userID)
	switch {
	case err == nil:
		return apicontract.TaskCommandMessage{Message: "调度已更新"}, nil
	case errors.Is(err, service.ErrTaskNotFound):
		return apicontract.TaskCommandMessage{}, &taskFailure{response.CodeNotFound, "任务不存在"}
	case errors.Is(err, service.ErrTaskNotRecurring):
		return apicontract.TaskCommandMessage{}, &taskFailure{response.CodeParamError, "仅周期任务支持修改调度频率"}
	case errors.Is(err, service.ErrInvalidCronExpr):
		return apicontract.TaskCommandMessage{}, &taskFailure{response.CodeParamError, err.Error()}
	default:
		return apicontract.TaskCommandMessage{}, taskBizFailure("更新调度失败: " + err.Error())
	}
}

type taskFailure struct {
	code    int
	message string
}

func taskBizFailure(message string) *taskFailure {
	return &taskFailure{response.CodeBizError, message}
}

func taskPage(current, size *int) (int, int) {
	page, pageSize := normalizePage(taskQueryValue(current)), taskQueryValue(size)
	if pageSize < 1 {
		pageSize = 20
	}
	return page, min(pageSize, 1000)
}

func taskQueryValue[T any](value *T) T {
	if value != nil {
		return *value
	}
	var zero T
	return zero
}

func (f *taskFailure) businessError() apicontract.BusinessError {
	return contractBusinessError(f.code, f.message)
}

func taskMissingBody() apicontract.BusinessError {
	return contractBusinessError(response.CodeParamError, "参数错误")
}

func taskStrictActor(ctx context.Context) (uint, bool) {
	c, _ := ctx.(*gin.Context)
	if c == nil {
		return 0, false
	}
	id := middleware.GetUserID(c)
	return id, id != 0
}

func taskUnauthorized() apicontract.UnauthorizedJSONResponse {
	return apicontract.UnauthorizedJSONResponse{Code: response.CodeUnauthorized, Msg: "未登录"}
}

func (h *TaskHandler) GetManagedTasks(_ context.Context, _ apicontract.GetManagedTasksRequestObject) (apicontract.GetManagedTasksResponseObject, error) {
	out := &apicontract.GetManagedTasks200JSONResponse{}
	data, failure := h.getManagedTasks()
	if failure != nil {
		return out, out.FromBusinessError(failure.businessError())
	}
	return out, out.FromManagedTasksSuccess(apicontract.ManagedTasksSuccess{Code: response.CodeOK, Msg: "success", Data: data})
}

func (h *TaskHandler) ListManagedTaskHistory(_ context.Context, request apicontract.ListManagedTaskHistoryRequestObject) (apicontract.ListManagedTaskHistoryResponseObject, error) {
	out := &apicontract.ListManagedTaskHistory200JSONResponse{}
	data, failure := h.listManagedTaskHistory(request.Params)
	if failure != nil {
		return out, out.FromBusinessError(failure.businessError())
	}
	return out, out.FromManagedTaskHistorySuccess(apicontract.ManagedTaskHistorySuccess{Code: response.CodeOK, Msg: "success", Data: data})
}

func (h *TaskHandler) RunManagedTask(ctx context.Context, request apicontract.RunManagedTaskRequestObject) (apicontract.RunManagedTaskResponseObject, error) {
	actor, ok := taskStrictActor(ctx)
	if !ok {
		return apicontract.RunManagedTask401JSONResponse{UnauthorizedJSONResponse: taskUnauthorized()}, nil
	}
	data, failure := h.runManagedTask(actor, request.Name)
	out := &apicontract.RunManagedTask200JSONResponse{}
	if failure != nil {
		if failure.code == http.StatusConflict {
			return apicontract.RunManagedTask409JSONResponse{Code: http.StatusConflict, Msg: failure.message}, nil
		}
		return out, out.FromBusinessError(failure.businessError())
	}
	return out, out.FromTaskCommandSuccess(apicontract.TaskCommandSuccess{Code: response.CodeOK, Msg: "success", Data: data})
}

func (h *TaskHandler) UpdateManagedTaskSchedule(ctx context.Context, request apicontract.UpdateManagedTaskScheduleRequestObject) (apicontract.UpdateManagedTaskScheduleResponseObject, error) {
	actor, ok := taskStrictActor(ctx)
	if !ok {
		return apicontract.UpdateManagedTaskSchedule401JSONResponse{UnauthorizedJSONResponse: taskUnauthorized()}, nil
	}
	out := &apicontract.UpdateManagedTaskSchedule200JSONResponse{}
	if request.Body == nil {
		return out, out.FromBusinessError(taskMissingBody())
	}
	data, failure := h.updateManagedTaskSchedule(actor, request.Name, *request.Body)
	if failure != nil {
		return out, out.FromBusinessError(failure.businessError())
	}
	return out, out.FromTaskCommandSuccess(apicontract.TaskCommandSuccess{Code: response.CodeOK, Msg: "success", Data: data})
}

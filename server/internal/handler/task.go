package handler

import (
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

func (h *TaskHandler) GetTasks(c *gin.Context) {
	items, err := h.svc.GetTasks()
	if err != nil {
		response.Fail(c, response.CodeBizError, "获取任务列表失败: "+err.Error())
		return
	}

	response.OK(c, items)
}

func (h *TaskHandler) GetHistory(c *gin.Context) {
	page, pageSize, err := parseUnboundedPaginationQuery(c, 20)
	if err != nil {
		response.Fail(c, response.CodeParamError, err.Error())
		return
	}

	executions, total, err := h.svc.GetExecutionHistory(c.Query("task_name"), c.Query("status"), page, pageSize)
	if err != nil {
		response.Fail(c, response.CodeBizError, "获取执行历史失败: "+err.Error())
		return
	}

	response.OKWithPage(c, executions, total, page, pageSize)
}

func (h *TaskHandler) RunTask(c *gin.Context) {
	taskName := c.Param("name")
	triggeredBy := middleware.GetUserID(c)

	definition, ok := h.svc.Registry().Get(taskName)
	if !ok {
		response.Fail(c, response.CodeNotFound, "任务不存在")
		return
	}
	if definition.RunFunc == nil {
		response.Fail(c, response.CodeBizError, "该任务不支持手动触发")
		return
	}

	handle, ok := h.svc.Registry().TryLock(taskName)
	if !ok {
		c.JSON(http.StatusConflict, gin.H{
			"code": http.StatusConflict,
			"msg":  "任务正在运行中，请稍后再试",
		})
		return
	}

	go func() {
		ctx := taskregistry.ContextWithLockHandle(context.Background(), handle)
		_ = h.svc.RunTaskLocked(ctx, taskName, &triggeredBy)
	}()

	response.OK(c, gin.H{"message": "任务已触发"})
}

func (h *TaskHandler) UpdateSchedule(c *gin.Context) {
	var req struct {
		CronExpr string `json:"cron_expr" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, response.CodeParamError, "参数错误: "+err.Error())
		return
	}

	err := h.svc.UpdateSchedule(c.Param("name"), req.CronExpr, middleware.GetUserID(c))
	if err != nil {
		switch {
		case errors.Is(err, service.ErrTaskNotFound):
			response.Fail(c, response.CodeNotFound, "任务不存在")
		case errors.Is(err, service.ErrTaskNotRecurring):
			response.Fail(c, response.CodeParamError, "仅周期任务支持修改调度频率")
		case errors.Is(err, service.ErrInvalidCronExpr):
			response.Fail(c, response.CodeParamError, err.Error())
		default:
			response.Fail(c, response.CodeBizError, "更新调度失败: "+err.Error())
		}
		return
	}

	response.OK(c, gin.H{"message": "调度已更新"})
}

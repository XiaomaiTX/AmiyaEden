package handler

import (
	"amiya-eden/global"
	"amiya-eden/internal/middleware"
	"amiya-eden/internal/repository"
	"amiya-eden/jobs"
	"amiya-eden/pkg/eve/esi"
	"amiya-eden/pkg/response"
	"context"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

// ESIRefreshHandler ESI 数据刷新队列处理器
type ESIRefreshHandler struct{}

func NewESIRefreshHandler() *ESIRefreshHandler {
	return &ESIRefreshHandler{}
}

// TaskInfoItem 任务定义信息（用于前端展示所有可用任务）
type TaskInfoItem struct {
	Name             string   `json:"name"`
	Description      string   `json:"description"`
	Priority         int      `json:"priority"`
	ActiveInterval   string   `json:"active_interval"`
	InactiveInterval string   `json:"inactive_interval"`
	RequiredScopes   []string `json:"required_scopes"`
}

type TaskStatusItem struct {
	TaskName      string     `json:"task_name"`
	Description   string     `json:"description"`
	CharacterID   int64      `json:"character_id"`
	CharacterName string     `json:"character_name,omitempty"`
	Priority      int        `json:"priority"`
	LastRun       *time.Time `json:"last_run,omitempty"`
	NextRun       *time.Time `json:"next_run,omitempty"`
	Status        string     `json:"status"`
	Error         string     `json:"error,omitempty"`
}

// GetTasks 获取所有已注册的刷新任务定义
//
// GET /api/v1/tasks/esi/tasks
func (h *ESIRefreshHandler) GetTasks(c *gin.Context) {
	allTasks := esi.AllTasks()
	items := make([]TaskInfoItem, 0, len(allTasks))
	for _, task := range allTasks {
		scopes := make([]string, 0)
		for _, scope := range task.RequiredScopes() {
			scopes = append(scopes, scope.Scope)
		}
		items = append(items, TaskInfoItem{
			Name:             task.Name(),
			Description:      task.Description(),
			Priority:         int(task.Priority()),
			ActiveInterval:   formatDuration(task.Interval().Active),
			InactiveInterval: formatDuration(task.Interval().Inactive),
			RequiredScopes:   scopes,
		})
	}

	sort.Slice(items, func(i, j int) bool {
		return items[i].Priority < items[j].Priority
	})

	response.OK(c, items)
}

// GetStatuses 获取所有任务的运行时状态（支持分页和筛选）
//
// GET /api/v1/tasks/esi/statuses?current=1&size=20&task_name=xxx&status=xxx&character=xxx
func (h *ESIRefreshHandler) GetStatuses(c *gin.Context) {
	queue := jobs.GetESIQueue()
	if queue == nil {
		response.OKWithPage(c, []interface{}{}, 0, 1, 20)
		return
	}

	all := queue.GetAllStatuses()
	charRepo := repository.NewEveCharacterRepository()
	characterNames := make(map[int64]string, len(all))
	characterIDs := make([]int64, 0, len(all))
	seenCharacterIDs := make(map[int64]struct{}, len(all))
	for _, status := range all {
		if _, exists := seenCharacterIDs[status.CharacterID]; exists {
			continue
		}
		seenCharacterIDs[status.CharacterID] = struct{}{}
		characterIDs = append(characterIDs, status.CharacterID)
	}
	if chars, err := charRepo.ListByCharacterIDs(characterIDs); err == nil {
		for _, char := range chars {
			characterNames[char.CharacterID] = char.CharacterName
		}
	}

	// 筛选
	taskNameFilter := c.Query("task_name")
	statusFilter := c.Query("status")
	characterFilter := strings.TrimSpace(c.Query("character"))
	characterFilterLower := strings.ToLower(characterFilter)
	characterIDFilter, characterParseErr := strconv.ParseInt(characterFilter, 10, 64)

	filtered := make([]TaskStatusItem, 0, len(all))
	for _, s := range all {
		if taskNameFilter != "" && s.TaskName != taskNameFilter {
			continue
		}
		if statusFilter != "" && s.Status != statusFilter {
			continue
		}
		characterName := characterNames[s.CharacterID]
		if characterFilter != "" {
			matchesCharacterID := characterParseErr == nil && s.CharacterID == characterIDFilter
			matchesCharacterName := strings.Contains(strings.ToLower(characterName), characterFilterLower)
			if !matchesCharacterID && !matchesCharacterName {
				continue
			}
		}
		filtered = append(filtered, TaskStatusItem{
			TaskName:      s.TaskName,
			Description:   s.Description,
			CharacterID:   s.CharacterID,
			CharacterName: characterName,
			Priority:      int(s.Priority),
			LastRun:       s.LastRun,
			NextRun:       s.NextRun,
			Status:        s.Status,
			Error:         s.Error,
		})
	}

	total := len(filtered)

	// 分页
	page, pageSize, err := parseUnboundedPaginationQuery(c, 20)
	if err != nil {
		response.Fail(c, response.CodeParamError, err.Error())
		return
	}

	start := (page - 1) * pageSize
	if start > total {
		start = total
	}
	end := start + pageSize
	if end > total {
		end = total
	}

	response.OKWithPage(c, filtered[start:end], int64(total), page, pageSize)
}

// RunTaskRequest 手动触发单个任务的请求（指定人物）
type RunTaskRequest struct {
	TaskName    string `json:"task_name" binding:"required"`
	CharacterID int64  `json:"character_id" binding:"required"`
}

// RunTask 手动触发指定任务（指定人物）
//
// POST /api/v1/tasks/esi/run
func (h *ESIRefreshHandler) RunTask(c *gin.Context) {
	var req RunTaskRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, response.CodeParamError, "参数错误: "+err.Error())
		return
	}

	queue := jobs.GetESIQueue()
	if queue == nil {
		response.Fail(c, response.CodeBizError, "刷新队列未初始化")
		return
	}

	if err := queue.RunTask(c.Request.Context(), req.TaskName, req.CharacterID); err != nil {
		response.Fail(c, response.CodeBizError, "任务触发失败: "+err.Error())
		return
	}

	response.OK(c, gin.H{"message": "任务已触发"})
}

// RunMyCharacterTask 手动触发指定任务（仅限自己的角色）
//
// POST /api/v1/info/esi-refresh
func (h *ESIRefreshHandler) RunMyCharacterTask(c *gin.Context) {
	userID := middleware.GetUserID(c)

	var req RunTaskRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, response.CodeParamError, "参数错误: "+err.Error())
		return
	}

	charRepo := repository.NewEveCharacterRepository()
	char, err := charRepo.GetByCharacterID(req.CharacterID)
	if err != nil {
		response.Fail(c, response.CodeBizError, "角色不存在")
		return
	}
	if char.UserID != userID {
		response.Fail(c, response.CodeForbidden, "无权操作此角色")
		return
	}

	queue := jobs.GetESIQueue()
	if queue == nil {
		response.Fail(c, response.CodeBizError, "刷新队列未初始化")
		return
	}

	if err := queue.RunTask(c.Request.Context(), req.TaskName, req.CharacterID); err != nil {
		response.Fail(c, response.CodeBizError, "任务触发失败: "+err.Error())
		return
	}

	response.OK(c, gin.H{"message": "任务已触发"})
}

// RunTaskByNameRequest 按任务名称触发所有人物
type RunTaskByNameRequest struct {
	TaskName string `json:"task_name" binding:"required"`
}

// RunTaskByName 手动触发指定任务（所有人物）
//
// POST /api/v1/tasks/esi/run-task
func (h *ESIRefreshHandler) RunTaskByName(c *gin.Context) {
	var req RunTaskByNameRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, response.CodeParamError, "参数错误: "+err.Error())
		return
	}

	queue := jobs.GetESIQueue()
	if queue == nil {
		response.Fail(c, response.CodeBizError, "刷新队列未初始化")
		return
	}

	if ok := global.EnsureBackgroundTaskManager().Go("esi_run_task_by_name", func(ctx context.Context) error {
		return queue.RunTaskByName(ctx, req.TaskName)
	}); !ok {
		response.Fail(c, response.CodeBizError, "服务正在关闭，任务未启动")
		return
	}

	response.OK(c, gin.H{"message": fmt.Sprintf("任务 %s 已触发（所有人物）", req.TaskName)})
}

// RunAll 手动触发全量刷新
//
// POST /api/v1/tasks/esi/run-all
func (h *ESIRefreshHandler) RunAll(c *gin.Context) {
	queue := jobs.GetESIQueue()
	if queue == nil {
		response.Fail(c, response.CodeBizError, "刷新队列未初始化")
		return
	}

	if ok := global.EnsureBackgroundTaskManager().Go("esi_run_all", queue.Run); !ok {
		response.Fail(c, response.CodeBizError, "服务正在关闭，任务未启动")
		return
	}
	response.OK(c, gin.H{"message": "全量刷新已触发"})
}

// formatDuration 格式化 time.Duration 为可读字符串。
func formatDuration(d time.Duration) string {
	if d >= 24*time.Hour {
		days := int(d / (24 * time.Hour))
		if days == 1 {
			return "1 Day"
		}
		return fmt.Sprintf("%d Days", days)
	}
	if d >= time.Hour {
		hours := int(d / time.Hour)
		if hours == 1 {
			return "1 Hour"
		}
		return fmt.Sprintf("%d Hours", hours)
	}
	if d >= time.Minute {
		minutes := int(d / time.Minute)
		if minutes == 1 {
			return "1 Minute"
		}
		return fmt.Sprintf("%d Minutes", minutes)
	}
	return d.String()
}

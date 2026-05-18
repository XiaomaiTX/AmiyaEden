package handler

import (
	"amiya-eden/global"
	"amiya-eden/internal/middleware"
	"amiya-eden/internal/service"
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
type ESIRefreshHandler struct {
	userSvc *service.UserService
}

func NewESIRefreshHandler() *ESIRefreshHandler {
	return &ESIRefreshHandler{userSvc: service.NewUserService()}
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

type MonitorOverview struct {
	Total    int `json:"total"`
	Healthy  int `json:"healthy"`
	Warning  int `json:"warning"`
	Critical int `json:"critical"`
	Running  int `json:"running"`
	Failed   int `json:"failed"`
	Overdue  int `json:"overdue"`
}

type MonitorTaskPanelItem struct {
	TaskName        string  `json:"task_name"`
	Description     string  `json:"description"`
	Priority        int     `json:"priority"`
	Total           int     `json:"total"`
	Healthy         int     `json:"healthy"`
	Warning         int     `json:"warning"`
	Critical        int     `json:"critical"`
	Running         int     `json:"running"`
	Failed          int     `json:"failed"`
	Overdue         int     `json:"overdue"`
	SuccessRate     float64 `json:"success_rate"`
	WorstLagSeconds int64   `json:"worst_lag_seconds"`
}

type MonitorFailureItem struct {
	TaskName      string     `json:"task_name"`
	Description   string     `json:"description"`
	CharacterID   int64      `json:"character_id"`
	CharacterName string     `json:"character_name,omitempty"`
	Error         string     `json:"error"`
	LastRun       *time.Time `json:"last_run,omitempty"`
}

type MonitorOverdueItem struct {
	TaskName       string     `json:"task_name"`
	Description    string     `json:"description"`
	CharacterID    int64      `json:"character_id"`
	CharacterName  string     `json:"character_name,omitempty"`
	NextRun        *time.Time `json:"next_run,omitempty"`
	OverdueSeconds int64      `json:"overdue_seconds"`
}

type MonitorResponse struct {
	GeneratedAt time.Time              `json:"generated_at"`
	Overview    MonitorOverview        `json:"overview"`
	TaskPanels  []MonitorTaskPanelItem `json:"task_panels"`
	FailureTop  []MonitorFailureItem   `json:"failure_top"`
	OverdueTop  []MonitorOverdueItem   `json:"overdue_top"`
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
	characterNames := h.buildCharacterNameMap(all)

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

const monitorListLimit = 20

// GetMonitor 获取 ESI 队列聚合监控数据。
//
// GET /api/v1/tasks/esi/monitor
func (h *ESIRefreshHandler) GetMonitor(c *gin.Context) {
	queue := jobs.GetESIQueue()
	if queue == nil {
		response.OK(c, MonitorResponse{
			GeneratedAt: time.Now().UTC(),
			Overview:    MonitorOverview{},
			TaskPanels:  []MonitorTaskPanelItem{},
			FailureTop:  []MonitorFailureItem{},
			OverdueTop:  []MonitorOverdueItem{},
		})
		return
	}

	now := time.Now().UTC()
	statuses := queue.GetAllStatuses()
	characterNames := h.buildCharacterNameMap(statuses)

	panelByTask := make(map[string]*MonitorTaskPanelItem)
	failureTop := make([]MonitorFailureItem, 0, len(statuses))
	overdueTop := make([]MonitorOverdueItem, 0, len(statuses))
	overview := MonitorOverview{Total: len(statuses)}

	for _, status := range statuses {
		if status == nil {
			continue
		}
		panel := ensureTaskPanel(panelByTask, status)
		panel.Total++

		overdueSeconds := calcOverdueSeconds(status, now)
		if overdueSeconds > 0 {
			panel.Overdue++
			overview.Overdue++
			overdueTop = append(overdueTop, MonitorOverdueItem{
				TaskName:       status.TaskName,
				Description:    status.Description,
				CharacterID:    status.CharacterID,
				CharacterName:  characterNames[status.CharacterID],
				NextRun:        status.NextRun,
				OverdueSeconds: overdueSeconds,
			})
		}

		if strings.TrimSpace(status.Error) != "" {
			failureTop = append(failureTop, MonitorFailureItem{
				TaskName:      status.TaskName,
				Description:   status.Description,
				CharacterID:   status.CharacterID,
				CharacterName: characterNames[status.CharacterID],
				Error:         status.Error,
				LastRun:       status.LastRun,
			})
		}

		if status.Status == "running" {
			panel.Running++
			overview.Running++
		}
		if status.Status == "failed" {
			panel.Failed++
			overview.Failed++
		}
		if status.Status == "success" {
			panel.SuccessRate += 1
		}
		if overdueSeconds > panel.WorstLagSeconds {
			panel.WorstLagSeconds = overdueSeconds
		}

		expectedIntervalSec := expectedIntervalSeconds(status)
		severity := classifySeverity(status, overdueSeconds, expectedIntervalSec)
		switch severity {
		case "critical":
			panel.Critical++
			overview.Critical++
		case "warning":
			panel.Warning++
			overview.Warning++
		default:
			panel.Healthy++
			overview.Healthy++
		}
	}

	taskPanels := make([]MonitorTaskPanelItem, 0, len(panelByTask))
	for _, panel := range panelByTask {
		if panel.Total > 0 {
			panel.SuccessRate = panel.SuccessRate / float64(panel.Total)
		}
		taskPanels = append(taskPanels, *panel)
	}

	sort.Slice(taskPanels, func(i, j int) bool {
		if taskPanels[i].Critical != taskPanels[j].Critical {
			return taskPanels[i].Critical > taskPanels[j].Critical
		}
		if taskPanels[i].Failed != taskPanels[j].Failed {
			return taskPanels[i].Failed > taskPanels[j].Failed
		}
		if taskPanels[i].Overdue != taskPanels[j].Overdue {
			return taskPanels[i].Overdue > taskPanels[j].Overdue
		}
		if taskPanels[i].Priority != taskPanels[j].Priority {
			return taskPanels[i].Priority < taskPanels[j].Priority
		}
		return taskPanels[i].TaskName < taskPanels[j].TaskName
	})

	sort.Slice(failureTop, func(i, j int) bool {
		left := failureTop[i].LastRun
		right := failureTop[j].LastRun
		if left == nil && right == nil {
			if failureTop[i].TaskName != failureTop[j].TaskName {
				return failureTop[i].TaskName < failureTop[j].TaskName
			}
			return failureTop[i].CharacterID < failureTop[j].CharacterID
		}
		if left == nil {
			return false
		}
		if right == nil {
			return true
		}
		if !left.Equal(*right) {
			return left.After(*right)
		}
		if failureTop[i].TaskName != failureTop[j].TaskName {
			return failureTop[i].TaskName < failureTop[j].TaskName
		}
		return failureTop[i].CharacterID < failureTop[j].CharacterID
	})

	sort.Slice(overdueTop, func(i, j int) bool {
		if overdueTop[i].OverdueSeconds != overdueTop[j].OverdueSeconds {
			return overdueTop[i].OverdueSeconds > overdueTop[j].OverdueSeconds
		}
		if overdueTop[i].TaskName != overdueTop[j].TaskName {
			return overdueTop[i].TaskName < overdueTop[j].TaskName
		}
		return overdueTop[i].CharacterID < overdueTop[j].CharacterID
	})

	if len(failureTop) > monitorListLimit {
		failureTop = failureTop[:monitorListLimit]
	}
	if len(overdueTop) > monitorListLimit {
		overdueTop = overdueTop[:monitorListLimit]
	}

	response.OK(c, MonitorResponse{
		GeneratedAt: now,
		Overview:    overview,
		TaskPanels:  taskPanels,
		FailureTop:  failureTop,
		OverdueTop:  overdueTop,
	})
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
	if !middleware.GetCorpRuleBool(c, service.CorpRuleSystemTaskAllowManualRun, true) {
		response.Fail(c, response.CodeForbidden, "当前军团策略禁止手动触发任务")
		return
	}

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

	char, err := h.userSvc.GetCharacterByID(req.CharacterID)
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
	if !middleware.GetCorpRuleBool(c, service.CorpRuleSystemTaskAllowManualRun, true) {
		response.Fail(c, response.CodeForbidden, "当前军团策略禁止手动触发任务")
		return
	}

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
	if !middleware.GetCorpRuleBool(c, service.CorpRuleSystemTaskAllowManualRun, true) {
		response.Fail(c, response.CodeForbidden, "当前军团策略禁止手动触发任务")
		return
	}

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

func (h *ESIRefreshHandler) buildCharacterNameMap(statuses []*esi.TaskStatus) map[int64]string {
	characterIDs := make([]int64, 0, len(statuses))
	seenCharacterIDs := make(map[int64]struct{}, len(statuses))
	for _, status := range statuses {
		if status == nil {
			continue
		}
		if _, exists := seenCharacterIDs[status.CharacterID]; exists {
			continue
		}
		seenCharacterIDs[status.CharacterID] = struct{}{}
		characterIDs = append(characterIDs, status.CharacterID)
	}
	return h.userSvc.ListCharacterNamesByIDs(characterIDs)
}

func ensureTaskPanel(panelByTask map[string]*MonitorTaskPanelItem, status *esi.TaskStatus) *MonitorTaskPanelItem {
	if panel, ok := panelByTask[status.TaskName]; ok {
		return panel
	}
	panel := &MonitorTaskPanelItem{
		TaskName:    status.TaskName,
		Description: status.Description,
		Priority:    int(status.Priority),
	}
	panelByTask[status.TaskName] = panel
	return panel
}

func calcOverdueSeconds(status *esi.TaskStatus, now time.Time) int64 {
	if status.NextRun == nil {
		return 0
	}
	nextRun := status.NextRun.UTC()
	if !nextRun.Before(now) {
		return 0
	}
	return int64(now.Sub(nextRun).Seconds())
}

func expectedIntervalSeconds(status *esi.TaskStatus) int64 {
	if status.NextRun != nil && status.LastRun != nil {
		interval := status.NextRun.UTC().Sub(status.LastRun.UTC())
		if interval > 0 {
			return int64(interval.Seconds())
		}
	}
	if task, ok := esi.GetTask(status.TaskName); ok {
		active := task.Interval().Active
		if active > 0 {
			return int64(active.Seconds())
		}
	}
	return int64(time.Hour.Seconds())
}

func classifySeverity(status *esi.TaskStatus, overdueSeconds, expectedIntervalSeconds int64) string {
	if status.Status == "failed" {
		return "critical"
	}
	if overdueSeconds > 0 && overdueSeconds >= expectedIntervalSeconds {
		return "critical"
	}
	if status.Status == "running" || status.Status == "pending" {
		return "warning"
	}
	if overdueSeconds > 0 {
		return "warning"
	}
	return "healthy"
}

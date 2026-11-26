package handler

import (
	"amiya-eden/global"
	"amiya-eden/internal/apicontract"
	"amiya-eden/internal/model"
	"amiya-eden/internal/service"
	"amiya-eden/jobs"
	"amiya-eden/pkg/eve/esi"
	"amiya-eden/pkg/response"
	"context"
	"encoding/json"
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

func (h *ESIRefreshHandler) listEsiRefreshTasks() []apicontract.EsiRefreshTask {
	allTasks := esi.AllTasks()
	items := make([]apicontract.EsiRefreshTask, 0, len(allTasks))
	for _, task := range allTasks {
		scopes := make([]string, 0)
		for _, scope := range task.RequiredScopes() {
			scopes = append(scopes, scope.Scope)
		}
		interval := esi.ResolveInterval(task.Name())
		items = append(items, apicontract.EsiRefreshTask{
			Name:                    task.Name(),
			Description:             task.Description(),
			Priority:                int(task.Priority()),
			ActiveInterval:          formatDuration(interval.Active),
			InactiveInterval:        formatDuration(interval.Inactive),
			ActiveIntervalMinutes:   int(interval.Active.Minutes()),
			InactiveIntervalMinutes: int(interval.Inactive.Minutes()),
			RequiredScopes:          scopes,
		})
	}

	sort.Slice(items, func(i, j int) bool {
		return items[i].Priority < items[j].Priority
	})

	return items
}

func emptyEsiStatuses() apicontract.EsiRefreshStatusPage {
	return apicontract.EsiRefreshStatusPage{List: []apicontract.EsiRefreshStatus{}, Page: 1, PageSize: 20}
}

func (h *ESIRefreshHandler) listEsiRefreshStatuses(params apicontract.ListEsiRefreshStatusesParams) apicontract.EsiRefreshStatusPage {
	queue := jobs.GetESIQueue()
	if queue == nil {
		return emptyEsiStatuses()
	}

	all := queue.GetAllStatuses()
	characterNames := h.buildCharacterNameMap(all)

	// 筛选
	taskNameFilter := taskQueryValue(params.TaskName)
	statusFilter := taskQueryValue(params.Status)
	characterFilter := strings.TrimSpace(taskQueryValue(params.Character))
	characterFilterLower := strings.ToLower(characterFilter)
	characterIDFilter, characterParseErr := strconv.ParseInt(characterFilter, 10, 64)

	filtered := make([]apicontract.EsiRefreshStatus, 0, len(all))
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
		filtered = append(filtered, apicontract.EsiRefreshStatus{
			TaskName:      s.TaskName,
			Description:   s.Description,
			CharacterID:   s.CharacterID,
			CharacterName: characterName,
			Priority:      int(s.Priority),
			LastRun:       s.LastRun,
			NextRun:       s.NextRun,
			Status:        apicontract.EsiRefreshState(s.Status),
			Error:         s.Error,
		})
	}

	total := len(filtered)
	page, pageSize := taskPage(params.Current, params.Size)
	start, end := esiStatusPageBounds(total, page, pageSize)
	return apicontract.EsiRefreshStatusPage{List: filtered[start:end], Total: int64(total), Page: page, PageSize: pageSize}
}

// esiStatusPageBounds expects a normalized page and page size from pagination parsing.
func esiStatusPageBounds(total, page, pageSize int) (start, end int) {
	// Bound the offset before multiplying so distant pages cannot overflow.
	start = total
	if pageIndex := page - 1; pageIndex <= total/pageSize {
		start = pageIndex * pageSize
	}
	return start, start + min(pageSize, total-start)
}

const monitorListLimit = 20

func (h *ESIRefreshHandler) getEsiRefreshMonitor() apicontract.EsiRefreshMonitorSnapshot {
	queue := jobs.GetESIQueue()
	if queue == nil {
		return apicontract.EsiRefreshMonitorSnapshot{
			GeneratedAt: time.Now().UTC(),
			Overview:    apicontract.EsiRefreshMonitorOverview{},
			TaskPanels:  []apicontract.EsiRefreshMonitorTaskPanel{},
			FailureTop:  []apicontract.EsiRefreshMonitorFailure{},
			OverdueTop:  []apicontract.EsiRefreshMonitorOverdue{},
		}
	}

	now := time.Now().UTC()
	statuses := queue.GetAllStatuses()
	characterNames := h.buildCharacterNameMap(statuses)

	panelByTask := make(map[string]*apicontract.EsiRefreshMonitorTaskPanel)
	failureTop := make([]apicontract.EsiRefreshMonitorFailure, 0, len(statuses))
	overdueTop := make([]apicontract.EsiRefreshMonitorOverdue, 0, len(statuses))
	overview := apicontract.EsiRefreshMonitorOverview{Total: len(statuses)}

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
			overdueTop = append(overdueTop, apicontract.EsiRefreshMonitorOverdue{
				TaskName:       status.TaskName,
				Description:    status.Description,
				CharacterID:    status.CharacterID,
				CharacterName:  characterNames[status.CharacterID],
				NextRun:        status.NextRun,
				OverdueSeconds: overdueSeconds,
			})
		}

		if strings.TrimSpace(status.Error) != "" {
			failureTop = append(failureTop, apicontract.EsiRefreshMonitorFailure{
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

	taskPanels := make([]apicontract.EsiRefreshMonitorTaskPanel, 0, len(panelByTask))
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

	return apicontract.EsiRefreshMonitorSnapshot{
		GeneratedAt: now,
		Overview:    overview,
		TaskPanels:  taskPanels,
		FailureTop:  failureTop,
		OverdueTop:  overdueTop,
	}
}

// RunTaskRequest 手动触发单个任务的请求（指定人物）
type RunTaskRequest struct {
	TaskName    string `json:"task_name" binding:"required"`
	CharacterID int64  `json:"character_id" binding:"required"`
}

func (h *ESIRefreshHandler) runEsiRefreshTask(ctx context.Context, req apicontract.EsiRefreshRunInput) (apicontract.TaskCommandMessage, *taskFailure) {
	queue := jobs.GetESIQueue()
	if queue == nil {
		return apicontract.TaskCommandMessage{}, taskBizFailure("刷新队列未初始化")
	}

	if err := queue.RunTask(ctx, req.TaskName, req.CharacterID); err != nil {
		return apicontract.TaskCommandMessage{}, taskBizFailure("任务触发失败: " + err.Error())
	}
	return apicontract.TaskCommandMessage{Message: "任务已触发"}, nil
}

func (h *ESIRefreshHandler) runMyCharacterTask(ctx context.Context, userID uint, input apicontract.RunMyCharacterTaskJSONRequestBody) (apicontract.TaskCommandMessage, *taskFailure) {
	char, err := h.userSvc.GetCharacterByID(input.CharacterID)
	if err != nil {
		return apicontract.TaskCommandMessage{}, taskBizFailure("角色不存在")
	}
	if char.UserID != userID {
		return apicontract.TaskCommandMessage{}, &taskFailure{response.CodeForbidden, "无权操作此角色"}
	}
	return h.runEsiRefreshTask(ctx, input)
}

func (h *ESIRefreshHandler) runEsiRefreshTaskByName(req apicontract.EsiRefreshRunByNameInput) (apicontract.TaskCommandMessage, *taskFailure) {
	queue := jobs.GetESIQueue()
	if queue == nil {
		return apicontract.TaskCommandMessage{}, taskBizFailure("刷新队列未初始化")
	}

	if ok := global.EnsureBackgroundTaskManager().Go("esi_run_task_by_name", func(ctx context.Context) error {
		return queue.RunTaskByName(ctx, req.TaskName)
	}); !ok {
		return apicontract.TaskCommandMessage{}, taskBizFailure("服务正在关闭，任务未启动")
	}
	return apicontract.TaskCommandMessage{Message: fmt.Sprintf("任务 %s 已触发（所有人物）", req.TaskName)}, nil
}

func (h *ESIRefreshHandler) runEsiRefreshAll() (apicontract.TaskCommandMessage, *taskFailure) {
	queue := jobs.GetESIQueue()
	if queue == nil {
		return apicontract.TaskCommandMessage{}, taskBizFailure("刷新队列未初始化")
	}

	if ok := global.EnsureBackgroundTaskManager().Go("esi_run_all", queue.Run); !ok {
		return apicontract.TaskCommandMessage{}, taskBizFailure("服务正在关闭，任务未启动")
	}
	return apicontract.TaskCommandMessage{Message: "全量刷新已触发"}, nil
}

func validateEsiRefreshIntervalTask(taskName string) *taskFailure {
	if _, ok := esi.GetTask(taskName); !ok {
		return &taskFailure{response.CodeParamError, "任务不存在: " + taskName}
	}
	return nil
}

func (h *ESIRefreshHandler) updateEsiRefreshTaskInterval(taskName string, req apicontract.EsiRefreshIntervalInput) (apicontract.TaskCommandMessage, *taskFailure) {
	if req.ActiveMinutes <= 0 || req.InactiveMinutes <= 0 {
		return apicontract.TaskCommandMessage{}, &taskFailure{response.CodeParamError, "间隔分钟数必须为正整数"}
	}

	// 读取当前覆盖配置
	var raw string
	if err := global.DB.Model(&model.SystemConfig{}).
		Where("key = ?", model.SysConfigESITaskIntervals).
		Pluck("value", &raw).Error; err != nil {
		return apicontract.TaskCommandMessage{}, taskBizFailure("读取配置失败: " + err.Error())
	}

	overrides := make(map[string]esi.RefreshIntervalMinutes)
	if strings.TrimSpace(raw) != "" {
		if err := json.Unmarshal([]byte(raw), &overrides); err != nil {
			overrides = make(map[string]esi.RefreshIntervalMinutes)
		}
	}

	overrides[taskName] = esi.RefreshIntervalMinutes{
		ActiveMinutes:   req.ActiveMinutes,
		InactiveMinutes: req.InactiveMinutes,
	}

	rawBytes, err := json.Marshal(overrides)
	if err != nil {
		return apicontract.TaskCommandMessage{}, taskBizFailure("序列化配置失败: " + err.Error())
	}
	rawJSON := string(rawBytes)

	// upsert: 先尝试 update，若无行则 insert
	res := global.DB.Model(&model.SystemConfig{}).
		Where("key = ?", model.SysConfigESITaskIntervals).
		Update("value", rawJSON)
	if res.Error != nil {
		return apicontract.TaskCommandMessage{}, taskBizFailure("保存配置失败: " + res.Error.Error())
	}
	if res.RowsAffected == 0 {
		if err := global.DB.Create(&model.SystemConfig{
			Key:   model.SysConfigESITaskIntervals,
			Value: rawJSON,
			Desc:  "ESI 子任务刷新间隔覆盖",
		}).Error; err != nil {
			return apicontract.TaskCommandMessage{}, taskBizFailure("创建配置失败: " + err.Error())
		}
	}

	// 同步内存缓存
	if err := esi.LoadIntervalOverrides(); err != nil {
		return apicontract.TaskCommandMessage{}, taskBizFailure("同步缓存失败: " + err.Error())
	}

	return apicontract.TaskCommandMessage{Message: "间隔已更新"}, nil
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

func ensureTaskPanel(panelByTask map[string]*apicontract.EsiRefreshMonitorTaskPanel, status *esi.TaskStatus) *apicontract.EsiRefreshMonitorTaskPanel {
	if panel, ok := panelByTask[status.TaskName]; ok {
		return panel
	}
	panel := &apicontract.EsiRefreshMonitorTaskPanel{
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
	interval := esi.ResolveInterval(status.TaskName)
	active := interval.Active
	if active > 0 {
		return int64(active.Seconds())
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

func taskHTTPRequestContext(ctx context.Context) context.Context {
	if c, ok := ctx.(*gin.Context); ok && c != nil && c.Request != nil {
		return c.Request.Context()
	}
	return ctx
}

func (h *ESIRefreshHandler) ListEsiRefreshTasks(_ context.Context, _ apicontract.ListEsiRefreshTasksRequestObject) (apicontract.ListEsiRefreshTasksResponseObject, error) {
	return apicontract.ListEsiRefreshTasks200JSONResponse{Code: response.CodeOK, Msg: "success", Data: h.listEsiRefreshTasks()}, nil
}

func (h *ESIRefreshHandler) ListEsiRefreshStatuses(_ context.Context, request apicontract.ListEsiRefreshStatusesRequestObject) (apicontract.ListEsiRefreshStatusesResponseObject, error) {
	out := &apicontract.ListEsiRefreshStatuses200JSONResponse{}
	data := h.listEsiRefreshStatuses(request.Params)
	return out, out.FromEsiRefreshStatusesSuccess(apicontract.EsiRefreshStatusesSuccess{Code: response.CodeOK, Msg: "success", Data: data})
}

func (h *ESIRefreshHandler) GetEsiRefreshMonitor(_ context.Context, _ apicontract.GetEsiRefreshMonitorRequestObject) (apicontract.GetEsiRefreshMonitorResponseObject, error) {
	return apicontract.GetEsiRefreshMonitor200JSONResponse{Code: response.CodeOK, Msg: "success", Data: h.getEsiRefreshMonitor()}, nil
}

func (h *ESIRefreshHandler) RunEsiRefreshTask(ctx context.Context, request apicontract.RunEsiRefreshTaskRequestObject) (apicontract.RunEsiRefreshTaskResponseObject, error) {
	out := &apicontract.RunEsiRefreshTask200JSONResponse{}
	if request.Body == nil {
		return out, out.FromBusinessError(taskMissingBody())
	}
	data, failure := h.runEsiRefreshTask(taskHTTPRequestContext(ctx), *request.Body)
	if failure != nil {
		return out, out.FromBusinessError(failure.businessError())
	}
	return out, out.FromTaskCommandSuccess(apicontract.TaskCommandSuccess{Code: response.CodeOK, Msg: "success", Data: data})
}

func (h *ESIRefreshHandler) RunEsiRefreshTaskByName(_ context.Context, request apicontract.RunEsiRefreshTaskByNameRequestObject) (apicontract.RunEsiRefreshTaskByNameResponseObject, error) {
	out := &apicontract.RunEsiRefreshTaskByName200JSONResponse{}
	if request.Body == nil {
		return out, out.FromBusinessError(taskMissingBody())
	}
	data, failure := h.runEsiRefreshTaskByName(*request.Body)
	if failure != nil {
		return out, out.FromBusinessError(failure.businessError())
	}
	return out, out.FromTaskCommandSuccess(apicontract.TaskCommandSuccess{Code: response.CodeOK, Msg: "success", Data: data})
}

func (h *ESIRefreshHandler) RunEsiRefreshAll(_ context.Context, _ apicontract.RunEsiRefreshAllRequestObject) (apicontract.RunEsiRefreshAllResponseObject, error) {
	out := &apicontract.RunEsiRefreshAll200JSONResponse{}
	data, failure := h.runEsiRefreshAll()
	if failure != nil {
		return out, out.FromBusinessError(failure.businessError())
	}
	return out, out.FromTaskCommandSuccess(apicontract.TaskCommandSuccess{Code: response.CodeOK, Msg: "success", Data: data})
}

func (h *ESIRefreshHandler) UpdateEsiRefreshTaskInterval(_ context.Context, request apicontract.UpdateEsiRefreshTaskIntervalRequestObject) (apicontract.UpdateEsiRefreshTaskIntervalResponseObject, error) {
	out := &apicontract.UpdateEsiRefreshTaskInterval200JSONResponse{}
	if failure := validateEsiRefreshIntervalTask(request.Name); failure != nil {
		return out, out.FromBusinessError(failure.businessError())
	}
	if request.Body == nil {
		return out, out.FromBusinessError(taskMissingBody())
	}
	data, failure := h.updateEsiRefreshTaskInterval(request.Name, *request.Body)
	if failure != nil {
		return out, out.FromBusinessError(failure.businessError())
	}
	return out, out.FromTaskCommandSuccess(apicontract.TaskCommandSuccess{Code: response.CodeOK, Msg: "success", Data: data})
}

func (h *ESIRefreshHandler) RunMyCharacterTask(ctx context.Context, request apicontract.RunMyCharacterTaskRequestObject) (apicontract.RunMyCharacterTaskResponseObject, error) {
	actor, ok := taskStrictActor(ctx)
	if !ok {
		return apicontract.RunMyCharacterTask401JSONResponse{UnauthorizedJSONResponse: taskUnauthorized()}, nil
	}
	out := &apicontract.RunMyCharacterTask200JSONResponse{}
	if request.Body == nil {
		return out, out.FromBusinessError(taskMissingBody())
	}
	data, failure := h.runMyCharacterTask(taskHTTPRequestContext(ctx), actor, *request.Body)
	if failure != nil {
		if failure.code == response.CodeForbidden {
			return apicontract.RunMyCharacterTask403JSONResponse{ForbiddenJSONResponse: apicontract.ForbiddenJSONResponse{
				Code: response.CodeForbidden, Msg: failure.message,
			}}, nil
		}
		return out, out.FromBusinessError(failure.businessError())
	}
	return out, out.FromTaskCommandSuccess(apicontract.TaskCommandSuccess{Code: response.CodeOK, Msg: "success", Data: data})
}

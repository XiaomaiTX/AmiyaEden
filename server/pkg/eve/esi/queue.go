package esi

import (
	"amiya-eden/global"
	"amiya-eden/internal/model"
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"go.uber.org/zap"
)

// ─────────────────────────────────────────────
//  队列引擎
// ─────────────────────────────────────────────

type TokenService interface {
	GetValidToken(ctx context.Context, characterID int64) (string, error)
}

type CharacterRepository interface {
	ListAllWithToken() ([]model.EveCharacter, error)
	GetByCharacterID(characterID int64) (*model.EveCharacter, error)
}

type Queue struct {
	client   *Client
	ssoSvc   TokenService
	charRepo CharacterRepository

	mu         sync.RWMutex
	statuses   map[string]*TaskStatus // key: "taskName:characterID"
	runLocks   map[string]*sync.Mutex
	runLocksMu sync.Mutex

	// 并发控制：同一时间最多执行的任务数
	concurrency atomic.Int32
}

func NewQueue(ssoSvc TokenService, charRepo CharacterRepository) *Queue {
	queue := &Queue{
		client:   NewClient(),
		ssoSvc:   ssoSvc,
		charRepo: charRepo,
		statuses: make(map[string]*TaskStatus),
		runLocks: make(map[string]*sync.Mutex),
	}
	queue.concurrency.Store(5)
	return queue
}

// SetConcurrency 设置最大并发数
func (q *Queue) SetConcurrency(n int) {
	if n < 1 {
		n = 1
	}
	q.concurrency.Store(int32(n))
}

// ─────────────────────────────────────────────
//  调度入口
// ─────────────────────────────────────────────

// Run 执行一次完整的刷新调度
// 由 cron 定时触发
func (q *Queue) Run(ctx context.Context) error {
	ctx = normalizeQueueContext(ctx)
	queueZapLogger().Info("[ESI Queue] 开始刷新调度")
	if err := ctx.Err(); err != nil {
		return err
	}

	// Tests and early-startup code can invoke the queue before the shared DB
	// has been configured. Exit cleanly instead of panicking in the repository.
	if global.DB == nil {
		queueZapLogger().Warn("[ESI Queue] 数据库未初始化，跳过本次刷新调度")
		return nil
	}

	// 1. 获取所有有 refresh_token 的人物
	characters, err := q.charRepo.ListAllWithToken()
	if err != nil {
		queueZapLogger().Error("[ESI Queue] 获取人物列表失败", zap.Error(err))
		return fmt.Errorf("list characters: %w", err)
	}

	if len(characters) == 0 {
		queueZapLogger().Info("[ESI Queue] 没有需要刷新的人物")
		return nil
	}

	// 2. 检测人物活跃度
	activityMap := q.checkActivity(ctx, characters)
	authorizedProviders, err := buildAuthorizedCorpKillmailProviders(characters)
	if err != nil {
		queueZapLogger().Warn("[ESI Queue] 构建军团 KM 覆盖集失败", zap.Error(err))
		authorizedProviders = map[int64]int64{}
	}
	authorizedCorps := make(map[int64]bool)
	corpProviderIDs := make(map[int64][]int64)
	corpHasActiveProvider := make(map[int64]bool)
	for characterID, corporationID := range authorizedProviders {
		authorizedCorps[corporationID] = true
		corpProviderIDs[corporationID] = append(corpProviderIDs[corporationID], characterID)
		if activityMap[characterID] {
			corpHasActiveProvider[corporationID] = true
		}
	}
	corpCoverage := make(map[int64]bool)
	for corporationID := range authorizedCorps {
		if q.corporationKillmailsFresh(
			corporationID,
			corpProviderIDs[corporationID],
			corpHasActiveProvider[corporationID],
		) {
			corpCoverage[corporationID] = true
		}
	}

	// 3. 获取所有任务并按优先级排序
	allTasks := AllTasks()
	sortedTasks := sortTasksByPriority(allTasks)

	// 4. 构建待执行任务列表
	type pendingJob struct {
		task      RefreshTask
		character model.EveCharacter
		isActive  bool
	}
	var jobs []pendingJob
	seenExecutionKeys := make(map[string]struct{})

	for _, task := range sortedTasks {
		for i := range characters {
			char := characters[i]
			isActive := activityMap[char.CharacterID]

			// 检查人物是否有该任务所需的 scope
			if !q.hasRequiredScopes(char, task) {
				continue
			}
			if task.Name() == "corporation_killmails" {
				if _, ok := authorizedProviders[char.CharacterID]; !ok {
					continue
				}
			}

			if q.shouldSkipAutomaticTask(char, task, corpCoverage) {
				continue
			}

			// 检查是否需要刷新（基于上次执行时间和刷新间隔）
			if !q.needsRefresh(task, char, isActive) {
				continue
			}
			executionKey := q.taskExecutionKey(task, char)
			if _, exists := seenExecutionKeys[executionKey]; exists {
				continue
			}
			seenExecutionKeys[executionKey] = struct{}{}

			jobs = append(jobs, pendingJob{
				task:      task,
				character: char,
				isActive:  isActive,
			})
		}
	}

	if len(jobs) == 0 {
		queueZapLogger().Info("[ESI Queue] 没有需要执行的任务")
		return nil
	}

	queueZapLogger().Info("[ESI Queue] 开始执行刷新任务",
		zap.Int("total_jobs", len(jobs)),
		zap.Int("characters", len(characters)),
	)

	// 5. 使用信号量控制并发执行
	sem := make(chan struct{}, q.concurrencyLimit())
	var wg sync.WaitGroup
	var firstErr error
	var firstErrMu sync.Mutex
	recordErr := func(err error) {
		if err == nil || errors.Is(err, ErrTaskSkipped) || errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			return
		}
		firstErrMu.Lock()
		defer firstErrMu.Unlock()
		if firstErr == nil {
			firstErr = err
		}
	}

	for _, job := range jobs {
		if err := ctx.Err(); err != nil {
			recordErr(err)
			break
		}
		wg.Add(1)
		sem <- struct{}{} // 占位

		go func(j pendingJob) {
			defer wg.Done()
			defer func() { <-sem }() // 释放

			recordErr(q.executeTask(ctx, j.task, j.character, j.isActive, false))
		}(job)
	}

	wg.Wait()
	queueZapLogger().Info("[ESI Queue] 刷新调度完成")
	if firstErr != nil {
		return firstErr
	}
	return ctx.Err()
}

// RunTask 手动执行某个指定任务（管理页面触发）
func (q *Queue) RunTask(ctx context.Context, taskName string, characterID int64) error {
	ctx = normalizeQueueContext(ctx)
	task, ok := GetTask(taskName)
	if !ok {
		return fmt.Errorf("task %q not found", taskName)
	}

	char, err := q.charRepo.GetByCharacterID(characterID)
	if err != nil {
		return fmt.Errorf("character %d not found: %w", characterID, err)
	}

	isActive := q.checkSingleActivity(ctx, *char)

	return q.executeTask(ctx, task, *char, isActive, true)
}

// RunAllForCharacter 对指定人物执行全部任务，忽略刷新间隔（用于新人物首次登录时的全量初始化）
func (q *Queue) RunAllForCharacter(ctx context.Context, characterID int64) error {
	ctx = normalizeQueueContext(ctx)
	char, err := q.charRepo.GetByCharacterID(characterID)
	if err != nil {
		queueZapLogger().Error("[ESI Queue] RunAllForCharacter: 人物不存在",
			zap.Int64("character_id", characterID),
			zap.Error(err),
		)
		return fmt.Errorf("character %d not found: %w", characterID, err)
	}

	isActive := q.checkSingleActivity(ctx, *char)
	allTasks := AllTasks()
	sortedTasks := sortTasksByPriority(allTasks)

	sem := make(chan struct{}, q.concurrencyLimit())
	var wg sync.WaitGroup
	var firstErr error
	var firstErrMu sync.Mutex
	recordErr := func(err error) {
		if err == nil || errors.Is(err, ErrTaskSkipped) || errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			return
		}
		firstErrMu.Lock()
		defer firstErrMu.Unlock()
		if firstErr == nil {
			firstErr = err
		}
	}

	for _, task := range sortedTasks {
		if err := ctx.Err(); err != nil {
			recordErr(err)
			break
		}
		if !q.hasRequiredScopes(*char, task) {
			continue
		}
		wg.Add(1)
		sem <- struct{}{}
		go func(t RefreshTask) {
			defer wg.Done()
			defer func() { <-sem }()
			recordErr(q.executeTask(ctx, t, *char, isActive, true))
		}(task)
	}

	wg.Wait()
	queueZapLogger().Info("[ESI Queue] 新人物全量刷新完成", zap.Int64("character_id", characterID))
	if firstErr != nil {
		return firstErr
	}
	return ctx.Err()
}

// RunTaskByName 对所有拥有所需 scope 的人物执行指定任务
func (q *Queue) RunTaskByName(ctx context.Context, taskName string) error {
	ctx = normalizeQueueContext(ctx)
	task, ok := GetTask(taskName)
	if !ok {
		return fmt.Errorf("task %q not found", taskName)
	}

	characters, err := q.charRepo.ListAllWithToken()
	if err != nil {
		return fmt.Errorf("list characters: %w", err)
	}

	activityMap := q.checkActivity(ctx, characters)
	authorizedProviders := map[int64]int64{}
	if task.Name() == "corporation_killmails" {
		var buildErr error
		authorizedProviders, buildErr = buildAuthorizedCorpKillmailProviders(characters)
		if buildErr != nil {
			return fmt.Errorf("build corporation killmail providers: %w", buildErr)
		}
	}

	sem := make(chan struct{}, q.concurrencyLimit())
	var wg sync.WaitGroup
	seenKeys := make(map[string]struct{})
	var firstErr error
	var firstErrMu sync.Mutex
	recordErr := func(err error) {
		if err == nil || errors.Is(err, ErrTaskSkipped) || errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			return
		}
		firstErrMu.Lock()
		defer firstErrMu.Unlock()
		if firstErr == nil {
			firstErr = err
		}
	}

	for i := range characters {
		if err := ctx.Err(); err != nil {
			recordErr(err)
			break
		}
		char := characters[i]
		if !q.hasRequiredScopes(char, task) {
			continue
		}
		if task.Name() == "corporation_killmails" {
			if _, ok := authorizedProviders[char.CharacterID]; !ok {
				continue
			}
		}
		jobKey := q.taskExecutionKey(task, char)
		if _, ok := seenKeys[jobKey]; ok {
			continue
		}
		seenKeys[jobKey] = struct{}{}
		isActive := activityMap[char.CharacterID]

		wg.Add(1)
		sem <- struct{}{}
		go func(ch model.EveCharacter, active bool) {
			defer wg.Done()
			defer func() { <-sem }()
			recordErr(q.executeTask(ctx, task, ch, active, true))
		}(char, isActive)
	}

	wg.Wait()
	if firstErr != nil {
		return firstErr
	}
	return ctx.Err()
}

// ─────────────────────────────────────────────
//  内部方法
// ─────────────────────────────────────────────

// executeTask 执行单个任务
func (q *Queue) executeTask(ctx context.Context, task RefreshTask, char model.EveCharacter, isActive bool, force bool) error {
	ctx = normalizeQueueContext(ctx)
	statusKey := fmt.Sprintf("%s:%d", task.Name(), char.CharacterID)
	executionKey := q.taskExecutionKey(task, char)
	lock := q.executionLock(executionKey)
	lock.Lock()
	defer lock.Unlock()
	if err := ctx.Err(); err != nil {
		q.setStatus(statusKey, &TaskStatus{
			TaskName:    task.Name(),
			Description: task.Description(),
			CharacterID: char.CharacterID,
			Priority:    task.Priority(),
			Status:      "failed",
			Error:       err.Error(),
		})
		return err
	}

	if !force && !q.needsRefresh(task, char, isActive) {
		lastRun, err := q.getLastRun(task, char)
		if err == nil {
			interval := task.Interval()
			nextDur := interval.Active
			if !isActive {
				nextDur = interval.Inactive
			}
			nextRun := lastRun.Add(nextDur)
			q.setStatus(statusKey, &TaskStatus{
				TaskName:    task.Name(),
				Description: task.Description(),
				CharacterID: char.CharacterID,
				Priority:    task.Priority(),
				LastRun:     &lastRun,
				NextRun:     &nextRun,
				Status:      "success",
			})
		}
		return nil
	}

	// 更新状态为 running
	q.setStatus(statusKey, &TaskStatus{
		TaskName:    task.Name(),
		Description: task.Description(),
		CharacterID: char.CharacterID,
		Priority:    task.Priority(),
		Status:      "running",
	})

	// 获取有效 Token
	accessToken, err := q.ssoSvc.GetValidToken(ctx, char.CharacterID)
	if err != nil {
		queueZapLogger().Error("[ESI Queue] 获取 Token 失败",
			zap.String("task", task.Name()),
			zap.Int64("character_id", char.CharacterID),
			zap.Error(err),
		)
		q.setStatus(statusKey, &TaskStatus{
			TaskName:    task.Name(),
			Description: task.Description(),
			CharacterID: char.CharacterID,
			Priority:    task.Priority(),
			Status:      "failed",
			Error:       err.Error(),
		})
		return err
	}

	// 执行任务
	taskCtx := &TaskContext{
		Context:     ctx,
		CharacterID: char.CharacterID,
		AccessToken: accessToken,
		Client:      q.client,
		IsActive:    isActive,
	}

	if err := task.Execute(taskCtx); err != nil {
		if errors.Is(err, ErrTaskSkipped) {
			q.setStatus(statusKey, &TaskStatus{
				TaskName:    task.Name(),
				Description: task.Description(),
				CharacterID: char.CharacterID,
				Priority:    task.Priority(),
				Status:      "skipped",
			})
			return err
		}
		queueZapLogger().Error("[ESI Queue] 任务执行失败",
			zap.String("task", task.Name()),
			zap.Int64("character_id", char.CharacterID),
			zap.Error(err),
		)
		q.setStatus(statusKey, &TaskStatus{
			TaskName:    task.Name(),
			Description: task.Description(),
			CharacterID: char.CharacterID,
			Priority:    task.Priority(),
			Status:      "failed",
			Error:       err.Error(),
		})
		return err
	}

	// 成功：记录上次执行时间
	now := time.Now()
	interval := task.Interval()
	nextDur := interval.Active
	if !isActive {
		nextDur = interval.Inactive
	}
	nextRun := now.Add(nextDur)

	q.setStatus(statusKey, &TaskStatus{
		TaskName:    task.Name(),
		Description: task.Description(),
		CharacterID: char.CharacterID,
		Priority:    task.Priority(),
		LastRun:     &now,
		NextRun:     &nextRun,
		Status:      "success",
	})

	// 将上次执行时间持久化到 Redis
	q.setLastRun(task, char, now)

	queueZapLogger().Debug("[ESI Queue] 任务执行成功",
		zap.String("task", task.Name()),
		zap.Int64("character_id", char.CharacterID),
	)
	return nil
}

func normalizeQueueContext(ctx context.Context) context.Context {
	if ctx == nil {
		return context.Background()
	}
	return ctx
}

func (q *Queue) concurrencyLimit() int {
	limit := int(q.concurrency.Load())
	if limit < 1 {
		return 1
	}
	return limit
}

// needsRefresh 判断任务是否需要刷新
func (q *Queue) needsRefresh(task RefreshTask, char model.EveCharacter, isActive bool) bool {
	lastRun, err := q.getLastRun(task, char)
	if err != nil {
		return true // 没有记录则需要刷新
	}

	interval := task.Interval()
	dur := interval.Active
	if !isActive {
		dur = interval.Inactive
	}

	return time.Since(lastRun) >= dur
}

// hasRequiredScopes 检查人物是否拥有任务所需的 scope
func (q *Queue) hasRequiredScopes(char model.EveCharacter, task RefreshTask) bool {
	charScopes := strings.Fields(char.Scopes)
	scopeSet := make(map[string]struct{}, len(charScopes))
	for _, s := range charScopes {
		scopeSet[s] = struct{}{}
	}

	for _, required := range task.RequiredScopes() {
		if _, ok := scopeSet[required.Scope]; !ok {
			return false
		}
	}
	return true
}

func (q *Queue) shouldSkipAutomaticTask(char model.EveCharacter, task RefreshTask, corpCoverage map[int64]bool) bool {
	if task.Name() != "character_killmails" || char.CorporationID == 0 {
		return false
	}
	return corpCoverage[char.CorporationID]
}

func queueZapLogger() *zap.Logger {
	if global.Logger != nil {
		return global.Logger
	}
	return zap.NewNop()
}

func (q *Queue) taskExecutionKey(task RefreshTask, char model.EveCharacter) string {
	if (task.Name() == "corporation_killmails" || task.Name() == "eve_structures") && char.CorporationID != 0 {
		return fmt.Sprintf("%s:corp:%d", task.Name(), char.CorporationID)
	}
	return fmt.Sprintf("%s:char:%d", task.Name(), char.CharacterID)
}

func buildAuthorizedCorpKillmailProviders(characters []model.EveCharacter) (map[int64]int64, error) {
	providers := make(map[int64]int64)
	candidates := make(map[int64]int64)
	for i := range characters {
		char := characters[i]
		if char.CorporationID == 0 {
			continue
		}
		for _, scope := range strings.Fields(char.Scopes) {
			if scope == "esi-killmails.read_corporation_killmails.v1" {
				candidates[char.CharacterID] = char.CorporationID
				break
			}
		}
	}
	if len(candidates) == 0 {
		return providers, nil
	}

	charIDs := make([]int64, 0, len(candidates))
	for characterID := range candidates {
		charIDs = append(charIDs, characterID)
	}

	var directorIDs []int64
	if err := global.DB.Model(&model.EveCharacterCorpRole{}).
		Where("character_id IN ? AND corp_role = ?", charIDs, "Director").
		Pluck("character_id", &directorIDs).Error; err != nil {
		return nil, err
	}
	for _, characterID := range directorIDs {
		providers[characterID] = candidates[characterID]
	}
	return providers, nil
}

func (q *Queue) corporationKillmailsFresh(corporationID int64, providerCharacterIDs []int64, isActive bool) bool {
	if corporationID == 0 {
		return false
	}
	corpChar := model.EveCharacter{CorporationID: corporationID}
	if !q.needsRefresh(&CorpKillmailsTask{}, corpChar, isActive) {
		return true
	}

	legacyRun, ok := q.getFreshLegacyCorpLastRun(providerCharacterIDs, isActive)
	if !ok {
		return false
	}
	q.setLastRun(&CorpKillmailsTask{}, corpChar, legacyRun)
	return true
}

func (q *Queue) executionLock(key string) *sync.Mutex {
	q.runLocksMu.Lock()
	defer q.runLocksMu.Unlock()
	if lock, ok := q.runLocks[key]; ok {
		return lock
	}
	lock := &sync.Mutex{}
	q.runLocks[key] = lock
	return lock
}

// ─────────────────────────────────────────────
//  Redis 状态存储
// ─────────────────────────────────────────────

const (
	lastRunKeyPrefix = "esi:refresh:lastrun:" // esi:refresh:lastrun:{task}:char:{characterID} or {task}:corp:{corporationID}
)

func (q *Queue) setLastRun(task RefreshTask, char model.EveCharacter, t time.Time) {
	if global.Redis == nil {
		return
	}
	key := fmt.Sprintf("%s%s", lastRunKeyPrefix, q.taskExecutionKey(task, char))
	global.Redis.Set(context.Background(), key, t.Unix(), 0)
}

func (q *Queue) getLastRun(task RefreshTask, char model.EveCharacter) (time.Time, error) {
	if global.Redis == nil {
		return time.Time{}, errors.New("redis is not initialized")
	}
	key := fmt.Sprintf("%s%s", lastRunKeyPrefix, q.taskExecutionKey(task, char))
	val, err := global.Redis.Get(context.Background(), key).Int64()
	if err == nil {
		return time.Unix(val, 0), nil
	}

	legacyKey, ok := q.legacyLastRunKey(task, char)
	if !ok {
		return time.Time{}, err
	}

	legacyVal, legacyErr := global.Redis.Get(context.Background(), legacyKey).Int64()
	if legacyErr != nil {
		return time.Time{}, err
	}

	lastRun := time.Unix(legacyVal, 0)
	q.setLastRun(task, char, lastRun)
	return lastRun, nil
}

func (q *Queue) legacyLastRunKey(task RefreshTask, char model.EveCharacter) (string, bool) {
	if char.CharacterID == 0 {
		return "", false
	}
	return fmt.Sprintf("%s%s:%d", lastRunKeyPrefix, task.Name(), char.CharacterID), true
}

func (q *Queue) getFreshLegacyCorpLastRun(providerCharacterIDs []int64, isActive bool) (time.Time, bool) {
	if len(providerCharacterIDs) == 0 {
		return time.Time{}, false
	}
	if global.Redis == nil {
		return time.Time{}, false
	}

	interval := (&CorpKillmailsTask{}).Interval()
	dur := interval.Active
	if !isActive {
		dur = interval.Inactive
	}

	var freshest time.Time
	for _, characterID := range providerCharacterIDs {
		legacyKey := fmt.Sprintf("%s%s:%d", lastRunKeyPrefix, (&CorpKillmailsTask{}).Name(), characterID)
		legacyVal, err := global.Redis.Get(context.Background(), legacyKey).Int64()
		if err != nil {
			continue
		}
		legacyRun := time.Unix(legacyVal, 0)
		if time.Since(legacyRun) >= dur {
			continue
		}
		if freshest.IsZero() || legacyRun.After(freshest) {
			freshest = legacyRun
		}
	}

	if freshest.IsZero() {
		return time.Time{}, false
	}
	return freshest, true
}

// ─────────────────────────────────────────────
//  状态管理（可视化用）
// ─────────────────────────────────────────────

func (q *Queue) setStatus(key string, status *TaskStatus) {
	q.mu.Lock()
	defer q.mu.Unlock()
	q.statuses[key] = status
}

// SetStatusesForTest 替换整个 statuses 映射（仅供测试使用）
func (q *Queue) SetStatusesForTest(statuses map[string]*TaskStatus) {
	q.mu.Lock()
	defer q.mu.Unlock()
	q.statuses = statuses
}

// GetAllStatuses 获取所有任务状态（用于 API 展示）
func (q *Queue) GetAllStatuses() []*TaskStatus {
	q.mu.RLock()
	defer q.mu.RUnlock()

	result := make([]*TaskStatus, 0, len(q.statuses))
	for _, s := range q.statuses {
		result = append(result, s)
	}

	// 按优先级排序
	sort.Slice(result, func(i, j int) bool {
		if result[i].Priority != result[j].Priority {
			return result[i].Priority < result[j].Priority
		}
		return result[i].TaskName < result[j].TaskName
	})
	return result
}

// GetTaskStatuses 获取指定任务的所有人物状态
func (q *Queue) GetTaskStatuses(taskName string) []*TaskStatus {
	q.mu.RLock()
	defer q.mu.RUnlock()

	var result []*TaskStatus
	prefix := taskName + ":"
	for key, s := range q.statuses {
		if strings.HasPrefix(key, prefix) {
			result = append(result, s)
		}
	}
	return result
}

// ─────────────────────────────────────────────
//  辅助
// ─────────────────────────────────────────────

// sortTasksByPriority 按优先级排序任务
func sortTasksByPriority(tasks map[string]RefreshTask) []RefreshTask {
	sorted := make([]RefreshTask, 0, len(tasks))
	for _, t := range tasks {
		sorted = append(sorted, t)
	}
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].Priority() < sorted[j].Priority()
	})
	return sorted
}

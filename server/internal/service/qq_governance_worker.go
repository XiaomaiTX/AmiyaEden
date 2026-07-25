package service

import (
	"amiya-eden/global"
	"amiya-eden/internal/model"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"gorm.io/gorm"
)

// StartWorker 启动参与服务关停的动作 worker；所有待执行状态留在数据库中。
func (s *QQGovernanceService) StartWorker() {
	if s == nil {
		return
	}
	s.workerOnce.Do(func() {
		global.EnsureBackgroundTaskManager().Go("qq_governance_action_worker", func(ctx context.Context) error {
			ticker := time.NewTicker(500 * time.Millisecond)
			defer ticker.Stop()
			for {
				if err := s.RunActionWorkerOnce(ctx); err != nil && !errors.Is(err, context.Canceled) {
					s.logger().Warn("QQ 群治理动作 worker 本轮失败")
				}
				select {
				case <-ctx.Done():
					return ctx.Err()
				case <-ticker.C:
				}
			}
		})
	})
}

// RunActionWorkerOnce 是可测试的单次领取与执行入口。
func (s *QQGovernanceService) RunActionWorkerOnce(ctx context.Context) error {
	token, err := newQQGovernanceLeaseToken()
	if err != nil {
		return err
	}
	task, err := s.repo.ClaimNextActionTask(s.now(), token, qqGovernanceLease)
	if err != nil || task == nil {
		return err
	}
	if task.ActionType == model.QQGovernanceActionSnapshot || task.ActionType == model.QQGovernanceActionRefreshGroupInfo || task.ActionType == model.QQGovernanceActionComputeBatch || task.ActionType == model.QQGovernanceActionRecheck {
		return s.runReconcileTask(ctx, task)
	}
	if wait, err := s.riskWait(task); err != nil {
		return s.failTask(task, err, true)
	} else if wait > 0 {
		return s.repo.RetryOrDeadActionTask(task.ID, task.LeaseToken, task.RetryCount, s.now().Add(wait), "OneBot 风险控制暂停自动动作", false)
	}
	if valid, reason, err := s.actionStillValid(task); err != nil {
		return s.failTask(task, err, true)
	} else if !valid {
		return s.repo.CancelActionTask(task.ID, task.LeaseToken, reason)
	}
	if wait, err := s.acquireQQGovernanceRateLimit(ctx, task); err != nil {
		return s.failTask(task, err, true)
	} else if wait > 0 {
		return s.repo.RetryOrDeadActionTask(task.ID, task.LeaseToken, task.RetryCount, s.now().Add(wait), "QQ 动作限流等待", false)
	}
	params, err := paramsForQQGovernanceAction(task)
	if err != nil {
		return s.failTask(task, err, false)
	}
	executor := s.actionExecutor()
	if executor == nil || !executor.OneBotConnected() {
		return s.failTask(task, &OneBotActionError{Message: "OneBot 机器人未连接", Retryable: true}, true)
	}
	actionCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	_, err = executor.CallOneBot(actionCtx, oneBotActionName(task.ActionType), params)
	if err != nil {
		_ = s.logQQGovernanceAction(task, "failed", err.Error())
		s.recordRiskOutcome(task, false, err)
		return s.failTask(task, err, isRetryableQQGovernanceError(err))
	}
	if err = s.logQQGovernanceAction(task, "succeeded", ""); err != nil {
		return err
	}
	s.recordRiskOutcome(task, true, nil)
	return s.repo.CompleteActionTask(task.ID, task.LeaseToken, s.now())
}

func (s *QQGovernanceService) actionStillValid(task *model.QQGovernanceActionTask) (bool, string, error) {
	if _, err := s.repo.GetEnabledPolicy(task.GroupID); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return false, "群治理规则已禁用或删除", nil
		}
		return false, "", fmt.Errorf("读取群治理规则失败: %w", err)
	}
	state, err := s.repo.GetMemberState(task.GroupID, task.QQ)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return false, "成员运行态不存在", nil
		}
		return false, "", fmt.Errorf("读取成员运行态失败: %w", err)
	}
	if state.Version != task.TargetVersion {
		return false, "成员状态已更新，任务版本过期", nil
	}
	if task.ActionType == model.QQGovernanceActionSetCard && state.Status != model.QQGovernanceMemberValid {
		return false, "成员不再处于有效在群状态", nil
	}
	if task.ActionType == model.QQGovernanceActionKick && state.Status != model.QQGovernanceMemberInvalidConf {
		return false, "成员尚未达到自动清退确认状态", nil
	}
	return true, "", nil
}
func (s *QQGovernanceService) failTask(task *model.QQGovernanceActionTask, err error, retryable bool) error {
	count := task.RetryCount + 1
	dead := !retryable || count >= qqGovernanceMaxRetries
	if dead {
		_, _ = s.createAlert("dead_task", task.GroupID, task.QQ, task.ID, "QQ 群治理动作任务已进入死信")
	}
	return s.repo.RetryOrDeadActionTask(task.ID, task.LeaseToken, count, s.now().Add(qqGovernanceRetryDelay(count)), truncateQQGovernanceError(err), dead)
}
func qqGovernanceRetryDelay(count int) time.Duration {
	switch count {
	case 1:
		return 45 * time.Second
	case 2:
		return 3 * time.Minute
	case 3:
		return 15 * time.Minute
	default:
		return 45 * time.Minute
	}
}
func isRetryableQQGovernanceError(err error) bool {
	var oneBotErr *OneBotActionError
	if errors.As(err, &oneBotErr) {
		return oneBotErr.Retryable
	}
	return true
}
func truncateQQGovernanceError(err error) string {
	if err == nil {
		return ""
	}
	v := []rune(err.Error())
	if len(v) > 1000 {
		return string(v[:1000])
	}
	return string(v)
}

func (s *QQGovernanceService) logQQGovernanceAction(task *model.QQGovernanceActionTask, result, errMessage string) error {
	return s.repo.CreateActionLog(&model.QQGovernanceActionLog{
		TaskID:         task.ID,
		ActionType:     task.ActionType,
		RequestSummary: fmt.Sprintf("action=%s group_id=%d qq=%d version=%d", task.ActionType, task.GroupID, task.QQ, task.TargetVersion),
		Result:         result,
		ErrorMessage:   errMessage,
		Attempt:        task.RetryCount + 1,
	})
}
func paramsForQQGovernanceAction(task *model.QQGovernanceActionTask) (map[string]any, error) {
	var payload qqGovernanceActionPayload
	if err := json.Unmarshal([]byte(task.PayloadJSON), &payload); err != nil {
		return nil, err
	}
	switch task.ActionType {
	case model.QQGovernanceActionApprove, model.QQGovernanceActionReject:
		if strings.TrimSpace(payload.RequestFlag) == "" {
			return nil, errors.New("入群申请动作缺少 request flag")
		}
		return map[string]any{"flag": payload.RequestFlag, "sub_type": "add", "approve": payload.Approve}, nil
	case model.QQGovernanceActionSetCard:
		if strings.TrimSpace(payload.Card) == "" {
			return nil, errors.New("群名片动作缺少目标名片")
		}
		return map[string]any{"group_id": task.GroupID, "user_id": task.QQ, "card": payload.Card}, nil
	case model.QQGovernanceActionKick:
		return map[string]any{"group_id": task.GroupID, "user_id": task.QQ, "reject_add_request": false}, nil
	default:
		return nil, fmt.Errorf("未知 QQ 治理动作: %s", task.ActionType)
	}
}
func oneBotActionName(kind string) string {
	if kind == model.QQGovernanceActionApprove || kind == model.QQGovernanceActionReject {
		return "set_group_add_request"
	}
	if kind == model.QQGovernanceActionSetCard {
		return "set_group_card"
	}
	if kind == model.QQGovernanceActionKick {
		return "set_group_kick"
	}
	return kind
}

// 所有 QQ 操作（OneBot 读取和写入）先经过同一个 Redis 限流器。Redis 不可用时进入重试，不绕过限流直连 QQ。
func (s *QQGovernanceService) acquireQQGovernanceRateLimit(ctx context.Context, task *model.QQGovernanceActionTask) (time.Duration, error) {
	if global.Redis == nil {
		return 0, errors.New("redis 不可用，拒绝执行 QQ 操作")
	}
	groupInterval := 8 * time.Second
	if task.ActionType == model.QQGovernanceActionKick {
		groupInterval = 30 * time.Second
	}
	const tokenBucketScript = `
local now = tonumber(ARGV[1])
local max_wait = 0
for i = 1, #KEYS do
  local arg = 2 + (i - 1) * 2
  local capacity = tonumber(ARGV[arg])
  local refill = tonumber(ARGV[arg + 1])
  local tokens = tonumber(redis.call('HGET', KEYS[i], 'tokens')) or capacity
  local last = tonumber(redis.call('HGET', KEYS[i], 'last')) or now
  tokens = math.min(capacity, tokens + math.max(0, now - last) * refill)
  if tokens < 1 then
    local wait = math.ceil((1 - tokens) / refill)
    if wait > max_wait then max_wait = wait end
  end
end
if max_wait > 0 then return {0, max_wait} end
for i = 1, #KEYS do
  local arg = 2 + (i - 1) * 2
  local capacity = tonumber(ARGV[arg])
  local refill = tonumber(ARGV[arg + 1])
  local tokens = tonumber(redis.call('HGET', KEYS[i], 'tokens')) or capacity
  local last = tonumber(redis.call('HGET', KEYS[i], 'last')) or now
  tokens = math.min(capacity, tokens + math.max(0, now - last) * refill) - 1
  redis.call('HMSET', KEYS[i], 'tokens', tokens, 'last', now)
  redis.call('PEXPIRE', KEYS[i], math.ceil(capacity / refill))
end
return {1, 0}`
	keys := []string{"qq_governance:rate:global", fmt.Sprintf("qq_governance:rate:group:%d", task.GroupID)}
	args := []any{s.now().UnixMilli(), 3, 1.0 / 3000.0, 1, 1.0 / float64(groupInterval.Milliseconds())}
	if task.QQ > 0 {
		keys = append(keys, fmt.Sprintf("qq_governance:rate:qq:%d", task.QQ))
		args = append(args, 1, 1.0/60000.0)
	}
	result, err := global.Redis.Eval(ctx, tokenBucketScript, keys, args...).Result()
	if err != nil {
		return 0, err
	}
	values, ok := result.([]interface{})
	if !ok || len(values) != 2 {
		return 0, errors.New("QQ 限流脚本返回无效结果")
	}
	allowed, ok := qqGovernanceInt64(values[0])
	if !ok {
		return 0, errors.New("QQ 限流脚本状态无效")
	}
	if allowed == 1 {
		return 0, nil
	}
	waitMS, ok := qqGovernanceInt64(values[1])
	if !ok || waitMS < 1 {
		waitMS = 1000
	}
	return time.Duration(waitMS) * time.Millisecond, nil
}

func qqGovernanceInt64(value any) (int64, bool) {
	switch v := value.(type) {
	case int64:
		return v, true
	case int:
		return int64(v), true
	case string:
		var out int64
		_, err := fmt.Sscan(v, &out)
		return out, err == nil
	case []byte:
		var out int64
		_, err := fmt.Sscan(string(v), &out)
		return out, err == nil
	default:
		return 0, false
	}
}

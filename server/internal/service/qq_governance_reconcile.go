package service

import (
	"amiya-eden/internal/model"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"math/rand/v2"
	"time"

	"gorm.io/gorm"
)

const (
	qqGovernanceReconcileBatchSize = 50
	qqGovernanceMaxMembersPerShard = 1200
)

type oneBotGroupMember struct {
	UserID int64 `json:"user_id"`
}

// EnqueueScheduledReconciliations 由 cron 或手动入口调用，只创建可恢复的扫描任务。
func (s *QQGovernanceService) EnqueueScheduledReconciliations(ctx context.Context, groupID int64) error {
	policies, err := s.repo.ListEnabledPolicies()
	if err != nil {
		return err
	}
	now := s.now()
	for _, policy := range policies {
		if groupID > 0 && policy.GroupID != groupID {
			continue
		}
		payload, _ := json.Marshal(qqGovernanceActionPayload{Shard: 0})
		bucket := now.Unix() / int64(policy.ScanIntervalMinutes*60)
		_, err := s.repo.CreateActionTaskIfAbsent(&model.QQGovernanceActionTask{
			ActionType: model.QQGovernanceActionScan, IdempotencyKey: fmt.Sprintf("scan:%d:0:%d", policy.GroupID, bucket), GroupID: policy.GroupID,
			PayloadJSON: string(payload), Status: model.QQGovernanceActionPending, Priority: 60, RunAfter: now, Source: "cron",
		})
		if err != nil {
			return err
		}
	}
	return nil
}

func (s *QQGovernanceService) runReconcileTask(ctx context.Context, task *model.QQGovernanceActionTask) error {
	var payload qqGovernanceActionPayload
	if err := json.Unmarshal([]byte(task.PayloadJSON), &payload); err != nil {
		return s.failTask(task, err, false)
	}
	policy, err := s.repo.GetEnabledPolicy(task.GroupID)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return s.repo.CancelActionTask(task.ID, task.LeaseToken, "群治理规则已禁用或删除")
	}
	if err != nil {
		return s.failTask(task, err, true)
	}
	if task.ActionType == model.QQGovernanceActionRecheck {
		return s.reconcileOneMember(ctx, task, policy, task.QQ)
	}
	executor := s.actionExecutor()
	if executor == nil || !executor.OneBotConnected() {
		return s.failTask(task, &OneBotActionError{Message: "OneBot 机器人未连接", Retryable: true}, true)
	}
	callCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	raw, err := executor.CallOneBot(callCtx, "get_group_member_list", map[string]any{"group_id": task.GroupID})
	if err != nil {
		return s.failTask(task, err, isRetryableQQGovernanceError(err))
	}
	var members []oneBotGroupMember
	if err := json.Unmarshal(raw, &members); err != nil {
		return s.failTask(task, fmt.Errorf("解析群成员列表失败: %w", err), true)
	}
	filtered := make([]int64, 0, len(members))
	shardCount := int(math.Ceil(float64(len(members)) / qqGovernanceMaxMembersPerShard))
	if shardCount < 1 {
		shardCount = 1
	}
	botQQ := NewSysConfigService().GetOneBotConfig().BotQQ
	for _, member := range members {
		if botQQ > 0 && member.UserID == botQQ {
			continue
		}
		if member.UserID > 0 && int(member.UserID%int64(shardCount)) == payload.Shard {
			filtered = append(filtered, member.UserID)
		}
	}
	cursor, err := s.repo.GetCursor(task.GroupID, payload.Shard)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		cursor = &model.QQGovernanceReconcileCursor{GroupID: task.GroupID, ShardIndex: payload.Shard, ShardCount: shardCount, NextRunAt: s.now()}
	} else if err != nil {
		return s.failTask(task, err, true)
	}
	start := 0
	for start < len(filtered) && filtered[start] <= cursor.LastQQ {
		start++
	}
	end := start + qqGovernanceReconcileBatchSize
	if end > len(filtered) {
		end = len(filtered)
	}
	for _, qq := range filtered[start:end] {
		if err := s.reconcileMember(ctx, policy, task.GroupID, qq, "scan"); err != nil {
			return s.failTask(task, err, true)
		}
	}
	now := s.now()
	cursor.ShardCount = shardCount
	if end >= len(filtered) {
		cursor.LastQQ = 0
		cursor.LastCompletedAt = &now
	} else {
		cursor.LastQQ = filtered[end-1]
	}
	cursor.NextRunAt = now.Add(15 * time.Minute)
	if err := s.repo.SaveCursor(cursor); err != nil {
		return s.failTask(task, err, true)
	}
	if end < len(filtered) {
		nextPayload, _ := json.Marshal(qqGovernanceActionPayload{Shard: payload.Shard})
		_, err = s.repo.CreateActionTaskIfAbsent(&model.QQGovernanceActionTask{ActionType: model.QQGovernanceActionScan, IdempotencyKey: fmt.Sprintf("scan:%d:%d:%d", task.GroupID, payload.Shard, cursor.LastQQ), GroupID: task.GroupID, PayloadJSON: string(nextPayload), Status: model.QQGovernanceActionPending, Priority: 60, RunAfter: now.Add(time.Second), Source: "reconcile"})
		if err != nil {
			return s.failTask(task, err, true)
		}
	}
	if payload.Shard == 0 {
		for shard := 1; shard < shardCount; shard++ {
			p, _ := json.Marshal(qqGovernanceActionPayload{Shard: shard})
			_, _ = s.repo.CreateActionTaskIfAbsent(&model.QQGovernanceActionTask{ActionType: model.QQGovernanceActionScan, IdempotencyKey: fmt.Sprintf("scan:%d:%d:%d", task.GroupID, shard, now.Unix()/900), GroupID: task.GroupID, PayloadJSON: string(p), Status: model.QQGovernanceActionPending, Priority: 60, RunAfter: now.Add(time.Second), Source: "reconcile"})
		}
	}
	_ = s.logQQGovernanceAction(task, "succeeded", "")
	return s.repo.CompleteActionTask(task.ID, task.LeaseToken, now)
}

func (s *QQGovernanceService) reconcileOneMember(ctx context.Context, task *model.QQGovernanceActionTask, policy *model.QQGroupGovernancePolicy, qq int64) error {
	if err := s.reconcileMember(ctx, policy, task.GroupID, qq, "recheck"); err != nil {
		return s.failTask(task, err, true)
	}
	return s.repo.CompleteActionTask(task.ID, task.LeaseToken, s.now())
}

func (s *QQGovernanceService) reconcileMember(_ context.Context, policy *model.QQGroupGovernancePolicy, groupID, qq int64, source string) error {
	return s.repo.Transaction(func(tx *gorm.DB) error {
		decision, err := s.evaluate(tx, policy, qq)
		if err != nil {
			return err
		}
		state, err := s.repo.GetMemberStateTx(tx, groupID, qq)
		if errors.Is(err, gorm.ErrRecordNotFound) {
			state = &model.QQGroupMemberState{GroupID: groupID, QQ: qq, Version: 1}
		} else if err != nil {
			return err
		} else {
			state.Version++
		}
		now := s.now()
		state.UserID = decision.UserID
		state.TargetCard = decision.TargetCard
		state.LastCheckedAt = now
		switch decision.Decision {
		case model.QQGovernanceReviewMatched:
			state.Status = model.QQGovernanceMemberValid
			state.MismatchCount = 0
			state.UnknownCount = 0
			state.FirstMismatchAt = nil
		case model.QQGovernanceReviewUnmatched:
			state.UnknownCount = 0
			state.MismatchCount++
			if state.FirstMismatchAt == nil {
				state.FirstMismatchAt = &now
			}
			state.Status = model.QQGovernanceMemberInvalidCand
			if policy.MemberViolationPolicy == model.QQGovernanceViolationAutoKick && state.MismatchCount >= policy.MismatchConfirmations && now.Sub(*state.FirstMismatchAt) >= time.Duration(policy.MismatchObservationHours)*time.Hour {
				state.Status = model.QQGovernanceMemberInvalidConf
			}
		default:
			state.Status = model.QQGovernanceMemberReview
			state.UnknownCount++
			state.MismatchCount = 0
			state.FirstMismatchAt = nil
		}
		if err := s.repo.SaveMemberStateTx(tx, state); err != nil {
			return err
		}
		roles, _ := json.Marshal(decision.Roles)
		if err := s.repo.CreateReviewTx(tx, &model.QQGovernanceReview{PolicyID: policy.ID, GroupID: groupID, QQ: qq, Source: source, Decision: decision.Decision, Reason: decision.Reason, UserID: decision.UserID, Nickname: decision.Nickname, PrimaryCharacterName: decision.CharacterName, PrimaryCorporationName: decision.CorporationName, RoleCodesJSON: string(roles)}); err != nil {
			return err
		}
		if decision.Decision == model.QQGovernanceReviewWait {
			if state.UnknownCount >= 3 {
				_, _ = s.repo.CreateAlertIfAbsent(&model.QQGovernanceAlert{AlertKey: fmt.Sprintf("unknown:%d:%d", groupID, qq), Kind: "unknown", GroupID: groupID, QQ: qq, Status: model.QQGovernanceAlertOpen, Message: "成员资料连续三次无法确认"})
			}
			p, _ := json.Marshal(qqGovernanceActionPayload{})
			_, err := s.repo.CreateActionTaskIfAbsentTx(tx, &model.QQGovernanceActionTask{ActionType: model.QQGovernanceActionRecheck, IdempotencyKey: fmt.Sprintf("recheck:%d:%d:%d", groupID, qq, now.Unix()/300), GroupID: groupID, QQ: qq, TargetVersion: state.Version, PayloadJSON: string(p), Status: model.QQGovernanceActionPending, Priority: 100, RunAfter: now.Add(5*time.Minute + time.Duration(rand.IntN(60))*time.Second), Source: "automatic"})
			return err
		}
		if state.Status == model.QQGovernanceMemberInvalidConf {
			p, _ := json.Marshal(qqGovernanceActionPayload{})
			_, err := s.repo.CreateActionTaskIfAbsentTx(tx, &model.QQGovernanceActionTask{ActionType: model.QQGovernanceActionKick, IdempotencyKey: fmt.Sprintf("kick:%d:%d:%d", groupID, qq, state.Version), GroupID: groupID, QQ: qq, TargetVersion: state.Version, PayloadJSON: string(p), Status: model.QQGovernanceActionPending, Priority: 90, RunAfter: now.Add(time.Minute + time.Duration(rand.IntN(181))*time.Second), Source: "automatic"})
			return err
		}
		return nil
	})
}

func (s *QQGovernanceService) RunGovernanceMaintenance(_ context.Context) error {
	now := s.now()
	return s.repo.Cleanup(now.AddDate(0, 0, -90), now.AddDate(0, 0, -180), now.AddDate(0, 0, -180))
}

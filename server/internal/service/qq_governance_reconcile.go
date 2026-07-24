package service

import (
	"amiya-eden/internal/model"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math/rand/v2"
	"sort"
	"time"

	"gorm.io/gorm"
)

const qqGovernanceReconcileBatchSize = 50

type oneBotGroupMember struct {
	UserID int64 `json:"user_id"`
}

// EnqueueScheduledReconciliations starts one durable full-membership run per
// group. Repeated cron/manual triggers reuse the existing run instead of
// competing over a shared cursor.
func (s *QQGovernanceService) EnqueueScheduledReconciliations(ctx context.Context, groupID int64) error {
	policies, err := s.repo.ListEnabledPolicies()
	if err != nil {
		return err
	}
	for _, policy := range policies {
		if groupID > 0 && policy.GroupID != groupID {
			continue
		}
		if err := s.startReconcileRun(ctx, policy.GroupID); err != nil {
			return err
		}
	}
	return nil
}

func (s *QQGovernanceService) startReconcileRun(_ context.Context, groupID int64) error {
	if _, err := s.repo.GetActiveReconcileRun(groupID); err == nil {
		return nil
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}
	now := s.now()
	run := &model.QQGovernanceReconcileRun{
		GroupID: groupID, ActiveKey: fmt.Sprintf("group:%d", groupID), Status: model.QQGovernanceRunPending, StartedAt: &now,
	}
	if err := s.repo.CreateReconcileRun(run); err != nil {
		// A concurrent cron/manual trigger may have won the unique active key.
		if _, activeErr := s.repo.GetActiveReconcileRun(groupID); activeErr == nil {
			return nil
		}
		return err
	}
	payload, _ := json.Marshal(qqGovernanceActionPayload{RunID: run.ID})
	_, err := s.repo.CreateActionTaskIfAbsent(&model.QQGovernanceActionTask{
		ActionType: model.QQGovernanceActionSnapshot, RunID: run.ID,
		IdempotencyKey: fmt.Sprintf("snapshot:%d:%d", groupID, run.ID), GroupID: groupID,
		PayloadJSON: string(payload), Status: model.QQGovernanceActionPending, Priority: 20, RunAfter: now, Source: "cron",
	})
	return err
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
	switch task.ActionType {
	case model.QQGovernanceActionSnapshot:
		return s.captureReconcileSnapshot(ctx, task, policy, payload.RunID)
	case model.QQGovernanceActionComputeBatch:
		return s.computeReconcileBatch(ctx, task, policy, payload.RunID, payload.Batch)
	case model.QQGovernanceActionRecheck:
		return s.reconcileOneMember(ctx, task, policy, task.QQ)
	default:
		return s.failTask(task, fmt.Errorf("未知治理巡检任务: %s", task.ActionType), false)
	}
}

func (s *QQGovernanceService) captureReconcileSnapshot(ctx context.Context, task *model.QQGovernanceActionTask, _ *model.QQGroupGovernancePolicy, runID uint) error {
	if runID == 0 {
		runID = task.RunID
	}
	run, err := s.repo.GetReconcileRun(runID)
	if err != nil {
		return s.failTask(task, fmt.Errorf("读取巡检轮次失败: %w", err), false)
	}
	if run.GroupID != task.GroupID {
		return s.failTask(task, errors.New("巡检轮次与群号不一致"), false)
	}
	if wait, err := s.acquireQQGovernanceRateLimit(ctx, task); err != nil {
		return s.failTask(task, err, true)
	} else if wait > 0 {
		return s.repo.RetryOrDeadActionTask(task.ID, task.LeaseToken, task.RetryCount, s.now().Add(wait), "QQ 操作限流等待", false)
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
	botQQ := NewSysConfigService().GetOneBotConfig().BotQQ
	seen := make(map[int64]struct{}, len(members))
	qqs := make([]int64, 0, len(members))
	for _, member := range members {
		if member.UserID <= 0 || member.UserID == botQQ {
			continue
		}
		if _, ok := seen[member.UserID]; ok {
			continue
		}
		seen[member.UserID] = struct{}{}
		qqs = append(qqs, member.UserID)
	}
	sort.Slice(qqs, func(i, j int) bool { return qqs[i] < qqs[j] })
	now := s.now()
	rows := make([]model.QQGovernanceReconcileMember, 0, len(qqs))
	for _, qq := range qqs {
		rows = append(rows, model.QQGovernanceReconcileMember{RunID: run.ID, GroupID: task.GroupID, QQ: qq, Status: model.QQGovernanceRunMemberPending})
	}
	if err := s.repo.Transaction(func(tx *gorm.DB) error {
		if err := s.repo.CreateRunMembersTx(tx, rows); err != nil {
			return err
		}
		run.Status, run.ExpectedCount, run.StartedAt, run.LastError = model.QQGovernanceRunRunning, len(rows), &now, ""
		return tx.Save(run).Error
	}); err != nil {
		return s.failTask(task, err, true)
	}
	snapshot, snapshotErr := s.repo.GetRuntimeSnapshot(task.GroupID)
	if errors.Is(snapshotErr, gorm.ErrRecordNotFound) {
		snapshot = &model.QQGroupRuntimeSnapshot{GroupID: task.GroupID}
	}
	if snapshotErr == nil || errors.Is(snapshotErr, gorm.ErrRecordNotFound) {
		snapshot.MemberCount, snapshot.LastSyncAttemptAt, snapshot.LastSyncError = len(members), now, ""
		_ = s.repo.SaveRuntimeSnapshot(snapshot)
	}
	payload, _ := json.Marshal(qqGovernanceActionPayload{RunID: run.ID, Batch: 0})
	_, err = s.repo.CreateActionTaskIfAbsent(&model.QQGovernanceActionTask{
		ActionType: model.QQGovernanceActionComputeBatch, RunID: run.ID,
		IdempotencyKey: fmt.Sprintf("compute:%d:%d", run.ID, 0), GroupID: task.GroupID,
		PayloadJSON: string(payload), Status: model.QQGovernanceActionPending, Priority: 40, RunAfter: now, Source: "reconcile",
	})
	if err != nil {
		return s.failTask(task, err, true)
	}
	_ = s.logQQGovernanceAction(task, "succeeded", "")
	return s.repo.CompleteActionTask(task.ID, task.LeaseToken, now)
}

func (s *QQGovernanceService) computeReconcileBatch(ctx context.Context, task *model.QQGovernanceActionTask, policy *model.QQGroupGovernancePolicy, runID uint, batch int) error {
	if runID == 0 {
		runID = task.RunID
	}
	run, err := s.repo.GetReconcileRun(runID)
	if err != nil || run.GroupID != task.GroupID || run.Status != model.QQGovernanceRunRunning {
		return s.repo.CancelActionTask(task.ID, task.LeaseToken, "巡检轮次已结束或不存在")
	}
	rows, err := s.repo.ListPendingRunMembers(run.ID, qqGovernanceReconcileBatchSize)
	if err != nil {
		return s.failTask(task, err, true)
	}
	for _, row := range rows {
		if err := s.reconcileMember(ctx, policy, task.GroupID, row.QQ, "scan"); err != nil {
			return s.failTask(task, err, true)
		}
		if err := s.repo.Transaction(func(tx *gorm.DB) error { return s.repo.CompleteRunMemberTx(tx, row.ID) }); err != nil {
			return s.failTask(task, err, true)
		}
	}
	now := s.now()
	pending, err := s.repo.CountPendingRunMembers(run.ID)
	if err != nil {
		return s.failTask(task, err, true)
	}
	// Recompute processed_count from the frozen membership set so a retry that
	// fires after the final batch already completed can still correct any
	// previously unsaved progress and finish the run.
	run.ProcessedCount = run.ExpectedCount - int(pending)
	if pending == 0 {
		run.Status, run.ActiveKey, run.CompletedAt = model.QQGovernanceRunCompleted, "", &now
		if err := s.repo.SaveReconcileRun(run); err != nil {
			return s.failTask(task, err, true)
		}
		if err := s.repo.MarkMissingMembersLeft(task.GroupID, run.ID, now); err != nil {
			return s.failTask(task, err, true)
		}
		if snapshot, err := s.repo.GetRuntimeSnapshot(task.GroupID); err == nil {
			snapshot.LastSyncedAt, snapshot.LastSyncError = &now, ""
			_ = s.repo.SaveRuntimeSnapshot(snapshot)
		}
	} else {
		if err := s.repo.SaveReconcileRun(run); err != nil {
			return s.failTask(task, err, true)
		}
		payload, _ := json.Marshal(qqGovernanceActionPayload{RunID: run.ID, Batch: batch + 1})
		_, err = s.repo.CreateActionTaskIfAbsent(&model.QQGovernanceActionTask{
			ActionType: model.QQGovernanceActionComputeBatch, RunID: run.ID,
			IdempotencyKey: fmt.Sprintf("compute:%d:%d", run.ID, batch+1), GroupID: task.GroupID,
			PayloadJSON: string(payload), Status: model.QQGovernanceActionPending, Priority: 40, RunAfter: now, Source: "reconcile",
		})
		if err != nil {
			return s.failTask(task, err, true)
		}
	}
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
		state.UserID, state.TargetCard, state.LastCheckedAt = decision.UserID, decision.TargetCard, now
		switch decision.Decision {
		case model.QQGovernanceReviewMatched:
			state.Status, state.MismatchCount, state.UnknownCount, state.FirstMismatchAt = model.QQGovernanceMemberValid, 0, 0, nil
		case model.QQGovernanceReviewUnmatched:
			state.UnknownCount, state.MismatchCount = 0, state.MismatchCount+1
			if state.FirstMismatchAt == nil {
				state.FirstMismatchAt = &now
			}
			state.Status = model.QQGovernanceMemberInvalidCand
			settings := s.GovernanceSettings()
			if policy.MemberViolationPolicy == model.QQGovernanceViolationAutoKick && state.MismatchCount >= settings.MismatchConfirmations && now.Sub(*state.FirstMismatchAt) >= time.Duration(settings.MismatchObservationHours)*time.Hour {
				state.Status = model.QQGovernanceMemberInvalidConf
			}
		default:
			state.Status, state.UnknownCount, state.MismatchCount, state.FirstMismatchAt = model.QQGovernanceMemberReview, state.UnknownCount+1, 0, nil
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

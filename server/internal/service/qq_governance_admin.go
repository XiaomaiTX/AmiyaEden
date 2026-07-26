package service

import (
	"amiya-eden/global"
	"amiya-eden/internal/model"
	"amiya-eden/internal/repository"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"gorm.io/gorm"
)

type QQGovernancePolicyInput struct {
	GroupID               int64    `json:"group_id"`
	Enabled               bool     `json:"enabled"`
	AllowedCorporationIDs []int64  `json:"allowed_corporation_ids"`
	AllowedRoleCodes      []string `json:"allowed_role_codes"`
	AutoRejectUnmatched   bool     `json:"auto_reject_unmatched"`
	MemberViolationPolicy string   `json:"member_violation_policy"`
	CardTemplate          string   `json:"card_template"`
	CardSyncEnabled       bool     `json:"card_sync_enabled"`
}

type QQGovernancePolicyView struct {
	ID                    uint                      `json:"id"`
	GroupID               int64                     `json:"group_id"`
	Enabled               bool                      `json:"enabled"`
	AllowedCorporationIDs []int64                   `json:"allowed_corporation_ids"`
	AllowedCorporations   []QQGovernanceCorporation `json:"allowed_corporations"`
	AllowedRoleCodes      []string                  `json:"allowed_role_codes"`
	AutoRejectUnmatched   bool                      `json:"auto_reject_unmatched"`
	MemberViolationPolicy string                    `json:"member_violation_policy"`
	CardTemplate          string                    `json:"card_template"`
	CardSyncEnabled       bool                      `json:"card_sync_enabled"`
	UpdatedBy             uint                      `json:"updated_by"`
	UpdatedAt             time.Time                 `json:"updated_at"`
}

// QQGovernanceCorporation pairs a stable corporation ID with its display name,
// for read-only display in the admin UI.
type QQGovernanceCorporation struct {
	CorporationID   int64  `json:"corporation_id"`
	CorporationName string `json:"corporation_name"`
}
type QQGovernancePageResult[T any] struct {
	List     []T   `json:"list"`
	Total    int64 `json:"total"`
	Page     int   `json:"page"`
	PageSize int   `json:"page_size"`
}
type QQGovernanceMetrics struct {
	WindowMinutes int     `json:"window_minutes"`
	Created       int64   `json:"created"`
	Succeeded     int64   `json:"succeeded"`
	Failed        int64   `json:"failed"`
	Dead          int64   `json:"dead"`
	FailureRate   float64 `json:"failure_rate"`
	Connected     bool    `json:"connected"`
	RiskLevel     int     `json:"risk_level"`
}

type QQGovernanceRecoveryResult struct {
	RecoveredTasks int64 `json:"recovered_tasks"`
}

type QQGovernanceReconcileTriggerResult struct {
	Status         string `json:"status"`
	RecoveredTasks int64  `json:"recovered_tasks"`
}
type QQGovernanceGroupStatus struct {
	GroupID               int64      `json:"group_id"`
	GroupName             string     `json:"group_name"`
	Enabled               bool       `json:"enabled"`
	MemberCount           int        `json:"member_count"`
	MaxMemberCount        int        `json:"max_member_count"`
	BotIsAdmin            *bool      `json:"bot_is_admin"`
	ValidCount            int64      `json:"valid_count"`
	ReviewCount           int64      `json:"review_count"`
	InvalidCandidateCount int64      `json:"invalid_candidate_count"`
	InvalidConfirmedCount int64      `json:"invalid_confirmed_count"`
	LastSyncedAt          *time.Time `json:"last_synced_at"`
	SnapshotState         string     `json:"snapshot_state"`
	ReconcileRunStatus    string     `json:"reconcile_run_status"`
	ReconcileExpected     int        `json:"reconcile_expected"`
	ReconcileProcessed    int        `json:"reconcile_processed"`
	ReconcileFailed       int        `json:"reconcile_failed"`
	ReconcileStartedAt    *time.Time `json:"reconcile_started_at"`
	ReconcileCompletedAt  *time.Time `json:"reconcile_completed_at"`
}

func (s *QQGovernanceService) ListPolicies(ctx context.Context) ([]QQGovernancePolicyView, error) {
	rows, err := s.repo.ListPolicies(nil)
	if err != nil {
		return nil, err
	}
	nameByID, err := s.collectPolicyCorporationNames(ctx, rows)
	if err != nil {
		return nil, err
	}
	result := make([]QQGovernancePolicyView, 0, len(rows))
	for _, row := range rows {
		view, err := qqGovernancePolicyView(row, nameByID)
		if err != nil {
			return nil, err
		}
		result = append(result, view)
	}
	return result, nil
}
func (s *QQGovernanceService) SavePolicy(ctx context.Context, input QQGovernancePolicyInput, operator uint) (*QQGovernancePolicyView, error) {
	if input.GroupID <= 0 {
		return nil, errors.New("群号必须为正整数")
	}
	if input.MemberViolationPolicy == "" {
		input.MemberViolationPolicy = model.QQGovernanceViolationReview
	}
	if input.MemberViolationPolicy != model.QQGovernanceViolationReview && input.MemberViolationPolicy != model.QQGovernanceViolationAutoKick {
		return nil, errors.New("成员违规策略无效")
	}
	corpJSON, err := json.Marshal(input.AllowedCorporationIDs)
	if err != nil {
		return nil, err
	}
	if _, err := parseQQGovernanceCorps(string(corpJSON)); err != nil {
		return nil, err
	}
	roleJSON, err := json.Marshal(input.AllowedRoleCodes)
	if err != nil {
		return nil, err
	}
	if _, err := parseQQGovernanceRoles(string(roleJSON)); err != nil {
		return nil, err
	}
	if _, err := renderQQGovernanceCard(input.CardTemplate, qqEligibility{Nickname: "n", CharacterName: "c", CorporationName: "o"}); err != nil {
		return nil, err
	}
	policy, err := s.repo.GetPolicy(input.GroupID)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		policy = &model.QQGroupGovernancePolicy{GroupID: input.GroupID}
	} else if err != nil {
		return nil, err
	}
	policy.Enabled = input.Enabled
	policy.AllowedCorporationIDsJSON = string(corpJSON)
	policy.AllowedRoleCodesJSON = string(roleJSON)
	policy.AutoRejectUnmatched = input.AutoRejectUnmatched
	policy.MemberViolationPolicy = input.MemberViolationPolicy
	policy.CardTemplate = strings.TrimSpace(input.CardTemplate)
	policy.CardSyncEnabled = input.CardSyncEnabled
	policy.UpdatedBy = operator
	if err := s.repo.SavePolicy(policy); err != nil {
		return nil, err
	}
	if !policy.CardSyncEnabled {
		if err := s.repo.CancelPendingActionTasks(input.GroupID, model.QQGovernanceActionSetCard, "群名片同步已关闭"); err != nil {
			return nil, err
		}
	}
	s.audit("qq_governance_policy_save", operator, "qq_group_governance_policy", fmt.Sprintf("%d", input.GroupID), map[string]any{"enabled": input.Enabled, "member_violation_policy": input.MemberViolationPolicy, "card_sync_enabled": input.CardSyncEnabled})
	nameByID, err := s.collectPolicyCorporationNames(ctx, []model.QQGroupGovernancePolicy{*policy})
	if err != nil {
		return nil, err
	}
	view, err := qqGovernancePolicyView(*policy, nameByID)
	return &view, err
}
func (s *QQGovernanceService) DeletePolicy(groupID int64, operator uint) error {
	if err := s.repo.DeletePolicy(groupID); err != nil {
		return err
	}
	s.audit("qq_governance_policy_delete", operator, "qq_group_governance_policy", fmt.Sprintf("%d", groupID), nil)
	return nil
}
func qqGovernancePolicyView(row model.QQGroupGovernancePolicy, nameByID map[int64]string) (QQGovernancePolicyView, error) {
	var corps []int64
	var roles []string
	if err := json.Unmarshal([]byte(row.AllowedCorporationIDsJSON), &corps); err != nil {
		return QQGovernancePolicyView{}, err
	}
	if err := json.Unmarshal([]byte(row.AllowedRoleCodesJSON), &roles); err != nil {
		return QQGovernancePolicyView{}, err
	}
	displays := make([]QQGovernanceCorporation, 0, len(corps))
	for _, corporationID := range corps {
		displays = append(displays, QQGovernanceCorporation{
			CorporationID:   corporationID,
			CorporationName: nameByID[corporationID],
		})
	}
	return QQGovernancePolicyView{
		ID:                    row.ID,
		GroupID:               row.GroupID,
		Enabled:               row.Enabled,
		AllowedCorporationIDs: corps,
		AllowedCorporations:   displays,
		AllowedRoleCodes:      roles,
		AutoRejectUnmatched:   row.AutoRejectUnmatched,
		MemberViolationPolicy: row.MemberViolationPolicy,
		CardTemplate:          row.CardTemplate,
		CardSyncEnabled:       row.CardSyncEnabled,
		UpdatedBy:             row.UpdatedBy,
		UpdatedAt:             row.UpdatedAt,
	}, nil
}

// collectPolicyCorporationNames resolves every allowed corporation ID across
// the given policy rows into a single id→name map. Resolution failures never
// block the policy listing — unresolved entries simply yield an empty name
// and the UI falls back to showing the ID.
func (s *QQGovernanceService) collectPolicyCorporationNames(ctx context.Context, rows []model.QQGroupGovernancePolicy) (map[int64]string, error) {
	seen := make(map[int64]struct{})
	for _, row := range rows {
		var corps []int64
		if err := json.Unmarshal([]byte(row.AllowedCorporationIDsJSON), &corps); err != nil {
			return nil, err
		}
		for _, id := range corps {
			if id > 0 {
				seen[id] = struct{}{}
			}
		}
	}
	if len(seen) == 0 {
		return map[int64]string{}, nil
	}
	ids := make([]int64, 0, len(seen))
	for id := range seen {
		ids = append(ids, id)
	}
	resolved := NewSysConfigService().resolveCorporationNamesAllowMissing(ctx, ids)
	return resolved, nil
}
func (s *QQGovernanceService) ListGroupStatuses(ctx context.Context) ([]QQGovernanceGroupStatus, error) {
	policies, err := s.repo.ListPolicies(nil)
	if err != nil {
		return nil, err
	}
	groupIDs := make([]int64, 0, len(policies))
	for _, policy := range policies {
		groupIDs = append(groupIDs, policy.GroupID)
	}
	snapshots, err := s.repo.ListRuntimeSnapshots(groupIDs)
	if err != nil {
		return nil, err
	}
	stateCounts, err := s.repo.CountMemberStatesByGroup(groupIDs)
	if err != nil {
		return nil, err
	}
	runs, err := s.repo.ListLatestReconcileRuns(groupIDs)
	if err != nil {
		return nil, err
	}
	snapshotByGroup := make(map[int64]model.QQGroupRuntimeSnapshot, len(snapshots))
	for _, snapshot := range snapshots {
		snapshotByGroup[snapshot.GroupID] = snapshot
	}
	runByGroup := make(map[int64]model.QQGovernanceReconcileRun, len(runs))
	for _, run := range runs {
		runByGroup[run.GroupID] = run
	}
	countsByGroup := make(map[int64]map[string]int64, len(groupIDs))
	for _, count := range stateCounts {
		if countsByGroup[count.GroupID] == nil {
			countsByGroup[count.GroupID] = make(map[string]int64)
		}
		countsByGroup[count.GroupID][count.Status] = count.Count
	}
	settings, now := s.GovernanceSettings(), s.now()
	result := make([]QQGovernanceGroupStatus, 0, len(policies))
	for _, policy := range policies {
		item := QQGovernanceGroupStatus{GroupID: policy.GroupID, Enabled: policy.Enabled, SnapshotState: "never_synced"}
		if snapshot, ok := snapshotByGroup[policy.GroupID]; ok {
			item.GroupName, item.MemberCount, item.MaxMemberCount, item.BotIsAdmin, item.LastSyncedAt = snapshot.GroupName, snapshot.MemberCount, snapshot.MaxMemberCount, snapshot.BotIsAdmin, snapshot.LastSyncedAt
			if snapshot.LastSyncedAt != nil {
				item.SnapshotState = "fresh"
				if now.Sub(*snapshot.LastSyncedAt) > time.Duration(settings.ScanIntervalMinutes*2)*time.Minute {
					item.SnapshotState = "stale"
				}
			}
		}
		counts := countsByGroup[policy.GroupID]
		item.ValidCount = counts[model.QQGovernanceMemberValid]
		item.ReviewCount = counts[model.QQGovernanceMemberReview]
		item.InvalidCandidateCount = counts[model.QQGovernanceMemberInvalidCand]
		item.InvalidConfirmedCount = counts[model.QQGovernanceMemberInvalidConf]
		if run, ok := runByGroup[policy.GroupID]; ok {
			item.ReconcileRunStatus = run.Status
			item.ReconcileExpected, item.ReconcileProcessed, item.ReconcileFailed = run.ExpectedCount, run.ProcessedCount, run.FailedCount
			item.ReconcileStartedAt, item.ReconcileCompletedAt = run.StartedAt, run.CompletedAt
		}
		result = append(result, item)
	}
	return result, nil
}

func (s *QQGovernanceService) GovernanceSettings() QQGovernanceSettings {
	return NewSysConfigService().GetQQGovernanceSettings()
}

func (s *QQGovernanceService) UpdateGovernanceSettings(input QQGovernanceSettings, operator uint) error {
	if err := NewSysConfigService().UpdateQQGovernanceSettings(input); err != nil {
		return err
	}
	s.audit("qq_governance_settings_update", operator, "system_config", model.SysConfigQQGovernanceScanIntervalMinutes, nil)
	return nil
}

func (s *QQGovernanceService) SearchCorporations(ctx context.Context, query string) ([]CorporationDisplay, error) {
	return NewSysConfigService().SearchCorporations(ctx, query)
}
func (s *QQGovernanceService) ListMemberStates(filter repository.QQGovernanceMemberFilter) (QQGovernancePageResult[model.QQGroupMemberState], error) {
	rows, total, err := s.repo.ListMemberStates(filter)
	return QQGovernancePageResult[model.QQGroupMemberState]{List: rows, Total: total, Page: filter.Page, PageSize: filter.PageSize}, err
}
func (s *QQGovernanceService) ListReviews(filter repository.QQGovernanceReviewFilter) (QQGovernancePageResult[model.QQGovernanceReview], error) {
	rows, total, err := s.repo.ListReviews(filter)
	return QQGovernancePageResult[model.QQGovernanceReview]{List: rows, Total: total, Page: filter.Page, PageSize: filter.PageSize}, err
}
func (s *QQGovernanceService) ListActionTasks(filter repository.QQGovernanceTaskFilter) (QQGovernancePageResult[model.QQGovernanceActionTask], error) {
	rows, total, err := s.repo.ListActionTasks(filter)
	return QQGovernancePageResult[model.QQGovernanceActionTask]{List: rows, Total: total, Page: filter.Page, PageSize: filter.PageSize}, err
}
func (s *QQGovernanceService) ListAlerts(filter repository.QQGovernanceAlertFilter) (QQGovernancePageResult[model.QQGovernanceAlert], error) {
	rows, total, err := s.repo.ListAlerts(filter)
	return QQGovernancePageResult[model.QQGovernanceAlert]{List: rows, Total: total, Page: filter.Page, PageSize: filter.PageSize}, err
}
func (s *QQGovernanceService) AcknowledgeAlert(id, operator uint) error {
	if err := s.repo.AcknowledgeAlert(id, operator, s.now()); err != nil {
		return err
	}
	s.audit("qq_governance_alert_ack", operator, "qq_governance_alert", fmt.Sprint(id), nil)
	return nil
}
func (s *QQGovernanceService) RetryTask(id, operator uint) error {
	if err := s.repo.RetryDeadActionTask(id, s.now()); err != nil {
		return err
	}
	s.audit("qq_governance_task_retry", operator, "qq_governance_action_task", fmt.Sprint(id), nil)
	return nil
}

// OnOneBotConnected is invoked by the protocol layer after a verified reverse
// WebSocket connection is installed. Dead tasks remain dead until an operator
// explicitly restores them.
func (s *QQGovernanceService) OnOneBotConnected() {
	count, err := s.repo.RecoverDisconnectedActionTasks(s.now(), false, 0, 0, nil)
	if err != nil {
		s.logger().Warn("恢复断连等待的 QQ 群治理任务失败")
		return
	}
	if count > 0 {
		s.logger().Info("OneBot 重连后恢复 QQ 群治理任务")
	}
}

func (s *QQGovernanceService) RecoverDisconnectedTasks(operator uint) (QQGovernanceRecoveryResult, error) {
	if executor := s.actionExecutor(); executor == nil || !executor.OneBotConnected() {
		return QQGovernanceRecoveryResult{}, errors.New("OneBot 机器人未连接，不能恢复连接阻塞任务")
	}
	count, err := s.repo.RecoverDisconnectedActionTasks(s.now(), true, 0, 0, nil)
	if err != nil {
		return QQGovernanceRecoveryResult{}, err
	}
	s.audit("qq_governance_disconnected_tasks_recover", operator, "qq_governance_action_task", "connection", map[string]any{"recovered_tasks": count})
	return QQGovernanceRecoveryResult{RecoveredTasks: count}, nil
}
func (s *QQGovernanceService) Metrics() (QQGovernanceMetrics, error) {
	counts, err := s.repo.MetricCounts(s.now().Add(-time.Hour))
	if err != nil {
		return QQGovernanceMetrics{}, err
	}
	result := QQGovernanceMetrics{WindowMinutes: 60, Created: counts.Created, Succeeded: counts.Succeeded, Failed: counts.Failed, Dead: counts.Dead, Connected: s.actionExecutor() != nil && s.actionExecutor().OneBotConnected()}
	if counts.Succeeded+counts.Failed > 0 {
		result.FailureRate = float64(counts.Failed) / float64(counts.Succeeded+counts.Failed)
	}
	if botQQ := NewSysConfigService().GetOneBotConfig().BotQQ; botQQ > 0 {
		if risk, err := s.repo.GetRiskState(botQQ); err == nil {
			result.RiskLevel = risk.Level
		}
	}
	return result, nil
}
func (s *QQGovernanceService) TriggerReconcile(ctx context.Context, groupID int64, operator uint) (QQGovernanceReconcileTriggerResult, error) {
	active, err := s.repo.GetActiveReconcileRun(groupID)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		if err := s.EnqueueScheduledReconciliations(ctx, groupID); err != nil {
			return QQGovernanceReconcileTriggerResult{}, err
		}
		s.audit("qq_governance_reconcile_trigger", operator, "qq_group_governance_policy", fmt.Sprint(groupID), map[string]any{"status": "created"})
		return QQGovernanceReconcileTriggerResult{Status: "created"}, nil
	}
	if err != nil {
		return QQGovernanceReconcileTriggerResult{}, err
	}
	if executor := s.actionExecutor(); executor != nil && executor.OneBotConnected() {
		count, recoverErr := s.repo.RecoverDisconnectedActionTasks(s.now(), true, groupID, active.ID, []string{model.QQGovernanceActionSnapshot})
		if recoverErr != nil {
			return QQGovernanceReconcileTriggerResult{}, recoverErr
		}
		if count > 0 {
			s.audit("qq_governance_reconcile_trigger", operator, "qq_governance_reconcile_run", fmt.Sprint(active.ID), map[string]any{"status": "resumed", "recovered_tasks": count})
			return QQGovernanceReconcileTriggerResult{Status: "resumed", RecoveredTasks: count}, nil
		}
	}
	tasks, err := s.repo.ListActionTasksForRun(active.ID)
	if err != nil {
		return QQGovernanceReconcileTriggerResult{}, err
	}
	for _, task := range tasks {
		if task.ActionType == model.QQGovernanceActionSnapshot && task.Status == model.QQGovernanceActionDead {
			return QQGovernanceReconcileTriggerResult{Status: "blocked"}, nil
		}
	}
	return QQGovernanceReconcileTriggerResult{Status: "running"}, nil
}
func (s *QQGovernanceService) audit(action string, operator uint, resourceType, resourceID string, details map[string]any) {
	if global.DB == nil {
		return
	}
	_ = NewAuditService().RecordEvent(context.Background(), AuditRecordInput{Category: "system", Action: action, ActorUserID: operator, ResourceType: resourceType, ResourceID: resourceID, Result: model.AuditResultSuccess, Details: details})
}

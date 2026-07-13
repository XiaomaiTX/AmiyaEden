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
	GroupID                  int64    `json:"group_id"`
	Enabled                  bool     `json:"enabled"`
	AllowedCorporationIDs    []int64  `json:"allowed_corporation_ids"`
	AllowedRoleCodes         []string `json:"allowed_role_codes"`
	AutoRejectUnmatched      bool     `json:"auto_reject_unmatched"`
	MemberViolationPolicy    string   `json:"member_violation_policy"`
	ScanIntervalMinutes      int      `json:"scan_interval_minutes"`
	MismatchConfirmations    int      `json:"mismatch_confirmations"`
	MismatchObservationHours int      `json:"mismatch_observation_hours"`
	CardTemplate             string   `json:"card_template"`
}

type QQGovernancePolicyView struct {
	ID                       uint      `json:"id"`
	GroupID                  int64     `json:"group_id"`
	Enabled                  bool      `json:"enabled"`
	AllowedCorporationIDs    []int64   `json:"allowed_corporation_ids"`
	AllowedRoleCodes         []string  `json:"allowed_role_codes"`
	AutoRejectUnmatched      bool      `json:"auto_reject_unmatched"`
	MemberViolationPolicy    string    `json:"member_violation_policy"`
	ScanIntervalMinutes      int       `json:"scan_interval_minutes"`
	MismatchConfirmations    int       `json:"mismatch_confirmations"`
	MismatchObservationHours int       `json:"mismatch_observation_hours"`
	CardTemplate             string    `json:"card_template"`
	UpdatedBy                uint      `json:"updated_by"`
	UpdatedAt                time.Time `json:"updated_at"`
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

func (s *QQGovernanceService) ListPolicies() ([]QQGovernancePolicyView, error) {
	rows, err := s.repo.ListPolicies(nil)
	if err != nil {
		return nil, err
	}
	result := make([]QQGovernancePolicyView, 0, len(rows))
	for _, row := range rows {
		view, err := qqGovernancePolicyView(row)
		if err != nil {
			return nil, err
		}
		result = append(result, view)
	}
	return result, nil
}
func (s *QQGovernanceService) SavePolicy(input QQGovernancePolicyInput, operator uint) (*QQGovernancePolicyView, error) {
	if input.GroupID <= 0 {
		return nil, errors.New("群号必须为正整数")
	}
	if input.MemberViolationPolicy == "" {
		input.MemberViolationPolicy = model.QQGovernanceViolationReview
	}
	if input.MemberViolationPolicy != model.QQGovernanceViolationReview && input.MemberViolationPolicy != model.QQGovernanceViolationAutoKick {
		return nil, errors.New("成员违规策略无效")
	}
	if input.ScanIntervalMinutes < 15 || input.ScanIntervalMinutes > 360 || input.ScanIntervalMinutes%15 != 0 {
		return nil, errors.New("扫描间隔必须为 15 到 360 分钟且为 15 的倍数")
	}
	if input.MismatchConfirmations < 2 || input.MismatchConfirmations > 3 {
		return nil, errors.New("连续不匹配次数必须为 2 到 3")
	}
	if input.MismatchObservationHours < 1 || input.MismatchObservationHours > 6 {
		return nil, errors.New("观察期必须为 1 到 6 小时")
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
	policy.ScanIntervalMinutes = input.ScanIntervalMinutes
	policy.MismatchConfirmations = input.MismatchConfirmations
	policy.MismatchObservationHours = input.MismatchObservationHours
	policy.CardTemplate = strings.TrimSpace(input.CardTemplate)
	policy.UpdatedBy = operator
	if err := s.repo.SavePolicy(policy); err != nil {
		return nil, err
	}
	s.audit("qq_governance_policy_save", operator, "qq_group_governance_policy", fmt.Sprintf("%d", input.GroupID), map[string]any{"enabled": input.Enabled, "member_violation_policy": input.MemberViolationPolicy})
	view, err := qqGovernancePolicyView(*policy)
	return &view, err
}
func (s *QQGovernanceService) DeletePolicy(groupID int64, operator uint) error {
	if err := s.repo.DeletePolicy(groupID); err != nil {
		return err
	}
	s.audit("qq_governance_policy_delete", operator, "qq_group_governance_policy", fmt.Sprintf("%d", groupID), nil)
	return nil
}
func qqGovernancePolicyView(row model.QQGroupGovernancePolicy) (QQGovernancePolicyView, error) {
	var corps []int64
	var roles []string
	if err := json.Unmarshal([]byte(row.AllowedCorporationIDsJSON), &corps); err != nil {
		return QQGovernancePolicyView{}, err
	}
	if err := json.Unmarshal([]byte(row.AllowedRoleCodesJSON), &roles); err != nil {
		return QQGovernancePolicyView{}, err
	}
	return QQGovernancePolicyView{ID: row.ID, GroupID: row.GroupID, Enabled: row.Enabled, AllowedCorporationIDs: corps, AllowedRoleCodes: roles, AutoRejectUnmatched: row.AutoRejectUnmatched, MemberViolationPolicy: row.MemberViolationPolicy, ScanIntervalMinutes: row.ScanIntervalMinutes, MismatchConfirmations: row.MismatchConfirmations, MismatchObservationHours: row.MismatchObservationHours, CardTemplate: row.CardTemplate, UpdatedBy: row.UpdatedBy, UpdatedAt: row.UpdatedAt}, nil
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
func (s *QQGovernanceService) TriggerReconcile(ctx context.Context, groupID int64, operator uint) error {
	if err := s.EnqueueScheduledReconciliations(ctx, groupID); err != nil {
		return err
	}
	s.audit("qq_governance_reconcile_trigger", operator, "qq_group_governance_policy", fmt.Sprint(groupID), nil)
	return nil
}
func (s *QQGovernanceService) ManualAction(action string, groupID, qq int64, operator uint) error {
	state, err := s.repo.GetMemberState(groupID, qq)
	if err != nil {
		return err
	}
	payload := qqGovernanceActionPayload{}
	switch action {
	case model.QQGovernanceActionApprove:
		payload.Approve = true
		fallthrough
	case model.QQGovernanceActionReject:
		if !payload.Approve {
			payload.Approve = false
		}
		flag, err := s.repo.GetLatestRequestFlag(groupID, qq)
		if err != nil {
			return errors.New("未找到可处理的入群申请")
		}
		payload.RequestFlag = flag
	case model.QQGovernanceActionSetCard:
		payload.Card = state.TargetCard
		if payload.Card == "" {
			return errors.New("成员没有可同步的目标名片")
		}
	case model.QQGovernanceActionKick:
	default:
		return errors.New("不支持的人工动作")
	}
	data, _ := json.Marshal(payload)
	_, err = s.repo.CreateActionTaskIfAbsent(&model.QQGovernanceActionTask{ActionType: action, IdempotencyKey: fmt.Sprintf("manual:%s:%d:%d:%d", action, groupID, qq, s.now().UnixNano()), GroupID: groupID, QQ: qq, TargetVersion: state.Version, PayloadJSON: string(data), Status: model.QQGovernanceActionPending, Priority: 10, RunAfter: s.now(), Source: "manual"})
	if err == nil {
		s.audit("qq_governance_manual_"+action, operator, "qq_group_member_state", fmt.Sprintf("%d:%d", groupID, qq), nil)
	}
	return err
}
func (s *QQGovernanceService) audit(action string, operator uint, resourceType, resourceID string, details map[string]any) {
	if global.DB == nil {
		return
	}
	_ = NewAuditService().RecordEvent(context.Background(), AuditRecordInput{Category: "system", Action: action, ActorUserID: operator, ResourceType: resourceType, ResourceID: resourceID, Result: model.AuditResultSuccess, Details: details})
}

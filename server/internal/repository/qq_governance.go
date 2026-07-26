package repository

import (
	"amiya-eden/global"
	"amiya-eden/internal/model"
	"errors"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type QQGovernancePage struct {
	Page     int
	PageSize int
}

func (p QQGovernancePage) normalize() (int, int) {
	if p.Page < 1 {
		p.Page = 1
	}
	if p.PageSize < 1 {
		p.PageSize = 20
	}
	if p.PageSize > 100 {
		p.PageSize = 100
	}
	return p.Page, p.PageSize
}

type QQGovernanceMemberFilter struct {
	GroupID int64
	QQ      int64
	Status  string
	QQGovernancePage
}
type QQGovernanceReviewFilter struct {
	GroupID  int64
	QQ       int64
	Decision string
	QQGovernancePage
}
type QQGovernanceTaskFilter struct {
	GroupID, QQ        int64
	Status, ActionType string
	QQGovernancePage
}
type QQGovernanceAlertFilter struct {
	Status string
	QQGovernancePage
}

// QQGovernanceRepository 只负责群治理持久状态与动作任务队列的数据访问。
type QQGovernanceRepository struct{ db *gorm.DB }

func NewQQGovernanceRepository() *QQGovernanceRepository {
	return NewQQGovernanceRepositoryWithDB(global.DB)
}
func NewQQGovernanceRepositoryWithDB(db *gorm.DB) *QQGovernanceRepository {
	return &QQGovernanceRepository{db: db}
}
func (r *QQGovernanceRepository) dbOrGlobal() *gorm.DB {
	if r != nil && r.db != nil {
		return r.db
	}
	return global.DB
}
func (r *QQGovernanceRepository) Transaction(fn func(*gorm.DB) error) error {
	return r.dbOrGlobal().Transaction(fn)
}

func (r *QQGovernanceRepository) GetEnabledPolicyTx(tx *gorm.DB, groupID int64) (*model.QQGroupGovernancePolicy, error) {
	var policy model.QQGroupGovernancePolicy
	if err := tx.Where("group_id = ? AND enabled = ?", groupID, true).First(&policy).Error; err != nil {
		return nil, err
	}
	return &policy, nil
}
func (r *QQGovernanceRepository) GetEnabledPolicy(groupID int64) (*model.QQGroupGovernancePolicy, error) {
	return r.GetEnabledPolicyTx(r.dbOrGlobal(), groupID)
}

func (r *QQGovernanceRepository) GetPolicy(groupID int64) (*model.QQGroupGovernancePolicy, error) {
	var policy model.QQGroupGovernancePolicy
	if err := r.dbOrGlobal().Where("group_id = ?", groupID).First(&policy).Error; err != nil {
		return nil, err
	}
	return &policy, nil
}

func (r *QQGovernanceRepository) ListPolicies(enabled *bool) ([]model.QQGroupGovernancePolicy, error) {
	query := r.dbOrGlobal().Order("group_id ASC")
	if enabled != nil {
		query = query.Where("enabled = ?", *enabled)
	}
	var policies []model.QQGroupGovernancePolicy
	return policies, query.Find(&policies).Error
}

func (r *QQGovernanceRepository) ListEnabledPolicies() ([]model.QQGroupGovernancePolicy, error) {
	enabled := true
	return r.ListPolicies(&enabled)
}

func (r *QQGovernanceRepository) SavePolicy(policy *model.QQGroupGovernancePolicy) error {
	return r.dbOrGlobal().Save(policy).Error
}

func (r *QQGovernanceRepository) DeletePolicy(groupID int64) error {
	return r.dbOrGlobal().Where("group_id = ?", groupID).Delete(&model.QQGroupGovernancePolicy{}).Error
}

// CreateEventIfNewTx 以事件唯一键去重；false 表示重复推送。
func (r *QQGovernanceRepository) CreateEventIfNewTx(tx *gorm.DB, event *model.QQGovernanceEvent) (bool, error) {
	if event == nil {
		return false, errors.New("qq governance event is required")
	}
	result := tx.Clauses(clause.OnConflict{DoNothing: true}).Create(event)
	return result.RowsAffected == 1, result.Error
}
func (r *QQGovernanceRepository) GetMemberStateTx(tx *gorm.DB, groupID, qq int64) (*model.QQGroupMemberState, error) {
	var state model.QQGroupMemberState
	if err := tx.Where("group_id = ? AND qq = ?", groupID, qq).First(&state).Error; err != nil {
		return nil, err
	}
	return &state, nil
}
func (r *QQGovernanceRepository) GetMemberState(groupID, qq int64) (*model.QQGroupMemberState, error) {
	return r.GetMemberStateTx(r.dbOrGlobal(), groupID, qq)
}
func (r *QQGovernanceRepository) SaveMemberStateTx(tx *gorm.DB, state *model.QQGroupMemberState) error {
	if state == nil {
		return errors.New("qq governance member state is required")
	}
	return tx.Save(state).Error
}
func (r *QQGovernanceRepository) GetRuntimeSnapshot(groupID int64) (*model.QQGroupRuntimeSnapshot, error) {
	var snapshot model.QQGroupRuntimeSnapshot
	if err := r.dbOrGlobal().Where("group_id = ?", groupID).First(&snapshot).Error; err != nil {
		return nil, err
	}
	return &snapshot, nil
}
func (r *QQGovernanceRepository) SaveRuntimeSnapshot(snapshot *model.QQGroupRuntimeSnapshot) error {
	if snapshot == nil {
		return errors.New("qq governance runtime snapshot is required")
	}
	return r.dbOrGlobal().Save(snapshot).Error
}

func (r *QQGovernanceRepository) GetActiveReconcileRun(groupID int64) (*model.QQGovernanceReconcileRun, error) {
	var run model.QQGovernanceReconcileRun
	if err := r.dbOrGlobal().Where("group_id = ? AND active_key <> ''", groupID).First(&run).Error; err != nil {
		return nil, err
	}
	return &run, nil
}

func (r *QQGovernanceRepository) GetReconcileRun(id uint) (*model.QQGovernanceReconcileRun, error) {
	var run model.QQGovernanceReconcileRun
	if err := r.dbOrGlobal().First(&run, id).Error; err != nil {
		return nil, err
	}
	return &run, nil
}
func (r *QQGovernanceRepository) ListLatestReconcileRuns(groupIDs []int64) ([]model.QQGovernanceReconcileRun, error) {
	if len(groupIDs) == 0 {
		return []model.QQGovernanceReconcileRun{}, nil
	}
	var rows []model.QQGovernanceReconcileRun
	if err := r.dbOrGlobal().Where("group_id IN ?", groupIDs).Order("group_id ASC").Order("id DESC").Find(&rows).Error; err != nil {
		return nil, err
	}
	latest := make(map[int64]model.QQGovernanceReconcileRun, len(groupIDs))
	for _, row := range rows {
		if _, exists := latest[row.GroupID]; !exists {
			latest[row.GroupID] = row
		}
	}
	result := make([]model.QQGovernanceReconcileRun, 0, len(latest))
	for _, groupID := range groupIDs {
		if row, ok := latest[groupID]; ok {
			result = append(result, row)
		}
	}
	return result, nil
}

func (r *QQGovernanceRepository) CreateReconcileRun(run *model.QQGovernanceReconcileRun) error {
	if run == nil {
		return errors.New("qq governance reconcile run is required")
	}
	return r.dbOrGlobal().Create(run).Error
}

func (r *QQGovernanceRepository) SaveReconcileRun(run *model.QQGovernanceReconcileRun) error {
	if run == nil {
		return errors.New("qq governance reconcile run is required")
	}
	return r.dbOrGlobal().Save(run).Error
}

func (r *QQGovernanceRepository) CreateRunMembersTx(tx *gorm.DB, members []model.QQGovernanceReconcileMember) error {
	if len(members) == 0 {
		return nil
	}
	return tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&members).Error
}

func (r *QQGovernanceRepository) ListPendingRunMembers(runID uint, limit int) ([]model.QQGovernanceReconcileMember, error) {
	if limit < 1 {
		limit = 50
	}
	var rows []model.QQGovernanceReconcileMember
	err := r.dbOrGlobal().Where("run_id = ? AND status = ?", runID, model.QQGovernanceRunMemberPending).
		Order("qq ASC").Limit(limit).Find(&rows).Error
	return rows, err
}

func (r *QQGovernanceRepository) CompleteRunMemberTx(tx *gorm.DB, id uint) error {
	return tx.Model(&model.QQGovernanceReconcileMember{}).Where("id = ? AND status = ?", id, model.QQGovernanceRunMemberPending).
		Update("status", model.QQGovernanceRunMemberDone).Error
}

func (r *QQGovernanceRepository) CountPendingRunMembers(runID uint) (int64, error) {
	var total int64
	err := r.dbOrGlobal().Model(&model.QQGovernanceReconcileMember{}).Where("run_id = ? AND status = ?", runID, model.QQGovernanceRunMemberPending).Count(&total).Error
	return total, err
}

// MarkMissingMembersLeft only runs after a complete, frozen membership scan.
func (r *QQGovernanceRepository) MarkMissingMembersLeft(groupID int64, runID uint, now time.Time) error {
	return r.dbOrGlobal().Model(&model.QQGroupMemberState{}).
		Where("group_id = ? AND status <> ? AND NOT EXISTS (SELECT 1 FROM qq_governance_reconcile_member m WHERE m.run_id = ? AND m.qq = qq_group_member_state.qq AND m.deleted_at IS NULL)", groupID, model.QQGovernanceMemberLeft, runID).
		Updates(map[string]any{"status": model.QQGovernanceMemberLeft, "target_card": "", "last_checked_at": now}).Error
}

func (r *QQGovernanceRepository) ListRuntimeSnapshots(groupIDs []int64) ([]model.QQGroupRuntimeSnapshot, error) {
	if len(groupIDs) == 0 {
		return []model.QQGroupRuntimeSnapshot{}, nil
	}
	var snapshots []model.QQGroupRuntimeSnapshot
	err := r.dbOrGlobal().Where("group_id IN ?", groupIDs).Find(&snapshots).Error
	return snapshots, err
}

type QQGovernanceMemberStateCount struct {
	GroupID int64
	Status  string
	Count   int64
}

func (r *QQGovernanceRepository) CountMemberStatesByGroup(groupIDs []int64) ([]QQGovernanceMemberStateCount, error) {
	if len(groupIDs) == 0 {
		return []QQGovernanceMemberStateCount{}, nil
	}
	var rows []QQGovernanceMemberStateCount
	err := r.dbOrGlobal().Model(&model.QQGroupMemberState{}).
		Select("group_id, status, COUNT(*) AS count").
		Where("group_id IN ?", groupIDs).
		Group("group_id, status").
		Scan(&rows).Error
	return rows, err
}
func (r *QQGovernanceRepository) CreateReviewTx(tx *gorm.DB, review *model.QQGovernanceReview) error {
	if review == nil {
		return errors.New("qq governance review is required")
	}
	return tx.Create(review).Error
}
func (r *QQGovernanceRepository) CreateActionTaskIfAbsentTx(tx *gorm.DB, task *model.QQGovernanceActionTask) (bool, error) {
	if task == nil {
		return false, errors.New("qq governance action task is required")
	}
	result := tx.Clauses(clause.OnConflict{DoNothing: true}).Create(task)
	return result.RowsAffected == 1, result.Error
}

func (r *QQGovernanceRepository) CreateActionTaskIfAbsent(task *model.QQGovernanceActionTask) (bool, error) {
	var created bool
	err := r.Transaction(func(tx *gorm.DB) error {
		var err error
		created, err = r.CreateActionTaskIfAbsentTx(tx, task)
		return err
	})
	return created, err
}

func (r *QQGovernanceRepository) GetLatestRequestFlag(groupID, qq int64) (string, error) {
	var event model.QQGovernanceEvent
	err := r.dbOrGlobal().Where("group_id = ? AND qq = ? AND event_type = ? AND request_flag <> ''", groupID, qq, "request/group_add").Order("created_at DESC").First(&event).Error
	if err != nil {
		return "", err
	}
	return event.RequestFlag, nil
}

func (r *QQGovernanceRepository) GetActionTask(id uint) (*model.QQGovernanceActionTask, error) {
	var task model.QQGovernanceActionTask
	if err := r.dbOrGlobal().First(&task, id).Error; err != nil {
		return nil, err
	}
	return &task, nil
}

func (r *QQGovernanceRepository) RetryDeadActionTask(id uint, now time.Time) error {
	result := r.dbOrGlobal().Model(&model.QQGovernanceActionTask{}).
		Where("id = ? AND status = ?", id, model.QQGovernanceActionDead).
		Updates(map[string]any{"status": model.QQGovernanceActionRetryWait, "run_after": now, "retry_count": 0, "retry_cause": model.QQGovernanceRetryCauseNone, "last_error": "", "completed_at": nil})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

// RecoverDisconnectedActionTasks makes only tasks explicitly blocked by an
// unavailable OneBot connection runnable again.
func (r *QQGovernanceRepository) RecoverDisconnectedActionTasks(now time.Time, includeDead bool, groupID int64, runID uint, actionTypes []string) (int64, error) {
	statuses := []string{model.QQGovernanceActionRetryWait}
	if includeDead {
		statuses = append(statuses, model.QQGovernanceActionDead)
	}
	query := r.dbOrGlobal().Model(&model.QQGovernanceActionTask{}).
		Where("status IN ? AND retry_cause = ?", statuses, model.QQGovernanceRetryCauseDisconnected)
	if groupID > 0 {
		query = query.Where("group_id = ?", groupID)
	}
	if runID > 0 {
		query = query.Where("run_id = ?", runID)
	}
	if len(actionTypes) > 0 {
		query = query.Where("action_type IN ?", actionTypes)
	}
	result := query.Updates(map[string]any{
		"status":           model.QQGovernanceActionPending,
		"run_after":        now,
		"retry_count":      0,
		"retry_cause":      model.QQGovernanceRetryCauseNone,
		"last_error":       "",
		"completed_at":     nil,
		"lease_token":      "",
		"claimed_at":       nil,
		"lease_expires_at": nil,
	})
	return result.RowsAffected, result.Error
}

func (r *QQGovernanceRepository) ListActionTasksForRun(runID uint) ([]model.QQGovernanceActionTask, error) {
	var rows []model.QQGovernanceActionTask
	err := r.dbOrGlobal().Where("run_id = ?", runID).Order("id ASC").Find(&rows).Error
	return rows, err
}

// CancelPendingActionTasks cancels queued tasks for a group/action pair so the
// worker no longer picks them up after a policy toggle disables the action.
func (r *QQGovernanceRepository) CancelPendingActionTasks(groupID int64, actionType, reason string) error {
	now := time.Now()
	result := r.dbOrGlobal().Model(&model.QQGovernanceActionTask{}).
		Where("group_id = ? AND action_type = ? AND status IN ?", groupID, actionType, []string{model.QQGovernanceActionPending, model.QQGovernanceActionRetryWait}).
		Updates(map[string]any{"status": model.QQGovernanceActionCancelled, "completed_at": now, "lease_token": "", "claimed_at": nil, "lease_expires_at": nil, "last_error": reason})
	return result.Error
}

// MarkMemberCardUpdated records a successful set_group_card against the matching
// version. The version guard prevents a stale task from overwriting the field
// after the member state has already advanced.
func (r *QQGovernanceRepository) MarkMemberCardUpdated(groupID, qq int64, version uint64, now time.Time) error {
	result := r.dbOrGlobal().Model(&model.QQGroupMemberState{}).
		Where("group_id = ? AND qq = ? AND version = ?", groupID, qq, version).
		Update("last_card_updated_at", now)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

// ClaimNextActionTask 采用条件更新领取任务；并发 worker 只能有一个成功取得租约。
func (r *QQGovernanceRepository) ClaimNextActionTask(now time.Time, token string, lease time.Duration) (*model.QQGovernanceActionTask, error) {
	if lease <= 0 {
		return nil, errors.New("lease duration must be positive")
	}
	var claimed *model.QQGovernanceActionTask
	err := r.Transaction(func(tx *gorm.DB) error {
		var candidates []model.QQGovernanceActionTask
		if err := tx.Where("(status IN ? AND run_after <= ?) OR (status = ? AND lease_expires_at IS NOT NULL AND lease_expires_at <= ?)", []string{model.QQGovernanceActionPending, model.QQGovernanceActionRetryWait}, now, model.QQGovernanceActionRunning, now).Order("priority ASC").Order("run_after ASC").Order("id ASC").Limit(10).Find(&candidates).Error; err != nil {
			return err
		}
		expires := now.Add(lease)
		for i := range candidates {
			candidate := candidates[i]
			result := tx.Model(&model.QQGovernanceActionTask{}).Where("id = ?", candidate.ID).Where("(status IN ? AND run_after <= ?) OR (status = ? AND lease_expires_at IS NOT NULL AND lease_expires_at <= ?)", []string{model.QQGovernanceActionPending, model.QQGovernanceActionRetryWait}, now, model.QQGovernanceActionRunning, now).Updates(map[string]any{"status": model.QQGovernanceActionRunning, "lease_token": token, "claimed_at": now, "lease_expires_at": expires})
			if result.Error != nil {
				return result.Error
			}
			if result.RowsAffected == 1 {
				candidate.Status, candidate.LeaseToken, candidate.LeaseExpiresAt, candidate.ClaimedAt = model.QQGovernanceActionRunning, token, &expires, &now
				claimed = &candidate
				return nil
			}
		}
		return nil
	})
	return claimed, err
}

func (r *QQGovernanceRepository) CompleteActionTask(id uint, token string, now time.Time) error {
	return r.updateClaimedTask(id, token, map[string]any{"status": model.QQGovernanceActionSucceeded, "completed_at": now, "lease_token": "", "claimed_at": nil, "lease_expires_at": nil, "retry_cause": model.QQGovernanceRetryCauseNone, "last_error": ""})
}
func (r *QQGovernanceRepository) CancelActionTask(id uint, token, reason string) error {
	return r.updateClaimedTask(id, token, map[string]any{"status": model.QQGovernanceActionCancelled, "completed_at": time.Now(), "lease_token": "", "claimed_at": nil, "lease_expires_at": nil, "last_error": reason})
}
func (r *QQGovernanceRepository) RetryOrDeadActionTask(id uint, token string, count int, runAfter time.Time, cause, errMessage string, dead bool) error {
	status := model.QQGovernanceActionRetryWait
	updates := map[string]any{"status": status, "retry_count": count, "retry_cause": cause, "lease_token": "", "claimed_at": nil, "lease_expires_at": nil, "last_error": errMessage, "run_after": runAfter}
	if dead {
		updates["status"] = model.QQGovernanceActionDead
		updates["completed_at"] = time.Now()
	}
	return r.updateClaimedTask(id, token, updates)
}
func (r *QQGovernanceRepository) updateClaimedTask(id uint, token string, updates map[string]any) error {
	result := r.dbOrGlobal().Model(&model.QQGovernanceActionTask{}).Where("id = ? AND status = ? AND lease_token = ?", id, model.QQGovernanceActionRunning, token).Updates(updates)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}
func (r *QQGovernanceRepository) CreateActionLog(log *model.QQGovernanceActionLog) error {
	if log == nil {
		return errors.New("qq governance action log is required")
	}
	return r.dbOrGlobal().Create(log).Error
}

func (r *QQGovernanceRepository) ListMemberStates(filter QQGovernanceMemberFilter) ([]model.QQGroupMemberState, int64, error) {
	page, size := filter.normalize()
	query := r.dbOrGlobal().Model(&model.QQGroupMemberState{})
	if filter.GroupID > 0 {
		query = query.Where("group_id = ?", filter.GroupID)
	}
	if filter.QQ > 0 {
		query = query.Where("qq = ?", filter.QQ)
	}
	if filter.Status != "" {
		query = query.Where("status = ?", filter.Status)
	}
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var rows []model.QQGroupMemberState
	return rows, total, query.Order("last_checked_at DESC").Order("id DESC").Offset((page - 1) * size).Limit(size).Find(&rows).Error
}

func (r *QQGovernanceRepository) ListReviews(filter QQGovernanceReviewFilter) ([]model.QQGovernanceReview, int64, error) {
	page, size := filter.normalize()
	query := r.dbOrGlobal().Model(&model.QQGovernanceReview{})
	if filter.GroupID > 0 {
		query = query.Where("group_id = ?", filter.GroupID)
	}
	if filter.QQ > 0 {
		query = query.Where("qq = ?", filter.QQ)
	}
	if filter.Decision != "" {
		query = query.Where("decision = ?", filter.Decision)
	}
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var rows []model.QQGovernanceReview
	return rows, total, query.Order("created_at DESC").Offset((page - 1) * size).Limit(size).Find(&rows).Error
}

func (r *QQGovernanceRepository) ListActionTasks(filter QQGovernanceTaskFilter) ([]model.QQGovernanceActionTask, int64, error) {
	page, size := filter.normalize()
	query := r.dbOrGlobal().Model(&model.QQGovernanceActionTask{})
	if filter.GroupID > 0 {
		query = query.Where("group_id = ?", filter.GroupID)
	}
	if filter.QQ > 0 {
		query = query.Where("qq = ?", filter.QQ)
	}
	if filter.Status != "" {
		query = query.Where("status = ?", filter.Status)
	}
	if filter.ActionType != "" {
		query = query.Where("action_type = ?", filter.ActionType)
	}
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var rows []model.QQGovernanceActionTask
	return rows, total, query.Order("created_at DESC").Offset((page - 1) * size).Limit(size).Find(&rows).Error
}

func (r *QQGovernanceRepository) ListAlerts(filter QQGovernanceAlertFilter) ([]model.QQGovernanceAlert, int64, error) {
	page, size := filter.normalize()
	query := r.dbOrGlobal().Model(&model.QQGovernanceAlert{})
	if filter.Status != "" {
		query = query.Where("status = ?", filter.Status)
	}
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var rows []model.QQGovernanceAlert
	return rows, total, query.Order("created_at DESC").Offset((page - 1) * size).Limit(size).Find(&rows).Error
}

func (r *QQGovernanceRepository) CreateAlertIfAbsent(alert *model.QQGovernanceAlert) (bool, error) {
	result := r.dbOrGlobal().Clauses(clause.OnConflict{DoNothing: true}).Create(alert)
	return result.RowsAffected == 1, result.Error
}
func (r *QQGovernanceRepository) AcknowledgeAlert(id, operator uint, now time.Time) error {
	result := r.dbOrGlobal().Model(&model.QQGovernanceAlert{}).Where("id = ? AND status = ?", id, model.QQGovernanceAlertOpen).Updates(map[string]any{"status": model.QQGovernanceAlertAcknowledged, "acknowledged_by": operator, "acknowledged_at": now})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}
func (r *QQGovernanceRepository) ResolveAlert(key string, now time.Time) error {
	return r.dbOrGlobal().Model(&model.QQGovernanceAlert{}).Where("alert_key = ? AND status IN ?", key, []string{model.QQGovernanceAlertOpen, model.QQGovernanceAlertAcknowledged}).Updates(map[string]any{"status": model.QQGovernanceAlertResolved, "resolved_at": now}).Error
}

func (r *QQGovernanceRepository) GetRiskState(botQQ int64) (*model.QQGovernanceRiskControlState, error) {
	var s model.QQGovernanceRiskControlState
	if err := r.dbOrGlobal().Where("bot_qq = ?", botQQ).First(&s).Error; err != nil {
		return nil, err
	}
	return &s, nil
}
func (r *QQGovernanceRepository) SaveRiskState(state *model.QQGovernanceRiskControlState) error {
	return r.dbOrGlobal().Save(state).Error
}

func (r *QQGovernanceRepository) CountActionLogsSince(since time.Time) (int64, int64, error) {
	var total, failed int64
	db := r.dbOrGlobal().Model(&model.QQGovernanceActionLog{}).Where("created_at >= ?", since)
	if err := db.Count(&total).Error; err != nil {
		return 0, 0, err
	}
	if err := db.Where("result = ?", "failed").Count(&failed).Error; err != nil {
		return 0, 0, err
	}
	return total, failed, nil
}
func (r *QQGovernanceRepository) Cleanup(beforeEvent, beforeReview, beforeLog time.Time) error {
	return r.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("created_at < ?", beforeEvent).Delete(&model.QQGovernanceEvent{}).Error; err != nil {
			return err
		}
		if err := tx.Where("created_at < ?", beforeReview).Delete(&model.QQGovernanceReview{}).Error; err != nil {
			return err
		}
		return tx.Where("created_at < ?", beforeLog).Delete(&model.QQGovernanceActionLog{}).Error
	})
}

type QQGovernanceMetricCounts struct{ Created, Succeeded, Failed, Dead int64 }

func (r *QQGovernanceRepository) MetricCounts(since time.Time) (QQGovernanceMetricCounts, error) {
	var out QQGovernanceMetricCounts
	db := r.dbOrGlobal()
	if err := db.Model(&model.QQGovernanceActionTask{}).Where("created_at >= ?", since).Count(&out.Created).Error; err != nil {
		return out, err
	}
	if err := db.Model(&model.QQGovernanceActionTask{}).Where("completed_at >= ? AND status = ?", since, model.QQGovernanceActionSucceeded).Count(&out.Succeeded).Error; err != nil {
		return out, err
	}
	if err := db.Model(&model.QQGovernanceActionLog{}).Where("created_at >= ? AND result = ?", since, "failed").Count(&out.Failed).Error; err != nil {
		return out, err
	}
	if err := db.Model(&model.QQGovernanceActionTask{}).Where("status = ?", model.QQGovernanceActionDead).Count(&out.Dead).Error; err != nil {
		return out, err
	}
	return out, nil
}

package repository

import (
	"amiya-eden/internal/model"
	"context"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"gorm.io/gorm/logger"
)

var qqReady = []string{model.QQQueued, model.QQRetryWait}
var qqPending = []string{model.QQQueued, model.QQRetryWait, model.QQExecuting}

type QQOfficialRepository struct{ db *gorm.DB }

func NewQQOfficialRepository(db *gorm.DB) *QQOfficialRepository { return &QQOfficialRepository{db: db} }

// QQ records contain scoped identifiers. Never interpolate their bind values into SQL logs,
// even if joined to a transaction originally opened by another business service.
func (r *QQOfficialRepository) dbFor(ctx context.Context, tx *gorm.DB) *gorm.DB {
	if tx == nil {
		tx = r.db
	}
	return tx.Session(&gorm.Session{Logger: logger.Default.LogMode(logger.Silent)}).WithContext(ctx)
}

func (r *QQOfficialRepository) Transaction(ctx context.Context, tx *gorm.DB, run func(*gorm.DB) error) error {
	db := r.dbFor(ctx, tx)
	if tx != nil {
		return run(db)
	}
	return db.Transaction(run)
}

const qqInstallationID = 1

func (r *QQOfficialRepository) EnsureInstallation(ctx context.Context, tx *gorm.DB) error {
	return r.dbFor(ctx, tx).Clauses(clause.OnConflict{DoNothing: true}).Create(&model.QQOfficialInstallation{ID: qqInstallationID}).Error
}

// A shared lock keeps a credential snapshot valid through admission. Settings
// writes take UPDATE on this row before locking any AppID rows.
func (r *QQOfficialRepository) Installation(ctx context.Context, tx *gorm.DB, lock string) (model.QQOfficialInstallation, error) {
	db := r.dbFor(ctx, tx)
	if lock != "" {
		db = db.Clauses(clause.Locking{Strength: lock})
	}
	var row model.QQOfficialInstallation
	err := db.Where("id = ?", qqInstallationID).Take(&row).Error
	return row, err
}

func (r *QQOfficialRepository) SaveInstallation(ctx context.Context, tx *gorm.DB, row *model.QQOfficialInstallation) error {
	return r.dbFor(ctx, tx).Save(row).Error
}

func (r *QQOfficialRepository) Clock(ctx context.Context, tx *gorm.DB, fallback time.Time) (time.Time, error) {
	db := r.dbFor(ctx, tx)
	if db.Name() != "postgres" {
		return fallback.UTC(), nil
	}
	var now time.Time
	err := db.Raw("SELECT clock_timestamp()").Scan(&now).Error
	return now.UTC(), err
}

func (r *QQOfficialRepository) EnsureApp(ctx context.Context, tx *gorm.DB, id string) error {
	app := model.QQOfficialApp{AppID: id, Generation: 1, TokenState: "idle"}
	return r.dbFor(ctx, tx).Clauses(clause.OnConflict{DoNothing: true}).Create(&app).Error
}

func (r *QQOfficialRepository) App(ctx context.Context, tx *gorm.DB, id string, lock bool) (model.QQOfficialApp, error) {
	db := r.dbFor(ctx, tx)
	if lock {
		db = db.Clauses(clause.Locking{Strength: "UPDATE"})
	}
	var row model.QQOfficialApp
	err := db.Where("app_id = ?", id).Take(&row).Error
	return row, err
}

func (r *QQOfficialRepository) AppIDs(ctx context.Context) ([]string, error) {
	var ids []string
	err := r.dbFor(ctx, nil).Model(&model.QQOfficialApp{}).Order("app_id").Pluck("app_id", &ids).Error
	return ids, err
}

func (r *QQOfficialRepository) SaveApp(ctx context.Context, tx *gorm.DB, row *model.QQOfficialApp) error {
	return r.dbFor(ctx, tx).Save(row).Error
}

func (r *QQOfficialRepository) group(ctx context.Context, tx *gorm.DB, appID, col, id string) (model.QQOfficialGroup, error) {
	var row model.QQOfficialGroup
	err := r.dbFor(ctx, tx).Where("app_id = ?", appID).Where(clause.Eq{Column: col, Value: id}).Take(&row).Error
	return row, err
}

func (r *QQOfficialRepository) Group(ctx context.Context, tx *gorm.DB, appID, id string) (model.QQOfficialGroup, error) {
	return r.group(ctx, tx, appID, "id", id)
}

func (r *QQOfficialRepository) GroupByOpenID(ctx context.Context, tx *gorm.DB, appID, id string) (model.QQOfficialGroup, error) {
	return r.group(ctx, tx, appID, "open_id", id)
}

func (r *QQOfficialRepository) SaveGroup(ctx context.Context, tx *gorm.DB, row *model.QQOfficialGroup) error {
	return r.dbFor(ctx, tx).Save(row).Error
}

func (r *QQOfficialRepository) Groups(ctx context.Context, appID string, offset, limit int) ([]model.QQOfficialGroup, int64, error) {
	db := r.dbFor(ctx, nil).Model(&model.QQOfficialGroup{}).Where("app_id = ?", appID)
	var count int64
	rows := []model.QQOfficialGroup{}
	if err := db.Count(&count).Error; err != nil {
		return rows, 0, err
	}
	err := db.Order("created_at DESC, id DESC").Offset(offset).Limit(limit).Find(&rows).Error
	return rows, count, err
}

func (r *QQOfficialRepository) GroupCount(ctx context.Context, tx *gorm.DB, appID string) (int64, error) {
	var count int64
	err := r.dbFor(ctx, tx).Model(&model.QQOfficialGroup{}).Where("app_id = ?", appID).Count(&count).Error
	return count, err
}

func (r *QQOfficialRepository) TargetGroups(ctx context.Context, tx *gorm.DB, appID, source string, corporation int64) ([]model.QQOfficialGroup, error) {
	db := r.dbFor(ctx, tx).Where("app_id = ? AND present AND authorized AND notification_enabled AND test_verified AND push_state = ?", appID, "allowed")
	switch source {
	case "fleet":
		db = db.Where("fleet_corporation = ?", corporation)
	case "structure":
		db = db.Where("structure_corporation = ?", corporation)
	default:
		return nil, gorm.ErrInvalidData
	}
	rows := []model.QQOfficialGroup{}
	err := db.Order("id").Find(&rows).Error
	return rows, err
}

func (r *QQOfficialRepository) Intent(ctx context.Context, tx *gorm.DB, appID, key string) (model.QQOfficialIntent, error) {
	var row model.QQOfficialIntent
	err := r.dbFor(ctx, tx).Where("app_id = ? AND stable_key = ?", appID, key).Take(&row).Error
	return row, err
}

func (r *QQOfficialRepository) InsertIntent(ctx context.Context, tx *gorm.DB, row *model.QQOfficialIntent, tasks []model.QQOfficialTask) error {
	db := r.dbFor(ctx, tx)
	if err := db.Create(row).Error; err != nil {
		return err
	}
	if len(tasks) > 0 {
		return db.CreateInBatches(tasks, 50).Error
	}
	return nil
}

func (r *QQOfficialRepository) MergedIntentCount(ctx context.Context, tx *gorm.DB, appID, taskID string) (int64, error) {
	var count int64
	err := r.dbFor(ctx, tx).Model(&model.QQOfficialIntent{}).Where("app_id = ? AND merged_task_id = ?", appID, taskID).Count(&count).Error
	return count, err
}

func (r *QQOfficialRepository) IntentTasks(ctx context.Context, tx *gorm.DB, appID, intent string) ([]model.QQOfficialTask, error) {
	rows := []model.QQOfficialTask{}
	err := r.dbFor(ctx, tx).Where("app_id = ? AND intent_id = ?", appID, intent).Order("id").Find(&rows).Error
	return rows, err
}

func (r *QQOfficialRepository) Task(ctx context.Context, tx *gorm.DB, appID, id string) (model.QQOfficialTask, error) {
	var row model.QQOfficialTask
	err := r.dbFor(ctx, tx).Where("app_id = ? AND id = ?", appID, id).Take(&row).Error
	return row, err
}

func (r *QQOfficialRepository) SaveTask(ctx context.Context, tx *gorm.DB, row *model.QQOfficialTask) error {
	return r.dbFor(ctx, tx).Save(row).Error
}

type QQOfficialTaskFilter struct{ State, Kind, Source, GroupID, Reference string }

func (r *QQOfficialRepository) Tasks(ctx context.Context, appID string, filter QQOfficialTaskFilter, offset, limit int) ([]model.QQOfficialTask, int64, error) {
	db := r.dbFor(ctx, nil).Model(&model.QQOfficialTask{}).Where("app_id = ?", appID)
	state, source, reference := filter.State, filter.Source, filter.Reference
	if filter.Kind != "" {
		db = db.Where("kind = ?", filter.Kind)
	}
	if filter.GroupID != "" {
		db = db.Where("group_id = ?", filter.GroupID)
	}
	if state != "" {
		db = db.Where("status = ?", state)
	}
	if source != "" {
		db = db.Where("source = ?", source)
	}
	if reference != "" {
		db = db.Where("reference = ?", reference)
	}
	var count int64
	rows := []model.QQOfficialTask{}
	if err := db.Count(&count).Error; err != nil {
		return rows, 0, err
	}
	err := db.Order("created_at DESC, id DESC").Offset(offset).Limit(limit).Find(&rows).Error
	return rows, count, err
}

func (r *QQOfficialRepository) ReadyHeads(ctx context.Context, tx *gorm.DB, appID string, now time.Time) ([]model.QQOfficialTask, error) {
	db := r.dbFor(ctx, tx)
	heads := db.Model(&model.QQOfficialTask{}).
		Select("id, next_run_at, ROW_NUMBER() OVER (PARTITION BY group_id, urgent ORDER BY created_at, id) AS position").
		Where("app_id = ? AND status IN ? AND expires_at > ?", appID, qqReady, now)
	rows := []model.QQOfficialTask{}
	err := db.Where("id IN (?)", db.Table("(?) AS ready", heads).Select("id").Where("position = 1 AND next_run_at <= ?", now)).Find(&rows).Error
	return rows, err
}

func (r *QQOfficialRepository) QueueDepth(ctx context.Context, tx *gorm.DB, appID, group string, now time.Time) (int64, error) {
	db := r.dbFor(ctx, tx).Model(&model.QQOfficialTask{}).Where("app_id = ? AND status IN ? AND expires_at > ?", appID, qqPending, now)
	if group != "" {
		db = db.Where("group_id = ?", group)
	}
	var count int64
	err := db.Count(&count).Error
	return count, err
}

func (r *QQOfficialRepository) OldestQueued(ctx context.Context, appID string, now time.Time) (*time.Time, error) {
	var task model.QQOfficialTask
	err := r.dbFor(ctx, nil).Select("created_at").Where("app_id = ? AND status IN ? AND expires_at > ?", appID, qqReady, now).
		Order("created_at").Take(&task).Error
	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	return &task.CreatedAt, err
}

func (r *QQOfficialRepository) PendingRead(ctx context.Context, tx *gorm.DB, appID, group, kind string, now time.Time) (model.QQOfficialTask, error) {
	var row model.QQOfficialTask
	err := r.dbFor(ctx, tx).Where("app_id = ? AND group_id = ? AND kind = ? AND status IN ? AND expires_at > ?", appID, group, kind, qqPending, now).
		Order("created_at DESC").Take(&row).Error
	return row, err
}

func (r *QQOfficialRepository) invalidate(ctx context.Context, tx *gorm.DB, db *gorm.DB, state, reason string, now time.Time) error {
	// Project queued decisions to their review record in the same transaction.
	ids := db.Session(&gorm.Session{}).Select("id")
	j := r.dbFor(ctx, tx).Model(&model.QQOfficialJoinRequest{}).Where("status = ? AND task_id IN (?)", "queued", ids)
	joinState := "failed"
	if state == model.QQExpired {
		joinState = "expired"
	}
	if err := j.Update("status", joinState).Error; err != nil {
		return err
	}
	return db.Updates(map[string]any{"status": state, "reason": reason, "finished_at": now, "text": "", "cursor": "", "visited_cursors": nil}).Error
}

func (r *QQOfficialRepository) InvalidateQueued(ctx context.Context, tx *gorm.DB, appID, group, mode, reason string, now time.Time) error {
	db := r.dbFor(ctx, tx).Model(&model.QQOfficialTask{}).Where("app_id = ? AND status IN ?", appID, qqReady)
	if group != "" {
		db = db.Where("group_id = ?", group)
	}
	switch mode {
	case "notification":
		db = db.Where("kind = ?", "send_text")
	case "approval":
		db = db.Where("kind IN ?", []string{"join_list", "join_decision"})
	case "approval_write":
		db = db.Where("kind = ?", "join_decision")
	}
	return r.invalidate(ctx, tx, db, model.QQCancelled, reason, now)
}

func (r *QQOfficialRepository) Expire(ctx context.Context, tx *gorm.DB, appID string, now time.Time) error {
	db := r.dbFor(ctx, tx).Model(&model.QQOfficialTask{}).Where("app_id = ? AND status IN ? AND expires_at <= ?", appID, qqReady, now)
	if err := r.invalidate(ctx, tx, db, model.QQExpired, "expired", now); err != nil {
		return err
	}
	return r.dbFor(ctx, tx).Model(&model.QQOfficialTask{}).
		Where("app_id = ? AND status = ? AND next_run_at <= ?", appID, model.QQAwaitingReview, now).
		Updates(map[string]any{"status": model.QQUnknown, "reason": "review_unconfirmed", "finished_at": now}).Error
}

func (r *QQOfficialRepository) Attempt(ctx context.Context, tx *gorm.DB, id string) (model.QQOfficialAttempt, error) {
	var row model.QQOfficialAttempt
	err := r.dbFor(ctx, tx).Where("id = ?", id).Take(&row).Error
	return row, err
}

func (r *QQOfficialRepository) SaveAttempt(ctx context.Context, tx *gorm.DB, row *model.QQOfficialAttempt) error {
	return r.dbFor(ctx, tx).Save(row).Error
}

func (r *QQOfficialRepository) Attempts(ctx context.Context, appID, task string) ([]model.QQOfficialAttempt, error) {
	rows := []model.QQOfficialAttempt{}
	err := r.dbFor(ctx, nil).Where("app_id = ? AND task_id = ?", appID, task).Order("admitted_at DESC").Limit(100).Find(&rows).Error
	return rows, err
}

func (r *QQOfficialRepository) attemptScope(ctx context.Context, tx *gorm.DB, appID, group, kind string, since time.Time) *gorm.DB {
	db := r.dbFor(ctx, tx).Model(&model.QQOfficialAttempt{}).Where("app_id = ? AND admitted_at > ?", appID, since)
	if group != "" {
		db = db.Where("group_id = ?", group)
	}
	if kind != "" {
		db = db.Where("kind = ?", kind)
	}
	return db
}

// AttemptAtLimit returns the oldest of the latest limit attempts, if that many exist in this window.
func (r *QQOfficialRepository) AttemptAtLimit(ctx context.Context, tx *gorm.DB, appID, group, kind string, since time.Time, limit int) (*time.Time, error) {
	var row model.QQOfficialAttempt
	err := r.attemptScope(ctx, tx, appID, group, kind, since).Select("admitted_at").Order("admitted_at DESC").Offset(limit - 1).Take(&row).Error
	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	return &row.AdmittedAt, err
}

func (r *QQOfficialRepository) CountAttempts(ctx context.Context, appID, kind string, since time.Time) (int64, error) {
	var count int64
	err := r.attemptScope(ctx, nil, appID, "", kind, since).Count(&count).Error
	return count, err
}

func (r *QQOfficialRepository) Cooldowns(ctx context.Context, tx *gorm.DB, appID string) ([]model.QQOfficialCooldown, error) {
	rows := []model.QQOfficialCooldown{}
	err := r.dbFor(ctx, tx).Where("app_id = ?", appID).Order("scope").Find(&rows).Error
	return rows, err
}

func (r *QQOfficialRepository) SaveCooldown(ctx context.Context, tx *gorm.DB, row *model.QQOfficialCooldown) error {
	return r.dbFor(ctx, tx).Save(row).Error
}

func (r *QQOfficialRepository) InsertEvent(ctx context.Context, tx *gorm.DB, row *model.QQOfficialEvent) (bool, error) {
	db := r.dbFor(ctx, tx).Clauses(clause.OnConflict{DoNothing: true}).Create(row)
	return db.RowsAffected == 1, db.Error
}

func (r *QQOfficialRepository) EventBacklog(ctx context.Context, tx *gorm.DB, appID string) (int64, error) {
	var count int64
	err := r.dbFor(ctx, tx).Model(&model.QQOfficialEvent{}).Where("app_id = ? AND processed_at IS NULL", appID).Count(&count).Error
	return count, err
}

func (r *QQOfficialRepository) Events(ctx context.Context, tx *gorm.DB, appID string, limit int) ([]model.QQOfficialEvent, error) {
	rows := []model.QQOfficialEvent{}
	err := r.dbFor(ctx, tx).Where("app_id = ? AND processed_at IS NULL", appID).Order("id").Limit(limit).Find(&rows).Error
	return rows, err
}

func (r *QQOfficialRepository) SaveEvent(ctx context.Context, tx *gorm.DB, row *model.QQOfficialEvent) error {
	return r.dbFor(ctx, tx).Save(row).Error
}

func (r *QQOfficialRepository) JoinRequest(ctx context.Context, tx *gorm.DB, appID, id string) (model.QQOfficialJoinRequest, error) {
	var row model.QQOfficialJoinRequest
	err := r.dbFor(ctx, tx).Where("app_id = ? AND id = ?", appID, id).Take(&row).Error
	return row, err
}

func (r *QQOfficialRepository) JoinRequestByKey(ctx context.Context, tx *gorm.DB, appID, group, key string) (model.QQOfficialJoinRequest, error) {
	var row model.QQOfficialJoinRequest
	err := r.dbFor(ctx, tx).Where("app_id = ? AND group_id = ? AND request_key = ?", appID, group, key).Take(&row).Error
	return row, err
}

func (r *QQOfficialRepository) SaveJoinRequest(ctx context.Context, tx *gorm.DB, row *model.QQOfficialJoinRequest) error {
	return r.dbFor(ctx, tx).Save(row).Error
}

func (r *QQOfficialRepository) JoinRequests(ctx context.Context, appID, group, state string, now time.Time, offset, limit int) ([]model.QQOfficialJoinRequest, int64, error) {
	db := r.dbFor(ctx, nil).Model(&model.QQOfficialJoinRequest{}).Where("app_id = ? AND group_id = ?", appID, group)
	switch state {
	case "pending":
		db = db.Where("status = ? AND expires_at > ?", state, now)
	case "expired":
		db = db.Where("(status = ? OR (status = ? AND expires_at <= ?))", state, "pending", now)
	case "":
	default:
		db = db.Where("status = ?", state)
	}
	var count int64
	rows := []model.QQOfficialJoinRequest{}
	if err := db.Count(&count).Error; err != nil {
		return rows, 0, err
	}
	err := db.Order("applied_at DESC, id DESC").Offset(offset).Limit(limit).Find(&rows).Error
	return rows, count, err
}

func (r *QQOfficialRepository) CloseMissingRequests(ctx context.Context, tx *gorm.DB, appID, group, refresh string, cutoff time.Time) error {
	return r.dbFor(ctx, tx).Model(&model.QQOfficialJoinRequest{}).
		Where("app_id = ? AND group_id = ? AND status = ? AND created_at < ? AND refresh_id <> ?", appID, group, "pending", cutoff, refresh).
		Update("status", "closed_elsewhere").Error
}

func (r *QQOfficialRepository) Cleanup(ctx context.Context, tx *gorm.DB, appID string, now time.Time) error {
	db := r.dbFor(ctx, tx)
	week := now.AddDate(0, 0, -7)
	audit := now.AddDate(0, 0, -90)
	for _, q := range []*gorm.DB{
		db.Where("app_id = ? AND created_at < ?", appID, week).Delete(&model.QQOfficialEvent{}),
		db.Where("app_id = ? AND created_at < ?", appID, week).Delete(&model.QQOfficialJoinRequest{}),
		db.Where("app_id = ? AND admitted_at < ?", appID, audit).Delete(&model.QQOfficialAttempt{}),
		db.Where("app_id = ? AND finished_at < ?", appID, audit).Delete(&model.QQOfficialTask{}),
		db.Where("app_id = ? AND created_at < ? AND NOT EXISTS (SELECT 1 FROM qq_official_task WHERE intent_id = qq_official_intent.id)", appID, audit).Delete(&model.QQOfficialIntent{}),
	} {
		if q.Error != nil {
			return q.Error
		}
	}
	return nil
}

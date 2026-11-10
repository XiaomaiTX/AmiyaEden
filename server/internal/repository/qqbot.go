package repository

import (
	"context"
	"errors"
	"time"

	"amiya-eden/internal/model"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type QQBotRepository struct{ db *gorm.DB }

func NewQQBotRepository(db *gorm.DB) *QQBotRepository { return &QQBotRepository{db: db} }
func (r *QQBotRepository) Transaction(ctx context.Context, fn func(*gorm.DB) error) error {
	return r.db.WithContext(ctx).Transaction(fn)
}
func (r *QQBotRepository) Setting(ctx context.Context) (model.QQBotSetting, error) {
	s := model.QQBotSetting{ID: 1, CIDRs: "127.0.0.1/32", IntervalMS: 1000}
	err := r.db.WithContext(ctx).First(&s, 1).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return s, nil
	}
	return s, err
}
func (r *QQBotRepository) SaveSetting(ctx context.Context, s model.QQBotSetting) error {
	return r.db.WithContext(ctx).Save(&s).Error
}
func (r *QQBotRepository) Submit(ctx context.Context, tx *gorm.DB, o *model.QQBotOperation) error {
	if tx == nil {
		tx = r.db
	}
	tx = tx.WithContext(ctx)
	if err := tx.Clauses(clause.OnConflict{Columns: []clause.Column{{Name: "business_key"}}, DoNothing: true}).Create(o).Error; err != nil {
		return err
	}
	var stored model.QQBotOperation
	err := tx.Where("business_key = ?", o.BusinessKey).First(&stored).Error
	if err == nil {
		*o = stored
	}
	return err
}
func (r *QQBotRepository) Get(ctx context.Context, id string) (model.QQBotOperation, error) {
	var o model.QQBotOperation
	err := r.db.WithContext(ctx).First(&o, "id = ?", id).Error
	return o, err
}
func (r *QQBotRepository) Operations(ctx context.Context) ([]model.QQBotOperation, error) {
	o := []model.QQBotOperation{}
	err := r.db.WithContext(ctx).Order("created_at desc").Limit(200).Find(&o).Error
	return o, err
}
func (r *QQBotRepository) Attempts(ctx context.Context) ([]model.QQBotAttempt, error) {
	o := []model.QQBotAttempt{}
	err := r.db.WithContext(ctx).Order("started_at desc").Limit(200).Find(&o).Error
	return o, err
}
func (r *QQBotRepository) Inboxes(ctx context.Context) ([]model.QQBotInbox, error) {
	o := []model.QQBotInbox{}
	err := r.db.WithContext(ctx).Order("created_at desc").Limit(100).Find(&o).Error
	return o, err
}
func (r *QQBotRepository) Deliveries(ctx context.Context) ([]model.QQBotDelivery, error) {
	o := []model.QQBotDelivery{}
	err := r.db.WithContext(ctx).Order("created_at desc").Limit(200).Find(&o).Error
	return o, err
}
func (r *QQBotRepository) Circuits(ctx context.Context) ([]model.QQBotCircuit, error) {
	o := []model.QQBotCircuit{}
	err := r.db.WithContext(ctx).Order("class").Find(&o).Error
	return o, err
}
func (r *QQBotRepository) Circuit(tx *gorm.DB, class string) (model.QQBotCircuit, error) {
	c := model.QQBotCircuit{Class: class, State: "closed"}
	if err := tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&c).Error; err != nil {
		return c, err
	}
	err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&c, "class = ?", class).Error
	return c, err
}
func (r *QQBotRepository) SaveCircuit(tx *gorm.DB, c *model.QQBotCircuit) error {
	return tx.Save(c).Error
}

// Claim uses compare-and-swap fencing even when SELECT locking is unavailable.
func (r *QQBotRepository) Claim(ctx context.Context, now time.Time) (*model.QQBotOperation, error) {
	var out *model.QQBotOperation
	err := r.Transaction(ctx, func(tx *gorm.DB) error {
		var rows []model.QQBotOperation
		for priority := 0; priority <= 2; priority++ {
			var candidate model.QQBotOperation
			result := tx.Where("status = ? AND available_at <= ? AND priority = ?", "queued", now, priority).Order("created_at asc, id asc").Limit(1).Find(&candidate)
			if result.Error != nil {
				return result.Error
			}
			if result.RowsAffected == 0 {
				continue
			}

			rows = append(rows, candidate)
		}

		best := -1
		score := -1
		for i, o := range rows {
			s := o.Priority + int(now.Sub(o.CreatedAt)/time.Minute)
			if s > 2 {
				s = 2
			}
			if s > score || (s == score && best >= 0 && o.CreatedAt.Before(rows[best].CreatedAt)) {
				best = i
				score = s
			}
		}
		if best < 0 {
			return nil
		}
		o := rows[best]
		token := uuid.NewString()
		result := tx.Model(&model.QQBotOperation{}).Where("id = ? AND status = ?", o.ID, "queued").Updates(map[string]interface{}{"status": "leased", "lease_token": token, "lease_until": now.Add(30 * time.Second)})
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			return nil
		}
		o.Status = "leased"
		o.LeaseToken = token
		o.LeaseUntil = now.Add(30 * time.Second)
		out = &o
		return nil
	})
	return out, err
}
func (r *QQBotRepository) UpdateOwned(tx *gorm.DB, o *model.QQBotOperation) error {
	result := tx.Model(&model.QQBotOperation{}).Where("id = ? AND lease_token = ? AND status IN ?", o.ID, o.LeaseToken, []string{"leased", "executing"}).Select("status", "available_at", "lease_until", "failures", "wait_reason", "result", "terminal_at", "updated_at").Updates(o)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected != 1 {
		return errors.New("lease lost")
	}
	return nil
}
func (r *QQBotRepository) RefreshOwned(tx *gorm.DB, o *model.QQBotOperation) error {
	return tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ? AND lease_token = ?", o.ID, o.LeaseToken).First(o).Error
}
func (r *QQBotRepository) CreateAttempt(tx *gorm.DB, a *model.QQBotAttempt) error {
	return tx.Create(a).Error
}
func (r *QQBotRepository) FinishAttempt(tx *gorm.DB, id, class string, now time.Time) error {
	return tx.Model(&model.QQBotAttempt{}).Where("id = ?", id).Updates(map[string]interface{}{"classification": class, "finished_at": now}).Error
}
func (r *QQBotRepository) Recoverable(ctx context.Context, now time.Time) ([]model.QQBotOperation, error) {
	var rows []model.QQBotOperation
	err := r.db.WithContext(ctx).Where("status IN ? AND lease_until < ?", []string{"leased", "executing"}, now).Find(&rows).Error
	return rows, err
}
func (r *QQBotRepository) Inbox(tx *gorm.DB, i *model.QQBotInbox) error {
	return tx.Clauses(clause.OnConflict{DoNothing: true}).Create(i).Error
}
func (r *QQBotRepository) Delivery(tx *gorm.DB, d *model.QQBotDelivery) error {
	return tx.Clauses(clause.OnConflict{DoNothing: true}).Create(d).Error
}
func (r *QQBotRepository) ClaimDelivery(ctx context.Context, topic, consumer string, now time.Time) (*model.QQBotDelivery, error) {
	var out *model.QQBotDelivery
	err := r.Transaction(ctx, func(tx *gorm.DB) error {
		var d model.QQBotDelivery
		result := tx.Where("topic = ? AND consumer = ? AND ((status = ? AND available_at <= ?) OR (status = ? AND lease_until < ?))", topic, consumer, "pending", now, "leased", now).Order("created_at").Limit(1).Find(&d)
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			return nil
		}

		token := uuid.NewString()
		res := tx.Model(&model.QQBotDelivery{}).Where("id = ? AND ((status = ? AND available_at <= ?) OR (status = ? AND lease_until < ?))", d.ID, "pending", now, "leased", now).Updates(map[string]interface{}{"status": "leased", "lease_token": token, "lease_until": now.Add(30 * time.Second), "attempts": gorm.Expr("attempts + 1")})
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected == 1 {
			d.Status = "leased"
			d.LeaseToken = token
			d.Attempts++
			out = &d
		}
		return nil
	})
	return out, err
}
func (r *QQBotRepository) AckDelivery(tx *gorm.DB, d *model.QQBotDelivery) error {
	res := tx.Model(&model.QQBotDelivery{}).Where("id = ? AND lease_token = ? AND status = ?", d.ID, d.LeaseToken, "leased").Select("status", "available_at", "last_error", "updated_at").Updates(d)
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected != 1 {
		return errors.New("delivery lease lost")
	}
	return nil
}
func (r *QQBotRepository) EventPayload(tx *gorm.DB, id string) (string, error) {
	var i model.QQBotInbox
	err := tx.First(&i, "id = ?", id).Error
	return i.Payload, err
}
func (r *QQBotRepository) ResultPayload(tx *gorm.DB, id string) (model.QQBotOperation, error) {
	var o model.QQBotOperation
	err := tx.First(&o, "id = ?", id).Error
	return o, err
}
func (r *QQBotRepository) Cancel(ctx context.Context, id string) error {
	return r.db.WithContext(ctx).Model(&model.QQBotOperation{}).Where("id = ? AND status IN ?", id, []string{"queued", "leased", "executing"}).Update("cancel_requested", true).Error
}
func (r *QQBotRepository) RetryDelivery(ctx context.Context, id string) error {
	res := r.db.WithContext(ctx).Model(&model.QQBotDelivery{}).Where("id = ? AND status = ? AND attempts < ?", id, "dead", 10).Updates(map[string]interface{}{"status": "pending", "available_at": time.Now()})
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected != 1 {
		return errors.New("retry not allowed")
	}
	return nil
}
func (r *QQBotRepository) Cleanup(ctx context.Context, now time.Time) error {
	return r.Transaction(ctx, func(tx *gorm.DB) error {
		// Bound every consumer lifetime, including repeatedly crashing consumers.
		if err := tx.Model(&model.QQBotDelivery{}).Where("status IN ? AND created_at < ?", []string{"pending", "leased"}, now.Add(-7*24*time.Hour)).Updates(map[string]interface{}{"status": "dead", "last_error": "consumer retention expired"}).Error; err != nil {
			return err
		}
		if err := tx.Where("created_at < ?", now.Add(-90*24*time.Hour)).Delete(&model.QQBotDelivery{}).Error; err != nil {
			return err
		}
		if err := tx.Where("created_at < ? AND id NOT IN (?)", now.Add(-7*24*time.Hour), tx.Model(&model.QQBotDelivery{}).Select("item_id").Where("topic = ? AND status IN ?", "event", []string{"pending", "leased", "dead"})).Delete(&model.QQBotInbox{}).Error; err != nil {
			return err
		}
		if err := tx.Where("started_at < ?", now.Add(-90*24*time.Hour)).Delete(&model.QQBotAttempt{}).Error; err != nil {
			return err
		}
		return tx.Where("terminal_at < ?", now.Add(-90*24*time.Hour)).Delete(&model.QQBotOperation{}).Error
	})
}

func (r *QQBotRepository) RecoverAttempts(tx *gorm.DB, id string, now time.Time) error {
	return tx.Model(&model.QQBotAttempt{}).Where("operation_id = ? AND finished_at IS NULL", id).Updates(map[string]interface{}{"classification": "lease_expired_unknown", "finished_at": now}).Error
}

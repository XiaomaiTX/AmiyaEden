package service

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"amiya-eden/internal/model"

	"gorm.io/gorm"
)

type QQBotSnapshot struct {
	RedisAvailable bool                   `json:"redis_available"`
	RateWaitMS     int64                  `json:"rate_wait_ms"`
	Connected      bool                   `json:"connected"`
	LastFrame      time.Time              `json:"last_frame"`
	IngressError   string                 `json:"ingress_error"`
	Operations     []model.QQBotOperation `json:"operations"`
	Attempts       []model.QQBotAttempt   `json:"attempts"`
	Events         []model.QQBotInbox     `json:"events"`
	Deliveries     []model.QQBotDelivery  `json:"deliveries"`
	Circuits       []model.QQBotCircuit   `json:"circuits"`
}

func (s *QQBotService) Snapshot(ctx context.Context) (QQBotSnapshot, error) {
	var v QQBotSnapshot
	var e error
	if e = s.repo.Transaction(ctx, func(tx *gorm.DB) error {
		for _, class := range []string{"read", "write"} {
			if _, err := s.repo.Circuit(tx, class); err != nil {
				return err
			}
		}
		return nil
	}); e != nil {
		return v, e
	}
	if s.redis != nil {
		redisCtx, cancel := context.WithTimeout(ctx, time.Second)
		if _, err := s.redis.Ping(redisCtx).Result(); err == nil {
			if ttl, err := s.redis.PTTL(redisCtx, "qqbot:rate:global").Result(); err == nil {
				v.RedisAvailable = true
				v.RateWaitMS = max(0, ttl.Milliseconds())
			}
		}
		cancel()
	}
	s.client.mu.Lock()
	v.Connected = s.client.current != nil
	v.LastFrame = s.client.lastFrame
	v.IngressError = s.client.ingressError
	s.client.mu.Unlock()
	if v.Operations, e = s.repo.Operations(ctx); e != nil {
		return v, e
	}
	if v.Attempts, e = s.repo.Attempts(ctx); e != nil {
		return v, e
	}
	if v.Events, e = s.repo.Inboxes(ctx); e != nil {
		return v, e
	}
	if v.Deliveries, e = s.repo.Deliveries(ctx); e != nil {
		return v, e
	}
	v.Circuits, e = s.repo.Circuits(ctx)
	return v, e
}
func (s *QQBotService) Cancel(ctx context.Context, id string) error {
	if err := s.repo.Cancel(ctx, id); err != nil {
		return err
	}
	s.mu.Lock()
	cancel := s.active[id]
	s.mu.Unlock()
	if cancel != nil {
		cancel()
	}
	return nil
}
func (s *QQBotService) RetryDelivery(ctx context.Context, id string) error {
	return s.repo.RetryDelivery(ctx, id)
}

// Retries create a new immutable result identity, never overwrite a consumed terminal result.
func (s *QQBotService) RetryOperation(ctx context.Context, id string) (model.QQBotOperation, error) {
	o, e := s.repo.Get(ctx, id)
	if e != nil {
		return o, e
	}
	if qqbotManagedBusiness(o.BusinessKey) {
		return o, errors.New("business notification retry requires a new business intent")
	}
	if o.Status != "failed" || o.Failures < 3 || !o.ExpiresAt.After(time.Now()) || len(o.BusinessKey) > 180 {
		return o, errors.New("retry not allowed")
	}
	// At most one manual retry generation, with its own deterministic key.
	if len(o.BusinessKey) >= 6 && o.BusinessKey[:6] == "retry:" {
		return o, errors.New("retry limit reached")
	}
	var req model.QQBotRequest
	if e = json.Unmarshal([]byte(o.Request), &req); e != nil {
		return o, e
	}
	return s.Submit(ctx, nil, "retry:"+o.ID, o.Priority, o.ExpiresAt, req)
}
func (s *QQBotService) Pause(ctx context.Context, class string, paused bool) error {
	if class != "read" && class != "write" {
		return errors.New("invalid circuit")
	}
	return s.repo.Transaction(ctx, func(tx *gorm.DB) error {
		c, e := s.repo.Circuit(tx, class)
		if e != nil {
			return e
		}
		c.Paused = paused
		return s.repo.SaveCircuit(tx, &c)
	})
}

func qqbotManagedBusiness(key string) bool {
	for _, prefix := range []string{"notification:", "application:", "card:", "kick:", "inspection:"} {
		if strings.HasPrefix(key, prefix) {
			return true
		}
	}
	return false
}

package service

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"amiya-eden/global"
	"amiya-eden/internal/model"
	"amiya-eden/internal/repository"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

func int64String(id int64) string { return strconv.FormatInt(id, 10) }

type QQBotConsumer func(context.Context, *gorm.DB, string) error
type qqbotConsumer struct {
	topic, name string
	fn          QQBotConsumer
}
type QQBotService struct {
	repo          *repository.QQBotRepository
	redis         redis.UniversalClient
	client        *qqbotClient
	mu            sync.Mutex
	consumers     []qqbotConsumer
	started       bool
	active        map[string]context.CancelFunc
	stepMu        sync.Mutex
	notifications *QQNotificationService
	groups        *QQGroupBusinessService
}

var qqbotSingleton struct {
	sync.Mutex
	svc *QQBotService
}

func DefaultQQBotService() *QQBotService {
	qqbotSingleton.Lock()
	defer qqbotSingleton.Unlock()
	if qqbotSingleton.svc == nil {
		var client redis.UniversalClient
		if global.Redis != nil {
			client = global.Redis
		}
		qqbotSingleton.svc = NewQQBotService(global.DB, client)
		qqbotSingleton.svc.groups = NewQQGroupBusinessService(global.DB, qqbotSingleton.svc)
		if err := qqbotSingleton.svc.RegisterConsumer("event", "qq-group-business", qqbotSingleton.svc.groups.ConsumeEvent); err != nil {
			panic(err)
		}
		if err := qqbotSingleton.svc.RegisterConsumer("result", "qq-group-business", qqbotSingleton.svc.groups.ConsumeResult); err != nil {
			panic(err)
		}
		qqbotSingleton.svc.notifications = NewQQNotificationService(global.DB, qqbotSingleton.svc)
		if err := qqbotSingleton.svc.RegisterConsumer("result", "qq-notifications", qqbotSingleton.svc.notifications.ConsumeResult); err != nil {
			panic(err)
		}
	}
	return qqbotSingleton.svc
}
func NewQQBotService(db *gorm.DB, r redis.UniversalClient) *QQBotService {
	return &QQBotService{repo: repository.NewQQBotRepository(db), redis: r, client: newQQBotClient(), active: make(map[string]context.CancelFunc)}
}
func (s *QQBotService) RegisterConsumer(topic, name string, fn QQBotConsumer) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.started || (topic != "event" && topic != "result") || name == "" || len(name) > 100 || fn == nil {
		return errors.New("invalid consumer registration")
	}
	for _, c := range s.consumers {
		if c.topic == topic && c.name == name {
			return errors.New("duplicate consumer")
		}
	}
	s.consumers = append(s.consumers, qqbotConsumer{topic, name, fn})
	return nil
}
func qqbotProtocol(r model.QQBotRequest) (string, string, map[string]interface{}, error) {
	p := map[string]interface{}{}
	class := "write"
	action := ""
	switch r.Kind {
	case "group_message":
		action = "send_group_msg"
		if r.Text == "" || len(r.Text) > 16000 {
			return "", "", nil, errors.New("invalid message")
		}
		p["message"] = []map[string]interface{}{{"type": "text", "data": map[string]string{"text": r.Text}}}
	case "group_info":
		action = "get_group_info"
		class = "read"
		p["no_cache"] = true
	case "group_list":
		action = "get_group_list"
		class = "read"
	case "member_info":
		action = "get_group_member_info"
		class = "read"
		p["no_cache"] = true
	case "member_list":
		action = "get_group_member_list"
		class = "read"
	case "group_request":
		action = "set_group_add_request"
		if r.Flag == "" || (r.SubType != "add" && r.SubType != "invite") {
			return "", "", nil, errors.New("invalid request")
		}
		p["flag"] = r.Flag
		p["sub_type"] = r.SubType
		p["approve"] = r.Approve
		p["reason"] = r.Text
	case "group_card":
		action = "set_group_card"
		if len(r.Text) > 180 {
			return "", "", nil, errors.New("card too long")
		}
		p["card"] = r.Text
	case "group_kick":
		action = "set_group_kick"
		p["reject_add_request"] = r.RejectAddRequest
	default:
		return "", "", nil, errors.New("unsupported operation")
	}
	if r.Kind != "group_list" && r.Kind != "group_request" {
		if r.GroupID <= 0 {
			return "", "", nil, errors.New("invalid group")
		}
		p["group_id"] = r.GroupID
	}
	if r.Kind == "member_info" || r.Kind == "group_card" || r.Kind == "group_kick" {
		if r.UserID <= 0 {
			return "", "", nil, errors.New("invalid user")
		}
		p["user_id"] = r.UserID
	}
	return action, class, p, nil
}
func qqbotRetryUnknown(kind string) bool {
	return kind == "group_info" || kind == "group_list" || kind == "member_info" || kind == "member_list" || kind == "group_card"
}
func qqbotHash(raw []byte) string { h := sha256.Sum256(raw); return hex.EncodeToString(h[:]) }
func (s *QQBotService) Submit(ctx context.Context, tx *gorm.DB, key string, priority int, expires time.Time, req model.QQBotRequest) (model.QQBotOperation, error) {
	_, class, _, err := qqbotProtocol(req)
	if err != nil {
		return model.QQBotOperation{}, err
	}
	now := time.Now()
	if strings.TrimSpace(key) == "" || len(key) > 200 || priority < 0 || priority > 2 || !expires.After(now) || expires.After(now.Add(7*24*time.Hour)) {
		return model.QQBotOperation{}, errors.New("invalid operation envelope")
	}
	raw, err := json.Marshal(req)
	if err != nil {
		return model.QQBotOperation{}, err
	}
	hash := qqbotHash(raw)
	o := model.QQBotOperation{ID: uuid.NewString(), BusinessKey: key, RequestHash: hash, Request: string(raw), Kind: req.Kind, Class: class, Status: "queued", Priority: priority, ExpiresAt: expires, AvailableAt: now, CreatedAt: now}
	err = s.repo.Submit(ctx, tx, &o)
	if err == nil && o.RequestHash != hash {
		err = errors.New("business key content conflict")
	}
	return o, err
}
func (s *QQBotService) GetOperation(ctx context.Context, id string) (model.QQBotOperation, error) {
	return s.repo.Get(ctx, id)
}
func (s *QQBotService) ServeReverse(w http.ResponseWriter, r *http.Request) {
	s.client.mu.Lock()
	generation := s.client.generation
	s.client.mu.Unlock()
	setting, err := s.repo.Setting(r.Context())
	if err != nil {
		http.Error(w, "unavailable", http.StatusServiceUnavailable)
		return
	}
	s.client.serve(w, r, setting, generation)
}
func (s *QQBotService) Setting(ctx context.Context) (model.QQBotSetting, error) {
	v, e := s.repo.Setting(ctx)
	v.Token = ""
	return v, e
}
func (s *QQBotService) SetSetting(ctx context.Context, v model.QQBotSetting, token *string) error {
	s.stepMu.Lock()
	defer s.stepMu.Unlock()
	if v.IntervalMS < 100 || v.IntervalMS > 60000 || v.BotID < 0 {
		return errors.New("invalid configuration")
	}
	for _, block := range strings.Split(v.CIDRs, ",") {
		if _, _, e := net.ParseCIDR(strings.TrimSpace(block)); e != nil {
			return errors.New("invalid CIDR")
		}
	}
	old, e := s.repo.Setting(ctx)
	if e != nil {
		return e
	}
	v.ID = 1
	v.Token = old.Token
	if token != nil {
		v.Token = *token
	}
	if v.Enabled && (v.Token == "" || v.BotID <= 0) {
		return errors.New("token and bot identity required")
	}
	if len(v.Token) > 256 {
		return errors.New("invalid token")
	}
	if e = s.repo.SaveSetting(ctx, v); e == nil {
		s.client.reconfigure()
	}
	return e
}
func (s *QQBotService) Close() { s.client.close() }

// Redis owns all admission math. Both buckets are checked before either is changed.
var qqbotLimit = redis.NewScript(`
for _,k in ipairs(KEYS) do
 local t=redis.call('PTTL',k)
 if t>0 then return t end
end
for _,k in ipairs(KEYS) do redis.call('SET',k,'1','PX',ARGV[1]) end
return 0`)

func (s *QQBotService) admit(ctx context.Context, group int64, ms int) (string, error) {
	if s.redis == nil {
		return "redis_unavailable", nil
	}
	keys := []string{"qqbot:rate:global"}
	if group > 0 {
		keys = append(keys, "qqbot:rate:group:"+int64String(group))
	}
	_, err := s.redis.Ping(ctx).Result()
	if err != nil {
		return "redis_unavailable", nil
	}
	wait, err := qqbotLimit.Run(ctx, s.redis, keys, ms).Int64()
	if err != nil {
		return "redis_unavailable", nil
	}
	if wait > 0 {
		return "rate_limited", nil
	}
	return "", nil
}
func (s *QQBotService) newDeliveries(tx *gorm.DB, topic, id string) error {
	for _, c := range s.consumers {
		if c.topic == topic {
			if err := s.repo.Delivery(tx, &model.QQBotDelivery{ID: uuid.NewString(), Topic: topic, ItemID: id, Consumer: c.name, Status: "pending", AvailableAt: time.Now()}); err != nil {
				return err
			}
		}
	}
	return nil
}
func (s *QQBotService) finish(tx *gorm.DB, o *model.QQBotOperation, status, result string, now time.Time) error {
	o.Status = status
	o.Result = result
	o.TerminalAt = &now
	o.UpdatedAt = now
	if err := s.repo.UpdateOwned(tx, o); err != nil {
		return err
	}
	return s.newDeliveries(tx, "result", o.ID)
}
func (s *QQBotService) wait(ctx context.Context, o *model.QQBotOperation, reason string, now time.Time) error {
	o.Status = "queued"
	o.WaitReason = reason
	o.AvailableAt = now.Add(time.Second)
	o.UpdatedAt = now
	return s.repo.Transaction(ctx, func(tx *gorm.DB) error { return s.repo.UpdateOwned(tx, o) })
}
func (s *QQBotService) Recover(ctx context.Context, now time.Time) error {
	rows, e := s.repo.Recoverable(ctx, now)
	if e != nil {
		return e
	}
	for _, row := range rows {
		o := row
		if e = s.repo.Transaction(ctx, func(tx *gorm.DB) error {
			if e := s.repo.RefreshOwned(tx, &o); e != nil {
				return e
			}
			if o.LeaseUntil.After(now) || (o.Status != "leased" && o.Status != "executing") {
				return nil
			}
			if err := s.repo.RecoverAttempts(tx, o.ID, now); err != nil {
				return err
			}
			if o.Status == "executing" && !qqbotRetryUnknown(o.Kind) {
				return s.finish(tx, &o, "unknown", "lease_expired_after_execution", now)
			}
			o.Status = "queued"
			o.WaitReason = "lease_recovered"
			o.AvailableAt = now
			o.UpdatedAt = now
			return s.repo.UpdateOwned(tx, &o)
		}); e != nil {
			return e
		}
	}
	return nil
}

// Inspect without reserving a half-open probe or consuming limiter tokens.
// A paused class must not repeatedly take the global bucket from healthy work.
func (s *QQBotService) riskReady(ctx context.Context, class string, now time.Time) (bool, error) {
	ready := false
	err := s.repo.Transaction(ctx, func(tx *gorm.DB) error {
		c, e := s.repo.Circuit(tx, class)
		if e != nil {
			return e
		}
		ready = !c.Paused && (c.State == "closed" || !c.ProbeAt.After(now))
		return nil
	})
	return ready, err
}
func (s *QQBotService) gate(ctx context.Context, class string, now time.Time) (bool, error) {
	allowed := false
	err := s.repo.Transaction(ctx, func(tx *gorm.DB) error {
		c, e := s.repo.Circuit(tx, class)
		if e != nil {
			return e
		}
		if c.Paused {
			return nil
		}
		if c.State == "closed" {
			allowed = true
			return nil
		}
		if c.ProbeAt.After(now) {
			return nil
		}
		c.State = "half_open"
		c.ProbeAt = now.Add(30 * time.Second)
		allowed = true
		return s.repo.SaveCircuit(tx, &c)
	})
	return allowed, err
}
func (s *QQBotService) recordCircuit(tx *gorm.DB, class, outcome string, now time.Time) error {
	c, e := s.repo.Circuit(tx, class)
	if e != nil {
		return e
	}
	if outcome == "success" {
		c.Successes++
		c.Failures = 0
		c.State = "closed"
	} else {
		c.Errors++
		c.Failures++
		if c.Failures >= 3 || c.State == "half_open" {
			c.State = "open"
			c.ProbeAt = now.Add(30 * time.Second)
		}
	}
	return s.repo.SaveCircuit(tx, &c)
}
func (s *QQBotService) RunOnce(ctx context.Context, now time.Time) error {
	s.stepMu.Lock()
	defer s.stepMu.Unlock()
	if e := s.Recover(ctx, now); e != nil {
		return e
	}
	o, e := s.repo.Claim(ctx, now)
	if e != nil || o == nil {
		return e
	}
	if !o.ExpiresAt.After(now) || o.CancelRequested {
		return s.repo.Transaction(ctx, func(tx *gorm.DB) error {
			status := "expired"
			if o.CancelRequested {
				status = "cancelled"
			}
			return s.finish(tx, o, status, "", now)
		})
	}
	setting, e := s.repo.Setting(ctx)
	if e != nil {
		return s.wait(ctx, o, "configuration_unavailable", now)
	}
	if !setting.Enabled || !s.client.connected() {
		return s.wait(ctx, o, "disconnected", now)
	}
	var req model.QQBotRequest
	if json.Unmarshal([]byte(o.Request), &req) != nil {
		return errors.New("corrupt operation")
	}
	action, _, params, e := qqbotProtocol(req)
	if e != nil {
		return e
	}
	ready, e := s.riskReady(ctx, o.Class, now)
	if e != nil {
		return e
	}
	if !ready {
		return s.wait(ctx, o, "risk_paused", now)
	}
	reason, e := s.admit(ctx, req.GroupID, setting.IntervalMS)
	if e != nil {
		return e
	}
	if reason != "" {
		return s.wait(ctx, o, reason, now)
	}
	allowed, e := s.gate(ctx, o.Class, now)
	if e != nil {
		return e
	}
	if !allowed {
		return s.wait(ctx, o, "risk_paused", now)
	}
	execCtx, execCancel := context.WithDeadline(ctx, o.ExpiresAt)
	s.mu.Lock()
	s.active[o.ID] = execCancel
	s.mu.Unlock()
	defer func() { execCancel(); s.mu.Lock(); delete(s.active, o.ID); s.mu.Unlock() }()
	a := model.QQBotAttempt{ID: uuid.NewString(), OperationID: o.ID, Classification: "executing", StartedAt: now}
	execute := false
	e = s.repo.Transaction(ctx, func(tx *gorm.DB) error {
		if s.notifications != nil {
			valid, err := s.notifications.valid(tx, o.ID)
			if err != nil {
				return err
			}
			if !valid {
				return s.finish(tx, o, "cancelled", "channel_disabled", time.Now())
			}
		}
		if s.groups != nil {
			valid, e := s.groups.valid(tx, *o)
			if errors.Is(e, errQQBusinessRecheck) {
				o.Status = "queued"
				o.WaitReason = "business_recheck"
				o.AvailableAt = time.Now().Add(5 * time.Minute)
				return s.repo.UpdateOwned(tx, o)
			}
			if e != nil {
				return e
			}
			if !valid {
				return s.finish(tx, o, "cancelled", "business_invalidated", time.Now())
			}
		}
		if e := s.repo.RefreshOwned(tx, o); e != nil {
			return e
		}
		if o.Status != "leased" {
			return errors.New("operation no longer leased")
		}
		if o.CancelRequested {
			return s.finish(tx, o, "cancelled", "", time.Now())
		}
		if !o.ExpiresAt.After(time.Now()) {
			return s.finish(tx, o, "expired", "", time.Now())
		}

		o.Status = "executing"
		if e := s.repo.UpdateOwned(tx, o); e != nil {
			return e
		}
		if e := s.repo.CreateAttempt(tx, &a); e != nil {
			return e
		}
		execute = true
		return nil
	})
	if e != nil || !execute {
		return e
	}
	resp, outcome := s.client.call(execCtx, action, params)
	if outcome == "success" && !qqbotValidateData(req, resp.Data) {
		outcome = "unknown"
		resp.FailureClass = "data"
	}
	finished := time.Now()
	// Persist even if shutdown cancelled the execution context; no database transaction spans call.
	persistCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	return s.repo.Transaction(persistCtx, func(tx *gorm.DB) error {
		if e := s.repo.FinishAttempt(tx, a.ID, qqbotAttemptClass(outcome, resp.FailureClass), finished); e != nil {
			return e
		}
		if outcome == "disconnected" || outcome == "busy" {
			o.Status = "queued"
			o.WaitReason = outcome
			o.AvailableAt = finished.Add(time.Second)
			return s.repo.UpdateOwned(tx, o)
		}
		if e := s.recordCircuit(tx, o.Class, outcome, finished); e != nil {
			return e
		}
		switch outcome {
		case "success":
			return s.finish(tx, o, "succeeded", string(resp.Data), finished)
		case "permanent":
			return s.finish(tx, o, "failed", fmt.Sprintf("retcode:%d", *resp.Retcode), finished)
		case "transient":
			o.Failures++
			if o.Failures >= 3 {
				return s.finish(tx, o, "failed", fmt.Sprintf("retcode:%d", *resp.Retcode), finished)
			}
		default:
			if !qqbotRetryUnknown(o.Kind) {
				return s.finish(tx, o, "unknown", "unconfirmed", finished)
			}
		}
		o.Status = "queued"
		o.WaitReason = outcome
		o.AvailableAt = finished.Add(time.Duration(1<<min(o.Failures+1, 6)) * time.Second)
		o.UpdatedAt = finished
		return s.repo.UpdateOwned(tx, o)
	})
}
func qqbotValidateData(req model.QQBotRequest, data json.RawMessage) bool {
	switch req.Kind {
	case "group_message":
		var d struct {
			ID *int64 `json:"message_id"`
		}
		return json.Unmarshal(data, &d) == nil && d.ID != nil
	case "group_info":
		var d struct {
			ID int64 `json:"group_id"`
		}
		return json.Unmarshal(data, &d) == nil && d.ID == req.GroupID
	case "member_info":
		var d struct {
			ID    int64 `json:"user_id"`
			Group int64 `json:"group_id"`
		}
		return json.Unmarshal(data, &d) == nil && d.ID == req.UserID && d.Group == req.GroupID
	case "group_list":
		var groups []struct {
			ID   int64   `json:"group_id"`
			Name *string `json:"group_name"`
		}
		if string(data) == "null" || json.Unmarshal(data, &groups) != nil {
			return false
		}
		for _, g := range groups {
			if g.ID <= 0 || g.Name == nil {
				return false
			}
		}
		return true
	case "member_list":
		var members []struct {
			ID    int64 `json:"user_id"`
			Group int64 `json:"group_id"`
		}
		if string(data) == "null" || json.Unmarshal(data, &members) != nil {
			return false
		}
		for _, m := range members {
			if m.ID <= 0 || m.Group != req.GroupID {
				return false
			}
		}
		return true

	default:
		return string(data) == "null"
	}
}
func (s *QQBotService) persistEvent(ctx context.Context, raw []byte) error {
	var v map[string]interface{}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.UseNumber()
	if err := decoder.Decode(&v); err != nil {
		return err
	}
	canonical, err := json.Marshal(v)
	if err != nil {
		return err
	}
	// Decode identity without float rounding.
	var identity struct {
		ID int64 `json:"self_id"`
	}
	if err = json.Unmarshal(raw, &identity); err != nil {
		return err
	}
	if identity.ID <= 0 {
		return errors.New("invalid event identity")
	}

	i := model.QQBotInbox{ID: qqbotHash(canonical), Payload: string(raw), CreatedAt: time.Now()}
	return s.repo.Transaction(ctx, func(tx *gorm.DB) error {
		if err := s.repo.Inbox(tx, &i); err != nil {
			return err
		}
		return s.newDeliveries(tx, "event", i.ID)
	})
}
func (s *QQBotService) consumeOnce(ctx context.Context, c qqbotConsumer) error {
	d, e := s.repo.ClaimDelivery(ctx, c.topic, c.name, time.Now())
	if e != nil || d == nil {
		return e
	}
	runCtx, cancel := context.WithTimeout(ctx, 20*time.Second)
	defer cancel()
	e = s.repo.Transaction(runCtx, func(tx *gorm.DB) error {
		var payload string
		if c.topic == "event" {
			var err error
			payload, err = s.repo.EventPayload(tx, d.ItemID)
			if err != nil {
				return err
			}
		} else {
			o, err := s.repo.ResultPayload(tx, d.ItemID)
			if err != nil {
				return err
			}
			raw, err := json.Marshal(o)
			if err != nil {
				return err
			}
			payload = string(raw)
		}
		if err := qqbotInvokeConsumer(c.fn, runCtx, tx, payload); err != nil {
			return err
		}
		d.Status = "done"
		return s.repo.AckDelivery(tx, d)
	})
	if e == nil {
		return nil
	}
	qqbotLog(e)
	d.Status = "pending"
	if d.Attempts >= 5 {
		d.Status = "dead"
	}
	d.LastError = "consumer failed"
	d.AvailableAt = time.Now().Add(time.Duration(1<<min(d.Attempts, 8)) * time.Second)
	return s.repo.Transaction(ctx, func(tx *gorm.DB) error { return s.repo.AckDelivery(tx, d) })
}
func qqbotInvokeConsumer(fn QQBotConsumer, ctx context.Context, tx *gorm.DB, payload string) (err error) {
	defer func() {
		if recover() != nil {
			err = errors.New("consumer panicked")
		}
	}()
	return fn(ctx, tx, payload)
}

// Drain accepted in-memory frames on shutdown, with an explicit error for every
// frame that cannot be persisted within the shared shutdown budget.
func (s *QQBotService) drainEvents(current []byte) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if current != nil {
		qqbotLog(s.persistEvent(ctx, current))
	}
	for {
		select {
		case raw := <-s.client.events:
			qqbotLog(s.persistEvent(ctx, raw))
		default:
			return
		}
	}
}
func (s *QQBotService) Run(ctx context.Context) error {
	s.mu.Lock()
	s.started = true
	s.mu.Unlock()
	defer s.Close()
	// Separate ingress persistence and consumers from the protocol reader and operation worker.
	var wg sync.WaitGroup
	launch := func(fn func()) { wg.Add(1); go func() { defer wg.Done(); fn() }() }
	launch(func() {
		for {
			select {
			case <-ctx.Done():
				s.drainEvents(nil)
				return
			case raw := <-s.client.events:
				for {
					if e := s.persistEvent(ctx, raw); e == nil {
						break
					} else {
						s.client.setIngressError("event persistence failed")
						s.client.disconnect()
						qqbotLog(e)
					}
					select {
					case <-ctx.Done():
						s.drainEvents(raw)
						return
					case <-time.After(time.Second):
					}
				}
			}
		}
	})
	for _, consumer := range s.consumers {
		c := consumer
		launch(func() {
			t := time.NewTicker(time.Second)
			defer t.Stop()
			for {
				select {
				case <-ctx.Done():
					return
				case <-t.C:
					qqbotLog(s.consumeOnce(ctx, c))
				}
			}
		})
	}
	ticker := time.NewTicker(250 * time.Millisecond)
	defer ticker.Stop()
	cleanup := time.NewTicker(time.Hour)
	defer cleanup.Stop()
	for {
		select {
		case <-ctx.Done():
			s.Close()
			wg.Wait()
			return nil
		case <-ticker.C:
			qqbotLog(s.RunOnce(ctx, time.Now()))
		case <-cleanup.C:
			qqbotLog(s.repo.Cleanup(ctx, time.Now()))
		}
	}
}
func qqbotLog(e error) {
	if e != nil && global.Logger != nil {
		global.Logger.Warn("qqbot infrastructure failure", zap.Error(e))
	}
}

func qqbotAttemptClass(outcome, cause string) string {
	if outcome == "unknown" && cause != "" {
		return "unknown_" + cause
	}
	return outcome
}

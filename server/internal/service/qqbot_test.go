package service

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"amiya-eden/internal/model"

	"github.com/alicebob/miniredis/v2"
	"github.com/glebarez/sqlite"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"golang.org/x/net/websocket"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func qqbotTestService(t *testing.T) (*QQBotService, *gorm.DB, *miniredis.Miniredis) {
	t.Helper()
	db, e := gorm.Open(sqlite.Open("file:"+uuid.NewString()+"?mode=memory&cache=shared"), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if e != nil {
		t.Fatal(e)
	}
	sql, e := db.DB()
	if e != nil {
		t.Fatal(e)
	}
	sql.SetMaxOpenConns(1)
	t.Cleanup(func() { _ = sql.Close() })
	if e = db.AutoMigrate(&model.QQBotOperation{}, &model.QQBotAttempt{}, &model.QQBotInbox{}, &model.QQBotDelivery{}, &model.QQBotCircuit{}, &model.QQBotSetting{}); e != nil {
		t.Fatal(e)
	}
	mr := miniredis.RunT(t)
	r := redis.NewClient(&redis.Options{Addr: mr.Addr(), MaxRetries: -1, DialTimeout: 100 * time.Millisecond, ReadTimeout: 100 * time.Millisecond})
	t.Cleanup(func() { _ = r.Close() })
	svc := NewQQBotService(db, r)
	token := "secret"
	if e = svc.SetSetting(context.Background(), model.QQBotSetting{Enabled: true, BotID: 42, CIDRs: "127.0.0.1/32,::1/128", IntervalMS: 100}, &token); e != nil {
		t.Fatal(e)
	}
	t.Cleanup(svc.Close)
	return svc, db, mr
}
func qqbotSubmit(t *testing.T, s *QQBotService, key, kind string) model.QQBotOperation {
	t.Helper()
	o, e := s.Submit(context.Background(), nil, key, 0, time.Now().Add(time.Hour), model.QQBotRequest{Kind: kind, GroupID: 8, Text: "hello", UserID: 9})
	if e != nil {
		t.Fatal(e)
	}
	return o
}
func qqbotDial(t *testing.T, s *QQBotService) *websocket.Conn {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(s.ServeReverse))
	t.Cleanup(srv.Close)
	cfg, e := websocket.NewConfig("ws"+strings.TrimPrefix(srv.URL, "http"), "http://localhost")
	if e != nil {
		t.Fatal(e)
	}
	cfg.Header.Set("Authorization", "Bearer secret")
	cfg.Header.Set("X-Self-ID", "42")
	cfg.Header.Set("X-Client-Role", "Universal")
	s.client.mu.Lock()
	previous := s.client.current
	s.client.mu.Unlock()
	conn, e := websocket.DialConfig(cfg)
	if e != nil {
		t.Fatal(e)
	}
	deadline := time.Now().Add(time.Second)
	for {
		s.client.mu.Lock()
		ready := s.client.current != nil && s.client.current != previous
		s.client.mu.Unlock()
		if ready {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("connection not installed")
		}
		time.Sleep(time.Millisecond)
	}
	t.Cleanup(func() { _ = conn.Close() })
	return conn
}
func TestQQBotAuthentication(t *testing.T) {
	s, _, _ := qqbotTestService(t)
	setting, e := s.repo.Setting(context.Background())
	if e != nil {
		t.Fatal(e)
	}
	for _, change := range []string{"token", "bot", "cidr", "role", "forwarded"} {
		r := httptest.NewRequest("GET", "/", nil)
		r.RemoteAddr = "127.0.0.1:1234"
		r.Header.Set("Authorization", "Bearer secret")
		r.Header.Set("X-Self-ID", "42")
		r.Header.Set("X-Client-Role", "Universal")
		if !qqbotAuthenticate(r, setting) {
			t.Fatal("valid rejected")
		}
		switch change {
		case "token":
			r.Header.Set("Authorization", "Bearer wrong")
		case "bot":
			r.Header.Set("X-Self-ID", "43")
		case "cidr":
			r.RemoteAddr = "192.0.2.1:123"
		case "role":
			r.Header.Set("X-Client-Role", "API")
		case "forwarded":
			r.RemoteAddr = "192.0.2.1:123"
			r.Header.Set("X-Forwarded-For", "127.0.0.1")
		}
		rec := httptest.NewRecorder()
		s.ServeReverse(rec, r)
		if rec.Code != 403 {
			t.Fatalf("%s: %d", change, rec.Code)
		}
	}
	masked, e := s.Setting(context.Background())
	raw, _ := json.Marshal(masked)
	if e != nil || strings.Contains(string(raw), "secret") {
		t.Fatal("token exposed")
	}
}
func TestQQBotIdempotencyAndTransaction(t *testing.T) {
	s, db, _ := qqbotTestService(t)
	first := qqbotSubmit(t, s, "stable", "group_message")
	second := qqbotSubmit(t, s, "stable", "group_message")
	if first.ID != second.ID {
		t.Fatal("not idempotent")
	}
	if _, e := s.Submit(context.Background(), nil, "stable", 0, time.Now().Add(time.Hour), model.QQBotRequest{Kind: "group_message", GroupID: 8, Text: "different"}); e == nil {
		t.Fatal("content conflict accepted")
	}
	e := db.Transaction(func(tx *gorm.DB) error {
		_, err := s.Submit(context.Background(), tx, "rolledback", 0, time.Now().Add(time.Hour), model.QQBotRequest{Kind: "group_info", GroupID: 8})
		if err != nil {
			return err
		}
		return errors.New("rollback")
	})
	if e == nil {
		t.Fatal("expected rollback")
	}
	var count int64
	db.Model(&model.QQBotOperation{}).Where("business_key = ?", "rolledback").Count(&count)
	if count != 0 {
		t.Fatal("intent survived rollback")
	}
}
func TestQQBotSlowEventsAndReplacement(t *testing.T) {
	s, _, _ := qqbotTestService(t)
	conn := qqbotDial(t, s)
	done := make(chan string, 1)
	go func() {
		_, out := s.client.call(context.Background(), "get_group_list", map[string]interface{}{})
		done <- out
	}()
	var action map[string]interface{}
	if e := websocket.JSON.Receive(conn, &action); e != nil {
		t.Fatal(e)
	}
	if e := websocket.Message.Send(conn, `{"self_id":42,"time":1,"post_type":"notice"}`); e != nil {
		t.Fatal(e)
	}
	if e := websocket.JSON.Send(conn, map[string]interface{}{"echo": action["echo"], "status": "ok", "retcode": 0, "data": []interface{}{}}); e != nil {
		t.Fatal(e)
	}
	select {
	case out := <-done:
		if out != "success" {
			t.Fatal(out)
		}
	case <-time.After(time.Second):
		t.Fatal("slow event blocked echo")
	}
	go func() { _, out := s.client.call(context.Background(), "get_group_list", nil); done <- out }()
	if e := websocket.JSON.Receive(conn, &action); e != nil {
		t.Fatal(e)
	}
	_ = qqbotDial(t, s)
	select {
	case out := <-done:
		if out != "unknown" {
			t.Fatal(out)
		}
	case <-time.After(time.Second):
		t.Fatal("replacement leaked pending")
	}
}
func TestQQBotStrictResponseAndTimeout(t *testing.T) {
	for _, response := range []string{`{"status":"ok","data":null}`, `{"status":"ok","retcode":1,"data":null}`, `{"status":"async","retcode":1,"data":null}`} {
		t.Run(response, func(t *testing.T) {
			s, _, _ := qqbotTestService(t)
			conn := qqbotDial(t, s)
			done := make(chan string, 1)
			go func() { _, out := s.client.call(context.Background(), "x", nil); done <- out }()
			var a map[string]interface{}
			if e := websocket.JSON.Receive(conn, &a); e != nil {
				t.Fatal(e)
			}
			var r map[string]interface{}
			_ = json.Unmarshal([]byte(response), &r)
			r["echo"] = a["echo"]
			if e := websocket.JSON.Send(conn, r); e != nil {
				t.Fatal(e)
			}
			if out := <-done; out != "unknown" {
				t.Fatal(out)
			}
		})
	}
	s, _, _ := qqbotTestService(t)
	_ = qqbotDial(t, s)
	s.client.timeout = 30 * time.Millisecond
	_, out := s.client.call(context.Background(), "x", nil)
	if out != "unknown" {
		t.Fatal(out)
	}
}
func TestQQBotSQLLeasesFairnessAndRecovery(t *testing.T) {
	s, db, _ := qqbotTestService(t)
	o := qqbotSubmit(t, s, "old-low", "group_message")
	db.Model(&model.QQBotOperation{}).Where("id = ?", o.ID).Update("created_at", time.Now().Add(-5*time.Minute))
	high := qqbotSubmit(t, s, "new-high", "group_info")
	db.Model(&model.QQBotOperation{}).Where("id = ?", high.ID).Update("priority", 2)
	var wg sync.WaitGroup
	ids := make(chan string, 4)
	for i := 0; i < 4; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			claimed, e := s.repo.Claim(context.Background(), time.Now())
			if e != nil {
				t.Error(e)
			}
			if claimed != nil {
				ids <- claimed.ID
			}
		}()
	}
	wg.Wait()
	close(ids)
	seen := map[string]bool{}
	for id := range ids {
		if seen[id] {
			t.Fatal("duplicate lease")
		}
		seen[id] = true
	}
	if len(seen) != 2 {
		t.Fatal(seen)
	}
	db.Model(&model.QQBotOperation{}).Where("id = ?", o.ID).Updates(map[string]interface{}{"status": "executing", "lease_until": time.Now().Add(-time.Minute)})
	db.Model(&model.QQBotOperation{}).Where("id = ?", high.ID).Update("lease_until", time.Now().Add(-time.Minute))
	restarted := NewQQBotService(db, s.redis)
	if e := restarted.Recover(context.Background(), time.Now()); e != nil {
		t.Fatal(e)
	}
	recovered, e := s.GetOperation(context.Background(), o.ID)
	if e != nil || recovered.Status != "unknown" {
		t.Fatal(recovered, e)
	}
	recovered, e = s.GetOperation(context.Background(), high.ID)
	if e != nil || recovered.Status != "queued" {
		t.Fatal(recovered, e)
	}
	claimed, e := s.repo.Claim(context.Background(), time.Now())
	if e != nil || claimed == nil || claimed.ID != high.ID {
		t.Fatal(claimed, e)
	}
}
func TestQQBotPauseExpiryAndRedis(t *testing.T) {
	s, db, mr := qqbotTestService(t)
	o := qqbotSubmit(t, s, "pause", "group_info")
	if e := s.RunOnce(context.Background(), time.Now()); e != nil {
		t.Fatal(e)
	}
	got, _ := s.GetOperation(context.Background(), o.ID)
	if got.Failures != 0 || got.WaitReason != "disconnected" {
		t.Fatal(got)
	}
	conn := qqbotDial(t, s)
	_ = conn
	mr.Close()
	if e := s.RunOnce(context.Background(), time.Now().Add(2*time.Second)); e != nil {
		t.Fatal(e)
	}
	got, _ = s.GetOperation(context.Background(), o.ID)
	if got.Failures != 0 || got.WaitReason != "redis_unavailable" {
		t.Fatal(got)
	}
	db.Model(&model.QQBotOperation{}).Where("id = ?", o.ID).Update("expires_at", time.Now().Add(-time.Second))
	if e := s.RunOnce(context.Background(), time.Now().Add(4*time.Second)); e != nil {
		t.Fatal(e)
	}
	got, _ = s.GetOperation(context.Background(), o.ID)
	if got.Status != "expired" {
		t.Fatal(got)
	}
}
func TestQQBotEventsConsumerRecoveryAndCleanup(t *testing.T) {
	s, db, _ := qqbotTestService(t)
	fail := true
	if e := s.RegisterConsumer("event", "test", func(_ context.Context, tx *gorm.DB, _ string) error {
		if fail {
			return errors.New("consumer failed")
		}
		return tx.Create(&model.QQBotInbox{ID: "side-effect", Payload: "{}"}).Error
	}); e != nil {
		t.Fatal(e)
	}
	raw := []byte(`{"time":1,"self_id":42,"post_type":"notice","user_id":9007199254740993}`)
	for i := 0; i < 2; i++ {
		if e := s.persistEvent(context.Background(), raw); e != nil {
			t.Fatal(e)
		}
	}
	var count int64
	db.Model(&model.QQBotInbox{}).Count(&count)
	if count != 1 {
		t.Fatal("event not deduped")
	}
	c := s.consumers[0]
	if e := s.consumeOnce(context.Background(), c); e != nil {
		t.Fatal(e)
	}
	var d model.QQBotDelivery
	db.First(&d)
	if d.Status != "pending" || d.Attempts != 1 {
		t.Fatal(d)
	}
	fail = false
	db.Model(&d).Update("available_at", time.Now().Add(-time.Second))
	if e := s.consumeOnce(context.Background(), c); e != nil {
		t.Fatal(e)
	}
	db.First(&d)
	if d.Status != "done" {
		t.Fatal(d)
	}
	db.Model(&model.QQBotInbox{}).Where("id <> ?", "side-effect").Update("created_at", time.Now().Add(-100*24*time.Hour))
	db.Model(&model.QQBotDelivery{}).Where("id = ?", d.ID).Update("created_at", time.Now().Add(-100*24*time.Hour))
	if e := s.repo.Cleanup(context.Background(), time.Now()); e != nil {
		t.Fatal(e)
	}
	db.Model(&model.QQBotDelivery{}).Count(&count)
	if count != 0 {
		t.Fatal("no physical cleanup")
	}
	db.Model(&model.QQBotInbox{}).Count(&count)
	if count != 1 {
		t.Fatal("inbox cleanup failed")
	}
}
func TestQQBotHalfOpenProbe(t *testing.T) {
	s, _, _ := qqbotTestService(t)
	ctx := context.Background()
	now := time.Now()
	for i := 0; i < 3; i++ {
		if e := s.repo.Transaction(ctx, func(tx *gorm.DB) error { return s.recordCircuit(tx, "write", "unknown", now) }); e != nil {
			t.Fatal(e)
		}
	}
	if ok, e := s.gate(ctx, "write", now); e != nil || ok {
		t.Fatal("open admitted", e)
	}
	if ok, e := s.gate(ctx, "read", now); e != nil || !ok {
		t.Fatal("write affected read", e)
	}
	if ok, e := s.gate(ctx, "write", now.Add(31*time.Second)); e != nil || !ok {
		t.Fatal("no half open", e)
	}
	if ok, e := s.gate(ctx, "write", now.Add(31*time.Second)); e != nil || ok {
		t.Fatal("multiple probes", e)
	}
	if e := s.repo.Transaction(ctx, func(tx *gorm.DB) error { return s.recordCircuit(tx, "write", "success", now.Add(32*time.Second)) }); e != nil {
		t.Fatal(e)
	}
	if ok, e := s.gate(ctx, "write", now.Add(33*time.Second)); e != nil || !ok {
		t.Fatal("not closed", e)
	}
}

func TestQQBotExecutionOutcomesAndResults(t *testing.T) {
	for _, tc := range []struct {
		kind, outcome, want string
		code                int
	}{
		{"group_message", "timeout", "unknown", 0}, {"group_card", "timeout", "queued", 0},
		{"group_message", "failed", "failed", 1403}, {"group_message", "ok", "succeeded", 0},
		{"group_info", "failed", "queued", 1500},
	} {
		t.Run(tc.kind+tc.outcome, func(t *testing.T) {
			s, db, _ := qqbotTestService(t)
			s.client.timeout = 30 * time.Millisecond
			conn := qqbotDial(t, s)
			consumed := ""
			if e := s.RegisterConsumer("result", "result-test", func(_ context.Context, _ *gorm.DB, p string) error { consumed = p; return nil }); e != nil {
				t.Fatal(e)
			}
			o := qqbotSubmit(t, s, "execution", tc.kind)
			served := make(chan error, 1)
			go func() {
				var a map[string]interface{}
				if e := websocket.JSON.Receive(conn, &a); e != nil {
					served <- e
					return
				}
				if tc.outcome != "timeout" {
					var data interface{}
					if tc.outcome == "ok" {
						data = map[string]interface{}{"message_id": 123}
					}
					served <- websocket.JSON.Send(conn, map[string]interface{}{"echo": a["echo"], "status": tc.outcome, "retcode": tc.code, "data": data})
				} else {
					served <- nil
				}
			}()
			if e := s.RunOnce(context.Background(), time.Now()); e != nil {
				t.Fatal(e)
			}
			if e := <-served; e != nil {
				t.Fatal(e)
			}
			got, e := s.GetOperation(context.Background(), o.ID)
			if e != nil || got.Status != tc.want {
				t.Fatal(got, e)
			}
			if tc.want == "unknown" {
				if _, e = s.RetryOperation(context.Background(), o.ID); e == nil {
					t.Fatal("unknown message retry allowed")
				}
			}
			if tc.want == "succeeded" {
				if e = s.consumeOnce(context.Background(), s.consumers[0]); e != nil {
					t.Fatal(e)
				}
				if !strings.Contains(consumed, "succeeded") || !strings.Contains(consumed, "123") {
					t.Fatal(consumed)
				}
			}
			if tc.outcome == "timeout" && got.Failures != 0 {
				t.Fatal("unknown used definite failure budget")
			}
			var attempts int64
			db.Model(&model.QQBotAttempt{}).Count(&attempts)
			if attempts != 1 {
				t.Fatal(attempts)
			}
		})
	}
}
func TestQQBotFairnessCancelAndRateRecovery(t *testing.T) {
	s, db, mr := qqbotTestService(t)
	low := qqbotSubmit(t, s, "aged", "group_info")
	high := qqbotSubmit(t, s, "high", "group_info")
	db.Model(&model.QQBotOperation{}).Where("id = ?", low.ID).Update("created_at", time.Now().Add(-3*time.Minute))
	db.Model(&model.QQBotOperation{}).Where("id = ?", high.ID).Update("priority", 2)
	claim, e := s.repo.Claim(context.Background(), time.Now())
	if e != nil || claim.ID != low.ID {
		t.Fatal("starvation", claim, e)
	}
	if e = s.Cancel(context.Background(), high.ID); e != nil {
		t.Fatal(e)
	}
	if e = s.RunOnce(context.Background(), time.Now()); e != nil {
		t.Fatal(e)
	}
	got, _ := s.GetOperation(context.Background(), high.ID)
	if got.Status != "cancelled" {
		t.Fatal(got)
	}
	reason, e := s.admit(context.Background(), 8, 100)
	if e != nil || reason != "" {
		t.Fatal(reason, e)
	}
	reason, e = s.admit(context.Background(), 9, 100)
	if e != nil || reason != "rate_limited" {
		t.Fatal(reason, e)
	}
	mr.FastForward(time.Second)
	reason, e = s.admit(context.Background(), 9, 100)
	if e != nil || reason != "" {
		t.Fatal(reason, e)
	}
}

func TestQQBotActualHalfOpenExecutionAndShutdown(t *testing.T) {
	s, db, _ := qqbotTestService(t)
	conn := qqbotDial(t, s)
	ctx := context.Background()
	if e := s.repo.Transaction(ctx, func(tx *gorm.DB) error {
		c, e := s.repo.Circuit(tx, "read")
		if e != nil {
			return e
		}
		c.State = "open"
		c.ProbeAt = time.Now().Add(-time.Second)
		return s.repo.SaveCircuit(tx, &c)
	}); e != nil {
		t.Fatal(e)
	}
	o := qqbotSubmit(t, s, "probe", "group_list")
	done := make(chan error, 1)
	go func() {
		var a map[string]interface{}
		if e := websocket.JSON.Receive(conn, &a); e != nil {
			done <- e
			return
		}
		done <- websocket.JSON.Send(conn, map[string]interface{}{"echo": a["echo"], "status": "ok", "retcode": 0, "data": []interface{}{}})
	}()
	if e := s.RunOnce(ctx, time.Now()); e != nil {
		t.Fatal(e)
	}
	if e := <-done; e != nil {
		t.Fatal(e)
	}
	got, _ := s.GetOperation(ctx, o.ID)
	if got.Status != "succeeded" {
		t.Fatal(got)
	}
	var c model.QQBotCircuit
	db.First(&c, "class = ?", "read")
	if c.State != "closed" || c.Successes != 1 {
		t.Fatal(c)
	}
	pending := make(chan string, 1)
	go func() { _, out := s.client.call(ctx, "get_group_list", nil); pending <- out }()
	var a map[string]interface{}
	if e := websocket.JSON.Receive(conn, &a); e != nil {
		t.Fatal(e)
	}
	s.Close()
	select {
	case out := <-pending:
		if out != "unknown" {
			t.Fatal(out)
		}
	case <-time.After(time.Second):
		t.Fatal("shutdown leaked waiter")
	}
}
func TestQQBotZeroConfigurationAndConsumerPanic(t *testing.T) {
	s, _, _ := qqbotTestService(t)
	ctx := context.Background()
	if e := s.SetSetting(ctx, model.QQBotSetting{Enabled: false, BotID: 0, CIDRs: "127.0.0.1/32", IntervalMS: 100}, nil); e != nil {
		t.Fatal(e)
	}
	got, e := s.Setting(ctx)
	if e != nil || got.BotID != 0 || got.Enabled {
		t.Fatal(got, e)
	}
	if e = s.SetSetting(ctx, model.QQBotSetting{Enabled: false, CIDRs: "127.0.0.1/32", IntervalMS: 0}, nil); e == nil {
		t.Fatal("zero must explicitly violate the security minimum")
	}
	if e := s.RegisterConsumer("event", "panic", func(context.Context, *gorm.DB, string) error { panic("test") }); e != nil {
		t.Fatal(e)
	}
	if e := s.persistEvent(ctx, []byte(`{"self_id":42,"time":1,"post_type":"notice"}`)); e != nil {
		t.Fatal(e)
	}
	if e := s.consumeOnce(ctx, s.consumers[0]); e != nil {
		t.Fatal(e)
	}
	snapshot, e := s.Snapshot(ctx)
	if e != nil || snapshot.Deliveries[0].Status != "pending" || snapshot.Deliveries[0].Attempts != 1 {
		t.Fatal(snapshot, e)
	}
}

func TestQQBotFailureBudgetAndRestrictedRetry(t *testing.T) {
	s, db, mr := qqbotTestService(t)
	conn := qqbotDial(t, s)
	o := qqbotSubmit(t, s, "transient-budget", "group_info")
	for i := 0; i < 3; i++ {
		done := make(chan error, 1)
		go func() {
			var a map[string]interface{}
			if e := websocket.JSON.Receive(conn, &a); e != nil {
				done <- e
				return
			}
			done <- websocket.JSON.Send(conn, map[string]interface{}{"echo": a["echo"], "status": "failed", "retcode": 1500, "data": nil})
		}()
		db.Model(&model.QQBotOperation{}).Where("id = ?", o.ID).Update("available_at", time.Now().Add(-time.Second))
		mr.FastForward(time.Second)
		if e := s.RunOnce(context.Background(), time.Now()); e != nil {
			t.Fatal(e)
		}
		if e := <-done; e != nil {
			t.Fatal(e)
		}
	}
	got, e := s.GetOperation(context.Background(), o.ID)
	if e != nil || got.Status != "failed" || got.Failures != 3 {
		t.Fatal(got, e)
	}
	retry, e := s.RetryOperation(context.Background(), o.ID)
	if e != nil || retry.ID == o.ID {
		t.Fatal(retry, e)
	}
	again, e := s.RetryOperation(context.Background(), o.ID)
	if e != nil || again.ID != retry.ID {
		t.Fatal(again, e)
	}
	db.Model(&model.QQBotOperation{}).Where("id = ?", retry.ID).Updates(map[string]interface{}{"status": "failed", "failures": 3})
	if _, e = s.RetryOperation(context.Background(), retry.ID); e == nil {
		t.Fatal("unbounded manual retry")
	}
	if qqbotValidateData(model.QQBotRequest{Kind: "member_list", GroupID: 8}, json.RawMessage(`[1]`)) {
		t.Fatal("malformed list accepted")
	}
}

func TestQQBotCancelExecutingWritePreservesUnknown(t *testing.T) {
	s, _, _ := qqbotTestService(t)
	conn := qqbotDial(t, s)
	o := qqbotSubmit(t, s, "cancel-in-flight", "group_message")
	done := make(chan error, 1)
	go func() { done <- s.RunOnce(context.Background(), time.Now()) }()
	var a map[string]interface{}
	if e := websocket.JSON.Receive(conn, &a); e != nil {
		t.Fatal(e)
	}
	if e := s.Cancel(context.Background(), o.ID); e != nil {
		t.Fatal(e)
	}
	select {
	case e := <-done:
		if e != nil {
			t.Fatal(e)
		}
	case <-time.After(time.Second):
		t.Fatal("cancel did not end pending wait")
	}
	got, e := s.GetOperation(context.Background(), o.ID)
	if e != nil || got.Status != "unknown" || !got.CancelRequested {
		t.Fatal(got, e)
	}
}

func TestQQBotRiskPauseAndRedisRestartResume(t *testing.T) {
	s, db, mr := qqbotTestService(t)
	conn := qqbotDial(t, s)
	o := qqbotSubmit(t, s, "resume", "group_list")
	ctx := context.Background()
	if e := s.Pause(ctx, "read", true); e != nil {
		t.Fatal(e)
	}
	if e := s.RunOnce(ctx, time.Now()); e != nil {
		t.Fatal(e)
	}
	got, _ := s.GetOperation(ctx, o.ID)
	if got.WaitReason != "risk_paused" || got.Failures != 0 {
		t.Fatal(got)
	}
	if e := s.Pause(ctx, "read", false); e != nil {
		t.Fatal(e)
	}
	mr.Close()
	db.Model(&model.QQBotOperation{}).Where("id = ?", o.ID).Update("available_at", time.Now().Add(-time.Second))
	if e := s.RunOnce(ctx, time.Now()); e != nil {
		t.Fatal(e)
	}
	got, _ = s.GetOperation(ctx, o.ID)
	if got.WaitReason != "redis_unavailable" || got.Failures != 0 {
		t.Fatal(got)
	}
	if e := mr.Restart(); e != nil {
		t.Fatal(e)
	}
	mr.FastForward(time.Second)
	deadline := time.Now().Add(5 * time.Second)
	for {
		if _, err := s.redis.Ping(ctx).Result(); err == nil {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("Redis did not recover")
		}
		time.Sleep(20 * time.Millisecond)
	}
	db.Model(&model.QQBotOperation{}).Where("id = ?", o.ID).Update("available_at", time.Now().Add(-time.Second))
	done := make(chan error, 1)
	go func() {
		var a map[string]interface{}
		if e := websocket.JSON.Receive(conn, &a); e != nil {
			done <- e
			return
		}
		done <- websocket.JSON.Send(conn, map[string]interface{}{"echo": a["echo"], "status": "ok", "retcode": 0, "data": []interface{}{}})
	}()
	if e := s.RunOnce(ctx, time.Now()); e != nil {
		t.Fatal(e)
	}
	if e := <-done; e != nil {
		t.Fatal(e)
	}
	got, _ = s.GetOperation(ctx, o.ID)
	if got.Status != "succeeded" || got.Failures != 0 {
		t.Fatal(got)
	}
}

func TestQQBotPausedClassDoesNotConsumeGlobalBudget(t *testing.T) {
	s, _, _ := qqbotTestService(t)
	_ = qqbotDial(t, s)
	ctx := context.Background()
	qqbotSubmit(t, s, "paused-write", "group_message")
	if e := s.Pause(ctx, "write", true); e != nil {
		t.Fatal(e)
	}
	if e := s.RunOnce(ctx, time.Now()); e != nil {
		t.Fatal(e)
	}
	if reason, e := s.admit(ctx, 8, 100); e != nil || reason != "" {
		t.Fatal("paused class consumed limiter", reason, e)
	}
}
func TestQQBotClosedTypedProtocol(t *testing.T) {
	for _, tc := range []struct {
		request       model.QQBotRequest
		action, class string
	}{
		{model.QQBotRequest{Kind: "group_message", GroupID: 8, Text: "[CQ:at,qq=all]"}, "send_group_msg", "write"},
		{model.QQBotRequest{Kind: "group_info", GroupID: 8}, "get_group_info", "read"},
		{model.QQBotRequest{Kind: "group_list"}, "get_group_list", "read"},
		{model.QQBotRequest{Kind: "member_info", GroupID: 8, UserID: 9}, "get_group_member_info", "read"},
		{model.QQBotRequest{Kind: "member_list", GroupID: 8}, "get_group_member_list", "read"},
		{model.QQBotRequest{Kind: "group_request", Flag: "flag", SubType: "add", Approve: true}, "set_group_add_request", "write"},
		{model.QQBotRequest{Kind: "group_card", GroupID: 8, UserID: 9, Text: "card"}, "set_group_card", "write"},
		{model.QQBotRequest{Kind: "group_kick", GroupID: 8, UserID: 9}, "set_group_kick", "write"},
	} {
		action, class, params, e := qqbotProtocol(tc.request)
		if e != nil || action != tc.action || class != tc.class {
			t.Fatal(action, class, e)
		}
		if tc.request.Kind == "group_message" {
			raw, _ := json.Marshal(params["message"])
			if !strings.Contains(string(raw), `"type":"text"`) {
				t.Fatal("message interpreted CQ instead of literal text")
			}
		}
	}
	if _, _, _, e := qqbotProtocol(model.QQBotRequest{Kind: "arbitrary_action"}); e == nil {
		t.Fatal("arbitrary action allowed")
	}
}

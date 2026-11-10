package service

import (
	"amiya-eden/internal/model"
	"context"
	"golang.org/x/net/websocket"
	"testing"
	"time"
)

func TestStructureAlertTransitionsAndResults(t *testing.T) {
	_, n, db := notificationTestService(t)
	ctx := context.Background()
	now := time.Now().UTC().Truncate(time.Second)
	if err := db.AutoMigrate(&model.QQStructureAlert{}, &model.CorpStructureInfo{}, &model.SystemConfig{}); err != nil {
		t.Fatal(err)
	}
	if err := n.Configure(ctx, "structure", true, []int64{8, 9}); err != nil {
		t.Fatal(err)
	}
	svc := NewQQStructureAlertService(db, n)
	row := model.CorpStructureInfo{StructureID: 1, Name: "Test", FuelExpires: now.Add(time.Hour).Format(time.RFC3339), UpdateAt: now.Unix()}
	if err := db.Create(&row).Error; err != nil {
		t.Fatal(err)
	}
	for range 2 {
		if err := svc.Scan(ctx, now); err != nil {
			t.Fatal(err)
		}
	}
	var count int64
	db.Model(&model.QQBotOperation{}).Count(&count)
	if count != 2 {
		t.Fatal("duplicate active alert", count)
	}
	targets, err := n.Targets(ctx)
	if err != nil {
		t.Fatal(err)
	}
	for _, r := range targets {
		if r.Status != "pending" {
			t.Fatal("enqueue marked delivered")
		}
	}
	// Recovery and re-entry create the next episode, cancelling old queued notices.
	row.FuelExpires = now.Add(30 * 24 * time.Hour).Format(time.RFC3339)
	db.Save(&row)
	if err := svc.Scan(ctx, now); err != nil {
		t.Fatal(err)
	}
	row.FuelExpires = now.Add(2 * time.Hour).Format(time.RFC3339)
	db.Save(&row)
	if err := svc.Scan(ctx, now); err != nil {
		t.Fatal(err)
	}
	db.Model(&model.QQBotOperation{}).Count(&count)
	if count != 4 {
		t.Fatal("new episode missing", count)
	}
	var state model.QQStructureAlert
	if err := db.First(&state, "structure_id = ? AND kind = ?", 1, "fuel").Error; err != nil {
		t.Fatal(err)
	}
	if state.Version != 3 {
		t.Fatal(state.Version)
	}
	// A stale recovery snapshot cannot close the current episode.
	row.FuelExpires = now.Add(30 * 24 * time.Hour).Format(time.RFC3339)
	row.UpdateAt = now.Add(-25 * time.Hour).Unix()
	db.Save(&row)
	if err := svc.Scan(ctx, now); err != nil {
		t.Fatal(err)
	}
	if err := db.First(&state, "structure_id = ? AND kind = ?", 1, "fuel").Error; err != nil {
		t.Fatal(err)
	}
	if !state.Active {
		t.Fatal("stale snapshot recovered alert")
	}
}
func TestStructureTimerTargetAndZeroThreshold(t *testing.T) {
	_, n, db := notificationTestService(t)
	ctx := context.Background()
	now := time.Now().UTC().Truncate(time.Second)
	if err := db.AutoMigrate(&model.QQStructureAlert{}); err != nil {
		t.Fatal(err)
	}
	if err := n.Configure(ctx, "structure", true, []int64{8}); err != nil {
		t.Fatal(err)
	}
	svc := NewQQStructureAlertService(db, n)
	row := model.CorpStructureInfo{StructureID: 2, Name: "Timer"}
	for _, end := range []time.Time{now.Add(time.Hour), now.Add(2 * time.Hour)} {
		if err := svc.evaluate(ctx, row, "timer", 7, end.Format(time.RFC3339), now); err != nil {
			t.Fatal(err)
		}
	}
	var count int64
	db.Model(&model.QQBotOperation{}).Count(&count)
	if count != 2 {
		t.Fatal(count)
	}
	if err := svc.evaluate(ctx, row, "timer", 0, now.Add(time.Hour).Format(time.RFC3339), now); err != nil {
		t.Fatal(err)
	}
	db.Model(&model.QQBotOperation{}).Count(&count)
	if count != 2 {
		t.Fatal("zero threshold sent notification")
	}
}

func TestStructureAlertConfirmationRequiresPlatformAndConsumer(t *testing.T) {
	d, n, db := notificationTestService(t)
	ctx := context.Background()
	now := time.Now()
	if e := db.AutoMigrate(&model.QQStructureAlert{}); e != nil {
		t.Fatal(e)
	}
	if e := n.Configure(ctx, "structure", true, []int64{8}); e != nil {
		t.Fatal(e)
	}
	svc := NewQQStructureAlertService(db, n)
	if e := svc.evaluate(ctx, model.CorpStructureInfo{StructureID: 7, Name: "Platform test"}, "fuel", 7, now.Add(time.Hour).Format(time.RFC3339), now); e != nil {
		t.Fatal(e)
	}
	peer := qqbotDial(t, d)
	done := make(chan error, 1)
	go func() {
		var req struct {
			Echo string `json:"echo"`
		}
		if e := websocket.JSON.Receive(peer, &req); e != nil {
			done <- e
			return
		}
		done <- websocket.JSON.Send(peer, map[string]interface{}{"echo": req.Echo, "status": "ok", "retcode": 0, "data": map[string]int64{"message_id": 42}})
	}()
	if e := d.RunOnce(ctx, time.Now()); e != nil {
		t.Fatal(e)
	}
	if e := <-done; e != nil {
		t.Fatal(e)
	}
	rows, e := n.Targets(ctx)
	if e != nil {
		t.Fatal(e)
	}
	if len(rows) != 1 || rows[0].Status != "pending" {
		t.Fatal("business result bypassed durable consumption", rows)
	}
	if e = d.consumeOnce(ctx, d.consumers[0]); e != nil {
		t.Fatal(e)
	}
	rows, e = n.Targets(ctx)
	if e != nil || rows[0].Status != "confirmed" {
		t.Fatal("platform confirmation missing", rows, e)
	}
}

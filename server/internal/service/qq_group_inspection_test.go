package service

import (
	"amiya-eden/internal/model"
	"context"
	"encoding/json"
	"fmt"
	"gorm.io/gorm"
	"testing"
	"time"
)

func inspectionRound(t *testing.T, g *QQGroupBusinessService, db *gorm.DB, at time.Time, present bool) model.QQGroupInspection {
	t.Helper()
	ctx := context.Background()
	if e := g.startInspection(ctx, 8, at); e != nil {
		t.Fatal(e)
	}
	var run model.QQGroupInspection
	if e := db.Order("created_at desc").First(&run).Error; e != nil {
		t.Fatal(e)
	}
	var op model.QQBotOperation
	db.First(&op, "id = ?", run.OperationID)
	op.Status = "succeeded"
	op.Result = "[]"
	if present {
		op.Result = `[{"user_id":9,"group_id":8}]`
	}
	if e := db.Transaction(func(tx *gorm.DB) error { return g.consumeSnapshot(tx, op) }); e != nil {
		t.Fatal(e)
	}
	return run
}
func TestQQInspectionCheckpointConfirmationAndRecovery(t *testing.T) {
	g, _, db := groupBusinessTest(t)
	seedQQQualification(t, db)
	ctx := context.Background()
	now := time.Now().UTC().Add(-26 * time.Hour)
	rule := model.QQGroupRule{GroupID: 8, InspectionEnabled: true, AutoKick: true, MismatchConfirmations: 3, ObservationHours: 24, CorporationIDs: []int64{300}}
	if e := g.Configure(ctx, rule); e != nil {
		t.Fatal(e)
	}
	for i := 0; i < 3; i++ {
		at := now.Add(time.Duration(i) * 13 * time.Hour)
		run := inspectionRound(t, g, db, at, true)
		// New service instance resumes the frozen checkpoint.
		resumed := NewQQGroupBusinessService(db, g.dispatch)
		for range 2 {
			if e := resumed.judgeMember(ctx, run.ID, 9, at); e != nil {
				t.Fatal(e)
			}
		}
		var m model.QQGroupMember
		db.First(&m)
		if m.Confirmations != i+1 {
			t.Fatal("same round double-counted", m.Confirmations)
		}
	}
	var m model.QQGroupMember
	db.First(&m)
	if m.KickOperationID == "" {
		t.Fatal("no kick intent")
	}
	var op model.QQBotOperation
	db.First(&op, "id = ?", m.KickOperationID)
	// Restored local qualification invalidates the old kick at final admission.
	db.Model(&model.EveCharacter{}).Where("character_id = ?", 100).Update("corporation_id", 300)
	if e := db.Transaction(func(tx *gorm.DB) error {
		valid, e := g.validKick(tx, op)
		if valid {
			t.Fatal("restored member kicked")
		}
		return e
	}); e != nil {
		t.Fatal(e)
	}
	m = model.QQGroupMember{}
	db.First(&m)
	if m.Confirmations != 0 || m.MismatchSince != nil {
		t.Fatalf("restoration did not reset chain: %+v", m)
	}
}
func TestQQInspectionProtectsNewJoinAndRecordMode(t *testing.T) {
	g, _, db := groupBusinessTest(t)
	seedQQQualification(t, db)
	ctx := context.Background()
	now := time.Now()
	rule := model.QQGroupRule{GroupID: 8, InspectionEnabled: true, MismatchConfirmations: 1, ObservationHours: 0, CorporationIDs: []int64{300}}
	if e := g.Configure(ctx, rule); e != nil {
		t.Fatal(e)
	}
	// Freeze a snapshot omitting a member that joins while the request was in flight.
	if e := db.Create(&model.QQGroupMember{GroupID: 8, QQ: 9, Present: false}).Error; e != nil {
		t.Fatal(e)
	}
	run := inspectionRound(t, g, db, now.Add(-time.Minute), false)
	event := fmt.Sprintf(`{"post_type":"notice","notice_type":"group_increase","group_id":8,"user_id":9,"time":%d}`, now.Unix())
	if e := db.Transaction(func(tx *gorm.DB) error { return g.ConsumeEvent(ctx, tx, event) }); e != nil {
		t.Fatal(e)
	}
	if e := g.judgeMember(ctx, run.ID, 9, now); e != nil {
		t.Fatal(e)
	}
	var m model.QQGroupMember
	db.First(&m)
	if !m.Present {
		t.Fatal("old snapshot marked new member absent")
	}
	// A later actual mismatch remains record-only.
	run = inspectionRound(t, g, db, now.Add(time.Hour), true)
	if e := g.judgeMember(ctx, run.ID, 9, now.Add(time.Hour)); e != nil {
		t.Fatal(e)
	}
	db.First(&m)
	if m.Confirmations != 1 || m.KickOperationID != "" {
		t.Fatal("record-only kicked", m)
	}
	rule.AutoKick = true
	if e := g.Configure(ctx, rule); e != nil {
		t.Fatal(e)
	}
	run = inspectionRound(t, g, db, now.Add(2*time.Hour), true)
	if e := g.judgeMember(ctx, run.ID, 9, now.Add(2*time.Hour)); e != nil {
		t.Fatal(e)
	}
	db.First(&m)
	var op model.QQBotOperation
	db.First(&op, "id = ?", m.KickOperationID)
	rule.AutoKick = false
	if e := g.Configure(ctx, rule); e != nil {
		t.Fatal(e)
	}
	if e := db.Transaction(func(tx *gorm.DB) error {
		ok, e := g.validKick(tx, op)
		if ok {
			t.Fatal("old kick survived record-only")
		}
		return e
	}); e != nil {
		t.Fatal(e)
	}
	raw, _ := json.Marshal(op)
	if e := db.Transaction(func(tx *gorm.DB) error { return g.ConsumeResult(ctx, tx, string(raw)) }); e != nil {
		t.Fatal(e)
	}
}

func TestQQInspectionUnknownResetsAndSameRuleIsIdempotent(t *testing.T) {
	g, _, db := groupBusinessTest(t)
	seedQQQualification(t, db)
	ctx := context.Background()
	now := time.Now().Add(-time.Hour)
	rule := model.QQGroupRule{GroupID: 8, InspectionEnabled: true, MismatchConfirmations: 3, ObservationHours: 24, CorporationIDs: []int64{300}}
	if e := g.Configure(ctx, rule); e != nil {
		t.Fatal(e)
	}
	run := inspectionRound(t, g, db, now, true)
	if e := g.judgeMember(ctx, run.ID, 9, now); e != nil {
		t.Fatal(e)
	}
	if e := g.Configure(ctx, rule); e != nil {
		t.Fatal(e)
	}
	var m model.QQGroupMember
	db.First(&m)
	if m.Confirmations != 1 {
		t.Fatal("identical config reset confirmations")
	}
	db.Model(&model.User{}).Where("id = ?", 1).Update("primary_character_id", 0)
	run = inspectionRound(t, g, db, now.Add(30*time.Minute), true)
	if e := g.judgeMember(ctx, run.ID, 9, now.Add(30*time.Minute)); e != nil {
		t.Fatal(e)
	}
	m = model.QQGroupMember{}
	db.First(&m)
	if m.Confirmations != 0 || m.MismatchSince != nil {
		t.Fatal("insufficient data accumulated mismatch")
	}
}

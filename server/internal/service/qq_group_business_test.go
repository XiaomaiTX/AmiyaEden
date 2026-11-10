package service

import (
	"amiya-eden/internal/model"
	"context"
	"errors"
	"fmt"
	"gorm.io/gorm"
	"testing"
	"time"
)

func groupBusinessTest(t *testing.T) (*QQGroupBusinessService, *QQBotService, *gorm.DB) {
	t.Helper()
	d, _, db := notificationTestService(t)
	if e := db.AutoMigrate(&model.QQGroupRule{}, &model.QQGroupApplication{}, &model.QQGroupMember{}, &model.QQGroupAppliedEvent{}, &model.QQGroupInspection{}, &model.QQGroupInspectionItem{}, &model.EveEntityNameCache{}, &model.User{}, &model.EveCharacter{}, &model.UserRole{}); e != nil {
		t.Fatal(e)
	}
	g := NewQQGroupBusinessService(db, d)
	d.groups = g
	return g, d, db
}
func seedQQQualification(t *testing.T, db *gorm.DB) {
	t.Helper()
	for _, v := range []interface{}{&model.User{BaseModel: model.BaseModel{ID: 1}, QQ: "9", Status: 1, PrimaryCharacterID: 100}, &model.EveCharacter{CharacterID: 100, UserID: 1, CharacterName: "Pilot", CorporationID: 200}, &model.UserRole{UserID: 1, RoleCode: model.RoleFC}} {
		if e := db.Create(v).Error; e != nil {
			t.Fatal(e)
		}
	}
}
func TestQQApplicationANDAndDedup(t *testing.T) {
	g, _, db := groupBusinessTest(t)
	seedQQQualification(t, db)
	ctx := context.Background()
	rule := model.QQGroupRule{GroupID: 8, MismatchConfirmations: 3, ObservationHours: 24, AdmissionEnabled: true, CorporationIDs: []int64{200}, Roles: []string{model.RoleFC}}
	if e := g.Configure(ctx, rule); e != nil {
		t.Fatal(e)
	}
	event := fmt.Sprintf(`{"post_type":"request","request_type":"group","sub_type":"add","group_id":8,"user_id":9,"flag":"stable","time":%d}`, time.Now().Unix())
	for range 2 {
		if e := db.Transaction(func(tx *gorm.DB) error { return g.ConsumeEvent(ctx, tx, event) }); e != nil {
			t.Fatal(e)
		}
	}
	var count int64
	db.Model(&model.QQBotOperation{}).Count(&count)
	if count != 1 {
		t.Fatal("duplicate application action", count)
	}
	var op model.QQBotOperation
	db.First(&op)
	if e := db.Transaction(func(tx *gorm.DB) error {
		valid, e := g.valid(tx, op)
		if !valid {
			t.Fatal("eligible invalid")
		}
		return e
	}); e != nil {
		t.Fatal(e)
	}
	// Removing one of the two required dimensions invalidates approval.
	db.Where("user_id = ?", 1).Delete(&model.UserRole{})
	if e := db.Transaction(func(tx *gorm.DB) error {
		valid, e := g.valid(tx, op)
		if valid {
			t.Fatal("AND was OR")
		}
		return e
	}); e != nil {
		t.Fatal(e)
	}
	rule.AutoReject = true
	if e := g.Configure(ctx, rule); e != nil {
		t.Fatal(e)
	}
	if e := db.Transaction(func(tx *gorm.DB) error { return g.ConsumeEvent(ctx, tx, event) }); e != nil {
		t.Fatal(e)
	}
	db.Model(&model.QQBotOperation{}).Count(&count)
	if count != 1 {
		t.Fatal("opposite action queued")
	}
}
func TestQQApplicationMissingAndDefaultReject(t *testing.T) {
	g, _, db := groupBusinessTest(t)
	ctx := context.Background()
	rule := model.QQGroupRule{GroupID: 8, MismatchConfirmations: 3, ObservationHours: 24, AdmissionEnabled: true}
	if e := g.Configure(ctx, rule); e != nil {
		t.Fatal(e)
	}
	event := `{"post_type":"request","request_type":"group","sub_type":"add","group_id":8,"user_id":9,"flag":"missing"}`
	if e := db.Transaction(func(tx *gorm.DB) error { return g.ConsumeEvent(ctx, tx, event) }); e != nil {
		t.Fatal(e)
	}
	apps, e := g.Applications(ctx)
	if e != nil || len(apps) != 1 || apps[0].Status != "waiting" {
		t.Fatal(apps, e)
	}
	seedQQQualification(t, db)
	if e = g.Recheck(ctx, time.Now().Add(10*time.Minute)); e != nil {
		t.Fatal(e)
	}
	var count int64
	db.Model(&model.QQBotOperation{}).Count(&count)
	if count != 0 {
		t.Fatal("empty rule approved")
	}
	rule.CorporationIDs = []int64{300}
	if e = g.Configure(ctx, rule); e != nil {
		t.Fatal(e)
	}
	if e = g.Recheck(ctx, time.Now().Add(20*time.Minute)); e != nil {
		t.Fatal(e)
	}
	db.Model(&model.QQBotOperation{}).Count(&count)
	if count != 0 {
		t.Fatal("default rejected")
	}
}

func TestQQCardSharedEntryDedupeAndDisable(t *testing.T) {
	g, _, db := groupBusinessTest(t)
	seedQQQualification(t, db)
	ctx := context.Background()
	rule := model.QQGroupRule{GroupID: 8, MismatchConfirmations: 3, ObservationHours: 24, CorporationIDs: []int64{200}, CardEnabled: true, CardTemplate: "{character_name} {corporation_name}"}
	if e := db.Create(&model.EveEntityNameCache{EntityID: 200, EntityType: model.EntityNameTypeCorporation, Name: "Real Corp"}).Error; e != nil {
		t.Fatal(e)
	}
	if e := g.Configure(ctx, rule); e != nil {
		t.Fatal(e)
	}
	event := fmt.Sprintf(`{"post_type":"notice","notice_type":"group_increase","group_id":8,"user_id":9,"time":%d}`, time.Now().Unix())
	for range 2 {
		if e := db.Transaction(func(tx *gorm.DB) error { return g.ConsumeEvent(ctx, tx, event) }); e != nil {
			t.Fatal(e)
		}
	}
	members, e := g.Members(ctx)
	if e != nil || len(members) != 1 || members[0].CardTarget != "Pilot Real Corp" {
		t.Fatal(members, e)
	}
	var op model.QQBotOperation
	db.First(&op, "id = ?", members[0].CardOperationID)
	if e = db.Transaction(func(tx *gorm.DB) error {
		ok, e := g.validCard(tx, op)
		if !ok {
			t.Fatal("card not admitted")
		}
		return e
	}); e != nil {
		t.Fatal(e)
	}
	if e = db.Transaction(func(tx *gorm.DB) error {
		ok, e := g.validCard(tx, op)
		if ok || !errors.Is(e, errQQBusinessRecheck) {
			t.Fatal("member frequency not enforced", e)
		}
		return nil
	}); e != nil {
		t.Fatal(e)
	}
	rule.CardTemplate = "{nickname} {character_name}"
	if e = g.Configure(ctx, rule); e != nil {
		t.Fatal(e)
	}
	if e = db.Transaction(func(tx *gorm.DB) error {
		m, e := g.repo.Member(tx, 8, 9)
		if e != nil {
			return e
		}
		r, e := g.repo.Rule(tx, 8)
		if e != nil {
			return e
		}
		if e = g.planCard(ctx, tx, r, &m); e != nil {
			return e
		}
		return g.repo.SaveMember(tx, &m)
	}); e != nil {
		t.Fatal(e)
	}
	var count int64
	db.Model(&model.QQBotOperation{}).Count(&count)
	if count != 2 {
		t.Fatal(count)
	}
	rule.CardEnabled = false
	if e = g.Configure(ctx, rule); e != nil {
		t.Fatal(e)
	}
	var pending int64
	db.Model(&model.QQBotOperation{}).Where("cancel_requested = ?", false).Count(&pending)
	if pending != 0 {
		t.Fatal("disable left valid cards")
	}
}

package service

import (
	"amiya-eden/internal/model"
	"amiya-eden/internal/repository"
	"encoding/json"
	"testing"
	"time"
)

func TestQQGovernanceServiceGroupRequestCreatesOneApprovalTask(t *testing.T) {
	db := newServiceTestDB(t, "qq_governance",
		&model.User{}, &model.EveCharacter{}, &model.UserRole{},
		&model.QQGroupGovernancePolicy{}, &model.QQGovernanceEvent{},
		&model.QQGroupMemberState{}, &model.QQGovernanceReview{},
		&model.QQGovernanceActionTask{}, &model.QQGovernanceActionLog{},
	)
	if err := db.Create(&model.User{BaseModel: model.BaseModel{ID: 9}, QQ: "123456", Nickname: "Amiya", PrimaryCharacterID: 42, Role: model.RoleGuest}).Error; err != nil {
		t.Fatalf("create user: %v", err)
	}
	if err := db.Create(&model.EveCharacter{CharacterID: 42, CharacterName: "Amiya Prime", UserID: 9, CorporationID: 1001}).Error; err != nil {
		t.Fatalf("create character: %v", err)
	}
	if err := db.Create(&model.UserRole{UserID: 9, RoleCode: model.RoleAdmin}).Error; err != nil {
		t.Fatalf("create role: %v", err)
	}
	corps, _ := json.Marshal([]int64{1001})
	roles, _ := json.Marshal([]string{model.RoleAdmin})
	if err := db.Create(&model.QQGroupGovernancePolicy{GroupID: 778899, Enabled: true, AllowedCorporationIDsJSON: string(corps), AllowedRoleCodesJSON: string(roles), CardTemplate: "{nickname}-{primary_character_name}"}).Error; err != nil {
		t.Fatalf("create policy: %v", err)
	}

	svc := NewQQGovernanceServiceWithRepository(repository.NewQQGovernanceRepositoryWithDB(db))
	now := time.Date(2026, 7, 11, 12, 0, 0, 0, time.UTC)
	svc.now = func() time.Time { return now }
	event := QQGovernanceInboundEvent{EventKey: "request/group_add:778899:123456:flag", EventType: "request/group_add", GroupID: 778899, QQ: 123456, RequestFlag: "flag"}
	if err := svc.HandleOneBotEvent(t.Context(), event); err != nil {
		t.Fatalf("handle event: %v", err)
	}
	if err := svc.HandleOneBotEvent(t.Context(), event); err != nil {
		t.Fatalf("handle duplicate event: %v", err)
	}

	var tasks []model.QQGovernanceActionTask
	if err := db.Find(&tasks).Error; err != nil {
		t.Fatalf("list tasks: %v", err)
	}
	if len(tasks) != 1 {
		t.Fatalf("task count = %d, want 1", len(tasks))
	}
	if tasks[0].ActionType != model.QQGovernanceActionApprove || tasks[0].Priority != 10 || tasks[0].TargetVersion != 1 {
		t.Fatalf("approval task = %#v", tasks[0])
	}
	var review model.QQGovernanceReview
	if err := db.First(&review).Error; err != nil {
		t.Fatalf("load review: %v", err)
	}
	if review.Decision != model.QQGovernanceReviewMatched || review.UserID != 9 {
		t.Fatalf("review = %#v", review)
	}
}

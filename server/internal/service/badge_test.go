package service

import (
	"amiya-eden/global"
	"amiya-eden/internal/model"
	"amiya-eden/internal/utils"
	"fmt"
	"reflect"
	"testing"
	"time"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestBadgeServiceGetBadgeCountsReturnsOnlyPermittedNonZeroFields(t *testing.T) {
	db := newBadgeServiceTestDB(t)
	originalDB := global.DB
	global.DB = db
	defer func() { global.DB = originalDB }()

	user := model.User{BaseModel: model.BaseModel{ID: 910001}, Nickname: "Pilot One", QQ: "12345"}
	if err := db.Create(&user).Error; err != nil {
		t.Fatalf("create user: %v", err)
	}

	welfare := model.Welfare{
		Name:      "Starter Pack",
		DistMode:  model.WelfareDistModePerUser,
		Status:    model.WelfareStatusActive,
		CreatedBy: user.ID,
	}
	if err := db.Create(&welfare).Error; err != nil {
		t.Fatalf("create welfare: %v", err)
	}

	requestedUserID := uint(999)
	if err := db.Create(&model.WelfareApplication{
		WelfareID:     welfare.ID,
		UserID:        &requestedUserID,
		CharacterID:   7001,
		CharacterName: "Other Pilot",
		Status:        model.WelfareAppStatusRequested,
	}).Error; err != nil {
		t.Fatalf("create welfare application: %v", err)
	}

	if err := db.Create(&model.SrpApplication{
		UserID:            user.ID,
		CharacterID:       5001,
		CharacterName:     "Pilot One",
		KillmailID:        8001,
		ShipTypeID:        100,
		SolarSystemID:     30000142,
		KillmailTime:      time.Unix(1_700_000_000, 0).UTC(),
		ReviewStatus:      model.SrpReviewSubmitted,
		PayoutStatus:      model.SrpPayoutNotPaid,
		FinalAmount:       10,
		RecommendedAmount: 10,
	}).Error; err != nil {
		t.Fatalf("create submitted srp application: %v", err)
	}
	if err := db.Create(&model.SrpApplication{
		UserID:            user.ID,
		CharacterID:       5002,
		CharacterName:     "Pilot One Alt",
		KillmailID:        8002,
		ShipTypeID:        101,
		SolarSystemID:     30000142,
		KillmailTime:      time.Unix(1_700_000_100, 0).UTC(),
		ReviewStatus:      model.SrpReviewApproved,
		PayoutStatus:      model.SrpPayoutNotPaid,
		FinalAmount:       20,
		RecommendedAmount: 20,
	}).Error; err != nil {
		t.Fatalf("create approved srp application: %v", err)
	}

	if err := db.Create(&model.ShopOrder{
		OrderNo:           "ORDER-1",
		UserID:            user.ID,
		MainCharacterName: "Pilot One",
		Nickname:          "Pilot One",
		ProductID:         1,
		ProductName:       "Item",
		ProductType:       model.ProductTypeNormal,
		Quantity:          1,
		UnitPrice:         1,
		TotalPrice:        1,
		Status:            model.OrderStatusRequested,
	}).Error; err != nil {
		t.Fatalf("create shop order: %v", err)
	}

	if err := db.Create(&model.MentorMenteeRelationship{
		MenteeUserID:                    user.ID + 100,
		MenteePrimaryCharacterIDAtStart: 7002,
		MentorUserID:                    user.ID,
		Status:                          model.MentorRelationStatusPending,
		AppliedAt:                       time.Unix(1_700_000_300, 0).UTC(),
	}).Error; err != nil {
		t.Fatalf("create mentor relationship: %v", err)
	}

	category := model.TicketCategory{
		Name:      "账号问题",
		NameEN:    "Account Issues",
		Enabled:   true,
		SortOrder: 0,
	}
	if err := db.Create(&category).Error; err != nil {
		t.Fatalf("create ticket category: %v", err)
	}
	if err := db.Create(&model.Ticket{
		UserID:      user.ID,
		CategoryID:  category.ID,
		Title:       "Pending Ticket",
		Description: "Pending ticket for badge count",
		Status:      model.TicketStatusPending,
		Priority:    model.TicketPriorityMedium,
	}).Error; err != nil {
		t.Fatalf("create pending ticket: %v", err)
	}
	if err := db.Create(&model.Ticket{
		UserID:      user.ID + 1,
		CategoryID:  category.ID,
		Title:       "Owned In Progress",
		Description: "Assigned ticket for badge count",
		Status:      model.TicketStatusInProgress,
		Priority:    model.TicketPriorityMedium,
		HandledBy:   &user.ID,
	}).Error; err != nil {
		t.Fatalf("create in progress ticket: %v", err)
	}
	otherHandlerID := user.ID + 500
	if err := db.Create(&model.Ticket{
		UserID:      user.ID + 2,
		CategoryID:  category.ID,
		Title:       "Other Handler",
		Description: "Should not count for current admin",
		Status:      model.TicketStatusInProgress,
		Priority:    model.TicketPriorityMedium,
		HandledBy:   &otherHandlerID,
	}).Error; err != nil {
		t.Fatalf("create other handler ticket: %v", err)
	}

	svc := NewBadgeService()
	tests := []struct {
		name  string
		roles []string
		want  BadgeCounts
	}{
		{
			name:  "ordinary user sees no badge counts before welfare cache is warmed",
			roles: []string{model.RoleUser},
			want:  BadgeCounts{},
		},
		{
			name:  "srp reviewer sees srp pending count",
			roles: []string{model.RoleSRP},
			want: BadgeCounts{
				BadgeCountSrpPending: 2,
			},
		},
		{
			name:  "welfare officer sees welfare pending count but not shop order count",
			roles: []string{model.RoleWelfare},
			want: BadgeCounts{
				BadgeCountWelfarePending: 1,
			},
		},
		{
			name:  "shop order officer sees shop order pending count only",
			roles: []string{model.RoleShopOrder},
			want: BadgeCounts{
				BadgeCountOrderPending: 1,
			},
		},
		{
			name:  "admin sees every non zero count",
			roles: []string{model.RoleAdmin},
			want: BadgeCounts{
				BadgeCountSrpPending:      2,
				BadgeCountWelfarePending:  1,
				BadgeCountOrderPending:    1,
				BadgeCountTicketAttention: 2,
			},
		},
		{
			name:  "mentor sees pending mentee application count",
			roles: []string{model.RoleMentor},
			want: BadgeCounts{
				BadgeCountMentorPendingApplications: 1,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := svc.GetBadgeCounts(user.ID, tt.roles)
			if err != nil {
				t.Fatalf("GetBadgeCounts() error = %v", err)
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("GetBadgeCounts() = %#v, want %#v", got, tt.want)
			}
		})
	}
}

func TestBadgeServiceGetBadgeCountsUsesCachedEligibleWelfareCount(t *testing.T) {
	db := newBadgeServiceTestDB(t)
	originalDB := global.DB
	global.DB = db
	defer func() { global.DB = originalDB }()

	user := model.User{BaseModel: model.BaseModel{ID: 910002}, Nickname: "Pilot Two", QQ: "67890"}
	if err := db.Create(&user).Error; err != nil {
		t.Fatalf("create user: %v", err)
	}

	characters := []model.EveCharacter{
		{CharacterID: 9001, CharacterName: "Alpha", UserID: user.ID},
		{CharacterID: 9002, CharacterName: "Beta", UserID: user.ID},
	}
	if err := db.Create(&characters).Error; err != nil {
		t.Fatalf("create characters: %v", err)
	}

	if err := db.Create(&model.Welfare{
		Name:      "Per Character Pack",
		DistMode:  model.WelfareDistModePerCharacter,
		Status:    model.WelfareStatusActive,
		CreatedBy: user.ID,
	}).Error; err != nil {
		t.Fatalf("create welfare: %v", err)
	}

	if _, err := NewWelfareService().GetEligibleWelfares(user.ID); err != nil {
		t.Fatalf("warm welfare badge cache: %v", err)
	}

	svc := NewBadgeService()
	got, err := svc.GetBadgeCounts(user.ID, []string{model.RoleUser})
	if err != nil {
		t.Fatalf("GetBadgeCounts() error = %v", err)
	}

	want := BadgeCounts{BadgeCountWelfareEligible: 1}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("GetBadgeCounts() = %#v, want %#v", got, want)
	}
}

func TestBadgeServiceGetBadgeCountsOmitsZeroCounts(t *testing.T) {
	db := newBadgeServiceTestDB(t)
	originalDB := global.DB
	global.DB = db
	defer func() { global.DB = originalDB }()

	user := model.User{BaseModel: model.BaseModel{ID: 910003}, Nickname: "Pilot Zero", QQ: "11111"}
	if err := db.Create(&user).Error; err != nil {
		t.Fatalf("create user: %v", err)
	}

	svc := NewBadgeService()
	got, err := svc.GetBadgeCounts(user.ID, []string{model.RoleUser})
	if err != nil {
		t.Fatalf("GetBadgeCounts() error = %v", err)
	}
	if len(got) != 0 {
		t.Fatalf("expected zero counts to be omitted, got %#v", got)
	}
}

func TestBadgeServiceGetBadgeCountsCountsPendingMentorApplicationsForCurrentMentor(t *testing.T) {
	db := newBadgeServiceTestDB(t)
	originalDB := global.DB
	global.DB = db
	defer func() { global.DB = originalDB }()

	mentor := model.User{BaseModel: model.BaseModel{ID: 910004}, Nickname: "Mentor One", QQ: "12345"}
	otherMentor := model.User{BaseModel: model.BaseModel{ID: 910005}, Nickname: "Mentor Two", QQ: "67890"}
	mentee := model.User{BaseModel: model.BaseModel{ID: 910006}, Nickname: "Mentee One", QQ: "00000"}
	if err := db.Create(&mentor).Error; err != nil {
		t.Fatalf("create mentor: %v", err)
	}
	if err := db.Create(&otherMentor).Error; err != nil {
		t.Fatalf("create other mentor: %v", err)
	}
	if err := db.Create(&mentee).Error; err != nil {
		t.Fatalf("create mentee: %v", err)
	}

	rows := []model.MentorMenteeRelationship{
		{
			MenteeUserID:                    mentee.ID,
			MenteePrimaryCharacterIDAtStart: 7001,
			MentorUserID:                    mentor.ID,
			Status:                          model.MentorRelationStatusPending,
			AppliedAt:                       time.Unix(1_700_000_000, 0).UTC(),
		},
		{
			MenteeUserID:                    mentee.ID + 1,
			MenteePrimaryCharacterIDAtStart: 7002,
			MentorUserID:                    mentor.ID,
			Status:                          model.MentorRelationStatusActive,
			AppliedAt:                       time.Unix(1_700_000_100, 0).UTC(),
		},
		{
			MenteeUserID:                    mentee.ID + 2,
			MenteePrimaryCharacterIDAtStart: 7003,
			MentorUserID:                    otherMentor.ID,
			Status:                          model.MentorRelationStatusPending,
			AppliedAt:                       time.Unix(1_700_000_200, 0).UTC(),
		},
	}
	if err := db.Create(&rows).Error; err != nil {
		t.Fatalf("create mentor relationships: %v", err)
	}

	svc := NewBadgeService()
	got, err := svc.GetBadgeCounts(mentor.ID, []string{model.RoleMentor})
	if err != nil {
		t.Fatalf("GetBadgeCounts() error = %v", err)
	}

	if got[BadgeCountMentorPendingApplications] != 1 {
		t.Fatalf("expected mentor pending applications badge count to be 1, got %#v", got)
	}
}

func TestBadgeServiceGetBadgeCountsIncludesCorporationStructuresAttentionForAdmin(t *testing.T) {
	db := newBadgeServiceTestDB(t)
	originalDB := global.DB
	global.DB = db
	utils.InvalidateAllowCorporationsCache()
	defer func() {
		global.DB = originalDB
		utils.InvalidateAllowCorporationsCache()
	}()

	admin := model.User{BaseModel: model.BaseModel{ID: 910007}, Nickname: "Admin One", QQ: "99999"}
	if err := db.Create(&admin).Error; err != nil {
		t.Fatalf("create admin: %v", err)
	}
	if err := db.Create(&model.UserRole{UserID: admin.ID, RoleCode: model.RoleAdmin}).Error; err != nil {
		t.Fatalf("create admin role: %v", err)
	}
	if err := db.Create(&model.EveCharacter{
		CharacterID:   9901,
		CharacterName: "Admin Character",
		UserID:        admin.ID,
		CorporationID: 9001,
	}).Error; err != nil {
		t.Fatalf("create admin character: %v", err)
	}
	if err := db.Create(&model.SystemConfig{
		Key:   model.SysConfigAllowCorporations,
		Value: "[9001]",
	}).Error; err != nil {
		t.Fatalf("create allow corps config: %v", err)
	}
	if err := db.Create(&model.SystemConfig{
		Key:   model.SysConfigDashboardCorpStructuresFuelNoticeThresholdDays,
		Value: "2",
	}).Error; err != nil {
		t.Fatalf("create fuel threshold config: %v", err)
	}
	if err := db.Create(&model.SystemConfig{
		Key:   model.SysConfigDashboardCorpStructuresTimerNoticeThresholdDays,
		Value: "2",
	}).Error; err != nil {
		t.Fatalf("create timer threshold config: %v", err)
	}

	now := time.Now().UTC()
	rows := []model.CorpStructureInfo{
		{
			CorporationID: 9001,
			StructureID:   1,
			FuelExpires:   now.Add(6 * time.Hour).Format(time.RFC3339),
		},
		{
			CorporationID: 9001,
			StructureID:   2,
			StateTimerEnd: now.Add(8 * time.Hour).Format(time.RFC3339),
		},
		{
			CorporationID: 9001,
			StructureID:   3,
			FuelExpires:   now.Add(9 * 24 * time.Hour).Format(time.RFC3339),
			StateTimerEnd: now.Add(9 * 24 * time.Hour).Format(time.RFC3339),
		},
	}
	if err := db.Create(&rows).Error; err != nil {
		t.Fatalf("create corporation structures: %v", err)
	}

	svc := NewBadgeService()
	got, err := svc.GetBadgeCounts(admin.ID, []string{model.RoleAdmin})
	if err != nil {
		t.Fatalf("GetBadgeCounts() error = %v", err)
	}
	if got[BadgeCountCorporationStructuresAttention] != 2 {
		t.Fatalf(
			"expected corporation structures attention badge count 2, got %#v",
			got,
		)
	}
}

func newBadgeServiceTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	dsn := fmt.Sprintf("file:badge_service_test_%d?mode=memory&cache=shared", time.Now().UnixNano())
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(
		&model.User{},
		&model.UserRole{},
		&model.SystemConfig{},
		&model.EveCharacter{},
		&model.EveCharacterCorpRole{},
		&model.MentorMenteeRelationship{},
		&model.Welfare{},
		&model.WelfareSkillPlan{},
		&model.WelfareApplication{},
		&model.ShopOrder{},
		&model.SrpApplication{},
		&model.CorpStructureInfo{},
		&model.TicketCategory{},
		&model.Ticket{},
	); err != nil {
		t.Fatalf("auto migrate: %v", err)
	}
	return db
}

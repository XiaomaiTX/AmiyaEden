package service

import (
	"amiya-eden/global"
	"amiya-eden/internal/model"
	"amiya-eden/internal/repository"
	"amiya-eden/pkg/eve/esi"
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

var adminDeliveryReviewerRoles = []string{model.RoleAdmin}

var welfareOfficerReviewerRoles = []string{model.RoleWelfare}

func TestValidateReviewTransition(t *testing.T) {
	tests := []struct {
		name          string
		currentStatus string
		action        string
		wantStatus    string
		wantErr       bool
	}{
		{
			name:          "deliver from requested succeeds",
			currentStatus: model.WelfareAppStatusRequested,
			action:        "deliver",
			wantStatus:    model.WelfareAppStatusDelivered,
		},
		{
			name:          "reject from requested succeeds",
			currentStatus: model.WelfareAppStatusRequested,
			action:        "reject",
			wantStatus:    model.WelfareAppStatusRejected,
		},
		{
			name:          "deliver from delivered is rejected",
			currentStatus: model.WelfareAppStatusDelivered,
			action:        "deliver",
			wantErr:       true,
		},
		{
			name:          "reject from delivered is rejected",
			currentStatus: model.WelfareAppStatusDelivered,
			action:        "reject",
			wantErr:       true,
		},
		{
			name:          "deliver from rejected is rejected",
			currentStatus: model.WelfareAppStatusRejected,
			action:        "deliver",
			wantErr:       true,
		},
		{
			name:          "reject from rejected is rejected",
			currentStatus: model.WelfareAppStatusRejected,
			action:        "reject",
			wantErr:       true,
		},
		{
			name:          "invalid action is rejected",
			currentStatus: model.WelfareAppStatusRequested,
			action:        "approve",
			wantErr:       true,
		},
		{
			name:          "empty action is rejected",
			currentStatus: model.WelfareAppStatusRequested,
			action:        "",
			wantErr:       true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotStatus, err := validateReviewTransition(tt.currentStatus, tt.action)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error, got status=%q", gotStatus)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if gotStatus != tt.wantStatus {
				t.Fatalf("got status=%q, want %q", gotStatus, tt.wantStatus)
			}
		})
	}
}

func TestCharacterAgeTooOld(t *testing.T) {
	now := time.Date(2026, 3, 23, 0, 0, 0, 0, time.UTC)

	bday := func(y, m, d int) *time.Time {
		t := time.Date(y, time.Month(m), d, 0, 0, 0, 0, time.UTC)
		return &t
	}

	tests := []struct {
		name     string
		birthday *time.Time
		months   int
		want     bool
	}{
		{
			name:     "nil birthday is not too old",
			birthday: nil,
			months:   6,
			want:     false,
		},
		{
			name:     "character born 3 months ago with 6 month limit is ok",
			birthday: bday(2025, 12, 23),
			months:   6,
			want:     false,
		},
		{
			name:     "character born exactly at limit is not too old",
			birthday: bday(2025, 9, 23),
			months:   6,
			want:     false,
		},
		{
			name:     "character born 7 months ago with 6 month limit is too old",
			birthday: bday(2025, 8, 22),
			months:   6,
			want:     true,
		},
		{
			name:     "character born 2 years ago with 12 month limit is too old",
			birthday: bday(2024, 3, 1),
			months:   12,
			want:     true,
		},
		{
			name:     "character born 11 months ago with 12 month limit is ok",
			birthday: bday(2025, 4, 24),
			months:   12,
			want:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := characterAgeTooOld(tt.birthday, tt.months, now)
			if got != tt.want {
				t.Fatalf("characterAgeTooOld() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestAnyCharacterTooOld(t *testing.T) {
	now := time.Date(2026, 3, 23, 0, 0, 0, 0, time.UTC)

	bday := func(y, m, d int) *time.Time {
		t := time.Date(y, time.Month(m), d, 0, 0, 0, 0, time.UTC)
		return &t
	}

	young := model.EveCharacter{Birthday: bday(2026, 1, 1)}
	old := model.EveCharacter{Birthday: bday(2024, 1, 1)}
	noBday := model.EveCharacter{Birthday: nil}

	tests := []struct {
		name       string
		characters []model.EveCharacter
		months     int
		want       bool
	}{
		{
			name:       "all young characters pass",
			characters: []model.EveCharacter{young, noBday},
			months:     6,
			want:       false,
		},
		{
			name:       "one old character fails the check",
			characters: []model.EveCharacter{young, old},
			months:     12,
			want:       true,
		},
		{
			name:       "empty character list passes",
			characters: []model.EveCharacter{},
			months:     6,
			want:       false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := anyCharacterTooOld(tt.characters, tt.months, now)
			if got != tt.want {
				t.Fatalf("anyCharacterTooOld() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestWelfareAgeRestrictionFailed(t *testing.T) {
	now := time.Date(2026, 3, 23, 0, 0, 0, 0, time.UTC)

	bday := func(y, m, d int) *time.Time {
		t := time.Date(y, time.Month(m), d, 0, 0, 0, 0, time.UTC)
		return &t
	}

	tests := []struct {
		name       string
		characters []model.EveCharacter
		maxMonths  *int
		want       bool
	}{
		{
			name:       "nil limit never blocks",
			characters: []model.EveCharacter{{Birthday: bday(2024, 1, 1)}},
			maxMonths:  nil,
			want:       false,
		},
		{
			name:       "zero limit never blocks",
			characters: []model.EveCharacter{{Birthday: bday(2024, 1, 1)}},
			maxMonths:  func() *int { v := 0; return &v }(),
			want:       false,
		},
		{
			name:       "old character blocks the welfare",
			characters: []model.EveCharacter{{Birthday: bday(2024, 1, 1)}},
			maxMonths:  func() *int { v := 12; return &v }(),
			want:       true,
		},
		{
			name:       "young characters pass the welfare",
			characters: []model.EveCharacter{{Birthday: bday(2026, 1, 1)}},
			maxMonths:  func() *int { v := 12; return &v }(),
			want:       false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := welfareAgeRestrictionFailed(tt.characters, tt.maxMonths, now)
			if got != tt.want {
				t.Fatalf("welfareAgeRestrictionFailed() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestWelfareMinimumPapRestrictionFailed(t *testing.T) {
	tests := []struct {
		name      string
		minimum   *int
		totalPap  float64
		wantBlock bool
	}{
		{
			name:      "nil minimum never blocks",
			minimum:   nil,
			totalPap:  0,
			wantBlock: false,
		},
		{
			name:      "zero minimum never blocks",
			minimum:   func() *int { v := 0; return &v }(),
			totalPap:  0,
			wantBlock: false,
		},
		{
			name:      "total equal to minimum blocks (strictly greater required)",
			minimum:   func() *int { v := 10; return &v }(),
			totalPap:  10,
			wantBlock: true,
		},
		{
			name:      "total below minimum blocks",
			minimum:   func() *int { v := 10; return &v }(),
			totalPap:  9.9,
			wantBlock: true,
		},
		{
			name:      "total above minimum passes",
			minimum:   func() *int { v := 10; return &v }(),
			totalPap:  10.1,
			wantBlock: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := welfareMinimumPapRestrictionFailed(tt.minimum, tt.totalPap)
			if got != tt.wantBlock {
				t.Fatalf("welfareMinimumPapRestrictionFailed() = %v, want %v", got, tt.wantBlock)
			}
		})
	}
}

func TestFillWelfareSkillPlanNamesReturnsEnrichedCopy(t *testing.T) {
	db := newWelfareServiceTestDB(t)
	useWelfareServiceTestDB(t, db)

	plans := []model.SkillPlan{
		{ID: 1, Title: "Starter Plan", CreatedBy: 1},
		{ID: 2, Title: "Logistics Plan", CreatedBy: 1},
	}
	if err := db.Create(&plans).Error; err != nil {
		t.Fatalf("create skill plans: %v", err)
	}

	svc := &WelfareService{planRepo: repository.NewSkillPlanRepository()}
	input := []model.Welfare{
		{BaseModel: model.BaseModel{ID: 11}, SkillPlanIDs: []uint{1, 2}},
		{BaseModel: model.BaseModel{ID: 12}},
	}

	enriched, err := svc.fillWelfareSkillPlanNames(input)
	if err != nil {
		t.Fatalf("fillWelfareSkillPlanNames() error = %v", err)
	}
	if len(enriched[0].SkillPlanNames) != 2 {
		t.Fatalf("enriched skill plan names = %#v, want 2 names", enriched[0].SkillPlanNames)
	}
	if len(input[0].SkillPlanNames) != 0 {
		t.Fatalf("expected input welfare slice to remain unchanged, got %#v", input[0].SkillPlanNames)
	}
	if enriched[1].SkillPlanNames == nil {
		t.Fatal("expected enriched welfare with no plans to get an empty skill plan name slice")
	}
}

func TestWelfareFuxiLegionYearsRestrictionFailed(t *testing.T) {
	tests := []struct {
		name         string
		characters   []model.EveCharacter
		minimumYears *int
		want         bool
	}{
		{
			name:         "nil minimum never blocks",
			characters:   []model.EveCharacter{{CorporationID: model.SystemCorporationID}},
			minimumYears: nil,
			want:         false,
		},
		{
			name:         "zero minimum never blocks",
			characters:   []model.EveCharacter{{CorporationID: model.SystemCorporationID}},
			minimumYears: func() *int { v := 0; return &v }(),
			want:         false,
		},
		{
			name: "character with enough cumulative legion tenure passes",
			characters: []model.EveCharacter{{
				CorporationID:        model.SystemCorporationID,
				FuxiLegionTenureDays: func() *int { v := 365 * 4; return &v }(),
			}},
			minimumYears: func() *int { v := 4; return &v }(),
			want:         false,
		},
		{
			name: "character below cumulative legion tenure is blocked",
			characters: []model.EveCharacter{{
				CorporationID:        model.SystemCorporationID,
				FuxiLegionTenureDays: func() *int { v := 364; return &v }(),
			}},
			minimumYears: func() *int { v := 1; return &v }(),
			want:         true,
		},
		{
			name: "former legion member still qualifies once cumulative tenure is enough",
			characters: []model.EveCharacter{{
				CorporationID:        12345,
				FuxiLegionTenureDays: func() *int { v := 365; return &v }(),
			}},
			minimumYears: func() *int { v := 1; return &v }(),
			want:         false,
		},
		{
			name: "three non-consecutive nine month periods can pass a two year threshold",
			characters: []model.EveCharacter{
				{CorporationID: 12345},
				{
					CorporationID:        12345,
					FuxiLegionTenureDays: func() *int { v := 30 * 27; return &v }(),
				},
			},
			minimumYears: func() *int { v := 2; return &v }(),
			want:         false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := welfareFuxiLegionYearsRestrictionFailed(tt.characters, tt.minimumYears)
			if got != tt.want {
				t.Fatalf("welfareFuxiLegionYearsRestrictionFailed() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestBuildEligibleWelfareRespKeepsMinimumPapRestrictedWelfareVisible(t *testing.T) {
	svc := &WelfareService{}

	user := &model.User{QQ: "12345", DiscordID: "discord-1"}
	characters := []model.EveCharacter{
		{CharacterID: 1001, CharacterName: "Alpha"},
		{CharacterID: 1002, CharacterName: "Beta"},
	}

	t.Run("per user welfare stays visible but disabled", func(t *testing.T) {
		minimumPap := func() *int { v := 10; return &v }()
		welfare := model.Welfare{
			BaseModel:        model.BaseModel{ID: 12},
			Name:             "Per User Minimum PAP",
			DistMode:         model.WelfareDistModePerUser,
			MinimumPap:       minimumPap,
			RequireSkillPlan: false,
		}

		got, ok := svc.buildEligibleWelfareResp(user, characters, nil, welfare, nil, true, false)
		if !ok {
			t.Fatal("expected minimum PAP restricted welfare to stay visible")
		}
		if got.CanApplyNow {
			t.Fatal("expected per-user welfare to be disabled when minimum PAP is not met")
		}
	})

	t.Run("per character welfare keeps rows visible but disabled", func(t *testing.T) {
		minimumPap := func() *int { v := 10; return &v }()
		welfare := model.Welfare{
			BaseModel:        model.BaseModel{ID: 13},
			Name:             "Per Character Minimum PAP",
			DistMode:         model.WelfareDistModePerCharacter,
			MinimumPap:       minimumPap,
			RequireSkillPlan: false,
		}

		got, ok := svc.buildEligibleWelfareResp(user, characters, nil, welfare, nil, true, false)
		if !ok {
			t.Fatal("expected minimum PAP restricted welfare to stay visible")
		}
		if len(got.EligibleCharacters) != 2 {
			t.Fatalf("expected 2 character rows, got %d", len(got.EligibleCharacters))
		}
		for _, row := range got.EligibleCharacters {
			if row.CanApplyNow {
				t.Fatal("expected per-character welfare rows to be disabled when minimum PAP is not met")
			}
		}
	})
}

func TestBuildEligibleWelfareRespKeepsFuxiLegionRestrictedWelfareVisible(t *testing.T) {
	svc := &WelfareService{}

	user := &model.User{QQ: "12345", DiscordID: "discord-1"}
	characters := []model.EveCharacter{
		{CharacterID: 1001, CharacterName: "Alpha", CorporationID: 12345},
		{CharacterID: 1002, CharacterName: "Beta", CorporationID: 67890},
	}
	minimumYears := func() *int { v := 3; return &v }()

	t.Run("per user welfare stays visible but disabled", func(t *testing.T) {
		welfare := model.Welfare{
			BaseModel:              model.BaseModel{ID: 14},
			Name:                   "Per User Legion Tenure",
			DistMode:               model.WelfareDistModePerUser,
			MinimumFuxiLegionYears: minimumYears,
		}

		got, ok := svc.buildEligibleWelfareResp(user, characters, nil, welfare, nil, false, true)
		if !ok {
			t.Fatal("expected legion tenure restricted welfare to stay visible")
		}
		if got.CanApplyNow {
			t.Fatal("expected per-user welfare to be disabled when legion tenure is not met")
		}
		if got.IneligibleReason != "legion_years" {
			t.Fatalf("ineligible reason = %q, want legion_years", got.IneligibleReason)
		}
	})

	t.Run("per character welfare keeps rows visible but disabled", func(t *testing.T) {
		welfare := model.Welfare{
			BaseModel:              model.BaseModel{ID: 15},
			Name:                   "Per Character Legion Tenure",
			DistMode:               model.WelfareDistModePerCharacter,
			MinimumFuxiLegionYears: minimumYears,
		}

		got, ok := svc.buildEligibleWelfareResp(user, characters, nil, welfare, nil, false, true)
		if !ok {
			t.Fatal("expected legion tenure restricted welfare to stay visible")
		}
		if len(got.EligibleCharacters) != 2 {
			t.Fatalf("expected 2 character rows, got %d", len(got.EligibleCharacters))
		}
		for _, row := range got.EligibleCharacters {
			if row.CanApplyNow {
				t.Fatal("expected per-character welfare rows to be disabled when legion tenure is not met")
			}
			if row.IneligibleReason != "legion_years" {
				t.Fatalf("ineligible reason = %q, want legion_years", row.IneligibleReason)
			}
		}
	})
}

func TestBuildEligibleWelfareRespIncludesFutureSkillOptions(t *testing.T) {
	svc := &WelfareService{}

	user := &model.User{
		QQ:        "12345",
		DiscordID: "discord-1",
	}
	characters := []model.EveCharacter{
		{CharacterID: 1001, CharacterName: "Alpha"},
		{CharacterID: 1002, CharacterName: "Beta"},
	}

	t.Run("per user welfare stays visible but disabled when only future skill growth could satisfy it", func(t *testing.T) {
		welfare := model.Welfare{
			BaseModel:        model.BaseModel{ID: 10},
			Name:             "Per User Welfare",
			DistMode:         model.WelfareDistModePerUser,
			RequireSkillPlan: true,
			SkillPlanIDs:     []uint{7},
			SkillPlanNames:   []string{"Alpha Plan"},
		}
		skillCache := map[int64]map[uint]bool{
			1001: {7: false},
			1002: {7: false},
		}

		got, ok := svc.buildEligibleWelfareResp(user, characters, nil, welfare, skillCache, false, false)
		if !ok {
			t.Fatal("expected future-eligible welfare to stay visible")
		}
		if got.CanApplyNow {
			t.Fatal("expected per-user welfare to be disabled when no character satisfies the skill plan yet")
		}
		if len(got.SkillPlanNames) != 1 || got.SkillPlanNames[0] != "Alpha Plan" {
			t.Fatalf("expected skill plan names to be propagated, got %+v", got.SkillPlanNames)
		}
		if len(got.EligibleCharacters) != 0 {
			t.Fatalf("expected no character rows for per-user welfare, got %d", len(got.EligibleCharacters))
		}
	})

	t.Run("per character welfare keeps both current and future rows", func(t *testing.T) {
		welfare := model.Welfare{
			BaseModel:        model.BaseModel{ID: 11},
			Name:             "Per Character Welfare",
			DistMode:         model.WelfareDistModePerCharacter,
			RequireSkillPlan: true,
			SkillPlanIDs:     []uint{7},
		}
		skillCache := map[int64]map[uint]bool{
			1001: {7: true},
			1002: {7: false},
		}

		got, ok := svc.buildEligibleWelfareResp(user, characters, nil, welfare, skillCache, false, false)
		if !ok {
			t.Fatal("expected per-character welfare to stay visible")
		}
		if len(got.EligibleCharacters) != 2 {
			t.Fatalf("expected 2 character rows, got %d", len(got.EligibleCharacters))
		}
		if !got.EligibleCharacters[0].CanApplyNow {
			t.Fatal("expected the first character to be currently eligible")
		}
		if got.EligibleCharacters[1].CanApplyNow {
			t.Fatal("expected the second character to be future-only")
		}
	})
}

func TestSkillPlanNamesForWelfarePreservesConfiguredOrder(t *testing.T) {
	got := skillPlanNamesForWelfare([]uint{7, 3, 9}, map[uint]string{
		3: "Shield Plan",
		7: "Armor Plan",
		9: "",
	})

	want := []string{"Armor Plan", "Shield Plan"}
	if len(got) != len(want) {
		t.Fatalf("expected %d plan names, got %d (%+v)", len(want), len(got), got)
	}
	for index := range want {
		if got[index] != want[index] {
			t.Fatalf("got[%d] = %q, want %q", index, got[index], want[index])
		}
	}
}

func TestBuildMyApplicationResponsesIncludesReviewerNickname(t *testing.T) {
	createdAt := time.Date(2026, 3, 30, 8, 0, 0, 0, time.UTC)
	reviewedAt := time.Date(2026, 3, 30, 9, 30, 0, 0, time.UTC)

	apps := []model.WelfareApplication{
		{
			BaseModel:     model.BaseModel{ID: 1, CreatedAt: createdAt},
			WelfareID:     10,
			CharacterName: "Alpha",
			Status:        model.WelfareAppStatusDelivered,
			ReviewedBy:    77,
			ReviewedAt:    &reviewedAt,
		},
		{
			BaseModel:     model.BaseModel{ID: 2, CreatedAt: createdAt},
			WelfareID:     11,
			CharacterName: "Beta",
			Status:        model.WelfareAppStatusRequested,
		},
	}

	got := buildMyApplicationResponses(apps, map[uint]string{
		10: "Starter Pack",
		11: "Advanced Pack",
	}, map[uint]string{
		77: "Officer Fox",
	})

	if len(got) != 2 {
		t.Fatalf("expected 2 responses, got %d", len(got))
	}
	if got[0].ReviewerName != "Officer Fox" {
		t.Fatalf("expected reviewer nickname to be included, got %q", got[0].ReviewerName)
	}
	if got[1].ReviewerName != "" {
		t.Fatalf("expected empty reviewer nickname for unreviewed applications, got %q", got[1].ReviewerName)
	}
}

func TestParseImportedWelfareApplicationsSupportsCommaAndTabSeparatedRows(t *testing.T) {
	apps, err := parseImportedWelfareApplications(7, "Alice, 12345\n\nBob\t67890\nCharlie")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(apps) != 3 {
		t.Fatalf("expected 3 parsed applications, got %d", len(apps))
	}

	if apps[0].WelfareID != 7 || apps[0].CharacterName != "Alice" || apps[0].QQ != "12345" {
		t.Fatalf("unexpected first application: %+v", apps[0])
	}
	if apps[0].Status != model.WelfareAppStatusDelivered {
		t.Fatalf("expected imported status %q, got %q", model.WelfareAppStatusDelivered, apps[0].Status)
	}
	if apps[0].UserID != nil {
		t.Fatalf("expected imported user ID to be nil, got %v", apps[0].UserID)
	}

	if apps[1].CharacterName != "Bob" || apps[1].QQ != "67890" {
		t.Fatalf("unexpected second application: %+v", apps[1])
	}

	if apps[2].CharacterName != "Charlie" || apps[2].QQ != "" {
		t.Fatalf("unexpected third application: %+v", apps[2])
	}
}

func TestParseImportedWelfareApplicationsRejectsEmptyResult(t *testing.T) {
	_, err := parseImportedWelfareApplications(7, "\n , \n\t")
	if err == nil {
		t.Fatal("expected error for empty parsed import result")
	}
}

func TestApplyForWelfareSmallFuxiCoinClaimsAutoDeliver(t *testing.T) {
	db := newWelfareServiceTestDB(t)
	useWelfareServiceTestDB(t, db)

	user := &model.User{
		Nickname:           "Pilot",
		QQ:                 "123456",
		PrimaryCharacterID: 90000001,
	}
	if err := db.Create(user).Error; err != nil {
		t.Fatalf("create user: %v", err)
	}

	character := &model.EveCharacter{
		CharacterID:   user.PrimaryCharacterID,
		CharacterName: "Pilot One",
		UserID:        user.ID,
	}
	if err := db.Create(character).Error; err != nil {
		t.Fatalf("create character: %v", err)
	}

	payout := 499
	welfare := &model.Welfare{
		Name:          "Starter Pack",
		DistMode:      model.WelfareDistModePerUser,
		PayByFuxiCoin: &payout,
		Status:        model.WelfareStatusActive,
		CreatedBy:     1,
	}
	if err := db.Create(welfare).Error; err != nil {
		t.Fatalf("create welfare: %v", err)
	}

	svc := NewWelfareService()
	app, err := svc.ApplyForWelfare(user.ID, &ApplyForWelfareRequest{WelfareID: welfare.ID})
	if err != nil {
		t.Fatalf("ApplyForWelfare() error = %v", err)
	}
	if app.Status != model.WelfareAppStatusDelivered {
		t.Fatalf("status = %q, want %q", app.Status, model.WelfareAppStatusDelivered)
	}
	if app.ReviewedAt == nil {
		t.Fatal("expected reviewed_at to be set for auto-delivered application")
	}
	if app.ReviewedBy != 0 {
		t.Fatalf("reviewed_by = %d, want 0", app.ReviewedBy)
	}

	var persisted model.WelfareApplication
	if err := db.First(&persisted, app.ID).Error; err != nil {
		t.Fatalf("reload application: %v", err)
	}
	if persisted.Status != model.WelfareAppStatusDelivered {
		t.Fatalf("persisted status = %q, want %q", persisted.Status, model.WelfareAppStatusDelivered)
	}
	if persisted.ReviewedAt == nil {
		t.Fatal("expected persisted reviewed_at to be set")
	}

	var wallet model.SystemWallet
	if err := db.Where("user_id = ?", user.ID).First(&wallet).Error; err != nil {
		t.Fatalf("load wallet: %v", err)
	}
	if wallet.Balance != 499 {
		t.Fatalf("wallet balance = %v, want 499", wallet.Balance)
	}

	var txs []model.WalletTransaction
	if err := db.Order("id ASC").Find(&txs).Error; err != nil {
		t.Fatalf("load wallet transactions: %v", err)
	}
	if len(txs) != 1 {
		t.Fatalf("wallet transaction count = %d, want 1", len(txs))
	}
	if txs[0].RefType != model.WalletRefWelfarePayout {
		t.Fatalf("wallet tx ref_type = %q, want %q", txs[0].RefType, model.WalletRefWelfarePayout)
	}
	if txs[0].OperatorID != 0 {
		t.Fatalf("wallet tx operator_id = %d, want 0", txs[0].OperatorID)
	}
}

func TestApplyForWelfareAtAutoApprovalThresholdStaysRequested(t *testing.T) {
	db := newWelfareServiceTestDB(t)
	useWelfareServiceTestDB(t, db)

	user := &model.User{
		Nickname:           "Pilot",
		QQ:                 "123456",
		PrimaryCharacterID: 90000001,
	}
	if err := db.Create(user).Error; err != nil {
		t.Fatalf("create user: %v", err)
	}

	character := &model.EveCharacter{
		CharacterID:   user.PrimaryCharacterID,
		CharacterName: "Pilot One",
		UserID:        user.ID,
	}
	if err := db.Create(character).Error; err != nil {
		t.Fatalf("create character: %v", err)
	}

	payout := 500
	welfare := &model.Welfare{
		Name:          "Starter Pack",
		DistMode:      model.WelfareDistModePerUser,
		PayByFuxiCoin: &payout,
		Status:        model.WelfareStatusActive,
		CreatedBy:     1,
	}
	if err := db.Create(welfare).Error; err != nil {
		t.Fatalf("create welfare: %v", err)
	}

	svc := NewWelfareService()
	app, err := svc.ApplyForWelfare(user.ID, &ApplyForWelfareRequest{WelfareID: welfare.ID})
	if err != nil {
		t.Fatalf("ApplyForWelfare() error = %v", err)
	}
	if app.Status != model.WelfareAppStatusRequested {
		t.Fatalf("status = %q, want %q", app.Status, model.WelfareAppStatusRequested)
	}
	if app.ReviewedAt != nil {
		t.Fatalf("reviewed_at = %v, want nil", app.ReviewedAt)
	}

	var txs []model.WalletTransaction
	if err := db.Find(&txs).Error; err != nil {
		t.Fatalf("load wallet transactions: %v", err)
	}
	if len(txs) != 0 {
		t.Fatalf("wallet transaction count = %d, want 0", len(txs))
	}

	var auditEvents []model.AuditEvent
	if err := db.Where("resource_type = ? AND resource_id = ?", "welfare_application", fmt.Sprintf("%d", app.ID)).
		Find(&auditEvents).Error; err != nil {
		t.Fatalf("load audit events: %v", err)
	}
	foundReject := false
	for _, event := range auditEvents {
		if event.Category == "approval" && event.Action == "welfare_application_reject" && event.Result == model.AuditResultSuccess {
			foundReject = true
			break
		}
	}
	if !foundReject {
		t.Fatalf("expected welfare_application_reject approval audit event, got %+v", auditEvents)
	}
}

func TestApplyForWelfareUsesConfiguredAutoApproveThreshold(t *testing.T) {
	db := newWelfareServiceTestDB(t)
	useWelfareServiceTestDB(t, db)

	user := &model.User{
		Nickname:           "Pilot",
		QQ:                 "123456",
		PrimaryCharacterID: 90000001,
	}
	if err := db.Create(user).Error; err != nil {
		t.Fatalf("create user: %v", err)
	}

	character := &model.EveCharacter{
		CharacterID:   user.PrimaryCharacterID,
		CharacterName: "Pilot One",
		UserID:        user.ID,
	}
	if err := db.Create(character).Error; err != nil {
		t.Fatalf("create character: %v", err)
	}

	payout := 400
	welfare := &model.Welfare{
		Name:          "Starter Pack",
		DistMode:      model.WelfareDistModePerUser,
		PayByFuxiCoin: &payout,
		Status:        model.WelfareStatusActive,
		CreatedBy:     1,
	}
	if err := db.Create(welfare).Error; err != nil {
		t.Fatalf("create welfare: %v", err)
	}

	svc := NewWelfareService()
	// fakeWelfareSettingsConfigStore is defined in welfare_settings_test.go (same package).
	svc.cfgRepo = &fakeWelfareSettingsConfigStore{threshold: 300, hasThreshold: true}

	app, err := svc.ApplyForWelfare(user.ID, &ApplyForWelfareRequest{WelfareID: welfare.ID})
	if err != nil {
		t.Fatalf("ApplyForWelfare() error = %v", err)
	}
	if app.Status != model.WelfareAppStatusRequested {
		t.Fatalf("status = %q, want %q", app.Status, model.WelfareAppStatusRequested)
	}

	var txs []model.WalletTransaction
	if err := db.Find(&txs).Error; err != nil {
		t.Fatalf("load wallet transactions: %v", err)
	}
	if len(txs) != 0 {
		t.Fatalf("wallet transaction count = %d, want 0", len(txs))
	}
}

func TestApplyForWelfareRejectsMissingFuxiLegionTenure(t *testing.T) {
	db := newWelfareServiceTestDB(t)
	useWelfareServiceTestDB(t, db)

	user := &model.User{
		Nickname:           "Pilot",
		QQ:                 "123456",
		PrimaryCharacterID: 90000001,
	}
	if err := db.Create(user).Error; err != nil {
		t.Fatalf("create user: %v", err)
	}

	character := &model.EveCharacter{
		CharacterID:          user.PrimaryCharacterID,
		CharacterName:        "Pilot One",
		UserID:               user.ID,
		CorporationID:        12345,
		FuxiLegionTenureDays: func() *int { v := 300; return &v }(),
	}
	if err := db.Create(character).Error; err != nil {
		t.Fatalf("create character: %v", err)
	}

	minimumYears := 2
	welfare := &model.Welfare{
		Name:                   "Veteran Reward",
		DistMode:               model.WelfareDistModePerUser,
		MinimumFuxiLegionYears: &minimumYears,
		Status:                 model.WelfareStatusActive,
		CreatedBy:              1,
	}
	if err := db.Create(welfare).Error; err != nil {
		t.Fatalf("create welfare: %v", err)
	}

	svc := NewWelfareService()
	_, err := svc.ApplyForWelfare(user.ID, &ApplyForWelfareRequest{WelfareID: welfare.ID})
	if err == nil {
		t.Fatal("expected ApplyForWelfare to reject users without enough Fuxi Legion tenure")
	}
}

func TestGetEligibleWelfaresRefreshesBadgeCache(t *testing.T) {
	db := newWelfareServiceTestDB(t)
	useWelfareServiceTestDB(t, db)

	user := &model.User{
		BaseModel:          model.BaseModel{ID: 930001},
		Nickname:           "Pilot Cache",
		QQ:                 "123456",
		PrimaryCharacterID: 93000001,
	}
	if err := db.Create(user).Error; err != nil {
		t.Fatalf("create user: %v", err)
	}

	character := &model.EveCharacter{
		CharacterID:   user.PrimaryCharacterID,
		CharacterName: "Pilot Cache One",
		UserID:        user.ID,
	}
	if err := db.Create(character).Error; err != nil {
		t.Fatalf("create character: %v", err)
	}

	welfare := &model.Welfare{
		Name:      "Cacheable Welfare",
		DistMode:  model.WelfareDistModePerUser,
		Status:    model.WelfareStatusActive,
		CreatedBy: 1,
	}
	if err := db.Create(welfare).Error; err != nil {
		t.Fatalf("create welfare: %v", err)
	}

	welfareSvc := NewWelfareService()
	if _, err := welfareSvc.GetEligibleWelfares(user.ID); err != nil {
		t.Fatalf("GetEligibleWelfares() error = %v", err)
	}

	got, err := NewBadgeService().GetBadgeCounts(user.ID, []string{model.RoleUser})
	if err != nil {
		t.Fatalf("GetBadgeCounts() error = %v", err)
	}
	if got[BadgeCountWelfareEligible] != 1 {
		t.Fatalf("expected warmed welfare badge cache to return 1, got %#v", got)
	}
}

func TestApplyForWelfareRefreshesBadgeCache(t *testing.T) {
	db := newWelfareServiceTestDB(t)
	useWelfareServiceTestDB(t, db)

	user := &model.User{
		BaseModel:          model.BaseModel{ID: 930002},
		Nickname:           "Pilot Apply",
		QQ:                 "654321",
		PrimaryCharacterID: 93000002,
	}
	if err := db.Create(user).Error; err != nil {
		t.Fatalf("create user: %v", err)
	}

	character := &model.EveCharacter{
		CharacterID:   user.PrimaryCharacterID,
		CharacterName: "Pilot Apply One",
		UserID:        user.ID,
	}
	if err := db.Create(character).Error; err != nil {
		t.Fatalf("create character: %v", err)
	}

	welfare := &model.Welfare{
		Name:      "Apply Once Welfare",
		DistMode:  model.WelfareDistModePerUser,
		Status:    model.WelfareStatusActive,
		CreatedBy: 1,
	}
	if err := db.Create(welfare).Error; err != nil {
		t.Fatalf("create welfare: %v", err)
	}

	welfareSvc := NewWelfareService()
	if _, err := welfareSvc.GetEligibleWelfares(user.ID); err != nil {
		t.Fatalf("GetEligibleWelfares() error = %v", err)
	}

	if _, err := welfareSvc.ApplyForWelfare(user.ID, &ApplyForWelfareRequest{WelfareID: welfare.ID}); err != nil {
		t.Fatalf("ApplyForWelfare() error = %v", err)
	}

	got, err := NewBadgeService().GetBadgeCounts(user.ID, []string{model.RoleUser})
	if err != nil {
		t.Fatalf("GetBadgeCounts() error = %v", err)
	}
	if _, ok := got[BadgeCountWelfareEligible]; ok {
		t.Fatalf("expected welfare badge cache to refresh to zero after apply, got %#v", got)
	}
}

func TestApplyForWelfareAvoidsEligibilityRefetchESIAfterApply(t *testing.T) {
	db := newWelfareServiceTestDB(t)
	useWelfareServiceTestDB(t, db)

	user := &model.User{
		BaseModel:          model.BaseModel{ID: 930003},
		Nickname:           "Pilot No Refetch",
		QQ:                 "111222",
		PrimaryCharacterID: 93000003,
	}
	if err := db.Create(user).Error; err != nil {
		t.Fatalf("create user: %v", err)
	}

	character := &model.EveCharacter{
		CharacterID:   user.PrimaryCharacterID,
		CharacterName: "Pilot No Refetch One",
		UserID:        user.ID,
	}
	if err := db.Create(character).Error; err != nil {
		t.Fatalf("create character: %v", err)
	}

	applyWelfare := &model.Welfare{
		Name:      "Apply Without Refetch",
		DistMode:  model.WelfareDistModePerUser,
		Status:    model.WelfareStatusActive,
		CreatedBy: 1,
	}
	if err := db.Create(applyWelfare).Error; err != nil {
		t.Fatalf("create apply welfare: %v", err)
	}

	minimumYears := 1
	blockedWelfare := &model.Welfare{
		Name:                   "Blocked By Tenure",
		DistMode:               model.WelfareDistModePerUser,
		MinimumFuxiLegionYears: &minimumYears,
		Status:                 model.WelfareStatusActive,
		CreatedBy:              1,
	}
	if err := db.Create(blockedWelfare).Error; err != nil {
		t.Fatalf("create blocked welfare: %v", err)
	}

	var requests int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		requests++
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte("[]"))
	}))
	t.Cleanup(server.Close)

	svc := NewWelfareService()
	svc.esiClient = esi.NewClientWithConfig(server.URL, "")

	if _, err := svc.ApplyForWelfare(user.ID, &ApplyForWelfareRequest{WelfareID: applyWelfare.ID}); err != nil {
		t.Fatalf("ApplyForWelfare() error = %v", err)
	}
	if requests != 0 {
		t.Fatalf("expected apply path to avoid unrelated ESI refetch, got %d requests", requests)
	}
}

func TestEnsureFuxiLegionTenureDaysSkipsCachedValuesForFormerMembers(t *testing.T) {
	var requests int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		requests++
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte("[]"))
	}))
	t.Cleanup(server.Close)

	existingTenureDays := 730
	svc := &WelfareService{esiClient: esi.NewClientWithConfig(server.URL, "")}
	characters := []model.EveCharacter{{
		CharacterID:          93000004,
		CorporationID:        12345,
		FuxiLegionTenureDays: &existingTenureDays,
	}}

	svc.ensureFuxiLegionTenureDays(characters)

	if requests != 0 {
		t.Fatalf("expected cached tenure for former member to skip ESI fetch, got %d requests", requests)
	}
	if characters[0].FuxiLegionTenureDays == nil || *characters[0].FuxiLegionTenureDays != existingTenureDays {
		t.Fatalf("expected cached tenure to remain unchanged, got %+v", characters[0].FuxiLegionTenureDays)
	}
}

func TestEnsureFuxiLegionTenureDaysSkipsFormerMembersWithoutCachedValues(t *testing.T) {
	var requests int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		requests++
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte("[]"))
	}))
	t.Cleanup(server.Close)

	svc := &WelfareService{esiClient: esi.NewClientWithConfig(server.URL, "")}
	characters := []model.EveCharacter{{
		CharacterID:   93000006,
		CorporationID: 12345,
	}}

	svc.ensureFuxiLegionTenureDays(characters)

	if requests != 0 {
		t.Fatalf("expected former member without cached tenure to skip ESI fetch, got %d requests", requests)
	}
	if characters[0].FuxiLegionTenureDays != nil {
		t.Fatalf("expected former member tenure to stay nil, got %+v", characters[0].FuxiLegionTenureDays)
	}
}

func TestEnsureFuxiLegionTenureDaysUsesPersistedValuesForCurrentMembers(t *testing.T) {
	db := newWelfareServiceTestDB(t)
	useWelfareServiceTestDB(t, db)

	persistedTenureDays := 12
	character := &model.EveCharacter{
		CharacterID:          93000005,
		CharacterName:        "Pilot Current Fuxi",
		UserID:               1,
		CorporationID:        model.SystemCorporationID,
		FuxiLegionTenureDays: &persistedTenureDays,
	}
	if err := db.Create(character).Error; err != nil {
		t.Fatalf("create character: %v", err)
	}

	var requests int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		requests++
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte("[]"))
	}))
	t.Cleanup(server.Close)

	svc := NewWelfareService()
	svc.esiClient = esi.NewClientWithConfig(server.URL, "")
	characters := []model.EveCharacter{{
		CharacterID:   character.CharacterID,
		CharacterName: character.CharacterName,
		UserID:        character.UserID,
		CorporationID: model.SystemCorporationID,
	}}

	svc.ensureFuxiLegionTenureDays(characters)

	if requests != 0 {
		t.Fatalf("expected current member tenure to reuse persisted state without ESI fetch, got %d requests", requests)
	}
	if characters[0].FuxiLegionTenureDays == nil || *characters[0].FuxiLegionTenureDays != persistedTenureDays {
		t.Fatalf("expected current member to load persisted tenure days, got %+v", characters[0].FuxiLegionTenureDays)
	}

	dbChar, err := svc.charRepo.GetByCharacterID(character.CharacterID)
	if err != nil {
		t.Fatalf("reload character: %v", err)
	}
	if dbChar.FuxiLegionTenureDays == nil || *dbChar.FuxiLegionTenureDays != persistedTenureDays {
		t.Fatalf("expected persisted tenure days to remain unchanged, got %+v", dbChar.FuxiLegionTenureDays)
	}
}

func TestAdminReviewApplicationDeliverCreditsConfiguredFuxiCoin(t *testing.T) {
	db := newWelfareServiceTestDB(t)
	useWelfareServiceTestDB(t, db)

	payout := 25
	welfare := &model.Welfare{
		Name:          "Starter Pack",
		DistMode:      model.WelfareDistModePerUser,
		PayByFuxiCoin: &payout,
		Status:        model.WelfareStatusActive,
		CreatedBy:     1,
	}
	if err := db.Create(welfare).Error; err != nil {
		t.Fatalf("create welfare: %v", err)
	}

	userID := uint(42)
	app := &model.WelfareApplication{
		WelfareID:     welfare.ID,
		UserID:        &userID,
		CharacterID:   90000001,
		CharacterName: "Pilot One",
		Status:        model.WelfareAppStatusRequested,
	}
	if err := db.Create(app).Error; err != nil {
		t.Fatalf("create application: %v", err)
	}

	svc := NewWelfareService()
	svc.deliveryMailSender = nil
	if _, err := svc.AdminReviewApplication(app.ID, 77, adminDeliveryReviewerRoles, &AdminReviewApplicationRequest{Action: "deliver"}); err != nil {
		t.Fatalf("AdminReviewApplication() error = %v", err)
	}

	var updated model.WelfareApplication
	if err := db.First(&updated, app.ID).Error; err != nil {
		t.Fatalf("reload application: %v", err)
	}
	if updated.Status != model.WelfareAppStatusDelivered {
		t.Fatalf("status = %q, want %q", updated.Status, model.WelfareAppStatusDelivered)
	}
	if updated.ReviewedBy != 77 {
		t.Fatalf("reviewed_by = %d, want 77", updated.ReviewedBy)
	}
	if updated.ReviewedAt == nil {
		t.Fatal("expected reviewed_at to be set")
	}

	var wallet model.SystemWallet
	if err := db.Where("user_id = ?", userID).First(&wallet).Error; err != nil {
		t.Fatalf("load wallet: %v", err)
	}
	if wallet.Balance != 25 {
		t.Fatalf("wallet balance = %v, want 25", wallet.Balance)
	}

	var reviewerWallet model.SystemWallet
	if err := db.Where("user_id = ?", 77).First(&reviewerWallet).Error; err != nil {
		t.Fatalf("load reviewer wallet: %v", err)
	}
	if reviewerWallet.Balance != 10 {
		t.Fatalf("reviewer wallet balance = %v, want 10", reviewerWallet.Balance)
	}

	var txs []model.WalletTransaction
	if err := db.Order("id ASC").Find(&txs).Error; err != nil {
		t.Fatalf("load wallet transactions: %v", err)
	}
	if len(txs) != 2 {
		t.Fatalf("wallet transaction count = %d, want 2", len(txs))
	}

	var payoutTx *model.WalletTransaction
	var awardTx *model.WalletTransaction
	for i := range txs {
		tx := &txs[i]
		switch tx.RefType {
		case model.WalletRefWelfarePayout:
			payoutTx = tx
		case "admin_award":
			awardTx = tx
		}
	}
	if payoutTx == nil {
		t.Fatal("expected welfare payout transaction")
	}
	if payoutTx.RefID != fmt.Sprintf("welfare_application:%d", app.ID) {
		t.Fatalf("wallet tx ref_id = %q", payoutTx.RefID)
	}
	if payoutTx.OperatorID != 77 {
		t.Fatalf("wallet tx operator_id = %d, want 77", payoutTx.OperatorID)
	}
	if awardTx == nil {
		t.Fatal("expected admin award transaction")
	}
	if awardTx.UserID != 77 {
		t.Fatalf("award wallet tx user_id = %d, want 77", awardTx.UserID)
	}
	if awardTx.Amount != 10 {
		t.Fatalf("award wallet tx amount = %v, want 10", awardTx.Amount)
	}
	if awardTx.RefID != fmt.Sprintf("admin_welfare_delivery:%d", app.ID) {
		t.Fatalf("award wallet tx ref_id = %q", awardTx.RefID)
	}
	if awardTx.OperatorID != 0 {
		t.Fatalf("award wallet tx operator_id = %d, want 0", awardTx.OperatorID)
	}
}

func TestAdminReviewApplicationDeliverDispatchesInGameMailAsynchronously(t *testing.T) {
	db := newWelfareServiceTestDB(t)
	useWelfareServiceTestDB(t, db)

	welfare := &model.Welfare{
		Name:      "Starter Pack",
		DistMode:  model.WelfareDistModePerUser,
		Status:    model.WelfareStatusActive,
		CreatedBy: 1,
	}
	if err := db.Create(welfare).Error; err != nil {
		t.Fatalf("create welfare: %v", err)
	}

	userID := uint(42)
	app := &model.WelfareApplication{
		WelfareID:     welfare.ID,
		UserID:        &userID,
		CharacterID:   90000001,
		CharacterName: "Pilot One",
		Status:        model.WelfareAppStatusRequested,
	}
	if err := db.Create(app).Error; err != nil {
		t.Fatalf("create application: %v", err)
	}

	svc := NewWelfareService()
	started := make(chan struct {
		reviewerID    uint
		welfareID     uint
		applicationID uint
	}, 1)
	finished := make(chan struct{})
	release := make(chan struct{})
	released := false
	t.Cleanup(func() {
		if !released {
			close(release)
		}
	})
	svc.deliveryMailSender = func(ctx context.Context, reviewerID uint, deliveredWelfare *model.Welfare, deliveredApp *model.WelfareApplication) (MailAttemptSummary, error) {
		defer close(finished)
		started <- struct {
			reviewerID    uint
			welfareID     uint
			applicationID uint
		}{reviewerID: reviewerID, welfareID: deliveredWelfare.ID, applicationID: deliveredApp.ID}
		<-release
		return MailAttemptSummary{}, errors.New("mail failed")
	}

	resultCh := make(chan struct {
		mailSummary MailAttemptSummary
		err         error
	}, 1)
	go func() {
		mailSummary, err := svc.AdminReviewApplication(app.ID, 77, adminDeliveryReviewerRoles, &AdminReviewApplicationRequest{Action: "deliver"})
		resultCh <- struct {
			mailSummary MailAttemptSummary
			err         error
		}{mailSummary: mailSummary, err: err}
	}()

	select {
	case attempt := <-started:
		if attempt.reviewerID != 77 {
			t.Fatalf("reviewerID = %d, want 77", attempt.reviewerID)
		}
		if attempt.welfareID != welfare.ID {
			t.Fatalf("welfare id = %d, want %d", attempt.welfareID, welfare.ID)
		}
		if attempt.applicationID != app.ID {
			t.Fatalf("application id = %d, want %d", attempt.applicationID, app.ID)
		}
	case <-time.After(time.Second):
		t.Fatal("expected deliver to start in-game mail dispatch")
	}

	var result struct {
		mailSummary MailAttemptSummary
		err         error
	}
	select {
	case result = <-resultCh:
	case <-time.After(200 * time.Millisecond):
		t.Fatal("expected AdminReviewApplication to return without waiting for mail sender")
	}

	if result.err != nil {
		t.Fatalf("AdminReviewApplication() error = %v", result.err)
	}
	if result.mailSummary != (MailAttemptSummary{}) {
		t.Fatalf("mailSummary = %#v, want empty because delivery mail is asynchronous", result.mailSummary)
	}

	var updated model.WelfareApplication
	if err := db.First(&updated, app.ID).Error; err != nil {
		t.Fatalf("reload application: %v", err)
	}
	if updated.Status != model.WelfareAppStatusDelivered {
		t.Fatalf("status = %q, want %q", updated.Status, model.WelfareAppStatusDelivered)
	}
	if updated.ReviewedBy != 77 {
		t.Fatalf("reviewed_by = %d, want 77", updated.ReviewedBy)
	}

	close(release)
	released = true
	select {
	case <-finished:
	case <-time.After(time.Second):
		t.Fatal("expected async mail sender to finish after release")
	}
}

func TestAdminReviewApplicationDeliverReturnsEmptyMailSummaryWhenMailRunsAsync(t *testing.T) {
	db := newWelfareServiceTestDB(t)
	useWelfareServiceTestDB(t, db)

	welfare := &model.Welfare{
		Name:      "Welcome Pack",
		DistMode:  model.WelfareDistModePerUser,
		Status:    model.WelfareStatusActive,
		CreatedBy: 1,
	}
	if err := db.Create(welfare).Error; err != nil {
		t.Fatalf("create welfare: %v", err)
	}

	userID := uint(42)
	app := &model.WelfareApplication{
		WelfareID:     welfare.ID,
		UserID:        &userID,
		CharacterID:   90000001,
		CharacterName: "Pilot One",
		Status:        model.WelfareAppStatusRequested,
	}
	if err := db.Create(app).Error; err != nil {
		t.Fatalf("create application: %v", err)
	}

	svc := NewWelfareService()
	mailAttempted := make(chan struct{}, 1)
	svc.deliveryMailSender = func(ctx context.Context, reviewerID uint, deliveredWelfare *model.Welfare, deliveredApp *model.WelfareApplication) (MailAttemptSummary, error) {
		mailAttempted <- struct{}{}
		return MailAttemptSummary{
			MailID:                     123456789,
			MailSenderCharacterID:      90000077,
			MailSenderCharacterName:    "Officer Main",
			MailRecipientCharacterID:   90000042,
			MailRecipientCharacterName: "Pilot Main",
		}, nil
	}

	mailSummary, err := svc.AdminReviewApplication(app.ID, 77, adminDeliveryReviewerRoles, &AdminReviewApplicationRequest{Action: "deliver"})
	if err != nil {
		t.Fatalf("AdminReviewApplication() error = %v", err)
	}
	if mailSummary != (MailAttemptSummary{}) {
		t.Fatalf("mailSummary = %#v, want empty because delivery mail is asynchronous", mailSummary)
	}
	select {
	case <-mailAttempted:
	case <-time.After(time.Second):
		t.Fatal("expected deliver to trigger in-game mail sender asynchronously")
	}
}

func TestBuildWelfareDeliveryMailContentIncludesBilingualOfficerNotice(t *testing.T) {
	subject, body := buildWelfareDeliveryMailContent("Starter Pack", "Amiya")

	if !strings.Contains(subject, "福利发放通知") || !strings.Contains(subject, "Welfare Delivery Notice") {
		t.Fatalf("unexpected subject: %q", subject)
	}
	if !strings.Contains(body, "你的福利「Starter Pack」已由福利官 Amiya 发放") {
		t.Fatalf("expected Chinese body to mention welfare name and officer nickname, got %q", body)
	}
	if !strings.Contains(body, "福利名称：Starter Pack") {
		t.Fatalf("expected Chinese body to include welfare name detail, got %q", body)
	}
	if !strings.Contains(body, "请检查你的伏羲币钱包或合同") {
		t.Fatalf("expected Chinese body to mention FuxiCoin wallet or contract, got %q", body)
	}
	if !strings.Contains(body, "如有疑问，请联系处理此申请的福利官。") {
		t.Fatalf("expected Chinese body to include a professional follow-up note, got %q", body)
	}
	if !strings.Contains(body, "Your welfare \"Starter Pack\" has been delivered by officer Amiya.") {
		t.Fatalf("expected English body to mention welfare name and officer nickname, got %q", body)
	}
	if !strings.Contains(body, "Welfare: Starter Pack") {
		t.Fatalf("expected English body to include welfare detail, got %q", body)
	}
	if !strings.Contains(body, "Please check your FuxiCoin wallet or contract.") {
		t.Fatalf("expected English body to mention FuxiCoin wallet or contract, got %q", body)
	}
	if !strings.Contains(body, "If anything looks incorrect, please contact the officer who handled this delivery.") {
		t.Fatalf("expected English body to include a professional follow-up note, got %q", body)
	}
}

func TestAdminReviewApplicationDeliverWithoutConfiguredPayoutStillCreditsAdminAward(t *testing.T) {
	db := newWelfareServiceTestDB(t)
	useWelfareServiceTestDB(t, db)

	welfare := &model.Welfare{
		Name:      "Starter Pack",
		DistMode:  model.WelfareDistModePerUser,
		Status:    model.WelfareStatusActive,
		CreatedBy: 1,
	}
	if err := db.Create(welfare).Error; err != nil {
		t.Fatalf("create welfare: %v", err)
	}

	userID := uint(42)
	app := &model.WelfareApplication{
		WelfareID:     welfare.ID,
		UserID:        &userID,
		CharacterID:   90000001,
		CharacterName: "Pilot One",
		Status:        model.WelfareAppStatusRequested,
	}
	if err := db.Create(app).Error; err != nil {
		t.Fatalf("create application: %v", err)
	}

	svc := NewWelfareService()
	svc.deliveryMailSender = nil
	if _, err := svc.AdminReviewApplication(app.ID, 77, adminDeliveryReviewerRoles, &AdminReviewApplicationRequest{Action: "deliver"}); err != nil {
		t.Fatalf("AdminReviewApplication() error = %v", err)
	}

	var txs []model.WalletTransaction
	if err := db.Find(&txs).Error; err != nil {
		t.Fatalf("load wallet transactions: %v", err)
	}
	if len(txs) != 1 {
		t.Fatalf("wallet transaction count = %d, want 1", len(txs))
	}
	if txs[0].UserID != 77 {
		t.Fatalf("wallet tx user_id = %d, want 77", txs[0].UserID)
	}
	if txs[0].RefType != "admin_award" {
		t.Fatalf("wallet tx ref_type = %q, want %q", txs[0].RefType, "admin_award")
	}
	if txs[0].Amount != 10 {
		t.Fatalf("wallet tx amount = %v, want 10", txs[0].Amount)
	}
}

func TestAdminReviewApplicationDeliverUsesApprovalTimePayoutConfig(t *testing.T) {
	db := newWelfareServiceTestDB(t)
	useWelfareServiceTestDB(t, db)

	initialPayout := 10
	welfare := &model.Welfare{
		Name:          "Starter Pack",
		DistMode:      model.WelfareDistModePerUser,
		PayByFuxiCoin: &initialPayout,
		Status:        model.WelfareStatusActive,
		CreatedBy:     1,
	}
	if err := db.Create(welfare).Error; err != nil {
		t.Fatalf("create welfare: %v", err)
	}

	userID := uint(42)
	app := &model.WelfareApplication{
		WelfareID:     welfare.ID,
		UserID:        &userID,
		CharacterID:   90000001,
		CharacterName: "Pilot One",
		Status:        model.WelfareAppStatusRequested,
	}
	if err := db.Create(app).Error; err != nil {
		t.Fatalf("create application: %v", err)
	}

	updatedPayout := 35
	if err := db.Model(&model.Welfare{}).Where("id = ?", welfare.ID).
		Update("pay_by_fuxi_coin", updatedPayout).Error; err != nil {
		t.Fatalf("update welfare payout: %v", err)
	}

	svc := NewWelfareService()
	svc.deliveryMailSender = nil
	if _, err := svc.AdminReviewApplication(app.ID, 77, adminDeliveryReviewerRoles, &AdminReviewApplicationRequest{Action: "deliver"}); err != nil {
		t.Fatalf("AdminReviewApplication() error = %v", err)
	}

	var wallet model.SystemWallet
	if err := db.Where("user_id = ?", userID).First(&wallet).Error; err != nil {
		t.Fatalf("load wallet: %v", err)
	}
	if wallet.Balance != 35 {
		t.Fatalf("wallet balance = %v, want 35", wallet.Balance)
	}

	var reviewerWallet model.SystemWallet
	if err := db.Where("user_id = ?", 77).First(&reviewerWallet).Error; err != nil {
		t.Fatalf("load reviewer wallet: %v", err)
	}
	if reviewerWallet.Balance != 10 {
		t.Fatalf("reviewer wallet balance = %v, want 10", reviewerWallet.Balance)
	}

	var txs []model.WalletTransaction
	if err := db.Order("id ASC").Find(&txs).Error; err != nil {
		t.Fatalf("load wallet transactions: %v", err)
	}
	if len(txs) != 2 {
		t.Fatalf("wallet transaction count = %d, want 2", len(txs))
	}

	var payoutTx *model.WalletTransaction
	var awardTx *model.WalletTransaction
	for i := range txs {
		tx := &txs[i]
		switch tx.RefType {
		case model.WalletRefWelfarePayout:
			payoutTx = tx
		case "admin_award":
			awardTx = tx
		}
	}
	if payoutTx == nil {
		t.Fatal("expected welfare payout transaction")
	}
	if payoutTx.Amount != 35 {
		t.Fatalf("wallet tx amount = %v, want 35", payoutTx.Amount)
	}
	if awardTx == nil {
		t.Fatal("expected admin award transaction")
	}
	if awardTx.Amount != 10 {
		t.Fatalf("award wallet tx amount = %v, want 10", awardTx.Amount)
	}
}

func TestAdminReviewApplicationDeliverWithConfiguredPayoutRequiresUserID(t *testing.T) {
	db := newWelfareServiceTestDB(t)
	useWelfareServiceTestDB(t, db)

	payout := 25
	welfare := &model.Welfare{
		Name:          "Starter Pack",
		DistMode:      model.WelfareDistModePerUser,
		PayByFuxiCoin: &payout,
		Status:        model.WelfareStatusActive,
		CreatedBy:     1,
	}
	if err := db.Create(welfare).Error; err != nil {
		t.Fatalf("create welfare: %v", err)
	}

	app := &model.WelfareApplication{
		WelfareID:     welfare.ID,
		CharacterID:   90000001,
		CharacterName: "Pilot One",
		Status:        model.WelfareAppStatusRequested,
	}
	if err := db.Create(app).Error; err != nil {
		t.Fatalf("create application: %v", err)
	}

	svc := NewWelfareService()
	if _, err := svc.AdminReviewApplication(app.ID, 77, adminDeliveryReviewerRoles, &AdminReviewApplicationRequest{Action: "deliver"}); err == nil {
		t.Fatal("expected deliver to fail when payout requires user_id")
	}

	var updated model.WelfareApplication
	if err := db.First(&updated, app.ID).Error; err != nil {
		t.Fatalf("reload application: %v", err)
	}
	if updated.Status != model.WelfareAppStatusRequested {
		t.Fatalf("status = %q, want %q", updated.Status, model.WelfareAppStatusRequested)
	}

	var txs []model.WalletTransaction
	if err := db.Find(&txs).Error; err != nil {
		t.Fatalf("load wallet transactions: %v", err)
	}
	if len(txs) != 0 {
		t.Fatalf("wallet transaction count = %d, want 0", len(txs))
	}
}

func TestAdminReviewApplicationRejectStillStampsReviewerAuditFields(t *testing.T) {
	db := newWelfareServiceTestDB(t)
	useWelfareServiceTestDB(t, db)

	payout := 25
	welfare := &model.Welfare{
		Name:          "Starter Pack",
		DistMode:      model.WelfareDistModePerUser,
		PayByFuxiCoin: &payout,
		Status:        model.WelfareStatusActive,
		CreatedBy:     1,
	}
	if err := db.Create(welfare).Error; err != nil {
		t.Fatalf("create welfare: %v", err)
	}

	userID := uint(42)
	app := &model.WelfareApplication{
		WelfareID:     welfare.ID,
		UserID:        &userID,
		CharacterID:   90000001,
		CharacterName: "Pilot One",
		Status:        model.WelfareAppStatusRequested,
	}
	if err := db.Create(app).Error; err != nil {
		t.Fatalf("create application: %v", err)
	}

	svc := NewWelfareService()
	if _, err := svc.AdminReviewApplication(app.ID, 77, adminDeliveryReviewerRoles, &AdminReviewApplicationRequest{Action: "reject"}); err != nil {
		t.Fatalf("AdminReviewApplication() error = %v", err)
	}

	var updated model.WelfareApplication
	if err := db.First(&updated, app.ID).Error; err != nil {
		t.Fatalf("reload application: %v", err)
	}
	if updated.Status != model.WelfareAppStatusRejected {
		t.Fatalf("status = %q, want %q", updated.Status, model.WelfareAppStatusRejected)
	}
	if updated.ReviewedBy != 77 {
		t.Fatalf("reviewed_by = %d, want 77", updated.ReviewedBy)
	}
	if updated.ReviewedAt == nil {
		t.Fatal("expected reviewed_at to be set")
	}

	var txs []model.WalletTransaction
	if err := db.Find(&txs).Error; err != nil {
		t.Fatalf("load wallet transactions: %v", err)
	}
	if len(txs) != 0 {
		t.Fatalf("wallet transaction count = %d, want 0", len(txs))
	}
}

func TestAdminReviewApplicationSecondDeliverAttemptDoesNotCreateSecondPayoutOrAward(t *testing.T) {
	db := newWelfareServiceTestDB(t)
	useWelfareServiceTestDB(t, db)

	payout := 25
	welfare := &model.Welfare{
		Name:          "Starter Pack",
		DistMode:      model.WelfareDistModePerUser,
		PayByFuxiCoin: &payout,
		Status:        model.WelfareStatusActive,
		CreatedBy:     1,
	}
	if err := db.Create(welfare).Error; err != nil {
		t.Fatalf("create welfare: %v", err)
	}

	userID := uint(42)
	app := &model.WelfareApplication{
		WelfareID:     welfare.ID,
		UserID:        &userID,
		CharacterID:   90000001,
		CharacterName: "Pilot One",
		Status:        model.WelfareAppStatusRequested,
	}
	if err := db.Create(app).Error; err != nil {
		t.Fatalf("create application: %v", err)
	}

	svc := NewWelfareService()
	svc.deliveryMailSender = nil
	if _, err := svc.AdminReviewApplication(app.ID, 77, adminDeliveryReviewerRoles, &AdminReviewApplicationRequest{Action: "deliver"}); err != nil {
		t.Fatalf("first deliver error = %v", err)
	}
	if _, err := svc.AdminReviewApplication(app.ID, 77, adminDeliveryReviewerRoles, &AdminReviewApplicationRequest{Action: "deliver"}); err == nil {
		t.Fatal("expected second deliver attempt to fail")
	}

	var txs []model.WalletTransaction
	if err := db.Find(&txs).Error; err != nil {
		t.Fatalf("load wallet transactions: %v", err)
	}
	if len(txs) != 2 {
		t.Fatalf("wallet transaction count = %d, want 2", len(txs))
	}
}

func TestAdminReviewApplicationDeliverWithZeroAdminAwardSkipsAwardCredit(t *testing.T) {
	db := newWelfareServiceTestDB(t)
	useWelfareServiceTestDB(t, db)

	if err := db.Create(&model.SystemConfig{
		Key:   "pap.admin_award",
		Value: "0",
		Desc:  "admin award",
	}).Error; err != nil {
		t.Fatalf("create system config: %v", err)
	}

	welfare := &model.Welfare{
		Name:      "Starter Pack",
		DistMode:  model.WelfareDistModePerUser,
		Status:    model.WelfareStatusActive,
		CreatedBy: 1,
	}
	if err := db.Create(welfare).Error; err != nil {
		t.Fatalf("create welfare: %v", err)
	}

	userID := uint(42)
	app := &model.WelfareApplication{
		WelfareID:     welfare.ID,
		UserID:        &userID,
		CharacterID:   90000001,
		CharacterName: "Pilot One",
		Status:        model.WelfareAppStatusRequested,
	}
	if err := db.Create(app).Error; err != nil {
		t.Fatalf("create application: %v", err)
	}

	svc := NewWelfareService()
	svc.deliveryMailSender = nil
	if _, err := svc.AdminReviewApplication(app.ID, 77, adminDeliveryReviewerRoles, &AdminReviewApplicationRequest{Action: "deliver"}); err != nil {
		t.Fatalf("AdminReviewApplication() error = %v", err)
	}

	var txs []model.WalletTransaction
	if err := db.Find(&txs).Error; err != nil {
		t.Fatalf("load wallet transactions: %v", err)
	}
	if len(txs) != 0 {
		t.Fatalf("wallet transaction count = %d, want 0", len(txs))
	}
}

func TestAdminReviewApplicationDeliverWithCustomAdminAwardUsesConfiguredAmount(t *testing.T) {
	db := newWelfareServiceTestDB(t)
	useWelfareServiceTestDB(t, db)

	if err := db.Create(&model.SystemConfig{
		Key:   model.SysConfigPAPAdminAward,
		Value: "18",
		Desc:  "admin award",
	}).Error; err != nil {
		t.Fatalf("create system config: %v", err)
	}

	welfare := &model.Welfare{
		Name:      "Starter Pack",
		DistMode:  model.WelfareDistModePerUser,
		Status:    model.WelfareStatusActive,
		CreatedBy: 1,
	}
	if err := db.Create(welfare).Error; err != nil {
		t.Fatalf("create welfare: %v", err)
	}

	userID := uint(42)
	app := &model.WelfareApplication{
		WelfareID:     welfare.ID,
		UserID:        &userID,
		CharacterID:   90000001,
		CharacterName: "Pilot One",
		Status:        model.WelfareAppStatusRequested,
	}
	if err := db.Create(app).Error; err != nil {
		t.Fatalf("create application: %v", err)
	}

	svc := NewWelfareService()
	svc.deliveryMailSender = nil
	if _, err := svc.AdminReviewApplication(app.ID, 77, adminDeliveryReviewerRoles, &AdminReviewApplicationRequest{Action: "deliver"}); err != nil {
		t.Fatalf("AdminReviewApplication() error = %v", err)
	}

	var reviewerWallet model.SystemWallet
	if err := db.Where("user_id = ?", 77).First(&reviewerWallet).Error; err != nil {
		t.Fatalf("load reviewer wallet: %v", err)
	}
	if reviewerWallet.Balance != 18 {
		t.Fatalf("reviewer wallet balance = %v, want 18", reviewerWallet.Balance)
	}
}

func TestAdminReviewApplicationDeliverByWelfareOfficerSkipsAdminAward(t *testing.T) {
	db := newWelfareServiceTestDB(t)
	useWelfareServiceTestDB(t, db)

	payout := 25
	welfare := &model.Welfare{
		Name:          "Starter Pack",
		DistMode:      model.WelfareDistModePerUser,
		PayByFuxiCoin: &payout,
		Status:        model.WelfareStatusActive,
		CreatedBy:     1,
	}
	if err := db.Create(welfare).Error; err != nil {
		t.Fatalf("create welfare: %v", err)
	}

	userID := uint(42)
	app := &model.WelfareApplication{
		WelfareID:     welfare.ID,
		UserID:        &userID,
		CharacterID:   90000001,
		CharacterName: "Pilot One",
		Status:        model.WelfareAppStatusRequested,
	}
	if err := db.Create(app).Error; err != nil {
		t.Fatalf("create application: %v", err)
	}

	svc := NewWelfareService()
	svc.deliveryMailSender = nil
	if _, err := svc.AdminReviewApplication(app.ID, 77, welfareOfficerReviewerRoles, &AdminReviewApplicationRequest{Action: "deliver"}); err != nil {
		t.Fatalf("AdminReviewApplication() error = %v", err)
	}

	var applicantWallet model.SystemWallet
	if err := db.Where("user_id = ?", userID).First(&applicantWallet).Error; err != nil {
		t.Fatalf("load applicant wallet: %v", err)
	}
	if applicantWallet.Balance != 25 {
		t.Fatalf("applicant wallet balance = %v, want 25", applicantWallet.Balance)
	}

	var txs []model.WalletTransaction
	if err := db.Order("id ASC").Find(&txs).Error; err != nil {
		t.Fatalf("load wallet transactions: %v", err)
	}
	if len(txs) != 1 {
		t.Fatalf("wallet transaction count = %d, want 1", len(txs))
	}
	if txs[0].RefType != model.WalletRefWelfarePayout {
		t.Fatalf("wallet tx ref_type = %q, want %q", txs[0].RefType, model.WalletRefWelfarePayout)
	}
	if txs[0].UserID != userID {
		t.Fatalf("wallet tx user_id = %d, want %d", txs[0].UserID, userID)
	}
}

func TestImportWelfareRecordsDoesNotCreateWalletTransactions(t *testing.T) {
	db := newWelfareServiceTestDB(t)
	useWelfareServiceTestDB(t, db)

	payout := 25
	welfare := &model.Welfare{
		Name:          "Starter Pack",
		DistMode:      model.WelfareDistModePerUser,
		PayByFuxiCoin: &payout,
		Status:        model.WelfareStatusActive,
		CreatedBy:     1,
	}
	if err := db.Create(welfare).Error; err != nil {
		t.Fatalf("create welfare: %v", err)
	}

	svc := NewWelfareService()
	count, err := svc.ImportWelfareRecords(&ImportWelfareRecordsRequest{
		WelfareID: welfare.ID,
		CSV:       "Alpha,12345\nBeta,67890",
	})
	if err != nil {
		t.Fatalf("ImportWelfareRecords() error = %v", err)
	}
	if count != 2 {
		t.Fatalf("import count = %d, want 2", count)
	}

	var txs []model.WalletTransaction
	if err := db.Find(&txs).Error; err != nil {
		t.Fatalf("load wallet transactions: %v", err)
	}
	if len(txs) != 0 {
		t.Fatalf("wallet transaction count = %d, want 0", len(txs))
	}
}

func TestAdminCreateWelfareRejectsNegativePayByFuxiCoin(t *testing.T) {
	db := newWelfareServiceTestDB(t)
	useWelfareServiceTestDB(t, db)

	negativePayout := -1
	svc := NewWelfareService()
	err := svc.AdminCreateWelfare(&model.Welfare{
		Name:          "Starter Pack",
		DistMode:      model.WelfareDistModePerUser,
		PayByFuxiCoin: &negativePayout,
		Status:        model.WelfareStatusActive,
		CreatedBy:     1,
	})
	if err == nil {
		t.Fatal("expected create to reject negative pay_by_fuxi_coin")
	}
}

func TestAdminCreateWelfareRejectsNegativeMinimumFuxiLegionYears(t *testing.T) {
	db := newWelfareServiceTestDB(t)
	useWelfareServiceTestDB(t, db)

	negativeYears := -1
	svc := NewWelfareService()
	err := svc.AdminCreateWelfare(&model.Welfare{
		Name:                   "Veteran Reward",
		DistMode:               model.WelfareDistModePerUser,
		MinimumFuxiLegionYears: &negativeYears,
		Status:                 model.WelfareStatusActive,
		CreatedBy:              1,
	})
	if err == nil {
		t.Fatal("expected create to reject negative minimum_fuxi_legion_years")
	}
}

func TestAdminUpdateWelfareRejectsNegativePayByFuxiCoin(t *testing.T) {
	db := newWelfareServiceTestDB(t)
	useWelfareServiceTestDB(t, db)

	welfare := &model.Welfare{
		Name:      "Starter Pack",
		DistMode:  model.WelfareDistModePerUser,
		Status:    model.WelfareStatusActive,
		CreatedBy: 1,
	}
	if err := db.Create(welfare).Error; err != nil {
		t.Fatalf("create welfare: %v", err)
	}

	negativePayout := -1
	svc := NewWelfareService()
	_, err := svc.AdminUpdateWelfare(welfare.ID, &AdminUpdateWelfareRequest{
		Name:             welfare.Name,
		Description:      welfare.Description,
		DistMode:         welfare.DistMode,
		PayByFuxiCoin:    &negativePayout,
		RequireSkillPlan: false,
		Status:           welfare.Status,
	})
	if err == nil {
		t.Fatal("expected update to reject negative pay_by_fuxi_coin")
	}
}

func TestAdminUpdateWelfareRejectsNegativeMinimumFuxiLegionYears(t *testing.T) {
	db := newWelfareServiceTestDB(t)
	useWelfareServiceTestDB(t, db)

	welfare := &model.Welfare{
		Name:      "Veteran Reward",
		DistMode:  model.WelfareDistModePerUser,
		Status:    model.WelfareStatusActive,
		CreatedBy: 1,
	}
	if err := db.Create(welfare).Error; err != nil {
		t.Fatalf("create welfare: %v", err)
	}

	negativeYears := -1
	svc := NewWelfareService()
	_, err := svc.AdminUpdateWelfare(welfare.ID, &AdminUpdateWelfareRequest{
		Name:                   welfare.Name,
		Description:            welfare.Description,
		DistMode:               welfare.DistMode,
		RequireSkillPlan:       false,
		MinimumFuxiLegionYears: &negativeYears,
		Status:                 welfare.Status,
	})
	if err == nil {
		t.Fatal("expected update to reject negative minimum_fuxi_legion_years")
	}
}

func newWelfareServiceTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	dsn := fmt.Sprintf("file:welfare_service_test_%d?mode=memory&cache=shared", time.Now().UnixNano())
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(
		&model.Welfare{},
		&model.WelfareSkillPlan{},
		&model.WelfareApplication{},
		&model.SkillPlan{},
		&model.SystemConfig{},
		&model.User{},
		&model.EveCharacter{},
		&model.SystemWallet{},
		&model.WalletTransaction{},
		&model.AuditEvent{},
	); err != nil {
		t.Fatalf("auto migrate: %v", err)
	}
	return db
}

func useWelfareServiceTestDB(t *testing.T, db *gorm.DB) {
	t.Helper()

	oldDB := global.DB
	global.DB = db
	t.Cleanup(func() {
		global.DB = oldDB
	})
}

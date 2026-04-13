package service

import (
	"amiya-eden/global"
	"amiya-eden/internal/model"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestShouldBlockSelfMentorApplication(t *testing.T) {
	if !shouldBlockSelfMentorApplication(42, 42) {
		t.Fatal("expected self-application to be blocked")
	}
	if shouldBlockSelfMentorApplication(42, 84) {
		t.Fatal("did not expect different mentor to be blocked")
	}
}

func TestFilterMentorCandidateUsersExcludesCurrentUser(t *testing.T) {
	users := []model.User{
		{BaseModel: model.BaseModel{ID: 7}},
		{BaseModel: model.BaseModel{ID: 42}},
		{BaseModel: model.BaseModel{ID: 84}},
	}

	filtered := filterMentorCandidateUsers(42, users)
	if len(filtered) != 2 {
		t.Fatalf("expected 2 users after filtering, got %d", len(filtered))
	}
	if filtered[0].ID != 7 || filtered[1].ID != 84 {
		t.Fatalf("unexpected filtered users: %+v", filtered)
	}
}

func TestCalculateMentorDaysActive(t *testing.T) {
	createdAt := time.Date(2026, time.January, 1, 12, 0, 0, 0, time.UTC)
	lastLogin := createdAt.Add(36 * time.Hour)

	if got := calculateMentorDaysActive(createdAt, &lastLogin); got != 1 {
		t.Fatalf("expected 1 active day, got %d", got)
	}
	if got := calculateMentorDaysActive(createdAt, nil); got != 0 {
		t.Fatalf("expected nil last login to produce 0 days, got %d", got)
	}
	beforeCreate := createdAt.Add(-24 * time.Hour)
	if got := calculateMentorDaysActive(createdAt, &beforeCreate); got != 0 {
		t.Fatalf("expected negative duration to clamp to 0, got %d", got)
	}
}

func TestBuildMentorCandidateIncludesContact(t *testing.T) {
	user := model.User{
		BaseModel: model.BaseModel{ID: 99},
		Nickname:  "Teacher",
		QQ:        "123456",
		DiscordID: "teacher#0001",
	}
	primaryChar := model.EveCharacter{
		CharacterID:   777,
		CharacterName: "Helpful Mentor",
	}

	got := buildMentorCandidate(user, primaryChar, 3)

	if got.MentorQQ != "123456" {
		t.Fatalf("expected mentor QQ to be preserved, got %q", got.MentorQQ)
	}
	if got.MentorDiscordID != "teacher#0001" {
		t.Fatalf("expected mentor Discord ID to be preserved, got %q", got.MentorDiscordID)
	}
	if got.ActiveMenteeCount != 3 {
		t.Fatalf("expected active mentee count to be preserved, got %d", got.ActiveMenteeCount)
	}
	payload, err := json.Marshal(got)
	if err != nil {
		t.Fatalf("marshal mentor candidate: %v", err)
	}
	var raw map[string]any
	if err := json.Unmarshal(payload, &raw); err != nil {
		t.Fatalf("unmarshal mentor candidate: %v", err)
	}
	if _, exists := raw["mentor_portrait_url"]; exists {
		t.Fatalf("expected mentor candidate to omit mentor_portrait_url, got %#v", raw["mentor_portrait_url"])
	}
}

func TestBuildMentorRelationshipViewIncludesMentorContact(t *testing.T) {
	rel := model.MentorMenteeRelationship{
		BaseModel:    model.BaseModel{ID: 15},
		MenteeUserID: 7,
		MentorUserID: 9,
		Status:       model.MentorRelationStatusActive,
	}
	appliedAt := time.Date(2026, time.April, 2, 8, 0, 0, 0, time.UTC)
	respondedAt := appliedAt.Add(2 * time.Hour)
	rel.AppliedAt = appliedAt
	rel.RespondedAt = &respondedAt

	mentorUser := model.User{
		BaseModel: model.BaseModel{ID: 9},
		Nickname:  "Teacher",
		QQ:        "123456",
		DiscordID: "teacher#0001",
	}
	menteeUser := model.User{
		BaseModel: model.BaseModel{ID: 7},
		Nickname:  "Student",
	}
	mentorChar := model.EveCharacter{
		CharacterID:   91,
		CharacterName: "Helpful Mentor",
	}
	menteeChar := model.EveCharacter{
		CharacterID:   71,
		CharacterName: "Curious Mentee",
	}

	got := buildMentorRelationshipView(rel, mentorUser, menteeUser, mentorChar, menteeChar)

	if got.MentorQQ != "123456" {
		t.Fatalf("expected mentor QQ to be preserved, got %q", got.MentorQQ)
	}
	if got.MentorDiscordID != "teacher#0001" {
		t.Fatalf("expected mentor Discord ID to be preserved, got %q", got.MentorDiscordID)
	}
	if got.MentorCharacterName != "Helpful Mentor" {
		t.Fatalf("expected mentor name to be preserved, got %q", got.MentorCharacterName)
	}
	payload, err := json.Marshal(got)
	if err != nil {
		t.Fatalf("marshal mentor relationship view: %v", err)
	}
	var raw map[string]any
	if err := json.Unmarshal(payload, &raw); err != nil {
		t.Fatalf("unmarshal mentor relationship view: %v", err)
	}
	if _, exists := raw["mentor_portrait_url"]; exists {
		t.Fatalf("expected relationship view to omit mentor_portrait_url, got %#v", raw["mentor_portrait_url"])
	}
	if _, exists := raw["mentee_portrait_url"]; exists {
		t.Fatalf("expected relationship view to omit mentee_portrait_url, got %#v", raw["mentee_portrait_url"])
	}
}

func TestMentorServiceListMyMenteesRejectsUnknownStatus(t *testing.T) {
	db := newMentorServiceTestDB(t)
	useMentorServiceTestDB(t, db)

	svc := NewMentorService()
	_, _, err := svc.ListMyMentees(42, "completed", 1, 20)
	if err == nil {
		t.Fatal("expected ListMyMentees to reject an unknown mentor relationship status")
	}
}

func TestMentorServiceAdminListAllRelationshipsBatchesProfileLookups(t *testing.T) {
	db := newMentorServiceTestDB(t)
	useMentorServiceTestDB(t, db)

	mentor := model.User{BaseModel: model.BaseModel{ID: 11}, Nickname: "Mentor", QQ: "123", DiscordID: "mentor#1", PrimaryCharacterID: 910001}
	menteeOne := model.User{BaseModel: model.BaseModel{ID: 21}, Nickname: "Mentee One", PrimaryCharacterID: 920001}
	menteeTwo := model.User{BaseModel: model.BaseModel{ID: 22}, Nickname: "Mentee Two", PrimaryCharacterID: 920002}
	users := []model.User{mentor, menteeOne, menteeTwo}
	if err := db.Create(&users).Error; err != nil {
		t.Fatalf("create users: %v", err)
	}

	characters := []model.EveCharacter{
		{CharacterID: 910001, CharacterName: "Helpful Mentor", UserID: mentor.ID},
		{CharacterID: 920001, CharacterName: "Curious Mentee", UserID: menteeOne.ID},
		{CharacterID: 920002, CharacterName: "Another Mentee", UserID: menteeTwo.ID},
	}
	if err := db.Create(&characters).Error; err != nil {
		t.Fatalf("create characters: %v", err)
	}

	appliedAt := time.Date(2026, time.April, 13, 9, 0, 0, 0, time.UTC)
	relationships := []model.MentorMenteeRelationship{
		{MentorUserID: mentor.ID, MenteeUserID: menteeOne.ID, Status: model.MentorRelationStatusActive, AppliedAt: appliedAt},
		{MentorUserID: mentor.ID, MenteeUserID: menteeTwo.ID, Status: model.MentorRelationStatusPending, AppliedAt: appliedAt.Add(-time.Hour)},
	}
	if err := db.Create(&relationships).Error; err != nil {
		t.Fatalf("create relationships: %v", err)
	}

	queryCount := 0
	const callbackName = "count_admin_relationship_queries"
	if err := db.Callback().Query().Before("gorm:query").Register(callbackName, func(tx *gorm.DB) {
		queryCount++
	}); err != nil {
		t.Fatalf("register query counter: %v", err)
	}
	t.Cleanup(func() {
		_ = db.Callback().Query().Remove(callbackName)
	})

	svc := NewMentorService()
	views, total, err := svc.AdminListAllRelationships("", "", 1, 20)
	if err != nil {
		t.Fatalf("AdminListAllRelationships() error = %v", err)
	}
	if total != 2 {
		t.Fatalf("AdminListAllRelationships() total = %d, want 2", total)
	}
	if len(views) != 2 {
		t.Fatalf("AdminListAllRelationships() len = %d, want 2", len(views))
	}
	if queryCount > 4 {
		t.Fatalf("AdminListAllRelationships() query count = %d, want <= 4", queryCount)
	}

	viewByMenteeID := make(map[uint]MentorRelationshipView, len(views))
	for _, view := range views {
		viewByMenteeID[view.MenteeUserID] = view
	}
	if viewByMenteeID[menteeOne.ID].MentorCharacterName != "Helpful Mentor" {
		t.Fatalf("mentee one mentor character = %q, want %q", viewByMenteeID[menteeOne.ID].MentorCharacterName, "Helpful Mentor")
	}
	if viewByMenteeID[menteeTwo.ID].MenteeCharacterName != "Another Mentee" {
		t.Fatalf("mentee two character = %q, want %q", viewByMenteeID[menteeTwo.ID].MenteeCharacterName, "Another Mentee")
	}
}

func newMentorServiceTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	dsn := fmt.Sprintf("file:mentor_service_test_%d?mode=memory&cache=shared", time.Now().UnixNano())
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&model.User{}, &model.EveCharacter{}, &model.MentorMenteeRelationship{}); err != nil {
		t.Fatalf("auto migrate mentor service models: %v", err)
	}
	return db
}

func useMentorServiceTestDB(t *testing.T, db *gorm.DB) {
	t.Helper()

	oldDB := global.DB
	global.DB = db
	t.Cleanup(func() {
		global.DB = oldDB
	})
}

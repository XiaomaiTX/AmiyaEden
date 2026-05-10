package repository

import (
	"amiya-eden/global"
	"amiya-eden/internal/model"
	"fmt"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var ticketRepoTestDBSeq uint64

func setupTicketRepositoryTestDB(t *testing.T) {
	t.Helper()
	name := strings.NewReplacer("/", "_", "\\", "_", " ", "_", ":", "_").Replace(t.Name())
	dsn := fmt.Sprintf(
		"file:ticket_repository_test_%s_%d_%d?mode=memory&cache=shared",
		name,
		time.Now().UnixNano(),
		atomic.AddUint64(&ticketRepoTestDBSeq, 1),
	)
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&model.User{}, &model.EveCharacter{}, &model.Ticket{}, &model.TicketCategory{}, &model.TicketReply{}, &model.TicketStatusHistory{}); err != nil {
		t.Fatalf("auto migrate: %v", err)
	}
	oldDB := global.DB
	global.DB = db
	t.Cleanup(func() { global.DB = oldDB })
}

func seedTicketRepositoryData(t *testing.T) (model.Ticket, model.Ticket) {
	t.Helper()
	users := []model.User{
		{BaseModel: model.BaseModel{ID: 1001}, Nickname: "User1001"},
		{BaseModel: model.BaseModel{ID: 1002}, Nickname: "User1002"},
	}
	if err := global.DB.Create(&users).Error; err != nil {
		t.Fatalf("create users: %v", err)
	}
	chars := []model.EveCharacter{
		{CharacterID: 9001001, CharacterName: "Char1001", UserID: 1001},
		{CharacterID: 9001002, CharacterName: "Char1002", UserID: 1002},
	}
	if err := global.DB.Create(&chars).Error; err != nil {
		t.Fatalf("create characters: %v", err)
	}
	if err := global.DB.Model(&model.User{}).
		Where("id = ?", 1001).
		Update("primary_character_id", chars[0].CharacterID).Error; err != nil {
		t.Fatalf("set user1001 primary character: %v", err)
	}
	if err := global.DB.Model(&model.User{}).
		Where("id = ?", 1002).
		Update("primary_character_id", chars[1].CharacterID).Error; err != nil {
		t.Fatalf("set user1002 primary character: %v", err)
	}

	category := model.TicketCategory{Name: "平台反馈", NameEN: "Platform Feedback", Enabled: true}
	if err := global.DB.Create(&category).Error; err != nil {
		t.Fatalf("create category: %v", err)
	}

	t1 := model.Ticket{UserID: 1001, CategoryID: category.ID, Title: "登录失败", Description: "无法进入游戏", Status: model.TicketStatusPending, Priority: model.TicketPriorityMedium}
	t2 := model.Ticket{UserID: 1002, CategoryID: category.ID, Title: "合同异常", Description: "合同描述检索测试", Status: model.TicketStatusInProgress, Priority: model.TicketPriorityHigh}
	if err := global.DB.Create(&t1).Error; err != nil {
		t.Fatalf("create ticket1: %v", err)
	}
	if err := global.DB.Create(&t2).Error; err != nil {
		t.Fatalf("create ticket2: %v", err)
	}

	return t1, t2
}

func TestTicketRepositoryListRepliesRespectsInternalFlag(t *testing.T) {
	setupTicketRepositoryTestDB(t)
	repo := NewTicketRepository()
	t1, _ := seedTicketRepositoryData(t)

	replies := []model.TicketReply{
		{TicketID: t1.ID, UserID: 1001, Content: "用户补充", IsInternal: false},
		{TicketID: t1.ID, UserID: 9001, Content: "内部备注", IsInternal: true},
	}
	if err := global.DB.Create(&replies).Error; err != nil {
		t.Fatalf("create replies: %v", err)
	}

	userVisible, err := repo.ListReplies(t1.ID, false)
	if err != nil {
		t.Fatalf("ListReplies(includeInternal=false) error: %v", err)
	}
	if len(userVisible) != 1 {
		t.Fatalf("user visible replies = %d, want 1", len(userVisible))
	}

	allReplies, err := repo.ListReplies(t1.ID, true)
	if err != nil {
		t.Fatalf("ListReplies(includeInternal=true) error: %v", err)
	}
	if len(allReplies) != 2 {
		t.Fatalf("all replies = %d, want 2", len(allReplies))
	}
}

func TestTicketRepositoryListTicketsAdminSupportsKeywordAndStatus(t *testing.T) {
	setupTicketRepositoryTestDB(t)
	repo := NewTicketRepository()
	_, _ = seedTicketRepositoryData(t)

	list, total, err := repo.ListTicketsAdmin(TicketListFilter{Status: model.TicketStatusInProgress, Keyword: "检索测试"}, 1, 20)
	if err != nil {
		t.Fatalf("ListTicketsAdmin() error: %v", err)
	}
	if total != 1 {
		t.Fatalf("total = %d, want 1", total)
	}
	if len(list) != 1 {
		t.Fatalf("list len = %d, want 1", len(list))
	}
	if list[0].Title != "合同异常" {
		t.Fatalf("title = %q, want %q", list[0].Title, "合同异常")
	}
}

func TestTicketRepositoryListTicketsAdminSupportsPriorityAndHidesCompletedByDefault(t *testing.T) {
	setupTicketRepositoryTestDB(t)
	repo := NewTicketRepository()
	t1, _ := seedTicketRepositoryData(t)

	completed := model.Ticket{
		UserID:      1001,
		CategoryID:  t1.CategoryID,
		Title:       "已完结工单",
		Description: "should be hidden by default",
		Status:      model.TicketStatusCompleted,
		Priority:    model.TicketPriorityLow,
	}
	if err := global.DB.Create(&completed).Error; err != nil {
		t.Fatalf("create completed ticket: %v", err)
	}

	list, _, err := repo.ListTicketsAdmin(TicketListFilter{}, 1, 20)
	if err != nil {
		t.Fatalf("ListTicketsAdmin() default error: %v", err)
	}
	for _, item := range list {
		if item.Status == model.TicketStatusCompleted {
			t.Fatal("completed tickets should be hidden by default")
		}
	}

	priorityList, _, err := repo.ListTicketsAdmin(TicketListFilter{Priority: model.TicketPriorityHigh}, 1, 20)
	if err != nil {
		t.Fatalf("ListTicketsAdmin(priority) error: %v", err)
	}
	if len(priorityList) != 1 || priorityList[0].Priority != model.TicketPriorityHigh {
		t.Fatalf("priority filter mismatch, got len=%d", len(priorityList))
	}
}

func TestTicketRepositoryCountByStatusIncludesDefaultKeys(t *testing.T) {
	setupTicketRepositoryTestDB(t)
	repo := NewTicketRepository()
	_, _ = seedTicketRepositoryData(t)

	byStatus, err := repo.CountByStatus()
	if err != nil {
		t.Fatalf("CountByStatus() error: %v", err)
	}

	if byStatus[model.TicketStatusPending] != 1 {
		t.Fatalf("pending count = %d, want 1", byStatus[model.TicketStatusPending])
	}
	if byStatus[model.TicketStatusInProgress] != 1 {
		t.Fatalf("in_progress count = %d, want 1", byStatus[model.TicketStatusInProgress])
	}
	if byStatus[model.TicketStatusCompleted] != 0 {
		t.Fatalf("completed count = %d, want 0", byStatus[model.TicketStatusCompleted])
	}
}

func TestTicketRepositoryCountBadgeTicketsForAdminCountsPendingAndOwnedInProgress(t *testing.T) {
	setupTicketRepositoryTestDB(t)
	repo := NewTicketRepository()
	ticket1, ticket2 := seedTicketRepositoryData(t)

	adminID := uint(7001)
	assigned := adminID
	otherAssigned := adminID + 1
	if err := global.DB.Create(&model.Ticket{
		UserID:      1003,
		CategoryID:  ticket1.CategoryID,
		Title:       "extra pending",
		Description: "extra pending ticket",
		Status:      model.TicketStatusPending,
		Priority:    model.TicketPriorityMedium,
	}).Error; err != nil {
		t.Fatalf("create extra pending ticket: %v", err)
	}
	if err := global.DB.Create(&model.Ticket{
		UserID:      1004,
		CategoryID:  ticket2.CategoryID,
		Title:       "assigned to current admin",
		Description: "assigned in progress ticket",
		Status:      model.TicketStatusInProgress,
		Priority:    model.TicketPriorityMedium,
		HandledBy:   &assigned,
	}).Error; err != nil {
		t.Fatalf("create assigned ticket: %v", err)
	}
	if err := global.DB.Create(&model.Ticket{
		UserID:      1005,
		CategoryID:  ticket2.CategoryID,
		Title:       "assigned to other admin",
		Description: "should not count",
		Status:      model.TicketStatusInProgress,
		Priority:    model.TicketPriorityMedium,
		HandledBy:   &otherAssigned,
	}).Error; err != nil {
		t.Fatalf("create other assigned ticket: %v", err)
	}

	got, err := repo.CountBadgeTicketsForAdmin(adminID)
	if err != nil {
		t.Fatalf("CountBadgeTicketsForAdmin() error: %v", err)
	}
	if got != 3 {
		t.Fatalf("CountBadgeTicketsForAdmin() = %d, want 3", got)
	}
}

func TestTicketRepositoryNicknameFallbackToPrimaryCharacter(t *testing.T) {
	setupTicketRepositoryTestDB(t)
	repo := NewTicketRepository()

	userWithChar := model.User{BaseModel: model.BaseModel{ID: 2001}, Nickname: ""}
	userNoChar := model.User{BaseModel: model.BaseModel{ID: 2002}, Nickname: ""}
	if err := global.DB.Create(&[]model.User{userWithChar, userNoChar}).Error; err != nil {
		t.Fatalf("create users: %v", err)
	}
	char := model.EveCharacter{
		CharacterID:   9200001,
		CharacterName: "Fallback Character",
		UserID:        2001,
	}
	if err := global.DB.Create(&char).Error; err != nil {
		t.Fatalf("create character: %v", err)
	}
	if err := global.DB.Model(&model.User{}).
		Where("id = ?", 2001).
		Update("primary_character_id", char.CharacterID).Error; err != nil {
		t.Fatalf("set primary character: %v", err)
	}

	category := model.TicketCategory{Name: "cat", NameEN: "cat", Enabled: true}
	if err := global.DB.Create(&category).Error; err != nil {
		t.Fatalf("create category: %v", err)
	}
	handledBy := uint(2002)
	ticket := model.Ticket{
		UserID:      2001,
		CategoryID:  category.ID,
		Title:       "title",
		Description: "desc",
		Status:      model.TicketStatusPending,
		Priority:    model.TicketPriorityMedium,
		HandledBy:   &handledBy,
	}
	if err := global.DB.Create(&ticket).Error; err != nil {
		t.Fatalf("create ticket: %v", err)
	}
	if err := global.DB.Create(&model.TicketReply{
		TicketID: ticket.ID, UserID: 2001, Content: "reply", IsInternal: false,
	}).Error; err != nil {
		t.Fatalf("create reply: %v", err)
	}
	if err := global.DB.Create(&model.TicketStatusHistory{
		TicketID: ticket.ID, ToStatus: model.TicketStatusPending, ChangedBy: 2001,
	}).Error; err != nil {
		t.Fatalf("create status history: %v", err)
	}

	adminList, _, err := repo.ListTicketsAdmin(TicketListFilter{}, 1, 20)
	if err != nil {
		t.Fatalf("ListTicketsAdmin() error: %v", err)
	}
	if len(adminList) != 1 {
		t.Fatalf("admin list len = %d, want 1", len(adminList))
	}
	if adminList[0].UserNickname != "Fallback Character" {
		t.Fatalf("user_nickname = %q, want %q", adminList[0].UserNickname, "Fallback Character")
	}
	if adminList[0].HandledByNickname != "-" {
		t.Fatalf("handled_by_nickname = %q, want -", adminList[0].HandledByNickname)
	}

	replies, err := repo.ListReplies(ticket.ID, true)
	if err != nil {
		t.Fatalf("ListReplies() error: %v", err)
	}
	if len(replies) != 1 {
		t.Fatalf("replies len = %d, want 1", len(replies))
	}
	if replies[0].UserNickname != "Fallback Character" {
		t.Fatalf("reply user_nickname = %q, want %q", replies[0].UserNickname, "Fallback Character")
	}

	history, err := repo.ListStatusHistories(ticket.ID)
	if err != nil {
		t.Fatalf("ListStatusHistories() error: %v", err)
	}
	if len(history) != 1 {
		t.Fatalf("history len = %d, want 1", len(history))
	}
	if history[0].ChangedByNickname != "Fallback Character" {
		t.Fatalf("changed_by_nickname = %q, want %q", history[0].ChangedByNickname, "Fallback Character")
	}
}

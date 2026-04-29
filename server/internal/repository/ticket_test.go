package repository

import (
	"amiya-eden/global"
	"amiya-eden/internal/model"
	"fmt"
	"testing"
	"time"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func setupTicketRepositoryTestDB(t *testing.T) {
	t.Helper()
	dsn := fmt.Sprintf("file:ticket_repository_test_%d?mode=memory&cache=shared", time.Now().UnixNano())
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&model.Ticket{}, &model.TicketCategory{}, &model.TicketReply{}, &model.TicketStatusHistory{}); err != nil {
		t.Fatalf("auto migrate: %v", err)
	}
	oldDB := global.DB
	global.DB = db
	t.Cleanup(func() { global.DB = oldDB })
}

func seedTicketRepositoryData(t *testing.T) (model.Ticket, model.Ticket) {
	t.Helper()
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

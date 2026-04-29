package service

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

func setupTicketServiceTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := fmt.Sprintf("file:ticket_service_test_%d?mode=memory&cache=shared", time.Now().UnixNano())
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}

	if err := db.AutoMigrate(&model.Ticket{}, &model.TicketCategory{}, &model.TicketReply{}, &model.TicketStatusHistory{}); err != nil {
		t.Fatalf("auto migrate: %v", err)
	}

	oldDB := global.DB
	global.DB = db
	t.Cleanup(func() {
		global.DB = oldDB
	})

	return db
}

func seedTicketCategory(t *testing.T) model.TicketCategory {
	t.Helper()
	category := model.TicketCategory{
		Name:    "账号问题",
		NameEN:  "Account Issues",
		Enabled: true,
	}
	if err := global.DB.Create(&category).Error; err != nil {
		t.Fatalf("seed category: %v", err)
	}
	return category
}

func TestTicketServiceCreateTicketDefaultsAndHistory(t *testing.T) {
	setupTicketServiceTestDB(t)
	category := seedTicketCategory(t)

	svc := NewTicketService()
	ticket, err := svc.CreateTicket(1001, category.ID, "  登录异常 ", "  无法登录客户端 ", "")
	if err != nil {
		t.Fatalf("CreateTicket() error = %v, want nil", err)
	}

	if ticket.Title != "登录异常" {
		t.Fatalf("Title = %q, want %q", ticket.Title, "登录异常")
	}
	if ticket.Priority != model.TicketPriorityMedium {
		t.Fatalf("Priority = %q, want %q", ticket.Priority, model.TicketPriorityMedium)
	}
	if ticket.Status != model.TicketStatusPending {
		t.Fatalf("Status = %q, want %q", ticket.Status, model.TicketStatusPending)
	}

	history, err := svc.ListStatusHistoryAsAdmin(ticket.ID)
	if err != nil {
		t.Fatalf("ListStatusHistoryAsAdmin() error = %v, want nil", err)
	}
	if len(history) != 1 {
		t.Fatalf("history count = %d, want 1", len(history))
	}
	if history[0].ToStatus != model.TicketStatusPending {
		t.Fatalf("history ToStatus = %q, want %q", history[0].ToStatus, model.TicketStatusPending)
	}
}

func TestTicketServiceGetMyTicketPermissionDeniedForOtherUser(t *testing.T) {
	setupTicketServiceTestDB(t)
	category := seedTicketCategory(t)
	svc := NewTicketService()

	ticket, err := svc.CreateTicket(2001, category.ID, "角色卡住", "请求解卡", model.TicketPriorityLow)
	if err != nil {
		t.Fatalf("CreateTicket() error = %v, want nil", err)
	}

	_, err = svc.GetMyTicket(2002, ticket.ID)
	if err == nil {
		t.Fatal("GetMyTicket() expected permission error, got nil")
	}
	if err.Error() != "无权限访问该工单" {
		t.Fatalf("GetMyTicket() err = %q, want %q", err.Error(), "无权限访问该工单")
	}
}

func TestTicketServiceReplyVisibilitySeparatesInternalNotes(t *testing.T) {
	setupTicketServiceTestDB(t)
	category := seedTicketCategory(t)
	svc := NewTicketService()

	ticket, err := svc.CreateTicket(3001, category.ID, "合同问题", "合同无法接收", model.TicketPriorityHigh)
	if err != nil {
		t.Fatalf("CreateTicket() error = %v, want nil", err)
	}

	if _, err := svc.AddReplyAsUser(3001, ticket.ID, "已补充截图"); err != nil {
		t.Fatalf("AddReplyAsUser() error = %v, want nil", err)
	}
	if _, err := svc.AddReplyAsAdmin(9001, ticket.ID, "收到，正在核查", false); err != nil {
		t.Fatalf("AddReplyAsAdmin(false) error = %v, want nil", err)
	}
	if _, err := svc.AddReplyAsAdmin(9001, ticket.ID, "内部备注：疑似配置异常", true); err != nil {
		t.Fatalf("AddReplyAsAdmin(true) error = %v, want nil", err)
	}

	userReplies, err := svc.ListRepliesAsUser(3001, ticket.ID)
	if err != nil {
		t.Fatalf("ListRepliesAsUser() error = %v, want nil", err)
	}
	if len(userReplies) != 2 {
		t.Fatalf("user replies count = %d, want 2", len(userReplies))
	}

	adminReplies, err := svc.ListRepliesAsAdmin(ticket.ID)
	if err != nil {
		t.Fatalf("ListRepliesAsAdmin() error = %v, want nil", err)
	}
	if len(adminReplies) != 3 {
		t.Fatalf("admin replies count = %d, want 3", len(adminReplies))
	}
}

func TestTicketServiceUpdateStatusSetsHandledAndClosed(t *testing.T) {
	setupTicketServiceTestDB(t)
	category := seedTicketCategory(t)
	svc := NewTicketService()

	ticket, err := svc.CreateTicket(4001, category.ID, "赏金结算问题", "金额未到账", model.TicketPriorityMedium)
	if err != nil {
		t.Fatalf("CreateTicket() error = %v, want nil", err)
	}

	updated, err := svc.UpdateStatusAsAdmin(9100, ticket.ID, model.TicketStatusInProgress)
	if err != nil {
		t.Fatalf("UpdateStatusAsAdmin(in_progress) error = %v, want nil", err)
	}
	if updated.HandledBy == nil || *updated.HandledBy != 9100 {
		t.Fatalf("HandledBy = %v, want 9100", updated.HandledBy)
	}
	if updated.HandledAt == nil {
		t.Fatal("HandledAt = nil, want non-nil")
	}
	if updated.ClosedAt != nil {
		t.Fatal("ClosedAt should be nil in in_progress")
	}

	updated, err = svc.UpdateStatusAsAdmin(9100, ticket.ID, model.TicketStatusCompleted)
	if err != nil {
		t.Fatalf("UpdateStatusAsAdmin(completed) error = %v, want nil", err)
	}
	if updated.ClosedAt == nil {
		t.Fatal("ClosedAt = nil, want non-nil")
	}

	history, err := svc.ListStatusHistoryAsAdmin(ticket.ID)
	if err != nil {
		t.Fatalf("ListStatusHistoryAsAdmin() error = %v, want nil", err)
	}
	if len(history) != 3 {
		t.Fatalf("history count = %d, want 3", len(history))
	}
}

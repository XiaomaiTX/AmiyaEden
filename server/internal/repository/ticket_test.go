package repository

import (
	"amiya-eden/global"
	"amiya-eden/internal/model"
	"fmt"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/glebarez/sqlite"
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
	category := model.TicketCategory{Name: "平台反馈", NameEN: "Platform Feedback", Enabled: true}
	if err := global.DB.Create(&category).Error; err != nil {
		t.Fatalf("create category: %v", err)
	}

	t1 := model.Ticket{UserID: 1001, CategoryID: category.ID, Title: "登录失败", Description: "无法进入游戏", Status: model.TicketStatusPending}
	t2 := model.Ticket{UserID: 1002, CategoryID: category.ID, Title: "合同异常", Description: "合同描述检索测试", Status: model.TicketStatusInProgress}
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
	if err := global.DB.Create(&model.User{
		BaseModel:          model.BaseModel{ID: 1001},
		Nickname:           "Ticket User",
		PrimaryCharacterID: 981001,
	}).Error; err != nil {
		t.Fatalf("create user #1001: %v", err)
	}
	if err := global.DB.Create(&model.EveCharacter{
		UserID:        1001,
		CharacterID:   981001,
		CharacterName: "Ticket Character",
	}).Error; err != nil {
		t.Fatalf("create character #1001: %v", err)
	}
	if err := global.DB.Create(&model.User{
		BaseModel:          model.BaseModel{ID: 9001},
		Nickname:           "Admin User",
		PrimaryCharacterID: 989001,
	}).Error; err != nil {
		t.Fatalf("create user #9001: %v", err)
	}
	if err := global.DB.Create(&model.EveCharacter{
		UserID:        9001,
		CharacterID:   989001,
		CharacterName: "Admin Character",
	}).Error; err != nil {
		t.Fatalf("create character #9001: %v", err)
	}

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
	if userVisible[0].ReplyUserNickname != "Ticket User" {
		t.Fatalf("ReplyUserNickname = %q, want %q", userVisible[0].ReplyUserNickname, "Ticket User")
	}
	if userVisible[0].UserNickname != "Ticket User" {
		t.Fatalf("UserNickname = %q, want %q", userVisible[0].UserNickname, "Ticket User")
	}

	allReplies, err := repo.ListReplies(t1.ID, true)
	if err != nil {
		t.Fatalf("ListReplies(includeInternal=true) error: %v", err)
	}
	if len(allReplies) != 2 {
		t.Fatalf("all replies = %d, want 2", len(allReplies))
	}
}

func TestTicketRepositoryListStatusHistoriesIncludesOperatorNames(t *testing.T) {
	setupTicketRepositoryTestDB(t)
	repo := NewTicketRepository()
	ticket, _ := seedTicketRepositoryData(t)
	operatorID := uint(9001)
	if err := global.DB.Create(&model.User{
		BaseModel:          model.BaseModel{ID: operatorID},
		Nickname:           "Operator Nick",
		PrimaryCharacterID: 991001,
	}).Error; err != nil {
		t.Fatalf("create operator user: %v", err)
	}
	if err := global.DB.Create(&model.EveCharacter{
		UserID:        operatorID,
		CharacterID:   991001,
		CharacterName: "Operator Character",
	}).Error; err != nil {
		t.Fatalf("create operator character: %v", err)
	}
	if err := repo.AddStatusHistory(ticket.ID, model.TicketStatusPending, model.TicketStatusInProgress, operatorID); err != nil {
		t.Fatalf("AddStatusHistory() error: %v", err)
	}
	history, err := repo.ListStatusHistories(ticket.ID)
	if err != nil {
		t.Fatalf("ListStatusHistories() error: %v", err)
	}
	if len(history) != 1 {
		t.Fatalf("history len = %d, want 1", len(history))
	}
	if history[0].ChangedByNickname != "Operator Nick" {
		t.Fatalf("ChangedByNickname = %q, want %q", history[0].ChangedByNickname, "Operator Nick")
	}
	if history[0].ChangedByName != "Operator Nick" {
		t.Fatalf("ChangedByName = %q, want %q", history[0].ChangedByName, "Operator Nick")
	}
	if history[0].ChangedByCharacterName != "Operator Character" {
		t.Fatalf("ChangedByCharacterName = %q, want %q", history[0].ChangedByCharacterName, "Operator Character")
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

func TestTicketRepositoryListTicketsAdminSupportsMultipleStatuses(t *testing.T) {
	setupTicketRepositoryTestDB(t)
	repo := NewTicketRepository()
	_, _ = seedTicketRepositoryData(t)
	list, total, err := repo.ListTicketsAdmin(TicketListFilter{Statuses: []string{model.TicketStatusPending, model.TicketStatusInProgress}}, 1, 20)
	if err != nil {
		t.Fatalf("ListTicketsAdmin() error: %v", err)
	}
	if total != 2 || len(list) != 2 {
		t.Fatalf("active tickets total/list = %d/%d, want 2/2", total, len(list))
	}
}

func TestTicketRepositoryListTicketsAdminIncludesSubmitterAndCategoryNames(t *testing.T) {
	setupTicketRepositoryTestDB(t)
	repo := NewTicketRepository()
	ticket, _ := seedTicketRepositoryData(t)
	if err := global.DB.Create(&model.User{
		BaseModel:          model.BaseModel{ID: ticket.UserID},
		Nickname:           "Alpha User",
		PrimaryCharacterID: 990001,
	}).Error; err != nil {
		t.Fatalf("create user: %v", err)
	}
	if err := global.DB.Create(&model.EveCharacter{
		UserID:        ticket.UserID,
		CharacterID:   990001,
		CharacterName: "Alpha Character",
	}).Error; err != nil {
		t.Fatalf("create character: %v", err)
	}
	list, total, err := repo.ListTicketsAdmin(TicketListFilter{Category: ticket.CategoryID}, 1, 20)
	if err != nil {
		t.Fatalf("ListTicketsAdmin() error: %v", err)
	}
	if total != 2 {
		t.Fatalf("total = %d, want 2", total)
	}
	if len(list) != 2 {
		t.Fatalf("list len = %d, want 2", len(list))
	}
	if list[1].RequesterName != "Alpha User" {
		t.Fatalf("RequesterName = %q, want %q", list[1].RequesterName, "Alpha User")
	}
	if list[1].UserNickname != "Alpha User" {
		t.Fatalf("UserNickname = %q, want %q", list[1].UserNickname, "Alpha User")
	}
	if list[1].RequesterCharacterName != "Alpha Character" {
		t.Fatalf("RequesterCharacterName = %q, want %q", list[1].RequesterCharacterName, "Alpha Character")
	}
	if list[1].CategoryName != "平台反馈" || list[1].CategoryNameEN != "Platform Feedback" {
		t.Fatalf("category names = %q/%q, want 平台反馈/Platform Feedback", list[1].CategoryName, list[1].CategoryNameEN)
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

func TestTicketRepositoryListTicketsAdminIncludesHandledByNickname(t *testing.T) {
	setupTicketRepositoryTestDB(t)
	repo := NewTicketRepository()
	ticket1, _ := seedTicketRepositoryData(t)

	handlerID := uint(7001)
	if err := global.DB.Create(&model.User{
		BaseModel:          model.BaseModel{ID: handlerID},
		Nickname:           "Handler User",
		PrimaryCharacterID: 997001,
	}).Error; err != nil {
		t.Fatalf("create handler user: %v", err)
	}
	if err := global.DB.Create(&model.EveCharacter{
		UserID:        handlerID,
		CharacterID:   997001,
		CharacterName: "Handler Character",
	}).Error; err != nil {
		t.Fatalf("create handler character: %v", err)
	}

	ticket1.HandledBy = &handlerID
	ticket1.Status = model.TicketStatusInProgress
	if err := global.DB.Save(&ticket1).Error; err != nil {
		t.Fatalf("update ticket handled_by: %v", err)
	}

	list, total, err := repo.ListTicketsAdmin(TicketListFilter{Status: model.TicketStatusInProgress}, 1, 20)
	if err != nil {
		t.Fatalf("ListTicketsAdmin() error: %v", err)
	}
	if total != 2 || len(list) != 2 {
		t.Fatalf("in-progress total/list = %d/%d, want 2/2", total, len(list))
	}

	var got string
	for _, item := range list {
		if item.ID == ticket1.ID {
			got = item.HandledByNickname
			break
		}
	}
	if got != "Handler User" {
		t.Fatalf("HandledByNickname = %q, want %q", got, "Handler User")
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
	}).Error; err != nil {
		t.Fatalf("create extra pending ticket: %v", err)
	}
	if err := global.DB.Create(&model.Ticket{
		UserID:      1004,
		CategoryID:  ticket2.CategoryID,
		Title:       "assigned to current admin",
		Description: "assigned in progress ticket",
		Status:      model.TicketStatusInProgress,
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

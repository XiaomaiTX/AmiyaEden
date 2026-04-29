package service

import (
	"amiya-eden/internal/model"
	"amiya-eden/internal/repository"
	"errors"
	"strings"
	"time"

	"gorm.io/gorm"
)

var (
	errTicketNotFound       = errors.New("工单不存在")
	errTicketNoPermission   = errors.New("无权限访问该工单")
	errInvalidTicketStatus  = errors.New("无效的工单状态")
	errInvalidTicketPrio    = errors.New("无效的工单优先级")
	errTicketCategoryAbsent = errors.New("工单分类不存在")
)

type TicketService struct {
	repo *repository.TicketRepository
}

func NewTicketService() *TicketService {
	return &TicketService{repo: repository.NewTicketRepository()}
}

func normalizeTicketStatus(status string) (string, error) {
	switch strings.TrimSpace(status) {
	case model.TicketStatusPending:
		return model.TicketStatusPending, nil
	case model.TicketStatusInProgress:
		return model.TicketStatusInProgress, nil
	case model.TicketStatusCompleted:
		return model.TicketStatusCompleted, nil
	default:
		return "", errInvalidTicketStatus
	}
}

func normalizeTicketPriority(priority string) (string, error) {
	if strings.TrimSpace(priority) == "" {
		return model.TicketPriorityMedium, nil
	}
	switch strings.TrimSpace(priority) {
	case model.TicketPriorityLow:
		return model.TicketPriorityLow, nil
	case model.TicketPriorityMedium:
		return model.TicketPriorityMedium, nil
	case model.TicketPriorityHigh:
		return model.TicketPriorityHigh, nil
	default:
		return "", errInvalidTicketPrio
	}
}

func (s *TicketService) CreateTicket(userID, categoryID uint, title, description, priority string) (*model.Ticket, error) {
	title = strings.TrimSpace(title)
	description = strings.TrimSpace(description)
	if title == "" || description == "" {
		return nil, errors.New("标题和描述不能为空")
	}
	_, err := s.repo.GetCategoryByID(categoryID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errTicketCategoryAbsent
		}
		return nil, err
	}
	normalizedPriority, err := normalizeTicketPriority(priority)
	if err != nil {
		return nil, err
	}
	ticket := &model.Ticket{
		UserID:      userID,
		CategoryID:  categoryID,
		Title:       title,
		Description: description,
		Status:      model.TicketStatusPending,
		Priority:    normalizedPriority,
	}
	if err := s.repo.CreateTicket(ticket); err != nil {
		return nil, err
	}
	_ = s.repo.AddStatusHistory(ticket.ID, "", model.TicketStatusPending, userID)
	return ticket, nil
}

func (s *TicketService) ListMyTickets(userID uint, status string, page, pageSize int) ([]model.Ticket, int64, error) {
	normalizePageRequest(&page, &pageSize, 20, 100)
	return s.repo.ListTicketsByUser(userID, strings.TrimSpace(status), page, pageSize)
}

func (s *TicketService) GetMyTicket(userID, ticketID uint) (*model.Ticket, error) {
	ticket, err := s.repo.GetTicketByID(ticketID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errTicketNotFound
		}
		return nil, err
	}
	if ticket.UserID != userID {
		return nil, errTicketNoPermission
	}
	return ticket, nil
}

func (s *TicketService) GetAdminTicket(ticketID uint) (*model.Ticket, error) {
	ticket, err := s.repo.GetTicketByID(ticketID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errTicketNotFound
		}
		return nil, err
	}
	return ticket, nil
}

func (s *TicketService) ListTicketsAdmin(filter repository.TicketListFilter, page, pageSize int) ([]model.Ticket, int64, error) {
	normalizePageRequest(&page, &pageSize, 20, 100)
	filter.Keyword = strings.TrimSpace(filter.Keyword)
	filter.Status = strings.TrimSpace(filter.Status)
	return s.repo.ListTicketsAdmin(filter, page, pageSize)
}

func (s *TicketService) AddReplyAsUser(userID, ticketID uint, content string) (*model.TicketReply, error) {
	content = strings.TrimSpace(content)
	if content == "" {
		return nil, errors.New("回复内容不能为空")
	}
	ticket, err := s.GetMyTicket(userID, ticketID)
	if err != nil {
		return nil, err
	}
	reply := &model.TicketReply{
		TicketID:   ticket.ID,
		UserID:     userID,
		Content:    content,
		IsInternal: false,
	}
	if err := s.repo.CreateReply(reply); err != nil {
		return nil, err
	}
	return reply, nil
}

func (s *TicketService) AddReplyAsAdmin(adminID, ticketID uint, content string, isInternal bool) (*model.TicketReply, error) {
	content = strings.TrimSpace(content)
	if content == "" {
		return nil, errors.New("回复内容不能为空")
	}
	ticket, err := s.GetAdminTicket(ticketID)
	if err != nil {
		return nil, err
	}
	reply := &model.TicketReply{
		TicketID:   ticket.ID,
		UserID:     adminID,
		Content:    content,
		IsInternal: isInternal,
	}
	if err := s.repo.CreateReply(reply); err != nil {
		return nil, err
	}
	return reply, nil
}

func (s *TicketService) ListRepliesAsUser(userID, ticketID uint) ([]model.TicketReply, error) {
	if _, err := s.GetMyTicket(userID, ticketID); err != nil {
		return nil, err
	}
	return s.repo.ListReplies(ticketID, false)
}

func (s *TicketService) ListRepliesAsAdmin(ticketID uint) ([]model.TicketReply, error) {
	if _, err := s.GetAdminTicket(ticketID); err != nil {
		return nil, err
	}
	return s.repo.ListReplies(ticketID, true)
}

func (s *TicketService) UpdateStatusAsAdmin(adminID, ticketID uint, status string) (*model.Ticket, error) {
	normalizedStatus, err := normalizeTicketStatus(status)
	if err != nil {
		return nil, err
	}
	ticket, err := s.GetAdminTicket(ticketID)
	if err != nil {
		return nil, err
	}
	now := time.Now()
	fromStatus := ticket.Status
	ticket.Status = normalizedStatus
	if normalizedStatus == model.TicketStatusInProgress || normalizedStatus == model.TicketStatusCompleted {
		ticket.HandledBy = &adminID
		if ticket.HandledAt == nil {
			ticket.HandledAt = &now
		}
	}
	if normalizedStatus == model.TicketStatusCompleted {
		ticket.ClosedAt = &now
	} else {
		ticket.ClosedAt = nil
	}
	if err := s.repo.UpdateTicket(ticket); err != nil {
		return nil, err
	}
	if fromStatus != normalizedStatus {
		_ = s.repo.AddStatusHistory(ticket.ID, fromStatus, normalizedStatus, adminID)
	}
	return ticket, nil
}

func (s *TicketService) UpdatePriorityAsAdmin(ticketID uint, priority string) (*model.Ticket, error) {
	normalizedPriority, err := normalizeTicketPriority(priority)
	if err != nil {
		return nil, err
	}
	ticket, err := s.GetAdminTicket(ticketID)
	if err != nil {
		return nil, err
	}
	ticket.Priority = normalizedPriority
	if err := s.repo.UpdateTicket(ticket); err != nil {
		return nil, err
	}
	return ticket, nil
}

func (s *TicketService) ListStatusHistoryAsAdmin(ticketID uint) ([]model.TicketStatusHistory, error) {
	if _, err := s.GetAdminTicket(ticketID); err != nil {
		return nil, err
	}
	return s.repo.ListStatusHistories(ticketID)
}

func (s *TicketService) ListCategories(enabledOnly bool) ([]model.TicketCategory, error) {
	return s.repo.ListCategories(enabledOnly)
}

func (s *TicketService) CreateCategory(category *model.TicketCategory) error {
	if strings.TrimSpace(category.Name) == "" || strings.TrimSpace(category.NameEN) == "" {
		return errors.New("分类中英文名称不能为空")
	}
	return s.repo.CreateCategory(category)
}

func (s *TicketService) UpdateCategory(id uint, req *model.TicketCategory) (*model.TicketCategory, error) {
	category, err := s.repo.GetCategoryByID(id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errTicketCategoryAbsent
		}
		return nil, err
	}
	if strings.TrimSpace(req.Name) != "" {
		category.Name = strings.TrimSpace(req.Name)
	}
	if strings.TrimSpace(req.NameEN) != "" {
		category.NameEN = strings.TrimSpace(req.NameEN)
	}
	category.Description = strings.TrimSpace(req.Description)
	category.SortOrder = req.SortOrder
	category.Enabled = req.Enabled
	if err := s.repo.UpdateCategory(category); err != nil {
		return nil, err
	}
	return category, nil
}

func (s *TicketService) DeleteCategory(id uint) error {
	return s.repo.DeleteCategory(id)
}

func (s *TicketService) GetStatistics() (map[string]any, error) {
	byStatus, err := s.repo.CountByStatus()
	if err != nil {
		return nil, err
	}
	byCategory, err := s.repo.CountByCategory()
	if err != nil {
		return nil, err
	}
	day7, err := s.repo.CountCreatedSince(time.Now().AddDate(0, 0, -7))
	if err != nil {
		return nil, err
	}
	day30, err := s.repo.CountCreatedSince(time.Now().AddDate(0, 0, -30))
	if err != nil {
		return nil, err
	}
	total := byStatus[model.TicketStatusPending] + byStatus[model.TicketStatusInProgress] + byStatus[model.TicketStatusCompleted]
	return map[string]any{
		"total":        total,
		"status":       byStatus,
		"category":     byCategory,
		"recent_7d":    day7,
		"recent_30d":   day30,
		"pendingCount": byStatus[model.TicketStatusPending],
	}, nil
}


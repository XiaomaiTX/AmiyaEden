package service

import (
	"amiya-eden/internal/model"
	"amiya-eden/internal/repository"
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"gorm.io/gorm"
)

var (
	errTicketNotFound       = errors.New("工单不存在")
	errTicketNoPermission   = errors.New("无权限访问该工单")
	errInvalidTicketStatus  = errors.New("无效的工单状态")
	errTicketCategoryAbsent = errors.New("工单分类不存在")
)

type TicketService struct {
	repo     *repository.TicketRepository
	auditSvc *AuditService
}

type TicketListFilter struct {
	Status   string
	Statuses []string
	Category uint
	UserID   uint
	Keyword  string
}

func NewTicketService() *TicketService {
	return &TicketService{repo: repository.NewTicketRepository(), auditSvc: NewAuditService()}
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

func (s *TicketService) CreateTicket(userID, categoryID uint, title, description string) (*model.Ticket, error) {
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
	ticket := &model.Ticket{
		UserID:      userID,
		CategoryID:  categoryID,
		Title:       title,
		Description: description,
		Status:      model.TicketStatusPending,
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

func (s *TicketService) GetAdminTicketDetail(ticketID uint) (*model.TicketListItem, error) {
	ticket, err := s.repo.GetTicketListItemByID(ticketID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errTicketNotFound
		}
		return nil, err
	}
	return ticket, nil
}

func (s *TicketService) ListTicketsAdmin(filter TicketListFilter, page, pageSize int) ([]model.TicketListItem, int64, error) {
	normalizePageRequest(&page, &pageSize, 20, 100)
	filter.Keyword = strings.TrimSpace(filter.Keyword)
	filter.Status = strings.TrimSpace(filter.Status)
	if filter.Status != "" {
		rawStatuses := strings.Split(filter.Status, ",")
		if len(rawStatuses) > 1 {
			filter.Status = ""
			filter.Statuses = make([]string, 0, len(rawStatuses))
			for _, rawStatus := range rawStatuses {
				normalizedStatus, err := normalizeTicketStatus(rawStatus)
				if err != nil {
					return nil, 0, err
				}
				filter.Statuses = append(filter.Statuses, normalizedStatus)
			}
		} else {
			normalizedStatus, err := normalizeTicketStatus(filter.Status)
			if err != nil {
				return nil, 0, err
			}
			filter.Status = normalizedStatus
		}
	}
	return s.repo.ListTicketsAdmin(repository.TicketListFilter{
		Status:   filter.Status,
		Statuses: filter.Statuses,
		Category: filter.Category,
		UserID:   filter.UserID,
		Keyword:  filter.Keyword,
	}, page, pageSize)
}

func (s *TicketService) AddReplyAsUser(userID, ticketID uint, content string) (*model.TicketReplyItem, error) {
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
	return s.repo.GetReplyByID(reply.ID)
}

func (s *TicketService) AddReplyAsAdmin(adminID, ticketID uint, content string, isInternal bool) (*model.TicketReplyItem, error) {
	content = strings.TrimSpace(content)
	if content == "" {
		s.recordTicketAudit("ticket_reply", adminID, ticketID, model.AuditResultFailed, map[string]any{"reason": "empty_content"})
		return nil, errors.New("回复内容不能为空")
	}
	ticket, err := s.GetAdminTicket(ticketID)
	if err != nil {
		s.recordTicketAudit("ticket_reply", adminID, ticketID, model.AuditResultFailed, map[string]any{"reason": err.Error()})
		return nil, err
	}
	reply := &model.TicketReply{
		TicketID:   ticket.ID,
		UserID:     adminID,
		Content:    content,
		IsInternal: isInternal,
	}
	if err := s.repo.CreateReply(reply); err != nil {
		s.recordTicketAudit("ticket_reply", adminID, ticketID, model.AuditResultFailed, map[string]any{"reason": err.Error()})
		return nil, err
	}
	item, err := s.repo.GetReplyByID(reply.ID)
	if err != nil {
		s.recordTicketAudit("ticket_reply", adminID, ticketID, model.AuditResultFailed, map[string]any{"reason": err.Error()})
		return nil, err
	}
	s.recordTicketAudit("ticket_reply", adminID, ticketID, model.AuditResultSuccess, map[string]any{"is_internal": isInternal, "reply_id": item.ID})
	return item, nil
}

func (s *TicketService) ListRepliesAsUser(userID, ticketID uint) ([]model.TicketReplyItem, error) {
	if _, err := s.GetMyTicket(userID, ticketID); err != nil {
		return nil, err
	}
	return s.repo.ListReplies(ticketID, false)
}

func (s *TicketService) ListRepliesAsAdmin(ticketID uint) ([]model.TicketReplyItem, error) {
	if _, err := s.GetAdminTicket(ticketID); err != nil {
		return nil, err
	}
	return s.repo.ListReplies(ticketID, true)
}

func (s *TicketService) UpdateStatusAsAdmin(adminID, ticketID uint, status string) (*model.Ticket, error) {
	normalizedStatus, err := normalizeTicketStatus(status)
	if err != nil {
		s.recordTicketAudit("ticket_status_update", adminID, ticketID, model.AuditResultFailed, map[string]any{"reason": err.Error()})
		return nil, err
	}
	ticket, err := s.GetAdminTicket(ticketID)
	if err != nil {
		s.recordTicketAudit("ticket_status_update", adminID, ticketID, model.AuditResultFailed, map[string]any{"reason": err.Error()})
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
		s.recordTicketAudit("ticket_status_update", adminID, ticketID, model.AuditResultFailed, map[string]any{"reason": err.Error()})
		return nil, err
	}
	if fromStatus != normalizedStatus {
		_ = s.repo.AddStatusHistory(ticket.ID, fromStatus, normalizedStatus, adminID)
	}
	s.recordTicketAudit("ticket_status_update", adminID, ticketID, model.AuditResultSuccess, map[string]any{"before_status": fromStatus, "after_status": normalizedStatus})
	return ticket, nil
}

func (s *TicketService) ListStatusHistoryAsAdmin(ticketID uint) ([]model.TicketStatusHistoryItem, error) {
	if _, err := s.GetAdminTicket(ticketID); err != nil {
		return nil, err
	}
	return s.repo.ListStatusHistories(ticketID)
}

func (s *TicketService) ListCategories(enabledOnly bool) ([]model.TicketCategory, error) {
	return s.repo.ListCategories(enabledOnly)
}

func (s *TicketService) CreateCategory(operatorID uint, category *model.TicketCategory) error {
	if strings.TrimSpace(category.Name) == "" || strings.TrimSpace(category.NameEN) == "" {
		s.recordTicketCategoryAudit("ticket_category_create", operatorID, 0, model.AuditResultFailed, map[string]any{"reason": "empty_name"})
		return errors.New("分类中英文名称不能为空")
	}
	if err := s.repo.CreateCategory(category); err != nil {
		s.recordTicketCategoryAudit("ticket_category_create", operatorID, 0, model.AuditResultFailed, map[string]any{"reason": err.Error()})
		return err
	}
	s.recordTicketCategoryAudit("ticket_category_create", operatorID, category.ID, model.AuditResultSuccess, map[string]any{"name": category.Name, "name_en": category.NameEN})
	return nil
}

func (s *TicketService) UpdateCategory(operatorID, id uint, req *model.TicketCategory) (*model.TicketCategory, error) {
	category, err := s.repo.GetCategoryByID(id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			s.recordTicketCategoryAudit("ticket_category_update", operatorID, id, model.AuditResultFailed, map[string]any{"reason": "category_not_found"})
			return nil, errTicketCategoryAbsent
		}
		s.recordTicketCategoryAudit("ticket_category_update", operatorID, id, model.AuditResultFailed, map[string]any{"reason": err.Error()})
		return nil, err
	}
	before := map[string]any{"name": category.Name, "name_en": category.NameEN, "description": category.Description, "sort_order": category.SortOrder, "enabled": category.Enabled}
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
		s.recordTicketCategoryAudit("ticket_category_update", operatorID, id, model.AuditResultFailed, map[string]any{"reason": err.Error()})
		return nil, err
	}
	s.recordTicketCategoryAudit("ticket_category_update", operatorID, id, model.AuditResultSuccess, map[string]any{"before": before, "after": map[string]any{"name": category.Name, "name_en": category.NameEN, "description": category.Description, "sort_order": category.SortOrder, "enabled": category.Enabled}})
	return category, nil
}

func (s *TicketService) DeleteCategory(operatorID, id uint) error {
	if err := s.repo.DeleteCategory(id); err != nil {
		s.recordTicketCategoryAudit("ticket_category_delete", operatorID, id, model.AuditResultFailed, map[string]any{"reason": err.Error()})
		return err
	}
	s.recordTicketCategoryAudit("ticket_category_delete", operatorID, id, model.AuditResultSuccess, map[string]any{"deleted_category_id": id})
	return nil
}

func (s *TicketService) recordTicketAudit(action string, actorID, ticketID uint, result string, details map[string]any) {
	if s.auditSvc == nil {
		return
	}
	_ = s.auditSvc.RecordEvent(context.Background(), AuditRecordInput{
		Category:     "ticket_admin",
		Action:       action,
		ActorUserID:  actorID,
		ResourceType: "ticket",
		ResourceID:   fmt.Sprintf("%d", ticketID),
		Result:       result,
		Details:      details,
	})
}

func (s *TicketService) recordTicketCategoryAudit(action string, actorID, categoryID uint, result string, details map[string]any) {
	if s.auditSvc == nil {
		return
	}
	_ = s.auditSvc.RecordEvent(context.Background(), AuditRecordInput{
		Category:     "ticket_admin",
		Action:       action,
		ActorUserID:  actorID,
		ResourceType: "ticket_category",
		ResourceID:   fmt.Sprintf("%d", categoryID),
		Result:       result,
		Details:      details,
	})
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

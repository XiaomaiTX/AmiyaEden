package repository

import (
	"amiya-eden/global"
	"amiya-eden/internal/model"
	"time"

	"gorm.io/gorm"
)

type TicketRepository struct{}

func NewTicketRepository() *TicketRepository {
	return &TicketRepository{}
}

type TicketListFilter struct {
	Status   string
	Priority string
	Category uint
	UserID   uint
	Keyword  string
}

func (r *TicketRepository) CreateTicket(ticket *model.Ticket) error {
	return global.DB.Create(ticket).Error
}

func (r *TicketRepository) GetTicketByID(id uint) (*model.Ticket, error) {
	var ticket model.Ticket
	if err := r.ticketBaseQuery(global.DB).
		Where("ticket.id = ?", id).
		First(&ticket).Error; err != nil {
		return nil, err
	}
	return &ticket, nil
}

func (r *TicketRepository) UpdateTicket(ticket *model.Ticket) error {
	return global.DB.Save(ticket).Error
}

func (r *TicketRepository) ListTicketsByUser(userID uint, status string, page, pageSize int) ([]model.Ticket, int64, error) {
	var tickets []model.Ticket
	var total int64
	query := global.DB.Model(&model.Ticket{}).Where("ticket.user_id = ?", userID)
	if status != "" {
		query = query.Where("ticket.status = ?", status)
	}
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	err := r.ticketBaseQuery(query).
		Order("ticket.updated_at DESC, ticket.id DESC").
		Offset((page - 1) * pageSize).
		Limit(pageSize).
		Find(&tickets).Error
	return tickets, total, err
}

func (r *TicketRepository) ListTicketsAdmin(filter TicketListFilter, page, pageSize int) ([]model.Ticket, int64, error) {
	var tickets []model.Ticket
	var total int64
	query := global.DB.Model(&model.Ticket{})
	if filter.Status != "" {
		query = query.Where("ticket.status = ?", filter.Status)
	} else {
		query = query.Where("ticket.status <> ?", model.TicketStatusCompleted)
	}
	if filter.Priority != "" {
		query = query.Where("ticket.priority = ?", filter.Priority)
	}
	if filter.Category > 0 {
		query = query.Where("ticket.category_id = ?", filter.Category)
	}
	if filter.UserID > 0 {
		query = query.Where("ticket.user_id = ?", filter.UserID)
	}
	if filter.Keyword != "" {
		query = applyKeywordLikeFilter(query, filter.Keyword, "LOWER(title) LIKE ?", "LOWER(description) LIKE ?")
	}
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	err := r.ticketBaseQuery(query).
		Order("ticket.updated_at DESC, ticket.id DESC").
		Offset((page - 1) * pageSize).
		Limit(pageSize).
		Find(&tickets).Error
	return tickets, total, err
}

func (r *TicketRepository) CreateReply(reply *model.TicketReply) error {
	return global.DB.Create(reply).Error
}

func (r *TicketRepository) ListReplies(ticketID uint, includeInternal bool) ([]model.TicketReply, error) {
	var replies []model.TicketReply
	query := global.DB.Model(&model.TicketReply{}).Where("ticket_reply.ticket_id = ?", ticketID)
	if !includeInternal {
		query = query.Where("ticket_reply.is_internal = ?", false)
	}
	err := query.
		Select(`
			ticket_reply.*,
			COALESCE(NULLIF(reply_user.nickname, ''), '-') AS user_nickname
		`).
		Joins(`LEFT JOIN "user" AS reply_user ON reply_user.id = ticket_reply.user_id`).
		Order("ticket_reply.created_at ASC, ticket_reply.id ASC").
		Find(&replies).Error
	return replies, err
}

func (r *TicketRepository) AddStatusHistory(ticketID uint, fromStatus, toStatus string, changedBy uint) error {
	h := &model.TicketStatusHistory{
		TicketID:   ticketID,
		FromStatus: fromStatus,
		ToStatus:   toStatus,
		ChangedBy:  changedBy,
	}
	return global.DB.Create(h).Error
}

func (r *TicketRepository) ListStatusHistories(ticketID uint) ([]model.TicketStatusHistory, error) {
	var list []model.TicketStatusHistory
	err := global.DB.Model(&model.TicketStatusHistory{}).
		Where("ticket_status_history.ticket_id = ?", ticketID).
		Select(`
			ticket_status_history.*,
			COALESCE(NULLIF(changed_user.nickname, ''), '-') AS changed_by_nickname
		`).
		Joins(`LEFT JOIN "user" AS changed_user ON changed_user.id = ticket_status_history.changed_by`).
		Order("ticket_status_history.changed_at ASC, ticket_status_history.id ASC").
		Find(&list).Error
	return list, err
}

func (r *TicketRepository) ticketBaseQuery(db *gorm.DB) *gorm.DB {
	return db.Select(`
			ticket.*,
			COALESCE(NULLIF(ticket_user.nickname, ''), '-') AS user_nickname,
			COALESCE(NULLIF(handled_user.nickname, ''), '-') AS handled_by_nickname
		`).
		Joins(`LEFT JOIN "user" AS ticket_user ON ticket_user.id = ticket.user_id`).
		Joins(`LEFT JOIN "user" AS handled_user ON handled_user.id = ticket.handled_by`)
}

func (r *TicketRepository) ListCategories(enabledOnly bool) ([]model.TicketCategory, error) {
	var categories []model.TicketCategory
	query := global.DB.Model(&model.TicketCategory{})
	if enabledOnly {
		query = query.Where("enabled = ?", true)
	}
	err := query.Order("sort_order ASC, id ASC").Find(&categories).Error
	return categories, err
}

func (r *TicketRepository) CreateCategory(category *model.TicketCategory) error {
	return global.DB.Create(category).Error
}

func (r *TicketRepository) GetCategoryByID(id uint) (*model.TicketCategory, error) {
	var category model.TicketCategory
	if err := global.DB.First(&category, id).Error; err != nil {
		return nil, err
	}
	return &category, nil
}

func (r *TicketRepository) UpdateCategory(category *model.TicketCategory) error {
	return global.DB.Save(category).Error
}

func (r *TicketRepository) DeleteCategory(id uint) error {
	return global.DB.Delete(&model.TicketCategory{}, id).Error
}

func (r *TicketRepository) CountByStatus() (map[string]int64, error) {
	type row struct {
		Status string
		Count  int64
	}
	var rows []row
	err := global.DB.Model(&model.Ticket{}).Select("status, COUNT(*) AS count").Group("status").Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	result := map[string]int64{
		model.TicketStatusPending:    0,
		model.TicketStatusInProgress: 0,
		model.TicketStatusCompleted:  0,
	}
	for _, item := range rows {
		result[item.Status] = item.Count
	}
	return result, nil
}

func (r *TicketRepository) CountByCategory() (map[uint]int64, error) {
	type row struct {
		CategoryID uint
		Count      int64
	}
	var rows []row
	err := global.DB.Model(&model.Ticket{}).Select("category_id, COUNT(*) AS count").Group("category_id").Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	result := make(map[uint]int64, len(rows))
	for _, item := range rows {
		result[item.CategoryID] = item.Count
	}
	return result, nil
}

func (r *TicketRepository) CountCreatedSince(since time.Time) (int64, error) {
	var count int64
	err := global.DB.Model(&model.Ticket{}).Where("created_at >= ?", since).Count(&count).Error
	return count, err
}

func (r *TicketRepository) CountBadgeTicketsForAdmin(userID uint) (int64, error) {
	var count int64
	err := global.DB.Model(&model.Ticket{}).
		Where("status = ?", model.TicketStatusPending).
		Or(global.DB.Where("status = ? AND handled_by = ?", model.TicketStatusInProgress, userID)).
		Count(&count).Error
	return count, err
}

func (r *TicketRepository) InTx(fn func(tx *gorm.DB) error) error {
	return global.DB.Transaction(fn)
}

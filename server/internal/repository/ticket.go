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
	Category uint
	UserID   uint
	Keyword  string
}

func (r *TicketRepository) CreateTicket(ticket *model.Ticket) error {
	return global.DB.Create(ticket).Error
}

func (r *TicketRepository) GetTicketByID(id uint) (*model.Ticket, error) {
	var ticket model.Ticket
	if err := global.DB.First(&ticket, id).Error; err != nil {
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
	query := global.DB.Model(&model.Ticket{}).Where("user_id = ?", userID)
	if status != "" {
		query = query.Where("status = ?", status)
	}
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	err := query.Order("updated_at DESC, id DESC").Offset((page - 1) * pageSize).Limit(pageSize).Find(&tickets).Error
	return tickets, total, err
}

func (r *TicketRepository) ListTicketsAdmin(filter TicketListFilter, page, pageSize int) ([]model.Ticket, int64, error) {
	var tickets []model.Ticket
	var total int64
	query := global.DB.Model(&model.Ticket{})
	if filter.Status != "" {
		query = query.Where("status = ?", filter.Status)
	}
	if filter.Category > 0 {
		query = query.Where("category_id = ?", filter.Category)
	}
	if filter.UserID > 0 {
		query = query.Where("user_id = ?", filter.UserID)
	}
	if filter.Keyword != "" {
		query = applyKeywordLikeFilter(query, filter.Keyword, "LOWER(title) LIKE ?", "LOWER(description) LIKE ?")
	}
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	err := query.Order("updated_at DESC, id DESC").Offset((page - 1) * pageSize).Limit(pageSize).Find(&tickets).Error
	return tickets, total, err
}

func (r *TicketRepository) CreateReply(reply *model.TicketReply) error {
	return global.DB.Create(reply).Error
}

func (r *TicketRepository) ListReplies(ticketID uint, includeInternal bool) ([]model.TicketReply, error) {
	var replies []model.TicketReply
	query := global.DB.Where("ticket_id = ?", ticketID)
	if !includeInternal {
		query = query.Where("is_internal = ?", false)
	}
	err := query.Order("created_at ASC, id ASC").Find(&replies).Error
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
	err := global.DB.Where("ticket_id = ?", ticketID).Order("changed_at ASC, id ASC").Find(&list).Error
	return list, err
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

func (r *TicketRepository) InTx(fn func(tx *gorm.DB) error) error {
	return global.DB.Transaction(fn)
}


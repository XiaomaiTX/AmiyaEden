package repository

import (
	"amiya-eden/global"
	"amiya-eden/internal/model"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// ShopRepository 商店数据访问层
type ShopRepository struct{}

func NewShopRepository() *ShopRepository {
	return &ShopRepository{}
}

func buildPendingBadgeShopOrderCountQuery(db *gorm.DB) *gorm.DB {
	return db.Model(&model.ShopOrder{}).
		Where("status = ?", model.OrderStatusRequested)
}

func (r *ShopRepository) CountPendingOrders() (int64, error) {
	var count int64
	err := buildPendingBadgeShopOrderCountQuery(global.DB).Count(&count).Error
	return count, err
}

func (r *ShopRepository) CountDeliveredByReviewers(userIDs []uint) (map[uint]int64, error) {
	result := make(map[uint]int64, len(userIDs))
	if len(userIDs) == 0 {
		return result, nil
	}

	type row struct {
		ReviewedBy uint
		Count      int64
	}

	var rows []row
	err := global.DB.Model(&model.ShopOrder{}).
		Select("reviewed_by, COUNT(*) AS count").
		Where("reviewed_by IN ? AND status = ?", userIDs, model.OrderStatusDelivered).
		Group("reviewed_by").
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}

	for _, row := range rows {
		result[row.ReviewedBy] = row.Count
	}
	return result, nil
}

// ─────────────────────────────────────────────
//  商品
// ─────────────────────────────────────────────

// CreateProduct 创建商品
func (r *ShopRepository) CreateProduct(p *model.ShopProduct) error {
	return global.DB.Create(p).Error
}

// UpdateProduct 更新商品
func (r *ShopRepository) UpdateProduct(p *model.ShopProduct) error {
	return global.DB.Save(p).Error
}

// DeleteProduct 删除商品（软删除）
func (r *ShopRepository) DeleteProduct(id uint) error {
	return global.DB.Delete(&model.ShopProduct{}, id).Error
}

// GetProductByID 根据 ID 获取商品
func (r *ShopRepository) GetProductByID(id uint) (*model.ShopProduct, error) {
	var p model.ShopProduct
	if err := global.DB.First(&p, id).Error; err != nil {
		return nil, err
	}
	return &p, nil
}

// ProductFilter 商品查询筛选
type ProductFilter struct {
	Status *int8
	Type   string
	Name   string
}

// ListProducts 分页查询商品
func (r *ShopRepository) ListProducts(page, pageSize int, filter ProductFilter) ([]model.ShopProduct, int64, error) {
	var list []model.ShopProduct
	var total int64
	offset := (page - 1) * pageSize

	db := global.DB.Model(&model.ShopProduct{})
	if filter.Status != nil {
		db = db.Where("status = ?", *filter.Status)
	}
	if filter.Type != "" {
		db = db.Where("type = ?", filter.Type)
	}
	if filter.Name != "" {
		db = db.Where("name ILIKE ?", "%"+filter.Name+"%")
	}

	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	if err := db.Order("sort_order DESC, id DESC").Offset(offset).Limit(pageSize).Find(&list).Error; err != nil {
		return nil, 0, err
	}
	return list, total, nil
}

// DecrStock 扣减库存（事务中使用，stock > 0 才扣减）
func (r *ShopRepository) DecrStockTx(tx *gorm.DB, productID uint, qty int) error {
	result := tx.Model(&model.ShopProduct{}).
		Where("id = ? AND stock >= ?", productID, qty).
		Update("stock", gorm.Expr("stock - ?", qty))
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound // 库存不足
	}
	return nil
}

// RestoreStockTx 恢复库存（仅对有限库存商品生效；无限库存或已删除商品跳过）
func (r *ShopRepository) RestoreStockTx(tx *gorm.DB, productID uint, qty int) error {
	return tx.Model(&model.ShopProduct{}).
		Where("id = ? AND stock >= 0", productID).
		UpdateColumn("stock", gorm.Expr("stock + ?", qty)).Error
}

// ─────────────────────────────────────────────
//  订单
// ─────────────────────────────────────────────

// CreateOrder 创建订单
func (r *ShopRepository) CreateOrder(o *model.ShopOrder) error {
	return global.DB.Create(o).Error
}

// CreateOrderTx 在事务中创建订单
func (r *ShopRepository) CreateOrderTx(tx *gorm.DB, o *model.ShopOrder) error {
	return tx.Create(o).Error
}

// UpdateOrder 更新订单
func (r *ShopRepository) UpdateOrder(o *model.ShopOrder) error {
	return global.DB.Save(o).Error
}

// UpdateOrderTx 在事务中更新订单
func (r *ShopRepository) UpdateOrderTx(tx *gorm.DB, o *model.ShopOrder) error {
	return tx.Save(o).Error
}

// UpdateOrderReviewTx 在事务中基于当前状态更新订单审核结果，避免并发审核相互覆盖。
func (r *ShopRepository) UpdateOrderReviewTx(
	tx *gorm.DB,
	orderID uint,
	expectedCurrentStatus string,
	nextStatus string,
	operatorID uint,
	reviewedAt time.Time,
	remark string,
) (bool, error) {
	result := tx.Model(&model.ShopOrder{}).
		Where("id = ? AND status = ?", orderID, expectedCurrentStatus).
		Updates(map[string]any{
			"status":        nextStatus,
			"reviewed_by":   operatorID,
			"reviewed_at":   reviewedAt,
			"review_remark": remark,
		})
	if result.Error != nil {
		return false, result.Error
	}
	return result.RowsAffected > 0, nil
}

// GetOrderByID 根据 ID 获取订单
func (r *ShopRepository) GetOrderByID(id uint) (*model.ShopOrder, error) {
	var o model.ShopOrder
	if err := global.DB.First(&o, id).Error; err != nil {
		return nil, err
	}
	return &o, nil
}

// GetOrderByIDForUpdateTx 在事务中按 ID 获取订单并加锁，避免并发审核重复处理。
func (r *ShopRepository) GetOrderByIDForUpdateTx(tx *gorm.DB, id uint) (*model.ShopOrder, error) {
	var o model.ShopOrder
	if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&o, id).Error; err != nil {
		return nil, err
	}
	return &o, nil
}

// GetOrderByOrderNo 根据订单号获取订单
func (r *ShopRepository) GetOrderByOrderNo(orderNo string) (*model.ShopOrder, error) {
	var o model.ShopOrder
	if err := global.DB.Where("order_no = ?", orderNo).First(&o).Error; err != nil {
		return nil, err
	}
	return &o, nil
}

// OrderFilter 订单查询筛选
type OrderFilter struct {
	UserID    *uint
	ProductID *uint
	Status    string   // 单状态精确匹配
	Statuses  []string // 多状态 IN 查询（优先于 Status）
	Keyword   string   // 商品名、主人物名或昵称模糊搜索
}

// ListOrders 分页查询订单
func (r *ShopRepository) ListOrders(page, pageSize int, filter OrderFilter) ([]model.ShopOrder, int64, error) {
	var list []model.ShopOrder
	var total int64
	offset := (page - 1) * pageSize

	db := global.DB.Model(&model.ShopOrder{})
	if filter.UserID != nil {
		db = db.Where("user_id = ?", *filter.UserID)
	}
	if filter.ProductID != nil {
		db = db.Where("product_id = ?", *filter.ProductID)
	}
	if len(filter.Statuses) > 0 {
		db = db.Where("status IN ?", filter.Statuses)
	} else if filter.Status != "" {
		db = db.Where("status = ?", filter.Status)
	}
	if filter.Keyword != "" {
		kw := "%" + filter.Keyword + "%"
		db = db.Where(
			"product_name ILIKE ? OR main_character_name ILIKE ? OR nickname ILIKE ?",
			kw, kw, kw,
		)
	}

	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	if err := db.Order("created_at DESC").Offset(offset).Limit(pageSize).Find(&list).Error; err != nil {
		return nil, 0, err
	}
	return list, total, nil
}

// CountUserProductPurchased 统计用户对某商品的有效购买数量（requested + delivered）
// limitPeriod 控制统计时间范围：forever=全部, daily=当天, weekly=本周, monthly=本月
func (r *ShopRepository) CountUserProductPurchased(userID, productID uint, limitPeriod string) (int64, error) {
	var total int64
	db := global.DB.Model(&model.ShopOrder{}).
		Where("user_id = ? AND product_id = ? AND status IN ?", userID, productID,
			[]string{model.OrderStatusRequested, model.OrderStatusDelivered})

	now := time.Now()
	switch limitPeriod {
	case model.LimitPeriodDaily:
		start := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
		db = db.Where("created_at >= ?", start)
	case model.LimitPeriodWeekly:
		weekday := int(now.Weekday())
		if weekday == 0 {
			weekday = 7
		}
		start := time.Date(now.Year(), now.Month(), now.Day()-weekday+1, 0, 0, 0, 0, now.Location())
		db = db.Where("created_at >= ?", start)
	case model.LimitPeriodMonthly:
		start := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
		db = db.Where("created_at >= ?", start)
		// forever 或其他值不加时间过滤
	}

	err := db.Select("COALESCE(SUM(quantity), 0)").Scan(&total).Error
	return total, err
}

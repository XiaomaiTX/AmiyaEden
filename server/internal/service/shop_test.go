package service

import (
	"amiya-eden/global"
	"amiya-eden/internal/model"
	"amiya-eden/internal/repository"
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestAdminDeliverOrderAttemptsInGameMailButIgnoresMailErrors(t *testing.T) {
	db := newShopServiceTestDB(t)
	useShopServiceTestDB(t, db)

	product := &model.ShopProduct{
		Name:      "Navy Omen",
		Price:     10,
		Stock:     -1,
		Type:      model.ProductTypeNormal,
		Status:    model.ProductStatusOnSale,
		SortOrder: 1,
	}
	if err := db.Create(product).Error; err != nil {
		t.Fatalf("create product: %v", err)
	}

	order := &model.ShopOrder{
		OrderNo:           "ORDER001",
		UserID:            42,
		MainCharacterName: "Pilot One",
		Nickname:          "Pilot",
		ProductID:         product.ID,
		ProductName:       product.Name,
		ProductType:       product.Type,
		Quantity:          2,
		UnitPrice:         product.Price,
		TotalPrice:        product.Price * 2,
		Status:            model.OrderStatusRequested,
	}
	if err := db.Create(order).Error; err != nil {
		t.Fatalf("create order: %v", err)
	}

	svc := NewShopService()
	mailAttempted := false
	svc.orderDeliveryMailSender = func(ctx context.Context, operatorID uint, deliveredOrder *model.ShopOrder) (MailAttemptSummary, error) {
		mailAttempted = true
		if operatorID != 77 {
			t.Fatalf("operatorID = %d, want 77", operatorID)
		}
		if deliveredOrder.ID != order.ID {
			t.Fatalf("order id = %d, want %d", deliveredOrder.ID, order.ID)
		}
		return MailAttemptSummary{
			MailSenderCharacterID:      90000077,
			MailSenderCharacterName:    "Officer Main",
			MailRecipientCharacterID:   90000042,
			MailRecipientCharacterName: "Pilot Main",
		}, errors.New("mail failed")
	}

	deliveredOrder, mailSummary, err := svc.AdminDeliverOrder(order.ID, 77, "contract issued")
	if err != nil {
		t.Fatalf("AdminDeliverOrder() error = %v", err)
	}
	if !mailAttempted {
		t.Fatal("expected deliver to attempt in-game mail after successful delivery")
	}
	if !strings.Contains(mailSummary.MailError, "mail failed") {
		t.Fatalf("mailError = %q, want to contain %q", mailSummary.MailError, "mail failed")
	}
	if mailSummary.MailSenderCharacterID != 90000077 || mailSummary.MailRecipientCharacterID != 90000042 {
		t.Fatalf("unexpected mail summary: %#v", mailSummary)
	}
	if deliveredOrder.Status != model.OrderStatusDelivered {
		t.Fatalf("status = %q, want %q", deliveredOrder.Status, model.OrderStatusDelivered)
	}

	var updated model.ShopOrder
	if err := db.First(&updated, order.ID).Error; err != nil {
		t.Fatalf("reload order: %v", err)
	}
	if updated.Status != model.OrderStatusDelivered {
		t.Fatalf("status = %q, want %q", updated.Status, model.OrderStatusDelivered)
	}
	if updated.ReviewedBy == nil || *updated.ReviewedBy != 77 {
		t.Fatalf("reviewed_by = %v, want 77", updated.ReviewedBy)
	}
}

func TestAdminDeliverOrderReturnsMailAttemptSummaryWhenMailSucceeds(t *testing.T) {
	db := newShopServiceTestDB(t)
	useShopServiceTestDB(t, db)

	product := &model.ShopProduct{
		Name:      "Navy Omen",
		Price:     10,
		Stock:     -1,
		Type:      model.ProductTypeNormal,
		Status:    model.ProductStatusOnSale,
		SortOrder: 1,
	}
	if err := db.Create(product).Error; err != nil {
		t.Fatalf("create product: %v", err)
	}

	order := &model.ShopOrder{
		OrderNo:           "ORDER002",
		UserID:            42,
		MainCharacterName: "Pilot One",
		ProductID:         product.ID,
		ProductName:       product.Name,
		Quantity:          1,
		UnitPrice:         product.Price,
		TotalPrice:        product.Price,
		Status:            model.OrderStatusRequested,
	}
	if err := db.Create(order).Error; err != nil {
		t.Fatalf("create order: %v", err)
	}

	svc := NewShopService()
	svc.orderDeliveryMailSender = func(ctx context.Context, operatorID uint, deliveredOrder *model.ShopOrder) (MailAttemptSummary, error) {
		return MailAttemptSummary{
			MailID:                     123456789,
			MailSenderCharacterID:      90000077,
			MailSenderCharacterName:    "Officer Main",
			MailRecipientCharacterID:   90000042,
			MailRecipientCharacterName: "Pilot Main",
		}, nil
	}

	_, mailSummary, err := svc.AdminDeliverOrder(order.ID, 77, "contract issued")
	if err != nil {
		t.Fatalf("AdminDeliverOrder() error = %v", err)
	}
	if mailSummary.MailError != "" {
		t.Fatalf("mailError = %q, want empty", mailSummary.MailError)
	}
	if mailSummary.MailID != 123456789 {
		t.Fatalf("mailID = %d, want 123456789", mailSummary.MailID)
	}
	if mailSummary.MailSenderCharacterName != "Officer Main" || mailSummary.MailRecipientCharacterName != "Pilot Main" {
		t.Fatalf("unexpected mail summary: %#v", mailSummary)
	}
}

func TestBuildShopOrderDeliveryMailContentIncludesBilingualOfficerNotice(t *testing.T) {
	subject, body := buildShopOrderDeliveryMailContent(
		"ORD-20260403",
		"Navy Omen",
		2,
		"Amiya",
		"请发到主角色",
		"已通过军团合同发放",
	)

	if !strings.Contains(subject, "订单发放通知") || !strings.Contains(subject, "Order Delivery Notice") {
		t.Fatalf("unexpected subject: %q", subject)
	}
	if !strings.Contains(body, "你的订单已由 Amiya 发放") {
		t.Fatalf("expected Chinese body to mention order item and officer nickname, got %q", body)
	}
	if !strings.Contains(body, "订单编号：ORD-20260403") {
		t.Fatalf("expected Chinese body to include order number, got %q", body)
	}
	if !strings.Contains(body, "订单内容：Navy Omen") {
		t.Fatalf("expected Chinese body to include order item, got %q", body)
	}
	if !strings.Contains(body, "数量：2") {
		t.Fatalf("expected Chinese body to include quantity, got %q", body)
	}
	if !strings.Contains(body, "备注：请发到主角色") {
		t.Fatalf("expected Chinese body to include order remark, got %q", body)
	}
	if !strings.Contains(body, "发放备注：已通过军团合同发放") {
		t.Fatalf("expected Chinese body to include delivery remark, got %q", body)
	}
	if !strings.Contains(body, "请检查你的钱包或合同") {
		t.Fatalf("expected Chinese body to mention wallet or contract, got %q", body)
	}
	if !strings.Contains(body, "感谢你的耐心等待。") {
		t.Fatalf("expected Chinese body to include a more professional tone, got %q", body)
	}
	if !strings.Contains(body, "Your shop order has been delivered by Amiya.") {
		t.Fatalf("expected English body to mention order item and officer nickname, got %q", body)
	}
	if !strings.Contains(body, "Order No: ORD-20260403") {
		t.Fatalf("expected English body to include order number, got %q", body)
	}
	if !strings.Contains(body, "Item: Navy Omen") {
		t.Fatalf("expected English body to include order item, got %q", body)
	}
	if !strings.Contains(body, "Quantity: 2") {
		t.Fatalf("expected English body to include quantity, got %q", body)
	}
	if !strings.Contains(body, "Remark: 请发到主角色") {
		t.Fatalf("expected English body to include order remark, got %q", body)
	}
	if !strings.Contains(body, "Delivery Remark: 已通过军团合同发放") {
		t.Fatalf("expected English body to include delivery remark, got %q", body)
	}
	if !strings.Contains(body, "Please check your wallet or contract.") {
		t.Fatalf("expected English body to mention wallet or contract, got %q", body)
	}
	if !strings.Contains(body, "Thank you for your patience.") {
		t.Fatalf("expected English body to include a more professional tone, got %q", body)
	}
}

func newShopServiceTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	dsn := fmt.Sprintf("file:shop_service_test_%d?mode=memory&cache=shared", time.Now().UnixNano())
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(
		&model.ShopProduct{},
		&model.ShopOrder{},
		&model.SystemWallet{},
		&model.WalletTransaction{},
	); err != nil {
		t.Fatalf("auto migrate: %v", err)
	}
	return db
}

func useShopServiceTestDB(t *testing.T, db *gorm.DB) {
	t.Helper()

	previous := global.DB
	global.DB = db
	t.Cleanup(func() {
		global.DB = previous
	})
}

func TestBuildShopOrderResponsesIncludesReviewerNickname(t *testing.T) {
	reviewerID := uint(77)
	createdAt := time.Date(2026, time.April, 3, 8, 0, 0, 0, time.UTC)

	orders := []model.ShopOrder{
		{
			BaseModel:  model.BaseModel{ID: 1, CreatedAt: createdAt},
			OrderNo:    "ORDER-1",
			Status:     model.OrderStatusDelivered,
			ReviewedBy: &reviewerID,
		},
		{
			BaseModel: model.BaseModel{ID: 2, CreatedAt: createdAt},
			OrderNo:   "ORDER-2",
			Status:    model.OrderStatusRequested,
		},
	}

	got := buildShopOrderResponses(orders, map[uint]string{reviewerID: "Logistics Fox"})

	if len(got) != 2 {
		t.Fatalf("expected 2 responses, got %d", len(got))
	}
	if got[0].ReviewerName != "Logistics Fox" {
		t.Fatalf("expected reviewer nickname to be included, got %q", got[0].ReviewerName)
	}
	if got[1].ReviewerName != "" {
		t.Fatalf("expected empty reviewer nickname for unreviewed order, got %q", got[1].ReviewerName)
	}
}

func TestAdminRejectOrderRollsBackRefundAndStockWhenOrderUpdateFails(t *testing.T) {
	db := newShopServiceTestDB(t)
	useShopServiceTestDB(t, db)

	product := &model.ShopProduct{
		Name:      "Navy Omen",
		Price:     10,
		Stock:     3,
		Type:      model.ProductTypeNormal,
		Status:    model.ProductStatusOnSale,
		SortOrder: 1,
	}
	if err := db.Create(product).Error; err != nil {
		t.Fatalf("create product: %v", err)
	}

	order := &model.ShopOrder{
		OrderNo:           "ORDER003",
		UserID:            42,
		MainCharacterName: "Pilot One",
		Nickname:          "Pilot",
		ProductID:         product.ID,
		ProductName:       product.Name,
		ProductType:       product.Type,
		Quantity:          2,
		UnitPrice:         product.Price,
		TotalPrice:        product.Price * 2,
		Status:            model.OrderStatusRequested,
	}
	if err := db.Create(order).Error; err != nil {
		t.Fatalf("create order: %v", err)
	}

	if err := db.Create(&model.SystemWallet{UserID: order.UserID, Balance: 50}).Error; err != nil {
		t.Fatalf("create wallet: %v", err)
	}

	updateErr := errors.New("inject order update failure")
	const callbackName = "shop_order_update_failure"
	if err := db.Callback().Update().Before("gorm:update").Register(callbackName, func(tx *gorm.DB) {
		if tx.Statement != nil && tx.Statement.Table == "shop_order" {
			if err := tx.AddError(updateErr); err != nil && !errors.Is(err, updateErr) {
				t.Fatalf("inject order update failure: %v", err)
			}
		}
	}); err != nil {
		t.Fatalf("register failing update callback: %v", err)
	}
	t.Cleanup(func() {
		_ = db.Callback().Update().Remove(callbackName)
	})

	svc := &ShopService{repo: repository.NewShopRepository(), walletSvc: NewSysWalletService()}
	_, err := svc.AdminRejectOrder(order.ID, 77, "refund failed")
	if err == nil {
		t.Fatal("expected AdminRejectOrder to fail when order update fails")
	}
	if !strings.Contains(err.Error(), "更新订单失败") {
		t.Fatalf("AdminRejectOrder() error = %v, want wrapped order update error", err)
	}

	var storedOrder model.ShopOrder
	if err := db.First(&storedOrder, order.ID).Error; err != nil {
		t.Fatalf("reload order: %v", err)
	}
	if storedOrder.Status != model.OrderStatusRequested {
		t.Fatalf("order status = %q, want %q", storedOrder.Status, model.OrderStatusRequested)
	}

	var storedProduct model.ShopProduct
	if err := db.First(&storedProduct, product.ID).Error; err != nil {
		t.Fatalf("reload product: %v", err)
	}
	if storedProduct.Stock != 3 {
		t.Fatalf("product stock = %d, want 3", storedProduct.Stock)
	}

	var storedWallet model.SystemWallet
	if err := db.Where("user_id = ?", order.UserID).First(&storedWallet).Error; err != nil {
		t.Fatalf("reload wallet: %v", err)
	}
	if storedWallet.Balance != 50 {
		t.Fatalf("wallet balance = %v, want 50", storedWallet.Balance)
	}

	var txCount int64
	if err := db.Model(&model.WalletTransaction{}).Count(&txCount).Error; err != nil {
		t.Fatalf("count wallet transactions: %v", err)
	}
	if txCount != 0 {
		t.Fatalf("wallet transaction count = %d, want 0", txCount)
	}
}

func TestUpdateOrderReviewTxRequiresExpectedStatus(t *testing.T) {
	db := newShopServiceTestDB(t)
	useShopServiceTestDB(t, db)

	product := &model.ShopProduct{
		Name:      "Navy Omen",
		Price:     10,
		Stock:     -1,
		Type:      model.ProductTypeNormal,
		Status:    model.ProductStatusOnSale,
		SortOrder: 1,
	}
	if err := db.Create(product).Error; err != nil {
		t.Fatalf("create product: %v", err)
	}

	order := &model.ShopOrder{
		OrderNo:           "ORDER004",
		UserID:            42,
		MainCharacterName: "Pilot One",
		Nickname:          "Pilot",
		ProductID:         product.ID,
		ProductName:       product.Name,
		ProductType:       product.Type,
		Quantity:          1,
		UnitPrice:         product.Price,
		TotalPrice:        product.Price,
		Status:            model.OrderStatusRequested,
	}
	if err := db.Create(order).Error; err != nil {
		t.Fatalf("create order: %v", err)
	}

	if err := db.Model(&model.ShopOrder{}).
		Where("id = ?", order.ID).
		Update("status", model.OrderStatusRejected).Error; err != nil {
		t.Fatalf("set order status rejected: %v", err)
	}

	repo := repository.NewShopRepository()
	var updated bool
	if err := db.Transaction(func(tx *gorm.DB) error {
		var err error
		updated, err = repo.UpdateOrderReviewTx(
			tx,
			order.ID,
			model.OrderStatusRequested,
			model.OrderStatusDelivered,
			77,
			time.Now(),
			"deliver now",
		)
		return err
	}); err != nil {
		t.Fatalf("UpdateOrderReviewTx() error = %v", err)
	}
	if updated {
		t.Fatal("expected UpdateOrderReviewTx to skip orders whose status no longer matches")
	}

	var storedOrder model.ShopOrder
	if err := db.First(&storedOrder, order.ID).Error; err != nil {
		t.Fatalf("reload order: %v", err)
	}
	if storedOrder.Status != model.OrderStatusRejected {
		t.Fatalf("order status = %q, want %q", storedOrder.Status, model.OrderStatusRejected)
	}
	if storedOrder.ReviewedBy != nil {
		t.Fatalf("reviewed_by = %v, want nil after skipped review update", storedOrder.ReviewedBy)
	}
}

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

var adminDeliveryOperatorRoles = []string{model.RoleAdmin}

var shopOrderManagerOperatorRoles = []string{model.RoleShopOrder}

func TestAdminDeliverOrderDispatchesInGameMailAsynchronously(t *testing.T) {
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
	started := make(chan struct {
		operatorID uint
		orderID    uint
	}, 1)
	finished := make(chan struct{})
	release := make(chan struct{})
	released := false
	t.Cleanup(func() {
		if !released {
			close(release)
		}
	})
	svc.orderDeliveryMailSender = func(ctx context.Context, operatorID uint, deliveredOrder *model.ShopOrder) (MailAttemptSummary, error) {
		defer close(finished)
		started <- struct {
			operatorID uint
			orderID    uint
		}{operatorID: operatorID, orderID: deliveredOrder.ID}
		<-release
		return MailAttemptSummary{}, errors.New("mail failed")
	}

	resultCh := make(chan struct {
		order       *model.ShopOrder
		mailSummary MailAttemptSummary
		err         error
	}, 1)
	go func() {
		deliveredOrder, mailSummary, err := svc.AdminDeliverOrder(order.ID, 77, adminDeliveryOperatorRoles, "contract issued")
		resultCh <- struct {
			order       *model.ShopOrder
			mailSummary MailAttemptSummary
			err         error
		}{order: deliveredOrder, mailSummary: mailSummary, err: err}
	}()

	select {
	case attempt := <-started:
		if attempt.operatorID != 77 {
			t.Fatalf("operatorID = %d, want 77", attempt.operatorID)
		}
		if attempt.orderID != order.ID {
			t.Fatalf("order id = %d, want %d", attempt.orderID, order.ID)
		}
	case <-time.After(time.Second):
		t.Fatal("expected deliver to start in-game mail dispatch")
	}

	var result struct {
		order       *model.ShopOrder
		mailSummary MailAttemptSummary
		err         error
	}
	select {
	case result = <-resultCh:
	case <-time.After(200 * time.Millisecond):
		t.Fatal("expected AdminDeliverOrder to return without waiting for mail sender")
	}

	if result.err != nil {
		t.Fatalf("AdminDeliverOrder() error = %v", result.err)
	}
	if result.mailSummary != (MailAttemptSummary{}) {
		t.Fatalf("mailSummary = %#v, want empty because delivery mail is asynchronous", result.mailSummary)
	}
	if result.order == nil {
		t.Fatal("expected delivered order")
	}
	deliveredOrder := result.order
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

	close(release)
	released = true
	select {
	case <-finished:
	case <-time.After(time.Second):
		t.Fatal("expected async mail sender to finish after release")
	}
}

func TestAdminDeliverOrderReturnsEmptyMailSummaryWhenMailRunsAsync(t *testing.T) {
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
	mailAttempted := make(chan struct{}, 1)
	svc.orderDeliveryMailSender = func(ctx context.Context, operatorID uint, deliveredOrder *model.ShopOrder) (MailAttemptSummary, error) {
		mailAttempted <- struct{}{}
		return MailAttemptSummary{
			MailID:                     123456789,
			MailSenderCharacterID:      90000077,
			MailSenderCharacterName:    "Officer Main",
			MailRecipientCharacterID:   90000042,
			MailRecipientCharacterName: "Pilot Main",
		}, nil
	}

	_, mailSummary, err := svc.AdminDeliverOrder(order.ID, 77, adminDeliveryOperatorRoles, "contract issued")
	if err != nil {
		t.Fatalf("AdminDeliverOrder() error = %v", err)
	}
	if mailSummary != (MailAttemptSummary{}) {
		t.Fatalf("mailSummary = %#v, want empty because delivery mail is asynchronous", mailSummary)
	}
	select {
	case <-mailAttempted:
	case <-time.After(time.Second):
		t.Fatal("expected deliver to trigger in-game mail sender asynchronously")
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
		&model.SystemConfig{},
		&model.SystemWallet{},
		&model.WalletTransaction{},
		&model.AuditEvent{},
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

func TestAdminDeliverOrderCreditsConfiguredAdminAward(t *testing.T) {
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
		OrderNo:           "ORDER-ADMIN-AWARD",
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

	svc := NewShopService()
	svc.orderDeliveryMailSender = nil

	if _, _, err := svc.AdminDeliverOrder(order.ID, 77, adminDeliveryOperatorRoles, "delivered"); err != nil {
		t.Fatalf("AdminDeliverOrder() error = %v", err)
	}

	var reviewerWallet model.SystemWallet
	if err := db.Where("user_id = ?", 77).First(&reviewerWallet).Error; err != nil {
		t.Fatalf("load reviewer wallet: %v", err)
	}
	if reviewerWallet.Balance != 10 {
		t.Fatalf("reviewer wallet balance = %v, want 10", reviewerWallet.Balance)
	}

	var txs []model.WalletTransaction
	if err := db.Order("id ASC").Find(&txs).Error; err != nil {
		t.Fatalf("load wallet transactions: %v", err)
	}
	if len(txs) != 1 {
		t.Fatalf("wallet transaction count = %d, want 1", len(txs))
	}
	if txs[0].UserID != 77 {
		t.Fatalf("wallet tx user_id = %d, want 77", txs[0].UserID)
	}
	if txs[0].RefType != "admin_award" {
		t.Fatalf("wallet tx ref_type = %q, want %q", txs[0].RefType, "admin_award")
	}
	if txs[0].RefID != fmt.Sprintf("admin_shop_delivery:%d", order.ID) {
		t.Fatalf("wallet tx ref_id = %q", txs[0].RefID)
	}
	if txs[0].OperatorID != 0 {
		t.Fatalf("wallet tx operator_id = %d, want 0", txs[0].OperatorID)
	}

	var auditEvents []model.AuditEvent
	if err := db.Where("resource_type = ? AND resource_id = ?", "shop_order", fmt.Sprintf("%d", order.ID)).
		Find(&auditEvents).Error; err != nil {
		t.Fatalf("load audit events: %v", err)
	}
	foundDeliver := false
	for _, event := range auditEvents {
		if event.Category == "approval" && event.Action == "shop_order_deliver" && event.Result == model.AuditResultSuccess {
			foundDeliver = true
			break
		}
	}
	if !foundDeliver {
		t.Fatalf("expected shop_order_deliver approval audit event, got %+v", auditEvents)
	}
}

func TestAdminDeliverOrderWithZeroAdminAwardSkipsCredit(t *testing.T) {
	db := newShopServiceTestDB(t)
	useShopServiceTestDB(t, db)

	if err := db.Create(&model.SystemConfig{
		Key:   "pap.admin_award",
		Value: "0",
		Desc:  "admin award",
	}).Error; err != nil {
		t.Fatalf("create system config: %v", err)
	}

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
		OrderNo:           "ORDER-ADMIN-AWARD-ZERO",
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

	svc := NewShopService()
	svc.orderDeliveryMailSender = nil

	if _, _, err := svc.AdminDeliverOrder(order.ID, 77, adminDeliveryOperatorRoles, "delivered"); err != nil {
		t.Fatalf("AdminDeliverOrder() error = %v", err)
	}

	var txs []model.WalletTransaction
	if err := db.Find(&txs).Error; err != nil {
		t.Fatalf("load wallet transactions: %v", err)
	}
	if len(txs) != 0 {
		t.Fatalf("wallet transaction count = %d, want 0", len(txs))
	}
}

func TestAdminDeliverOrderWithCustomAdminAwardCreditsConfiguredAmount(t *testing.T) {
	db := newShopServiceTestDB(t)
	useShopServiceTestDB(t, db)

	if err := db.Create(&model.SystemConfig{
		Key:   model.SysConfigPAPAdminAward,
		Value: "18",
		Desc:  "admin award",
	}).Error; err != nil {
		t.Fatalf("create system config: %v", err)
	}

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
		OrderNo:           "ORDER-ADMIN-AWARD-CUSTOM",
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

	svc := NewShopService()
	svc.orderDeliveryMailSender = nil

	if _, _, err := svc.AdminDeliverOrder(order.ID, 77, adminDeliveryOperatorRoles, "delivered"); err != nil {
		t.Fatalf("AdminDeliverOrder() error = %v", err)
	}

	var reviewerWallet model.SystemWallet
	if err := db.Where("user_id = ?", 77).First(&reviewerWallet).Error; err != nil {
		t.Fatalf("load reviewer wallet: %v", err)
	}
	if reviewerWallet.Balance != 18 {
		t.Fatalf("reviewer wallet balance = %v, want 18", reviewerWallet.Balance)
	}
}

func TestAdminDeliverOrderByShopOrderManagerSkipsAdminAward(t *testing.T) {
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
		OrderNo:           "ORDER-ADMIN-AWARD-SHOP-ROLE",
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

	svc := NewShopService()
	svc.orderDeliveryMailSender = nil

	if _, _, err := svc.AdminDeliverOrder(order.ID, 77, shopOrderManagerOperatorRoles, "delivered"); err != nil {
		t.Fatalf("AdminDeliverOrder() error = %v", err)
	}

	var txs []model.WalletTransaction
	if err := db.Find(&txs).Error; err != nil {
		t.Fatalf("load wallet transactions: %v", err)
	}
	if len(txs) != 0 {
		t.Fatalf("wallet transaction count = %d, want 0", len(txs))
	}
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

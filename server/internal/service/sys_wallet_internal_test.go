package service

import (
	"amiya-eden/global"
	"amiya-eden/internal/model"
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"time"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestCreditUserStoresSystemOperatorOnWalletTransaction(t *testing.T) {
	db := newSysWalletServiceTestDB(t)
	originalDB := global.DB
	global.DB = db
	defer func() { global.DB = originalDB }()

	svc := NewSysWalletService()
	if err := svc.CreditUser(42, 15.5, "shop order", "shop", "order:1"); err != nil {
		t.Fatalf("CreditUser() error = %v", err)
	}

	var txs []model.WalletTransaction
	if err := db.Order("id ASC").Find(&txs).Error; err != nil {
		t.Fatalf("load wallet transactions: %v", err)
	}
	if len(txs) != 1 {
		t.Fatalf("wallet transaction count = %d, want 1", len(txs))
	}

	tx := txs[0]
	if tx.UserID != 42 {
		t.Fatalf("wallet transaction user_id = %d, want 42", tx.UserID)
	}
	if tx.Amount != 15.5 {
		t.Fatalf("wallet transaction amount = %f, want 15.5", tx.Amount)
	}
	if tx.BalanceAfter != 15.5 {
		t.Fatalf("wallet transaction balance_after = %f, want 15.5", tx.BalanceAfter)
	}
	if tx.OperatorID != 0 {
		t.Fatalf("wallet transaction operator_id = %d, want 0", tx.OperatorID)
	}
	if tx.RefType != "shop" || tx.RefID != "order:1" || tx.Reason != "shop order" {
		t.Fatalf("unexpected transaction metadata: %+v", tx)
	}

	var auditEvents []model.AuditEvent
	if err := db.Order("id ASC").Find(&auditEvents).Error; err != nil {
		t.Fatalf("load audit events: %v", err)
	}
	if len(auditEvents) != 1 {
		t.Fatalf("audit event count = %d, want 1", len(auditEvents))
	}
	if auditEvents[0].Action != "apply_wallet_delta" {
		t.Fatalf("audit action = %q, want apply_wallet_delta", auditEvents[0].Action)
	}
	if auditEvents[0].Category != "fuxi_wallet" {
		t.Fatalf("audit category = %q, want fuxi_wallet", auditEvents[0].Category)
	}
	if auditEvents[0].TargetUserID != 42 {
		t.Fatalf("audit target_user_id = %d, want 42", auditEvents[0].TargetUserID)
	}
	if auditEvents[0].Result != model.AuditResultSuccess {
		t.Fatalf("audit result = %q, want %q", auditEvents[0].Result, model.AuditResultSuccess)
	}
}

func TestCreditUserTruncatesOverlongWalletReason(t *testing.T) {
	db := newSysWalletServiceTestDB(t)
	originalDB := global.DB
	global.DB = db
	defer func() { global.DB = originalDB }()

	overlongReason := strings.Repeat("伏", walletTransactionReasonMaxLength+16)
	svc := NewSysWalletService()
	if err := svc.CreditUser(42, 10, overlongReason, "shop", "order:2"); err != nil {
		t.Fatalf("CreditUser() error = %v", err)
	}

	var tx model.WalletTransaction
	if err := db.First(&tx).Error; err != nil {
		t.Fatalf("load wallet transaction: %v", err)
	}
	if got := len([]rune(tx.Reason)); got != walletTransactionReasonMaxLength {
		t.Fatalf("wallet transaction reason length = %d, want %d", got, walletTransactionReasonMaxLength)
	}
}

func TestNormalizeLedgerPageSizeUsesLedgerStandardBounds(t *testing.T) {
	tests := []struct {
		name string
		size int
		want int
	}{
		{name: "defaults when zero", size: 0, want: 200},
		{name: "preserves smaller valid page", size: 20, want: 20},
		{name: "keeps ledger default", size: 200, want: 200},
		{name: "allows larger ledger page", size: 500, want: 500},
		{name: "caps at thousand", size: 5000, want: 1000},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := normalizeLedgerPageSize(tt.size); got != tt.want {
				t.Fatalf("normalizeLedgerPageSize(%d) = %d, want %d", tt.size, got, tt.want)
			}
		})
	}
}

func TestGetMyTransactionsIncludesOperatorName(t *testing.T) {
	db := newSysWalletTransactionListTestDB(t)
	originalDB := global.DB
	global.DB = db
	defer func() { global.DB = originalDB }()

	operatorCharacterID := int64(90000077)
	if err := db.Create(&model.User{
		BaseModel:          model.BaseModel{ID: 77},
		Nickname:           "Officer Fox",
		PrimaryCharacterID: operatorCharacterID,
	}).Error; err != nil {
		t.Fatalf("create operator user: %v", err)
	}
	if err := db.Create(&model.EveCharacter{
		CharacterID:   operatorCharacterID,
		CharacterName: "Operator Main",
		UserID:        77,
	}).Error; err != nil {
		t.Fatalf("create operator character: %v", err)
	}
	if err := db.Create(&model.WalletTransaction{
		UserID:       42,
		Amount:       12.5,
		Reason:       "manual payout",
		RefType:      model.WalletRefManual,
		RefID:        "manual:1",
		BalanceAfter: 88.8,
		OperatorID:   77,
	}).Error; err != nil {
		t.Fatalf("create wallet transaction: %v", err)
	}

	svc := NewSysWalletService()
	records, total, err := svc.GetMyTransactions(42, 1, 20)
	if err != nil {
		t.Fatalf("GetMyTransactions() error = %v", err)
	}
	if total != 1 {
		t.Fatalf("transaction total = %d, want 1", total)
	}
	if len(records) != 1 {
		t.Fatalf("transaction count = %d, want 1", len(records))
	}
	if records[0].OperatorName != "Officer Fox" {
		t.Fatalf("operator_name = %q, want %q", records[0].OperatorName, "Officer Fox")
	}
}

func TestValidateWalletAnalyticsRequest(t *testing.T) {
	tests := []struct {
		name    string
		req     WalletAnalyticsRequest
		wantErr bool
	}{
		{
			name: "valid default top n with date range",
			req: WalletAnalyticsRequest{
				StartDate: "2026-01-01",
				EndDate:   "2026-01-30",
				TopN:      0,
			},
			wantErr: false,
		},
		{
			name: "valid full history when dates empty",
			req: WalletAnalyticsRequest{
				TopN: 10,
			},
			wantErr: false,
		},
		{
			name: "start after end",
			req: WalletAnalyticsRequest{
				StartDate: "2026-02-01",
				EndDate:   "2026-01-01",
				TopN:      10,
			},
			wantErr: true,
		},
		{
			name: "range over 365 days",
			req: WalletAnalyticsRequest{
				StartDate: "2025-01-01",
				EndDate:   "2026-02-01",
				TopN:      10,
			},
			wantErr: true,
		},
		{
			name: "top n out of range",
			req: WalletAnalyticsRequest{
				StartDate: "2026-01-01",
				EndDate:   "2026-01-02",
				TopN:      51,
			},
			wantErr: true,
		},
		{
			name: "only start date is invalid",
			req: WalletAnalyticsRequest{
				StartDate: "2026-01-01",
				TopN:      10,
			},
			wantErr: true,
		},
		{
			name: "only end date is invalid",
			req: WalletAnalyticsRequest{
				EndDate: "2026-01-01",
				TopN:    10,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, _, _, _, err := validateWalletAnalyticsRequest(&tt.req)
			if (err != nil) != tt.wantErr {
				t.Fatalf("validateWalletAnalyticsRequest() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidateWalletAnalyticsRequestUseTimeRangeFlag(t *testing.T) {
	start, end, topN, useTimeRange, err := validateWalletAnalyticsRequest(&WalletAnalyticsRequest{
		StartDate: "2026-01-01",
		EndDate:   "2026-01-01",
		TopN:      0,
	})
	if err != nil {
		t.Fatalf("validateWalletAnalyticsRequest() unexpected error: %v", err)
	}
	if !useTimeRange {
		t.Fatal("expected useTimeRange=true when start/end are provided")
	}
	if topN != 10 {
		t.Fatalf("topN = %d, want 10", topN)
	}
	if start.IsZero() || end.IsZero() {
		t.Fatalf("expected non-zero start/end, got start=%v end=%v", start, end)
	}

	start, end, topN, useTimeRange, err = validateWalletAnalyticsRequest(&WalletAnalyticsRequest{TopN: 5})
	if err != nil {
		t.Fatalf("validateWalletAnalyticsRequest() unexpected error for full history: %v", err)
	}
	if useTimeRange {
		t.Fatal("expected useTimeRange=false when dates are empty")
	}
	if topN != 5 {
		t.Fatalf("topN = %d, want 5", topN)
	}
	if !start.IsZero() || !end.IsZero() {
		t.Fatalf("expected zero start/end for full history, got start=%v end=%v", start, end)
	}
}

func TestAdminAdjustCreatesAuditEvent(t *testing.T) {
	db := newSysWalletServiceTestDB(t)
	originalDB := global.DB
	global.DB = db
	defer func() { global.DB = originalDB }()

	svc := NewSysWalletService()
	req := &AdminAdjustRequest{
		TargetUID: 42,
		Action:    model.WalletActionAdd,
		Amount:    20,
		Reason:    "ops adjustment",
	}
	if _, err := svc.AdminAdjust(7, req); err != nil {
		t.Fatalf("AdminAdjust() error = %v", err)
	}

	var logs []model.WalletLog
	if err := db.Find(&logs).Error; err != nil {
		t.Fatalf("load wallet logs: %v", err)
	}
	if len(logs) != 1 {
		t.Fatalf("wallet log count = %d, want 1", len(logs))
	}

	var auditEvents []model.AuditEvent
	if err := db.Order("id ASC").Find(&auditEvents).Error; err != nil {
		t.Fatalf("load audit events: %v", err)
	}
	if len(auditEvents) != 2 {
		t.Fatalf("audit event count = %d, want 2", len(auditEvents))
	}
	if auditEvents[0].Action != "apply_wallet_delta" && auditEvents[1].Action != "apply_wallet_delta" {
		t.Fatalf("expected apply_wallet_delta audit event, got %+v", auditEvents)
	}
	foundAdminAdjust := false
	for _, event := range auditEvents {
		if event.Action == "admin_adjust" {
			foundAdminAdjust = true
			if event.ActorUserID != 7 {
				t.Fatalf("admin_adjust actor_user_id = %d, want 7", event.ActorUserID)
			}
			if event.TargetUserID != 42 {
				t.Fatalf("admin_adjust target_user_id = %d, want 42", event.TargetUserID)
			}
		}
	}
	if !foundAdminAdjust {
		t.Fatal("expected admin_adjust audit event")
	}
}

func newSysWalletServiceTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	dsn := fmt.Sprintf("file:sys_wallet_service_test_%d?mode=memory&cache=shared", time.Now().UnixNano())
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(
		&model.SystemWallet{},
		&model.WalletTransaction{},
		&model.WalletLog{},
		&model.AuditEvent{},
		&model.SystemConfig{},
		&model.User{},
		&model.EveCharacter{},
	); err != nil {
		t.Fatalf("auto migrate: %v", err)
	}
	seedWalletCapabilityEnabledUser(t, db, 42, 900042, 1001)
	seedWalletCapabilityEnabledUser(t, db, 7, 900007, 1001)
	setWalletCapabilityPolicy(t, db, 1001, true)
	return db
}

func TestWalletDisabledUserAlwaysZeroAndRejectsMutations(t *testing.T) {
	db := newSysWalletServiceTestDB(t)
	originalDB := global.DB
	global.DB = db
	defer func() { global.DB = originalDB }()

	seedWalletCapabilityEnabledUser(t, db, 84, 900084, 2002)
	setWalletCapabilityPolicy(t, db, 2002, false)
	if err := db.Create(&model.SystemWallet{UserID: 84, Balance: 123.45}).Error; err != nil {
		t.Fatalf("seed disabled wallet: %v", err)
	}

	svc := NewSysWalletService()
	wallet, err := svc.GetMyWallet(84)
	if err != nil {
		t.Fatalf("GetMyWallet() error = %v", err)
	}
	if wallet.Balance != 0 {
		t.Fatalf("wallet balance = %f, want 0", wallet.Balance)
	}
	if err := svc.CreditUser(84, 10, "blocked", "test", "ref:1"); err == nil {
		t.Fatal("CreditUser() expected error, got nil")
	}
	if err := svc.DebitUser(84, 1, "blocked", "test", "ref:2"); err == nil {
		t.Fatal("DebitUser() expected error, got nil")
	}
	var persisted model.SystemWallet
	if err := db.Where("user_id = ?", 84).First(&persisted).Error; err != nil {
		t.Fatalf("load persisted wallet: %v", err)
	}
	if persisted.Balance != 0 {
		t.Fatalf("persisted balance = %f, want 0", persisted.Balance)
	}
}

func newSysWalletTransactionListTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	dsn := fmt.Sprintf("file:sys_wallet_tx_list_test_%d?mode=memory&cache=shared", time.Now().UnixNano())
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&model.User{}, &model.EveCharacter{}, &model.WalletTransaction{}, &model.SystemConfig{}); err != nil {
		t.Fatalf("auto migrate transaction list db: %v", err)
	}
	seedWalletCapabilityEnabledUser(t, db, 42, 900042, 1001)
	setWalletCapabilityPolicy(t, db, 1001, true)
	return db
}

func seedWalletCapabilityEnabledUser(t *testing.T, db *gorm.DB, userID uint, characterID int64, corporationID int64) {
	t.Helper()
	if err := db.Create(&model.User{
		BaseModel:          model.BaseModel{ID: userID},
		Nickname:           fmt.Sprintf("user_%d", userID),
		PrimaryCharacterID: characterID,
		Role:               model.RoleUser,
	}).Error; err != nil {
		t.Fatalf("seed user: %v", err)
	}
	if err := db.Create(&model.EveCharacter{
		CharacterID:   characterID,
		CharacterName: fmt.Sprintf("char_%d", characterID),
		UserID:        userID,
		CorporationID: corporationID,
	}).Error; err != nil {
		t.Fatalf("seed character: %v", err)
	}
}

func setWalletCapabilityPolicy(t *testing.T, db *gorm.DB, corporationID int64, enabled bool) {
	t.Helper()
	capabilities := []string{}
	if enabled {
		capabilities = append(capabilities, model.CorpCapabilityWalletUserEnabled)
	}
	raw, err := json.Marshal(map[string]any{
		"version":      1,
		"default_mode": "deny",
		"policies": []map[string]any{
			{
				"corporation_id": corporationID,
				"full_access":    false,
				"capabilities":   capabilities,
				"rules":          map[string]any{},
			},
		},
	})
	if err != nil {
		t.Fatalf("marshal policy: %v", err)
	}
	if err := db.Where("key = ?", model.SysConfigCorporationAccessPolicies).
		Assign(model.SystemConfig{
			Key:   model.SysConfigCorporationAccessPolicies,
			Value: string(raw),
			Desc:  "test policy",
		}).
		FirstOrCreate(&model.SystemConfig{}).Error; err != nil {
		t.Fatalf("upsert policy: %v", err)
	}
	clearCorpPolicyCache()
}

package service

import (
	"testing"
	"time"

	"amiya-eden/global"
	"amiya-eden/internal/model"

	"gorm.io/gorm"
)

const (
	walletAnalyticsTestUserID      uint  = 9101
	walletAnalyticsTestCharacterID int64 = 910000001
)

// newWalletAnalyticsTestService 准备一个绑定好人物归属、只含钱包相关表的内存库。
func newWalletAnalyticsTestService(t *testing.T) (*EveInfoService, *gorm.DB, uint, int64) {
	t.Helper()

	db := newServiceTestDB(
		t,
		"wallet_analytics",
		model.User{},
		model.EveCharacter{},
		model.EVECharacterWallet{},
		model.EVECharacterWalletJournal{},
	)

	previous := global.DB
	global.DB = db
	t.Cleanup(func() { global.DB = previous })

	seedWalletCapabilityEnabledUserForTests(t, db, walletAnalyticsTestUserID, walletAnalyticsTestCharacterID, 98185110)

	return NewEveInfoService(), db, walletAnalyticsTestUserID, walletAnalyticsTestCharacterID
}

func seedWalletJournalEntry(
	t *testing.T,
	db *gorm.DB,
	id int64,
	characterID int64,
	date time.Time,
	amount float64,
	tax float64,
	balance float64,
	refType string,
) {
	t.Helper()

	entry := model.EVECharacterWalletJournal{
		ID:          id,
		CharacterID: characterID,
		Date:        date,
		Amount:      amount,
		Tax:         tax,
		Balance:     balance,
		RefType:     refType,
	}
	if err := db.Create(&entry).Error; err != nil {
		t.Fatalf("seed wallet journal entry %d: %v", id, err)
	}
}

// TestGetWalletAnalyticsBucketsByUTCDayAndFillsGaps 覆盖分析口径的核心不变量：
//   - 日界按 UTC 切分（不是数据库会话时区）——用例里刻意放了会因 +08 切日而错位的两笔；
//   - 无流水的日子照样输出、余额沿用上一日；
//   - 期初余额取区间前最后一笔的余额，日终余额取当日最后一笔；
//   - 总览口径与图表一致。
func TestGetWalletAnalyticsBucketsByUTCDayAndFillsGaps(t *testing.T) {
	svc, db, userID, characterID := newWalletAnalyticsTestService(t)

	// 区间外的一笔，用来验证期初余额
	seedWalletJournalEntry(t, db, 1, characterID, time.Date(2026, 9, 20, 12, 0, 0, 0, time.UTC), 100, 0, 1100, "bounty_prizes")
	// 17:00 UTC = 次日 01:00 (UTC+8)：按会话时区会被算到 09-22，按 UTC 必须算 09-21
	seedWalletJournalEntry(t, db, 2, characterID, time.Date(2026, 9, 21, 17, 0, 0, 0, time.UTC), 50, 5, 1150, "bounty_prizes")
	// 16:30 UTC = 次日 00:30 (UTC+8)：按会话时区会被算到 09-23，按 UTC 必须算 09-22
	seedWalletJournalEntry(t, db, 3, characterID, time.Date(2026, 9, 22, 16, 30, 0, 0, time.UTC), -20, 0, 1130, "market_transaction")
	seedWalletJournalEntry(t, db, 4, characterID, time.Date(2026, 9, 22, 23, 59, 0, 0, time.UTC), 30, 0, 1160, "bounty_prizes")
	// 09-23 全天无流水，用来验证补空与余额顺延
	seedWalletJournalEntry(t, db, 5, characterID, time.Date(2026, 9, 24, 8, 0, 0, 0, time.UTC), -200, 10, 960, "player_donation")

	result, err := svc.GetWalletAnalytics(userID, &InfoWalletAnalyticsRequest{
		CharacterID: characterID,
		From:        "2026-09-21",
		To:          "2026-09-24",
	})
	if err != nil {
		t.Fatalf("GetWalletAnalytics() error = %v", err)
	}

	if result.Summary.AvailableFrom != "2026-09-20" || result.Summary.AvailableTo != "2026-09-24" {
		t.Fatalf("available range = [%s, %s], want [2026-09-20, 2026-09-24]",
			result.Summary.AvailableFrom, result.Summary.AvailableTo)
	}

	if result.Summary.OpeningBalance == nil || *result.Summary.OpeningBalance != 1100 {
		t.Fatalf("opening balance = %v, want 1100", result.Summary.OpeningBalance)
	}
	if result.Summary.ClosingBalance == nil || *result.Summary.ClosingBalance != 960 {
		t.Fatalf("closing balance = %v, want 960", result.Summary.ClosingBalance)
	}

	want := []InfoWalletDailyPoint{
		{Date: "2026-09-21", Income: 50, Expense: 0, Net: 50, Tax: 5, Balance: 1150, Count: 1},
		{Date: "2026-09-22", Income: 30, Expense: 20, Net: 10, Tax: 0, Balance: 1160, Count: 2},
		{Date: "2026-09-23", Income: 0, Expense: 0, Net: 0, Tax: 0, Balance: 1160, Count: 0},
		{Date: "2026-09-24", Income: 0, Expense: 200, Net: -200, Tax: 10, Balance: 960, Count: 1},
	}
	if len(result.DailySeries) != len(want) {
		t.Fatalf("daily series length = %d, want %d (%+v)", len(result.DailySeries), len(want), result.DailySeries)
	}
	for i := range want {
		if result.DailySeries[i] != want[i] {
			t.Errorf("daily[%d] = %+v, want %+v", i, result.DailySeries[i], want[i])
		}
	}

	if result.Summary.TotalIncome != 80 || result.Summary.TotalExpense != 220 {
		t.Fatalf("totals = income %.2f / expense %.2f, want 80 / 220",
			result.Summary.TotalIncome, result.Summary.TotalExpense)
	}
	if result.Summary.TotalNet != -140 || result.Summary.TotalTax != 15 || result.Summary.EntryCount != 4 {
		t.Fatalf("totals = net %.2f / tax %.2f / count %d, want -140 / 15 / 4",
			result.Summary.TotalNet, result.Summary.TotalTax, result.Summary.EntryCount)
	}

	wantBreakdown := []InfoWalletRefTypeItem{
		{RefType: "player_donation", Income: 0, Expense: 200, Count: 1},
		{RefType: "bounty_prizes", Income: 80, Expense: 0, Count: 2},
		{RefType: "market_transaction", Income: 0, Expense: 20, Count: 1},
	}
	if len(result.RefTypeBreakdown) != len(wantBreakdown) {
		t.Fatalf("ref type breakdown = %+v, want %+v", result.RefTypeBreakdown, wantBreakdown)
	}
	for i := range wantBreakdown {
		if result.RefTypeBreakdown[i] != wantBreakdown[i] {
			t.Errorf("breakdown[%d] = %+v, want %+v", i, result.RefTypeBreakdown[i], wantBreakdown[i])
		}
	}
}

// TestGetWalletAnalyticsClampsSeriesStartToFirstEntry 区间起点早于库内首笔流水时，
// 期初余额用首笔的 balance - amount 反推，序列从首笔流水所在日开始，
// 不输出无从得知的历史余额。
func TestGetWalletAnalyticsClampsSeriesStartToFirstEntry(t *testing.T) {
	svc, db, userID, characterID := newWalletAnalyticsTestService(t)

	seedWalletJournalEntry(t, db, 1, characterID, time.Date(2026, 9, 21, 10, 0, 0, 0, time.UTC), 100, 0, 1100, "bounty_prizes")

	result, err := svc.GetWalletAnalytics(userID, &InfoWalletAnalyticsRequest{
		CharacterID: characterID,
		From:        "2026-09-01",
		To:          "2026-09-22",
	})
	if err != nil {
		t.Fatalf("GetWalletAnalytics() error = %v", err)
	}

	if result.Summary.OpeningBalance == nil || *result.Summary.OpeningBalance != 1000 {
		t.Fatalf("opening balance = %v, want 1000 (首笔 balance 1100 - amount 100)", result.Summary.OpeningBalance)
	}

	want := []InfoWalletDailyPoint{
		{Date: "2026-09-21", Income: 100, Expense: 0, Net: 100, Tax: 0, Balance: 1100, Count: 1},
		{Date: "2026-09-22", Income: 0, Expense: 0, Net: 0, Tax: 0, Balance: 1100, Count: 0},
	}
	if len(result.DailySeries) != len(want) {
		t.Fatalf("daily series = %+v, want %+v", result.DailySeries, want)
	}
	for i := range want {
		if result.DailySeries[i] != want[i] {
			t.Errorf("daily[%d] = %+v, want %+v", i, result.DailySeries[i], want[i])
		}
	}
}

// TestGetWalletAnalyticsReturnsEmptySeriesWithoutData 库内没有任何流水时不编造余额：
// 序列为空、期初/期末为 null、可得范围为空字符串。
func TestGetWalletAnalyticsReturnsEmptySeriesWithoutData(t *testing.T) {
	svc, _, userID, characterID := newWalletAnalyticsTestService(t)

	result, err := svc.GetWalletAnalytics(userID, &InfoWalletAnalyticsRequest{
		CharacterID: characterID,
		From:        "2026-09-01",
		To:          "2026-09-05",
	})
	if err != nil {
		t.Fatalf("GetWalletAnalytics() error = %v", err)
	}

	if len(result.DailySeries) != 0 {
		t.Fatalf("daily series = %+v, want empty", result.DailySeries)
	}
	if len(result.RefTypeBreakdown) != 0 {
		t.Fatalf("ref type breakdown = %+v, want empty", result.RefTypeBreakdown)
	}
	if result.Summary.OpeningBalance != nil || result.Summary.ClosingBalance != nil {
		t.Fatalf("opening/closing = %v / %v, want nil / nil",
			result.Summary.OpeningBalance, result.Summary.ClosingBalance)
	}
	if result.Summary.AvailableFrom != "" || result.Summary.AvailableTo != "" {
		t.Fatalf("available range = [%s, %s], want empty", result.Summary.AvailableFrom, result.Summary.AvailableTo)
	}
	if result.Summary.EntryCount != 0 {
		t.Fatalf("entry count = %d, want 0", result.Summary.EntryCount)
	}
}

// TestGetWalletAnalyticsDefaultsToAvailableRangeWhenDatesOmitted 请求不传日期时，
// 分析范围回退到本地可得范围（最早流水日 ~ 最晚流水日），调用方无需知道 ESI 的 30 天窗口。
func TestGetWalletAnalyticsDefaultsToAvailableRangeWhenDatesOmitted(t *testing.T) {
	svc, db, userID, characterID := newWalletAnalyticsTestService(t)

	seedWalletJournalEntry(t, db, 1, characterID, time.Date(2026, 9, 21, 10, 0, 0, 0, time.UTC), 100, 0, 1100, "bounty_prizes")
	seedWalletJournalEntry(t, db, 2, characterID, time.Date(2026, 9, 23, 10, 0, 0, 0, time.UTC), -40, 0, 1060, "market_transaction")

	result, err := svc.GetWalletAnalytics(userID, &InfoWalletAnalyticsRequest{CharacterID: characterID})
	if err != nil {
		t.Fatalf("GetWalletAnalytics() error = %v", err)
	}

	wantDates := []string{"2026-09-21", "2026-09-22", "2026-09-23"}
	gotDates := make([]string, 0, len(result.DailySeries))
	for _, point := range result.DailySeries {
		gotDates = append(gotDates, point.Date)
	}
	if len(gotDates) != len(wantDates) {
		t.Fatalf("daily series dates = %v, want %v", gotDates, wantDates)
	}
	for i := range wantDates {
		if gotDates[i] != wantDates[i] {
			t.Errorf("daily[%d] date = %s, want %s", i, gotDates[i], wantDates[i])
		}
	}
	if result.Summary.EntryCount != 2 || result.Summary.TotalIncome != 100 || result.Summary.TotalExpense != 40 {
		t.Fatalf("summary = count %d / income %.2f / expense %.2f, want 2 / 100 / 40",
			result.Summary.EntryCount, result.Summary.TotalIncome, result.Summary.TotalExpense)
	}
}

// TestGetWalletAnalyticsWithoutDatesAndWithoutData 没有任何流水且不传日期时返回空结果，不报错也不编造区间。
func TestGetWalletAnalyticsWithoutDatesAndWithoutData(t *testing.T) {
	svc, _, userID, characterID := newWalletAnalyticsTestService(t)

	result, err := svc.GetWalletAnalytics(userID, &InfoWalletAnalyticsRequest{CharacterID: characterID})
	if err != nil {
		t.Fatalf("GetWalletAnalytics() error = %v", err)
	}
	if len(result.DailySeries) != 0 || result.Summary.EntryCount != 0 {
		t.Fatalf("series = %+v / count = %d, want empty", result.DailySeries, result.Summary.EntryCount)
	}
	if result.Summary.AvailableFrom != "" || result.Summary.AvailableTo != "" {
		t.Fatalf("available range = [%s, %s], want empty", result.Summary.AvailableFrom, result.Summary.AvailableTo)
	}
}

func TestGetWalletAnalyticsRejectsInvalidRequests(t *testing.T) {
	svc, db, userID, characterID := newWalletAnalyticsTestService(t)

	seedWalletJournalEntry(t, db, 1, characterID, time.Date(2026, 9, 21, 10, 0, 0, 0, time.UTC), 100, 0, 1100, "bounty_prizes")

	cases := []struct {
		name string
		req  InfoWalletAnalyticsRequest
	}{
		{
			name: "结束日期早于起始日期",
			req:  InfoWalletAnalyticsRequest{CharacterID: characterID, From: "2026-09-10", To: "2026-09-01"},
		},
		{
			name: "起始日期格式错误",
			req:  InfoWalletAnalyticsRequest{CharacterID: characterID, From: "2026/09/01", To: "2026-09-10"},
		},
		{
			name: "结束日期格式错误",
			req:  InfoWalletAnalyticsRequest{CharacterID: characterID, From: "2026-09-01", To: "not-a-date"},
		},
		{
			name: "人物不属于当前用户",
			req:  InfoWalletAnalyticsRequest{CharacterID: characterID + 1, From: "2026-09-01", To: "2026-09-10"},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := svc.GetWalletAnalytics(userID, &tc.req); err == nil {
				t.Fatalf("GetWalletAnalytics() error = nil, want error for %s", tc.name)
			}
		})
	}
}

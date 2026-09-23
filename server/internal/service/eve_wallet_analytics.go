package service

import (
	"errors"
	"fmt"
	"sort"
	"time"

	"amiya-eden/internal/repository"
)

// 个人钱包分析的时间口径：一律按 UTC 自然日切分。EVE 的每日维护、市场历史、
// 技能队列、ESI 日期字段全部以 UTC 换日，钱包统计跟随 UTC 才不会与其他模块错位。
//
// 不在数据库侧按天聚合的原因：本项目 PG 会话时区取决于部署环境（本机实测为
// Asia/Shanghai），数据库侧的 DATE(date) 会按会话时区切日，北京时间 00:00-08:00
// 的流水会被算到前一天。因此分桶放在 Go 里显式用 UTC 完成，与部署环境时区无关。
const walletAnalyticsDayLayout = "2006-01-02"

// InfoWalletAnalyticsRequest 钱包收支分析请求。From / To 为 UTC 自然日，闭区间。
//
// From / To 留空表示该侧不设边界，此时回退到本地留存的可得范围（From 取最早一天，
// To 取最晚一天）。前端首次加载不传日期即可直接拿到全量可得范围，
// 再由 summary.available_from / available_to 约束日期选择器。
type InfoWalletAnalyticsRequest struct {
	CharacterID int64  `json:"character_id" binding:"required"`
	From        string `json:"from"`
	To          string `json:"to"`
}

// InfoWalletAnalyticsSummary 分析总览
type InfoWalletAnalyticsSummary struct {
	OpeningBalance *float64 `json:"opening_balance"` // 区间起点余额，无流水时为 null
	ClosingBalance *float64 `json:"closing_balance"` // 区间终点余额，无流水时为 null
	TotalIncome    float64  `json:"total_income"`
	TotalExpense   float64  `json:"total_expense"`
	TotalNet       float64  `json:"total_net"`
	TotalTax       float64  `json:"total_tax"`
	EntryCount     int64    `json:"entry_count"`
	AvailableFrom  string   `json:"available_from"` // 本地留存的可得最早自然日（UTC），空字符串表示无数据
	AvailableTo    string   `json:"available_to"`   // 本地留存的可得最晚自然日（UTC），空字符串表示无数据
}

// InfoWalletDailyPoint 单日收支点（UTC 自然日）
type InfoWalletDailyPoint struct {
	Date    string  `json:"date"`
	Income  float64 `json:"income"`
	Expense float64 `json:"expense"`
	Net     float64 `json:"net"`
	Tax     float64 `json:"tax"`
	Balance float64 `json:"balance"` // 当日日终余额；当日无流水则沿用上一日
	Count   int64   `json:"count"`
}

// InfoWalletRefTypeItem 按交易类型（ref_type）聚合的收支
type InfoWalletRefTypeItem struct {
	RefType string  `json:"ref_type"`
	Income  float64 `json:"income"`
	Expense float64 `json:"expense"`
	Count   int64   `json:"count"`
}

// InfoWalletAnalyticsResponse 钱包收支分析响应
type InfoWalletAnalyticsResponse struct {
	Summary          InfoWalletAnalyticsSummary `json:"summary"`
	DailySeries      []InfoWalletDailyPoint     `json:"daily_series"`
	RefTypeBreakdown []InfoWalletRefTypeItem    `json:"ref_type_breakdown"`
}

// GetWalletAnalytics 个人钱包收支分析：按 UTC 自然日聚合日收支与日终余额，并按交易类型拆解收支构成。
//
// 数据来自本地已同步的钱包流水（eve_character_wallet_journal），不额外请求 ESI。
// ESI 的 wallet/journal 只回溯 30 天，但本地表只增不删，可得范围随运行时长自然增长，
// 前端应以 summary.available_from / available_to 作为日期选择器边界，不得写死 30 天。
func (s *EveInfoService) GetWalletAnalytics(userID uint, req *InfoWalletAnalyticsRequest) (*InfoWalletAnalyticsResponse, error) {
	if err := requireOwnedCharacter(s.charRepo, userID, req.CharacterID); err != nil {
		return nil, err
	}

	resp := &InfoWalletAnalyticsResponse{
		DailySeries:      make([]InfoWalletDailyPoint, 0),
		RefTypeBreakdown: make([]InfoWalletRefTypeItem, 0),
	}

	minDate, maxDate, err := s.walletRepo.GetWalletJournalDateBounds(req.CharacterID)
	if err != nil {
		return nil, fmt.Errorf("获取钱包流水可得范围失败: %w", err)
	}
	if minDate != nil {
		resp.Summary.AvailableFrom = walletAnalyticsDay(*minDate).Format(walletAnalyticsDayLayout)
	}
	if maxDate != nil {
		resp.Summary.AvailableTo = walletAnalyticsDay(*maxDate).Format(walletAnalyticsDayLayout)
	}

	fromDay, toDay, err := resolveWalletAnalyticsRange(req, minDate, maxDate)
	if err != nil {
		return nil, err
	}
	if fromDay == nil || toDay == nil {
		// 本地还没有该人物的流水：返回空序列，不用占位日期编造曲线
		return resp, nil
	}

	// 闭区间 [fromDay, toDay] 换算为左闭右开的查询边界 [rangeStart, rangeEnd)
	rangeStart := fromDay.UTC()
	rangeEnd := toDay.UTC().AddDate(0, 0, 1)

	entries, err := s.walletRepo.ListWalletJournalFlowEntries(req.CharacterID, rangeStart, rangeEnd)
	if err != nil {
		return nil, fmt.Errorf("获取钱包流水失败: %w", err)
	}

	balanceBeforeRange, err := s.walletRepo.GetWalletJournalLastBalanceBefore(req.CharacterID, rangeStart)
	if err != nil {
		return nil, fmt.Errorf("获取期初余额失败: %w", err)
	}

	opening := resolveWalletOpeningBalance(entries, balanceBeforeRange)
	startDay := resolveWalletSeriesStart(rangeStart, balanceBeforeRange, entries)
	resp.Summary.OpeningBalance = opening

	points := buildWalletDailySeries(entries, startDay, rangeEnd, opening)
	resp.DailySeries = points
	resp.RefTypeBreakdown = buildWalletRefTypeBreakdown(entries)

	income, expense, tax, count := summarizeWalletDailySeries(points)
	resp.Summary.TotalIncome = income
	resp.Summary.TotalExpense = expense
	resp.Summary.TotalNet = income - expense
	resp.Summary.TotalTax = tax
	resp.Summary.EntryCount = count
	if len(points) > 0 {
		closing := points[len(points)-1].Balance
		resp.Summary.ClosingBalance = &closing
	}

	return resp, nil
}

// resolveWalletAnalyticsRange 解析请求的闭区间，留空的一侧回退到本地可得范围。
// 本地没有任何流水且请求未显式给出日期时返回 nil 区间。
func resolveWalletAnalyticsRange(req *InfoWalletAnalyticsRequest, minDate, maxDate *time.Time) (*time.Time, *time.Time, error) {
	var fromDay, toDay *time.Time

	if req.From != "" {
		parsed, err := time.Parse(walletAnalyticsDayLayout, req.From)
		if err != nil {
			return nil, nil, fmt.Errorf("起始日期格式错误，应为 YYYY-MM-DD: %w", err)
		}
		fromDay = &parsed
	} else if minDate != nil {
		day := walletAnalyticsDay(*minDate)
		fromDay = &day
	}

	if req.To != "" {
		parsed, err := time.Parse(walletAnalyticsDayLayout, req.To)
		if err != nil {
			return nil, nil, fmt.Errorf("结束日期格式错误，应为 YYYY-MM-DD: %w", err)
		}
		toDay = &parsed
	} else if maxDate != nil {
		day := walletAnalyticsDay(*maxDate)
		toDay = &day
	}

	if fromDay == nil || toDay == nil {
		return nil, nil, nil
	}
	if toDay.Before(*fromDay) {
		return nil, nil, errors.New("结束日期不能早于起始日期")
	}
	return fromDay, toDay, nil
}

// walletAnalyticsDay 把任意时刻折算为它所属的 UTC 自然日 00:00:00。
func walletAnalyticsDay(t time.Time) time.Time {
	utc := t.UTC()
	return time.Date(utc.Year(), utc.Month(), utc.Day(), 0, 0, 0, 0, time.UTC)
}

// resolveWalletOpeningBalance 计算区间起点余额。
// 优先取区间之前最后一笔流水的余额；若区间起点不早于库内首笔流水（拿不到"之前"的余额），
// 退化为区间首笔流水的 balance - amount 反推。没有任何流水时返回 nil（响应中为 null）。
func resolveWalletOpeningBalance(entries []repository.WalletJournalFlowEntry, balanceBeforeRange *float64) *float64 {
	if balanceBeforeRange != nil {
		return balanceBeforeRange
	}
	if len(entries) == 0 {
		return nil
	}
	opening := entries[0].Balance - entries[0].Amount
	return &opening
}

// resolveWalletSeriesStart 决定日序列起始日。
// 区间起点早于库内首笔流水时，序列从首笔流水所在日开始，避免输出无从得知的日终余额。
func resolveWalletSeriesStart(fromDay time.Time, balanceBeforeRange *float64, entries []repository.WalletJournalFlowEntry) time.Time {
	fromDay = walletAnalyticsDay(fromDay)
	if balanceBeforeRange != nil || len(entries) == 0 {
		return fromDay
	}
	firstDay := walletAnalyticsDay(entries[0].Date)
	if firstDay.After(fromDay) {
		return firstDay
	}
	return fromDay
}

// buildWalletDailySeries 把区间内的流水按 UTC 自然日聚合成连续的日序列。
//   - 区间内没有流水的日子照样输出（收支与笔数为 0），余额沿用上一日：没有交易余额就不变。
//   - 日终余额取当日最后一笔流水的 balance（entries 已按 date ASC, id ASC 排序，最后一笔自然覆盖前者）。
//   - rangeEnd 是开区间上界（最后一天 + 1 天）；按 UTC 日推进（UTC 无夏令时，AddDate 安全）。
//   - opening 为 nil 表示无从得知期初余额（区间内无任何流水），此时返回空序列而不是编造余额。
func buildWalletDailySeries(entries []repository.WalletJournalFlowEntry, startDay, rangeEnd time.Time, opening *float64) []InfoWalletDailyPoint {
	points := make([]InfoWalletDailyPoint, 0)
	if opening == nil {
		return points
	}

	type dailyBucket struct {
		income  float64
		expense float64
		tax     float64
		count   int64
		balance float64
	}
	buckets := make(map[string]*dailyBucket, len(entries))
	for _, entry := range entries {
		key := walletAnalyticsDay(entry.Date).Format(walletAnalyticsDayLayout)
		bucket, ok := buckets[key]
		if !ok {
			bucket = &dailyBucket{}
			buckets[key] = bucket
		}
		if entry.Amount > 0 {
			bucket.income += entry.Amount
		} else if entry.Amount < 0 {
			bucket.expense += -entry.Amount
		}
		bucket.tax += entry.Tax
		bucket.count++
		bucket.balance = entry.Balance
	}

	carry := *opening
	end := rangeEnd.UTC()
	for day := startDay.UTC(); day.Before(end); day = day.AddDate(0, 0, 1) {
		point := InfoWalletDailyPoint{
			Date:    day.Format(walletAnalyticsDayLayout),
			Balance: carry,
		}
		if bucket, ok := buckets[point.Date]; ok {
			point.Income = bucket.income
			point.Expense = bucket.expense
			point.Tax = bucket.tax
			point.Count = bucket.count
			point.Net = bucket.income - bucket.expense
			point.Balance = bucket.balance
			carry = bucket.balance
		}
		points = append(points, point)
	}
	return points
}

// buildWalletRefTypeBreakdown 按交易类型聚合收支，按收支总额降序（同额按类型名升序，保证输出稳定）。
func buildWalletRefTypeBreakdown(entries []repository.WalletJournalFlowEntry) []InfoWalletRefTypeItem {
	index := make(map[string]*InfoWalletRefTypeItem, 16)
	keys := make([]string, 0, 16)
	for _, entry := range entries {
		item, ok := index[entry.RefType]
		if !ok {
			item = &InfoWalletRefTypeItem{RefType: entry.RefType}
			index[entry.RefType] = item
			keys = append(keys, entry.RefType)
		}
		if entry.Amount > 0 {
			item.Income += entry.Amount
		} else if entry.Amount < 0 {
			item.Expense += -entry.Amount
		}
		item.Count++
	}

	items := make([]InfoWalletRefTypeItem, 0, len(keys))
	for _, key := range keys {
		items = append(items, *index[key])
	}
	sort.SliceStable(items, func(i, j int) bool {
		left := items[i].Income + items[i].Expense
		right := items[j].Income + items[j].Expense
		if left != right {
			return left > right
		}
		return items[i].RefType < items[j].RefType
	})
	return items
}

// summarizeWalletDailySeries 从日序列汇总区间总量，保证总览口径与图表完全一致。
func summarizeWalletDailySeries(points []InfoWalletDailyPoint) (income, expense, tax float64, count int64) {
	for _, point := range points {
		income += point.Income
		expense += point.Expense
		tax += point.Tax
		count += point.Count
	}
	return income, expense, tax, count
}

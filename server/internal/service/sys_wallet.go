package service

import (
	"amiya-eden/global"
	"amiya-eden/internal/model"
	"amiya-eden/internal/repository"
	"errors"
	"fmt"
	"math"
	"sort"
	"strings"
	"time"

	"gorm.io/gorm"
)

// SysWalletService 伏羲币业务逻辑层
type SysWalletService struct {
	repo *repository.SysWalletRepository
}

const walletTransactionReasonMaxLength = 256

func buildWalletTransaction(userID uint, operatorID uint, delta float64, newBalance float64, reason, refType, refID string) *model.WalletTransaction {
	if reasonRunes := []rune(reason); len(reasonRunes) > walletTransactionReasonMaxLength {
		reason = string(reasonRunes[:walletTransactionReasonMaxLength])
	}

	return &model.WalletTransaction{
		UserID:       userID,
		Amount:       delta,
		Reason:       reason,
		RefType:      refType,
		RefID:        refID,
		BalanceAfter: newBalance,
		OperatorID:   operatorID,
	}
}

func (s *SysWalletService) applyWalletDeltaTx(tx *gorm.DB, userID uint, operatorID uint, delta float64, newBalance float64, reason, refType, refID string) error {
	if err := s.repo.UpdateBalanceTx(tx, userID, newBalance); err != nil {
		return err
	}

	return s.repo.CreateTransactionTx(
		tx,
		buildWalletTransaction(userID, operatorID, delta, newBalance, reason, refType, refID),
	)
}

func NewSysWalletService() *SysWalletService {
	return &SysWalletService{
		repo: repository.NewSysWalletRepository(),
	}
}

// ─────────────────────────────────────────────
//  用户端
// ─────────────────────────────────────────────

// GetMyWallet 获取当前用户钱包
func (s *SysWalletService) GetMyWallet(userID uint) (*model.SystemWallet, error) {
	return s.repo.GetOrCreateWallet(userID)
}

// GetMyTransactions 获取当前用户流水
func (s *SysWalletService) GetMyTransactions(userID uint, page, pageSize int) ([]model.TransactionWithCharacter, int64, error) {
	normalizeLedgerPageRequest(&page, &pageSize)
	filter := repository.WalletTransactionFilter{UserID: &userID}
	return s.repo.ListTransactionsWithCharacter(page, pageSize, filter)
}

// ─────────────────────────────────────────────
//  管理员端
// ─────────────────────────────────────────────

// AdminAdjustRequest 管理员调整钱包请求
type AdminAdjustRequest struct {
	TargetUID uint    `json:"target_uid" binding:"required"` // 目标用户 ID
	Action    string  `json:"action" binding:"required,oneof=add deduct set"`
	Amount    float64 `json:"amount" binding:"required,gt=0"` // 操作金额（必须正数）
	Reason    string  `json:"reason" binding:"required"`      // 操作原因
}

type WalletAnalyticsRequest struct {
	StartDate   string   `json:"start_date" binding:"required"`
	EndDate     string   `json:"end_date" binding:"required"`
	RefTypes    []string `json:"ref_types"`
	UserKeyword string   `json:"user_keyword"`
	TopN        int      `json:"top_n"`
}

type WalletAnalyticsSummary struct {
	WalletCount       int64   `json:"wallet_count"`
	ActiveWalletCount int64   `json:"active_wallet_count"`
	TotalBalance      float64 `json:"total_balance"`
	IncomeTotal       float64 `json:"income_total"`
	ExpenseTotal      float64 `json:"expense_total"`
	NetFlow           float64 `json:"net_flow"`
}

type WalletAnalyticsDailyPoint struct {
	Date    string  `json:"date"`
	Income  float64 `json:"income"`
	Expense float64 `json:"expense"`
	NetFlow float64 `json:"net_flow"`
}

type WalletAnalyticsRefTypeItem struct {
	RefType string  `json:"ref_type"`
	Income  float64 `json:"income"`
	Expense float64 `json:"expense"`
	Count   int64   `json:"count"`
}

type WalletAnalyticsTopUser struct {
	UserID        uint    `json:"user_id"`
	CharacterName string  `json:"character_name"`
	Amount        float64 `json:"amount"`
}

type WalletAnalyticsOperatorItem struct {
	OperatorID   uint    `json:"operator_id"`
	OperatorName string  `json:"operator_name"`
	Count        int64   `json:"count"`
	AmountTotal  float64 `json:"amount_total"`
}

type WalletAnalyticsAdminAdjustStats struct {
	Count       int64                         `json:"count"`
	AmountTotal float64                       `json:"amount_total"`
	ByOperator  []WalletAnalyticsOperatorItem `json:"by_operator"`
}

type WalletAnalyticsLargeTransaction struct {
	ID            uint    `json:"id"`
	UserID        uint    `json:"user_id"`
	CharacterName string  `json:"character_name"`
	Amount        float64 `json:"amount"`
	RefType       string  `json:"ref_type"`
	CreatedAt     string  `json:"created_at"`
}

type WalletAnalyticsFrequentAdjustment struct {
	TargetUID          uint    `json:"target_uid"`
	CharacterName      string  `json:"character_name"`
	AdjustCount        int64   `json:"adjust_count"`
	AmountTotal        float64 `json:"amount_total"`
	LastAdjustmentTime string  `json:"last_adjustment_time"`
}

type WalletAnalyticsOperatorConcentration struct {
	OperatorID   uint    `json:"operator_id"`
	OperatorName string  `json:"operator_name"`
	Count        int64   `json:"count"`
	AmountTotal  float64 `json:"amount_total"`
	Ratio        float64 `json:"ratio"`
}

type WalletAnalyticsAnomalies struct {
	LargeTransactions     []WalletAnalyticsLargeTransaction      `json:"large_transactions"`
	FrequentAdjustments   []WalletAnalyticsFrequentAdjustment    `json:"frequent_adjustments"`
	OperatorConcentration []WalletAnalyticsOperatorConcentration `json:"operator_concentration"`
}

type WalletAnalyticsResponse struct {
	Summary          WalletAnalyticsSummary          `json:"summary"`
	DailySeries      []WalletAnalyticsDailyPoint     `json:"daily_series"`
	RefTypeBreakdown []WalletAnalyticsRefTypeItem    `json:"ref_type_breakdown"`
	TopInflowUsers   []WalletAnalyticsTopUser        `json:"top_inflow_users"`
	TopOutflowUsers  []WalletAnalyticsTopUser        `json:"top_outflow_users"`
	AdminAdjustStats WalletAnalyticsAdminAdjustStats `json:"admin_adjust_stats"`
	Anomalies        WalletAnalyticsAnomalies        `json:"anomalies"`
}

// AdminAdjust 管理员调整用户钱包余额
func (s *SysWalletService) AdminAdjust(operatorID uint, req *AdminAdjustRequest) (*model.SystemWallet, error) {
	var adjustedWallet *model.SystemWallet
	err := global.DB.Transaction(func(tx *gorm.DB) error {
		wallet, err := s.repo.GetOrCreateWalletTx(tx, req.TargetUID)
		if err != nil {
			return fmt.Errorf("获取用户钱包失败: %w", err)
		}

		oldBalance := wallet.Balance
		var newBalance float64
		var txAmount float64

		switch req.Action {
		case model.WalletActionAdd:
			newBalance = oldBalance + req.Amount
			txAmount = req.Amount
		case model.WalletActionDeduct:
			newBalance = oldBalance - req.Amount
			if newBalance < 0 {
				return errors.New("余额不足，无法扣减")
			}
			txAmount = -req.Amount
		case model.WalletActionSet:
			newBalance = req.Amount
			txAmount = newBalance - oldBalance
		default:
			return errors.New("无效的操作类型")
		}

		if err := s.repo.UpdateBalanceTx(tx, req.TargetUID, newBalance); err != nil {
			return fmt.Errorf("更新余额失败: %w", err)
		}

		walletTx := &model.WalletTransaction{
			UserID:       req.TargetUID,
			Amount:       txAmount,
			Reason:       req.Reason,
			RefType:      model.WalletRefAdminAdjust,
			RefID:        fmt.Sprintf("admin:%d", operatorID),
			BalanceAfter: newBalance,
			OperatorID:   operatorID,
		}
		if err := s.repo.CreateTransactionTx(tx, walletTx); err != nil {
			return fmt.Errorf("写入流水失败: %w", err)
		}

		log := &model.WalletLog{
			OperatorID: operatorID,
			TargetUID:  req.TargetUID,
			Action:     req.Action,
			Amount:     req.Amount,
			Before:     oldBalance,
			After:      newBalance,
			Reason:     req.Reason,
		}
		if err := s.repo.CreateLogTx(tx, log); err != nil {
			return fmt.Errorf("写入操作日志失败: %w", err)
		}

		wallet.Balance = newBalance
		adjustedWallet = wallet
		return nil
	})
	if err != nil {
		return nil, err
	}
	return adjustedWallet, nil
}

// AdminListWallets 管理员查询所有钱包（附带主人物名）
func (s *SysWalletService) AdminListWallets(page, pageSize int, filter repository.WalletListFilter) ([]model.WalletWithCharacter, int64, error) {
	normalizeLedgerPageRequest(&page, &pageSize)
	return s.repo.ListWalletsWithCharacter(page, pageSize, filter)
}

// AdminGetWallet 管理员查看指定用户钱包
func (s *SysWalletService) AdminGetWallet(userID uint) (*model.SystemWallet, error) {
	return s.repo.GetOrCreateWallet(userID)
}

// AdminListTransactions 管理员查询流水（可按用户/类型筛选，附带人物名）
func (s *SysWalletService) AdminListTransactions(page, pageSize int, filter repository.WalletTransactionFilter) ([]model.TransactionWithCharacter, int64, error) {
	normalizeLedgerPageRequest(&page, &pageSize)
	return s.repo.ListTransactionsWithCharacter(page, pageSize, filter)
}

// AdminListLogs 管理员查询操作日志（附带人物名）
func (s *SysWalletService) AdminListLogs(page, pageSize int, filter repository.WalletLogFilter) ([]model.LogWithCharacter, int64, error) {
	normalizeLedgerPageRequest(&page, &pageSize)
	return s.repo.ListLogsWithCharacter(page, pageSize, filter)
}

func (s *SysWalletService) AdminGetAnalytics(req *WalletAnalyticsRequest) (*WalletAnalyticsResponse, error) {
	start, end, topN, err := validateWalletAnalyticsRequest(req)
	if err != nil {
		return nil, err
	}

	filter := repository.WalletAnalyticsFilter{
		StartAt:     start,
		EndAt:       end,
		RefTypes:    req.RefTypes,
		UserKeyword: strings.TrimSpace(req.UserKeyword),
	}

	summaryAgg, err := s.repo.GetWalletAnalyticsSummary(filter)
	if err != nil {
		return nil, fmt.Errorf("查询分析总览失败: %w", err)
	}

	dailyAgg, err := s.repo.ListWalletAnalyticsDailySeries(filter)
	if err != nil {
		return nil, fmt.Errorf("查询趋势失败: %w", err)
	}

	refAgg, err := s.repo.ListWalletAnalyticsRefTypeBreakdown(filter)
	if err != nil {
		return nil, fmt.Errorf("查询类型结构失败: %w", err)
	}

	topInflowAgg, err := s.repo.ListWalletAnalyticsTopUsers(filter, true, topN)
	if err != nil {
		return nil, fmt.Errorf("查询收入用户排行失败: %w", err)
	}
	topOutflowAgg, err := s.repo.ListWalletAnalyticsTopUsers(filter, false, topN)
	if err != nil {
		return nil, fmt.Errorf("查询支出用户排行失败: %w", err)
	}

	adjustStatsAgg, err := s.repo.GetWalletAnalyticsAdminAdjustStats(filter, topN)
	if err != nil {
		return nil, fmt.Errorf("查询管理员调整统计失败: %w", err)
	}

	absAmounts, err := s.repo.ListWalletAnalyticsAbsoluteAmounts(filter)
	if err != nil {
		return nil, fmt.Errorf("查询流水金额分布失败: %w", err)
	}
	p95 := calcPercentile(absAmounts, 0.95)
	if p95 < 100 {
		p95 = 100
	}
	largeTxAgg, err := s.repo.ListWalletAnalyticsLargeTransactions(filter, p95, topN)
	if err != nil {
		return nil, fmt.Errorf("查询大额流水异常失败: %w", err)
	}
	frequentAdjustAgg, err := s.repo.ListWalletAnalyticsFrequentAdjustments(filter, topN)
	if err != nil {
		return nil, fmt.Errorf("查询频繁调整异常失败: %w", err)
	}
	concentrationAgg, err := s.repo.ListWalletAnalyticsOperatorConcentration(filter, topN)
	if err != nil {
		return nil, fmt.Errorf("查询操作人集中度异常失败: %w", err)
	}

	resp := &WalletAnalyticsResponse{
		Summary: WalletAnalyticsSummary{
			WalletCount:       summaryAgg.WalletCount,
			ActiveWalletCount: summaryAgg.ActiveWalletCount,
			TotalBalance:      summaryAgg.TotalBalance,
			IncomeTotal:       summaryAgg.IncomeTotal,
			ExpenseTotal:      summaryAgg.ExpenseTotal,
			NetFlow:           summaryAgg.IncomeTotal - summaryAgg.ExpenseTotal,
		},
		DailySeries:      make([]WalletAnalyticsDailyPoint, 0, len(dailyAgg)),
		RefTypeBreakdown: make([]WalletAnalyticsRefTypeItem, 0, len(refAgg)),
		TopInflowUsers:   make([]WalletAnalyticsTopUser, 0, len(topInflowAgg)),
		TopOutflowUsers:  make([]WalletAnalyticsTopUser, 0, len(topOutflowAgg)),
		AdminAdjustStats: WalletAnalyticsAdminAdjustStats{
			Count:       adjustStatsAgg.Count,
			AmountTotal: adjustStatsAgg.AmountTotal,
			ByOperator:  make([]WalletAnalyticsOperatorItem, 0, len(adjustStatsAgg.ByOperator)),
		},
		Anomalies: WalletAnalyticsAnomalies{
			LargeTransactions:     make([]WalletAnalyticsLargeTransaction, 0, len(largeTxAgg)),
			FrequentAdjustments:   make([]WalletAnalyticsFrequentAdjustment, 0, len(frequentAdjustAgg)),
			OperatorConcentration: make([]WalletAnalyticsOperatorConcentration, 0, len(concentrationAgg)),
		},
	}

	for _, row := range dailyAgg {
		resp.DailySeries = append(resp.DailySeries, WalletAnalyticsDailyPoint{
			Date:    row.Date,
			Income:  row.Income,
			Expense: row.Expense,
			NetFlow: row.Income - row.Expense,
		})
	}

	for _, row := range refAgg {
		resp.RefTypeBreakdown = append(resp.RefTypeBreakdown, WalletAnalyticsRefTypeItem{
			RefType: row.RefType,
			Income:  row.Income,
			Expense: row.Expense,
			Count:   row.Count,
		})
	}

	for _, row := range topInflowAgg {
		resp.TopInflowUsers = append(resp.TopInflowUsers, WalletAnalyticsTopUser{
			UserID:        row.UserID,
			CharacterName: row.CharacterName,
			Amount:        row.Amount,
		})
	}
	for _, row := range topOutflowAgg {
		resp.TopOutflowUsers = append(resp.TopOutflowUsers, WalletAnalyticsTopUser{
			UserID:        row.UserID,
			CharacterName: row.CharacterName,
			Amount:        row.Amount,
		})
	}

	for _, row := range adjustStatsAgg.ByOperator {
		resp.AdminAdjustStats.ByOperator = append(resp.AdminAdjustStats.ByOperator, WalletAnalyticsOperatorItem{
			OperatorID:   row.OperatorID,
			OperatorName: row.OperatorName,
			Count:        row.Count,
			AmountTotal:  row.AmountTotal,
		})
	}

	for _, row := range largeTxAgg {
		resp.Anomalies.LargeTransactions = append(resp.Anomalies.LargeTransactions, WalletAnalyticsLargeTransaction{
			ID:            row.ID,
			UserID:        row.UserID,
			CharacterName: row.CharacterName,
			Amount:        row.Amount,
			RefType:       row.RefType,
			CreatedAt:     row.CreatedAt.Format("2006-01-02 15:04:05"),
		})
	}
	for _, row := range frequentAdjustAgg {
		resp.Anomalies.FrequentAdjustments = append(resp.Anomalies.FrequentAdjustments, WalletAnalyticsFrequentAdjustment{
			TargetUID:          row.TargetUID,
			CharacterName:      row.CharacterName,
			AdjustCount:        row.AdjustCount,
			AmountTotal:        row.AmountTotal,
			LastAdjustmentTime: row.LastAdjustmentTime.Format("2006-01-02 15:04:05"),
		})
	}
	for _, row := range concentrationAgg {
		if adjustStatsAgg.AmountTotal <= 0 {
			continue
		}
		ratio := row.AmountTotal / adjustStatsAgg.AmountTotal
		if ratio < 0.4 {
			continue
		}
		resp.Anomalies.OperatorConcentration = append(resp.Anomalies.OperatorConcentration, WalletAnalyticsOperatorConcentration{
			OperatorID:   row.OperatorID,
			OperatorName: row.OperatorName,
			Count:        row.Count,
			AmountTotal:  row.AmountTotal,
			Ratio:        ratio,
		})
	}
	sort.Slice(resp.Anomalies.OperatorConcentration, func(i, j int) bool {
		return resp.Anomalies.OperatorConcentration[i].AmountTotal > resp.Anomalies.OperatorConcentration[j].AmountTotal
	})

	return resp, nil
}

func validateWalletAnalyticsRequest(req *WalletAnalyticsRequest) (time.Time, time.Time, int, error) {
	start, err := time.Parse("2006-01-02", strings.TrimSpace(req.StartDate))
	if err != nil {
		return time.Time{}, time.Time{}, 0, errors.New("start_date 格式错误，需为 YYYY-MM-DD")
	}
	end, err := time.Parse("2006-01-02", strings.TrimSpace(req.EndDate))
	if err != nil {
		return time.Time{}, time.Time{}, 0, errors.New("end_date 格式错误，需为 YYYY-MM-DD")
	}
	if start.After(end) {
		return time.Time{}, time.Time{}, 0, errors.New("start_date 不能晚于 end_date")
	}
	if end.Sub(start) > 365*24*time.Hour {
		return time.Time{}, time.Time{}, 0, errors.New("时间范围不能超过 365 天")
	}
	topN := req.TopN
	if topN == 0 {
		topN = 10
	}
	if topN < 1 || topN > 50 {
		return time.Time{}, time.Time{}, 0, errors.New("top_n 必须在 1-50 之间")
	}
	return start, end.Add(24 * time.Hour), topN, nil
}

func calcPercentile(values []float64, p float64) float64 {
	if len(values) == 0 {
		return 0
	}
	sorted := append([]float64(nil), values...)
	sort.Float64s(sorted)
	if p <= 0 {
		return sorted[0]
	}
	if p >= 1 {
		return sorted[len(sorted)-1]
	}
	pos := p * float64(len(sorted)-1)
	lower := int(math.Floor(pos))
	upper := int(math.Ceil(pos))
	if lower == upper {
		return sorted[lower]
	}
	weight := pos - float64(lower)
	return sorted[lower] + (sorted[upper]-sorted[lower])*weight
}

// ─────────────────────────────────────────────
//  内部调用（供其他 Service 调用）
// ─────────────────────────────────────────────

// CreditUser 给用户加钱（内部调用，如 PAP 奖励、SRP 发放等）
func (s *SysWalletService) CreditUser(userID uint, amount float64, reason, refType, refID string) error {
	if amount <= 0 {
		return errors.New("金额必须大于 0")
	}

	return global.DB.Transaction(func(tx *gorm.DB) error {
		wallet, err := s.repo.GetOrCreateWalletTx(tx, userID)
		if err != nil {
			return fmt.Errorf("获取用户钱包失败: %w", err)
		}
		newBalance := wallet.Balance + amount
		return s.applyWalletDeltaTx(tx, userID, 0, amount, newBalance, reason, refType, refID)
	})
}

// DebitUser 扣减用户余额（内部调用，如商城购买）
func (s *SysWalletService) DebitUser(userID uint, amount float64, reason, refType, refID string) error {
	if amount <= 0 {
		return errors.New("金额必须大于 0")
	}

	return global.DB.Transaction(func(tx *gorm.DB) error {
		wallet, err := s.repo.GetOrCreateWalletTx(tx, userID)
		if err != nil {
			return fmt.Errorf("获取用户钱包失败: %w", err)
		}
		if wallet.Balance < amount {
			return errors.New("余额不足")
		}
		newBalance := wallet.Balance - amount
		return s.applyWalletDeltaTx(tx, userID, 0, -amount, newBalance, reason, refType, refID)
	})
}

// ApplyWalletDeltaTx 在已有事务中对用户钱包应用差量（正=充值，负=扣减），用于 PAP 重复发放去重
func (s *SysWalletService) ApplyWalletDeltaTx(tx *gorm.DB, userID uint, delta float64, reason, refType, refID string) error {
	return s.ApplyWalletDeltaByOperatorTx(tx, userID, 0, delta, reason, refType, refID)
}

// ApplyWalletDeltaByOperatorTx applies a wallet delta inside an existing transaction.
// Negative deltas clamp the resulting balance at zero instead of failing.
func (s *SysWalletService) ApplyWalletDeltaByOperatorTx(tx *gorm.DB, userID uint, operatorID uint, delta float64, reason, refType, refID string) error {
	if delta == 0 {
		return nil
	}
	wallet, err := s.repo.GetOrCreateWalletTx(tx, userID)
	if err != nil {
		return fmt.Errorf("获取用户钱包失败: %w", err)
	}
	newBalance := wallet.Balance + delta
	if newBalance < 0 {
		newBalance = 0
	}
	return s.applyWalletDeltaTx(tx, userID, operatorID, delta, newBalance, reason, refType, refID)
}

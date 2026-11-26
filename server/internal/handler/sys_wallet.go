package handler

import (
	"amiya-eden/internal/apicontract"
	"amiya-eden/internal/model"
	"amiya-eden/internal/service"
	"amiya-eden/pkg/response"
	"context"
)

// SysWalletHandler 伏羲币 HTTP 处理器
type SysWalletHandler struct {
	svc *service.SysWalletService
}

func NewSysWalletHandler() *SysWalletHandler {
	return &SysWalletHandler{svc: service.NewSysWalletService()}
}

// ─────────────────────────────────────────────
//  用户端（POST 接口）
// ─────────────────────────────────────────────

func (h *SysWalletHandler) getMyShopWallet(userID uint) (apicontract.ShopWallet, error) {
	wallet, err := h.svc.GetMyWallet(userID)
	if err != nil {
		return apicontract.ShopWallet{}, err
	}
	return systemWalletDTO(*wallet), nil
}

func (h *SysWalletHandler) listMyShopWalletTransactions(req apicontract.ListMyShopWalletTransactionsJSONRequestBody, badBody bool, userID uint) (apicontract.ShopWalletTransactionPage, error) {
	if badBody {
		req.Current = 1
		req.Size = 20
	}
	req.Current, req.Size = normalizeLedgerPagination(req.Current, req.Size)

	records, total, err := h.svc.GetMyTransactions(userID, req.Current, req.Size)
	if err != nil {
		return apicontract.ShopWalletTransactionPage{}, err
	}
	listDTO := mapContractRows(records, shopWalletTransactionDTO)
	return apicontract.ShopWalletTransactionPage{List: &listDTO, Total: total, Page: req.Current, PageSize: req.Size}, nil
}

func shopWalletTransactionDTO(record model.TransactionWithCharacter) apicontract.ShopWalletTransaction {
	return apicontract.ShopWalletTransaction{
		ID: record.ID, UserID: record.UserID, Amount: record.Amount, Reason: record.Reason,
		RefType: record.RefType, RefID: record.RefID, BalanceAfter: record.BalanceAfter,
		OperatorID: record.OperatorID, CreatedAt: record.CreatedAt, CharacterName: record.CharacterName,
		Nickname: record.Nickname, OperatorName: record.OperatorName,
	}
}

// ─────────────────────────────────────────────
//  管理员端（POST 接口）
// ─────────────────────────────────────────────

func (h *SysWalletHandler) adminListSystemWallets(req apicontract.AdminListSystemWalletsJSONRequestBody, badBody bool) (apicontract.SystemWalletListPage, error) {
	if badBody {
		req.Current = 1
		req.Size = 20
	}
	req.Current, req.Size = normalizeLedgerPagination(req.Current, req.Size)
	filter := service.WalletListFilter{
		UserKeyword: req.UserKeyword,
		SortBy:      req.SortBy,
		SortOrder:   req.SortOrder,
	}

	wallets, total, err := h.svc.AdminListWallets(req.Current, req.Size, filter)
	if err != nil {
		return apicontract.SystemWalletListPage{}, err
	}
	listDTO := mapContractRows(wallets, systemWalletListItemDTO)
	return apicontract.SystemWalletListPage{List: &listDTO, Total: total, Page: req.Current, PageSize: req.Size}, nil
}

func (h *SysWalletHandler) adminGetSystemWallet(req apicontract.AdminGetSystemWalletJSONRequestBody) (apicontract.ShopWallet, error) {
	wallet, err := h.svc.AdminGetWallet(req.UserID)
	if err != nil {
		return apicontract.ShopWallet{}, err
	}
	return systemWalletDTO(*wallet), nil
}

func (h *SysWalletHandler) adminAdjustSystemWallet(req apicontract.AdminAdjustSystemWalletJSONRequestBody, operatorID uint) (apicontract.ShopWallet, error) {
	wallet, err := h.svc.AdminAdjust(operatorID, &service.AdminAdjustRequest{
		TargetUID: req.TargetUID,
		Action:    req.Action,
		Amount:    req.Amount,
		Reason:    req.Reason,
	})
	if err != nil {
		return apicontract.ShopWallet{}, err
	}
	return systemWalletDTO(*wallet), nil
}

func (h *SysWalletHandler) adminListSystemWalletTransactions(req apicontract.AdminListSystemWalletTransactionsJSONRequestBody, badBody bool) (apicontract.ShopWalletTransactionPage, error) {
	if badBody {
		req.Current = 1
		req.Size = 20
	}
	req.Current, req.Size = normalizeLedgerPagination(req.Current, req.Size)

	filter := service.WalletTransactionFilter{
		UserID:      req.UserID,
		UserKeyword: req.UserKeyword,
		RefType:     req.RefType,
		SortBy:      req.SortBy,
		SortOrder:   req.SortOrder,
	}

	records, total, err := h.svc.AdminListTransactions(req.Current, req.Size, filter)
	if err != nil {
		return apicontract.ShopWalletTransactionPage{}, err
	}
	listDTO := mapContractRows(records, shopWalletTransactionDTO)
	return apicontract.ShopWalletTransactionPage{List: &listDTO, Total: total, Page: req.Current, PageSize: req.Size}, nil
}

func (h *SysWalletHandler) adminListSystemWalletLogs(req apicontract.AdminListSystemWalletLogsJSONRequestBody, badBody bool) (apicontract.SystemWalletLogPage, error) {
	if badBody {
		req.Current = 1
		req.Size = 20
	}
	req.Current, req.Size = normalizeLedgerPagination(req.Current, req.Size)

	filter := service.WalletLogFilter{
		OperatorID: req.OperatorID,
		TargetUID:  req.TargetUID,
		Action:     req.Action,
		SortBy:     req.SortBy,
		SortOrder:  req.SortOrder,
	}

	records, total, err := h.svc.AdminListLogs(req.Current, req.Size, filter)
	if err != nil {
		return apicontract.SystemWalletLogPage{}, err
	}
	listDTO := mapContractRows(records, systemWalletLogDTO)
	return apicontract.SystemWalletLogPage{List: &listDTO, Total: total, Page: req.Current, PageSize: req.Size}, nil
}

func (h *SysWalletHandler) adminGetSystemWalletAnalytics(req apicontract.AdminGetSystemWalletAnalyticsJSONRequestBody) (apicontract.SystemWalletAnalytics, error) {
	result, err := h.svc.AdminGetAnalytics(&service.WalletAnalyticsRequest{
		StartDate:   req.StartDate,
		EndDate:     req.EndDate,
		RefTypes:    req.RefTypes,
		UserKeyword: req.UserKeyword,
		TopN:        req.TopN,
	})
	if err != nil {
		return apicontract.SystemWalletAnalytics{}, err
	}
	return systemWalletAnalyticsDTO(*result), nil
}

// ─────────────────────────────────────────────
//  兼容旧接口：供 fleet 路由复用
// ─────────────────────────────────────────────

func systemWalletDTO(value model.SystemWallet) apicontract.ShopWallet {
	return apicontract.ShopWallet{
		ID:        value.ID,
		UserID:    value.UserID,
		Balance:   value.Balance,
		UpdatedAt: value.UpdatedAt,
	}
}

func systemWalletListItemDTO(value model.WalletWithCharacter) apicontract.SystemWalletListItem {
	return apicontract.SystemWalletListItem{
		ID:            value.ID,
		UserID:        value.UserID,
		Balance:       value.Balance,
		UpdatedAt:     value.UpdatedAt,
		CharacterName: value.CharacterName,
	}
}

func systemWalletLogDTO(value model.LogWithCharacter) apicontract.SystemWalletLog {
	return apicontract.SystemWalletLog{
		ID:                    value.ID,
		OperatorID:            value.OperatorID,
		TargetUID:             value.TargetUID,
		Action:                value.Action,
		Amount:                value.Amount,
		Before:                value.Before,
		After:                 value.After,
		Reason:                value.Reason,
		CreatedAt:             value.CreatedAt,
		TargetCharacterName:   value.TargetCharacterName,
		OperatorCharacterName: value.OperatorCharacterName,
	}
}

func systemWalletAnalyticsSummaryDTO(value service.WalletAnalyticsSummary) apicontract.SystemWalletAnalyticsSummary {
	return apicontract.SystemWalletAnalyticsSummary{
		WalletCount:       value.WalletCount,
		ActiveWalletCount: value.ActiveWalletCount,
		TotalBalance:      value.TotalBalance,
		IncomeTotal:       value.IncomeTotal,
		ExpenseTotal:      value.ExpenseTotal,
		NetFlow:           value.NetFlow,
	}
}

func systemWalletAnalyticsDailyPointDTO(value service.WalletAnalyticsDailyPoint) apicontract.SystemWalletAnalyticsDailyPoint {
	return apicontract.SystemWalletAnalyticsDailyPoint{
		Date:    value.Date,
		Income:  value.Income,
		Expense: value.Expense,
		NetFlow: value.NetFlow,
	}
}

func systemWalletAnalyticsRefTypeItemDTO(value service.WalletAnalyticsRefTypeItem) apicontract.SystemWalletAnalyticsRefTypeItem {
	return apicontract.SystemWalletAnalyticsRefTypeItem{
		RefType: value.RefType,
		Income:  value.Income,
		Expense: value.Expense,
		Count:   value.Count,
	}
}

func systemWalletAnalyticsTopUserDTO(value service.WalletAnalyticsTopUser) apicontract.SystemWalletAnalyticsTopUser {
	return apicontract.SystemWalletAnalyticsTopUser{
		UserID:        value.UserID,
		CharacterName: value.CharacterName,
		Amount:        value.Amount,
	}
}

func systemWalletAnalyticsOperatorItemDTO(value service.WalletAnalyticsOperatorItem) apicontract.SystemWalletAnalyticsOperatorItem {
	return apicontract.SystemWalletAnalyticsOperatorItem{
		OperatorID:   value.OperatorID,
		OperatorName: value.OperatorName,
		Count:        value.Count,
		AmountTotal:  value.AmountTotal,
	}
}

func systemWalletAnalyticsAdminAdjustStatsDTO(value service.WalletAnalyticsAdminAdjustStats) apicontract.SystemWalletAnalyticsAdminAdjustStats {
	return apicontract.SystemWalletAnalyticsAdminAdjustStats{
		Count:       value.Count,
		AmountTotal: value.AmountTotal,
		ByOperator:  mapContractRows(value.ByOperator, systemWalletAnalyticsOperatorItemDTO),
	}
}

func systemWalletAnalyticsLargeTransactionDTO(value service.WalletAnalyticsLargeTransaction) apicontract.SystemWalletAnalyticsLargeTransaction {
	return apicontract.SystemWalletAnalyticsLargeTransaction{
		ID:            value.ID,
		UserID:        value.UserID,
		CharacterName: value.CharacterName,
		Amount:        value.Amount,
		RefType:       value.RefType,
		CreatedAt:     value.CreatedAt,
	}
}

func systemWalletAnalyticsFrequentAdjustmentDTO(value service.WalletAnalyticsFrequentAdjustment) apicontract.SystemWalletAnalyticsFrequentAdjustment {
	return apicontract.SystemWalletAnalyticsFrequentAdjustment{
		TargetUID:          value.TargetUID,
		CharacterName:      value.CharacterName,
		AdjustCount:        value.AdjustCount,
		AmountTotal:        value.AmountTotal,
		LastAdjustmentTime: value.LastAdjustmentTime,
	}
}

func systemWalletAnalyticsOperatorConcentrationDTO(value service.WalletAnalyticsOperatorConcentration) apicontract.SystemWalletAnalyticsOperatorConcentration {
	return apicontract.SystemWalletAnalyticsOperatorConcentration{
		OperatorID:   value.OperatorID,
		OperatorName: value.OperatorName,
		Count:        value.Count,
		AmountTotal:  value.AmountTotal,
		Ratio:        value.Ratio,
	}
}

func systemWalletAnalyticsAnomaliesDTO(value service.WalletAnalyticsAnomalies) apicontract.SystemWalletAnalyticsAnomalies {
	return apicontract.SystemWalletAnalyticsAnomalies{
		LargeTransactions:     mapContractRows(value.LargeTransactions, systemWalletAnalyticsLargeTransactionDTO),
		FrequentAdjustments:   mapContractRows(value.FrequentAdjustments, systemWalletAnalyticsFrequentAdjustmentDTO),
		OperatorConcentration: mapContractRows(value.OperatorConcentration, systemWalletAnalyticsOperatorConcentrationDTO),
	}
}

func systemWalletAnalyticsDTO(value service.WalletAnalyticsResponse) apicontract.SystemWalletAnalytics {
	return apicontract.SystemWalletAnalytics{
		Summary:          systemWalletAnalyticsSummaryDTO(value.Summary),
		DailySeries:      mapContractRows(value.DailySeries, systemWalletAnalyticsDailyPointDTO),
		RefTypeBreakdown: mapContractRows(value.RefTypeBreakdown, systemWalletAnalyticsRefTypeItemDTO),
		TopInflowUsers:   mapContractRows(value.TopInflowUsers, systemWalletAnalyticsTopUserDTO),
		TopOutflowUsers:  mapContractRows(value.TopOutflowUsers, systemWalletAnalyticsTopUserDTO),
		AdminAdjustStats: systemWalletAnalyticsAdminAdjustStatsDTO(value.AdminAdjustStats),
		Anomalies:        systemWalletAnalyticsAnomaliesDTO(value.Anomalies),
	}
}

func (h *SysWalletHandler) GetMyShopWallet(ctx context.Context, _ apicontract.GetMyShopWalletRequestObject) (apicontract.GetMyShopWalletResponseObject, error) {
	out := &apicontract.GetMyShopWallet200JSONResponse{}
	userID, _ := commerceIdentity(ctx)
	data, err := h.getMyShopWallet(userID)
	if err != nil {
		return out, out.FromBusinessError(contractBusinessError(response.CodeBizError, err.Error()))
	}
	return out, out.FromShopWalletSuccess(apicontract.ShopWalletSuccess{Code: response.CodeOK, Msg: "success", Data: data})
}

func (h *SysWalletHandler) ListMyShopWalletTransactions(ctx context.Context, request apicontract.ListMyShopWalletTransactionsRequestObject) (apicontract.ListMyShopWalletTransactionsResponseObject, error) {
	out := &apicontract.ListMyShopWalletTransactions200JSONResponse{}
	body, badBody := commerceOptionalBody(ctx, request.Body)
	userID, _ := commerceIdentity(ctx)
	data, err := h.listMyShopWalletTransactions(body, badBody, userID)
	if err != nil {
		return out, out.FromBusinessError(contractBusinessError(response.CodeBizError, err.Error()))
	}
	return out, out.FromShopWalletTransactionPageSuccess(apicontract.ShopWalletTransactionPageSuccess{Code: response.CodeOK, Msg: "success", Data: data})
}

func (h *SysWalletHandler) AdminListSystemWallets(ctx context.Context, request apicontract.AdminListSystemWalletsRequestObject) (apicontract.AdminListSystemWalletsResponseObject, error) {
	out := &apicontract.AdminListSystemWallets200JSONResponse{}
	body, badBody := commerceOptionalBody(ctx, request.Body)
	data, err := h.adminListSystemWallets(body, badBody)
	if err != nil {
		return out, out.FromBusinessError(contractBusinessError(response.CodeBizError, err.Error()))
	}
	return out, out.FromSystemWalletListPageSuccess(apicontract.SystemWalletListPageSuccess{Code: response.CodeOK, Msg: "success", Data: data})
}

func (h *SysWalletHandler) AdminGetSystemWallet(_ context.Context, request apicontract.AdminGetSystemWalletRequestObject) (apicontract.AdminGetSystemWalletResponseObject, error) {
	out := &apicontract.AdminGetSystemWallet200JSONResponse{}
	if request.Body == nil {
		return out, out.FromBusinessError(commerceMissingBody())
	}
	data, err := h.adminGetSystemWallet(*request.Body)
	if err != nil {
		return out, out.FromBusinessError(contractBusinessError(response.CodeBizError, err.Error()))
	}
	return out, out.FromShopWalletSuccess(apicontract.ShopWalletSuccess{Code: response.CodeOK, Msg: "success", Data: data})
}

func (h *SysWalletHandler) AdminAdjustSystemWallet(ctx context.Context, request apicontract.AdminAdjustSystemWalletRequestObject) (apicontract.AdminAdjustSystemWalletResponseObject, error) {
	out := &apicontract.AdminAdjustSystemWallet200JSONResponse{}
	if request.Body == nil {
		return out, out.FromBusinessError(commerceMissingBody())
	}
	operatorID, _ := commerceIdentity(ctx)
	data, err := h.adminAdjustSystemWallet(*request.Body, operatorID)
	if err != nil {
		return out, out.FromBusinessError(contractBusinessError(response.CodeBizError, err.Error()))
	}
	return out, out.FromShopWalletSuccess(apicontract.ShopWalletSuccess{Code: response.CodeOK, Msg: "success", Data: data})
}

func (h *SysWalletHandler) AdminListSystemWalletTransactions(ctx context.Context, request apicontract.AdminListSystemWalletTransactionsRequestObject) (apicontract.AdminListSystemWalletTransactionsResponseObject, error) {
	out := &apicontract.AdminListSystemWalletTransactions200JSONResponse{}
	body, badBody := commerceOptionalBody(ctx, request.Body)
	data, err := h.adminListSystemWalletTransactions(body, badBody)
	if err != nil {
		return out, out.FromBusinessError(contractBusinessError(response.CodeBizError, err.Error()))
	}
	return out, out.FromShopWalletTransactionPageSuccess(apicontract.ShopWalletTransactionPageSuccess{Code: response.CodeOK, Msg: "success", Data: data})
}

func (h *SysWalletHandler) AdminListSystemWalletLogs(ctx context.Context, request apicontract.AdminListSystemWalletLogsRequestObject) (apicontract.AdminListSystemWalletLogsResponseObject, error) {
	out := &apicontract.AdminListSystemWalletLogs200JSONResponse{}
	body, badBody := commerceOptionalBody(ctx, request.Body)
	data, err := h.adminListSystemWalletLogs(body, badBody)
	if err != nil {
		return out, out.FromBusinessError(contractBusinessError(response.CodeBizError, err.Error()))
	}
	return out, out.FromSystemWalletLogPageSuccess(apicontract.SystemWalletLogPageSuccess{Code: response.CodeOK, Msg: "success", Data: data})
}

func (h *SysWalletHandler) AdminGetSystemWalletAnalytics(_ context.Context, request apicontract.AdminGetSystemWalletAnalyticsRequestObject) (apicontract.AdminGetSystemWalletAnalyticsResponseObject, error) {
	out := &apicontract.AdminGetSystemWalletAnalytics200JSONResponse{}
	if request.Body == nil {
		return out, out.FromBusinessError(commerceMissingBody())
	}
	data, err := h.adminGetSystemWalletAnalytics(*request.Body)
	if err != nil {
		return out, out.FromBusinessError(contractBusinessError(response.CodeParamError, err.Error()))
	}
	return out, out.FromSystemWalletAnalyticsSuccess(apicontract.SystemWalletAnalyticsSuccess{Code: response.CodeOK, Msg: "success", Data: data})
}

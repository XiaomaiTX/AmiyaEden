package handler

import (
	"amiya-eden/internal/apicontract"
	"amiya-eden/internal/service"
	"amiya-eden/pkg/response"
	"context"
)

// DashboardHandler 工作台 HTTP 处理器
type DashboardHandler struct {
	svc *service.DashboardService
}

func NewDashboardHandler() *DashboardHandler {
	return &DashboardHandler{
		svc: service.NewDashboardService(),
	}
}

func dashboardDTO(result service.DashboardResult) apicontract.DashboardResult {
	papMonthly := func(row service.DashboardPapMonthly) apicontract.DashboardPapMonthly {
		return apicontract.DashboardPapMonthly{Year: row.Year, Month: row.Month, TotalPap: row.TotalPap}
	}
	return apicontract.DashboardResult{
		Cards: apicontract.DashboardCards{
			EveWalletBalance: result.Cards.EveWalletBalance, EveSkillPoints: result.Cards.EveSkillPoints,
			SystemWalletBalance: result.Cards.SystemWalletBalance, AlliancePap: result.Cards.AlliancePap,
		},
		Fleets: dashboardDTOs(result.Fleets, func(row service.DashboardFleetItem) apicontract.DashboardFleetItem {
			return apicontract.DashboardFleetItem{
				Source: apicontract.DashboardFleetSource(row.Source), Id: row.ID, Title: row.Title,
				StartAt: row.StartAt, EndAt: row.EndAt, Importance: row.Importance,
				PapCount: row.PapCount, ShipTypeName: row.ShipTypeName, CharacterName: row.CharacterName,
			}
		}),
		PapStats: apicontract.DashboardPapStats{
			Alliance: dashboardDTOs(result.PapStats.Alliance, papMonthly),
			Internal: dashboardDTOs(result.PapStats.Internal, papMonthly),
		},
		SrpList: dashboardDTOs(result.SrpList, func(row service.DashboardSrpItem) apicontract.DashboardSrpItem {
			return apicontract.DashboardSrpItem{
				Id: row.ID, CharacterName: row.CharacterName, ShipName: row.ShipName,
				SolarSystemName: row.SolarSystemName, KillmailTime: row.KillmailTime, CreatedAt: row.CreatedAt,
				RecommendedAmount: row.RecommendedAmount, FinalAmount: row.FinalAmount,
				ReviewStatus: row.ReviewStatus, PayoutStatus: row.PayoutStatus,
			}
		}),
	}
}

func dashboardDTOs[M, D any](rows []M, convert func(M) D) []D {
	if rows == nil {
		return nil
	}
	result := make([]D, len(rows))
	for i, row := range rows {
		result[i] = convert(row)
	}
	return result
}

func (h *DashboardHandler) GetDashboard(ctx context.Context, _ apicontract.GetDashboardRequestObject) (apicontract.GetDashboardResponseObject, error) {
	out := &apicontract.GetDashboard200JSONResponse{}
	_, userID := strictGinRequestDetails(ctx)
	result, err := h.svc.GetDashboard(userID)
	if err != nil {
		return out, out.FromBusinessError(contractBusinessError(response.CodeBizError, err.Error()))
	}
	return out, out.FromDashboardSuccess(apicontract.DashboardSuccess{Code: response.CodeOK, Msg: "success", Data: dashboardDTO(*result)})
}

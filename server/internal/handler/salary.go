package handler

import (
	"amiya-eden/internal/middleware"
	"amiya-eden/internal/service"
	"amiya-eden/pkg/response"
	"github.com/gin-gonic/gin"
)

type SalaryHandler struct{ svc *service.SalaryService }

func NewSalaryHandler() *SalaryHandler { return &SalaryHandler{svc: service.NewSalaryService()} }

func (h *SalaryHandler) OperationRates(c *gin.Context) {
	rates, err := h.svc.OperationRates()
	if err != nil {
		response.Fail(c, response.CodeBizError, "读取操作工资失败，请重试")
		return
	}
	response.OK(c, rates)
}

func (h *SalaryHandler) Get(c *gin.Context) {
	settings, err := h.svc.Settings()
	if err != nil {
		response.Fail(c, response.CodeBizError, "读取工资配置失败")
		return
	}
	response.OK(c, settings)
}

func (h *SalaryHandler) Update(c *gin.Context) {
	var req struct {
		Recruitment          *float64 `json:"recruitment" binding:"required,gte=0,lte=1000000000000"`
		WelfareDelivery      *float64 `json:"welfare_delivery" binding:"required,gte=0,lte=1000000000000"`
		ShopDelivery         *float64 `json:"shop_delivery" binding:"required,gte=0,lte=1000000000000"`
		SRPPayout            *float64 `json:"srp_payout" binding:"required,gte=0,lte=1000000000000"`
		FCSalary             *float64 `json:"fc_salary" binding:"required,gte=0,lte=1000000000000"`
		FCSalaryMonthlyLimit *int     `json:"fc_salary_monthly_limit" binding:"required,gte=0"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, response.CodeParamError, "请填写完整、有效的工资配置")
		return
	}
	settings, err := h.svc.Update(service.SalarySettings{Recruitment: *req.Recruitment, WelfareDelivery: *req.WelfareDelivery, ShopDelivery: *req.ShopDelivery, SRPPayout: *req.SRPPayout, FCSalary: *req.FCSalary, FCSalaryMonthlyLimit: *req.FCSalaryMonthlyLimit}, middleware.GetUserID(c))
	if err != nil {
		response.Fail(c, response.CodeBizError, "保存失败，请检查金额最多两位小数并重试")
		return
	}
	response.OK(c, settings)
}

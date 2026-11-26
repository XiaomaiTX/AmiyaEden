package handler

import (
	"amiya-eden/internal/apicontract"
	"amiya-eden/internal/middleware"
	"amiya-eden/internal/model"
	"amiya-eden/internal/service"
	"amiya-eden/pkg/response"
	"context"
	"github.com/gin-gonic/gin"
	"io"
)

// ShopHandler 商店 HTTP 处理器
type ShopHandler struct {
	svc *service.ShopService
}

func NewShopHandler() *ShopHandler {
	return &ShopHandler{svc: service.NewShopService()}
}

// ─────────────────────────────────────────────
//  用户端（全部 POST）
// ─────────────────────────────────────────────

func (h *ShopHandler) listShopProducts(req apicontract.ListShopProductsJSONRequestBody, badBody bool) (apicontract.ShopProductPage, error) {
	if badBody {
		req.Current = 1
		req.Size = 20
	}
	req.Current, req.Size = normalizePagination(req.Current, req.Size, 20, 100)

	list, total, err := h.svc.ListOnSaleProducts(req.Current, req.Size, req.Type)
	if err != nil {
		return apicontract.ShopProductPage{}, err
	}
	listDTO := mapContractRows(list, shopProductDTO)
	return apicontract.ShopProductPage{List: &listDTO, Total: total, Page: req.Current, PageSize: req.Size}, nil
}

func (h *ShopHandler) getShopProductDetail(req apicontract.GetShopProductDetailJSONRequestBody) (apicontract.ShopProduct, error) {
	product, err := h.svc.GetProductDetail(req.ProductID)
	if err != nil {
		return apicontract.ShopProduct{}, err
	}
	return shopProductDTO(*product), nil
}

func (h *ShopHandler) buyShopProduct(req apicontract.BuyShopProductJSONRequestBody, userID uint) (apicontract.ShopOrder, error) {
	order, err := h.svc.BuyProduct(userID, &service.BuyRequest{
		ProductID: req.ProductID, Quantity: req.Quantity, Remark: req.Remark,
	})
	if err != nil {
		return apicontract.ShopOrder{}, err
	}
	return shopOrderDTO(*order), nil
}

func (h *ShopHandler) listMyShopOrders(req apicontract.ListMyShopOrdersJSONRequestBody, badBody bool, userID uint) (apicontract.ShopOrderPage, error) {
	if badBody {
		req.Current = 1
		req.Size = 20
	}
	req.Current, req.Size = normalizePagination(req.Current, req.Size, 20, 100)

	list, total, err := h.svc.GetMyOrders(userID, req.Current, req.Size, req.Status)
	if err != nil {
		return apicontract.ShopOrderPage{}, err
	}
	return apicontract.ShopOrderPage{List: mapContractRows(list, shopOrderListItemDTO), Total: total, Page: req.Current, PageSize: req.Size}, nil
}

// ─────────────────────────────────────────────
//  管理员端（全部 POST）
// ─────────────────────────────────────────────

func (h *ShopHandler) adminCreateShopProduct(req apicontract.AdminCreateShopProductJSONRequestBody) (apicontract.ShopProduct, error) {
	product := &model.ShopProduct{
		Name:        req.Name,
		Description: req.Description,
		Image:       req.Image,
		Price:       req.Price,
		Stock:       req.Stock,
		MaxPerUser:  req.MaxPerUser,
		LimitPeriod: req.LimitPeriod,
		Type:        req.Type,
		Status:      req.Status,
		SortOrder:   req.SortOrder,
	}

	if err := h.svc.AdminCreateProduct(product); err != nil {
		return apicontract.ShopProduct{}, err
	}
	return shopProductDTO(*product), nil
}

func (h *ShopHandler) adminUpdateShopProduct(req apicontract.AdminUpdateShopProductJSONRequestBody) (apicontract.ShopProduct, error) {
	product, err := h.svc.AdminUpdateProduct(req.ID, &service.AdminProductUpdateRequest{
		Name:        req.Name,
		Description: req.Description,
		Image:       req.Image,
		Price:       req.Price,
		Stock:       req.Stock,
		MaxPerUser:  req.MaxPerUser,
		LimitPeriod: req.LimitPeriod,
		Type:        req.Type,
		Status:      req.Status,
		SortOrder:   req.SortOrder,
	})
	if err != nil {
		return apicontract.ShopProduct{}, err
	}
	return shopProductDTO(*product), nil
}

func (h *ShopHandler) adminDeleteShopProduct(req apicontract.AdminDeleteShopProductJSONRequestBody) error {
	return h.svc.AdminDeleteProduct(req.ID)
}

func (h *ShopHandler) adminListShopProducts(req apicontract.AdminListShopProductsJSONRequestBody, badBody bool) (apicontract.ShopProductPage, error) {
	if badBody {
		req.Current = 1
		req.Size = 20
	}
	req.Current, req.Size = normalizePagination(req.Current, req.Size, 20, 100)

	filter := service.ProductFilter{
		Status: req.Status,
		Type:   req.Type,
		Name:   req.Name,
	}

	list, total, err := h.svc.AdminListProducts(req.Current, req.Size, filter)
	if err != nil {
		return apicontract.ShopProductPage{}, err
	}
	listDTO := mapContractRows(list, shopProductDTO)
	return apicontract.ShopProductPage{List: &listDTO, Total: total, Page: req.Current, PageSize: req.Size}, nil
}

func (h *ShopHandler) adminListShopOrders(req apicontract.AdminListShopOrdersJSONRequestBody, badBody bool) (apicontract.ShopOrderPage, error) {
	if badBody {
		req.Current = 1
		req.Size = 20
	}
	req.Current, req.Size = normalizeLedgerPagination(req.Current, req.Size)

	filter := service.OrderFilter{
		Keyword:  req.Keyword,
		Statuses: req.Statuses,
		Status:   req.Status,
	}

	list, total, err := h.svc.AdminListOrders(req.Current, req.Size, filter)
	if err != nil {
		return apicontract.ShopOrderPage{}, err
	}
	return apicontract.ShopOrderPage{List: mapContractRows(list, shopOrderListItemDTO), Total: total, Page: req.Current, PageSize: req.Size}, nil
}

func (h *ShopHandler) adminDeliverShopOrder(req apicontract.AdminDeliverShopOrderJSONRequestBody, operatorID uint, operatorRoles []string) (apicontract.ShopOrderActionResultData, error) {
	order, mailSummary, err := h.svc.AdminDeliverOrder(req.OrderID, operatorID, operatorRoles, req.Remark)
	if err != nil {
		return apicontract.ShopOrderActionResultData{}, err
	}
	return shopOrderActionDTO(*order, mailSummary), nil
}

func (h *ShopHandler) adminRejectShopOrder(req apicontract.AdminRejectShopOrderJSONRequestBody, operatorID uint) (apicontract.ShopOrder, error) {
	order, err := h.svc.AdminRejectOrder(req.OrderID, operatorID, req.Remark)
	if err != nil {
		return apicontract.ShopOrder{}, err
	}
	return shopOrderDTO(*order), nil
}

func shopProductDTO(product model.ShopProduct) apicontract.ShopProduct {
	return apicontract.ShopProduct{
		ID: product.ID, CreatedAt: product.CreatedAt, UpdatedAt: product.UpdatedAt,
		Name: product.Name, Description: product.Description, Image: product.Image,
		Price: product.Price, Stock: product.Stock, MaxPerUser: product.MaxPerUser,
		LimitPeriod: product.LimitPeriod, Type: product.Type, Status: product.Status, SortOrder: product.SortOrder,
	}
}

func shopOrderDTO(order model.ShopOrder) apicontract.ShopOrder {
	return apicontract.ShopOrder{
		ID: order.ID, CreatedAt: order.CreatedAt, UpdatedAt: order.UpdatedAt,
		OrderNo: order.OrderNo, UserID: order.UserID, MainCharacterName: order.MainCharacterName,
		Nickname: order.Nickname, QQ: order.QQ, DiscordID: order.DiscordID,
		ProductID: order.ProductID, ProductName: order.ProductName, ProductType: order.ProductType,
		Quantity: order.Quantity, UnitPrice: order.UnitPrice, TotalPrice: order.TotalPrice,
		Status: order.Status, TransactionID: order.TransactionID, Remark: order.Remark,
		ReviewedBy: order.ReviewedBy, ReviewedAt: order.ReviewedAt, ReviewRemark: order.ReviewRemark,
	}
}

func shopOrderListItemDTO(order service.ShopOrderResponse) apicontract.ShopOrder {
	item := shopOrderDTO(order.ShopOrder)
	if order.ReviewerName != "" {
		item.ReviewerName = &order.ReviewerName
	}
	return item
}

func shopOrderActionDTO(order model.ShopOrder, mail service.MailAttemptSummary) apicontract.ShopOrderActionResultData {
	return apicontract.ShopOrderActionResultData{
		ID: order.ID, CreatedAt: order.CreatedAt, UpdatedAt: order.UpdatedAt,
		OrderNo: order.OrderNo, UserID: order.UserID, MainCharacterName: order.MainCharacterName,
		Nickname: order.Nickname, QQ: order.QQ, DiscordID: order.DiscordID,
		ProductID: order.ProductID, ProductName: order.ProductName, ProductType: order.ProductType,
		Quantity: order.Quantity, UnitPrice: order.UnitPrice, TotalPrice: order.TotalPrice,
		Status: order.Status, TransactionID: order.TransactionID, Remark: order.Remark,
		ReviewedBy: order.ReviewedBy, ReviewedAt: order.ReviewedAt, ReviewRemark: order.ReviewRemark,
		MailError:                  optionalContractValue(mail.MailError),
		MailID:                     optionalContractValue(mail.MailID),
		MailSenderCharacterID:      optionalContractValue(mail.MailSenderCharacterID),
		MailSenderCharacterName:    optionalContractValue(mail.MailSenderCharacterName),
		MailRecipientCharacterID:   optionalContractValue(mail.MailRecipientCharacterID),
		MailRecipientCharacterName: optionalContractValue(mail.MailRecipientCharacterName),
	}
}

func optionalContractValue[T comparable](value T) *T {
	var zero T
	if value == zero {
		return nil
	}
	return &value
}

// Preserve both nil and empty slices because they have different JSON representations.
func mapContractRows[T, U any](rows []T, dto func(T) U) []U {
	if rows == nil {
		return nil
	}
	result := make([]U, len(rows))
	for i, row := range rows {
		result[i] = dto(row)
	}
	return result
}

// The strict Gin adapter passes the same context used by the authorization middleware.
func commerceIdentity(ctx context.Context) (uint, []string) {
	c, _ := ctx.(*gin.Context)
	if c == nil {
		return 0, nil
	}
	return middleware.GetUserID(c), middleware.GetUserRoles(c)
}

// Optional lists can retain filters decoded before a JSON type error. Distinguish
// a missing or malformed body from a valid JSON null (decoded as a zero value).
func commerceOptionalBody[T any](ctx context.Context, body *T) (T, bool) {
	if body == nil {
		var zero T
		return zero, true
	}
	return *body, apicontract.StrictGinJSONDecodeError(ctx) != nil
}

func commerceMissingBody() apicontract.BusinessError {
	return contractBusinessError(response.CodeParamError, "请求参数错误: "+io.EOF.Error())
}

func (h *ShopHandler) ListShopProducts(ctx context.Context, request apicontract.ListShopProductsRequestObject) (apicontract.ListShopProductsResponseObject, error) {
	out := &apicontract.ListShopProducts200JSONResponse{}
	body, badBody := commerceOptionalBody(ctx, request.Body)
	data, err := h.listShopProducts(body, badBody)
	if err != nil {
		return out, out.FromBusinessError(contractBusinessError(response.CodeBizError, err.Error()))
	}
	return out, out.FromShopProductPageSuccess(apicontract.ShopProductPageSuccess{Code: response.CodeOK, Msg: "success", Data: data})
}

func (h *ShopHandler) GetShopProductDetail(_ context.Context, request apicontract.GetShopProductDetailRequestObject) (apicontract.GetShopProductDetailResponseObject, error) {
	out := &apicontract.GetShopProductDetail200JSONResponse{}
	if request.Body == nil {
		return out, out.FromBusinessError(commerceMissingBody())
	}
	data, err := h.getShopProductDetail(*request.Body)
	if err != nil {
		return out, out.FromBusinessError(contractBusinessError(response.CodeBizError, err.Error()))
	}
	return out, out.FromShopProductSuccess(apicontract.ShopProductSuccess{Code: response.CodeOK, Msg: "success", Data: data})
}

func (h *ShopHandler) BuyShopProduct(ctx context.Context, request apicontract.BuyShopProductRequestObject) (apicontract.BuyShopProductResponseObject, error) {
	out := &apicontract.BuyShopProduct200JSONResponse{}
	if request.Body == nil {
		return out, out.FromBusinessError(commerceMissingBody())
	}
	userID, _ := commerceIdentity(ctx)
	data, err := h.buyShopProduct(*request.Body, userID)
	if err != nil {
		return out, out.FromBusinessError(contractBusinessError(response.CodeBizError, err.Error()))
	}
	return out, out.FromShopOrderSuccess(apicontract.ShopOrderSuccess{Code: response.CodeOK, Msg: "success", Data: data})
}

func (h *ShopHandler) ListMyShopOrders(ctx context.Context, request apicontract.ListMyShopOrdersRequestObject) (apicontract.ListMyShopOrdersResponseObject, error) {
	out := &apicontract.ListMyShopOrders200JSONResponse{}
	body, badBody := commerceOptionalBody(ctx, request.Body)
	userID, _ := commerceIdentity(ctx)
	data, err := h.listMyShopOrders(body, badBody, userID)
	if err != nil {
		return out, out.FromBusinessError(contractBusinessError(response.CodeBizError, err.Error()))
	}
	return out, out.FromShopOrderPageSuccess(apicontract.ShopOrderPageSuccess{Code: response.CodeOK, Msg: "success", Data: data})
}

func (h *ShopHandler) AdminCreateShopProduct(_ context.Context, request apicontract.AdminCreateShopProductRequestObject) (apicontract.AdminCreateShopProductResponseObject, error) {
	out := &apicontract.AdminCreateShopProduct200JSONResponse{}
	if request.Body == nil {
		return out, out.FromBusinessError(commerceMissingBody())
	}
	data, err := h.adminCreateShopProduct(*request.Body)
	if err != nil {
		return out, out.FromBusinessError(contractBusinessError(response.CodeBizError, err.Error()))
	}
	return out, out.FromShopProductSuccess(apicontract.ShopProductSuccess{Code: response.CodeOK, Msg: "success", Data: data})
}

func (h *ShopHandler) AdminUpdateShopProduct(_ context.Context, request apicontract.AdminUpdateShopProductRequestObject) (apicontract.AdminUpdateShopProductResponseObject, error) {
	out := &apicontract.AdminUpdateShopProduct200JSONResponse{}
	if request.Body == nil {
		return out, out.FromBusinessError(commerceMissingBody())
	}
	data, err := h.adminUpdateShopProduct(*request.Body)
	if err != nil {
		return out, out.FromBusinessError(contractBusinessError(response.CodeBizError, err.Error()))
	}
	return out, out.FromShopProductSuccess(apicontract.ShopProductSuccess{Code: response.CodeOK, Msg: "success", Data: data})
}

func (h *ShopHandler) AdminDeleteShopProduct(_ context.Context, request apicontract.AdminDeleteShopProductRequestObject) (apicontract.AdminDeleteShopProductResponseObject, error) {
	out := &apicontract.AdminDeleteShopProduct200JSONResponse{}
	if request.Body == nil {
		return out, out.FromBusinessError(commerceMissingBody())
	}
	if err := h.adminDeleteShopProduct(*request.Body); err != nil {
		return out, out.FromBusinessError(contractBusinessError(response.CodeBizError, err.Error()))
	}
	return out, out.FromNullSuccess(apicontract.NullSuccess{Code: response.CodeOK, Msg: "success"})
}

func (h *ShopHandler) AdminListShopProducts(ctx context.Context, request apicontract.AdminListShopProductsRequestObject) (apicontract.AdminListShopProductsResponseObject, error) {
	out := &apicontract.AdminListShopProducts200JSONResponse{}
	body, badBody := commerceOptionalBody(ctx, request.Body)
	data, err := h.adminListShopProducts(body, badBody)
	if err != nil {
		return out, out.FromBusinessError(contractBusinessError(response.CodeBizError, err.Error()))
	}
	return out, out.FromShopProductPageSuccess(apicontract.ShopProductPageSuccess{Code: response.CodeOK, Msg: "success", Data: data})
}

func (h *ShopHandler) AdminListShopOrders(ctx context.Context, request apicontract.AdminListShopOrdersRequestObject) (apicontract.AdminListShopOrdersResponseObject, error) {
	out := &apicontract.AdminListShopOrders200JSONResponse{}
	body, badBody := commerceOptionalBody(ctx, request.Body)
	data, err := h.adminListShopOrders(body, badBody)
	if err != nil {
		return out, out.FromBusinessError(contractBusinessError(response.CodeBizError, err.Error()))
	}
	return out, out.FromShopOrderPageSuccess(apicontract.ShopOrderPageSuccess{Code: response.CodeOK, Msg: "success", Data: data})
}

func (h *ShopHandler) AdminDeliverShopOrder(ctx context.Context, request apicontract.AdminDeliverShopOrderRequestObject) (apicontract.AdminDeliverShopOrderResponseObject, error) {
	out := &apicontract.AdminDeliverShopOrder200JSONResponse{}
	if request.Body == nil {
		return out, out.FromBusinessError(commerceMissingBody())
	}
	operatorID, operatorRoles := commerceIdentity(ctx)
	data, err := h.adminDeliverShopOrder(*request.Body, operatorID, operatorRoles)
	if err != nil {
		return out, out.FromBusinessError(contractBusinessError(response.CodeBizError, err.Error()))
	}
	return out, out.FromShopOrderActionSuccess(apicontract.ShopOrderActionSuccess{Code: response.CodeOK, Msg: "success", Data: data})
}

func (h *ShopHandler) AdminRejectShopOrder(ctx context.Context, request apicontract.AdminRejectShopOrderRequestObject) (apicontract.AdminRejectShopOrderResponseObject, error) {
	out := &apicontract.AdminRejectShopOrder200JSONResponse{}
	if request.Body == nil {
		return out, out.FromBusinessError(commerceMissingBody())
	}
	operatorID, _ := commerceIdentity(ctx)
	data, err := h.adminRejectShopOrder(*request.Body, operatorID)
	if err != nil {
		return out, out.FromBusinessError(contractBusinessError(response.CodeBizError, err.Error()))
	}
	return out, out.FromShopOrderSuccess(apicontract.ShopOrderSuccess{Code: response.CodeOK, Msg: "success", Data: data})
}

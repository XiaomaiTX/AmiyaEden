package handler

import (
	"context"
	"errors"
	"math"

	"amiya-eden/internal/apicontract"
	"amiya-eden/internal/middleware"
	"amiya-eden/internal/model"
	"amiya-eden/internal/service"
	"amiya-eden/pkg/response"

	"github.com/gin-gonic/gin"
)

type fuxiHallService interface {
	RefreshContributions(ctx context.Context, operatorID uint) error
	PublicHall(viewerID uint) (*service.FuxiHallPublicPageResponse, error)
	UpdateOwnCard(userID, id uint, req *service.FuxiHallOwnCardRequest) (*model.FuxiHallCard, error)
	GetPageConfig(pageKey string) (*model.FuxiHallPage, error)
	UpdatePageConfig(operatorID uint, pageKey string, req *service.FuxiHallUpdatePageRequest) (*model.FuxiHallPage, error)
	ListCards(pageKey string, visibleOnly bool) ([]model.FuxiHallCard, error)
	ListManageCards(pageKey string) ([]service.FuxiHallManageCard, error)
	CreateCard(operatorID uint, req *service.FuxiHallCreateCardRequest) (*model.FuxiHallCard, error)
	UpdateCard(operatorID, id uint, operatorRoles []string, req *service.FuxiHallUpdateCardRequest) (*model.FuxiHallCard, error)
	ReorderCards(operatorID uint, req *service.FuxiHallReorderRequest) error
	DeleteCard(operatorID, id uint) error
}

// FuxiHallHandler 伏羲大厅处理器
type FuxiHallHandler struct {
	svc fuxiHallService
}

func NewFuxiHallHandler() *FuxiHallHandler {
	return &FuxiHallHandler{svc: service.NewFuxiHallService()}
}

var (
	errHallInvalidOwnCardID = errors.New("无效的card")
	errHallInvalidCardID    = errors.New("无效的卡片 ID")
	errHallOwnCardBody      = errors.New("fuxiHall.errors.invalid")
	errHallBody             = errors.New("请求参数错误")
)

func fuxiHallError(err error, fallback string) (int, string) {
	switch err {
	case errHallInvalidOwnCardID, errHallInvalidCardID, errHallOwnCardBody, errHallBody:
		return response.CodeParamError, err.Error()
	}
	if service.IsUserVisibleError(err) {
		return response.CodeBizError, err.Error()
	}
	return response.CodeBizError, fallback
}

func (h *FuxiHallHandler) refreshHallContributions(ctx context.Context, operatorID uint) error {
	return h.svc.RefreshContributions(ctx, operatorID)
}

func (h *FuxiHallHandler) getHall(viewerID uint) (*apicontract.HallPublic, error) {
	data, err := h.svc.PublicHall(viewerID)
	if err != nil || data == nil {
		return nil, err
	}
	return &apicontract.HallPublic{Page: hallPageDTO(data.Page), Cards: hallCardsDTO(data.Cards)}, nil
}

func (h *FuxiHallHandler) updateHallOwnCard(userID, id uint, req *apicontract.HallOwnCardInput) (*apicontract.HallCard, error) {
	if !validHallCardID(id) {
		return nil, errHallInvalidOwnCardID
	}
	if req == nil {
		return nil, errHallOwnCardBody
	}
	card, err := h.svc.UpdateOwnCard(userID, id, &service.FuxiHallOwnCardRequest{Nickname: req.Nickname, DescriptionHTML: req.DescriptionHtml})
	return mapHallObject(card, err, hallCardDTO)
}

func (h *FuxiHallHandler) getHallPageConfig(pageKey string) (*apicontract.HallPage, error) {
	page, err := h.svc.GetPageConfig(pageKey)
	return mapHallObject(page, err, hallPageDTO)
}

func (h *FuxiHallHandler) updateHallPageConfig(operatorID uint, pageKey string, req *apicontract.HallPageInput) (*apicontract.HallPage, error) {
	if req == nil {
		return nil, errHallBody
	}
	page, err := h.svc.UpdatePageConfig(operatorID, pageKey, &service.FuxiHallUpdatePageRequest{
		Floors: hallModelFloors(req.Floors), Title: req.Title, Subtitle: req.Subtitle, DescriptionHTML: req.DescriptionHtml,
	})
	return mapHallObject(page, err, hallPageDTO)
}

func (h *FuxiHallHandler) listHallCards(pageKey string) (apicontract.HallManageCards, error) {
	cards, err := h.svc.ListManageCards(pageKey)
	if err != nil || cards == nil {
		return nil, err
	}
	out := make(apicontract.HallManageCards, len(cards))
	for i, card := range cards {
		out[i] = hallManageCardDTO(card)
	}
	return out, nil
}

func (h *FuxiHallHandler) createHallCard(operatorID uint, req *apicontract.HallCardCreate) (*apicontract.HallCard, error) {
	if req == nil {
		return nil, errHallBody
	}
	card, err := h.svc.CreateCard(operatorID, &service.FuxiHallCreateCardRequest{
		ContributionOffsets: hallContributionInput(req.ContributionOffsets), FloorKey: req.FloorKey,
		IsCurrentManagement: req.IsCurrentManagement, PageKey: req.PageKey, Nickname: req.Nickname,
		MainCharacterName: req.MainCharacterName, TitleTags: req.TitleTags, DescriptionHTML: req.DescriptionHtml,
		AccentColor: req.AccentColor, AvatarShape: req.AvatarShape, FontScale: req.FontScale, Visible: req.Visible,
	})
	return mapHallObject(card, err, hallCardDTO)
}

func (h *FuxiHallHandler) updateHallCard(operatorID, id uint, operatorRoles []string, req *apicontract.HallCardUpdate) (*apicontract.HallCard, error) {
	if !validHallCardID(id) {
		return nil, errHallInvalidCardID
	}
	if req == nil {
		return nil, errHallBody
	}
	card, err := h.svc.UpdateCard(operatorID, id, operatorRoles, &service.FuxiHallUpdateCardRequest{
		ContributionOffsets: hallContributionInput(req.ContributionOffsets), FloorKey: req.FloorKey,
		IsCurrentManagement: req.IsCurrentManagement, Nickname: req.Nickname, MainCharacterName: req.MainCharacterName,
		TitleTags: req.TitleTags, DescriptionHTML: req.DescriptionHtml, AccentColor: req.AccentColor,
		AvatarShape: req.AvatarShape, FontScale: req.FontScale, Visible: req.Visible,
	})
	return mapHallObject(card, err, hallCardDTO)
}

func (h *FuxiHallHandler) reorderHallCards(operatorID uint, req *apicontract.HallReorderInput) error {
	if req == nil {
		return errHallBody
	}
	return h.svc.ReorderCards(operatorID, &service.FuxiHallReorderRequest{PageKey: req.PageKey, OrderedIDs: req.OrderedIds})
}

func (h *FuxiHallHandler) deleteHallCard(operatorID, id uint) error {
	if !validHallCardID(id) {
		return errHallInvalidCardID
	}
	return h.svc.DeleteCard(operatorID, id)
}

func validHallCardID(id uint) bool {
	return id > 0 && id <= math.MaxUint32
}

func mapHallObject[T, U any](value *T, err error, mapper func(T) U) (*U, error) {
	if err != nil || value == nil {
		return nil, err
	}
	out := mapper(*value)
	return &out, nil
}

func (h *FuxiHallHandler) GetHall(ctx context.Context, _ apicontract.GetHallRequestObject) (apicontract.GetHallResponseObject, error) {
	data, err := h.getHall(middleware.GetUserID(ctx.(*gin.Context)))
	var out apicontract.GetHall200JSONResponse
	if err != nil || data == nil {
		return &out, out.FromBusinessError(hallBusinessError(err, "fuxiHall.errors.load"))
	}
	return &out, out.FromHallPublicResultSuccess(apicontract.HallPublicResultSuccess{Code: response.CodeOK, Msg: "success", Data: *data})
}

func (h *FuxiHallHandler) UpdateHallOwnCard(ctx context.Context, request apicontract.UpdateHallOwnCardRequestObject) (apicontract.UpdateHallOwnCardResponseObject, error) {
	data, err := h.updateHallOwnCard(middleware.GetUserID(ctx.(*gin.Context)), request.Id, request.Body)
	var out apicontract.UpdateHallOwnCard200JSONResponse
	if err != nil || data == nil {
		return &out, out.FromBusinessError(hallBusinessError(err, "fuxiHall.errors.save"))
	}
	return &out, out.FromHallCardResultSuccess(hallCardSuccess(*data))
}

func (h *FuxiHallHandler) RefreshContributions(ctx context.Context, _ apicontract.RefreshContributionsRequestObject) (apicontract.RefreshContributionsResponseObject, error) {
	c := ctx.(*gin.Context)
	err := h.refreshHallContributions(c.Request.Context(), middleware.GetUserID(c))
	var out apicontract.RefreshContributions200JSONResponse
	if err != nil {
		return &out, out.FromBusinessError(hallBusinessError(err, "fuxiHall.errors.refresh"))
	}
	return &out, out.FromNullSuccess(hallNullSuccess())
}

func (h *FuxiHallHandler) GetHallPageConfig(_ context.Context, request apicontract.GetHallPageConfigRequestObject) (apicontract.GetHallPageConfigResponseObject, error) {
	data, err := h.getHallPageConfig(request.PageKey)
	var out apicontract.GetHallPageConfig200JSONResponse
	if err != nil || data == nil {
		return &out, out.FromBusinessError(hallBusinessError(err, "获取页面配置失败"))
	}
	return &out, out.FromHallPageResultSuccess(hallPageSuccess(*data))
}

func (h *FuxiHallHandler) UpdateHallPageConfig(ctx context.Context, request apicontract.UpdateHallPageConfigRequestObject) (apicontract.UpdateHallPageConfigResponseObject, error) {
	data, err := h.updateHallPageConfig(middleware.GetUserID(ctx.(*gin.Context)), request.PageKey, request.Body)
	var out apicontract.UpdateHallPageConfig200JSONResponse
	if err != nil || data == nil {
		return &out, out.FromBusinessError(hallBusinessError(err, "更新页面配置失败"))
	}
	return &out, out.FromHallPageResultSuccess(hallPageSuccess(*data))
}

func (h *FuxiHallHandler) ListHallCards(_ context.Context, request apicontract.ListHallCardsRequestObject) (apicontract.ListHallCardsResponseObject, error) {
	// The GET route's card_key names a page; only PUT and DELETE use a numeric card ID.
	data, err := h.listHallCards(request.CardKey)
	var out apicontract.ListHallCards200JSONResponse
	if err != nil {
		return &out, out.FromBusinessError(hallBusinessError(err, "获取卡片列表失败"))
	}
	if data == nil {
		data = apicontract.HallManageCards{}
	}
	return &out, out.FromHallCardsResultSuccess(apicontract.HallCardsResultSuccess{Code: response.CodeOK, Msg: "success", Data: data})
}

func (h *FuxiHallHandler) CreateCard(ctx context.Context, request apicontract.CreateCardRequestObject) (apicontract.CreateCardResponseObject, error) {
	data, err := h.createHallCard(middleware.GetUserID(ctx.(*gin.Context)), request.Body)
	var out apicontract.CreateCard200JSONResponse
	if err != nil || data == nil {
		return &out, out.FromBusinessError(hallBusinessError(err, "创建卡片失败"))
	}
	return &out, out.FromHallCardResultSuccess(hallCardSuccess(*data))
}

func (h *FuxiHallHandler) UpdateHallCard(ctx context.Context, request apicontract.UpdateHallCardRequestObject) (apicontract.UpdateHallCardResponseObject, error) {
	c := ctx.(*gin.Context)
	data, err := h.updateHallCard(middleware.GetUserID(c), request.CardKey, middleware.GetUserRoles(c), request.Body)
	var out apicontract.UpdateHallCard200JSONResponse
	if err != nil || data == nil {
		return &out, out.FromBusinessError(hallBusinessError(err, "更新卡片失败"))
	}
	return &out, out.FromHallCardResultSuccess(hallCardSuccess(*data))
}

func (h *FuxiHallHandler) ReorderCards(ctx context.Context, request apicontract.ReorderCardsRequestObject) (apicontract.ReorderCardsResponseObject, error) {
	err := h.reorderHallCards(middleware.GetUserID(ctx.(*gin.Context)), request.Body)
	var out apicontract.ReorderCards200JSONResponse
	if err != nil {
		return &out, out.FromBusinessError(hallBusinessError(err, "更新排序失败"))
	}
	return &out, out.FromNullSuccess(hallNullSuccess())
}

func (h *FuxiHallHandler) DeleteHallCard(ctx context.Context, request apicontract.DeleteHallCardRequestObject) (apicontract.DeleteHallCardResponseObject, error) {
	err := h.deleteHallCard(middleware.GetUserID(ctx.(*gin.Context)), request.CardKey)
	var out apicontract.DeleteHallCard200JSONResponse
	if err != nil {
		return &out, out.FromBusinessError(hallBusinessError(err, "删除卡片失败"))
	}
	return &out, out.FromNullSuccess(hallNullSuccess())
}

func hallBusinessError(err error, fallback string) apicontract.BusinessError {
	// A nil object with no error is not a production service success. Do not emit a null object outside the contract.
	if err == nil {
		return contractBusinessError(response.CodeBizError, fallback)
	}
	code, msg := fuxiHallError(err, fallback)
	return contractBusinessError(code, msg)
}

func hallCardSuccess(data apicontract.HallCard) apicontract.HallCardResultSuccess {
	return apicontract.HallCardResultSuccess{Code: response.CodeOK, Msg: "success", Data: data}
}

func hallPageSuccess(data apicontract.HallPage) apicontract.HallPageResultSuccess {
	return apicontract.HallPageResultSuccess{Code: response.CodeOK, Msg: "success", Data: data}
}

func hallNullSuccess() apicontract.NullSuccess {
	return apicontract.NullSuccess{Code: response.CodeOK, Msg: "success"}
}

func hallPageDTO(page model.FuxiHallPage) apicontract.HallPage {
	var floors []apicontract.HallFloor
	if page.Floors != nil {
		floors = make([]apicontract.HallFloor, len(page.Floors))
		for i, floor := range page.Floors {
			floors[i] = apicontract.HallFloor{Key: floor.Key, Name: floor.Name, Capacity: floor.Capacity}
		}
	}
	return apicontract.HallPage{
		Id: page.ID, PageKey: apicontract.HallPageKey(page.PageKey), Title: page.Title,
		Subtitle: page.Subtitle, DescriptionHtml: page.DescriptionHTML, Floors: floors,
	}
}

func hallModelFloors(floors *[]apicontract.HallFloor) *[]model.FuxiHallFloor {
	if floors == nil {
		return nil
	}
	out := make([]model.FuxiHallFloor, len(*floors))
	for i, floor := range *floors {
		out[i] = model.FuxiHallFloor{Key: floor.Key, Name: floor.Name, Capacity: floor.Capacity}
	}
	return &out
}

func hallContributionInput(value *apicontract.HallContributionInput) *model.FuxiHallContributions {
	if value == nil {
		return nil
	}
	return &model.FuxiHallContributions{
		CorpPAP: value.CorpPap, Fleets: value.Fleets, Welfare: value.Welfare,
		SRP: value.Srp, Recruitment: value.Recruitment, Mentoring: value.Mentoring,
	}
}

func hallContributionDTO(value model.FuxiHallContributions) apicontract.HallContributions {
	return apicontract.HallContributions{
		CorpPap: value.CorpPAP, Fleets: value.Fleets, Welfare: value.Welfare,
		Srp: value.SRP, Recruitment: value.Recruitment, Mentoring: value.Mentoring,
	}
}

func hallCardsDTO(cards []model.FuxiHallCard) []apicontract.HallCard {
	if cards == nil {
		return nil
	}
	out := make([]apicontract.HallCard, len(cards))
	for i, card := range cards {
		out[i] = hallCardDTO(card)
	}
	return out
}

func hallCardDTO(card model.FuxiHallCard) apicontract.HallCard {
	return apicontract.HallCard{
		Id: card.ID, PageKey: apicontract.HallPageKey(card.PageKey), FloorKey: card.FloorKey, Nickname: card.Nickname,
		IsCurrentManagement: card.IsCurrentManagement, MainCharacterName: card.MainCharacterName,
		TitleTags: card.TitleTags, DescriptionHtml: card.DescriptionHTML, AccentColor: card.AccentColor,
		AvatarShape: apicontract.HallAvatarShape(card.AvatarShape), FontScale: card.FontScale, Visible: card.Visible,
		Qq: card.QQ, Contributions: hallContributionDTO(card.Contributions), ContributionsUpdatedAt: card.ContributionsUpdatedAt,
		CanEdit: card.CanEdit, TenureMonths: card.TenureMonths, MainCharacterId: card.MainCharacterID,
		AvatarImage: card.AvatarImage, SortOrder: card.SortOrder, CreatedAt: card.CreatedAt, UpdatedAt: card.UpdatedAt,
	}
}

func hallManageCardDTO(card service.FuxiHallManageCard) apicontract.HallManageCard {
	base := hallCardDTO(card.FuxiHallCard)
	return apicontract.HallManageCard{
		Id: base.Id, PageKey: base.PageKey, FloorKey: base.FloorKey, Nickname: base.Nickname,
		IsCurrentManagement: base.IsCurrentManagement, MainCharacterName: base.MainCharacterName,
		TitleTags: base.TitleTags, DescriptionHtml: base.DescriptionHtml, AccentColor: base.AccentColor,
		AvatarShape: base.AvatarShape, FontScale: base.FontScale, Visible: base.Visible,
		Qq: base.Qq, Contributions: base.Contributions, ContributionsUpdatedAt: base.ContributionsUpdatedAt,
		CanEdit: base.CanEdit, TenureMonths: base.TenureMonths, MainCharacterId: base.MainCharacterId,
		AvatarImage: base.AvatarImage, SortOrder: base.SortOrder, CreatedAt: base.CreatedAt, UpdatedAt: base.UpdatedAt,
		ContributionOffsets: hallContributionDTO(card.ContributionOffsets),
	}
}

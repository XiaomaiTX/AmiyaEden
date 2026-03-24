package service

import (
	"amiya-eden/global"
	"amiya-eden/internal/model"
	"amiya-eden/internal/repository"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"net/http"
	"strings"
	"time"

	"go.uber.org/zap"
)

// SrpService 补损业务逻辑层
type SrpService struct {
	repo      *repository.SrpRepository
	fleetRepo *repository.FleetRepository
	charRepo  *repository.EveCharacterRepository
	userRepo  *repository.UserRepository
	sdeRepo   *repository.SdeRepository
	ssoSvc    *EveSSOService
}

func NewSrpService() *SrpService {
	return &SrpService{
		repo:      repository.NewSrpRepository(),
		fleetRepo: repository.NewFleetRepository(),
		charRepo:  repository.NewEveCharacterRepository(),
		userRepo:  repository.NewUserRepository(),
		sdeRepo:   repository.NewSdeRepository(),
		ssoSvc:    NewEveSSOService(),
	}
}

const esiMailSendScope = "esi-mail.send_mail.v1"

// ─────────────────────────────────────────────
//  KM 解析辅助
// ─────────────────────────────────────────────

// resolveCharacterKillmail 确认 killmailID 与 characterID 有关联，并返回 EveKillmailList
func resolveCharacterKillmail(killmailID int64, characterID int64) (*model.EveKillmailList, error) {
	// 验证角色-KM 关联关系
	var ckm model.EveCharacterKillmail
	if err := global.DB.Where("character_id = ? AND killmail_id = ?", characterID, killmailID).First(&ckm).Error; err != nil {
		return nil, errors.New("该 KM 不属于指定角色，或尚未被 ESI 刷新任务录入")
	}
	// 加载 KM 详情
	var km model.EveKillmailList
	if err := global.DB.Where("kill_mail_id = ?", killmailID).First(&km).Error; err != nil {
		return nil, errors.New("KM 详情不存在")
	}
	return &km, nil
}

// ─────────────────────────────────────────────
//  舰船价格表
// ─────────────────────────────────────────────

// ListShipPrices 返回所有（可按关键字过滤）舰船价格
func (s *SrpService) ListShipPrices(keyword string) ([]model.SrpShipPrice, error) {
	return s.repo.ListShipPrices(keyword)
}

// UpsertShipPriceRequest 创建/更新舰船价格请求
type UpsertShipPriceRequest struct {
	ID         uint    `json:"id"` // 0=新建，非0=更新
	ShipTypeID int64   `json:"ship_type_id" binding:"required"`
	ShipName   string  `json:"ship_name"    binding:"required"`
	Amount     float64 `json:"amount"       binding:"required,min=0"`
}

// UpsertShipPrice 创建或更新舰船价格
func (s *SrpService) UpsertShipPrice(userID uint, req *UpsertShipPriceRequest) (*model.SrpShipPrice, error) {
	p := &model.SrpShipPrice{
		ID:         req.ID,
		ShipTypeID: req.ShipTypeID,
		ShipName:   req.ShipName,
		Amount:     req.Amount,
		UpdatedBy:  userID,
	}
	if req.ID == 0 {
		p.CreatedBy = userID
	} else {
		// 保留原始 created_by
		existing, err := s.repo.GetShipPriceByTypeID(req.ShipTypeID)
		if err == nil {
			p.CreatedBy = existing.CreatedBy
		}
	}
	if err := s.repo.UpsertShipPrice(p); err != nil {
		return nil, err
	}
	return p, nil
}

// DeleteShipPrice 删除舰船价格
func (s *SrpService) DeleteShipPrice(id uint) error {
	return s.repo.DeleteShipPrice(id)
}

// ─────────────────────────────────────────────
//  申请提交
// ─────────────────────────────────────────────

// SubmitApplicationRequest 提交补损申请请求
type SubmitApplicationRequest struct {
	CharacterID int64   `json:"character_id"  binding:"required"` // 受损角色 ID
	KillmailID  int64   `json:"killmail_id"   binding:"required"` // zkillboard killmail id
	FleetID     *string `json:"fleet_id"`                         // 关联舰队（可选）
	Note        string  `json:"note"`                             // 备注（无舰队时必填）
	FinalAmount float64 `json:"final_amount"`                     // 用户可以修改推荐金额（后台也可修改）
}

// SubmitApplication 提交补损申请
func (s *SrpService) SubmitApplication(userID uint, req *SubmitApplicationRequest) (*model.SrpApplication, error) {
	// 1. 验证角色属于当前用户
	char, err := s.charRepo.GetByCharacterID(req.CharacterID)
	if err != nil || char.UserID != userID {
		return nil, errors.New("角色不属于当前用户或不存在")
	}

	// 2. 无舰队时需要填写备注
	if req.FleetID == nil && req.Note == "" {
		return nil, errors.New("未关联舰队时，备注不能为空")
	}

	// 3. 检查是否重复提交
	if s.repo.ExistsApplicationByKillmail(req.KillmailID, req.CharacterID) {
		return nil, errors.New("该 KM 已提交过补损申请，不能重复提交")
	}

	// 4. 获取 KM 详情（验证角色与 KM 关联）
	km, err := resolveCharacterKillmail(req.KillmailID, req.CharacterID)
	if err != nil {
		return nil, err
	}

	// 5. 确认该 KM 的受害者确实是这个角色
	if km.CharacterID != req.CharacterID {
		return nil, errors.New("该 KM 的受害者不是指定角色，无法申请补损")
	}

	// 6. 关联舰队时验证
	if req.FleetID != nil && *req.FleetID != "" {
		fleet, ferr := s.fleetRepo.GetByID(*req.FleetID)
		if ferr != nil {
			return nil, errors.New("关联的舰队不存在")
		}
		// KM 时间必须在舰队时间范围内
		if km.KillmailTime.Before(fleet.StartAt) || km.KillmailTime.After(fleet.EndAt) {
			return nil, errors.New("KM 时间不在舰队活动时间范围内")
		}
		// 角色必须是舰队成员
		members, _ := s.fleetRepo.ListMembers(*req.FleetID)
		isMember := false
		for _, m := range members {
			if m.CharacterID == req.CharacterID {
				isMember = true
				break
			}
		}
		if !isMember {
			return nil, errors.New("该角色不是该舰队的成员，无法申请补损")
		}
	}

	// 7. 查找推荐金额
	recommended := 0.0
	if priceRecord, perr := s.repo.GetShipPriceByTypeID(km.ShipTypeID); perr == nil {
		recommended = priceRecord.Amount
	}

	finalAmount := req.FinalAmount
	if finalAmount <= 0 {
		finalAmount = recommended
	}

	// 8. 构建申请
	app := &model.SrpApplication{
		UserID:            userID,
		CharacterID:       req.CharacterID,
		CharacterName:     char.CharacterName,
		KillmailID:        req.KillmailID,
		FleetID:           req.FleetID,
		Note:              req.Note,
		ShipTypeID:        km.ShipTypeID,
		ShipName:          "", // 由前端或 SDE 填写；此处留空
		SolarSystemID:     km.SolarSystemID,
		SolarSystemName:   "", // 同上
		KillmailTime:      km.KillmailTime,
		CorporationID:     km.CorporationID,
		AllianceID:        km.AllianceID,
		RecommendedAmount: recommended,
		FinalAmount:       finalAmount,
		ReviewStatus:      model.SrpReviewPending,
		PayoutStatus:      model.SrpPayoutPending,
	}

	if err := s.repo.CreateApplication(app); err != nil {
		return nil, err
	}
	return app, nil
}

// ─────────────────────────────────────────────
//  申请列表（管理端）
// ─────────────────────────────────────────────

// SrpApplicationResponse 补损申请响应（含舰队信息）
type SrpApplicationResponse struct {
	model.SrpApplication
	FleetTitle  string `json:"fleet_title,omitempty"`
	FleetFCName string `json:"fleet_fc_name,omitempty"`
}

// enrichWithFleetInfo 为申请列表填充舰队信息
func (s *SrpService) enrichWithFleetInfo(apps []model.SrpApplication) []SrpApplicationResponse {
	result := make([]SrpApplicationResponse, len(apps))
	// 收集所有非空 fleet_id
	fleetIDSet := make(map[string]bool)
	for _, app := range apps {
		if app.FleetID != nil && *app.FleetID != "" {
			fleetIDSet[*app.FleetID] = true
		}
	}
	// 批量查询舰队信息
	fleetMap := make(map[string]*model.Fleet)
	for fid := range fleetIDSet {
		if fleet, err := s.fleetRepo.GetByID(fid); err == nil {
			fleetMap[fid] = fleet
		}
	}
	// 组装响应
	for i, app := range apps {
		resp := SrpApplicationResponse{SrpApplication: app}
		if app.FleetID != nil && *app.FleetID != "" {
			if fleet, ok := fleetMap[*app.FleetID]; ok {
				resp.FleetTitle = fleet.Title
				resp.FleetFCName = fleet.FCCharacterName
			}
		}
		result[i] = resp
	}
	return result
}

// ListApplications 管理员端分页查询申请列表
func (s *SrpService) ListApplications(page, pageSize int, filter repository.SrpApplicationFilter) ([]SrpApplicationResponse, int64, error) {
	apps, total, err := s.repo.ListApplications(page, pageSize, filter)
	if err != nil {
		return nil, 0, err
	}
	return s.enrichWithFleetInfo(apps), total, nil
}

// ListMyApplications 当前用户申请列表
func (s *SrpService) ListMyApplications(userID uint, page, pageSize int) ([]SrpApplicationResponse, int64, error) {
	apps, total, err := s.repo.ListMyApplications(userID, page, pageSize)
	if err != nil {
		return nil, 0, err
	}
	return s.enrichWithFleetInfo(apps), total, nil
}

// GetApplication 查询单条申请
func (s *SrpService) GetApplication(id uint) (*SrpApplicationResponse, error) {
	app, err := s.repo.GetApplicationByID(id)
	if err != nil {
		return nil, err
	}
	resp := &SrpApplicationResponse{SrpApplication: *app}
	if app.FleetID != nil && *app.FleetID != "" {
		if fleet, ferr := s.fleetRepo.GetByID(*app.FleetID); ferr == nil {
			resp.FleetTitle = fleet.Title
			resp.FleetFCName = fleet.FCCharacterName
		}
	}
	return resp, nil
}

// ─────────────────────────────────────────────
//  审批
// ─────────────────────────────────────────────

// ReviewApplicationRequest 审批请求
type ReviewApplicationRequest struct {
	Action      string  `json:"action"       binding:"required,oneof=approve reject"` // "approve" | "reject"
	ReviewNote  string  `json:"review_note"`                                          // 拒绝时必须填写
	FinalAmount float64 `json:"final_amount"`                                         // 批准时可以修改金额
}

// ReviewApplication 审批补损申请（srp/fc/admin 可操作）
// 支持对已批准/已拒绝的申请重新审批（编辑/重新拒绝）
func (s *SrpService) ReviewApplication(reviewerID uint, appID uint, req *ReviewApplicationRequest) (*model.SrpApplication, error) {
	app, err := s.repo.GetApplicationByID(appID)
	if err != nil {
		return nil, errors.New("申请不存在")
	}
	// 已发放的申请不允许重新审批
	if app.PayoutStatus == model.SrpPayoutPaid {
		return nil, errors.New("该申请已发放，不能修改审批状态")
	}
	if req.Action == "reject" && req.ReviewNote == "" {
		return nil, errors.New("拒绝时必须填写审批备注")
	}

	now := time.Now()
	app.ReviewedBy = &reviewerID
	app.ReviewedAt = &now
	app.ReviewNote = req.ReviewNote

	switch req.Action {
	case "approve":
		app.ReviewStatus = model.SrpReviewApproved
		if req.FinalAmount > 0 {
			app.FinalAmount = req.FinalAmount
		}
	case "reject":
		app.ReviewStatus = model.SrpReviewRejected
	}

	if err := s.repo.UpdateApplication(app); err != nil {
		return nil, err
	}
	return app, nil
}

// ─────────────────────────────────────────────
//  发放
// ─────────────────────────────────────────────

// PayoutRequest 发放请求
type SrpPayoutRequest struct {
	FinalAmount float64 `json:"final_amount"` // 允许最终覆盖金额（0=保持原值）
}

// SrpPayoutBatchRequest 批量发放请求
type SrpPayoutBatchRequest struct {
	ApplicationIDs []uint             `json:"application_ids" binding:"required,min=1"`
	FinalAmountMap map[string]float64 `json:"final_amount_map"`
}

// SrpPayoutBatchFailure 单条发放失败
type SrpPayoutBatchFailure struct {
	ApplicationID uint   `json:"application_id"`
	Reason        string `json:"reason"`
}

// SrpPayoutMailFailure 单条邮件失败
type SrpPayoutMailFailure struct {
	ApplicationID uint   `json:"application_id"`
	Reason        string `json:"reason"`
}

// SrpPayoutBatchResult 批量发放结果
type SrpPayoutBatchResult struct {
	PayoutSuccessCount int                     `json:"payout_success_count"`
	PayoutFailedCount  int                     `json:"payout_failed_count"`
	PayoutFailures     []SrpPayoutBatchFailure `json:"payout_failures"`
	MailSuccessCount   int                     `json:"mail_success_count"`
	MailFailedCount    int                     `json:"mail_failed_count"`
	MailFailures       []SrpPayoutMailFailure  `json:"mail_failures"`
}

// Payout 发放补损（srp/admin 可操作）
func (s *SrpService) Payout(payerID uint, appID uint, req *SrpPayoutRequest) (*model.SrpApplication, error) {
	app, err := s.payoutCore(payerID, appID, req)
	if err != nil {
		return nil, err
	}

	// 发放成功后发送 EVE 邮件（失败不回滚发放）
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := s.sendPayoutMail(ctx, payerID, app); err != nil {
		global.Logger.Warn("SRP 发放邮件发送失败",
			zap.Uint("application_id", app.ID),
			zap.Uint("payer_user_id", payerID),
			zap.Error(err),
		)
	}

	return app, nil
}

func (s *SrpService) payoutCore(payerID uint, appID uint, req *SrpPayoutRequest) (*model.SrpApplication, error) {
	app, err := s.repo.GetApplicationByID(appID)
	if err != nil {
		return nil, errors.New("申请不存在")
	}
	if app.ReviewStatus != model.SrpReviewApproved {
		return nil, errors.New("申请未被批准，无法发放")
	}
	if app.PayoutStatus == model.SrpPayoutPaid {
		return nil, errors.New("该申请已发放，不能重复操作")
	}
	if req.FinalAmount > 0 {
		app.FinalAmount = req.FinalAmount
	}
	now := time.Now()
	app.PayoutStatus = model.SrpPayoutPaid
	app.PaidBy = &payerID
	app.PaidAt = &now

	if err := s.repo.UpdateApplication(app); err != nil {
		return nil, err
	}
	return app, nil
}

// PayoutBatch 批量发放补损（srp/admin 可操作）
func (s *SrpService) PayoutBatch(payerID uint, req *SrpPayoutBatchRequest) (*SrpPayoutBatchResult, error) {
	result := &SrpPayoutBatchResult{
		PayoutFailures: make([]SrpPayoutBatchFailure, 0),
		MailFailures:   make([]SrpPayoutMailFailure, 0),
	}

	successApps := make([]*model.SrpApplication, 0)

	for _, appID := range req.ApplicationIDs {
		payoutReq := &SrpPayoutRequest{}
		if override, ok := req.FinalAmountMap[fmt.Sprintf("%d", appID)]; ok {
			payoutReq.FinalAmount = override
		}

		app, err := s.payoutCore(payerID, appID, payoutReq)
		if err != nil {
			result.PayoutFailedCount++
			result.PayoutFailures = append(result.PayoutFailures, SrpPayoutBatchFailure{
				ApplicationID: appID,
				Reason:        err.Error(),
			})
			continue
		}

		result.PayoutSuccessCount++
		successApps = append(successApps, app)
	}

	mailFailureMap := s.sendPayoutMailBatch(context.Background(), payerID, successApps)
	for _, app := range successApps {
		reason, failed := mailFailureMap[app.ID]
		if !failed {
			result.MailSuccessCount++
			continue
		}
		result.MailFailedCount++
		result.MailFailures = append(result.MailFailures, SrpPayoutMailFailure{
			ApplicationID: app.ID,
			Reason:        reason,
		})
	}

	return result, nil
}

func (s *SrpService) sendPayoutMailBatch(ctx context.Context, payerID uint, apps []*model.SrpApplication) map[uint]string {
	failures := make(map[uint]string)
	if len(apps) == 0 {
		return failures
	}

	senderCharacterID, err := s.resolveUserPrimaryCharacterID(payerID)
	if err != nil {
		for _, app := range apps {
			s.logPayoutMailFailed(app.ID, 0, 0, err)
			failures[app.ID] = err.Error()
		}
		return failures
	}

	senderChar, err := s.charRepo.GetByCharacterID(senderCharacterID)
	if err != nil {
		reason := fmt.Sprintf("发信角色不存在: %v", err)
		for _, app := range apps {
			s.logPayoutMailFailed(app.ID, senderCharacterID, 0, errors.New(reason))
			failures[app.ID] = reason
		}
		return failures
	}

	if !hasScope(senderChar.Scopes, esiMailSendScope) {
		reason := fmt.Sprintf("发信角色未授权 scope: %s", esiMailSendScope)
		for _, app := range apps {
			s.logPayoutMailFailed(app.ID, senderCharacterID, 0, errors.New(reason))
			failures[app.ID] = reason
		}
		return failures
	}

	token, err := s.ssoSvc.GetValidToken(ctx, senderCharacterID)
	if err != nil {
		reason := fmt.Sprintf("获取发信 token 失败: %v", err)
		for _, app := range apps {
			s.logPayoutMailFailed(app.ID, senderCharacterID, 0, errors.New(reason))
			failures[app.ID] = reason
		}
		return failures
	}

	grouped := make(map[int64][]*model.SrpApplication)

	// 在批量路径中缓存每个用户的主角色，避免同一 user 多次查库
	userPrimaryCharCache := make(map[uint]int64)
	userPrimaryCharErr := make(map[uint]error)

	for _, app := range apps {
		if s.repo.HasPayoutMailSuccess(app.ID) {
			continue
		}

		// 先从缓存中查找该用户的主角色 ID
		recipientCharacterID, ok := userPrimaryCharCache[app.UserID]
		if !ok {
			// 如果之前解析该用户时出过错，直接复用错误信息
			if err, existed := userPrimaryCharErr[app.UserID]; existed {
				s.logPayoutMailFailed(app.ID, senderCharacterID, 0, err)
				failures[app.ID] = err.Error()
				continue
			}

			resolvedID, err := s.resolveUserPrimaryCharacterID(app.UserID)
			if err != nil {
				// 记录该用户解析失败，后续同一用户的申请复用该错误
				userPrimaryCharErr[app.UserID] = err
				s.logPayoutMailFailed(app.ID, senderCharacterID, 0, err)
				failures[app.ID] = err.Error()
				continue
			}

			recipientCharacterID = resolvedID
			userPrimaryCharCache[app.UserID] = resolvedID
		}
		grouped[recipientCharacterID] = append(grouped[recipientCharacterID], app)
	}

	for recipientCharacterID, groupApps := range grouped {
		fleetTitles := make(map[uint]string, len(groupApps))
		for _, app := range groupApps {
			fleetTitles[app.ID] = s.resolveFleetTitle(app)
		}

		subject, body := buildBatchPayoutMailContent(groupApps, fleetTitles)
		mailID, statusCode, err := sendEveMail(ctx, senderCharacterID, token, eveMailSendRequest{
			Subject: subject,
			Body:    body,
			Recipients: []eveMailRecipient{
				{RecipientID: recipientCharacterID, RecipientType: "character"},
			},
		})
		if err != nil {
			wrappedErr := fmt.Errorf("ESI 邮件发送失败(status=%d): %w", statusCode, err)
			for _, app := range groupApps {
				s.logPayoutMailFailed(app.ID, senderCharacterID, recipientCharacterID, wrappedErr)
				failures[app.ID] = wrappedErr.Error()
			}
			continue
		}

		for _, app := range groupApps {
			mailLog := &model.SrpPayoutMailLog{
				ApplicationID:        app.ID,
				RecipientCharacterID: recipientCharacterID,
				SenderCharacterID:    senderCharacterID,
				MailID:               &mailID,
				Status:               model.SrpPayoutMailSuccess,
			}
			if err := s.repo.CreatePayoutMailLog(mailLog); err != nil {
				// 邮件已发送成功，但成功日志写入失败，这里仅记录告警，不将其视为可重试的邮件发送失败，
				// 以避免后续重试时重复发送同一封邮件。
				global.Logger.Warn("记录 SRP 发放邮件成功日志失败",
					zap.Uint("application_id", app.ID),
					zap.Int64("mail_id", mailID),
					zap.Error(err),
				)
			}
		}
	}

	return failures
}

// ListPayoutMailLogs 查询发放邮件日志
func (s *SrpService) ListPayoutMailLogs(page, pageSize int, filter repository.SrpPayoutMailLogFilter) ([]model.SrpPayoutMailLog, int64, error) {
	return s.repo.ListPayoutMailLogs(page, pageSize, filter)
}

// RetryPayoutMail 重试发放邮件（仅失败记录）
func (s *SrpService) RetryPayoutMail(appID uint) error {
	app, err := s.repo.GetApplicationByID(appID)
	if err != nil {
		return errors.New("申请不存在")
	}
	if app.PayoutStatus != model.SrpPayoutPaid {
		return errors.New("申请未发放，无法重试邮件")
	}
	if app.PaidBy == nil || *app.PaidBy == 0 {
		return errors.New("申请缺少原发放人信息，无法重试邮件")
	}
	if s.repo.HasPayoutMailSuccess(appID) {
		return errors.New("该申请已存在成功邮件记录，无需重试")
	}

	return s.sendPayoutMail(context.Background(), *app.PaidBy, app)
}

type eveMailRecipient struct {
	RecipientID   int64  `json:"recipient_id"`
	RecipientType string `json:"recipient_type"`
}

type eveMailSendRequest struct {
	Subject    string             `json:"subject"`
	Body       string             `json:"body"`
	Recipients []eveMailRecipient `json:"recipients"`
}

func (s *SrpService) sendPayoutMail(ctx context.Context, payerID uint, app *model.SrpApplication) error {
	if s.repo.HasPayoutMailSuccess(app.ID) {
		return nil
	}

	senderCharacterID, err := s.resolveUserPrimaryCharacterID(payerID)
	if err != nil {
		s.logPayoutMailFailed(app.ID, 0, 0, err)
		return err
	}

	recipientCharacterID, err := s.resolveUserPrimaryCharacterID(app.UserID)
	if err != nil {
		s.logPayoutMailFailed(app.ID, senderCharacterID, 0, err)
		return err
	}

	senderChar, err := s.charRepo.GetByCharacterID(senderCharacterID)
	if err != nil {
		s.logPayoutMailFailed(app.ID, senderCharacterID, recipientCharacterID, fmt.Errorf("发信角色不存在: %w", err))
		return err
	}

	if !hasScope(senderChar.Scopes, esiMailSendScope) {
		err = fmt.Errorf("发信角色未授权 scope: %s", esiMailSendScope)
		s.logPayoutMailFailed(app.ID, senderCharacterID, recipientCharacterID, err)
		return err
	}

	token, err := s.ssoSvc.GetValidToken(ctx, senderCharacterID)
	if err != nil {
		s.logPayoutMailFailed(app.ID, senderCharacterID, recipientCharacterID, fmt.Errorf("获取发信 token 失败: %w", err))
		return err
	}

	subject := fmt.Sprintf("[SRP 发放通知 / SRP Payout Notice] %s - %s", app.CharacterName, formatISKCompact(app.FinalAmount))
	body := buildSinglePayoutMailBody(app, s.resolveFleetTitle(app))

	mailID, statusCode, err := sendEveMail(ctx, senderCharacterID, token, eveMailSendRequest{
		Subject: subject,
		Body:    body,
		Recipients: []eveMailRecipient{
			{RecipientID: recipientCharacterID, RecipientType: "character"},
		},
	})
	if err != nil {
		wrappedErr := fmt.Errorf("ESI 邮件发送失败(status=%d): %w", statusCode, err)
		s.logPayoutMailFailed(app.ID, senderCharacterID, recipientCharacterID, wrappedErr)
		return wrappedErr
	}

	mailLog := &model.SrpPayoutMailLog{
		ApplicationID:        app.ID,
		RecipientCharacterID: recipientCharacterID,
		SenderCharacterID:    senderCharacterID,
		MailID:               &mailID,
		Status:               model.SrpPayoutMailSuccess,
	}

	const maxPayoutMailLogRetries = 3
	var lastErr error
	for attempt := 1; attempt <= maxPayoutMailLogRetries; attempt++ {
		if err := s.repo.CreatePayoutMailLog(mailLog); err != nil {
			lastErr = err
			global.Logger.Warn("记录 SRP 发放邮件成功日志失败",
				zap.Uint("application_id", app.ID),
				zap.Int64("mail_id", mailID),
				zap.Int("attempt", attempt),
				zap.Error(err),
			)
			if attempt < maxPayoutMailLogRetries {
				time.Sleep(100 * time.Millisecond)
			}
			continue
		}
		lastErr = nil
		break
	}

	if lastErr != nil {
		wrappedErr := fmt.Errorf("记录 SRP 发放邮件成功日志失败(重试 %d 次仍失败)", maxPayoutMailLogRetries)
		global.Logger.Error("SRP 发放邮件成功但日志落库最终失败",
			zap.Uint("application_id", app.ID),
			zap.Int64("mail_id", mailID),
			zap.Error(lastErr),
		)
		return wrappedErr
	}

	return nil
}

func (s *SrpService) resolveUserPrimaryCharacterID(userID uint) (int64, error) {
	user, err := s.userRepo.GetByID(userID)
	if err != nil {
		return 0, fmt.Errorf("用户不存在(user_id=%d): %w", userID, err)
	}
	if user.PrimaryCharacterID == 0 {
		return 0, fmt.Errorf("用户主角色未设置(user_id=%d)", userID)
	}
	return user.PrimaryCharacterID, nil
}

func (s *SrpService) resolveFleetTitle(app *model.SrpApplication) string {
	if app == nil || app.FleetID == nil || *app.FleetID == "" {
		return ""
	}
	fleet, err := s.fleetRepo.GetByID(*app.FleetID)
	if err != nil || fleet == nil {
		return ""
	}
	return fleet.Title
}

func (s *SrpService) logPayoutMailFailed(applicationID uint, senderCharacterID, recipientCharacterID int64, err error) {
	// 记录完整错误到结构化日志，便于排查
	global.Logger.Error("SRP 发放邮件失败",
		zap.Uint("application_id", applicationID),
		zap.Int64("sender_character_id", senderCharacterID),
		zap.Int64("recipient_character_id", recipientCharacterID),
		zap.Error(err),
	)

	// ErrorMessage 字段限制为 1024 字符，这里对 err.Error() 做长度截断，避免插入失败
	const maxErrorMessageLen = 1024
	errMsg := err.Error()
	if len(errMsg) > maxErrorMessageLen {
		errMsg = errMsg[:maxErrorMessageLen]
	}

	mailLog := &model.SrpPayoutMailLog{
		ApplicationID:        applicationID,
		RecipientCharacterID: recipientCharacterID,
		SenderCharacterID:    senderCharacterID,
		Status:               model.SrpPayoutMailFailed,
		ErrorMessage:         errMsg,
	}
	if createErr := s.repo.CreatePayoutMailLog(mailLog); createErr != nil {
		global.Logger.Warn("记录 SRP 发放邮件失败日志失败",
			zap.Uint("application_id", applicationID),
			zap.Error(createErr),
		)
	}
}

func formatISKCompact(amount float64) string {
	abs := math.Abs(amount)
	sign := ""
	if amount < 0 {
		sign = "-"
	}

	switch {
	case abs >= 1_000_000_000:
		return fmt.Sprintf("%s%.2fB ISK", sign, abs/1_000_000_000)
	case abs >= 1_000_000:
		return fmt.Sprintf("%s%.2fM ISK", sign, abs/1_000_000)
	case abs >= 1_000:
		return fmt.Sprintf("%s%.2fK ISK", sign, abs/1_000)
	default:
		return fmt.Sprintf("%s%.2f ISK", sign, abs)
	}
}

func buildSinglePayoutMailBody(app *model.SrpApplication, fleetTitle string) string {
	shipDisplay := app.ShipName
	if shipDisplay == "" {
		shipDisplay = fmt.Sprintf("Type %d", app.ShipTypeID)
	}
	amountDisplay := formatISKCompact(app.FinalAmount)
	zkillURL := fmt.Sprintf("https://zkillboard.com/kill/%d/", app.KillmailID)

	var bodyBuilder strings.Builder
	bodyBuilder.WriteString(fmt.Sprintf("%s 你好，\n\n", app.CharacterName))
	bodyBuilder.WriteString("你的 SRP 补损已发放完成，详情如下：\n")
	if fleetTitle != "" {
		bodyBuilder.WriteString(fmt.Sprintf("关联舰队：%s\n", fleetTitle))
	}
	bodyBuilder.WriteString(fmt.Sprintf("损失时间：%s\n", app.KillmailTime.Format("2006-01-02 15:04:05")))
	bodyBuilder.WriteString(fmt.Sprintf("损失舰船：%s\n", shipDisplay))
	bodyBuilder.WriteString(fmt.Sprintf("发放金额：%s\n", amountDisplay))
	bodyBuilder.WriteString(fmt.Sprintf("zKillboard：<url=%s>查看击毁报告</url>\n\n", zkillURL))

	bodyBuilder.WriteString("Hello ")
	bodyBuilder.WriteString(app.CharacterName)
	bodyBuilder.WriteString(",\n\n")
	bodyBuilder.WriteString("Your SRP payout has been completed. Details are as follows:\n")
	if fleetTitle != "" {
		bodyBuilder.WriteString(fmt.Sprintf("Linked Fleet: %s\n", fleetTitle))
	}
	bodyBuilder.WriteString(fmt.Sprintf("Loss Time: %s\n", app.KillmailTime.Format("2006-01-02 15:04:05")))
	bodyBuilder.WriteString(fmt.Sprintf("Ship Lost: %s\n", shipDisplay))
	bodyBuilder.WriteString(fmt.Sprintf("Payout Amount: %s\n", amountDisplay))
	bodyBuilder.WriteString(fmt.Sprintf("zKillboard: <url=%s>View Killmail</url>\n\n", zkillURL))

	bodyBuilder.WriteString("────────────────────────\n")
	bodyBuilder.WriteString("该打的仗一场不少，该补的损一分不差。\n")
	bodyBuilder.WriteString("No battle is missed, no reimbursement is short.\n\n")
	bodyBuilder.WriteString("此邮件由 FUXI 后勤署（军团管理系统）自动发放，请勿回复。\n")
	bodyBuilder.WriteString("This mail is automatically issued by FUXI Logistics Office (Corporation Management System). Please do not reply.")

	return bodyBuilder.String()
}

// buildBatchPayoutMailContent 为批量发放生成聚合邮件主题和内容
// apps: 同一个收件人（recipient_character_id）的所有申请
func buildBatchPayoutMailContent(apps []*model.SrpApplication, fleetTitles map[uint]string) (string, string) {
	if len(apps) == 0 {
		return "", ""
	}

	totalISK := 0.0
	for _, app := range apps {
		totalISK += app.FinalAmount
	}

	recipientName := apps[0].CharacterName
	subject := fmt.Sprintf("[SRP 发放通知 / SRP Payout Notice] %s - %s", recipientName, formatISKCompact(totalISK))

	var bodyBuilder strings.Builder
	bodyBuilder.WriteString(fmt.Sprintf("%s 你好，\n\n以下 %d 条 SRP 补损申请已完成批量发放，合计 %s，详情如下：\n\n", recipientName, len(apps), formatISKCompact(totalISK)))
	bodyBuilder.WriteString(fmt.Sprintf("Hello %s,\n\n", recipientName))
	bodyBuilder.WriteString(fmt.Sprintf("Your %d SRP payouts have been completed. Total: %s. Details are as follows:\n\n", len(apps), formatISKCompact(totalISK)))

	for i, app := range apps {
		shipDisplay := app.ShipName
		if shipDisplay == "" {
			shipDisplay = fmt.Sprintf("Type %d", app.ShipTypeID)
		}
		zkillURL := fmt.Sprintf("https://zkillboard.com/kill/%d/", app.KillmailID)
		fleetTitle := ""
		if fleetTitles != nil {
			fleetTitle = fleetTitles[app.ID]
		}
		bodyBuilder.WriteString(fmt.Sprintf("────────────────────────\n"))
		bodyBuilder.WriteString(fmt.Sprintf("第 %d 条\n", i+1))
		if fleetTitle != "" {
			bodyBuilder.WriteString(fmt.Sprintf("  关联舰队：%s\n", fleetTitle))
		}
		bodyBuilder.WriteString(fmt.Sprintf("  zKillboard：<url=%s>查看击毁报告</url>\n", zkillURL))
		bodyBuilder.WriteString(fmt.Sprintf("  损失时间：%s\n", app.KillmailTime.Format("2006-01-02 15:04:05")))
		bodyBuilder.WriteString(fmt.Sprintf("  损失舰船：%s\n", shipDisplay))
		bodyBuilder.WriteString(fmt.Sprintf("  发放金额：%s\n", formatISKCompact(app.FinalAmount)))

		bodyBuilder.WriteString(fmt.Sprintf("  Item %d\n", i+1))
		if fleetTitle != "" {
			bodyBuilder.WriteString(fmt.Sprintf("  Linked Fleet: %s\n", fleetTitle))
		}
		bodyBuilder.WriteString(fmt.Sprintf("  zKillboard: <url=%s>View Killmail</url>\n", zkillURL))
		bodyBuilder.WriteString(fmt.Sprintf("  Loss Time: %s\n", app.KillmailTime.Format("2006-01-02 15:04:05")))
		bodyBuilder.WriteString(fmt.Sprintf("  Ship Lost: %s\n", shipDisplay))
		bodyBuilder.WriteString(fmt.Sprintf("  Payout Amount: %s\n", formatISKCompact(app.FinalAmount)))
	}

	bodyBuilder.WriteString("────────────────────────\n\n")
	bodyBuilder.WriteString("该打的仗一场不少，该补的损一分不差。\n")
	bodyBuilder.WriteString("No battle is missed, no reimbursement is short.\n\n")
	bodyBuilder.WriteString("此邮件由 FUXI 后勤署（军团管理系统）自动发放，请勿回复。\n")
	bodyBuilder.WriteString("This mail is automatically issued by FUXI Logistics Office (Corporation Management System). Please do not reply.")
	return subject, bodyBuilder.String()
}

func hasScope(scopes, target string) bool {
	for _, s := range strings.Fields(scopes) {
		if s == target {
			return true
		}
	}
	return false
}

func sendEveMail(ctx context.Context, senderCharacterID int64, accessToken string, req eveMailSendRequest) (int64, int, error) {
	bodyBytes, err := json.Marshal(req)
	if err != nil {
		return 0, 0, fmt.Errorf("序列化邮件请求失败: %w", err)
	}

	url := fmt.Sprintf("https://esi.evetech.net/latest/characters/%d/mail/", senderCharacterID)
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(bodyBytes))
	if err != nil {
		return 0, 0, fmt.Errorf("构建请求失败: %w", err)
	}
	httpReq.Header.Set("Authorization", "Bearer "+accessToken)
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(httpReq)
	if err != nil {
		return 0, 0, err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		respBody, _ := io.ReadAll(resp.Body)
		return 0, resp.StatusCode, fmt.Errorf("%s", strings.TrimSpace(string(respBody)))
	}

	var mailID int64
	if err := json.NewDecoder(resp.Body).Decode(&mailID); err != nil {
		return 0, resp.StatusCode, fmt.Errorf("解析 ESI 响应失败: %w", err)
	}

	return mailID, resp.StatusCode, nil
}

// ─────────────────────────────────────────────
//  ESI: Open Information Window
// ─────────────────────────────────────────────

// OpenInfoWindowRequest 打开角色信息窗口请求
type OpenInfoWindowRequest struct {
	CharacterID int64 `json:"character_id" binding:"required"` // 操作者角色 ID（用于获取 token）
	TargetID    int64 `json:"target_id"    binding:"required"` // 要打开信息窗口的目标 ID
}

// OpenInfoWindow 通过 ESI 在客户端打开角色信息窗口
// POST /ui/openwindow/information?target_id=xxx
// 需要 scope: esi-ui.open_window.v1
func (s *SrpService) OpenInfoWindow(userID uint, req *OpenInfoWindowRequest) error {
	// 1. 验证角色属于当前用户
	char, err := s.charRepo.GetByCharacterID(req.CharacterID)
	if err != nil || char.UserID != userID {
		return errors.New("角色不属于当前用户或不存在")
	}

	// 2. 获取有效 token
	ctx := context.Background()
	token, err := s.ssoSvc.GetValidToken(ctx, req.CharacterID)
	if err != nil {
		return fmt.Errorf("获取 token 失败: %w", err)
	}

	// 3. 调用 ESI Open Information Window
	url := fmt.Sprintf("https://esi.evetech.net/ui/openwindow/information/?target_id=%d", req.TargetID)
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, nil)
	if err != nil {
		return fmt.Errorf("构建请求失败: %w", err)
	}
	httpReq.Header.Set("Authorization", "Bearer "+token)
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(httpReq)
	if err != nil {
		return fmt.Errorf("调用 ESI Open Window 失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("ESI error %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

// ─────────────────────────────────────────────
//  快捷申请：获取符合舰队的我的 KM 列表
// ─────────────────────────────────────────────

// FleetKillmailItem KM 列表条目（给前端下拉用）
type FleetKillmailItem struct {
	KillmailID    int64     `json:"killmail_id"`
	KillmailTime  time.Time `json:"killmail_time"`
	ShipTypeID    int64     `json:"ship_type_id"`
	SolarSystemID int64     `json:"solar_system_id"`
	CharacterID   int64     `json:"character_id"`
	VictimName    string    `json:"victim_name"`
}

// GetMyKillmails 获取当前用户所有角色作为受害者的 KM 列表（不限舰队，最近 200 条）
// 若 characterID > 0，则只返回指定角色的 KM（需属于当前用户）
func (s *SrpService) GetMyKillmails(userID uint, characterID int64) ([]FleetKillmailItem, error) {
	chars, err := s.charRepo.ListByUserID(userID)
	if err != nil || len(chars) == 0 {
		return []FleetKillmailItem{}, nil
	}
	charNameMap := make(map[int64]string)
	var charIDs []int64
	if characterID > 0 {
		for _, c := range chars {
			if c.CharacterID == characterID {
				charIDs = []int64{characterID}
				charNameMap[characterID] = c.CharacterName
				break
			}
		}
		if len(charIDs) == 0 {
			return []FleetKillmailItem{}, nil
		}
	} else {
		for _, c := range chars {
			charIDs = append(charIDs, c.CharacterID)
			charNameMap[c.CharacterID] = c.CharacterName
		}
	}

	var ckmList []model.EveCharacterKillmail
	if err := global.DB.Where("character_id IN ?", charIDs).Find(&ckmList).Error; err != nil {
		return nil, err
	}
	if len(ckmList) == 0 {
		return []FleetKillmailItem{}, nil
	}

	kmIDs := make([]int64, 0, len(ckmList))
	kmCharMap := make(map[int64]int64) // killmail_id -> character_id
	for _, ckm := range ckmList {
		kmIDs = append(kmIDs, ckm.KillmailID)
		kmCharMap[ckm.KillmailID] = ckm.CharacterID
	}

	// 只查询最近 30 天的 KM
	since := time.Now().AddDate(0, 0, -30)

	var kms []model.EveKillmailList
	if err := global.DB.Where("kill_mail_id IN ? AND kill_mail_time >= ?", kmIDs, since).
		Order("kill_mail_time DESC").
		Limit(200).
		Find(&kms).Error; err != nil {
		return nil, err
	}

	charIDSet := make(map[int64]bool)
	for _, id := range charIDs {
		charIDSet[id] = true
	}

	result := make([]FleetKillmailItem, 0, len(kms))
	for _, km := range kms {
		// 只返回受害者是当前用户角色的 KM
		if !charIDSet[km.CharacterID] {
			continue
		}
		result = append(result, FleetKillmailItem{
			KillmailID:    km.KillmailID,
			KillmailTime:  km.KillmailTime,
			ShipTypeID:    km.ShipTypeID,
			SolarSystemID: km.SolarSystemID,
			CharacterID:   km.CharacterID,
			VictimName:    charNameMap[km.CharacterID],
		})
	}
	return result, nil
}

// GetFleetKillmails 获取符合舰队时间范围和成员资格的当前用户 KM 列表
func (s *SrpService) GetFleetKillmails(userID uint, fleetID string) ([]FleetKillmailItem, error) {
	// 1. 获取舰队信息
	fleet, err := s.fleetRepo.GetByID(fleetID)
	if err != nil {
		return nil, errors.New("舰队不存在")
	}

	// 2. 获取当前用户绑定的角色
	chars, err := s.charRepo.ListByUserID(userID)
	if err != nil || len(chars) == 0 {
		return nil, errors.New("当前用户未绑定角色")
	}

	// 3. 筛选出参与过该舰队的角色 ID
	members, err := s.fleetRepo.ListMembers(fleetID)
	if err != nil {
		return nil, err
	}
	memberSet := make(map[int64]bool)
	for _, m := range members {
		memberSet[m.CharacterID] = true
	}
	var validCharIDs []int64
	charNameMap := make(map[int64]string)
	for _, c := range chars {
		if memberSet[c.CharacterID] {
			validCharIDs = append(validCharIDs, c.CharacterID)
			charNameMap[c.CharacterID] = c.CharacterName
		}
	}
	if len(validCharIDs) == 0 {
		return []FleetKillmailItem{}, nil
	}

	// 4. 查询这些角色在舰队时间段内的 KM
	var ckmList []model.EveCharacterKillmail
	if err := global.DB.Where("character_id IN ?", validCharIDs).Find(&ckmList).Error; err != nil {
		return nil, err
	}
	if len(ckmList) == 0 {
		return []FleetKillmailItem{}, nil
	}
	kmIDSet := make(map[int64]int64) // killmail_id -> character_id
	for _, ckm := range ckmList {
		kmIDSet[ckm.KillmailID] = ckm.CharacterID
	}
	kmIDs := make([]int64, 0, len(kmIDSet))
	for kid := range kmIDSet {
		kmIDs = append(kmIDs, kid)
	}

	var kms []model.EveKillmailList
	if err := global.DB.Where("kill_mail_id IN ? AND kill_mail_time >= ? AND kill_mail_time <= ?",
		kmIDs, fleet.StartAt, fleet.EndAt).Find(&kms).Error; err != nil {
		return nil, err
	}

	// 5. 只返回受害角色是用户自己角色的 KM
	result := make([]FleetKillmailItem, 0, len(kms))
	for _, km := range kms {
		if !memberSet[km.CharacterID] {
			continue
		}
		name := charNameMap[km.CharacterID]
		result = append(result, FleetKillmailItem{
			KillmailID:    km.KillmailID,
			KillmailTime:  km.KillmailTime,
			ShipTypeID:    km.ShipTypeID,
			SolarSystemID: km.SolarSystemID,
			CharacterID:   km.CharacterID,
			VictimName:    name,
		})
	}
	return result, nil
}

// ─────────────────────────────────────────────
//  KM 装配详情
// ─────────────────────────────────────────────

// KillmailDetailRequest 请求参数
type KillmailDetailRequest struct {
	KillmailID int64  `json:"killmail_id" binding:"required"`
	Language   string `json:"language"` // "zh" / "en"
}

// KillmailSlotItem 单个槽位中合并后的物品
type KillmailSlotItem struct {
	ItemID   int    `json:"item_id"`
	ItemName string `json:"item_name"`
	Quantity int64  `json:"quantity"`
	Dropped  bool   `json:"dropped"` // true=掉落, false=摧毁
}

// KillmailSlotGroup 按槽位分组
type KillmailSlotGroup struct {
	FlagID   int                `json:"flag_id"`
	FlagName string             `json:"flag_name"`
	FlagText string             `json:"flag_text"`
	OrderID  int                `json:"order_id"`
	Items    []KillmailSlotItem `json:"items"`
}

// KillmailDetailResponse KM 装配详情响应
type KillmailDetailResponse struct {
	KillmailID    int64               `json:"killmail_id"`
	KillmailTime  time.Time           `json:"killmail_time"`
	ShipTypeID    int64               `json:"ship_type_id"`
	ShipName      string              `json:"ship_name"`
	SolarSystemID int64               `json:"solar_system_id"`
	SystemName    string              `json:"system_name"`
	CharacterID   int64               `json:"character_id"`
	CharacterName string              `json:"character_name"`
	JaniceAmount  *float64            `json:"janice_amount"`
	Slots         []KillmailSlotGroup `json:"slots"`
}

// slotCategory 将 HiSlot0, HiSlot1 等 flagName 归类为 "HiSlot"
func slotCategory(flagName string) string {
	return strings.TrimRight(flagName, "0123456789")
}

// slotCategoryNames 槽位类别的中英文显示名
var slotCategoryNames = map[string]map[string]string{
	"HiSlot":              {"zh": "高槽", "en": "High Slots"},
	"MedSlot":             {"zh": "中槽", "en": "Medium Slots"},
	"LoSlot":              {"zh": "低槽", "en": "Low Slots"},
	"RigSlot":             {"zh": "改装件", "en": "Rig Slots"},
	"SubSystemSlot":       {"zh": "子系统", "en": "Subsystem Slots"},
	"DroneBay":            {"zh": "无人机舱", "en": "Drone Bay"},
	"FighterBay":          {"zh": "战斗机机库", "en": "Fighter Bay"},
	"Cargo":               {"zh": "货柜舱", "en": "Cargo"},
	"FleetHangar":         {"zh": "舰队机库", "en": "Fleet Hangar"},
	"Implant":             {"zh": "植入体", "en": "Implants"},
	"SpecializedFuelBay":  {"zh": "燃料舱", "en": "Fuel Bay"},
	"SpecializedOreHold":  {"zh": "矿石舱", "en": "Ore Hold"},
	"SpecializedAmmoHold": {"zh": "弹药舱", "en": "Ammo Hold"},
}

// GetKillmailDetail 查询 KM 装配详情
func (s *SrpService) GetKillmailDetail(req *KillmailDetailRequest) (*KillmailDetailResponse, error) {
	lang := req.Language
	if lang == "" {
		lang = "zh"
	}

	// 1. 查询 KM 主记录
	var km model.EveKillmailList
	if err := global.DB.Where("kill_mail_id = ?", req.KillmailID).First(&km).Error; err != nil {
		return nil, errors.New("KM 不存在")
	}

	// 2. 查询 KM 所有物品
	var items []model.EveKillmailItem
	if err := global.DB.Where("kill_mail_id = ?", req.KillmailID).Find(&items).Error; err != nil {
		return nil, err
	}

	// 3. 收集所有 flagID 查 invFlags
	flagIDSet := make(map[int]bool)
	for _, it := range items {
		flagIDSet[it.Flag] = true
	}
	flagIDs := make([]int, 0, len(flagIDSet))
	for fid := range flagIDSet {
		flagIDs = append(flagIDs, fid)
	}
	flags, err := s.sdeRepo.GetFlags(flagIDs)
	if err != nil {
		return nil, err
	}
	flagMap := make(map[int]repository.FlagInfo)
	for _, f := range flags {
		flagMap[f.FlagID] = f
	}

	// 4. 收集所有 typeID（物品 + 舰船），查翻译名
	typeIDSet := make(map[int]bool)
	typeIDSet[int(km.ShipTypeID)] = true
	for _, it := range items {
		typeIDSet[it.ItemID] = true
	}
	typeIDs := make([]int, 0, len(typeIDSet))
	for tid := range typeIDSet {
		typeIDs = append(typeIDs, tid)
	}
	nameMap, err := s.sdeRepo.GetNames(map[string][]int{"type": typeIDs}, lang)
	if err != nil {
		return nil, err
	}

	// 5. 查星系名
	sysNameMap, _ := s.sdeRepo.GetNames(map[string][]int{"solar_system": {int(km.SolarSystemID)}}, lang)

	// 6. 查角色名
	charName := ""
	var char model.EveCharacter
	if err := global.DB.Where("character_id = ?", km.CharacterID).First(&char).Error; err == nil {
		charName = char.CharacterName
	}

	// 7. 按 (槽位类别, item_id, dropped) 合并，同时按类别分组
	type mergeKey struct {
		Category string
		ItemID   int
		Dropped  bool
	}
	merged := make(map[mergeKey]*KillmailSlotItem)

	catMap := make(map[string]*KillmailSlotGroup)
	catOrder := make([]string, 0)

	for _, it := range items {
		dropped := it.DropType != nil && *it.DropType
		fi := flagMap[it.Flag]
		cat := slotCategory(fi.FlagName)

		// 确保类别组已创建
		if _, ok := catMap[cat]; !ok {
			displayName := fi.FlagText
			if names, exists := slotCategoryNames[cat]; exists {
				if n, ok := names[lang]; ok {
					displayName = n
				}
			}
			catMap[cat] = &KillmailSlotGroup{
				FlagID:   it.Flag,
				FlagName: cat,
				FlagText: displayName,
				OrderID:  fi.OrderID,
				Items:    []KillmailSlotItem{},
			}
			catOrder = append(catOrder, cat)
		} else if fi.OrderID < catMap[cat].OrderID {
			catMap[cat].OrderID = fi.OrderID
		}

		// 按 (category, item_id, dropped) 合并
		key := mergeKey{Category: cat, ItemID: it.ItemID, Dropped: dropped}
		if existing, ok := merged[key]; ok {
			existing.Quantity += it.ItemNum
		} else {
			itemName := nameMap[it.ItemID]
			if itemName == "" {
				itemName = "Unknown"
			}
			si := &KillmailSlotItem{
				ItemID:   it.ItemID,
				ItemName: itemName,
				Quantity: it.ItemNum,
				Dropped:  dropped,
			}
			merged[key] = si
			catMap[cat].Items = append(catMap[cat].Items, *si)
		}
	}

	// 回写合并后数量（指针合并后 slice 中是副本，需要同步）
	for cat, g := range catMap {
		for i := range g.Items {
			key := mergeKey{Category: cat, ItemID: g.Items[i].ItemID, Dropped: g.Items[i].Dropped}
			g.Items[i].Quantity = merged[key].Quantity
		}
	}

	// 按 orderID 排序
	slots := make([]KillmailSlotGroup, 0, len(catOrder))
	for _, cat := range catOrder {
		slots = append(slots, *catMap[cat])
	}
	for i := 1; i < len(slots); i++ {
		for j := i; j > 0 && slots[j].OrderID < slots[j-1].OrderID; j-- {
			slots[j], slots[j-1] = slots[j-1], slots[j]
		}
	}

	shipName := nameMap[int(km.ShipTypeID)]
	if shipName == "" {
		shipName = "Unknown"
	}
	sysName := sysNameMap[int(km.SolarSystemID)]

	return &KillmailDetailResponse{
		KillmailID:    km.KillmailID,
		KillmailTime:  km.KillmailTime,
		ShipTypeID:    km.ShipTypeID,
		ShipName:      shipName,
		SolarSystemID: km.SolarSystemID,
		SystemName:    sysName,
		CharacterID:   km.CharacterID,
		CharacterName: charName,
		JaniceAmount:  km.JaniceAmount,
		Slots:         slots,
	}, nil
}

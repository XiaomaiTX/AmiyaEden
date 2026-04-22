package service

import (
	"amiya-eden/global"
	"amiya-eden/internal/model"
	"amiya-eden/internal/repository"
	"errors"
	"fmt"
	"strings"
	"time"

	"go.uber.org/zap"
	"gorm.io/gorm"
)

// resolveEntryStatus is a pure function: given entry timing, whether a user was found,
// and when that user was created, return the new status.
func resolveEntryStatus(enteredAt time.Time, userCreatedAt time.Time, userFound bool, now time.Time, cooldownDays int) string {
	if userFound {
		if userCreatedAt.After(enteredAt) {
			return model.RecruitEntryStatusValid
		}
		return model.RecruitEntryStatusStalled
	}
	if now.Sub(enteredAt) > time.Duration(cooldownDays)*24*time.Hour {
		return model.RecruitEntryStatusStalled
	}
	return model.RecruitEntryStatusOngoing
}

// ─── Types ───────────────────────────────────────────────────────────────────

type RecruitEntryResult struct {
	ProcessedCount   int
	ValidCount       int
	StalledCount     int
	TotalCoinAwarded float64
}

type DirectReferralStatus struct {
	ShowCard       bool `json:"show_card"`
	NeedsProfileQQ bool `json:"needs_profile_qq"`
}

type DirectReferrerCandidate struct {
	UserID               uint   `json:"user_id"`
	Nickname             string `json:"nickname"`
	PrimaryCharacterID   int64  `json:"primary_character_id"`
	PrimaryCharacterName string `json:"primary_character_name"`
}

type recruitmentEntryUserRepository interface {
	GetByID(id uint) (*model.User, error)
	ListByQQ(qq string) ([]model.User, error)
	GetByIDForUpdateTx(tx *gorm.DB, id uint) (*model.User, error)
}

// ─── Service ─────────────────────────────────────────────────────────────────

type RecruitmentEntryService struct {
	repo        *repository.NewbroRecruitmentRepository
	userRepo    recruitmentEntryUserRepository
	roleRepo    *repository.RoleRepository
	charRepo    *repository.EveCharacterRepository
	walletSvc   *SysWalletService
	settingsSvc *NewbroSettingsService
}

func NewRecruitmentEntryService() *RecruitmentEntryService {
	return &RecruitmentEntryService{
		repo:        repository.NewNewbroRecruitmentRepository(),
		userRepo:    repository.NewUserRepository(),
		roleRepo:    repository.NewRoleRepository(),
		charRepo:    repository.NewEveCharacterRepository(),
		walletSvc:   NewSysWalletService(),
		settingsSvc: NewNewbroSettingsService(),
	}
}

const directReferralWindow = 7 * 24 * time.Hour

func buildRecruitRewardRefID(userID uint) string {
	return fmt.Sprintf("recruit_matched_user:%d", userID)
}

func (s *RecruitmentEntryService) getUniqueUserByQQ(qq string) (*model.User, error) {
	users, err := s.userRepo.ListByQQ(qq)
	if err != nil {
		return nil, err
	}
	if len(users) != 1 {
		return nil, errors.New("未找到符合条件的推荐人")
	}
	user := users[0]
	return &user, nil
}

func (s *RecruitmentEntryService) loadUserRoleCodes(user *model.User) ([]string, error) {
	roleCodes, err := s.roleRepo.GetUserRoleCodes(user.ID)
	if err != nil {
		return nil, err
	}
	return model.NormalizeRoleCodes(roleCodes, user.Role), nil
}

func (s *RecruitmentEntryService) canUseDirectReferral(user *model.User, roleCodes []string, now time.Time) bool {
	if user == nil {
		return false
	}
	if !model.HasNonGuestRole(roleCodes) {
		return false
	}
	return now.Sub(user.CreatedAt) <= directReferralWindow
}

func (s *RecruitmentEntryService) buildDirectReferrerCandidate(user *model.User) (*DirectReferrerCandidate, error) {
	primaryCharacterID := user.PrimaryCharacterID
	primaryCharacterName := user.Nickname
	if primaryCharacterID != 0 {
		if char, err := s.charRepo.GetByCharacterID(primaryCharacterID); err == nil && char != nil {
			primaryCharacterName = char.CharacterName
		}
	}

	return &DirectReferrerCandidate{
		UserID:               user.ID,
		Nickname:             user.Nickname,
		PrimaryCharacterID:   primaryCharacterID,
		PrimaryCharacterName: primaryCharacterName,
	}, nil
}

func validateDirectReferralStatus(status *DirectReferralStatus) error {
	if !status.ShowCard {
		return errors.New("当前不满足补录推荐人条件")
	}
	if status.NeedsProfileQQ {
		return errors.New("请先在联系方式中填写自己的 QQ 并保存")
	}
	return nil
}

func (s *RecruitmentEntryService) ensureDirectReferrerEligible(referrer *model.User, currentUserID uint, selfReferralMessage string) error {
	if referrer == nil {
		return errors.New("未找到符合条件的推荐人")
	}
	if referrer.ID == currentUserID {
		return errors.New(selfReferralMessage)
	}

	roleCodes, err := s.loadUserRoleCodes(referrer)
	if err != nil {
		return err
	}
	if !model.HasNonGuestRole(roleCodes) {
		return errors.New("未找到符合条件的推荐人")
	}
	return nil
}

func (s *RecruitmentEntryService) loadDirectReferrerByQQ(currentUserID uint, referrerQQ string) (*model.User, error) {
	referrer, err := s.getUniqueUserByQQ(referrerQQ)
	if err != nil {
		return nil, errors.New("未找到符合条件的推荐人")
	}
	if err := s.ensureDirectReferrerEligible(referrer, currentUserID, "不能将自己填写为推荐人"); err != nil {
		return nil, err
	}
	return referrer, nil
}

func (s *RecruitmentEntryService) loadDirectReferrerByID(currentUserID, referrerUserID uint) (*model.User, error) {
	referrer, err := s.userRepo.GetByID(referrerUserID)
	if err != nil {
		return nil, errors.New("未找到符合条件的推荐人")
	}
	if err := s.ensureDirectReferrerEligible(referrer, currentUserID, "未找到符合条件的推荐人"); err != nil {
		return nil, err
	}
	return referrer, nil
}

func (s *RecruitmentEntryService) GetDirectReferralStatus(userID uint, now time.Time) (*DirectReferralStatus, error) {
	user, err := s.userRepo.GetByID(userID)
	if err != nil {
		return nil, err
	}

	roleCodes, err := s.loadUserRoleCodes(user)
	if err != nil {
		return nil, err
	}

	status := &DirectReferralStatus{}
	if !s.canUseDirectReferral(user, roleCodes, now) {
		return status, nil
	}

	status.ShowCard = true
	status.NeedsProfileQQ = strings.TrimSpace(user.QQ) == ""

	alreadyRewarded, err := s.repo.HasEntryWithWalletRefID(buildRecruitRewardRefID(userID))
	if err != nil {
		return nil, err
	}
	if alreadyRewarded {
		status.ShowCard = false
		return status, nil
	}

	return status, nil
}

func (s *RecruitmentEntryService) LookupDirectReferrer(currentUserID uint, referrerQQ string, now time.Time) (*DirectReferrerCandidate, error) {
	status, err := s.GetDirectReferralStatus(currentUserID, now)
	if err != nil {
		return nil, err
	}
	if err := validateDirectReferralStatus(status); err != nil {
		return nil, err
	}

	referrerQQ = strings.TrimSpace(referrerQQ)
	if err := validateQQ(referrerQQ); err != nil {
		return nil, err
	}

	referrer, err := s.loadDirectReferrerByQQ(currentUserID, referrerQQ)
	if err != nil {
		return nil, err
	}

	candidate, err := s.buildDirectReferrerCandidate(referrer)
	if err != nil {
		return nil, err
	}
	return candidate, nil
}

func (s *RecruitmentEntryService) ConfirmDirectReferral(currentUserID, referrerUserID uint, now time.Time) (*DirectReferrerCandidate, error) {
	status, err := s.GetDirectReferralStatus(currentUserID, now)
	if err != nil {
		return nil, err
	}
	if err := validateDirectReferralStatus(status); err != nil {
		return nil, err
	}

	referrer, err := s.loadDirectReferrerByID(currentUserID, referrerUserID)
	if err != nil {
		return nil, err
	}

	confirmed, err := s.buildDirectReferrerCandidate(referrer)
	if err != nil {
		return nil, err
	}

	rewardAmount := s.settingsSvc.GetSettings().RecruitRewardAmount
	rewardRefID := buildRecruitRewardRefID(currentUserID)

	if err := global.DB.Transaction(func(tx *gorm.DB) error {
		return s.confirmDirectReferralTx(tx, currentUserID, referrerUserID, rewardAmount, rewardRefID, now)
	}); err != nil {
		return nil, err
	}

	return confirmed, nil
}

func (s *RecruitmentEntryService) confirmDirectReferralTx(tx *gorm.DB, currentUserID, referrerUserID uint, rewardAmount float64, rewardRefID string, now time.Time) error {
	lockedCurrentUser, lockedReferrer, err := s.lockDirectReferralUsers(tx, currentUserID, referrerUserID)
	if err != nil {
		return err
	}
	if lockedReferrer.ID == currentUserID {
		return errors.New("未找到符合条件的推荐人")
	}

	alreadyRewarded, err := s.repo.HasEntryWithWalletRefIDTx(tx, rewardRefID)
	if err != nil {
		return err
	}
	if alreadyRewarded {
		return errors.New("当前账号已经存在招募记录")
	}

	currentUserQQ := strings.TrimSpace(lockedCurrentUser.QQ)
	if currentUserQQ == "" {
		return errors.New("请先在联系方式中填写自己的 QQ 并保存")
	}

	return s.createDirectReferralRewardTx(
		tx,
		currentUserID,
		referrerUserID,
		lockedCurrentUser.CreatedAt,
		currentUserQQ,
		rewardAmount,
		rewardRefID,
		now,
	)
}

func (s *RecruitmentEntryService) lockDirectReferralUsers(tx *gorm.DB, currentUserID, referrerUserID uint) (*model.User, *model.User, error) {
	lockedCurrentUser, err := s.userRepo.GetByIDForUpdateTx(tx, currentUserID)
	if err != nil {
		return nil, nil, err
	}
	lockedReferrer, err := s.userRepo.GetByIDForUpdateTx(tx, referrerUserID)
	if err != nil {
		return nil, nil, err
	}
	return lockedCurrentUser, lockedReferrer, nil
}

func (s *RecruitmentEntryService) createDirectReferralRewardTx(tx *gorm.DB, currentUserID, referrerUserID uint, currentUserCreatedAt time.Time, currentUserQQ string, rewardAmount float64, rewardRefID string, now time.Time) error {
	recruitment := &model.NewbroRecruitment{
		UserID:      referrerUserID,
		Source:      model.RecruitmentSourceDirectReferral,
		GeneratedAt: now,
		Code:        fmt.Sprintf("~%d", currentUserID),
	}
	if err := s.repo.CreateTx(tx, recruitment); err != nil {
		return err
	}
	recruitment.Code = base62Encode(recruitment.ID)
	if err := s.repo.UpdateCodeTx(tx, recruitment.ID, recruitment.Code); err != nil {
		return err
	}

	rewardedAt := now
	walletRefID := rewardRefID
	entry := &model.NewbroRecruitmentEntry{
		RecruitmentID: recruitment.ID,
		QQ:            currentUserQQ,
		EnteredAt:     currentUserCreatedAt,
		Source:        model.RecruitEntrySourceDirectReferral,
		Status:        model.RecruitEntryStatusValid,
		MatchedUserID: currentUserID,
		RewardedAt:    &rewardedAt,
		WalletRefID:   &walletRefID,
	}
	if err := s.repo.CreateEntryTx(tx, entry); err != nil {
		return err
	}

	return s.walletSvc.ApplyWalletDeltaTx(
		tx,
		referrerUserID,
		rewardAmount,
		buildDirectReferralRewardReason(currentUserQQ),
		model.WalletRefRecruitReward,
		rewardRefID,
	)
}

func buildDirectReferralRewardReason(currentUserQQ string) string {
	if currentUserQQ == "" {
		return "招募链接奖励（直接推荐）"
	}
	return fmt.Sprintf("招募链接奖励（直接推荐 QQ %s）", currentUserQQ)
}

// validateQQ returns an error if the QQ string is not 5-20 digits.
func validateQQ(qq string) error {
	qq = strings.TrimSpace(qq)
	if len(qq) < 5 || len(qq) > 20 {
		return errors.New("QQ 号码长度必须在 5 到 20 位之间")
	}
	for _, c := range qq {
		if c < '0' || c > '9' {
			return errors.New("QQ 号码只能包含数字")
		}
	}
	return nil
}

// SubmitQQ records an entry for the given recruitment code and QQ number.
// Returns the configured QQ group invitation URL on success.
func (s *RecruitmentEntryService) SubmitQQ(code, qq string, now time.Time) (string, error) {
	qq = strings.TrimSpace(qq)
	if err := validateQQ(qq); err != nil {
		return "", err
	}

	settings := s.settingsSvc.GetSettings()
	if settings.RecruitQQURL == "" {
		return "", errors.New("QQ 群邀请地址尚未配置，请联系管理员")
	}

	if err := global.DB.Transaction(func(tx *gorm.DB) error {
		rec, err := s.repo.GetByCodeForUpdateTx(tx, code)
		if err != nil {
			return err
		}
		if rec == nil {
			return errors.New("招募链接不存在")
		}
		if model.NormalizeRecruitmentSource(rec.Source) == model.RecruitmentSourceDirectReferral {
			return errors.New("该记录不支持公开提交")
		}

		existing, err := s.repo.GetEntryByRecruitmentIDAndQQTx(tx, rec.ID, qq)
		if err != nil {
			return err
		}
		if existing != nil {
			return nil
		}

		entry := &model.NewbroRecruitmentEntry{
			RecruitmentID: rec.ID,
			QQ:            qq,
			EnteredAt:     now,
			Source:        model.RecruitEntrySourceLink,
			Status:        model.RecruitEntryStatusOngoing,
		}
		if err := s.repo.CreateEntryTx(tx, entry); err != nil {
			// Another concurrent submit may have inserted the same (recruitment_id, qq) pair.
			duplicate, lookupErr := s.repo.GetEntryByRecruitmentIDAndQQTx(tx, rec.ID, qq)
			if lookupErr == nil && duplicate != nil {
				return nil
			}
			return err
		}
		return nil
	}); err != nil {
		return "", err
	}

	return settings.RecruitQQURL, nil
}

func mergeRecruitEntryResult(result *RecruitEntryResult, delta RecruitEntryResult) {
	result.ProcessedCount += delta.ProcessedCount
	result.ValidCount += delta.ValidCount
	result.StalledCount += delta.StalledCount
	result.TotalCoinAwarded += delta.TotalCoinAwarded
}

func (s *RecruitmentEntryService) lookupMatchedUserForEntry(entry model.NewbroRecruitmentEntry) (*model.User, bool, bool) {
	users, err := s.userRepo.ListByQQ(entry.QQ)
	if err != nil {
		global.Logger.Error("按 QQ 查询用户失败",
			zap.Uint("entry_id", entry.ID), zap.String("qq", entry.QQ), zap.Error(err))
		return nil, false, true
	}
	if len(users) > 1 {
		global.Logger.Error("按 QQ 匹配到多个用户，跳过本次招募归因",
			zap.Uint("entry_id", entry.ID), zap.String("qq", entry.QQ), zap.Int("match_count", len(users)))
		return nil, false, true
	}
	if len(users) == 0 {
		return nil, false, false
	}
	user := users[0]
	return &user, true, false
}

func (s *RecruitmentEntryService) markEntryStalledResult(entryID uint, logMessage string) RecruitEntryResult {
	if err := s.repo.MarkEntryStalled(entryID); err != nil {
		global.Logger.Error(logMessage, zap.Uint("entry_id", entryID), zap.Error(err))
		return RecruitEntryResult{}
	}
	return RecruitEntryResult{ProcessedCount: 1, StalledCount: 1}
}

func (s *RecruitmentEntryService) processValidOngoingEntry(entry model.NewbroRecruitmentEntry, user *model.User, rewardAmount float64, now time.Time) RecruitEntryResult {
	rewardRefID := buildRecruitRewardRefID(user.ID)
	alreadyClaimed, err := s.repo.HasEntryWithWalletRefID(rewardRefID)
	if err != nil {
		global.Logger.Error("检查招募奖励去重失败", zap.Uint("entry_id", entry.ID), zap.Error(err))
		return RecruitEntryResult{}
	}
	if alreadyClaimed {
		return s.markEntryStalledResult(entry.ID, "标记重复招募记录为 stalled 失败")
	}

	recruitment, err := s.repo.GetRecruitmentByID(entry.RecruitmentID)
	if err != nil {
		global.Logger.Error("加载招募记录失败", zap.Uint("entry_id", entry.ID), zap.Uint("recruitment_id", entry.RecruitmentID), zap.Error(err))
		return RecruitEntryResult{}
	}
	if recruitment == nil {
		global.Logger.Error("招募记录不存在", zap.Uint("entry_id", entry.ID), zap.Uint("recruitment_id", entry.RecruitmentID))
		return RecruitEntryResult{}
	}

	if err := s.rewardOngoingEntry(entry, recruitment.UserID, user.ID, rewardAmount, rewardRefID, now); err != nil {
		delta := s.resolveRewardConflict(entry.ID, rewardRefID)
		if delta.ProcessedCount != 0 {
			return delta
		}
		global.Logger.Error("处理招募奖励失败", zap.Uint("entry_id", entry.ID), zap.Error(err))
		return RecruitEntryResult{}
	}

	return RecruitEntryResult{ProcessedCount: 1, ValidCount: 1, TotalCoinAwarded: rewardAmount}
}

func (s *RecruitmentEntryService) rewardOngoingEntry(entry model.NewbroRecruitmentEntry, recruiterUserID, matchedUserID uint, rewardAmount float64, rewardRefID string, now time.Time) error {
	return global.DB.Transaction(func(tx *gorm.DB) error {
		rewardedAt := now
		if err := s.repo.MarkEntryValidTx(tx, entry.ID, matchedUserID, rewardedAt, rewardRefID); err != nil {
			return err
		}
		return s.walletSvc.ApplyWalletDeltaTx(
			tx,
			recruiterUserID,
			rewardAmount,
			fmt.Sprintf("招募链接奖励（招募 QQ %s）", entry.QQ),
			model.WalletRefRecruitReward,
			rewardRefID,
		)
	})
}

func (s *RecruitmentEntryService) resolveRewardConflict(entryID uint, rewardRefID string) RecruitEntryResult {
	alreadyClaimed, err := s.repo.HasEntryWithWalletRefID(rewardRefID)
	if err == nil && alreadyClaimed {
		return s.markEntryStalledResult(entryID, "标记重复招募记录为 stalled 失败")
	}
	return RecruitEntryResult{}
}

func (s *RecruitmentEntryService) processOngoingEntry(entry model.NewbroRecruitmentEntry, now time.Time, cooldownDays int, rewardAmount float64) RecruitEntryResult {
	user, userFound, shouldSkip := s.lookupMatchedUserForEntry(entry)
	if shouldSkip {
		return RecruitEntryResult{}
	}

	userCreatedAt := time.Time{}
	if userFound {
		userCreatedAt = user.CreatedAt
	}

	newStatus := resolveEntryStatus(entry.EnteredAt, userCreatedAt, userFound, now, cooldownDays)
	switch newStatus {
	case model.RecruitEntryStatusOngoing:
		return RecruitEntryResult{}
	case model.RecruitEntryStatusStalled:
		return s.markEntryStalledResult(entry.ID, "标记招募记录为 stalled 失败")
	default:
		return s.processValidOngoingEntry(entry, user, rewardAmount, now)
	}
}

// ProcessOngoingEntries is the daily job logic. It resolves pending entry statuses.
func (s *RecruitmentEntryService) ProcessOngoingEntries(now time.Time) (*RecruitEntryResult, error) {
	settings := s.settingsSvc.GetSettings()
	cooldownDays := settings.RecruitCooldownDays
	rewardAmount := settings.RecruitRewardAmount

	const batchSize = 200
	result := &RecruitEntryResult{}
	var lastID uint

	for {
		entries, err := s.repo.ListOngoingEntriesAfterID(lastID, batchSize)
		if err != nil {
			return nil, err
		}
		if len(entries) == 0 {
			break
		}

		for _, entry := range entries {
			lastID = entry.ID
			mergeRecruitEntryResult(result, s.processOngoingEntry(entry, now, cooldownDays, rewardAmount))
		}

		if len(entries) < batchSize {
			break
		}
	}

	return result, nil
}

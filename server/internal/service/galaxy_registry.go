package service

import (
	"amiya-eden/global"
	"amiya-eden/internal/model"
	"amiya-eden/internal/repository"
	"amiya-eden/pkg/eve/esi"
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"go.uber.org/zap"
	"gorm.io/gorm"
)

type galaxyRegistryTokenProvider interface {
	GetValidToken(ctx context.Context, characterID int64) (string, error)
}

type GalaxyRegistryService struct {
	repo     *repository.GalaxyRegistryRepository
	charRepo *repository.EveCharacterRepository
	userRepo *repository.UserRepository
	ssoSvc   galaxyRegistryTokenProvider
	wallet   interface {
		Execute(ctx *esi.TaskContext) error
	}
	esiClient *esi.Client
}

func NewGalaxyRegistryService() *GalaxyRegistryService {
	cfg := configuredEveSSOConfig()
	return &GalaxyRegistryService{
		repo:     repository.NewGalaxyRegistryRepository(),
		charRepo: repository.NewEveCharacterRepository(),
		userRepo: repository.NewUserRepository(),
		ssoSvc:   NewEveSSOService(),
		wallet:   &esi.WalletTask{},
		esiClient: esi.NewClientWithConfig(
			cfg.ESIBaseURL,
			cfg.ESIAPIPrefix,
		),
	}
}

type GalaxyRegistrySystemSummary struct {
	IdleCount    int `json:"idle_count"`
	BusyCount    int `json:"busy_count"`
	OverdueCount int `json:"overdue_count"`
}

type GalaxyRegistrySystemActiveEntry struct {
	EntryID              uint      `json:"entry_id"`
	CaptainUserID        uint      `json:"captain_user_id"`
	CaptainCharacterID   int64     `json:"captain_character_id"`
	CaptainCharacterName string    `json:"captain_character_name"`
	CaptainNickname      string    `json:"captain_nickname"`
	ExpectedEndAt        time.Time `json:"expected_end_at"`
	ActualStartAt        time.Time `json:"actual_start_at"`
	IsOverdue            bool      `json:"is_overdue"`
	IsMine               bool      `json:"is_mine"`
}

type GalaxyRegistrySystemItem struct {
	ID                uint                             `json:"system_config_id"`
	SolarSystemID     int64                            `json:"solar_system_id"`
	SolarSystemName   string                           `json:"solar_system_name"`
	RegionID          int64                            `json:"region_id"`
	RegionName        string                           `json:"region_name"`
	ConstellationID   int64                            `json:"constellation_id"`
	ConstellationName string                           `json:"constellation_name"`
	Security          float64                          `json:"security"`
	Note              string                           `json:"note"`
	MinBountyAmount   float64                          `json:"min_bounty_amount"`
	IsEnabled         bool                             `json:"is_enabled"`
	Status            string                           `json:"status"`
	ActiveEntry       *GalaxyRegistrySystemActiveEntry `json:"active_entry"`
}

type GalaxyRegistrySystemsResponse struct {
	Summary GalaxyRegistrySystemSummary `json:"summary"`
	Items   []GalaxyRegistrySystemItem  `json:"items"`
}

type GalaxyRegistryCreateEntryRequest struct {
	SystemConfigID uint
	ExpectedEndAt  time.Time
}

type GalaxyRegistryUpdateExpectedEndAtRequest struct {
	ExpectedEndAt time.Time
}

type GalaxyRegistryEntryItem struct {
	ID                    uint       `json:"id"`
	SystemConfigID        uint       `json:"system_config_id"`
	SolarSystemID         int64      `json:"solar_system_id"`
	SolarSystemName       string     `json:"solar_system_name"`
	CaptainUserID         uint       `json:"captain_user_id"`
	CaptainCharacterID    int64      `json:"captain_character_id"`
	CaptainCharacterName  string     `json:"captain_character_name"`
	CaptainNickname       string     `json:"captain_nickname"`
	Status                string     `json:"status"`
	ValidationStatus      string     `json:"validation_status"`
	ExpectedEndAt         time.Time  `json:"expected_end_at"`
	ActualStartAt         time.Time  `json:"actual_start_at"`
	ActualEndAt           *time.Time `json:"actual_end_at"`
	EndedByUserID         uint       `json:"ended_by_user_id"`
	ForceEndedByAdmin     bool       `json:"force_ended_by_admin"`
	FrozenMinBountyAmount float64    `json:"frozen_min_bounty_amount"`
	ValidatedAt           *time.Time `json:"validated_at"`
	ValidatedBountyAmount float64    `json:"validated_bounty_amount"`
	ValidatedBountyCount  int        `json:"validated_bounty_count"`
	ViolationReason       string     `json:"violation_reason"`
	CreatedAt             time.Time  `json:"created_at"`
	UpdatedAt             time.Time  `json:"updated_at"`
}

type GalaxyRegistryEntryListFilter struct {
	SystemConfigID   *uint
	Keyword          string
	Status           string
	ValidationStatus string
	StartDateFrom    *time.Time
	StartDateTo      *time.Time
	EndDateFrom      *time.Time
	EndDateTo        *time.Time
}

type GalaxyRegistryAdminAnalytics struct {
	RangeStart       string                                   `json:"range_start"`
	RangeEnd         string                                   `json:"range_end"`
	CurrentSnapshot  GalaxyRegistryCurrentSnapshot            `json:"current_snapshot"`
	Recent7D         GalaxyRegistryPeriodSummary              `json:"recent_7d"`
	Recent30D        GalaxyRegistryPeriodSummary              `json:"recent_30d"`
	TopSystems       []repository.GalaxyRegistryTopSystemStat `json:"top_systems"`
	RecentViolations []GalaxyRegistryEntryItem                `json:"recent_violations"`
}

type GalaxyRegistryCurrentSnapshot struct {
	IdleCount    int64 `json:"idle_count"`
	BusyCount    int64 `json:"busy_count"`
	OverdueCount int64 `json:"overdue_count"`
}

type GalaxyRegistryPeriodSummary struct {
	EntryCount     int64   `json:"entry_count"`
	ValidCount     int64   `json:"valid_count"`
	ViolationCount int64   `json:"violation_count"`
	PendingCount   int64   `json:"pending_count"`
	ValidRate      float64 `json:"valid_rate"`
}

type galaxyRegistryWalletRefreshItem struct {
	CharacterID int64
	Refreshed   bool
	Skipped     bool
	Reason      string
}

type galaxyRegistryWalletRefreshResult struct {
	Items          []galaxyRegistryWalletRefreshItem
	RefreshedCount int
	SkippedCount   int
}

func isGalaxyRegistrySkippableTokenError(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "该人物的 token 已失效") ||
		strings.Contains(msg, "invalid_token") ||
		strings.Contains(msg, "invalid_grant")
}

func chooseCaptainDisplayCharacter(user *model.User, characters []model.EveCharacter) *model.EveCharacter {
	if len(characters) == 0 {
		return nil
	}
	if user != nil && user.PrimaryCharacterID > 0 {
		for i := range characters {
			if characters[i].CharacterID == user.PrimaryCharacterID {
				return &characters[i]
			}
		}
	}
	return &characters[0]
}

func buildGalaxyRegistryCaptainNickname(user *model.User, characterName string) string {
	if user != nil && strings.TrimSpace(user.Nickname) != "" {
		return strings.TrimSpace(user.Nickname)
	}
	return characterName
}

func (s *GalaxyRegistryService) ListSystems(viewerUserID uint) (*GalaxyRegistrySystemsResponse, error) {
	rows, err := s.repo.ListSystems(repository.GalaxyRegistrySystemListFilter{})
	if err != nil {
		return nil, err
	}

	response := &GalaxyRegistrySystemsResponse{
		Items: make([]GalaxyRegistrySystemItem, 0, len(rows)),
	}
	if len(rows) == 0 {
		return response, nil
	}

	systemIDs := make([]uint, 0, len(rows))
	for _, row := range rows {
		systemIDs = append(systemIDs, row.ID)
	}

	activeEntries, err := s.repo.ListActiveEntriesBySystemConfigIDs(systemIDs)
	if err != nil {
		return nil, err
	}

	userIDs := make([]uint, 0, len(activeEntries))
	for _, entry := range activeEntries {
		userIDs = append(userIDs, entry.CaptainUserID)
	}
	userMap, err := s.loadUserMap(userIDs)
	if err != nil {
		return nil, err
	}

	activeMap := make(map[uint]repository.GalaxyRegistryActiveEntryView, len(activeEntries))
	for _, entry := range activeEntries {
		activeMap[entry.SystemConfigID] = entry
	}

	now := time.Now()
	for _, row := range rows {
		item := GalaxyRegistrySystemItem{
			ID:                row.ID,
			SolarSystemID:     row.SolarSystemID,
			SolarSystemName:   row.SolarSystemName,
			RegionID:          row.RegionID,
			RegionName:        row.RegionName,
			ConstellationID:   row.ConstellationID,
			ConstellationName: row.ConstellationName,
			Security:          row.Security,
			Note:              row.Note,
			MinBountyAmount:   row.MinBountyAmount,
			IsEnabled:         row.IsEnabled,
			Status:            model.GalaxyRegistryStatusIdle,
		}
		if active, ok := activeMap[row.ID]; ok {
			isOverdue := active.ExpectedEndAt.Before(now)
			item.Status = model.GalaxyRegistryStatusBusy
			if isOverdue {
				item.Status = model.GalaxyRegistryStatusOverdue
				response.Summary.OverdueCount++
			}
			response.Summary.BusyCount++
			item.ActiveEntry = &GalaxyRegistrySystemActiveEntry{
				EntryID:              active.ID,
				CaptainUserID:        active.CaptainUserID,
				CaptainCharacterID:   active.CaptainCharacterID,
				CaptainCharacterName: active.CaptainCharacterName,
				CaptainNickname:      buildGalaxyRegistryCaptainNickname(userMap[active.CaptainUserID], active.CaptainCharacterName),
				ExpectedEndAt:        active.ExpectedEndAt,
				ActualStartAt:        active.ActualStartAt,
				IsOverdue:            isOverdue,
				IsMine:               viewerUserID > 0 && active.CaptainUserID == viewerUserID,
			}
		} else {
			response.Summary.IdleCount++
		}
		response.Items = append(response.Items, item)
	}

	return response, nil
}

func (s *GalaxyRegistryService) StartEntry(userID uint, req GalaxyRegistryCreateEntryRequest) (*GalaxyRegistryEntryItem, error) {
	if req.SystemConfigID == 0 {
		return nil, NewUserVisibleError("星系配置不存在")
	}
	if !req.ExpectedEndAt.After(time.Now()) {
		return nil, NewUserVisibleError("预计结束时间必须晚于当前时间")
	}

	user, err := s.userRepo.GetByID(userID)
	if err != nil {
		return nil, err
	}
	characters, err := s.charRepo.ListByUserID(userID)
	if err != nil {
		return nil, err
	}
	displayCharacter := chooseCaptainDisplayCharacter(user, characters)
	if displayCharacter == nil {
		return nil, NewUserVisibleError("当前队长未绑定任何角色")
	}

	now := time.Now()
	var created *model.GalaxyRegistryEntry
	err = withTx(func(tx *gorm.DB) error {
		system, txErr := s.repo.GetSystemByIDForUpdateTx(tx, req.SystemConfigID)
		if txErr != nil {
			return txErr
		}
		if !system.IsEnabled {
			return NewUserVisibleError("该星系未启用登记")
		}
		if _, txErr = s.repo.FindActiveEntryBySystemConfigIDForUpdateTx(tx, req.SystemConfigID); txErr == nil {
			return NewUserVisibleError("该星系当前已有生产登记")
		} else if !errors.Is(txErr, gorm.ErrRecordNotFound) {
			return txErr
		}

		row := &model.GalaxyRegistryEntry{
			SystemConfigID:        system.ID,
			SolarSystemID:         system.SolarSystemID,
			SolarSystemName:       system.SolarSystemName,
			CaptainUserID:         userID,
			CaptainCharacterID:    displayCharacter.CharacterID,
			CaptainCharacterName:  displayCharacter.CharacterName,
			Status:                model.GalaxyRegistryEntryStatusActive,
			ValidationStatus:      model.GalaxyRegistryValidationPending,
			ExpectedEndAt:         req.ExpectedEndAt,
			ActualStartAt:         now,
			FrozenMinBountyAmount: system.MinBountyAmount,
		}
		if txErr = tx.Create(row).Error; txErr != nil {
			return txErr
		}
		created = row
		return nil
	})
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, NewUserVisibleError("星系配置不存在")
		}
		return nil, err
	}

	return s.buildEntryItem(created, user)
}

func (s *GalaxyRegistryService) EndMyEntry(userID uint, entryID uint) (*GalaxyRegistryEntryItem, error) {
	return s.EndMyEntryWithContext(context.Background(), userID, entryID)
}

func (s *GalaxyRegistryService) ForceEndEntry(operatorID uint, entryID uint) (*GalaxyRegistryEntryItem, error) {
	return s.ForceEndEntryWithContext(context.Background(), operatorID, entryID)
}

func (s *GalaxyRegistryService) EndMyEntryWithContext(ctx context.Context, userID uint, entryID uint) (*GalaxyRegistryEntryItem, error) {
	return s.endEntry(ctx, userID, entryID, false)
}

func (s *GalaxyRegistryService) ForceEndEntryWithContext(ctx context.Context, operatorID uint, entryID uint) (*GalaxyRegistryEntryItem, error) {
	return s.endEntry(ctx, operatorID, entryID, true)
}

func (s *GalaxyRegistryService) endEntry(ctx context.Context, operatorID uint, entryID uint, force bool) (*GalaxyRegistryEntryItem, error) {
	now := time.Now()
	var updated *model.GalaxyRegistryEntry
	err := withTx(func(tx *gorm.DB) error {
		row, txErr := s.repo.GetEntryByIDForUpdateTx(tx, entryID)
		if txErr != nil {
			return txErr
		}
		if row.Status != model.GalaxyRegistryEntryStatusActive {
			return NewUserVisibleError("该登记已结束")
		}
		if !force && row.CaptainUserID != operatorID {
			return NewUserVisibleError("只能结束自己的登记")
		}
		updates := map[string]any{
			"status":               model.GalaxyRegistryEntryStatusCompleted,
			"actual_end_at":        now,
			"ended_by_user_id":     operatorID,
			"force_ended_by_admin": force,
			"validation_status":    model.GalaxyRegistryValidationPending,
		}
		if txErr = tx.Model(&model.GalaxyRegistryEntry{}).Where("id = ?", entryID).Updates(updates).Error; txErr != nil {
			return txErr
		}
		row.Status = model.GalaxyRegistryEntryStatusCompleted
		row.ActualEndAt = &now
		row.EndedByUserID = operatorID
		row.ForceEndedByAdmin = force
		row.ValidationStatus = model.GalaxyRegistryValidationPending
		updated = row
		return nil
	})
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, NewUserVisibleError("登记不存在")
		}
		return nil, err
	}

	if err := s.finalizeEntryValidation(ctx, updated); err != nil {
		return nil, err
	}

	user, err := s.userRepo.GetByID(updated.CaptainUserID)
	if err != nil {
		return nil, err
	}
	return s.buildEntryItem(updated, user)
}

func (s *GalaxyRegistryService) ListMyEntries(userID uint, filter GalaxyRegistryEntryListFilter, page int, pageSize int) ([]GalaxyRegistryEntryItem, int64, error) {
	return s.listEntries(&userID, filter, page, pageSize)
}

func (s *GalaxyRegistryService) UpdateMyExpectedEndAt(
	userID uint,
	entryID uint,
	req GalaxyRegistryUpdateExpectedEndAtRequest,
) (*GalaxyRegistryEntryItem, error) {
	if req.ExpectedEndAt.IsZero() {
		return nil, NewUserVisibleError("预计结束时间不能为空")
	}

	var updated *model.GalaxyRegistryEntry
	err := withTx(func(tx *gorm.DB) error {
		row, txErr := s.repo.GetEntryByIDForUpdateTx(tx, entryID)
		if txErr != nil {
			return txErr
		}
		if row.CaptainUserID != userID {
			return NewUserVisibleError("只能修改自己的登记")
		}
		if row.Status != model.GalaxyRegistryEntryStatusActive {
			return NewUserVisibleError("只能修改进行中的登记")
		}
		if !req.ExpectedEndAt.After(row.ActualStartAt) {
			return NewUserVisibleError("预计结束时间必须晚于实际开始时间")
		}
		if txErr = tx.Model(&model.GalaxyRegistryEntry{}).
			Where("id = ?", entryID).
			Update("expected_end_at", req.ExpectedEndAt).Error; txErr != nil {
			return txErr
		}
		row.ExpectedEndAt = req.ExpectedEndAt
		updated = row
		return nil
	})
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, NewUserVisibleError("登记不存在")
		}
		return nil, err
	}

	user, err := s.userRepo.GetByID(updated.CaptainUserID)
	if err != nil {
		return nil, err
	}
	return s.buildEntryItem(updated, user)
}

func (s *GalaxyRegistryService) ListAdminEntries(filter GalaxyRegistryEntryListFilter, page int, pageSize int) ([]GalaxyRegistryEntryItem, int64, error) {
	return s.listEntries(nil, filter, page, pageSize)
}

func (s *GalaxyRegistryService) OverrideEntryValidation(
	entryID uint,
	validationStatus string,
	violationReason *string,
) (*GalaxyRegistryEntryItem, error) {
	switch validationStatus {
	case model.GalaxyRegistryValidationValid, model.GalaxyRegistryValidationViolation:
	default:
		return nil, NewUserVisibleError("无效的校验状态")
	}

	var updated *model.GalaxyRegistryEntry
	err := withTx(func(tx *gorm.DB) error {
		row, txErr := s.repo.GetEntryByIDForUpdateTx(tx, entryID)
		if txErr != nil {
			return txErr
		}
		if row.Status != model.GalaxyRegistryEntryStatusCompleted {
			return NewUserVisibleError("只能修改已结束登记的校验结果")
		}

		reason := ""
		if validationStatus == model.GalaxyRegistryValidationViolation && violationReason != nil {
			reason = strings.TrimSpace(*violationReason)
		}
		now := time.Now()
		if txErr = tx.Model(&model.GalaxyRegistryEntry{}).
			Where("id = ?", entryID).
			Updates(map[string]any{
				"validation_status": validationStatus,
				"validated_at":      now,
				"violation_reason":  reason,
			}).Error; txErr != nil {
			return txErr
		}
		row.ValidationStatus = validationStatus
		row.ValidatedAt = &now
		row.ViolationReason = reason
		updated = row
		return nil
	})
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, NewUserVisibleError("登记不存在")
		}
		return nil, err
	}

	user, err := s.userRepo.GetByID(updated.CaptainUserID)
	if err != nil {
		return nil, err
	}
	return s.buildEntryItem(updated, user)
}

func (s *GalaxyRegistryService) RevalidateEntryWithContext(
	ctx context.Context,
	entryID uint,
) (*GalaxyRegistryEntryItem, error) {
	row, err := s.repo.GetEntryByID(entryID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, NewUserVisibleError("登记不存在")
		}
		return nil, err
	}
	if row.Status != model.GalaxyRegistryEntryStatusCompleted {
		return nil, NewUserVisibleError("只能重新校验已结束的登记")
	}

	if err := s.finalizeEntryValidation(ctx, row); err != nil {
		return nil, err
	}

	user, err := s.userRepo.GetByID(row.CaptainUserID)
	if err != nil {
		return nil, err
	}
	return s.buildEntryItem(row, user)
}

func (s *GalaxyRegistryService) listEntries(ownerUserID *uint, filter GalaxyRegistryEntryListFilter, page int, pageSize int) ([]GalaxyRegistryEntryItem, int64, error) {
	repoFilter := repository.GalaxyRegistryEntryListFilter{
		SystemConfigID:   filter.SystemConfigID,
		CaptainUserID:    ownerUserID,
		Keyword:          filter.Keyword,
		Status:           filter.Status,
		ValidationStatus: filter.ValidationStatus,
		StartDateFrom:    filter.StartDateFrom,
		StartDateTo:      filter.StartDateTo,
		EndDateFrom:      filter.EndDateFrom,
		EndDateTo:        filter.EndDateTo,
	}
	rows, total, err := s.repo.ListEntries(repoFilter, page, pageSize)
	if err != nil {
		return nil, 0, err
	}
	userIDs := make([]uint, 0, len(rows))
	for _, row := range rows {
		userIDs = append(userIDs, row.CaptainUserID)
	}
	userMap, err := s.loadUserMap(userIDs)
	if err != nil {
		return nil, 0, err
	}
	items := make([]GalaxyRegistryEntryItem, 0, len(rows))
	for i := range rows {
		item, buildErr := s.buildEntryItem(&rows[i], userMap[rows[i].CaptainUserID])
		if buildErr != nil {
			return nil, 0, buildErr
		}
		items = append(items, *item)
	}
	return items, total, nil
}

func (s *GalaxyRegistryService) ListAdminSystems() ([]model.GalaxyRegistrySystem, error) {
	return s.repo.ListSystems(repository.GalaxyRegistrySystemListFilter{})
}

func (s *GalaxyRegistryService) CreateAdminSystem(
	solarSystemID int64,
	note string,
	minBountyAmount float64,
	isEnabled bool,
) (*model.GalaxyRegistrySystem, error) {
	if solarSystemID <= 0 {
		return nil, NewUserVisibleError("无效的星系 ID")
	}
	if minBountyAmount < 0 {
		return nil, NewUserVisibleError("最低有效赏金阈值不能为负数")
	}
	sdeRows, err := repository.NewSdeRepository().SearchSolarSystemsByKeyword("", solarSystemID, 1)
	if err != nil {
		return nil, err
	}
	if len(sdeRows) == 0 {
		return nil, NewUserVisibleError("未找到对应星系")
	}
	row := &model.GalaxyRegistrySystem{
		SolarSystemID:     sdeRows[0].SolarSystemID,
		SolarSystemName:   sdeRows[0].SolarSystemName,
		RegionID:          sdeRows[0].RegionID,
		RegionName:        sdeRows[0].RegionName,
		ConstellationID:   sdeRows[0].ConstellationID,
		ConstellationName: sdeRows[0].ConstellationName,
		Security:          sdeRows[0].Security,
		Note:              strings.TrimSpace(note),
		MinBountyAmount:   minBountyAmount,
		IsEnabled:         isEnabled,
	}
	if row.MinBountyAmount == 0 {
		row.MinBountyAmount = model.GalaxyRegistryDefaultMinBountyAmount
	}
	if err := s.repo.CreateSystem(row); err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "duplicate") {
			return nil, NewUserVisibleError("该星系已存在配置")
		}
		return nil, err
	}
	return row, nil
}

func (s *GalaxyRegistryService) UpdateAdminSystem(
	id uint,
	note *string,
	minBountyAmount *float64,
	isEnabled *bool,
) (*model.GalaxyRegistrySystem, error) {
	updates := map[string]any{}
	if note != nil {
		updates["note"] = strings.TrimSpace(*note)
	}
	if minBountyAmount != nil {
		if *minBountyAmount < 0 {
			return nil, NewUserVisibleError("最低有效赏金阈值不能为负数")
		}
		updates["min_bounty_amount"] = *minBountyAmount
	}
	if isEnabled != nil {
		updates["is_enabled"] = *isEnabled
	}
	if len(updates) == 0 {
		return s.repo.GetSystemByID(id)
	}
	if err := s.repo.UpdateSystem(id, updates); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, NewUserVisibleError("星系配置不存在")
		}
		return nil, err
	}
	return s.repo.GetSystemByID(id)
}

func (s *GalaxyRegistryService) DeleteAdminSystem(id uint) error {
	hasActive, err := s.repo.HasActiveEntryBySystemConfigID(id)
	if err != nil {
		return err
	}
	if hasActive {
		return NewUserVisibleError("该星系存在进行中的登记，不能删除")
	}
	if err := s.repo.DeleteSystem(id); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return NewUserVisibleError("星系配置不存在")
		}
		return err
	}
	return nil
}

func (s *GalaxyRegistryService) SearchAdminSdeSystems(keyword string, limit int) ([]repository.GalaxyRegistrySdeSystem, error) {
	if limit <= 0 {
		limit = 20
	}
	keyword = strings.TrimSpace(keyword)
	if keyword == "" {
		return []repository.GalaxyRegistrySdeSystem{}, nil
	}
	return repository.NewSdeRepository().SearchSolarSystemsByKeyword(keyword, 0, limit)
}

func (s *GalaxyRegistryService) GetAdminAnalytics(startDate *time.Time, endDate *time.Time) (*GalaxyRegistryAdminAnalytics, error) {
	now := time.Now()
	currentStart := now.AddDate(0, 0, -30)
	currentEnd := now
	if startDate != nil {
		currentStart = *startDate
	}
	if endDate != nil {
		currentEnd = *endDate
	}

	idleEnabledCount, err := s.repo.CountSystemsByEnabled(true)
	if err != nil {
		return nil, err
	}
	busyCount, err := s.repo.CountCurrentBusySystems()
	if err != nil {
		return nil, err
	}
	overdueCount, err := s.repo.CountCurrentOverdueEntries(now)
	if err != nil {
		return nil, err
	}
	if idleEnabledCount < busyCount {
		idleEnabledCount = busyCount
	}

	recent7D, err := s.buildPeriodSummary(now.AddDate(0, 0, -7), now)
	if err != nil {
		return nil, err
	}
	recent30D, err := s.buildPeriodSummary(currentStart, currentEnd)
	if err != nil {
		return nil, err
	}
	topSystems, err := s.repo.ListTopSystemsByRegistrations(currentStart, currentEnd, 10)
	if err != nil {
		return nil, err
	}
	violations, err := s.repo.ListRecentViolations(currentStart, currentEnd, 10)
	if err != nil {
		return nil, err
	}
	userIDs := make([]uint, 0, len(violations))
	for _, row := range violations {
		userIDs = append(userIDs, row.CaptainUserID)
	}
	userMap, err := s.loadUserMap(userIDs)
	if err != nil {
		return nil, err
	}
	violationItems := make([]GalaxyRegistryEntryItem, 0, len(violations))
	for i := range violations {
		item, buildErr := s.buildEntryItem(&violations[i], userMap[violations[i].CaptainUserID])
		if buildErr != nil {
			return nil, buildErr
		}
		violationItems = append(violationItems, *item)
	}

	return &GalaxyRegistryAdminAnalytics{
		RangeStart: currentStart.Format(time.RFC3339),
		RangeEnd:   currentEnd.Format(time.RFC3339),
		CurrentSnapshot: GalaxyRegistryCurrentSnapshot{
			IdleCount:    idleEnabledCount - busyCount,
			BusyCount:    busyCount,
			OverdueCount: overdueCount,
		},
		Recent7D:         recent7D,
		Recent30D:        recent30D,
		TopSystems:       topSystems,
		RecentViolations: violationItems,
	}, nil
}

func (s *GalaxyRegistryService) buildPeriodSummary(start time.Time, end time.Time) (GalaxyRegistryPeriodSummary, error) {
	entryCount, err := s.repo.CountEntriesCompletedBetween(start, end)
	if err != nil {
		return GalaxyRegistryPeriodSummary{}, err
	}
	validCount, err := s.repo.CountEntriesByValidationBetween(model.GalaxyRegistryValidationValid, start, end)
	if err != nil {
		return GalaxyRegistryPeriodSummary{}, err
	}
	violationCount, err := s.repo.CountEntriesByValidationBetween(model.GalaxyRegistryValidationViolation, start, end)
	if err != nil {
		return GalaxyRegistryPeriodSummary{}, err
	}
	pendingCount, err := s.repo.CountEntriesByValidationBetween(model.GalaxyRegistryValidationPending, start, end)
	if err != nil {
		return GalaxyRegistryPeriodSummary{}, err
	}
	validRate := 0.0
	if entryCount > 0 {
		validRate = float64(validCount) / float64(entryCount)
	}
	return GalaxyRegistryPeriodSummary{
		EntryCount:     entryCount,
		ValidCount:     validCount,
		ViolationCount: violationCount,
		PendingCount:   pendingCount,
		ValidRate:      validRate,
	}, nil
}

func (s *GalaxyRegistryService) ValidateCompletedEntries(limit int) (int, error) {
	rows, err := s.repo.ListPendingValidationCandidates(limit)
	if err != nil {
		return 0, err
	}
	validated := 0
	for _, row := range rows {
		entry, loadErr := s.repo.GetEntryByID(row.ID)
		if loadErr != nil {
			return validated, loadErr
		}
		status, amount, count, reason, evaluatedAt, evalErr := s.evaluateEntryValidation(entry)
		if evalErr != nil {
			return validated, evalErr
		}
		if err = s.repo.UpdateEntryValidationResult(row.ID, status, amount, count, evaluatedAt, reason); err != nil {
			return validated, err
		}
		validated++
	}
	return validated, nil
}

func (s *GalaxyRegistryService) finalizeEntryValidation(ctx context.Context, row *model.GalaxyRegistryEntry) error {
	characters, err := s.charRepo.ListByUserID(row.CaptainUserID)
	if err != nil {
		return err
	}
	refreshResult, err := s.refreshWalletJournals(ctx, characters)
	if err != nil {
		return NewUserVisibleError("登记已结束，但拉取钱包 ESI 失败，暂未完成校验")
	}
	if refreshResult.RefreshedCount == 0 {
		return NewUserVisibleError("登记已结束，但拉取钱包 ESI 失败，暂未完成校验")
	}

	status, amount, count, reason, evaluatedAt, err := s.evaluateEntryValidation(row)
	if err != nil {
		return err
	}
	if err := s.repo.UpdateEntryValidationResult(row.ID, status, amount, count, evaluatedAt, reason); err != nil {
		return err
	}
	row.ValidationStatus = status
	row.ValidatedBountyAmount = amount
	row.ValidatedBountyCount = count
	row.ValidatedAt = &evaluatedAt
	row.ViolationReason = reason
	return nil
}

func (s *GalaxyRegistryService) refreshWalletJournals(
	ctx context.Context,
	characters []model.EveCharacter,
) (*galaxyRegistryWalletRefreshResult, error) {
	result := &galaxyRegistryWalletRefreshResult{
		Items: make([]galaxyRegistryWalletRefreshItem, 0, len(characters)),
	}
	for i := range characters {
		item := galaxyRegistryWalletRefreshItem{CharacterID: characters[i].CharacterID}

		token, err := s.ssoSvc.GetValidToken(ctx, characters[i].CharacterID)
		if err != nil {
			if isGalaxyRegistrySkippableTokenError(err) {
				item.Skipped = true
				item.Reason = err.Error()
				result.Items = append(result.Items, item)
				result.SkippedCount++
				global.Logger.Warn("GalaxyRegistry: 角色 token 失效，跳过钱包刷新",
					zap.Int64("character_id", characters[i].CharacterID),
					zap.String("phase", "get_valid_token"),
					zap.Error(err),
				)
				continue
			}
			return nil, fmt.Errorf("get valid token for character %d: %w", characters[i].CharacterID, err)
		}

		if err := s.wallet.Execute(&esi.TaskContext{
			Context:     ctx,
			CharacterID: characters[i].CharacterID,
			AccessToken: token,
			Client:      s.esiClient,
			IsActive:    true,
		}); err != nil {
			if isGalaxyRegistrySkippableTokenError(err) {
				item.Skipped = true
				item.Reason = err.Error()
				result.Items = append(result.Items, item)
				result.SkippedCount++
				global.Logger.Warn("GalaxyRegistry: 角色 token 失效，跳过钱包刷新",
					zap.Int64("character_id", characters[i].CharacterID),
					zap.String("phase", "refresh_wallet"),
					zap.Error(err),
				)
				continue
			}
			return nil, fmt.Errorf("refresh wallet for character %d: %w", characters[i].CharacterID, err)
		}

		item.Refreshed = true
		result.Items = append(result.Items, item)
		result.RefreshedCount++
	}
	return result, nil
}

func (s *GalaxyRegistryService) evaluateEntryValidation(
	row *model.GalaxyRegistryEntry,
) (string, float64, int, string, time.Time, error) {
	characters, err := s.charRepo.ListByUserID(row.CaptainUserID)
	if err != nil {
		return "", 0, 0, "", time.Time{}, err
	}
	characterIDs := make([]int64, 0, len(characters))
	for _, character := range characters {
		characterIDs = append(characterIDs, character.CharacterID)
	}

	endAt := time.Now()
	if row.ActualEndAt != nil {
		endAt = *row.ActualEndAt
	}
	amount, count, err := s.repo.SumBountyJournalInWindow(characterIDs, row.SolarSystemID, row.ActualStartAt, endAt)
	if err != nil {
		return "", 0, 0, "", time.Time{}, err
	}

	status := model.GalaxyRegistryValidationViolation
	reason := ""
	if amount >= row.FrozenMinBountyAmount {
		status = model.GalaxyRegistryValidationValid
	} else if count == 0 {
		reason = model.GalaxyRegistryViolationNoBountyInWindow
	} else {
		reason = model.GalaxyRegistryViolationBountyBelowThreshold
	}

	return status, amount, int(count), reason, time.Now(), nil
}

func (s *GalaxyRegistryService) buildEntryItem(row *model.GalaxyRegistryEntry, user *model.User) (*GalaxyRegistryEntryItem, error) {
	return &GalaxyRegistryEntryItem{
		ID:                    row.ID,
		SystemConfigID:        row.SystemConfigID,
		SolarSystemID:         row.SolarSystemID,
		SolarSystemName:       row.SolarSystemName,
		CaptainUserID:         row.CaptainUserID,
		CaptainCharacterID:    row.CaptainCharacterID,
		CaptainCharacterName:  row.CaptainCharacterName,
		CaptainNickname:       buildGalaxyRegistryCaptainNickname(user, row.CaptainCharacterName),
		Status:                row.Status,
		ValidationStatus:      row.ValidationStatus,
		ExpectedEndAt:         row.ExpectedEndAt,
		ActualStartAt:         row.ActualStartAt,
		ActualEndAt:           row.ActualEndAt,
		EndedByUserID:         row.EndedByUserID,
		ForceEndedByAdmin:     row.ForceEndedByAdmin,
		FrozenMinBountyAmount: row.FrozenMinBountyAmount,
		ValidatedAt:           row.ValidatedAt,
		ValidatedBountyAmount: row.ValidatedBountyAmount,
		ValidatedBountyCount:  row.ValidatedBountyCount,
		ViolationReason:       row.ViolationReason,
		CreatedAt:             row.CreatedAt,
		UpdatedAt:             row.UpdatedAt,
	}, nil
}

func (s *GalaxyRegistryService) loadUserMap(userIDs []uint) (map[uint]*model.User, error) {
	result := make(map[uint]*model.User)
	if len(userIDs) == 0 {
		return result, nil
	}
	users, err := s.userRepo.ListByIDs(userIDs)
	if err != nil {
		return nil, err
	}
	for i := range users {
		user := users[i]
		result[user.ID] = &user
	}
	return result, nil
}

func withTx(fn func(tx *gorm.DB) error) error {
	return global.DB.Transaction(fn)
}

func (s *GalaxyRegistryService) LoadSystem(id uint) (*model.GalaxyRegistrySystem, error) {
	return s.repo.GetSystemByID(id)
}

func (s *GalaxyRegistryService) DescribeViolationReason(reason string) string {
	switch reason {
	case model.GalaxyRegistryViolationNoBountyInWindow:
		return "no_bounty_in_window"
	case model.GalaxyRegistryViolationBountyBelowThreshold:
		return "bounty_below_threshold"
	default:
		return reason
	}
}

func (s *GalaxyRegistryService) ValidateSystemConfigExists(id uint) error {
	if id == 0 {
		return NewUserVisibleError("星系配置不存在")
	}
	if _, err := s.repo.GetSystemByID(id); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return NewUserVisibleError("星系配置不存在")
		}
		return fmt.Errorf("load system config: %w", err)
	}
	return nil
}

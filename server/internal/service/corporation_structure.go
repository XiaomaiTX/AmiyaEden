package service

import (
	"amiya-eden/global"
	"amiya-eden/internal/model"
	"amiya-eden/internal/repository"
	"amiya-eden/internal/utils"
	"amiya-eden/pkg/eve/esi"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"sort"
	"strconv"
	"strings"
	"time"

	"go.uber.org/zap"
	"gorm.io/gorm"
)

const (
	corporationStructureFuelBucketAll    = "all"
	corporationStructureFuelBucket24h    = "lt_24h"
	corporationStructureFuelBucket72h    = "lt_72h"
	corporationStructureFuelBucket168h   = "lt_168h"
	corporationStructureFuelBucketCustom = "custom"

	corporationStructureTimerBucketAll         = "all"
	corporationStructureTimerBucketCurrentHour = "current_hour"
	corporationStructureTimerBucketNext2Hours  = "next_2_hours"
	corporationStructureTimerBucketCustom      = "custom"

	corporationStructureServiceMatchAnd = "and"
	corporationStructureServiceMatchOr  = "or"

	corporationStructureSortFuelRemainingHours = "fuel_remaining_hours"
	corporationStructureSortSecurity           = "security"
	corporationStructureSortReinforceHour      = "reinforce_hour"
	corporationStructureSortStateTimerEnd      = "state_timer_end"
	corporationStructureSortUpdatedAt          = "updated_at"
	corporationStructureSortSystemName         = "system_name"
	corporationStructureSortName               = "name"
	corporationStructureSortTypeName           = "type_name"
	corporationStructureSortCorporationName    = "corporation_name"

	corporationStructureSortOrderAsc  = "asc"
	corporationStructureSortOrderDesc = "desc"

	hoursPerDay = 24
)

var (
	corporationStructureStateGroupMap = map[string][]string{
		"online": {
			"shield_vulnerable",
		},
		"low_power": {
			"low_power",
		},
		"abandoned": {
			"abandoned",
		},
		"reinforced": {
			"shield_reinforce",
			"armor_reinforce",
			"armor_vulnerable",
			"hull_reinforce",
			"hull_vulnerable",
		},
	}
	corporationStructureSupportedSortBy = map[string]struct{}{
		corporationStructureSortFuelRemainingHours: {},
		corporationStructureSortSecurity:           {},
		corporationStructureSortReinforceHour:      {},
		corporationStructureSortStateTimerEnd:      {},
		corporationStructureSortUpdatedAt:          {},
		corporationStructureSortSystemName:         {},
		corporationStructureSortName:               {},
		corporationStructureSortTypeName:           {},
		corporationStructureSortCorporationName:    {},
	}
)

type CorporationStructureService struct {
	roleRepo      *repository.RoleRepository
	charRepo      *repository.EveCharacterRepository
	sysConfigRepo *repository.SysConfigRepository
	sdeRepo       *repository.SdeRepository
	repo          *repository.CorporationStructureRepository
	esiClient     *esi.Client
	auditSvc      *AuditService
	nameResolver  *EntityNameResolver
	fuelRateRepo  *repository.StructureServiceFuelRateRepository
	groupResolver StructureTypeGroupResolver
	alertNotifier corporationStructureAlertNotifier
}

type corporationStructureAlertNotifier interface {
	EnqueueStructureAlertNotifications([]int64, string) error
}

func NewCorporationStructureService() *CorporationStructureService {
	cfg := global.Config.EveSSO
	sdeRepo := repository.NewSdeRepository()
	return &CorporationStructureService{
		roleRepo:      repository.NewRoleRepository(),
		charRepo:      repository.NewEveCharacterRepository(),
		sysConfigRepo: repository.NewSysConfigRepository(),
		sdeRepo:       sdeRepo,
		repo:          repository.NewCorporationStructureRepository(),
		esiClient:     esi.NewClientWithConfig(cfg.ESIBaseURL, cfg.ESIAPIPrefix),
		auditSvc:      NewAuditService(),
		nameResolver:  NewEntityNameResolver(),
		fuelRateRepo:  repository.NewStructureServiceFuelRateRepository(),
		groupResolver: sdeRepo,
		alertNotifier: DefaultQQGovernanceService(),
	}
}

// loadServiceFuelRateMap 加载服务燃料率映射（一次性，供批量构建行复用）。
// DB 中的记录覆盖默认回退表：DB 有则以 DB 为准，DB 缺失的服务沿用默认值。
func (s *CorporationStructureService) loadServiceFuelRateMap() map[string]float64 {
	return loadRateMapWithRepo(s.fuelRateRepo)
}

// loadStructureGroupIDMap 批量解析建筑 typeID → groupID（供燃料折扣判定）。
// 缺失的 typeID 不在结果中（按无折扣计算并记录告警）。
func (s *CorporationStructureService) loadStructureGroupIDMap(structures []model.CorpStructureInfo) map[int64]int {
	if len(structures) == 0 {
		return map[int64]int{}
	}
	typeIDSet := make(map[int]struct{}, len(structures))
	for _, st := range structures {
		if st.TypeID > 0 {
			typeIDSet[int(st.TypeID)] = struct{}{}
		}
	}
	if len(typeIDSet) == 0 {
		return map[int64]int{}
	}
	typeIDs := make([]int, 0, len(typeIDSet))
	for id := range typeIDSet {
		typeIDs = append(typeIDs, id)
	}
	groupMap, err := resolveGroupIDMap(s.groupResolver, typeIDs)
	if err != nil {
		logCorporationStructuresWarn(
			"[CorporationStructures] 解析建筑 groupID 失败，燃料折扣按无折扣计算",
			err,
		)
		return map[int64]int{}
	}
	missing := 0
	for _, id := range typeIDs {
		if _, ok := groupMap[int64(id)]; !ok {
			missing++
		}
	}
	if missing > 0 {
		logCorporationStructuresWarn(
			"[CorporationStructures] 部分 typeID 缺失 SDE 分组，对应建筑燃料按无折扣计算",
			nil,
			zap.Int("missing_count", missing),
		)
	}
	return groupMap
}

type CorporationStructureServiceInfo struct {
	Name  string `json:"name"`
	State string `json:"state"`
}

type CorporationStructureRow struct {
	CorporationID          int64                             `json:"corporation_id"`
	CorporationName        string                            `json:"corporation_name"`
	AssignedUserID         uint                              `json:"assigned_user_id"`
	AssignedCharacterID    int64                             `json:"assigned_character_id"`
	AssignedCharacterName  string                            `json:"assigned_character_name"`
	StructureID            int64                             `json:"structure_id"`
	Name                   string                            `json:"name"`
	TypeID                 int64                             `json:"type_id"`
	TypeName               string                            `json:"type_name"`
	SystemID               int64                             `json:"system_id"`
	SystemName             string                            `json:"system_name"`
	RegionID               int64                             `json:"region_id"`
	RegionName             string                            `json:"region_name"`
	Security               float64                           `json:"security"`
	State                  string                            `json:"state"`
	Services               []CorporationStructureServiceInfo `json:"services"`
	FuelExpires            string                            `json:"fuel_expires"`
	FuelRemaining          string                            `json:"fuel_remaining"`
	FuelRemainingHours     *int                              `json:"fuel_remaining_hours"`
	FuelPerHour            *float64                          `json:"fuel_per_hour"`            // 预计每小时消耗燃料块
	FuelToMonthEnd         *int                              `json:"fuel_to_month_end"`        // 预计到月底需补燃料块
	FuelEstimateIncomplete bool                              `json:"fuel_estimate_incomplete"` // 存在未配置在线服务时为 true，fuel_per_hour 暂不可用
	FuelUnknownServices    []string                          `json:"fuel_unknown_services"`    // 未映射服务原始名列表
	ReinforceHour          int                               `json:"reinforce_hour"`
	StateTimerStart        string                            `json:"state_timer_start"`
	StateTimerEnd          string                            `json:"state_timer_end"`
	UpdatedAt              int64                             `json:"updated_at"`
}

type CorporationStructureListRequest struct {
	CorporationID    int64    `json:"corporation_id"`
	Page             int      `json:"page"`
	PageSize         int      `json:"page_size"`
	Keyword          string   `json:"keyword"`
	StateGroups      []string `json:"state_groups"`
	FuelBucket       string   `json:"fuel_bucket"`
	FuelMinHours     *int     `json:"fuel_min_hours"`
	FuelMaxHours     *int     `json:"fuel_max_hours"`
	SystemIDs        []int64  `json:"system_ids"`
	SecurityBands    []string `json:"security_bands"`
	SecurityMin      *float64 `json:"security_min"`
	SecurityMax      *float64 `json:"security_max"`
	TypeIDs          []int64  `json:"type_ids"`
	ServiceNames     []string `json:"service_names"`
	ServiceMatchMode string   `json:"service_match_mode"`
	TimerBucket      string   `json:"timer_bucket"`
	TimerStart       string   `json:"timer_start"`
	TimerEnd         string   `json:"timer_end"`
	SortBy           string   `json:"sort_by"`
	SortOrder        string   `json:"sort_order"`
}

type CorporationStructureListResponse struct {
	Items    []CorporationStructureRow `json:"items"`
	Total    int                       `json:"total"`
	Page     int                       `json:"page"`
	PageSize int                       `json:"page_size"`
}

type DirectorCharacterOption struct {
	UserID        uint   `json:"user_id"`
	CharacterID   int64  `json:"character_id"`
	CharacterName string `json:"character_name"`
}

type ManageCorporationOption struct {
	CorporationID         int64                     `json:"corporation_id"`
	CorporationName       string                    `json:"corporation_name"`
	AuthorizedCharacterID int64                     `json:"authorized_character_id"`
	DirectorCharacters    []DirectorCharacterOption `json:"director_characters"`
}

type CorporationStructuresSettingsResponse struct {
	Corporations             []ManageCorporationOption `json:"corporations"`
	FuelNoticeThresholdDays  int                       `json:"fuel_notice_threshold_days"`
	TimerNoticeThresholdDays int                       `json:"timer_notice_threshold_days"`
	AlertEnabled             bool                      `json:"alert_enabled"`
	AlertGroupIDs            []int64                   `json:"alert_group_ids"`
}

type CorporationStructureAuthorizationBinding struct {
	CorporationID int64 `json:"corporation_id"`
	CharacterID   int64 `json:"character_id"`
}

type CorporationStructureAuthorizationUpdate struct {
	Authorizations           []CorporationStructureAuthorizationBinding `json:"authorizations"`
	FuelNoticeThresholdDays  *int                                       `json:"fuel_notice_threshold_days"`
	TimerNoticeThresholdDays *int                                       `json:"timer_notice_threshold_days"`
	AlertEnabled             *bool                                      `json:"alert_enabled"`
	AlertGroupIDs            *[]int64                                   `json:"alert_group_ids"`
	OperatorUserID           uint                                       `json:"-"`
}

type CorporationStructureRunTaskRequest struct {
	CorporationID int64 `json:"corporation_id"`
}

type CorporationStructureRunTaskResponse struct {
	CorporationID int64  `json:"corporation_id"`
	Queued        bool   `json:"queued"`
	Running       bool   `json:"running"`
	Message       string `json:"message"`
}

type CorporationStructureAssignmentItem struct {
	CorporationID         int64   `json:"corporation_id"`
	CorporationName       string  `json:"corporation_name"`
	StructureID           int64   `json:"structure_id"`
	StructureName         string  `json:"structure_name"`
	SystemID              int64   `json:"system_id"`
	SystemName            string  `json:"system_name"`
	RegionID              int64   `json:"region_id"`
	RegionName            string  `json:"region_name"`
	Security              float64 `json:"security"`
	TypeID                int64   `json:"type_id"`
	TypeName              string  `json:"type_name"`
	AssignedUserID        uint    `json:"assigned_user_id"`
	AssignedCharacterID   int64   `json:"assigned_character_id"`
	AssignedCharacterName string  `json:"assigned_character_name"`
}

type CorporationStructureAssignmentListResponse struct {
	Items        []CorporationStructureAssignmentItem `json:"items"`
	FuelOfficers []repository.FuelOfficerUserOption   `json:"fuel_officers"`
}

type CorporationStructureAssignmentBinding struct {
	CorporationID int64 `json:"corporation_id"`
	StructureID   int64 `json:"structure_id"`
	UserID        uint  `json:"user_id"`
}

type CorporationStructureAssignmentUpdateRequest struct {
	Assignments    []CorporationStructureAssignmentBinding `json:"assignments"`
	OperatorUserID uint                                    `json:"-"`
}

type FuelSalarySettingsResponse struct {
	SalaryPerStructureMonthly int `json:"salary_per_structure_monthly"`
}

type FuelSalarySettingsUpdateRequest struct {
	SalaryPerStructureMonthly int  `json:"salary_per_structure_monthly"`
	OperatorUserID            uint `json:"-"`
}

type FuelSalaryPayoutRunRequest struct {
	SettlementMonth string `json:"settlement_month"`
	OperatorUserID  uint   `json:"-"`
}

type FuelSalaryPayoutRunItem struct {
	UserID        uint   `json:"user_id"`
	AssignedCount int    `json:"assigned_count"`
	UnitSalary    int    `json:"unit_salary"`
	Amount        int    `json:"amount"`
	WalletRefID   string `json:"wallet_ref_id"`
}

type FuelSalaryPayoutRunResponse struct {
	SettlementMonth string                    `json:"settlement_month"`
	Items           []FuelSalaryPayoutRunItem `json:"items"`
}

type CorporationStructureFilterOptionsRequest struct {
	CorporationID int64 `json:"corporation_id" form:"corporation_id"`
}

type CorporationStructureSystemOption struct {
	SystemID   int64   `json:"system_id"`
	SystemName string  `json:"system_name"`
	RegionID   int64   `json:"region_id"`
	RegionName string  `json:"region_name"`
	Security   float64 `json:"security"`
}

type CorporationStructureTypeOption struct {
	TypeID   int64  `json:"type_id"`
	TypeName string `json:"type_name"`
}

type CorporationStructureServiceOption struct {
	Name string `json:"name"`
}

type CorporationStructureFilterOptionsResponse struct {
	Systems  []CorporationStructureSystemOption  `json:"systems"`
	Types    []CorporationStructureTypeOption    `json:"types"`
	Services []CorporationStructureServiceOption `json:"services"`
}

type corpManageContext struct {
	corporationIDs []int64
	corpNameByID   map[int64]string
	directorByCorp map[int64][]repository.DirectorCharacterOption
}

type corporationStructureSystemMeta struct {
	SystemName string
	Security   float64
	RegionID   int64
	RegionName string
}

func (s *CorporationStructureService) GetSettings(ctx context.Context) (*CorporationStructuresSettingsResponse, error) {
	manageCtx, err := s.buildManageContext(ctx, true)
	if err != nil {
		return nil, err
	}
	authMap := s.loadAuthorizationMap()
	thresholds := s.loadNoticeThresholdSettings()
	alertEnabled := s.loadAlertEnabled()
	alertGroupIDs, err := s.loadAlertGroupIDs()
	if err != nil {
		return nil, err
	}

	corporations := make([]ManageCorporationOption, 0, len(manageCtx.corporationIDs))
	for _, corpID := range manageCtx.corporationIDs {
		directors := manageCtx.directorByCorp[corpID]
		directorOptions := make([]DirectorCharacterOption, 0, len(directors))
		allowedCharacters := make(map[int64]struct{}, len(directors))
		for _, director := range directors {
			directorOptions = append(directorOptions, DirectorCharacterOption{
				UserID:        director.UserID,
				CharacterID:   director.CharacterID,
				CharacterName: director.CharacterName,
			})
			allowedCharacters[director.CharacterID] = struct{}{}
		}
		sort.Slice(directorOptions, func(i, j int) bool {
			if directorOptions[i].CharacterName != directorOptions[j].CharacterName {
				return directorOptions[i].CharacterName < directorOptions[j].CharacterName
			}
			return directorOptions[i].CharacterID < directorOptions[j].CharacterID
		})

		authorizedCharacterID := authMap[corpID]
		if _, ok := allowedCharacters[authorizedCharacterID]; !ok {
			authorizedCharacterID = 0
		}

		corporations = append(corporations, ManageCorporationOption{
			CorporationID:         corpID,
			CorporationName:       manageCtx.corpNameByID[corpID],
			AuthorizedCharacterID: authorizedCharacterID,
			DirectorCharacters:    directorOptions,
		})
	}

	return &CorporationStructuresSettingsResponse{
		Corporations:             corporations,
		FuelNoticeThresholdDays:  thresholds.FuelNoticeThresholdDays,
		TimerNoticeThresholdDays: thresholds.TimerNoticeThresholdDays,
		AlertEnabled:             alertEnabled,
		AlertGroupIDs:            alertGroupIDs,
	}, nil
}

func (s *CorporationStructureService) UpdateAuthorizations(
	ctx context.Context,
	req CorporationStructureAuthorizationUpdate,
) error {
	alertEnabled := s.loadAlertEnabled()
	if req.AlertEnabled != nil {
		alertEnabled = *req.AlertEnabled
	}
	alertGroupIDs, err := s.loadAlertGroupIDs()
	if err != nil {
		return err
	}
	if req.AlertGroupIDs != nil {
		alertGroupIDs, err = normalizeCorporationStructureAlertGroupIDs(*req.AlertGroupIDs)
		if err != nil {
			return err
		}
	}
	if alertEnabled && len(alertGroupIDs) == 0 {
		return errors.New("启用军团建筑 QQ 预警时至少配置一个 QQ 群号")
	}
	manageCtx, err := s.buildManageContext(ctx, false)
	if err != nil {
		return err
	}
	managedCorps := make(map[int64]struct{}, len(manageCtx.corporationIDs))
	for _, corpID := range manageCtx.corporationIDs {
		managedCorps[corpID] = struct{}{}
	}

	directorSetByCorp := make(map[int64]map[int64]struct{}, len(manageCtx.directorByCorp))
	for corpID, directors := range manageCtx.directorByCorp {
		charSet := make(map[int64]struct{}, len(directors))
		for _, director := range directors {
			charSet[director.CharacterID] = struct{}{}
		}
		directorSetByCorp[corpID] = charSet
	}

	currentMap := s.loadAuthorizationMap()
	nextMap := make(map[int64]int64, len(currentMap))
	for corpID, charID := range currentMap {
		nextMap[corpID] = charID
	}

	if err := validateAuthorizationBindings(req.Authorizations, managedCorps, directorSetByCorp); err != nil {
		return err
	}
	for _, binding := range req.Authorizations {
		if binding.CharacterID == 0 {
			delete(nextMap, binding.CorporationID)
		} else {
			nextMap[binding.CorporationID] = binding.CharacterID
		}
	}

	if err := s.saveAuthorizationMap(nextMap); err != nil {
		return err
	}

	validCorporationIDs := make([]int64, 0, len(nextMap))
	for _, corpID := range manageCtx.corporationIDs {
		if charID, ok := nextMap[corpID]; ok && charID > 0 {
			validCorporationIDs = append(validCorporationIDs, corpID)
		}
	}

	deletedSnapshotRows, err := s.repo.DeleteCorpStructuresNotInCorporationIDs(validCorporationIDs)
	if err != nil {
		return errors.New("删除军团建筑快照失败")
	}

	thresholds := s.loadNoticeThresholdSettings()
	if req.FuelNoticeThresholdDays != nil {
		thresholds.FuelNoticeThresholdDays = *req.FuelNoticeThresholdDays
	}
	if req.TimerNoticeThresholdDays != nil {
		thresholds.TimerNoticeThresholdDays = *req.TimerNoticeThresholdDays
	}
	if err := s.saveNoticeThresholdSettings(thresholds); err != nil {
		return err
	}
	if req.AlertGroupIDs != nil {
		if err := s.saveAlertGroupIDs(alertGroupIDs); err != nil {
			return err
		}
	}
	if req.AlertEnabled != nil {
		if err := s.saveAlertEnabled(alertEnabled); err != nil {
			return err
		}
	}

	if s.auditSvc != nil {
		_ = s.auditSvc.RecordEvent(ctx, AuditRecordInput{
			Category:     "config",
			Action:       "corp_structure_authorization_update",
			ActorUserID:  req.OperatorUserID,
			ResourceType: "system_config",
			ResourceID:   model.SysConfigDashboardCorpStructuresAuth,
			Result:       model.AuditResultSuccess,
			Details: map[string]any{
				"authorizations_count":        len(req.Authorizations),
				"valid_corporations_count":    len(validCorporationIDs),
				"deleted_snapshot_rows":       deletedSnapshotRows,
				"fuel_notice_threshold_days":  thresholds.FuelNoticeThresholdDays,
				"timer_notice_threshold_days": thresholds.TimerNoticeThresholdDays,
				"alert_enabled_updated":       req.AlertEnabled != nil,
				"alert_group_ids_updated":     req.AlertGroupIDs != nil,
			},
		})
	}
	return nil
}

func (s *CorporationStructureService) CountAttentionStructures(ctx context.Context) (int64, error) {
	manageCtx, err := s.buildManageContext(ctx, false)
	if err != nil {
		return 0, err
	}

	thresholds := s.loadNoticeThresholdSettings()
	if thresholds.FuelNoticeThresholdDays <= 0 && thresholds.TimerNoticeThresholdDays <= 0 {
		return 0, nil
	}

	structures, err := s.repo.ListCorpStructures(manageCtx.corporationIDs)
	if err != nil {
		return 0, errors.New("查询军团建筑提醒数据失败")
	}
	if len(structures) == 0 {
		return 0, nil
	}

	now := time.Now()
	fuelThresholdHours := thresholds.FuelNoticeThresholdDays * hoursPerDay
	timerDeadline := now.Add(time.Duration(thresholds.TimerNoticeThresholdDays*hoursPerDay) * time.Hour)
	attentionStructures := make(map[string]struct{}, len(structures))

	for _, st := range structures {
		matched := false
		if thresholds.FuelNoticeThresholdDays > 0 {
			remainingHours, _ := calculateFuelRemaining(st.FuelExpires, now)
			if remainingHours != nil && *remainingHours <= fuelThresholdHours {
				matched = true
			}
		}

		if !matched && thresholds.TimerNoticeThresholdDays > 0 {
			timerEnd, ok := parseFlexibleTime(st.StateTimerEnd)
			if ok && !timerEnd.Before(now) && !timerEnd.After(timerDeadline) {
				matched = true
			}
		}

		if matched {
			key := fmt.Sprintf("%d:%d", st.CorporationID, st.StructureID)
			attentionStructures[key] = struct{}{}
		}
	}

	return int64(len(attentionStructures)), nil
}

type corporationStructureAlertCandidate struct {
	key             repository.CorporationStructureAlertStateKey
	corporationName string
	structureName   string
	deadline        time.Time
	remaining       string
}

// RunAlertScan checks only persisted corporation-structure snapshots. It never
// triggers an ESI refresh, so the existing ESI queue remains the sole owner of
// snapshot freshness and API consumption.
func (s *CorporationStructureService) RunAlertScan(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if !s.loadAlertEnabled() {
		_, err := s.repo.ReconcileAlertStates(nil, time.Now())
		return err
	}
	groupIDs, err := s.loadAlertGroupIDs()
	if err != nil {
		return err
	}
	if len(groupIDs) == 0 {
		return errors.New("军团建筑 QQ 预警已启用但未配置目标群号")
	}

	thresholds := s.loadNoticeThresholdSettings()
	authorizations := s.loadAuthorizationMap()
	corporationIDs := make([]int64, 0, len(authorizations))
	for corporationID := range authorizations {
		corporationIDs = append(corporationIDs, corporationID)
	}
	sort.Slice(corporationIDs, func(i, j int) bool { return corporationIDs[i] < corporationIDs[j] })
	structures, err := s.repo.ListCorpStructures(corporationIDs)
	if err != nil {
		return errors.New("查询军团建筑预警快照失败")
	}

	now := time.Now()
	candidates := make(map[repository.CorporationStructureAlertStateKey]corporationStructureAlertCandidate)
	fuelThresholdHours := thresholds.FuelNoticeThresholdDays * hoursPerDay
	timerDeadline := now.Add(time.Duration(thresholds.TimerNoticeThresholdDays*hoursPerDay) * time.Hour)
	for _, structure := range structures {
		if err := ctx.Err(); err != nil {
			return err
		}
		corporationName := structure.CorporationName
		if corporationName == "" {
			corporationName = fmt.Sprintf("Corporation-%d", structure.CorporationID)
		}
		structureName := structure.Name
		if structureName == "" {
			structureName = fmt.Sprintf("Structure-%d", structure.StructureID)
		}
		if thresholds.FuelNoticeThresholdDays > 0 {
			remainingHours, remaining := calculateFuelRemaining(structure.FuelExpires, now)
			if remainingHours != nil && *remainingHours <= fuelThresholdHours {
				deadline, ok := parseFlexibleTime(structure.FuelExpires)
				if ok {
					key := repository.CorporationStructureAlertStateKey{CorporationID: structure.CorporationID, StructureID: structure.StructureID, AlertType: model.CorpStructureAlertFuelExpiring}
					candidates[key] = corporationStructureAlertCandidate{key: key, corporationName: corporationName, structureName: structureName, deadline: deadline, remaining: remaining}
				}
			}
		}
		if thresholds.TimerNoticeThresholdDays > 0 {
			timerEnd, ok := parseFlexibleTime(structure.StateTimerEnd)
			if ok && !timerEnd.Before(now) && !timerEnd.After(timerDeadline) {
				key := repository.CorporationStructureAlertStateKey{CorporationID: structure.CorporationID, StructureID: structure.StructureID, AlertType: model.CorpStructureAlertReinforceEnding}
				candidates[key] = corporationStructureAlertCandidate{key: key, corporationName: corporationName, structureName: structureName, deadline: timerEnd, remaining: formatAlertRemaining(timerEnd, now)}
			}
		}
	}
	keys := make([]repository.CorporationStructureAlertStateKey, 0, len(candidates))
	for key := range candidates {
		keys = append(keys, key)
	}
	pending, err := s.repo.ReconcileAlertStates(keys, now)
	if err != nil || len(pending) == 0 {
		return err
	}
	if s.alertNotifier == nil {
		return errors.New("QQ 建筑预警通知服务不可用")
	}

	byType := make(map[string][]corporationStructureAlertCandidate, 2)
	keysByType := make(map[string][]repository.CorporationStructureAlertStateKey, 2)
	for _, key := range pending {
		candidate, ok := candidates[key]
		if !ok {
			continue
		}
		byType[key.AlertType] = append(byType[key.AlertType], candidate)
		keysByType[key.AlertType] = append(keysByType[key.AlertType], key)
	}
	for _, alertType := range []string{model.CorpStructureAlertFuelExpiring, model.CorpStructureAlertReinforceEnding} {
		items := byType[alertType]
		if len(items) == 0 {
			continue
		}
		message := formatCorporationStructureAlertMessage(alertType, items)
		if err := s.alertNotifier.EnqueueStructureAlertNotifications(groupIDs, message); err != nil {
			return fmt.Errorf("入队军团建筑 QQ 预警: %w", err)
		}
		if err := s.repo.MarkAlertStatesDelivered(keysByType[alertType]); err != nil {
			return fmt.Errorf("更新军团建筑预警投递状态: %w", err)
		}
	}
	return nil
}

func formatAlertRemaining(deadline, now time.Time) string {
	diff := deadline.Sub(now)
	if diff <= 0 {
		return "已结束"
	}
	hours := int(math.Ceil(diff.Hours()))
	if hours >= hoursPerDay {
		return fmt.Sprintf("%dd %dh", hours/hoursPerDay, hours%hoursPerDay)
	}
	return fmt.Sprintf("%dh", hours)
}

func formatCorporationStructureAlertMessage(alertType string, items []corporationStructureAlertCandidate) string {
	sort.Slice(items, func(i, j int) bool {
		if !items[i].deadline.Equal(items[j].deadline) {
			return items[i].deadline.Before(items[j].deadline)
		}
		if items[i].corporationName != items[j].corporationName {
			return items[i].corporationName < items[j].corporationName
		}
		if items[i].structureName != items[j].structureName {
			return items[i].structureName < items[j].structureName
		}
		return items[i].key.StructureID < items[j].key.StructureID
	})
	title := "⚠️ 军团建筑燃料预警"
	if alertType == model.CorpStructureAlertReinforceEnding {
		title = "⚠️ 军团建筑增强状态结束预警"
	}
	lines := make([]string, 0, len(items)+1)
	lines = append(lines, title)
	for _, item := range items {
		lines = append(lines, fmt.Sprintf("- %s / %s：剩余 %s，结束于 %s UTC", item.corporationName, item.structureName, item.remaining, item.deadline.UTC().Format("2006-01-02 15:04")))
	}
	return strings.Join(lines, "\n")
}

func (s *CorporationStructureService) ListStructures(
	ctx context.Context,
	req CorporationStructureListRequest,
) (*CorporationStructureListResponse, error) {
	manageCtx, err := s.buildManageContext(ctx, false)
	if err != nil {
		return nil, err
	}
	targetCorps, err := resolveTargetCorporations(manageCtx.corporationIDs, req.CorporationID)
	if err != nil {
		return nil, err
	}

	structures, err := s.repo.ListCorpStructures(targetCorps)
	if err != nil {
		return nil, errors.New("查询建筑快照失败")
	}
	if len(structures) == 0 {
		page, pageSize := normalizePagination(req.Page, req.PageSize)
		return &CorporationStructureListResponse{
			Items:    []CorporationStructureRow{},
			Total:    0,
			Page:     page,
			PageSize: pageSize,
		}, nil
	}

	assignments, err := s.repo.ListAssignmentsByCorporations(targetCorps)
	if err != nil {
		return nil, errors.New("查询建筑指派配置失败")
	}
	fuelOfficers, err := s.repo.ListFuelOfficerUsersByCorporations(targetCorps)
	if err != nil {
		logCorporationStructuresWarn(
			"[CorporationStructures] 查询燃料官列表失败",
			err,
			zap.Int64s("corporation_ids", targetCorps),
			zap.Int64("filter_corporation_id", req.CorporationID),
		)
		return nil, errors.New("查询燃料官列表失败")
	}
	assignByStructure := make(map[int64]model.CorpStructureAssignment, len(assignments))
	for _, a := range assignments {
		assignByStructure[a.StructureID] = a
	}
	nameByUserID := make(map[uint]string, len(fuelOfficers))
	for _, o := range fuelOfficers {
		nameByUserID[o.UserID] = o.DisplayName
	}

	systemMeta := s.loadSystemMetaMap(collectSystemIDs(structures))
	now := time.Now()
	items := buildCorporationStructureRows(structures, now, systemMeta, assignByStructure, nameByUserID)
	applyFuelEstimates(items, now, s.loadServiceFuelRateMap(), s.loadStructureGroupIDMap(structures))

	filtered := filterCorporationStructureRows(items, req, now)
	sortCorporationStructureRows(filtered, req.SortBy, req.SortOrder)
	pageRows, total, page, pageSize := paginateCorporationStructureRows(filtered, req.Page, req.PageSize)

	return &CorporationStructureListResponse{
		Items:    pageRows,
		Total:    total,
		Page:     page,
		PageSize: pageSize,
	}, nil
}

func (s *CorporationStructureService) GetFilterOptions(
	ctx context.Context,
	req CorporationStructureFilterOptionsRequest,
) (*CorporationStructureFilterOptionsResponse, error) {
	manageCtx, err := s.buildManageContext(ctx, false)
	if err != nil {
		return nil, err
	}
	targetCorps, err := resolveTargetCorporations(manageCtx.corporationIDs, req.CorporationID)
	if err != nil {
		return nil, err
	}

	structures, err := s.repo.ListCorpStructures(targetCorps)
	if err != nil {
		return nil, errors.New("查询建筑筛选选项失败")
	}

	systemMeta := s.loadSystemMetaMap(collectSystemIDs(structures))
	systemByID := make(map[int64]CorporationStructureSystemOption)
	typeByID := make(map[int64]CorporationStructureTypeOption)
	serviceSet := make(map[string]struct{})

	for _, st := range structures {
		meta := systemMeta[st.SystemID]
		systemName := fallbackSystemName(st.SystemID, st.SystemName, meta.SystemName)
		if st.SystemID > 0 {
			systemByID[st.SystemID] = CorporationStructureSystemOption{
				SystemID:   st.SystemID,
				SystemName: systemName,
				RegionID:   meta.RegionID,
				RegionName: meta.RegionName,
				Security:   chooseSecurity(st.Security, meta.Security, meta.SystemName != ""),
			}
		}

		if st.TypeID > 0 {
			typeName := st.TypeName
			if typeName == "" {
				typeName = fmt.Sprintf("Type-%d", st.TypeID)
			}
			typeByID[st.TypeID] = CorporationStructureTypeOption{
				TypeID:   st.TypeID,
				TypeName: typeName,
			}
		}

		for _, service := range convertStructureServices(st.Services) {
			name := strings.TrimSpace(service.Name)
			if name == "" {
				continue
			}
			serviceSet[name] = struct{}{}
		}
	}

	systems := make([]CorporationStructureSystemOption, 0, len(systemByID))
	for _, item := range systemByID {
		systems = append(systems, item)
	}
	sort.Slice(systems, func(i, j int) bool {
		if systems[i].SystemName != systems[j].SystemName {
			return strings.ToLower(systems[i].SystemName) < strings.ToLower(systems[j].SystemName)
		}
		return systems[i].SystemID < systems[j].SystemID
	})

	types := make([]CorporationStructureTypeOption, 0, len(typeByID))
	for _, item := range typeByID {
		types = append(types, item)
	}
	sort.Slice(types, func(i, j int) bool {
		if types[i].TypeName != types[j].TypeName {
			return strings.ToLower(types[i].TypeName) < strings.ToLower(types[j].TypeName)
		}
		return types[i].TypeID < types[j].TypeID
	})

	services := make([]CorporationStructureServiceOption, 0, len(serviceSet))
	for name := range serviceSet {
		services = append(services, CorporationStructureServiceOption{Name: name})
	}
	sort.Slice(services, func(i, j int) bool {
		return strings.ToLower(services[i].Name) < strings.ToLower(services[j].Name)
	})

	return &CorporationStructureFilterOptionsResponse{
		Systems:  systems,
		Types:    types,
		Services: services,
	}, nil
}

func (s *CorporationStructureService) GetAssignments(
	ctx context.Context,
	req CorporationStructureFilterOptionsRequest,
) (*CorporationStructureAssignmentListResponse, error) {
	manageCtx, err := s.buildManageContext(ctx, false)
	if err != nil {
		return nil, err
	}
	targetCorps, err := resolveTargetCorporations(manageCtx.corporationIDs, req.CorporationID)
	if err != nil {
		return nil, err
	}

	structures, err := s.repo.ListCorpStructures(targetCorps)
	if err != nil {
		return nil, errors.New("查询建筑指派列表失败")
	}
	assignments, err := s.repo.ListAssignmentsByCorporations(targetCorps)
	if err != nil {
		return nil, errors.New("查询建筑指派配置失败")
	}
	fuelOfficers, err := s.repo.ListFuelOfficerUsersByCorporations(targetCorps)
	if err != nil {
		logCorporationStructuresWarn(
			"[CorporationStructures] 查询燃料官列表失败",
			err,
			zap.Int64s("corporation_ids", targetCorps),
			zap.Int64("filter_corporation_id", req.CorporationID),
		)
		return nil, errors.New("查询燃料官列表失败")
	}

	assignByStructure := make(map[int64]model.CorpStructureAssignment, len(assignments))
	for _, a := range assignments {
		assignByStructure[a.StructureID] = a
	}
	nameByUserID := make(map[uint]string, len(fuelOfficers))
	for _, o := range fuelOfficers {
		nameByUserID[o.UserID] = o.DisplayName
	}

	systemMeta := s.loadSystemMetaMap(collectSystemIDs(structures))
	now := time.Now()
	items := make([]CorporationStructureAssignmentItem, 0, len(structures))
	for _, st := range structures {
		row := buildCorporationStructureRow(st, now, systemMeta[st.SystemID])
		item := CorporationStructureAssignmentItem{
			CorporationID:   row.CorporationID,
			CorporationName: row.CorporationName,
			StructureID:     row.StructureID,
			StructureName:   row.Name,
			SystemID:        row.SystemID,
			SystemName:      row.SystemName,
			RegionID:        row.RegionID,
			RegionName:      row.RegionName,
			Security:        row.Security,
			TypeID:          row.TypeID,
			TypeName:        row.TypeName,
		}
		if assignment, ok := assignByStructure[st.StructureID]; ok {
			item.AssignedUserID = assignment.AssignedUserID
			item.AssignedCharacterID = assignment.AssignedCharacterID
			item.AssignedCharacterName = nameByUserID[assignment.AssignedUserID]
		}
		items = append(items, item)
	}

	return &CorporationStructureAssignmentListResponse{
		Items:        items,
		FuelOfficers: fuelOfficers,
	}, nil
}

func (s *CorporationStructureService) UpdateAssignments(
	ctx context.Context,
	req CorporationStructureAssignmentUpdateRequest,
) error {
	manageCtx, err := s.buildManageContext(ctx, false)
	if err != nil {
		return err
	}
	managedCorps := toInt64Set(manageCtx.corporationIDs)
	structures, err := s.repo.ListCorpStructures(manageCtx.corporationIDs)
	if err != nil {
		return errors.New("读取军团建筑失败")
	}
	structureByID := make(map[int64]model.CorpStructureInfo, len(structures))
	for _, st := range structures {
		structureByID[st.StructureID] = st
	}

	fuelOfficers, err := s.repo.ListFuelOfficerUsersByCorporations(manageCtx.corporationIDs)
	if err != nil {
		logCorporationStructuresWarn(
			"[CorporationStructures] 读取燃料官列表失败",
			err,
			zap.Int64s("corporation_ids", manageCtx.corporationIDs),
			zap.Uint("operator_user_id", req.OperatorUserID),
		)
		return errors.New("读取燃料官列表失败")
	}
	officerByUserID := make(map[uint]repository.FuelOfficerUserOption, len(fuelOfficers))
	for _, o := range fuelOfficers {
		officerByUserID[o.UserID] = o
	}

	next := make([]model.CorpStructureAssignment, 0, len(req.Assignments))
	seenStructures := make(map[int64]struct{}, len(req.Assignments))
	deleteIDs := make([]int64, 0)
	for _, item := range req.Assignments {
		if item.CorporationID <= 0 || item.StructureID <= 0 {
			return errors.New("corporation_id 和 structure_id 必须为正整数")
		}
		if _, ok := managedCorps[item.CorporationID]; !ok {
			return fmt.Errorf("军团 %d 不在可管理范围内", item.CorporationID)
		}
		st, ok := structureByID[item.StructureID]
		if !ok || st.CorporationID != item.CorporationID {
			return fmt.Errorf("建筑 %d 不在军团 %d 下", item.StructureID, item.CorporationID)
		}
		if _, duplicated := seenStructures[item.StructureID]; duplicated {
			return fmt.Errorf("建筑 %d 的指派重复", item.StructureID)
		}
		seenStructures[item.StructureID] = struct{}{}
		if item.UserID == 0 {
			deleteIDs = append(deleteIDs, item.StructureID)
			continue
		}
		officer, ok := officerByUserID[item.UserID]
		if !ok {
			return fmt.Errorf("用户 %d 不是可选燃料官", item.UserID)
		}
		next = append(next, model.CorpStructureAssignment{
			CorporationID:       item.CorporationID,
			StructureID:         item.StructureID,
			AssignedUserID:      officer.UserID,
			AssignedCharacterID: officer.CharacterID,
		})
	}
	if err := s.repo.UpsertAssignments(next); err != nil {
		return errors.New("保存建筑指派失败")
	}
	if err := s.repo.DeleteAssignmentsByStructureIDs(deleteIDs); err != nil {
		return errors.New("清理建筑指派失败")
	}
	return nil
}

func (s *CorporationStructureService) GetFuelSalarySettings() (*FuelSalarySettingsResponse, error) {
	value := s.sysConfigRepo.GetInt(
		model.SysConfigFuelSalaryPerStructureMonthly,
		model.SysConfigDefaultFuelSalaryPerStructureMonthly,
	)
	if value < 0 {
		value = 0
	}
	return &FuelSalarySettingsResponse{SalaryPerStructureMonthly: value}, nil
}

func (s *CorporationStructureService) UpdateFuelSalarySettings(req FuelSalarySettingsUpdateRequest) error {
	if req.SalaryPerStructureMonthly < 0 {
		return errors.New("每建筑月工资不能小于 0")
	}
	return s.sysConfigRepo.Set(
		model.SysConfigFuelSalaryPerStructureMonthly,
		strconv.Itoa(req.SalaryPerStructureMonthly),
		"燃料官每建筑每月工资",
	)
}

func (s *CorporationStructureService) RunFuelSalaryPayout(
	ctx context.Context,
	req FuelSalaryPayoutRunRequest,
) (*FuelSalaryPayoutRunResponse, error) {
	if _, err := time.Parse("2006-01", req.SettlementMonth); err != nil {
		return nil, errors.New("settlement_month 格式必须为 YYYY-MM")
	}
	unitSalary := s.sysConfigRepo.GetInt(
		model.SysConfigFuelSalaryPerStructureMonthly,
		model.SysConfigDefaultFuelSalaryPerStructureMonthly,
	)
	if unitSalary < 0 {
		unitSalary = 0
	}

	assignments, err := s.repo.ListAssignmentsByCorporations(s.buildAllManagedCorpsForFuel())
	if err != nil {
		return nil, errors.New("查询燃料官指派失败")
	}
	countByUserID := make(map[uint]int)
	for _, a := range assignments {
		if a.AssignedUserID > 0 {
			countByUserID[a.AssignedUserID]++
		}
	}

	walletSvc := NewSysWalletService()
	items := make([]FuelSalaryPayoutRunItem, 0, len(countByUserID))
	for userID, assignedCount := range countByUserID {
		if assignedCount <= 0 {
			continue
		}
		exists, err := s.repo.ExistsFuelSalaryPayout(req.SettlementMonth, userID)
		if err != nil {
			return nil, errors.New("检查工资发放记录失败")
		}
		if exists {
			continue
		}
		amount := unitSalary * assignedCount
		if amount <= 0 {
			continue
		}
		walletRefID := fmt.Sprintf("fuel_salary:%s:%d", req.SettlementMonth, userID)
		err = global.DB.Transaction(func(tx *gorm.DB) error {
			if err := walletSvc.ApplyWalletDeltaByOperatorTx(
				tx,
				userID,
				req.OperatorUserID,
				float64(amount),
				fmt.Sprintf("燃料官 %s 月工资", req.SettlementMonth),
				model.WalletRefFuelSalary,
				walletRefID,
			); err != nil {
				return err
			}
			return s.repo.CreateFuelSalaryPayoutTx(tx, &model.FuelSalaryPayout{
				SettlementMonth: req.SettlementMonth,
				UserID:          userID,
				AssignedCount:   assignedCount,
				UnitSalary:      unitSalary,
				Amount:          amount,
				WalletRefID:     walletRefID,
				OperatorUserID:  req.OperatorUserID,
			})
		})
		if err != nil {
			return nil, err
		}
		items = append(items, FuelSalaryPayoutRunItem{
			UserID:        userID,
			AssignedCount: assignedCount,
			UnitSalary:    unitSalary,
			Amount:        amount,
			WalletRefID:   walletRefID,
		})
	}
	return &FuelSalaryPayoutRunResponse{
		SettlementMonth: req.SettlementMonth,
		Items:           items,
	}, nil
}

func (s *CorporationStructureService) ListMyAssignedStructures(
	ctx context.Context,
	userID uint,
	req CorporationStructureListRequest,
) (*CorporationStructureListResponse, error) {
	assignments, err := s.repo.ListAssignmentsByUserID(userID)
	if err != nil {
		return nil, errors.New("查询我的建筑指派失败")
	}
	if len(assignments) == 0 {
		page, pageSize := normalizePagination(req.Page, req.PageSize)
		return &CorporationStructureListResponse{Items: []CorporationStructureRow{}, Total: 0, Page: page, PageSize: pageSize}, nil
	}
	structureIDs := make(map[int64]struct{}, len(assignments))
	corpSet := make(map[int64]struct{}, len(assignments))
	for _, a := range assignments {
		structureIDs[a.StructureID] = struct{}{}
		corpSet[a.CorporationID] = struct{}{}
	}
	corps := make([]int64, 0, len(corpSet))
	for c := range corpSet {
		corps = append(corps, c)
	}
	assignByStructure := make(map[int64]model.CorpStructureAssignment, len(assignments))
	for _, a := range assignments {
		assignByStructure[a.StructureID] = a
	}
	structures, err := s.repo.ListCorpStructures(corps)
	if err != nil {
		return nil, errors.New("查询建筑快照失败")
	}
	fuelOfficers, err := s.repo.ListFuelOfficerUsersByCorporations(corps)
	if err != nil {
		logCorporationStructuresWarn(
			"[CorporationStructures] 查询燃料官列表失败",
			err,
			zap.Int64s("corporation_ids", corps),
			zap.Uint("user_id", userID),
		)
		return nil, errors.New("查询燃料官列表失败")
	}
	nameByUserID := make(map[uint]string, len(fuelOfficers))
	for _, o := range fuelOfficers {
		nameByUserID[o.UserID] = o.DisplayName
	}
	systemMeta := s.loadSystemMetaMap(collectSystemIDs(structures))
	now := time.Now()
	rows := buildCorporationStructureRows(structures, now, systemMeta, assignByStructure, nameByUserID)
	applyFuelEstimates(rows, now, s.loadServiceFuelRateMap(), s.loadStructureGroupIDMap(structures))
	filteredRows := make([]CorporationStructureRow, 0, len(rows))
	for _, row := range rows {
		if _, ok := structureIDs[row.StructureID]; !ok {
			continue
		}
		filteredRows = append(filteredRows, row)
	}
	filtered := filterCorporationStructureRows(filteredRows, req, now)
	sortCorporationStructureRows(filtered, req.SortBy, req.SortOrder)
	pageRows, total, page, pageSize := paginateCorporationStructureRows(filtered, req.Page, req.PageSize)
	return &CorporationStructureListResponse{Items: pageRows, Total: total, Page: page, PageSize: pageSize}, nil
}

func (s *CorporationStructureService) buildAllManagedCorpsForFuel() []int64 {
	ctx := context.Background()
	manageCtx, err := s.buildManageContext(ctx, false)
	if err != nil {
		return []int64{}
	}
	return manageCtx.corporationIDs
}

func (s *CorporationStructureService) buildManageContext(
	ctx context.Context,
	resolveNames bool,
) (*corpManageContext, error) {
	adminIDs, err := s.roleRepo.GetRoleUserIDs(model.RoleAdmin)
	if err != nil {
		return nil, errors.New("读取管理员用户失败")
	}
	superAdminIDs, err := s.roleRepo.GetRoleUserIDs(model.RoleSuperAdmin)
	if err != nil {
		return nil, errors.New("读取超级管理员用户失败")
	}

	userIDSet := make(map[uint]struct{}, len(adminIDs)+len(superAdminIDs))
	for _, uid := range adminIDs {
		userIDSet[uid] = struct{}{}
	}
	for _, uid := range superAdminIDs {
		userIDSet[uid] = struct{}{}
	}
	userIDs := make([]uint, 0, len(userIDSet))
	for uid := range userIDSet {
		userIDs = append(userIDs, uid)
	}

	chars, err := s.charRepo.ListByUserIDs(userIDs)
	if err != nil {
		return nil, errors.New("读取管理员角色失败")
	}

	corporationIDs := deduplicateManagedCorporationIDs(chars, utils.GetAllowCorporations())

	directors, err := s.repo.ListDirectorCharactersByCorporations(corporationIDs)
	if err != nil {
		return nil, errors.New("读取 Director 授权角色失败")
	}
	directorByCorp := make(map[int64][]repository.DirectorCharacterOption, len(corporationIDs))
	for _, director := range directors {
		directorByCorp[director.CorporationID] = append(directorByCorp[director.CorporationID], director)
	}

	corpNameByID := make(map[int64]string, len(corporationIDs))
	for _, corpID := range corporationIDs {
		corpNameByID[corpID] = fmt.Sprintf("Corporation-%d", corpID)
	}
	if resolveNames {
		corpNameByID = s.resolveCorporationNames(ctx, corporationIDs)
	}
	return &corpManageContext{
		corporationIDs: corporationIDs,
		corpNameByID:   corpNameByID,
		directorByCorp: directorByCorp,
	}, nil
}

func (s *CorporationStructureService) ResolveRefreshAuthorizationCharacter(
	ctx context.Context,
	corporationID int64,
) (int64, error) {
	if corporationID <= 0 {
		return 0, errors.New("corporation_id 必须为正整数")
	}

	manageCtx, err := s.buildManageContext(ctx, false)
	if err != nil {
		return 0, err
	}
	managed := false
	for _, corpID := range manageCtx.corporationIDs {
		if corpID == corporationID {
			managed = true
			break
		}
	}
	if !managed {
		return 0, errors.New("无权刷新该军团建筑数据")
	}

	authMap := s.loadAuthorizationMap()
	characterID := authMap[corporationID]
	if characterID == 0 {
		return 0, errors.New("该军团未配置 Director 授权角色")
	}

	allowedCharacter := false
	for _, option := range manageCtx.directorByCorp[corporationID] {
		if option.CharacterID == characterID {
			allowedCharacter = true
			break
		}
	}
	if !allowedCharacter {
		return 0, errors.New("已配置角色不再具备 Director 权限，请重新配置")
	}
	return characterID, nil
}

func (s *CorporationStructureService) resolveCorporationNames(
	ctx context.Context,
	corporationIDs []int64,
) map[int64]string {
	names := make(map[int64]string, len(corporationIDs))
	for _, corpID := range corporationIDs {
		names[corpID] = fmt.Sprintf("Corporation-%d", corpID)
	}
	if len(corporationIDs) == 0 {
		return names
	}

	if s.nameResolver == nil {
		s.nameResolver = NewEntityNameResolver()
	}
	resolved := s.nameResolver.Resolve(ctx, corporationIDs)
	for id, name := range resolved.Names {
		names[id] = name
	}
	if len(resolved.Miss) > 0 {
		logCorporationStructuresWarn("[CorporationStructures] 部分军团名称解析失败", fmt.Errorf("misses=%d", len(resolved.Miss)))
	}
	return names
}

func (s *CorporationStructureService) loadAuthorizationMap() map[int64]int64 {
	raw := s.sysConfigRepo.GetString(model.SysConfigDashboardCorpStructuresAuth, "{}")
	parsed := make(map[string]int64)
	if err := json.Unmarshal([]byte(raw), &parsed); err != nil {
		return map[int64]int64{}
	}
	result := make(map[int64]int64, len(parsed))
	for corpID, characterID := range parsed {
		id, err := strconv.ParseInt(corpID, 10, 64)
		if err != nil || id <= 0 || characterID <= 0 {
			continue
		}
		result[id] = characterID
	}
	return result
}

func (s *CorporationStructureService) saveAuthorizationMap(authorizations map[int64]int64) error {
	payload := make(map[string]int64, len(authorizations))
	for corpID, characterID := range authorizations {
		if corpID <= 0 || characterID <= 0 {
			continue
		}
		payload[strconv.FormatInt(corpID, 10)] = characterID
	}
	data, err := json.Marshal(payload)
	if err != nil {
		return errors.New("序列化授权配置失败")
	}
	return s.sysConfigRepo.Set(
		model.SysConfigDashboardCorpStructuresAuth,
		string(data),
		"dashboard 军团建筑 director 授权角色映射",
	)
}

type corporationStructureNoticeThresholdSettings struct {
	FuelNoticeThresholdDays  int
	TimerNoticeThresholdDays int
}

func (s *CorporationStructureService) loadNoticeThresholdSettings() corporationStructureNoticeThresholdSettings {
	return corporationStructureNoticeThresholdSettings{
		FuelNoticeThresholdDays: normalizeNoticeThresholdDays(s.sysConfigRepo.GetInt(
			model.SysConfigDashboardCorpStructuresFuelNoticeThresholdDays,
			model.SysConfigDefaultDashboardCorpStructuresFuelNoticeThresholdDays,
		)),
		TimerNoticeThresholdDays: normalizeNoticeThresholdDays(s.sysConfigRepo.GetInt(
			model.SysConfigDashboardCorpStructuresTimerNoticeThresholdDays,
			model.SysConfigDefaultDashboardCorpStructuresTimerNoticeThresholdDays,
		)),
	}
}

func (s *CorporationStructureService) saveNoticeThresholdSettings(
	settings corporationStructureNoticeThresholdSettings,
) error {
	if settings.FuelNoticeThresholdDays < 0 {
		return errors.New("燃料剩余提醒阈值不能小于 0")
	}
	if settings.TimerNoticeThresholdDays < 0 {
		return errors.New("增强时间提醒阈值不能小于 0")
	}

	return s.sysConfigRepo.SetMany([]repository.SysConfigUpsertItem{
		{
			Key:   model.SysConfigDashboardCorpStructuresFuelNoticeThresholdDays,
			Value: strconv.Itoa(settings.FuelNoticeThresholdDays),
			Desc:  "dashboard 军团建筑提醒：燃料剩余阈值（天）",
		},
		{
			Key:   model.SysConfigDashboardCorpStructuresTimerNoticeThresholdDays,
			Value: strconv.Itoa(settings.TimerNoticeThresholdDays),
			Desc:  "dashboard 军团建筑提醒：增强时间阈值（天）",
		},
	})
}

func (s *CorporationStructureService) loadAlertGroupIDs() ([]int64, error) {
	raw := strings.TrimSpace(s.sysConfigRepo.GetString(model.SysConfigDashboardCorpStructuresAlertGroupIDs, "[]"))
	if raw == "" {
		return []int64{}, nil
	}
	var groupIDs []int64
	if err := json.Unmarshal([]byte(raw), &groupIDs); err != nil {
		return nil, errors.New("军团建筑 QQ 预警群号配置无效")
	}
	return normalizeCorporationStructureAlertGroupIDs(groupIDs)
}

func (s *CorporationStructureService) loadAlertEnabled() bool {
	return s.sysConfigRepo.GetBool(model.SysConfigDashboardCorpStructuresAlertEnabled, false)
}

func (s *CorporationStructureService) saveAlertEnabled(enabled bool) error {
	return s.sysConfigRepo.Set(
		model.SysConfigDashboardCorpStructuresAlertEnabled,
		strconv.FormatBool(enabled),
		"dashboard 军团建筑 QQ 预警开关",
	)
}

func (s *CorporationStructureService) saveAlertGroupIDs(groupIDs []int64) error {
	normalized, err := normalizeCorporationStructureAlertGroupIDs(groupIDs)
	if err != nil {
		return err
	}
	payload, err := json.Marshal(normalized)
	if err != nil {
		return errors.New("序列化军团建筑 QQ 预警群号失败")
	}
	return s.sysConfigRepo.Set(model.SysConfigDashboardCorpStructuresAlertGroupIDs, string(payload), "dashboard 军团建筑 QQ 预警目标群号数组")
}

func normalizeCorporationStructureAlertGroupIDs(groupIDs []int64) ([]int64, error) {
	seen := make(map[int64]struct{}, len(groupIDs))
	normalized := make([]int64, 0, len(groupIDs))
	for _, groupID := range groupIDs {
		if groupID <= 0 {
			return nil, errors.New("军团建筑 QQ 预警群号必须为正数")
		}
		if _, exists := seen[groupID]; exists {
			return nil, errors.New("军团建筑 QQ 预警群号不能重复")
		}
		seen[groupID] = struct{}{}
		normalized = append(normalized, groupID)
	}
	sort.Slice(normalized, func(i, j int) bool { return normalized[i] < normalized[j] })
	return normalized, nil
}

func normalizeNoticeThresholdDays(days int) int {
	if days < 0 {
		return 0
	}
	return days
}

func (s *CorporationStructureService) loadSystemMetaMap(systemIDs []int64) map[int64]corporationStructureSystemMeta {
	result := make(map[int64]corporationStructureSystemMeta, len(systemIDs))
	if len(systemIDs) == 0 {
		return result
	}

	rows := make([]model.MapSolarSystem, 0, len(systemIDs))
	if err := global.DB.
		Where(`"solarSystemID" IN ?`, systemIDs).
		Find(&rows).Error; err != nil {
		logCorporationStructuresWarn("[CorporationStructures] 读取星系信息失败", err)
		return result
	}

	regionIDSet := make(map[int]struct{}, len(rows))
	for _, row := range rows {
		regionIDSet[row.RegionID] = struct{}{}
	}
	regionIDs := make([]int, 0, len(regionIDSet))
	for id := range regionIDSet {
		regionIDs = append(regionIDs, id)
	}

	regionNameByID := make(map[int]string, len(regionIDs))
	if len(regionIDs) > 0 {
		regions := make([]model.MapRegion, 0, len(regionIDs))
		if err := global.DB.
			Where(`"regionID" IN ?`, regionIDs).
			Find(&regions).Error; err != nil {
			logCorporationStructuresWarn("[CorporationStructures] 读取区域信息失败", err)
		} else {
			for _, region := range regions {
				regionNameByID[region.RegionID] = region.RegionName
			}
		}
	}

	for _, row := range rows {
		result[int64(row.SolarSystemID)] = corporationStructureSystemMeta{
			SystemName: row.SolarSystemName,
			Security:   row.Security,
			RegionID:   int64(row.RegionID),
			RegionName: regionNameByID[row.RegionID],
		}
	}
	return result
}

func convertStructureServices(raw string) []CorporationStructureServiceInfo {
	services := repository.DecodeStructureServices(raw)
	result := make([]CorporationStructureServiceInfo, 0, len(services))
	for _, item := range services {
		result = append(result, CorporationStructureServiceInfo{
			Name:  item.Name,
			State: item.State,
		})
	}
	return result
}

func deduplicateManagedCorporationIDs(chars []model.EveCharacter, allowCorps []int64) []int64 {
	allowSet := make(map[int64]struct{}, len(allowCorps))
	for _, corpID := range allowCorps {
		if corpID > 0 {
			allowSet[corpID] = struct{}{}
		}
	}

	corpSet := make(map[int64]struct{})
	for _, char := range chars {
		if char.CorporationID <= 0 {
			continue
		}
		if _, ok := allowSet[char.CorporationID]; !ok {
			continue
		}
		corpSet[char.CorporationID] = struct{}{}
	}

	corpIDs := make([]int64, 0, len(corpSet))
	for corpID := range corpSet {
		corpIDs = append(corpIDs, corpID)
	}
	sort.Slice(corpIDs, func(i, j int) bool { return corpIDs[i] < corpIDs[j] })
	return corpIDs
}

func validateAuthorizationBindings(
	bindings []CorporationStructureAuthorizationBinding,
	managedCorps map[int64]struct{},
	directorSetByCorp map[int64]map[int64]struct{},
) error {
	seen := make(map[int64]struct{}, len(bindings))
	for _, binding := range bindings {
		if binding.CorporationID <= 0 {
			return errors.New("corporation_id 必须为正整数")
		}
		if _, exists := managedCorps[binding.CorporationID]; !exists {
			return fmt.Errorf("军团 %d 不在可管理范围内", binding.CorporationID)
		}
		if _, duplicated := seen[binding.CorporationID]; duplicated {
			return fmt.Errorf("军团 %d 的授权配置重复", binding.CorporationID)
		}
		seen[binding.CorporationID] = struct{}{}

		if binding.CharacterID == 0 {
			continue
		}

		directors := directorSetByCorp[binding.CorporationID]
		if _, ok := directors[binding.CharacterID]; !ok {
			return fmt.Errorf("角色 %d 不是军团 %d 的 Director 授权角色", binding.CharacterID, binding.CorporationID)
		}
	}
	return nil
}

func resolveTargetCorporations(managed []int64, corporationID int64) ([]int64, error) {
	if corporationID <= 0 {
		return managed, nil
	}
	for _, corpID := range managed {
		if corpID == corporationID {
			return []int64{corporationID}, nil
		}
	}
	return nil, errors.New("无权访问该军团建筑数据")
}

func collectSystemIDs(structures []model.CorpStructureInfo) []int64 {
	systemSet := make(map[int64]struct{}, len(structures))
	for _, st := range structures {
		if st.SystemID > 0 {
			systemSet[st.SystemID] = struct{}{}
		}
	}
	ids := make([]int64, 0, len(systemSet))
	for id := range systemSet {
		ids = append(ids, id)
	}
	return ids
}

func buildCorporationStructureRow(
	st model.CorpStructureInfo,
	now time.Time,
	meta corporationStructureSystemMeta,
) CorporationStructureRow {
	fuelRemainingHours, fuelRemaining := calculateFuelRemaining(st.FuelExpires, now)
	row := CorporationStructureRow{
		CorporationID:      st.CorporationID,
		CorporationName:    st.CorporationName,
		StructureID:        st.StructureID,
		Name:               st.Name,
		TypeID:             st.TypeID,
		TypeName:           st.TypeName,
		SystemID:           st.SystemID,
		SystemName:         fallbackSystemName(st.SystemID, st.SystemName, meta.SystemName),
		RegionID:           meta.RegionID,
		RegionName:         meta.RegionName,
		Security:           chooseSecurity(st.Security, meta.Security, meta.SystemName != ""),
		State:              st.State,
		Services:           convertStructureServices(st.Services),
		FuelExpires:        st.FuelExpires,
		FuelRemaining:      fuelRemaining,
		FuelRemainingHours: fuelRemainingHours,
		ReinforceHour:      st.ReinforceHour,
		StateTimerStart:    st.StateTimerStart,
		StateTimerEnd:      st.StateTimerEnd,
		UpdatedAt:          st.UpdateAt,
	}
	if row.Name == "" {
		row.Name = fmt.Sprintf("Structure-%d", st.StructureID)
	}
	if row.CorporationName == "" {
		row.CorporationName = fmt.Sprintf("Corporation-%d", st.CorporationID)
	}
	if row.TypeName == "" {
		row.TypeName = fmt.Sprintf("Type-%d", st.TypeID)
	}
	return row
}

func buildCorporationStructureRows(
	structures []model.CorpStructureInfo,
	now time.Time,
	systemMeta map[int64]corporationStructureSystemMeta,
	assignByStructure map[int64]model.CorpStructureAssignment,
	nameByUserID map[uint]string,
) []CorporationStructureRow {
	rows := make([]CorporationStructureRow, 0, len(structures))
	for _, st := range structures {
		row := buildCorporationStructureRow(st, now, systemMeta[st.SystemID])
		if assignByStructure != nil {
			if assignment, ok := assignByStructure[st.StructureID]; ok {
				applyCorporationStructureAssignment(&row, assignment, nameByUserID)
			}
		}
		rows = append(rows, row)
	}
	return rows
}

// applyFuelEstimates 填充每行的燃料消耗估算（每小时块数 + 到耗尽月月底需补充量）。
// 仅在需要展示这两列的列表（建筑列表 / 燃料官列表）调用；指派管理列表不调用。
//
//   - rateMap: service name(归一化) → 原始每小时块数（DB 覆盖默认表）
//   - groupMap: 建筑 typeID → groupID（来自 SDE，决定折扣系数；缺失按无折扣）
//
// 存在未配置（无法在 rate map 命中）的在线服务时，判定为「不完整估算」：
// 不返回部分燃料合计（fuel_per_hour / fuel_to_month_end 保持 nil），改为标记
// fuel_estimate_incomplete=true 并在 fuel_unknown_services 中列出未映射服务原始名。
func applyFuelEstimates(
	rows []CorporationStructureRow,
	now time.Time,
	rateMap map[string]float64,
	groupMap map[int64]int,
) {
	if len(rateMap) == 0 {
		return
	}
	for i := range rows {
		row := &rows[i]
		groupID := groupMap[row.TypeID]
		est := EstimateFuelPerHour(groupID, row.TypeID, row.Services, rateMap)
		if len(est.UnknownServices) > 0 {
			row.FuelEstimateIncomplete = true
			row.FuelUnknownServices = est.UnknownServices
			continue
		}
		if est.FuelPerHour > 0 {
			fuelPerHour := est.FuelPerHour
			row.FuelPerHour = &fuelPerHour
			row.FuelToMonthEnd = EstimateFuelToMonthEnd(row.FuelExpires, fuelPerHour, now)
		}
	}
}

func applyCorporationStructureAssignment(
	row *CorporationStructureRow,
	assignment model.CorpStructureAssignment,
	nameByUserID map[uint]string,
) {
	row.AssignedUserID = assignment.AssignedUserID
	row.AssignedCharacterID = assignment.AssignedCharacterID
	if nameByUserID != nil {
		row.AssignedCharacterName = nameByUserID[assignment.AssignedUserID]
	}
}

func fallbackSystemName(systemID int64, snapshotName string, sdeName string) string {
	if sdeName != "" {
		return sdeName
	}
	if snapshotName != "" {
		return snapshotName
	}
	if systemID > 0 {
		return fmt.Sprintf("System-%d", systemID)
	}
	return ""
}

func chooseSecurity(snapshotSecurity float64, sdeSecurity float64, hasSDE bool) float64 {
	if hasSDE {
		return sdeSecurity
	}
	return snapshotSecurity
}

func calculateFuelRemaining(fuelExpires string, now time.Time) (*int, string) {
	if strings.TrimSpace(fuelExpires) == "" {
		return nil, ""
	}
	ts, ok := parseFlexibleTime(fuelExpires)
	if !ok {
		return nil, ""
	}
	diff := ts.Sub(now)
	if diff <= 0 {
		expired := 0
		return &expired, "expired"
	}
	hours := int(math.Ceil(diff.Hours()))
	if hours < 0 {
		hours = 0
	}
	days := hours / 24
	leftHours := hours % 24
	if days > 0 {
		return &hours, fmt.Sprintf("%dd %dh", days, leftHours)
	}
	return &hours, fmt.Sprintf("%dh", leftHours)
}

func filterCorporationStructureRows(
	rows []CorporationStructureRow,
	req CorporationStructureListRequest,
	now time.Time,
) []CorporationStructureRow {
	stateSet := buildSelectedStateSet(req.StateGroups)
	systemSet := toInt64Set(req.SystemIDs)
	typeSet := toInt64Set(req.TypeIDs)
	serviceNames := normalizeLowerStringList(req.ServiceNames)
	keyword := strings.ToLower(strings.TrimSpace(req.Keyword))
	securityBandSet := toStringSet(req.SecurityBands)
	timerStart, hasTimerStart := parseTimeFilter(req.TimerStart)
	timerEnd, hasTimerEnd := parseTimeFilter(req.TimerEnd)
	matchMode := normalizeServiceMatchMode(req.ServiceMatchMode)
	fuelBucket := normalizeFuelBucket(req.FuelBucket)
	timerBucket := normalizeTimerBucket(req.TimerBucket)

	filtered := make([]CorporationStructureRow, 0, len(rows))
	for _, row := range rows {
		if keyword != "" {
			searchText := strings.ToLower(row.Name + " " + row.SystemName)
			if !strings.Contains(searchText, keyword) {
				continue
			}
		}
		if len(stateSet) > 0 {
			if _, ok := stateSet[row.State]; !ok {
				continue
			}
		}
		if len(systemSet) > 0 {
			if _, ok := systemSet[row.SystemID]; !ok {
				continue
			}
		}
		if len(typeSet) > 0 {
			if _, ok := typeSet[row.TypeID]; !ok {
				continue
			}
		}
		if !matchSecurityBands(row.Security, securityBandSet) {
			continue
		}
		if !matchSecurityRange(row.Security, req.SecurityMin, req.SecurityMax) {
			continue
		}
		if !matchFuelFilter(row.FuelRemainingHours, fuelBucket, req.FuelMinHours, req.FuelMaxHours) {
			continue
		}
		if !matchServices(row.Services, serviceNames, matchMode) {
			continue
		}
		if !matchTimerFilter(row.StateTimerEnd, timerBucket, now, timerStart, hasTimerStart, timerEnd, hasTimerEnd) {
			continue
		}
		filtered = append(filtered, row)
	}
	return filtered
}

func buildSelectedStateSet(stateGroups []string) map[string]struct{} {
	selected := make(map[string]struct{})
	for _, group := range stateGroups {
		groupStates, ok := corporationStructureStateGroupMap[group]
		if !ok {
			continue
		}
		for _, state := range groupStates {
			selected[state] = struct{}{}
		}
	}
	return selected
}

func matchSecurityBands(security float64, bands map[string]struct{}) bool {
	if len(bands) == 0 {
		return true
	}
	for band := range bands {
		switch band {
		case "highsec":
			if security >= 0.5 {
				return true
			}
		case "lowsec":
			if security >= 0 && security < 0.5 {
				return true
			}
		case "nullsec":
			if security < 0 {
				return true
			}
		}
	}
	return false
}

func matchSecurityRange(security float64, min *float64, max *float64) bool {
	if min != nil && security < *min {
		return false
	}
	if max != nil && security > *max {
		return false
	}
	return true
}

func matchFuelFilter(hours *int, bucket string, min *int, max *int) bool {
	if bucket == corporationStructureFuelBucketAll {
		return true
	}
	if hours == nil {
		return false
	}

	switch bucket {
	case corporationStructureFuelBucket24h:
		return *hours < 24
	case corporationStructureFuelBucket72h:
		return *hours < 72
	case corporationStructureFuelBucket168h:
		return *hours < 168
	case corporationStructureFuelBucketCustom:
		if min != nil && *hours < *min {
			return false
		}
		if max != nil && *hours > *max {
			return false
		}
		return true
	default:
		return true
	}
}

func matchServices(
	services []CorporationStructureServiceInfo,
	targets []string,
	matchMode string,
) bool {
	if len(targets) == 0 {
		return true
	}

	serviceSet := make(map[string]struct{}, len(services))
	for _, item := range services {
		name := strings.TrimSpace(strings.ToLower(item.Name))
		if name == "" {
			continue
		}
		serviceSet[name] = struct{}{}
	}
	if len(serviceSet) == 0 {
		return false
	}

	if matchMode == corporationStructureServiceMatchOr {
		for _, target := range targets {
			if _, ok := serviceSet[target]; ok {
				return true
			}
		}
		return false
	}

	for _, target := range targets {
		if _, ok := serviceSet[target]; !ok {
			return false
		}
	}
	return true
}

func matchTimerFilter(
	rawEnd string,
	bucket string,
	now time.Time,
	customStart time.Time,
	hasCustomStart bool,
	customEnd time.Time,
	hasCustomEnd bool,
) bool {
	if bucket == corporationStructureTimerBucketAll {
		return true
	}
	timerEnd, ok := parseTimeFilter(rawEnd)
	if !ok {
		return false
	}

	switch bucket {
	case corporationStructureTimerBucketCurrentHour:
		base := now.Truncate(time.Hour)
		return !timerEnd.Before(base) && timerEnd.Before(base.Add(time.Hour))
	case corporationStructureTimerBucketNext2Hours:
		return !timerEnd.Before(now) && timerEnd.Before(now.Add(2*time.Hour))
	case corporationStructureTimerBucketCustom:
		if hasCustomStart && timerEnd.Before(customStart) {
			return false
		}
		if hasCustomEnd && timerEnd.After(customEnd) {
			return false
		}
		return hasCustomStart || hasCustomEnd
	default:
		return true
	}
}

func sortCorporationStructureRows(rows []CorporationStructureRow, sortBy string, sortOrder string) {
	normalizedSortBy := sortBy
	if _, ok := corporationStructureSupportedSortBy[normalizedSortBy]; !ok {
		normalizedSortBy = corporationStructureSortFuelRemainingHours
	}
	desc := strings.ToLower(sortOrder) == corporationStructureSortOrderDesc

	sort.SliceStable(rows, func(i, j int) bool {
		a, b := rows[i], rows[j]
		order := compareCorporationStructureRows(a, b, normalizedSortBy)
		if order == 0 {
			if a.CorporationID != b.CorporationID {
				order = compareInt64(a.CorporationID, b.CorporationID)
			} else if a.SystemName != b.SystemName {
				order = compareString(a.SystemName, b.SystemName)
			} else {
				order = compareString(a.Name, b.Name)
			}
		}
		if desc {
			return order > 0
		}
		return order < 0
	})
}

func compareCorporationStructureRows(a CorporationStructureRow, b CorporationStructureRow, sortBy string) int {
	switch sortBy {
	case corporationStructureSortFuelRemainingHours:
		return compareNullableInt(a.FuelRemainingHours, b.FuelRemainingHours)
	case corporationStructureSortSecurity:
		return compareFloat64(a.Security, b.Security)
	case corporationStructureSortReinforceHour:
		return compareInt(a.ReinforceHour, b.ReinforceHour)
	case corporationStructureSortStateTimerEnd:
		return compareNullableTime(a.StateTimerEnd, b.StateTimerEnd)
	case corporationStructureSortUpdatedAt:
		return compareInt64(a.UpdatedAt, b.UpdatedAt)
	case corporationStructureSortSystemName:
		return compareString(a.SystemName, b.SystemName)
	case corporationStructureSortName:
		return compareString(a.Name, b.Name)
	case corporationStructureSortTypeName:
		return compareString(a.TypeName, b.TypeName)
	case corporationStructureSortCorporationName:
		return compareString(a.CorporationName, b.CorporationName)
	default:
		return compareNullableInt(a.FuelRemainingHours, b.FuelRemainingHours)
	}
}

func paginateCorporationStructureRows(rows []CorporationStructureRow, page int, pageSize int) ([]CorporationStructureRow, int, int, int) {
	normalizedPage, normalizedPageSize := normalizePagination(page, pageSize)
	total := len(rows)
	start := (normalizedPage - 1) * normalizedPageSize
	if start >= total {
		return []CorporationStructureRow{}, total, normalizedPage, normalizedPageSize
	}
	end := start + normalizedPageSize
	if end > total {
		end = total
	}
	return rows[start:end], total, normalizedPage, normalizedPageSize
}

func normalizePagination(page int, pageSize int) (int, int) {
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 20
	}
	if pageSize > 200 {
		pageSize = 200
	}
	return page, pageSize
}

func normalizeFuelBucket(bucket string) string {
	switch bucket {
	case corporationStructureFuelBucket24h,
		corporationStructureFuelBucket72h,
		corporationStructureFuelBucket168h,
		corporationStructureFuelBucketCustom:
		return bucket
	default:
		return corporationStructureFuelBucketAll
	}
}

func normalizeTimerBucket(bucket string) string {
	switch bucket {
	case corporationStructureTimerBucketCurrentHour,
		corporationStructureTimerBucketNext2Hours,
		corporationStructureTimerBucketCustom:
		return bucket
	default:
		return corporationStructureTimerBucketAll
	}
}

func normalizeServiceMatchMode(mode string) string {
	if strings.ToLower(mode) == corporationStructureServiceMatchOr {
		return corporationStructureServiceMatchOr
	}
	return corporationStructureServiceMatchAnd
}

func toInt64Set(items []int64) map[int64]struct{} {
	set := make(map[int64]struct{}, len(items))
	for _, item := range items {
		if item > 0 {
			set[item] = struct{}{}
		}
	}
	return set
}

func toStringSet(items []string) map[string]struct{} {
	set := make(map[string]struct{}, len(items))
	for _, item := range items {
		key := strings.TrimSpace(strings.ToLower(item))
		if key != "" {
			set[key] = struct{}{}
		}
	}
	return set
}

func normalizeLowerStringList(items []string) []string {
	result := make([]string, 0, len(items))
	seen := make(map[string]struct{}, len(items))
	for _, item := range items {
		key := strings.TrimSpace(strings.ToLower(item))
		if key == "" {
			continue
		}
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		result = append(result, key)
	}
	return result
}

func parseTimeFilter(raw string) (time.Time, bool) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return time.Time{}, false
	}
	return parseFlexibleTime(raw)
}

func parseFlexibleTime(raw string) (time.Time, bool) {
	layouts := []string{
		time.RFC3339,
		"2006-01-02T15:04:05",
		"2006-01-02 15:04:05",
		"2006-01-02T15:04",
		"2006-01-02 15:04",
	}
	for _, layout := range layouts {
		if ts, err := time.Parse(layout, raw); err == nil {
			return ts, true
		}
	}
	return time.Time{}, false
}

func compareNullableInt(a *int, b *int) int {
	if a == nil && b == nil {
		return 0
	}
	if a == nil {
		return 1
	}
	if b == nil {
		return -1
	}
	return compareInt(*a, *b)
}

func compareNullableTime(aRaw string, bRaw string) int {
	a, aOK := parseTimeFilter(aRaw)
	b, bOK := parseTimeFilter(bRaw)
	if !aOK && !bOK {
		return 0
	}
	if !aOK {
		return 1
	}
	if !bOK {
		return -1
	}
	if a.Before(b) {
		return -1
	}
	if a.After(b) {
		return 1
	}
	return 0
}

func compareString(a string, b string) int {
	aNorm := strings.ToLower(a)
	bNorm := strings.ToLower(b)
	if aNorm < bNorm {
		return -1
	}
	if aNorm > bNorm {
		return 1
	}
	return 0
}

func compareInt(a int, b int) int {
	if a < b {
		return -1
	}
	if a > b {
		return 1
	}
	return 0
}

func compareInt64(a int64, b int64) int {
	if a < b {
		return -1
	}
	if a > b {
		return 1
	}
	return 0
}

func compareFloat64(a float64, b float64) int {
	if a < b {
		return -1
	}
	if a > b {
		return 1
	}
	return 0
}

func logCorporationStructuresWarn(message string, err error, fields ...zap.Field) {
	if global.Logger != nil {
		fields = append([]zap.Field{zap.Error(err)}, fields...)
		global.Logger.Warn(message, fields...)
	}
}

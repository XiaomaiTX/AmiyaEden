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
}

func NewCorporationStructureService() *CorporationStructureService {
	cfg := global.Config.EveSSO
	return &CorporationStructureService{
		roleRepo:      repository.NewRoleRepository(),
		charRepo:      repository.NewEveCharacterRepository(),
		sysConfigRepo: repository.NewSysConfigRepository(),
		sdeRepo:       repository.NewSdeRepository(),
		repo:          repository.NewCorporationStructureRepository(),
		esiClient:     esi.NewClientWithConfig(cfg.ESIBaseURL, cfg.ESIAPIPrefix),
	}
}

type CorporationStructureServiceInfo struct {
	Name  string `json:"name"`
	State string `json:"state"`
}

type CorporationStructureRow struct {
	CorporationID      int64                             `json:"corporation_id"`
	CorporationName    string                            `json:"corporation_name"`
	StructureID        int64                             `json:"structure_id"`
	Name               string                            `json:"name"`
	TypeID             int64                             `json:"type_id"`
	TypeName           string                            `json:"type_name"`
	SystemID           int64                             `json:"system_id"`
	SystemName         string                            `json:"system_name"`
	RegionID           int64                             `json:"region_id"`
	RegionName         string                            `json:"region_name"`
	Security           float64                           `json:"security"`
	State              string                            `json:"state"`
	Services           []CorporationStructureServiceInfo `json:"services"`
	FuelExpires        string                            `json:"fuel_expires"`
	FuelRemaining      string                            `json:"fuel_remaining"`
	FuelRemainingHours *int                              `json:"fuel_remaining_hours"`
	ReinforceHour      int                               `json:"reinforce_hour"`
	StateTimerStart    string                            `json:"state_timer_start"`
	StateTimerEnd      string                            `json:"state_timer_end"`
	UpdatedAt          int64                             `json:"updated_at"`
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
}

type CorporationStructureAuthorizationBinding struct {
	CorporationID int64 `json:"corporation_id"`
	CharacterID   int64 `json:"character_id"`
}

type CorporationStructureAuthorizationUpdate struct {
	Authorizations           []CorporationStructureAuthorizationBinding `json:"authorizations"`
	FuelNoticeThresholdDays  *int                                       `json:"fuel_notice_threshold_days"`
	TimerNoticeThresholdDays *int                                       `json:"timer_notice_threshold_days"`
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
	}, nil
}

func (s *CorporationStructureService) UpdateAuthorizations(
	ctx context.Context,
	req CorporationStructureAuthorizationUpdate,
) error {
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
	if err := validateAuthorizationBindings(req.Authorizations, managedCorps, directorSetByCorp); err != nil {
		return err
	}
	for _, binding := range req.Authorizations {
		if binding.CharacterID == 0 {
			delete(currentMap, binding.CorporationID)
		} else {
			currentMap[binding.CorporationID] = binding.CharacterID
		}
	}

	if err := s.saveAuthorizationMap(currentMap); err != nil {
		return err
	}

	thresholds := s.loadNoticeThresholdSettings()
	if req.FuelNoticeThresholdDays != nil {
		thresholds.FuelNoticeThresholdDays = *req.FuelNoticeThresholdDays
	}
	if req.TimerNoticeThresholdDays != nil {
		thresholds.TimerNoticeThresholdDays = *req.TimerNoticeThresholdDays
	}
	return s.saveNoticeThresholdSettings(thresholds)
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

	systemMeta := s.loadSystemMetaMap(collectSystemIDs(structures))
	now := time.Now()
	items := make([]CorporationStructureRow, 0, len(structures))
	for _, st := range structures {
		row := buildCorporationStructureRow(st, now, systemMeta[st.SystemID])
		items = append(items, row)
	}

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

	type esiNameEntry struct {
		ID   int64  `json:"id"`
		Name string `json:"name"`
	}
	var entries []esiNameEntry
	if err := s.esiClient.PostJSON(ctx, "/universe/names?datasource=tranquility", "", corporationIDs, &entries); err != nil {
		logCorporationStructuresWarn("[CorporationStructures] 解析军团名称失败", err)
		return names
	}
	for _, entry := range entries {
		if entry.Name != "" {
			names[entry.ID] = entry.Name
		}
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

func logCorporationStructuresWarn(message string, err error) {
	if global.Logger != nil {
		global.Logger.Warn(message, zap.Error(err))
	}
}

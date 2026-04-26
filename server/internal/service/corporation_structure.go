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
	"sort"
	"strconv"
	"sync"
	"time"

	"go.uber.org/zap"
)

type CorporationStructureService struct {
	roleRepo      *repository.RoleRepository
	charRepo      *repository.EveCharacterRepository
	sysConfigRepo *repository.SysConfigRepository
	sdeRepo       *repository.SdeRepository
	repo          *repository.CorporationStructureRepository
	ssoSvc        *EveSSOService
	esiClient     *esi.Client
	refreshGuards sync.Map
}

const (
	corporationStructurePaginationConcurrency = 2
	corporationStructureDetailInterval        = 500 * time.Millisecond
)

func NewCorporationStructureService() *CorporationStructureService {
	cfg := global.Config.EveSSO
	return &CorporationStructureService{
		roleRepo:      repository.NewRoleRepository(),
		charRepo:      repository.NewEveCharacterRepository(),
		sysConfigRepo: repository.NewSysConfigRepository(),
		sdeRepo:       repository.NewSdeRepository(),
		repo:          repository.NewCorporationStructureRepository(),
		ssoSvc:        NewEveSSOService(),
		esiClient:     esi.NewClientWithConfig(cfg.ESIBaseURL, cfg.ESIAPIPrefix),
	}
}

type CorporationStructureServiceInfo struct {
	Name  string `json:"name"`
	State string `json:"state"`
}

type CorporationStructureRow struct {
	CorporationID   int64                             `json:"corporation_id"`
	CorporationName string                            `json:"corporation_name"`
	StructureID     int64                             `json:"structure_id"`
	Name            string                            `json:"name"`
	TypeID          int64                             `json:"type_id"`
	TypeName        string                            `json:"type_name"`
	SystemID        int64                             `json:"system_id"`
	SystemName      string                            `json:"system_name"`
	Security        float64                           `json:"security"`
	State           string                            `json:"state"`
	Services        []CorporationStructureServiceInfo `json:"services"`
	FuelExpires     string                            `json:"fuel_expires"`
	FuelRemaining   string                            `json:"fuel_remaining"`
	ReinforceHour   int                               `json:"reinforce_hour"`
	UpdatedAt       int64                             `json:"updated_at"`
}

type CorporationStructureListRequest struct {
	CorporationID int64 `json:"corporation_id"`
}

type CorporationStructureListResponse struct {
	Items []CorporationStructureRow `json:"items"`
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
	Corporations []ManageCorporationOption `json:"corporations"`
}

type CorporationStructureAuthorizationBinding struct {
	CorporationID int64 `json:"corporation_id"`
	CharacterID   int64 `json:"character_id"`
}

type CorporationStructureAuthorizationUpdate struct {
	Authorizations []CorporationStructureAuthorizationBinding `json:"authorizations"`
}

type CorporationStructureRefreshRequest struct {
	CorporationID int64 `json:"corporation_id"`
}

type CorporationStructureRefreshResponse struct {
	CorporationID int64  `json:"corporation_id"`
	Queued        bool   `json:"queued"`
	Running       bool   `json:"running"`
	Message       string `json:"message"`
}

type corpManageContext struct {
	corporationIDs []int64
	corpNameByID   map[int64]string
	directorByCorp map[int64][]repository.DirectorCharacterOption
}

func (s *CorporationStructureService) GetSettings(ctx context.Context) (*CorporationStructuresSettingsResponse, error) {
	manageCtx, err := s.buildManageContext(ctx, true)
	if err != nil {
		return nil, err
	}
	authMap := s.loadAuthorizationMap()

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

	return &CorporationStructuresSettingsResponse{Corporations: corporations}, nil
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

	return s.saveAuthorizationMap(currentMap)
}

func (s *CorporationStructureService) ListStructures(
	ctx context.Context,
	req CorporationStructureListRequest,
) (*CorporationStructureListResponse, error) {
	manageCtx, err := s.buildManageContext(ctx, true)
	if err != nil {
		return nil, err
	}

	targetCorps := manageCtx.corporationIDs
	if req.CorporationID > 0 {
		found := false
		for _, corpID := range manageCtx.corporationIDs {
			if corpID == req.CorporationID {
				found = true
				break
			}
		}
		if !found {
			return nil, errors.New("无权访问该军团建筑数据")
		}
		targetCorps = []int64{req.CorporationID}
	}

	structures, err := s.repo.ListCorpStructures(targetCorps)
	if err != nil {
		return nil, errors.New("查询建筑快照失败")
	}

	typeIDs := make([]int, 0)
	systemIDs := make([]int64, 0)
	typeSet := map[int]struct{}{}
	systemSet := map[int64]struct{}{}
	for _, st := range structures {
		if st.TypeID > 0 {
			typeID := int(st.TypeID)
			if _, ok := typeSet[typeID]; !ok {
				typeSet[typeID] = struct{}{}
				typeIDs = append(typeIDs, typeID)
			}
		}
		if st.SystemID > 0 {
			if _, ok := systemSet[st.SystemID]; !ok {
				systemSet[st.SystemID] = struct{}{}
				systemIDs = append(systemIDs, st.SystemID)
			}
		}
	}

	typeNames := map[int64]string{}
	if len(typeIDs) > 0 {
		typeInfos, typeErr := s.sdeRepo.GetTypes(typeIDs, nil, "zh")
		if typeErr == nil {
			for _, info := range typeInfos {
				typeNames[int64(info.TypeID)] = info.TypeName
			}
		}
	}

	systemInfoByID := s.resolveSystems(ctx, systemIDs)

	items := make([]CorporationStructureRow, 0, len(structures))
	for _, st := range structures {
		systemInfo := systemInfoByID[st.SystemID]
		row := CorporationStructureRow{
			CorporationID:   st.CorporationID,
			CorporationName: manageCtx.corpNameByID[st.CorporationID],
			StructureID:     st.StructureID,
			Name:            st.Name,
			TypeID:          st.TypeID,
			TypeName:        typeNames[st.TypeID],
			SystemID:        st.SystemID,
			SystemName:      systemInfo.Name,
			Security:        systemInfo.Security,
			State:           st.State,
			Services:        convertStructureServices(st.Services),
			FuelExpires:     st.FuelExpires,
			FuelRemaining:   formatFuelRemaining(st.FuelExpires),
			ReinforceHour:   st.ReinforceHour,
			UpdatedAt:       st.UpdateAt,
		}
		if row.Name == "" {
			row.Name = fmt.Sprintf("Structure-%d", st.StructureID)
		}
		if row.TypeName == "" {
			row.TypeName = fmt.Sprintf("Type-%d", st.TypeID)
		}
		if row.SystemName == "" && st.SystemID > 0 {
			row.SystemName = fmt.Sprintf("System-%d", st.SystemID)
		}
		items = append(items, row)
	}

	sort.Slice(items, func(i, j int) bool {
		if items[i].CorporationID != items[j].CorporationID {
			return items[i].CorporationID < items[j].CorporationID
		}
		if items[i].SystemName != items[j].SystemName {
			return items[i].SystemName < items[j].SystemName
		}
		return items[i].Name < items[j].Name
	})

	return &CorporationStructureListResponse{Items: items}, nil
}

func (s *CorporationStructureService) RefreshStructures(
	ctx context.Context,
	req CorporationStructureRefreshRequest,
) (*CorporationStructureRefreshResponse, error) {
	guard, err := s.prepareRefresh(ctx, req)
	if err != nil {
		return nil, err
	}
	if err := guard.Start("该军团建筑刷新任务"); err != nil {
		return &CorporationStructureRefreshResponse{
			CorporationID: req.CorporationID,
			Queued:        false,
			Running:       true,
			Message:       err.Error(),
		}, nil
	}
	defer guard.Finish()

	refreshed, runErr := s.runRefreshStructures(ctx, req)
	if runErr != nil {
		return nil, runErr
	}
	return &CorporationStructureRefreshResponse{
		CorporationID: req.CorporationID,
		Queued:        false,
		Running:       false,
		Message:       fmt.Sprintf("军团建筑数据刷新完成，共更新 %d 条", refreshed),
	}, nil
}

func (s *CorporationStructureService) EnqueueRefreshStructures(
	ctx context.Context,
	req CorporationStructureRefreshRequest,
) (*CorporationStructureRefreshResponse, error) {
	guard, err := s.prepareRefresh(ctx, req)
	if err != nil {
		return nil, err
	}
	if err := guard.Start("该军团建筑刷新任务"); err != nil {
		return &CorporationStructureRefreshResponse{
			CorporationID: req.CorporationID,
			Queued:        false,
			Running:       true,
			Message:       err.Error(),
		}, nil
	}

	taskName := fmt.Sprintf("dashboard_corp_structures_refresh_%d", req.CorporationID)
	ok := global.EnsureBackgroundTaskManager().Go(taskName, func(bgCtx context.Context) error {
		defer guard.Finish()
		refreshed, runErr := s.runRefreshStructures(bgCtx, req)
		if runErr != nil {
			global.Logger.Warn("[CorporationStructures] 后台刷新失败",
				zap.Int64("corporation_id", req.CorporationID),
				zap.Error(runErr))
			return runErr
		}
		global.Logger.Info("[CorporationStructures] 后台刷新完成",
			zap.Int64("corporation_id", req.CorporationID),
			zap.Int("refreshed", refreshed))
		return nil
	})
	if !ok {
		guard.Finish()
		return nil, errors.New("服务正在关闭，任务未启动")
	}

	return &CorporationStructureRefreshResponse{
		CorporationID: req.CorporationID,
		Queued:        true,
		Running:       false,
		Message:       "已加入后台刷新队列",
	}, nil
}

func (s *CorporationStructureService) runRefreshStructures(
	ctx context.Context,
	req CorporationStructureRefreshRequest,
) (int, error) {
	if req.CorporationID <= 0 {
		return 0, errors.New("corporation_id 必须为正整数")
	}
	characterID, err := s.resolveRefreshAuthorization(ctx, req.CorporationID)
	if err != nil {
		return 0, err
	}

	token, err := s.ssoSvc.GetValidToken(ctx, characterID)
	if err != nil {
		return 0, errors.New("获取授权角色 Token 失败")
	}

	type corpStructureESIResp struct {
		FuelExpires        string `json:"fuel_expires"`
		Name               string `json:"name"`
		NextReinforceApply string `json:"next_reinforce_apply"`
		NextReinforceHour  int    `json:"next_reinforce_hour"`
		ProfileID          int64  `json:"profile_id"`
		ReinforceHour      int    `json:"reinforce_hour"`
		Services           []struct {
			Name  string `json:"name"`
			State string `json:"state"`
		} `json:"services"`
		State           string `json:"state"`
		StateTimerEnd   string `json:"state_timer_end"`
		StateTimerStart string `json:"state_timer_start"`
		StructureID     int64  `json:"structure_id"`
		SystemID        int64  `json:"system_id"`
		TypeID          int64  `json:"type_id"`
		UnanchorsAt     string `json:"unanchors_at"`
	}

	structuresPath := fmt.Sprintf("/corporations/%d/structures/", req.CorporationID)
	var esiStructures []corpStructureESIResp
	if _, err := s.esiClient.GetPaginatedWithConcurrency(
		ctx,
		structuresPath,
		token,
		&esiStructures,
		corporationStructurePaginationConcurrency,
	); err != nil {
		return 0, fmt.Errorf("拉取军团建筑失败: %w", err)
	}

	now := time.Now().Unix()
	corpRecords := make([]model.CorpStructureInfo, 0, len(esiStructures))
	type structureDetail struct {
		Name     string `json:"name"`
		OwnerID  int64  `json:"owner_id"`
		Position struct {
			X float64 `json:"x"`
			Y float64 `json:"y"`
			Z float64 `json:"z"`
		} `json:"position"`
		SolarSystemID int64 `json:"solar_system_id"`
		TypeID        int64 `json:"type_id"`
	}
	eveStructures := make([]model.EveStructure, 0, len(esiStructures))

	for idx, st := range esiStructures {
		servicesJSON, _ := json.Marshal(st.Services)
		corpRecords = append(corpRecords, model.CorpStructureInfo{
			CorporationID:      req.CorporationID,
			StructureID:        st.StructureID,
			Services:           string(servicesJSON),
			FuelExpires:        st.FuelExpires,
			Name:               st.Name,
			NextReinforceApply: st.NextReinforceApply,
			NextReinforceHour:  st.NextReinforceHour,
			ProfileID:          st.ProfileID,
			ReinforceHour:      st.ReinforceHour,
			State:              st.State,
			StateTimerEnd:      st.StateTimerEnd,
			StateTimerStart:    st.StateTimerStart,
			SystemID:           st.SystemID,
			TypeID:             st.TypeID,
			UnanchorsAt:        st.UnanchorsAt,
			UpdateAt:           now,
		})

		if idx > 0 {
			select {
			case <-time.After(corporationStructureDetailInterval):
			case <-ctx.Done():
				return 0, ctx.Err()
			}
		}

		var detail structureDetail
		path := fmt.Sprintf("/universe/structures/%d/", st.StructureID)
		if err := s.esiClient.Get(ctx, path, token, &detail); err != nil {
			global.Logger.Warn("[CorporationStructures] 拉取建筑详情失败",
				zap.Int64("structure_id", st.StructureID),
				zap.Int64("corporation_id", req.CorporationID),
				zap.Error(err))
			continue
		}
		eveStructures = append(eveStructures, model.EveStructure{
			StructureID:   st.StructureID,
			StructureName: detail.Name,
			OwnerID:       detail.OwnerID,
			TypeID:        detail.TypeID,
			SolarSystemID: detail.SolarSystemID,
			X:             detail.Position.X,
			Y:             detail.Position.Y,
			Z:             detail.Position.Z,
			UpdateAt:      now,
		})
	}

	if err := s.repo.UpsertCorpStructures(corpRecords); err != nil {
		return 0, errors.New("写入军团建筑快照失败")
	}
	if err := s.repo.UpsertStructures(eveStructures); err != nil {
		return 0, errors.New("写入建筑详情快照失败")
	}

	return len(corpRecords), nil
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

func (s *CorporationStructureService) resolveRefreshAuthorization(
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

func (s *CorporationStructureService) getRefreshGuard(corporationID int64) *exclusiveRunGuard {
	guard, _ := s.refreshGuards.LoadOrStore(corporationID, &exclusiveRunGuard{})
	return guard.(*exclusiveRunGuard)
}

func (s *CorporationStructureService) prepareRefresh(
	ctx context.Context,
	req CorporationStructureRefreshRequest,
) (*exclusiveRunGuard, error) {
	if _, err := s.resolveRefreshAuthorization(ctx, req.CorporationID); err != nil {
		return nil, err
	}
	return s.getRefreshGuard(req.CorporationID), nil
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
		global.Logger.Warn("[CorporationStructures] 解析军团名称失败", zap.Error(err))
		return names
	}
	for _, entry := range entries {
		if entry.Name != "" {
			names[entry.ID] = entry.Name
		}
	}
	return names
}

type systemInfo struct {
	Name     string
	Security float64
}

func (s *CorporationStructureService) resolveSystems(ctx context.Context, systemIDs []int64) map[int64]systemInfo {
	result := make(map[int64]systemInfo, len(systemIDs))
	type esiSystem struct {
		Name           string  `json:"name"`
		SecurityStatus float64 `json:"security_status"`
	}
	for _, systemID := range systemIDs {
		var info esiSystem
		path := fmt.Sprintf("/universe/systems/%d/", systemID)
		if err := s.esiClient.Get(ctx, path, "", &info); err != nil {
			continue
		}
		result[systemID] = systemInfo{
			Name:     info.Name,
			Security: info.SecurityStatus,
		}
	}
	return result
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

func formatFuelRemaining(fuelExpires string) string {
	if fuelExpires == "" {
		return ""
	}
	ts, err := time.Parse(time.RFC3339, fuelExpires)
	if err != nil {
		return ""
	}
	diff := time.Until(ts)
	if diff <= 0 {
		return "expired"
	}
	totalHours := int(diff.Hours())
	days := totalHours / 24
	hours := totalHours % 24
	if days > 0 {
		return fmt.Sprintf("%dd %dh", days, hours)
	}
	return fmt.Sprintf("%dh", hours)
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

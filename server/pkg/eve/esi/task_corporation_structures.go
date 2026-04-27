package esi

import (
	"amiya-eden/global"
	"amiya-eden/internal/model"
	"amiya-eden/internal/repository"
	"amiya-eden/internal/utils"
	apputils "amiya-eden/pkg/utils"
	"context"
	"encoding/json"
	"fmt"
	"time"

	"go.uber.org/zap"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const (
	corporationStructureTaskPaginationConcurrency = 2
	corporationStructureTaskDetailInterval        = 500 * time.Millisecond
)

func init() {
	Register(&CorporationStructuresTask{})
}

type CorporationStructuresTask struct{}

func (t *CorporationStructuresTask) Name() string        { return "corporation_structures" }
func (t *CorporationStructuresTask) Description() string { return "军团建筑信息" }
func (t *CorporationStructuresTask) Priority() Priority  { return PriorityLow }

func (t *CorporationStructuresTask) Interval() RefreshInterval {
	return RefreshInterval{
		Active:   3 * 24 * time.Hour,
		Inactive: 7 * 24 * time.Hour,
	}
}

func (t *CorporationStructuresTask) RequiredScopes() []TaskScope {
	return []TaskScope{
		{Scope: "esi-corporations.read_structures.v1", Description: "读取军团建筑信息"},
		{Scope: "esi-universe.read_structures.v1", Description: "读取建筑信息"},
	}
}

type corpStructureESIResponse struct {
	CorporationID      int64  `json:"corporation_id"`
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

type esiNameResponse struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
}

type corpStructureSystemSnapshot struct {
	Name     string
	Security float64
}

func corpStructuresLogger() *zap.Logger {
	if global.Logger != nil {
		return global.Logger
	}
	return zap.NewNop()
}

func (t *CorporationStructuresTask) Execute(ctx *TaskContext) error {
	bgCtx := ctx.ContextOrBackground()
	now := time.Now().Unix()

	var corpRoles []string
	if err := global.DB.Model(&model.EveCharacterCorpRole{}).
		Where("character_id = ?", ctx.CharacterID).
		Pluck("corp_role", &corpRoles).Error; err != nil {
		return fmt.Errorf("query corp roles: %w", err)
	}
	if !apputils.ContainsAny(corpRoles, []string{"Director"}) {
		corpStructuresLogger().Debug("[ESI] 人物没有足够的军团职权，跳过军团建筑刷新",
			zap.Int64("character_id", ctx.CharacterID),
			zap.Strings("corp_roles", corpRoles))
		return ErrTaskSkipped
	}

	var corporationID int64
	if err := global.DB.Model(&model.EveCharacter{}).
		Where("character_id = ?", ctx.CharacterID).
		Pluck("corporation_id", &corporationID).Error; err != nil {
		return fmt.Errorf("query corporation id: %w", err)
	}
	if !isCorporationAllowed(corporationID, utils.GetAllowCorporations()) {
		corpStructuresLogger().Debug("[ESI] 人物所在军团不在 allow_corporations，跳过军团建筑刷新",
			zap.Int64("character_id", ctx.CharacterID),
			zap.Int64("corporation_id", corporationID))
		return ErrTaskSkipped
	}

	authorizations, err := loadCorpStructureAuthorizationMap()
	if err != nil {
		return fmt.Errorf("load corp structure authorizations: %w", err)
	}
	if authorizations[corporationID] != ctx.CharacterID {
		corpStructuresLogger().Debug("[ESI] 人物不是军团建筑授权角色，跳过军团建筑刷新",
			zap.Int64("character_id", ctx.CharacterID),
			zap.Int64("corporation_id", corporationID))
		return ErrTaskSkipped
	}

	var esiStructures []corpStructureESIResponse
	corpStructuresPath := fmt.Sprintf("/corporations/%d/structures/", corporationID)
	if _, err := ctx.Client.GetPaginatedWithConcurrency(
		bgCtx,
		corpStructuresPath,
		ctx.AccessToken,
		&esiStructures,
		corporationStructureTaskPaginationConcurrency,
	); err != nil {
		corpStructuresLogger().Warn("[ESI] 获取军团建筑信息失败",
			zap.Int64("character_id", ctx.CharacterID),
			zap.Int64("corporation_id", corporationID),
			zap.Error(err),
		)
		return fmt.Errorf("fetch corp structures: %w", err)
	}

	if len(esiStructures) == 0 {
		deletedCount, err := syncCorporationStructureSnapshots(corporationID, nil, nil)
		if err != nil {
			return err
		}
		corpStructuresLogger().Debug("[ESI] 军团建筑信息刷新完成",
			zap.Int64("character_id", ctx.CharacterID),
			zap.Int64("corporation_id", corporationID),
			zap.Int("count", 0),
			zap.Int64("deleted_count", deletedCount),
		)
		return nil
	}

	structureIDs := make([]int64, 0, len(esiStructures))
	typeSet := make(map[int]struct{}, len(esiStructures))
	systemSet := make(map[int64]struct{}, len(esiStructures))
	for _, structure := range esiStructures {
		structureIDs = append(structureIDs, structure.StructureID)
		if structure.TypeID > 0 {
			typeSet[int(structure.TypeID)] = struct{}{}
		}
		if structure.SystemID > 0 {
			systemSet[structure.SystemID] = struct{}{}
		}
	}
	existingByStructureID := loadExistingCorpStructureSnapshots(corporationID, structureIDs)

	corporationName := resolveCorporationNameSnapshot(
		bgCtx,
		ctx.Client,
		corporationID,
		existingByStructureID,
	)
	typeNamesByTypeID := resolveTypeNameSnapshots(typeSet, existingByStructureID)
	systemSnapshotBySystemID := resolveSystemSnapshots(systemSet, existingByStructureID)

	corpRecords := make([]model.CorpStructureInfo, 0, len(esiStructures))
	for _, structure := range esiStructures {
		servicesJSON, _ := json.Marshal(structure.Services)
		existing := existingByStructureID[structure.StructureID]
		typeName := chooseSnapshotText(
			typeNamesByTypeID[structure.TypeID],
			existing.TypeName,
			fmt.Sprintf("Type-%d", structure.TypeID),
		)
		systemSnapshot := systemSnapshotBySystemID[structure.SystemID]
		systemName := chooseSnapshotText(
			systemSnapshot.Name,
			existing.SystemName,
			fmt.Sprintf("System-%d", structure.SystemID),
		)
		security := existing.Security
		if systemSnapshot.Name != "" {
			security = systemSnapshot.Security
		}

		corpRecords = append(corpRecords, model.CorpStructureInfo{
			CorporationID:      corporationID,
			CorporationName:    chooseSnapshotText(corporationName, existing.CorporationName, fmt.Sprintf("Corporation-%d", corporationID)),
			StructureID:        structure.StructureID,
			Services:           string(servicesJSON),
			FuelExpires:        structure.FuelExpires,
			Name:               structure.Name,
			NextReinforceApply: structure.NextReinforceApply,
			NextReinforceHour:  structure.NextReinforceHour,
			ProfileID:          structure.ProfileID,
			ReinforceHour:      structure.ReinforceHour,
			State:              structure.State,
			StateTimerEnd:      structure.StateTimerEnd,
			StateTimerStart:    structure.StateTimerStart,
			SystemID:           structure.SystemID,
			SystemName:         systemName,
			Security:           security,
			TypeID:             structure.TypeID,
			TypeName:           typeName,
			UnanchorsAt:        structure.UnanchorsAt,
			UpdateAt:           now,
		})
	}
	deletedCount, err := syncCorporationStructureSnapshots(corporationID, corpRecords, structureIDs)
	if err != nil {
		return err
	}

	structureDetails := make([]model.EveStructure, 0, len(esiStructures))
	for i, structure := range esiStructures {
		if i > 0 {
			select {
			case <-time.After(corporationStructureTaskDetailInterval):
			case <-bgCtx.Done():
				return bgCtx.Err()
			}
		}

		var detail universeStructureDetail
		structurePath := fmt.Sprintf("/universe/structures/%d/", structure.StructureID)
		if err := ctx.Client.Get(bgCtx, structurePath, ctx.AccessToken, &detail); err != nil {
			corpStructuresLogger().Warn("[ESI] 获取建筑详情失败",
				zap.Int64("structure_id", structure.StructureID),
				zap.Error(err),
			)
			continue
		}

		structureDetails = append(structureDetails, model.EveStructure{
			StructureID:   structure.StructureID,
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
	if len(structureDetails) > 0 {
		if err := global.DB.Clauses(clause.OnConflict{UpdateAll: true}).Create(&structureDetails).Error; err != nil {
			corpStructuresLogger().Warn("[ESI] Upsert 建筑详情失败",
				zap.Int64("corporation_id", corporationID),
				zap.Error(err),
			)
		}
	}

	corpStructuresLogger().Debug("[ESI] 军团建筑信息刷新完成",
		zap.Int64("character_id", ctx.CharacterID),
		zap.Int64("corporation_id", corporationID),
		zap.Int("count", len(esiStructures)),
		zap.Int64("deleted_count", deletedCount),
	)
	return nil
}

func syncCorporationStructureSnapshots(
	corporationID int64,
	records []model.CorpStructureInfo,
	structureIDs []int64,
) (int64, error) {
	var deletedRows int64
	err := global.DB.Transaction(func(tx *gorm.DB) error {
		if len(records) > 0 {
			if err := tx.Clauses(clause.OnConflict{UpdateAll: true}).Create(&records).Error; err != nil {
				return fmt.Errorf("upsert corp structures: %w", err)
			}
		}

		deleteQuery := tx.Where("corporation_id = ?", corporationID)
		if len(structureIDs) > 0 {
			deleteQuery = deleteQuery.Where("structure_id NOT IN ?", structureIDs)
		}
		result := deleteQuery.Delete(&model.CorpStructureInfo{})
		if result.Error != nil {
			return fmt.Errorf("delete stale corp structures: %w", result.Error)
		}
		deletedRows = result.RowsAffected
		return nil
	})
	if err != nil {
		return 0, err
	}
	return deletedRows, nil
}

func loadExistingCorpStructureSnapshots(
	corporationID int64,
	structureIDs []int64,
) map[int64]model.CorpStructureInfo {
	result := make(map[int64]model.CorpStructureInfo, len(structureIDs))
	if len(structureIDs) == 0 {
		return result
	}
	rows := make([]model.CorpStructureInfo, 0, len(structureIDs))
	if err := global.DB.
		Where("corporation_id = ? AND structure_id IN ?", corporationID, structureIDs).
		Find(&rows).Error; err != nil {
		return result
	}
	for _, row := range rows {
		result[row.StructureID] = row
	}
	return result
}

func resolveCorporationNameSnapshot(
	ctx context.Context,
	client *Client,
	corporationID int64,
	existingByStructureID map[int64]model.CorpStructureInfo,
) string {
	var entries []esiNameResponse
	if err := client.PostJSON(
		ctx,
		"/universe/names?datasource=tranquility",
		"",
		[]int64{corporationID},
		&entries,
	); err == nil {
		for _, entry := range entries {
			if entry.ID == corporationID && entry.Name != "" {
				return entry.Name
			}
		}
	}

	for _, existing := range existingByStructureID {
		if existing.CorporationName != "" {
			return existing.CorporationName
		}
	}
	return ""
}

func resolveTypeNameSnapshots(
	typeSet map[int]struct{},
	existingByStructureID map[int64]model.CorpStructureInfo,
) map[int64]string {
	result := make(map[int64]string, len(typeSet))
	if len(typeSet) == 0 {
		return result
	}
	typeIDs := make([]int, 0, len(typeSet))
	for typeID := range typeSet {
		typeIDs = append(typeIDs, typeID)
	}

	sdeRepo := repository.NewSdeRepository()
	typeInfos, err := sdeRepo.GetTypes(typeIDs, nil, "zh")
	if err != nil {
		for _, existing := range existingByStructureID {
			if existing.TypeID > 0 && existing.TypeName != "" {
				result[existing.TypeID] = existing.TypeName
			}
		}
		return result
	}

	for _, info := range typeInfos {
		if info.TypeName != "" {
			result[int64(info.TypeID)] = info.TypeName
		}
	}
	return result
}

func resolveSystemSnapshots(
	systemSet map[int64]struct{},
	existingByStructureID map[int64]model.CorpStructureInfo,
) map[int64]corpStructureSystemSnapshot {
	result := make(map[int64]corpStructureSystemSnapshot, len(systemSet))
	if len(systemSet) == 0 {
		return result
	}
	systemIDs := make([]int64, 0, len(systemSet))
	for systemID := range systemSet {
		systemIDs = append(systemIDs, systemID)
	}

	rows := make([]model.MapSolarSystem, 0, len(systemIDs))
	if err := global.DB.
		Where(`"solarSystemID" IN ?`, systemIDs).
		Find(&rows).Error; err != nil {
		for _, existing := range existingByStructureID {
			if existing.SystemID > 0 && existing.SystemName != "" {
				result[existing.SystemID] = corpStructureSystemSnapshot{
					Name:     existing.SystemName,
					Security: existing.Security,
				}
			}
		}
		return result
	}

	for _, row := range rows {
		result[int64(row.SolarSystemID)] = corpStructureSystemSnapshot{
			Name:     row.SolarSystemName,
			Security: row.Security,
		}
	}
	return result
}

func chooseSnapshotText(preferred string, oldValue string, placeholder string) string {
	if preferred != "" {
		return preferred
	}
	if oldValue != "" {
		return oldValue
	}
	return placeholder
}

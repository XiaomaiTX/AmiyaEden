package esi

import (
	"amiya-eden/global"
	"amiya-eden/internal/model"
	"amiya-eden/internal/utils"
	apputils "amiya-eden/pkg/utils"
	"encoding/json"
	"fmt"
	"time"

	"go.uber.org/zap"
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
		global.Logger.Debug("[ESI] 人物没有足够的军团职权，跳过军团建筑刷新",
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
		global.Logger.Debug("[ESI] 人物所在军团不在 allow_corporations，跳过军团建筑刷新",
			zap.Int64("character_id", ctx.CharacterID),
			zap.Int64("corporation_id", corporationID))
		return ErrTaskSkipped
	}

	authorizations, err := loadCorpStructureAuthorizationMap()
	if err != nil {
		return fmt.Errorf("load corp structure authorizations: %w", err)
	}
	if authorizations[corporationID] != ctx.CharacterID {
		global.Logger.Debug("[ESI] 人物不是军团建筑授权角色，跳过军团建筑刷新",
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
		global.Logger.Warn("[ESI] 获取军团建筑信息失败",
			zap.Int64("character_id", ctx.CharacterID),
			zap.Int64("corporation_id", corporationID),
			zap.Error(err),
		)
		return fmt.Errorf("fetch corp structures: %w", err)
	}

	if len(esiStructures) == 0 {
		return nil
	}

	corpRecords := make([]model.CorpStructureInfo, 0, len(esiStructures))
	for _, structure := range esiStructures {
		servicesJSON, _ := json.Marshal(structure.Services)
		corpRecords = append(corpRecords, model.CorpStructureInfo{
			CorporationID:      corporationID,
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
			TypeID:             structure.TypeID,
			UnanchorsAt:        structure.UnanchorsAt,
			UpdateAt:           now,
		})
	}
	if err := global.DB.Clauses(clause.OnConflict{UpdateAll: true}).Create(&corpRecords).Error; err != nil {
		return fmt.Errorf("upsert corp structures: %w", err)
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
			global.Logger.Warn("[ESI] 获取建筑详情失败",
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
			global.Logger.Warn("[ESI] Upsert 建筑详情失败",
				zap.Int64("corporation_id", corporationID),
				zap.Error(err),
			)
		}
	}

	global.Logger.Debug("[ESI] 军团建筑信息刷新完成",
		zap.Int64("character_id", ctx.CharacterID),
		zap.Int64("corporation_id", corporationID),
		zap.Int("count", len(esiStructures)),
	)
	return nil
}

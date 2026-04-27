package esi

import (
	"amiya-eden/global"
	"amiya-eden/internal/model"
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"go.uber.org/zap"
	"gorm.io/gorm/clause"
)

const (
	personalStructureTaskDetailInterval = 500 * time.Millisecond
	minStructureID                      = 1_000_000_000_000
)

func init() {
	Register(&StructureTask{})
}

type StructureTask struct{}

func (t *StructureTask) Name() string        { return "eve_structures" }
func (t *StructureTask) Description() string { return "EVE 建筑详情（个人相关）" }
func (t *StructureTask) Priority() Priority  { return PriorityLow }

func (t *StructureTask) Interval() RefreshInterval {
	return RefreshInterval{
		Active:   3 * 24 * time.Hour,
		Inactive: 7 * 24 * time.Hour,
	}
}

func (t *StructureTask) RequiredScopes() []TaskScope {
	return []TaskScope{
		{Scope: "esi-universe.read_structures.v1", Description: "读取建筑信息"},
	}
}

type universeStructureDetail struct {
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

func (t *StructureTask) Execute(ctx *TaskContext) error {
	bgCtx := ctx.ContextOrBackground()

	structureIDs, err := loadPersonalStructureIDs(bgCtx, ctx.CharacterID)
	if err != nil {
		return fmt.Errorf("load personal structure ids: %w", err)
	}
	if len(structureIDs) == 0 {
		return nil
	}

	now := time.Now().Unix()
	records := make([]model.EveStructure, 0, len(structureIDs))
	for i, structureID := range structureIDs {
		if i > 0 {
			select {
			case <-time.After(personalStructureTaskDetailInterval):
			case <-bgCtx.Done():
				return bgCtx.Err()
			}
		}

		var detail universeStructureDetail
		structurePath := fmt.Sprintf("/universe/structures/%d/", structureID)
		if err := ctx.Client.Get(bgCtx, structurePath, ctx.AccessToken, &detail); err != nil {
			global.Logger.Debug("[ESI] 获取个人关联建筑详情失败，跳过",
				zap.Int64("character_id", ctx.CharacterID),
				zap.Int64("structure_id", structureID),
				zap.Error(err),
			)
			continue
		}

		records = append(records, model.EveStructure{
			StructureID:   structureID,
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

	if len(records) == 0 {
		return nil
	}

	if err := global.DB.Clauses(clause.OnConflict{UpdateAll: true}).Create(&records).Error; err != nil {
		return fmt.Errorf("upsert personal structures: %w", err)
	}

	global.Logger.Debug("[ESI] 个人关联建筑详情刷新完成",
		zap.Int64("character_id", ctx.CharacterID),
		zap.Int("count", len(records)),
	)
	return nil
}

func loadPersonalStructureIDs(ctx context.Context, characterID int64) ([]int64, error) {
	idSet := make(map[int64]struct{})

	var assetLocationIDs []int64
	if err := global.DB.WithContext(ctx).
		Model(&model.EveCharacterAsset{}).
		Where("character_id = ? AND location_id > 0", characterID).
		Distinct("location_id").
		Pluck("location_id", &assetLocationIDs).Error; err != nil {
		return nil, err
	}
	for _, locationID := range assetLocationIDs {
		if isLikelyStructureID(locationID) {
			idSet[locationID] = struct{}{}
		}
	}

	var homeLocationIDs []int64
	if err := global.DB.WithContext(ctx).
		Model(&model.EveCharacterCloneBaseInfo{}).
		Where("character_id = ? AND home_location_id > 0 AND LOWER(home_location_type) = ?", characterID, "structure").
		Distinct("home_location_id").
		Pluck("home_location_id", &homeLocationIDs).Error; err != nil {
		return nil, err
	}
	for _, locationID := range homeLocationIDs {
		if isLikelyStructureID(locationID) {
			idSet[locationID] = struct{}{}
		}
	}

	var implantLocations []struct {
		LocationID   int64
		LocationType string
	}
	if err := global.DB.WithContext(ctx).
		Model(&model.EveCharacterImplants{}).
		Select("location_id, location_type").
		Where("character_id = ? AND location_id > 0", characterID).
		Scan(&implantLocations).Error; err != nil {
		return nil, err
	}
	for _, location := range implantLocations {
		if strings.EqualFold(location.LocationType, "structure") && isLikelyStructureID(location.LocationID) {
			idSet[location.LocationID] = struct{}{}
		}
	}

	ids := make([]int64, 0, len(idSet))
	for structureID := range idSet {
		ids = append(ids, structureID)
	}
	sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })
	return ids, nil
}

func isLikelyStructureID(locationID int64) bool {
	return locationID >= minStructureID
}

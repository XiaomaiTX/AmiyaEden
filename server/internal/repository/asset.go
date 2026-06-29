package repository

import (
	"amiya-eden/global"
	"amiya-eden/internal/model"
	"strings"
	"time"

	"gorm.io/gorm/clause"
)

// AssetRepository 人物资产数据访问层
type AssetRepository struct{}

func NewAssetRepository() *AssetRepository { return &AssetRepository{} }

// ─────────────────────────────────────────────
//  位置摘要查询
// ─────────────────────────────────────────────

// AssetLocationSummaryRow 位置摘要查询结果行
type AssetLocationSummaryRow struct {
	LocationID     int64  `gorm:"column:location_id"`
	LocationType   string `gorm:"column:location_type"`
	TopLevelCount  int    `gorm:"column:top_level_count"`
	RootItemCount  int    `gorm:"column:root_item_count"` // 根物品数，非递归总数
	CharacterCount int    `gorm:"column:character_count"`
}

// ListAssetLocationSummaries 查询所有位置摘要（不分页、不过滤关键词）。
// 调用方负责按位置名过滤和分页。
func (r *AssetRepository) ListAssetLocationSummaries(characterIDs []int64) ([]AssetLocationSummaryRow, int64, error) {
	if len(characterIDs) == 0 {
		return nil, 0, nil
	}

	selectClause := `
		SELECT
			loc.location_id,
			loc.location_type,
			COUNT(*) AS top_level_count,
			COUNT(DISTINCT loc.character_id) AS character_count
		FROM eve_character_asset loc
		WHERE loc.character_id IN ?
			AND (
				loc.location_type != 'item'
				OR (
					loc.location_type = 'item'
					AND loc.location_id NOT IN (
						SELECT item_id FROM eve_character_asset sub
						WHERE sub.character_id IN ?
					)
				)
			)`

	groupClause := ` GROUP BY loc.location_id, loc.location_type`
	groupedQuery := selectClause + groupClause

	// 计数：在外层对分组后的子查询做 COUNT(*)
	countSQL := `SELECT COUNT(*) FROM (` + groupedQuery + `) AS grouped`
	var total int64
	args := []interface{}{characterIDs, characterIDs}
	if err := global.DB.Raw(countSQL, args...).Scan(&total).Error; err != nil {
		return nil, 0, err
	}
	if total == 0 {
		return nil, 0, nil
	}

	orderClause := ` ORDER BY top_level_count DESC`

	var rows []AssetLocationSummaryRow
	err := global.DB.Raw(groupedQuery+orderClause, args...).Scan(&rows).Error
	if err != nil {
		return nil, 0, err
	}

	// RootItemCount 当前等于根物品数（非递归总数）
	for i := range rows {
		rows[i].RootItemCount = rows[i].TopLevelCount
	}

	return rows, total, nil
}

// ─────────────────────────────────────────────
//  总资产计数
// ─────────────────────────────────────────────

// CountAssetsByCharacterIDs 统计指定角色集合的资产事实表总条数。
// 不带位置分页，不带位置名过滤，是总资产的稳定计数来源。
func (r *AssetRepository) CountAssetsByCharacterIDs(characterIDs []int64) (int64, error) {
	if len(characterIDs) == 0 {
		return 0, nil
	}
	var count int64
	err := global.DB.Table("eve_character_asset").
		Where("character_id IN ?", characterIDs).
		Count(&count).Error
	return count, err
}

// ─────────────────────────────────────────────
//  位置类型查询
// ─────────────────────────────────────────────

// GetLocationType 查询指定位置在用户资产中的顶层 location_type。
// 同一 location_id 在数据里出现多种类型时优先返回非 item 类型（station/structure/solar_system）。
// 如果只存在 item 类型则返回 "item"，调用方需据此分支解析位置名。
func (r *AssetRepository) GetLocationType(characterIDs []int64, locationID int64) (string, error) {
	if len(characterIDs) == 0 {
		return "", nil
	}

	var locationType string
	err := global.DB.Table("eve_character_asset").
		Select("location_type").
		Where("character_id IN ?", characterIDs).
		Where("location_id = ?", locationID).
		Where("location_type != 'item'").
		Order("location_type ASC").
		Limit(1).
		Scan(&locationType).Error
	if err != nil {
		return "", err
	}
	if locationType != "" {
		return locationType, nil
	}

	// 如果只有 item 类型，回退查询 item
	err = global.DB.Table("eve_character_asset").
		Select("location_type").
		Where("character_id IN ?", characterIDs).
		Where("location_id = ?", locationID).
		Limit(1).
		Scan(&locationType).Error
	return locationType, err
}

// GetAssetByItemID 按 item_id 查询单条资产行（用于解析 item 类型的位置名）。
func (r *AssetRepository) GetAssetByItemID(characterIDs []int64, itemID int64) (*model.EveCharacterAsset, error) {
	if len(characterIDs) == 0 {
		return nil, nil
	}
	var asset model.EveCharacterAsset
	err := global.DB.
		Where("character_id IN ?", characterIDs).
		Where("item_id = ?", itemID).
		First(&asset).Error
	if err != nil {
		return nil, err
	}
	return &asset, nil
}

// GetAssetsByItemIDs 批量按 item_id 查询资产行（用于批量解析 item 类型的位置名）。
func (r *AssetRepository) GetAssetsByItemIDs(characterIDs []int64, itemIDs []int64) ([]model.EveCharacterAsset, error) {
	if len(characterIDs) == 0 || len(itemIDs) == 0 {
		return nil, nil
	}
	var assets []model.EveCharacterAsset
	err := global.DB.
		Where("character_id IN ?", characterIDs).
		Where("item_id IN ?", itemIDs).
		Find(&assets).Error
	if err != nil {
		return nil, err
	}
	return assets, nil
}

// ─────────────────────────────────────────────
//  指定位置根物品查询
// ─────────────────────────────────────────────

// ListRootAssetsByLocation 分页查询指定位置的根物品。
func (r *AssetRepository) ListRootAssetsByLocation(characterIDs []int64, locationID int64, keyword string, page, pageSize int) ([]model.EveCharacterAsset, int64, error) {
	if len(characterIDs) == 0 {
		return nil, 0, nil
	}

	// 根物品: location_type != 'item'
	//   OR (location_type == 'item' AND location_id NOT IN 角色资产的 item_id)
	containerSub := `SELECT DISTINCT sub.item_id FROM eve_character_asset sub WHERE sub.character_id IN ?`

	rootCondition := `(
		location_type != 'item'
		OR (
			location_type = 'item'
			AND location_id NOT IN (` + containerSub + `)
		)
	)`

	baseQuery := global.DB.Table("eve_character_asset").
		Where("character_id IN ?", characterIDs).
		Where("location_id = ?", locationID).
		Where(rootCondition, characterIDs)

	if keyword != "" {
		like := "%" + strings.ToLower(keyword) + "%"
		baseQuery = baseQuery.Where(
			"(LOWER(asset_name) LIKE ? OR CAST(item_id AS TEXT) LIKE ?)",
			like, like,
		)
	}

	var total int64
	if err := baseQuery.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	if total == 0 {
		return nil, 0, nil
	}

	offset := (page - 1) * pageSize
	if offset < 0 {
		offset = 0
	}

	var assets []model.EveCharacterAsset
	err := baseQuery.
		Offset(offset).
		Limit(pageSize).
		Order("item_id ASC").
		Find(&assets).Error
	if err != nil {
		return nil, 0, err
	}

	return assets, total, nil
}

// ─────────────────────────────────────────────
//  子物品查询
// ─────────────────────────────────────────────

// ListAssetChildren 获取指定父物品的直接子物品列表。
func (r *AssetRepository) ListAssetChildren(characterIDs []int64, parentItemID int64) ([]model.EveCharacterAsset, error) {
	if len(characterIDs) == 0 {
		return nil, nil
	}

	var assets []model.EveCharacterAsset
	err := global.DB.
		Where("character_id IN ?", characterIDs).
		Where("location_type = 'item' AND location_id = ?", parentItemID).
		Order("item_id ASC").
		Find(&assets).Error
	return assets, err
}

// ListChildCountsByParentIDs 批量查询各父物品的直接子物品数量。
func (r *AssetRepository) ListChildCountsByParentIDs(characterIDs []int64, parentItemIDs []int64) (map[int64]int, error) {
	result := make(map[int64]int)
	if len(characterIDs) == 0 || len(parentItemIDs) == 0 {
		return result, nil
	}

	type row struct {
		ParentItemID int64 `gorm:"column:parent_item_id"`
		ChildCount   int   `gorm:"column:child_count"`
	}

	var rows []row
	err := global.DB.Table("eve_character_asset").
		Select("location_id AS parent_item_id, COUNT(*) AS child_count").
		Where("character_id IN ?", characterIDs).
		Where("location_type = 'item' AND location_id IN ?", parentItemIDs).
		Group("location_id").
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}

	for _, r := range rows {
		result[r.ParentItemID] = r.ChildCount
	}
	return result, nil
}

// ─────────────────────────────────────────────
//  全量拉取 (保留兼容旧接口)
// ─────────────────────────────────────────────

// GetAssetsByCharacterID 获取人物的所有资产
func (r *AssetRepository) GetAssetsByCharacterID(characterID int64) ([]model.EveCharacterAsset, error) {
	var assets []model.EveCharacterAsset
	err := global.DB.Where("character_id = ?", characterID).Find(&assets).Error
	return assets, err
}

// GetAssetsByCharacterIDs 批量获取多个人物的资产
func (r *AssetRepository) GetAssetsByCharacterIDs(characterIDs []int64) ([]model.EveCharacterAsset, error) {
	var assets []model.EveCharacterAsset
	err := global.DB.Where("character_id IN ?", characterIDs).Find(&assets).Error
	return assets, err
}

// ─────────────────────────────────────────────
//  位置名缓存查询
// ─────────────────────────────────────────────

// GetStructureByID 根据建筑 ID 获取建筑信息。
//
// 资产页需要展示已有建筑名，即使缓存较旧也要返回，因此读取时不按
// update_at 做新鲜度过滤；需要“新鲜缓存”语义的调用方请用 GetFreshStructureByID。
func (r *AssetRepository) GetStructureByID(structureID int64) (*model.EveStructure, error) {
	var s model.EveStructure
	err := global.DB.Where("structure_id = ?", structureID).First(&s).Error
	return &s, err
}

// GetFreshStructureByID 根据建筑 ID 获取近 15 天内更新过的建筑信息。
// 仅供需要“新鲜缓存”语义的场景使用；资产展示请用 GetStructureByID。
func (r *AssetRepository) GetFreshStructureByID(structureID int64) (*model.EveStructure, error) {
	var s model.EveStructure
	fifteenDaysAgo := time.Now().Add(-15 * 24 * time.Hour).Unix()
	err := global.DB.Where("structure_id = ? AND update_at > ?", structureID, fifteenDaysAgo).First(&s).Error
	return &s, err
}

// GetStructuresByIDs 批量根据建筑 ID 获取建筑信息（不按 update_at 过滤，资产展示用）。
func (r *AssetRepository) GetStructuresByIDs(structureIDs []int64) ([]model.EveStructure, error) {
	if len(structureIDs) == 0 {
		return nil, nil
	}
	var structures []model.EveStructure
	err := global.DB.Where("structure_id IN ?", structureIDs).Find(&structures).Error
	return structures, err
}

// GetStationByID 根据空间站 ID 获取空间站信息
func (r *AssetRepository) GetStationByID(stationID int64) (*model.EveStation, error) {
	var s model.EveStation
	err := global.DB.Where("station_id = ?", stationID).First(&s).Error
	return &s, err
}

// UpsertStation 创建或更新空间站信息
func (r *AssetRepository) UpsertStation(s *model.EveStation) error {
	return global.DB.Clauses(clause.OnConflict{UpdateAll: true}).Create(s).Error
}

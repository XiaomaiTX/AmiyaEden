package service

import (
	"amiya-eden/global"
	"amiya-eden/internal/model"
	"amiya-eden/internal/repository"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"go.uber.org/zap"
)

// ─────────────────────────────────────────────
//  请求 & 响应结构
// ─────────────────────────────────────────────

// InfoAssetsRequest 资产请求
type InfoAssetsRequest struct {
	Language string `json:"language"`
}

// AssetLocationNode 前端资产树的「位置节点」
type AssetLocationNode struct {
	LocationID   int64           `json:"location_id"`
	LocationType string          `json:"location_type"` // station / structure / solar_system / other
	LocationName string          `json:"location_name"`
	Items        []AssetItemNode `json:"items"`
}

// AssetItemNode 前端资产树的「物品节点」
type AssetItemNode struct {
	ItemID          int64           `json:"item_id"`
	TypeID          int             `json:"type_id"`
	TypeName        string          `json:"type_name"`
	GroupName       string          `json:"group_name"`
	CategoryID      int             `json:"category_id"`
	Quantity        int             `json:"quantity"`
	LocationFlag    string          `json:"location_flag"`
	IsSingleton     bool            `json:"is_singleton"`
	IsBlueprintCopy *bool           `json:"is_blueprint_copy,omitempty"`
	AssetName       string          `json:"asset_name,omitempty"`
	CharacterID     int64           `json:"character_id"`
	CharacterName   string          `json:"character_name"`
	Children        []AssetItemNode `json:"children,omitempty"`
}

// InfoAssetsResponse 资产响应（旧全量树接口）
type InfoAssetsResponse struct {
	TotalItems int                 `json:"total_items"`
	Locations  []AssetLocationNode `json:"locations"`
}

// ─── 分页资产接口 DTO ───

// AssetLocationsRequest 位置摘要分页请求
type AssetLocationsRequest struct {
	Language string `json:"language"`
	Page     int    `json:"page"`
	PageSize int    `json:"page_size"`
	Keyword  string `json:"keyword"`
}

// AssetLocationSummary 位置摘要
type AssetLocationSummary struct {
	LocationID     int64  `json:"location_id"`
	LocationType   string `json:"location_type"`
	LocationName   string `json:"location_name"`
	TopLevelCount  int    `json:"top_level_count"`
	RootItemCount  int    `json:"root_item_count"` // 根物品数，非递归总数
	CharacterCount int    `json:"character_count"`
}

// AssetLocationsResponse 位置摘要列表响应
type AssetLocationsResponse struct {
	TotalLocations int                    `json:"total_locations"`
	TotalItems     int                    `json:"total_items"`
	Locations      []AssetLocationSummary `json:"locations"`
}

// AssetLocationItemsRequest 位置根物品列表请求
type AssetLocationItemsRequest struct {
	Language   string `json:"language"`
	LocationID int64  `json:"location_id"`
	Page       int    `json:"page"`
	PageSize   int    `json:"page_size"`
	Keyword    string `json:"keyword"`
}

// AssetListItemNode 扁平物品节点（不含递归 children）
type AssetListItemNode struct {
	ItemID          int64  `json:"item_id"`
	TypeID          int    `json:"type_id"`
	TypeName        string `json:"type_name"`
	GroupName       string `json:"group_name"`
	CategoryID      int    `json:"category_id"`
	Quantity        int    `json:"quantity"`
	LocationFlag    string `json:"location_flag"`
	IsSingleton     bool   `json:"is_singleton"`
	IsBlueprintCopy *bool  `json:"is_blueprint_copy,omitempty"`
	AssetName       string `json:"asset_name,omitempty"`
	CharacterID     int64  `json:"character_id"`
	CharacterName   string `json:"character_name"`
	HasChildren     bool   `json:"has_children"`
	ChildCount      int    `json:"child_count"`
}

// AssetLocationItemsResponse 位置根物品列表响应
type AssetLocationItemsResponse struct {
	LocationID     int64               `json:"location_id"`
	LocationName   string              `json:"location_name"`
	TotalRootItems int                 `json:"total_root_items"`
	Items          []AssetListItemNode `json:"items"`
}

// AssetChildrenRequest 子物品列表请求
type AssetChildrenRequest struct {
	Language     string `json:"language"`
	ParentItemID int64  `json:"parent_item_id"`
}

// AssetChildrenResponse 子物品列表响应
type AssetChildrenResponse struct {
	ParentItemID int64               `json:"parent_item_id"`
	Items        []AssetListItemNode `json:"items"`
}

// ─────────────────────────────────────────────
//  Service
// ─────────────────────────────────────────────

// AssetService 资产业务逻辑
type AssetService struct {
	charRepo  *repository.EveCharacterRepository
	assetRepo *repository.AssetRepository
	sdeRepo   *repository.SdeRepository
	ssoSvc    *EveSSOService
	http      *http.Client
}

func NewAssetService() *AssetService {
	return &AssetService{
		charRepo:  repository.NewEveCharacterRepository(),
		assetRepo: repository.NewAssetRepository(),
		sdeRepo:   repository.NewSdeRepository(),
		ssoSvc:    NewEveSSOService(),
		http:      &http.Client{Timeout: 30 * time.Second},
	}
}

// GetUserAssets 获取用户名下所有人物的资产汇总
func (s *AssetService) GetUserAssets(userID uint, req *InfoAssetsRequest) (*InfoAssetsResponse, error) {
	startTotal := time.Now()
	lang := req.Language
	if lang == "" {
		lang = "zh"
	}

	// 1. 获取用户的所有人物
	stageStart := time.Now()
	chars, err := listOwnedCharacters(s.charRepo, userID)
	if err != nil {
		return nil, err
	}
	if len(chars) == 0 {
		return &InfoAssetsResponse{Locations: []AssetLocationNode{}}, nil
	}
	global.Logger.Debug("[Asset] 角色查询完成",
		zap.Uint("user_id", userID),
		zap.Int("character_count", len(chars)),
		zap.Int64("duration_ms", time.Since(stageStart).Milliseconds()),
	)

	charIDs := make([]int64, 0, len(chars))
	charNameMap := make(map[int64]string)
	for _, c := range chars {
		charIDs = append(charIDs, c.CharacterID)
		charNameMap[c.CharacterID] = c.CharacterName
	}

	// 2. 获取所有人物的资产
	stageStart = time.Now()
	allAssets, err := s.assetRepo.GetAssetsByCharacterIDs(charIDs)
	if err != nil {
		return nil, errors.New("获取资产数据失败")
	}
	if len(allAssets) == 0 {
		return &InfoAssetsResponse{Locations: []AssetLocationNode{}}, nil
	}
	global.Logger.Debug("[Asset] 资产查询完成",
		zap.Uint("user_id", userID),
		zap.Int("asset_count", len(allAssets)),
		zap.Int64("duration_ms", time.Since(stageStart).Milliseconds()),
	)

	// 3. 收集所有 typeID 查名称
	stageStart = time.Now()
	typeIDSet := make(map[int]struct{})
	for _, a := range allAssets {
		typeIDSet[a.TypeID] = struct{}{}
	}
	typeIDs := make([]int, 0, len(typeIDSet))
	for id := range typeIDSet {
		typeIDs = append(typeIDs, id)
	}

	typeInfoMap := make(map[int]repository.TypeInfo)
	const typeBatch = 500
	for i := 0; i < len(typeIDs); i += typeBatch {
		end := i + typeBatch
		if end > len(typeIDs) {
			end = len(typeIDs)
		}
		infos, err := s.sdeRepo.GetTypes(typeIDs[i:end], nil, lang)
		if err == nil {
			for _, info := range infos {
				typeInfoMap[info.TypeID] = info
			}
		}
	}
	global.Logger.Debug("[Asset] SDE 类型补全完成",
		zap.Uint("user_id", userID),
		zap.Int("type_count", len(typeIDs)),
		zap.Int64("duration_ms", time.Since(stageStart).Milliseconds()),
	)

	// 4. 构建 item_id -> asset map 以及 parent-child 关系
	stageStart = time.Now()
	itemMap := make(map[int64]*model.EveCharacterAsset)
	for i := range allAssets {
		itemMap[allAssets[i].ItemID] = &allAssets[i]
	}

	// 5. 建立位置分组
	rootLocationIDs := make(map[int64]string)
	childrenMap := make(map[int64][]model.EveCharacterAsset)

	for _, a := range allAssets {
		if a.LocationType == "item" {
			if _, isParentAsset := itemMap[a.LocationID]; isParentAsset {
				childrenMap[a.LocationID] = append(childrenMap[a.LocationID], a)
			} else {
				rootLocationIDs[a.LocationID] = "structure"
			}
		} else {
			rootLocationIDs[a.LocationID] = a.LocationType
		}
	}

	// 6. 解析位置名称（纯本地，不触发 ESI）
	locationNames := make(map[int64]string)
	for locID, locType := range rootLocationIDs {
		locationNames[locID] = s.resolveLocationNameLocal(locID, locType)
	}
	global.Logger.Debug("[Asset] 位置分组与名称解析完成",
		zap.Uint("user_id", userID),
		zap.Int("root_location_count", len(rootLocationIDs)),
		zap.Int64("duration_ms", time.Since(stageStart).Milliseconds()),
	)

	// 7. 按位置分组根物品
	locationItemsMap := make(map[int64][]model.EveCharacterAsset)
	for _, a := range allAssets {
		if a.LocationType != "item" {
			locationItemsMap[a.LocationID] = append(locationItemsMap[a.LocationID], a)
		} else if _, isParentAsset := itemMap[a.LocationID]; !isParentAsset {
			locationItemsMap[a.LocationID] = append(locationItemsMap[a.LocationID], a)
		}
	}

	// 8. 递归构建资产树
	var buildChildren func(parentItemID int64) []AssetItemNode
	buildChildren = func(parentItemID int64) []AssetItemNode {
		children, ok := childrenMap[parentItemID]
		if !ok {
			return nil
		}
		result := make([]AssetItemNode, 0, len(children))
		for _, c := range children {
			tInfo := typeInfoMap[c.TypeID]
			node := AssetItemNode{
				ItemID:          c.ItemID,
				TypeID:          c.TypeID,
				TypeName:        tInfo.TypeName,
				GroupName:       tInfo.GroupName,
				CategoryID:      tInfo.CategoryID,
				Quantity:        c.Quantity,
				LocationFlag:    c.LocationFlag,
				IsSingleton:     c.IsSingleton,
				IsBlueprintCopy: c.IsBlueprintCopy,
				AssetName:       c.AssetName,
				CharacterID:     c.CharacterID,
				CharacterName:   charNameMap[c.CharacterID],
				Children:        buildChildren(c.ItemID),
			}
			result = append(result, node)
		}
		return result
	}

	// 9. 组装最终响应
	locations := make([]AssetLocationNode, 0)
	for locID, items := range locationItemsMap {
		locNode := AssetLocationNode{
			LocationID:   locID,
			LocationType: rootLocationIDs[locID],
			LocationName: locationNames[locID],
			Items:        make([]AssetItemNode, 0, len(items)),
		}
		for _, a := range items {
			tInfo := typeInfoMap[a.TypeID]
			node := AssetItemNode{
				ItemID:          a.ItemID,
				TypeID:          a.TypeID,
				TypeName:        tInfo.TypeName,
				GroupName:       tInfo.GroupName,
				CategoryID:      tInfo.CategoryID,
				Quantity:        a.Quantity,
				LocationFlag:    a.LocationFlag,
				IsSingleton:     a.IsSingleton,
				IsBlueprintCopy: a.IsBlueprintCopy,
				AssetName:       a.AssetName,
				CharacterID:     a.CharacterID,
				CharacterName:   charNameMap[a.CharacterID],
				Children:        buildChildren(a.ItemID),
			}
			locNode.Items = append(locNode.Items, node)
		}
		locations = append(locations, locNode)
	}

	global.Logger.Debug("[Asset] 资产查询总耗时",
		zap.Uint("user_id", userID),
		zap.Int("response_location_count", len(locations)),
		zap.Int("response_item_count", len(allAssets)),
		zap.Int64("total_duration_ms", time.Since(startTotal).Milliseconds()),
	)

	return &InfoAssetsResponse{
		TotalItems: len(allAssets),
		Locations:  locations,
	}, nil
}

// ─────────────────────────────────────────────
//  分页资产查询 (新接口)
// ─────────────────────────────────────────────

// GetUserAssetLocations 分页查询用户的资产位置摘要。
func (s *AssetService) GetUserAssetLocations(userID uint, req *AssetLocationsRequest) (*AssetLocationsResponse, error) {
	if req.Page <= 0 {
		req.Page = 1
	}
	if req.PageSize <= 0 || req.PageSize > 100 {
		req.PageSize = 20
	}

	chars, err := listOwnedCharacters(s.charRepo, userID)
	if err != nil {
		return nil, err
	}
	if len(chars) == 0 {
		return &AssetLocationsResponse{Locations: []AssetLocationSummary{}}, nil
	}

	charIDs := make([]int64, len(chars))
	for i, c := range chars {
		charIDs[i] = c.CharacterID
	}

	// 1. 获取全部位置摘要（不分页，不过滤关键词）
	allSummaries, _, err := s.assetRepo.ListAssetLocationSummaries(charIDs)
	if err != nil {
		return nil, fmt.Errorf("查询位置摘要失败: %w", err)
	}

	// 2. 补全位置名
	type enriched struct {
		AssetLocationSummary
		nameLower string
	}

	// 收集所有 item 类型的位置 ID，批量解析
	itemIDs := make([]int64, 0)
	for _, row := range allSummaries {
		if row.LocationType == "item" {
			itemIDs = append(itemIDs, row.LocationID)
		}
	}
	itemNameMap := s.resolveItemLocationNamesLocal(charIDs, itemIDs, req.Language)

	enrichedList := make([]enriched, len(allSummaries))
	for i, row := range allSummaries {
		var name string
		if row.LocationType == "item" {
			name = itemNameMap[row.LocationID]
			if name == "" {
				name = fmt.Sprintf("Item-%d", row.LocationID)
			}
		} else {
			name = s.resolveLocationNameLocal(row.LocationID, row.LocationType)
		}
		enrichedList[i] = enriched{
			AssetLocationSummary: AssetLocationSummary{
				LocationID:     row.LocationID,
				LocationType:   row.LocationType,
				LocationName:   name,
				TopLevelCount:  row.TopLevelCount,
				RootItemCount:  row.RootItemCount,
				CharacterCount: row.CharacterCount,
			},
			nameLower: strings.ToLower(name),
		}
	}

	// 3. 按位置名过滤关键词
	keyword := strings.ToLower(strings.TrimSpace(req.Keyword))
	if keyword != "" {
		filtered := make([]enriched, 0, len(enrichedList))
		for _, e := range enrichedList {
			if strings.Contains(e.nameLower, keyword) {
				filtered = append(filtered, e)
			}
		}
		enrichedList = filtered
	}

	// 4. 获取总资产条目数（不受位置分页和位置名搜索影响）
	totalItems, err := s.assetRepo.CountAssetsByCharacterIDs(charIDs)
	if err != nil {
		return nil, fmt.Errorf("查询资产总数失败: %w", err)
	}

	// 5. 计算过滤后的位置总数
	totalLocations := len(enrichedList)

	// 6. 服务层分页
	start := (req.Page - 1) * req.PageSize
	end := start + req.PageSize
	if start > len(enrichedList) {
		start = len(enrichedList)
	}
	if end > len(enrichedList) {
		end = len(enrichedList)
	}

	paged := enrichedList[start:end]

	// 7. 组装响应
	locations := make([]AssetLocationSummary, len(paged))
	for i, e := range paged {
		locations[i] = e.AssetLocationSummary
	}

	return &AssetLocationsResponse{
		TotalLocations: totalLocations,
		TotalItems:     int(totalItems),
		Locations:      locations,
	}, nil
}

// GetUserAssetLocationItems 分页查询指定位置的根物品。
func (s *AssetService) GetUserAssetLocationItems(userID uint, req *AssetLocationItemsRequest) (*AssetLocationItemsResponse, error) {
	if req.Page <= 0 {
		req.Page = 1
	}
	if req.PageSize <= 0 || req.PageSize > 100 {
		req.PageSize = 50
	}

	chars, err := listOwnedCharacters(s.charRepo, userID)
	if err != nil {
		return nil, err
	}
	if len(chars) == 0 {
		return &AssetLocationItemsResponse{Items: []AssetListItemNode{}}, nil
	}

	charIDs := make([]int64, len(chars))
	charNameMap := make(map[int64]string)
	for i, c := range chars {
		charIDs[i] = c.CharacterID
		charNameMap[c.CharacterID] = c.CharacterName
	}

	assets, total, err := s.assetRepo.ListRootAssetsByLocation(charIDs, req.LocationID, req.Keyword, req.Page, req.PageSize)
	if err != nil {
		return nil, fmt.Errorf("查询位置物品失败: %w", err)
	}

	// 补全类型名
	typeIDs := collectTypeIDs(assets)
	typeInfoMap := s.batchGetTypeInfo(typeIDs, req.Language)

	// 批量查询子物品数量
	parentIDs := make([]int64, len(assets))
	for i, a := range assets {
		parentIDs[i] = a.ItemID
	}
	childCounts, _ := s.assetRepo.ListChildCountsByParentIDs(charIDs, parentIDs)

	// 组装响应
	items := make([]AssetListItemNode, len(assets))
	for i, a := range assets {
		tInfo := typeInfoMap[a.TypeID]
		count := childCounts[a.ItemID]
		items[i] = AssetListItemNode{
			ItemID:          a.ItemID,
			TypeID:          a.TypeID,
			TypeName:        tInfo.TypeName,
			GroupName:       tInfo.GroupName,
			CategoryID:      tInfo.CategoryID,
			Quantity:        a.Quantity,
			LocationFlag:    a.LocationFlag,
			IsSingleton:     a.IsSingleton,
			IsBlueprintCopy: a.IsBlueprintCopy,
			AssetName:       a.AssetName,
			CharacterID:     a.CharacterID,
			CharacterName:   charNameMap[a.CharacterID],
			HasChildren:     count > 0,
			ChildCount:      count,
		}
	}

	locationType, err := s.assetRepo.GetLocationType(charIDs, req.LocationID)
	if err != nil {
		global.Logger.Warn("[Asset] 查询位置类型失败",
			zap.Int64("location_id", req.LocationID),
			zap.Error(err),
		)
	}
	if locationType == "" {
		locationType = "other"
	}

	var locationName string
	switch locationType {
	case "item":
		locationName = s.resolveItemLocationNameLocal(charIDs, req.LocationID, req.Language)
	default:
		locationName = s.resolveLocationNameLocal(req.LocationID, locationType)
	}

	return &AssetLocationItemsResponse{
		LocationID:     req.LocationID,
		LocationName:   locationName,
		TotalRootItems: int(total),
		Items:          items,
	}, nil
}

// GetUserAssetChildren 查询指定父物品的直接子物品。
func (s *AssetService) GetUserAssetChildren(userID uint, req *AssetChildrenRequest) (*AssetChildrenResponse, error) {
	chars, err := listOwnedCharacters(s.charRepo, userID)
	if err != nil {
		return nil, err
	}
	if len(chars) == 0 {
		return &AssetChildrenResponse{Items: []AssetListItemNode{}}, nil
	}

	charIDs := make([]int64, len(chars))
	charNameMap := make(map[int64]string)
	for i, c := range chars {
		charIDs[i] = c.CharacterID
		charNameMap[c.CharacterID] = c.CharacterName
	}

	children, err := s.assetRepo.ListAssetChildren(charIDs, req.ParentItemID)
	if err != nil {
		return nil, fmt.Errorf("查询子物品失败: %w", err)
	}

	// 补全类型名
	typeIDs := collectTypeIDs(children)
	typeInfoMap := s.batchGetTypeInfo(typeIDs, req.Language)

	// 批量查询孙子物品数量
	parentIDs := make([]int64, len(children))
	for i, c := range children {
		parentIDs[i] = c.ItemID
	}
	childCounts, _ := s.assetRepo.ListChildCountsByParentIDs(charIDs, parentIDs)

	items := make([]AssetListItemNode, len(children))
	for i, c := range children {
		tInfo := typeInfoMap[c.TypeID]
		count := childCounts[c.ItemID]
		items[i] = AssetListItemNode{
			ItemID:          c.ItemID,
			TypeID:          c.TypeID,
			TypeName:        tInfo.TypeName,
			GroupName:       tInfo.GroupName,
			CategoryID:      tInfo.CategoryID,
			Quantity:        c.Quantity,
			LocationFlag:    c.LocationFlag,
			IsSingleton:     c.IsSingleton,
			IsBlueprintCopy: c.IsBlueprintCopy,
			AssetName:       c.AssetName,
			CharacterID:     c.CharacterID,
			CharacterName:   charNameMap[c.CharacterID],
			HasChildren:     count > 0,
			ChildCount:      count,
		}
	}

	return &AssetChildrenResponse{
		ParentItemID: req.ParentItemID,
		Items:        items,
	}, nil
}

// collectTypeIDs 收集资产中的 typeID 集合
func collectTypeIDs(assets []model.EveCharacterAsset) []int {
	set := make(map[int]struct{})
	for _, a := range assets {
		set[a.TypeID] = struct{}{}
	}
	ids := make([]int, 0, len(set))
	for id := range set {
		ids = append(ids, id)
	}
	return ids
}

// batchGetTypeInfo 批量查询 type 信息
func (s *AssetService) batchGetTypeInfo(typeIDs []int, lang string) map[int]repository.TypeInfo {
	result := make(map[int]repository.TypeInfo)
	if len(typeIDs) == 0 {
		return result
	}
	if lang == "" {
		lang = "zh"
	}

	const batch = 500
	for i := 0; i < len(typeIDs); i += batch {
		end := i + batch
		if end > len(typeIDs) {
			end = len(typeIDs)
		}
		infos, err := s.sdeRepo.GetTypes(typeIDs[i:end], nil, lang)
		if err == nil {
			for _, info := range infos {
				result[info.TypeID] = info
			}
		}
	}
	return result
}

// ─────────────────────────────────────────────
//  位置解析
// ─────────────────────────────────────────────

// resolveLocationNameLocal 纯本地解析位置名称，不触发 ESI 请求。
// 缓存缺失时返回占位名，不阻塞查询。
func (s *AssetService) resolveLocationNameLocal(locationID int64, locationType string) string {
	if locationID == 0 {
		return ""
	}

	switch locationType {
	case "station":
		return s.resolveStationNameLocal(locationID)
	case "solar_system":
		names, err := s.sdeRepo.GetNames(map[string][]int{
			"solar_system": {int(locationID)},
		}, "zh")
		if err == nil {
			if solarNames, ok := names["solar_system"]; ok {
				if name, ok := solarNames[int(locationID)]; ok {
					return name
				}
			}
		}
		return fmt.Sprintf("System-%d", locationID)
	case "structure", "other":
		return s.resolveStructureNameLocal(locationID)
	default:
		return s.resolveStructureNameLocal(locationID)
	}
}

// resolveItemLocationNameLocal 解析 item 类型的位置名（容器/嵌套资产）。
// 解析顺序：
//  1. 查父资产行，取 asset_name
//  2. asset_name 为空时，用父资产 type_id 查 SDE type_name
//  3. 两者都失败时回退为 Item-<itemID>
func (s *AssetService) resolveItemLocationNameLocal(charIDs []int64, itemID int64, language string) string {
	parent, err := s.assetRepo.GetAssetByItemID(charIDs, itemID)
	if err != nil || parent == nil {
		return fmt.Sprintf("Item-%d", itemID)
	}

	if parent.AssetName != "" {
		return parent.AssetName
	}

	if language == "" {
		language = "zh"
	}
	infos, err := s.sdeRepo.GetTypes([]int{parent.TypeID}, nil, language)
	if err == nil && len(infos) > 0 && infos[0].TypeName != "" {
		return infos[0].TypeName
	}

	return fmt.Sprintf("Item-%d", itemID)
}

// resolveItemLocationNamesLocal 批量解析 item 类型的位置名。
// 内部做两段批量查询：先查父资产行，再对 asset_name 为空的补查 SDE type_name。
func (s *AssetService) resolveItemLocationNamesLocal(charIDs []int64, itemIDs []int64, language string) map[int64]string {
	result := make(map[int64]string, len(itemIDs))
	if len(itemIDs) == 0 {
		return result
	}

	parents, err := s.assetRepo.GetAssetsByItemIDs(charIDs, itemIDs)
	if err != nil || len(parents) == 0 {
		for _, id := range itemIDs {
			result[id] = fmt.Sprintf("Item-%d", id)
		}
		return result
	}

	// 第一遍：有 asset_name 的直接用，没 asset_name 的收集 type_id
	typeIDs := make([]int, 0)
	parentByItemID := make(map[int64]*model.EveCharacterAsset, len(parents))
	for i := range parents {
		parentByItemID[parents[i].ItemID] = &parents[i]
	}

	for _, itemID := range itemIDs {
		parent, ok := parentByItemID[itemID]
		if !ok {
			result[itemID] = fmt.Sprintf("Item-%d", itemID)
			continue
		}
		if parent.AssetName != "" {
			result[itemID] = parent.AssetName
		} else {
			typeIDs = append(typeIDs, parent.TypeID)
		}
	}

	// 第二遍：对没有 asset_name 的，批量查 SDE type_name
	if len(typeIDs) > 0 {
		if language == "" {
			language = "zh"
		}
		infos, err := s.sdeRepo.GetTypes(typeIDs, nil, language)
		if err == nil {
			typeNameByTypeID := make(map[int]string, len(infos))
			for _, info := range infos {
				if info.TypeName != "" {
					typeNameByTypeID[info.TypeID] = info.TypeName
				}
			}
			for _, itemID := range itemIDs {
				if _, alreadyResolved := result[itemID]; alreadyResolved {
					continue
				}
				parent, ok := parentByItemID[itemID]
				if !ok {
					continue
				}
				if name, ok := typeNameByTypeID[parent.TypeID]; ok {
					result[itemID] = name
				} else {
					result[itemID] = fmt.Sprintf("Item-%d", itemID)
				}
			}
		} else {
			// SDE 查询失败时，给剩余未解析的补 fallback
			for _, itemID := range itemIDs {
				if _, resolved := result[itemID]; !resolved {
					result[itemID] = fmt.Sprintf("Item-%d", itemID)
				}
			}
		}
	}

	return result
}

// resolveStationNameLocal 查本地缓存/SDE 获取空间站名称，不触发 ESI。
func (s *AssetService) resolveStationNameLocal(stationID int64) string {
	station, err := s.assetRepo.GetStationByID(stationID)
	if err == nil && station.StationName != "" {
		return station.StationName
	}

	var name string
	if err := global.DB.Table(`"staStations"`).
		Select(`"stationName"`).
		Where(`"stationID" = ?`, stationID).
		Scan(&name).Error; err == nil && name != "" {
		return name
	}

	return fmt.Sprintf("Station-%d", stationID)
}

// resolveStructureNameLocal 查本地缓存获取建筑名称，不触发 ESI。
func (s *AssetService) resolveStructureNameLocal(structureID int64) string {
	structure, err := s.assetRepo.GetStructureByID(structureID)
	if err == nil && structure.StructureName != "" {
		return structure.StructureName
	}
	return fmt.Sprintf("Structure-%d", structureID)
}



// Deprecated: fetchAndCacheStation 同步请求 ESI /universe/stations/。
// 异步补全使用 EnsureAssetLocationsCached。
func (s *AssetService) fetchAndCacheStation(stationID int64) string {
	type stationDetail struct {
		Name          string `json:"name"`
		Owner         int64  `json:"owner"`
		SolarSystemID int64  `json:"system_id"`
		TypeID        int64  `json:"type_id"`
		Position      struct {
			X float64 `json:"x"`
			Y float64 `json:"y"`
			Z float64 `json:"z"`
		} `json:"position"`
	}

	var detail stationDetail
	path := fmt.Sprintf("/universe/stations/%d/", stationID)
	if err := s.esiGetPublic(context.Background(), path, &detail); err != nil {
		global.Logger.Warn("[Asset] 获取空间站详情失败",
			zap.Int64("station_id", stationID),
			zap.Error(err),
		)
		return fmt.Sprintf("Station-%d", stationID)
	}

	if err := s.assetRepo.UpsertStation(&model.EveStation{
		StationID:     stationID,
		StationName:   detail.Name,
		OwnerID:       detail.Owner,
		TypeID:        detail.TypeID,
		SolarSystemID: detail.SolarSystemID,
		X:             detail.Position.X,
		Y:             detail.Position.Y,
		Z:             detail.Position.Z,
		UpdateAt:      time.Now().Unix(),
	}); err != nil {
		global.Logger.Warn("[Asset] 缓存空间站信息失败", zap.Int64("station_id", stationID), zap.Error(err))
	}

	return detail.Name
}


// ─────────────────────────────────────────────
//  异步位置名缓存补全
// ─────────────────────────────────────────────

// EnsureAssetLocationsCached 异步补全指定角色资产涉及的位置名缓存。
// 调用后立即返回，在 goroutine 中执行 ESI 查询。
// 仅补全 NPC 空间站（公共 ESI 无需 token），建筑需等后续有 token 上下文时再补全。
func (s *AssetService) EnsureAssetLocationsCached(characterIDs []int64) {
	go func() {
		if len(characterIDs) == 0 {
			return
		}

		type locRow struct {
			LocationID   int64  `gorm:"column:location_id"`
			LocationType string `gorm:"column:location_type"`
		}

		var locs []locRow
		qErr := global.DB.Table("eve_character_asset").
			Select("DISTINCT location_id, location_type").
			Where("character_id IN ?", characterIDs).
			Where(`location_type = 'station'`).
			Find(&locs).Error
		if qErr != nil {
			global.Logger.Warn("[Asset] 异步补全位置名失败：查询位置列表出错", zap.Error(qErr))
			return
		}

		for _, loc := range locs {
			station, err := s.assetRepo.GetStationByID(loc.LocationID)
			if err == nil && station.StationName != "" {
				continue
			}
			s.fetchAndCacheStation(loc.LocationID)
		}
	}()
}

// ─────────────────────────────────────────────
//  ESI HTTP 辅助
// ─────────────────────────────────────────────

func (s *AssetService) esiGetPublic(ctx context.Context, path string, out interface{}) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, global.Config.EveSSO.ESIBaseURL+path, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Accept", "application/json")

	resp, err := s.http.Do(req)
	if err != nil {
		return err
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			global.Logger.Warn("[Asset] 关闭响应体失败", zap.Error(err))
		}
	}()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("ESI GET %s 返回 %d: %s", path, resp.StatusCode, string(body))
	}

	return json.NewDecoder(resp.Body).Decode(out)
}

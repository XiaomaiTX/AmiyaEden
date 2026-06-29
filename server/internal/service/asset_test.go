package service

import (
	"amiya-eden/global"
	"amiya-eden/internal/model"
	"fmt"
	"testing"

	"gorm.io/gorm"
)

func setupAssetTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	prevDB := global.DB
	db := newServiceTestDB(t, "asset",
		&model.User{},
		&model.EveCharacter{},
		&model.EveCharacterAsset{},
		&model.EveStation{},
		&model.EveStructure{},
	)
	global.DB = db
	t.Cleanup(func() { global.DB = prevDB })

	// 创建用户和角色
	db.Create(&model.User{BaseModel: model.BaseModel{ID: 1}, Nickname: "testuser", PrimaryCharacterID: 1001})
	db.Create(&model.EveCharacter{
		CharacterID:   1001,
		CharacterName: "Amiya",
		UserID:        1,
		TokenInvalid:  false,
	})

	// 创建空间站缓存
	db.Create(&model.EveStation{
		StationID:   60003760,
		StationName: "Jita IV - Moon 4 - Caldari Navy Assembly Plant",
		UpdateAt:    9999999999,
	})

	return db
}

func newAssetService() *AssetService {
	return NewAssetService()
}

func TestGetUserAssetLocations_FilterKeywordByLocationName(t *testing.T) {
	setupAssetTestDB(t)
	svc := newAssetService()

	// 插入资产：位置 60003760 (station, Jita) 有 2 个物品
	global.DB.Create(&model.EveCharacterAsset{
		ItemID:       1,
		CharacterID:  1001,
		LocationID:   60003760,
		LocationType: "station",
		TypeID:       34,
		Quantity:     100,
	})
	global.DB.Create(&model.EveCharacterAsset{
		ItemID:       2,
		CharacterID:  1001,
		LocationID:   60003760,
		LocationType: "station",
		TypeID:       35,
		Quantity:     200,
	})

	// 查询关键词 "Jita" 应匹配
	result, err := svc.GetUserAssetLocations(1, &AssetLocationsRequest{
		Page:     1,
		PageSize: 20,
		Keyword:  "Jita",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.TotalLocations != 1 {
		t.Fatalf("expected 1 location matching 'Jita', got %d", result.TotalLocations)
	}
	if len(result.Locations) != 1 {
		t.Fatalf("expected 1 location in page, got %d", len(result.Locations))
	}
	if result.Locations[0].LocationName != "Jita IV - Moon 4 - Caldari Navy Assembly Plant" {
		t.Fatalf("expected Jita station name, got %q", result.Locations[0].LocationName)
	}
	if result.Locations[0].TopLevelCount != 2 {
		t.Fatalf("expected 2 top-level items, got %d", result.Locations[0].TopLevelCount)
	}

	// 不匹配的关键词应返回 0
	result2, err := svc.GetUserAssetLocations(1, &AssetLocationsRequest{
		Page:     1,
		PageSize: 20,
		Keyword:  "Amarr",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result2.TotalLocations != 0 {
		t.Fatalf("expected 0 locations matching 'Amarr', got %d", result2.TotalLocations)
	}
}

func TestGetUserAssetLocationItems_UsesRealLocationType(t *testing.T) {
	setupAssetTestDB(t)
	svc := newAssetService()

	// 插入资产
	global.DB.Create(&model.EveCharacterAsset{
		ItemID:       1,
		CharacterID:  1001,
		LocationID:   60003760,
		LocationType: "station",
		TypeID:       34,
		Quantity:     100,
	})

	result, err := svc.GetUserAssetLocationItems(1, &AssetLocationItemsRequest{
		LocationID: 60003760,
		Page:       1,
		PageSize:   50,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.LocationID != 60003760 {
		t.Fatalf("expected location_id 60003760, got %d", result.LocationID)
	}
	// 不应使用空类型 fallback "Structure-60003760"
	if result.LocationName != "Jita IV - Moon 4 - Caldari Navy Assembly Plant" {
		t.Fatalf("expected Jita station name, got %q", result.LocationName)
	}
}

func TestGetUserAssetLocations_EmptyResult(t *testing.T) {
	setupAssetTestDB(t)
	svc := newAssetService()

	// 无资产数据
	result, err := svc.GetUserAssetLocations(1, &AssetLocationsRequest{
		Page:     1,
		PageSize: 20,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.TotalLocations != 0 {
		t.Fatalf("expected 0 locations, got %d", result.TotalLocations)
	}
	if len(result.Locations) != 0 {
		t.Fatalf("expected empty locations slice, got %d items", len(result.Locations))
	}
}

func TestGetUserAssetLocationItems_ResolvesItemLocationNameFromAssetName(t *testing.T) {
	setupAssetTestDB(t)
	svc := newAssetService()

	// 父资产（容器）：item_id=9001, 在 Jita 空间站内, 有自定义名
	global.DB.Create(&model.EveCharacterAsset{
		ItemID:       9001,
		CharacterID:  1001,
		LocationID:   60003760,
		LocationType: "station",
		TypeID:       34,
		Quantity:     1,
		AssetName:    "My Container",
	})
	// 子资产：在容器 9001 内, location_type='item'
	global.DB.Create(&model.EveCharacterAsset{
		ItemID:       9002,
		CharacterID:  1001,
		LocationID:   9001,
		LocationType: "item",
		TypeID:       35,
		Quantity:     100,
	})

	result, err := svc.GetUserAssetLocationItems(1, &AssetLocationItemsRequest{
		LocationID: 9001,
		Page:       1,
		PageSize:   50,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.LocationName != "My Container" {
		t.Fatalf("expected parent asset_name 'My Container', got %q", result.LocationName)
	}
}

func TestGetUserAssetLocationItems_ResolvesItemLocationNameFallbackToTypeName(t *testing.T) {
	setupAssetTestDB(t)
	svc := newAssetService()

	// 父资产（容器）：无 asset_name, type_id=34
	global.DB.Create(&model.EveCharacterAsset{
		ItemID:       9001,
		CharacterID:  1001,
		LocationID:   60003760,
		LocationType: "station",
		TypeID:       34,
		Quantity:     1,
	})
	// 子资产：在容器 9001 内
	global.DB.Create(&model.EveCharacterAsset{
		ItemID:       9002,
		CharacterID:  1001,
		LocationID:   9001,
		LocationType: "item",
		TypeID:       35,
		Quantity:     100,
	})

	result, err := svc.GetUserAssetLocationItems(1, &AssetLocationItemsRequest{
		LocationID: 9001,
		Page:       1,
		PageSize:   50,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// 核心回归守卫：不应回退到 Structure 占位名
	if result.LocationName == fmt.Sprintf("Structure-%d", 9001) {
		t.Fatalf("should not fall back to structure placeholder, got %q", result.LocationName)
	}
	if result.LocationName == "" {
		t.Fatalf("expected non-empty location name")
	}
	// SDE type_name 仅在集成环境中可解析；单测回退到 Item-<id> 也可接受
	if result.LocationName == fmt.Sprintf("Item-%d", 9001) {
		t.Log("SDE not available in test DB, fallback to Item-9001 is acceptable")
	}
}

func TestGetUserAssetLocations_TotalItemsStableAcrossPagination(t *testing.T) {
	setupAssetTestDB(t)
	svc := newAssetService()

	// 构造三个位置，共 5 条资产记录
	global.DB.Create(&model.EveCharacterAsset{
		ItemID: 1, CharacterID: 1001, LocationID: 60003760, LocationType: "station", TypeID: 34, Quantity: 100,
	})
	global.DB.Create(&model.EveCharacterAsset{
		ItemID: 2, CharacterID: 1001, LocationID: 60003760, LocationType: "station", TypeID: 35, Quantity: 200,
	})
	global.DB.Create(&model.EveCharacterAsset{
		ItemID: 3, CharacterID: 1001, LocationID: 60003761, LocationType: "station", TypeID: 36, Quantity: 50,
	})
	global.DB.Create(&model.EveCharacterAsset{
		ItemID: 4, CharacterID: 1001, LocationID: 60003762, LocationType: "station", TypeID: 37, Quantity: 10,
	})
	global.DB.Create(&model.EveCharacterAsset{
		ItemID: 5, CharacterID: 1001, LocationID: 60003762, LocationType: "station", TypeID: 38, Quantity: 20,
	})

	// 第一页
	page1, err := svc.GetUserAssetLocations(1, &AssetLocationsRequest{
		Page: 1, PageSize: 2,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if page1.TotalItems != 5 {
		t.Fatalf("expected total_items=5 (all asset rows), got %d", page1.TotalItems)
	}
	if page1.TotalLocations != 3 {
		t.Fatalf("expected total_locations=3, got %d", page1.TotalLocations)
	}
	if len(page1.Locations) != 2 {
		t.Fatalf("expected page size 2, got %d locations", len(page1.Locations))
	}

	// 翻到第二页，total_items 必须不变
	page2, err := svc.GetUserAssetLocations(1, &AssetLocationsRequest{
		Page: 2, PageSize: 2,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if page2.TotalItems != 5 {
		t.Fatalf("expected total_items=5 on page 2, got %d", page2.TotalItems)
	}
	if page2.TotalLocations != 3 {
		t.Fatalf("expected total_locations=3 on page 2, got %d", page2.TotalLocations)
	}
}

func TestGetUserAssetLocations_TotalItemsStableWithSearch(t *testing.T) {
	setupAssetTestDB(t)
	svc := newAssetService()

	// 两个位置：Jita (60003760) 有 2 条资产，Amarr (60003761) 有 1 条
	global.DB.Create(&model.EveCharacterAsset{
		ItemID: 1, CharacterID: 1001, LocationID: 60003760, LocationType: "station", TypeID: 34, Quantity: 100,
	})
	global.DB.Create(&model.EveCharacterAsset{
		ItemID: 2, CharacterID: 1001, LocationID: 60003760, LocationType: "station", TypeID: 35, Quantity: 200,
	})
	global.DB.Create(&model.EveCharacterAsset{
		ItemID: 3, CharacterID: 1001, LocationID: 60003761, LocationType: "station", TypeID: 36, Quantity: 50,
	})

	result, err := svc.GetUserAssetLocations(1, &AssetLocationsRequest{
		Page: 1, PageSize: 20, Keyword: "Jita",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// 搜索 Jita 只返回一个位置
	if result.TotalLocations != 1 {
		t.Fatalf("expected 1 matching location, got %d", result.TotalLocations)
	}
	if len(result.Locations) != 1 {
		t.Fatalf("expected 1 location in result, got %d", len(result.Locations))
	}
	// total_items 仍等于该用户全部资产条目总数（3），不受搜索影响
	if result.TotalItems != 3 {
		t.Fatalf("expected total_items=3 (all user assets), got %d", result.TotalItems)
	}
}

func TestGetUserAssetLocations_TopLevelItemShowsStructureName(t *testing.T) {
	setupAssetTestDB(t)
	svc := newAssetService()

	// 玩家建筑：eve_structures 缓存中存在同 ID 名称
	structureID := int64(1052703947117)
	global.DB.Create(&model.EveStructure{
		StructureID:   structureID,
		StructureName: "Amiya's Fortizar",
		TypeID:        35832,
		UpdateAt:      9999999999,
	})
	// 顶层资产：location_type=item, location_id 指向建筑，且无对应 item_id 行
	// （建筑不会作为资产 item 出现，因此该位置是顶层位置而非容器）
	global.DB.Create(&model.EveCharacterAsset{
		ItemID:       1,
		CharacterID:  1001,
		LocationID:   structureID,
		LocationType: "item",
		TypeID:       34,
		Quantity:     100,
	})

	result, err := svc.GetUserAssetLocations(1, &AssetLocationsRequest{
		Page: 1, PageSize: 20,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Locations) != 1 {
		t.Fatalf("expected 1 location, got %d", len(result.Locations))
	}
	if result.Locations[0].LocationName != "Amiya's Fortizar" {
		t.Fatalf("expected structure name, got %q", result.Locations[0].LocationName)
	}
	if result.Locations[0].LocationName == fmt.Sprintf("Item-%d", structureID) {
		t.Fatalf("should not fall back to Item-<id> when structure is cached")
	}
	// 命中建筑缓存时 location_type 规范为 structure
	if result.Locations[0].LocationType != "structure" {
		t.Fatalf("expected location_type=structure, got %q", result.Locations[0].LocationType)
	}
}

func TestGetUserAssetLocations_TopLevelItemShowsStructureNameWithStaleCache(t *testing.T) {
	setupAssetTestDB(t)
	svc := newAssetService()

	// 建筑缓存 update_at 为旧时间戳，资产页仍应显示建筑名
	structureID := int64(1052703947118)
	global.DB.Create(&model.EveStructure{
		StructureID:   structureID,
		StructureName: "Stale Astrahus",
		TypeID:        35825,
		UpdateAt:      1, // 远早于 15 天阈值
	})
	global.DB.Create(&model.EveCharacterAsset{
		ItemID:       1,
		CharacterID:  1001,
		LocationID:   structureID,
		LocationType: "item",
		TypeID:       34,
		Quantity:     100,
	})

	result, err := svc.GetUserAssetLocations(1, &AssetLocationsRequest{
		Page: 1, PageSize: 20,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Locations) != 1 {
		t.Fatalf("expected 1 location, got %d", len(result.Locations))
	}
	if result.Locations[0].LocationName != "Stale Astrahus" {
		t.Fatalf("expected structure name from stale cache, got %q", result.Locations[0].LocationName)
	}
}

func TestGetUserAssetLocationItems_TopLevelItemShowsStructureName(t *testing.T) {
	setupAssetTestDB(t)
	svc := newAssetService()

	// 玩家建筑缓存
	structureID := int64(1052703947117)
	global.DB.Create(&model.EveStructure{
		StructureID:   structureID,
		StructureName: "Amiya's Fortizar",
		TypeID:        35832,
		UpdateAt:      9999999999,
	})
	// 顶层资产：展开该位置时标题应显示建筑名
	global.DB.Create(&model.EveCharacterAsset{
		ItemID:       1,
		CharacterID:  1001,
		LocationID:   structureID,
		LocationType: "item",
		TypeID:       34,
		Quantity:     100,
	})

	result, err := svc.GetUserAssetLocationItems(1, &AssetLocationItemsRequest{
		LocationID: structureID,
		Page:       1,
		PageSize:   50,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.LocationName != "Amiya's Fortizar" {
		t.Fatalf("expected structure name as location title, got %q", result.LocationName)
	}
	if result.LocationName == fmt.Sprintf("Item-%d", structureID) {
		t.Fatalf("should not fall back to Item-<id> when structure is cached")
	}
}

func TestResolveItemLocationNamesLocal_BatchPriority(t *testing.T) {
	setupAssetTestDB(t)
	svc := newAssetService()

	// 构造最小 SDE 类型表，让 type_name 分支可被真实解析
	for _, stmt := range []string{
		`CREATE TABLE invCategories (categoryID INTEGER PRIMARY KEY, categoryName TEXT)`,
		`CREATE TABLE invGroups (groupID INTEGER PRIMARY KEY, categoryID INTEGER, groupName TEXT)`,
		`CREATE TABLE invMarketGroups (marketGroupID INTEGER PRIMARY KEY, marketGroupName TEXT)`,
		`CREATE TABLE invTypes (typeID INTEGER PRIMARY KEY, groupID INTEGER, typeName TEXT, marketGroupID INTEGER, published INTEGER)`,
		`CREATE TABLE trnTranslations (tcID INTEGER, keyID INTEGER, languageID TEXT, text TEXT, PRIMARY KEY (tcID, keyID, languageID))`,
	} {
		if err := global.DB.Exec(stmt).Error; err != nil {
			t.Fatalf("create sde table failed: %v", err)
		}
	}
	if err := global.DB.Exec(`INSERT INTO invCategories (categoryID, categoryName) VALUES (?, ?)`, 1, "Test Category").Error; err != nil {
		t.Fatalf("insert invCategories failed: %v", err)
	}
	if err := global.DB.Exec(`INSERT INTO invGroups (groupID, categoryID, groupName) VALUES (?, ?, ?)`, 2, 1, "Test Group").Error; err != nil {
		t.Fatalf("insert invGroups failed: %v", err)
	}
	if err := global.DB.Exec(`INSERT INTO invMarketGroups (marketGroupID, marketGroupName) VALUES (?, ?)`, 3, "Test Market").Error; err != nil {
		t.Fatalf("insert invMarketGroups failed: %v", err)
	}
	if err := global.DB.Exec(`INSERT INTO invTypes (typeID, groupID, typeName, marketGroupID, published) VALUES (?, ?, ?, ?, ?)`, 34, 2, "Tritanium", 3, 1).Error; err != nil {
		t.Fatalf("insert invTypes 34 failed: %v", err)
	}
	if err := global.DB.Exec(`INSERT INTO invTypes (typeID, groupID, typeName, marketGroupID, published) VALUES (?, ?, ?, ?, ?)`, 35, 2, "Pyerite", 3, 1).Error; err != nil {
		t.Fatalf("insert invTypes 35 failed: %v", err)
	}

	// 父资产 999：带自定义资产名，验证 asset_name 优先级
	if err := global.DB.Create(&model.EveCharacterAsset{
		ItemID:       999,
		CharacterID:  1001,
		LocationID:   60003760,
		LocationType: "station",
		TypeID:       34,
		Quantity:     1,
		AssetName:    "My Container",
	}).Error; err != nil {
		t.Fatalf("insert parent asset 999 failed: %v", err)
	}
	// 父资产 888：无 asset_name，验证 type_name 回退
	if err := global.DB.Create(&model.EveCharacterAsset{
		ItemID:       888,
		CharacterID:  1001,
		LocationID:   60003760,
		LocationType: "station",
		TypeID:       35,
		Quantity:     1,
	}).Error; err != nil {
		t.Fatalf("insert parent asset 888 failed: %v", err)
	}

	result := svc.resolveItemLocationNamesLocal([]int64{1001}, []int64{999, 888, 777}, "en")

	if got := result[999]; got != "My Container" {
		t.Fatalf("expected asset_name priority for 999, got %q", got)
	}
	if got := result[888]; got != "Pyerite" {
		t.Fatalf("expected type_name fallback for 888, got %q", got)
	}
	if got := result[777]; got != "Item-777" {
		t.Fatalf("expected fallback for missing item 777, got %q", got)
	}
}

package repository

import (
	"amiya-eden/global"
	"amiya-eden/internal/model"
	"fmt"
)

var TC_ID = map[string]int{
	"type":          8,
	"group":         7,
	"category":      6,
	"description":   33,
	"tech":          34,
	"market_group":  36,
	"solar_system":  40,
	"constellation": 41,
	"region":        42,
}

// SdeRepository SDE 数据访问层
type SdeRepository struct{}

func NewSdeRepository() *SdeRepository { return &SdeRepository{} }

// ---- SDE 版本管理 ----

// GetLatestVersion 获取最新已导入的 SDE 版本
func (r *SdeRepository) GetLatestVersion() (*model.SdeVersion, error) {
	var v model.SdeVersion
	err := global.DB.Order("id DESC").First(&v).Error
	return &v, err
}

// CreateVersion 记录新版本
func (r *SdeRepository) CreateVersion(v *model.SdeVersion) error {
	return global.DB.Create(v).Error
}

// VersionExists 检查某个版本是否已存在
func (r *SdeRepository) VersionExists(version string) (bool, error) {
	var count int64
	err := global.DB.Model(&model.SdeVersion{}).Where("version = ?", version).Count(&count).Error
	return count > 0, err
}

// ---- trnTranslations 翻译查询 ----

// GetTranslation 精确查询翻译
func (r *SdeRepository) GetTranslation(tcID int, keyID int, languageID string) (*model.TrnTranslation, error) {
	var t model.TrnTranslation
	err := global.DB.Where(`"tcID" = ? AND "keyID" = ? AND "languageID" = ?`, tcID, keyID, languageID).First(&t).Error
	return &t, err
}

// ---- invTypes 查询 ----
type TypeInfo struct {
	TypeID          int    `json:"type_id"`
	TypeName        string `json:"type_name"`
	GroupID         int    `json:"group_id"`
	GroupName       string `json:"group_name"`
	MarketGroupID   int    `json:"market_group_id"`
	MarketGroupName string `json:"market_group_name"`
	CategoryID      int    `json:"category_id"`
	CategoryName    string `json:"category_name"`
}

// GetNames 批量查询 id -> name 映射，ids 为 tcID key -> id列表
func (r *SdeRepository) GetNames(ids map[string][]int, languageID string) (map[int]string, error) {
	result := make(map[int]string)

	type row struct {
		KeyID int    `gorm:"column:keyID"`
		Text  string `gorm:"column:text"`
	}

	for key, keyIDs := range ids {
		if len(keyIDs) == 0 {
			continue
		}
		tcID, ok := TC_ID[key]
		if !ok {
			continue // 忽略不存在的 key
		}
		var rows []row
		if err := global.DB.Table(`"trnTranslations"`).
			Select(`"keyID", text`).
			Where(`"tcID" = ? AND "keyID" IN ? AND "languageID" = ?`, tcID, keyIDs, languageID).
			Scan(&rows).Error; err != nil {
			return nil, err
		}
		for _, r := range rows {
			result[r.KeyID] = r.Text
		}
	}

	return result, nil
}

// GetTypes 查询物品(组)信息
func (r *SdeRepository) GetTypes(typeIDs []int, published *bool, languageID string) ([]TypeInfo, error) {
	result, err := r.getTypesWithLayout(typeIDs, published, languageID, true)
	if err == nil {
		return result, nil
	}

	// Some SDE imports end up with unquoted lowercase PostgreSQL identifiers.
	fallbackResult, fallbackErr := r.getTypesWithLayout(typeIDs, published, languageID, false)
	if fallbackErr == nil {
		return fallbackResult, nil
	}

	return nil, fmt.Errorf("%w; fallback query failed: %v", err, fallbackErr)
}

func (r *SdeRepository) getTypesWithLayout(typeIDs []int, published *bool, languageID string, camelCase bool) ([]TypeInfo, error) {
	var result []TypeInfo

	tableInvTypes := `"invTypes" t`
	joinInvGroups := `"invGroups" g`
	joinInvCategories := `"invCategories" c`
	joinInvMarketGroups := `"invMarketGroups" mg`
	joinTranslationsType := `"trnTranslations" t_name`
	joinTranslationsGroup := `"trnTranslations" g_name`
	joinTranslationsCategory := `"trnTranslations" c_name`
	joinTranslationsMarket := `"trnTranslations" mg_name`

	typeIDCol := `t."typeID"`
	groupIDCol := `t."groupID"`
	marketGroupIDCol := `t."marketGroupID"`
	publishedCol := `t.published`
	groupGroupIDCol := `g."groupID"`
	groupCategoryIDCol := `g."categoryID"`
	categoryIDCol := `c."categoryID"`
	marketGroupJoinIDCol := `mg."marketGroupID"`
	typeNameBaseCol := `t."typeName"`
	groupNameBaseCol := `g."groupName"`
	categoryNameBaseCol := `c."categoryName"`
	marketGroupNameBaseCol := `mg."marketGroupName"`
	trTcIDCol := `"tcID"`
	trKeyIDCol := `"keyID"`
	trLanguageIDCol := `"languageID"`

	if !camelCase {
		tableInvTypes = `invtypes t`
		joinInvGroups = `invgroups g`
		joinInvCategories = `invcategories c`
		joinInvMarketGroups = `invmarketgroups mg`
		joinTranslationsType = `trntranslations t_name`
		joinTranslationsGroup = `trntranslations g_name`
		joinTranslationsCategory = `trntranslations c_name`
		joinTranslationsMarket = `trntranslations mg_name`

		typeIDCol = `t.typeid`
		groupIDCol = `t.groupid`
		marketGroupIDCol = `t.marketgroupid`
		publishedCol = `t.published`
		groupGroupIDCol = `g.groupid`
		groupCategoryIDCol = `g.categoryid`
		categoryIDCol = `c.categoryid`
		marketGroupJoinIDCol = `mg.marketgroupid`
		typeNameBaseCol = `t.typename`
		groupNameBaseCol = `g.groupname`
		categoryNameBaseCol = `c.categoryname`
		marketGroupNameBaseCol = `mg.marketgroupname`
		trTcIDCol = `tcid`
		trKeyIDCol = `keyid`
		trLanguageIDCol = `languageid`
	}

	query := global.DB.Table(tableInvTypes).
		Select(fmt.Sprintf(`
            %s AS type_id,
            COALESCE(NULLIF(t_name.text, ''), %s) AS type_name,
            %s AS group_id,
            COALESCE(NULLIF(g_name.text, ''), %s) AS group_name,
            %s AS market_group_id,
            COALESCE(NULLIF(mg_name.text, ''), %s) AS market_group_name,
            %s AS category_id,
            COALESCE(NULLIF(c_name.text, ''), %s) AS category_name
        `, typeIDCol, typeNameBaseCol, groupGroupIDCol, groupNameBaseCol, marketGroupIDCol, marketGroupNameBaseCol, categoryIDCol, categoryNameBaseCol)).
		// invTypes -> invGroups
		Joins(fmt.Sprintf(`LEFT JOIN %s ON %s = %s`, joinInvGroups, groupGroupIDCol, groupIDCol)).
		// invGroups -> invCategories
		Joins(fmt.Sprintf(`LEFT JOIN %s ON %s = %s`, joinInvCategories, categoryIDCol, groupCategoryIDCol)).
		// invGroups -> invMarketGroups
		Joins(fmt.Sprintf(`LEFT JOIN %s ON %s = %s`, joinInvMarketGroups, marketGroupJoinIDCol, marketGroupIDCol)).
		// 物品名翻译
		Joins(fmt.Sprintf(`LEFT JOIN %s ON t_name.%s = ? AND t_name.%s = %s AND t_name.%s = ?`, joinTranslationsType, trTcIDCol, trKeyIDCol, typeIDCol, trLanguageIDCol),
			TC_ID["type"], languageID).
		// 组名翻译
		Joins(fmt.Sprintf(`LEFT JOIN %s ON g_name.%s = ? AND g_name.%s = %s AND g_name.%s = ?`, joinTranslationsGroup, trTcIDCol, trKeyIDCol, groupGroupIDCol, trLanguageIDCol),
			TC_ID["group"], languageID).
		// 分类名翻译
		Joins(fmt.Sprintf(`LEFT JOIN %s ON c_name.%s = ? AND c_name.%s = %s AND c_name.%s = ?`, joinTranslationsCategory, trTcIDCol, trKeyIDCol, categoryIDCol, trLanguageIDCol),
			TC_ID["category"], languageID).
		// 市场组名翻译
		Joins(fmt.Sprintf(`LEFT JOIN %s ON mg_name.%s = ? AND mg_name.%s = %s AND mg_name.%s = ?`, joinTranslationsMarket, trTcIDCol, trKeyIDCol, marketGroupJoinIDCol, trLanguageIDCol),
			TC_ID["market_group"], languageID)

	if len(typeIDs) > 0 {
		query = query.Where(fmt.Sprintf(`%s IN ?`, typeIDCol), typeIDs)
	}
	if published != nil && *published {
		query = query.Where(fmt.Sprintf(`%s = ?`, publishedCol), 1)
	}

	if err := query.Scan(&result).Error; err != nil {
		return nil, err
	}
	return result, nil
}

// GetTypesByCategoryID 获取指定 categoryID 下所有已发布的物品（含翻译）
// 主要用于技能模块获取 categoryID=16 的全量技能列表
func (r *SdeRepository) GetTypesByCategoryID(categoryID int, languageID string) ([]TypeInfo, error) {
	result, err := r.getTypesByCategoryIDWithLayout(categoryID, languageID, true)
	if err == nil {
		return result, nil
	}

	// Some SDE imports end up with unquoted lowercase PostgreSQL identifiers.
	fallbackResult, fallbackErr := r.getTypesByCategoryIDWithLayout(categoryID, languageID, false)
	if fallbackErr == nil {
		return fallbackResult, nil
	}

	return nil, fmt.Errorf("%w; fallback query failed: %v", err, fallbackErr)
}

func (r *SdeRepository) getTypesByCategoryIDWithLayout(categoryID int, languageID string, camelCase bool) ([]TypeInfo, error) {
	var result []TypeInfo

	tableInvTypes := `"invTypes" t`
	joinInvGroups := `"invGroups" g`
	joinInvCategories := `"invCategories" c`
	joinInvMarketGroups := `"invMarketGroups" mg`
	joinTranslationsType := `"trnTranslations" t_name`
	joinTranslationsGroup := `"trnTranslations" g_name`
	joinTranslationsCategory := `"trnTranslations" c_name`
	joinTranslationsMarket := `"trnTranslations" mg_name`

	typeIDCol := `t."typeID"`
	groupIDCol := `t."groupID"`
	marketGroupIDCol := `t."marketGroupID"`
	publishedCol := `t.published`
	groupGroupIDCol := `g."groupID"`
	groupCategoryIDCol := `g."categoryID"`
	categoryIDCol := `c."categoryID"`
	marketGroupJoinIDCol := `mg."marketGroupID"`
	typeNameBaseCol := `t."typeName"`
	groupNameBaseCol := `g."groupName"`
	categoryNameBaseCol := `c."categoryName"`
	marketGroupNameBaseCol := `mg."marketGroupName"`
	trTcIDCol := `"tcID"`
	trKeyIDCol := `"keyID"`
	trLanguageIDCol := `"languageID"`

	if !camelCase {
		tableInvTypes = `invtypes t`
		joinInvGroups = `invgroups g`
		joinInvCategories = `invcategories c`
		joinInvMarketGroups = `invmarketgroups mg`
		joinTranslationsType = `trntranslations t_name`
		joinTranslationsGroup = `trntranslations g_name`
		joinTranslationsCategory = `trntranslations c_name`
		joinTranslationsMarket = `trntranslations mg_name`

		typeIDCol = `t.typeid`
		groupIDCol = `t.groupid`
		marketGroupIDCol = `t.marketgroupid`
		publishedCol = `t.published`
		groupGroupIDCol = `g.groupid`
		groupCategoryIDCol = `g.categoryid`
		categoryIDCol = `c.categoryid`
		marketGroupJoinIDCol = `mg.marketgroupid`
		typeNameBaseCol = `t.typename`
		groupNameBaseCol = `g.groupname`
		categoryNameBaseCol = `c.categoryname`
		marketGroupNameBaseCol = `mg.marketgroupname`
		trTcIDCol = `tcid`
		trKeyIDCol = `keyid`
		trLanguageIDCol = `languageid`
	}

	query := global.DB.Table(tableInvTypes).
		Select(fmt.Sprintf(`
            %s AS type_id,
            COALESCE(NULLIF(t_name.text, ''), %s) AS type_name,
            %s AS group_id,
            COALESCE(NULLIF(g_name.text, ''), %s) AS group_name,
            %s AS market_group_id,
            COALESCE(NULLIF(mg_name.text, ''), %s) AS market_group_name,
            %s AS category_id,
            COALESCE(NULLIF(c_name.text, ''), %s) AS category_name
        `, typeIDCol, typeNameBaseCol, groupGroupIDCol, groupNameBaseCol, marketGroupIDCol, marketGroupNameBaseCol, categoryIDCol, categoryNameBaseCol)).
		Joins(fmt.Sprintf(`LEFT JOIN %s ON %s = %s`, joinInvGroups, groupGroupIDCol, groupIDCol)).
		Joins(fmt.Sprintf(`LEFT JOIN %s ON %s = %s`, joinInvCategories, categoryIDCol, groupCategoryIDCol)).
		Joins(fmt.Sprintf(`LEFT JOIN %s ON %s = %s`, joinInvMarketGroups, marketGroupJoinIDCol, marketGroupIDCol)).
		Joins(fmt.Sprintf(`LEFT JOIN %s ON t_name.%s = ? AND t_name.%s = %s AND t_name.%s = ?`, joinTranslationsType, trTcIDCol, trKeyIDCol, typeIDCol, trLanguageIDCol),
			TC_ID["type"], languageID).
		Joins(fmt.Sprintf(`LEFT JOIN %s ON g_name.%s = ? AND g_name.%s = %s AND g_name.%s = ?`, joinTranslationsGroup, trTcIDCol, trKeyIDCol, groupGroupIDCol, trLanguageIDCol),
			TC_ID["group"], languageID).
		Joins(fmt.Sprintf(`LEFT JOIN %s ON c_name.%s = ? AND c_name.%s = %s AND c_name.%s = ?`, joinTranslationsCategory, trTcIDCol, trKeyIDCol, categoryIDCol, trLanguageIDCol),
			TC_ID["category"], languageID).
		Joins(fmt.Sprintf(`LEFT JOIN %s ON mg_name.%s = ? AND mg_name.%s = %s AND mg_name.%s = ?`, joinTranslationsMarket, trTcIDCol, trKeyIDCol, marketGroupJoinIDCol, trLanguageIDCol),
			TC_ID["market_group"], languageID).
		Where(fmt.Sprintf(`%s = ? AND %s = 1`, categoryIDCol, publishedCol), categoryID)

	if err := query.Scan(&result).Error; err != nil {
		return nil, err
	}
	return result, nil
}

// FlagInfo invFlags 表行
type FlagInfo struct {
	FlagID   int    `json:"flag_id"   gorm:"column:flagID"`
	FlagName string `json:"flag_name" gorm:"column:flagName"`
	FlagText string `json:"flag_text" gorm:"column:flagText"`
	OrderID  int    `json:"order_id"  gorm:"column:orderID"`
}

// FuzzySearchItem 模糊搜索结果条目
type FuzzySearchItem struct {
	ID        int    `json:"id"         gorm:"column:id"`
	Name      string `json:"name"       gorm:"column:name"`
	GroupID   int    `json:"group_id"   gorm:"column:group_id"`
	GroupName string `json:"group_name" gorm:"column:group_name"`
	Category  string `json:"category"   gorm:"column:category"` // "type" | "character"
}

// GetTypeIDsByNames 通过英文名称批量反查 typeID（用于 EFT 解析）
func (r *SdeRepository) GetTypeIDsByNames(names []string) (map[string]int64, error) {
	if len(names) == 0 {
		return map[string]int64{}, nil
	}
	result, err := r.getTypeIDsByNamesWithLayout(names, true)
	if err == nil {
		return result, nil
	}

	// Fallback for lowercase PostgreSQL SDE imports.
	fallbackResult, fallbackErr := r.getTypeIDsByNamesWithLayout(names, false)
	if fallbackErr == nil {
		return fallbackResult, nil
	}

	return nil, fmt.Errorf("%w; fallback query failed: %v", err, fallbackErr)
}

func (r *SdeRepository) getTypeIDsByNamesWithLayout(names []string, camelCase bool) (map[string]int64, error) {
	type row struct {
		TypeID   int64  `gorm:"column:type_id"`
		TypeName string `gorm:"column:type_name"`
	}
	var rows []row

	tableName := `"invTypes"`
	typeIDCol := `"typeID"`
	typeNameCol := `"typeName"`
	if !camelCase {
		tableName = `invtypes`
		typeIDCol = `typeid`
		typeNameCol = `typename`
	}

	err := global.DB.Table(tableName).
		Select(fmt.Sprintf(`%s AS type_id, %s AS type_name`, typeIDCol, typeNameCol)).
		Where(fmt.Sprintf(`%s IN ?`, typeNameCol), names).
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	result := make(map[string]int64, len(rows))
	for _, r := range rows {
		result[r.TypeName] = r.TypeID
	}
	return result, nil
}

// FuzzySearch 模糊搜索 trnTranslations（物品名称）及成员名称
func (r *SdeRepository) FuzzySearch(keyword string, languageID string, categoryIDs []int, excludeCategoryIDs []int, limit int, searchMember bool) ([]FuzzySearchItem, error) {
	result, err := r.fuzzySearchWithLayout(keyword, languageID, categoryIDs, excludeCategoryIDs, limit, searchMember, true)
	if err == nil {
		return result, nil
	}

	// Some SDE imports end up with unquoted lowercase PostgreSQL identifiers.
	fallbackResult, fallbackErr := r.fuzzySearchWithLayout(keyword, languageID, categoryIDs, excludeCategoryIDs, limit, searchMember, false)
	if fallbackErr == nil {
		return fallbackResult, nil
	}

	return nil, fmt.Errorf("%w; fallback query failed: %v", err, fallbackErr)
}

func (r *SdeRepository) fuzzySearchWithLayout(keyword string, languageID string, categoryIDs []int, excludeCategoryIDs []int, limit int, searchMember bool, camelCase bool) ([]FuzzySearchItem, error) {
	if keyword == "" {
		return nil, nil
	}
	if limit <= 0 {
		limit = 20
	}
	if languageID == "" {
		languageID = "en"
	}

	var result []FuzzySearchItem
	pattern := "%" + keyword + "%"

	tableInvTypes := `"invTypes" t`
	joinInvGroups := `"invGroups" g`
	joinInvCategories := `"invCategories" c`
	joinTypeTranslations := `"trnTranslations" tr`
	joinGroupTranslations := `"trnTranslations" g_name`

	typeIDCol := `t."typeID"`
	groupIDCol := `t."groupID"`
	publishedCol := `t.published`
	groupGroupIDCol := `g."groupID"`
	groupCategoryIDCol := `g."categoryID"`
	categoryIDCol := `c."categoryID"`
	typeNameBaseCol := `t."typeName"`
	groupNameBaseCol := `g."groupName"`
	trTcIDCol := `"tcID"`
	trKeyIDCol := `"keyID"`
	trLanguageIDCol := `"languageID"`

	if !camelCase {
		tableInvTypes = `invtypes t`
		joinInvGroups = `invgroups g`
		joinInvCategories = `invcategories c`
		joinTypeTranslations = `trntranslations tr`
		joinGroupTranslations = `trntranslations g_name`

		typeIDCol = `t.typeid`
		groupIDCol = `t.groupid`
		publishedCol = `t.published`
		groupGroupIDCol = `g.groupid`
		groupCategoryIDCol = `g.categoryid`
		categoryIDCol = `c.categoryid`
		typeNameBaseCol = `t.typename`
		groupNameBaseCol = `g.groupname`
		trTcIDCol = `tcid`
		trKeyIDCol = `keyid`
		trLanguageIDCol = `languageid`
	}

	// 1. 搜索 SDE 物品名称。显示名称优先当前语言翻译，搜索关键字同时兼容当前语言翻译和英文基础名。
	query := global.DB.Table(tableInvTypes).
		Select(fmt.Sprintf(`
			%s AS id,
			COALESCE(NULLIF(tr.text, ''), %s) AS name,
			%s AS group_id,
			COALESCE(NULLIF(g_name.text, ''), %s) AS group_name,
			'type' AS category
		`, typeIDCol, typeNameBaseCol, groupGroupIDCol, groupNameBaseCol)).
		Joins(fmt.Sprintf(`JOIN %s ON %s = %s`, joinInvGroups, groupGroupIDCol, groupIDCol)).
		Joins(fmt.Sprintf(`JOIN %s ON %s = %s`, joinInvCategories, categoryIDCol, groupCategoryIDCol)).
		Joins(fmt.Sprintf(`LEFT JOIN %s ON tr.%s = ? AND tr.%s = %s AND tr.%s = ?`, joinTypeTranslations, trTcIDCol, trKeyIDCol, typeIDCol, trLanguageIDCol),
			TC_ID["type"], languageID).
		Joins(fmt.Sprintf(`LEFT JOIN %s ON g_name.%s = ? AND g_name.%s = %s AND g_name.%s = ?`, joinGroupTranslations, trTcIDCol, trKeyIDCol, groupGroupIDCol, trLanguageIDCol),
			TC_ID["group"], languageID).
		Where(fmt.Sprintf(`%s = 1`, publishedCol)).
		Where(fmt.Sprintf(`(tr.text ILIKE ? OR %s ILIKE ?)`, typeNameBaseCol), pattern, pattern)

	if len(categoryIDs) > 0 {
		query = query.Where(fmt.Sprintf(`%s IN ?`, categoryIDCol), categoryIDs)
	}
	if len(excludeCategoryIDs) > 0 {
		query = query.Where(fmt.Sprintf(`%s NOT IN ?`, categoryIDCol), excludeCategoryIDs)
	}

	query = query.Limit(limit)
	if err := query.Scan(&result).Error; err != nil {
		return nil, err
	}

	// 2. 搜索成员名称（eve_character 表）
	if searchMember && limit > len(result) {
		remaining := limit - len(result)
		var members []FuzzySearchItem
		if err := global.DB.Table("eve_character").
			Select(`character_id AS id, character_name AS name, 0 AS group_id, '' AS group_name, 'character' AS category`).
			Where("character_name ILIKE ?", pattern).
			Limit(remaining).
			Scan(&members).Error; err != nil {
			return nil, err
		}
		result = append(result, members...)
	}

	return result, nil
}

// GetFlags 批量查询 invFlags
func (r *SdeRepository) GetFlags(flagIDs []int) ([]FlagInfo, error) {
	var result []FlagInfo
	if len(flagIDs) == 0 {
		return result, nil
	}
	err := global.DB.Table(`"invFlags"`).
		Where(`"flagID" IN ?`, flagIDs).
		Order(`"orderID" ASC`).
		Scan(&result).Error
	return result, err
}

// ---- 舰船 & 技能需求查询 ----

// ShipInfo 舰船基础信息（含翻译 + raceID + marketGroupID）
type ShipInfo struct {
	TypeID          int    `json:"type_id"           gorm:"column:type_id"`
	TypeName        string `json:"type_name"         gorm:"column:type_name"`
	GroupID         int    `json:"group_id"           gorm:"column:group_id"`
	GroupName       string `json:"group_name"         gorm:"column:group_name"`
	MarketGroupID   int    `json:"market_group_id"    gorm:"column:market_group_id"`
	MarketGroupName string `json:"market_group_name"  gorm:"column:market_group_name"`
	RaceID          int    `json:"race_id"            gorm:"column:race_id"`
}

// GetShipsByCategoryID 获取 categoryID=6 的所有已发布舰船（含翻译、raceID、marketGroupID）
func (r *SdeRepository) GetShipsByCategoryID(languageID string) ([]ShipInfo, error) {
	const shipCategoryID = 6
	var result []ShipInfo

	err := global.DB.Table(`"invTypes" t`).
		Select(`
			t."typeID"        AS type_id,
			t_name.text       AS type_name,
			g."groupID"       AS group_id,
			g_name.text       AS group_name,
			t."marketGroupID" AS market_group_id,
			mg_name.text      AS market_group_name,
			t."raceID"        AS race_id
		`).
		Joins(`JOIN "invGroups" g ON g."groupID" = t."groupID"`).
		Joins(`JOIN "invCategories" c ON c."categoryID" = g."categoryID"`).
		Joins(`LEFT JOIN "invMarketGroups" mg ON mg."marketGroupID" = t."marketGroupID"`).
		Joins(`LEFT JOIN "trnTranslations" t_name ON t_name."tcID" = ? AND t_name."keyID" = t."typeID" AND t_name."languageID" = ?`,
			TC_ID["type"], languageID).
		Joins(`LEFT JOIN "trnTranslations" g_name ON g_name."tcID" = ? AND g_name."keyID" = g."groupID" AND g_name."languageID" = ?`,
			TC_ID["group"], languageID).
		Joins(`LEFT JOIN "trnTranslations" mg_name ON mg_name."tcID" = ? AND mg_name."keyID" = mg."marketGroupID" AND mg_name."languageID" = ?`,
			TC_ID["market_group"], languageID).
		Where(`c."categoryID" = ? AND t.published = 1`, shipCategoryID).
		Scan(&result).Error
	return result, err
}

// ShipSkillReq 舰船技能需求行（含递归层深）
type ShipSkillReq struct {
	ShipTypeID    int `json:"ship_type_id"    gorm:"column:ship_type_id"`
	SkillTypeID   int `json:"skill_type_id"   gorm:"column:skill_type_id"`
	RequiredLevel int `json:"required_level"  gorm:"column:required_level"`
	Depth         int `json:"depth"           gorm:"column:depth"`
}

// GetShipSkillRequirements 批量获取舰船技能需求（含前置技能递归）
func (r *SdeRepository) GetShipSkillRequirements(shipTypeIDs []int) ([]ShipSkillReq, error) {
	if len(shipTypeIDs) == 0 {
		return nil, nil
	}
	var result []ShipSkillReq

	sql := `
WITH RECURSIVE skill_tree AS (
  SELECT
    sk."typeID"   AS ship_type_id,
    sk."valueInt" AS skill_type_id,
    lv."valueInt" AS required_level,
    1             AS depth
  FROM "dgmTypeAttributes" sk
  JOIN "dgmTypeAttributes" lv
    ON sk."typeID" = lv."typeID"
    AND lv."attributeID" = CASE sk."attributeID"
      WHEN 182  THEN 277  WHEN 183 THEN 278  WHEN 184  THEN 279
      WHEN 1285 THEN 1286 WHEN 1289 THEN 1287 WHEN 1290 THEN 1288
    END
  WHERE sk."typeID" IN ?
    AND sk."attributeID" IN (182, 183, 184, 1285, 1289, 1290)
    AND sk."valueInt" IS NOT NULL

  UNION

  SELECT
    st.ship_type_id,
    sk."valueInt" AS skill_type_id,
    lv."valueInt" AS required_level,
    st.depth + 1
  FROM skill_tree st
  JOIN "dgmTypeAttributes" sk
    ON sk."typeID" = st.skill_type_id
    AND sk."attributeID" IN (182, 183, 184, 1285, 1289, 1290)
  JOIN "dgmTypeAttributes" lv
    ON sk."typeID" = lv."typeID"
    AND lv."attributeID" = CASE sk."attributeID"
      WHEN 182  THEN 277  WHEN 183 THEN 278  WHEN 184  THEN 279
      WHEN 1285 THEN 1286 WHEN 1289 THEN 1287 WHEN 1290 THEN 1288
    END
  WHERE sk."valueInt" IS NOT NULL
    AND st.depth < 10
)
SELECT
  ship_type_id,
  skill_type_id,
  MAX(required_level) AS required_level,
  MIN(depth)          AS depth
FROM skill_tree
GROUP BY ship_type_id, skill_type_id`

	err := global.DB.Raw(sql, shipTypeIDs).Scan(&result).Error
	return result, err
}

// RaceInfo chrRaces 表行
type RaceInfo struct {
	RaceID   int    `json:"race_id"   gorm:"column:raceID"`
	RaceName string `json:"race_name" gorm:"column:raceName"`
}

// GetAllRaces 获取所有种族
func (r *SdeRepository) GetAllRaces() ([]RaceInfo, error) {
	var result []RaceInfo
	err := global.DB.Table(`"chrRaces"`).
		Select(`"raceID", "raceName"`).
		Scan(&result).Error
	return result, err
}

// MarketGroupParent 返回 marketGroupID -> parentGroupID 映射
type MarketGroupNode struct {
	MarketGroupID int `gorm:"column:marketGroupID"`
	ParentGroupID int `gorm:"column:parentGroupID"`
}

// GetMarketGroupTree 获取所有市场分组的父子关系
func (r *SdeRepository) GetMarketGroupTree() ([]MarketGroupNode, error) {
	var result []MarketGroupNode
	err := global.DB.Table(`"invMarketGroups"`).
		Select(`"marketGroupID", COALESCE("parentGroupID", 0) AS "parentGroupID"`).
		Scan(&result).Error
	return result, err
}

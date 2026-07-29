package service

import (
	"amiya-eden/global"
	"amiya-eden/internal/model"
	"amiya-eden/internal/repository"
	"amiya-eden/pkg/eve/esi"
	"context"
	"fmt"
	"math"
	"strings"
	"time"

	"go.uber.org/zap"
)

// ─────────────────────────────────────────────
//  建筑服务模块燃料消耗
//
//  Upwell 建筑本身没有基础燃料消耗，燃料仅由「在线的服务模块」消耗。
//  每小时消耗 = Σ(在线服务的有效率)，有效率 = 服务率 × 结构类别系数。
//  参见 docs/reference 与 EVE University Wiki: Upwell structure。
// ─────────────────────────────────────────────

const (
	// 结构类别 group_id（来自 ESI /universe/types/ 与 SDE invTypes.groupID）
	structureGroupCitadel            = 1657 // Astrahus / Fortizar / Keepstar（堡垒）
	structureGroupEngineeringComplex = 1404 // Raitaru / Azbel / Sotiyo（工业综合体）
	structureGroupRefinery           = 1406 // Athanor / Tatara（精炼厂）

	// 已知的 Upwell 建筑 typeID（精炼厂系数需区分 Athanor / Tatara）
	structureTypeIDAthanor = 35835

	// dogma 属性
	dogmaAttrServiceModuleFuelAmount = 2109 // serviceModuleFuelAmount：每小时燃料块
)

// 结构类别对服务的燃料折扣系数（值自 2018 Upwell 改版后稳定）。
// 不匹配类别的服务系数为 1.0（无折扣）。
const (
	fuelFactorCitadel            = 0.75 // 堡垒：市场/克隆
	fuelFactorEngineeringComplex = 0.75 // 工业综合体：制造/研究/发明
	fuelFactorRefineryAthanor    = 0.80 // 精炼厂 Athanor：再处理/反应
	fuelFactorRefineryOther      = 0.75 // 精炼厂（Tatara 等）：再处理/反应
)

// normalizeServiceName 把 ESI 建筑快照 services[].name 的展示字符串
// 归一化为燃料率映射表使用的 snake_case 键：去首尾空白、转小写、
// 把连续空格 / 连字符 / 下划线折叠为单个下划线。
//
// 这样 ESI 返回的 "Market" / "Clone Bay" / "Moon Drilling" 会稳定归一为
// 默认表与 DB 中已有的 market / clone_bay / moon_drilling 键。
// 既有 snake_case（如 "clone_bay"）与连字符（如 "moon-drilling"）同样兼容。
func normalizeServiceName(name string) string {
	s := strings.TrimSpace(name)
	if s == "" {
		return ""
	}
	s = strings.ToLower(s)
	var b strings.Builder
	b.Grow(len(s))
	inSep := false
	for _, r := range s {
		switch r {
		case ' ', '\t', '-', '_':
			inSep = true
		default:
			if inSep {
				b.WriteByte('_')
				inSep = false
			}
			b.WriteRune(r)
		}
	}
	return b.String()
}

// 服务类别：用于判断该服务是否可享受某类建筑的折扣。
type structureServiceCategory int

const (
	serviceCategoryOther              structureServiceCategory = iota
	serviceCategoryCitadelOnly                                 // 市场克隆（堡垒折扣）
	serviceCategoryEngineeringComplex                          // 制造研究发明（工业综合体折扣）
	serviceCategoryRefinery                                    // 再处理反应（精炼厂折扣）
)

// 默认服务模块燃料率是系统目录的兼容视图。运行时估算按 type_id
// 读取 builtinStructureServiceModules；这里仅保留给既有燃料率同步与测试。
type defaultFuelRate struct {
	TypeID      int
	TypeName    string
	FuelPerHour float64
	Category    structureServiceCategory
}

// defaultServiceFuelRates returns one stable internal key per physical module.
func defaultServiceFuelRates() map[string]defaultFuelRate {
	result := make(map[string]defaultFuelRate)
	for _, module := range builtinStructureServiceModules() {
		result[module.ServiceName] = defaultFuelRate{
			TypeID:      module.TypeID,
			TypeName:    module.TypeName,
			FuelPerHour: module.FuelPerHour,
			Category:    parseCategory(module.FuelCategory),
		}
	}
	// Legacy EstimateFuelPerHour callers still accept this old generic label.
	// The module-aware estimator does not use it.
	result["industry"] = result["manufacturing"]
	return result
}

// serviceFuelFactor 判断服务在指定建筑分组下享受的燃料系数。
// 系数按「建筑分组 × 服务类别」匹配；不匹配返回 1.0。
//
// 规则（值自 2018 Upwell 改版后稳定）：
//   - 工业综合体服务（制造/研究/发明）：0.75
//   - 堡垒服务（市场/克隆）：0.75
//   - 精炼厂服务（再处理/反应）：Athanor 0.80，其余（Tatara 等）0.75
//   - 其他服务（诱导/跳桥等）：1.0（无折扣）
//
// structureGroupID 来自 SDE invTypes.groupID；缺失（0）或未知分组按无折扣计算。
// refineryIsAthanor 用于在精炼厂分组内区分 Athanor（typeID 35835）。
func serviceFuelFactor(structureGroupID int, structureTypeID int64, cat structureServiceCategory) float64 {
	switch cat {
	case serviceCategoryOther:
		return 1.0
	case serviceCategoryEngineeringComplex:
		if structureGroupID == structureGroupEngineeringComplex {
			return fuelFactorEngineeringComplex
		}
		return 1.0
	case serviceCategoryCitadelOnly:
		if structureGroupID == structureGroupCitadel {
			return fuelFactorCitadel
		}
		return 1.0
	case serviceCategoryRefinery:
		if structureGroupID != structureGroupRefinery {
			return 1.0
		}
		// Athanor（typeID 35835）是唯一使用 0.80 的建筑，其余精炼厂 0.75
		if structureTypeID == structureTypeIDAthanor {
			return fuelFactorRefineryAthanor
		}
		return fuelFactorRefineryOther
	}
	return 1.0
}

// ─────────────────────────────────────────────
//  SDE 分组查询接口（便于测试注入）
// ─────────────────────────────────────────────

// StructureTypeGroupResolver 批量解析建筑 typeID → groupID。
type StructureTypeGroupResolver interface {
	GetTypes(typeIDs []int, published *bool, languageID string) ([]repository.TypeInfo, error)
}

// resolveGroupIDMap 批量查询 typeID → groupID；缺失的 typeID 不在结果中（按无折扣处理）。
func resolveGroupIDMap(resolver StructureTypeGroupResolver, typeIDs []int) (map[int64]int, error) {
	result := make(map[int64]int, len(typeIDs))
	if len(typeIDs) == 0 || resolver == nil {
		return result, nil
	}
	infos, err := resolver.GetTypes(typeIDs, nil, "en")
	if err != nil {
		return nil, err
	}
	for _, info := range infos {
		result[int64(info.TypeID)] = info.GroupID
	}
	return result, nil
}

// ─────────────────────────────────────────────
//  ESI 同步
// ─────────────────────────────────────────────

// esiTypeResponse 是 ESI /universe/types/{id}/ 响应的精简结构。
type esiTypeResponse struct {
	TypeID          int                 `json:"type_id"`
	Name            string              `json:"name"`
	GroupID         int                 `json:"group_id"`
	DogmaAttributes []esiDogmaAttribute `json:"dogma_attributes,omitempty"`
}

type esiDogmaAttribute struct {
	AttributeID int     `json:"attribute_id"`
	Value       float64 `json:"value"`
}

// ESITypeFetcher 仅暴露拉取 type 详情的能力，便于测试注入假 ESI。
type ESITypeFetcher interface {
	Get(ctx context.Context, path string, accessToken string, dest interface{}) error
}

// StructureFuelRateService 维护服务模块燃料率映射表
type StructureFuelRateService struct {
	repo      *repository.StructureServiceFuelRateRepository
	esiClient ESITypeFetcher
}

// NewStructureFuelRateService 构造服务。
func NewStructureFuelRateService() *StructureFuelRateService {
	cfg := global.Config.EveSSO
	return &StructureFuelRateService{
		repo:      repository.NewStructureServiceFuelRateRepository(),
		esiClient: esi.NewClientWithConfig(cfg.ESIBaseURL, cfg.ESIAPIPrefix),
	}
}

// LoadRateMap 加载全量映射，返回 service name(归一化) → 每小时块数。
// 数据库中的记录会「覆盖」默认回退表：DB 有则以 DB 为准，DB 缺失的服务沿用默认值。
// 服务名会经 normalizeServiceName 归一化为 snake_case 键，兼容 ESI 展示名（如 "Clone Bay"）。
// 注意：返回的是「原始」服务率（未乘结构系数），系数由调用方按建筑分组计算。
func (s *StructureFuelRateService) LoadRateMap() map[string]float64 {
	rateMap := defaultRateMap()
	rows, err := s.repo.ListAll()
	if err != nil {
		logStructureFuelRateWarn("加载服务燃料率映射失败，使用硬编码回退", err)
		return rateMap
	}
	for _, row := range rows {
		name := normalizeServiceName(row.ServiceName)
		if name == "" {
			continue
		}
		// DB 覆盖默认表（包括默认表中没有的新服务）
		rateMap[name] = row.FuelPerHour
	}
	return rateMap
}

// loadRateMapWithRepo 与 LoadRateMap 同语义，但接受显式 repo（供集成测试复用）。
func loadRateMapWithRepo(repo *repository.StructureServiceFuelRateRepository) map[string]float64 {
	rateMap := defaultRateMap()
	if repo == nil {
		return rateMap
	}
	rows, err := repo.ListAll()
	if err != nil {
		return rateMap
	}
	for _, row := range rows {
		name := normalizeServiceName(row.ServiceName)
		if name == "" {
			continue
		}
		rateMap[name] = row.FuelPerHour
	}
	return rateMap
}

// defaultRateMap 仅返回原始燃料率（不含类别信息），供回退。
func defaultRateMap() map[string]float64 {
	m := make(map[string]float64, len(defaultServiceFuelRates()))
	for name, info := range defaultServiceFuelRates() {
		m[name] = info.FuelPerHour
	}
	return m
}

// IsEmpty 判断映射表是否为空（用于启动时判断是否需要补跑同步）。
func (s *StructureFuelRateService) IsEmpty() (bool, error) {
	count, err := s.repo.Count()
	if err != nil {
		return false, err
	}
	return count == 0, nil
}

// SyncFuelRatesIfEmpty 当映射表为空时补跑一次同步。
// 返回是否实际执行了同步（以及同步错误）。由调用方（bootstrap）决定如何触发。
func (s *StructureFuelRateService) SyncFuelRatesIfEmpty(ctx context.Context) (bool, error) {
	empty, err := s.IsEmpty()
	if err != nil {
		return false, err
	}
	if !empty {
		return false, nil
	}
	if err := s.SyncFuelRates(ctx); err != nil {
		return true, err
	}
	return true, nil
}

// SyncFuelRates 从 ESI 拉取各服务模块的每小时燃料率，写库；ESI 失败则保留硬编码值。
func (s *StructureFuelRateService) SyncFuelRates(ctx context.Context) error {
	modules := builtinStructureServiceModules()

	// 收集去重的 typeID（多个 service name 可能映射同一 typeID）
	typeIDSet := make(map[int]struct{}, len(modules))
	for _, module := range modules {
		typeIDSet[module.TypeID] = struct{}{}
	}

	// typeID → ESI 实际燃料率 + typeName
	type esiResult struct {
		fuelPerHour float64
		typeName    string
		ok          bool
	}
	esiResults := make(map[int]esiResult, len(typeIDSet))
	for typeID := range typeIDSet {
		path := fmt.Sprintf("/universe/types/%d/", typeID)
		var resp esiTypeResponse
		if err := s.esiClient.Get(ctx, path, "", &resp); err != nil {
			logStructureFuelRateWarn("ESI 拉取服务模块燃料率失败，使用硬编码回退",
				err, zap.Int("type_id", typeID))
			continue
		}
		fuel := readDogmaFuelAmount(resp.DogmaAttributes)
		if fuel <= 0 {
			continue
		}
		esiResults[typeID] = esiResult{fuelPerHour: fuel, typeName: resp.Name, ok: true}
	}

	// 组装记录（每个 service name 一条；同 typeID 的多条共享 ESI 结果）
	// Build one row per physical module; display-name aliases are never
	// persisted by the synchroniser.
	rows := make([]model.StructureServiceFuelRate, 0, len(modules))
	hitCount := 0
	for _, module := range modules {
		row := module
		if res, ok := esiResults[module.TypeID]; ok && res.ok {
			row.FuelPerHour = res.fuelPerHour
			if res.typeName != "" {
				row.TypeName = res.typeName
			}
			hitCount++
		}
		rows = append(rows, row)
	}

	if err := s.repo.UpsertBatch(rows); err != nil {
		return fmt.Errorf("upsert 服务燃料率映射: %w", err)
	}

	if global.Logger != nil {
		global.Logger.Info(
			"服务燃料率同步完成",
			zap.Int("total", len(rows)),
			zap.Int("esi_hit", hitCount),
			zap.Int("fallback", len(rows)-hitCount),
		)
	}
	return nil
}

// readDogmaFuelAmount 从 dogma_attributes 中取 2109（serviceModuleFuelAmount）。
func readDogmaFuelAmount(attrs []esiDogmaAttribute) float64 {
	for _, a := range attrs {
		if a.AttributeID == dogmaAttrServiceModuleFuelAmount {
			return a.Value
		}
	}
	return 0
}

func logStructureFuelRateWarn(msg string, err error, fields ...zap.Field) {
	if global.Logger != nil {
		all := append([]zap.Field{zap.Error(err)}, fields...)
		global.Logger.Warn(msg, all...)
	}
}

// ─────────────────────────────────────────────
//  燃料消耗估算（纯函数，便于测试）
// ─────────────────────────────────────────────

// FuelEstimate 是 EstimateFuelPerHour 的返回值。
//
//   - FuelPerHour: 已识别在线服务的每小时燃料块合计（未乘 / 已乘结构系数后的结果）。
//     仅当 UnknownServices 为空时该值才完整可信；调用方应据此决定是否对外返回数值。
//   - UnknownServices: 归一化后仍无法在 rate map 命中的在线服务「原始名」列表
//     （保留 ESI 可读文本，便于前端展示）。非空表示估算不完整。
type FuelEstimate struct {
	FuelPerHour     float64
	UnknownServices []string
	Status          string
}

// EstimateFuelPerHour 估算建筑每小时燃料块消耗。
//   - structureGroupID: 建筑分组（来自 SDE invTypes.groupID，决定系数）
//   - structureTypeID: 建筑 typeID（精炼厂内区分 Athanor）
//   - services: 建筑 services 快照（name + state）
//   - rateMap: service name(归一化) → 原始每小时块数（DB 覆盖默认表）
//
// 仅累加 state == "online" 的服务。服务名经 normalizeServiceName 归一化后查表，
// 兼容 ESI 展示名（如 "Market" / "Clone Bay" / "Moon Drilling"）。
//
// 返回 FuelEstimate：能命中 rate map 的服务计入 FuelPerHour；
// 无法命中的在线服务原始名收集到 UnknownServices（不再静默丢弃）。
// 调用方应在 UnknownServices 非空时拒绝对外返回部分数值，避免低估。
//
// 缺失分组（0）或未知分组的服务按无折扣（系数 1.0）计算。
// FuelPerHour 四舍五入到两位小数。
func EstimateFuelPerHour(
	structureGroupID int,
	structureTypeID int64,
	services []CorporationStructureServiceInfo,
	rateMap map[string]float64,
) FuelEstimate {
	est := FuelEstimate{Status: fuelEstimateStatusAvailable}
	if len(services) == 0 || len(rateMap) == 0 {
		return est
	}
	defaults := defaultServiceFuelRates()
	total := 0.0
	for _, svc := range services {
		if !strings.EqualFold(svc.State, "online") {
			continue
		}
		name := normalizeServiceName(svc.Name)
		if name == "" {
			continue
		}
		rate, ok := rateMap[name]
		if !ok || rate <= 0 {
			est.UnknownServices = append(est.UnknownServices, svc.Name)
			continue
		}
		// 类别取自默认表（用于系数）；若默认表无则视为 Other（系数 1.0）
		cat := serviceCategoryOther
		if info, found := defaults[name]; found {
			cat = info.Category
		}
		total += rate * serviceFuelFactor(structureGroupID, structureTypeID, cat)
	}
	est.FuelPerHour = math.Round(total*100) / 100
	return est
}

// EstimateFuelToMonthEnd 估算「到耗尽月月底还需补充的燃料块数」。
//   - fuelExpires: ESI fuel_expires（date-time 字符串）
//   - rate: 每小时燃料块消耗（来自 EstimateFuelPerHour）
//   - now: 当前时间
//
// 规则：
//   - fuelExpires 为空 / 解析失败 / 已过期 → nil（无法估算）
//   - rate <= 0 → nil
//   - 目标 = fuelExpires 所在自然月（EVE UTC）的月底；hours = 月底 − fuelExpires；blocks = ceil(hours * rate)
func EstimateFuelToMonthEnd(fuelExpires string, rate float64, now time.Time) *int {
	if rate <= 0 {
		return nil
	}
	expiry, ok := parseFlexibleTime(fuelExpires)
	if !ok {
		return nil
	}
	if !expiry.After(now) {
		// 已过期，无需补充
		return nil
	}
	monthEnd := endOfMonth(expiry)
	hours := monthEnd.Sub(expiry).Hours()
	if hours <= 0 {
		return nil
	}
	blocks := int(math.Ceil(hours * rate))
	if blocks <= 0 {
		return nil
	}
	return &blocks
}

// endOfMonth 返回 t 所在自然月的最后一刻（下月 1 日 00:00 减 1 纳秒）。
// 沿用 t 的时区；EVE 时间相关字段均为 UTC（parseFlexibleTime 解析 RFC3339 保留时区）。
func endOfMonth(t time.Time) time.Time {
	firstOfNextMonth := time.Date(t.Year(), t.Month()+1, 1, 0, 0, 0, 0, t.Location())
	return firstOfNextMonth.Add(-time.Nanosecond)
}

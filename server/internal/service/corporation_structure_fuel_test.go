package service

import (
	"amiya-eden/global"
	"amiya-eden/internal/model"
	"amiya-eden/internal/repository"
	"amiya-eden/internal/utils"
	"context"
	"testing"
	"time"
)

// ─────────────────────────────────────────────
//  假 SDE 分组解析器
// ─────────────────────────────────────────────

// fakeGroupResolver 模拟 SdeRepository.GetTypes，按预设 typeID→groupID 返回。
type fakeGroupResolver struct {
	groupByTypeID map[int]int
}

func (f *fakeGroupResolver) GetTypes(typeIDs []int, published *bool, languageID string) ([]repository.TypeInfo, error) {
	result := make([]repository.TypeInfo, 0, len(typeIDs))
	for _, id := range typeIDs {
		if gid, ok := f.groupByTypeID[id]; ok {
			result = append(result, repository.TypeInfo{TypeID: id, GroupID: gid})
		}
	}
	return result, nil
}

// newCorpStructureServiceForFuelTest 构造带假分组解析器与真实燃料率仓库的服务。
// global.DB 必须在调用前指向测试库。
func newCorpStructureServiceForFuelTest(groupResolver StructureTypeGroupResolver) *CorporationStructureService {
	return &CorporationStructureService{
		roleRepo:      repository.NewRoleRepository(),
		charRepo:      repository.NewEveCharacterRepository(),
		sysConfigRepo: repository.NewSysConfigRepository(),
		sdeRepo:       repository.NewSdeRepository(),
		repo:          repository.NewCorporationStructureRepository(),
		esiClient:     nil,
		auditSvc:      NewAuditService(),
		nameResolver:  NewEntityNameResolver(),
		fuelRateRepo:  repository.NewStructureServiceFuelRateRepository(),
		groupResolver: groupResolver,
	}
}

// ─────────────────────────────────────────────
//  列表集成测试：燃料估算字段
// ─────────────────────────────────────────────

func TestCorporationStructureListFuelEstimates(t *testing.T) {
	db := newCorporationStructureServiceTestDB(t)
	// 燃料率表也需在同一个测试库
	if err := db.AutoMigrate(&model.StructureServiceFuelRate{}); err != nil {
		t.Fatalf("migrate fuel rate: %v", err)
	}
	oldDB := global.DB
	global.DB = db
	utils.InvalidateAllowCorporationsCache()
	t.Cleanup(func() {
		global.DB = oldDB
		utils.InvalidateAllowCorporationsCache()
	})

	seedCorporationStructureManageScope(t, db, 9001)

	// 持久化一条燃料率（Market 覆盖为 50，验证 DB 覆盖默认）。
	// DB 记录用 ESI 展示名（首字母大写），验证 normalizeServiceName 在加载时归一化。
	fuelRepo := repository.NewStructureServiceFuelRateRepository()
	if err := fuelRepo.UpsertBatch([]model.StructureServiceFuelRate{
		{ServiceName: "Market", TypeID: 35892, TypeName: "Standup Market Hub I", FuelPerHour: 50},
	}); err != nil {
		t.Fatalf("seed fuel rate: %v", err)
	}

	// Fortizar(35833, group 1657 堡垒) 装 Market(覆盖50)+Clone Bay(默认10)，系数 0.75 → (50+10)*0.75=45
	// 服务名使用 ESI 原始展示名（含空格/大小写），验证查表前归一化。
	if err := db.Create(&model.CorpStructureInfo{
		CorporationID:   9001,
		CorporationName: "Test Corp",
		StructureID:     111,
		Name:            "Fortizar with Market",
		TypeID:          35833,
		TypeName:        "Fortizar",
		SystemID:        30000142,
		SystemName:      "Jita",
		Security:        0.9,
		State:           "shield_vulnerable",
		Services:        `[{"name":"Market","state":"online"},{"name":"Clone Bay","state":"online"}]`,
		FuelExpires:     farFutureInCurrentMonth(time.Now()),
		UpdateAt:        time.Now().Unix(),
	}).Error; err != nil {
		t.Fatalf("seed fortizar: %v", err)
	}

	// 无在线服务的建筑 → 燃料字段为 nil
	if err := db.Create(&model.CorpStructureInfo{
		CorporationID: 9001,
		StructureID:   222,
		Name:          "Empty EC",
		TypeID:        35826,
		TypeName:      "Azbel",
		SystemID:      30002187,
		SystemName:    "Amarr",
		Security:      1.0,
		State:         "shield_vulnerable",
		Services:      `[]`,
		FuelExpires:   farFutureInCurrentMonth(time.Now()),
		UpdateAt:      time.Now().Unix(),
	}).Error; err != nil {
		t.Fatalf("seed empty ec: %v", err)
	}

	// 分组解析器：Fortizar=1657（堡垒），Azbel=1404（工业站）
	resolver := &fakeGroupResolver{groupByTypeID: map[int]int{
		35833: structureGroupCitadel,
		35826: structureGroupEngineeringComplex,
	}}
	svc := newCorpStructureServiceForFuelTest(resolver)

	resp, err := svc.ListStructures(context.Background(), CorporationStructureListRequest{CorporationID: 9001})
	if err != nil {
		t.Fatalf("ListStructures: %v", err)
	}
	byID := make(map[int64]CorporationStructureRow, len(resp.Items))
	for _, item := range resp.Items {
		byID[item.StructureID] = item
	}

	// Fortizar：(50+10)*0.75 = 45（Market+Clone Bay 归一化后均命中）
	fort := byID[111]
	if fort.FuelPerHour == nil {
		t.Fatal("Fortizar FuelPerHour 不应为 nil")
	}
	if *fort.FuelPerHour != 45 {
		t.Errorf("Fortizar FuelPerHour 期望 45（DB覆盖50+默认10，×0.75），实际 %v", *fort.FuelPerHour)
	}
	if fort.FuelEstimateIncomplete {
		t.Errorf("Fortizar FuelEstimateIncomplete 应为 false（所有在线服务已配置），实际 true；未知服务=%v", fort.FuelUnknownServices)
	}
	if fort.FuelToMonthEnd == nil {
		t.Fatal("Fortizar FuelToMonthEnd 不应为 nil（有未来 fuel_expires）")
	}

	// 空服务建筑 → 燃料字段 nil
	ec := byID[222]
	if ec.FuelPerHour != nil {
		t.Errorf("空服务建筑 FuelPerHour 应为 nil，实际 %v", *ec.FuelPerHour)
	}
	if ec.FuelToMonthEnd != nil {
		t.Errorf("空服务建筑 FuelToMonthEnd 应为 nil，实际 %v", *ec.FuelToMonthEnd)
	}
}

// ─────────────────────────────────────────────
//  分组缺失 → 无折扣（不报错）
// ─────────────────────────────────────────────

func TestCorporationStructureListFuelEstimates_MissingGroup(t *testing.T) {
	db := newCorporationStructureServiceTestDB(t)
	if err := db.AutoMigrate(&model.StructureServiceFuelRate{}); err != nil {
		t.Fatalf("migrate fuel rate: %v", err)
	}
	oldDB := global.DB
	global.DB = db
	utils.InvalidateAllowCorporationsCache()
	t.Cleanup(func() {
		global.DB = oldDB
		utils.InvalidateAllowCorporationsCache()
	})

	seedCorporationStructureManageScope(t, db, 9001)

	// 建筑用了未知 typeID（解析器不返回 groupID）→ 市场按无折扣 = 40
	if err := db.Create(&model.CorpStructureInfo{
		CorporationID:   9001,
		CorporationName: "Test Corp",
		StructureID:     333,
		Name:            "Unknown Type",
		TypeID:          99999,
		TypeName:        "Mystery",
		SystemID:        30000142,
		SystemName:      "Jita",
		Security:        0.9,
		State:           "shield_vulnerable",
		Services:        `[{"name":"market","state":"online"}]`,
		FuelExpires:     farFutureInCurrentMonth(time.Now()),
		UpdateAt:        time.Now().Unix(),
	}).Error; err != nil {
		t.Fatalf("seed: %v", err)
	}

	// 解析器不包含 99999 → groupID 缺失 → 无折扣
	resolver := &fakeGroupResolver{groupByTypeID: map[int]int{}}
	svc := newCorpStructureServiceForFuelTest(resolver)

	resp, err := svc.ListStructures(context.Background(), CorporationStructureListRequest{CorporationID: 9001})
	if err != nil {
		t.Fatalf("ListStructures: %v", err)
	}
	if len(resp.Items) != 1 {
		t.Fatalf("期望 1 项，实际 %d", len(resp.Items))
	}
	row := resp.Items[0]
	if row.FuelPerHour == nil {
		t.Fatal("FuelPerHour 不应为 nil")
	}
	// 市场默认 40，无折扣（分组缺失）→ 40
	if *row.FuelPerHour != 40 {
		t.Errorf("缺失分组应无折扣 market=40，实际 %v", *row.FuelPerHour)
	}
}

// ─────────────────────────────────────────────
//  存在未配置在线服务 → 不完整估算（不返回部分数值）
// ─────────────────────────────────────────────

func TestCorporationStructureListFuelEstimates_Incomplete(t *testing.T) {
	db := newCorporationStructureServiceTestDB(t)
	if err := db.AutoMigrate(&model.StructureServiceFuelRate{}); err != nil {
		t.Fatalf("migrate fuel rate: %v", err)
	}
	oldDB := global.DB
	global.DB = db
	utils.InvalidateAllowCorporationsCache()
	t.Cleanup(func() {
		global.DB = oldDB
		utils.InvalidateAllowCorporationsCache()
	})

	seedCorporationStructureManageScope(t, db, 9001)

	// Fortizar 同时挂 Market（已配置）和一个 ESI 未映射的在线服务。
	// 由于存在未配置在线服务 → 不完整估算，不返回部分燃料合计。
	if err := db.Create(&model.CorpStructureInfo{
		CorporationID:   9001,
		CorporationName: "Test Corp",
		StructureID:     777,
		Name:            "Fortizar Partial",
		TypeID:          35833,
		TypeName:        "Fortizar",
		SystemID:        30000142,
		SystemName:      "Jita",
		Security:        0.9,
		State:           "shield_vulnerable",
		Services:        `[{"name":"Market","state":"online"},{"name":"Future Module","state":"online"}]`,
		FuelExpires:     farFutureInCurrentMonth(time.Now()),
		UpdateAt:        time.Now().Unix(),
	}).Error; err != nil {
		t.Fatalf("seed: %v", err)
	}

	resolver := &fakeGroupResolver{groupByTypeID: map[int]int{35833: structureGroupCitadel}}
	svc := newCorpStructureServiceForFuelTest(resolver)

	resp, err := svc.ListStructures(context.Background(), CorporationStructureListRequest{CorporationID: 9001})
	if err != nil {
		t.Fatalf("ListStructures: %v", err)
	}
	if len(resp.Items) != 1 {
		t.Fatalf("期望 1 项，实际 %d", len(resp.Items))
	}
	row := resp.Items[0]
	// 不完整估算 → 燃料字段保持 nil，避免低估
	if row.FuelPerHour != nil {
		t.Errorf("不完整估算 FuelPerHour 应为 nil，实际 %v", *row.FuelPerHour)
	}
	if row.FuelToMonthEnd != nil {
		t.Errorf("不完整估算 FuelToMonthEnd 应为 nil，实际 %v", *row.FuelToMonthEnd)
	}
	if !row.FuelEstimateIncomplete {
		t.Error("FuelEstimateIncomplete 应为 true")
	}
	// 未映射服务原始名应出现在列表中
	found := false
	for _, name := range row.FuelUnknownServices {
		if name == "Future Module" {
			found = true
		}
	}
	if !found {
		t.Errorf("FuelUnknownServices 应含原始名 'Future Module'，实际 %v", row.FuelUnknownServices)
	}
}

// ─────────────────────────────────────────────
//  指派列表不做燃料估算
// ─────────────────────────────────────────────

func TestCorporationStructureAssignmentList_NoFuelEstimate(t *testing.T) {
	db := newCorporationStructureServiceTestDB(t)
	if err := db.AutoMigrate(&model.StructureServiceFuelRate{}); err != nil {
		t.Fatalf("migrate fuel rate: %v", err)
	}
	oldDB := global.DB
	global.DB = db
	utils.InvalidateAllowCorporationsCache()
	t.Cleanup(func() {
		global.DB = oldDB
		utils.InvalidateAllowCorporationsCache()
	})

	seedCorporationStructureManageScope(t, db, 9001)

	if err := db.Create(&model.CorpStructureInfo{
		CorporationID:   9001,
		CorporationName: "Test Corp",
		StructureID:     444,
		Name:            "Has Services",
		TypeID:          35833,
		TypeName:        "Fortizar",
		SystemID:        30000142,
		SystemName:      "Jita",
		Security:        0.9,
		State:           "shield_vulnerable",
		Services:        `[{"name":"market","state":"online"}]`,
		FuelExpires:     farFutureInCurrentMonth(time.Now()),
		UpdateAt:        time.Now().Unix(),
	}).Error; err != nil {
		t.Fatalf("seed: %v", err)
	}

	// 即使注入了解析器，指派列表也不应触发燃料估算
	resolver := &fakeGroupResolver{groupByTypeID: map[int]int{35833: structureGroupCitadel}}
	svc := newCorpStructureServiceForFuelTest(resolver)

	resp, err := svc.GetAssignments(context.Background(), CorporationStructureFilterOptionsRequest{CorporationID: 9001})
	if err != nil {
		t.Fatalf("GetAssignments: %v", err)
	}
	if len(resp.Items) != 1 {
		t.Fatalf("期望 1 项，实际 %d", len(resp.Items))
	}
	// 指派项不含燃料字段（CorporationStructureAssignmentItem 无这些字段）
	// 仅验证不 panic、返回正确结构
	if resp.Items[0].StructureID != 444 {
		t.Errorf("期望 structure_id=444，实际 %d", resp.Items[0].StructureID)
	}
}

// ─────────────────────────────────────────────
//  燃料官列表也填充燃料估算字段
// ─────────────────────────────────────────────

func TestCorporationStructureMyAssignedListFuelEstimates(t *testing.T) {
	db := newCorporationStructureServiceTestDB(t)
	if err := db.AutoMigrate(&model.StructureServiceFuelRate{}); err != nil {
		t.Fatalf("migrate fuel rate: %v", err)
	}
	oldDB := global.DB
	global.DB = db
	utils.InvalidateAllowCorporationsCache()
	t.Cleanup(func() {
		global.DB = oldDB
		utils.InvalidateAllowCorporationsCache()
	})

	seedCorporationStructureManageScope(t, db, 9001)

	// 燃料官用户（user_id=2）
	if err := db.Create(&model.User{BaseModel: model.BaseModel{ID: 2}, Nickname: "fuel", Role: model.RoleFuelOfficer}).Error; err != nil {
		t.Fatalf("create fuel officer: %v", err)
	}
	if err := db.Create(&model.UserRole{UserID: 2, RoleCode: model.RoleFuelOfficer}).Error; err != nil {
		t.Fatalf("create fuel officer role: %v", err)
	}

	// 建筑 + 指派给燃料官。服务名使用 ESI 原始展示名（验证归一化查表）。
	if err := db.Create(&model.CorpStructureInfo{
		CorporationID:   9001,
		CorporationName: "Test Corp",
		StructureID:     555,
		Name:            "My Azbel",
		TypeID:          35826,
		TypeName:        "Azbel",
		SystemID:        30000142,
		SystemName:      "Jita",
		Security:        0.9,
		State:           "shield_vulnerable",
		Services:        `[{"name":"Manufacturing","state":"online"}]`,
		FuelExpires:     farFutureInCurrentMonth(time.Now()),
		UpdateAt:        time.Now().Unix(),
	}).Error; err != nil {
		t.Fatalf("seed structure: %v", err)
	}
	if err := db.Create(&model.CorpStructureAssignment{
		CorporationID:  9001,
		StructureID:    555,
		AssignedUserID: 2,
	}).Error; err != nil {
		t.Fatalf("seed assignment: %v", err)
	}

	// Azbel(35826) → group 1404（工业站），manufacturing 默认 12，系数 0.75 → 9
	resolver := &fakeGroupResolver{groupByTypeID: map[int]int{35826: structureGroupEngineeringComplex}}
	svc := newCorpStructureServiceForFuelTest(resolver)

	resp, err := svc.ListMyAssignedStructures(context.Background(), 2, CorporationStructureListRequest{})
	if err != nil {
		t.Fatalf("ListMyAssignedStructures: %v", err)
	}
	if len(resp.Items) != 1 {
		t.Fatalf("期望 1 项，实际 %d", len(resp.Items))
	}
	row := resp.Items[0]
	if row.FuelPerHour == nil {
		t.Fatal("燃料官列表 FuelPerHour 不应为 nil")
	}
	if *row.FuelPerHour != 9 {
		t.Errorf("Azbel manufacturing 期望 9（12×0.75），实际 %v", *row.FuelPerHour)
	}
	if row.FuelToMonthEnd == nil {
		t.Fatal("燃料官列表 FuelToMonthEnd 不应为 nil")
	}
}

// ─────────────────────────────────────────────
//  辅助
// ─────────────────────────────────────────────

// farFutureInCurrentMonth 返回当前自然月内一个未来的 RFC3339 时间字符串，
// 保证 EstimateFuelToMonthEnd 不返回 nil（本月耗尽 → 算到本月月底）。
func farFutureInCurrentMonth(now time.Time) string {
	// 取本月最后一天 18:00 UTC（必然在 now 之后，除非今天是月底最后几小时）
	end := endOfMonth(now)
	// 留 6 小时余量
	ts := end.Add(-6 * time.Hour)
	if !ts.After(now) {
		// 极端情况（月底最后6小时内）：用下月月中
		ts = time.Date(now.Year(), now.Month()+2, 15, 0, 0, 0, 0, time.UTC)
	}
	return ts.Format(time.RFC3339)
}

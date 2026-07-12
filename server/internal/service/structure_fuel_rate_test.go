package service

import (
	"amiya-eden/global"
	"amiya-eden/internal/model"
	"amiya-eden/internal/repository"
	"context"
	"errors"
	"math"
	"reflect"
	"sync/atomic"
	"testing"
	"time"

	"gorm.io/gorm"
)

// ─────────────────────────────────────────────
//  normalizeServiceName — ESI 展示名归一化
// ─────────────────────────────────────────────

func TestNormalizeServiceName(t *testing.T) {
	tests := []struct {
		in   string
		want string
	}{
		{"Market", "market"},
		{"Clone Bay", "clone_bay"},
		{"Moon Drilling", "moon_drilling"},
		{"clone_bay", "clone_bay"},
		{"moon_drilling", "moon_drilling"},
		{"moon-drilling", "moon_drilling"},
		{"Capital Shipyard", "capital_shipyard"},
		{"  Capital   Ship-Yard ", "capital_ship_yard"},
		{"Cynosural  Generator", "cynosural_generator"}, // 连续空格折叠为单个下划线
		{"  ", ""},
		{"", ""},
	}
	for _, tt := range tests {
		t.Run(tt.in, func(t *testing.T) {
			if got := normalizeServiceName(tt.in); got != tt.want {
				t.Errorf("normalizeServiceName(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

// ─────────────────────────────────────────────
//  EstimateFuelPerHour — 分组折扣
// ─────────────────────────────────────────────

func TestEstimateFuelPerHour(t *testing.T) {
	rateMap := defaultRateMap()

	tests := []struct {
		name         string
		groupID      int
		structureID  int64
		services     []CorporationStructureServiceInfo
		want         float64
		wantExact    bool
		wantUnknown  []string
		wantUnknownN int // 当不关心具体内容、只关心数量时用 -1 表示「断言非空」
	}{
		{
			name:     "空服务返回 0",
			groupID:  structureGroupCitadel,
			services: nil,
			want:     0,
		},
		{
			name:        "工业站 Azbel(1404) 三个 12 块服务应用 0.75 = 27",
			groupID:     structureGroupEngineeringComplex,
			structureID: 35826,
			services: []CorporationStructureServiceInfo{
				{Name: "manufacturing", State: "online"},
				{Name: "research_lab", State: "online"},
				{Name: "invention_lab", State: "online"},
			},
			want:      27,
			wantExact: true,
		},
		{
			name:        "Athanor(1406,type35835) 再处理应用 0.80 = 8",
			groupID:     structureGroupRefinery,
			structureID: structureTypeIDAthanor,
			services: []CorporationStructureServiceInfo{
				{Name: "reprocessing", State: "online"},
			},
			want:      8,
			wantExact: true,
		},
		{
			name:        "Tatara(1406,type35836) 再处理应用 0.75 = 7.5",
			groupID:     structureGroupRefinery,
			structureID: 35836,
			services: []CorporationStructureServiceInfo{
				{Name: "reprocessing", State: "online"},
			},
			want:      7.5,
			wantExact: true,
		},
		{
			name:        "Fortizar(1657) 市场+克隆应用 0.75 = 37.5",
			groupID:     structureGroupCitadel,
			structureID: 35833,
			services: []CorporationStructureServiceInfo{
				{Name: "market", State: "online"},
				{Name: "clone_bay", State: "online"},
			},
			want:      37.5,
			wantExact: true,
		},
		{
			name:        "错误分组不折扣：堡垒分组下的制造服务 = 12（系数 1.0）",
			groupID:     structureGroupCitadel,
			structureID: 35833,
			services: []CorporationStructureServiceInfo{
				{Name: "manufacturing", State: "online"},
			},
			want:      12,
			wantExact: true,
		},
		{
			name:        "错误分组不折扣：工业站分组下的市场 = 40（系数 1.0）",
			groupID:     structureGroupEngineeringComplex,
			structureID: 35826,
			services: []CorporationStructureServiceInfo{
				{Name: "market", State: "online"},
			},
			want:      40,
			wantExact: true,
		},
		{
			name:        "未知分组（0）不折扣 = 12",
			groupID:     0,
			structureID: 99999,
			services: []CorporationStructureServiceInfo{
				{Name: "manufacturing", State: "online"},
			},
			want:      12,
			wantExact: true,
		},
		{
			name:        "未知分组（99999）不折扣 = 40",
			groupID:     99999,
			structureID: 99999,
			services: []CorporationStructureServiceInfo{
				{Name: "market", State: "online"},
			},
			want:      40,
			wantExact: true,
		},
		{
			name:        "offline 服务不计入",
			groupID:     structureGroupEngineeringComplex,
			structureID: 35826,
			services: []CorporationStructureServiceInfo{
				{Name: "manufacturing", State: "offline"},
				{Name: "research_lab", State: "online"},
			},
			want:      9, // 12 * 0.75
			wantExact: true,
		},
		{
			name:        "未配置在线服务产生不完整估算，已知部分仍累加且 UnknownServices 含原始名",
			groupID:     structureGroupCitadel,
			structureID: 35833,
			services: []CorporationStructureServiceInfo{
				{Name: "unknown_service", State: "online"},
				{Name: "market", State: "online"},
			},
			want:        30, // 已知 market 部分仍累加：40 * 0.75（调用方据 UnknownServices 决定是否对外展示）
			wantExact:   true,
			wantUnknown: []string{"unknown_service"},
		},
		{
			name:        "跳桥等无折扣服务系数 1.0（无论分组）",
			groupID:     structureGroupCitadel,
			structureID: 35833,
			services: []CorporationStructureServiceInfo{
				{Name: "jump_bridge", State: "online"},
			},
			want:      30, // 30 * 1.0
			wantExact: true,
		},
		{
			name:        "industry 别名等同 manufacturing",
			groupID:     structureGroupEngineeringComplex,
			structureID: 35826,
			services: []CorporationStructureServiceInfo{
				{Name: "industry", State: "online"},
			},
			want:      9, // 12 * 0.75
			wantExact: true,
		},
		// ── 规范化修复回归（ESI 展示名含空格/大小写）──
		{
			name:        "ESI 展示名 Market+Clone Bay 在堡垒归一化后命中 = 37.5",
			groupID:     structureGroupCitadel,
			structureID: 35833,
			services: []CorporationStructureServiceInfo{
				{Name: "Market", State: "online"},
				{Name: "Clone Bay", State: "online"},
			},
			want:      37.5, // (40+10)*0.75
			wantExact: true,
		},
		{
			name:        "ESI 展示名 Moon Drilling 归一化后命中 = 5",
			groupID:     structureGroupCitadel,
			structureID: 35833,
			services: []CorporationStructureServiceInfo{
				{Name: "Moon Drilling", State: "online"},
			},
			want:      5, // 5 * 1.0（无折扣类别）
			wantExact: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := EstimateFuelPerHour(tt.groupID, tt.structureID, tt.services, rateMap)
			if tt.wantUnknown != nil {
				if !reflect.DeepEqual(got.UnknownServices, tt.wantUnknown) {
					t.Errorf("UnknownServices 期望 %v，实际 %v", tt.wantUnknown, got.UnknownServices)
				}
			} else if tt.wantUnknownN < 0 {
				if len(got.UnknownServices) == 0 {
					t.Errorf("期望 UnknownServices 非空，实际为空")
				}
			} else if len(got.UnknownServices) != 0 {
				t.Errorf("期望 UnknownServices 为空，实际 %v", got.UnknownServices)
			}
			if !tt.wantExact {
				if got.FuelPerHour != 0 {
					t.Errorf("期望 FuelPerHour 0，实际 %v", got.FuelPerHour)
				}
				return
			}
			if math.Abs(got.FuelPerHour-tt.want) > 0.01 {
				t.Errorf("期望 %v，实际 %v", tt.want, got.FuelPerHour)
			}
		})
	}
}

func TestEstimateFuelPerHour_EmptyRateMap(t *testing.T) {
	got := EstimateFuelPerHour(structureGroupCitadel, 35833, []CorporationStructureServiceInfo{
		{Name: "market", State: "online"},
	}, map[string]float64{})
	if got.FuelPerHour != 0 {
		t.Errorf("空 rateMap 应返回 FuelPerHour 0，实际 %v", got.FuelPerHour)
	}
	// 空 rateMap 不视为「不完整估算」（直接短路，不收集未知服务）
	if len(got.UnknownServices) != 0 {
		t.Errorf("空 rateMap 应 UnknownServices 为空，实际 %v", got.UnknownServices)
	}
}

// ─────────────────────────────────────────────
//  EstimateFuelToMonthEnd — UTC 跨月与未来耗尽月
// ─────────────────────────────────────────────

func TestEstimateFuelToMonthEnd(t *testing.T) {
	now := time.Date(2026, 7, 11, 12, 0, 0, 0, time.UTC)

	tests := []struct {
		name        string
		fuelExpires string
		rate        float64
		wantNil     bool
		checkVal    func(t *testing.T, got *int)
	}{
		{
			name:        "rate<=0 返回 nil",
			fuelExpires: "2026-07-20T00:00:00Z",
			rate:        0,
			wantNil:     true,
		},
		{
			name:        "fuelExpires 为空返回 nil",
			fuelExpires: "",
			rate:        10,
			wantNil:     true,
		},
		{
			name:        "已过期返回 nil",
			fuelExpires: "2026-06-01T00:00:00Z",
			rate:        10,
			wantNil:     true,
		},
		{
			name:        "本月耗尽 → 算到本月(7月)月底",
			fuelExpires: "2026-07-20T00:00:00Z",
			rate:        10,
			checkVal: func(t *testing.T, got *int) {
				if got == nil {
					t.Fatal("期望非 nil")
				}
				// 7 月底 2026-08-01 UTC；expiry 2026-07-20 → 12*24=288h；288*10=2880
				if *got != 2880 {
					t.Errorf("期望 2880，实际 %d", *got)
				}
			},
		},
		{
			name:        "未来某月耗尽 → 算到该月月底",
			fuelExpires: "2026-08-15T00:00:00Z",
			rate:        1,
			checkVal: func(t *testing.T, got *int) {
				if got == nil {
					t.Fatal("期望非 nil")
				}
				// 8 月底 2026-09-01 UTC；expiry 2026-08-15 → 17*24=408h；408*1=408
				if *got != 408 {
					t.Errorf("期望 408，实际 %d", *got)
				}
			},
		},
		{
			name:        "小数 rate 向上取整",
			fuelExpires: "2026-07-20T00:00:00Z",
			rate:        0.1,
			checkVal: func(t *testing.T, got *int) {
				if got == nil {
					t.Fatal("期望非 nil")
				}
				// 288 * 0.1 = 28.8 → ceil = 29
				if *got != 29 {
					t.Errorf("期望 29，实际 %d", *got)
				}
			},
		},
		{
			name:        "UTC 跨月：9月底耗尽算到9月底（now=7月，未来）",
			fuelExpires: "2026-09-28T00:00:00Z",
			rate:        1,
			checkVal: func(t *testing.T, got *int) {
				if got == nil {
					t.Fatal("期望非 nil")
				}
				// now=7月，expiry=9月28日（未来）；9月底 2026-10-01 UTC；28日→10月1日 = 3天 = 72h
				if *got != 72 {
					t.Errorf("期望 72，实际 %d", *got)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := EstimateFuelToMonthEnd(tt.fuelExpires, tt.rate, now)
			if tt.wantNil {
				if got != nil {
					t.Errorf("期望 nil，实际 %v", *got)
				}
				return
			}
			tt.checkVal(t, got)
		})
	}
}

func TestEndOfMonth(t *testing.T) {
	tests := []struct {
		name string
		t    time.Time
		want time.Time
	}{
		{"7月", time.Date(2026, 7, 15, 0, 0, 0, 0, time.UTC), time.Date(2026, 7, 31, 23, 59, 59, 999999999, time.UTC)},
		{"2月非闰年", time.Date(2025, 2, 1, 0, 0, 0, 0, time.UTC), time.Date(2025, 2, 28, 23, 59, 59, 999999999, time.UTC)},
		{"12月跨年", time.Date(2026, 12, 15, 0, 0, 0, 0, time.UTC), time.Date(2026, 12, 31, 23, 59, 59, 999999999, time.UTC)},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := endOfMonth(tt.t)
			if !got.Equal(tt.want) {
				t.Errorf("期望 %v，实际 %v", tt.want, got)
			}
		})
	}
}

// ─────────────────────────────────────────────
//  DB 覆盖默认表（overlay）—— loadRateMapWithRepo
// ─────────────────────────────────────────────

func TestLoadRateMapWithRepo_Overlay(t *testing.T) {
	db := newStructureFuelRateTestDB(t)
	oldDB := global.DB
	global.DB = db
	t.Cleanup(func() { global.DB = oldDB })

	repo := repository.NewStructureServiceFuelRateRepository()

	// 空表 → 纯默认表
	m := loadRateMapWithRepo(repo)
	if m["market"] != 40 {
		t.Errorf("空表应回退默认值 market=40，实际 %v", m["market"])
	}

	// DB 覆盖 market 为 99，并新增一个默认表没有的服务
	if err := repo.UpsertBatch([]model.StructureServiceFuelRate{
		{ServiceName: "market", TypeID: 35892, TypeName: "Standup Market Hub I", FuelPerHour: 99},
		{ServiceName: "custom_new_service", TypeID: 12345, TypeName: "Custom", FuelPerHour: 7},
	}); err != nil {
		t.Fatalf("upsert: %v", err)
	}

	m = loadRateMapWithRepo(repo)
	// DB 覆盖默认值
	if m["market"] != 99 {
		t.Errorf("DB 应覆盖 market=99，实际 %v", m["market"])
	}
	// 默认表的服务仍保留（未被 DB 清空）
	if m["manufacturing"] != 12 {
		t.Errorf("默认表 manufacturing=12 应保留，实际 %v", m["manufacturing"])
	}
	// DB 新增的默认表没有的服务
	if m["custom_new_service"] != 7 {
		t.Errorf("DB 新增 custom_new_service=7 应出现，实际 %v", m["custom_new_service"])
	}
}

// TestUpsertBatch_ReinsertUpdatesByServiceName 验证对已存在的 service_name
// 再次 UpsertBatch 会走 UPDATE 而非 INSERT（不抛 23505 unique constraint）。
// 这是二次同步（10 天后再次跑 structure_fuel_rate_sync）的路径。
func TestUpsertBatch_ReinsertUpdatesByServiceName(t *testing.T) {
	db := newStructureFuelRateTestDB(t)
	oldDB := global.DB
	global.DB = db
	t.Cleanup(func() { global.DB = oldDB })

	repo := repository.NewStructureServiceFuelRateRepository()

	// 第一次写入（模拟首次同步）
	first := []model.StructureServiceFuelRate{
		{ServiceName: "market", TypeID: 35892, TypeName: "Standup Market Hub I", FuelPerHour: 40},
		{ServiceName: "manufacturing", TypeID: 35878, TypeName: "Standup Manufacturing Plant I", FuelPerHour: 12},
	}
	if err := repo.UpsertBatch(first); err != nil {
		t.Fatalf("第一次 upsert: %v", err)
	}
	count, _ := repo.Count()
	if count != 2 {
		t.Fatalf("第一次后应有 2 条，实际 %d", count)
	}

	// 第二次写入相同 service_name 但 fuel_per_hour 不同（模拟二次同步，ESI 覆盖）
	second := []model.StructureServiceFuelRate{
		{ServiceName: "market", TypeID: 35892, TypeName: "Standup Market Hub I (ESI)", FuelPerHour: 99},
		{ServiceName: "manufacturing", TypeID: 35878, TypeName: "Standup Manufacturing Plant I", FuelPerHour: 12},
	}
	if err := repo.UpsertBatch(second); err != nil {
		t.Fatalf("第二次 upsert 不应报 unique constraint（23505）: %v", err)
	}

	// 仍应是 2 条（更新而非新增）
	count, _ = repo.Count()
	if count != 2 {
		t.Fatalf("第二次后仍应 2 条（UPDATE 而非 INSERT），实际 %d", count)
	}

	// market 的 fuel_per_hour 应被更新为 99
	rows, _ := repo.ListAll()
	byName := make(map[string]model.StructureServiceFuelRate, len(rows))
	for _, r := range rows {
		byName[r.ServiceName] = r
	}
	if byName["market"].FuelPerHour != 99 {
		t.Errorf("market fuel_per_hour 应更新为 99，实际 %v", byName["market"].FuelPerHour)
	}
	if byName["market"].TypeName != "Standup Market Hub I (ESI)" {
		t.Errorf("market type_name 应更新，实际 %v", byName["market"].TypeName)
	}
}

// ─────────────────────────────────────────────
//  SyncFuelRates —— 假 ESI 注入
// ─────────────────────────────────────────────

// fakeESITypeFetcher 模拟 ESI /universe/types/{id}/ 响应。
type fakeESITypeFetcher struct {
	// typeID → 响应；未配置的 typeID 返回 err（触发回退）
	responses map[int]esiTypeResponse
	err       error
	callCount int32
}

func (f *fakeESITypeFetcher) Get(ctx context.Context, path string, accessToken string, dest interface{}) error {
	atomic.AddInt32(&f.callCount, 1)
	if f.err != nil {
		return f.err
	}
	typeID, err := parseIntFromPath(path, "/universe/types/")
	if err != nil {
		return err
	}
	resp, ok := f.responses[typeID]
	if !ok {
		return errors.New("esi: not configured for this type")
	}
	if d, ok := dest.(*esiTypeResponse); ok {
		*d = resp
	}
	return nil
}

func newStructureFuelRateServiceWithFakeESI(t *testing.T, fetcher *fakeESITypeFetcher) *StructureFuelRateService {
	t.Helper()
	db := newStructureFuelRateTestDB(t)
	oldDB := global.DB
	global.DB = db
	t.Cleanup(func() { global.DB = oldDB })
	return &StructureFuelRateService{
		repo:      repository.NewStructureServiceFuelRateRepository(),
		esiClient: fetcher,
	}
}

func TestSyncFuelRates_ESIOverridesDefault(t *testing.T) {
	// 假 ESI：market(35892) 返回 2109=99，其余 typeID 未配置（回退默认）
	fetcher := &fakeESITypeFetcher{
		responses: map[int]esiTypeResponse{
			35892: {
				TypeID: 35892,
				Name:   "Standup Market Hub I (ESI)",
				DogmaAttributes: []esiDogmaAttribute{
					{AttributeID: 2109, Value: 99},
				},
			},
		},
	}
	svc := newStructureFuelRateServiceWithFakeESI(t, fetcher)

	if err := svc.SyncFuelRates(context.Background()); err != nil {
		t.Fatalf("SyncFuelRates: %v", err)
	}

	rows, err := svc.repo.ListAll()
	if err != nil {
		t.Fatalf("ListAll: %v", err)
	}
	byName := make(map[string]model.StructureServiceFuelRate, len(rows))
	for _, r := range rows {
		byName[r.ServiceName] = r
	}

	// ESI 命中：market 覆盖为 99
	if byName["market"].FuelPerHour != 99 {
		t.Errorf("market 应被 ESI 覆盖为 99，实际 %v", byName["market"].FuelPerHour)
	}
	if byName["market"].TypeName != "Standup Market Hub I (ESI)" {
		t.Errorf("market typeName 应被 ESI 覆盖，实际 %v", byName["market"].TypeName)
	}
	// ESI 未命中：manufacturing 保留默认 12
	if byName["manufacturing"].FuelPerHour != 12 {
		t.Errorf("manufacturing 应保留默认 12，实际 %v", byName["manufacturing"].FuelPerHour)
	}
}

func TestSyncFuelRates_AllESIFails_KeepsDefaults(t *testing.T) {
	fetcher := &fakeESITypeFetcher{err: errors.New("network down")}
	svc := newStructureFuelRateServiceWithFakeESI(t, fetcher)

	if err := svc.SyncFuelRates(context.Background()); err != nil {
		t.Fatalf("SyncFuelRates 应在 ESI 全失败时仍写默认值（不报错）: %v", err)
	}

	rows, err := svc.repo.ListAll()
	if err != nil {
		t.Fatalf("ListAll: %v", err)
	}
	if len(rows) == 0 {
		t.Fatal("ESI 全失败时仍应写入默认值")
	}
	byName := make(map[string]model.StructureServiceFuelRate, len(rows))
	for _, r := range rows {
		byName[r.ServiceName] = r
	}
	// 全部保留默认
	if byName["market"].FuelPerHour != 40 {
		t.Errorf("market 默认 40，实际 %v", byName["market"].FuelPerHour)
	}
	if byName["manufacturing"].FuelPerHour != 12 {
		t.Errorf("manufacturing 默认 12，实际 %v", byName["manufacturing"].FuelPerHour)
	}
}

func TestSyncFuelRates_SingleESIFails_OthersSucceed(t *testing.T) {
	// 配置 manufacturing(35878) 成功，其余未配置（失败）
	fetcher := &fakeESITypeFetcher{
		responses: map[int]esiTypeResponse{
			35878: {
				TypeID: 35878,
				Name:   "Standup Manufacturing Plant I (ESI)",
				DogmaAttributes: []esiDogmaAttribute{
					{AttributeID: 2109, Value: 50},
				},
			},
		},
	}
	svc := newStructureFuelRateServiceWithFakeESI(t, fetcher)

	if err := svc.SyncFuelRates(context.Background()); err != nil {
		t.Fatalf("SyncFuelRates: %v", err)
	}

	rows, _ := svc.repo.ListAll()
	byName := make(map[string]model.StructureServiceFuelRate, len(rows))
	for _, r := range rows {
		byName[r.ServiceName] = r
	}
	// 命中的覆盖
	if byName["manufacturing"].FuelPerHour != 50 {
		t.Errorf("manufacturing 应覆盖为 50，实际 %v", byName["manufacturing"].FuelPerHour)
	}
	// 未命中的保留默认
	if byName["market"].FuelPerHour != 40 {
		t.Errorf("market 应保留默认 40，实际 %v", byName["market"].FuelPerHour)
	}
}

func TestSyncFuelRatesIfEmpty(t *testing.T) {
	fetcher := &fakeESITypeFetcher{
		responses: map[int]esiTypeResponse{
			35892: {TypeID: 35892, Name: "Market", DogmaAttributes: []esiDogmaAttribute{{AttributeID: 2109, Value: 40}}},
		},
	}
	svc := newStructureFuelRateServiceWithFakeESI(t, fetcher)

	// 空表 → 应执行同步
	ran, err := svc.SyncFuelRatesIfEmpty(context.Background())
	if err != nil {
		t.Fatalf("SyncFuelRatesIfEmpty: %v", err)
	}
	if !ran {
		t.Fatal("空表应触发同步 ran=true")
	}
	count, _ := svc.repo.Count()
	if count == 0 {
		t.Fatal("同步后表不应为空")
	}

	// 非空表 → 不应再同步
	ran2, err := svc.SyncFuelRatesIfEmpty(context.Background())
	if err != nil {
		t.Fatalf("第二次 SyncFuelRatesIfEmpty: %v", err)
	}
	if ran2 {
		t.Fatal("非空表不应再触发同步 ran2=false")
	}
}

// ─────────────────────────────────────────────
//  辅助
// ─────────────────────────────────────────────

// newStructureFuelRateTestDB 创建仅含燃料率表的内存 SQLite。
func newStructureFuelRateTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	return newServiceTestDB(t, "fuel_rate_test", &model.StructureServiceFuelRate{})
}

// parseIntFromPath 从形如 "/universe/types/35878/" 的路径中解析末尾数字。
func parseIntFromPath(path, prefix string) (int, error) {
	if len(path) <= len(prefix) {
		return 0, errors.New("path too short")
	}
	rest := path[len(prefix):]
	for len(rest) > 0 && rest[len(rest)-1] == '/' {
		rest = rest[:len(rest)-1]
	}
	var n int
	for _, c := range rest {
		if c < '0' || c > '9' {
			return 0, errors.New("not a number")
		}
		n = n*10 + int(c-'0')
	}
	return n, nil
}

package service

import (
	"amiya-eden/global"
	"amiya-eden/internal/model"
	"amiya-eden/internal/repository"
	"testing"
)

func TestReconcileStructureServiceCatalogMigratesLegacyActivityMappings(t *testing.T) {
	db := newServiceTestDB(t, "structure_catalog", &model.StructureServiceFuelRate{}, &model.StructureServiceActivity{}, &model.StructureServiceActivityCandidate{})
	previous := global.DB
	global.DB = db
	t.Cleanup(func() { global.DB = previous })
	if err := db.Create(&model.StructureServiceActivity{ActivityName: "Future Activity", TypeID: 45537}).Error; err != nil {
		t.Fatalf("seed legacy activity: %v", err)
	}
	if err := db.Create(&model.StructureServiceActivity{ActivityName: "Moon Drilling", TypeID: 82941}).Error; err != nil {
		t.Fatalf("seed stale legacy moon drilling mapping: %v", err)
	}
	if err := ReconcileStructureServiceCatalog(); err != nil {
		t.Fatalf("reconcile catalog: %v", err)
	}
	activities := loadActivityCandidates(repository.NewStructureServiceActivityCandidateRepository())
	if got := activities["moon_drilling"]; len(got) != 1 || got[0] != 45009 {
		t.Fatalf("moon drilling candidates = %v, want [45009]", got)
	}
	if got := activities["future_activity"]; len(got) != 1 || got[0] != 45537 {
		t.Fatalf("legacy candidate = %v, want [45537]", got)
	}
}

func TestEffectiveServiceModuleCatalogIncludesDefaultsAndOverrides(t *testing.T) {
	catalog := effectiveServiceModuleCatalog([]model.StructureServiceFuelRate{
		{ServiceName: "market", TypeID: 35892, TypeName: "Verified Market", FuelPerHour: 41, FuelCategory: "citadel"},
		{ServiceName: "new_module", TypeID: 999999, TypeName: "Verified New Module", FuelPerHour: 7, FuelCategory: "other"},
	})
	byTypeID := make(map[int]model.StructureServiceFuelRate, len(catalog))
	for _, row := range catalog {
		byTypeID[row.TypeID] = row
	}
	if got, ok := byTypeID[82941]; !ok || got.FuelPerHour != 5 {
		t.Fatalf("default Metenox module = %#v, want verified default with rate 5", got)
	}
	if got := byTypeID[35892]; got.TypeName != "Verified Market" || got.FuelPerHour != 41 {
		t.Fatalf("persisted module override = %#v, want database row", got)
	}
	if _, ok := byTypeID[999999]; !ok {
		t.Fatalf("administrator module missing from effective catalogue: %#v", catalog)
	}
}

func TestEstimateFuelFromModules_DeduplicatesActivitiesByPhysicalModule(t *testing.T) {
	rates := defaultModuleFuelRates()
	activities := builtinActivityCandidates()
	estimate := EstimateFuelFromModules(
		structureGroupEngineeringComplex,
		35826,
		[]CorporationStructureServiceInfo{
			{Name: "Material Efficiency Research", State: "online"},
			{Name: "Blueprint Copying", State: "online"},
			{Name: "Time Efficiency Research", State: "online"},
			{Name: "Invention", State: "online"},
			{Name: "Manufacturing (Standard)", State: "online"},
			{Name: "Manufacturing (Capitals)", State: "online"},
		},
		true,
		[]structureServiceModuleSnapshot{
			{TypeID: 35891, Slot: "ServiceSlot0"},
			{TypeID: 35886, Slot: "ServiceSlot1"},
			{TypeID: 35878, Slot: "ServiceSlot2"},
			{TypeID: 35881, Slot: "ServiceSlot3"},
		},
		activities,
		rates,
	)
	if estimate.Status != fuelEstimateStatusAvailable || estimate.FuelPerHour != 45 {
		t.Fatalf("estimate = %#v, want status available and 45", estimate)
	}
}

func TestEstimateFuelFromModules_RejectsUnsafeInputs(t *testing.T) {
	rates := defaultModuleFuelRates()
	activities := builtinActivityCandidates()
	tests := []struct {
		name    string
		known   bool
		modules []structureServiceModuleSnapshot
		service []CorporationStructureServiceInfo
		status  string
	}{
		{"authorization missing", false, nil, []CorporationStructureServiceInfo{{Name: "Market", State: "online"}}, fuelEstimateStatusAuthorizationRequired},
		{"module missing", true, nil, []CorporationStructureServiceInfo{{Name: "Market", State: "online"}}, fuelEstimateStatusModuleMismatch},
		{"unknown activity", true, []structureServiceModuleSnapshot{{TypeID: 35892, Slot: "ServiceSlot0"}}, []CorporationStructureServiceInfo{{Name: "Future Activity", State: "online"}}, fuelEstimateStatusActivityMappingRequired},
		{"duplicate module", true, []structureServiceModuleSnapshot{{TypeID: 35892, Slot: "ServiceSlot0"}, {TypeID: 35892, Slot: "ServiceSlot1"}}, []CorporationStructureServiceInfo{{Name: "Market", State: "online"}}, fuelEstimateStatusAmbiguousModule},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			estimate := EstimateFuelFromModules(structureGroupCitadel, 35833, test.service, test.known, test.modules, activities, rates)
			if estimate.Status != test.status || estimate.FuelPerHour != 0 {
				t.Fatalf("estimate = %#v, want status %q and no value", estimate, test.status)
			}
		})
	}
}

func TestEstimateFuelFromModules_ResolvesMoonDrillsReactionsAndResearchVariants(t *testing.T) {
	rates := defaultModuleFuelRates()
	activities := builtinActivityCandidates()
	tests := []struct {
		name     string
		groupID  int
		typeID   int64
		services []CorporationStructureServiceInfo
		modules  []structureServiceModuleSnapshot
		wantFuel float64
	}{
		{
			name:    "Athanor moon drilling uses standard drill not Metenox",
			groupID: structureGroupRefinery, typeID: structureTypeIDAthanor,
			services: []CorporationStructureServiceInfo{{Name: "Moon Drilling", State: "online"}},
			modules:  []structureServiceModuleSnapshot{{TypeID: 45009, Slot: "ServiceSlot0"}}, wantFuel: 5,
		},
		{
			name:     "Metenox automatic drilling uses Metenox module",
			services: []CorporationStructureServiceInfo{{Name: "Automatic Moon Drilling", State: "online"}},
			modules:  []structureServiceModuleSnapshot{{TypeID: 82941, Slot: "ServiceSlot0"}}, wantFuel: 5,
		},
		{
			name:    "Tatara three reactions and reprocessing",
			groupID: structureGroupRefinery, typeID: 35836,
			services: []CorporationStructureServiceInfo{{Name: "Composite Reactions", State: "online"}, {Name: "Hybrid Reactions", State: "online"}, {Name: "Biochemical Reactions", State: "online"}, {Name: "Reprocessing", State: "online"}},
			modules:  []structureServiceModuleSnapshot{{TypeID: 45537, Slot: "ServiceSlot0"}, {TypeID: 45538, Slot: "ServiceSlot1"}, {TypeID: 45539, Slot: "ServiceSlot2"}, {TypeID: 35899, Slot: "ServiceSlot3"}}, wantFuel: 41.25,
		},
		{
			name:    "Hyasyoda research lab uses its own rate",
			groupID: structureGroupEngineeringComplex, typeID: 35826,
			services: []CorporationStructureServiceInfo{{Name: "Material Efficiency Research", State: "online"}, {Name: "Blueprint Copying", State: "online"}},
			modules:  []structureServiceModuleSnapshot{{TypeID: 45550, Slot: "ServiceSlot0"}}, wantFuel: 7.5,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			est := EstimateFuelFromModules(test.groupID, test.typeID, test.services, true, test.modules, activities, rates)
			if est.Status != fuelEstimateStatusAvailable || est.FuelPerHour != test.wantFuel {
				t.Fatalf("estimate = %#v, want available with %v", est, test.wantFuel)
			}
		})
	}
}

func TestEstimateFuelFromModules_RejectsAmbiguousGenericReaction(t *testing.T) {
	est := EstimateFuelFromModules(
		structureGroupRefinery, 35836,
		[]CorporationStructureServiceInfo{{Name: "Reaction", State: "online"}}, true,
		[]structureServiceModuleSnapshot{{TypeID: 45537, Slot: "ServiceSlot0"}, {TypeID: 45538, Slot: "ServiceSlot1"}},
		builtinActivityCandidates(), defaultModuleFuelRates(),
	)
	if est.Status != fuelEstimateStatusAmbiguousModule || est.FuelPerHour != 0 {
		t.Fatalf("estimate = %#v, want ambiguous with no value", est)
	}
}

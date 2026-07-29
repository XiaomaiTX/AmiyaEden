package service

import "testing"

func TestEstimateFuelFromModules_DeduplicatesActivitiesByPhysicalModule(t *testing.T) {
	rates := defaultModuleFuelRates()
	activities := defaultActivityMappings()
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
	activities := defaultActivityMappings()
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

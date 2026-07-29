package service

import (
	"amiya-eden/internal/model"
	"amiya-eden/internal/repository"
	"encoding/json"
	"math"
	"strings"
)

const (
	fuelEstimateStatusAvailable               = "available"
	fuelEstimateStatusAuthorizationRequired   = "authorization_required"
	fuelEstimateStatusActivityMappingRequired = "activity_mapping_required"
	fuelEstimateStatusModuleMismatch          = "module_mismatch"
	fuelEstimateStatusRateUnavailable         = "rate_unavailable"
	fuelEstimateStatusAmbiguousModule         = "ambiguous_module"
)

type structureServiceModuleSnapshot struct {
	TypeID int    `json:"type_id"`
	Slot   string `json:"slot"`
}

type moduleFuelRate struct {
	FuelPerHour float64
	Category    structureServiceCategory
}

func decodeStructureServiceModules(raw string) []structureServiceModuleSnapshot {
	if strings.TrimSpace(raw) == "" {
		return []structureServiceModuleSnapshot{}
	}
	var modules []structureServiceModuleSnapshot
	if err := json.Unmarshal([]byte(raw), &modules); err != nil {
		return []structureServiceModuleSnapshot{}
	}
	return modules
}

func categoryString(category structureServiceCategory) string {
	switch category {
	case serviceCategoryCitadelOnly:
		return "citadel"
	case serviceCategoryEngineeringComplex:
		return "engineering_complex"
	case serviceCategoryRefinery:
		return "refinery"
	default:
		return "other"
	}
}

func parseCategory(value string) structureServiceCategory {
	switch value {
	case "citadel":
		return serviceCategoryCitadelOnly
	case "engineering_complex":
		return serviceCategoryEngineeringComplex
	case "refinery":
		return serviceCategoryRefinery
	default:
		return serviceCategoryOther
	}
}

func defaultModuleFuelRates() map[int]moduleFuelRate {
	result := make(map[int]moduleFuelRate)
	for _, info := range defaultServiceFuelRates() {
		result[info.TypeID] = moduleFuelRate{FuelPerHour: info.FuelPerHour, Category: info.Category}
	}
	return result
}

func loadModuleFuelRateMap(repo *repository.StructureServiceFuelRateRepository) map[int]moduleFuelRate {
	result := defaultModuleFuelRates()
	if repo == nil {
		return result
	}
	rows, err := repo.ListAll()
	if err != nil {
		return result
	}
	for _, row := range rows {
		if row.TypeID <= 0 {
			continue
		}
		category := parseCategory(row.FuelCategory)
		// Existing rows predate fuel_category and database defaults materialize as
		// "other". Built-in module categories remain authoritative.
		if fallback, ok := result[row.TypeID]; ok {
			category = fallback.Category
		}
		result[row.TypeID] = moduleFuelRate{FuelPerHour: row.FuelPerHour, Category: category}
	}
	return result
}

func defaultActivityMappings() map[string]int {
	groups := map[int][]string{
		35878: {"manufacturing", "industry", "Manufacturing (Standard)"},
		35881: {"capital_shipyard", "Manufacturing (Capitals)"},
		35877: {"supercapital_shipyard", "Manufacturing (Supercapitals)"},
		35891: {"research_lab", "Material Efficiency Research", "Time Efficiency Research", "Blueprint Copying"},
		35886: {"invention_lab", "Invention"},
		35892: {"market", "Market"},
		35894: {"clone_bay", "Clone Bay"},
		35899: {"reprocessing", "Reprocessing"},
		45537: {"reaction", "Reaction"},
		35912: {"cynosural_generator", "Cynosural Generator"},
		35913: {"jump_bridge", "Jump Bridge"},
		35914: {"cynosural_jammer", "Cynosural Jammer"},
		82941: {"moon_drilling", "Moon Drilling", "Automatic Moon Drilling"},
	}
	result := make(map[string]int)
	for typeID, names := range groups {
		for _, name := range names {
			result[normalizeServiceName(name)] = typeID
		}
	}
	return result
}

func loadActivityMapping(repo *repository.StructureServiceActivityRepository) map[string]int {
	result := defaultActivityMappings()
	if repo == nil {
		return result
	}
	rows, err := repo.ListAll()
	if err != nil {
		return result
	}
	for _, row := range rows {
		if name := normalizeServiceName(row.ActivityName); name != "" && row.TypeID > 0 {
			result[name] = row.TypeID
		}
	}
	return result
}

func seedDefaultActivityMappings(repo *repository.StructureServiceActivityRepository) error {
	if repo == nil {
		return nil
	}
	defaults := defaultActivityMappings()
	rows := make([]model.StructureServiceActivity, 0, len(defaults))
	for name, typeID := range defaults {
		rows = append(rows, model.StructureServiceActivity{ActivityName: name, TypeID: typeID})
	}
	return repo.Upsert(rows)
}

func EstimateFuelFromModules(
	structureGroupID int,
	structureTypeID int64,
	services []CorporationStructureServiceInfo,
	modulesKnown bool,
	modules []structureServiceModuleSnapshot,
	activityMap map[string]int,
	rateMap map[int]moduleFuelRate,
) FuelEstimate {
	est := FuelEstimate{Status: fuelEstimateStatusAvailable}
	if !modulesKnown {
		est.Status = fuelEstimateStatusAuthorizationRequired
		return est
	}
	installed := make(map[int]int, len(modules))
	for _, module := range modules {
		if module.TypeID > 0 {
			installed[module.TypeID]++
		}
	}
	active := make(map[int]struct{})
	for _, service := range services {
		if !strings.EqualFold(service.State, "online") {
			continue
		}
		typeID, ok := activityMap[normalizeServiceName(service.Name)]
		if !ok {
			est.UnknownServices = append(est.UnknownServices, service.Name)
			continue
		}
		if installed[typeID] == 0 {
			est.UnknownServices = append(est.UnknownServices, service.Name)
			est.Status = fuelEstimateStatusModuleMismatch
			continue
		}
		if installed[typeID] > 1 {
			est.UnknownServices = append(est.UnknownServices, service.Name)
			est.Status = fuelEstimateStatusAmbiguousModule
			continue
		}
		active[typeID] = struct{}{}
	}
	if len(est.UnknownServices) > 0 {
		if est.Status == fuelEstimateStatusAvailable {
			est.Status = fuelEstimateStatusActivityMappingRequired
		}
		return est
	}
	total := 0.0
	for typeID := range active {
		rate, ok := rateMap[typeID]
		if !ok || rate.FuelPerHour <= 0 {
			est.Status = fuelEstimateStatusRateUnavailable
			return est
		}
		total += rate.FuelPerHour * serviceFuelFactor(structureGroupID, structureTypeID, rate.Category)
	}
	est.FuelPerHour = math.Round(total*100) / 100
	return est
}

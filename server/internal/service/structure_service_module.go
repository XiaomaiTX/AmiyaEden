package service

import (
	"amiya-eden/global"
	"amiya-eden/internal/model"
	"amiya-eden/internal/repository"
	"encoding/json"
	"math"
	"sort"
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

// builtinStructureServiceModules is the single system-owned catalogue. The
// numbers are fallback values from ESI dogma 2109; the scheduled sync refreshes
// them from ESI by TypeID.
func builtinStructureServiceModules() []model.StructureServiceFuelRate {
	return []model.StructureServiceFuelRate{
		{ServiceName: "manufacturing", TypeID: 35878, TypeName: "Standup Manufacturing Plant I", FuelPerHour: 12, FuelCategory: "engineering_complex"},
		{ServiceName: "capital_shipyard", TypeID: 35881, TypeName: "Standup Capital Shipyard I", FuelPerHour: 24, FuelCategory: "engineering_complex"},
		{ServiceName: "supercapital_shipyard", TypeID: 35877, TypeName: "Standup Supercapital Shipyard I", FuelPerHour: 36, FuelCategory: "engineering_complex"},
		{ServiceName: "research_lab", TypeID: 35891, TypeName: "Standup Research Lab I", FuelPerHour: 12, FuelCategory: "engineering_complex"},
		{ServiceName: "hyasyoda_research_lab", TypeID: 45550, TypeName: "Standup Hyasyoda Research Lab", FuelPerHour: 10, FuelCategory: "engineering_complex"},
		{ServiceName: "legacy_time_efficiency_lab", TypeID: 35889, TypeName: "Structure Time Efficiency Laboratory", FuelPerHour: 5, FuelCategory: "engineering_complex"},
		{ServiceName: "legacy_material_efficiency_lab", TypeID: 35890, TypeName: "Structure Material Efficiency Laboratory", FuelPerHour: 10, FuelCategory: "engineering_complex"},
		{ServiceName: "invention_lab", TypeID: 35886, TypeName: "Standup Invention Lab I", FuelPerHour: 12, FuelCategory: "engineering_complex"},
		{ServiceName: "market", TypeID: 35892, TypeName: "Standup Market Hub I", FuelPerHour: 40, FuelCategory: "citadel"},
		{ServiceName: "clone_bay", TypeID: 35894, TypeName: "Standup Cloning Center I", FuelPerHour: 10, FuelCategory: "citadel"},
		{ServiceName: "reprocessing", TypeID: 35899, TypeName: "Standup Reprocessing Facility I", FuelPerHour: 10, FuelCategory: "refinery"},
		{ServiceName: "composite_reactor", TypeID: 45537, TypeName: "Standup Composite Reactor I", FuelPerHour: 15, FuelCategory: "refinery"},
		{ServiceName: "hybrid_reactor", TypeID: 45538, TypeName: "Standup Hybrid Reactor I", FuelPerHour: 15, FuelCategory: "refinery"},
		{ServiceName: "biochemical_reactor", TypeID: 45539, TypeName: "Standup Biochemical Reactor I", FuelPerHour: 15, FuelCategory: "refinery"},
		{ServiceName: "moon_drilling", TypeID: 45009, TypeName: "Standup Moon Drill I", FuelPerHour: 5, FuelCategory: "other"},
		{ServiceName: "automatic_moon_drilling", TypeID: 82941, TypeName: "Standup Metenox Moon Drill", FuelPerHour: 5, FuelCategory: "other"},
		{ServiceName: "cynosural_generator", TypeID: 35912, TypeName: "Standup Cynosural Field Generator I", FuelPerHour: 15, FuelCategory: "other"},
		{ServiceName: "jump_bridge", TypeID: 35913, TypeName: "Standup Conduit Generator I", FuelPerHour: 30, FuelCategory: "other"},
		{ServiceName: "cynosural_jammer", TypeID: 35914, TypeName: "Standup Cynosural System Jammer I", FuelPerHour: 40, FuelCategory: "other"},
	}
}

// builtinActivityCandidates is intentionally many-to-many. The slot snapshot,
// rather than the display label, selects the physical module at estimate time.
func builtinActivityCandidates() map[string][]int {
	research := []int{35891, 45550}
	return map[string][]int{
		"manufacturing":                 {35878},
		"industry":                      {35878},
		"manufacturing_(standard)":      {35878},
		"capital_shipyard":              {35881},
		"manufacturing_(capitals)":      {35881},
		"supercapital_shipyard":         {35877},
		"manufacturing_(supercapitals)": {35877},
		"research_lab":                  research,
		"material_efficiency_research":  {35891, 45550, 35890},
		"time_efficiency_research":      {35891, 45550, 35889},
		"blueprint_copying":             research,
		"invention_lab":                 {35886},
		"invention":                     {35886},
		"market":                        {35892},
		"clone_bay":                     {35894},
		"reprocessing":                  {35899},
		"composite_reactions":           {45537},
		"hybrid_reactions":              {45538},
		"biochemical_reactions":         {45539},
		"reaction":                      {45537, 45538, 45539},
		"moon_drilling":                 {45009},
		"automatic_moon_drilling":       {82941},
		"cynosural_generator":           {35912},
		"jump_bridge":                   {35913},
		"cynosural_jammer":              {35914},
	}
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
	for _, info := range builtinStructureServiceModules() {
		result[info.TypeID] = moduleFuelRate{FuelPerHour: info.FuelPerHour, Category: parseCategory(info.FuelCategory)}
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
		// A stale display-name alias may share a type ID with the canonical
		// module row. Only the canonical row is allowed to override a built-in
		// module's dogma rate.
		if builtin, ok := builtinModuleByTypeID(row.TypeID); ok && normalizeServiceName(row.ServiceName) != builtin.ServiceName {
			continue
		}
		category := parseCategory(row.FuelCategory)
		if fallback, ok := result[row.TypeID]; ok {
			category = fallback.Category
		}
		result[row.TypeID] = moduleFuelRate{FuelPerHour: row.FuelPerHour, Category: category}
	}
	return result
}

func builtinModuleByTypeID(typeID int) (model.StructureServiceFuelRate, bool) {
	for _, module := range builtinStructureServiceModules() {
		if module.TypeID == typeID {
			return module, true
		}
	}
	return model.StructureServiceFuelRate{}, false
}

func loadActivityCandidates(repo *repository.StructureServiceActivityCandidateRepository) map[string][]int {
	result := builtinActivityCandidates()
	if repo == nil {
		return result
	}
	rows, err := repo.ListAll()
	if err != nil {
		return result
	}
	custom := make(map[string][]int)
	for _, row := range rows {
		name := normalizeServiceName(row.ActivityName)
		if name == "" || row.TypeID <= 0 {
			continue
		}
		if _, managed := result[name]; managed {
			continue
		}
		custom[name] = append(custom[name], row.TypeID)
	}
	for name, typeIDs := range custom {
		result[name] = uniqueSortedTypeIDs(typeIDs)
	}
	return result
}

func uniqueSortedTypeIDs(values []int) []int {
	set := make(map[int]struct{}, len(values))
	for _, value := range values {
		if value > 0 {
			set[value] = struct{}{}
		}
	}
	result := make([]int, 0, len(set))
	for value := range set {
		result = append(result, value)
	}
	sort.Ints(result)
	return result
}

// ReconcileStructureServiceCatalog upgrades legacy persisted rows and seeds
// canonical mappings. It is idempotent and deliberately does not overwrite
// administrator mappings for unknown activity labels.
func ReconcileStructureServiceCatalog() error {
	if global.DB == nil {
		return nil
	}
	fuelRepo := repository.NewStructureServiceFuelRateRepository()
	if err := ensureBuiltinServiceModules(fuelRepo); err != nil {
		return err
	}
	legacyRepo := repository.NewStructureServiceActivityRepository()
	legacy, err := legacyRepo.ListAll()
	if err != nil {
		return err
	}
	candidateRepo := repository.NewStructureServiceActivityCandidateRepository()
	managed := builtinActivityCandidates()
	candidateCount, err := candidateRepo.Count()
	if err != nil {
		return err
	}
	// The legacy table is a one-time migration source. Re-importing it on every
	// startup would resurrect mappings an administrator has intentionally
	// replaced in the new many-to-many catalogue.
	if candidateCount == 0 {
		legacyRows := make([]model.StructureServiceActivityCandidate, 0, len(legacy))
		for _, row := range legacy {
			name := normalizeServiceName(row.ActivityName)
			if _, isManaged := managed[name]; !isManaged && name != "" && row.TypeID > 0 {
				legacyRows = append(legacyRows, model.StructureServiceActivityCandidate{ActivityName: name, TypeID: row.TypeID})
			}
		}
		if err := candidateRepo.Upsert(legacyRows); err != nil {
			return err
		}
	}
	systemRows := make([]model.StructureServiceActivityCandidate, 0, 32)
	for activityName, typeIDs := range managed {
		for _, typeID := range typeIDs {
			systemRows = append(systemRows, model.StructureServiceActivityCandidate{ActivityName: activityName, TypeID: typeID, SystemManaged: true})
		}
	}
	return candidateRepo.ReplaceSystem(systemRows)
}

// ensureBuiltinServiceModules adds missing modules and corrects rows whose
// stable key previously pointed to the wrong physical type. It intentionally
// preserves a matching row's ESI-synchronised dogma value rather than resetting
// it to a fallback on every process start.
func ensureBuiltinServiceModules(repo *repository.StructureServiceFuelRateRepository) error {
	existing, err := repo.ListAll()
	if err != nil {
		return err
	}
	byName := make(map[string]model.StructureServiceFuelRate, len(existing))
	for _, row := range existing {
		byName[row.ServiceName] = row
	}
	toWrite := make([]model.StructureServiceFuelRate, 0)
	for _, module := range builtinStructureServiceModules() {
		row, ok := byName[module.ServiceName]
		if !ok || row.TypeID != module.TypeID {
			toWrite = append(toWrite, module)
		}
	}
	return repo.UpsertBatch(toWrite)
}

func EstimateFuelFromModules(
	structureGroupID int,
	structureTypeID int64,
	services []CorporationStructureServiceInfo,
	modulesKnown bool,
	modules []structureServiceModuleSnapshot,
	activityCandidates map[string][]int,
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
		candidates, ok := activityCandidates[normalizeServiceName(service.Name)]
		if !ok || len(candidates) == 0 {
			est.UnknownServices = append(est.UnknownServices, service.Name)
			continue
		}
		matched := make([]int, 0, len(candidates))
		for _, typeID := range candidates {
			if installed[typeID] == 1 {
				matched = append(matched, typeID)
				continue
			}
			if installed[typeID] > 1 {
				est.UnknownServices = append(est.UnknownServices, service.Name)
				est.Status = fuelEstimateStatusAmbiguousModule
			}
		}
		if len(matched) == 0 {
			if est.Status == fuelEstimateStatusAvailable {
				est.Status = fuelEstimateStatusModuleMismatch
			}
			est.UnknownServices = append(est.UnknownServices, service.Name)
			continue
		}
		if len(matched) > 1 {
			est.UnknownServices = append(est.UnknownServices, service.Name)
			est.Status = fuelEstimateStatusAmbiguousModule
			continue
		}
		active[matched[0]] = struct{}{}
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

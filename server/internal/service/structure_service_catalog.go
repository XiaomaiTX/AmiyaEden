package service

import (
	"amiya-eden/internal/model"
	"context"
	"fmt"
	"sort"
	"strings"
)

type StructureServiceActivityCatalogItem struct {
	ActivityName  string `json:"activity_name"`
	TypeIDs       []int  `json:"type_ids"`
	SystemManaged bool   `json:"system_managed"`
}

type StructureServicePendingActivity struct {
	ActivityName           string `json:"activity_name"`
	StructureID            int64  `json:"structure_id"`
	StructureName          string `json:"structure_name"`
	InstalledModuleTypeIDs []int  `json:"installed_module_type_ids"`
}

type StructureServiceCatalog struct {
	Modules            []model.StructureServiceFuelRate      `json:"modules"`
	Activities         []StructureServiceActivityCatalogItem `json:"activities"`
	UnmappedActivities []StructureServicePendingActivity     `json:"unmapped_activities"`
}

type StructureServiceCatalogUpdate struct {
	Modules    []StructureServiceModuleInput   `json:"modules"`
	Activities []StructureServiceActivityInput `json:"activities"`
}

type StructureServiceModuleInput struct {
	ServiceName  string `json:"service_name"`
	TypeID       int    `json:"type_id"`
	FuelCategory string `json:"fuel_category"`
}

type StructureServiceActivityInput struct {
	ActivityName string `json:"activity_name"`
	TypeIDs      []int  `json:"type_ids"`
}

func (s *CorporationStructureService) GetFuelServiceCatalog(ctx context.Context) (*StructureServiceCatalog, error) {
	persistedModules, err := s.fuelRateRepo.ListAll()
	if err != nil {
		return nil, fmt.Errorf("load service module catalog: %w", err)
	}
	modules := effectiveServiceModuleCatalog(persistedModules)
	known := loadActivityCandidates(s.activityCandidateRepo)
	activities := effectiveActivityCatalog(known)
	structures, err := s.repo.ListAllCorpStructures()
	if err != nil {
		return nil, fmt.Errorf("load structures for activity catalog: %w", err)
	}
	pending := make([]StructureServicePendingActivity, 0)
	seen := make(map[string]struct{})
	for _, structure := range structures {
		installed := make([]int, 0)
		for _, module := range decodeStructureServiceModules(structure.ServiceModules) {
			installed = append(installed, module.TypeID)
		}
		installed = uniqueSortedTypeIDs(installed)
		for _, activity := range convertStructureServices(structure.Services) {
			if !strings.EqualFold(activity.State, "online") || strings.TrimSpace(activity.Name) == "" {
				continue
			}
			if _, ok := known[normalizeServiceName(activity.Name)]; ok {
				continue
			}
			key := fmt.Sprintf("%d:%s", structure.StructureID, normalizeServiceName(activity.Name))
			if _, exists := seen[key]; exists {
				continue
			}
			seen[key] = struct{}{}
			pending = append(pending, StructureServicePendingActivity{
				ActivityName: activity.Name, StructureID: structure.StructureID, StructureName: structure.Name,
				InstalledModuleTypeIDs: installed,
			})
		}
	}
	sort.Slice(pending, func(i, j int) bool {
		if pending[i].ActivityName == pending[j].ActivityName {
			return pending[i].StructureID < pending[j].StructureID
		}
		return strings.ToLower(pending[i].ActivityName) < strings.ToLower(pending[j].ActivityName)
	})
	return &StructureServiceCatalog{Modules: modules, Activities: activities, UnmappedActivities: pending}, nil
}

func effectiveActivityCatalog(candidates map[string][]int) []StructureServiceActivityCatalogItem {
	managed := builtinActivityCandidates()
	result := make([]StructureServiceActivityCatalogItem, 0, len(candidates))
	for name, typeIDs := range candidates {
		_, systemManaged := managed[name]
		result = append(result, StructureServiceActivityCatalogItem{
			ActivityName: name, TypeIDs: uniqueSortedTypeIDs(typeIDs), SystemManaged: systemManaged,
		})
	}
	sort.Slice(result, func(i, j int) bool { return result[i].ActivityName < result[j].ActivityName })
	return result
}

// effectiveServiceModuleCatalog keeps the verified system catalogue available
// before scheduled ESI dogma sync has created its first persisted rows.
func effectiveServiceModuleCatalog(persisted []model.StructureServiceFuelRate) []model.StructureServiceFuelRate {
	byTypeID := make(map[int]model.StructureServiceFuelRate)
	for _, module := range builtinStructureServiceModules() {
		byTypeID[module.TypeID] = module
	}
	for _, row := range persisted {
		if row.TypeID > 0 {
			if builtin, ok := builtinModuleByTypeID(row.TypeID); ok && normalizeServiceName(row.ServiceName) != builtin.ServiceName {
				continue
			}
			byTypeID[row.TypeID] = row
		}
	}
	modules := make([]model.StructureServiceFuelRate, 0, len(byTypeID))
	for _, row := range byTypeID {
		modules = append(modules, row)
	}
	sort.Slice(modules, func(i, j int) bool {
		return strings.ToLower(modules[i].TypeName) < strings.ToLower(modules[j].TypeName)
	})
	return modules
}

func (s *CorporationStructureService) UpdateFuelServiceCatalog(ctx context.Context, req StructureServiceCatalogUpdate) error {
	if len(req.Modules) == 0 && len(req.Activities) == 0 {
		return nil
	}
	rates := NewStructureFuelRateService()
	moduleRows := make([]model.StructureServiceFuelRate, 0, len(req.Modules))
	knownTypes := make(map[int]struct{})
	for _, module := range effectiveServiceModuleCatalog(nil) {
		knownTypes[module.TypeID] = struct{}{}
	}
	for _, row := range req.Modules {
		key := normalizeServiceName(row.ServiceName)
		if key == "" || row.TypeID <= 0 || !validFuelCategory(row.FuelCategory) {
			return fmt.Errorf("invalid service module")
		}
		var response esiTypeResponse
		if err := rates.esiClient.Get(ctx, fmt.Sprintf("/universe/types/%d/", row.TypeID), "", &response); err != nil {
			return fmt.Errorf("validate service module %d: %w", row.TypeID, err)
		}
		fuel := readDogmaFuelAmount(response.DogmaAttributes)
		if fuel <= 0 {
			return fmt.Errorf("type %d is not a fuel-consuming service module", row.TypeID)
		}
		name := response.Name
		if name == "" {
			name = fmt.Sprintf("Type-%d", row.TypeID)
		}
		moduleRows = append(moduleRows, model.StructureServiceFuelRate{ServiceName: key, TypeID: row.TypeID, TypeName: name, FuelPerHour: fuel, FuelCategory: row.FuelCategory})
		knownTypes[row.TypeID] = struct{}{}
	}
	if err := s.fuelRateRepo.UpsertBatch(moduleRows); err != nil {
		return fmt.Errorf("save service modules: %w", err)
	}
	managed := builtinActivityCandidates()
	for _, row := range req.Activities {
		name := normalizeServiceName(row.ActivityName)
		typeIDs := uniqueSortedTypeIDs(row.TypeIDs)
		if name == "" || len(typeIDs) == 0 {
			return fmt.Errorf("invalid service activity")
		}
		if _, isManaged := managed[name]; isManaged {
			return fmt.Errorf("system-managed activity %q cannot be changed", row.ActivityName)
		}
		for _, typeID := range typeIDs {
			if _, ok := knownTypes[typeID]; !ok {
				return fmt.Errorf("activity references an unregistered module type %d", typeID)
			}
		}
		if err := s.activityCandidateRepo.ReplaceCustom(name, typeIDs); err != nil {
			return fmt.Errorf("save service activity: %w", err)
		}
	}
	return nil
}

func validFuelCategory(value string) bool {
	switch value {
	case "other", "citadel", "engineering_complex", "refinery":
		return true
	default:
		return false
	}
}

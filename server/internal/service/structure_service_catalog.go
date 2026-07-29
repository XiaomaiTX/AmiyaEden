package service

import (
	"amiya-eden/internal/model"
	"context"
	"fmt"
	"sort"
	"strings"
)

type StructureServiceCatalog struct {
	Modules            []model.StructureServiceFuelRate `json:"modules"`
	Activities         []model.StructureServiceActivity `json:"activities"`
	UnmappedActivities []string                         `json:"unmapped_activities"`
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
	TypeID       int    `json:"type_id"`
}

func (s *CorporationStructureService) GetFuelServiceCatalog(ctx context.Context) (*StructureServiceCatalog, error) {
	modules, err := s.fuelRateRepo.ListAll()
	if err != nil {
		return nil, fmt.Errorf("load service module catalog: %w", err)
	}
	activities, err := s.activityRepo.ListAll()
	if err != nil {
		return nil, fmt.Errorf("load service activity catalog: %w", err)
	}
	known := loadActivityMapping(s.activityRepo)
	structures, err := s.repo.ListCorpStructures(nil)
	if err != nil {
		return nil, fmt.Errorf("load structures for activity catalog: %w", err)
	}
	unknownSet := make(map[string]struct{})
	for _, structure := range structures {
		for _, activity := range convertStructureServices(structure.Services) {
			if !strings.EqualFold(activity.State, "online") {
				continue
			}
			if _, ok := known[normalizeServiceName(activity.Name)]; !ok && strings.TrimSpace(activity.Name) != "" {
				unknownSet[activity.Name] = struct{}{}
			}
		}
	}
	unknown := make([]string, 0, len(unknownSet))
	for name := range unknownSet {
		unknown = append(unknown, name)
	}
	sort.Slice(unknown, func(i, j int) bool { return strings.ToLower(unknown[i]) < strings.ToLower(unknown[j]) })
	return &StructureServiceCatalog{Modules: modules, Activities: activities, UnmappedActivities: unknown}, nil
}

func (s *CorporationStructureService) UpdateFuelServiceCatalog(ctx context.Context, req StructureServiceCatalogUpdate) error {
	if len(req.Modules) == 0 && len(req.Activities) == 0 {
		return nil
	}
	rates := NewStructureFuelRateService()
	moduleRows := make([]model.StructureServiceFuelRate, 0, len(req.Modules))
	knownTypes := make(map[int]struct{})
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
	for typeID := range defaultModuleFuelRates() {
		knownTypes[typeID] = struct{}{}
	}
	activityRows := make([]model.StructureServiceActivity, 0, len(req.Activities))
	for _, row := range req.Activities {
		name := normalizeServiceName(row.ActivityName)
		if name == "" || row.TypeID <= 0 {
			return fmt.Errorf("invalid service activity")
		}
		if _, ok := knownTypes[row.TypeID]; !ok {
			return fmt.Errorf("activity references an unregistered module type %d", row.TypeID)
		}
		activityRows = append(activityRows, model.StructureServiceActivity{ActivityName: name, TypeID: row.TypeID})
	}
	if err := s.activityRepo.Upsert(activityRows); err != nil {
		return fmt.Errorf("save service activities: %w", err)
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

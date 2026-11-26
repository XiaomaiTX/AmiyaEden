package handler

import (
	"amiya-eden/global"
	"amiya-eden/internal/apicontract"
	"amiya-eden/internal/model"
	"amiya-eden/internal/service"
	"amiya-eden/jobs"
	"amiya-eden/pkg/response"
	"context"
	"errors"
	"fmt"
)

type CorporationStructureHandler struct {
	svc *service.CorporationStructureService
}

func NewCorporationStructureHandler() *CorporationStructureHandler {
	return &CorporationStructureHandler{
		svc: service.NewCorporationStructureService(),
	}
}

func (h *CorporationStructureHandler) runCorpStructureTask(ctx context.Context, req apicontract.CorpStructureTaskInput) (apicontract.CorpStructureTask, error) {
	characterID, err := h.svc.ResolveRefreshAuthorizationCharacter(ctx, req.CorporationID)
	if err != nil {
		return apicontract.CorpStructureTask{}, err
	}

	queue := jobs.GetESIQueue()
	if queue == nil {
		return apicontract.CorpStructureTask{}, errors.New("刷新队列未初始化")
	}

	taskName := fmt.Sprintf("dashboard_run_corporation_structures_%d", req.CorporationID)
	if ok := global.EnsureBackgroundTaskManager().Go(taskName, func(ctx context.Context) error {
		return queue.RunTask(ctx, "corporation_structures", characterID)
	}); !ok {
		return apicontract.CorpStructureTask{}, errors.New("服务正在关闭，任务未启动")
	}

	return apicontract.CorpStructureTask{
		CorporationID: req.CorporationID,
		Queued:        true,
		Running:       false,
		Message:       "已触发后台 ESI 刷新任务",
	}, nil
}

func (h *CorporationStructureHandler) runCorpStructureFuelSalaryPayout(ctx context.Context, actor uint, req apicontract.CorpStructureFuelPayoutInput) (apicontract.CorpStructureFuelPayout, error) {
	result, err := h.svc.RunFuelSalaryPayout(ctx, service.FuelSalaryPayoutRunRequest{SettlementMonth: req.SettlementMonth, OperatorUserID: actor})
	if err != nil {
		return apicontract.CorpStructureFuelPayout{}, err
	}
	return corpStructureFuelPayoutDTO(*result), nil
}

func corpStructureFilterInput(corporationID *int64) service.CorporationStructureFilterOptionsRequest {
	if corporationID == nil {
		return service.CorporationStructureFilterOptionsRequest{}
	}
	return service.CorporationStructureFilterOptionsRequest{CorporationID: *corporationID}
}

func corpStructureServiceDTO(value service.CorporationStructureServiceInfo) apicontract.CorpStructureService {
	return apicontract.CorpStructureService{
		Name:  value.Name,
		State: value.State,
	}
}

func corpStructureRowDTO(value service.CorporationStructureRow) apicontract.CorpStructureRow {
	return apicontract.CorpStructureRow{
		CorporationID:          value.CorporationID,
		CorporationName:        value.CorporationName,
		AssignedUserID:         value.AssignedUserID,
		AssignedCharacterID:    value.AssignedCharacterID,
		AssignedCharacterName:  value.AssignedCharacterName,
		StructureID:            value.StructureID,
		Name:                   value.Name,
		TypeID:                 value.TypeID,
		TypeName:               value.TypeName,
		SystemID:               value.SystemID,
		SystemName:             value.SystemName,
		RegionID:               value.RegionID,
		RegionName:             value.RegionName,
		Security:               value.Security,
		State:                  value.State,
		Services:               mapContractRows(value.Services, corpStructureServiceDTO),
		FuelExpires:            value.FuelExpires,
		FuelRemaining:          value.FuelRemaining,
		FuelRemainingHours:     value.FuelRemainingHours,
		FuelPerHour:            value.FuelPerHour,
		FuelToMonthEnd:         value.FuelToMonthEnd,
		FuelEstimateIncomplete: value.FuelEstimateIncomplete,
		FuelUnknownServices:    &value.FuelUnknownServices,
		FuelEstimateStatus:     value.FuelEstimateStatus,
		ReinforceHour:          value.ReinforceHour,
		StateTimerStart:        value.StateTimerStart,
		StateTimerEnd:          value.StateTimerEnd,
		UpdatedAt:              value.UpdatedAt,
	}
}

func corpStructureListDTO(value service.CorporationStructureListResponse) apicontract.CorpStructureList {
	return apicontract.CorpStructureList{
		Items:    mapContractRows(value.Items, corpStructureRowDTO),
		Total:    value.Total,
		Page:     value.Page,
		PageSize: value.PageSize,
	}
}

func corpStructureDirectorDTO(value service.DirectorCharacterOption) apicontract.CorpStructureDirector {
	return apicontract.CorpStructureDirector{
		UserID:        value.UserID,
		CharacterID:   value.CharacterID,
		CharacterName: value.CharacterName,
	}
}

func corpStructureCorporationDTO(value service.ManageCorporationOption) apicontract.CorpStructureCorporation {
	return apicontract.CorpStructureCorporation{
		CorporationID:         value.CorporationID,
		CorporationName:       value.CorporationName,
		AuthorizedCharacterID: value.AuthorizedCharacterID,
		DirectorCharacters:    mapContractRows(value.DirectorCharacters, corpStructureDirectorDTO),
	}
}

func corpStructureSettingsDTO(value service.CorporationStructuresSettingsResponse) apicontract.CorpStructureSettings {
	return apicontract.CorpStructureSettings{
		Corporations:             mapContractRows(value.Corporations, corpStructureCorporationDTO),
		FuelNoticeThresholdDays:  value.FuelNoticeThresholdDays,
		TimerNoticeThresholdDays: value.TimerNoticeThresholdDays,
	}
}

func corpStructureAssignmentDTO(value service.CorporationStructureAssignmentItem) apicontract.CorpStructureAssignment {
	return apicontract.CorpStructureAssignment{
		CorporationID:         value.CorporationID,
		CorporationName:       value.CorporationName,
		StructureID:           value.StructureID,
		StructureName:         value.StructureName,
		SystemID:              value.SystemID,
		SystemName:            value.SystemName,
		RegionID:              value.RegionID,
		RegionName:            value.RegionName,
		Security:              value.Security,
		TypeID:                value.TypeID,
		TypeName:              value.TypeName,
		AssignedUserID:        value.AssignedUserID,
		AssignedCharacterID:   value.AssignedCharacterID,
		AssignedCharacterName: value.AssignedCharacterName,
	}
}

func corpStructureAssignmentsDTO(value service.CorporationStructureAssignmentListResponse) apicontract.CorpStructureAssignments {
	return apicontract.CorpStructureAssignments{
		Items:        mapContractRows(value.Items, corpStructureAssignmentDTO),
		FuelOfficers: mapContractRows(value.FuelOfficers, corpStructureFuelOfficerDTO),
	}
}

func corpStructureFuelSalaryDTO(value service.FuelSalarySettingsResponse) apicontract.CorpStructureFuelSalarySettings {
	return apicontract.CorpStructureFuelSalarySettings{
		SalaryPerStructureMonthly: value.SalaryPerStructureMonthly,
	}
}

func corpStructureFuelPayoutItemDTO(value service.FuelSalaryPayoutRunItem) apicontract.CorpStructureFuelPayoutItem {
	return apicontract.CorpStructureFuelPayoutItem{
		UserID:        value.UserID,
		AssignedCount: value.AssignedCount,
		UnitSalary:    value.UnitSalary,
		Amount:        value.Amount,
		WalletRefID:   value.WalletRefID,
	}
}

func corpStructureFuelPayoutDTO(value service.FuelSalaryPayoutRunResponse) apicontract.CorpStructureFuelPayout {
	return apicontract.CorpStructureFuelPayout{
		SettlementMonth: value.SettlementMonth,
		Items:           mapContractRows(value.Items, corpStructureFuelPayoutItemDTO),
	}
}

func corpStructureSystemDTO(value service.CorporationStructureSystemOption) apicontract.CorpStructureSystem {
	return apicontract.CorpStructureSystem{
		SystemID:   value.SystemID,
		SystemName: value.SystemName,
		RegionID:   value.RegionID,
		RegionName: value.RegionName,
		Security:   value.Security,
	}
}

func corpStructureTypeDTO(value service.CorporationStructureTypeOption) apicontract.CorpStructureType {
	return apicontract.CorpStructureType{
		TypeID:   value.TypeID,
		TypeName: value.TypeName,
	}
}

func corpStructureRegionDTO(value service.CorporationStructureRegionOption) apicontract.CorpStructureRegion {
	return apicontract.CorpStructureRegion{
		RegionID:   value.RegionID,
		RegionName: value.RegionName,
	}
}

func corpStructureServiceNameDTO(value service.CorporationStructureServiceOption) apicontract.CorpStructureServiceName {
	return apicontract.CorpStructureServiceName{
		Name: value.Name,
	}
}

func corpStructureFiltersDTO(value service.CorporationStructureFilterOptionsResponse) apicontract.CorpStructureFilters {
	return apicontract.CorpStructureFilters{
		Systems:  mapContractRows(value.Systems, corpStructureSystemDTO),
		Regions:  mapContractRows(value.Regions, corpStructureRegionDTO),
		Types:    mapContractRows(value.Types, corpStructureTypeDTO),
		Services: mapContractRows(value.Services, corpStructureServiceNameDTO),
	}
}

func corpStructureFuelOfficerDTO(value model.FuelOfficerUserOption) apicontract.CorpStructureFuelOfficer {
	return apicontract.CorpStructureFuelOfficer{
		UserID:        value.UserID,
		DisplayName:   value.DisplayName,
		CharacterID:   value.CharacterID,
		CharacterName: value.CharacterName,
	}
}

func corpStructureActivityDTO(value service.StructureServiceActivityCatalogItem) apicontract.CorpStructureActivity {
	return apicontract.CorpStructureActivity{
		ActivityName:  value.ActivityName,
		TypeIDs:       value.TypeIDs,
		SystemManaged: value.SystemManaged,
	}
}

func corpStructurePendingActivityDTO(value service.StructureServicePendingActivity) apicontract.CorpStructurePendingActivity {
	return apicontract.CorpStructurePendingActivity{
		ActivityName:           value.ActivityName,
		StructureID:            value.StructureID,
		StructureName:          value.StructureName,
		InstalledModuleTypeIDs: value.InstalledModuleTypeIDs,
	}
}

func corpStructureCatalogDTO(value service.StructureServiceCatalog) apicontract.CorpStructureCatalog {
	return apicontract.CorpStructureCatalog{
		Modules:            mapContractRows(value.Modules, corpStructureCatalogModuleDTO),
		Activities:         mapContractRows(value.Activities, corpStructureActivityDTO),
		UnmappedActivities: mapContractRows(value.UnmappedActivities, corpStructurePendingActivityDTO),
	}
}

func corpStructureCatalogModuleDTO(value model.StructureServiceFuelRate) apicontract.CorpStructureCatalogModule {
	return apicontract.CorpStructureCatalogModule{
		ID:           value.ID,
		CreatedAt:    value.CreatedAt,
		UpdatedAt:    value.UpdatedAt,
		ServiceName:  value.ServiceName,
		TypeID:       value.TypeID,
		TypeName:     value.TypeName,
		FuelPerHour:  value.FuelPerHour,
		FuelCategory: value.FuelCategory,
	}
}

func corpStructureListInput(value apicontract.CorpStructureListInput) service.CorporationStructureListRequest {
	return service.CorporationStructureListRequest{
		CorporationID:    value.CorporationID,
		Page:             value.Page,
		PageSize:         value.PageSize,
		Keyword:          value.Keyword,
		StateGroups:      value.StateGroups,
		FuelBucket:       value.FuelBucket,
		FuelMinHours:     value.FuelMinHours,
		FuelMaxHours:     value.FuelMaxHours,
		SystemIDs:        value.SystemIDs,
		RegionIDs:        value.RegionIDs,
		SecurityBands:    value.SecurityBands,
		SecurityMin:      value.SecurityMin,
		SecurityMax:      value.SecurityMax,
		TypeIDs:          value.TypeIDs,
		ServiceNames:     value.ServiceNames,
		ServiceMatchMode: value.ServiceMatchMode,
		TimerBucket:      value.TimerBucket,
		TimerStart:       value.TimerStart,
		TimerEnd:         value.TimerEnd,
		SortBy:           value.SortBy,
		SortOrder:        value.SortOrder,
	}
}

func corpStructureAuthorizationBindingInput(value apicontract.CorpStructureAuthorization) service.CorporationStructureAuthorizationBinding {
	return service.CorporationStructureAuthorizationBinding{
		CorporationID: value.CorporationID,
		CharacterID:   value.CharacterID,
	}
}

func corpStructureAuthorizationInput(value apicontract.CorpStructureAuthorizationInput, operatorID uint) service.CorporationStructureAuthorizationUpdate {
	return service.CorporationStructureAuthorizationUpdate{
		Authorizations:           mapContractRows(value.Authorizations, corpStructureAuthorizationBindingInput),
		FuelNoticeThresholdDays:  value.FuelNoticeThresholdDays,
		TimerNoticeThresholdDays: value.TimerNoticeThresholdDays,
		OperatorUserID:           operatorID,
	}
}

func corpStructureAssignmentBindingInput(value apicontract.CorpStructureAssignmentBinding) service.CorporationStructureAssignmentBinding {
	return service.CorporationStructureAssignmentBinding{
		CorporationID: value.CorporationID,
		StructureID:   value.StructureID,
		UserID:        value.UserID,
	}
}

func corpStructureAssignmentInput(value apicontract.CorpStructureAssignmentInput, operatorID uint) service.CorporationStructureAssignmentUpdateRequest {
	return service.CorporationStructureAssignmentUpdateRequest{
		Assignments:    mapContractRows(value.Assignments, corpStructureAssignmentBindingInput),
		OperatorUserID: operatorID,
	}
}

func corpStructureCatalogInput(value apicontract.CorpStructureCatalogInput) service.StructureServiceCatalogUpdate {
	return service.StructureServiceCatalogUpdate{
		Modules:    mapContractRows(value.Modules, corpStructureCatalogModuleInput),
		Activities: mapContractRows(value.Activities, corpStructureCatalogActivityInput),
	}
}

func corpStructureCatalogModuleInput(value apicontract.CorpStructureCatalogModuleInput) service.StructureServiceModuleInput {
	return service.StructureServiceModuleInput{
		ServiceName:  value.ServiceName,
		TypeID:       value.TypeID,
		FuelCategory: value.FuelCategory,
	}
}

func corpStructureCatalogActivityInput(value apicontract.CorpStructureCatalogActivityInput) service.StructureServiceActivityInput {
	return service.StructureServiceActivityInput{
		ActivityName: value.ActivityName,
		TypeIDs:      value.TypeIDs,
	}
}

func corpStructureUnauthorized() apicontract.UnauthorizedJSONResponse {
	return apicontract.UnauthorizedJSONResponse{Code: response.CodeUnauthorized, Msg: "未登录"}
}

func corpStructureBusinessError(err error) apicontract.BusinessError {
	return contractBusinessError(response.CodeBizError, err.Error())
}

func (h *CorporationStructureHandler) GetCorpStructureSettings(ctx context.Context, _ apicontract.GetCorpStructureSettingsRequestObject) (apicontract.GetCorpStructureSettingsResponseObject, error) {
	requestCtx, _ := strictGinRequestDetails(ctx)
	out := &apicontract.GetCorpStructureSettings200JSONResponse{}
	result, err := h.svc.GetSettings(requestCtx)
	if err != nil {
		return out, out.FromBusinessError(corpStructureBusinessError(err))
	}
	return out, out.FromCorpStructureSettingsSuccess(apicontract.CorpStructureSettingsSuccess{Code: response.CodeOK, Msg: "success", Data: corpStructureSettingsDTO(*result)})
}

func (h *CorporationStructureHandler) UpdateCorpStructureAuthorizations(ctx context.Context, request apicontract.UpdateCorpStructureAuthorizationsRequestObject) (apicontract.UpdateCorpStructureAuthorizationsResponseObject, error) {
	requestCtx, actor := strictGinRequestDetails(ctx)
	if actor == 0 {
		return apicontract.UpdateCorpStructureAuthorizations401JSONResponse{UnauthorizedJSONResponse: corpStructureUnauthorized()}, nil
	}
	out := &apicontract.UpdateCorpStructureAuthorizations200JSONResponse{}
	if err := h.svc.UpdateAuthorizations(requestCtx, corpStructureAuthorizationInput(*request.Body, actor)); err != nil {
		return out, out.FromBusinessError(corpStructureBusinessError(err))
	}
	return out, out.FromNullSuccess(apicontract.NullSuccess{Code: response.CodeOK, Msg: "success"})
}

func (h *CorporationStructureHandler) GetCorpStructureServiceCatalog(ctx context.Context, _ apicontract.GetCorpStructureServiceCatalogRequestObject) (apicontract.GetCorpStructureServiceCatalogResponseObject, error) {
	requestCtx, _ := strictGinRequestDetails(ctx)
	out := &apicontract.GetCorpStructureServiceCatalog200JSONResponse{}
	result, err := h.svc.GetFuelServiceCatalog(requestCtx)
	if err != nil {
		return out, out.FromBusinessError(corpStructureBusinessError(err))
	}
	return out, out.FromCorpStructureCatalogSuccess(apicontract.CorpStructureCatalogSuccess{Code: response.CodeOK, Msg: "success", Data: corpStructureCatalogDTO(*result)})
}

func (h *CorporationStructureHandler) UpdateCorpStructureServiceCatalog(ctx context.Context, request apicontract.UpdateCorpStructureServiceCatalogRequestObject) (apicontract.UpdateCorpStructureServiceCatalogResponseObject, error) {
	requestCtx, _ := strictGinRequestDetails(ctx)
	out := &apicontract.UpdateCorpStructureServiceCatalog200JSONResponse{}
	if err := h.svc.UpdateFuelServiceCatalog(requestCtx, corpStructureCatalogInput(*request.Body)); err != nil {
		return out, out.FromBusinessError(corpStructureBusinessError(err))
	}
	return out, out.FromNullSuccess(apicontract.NullSuccess{Code: response.CodeOK, Msg: "success"})
}

func (h *CorporationStructureHandler) GetCorpStructureFilterOptions(ctx context.Context, request apicontract.GetCorpStructureFilterOptionsRequestObject) (apicontract.GetCorpStructureFilterOptionsResponseObject, error) {
	requestCtx, _ := strictGinRequestDetails(ctx)
	out := &apicontract.GetCorpStructureFilterOptions200JSONResponse{}
	result, err := h.svc.GetFilterOptions(requestCtx, corpStructureFilterInput(request.Params.CorporationId))
	if err != nil {
		return out, out.FromBusinessError(corpStructureBusinessError(err))
	}
	return out, out.FromCorpStructureFiltersSuccess(apicontract.CorpStructureFiltersSuccess{Code: response.CodeOK, Msg: "success", Data: corpStructureFiltersDTO(*result)})
}

func (h *CorporationStructureHandler) ListCorpStructures(ctx context.Context, request apicontract.ListCorpStructuresRequestObject) (apicontract.ListCorpStructuresResponseObject, error) {
	requestCtx, _ := strictGinRequestDetails(ctx)
	out := &apicontract.ListCorpStructures200JSONResponse{}
	result, err := h.svc.ListStructures(requestCtx, corpStructureListInput(*request.Body))
	if err != nil {
		return out, out.FromBusinessError(corpStructureBusinessError(err))
	}
	return out, out.FromCorpStructureListSuccess(apicontract.CorpStructureListSuccess{Code: response.CodeOK, Msg: "success", Data: corpStructureListDTO(*result)})
}

func (h *CorporationStructureHandler) RunCorpStructureTask(ctx context.Context, request apicontract.RunCorpStructureTaskRequestObject) (apicontract.RunCorpStructureTaskResponseObject, error) {
	requestCtx, _ := strictGinRequestDetails(ctx)
	out := &apicontract.RunCorpStructureTask200JSONResponse{}
	data, err := h.runCorpStructureTask(requestCtx, *request.Body)
	if err != nil {
		return out, out.FromBusinessError(corpStructureBusinessError(err))
	}
	return out, out.FromCorpStructureTaskSuccess(apicontract.CorpStructureTaskSuccess{Code: response.CodeOK, Msg: "success", Data: data})
}

func (h *CorporationStructureHandler) GetCorpStructureAssignments(ctx context.Context, request apicontract.GetCorpStructureAssignmentsRequestObject) (apicontract.GetCorpStructureAssignmentsResponseObject, error) {
	requestCtx, _ := strictGinRequestDetails(ctx)
	out := &apicontract.GetCorpStructureAssignments200JSONResponse{}
	result, err := h.svc.GetAssignments(requestCtx, corpStructureFilterInput(request.Params.CorporationId))
	if err != nil {
		return out, out.FromBusinessError(corpStructureBusinessError(err))
	}
	return out, out.FromCorpStructureAssignmentsSuccess(apicontract.CorpStructureAssignmentsSuccess{Code: response.CodeOK, Msg: "success", Data: corpStructureAssignmentsDTO(*result)})
}

func (h *CorporationStructureHandler) UpdateCorpStructureAssignments(ctx context.Context, request apicontract.UpdateCorpStructureAssignmentsRequestObject) (apicontract.UpdateCorpStructureAssignmentsResponseObject, error) {
	requestCtx, actor := strictGinRequestDetails(ctx)
	if actor == 0 {
		return apicontract.UpdateCorpStructureAssignments401JSONResponse{UnauthorizedJSONResponse: corpStructureUnauthorized()}, nil
	}
	out := &apicontract.UpdateCorpStructureAssignments200JSONResponse{}
	if err := h.svc.UpdateAssignments(requestCtx, corpStructureAssignmentInput(*request.Body, actor)); err != nil {
		return out, out.FromBusinessError(corpStructureBusinessError(err))
	}
	return out, out.FromNullSuccess(apicontract.NullSuccess{Code: response.CodeOK, Msg: "success"})
}

func (h *CorporationStructureHandler) GetCorpStructureFuelSalarySettings(_ context.Context, _ apicontract.GetCorpStructureFuelSalarySettingsRequestObject) (apicontract.GetCorpStructureFuelSalarySettingsResponseObject, error) {
	out := &apicontract.GetCorpStructureFuelSalarySettings200JSONResponse{}
	result, err := h.svc.GetFuelSalarySettings()
	if err != nil {
		return out, out.FromBusinessError(corpStructureBusinessError(err))
	}
	return out, out.FromCorpStructureFuelSalarySettingsSuccess(apicontract.CorpStructureFuelSalarySettingsSuccess{Code: response.CodeOK, Msg: "success", Data: corpStructureFuelSalaryDTO(*result)})
}

func (h *CorporationStructureHandler) UpdateCorpStructureFuelSalarySettings(ctx context.Context, request apicontract.UpdateCorpStructureFuelSalarySettingsRequestObject) (apicontract.UpdateCorpStructureFuelSalarySettingsResponseObject, error) {
	_, actor := strictGinRequestDetails(ctx)
	if actor == 0 {
		return apicontract.UpdateCorpStructureFuelSalarySettings401JSONResponse{UnauthorizedJSONResponse: corpStructureUnauthorized()}, nil
	}
	out := &apicontract.UpdateCorpStructureFuelSalarySettings200JSONResponse{}
	if err := h.svc.UpdateFuelSalarySettings(service.FuelSalarySettingsUpdateRequest{
		SalaryPerStructureMonthly: request.Body.SalaryPerStructureMonthly,
		OperatorUserID:            actor,
	}); err != nil {
		return out, out.FromBusinessError(corpStructureBusinessError(err))
	}
	return out, out.FromNullSuccess(apicontract.NullSuccess{Code: response.CodeOK, Msg: "success"})
}

func (h *CorporationStructureHandler) RunCorpStructureFuelSalaryPayout(ctx context.Context, request apicontract.RunCorpStructureFuelSalaryPayoutRequestObject) (apicontract.RunCorpStructureFuelSalaryPayoutResponseObject, error) {
	requestCtx, actor := strictGinRequestDetails(ctx)
	if actor == 0 {
		return apicontract.RunCorpStructureFuelSalaryPayout401JSONResponse{UnauthorizedJSONResponse: corpStructureUnauthorized()}, nil
	}
	out := &apicontract.RunCorpStructureFuelSalaryPayout200JSONResponse{}
	data, err := h.runCorpStructureFuelSalaryPayout(requestCtx, actor, *request.Body)
	if err != nil {
		return out, out.FromBusinessError(corpStructureBusinessError(err))
	}
	return out, out.FromCorpStructureFuelPayoutSuccess(apicontract.CorpStructureFuelPayoutSuccess{Code: response.CodeOK, Msg: "success", Data: data})
}

func (h *CorporationStructureHandler) ListMyAssignedCorpStructures(ctx context.Context, request apicontract.ListMyAssignedCorpStructuresRequestObject) (apicontract.ListMyAssignedCorpStructuresResponseObject, error) {
	requestCtx, actor := strictGinRequestDetails(ctx)
	if actor == 0 {
		return apicontract.ListMyAssignedCorpStructures401JSONResponse{UnauthorizedJSONResponse: corpStructureUnauthorized()}, nil
	}
	out := &apicontract.ListMyAssignedCorpStructures200JSONResponse{}
	result, err := h.svc.ListMyAssignedStructures(requestCtx, actor, corpStructureListInput(*request.Body))
	if err != nil {
		return out, out.FromBusinessError(corpStructureBusinessError(err))
	}
	return out, out.FromCorpStructureListSuccess(apicontract.CorpStructureListSuccess{Code: response.CodeOK, Msg: "success", Data: corpStructureListDTO(*result)})
}

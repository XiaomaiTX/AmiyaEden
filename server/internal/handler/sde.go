package handler

import (
	"amiya-eden/global"
	"amiya-eden/internal/apicontract"
	"amiya-eden/internal/service"
	"amiya-eden/pkg/response"
	"context"
	"sort"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// SdeHandler SDE HTTP 处理器
type SdeHandler struct {
	svc                *service.SdeService
	entityNameResolver *service.EntityNameResolver
}

func NewSdeHandler() *SdeHandler {
	return &SdeHandler{
		svc:                service.NewSdeService(),
		entityNameResolver: service.NewEntityNameResolver(),
	}
}

const sdeNoVersionMessage = "尚未导入任何 SDE 版本"

func (h *SdeHandler) getSdeVersion() (*apicontract.SdeVersion, *contractOperationFailure) {
	v, err := h.svc.GetCurrentVersion()
	if err != nil {
		return nil, &contractOperationFailure{response.CodeBizError, "查询版本失败: " + err.Error()}
	}
	if v == nil {
		return nil, nil
	}
	return &apicontract.SdeVersion{
		Id: v.ID, Version: v.Version, BuildHash: v.BuildHash, Note: v.Note,
		CreatedAt: v.CreatedAt, UpdatedAt: v.UpdatedAt,
	}, nil
}

func (h *SdeHandler) getSdeTypes(req apicontract.GetSdeTypesJSONRequestBody) ([]apicontract.SdeType, *contractOperationFailure) {
	if len(req.TypeIds) == 0 {
		return nil, &contractOperationFailure{response.CodeParamError, "type_ids 不能为空"}
	}
	if req.LanguageId == "" {
		req.LanguageId = "en"
	}

	list, err := h.svc.GetTypes(req.TypeIds, req.Published, req.LanguageId)
	if err != nil {
		return nil, &contractOperationFailure{response.CodeBizError, "查询失败: " + err.Error()}
	}
	var out []apicontract.SdeType
	if list != nil {
		out = make([]apicontract.SdeType, len(list))
		for i, item := range list {
			out[i] = apicontract.SdeType{
				TypeId: item.TypeID, TypeName: item.TypeName, GroupId: item.GroupID, GroupName: item.GroupName,
				MarketGroupId: item.MarketGroupID, MarketGroupName: item.MarketGroupName,
				CategoryId: item.CategoryID, CategoryName: item.CategoryName,
			}
		}
	}
	return out, nil
}

type getNamesESIEntry struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

func newGetNamesResponse() apicontract.SdeNames {
	return apicontract.SdeNames{
		Flat:  make(map[int]string),
		Names: make(map[string]map[int]string),
	}
}

func mergeGetNamesNamespaces(result *apicontract.SdeNames, names service.SdeNameMap) {
	namespaces := make([]string, 0, len(names))
	for namespace := range names {
		namespaces = append(namespaces, namespace)
	}
	sort.Strings(namespaces)

	for _, namespace := range namespaces {
		result.Names[namespace] = names[namespace]
		for id, name := range names[namespace] {
			if _, exists := result.Flat[id]; !exists {
				result.Flat[id] = name
			}
		}
	}
}

func mergeGetNamesESI(result *apicontract.SdeNames, entries []getNamesESIEntry) {
	if _, ok := result.Names["esi"]; !ok {
		result.Names["esi"] = make(map[int]string, len(entries))
	}
	for _, entry := range entries {
		result.Names["esi"][entry.ID] = entry.Name
		if _, exists := result.Flat[entry.ID]; !exists {
			result.Flat[entry.ID] = entry.Name
		}
	}
}

func mergeGetNamesESIMisses(result *apicontract.SdeNames, misses []int64) {
	if len(misses) == 0 {
		return
	}
	if result.Misses == nil {
		result.Misses = &apicontract.SdeNameMisses{}
	}
	result.Misses.Esi = make([]int, 0, len(misses))
	for _, id := range misses {
		if id > 0 {
			result.Misses.Esi = append(result.Misses.Esi, int(id))
		}
	}
}

// sdeRequestLanguage preserves body, Accept-Language, cookie, then English precedence.
func sdeRequestLanguage(c *gin.Context, bodyLanguage string) string {
	if bodyLanguage != "" {
		return bodyLanguage
	}
	if c != nil {
		if headerLanguage := c.GetHeader("Accept-Language"); headerLanguage != "" {
			return headerLanguage
		}
		if cookieLanguage, err := c.Cookie("language"); err == nil && cookieLanguage != "" {
			return cookieLanguage
		}
	}
	return "en"
}

func (h *SdeHandler) getSdeNames(req apicontract.GetSdeNamesJSONRequestBody) (apicontract.SdeNames, *contractOperationFailure) {
	if len(req.Ids) == 0 && len(req.Esi) == 0 {
		return apicontract.SdeNames{}, &contractOperationFailure{response.CodeParamError, "ids 和 esi 不能同时为空"}
	}

	result := newGetNamesResponse()

	// 查数据库翻译
	if len(req.Ids) > 0 {
		dbNames, err := h.svc.GetNames(req.Ids, req.Language)
		if err != nil {
			return apicontract.SdeNames{}, &contractOperationFailure{response.CodeBizError, "查询失败: " + err.Error()}
		}
		mergeGetNamesNamespaces(&result, dbNames)
	}

	// 调用 ESI /universe/names 查询 character/corporation/alliance
	if len(req.Esi) > 0 {
		// 过滤掉无效 ID（0 或负数）
		validESI := make([]int64, 0, len(req.Esi))
		for _, id := range req.Esi {
			if id > 0 {
				validESI = append(validESI, id)
			}
		}
		if len(validESI) > 0 {
			resolved := h.entityNameResolver.Resolve(context.Background(), validESI)
			if len(resolved.Names) == 0 && len(resolved.Miss) == len(validESI) {
				global.Logger.Warn("[SDE] ESI name resolve returned zero hit",
					zap.Int("requested", len(validESI)),
				)
			}
			esiResult := make([]getNamesESIEntry, 0, len(resolved.Names))
			for id, name := range resolved.Names {
				esiResult = append(esiResult, getNamesESIEntry{
					ID:   int(id),
					Name: name,
				})
			}
			mergeGetNamesESI(&result, esiResult)
			mergeGetNamesESIMisses(&result, resolved.Miss)
		}
	}

	return result, nil
}

func (h *SdeHandler) searchSde(req apicontract.SearchSdeJSONRequestBody) ([]apicontract.SdeSearchItem, *contractOperationFailure) {
	if req.Limit <= 0 {
		req.Limit = 20
	}

	list, err := h.svc.FuzzySearch(req.Keyword, req.Language, req.CategoryIds, req.ExcludeCategoryIds, req.Limit, req.SearchMember)
	if err != nil {
		return nil, &contractOperationFailure{response.CodeBizError, "查询失败: " + err.Error()}
	}
	var out []apicontract.SdeSearchItem
	if list != nil {
		out = make([]apicontract.SdeSearchItem, len(list))
		for i, item := range list {
			out[i] = apicontract.SdeSearchItem{Id: item.ID, Name: item.Name, GroupId: item.GroupID, GroupName: item.GroupName, Category: item.Category}
		}
	}
	return out, nil
}

func (h *SdeHandler) GetSdeVersion(_ context.Context, _ apicontract.GetSdeVersionRequestObject) (apicontract.GetSdeVersionResponseObject, error) {
	out := &apicontract.GetSdeVersion200JSONResponse{}
	version, failure := h.getSdeVersion()
	switch {
	case failure != nil:
		return out, out.FromBusinessError(contractBusinessError(failure.code, failure.message))
	case version == nil:
		return out, out.FromSdeNoVersionSuccess(apicontract.SdeNoVersionSuccess{
			Code: response.CodeOK, Msg: "success", Data: apicontract.SdeNoVersion{Message: sdeNoVersionMessage},
		})
	default:
		return out, out.FromSdeVersionSuccess(apicontract.SdeVersionSuccess{
			Code: response.CodeOK, Msg: "success", Data: *version,
		})
	}
}

func (h *SdeHandler) GetSdeTypes(_ context.Context, request apicontract.GetSdeTypesRequestObject) (apicontract.GetSdeTypesResponseObject, error) {
	out := &apicontract.GetSdeTypes200JSONResponse{}
	if request.Body == nil {
		return out, out.FromBusinessError(contractBusinessError(response.CodeParamError, "参数错误"))
	}
	data, failure := h.getSdeTypes(*request.Body)
	if failure != nil {
		return out, out.FromBusinessError(contractBusinessError(failure.code, failure.message))
	}
	return out, out.FromSdeTypesSuccess(apicontract.SdeTypesSuccess{Code: response.CodeOK, Msg: "success", Data: &data})
}

func (h *SdeHandler) GetSdeNames(ctx context.Context, request apicontract.GetSdeNamesRequestObject) (apicontract.GetSdeNamesResponseObject, error) {
	out := &apicontract.GetSdeNames200JSONResponse{}
	if request.Body == nil {
		return out, out.FromBusinessError(contractBusinessError(response.CodeParamError, "参数错误"))
	}
	c, _ := ctx.(*gin.Context)
	input := *request.Body
	input.Language = sdeRequestLanguage(c, input.Language)
	data, failure := h.getSdeNames(input)
	if failure != nil {
		return out, out.FromBusinessError(contractBusinessError(failure.code, failure.message))
	}
	return out, out.FromSdeNamesSuccess(apicontract.SdeNamesSuccess{Code: response.CodeOK, Msg: "success", Data: data})
}

func (h *SdeHandler) SearchSde(ctx context.Context, request apicontract.SearchSdeRequestObject) (apicontract.SearchSdeResponseObject, error) {
	out := &apicontract.SearchSde200JSONResponse{}
	if request.Body == nil {
		return out, out.FromBusinessError(contractBusinessError(response.CodeParamError, "参数错误"))
	}
	c, _ := ctx.(*gin.Context)
	input := *request.Body
	input.Language = sdeRequestLanguage(c, input.Language)
	data, failure := h.searchSde(input)
	if failure != nil {
		return out, out.FromBusinessError(contractBusinessError(failure.code, failure.message))
	}
	return out, out.FromSdeSearchSuccess(apicontract.SdeSearchSuccess{Code: response.CodeOK, Msg: "success", Data: &data})
}

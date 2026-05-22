package handler

import (
	"amiya-eden/global"
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

// GetVersion godoc
// GET /api/v1/sde/version
// 获取当前已导入的 SDE 版本信息
func (h *SdeHandler) GetVersion(c *gin.Context) {
	v, err := h.svc.GetCurrentVersion()
	if err != nil {
		response.Fail(c, response.CodeBizError, "查询版本失败: "+err.Error())
		return
	}
	if v == nil {
		response.OK(c, gin.H{"version": nil, "message": "尚未导入任何 SDE 版本"})
		return
	}
	response.OK(c, v)
}

// GetTypes godoc
// POST /api/v1/sde/types
// 批量查询物品信息（含 group + category + market_group 翻译）
type GetTypesRequest struct {
	TypeIDs    []int  `json:"type_ids"`
	Published  *bool  `json:"published"`
	LanguageID string `json:"language_id"`
}

func (h *SdeHandler) GetTypes(c *gin.Context) {
	var req GetTypesRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, response.CodeParamError, "参数错误: "+err.Error())
		return
	}
	if len(req.TypeIDs) == 0 {
		response.Fail(c, response.CodeParamError, "type_ids 不能为空")
		return
	}
	if req.LanguageID == "" {
		req.LanguageID = "en"
	}

	list, err := h.svc.GetTypes(req.TypeIDs, req.Published, req.LanguageID)
	if err != nil {
		response.Fail(c, response.CodeBizError, "查询失败: "+err.Error())
		return
	}
	response.OK(c, list)
}

// GetNames godoc
// POST /api/v1/sde/names
// 批量查询名称映射
type GetNamesRequest struct {
	Language string           `json:"language"`
	IDs      map[string][]int `json:"ids"` // key 为 tcID 名称：type/group/category/region/constellation/solar_system/market_group/tech/description
	ESI      []int64          `json:"esi"` // character/corporation/alliance id，调用 ESI /universe/names
}

type GetNamesResponse struct {
	Flat  map[int]string            `json:"flat"`
	Names map[string]map[int]string `json:"names"`
	Miss  *GetNamesMisses           `json:"misses,omitempty"`
}

type GetNamesMisses struct {
	ESI []int `json:"esi,omitempty"`
}

type getNamesESIEntry struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

func newGetNamesResponse() GetNamesResponse {
	return GetNamesResponse{
		Flat:  make(map[int]string),
		Names: make(map[string]map[int]string),
	}
}

func mergeGetNamesNamespaces(result *GetNamesResponse, names service.SdeNameMap) {
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

func mergeGetNamesESI(result *GetNamesResponse, entries []getNamesESIEntry) {
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

func mergeGetNamesESIMisses(result *GetNamesResponse, misses []int64) {
	if len(misses) == 0 {
		return
	}
	if result.Miss == nil {
		result.Miss = &GetNamesMisses{}
	}
	result.Miss.ESI = make([]int, 0, len(misses))
	for _, id := range misses {
		if id > 0 {
			result.Miss.ESI = append(result.Miss.ESI, int(id))
		}
	}
}

func (h *SdeHandler) GetNames(c *gin.Context) {
	var req GetNamesRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, response.CodeParamError, "参数错误: "+err.Error())
		return
	}
	// language 优先 body，其次 Accept-Language header，其次 cookie，最后默认 en
	if req.Language == "" {
		req.Language = c.GetHeader("Accept-Language")
	}
	if req.Language == "" {
		if lang, err := c.Cookie("language"); err == nil && lang != "" {
			req.Language = lang
		}
	}
	if req.Language == "" {
		req.Language = "en"
	}
	if len(req.IDs) == 0 && len(req.ESI) == 0 {
		response.Fail(c, response.CodeParamError, "ids 和 esi 不能同时为空")
		return
	}

	result := newGetNamesResponse()

	// 查数据库翻译
	if len(req.IDs) > 0 {
		dbNames, err := h.svc.GetNames(req.IDs, req.Language)
		if err != nil {
			response.Fail(c, response.CodeBizError, "查询失败: "+err.Error())
			return
		}
		mergeGetNamesNamespaces(&result, dbNames)
	}

	// 调用 ESI /universe/names 查询 character/corporation/alliance
	if len(req.ESI) > 0 {
		// 过滤掉无效 ID（0 或负数）
		validESI := make([]int64, 0, len(req.ESI))
		for _, id := range req.ESI {
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

	response.OK(c, result)
}

// FuzzySearch godoc
// POST /api/v1/sde/search
// 模糊搜索物品/成员名称
type FuzzySearchRequest struct {
	Keyword            string `json:"keyword" binding:"required"`
	Language           string `json:"language"`
	CategoryIDs        []int  `json:"category_ids"`
	ExcludeCategoryIDs []int  `json:"exclude_category_ids"`
	Limit              int    `json:"limit"`
	SearchMember       bool   `json:"search_member"`
}

func (h *SdeHandler) FuzzySearch(c *gin.Context) {
	var req FuzzySearchRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, response.CodeParamError, "参数错误: "+err.Error())
		return
	}
	if req.Language == "" {
		req.Language = c.GetHeader("Accept-Language")
	}
	if req.Language == "" {
		if lang, err := c.Cookie("language"); err == nil && lang != "" {
			req.Language = lang
		}
	}
	if req.Language == "" {
		req.Language = "en"
	}
	if req.Limit <= 0 {
		req.Limit = 20
	}

	list, err := h.svc.FuzzySearch(req.Keyword, req.Language, req.CategoryIDs, req.ExcludeCategoryIDs, req.Limit, req.SearchMember)
	if err != nil {
		response.Fail(c, response.CodeBizError, "查询失败: "+err.Error())
		return
	}
	response.OK(c, list)
}

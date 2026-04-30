package handler

import (
	"amiya-eden/internal/utils"
	"amiya-eden/pkg/response"
	"fmt"
	"math"
	"strconv"

	"github.com/gin-gonic/gin"
)

func bindJSON(c *gin.Context, req any) bool {
	if err := c.ShouldBindJSON(req); err != nil {
		response.Fail(c, response.CodeParamError, "请求参数错误: "+err.Error())
		return false
	}
	return true
}

// requireUintID extracts, validates, and returns a uint path param.
// Returns 0 and writes an error response if the param is missing or invalid.
func requireUintID(c *gin.Context, param string, displayNames ...string) uint {
	raw := c.Param(param)
	id, err := parseStrictUint(raw)
	if err != nil {
		displayName := param
		if len(displayNames) > 0 && displayNames[0] != "" {
			displayName = displayNames[0]
		}
		response.Fail(c, response.CodeParamError, fmt.Sprintf("无效的%s", displayName))
		return 0
	}
	return uint(id)
}

func normalizePage(page int) int {
	return utils.NormalizePage(page)
}

func normalizePageSize(pageSize, defaultPageSize, maxPageSize int) int {
	return utils.NormalizePageSize(pageSize, defaultPageSize, maxPageSize)
}

func normalizeLedgerPageSize(pageSize int) int {
	return utils.NormalizeLedgerPageSize(pageSize)
}

func normalizePagination(page, pageSize, defaultPageSize, maxPageSize int) (int, int) {
	return normalizePage(page), normalizePageSize(pageSize, defaultPageSize, maxPageSize)
}

func normalizeLedgerPagination(page, pageSize int) (int, int) {
	return normalizePage(page), normalizeLedgerPageSize(pageSize)
}

func parsePaginationQuery(c *gin.Context, defaultPageSize, maxPageSize int) (int, int, error) {
	page, err := parseIntQuery(c, "current", utils.FirstPage)
	if err != nil {
		return 0, 0, err
	}

	pageSize, err := parseIntQuery(c, "size", defaultPageSize)
	if err != nil {
		return 0, 0, err
	}

	page, pageSize = normalizePagination(page, pageSize, defaultPageSize, maxPageSize)
	return page, pageSize, nil
}

func parseUnboundedPaginationQuery(c *gin.Context, defaultPageSize int) (int, int, error) {
	page, err := parseIntQuery(c, "current", utils.FirstPage)
	if err != nil {
		return 0, 0, err
	}

	pageSize, err := parseIntQuery(c, "size", defaultPageSize)
	if err != nil {
		return 0, 0, err
	}

	page = normalizePage(page)
	if pageSize < utils.FirstPage {
		pageSize = defaultPageSize
	}
	// Cap at a safe maximum to prevent oversized responses
	const maxUnboundedPageSize = 1000
	if pageSize > maxUnboundedPageSize {
		pageSize = maxUnboundedPageSize
	}

	return page, pageSize, nil
}

func parseLedgerPaginationQuery(c *gin.Context, defaultPageSize int) (int, int, error) {
	page, err := parseIntQuery(c, "current", utils.FirstPage)
	if err != nil {
		return 0, 0, err
	}

	pageSize, err := parseIntQuery(c, "size", defaultPageSize)
	if err != nil {
		return 0, 0, err
	}

	page = normalizePage(page)
	if pageSize < utils.FirstPage {
		pageSize = defaultPageSize
	} else {
		pageSize = normalizeLedgerPageSize(pageSize)
	}

	return page, pageSize, nil
}

func parseIntQuery(c *gin.Context, key string, defaultValue int) (int, error) {
	raw := c.Query(key)
	if raw == "" {
		return defaultValue, nil
	}

	value, err := strconv.Atoi(raw)
	if err != nil {
		return 0, fmt.Errorf("invalid %s query parameter: expected integer", key)
	}

	return value, nil
}

func parseStrictUint(raw string) (uint, error) {
	parsed, err := strconv.ParseUint(raw, 10, 64)
	if err != nil || parsed == 0 || parsed > math.MaxUint32 {
		return 0, fmt.Errorf("invalid unsigned integer")
	}
	return uint(parsed), nil
}

func parseRequiredUintQueryParam(field, raw string) (uint, error) {
	v, err := parseStrictUint(raw)
	if err != nil {
		return 0, fmt.Errorf("invalid parameter %s", field)
	}
	return v, nil
}

func parseOptionalUintQueryParam(field, raw string) (*uint, error) {
	if raw == "" {
		return nil, nil
	}
	v, err := parseRequiredUintQueryParam(field, raw)
	if err != nil {
		return nil, err
	}
	return &v, nil
}

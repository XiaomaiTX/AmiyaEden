package esi

import (
	"errors"
	"fmt"
)

// HTTPError 表示 ESI 返回了非 2xx 状态码。
// Error() 文本与历史 fmt.Errorf 格式保持一致，供调用方按状态码优雅降级。
type HTTPError struct {
	StatusCode int
	Path       string
	Body       string
	Page       int    // 分页请求的页码，0 表示非分页
	op         string // 错误消息中的方法片段，如 "POST "；GET 为空
}

func (e *HTTPError) Error() string {
	if e.Page > 0 {
		return fmt.Sprintf("ESI error %d on page %d of %s: %s", e.StatusCode, e.Page, e.Path, e.Body)
	}
	return fmt.Sprintf("ESI error %d on %s%s: %s", e.StatusCode, e.op, e.Path, e.Body)
}

func newHTTPError(statusCode int, op, path, body string) *HTTPError {
	return &HTTPError{StatusCode: statusCode, Path: path, Body: body, op: op}
}

func newPaginatedHTTPError(statusCode, page int, path, body string) *HTTPError {
	return &HTTPError{StatusCode: statusCode, Path: path, Body: body, Page: page}
}

// IsHTTPStatus 判断 err（含 wrap）是否为指定状态码的 ESI HTTPError
func IsHTTPStatus(err error, statusCode int) bool {
	var httpErr *HTTPError
	if errors.As(err, &httpErr) {
		return httpErr.StatusCode == statusCode
	}
	return false
}

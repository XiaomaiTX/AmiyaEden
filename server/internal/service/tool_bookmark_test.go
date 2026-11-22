package service

import (
	"amiya-eden/global"
	"amiya-eden/internal/model"
	"net/http"
	"strings"
	"testing"
)

type stubHTTPClient struct {
	do func(req *http.Request) (*http.Response, error)
}

func (s stubHTTPClient) Do(req *http.Request) (*http.Response, error) {
	return s.do(req)
}

func newHTTPResponse(status int, body string) *http.Response {
	return &http.Response{
		StatusCode: status,
		Body:       ioNopCloser{Reader: strings.NewReader(body)},
		Header:     make(http.Header),
	}
}

type ioNopCloser struct {
	Reader *strings.Reader
}

func (c ioNopCloser) Read(p []byte) (int, error) {
	return c.Reader.Read(p)
}

func (c ioNopCloser) Close() error {
	return nil
}

func TestToolBookmarkServiceCreateWithHTMLIcon(t *testing.T) {
	db := newServiceTestDB(t, "tool_bookmark_html", &model.ToolBookmark{})
	originDB := global.DB
	global.DB = db
	t.Cleanup(func() { global.DB = originDB })

	svc := NewToolBookmarkService()
	svc.httpClient = stubHTTPClient{
		do: func(req *http.Request) (*http.Response, error) {
			if req.Method == http.MethodGet {
				return newHTTPResponse(http.StatusOK, `<html><head><link rel="icon" href="/assets/favicon.png"></head></html>`), nil
			}
			return newHTTPResponse(http.StatusOK, ""), nil
		},
	}

	row, err := svc.AdminCreate(7, ToolBookmarkUpsertRequest{
		Name: "工具A",
		URL:  "https://example.com/tools",
	})
	if err != nil {
		t.Fatalf("AdminCreate() error = %v", err)
	}
	if row.LogoURL != "https://example.com/assets/favicon.png" {
		t.Fatalf("logo_url = %q", row.LogoURL)
	}
	if row.LogoSource != "html" {
		t.Fatalf("logo_source = %q", row.LogoSource)
	}
	if row.CreatedBy != 7 {
		t.Fatalf("created_by = %d", row.CreatedBy)
	}
}

func TestToolBookmarkServiceGroups(t *testing.T) {
	db := newServiceTestDB(t, "tool_bookmark_groups", &model.ToolBookmark{})
	originDB := global.DB
	global.DB = db
	t.Cleanup(func() { global.DB = originDB })
	svc := NewToolBookmarkService()
	svc.auditSvc = nil
	svc.httpClient = stubHTTPClient{do: func(req *http.Request) (*http.Response, error) {
		return newHTTPResponse(http.StatusOK, ""), nil
	}}
	group := "  工具  "
	row, err := svc.AdminCreate(1, ToolBookmarkUpsertRequest{Name: "A", URL: "https://example.com", GroupName: &group})
	if err != nil || row.GroupName != "工具" {
		t.Fatalf("create group: row=%+v err=%v", row, err)
	}
	row, err = svc.AdminUpdate(1, row.ID, ToolBookmarkUpsertRequest{Name: "B"})
	if err != nil || row.GroupName != "工具" {
		t.Fatalf("omitted group: row=%+v err=%v", row, err)
	}
	group = "规章"
	row, err = svc.AdminUpdate(1, row.ID, ToolBookmarkUpsertRequest{GroupName: &group})
	if err != nil || row.GroupName != group {
		t.Fatalf("move group: row=%+v err=%v", row, err)
	}
	rows, err := svc.ListVisibleBookmarks()
	if err != nil || len(rows) != 1 || rows[0].GroupName != group {
		t.Fatalf("visible groups: rows=%+v err=%v", rows, err)
	}
	group = strings.Repeat("组", 65)
	if _, err = svc.AdminUpdate(1, row.ID, ToolBookmarkUpsertRequest{GroupName: &group}); !IsUserVisibleError(err) {
		t.Fatalf("expected group length error, got %v", err)
	}
	group = "  "
	row, err = svc.AdminUpdate(1, row.ID, ToolBookmarkUpsertRequest{GroupName: &group})
	if err != nil || row.GroupName != "" {
		t.Fatalf("clear group: row=%+v err=%v", row, err)
	}
}

func TestToolBookmarkServiceCreateFallbackToFavicon(t *testing.T) {
	db := newServiceTestDB(t, "tool_bookmark_fallback", &model.ToolBookmark{})
	originDB := global.DB
	global.DB = db
	t.Cleanup(func() { global.DB = originDB })

	svc := NewToolBookmarkService()
	svc.httpClient = stubHTTPClient{
		do: func(req *http.Request) (*http.Response, error) {
			if req.Method == http.MethodGet {
				return newHTTPResponse(http.StatusOK, `<html><head></head></html>`), nil
			}
			return newHTTPResponse(http.StatusOK, ""), nil
		},
	}

	row, err := svc.AdminCreate(1, ToolBookmarkUpsertRequest{
		Name: "工具B",
		URL:  "https://example.org",
	})
	if err != nil {
		t.Fatalf("AdminCreate() error = %v", err)
	}
	if row.LogoURL != "https://example.org/favicon.ico" {
		t.Fatalf("logo_url = %q", row.LogoURL)
	}
	if row.LogoSource != "favicon" {
		t.Fatalf("logo_source = %q", row.LogoSource)
	}
}

func TestToolBookmarkServiceCreateInvalidURL(t *testing.T) {
	db := newServiceTestDB(t, "tool_bookmark_invalid", &model.ToolBookmark{})
	originDB := global.DB
	global.DB = db
	t.Cleanup(func() { global.DB = originDB })

	svc := NewToolBookmarkService()
	_, err := svc.AdminCreate(1, ToolBookmarkUpsertRequest{
		Name: "工具C",
		URL:  "ftp://example.com",
	})
	if err == nil {
		t.Fatal("expected error for invalid URL scheme")
	}
}

func TestToolBookmarkServiceListVisibleFiltersDisabled(t *testing.T) {
	db := newServiceTestDB(t, "tool_bookmark_list", &model.ToolBookmark{})
	originDB := global.DB
	global.DB = db
	t.Cleanup(func() { global.DB = originDB })

	if err := db.Create(&model.ToolBookmark{Name: "A", URL: "https://a.com", IsEnabled: true, SortOrder: 2}).Error; err != nil {
		t.Fatalf("seed A: %v", err)
	}
	disabled := &model.ToolBookmark{Name: "B", URL: "https://b.com", IsEnabled: true, SortOrder: 1}
	if err := db.Create(disabled).Error; err != nil {
		t.Fatalf("seed B: %v", err)
	}
	if err := db.Model(&model.ToolBookmark{}).Where("id = ?", disabled.ID).Update("is_enabled", false).Error; err != nil {
		t.Fatalf("disable B: %v", err)
	}
	if err := db.Create(&model.ToolBookmark{Name: "C", URL: "https://c.com", IsEnabled: true, SortOrder: 1}).Error; err != nil {
		t.Fatalf("seed C: %v", err)
	}

	svc := NewToolBookmarkService()
	rows, err := svc.ListVisibleBookmarks()
	if err != nil {
		t.Fatalf("ListVisibleBookmarks() error = %v", err)
	}
	if len(rows) != 2 {
		t.Fatalf("len(rows) = %d, want 2", len(rows))
	}
	if rows[0].Name != "C" || rows[1].Name != "A" {
		t.Fatalf("unexpected order: %+v", rows)
	}
}

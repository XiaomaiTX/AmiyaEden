package service

import (
	"amiya-eden/internal/model"
	"amiya-eden/internal/repository"
	"errors"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	"gorm.io/gorm"
)

var toolBookmarkIconHrefPattern = regexp.MustCompile(`(?is)<link[^>]*rel=["'][^"']*icon[^"']*["'][^>]*href=["']([^"']+)["'][^>]*>`)

type toolBookmarkHTTPClient interface {
	Do(req *http.Request) (*http.Response, error)
}

type ToolBookmarkService struct {
	repo       *repository.ToolBookmarkRepository
	httpClient toolBookmarkHTTPClient
}

type ToolBookmarkUpsertRequest struct {
	Name        string `json:"name"`
	URL         string `json:"url"`
	Description string `json:"description"`
	IsEnabled   *bool  `json:"is_enabled"`
	SortOrder   *int   `json:"sort_order"`
}

func NewToolBookmarkService() *ToolBookmarkService {
	return &ToolBookmarkService{
		repo: repository.NewToolBookmarkRepository(),
		httpClient: &http.Client{
			Timeout: 6 * time.Second,
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				if len(via) >= 5 {
					return http.ErrUseLastResponse
				}
				return nil
			},
		},
	}
}

func (s *ToolBookmarkService) ListVisibleBookmarks() ([]model.ToolBookmark, error) {
	rows, err := s.repo.ListVisible()
	if err != nil {
		return nil, err
	}
	if rows == nil {
		return []model.ToolBookmark{}, nil
	}
	return rows, nil
}

func (s *ToolBookmarkService) AdminList() ([]model.ToolBookmark, error) {
	rows, err := s.repo.ListAdmin()
	if err != nil {
		return nil, err
	}
	if rows == nil {
		return []model.ToolBookmark{}, nil
	}
	return rows, nil
}

func (s *ToolBookmarkService) AdminCreate(operatorID uint, req ToolBookmarkUpsertRequest) (*model.ToolBookmark, error) {
	name := strings.TrimSpace(req.Name)
	if name == "" {
		return nil, NewUserVisibleError("名称不能为空")
	}
	normalizedURL, parsedURL, err := normalizeToolBookmarkURL(req.URL)
	if err != nil {
		return nil, err
	}

	maxSortOrder, err := s.repo.MaxSortOrder()
	if err != nil {
		return nil, err
	}

	logoURL, logoSource := s.resolveLogo(parsedURL)

	isEnabled := true
	if req.IsEnabled != nil {
		isEnabled = *req.IsEnabled
	}
	sortOrder := maxSortOrder + 1
	if req.SortOrder != nil {
		sortOrder = *req.SortOrder
	}
	row := &model.ToolBookmark{
		Name:        name,
		URL:         normalizedURL,
		Description: strings.TrimSpace(req.Description),
		LogoURL:     logoURL,
		LogoSource:  logoSource,
		IsEnabled:   isEnabled,
		SortOrder:   sortOrder,
		CreatedBy:   operatorID,
	}
	if err := s.repo.Create(row); err != nil {
		return nil, err
	}
	return row, nil
}

func (s *ToolBookmarkService) AdminUpdate(id uint, req ToolBookmarkUpsertRequest) (*model.ToolBookmark, error) {
	row, err := s.repo.GetByID(id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, NewUserVisibleError("书签不存在")
		}
		return nil, err
	}

	updates := map[string]interface{}{}
	if strings.TrimSpace(req.Name) != "" {
		updates["name"] = strings.TrimSpace(req.Name)
	}
	if req.Description != "" {
		updates["description"] = strings.TrimSpace(req.Description)
	}
	if req.IsEnabled != nil {
		updates["is_enabled"] = *req.IsEnabled
	}
	if req.SortOrder != nil {
		updates["sort_order"] = *req.SortOrder
	}
	if strings.TrimSpace(req.URL) != "" {
		normalizedURL, parsedURL, normalizeErr := normalizeToolBookmarkURL(req.URL)
		if normalizeErr != nil {
			return nil, normalizeErr
		}
		updates["url"] = normalizedURL
		logoURL, logoSource := s.resolveLogo(parsedURL)
		updates["logo_url"] = logoURL
		updates["logo_source"] = logoSource
	}

	if len(updates) == 0 {
		return row, nil
	}
	if err := s.repo.UpdateFields(id, updates); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, NewUserVisibleError("书签不存在")
		}
		return nil, err
	}
	return s.repo.GetByID(id)
}

func (s *ToolBookmarkService) AdminDelete(id uint) error {
	err := s.repo.Delete(id)
	if err != nil && errors.Is(err, gorm.ErrRecordNotFound) {
		return NewUserVisibleError("书签不存在")
	}
	return err
}

func normalizeToolBookmarkURL(raw string) (string, *url.URL, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return "", nil, NewUserVisibleError("URL 不能为空")
	}
	parsed, err := url.Parse(trimmed)
	if err != nil || parsed == nil {
		return "", nil, NewUserVisibleError("URL 格式无效")
	}
	scheme := strings.ToLower(parsed.Scheme)
	if scheme != "http" && scheme != "https" {
		return "", nil, NewUserVisibleError("URL 仅支持 http 或 https")
	}
	if strings.TrimSpace(parsed.Host) == "" {
		return "", nil, NewUserVisibleError("URL 缺少主机名")
	}
	parsed.Fragment = ""
	return parsed.String(), parsed, nil
}

func (s *ToolBookmarkService) resolveLogo(baseURL *url.URL) (string, string) {
	if baseURL == nil {
		return "", ""
	}
	if iconURL, err := s.fetchIconFromHTML(baseURL); err == nil && iconURL != "" {
		return iconURL, "html"
	}
	return (&url.URL{Scheme: baseURL.Scheme, Host: baseURL.Host, Path: "/favicon.ico"}).String(), "favicon"
}

func (s *ToolBookmarkService) fetchIconFromHTML(baseURL *url.URL) (string, error) {
	req, err := http.NewRequest(http.MethodGet, baseURL.String(), nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", "AmiyaEdenBot/1.0")
	resp, err := s.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 400 {
		if closeErr := resp.Body.Close(); closeErr != nil {
			return "", closeErr
		}
		return "", errors.New("non-2xx response")
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, 256*1024))
	closeErr := resp.Body.Close()
	if err != nil {
		return "", err
	}
	if closeErr != nil {
		return "", closeErr
	}
	match := toolBookmarkIconHrefPattern.FindStringSubmatch(string(body))
	if len(match) < 2 {
		return "", errors.New("icon link not found")
	}
	href := strings.TrimSpace(match[1])
	if href == "" {
		return "", errors.New("icon href empty")
	}
	ref, err := url.Parse(href)
	if err != nil {
		return "", err
	}
	return baseURL.ResolveReference(ref).String(), nil
}

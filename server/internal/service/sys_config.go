package service

import (
	"context"
	"errors"
	"fmt"
	"strconv"

	"amiya-eden/global"
	"amiya-eden/internal/model"
	"amiya-eden/internal/repository"
	"amiya-eden/internal/utils"
	"amiya-eden/pkg/eve/esi"

	"go.uber.org/zap"
)

var ErrInvalidAllowCorporations = errors.New("军团 ID 必须为正整数")

type SysConfigService struct {
	repo *repository.SysConfigRepository
}

type SDEConfig struct {
	APIKey      string
	Proxy       string
	DownloadURL string
}

type CorporationDisplay struct {
	CorporationID   int64  `json:"corporation_id"`
	CorporationName string `json:"corporation_name"`
}

func NewSysConfigService() *SysConfigService {
	return NewSysConfigServiceWithRepository(repository.NewSysConfigRepository())
}

func NewSysConfigServiceWithRepository(repo *repository.SysConfigRepository) *SysConfigService {
	return &SysConfigService{repo: repo}
}

func (s *SysConfigService) GetSDEConfig() SDEConfig {
	apiKey, _ := s.repo.Get(model.SysConfigSDEAPIKey, model.SysConfigDefaultSDEAPIKey)
	proxy, _ := s.repo.Get(model.SysConfigSDEProxy, model.SysConfigDefaultSDEProxy)
	downloadURL, _ := s.repo.Get(model.SysConfigSDEDownloadURL, model.SysConfigDefaultSDEDownloadURL)

	return SDEConfig{
		APIKey:      apiKey,
		Proxy:       proxy,
		DownloadURL: downloadURL,
	}
}

func (s *SysConfigService) UpdateSDEConfig(apiKey, proxy, downloadURL *string) error {
	if apiKey != nil {
		if err := s.repo.Set(model.SysConfigSDEAPIKey, *apiKey, "SDE 查询 API Key"); err != nil {
			return errors.New("更新 API Key 失败")
		}
	}
	if proxy != nil {
		if err := s.repo.Set(model.SysConfigSDEProxy, *proxy, "SDE 下载代理"); err != nil {
			return errors.New("更新代理配置失败")
		}
	}
	if downloadURL != nil {
		if err := s.repo.Set(model.SysConfigSDEDownloadURL, *downloadURL, "SDE 下载地址"); err != nil {
			return errors.New("更新下载地址失败")
		}
	}
	return nil
}

func (s *SysConfigService) GetAllowCorporations() []int64 {
	return utils.GetAllowCorporations()
}

func (s *SysConfigService) GetAllowCorporationDisplays(ctx context.Context) []CorporationDisplay {
	allowCorporations := s.GetAllowCorporations()
	if len(allowCorporations) == 0 {
		return []CorporationDisplay{}
	}

	nameMap, err := s.resolveCorporationNames(ctx, allowCorporations)
	if err != nil {
		if global.Logger != nil {
			global.Logger.Warn("[SysConfig] resolve allow corporations names failed", zap.Error(err))
		}
	}

	displays := make([]CorporationDisplay, 0, len(allowCorporations))
	for _, corporationID := range allowCorporations {
		displays = append(displays, CorporationDisplay{
			CorporationID:   corporationID,
			CorporationName: nameMap[corporationID],
		})
	}
	return displays
}

func (s *SysConfigService) UpdateAllowCorporations(allowCorporations []int64) error {
	if err := utils.ValidateAllowCorporations(allowCorporations); err != nil {
		return ErrInvalidAllowCorporations
	}
	normalizedAllowCorporations := utils.NormalizeAllowCorporations(allowCorporations)
	if err := s.repo.SetInt64Slice(model.SysConfigAllowCorporations, normalizedAllowCorporations, "允许访问的公司 ID 列表"); err != nil {
		return errors.New("更新允许的军团列表失败")
	}
	utils.InvalidateAllowCorporationsCache()
	return nil
}

type sysConfigESINameEntry struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

func (s *SysConfigService) resolveCorporationNames(ctx context.Context, corporationIDs []int64) (map[int64]string, error) {
	nameMap := make(map[int64]string, len(corporationIDs))
	if len(corporationIDs) == 0 {
		return nameMap, nil
	}

	client := esi.NewClient()
	if global.Config != nil {
		client = esi.NewClientWithConfig(global.Config.EveSSO.ESIBaseURL, global.Config.EveSSO.ESIAPIPrefix)
	}

	var result []sysConfigESINameEntry
	if err := client.PostJSON(
		ctx,
		"/universe/names?datasource=tranquility",
		"",
		corporationIDs,
		&result,
	); err != nil {
		return nameMap, fmt.Errorf("fetch corporation names from ESI: %w", err)
	}

	for _, entry := range result {
		nameMap[int64(entry.ID)] = entry.Name
	}
	return nameMap, nil
}

func (s *SysConfigService) GetCharacterESIRestrictionConfig() bool {
	return s.repo.GetBool(
		model.SysConfigEnforceCharacterESIRestriction,
		model.SysConfigDefaultEnforceCharacterESIRestriction,
	)
}

func (s *SysConfigService) UpdateCharacterESIRestrictionConfig(enforce bool) error {
	if err := s.repo.Set(
		model.SysConfigEnforceCharacterESIRestriction,
		strconv.FormatBool(enforce),
		"是否强制限制失效人物 ESI 停留在人物页面",
	); err != nil {
		return errors.New("更新人物 ESI 限制配置失败")
	}
	return nil
}

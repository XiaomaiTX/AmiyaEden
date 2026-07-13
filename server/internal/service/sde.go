package service

import (
	"amiya-eden/global"
	"amiya-eden/internal/model"
	"amiya-eden/internal/repository"
	"archive/zip"
	"bufio"
	"compress/bzip2"
	"compress/gzip"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"go.uber.org/zap"
	"gorm.io/gorm"
)

// sdeDownloadDir 临时文件存放目录
const sdeDownloadDir = "tmp/sde"

const (
	sdeCheckMinInterval = 30 * time.Minute
	sdeUserAgent        = "AmiyaEden-SDE-Updater/1.0"
	maxExtractDepth     = 4
)

var sdeUpdateState = struct {
	mu              sync.Mutex
	lastAutoCheckAt time.Time
	lastSeenVersion string
}{}

type TypeInfo = repository.TypeInfo
type SdeNameMap = repository.SdeNameMap
type FuzzySearchItem = repository.FuzzySearchItem

// SdeService SDE 业务逻辑层
type SdeService struct {
	repo      *repository.SdeRepository
	sysConfig *repository.SysConfigRepository
}

type SDEStatus struct {
	CurrentVersion    string `json:"current_version"`
	LatestVersion     string `json:"latest_version"`
	HasUpdate         bool   `json:"has_update"`
	LastQueryError    string `json:"last_query_error"`
	LastQueryErrorAt  int64  `json:"last_query_error_at"`
	LastQuerySource   string `json:"last_query_error_source"`
	LastCheckAt       int64  `json:"last_check_at"`
	LastCheckSuccess  bool   `json:"last_check_success"`
	LastCheckError    string `json:"last_check_error"`
	LastUpdateAt      int64  `json:"last_update_at"`
	LastUpdateSuccess bool   `json:"last_update_success"`
	LastUpdateError   string `json:"last_update_error"`
	IsUpdating        bool   `json:"is_updating"`
	UpdateStage       string `json:"update_stage"`
}

const (
	sdeUpdateStageChecking    = "checking"
	sdeUpdateStageDownloading = "downloading"
	sdeUpdateStageExtracting  = "extracting"
	sdeUpdateStageImporting   = "importing"
	sdeUpdateStageRecording   = "recording"
	sdeUpdateStageDone        = "done"
	sdeUpdateStageFailed      = "failed"
)

func NewSdeService() *SdeService {
	return &SdeService{
		repo:      repository.NewSdeRepository(),
		sysConfig: repository.NewSysConfigRepository(),
	}
}

// 获取配置的辅助方法
func (s *SdeService) getProxy() string {
	proxy, _ := s.sysConfig.Get(model.SysConfigSDEProxy, model.SysConfigDefaultSDEProxy)
	return proxy
}

func (s *SdeService) getDownloadURL() string {
	url, _ := s.sysConfig.Get(model.SysConfigSDEDownloadURL, model.SysConfigDefaultSDEDownloadURL)
	return url
}

// ---- 版本管理 ----

// GetCurrentVersion 获取当前已导入的 SDE 版本
func (s *SdeService) GetCurrentVersion() (*model.SdeVersion, error) {
	v, err := s.repo.GetLatestVersion()
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return v, err
}

func (s *SdeService) GetStatus() (SDEStatus, error) {
	status, err := s.getStatusSnapshot()
	if err != nil {
		return SDEStatus{}, err
	}

	currentVersion, err := s.getCurrentVersionString()
	if err != nil {
		return SDEStatus{}, err
	}
	status.CurrentVersion = currentVersion
	if status.LatestVersion != "" && status.CurrentVersion != "" {
		status.HasUpdate = status.LatestVersion != status.CurrentVersion
	}
	return status, nil
}

func (s *SdeService) CheckLatestVersion() (SDEStatus, error) {
	status, err := s.GetStatus()
	if err != nil {
		return SDEStatus{}, err
	}

	release, err := s.fetchLatestRelease()
	now := time.Now().Unix()
	status.LastCheckAt = now
	if err != nil {
		status.LastCheckSuccess = false
		status.LastCheckError = err.Error()
		if saveErr := s.setStatusSnapshot(status); saveErr != nil {
			global.Logger.Warn("[SDE] 保存状态快照失败", zap.Error(saveErr))
		}
		return status, fmt.Errorf("获取 GitHub Release 失败: %w", err)
	}

	status.LatestVersion = release.TagName
	status.LastCheckSuccess = true
	status.LastCheckError = ""
	status.HasUpdate = status.CurrentVersion != "" && status.CurrentVersion != status.LatestVersion
	if status.CurrentVersion == "" && status.LatestVersion != "" {
		status.HasUpdate = true
	}
	if saveErr := s.setStatusSnapshot(status); saveErr != nil {
		global.Logger.Warn("[SDE] 保存状态快照失败", zap.Error(saveErr))
	}
	return status, nil
}

func (s *SdeService) TriggerManualUpdateWithStatus() (SDEStatus, error) {
	status, err := s.GetStatus()
	if err != nil {
		return SDEStatus{}, err
	}

	status.IsUpdating = true
	status.UpdateStage = sdeUpdateStageChecking
	status.LastUpdateError = ""
	if saveErr := s.setStatusSnapshot(status); saveErr != nil {
		global.Logger.Warn("[SDE] 保存状态快照失败", zap.Error(saveErr))
	}

	version, updateErr := s.TriggerManualUpdate(func(stage string) {
		status.UpdateStage = stage
		status.IsUpdating = stage != sdeUpdateStageDone && stage != sdeUpdateStageFailed
		if saveErr := s.setStatusSnapshot(status); saveErr != nil {
			global.Logger.Warn("[SDE] 保存状态快照失败", zap.Error(saveErr))
		}
	})
	now := time.Now().Unix()
	status.LastUpdateAt = now
	status.LatestVersion = version
	if updateErr != nil {
		status.LastUpdateSuccess = false
		status.LastUpdateError = updateErr.Error()
		status.IsUpdating = false
		status.UpdateStage = sdeUpdateStageFailed
		status.HasUpdate = status.CurrentVersion != "" && status.CurrentVersion != status.LatestVersion
		if status.CurrentVersion == "" && status.LatestVersion != "" {
			status.HasUpdate = true
		}
		if saveErr := s.setStatusSnapshot(status); saveErr != nil {
			global.Logger.Warn("[SDE] 保存状态快照失败", zap.Error(saveErr))
		}
		return status, updateErr
	}

	status.CurrentVersion = version
	status.LastUpdateSuccess = true
	status.LastUpdateError = ""
	status.HasUpdate = false
	status.IsUpdating = false
	status.UpdateStage = sdeUpdateStageDone
	if saveErr := s.setStatusSnapshot(status); saveErr != nil {
		global.Logger.Warn("[SDE] 保存状态快照失败", zap.Error(saveErr))
	}
	return status, nil
}

func (s *SdeService) ReportQueryError(source, operation string, err error) {
	if err == nil {
		return
	}

	global.Logger.Warn("[SDE] 查询失败",
		zap.String("source", source),
		zap.String("operation", operation),
		zap.Error(err))

	status, statusErr := s.getStatusSnapshot()
	if statusErr != nil {
		global.Logger.Warn("[SDE] 读取状态快照失败，使用默认状态写入查询错误", zap.Error(statusErr))
		status = SDEStatus{}
	}

	source = strings.TrimSpace(source)
	operation = strings.TrimSpace(operation)
	if source == "" {
		source = "unknown"
	}
	if operation != "" {
		source = source + "." + operation
	}

	status.LastQuerySource = source
	status.LastQueryErrorAt = time.Now().Unix()
	status.LastQueryError = err.Error()
	if saveErr := s.setStatusSnapshot(status); saveErr != nil {
		global.Logger.Warn("[SDE] 保存状态快照失败", zap.Error(saveErr))
	}
}

// ---- SDE 更新 ----

// githubRelease GitHub Releases API 响应结构（仅摘取需要的字段）
type githubRelease struct {
	TagName string        `json:"tag_name"`
	Assets  []githubAsset `json:"assets"`
}

type githubAsset struct {
	Name               string `json:"name"`
	BrowserDownloadURL string `json:"browser_download_url"`
}

func pickSQLAsset(assets []githubAsset) *githubAsset {
	var fallback *githubAsset
	for i := range assets {
		name := strings.ToLower(assets[i].Name)
		isPostgres := strings.Contains(name, "postgres") || strings.Contains(name, "pgsql")
		isSQL := strings.Contains(name, ".sql")
		if !isPostgres && !isSQL {
			continue
		}

		candidate := &assets[i]
		if candidate.BrowserDownloadURL == "" {
			continue
		}
		if isPostgres {
			return candidate
		}
		if fallback == nil {
			fallback = candidate
			continue
		}
		if strings.HasSuffix(name, ".bz2") || strings.HasSuffix(name, ".gz") {
			fallback = candidate
		}
	}
	return fallback
}

// CheckAndUpdate 检查最新 SDE 版本，若有新版本则自动下载并导入
// 返回 (isUpdated, version, error)
func (s *SdeService) CheckAndUpdate() (bool, string, error) {
	sdeUpdateState.mu.Lock()
	defer sdeUpdateState.mu.Unlock()

	now := time.Now()
	if !sdeUpdateState.lastAutoCheckAt.IsZero() && now.Sub(sdeUpdateState.lastAutoCheckAt) < sdeCheckMinInterval {
		global.Logger.Info("[SDE] 跳过重复自动检查",
			zap.Duration("min_interval", sdeCheckMinInterval),
			zap.Time("last_check_at", sdeUpdateState.lastAutoCheckAt),
			zap.String("last_seen_version", sdeUpdateState.lastSeenVersion))
		return false, sdeUpdateState.lastSeenVersion, nil
	}
	sdeUpdateState.lastAutoCheckAt = now

	release, err := s.fetchLatestRelease()
	if err != nil {
		return false, "", fmt.Errorf("获取 GitHub Release 失败: %w", err)
	}

	version := release.TagName
	sdeUpdateState.lastSeenVersion = version
	exists, err := s.repo.VersionExists(version)
	if err != nil {
		global.Logger.Error("[SDE] 查询版本记录失败", zap.String("version", version), zap.Error(err))
		return false, version, fmt.Errorf("查询版本记录失败: %w", err)
	}
	if exists {
		global.Logger.Info("[SDE] 当前版本已是最新", zap.String("version", version))
		return false, version, nil
	}

	global.Logger.Info("[SDE] 发现新版本，开始更新", zap.String("version", version))
	if err := s.doImport(release, nil); err != nil {
		global.Logger.Error("[SDE] 更新失败", zap.String("version", version), zap.Error(err))
		return false, version, fmt.Errorf("导入 SDE 失败: %w", err)
	}

	if err := s.repo.CreateVersion(&model.SdeVersion{
		Version: version,
		Note:    "auto import from " + release.TagName,
	}); err != nil {
		global.Logger.Error("[SDE] 记录版本信息失败", zap.String("version", version), zap.Error(err))
		return true, version, fmt.Errorf("记录版本失败: %w", err)
	}

	global.Logger.Info("[SDE] 更新完成", zap.String("version", version))
	return true, version, nil
}

// TriggerManualUpdate 手动触发更新，强制重新导入
func (s *SdeService) TriggerManualUpdate(progress func(stage string)) (string, error) {
	sdeUpdateState.mu.Lock()
	defer sdeUpdateState.mu.Unlock()

	if progress != nil {
		progress(sdeUpdateStageChecking)
	}
	release, err := s.fetchLatestRelease()
	if err != nil {
		return "", fmt.Errorf("获取 GitHub Release 失败: %w", err)
	}

	version := release.TagName
	sdeUpdateState.lastSeenVersion = version
	global.Logger.Info("[SDE] 手动触发更新", zap.String("version", version))

	if err := s.doImport(release, progress); err != nil {
		return version, fmt.Errorf("导入 SDE 失败: %w", err)
	}

	// 如果版本已存在就更新，否则插入
	if progress != nil {
		progress(sdeUpdateStageRecording)
	}
	exists, err := s.repo.VersionExists(version)
	if err != nil {
		return version, fmt.Errorf("检查版本记录失败: %w", err)
	}
	if !exists {
		if err := s.repo.CreateVersion(&model.SdeVersion{
			Version: version,
			Note:    "manual import",
		}); err != nil {
			return version, fmt.Errorf("记录版本失败: %w", err)
		}
	}

	if progress != nil {
		progress(sdeUpdateStageDone)
	}
	global.Logger.Info("[SDE] 手动更新完成", zap.String("version", version))
	return version, nil
}

func (s *SdeService) getCurrentVersionString() (string, error) {
	current, err := s.GetCurrentVersion()
	if err != nil {
		return "", err
	}
	if current == nil {
		return "", nil
	}
	return current.Version, nil
}

func (s *SdeService) getStatusSnapshot() (SDEStatus, error) {
	raw, err := s.sysConfig.Get(model.SysConfigSDEStatus, "")
	if err != nil {
		return SDEStatus{}, err
	}
	if strings.TrimSpace(raw) == "" {
		return SDEStatus{}, nil
	}
	var status SDEStatus
	if err := json.Unmarshal([]byte(raw), &status); err != nil {
		global.Logger.Warn("[SDE] 状态快照解析失败，已回退默认值", zap.Error(err))
		return SDEStatus{}, nil
	}
	return status, nil
}

func (s *SdeService) setStatusSnapshot(status SDEStatus) error {
	data, err := json.Marshal(status)
	if err != nil {
		return err
	}
	return s.sysConfig.Set(model.SysConfigSDEStatus, string(data), "SDE 状态快照")
}

// doImport 找到 PostgreSQL SQL 资源并导入数据库
func (s *SdeService) doImport(release *githubRelease, progress func(stage string)) error {
	asset := pickSQLAsset(release.Assets)
	if asset == nil {
		return errors.New("未找到 PostgreSQL SQL 资源文件")
	}

	// 确保临时目录存在
	if err := os.MkdirAll(sdeDownloadDir, 0755); err != nil {
		return fmt.Errorf("创建临时目录失败: %w", err)
	}

	// 下载
	dlPath := filepath.Join(sdeDownloadDir, asset.Name)
	if progress != nil {
		progress(sdeUpdateStageDownloading)
	}
	global.Logger.Info("[SDE] 下载中", zap.String("url", asset.BrowserDownloadURL))
	if err := s.downloadFile(asset.BrowserDownloadURL, dlPath); err != nil {
		return fmt.Errorf("下载失败: %w", err)
	}
	if info, statErr := os.Stat(dlPath); statErr != nil {
		return fmt.Errorf("读取下载文件信息失败: %w", statErr)
	} else if info.Size() <= 0 {
		return errors.New("下载文件为空")
	}
	defer func() {
		if err := os.Remove(dlPath); err != nil {
			global.Logger.Warn("[SDE] 删除临时文件失败", zap.String("file", dlPath), zap.Error(err))
		}
	}()

	// 解压得到 .sql 文件路径
	if progress != nil {
		progress(sdeUpdateStageExtracting)
	}
	sqlPath, err := extractSQL(dlPath, sdeDownloadDir)
	if err != nil {
		return fmt.Errorf("解压失败: %w", err)
	}
	if !strings.HasSuffix(strings.ToLower(sqlPath), ".sql") {
		return fmt.Errorf("解压结果不是 SQL 文件: %s", sqlPath)
	}
	if info, statErr := os.Stat(sqlPath); statErr != nil {
		return fmt.Errorf("读取 SQL 文件信息失败: %w", statErr)
	} else if info.Size() <= 0 {
		return errors.New("解压后的 SQL 文件为空")
	}
	if sqlPath != dlPath {
		defer func() {
			if err := os.Remove(sqlPath); err != nil {
				global.Logger.Warn("[SDE] 删除临时文件失败", zap.String("file", sqlPath), zap.Error(err))
			}
		}()
	}

	// 导入到 PostgreSQL
	if progress != nil {
		progress(sdeUpdateStageImporting)
	}
	global.Logger.Info("[SDE] 导入数据库中", zap.String("file", sqlPath))
	if err := importSQL(sqlPath); err != nil {
		return fmt.Errorf("导入数据库失败: %w", err)
	}
	if err := s.verifySDEDataAvailability(); err != nil {
		return fmt.Errorf("SDE 导入后健康检查失败: %w", err)
	}

	return nil
}

// ---- 工具函数 ----

func (s *SdeService) newHTTPClientWithProxy(timeout time.Duration, useProxy bool) *http.Client {
	transport := &http.Transport{}
	if useProxy {
		if proxyAddr := s.getProxy(); proxyAddr != "" {
			if proxyURL, err := url.Parse(proxyAddr); err == nil {
				transport.Proxy = http.ProxyURL(proxyURL)
			} else {
				global.Logger.Warn("[SDE] 代理地址解析失败，将不使用代理", zap.String("proxy", proxyAddr), zap.Error(err))
			}
		}
	}
	return &http.Client{Timeout: timeout, Transport: transport}
}

func (s *SdeService) doRequestWithProxyFallback(timeout time.Duration, build func(*http.Client) (*http.Response, error)) (*http.Response, error) {
	client := s.newHTTPClientWithProxy(timeout, true)
	resp, err := build(client)
	if err == nil {
		return resp, nil
	}

	if !s.shouldRetryWithoutProxy(err) {
		return nil, err
	}

	global.Logger.Warn("[SDE] 代理请求失败，回退为直连重试", zap.Error(err))
	directClient := s.newHTTPClientWithProxy(timeout, false)
	return build(directClient)
}

func (s *SdeService) shouldRetryWithoutProxy(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "proxyconnect") ||
		strings.Contains(msg, "connect: connection refused") ||
		strings.Contains(msg, "socks")
}

// fetchLatestRelease 获取 GitHub 最新 release 信息
func (s *SdeService) fetchLatestRelease() (*githubRelease, error) {
	url := s.getDownloadURL()
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("build GitHub release request: %w", err)
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", sdeUserAgent)
	resp, err := s.doRequestWithProxyFallback(30*time.Second, func(client *http.Client) (*http.Response, error) {
		return client.Do(req.Clone(context.Background()))
	})
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GitHub API 返回非 200: %d", resp.StatusCode)
	}

	var release githubRelease
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return nil, err
	}
	release.TagName = strings.TrimSpace(release.TagName)
	if release.TagName == "" {
		return nil, errors.New("GitHub Release 缺少 tag_name")
	}
	if len(release.Assets) == 0 {
		return nil, errors.New("GitHub Release 未包含可下载资源")
	}
	if pickSQLAsset(release.Assets) == nil {
		return nil, errors.New("GitHub Release 缺少可识别的 SQL 资源")
	}
	return &release, nil
}

// downloadFile 下载文件到指定路径
func (s *SdeService) downloadFile(url, destPath string) error {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", sdeUserAgent)

	resp, err := s.doRequestWithProxyFallback(10*time.Minute, func(client *http.Client) (*http.Response, error) {
		return client.Do(req.Clone(context.Background()))
	})
	if err != nil {
		return err
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			global.Logger.Warn("[SDE] 关闭响应体失败", zap.Error(err))
		}
	}()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("下载返回非 200: %d", resp.StatusCode)
	}

	f, err := os.Create(destPath)
	if err != nil {
		return err
	}
	defer func() {
		if err := f.Close(); err != nil {
			global.Logger.Warn("[SDE] 关闭文件失败", zap.String("file", destPath), zap.Error(err))
		}
	}()

	_, err = io.Copy(f, resp.Body)
	return err
}

// extractSQL 通过魔数检测文件类型并递归解压，最终返回 .sql 文件路径
// 支持：plain SQL / gzip(.gz/.sql.gz) / zip（内含 .sql 或 .sql.gz）
func extractSQL(srcPath, destDir string) (string, error) {
	return extractSQLWithDepth(srcPath, destDir, 0)
}

func extractSQLWithDepth(srcPath, destDir string, depth int) (string, error) {
	if depth > maxExtractDepth {
		return "", fmt.Errorf("压缩递归层级过深（>%d）", maxExtractDepth)
	}

	magic, err := readMagicBytes(srcPath)
	if err != nil {
		return "", fmt.Errorf("读取文件魔数失败: %w", err)
	}

	switch {
	case isGzipMagic(magic):
		return extractGzip(srcPath, destDir, depth)
	case isBzip2Magic(magic):
		return extractBzip2(srcPath, destDir, depth)
	case isZipMagic(magic):
		return extractZip(srcPath, destDir, depth)
	default:
		// 当作纯 SQL 文件处理
		return srcPath, nil
	}
}

// readMagicBytes 读取文件头 4 字节
func readMagicBytes(path string) ([]byte, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer func() {
		if err := f.Close(); err != nil {
			global.Logger.Warn("[SDE] 关闭文件失败", zap.String("file", path), zap.Error(err))
		}
	}()
	buf := make([]byte, 4)
	n, _ := f.Read(buf)
	return buf[:n], nil
}

// isGzipMagic 判断是否为 gzip 格式（魔数 1f 8b）
func isGzipMagic(magic []byte) bool {
	return len(magic) >= 2 && magic[0] == 0x1f && magic[1] == 0x8b
}

// isBzip2Magic 判断是否为 bzip2 格式（魔数 42 5a 68 = "BZh"）
func isBzip2Magic(magic []byte) bool {
	return len(magic) >= 3 && magic[0] == 0x42 && magic[1] == 0x5a && magic[2] == 0x68
}

// isZipMagic 判断是否为 zip 格式（魔数 50 4b 03 04）
func isZipMagic(magic []byte) bool {
	return len(magic) >= 4 && magic[0] == 0x50 && magic[1] == 0x4b &&
		magic[2] == 0x03 && magic[3] == 0x04
}

// safeJoin 构造位于 destDir 下的安全路径，防止路径遍历
func safeJoin(destDir, name string) (string, error) {
	if name == "" {
		return "", fmt.Errorf("invalid empty file name in archive")
	}
	// 只保留最终文件名，丢弃任何目录部分
	base := filepath.Base(name)
	if base == "." || base == string(filepath.Separator) {
		return "", fmt.Errorf("invalid file name in archive: %q", name)
	}
	// 基于安全的 base 名构造输出路径
	outPath := filepath.Join(destDir, base)
	// 规范化并确保仍然位于目标目录下
	cleanDest := filepath.Clean(destDir)
	cleanOut := filepath.Clean(outPath)
	if !strings.HasPrefix(cleanOut+string(filepath.Separator), cleanDest+string(filepath.Separator)) {
		return "", fmt.Errorf("archive entry resolves outside destination directory: %q", name)
	}
	return outPath, nil
}

// extractGzip 解压 gzip 文件，输出文件名去掉 .gz 后缀
func extractGzip(srcPath, destDir string, depth int) (string, error) {
	f, err := os.Open(srcPath)
	if err != nil {
		return "", err
	}
	defer func() {
		if err := f.Close(); err != nil {
			global.Logger.Warn("[SDE] 关闭文件失败", zap.String("file", srcPath), zap.Error(err))
		}
	}()

	gr, err := gzip.NewReader(f)
	if err != nil {
		return "", err
	}
	defer func() {
		if err := gr.Close(); err != nil {
			global.Logger.Warn("[SDE] 关闭 gzip reader 失败", zap.Error(err))
		}
	}()

	// gzip 头中存有原始文件名
	outName := gr.Name
	if outName == "" {
		outName = strings.TrimSuffix(filepath.Base(srcPath), ".gz")
	}
	outPath, err := safeJoin(destDir, outName)
	if err != nil {
		return "", err
	}
	if _, statErr := os.Stat(outPath); statErr == nil {
		global.Logger.Warn("[SDE] 解压目标文件已存在，将覆盖", zap.String("file", outPath))
	}
	out, err := os.Create(outPath)
	if err != nil {
		return "", err
	}
	defer func() {
		if err := out.Close(); err != nil {
			global.Logger.Warn("[SDE] 关闭文件失败", zap.String("file", outPath), zap.Error(err))
		}
	}()

	if _, err = io.Copy(out, gr); err != nil {
		return "", err
	}

	// 解压结果可能还是压缩包，递归处理
	return extractSQLWithDepth(outPath, destDir, depth+1)
}

// extractBzip2 解压 bzip2 文件，输出文件名去掉 .bz2 后缀
func extractBzip2(srcPath, destDir string, depth int) (string, error) {
	f, err := os.Open(srcPath)
	if err != nil {
		return "", err
	}
	defer func() {
		if err := f.Close(); err != nil {
			global.Logger.Warn("[SDE] 关闭文件失败", zap.String("file", srcPath), zap.Error(err))
		}
	}()

	br := bzip2.NewReader(f)

	outName := strings.TrimSuffix(filepath.Base(srcPath), ".bz2")
	outPath, err := safeJoin(destDir, outName)
	if err != nil {
		return "", err
	}
	if _, statErr := os.Stat(outPath); statErr == nil {
		global.Logger.Warn("[SDE] 解压目标文件已存在，将覆盖", zap.String("file", outPath))
	}
	out, err := os.Create(outPath)
	if err != nil {
		return "", err
	}
	defer func() {
		if err := out.Close(); err != nil {
			global.Logger.Warn("[SDE] 关闭文件失败", zap.String("file", outPath), zap.Error(err))
		}
	}()

	if _, err = io.Copy(out, br); err != nil {
		return "", err
	}

	// 解压结果可能还是压缩包，递归处理
	return extractSQLWithDepth(outPath, destDir, depth+1)
}

// extractZip 解压 zip，找到第一个 SQL 相关条目（.sql 或 .sql.gz）并递归解压
func extractZip(srcPath, destDir string, depth int) (string, error) {
	r, err := zip.OpenReader(srcPath)
	if err != nil {
		return "", err
	}
	defer func() {
		if err := r.Close(); err != nil {
			global.Logger.Warn("[SDE] 关闭 zip reader 失败", zap.Error(err))
		}
	}()

	for _, f := range r.File {
		name := strings.ToLower(f.Name)
		if !strings.Contains(name, ".sql") {
			continue
		}
		outPath, err := safeJoin(destDir, f.Name)
		if err != nil {
			// 非法路径，跳过该条目继续寻找其他 SQL 文件
			continue
		}
		rc, err := f.Open()
		if err != nil {
			return "", err
		}

		if _, statErr := os.Stat(outPath); statErr == nil {
			global.Logger.Warn("[SDE] 解压目标文件已存在，将覆盖", zap.String("file", outPath))
		}
		out, err := os.Create(outPath)
		if err != nil {
			_ = rc.Close()
			return "", err
		}
		_, copyErr := io.Copy(out, rc)
		_ = out.Close()
		_ = rc.Close()
		if copyErr != nil {
			return "", copyErr
		}

		// 递归：内部文件可能还是 gzip
		return extractSQLWithDepth(outPath, destDir, depth+1)
	}
	return "", errors.New("zip 中未找到 SQL 相关文件")
}

// importSQL 读取 PostgreSQL SQL dump 文件并执行
// 上游 SQL 以多行 INSERT ... VALUES (...) 形式组织大量数据（尤其是 trnTranslations）。
// 为避免超大单条 INSERT 导致执行失败或导入不完整，这里会按行流式拆分为较小块执行。
//
// 执行策略：
//  1. 使用专用连接，确保 session_replication_role 全程生效
//  2. DDL（CREATE/DROP/ALTER/TRUNCATE/SET）自动提交当前 DML 事务后立即执行
//  3. DML 每 batchSize 条提交一次事务
//  4. 多行 INSERT ... VALUES 每 insertChunkSize 行拆分为独立 INSERT 执行
//  5. DML 失败视为致命错误，立即中止，避免产生“成功但数据不完整”的导入结果
const (
	batchSize       = 200
	insertChunkSize = 1000
)

func importSQL(sqlPath string) error {
	sqlDB, err := global.DB.DB()
	if err != nil {
		return fmt.Errorf("获取数据库连接失败: %w", err)
	}

	// 占用一条专用连接，保证 session 级设置全程有效
	conn, err := sqlDB.Conn(context.Background())
	if err != nil {
		return fmt.Errorf("获取专用连接失败: %w", err)
	}
	defer func() {
		if err := conn.Close(); err != nil {
			global.Logger.Warn("[SDE] 关闭数据库连接失败", zap.Error(err))
		}
	}()

	// 禁用触发器/外键约束检查，关闭同步提交以提速
	for _, pragma := range []string{
		"SET session_replication_role = 'replica'",
		"SET synchronous_commit = OFF",
	} {
		if _, e := conn.ExecContext(context.Background(), pragma); e != nil {
			global.Logger.Warn("[SDE] 设置优化参数失败", zap.String("sql", pragma), zap.Error(e))
		}
	}
	defer func() {
		_, _ = conn.ExecContext(context.Background(), "SET session_replication_role = 'origin'")
		_, _ = conn.ExecContext(context.Background(), "SET synchronous_commit = ON")
	}()

	f, err := os.Open(sqlPath)
	if err != nil {
		return err
	}
	defer func() {
		if err := f.Close(); err != nil {
			global.Logger.Warn("[SDE] 关闭文件失败", zap.Error(err))
		}
	}()

	reader := bufio.NewReaderSize(f, 1024*1024)
	var stmtCount, ddlErrCount int

	// 开启第一个事务
	tx, err := conn.BeginTx(context.Background(), nil)
	if err != nil {
		return fmt.Errorf("开启事务失败: %w", err)
	}
	txCount := 0

	// commitTx 提交当前事务并立即开启下一个
	commitTx := func() {
		if err := tx.Commit(); err != nil {
			global.Logger.Warn("[SDE] 事务提交失败，尝试回滚", zap.Error(err))
			_ = tx.Rollback()
		}
		var beginErr error
		tx, beginErr = conn.BeginTx(context.Background(), nil)
		if beginErr != nil {
			global.Logger.Error("[SDE] 开启事务失败", zap.Error(beginErr))
		}
		txCount = 0
	}

	// commitFinalTx 仅提交最后一个事务，不再开启新事务（避免 conn.Close 时遗留 dangling transaction 阻塞连接池）
	commitFinalTx := func() {
		if err := tx.Commit(); err != nil {
			global.Logger.Warn("[SDE] 最终事务提交失败，尝试回滚", zap.Error(err))
			_ = tx.Rollback()
		}
	}

	// rollbackTx 回滚当前事务并立即开启下一个（DML 失败时调用）
	rollbackTx := func() {
		_ = tx.Rollback()
		var beginErr error
		tx, beginErr = conn.BeginTx(context.Background(), nil)
		if beginErr != nil {
			global.Logger.Error("[SDE] 开启事务失败", zap.Error(beginErr))
		}
		txCount = 0
	}

	isDDL := func(upper string) bool {
		for _, kw := range []string{"CREATE ", "DROP ", "ALTER ", "TRUNCATE ", "SET "} {
			if strings.HasPrefix(upper, kw) {
				return true
			}
		}
		return false
	}

	isCriticalDDL := func(upper string) bool {
		for _, kw := range []string{"CREATE ", "DROP ", "ALTER ", "TRUNCATE "} {
			if strings.HasPrefix(upper, kw) {
				return true
			}
		}
		return false
	}

	execDDL := func(sql string) error {
		commitTx()
		upper := strings.ToUpper(strings.TrimSpace(sql))
		if _, execErr := conn.ExecContext(context.Background(), sql); execErr != nil {
			ddlErrCount++
			if isCriticalDDL(upper) {
				return fmt.Errorf("关键 DDL 执行失败: %w", execErr)
			}
			global.Logger.Warn("[SDE] 非关键 DDL 执行失败，已跳过",
				zap.String("err", execErr.Error()),
				zap.String("sql_prefix", truncate(sql, 120)))
			return nil
		}
		stmtCount++
		return nil
	}

	execDML := func(sql string, table string, chunkIndex int, rowOffset int) error {
		if _, execErr := tx.Exec(sql); execErr != nil {
			global.Logger.Error("[SDE] DML 执行失败，导入终止",
				zap.String("err", execErr.Error()),
				zap.String("table", table),
				zap.Int("chunk_index", chunkIndex),
				zap.Int("row_offset", rowOffset),
				zap.String("sql_prefix", truncate(sql, 160)))
			rollbackTx()
			return execErr
		}

		stmtCount++
		txCount++
		if txCount >= batchSize {
			commitTx()
		}
		return nil
	}

	stmts, err := splitSQLStatements(reader)
	if err != nil {
		return err
	}

	for _, sql := range stmts {
		trimmed := strings.TrimSpace(sql)
		if trimmed == "" || trimmed == ";" {
			continue
		}

		upper := strings.ToUpper(trimmed)
		if upper == "BEGIN" || upper == "COMMIT" || upper == "ROLLBACK" {
			continue
		}
		if isDDL(upper) {
			if err := execDDL(trimmed); err != nil {
				return err
			}
			continue
		}

		header, table, rows, ok := parseInsertValuesStatement(trimmed)
		if !ok || len(rows) == 0 {
			if err := execDML(trimmed, "", 0, 0); err != nil {
				return fmt.Errorf("执行 SQL 失败: %w", err)
			}
			continue
		}

		for offset := 0; offset < len(rows); offset += insertChunkSize {
			end := offset + insertChunkSize
			if end > len(rows) {
				end = len(rows)
			}
			chunk := header + strings.Join(rows[offset:end], ",") + ";"
			if err := execDML(chunk, table, offset/insertChunkSize+1, offset+1); err != nil {
				return fmt.Errorf("执行批量 INSERT 失败: %w", err)
			}
		}
	}

	// 提交剩余事务（最后一次不再开启新事务）
	commitFinalTx()

	global.Logger.Info("[SDE] SQL 导入完成",
		zap.Int("成功语句数", stmtCount),
		zap.Int("DDL失败语句数", ddlErrCount))
	return nil
}

func (s *SdeService) verifySDEDataAvailability() error {
	for _, table := range []string{"invTypes", "invGroups", "trnTranslations"} {
		if err := ensureSDETableReadable(table); err != nil {
			return fmt.Errorf("关键表 %s 不可用: %w", table, err)
		}
	}
	if err := ensurePublishedFilterQueryable(); err != nil {
		return err
	}
	return nil
}

func ensureSDETableReadable(camelName string) error {
	lowerName := strings.ToLower(camelName)
	queries := []string{
		fmt.Sprintf(`SELECT 1 FROM "%s" LIMIT 1`, camelName),
		fmt.Sprintf(`SELECT 1 FROM %s LIMIT 1`, lowerName),
	}

	var lastErr error
	for _, query := range queries {
		var probe int
		if err := global.DB.Raw(query).Scan(&probe).Error; err == nil {
			return nil
		} else {
			lastErr = err
		}
	}
	if lastErr == nil {
		lastErr = errors.New("unknown table read error")
	}
	return lastErr
}

func ensurePublishedFilterQueryable() error {
	queries := []string{
		`SELECT 1 FROM "invTypes" WHERE "published" = true LIMIT 1`,
		`SELECT 1 FROM invtypes WHERE published = true LIMIT 1`,
	}

	var lastErr error
	for _, query := range queries {
		var probe int
		if err := global.DB.Raw(query).Scan(&probe).Error; err == nil {
			return nil
		} else {
			lastErr = err
		}
	}
	if lastErr == nil {
		lastErr = errors.New("published filter check failed")
	}
	return fmt.Errorf("published 过滤不可执行: %w", lastErr)
}

func splitSQLStatements(r *bufio.Reader) ([]string, error) {
	var out []string
	var sb strings.Builder
	inSingleQuote := false
	for {
		ch, _, err := r.ReadRune()
		if err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			return nil, fmt.Errorf("读取 SQL 文件失败: %w", err)
		}

		sb.WriteRune(ch)
		if ch == '\'' {
			if inSingleQuote {
				next, _, nextErr := r.ReadRune()
				if nextErr != nil && !errors.Is(nextErr, io.EOF) {
					return nil, fmt.Errorf("读取 SQL 文件失败: %w", nextErr)
				}
				if nextErr == nil {
					if next == '\'' {
						sb.WriteRune(next)
						continue
					}
					if unreadErr := r.UnreadRune(); unreadErr != nil {
						return nil, fmt.Errorf("读取 SQL 文件失败: %w", unreadErr)
					}
				}
				inSingleQuote = false
				continue
			}
			inSingleQuote = true
			continue
		}

		if !inSingleQuote && ch == ';' {
			out = append(out, sb.String())
			sb.Reset()
		}
	}

	if inSingleQuote {
		return nil, errors.New("SQL dump 包含未闭合的字符串")
	}
	if strings.TrimSpace(sb.String()) != "" {
		out = append(out, sb.String())
	}
	return out, nil
}

func parseInsertValuesStatement(sql string) (string, string, []string, bool) {
	upper := strings.ToUpper(sql)
	if !strings.HasPrefix(upper, "INSERT INTO ") {
		return "", "", nil, false
	}
	valuesIdx := strings.Index(upper, " VALUES ")
	if valuesIdx <= 0 {
		return "", "", nil, false
	}

	header := sql[:valuesIdx+8]
	body := strings.TrimSpace(sql[valuesIdx+8:])
	body = strings.TrimSuffix(body, ";")
	table := parseInsertTableName(sql[:valuesIdx])

	rows := make([]string, 0, 64)
	inSingleQuote := false
	depth := 0
	rowStart := -1
	for i := 0; i < len(body); i++ {
		ch := body[i]
		if ch == '\'' {
			if inSingleQuote {
				if i+1 < len(body) && body[i+1] == '\'' {
					i++
					continue
				}
				inSingleQuote = false
			} else {
				inSingleQuote = true
			}
			continue
		}
		if inSingleQuote {
			continue
		}

		switch ch {
		case '(':
			depth++
			if depth == 1 {
				rowStart = i
			}
		case ')':
			if depth == 0 {
				return "", "", nil, false
			}
			depth--
			if depth == 0 && rowStart >= 0 {
				rows = append(rows, strings.TrimSpace(body[rowStart:i+1]))
				rowStart = -1
			}
		}
	}
	if inSingleQuote || depth != 0 || len(rows) == 0 {
		return "", "", nil, false
	}
	return header, table, rows, true
}

func parseInsertTableName(prefix string) string {
	trimmed := strings.TrimSpace(prefix)
	if len(trimmed) < len("INSERT INTO ") {
		return ""
	}
	rest := strings.TrimSpace(trimmed[len("INSERT INTO "):])
	if rest == "" {
		return ""
	}
	if rest[0] == '"' {
		end := strings.Index(rest[1:], "\"")
		if end >= 0 {
			return rest[:end+2]
		}
	}
	for i := 0; i < len(rest); i++ {
		if rest[i] == '(' || rest[i] == ' ' || rest[i] == '\t' || rest[i] == '\n' || rest[i] == '\r' {
			return rest[:i]
		}
	}
	return rest
}

// truncate 截断字符串，用于日志输出
func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}

// ---- 数据查询 ----

// GetTypes 批量查询物品信息（含 group + category + market_group 翻译）
func (s *SdeService) GetTypes(typeIDs []int, published *bool, languageID string) ([]TypeInfo, error) {
	return s.repo.GetTypes(typeIDs, published, languageID)
}

// GetNames 批量查询按 namespace 分组的 id -> name 映射（仅查数据库翻译表）
func (s *SdeService) GetNames(ids map[string][]int, languageID string) (SdeNameMap, error) {
	return s.repo.GetNames(ids, languageID)
}

// FuzzySearch 模糊搜索物品/成员名称
func (s *SdeService) FuzzySearch(keyword string, languageID string, categoryIDs []int, excludeCategoryIDs []int, limit int, searchMember bool) ([]FuzzySearchItem, error) {
	return s.repo.FuzzySearch(keyword, languageID, categoryIDs, excludeCategoryIDs, limit, searchMember)
}

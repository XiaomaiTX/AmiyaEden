package repository

import (
	"amiya-eden/internal/model"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"amiya-eden/global"

	"go.uber.org/zap"
)

var sdeTranslationCategoryIDs = map[string]int{
	"type":          8,
	"group":         7,
	"category":      6,
	"description":   33,
	"tech":          34,
	"market_group":  36,
	"solar_system":  40,
	"constellation": 41,
	"region":        42,
}

// SdeNameMap 按 namespace 分组的名称映射。
type SdeNameMap map[string]map[int]string

// SdeRepository SDE 数据访问层
type SdeRepository struct{}

func NewSdeRepository() *SdeRepository { return &SdeRepository{} }

const (
	sdeQueryErrorSource         = "sde_repository"
	sdeQueryErrorThrottleWindow = 60 * time.Second
)

var sdeQueryErrorState = struct {
	mu       sync.Mutex
	lastSeen map[string]time.Time
}{
	lastSeen: make(map[string]time.Time),
}

type sdeNaming struct {
	camelCase bool
}

func newSDENaming(camelCase bool) sdeNaming {
	return sdeNaming{camelCase: camelCase}
}

func (n sdeNaming) table(base string, alias ...string) string {
	name := base
	if !n.camelCase {
		name = strings.ToLower(base)
	}

	if n.camelCase {
		if len(alias) == 0 || alias[0] == "" {
			return fmt.Sprintf(`"%s"`, name)
		}
		return fmt.Sprintf(`"%s" %s`, name, alias[0])
	}

	if len(alias) == 0 || alias[0] == "" {
		return name
	}
	return fmt.Sprintf(`%s %s`, name, alias[0])
}

func (n sdeNaming) col(alias string, name string) string {
	if n.camelCase {
		return fmt.Sprintf(`%s."%s"`, alias, name)
	}
	return fmt.Sprintf(`%s.%s`, alias, strings.ToLower(name))
}

func (n sdeNaming) bareCol(name string) string {
	if n.camelCase {
		return fmt.Sprintf(`"%s"`, name)
	}
	return strings.ToLower(name)
}

func reportSDEQueryError(operation string, err error) {
	if err == nil {
		return
	}
	msg := strings.TrimSpace(err.Error())
	if msg == "" {
		return
	}

	source := sdeQueryErrorSource
	if op := strings.TrimSpace(operation); op != "" {
		source = source + "." + op
	}

	signature := source + "|" + msg
	now := time.Now()
	sdeQueryErrorState.mu.Lock()
	if last, exists := sdeQueryErrorState.lastSeen[signature]; exists && now.Sub(last) < sdeQueryErrorThrottleWindow {
		sdeQueryErrorState.mu.Unlock()
		return
	}
	sdeQueryErrorState.lastSeen[signature] = now
	sdeQueryErrorState.mu.Unlock()

	if global.Logger != nil {
		global.Logger.Warn("[SDE] 仓库查询失败",
			zap.String("source", source),
			zap.Error(err))
	}

	repo := NewSysConfigRepository()
	raw, getErr := repo.Get(model.SysConfigSDEStatus, "")
	if getErr != nil {
		if global.Logger != nil {
			global.Logger.Warn("[SDE] 读取状态快照失败", zap.Error(getErr))
		}
		return
	}

	status := map[string]interface{}{}
	if strings.TrimSpace(raw) != "" {
		if unmarshalErr := json.Unmarshal([]byte(raw), &status); unmarshalErr != nil {
			if global.Logger != nil {
				global.Logger.Warn("[SDE] 状态快照解析失败，改用空快照写入查询错误", zap.Error(unmarshalErr))
			}
			status = map[string]interface{}{}
		}
	}

	status["last_query_error"] = msg
	status["last_query_error_at"] = now.Unix()
	status["last_query_error_source"] = source

	data, marshalErr := json.Marshal(status)
	if marshalErr != nil {
		if global.Logger != nil {
			global.Logger.Warn("[SDE] 状态快照序列化失败", zap.Error(marshalErr))
		}
		return
	}
	if setErr := repo.Set(model.SysConfigSDEStatus, string(data), "SDE 状态快照"); setErr != nil {
		if global.Logger != nil {
			global.Logger.Warn("[SDE] 保存状态快照失败", zap.Error(setErr))
		}
	}
}

func wrapAndReportSDEFallbackError(operation string, primaryErr, fallbackErr error) error {
	finalErr := wrapSDEFallbackError(primaryErr, fallbackErr)
	if finalErr != nil {
		reportSDEQueryError(operation, finalErr)
	}
	return finalErr
}

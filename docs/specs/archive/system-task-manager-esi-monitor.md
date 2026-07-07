---
status: completed
doc_type: completed
owner: engineering
last_reviewed: 2026-07-07
completed: 2026-07-07
source_of_truth:
  - docs/features/current/task-manager.md
  - server/internal/handler/esi_refresh.go
  - server/pkg/eve/esi/queue.go
  - static/src/views/system/task-manager/modules/EsiMonitorTab.vue
  - static/src/views/system/task-manager/modules/EsiStatusesTab.vue
  - static/src/views/system/task-manager/index.vue
---

# System Task Manager ESI 监控页方案（已落地）

> 本方案已落地实现，当前行为以 `docs/features/current/task-manager.md` 为准。本文件保留为历史设计记录，不涉及 `docs/specs/draft/esi-refresh-intervals.md` 的改动。

## 背景与目标

- 当前 `/system/task-manager` 已有：
  - ESI 任务控制（全量触发、按任务触发）
  - ESI 状态明细表（按任务/人物查看）
- 现状缺口：缺少运维视角的聚合监控，无法快速回答“现在是否健康、哪里最危险、该先处理什么”。
- 目标：在同页面新增 ESI 监控 Tab，提供面向运维的健康概览、异常排行与超期风险面板。

## 范围

- 仅覆盖 `static/` Vue 页面。
- 在 `/system/task-manager` 内新增 `ESI 监控` Tab，不新增路由。
- 新增后端汇总接口 `GET /api/v1/tasks/esi/monitor`（`admin` 权限）。
- 不覆盖 `static-react`。

## 代码文件级实施清单

### 后端（Go）

| 文件 | 变更点 | 目标 |
| --- | --- | --- |
| `server/internal/handler/esi_refresh.go` | 新增 `GetMonitor` handler；新增监控响应结构体（`MonitorOverview`、`MonitorTaskPanelItem`、`MonitorFailureItem`、`MonitorOverdueItem`、`MonitorResponse`）；抽取复用人物名补全逻辑 | 输出聚合监控接口数据 |
| `server/internal/router/router.go` | 在 `esiTasks` 分组新增 `GET /monitor` 路由绑定 `esiH.GetMonitor` | 对外暴露新接口 |
| `server/internal/handler/esi_refresh_handler_test.go` | 新增 `GetMonitor` 空集、混合状态、排序测试 | 保证聚合口径与稳定性 |
| `server/internal/router/router_test.go` | 新增路由存在性断言：`/api/v1/tasks/esi/monitor` | 防止漏挂路由 |

### 前端（Vue + TS）

| 文件 | 变更点 | 目标 |
| --- | --- | --- |
| `static/src/views/system/task-manager/index.vue` | 新增 `ESI 监控` Tab（`name: esi-monitor`）；挂载 `EsiMonitorTab` 组件 | 页面入口 |
| `static/src/views/system/task-manager/modules/EsiMonitorTab.vue` | 新增监控页模块（指标卡、任务健康表、失败 Top、超期 Top、手动刷新、30 秒自动刷新、错误态） | 运维聚合面板 |
| `static/src/api/esi-refresh.ts` | 新增 `fetchESIRefreshMonitor()` 包装，调用 `/api/v1/tasks/esi/monitor` | 前端访问接口 |
| `static/src/types/api/api.d.ts` | 在 `Api.ESIRefresh` 下新增监控返回类型定义 | 保持类型契约同步 |
| `static/src/locales/langs/zh.json` | 增加 `taskManager.tabs.esiMonitor` 及监控面板文案键 | 中文文案 |
| `static/src/locales/langs/en.json` | 同步英文文案键 | 英文文案 |
| `static/src/views/system/task-manager/index.test.ts` | 增加新 Tab、新模块、新 API 包装断言 | 前端回归保护 |

### 文档与路由索引

| 文件 | 变更点 | 目标 |
| --- | --- | --- |
| `docs/api/route-index.md` | 在 ESI Task Operations 中新增 `GET /tasks/esi/monitor` | API 文档对齐 |
| `docs/features/current/task-manager.md` | 实现完成后补充“ESI 监控 Tab”能力说明与数据口径 | 功能文档转正 |

## 接口设计

### 路由

- `GET /api/v1/tasks/esi/monitor`
- 权限：`RequireRole(admin)`（与 `/api/v1/tasks/esi/*` 一致）

### 响应结构（草案）

```json
{
  "generated_at": "2026-05-10T15:04:05Z",
  "overview": {
    "total": 0,
    "healthy": 0,
    "warning": 0,
    "critical": 0,
    "running": 0,
    "failed": 0,
    "overdue": 0
  },
  "task_panels": [
    {
      "task_name": "character_skill",
      "description": "人物技能信息",
      "priority": 50,
      "total": 0,
      "healthy": 0,
      "warning": 0,
      "critical": 0,
      "running": 0,
      "failed": 0,
      "overdue": 0,
      "success_rate": 0,
      "worst_lag_seconds": 0
    }
  ],
  "failure_top": [
    {
      "task_name": "character_wallet",
      "description": "人物钱包信息",
      "character_id": 9001,
      "character_name": "Amiya Main",
      "error": "token expired",
      "last_run": "2026-05-10T14:20:00Z"
    }
  ],
  "overdue_top": [
    {
      "task_name": "character_skill",
      "description": "人物技能信息",
      "character_id": 9002,
      "character_name": "Kal'tsit Alt",
      "next_run": "2026-05-10T12:00:00Z",
      "overdue_seconds": 7200
    }
  ]
}
```

### 计算口径

- 数据源：`queue.GetAllStatuses()` 的运行态快照。
- 角色名称：沿用当前 `GetStatuses` 的角色仓库补全逻辑。
- `overdue_seconds`：`now - next_run`（仅当 `next_run < now`）。
- `success_rate`：
  - 分母：该任务下状态项总数（`total`）
  - 分子：`status == success` 数量
  - 返回范围 `0~1` 浮点值。

## 后端函数级设计

### `server/internal/handler/esi_refresh.go`

- 新增导出方法：
  - `func (h *ESIRefreshHandler) GetMonitor(c *gin.Context)`
- 新增内部辅助函数（命名可微调，但职责需一致）：
  - `buildCharacterNameMap(statuses []*esi.TaskStatus) map[int64]string`
  - `expectedIntervalSeconds(taskName string) int64`
  - `classifySeverity(status *esi.TaskStatus, now time.Time, expectedIntervalSeconds int64) string`
  - `buildMonitorResponse(statuses []*esi.TaskStatus, characterNames map[int64]string, now time.Time) MonitorResponse`

### `GetMonitor` 流程（必须项）

1. 从 `jobs.GetESIQueue()` 获取队列；若为空返回零值监控响应，不报错。
2. 拉取 `queue.GetAllStatuses()` 快照并补齐 `character_name`。
3. 逐条状态计算：
   - `overdue_seconds`（仅 `next_run` 早于当前时间）
   - 分层等级（`healthy/warning/critical`）
4. 聚合输出：
   - `overview`
   - `task_panels`（按任务名聚合）
   - `failure_top`（按 `last_run` 逆序，空时间排后）
   - `overdue_top`（按 `overdue_seconds` 降序）
5. 对结果做稳定排序：
   - `task_panels`: `critical desc` -> `failed desc` -> `overdue desc` -> `priority asc` -> `task_name asc`
   - `failure_top`: `last_run desc` -> `task_name asc` -> `character_id asc`
   - `overdue_top`: `overdue_seconds desc` -> `task_name asc` -> `character_id asc`
6. 使用 `response.OK(c, monitorResponse)` 返回。

## 前端模块级设计

### `static/src/views/system/task-manager/index.vue`

- 新增 Tab：
  - `label: t('taskManager.tabs.esiMonitor')`
  - `name: 'esi-monitor'`
- 新增组件：
  - `EsiMonitorTab v-if="activeTab === 'esi-monitor'"`

### `static/src/views/system/task-manager/modules/EsiMonitorTab.vue`

- 数据状态：
  - `monitorData`（接口返回）
  - `loading`、`error`
  - `autoRefreshEnabled`（默认 true）
  - `refreshTimer`（30 秒）
- 生命周期：
  - `onMounted` 首次拉取并启动定时器
  - `onBeforeUnmount` 清理定时器
  - `visibilitychange` 页面隐藏暂停、可见恢复
- 展示区块：
  - 指标卡（overview）
  - 任务健康表（task_panels）
  - 失败列表（failure_top）
  - 超期列表（overdue_top）
- 交互：
  - 手动刷新按钮
  - 自动刷新开关
  - 请求失败时提示并保留最后一次成功数据（若有）

### `static/src/api/esi-refresh.ts`

- 新增：
  - `export function fetchESIRefreshMonitor()`
  - 返回类型：`Api.ESIRefresh.MonitorResponse`
  - URL：`/api/v1/tasks/esi/monitor`

### `static/src/types/api/api.d.ts`

- 在 `namespace ESIRefresh` 下新增：
  - `interface MonitorOverview`
  - `interface MonitorTaskPanelItem`
  - `interface MonitorFailureItem`
  - `interface MonitorOverdueItem`
  - `interface MonitorResponse`

## 字段契约补充（实现约束）

- `generated_at`：UTC RFC3339 字符串。
- `success_rate`：保留 `0~1` 浮点值，不在后端转换百分比字符串。
- `worst_lag_seconds`：任务维度内最大超期秒数，无超期时为 `0`。
- `error`：空字符串不进入 `failure_top`。
- `failure_top`、`overdue_top` 建议固定上限 `20` 条，防止首屏过载。

## i18n 键清单（建议）

- `taskManager.tabs.esiMonitor`
- `taskManager.esi.monitor.generatedAt`
- `taskManager.esi.monitor.autoRefresh`
- `taskManager.esi.monitor.sections.overview`
- `taskManager.esi.monitor.sections.taskPanels`
- `taskManager.esi.monitor.sections.failureTop`
- `taskManager.esi.monitor.sections.overdueTop`
- `taskManager.esi.monitor.cards.total`
- `taskManager.esi.monitor.cards.healthy`
- `taskManager.esi.monitor.cards.warning`
- `taskManager.esi.monitor.cards.critical`
- `taskManager.esi.monitor.cards.running`
- `taskManager.esi.monitor.cards.failed`
- `taskManager.esi.monitor.cards.overdue`
- `taskManager.esi.monitor.columns.successRate`
- `taskManager.esi.monitor.columns.worstLag`
- `taskManager.esi.monitor.columns.overdueSeconds`
- `taskManager.esi.monitor.messages.loadFailed`

## 指标与阈值模型（固定阈值）

首版不做配置化阈值，固定规则如下：

- `critical`
  - `status == failed`
  - 或已超期且超期时长达到该任务刷新周期（`overdue_seconds >= expected_interval_seconds`）
- `warning`
  - `status == running` 或 `status == pending`
  - 或已超期但未达到 `critical` 门槛
- `healthy`
  - 其余状态（例如 `success`、`skipped` 且未超期）

说明：

- `expected_interval_seconds` 基于任务当前周期口径（活跃/非活跃区间）推导；若无法判定，降级使用任务活跃间隔。
- 阈值判定只用于监控分层，不改变任务调度逻辑。

## 前端交互设计（`/system/task-manager`）

新增 `ESI 监控` Tab，页面包含：

- 指标卡区：
  - 总监控项
  - 健康 / 警告 / 严重
  - 运行中
  - 失败
  - 超期
- 任务健康表：
  - 按任务聚合展示 `total/failed/overdue/running/success_rate/worst_lag`
  - 默认按风险排序（`critical` 优先，再看失败数与超期时长）
- 异常面板：
  - `failure_top`（失败明细）
  - `overdue_top`（超期明细）
- 刷新机制：
  - 手动刷新按钮
  - 30 秒自动刷新
  - 页面隐藏或离开 Tab 时暂停自动刷新
- 错误态：
  - 接口失败显示错误提示与空状态，不影响现有 `ESI 状态` Tab。

## 兼容性与关联

- 本草案是新增能力，不改变现有 `/api/v1/tasks/esi/statuses` 的返回结构与筛选语义。
- 现有 ESI 触发入口（`run` / `run-task` / `run-all`）保持不变。
- 不引入新的持久化表与迁移。

## 实施边界

- 不做历史趋势存储与时序图。
- 不做可配置阈值（首版内置常量）。
- 不新增路由，保持在 `task-manager` 页面内。

## 验收标准

- 后端返回的 `overview/task_panels/failure_top/overdue_top` 统计口径一致且可复现。
- 任务健康表排序稳定，异常项优先可见。
- 当接口错误或空数据时，页面展示明确错误/空态，不出现崩溃或假数据。
- API 类型声明与 `zh/en` 文案键同步补齐。

## 测试计划

- 后端
  - `GET /api/v1/tasks/esi/monitor` 空集聚合测试。
  - 混合状态（`success/running/failed/overdue`）聚合与分层测试。
  - `failure_top`、`overdue_top` 排序测试。
  - 队列未初始化时返回零值结构测试。
- 前端
  - 新 Tab 渲染与切换测试。
  - 监控面板渲染测试（指标卡、任务表、异常面板）。
  - 自动刷新与暂停逻辑测试。
  - 错误态提示测试。
  - `index.test.ts` 追加断言：
    - `index.vue` 包含 `esi-monitor` Tab 与 `EsiMonitorTab` 挂载
    - 新模块文件存在并调用 `fetchESIRefreshMonitor`
    - `esi-refresh.ts` 包含 `/api/v1/tasks/esi/monitor`
    - `zh/en` 包含 `taskManager.tabs.esiMonitor` 与监控区块键
- 文档与契约
  - API 类型声明、i18n 键存在性校验。

## 假设

- 当前 ESI 监控以运行态快照为主，不引入新存储表。
- 超期判断基于 `next_run` 与任务周期对比。
- 本草案作为实现输入，能力落地后再迁移到 `docs/features/current/`。

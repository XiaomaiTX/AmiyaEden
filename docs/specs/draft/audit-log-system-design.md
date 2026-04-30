---
status: draft
doc_type: draft
owner: engineering
last_reviewed: 2026-04-29
source_of_truth:
  - server/internal/model/audit_event.go
  - server/internal/repository/audit_event.go
  - server/internal/service/audit_service.go
  - server/internal/handler/audit_event.go
  - server/internal/middleware/operation_log.go
  - server/internal/model/operation_log.go
  - server/internal/model/sys_wallet.go
  - server/internal/service/sys_wallet.go
  - server/internal/service/role.go
  - server/internal/repository/sys_wallet.go
  - server/internal/router/router.go
  - static/src/views/system/wallet
  - static/src/api/sys-wallet.ts
  - static/src/types/api/api.d.ts
  - docs/features/current/commerce.md
  - docs/architecture/auth-and-permissions.md
---

# 审计日志系统方案（权限操作 + 伏羲币变动 + 运维查询导出）

## 当前状态

- 已实现：
  - 请求级操作日志：`operation_log`（中间件自动记录请求轨迹）
  - 伏羲币账本日志：`wallet_transaction`
  - 管理员钱包调整日志：`wallet_log`
  - 系统钱包管理页具备日志/流水查询与分析能力（`/system/wallet`）
  - 统一审计事件模型：`audit_event`（已接入 AutoMigrate 与关键索引）
  - 权限变更审计：`RoleService.SetUserRoles`（成功/失败均写入）
  - 伏羲币审计：`AdminAdjust`、`ApplyWalletDeltaByOperatorTx`
  - 运维在线查询接口：`POST /api/v1/system/audit/events`
- 未实现：
  - 跨模块审计覆盖扩展（配置、审批、任务运维等）
  - 审计导出任务（`POST /system/audit/export`、`GET /system/audit/export/:task_id`）
  - 审计前端管理页与下载面板

## 背景

当前系统已经有三类日志，但职责分散：

- `operation_log` 更偏请求轨迹，不含完整业务语义（如 before/after）
- `wallet_transaction` 是资金账本事实，不等同于操作审计
- `wallet_log` 只覆盖管理员手动调账，不覆盖所有伏羲币变动链路

这会导致审计工作在以下场景成本高：

- 权限分配、自动权限映射变更的追溯
- 多模块伏羲币收支（PAP/SRP/福利/商城/新人帮扶）统一稽核
- 运维按条件筛查并导出证据链

## 提案内容

### 1. 新增统一审计事件表 `audit_event`

新增统一审计事实表，作为业务审计主入口，不替代原账本表：

- 核心字段：
  - `event_id`（唯一键）
  - `occurred_at`
  - `category`（`permission` / `fuxi_wallet` / `config` / `approval` / `task_ops` / `security`）
  - `action`（具体动作编码）
  - `actor_user_id`
  - `target_user_id`
  - `resource_type`、`resource_id`
  - `result`（`success` / `failed`）
  - `request_id`、`ip`、`user_agent`
  - `details_json`（before/after、原因、补充上下文）

### 2. 一期接入范围（分阶段策略）

一期先覆盖高风险与高频审计面：

- 权限操作
  - 用户职权分配：`RoleService.SetUserRoles`
  - 自动权限映射规则变更：`esi_role_mapping`、`esi_title_mapping` 增删改
- 伏羲币操作
  - 管理员余额调整：`SysWalletService.AdminAdjust`
  - 统一钱包差量入口：`ApplyWalletDelta* / CreditUser / DebitUser`（跨 PAP、SRP、福利、商城、新人帮扶）
- 审批与运维动作
  - 关键审批：福利发放/拒绝、SRP 发放、商城订单发放/拒绝
  - 手动任务执行、调度变更（启停/cron 更新）
  - 关键配置变更（`system_config`）

### 3. 在线审计查询能力（系统内）

新增统一管理端审计页面与查询接口，支持运维排查：

- 筛选条件：
  - 时间范围
  - 分类（category）
  - 动作（action）
  - 操作者 / 目标用户
  - 结果状态
  - `request_id`
  - 资源 ID
  - 关键词（reason/details）
- 查询能力：
  - 分页、稳定排序
  - 一键按 `request_id` 精确检索
  - 关联跳转（`operation_log` / `wallet_transaction`）

### 4. 在线导出能力（下载）

提供异步导出任务：

- 支持 `CSV` 与 `JSON`
- 导出“当前筛选结果”
- 提供导出任务状态查询（创建中 / 处理中 / 完成 / 失败 / 过期）
- 导出记录包含：导出人、导出时间、筛选条件摘要
- 导出行为本身写入 `audit_event`（防止“谁导出了什么”不可追溯）

### 5. 生命周期与可持续运维

- 保留策略：在线 90 天 + 归档 1 年
- 每日归档任务：将超 90 天日志导出归档存储并清理在线库
- 归档任务要求幂等，可重试
- 监控告警：
  - 审计写入失败率
  - 导出任务失败率
  - 归档任务失败率
  - 异常事件突增（如高频权限变更、高频调账）

## 实施清单（具体修改点）

### 后端模型与迁移

- 新增：`server/internal/model/audit_event.go`
  - 定义 `AuditEvent`、`AuditExportTask`（可选）模型
- 修改：`server/bootstrap/db.go`
  - 将 `AuditEvent`（和导出任务表）加入 AutoMigrate
- 新增索引（迁移 SQL 或 migrator）
  - `idx_audit_event_occurred_category` (`occurred_at`, `category`)
  - `idx_audit_event_actor` (`actor_user_id`)
  - `idx_audit_event_target` (`target_user_id`)
  - `idx_audit_event_request_id` (`request_id`)

### 后端仓储层

- 新增：`server/internal/repository/audit_event.go`
  - `CreateTx(tx, event)`
  - `Create(event)`
  - `List(page, size, filter)`
  - `Count(filter)`
  - `ListForExport(filter, limit)`

### 后端服务层

- 新增：`server/internal/service/audit_service.go`
  - 统一写入入口：`RecordEventTx`, `RecordEvent`
  - 查询入口：`AdminListAuditEvents`
  - 导出入口：`CreateExportTask`, `RunExportTask`, `GetExportTaskStatus`
- 修改：`server/internal/service/role.go`
  - 在 `SetUserRoles` 成功路径写权限审计事件
- 修改：`server/internal/service/sys_wallet.go`
  - 在 `AdminAdjust`、`ApplyWalletDeltaByOperatorTx` 写伏羲币审计事件

### 后端 Handler 与路由

- 新增：`server/internal/handler/audit_event.go`
  - `POST /system/audit/events`
  - `POST /system/audit/export`
  - `GET /system/audit/export/:task_id`
- 修改：`server/internal/router/router.go`
  - 将上述路由挂到 `admin`（或更高）权限组

### 前端 API、类型、页面

- 新增：`static/src/api/audit.ts`
- 修改：`static/src/types/api/api.d.ts`
  - 增加 `Api.Audit.*` 请求/响应类型
- 新增：`static/src/views/system/audit/index.vue`
  - 筛选、列表、详情抽屉、导出任务面板
- 修改：`static/src/router/modules/system.ts`
  - 新增审计页面路由与按钮权限
- 修改：`static/src/locales/langs/zh.json`、`static/src/locales/langs/en.json`
  - 增加审计页文案

### 定时任务与归档

- 新增：`server/jobs/audit_archive.go`
  - 每日执行：归档 >90 天在线日志
- 修改：`server/jobs/registry/...`（按现有任务注册方式）
  - 注册 `audit_archive_daily` 任务

## TODO List（计划跟踪）

### 阶段 1：最小可用（P0）

- [x] 新建 `AuditEvent` 模型并完成 AutoMigrate 接入
- [x] 完成 `audit_event` 关键索引创建与校验
- [x] 新建 `AuditEventRepository`（Create/List/Count）
- [x] 新建 `AuditService` 基础能力（`RecordEventTx/RecordEvent`）
- [x] 接入 `RoleService.SetUserRoles` 审计（成功/失败都记录）
- [x] 接入 `SysWalletService.AdminAdjust` 审计（事务内写入）
- [x] 接入 `ApplyWalletDeltaByOperatorTx` 审计
- [x] 新增 `POST /system/audit/events` 查询接口
- [x] 新增审计管理页基础列表（筛选 + 分页 + 详情）
- [ ] 补齐阶段 1 的后端单测与集成测试（进行中：已补充服务层与 Handler 定向用例）

### 阶段 2：运维增强（P1）

- [x] 新增导出任务模型与仓储（状态流转）
- [x] 新增 `POST /system/audit/export` 与 `GET /system/audit/export/:task_id`
- [x] 实现导出 worker（CSV/JSON）与轮询状态
- [x] 前端接入导出按钮、任务面板、下载入口
- [x] 导出行为写审计事件（谁导出、筛选条件、结果）
- [ ] 接入审批动作审计（福利/SRP/商城订单）
- [ ] 接入关键配置变更审计（`system_config`）
- [ ] 接入手动任务执行与调度变更审计
- [ ] 补齐阶段 2 回归测试

### 阶段 3：长期治理（P2）

- [ ] 新增归档任务 `audit_archive_daily`
- [ ] 实现在线 90 天 + 归档 1 年策略
- [ ] 实现归档任务幂等与失败重试
- [ ] 增加审计写入/导出/归档监控指标
- [ ] 增加异常告警规则（高频改权、高频调账等）
- [ ] 输出运维 Runbook（排障、回放、恢复）
- [ ] 将落地行为迁移到 `docs/features/current/*` 与 `docs/api/route-index.md`

### 当前进度（维护规则）

- [x] 2026-04-29 迭代同步
  - 已完成项（后端 P0）：
    - `audit_event` 模型、仓储、服务、handler、router 接入完成
    - `SetUserRoles`、`AdminAdjust`、`ApplyWalletDeltaByOperatorTx` 审计接入完成
    - `POST /api/v1/system/audit/events` 已上线（管理端）
  - 已完成项（前端 P0）：
    - 新增 `static/src/api/audit.ts`
    - 新增 `static/src/views/system/audit/index.vue`（筛选 + 列表 + 详情抽屉）
    - 新增系统路由 `static/src/router/modules/system.ts -> /system/audit`
    - 新增 `Api.Audit.*` 类型与中英文文案（`api.d.ts` / `zh.json` / `en.json`）
  - 测试进度：
    - 已新增并通过：`role_audit_test.go`、`sys_wallet_internal_test.go` 审计断言、`audit_event_test.go`
    - 前端类型检查通过：`pnpm exec vue-tsc --noEmit`
  - 接口变更：
    - 新增 `POST /api/v1/system/audit/events`

- [x] 2026-04-30 导出链路落地
  - 已完成项（后端 P1）：
    - 新增 `audit_export_task` 模型并接入 AutoMigrate
    - 新增导出任务仓储与状态流转（`pending/running/done/failed/expired`）
    - 新增 `POST /api/v1/system/audit/export` 与 `GET /api/v1/system/audit/export/:task_id`
    - 新增导出执行逻辑（CSV/JSON）与静态下载路径 `/uploads/audit-exports/*`
    - 导出创建与导出完成行为已写入 `audit_event`
  - 已完成项（前端 P1）：
    - 审计页新增 CSV/JSON 导出按钮
    - 新增任务状态轮询与状态展示（`pending/running/done/failed/expired`）
    - `done` 状态下载入口已接入
    - `Api.Audit` 类型与 `static/src/api/audit.ts` 导出 API 已补齐
  - 测试进度：
    - 后端：`go test ./internal/handler -run AuditEvent -count=1` 通过
    - 前端：`pnpm -C static exec vue-tsc --noEmit` 通过
  - 待继续项：
    - 导出任务历史面板（多任务列表）
    - 审批动作/配置变更/任务调度变更审计接入
    - 阶段 2 回归测试补齐
### 下一步执行顺序（P1 拆解）

1. 导出数据模型与状态机
   - 新增 `audit_export_task` 模型（`pending/running/done/failed/expired`）
   - 新增仓储接口：创建任务、抢占待执行任务、状态流转、按 `task_id` 查询
2. 导出接口与权限
   - 新增 `POST /api/v1/system/audit/export`、`GET /api/v1/system/audit/export/:task_id`
   - 路由挂载在 `/system/audit/*` 管理权限组，与查询接口一致
3. 导出执行器
   - 先落地 CSV（主路径）+ JSON（兼容）
   - 复用 `AuditEventFilter` 查询，限制单任务最大行数，失败写回错误信息
4. 前端导出面板
   - 在 `static/src/views/system/audit/index.vue` 增加导出按钮与任务状态轮询
   - 下载入口仅在 `done` 状态展示，`failed/expired` 给出明确提示
5. 审计闭环与回归
   - 导出动作写入 `audit_event`（含操作者、筛选条件摘要、结果）
   - 补齐导出链路测试：状态机流转、异常分支、权限校验
## API 契约草案（v1）

### `POST /api/v1/system/audit/events`

请求体：

```ts
{
  current: number
  size: number
  start_date?: string
  end_date?: string
  category?: string
  action?: string
  actor_user_id?: number
  target_user_id?: number
  result?: 'success' | 'failed'
  request_id?: string
  resource_id?: string
  keyword?: string
}
```

响应体：

```ts
{
  records: Array<{
    event_id: string
    occurred_at: string
    category: string
    action: string
    actor_user_id: number
    target_user_id: number
    resource_type: string
    resource_id: string
    result: 'success' | 'failed'
    request_id: string
    ip: string
    details_json: Record<string, unknown>
  }>
  total: number
  current: number
  size: number
}
```

### `POST /api/v1/system/audit/export`

请求体：

```ts
{
  format: 'csv' | 'json'
  filter: {
    start_date?: string
    end_date?: string
    category?: string
    action?: string
    actor_user_id?: number
    target_user_id?: number
    result?: 'success' | 'failed'
    request_id?: string
    resource_id?: string
    keyword?: string
  }
}
```

响应体：

```ts
{
  task_id: string
  status: 'pending'
}
```

### `GET /api/v1/system/audit/export/:task_id`

响应体：

```ts
{
  task_id: string
  status: 'pending' | 'running' | 'done' | 'failed' | 'expired'
  download_url?: string
  error_message?: string
  expire_at?: string
}
```

## 伪代码（实现级）

### 1. 审计写入统一入口（Service）

```go
// server/internal/service/audit_service.go

type AuditRecordInput struct {
    Category     string
    Action       string
    ActorUserID  uint
    TargetUserID uint
    ResourceType string
    ResourceID   string
    Result       string
    RequestID    string
    IP           string
    UserAgent    string
    Details      map[string]any
}

func (s *AuditService) RecordEventTx(tx *gorm.DB, in AuditRecordInput) error {
    payload, _ := json.Marshal(in.Details)
    event := model.AuditEvent{
        EventID:      uuid.NewString(),
        OccurredAt:   time.Now(),
        Category:     in.Category,
        Action:       in.Action,
        ActorUserID:  in.ActorUserID,
        TargetUserID: in.TargetUserID,
        ResourceType: in.ResourceType,
        ResourceID:   in.ResourceID,
        Result:       in.Result,
        RequestID:    in.RequestID,
        IP:           in.IP,
        UserAgent:    in.UserAgent,
        DetailsJSON:  string(payload),
    }
    return s.repo.CreateTx(tx, &event)
}
```

### 2. 权限变更接入点

```go
// server/internal/service/role.go -> SetUserRoles

before := currentCodes
requested := requestedCodes
err := s.repo.SetUserRoles(userID, requested)
if err != nil {
    _ = auditSvc.RecordEvent(ctx, AuditRecordInput{
        Category: "permission", Action: "set_user_roles",
        ActorUserID: operatorID, TargetUserID: userID,
        ResourceType: "user_role", ResourceID: strconv.Itoa(int(userID)),
        Result: "failed",
        Details: map[string]any{"before_roles": before, "after_roles": requested, "error": err.Error()},
    })
    return err
}

after, _ := s.repo.GetUserRoleCodes(userID)
_ = auditSvc.RecordEvent(ctx, AuditRecordInput{
    Category: "permission", Action: "set_user_roles",
    ActorUserID: operatorID, TargetUserID: userID,
    ResourceType: "user_role", ResourceID: strconv.Itoa(int(userID)),
    Result: "success",
    Details: map[string]any{"before_roles": before, "after_roles": after},
})
```

### 3. 伏羲币变动接入点

```go
// server/internal/service/sys_wallet.go -> AdminAdjust

return global.DB.Transaction(func(tx *gorm.DB) error {
    // 1) 查钱包并计算 newBalance
    // 2) 写 system_wallet + wallet_transaction + wallet_log

    // 3) 同事务写统一审计事件
    err = auditSvc.RecordEventTx(tx, AuditRecordInput{
        Category: "fuxi_wallet", Action: "admin_adjust",
        ActorUserID: operatorID, TargetUserID: req.TargetUID,
        ResourceType: "system_wallet", ResourceID: strconv.Itoa(int(req.TargetUID)),
        Result: "success",
        Details: map[string]any{
            "adjust_action": req.Action,
            "amount": req.Amount,
            "before": oldBalance,
            "after": newBalance,
            "reason": req.Reason,
            "ref_type": "admin_adjust",
        },
    })
    if err != nil { return err }

    return nil
})
```

```go
// server/internal/service/sys_wallet.go -> ApplyWalletDeltaByOperatorTx

if delta != 0 {
    // 写 wallet_transaction 后
    _ = auditSvc.RecordEventTx(tx, AuditRecordInput{
        Category: "fuxi_wallet", Action: "apply_wallet_delta",
        ActorUserID: operatorID, TargetUserID: userID,
        ResourceType: "wallet_transaction", ResourceID: refID,
        Result: "success",
        Details: map[string]any{
            "delta": delta,
            "ref_type": refType,
            "reason": reason,
            "balance_after": newBalance,
        },
    })
}
```

### 4. 查询与导出 Handler

```go
// server/internal/handler/audit_event.go

func (h *AuditEventHandler) AdminList(c *gin.Context) {
    var req adminAuditListRequest
    if err := c.ShouldBindJSON(&req); err != nil { failParam(...) ; return }
    req.Current, req.Size = normalizeLedgerPagination(req.Current, req.Size)

    filter := repository.AuditEventFilter{...}
    records, total, err := h.svc.AdminListAuditEvents(req.Current, req.Size, filter)
    if err != nil { failBiz(...) ; return }
    response.OKWithPage(c, records, total, req.Current, req.Size)
}

func (h *AuditEventHandler) CreateExportTask(c *gin.Context) {
    operatorID := middleware.GetUserID(c)
    task, err := h.svc.CreateExportTask(operatorID, req.Format, req.Filter)
    if err != nil { failBiz(...) ; return }
    response.OK(c, task)
}
```

### 5. 导出任务执行

```go
// server/internal/service/audit_service.go

func (s *AuditService) RunExportTask(ctx context.Context, taskID string) error {
    task := repo.GetExportTaskForUpdate(taskID)
    if task.Status != "pending" { return nil }

    repo.MarkTaskRunning(taskID)
    rows := repo.ListForExport(task.Filter, maxExportRows)

    filePath, err := exporter.Write(task.Format, rows) // csv/json
    if err != nil {
        repo.MarkTaskFailed(taskID, err.Error())
        return err
    }

    repo.MarkTaskDone(taskID, filePath, expireAt)

    _ = s.RecordEvent(ctx, AuditRecordInput{
        Category: "task_ops", Action: "audit_export_generated",
        ActorUserID: task.OperatorID,
        ResourceType: "audit_export_task", ResourceID: taskID,
        Result: "success",
        Details: map[string]any{"format": task.Format, "row_count": len(rows)},
    })
    return nil
}
```

### 6. 归档任务

```go
// server/jobs/audit_archive.go

func runAuditArchiveDaily(ctx context.Context) {
    cutoff := time.Now().AddDate(0, 0, -90)

    batches := repo.ListArchiveCandidates(cutoff, batchSize)
    for _, batch := range batches {
        // 1) 导出 batch 到归档存储
        // 2) 校验写入成功
        // 3) 删除在线库对应记录
    }

    // 记录任务执行审计
    _ = auditSvc.RecordEvent(ctx, AuditRecordInput{
        Category: "task_ops", Action: "audit_archive_daily",
        Result: "success",
        Details: map[string]any{"cutoff": cutoff.Format(time.RFC3339), "archived_count": total},
    })
}
```

### 7. 前端页面交互

```ts
// static/src/views/system/audit/index.vue (pseudo)

const filters = reactive({
  startDate: '', endDate: '', category: '', action: '',
  actorUserId: undefined, targetUserId: undefined,
  result: '', requestId: '', resourceId: '', keyword: ''
})

async function fetchAuditEvents() {
  loading.value = true
  const resp = await fetchAuditEventsApi({ current, size, ...filters })
  tableData.value = resp.records
  total.value = resp.total
  loading.value = false
}

async function createExport(format: 'csv' | 'json') {
  const task = await fetchCreateAuditExportTask({ format, filter: mapFilters(filters) })
  startPollingTask(task.task_id)
}

async function startPollingTask(taskId: string) {
  const timer = setInterval(async () => {
    const status = await fetchAuditExportTaskStatus(taskId)
    if (status.status === 'done') {
      clearInterval(timer)
      window.open(status.download_url)
    }
    if (status.status === 'failed' || status.status === 'expired') {
      clearInterval(timer)
      showError(status.error_message || t('audit.exportFailed'))
    }
  }, 3000)
}
```

## 测试断言矩阵

- `RoleService.SetUserRoles`
  - 成功写 1 条 `permission/set_user_roles`，before/after 正确
  - 失败写 1 条 failed 事件并含 error
- `SysWalletService.AdminAdjust`
  - 事务内四件套一致：`system_wallet`、`wallet_transaction`、`wallet_log`、`audit_event`
- `ApplyWalletDeltaByOperatorTx`
  - `delta=0` 不写审计；非 0 写 `fuxi_wallet/apply_wallet_delta`
- `POST /system/audit/events`
  - 全筛选条件可组合，分页稳定
- 导出任务
  - pending->running->done 正常流转
  - 失败场景写 failed，并返回错误原因
- 归档任务
  - 重复运行幂等，不重复删除

## 设计理由

- 决策：新增 `audit_event` 统一审计层，保留现有 `operation_log` 与 `wallet_*` 表
- 理由：
  - 降低跨表拼接成本，形成明确“审计事实模型”
  - 不破坏现有账本语义和既有查询页面，改造风险可控
  - 通过分阶段接入优先覆盖高风险链路，缩短上线周期
- 取舍 / 未采用方案：
  - 未采用“仅扩展 `operation_log`”：其粒度偏请求，不适合作为业务审计事实
  - 未采用“仅扩展 `wallet_log`”：无法覆盖权限、配置、审批等非钱包行为
  - 未采用“在线全量永久保留”：长期成本高，索引膨胀影响查询性能
- 如果落地，必须迁移到的权威文档：
  - `docs/features/current/commerce.md`
  - `docs/features/current/administration.md`
  - `docs/features/current/welfare.md`
  - `docs/features/current/srp.md`
  - `docs/features/current/task-manager.md`
  - `docs/architecture/database-schema.md`
  - `docs/api/route-index.md`

## 未决问题

- 审计导出文件最终落地介质（对象存储/本地卷）与保密级别
- `details_json` 的字段白名单与脱敏策略（昵称、QQ、IP 是否部分脱敏）
- 是否在一期就引入审计告警看板，或先只做事件落库 + 查询导出

## 明确声明

- 本文档是提案，不代表当前已实现行为
- 不能覆盖 `docs/ai/repo-rules.md`、`docs/architecture/`、`docs/api/`、`docs/features/current/`

## 升级路径

- 阶段 1（最小可用）：
  - 建立 `audit_event` 表与服务层写入能力
  - 接入权限分配、管理员调账、统一钱包入口
  - 提供在线筛选查询
- 阶段 2（运维增强）：
  - 接入审批动作、任务运维动作、配置变更
  - 提供异步导出（CSV/JSON）与下载
- 阶段 3（长期治理）：
  - 上线 90 天在线 + 1 年归档
  - 补齐告警规则与异常审计面板
  - 将已落地行为迁移到 `docs/features/current/*` 与 `docs/api/route-index.md`





---
status: draft
doc_type: draft
owner: engineering
last_reviewed: 2026-04-30
source_of_truth:
  - server/internal/model/audit_event.go
  - server/internal/repository/audit_event.go
  - server/internal/service/audit_service.go
  - server/internal/handler/audit_event.go
  - server/internal/model/operation_log.go
  - server/internal/model/sys_wallet.go
  - server/internal/service/sys_wallet.go
  - server/internal/service/role.go
  - server/internal/service/welfare.go
  - server/internal/service/srp.go
  - server/internal/service/shop.go
  - server/internal/service/task.go
  - server/internal/service/sys_webhook.go
  - server/internal/repository/sys_wallet.go
  - server/internal/router/router.go
  - static/src/views/system/wallet
  - static/src/views/system/audit
  - static/src/api/sys-wallet.ts
  - static/src/api/audit.ts
  - static/src/types/api/api.d.ts
  - static/src/router/modules/system.ts
  - docs/features/current/audit-log.md
  - docs/api/route-index.md
  - docs/features/current/administration.md
  - docs/features/current/commerce.md
  - docs/features/current/srp.md
  - docs/features/current/task-manager.md
  - docs/features/current/welfare.md
  - docs/architecture/auth-and-permissions.md
  - docs/guides/audit-log-runbook.md
---

# 审计日志系统方案（权限操作 + 伏羲币变动 + 运维查询导出）

## 当前状态

- 已实现：
  - 统一审计事实表：`audit_event` + 异步导出任务表：`audit_export_task`
  - 审计在线查询、导出创建、导出状态查询、导出历史列表
  - 系统审计管理页（筛选、详情抽屉、导出任务面板）
  - 已接入审计面的主链路：
    - `permission`：`RoleService.SetUserRoles`
    - `fuxi_wallet`：`AdminAdjust`、`ApplyWalletDeltaByOperatorTx`、`CreditUser`、`DebitUser`
    - `approval`：`WelfareService.AdminReviewApplication`、`SrpService.ReviewApplication`、`ShopService.AdminDeliverOrder`、`ShopService.AdminRejectOrder`
    - `task_ops`：`TaskService.RunTask`、`TaskService.UpdateSchedule`
    - `config`：`WebhookService.SetConfigByOperator`、`PAPExchangeService.UpdateConfigByOperator`、`WelfareSettingsService.UpdateSettingsByOperator`、`MentorSettingsService.UpdateSettingsByOperator`、`NewbroSettingsService.UpdateSupportSettingsByOperator`、`NewbroSettingsService.UpdateRecruitSettingsByOperator`
  - 审计写入、导出与查询的服务/仓储/handler/router/API/类型已串通
- 未实现：
  - （无）

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
  - 关键配置变更（`system_config`、Webhook）

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
- [x] 补齐阶段 1 的后端单测与 Handler 定向用例

### 阶段 2：运维增强（P1）

- [x] 新增导出任务模型与仓储（状态流转）
- [x] 新增 `POST /system/audit/export`、`GET /system/audit/export/:task_id` 与 `POST /system/audit/export/list`
- [x] 实现导出 worker（CSV/JSON）与轮询状态
- [x] 前端接入导出按钮、任务面板、下载入口
- [x] 导出行为写审计事件（谁导出、筛选条件、结果）
- [x] 接入审批动作审计（福利/SRP/商城订单）
- [x] 接入任务运维审计（手动任务执行与调度变更）
- [x] 接入 Webhook 配置变更审计
- [x] 接入 `system_config` 审计（Webhook/PAP/Newbro/Mentor/Welfare 高优入口已覆盖）
- [x] 补齐 `system_config` 长尾配置入口
- [x] 接入自动权限映射变更审计（`esi_role_mapping` / `esi_title_mapping` 增删）
- [x] 补齐阶段 2 跨模块回归测试

### 阶段 3：长期治理（P2）

- [x] 新增归档任务 `audit_archive_daily`
- [x] 实现在线 90 天 + 归档 1 年策略
- [x] 实现归档任务幂等与失败重试
- [x] 增加审计写入/导出/归档监控指标
- [x] 增加异常告警规则（高频改权、高频调账等）
- [x] 输出运维 Runbook（排障、回放、恢复）

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
    - 已新增定向用例：`role_audit_test.go`、`sys_wallet_internal_test.go` 审计断言、`audit_event_test.go`
    - 前端类型契约已补齐：`Api.Audit.*`、`audit.ts`、`system/audit` 页面
  - 接口变更：
    - 新增 `POST /api/v1/system/audit/events`

- [x] 2026-04-30 导出与审批链路落地
  - 已完成项（后端 P1）：
    - 新增 `audit_export_task` 模型并接入 AutoMigrate
    - 新增导出任务仓储与状态流转（`pending/running/done/failed/expired`）
    - 新增 `POST /api/v1/system/audit/export`、`GET /api/v1/system/audit/export/:task_id` 与 `POST /api/v1/system/audit/export/list`
    - 新增导出执行逻辑（CSV/JSON）与静态下载路径 `/uploads/audit-exports/*`
    - 导出创建与导出完成行为已写入 `audit_event`
    - `WelfareService.AdminReviewApplication` 已写入 `approval/welfare_application_deliver|reject`
    - `SrpService.ReviewApplication` 已写入 `approval/srp_application_approve|reject`
    - `ShopService.AdminDeliverOrder` / `AdminRejectOrder` 已写入 `approval/shop_order_deliver|reject`
    - `TaskService.RunTask` / `UpdateSchedule` 已写入 `task_ops/task_manual_run|task_schedule_update`
    - `WebhookService.SetConfigByOperator` 已写入 `config/webhook_config_update`
  - 已完成项（前端 P1）：
    - 审计页新增 CSV/JSON 导出按钮
    - 新增任务状态轮询与状态展示（`pending/running/done/failed/expired`）
    - `done` 状态下载入口已接入
    - 导出任务历史面板（多任务列表，含状态与下载入口）已接入
  - 测试进度：
    - `role_audit_test.go`、`sys_wallet_internal_test.go`、`audit_event_test.go`
    - `task_test.go`、`task_handler_test.go`
    - `sys_webhook_test.go`、`sys_webhook` handler 级审计回归
    - `welfare_test.go`、`srp_test.go`、`shop_test.go` 相关审计断言
  - 状态：
    - 阶段 2 已覆盖主链路，剩余是 `system_config` 长尾配置 / 更细的安全审计面

- [x] 2026-04-30 自动权限映射与配置审计补充
  - 已完成项（后端 P1）：
    - `AutoRoleService` 已接入自动权限映射审计：
      - `permission/esi_role_mapping_create|delete`
      - `permission/esi_title_mapping_create|delete`
    - `PAPExchangeService.UpdateConfigByOperator` 已接入配置审计：
      - `config/pap_exchange_config_update`（`resource_type=system_config`）
  - 测试进度：
    - 新增 `auto_role_audit_test.go`
    - 新增 `pap_exchange_test.go` 的配置审计断言
  - 状态：
    - 自动权限映射审计已完成，`system_config` 仍需继续补齐其他配置入口

- [x] 2026-04-30 `system_config` 高优入口补齐（第二批）
  - 已完成项（后端 P1）：
    - `WelfareSettingsService.UpdateSettingsByOperator` -> `config/welfare_settings_update`
    - `MentorSettingsService.UpdateSettingsByOperator` -> `config/mentor_settings_update`
    - `NewbroSettingsService`：
      - `UpdateSupportSettingsByOperator` -> `config/newbro_support_settings_update`
      - `UpdateRecruitSettingsByOperator` -> `config/newbro_recruit_settings_update`
    - 对应管理端 handler 已统一传入操作者 ID（`middleware.GetUserID`）
  - 测试进度：
    - `welfare_settings_test.go` 新增审计断言
    - `mentor_settings_test.go` 新增审计断言
    - `newbro_settings_test.go` 新增审计断言（support/recruit）
  - 状态：
    - `system_config` 审计高频入口已覆盖完成，剩余长尾配置可按模块继续补齐
  - 测试补充：go test ./internal/...（handler/service/repository/router 全通过）

- [x] 2026-04-30 计划收口（归档 + 安全审计 + Runbook）
  - 已完成项（后端 P2）：
    - 新增 `audit_archive_daily` 周期任务并完成任务注册
    - 新增 `AuditArchiveService`，实现 `audit_event` 90 天在线保留、批量归档与在线清理
    - 归档流程支持批量空跑幂等，失败后可直接重试任务
    - `EveSSOHandler` 新增 `security` 审计事件：
      - `eve_sso_login_start|scope_rejected|login_url_failed`
      - `eve_sso_callback_success|callback_failed|callback_denied`
      - `eve_sso_bind_start|bind_scope_rejected|bind_url_failed`
    - `CorporationStructureService.UpdateAuthorizations` 接入配置审计：
      - `config/corp_structure_authorization_update`
  - 已完成项（运维文档）：
    - 新增 `docs/guides/audit-log-runbook.md`
    - 补齐审计写入/导出/归档监控指标与告警规则
    - 补齐故障排障与恢复流程
  - 测试进度：
    - `go test ./internal/service -run AuditArchive -count=1`
    - `go test ./jobs -run RegisterAllRegistersExpectedTaskDefinitions -count=1`
  - 状态：
    - 阶段 1/2/3 计划项已全部落地，后续进入日常运维与持续优化阶段

### 下一步执行顺序（P1 拆解）

1. 已完成，进入维护阶段
   - 按 `docs/guides/audit-log-runbook.md` 执行监控、告警与恢复
2. 持续优化
   - 根据线上数据扩展 `security` 事件动作与阈值策略
3. 文档迁移
   - 将稳定结论按规范同步到 `docs/features/current/` 与 `docs/api/route-index.md`
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
  - 已迁移当前审计能力说明到 `docs/features/current/audit-log.md` 与 `docs/api/route-index.md`


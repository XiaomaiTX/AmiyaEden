---
status: active
doc_type: feature
owner: engineering
last_reviewed: 2026-04-30
source_of_truth:
  - server/internal/model/audit_event.go
  - server/internal/repository/audit_event.go
  - server/internal/service/audit_service.go
  - server/internal/service/role.go
  - server/internal/service/sys_wallet.go
  - server/internal/service/welfare.go
  - server/internal/service/srp.go
  - server/internal/service/shop.go
  - server/internal/service/task.go
  - server/internal/service/sys_webhook.go
  - server/internal/handler/audit_event.go
  - server/internal/router/router.go
  - static/src/api/audit.ts
  - static/src/types/api/api.d.ts
  - static/src/router/modules/system.ts
  - static/src/views/system/audit
  - docs/api/route-index.md
---

# 审计日志

## 当前能力

- 管理端统一审计页 `/system/audit`
- 页面分为「审计事件」与「导出日志」两个 tab
- 审计事件分页查询，支持按时间、分类、动作、操作者、目标用户、结果、`request_id`、资源 ID、关键词筛选
- 审计事件详情抽屉，展示事件主字段与 `details_json`
- 阶段一止损：前端已临时隐藏导出触发、导出任务列表与下载入口；导出日志 tab 显示“暂不可用”
- 后端导出接口与任务机制仍保留（未移除）
- 当前已接入的审计链路：
  - `permission`：用户职权分配
  - `fuxi_wallet`：伏羲币调账与统一钱包差量入口
  - `approval`：福利、SRP、商城订单审批
  - `task_ops`：任务手动执行与调度更新
  - `config`：Webhook 配置变更
- 审计导出本身也会写入审计事件，保留“谁导出了什么”的可追溯性

## 入口

### 前端页面

- `static/src/views/system/audit`
- `static/src/router/modules/system.ts` 中的 `/system/audit`

### 前端 API

- `static/src/api/audit.ts`
- `static/src/types/api/api.d.ts`

### 后端路由

- `/api/v1/system/audit/events`
- `/api/v1/system/audit/export`
- `/api/v1/system/audit/export/:task_id`
- `/api/v1/system/audit/export/list`

## 权限边界

- `/system/audit` 页面仅 `admin` 可见
- `/api/v1/system/audit/*` 默认要求 `admin`
- 审计查询与导出均不开放给普通登录用户

## 关键不变量

- `audit_event` 是统一审计事实表，不替代 `operation_log` 或账本流水表
- 现有钱包、审批、任务与配置审计继续保留各自领域表，审计表只承载统一检索与留痕视图
- 审计导出任务状态固定为 `pending`、`running`、`done`、`failed`、`expired`
- 导出结果通过 `/uploads/audit-exports/*` 提供静态下载
- 前端当前不直接暴露下载入口；后续阶段会恢复受控下载能力
- `details_json` 主要保存 before/after、原因和补充上下文，查询页默认只展示原始 JSON
- 当审计表在测试或临时环境中不存在时，写入入口会尽量降级为 noop，以免影响主流程

## 主要代码文件

- `server/internal/model/audit_event.go`
- `server/internal/repository/audit_event.go`
- `server/internal/service/audit_service.go`
- `server/internal/service/role.go`
- `server/internal/service/sys_wallet.go`
- `server/internal/service/welfare.go`
- `server/internal/service/srp.go`
- `server/internal/service/shop.go`
- `server/internal/service/task.go`
- `server/internal/service/sys_webhook.go`
- `server/internal/handler/audit_event.go`
- `server/internal/router/router.go`
- `static/src/api/audit.ts`
- `static/src/types/api/api.d.ts`
- `static/src/router/modules/system.ts`
- `static/src/views/system/audit`

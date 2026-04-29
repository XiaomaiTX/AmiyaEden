---
status: active
doc_type: feature
owner: engineering
last_reviewed: 2026-04-29
source_of_truth:
  - server/internal/model/ticket.go
  - server/internal/repository/ticket.go
  - server/internal/service/ticket.go
  - server/internal/handler/ticket.go
  - server/internal/router/router.go
  - server/bootstrap/db.go
  - docs/api/route-index.md
  - static/src/api/ticket.ts
  - static/src/router/modules/ticket.ts
  - static/src/views/ticket
  - static/src/views/system/ticket-management
  - static/src/views/system/ticket-detail
  - static/src/views/system/ticket-categories
  - static/src/views/system/ticket-statistics
  - static/src/components/ticket
---

# 工单系统

## 当前能力

- 提供成员侧工单提交、我的工单列表、工单详情与追加回复能力
- 提供管理员侧工单管理列表、状态更新、优先级更新、回复与内部备注能力
- 提供工单分类管理（列表、创建、更新、删除）
- 提供状态变更历史查询与统计看板（总量、状态分布、分类分布、近 7/30 天）
- 工单创建时默认状态为 `pending`，优先级缺省为 `medium`
- 管理员回复可标记 `is_internal`，成员侧回复列表不会返回内部备注
- 状态更新到 `in_progress` / `completed` 时自动记录处理人；更新到 `completed` 时记录关闭时间
- 系统启动时会初始化默认工单分类（若不存在）

## 入口

### 前端页面

- `/ticket/my-tickets`
- `/ticket/create`
- `/ticket/detail/:id`
- `/system/ticket-management`
- `/system/ticket-categories`
- `/system/ticket-statistics`
- `/system/ticket-detail/:id`

### 后端路由

成员侧：

- `POST /api/v1/ticket/tickets`
- `GET /api/v1/ticket/tickets/me`
- `GET /api/v1/ticket/tickets/:id`
- `POST /api/v1/ticket/tickets/:id/replies`
- `GET /api/v1/ticket/tickets/:id/replies`
- `GET /api/v1/ticket/categories`

管理侧：

- `GET /api/v1/system/ticket/tickets`
- `GET /api/v1/system/ticket/tickets/:id`
- `PUT /api/v1/system/ticket/tickets/:id/status`
- `PUT /api/v1/system/ticket/tickets/:id/priority`
- `POST /api/v1/system/ticket/tickets/:id/replies`
- `GET /api/v1/system/ticket/tickets/:id/replies`
- `GET /api/v1/system/ticket/tickets/:id/status-history`
- `GET /api/v1/system/ticket/categories`
- `POST /api/v1/system/ticket/categories`
- `PUT /api/v1/system/ticket/categories/:id`
- `DELETE /api/v1/system/ticket/categories/:id`
- `GET /api/v1/system/ticket/statistics`

## 权限边界

- 成员侧能力要求 `Login`（非 guest）
- 成员只能访问自己创建的工单
- 管理侧能力要求 `RequireRole(admin)`（`super_admin` 通过角色匹配隐式可用）
- 前端路由权限用于 UX，引导不构成安全边界；最终以后端鉴权为准

## 关键不变量

- 工单状态只支持：`pending`、`in_progress`、`completed`
- 工单优先级只支持：`low`、`medium`、`high`
- 成员侧回复查询必须过滤 `is_internal = true`
- 状态变更只有在状态值实际变化时才写入 `ticket_status_history`
- 统计接口返回值至少包含 `total`、`status`、`category`、`recent_7d`、`recent_30d`、`pendingCount`

## 当前非目标

- 尚未落地菜单栏徽章提醒（不接入独立通知中心）
- 尚未落地管理员间转派能力
- 尚未落地归档 / 清理策略
- 尚未补齐 E2E 回归

## 主要代码文件

- `server/internal/model/ticket.go`
- `server/internal/repository/ticket.go`
- `server/internal/service/ticket.go`
- `server/internal/handler/ticket.go`
- `server/internal/router/router.go`
- `server/bootstrap/db.go`
- `static/src/api/ticket.ts`
- `static/src/router/modules/ticket.ts`
- `static/src/views/ticket`
- `static/src/views/system/ticket-management`
- `static/src/views/system/ticket-detail`
- `static/src/views/system/ticket-categories`
- `static/src/views/system/ticket-statistics`
- `static/src/components/ticket`

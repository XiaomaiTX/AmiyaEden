---
status: active
doc_type: architecture
owner: engineering
last_reviewed: 2026-11-26
source_of_truth:
  - server/internal
  - server/pkg
  - static/src
  - static-react/src
  - docs/ai/repo-rules.md
---

# 模块地图

## 目的

本文件回答两个问题：

- 这个仓库里某类职责应该去哪个目录找
- 修改某类能力时，通常要同时看哪些文件

它不是新的规范来源，而是 `docs/ai/repo-rules.md` 与现有 architecture / api / feature 文档的导航图。

## Backend 目录职责

| 路径 | 主要职责 | 修改时通常还要看 |
| --- | --- | --- |
| `server/bootstrap/` | 应用启动、配置装配、路由/日志/任务初始化 | `server/main.go`、`docs/architecture/runtime-and-startup.md` |
| `server/config/` | 配置结构与配置文件模板，包括 `EveSSOConfig`（SSO/ESI 端点、ClientID/Secret 等） | `README.md`、`docs/guides/local-development.md`、`docs/features/current/auth-and-characters.md` |
| `server/internal/router/` | 路由注册、分组、中间件挂载 | `server/internal/handler/`、`docs/api/route-index.md` |
| `server/internal/middleware/` | JWT、日志、CORS、统一请求前置逻辑 | `server/internal/router/`、`docs/architecture/auth-and-permissions.md` |
| `server/internal/handler/` | HTTP 请求解析与响应返回 | `server/internal/service/`、`server/pkg/response/` |
| `server/internal/service/` | 业务规则、权限判断、跨仓储编排、ESI/SSO 集成 | `server/internal/repository/`、对应 feature doc |
| `server/internal/taskregistry/` | 运行时任务定义、任务分类元数据、进程内执行锁 | `server/jobs/`、`server/bootstrap/cron.go`、`server/internal/service/task.go` |
| `server/internal/repository/` | 数据访问、查询拼接、结果映射 | `server/internal/model/`、`docs/standards/testing-and-verification.md` |
| `server/internal/model/` | GORM 模型、职权常量 | `docs/architecture/database-schema.md`、`docs/architecture/auth-and-permissions.md` |
| `server/jobs/` | 后台任务定义与执行实现，包含 cron 与事件驱动任务 | `server/internal/taskregistry/`、`server/bootstrap/`、`docs/architecture/runtime-and-startup.md` |
| `server/pkg/eve/` | EVE SSO / ESI 基础能力 | `server/internal/service/`、`docs/guides/adding-esi-feature.md` |
| `server/pkg/response/` | 统一响应封装 | `server/internal/handler/` |

## Frontend 目录职责

| 路径 | 主要职责 | 修改时通常还要看 |
| --- | --- | --- |
| `static/src/views/` | 页面视图与页面级状态 | `static/src/api/`、`static/src/hooks/`、对应 feature doc |
| `static/src/api/` | 前端 API 包装层 | `server/internal/handler/`、`static/src/types/api/api.d.ts` |
| `static/src/components/` | 可复用 UI 组件 | `static/src/hooks/`、`static/src/locales/` |
| `static/src/hooks/` | 共享逻辑、状态转换、纯 helper | `docs/guides/testing-guide.md` |
| `static/src/store/` | Pinia 状态 | `static/src/router/`、`static/src/hooks/core/useAuth.ts` |
| `static/src/router/` | 路由核心、静态路由模块、守卫 | `docs/architecture/routing-and-menus.md`、`docs/architecture/auth-and-permissions.md` |
| `static/src/types/` | TS 类型、导入声明、契约类型 | `static/src/api/`、`server/internal/model/` |
| `static/src/locales/` | i18n 文案 | `docs/ai/repo-rules.md`「Non-Negotiable Rules」第 3 条 |

React 迁移侧对应职责：

| 路径 | 主要职责 | 修改时通常还要看 |
| --- | --- | --- |
| `static-react/src/pages/` | React 页面与页面级状态 | `static-react/src/api/`、对应 feature doc |
| `static-react/src/api/` | React API 包装层 | 后端 handler、`static-react/src/types/api/` |
| `static-react/src/components/` | 可复用 React UI 组件 | hooks、stores、i18n |
| `static-react/src/stores/` | Zustand 会话与跨页面状态 | `static-react/src/auth/`、对应 feature doc |
| `static-react/src/app/` | React 路由、路由元数据和访问守卫 | `docs/architecture/routing-and-menus.md`、`auth-and-permissions.md` |
| `static-react/src/types/api/` | React 本地模块化 API 类型 | `static-react/src/api/`、后端 response |
| `static-react/src/i18n/` | React 本地化文案与解析 | Vue locale 内容、对应 feature doc |

## 文档目录职责

| 路径 | 主要职责 |
| --- | --- |
| `docs/standards/` | 工程标准，回答“必须 / 不得 / 推荐” |
| `docs/architecture/` | 当前已存在结构与运行方式 |
| `docs/api/` | 接口约定、路由与边界 |
| `docs/features/current/` | 已落地功能行为 |
| `docs/guides/` | 操作步骤与实践指南 |
| `docs/reference/` | 参考资产，不作为当前实现裁决依据 |

## 常见任务落点

### 修改军团加入申请

通常需要一起看：

1. `server/internal/model/admission.go`
2. `server/internal/repository/admission.go`
3. `server/bootstrap/admission_migration.go`
4. [军团加入申请](../features/current/corporation-admissions.md)

### 新增后端接口

通常需要一起看：

1. [OpenAPI 根规范](../../contracts/openapi.yaml)或对应业务片段 — 遵循[契约同步流程](../api/conventions.md#契约同步顺序)。
2. 对应 handler 及受影响的服务、仓储。
3. 两端生成客户端和实际消费的页面、hook 或 store。
4. 受影响的功能文档；[路由索引](../api/route-index.md)仅在业务域入口变化时同步。

### 修改后台任务、调度或执行历史

通常需要一起看：

1. `server/internal/taskregistry/`
2. `server/jobs/`
3. `server/bootstrap/cron.go`
4. `server/internal/service/task.go`
5. `server/internal/repository/task.go`
6. `server/internal/model/task.go`
7. `static/src/api/generated/client.ts` 或 React 生成客户端
8. `static/src/views/system/task-manager`
9. `docs/features/current/task-manager.md`

### 修改权限、菜单、按钮点位

通常需要一起看：

1. `server/internal/router/`
2. `server/internal/middleware/`
3. `static/src/router/` and/or `static-react/src/app/`
4. 页面里的 `v-auth` or React permission gate/hook
5. `docs/architecture/routing-and-menus.md`
6. `docs/architecture/auth-and-permissions.md`

### 修改 ESI / SSO 行为

通常需要一起看：

1. `server/pkg/eve/`
2. `server/internal/service/`
3. `server/internal/handler/eve_sso.go`
4. `docs/features/current/auth-and-characters.md`
5. `docs/features/current/task-manager.md`
6. `docs/guides/adding-esi-feature.md`

### 修改军团建筑管理

通常需要一起看：

1. `server/internal/router/`
2. `server/internal/handler/corporation_structure.go`
3. `server/internal/service/corporation_structure.go`
4. `server/internal/repository/corporation_structure.go`
5. `server/internal/model/sys_config.go`
6. `contracts/modules/corp-structures.yaml` 与两端生成客户端
7. `static/src/views/dashboard/corporation-structures` 及 React 对应页面
8. `docs/features/current/corporation-structures.md`
9. `docs/features/current/badge-counts.md`
10. `docs/api/route-index.md`（阅读索引）

### 修改表格页、卡片记录页或复杂筛选页

通常需要一起看：

1. Vue `static/src/views/目标页面` or React `static-react/src/pages/目标页面`
2. the frontend's shared table components
3. the frontend's hooks/stores
4. `docs/standards/frontend-table-pages.md`
5. `docs/standards/frontend-record-card-pages.md`

## 读代码顺序建议

对不熟悉的模块，优先按下面顺序建立上下文：

1. 相关 feature doc
2. 页面或 handler 入口
3. 对应 service
4. 对应 repository / api wrapper
5. 关联类型与文档边界

[QQ 官方机器人](../features/current/qq-official.md)：调度、群授权及业务提交由 `server/internal/service/qq_official*.go` 管理；私有协议包不暴露给业务端。Vue、React 和 Go 同步使用 [OpenAPI 模块](../../contracts/modules/qq-official.yaml)；群事件由后台主动连接 QQ 官方 WebSocket 网关接收，不注册 HTTP 回调路由。见[调度架构](qq-official-dispatch.md)。

统一工资管理：`service/salary.go`、`handler/salary.go`、`model/salary.go` 和 `bootstrap/salary_migration.go`；双端 `/system/salary`。各业务最终完成事务调用工资服务，不在审批初审或自动发放路径计操作工资；参见[工资管理](../features/current/operation-salaries.md)。

联盟黑名单：`service/blacklist.go`（刷新编排与查询）、`blacklist_source.go`（Sheet 下载与解析）、`blacklist_esi.go`（ESI 批量解析人名），以及 `handler/blacklist.go`、`repository/blacklist.go`、`model/blacklist.go` 和 `bootstrap/blacklist_migration.go`；双端 `/system/blacklist`。参见[系统管理 · 联盟黑名单](../features/current/administration.md#联盟黑名单)。

---
status: active
doc_type: index
owner: engineering
last_reviewed: 2026-11-26
source_of_truth:
  - docs/ai/repo-rules.md
---

# AmiyaEden 文档索引

## Vue / React 前端迁移入口

迁移期间的唯一状态入口是 [`specs/draft/frontend-react-migration-plan/index.md`](specs/draft/frontend-react-migration-plan/index.md)。

- 页面范围、实现状态和文档适配矩阵：[`migration-scope-baseline.md`](specs/draft/frontend-react-migration-plan/migration-scope-baseline.md)
- 基础组件与权限能力替换：[`component-replacement-list.md`](specs/draft/frontend-react-migration-plan/component-replacement-list.md)
- API 类型与 wrapper 迁移：[`api-contract-migration-plan.md`](specs/draft/frontend-react-migration-plan/api-contract-migration-plan.md)
- 替换、回切和下线检查：[`coexistence-migration-todo.md`](specs/draft/frontend-react-migration-plan/coexistence-migration-todo.md)

过渡期 active 文档描述 Vue/React 必须一致的产品行为；Vue-only 的实现限制只能作为明确的迁移缺口出现。归档文档保留历史实现范围，不作为当前行为来源。

## 目的

本索引按任务指向工程标准、当前架构、API、模块行为和设计提案。[仓库首页](../README.md)是产品与上手入口；`AGENTS.md` 和 `CLAUDE.md` 指向[仓库规则](ai/repo-rules.md)。

## 信任顺序

来源冲突时，参照仓库规则中的[权威顺序](ai/repo-rules.md#authority-order)与[代码和文档的判定规则](ai/repo-rules.md#code-vs-docs-rule)。

## 受众分类

文档按主要受众分为两类：

### AI Agent 中心（agent-centric）

以下目录的文档主要服务于 AI 代理和自动化工具。人类工程师可能偶尔阅读（特别是 `docs/architecture/`），但内容以 agent 消费为主：

- `docs/ai/` — 代理规则、onboarding、Harness 原则
- `docs/standards/` — 约束性标准与回归测试策略
- `docs/architecture/` — 系统结构（人类也会少量阅读）
- `docs/api/` — 接口约定与路由索引
- `docs/features/current/` — 功能行为规格

### 人类工程师中心（human-centric）

以下目录的文档主要服务于人类开发者，提供日常开发所需的操作指南：

- `docs/guides/` — 本地开发、调试流程、测试编写、ESI 模块添加

### 通用（utility）

以下目录同时服务两类受众：

- `docs/reference/` — 离线参考资产
- `docs/specs/draft/` — 提案与未完成设计
- `docs/templates/` — 文档模板

放置新文档时，根据主要受众选择目录。如果某文档主要由 agent 消费，不应放在 `docs/guides/`；如果某文档主要面向人类操作流程，不应放在 `docs/standards/` 或 `docs/ai/`。

## 目录结构

| 路径 | 类型 | 受众 | 作用 |
| --- | --- | --- | --- |
| `docs/ai/` | agent guide | agent | 给 AI / 自动化代理的规则、onboarding、Harness 原则 |
| `docs/standards/` | standard | agent | 约束性标准，描述”必须 / 不得 / 推荐”（含依赖分层、回归测试策略、预完成检查清单） |
| `docs/architecture/` | architecture | agent（人类少量阅读） | 只描述当前已经存在的系统结构与运行方式 |
| `docs/api/` | api | agent | 接口约定、响应格式、路由索引 |
| `docs/features/current/` | feature | agent | 当前已落地功能的模块级说明 |
| `docs/guides/` | guide | human | 面向人类工程师的过程型指南（本地开发、调试、测试、ESI 模块添加） |
| `docs/reference/` | reference | both | 离线参考资产，不作为当前实现的 source of truth |
| `docs/specs/draft/` | draft | both | 提案、未来增强、未完成设计 |
| `docs/templates/` | template | both | 新建文档时复用的模板 |

## 状态字段

所有新的规范性文档都应包含 front matter，并至少声明：

- `status`: `active` / `draft` / `deprecated` / `template`
- `doc_type`: `standard` / `architecture` / `api` / `feature` / `guide` / `reference` / `draft` / `template` / `index`
- `owner`
- `last_reviewed`

约定：

- `docs/templates/*` 使用 `status: template`，不要写成 `active`
- `docs/specs/draft/*` 使用 `status: draft`

## 文档更新规则

文档与行为在同一次变更中同步；归属、设计决策记录及维护规则见[文档治理规范](standards/documentation-governance.md)。按实际影响更新：

- **认证、鉴权、RBAC 职权提升或自动权限映射**：在[认证与权限](architecture/auth-and-permissions.md)记录变化的共享边界；模块文档说明该模块的影响，并链接共享规则。
- **路由或契约**：按 [API 约定](api/conventions.md#契约同步顺序)更新契约与生成物；[路由索引](api/route-index.md)只维护业务域入口映射，不重复记录路径和权限。模块有独立影响时，写明该影响并链接对应规则。
- **数据库表、核心关系或历史兼容列**：更新[数据库 schema 文档](architecture/database-schema.md)；其他架构、API 或模块文档有独立影响时，在相应文档补充并链接。

## 按任务查阅

- **上手或配置环境**：[仓库首页](../README.md)、[本地开发指南](guides/local-development.md)。代理按[任务导航](ai/agent-onboarding.md#change-routing)查阅相关文档。
- **定位实现或理解模块**：[模块图](architecture/module-map.md)、[当前功能索引](features/README.md)；跨模块变更可参考[架构概览](architecture/overview.md)。
- **核对工程边界**：[依赖分层](standards/dependency-layering.md)、[API 约定](api/conventions.md)、[认证与权限](architecture/auth-and-permissions.md)，按改动范围选读。
- **验证与排查**：[完成清单](standards/pre-completion-checklist.md)、[测试与验证](standards/testing-and-verification.md)、[调试指南](guides/debugging-guide.md)。
- **规划未落地设计**：查阅 `specs/draft/`；局部 `README.md` 用于补充模块实现细节。

## 军团公告

- [军团公告](features/current/announcements.md) 记录阅读版本、隐藏、图文存储和权限规则。

## 军团技能管理查询

- [仪表盘军团技能达标查询](features/current/dashboard-skill-readiness.md)：后端提供统计和技能明细，Vue 与 React 均使用生成客户端。

## 2.0.0 升级清理

- [手动清理 2.0.0 以前的残留](guides/pre-2.0-cleanup.md)

## QQ Bot 下线

- [当前下线边界](features/current/qq-group-governance.md)
- [PostgreSQL / Redis 清理与恢复](guides/qq-bot-retirement-cleanup.md)
- [待评审业务场景](specs/draft/qq-bot-business-scenarios.md)
- 历史资料（不代表当前能力）：[OneBot V11 设计](architecture/qqbot-dispatch.md)、[验证记录](guides/qqbot-verification.md)、[调度说明](features/current/qqbot.md)、[通知与群业务说明](features/current/qq-notifications.md)

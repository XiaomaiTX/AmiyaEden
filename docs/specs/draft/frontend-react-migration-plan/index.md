---
status: draft
doc_type: draft
owner: engineering
last_reviewed: 2026-07-30
source_of_truth:
  - static/src
  - static-react/src
  - docs/ai/repo-rules.md
  - docs/architecture/routing-and-menus.md
  - docs/architecture/auth-and-permissions.md
---

# 前端迁移计划草案入口（Vue3 -> React）

> 本页是入口页，只保留总览与导航。详细设计拆分到同目录子文档。

## 草案目标

- 迁移策略：`独立 React 重建 + Hash 路由 + 先保真后优化 UI`
- 部署策略：`Vue frontend` 与 `React frontend-react` 保持独立镜像并行部署；不在本草稿规划入口替换或 Vue 下线
- 目标技术栈：`Vite + React + TypeScript + React Router + shadcn/ui + Tailwind + Zustand`
- 当前前端现状：`Vue3 + Vue Router + Pinia + Element Plus + vue-i18n`

## 文档导航

- 迁移总清单：`./todolist.md`
- 组件替换列表：`./component-replacement-list.md`
- shadcn/ui 规范收敛债务：`./ui-standardization-backlog.md`
- 独立 React 替换迁移清单：`./coexistence-migration-todo.md`
- API 类型契约迁移方案：`./api-contract-migration-plan.md`

## 使用约定

- 入口页只维护范围、决策和链接，不维护长清单。
- 所有可执行项在子文档里维护，避免单文件过长和更新冲突。
- 子文档状态变更时同步更新 `last_reviewed`。
- React 迁移的 i18n 必须对齐旧 Vue 前端的实际翻译内容，尤其是人物管理、招新/直推、军团 KM、锁定提示等专有场景；只改调用形式不算完成对齐。
- 允许在 React 运行时补齐 Vue 既有的 `@:引用` 和变量插值能力，但翻译文本本身要以旧 Vue 语义和措辞为准，不得自行简化成通用描述。

## 模块进度摘要（2026-07-30 执行）

- 批次 A/B/C/D 的全部原计划路由已在 React 注册，并替换为真实业务页面。
- React 壳层的 `SidebarContext` 已从 `sidebar.tsx` 拆出为独立模块，侧边栏状态、移动端展开状态、cookie 记忆和快捷键切换属于已完成基座能力，不单独占用业务批次。
- Stage 0A 已完成当前已迁移 React 页面与 Vue 的 capability/menu/button parity；具体 AND/OR 语义和 reserved key 规则以 `docs/architecture/routing-and-menus.md` 与 `docs/features/current/corporation-access-policy.md` 为准。
- 冻结期后的 `/characters` 与 `/dashboard/fuel-officer-structures` 已完成 React 对齐。
- `/dashboard/galaxy-registry` 与 `/system/qq-governance` 已具备 React 路由、API、类型和基础业务页，但完整管理行为仍按 `./todolist.md` 的明确缺口继续追赶。
- `/fuxi-hall/{leadership, contributors, manage}` 与 `/info/tool-bookmarks` 已完成 React 对齐。
- Vue 侧已于 2026-05-12 移除 `hall-of-fame/*`，被 `fuxi-hall` 模块取代；React 已迁移 `fuxi-hall/*` 并移除三条历史遗留 stub。
- 基础设施已完成业务路由懒加载、启动 `/me` 刷新、资料/ESI 锁、WorkTab、`PermissionGate`/`RoleGate`、badge store/菜单徽章与 TanStack `DataTable`。其余 store 只在出现真实跨页状态需求时拆分，详见 `./component-replacement-list.md`。

## 明确声明

- 本文档组是提案草案，不代表当前已实现行为。
- Vue 与 React 的路由、菜单和权限实现必须完全隔离：不共享 manifest、运行时代码、组件、状态或源码读取脚本。迁移期的并行定义只在本目录维护；任何一端的变更先更新本 draft，再分别实施和验证。
- React 使用其自有的 `migration-routes.ts`、React Router、Zustand、Lucide 与 i18n；不复刻 Vue 的 `RouteRegistry`、`MenuProcessor` 或 `RouteTransformer` 处理链。

## 文档适配约定

- 迁移期间，`docs/architecture/`、`docs/api/`、`docs/standards/` 和 `docs/features/current/` 描述双端必须一致的行为，并在实现映射处注明 Vue/React 状态。
- Vue-only 的实现限制只能出现在明确的迁移阶段说明中，不得继续作为产品行为或通用工程规则。
- 当前 React capability/menu/button parity、主要壳层基座及 Galaxy Registry、QQ Governance 的完整业务同构已完成；后续工作聚焦全量跨角色回归、双端契约与文案对齐，以及 shadcn/ui 组件收敛，不将其重新描述为 0A 权限缺口。
- 功能文档的 React 状态统一引用 `migration-scope-baseline.md`，不在各 feature 文档维护第二套迁移清单。
- 不覆盖 `docs/ai/repo-rules.md`、`docs/architecture/`、`docs/api/`、`docs/features/current/` 的当前权威定义。
- 落地后转正路径：
  - 架构事实迁移到 `docs/architecture/`
  - 功能行为迁移到 `docs/features/current/`
  - 接口边界变化同步更新 `docs/api/`

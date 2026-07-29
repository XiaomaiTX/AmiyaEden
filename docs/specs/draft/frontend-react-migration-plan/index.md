---
status: draft
doc_type: draft
owner: engineering
last_reviewed: 2026-07-23
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
- 替换策略：`完成全部模块迁移与回归后，React 前端直接替换现有 Vue 前端`
- 目标技术栈：`Vite + React + TypeScript + React Router + shadcn/ui + Tailwind + Zustand`
- 当前前端现状：`Vue3 + Vue Router + Pinia + Element Plus + vue-i18n`

## 文档导航

- 迁移总清单：`./todolist.md`
- 组件替换列表：`./component-replacement-list.md`
- 独立 React 替换迁移清单：`./coexistence-migration-todo.md`
- API 类型契约迁移方案：`./api-contract-migration-plan.md`

## 使用约定

- 入口页只维护范围、决策和链接，不维护长清单。
- 所有可执行项在子文档里维护，避免单文件过长和更新冲突。
- 子文档状态变更时同步更新 `last_reviewed`。
- React 迁移的 i18n 必须对齐旧 Vue 前端的实际翻译内容，尤其是人物管理、招新/直推、军团 KM、锁定提示等专有场景；只改调用形式不算完成对齐。
- 允许在 React 运行时补齐 Vue 既有的 `@:引用` 和变量插值能力，但翻译文本本身要以旧 Vue 语义和措辞为准，不得自行简化成通用描述。

## 模块进度摘要（2026-07-22 审计）

- 批次 A/B/C/D 的全部原计划路由已在 React 注册，并替换为真实业务页面。
- React 壳层的 `SidebarContext` 已从 `sidebar.tsx` 拆出为独立模块，侧边栏状态、移动端展开状态、cookie 记忆和快捷键切换属于已完成基座能力，不单独占用业务批次。
- Stage 0A 已完成当前已迁移 React 页面与 Vue 的 capability/menu/button parity；具体 AND/OR 语义和 reserved key 规则以 `docs/architecture/routing-and-menus.md` 与 `docs/features/current/corporation-access-policy.md` 为准。
- Vue 侧在冻结期之后新增以下路由，React 侧尚未对齐，列入追赶清单（见 `./migration-scope-baseline.md`、`./todolist.md`）：
  - `/characters`（顶层路由，与 `dashboard/characters` 复用同一页，2026-05-22 落地）
  - `/dashboard/fuel-officer-structures`（2026-05-11 落地，`super_admin/fuel_officer`）
  - `/dashboard/galaxy-registry`（2026-06-04 落地，`super_admin/admin/captain/user`）
  - `/info/tool-bookmarks`（2026-05-13 落地）
  - `/system/qq-governance`（2026-07-12 落地，`super_admin`）
  - `/fuxi-hall/{leadership, contributors, manage}`（2026-05-12 落地，取代旧 `hall-of-fame` 模块）
- Vue 侧已于 2026-05-12 移除 `hall-of-fame/*`，被 `fuxi-hall` 模块取代；React 已迁移 `fuxi-hall/*` 并移除三条历史遗留 stub。
- 基础设施补齐仍未完成：WorkTab 多标签页、`v-auth`/`v-roles` 的 React 对应物（`PermissionGate`/`RoleGate`/`usePermission`/`useRole`）、Zustand `user/menu/worktab/setting/table/badge/sys-config` 等非 session/preference store 均未落地，详见 `./component-replacement-list.md`。

## 明确声明

- 本文档组是提案草案，不代表当前已实现行为。

## 文档适配约定

- 迁移期间，`docs/architecture/`、`docs/api/`、`docs/standards/` 和 `docs/features/current/` 描述双端必须一致的行为，并在实现映射处注明 Vue/React 状态。
- Vue-only 的实现限制只能出现在明确的迁移阶段说明中，不得继续作为产品行为或通用工程规则。
- 当前已迁移页面的 React capability/menu/button parity 已完成；替换 Vue 前仍需完成未迁移范围漂移页面，以及 WorkTab/KeepAlive、徽标和其他菜单基础设施。后续不要把这些剩余迁移项重新描述成 0A 权限缺口。
- 功能文档的 React 状态统一引用 `migration-scope-baseline.md`，不在各 feature 文档维护第二套迁移清单。
- 不覆盖 `docs/ai/repo-rules.md`、`docs/architecture/`、`docs/api/`、`docs/features/current/` 的当前权威定义。
- 落地后转正路径：
  - 架构事实迁移到 `docs/architecture/`
  - 功能行为迁移到 `docs/features/current/`
  - 接口边界变化同步更新 `docs/api/`

---
status: draft
doc_type: draft
owner: engineering
last_reviewed: 2026-05-01
source_of_truth:
  - static/src
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
- `hall-of-fame/temple` 当前取消实现，React 侧仅保留 stub 占位；后续重构另起范围后再恢复迁移。
- `C-1 shop` 已在 React 侧落地，后续波次继续推进 `skill-planning/*` 与 `operation/*`。
- `C-2` 已完成 `skill-planning/completion-check`、`skill-planning/skill-plans`、`skill-planning/personal-skill-plans`、`operation/join`、`operation/pap` 的 React 落地。
- `D-1` 已完成 `operation/fleets`、`operation/fleet-detail`、`operation/fleet-configs`、`operation/corporation-pap` 的 React 落地。
- `D-2` 已完成 `system/user`、`system/task-manager`、`system/wallet` 的 React 落地。

## 明确声明

- 本文档组是提案草案，不代表当前已实现行为。
- 不覆盖 `docs/ai/repo-rules.md`、`docs/architecture/`、`docs/api/`、`docs/features/current/` 的当前权威定义。
- 落地后转正路径：
  - 架构事实迁移到 `docs/architecture/`
  - 功能行为迁移到 `docs/features/current/`
  - 接口边界变化同步更新 `docs/api/`

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

- 迁移策略：`双栈渐进 + Hash 路由 + 先保真后优化 UI`
- 目标技术栈：`Vite + React + TypeScript + React Router + shadcn/ui + Tailwind + Zustand`
- 当前前端现状：`Vue3 + Vue Router + Pinia + Element Plus + vue-i18n`

## 文档导航

- 迁移总清单：`./todolist.md`
- 组件替换列表：`./component-replacement-list.md`
- Vue/React 初期共存迁移清单：`./coexistence-migration-todo.md`
- API 类型契约迁移方案：`./api-contract-migration-plan.md`

## 使用约定

- 入口页只维护范围、决策和链接，不维护长清单。
- 所有可执行项在子文档里维护，避免单文件过长和更新冲突。
- 子文档状态变更时同步更新 `last_reviewed`。

## 明确声明

- 本文档组是提案草案，不代表当前已实现行为。
- 不覆盖 `docs/ai/repo-rules.md`、`docs/architecture/`、`docs/api/`、`docs/features/current/` 的当前权威定义。
- 落地后转正路径：
  - 架构事实迁移到 `docs/architecture/`
  - 功能行为迁移到 `docs/features/current/`
  - 接口边界变化同步更新 `docs/api/`

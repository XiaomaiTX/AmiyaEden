---
status: draft
doc_type: draft
owner: engineering
last_reviewed: 2026-08-01
source_of_truth:
  - static/src
  - static-react/src
  - docker-compose.example.yml
  - docs/architecture/routing-and-menus.md
  - docs/architecture/auth-and-permissions.md
---

# Vue 与 React 双前端运行 Todo

## 运行事实

- Vue `frontend` 与 React `frontend-react` 是独立构建、验证和部署的双镜像服务。
- 本草稿不规划 Vue 下线、生产入口替换、回切开关或长期共存的结束日期。
- 两端不共享路由、菜单、状态或运行时代码；共同后端契约、权限语义和用户可见行为持续对齐。

## 已完成

- [x] React Hash 路由、404/500/403 与 SSO 登录闭环。
- [x] Token、会话、权限、按人物隔离的侧边栏菜单组展开状态、主题和徽章基础能力。
- [x] React 独立镜像、Compose `frontend-react` 服务及 verify/main/preview 工作流。
- [x] 全部计划内路由和冻结后的范围漂移页面均有 React 真实业务实现。

## 持续工作

- [ ] 按模块执行普通成员、管理员和受限角色的跨角色回归，并记录差异。
- [ ] 每次后端契约变更同步 Vue 与 React wrapper、类型、文案和调用页。
- [ ] 收敛 React 共享 shadcn/ui 基元与表格、表单、对话框模式。
- [ ] 维护 `migration-scope-baseline.md` 的路由/实现状态及原生组件债务清单。

## 不在本阶段实施

- CI UI 规范阻断。待现有原生组件债务显著收敛后，另行制定检查规则与启用条件。

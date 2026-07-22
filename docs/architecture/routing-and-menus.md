---
status: active
doc_type: architecture
owner: frontend
last_reviewed: 2026-07-22
source_of_truth:
  - static/src/router/core
  - static/src/router/routes
  - static/src/router/modules
  - static-react/src/app/migration-routes.ts
  - static-react/src/auth/route-access-gate.tsx
---

# 路由

## 前端路由模式

系统使用前端静态路由模式，所有路由定义在前端代码中。

## 前端路由源

Vue 静态模块主要位于：

- `static/src/router/modules/dashboard.ts`
- `static/src/router/modules/operation.ts`
- `static/src/router/modules/skill-planning.ts`
- `static/src/router/modules/info.ts`
- `static/src/router/modules/shop.ts`
- `static/src/router/modules/welfare.ts`
- `static/src/router/modules/newbro.ts`
- `static/src/router/modules/srp.ts`
- `static/src/router/modules/system.ts`

基础静态路由位于：

- `static/src/router/routes/staticRoutes.ts`

React 迁移侧的路由源是 `static-react/src/app/migration-routes.ts`，由 `static-react/src/app/router.tsx` 注册，并通过 `RouteAccessGate` 消费登录、角色和 `authList` 元数据。React 当前尚未完整承接 `corpCapabilities`、动态菜单处理、按钮级权限和 WorkTab/KeepAlive，这些是 Vue 下线前的阻断项。

两套前端必须遵守同一组行为约定：

- `meta.login = true` 对应 API / feature 文档中的 `Login`
- `meta.roles` 只表示显式职权白名单
- `meta.corpCapabilities` 表示军团能力策略要求；`super_admin` 在 capability 层自动放行
- `meta.requiresNewbro = true` 表示还要通过当前用户的新人大类资格快照检查
- 同一路由不要再用 `meta.roles` 伪装“任意非 guest 登录用户”
- guest 可访问的 onboarding / self-service 页面不要错误标成 `meta.login = true`，因为这会把它们提升为“非 guest 才可访问”
- `meta.isHide = true` 会同时从菜单树、水平菜单和全局搜索中移除该路由入口，但不等于删除路由本身

军团能力路由约定：

- 顶层业务菜单使用 `menu.*` 能力控制入口可见性。
- 管理页在菜单能力之外叠加对应管理能力，例如 `ticket.manage`、`shop.manage`、`system.manage`。
- 前端 capability 只做 UX 收敛；后端必须在对应业务路由上同步使用 `RequireCorpCapability`。
- 配置页展示 capability 时必须使用 i18n 标签，不直接展示 `srp.user`、`menu.srp` 等原始 key。

## 按钮权限

Vue 通过 `v-auth` 或权限 hook 消费按钮权限；React 应通过 `PermissionGate` / `usePermission` 等价能力消费同一 `meta.authList` 语义。当前 React 仅保存部分路由权限上下文，未完成的按钮门禁不得视为已对齐。

## 当前不变量

- `新人选队长` 仅受前端静态路由过滤影响，通过 `meta.requiresNewbro = true` 和后端返回的 `is_currently_newbro` 状态控制访问权限
- `队长管理` 页面允许 `captain` 进入只读页签，但不能因此绕过后端的管理权限边界
- `招新链接` 页面为 `admin` / `super_admin` 追加 `全部链接` 与 `链接设置` tab，但不新增独立系统管理路由
- `导师奖励阶段` 不再是独立前端路由；管理员通过 `新人帮扶 / 导师管理` 页面的 `设置奖励阶段` tab 进入该管理能力，路由权限仍由 `newbro/mentor-manage` 的 `meta.roles` 控制
- `伏羲中心 / Fuxi Center` 当前被前端路由标记为 `meta.isHide = true`，因此暂时不在导航和全局搜索中展示，但直达路由仍保留
- 路由改动若涉及权限边界，必须同步更新 API / feature 文档
- 路由架构说明只维护在 `docs/` 中

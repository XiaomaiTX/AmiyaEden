---
status: active
doc_type: architecture
owner: frontend
last_reviewed: 2026-07-23
source_of_truth:
  - static/src/router/core
  - static/src/router/routes
  - static/src/router/modules
  - static/src/hooks/core/useCorpCapability.ts
  - static-react/src/app/migration-routes.ts
  - static-react/src/app/route-access.ts
  - static-react/src/auth/route-access-gate.tsx
  - static-react/src/hooks/use-corp-capability.ts
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

React 迁移侧的路由源是 `static-react/src/app/migration-routes.ts`，由 `static-react/src/app/router.tsx` 注册，并通过 `RouteAccessGate` 消费登录、角色、`authList` 和 `corpCapabilitiesAll` / `corpCapabilitiesAny` 元数据。Vue 与 React 共用同一份 enforced capability 目录与同一套 AND/OR 语义。

两套前端必须遵守同一组行为约定：

- `meta.login = true` 对应 API / feature 文档中的 `Login`
- `meta.roles` 只表示显式职权白名单
- `meta.corpCapabilitiesAll = string[]`：AND 语义，要求用户命中每一个 capability
- `meta.corpCapabilitiesAny = string[]`：OR 语义，要求用户至少命中其中一个 capability
- 旧 `meta.corpCapabilities` 已废弃；Stage 0A 之后新路由必须显式选择 All 或 Any，避免单一字段在不同路由上承担不同语义
- `super_admin` 在 capability 层自动放行，与后端 `RequireCorpCapability` 对齐
- `meta.requiresNewbro = true` 表示还要通过当前用户的新人大类资格快照检查
- 同一路由不要再用 `meta.roles` 伪装"任意非 guest 登录用户"
- guest 可访问的 onboarding / self-service 页面不要错误标成 `meta.login = true`，因为这会把它们提升为"非 guest 才可访问"
- `meta.isHide = true` 会同时从菜单树、水平菜单和全局搜索中移除该路由入口，但不等于删除路由本身

军团能力路由约定：

- 顶层业务菜单仅在后端路由确实强制对应 `menu.*` 时使用该能力控制入口可见性；没有已强制菜单 key 的域必须以其有效子能力作 OR 门禁。例如 SRP 使用 `srp.user` / `srp.manage`，系统管理使用各系统子能力，不能使用 reserved 的 `menu.srp` / `menu.system` 等键。
- 管理页在菜单能力之外叠加对应管理能力，例如 `ticket.manage`、`shop.manage`、`system.manage`。
- 需要多份管理能力同时满足时（如 `shop.manage + shop.admin.product.manage`）必须用 `corpCapabilitiesAll` 显式声明 AND，而不是依赖隐式行为。
- 前端 capability 只做 UX 收敛；后端必须在对应业务路由上同步使用 `RequireCorpCapability`。
- 配置页展示 capability 时必须使用 i18n 标签，不直接展示 `srp.user`、`menu.srp` 等原始 key。
- 后端 enforced 目录变更时，必须同时更新 Vue 与 React 的 `migration-routes.ts` / `modules/*.ts`，避免双前端能力门禁漂移。
- reserved capability 不得用于路由、菜单或按钮门禁；否则普通用户无法通过策略获得该 key，`full_access` 用户的 `/me` 也不会收到它。

## 按钮权限

按钮级权限（下单、回复工单、运行任务、调整钱包、导出审计）必须用 `useCorpCapability` hook 独立 gate，不能只靠路由 meta。Vue 与 React 各自的实现：

- Vue：`static/src/hooks/core/useCorpCapability.ts`
- React：`static-react/src/hooks/use-corp-capability.ts`

两个 hook 暴露相同的三个方法：`hasCapability`、`hasAllCapabilities`、`hasAnyCapability`，且都 `super_admin` 短路放行。Stage 0A 已落地的 5 个按钮 gate：

| 动作 | capability | 说明 |
| --- | --- | --- |
| 商城下单 | `shop.order.create` | 路由只要求 `shop.order.read_self` |
| 工单回复 | `ticket.user.reply` | 路由只要求 `ticket.user.create` |
| 手动运行任务 | `system.task.run` | 路由只要求 `system.task.read` |
| 调整伏羲币余额 | `system.wallet.adjust` | 路由只要求 `system.wallet.read` |
| 审计日志导出 | `system.audit.export` | 路由只要求 `system.audit.read` |

Vue 通过 `v-auth` 或权限 hook 消费 `meta.authList` 语义；Stage 0A 已迁移页面的 React 按钮 capability 通过 `useCorpCapability` 消费同一套能力语义。通用 `PermissionGate` / `usePermission` 仍属于后续迁移基础设施，不再作为 0A capability parity 缺口。

## 当前不变量

- `新人选队长` 仅受前端静态路由过滤影响，通过 `meta.requiresNewbro = true` 和后端返回的 `is_currently_newbro` 状态控制访问权限
- `队长管理` 页面允许 `captain` 进入只读页签，但不能因此绕过后端的管理权限边界
- `招新链接` 页面为 `admin` / `super_admin` 追加 `全部链接` 与 `链接设置` tab，但不新增独立系统管理路由
- `导师奖励阶段` 不再是独立前端路由；管理员通过 `新人帮扶 / 导师管理` 页面的 `设置奖励阶段` tab 进入该管理能力，路由权限仍由 `newbro/mentor-manage` 的 `meta.roles` 控制
- `伏羲中心 / Fuxi Center` 当前被前端路由标记为 `meta.isHide = true`，因此暂时不在导航和全局搜索中展示，但直达路由仍保留
- 路由改动若涉及权限边界，必须同步更新 API / feature 文档
- 路由架构说明只维护在 `docs/` 中

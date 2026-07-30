---
status: draft
doc_type: draft
owner: engineering
last_reviewed: 2026-07-30
source_of_truth:
  - static/src/views
  - static/src/router
  - static/src/api
  - static-react/src
  - docs/specs/draft/frontend-react-migration-plan/todolist.md
---

# 迁移范围基线清单（冻结 + 漂移追赶）

## 冻结规则

- 冻结日期：2026-05-01
- 冻结来源：`static/src/router/modules/*.ts` + `static/src/router/routes/staticRoutes.ts`
- 冻结口径：仅统计“页面级路由组件”（`component: '/xxx/yyy'` 或 `@views/.../index.vue`）
- owner 规则：当前统一标记为 `FE-owner(TBD)`，按批次启动前补齐到个人
- 漂移规则：2026-05-01 之后 Vue 侧新增或删除的路由单独列入“范围漂移追赶项”，不修改原冻结表

## 页面范围清单（按迁移批次）

`React 状态` 列以 2026-07-22 审计结果为准，取值：
- `真实页`：React 侧已有真实业务页实现
- `部分对齐`：React 侧已有可运行真实页，但仍缺 Vue 当前行为的一部分
- `stub`：React 侧仅有占位 stub
- `未对齐`：Vue 侧有路由，React 侧未注册或未实现

| 批次 | Vue 页面组件 | 路由路径 | 优先级 | 依赖 API（主） | 权限/约束 | React 状态 | owner |
|---|---|---|---|---|---|---|---|
| A | `/dashboard/console` | `/dashboard/console` | P1 | `dashboard.ts`, `notification.ts` | `login` | 真实页 | FE-owner(TBD) |
| A | `/dashboard/npc-kills` | `/dashboard/npc-kills` | P1 | `npc-kill.ts` | `roles: super_admin/admin` | 真实页 | FE-owner(TBD) |
| A | `/dashboard/corporation-structures` | `/dashboard/corporation-structures` | P1 | `corporation-structures.ts` | `roles: super_admin/admin` | 真实页 | FE-owner(TBD) |
| A | `/info/wallet` | `/info/wallet` | P1 | `eve-info.ts` | `login` | 真实页 | FE-owner(TBD) |
| A | `/info/skill` | `/info/skill` | P1 | `eve-info.ts` | `login` | 真实页 | FE-owner(TBD) |
| A | `/info/npc-kills` | `/info/npc-kills` | P1 | `npc-kill.ts` | `login` | 真实页 | FE-owner(TBD) |
| A | `/info/ships` | `/info/ships` | P1 | `eve-info.ts` | `login` | 真实页 | FE-owner(TBD) |
| A | `/info/implants` | `/info/implants` | P1 | `eve-info.ts` | `login` | 真实页 | FE-owner(TBD) |
| A | `/info/fittings` | `/info/fittings` | P1 | `eve-info.ts` | `login` | 真实页 | FE-owner(TBD) |
| A | `/info/assets` | `/info/assets` | P1 | `eve-info.ts` | `login` | 真实页 | FE-owner(TBD) |
| A | `/info/contracts` | `/info/contracts` | P1 | `eve-info.ts` | `login` | 真实页 | FE-owner(TBD) |
| A | `/info/esi-check` | `/info/esi-check` | P1 | `eve-info.ts`, `esi-refresh.ts` | `login` | 真实页 | FE-owner(TBD) |
| B | `/ticket/my-tickets` | `/ticket/my-tickets` | P1 | `ticket.ts` | `login` | 真实页 | FE-owner(TBD) |
| B | `/ticket/create` | `/ticket/create` | P1 | `ticket.ts`, `upload.ts` | `login` | 真实页 | FE-owner(TBD) |
| B | `/ticket/detail` | `/ticket/detail/:id` | P1 | `ticket.ts` | `login` | 真实页 | FE-owner(TBD) |
| B | `/system/ticket-management` | `/ticket/management` | P1 | `ticket.ts` | `roles: super_admin/admin` | 真实页 | FE-owner(TBD) |
| B | `/system/ticket-categories` | `/ticket/categories` | P1 | `ticket.ts` | `roles: super_admin/admin` | 真实页 | FE-owner(TBD) |
| B | `/system/ticket-statistics` | `/ticket/statistics` | P1 | `ticket.ts` | `roles: super_admin/admin` | 真实页 | FE-owner(TBD) |
| B | `/system/ticket-detail` | `/ticket/admin-detail/:id` | P1 | `ticket.ts` | `roles: super_admin/admin` | 真实页 | FE-owner(TBD) |
| B | `/welfare/my` | `/welfare/my` | P1 | `welfare.ts` | `login` | 真实页 | FE-owner(TBD) |
| B | `/welfare/approval` | `/welfare/approval` | P1 | `welfare.ts` | `roles: super_admin/admin/welfare` | 真实页 | FE-owner(TBD) |
| B | `/welfare/settings` | `/welfare/settings` | P1 | `welfare.ts` | `roles: super_admin/admin/welfare` | 真实页 | FE-owner(TBD) |
| B | `/newbro/select-captain` | `/newbro/select-captain` | P1 | `newbro.ts` | `login`, `requiresNewbro` | 真实页 | FE-owner(TBD) |
| B | `/newbro/select-mentor` | `/newbro/select-mentor` | P1 | `newbro.ts`, `mentor.ts` | `login`, `requiresMentorMenteeEligibility` | 真实页 | FE-owner(TBD) |
| B | `/newbro/captain` | `/newbro/captain` | P1 | `newbro.ts` | `roles: captain` | 真实页 | FE-owner(TBD) |
| B | `/newbro/mentor` | `/newbro/mentor` | P1 | `mentor.ts` | `roles: mentor` | 真实页 | FE-owner(TBD) |
| B | `/newbro/manage` | `/newbro/manage` | P1 | `newbro.ts` | `roles: super_admin/admin/captain` | 真实页 | FE-owner(TBD) |
| B | `/newbro/mentor-manage` | `/newbro/mentor-manage` | P1 | `mentor.ts` | `roles: super_admin/admin` | 真实页 | FE-owner(TBD) |
| B | `/newbro/recruit-link` | `/newbro/recruit-link` | P1 | `newbro.ts` | `login` | 真实页 | FE-owner(TBD) |
| B | `/srp/apply` | `/srp/srp-apply` | P1 | `srp.ts` | `login` | 真实页 | FE-owner(TBD) |
| B | `/srp/manage` | `/srp/srp-manage` | P1 | `srp.ts` | `roles: super_admin/admin/senior_fc/srp`, `auth: approve` | 真实页 | FE-owner(TBD) |
| B | `/srp/prices` | `/srp/srp-prices` | P1 | `srp.ts` | `roles: super_admin/admin/senior_fc/srp` | 真实页 | FE-owner(TBD) |
| C | `/shop/browse` | `/shop/browse` | P1 | `shop.ts` | `login` | 真实页 | FE-owner(TBD) |
| C | `/shop/manage` | `/shop/manage` | P1 | `shop.ts` | `roles: super_admin/admin`, `auth: add_product/edit_product/delete_product` | 真实页 | FE-owner(TBD) |
| C | `/shop/order-manage` | `/shop/order-manage` | P1 | `shop.ts` | `roles: super_admin/admin/shop_order_manage`, `auth: approve_order` | 真实页 | FE-owner(TBD) |
| C | `/shop/wallet` | `/shop/wallet` | P1 | `shop.ts` | `login` | 真实页 | FE-owner(TBD) |
| C | `/skill-planning/completion-check` | `/skill-planning/completion-check` | P1 | `skill-plan.ts` | `login` | 真实页 | FE-owner(TBD) |
| C | `/skill-planning/skill-plans` | `/skill-planning/skill-plans` | P1 | `skill-plan.ts` | `login` | 真实页 | FE-owner(TBD) |
| C | `/skill-planning/personal-skill-plans` | `/skill-planning/personal-skill-plans` | P1 | `skill-plan.ts` | `login` | 真实页 | FE-owner(TBD) |
| C | `/operation/join` | `/operation/join` | P1 | `fleet.ts` | `login` | 真实页 | FE-owner(TBD) |
| C | `/operation/pap` | `/operation/pap` | P1 | `alliance-pap.ts` | `login` | 真实页 | FE-owner(TBD) |
| D | `/operation/fleets` | `/operation/fleets` | P1 | `fleet.ts` | `roles: super_admin/admin/fc/senior_fc` | 真实页 | FE-owner(TBD) |
| D | `/operation/fleet-detail` | `/operation/fleet-detail/:id` | P1 | `fleet.ts` | `roles: super_admin/admin/fc/senior_fc` | 真实页 | FE-owner(TBD) |
| D | `/operation/fleet-configs` | `/operation/fleet-configs` | P1 | `fleet-config.ts` | `login` | 真实页 | FE-owner(TBD) |
| D | `/operation/corporation-pap` | `/operation/corporation-pap` | P1 | `alliance-pap.ts` | `login` | 真实页 | FE-owner(TBD) |
| D | `/system/user` | `/system/user` | P1 | `system-manage.ts` | `roles: super_admin/admin`, `auth: delete_user/assign_role` | 真实页 | FE-owner(TBD) |
| D | `/system/task-manager` | `/system/task-manager` | P1 | `task-manager.ts` | `roles: super_admin/admin`, `auth: execute_task/update_schedule` | 真实页 | FE-owner(TBD) |
| D | `/system/wallet` | `/system/wallet` | P1 | `sys-wallet.ts` | `roles: super_admin/admin`, `auth: adjust_balance/view_log` | 真实页 | FE-owner(TBD) |
| D | `/system/audit` | `/system/audit` | P1 | `audit.ts` | `roles: super_admin/admin`, `auth: view_audit_detail` | 真实页 | FE-owner(TBD) |
| D | `/system/pap-exchange` | `/system/pap-exchange` | P1 | `pap-exchange.ts` | `roles: super_admin/admin`, `auth: edit_exchange_rate` | 真实页 | FE-owner(TBD) |
| D | `/system/pap` | `/system/pap` | P1 | `alliance-pap.ts` | `roles: super_admin/admin`, `auth: manual_fetch` | 真实页 | FE-owner(TBD) |
| D | `/system/auto-role` | `/system/auto-role` | P1 | `system-manage.ts` | `roles: super_admin` | 真实页 | FE-owner(TBD) |
| D | `/system/user-center` | `/system/user-center` | P1 | `system-manage.ts` | `isHide`, `isHideTab` | 真实页 | FE-owner(TBD) |
| D | `/system/webhook` | `/system/webhook` | P1 | `webhook.ts` | `roles: super_admin` | 真实页 | FE-owner(TBD) |
| D | `/system/basic-config` | `/system/basic-config` | P1 | `sys-config.ts` | `roles: super_admin` | 真实页 | FE-owner(TBD) |
| 收尾 | `@views/auth/login/index.vue` | `/auth/login` | P1 | `auth.ts` | 静态路由 | 真实页 | FE-owner(TBD) |
| 收尾 | `@views/auth/callback/index.vue` | `/auth/callback` | P1 | `auth.ts` | 静态路由 | 真实页 | FE-owner(TBD) |
| 收尾 | `@views/auth/recruit/index.vue` | `/r/:code` | P1 | `auth.ts` | 静态路由 | 真实页 | FE-owner(TBD) |
| 收尾 | `@views/outside/Iframe.vue` | `/outside/iframe/:path` | P2 | 无后端依赖（外链） | 静态路由 | 真实页 | FE-owner(TBD) |
| 收尾 | `@views/exception/403/index.vue` | `/403` | P2 | 无 | 静态路由 | 真实页 | FE-owner(TBD) |
| 收尾 | `@views/exception/404/index.vue` | `/:pathMatch(.*)*` | P2 | 无 | 静态路由 | 真实页 | FE-owner(TBD) |
| 收尾 | `@views/exception/500/index.vue` | `/500` | P2 | 无 | 静态路由 | 真实页 | FE-owner(TBD) |

## 范围漂移追赶项（2026-05-01 冻结后新增/删除）

### Vue 侧新增路由（React 侧追赶状态）

| Vue 页面组件 | 路由路径 | Vue 落地日期 | 依赖 API（主） | 权限/约束 | React 状态 | owner |
|---|---|---|---|---|---|---|
| 人物管理 | `/characters`（顶层） | 2026-05-22 | `auth.ts` | `JWT`（含 guest） | 真实页 | FE-owner(TBD) |
| `/dashboard/fuel-officer-structures` | `/dashboard/fuel-officer-structures` | 2026-05-11 | `corporation-structures.ts` | `roles: super_admin/fuel_officer` | 真实页 | FE-owner(TBD) |
| `/dashboard/galaxy-registry` | `/dashboard/galaxy-registry` | 2026-06-04 | `galaxy-registry.ts` | `roles: super_admin/admin/captain/user` | 部分对齐 | FE-owner(TBD) |
| `/info/tool-bookmarks` | `/info/tool-bookmarks` | 2026-05-13 | `tool-bookmarks.ts` | `login`, `corpCapabilitiesAny: menu.info` | 真实页 | FE-owner(TBD) |
| `/system/qq-governance` | `/system/qq-governance` | 2026-07-12 | `qq-governance.ts` | `roles: super_admin` | 部分对齐 | FE-owner(TBD) |
| `/fuxi-hall/leadership` | `/fuxi-hall/leadership` | 2026-05-12 | `fuxi-hall.ts` | `login` | 真实页 | FE-owner(TBD) |
| `/fuxi-hall/contributors` | `/fuxi-hall/contributors` | 2026-05-12 | `fuxi-hall.ts` | `login` | 真实页 | FE-owner(TBD) |
| `/fuxi-hall/manage` | `/fuxi-hall/manage` | 2026-05-12 | `fuxi-hall.ts` | `roles: super_admin/admin` | 真实页 | FE-owner(TBD) |

### Vue 侧已删除（React 侧仍为遗留 stub）

| React 遗留 stub 路径 | Vue 删除日期 | 处理动作 |
|---|---|---|
| `hall-of-fame/temple` | 2026-05-12 | 已移除 |
| `hall-of-fame/manage` | 2026-05-12 | 已移除 |
| `hall-of-fame/current-manage` | 2026-05-12 | 已移除 |

## 当前进展

- 2026-05-01：static-react 已完成批次 A 全部路由注册与占位页接入，且对关键角色门禁完成测试回归。
- 2026-05-01 至 2026-05-10：批次 A/B/C/D 全部原计划路由完成 React 真实页迁移，壳层、i18n、SSO 登录闭环、暗色模式、API 类型本地化同步落地。
- 2026-05-13：`/dashboard/corporation-structures` 增加 fuel officer 列展示（Vue/React 同步）。
- 2026-06-05：`/info/npc-kills` 移除误导性的 estimated hours 指标。
- 2026-06-06：`/info/assets` 增加分页懒加载与显式错误态。
- 2026-06-29：`/info/npc-kills` 增加 incursion/mission reward、统一筛选与按用户/人物筛选；payout 类型改名。
- 2026-07-22：本轮审计确认所有原计划路由在 React 侧均有真实业务页实现，`hall-of-fame/*` 三条 stub 成为历史遗留 stub；范围漂移追赶项（8 条新增 Vue 路由）尚未对齐。
- 2026-07-29：React 完成 `fuxi-hall/*` 三页迁移并移除三条 `hall-of-fame/*` stub；完成 `/info/tool-bookmarks` 迁移后，剩余范围漂移页面为 4 条路由域。
- 2026-07-30：React 完成 `/characters`、fuel officer 页面和迁移基础设施；Galaxy Registry 与 QQ Governance 建立可运行基础页，完整业务同构仍未关闭。

## 说明与已知风险

- 本清单冻结的是“路由页面范围”，不含路由内子组件与纯工具文件（`*.helpers.ts`、`*.test.ts`）。
- `role/*` 在 `static/src/router/modules/role.ts` 存在但未被 `modules/index.ts` 引用，对应 `views/role/*` 不存在，视为死代码，不纳入迁移范围。
- `views/auth/register/index.vue` 无对应路由，属于模板遗留，不纳入迁移范围。
- Vue 侧 `hall-of-fame/*` 已整体删除并由 `fuxi-hall` 取代；React 已移除对应 stub，旧路径按未注册路由处理。
- Vue 工单管理页面组件位于 `views/system/*`，路由暴露在 `/ticket/*` 下；迁移时组件路径与路由路径不一致，需要单独维护映射。
- Vue 静态路由与 `exception` 模块同名（`Exception403/404/500`），静态路由先生效；React 侧已通过扁平 `RouteAccessGate` + 静态路由表实现等价行为。
- 批次执行前必须补齐 owner，并基于当前后端接口再确认 API 依赖是否存在跨模块调用。

## 文档适配矩阵（2026-07-22）

本矩阵只维护“文档是否已正确表达双端约束”，不重复页面迁移清单。页面、路由和 React 状态仍以本文上方的范围表及 `todolist.md` 为准。

| 文档域 | Vue 当前实现 | React 当前体现 | 适配状态 | 切换阻断 |
|---|---|---|---|---|
| 认证与人物 | `static/src/api/auth.ts`、Vue router guard | `static-react/src/api/auth.ts`、`session-store`、`RouteAccessGate` | 已建立双端映射 | capability、完整锁定流程需回归 |
| 路由与菜单 | `static/src/router`、`MenuProcessor` | 路由懒加载、`RouteAccessGate`、WorkTab、badge 菜单汇总 | 已建立等价基座 | 完整跨角色回归 |
| 按钮权限 | `v-auth` / `v-roles` | `PermissionGate` / `RoleGate` 与 hooks | 已对齐 | 新增使用点持续回归 |
| 状态管理 | Pinia 多 store | Zustand `session/preference/worktab/badge`，其余按真实跨页需求拆分 | 部分对齐 | 运行时 sys-config 等尚未迁移 |
| API 契约 | `static/src/api` + `api.d.ts` | `static-react/src/api` + 模块化本地类型 | 已建立双端契约 | 每个迁移模块分别通过类型和契约检查 |
| i18n | Vue `zh/en` locale | `static-react/src/i18n/messages` | 已建立双端映射 | 文案语义、插值和 `@:引用` 对齐 |
| 表格/卡片布局 | `ArtTable`、Element Plus、Vue overflow 规则 | TanStack `DataTable` 基座 + 存量页面按需迁移 | 部分对齐 | 存量复杂表格渐进迁移、暗色主题回归 |
| 格式化 helper | Vue ISK/time helper | React ISK helper；时间 helper 待统一 | 部分对齐 | 禁止页面本地格式化变体 |
| 测试与交付 | `vue-tsc`、`test:unit`、Vue build | `tsc -b`、`test`、API contract、React build | 已纳入规范 | 双端命令和模块回归均可执行 |
| 范围漂移页面 | Vue 已落地 | characters/fuel officer 已完成；galaxy registry/QQ governance 基础页已落地 | 进行中 | Galaxy Registry 与 QQ Governance 完整业务同构 |

文档适配完成定义：所有 active 规范使用行为级表述；每个 current feature doc 通过本文引用迁移状态；所有 Vue-only 限制都明确标注为过渡期限制或历史归档内容。

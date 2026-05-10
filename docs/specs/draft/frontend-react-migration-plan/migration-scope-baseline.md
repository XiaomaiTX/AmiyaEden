---
status: draft
doc_type: draft
owner: engineering
last_reviewed: 2026-05-01
source_of_truth:
  - static/src/views
  - static/src/router
  - static/src/api
  - docs/specs/draft/frontend-react-migration-plan/todolist.md
---

# 迁移范围基线清单（冻结）

## 冻结规则

- 冻结日期：2026-05-01
- 冻结来源：`static/src/router/modules/*.ts` + `static/src/router/routes/staticRoutes.ts`
- 冻结口径：仅统计“页面级路由组件”（`component: '/xxx/yyy'` 或 `@views/.../index.vue`）
- owner 规则：当前统一标记为 `FE-owner(TBD)`，按批次启动前补齐到个人

## 页面范围清单（按迁移批次）

| 批次 | Vue 页面组件 | 路由路径 | 优先级 | 依赖 API（主） | 权限/约束 | owner |
|---|---|---|---|---|---|---|
| A | `/dashboard/console` | `/dashboard/console` | P1 | `dashboard.ts`, `notification.ts` | `login` | FE-owner(TBD) |
| A | `/dashboard/characters` | `/dashboard/characters` | P1 | `dashboard.ts` | `login` | FE-owner(TBD) |
| A | `/dashboard/npc-kills` | `/dashboard/npc-kills` | P1 | `npc-kill.ts` | `roles: super_admin/admin` | FE-owner(TBD) |
| A | `/dashboard/corporation-structures` | `/dashboard/corporation-structures` | P1 | `corporation-structures.ts` | `roles: super_admin/admin` | FE-owner(TBD) |
| A | `/info/wallet` | `/info/wallet` | P1 | `eve-info.ts` | `login` | FE-owner(TBD) |
| A | `/info/skill` | `/info/skill` | P1 | `eve-info.ts` | `login` | FE-owner(TBD) |
| A | `/info/npc-kills` | `/info/npc-kills` | P1 | `npc-kill.ts` | `login` | FE-owner(TBD) |
| A | `/info/ships` | `/info/ships` | P1 | `eve-info.ts` | `login` | FE-owner(TBD) |
| A | `/info/implants` | `/info/implants` | P1 | `eve-info.ts` | `login` | FE-owner(TBD) |
| A | `/info/fittings` | `/info/fittings` | P1 | `eve-info.ts` | `login` | FE-owner(TBD) |
| A | `/info/assets` | `/info/assets` | P1 | `eve-info.ts` | `login` | FE-owner(TBD) |
| A | `/info/contracts` | `/info/contracts` | P1 | `eve-info.ts` | `login` | FE-owner(TBD) |
| A | `/info/esi-check` | `/info/esi-check` | P1 | `eve-info.ts`, `esi-refresh.ts` | `login` | FE-owner(TBD) |
| A | `/hall-of-fame/temple` | `/hall-of-fame/temple` | P1 | `hall-of-fame.ts` | `login` | FE-owner(TBD) |
| B | `/ticket/my-tickets` | `/ticket/my-tickets` | P1 | `ticket.ts` | `login` | FE-owner(TBD) |
| B | `/ticket/create` | `/ticket/create` | P1 | `ticket.ts`, `upload.ts` | `login` | FE-owner(TBD) |
| B | `/ticket/detail` | `/ticket/detail/:id` | P1 | `ticket.ts` | `login` | FE-owner(TBD) |
| B | `/system/ticket-management` | `/ticket/management` | P1 | `ticket.ts` | `roles: super_admin/admin` | FE-owner(TBD) |
| B | `/system/ticket-categories` | `/ticket/categories` | P1 | `ticket.ts` | `roles: super_admin/admin` | FE-owner(TBD) |
| B | `/system/ticket-statistics` | `/ticket/statistics` | P1 | `ticket.ts` | `roles: super_admin/admin` | FE-owner(TBD) |
| B | `/system/ticket-detail` | `/ticket/admin-detail/:id` | P1 | `ticket.ts` | `roles: super_admin/admin` | FE-owner(TBD) |
| B | `/welfare/my` | `/welfare/my` | P1 | `welfare.ts` | `login` | FE-owner(TBD) |
| B | `/welfare/approval` | `/welfare/approval` | P1 | `welfare.ts` | `roles: super_admin/admin/welfare` | FE-owner(TBD) |
| B | `/welfare/settings` | `/welfare/settings` | P1 | `welfare.ts` | `roles: super_admin/admin/welfare` | FE-owner(TBD) |
| B | `/newbro/select-captain` | `/newbro/select-captain` | P1 | `newbro.ts` | `login`, `requiresNewbro` | FE-owner(TBD) |
| B | `/newbro/select-mentor` | `/newbro/select-mentor` | P1 | `newbro.ts`, `mentor.ts` | `login`, `requiresMentorMenteeEligibility` | FE-owner(TBD) |
| B | `/newbro/captain` | `/newbro/captain` | P1 | `newbro.ts` | `roles: captain` | FE-owner(TBD) |
| B | `/newbro/mentor` | `/newbro/mentor` | P1 | `mentor.ts` | `roles: mentor` | FE-owner(TBD) |
| B | `/newbro/manage` | `/newbro/manage` | P1 | `newbro.ts` | `roles: super_admin/admin/captain` | FE-owner(TBD) |
| B | `/newbro/mentor-manage` | `/newbro/mentor-manage` | P1 | `mentor.ts` | `roles: super_admin/admin` | FE-owner(TBD) |
| B | `/newbro/recruit-link` | `/newbro/recruit-link` | P1 | `newbro.ts` | `login` | FE-owner(TBD) |
| B | `/srp/apply` | `/srp/srp-apply` | P1 | `srp.ts` | `login` | FE-owner(TBD) |
| B | `/srp/manage` | `/srp/srp-manage` | P1 | `srp.ts` | `roles: super_admin/admin/senior_fc/srp`, `auth: approve` | FE-owner(TBD) |
| B | `/srp/prices` | `/srp/srp-prices` | P1 | `srp.ts` | `roles: super_admin/admin/senior_fc/srp` | FE-owner(TBD) |
| C | `/shop/browse` | `/shop/browse` | P1 | `shop.ts` | `login` | FE-owner(TBD) |
| C | `/shop/manage` | `/shop/manage` | P1 | `shop.ts` | `roles: super_admin/admin`, `auth: add_product/edit_product/delete_product` | FE-owner(TBD) |
| C | `/shop/order-manage` | `/shop/order-manage` | P1 | `shop.ts` | `roles: super_admin/admin/shop_order_manage`, `auth: approve_order` | FE-owner(TBD) |
| C | `/shop/wallet` | `/shop/wallet` | P1 | `shop.ts` | `login` | FE-owner(TBD) |
| C | `/skill-planning/completion-check` | `/skill-planning/completion-check` | P1 | `skill-plan.ts` | `login` | FE-owner(TBD) |
| C | `/skill-planning/skill-plans` | `/skill-planning/skill-plans` | P1 | `skill-plan.ts` | `login` | FE-owner(TBD) |
| C | `/skill-planning/personal-skill-plans` | `/skill-planning/personal-skill-plans` | P1 | `skill-plan.ts` | `login` | FE-owner(TBD) |
| C | `/operation/join` | `/operation/join` | P1 | `fleet.ts` | `login` | FE-owner(TBD) |
| C | `/operation/pap` | `/operation/pap` | P1 | `alliance-pap.ts` | `login` | FE-owner(TBD) |
| D | `/operation/fleets` | `/operation/fleets` | P1 | `fleet.ts` | `roles: super_admin/admin/fc/senior_fc` | FE-owner(TBD) |
| D | `/operation/fleet-detail` | `/operation/fleet-detail/:id` | P1 | `fleet.ts` | `roles: super_admin/admin/fc/senior_fc` | FE-owner(TBD) |
| D | `/operation/fleet-configs` | `/operation/fleet-configs` | P1 | `fleet-config.ts` | `login` | FE-owner(TBD) |
| D | `/operation/corporation-pap` | `/operation/corporation-pap` | P1 | `alliance-pap.ts` | `login` | FE-owner(TBD) |
| D | `/system/user` | `/system/user` | P1 | `system-manage.ts` | `roles: super_admin/admin`, `auth: delete_user/assign_role` | FE-owner(TBD) |
| D | `/system/task-manager` | `/system/task-manager` | P1 | `task-manager.ts` | `roles: super_admin/admin`, `auth: execute_task/update_schedule` | FE-owner(TBD) |
| D | `/system/wallet` | `/system/wallet` | P1 | `sys-wallet.ts` | `roles: super_admin/admin`, `auth: adjust_balance/view_log` | FE-owner(TBD) |
| D | `/system/audit` | `/system/audit` | P1 | `audit.ts` | `roles: super_admin/admin`, `auth: view_audit_detail` | FE-owner(TBD) |
| D | `/system/pap-exchange` | `/system/pap-exchange` | P1 | `pap-exchange.ts` | `roles: super_admin/admin`, `auth: edit_exchange_rate` | FE-owner(TBD) |
| D | `/system/pap` | `/system/pap` | P1 | `alliance-pap.ts` | `roles: super_admin/admin`, `auth: manual_fetch` | FE-owner(TBD) |
| D | `/system/auto-role` | `/system/auto-role` | P1 | `system-manage.ts` | `roles: super_admin` | FE-owner(TBD) |
| D | `/system/user-center` | `/system/user-center` | P1 | `system-manage.ts` | `isHide`, `isHideTab` | FE-owner(TBD) |
| D | `/system/webhook` | `/system/webhook` | P1 | `webhook.ts` | `roles: super_admin` | FE-owner(TBD) |
| D | `/system/basic-config` | `/system/basic-config` | P1 | `sys-config.ts` | `roles: super_admin` | FE-owner(TBD) |
| 收尾 | `@views/auth/login/index.vue` | `/auth/login` | P1 | `auth.ts` | 静态路由 | FE-owner(TBD) |
| 收尾 | `@views/auth/callback/index.vue` | `/auth/callback` | P1 | `auth.ts` | 静态路由 | FE-owner(TBD) |
| 收尾 | `@views/auth/recruit/index.vue` | `/r/:code` | P1 | `auth.ts` | 静态路由 | FE-owner(TBD) |
| 收尾 | `@views/outside/Iframe.vue` | `/outside/iframe/:path` | P2 | 无后端依赖（外链） | 静态路由 | FE-owner(TBD) |
| 收尾 | `@views/exception/403/index.vue` | `/403` | P2 | 无 | 静态路由 | FE-owner(TBD) |
| 收尾 | `@views/exception/404/index.vue` | `/:pathMatch(.*)*` | P2 | 无 | 静态路由 | FE-owner(TBD) |
| 收尾 | `@views/exception/500/index.vue` | `/500` | P2 | 无 | 静态路由 | FE-owner(TBD) |

## 当前进展

- 2026-05-01：static-react 已完成批次 A 全部路由注册与占位页接入，且对关键角色门禁完成测试回归。
- 2026-05-01：/dashboard/console 已替换为 React 真实数据页（接入 /api/v1/dashboard，含加载/错误态与测试覆盖）。
- 2026-05-01：/dashboard/characters 已替换为 React 真实数据页（人物资料、直推、绑定/解绑、主人物切换，含测试覆盖）。

- 2026-05-01：`/info/wallet` 已替换为 React 真实数据页（角色切换、流水类型筛选、余额与流水展示，含测试覆盖）。
- 2026-05-01：`/info/skill` 已替换为 React 真实数据页（角色切换、技能筛选、技能队列展示与 ESI 拉取触发，含测试覆盖）。
- 2026-05-01：`/info/ships` 已替换为 React 真实数据页（角色切换、舰船分组筛选、可驾驶状态展示，含测试覆盖）。
- 2026-05-01：`/info/implants` 已替换为 React 真实数据页（角色切换、疲劳状态、活跃植入体与跳克列表展示，含测试覆盖）。
- 2026-05-01：`/info/fittings` 已替换为 React 真实数据页（种族/分组/关键字筛选、分组折叠、装配详情展示）。
- 2026-05-01：`/info/assets` 已替换为 React 真实数据页（位置分组、递归子物品、关键字筛选展示）。

## 说明与已知风险

- 本清单冻结的是“路由页面范围”，不含路由内子组件与纯工具文件（`*.helpers.ts`、`*.test.ts`）。
- `role/*` 页面在 `static/src/views` 存量中不存在且未在 `routeModules` 注册，暂不纳入本次迁移范围。
- 路由文件中出现的中文乱码标题未在本次清单修复；权限标识以 `authMark` 英文字段为准。
- 批次执行前必须补齐 owner，并基于当前后端接口再确认 API 依赖是否存在跨模块调用。




---
status: draft
doc_type: draft
owner: engineering
last_reviewed: 2026-07-22
source_of_truth:
  - static/src
  - static-react/src
---

# 组件替换列表（2026-07-22 审计）

## 布局与导航（P0）

- [x] `ArtSidebarMenu` -> React Sidebar + MenuTree（基线）
- [x] `ArtHeaderBar` -> React HeaderBar（基线）
- [x] `ArtPageContent` -> React PageContent（基线；KeepAlive 替代策略待补）
- [ ] `ArtWorkTab` -> React WorkTab（标签页、固定页、批量关闭，未实现）
- [x] `ArtGlobalComponent` -> React GlobalHost（基线占位）

参考：

- `static/src/views/index/index.vue`
- `static/src/components/core/layouts/*`
- `static-react/src/layout/{app-shell,header-bar,page-content,global-host}.tsx`
- `static-react/src/components/{app-sidebar,ui/sidebar,ui/sidebar-context}.tsx`

## 权限与路由基础设施（P0）

- [ ] `v-auth` 指令 -> `PermissionGate` / `usePermission`（未实现；Vue 侧实际仅 `system/pap-exchange` 模板使用 `v-auth`）
- [ ] `v-roles` 指令 -> `RoleGate` / `useRole`（未实现；Vue 侧当前模板层零引用，属低优先级）
- [x] 未登录守卫 -> React `RequireAuth` + `RouteAccessGate`（替代 Vue `beforeEach`）
- [x] 角色与 `authList` 元数据消费 -> `RouteAccessGate`（替代 Vue 守卫链；新菜单/角色/能力过滤的等价性仍需回归）
- [ ] Vue `RouteRegistry/MenuProcessor/RouteTransformer` 等价能力 -> 当前用扁平 `RouteAccessGate` + `migration-routes.ts` 静态表代替，长链路菜单处理能力需评估是否完整覆盖

参考：

- `static/src/directives/core/auth.ts`
- `static/src/directives/core/roles.ts`
- `static/src/router/guards/*`
- `static/src/router/core/*`
- `static-react/src/auth/{require-auth,route-access-gate,unauthorized-bridge,unauthorized}.tsx`

## 状态管理模块（P0-P1）

Zustand 侧当前仅有 `session` 与 `preference` 两个 store，业务 store 未拆分：

- [~] `userStore` -> `useUserStore`（部分能力由 `useSessionStore` 承载，但搜索历史、锁屏等字段尚未迁移）
- [ ] `menuStore` -> `useMenuStore`（未实现；菜单通过 `layout/menu-config.ts` 在壳层内构造）
- [ ] `worktabStore` -> `useWorktabStore`（未实现；WorkTab 整体未落地）
- [ ] `settingStore` -> `useSettingStore`（未实现；主题与布局偏好部分由 `usePreferenceStore` 承载）
- [ ] `badgeStore` -> Zustand slice（未实现；菜单徽标未接入）
- [ ] `sys-configStore` -> Zustand slice（未实现；运行时站点配置未接入）
- [ ] `tableStore` -> Zustand slice（未实现；表格展示偏好未接入）

参考：

- `static/src/store/modules/*.ts`
- `static-react/src/stores/{session-store,preference-store,index}.ts`

## 基础 UI 替换（P1）

- [~] Element Plus 基础组件（Button/Input/Dialog/Form/Table/Tabs/Dropdown）-> shadcn/ui 等价组件已覆盖各迁移页面常见用法，剩余高耦合控件（日期范围、富文本、图表、拖拽）按需补齐
- [x] 消息与确认框（`ElMessage`/`ElMessageBox`）-> React 反馈层（toast + confirm）已落地
- [~] 表格与分页能力 -> 已在批次 A-D 页面中按需实现；尚未沉淀为统一 `ArtTable` 等价封装
- [ ] 日期、上传、富文本、图表、拖拽等高耦合组件 -> 按页面需要逐项补齐，暂无统一方案

## 商店迁移（C-1）

- [x] `shop/browse` -> React `ShopBrowsePage`（商品卡片、购买弹窗、我的订单）
- [x] `shop/manage` -> React `ShopManagePage`（筛选、商品 CRUD、分页）
- [x] `shop/order-manage` -> React `ShopOrderManagePage`（待发放 / 历史订单、审核弹窗）
- [x] `shop/wallet` -> React `ShopWalletPage`（余额卡片、流水表格、分页）
- [ ] SDE 搜索与商品图片自动填充仍留在后续波次处理

## 技能规划与操作（C-2）

- [x] `skill-planning/completion-check` -> React `SkillPlanCompletionCheckPage`（人物选择、计划选择、完成度检查）
- [x] `skill-planning/skill-plans` -> React `SkillPlansPage` + `SkillPlanManagementPage`（军团技能计划列表、创建、编辑、删除、排序）
- [x] `skill-planning/personal-skill-plans` -> React `PersonalSkillPlansPage` + `SkillPlanManagementPage`（个人技能计划列表、创建、编辑、删除、排序）
- [x] `operation/join` -> React `OperationJoinPage`（邀请入团、角色选择）
- [x] `operation/pap` -> React `OperationPapPage`（个人 PAP、联盟 PAP）

## 系统管理（D-2）

- [x] `system/user` -> React `SystemUserPage`（用户列表、角色管理、ESI 限制开关）
- [x] `system/task-manager` -> React `SystemTaskManagerPage`（任务、ESI 状态、历史）
- [x] `system/wallet` -> React `SystemWalletPage`（钱包列表、流水、日志、分析）
- [x] `system/audit` -> React `SystemAuditPage`（审计日志、筛选、详情、导出占位）
- [x] `system/pap-exchange` -> React `SystemPAPExchangePage`（PAP 兑换配置、费率、FC 工资）
- [x] `system/pap` -> React `SystemPAPPage`（联盟 PAP 抓取、导入、结算）
- [x] `system/auto-role` -> React `SystemAutoRolePage`（ESI 角色映射、头衔映射、同步触发）
- [x] `system/user-center` -> React `SystemUserCenterPage`（本地资料草稿、头像、密码、退出占位）
- [x] `system/webhook` -> React `SystemWebhookPage`（Webhook 配置、测试发送）
- [x] `system/basic-config` -> React `SystemBasicConfigPage`（基础配置、可选军团、SDE 配置）

## Vue 侧范围漂移追赶项（2026-05-01 冻结后新增）

以下页面在 Vue 侧已落地，React 侧尚未实现，按业务模块归类到下一波次：

- [ ] `characters`（顶层公开页，复用 `dashboard/characters` 实现）
- [ ] `dashboard/fuel-officer-structures`（fuel officer 专属建筑视图）
- [ ] `dashboard/galaxy-registry`（星系登记簿）
- [ ] `info/tool-bookmarks`（工具书签管理）
- [ ] `system/qq-governance`（QQ 治理后台）
- [ ] `fuxi-hall/leadership`、`fuxi-hall/contributors`、`fuxi-hall/manage`（取代旧 `hall-of-fame` 模块）

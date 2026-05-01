---
status: draft
doc_type: draft
owner: engineering
last_reviewed: 2026-05-01
source_of_truth:
  - static/src
---

# 组件替换列表（首版）

## 布局与导航（P0）

- [x] `ArtSidebarMenu` -> React Sidebar + MenuTree（基线）
- [x] `ArtHeaderBar` -> React HeaderBar（基线）
- [x] `ArtPageContent` -> React PageContent（基线；KeepAlive 替代策略待补）
- [ ] `ArtWorkTab` -> React WorkTab（标签页、固定页、批量关闭）
- [x] `ArtGlobalComponent` -> React GlobalHost（基线占位）

参考：

- `static/src/views/index/index.vue`
- `static/src/components/core/layouts/*`

## 权限与路由基础设施（P0）

- [ ] `v-auth` 指令 -> `PermissionGate` / `usePermission`
- [ ] `v-roles` 指令 -> `RoleGate` / `useRole`
- [ ] `beforeEach/afterEach` 守卫逻辑 -> React Router 中间层封装
- [ ] `RouteRegistry/MenuProcessor/RouteTransformer` -> React 路由构建器

参考：

- `static/src/directives/core/auth.ts`
- `static/src/directives/core/roles.ts`
- `static/src/router/guards/*`
- `static/src/router/core/*`

## 状态管理模块（P0-P1）

- [ ] `userStore` -> `useUserStore`（Zustand）
- [ ] `menuStore` -> `useMenuStore`
- [ ] `worktabStore` -> `useWorktabStore`
- [ ] `settingStore` -> `useSettingStore`
- [ ] `badgeStore/sys-configStore/tableStore` -> 对应 Zustand slice

参考：

- `static/src/store/modules/*.ts`

## 基础 UI 替换（P1）

- [ ] Element Plus 基础组件（Button/Input/Dialog/Form/Table/Tabs/Dropdown）
- [ ] 消息与确认框（`ElMessage`/`ElMessageBox`）
- [ ] 表格与分页能力（对齐现有 `ArtTable` 使用规范）
- [ ] 日期、上传、富文本、图表、拖拽等高耦合组件

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

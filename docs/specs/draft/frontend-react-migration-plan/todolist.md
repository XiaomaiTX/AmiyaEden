---
status: draft
doc_type: draft
owner: engineering
last_reviewed: 2026-07-30
source_of_truth:
  - static/src
  - static-react/src
  - docs/ai/repo-rules.md
---

# 迁移 TodoList

## P0 基座与规范

- [x] 创建 React 子应用目录与入口（与现有 Vue 应用并行，路径：`static-react/`）
- [x] 建立 TS、ESLint、Prettier、测试脚本基线（含 `lint` / `test` / `build` 可执行）
- [x] 配置 React Router Hash 模式并对齐基础 404/500 页面（`/`、`/500`、`*`）
- [x] 接入 Tailwind + shadcn/ui 并建立基础主题变量（已执行 `shadcn init -t vite`）
- [x] 接入 Zustand + persist 中间件，定义首批全局 store 边界（session/auth 快照 + preference）
- [x] 定义 API 类型契约迁移方案（React 使用本地模块化类型，详见 `./api-contract-migration-plan.md`）

## P0 壳层能力迁移

- [x] 迁移应用壳层（侧边菜单、头部、内容容器、全局层，已完成 React 基线壳层）
- [x] 迁移登录态守卫与未授权处理链路（RequireAuth + 401 统一回跳 `/login?redirect=`）
- [x] 迁移路由权限元数据消费逻辑（`login/roles/authList`，含 403 分支与 authList 注入）
- [x] Stage 0A：当前已迁移 React 页面补齐 capability/menu/button parity；使用 `corpCapabilitiesAll` / `corpCapabilitiesAny`，并放行 `super_admin` 的 Captain/Mentor 页面
- [x] 迁移 i18n 基础能力（`zh/en` 双语与切换，含 `I18nProvider + useI18n`）
- [x] 迁移全局消息与错误提示能力（替代 `ElMessage/ElMessageBox`，含 toast + confirm）
- [x] React 前端接入深色模式主题切换（`ThemeProvider + ModeToggle`，遵循 `dark` 类驱动）
- [x] React 壳层 `SidebarContext` 独立文件化（侧边栏状态、移动端展开状态、cookie 记忆、快捷键切换已从 `sidebar.tsx` 拆分完成）

- [x] React 真实 SSO 登录闭环落地（`/auth/login` + `/auth/callback` + 401 回登 + redirect 回跳）

## P1 业务模块迁移

### 模块分批顺序（详细）

- [x] 产出“迁移范围基线清单”：冻结 `static/src/views` 页面清单，标注 owner、优先级、依赖 API、权限码（见 `./migration-scope-baseline.md`）
- [x] `C-1` 商店首波完成：`shop/browse`、`shop/manage`、`shop/order-manage`、`shop/wallet`
- [x] `C-2` 技能规划与操作首波完成：`skill-planning/completion-check`、`skill-planning/skill-plans`、`skill-planning/personal-skill-plans`、`operation/join`、`operation/pap`
- [x] 批次 A 路由骨架已在 React 注册（页面全部已替换为真实业务页）
- [x] 批次 A 页面：`dashboard/console`、`info/wallet`、`info/skill`、`info/ships`、`info/implants`、`info/fittings`、`info/assets`、`info/contracts`、`info/esi-check`、`dashboard/npc-kills`、`dashboard/corporation-structures`、`info/npc-kills` 已全部完成 React 真实页迁移
- [x] 批次 B 页面：`ticket/my-tickets`、`ticket/create`、`ticket/detail`、`ticket/management`、`ticket/categories`、`ticket/statistics`、`ticket/admin-detail`、`welfare/my`、`welfare/approval`、`welfare/settings`、`newbro/select-captain`、`newbro/select-mentor`、`newbro/captain`、`newbro/mentor`、`newbro/manage`、`newbro/mentor-manage`、`newbro/recruit-link`、`srp/srp-apply`、`srp/srp-manage`、`srp/srp-prices` 已全部完成 React 真实页迁移
- [x] 批次 C 页面：`shop/*`、`skill-planning/completion-check`、`skill-planning/skill-plans`、`skill-planning/personal-skill-plans`、`operation/join`、`operation/pap` 已全部完成 React 真实页迁移
- [x] 批次 D 页面：`operation/fleets`、`operation/fleet-detail`、`operation/fleet-configs`、`operation/corporation-pap`、`system/user`、`system/task-manager`、`system/wallet`、`system/audit`、`system/pap-exchange`、`system/pap`、`system/auto-role`、`system/user-center`、`system/webhook`、`system/basic-config` 已全部完成 React 真实页迁移
- [x] 收尾批次壳层与公共页：`auth/login`、`auth/callback`、`/r/:code`、`outside/iframe`、`403`、`404`、`500` 已全部完成 React 真实页迁移

### Vue 侧范围漂移追赶项（2026-05-01 冻结后新增）

Vue 侧在 2026-05-01 冻结后陆续新增以下路由，React 侧尚未对齐：

- [x] `/characters` 顶层人物管理路由（资料/ESI 锁定唯一落点）
- [x] `/dashboard/fuel-officer-structures` React 落地（`super_admin/fuel_officer`，含分页、加载/空/错误态）
- [~] `/dashboard/galaxy-registry` React 基础页、路由、API 与类型已落地；预计结束时间修改、管理员星系配置、人工校验、分析与浏览器超时通知仍待完整同构
- [x] `/info/tool-bookmarks` React 落地（2026-05-13 Vue 落地；普通用户读取启用项，管理员可维护全部书签）
- [~] `/system/qq-governance` React 基础页、路由、API 与类型已落地；规则 CRUD、成员/判断记录、军团搜索和完整限流观测仍待完整同构
- [x] `/fuxi-hall/leadership`、`/fuxi-hall/contributors`、`/fuxi-hall/manage` React 落地（取代旧 `hall-of-fame` 模块，2026-05-12 Vue 落地）
- [x] 移除 React 侧历史遗留的 `hall-of-fame/{temple, manage, current-manage}` 三条 stub（Vue 已删除）

### 批次执行与验收节奏

- [ ] 每批次开始前完成“冻结范围 + 风险评审 + 回归清单确认”
- [ ] 每批次结束时输出“已迁移路由清单 + 未决问题清单 + 下一批依赖项”
- [ ] 每批次至少完成一次跨角色回归（普通成员/管理员/受限角色）
- [ ] 每批次完成后更新 `component-replacement-list.md` 与本清单状态

### 回归验收（未关闭）

- [ ] 批次 A 回归通过：路由可达、查询参数一致、表格筛选与分页行为一致
- [ ] 批次 B 回归通过：创建/编辑/审批链路、状态流转、按钮权限一致
- [ ] 批次 C 回归通过：复杂筛选、批量操作、导入导出、弹窗编辑一致
- [ ] 批次 D 回归通过：多角色权限矩阵、长链路事务、跨页面状态一致
- [ ] 收尾批次回归通过：历史入口兼容策略与替换后可访问性验证

## 基础设施补齐（未关闭）

壳层与权限基座已落地，但以下 Vue 既有能力在 React 侧尚未实现，替换发布前必须补齐：

- [x] WorkTab 多标签页（固定、批量关闭、按人物隔离、完整 URL 恢复；React 采用 URL/store 恢复，不保留隐藏挂载的页面树）
- [x] `PermissionGate` / `usePermission`（对应 Vue `v-auth`；权限由当前叶子路由 Context 提供，不写入 session）
- [x] `RoleGate` / `useRole`（对应 Vue `v-roles`）
- [~] Zustand 业务 store：已完成 `session`、`preference`、`worktab`、`badge`；`user/menu/setting/table/sys-config` 仅在出现真实跨页状态需求时继续拆分
- [x] 共享 `DataTable` 基座（TanStack Table；统一加载/错误/空态、服务端分页、排序与选择接口）
- [x] React 路由与菜单基础设施：使用扁平 `RouteAccessGate`、React Router 和菜单构建器，不引入或复刻另一端的处理链

## 下一步代码执行顺序（2026-07-30）

1. 完成 Galaxy Registry 同构：先补齐类型与 API wrapper，再拆分状态/队长/管理员三个 Tab，最后迁移超时通知 helper；覆盖 `user/captain/admin/super_admin` 四类角色回归。
2. 完成 QQ Governance 同构：先补齐 policy/member/review/corporation/rate-limit 契约，再实现规则编辑和运行监控；所有写操作继续只做前端 UX gate，后端 `super_admin + system.manage` 保持最终边界。
3. 将新增复杂表格优先迁入共享 `DataTable`，存量页面只在业务变更触及时渐进迁移，避免无关大面积重写。
4. 完成替换发布门槛：生产构建体积基线、关键路由冒烟、四角色矩阵、React/Vue 同契约检查与回切演练。

### 路由与菜单并行契约（2026-07-30）

- [x] React 路由与菜单保持独立实现：以 `migration-routes.ts`、React Router、Zustand 和 `buildShellMenuGroups` 为唯一 React 运行时来源。
- [x] Vue 与 React 不建立代码联系：禁止共享路由包、manifest、运行时、AST 读取或源码比较；Vue 的既有行为仅通过本 draft 的并行契约进入迁移计划。
- [x] 工单分类读路径为 `ticket.manage + ticket.admin.read`，创建、编辑、删除在页面内额外使用 `ticket.admin.manage` gate；自动角色页仅保留 `super_admin` 角色约束。

## 页面迁移完成定义（详细 DoD）

- [ ] DoD-01 页面路由已在 React 注册，路径、参数、404 行为与 Vue 侧一致
- [ ] DoD-02 页面所需菜单元数据、`authList`、角色约束已接入并生效
- [ ] DoD-03 页面涉及 API 全部使用 React 模块化共享类型，禁止新增同义重复 interface 或回退引用 Vue `Api.*`
- [ ] DoD-04 页面请求成功态、空态、加载态、错误态完整可见，401/403/500 处理一致
- [ ] DoD-05 页面表单校验规则、默认值、提交前后行为与 Vue 侧一致
- [ ] DoD-06 页面表格能力一致：筛选、排序、分页、列显隐、批量操作、导出
- [ ] DoD-07 页面按钮权限与操作权限一致，前端仅做 UX 控制，服务端鉴权不回退
- [ ] DoD-08 页面 i18n 完整：`zh/en` 文案齐全，不引入硬编码文案
- [ ] DoD-09 页面样式与交互完成基线对齐：信息架构一致，关键操作路径无断点
- [ ] DoD-10 页面埋点/日志/错误提示策略（若该页已有）在 React 侧等价落地
- [ ] DoD-11 页面最小回归通过：`lint`、类型检查、对应测试、手工冒烟清单
- [ ] DoD-12 页面迁移记录已回填：迁移人、完成日期、风险点、回滚要点

## 暂缓项

- [x] `hall-of-fame/*` 模块在 Vue 侧已于 2026-05-12 整体删除并被 `fuxi-hall` 取代；React 侧遗留三条 stub 已在 Fuxi Hall 迁移时移除
- [x] Vue 侧 `router/modules/role.ts` 未被 `modules/index.ts` 引用，对应的 `views/role/*` 不存在；React 侧不纳入迁移范围
- [x] Vue 侧 `views/auth/register/index.vue` 无对应路由，属于模板遗留，React 侧不纳入迁移范围

## P1 替换发布与回切

- [ ] 制定替换发布策略（按环境批次执行，不做长期 Vue/React 共存）
- [ ] 建立回切开关与回切演练流程
- [ ] 明确替换门槛（错误率、关键路径成功率、性能指标）
- [ ] 完成 Vue 下线前最终全量回归

## 文档适配与切换阻断

- [ ] 完成 active 架构、API、标准和指南的 Vue/React 双端映射
- [ ] 完成 current feature docs 的实现映射，未迁移功能明确标记为 React 缺口
- [ ] 完成 `migration-scope-baseline.md` 文档适配矩阵，并将其作为唯一迁移状态来源
- [x] React capability/menu/button permission parity 已通过回归；active 文档中的 capability 规则已按 0A 实现更新。剩余 Vue-only 限制仅对应未迁移范围或其他基础设施。

## 验收基线

- [ ] P0 项有 owner 与目标时间
- [ ] 路由、权限、登录态、API 契约回归通过
- [ ] 替换发布流程和回切流程可演练

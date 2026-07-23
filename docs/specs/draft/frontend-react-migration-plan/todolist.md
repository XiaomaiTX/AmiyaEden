---
status: draft
doc_type: draft
owner: engineering
last_reviewed: 2026-07-23
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
- [x] 批次 A 页面：`dashboard/console`、`dashboard/characters`、`info/wallet`、`info/skill`、`info/ships`、`info/implants`、`info/fittings`、`info/assets`、`info/contracts`、`info/esi-check`、`dashboard/npc-kills`、`dashboard/corporation-structures`、`info/npc-kills` 已全部完成 React 真实页迁移
- [x] 批次 B 页面：`ticket/my-tickets`、`ticket/create`、`ticket/detail`、`ticket/management`、`ticket/categories`、`ticket/statistics`、`ticket/admin-detail`、`welfare/my`、`welfare/approval`、`welfare/settings`、`newbro/select-captain`、`newbro/select-mentor`、`newbro/captain`、`newbro/mentor`、`newbro/manage`、`newbro/mentor-manage`、`newbro/recruit-link`、`srp/srp-apply`、`srp/srp-manage`、`srp/srp-prices` 已全部完成 React 真实页迁移
- [x] 批次 C 页面：`shop/*`、`skill-planning/completion-check`、`skill-planning/skill-plans`、`skill-planning/personal-skill-plans`、`operation/join`、`operation/pap` 已全部完成 React 真实页迁移
- [x] 批次 D 页面：`operation/fleets`、`operation/fleet-detail`、`operation/fleet-configs`、`operation/corporation-pap`、`system/user`、`system/task-manager`、`system/wallet`、`system/audit`、`system/pap-exchange`、`system/pap`、`system/auto-role`、`system/user-center`、`system/webhook`、`system/basic-config` 已全部完成 React 真实页迁移
- [x] 收尾批次壳层与公共页：`auth/login`、`auth/callback`、`/r/:code`、`outside/iframe`、`403`、`404`、`500` 已全部完成 React 真实页迁移

### Vue 侧范围漂移追赶项（2026-05-01 冻结后新增）

Vue 侧在 2026-05-01 冻结后陆续新增以下路由，React 侧尚未对齐：

- [ ] `/characters` 顶层路由对齐（与 `dashboard/characters` 共享页面，但路由路径独立，2026-05-22 Vue 落地）
- [ ] `/dashboard/fuel-officer-structures` React 落地（`super_admin/fuel_officer`，2026-05-11 Vue 落地）
- [ ] `/dashboard/galaxy-registry` React 落地（`super_admin/admin/captain/user`，2026-06-04 Vue 落地）
- [ ] `/info/tool-bookmarks` React 落地（2026-05-13 Vue 落地）
- [ ] `/system/qq-governance` React 落地（`super_admin`，2026-07-12 Vue 落地）
- [ ] `/fuxi-hall/leadership`、`/fuxi-hall/contributors`、`/fuxi-hall/manage` React 落地（取代旧 `hall-of-fame` 模块，2026-05-12 Vue 落地）
- [ ] 移除 React 侧历史遗留的 `hall-of-fame/{temple, manage, current-manage}` 三条 stub（Vue 已删除）

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

- [ ] WorkTab 多标签页（Vue `worktabStore` + `ArtWorkTab` 对应能力，含固定、批量关闭、KeepAlive 缓存）
- [ ] `PermissionGate` / `usePermission`（对应 Vue `v-auth`，当前 Vue 侧仅 `system/pap-exchange` 模板使用 `v-auth`）
- [ ] `RoleGate` / `useRole`（对应 Vue `v-roles`，Vue 侧当前实际无模板引用，可降优先级）
- [ ] Zustand 业务 store 补齐：`user`、`menu`、`worktab`、`setting`、`table`、`badge`、`sys-config`（当前 React 仅有 `session` 与 `preference` 两个 store）
- [ ] React 路由守卫中间层与 `RouteRegistry/MenuProcessor` 等价能力（当前用扁平的 `RouteAccessGate` 代替，需评估是否覆盖菜单折叠/能力过滤等既有行为）

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

- [x] `hall-of-fame/*` 模块在 Vue 侧已于 2026-05-12 整体删除并被 `fuxi-hall` 取代；React 侧目前仍保留三条 stub 占位，属于遗留 stub，需在 `fuxi-hall/*` 迁移落地时一并移除
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


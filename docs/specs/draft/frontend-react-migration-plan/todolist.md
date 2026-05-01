---
status: draft
doc_type: draft
owner: engineering
last_reviewed: 2026-05-01
source_of_truth:
  - static/src
  - docs/ai/repo-rules.md
---

# 迁移 TodoList

## P0 基座与规范

- [x] 创建 React 子应用目录与入口（与现有 Vue 应用并行，路径：`static-react/`）
- [x] 建立 TS、ESLint、Prettier、测试脚本基线（含 `lint` / `test` / `build` 可执行）
- [x] 配置 React Router Hash 模式并对齐基础 404/500 页面（`/`、`/500`、`*`）
- [x] 接入 Tailwind + shadcn/ui 并建立基础主题变量（已执行 `shadcn init -t vite`）
- [x] 接入 Zustand + persist 中间件，定义首批全局 store 边界（session/auth 快照 + preference）
- [x] 定义 API 类型契约迁移方案（沿用 `static/src/types/api/api.d.ts`，详见 `./api-contract-migration-plan.md`）

## P0 壳层能力迁移

- [x] 迁移应用壳层（侧边菜单、头部、内容容器、全局层，已完成 React 基线壳层）
- [x] 迁移登录态守卫与未授权处理链路（RequireAuth + 401 统一回跳 `/login?redirect=`）
- [x] 迁移路由权限元数据消费逻辑（`login/roles/authList`，含 403 分支与 authList 注入）
- [x] 迁移 i18n 基础能力（`zh/en` 双语与切换，含 `I18nProvider + useI18n`）
- [x] 迁移全局消息与错误提示能力（替代 `ElMessage/ElMessageBox`，含 toast + confirm）
- [x] React 前端接入深色模式主题切换（`ThemeProvider + ModeToggle`，遵循 `dark` 类驱动）

- [x] React 真实 SSO 登录闭环落地（`/auth/login` + `/auth/callback` + 401 回登 + redirect 回跳）

## P1 业务模块迁移

### 模块分批顺序（详细）

- [x] 产出“迁移范围基线清单”：冻结 `static/src/views` 页面清单，标注 owner、优先级、依赖 API、权限码（见 `./migration-scope-baseline.md`）
- [x] 批次 A 路由骨架已在 React 注册（页面为迁移占位实现，待逐页补业务逻辑）
- [x] 批次 A 页面进度：`dashboard/console` 已完成 React 真实页迁移（其余页面待迁移）
- [x] 批次 A 页面进度：`dashboard/characters` 已完成 React 真实页迁移（人物资料、直推、绑定/解绑、主人物切换）
- [x] 批次 A 页面进度：`info/wallet` 已完成 React 真实页迁移（其余页面待迁移）
- [x] 批次 A 页面进度：`info/skill` 已完成 React 真实页迁移（其余页面待迁移）
- [x] 批次 A 页面进度：`info/ships` 已完成 React 真实页迁移（其余页面待迁移）
- [x] 批次 A 页面进度：`info/implants` 已完成 React 真实页迁移（其余页面待迁移）
- [x] 批次 A 页面进度：`info/fittings` 已完成 React 真实页迁移（其余页面待迁移）
- [x] 批次 A 页面进度：`info/assets` 已完成 React 真实页迁移（其余页面待迁移）
- [x] 批次 A 页面进度：`info/contracts` 已完成 React 真实页迁移（合同列表与详情侧栏，含测试覆盖）
- [x] 批次 A 页面进度：`info/esi-check` 已完成 React 真实页迁移（授权总览与人物详情，含测试覆盖）
- [x] 批次 A 页面进度：`dashboard/npc-kills` 已完成 React 真实页迁移（军团刷怪报表）
- [x] 批次 A 页面进度：`dashboard/corporation-structures` 已完成 React 真实页迁移（军团建筑）
- [x] 批次 A 页面进度：`info/npc-kills` 已完成 React 真实页迁移（刷怪报表、流水明细）
- [x] 批次 B 页面进度：`ticket/my-tickets` 已完成 React 真实页迁移（我的工单列表）
- [x] 批次 B 页面进度：`ticket/create` 已完成 React 真实页迁移（提交工单）
- [x] 批次 B 页面进度：`ticket/detail` 已完成 React 真实页迁移（工单详情与回复）
- [x] 批次 B 页面进度：`ticket/management`、`ticket/categories`、`ticket/statistics`、`ticket/admin-detail` 已完成 React 真实页迁移
- [x] 批次 B 页面进度：`welfare/my`、`welfare/approval`、`welfare/settings` 已完成 React 真实页迁移
- [x] 批次 B 页面进度：`newbro/select-captain`、`newbro/select-mentor`、`newbro/captain`、`newbro/mentor`、`newbro/manage`、`newbro/mentor-manage`、`newbro/recruit-link` 已完成 React 真实页迁移
- [x] 批次 B 页面进度：`srp/srp-apply`、`srp/srp-manage`、`srp/srp-prices` 已完成 React 真实页迁移
- [x] 批次 A（低耦合只读页）路由与页面迁移完成
- [ ] 批次 A 包含模块：`dashboard/*`、`info/*`
- [ ] 批次 A 回归通过：路由可达、查询参数一致、表格筛选与分页行为一致
- [x] 批次 B（中耦合流程页）路由与页面迁移完成
- [x] 批次 B 包含模块：`ticket/*`、`welfare/*`、`newbro/*`、`srp/*`
- [ ] 批次 B 回归通过：创建/编辑/审批链路、状态流转、按钮权限一致
- [ ] 批次 C（中高耦合业务页）路由与页面迁移完成
- [ ] 批次 C 包含模块：`shop/*`、`skill-planning/*`、`operation/join`、`operation/pap`
- [ ] 批次 C 回归通过：复杂筛选、批量操作、导入导出、弹窗编辑一致
- [ ] 批次 D（高耦合核心页）路由与页面迁移完成
- [ ] 批次 D 包含模块：`operation/fleets`、`operation/fleet-detail`、`operation/fleet-configs`、`operation/corporation-pap`、`system/*`
- [ ] 批次 D 回归通过：多角色权限矩阵、长链路事务、跨页面状态一致
- [ ] 收尾批次（边缘与遗留）完成
- [ ] 收尾批次包含模块：`auth/*`（仅保留 EVE SSO 所需页面）、`outside/*`、`hall-of-fame/manage`、`hall-of-fame/current-manage`
- [ ] 收尾批次回归通过：历史入口兼容策略与替换后可访问性验证

### 页面迁移完成定义（详细 DoD）

- [ ] DoD-01 页面路由已在 React 注册，路径、参数、404 行为与 Vue 侧一致
- [ ] DoD-02 页面所需菜单元数据、`authList`、角色约束已接入并生效
- [ ] DoD-03 页面涉及 API 全部使用共享 `Api.*` 类型，禁止新增同义重复 interface
- [ ] DoD-04 页面请求成功态、空态、加载态、错误态完整可见，401/403/500 处理一致
- [ ] DoD-05 页面表单校验规则、默认值、提交前后行为与 Vue 侧一致
- [ ] DoD-06 页面表格能力一致：筛选、排序、分页、列显隐、批量操作、导出
- [ ] DoD-07 页面按钮权限与操作权限一致，前端仅做 UX 控制，服务端鉴权不回退
- [ ] DoD-08 页面 i18n 完整：`zh/en` 文案齐全，不引入硬编码文案
- [ ] DoD-09 页面样式与交互完成基线对齐：信息架构一致，关键操作路径无断点
- [ ] DoD-10 页面埋点/日志/错误提示策略（若该页已有）在 React 侧等价落地
- [ ] DoD-11 页面最小回归通过：`lint`、类型检查、对应测试、手工冒烟清单
- [ ] DoD-12 页面迁移记录已回填：迁移人、完成日期、风险点、回滚要点

### 批次执行与验收节奏

- [ ] 每批次开始前完成“冻结范围 + 风险评审 + 回归清单确认”
- [ ] 每批次结束时输出“已迁移路由清单 + 未决问题清单 + 下一批依赖项”
- [ ] 每批次至少完成一次跨角色回归（普通成员/管理员/受限角色）
- [ ] 每批次完成后更新 `component-replacement-list.md` 与本清单状态

## 暂缓项

- [ ] `hall-of-fame/temple` 本轮取消实现，React 侧仅保留 stub 占位，等待后续重构单独立项

## P1 替换发布与回切
 
- [ ] 制定替换发布策略（按环境批次执行，不做长期 Vue/React 共存）
- [ ] 建立回切开关与回切演练流程
- [ ] 明确替换门槛（错误率、关键路径成功率、性能指标）
- [ ] 完成 Vue 下线前最终全量回归

## 验收基线

- [ ] P0 项有 owner 与目标时间
- [ ] 路由、权限、登录态、API 契约回归通过
- [ ] 替换发布流程和回切流程可演练






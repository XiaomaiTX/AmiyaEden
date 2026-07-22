---
status: draft
doc_type: draft
owner: engineering
last_reviewed: 2026-07-22
source_of_truth:
  - static/src
  - static-react/src
  - docs/architecture/routing-and-menus.md
  - docs/architecture/auth-and-permissions.md
---

# 独立 React 替换迁移 Todo

## 路由与入口策略

- [x] 统一 React 路由与页面入口（不再新增 Vue/React 双入口分流策略）
- [x] 保持 Hash 模式一致，避免服务端 rewrite 变更
- [x] 对齐 404/500 未命中行为（含 `/403` 静态页）
- [ ] 明确最终替换时的入口切换步骤（构建产物、部署路径、回切指令）

## 鉴权与会话对齐

- [x] 对齐 Token 读取与过期处理策略（React `RouteAccessGate` + `unauthorized-bridge` 处理 401）
- [x] 对齐未授权退出行为（401 处理语义一致）
- [x] 对齐登录后回跳参数语义（`redirect`）
- [x] 保证用户信息与权限拉取链路在 React 侧闭环（SSO 登录 + 回调 + 角色注入）

## 状态与核心能力迁移

- [~] 定义全局状态最小集合（用户、权限、语言、主题）—— 当前由 `useSessionStore` 与 `usePreferenceStore` 承载；菜单、工作台、徽标等业务 store 未拆分
- [ ] 完成 Pinia -> Zustand 的状态迁移映射（`user/menu/worktab/setting/table/badge/sys-config` 7 个 Vue store 中尚有 5 个未迁移）
- [x] 明确 React 侧单一权威源，禁止新增 Vue 侧状态耦合
- [ ] 对齐工作台标签页等核心能力的最终行为（WorkTab 未落地，KeepAlive 等价策略未定）

## UI 与样式迁移

- [x] 明确 Tailwind 与现有 SCSS/Element 样式隔离策略（React 子应用独立 `static-react/`，Tailwind + shadcn/ui 自洽）
- [x] 避免全局样式污染（命名空间/容器范围）
- [x] 首轮以信息架构和交互一致为目标，不做大规模视觉差异
- [x] 统一暗色模式与主题变量来源（`ThemeProvider` + `usePreferenceStore.theme`）

## 交付与替换发布

- [ ] 记录已迁移路由与未迁移路由清单（以 `migration-scope-baseline.md` 为准，范围漂移追赶项尚未对齐）
- [ ] 建立“模块回归 -> 全量回归 -> Vue 下线”固定流程
- [ ] 定义替换门槛（页面覆盖率、关键链路成功率、错误率、性能指标）
- [ ] 完成 Vue 前端下线检查项与演练记录

## 待确认项

- 替换窗口：是否在关键业务周期前设置迁移冻结时间段
- 模块优先级：首批迁移模块的业务优先级与资源顺序（当前漂移追赶项已隐含优先级：`characters`、`fuxi-hall/*`、`system/qq-governance` 等）
- 回切策略：替换后若出现重大问题的回切条件与执行时限


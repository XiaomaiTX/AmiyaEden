---
status: draft
doc_type: draft
owner: engineering
last_reviewed: 2026-07-30
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

- [x] 定义全局状态最小集合：`session`、`preference`、`worktab`、`badge`；菜单保持由路由声明派生
- [~] 完成 Pinia -> Zustand 的行为映射；不按 Pinia 文件数量机械复制 store，仅为真实跨页持久状态建 Zustand store
- [x] 明确 React 侧单一权威源，禁止新增 Vue 侧状态耦合
- [x] 对齐工作台核心行为：固定、批量关闭、按人物隔离和完整 URL 恢复；React 不保留隐藏挂载页面树

## UI 与样式迁移

- [x] 明确 Tailwind 与现有 SCSS/Element 样式隔离策略（React 子应用独立 `static-react/`，Tailwind + shadcn/ui 自洽）
- [x] 避免全局样式污染（命名空间/容器范围）
- [x] 首轮以信息架构和交互一致为目标，不做大规模视觉差异
- [x] 统一暗色模式与主题变量来源（`ThemeProvider` + `usePreferenceStore.theme`）

## 交付与替换发布

- [x] 建立迁移期独立 React 镜像与 Compose `frontend-react` 服务，保持 Vue `frontend` 不变并行运行
- [x] 将 React lint、类型检查、测试、契约检查和构建接入现有 verify/main/preview 工作流
- [x] 记录已迁移路由与未迁移路由清单（以 `migration-scope-baseline.md` 为准）
- [ ] 建立“模块回归 -> 全量回归 -> Vue 下线”固定流程
- [ ] 定义替换门槛（页面覆盖率、关键链路成功率、错误率、性能指标）
- [ ] 完成 Vue 前端下线检查项与演练记录

## 文档同步门槛

- [ ] 每个已迁移模块的 current feature doc 已包含 Vue/React 实现映射
- [ ] active 规范不再把 Vue 路径、Pinia、`v-auth` 或 `vue-tsc` 作为唯一实现
- [ ] 未迁移路由和 React 基础能力缺口均登记在迁移基线，不在 feature 文档中重复维护
- [ ] Vue 下线前完成 active 文档的 Vue-only 约束扫描与失效链接扫描

## 待确认项

- 替换窗口：是否在关键业务周期前设置迁移冻结时间段
- 模块优先级：先完成 Galaxy Registry 管理/分析/通知同构，再完成 QQ Governance 规则/观测同构
- 回切策略：替换后若出现重大问题的回切条件与执行时限


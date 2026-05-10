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

## P1 业务模块迁移

- [ ] 梳理模块分批顺序（先低耦合、后高耦合）
- [ ] 建立页面迁移完成定义（功能、权限、接口、回归）
- [ ] 完成首批低耦合页面迁移并灰度验证
- [ ] 完成高耦合页面迁移（复杂表单、拖拽、编辑器、工作台）

## P1 切流与回滚
 
- [ ] 制定灰度策略（按路由、用户组、环境开关）
- [ ] 建立回滚开关与回滚演练流程
- [ ] 明确切流门槛（错误率、关键路径成功率、性能指标）
- [ ] 完成双栈窗口关闭前最终回归

## 验收基线

- [ ] P0 项有 owner 与目标时间
- [ ] 路由、权限、登录态、API 契约回归通过
- [ ] 灰度流程和回滚流程可演练








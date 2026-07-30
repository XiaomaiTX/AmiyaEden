---
status: draft
doc_type: draft
owner: engineering
last_reviewed: 2026-07-22
source_of_truth:
  - static-react/src
  - docs/ai/repo-rules.md
---

# API 类型契约迁移方案（React 独立）

## 目标

- React 侧保持本地 API 类型单一事实源，避免与旧 Vue 前端共享契约文件。
- 直接采用 React 友好的模块化组织形式，不再依赖 `static/src/types/api/api.d.ts` 或全局 `Api.*` 命名空间。

## 当前基线（2026-07-22 审计）

- React 权威类型定义：`static-react/src/types/api/*`（25 个业务域类型文件）。
- React API 封装调用：`static-react/src/api/*.ts`（27 个模块，含 `http-client.ts` 与 `response.ts`）。
- React 子应用当前状态：已切换为本地契约文件，不再读取 Vue 类型文件；`static-react/src/types/api-contract.d.ts` 已移除 Vue 共享 import。
- 业务域覆盖：auth、dashboard、eve-info、npc-kill、corporation-structures、fleet、fleet-config、alliance-pap、pap-exchange、skill-plan、shop、srp、ticket、welfare、newbro、mentor、system-manage、sys-wallet、sys-config、task-manager、audit、webhook、upload、tool-bookmark。
- 漂移追赶：`qq-governance`、`galaxy-registry` 与 `fuel-officer-structures` 均已建立 React 本地类型和 API wrapper；Galaxy Registry 与 QQ Governance 的完整页面同构已关闭，后续字段变更按本方案同步维护。

## 迁移原则

- 单一来源：迁移阶段只维护 `static-react/src/types/api/*`，禁止回退引用 Vue 类型文件。
- React 优先组织：React 侧按业务模块导出类型与 API，不在页面层直接依赖全局命名空间。
- 先类型后封装：先完成模块化类型出口，再分批迁移 API wrapper 与页面调用。
- 小步切换：每迁一个业务模块，就在同一提交内对齐 `后端契约 -> 类型 -> wrapper -> 页面`。
- 禁止兼容分叉：除明确灰度需求外，不新增并行旧新字段别名。

## 执行步骤

### Step 1：类型来源接入（已完成）

- `static-react/src/types/api-contract.d.ts` 已移除 Vue 共享 import。
- React 侧类型来源统一来自本地契约文件，不再新增共享桥接。

### Step 2：模块化类型出口（已完成）

- `static-react/src/types/api/` 已按业务域建立 25 个类型文件。
- 页面与组件只依赖模块导出，不再直接耦合 `Api.*`。
- 后续新增业务域（如 `qq-governance`、`galaxy-registry`）必须沿用该结构新增类型文件。

### Step 3：封装与页面迁移（持续）

- `static-react/src/api` 已建立与 Vue 的 `request` 行为语义一致的请求封装：401 处理、错误提示、重试策略对齐。
- 27 个 API wrapper 已覆盖当前已迁移的全部业务页面。
- 每个已迁模块已关闭在 React 侧对全局命名空间的直接引用。
- 漂移追赶项落地时，必须同步补齐对应 API wrapper 与类型文件，不得复用 Vue 类型。

## DoD（完成定义）

- [x] React 侧新增页面不再定义重复接口类型。
- [x] React 侧页面与组件不再直接引用全局 `Api.*` 命名空间，只通过模块化类型出口使用契约。
- [~] 每个已迁模块均满足：与后端响应字段一致；Wrapper 与页面调用通过 TS 类型检查（持续维护，漂移项落地时一并校验）。
- [x] 任一接口字段变更时，仅在 React 本地契约与调用链内完成对齐。

## 风险与防护

- 风险：本地契约与后端响应字段可能发生漂移。
- 防护：每个业务域的类型出口必须与对应 API wrapper 和页面测试同步更新。
- 风险：迁移过渡期可能出现页面直接引用 `Api.*` 与模块导出并存。
- 防护：静态检查禁止 `static/src` 引用和 `Api.` 全局命名空间残留。
- 风险：Vue 侧持续新增业务域（如 2026-05 后的 `qq-governance`）造成 React 类型出口滞后。
- 防护：将“漂移追赶项”纳入 `migration-scope-baseline.md`，每发现一个新业务域立即补建 React 类型出口与 wrapper，不留空白引用。

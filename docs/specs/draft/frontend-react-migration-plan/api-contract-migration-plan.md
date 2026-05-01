---
status: draft
doc_type: draft
owner: engineering
last_reviewed: 2026-05-01
source_of_truth:
  - static-react/src
  - docs/ai/repo-rules.md
---

# API 类型契约迁移方案（React 独立）

## 目标

- React 侧保持本地 API 类型单一事实源，避免与旧 Vue 前端共享契约文件。
- 直接采用 React 友好的模块化组织形式，不再依赖 `static/src/types/api/api.d.ts` 或全局 `Api.*` 命名空间。

## 当前基线

- React 权威类型定义：`static-react/src/types/api/*`。
- React API 封装调用：`static-react/src/api/*.ts`。
- React 子应用当前状态：已切换为本地契约文件，不再读取 Vue 类型文件。

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

### Step 2：模块化类型出口（进行中）

- 在 `static-react/src/types/api` 建立按业务域组织的类型出口。
- 页面与组件只依赖模块导出，不直接耦合 `Api.*`。

### Step 3：封装与页面迁移（进行中持续）

- 在 `static-react/src/api` 建立新的请求封装（与 Vue 的 `request` 行为语义保持一致：401、错误提示、重试策略）。
- 首批迁移 `auth`、`badge` 等低耦合 API wrapper，并切换对应页面到模块化类型与 API。
- 以业务域为单位推进，完成一个域就关闭该域在 React 侧对全局命名空间的直接引用。

## DoD（完成定义）

- React 侧新增页面不再定义重复接口类型。
- React 侧页面与组件不再直接引用全局 `Api.*` 命名空间，只通过模块化类型出口使用契约。
- 每个已迁模块均满足：
  - 与后端响应字段一致；
  - Wrapper 与页面调用通过 TS 类型检查。
- 任一接口字段变更时，仅在 React 本地契约与调用链内完成对齐。

## 风险与防护

- 风险：本地契约与后端响应字段可能发生漂移。
- 防护：每个业务域的类型出口必须与对应 API wrapper 和页面测试同步更新。
- 风险：迁移过渡期可能出现页面直接引用 `Api.*` 与模块导出并存。
- 防护：静态检查禁止 `static/src` 引用和 `Api.` 全局命名空间残留。

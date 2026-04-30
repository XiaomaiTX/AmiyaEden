---
status: draft
doc_type: draft
owner: engineering
last_reviewed: 2026-05-01
source_of_truth:
  - static/src/types/api/api.d.ts
  - static/src/api
  - static-react/src
  - docs/ai/repo-rules.md
---

# API 类型契约迁移方案（Vue3 -> React）

## 目标

- React 迁移期间保持 API 类型单一事实源，避免 Vue/React 双份接口类型漂移。
- 优先复用现有 `Api.*` 命名空间，后续再按模块增量演进为 React 友好组织形式。

## 当前基线

- 现有权威类型定义：`static/src/types/api/api.d.ts`。
- 现有 API 封装调用：`static/src/api/*.ts`，广泛依赖全局 `Api.*` 命名空间。
- React 子应用当前状态：已可通过 `static-react/src/types/api-contract.d.ts` 直接引用现有全局类型。

## 迁移原则

- 单一来源：迁移阶段只维护 `static/src/types/api/api.d.ts`，禁止复制到 `static-react`。
- 先类型后封装：先确保 React 页面能稳定使用同一套类型，再分批迁移 API wrapper。
- 小步切换：每迁一个业务模块，就在同一提交内对齐 `后端契约 -> 类型 -> wrapper -> 页面`。
- 禁止兼容分叉：除明确灰度需求外，不新增并行旧新字段别名。

## 执行步骤

### Step 1：类型接入（已完成）

- 在 `static-react/src/types/api-contract.d.ts` 通过 `import` 声明接入 `static/src/types/api/api.d.ts`。
- React 侧可直接使用 `Api.Auth.*`、`Api.SystemManage.*` 等现有命名空间类型。

### Step 2：封装基线（待做）

- 在 `static-react/src/api` 建立新的请求封装（与 Vue 的 `request` 行为语义保持一致：401、错误提示、重试策略）。
- 首批迁移 `auth`、`badge` 等低耦合 API wrapper，优先使用现有 `Api.*` 类型。

### Step 3：模块化收敛（进行中持续）

- 以业务域为单位引入 React 侧局部类型别名（例如 `AuthApi`、`FleetApi`），但底层仍指向 `Api.*`。
- 当某域完成全量迁移后，再评估是否把该域从全局命名空间逐步收敛到模块导出类型。

## DoD（完成定义）

- React 侧新增页面不再定义重复接口类型。
- 每个已迁模块均满足：
  - 使用 `Api.*` 或其受控别名；
  - 与后端响应字段一致；
  - Wrapper 与页面调用通过 TS 类型检查。
- 任一接口字段变更时，Vue/React 侧均在同一提交内完成类型对齐。

## 风险与防护

- 风险：`api.d.ts` 全局命名空间过大，改动影响面广。
- 防护：按模块迁移时只触达对应 namespace；变更必须附带该模块最小回归。
- 风险：React 团队可能绕过共享类型直接写局部 interface。
- 防护：Code Review 明确禁止同义重复类型，优先复用 `Api.*`。


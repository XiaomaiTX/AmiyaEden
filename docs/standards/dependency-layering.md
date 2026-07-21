---
status: active
doc_type: standard
owner: engineering
last_reviewed: 2026-03-23
source_of_truth:
  - docs/ai/repo-rules.md
  - server/internal
  - static/src
---

# 依赖分层规范

## 适用范围

本规范约束后端与前端各层之间的 import 方向，适用于本仓库的所有代码变更。

## 后端依赖方向

```text
model → repository → service → handler → router/middleware
  ↑                                            ↑
  pkg/* (共享基础设施)                          bootstrap/
```

### 后端规则

- `model`
  允许 import：标准库、GORM tag
  禁止 import：`repository`、`service`、`handler`、`router`、`middleware`
- `repository`
  允许 import：`model`、标准库、GORM、`pkg/*`
  禁止 import：`service`、`handler`、`router`、`middleware`
- `service`
  允许 import：`model`、`repository`、`pkg/*`、其他 service
  禁止 import：`handler`、`router`、`middleware`
- `handler`
  允许 import：`service`、用于请求/响应类型的 `model`、`pkg/response`
  禁止 import：直接 import `repository`
- `router`
  允许 import：`handler`、`middleware`、用于 DI 的 `service`
  禁止 import：直接 import `repository`
- `middleware`
  允许 import：用于角色常量的 `model`、`pkg/*`、用于鉴权的 `service`
  禁止 import：直接 import `handler`、`repository`
- `pkg/*`
  允许 import：标准库、外部包
  禁止 import：`internal/*`

### Handler 输入解析

- Handler 负责请求解析与类型转换，包括路径参数、查询参数与请求体。
- 当把来自请求的数字 ID 从 `uint64` 转换为 `uint` 时，handler 必须在强转之前拒绝大于 `math.MaxUint32` 的值。
- 不得在没有显式上界检查的情况下直接写 `uint(strconv.ParseUint(...))` 风格的转换。
- 该校验必须留在 handler 中，不要把请求解析推入 service。

推荐写法：

```go
id, err := strconv.ParseUint(c.Param("id"), 10, 64)
if err != nil || id > math.MaxUint32 {
    response.Fail(c, response.CodeParamError, "invalid id")
    return
}

typedID := uint(id)
```

### 后端快速规则

- `handler` 必须调用 `service`，不得直接调用 `repository`
- `repository` 必须保持纯数据访问；鉴权与编排属于 `service`
- `model` 不得依赖更高层级

## 前端依赖方向

```text
types → api → hooks/store → components → views
```

### 前端规则

- `types/`
  允许 import：无；仅允许纯类型定义
  禁止 import：`api/`、`hooks/`、`store/`、`components/`、`views/`
- `api/`
  允许 import：`types/`、HTTP client 工具
  禁止 import：`hooks/`、`store/`、`components/`、`views/`
- `hooks/`
  允许 import：`types/`、`api/`、`store/`、其他 hook
  禁止 import：`views/`、特定功能 `components/`
- `store/`
  允许 import：`types/`、`api/`、`hooks/`
  禁止 import：`views/`、`components/`
- `components/`
  允许 import：`types/`、`hooks/`、`store/`、其他 component
  禁止 import：`views/`、直接 import `api/`
- `views/`
  允许 import：以上所有层
  禁止被其他应用层 import

### 前端快速规则

- `views` 不得直接发起 HTTP 调用，必须使用 `api/`
- 共享契约必须放在 `types/`，不得放在 `views/` 下
- `components` 访问后端数据应通过 hook 或 store，不得直接 import `api/`

## 跨边界规则

### 后端 ↔ 前端契约

- 变更顺序：
  1. 后端请求或响应结构
  2. 如有需要，后端 service 逻辑
  3. 前端 `static/src/api/*.ts`
  4. 前端 `static/src/types/api/api.d.ts`
  5. 调用方 view 或 component
- 后端 JSON 字段名必须与前端类型定义完全一致
- 不得跨边界悄悄重命名字段

### 基础设施层（`pkg/*`）

`pkg/*` 是 `internal/*` 的共享基础设施。

- `pkg/*` 永远不得 import `internal/*`
- 若共享代码需要被 `internal/*` 使用的类型，应把类型上移到 `pkg/*`，或通过接口做 DI

## 执行

- 通过代码评审、代理校验与 `docs/standards/pre-completion-checklist.md` 执行
- 若违反点位于你正在改动的代码里，应在同一次变更中修复
- 若违反点与本次变更无关，记录下来但不要扩大范围
- 永远不要引入新的分层违反

## 提交前核对

- 没有新的从低层向高层的 import
- `handler` 未 import `repository`
- `repository` 不含业务逻辑
- `views` 未直接调用 HTTP
- `types/` 未 import 任何应用层
- `pkg/*` 未 import `internal/*`

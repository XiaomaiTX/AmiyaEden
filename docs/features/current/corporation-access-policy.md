---
status: active
doc_type: feature
owner: engineering
last_reviewed: 2026-05-16
source_of_truth:
  - server/internal/model/corporation_capability.go
  - server/internal/model/sys_config.go
  - server/internal/service/corporation_policy.go
  - server/internal/service/auto_srp.go
  - server/internal/middleware/auth.go
  - server/internal/router/router.go
  - server/internal/handler/me.go
  - server/internal/handler/sys_config.go
  - static/src/router/core/menuAccess.ts
  - static/src/views/system/basic-config/index.vue
---

# 军团能力策略（一期）

## 当前能力

- 在 `allow_corporations` 与 RBAC 之上新增军团能力策略层。
- 通过 `app.corporation_access_policies` 配置军团级 `capabilities` 与 `rules`。
- 后端在 JWT 链路注入 `primary_corporation_id`、`corp_capabilities`、`corp_rules`，并通过 `RequireCorpCapability` 强制鉴权。
- `/api/v1/me` 返回军团能力上下文，前端路由与菜单按 capability 严格收敛。
- 一期业务规则示例：`srp.recommendation_multiplier`（0~1，默认回退 1）。

## 配置结构

- `version: number`
- `default_mode: "deny"`（一期固定 deny）
- `policies: CorporationPolicy[]`

`CorporationPolicy`：

- `corporation_id: int64`
- `full_access: bool`
- `capabilities: string[]`
- `rules: Record<string, number | string | boolean>`

## 权限边界

- `super_admin` 永远放行 capability 检查。
- `full_access=true` 的军团跳过 capability 检查。
- 未命中策略按 `deny` 处理。
- 一期 capability 字典：
  - `srp.user`
  - `srp.manage`
  - `welfare.user`
  - `welfare.approval`
  - `welfare.settings`
  - `menu.srp`
  - `menu.welfare`

## SRP 规则说明

- 规则键：`srp.recommendation_multiplier`。
- 生效范围：SRP 推荐金额计算链路（提交申请与管理端推荐金额刷新共用路径）。
- 取值约束：`0 <= multiplier <= 1`，缺省或非法值回退 `1`。
- 该规则只影响 `recommended_amount` 计算，不改变已有审批覆盖金额流程。

## 主要接口

- `GET /api/v1/system/basic-config/corporation-access-policies`
- `PUT /api/v1/system/basic-config/corporation-access-policies`
- `GET /api/v1/me`（含 `primary_corporation_id`、`corp_capabilities`、`corp_rules`）

## 关键不变量

- 不改变 `allow_corporations` 的准入语义。
- 不替代现有 RBAC，能力策略仅叠加在职权之上。
- 前后端 capability 语义保持一致，不允许别名映射。
- 一期仅 `static/` 承接配置 UI 与 capability 门禁，`static-react/` 不纳入范围。

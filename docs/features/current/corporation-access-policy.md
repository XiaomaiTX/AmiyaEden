---
status: active
doc_type: feature
owner: engineering
last_reviewed: 2026-05-17
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

# 军团能力策略

## 当前能力

- 在 `allow_corporations` 与 RBAC 之上新增军团能力策略层。
- 通过 `app.corporation_access_policies` 配置军团级 `capabilities` 与 `rules`。
- 后端在 JWT 链路注入 `primary_corporation_id`、`corp_capabilities`、`corp_rules`，并通过 `RequireCorpCapability` 强制鉴权。
- `/api/v1/me` 返回军团能力上下文，前端路由与菜单按 capability 严格收敛。
- 业务规则示例：`srp.recommendation_multiplier`（0~1，默认回退 1）。
- 钱包能力示例：`wallet.user.enabled`；未启用时该军团成员伏羲币钱包余额恒为 0，且拒绝余额变动写入。

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
- capability 字典：
  - `menu.dashboard`
  - `menu.operation`
  - `menu.role`
  - `menu.newbro`
  - `menu.fuxi_hall`
  - `menu.ticket`
  - `menu.shop`
  - `menu.system`
  - `menu.info`
  - `menu.skill_planning`
  - `srp.user`
  - `srp.manage`
  - `welfare.user`
  - `welfare.approval`
  - `welfare.settings`
  - `menu.srp`
  - `menu.welfare`
  - `ticket.manage`
  - `shop.manage`
  - `system.manage`

## 路由与菜单能力边界

- 前端所有主要业务菜单域都通过 `meta.corpCapabilities` 叠加菜单能力控制；仅配置 `menu.srp + srp.user` 时，SRP 以外业务域不会出现在菜单中。
- 后端主要业务路由同步叠加 `RequireCorpCapability`，避免直连接口绕过前端菜单过滤。
- 管理动作使用域内管理能力进一步收敛：工单管理要求 `ticket.manage`，商店管理要求 `shop.manage`，系统管理要求 `system.manage`。
- `super_admin` 和 `full_access=true` 军团在 capability 层自动放行，但仍受其他业务规则约束。
- `/me`、SSO 人物绑定、通知等登录基础能力不挂业务菜单 capability，避免影响准入与启动上下文。

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
- 仅 `static/` 承接配置 UI 与 capability 门禁，`static-react/` 不纳入范围。

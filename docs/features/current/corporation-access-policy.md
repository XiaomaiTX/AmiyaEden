---
status: active
doc_type: feature
owner: engineering
last_reviewed: 2026-07-23
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
  - static/src/hooks/core/useCorpCapability.ts
  - static/src/views/system/basic-config/index.vue
  - static-react/src/app/route-access.ts
  - static-react/src/auth/route-access-gate.tsx
  - static-react/src/hooks/use-corp-capability.ts
---

# 军团能力策略

## 当前能力

- 在 `allow_corporations` 与 RBAC 之上新增军团能力策略层。
- 通过 `app.corporation_access_policies` 配置军团级 `capabilities` 与 `rules`。
- 后端在 JWT 链路注入 `primary_corporation_id`、`corp_capabilities`、`corp_rules`，并通过 `RequireCorpCapability` 强制鉴权。
- `/api/v1/me` 返回军团能力上下文，前端路由与菜单按 capability 严格收敛。
- 业务规则示例：`srp.recommendation_multiplier`（0~1，默认回退 1）。
- 钱包能力示例：`wallet.user.enabled`；未启用时该军团成员伏羲币钱包余额恒为 0，且拒绝余额变动写入。

## 能力目录：已强制 vs 仅注册

Stage 0A 之后，后端把 capability 目录分成两层：

- **已强制（enforced）**：被路由中间件 `RequireCorpCapability` 或服务层显式消费，共 42 项。`full_access=true` 与 `default_mode=allow` 在 `/me.corp_capabilities` 中只会展开为 enforced 集合。
- **仅注册（reserved）**：仅在目录中保留，后端不强制；策略写入时会被拒绝，但历史已存的保留键会在读接口中以 `legacy_capabilities` 返回，便于运维清理。

获取 enforced 目录：

- Go：`model.EnforcedCorpCapabilities()` / `model.IsEnforcedCorpCapability(cap string)`。
- HTTP：`GET /api/v1/system/basic-config/corporation-access-policies` 响应的 `enforced_capabilities` 字段。

## 配置结构

- `version: number`
- `default_mode: "allow" | "deny"`（默认 `allow`）
- `policies: CorporationPolicy[]`

`CorporationPolicy`：

- `corporation_id: int64`
- `full_access: bool`
- `capabilities: string[]`
- `rules: Record<string, number | string | boolean>`

写入语义：

- 后端在 `PUT` 时对 `capabilities` 强制 enforced 目录校验；reserved 键会被拒绝。
- 读取接口对历史 payload 做宽松解析，把 enforced 之外的键剥离到 `legacy_capabilities[corp_id]` 而非整条拒绝。

## 权限边界

- `super_admin` 永远放行 capability 检查。
- `full_access=true` 的军团跳过 capability 检查。
- 未命中策略按 `default_mode` 处理（默认 `allow`）。
- capability 字典：见 `model.enforcedCorpCapabilities`，共 42 项；包括已执法的菜单域、SRP、福利、工单、商店、系统、运营、信息、技能规划、新人/导师、伏羲殿等。

## 路由与菜单能力边界

- 前端路由 meta 通过 `corpCapabilitiesAll`（AND 语义）/`corpCapabilitiesAny`（OR 语义）显式声明每条路由所需 capability；旧 `meta.corpCapabilities` 已废弃。
- 前端路由、菜单与按钮只能引用 enforced capability。未被后端执法的 reserved 键不能作为父菜单的占位门禁；这类父菜单应使用其有效子能力的 OR 条件。
- 管理动作使用域内管理能力进一步收敛：工单管理要求 `ticket.manage` 与 `ticket.admin.read` 同时满足（AND），商店管理要求 `shop.manage` 与 `shop.admin.product.manage`（AND）。
- 前端按钮级门禁通过 `useCorpCapability`（Vue）或 `useCorpCapability`（React）显式 gate：`shop.order.create`、`ticket.user.reply`、`system.task.run`、`system.wallet.adjust`、`system.audit.export`。
- 后端主要业务路由同步叠加 `RequireCorpCapability`，避免直连接口绕过前端菜单过滤。
- `super_admin` 和 `full_access=true` 军团在 capability 层自动放行，但仍受其他业务规则约束。
- `/me`、SSO 人物绑定、通知等登录基础能力不挂业务菜单 capability，避免影响准入与启动上下文。

## SRP 规则说明

- 规则键：`srp.recommendation_multiplier`。
- 生效范围：SRP 推荐金额计算链路（提交申请与管理端推荐金额刷新共用路径）。
- 取值约束：`0 <= multiplier <= 1`，缺省或非法值回退 `1`。
- 该规则只影响 `recommended_amount` 计算，不改变已有审批覆盖金额流程。

## 审计与缓存

- `PUT /api/v1/system/basic-config/corporation-access-policies` 会写一条 `category=config`、`action=corporation_access_policy_update` 的审计事件，`details_json` 同时记录 before/after。
- 读取策略直接从 `sys_config`（共享 Redis 缓存 + DB fallback）拿；进程内 cache 已移除。
- `SyncConfigSuperAdmins` 在 grant/revoke 后会调用 `roleSvc.InvalidateUserCache` 清掉对应用户的 `user_roles` 缓存，避免 super-admin 变更在下一次登录前才生效。

## 主要接口

- `GET /api/v1/system/basic-config/corporation-access-policies`（含 `enforced_capabilities`、`legacy_capabilities`）
- `PUT /api/v1/system/basic-config/corporation-access-policies`
- `GET /api/v1/me`（含 `primary_corporation_id`、`corp_capabilities`、`corp_rules`）

## 关键不变量

- 不改变 `allow_corporations` 的准入语义。
- 不替代现有 RBAC，能力策略仅叠加在职权之上。
- 前后端 capability 语义保持一致，不允许别名映射。
- 已强制的 42 项之外的键一律视为 reserved；reserved 键不再写库。
- `/me.corp_capabilities` 永远只返回 enforced 集合；`full_access=true` 与 `default_mode=allow` 都按 enforced 集合展开。
- 迁移阶段 Vue 与 React 双前端共用同一份 enforced 目录与同一套 `corpCapabilitiesAll`/`corpCapabilitiesAny` 语义。

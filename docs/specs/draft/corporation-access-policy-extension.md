---
status: draft
doc_type: spec
owner: engineering
last_reviewed: 2026-07-22
source_of_truth:
  - server/internal/router/router.go
  - server/internal/middleware/auth.go
  - server/internal/service/role.go
  - server/internal/service/srp.go
  - server/internal/service/welfare.go
  - server/internal/handler/sys_config.go
  - server/internal/model/sys_config.go
  - server/internal/utils/allow_corporations.go
  - static/src/router/core/menuAccess.ts
  - static/src/router/modules/srp.ts
  - static/src/router/modules/welfare.ts
  - static/src/views/system/basic-config/index.vue
---

# 军团能力策略扩展草案（基于 allow_corporations）

> 实现状态（2026-07-23）：Stage 0A 已落地。当前实现以 `docs/features/current/corporation-access-policy.md`、`docs/architecture/auth-and-permissions.md`、`docs/architecture/routing-and-menus.md` 和 `docs/api/route-index.md` 为准。本草案保留原始设计讨论；其中把 `menu.srp`、`menu.welfare`、`menu.role`、`menu.system` 当作前端门禁，或使用旧 `meta.corpCapabilities` 的段落均已被当前实现取代，不得作为实现依据。

> ⚠️ 实现差异说明（2026-07-07）：本草案原设计为 **默认 `deny`**（最小权限）。**实际落地为默认 `allow`** —— 代码 `server/internal/service/corporation_policy.go`（`DefaultMode = corpPolicyDefaultModeAllow`）与 `docs/features/current/corporation-access-policy.md` 均使用 `default_mode: "allow"`。下文出现的「默认 deny / 一期固定使用 deny」是历史设计意图，保留作决策记录；**以 feature doc 与代码为准**。

## 背景

当前系统只有两层权限语义：

- `allow_corporations` 决定用户是 `guest` 还是 `user` 的准入基线。
- `roles` 决定页面/API 的职权权限。

现状缺口：

- 不能按军团 ID 精细控制“可访问页面/功能”。
- 不能按军团配置业务规则（如补损推荐金额比例）。
- 超级管理员缺少“军团 -> 能力/规则”的统一配置界面。

## 目标

- 在不破坏现有 `allow_corporations` 和 RBAC 的前提下，新增“军团策略层”。
- 支持同角色不同军团的能力差异化访问。
- 支持可扩展的业务规则机制，一期先接入 SRP 规则示例。
- 前后端同时生效：后端强制鉴权，前端菜单和路由做 UX 收敛。

## 范围

一期范围：

- 功能域：SRP、福利、菜单/页面可见性。
- 配置端：`super_admin` 在系统配置页维护军团策略。
- 规则端：提供通用规则容器，并落地 SRP 推荐金额乘数规则。

非目标：

- 一次性改造所有业务模块。
- 替换现有 RBAC 模型。
- 改变 `allow_corporations` 的准入语义。

## 决策约束

已确认决策：

- 执法层级：前后端都做。
- 军团归属：按主人物 `primary_character` 所属军团判定。
- 默认策略：最小权限（deny by default）。
- 一期实施范围：SRP + 福利 + 菜单页。
- SRP 仅作为规则引擎接入示例，机制应支持后续扩展到其他业务。

## 设计方案

### 1. 配置模型

新增 `system_config` key：

- `app.corporation_access_policies`

建议 JSON 结构：

- `version: number`
- `default_mode: "deny" | "allow"`
- `policies: CorporationPolicy[]`

`CorporationPolicy` 字段：

- `corporation_id: int64`
- `full_access: bool`
- `capabilities: string[]`
- `rules: map[string]number|string|bool`

策略解析规则：

- `super_admin` 永远全放行，不受军团策略限制。
- `full_access=true` 时该军团跳过 capability 检查。
- 未命中策略按 `default_mode` 处理；一期固定使用 `deny`。
- 配置更新后需要主动失效内存缓存。

### 2. Capability 字典

一期（已落地）：

- `srp.user`
- `srp.manage`
- `welfare.user`
- `welfare.approval`
- `welfare.settings`
- `menu.srp`
- `menu.welfare`

二期（计划扩展，粗粒度）：

- `menu.dashboard`
- `menu.operation`
- `menu.role`
- `menu.newbro`
- `menu.fuxi_hall`
- `menu.ticket`
- `menu.shop`
- `menu.system`
- `ticket.manage`
- `shop.manage`
- `system.manage`

约束：

- capability 为后端常量枚举，不接受自由字符串写入。
- 前后端共用同一 capability 语义，不做别名映射。
- 二期优先按菜单域粗粒度命名，必要时再补充 `*.manage` 管理能力。
- 后续若需更细能力控制，在保持向后兼容的前提下按域内拆分。

### 3. 后端鉴权

新增中间件：

- `RequireCorpCapability(capability string)`

执行顺序：

- 先执行现有 `RequireRole` 或 `RequireLoginUser`。
- 再执行 `RequireCorpCapability`。
- 两者均通过才允许访问。

路由接入（一期）：

- `/api/v1/srp/*` 用户侧接口增加 `srp.user`。
- `/api/v1/srp/*` 管理侧接口增加 `srp.manage`。
- `/api/v1/welfare/*` 用户侧接口增加 `welfare.user`。
- `/api/v1/system/welfare/applications`、`/review` 增加 `welfare.approval`。
- 福利设置写接口增加 `welfare.settings`。

### 4. 业务规则引擎

新增规则读取抽象：

- `GetRuleFloat(corporationID, key, defaultValue)`

一期规则键：

- `srp.recommendation_multiplier`（范围 0~1，默认 1）

一期生效点：

- SRP 推荐金额生成路径读取并乘算该比例。
- 该规则仅示例接入；引擎保持通用，不绑定 SRP 专用结构。

### 5. 前端接入（迁移期双端）

`GET /api/v1/me` 新增返回字段：

- `primary_corporation_id`
- `corp_capabilities`
- `corp_rules`

路由与菜单：

- 路由 `meta` 增加 `corpCapabilities?: string[]`。
- `menuAccess` 联合判断 `roles/login` 与 `corpCapabilities`。
- 菜单构建按 `menu.srp`、`menu.welfare` 控制显示。

配置页面：

- 在系统基础配置页新增“军团能力策略”编辑区。
- 以 `allow_corporations` 列表为军团来源。
- 支持 capability 勾选与规则值编辑。
- 保存时调用新策略接口。

### 6. 二期：全业务能力化（粗粒度）

根因复盘：

- 一期仅覆盖 SRP/福利 capability，其他模块当前仍按 `roles/login` 判定可见与可访问。
- 因此“只配置 `srp.user + menu.srp`”时，用户仍可能看到并访问其他满足角色条件的页面。

扩展策略：

- 将 capability 执法从 SRP/福利扩展到全业务菜单域。
- 前后端统一执行“角色 + capability”双门禁。
- 保持 `default_mode=deny`、`super_admin/full_access` 特判逻辑不变。

落地方式：

- 前端：为全业务路由模块补齐 `RouteMeta.corpCapabilitiesAll` / `RouteMeta.corpCapabilitiesAny`，菜单与路由过滤统一走 `menuAccess`。
- 后端：为对应业务路由组补齐 `RequireCorpCapability`，确保直链与接口调用均受控。
- 配置页：按业务域分组展示 capability，减少配置歧义与漏配。
- 配置页 i18n：能力项、分组标题、帮助文案、校验提示必须使用 i18n 文案键展示，不直接暴露 capability 原始 key（如 `srp.user`、`menu.srp`）。

### 7. 管理员配置指引（角色层 + 军团层）

权限模型：

- 角色层（RBAC）定义“理论上可做什么”（岗位上限）。
- 军团层（capability）定义“该军团允许启用什么”（范围收敛）。
- 最终权限判定为交集：`RoleAllowed AND CorpCapabilityAllowed`。

能力语义约定：

- `menu.*`：控制模块入口是否可见（菜单/路由入口）。
- `*.user`：控制普通用户功能是否可用（用户侧读写接口）。
- `*.manage` / `*.approval` / `*.settings`：控制管理动作是否可用（管理侧接口）。

推荐配置：

- SRP 普通成员：`srp.user`（父菜单使用 `srp.user` / `srp.manage` OR）。
- SRP 管理成员：`srp.user + srp.manage`（且角色满足管理角色约束）。
- 完全禁用某域：不配置对应 enforced capability。

反模式（需避免）：

- 父菜单使用未执法的 reserved `menu.*`：有效子能力无法让入口显示，易造成“后端允许但前端隐藏”错觉。
- 页面同时需要模块入口和页面能力时只配置其中一项：前端入口与后端 AND 门禁会不一致。
- 将军团层当作“授权层”使用：军团策略只能收敛范围，不应替代角色授予。

## 当前完成度（2026-05-17）

### 总体状态

- 一期范围已完成：后端能力执法、SRP 规则示例接入、前端能力门禁与策略配置 UI、相关文档同步。
- 已转正文档：`docs/features/current/corporation-access-policy.md`。

### ToDo（按状态）

- [x] 新增 `app.corporation_access_policies` 配置 key（`server/internal/model/sys_config.go`）。
- [x] 新增 capability 常量与合法性校验（`server/internal/model/corporation_capability.go`）。
- [x] 新增策略服务（配置读取/校验/匹配/缓存失效/用户策略上下文）（`server/internal/service/corporation_policy.go`）。
- [x] 在 JWT 鉴权链路注入 `primary_corporation_id`、`corp_capabilities`、`corp_rules`、`corp_full_access`（`server/internal/middleware/auth.go`）。
- [x] 新增 `RequireCorpCapability(capability string)` 中间件（`server/internal/middleware/auth.go`）。
- [x] 新增策略配置接口：
  - `GET /api/v1/system/basic-config/corporation-access-policies`
  - `PUT /api/v1/system/basic-config/corporation-access-policies`
  （`server/internal/handler/sys_config.go` + `server/internal/router/router.go`）
- [x] SRP 路由叠加 capability 门禁（`srp.user` / `srp.manage`）（`server/internal/router/router.go`）。
- [x] 福利路由叠加 capability 门禁（`welfare.user` / `welfare.approval` / `welfare.settings`）（`server/internal/router/router.go`）。
- [x] `/api/v1/me` 返回扩展 `primary_corporation_id`、`corp_capabilities`、`corp_rules`（`server/internal/handler/me.go`）。
- [x] 新增后端测试：
  - 中间件能力判定测试（`server/internal/middleware/auth_corporation_capability_test.go`）
  - 策略服务测试（`server/internal/service/corporation_policy_test.go`）
  - 策略接口测试（`server/internal/handler/sys_config_corporation_policy_test.go`）
  - `/me` 新字段断言（`server/internal/handler/me_test.go`）
  - 路由 capability 断言（`server/internal/router/router_test.go`）
- [x] SRP 规则引擎示例接入 `srp.recommendation_multiplier`（`server/internal/service/auto_srp.go` / `srp.go`）。
- [x] 前端能力门禁与策略配置 UI（`static/` 相关文件）。
- [x] API 文档更新（`docs/api/route-index.md`）。
- [x] 草案转正并同步 `docs/features/current/corporation-access-policy.md`。
- [ ] 二期前端全路由补 `corpCapabilities`（覆盖 SRP/福利以外模块）。
- [ ] 二期后端各业务路由补 `RequireCorpCapability`（与前端能力域对齐）。
- [ ] 配置页 capability 分组扩展，并同步 API/特性文档权限说明。
- [ ] 配置页 capability 展示完成 i18n（能力标签/分组标题/说明/校验文案），禁止显示原始 capability key。
- [ ] 新增跨域能力回归测试（菜单可见性 + 接口 200/403 一致性）。

### 说明

- 当前实现默认 `default_mode=allow`，同时支持显式 `deny`；具体行为以 `docs/features/current/corporation-access-policy.md` 为准。
- 一期实现历史上仅落在 `static/`（Vue）端；这是阶段性实施范围，不是产品行为限制。
- Stage 0A 已完成当前已迁移 React 页面与 Vue 的 capability/menu/button parity；剩余 React 切换阻断项以迁移基线中的未迁移范围和基础设施为准。

### 收口验证记录（2026-05-17）

- 后端特性相关测试通过：
  - `go test ./internal/middleware ./internal/router ./internal/service -run "CorporationPolicy|RequireCorpCapability|SRP|Corp"`
  - `go test ./internal/handler -run "CorporationAccessPolicies|MeResponse"`
- 前端特性相关测试通过：
  - `pnpm exec tsx --test src/router/core/menuAccess.test.ts`

## 代码文件级实施清单

### 后端（Go）

| 文件 | 变更点 | 目标 |
| --- | --- | --- |
| `server/internal/model/sys_config.go` | 新增 `app.corporation_access_policies` 配置 key 常量与默认说明 | 配置键统一入口 |
| `server/internal/model`（新增 capability 常量文件） | 定义一期 capability 枚举：`srp.user`、`srp.manage`、`welfare.user`、`welfare.approval`、`welfare.settings`、`menu.srp`、`menu.welfare` | 禁止自由字符串漂移 |
| `server/internal/service`（新增 corporation policy 服务文件） | 实现策略读取、校验、匹配、规则读取（含 `srp.recommendation_multiplier`） | 策略核心逻辑集中 |
| `server/internal/handler/sys_config.go` | 新增策略 GET/PUT handler 与请求校验结构 | 超管配置入口 |
| `server/internal/middleware/auth.go`（或新增中间件文件） | 新增 `RequireCorpCapability(capability string)`；把主军团、capabilities、rules 写入 context | 后端强制鉴权 |
| `server/internal/router/router.go` | SRP/福利路由挂 capability 中间件；新增策略配置路由 | 路由执法落地 |
| `server/internal/handler/me.go` | `/api/v1/me` 增加 `primary_corporation_id`、`corp_capabilities`、`corp_rules` | 前端判定数据源 |
| `server/internal/service/auto_srp.go` | 推荐金额链路接入 `srp.recommendation_multiplier`（含默认回退） | 规则引擎一期示例落地 |

### 前端（Vue + React + TS）

| 文件 | 变更点 | 目标 |
| --- | --- | --- |
| `static/src/types/api/api.d.ts` / `static-react/src/types/api/*` | 扩展 `/me` 与策略类型 | 双端类型契约同步 |
| `static/src/api/auth.ts` | 映射 `/me` 新字段到 `UserInfo` | 登录态能力数据注入 |
| `static/src/types/router/index.ts` | `RouteMeta` 增加 `corpCapabilities?: string[]` | 路由元数据承载能力要求 |
| `static/src/router/core/menuAccess.ts` | 菜单过滤增加 capability 判定 | 菜单收敛 |
| `static/src/router/modules/srp.ts` | 标注 `menu.srp`、`srp.user`、`srp.manage` | SRP 页面能力门禁 |
| `static/src/router/modules/welfare.ts` | 标注 `menu.welfare`、`welfare.user`、`welfare.approval`、`welfare.settings` | 福利页面能力门禁 |
| `static/src/api/sys-config.ts` | 新增策略 GET/PUT API 包装 | 配置页接口接入 |
| `static/src/views/system/basic-config/index.vue` | 新增“军团能力策略”编辑区（能力勾选 + 规则编辑 + 保存） | 超管可视化配置 |
| Vue locale / `static-react/src/i18n/messages/*` | 补齐策略配置与校验文案键 | 双端文案与 i18n 完整 |

### 测试与回归

| 文件 | 变更点 | 目标 |
| --- | --- | --- |
| `server/internal/router/router_test.go` | 增加 SRP/福利 capability 路由保护断言（role 通过但 capability 不通过时返回 403） | 防路由执法回退 |
| `server/internal/handler/me_test.go` | 断言 `/me` 新字段返回 | 合同字段稳定 |
| `server/internal/service`（新增 policy service 测试文件） | 覆盖策略解析、校验、default deny、super_admin 放行、规则读取默认值 | 核心逻辑回归保护 |
| `static/src/router/core/menuAccess.test.ts` | 新增 capability 过滤分支测试 | 菜单权限边界稳定 |

### 文档与契约对齐

| 文件 | 变更点 | 目标 |
| --- | --- | --- |
| `docs/api/route-index.md` | 增加策略配置接口与 `/me` 新字段说明 | API 文档一致性 |
| `docs/features/current/*`（实现转正后） | 补充军团策略能力说明与边界 | 草案转正收口 |
## API 草案

- `GET /api/v1/system/basic-config/corporation-access-policies`
- `PUT /api/v1/system/basic-config/corporation-access-policies`

权限：

- `RequireRole(super_admin)`

`GET /api/v1/me` 响应扩展：

- `primary_corporation_id: number`
- `corp_capabilities: string[]`
- `corp_rules: Record<string, number | string | boolean>`

## 兼容与迁移

- 若新配置不存在，按 `default_mode=deny` 解释。
- 为避免一次性锁死，初始化脚本应为伏羲军团写入 `full_access=true`。
- 现有 `allow_corporations` 不变，仍用于准入与基础职权调整。
- 现有角色权限不变，仅在其上叠加军团能力约束。

## 测试计划

后端单测：

- 配置解析、校验、缓存失效。
- capability 判定：`super_admin`、命中策略、未命中策略、default deny。
- SRP multiplier 规则生效与默认值回退。

后端路由测试：

- 同角色不同军团访问同一接口返回差异（200/403）。
- role 通过但 capability 不通过时应返回 403。

前端测试：

- `menuAccess` capability 判定分支。
- 菜单构建 capability 过滤。
- 配置页读写 payload 和回显一致性。
- 配置页 capability i18n 展示（分组、标签、说明、校验）不出现原始 key。

回归测试：

- `allow_corporations` 基线行为不变。
- 未改造模块不受影响。
- `super_admin` 全量访问不受策略限制。
- 关键能力组合回归：`menu.*` 与 `*.user/*.manage` 同时配置时行为一致；缺失任一能力时菜单或接口按预期收敛。

## 验收标准

- 超管可按军团配置能力矩阵并保存。
- 不同军团成员在 SRP/福利访问面上产生可观测差异。
- 仅配置 `srp.user` 的军团成员可见并访问 SRP 用户页面，不可访问非 SRP 域页面与接口。
- 未授权能力在后端被拒绝，前端入口隐藏或 403。
- 菜单显示结果与接口 403 结果一致，不出现“可见但不可调”或“不可见但可调”错位。
- 策略配置页所有 capability 相关展示均为可理解的 i18n 文案，不出现 `srp.user`、`menu.srp` 等原始 key。
- SRP 推荐金额可被军团规则按比例调整。
- 现有 RBAC 与准入基线行为保持兼容。

## 假设

- 主人物军团信息在 `eve_character.corporation_id` 中可用且及时更新。
- 军团策略只针对登录后功能访问，不替代 SSO 准入判断。
- Stage 0A 已完成 Vue/React 当前已迁移页面的等价 capability 门禁；后续 Vue 下线阻断项以 `docs/specs/draft/frontend-react-migration-plan/` 的未迁移范围和基础设施清单为准。
- 二期 capability 先按粗粒度落地，细粒度拆分作为后续增量演进。

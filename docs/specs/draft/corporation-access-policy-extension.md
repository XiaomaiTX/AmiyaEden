---
status: draft
doc_type: spec
owner: engineering
last_reviewed: 2026-05-15
source_of_truth:
  - server/internal/router/router.go
  - server/internal/middleware/auth.go
  - server/internal/service/role.go
  - server/internal/service/srp.go
  - server/internal/service/welfare.go
  - server/internal/handler/sys_config.go
  - server/internal/model/sys_config.go
  - server/internal/utils/allow_corporations.go
  - static-react/src/auth/route-access-gate.tsx
  - static-react/src/layout/menu-config.ts
  - static-react/src/pages/system-basic-config-page.tsx
---

# 军团能力策略扩展草案（基于 allow_corporations）

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

### 2. Capability 字典（一期）

- `srp.user`
- `srp.manage`
- `welfare.user`
- `welfare.approval`
- `welfare.settings`
- `menu.srp`
- `menu.welfare`

约束：

- capability 为后端常量枚举，不接受自由字符串写入。
- 前后端共用同一 capability 语义，不做别名映射。

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

### 5. 前端接入（React）

`GET /api/v1/me` 新增返回字段：

- `primary_corporation_id`
- `corp_capabilities`
- `corp_rules`

路由与菜单：

- 路由 `meta` 增加 `corpCapabilities?: string[]`。
- `RouteAccessGate` 联合判断 `roles` 与 `corpCapabilities`。
- 菜单构建按 `menu.srp`、`menu.welfare` 控制显示。

配置页面：

- 在系统基础配置页新增“军团能力策略”编辑区。
- 以 `allow_corporations` 列表为军团来源。
- 支持 capability 勾选与规则值编辑。
- 保存时调用新策略接口。

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

- `RouteAccessGate` capability 判定分支。
- 菜单构建 capability 过滤。
- 配置页读写 payload 和回显一致性。

回归测试：

- `allow_corporations` 基线行为不变。
- 未改造模块不受影响。
- `super_admin` 全量访问不受策略限制。

## 验收标准

- 超管可按军团配置能力矩阵并保存。
- 不同军团成员在 SRP/福利访问面上产生可观测差异。
- 未授权能力在后端被拒绝，前端入口隐藏或 403。
- SRP 推荐金额可被军团规则按比例调整。
- 现有 RBAC 与准入基线行为保持兼容。

## 假设

- 主人物军团信息在 `eve_character.corporation_id` 中可用且及时更新。
- 军团策略只针对登录后功能访问，不替代 SSO 准入判断。
- 一期仅 React 端承接策略配置 UI，旧 Vue 端先不扩展该编辑能力。
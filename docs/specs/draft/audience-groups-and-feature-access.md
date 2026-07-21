---
status: draft
doc_type: draft
owner: engineering
last_reviewed: 2026-07-21
source_of_truth:
  - server/pkg/jwt/jwt.go
  - server/internal/model/role.go
  - server/internal/model/corporation_capability.go
  - server/internal/middleware/auth.go
  - server/internal/router/router.go
  - server/internal/service/role.go
  - server/internal/service/eve_sso.go
  - server/internal/service/corporation_policy.go
  - server/internal/handler/me.go
  - static/src/api/auth.ts
  - static/src/router/core/menuAccess.ts
  - static/src/router/modules
  - static-react/src/app/route-access.ts
  - static-react/src/app/migration-routes.ts
related_docs:
  - docs/architecture/auth-and-permissions.md
  - docs/architecture/routing-and-menus.md
  - docs/features/current/corporation-access-policy.md
  - docs/specs/draft/frontend-react-migration-plan/index.md
---

# 用户分组与统一 Feature Access 草案

> 本文记录 2026-07-21 对认证、职权、军团能力和前端门禁代码的检查结果，并提出尚未实现的用户分组 / 标签方案。本文是提案，不代表当前已实现行为；当前行为仍以代码和 `docs/architecture/`、`docs/api/`、`docs/features/current/` 为准。

## 当前状态与证据约定

- 已实现：EVE SSO、JWT、静态 canonical Role、多职权、军团 capability、Vue 静态路由门禁、独立业务资格检查。
- 部分实现：React 迁移前端已有 Login / Role 门禁，但尚未承接军团 capability。
- 未实现：Audience Group、成员申请 / 审核、Feature Audience Policy、统一 `feature_access` 与 `RequireFeatureAccess`。
- “已确认事实”均由当前代码核对；“提案”是未来设计；“风险 / 推断”需在实施时继续验证。
- 本次只新增草案，不修改运行时代码、数据库或接口。

## 1. 现有实现概览

| 层 | 已确认事实 | 主要入口 |
| --- | --- | --- |
| 身份 | SSO 后签发 JWT；请求时不信任 token 内的兼容单值 Role，而是重新加载 `user_role` | `eve_sso.go`、`jwt.go`、`middleware/auth.go` |
| Login | `RequireLoginUser()` 要求至少一个非 `guest` Role；`/me`、人物绑定等只要求 JWT | `middleware/auth.go`、`router/router.go` |
| Role | Go 常量定义，多职权；`super_admin` 是 `RequireRole` 特例 | `model/role.go`、`service/role.go` |
| Corporation Capability | 按主人物军团读取 `app.corporation_access_policies` | `service/corporation_policy.go` |
| Business Eligibility | 新人、导师学员、ESI token、资料、用户状态和业务状态机独立判断 | `handler/me.go` 及各业务 service |
| Vue UX | `/me` 写入 Pinia，动态注册前按 `meta.login`、`roles`、`corpCapabilities` 过滤 | `static/src/api/auth.ts`、`router/core/menuAccess.ts` |
| React UX | 迁移壳只消费 Login、Role、资格和静态 `authList`，没有 capability | `static-react/src/app/route-access.ts`、`auth/route-access-gate.tsx` |

当前 `server/internal/model/role.go` 实际定义 12 个 Role：

```text
super_admin, admin, senior_fc, fc, srp, shop_order_manage,
welfare, fuel_officer, captain, mentor, user, guest
```

Role 应继续由代码定义，不应为了兴趣、白名单或标签改成数据库动态 Role。

当前文档也有漂移：

- `docs/features/current/corporation-access-policy.md` 只列早期 20 项 capability，代码已注册 66 项。
- 该文档声称前端会对 `full_access=true` 自动放行；当前 `/me` 不返回 `corp_full_access`，Vue 也不消费它。
- 旧草案中的历史默认值是 deny；当前 `defaultCorpPolicyConfig()` 的缺省值是 allow。
- 权威架构仍把 Vue `static/` 视为产品前端；React 是迁移草案实现，但其权限缺口必须成为切换阻断项。

## 2. 当前权限链路

### 2.1 JWT 与 Role

```text
EVE SSO callback
  -> allow_corporations 决定初始 guest / user
  -> ESI affiliation / corp role 与 auto-role 同步
  -> SyncConfigSuperAdmins()
  -> jwt.GenerateToken(uid, primary_character_id, legacy user.role, expiry)
  -> 请求进入 JWTAuth()
  -> Redis user_roles:{userID}，miss 时查 user_role，TTL 30 分钟
  -> Gin Context 写入 userID、characterID、roles
```

`jwt.Claims.Role` 只是历史兼容字段；实际多职权来自数据库 / Redis。因此 Group 不应写入 JWT，否则撤销、过期和归档会被 token 生命周期拖延。

`SetUserRoles()`、auto-role 和准入重算会清除 Role 缓存。一个已确认例外是 `SyncConfigSuperAdmins()` 直接调用 repository 增删 Role，却不清除 `user_roles:*`；登录前若已有缓存，super-admin 授予 / 撤销可能延迟最多 30 分钟。统一 Feature Access 不能复制此类旁路写入。

### 2.2 主军团与 policy

`JWTAuth()` 每次请求调用 `BuildUserPolicyContext(userID)`：

1. 读取 `user.primary_character_id`。
2. 读取主人物的 `eve_character.corporation_id`。
3. 从进程内 `corpPolicyCache` 读取策略，miss 时查 `system_config`。
4. 命中策略时返回 `Capabilities`、`Rules`、`FullAccess`。
5. 未命中且 `default_mode=allow` 时，用 `FullAccess=true` 表达军团层全放行。

策略更新只清除当前进程内存缓存，没有 TTL 或跨实例失效。单实例可立即生效；多实例可能长期读取旧策略。

### 2.3 Gin Context 与 `/me`

`JWTAuth()` 注入：

```text
userID, characterID, roles, primaryCorporationID,
corpCapabilities, corpRules, corpFullAccess
```

`MeHandler.GetMe()` 返回前六项业务数据中的 `roles`、`primary_corporation_id`、`corp_capabilities`、`corp_rules`，但不返回 `corp_full_access`。结果是：

- `full_access=true` 的普通用户后端通过全部 capability，Vue 可能隐藏菜单；
- `default_mode=allow` 且未命中策略的普通用户也有同样问题。

### 2.4 后端中间件顺序

共同父链为 `JWTAuth -> RequireLoginUser（可选）`，之后各路由按声明顺序执行 Role / capability。当前没有统一坚持 Role 在 capability 之前，例如 SRP 写路由先 capability 后 Role。

目标顺序应是：

```text
JWTAuth
AND RequireLoginUser（若需要）
AND RequireRole（若需要）
AND RequireFeatureAccess
AND service 内的 Business Eligibility
```

Feature Access 只收敛功能范围，不能代替 Role 或业务状态机。

### 2.5 Vue 门禁

```text
/me -> static/src/api/auth.ts -> Pinia
  -> MenuProcessor.getMenuList()
  -> applyMenuAccessFilter()
     -> login
     -> roles（任一命中）
     -> corpCapabilities（任一命中）
     -> 新人 / 导师学员资格
  -> 只注册过滤后的动态路由
```

`meta.corpCapabilities` 数组由 `some()` 计算，即数组内部是 OR；父子路由之间才是 AND。`v-auth` / `useAuth()` 也不读取服务端权限或 capability，只判断当前路由静态 `authList` 是否含标记。新模型中按钮必须显式组合 Role、Feature Access 和业务状态。

动态路由只在初始化时注册。之后 `/me` 即使刷新 store，Role / capability 变化也不会自动重建路由树；后端仍安全拒绝，但菜单可能陈旧。

### 2.6 React 门禁

React 的 `MeResponse`、Zustand、route meta、`RouteAccessGate` 和 sidebar 都没有 `corp_capabilities` / `feature_access`。若直接切换，会系统性出现入口可见而后端 403。React 切换不得早于 Feature Access 对齐。

## 3. Capability 使用清单

`model/corporation_capability.go` 注册 66 项。下表“后端”只统计实际路由中间件或业务 service，“Vue”只统计实际路由过滤，不把类型 / 配置 UI 字符串算作执法。

| 分类 | 完整 key 清单 | 后端 | Vue | 统一 key 建议 |
| --- | --- | --- | --- | --- |
| SRP | `srp.user`、`srp.manage` | 路由 | 路由 | 复用；manage 仅 `managed_only` |
| 福利 | `welfare.user`、`welfare.approval`、`welfare.settings` | 路由 | 路由 | 复用；审批 / 设置仅 `managed_only` |
| 菜单双端 | `menu.dashboard`、`menu.operation`、`menu.newbro`、`menu.fuxi_hall`、`menu.ticket`、`menu.shop`、`menu.info`、`menu.skill_planning` | 路由组 | 父路由 | 核对全域前先 `none` |
| 菜单仅 Vue | `menu.srp`、`menu.welfare`、`menu.role`、`menu.system` | 无 | 父路由 | 暂不可绑定 Audience Policy |
| 粗管理 | `ticket.manage`、`shop.manage`、`system.manage` | 路由组 | 路由 | 复用，需先修 OR / AND |
| 信息域 | `info.wallet.read`、`info.npc_kills.self`、`info.npc_kills.corp`、`info.skills.read`、`info.assets.read`、`info.contracts.read`、`info.fittings.manage` | 路由 | 路由 | 复用，逐项定敏感度 |
| 商店用户 | `shop.wallet.read`、`shop.order.read_self` | 路由 | 路由 | 复用 |
| 商店购买 | `shop.order.create` | 路由 | 无按钮检查 | 复用，先补按钮门禁 |
| 钱包开关 | `wallet.user.enabled` | 业务 service | 无 | 保持业务规则，第一版 `none` |
| 仪表盘 | `dashboard.npc_kills.corp` | 路由 | 路由 | 复用，先修组合语义 |
| 建筑 | `dashboard.corp_structures.read`、`dashboard.corp_structures.manage` | 无 | read 有，manage 无 | 暂不可绑定 |
| 舰队预留 | `operation.fleet.read_self`、`operation.fleet.manage`、`operation.fleet.pap.manage` | 无 | 无 | 暂不可绑定 |
| 技能计划预留 | `skill_planning.corp.read`、`skill_planning.corp.manage`、`skill_planning.personal.read`、`skill_planning.personal.manage` | 无 | 无 | 暂不可绑定 |
| 新人 / 导师预留 | `newbro.user.actions`、`newbro.captain.actions`、`newbro.admin.read`、`newbro.admin.manage`、`mentor.user.actions`、`mentor.mentor.actions`、`mentor.admin.manage` | 无 | 无 | 暂不可绑定，且不能代替实时资格 |
| 系统任务 | `system.task.read`、`system.task.run` | 路由 | read 绑错页，run 无按钮检查 | 复用，先修 Vue |
| 基础配置 | `system.basic_config.read`、`system.basic_config.manage` | 路由 | read 绑在 auto-role，manage 无 | 复用，仅 `managed_only` |
| 系统钱包 | `system.wallet.read`、`system.wallet.adjust` | 路由 | read 绑错页，adjust 无 | 复用，仅 `managed_only` |
| 审计 | `system.audit.read`、`system.audit.export` | 路由 | read 绑错页，export 无 | 复用，仅 `managed_only` |
| 工具书签预留 | `system.tool_bookmark.read`、`system.tool_bookmark.manage` | 无 | 无 | 实际依赖其他 key，暂不可绑定 |
| 工单用户 | `ticket.user.create`、`ticket.user.reply` | 路由 | create 有，reply 无按钮检查 | 复用 |
| 工单管理 | `ticket.admin.read`、`ticket.admin.manage` | 路由 | 路由 | 复用，仅 `managed_only` |
| 商店管理 | `shop.admin.product.manage`、`shop.admin.order.manage` | 路由 | 路由 | 复用，仅 `managed_only` |
| 大厅预留 | `fuxi_hall.public.read`、`fuxi_hall.admin.manage` | 无 | 无 | 暂不可绑定 |

汇总：35 项后端和 Vue 都有运行时引用，5 项仅 Vue，6 项仅后端路由，1 项仅后端业务 service，19 项只是注册但未执法。字符串命名一致，漂移发生在挂载位置、组合语义和未接入状态。

统一 Feature Catalog 需要区分 Key 的语义，而不是把全部 capability 视为同类权限：

```text
key
kind = menu | resource | action | business_switch
display_i18n_key
description_i18n_key
expose_to_client
audience_grant_scope = none | super_admin_managed | admin_managed | self_enrollable
legacy_menu_key
```

约束：

- `menu` 只用于迁移期导航兼容，不作为后端安全边界，也不可绑定 Audience Policy；长期由子资源访问结果聚合菜单可见性。
- `resource` 和 `action` 可以作为后端原子授权目标。
- `business_switch` 继续由业务 service 判断，默认不可绑定 Audience Policy。
- `audience_grant_scope` 同时限制可引用的 Group 类型和成员维护主体，避免通过维护成员间接扩权。
- `expose_to_client=false` 的后端内部 Key 不进入 `/me.feature_access`。

`enforcement_status = aligned | backend_only | frontend_only | catalog_only` 仍应保留为检查结果，但不能由管理员或开发者手工维护。应由机器可读 Catalog、代码生成和 CI 扫描产生。只有检查结果为 `aligned` 且 `audience_grant_scope != none` 的 Key 才能启用 Audience Policy。现有 66 项初始全部设置为 `none`，逐项核对后再开放，确保新增分组能力默认不改变现有权限。

## 4. 发现的问题

| 优先级 | 已确认问题 | 影响 |
| --- | --- | --- |
| P0 | `full_access/default allow` 没有等价传给 Vue | 菜单隐藏但 API 可访问 |
| P0 | Vue 同数组 OR，后端连续中间件 AND | 商店、工单、军团 NPC 报表可见后 403 |
| P0 | Vue 系统页 key 错位 | 用户页绑 `task.read`，任务页绑 `wallet.read`，钱包绑 `audit.read`，审计只绑 `system.manage` |
| P1 | `shop.order.create`、`ticket.user.reply`、`system.task.run`、`system.wallet.adjust`、`system.audit.export` 等无按钮门禁 | 用户点击后才收到 403 |
| P1 | 19 个预留 key 可在配置 UI 勾选但不执法 | 管理员误判配置已生效 |
| P1 | React 完全忽略 capability | 切换后入口范围扩大 |
| P1 | 后端任意 `RequireRole` 都让 super-admin 通过，Vue Captain / Mentor 路由未包含它 | API 可调但入口隐藏 |
| P1 | super-admin 同步旁路 Role 缓存失效 | 授予 / 撤销延迟 |
| P1 | corporation policy 仅本进程缓存失效 | 多实例长期不一致 |
| P2 | corporation policy 更新未写现有业务审计 | 敏感配置缺少变更记录 |
| P2 | 权限快照变化不重建 Vue 动态路由 | 菜单在刷新 / 重登前陈旧 |

`RequireCorpCapability` 本身仍只判断军团能力，没有重复 JWT、Role 或业务状态。问题主要在上下文传播和路由挂载，因此不应继续扩张它；应新增上层 `RequireFeatureAccess`，保留军团 evaluator 为独立 service。

上述 P0/P1 缺陷必须先建立回归测试并修复。Audience Group 不应建立在当前前后端门禁仍不一致的基线上。

## 5. 推荐目标架构

```text
Identity / Login
  AND Role
  AND FeatureAccess(CorporationAccess AND AudienceAccess)
  AND Business Eligibility / Resource State
```

- Role：管理职权，继续使用 canonical Role。
- Corporation Capability：主人物军团允许的功能，保持现有配置和 `corp_rules` 语义。
- Audience Group：用户所属受众、兴趣或人工资格，不产生 Role。
- Feature Access：组合军团能力与用户分组策略，不重复身份、Role 或业务状态判断。
- Business Eligibility：新人、导师学员、ESI token、资料、用户状态和业务状态机，继续独立计算。

现有 `fuel_officer`、`srp`、`welfare` 等 Role 第一版不迁移为 Group。只要某个 Role 当前授予管理动作，就继续属于 Role。

### 5.1 核心不变量

- Group 不写 `user_role`，不参与 `guest/user/admin`、Role 排序、auto-role 或 super-admin 同步。
- Group 不能绕过 `RequireRole`，也不能单独授予审批、删除、配置、调账等管理动作。
- super-admin 通过 CorporationAccess 和 AudienceAccess，但仍受目标保护、数据约束、ESI 状态和业务状态机限制。
- `full_access=true` 只令 CorporationAccess 为 true，不绕过 AudienceAccess。
- 无 Audience Policy 或 Policy disabled 时 AudienceAccess=true。
- Audience Policy 的全量拒绝必须使用显式 `effect=deny_all`，不得通过空 Block 隐式表达。
- 条件块之间为 OR，同一条件块内 Group 之间为 AND；第一版不引入 deny、权重、继承、优先级或脚本。
- 前端只做 UX 收敛，后端始终执行相同的原子 Feature 检查。

### 5.2 三态决策

Feature Access 不能只返回布尔值，必须区分：

```text
ALLOW
DENY
INDETERMINATE
```

推荐结果结构：

```text
FeatureAccessDecision
- decision
- feature_key
- corporation_result
- audience_result
- matched_block
- reason_codes
- policy_revision_id
- policy_version
```

HTTP 行为：

| 决策 | 行为 |
| --- | --- |
| `ALLOW` | 继续执行请求 |
| `DENY` | 返回 `403 Forbidden` |
| `INDETERMINATE` | 返回 `503 Service Unavailable` |

数据库、上下文或规则加载错误不能解释为“无策略”，也不能伪装成普通权限不足。

### 5.3 Service 分层

新增：

```text
AccessContextLoader
FeatureAccessEvaluator
FeatureAccessService
RequireFeatureAccess(featureKey)
```

职责：

- `AccessContextLoader`：加载 Role、主人物军团、军团策略和当前有效 Group。
- `FeatureAccessEvaluator`：纯计算，不访问数据库、Redis、Gin Context，不写审计，不生成 HTTP 响应。
- `FeatureAccessService`：编排上下文加载、Catalog、Policy 和批量计算。
- `RequireFeatureAccess`：把决策转换为继续、403 或 503；不重复 JWT、Login、Role 和业务资格检查。

目标路由顺序：

```text
JWTAuth
AND RequireLoginUser（若需要）
AND RequireRole（若需要）
AND RequireFeatureAccess
AND service 内 Business Eligibility
```

阶段 1 中 AudienceAccess 恒 true，新旧后端 evaluator 必须逐 Key 等价。

## 6. Feature Catalog

### 6.1 单一来源

Feature Key 必须有编译期单一来源。推荐建立：

```text
permissions/features.json
```

由代码生成：

```text
server/internal/model/feature_key_gen.go
static/src/constants/feature-key.gen.ts
static-react/src/constants/feature-key.gen.ts
```

运行时 `GET /api/v1/system/feature-catalog` 只用于管理页面展示，不替代编译期类型检查。

CI 应检查：

- 后端路由和 service 引用；
- Vue 路由和按钮引用；
- React 路由和按钮引用；
- 未注册字符串；
- `backend_only`、`frontend_only`、`catalog_only`；
- Vue / React 同一路由的 Key 与组合语义差异。

`enforcement_status` 是生成的检查结果，不是可编辑配置。

### 6.2 Key 类型

```text
kind = menu | resource | action | business_switch
```

| kind | 语义 | 后端执法 | Audience Policy |
| --- | --- | ---: | ---: |
| `menu` | 导航兼容标记 | 否 | 否 |
| `resource` | 页面、模块或资源访问 | 是 | 可配置 |
| `action` | 明确写入或管理动作 | 是 | 可配置 |
| `business_switch` | 军团或业务参数开关 | service 内 | 默认禁止 |

`menu.*` 迁移期标记为 legacy。长期菜单可见性由“至少一个子资源可访问”派生，不能把菜单 Key 当成安全边界。

### 6.3 用户分组授权范围

```text
audience_grant_scope =
  none |
  super_admin_managed |
  admin_managed |
  self_enrollable
```

| grant scope | 可引用 Group |
| --- | --- |
| `none` | 不可绑定 Audience Policy |
| `super_admin_managed` | `managed_tag + membership_authority=super_admin_only` |
| `admin_managed` | `managed_tag + membership_authority=admin/super_admin_only` |
| `self_enrollable` | 可以引用符合配置的 `interest_group` |

敏感操作应使用 `super_admin_managed`，包括但不限于调账、审计导出、基础配置、Webhook、模拟登录和删除类动作。

开放兴趣组仅适合非敏感功能试用、主动订阅、个性化入口或低风险功能。

## 7. 数据模型建议

### 7.1 `audience_group`

```text
id
code
name
description
kind = managed_tag | interest_group
visibility = listed | hidden
join_policy = open | approval | admin_only
status = active | suspended | archived
membership_authority = super_admin_only | admin
created_by
updated_by
created_at
updated_at
```

约束：

- `code` 唯一且创建后不可修改。
- Group 不物理删除。
- `managed_tag` 强制 `hidden + admin_only`。
- `membership_authority=super_admin_only` 的 Group 对 admin 隐藏，admin 不能查看或管理成员。
- `join_policy=open` 时 `visibility` 必须为 `listed`。
- `suspended` 是可恢复暂停状态；成员暂时不生效，恢复后原 active 且未过期成员重新生效。
- `archived` 是终态，第一版不提供恢复；需要重新启用时创建新 Group 并重新确认成员与策略。
- suspended / archived Group 都不参与有效成员匹配。

### 7.2 `audience_group_membership`

```text
id
group_id
user_id
status = pending | active | rejected | withdrawn | left | revoked
source = admin_assignment | self_join | application_approval
expires_at
requested_at
decided_at
version
created_by
updated_by
created_at
updated_at
```

索引：

```text
UNIQUE(group_id, user_id)
INDEX(user_id, status, expires_at)
INDEX(group_id, status, expires_at)
```

有效成员必须由 SQL 直接筛选：

```sql
status = 'active'
AND (expires_at IS NULL OR expires_at > NOW())
```

同时要求 Group `status=active`。不能加载全部记录后在 Go 中过滤，也不能依赖定时任务修改过期状态。

Membership 使用 `version` 乐观锁，避免批准、撤回、退出和撤销等并发操作互相覆盖。

### 7.3 Membership 状态机

| 当前状态 | 操作 | 新状态 | 操作者 |
| --- | --- | --- | --- |
| 无记录 | `apply` | `pending` | 用户 |
| 无记录 | `join` | `active` | 用户，仅 open Group |
| 无记录 | `assign` | `active` | 有权限管理员 |
| `pending` | `withdraw` | `withdrawn` | 用户本人 |
| `pending` | `approve` | `active` | 有权限管理员 |
| `pending` | `reject` | `rejected` | 有权限管理员 |
| `rejected` / `withdrawn` | `reapply` | `pending` | 用户本人 |
| `active` | `leave` | `left` | 用户本人，仅允许主动退出的兴趣组 |
| `active` | `revoke` | `revoked` | 有权限管理员 |
| `left` | `join` | `active` | 用户，仅 open Group |
| `revoked` | `assign` | `active` | 有权限管理员 |

Service 必须集中校验：

- Group 类型、可见性和加入策略；
- 操作者权限；
- 是否允许主动退出；
- `expires_at` 的设置与清理；
- `requested_at`、`decided_at` 的更新；
- `version` 冲突；
- 同事务审计。

Handler 不得直接修改 Membership 状态。

### 7.4 Audience Policy 与不可变 Revision

```text
feature_audience_policy
- id
- feature_key unique
- enabled
- current_revision_id
- version
- created_by
- updated_by
- created_at
- updated_at

feature_audience_policy_revision
- id
- policy_id
- version
- effect = rules | deny_all
- created_by
- created_at

feature_audience_policy_revision_block
- id
- revision_id
- sort

feature_audience_policy_revision_block_group
- block_id
- group_id
```

索引与约束：

```text
UNIQUE(policy_id, version)
UNIQUE(revision_id, sort)
UNIQUE(block_id, group_id)
INDEX(feature_audience_policy.feature_key)
```

规则：

- `effect=rules` 时至少包含一个 Block，每个 Block 至少包含一个 Group。
- `effect=deny_all` 时不得包含 Block；该操作必须在 UI 中显式标记为全量封锁并二次确认。
- Block 之间为 OR，Block 内 Group 之间为 AND。
- 新 Revision 禁止引用 suspended / archived Group。
- 已引用 Group 后续 suspended / archived 时，该 Group 条件不匹配。
- Policy 更新必须验证 Feature Catalog 的 `kind`、`audience_grant_scope`、客户端暴露范围和当前执法状态。
- Policy 更新使用 `expected_version` 乐观锁。
- 每次变更创建不可变 Revision，并在同一事务中切换 `current_revision_id`、提升 version 和写审计。
- 外键使用 restrict；不级联物理删除历史策略和审计数据。

Model 建议落在：

```text
server/internal/model/audience_group.go
server/internal/model/feature_audience_policy.go
```

通过 `AutoMigrate` 创建新表和索引；涉及后续删除或破坏性修改时必须使用显式 migration，不依赖 `AutoMigrate`。

## 8. 后端改造范围

| 层 | 提案 |
| --- | --- |
| Catalog | 机器可读 Feature 定义、Go/TS 生成、CI 扫描 |
| Model | Group、Membership、Policy、Revision、Block、BlockGroup 与受控枚举 |
| Repository | CRUD、带状态和有效期的批量查询、Policy Revision 聚合；不写业务规则 |
| Service | 状态机、管理矩阵、Catalog 校验、Context Loader、Evaluator、事务审计 |
| Middleware | `RequireFeatureAccess(featureKey)`；不重复 JWT、Login、Role、资格 |
| Handler | 请求绑定与响应；不直接写状态 |
| Router | 自助接口要求 Login；Group 管理要求 admin；Policy 写仅 super-admin |
| `/me` | 新增权威 `feature_access` 和 `feature_access_status`；迁移期保留旧字段和 `corp_rules` |
| Audit | 关键写入与业务数据同事务调用 `RecordEventTx` |
| Observability | 按 Key 统计 ALLOW、DENY、INDETERMINATE、耗时和 shadow mismatch |

建议接口：

```text
GET  /api/v1/audience-groups
GET  /api/v1/audience-groups/me
POST /api/v1/audience-groups/:code/join
POST /api/v1/audience-groups/:code/apply
POST /api/v1/audience-groups/:code/withdraw
POST /api/v1/audience-groups/:code/leave

GET|POST /api/v1/system/audience-groups
PUT      /api/v1/system/audience-groups/:id
POST     /api/v1/system/audience-groups/:id/suspend
POST     /api/v1/system/audience-groups/:id/resume
POST     /api/v1/system/audience-groups/:id/archive
GET      /api/v1/system/audience-groups/:id/members
GET      /api/v1/system/audience-groups/:id/applications
PUT      /api/v1/system/audience-groups/:id/members/:user_id
DELETE   /api/v1/system/audience-groups/:id/members/:user_id
POST     /api/v1/system/audience-groups/:id/applications/:user_id/approve
POST     /api/v1/system/audience-groups/:id/applications/:user_id/reject

GET /api/v1/system/feature-catalog
GET /api/v1/system/feature-audience-policies
GET /api/v1/system/feature-audience-policies/:feature_key/revisions
PUT /api/v1/system/feature-audience-policies/:feature_key
POST /api/v1/system/feature-audience-policies/:feature_key/rollback/:revision_id
```

普通用户只看 listed interest group 和自己的可见状态；看不到 managed tag、hidden group、他人成员或策略。admin 只管理 `membership_authority=admin` 的 Group，不能修改 Policy。super-admin 管理全部。

不在 `JWTAuth()` 中无条件加载 Group。`RequireFeatureAccess` 首次需要时加载本请求有效 Group，并写入请求级 Context 复用；`/me` 使用批量查询计算全部客户端可见 Key，避免 N+1。

### 8.1 `/me` 失败语义

建议返回：

```json
{
  "feature_access": {},
  "feature_access_status": "ready"
}
```

状态：

```text
ready
unavailable
```

当权限上下文无法可靠计算时：

- `/me` 仍返回身份、人物和 JWT-only 自助所需数据；
- `feature_access_status=unavailable`；
- `feature_access` 不返回正向权限；
- 前端隐藏全部受 Feature Access 保护的业务功能并显示异常提示；
- 人物管理、资料修复和自助注销等 JWT-only 能力仍可使用；
- 实际业务接口返回 503，而不是 403。

## 9. 前端改造范围

### 9.1 通用契约

前端只消费后端生成的：

```text
feature_access: Record<FeatureKey, boolean>
feature_access_status: ready | unavailable
```

不自行组合 `corp_capabilities`、Group 和 Policy。

路由元数据显式区分：

```ts
featureAccessAll?: FeatureKey[]
featureAccessAny?: FeatureKey[]
```

判定：

```text
全部 featureAccessAll 满足
AND
featureAccessAny 为空或至少满足一项
```

阶段 0 修复旧 `corpCapabilities` 时必须逐路由确认真实 OR / AND 语义，不能全局把 `some()` 替换成 `every()`。

按钮权限统一为：

```text
Role
AND FeatureAccess
AND Business State
```

### 9.2 Vue

- `MeResponse` / `UserInfo` 增加 feature map 和 status。
- 菜单、动态路由、直达校验和按钮统一使用相同 helper。
- 新增 `hasFeatureAccess`、`hasAllFeatureAccess`、`hasAnyFeatureAccess`。
- 新增兴趣组自助页、Group / 成员 / 审核页、super-admin Policy 页和 Revision 历史页。
- `/me` 的 roles、feature map 或 status 变化时：重建路由、更新菜单、清理失效 KeepAlive 页面和页面级敏感缓存；当前页面不再可访问时跳转 403 或默认首页。
- 新文案同步中英文 i18n。

迁移窗口内保留：

```text
corp_capabilities  仅供未迁移 Vue 门禁
feature_access     后端权威最终功能结果
corp_rules         继续保留的军团业务规则
```

### 9.3 React

阶段 1 必须同时完成 React parity：

- 扩展 auth 类型、API 映射和 Zustand；
- `RouteAccessMeta`、`RouteAccessGate`、sidebar 与 Vue 使用相同 all/any 语义；
- 迁移路由和动作按钮补齐 Key；
- 使用同一生成的 FeatureKey 类型；
- 建立 Vue / React 权限快照契约测试。

React parity 是替换 Vue 的发布阻断项，不能留到 Audience Policy 上线后再补。

## 10. 缓存与审计方案

- Group、Policy、Feature Access 不写 JWT。
- 第一版 Membership 使用带索引数据库批量查询，只做请求内复用，不跨请求缓存正向成员资格。
- 第一版不缓存 `user:{id}:feature_access`，避免 Group、Policy、军团、主人物和 Role 的扇出失效。
- Redis 不可用时回源 DB；DB 同时不可用时返回 INDETERMINATE，不得使用无限期旧的正向资格。
- 无 Policy 记录是 allow；数据库错误不是无 Policy。

Policy 缓存推荐：

```text
feature:{key}:audience_policy:current
```

值包含：

```text
version
revision_id
policy snapshot
```

更新顺序：

1. 数据库事务提交；
2. 删除或覆盖 Redis current；
3. 发布跨实例失效事件；
4. 各实例清除本地缓存；
5. 本地缓存使用短 TTL 兜底。

第一版也可以不使用本地 Policy 缓存，只使用 Redis + DB fallback。

实施统一 Feature Access 前必须先修复：

- `SyncConfigSuperAdmins()` 旁路 Role 缓存失效；
- corporation policy 只清除当前进程缓存、无 TTL 的问题。

必须失效或重算的事件：

- 加入、批准、拒绝、撤回、退出、撤销、有效期修改；
- Group suspend / resume / archive；
- Policy 更新、启用、停用、回滚；
- 主人物或军团变化；
- corporation policy 更新；
- Role 变化。

审计复用 `AuditService`，分类建议为 `permission`，覆盖：

- Group create/update/suspend/resume/archive；
- 所有 Membership 状态转换；
- Policy create/update/enable/disable/deny_all/rollback；
- 影响模拟和 shadow mismatch 管理操作。

记录 actor、target、before/after、source、expiry、policy revision、request ID、IP、User-Agent。成功审计必须与业务写在同一 service 事务中。

## 11. 分阶段迁移计划

### 阶段 0A：修复现有权限缺陷

- 目标：不引入新抽象，先修复当前已确认的权限漂移。
- 范围：full-access/default-allow、逐路由 OR/AND、系统页 Key 错绑、关键按钮门禁、super-admin Captain / Mentor 前端入口、Role 缓存失效、corporation policy 多实例缓存、现有策略审计。
- DB：无。
- API：保持现有契约。
- 测试：固化第 4 节全部 P0/P1 场景。
- 完成条件：无已知 P0 漂移；关键 P1 有明确修复或阻断。
- 回滚：按单项提交回退，不触碰 Group 或 Policy 数据。

### 阶段 0B：建立权限契约基线

- 目标：建立 Feature Key 单一来源和自动一致性检查。
- 范围：机器可读 Catalog、Go/TS 代码生成、后端/Vue/React 挂载扫描、权限矩阵回归、Vue/React parity 测试框架。
- DB：无。
- 兼容：现有 capability 字符串值不变。
- 完成条件：未注册 Key 构建失败；`backend_only/frontend_only/catalog_only` 可自动报告。
- 回滚：保留生成产物，关闭 CI 阻断但不改变运行时权限。

### 阶段 1：统一 Feature Access

- 目标：`RequireFeatureAccess(key)` 在 AudienceAccess 恒 true 时与现有 `RequireCorpCapability(key)` 等价。
- 范围：Context Loader、纯 Evaluator、Service、Middleware、三态决策、corporation-only `feature_access`、`feature_access_status`、Vue 与 React 同步消费。
- DB：无。
- 兼容：保留 `corp_capabilities` 和 `corp_rules`。
- 测试：66 项逐 Key 比较；覆盖 super-admin、full-access、default allow/deny、Redis/DB 故障和两端前端快照。
- 完成条件：迁移路由的 Role/军团矩阵与切换前一致，React parity 通过。
- 回滚：单路由切回旧 Middleware；前端切回旧字段。

### 阶段 2：Audience Group 基础能力

- 目标：完成 Group、Membership、状态机、加入/申请/审核、有效期、suspend/archive 和审计，不绑定任何功能。
- DB：新增 Group 和 Membership 表及索引，无回填。
- API / 前端：新增自助与管理页面。
- 兼容：全部 Catalog `audience_grant_scope=none`。
- 测试：完整状态机、并发版本冲突、隐藏标签、过期、suspend/archive、事务审计和越权。
- 完成条件：任何 Group 操作都不改变 Role 或 Feature Access。
- 回滚：移除路由/UI，保留惰性数据。

### 阶段 3：Audience Policy

- 目标：实现 Revision、`rules/deny_all`、Block OR/AND、Catalog 授权主体限制和合并计算。
- DB：新增 Policy、Revision、Block、BlockGroup 表。
- API / 前端：super-admin Catalog、Policy 编辑器、Revision 历史与回滚。
- 兼容：不创建 Policy；无策略或 disabled 均 allow。
- 测试：rules、deny_all、AND/OR、suspended/archived Group、过期成员、grant scope、乐观锁、回滚和故障三态。
- 完成条件：引擎可用但生产权限结果默认不变。
- 回滚：停用单 Feature Policy；必要时切回 corporation-only evaluator。

### 阶段 4：Shadow 与 Pilot

- 目标：先观察新决策，不立即扩大接入范围。
- 范围：shadow decision 日志、新旧 mismatch、按 Feature Key 聚合的 403/503、命中率和评估耗时。
- Pilot 选择原则：非管理、非资金、非审批、非删除、非机密、访问量可控、容易人工验证。
- 推荐：优先新增专用 `beta.*` 测试功能；其次选择低风险只读页面。不要直接把 `info.fittings.manage` 作为默认首个 Pilot。
- 完成条件：监控窗口内无未解释 mismatch 和异常拒绝。
- 回滚：停用 Pilot Policy 或恢复 corporation-only 执法。

### 阶段 5：逐功能接入与清理

- 目标：按风险从低到高逐项启用 Audience Policy，最终前端只消费 Feature Access。
- 顺序：低风险只读 → 用户非关键附加功能 → 普通自助写入 → 管理读取 → 管理写入 → 配置/资金/审批/删除。
- 每项范围：后端路由/service、前端路由/菜单/按钮、Catalog grant scope、测试、文档、监控和回滚开关。
- 清理：全部消费者迁移后删除 `meta.corpCapabilities` 和前端直接使用 `/me.corp_capabilities`；保留 `corp_rules` 和内部 corporation policy。
- 回滚：停用单 Feature Policy、切回旧 Middleware，或回退到仍返回旧字段的上一发布。

## 12. 测试计划

### 12.1 当前权限回归

- full-access 和 default allow 前后端一致；
- 商店、工单、军团 NPC 报表的 all/any 语义；
- 四个系统页正确 Key；
- 关键动作按钮门禁；
- super-admin Captain / Mentor 一致性；
- Role 和 corporation policy 跨实例缓存失效。

### 12.2 Role 与 Corporation

- Group 不产生 Role、不绕过 `RequireRole`；
- guest fallback、auto-role、super-admin 同步不受影响；
- 新旧 Corporation evaluator 全 Key 等价；
- full-access 只绕过军团层；
- default mode 和 `corp_rules` 不变。

### 12.3 Group 与 Membership

- assign、join、apply、approve、reject、withdraw、leave、revoke、reapply；
- expiry、suspend、resume、archive；
- 非法状态转换；
- 乐观锁并发冲突；
- admin 越权访问 super-admin-only Group；
- 普通用户猜测 ID、查看他人、加入 managed tag、修改有效期。

### 12.4 Policy

- 无策略、disabled、rules、deny_all；
- 单 Group、块内 AND、块间 OR；
- suspended/archived Group、过期成员；
- `audience_grant_scope` 与 Group 类型/authority 校验；
- version 冲突、Revision 历史和 rollback；
- 数据库错误返回 INDETERMINATE。

### 12.5 性质测试

验证：

```text
增加一个 AND Group 不会扩大访问范围
增加一个 OR Block 不会缩小访问范围
Group suspend/archive 不会新增访问
Membership 过期不会新增访问
disabled Policy 等价于无 Policy
full_access 不影响 AudienceAccess
```

可使用 table-driven 或 property-based 测试。

### 12.6 Repository、审计和故障

- 有效成员条件在 SQL 中执行；
- 批量评估无 N+1；
- 成功/失败审计、before/after、Revision；
- 事务回滚不留下成功事件；
- Redis 不可用时回源；
- DB 错误 fail closed 并返回 503；
- 跨实例 Policy 失效；
- 成员到期不依赖定时任务。

### 12.7 Vue / React 契约

- Feature Key 生成类型；
- all/any 语义；
- 菜单、路由、直达和按钮；
- 动态路由重建和当前页面退出；
- `feature_access_status=unavailable`；
- 隐藏标签不泄露；
- Vue / React 同一快照同一结果。

### 12.8 可观察性

- DENY 记录 reason code，但不向普通用户泄露隐藏 Group 名称；
- INDETERMINATE 产生 503 指标；
- shadow mismatch 可按 Key 查询；
- 评估耗时、Policy 命中率和 Revision 可追踪。

## 13. 风险与回滚方案

| 风险 | 控制 | 回滚 |
| --- | --- | --- |
| Role / Group 混淆 | 独立表、API 和 Service；禁止写 Role | 停用 Group 路由 / Policy |
| 管理员通过成员维护间接扩权 | `audience_grant_scope` 同时校验 Group 类型和 authority | 停用单 Feature Policy |
| 开放组绑定敏感功能 | 默认 `none`；敏感项 `super_admin_managed` | 停用策略并审查成员变更 |
| 空数据意外全量封锁 | 显式 `effect=deny_all`，二次确认 | 切换 disabled 或回滚 Revision |
| 策略历史不可还原 | 不可变 Revision + 审计 | 回滚到已知 Revision |
| 过期 / 撤销后缓存放行 | v1 不跨请求缓存正向 Membership | 禁用可选缓存、回源 DB |
| 前后端 Key 或 all/any 漂移 | 单一 Manifest、代码生成、CI 契约测试 | 后端继续拒绝，回退前端元数据 |
| super-admin / full-access 范围错误 | 前者绕过两层，后者只绕过军团层 | 停用 Audience Policy / 回退 Middleware |
| archived Group 恢复导致权限复活 | archived 终态；暂停使用 suspended | 保持 archived，创建新 Group |
| 无策略时意外封锁 | 全部 grant scope 默认 none，不创建 Policy | 切回 corporation-only evaluator |
| 多实例缓存不一致 | current pointer、通知、短 TTL | 禁用缓存回源 DB |
| `/me` 权限异常阻断自助修复 | `feature_access_status=unavailable` | 保留 JWT-only 页面 |
| React 切换扩大入口 | 阶段 1 parity 作为发布阻断 | 继续服务 Vue |

## 14. 建议实施顺序

1. 修复现有 P0/P1 权限漂移和缓存失效。
2. 建立机器可读 Feature Catalog、Go/TS 生成与 CI 检查。
3. 完成三态 Feature Access、`/me` 状态和 Vue/React parity。
4. 实现独立 Group / Membership，不绑定功能。
5. 实现 Revision Policy、显式 deny_all、事务审计和故障测试。
6. 通过 shadow 模式验证新旧决策。
7. 使用专用 beta 或低风险只读功能作为 Pilot。
8. 按风险顺序逐项接入，最后删除旧前端消费路径。

## 15. 暂不实施的能力

- 动态 Role 或完整 RBAC 重构。
- Discord 式排序、继承和 allow/deny 覆盖。
- 组内管理员、委派层级和自动规则分组。
- 脚本条件、任意 JSON 表达式、deny Group、权重和资源实例 ACL。
- 把新人、导师资格、ESI token、资料、用户或业务状态同步为 Group。
- 把 Group、Policy 或最终 Feature Access 写入 JWT。
- 物理删除 Group / Membership / Policy Revision。
- 对未双端对齐或 grant scope 为 none 的 Key 启用 Audience Policy。
- 第一版跨请求缓存正向 Membership 或最终 Feature Access。

## 未决问题

- `membership_authority` 两档是否足够；第一版保持 `super_admin_only | admin`，不引入组内管理员。
- 首个 Pilot 需要结合实际使用量、敏感度和可接受的 403/503 监控窗口确认。
- `menu.*` 的最终移除范围需要在逐模块迁移时确定，但不得继续新增。

## 实施准入条件

进入 Audience Group 开发前必须完成：

1. 修复当前 P0 权限漂移。
2. 修复 Role 与 corporation policy 缓存失效。
3. 建立 Feature Key 单一来源和类型划分。
4. 明确并实现前端 all/any 语义。
5. Feature Access 使用 ALLOW / DENY / INDETERMINATE。
6. 定义 `/me.feature_access_status`。
7. Policy 使用显式 `deny_all`。
8. Policy 使用不可变 Revision。
9. Catalog 校验间接授权主体。
10. Group 使用 suspended / archived 区分暂停与终态退役。

## 明确声明与升级路径

- 本文是提案，不代表 Audience Group 或统一 Feature Access 已实现。
- 本文不能覆盖 repo rules、architecture、API 或 current feature 文档。
- 实施时必须先输出每个阶段的文件级修改计划，不得从本草案直接一次性修改全部权限链路。
- 落地后，架构与不变量迁移到 `auth-and-permissions.md` / `routing-and-menus.md`，接口迁移到 `route-index.md`，产品行为进入 `docs/features/current/`，表关系同步 `database-schema.md`。
- 转正时删除已完成的迁移叙述，只保留最终行为、理由、不变量和主要代码入口。

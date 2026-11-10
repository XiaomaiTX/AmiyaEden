---
status: active
doc_type: architecture
owner: engineering
last_reviewed: 2026-11-10
source_of_truth:
  - server/bootstrap/db.go
  - server/internal/model
  - server/internal/model/sys_config.go
  - server/internal/service/role.go
  - server/internal/service/eve_sso.go
---

# 数据库 Schema

## 目的

本文件描述当前应用自己的数据库结构边界与关键关系。

它回答的是：

- 当前应用把哪些业务数据持久化到 PostgreSQL
- 用户、职权关联、自动权限映射等核心表如何关联
- 哪些列是当前权威关系的一部分，哪些只是兼容字段

它不试图替代代码中的完整字段定义。
精确字段类型、索引与 GORM tag 仍以 `server/internal/model/*` 和 `server/bootstrap/db.go` 为准。

## 真实来源

当前应用 schema 的真实来源是：

1. `server/bootstrap/db.go` 中注册到 `AutoMigrate` 的模型
2. `server/internal/model/*` 中的 GORM 模型定义
3. `server/bootstrap/db.go` 中的启动期 schema 规范化逻辑

`docs/reference/sde-schema.sql` 只是历史 SDE 参考资产，不代表本应用当前 live schema。

## Schema 生成方式

应用启动初始化数据库时，当前会执行：

- `AutoMigrate`
- 必要的 schema 规范化与兼容处理

因此，当前 schema 不是通过单独的 SQL migration 目录维护，而是通过 GORM 模型 + 启动时补偿逻辑维护。

已被代码移除的历史列或表，也通过 `server/bootstrap/db.go` 里的启动期 cleanup 逻辑条件删除，而不是放在独立 SQL migration 目录中。

## 核心表分组

当前应用表大致分为这些分组：

- 认证与用户：`user`、`eve_character`、`mumble_identity`
- RBAC：`user_role`
- 自动权限映射：`esi_role_mapping`、`esi_title_mapping`、`eve_character_corp_role`
- ESI 快照：资产、通知、技能、合同、装配、结构、钱包等 `eve_*` / `esi_*` 相关表
- 业务模块：`fleet*`、`srp*`、`shop*`、`skill_plan*`、`welfare*`、`captain_refund`、`captain_refund_claim`、`alliance_pap*`、`system_wallet*`、`wallet_log`、`wallet_transaction`
- 基础设施：`operation_log`、`sde_versions`、`sys_config`

## 用户与认证

### `user`

当前产品用户表不承载用户名 / 密码登录模型。

关键列包括：

- `id`
- `nickname`
- `qq`
- `discord_id`
- `status`
- `role`
- `primary_character_id`
- `last_login_at`
- `last_login_ip`
- `created_at` / `updated_at` / `deleted_at`

说明：

- 当前产品认证入口是 EVE SSO，不是账号密码
- `qq` 与 `discord_id` 是当前资料补全与唯一性校验的一部分
- `qq` 在数据库层面对非空值有唯一约束；未填写时使用空值，不参与唯一性判断
- `primary_character_id` 指向用户当前主人物的 EVE `character_id`
- 用户头像 URL 不再持久化，接口与前端都在读取时根据 `primary_character_id` 推导
- `role` 仍然保留，但它不是当前 RBAC 的权威来源

### `eve_character`

`eve_character` 表示绑定到平台用户的 EVE 人物。

### `mumble_identity`

`mumble_identity` 是 Seat 用户到 Mumble 协议身份的单向投影，不是 Mumble 用户副本。它以独立表的主键作为稳定 Mumble User ID（保留值 `0` 永不分配），并以 `seat_user_id` 一对一关联 `user`。表中只保存 Argon2id 凭据哈希、启用状态、版本与使用时间；绝不保存 Mumble 明文密码或 Seat/EVE 凭据。

稳定 Mumble ID 不等于 `user.id`，也不会因凭据轮换、职权或主人物变化而改变。

关键列包括：

- `character_id`
- `character_name`
- `user_id`
- `access_token`
- `refresh_token`
- `token_expiry`
- `scopes`
- `token_invalid`
- `birthday`
- `fuxi_legion_tenure_days`
- `corporation_id`
- `alliance_id`
- `faction_id`

说明：

- `scopes` 是"历史授予并集 ∩ 当前注册 scope 集"的快照，不是当前 access_token 链的实时镜像：SSO 回调按并集合并写入，后台 token 刷新不写该列；管理员专属可选 scope 在归属用户失去管理员职权后的下一次回调中被剥离
- `access_token` / `refresh_token` / `token_expiry` 的写入路径（SSO 回调与后台刷新）共用 per-character 互斥锁，保证轮换式 refresh token 不被旧值覆盖

关系上：

- 一个 `user` 可以绑定多个 `eve_character`
- `user.primary_character_id` 记录主人物的 EVE `character_id`
- 人物头像 URL 不再持久化，接口与前端都在读取时根据 `character_id` 推导
- `fuxi_legion_tenure_days` 是从 `character_corporation_history` 推导出的缓存字段，不应再视为 request-time ESI 读取结果

### `character_corporation_history`

`character_corporation_history` 保存绑定人物的完整 ESI corporation history 快照。

关键列包括：

- `character_id`
- `record_id`
- `corporation_id`
- `is_deleted`
- `start_date`

说明：

- 行集按人物维度由 ESI 刷新队列全量替换，不保留软删除历史
- `record_id` 保留 CCP 返回的原始记录标识，用于避免重复行和支持后续复用
- 该表是 `eve_character.fuxi_legion_tenure_days` 的上游来源，也为后续 employment history 相关功能提供持久化数据

## 职权关联与权限数据

### 当前权威 RBAC 持久化表

当前数据库里与 RBAC 直接相关的持久化表基于：

- `user_role`

这是当前用户职权分配的权威 schema。

当前实现中：

- canonical 系统职权编码定义在代码常量与 `SystemRoleDefinitions` 中，不存储在独立 `role` 表
- 路由级权限边界通过后端中间件 `RequireRole` / `RequireLoginUser` 实现，不依赖数据库权限表
- 前端路由 / 菜单元数据属于代码与前端配置，不属于当前数据库 schema

如需查看 canonical 职权编码与权限边界，见 `docs/architecture/auth-and-permissions.md`。

### `user_role`

`user_role` 是用户和职权的多对多关联表：

- `user_id`
- `role_code`

它是当前用户职权分配的权威来源。

## 兼容字段

### `user.role` 的当前定位

`user.role` 是当前 schema 中保留的兼容字段。

它仍然存在的原因是：

- 兼容旧 JWT / 旧前端消费者
- 在 `user_role` 为空时提供 fallback
- 为仍依赖单职权字段的返回结构提供兼容值

但当前设计上：

- 职权真实分配以 `user_role` 为准
- `user.role` 只是镜像 / fallback / 兼容字段

### 运行时同步行为

用户职权被重新分配时，服务层会把 `user.role` 同步为最高优先级职权，以降低旧消费者漂移风险。

### 文档约束

因此，今后讨论“当前职权 schema”时：

- 应优先说 `user_role`
- 不应把 `user.role` 描述成权威职权来源
- 需要明确标注它是兼容历史单职权模型的保留列

## 自动权限映射相关表

### `esi_role_mapping`

ESI 军团职权到系统职权的映射表。

关键列：

- `esi_role`
- `role_code`

### `esi_title_mapping`

ESI 头衔到系统职权的映射表。

关键列：

- `corporation_id`
- `title_id`
- `title_name`
- `role_code`

### `eve_character_corp_role`

人物当前 ESI 军团职权快照表。

关键列：

- `character_id`
- `corp_role`

说明：

- 这是自动权限同步的输入快照之一
- `eve_character_title` 只用于显式 title mapping

## 新人帮扶相关表

### `captain_refund`

保存申请人、状态、奖励、创建与审批时间、审批人、拒绝原因和钱包业务键。`snapshot` 是提交时冻结的 JSON，包含请求时间段和星系、成员及所属用户、资格判定资料、配置、赏金和匹配证据。审批不修改快照。

### `captain_refund_claim`

以 `(character_id, journal_id)` 作为复合主键，保存占用申请 ID 和流水发生时间。待审核与已批准申请占用流水，拒绝在事务内释放占用，原快照保留。复合主键跨队长和角色身份防重。

统一资格不再使用缓存表。升级删除旧队长绑定、赏金归因、同步状态、资格缓存、结算表，保留钱包账目。详细不变量见 [新人资格与队长退税](../features/current/newbro-support.md)。

### 自动映射输入边界

这是自动映射需要明确保留的 schema 语义：

- `eve_character_corp_role` 承载的是 ESI 返回的真实 corp permission 快照
- `esi_role_mapping` 使用这里的 `corp_role` 配置显式映射
- `esi_title_mapping` 使用 title 数据配置显式映射，不能和 corp role 混为一谈

因此，corp role 与 title 是两条独立的自动映射输入链路。

## 福利与伏羲币关联表

### `welfare`

关键列包括：

- `id`
- `dist_mode`
- `pay_by_fuxi_coin`
- `require_skill_plan`
- `max_char_age_months`
- `minimum_pap`
- `minimum_fuxi_legion_years`
- `require_evidence`
- `example_evidence`
- `status`
- `sort_order`
- `created_by`

说明：

- `pay_by_fuxi_coin` 是福利定义上的可选整数配置，不会在申请时冻结到 `welfare_application`
- 发放时，服务层会重新读取当前 `welfare` 行来决定是否发放伏羲币；若 `0 < pay_by_fuxi_coin < system_config.welfare.auto_approve_fuxi_coin_threshold`，该发放会在申请提交事务内直接发生
- `system_config.welfare.auto_approve_fuxi_coin_threshold` 默认值为 `500`；管理员可在 `/welfare/settings` 调整，设置为 `0` 时关闭自动审批
- 技能计划关联当前通过 `welfare_skill_plans` 关系表维护，不直接内嵌在 `welfare` 表列里
- `minimum_fuxi_legion_years` 要求用户拥有至少一个角色，其在伏羲军团的累计任职时长达到该值；累计时长从 `character_corporation_history` 推导并缓存到 `eve_character.fuxi_legion_tenure_days`

### `welfare_application`

关键列包括：

- `welfare_id`
- `user_id`
- `character_id`
- `character_name`
- `qq`
- `discord_id`
- `evidence_image`
- `status`
- `reviewed_by`
- `reviewed_at`

说明：

- 常规人工审批路径会在事务内重新读取并锁定申请行，再执行 `requested -> delivered / rejected`
- 若当前 `welfare.pay_by_fuxi_coin` 满足 `0 < value < system_config.welfare.auto_approve_fuxi_coin_threshold` 且申请资格校验通过，申请会直接以 `delivered` 状态写入，并同步写入 `reviewed_at`；此时 `reviewed_by = 0`
- 历史导入允许 `user_id` 为空并直接写入 `delivered` 记录

### `wallet_transaction`

与福利模块相关的当前约定：

- 当 `welfare.pay_by_fuxi_coin > 0` 且福利申请被发放为 `delivered` 时，系统会写入一条钱包流水；这既包括人工审批的 `requested -> delivered`，也包括小额伏羲币福利在申请时的自动发放
- 该流水使用 `ref_type = welfare_payout`
- `ref_id` 当前按 `welfare_application:<application_id>` 生成，用于追踪该次福利发放
- 导入历史福利记录不会补写 `welfare_payout` 钱包流水

## `system_config`

`system_config` 仍然是当前应用的全局键值配置表。

与军团建筑管理相关的当前键包括：

- `dashboard.corporation_structures_authorizations`
  - 保存军团到授权 Director 人物的 JSON 映射
  - 这是 Dashboard 军团建筑管理页写入授权关系的持久化来源
- `dashboard.corporation_structures_fuel_notice_threshold_days`
  - 保存燃料提醒阈值，单位为天
  - 默认值为 `7`
  - 值为 `0` 时关闭燃料提醒
- `dashboard.corporation_structures_timer_notice_threshold_days`
  - 保存增强计时提醒阈值，单位为天
  - 默认值为 `7`
  - 值为 `0` 时关闭增强计时提醒
说明：

- 这些配置键由军团建筑设置页读写
- 提醒计数和列表页都依赖它们来决定当前可见范围与徽章数量
- 授权映射必须指向当前可管理军团中的有效 Director 人物

## 当前未采用的 schema 设计

以下都不是当前产品 schema 的有效方向：

- 用户名 / 密码 / 盐值认证表
- 以单个 `user.role` 作为唯一职权来源
- 把历史 `role` / `menu` / `role_menu` 继续当作当前权威 RBAC schema
- 把历史 `docs/reference/sde-schema.sql` 当作应用业务表定义

仓库里可能仍有旧页面、旧文案或历史兼容逻辑，但它们不应被重新解释成当前数据库设计要求。

## 变更规则

当你修改应用 schema 时，通常需要同步这些层：

1. `server/internal/model/*`
2. `server/bootstrap/db.go`
3. 相关 `repository` / `service`
4. 前端 API 与类型文件
5. 对应 feature / API 文档
6. 本文件

如果变更涉及：

- 职权模型
- 用户资料字段
- 前端路由 / 菜单权限边界
- 自动权限映射表

应优先检查 `docs/architecture/auth-and-permissions.md` 与 `docs/features/current/administration.md` 是否也需要更新。

旧 QQ 治理/预警专属模型、索引和自动迁移已移除，新基础设施采用下文独立 qqbot 表。现存旧表不会自动删除；清理范围及恢复要求见 [QQ Bot 数据库清理方案](../guides/qq-bot-retirement-cleanup.md)。共享 `user.qq` 及其唯一索引继续保留。

## QQ 调度基础设施

新独立表 `qqbot_operation`、`qqbot_attempt`、`qqbot_inbox`、`qqbot_delivery`、`qqbot_circuit`、`qqbot_setting` 由启动迁移注册，不复用旧治理表。业务键与消费复合键使用数据库唯一约束；租约 token 条件更新 fencing。操作终态与结果消费记录在同一事务提交。状态和保留语义见 [QQ 调度](../features/current/qqbot.md)。

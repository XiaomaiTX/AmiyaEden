---
status: active
doc_type: api
owner: engineering
last_reviewed: 2026-04-27
source_of_truth:
  - server/internal/router/router.go
---

# API 路由索引

## 说明

本文件只记录当前 `server/internal/router/router.go` 已注册的路由分组、路径与主要权限边界。
权限列说明：

- `JWT`：任意持有有效 JWT 的已认证用户可访问，包含 `guest`
- `Login`：任意已认证且非 `guest` 的产品用户可访问

## Public

### EVE SSO

| Method | Path | 说明 | 权限 |
| --- | --- | --- | --- |
| GET | `/sso/eve/login` | 获取 SSO 登录地址 | Public |
| GET | `/sso/eve/callback` | 处理 SSO 回调 | Public |
| GET | `/sso/eve/scopes` | 获取当前注册 scopes | Public |

### SDE

| Method | Path | 说明 | 权限 |
| --- | --- | --- | --- |
| GET | `/sde/version` | 当前 SDE 版本 | Public |
| POST | `/sde/types` | 批量查询 type 信息 | Public |
| POST | `/sde/names` | 批量查询名称映射 | Public |
| POST | `/sde/search` | 模糊搜索物品 / 成员 | Public |

### 招募链接

| Method | Path | 说明 | 权限 |
| --- | --- | --- | --- |
| POST | `/recruit/:code/submit` | 提交 QQ 号 | Public |

### Fuxi Admin Directory

| Method | Path | 说明 | 权限 |
| --- | --- | --- | --- |
| GET | `/fuxi-admins` | 登录用户目录（配置 + 层级 + 管理员） | Login |

## Authenticated Base

| Method | Path | 说明 | 权限 |
| --- | --- | --- | --- |
| GET | `/sso/eve/characters` | 当前用户绑定人物 | JWT |
| GET | `/sso/eve/bind` | 获取绑定新人物的 SSO 地址 | JWT |
| PUT | `/sso/eve/primary/:character_id` | 设为主人物 | JWT |
| DELETE | `/sso/eve/characters/:character_id` | 解绑人物 | JWT |
| GET | `/me` | 当前用户、人物、职权、绑定人物，并返回 `enforce_character_esi_restriction`；主人物 ESI 已失效时仍返回启动上下文（含 `token_invalid` 状态），由前端决定是否锁定 | JWT |
| PUT | `/me` | 更新当前用户昵称 / QQ / Discord ID | JWT |
| POST | `/dashboard` | Dashboard 聚合数据 | JWT |
| GET | `/dashboard/corporation-structures/settings` | 获取可管理军团列表、每个军团可选 Director 角色与当前授权映射，同时返回全局通知阈值 `fuel_notice_threshold_days` / `timer_notice_threshold_days`（天） | `RequireRole(admin)` |
| PUT | `/dashboard/corporation-structures/settings/authorizations` | 保存军团到 Director 角色授权映射；可同时更新全局通知阈值 `fuel_notice_threshold_days` / `timer_notice_threshold_days`（`>=0`，`0` 表示关闭） | `RequireRole(admin)` |
| GET | `/dashboard/corporation-structures/filter-options` | 获取建筑列表筛选元数据（星系、类型、服务） | `RequireRole(admin)` |
| POST | `/dashboard/corporation-structures/list` | 按多条件筛选读取军团建筑快照分页列表（支持排序） | `RequireRole(admin)` |
| POST | `/dashboard/corporation-structures/run-task` | 使用已授权 Director 角色触发单个军团建筑后台 ESI 任务（异步入队） | `RequireRole(admin)` |
| GET | `/badge-counts` | 导航徽章计数；仅返回当前登录用户可见且非零的计数字段。福利可申请数仅读取内存缓存，不会在该接口内重新计算资格。`super_admin/admin` 额外返回军团建筑提醒计数 `corporation_structures_attention` | Login |
| POST | `/notification/list` | 通知列表 | JWT |
| POST | `/notification/unread-count` | 未读数 | JWT |
| POST | `/notification/read` | 标记已读 | Login |
| POST | `/notification/read-all` | 全部已读 | Login |

## Operation

### Fleets

| Method | Path | 说明 | 权限 |
| --- | --- | --- | --- |
| POST | `/operation/fleets` | 创建舰队 | `RequireRole(admin, fc, senior_fc)` |
| GET | `/operation/fleets` | 舰队列表 | `RequireRole(admin, fc, senior_fc)` |
| GET | `/operation/fleets/me` | 我的舰队 | Login |
| GET | `/operation/fleets/:id` | 舰队详情 | `RequireRole(admin, fc, senior_fc)` |
| PUT | `/operation/fleets/:id` | 更新舰队 | `RequireRole(admin, fc, senior_fc)` |
| DELETE | `/operation/fleets/:id` | 删除舰队 | `RequireRole(admin)` |
| POST | `/operation/fleets/:id/refresh-esi` | 刷新舰队 ESI 数据 | `RequireRole(admin, fc, senior_fc)` |
| GET | `/operation/fleets/:id/members` | 舰队成员 | `RequireRole(admin, fc, senior_fc)` |
| GET | `/operation/fleets/:id/members-pap` | 舰队成员与 PAP | `RequireRole(admin, fc, senior_fc)` |
| POST | `/operation/fleets/:id/members/manual` | 手动添加成员 | `RequireRole(admin, fc, senior_fc)` |
| POST | `/operation/fleets/:id/members/sync` | 同步 ESI 成员 | `RequireRole(admin, fc, senior_fc)` |
| POST | `/operation/fleets/:id/pap` | 发放 PAP | `RequireRole(admin, fc, senior_fc)` |
| GET | `/operation/fleets/:id/pap` | PAP 日志 | `RequireRole(admin, fc, senior_fc)` |
| GET | `/operation/fleets/pap/me` | 我的 PAP 日志 | Login |
| GET | `/operation/fleets/pap/corporation` | 军团 PAP 汇总 | Login |
| GET | `/operation/fleets/pap/alliance` | 我的联盟 PAP | Login |
| POST | `/operation/fleets/:id/invites` | 创建邀请 | `RequireRole(admin, fc, senior_fc)` |
| GET | `/operation/fleets/:id/invites` | 邀请列表 | `RequireRole(admin, fc, senior_fc)` |
| DELETE | `/operation/fleets/invites/:invite_id` | 停用邀请 | `RequireRole(admin, fc, senior_fc)` |
| POST | `/operation/fleets/join` | 加入舰队 | Login |
| GET | `/operation/fleets/esi/:character_id` | 查询人物当前舰队 | Login |
| POST | `/operation/fleets/:id/ping` | 发送 Webhook Ping | `RequireRole(admin, fc, senior_fc)` |

### Fleet Configs

| Method | Path | 说明 | 权限 |
| --- | --- | --- | --- |
| GET | `/operation/fleet-configs` | 配置列表 | Login |
| GET | `/operation/fleet-configs/:id` | 配置详情 | Login |
| GET | `/operation/fleet-configs/:id/eft` | 获取 EFT 文本 | Login |
| POST | `/operation/fleet-configs` | 创建配置 | `RequireRole(admin, senior_fc)` |
| PUT | `/operation/fleet-configs/:id` | 更新配置；同一装配内按 `flag + type_id + quantity` 组合匹配物品，匹配项保留原有设置，其他项重置 | `RequireRole(admin, senior_fc)` |
| DELETE | `/operation/fleet-configs/:id` | 删除配置 | `RequireRole(admin, senior_fc)` |
| POST | `/operation/fleet-configs/import-fitting` | 从人物装配导入 | `RequireRole(admin, senior_fc)` |
| POST | `/operation/fleet-configs/export-esi` | 导出到 ESI | Login |
| GET | `/operation/fleet-configs/:id/fittings/:fitting_id/items` | 装配物品 | Login |
| PUT | `/operation/fleet-configs/:id/fittings/:fitting_id/items/settings` | 更新物品设置 | `RequireRole(admin, senior_fc)` |

## Skill Planning

### Skill Plans

| Method | Path | 说明 | 权限 |
| --- | --- | --- | --- |
| GET | `/skill-planning/skill-plans/check/selection` | 获取当前用户保存的完成度检查人物选择 | Login |
| PUT | `/skill-planning/skill-plans/check/selection` | 保存当前用户的完成度检查人物选择 | Login |
| POST | `/skill-planning/skill-plans/check/run` | 执行技能规划完成度检查 | Login |
| GET | `/skill-planning/skill-plans/check/plan-selection` | 获取当前用户保存的完成度检查规划选择 | Login |
| PUT | `/skill-planning/skill-plans/check/plan-selection` | 保存当前用户的完成度检查规划选择 | Login |
| GET | `/skill-planning/skill-plans` | 技能计划列表 | Login |
| GET | `/skill-planning/skill-plans/:id` | 技能计划详情 | Login |
| POST | `/skill-planning/skill-plans` | 创建技能计划 | `RequireRole(admin, senior_fc)` |
| PUT | `/skill-planning/skill-plans/reorder` | 调整技能计划排序 | `RequireRole(admin, senior_fc)` |
| PUT | `/skill-planning/skill-plans/:id` | 更新技能计划 | `RequireRole(admin, senior_fc)` |
| DELETE | `/skill-planning/skill-plans/:id` | 删除技能计划 | `RequireRole(admin, senior_fc)` |
| GET | `/skill-planning/personal-skill-plans` | 当前用户个人技能计划列表 | Login |
| GET | `/skill-planning/personal-skill-plans/:id` | 当前用户个人技能计划详情 | Login |
| POST | `/skill-planning/personal-skill-plans` | 创建当前用户个人技能计划 | Login |
| PUT | `/skill-planning/personal-skill-plans/reorder` | 调整当前用户个人技能计划排序 | Login |
| PUT | `/skill-planning/personal-skill-plans/:id` | 更新当前用户个人技能计划 | Login |
| DELETE | `/skill-planning/personal-skill-plans/:id` | 删除当前用户个人技能计划 | Login |

## Info

| Method | Path | 说明 | 权限 |
| --- | --- | --- | --- |
| POST | `/info/wallet` | 钱包流水 | Login |
| POST | `/info/skills` | 技能列表 | Login |
| POST | `/info/ships` | 舰船列表 | Login |
| POST | `/info/implants` | 植入体 | Login |
| POST | `/info/assets` | 资产 | Login |
| POST | `/info/contracts` | 合同列表 | Login |
| POST | `/info/contracts/detail` | 合同详情 | Login |
| POST | `/info/esi-refresh` | 手动触发指定角色的技能 ESI 刷新（仅限自己的角色） | Login |
| POST | `/info/fittings` | 装配列表 | Login |
| POST | `/info/fittings/save` | 保存装配 | Login |
| POST | `/info/npc-kills` | 个人 NPC 刷怪报表 | Login |
| POST | `/info/npc-kills/all` | 全部 NPC 刷怪报表 | Login |

## Shop

### Common User Side

| Method | Path | 说明 | 权限 |
| --- | --- | --- | --- |
| POST | `/shop/wallet/my` | 我的钱包 | Login |
| POST | `/shop/wallet/my/transactions` | 我的钱包流水 | Login |
| POST | `/shop/products` | 商品列表 | Login |
| POST | `/shop/product/detail` | 商品详情 | Login |
| POST | `/shop/buy` | 购买商品 | Login |
| POST | `/shop/orders` | 我的订单 | Login |

## Hall of Fame

### Temple

| Method | Path | 说明 | 权限 |
| --- | --- | --- | --- |
| GET | `/hall-of-fame/temple` | 获取圣殿画布配置与全部可见英雄卡片 | Login |

## Newbro Support

### Newbro User Side

| Method | Path | 说明 | 权限 |
| --- | --- | --- | --- |
| GET | `/newbro/captains` | 当前新人可选择的队长列表 | `Login` + 当前新人资格 |
| GET | `/newbro/affiliation/me` | 当前用户的新人资格快照与队长关联历史 | Login |
| GET | `/newbro/affiliations/history` | 当前用户的帮扶关系变更历史 | Login |
| POST | `/newbro/affiliation/select` | 选择或切换队长 | `Login` + 当前新人资格 |

| POST | `/newbro/affiliation/end` | 结束当前与队长的帮扶关系 | `Login` + 当前新人资格 |

### Recruit Link User Side

| Method | Path | 说明 | 权限 |
| --- | --- | --- | --- |
| POST | `/newbro/recruit/link` | 生成招募链接 | Login |
| GET | `/newbro/recruit/links` | 获取我的招募链接列表 | Login |
| GET | `/newbro/recruit/direct-referral` | 获取当前用户是否可补录推荐人 | Login |
| POST | `/newbro/recruit/direct-referral/check` | 按 QQ 检查直接推荐人 | Login |
| POST | `/newbro/recruit/direct-referral/confirm` | 按推荐人 `user_id` 确认直接推荐并立即发放招募奖励 | Login |

### Captain Side

| Method | Path | 说明 | 权限 |
| --- | --- | --- | --- |
| GET | `/newbro/captain/overview` | 当前队长概览 | `RequireRole(captain)` |
| GET | `/newbro/captain/players` | 当前队长名下新人列表 | `RequireRole(captain)` |
| GET | `/newbro/captain/attributions` | 当前队长赏金归因明细 | `RequireRole(captain)` |
| GET | `/newbro/captain/rewards` | 当前队长奖励发放历史 | `RequireRole(captain)` |
| GET | `/newbro/captain/eligible-players` | 可加入帮扶的新人列表 | `RequireRole(captain)` |
| POST | `/newbro/captain/enroll` | 手动加入新人到帮扶 | `RequireRole(captain)` |
| POST | `/newbro/captain/affiliation/end` | 解除指定新人与当前队长的关系 | `RequireRole(captain)` |

### Recruit Link Admin Side

| Method | Path | 说明 | 权限 |
| --- | --- | --- | --- |
| GET | `/system/newbro/recruit/links` | 分页获取所有用户招募链接 | `RequireRole(admin)` |

## Mentor System

### Mentee Side

| Method | Path | 说明 | 权限 |
| --- | --- | --- | --- |
| GET | `/mentor/mentors` | 当前符合资格学员可申请的导师列表 | `Login` + 当前学员资格（服务层） |
| GET | `/mentor/me` | 当前用户的学员资格与导师关系快照 | Login |
| POST | `/mentor/apply` | 向指定导师提交申请 | `Login` + 当前学员资格（服务层） |

### Mentor Side

| Method | Path | 说明 | 权限 |
| --- | --- | --- | --- |
| GET | `/mentor/dashboard/applications` | 当前导师待处理申请列表 | `RequireRole(mentor)` |
| GET | `/mentor/dashboard/mentees` | 当前导师的学员列表 | `RequireRole(mentor)` |
| GET | `/mentor/dashboard/reward-stages` | 当前导师可见的只读奖励阶段配置 | `RequireRole(mentor)` |
| POST | `/mentor/dashboard/accept` | 接受学员申请 | `RequireRole(mentor)` |
| POST | `/mentor/dashboard/reject` | 拒绝学员申请 | `RequireRole(mentor)` |

## Welfare

### Welfare User Side

| Method | Path | 说明 | 权限 |
| --- | --- | --- | --- |
| POST | `/welfare/eligible` | 可申请福利列表 | Login |
| POST | `/welfare/apply` | 申请福利；若 `0 < pay_by_fuxi_coin < 当前自动审批阈值` 且资格校验通过，申请会直接自动发放并同步写入 `welfare_payout` 钱包流水。该阈值默认 `500`，由管理员在 `/welfare/settings` 配置，设为 `0` 时关闭自动审批 | Login |
| POST | `/welfare/my-applications` | 我的福利申请 | Login |
| POST | `/welfare/upload-evidence` | 上传福利申请凭证 | Login |

## Ticket

### Ticket User Side

| Method | Path | 说明 | 权限 |
| --- | --- | --- | --- |
| POST | `/ticket/tickets` | 提交工单 | Login |
| GET | `/ticket/tickets/me` | 我的工单列表（分页，支持 `status`） | Login |
| GET | `/ticket/tickets/:id` | 我的工单详情 | Login |
| POST | `/ticket/tickets/:id/replies` | 我的工单新增回复 | Login |
| GET | `/ticket/tickets/:id/replies` | 我的工单回复列表 | Login |
| GET | `/ticket/categories` | 可用工单分类 | Login |

## Upload

| Method | Path | 说明 | 权限 |
| --- | --- | --- | --- |
| POST | `/upload/image` | 上传图片，返回 base64 data URL（不落盘，最大 2MB，仅支持 jpeg/png/webp） | Login |

## SRP

| Method | Path | 说明 | 权限 |
| --- | --- | --- | --- |
| GET | `/srp/prices` | 价格表 | Login |
| POST | `/srp/prices` | 新增或更新价格 | `RequireRole(admin, senior_fc)` |
| DELETE | `/srp/prices/:id` | 删除价格 | `RequireRole(admin, senior_fc)` |
| POST | `/srp/applications` | 提交补损申请 | Login |
| GET | `/srp/applications/me` | 我的补损申请 | Login |
| GET | `/srp/killmails/me` | 我的 KM；支持可选查询参数 `character_id`、`limit`、`exclude_submitted` | Login |
| GET | `/srp/killmails/fleet/:fleet_id` | 指定舰队 KM；支持可选查询参数 `limit`、`exclude_submitted` | Login |
| POST | `/srp/killmails/detail` | KM 详情 | Login |
| POST | `/srp/open-info-window` | 打开游戏内信息窗口 | Login |
| GET | `/srp/config` | 获取 SRP 配置 | `RequireRole(admin)` |
| PUT | `/srp/config` | 更新 SRP 配置 | `RequireRole(admin)` |
| GET | `/srp/applications` | 审核列表 | `RequireRole(srp, senior_fc, admin)` |
| GET | `/srp/applications/:id` | 审核详情 | `RequireRole(srp, senior_fc, admin)` |
| PUT | `/srp/applications/:id/review` | 审核申请 | `RequireRole(srp, senior_fc, admin)` |
| PUT | `/srp/applications/auto-approve` | 对指定 `fleet_id` 自动审批符合规则的待审批申请 | `RequireRole(srp, senior_fc, admin)` |
| GET | `/srp/applications/batch-payout-summary` | 批量发放汇总 | `RequireRole(srp, senior_fc, admin)` |
| PUT | `/srp/applications/fuxi-payout` | 将全部已批准未发放的申请按 1,000,000 ISK : 1 伏羲币批量发放并结案 | `RequireRole(srp, senior_fc, admin)` |
| PUT | `/srp/applications/:id/payout` | 发放补损 | `RequireRole(srp, senior_fc, admin)` |
| PUT | `/srp/applications/users/:user_id/payout` | 按用户批量发放补损 | `RequireRole(srp, senior_fc, admin)` |

## Tasks

| Method | Path | 说明 | 权限 |
| --- | --- | --- | --- |
| GET | `/tasks` | 任务列表；返回任务分类、类型、当前调度、默认调度、最近一次执行与是否支持手动触发 | `RequireRole(admin)` |
| GET | `/tasks/history` | 执行历史分页；支持按 `task_name` 与 `status` 过滤 | `RequireRole(admin)` |
| POST | `/tasks/:name/run` | 手动触发通用任务；同名任务正在运行时返回 `409 Conflict` | `RequireRole(admin)` |
| PUT | `/tasks/:name/schedule` | 更新周期任务调度（6 段 cron 或 `@every` 描述符），并立即重载运行时调度 | `RequireRole(super_admin)` |

### ESI Task Operations

| Method | Path | 说明 | 权限 |
| --- | --- | --- | --- |
| GET | `/tasks/esi/tasks` | ESI 队列任务定义列表 | `RequireRole(admin)` |
| GET | `/tasks/esi/statuses` | ESI 队列人物状态汇总 | `RequireRole(admin)` |
| POST | `/tasks/esi/run` | 对指定人物执行单个 ESI 任务 | `RequireRole(admin)` |
| POST | `/tasks/esi/run-task` | 对全部人物执行指定 ESI 任务 | `RequireRole(admin)` |
| POST | `/tasks/esi/run-all` | 对全部人物执行 ESI 全量刷新 | `RequireRole(admin)` |

## System

所有 `/system/*` 路由默认要求 `RequireRole(admin)`。

### Basic Config

| Method | Path | 说明 | 权限 |
| --- | --- | --- | --- |
| GET | `/system/basic-config` | 获取固定系统标识（军团 ID / 网站标题） | `RequireRole(super_admin)` |
| GET | `/system/basic-config/allow-corporations` | 获取允许军团列表 | `RequireRole(super_admin)` |
| PUT | `/system/basic-config/allow-corporations` | 更新允许军团列表 | `RequireRole(super_admin)` |
| GET | `/system/basic-config/character-esi-restriction` | 获取任一绑定人物 ESI 失效时是否强制停留人物页的配置 | `RequireRole(super_admin)` |
| PUT | `/system/basic-config/character-esi-restriction` | 更新任一绑定人物 ESI 失效时是否强制停留人物页的配置 | `RequireRole(super_admin)` |

### Welfare Config

| Method | Path | 说明 | 权限 |
| --- | --- | --- | --- |
| GET | `/system/welfare/settings` | 获取福利自动审批伏羲币阈值配置 | `RequireRole(admin)` |
| PUT | `/system/welfare/settings` | 更新福利自动审批伏羲币阈值配置；`0` 表示关闭自动审批 | `RequireRole(admin)` |

### SDE Config

| Method | Path | 说明 | 权限 |
| --- | --- | --- | --- |
| GET | `/system/sde-config` | 获取 SDE 配置 | `RequireRole(super_admin)` |
| PUT | `/system/sde-config` | 更新 SDE 配置 | `RequireRole(super_admin)` |

### NPC Kills / Alliance PAP

| Method | Path | 说明 | 权限 |
| --- | --- | --- | --- |
| POST | `/system/npc-kills` | 公司级 NPC 刷怪报表 | `RequireRole(admin)` |
| GET | `/system/pap` | 联盟 PAP 列表 | `RequireRole(admin)` |
| POST | `/system/pap/fetch` | 手动抓取联盟 PAP | `RequireRole(admin)` |
| POST | `/system/pap/import` | 导入联盟 PAP | `RequireRole(admin)` |
| POST | `/system/pap/settle` | 月度归档 | `RequireRole(admin)` |

### PAP 兑换汇率

| Method | Path | 说明 | 权限 |
| --- | --- | --- | --- |
| GET | `/system/pap-exchange/rates` | 获取 PAP 类型兑换汇率列表 | `RequireRole(admin)` |
| PUT | `/system/pap-exchange/rates` | 更新 PAP 类型兑换汇率 | `RequireRole(admin)` |

### Mentor Admin

| Method | Path | 说明 | 权限 |
| --- | --- | --- | --- |
| GET | `/system/mentor/relationships` | 全量导师关系列表 | `RequireRole(admin)` |
| GET | `/system/mentor/reward-distributions` | 导师奖励发放记录分页列表；关键字支持导师人物名 / 昵称 | `RequireRole(admin)` |
| POST | `/system/mentor/revoke` | 撤销指定导师关系 | `RequireRole(admin)` |
| GET | `/system/mentor/settings` | 导师学员资格阈值配置 | `RequireRole(admin)` |
| PUT | `/system/mentor/settings` | 更新导师学员资格阈值配置 | `RequireRole(admin)` |
| GET | `/system/mentor/reward-stages` | 导师奖励阶段配置 | `RequireRole(admin)` |
| PUT | `/system/mentor/reward-stages` | 更新导师奖励阶段配置 | `RequireRole(admin)` |
| POST | `/system/mentor/reward/process` | 手动执行导师奖励处理 | `RequireRole(admin)` |

### Role / User

| Method | Path | 说明 | 权限 |
| --- | --- | --- | --- |
| GET | `/system/role/definitions` | 系统职权定义列表（只读） | `RequireRole(admin)` |
| GET | `/system/user` | 用户列表；默认按 `last_login_at` 倒序，关键字支持昵称 / QQ / 已绑定人物名；职权字段仅返回有序 `roles[]`，不再返回历史单值 `role`，并附带已绑定人物与每个人物的 `total_sp`、`token_invalid` 快照 | `RequireRole(admin)` |
| GET | `/system/user/:id` | 用户详情 | `RequireRole(admin)` |
| PUT | `/system/user/:id` | 更新用户昵称 / 状态；当操作者为 `super_admin` 且目标不是 `super_admin` 时，还可更新 QQ / Discord ID；`admin` 也不可编辑其他 `admin` | `RequireRole(admin)` |
| DELETE | `/system/user/:id` | 删除用户；`super_admin` 用户不可删除；`admin` 不可删除其他 `admin`，且已登记 QQ / Discord ID 的用户仅 `super_admin` 可删除 | `RequireRole(admin)` |
| GET | `/system/user/:id/roles` | 获取用户职权 | `RequireRole(admin)` |
| PUT | `/system/user/:id/roles` | 设置用户职权；`super_admin` 职权不可通过 API 分配或修改（仅通过配置文件管理）；仅 `super_admin` 可分配 `admin` | `RequireRole(admin)` |
| POST | `/system/user/:id/impersonate` | 模拟登录，需 `super_admin`；若目标用户主人物 ESI 已失效则拒绝签发 token | `RequireRole(admin)` + `super_admin` |

### Fuxi Coin

| Method | Path | 说明 | 权限 |
| --- | --- | --- | --- |
| POST | `/system/wallet/list` | 钱包列表 | `RequireRole(admin)` |
| POST | `/system/wallet/detail` | 钱包详情 | `RequireRole(admin)` |
| POST | `/system/wallet/adjust` | 调整余额 | `RequireRole(admin)` |
| POST | `/system/wallet/transactions` | 钱包流水 | `RequireRole(admin)` |
| POST | `/system/wallet/logs` | 调整日志 | `RequireRole(admin)` |

### Welfare Admin

| Method | Path | 说明 | 权限 |
| --- | --- | --- | --- |
| POST | `/system/welfare/list` | 福利列表 | `RequireRole(admin, welfare)` |
| POST | `/system/welfare/add` | 创建福利 | `RequireRole(admin)` |
| POST | `/system/welfare/edit` | 编辑福利 | `RequireRole(admin)` |
| POST | `/system/welfare/delete` | 删除福利 | `RequireRole(admin)` |
| POST | `/system/welfare/reorder` | 调整福利排序 | `RequireRole(admin)` |
| POST | `/system/welfare/import` | 导入历史福利记录 | `RequireRole(admin)` |
| POST | `/system/welfare/applications` | 福利申请列表（审批端） | `RequireRole(admin, welfare)` |
| POST | `/system/welfare/applications/delete` | 删除单条福利申请记录 | `RequireRole(admin)` |
| POST | `/system/welfare/review` | 审批仍处于 `requested` 的福利申请（发放/拒绝；若当前福利配置 `pay_by_fuxi_coin > 0`，同步写入 `welfare_payout` 钱包流水；发放成功后尽力发送一封以发放福利官主人物名义发出的双语游戏内邮件，失败不回滚） | `RequireRole(admin, welfare)` |

### Ticket Admin

| Method | Path | 说明 | 权限 |
| --- | --- | --- | --- |
| GET | `/system/ticket/tickets` | 工单管理列表（分页，支持 `status`、`category_id`、`user_id`、`keyword`） | `RequireRole(admin)` |
| GET | `/system/ticket/tickets/:id` | 工单详情 | `RequireRole(admin)` |
| PUT | `/system/ticket/tickets/:id/status` | 更新工单状态 | `RequireRole(admin)` |
| PUT | `/system/ticket/tickets/:id/priority` | 更新工单优先级 | `RequireRole(admin)` |
| POST | `/system/ticket/tickets/:id/replies` | 管理员回复（支持内部备注） | `RequireRole(admin)` |
| GET | `/system/ticket/tickets/:id/replies` | 管理员查看回复（含内部备注） | `RequireRole(admin)` |
| GET | `/system/ticket/tickets/:id/status-history` | 状态变更历史 | `RequireRole(admin)` |
| GET | `/system/ticket/categories` | 分类列表（含禁用） | `RequireRole(admin)` |
| POST | `/system/ticket/categories` | 创建分类 | `RequireRole(admin)` |
| PUT | `/system/ticket/categories/:id` | 更新分类 | `RequireRole(admin)` |
| DELETE | `/system/ticket/categories/:id` | 删除分类 | `RequireRole(admin)` |
| GET | `/system/ticket/statistics` | 工单统计（总量、状态、分类、近 7/30 天） | `RequireRole(admin)` |

### Hall of Fame Admin

| Method | Path | 说明 | 权限 |
| --- | --- | --- | --- |
| GET | `/system/hall-of-fame/config` | 获取名人堂画布配置（单例） | `RequireRole(admin)` |
| PUT | `/system/hall-of-fame/config` | 更新名人堂画布尺寸与背景图 | `RequireRole(admin)` |
| POST | `/system/hall-of-fame/upload-background` | 上传名人堂背景图（base64 data URL，最大 5MB，仅支持 jpeg/png/webp） | `RequireRole(admin)` |
| GET | `/system/hall-of-fame/cards` | 获取全部英雄卡片（含隐藏卡片） | `RequireRole(admin)` |
| POST | `/system/hall-of-fame/cards` | 创建英雄卡片 | `RequireRole(admin)` |
| PUT | `/system/hall-of-fame/cards/batch-layout` | 批量保存卡片坐标、尺寸与层级 | `RequireRole(admin)` |
| PUT | `/system/hall-of-fame/cards/:id` | 更新单张英雄卡片内容、样式或显示状态 | `RequireRole(admin)` |
| DELETE | `/system/hall-of-fame/cards/:id` | 删除英雄卡片 | `RequireRole(admin)` |

### Fuxi Admin Directory Admin

| Method | Path | 说明 | 权限 |
| --- | --- | --- | --- |
| GET | `/system/fuxi-admins/manage-directory` | 获取管理端名录（含带队次数、福利发放次数与历史偏移） | `RequireRole(admin)` |
| GET | `/system/fuxi-admins/config` | 获取伏羲管理名录配置（单例） | `RequireRole(admin)` |
| PUT | `/system/fuxi-admins/config` | 更新字体大小配置 | `RequireRole(admin)` |
| GET | `/system/fuxi-admins/tiers` | 获取层级列表 | `RequireRole(admin)` |
| POST | `/system/fuxi-admins/tiers` | 创建新层级 | `RequireRole(admin)` |
| PUT | `/system/fuxi-admins/tiers/:id` | 更新层级名称 | `RequireRole(admin)` |
| DELETE | `/system/fuxi-admins/tiers/:id` | 删除层级（级联删除管理员） | `RequireRole(admin)` |
| POST | `/system/fuxi-admins` | 创建管理员；响应包含管理端统计字段 | `RequireRole(admin)` |
| PUT | `/system/fuxi-admins/:id` | 更新管理员；`welfare_delivery_offset` 仅 `super_admin` 可改，响应包含管理端统计字段 | `RequireRole(admin)` |
| DELETE | `/system/fuxi-admins/:id` | 删除管理员 | `RequireRole(admin)` |

### Newbro Admin

| Method | Path | 说明 | 权限 |
| --- | --- | --- | --- |
| GET | `/system/newbro/support-settings` | 获取帮扶设置（资格阈值、刷新间隔、奖励比例） | `RequireRole(admin)` |
| PUT | `/system/newbro/support-settings` | 更新帮扶设置（资格阈值、刷新间隔、奖励比例） | `RequireRole(admin)` |
| GET | `/system/newbro/recruit-settings` | 获取招募链接设置（QQ 邀请链接、奖励金额、冷却天数） | `RequireRole(admin)` |
| PUT | `/system/newbro/recruit-settings` | 更新招募链接设置（QQ 邀请链接、奖励金额、冷却天数） | `RequireRole(admin)` |
| GET | `/system/newbro/captains` | 队长绩效列表 | `RequireRole(admin)` |
| GET | `/system/newbro/captains/:user_id` | 队长详情（概览、关联玩家、归因明细） | `RequireRole(admin)` |
| GET | `/system/newbro/affiliations/history` | 新人帮扶关系变更历史 | `RequireRole(admin, captain)` |
| GET | `/system/newbro/rewards` | 队长奖励发放历史 | `RequireRole(admin, captain)` |

### Shop Admin

| Method | Path | 说明 | 权限 |
| --- | --- | --- | --- |
| POST | `/system/shop/product/list` | 商品列表 | `RequireRole(admin)` |
| POST | `/system/shop/product/add` | 新增商品 | `RequireRole(admin)` |
| POST | `/system/shop/product/edit` | 编辑商品 | `RequireRole(admin)` |
| POST | `/system/shop/product/delete` | 删除商品 | `RequireRole(admin)` |
| POST | `/system/shop/order/list` | 订单列表 | `RequireRole(admin, shop_order_manage)` |
| POST | `/system/shop/order/deliver` | 发放订单（成功后尽力发送一封以执行发放官员主人物名义发出的双语游戏内邮件，失败不回滚） | `RequireRole(admin, shop_order_manage)` |
| POST | `/system/shop/order/reject` | 驳回订单 | `RequireRole(admin, shop_order_manage)` |

### Auto Role / Webhook

| Method | Path | 说明 | 权限 |
| --- | --- | --- | --- |
| GET | `/system/auto-role/esi-roles` | ESI corp roles 列表 | `RequireRole(super_admin)` |
| GET | `/system/auto-role/esi-role-mappings` | ESI role 映射列表 | `RequireRole(super_admin)` |
| POST | `/system/auto-role/esi-role-mappings` | 新增 ESI role 映射 | `RequireRole(super_admin)` |
| DELETE | `/system/auto-role/esi-role-mappings/:id` | 删除 ESI role 映射 | `RequireRole(super_admin)` |
| GET | `/system/auto-role/corp-titles` | Corp titles 列表（含军团名称） | `RequireRole(super_admin)` |
| GET | `/system/auto-role/esi-title-mappings` | Title 映射列表 | `RequireRole(super_admin)` |
| POST | `/system/auto-role/esi-title-mappings` | 新增 title 映射 | `RequireRole(super_admin)` |
| DELETE | `/system/auto-role/esi-title-mappings/:id` | 删除 title 映射 | `RequireRole(super_admin)` |
| POST | `/system/auto-role/sync` | 手动触发同步 | `RequireRole(super_admin)` |
| GET | `/system/webhook/config` | 获取 Webhook 配置 | `RequireRole(super_admin)` |
| PUT | `/system/webhook/config` | 保存 Webhook 配置 | `RequireRole(super_admin)` |
| POST | `/system/webhook/test` | 测试 Webhook | `RequireRole(super_admin)` |

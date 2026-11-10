---
status: active
doc_type: feature
owner: engineering
last_reviewed: 2026-11-10
source_of_truth:
  - server/internal/service/captain_refund.go
  - server/internal/service/newbro_eligibility.go
  - server/internal/repository/captain_refund.go
  - server/internal/model/captain_refund.go
  - server/internal/router/router.go
---

# 新人资格与队长退税

## 统一资格与设置

管理员在 `系统管理 / 新人资格与队长退税` 配置用户全部绑定角色 SP 总和上限（默认 1000 万）、最老角色年龄上限（默认 90 天）、退税税率（默认 30%）和允许退税的星系 ID。配置以 `newbro.refund_settings` 存储，星系列表默认空。零是有效上限或税率，不能回退为默认值。

所有角色 SP 总和及最老角色年龄均不超过上限才是新人，边界包含。年龄使用 EVE 角色生日，而非本站注册时间。无角色、缺少任一角色技能或生日数据、生日在未来时不能认定为新人。导师入口与申请和队长退税共用该服务，不缓存永久毕业状态。规则变更不会撤销已有导师关系。

## 申请与核算

- 真实 `captain` 职权用户在 `新人帮扶 / 队长退税申请` 选择自己绑定的队长角色，填入多个队员游戏角色昵称，选择一个允许的星系及 EVE UTC 时间段。
- 昵称去除首尾空格、忽略大小写精确匹配。去重、排除队长角色；未注册昵称忽略并明确展示。最多 250 个输入昵称。
- 时间段采用 `[start, end)`，限最近 30 天，不得包含未来时间。接口使用带时区的 RFC3339 时间；界面输入为 UTC。
- 使用本地同步的正数 `bounty_prizes`，且流水上下文必须为目标太阳系。队长自己的赏金直接计入；队员流水须匹配所选队长角色同星系、同类型、五分钟内的流水，双方都必须在申请时间段内。
- 候选证据按时间差、日期、流水 ID 排序。一条队长证据可对应多个队员，每条计奖流水只累计一次。
- 至少一名非申请人所属用户的新人队员必须有有效匹配赏金；申请人的其他角色不能提供新人名额。
- 退税为 `赏金合计 × (tax_rate / 100) / 1,000,000`，四舍五入保留两位小数。固定兑换常量为每币 100 万 ISK，不还原军团税前金额。舍入后为零不能提交。
- 预览显示成员、资格依据、忽略昵称、逐人赏金、匹配证据、冲突、税率、总额及奖励。钱包余额更新时间仅作同步参考，不保证钱包流水完整；核算不主动刷新 ESI。

## 防重复、快照与审批

`captain_refund` 存申请与不可变 JSON 快照，包含成员所属用户、资格依据及角色资料、规则、计奖流水和匹配证据。审批字段独立更新，配置变化不会改写历史申请。

`captain_refund_claim` 以 `(character_id, journal_id)` 为复合主键，跨申请人和队伍身份占用计奖流水。待审核和已批准申请持续占用，拒绝后删除占用；仅作证据的流水不额外占用。任何计奖流水冲突都会整单阻止提交，预览仅公开冲突角色、时间、流水和申请编号，不开放他人申请详情。

提交时后端重新计算，在同一事务写入申请、快照、占用和审计。数据库唯一约束是并发防重的最终保证。

状态仅 `pending`、`approved`、`rejected`，不支持修改或撤回。队长只能看自己的申请；管理员查看全量并批准或拒绝，拒绝必须填写原因。批准时锁定申请并使用条件状态转换，在同一事务内发奖与记录审计，钱包业务键固定 `captain_refund:{id}`，类型为 `captain_refund`。重复审批被拒绝，任何发放失败使整个审批回滚。

新系统不使用队长队员绑定、归因作业或结算任务。数据库升级清理旧绑定、归因、同步状态、资格缓存和结算表、旧配置及两个队长任务的调度与执行记录；保留全部钱包余额和历史流水类型 `newbro_captain_reward`。导师奖励任务、招募任务及通用 ESI 同步不受影响。

## 接口与实现

队长接口位于 `/api/v1/newbro/captain/refunds`：`GET /options`、`POST /preview`、`POST /`、`GET /`、`GET /:id`。请求字段为 `captain_character_id`、`member_names`、`system_id`、`start`、`end`；提交不接受前端金额。

管理员接口位于 `/api/v1/system/newbro/refunds`：列表、详情、`POST /:id/approve`、`POST /:id/reject`（请求 `reason`）。系统设置为 `GET/PUT /api/v1/system/config/newbro-refund`，字段 `max_total_sp`、`max_age_days`、`tax_rate`、`allowed_system_ids`。

Vue 与 React 均提供申请预览、分页历史、详情、审批和独立系统设置入口；旧接口与页面彻底移除，不提供兼容路由。

## 招募链接

### 功能概述

- 每位已登录且非 `guest` 的用户都可在 `新人帮扶 -> 招新链接` 页面生成专属招募链接
- 链接 `code` 使用 `newbro_recruitment.id` 的 base62 编码，前端公开落地页路径为 `/#/r/:code`
- 用户重新生成链接前必须等待配置的冷却天数，默认值为 `90` 天
- 注册时间在 `7` 天内、具备真实非 `guest` 职权、且尚未通过招募链接或直接推荐形成有效招募奖励的用户，可在 `Dashboard -> 人物管理` 的“联系方式与昵称”卡片下补录推荐人 QQ；若本人资料里尚未保存 QQ，卡片会先提示补齐后再允许确认
- 直接推荐会先按推荐人 QQ 检查系统内有效用户；确认时提交候选人的 `user_id`，通过后直接创建一条 `source = direct_referral` 的招募记录，并按招募链接成功招募的同等金额立即发放奖励
- 直接推荐生成的记录仅用于展示和奖励归档，不参与“重新生成招募链接”的冷却判断
- 公开访客访问链接后，可在无需登录的落地页提交 QQ 号码；提交成功后前端展示加入 QQ 群按钮，跳转目标由管理员配置
- 系统每天凌晨 `2` 点执行 `recruit_link_check` 任务，批量检查所有 `ongoing` 状态的招募条目：
  - 若 QQ 号已对应系统内一个在提交时间之后创建的用户，则该条目标记为 `valid`
  - 若 QQ 号对应的用户创建时间早于提交时间，则该条目标记为 `stalled`
  - 若条目在冷却期内始终未匹配到用户，且提交时间已超过冷却天数，则该条目标记为 `stalled`
- 有效招募会发放伏羲币奖励，钱包流水 `ref_type = recruit_link_reward`
- 管理员可在 `新人帮扶 -> 招新链接 -> 链接设置` 配置 QQ 群邀请链接、每次有效招募奖励金额与冷却天数

### 入口

用户侧：

- `POST /api/v1/newbro/recruit/link` — 生成招募链接
- `GET /api/v1/newbro/recruit/links` — 获取当前用户全部招募链接及其招募条目
- `GET /api/v1/newbro/recruit/direct-referral` — 获取当前用户是否可补录推荐人
- `POST /api/v1/newbro/recruit/direct-referral/check` — 按 QQ 检查推荐人是否有效
- `POST /api/v1/newbro/recruit/direct-referral/confirm` — 按推荐人 `user_id` 确认直接推荐并立即发奖

公开（无需登录）：

- `POST /api/v1/recruit/:code/submit` — 提交 QQ 号码

管理侧：

- `GET /api/v1/system/newbro/recruit/links` — 分页获取全部用户的招募链接
- `GET /api/v1/system/newbro/recruit-settings`
- `PUT /api/v1/system/newbro/recruit-settings`

### 前端页面

- `static/src/views/newbro/recruit-link` — 用户侧招新链接页；管理员会额外看到“全部链接”与“链接设置”tab
- `static/src/views/dashboard/characters` — 联系方式资料卡下的直接推荐补录入口
- `static/src/views/auth/recruit` — 公开落地页

### 关键不变量

- 单个用户每次重新生成链接都会产生新记录，但历史链接的招募条目仍会继续参与状态检查
- 同一条招募链接下 `(recruitment_id, qq)` 只允许存在一条提交记录，重复提交按幂等成功处理
- 招募奖励通过 `wallet_ref_id = recruit_matched_user:{matched_user_id}` 保证按被招募人幂等，不会因重复提交相同 QQ 而重复发放
- 直接推荐与招募链接共用同一套奖励幂等键；同一被招募用户只能成功结算一次
- 直接推荐记录必须标记 `source = direct_referral`，便于在招募记录中与普通招募链接区分
- 公开招募落地页不依赖登录态

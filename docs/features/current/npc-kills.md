---
status: active
doc_type: feature
owner: engineering
last_reviewed: 2026-04-02
source_of_truth:
  - server/internal/service/npc_kill.go
  - server/internal/repository/npc_kill.go
  - server/internal/handler/npc_kill.go
  - static/src/api/npc-kill.ts
  - static/src/views/info/npc-kills
  - static/src/views/dashboard/npc-kills
---

# NPC 刷怪报表

## 功能概述

NPC 刷怪报表展示人物通过 PvE 活动获得的 NPC 来源收入，涵盖以下来源：

| 收入来源 | wallet journal ref_type | 说明 |
| --- | --- | --- |
| 标准刷怪悬赏 | `bounty_prizes` | 星域/星系 NPC 悬赏奖励，含可从 `reason` 解析 NPC 明细的 PvE 收入 |
| 萨沙入侵奖励 | `corporate_reward_payout` | 入侵活动军团奖励支付，按刷怪赏金口径并入总赏金和记录数 |
| ESS 分账 | `ess_escrow_transfer` | 零空 ESS（紧急安全容器）到期结算转账 |
| 任务奖励 | `agent_mission_reward` | 任务奖励，单独计入实际收入 |

`bounty_prizes` 可通过流水 `reason` 字段中的 NPC ID 区分活动类型，体现在「按 NPC 统计」分组中。`corporate_reward_payout` 不提供 NPC 击杀明细，因此只进入总览、趋势、成员排行和流水明细。当前仓库没有单独建模 `corporate_reward_tax` 流水，税额继续沿用 journal 的 `tax` 字段累计，不额外拆分重复口径。

## 当前能力

- 个人单人物 NPC 刷怪报表
- 个人名下所有人物汇总报表
- 公司全员刷怪报表（管理员）
- 总览数据：总悬赏、ESS、税金、实际收入、记录数
- 按 NPC 分类统计（悬赏流水 reason 字段解析）
- 按星系分类统计
- 按天趋势统计
- 分页流水明细
- 多维度筛选：收入类型、星系 ID、人物 ID、用户 ID、金额范围

## 入口

### 前端页面

- `static/src/views/info/npc-kills` — 个人报表（单人物 / 全人物切换）
- `static/src/views/dashboard/npc-kills` — 管理员公司报表

### 后端路由

- `POST /api/v1/info/npc-kills` — 个人单人物报表
- `POST /api/v1/info/npc-kills/all` — 个人全人物汇总
- `POST /api/v1/system/npc-kills` — 公司全员报表（admin，支持可选 `corp_tickers` 逗号分隔筛选）

## 权限边界

- 个人接口要求 `Login`
- 公司报表页面入口位于 `/dashboard/npc-kills`，仅 `admin` 或 `super_admin` 可见
- 公司接口仍为 `/api/v1/system/npc-kills`，后端权限要求 `admin` 或 `super_admin`
- 个人接口在服务层校验 `character_id` 归属，非本人人物返回错误

## 数据来源

所有数据来自本地持久化的 ESI 钱包流水表（`eve_character_wallet_journals`），不实时调用 CCP API。
星系名称来自本地 SDE 数据库（`mapSolarSystems` 表），NPC 名称通过 SDE `GetTypes` 接口查询，支持中英文。

## 核心计算逻辑

Request and response structures are defined in handler code and `static/src/types/api/api.d.ts`.

### 总览（calcSummary）

| 字段 | 计算方式 |
| --- | --- |
| `total_bounty` | 所有 `bounty_prizes` 与 `corporate_reward_payout` 条目 `amount` 之和 |
| `total_ess` | 所有 `ess_escrow_transfer` 条目 `amount` 之和 |
| `total_incursion` | 所有 `corporate_reward_payout` 条目 `amount` 之和，仅作为明细字段保留 |
| `total_mission` | 所有 `agent_mission_reward` 条目 `amount` 之和 |
| `total_tax` | 所有条目 `tax` 之和（通常为负数） |
| `actual_income` | `total_bounty + total_ess + total_mission + total_tax` |
| `total_records` | `bounty_prizes` 与 `corporate_reward_payout` 条目数 |

`corporate_reward_payout` 已并入 `total_bounty`，因此 `actual_income` 不再额外加 `total_incursion`，避免重复计算。

### 按 NPC 统计（calcByNpc）

仅处理 `bounty_prizes` 条目，解析 `reason` 字段格式：

```text
"npc_type_id: kill_count, npc_type_id: kill_count, ..."
```

相同 NPC ID 跨条目累加击杀数，从 SDE 查询本地化 NPC 名称，按击杀数降序排列。
通过此分组，可区分标准刷怪、血族入侵（Sansha NPC ID）和 Pochven（三角洲 NPC ID）等活动类型。

### 按星系统计（calcBySystem）

仅处理 `bounty_prizes` 条目，使用 `context_id` 作为星系 ID，统计每个星系的记录数和总金额，按金额降序排列。`ess_escrow_transfer` 不参与星系统计。

### 趋势（calcTrend）

统计 `bounty_prizes`、`corporate_reward_payout` 与 `agent_mission_reward` 条目，按 `YYYY-MM-DD` 聚合每天的总金额和记录数，按日期升序排列。`ess_escrow_transfer` 不参与趋势。

### 多维度筛选

个人单人物、个人全人物和公司报表接口支持以下筛选字段：

| 字段 | 说明 |
| --- | --- |
| `ref_types` | 收入类型白名单，可选 `bounty_prizes`、`ess_escrow_transfer`、`corporate_reward_payout`、`agent_mission_reward` |
| `solar_system_ids` | 星系 ID 列表，仅匹配 `bounty_prizes.context_id`，因此会排除无星系上下文的 `corporate_reward_payout`、ESS 和任务奖励 |
| `character_ids` | 人物 ID 列表；个人全人物接口会限制在当前用户人物内，公司接口会限制在军团 ticker 过滤后的已绑定人物内 |
| `user_ids` | 用户 ID 列表；个人接口只允许当前用户，公司接口限制在可见人物对应用户内 |
| `min_amount` / `max_amount` | 按流水 `amount` 做闭区间过滤 |

筛选条件影响全部统计结果，包括总览、成员排行、NPC/星系/趋势和流水明细。

## 流水明细字段

| 字段 | 说明 |
| --- | --- |
| `ref_type` | `bounty_prizes`、`corporate_reward_payout`、`ess_escrow_transfer` 或 `agent_mission_reward` |
| `amount` | 本次收入金额（正数） |
| `tax` | 扣税金额（通常为负数） |
| `solar_system_name` | 仅 `bounty_prizes` 有值（来自 context_id） |
| `reason` | NPC ID 原始字符串，格式同上 |
| `character_name` | 全人物汇总和公司报表时填充 |

## UI 呈现

### 个人页面

1. 人物选择器（下拉，含头像）+ 「所有人物」选项 + 日期范围选择 + 收入类型 / 星系 ID / 人物 ID / 用户 ID / 金额范围筛选
2. 4 卡片总览：总悬赏 / 总税金 / 实际收入 / 记录数
3. 双列布局：按 NPC 统计表 + 按星系统计表
4. 时间趋势表（有数据时显示）
5. 分页流水明细表（ref_type 以 tag 展示，金额带颜色）

### 管理员页面

1. 日期范围选择 + 军团 ticker 多选筛选（默认 `FUXI`、`FMA.1`）+ 收入类型 / 星系 ID / 用户 ID / 人物 ID / 金额范围筛选
2. 4 卡片总览（同上）
3. 成员列表（按实际收入降序，按系统用户聚合，展示昵称、角色数、悬赏 / ESS / 税 / 实际收入 / 记录数）
4. 双列布局：按星系统计 + 时间趋势
5. 无流水明细（管理视角不展示个人流水）

## 关键不变量

- `bounty_prizes`、`corporate_reward_payout`、`ess_escrow_transfer` 和 `agent_mission_reward` 是唯一参与计算的 ref_type，其余钱包类型不纳入
- `corporate_reward_payout` 并入 `total_bounty` 和记录数，但 `total_incursion` 不重复计入 `actual_income`
- 税金（`tax` 字段）为负数，参与实际收入计算时相当于扣减
- 当前仓库未单独拆分 `corporate_reward_tax` 流水；如果后续接入该 ref_type，需要重新评估是否按独立税项口径入账，避免和 `tax` 字段重复
- 公司成员排行按系统用户聚合，不再按单角色聚合
- 公司成员显示名优先 `user.nickname`，为空时回退为该用户本次统计内角色名字典序首个名称
- 星系统计仅基于 `bounty_prizes`，`corporate_reward_payout`、ESS 和任务奖励不带可靠星系上下文
- 趋势统计包含 `bounty_prizes`、`corporate_reward_payout` 和 `agent_mission_reward`
- 个人接口强制校验人物归属，不可跨用户查询
- 公司接口只涵盖当前已绑定有效 token 的人物

## 主要代码文件

- `server/internal/service/npc_kill.go` — 业务逻辑（汇总计算、NPC/星系/趋势解析）
- `server/internal/repository/npc_kill.go` — 数据查询（钱包流水、星系名称）
- `server/internal/handler/npc_kill.go` — HTTP 处理层
- `static/src/api/npc-kill.ts` — 前端 API 封装
- `static/src/views/info/npc-kills` — 个人报表页面
- `static/src/views/dashboard/npc-kills` — 管理员报表页面

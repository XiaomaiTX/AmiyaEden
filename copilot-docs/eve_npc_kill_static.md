# EVE 刷怪报表

## 概述

通过`eve_character_wallet_journal`表，我们可以分析玩家在EVE Online中的刷怪活动。将报表展示给个人/管理。

## 数据分析

1. 数据结构

    - `eve_character_wallet_journal.ref_type`: `ess_escrow_transfer` / `bounty_prizes`
       > 刷怪数据 由`bounty_prizes`提供，`ess_escrow_transfer`不包含刷怪数据。但刷怪总赏金由 `bounty_prizes` + `ess_escrow_transfer` 提供。
    - `eve_character_wallet_journal.amount`: 刷怪金额
    - `reason`: `npc_id: num,npc_id: num`，包含NPC ID和数量
       > `npc_id` -> `invTypes.typeID` | `npc_name` -> `invTypes.typeName`
    - `eve_character_wallet_journal.context_id`: 刷怪地点
       > `context_id` -> `solar_system_id` -> `mapSolarSystems.solarSystemName`
    - `eve_character_wallet_journal.tax`: 交税金额

2. 数据处理

    - 过滤数据：仅保留`ref_type`为`bounty_prizes` 和 `ess_escrow_transfer`的记录。
    - 解析`reason`字段：提取NPC ID和数量，关联`invTypes`表获取NPC名称。
    - 关联`context_id`：获取刷怪地点名称。
    - 计算总赏金：将`amount`与`tax`相加，得到实际获得的赏金。

## 报表展示

    - 总览：显示总刷怪金额、总交税金额、实际获得的赏金。
    - 按NPC分类：展示每种NPC的刷怪金额和数量。
    - 按地点分类：展示每个刷怪地点的刷怪金额和数量。
    - 时间趋势：展示刷怪金额随时间的变化趋势。
    - 大致时长：每一条记录相当于刷了20min，但是有些明显低于平均金额的不算

## router

1. /info/npc-kills 显示当前用户的刷怪报表
2. /corp/npc-kills 给admin显示公司内所有成员的刷怪报表

## 要求

1. 表格遵循 `./TABLE_REQUIRE.md` 的规范。
2. 只接受GET/POST请求。
3. 数据只在用户请求时计算，不进行定时任务。

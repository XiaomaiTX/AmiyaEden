---
status: active
doc_type: feature
owner: engineering
last_reviewed: 2026-03-26
source_of_truth:
  - server/internal/model/pap_type_rate.go
  - server/internal/repository/pap_type_rate.go
  - server/internal/service/pap_exchange.go
  - server/internal/model/sys_config.go
  - server/internal/model/sys_wallet.go
  - server/internal/repository/sys_wallet.go
  - server/internal/service/fleet.go
  - server/internal/handler/pap_exchange.go
  - server/internal/router/router.go
  - static/src/api/pap-exchange.ts
  - static/src/views/system/pap-exchange
  - server/internal/service/multichar_reward.go
---

# PAP 兑换汇率

## 概述

PAP 兑换汇率功能允许管理员为每种舰队行动类型（Skirmish / Strategic / CTA）单独配置伏羲币兑换比率，并为 FC 设置固定工资和每月工资上限。FC 发放 PAP 时，后端会优先判断成员是否为该舰队的 `FCUserID`；若是，则发放固定工资，但同一 FC 每月最多领取配置上限次数；否则按舰队重要性换算为对应伏羲币金额。

此功能与联盟 PAP（Alliance PAP）系统完全独立。联盟 PAP 月度归档当前为纯归档操作，不涉及钱包兑换。

## 当前能力

- 管理员可在「系统管理 → PAP兑换」页面查看并修改三种 PAP 类型的钱包兑换汇率
- 默认汇率：Skirmish 10、Strategic 30、CTA 50（伏羲币 / 1 PAP）
- 管理员可在同一页面设置 `FC工资`，默认值为 400 伏羲币
- 管理员可在同一页面设置 `FC工资上限次数`，默认值为每月 5 次
- 管理员可在同一页面设置 `admin-award`，默认值为 10 伏羲币；当管理员在福利审批页或商店订单管理页执行人工发放成功时，会自动获得该奖励，设为 `0` 可关闭；仅有 `welfare` / `shop_order_manage` 职权的执行人不会领取该奖励
- FC 对某舰队执行「发放 PAP」时，系统根据该舰队的 `importance` 字段自动选择对应汇率
- 若被发放成员的 `user_id` 等于舰队 `FCUserID`，则该成员按固定工资发放；若该 FC 本月已达到工资上限，则本次工资记为 0
- 汇率持久化于数据库 `pap_type_rate` 表；缺失行在首次读取时用默认值自动补全
- FC 工资持久化于 `system_config` 表，键名为 `pap.fc_salary`
- FC 工资上限次数持久化于 `system_config` 表，键名为 `pap.fc_salary_limit`
- 管理发放奖励持久化于 `system_config` 表，键名为 `pap.admin_award`

## PAP 类型映射

| 舰队重要性（`fleet.importance`） | PAP 类型 | 默认汇率 |
| --- | --- | --- |
| `cta` | CTA（全面集结） | 50 |
| `strat_op` | Strategic（战略行动） | 30 |
| `other`（及其他未知值） | Skirmish（游击队） | 10 |

## 关键不变量

- 汇率配置存储在 `pap_type_rate` 表，以 `pap_type`（`skirmish` / `strat_op` / `cta`）为主键
- `pap_type_rate` 与 `system_config` 中的 `pap.wallet_per_pap` 互相独立；后者仅用于联盟 PAP 月度兑换伏羲币结算
- `pap.fc_salary` 与 `pap.fc_salary_limit` 分别控制 FC 工资金额与每月领取次数；两者都与 PAP 类型汇率互相独立
- `papImportanceToWalletRate` 是将舰队重要性映射到汇率的纯函数；当 `pap_type` 不在汇率表中时回退到 1
- FC 工资单独写入 `wallet_transaction` 的 `pap_fc_salary` 流水类型，便于按月计数与审计
- 三种 PAP 类型固定不可增删；管理页面仅允许修改汇率数值
- 重新发放 PAP（re-issue）时钱包差量按汇率换算，与首次发放一致

## 多人物奖励限制

当同一用户有多个人物参与同一舰队时，PAP 奖励按人物数量递减：

| 层级 | 默认人物数 | 奖励比例 |
| --- | --- | --- |
| 满额 | 前 3 个 | 100% |
| 折扣 | 第 4–6 个 | 50% |
| 无奖励 | 第 7 个起 | 0% |

管理员可在「系统管理 → PAP兑换」页面配置三项参数：

- `multichar.full_reward_count`（满额奖励人物数，默认 3）
- `multichar.reduced_reward_count`（折扣奖励人物数，默认 3）
- `multichar.reduced_reward_pct`（折扣百分比，默认 50）

此限制仅影响 PAP 钱包奖励，不影响 FC 工资（FC 工资按自然人发放，不受人物数量限制）。

人物排序方式为按 `CharacterID` 升序；排序仅在用户人物数超过满额层级时影响哪些人物落入折扣或无奖励层级。

## 入口

### 前端页面

- `static/src/views/system/pap-exchange`

### 后端路由

- `GET /api/v1/system/pap-exchange/rates`
- `PUT /api/v1/system/pap-exchange/rates`

## 主要代码文件

- `server/internal/model/pap_type_rate.go` — 模型、类型常量、`NormalizePAPLevel`
- `server/internal/repository/pap_type_rate.go` — 数据访问、默认值补全
- `server/internal/service/pap_exchange.go` — 汇率 CRUD 业务逻辑
- `server/internal/service/fleet.go` — `papImportanceToWalletRate`、FC 工资上限、FC 工资与钱包差量计算
- `server/internal/handler/pap_exchange.go` — HTTP 处理器
- `static/src/api/pap-exchange.ts` — 前端 API 包装层
- `static/src/views/system/pap-exchange/index.vue` — 管理页面

## 回归测试

- `server/internal/model/pap_type_rate_test.go` — `NormalizePAPLevel` 的所有输入分支
- `server/internal/service/fleet_test.go` — `papImportanceToWalletRate` 的所有映射分支与缺失键回退
- `server/internal/service/fleet_test.go` — FC 工资与每月上限的计算分支

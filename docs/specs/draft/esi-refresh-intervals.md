---
status: draft
doc_type: draft
owner: engineering
last_reviewed: 2026-05-10
source_of_truth:
  - server/pkg/eve/esi/queue.go
  - server/pkg/eve/esi/activity.go
  - server/pkg/eve/esi/task_killmails.go
  - server/pkg/eve/esi/task_corp_roles.go
  - server/pkg/eve/esi/task_online.go
  - server/pkg/eve/esi/task_corp_killmails.go
  - server/pkg/eve/esi/task_corporation_history.go
  - server/pkg/eve/esi/task_skill.go
  - server/pkg/eve/esi/task_titles.go
  - server/pkg/eve/esi/task_fittings.go
  - server/pkg/eve/esi/task_assets.go
  - server/pkg/eve/esi/task_affiliation.go
  - server/pkg/eve/esi/task_clones.go
  - server/pkg/eve/esi/task_contracts.go
  - server/pkg/eve/esi/task_notifications.go
  - server/pkg/eve/esi/task_wallet.go
  - server/pkg/eve/esi/task_structure.go
  - server/pkg/eve/esi/task_corporation_structures.go
---

# ESI 刷新间隔草案

> 本文是刷新间隔建议稿，不直接改变当前运行时行为。

## 口径说明

- `Active` / `Inactive` 是调度器分档，不是“网页系统在线状态”。
- 当前实现里，人物是否活跃由 ESI `character_online` 的 `last_login` 与活跃缓存判定；近 7 天内登录视为活跃。
- 当活跃判定失败时，队列默认按活跃处理，避免任务被错误长期压低。
- `character_assets` 不应只按间隔问题处理，需要单独评估并发、名称查询和整表重建的处理状态。

## 建议间隔表

| 任务名称 | 描述 | 优先级 | 建议活跃间隔 | 建议非活跃间隔 | 所需权限 | 备注 |
| --- | --- | --- | --- | --- | --- | --- |
| `character_killmails` | 人物击杀/损失邮件 | 极高 | `1 Hour` | `7 Days` | `esi-killmails.read_killmails.v1` | 作为 SRP 申请和补损核对的时间敏感数据，建议尽快收敛到 1 小时级。 |
| `character_corp_roles` | 人物军团职权 | 高 | `1 Day` | `1 Day` | `esi-characters.read_corporation_roles.v1` | 保持日更即可，主要用于权限同步和军团信号。 |
| `character_online` | 人物在线状态（活跃度检测） | 高 | `2 Hours` | `1 Day` | `esi-location.read_online.v1` | 活跃用 2 小时节奏，非活跃可放宽到日更，避免无谓轮询。 |
| `corporation_killmails` | 军团击杀/损失邮件（管理员） | 普通 | `1 Hour` | `1 Day` | `esi-killmails.read_corporation_killmails.v1` | 维持军团 KM 覆盖率，服务 SRP 和军团级补损。 |
| `character_corporation_history` | 人物军团任职历史 | 普通 | `7 Days` | `7 Days` | 无需权限 | 任职历史变化低频，周更足够。 |
| `character_skill` | 人物技能信息 | 普通 | `6 Hours` | `7 Days` | `esi-skills.read_skills.v1` / `esi-skills.read_skillqueue.v1` | 优先保证当天能看到技能变化；如果更重视负载，可再回退到 12 小时。 |
| `character_titles` | 人物军团头衔 | 普通 | `1 Day` | `1 Day` | `esi-characters.read_titles.v1` | 头衔变化通常与军团管理同步，日更足够。 |
| `character_fittings` | 人物装配 | 普通 | `6 Hours` | `7 Days` | `esi-fittings.read_fittings.v1` / `esi-fittings.write_fittings.v1` | 保持现有节奏即可。 |
| `character_assets` | 人物资产（物品/位置/名称） | 普通 | `1 Day` | `7 Days` | `esi-assets.read_assets.v1` | 需单独评估并发和处理状态，暂不只按间隔收敛。 |
| `character_affiliation` | 人物归属信息（军团/联盟/阵营） | 普通 | `6 Hours` | `6 Hours` | 无需权限 | 归属信息相对稳定，但需要比日更更快反映变更。 |
| `character_clones` | 人物克隆体/植入体/跳跃疲劳 | 普通 | `12 Hours` | `7 Days` | `esi-clones.read_clones.v1` / `esi-clones.read_implants.v1` / `esi-characters.read_fatigue.v1` | 介于“变化不频繁”与“当天可见”之间，12 小时更平衡。 |
| `character_contracts` | 人物合同（含竞标/物品） | 普通 | `12 Hours` | `7 Days` | `esi-contracts.read_character_contracts.v1` | 合同的可见性对日内操作更有价值，12 小时可接受。 |
| `character_notifications` | 人物通知消息 | 普通 | `1 Day` | `7 Days` | `esi-characters.read_notifications.v1` | 通知更偏追溯型，不必高频。 |
| `character_wallet` | 人物钱包信息 | 普通 | `12 Hours` | `7 Days` | `esi-wallet.read_character_wallet.v1` | 余额和流水都需要一定时效，但无需高频轮询。 |
| `eve_structures` | EVE 建筑详情（个人相关） | 低 | `3 Days` | `7 Days` | `esi-universe.read_structures.v1` | 个人相关建筑通常作为补充信息即可。 |
| `corporation_structures` | 军团建筑信息 | 高 | `6 Hours` | `1 Day` | `esi-corporations.read_structures.v1` / `esi-universe.read_structures.v1` | 比现有 1 天更积极，但仍不必收缩到 1 小时以内。 |

## 设计理由

- `character_killmails` 的目标是让 SRP 与补损核对尽快看到新 KM，1 小时能覆盖大多数出击后提交窗口。
- `character_online` 不需要 30 分钟级轮询；调度目标是稳定分层，不是实时在线监控。
- `character_skill`、`character_contracts`、`character_clones` 属于低频变化但对当日可见性有价值的数据，6-12 小时更合适。
- `corporation_structures` 需要比日更更快发现燃料、计时器和状态变化，但仍需照顾 ESI 负载。
- `character_assets` 的复杂度不在间隔本身，而在全量重建、名称批量查询和并发风暴控制，需单独拆出评估。

## 结论

- 本轮建议只调整你点名的几项间隔，不扩散到未明确要求的任务。
- `character_online` 采用 `2 Hours / 1 Day`，活跃与非活跃分开处理。
- `character_skill` 采用 `6 Hours / 7 Days`，先兼顾当天可见性与负载。
- `character_assets` 保持当前节奏视为临时方案，后续单独做并发与状态设计。

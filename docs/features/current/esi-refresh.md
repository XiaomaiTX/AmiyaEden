---
status: active
doc_type: feature
owner: backend
last_reviewed: 2026-04-09
source_of_truth:
  - server/jobs/esi_refresh.go
  - server/jobs/auto_srp_schedule.go
  - server/internal/handler/esi_refresh.go
  - server/internal/model/esi/corporation_history.go
  - server/pkg/eve/esi/task_corp_killmails.go
  - server/pkg/eve/esi/task_corporation_history.go
  - static/src/api/esi-refresh.ts
  - static/src/views/system/esi-refresh
---

# ESI 刷新队列

## 当前能力

- 周期性运行 ESI 刷新队列
- 查看任务列表与状态
- 手动执行队列调度
- 按任务名执行
- 对指定人物执行全部任务
- 新人物登录 / 绑定后触发同步钩子
- 舰队 PAP 发放后登记一次性自动 SRP 延迟调度
- 服务启动时恢复未执行的一次性自动 SRP 调度
- 服务启动时立即补跑一次按 freshness 判定的 ESI 队列调度，避免新任务首周空窗
- 军团击杀邮件定期拉取（管理员授权的可选 scope）
- 对已有军团 KM 覆盖的成员，自动队列跳过个人 killmail 刷新

## 任务列表

间隔值为各任务 `Interval()` 返回的标准值；修改任务间隔前，必须先更新此表并在 PR 中说明原因。

| 任务名 | Scope | 可选 | 活跃间隔 | 非活跃间隔 | 优先级 | 用途 |
|--------|-------|------|----------|-----------|--------|------|
| `character_affiliation` | 无（公开接口） | 否 | 6 小时 | 6 小时 | Normal | 人物归属信息（军团/联盟/阵营）；不区分活跃度 |
| `character_assets` | `esi-assets.read_assets.v1` | 否 | 24 小时 | 7 天 | Normal | 人物资产 |
| `character_clones` | `esi-clones.read_clones.v1` 等 3 个 | 否 | 24 小时 | 24 小时 | Normal | 克隆体、植入体、跳跃疲劳 |
| `character_corporation_history` | 无（公开接口） | 否 | 7 天 | 7 天 | Normal | 人物军团任职历史；为服役年限等后续用例提供持久化数据 |
| `character_contracts` | `esi-contracts.read_character_contracts.v1` | 否 | 24 小时 | 7 天 | Normal | 人物合同 |
| `character_corp_roles` | `esi-characters.read_corporation_roles.v1` | 否 | 24 小时 | 24 小时 | High | 人物军团职权 |
| `character_fittings` | `esi-fittings.read_fittings.v1` 等 2 个 | 否 | 6 小时 | 7 天 | Normal | 人物装配 |
| `character_killmails` | `esi-killmails.read_killmails.v1` | 否 | 24 小时 | 7 天 | Critical | 人物击杀/损失邮件 |
| `character_notifications` | `esi-characters.read_notifications.v1` | 否 | 24 小时 | 7 天 | Normal | 人物通知 |
| `character_online` | `esi-location.read_online.v1` | 否 | 30 分钟 | 2 小时 | High | 人物在线状态 |
| `character_skill` | `esi-skills.read_skills.v1` 等 2 个 | 否 | 24 小时 | 7 天 | Normal | 技能与技能队列 |
| `character_titles` | `esi-characters.read_titles.v1` | 否 | 24 小时 | 24 小时 | Normal | 人物头衔 |
| `character_wallet` | `esi-wallet.read_character_wallet.v1` | 否 | 12 小时 | 7 天 | Normal | 人物钱包 |
| `corporation_killmails` | `esi-killmails.read_corporation_killmails.v1` | 是（仅管理员） | 60 分钟 | 1 天 | Normal | 军团范围击杀邮件拉取，为自动 SRP 提供更全面的 KM 覆盖 |
| `eve_structures` | `esi-universe.read_structures.v1` 等 2 个 | 否 | 3 天 | 7 天 | Low | 建筑信息 |

## 入口

- 管理页面：`static/src/views/system/esi-refresh`
- 路由：`/api/v1/esi/refresh/*`
- 运行时调度：`server/jobs/esi_refresh.go`

## 权限边界

- 所有 `/api/v1/esi/refresh/*` 路由要求 `admin`

## 关键不变量

- 新增 ESI 数据模块时，通常不只改一个 handler，还需要任务注册、scope、持久化、前端消费一起落地
- 队列与登录后同步钩子共享同一套任务体系
- `character_corporation_history` 在成功同步时会全量替换该人物的历史行集，并重算 `eve_character.fuxi_legion_tenure_days`
- PAP 发放不会直接为舰队成员逐个触发 `character_killmails`；自动 SRP 依赖已有的个人或军团 KM 数据
- 当某军团已有同时具备 `corporation_killmails` scope 与 `Director` 职权、且最近一次军团 KM 刷新仍在有效期内的授权人物时，自动队列不会再为该军团成员安排 `character_killmails`
- 自动 SRP 的 PAP 后延迟计划是一次性尝试；如果旧计划执行期间同舰队又重新发放 PAP，旧计划不会清掉更新后的计划时间
- 如果要新增模块，请先遵循 `docs/guides/adding-esi-feature.md`
- 所有 ESI API 端点通过 `server/config/config.go` 中的 `EveSSOConfig.ESIBaseURL` 和 `ESIAPIPrefix` 配置管理，禁止在 service 层硬编码 ESI URL
- ESI 刷新队列通过接口注入（`TokenService`、`CharacterRepository`）避免循环依赖，不直接依赖具体 service / repository 实现

## 主要代码文件

- `server/jobs/esi_refresh.go`
- `server/internal/handler/esi_refresh.go`
- `server/internal/model/esi/corporation_history.go`
- `server/pkg/eve/esi/task_corp_killmails.go`
- `server/pkg/eve/esi/task_corporation_history.go`
- `server/pkg/eve/esi`
- `static/src/api/esi-refresh.ts`
- `static/src/views/system/esi-refresh`

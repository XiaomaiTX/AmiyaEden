---
status: draft
doc_type: draft
owner: engineering
last_reviewed: 2026-03-27
---

# 新人帮扶设计

## 目标

在现有 AmiyaEden 系统中新增 `新人帮扶` 模块，支持：

- 管理员将已有用户授予新的系统角色 `captain`
- 新人玩家在 `新人帮扶` 页面对队长进行选择或切换
- 记录玩家与队长之间的帮扶关系历史，包含开始时间与结束时间
- 基于钱包刷怪收入与星系/时间窗口匹配，生成持久化的队长归因台账
- 队长查看自己名下玩家与归因收益
- 管理员查看全部队长绩效

本设计中统一使用术语 `队长`，不使用 `导师`。

## 范围

本次设计覆盖：

- RBAC 角色、菜单、页面入口
- 新人资格判定
- 玩家选择队长与关系历史
- 队长归因台账生成与查询
- 队长视角与管理员视角报表
- 前后端契约与验证要求

本次设计不覆盖：

- 队长奖金结算规则
- 自动发放奖金
- 对既有钱包同步任务做大规模重构

## 术语与业务规则

### 队长

- `captain` 是新的真实系统角色
- 是否能访问队长页面，最终由后端角色校验决定
- `captain` 的授予沿用现有 `/api/v1/system/user/:id/roles` 管理流程，不新增独立分配接口

### 新人玩家定义

用户被视为 `新人玩家`，必须同时满足：

1. 其名下不存在任何一个绑定角色满足 `total_sp >= 20,000,000`
2. 其名下位于 `allow_corporations` 内的绑定角色数量小于 `4`

只要任一条件不满足，该用户就不能进入队长选择页面，也不能调用相关选择接口。

### 新人资格持久化策略

由于新人资格判定未来可能从硬编码规则演进为可配置规则，本设计不采用“永久毕业且不可回退”的单向状态。

改为采用“持久化资格快照 + 按需重算”：

- 将当前资格评估结果持久化，避免每次访问时都全量实时重算
- 当资格规则、角色技能数据或军团归属数据发生变化时，允许重新评估
- 后续用户侧页面与接口优先读取该持久化快照进行快速判定

毕业条件与上文定义一致：

1. 任一绑定角色满足 `total_sp >= 20,000,000`
2. 位于 `allow_corporations` 内的绑定角色数量达到 `4` 或以上

该状态不是永久毕业标记，而是“按当前规则评估后的最近一次结果”。

### 玩家与队长关系

- 关系绑定到平台 `user`
- 同一时刻一个玩家只能有一个有效队长
- 玩家可以频繁切换队长
- 玩家可以在结束后再次重新选择曾经的队长
- 关系历史需要保留开始时间与结束时间

### 角色与匹配口径

- 新人资格：检查用户名下全部绑定角色
- 玩家选择队长：使用玩家当前主角色进行展示与选择体验，但关系本身绑定到 `player_user_id`
- 玩家收益匹配：检查玩家名下全部绑定角色的钱包流水
- 队长归因匹配：只检查队长当前主角色的钱包流水

## 方案比较

### 方案 A：历史关系 + 持久化归因台账 + 增量同步

核心做法：

- 存储玩家与队长关系历史
- 对每条玩家刷怪收入生成持久化归因台账
- 通过增量同步处理新钱包流水

优点：

- 满足“像总账一样持久化”的要求
- 后续奖金结算直接读取台账，避免重复计算或重复发放
- 可审计、可追溯、可做明细展示

缺点：

- 需要新增表结构与同步状态
- 比纯查询方案略复杂

### 方案 B：按天聚合快照

核心做法：

- 保存每日/每队长汇总，不保留逐条归因明细

缺点：

- 审计性弱
- 难以支撑后续奖金核对
- 不适合“不要重新计算或重复发放”的要求

### 方案 C：查询时实时计算

核心做法：

- 每次打开页面时临时从钱包流水计算

缺点：

- 不满足持久化台账要求
- 队长主角色变化后结果会漂移
- 不适合作为奖金结算依据

### 推荐

采用方案 A。

## 数据模型

### 1. `newbro_player_state`

用于存储新人资格的最近一次评估结果，避免每次都重复计算。

建议字段：

- `id`
- `user_id`
- `is_currently_newbro`
- `evaluated_at`
- `rule_version`
- `disqualified_reason`
- `created_at`
- `updated_at`

约束：

- `user_id` 唯一

说明：

- `is_currently_newbro` 表示按当前已生效规则评估后的结果
- `rule_version` 用于识别资格规则变更后的缓存失效与重算
- `disqualified_reason` 建议枚举化，例如：
  - `skill_point_threshold_reached`
  - `allow_corporation_character_count_reached`
- 当用户当前仍属于新人时，`disqualified_reason` 为空
- 本表是资格快照，不是永久毕业日志

### 2. `newbro_captain_affiliation`

用于记录玩家与队长的帮扶关系历史。

建议字段：

- `id`
- `player_user_id`
- `player_primary_character_id_at_start`
- `captain_user_id`
- `started_at`
- `ended_at`
- `created_at`
- `updated_at`

约束：

- 同一个 `player_user_id` 同时只能存在一条有效关系
- `ended_at IS NULL` 表示当前有效关系

说明：

- 不需要 `ended_reason`
- 允许高频结束与重建，例如一天多次切换

### 3. `captain_bounty_attribution`

用于持久化队长归因台账，是后续奖金计算的唯一依据。

建议字段：

- `id`
- `affiliation_id`
- `player_user_id`
- `player_character_id`
- `captain_user_id`
- `captain_character_id`
- `captain_wallet_journal_id`
- `wallet_journal_id`
- `ref_type`
- `system_id`
- `journal_at`
- `amount`
- `created_at`
- `updated_at`

约束：

- `wallet_journal_id` 唯一，确保同一条玩家钱包流水不会被重复归因

说明：

- `ref_type` 当前写入值只会来自 `bounty_prizes`
- `captain_character_id` 会被持久化，后续展示和奖金结算都读取该值
- `captain_wallet_journal_id` 用于记录本次归因所匹配到的队长钱包流水，便于审计与去歧义

### 4. `captain_bounty_sync_state`

用于增量归因处理的同步进度控制。

建议字段：

- `id`
- `sync_key`
- `last_wallet_journal_id`
- `last_journal_at`
- `updated_at`

说明：

- 只要能稳定支持增量扫描即可，字段形式可以按实现便利性微调

## 归因规则

### 归因输入

输入源来自玩家钱包流水表中的 `bounty_prizes` 记录。

### 系统级回溯窗口

归因同步存在系统级时间窗口限制：

- 只处理距离当前时间不超过 `1` 个月的玩家钱包流水
- 超过 `1` 个月的历史记录，即使尚未归因，也不再补算

该窗口同样适用于首次上线后的历史补算。

### 归因条件

一条玩家钱包流水可被归因为某队长，必须同时满足：

1. 该流水所属角色归属于某个玩家用户
2. 该玩家在 `journal_at` 对应时间点存在有效的 `newbro_captain_affiliation`
3. 能定位到该关系对应的队长用户
4. 取该队长用户在归因处理时的当前 `primary_character_id`
5. 队长该主角色存在一条钱包流水，与玩家流水满足：
   - `system_id` 相同
   - `reason` 字段解析出的 NPC ID 与数量一致
   - 时间差不超过 `15` 分钟
6. 玩家流水尚未写入 `captain_bounty_attribution`

满足条件后，写入一条 `captain_bounty_attribution` 记录。

### 多候选匹配的确定性规则

若在同一 `system_id` 与 `15` 分钟窗口内，队长钱包流水存在多条候选记录，则必须按以下顺序确定唯一匹配：

1. 优先选择与玩家流水 `journal_at` 绝对时间差最小的记录
2. 若时间差相同，优先选择源钱包流水原始时间列 `date` 更早的记录
3. 若仍相同，优先选择 `id` 更小的记录

说明：

- 不要求玩家流水金额与队长流水金额相同
- 队长候选流水当前只从 `bounty_prizes` 中选取
- 最终选中的队长流水 `id` 应写入 `captain_wallet_journal_id`
- 这里的源时间字段明确指钱包流水表中的原始时间列 `date`；写入归因台账后统一映射到 `journal_at`

### 重要 Caveat

本设计不维护队长主角色历史表。

因此：

- `captain_bounty_attribution.captain_character_id` 代表的是“归因处理执行时，队长当前主角色”的值
- 它不保证严格等于“玩家该笔收益发生时，队长当时的主角色”

这是本设计有意接受的业务边界。后续所有报表与奖金结算都以归因台账中已持久化的 `captain_character_id` 为准，不再回溯重算。

## 写入流程

### 玩家选择队长

当玩家提交选择时：

1. 后端优先读取 `newbro_player_state`
2. 若状态不存在、规则版本过期或依赖数据已刷新，则先重算资格快照
3. 校验目标用户是否拥有 `captain` 角色
4. 查询当前是否已有有效关系
5. 如果已有且目标队长相同，可返回当前关系，不重复创建
6. 如果已有且目标队长不同：
   - 将旧关系 `ended_at` 设为当前时间
   - 创建新关系，`started_at` 为当前时间
7. 如果没有有效关系：
   - 直接创建新关系

### 新人资格状态推进

建议在以下场景尝试推进 `newbro_player_state`：

- 角色技能数据刷新后
- 角色归属/军团信息刷新后
- 新人资格规则配置变更后
- 用户访问新人帮扶入口且本地状态不存在或已过期时

推进规则：

1. 读取当前绑定角色与技能/军团数据
2. 按当前 `rule_version` 计算资格
3. 将结果写回 `newbro_player_state`
4. 若未来规则放宽，允许已判定为非新人的用户在重算后重新成为新人

### 归因同步

建议先实现“手动触发 + 可复用服务”的模式。

处理步骤：

1. 从同步状态中读取上次处理位置
2. 只扫描 `date >= now - 1 month` 且尚未归因的玩家钱包流水
3. 过滤 `bounty_prizes`
4. 解析该角色所属玩家用户
5. 读取该用户当前 `newbro_player_state`
6. 只有当用户当前仍是新人时，才继续归因流程
7. 查询该时间点有效的帮扶关系
8. 取对应队长用户的当前主角色
9. 查找匹配的队长钱包流水
10. 满足条件则写入归因台账
11. 推进同步状态

该服务应保证幂等性：

- 同一玩家钱包流水重复扫描时，不会生成重复归因记录

### 首次上线后的补算规则

- 首次上线后，允许对“当前仍是新人”的用户补算尚未归因的历史记录
- 补算范围仅限最近 `1` 个月内的未归因记录
- 若用户当前已经不是新人，则即使其更早时期存在符合条件的关系与钱包流水，也不补算
- 已经写入 `captain_bounty_attribution` 的记录不会因用户后续资格变化而被删除或重算

## 权限与菜单

### 新角色

新增系统角色：

- `captain`

### 菜单结构

新增根菜单：

- `新人帮扶`

子页面：

- `新人选队长`
- `队长帮扶`
- `帮扶管理`

### 可见性与后端边界

#### 新人选队长

- 前端：仅对登录用户展示
- 后端：`RequireLoginUser()`，并在 service 中优先校验持久化的新人状态，必要时再做兜底资格计算

#### 队长帮扶

- 前端：仅对 `captain` 角色展示
- 后端：`RequireRole(captain)`

#### 帮扶管理

- 前端：仅对 `admin` 展示
- 后端：`RequireRole(admin)`

说明：

- 前端隐藏仅用于 UX
- 最终权限必须由后端决定

## API 草案

### 用户侧

- `GET /api/v1/newbro/captains`
  - 权限边界：`Login`
  - 业务边界：仅当前评估结果仍为新人的用户可成功调用；当前非新人用户返回业务拒绝
  - 返回可选择的队长列表
  - 最小返回字段：
    - `captain_user_id`
    - `captain_character_id`
    - `captain_character_name`
    - `captain_portrait_url`
    - `active_newbro_count`
- `GET /api/v1/newbro/affiliation/me`
  - 权限边界：`Login`
  - 业务边界：任意登录用户都可调用，用于返回当前用户的新人状态与自己的当前/最近关系摘要
  - 返回当前用户的当前关系与最近关系历史
  - 最小返回字段：
    - `is_currently_newbro`
    - `evaluated_at`
    - `rule_version`
    - `disqualified_reason`
    - `current_affiliation`
    - `recent_affiliations`
  - `current_affiliation` 最小字段：
    - `affiliation_id`
    - `captain_user_id`
    - `captain_character_id`
    - `captain_character_name`
    - `captain_portrait_url`
    - `started_at`
    - `ended_at`
  - `recent_affiliations` 定义为最近 `10` 条关系记录，按 `started_at DESC, id DESC` 排序
  - `recent_affiliations` 列表项字段与 `current_affiliation` 一致
- `POST /api/v1/newbro/affiliation/select`
  - 权限边界：`Login`
  - 业务边界：仅当前评估结果仍为新人的用户可成功调用；当前非新人用户返回业务拒绝
  - 选择或切换队长
  - 请求体最小字段：
    - `captain_user_id`
  - 返回最小字段：
    - `affiliation_id`
    - `captain_user_id`
    - `started_at`

### 队长侧

- `GET /api/v1/newbro/captain/overview`
  - 返回队长总览
  - 最小返回字段：
    - `captain_user_id`
    - `captain_character_id`
    - `captain_character_name`
    - `active_player_count`
    - `historical_player_count`
    - `attributed_bounty_total`
    - `attribution_record_count`
- `GET /api/v1/newbro/captain/players`
  - 返回当前/历史玩家列表
  - 默认按“当前有效关系优先、开始时间倒序”排序
  - 支持分页，默认第一页、默认页大小 `20`
  - 查询参数最小字段：
    - `current`
    - `size`
    - `status`，取值建议为 `active | historical | all`
  - 列表项最小字段：
    - `player_user_id`
    - `player_character_id`
    - `player_character_name`
    - `player_portrait_url`
    - `started_at`
    - `ended_at`
    - `attributed_bounty_total`
- `GET /api/v1/newbro/captain/attributions`
  - 返回归因台账明细与汇总
  - 默认按 `journal_at DESC, id DESC` 排序
  - 支持分页，默认第一页、默认页大小 `20`
  - 查询参数最小字段：
    - `current`
    - `size`
    - `player_user_id`
    - `ref_type`
    - `start_date`
    - `end_date`
  - 汇总字段最小集合：
    - `attributed_bounty_total`
    - `record_count`
  - 列表项最小字段：
    - `id`
    - `player_user_id`
    - `player_character_id`
    - `player_character_name`
    - `captain_character_id`
    - `captain_character_name`
    - `captain_wallet_journal_id`
    - `wallet_journal_id`
    - `ref_type`
    - `system_id`
    - `journal_at`
    - `amount`

### 管理侧

- `GET /api/v1/system/newbro/captains`
  - 返回全部队长绩效列表
  - 查询参数最小字段：
    - `current`
    - `size`
    - `keyword`
  - 列表项最小字段：
    - `captain_user_id`
    - `captain_character_id`
    - `captain_character_name`
    - `active_player_count`
    - `historical_player_count`
    - `attributed_bounty_total`
    - `attribution_record_count`
- `GET /api/v1/system/newbro/captains/:user_id`
  - 返回单个队长详情
  - 最小返回字段：
    - `overview`
    - `players`
    - `attributions`
- `POST /api/v1/system/newbro/attribution/sync`
  - 手动触发归因同步
  - 最小返回字段：
    - `processed_count`
    - `inserted_count`
    - `skipped_count`
    - `last_wallet_journal_id`

## 路由权限矩阵

### 用户侧

- `GET /api/v1/newbro/captains`
  - 路由权限：`Login`
  - service 行为：若当前资格快照不存在或过期，则先重算；若结果不是新人，则拒绝返回队长列表
- `GET /api/v1/newbro/affiliation/me`
  - 路由权限：`Login`
  - service 行为：始终返回当前用户的新人状态；若快照不存在或过期，则先重算；若当前不是新人，可返回其当前/最近关系摘要，但不提供可选队长列表
- `POST /api/v1/newbro/affiliation/select`
  - 路由权限：`Login`
  - service 行为：若快照不存在或过期，则先重算；仅当前仍是新人时允许成功创建或切换关系

### 队长侧

- `GET /api/v1/newbro/captain/overview`
  - 路由权限：`RequireRole(captain)`
- `GET /api/v1/newbro/captain/players`
  - 路由权限：`RequireRole(captain)`
- `GET /api/v1/newbro/captain/attributions`
  - 路由权限：`RequireRole(captain)`

### 管理侧

- `GET /api/v1/system/newbro/captains`
  - 路由权限：`RequireRole(admin)`
- `GET /api/v1/system/newbro/captains/:user_id`
  - 路由权限：`RequireRole(admin)`
- `POST /api/v1/system/newbro/attribution/sync`
  - 路由权限：`RequireRole(admin)`

## 前端页面

### 新人选队长页

展示：

- 当前是否符合新人资格
- 当前生效中的队长
- 可选择的队长列表

行为：

- 若当前不满足新人资格，页面不可使用并显示原因
- 若已有当前队长，可切换到其他队长
- 用户不能选择自己作为队长；若自己具备 `captain` 角色，也不应出现在可选队长列表中

### 队长帮扶页

展示：

- 当前正在选择自己的玩家
- 历史上曾选择过自己的玩家
- 按玩家汇总的归因收益
- 归因明细列表

行为：

- 队长不能把自己招募为自己名下的新人；无论前端列表还是后端接口都应拒绝该操作

建议筛选：

- 时间范围
- 玩家
- `ref_type`
- 当前有效 / 历史关系

### 帮扶管理页

展示：

- 所有队长绩效排行/列表
- 单个队长的明细下钻
- 归因同步入口

## 后端实现分层

遵循仓库既有分层：

- `router`
- `handler`
- `service`
- `repository`
- `model`

建议新增模块：

- `server/internal/model/newbro_*.go`
- `server/internal/repository/newbro_*.go`
- `server/internal/service/newbro_*.go`
- `server/internal/handler/newbro_*.go`

前端对应：

- `static/src/views/newbro/`
- `static/src/api/newbro.ts`
- `static/src/types/api/api.d.ts`

## 前后端契约

需要同步更新：

1. 后端请求与响应结构
2. 前端 API wrapper
3. `static/src/types/api/api.d.ts`
4. 页面消费逻辑
5. 菜单/i18n 文案
6. 相关文档

## 测试与验证

### 后端测试重点

- 新人资格判定
  - 任一角色 `>= 20,000,000 SP` 时不再属于新人
  - `allow_corporations` 内角色数达到 `4` 时不再属于新人
- 新人资格持久化
  - 资格结果会被写入快照表
  - 规则版本变化后可触发重算
  - 规则放宽后，原先非新人用户可在重算后重新成为新人
- 玩家切换队长时，只保留一条有效关系
- 同一队长可被同一玩家多次重新选择
- 归因条件
  - 同星系
  - 时间差 `<= 15` 分钟
  - 仅 `bounty_prizes`
- 同一玩家钱包流水不会重复入账
- 非队长不能访问队长接口
- 非管理员不能访问管理接口

### 前端验证重点

- 菜单显示与隐藏
- 新人页的资格状态展示
- 队长页与管理页表格查询
- 类型检查通过

### 仓库验证

- `cd server && go test ./...`
- `cd server && go build ./...`
- `cd static && pnpm lint .`
- `cd static && pnpm exec vue-tsc --noEmit`

## 实施建议

建议按以下顺序进入实现计划：

1. 数据模型与迁移
2. 角色与菜单种子
3. 用户侧队长选择接口与页面
4. 归因同步服务与管理触发入口
5. 队长视图与管理员视图
6. 文档、测试、契约收尾

---
status: draft
doc_type: draft
owner: engineering
last_reviewed: 2026-06-04
source_of_truth:
  - server/internal/router/router.go
  - server/internal/repository/npc_kill.go
  - server/internal/model/esi/wallet.go
  - server/internal/repository/eve_wallet.go
  - server/internal/repository/eve_character.go
  - static/src/router/modules/dashboard.ts
  - static/src/api/dashboard.ts
  - static/src/api/sde.ts
  - docs/features/current/corporation-structures.md
---

# Dashboard 星系登记草案

> 本文是新增能力草案，不代表当前已实现行为。该草案仅覆盖旧 Vue 前端 `static/`，不覆盖 `static-react/`。

## 当前状态

- 已实现：
  - Dashboard 已有旧 Vue 路由与页面体系。
  - `captain` 职权已存在，可作为队长登记权限边界。
  - 用户绑定 EVE 角色已持久化，可用于获取队长当前绑定的所有角色。
  - ESI 钱包流水已持久化 `character_id`、`context_id`、`ref_type`、`date` 与 `amount`，可直接用于星系生产登记有效性校验。
  - SDE 已提供公开查询与模糊搜索能力，但当前 `/api/v1/sde/search` 只面向物品与人物，不支持星系搜索。
- 未实现：
  - 没有“可登记星系”配置表。
  - 没有“队长登记生产状态”记录表。
  - 没有“星系繁忙/空闲”仪表盘页面。
  - 没有针对登记记录的有效/违规校验流程与管理分析面板。

## 背景

- 当前用户无法在系统内共享“某个星系正在被哪位队长生产”的实时状态。
- 新人帮扶中的队长赏金归因服务于带新奖励结算，不是星系占用登记的数据源。
- 星系登记只需要判断队长当前绑定角色在登记时间窗内是否产生对应星系的 `bounty_prizes` 钱包流水，因此应直接查询钱包流水，不复用 `captain_bounty_attribution` 记录。
- 运营目标是提供一个低摩擦的队长登记台，让成员快速判断：
  - 哪些星系当前繁忙
  - 哪些星系空闲可用
  - 谁正在某个星系生产
  - 某次登记最终是否有真实刷怪赏金支撑
- 管理目标是让 `admin`：
  - 维护允许登记的星系列表
  - 为每个星系补充运营备注和有效性阈值
  - 查看最近登记量、有效率、违规与超时情况

## 目标

- 在旧 Vue Dashboard 新增一个独立页面 `/dashboard/galaxy-registry`。
- `admin` 可维护可登记星系配置。
- `captain` 可对空闲星系发起“生产中”登记，并设置预计结束时间。
- 系统记录实际开始与实际结束时间，并在登记结束后校验该时段内是否存在该星系的 `bounty_prizes` 收入。
- `admin` 可在同一页面查看登记记录与分析面板。

## 非目标

- 不实现 `static-react` 页面、路由、API 或类型。
- 不把该能力并入现有 `newbro/captain` 页面。
- 不支持一个星系同时被多位队长并行登记。
- 不做自动结束登记；预计结束时间仅用于展示与超时提示。
- 不把有效性阈值做成全局配置。
- 不把 ESS、任务奖励、入侵收入纳入登记有效性判定；首版只认 `bounty_prizes`。

## 范围

### 前端

- 仅覆盖旧 Vue 前端 `static/src/`。
- 在 `static/src/router/modules/dashboard.ts` 新增 Dashboard 子路由：
  - `path: 'galaxy-registry'`
  - `name: 'DashboardGalaxyRegistry'`
  - `component: '/dashboard/galaxy-registry'`
- 新增页面目录：
  - `static/src/views/dashboard/galaxy-registry`

### 后端

- 新增 `/api/v1/dashboard/galaxy-registry/*` 路由组。
- 新增独立 handler / service / repository / model。
- 新增一个后台校验任务，用于把已结束登记从 `pending` 转为 `valid` 或 `violation`。

### 文档

- 本文档作为实现草案。
- 能力落地后，需要迁移到：
  - `docs/features/current/` 下的新 feature 文档
  - `docs/api/route-index.md`

## 核心决策

### 1. 前端只做旧 Vue

- 本次能力不进入 `static-react/`。
- 所有页面、API 包装、全局类型、i18n 文案仅在 `static/` 侧新增。

### 2. 星系占用规则

- 同一星系同一时间只允许一条 `active` 登记。
- 星系只展示两类实时运营状态：
  - `busy`：存在 active 登记
  - `idle`：不存在 active 登记
- 若 active 登记超过预计结束时间未结束，页面额外标记 `overdue`，但底层仍属于 `busy`。

### 3. 结束规则

- 登记只能由队长手动结束，或由 `admin` 强制结束。
- 队长结束时写入 `actual_end_at`。
- 预计结束时间不触发自动关闭。

### 4. 有效性校验规则

- 校验对象：已结束且验证状态为 `pending` 的登记。
- 校验数据源：队长当前绑定角色的 `eve_character_wallet_journal` 钱包流水。
- 不使用 `captain_bounty_attribution`，也不生成或消费新人帮扶归因记录。
- 校验范围：登记开始与结束时间之间，该队长当前绑定的任意角色钱包流水。
- 必须同时满足：
  - `character_id IN 该队长当前绑定角色集合`
  - `context_id = system_id`
  - `ref_type = bounty_prizes`
  - `date` 落在 `actual_start_at ~ actual_end_at`
  - `amount > 0`
- 聚合后的赏金总额 `>= frozen_min_bounty_amount` 则标记 `valid`。
- 若登记结束后超过 `24` 小时仍未满足阈值，则标记 `violation`。

### 5. 最低赏金阈值

- 阈值按星系单独配置。
- 默认值为 `10,000,000 ISK`。
- 登记创建时冻结到记录中，后续配置变更不追溯影响历史判定。

## 数据模型草案

### 1. `galaxy_registry_system`

用途：管理员维护“允许登记的星系”。

建议字段：

| 字段 | 类型 | 说明 |
| --- | --- | --- |
| `id` | uint | 主键 |
| `solar_system_id` | int64 | SDE 星系 ID，唯一 |
| `solar_system_name` | string | 冻结保存的星系名 |
| `region_id` | int64 | 星域 ID |
| `region_name` | string | 星域名 |
| `constellation_id` | int64 | 星座 ID |
| `constellation_name` | string | 星座名 |
| `security` | float64 | 安全等级 |
| `note` | string | 运营备注，例如可供多少避难 |
| `min_bounty_amount` | float64 | 最低有效赏金阈值，默认 10,000,000 |
| `is_enabled` | bool | 是否启用 |
| `created_at / updated_at / deleted_at` | 标准字段 | 继承 `BaseModel` |

约束：

- `solar_system_id` 唯一。
- `min_bounty_amount >= 0`。

### 2. `galaxy_registry_entry`

用途：记录每次生产登记。

建议字段：

| 字段 | 类型 | 说明 |
| --- | --- | --- |
| `id` | uint | 主键 |
| `system_config_id` | uint | 关联 `galaxy_registry_system.id` |
| `solar_system_id` | int64 | 冻结星系 ID |
| `solar_system_name` | string | 冻结星系名 |
| `captain_user_id` | uint | 队长用户 ID |
| `captain_character_id` | int64 | 登记发起时的展示角色 ID |
| `captain_character_name` | string | 登记发起时的展示角色名 |
| `status` | string | `active` / `completed` |
| `validation_status` | string | `pending` / `valid` / `violation` |
| `expected_end_at` | time.Time | 队长填写的预计结束时间 |
| `actual_start_at` | time.Time | 实际开始时间 |
| `actual_end_at` | *time.Time | 实际结束时间 |
| `ended_by_user_id` | uint | 结束操作者 |
| `force_ended_by_admin` | bool | 是否管理员强制结束 |
| `frozen_min_bounty_amount` | float64 | 冻结阈值 |
| `validated_at` | *time.Time | 最近一次完成校验时间 |
| `validated_bounty_amount` | float64 | 校验命中的赏金总额 |
| `validated_bounty_count` | int | 命中的流水条数 |
| `violation_reason` | string | 违规原因 |
| `created_at / updated_at / deleted_at` | 标准字段 | 继承 `BaseModel` |

约束：

- 一个星系同一时间仅允许一个 `status = active` 的记录。
- `actual_end_at` 不能为空时，`status` 必须为 `completed`。
- `validation_status` 初始为 `pending`。

## 页面设计草案

## `/dashboard/galaxy-registry`

页面分三类视图，但保留在同一路由内：

- 全员可见区：
  - 星系状态总览卡片
  - 星系列表
- 队长可见区：
  - 发起登记弹窗
  - 我的登记记录
- 管理员可见区：
  - 星系配置 Tab
  - 登记记录 Tab
  - 分析面板 Tab

### 1. 星系列表

展示字段：

- 星系名
- 星域 / 星座
- 安全等级
- 备注
- 当前状态：`空闲` / `生产中` / `超时`
- 当前登记队长
- 预计结束时间
- 最低有效赏金阈值

行为：

- 若用户具备 `captain` 且该星系空闲，则展示“登记生产”按钮。
- 若当前 active 登记属于本人，则展示“结束登记”按钮。
- 若当前 active 登记不属于本人，则只读展示。

### 2. 队长登记弹窗

字段：

- 星系：从列表入口带入，不允许自由修改
- 预计结束时间：必填，且必须晚于当前时间

提交后端前校验：

- 当前用户有 `captain`
- 当前用户至少绑定一个角色
- 目标星系启用中
- 目标星系当前无 active 登记

### 3. 我的登记记录

展示字段：

- 星系
- 实际开始 / 实际结束
- 预计结束
- 状态
- 校验状态
- 命中奖金
- 违规原因

支持按状态、校验状态、日期筛选。

### 4. 管理员星系配置

能力：

- 按关键字搜索 SDE 星系并添加
- 编辑备注
- 编辑最低赏金阈值
- 启用 / 停用
- 删除配置

SDE 搜索结果需返回：

- `solar_system_id`
- `solar_system_name`
- `region_id`
- `region_name`
- `constellation_id`
- `constellation_name`
- `security`

### 5. 管理员登记记录

支持查看全部登记并过滤：

- 星系
- 队长昵称 / 人物名
- `status`
- `validation_status`
- 开始日期 / 结束日期

支持操作：

- 强制结束 active 登记

### 6. 管理员分析面板

首版输出以下聚合：

- 当前空闲星系数
- 当前繁忙星系数
- 当前超时登记数
- 最近 7 天登记数
- 最近 30 天登记数
- 最近 30 天有效率
- 最近 30 天违规数
- 最近 30 天最常被登记的星系 Top N
- 最近违规记录列表

## 接口草案

### 用户侧

#### `GET /api/v1/dashboard/galaxy-registry/systems`

用途：获取星系状态列表。

权限：

- `Login`
- 建议叠加 `menu.dashboard`

响应字段建议：

```json
{
  "summary": {
    "idle_count": 0,
    "busy_count": 0,
    "overdue_count": 0
  },
  "items": [
    {
      "system_config_id": 1,
      "solar_system_id": 30000142,
      "solar_system_name": "Jita",
      "region_name": "The Forge",
      "constellation_name": "Kimotoro",
      "security": 0.9,
      "note": "2 个避难",
      "min_bounty_amount": 10000000,
      "status": "idle",
      "active_entry": null
    }
  ]
}
```

`active_entry` 非空时返回：

- `entry_id`
- `captain_user_id`
- `captain_character_id`
- `captain_character_name`
- `expected_end_at`
- `actual_start_at`
- `is_overdue`
- `is_mine`

#### `POST /api/v1/dashboard/galaxy-registry/entries`

用途：队长发起登记。

权限：

- `RequireRole(captain)`

请求体：

```json
{
  "system_config_id": 1,
  "expected_end_at": "2026-06-04T23:30:00+08:00"
}
```

服务规则：

- 读取星系配置并检查启用状态。
- 校验当前用户至少绑定一个角色。
- 校验该星系无 active 登记。
- 写入新记录，`status=active`，`validation_status=pending`。

#### `POST /api/v1/dashboard/galaxy-registry/entries/:id/end`

用途：队长结束自己的 active 登记。

权限：

- `RequireRole(captain)`

服务规则：

- 仅允许结束属于当前队长自己的 active 记录。
- 写入 `actual_end_at`、`ended_by_user_id`。
- `status` 置为 `completed`。
- `validation_status` 保持 `pending`，等待后台校验。

#### `GET /api/v1/dashboard/galaxy-registry/my-entries`

用途：队长查看自己的登记历史。

权限：

- `RequireRole(captain)`

支持过滤：

- `status`
- `validation_status`
- `start_date`
- `end_date`
- `page`
- `page_size`

### 管理侧

#### `GET /api/v1/dashboard/galaxy-registry/admin/sde-systems`

用途：搜索 SDE 星系。

权限：

- `RequireRole(admin)`

请求参数：

- `keyword`
- `limit`

说明：

- 这是新增能力，不复用当前 `/api/v1/sde/search` 返回结构。
- 应返回真实星系元信息，而非物品/人物混合搜索结果。

#### `GET /api/v1/dashboard/galaxy-registry/admin/systems`

用途：查看已配置星系列表。

权限：

- `RequireRole(admin)`

#### `POST /api/v1/dashboard/galaxy-registry/admin/systems`

用途：新增可登记星系。

权限：

- `RequireRole(admin)`

请求体：

```json
{
  "solar_system_id": 30000142,
  "note": "2 个避难",
  "min_bounty_amount": 10000000,
  "is_enabled": true
}
```

服务规则：

- 星系基础信息必须从 SDE 查询回填，不信任前端上传名称。

#### `PUT /api/v1/dashboard/galaxy-registry/admin/systems/:id`

用途：编辑备注、阈值、启用状态。

#### `DELETE /api/v1/dashboard/galaxy-registry/admin/systems/:id`

用途：删除星系配置。

约束：

- 若存在 active 登记，删除应拒绝，避免产生孤儿记录。

#### `GET /api/v1/dashboard/galaxy-registry/admin/entries`

用途：管理员查看全部登记记录。

权限：

- `RequireRole(admin)`

过滤条件：

- `system_config_id`
- `keyword`
- `status`
- `validation_status`
- `start_date`
- `end_date`
- `page`
- `page_size`

#### `POST /api/v1/dashboard/galaxy-registry/admin/entries/:id/force-end`

用途：管理员强制结束遗留 active 登记。

服务规则：

- 仅允许结束 active 记录。
- 写入 `actual_end_at`、`ended_by_user_id`、`force_ended_by_admin=true`。

#### `GET /api/v1/dashboard/galaxy-registry/admin/analytics`

用途：管理员查看登记系统近况。

权限：

- `RequireRole(admin)`

响应建议：

- `current_snapshot`
- `recent_7d`
- `recent_30d`
- `top_systems`
- `recent_violations`

## 后台任务草案

任务名建议：

- `galaxy_registry_validation`

执行职责：

1. 拉取 `status = completed AND validation_status = pending` 的登记。
2. 对每条登记按冻结信息查询钱包流水。
3. 统计命中总额与命中条数。
4. 满足阈值则置为 `valid`。
5. 不满足阈值但距 `actual_end_at` 未满 24 小时，保留 `pending`。
6. 不满足阈值且超过 24 小时，置为 `violation`，原因写入：
   - `no_bounty_in_window`
   - 或 `bounty_below_threshold`

调度建议：

- 默认 `@every 1h`

## 权限边界

- 页面入口建议放在 Dashboard 下。
- 路由可见性：
  - `captain`、`admin`、`super_admin` 可见
- 后端才是最终权限边界：
  - 查看星系列表：登录用户
  - 登记 / 结束：`captain`
  - 配置 / 管理 / 分析：`admin`
- `super_admin` 通过 `RequireRole(admin)` 的既有兼容规则自动放行。

## 代码文件级实施清单

### 后端（Go）

| 文件 | 变更点 | 目标 |
| --- | --- | --- |
| `server/internal/model/` | 新增星系配置与登记记录模型 | 持久化 |
| `server/internal/repository/` | 新增登记模块仓储；补充星系 SDE 查询 | 数据访问 |
| `server/internal/service/` | 新增登记业务、校验任务、分析汇总 | 业务规则 |
| `server/internal/handler/` | 新增 dashboard galaxy registry handler | HTTP 接口 |
| `server/internal/router/router.go` | 注册 `/dashboard/galaxy-registry/*` 路由 | 暴露接口 |
| `server/bootstrap/db.go` | AutoMigrate 新模型 | 表结构接入 |
| `server/jobs/` 或现有任务注册点 | 注册校验任务 | 自动校验 |

### 前端（旧 Vue）

| 文件 | 变更点 | 目标 |
| --- | --- | --- |
| `static/src/router/modules/dashboard.ts` | 新增菜单路由 | 页面入口 |
| `static/src/views/dashboard/galaxy-registry/index.vue` | 新增页面 | UI 主体 |
| `static/src/api/galaxy-registry.ts` | 新增 API 包装 | 前端请求 |
| `static/src/types/api/api.d.ts` | 新增 `Api.Dashboard.GalaxyRegistry*` 类型 | 契约同步 |
| `static/src/locales/langs/zh.json` | 新增中文文案 | i18n |
| `static/src/locales/langs/en.json` | 新增英文文案 | i18n |

### 文档

| 文件 | 变更点 | 目标 |
| --- | --- | --- |
| `docs/api/route-index.md` | 补充新接口 | API 索引 |
| `docs/features/current/` 新文档 | 实现完成后转正 | 功能文档 |

## 测试计划

### 后端

- 模型约束测试：
  - 同一星系不能存在两条 active 登记
  - 删除有 active 记录的星系配置应失败
- 服务层测试：
- 未绑定任何角色的队长不能登记
  - 非队长不能登记
  - 非本人不能结束别人的登记
  - 结束后记录进入 `pending`
- 命中 `绑定任意角色 + bounty_prizes + system_id + time window` 且总额达阈值时标记 `valid`
  - 超过 24 小时仍未达阈值时标记 `violation`
  - 阈值冻结后不受后续星系配置修改影响
- 仓储测试：
  - SDE 星系搜索返回星系元信息
  - 管理端列表过滤与排序稳定

### 前端

- 页面存在性测试：
  - Dashboard 路由包含 `galaxy-registry`
  - 页面包含星系列表、队长区、管理员区
- API wrapper 测试：
  - 路径、方法、参数映射正确
- 交互测试：
  - 空闲星系才允许登记
  - 本人 active 记录才显示结束按钮
  - 管理员额外可见配置与分析 Tab

### 最低验证命令

```bash
cd server && go test ./...
cd server && go build ./...
cd static && pnpm lint .
cd static && pnpm exec vue-tsc --noEmit
cd static && pnpm test:unit
```

## 设计理由

- 决策：将能力放在 Dashboard 独立页面，而不是并入 `newbro/captain`。
- 理由：
  - 该能力服务于“星系占用可视化”，不是新人帮扶流程本身。
  - Dashboard 更适合承载全局运营状态与管理员分析面板。

- 决策：有效性判定只认 `bounty_prizes`。
- 理由：
  - 登记系统只判断“该队长是否在登记星系实际生产”，不参与新人帮扶奖励结算。
  - 直接查询队长绑定角色的钱包流水即可满足判定需求，避免把登记系统耦合到 `captain_bounty_attribution`。
  - `context_id = system_id` 是 ESI 钱包流水中表达星系来源的字段，适合登记有效性校验。
  - 首版保持单一口径，避免把 ESS 等其他收入混入生产登记的判断。
  - 同时放宽到“该队长绑定的任意角色”，以适配一个队长多角色轮换生产的真实使用场景。

- 决策：每星系单独配置最低赏金阈值。
- 理由：
  - 不同星系产能差异大，统一全局阈值误差更大。
  - 星系备注和阈值都属于运营配置，适合放在同一个管理面板。

- 决策：只允许单队长占用。
- 理由：
  - 用户目标是快速看到“繁忙 / 空闲”。
  - 单占用规则能把 UI、数据库约束、登记冲突处理都保持简单稳定。

- 取舍 / 未采用方案：
  - 未采用“预计结束时间自动结束”：会把实际结束时间混淆成计划值，不利于后续校验。
  - 未采用“按备注容量允许多人并行”：需要新增容量字段与冲突算法，首版复杂度过高。
  - 未采用“在 React 和 Vue 双端同时实现”：用户已明确要求只写旧 Vue。

- 如果落地，必须迁移到的权威文档：
  - `docs/features/current/` 新增 feature 文档
  - `docs/api/route-index.md`
  - 如涉及权限与菜单模型调整，再评估是否补 `docs/architecture/routing-and-menus.md`

## 未决问题

- 是否需要对 active 登记增加“最长允许时长”硬限制。
  - 否，没有最长允许时长限制
- 管理员分析面板是否需要支持日期筛选，还是首版固定展示最近 7 天 / 30 天。
  - 需要支持日期筛选
- 是否需要为该功能新增独立军团能力键，而不只依赖 `menu.dashboard + role`。
  - 不动该部分

## 明确声明

- 本文档是提案，不代表当前已实现行为。
- 本文档不能覆盖 `docs/ai/repo-rules.md`、`docs/architecture/`、`docs/api/`、`docs/features/current/`。
- 实现时必须保持后端分层：`router -> handler -> service -> repository -> model`。
- 本草案默认所有前端文案都需要同步写入 `zh.json` 与 `en.json`。

## 升级路径

- 第一步：按本文草案完成后端模型、接口、任务与旧 Vue 页面。
- 第二步：在实现完成后，把“当前行为”“入口”“权限边界”“关键不变量”迁移到 `docs/features/current/` 新文档。
- 第三步：把新增路由补进 `docs/api/route-index.md`。
- 第四步：如果后续再做 React 迁移，应单独立项，不在本草案中隐含双端一致实现要求。

## TODO

### 进行中

- [ ] 后端模型：新增 `GalaxyRegistrySystem` 与 `GalaxyRegistryEntry`，冻结星系、队长、时间和校验字段。
- [ ] 数据库迁移：接入 AutoMigrate，并补充“每个星系最多一个 active 登记”的部分唯一索引。
- [ ] 仓储基础：新增星系配置、登记记录、SDE 星系元信息查询与登记校验所需的钱包流水聚合。
- [ ] 服务层：实现列表、登记、结束、管理配置、强制结束、分析与后台校验业务规则。
- [ ] Handler：实现用户侧与管理侧 HTTP 接口，并保持参数校验在传输层。
- [ ] Router：注册 `/api/v1/dashboard/galaxy-registry/*` 路由组并接入 `menu.dashboard`、`captain`、`admin` 权限。

### 待办

- [ ] 后台任务：注册 `galaxy_registry_validation`，默认每小时校验已结束的 pending 登记。
- [ ] 前端 API：新增 `static/src/api/galaxy-registry.ts`。
- [ ] 前端类型：在 `Api.Dashboard` 下新增星系登记契约类型。
- [ ] 前端路由：在 Dashboard 下新增 `galaxy-registry` 子路由。
- [ ] 前端页面：实现星系列表、队长登记/结束、我的记录、管理员配置/记录/分析 Tab。
- [ ] 前端 i18n：同步补齐 `zh.json` 与 `en.json` 文案。
- [ ] 测试：补服务规则、仓储查询、路由/API wrapper 和页面存在性测试。
- [ ] 文档转正：实现完成后新增 `docs/features/current/dashboard-galaxy-registry.md` 并补充 `docs/api/route-index.md`。

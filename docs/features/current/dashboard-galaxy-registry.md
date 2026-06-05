---
status: active
doc_type: feature
owner: engineering
last_reviewed: 2026-06-04
source_of_truth:
  - server/internal/router/router.go
  - server/internal/handler/galaxy_registry.go
  - server/internal/service/galaxy_registry.go
  - server/internal/repository/galaxy_registry.go
  - server/internal/repository/sde_galaxy_registry.go
  - server/internal/model/galaxy_registry.go
  - static/src/router/modules/dashboard.ts
  - static/src/api/galaxy-registry.ts
  - static/src/views/dashboard/galaxy-registry/index.vue
---

# Dashboard 星系登记

## 当前能力

- Dashboard `galaxy-registry` 页面对 `captain` / `admin` / `super_admin` 可见，并拆为 `当前状态`、`队长登记与查询`、`管理员登记与查询` 三个主 Tab
- 登录用户可查看启用星系配置、当前空闲/繁忙/超时状态，以及正在生产的队长信息
- `captain` 可对空闲且启用中的星系发起登记、手动结束自己的 active 登记，并在生产过程中修改预计结束时间
- 每个星系同一时间只允许一条 active 登记
- 队长点击结束生产时，服务层会同步拉取该队长当前绑定角色的钱包 ESI，并立刻写回最终校验结果
- 校验口径只认 `eve_character_wallet_journal` 中满足以下条件的 `bounty_prizes`：
  - `character_id` 属于该队长当前仍绑定的任意角色
  - `context_id = solar_system_id`
  - `date` 落在 `actual_start_at ~ actual_end_at`
  - `amount > 0`
- 命中奖金总额 `>= frozen_min_bounty_amount` 时标记 `valid`
- 未达到阈值时立即标记 `violation`；无流水时为 `no_bounty_in_window`，有流水但不足阈值时为 `bounty_below_threshold`
- `admin` 可搜索 SDE 星系、维护可登记星系配置、使用全局保存提交新增或修改、查看全量登记记录、强制结束 active 登记、人工覆盖校验结果，并查看统计面板
- `admin` 还可对已结束登记重新触发校验；该操作会重新拉取当前绑定角色钱包 ESI，并用最新重算结果覆盖现有校验结果
- 旧的 `galaxy_registry_validation` 定时任务可保留为兜底补偿，但不再是正常业务流程的依赖

## 入口

### 前端页面

- `static/src/views/dashboard/galaxy-registry`
- `static/src/router/modules/dashboard.ts` 中的 `DashboardGalaxyRegistry`

### 后端路由

- `GET /api/v1/dashboard/galaxy-registry/systems`
- `POST /api/v1/dashboard/galaxy-registry/entries`
- `POST /api/v1/dashboard/galaxy-registry/entries/:id/end`
- `PUT /api/v1/dashboard/galaxy-registry/entries/:id/expected-end-at`
- `GET /api/v1/dashboard/galaxy-registry/my-entries`
- `GET /api/v1/dashboard/galaxy-registry/admin/sde-systems`
- `GET /api/v1/dashboard/galaxy-registry/admin/systems`
- `POST /api/v1/dashboard/galaxy-registry/admin/systems`
- `PUT /api/v1/dashboard/galaxy-registry/admin/systems/:id`
- `DELETE /api/v1/dashboard/galaxy-registry/admin/systems/:id`
- `GET /api/v1/dashboard/galaxy-registry/admin/entries`
- `POST /api/v1/dashboard/galaxy-registry/admin/entries/:id/force-end`
- `POST /api/v1/dashboard/galaxy-registry/admin/entries/:id/revalidate`
- `PUT /api/v1/dashboard/galaxy-registry/admin/entries/:id/validation`
- `GET /api/v1/dashboard/galaxy-registry/admin/analytics`

## 权限边界

- 页面路由仅对 `captain` / `admin` / `super_admin` 可见
- 所有接口都要求登录用户且具备 `menu.dashboard`
- 星系列表接口对任意登录产品用户开放
- 登记、结束本人登记、修改本人预计结束时间、查看本人记录要求 `RequireRole(captain)`
- 配置、全量记录、强制结束、重新触发校验、人工修改校验结果、分析面板要求 `RequireRole(admin)`
- 没有新增独立 capability；继续复用 Dashboard 菜单能力与角色边界

## 关键不变量

- `galaxy_registry_entry` 通过部分唯一索引保证每个 `system_config_id` 最多一条 active 记录
- `frozen_min_bounty_amount` 在创建登记时冻结，后续配置变更不追溯历史记录
- 预计结束时间只用于展示和超时标记，不会自动结束登记
- 删除星系配置前必须确认不存在 active 登记
- 校验时读取的是“当前绑定角色集合”，不是登记时快照的角色集合
- 管理员重新触发校验会直接覆盖原有 `validation_status`、`validated_*`、`validated_at`、`violation_reason`
- 结束登记后的校验时间窗仍然只看 `actual_start_at ~ actual_end_at`，不会使用 `expected_end_at`
- 超时状态是 UI 层派生状态，底层登记状态仍然是 active

## 主要代码文件

- `server/internal/model/galaxy_registry.go`
- `server/internal/repository/galaxy_registry.go`
- `server/internal/repository/sde_galaxy_registry.go`
- `server/internal/service/galaxy_registry.go`
- `server/internal/handler/galaxy_registry.go`
- `server/internal/router/router.go`
- `static/src/api/galaxy-registry.ts`
- `static/src/router/modules/dashboard.ts`
- `static/src/views/dashboard/galaxy-registry/index.vue`

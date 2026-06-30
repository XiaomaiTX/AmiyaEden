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
  - static/src/hooks/galaxy-registry/useGalaxyRegistryTimeoutNotification.ts
---

# Dashboard 星系登记

## 当前能力

- Dashboard `galaxy-registry` 页面对 `captain` / `admin` / `super_admin` 可见，并拆为 `当前状态`、`队长登记与查询`、`管理员登记与查询` 三个主 Tab
- 登录用户可查看启用星系配置、当前空闲/繁忙/超时状态，以及正在生产的队长信息
- `captain` 可对空闲且启用中的星系发起登记、手动结束自己的 active 登记，并在生产过程中修改预计结束时间；单次登记最长 2 小时
- 每个星系同一时间只允许一条 active 登记
- 队长点击结束生产时，服务层只结束登记并释放星系，校验保持 `pending`
- 满 2 小时仍未结束的 active 登记会由后台任务自动结束，`actual_end_at` 固定写为 `actual_start_at + 2h`
- 用户打开 AmiyaEden 主布局页面时，前端会监控本人 active 登记；登记达到 2 小时仍未结束时，若浏览器通知权限已授权，则发送一次浏览器系统通知
- 若浏览器通知权限仍为默认状态，前端会在检测到本人 active 登记时弹出一次站内确认，引导用户授权；用户取消后，同一浏览器 profile 不再反复提示
- 登记结算由后台任务统一处理：先通过 ESI 队列刷新队长当前绑定角色的 `character_wallet`，再读取本地钱包流水写回最终校验结果
- 部分角色钱包刷新失败时，后台任务会继续用其余成功刷新的角色尝试结算
- 若全部绑定角色钱包刷新失败，则登记保持 `pending`，等待下次后台任务或管理员重新提交结算
- 校验口径只认 `eve_character_wallet_journal` 中满足以下条件的 `bounty_prizes`：
  - `character_id` 属于该队长当前仍绑定的任意角色
  - `context_id = solar_system_id`
  - `date` 落在 `actual_start_at ~ actual_end_at`
  - `amount > 0`
- 命中奖金总额 `>= frozen_min_bounty_amount` 时标记 `valid`
- 未达到阈值时立即标记 `violation`；无流水时为 `no_bounty_in_window`，有流水但不足阈值时为 `bounty_below_threshold`
- `admin` 可搜索 SDE 星系、维护可登记星系配置、使用全局保存提交新增或修改、查看全量登记记录、强制结束 active 登记、人工覆盖校验结果，并查看统计面板
- `admin` 还可对已结束登记重新提交后台结算；该操作会把记录重置为 `pending`，后续由后台任务通过 ESI 队列刷新钱包并覆盖现有校验结果
- `galaxy_registry_validation` 定时任务是星系登记自动下线与异步结算入口，默认每 5 分钟执行一次

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
- 预计结束时间只用于展示和超时标记；强制下线统一按 `actual_start_at + 2h` 判断
- 删除星系配置前必须确认不存在 active 登记
- 校验时读取的是“当前绑定角色集合”，不是登记时快照的角色集合
- 管理员重新触发校验会先重置为 `pending`，后台结算完成后覆盖原有 `validation_status`、`validated_*`、`validated_at`、`violation_reason`
- 结束登记后的校验时间窗仍然只看 `actual_start_at ~ actual_end_at`，不会使用 `expected_end_at`
- active 登记超过 2 小时后会被后台任务结束；任务未执行前的超时状态仍是 UI 层派生状态
- 浏览器系统通知只在站点已打开时工作，不提供浏览器关闭后的 Push 推送；每个 `entry_id` 在同一浏览器 profile 中只通知一次

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
- `static/src/hooks/galaxy-registry/useGalaxyRegistryTimeoutNotification.ts`
- `static/src/hooks/galaxy-registry/galaxyRegistryTimeoutNotification.ts`

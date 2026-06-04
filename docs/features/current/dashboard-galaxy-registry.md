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
  - server/jobs/galaxy_registry_validation.go
  - static/src/router/modules/dashboard.ts
  - static/src/api/galaxy-registry.ts
  - static/src/views/dashboard/galaxy-registry/index.vue
---

# Dashboard 星系登记

## 当前能力

- 旧 Vue Dashboard 新增 `galaxy-registry` 页面，入口对 `captain` / `admin` / `super_admin` 可见
- 登录用户可查看启用与未启用星系配置、当前空闲/繁忙/超时状态，以及正在生产的队长信息
- `captain` 可对空闲且启用中的星系发起登记，并手动结束自己的 active 登记
- 每个星系同一时间只允许一条 active 登记
- 登记结束后会进入 `pending` 校验状态，后台任务 `galaxy_registry_validation` 每小时检查一次
- 校验口径只认 `eve_character_wallet_journal` 中满足以下条件的 `bounty_prizes`：
  - `character_id` 属于该队长当前仍绑定的任意角色
  - `context_id = solar_system_id`
  - `date` 落在 `actual_start_at ~ actual_end_at`
  - `amount > 0`
- 命中奖金总额 `>= frozen_min_bounty_amount` 时标记 `valid`
- 若登记结束超过 24 小时仍未达到阈值，则标记 `violation`
- `admin` 可搜索 SDE 星系、维护可登记星系配置、查看全量登记记录、强制结束 active 登记，并查看基础分析面板

## 入口

### 前端页面

- `static/src/views/dashboard/galaxy-registry`
- `static/src/router/modules/dashboard.ts` 中的 `DashboardGalaxyRegistry`

### 后端路由

- `GET /api/v1/dashboard/galaxy-registry/systems`
- `POST /api/v1/dashboard/galaxy-registry/entries`
- `POST /api/v1/dashboard/galaxy-registry/entries/:id/end`
- `GET /api/v1/dashboard/galaxy-registry/my-entries`
- `GET /api/v1/dashboard/galaxy-registry/admin/sde-systems`
- `GET /api/v1/dashboard/galaxy-registry/admin/systems`
- `POST /api/v1/dashboard/galaxy-registry/admin/systems`
- `PUT /api/v1/dashboard/galaxy-registry/admin/systems/:id`
- `DELETE /api/v1/dashboard/galaxy-registry/admin/systems/:id`
- `GET /api/v1/dashboard/galaxy-registry/admin/entries`
- `POST /api/v1/dashboard/galaxy-registry/admin/entries/:id/force-end`
- `GET /api/v1/dashboard/galaxy-registry/admin/analytics`

## 权限边界

- 页面路由仅对 `captain` / `admin` / `super_admin` 可见
- 所有接口都要求登录用户且具备 `menu.dashboard`
- 星系列表接口对任意登录产品用户开放
- 登记、结束本人登记、查看本人记录要求 `RequireRole(captain)`
- 配置、全量记录、强制结束、分析面板要求 `RequireRole(admin)`
- 没有新增独立 capability；继续复用 Dashboard 菜单能力与角色边界

## 关键不变量

- `galaxy_registry_entry` 通过部分唯一索引保证每个 `system_config_id` 最多一条 active 记录
- `frozen_min_bounty_amount` 在创建登记时冻结，后续配置变更不追溯历史记录
- 预计结束时间只用于展示和超时标记，不会自动结束登记
- 删除星系配置前必须确认不存在 active 登记
- 校验时读取的是“当前绑定角色集合”，不是登记时快照的角色集合
- 超时状态是 UI 层派生状态，底层登记状态仍然是 active

## 主要代码文件

- `server/internal/model/galaxy_registry.go`
- `server/internal/repository/galaxy_registry.go`
- `server/internal/repository/sde_galaxy_registry.go`
- `server/internal/service/galaxy_registry.go`
- `server/internal/handler/galaxy_registry.go`
- `server/internal/router/router.go`
- `server/jobs/galaxy_registry_validation.go`
- `static/src/api/galaxy-registry.ts`
- `static/src/router/modules/dashboard.ts`
- `static/src/views/dashboard/galaxy-registry/index.vue`

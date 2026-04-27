---
status: active
doc_type: feature
owner: engineering
last_reviewed: 2026-04-27
source_of_truth:
  - server/internal/router/router.go
  - server/internal/handler/corporation_structure.go
  - server/internal/service/corporation_structure.go
  - server/internal/repository/corporation_structure.go
  - server/internal/model/sys_config.go
  - server/internal/service/badge.go
  - static/src/router/modules/dashboard.ts
  - static/src/api/corporation-structures.ts
  - static/src/views/dashboard/corporation-structures
  - static/src/store/modules/badge.helpers.ts
---

# 军团建筑管理

## 当前能力

- Dashboard 里的 `corporation-structures` 页面面向 `admin` / `super_admin`
- 页面使用 `list` / `settings` 两个 tab：列表页查看军团建筑快照，设置页维护授权映射与阈值
- 列表页支持按军团、关键词、星系、状态组、燃料区间、安全等级、类型、服务、增强计时筛选，并支持分页与排序
- 设置页可以为每个可管理军团绑定一个已授权的 Director 人物，并设置燃料与计时器提醒阈值
- 刷新按钮会把单个军团的结构刷新任务异步丢进后台任务系统，不阻塞当前请求
- 导航徽章 `corporation_structures_attention` 会在 `admin` / `super_admin` 的导航中显示需要关注的建筑数量
- 同步过程会清理 ESI 不再返回的旧结构；当 ESI 返回空列表时，会清空对应军团的结构记录，避免陈旧快照残留

## 入口

### 前端页面

- `static/src/views/dashboard/corporation-structures`
- `static/src/router/modules/dashboard.ts` 中的 `DashboardCorporationStructures`

### 后端路由

- `GET /api/v1/dashboard/corporation-structures/settings`
- `PUT /api/v1/dashboard/corporation-structures/settings/authorizations`
- `GET /api/v1/dashboard/corporation-structures/filter-options`
- `POST /api/v1/dashboard/corporation-structures/list`
- `POST /api/v1/dashboard/corporation-structures/run-task`

### 关联展示

- `GET /api/v1/badge-counts` 返回 `corporation_structures_attention`

## 权限边界

- 前端路由只对 `admin` / `super_admin` 可见
- 所有后端接口都要求 `RequireRole(admin)`
- 只有被系统判定为当前可管理军团的 Director 人物才能写入授权映射
- `0` 天阈值表示关闭对应提醒；前端设置会把它当作显式关闭处理
- `run-task` 只触发当前军团的后台 ESI 刷新，不暴露通用 ESI 任务入口

## 关键不变量

- 授权映射以 `system_config.dashboard.corporation_structures_authorizations` 为准，页面只允许在当前可管理军团与其 Director 候选集之间绑定
- 结构列表与提醒统计都只针对当前用户可管理军团集合
- `corporation_structures_attention` 只统计去重后的 `corp_id:structure_id`，燃料与计时器任一条件命中即可计入
- `run-task` 返回成功只代表任务已入队，不代表 ESI 刷新已完成
- 后台刷新失败不应通过页面同步重试成阻塞流程，重试应由用户再次触发
- 列表与筛选选项都基于当前快照，不会在页面内实时回源 ESI

## 主要代码文件

- `server/internal/handler/corporation_structure.go`
- `server/internal/service/corporation_structure.go`
- `server/internal/repository/corporation_structure.go`
- `server/internal/model/sys_config.go`
- `server/internal/service/badge.go`
- `server/internal/router/router.go`
- `static/src/api/corporation-structures.ts`
- `static/src/router/modules/dashboard.ts`
- `static/src/views/dashboard/corporation-structures/index.vue`
- `static/src/store/modules/badge.helpers.ts`

---
status: active
doc_type: feature
owner: engineering
last_reviewed: 2026-09-04
source_of_truth:
  - server/internal/router/router.go
  - server/internal/handler/corporation_structure.go
  - server/internal/service/corporation_structure.go
  - server/internal/service/structure_fuel_rate.go
  - server/internal/service/structure_service_module.go
  - server/internal/service/structure_service_catalog.go
  - server/internal/repository/corporation_structure.go
  - server/internal/repository/structure_service_fuel_rate.go
  - server/internal/repository/structure_service_activity_candidate.go
  - server/internal/model/structure_service_fuel_rate.go
  - server/internal/model/structure_service_activity_candidate.go
  - server/internal/model/sys_config.go
  - server/internal/service/badge.go
  - server/pkg/eve/esi/task_corporation_structures.go
  - server/jobs/structure_fuel_rate_sync.go
  - static/src/router/modules/dashboard.ts
  - static/src/api/corporation-structures.ts
  - static/src/views/dashboard/corporation-structures
  - static/src/store/modules/badge.helpers.ts
  - static-react/src/api/corporation-structures.ts
  - static-react/src/pages/dashboard-corporation-structures-page.tsx
  - static-react/src/pages/dashboard-fuel-officer-structures-page.tsx
---

# 军团建筑管理

## 当前能力

- Dashboard 里的 `corporation-structures` 页面面向 `admin` / `super_admin`
- 页面使用 `list` / `settings` 两个 tab：列表页查看军团建筑快照，设置页维护授权映射与阈值
- 列表页支持按军团、关键词、星域、星系、状态组、燃料区间、安全等级、类型、服务、增强计时筛选，并支持分页与排序
- 列表页会展示每个建筑当前指派的燃料官；未指派时显示“未分配”
- 列表页“服务”列只显示服务数量；点击数量打开弹窗查看该建筑的服务名与状态（状态本地化）
- 设置页可以为每个可管理军团绑定一个已授权的 Director 人物，并设置燃料与计时器提醒阈值
- 设置页以显式开关控制 QQ 建筑预警，并维护专用 QQ 群号列表；每行填写一个群号。启用时必须至少配置一个群号，关闭不会影响页面徽章提醒
- QQ 建筑预警按剩余时间从短到长汇总；每条展示建筑所属星域和建筑类型，但不重复附加星系名
- 拥有 `system.task.run` capability 的管理员可在设置页手动触发一次 QQ 建筑预警巡查；该入口复用任务管理器的运行锁、执行历史与队列投递语义
- 支持管理员将建筑指派给 `fuel_officer`（燃料官）系统用户（按 `user_id` 分配，不按角色分配），并配置每建筑每月伏羲币工资单价
- 支持管理员按月份手动批量发放燃料官工资（按当前指派建筑数量结算）
- `fuel_officer` 可通过专用接口查看自己被指派建筑的燃料剩余与增强计时
- 定时任务 `corporation_structure_alert_scan` 默认每小时只扫描当前建筑快照：燃料或增强状态计时首次进入已配置阈值时，分别汇总后通过 QQ 群治理 OneBot 队列投递到每个目标群；恢复后再次进入阈值才会重发
- 设置页支持为军团选择“无（不启用该军团建筑仪表盘）”，等价于 `character_id=0`，该军团不会参与军团建筑 ESI 刷新
- 刷新按钮会把单个军团的结构刷新任务异步丢进后台任务系统，不阻塞当前请求
- 导航徽章 `corporation_structures_attention` 会在 `admin` / `super_admin` 的导航中显示需要关注的建筑数量
- 同步过程会清理 ESI 不再返回的旧结构；当 ESI 返回空列表时，会清空对应军团的结构记录，避免陈旧快照残留
- 列表页与燃料官个人页额外展示两个燃料消耗估算列：
  - **预计每小时消耗（燃料块）**：基于建筑在线服务模块与建筑分组折扣系数估算的每小时燃料块消耗
  - **到月底需补充（燃料块）**：基于 `fuel_expires` 估算到「耗尽所在自然月（EVE UTC）月底」还需补充的燃料块数；数值后带 `+N` 徽标时，N 为耗尽月距当前月的整月差（UTC 口径，即还能再烧几个月），燃料在当前月内耗尽（N=0）或无估算时不显示徽标
  - 两列均支持 `sort_by` 服务端排序（`fuel_per_hour` / `fuel_to_month_end`）；估算为 `null`（无估算、无在线服务或无未来到期时间）的行在升序与降序两个方向下都排在列表末尾。既有排序键（如 `fuel_remaining_hours` 降序时 `null` 排前）行为保持不变
- 服务模块燃料率（每小时燃料块）由独立定时任务 `structure_fuel_rate_sync` 每 10 天从 ESI `/universe/types/{id}/` 的 dogma 属性 2109 同步到 `structure_service_fuel_rate` 表；ESI 拉取失败时沿用硬编码默认值，DB 记录覆盖默认表（DB 缺失的服务继续用默认值）
- 折扣系数按建筑分组（`invTypes.groupID`）判定：堡垒（1657）市场/克隆 −25%、工业综合体（1404）制造/研究/发明 −25%、精炼厂（1406）再处理/反应 Athanor −20%/其余 −25%；分组缺失或未知时按无折扣计算
- 燃料估算仅在建筑列表与燃料官列表加载；指派管理列表不做估算（不查询燃料率与 SDE 分组）

## 燃料消耗估算

- 燃料率以公司资产中建筑 `ServiceSlot0`–`ServiceSlot7` 的服务模块 `type_id` 为唯一身份；建筑接口的 `services[].name` 只用于判断活动是否在线。
- 系统服务目录维护模块 `type_id`、dogma 属性 2109 的每小时燃料率及折扣类别；`structure_fuel_rate_sync` 仅按模块 `type_id` 同步 dogma 值。启动迁移会补齐目录、修正历史错误的月球钻井/反应器映射，但不会覆盖已同步的匹配 dogma 值。
- 活动目录是“活动名 → 候选模块 `type_id[]`”的多对多关系。估算时把候选集与该建筑实际安装模块相交：恰好一个匹配才计入；零匹配为模块不一致；多个匹配或同类型多实例为模块歧义；未映射为活动待配置。
- 系统已验证映射由程序托管，管理员不能覆盖；管理员只能为新观察到的未知活动添加一个或多个已验证候选模块。服务目录会显示原始活动名、建筑与已安装模块 Type ID，避免无依据猜测。
- 同一物理模块对应多个在线活动时按模块去重，只计算一次；任一在线活动无法唯一识别时，`fuel_per_hour` 和 `fuel_to_month_end` 均返回 `null`，绝不返回部分数值。
- 精确识别需要被选为建筑 Director 的人物额外授权可选 scope `esi-assets.read_corporation_assets.v1`。该授权按并集持久化在人物 scopes 快照中，普通重新登录不再丢失，且重新登录时后端会自动引导补授缺失的该类可选 scope（详见认证文档不变量）。
- 当前 token 链缺少该 scope（ESI 返回 403）时基础建筑同步仍完成，服务模块标记为未知（`service_modules_known=false`），燃料估算状态为 `authorization_required`；军团击杀邮件任务遇 403 按跳过处理，不视为任务失败。
- 计算模型：每小时消耗 = Σ(唯一已识别在线模块的有效率)，有效率 = 模块燃料率 × 建筑分组系数；建筑本身无基础消耗。
- 月底补料：目标 = `fuel_expires` 所在自然月月底（EVE UTC）；blocks = ceil((月底 − fuel_expires) × 每小时消耗)；`fuel_expires` 为空/已过期/rate≤0 时该字段为 `null`
- 不完整估算：`fuel_estimate_incomplete=true` 并在 `fuel_unknown_services` 返回造成失败的原始活动名；状态可为 `authorization_required`、`activity_mapping_required`、`module_mismatch`、`ambiguous_module` 或 `rate_unavailable`。前端列表展示对应本地化状态而非部分数值。

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
- `GET /api/v1/dashboard/corporation-structures/assignments`
- `PUT /api/v1/dashboard/corporation-structures/assignments`
- `GET /api/v1/dashboard/corporation-structures/fuel-salary-settings`
- `PUT /api/v1/dashboard/corporation-structures/fuel-salary-settings`
- `POST /api/v1/dashboard/corporation-structures/fuel-salary-payouts/run`
- `POST /api/v1/dashboard/corporation-structures/my-assigned-list`
- `GET /api/v1/dashboard/corporation-structures/service-catalog`
- `PUT /api/v1/dashboard/corporation-structures/service-catalog`

### 关联展示

- `GET /api/v1/badge-counts` 返回 `corporation_structures_attention`

## 权限边界

- 前端路由只对 `admin` / `super_admin` 可见
- 所有后端接口都要求 `RequireRole(admin)`
- 燃料官个人列表接口要求 `RequireRole(fuel_officer)`（`super_admin` 同样可通过）
- 只有被系统判定为当前可管理军团的 Director 人物才能写入授权映射
- `0` 天阈值表示关闭对应提醒；前端设置会把它当作显式关闭处理
- `run-task` 只触发当前军团的后台 ESI 刷新，不暴露通用 ESI 任务入口
- ESI 队列中的 `corporation_structures` 子任务刷新间隔为 6 小时（active）/ 1 天（inactive）
- QQ 预警不主动刷新 ESI，快照的新鲜度仍由上述 ESI 队列负责；QQ 投递沿用 OneBot 连接校验、限流、重试和死信处理

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
- `server/internal/service/structure_fuel_rate.go`
- `server/internal/service/structure_service_module.go`
- `server/internal/service/structure_service_catalog.go`
- `server/internal/repository/corporation_structure.go`
- `server/internal/repository/structure_service_fuel_rate.go`
- `server/internal/repository/structure_service_activity_candidate.go`
- `server/internal/model/structure_service_fuel_rate.go`
- `server/internal/model/structure_service_activity_candidate.go`
- `server/internal/model/sys_config.go`
- `server/internal/service/badge.go`
- `server/internal/router/router.go`
- `server/jobs/structure_fuel_rate_sync.go`
- `static/src/api/corporation-structures.ts`
- `static/src/router/modules/dashboard.ts`
- `static/src/views/dashboard/corporation-structures/index.vue`
- `static/src/store/modules/badge.helpers.ts`


## 关键不变量（2026-05-11）

- 当军团在设置页被显式禁用（`character_id=0`）并保存设置时，后端必须在同一更新流程里清空该军团在 `corp_structure` 中的所有记录，避免历史快照继续被展示或统计。

## 前端实现映射（迁移期）

- Vue 当前实现位于 `static/src`。
- React 已承接 `/dashboard/corporation-structures`，对应 API 和页面状态以 React 迁移基线为准。
- React 列表表格支持与 Vue 对齐的服务端排序表头（星系、名称、类型、燃料剩余、每小时消耗、到月底需补充、增强小时、结束时间、更新时间）。
- `fuel-officer-structures` 已由 Vue 与 React 双端承接；React 使用共享 `DataTable` 和同一 `my-assigned-list` 契约。

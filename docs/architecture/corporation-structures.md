---
status: active
doc_type: architecture
owner: engineering
last_reviewed: 2026-04-27
source_of_truth:
  - server/internal/handler/corporation_structure.go
  - server/internal/repository/corporation_structure.go
  - server/internal/service/corporation_structure.go
  - server/internal/model/sys_config.go
  - server/internal/service/badge.go
  - server/internal/router/router.go
  - server/jobs
  - server/pkg/background/manager.go
  - static/src/router/modules/dashboard.ts
  - static/src/api/corporation-structures.ts
  - static/src/views/dashboard/corporation-structures
---

# 军团建筑管理架构

## 目的

本文件记录军团建筑管理模块的当前实现方式与长期约束，避免页面交互、授权映射、缓存策略和后台刷新生命周期在后续变更中彼此漂移。

## 当前实现

### 路由与页面

- 模块入口挂在 Dashboard 下，前端静态路由名是 `DashboardCorporationStructures`
- 页面通过 query 参数在 `list` / `settings` 两个 tab 间切换
- 路由与页面权限都只面向 `admin` / `super_admin`

### 设置与授权

- `GetSettings` 先根据允许军团与当前人物的 `Director` corp role 计算可管理范围，再读取已保存的授权映射与阈值配置
- `UpdateAuthorizations` 只允许在“当前可管理军团 + 该军团 Director 候选人物”的交集内写入绑定
- 授权映射和阈值都写入 `system_config`，不会单独拆成新表

### 列表与筛选

- `ListStructures` 基于当前可管理军团读取结构快照，再补充系统名称、区域与安全等级等元数据
- 列表的排序、分页和筛选都在服务层完成，页面不会把复杂筛选逻辑拆回到前端
- `GetFilterOptions` 不是静态枚举，而是从当前快照派生可选系统、类型和服务集合

### 刷新与关停

- `RunTask` 先解析当前军团对应的授权 Director 人物，再把单军团结构刷新任务交给共享后台任务管理器
- 真正执行时走 ESI 队列里的 `corporation_structures` 任务类型
- 这个入口的语义是“已入队”，不是“已刷新完成”
- 因为这条路径会跨出当前请求生命周期，所以必须进入 `background.Manager` 的受跟踪生命周期，不能退化成裸 goroutine

### 告警计数

- `CountAttentionStructures` 与设置页共享同一可管理军团范围和阈值配置
- 告警命中条件是燃料剩余小时数落入燃料阈值，或增强计时器落入计时器窗口
- 统计结果按 `corp_id:structure_id` 去重，避免同一结构被重复计数

### 同步清理

- 同步逻辑会删除 ESI 未返回的结构记录
- 如果 ESI 返回空列表，会清空对应军团的本地结构记录
- 这样 dashboard 始终展示当前快照，不会把历史脏数据误认为有效状态

## 设计决策

### 决策：用 system_config 存授权映射和提醒阈值

- 理由：这组数据属于少量全局配置，不需要单独拆表，也不应该和结构快照混在一起
- 不变量：授权映射必须始终指向当前可管理军团中的有效 Director 人物；阈值为 `0` 时表示关闭对应提醒

### 决策：结构管理采用 snapshot-first

- 理由：列表页和徽章只需要稳定可读的快照，不应在每次打开页面时都回源 ESI
- 不变量：页面、筛选和徽章都只读当前快照与缓存配置，实时 ESI 只在用户主动触发刷新时进入队列

### 决策：刷新走共享后台任务管理器

- 理由：刷新任务要支持异步返回、关停传播和统一拒绝新任务，不能靠临时 goroutine 完成
- 不变量：`RunTask` 成功只表示任务已入队；服务进入关停后必须拒绝新刷新任务

## 关键入口文件

- `server/internal/router/router.go`
- `server/internal/handler/corporation_structure.go`
- `server/internal/service/corporation_structure.go`
- `server/internal/repository/corporation_structure.go`
- `server/internal/model/sys_config.go`
- `server/internal/service/badge.go`
- `server/pkg/background/manager.go`
- `server/jobs`
- `static/src/router/modules/dashboard.ts`
- `static/src/api/corporation-structures.ts`
- `static/src/views/dashboard/corporation-structures`

## 当前不变量

- 只允许 `admin` / `super_admin` 进入该模块
- 授权映射、阈值和列表数据都以当前快照为准，不保留旧结构作为“兼容兜底”
- `corporation_structures_attention` 只应反映当前可管理军团内、满足阈值条件的结构数量
- `run-task` 必须异步入队，且必须依赖共享后台任务管理器的生命周期管理
- 任何后续刷新行为都不应把页面写成直接回源 ESI 的实时视图

---
status: active
doc_type: feature
owner: engineering
last_reviewed: 2026-04-23
source_of_truth:
  - server/main.go
  - server/global/global.go
  - server/pkg/background/manager.go
  - server/internal/router/router.go
  - server/bootstrap/cron.go
  - server/internal/taskregistry/registry.go
  - server/internal/model/task.go
  - server/internal/repository/task.go
  - server/internal/service/task.go
  - server/internal/handler/task.go
  - server/internal/handler/esi_refresh.go
  - server/jobs/jobs.go
  - server/jobs/esi_refresh.go
  - server/jobs/auto_srp_schedule.go
  - static/src/router/modules/system.ts
  - static/src/api/task-manager.ts
  - static/src/api/esi-refresh.ts
  - static/src/views/system/task-manager
---

# 任务管理

## 当前能力

- `/system/task-manager` 将后台任务统一展示为「任务」「ESI 状态」与「执行历史」三个 tab
- 任务页展示任务名、说明、分类、任务类型、当前生效 cron、默认 cron 与最近一次执行结果
- ESI 状态页集中展示按人物执行、按任务名执行、全量执行与状态查询入口
- 管理员可手动触发支持运行的任务；ESI 相关控制仍保留在同一页面中
- `super_admin` 可修改周期任务的 cron；修改会立即重载运行时调度，并持久化到 `task_schedules`
- 通用任务的 cron/manual 执行都会写入 `task_executions`，执行历史页支持按任务名与状态筛选
- 执行历史页会分别展示触发者昵称与触发者 ID；cron 触发的记录没有人工触发者信息
- 任务定义来自运行时注册表，当前已覆盖 ESI 刷新、联盟 PAP 抓取/归档、自动职权同步、执行历史清理、军团准入检查、队长归因同步、队长奖励处理、导师奖励与自动 SRP
- `auto_srp` 属于事件驱动任务：会显示在任务页中，但没有手动触发入口，也不支持 cron 编辑
- 服务启动时会注册任务定义、恢复 `task_schedules` 覆盖、恢复待执行的自动 SRP 延迟调度，并启动周期任务调度器
- `/api/v1/tasks/:name/run` 与 ESI 管理页里会 fan-out 的后台触发入口会把执行交给共享后台任务管理器；服务进入关停后，这些入口会拒绝新任务而不是继续创建裸 goroutine

## 当前任务集合

| 任务名 | 分类 | 类型 | 支持手动触发 | 用途 |
| --- | --- | --- | --- | --- |
| `esi_refresh` | `esi` | `recurring` | 是 | 刷新已登记人物的 ESI 队列 |
| `alliance_pap_hourly` | `operation` | `recurring` | 是 | 抓取联盟 PAP 增量数据 |
| `alliance_pap_archive` | `operation` | `recurring` | 是 | 执行联盟 PAP 月度归档 |
| `auto_role_sync` | `system` | `recurring` | 是 | 同步自动职权 |
| `task_execution_history_cleanup` | `system` | `recurring` | 是 | 清理超出保留期的任务执行历史 |
| `corp_access_check` | `system` | `recurring` | 是 | 校验军团准入并调整基础职权 |
| `captain_attribution_sync` | `operation` | `recurring` | 是 | 同步队长赏金归因，默认 `@every 13h` |
| `captain_reward_processing` | `operation` | `recurring` | 是 | 处理队长归因奖励，默认 `@every 100h` |
| `mentor_reward` | `operation` | `recurring` | 是 | 结算导师奖励 |
| `auto_srp` | `operation` | `triggered` | 否 | 跟踪 PAP 事件驱动的自动 SRP 处理 |

## 入口

### 前端页面

- `static/src/views/system/task-manager`
- `static/src/router/modules/system.ts` 中的 `/system/task-manager`

### 后端路由

- `/api/v1/tasks`
- `/api/v1/tasks/history`
- `/api/v1/tasks/:name/run`
- `/api/v1/tasks/:name/schedule`
- `/api/v1/tasks/esi/*`
- `/api/v1/info/esi-refresh`

## 权限边界

- `/system/task-manager` 页面仅 `super_admin`、`admin` 可见
- `/api/v1/tasks`、`/api/v1/tasks/history`、`/api/v1/tasks/:name/run` 与 `/api/v1/tasks/esi/*` 要求 `admin`
- `/api/v1/tasks/:name/schedule` 额外要求 `super_admin`
- `/api/v1/info/esi-refresh` 要求 `Login`，并在服务端校验目标人物归属，用户只能刷新自己的角色
- 前端路由声明了 `execute_task` 与 `update_schedule` 按钮权限，但真正的权限边界以后端路由为准
- 没有 `RunFunc` 的任务只用于可视化与历史记录展示，不会暴露手动执行入口

## 设计决策

- 决策：任务管理模块继续把“是否允许执行、是否允许并发、是否写历史”交给 `taskregistry.Registry` 和 `TaskService` 负责，但把“请求返回后仍要继续运行的执行生命周期”交给共享 `background.Manager` 负责。
- 理由：管理员触发任务和 ESI fan-out 刷新都需要快速返回响应；如果这类入口只起裸 goroutine，就无法在服务关停时统一拒绝新任务、取消已启动任务，或保证锁释放和执行历史落库发生在受控生命周期内。
- 必须保留的不变量：
  - `/api/v1/tasks/:name/run` 必须先获取注册表锁，再把带锁上下文交给 `background.Manager`；如果调度失败，锁必须立即释放。
  - `/api/v1/tasks/esi/run` 与 `/api/v1/info/esi-refresh` 这种单人物即时触发仍使用请求上下文同步执行；`/api/v1/tasks/esi/run-task` 与 `/api/v1/tasks/esi/run-all` 这类 fan-out 入口必须使用共享后台任务管理器。
  - 任务执行历史仍由 `TaskService` 负责落库；迁移到 `background.Manager` 只改变生命周期管理，不改变任务定义、权限边界或历史记录模型。

## 关键不变量

- 任务定义与执行锁属于运行时注册表；数据库只保存周期任务的调度覆盖和执行流水
- 仅 `recurring` 任务允许修改调度；支持带秒字段的 6 段 cron 表达式，也支持 `@every <duration>` 描述符
- 同一任务在同一进程内不会并发执行两次；手动触发冲突时返回 `409 Conflict`，cron 触发冲突时跳过本轮执行
- 调度修改会先更新运行时 cron，再写入 `task_schedules`；若持久化失败，服务会尝试回滚旧调度
- `/api/v1/tasks/esi/*` 仍然是 ESI 队列的专用管理入口，不等同于通用任务执行历史
- 服务启动后仍会立即补跑一次 `esi_refresh` 队列，并从舰队状态中恢复尚未到点的自动 SRP 延迟执行
- 通用任务的异步手动触发与 ESI fan-out 触发必须处于共享后台任务管理器的受跟踪生命周期内；服务进入关停后，这些入口必须返回显式失败，而不是继续接收新任务

## 主要代码文件

- `server/main.go`
- `server/global/global.go`
- `server/pkg/background/manager.go`
- `server/internal/taskregistry/registry.go`
- `server/internal/model/task.go`
- `server/internal/repository/task.go`
- `server/internal/service/task.go`
- `server/internal/handler/task.go`
- `server/internal/handler/esi_refresh.go`
- `server/bootstrap/cron.go`
- `server/jobs/jobs.go`
- `server/jobs/esi_refresh.go`
- `server/jobs/auto_srp_schedule.go`
- `static/src/api/task-manager.ts`
- `static/src/api/esi-refresh.ts`
- `static/src/views/system/task-manager`

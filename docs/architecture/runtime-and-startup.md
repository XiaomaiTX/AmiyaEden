---
status: active
doc_type: architecture
owner: engineering
last_reviewed: 2026-04-23
source_of_truth:
  - server/main.go
  - server/global/global.go
  - server/pkg/background/manager.go
  - server/bootstrap
  - server/jobs/esi_refresh.go
  - server/internal/service/mail_dispatch_async.go
---

# 运行与启动

本文档描述后端启动顺序与运行时行为。依赖要求与本地启动流程见 `docs/guides/local-development.md`。

## 后端启动顺序

`server/main.go` 当前启动流程：

1. 初始化配置
2. 初始化日志
3. 初始化共享后台任务管理器
4. 初始化 JWT
5. 初始化数据库
6. 初始化 Redis
7. 初始化任务注册表与 cron 调度
8. 异步检查 SDE
9. 注册 ESI scopes
10. 初始化 HTTP 路由
11. 启动服务

SDE 检查更新当前行为：

- 通过 `sde.download_url` 获取最新 release 信息
- 若本地配置了 `sde.proxy`，会优先尝试通过代理下载
- 若代理连接失败，会自动回退为直连重试
- 导入成功后在 `sde_versions` 中记录当前版本

## 数据库初始化副作用

数据库初始化不仅建立连接，还会执行：

- `AutoMigrate`
- 自定义索引补齐
- schema 规范化与兼容处理

## 任务调度启动副作用

- `bootstrap.InitCron()` 会构建运行时 `taskregistry.Registry`，并由 `server/jobs/` 注册所有当前任务定义
- 周期任务会优先读取 `task_schedules` 中的管理员覆盖 cron；未配置时使用代码中的默认 cron
- 通过 cron 或 `/api/v1/tasks/:name/run` 触发的通用任务会写入 `task_executions`
- `auto_srp` 会在启动时恢复未到点的延迟执行 timer，但它本身不是 cron 周期任务
- ESI 刷新队列在服务启动后仍会立即补跑一次，避免新启动实例长时间等待下一个周期
- QQ 群治理 worker 随 cron 初始化启动，并由共享后台任务管理器参与关停；成员快照、批量核验、写动作、租约和重试状态始终保存在 PostgreSQL，不依赖进程内 timer

## 设计决策

### 共享后台任务管理器负责需要参与关停的后台工作

- 决策：服务在日志初始化后立即创建一个进程级 `background.Manager`，只用于管理那些会跨出当前请求生命周期、且必须参与进程关停的后台任务。当前覆盖 cron 触发的通用任务、任务管理页的异步手动触发、ESI 启动补跑 / 新人物全量刷新 / 管理端 fan-out 刷新，以及异步邮件投递。
- 理由：这些路径如果继续使用裸 goroutine，服务关停时就无法统一拒绝新任务、向已启动任务传播取消信号，也无法在同一个超时预算内等待收尾，容易留下半执行状态、重复补跑或无上下文的失败日志。
- 必须保留的不变量：
  - `background.Manager` 必须先于任何会提交受跟踪后台任务的代码初始化，并通过 `global` 暴露给任务、handler 与服务层复用。
  - 收到关停信号后，进程会先停止 cron 接受新的周期触发，再在同一个超时预算内依次等待 HTTP 请求排空、已启动的 cron 作业结束，以及 `background.Manager` 里的受跟踪任务收尾。
  - 进入关停后，`background.Manager` 必须拒绝新任务，并向已跟踪任务广播取消；任务代码如果接收了上下文，就应把取消继续向下游传播。
- 只有那些在 manager 已进入关停时仍必须完成同步语义的调用点，才允许通过 `background.RunOrSchedule` 明确回退为调用方上下文内联执行；不能静默丢弃任务。

### QQ 群治理 OneBot 反向连接只接受私有、单机器人接入

- 决策：QQ 群治理独立使用 `/internal/onebot/v11/ws` 的 OneBot V11 反向 WebSocket，不复用通用 `webhook.onebot` 配置或其任意 URL 发送能力。
- 理由：群治理会接收事件并通过受控队列读取成员快照或发起审批、拒绝、名片等自动写操作，需要把机器人身份、来源网络、动作 echo 和持久动作队列作为单独的安全边界。
- 必须保留的不变量：连接必须同时验证专用 Bearer Token、`X-Self-ID` 与配置的唯一机器人 QQ、以及 `onebot.allowed_cidrs`；配置不完整、来源不受控或 Redis 不可用时，不得执行任何 QQ 操作（包括成员读取）。

## 关键入口文件

- `server/main.go`
- `server/global/global.go`
- `server/pkg/background/manager.go`
- `server/bootstrap/cron.go`
- `server/internal/handler/task.go`
- `server/internal/handler/esi_refresh.go`
- `server/jobs/esi_refresh.go`
- `server/internal/service/task.go`
- `server/internal/service/mail_dispatch_async.go`

## 当前不变量

- `taskregistry.Registry` 仍然负责进程内任务去重与锁管理；`background.Manager` 只负责生命周期与关停传播，不替代任务锁。
- 任务管理与 ESI fan-out 入口在返回 `任务已触发` 之后，对应执行必须处于受跟踪状态；服务进入关停后，这些入口应显式拒绝新任务，而不是启动裸 goroutine。
- 本文档中的“统一后台任务管理”只覆盖必须参与进程关停的后台任务，不意味着仓库内所有 goroutine 都已经迁移到 `background.Manager`。

## 运行时提示

- 新人物 SSO 成功后，后台会触发 ESI 全量刷新与自动权限同步
- 任务管理相关的异步执行使用共享后台任务管理器；周期任务支持管理员查看与超级管理员重设 cron
- SDE 缺失会直接影响舰队配置 EFT 解析、名称翻译、搜索等共享能力
- `register` 页面源码仍在仓库中，但不是当前支持的登录架构；`forget-password` 页面已移除

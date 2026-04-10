---
status: active
doc_type: architecture
owner: engineering
last_reviewed: 2026-04-10
source_of_truth:
  - server/main.go
  - server/bootstrap
---

# 运行与启动

本文档描述后端启动顺序与运行时行为。依赖要求与本地启动流程见 `docs/guides/local-development.md`。

## 后端启动顺序

`server/main.go` 当前启动流程：

1. 初始化配置
2. 初始化日志
3. 初始化 JWT
4. 初始化数据库
5. 初始化 Redis
6. 初始化任务注册表与 cron 调度
7. 异步检查 SDE
8. 注册 ESI scopes
9. 初始化 HTTP 路由
10. 启动服务

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

## 运行时提示

- 新人物 SSO 成功后，后台会触发 ESI 全量刷新与自动权限同步
- 后台任务由统一任务管理器调度；周期任务支持管理员查看与超级管理员重设 cron
- SDE 缺失会直接影响舰队配置 EFT 解析、名称翻译、搜索等共享能力
- `register` 页面源码仍在仓库中，但不是当前支持的登录架构；`forget-password` 页面已移除

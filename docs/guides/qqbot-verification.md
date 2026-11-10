---
status: active
doc_type: guide
owner: engineering
last_reviewed: 2026-11-10
---

# OneBot 基础设施验证与运行

## 清理基线

核实第一阶段清理提交 `b90b56a`：旧治理 handler/service/repository/model、建筑 QQ 扫描、任务注册、OneBot webhook 分支、双前端旧页面/配置/API 及 nginx 专用代理均已移除。旧 webhook 类型继续被明确拒绝，未恢复舰队、群治理或建筑自动写。清理遗留的两个 Go 无效分支/函数及 Vue 未使用导入、格式问题在本次修正。历史数据库人工清理仍遵循 [清理指南](qq-bot-retirement-cleanup.md)，本次未连接生产数据库。

## 回归入口与边界

项目必需命令见 [验证规范](../standards/testing-and-verification.md)。本次环境 pnpm 的引擎仓库锁不可写，使用已安装的 `node_modules/.bin` 执行相同工具。网络测试及 tsx 的本地 socket 在允许本地监听的环境执行。

- `service/qqbot_test.go`：错误鉴权、CIDR/转发头、身份、角色、脱敏、默认超时、响应结构、慢事件、替换和关闭 pending、事务回滚、幂等键及冲突、读取/写入未知结果、取消、租约恢复、风控暂停、真实半开操作、限流、Redis 重启、消费者失败/崩溃、结果消费、优先级老化、受限重试及物理清理。
- `repository/qqbot_test.go`：过期历史物理删除、优先级老化及旧 token fencing。
- `router/router_test.go`：实际管理路由拒绝未登录、普通用户和非超管管理员。
- `service/qqbot_integration_test.go`：专用 PostgreSQL 随机 schema 中的真实唯一约束与并发认领；真实 Redis Lua 全局原子限流和 TTL 恢复。通过 `QQBOT_TEST_POSTGRES`（关键字 DSN）及 `QQBOT_TEST_REDIS` 显式提供隔离测试端点；无环境变量时此一集成用例明确 skip，不会读取应用生产配置。集成测试移除自己创建的随机 schema，不清空共享表。
- React API 与页面测试：超管路由、契约、未知消息无重试按钮、页面加载不会提交任何写操作。

此次集成验证使用无持久卷的临时 PostgreSQL 17、Redis 7.4 容器，以及真实回环 WebSocket 客户端；没有连接真实 QQ 平台或提交实际业务操作。Go 专项包含竞态检测。React 大型既有页面测试出现环境相关的 15 秒超时，单 worker 定向重跑通过；完整回归结果以实际输出为准。

## 部署与使用

只部署一个持有连接的后端实例。管理页为双前端 `/system/qqbot`，默认关闭接入；从超管页设置机器人 ID、唯一 Token、允许的 TCP 对端 CIDR 和统一限流间隔。Token 只写不读，留空保留原值。配置变更会断开旧连接。机器人以 Universal 反向 WebSocket 直连后端 `/internal/onebot/v11/ws`，使用 Bearer Token、X-Self-ID、X-Client-Role。不要把客户端可伪造的转发头当作 CIDR 来源。

业务方将来只能通过 Submit、GetOperation 和启动前注册消费者接入，事务内提交意图须等待业务事务提交；本期没有任何业务调用者。消费者应使用传入事务写本地副作用，保持幂等，不在消费事务中等待外部调用。结果成功仅是平台确认，不是用户已读。消息/申请/清退未知结果不可重试。清理自动每小时执行；长期停机后的过期操作不会发送。

设计、不变量与限制见 [PRD 与设计](../architecture/qqbot-dispatch.md)，尤其是 OneBot 没有持久事件 ack/replay 的限制。

## 本次结果

- 后端完整 `go test ./...`、构建、golangci-lint 通过；新增调度专项及 PostgreSQL/Redis 验证含 `-race`，通过。
- Vue lint、类型检查、构建及 126 项单测通过。
- React lint、类型检查、API 契约检查和构建通过；完整回归 180/182 通过，两个既有大型页面用例触发默认 15 秒超时。将这两个文件单独以单 worker、60 秒 CLI 超时复核，9/9 通过；新增 QQBot API、权限和页面测试全部通过。未修改仓库测试超时配置。
- 静态搜索确认 QQ 协议发送只在私有客户端中，生产调用点只有持久调度执行器；没有业务 Submit 或消费注册调用点。
- 临时 PostgreSQL/Redis 容器已停止并移除，无生产配置、凭据或业务数据变更。

---
status: active
doc_type: architecture
owner: engineering
last_reviewed: 2026-11-26
source_of_truth:
  - server/internal/model/qq_official.go
  - server/internal/repository/qq_official.go
  - server/internal/service/qq_official.go
  - server/internal/service/qq_official_runtime.go
  - server/internal/service/qq_official_gateway.go
  - server/internal/service/qq_official_dispatch.go
  - server/internal/service/internal/qqapi/gateway.go
  - server/internal/service/internal/qqapi/client.go
---

# QQ 官方机器人统一调度

## 入口与边界

业务提交类型化意图并指定业务幂等键、期限和授权军团，不直接访问 QQ。首次提交在 PostgreSQL 事务中冻结目标群；同键同请求返回原任务，异内容冲突。待办、尝试、限流窗口、熔断与租约都由同一数据库持久化：在本功能的低速率下，避免把队列与限流放到两个不同数据系统。通用 cron 只扫描业务数据或清理；后台任务管理器只负责 worker 生命周期。

后台先通过官方 HTTPS `GET /gateway` 发现接入点，再主动连接[腾讯官方 WebSocket](https://bot.q.qq.com/wiki/develop/api-v2/dev-prepare/event-emit/websocket.html)，单分片订阅 `GROUP_AND_C2C_EVENT`。不提供入站 HTTP 回调。群通知、群状态查询及人工审批仍调用官方 HTTPS OpenAPI。网关发现和凭证兑换占用原有单 HTTP 执行槽并计入共享预算；已建立的 WSS 连接与心跳不占用该槽。收事件不受本站出站投递开关控制。

多个后端共享同一 PostgreSQL 时，以 App 行上的独立短租约选出一个网关接收实例；租约按数据库时间续约，续约失败就关闭连接。白名单事件只持久化必要字段，投影提交后才推进 Resume 游标。事件消费和任务准入使用同一 App 行的锁，已持久化的退出/拒收事件优先于新的业务外呼。会话及游标只保存在当前接收进程内：连接断开优先按协议 Resume，但进程重启、换主、会话失效时不能保证补齐事件。忽略私聊、群聊天内容等无关事件。启用人工审批写入时，额外订阅[入群申请事件所需的 `GROUP_MEMBER_EVENT`](https://bot.q.qq.com/wiki/develop/api-v2/autogen/event/group_join_request.html)；若 QQ 拒绝该组合的订阅权限，则改用基础群事件订阅，不重复申请被拒的组合。人工刷新始终使用 HTTPS 列表。

## 后台配置与切换

`qq_official_installation` 只保存当前 AppID、人工审批写入开关、配置版本和密文，不经过通用系统配置列表。AppSecret 使用基于现有部署 `jwt.secret` 派生的独立密钥，结合随机 nonce 与 AppID 认证加密（HKDF-SHA-256 / AES-256-GCM）。API 和审计不返回密文、明文或平台令牌；令牌仍只在进程内。各副本从 PostgreSQL 读取同一版本，无需重启或 Redis 广播。

锁定顺序为配置行、AppID 行：配置修改独占配置行，任务准入、接收租约续约及事件入箱持共享锁并验证所用配置版本。换 AppID 或轮换密钥必须等现有租约结算，取消尚未发出的任务并停止投递、重置人工审批开关，不重置额度或搬迁队列。禁用人工审批时立即取消待执行决定；已经准入的写操作仍记录真实结果，但配置一旦变化就不重试该审批。

## 准入规则

- 唯一执行槽：同一 AppID 共享同一个 PostgreSQL，短事务锁 App 行，检查版本/到期/额度/风控，写入尝试和 30 秒租约。提交后才允许私有传输发出一个 HTTPS 请求。5 秒的截止时间从请求准入前算起；更新结果需持有同一个租约。禁止横跨不共享调度状态的部署共用同一 AppID。不承诺超时后 QQ 一侧已经停止处理。
- 所有 OpenAPI HTTPS 请求（含凭证兑换和网关发现）至少间隔 1 秒，滚动 60 秒最多 48 次；主动群消息默认 24 条/分钟（确认认证后 48）、单群 12 条/分钟、滚动 24 小时 800 条。申请列表和机器人群状态各 20 次/分钟，人工决定 40 次/分钟并且同群至少间隔 10 秒。计算窗口保守包含请求准入与实际发出的短时差。它们是本站预算，不是[平台配额](https://bot.q.qq.com/wiki/develop/api-v2/autogen/api/v2_groups_group_openid_messages.post.html)，不能保证没有其他平台约束。凭证单独降频并缓存，不在业务端获取。
- 任务按优先级与目标群轮转，同群同优先级按队列顺序；可执行的普通任务在连续 3 次高优执行后优先。软上限为普通任务每 App 1600、每群 80，总上限 2000、100，分别留出高优容量。不可执行的群顺延，不拖住仍有额度的其他群。
- 平台 429 按 `Retry-After` 和本站共享冷却处理；无值时至少 60 秒，反复命中升级至 5/15 分钟。HTTP 429 来源不明时保护 App，已知的主动消息频控只影响本 App 消息类。读、消息、群管理写分别累积暂态失败并熔断，半开仍受并发与配额限制。账号封禁/凭证错误停发等待处理。不使用平台错误文案判定。数据库不可用没有兜底直连。

## 结果与留存

`queued / executing / retry_wait / awaiting_review / succeeded / failed / unknown / expired / cancelled` 是账本状态。平台确认需端点明确证据，群消息确认后保留回执摘要，不存原始消息 ID。平台审核受理不是成功；当前没有经目标账号实测、可供自动关联的审核事件，等待超时后标未知。写操作可能已经提交时不自动重试，包括进程在提交尝试与落结果之间消失的情况。可重试的读取和明确未执行的暂态失败受次数和期限双重限制。

最小入箱事件、申请资料按 7 天清理；事件投影后立即清掉重复申请字段，消息正文随终态删除。清理任务不需 QQ 凭证，也处理已停用或换出的 App 数据。脱敏尝试和任务保留约 90 天，幂等保证依赖这段留存期。应用于开放平台时还要按[运营规范](https://bot.q.qq.com/wiki/business/)和[运维指南](../guides/qq-official-operations.md)响应数据清除，不依赖定期清理覆盖更短的要求。

主要实现：[调度服务](../../server/internal/service/qq_official_dispatch.go)、[类型化业务提交](../../server/internal/service/qq_official.go)、[官方适配器](../../server/internal/service/internal/qqapi/client.go)、[数据仓储](../../server/internal/repository/qq_official.go)。

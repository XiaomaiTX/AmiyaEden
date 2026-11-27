---
status: active
doc_type: guide
owner: engineering
last_reviewed: 2026-11-27
source_of_truth:
  - server/bootstrap/qq_official.go
  - server/internal/service/qq_official_runtime.go
  - server/internal/service/qq_official_gateway.go
  - server/internal/service/internal/qqapi/gateway.go
---

# QQ 官方机器人部署与运维

## 首次配置

1. 在[QQ 开放平台](https://bot.q.qq.com/open)建立专用机器人并完成主体、权限和隐私配置。测试与生产使用不同 AppID；同一 AppID 只由共享同一 PostgreSQL 的本站实例使用。平台出站 IP 白名单只放本站固定出口 IPv4。仅在受信任的 HTTPS 管理页面填写凭证，不要将 AppSecret 放进 URL、日志、工单或群消息。
2. 超管打开「系统管理 → QQ 官方机器人」，填写 AppID 和 AppSecret 后保存。后台加密保存 AppSecret，只显示是否已保存，不回显密钥。调度默认停用，人工审批写入也默认关闭；两者要分别开启。不需要 QQ 专用环境变量。
3. 按[腾讯官方 WebSocket 文档](https://bot.q.qq.com/wiki/develop/api-v2/dev-prepare/event-emit/websocket.html)确认机器人有 `GROUP_AND_C2C_EVENT` 订阅权限。部署需允许后端向 `api.bot.qq.com` 访问 HTTPS 和 WSS（TCP 443），不需要配置 Webhook 地址或开放任何入站端口。若之前配置过 QQ Webhook，切换时在平台撤下旧回调地址，并清理部署中的专用代理规则；本站不再接收该回调。
4. 确认页面网关状态为「已连接」，再使用受控测试群入群并确认收到事件，或通过有权限的群状态查询核验。填写别名、确认群主授权；应用启用后发送固定测试。验证平台回执与群内实际展示，再配置具体业务军团。需要审批时，先用专门样本验证申请读取、确认机器人在群中是管理员，再在页面开启「人工审批写入」并仅为已验证群开启审批。实时入群申请还需平台授予 `GROUP_MEMBER_EVENT`；没有该权限时仍可用页面手动刷新列表，后端会继续接收基础群事件。新增该权限后可在维护窗口重启后端以重新订阅。禁止通过试探发送规模确定配额；压力验证只访问模拟上游。

## 故障与灰度

- 按一个测试群、一个业务试点群、逐军团的顺序启用。监视最老待办时间、队列深度、失败/未知和 429。看到限额或 429 时降低产出或等待，不提高额度或跳过冷却。群拒收、机器人退群、权限不足均需要在 QQ 处理原因，不使用其他形式绕过。
- 只需止血时暂停出站；需要废弃待办时停用。两者均不关闭事件连接。轮换 AppSecret 或更换 AppID 后，投递和人工审批写入保持关闭，未执行任务不会重放。已有 HTTP 请求占用租约时，先暂停并等该请求结算，再尝试保存新凭证。AppID 之间不迁移群目标或待办；切换后重新核验连接和群。撤下应用时在平台停用机器人并吊销凭证。
- 网关普通断线会退避重连。进程重启或会话失效后不能保证收齐离线事件，恢复后要核对受影响群的授权和主动接收状态。若状态为「连接受阻」，先在 QQ 平台解决权限、凭证或账号状态问题，再在后台重新填写当前 AppSecret（或者保存新凭证）以发起重连。
- 加密依赖所有后端实例一致、稳定的现有 `jwt.secret`（不能用示例值），不新增密钥环境变量。数据库备份与部署密钥应分别妥善保护。JWT 密钥轮换后，旧 AppSecret 无法解密；升级所有副本后在后台重新录入 AppSecret，再启用。不需要回填或迁移旧的机器人环境变量。
- 状态未知先根据脱敏 TraceID、QQ 客户端和实际群状态核对；不要按失败补发。超管审批功能默认不向未经验证的环境开放。线上不得用压力测试填满队列。

## 本地自动化验证

[双 worker 压测](../../server/internal/service/qq_official_test.go)使用模拟 QQ，在空的隔离 PostgreSQL 数据库上验证千条混合任务、容量和单在途保护。数据库必须单独创建并命名为 `qq_official_test`；只给测试进程注入 `QQ_TEST_POSTGRES_DSN`，再在后端目录运行 `go test ./internal/service -run '^TestQQOfficialPostgresParallelLoad$' -count=1`。测试不接受其他数据库名，不读取部署机器人凭证，不连接 QQ。

## 数据留存与删除

最少存储群绑定、必要申请字段、待办和脱敏尝试。处理用户删除申请时先依合规流程验证用户身份和对应群内 OpenID，再暂停相关群/能力；运维在维护窗口按 AppID 与组内身份范围核对并清理申请和入箱对应行，以及可能关联的未运行任务。不使用站内 QQ 号做代换，不扩大范围。不再提供服务时按[平台规范](https://bot.q.qq.com/wiki/business/)及时删除收集的个人数据并处置相关备份；常规定期清理不能替代更短的合规时限。执行任何数据删除前确认目标库、条件与所需审批，不自动清理其他模块数据。

- 业务状态和权限见[功能说明](../features/current/qq-official.md)。
- 并发、退避和结果边界见[调度架构](../architecture/qq-official-dispatch.md)。

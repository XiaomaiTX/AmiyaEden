---
status: active
doc_type: feature
owner: engineering
last_reviewed: 2026-11-22
source_of_truth:
  - server/internal/router/router.go
  - server/jobs/jobs.go
  - server/bootstrap/db.go
---

# QQ Bot 下线边界

OneBot 反向连接、统一调度与运维、通用群通知、舰队 QQ Ping、建筑 QQ 预警、入群申请处理、群名片同步和成员巡检已全部下线。没有反向 WebSocket、QQ Bot 管理 API、服务契约、群业务任务、旧 Webhook 类型或双前端管理入口，也没有兼容层或排队数据重放。

用户 QQ 联系字段、自助资料维护、超级管理员联系方式管理、招新 QQ 登记与链接、导师联系方式及其他业务中的 QQ 展示继续保留。舰队创建时的 `send_ping` 与手动 Ping 仍可通过 Discord、飞书或钉钉 Webhook 发送，不发送 QQ 消息。建筑快照/授权/燃料管理及站内提醒阈值与角标不受影响。

应用不自动删除历史数据，不再迁移 QQ Bot 专属模型或注册其任务。数据库及 Redis 仅按 [显式清理指南](../../guides/qq-bot-retirement-cleanup.md) 备份、预览并另行批准清理。[业务场景清单](../../specs/draft/qq-bot-business-scenarios.md)仅记录待评审诉求，历史设计不构成当前能力或未来实现要求。

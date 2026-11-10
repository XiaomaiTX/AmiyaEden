---
status: active
doc_type: feature
owner: engineering
last_reviewed: 2026-11-10
source_of_truth:
  - server/internal/router/router.go
  - server/jobs/jobs.go
  - server/bootstrap/db.go
---

# 旧 QQ Bot 业务下线边界

旧 QQ 群治理、舰队 QQ Ping、Webhook QQ 测试发送和建筑 QQ 预警均已下线，旧治理 API、任务、页面、菜单、配置和通知类型不再提供，没有兼容接口。新的 [项目调度基础设施](qqbot.md) 提供独立 OneBot 接入与超管运维，默认关闭；没有恢复以上业务，也没有接入任何真实业务自动写操作。

用户 QQ 联系字段、自助资料维护、超级管理员联系方式管理、招新 QQ 登记与链接、导师联系方式及其他业务中的 QQ 展示继续保留。舰队业务、建筑快照/授权/燃料管理、站内提醒阈值与角标、Discord/飞书/钉钉 Webhook 不受影响。

数据库不会自动删除旧数据，也不会再自动注册旧模型、迁移旧任务或创建专属索引。清理前须按[数据库清理方案](../../guides/qq-bot-retirement-cleanup.md)备份和审查。[第二阶段场景清单](../../specs/draft/qq-bot-business-scenarios.md)仅记录待确认业务诉求，不自动授权接入业务；本期仅交付独立基础设施。旧设计草案是历史材料，不能作为当前能力或新实现要求。

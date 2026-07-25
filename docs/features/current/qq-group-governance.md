---
status: active
doc_type: feature
owner: engineering
last_reviewed: 2026-07-22
source_of_truth:
  - server/internal/service/sys_config.go
  - server/internal/handler/qq_governance_onebot.go
  - server/internal/service/qq_governance.go
  - server/internal/service/qq_governance_reconcile.go
  - server/internal/service/qq_governance_worker.go
  - server/internal/repository/qq_governance.go
  - server/internal/model/qq_governance.go
---

# QQ 群治理

## 当前能力

- NapCat 通过 OneBot V11 反向 WebSocket 连接 `/internal/onebot/v11/ws`。
- 连接同时校验专用 Bearer Token、`X-Self-ID` 机器人 QQ 与 `onebot.allowed_cidrs`；未配置或不匹配时拒绝连接。
- 已启用群规则会处理入群申请、成员加入和成员离开事件；事件以平台稳定字段去重。
- 准入判断只读取 Seat 本地 QQ 绑定、主人物军团和职权，并同时满足已配置的军团与职权条件。未绑定、未设置主人物、人物不存在或缺少军团等确定性资料缺失直接判为 `unmatched`；仅数据库读取异常等暂态故障才进入 `review_wait` 自动复查。空规则不会自动放行。
- 明确匹配的申请创建批准任务；配置 `auto_reject_unmatched` 时，明确不匹配的申请创建拒绝任务；成员加入后可异步同步名片。
- 动作任务、成员状态、审查记录和动作日志持久化到 PostgreSQL；worker 使用租约领取、状态版本校验、指数退避与死信状态。
- 所有 OneBot 读取与写入都进入 QQ 群治理专属的 Redis 限流器（全局、群、写入时再加 QQ 维度）；Redis 不可用时只会重试，不会绕过限流访问 QQ。限流和风控不复用系统通用任务或 ESI 队列。
- 超级管理员可在 `/system/qq-governance` 管理群规则、查看成员状态/判断记录/队列/告警/指标，重试死信、启动或复用一次完整巡检、确认告警和解除熔断；没有任何人工批准、拒绝、改名片或清退 QQ 成员的接口。
- 一次巡检先受限流地读取完整成员快照，排除机器人并去重排序后将 QQ 集合固化到数据库；再以每批最多 50 人的持久批任务计算资格。进程重启或再次触发会从未完成的快照成员续跑，绝不使用 OneBot 返回顺序或 `LastQQ` 作为游标。完整结束后才标记快照中不存在的旧成员为离群。
- 管理页按规则、群状态、运行监控和 OneBot/全局设置四个页面组织。群状态展示已配置治理规则的群，并显示群名、成员数、机器人是否为管理员、本轮已处理数/快照总数及运行状态；成员快照和群资料刷新均为独立的后台 OneBot 读取任务，每次调用都经过 Redis 限流，页面只展示最近持久化结果，以便断连时仍可显示最近结果。
- 规则编辑中的允许军团必须从 ESI 名称解析结果选择，持久化稳定的军团 ID；群号使用纯数字输入，不提供增减步进控件。军团搜索走 ESI 公开接口 `POST /universe/ids/`，只对输入的完整军团名称进行精确匹配，因此无需 SSO Token。规则列表展示当前成员数，编辑弹窗中的已选军团统一展示为 `军团名 (军团ID)`，规则响应额外返回只读的 `allowed_corporations`，数据库仍只持久化稳定的军团 ID。
- `UNKNOWN` 会延迟自动复查，连续三次创建治理页告警，不形成待人工审核工作流；动作失败率会触发三级熔断。三级熔断暂停 QQ 写操作，但仍允许受限流的成员快照和本地计算继续恢复可观测性。

## 配置与边界

`onebot` 配置保存在 `system_config`，仅超级管理员可在系统基础配置页面维护；它与现有通用 `webhook.onebot` 完全独立。生产环境必须使用随机令牌、唯一机器人 QQ，并仅配置 NapCat 所在受控网段。

事件保留 90 天，审查与动作日志各保留 180 天。NapCat 镜像、容器和生产网络不由本仓库管理；多机器人和多实例连接选主仍未实现。

## 前端实现映射（迁移期）

- 当前管理行为由 Vue `static/` 侧承接。
- React 尚未承接 `/system/qq-governance`，该页面、API wrapper 和类型出口属于迁移基线的范围漂移追赶项。
- OneBot 安全约束、日志保留和超级管理员边界不因前端迁移而改变。

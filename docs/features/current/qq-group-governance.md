---
status: active
doc_type: feature
owner: engineering
last_reviewed: 2026-07-22
source_of_truth:
  - server/internal/service/sys_config.go
  - server/internal/handler/qq_governance_onebot.go
  - server/internal/service/qq_governance.go
  - server/internal/service/qq_governance_worker.go
  - server/internal/repository/qq_governance.go
  - server/internal/model/qq_governance.go
---

# QQ 群治理（首期）

## 当前能力

- NapCat 通过 OneBot V11 反向 WebSocket 连接 `/internal/onebot/v11/ws`。
- 连接同时校验专用 Bearer Token、`X-Self-ID` 机器人 QQ 与 `onebot.allowed_cidrs`；未配置或不匹配时拒绝连接。
- 已启用群规则会处理入群申请、成员加入和成员离开事件；事件以平台稳定字段去重。
- 准入判断只读取 Seat 本地 QQ 绑定、主人物军团和职权。资料缺失进入 `review_wait`，空规则不会自动放行。
- 明确匹配的申请创建批准任务；配置 `auto_reject_unmatched` 时，明确不匹配的申请创建拒绝任务；成员加入后可异步同步名片。
- 动作任务、成员状态、审查记录和动作日志持久化到 PostgreSQL；worker 使用租约领取、状态版本校验、指数退避与死信状态。
- 自动写操作依赖 Redis 的全局、群和 QQ 级限流；Redis 不可用时只会重试，不会绕过限流写入 QQ。
- 超级管理员可在 `/system/qq-governance` 管理群规则、查看成员/审查/动作/告警与指标，重试死信、执行人工动作、立即巡检和解除熔断。
- 周期巡检使用全局设置的扫描间隔创建持久扫描任务；成员以稳定 QQ 分片、每批最多 50 人处理。所有受治理群共用连续不匹配确认次数与观察期，明确不匹配满足这两项全局条件后才会创建清退任务。
- 管理页按规则、群状态、运行监控和 OneBot/全局设置四个页面组织。群状态只展示已配置治理规则的群，快照由巡检时的 OneBot 群信息读取更新，以便断连时仍可显示最近一次同步结果。
- 规则编辑中的允许军团必须从 ESI 名称解析结果选择，持久化稳定的军团 ID；群号使用纯数字输入，不提供增减步进控件。军团搜索走 ESI 公开接口 `POST /universe/ids/`，只对输入的完整军团名称进行精确匹配，因此无需 SSO Token。规则列表、搜索候选和编辑弹窗的已选项都统一展示为 `军团名 (军团ID)`，规则响应额外返回只读的 `allowed_corporations`，数据库仍只持久化稳定的军团 ID。
- `UNKNOWN` 会延迟复查，连续三次创建治理页告警；动作失败率会触发三级熔断，自动写操作在熔断期间按级别暂停。

## 配置与边界

`onebot` 配置保存在 `system_config`，仅超级管理员可在系统基础配置页面维护；它与现有通用 `webhook.onebot` 完全独立。生产环境必须使用随机令牌、唯一机器人 QQ，并仅配置 NapCat 所在受控网段。

事件保留 90 天，审查与动作日志各保留 180 天。NapCat 镜像、容器和生产网络不由本仓库管理；多机器人和多实例连接选主仍未实现。

## 前端实现映射（迁移期）

- 当前管理行为由 Vue `static/` 侧承接。
- React 尚未承接 `/system/qq-governance`，该页面、API wrapper 和类型出口属于迁移基线的范围漂移追赶项。
- OneBot 安全约束、日志保留和超级管理员边界不因前端迁移而改变。

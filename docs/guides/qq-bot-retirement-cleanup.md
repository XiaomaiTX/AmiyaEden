---
status: active
doc_type: guide
owner: engineering
last_reviewed: 2026-11-10
---

# QQ Bot 下线清理与审查清单

本方案仅供人工审查与后续运维执行。本次未连接生产数据库，未执行 DROP、DELETE 或 Redis 清理。代码已停止创建、读写及注册这些对象；遗留数据不会被自动重放。

## 依赖与删除清单

| 层面 | 下线范围 | 共享边界 |
| --- | --- | --- |
| 后端 | OneBot 反向连接、鉴权、连接管理、事件处理、治理 service/repository/model、队列、租约、限流、风控、审查、日志与治理告警 | ESI、SSO、Mumble、Redis 通用基础设施不删除 |
| 通知 | OneBot HTTP Webhook 类型、治理队列通知、舰队 QQ Ping、建筑 QQ 预警及投递状态 | Discord、飞书、钉钉及舰队模板保留 |
| 双前端 | 治理页面/路由/菜单/API/类型/专属文案/测试，Vue 基础 OneBot 配置，建筑预警开关/群号/手动扫描 | 用户 QQ 联系方式及管理员维护，舰队、建筑列表/授权/服务/燃料管理/阈值/角标保留 |
| 启动与任务 | QQ worker、两项治理周期任务、建筑 QQ 扫描、模型注册、专属迁移与索引 | 其他任务与执行历史基础设施保留 |
| 配置与部署 | OneBot 配置 API、专用 WebSocket nginx 代理 | API 代理、其他服务配置保留 |
| 文档 | 当前功能/路由/架构更新，旧设计标为历史，迁移清单移除旧范围 | 历史 changelog 与旧设计可留存 |

删除文件：

- `server/internal/handler/qq_governance_admin.go`
- `server/internal/handler/qq_governance_onebot.go`
- `server/internal/handler/qq_governance_onebot_test.go`
- `server/internal/model/corporation_structure_alert.go`
- `server/internal/model/qq_governance.go`
- `server/internal/repository/qq_governance.go`
- `server/internal/service/qq_governance.go`
- `server/internal/service/qq_governance_admin.go`
- `server/internal/service/qq_governance_admin_test.go`
- `server/internal/service/qq_governance_card_sync_test.go`
- `server/internal/service/qq_governance_rate.go`
- `server/internal/service/qq_governance_reconcile.go`
- `server/internal/service/qq_governance_reconcile_refresh_test.go`
- `server/internal/service/qq_governance_recovery_test.go`
- `server/internal/service/qq_governance_risk.go`
- `server/internal/service/qq_governance_test.go`
- `server/internal/service/qq_governance_worker.go`
- `server/internal/service/sys_config_qq_governance_test.go`
- `server/jobs/corporation_structure_alert_scan.go`
- `server/jobs/qq_governance.go`
- `static-react/src/api/qq-governance.test.ts`
- `static-react/src/api/qq-governance.ts`
- `static-react/src/pages/system-qq-governance-page.tsx`
- `static-react/src/types/api/qq-governance.ts`
- `static/src/api/qq-governance.ts`
- `static/src/views/system/qq-governance/index.vue`

## 数据库人工清理顺序

1. 确认所有旧进程、旧部署副本和 NapCat 连接已停止；撤销旧机器人令牌及网络入口，避免旧实例继续发送。记录发布版本、备份时间和负责人。
2. 使用受限运维账号，备份 PostgreSQL 完整 schema、下列旧表数据、相关配置及指定任务的 schedule/execution 记录；记录行数、索引定义和校验和。备份含 QQ 与令牌，须加密、限制访问并明确保留期限。先在隔离数据库实际验证恢复。
3. 在目标 schema 内人工核对以下精确表名及外键/视图依赖，不按 `qq%` 通配删除，不使用 CASCADE。逐表审查后才可在独立维护事务中清理。

旧表白名单（包括其主键、GORM 自动索引与表专属约束）：

- `qq_group_governance_policy`
- `qq_governance_event`
- `qq_group_member_state`
- `qq_group_runtime_snapshot`
- `qq_governance_reconcile_run`
- `qq_governance_reconcile_member`
- `qq_governance_review`
- `qq_governance_action_task`
- `qq_governance_action_log`
- `qq_governance_risk_control_state`
- `qq_governance_alert`
- `corp_structure_alert_state`（仅旧 QQ 投递去重，不是建筑业务快照）

专属手工索引：`idx_qq_governance_reconcile_run_active_key`（历史版本）、`idx_qq_governance_reconcile_run_active_group`、`idx_qq_governance_action_task_ready`、`idx_qq_governance_review_member_created`、`idx_qq_governance_action_log_type_created`、`idx_qq_governance_alert_status_created`。随所属旧表清理，无须触碰共享索引。其余索引以备份中的表归属清单为准。

4. 人工审查 `system_config` 精确键：

- `onebot.enabled`、`onebot.access_token`、`onebot.bot_qq`、`onebot.allowed_cidrs`
- `qq_governance.scan_interval_minutes`、`qq_governance.mismatch_confirmations`、`qq_governance.mismatch_observation_hours`
- `webhook.ob_target_type`、`webhook.ob_target_id`、`webhook.ob_token`、`webhook.qq_group_ids`
- `dashboard.corporation_structures_alert_enabled`、`dashboard.corporation_structures_alert_group_ids`

只有 `webhook.type` 当前值为 `onebot` 或 `qq_governance_onebot` 时，才将此旧配置的 `webhook.enabled` 关闭，并清理旧 `webhook.type`/`webhook.url`；不要修改有效的非 QQ Webhook。共享 `webhook.fleet_template` 保留，不自动改投其他渠道。旧类型会被发送与保存校验拒绝，绝不会走 Discord 兜底。

5. 在共享 `task_schedule` / `task_execution` 中仅按精确任务名核对 `qq_group_governance_reconcile`、`qq_governance_maintenance`、`corporation_structure_alert_scan`；历史执行记录可归档保留，配置可备份后人工删除。未知任务不会注册执行，不清空整个任务表。
6. Redis 历史限流键只允许按旧实现确认的实例前缀逐项核对并清理；禁止 FLUSHDB/FLUSHALL。历史键无需清理即可运行新版本，不恢复其旧限流状态。
7. 清理后复核共享表行数、用户 QQ 唯一性、舰队、建筑快照/授权/燃料、非 QQ Webhook 配置；确认新进程不会重建旧表或执行旧任务。

## 恢复要求

维护事务提交前可回滚。提交后从已验证备份恢复旧表 schema、数据、索引和指定配置/任务记录到隔离环境，核对行数与校验和后才可批准恢复目标库。恢复旧数据不应自动恢复机器人凭据、连接或重放旧队列；如需回退代码，应先保持机器人及发送配置关闭，重新确认积压动作有效性并单独授权启用。共享业务数据不在本方案删除或恢复范围内，禁止用全库覆盖恢复破坏清理后产生的新业务数据。

## 搜索残留分类

- 运行实现：不保留 OneBot/治理/QQ 发送调用、代理、配置契约和专属扫描。
- 回归断言：可包含已删除的路径、类型和任务名，用于确认拒绝旧类型、不注册旧路由及任务。
- 共享 QQ：用户联系方式、唯一索引、招新登记/链接、导师/订单/福利联系方式和前端技术支持群文案继续保留，均不发送 QQ 消息。
- 文档：本清单、当前下线边界、第二阶段业务场景、明确标为历史的旧草案及 CHANGELOG 保留历史名称，不代表运行能力。

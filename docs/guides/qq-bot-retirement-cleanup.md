---
status: active
doc_type: guide
owner: engineering
last_reviewed: 2026-11-22
source_of_truth:
  - scripts/qq-bot-retirement.sql
  - scripts/qq-bot-retirement-redis.sh
---

# QQ Bot 下线：显式清理与恢复

[OneBot / NapCat 接入已在 2.0.0 下线](../../CHANGELOG.md#200---2026-11-23)，但不会自动删除旧数据。本指南只针对旧接入，不影响 [QQ 官方机器人](../features/current/qq-official.md)。PostgreSQL 和 Redis 脚本互相独立，默认仅预览；只有另行批准后才能应用。代码变更本身不执行清理。

## 准备与备份

1. 确认已部署下线版本，停止所有仍能创建或使用 QQ Bot 数据的旧实例、定时任务及 NapCat 连接；移除代理入口并吊销旧令牌。确认共享 Webhook 配置在维护窗口内不会被修改。
2. 逐项核对目标 PostgreSQL 主机、端口、数据库、schema 和 Redis 实例、端口、逻辑 DB；不要沿用应用默认连接。记录版本、时间、负责人和经批准的目标。
3. 用 PostgreSQL 客户端的安全凭据机制（如 `.pgpass`）导出受保护的目标库归档。示例变量须预先设置且非空；不要把令牌、密码或原始事件打印到日志。

   ```bash
   umask 077
   pg_dump -h "${PG_HOST:?}" -p "${PG_PORT:?}" -U "${PG_USER:?}" \
     -d "${PG_DB:?}" -Fc -f "${BACKUP_DIR:?}/qq-bot-before.dump"
   sha256sum "${BACKUP_DIR:?}/qq-bot-before.dump"
   pg_restore --list "${BACKUP_DIR:?}/qq-bot-before.dump"
   ```

   归档须包含目标 schema、专属表及所属索引/约束、`system_config`、任务计划及历史。加密、限制访问并设置保留期限。在**隔离数据库**完成 `pg_restore --exit-on-error` 演练，核对表行数、配置/任务记录和依赖后再批准清理；不要只验证归档可列出目录。
4. Redis 限流键是短期运行状态，通常不必恢复。若运维要求回滚能力，应在预览前保留受保护的 Redis 快照或指定键的 `DUMP`/TTL 备份，在隔离实例验证恢复。不要在共享目标实例全量导入 RDB，也不要恢复过期桶或让旧 worker 重新创建它们。

## PostgreSQL：预览与应用

```bash
psql -X -h "${PG_HOST:?}" -p "${PG_PORT:?}" -U "${PG_USER:?}" -d "${PG_DB:?}" \
  -v target_database="${PG_DB:?}" -v target_schema="${PG_SCHEMA:?}" \
  -f scripts/qq-bot-retirement.sql
```

脚本验证当前数据库与 `target_database` 一致，且目标 schema 存在。预览使用只读事务，输出表名、行数及处置动作，不输出配置值、URL、凭据、QQ 号或消息。精确 `count(*)` 在大表上可能较慢。核对对象、类型、依赖和下列清单后，另行批准并在同一命令中增加 `-v apply=true`。应用在单一事务中执行；短锁超时、意外对象类型或外部依赖导致中止，不使用 `CASCADE`。仅检查预览不能保证提交成功，尤其不能代替依赖审查。缺失的白名单对象会跳过，因此成功后可再次预览。

下线前最后一版的 16 张专属表：

| 范围 | 精确表名 |
| --- | --- |
| 统一调度（6） | `qqbot_operation`、`qqbot_attempt`、`qqbot_inbox`、`qqbot_delivery`、`qqbot_circuit`、`qqbot_setting` |
| 通知与建筑（4） | `qq_notification_channel`、`qq_notification`、`qq_notification_target`、`qq_structure_alert` |
| 群业务（6） | `qq_group_rule`、`qq_group_application`、`qq_group_member`、`qq_group_applied_event`、`qq_group_inspection`、`qq_group_inspection_item` |

12 张历史专属表：`qq_group_governance_policy`、`qq_governance_event`、`qq_group_member_state`、`qq_group_runtime_snapshot`、`qq_governance_reconcile_run`、`qq_governance_reconcile_member`、`qq_governance_review`、`qq_governance_action_task`、`qq_governance_action_log`、`qq_governance_risk_control_state`、`qq_governance_alert`、`corp_structure_alert_state`。最后一张仅是旧 QQ 投递去重，不是建筑业务快照。表内索引和专属约束随表删除，不单独操作共享索引。

`system_config` 仅删除下列专属键：

- `onebot.enabled`、`onebot.access_token`、`onebot.bot_qq`、`onebot.allowed_cidrs`
- `qq_governance.scan_interval_minutes`、`qq_governance.mismatch_confirmations`、`qq_governance.mismatch_observation_hours`
- `webhook.ob_target_type`、`webhook.ob_target_id`、`webhook.ob_token`、`webhook.qq_group_ids`
- `dashboard.corporation_structures_alert_enabled`、`dashboard.corporation_structures_alert_group_ids`

只有同一事务内确认 `webhook.type` 为 `onebot` 或 `qq_governance_onebot`，才写入 `webhook.enabled=false` 并删除 `webhook.type` / `webhook.url`。其他 Webhook 类型的 URL、开关与配置不变；`webhook.fleet_template` 始终保留，不自动改投其他渠道。

只清理 `task_schedules`（若存在历史单数 `task_schedule` 也同样处理）中以下任务的计划覆盖：`qq_group_business`、`qq_structure_alert`、`qq_group_governance_reconcile`、`qq_governance_maintenance`、`corporation_structure_alert_scan`。不删除任何 `task_executions` / `task_execution` 历史，不清空共享表。

## Redis：预览与应用

```bash
bash scripts/qq-bot-retirement-redis.sh --host "${REDIS_HOST:?}" \
  --port "${REDIS_PORT:?}" --db "${REDIS_DB:?}"
```

仅支持 Redis 7+ 独立实例（含对应版本的 `redis-cli`），须显式指定主机、端口、逻辑 DB。认证使用 `REDISCLI_AUTH`，可传 `--user USER`；TLS 可传 `--tls --cacert FILE`。预览使用只读脚本，只输出匹配类别和数量，不输出群号、单个群键或值。复核后另行批准并追加 `--apply`。

只匹配 `qqbot:rate:global` 和完整的 `qqbot:rate:group:<ASCII 十进制数字>` 键。分批 `SCAN` 后对符合精确条件的键逐项 `UNLINK`，不执行 `KEYS`、`FLUSHDB`、`FLUSHALL`，不清理其他前缀或逻辑 DB。`SCAN` 不是一致性快照、可能重复计数；应用输出实际删除数。整个扫描不跨批原子，若中断请先确认目标与原因，再重新预览和批准重试。

## 复核与恢复

- 再跑两个预览，确认已清理目标不存在或计数为零。复核任务执行历史、用户 QQ 唯一性、舰队和模板、建筑快照/授权/燃料及非 QQ Webhook 配置不变；应用启动不应重建专属表、键或任务。
- PostgreSQL 事务提交前可回滚。提交后须先在隔离环境恢复备份并校验，再经审批选择性恢复专属表、数据、索引及指定配置/任务行。合并共享表时逐行核对当前值，不能用旧数据覆盖清理后产生的业务记录。恢复历史任务执行数据不在本次清理的回滚范围内（它们没有删除）。
- Redis 如确需恢复，先从备份还原到隔离实例；按精确键、原 DB 和剩余 TTL 复核，仅恢复获批的未过期键，不覆盖新值或全库。恢复两类存储的数据都不等于恢复机器人：不得自动启用旧凭据、接回 NapCat 或重放待发送队列。

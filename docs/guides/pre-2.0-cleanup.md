---
status: active
doc_type: guide
owner: engineering
last_reviewed: 2026-11-26
source_of_truth:
  - scripts/cleanup-pre-2.0.sql
  - scripts/cleanup-pre-2.0-extra.sql
  - scripts/qq-bot-retirement.sql
  - server/bootstrap/db.go
  - server/bootstrap/salary_migration.go
  - server/bootstrap/fuxi_hall_migration.go
---

# 手动清理 2.0.0 以前的残留

这是 **PostgreSQL 升级后的清理脚本**，不能代替启动迁移。它只在显式指定的 schema 内按白名单清理，不按日期或名称前缀扫库。旧 OneBot 的 Redis 残留需单独处理。

## 准备与备份

1. 升级前先备份。核对 PostgreSQL 主机、端口、数据库、schema 与维护账号，使用 `.pgpass` 等机制提供凭据，不把密码放在命令或日志中。部署并启动 2.0.0 或更新版本，确认启动迁移完成、业务验证通过。
2. 进入维护窗口，停止所有连接目标库的 AmiyaEden 实例、后台任务及其他可能写入目标 schema 的进程。清理前再备份一次，每次都使用新的 `BACKUP_FILE` 和已有的受限目录。备份整个数据库，以保留目标表的依赖、数据、索引与约束：

   ```bash
   umask 077
   pg_dump -h "${PG_HOST:?}" -p "${PG_PORT:?}" -U "${PG_USER:?}" -d "${PG_DB:?}" \
     -Fc -f "${BACKUP_FILE:?}" &&
   sha256sum "${BACKUP_FILE:?}" > "${BACKUP_FILE:?}.sha256" &&
   pg_restore --list "${BACKUP_FILE:?}"
   ```

3. 在独立的测试 PostgreSQL 实例上新建空库并验证恢复。先核对 `RESTORE_*` 都指向隔离实例，不能指向业务库。演练使用隔离环境自己的账号和权限，不套用归档中的所有者与授权：

   ```bash
   sha256sum -c "${BACKUP_FILE:?}.sha256" &&
   createdb -h "${RESTORE_HOST:?}" -p "${RESTORE_PORT:?}" -U "${RESTORE_USER:?}" \
     -T template0 -- "${RESTORE_DB:?}" &&
   pg_restore --exit-on-error --single-transaction --no-owner --no-privileges \
     -h "${RESTORE_HOST:?}" -p "${RESTORE_PORT:?}" -U "${RESTORE_USER:?}" \
     -d "${RESTORE_DB:?}" "${BACKUP_FILE:?}"
   ```

   核对恢复出的目标 schema、表行数、关键配置和依赖，确认备份可用后再继续。归档含业务数据，应加密保存、限制访问并设置保留期限。

## 预览与应用

从仓库根目录通过独立 `psql` 进程运行，不要加 `--single-transaction` 或从已有事务中包含脚本。默认使用只读事务，仅报告动作和计数，不提交数据变更。预览会执行精确计数，大表上可能较慢。

```bash
psql -X -h "${PG_HOST:?}" -p "${PG_PORT:?}" -U "${PG_USER:?}" -d "${PG_DB:?}" \
  -v target_database="${PG_DB:?}" -v target_schema="${PG_SCHEMA:?}" \
  -f scripts/cleanup-pre-2.0.sql
```

核对预览清单、目标对象的依赖和已验证的备份。复核后，在同一条命令中增加 `-v apply=true` 才会提交。应用使用同一个事务，获取所需排他锁；若遇到锁超时、异常对象结构、行级安全过滤或外部依赖则中止，不会用 `CASCADE` 强制删除。预览成功不能替代依赖审查。

### 会清理什么

- [OneBot / NapCat 下线清理白名单](qq-bot-retirement-cleanup.md#postgresql预览与应用)内的旧专属表、配置和计划；保留旧 QQ 任务执行记录。本脚本已包含该 PostgreSQL 清理，不必再单独运行。
- 启动迁移已标记为退役的旧队长绑定、资格缓存、归因和结算表，以及两个旧队长任务的计划和执行记录；还有旧抽奖、兑换码、补损发放邮件日志、人物通知表及明确退役的列。启动时已删除的对象会跳过。
- 旧新人和导师配置；新运营薪资配置迁移齐全后才移除 `pap.admin_award`。旧 Mumble 地址和端口仅在 `mumble.public_nodes` 已存在时清理，否则保留供人工核对。
- 名人堂新页面已存在时，只移除**没有任何卡片引用**的旧 `leadership` / `contributors` 页面，保留所有卡片及其引用的页面。

清理不会改动账号及联系方式、权限映射、钱包余额、历史流水、订单、旧入团工单、审计、非 OneBot Webhook、QQ 官方机器人或仍在使用的舰队与建筑数据，也不会重跑业务数据迁移。定制表、附件、旧日志和没有可靠退役依据的数据不自动处理。

曾部署 OneBot / NapCat 的实例，还应停用旧连接并按[下线清理指南](qq-bot-retirement-cleanup.md#redis预览与应用)单独预览 Redis 残留；PostgreSQL 脚本不会连接或清空 Redis。

## 复核与恢复

- 应用后再次预览，确认没有意外遗漏，再启动 2.0.0+ 实例并验证业务。脚本可以重复运行，仍有引用的页面或未迁移的 Mumble 设置会继续保留。
- 如果应用失败，事务不会提交。先检查错误及依赖再重新预览；不要通过扩大删除范围绕过保护。
- 如果提交后需恢复，先按上文把清理前备份还原到隔离库，核对受影响对象和当前业务库里的新数据。经审核后，在维护窗口内选择性恢复获批的表、列、约束及指定记录。恢复共享表时逐行比较当前值，不要整表覆盖，也不要把旧库还原到正在使用的数据库。

## 在隔离库验证脚本

下面的测试会创建 `cleanup_v2_fixture` 和 `cleanup_v2_other` schema 并留下测试数据。**仅在可丢弃的测试库**运行，验证默认预览不写入、应用可重复执行，以及受保护的记录和其他 schema 未改变。

```bash
psql -X -d "${TEST_DATABASE:?}" -v target_database="${TEST_DATABASE:?}" \
  -v disposable=true -f scripts/tests/cleanup-pre-2.0.sql
```

---
status: active
doc_type: guide
owner: engineering
last_reviewed: 2026-11-24
source_of_truth:
  - scripts/cleanup-pre-2.0.sql
  - scripts/cleanup-pre-2.0-extra.sql
  - scripts/qq-bot-retirement.sql
  - server/bootstrap/db.go
  - server/bootstrap/salary_migration.go
  - server/bootstrap/fuxi_hall_migration.go
---

# 手动清理 2.0.0 以前的残留

这是**升级后的清理脚本**，不能代替启动迁移。它按白名单找退役对象，不按日期删除记录。未列入白名单的对象不会因为名称相似或版本更旧而被删除。

## PostgreSQL

1. **升级前备份。** 核对 PostgreSQL 主机、端口、数据库和 schema。备份当前库并在隔离库验证可恢复，方法见 [QQ Bot 清理指南](qq-bot-retirement-cleanup.md#准备与备份)。
2. 部署并启动 2.0.0 或更新版本，确认启动迁移完成、业务验证通过。清理前再备份一次，停掉所有连接目标库的 AmiyaEden 实例、旧任务和 NapCat，避免维护期间重写目标对象或配置。
3. 从仓库根目录运行下列命令。使用 `.pgpass` 等凭据机制，先只预览并核对清理清单、待删对象的依赖和备份。预览会执行计数，大表上可能较慢。

   ```bash
   psql -X -h "${PG_HOST:?}" -p "${PG_PORT:?}" -U "${PG_USER:?}" -d "${PG_DB:?}" \
     -v target_database="${PG_DB:?}" -v target_schema="${PG_SCHEMA:?}" \
     -f scripts/cleanup-pre-2.0.sql
   ```

4. 复核后，在同一条命令中增加 `-v apply=true` 才会提交。所有 PostgreSQL 操作在同一个事务中；意外依赖、锁超时或 SQL 错误会使事务失败，不使用 `CASCADE` 强制删除。应用后再次预览，再启动 2.0.0+ 实例并验证业务。若需恢复，先在隔离库验证备份，再选择性恢复；不要用旧库覆盖升级后新增的业务数据。

### 会清理什么

- [QQ Bot 白名单](qq-bot-retirement-cleanup.md#postgresql预览与应用)内的旧专属表、配置和计划，不删除 QQ 任务执行历史。不用再单独跑 QQ Bot 的 PostgreSQL 脚本。
- 启动迁移已标记为退役的旧队长绑定、资格缓存、归因和结算表，以及两个旧队长任务的计划和执行记录；还有旧抽奖、兑换码、补损发放邮件日志、人物通知表及明确退役的列。启动时已经删除的对象会跳过。
- 旧新人和导师配置；新运营薪资配置迁移齐全后才移除 `pap.admin_award`。旧 Mumble 地址和端口仅在 `mumble.public_nodes` 已存在时清理，否则保留供人工核对。
- 名人堂新页面已存在时，只移除**没有任何卡片引用**的旧 `leadership` / `contributors` 页面，保留所有卡片及其引用的页面。

保留用户账号及联系方式、权限映射、钱包余额、历史流水、订单、旧入团工单、审计、非 QQ Webhook 或仍在使用的舰队与建筑数据。不重跑联系方式、角色映射、入团分类等业务迁移。更早部署的定制表、旧附件、旧日志和没有可靠退役依据的数据不自动处理。

## Redis 和部署残留

- 如以前启用过 QQ Bot，按 [Redis 清理步骤](qq-bot-retirement-cleanup.md#redis预览与应用)预览并清除专属限流键。这一步和 PostgreSQL 分开执行，不清空 Redis 实例，也不影响其他缓存和会话。
- 断开旧 NapCat 与本实例的连接，移除专用于此实例的 OneBot 反代路由并吊销旧令牌。容器、镜像、日志和卷因部署而异：先确认确实属于已退役服务、没有其他实例引用且已有备份，再逐项清理。不要执行全局 Docker 清理或删掉 PostgreSQL、Redis、上传、日志的共享持久卷。

## 在隔离库验证脚本

下面的测试会创建 `cleanup_v2_fixture` schema 并留下测试数据。**仅在可丢弃的测试库**运行，验证预览不写入、应用可重复执行，以及受保护的记录未改变。

```bash
psql -X -d "${TEST_DATABASE:?}" -v target_database="${TEST_DATABASE:?}" \
  -v disposable=true -f scripts/tests/cleanup-pre-2.0.sql
```

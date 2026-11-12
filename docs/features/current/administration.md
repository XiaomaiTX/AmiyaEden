---
status: active
doc_type: feature
owner: engineering
last_reviewed: 2026-11-12
source_of_truth:
  - server/internal/model/system_identity.go
  - server/internal/router/router.go
  - server/internal/service/role.go
  - server/internal/service/user.go
  - server/internal/service/auto_role.go
  - server/internal/service/alliance_pap.go
  - server/internal/service/sys_webhook.go
  - server/internal/service/blacklist.go
  - server/internal/service/blacklist_source.go
  - server/internal/service/blacklist_esi.go
  - server/internal/handler/sys_config.go
  - server/internal/utils/allow_corporations.go
  - static/src/constants/system-identity.ts
  - static/src/api/sys-config.ts
  - static/src/api/system-manage.ts
  - static/src/api/webhook.ts
  - static/src/views/system
---

# System 管理模块

## 当前能力

- 固定系统标识读取
- 系统职权定义只读查询
- 用户管理、用户职权分配
- 管理员可维护普通用户昵称与状态
- `super_admin` 可在管理端维护非 `super_admin` 用户的 QQ / Discord ID；其他场景下联系方式仍由用户本人通过 `/api/v1/me` 维护
- 用户管理列表默认按最后登录时间倒序，并支持按昵称、QQ、任意已绑定人物名搜索
- 用户管理列表可展开每个用户行，查看该用户全部已绑定人物的头像、人物 ID、人物名、ESI 状态与人物总技能点
- 用户管理展开人物列表在人物名右侧提供共享内联复制按钮，便于复制已绑定人物名
- `super_admin` 可在 `/system/user` 切换“已失效人物 ESI 限制”，决定是否强制用户在任一已绑定人物 ESI 失效时停留在人物管理页
- 超级管理员模拟登录
- 自动权限映射（ESI corp roles / corp titles -> system roles）
- 联盟 PAP 列表、抓取、导入、月度归档（钱包兑换暂不启用）
- 军团 PAP 兑换汇率配置（Skirmish / Strategic / CTA 三种类型，外加 FC 工资、每月工资上限与人工发放奖励，影响舰队 PAP 发放与福利/商城人工发放时的钱包行为）
- Webhook 配置与测试
- 联盟黑名单：从联盟公开 Google Sheet 同步本地黑名单，供管理员浏览与搜索（见下文“联盟黑名单”）

## Webhook 通知类型

支持的 `webhook.type` 值：

- `discord` — Discord 频道 Webhook，URL 走 HTTPS 域名白名单
- `feishu` — 飞书自定义机器人
- `dingtalk` — 钉钉自定义机器人
QQ Bot 通知已下线；旧类型配置需人工清理，不会映射到其他通知渠道。

## 入口

### 前端页面

- `static/src/views/system/basic-config`
- `static/src/views/system/user`
- `static/src/views/system/auto-role`
- `static/src/views/system/pap`
- `static/src/views/system/pap-exchange`
- `static/src/views/system/webhook`
- `static/src/views/system/blacklist`（React：`static-react/src/pages/system-blacklist-page.tsx`）

### 后端路由

- `/api/v1/system/basic-config`
- `/api/v1/system/sde-config`
- `/api/v1/system/basic-config/allow-corporations`
- `/api/v1/system/basic-config/character-esi-restriction`
- `/api/v1/system/role/definitions`
- `/api/v1/system/user/*`
- `/api/v1/system/auto-role/*`
- `/api/v1/system/pap/*`
- `/api/v1/system/pap-exchange/*`
- `/api/v1/system/webhook/*`
- `/api/v1/system/blacklist`、`/api/v1/system/blacklist/refresh`

## 权限边界

- `/system/*` 默认要求 `admin`
- `/system/basic-config` 页面及 `/api/v1/system/basic-config*`、`/api/v1/system/sde-config` 接口仅 `super_admin` 可见且可用
- `/system/auto-role/*` 额外要求 `super_admin`
- `/system/webhook/*` 额外要求 `super_admin`
- `/system/user/:id/impersonate` 额外要求 `super_admin`
- `/system/user/:id/impersonate` 在目标用户主人物 ESI 已失效时拒绝签发模拟登录 token
- `/system/user/:id` 仅 `super_admin` 可修改非 `super_admin` 目标的 QQ / Discord ID；`admin` 仍不可编辑其他 `admin`
- `/system/user/:id` 删除用户时，若目标用户已登记 QQ 或 Discord ID，则仅 `super_admin` 可执行
- `GET /system/role/definitions` 仅用于前端加载系统职权定义，属于只读数据源
- `GET /system/basic-config` 仅返回固定系统标识，不提供写接口
- `/system/blacklist*` 额外要求 `system.manage` 军团能力（`super_admin` 直接通过）；“更新黑名单”按钮与页面权限相同，不另设按钮权限码。数据来自公开表格，不按机密处理；招新审核另向审核人显示申请人的命中条目，见[军团申请](corporation-admissions.md)

## 关键不变量

For role assignment rules and super_admin protection, see `docs/architecture/auth-and-permissions.md`. Below are invariants specific to the administration module:

- 系统军团 ID 与网站标题由代码常量提供，当前不通过数据库、API 或 UI 修改
- 基础配置页不再允许编辑军团 ID 或网站标题
- 管理员侧用户列表 `/api/v1/system/user` 的职权列只以有序 `roles[]` 为准，不再暴露历史单值 `role`
- 管理员侧用户列表同时返回该用户全部已绑定人物及每个人物的 `total_sp` 快照与 `token_invalid` 状态
- 管理员侧用户列表支持单职权筛选；职权匹配只以当前 `user_role` 关联为准
- `allow_corporations` 配置存储在数据库 `system_config` 表（键名 `app.allow_corporations`），运行时总会强制包含伏羲军团 Fuxi Legion（`98185110`）
- 失效人物 ESI 页面停留限制存储在数据库 `system_config` 表（键名 `auth.enforce_character_esi_restriction`），默认开启
- 联盟 PAP 的管理接口与用户查看接口分属不同模块
- Webhook 是系统配置能力，不应散落到页面里直接拼接

## 联盟黑名单

黑名单用于“核查”：给定一个 character_id、QQ 或 Discord，用一次等值查询判定是否在黑名单并取得理由。不记录主号与小号关系，也不记录联系方式属于哪个人物。

招新审核读取本地黑名单比对待处理申请，命中时不能登记“已邀请”；比对范围与可见性见[军团申请](corporation-admissions.md)。审核路径与本页共用 `normalizeBlacklistQQ` / `normalizeBlacklistDiscord`，保证申请人的值与条目按同一规则规范化。

### 来源

- 来源是代码常量：公开 Google Sheet `1nKtgFowqPEI7I1qzuXnKIFtNZurpKw7K5mZmCY4AHFg` 的 “The List” 页签（gid `0`），以 CSV 导出地址下载；不经数据库或 UI 修改，其他页签不导入。理由：列布局与该表强绑定，换表通常也要改解析代码。
- 下载使用独立 HTTP client：30 s 超时、响应体上限 10 MiB、跟随重定向。
- 表头在前 5 行内查找：A 列以 `Main` 开头，D/F/G/H/I/J 分别含 `Reason`、`alt`、`alt`、`alt`、`Discord`、`QQ`（不区分大小写）。表头之后每一行都是数据行。

### 数据形状

- `blacklist_entry`：一行一个核查值，`kind` 为 `character` / `qq` / `discord`，`(kind, value)` 唯一。三类是相互独立的核查集合：
  - 人物：A 与 F/G/H 列人名（按换行和逗号切分）经 ESI 解析出的 character_id；`display` 为 ESI 规范名。
  - QQ：J 列按空白、`,`、`;`、`/` 切分，只保留 `^[1-9][0-9]{4,11}$` 的数字串，其余碎片丢弃。
  - Discord：I 列按换行切分，去首尾空白与开头 `@`；`value` 为小写，`display` 保留首次出现的写法。不按逗号或空格切分，因为旧式用户名可以含空格。
- QQ 与 Discord 取自每一个数据行，包括只有联系方式的行和人名无法解析的行。
- 同一 `(kind, value)` 出现在多行时只保留一条，各行非空理由按出现顺序去重后以换行合并。
- QQ 只做精确匹配：列表关键字为纯数字时按 `value` 精确匹配人物 ID 与 QQ，否则在人物名与 Discord 的 `display` 中做不区分大小写的子串匹配。理由：子串匹配会让 `12345` 误命中 `123456`。
- `blacklist_unresolved_name`：无法解析的人名缓存（小写唯一），`not_found` 为 ESI 未返回该人物，`invalid_format` 为不符合人物名格式（长度 3–37，只含字母、数字、空格与 `.'_-`）。
- `blacklist_refresh_run`：每次成功刷新一行，记录操作人、完成时间与计数（含本次查询失败或超时延后的名字数 `lookup_failed`，历史记录默认 0）；页面显示最新一行。

### 刷新

- `POST /system/blacklist/refresh` 同步执行，服务端 90 s context，两端前端只对这一个请求放宽到 120 s。
- 完整刷新成功（`lookup_failed = 0`）后最小间隔为 24 小时。服务端持锁检查最近成功刷新记录，冷却中返回 `blacklist.errors.refreshCooldown`，不下载表格、不访问 ESI、不写入刷新记录或审计；限制跨操作人和进程重启生效。有延后名字时允许立即重试，失败刷新不启动新冷却。摘要 `next_refresh_at` 为完整刷新完成时间加 24 小时，部分完成为 `null`；两端显示下次时间，并在冷却期间禁用按钮，到时恢复。
- 名字解析：已有人物条目的 `display`（小写）直接复用其 ID；命中未解析缓存的跳过（`skipped_unresolved`）；其余每批 50 个调用 ESI `POST /universe/ids/`，只采信 `characters` 中小写名完全相等的结果。理由：大批次容易出现网关超时，且该接口会在 `corporations` / `alliances` 中返回非精确匹配。没有新名字或延后名字时，稳态刷新不访问 ESI。
- ESI 容错：5xx、429、网络错误（含单次请求超时）最多重试 2 次，间隔 1 s、2 s；其他 4xx 不重试。重试后仍失败的批次延后到下次刷新，不写入未解析名缓存。全部批次与退避共享 60 s 查询预算，预算耗尽时取消当前请求并延后剩余名字；继续写入已知人物、成功解析的人物、所有 QQ 与 Discord，并移除表格已删除的条目。已知人物直接复用本地 ID，不依赖本次 ESI，因此查询失败不会误删仍在表上的已知人物。不增加后台任务。
- 摘要、刷新记录与审计 details 包含 `lookup_failed`；大于 0 时，两端摘要和成功提示均显示“N 个人名本次查询失败，下次刷新重试”，支持中英文。首次全量同步可能需要多次刷新补齐人物。
- 未解析名只要还在表上就不再查询；名字从表上消失后缓存随之删除，因此修正拼写后会重新解析。
- 对齐：本地结果与表格完全一致，表上消失的条目被删除，理由或展示名变化的条目被更新。
- 失败护栏，任一命中都不写库、保留原数据：
  - 下载失败、非 200 或超限：`blacklist.errors.sourceUnavailable`
  - 找不到表头：`blacklist.errors.sourceLayoutChanged`
  - 结果中没有任何人物条目：`blacklist.errors.emptyResult`
- 下载与 ESI 在事务外完成。条目、未解析名、刷新记录与审计事件（`category=blacklist`、`action=blacklist_refresh`，details 只含计数，不含 QQ / Discord）在同一事务内写入；审计失败则整体回滚。
- 进程内 `sync.Mutex.TryLock` 保证同一时间只有一次刷新，冲突返回 `blacklist.errors.refreshInProgress`。当前为单实例部署；多实例并发时两次刷新都对齐到同一份表格，最终状态一致。

## 主要代码文件

- `server/internal/service/role.go`
- `server/internal/service/user.go`
- `server/internal/service/auto_role.go`
- `server/internal/service/alliance_pap.go`
- `server/internal/service/pap_exchange.go`
- `server/internal/service/sys_webhook.go`
- `server/internal/service/blacklist.go`、`blacklist_source.go`、`blacklist_esi.go`
- `server/internal/repository/blacklist.go`、`server/internal/model/blacklist.go`、`server/bootstrap/blacklist_migration.go`
- `static/src/api/system-manage.ts`
- `static/src/api/pap-exchange.ts`
- `static/src/api/webhook.ts`
- `static/src/views/system`

## 前端实现映射（迁移期）

- Vue 当前实现位于 `static/src`。
- React 已承接系统用户、任务、钱包、审计、PAP、自动角色、Webhook 和基础配置页面，入口与状态以 `docs/specs/draft/frontend-react-migration-plan/migration-scope-baseline.md` 为准。
- 本文只定义管理行为；Vue/React 的迁移状态不在此重复维护。

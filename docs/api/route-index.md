---
status: active
doc_type: api
owner: engineering
last_reviewed: 2026-11-26
source_of_truth:
  - contracts/openapi.yaml
  - contracts/modules
  - server/internal/apicontract/api.gen.go
  - server/internal/router/router.go
---

# API 路由索引

## 说明

站内 `/api/v1` 的接口只在 [OpenAPI 契约](../../contracts/openapi.yaml)里定义。路径、方法、参数、响应和访问链都以契约为准，服务启动时由生成路由注册；本文件不复制这些内容，只说明两件事：

- 某个业务域的接口去哪个契约片段看；
- 哪些入口不在 OpenAPI 里。

业务规则以对应功能文档和 service 代码为准。文档与代码不一致时改文档，不要为了迁就旧文档改代码。

## 接口定义在哪

- [OpenAPI 根规范](../../contracts/openapi.yaml)：通用响应、安全方案、跨模块 schema，以及公告、SDE、`/me`、Mumble 凭据等接口。
- [业务模块](../../contracts/modules/)：按业务域拆分的 `paths` 与 `components`，合并生成时不允许重复。
- 生成：`pnpm -C contracts api:generate`；校验生成物：`pnpm -C contracts api:check`。
- 访问链：每个操作用 `x-gin-access` 声明（`jwt`、`login`、`role:`、`cap:`），规则见[路由与绑定](conventions.md#路由与绑定)。所有权等业务授权在 service 内，不在这里重复。
- 生成路由：[api.gen.go](../../server/internal/apicontract/api.gen.go)，请勿手改。

## 不在 OpenAPI 里的入口

### 私有 Mumble 身份接口

以下入口位于 `/api/internal/mumble/v1`，仅接受 `mumble.service_token` 对应的 Bearer 服务凭据；普通 Seat JWT 一律拒绝。`/api/internal` 是服务间路由分类，不面向浏览器用户。生产部署还应使用私网 HTTPS，并推荐 mTLS。行为细节见 [Mumble 外部身份](../architecture/mumble-external-identity.md)。

| Method | Path                                         | 说明                                                                                                                                                                             |
| ------ | -------------------------------------------- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| POST   | `/api/internal/mumble/v1/authenticate`       | 验证主人物名和 Mumble App Password；`server_instance_id` 可选且当前仅供调用方标识；成功返回稳定 Mumble User ID、canonical name 与 `fuxi_*` runtime groups；失败统一返回业务 deny |
| POST   | `/api/internal/mumble/v1/identities/resolve` | 最多批量解析 500 个稳定 ID 和/或 canonical name，供在线会话重验与协议身份查找；返回当前资格、名称与 runtime groups                                                               |

### 静态文件

`GET /uploads/*filepath`（含 `HEAD`）由 [router.go](../../server/internal/router/router.go) 挂载，内容来自部署目录 `./uploads`，不经过 OpenAPI 契约。

## 业务域索引

| 业务域                           | 契约片段                                                                                                                                                                                                 | 行为文档                                                                                                                                                |
| -------------------------------- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------- |
| 登录、SSO、绑定人物、注册        | [sso.yaml](../../contracts/modules/sso.yaml)                                                                                                                                                             | [账号与人物](../features/current/auth-and-characters.md)                                                                                                |
| 当前用户、Mumble 凭据            | [openapi.yaml](../../contracts/openapi.yaml)                                                                                                                                                             | [账号与人物](../features/current/auth-and-characters.md)、[Mumble 外部身份](../architecture/mumble-external-identity.md)                                |
| 用户、职权、模拟登录             | [users.yaml](../../contracts/modules/users.yaml)                                                                                                                                                         | [系统管理](../features/current/administration.md)                                                                                                       |
| 入团申请与审核                   | [admissions.yaml](../../contracts/modules/admissions.yaml)                                                                                                                                               | [军团申请](../features/current/corporation-admissions.md)                                                                                               |
| 个人钱包、技能、资产、合同       | [info-character.yaml](../../contracts/modules/info-character.yaml)、[info-assets-contracts.yaml](../../contracts/modules/info-assets-contracts.yaml)                                                     | [信息与报表](../features/current/info-and-reporting.md)                                                                                                 |
| 装配、NPC 刷怪                   | [info-fittings-npc.yaml](../../contracts/modules/info-fittings-npc.yaml)                                                                                                                                 | [信息与报表](../features/current/info-and-reporting.md)、[NPC 刷怪](../features/current/npc-kills.md)                                                   |
| 舰队、舰队配置、联盟 PAP         | [fleets.yaml](../../contracts/modules/fleets.yaml)、[fleet-config-admin-pap.yaml](../../contracts/modules/fleet-config-admin-pap.yaml)                                                                   | [舰队行动](../features/current/operation.md)                                                                                                            |
| PAP 兑换、工资、徽章计数         | [pap-exchange.yaml](../../contracts/modules/pap-exchange.yaml)、[badge-salary.yaml](../../contracts/modules/badge-salary.yaml)                                                                           | [PAP 兑换](../features/current/pap-exchange.md)、[工资管理](../features/current/operation-salaries.md)、[徽章计数](../features/current/badge-counts.md) |
| 技能规划、技能达标查询           | [skill-plans.yaml](../../contracts/modules/skill-plans.yaml)、[dashboard-readiness.yaml](../../contracts/modules/dashboard-readiness.yaml)                                                               | [技能规划](../features/current/skill-planning.md)、[技能达标](../features/current/dashboard-skill-readiness.md)                                         |
| 军团建筑、燃料官工资             | [corp-structures.yaml](../../contracts/modules/corp-structures.yaml)                                                                                                                                     | [军团建筑](../features/current/corporation-structures.md)                                                                                               |
| 星系生产登记                     | [galaxy-registry.yaml](../../contracts/modules/galaxy-registry.yaml)                                                                                                                                     | [星系登记](../features/current/dashboard-galaxy-registry.md)                                                                                            |
| 商店                             | [shop.yaml](../../contracts/modules/shop.yaml)                                                                                                                                                           | [商店与交易](../features/current/commerce.md)                                                                                                           |
| 钱包与福利                       | [wallet-welfare.yaml](../../contracts/modules/wallet-welfare.yaml)                                                                                                                                       | [福利](../features/current/welfare.md)、[商店与交易](../features/current/commerce.md)                                                                   |
| 新人招募、导师                   | [newbro.yaml](../../contracts/modules/newbro.yaml)、[mentor.yaml](../../contracts/modules/mentor.yaml)                                                                                                   | [新人帮扶](../features/current/newbro-support.md)、[导师系统](../features/current/mentor-system.md)                                                     |
| 工单                             | [ticket.yaml](../../contracts/modules/ticket.yaml)                                                                                                                                                       | [工单系统](../features/current/ticket-system.md)                                                                                                        |
| SRP                              | [srp.yaml](../../contracts/modules/srp.yaml)                                                                                                                                                             | [SRP](../features/current/srp.md)                                                                                                                       |
| 公告                             | [openapi.yaml](../../contracts/openapi.yaml)                                                                                                                                                             | [军团公告](../features/current/announcements.md)                                                                                                        |
| 伏羲大厅                         | [fuxi-hall.yaml](../../contracts/modules/fuxi-hall.yaml)                                                                                                                                                 | [伏羲大厅](../features/current/fuxi-hall.md)                                                                                                            |
| 任务与审计、Webhook              | [tasks.yaml](../../contracts/modules/tasks.yaml)、[audit-webhook.yaml](../../contracts/modules/audit-webhook.yaml)                                                                                       | [任务管理](../features/current/task-manager.md)、[审计日志](../features/current/audit-log.md)                                                           |
| 系统配置、自动职权、书签、黑名单 | [system-config.yaml](../../contracts/modules/system-config.yaml)、[auto-role.yaml](../../contracts/modules/auto-role.yaml)、[bookmarks-blacklist.yaml](../../contracts/modules/bookmarks-blacklist.yaml) | [系统管理](../features/current/administration.md)                                                                                                       |
| SDE 查询与配置                   | [openapi.yaml](../../contracts/openapi.yaml)、[system-config.yaml](../../contracts/modules/system-config.yaml)                                                                                           | [SDE](../features/current/sde.md)                                                                                                                       |
| 图片上传                         | [uploads.yaml](../../contracts/modules/uploads.yaml)                                                                                                                                                     | —（限制见契约 `x-gin-body` 与对应页面）                                                                                                                 |
| QQ 官方机器人与任务 | [qq-official.yaml](../../contracts/modules/qq-official.yaml)、[fleets.yaml](../../contracts/modules/fleets.yaml) | [QQ 官方机器人](../features/current/qq-official.md) |

## 维护约定

- 新增或调整接口：先改契约并重新生成；本文件只在业务域或阅读入口变化时同步，不新增路径和权限表。
- 已移除的接口不在这里保留兼容登记。
- 需要接口细节时直接打开契约片段或生成客户端，不要从本文件转抄。

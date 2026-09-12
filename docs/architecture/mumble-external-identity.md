---
status: active
doc_type: architecture
owner: engineering
last_reviewed: 2026-09-12
source_of_truth:
  - server/internal/model/mumble_identity.go
  - server/internal/service/mumble_identity.go
  - server/internal/service/mumble_revalidation.go
  - server/internal/service/sys_config.go
  - server/internal/handler/mumble.go
  - server/internal/router/router.go
---

# Mumble 外部身份权威

## 决策

Fuxi Seat 是 Mumble 用户身份、登录资格与运行时职权声明的唯一权威来源。go-mumble-server 只执行 Mumble 协议、会话、频道和 ACL；生产 external 模式不得读取 `registered_users` 作为回退，也不得持久化 Seat 职权或群组成员关系。

## 身份与凭据

每个 Seat 用户最多有一份 `mumble_identity` 投影。该表自身的独立主键是稳定 Mumble User ID，不复用 `user.id`，且永远不分配 `0`。主人物更换只改变下一次认证返回的 canonical name，不改变稳定 ID。

Mumble App Password 由 `crypto/rand` 生成，数据库仅保存 Argon2id 哈希。创建和轮换仅在响应中返回一次明文；吊销和轮换立即使旧密码失效。审计、日志、缓存与数据库均不得记录明文密码、Seat JWT 或 EVE token。

## 资格与群组

认证同时要求：用户存在且启用、至少一个非 `guest` 职权、主人物存在且归属该用户、以及已启用的 Mumble 凭据。角色和军团准入仍由既有 Seat 逻辑计算，Mumble 模块不复制 ESI 或军团策略。

成功登录返回 `fuxi_authenticated` 与每个有效角色对应的 `fuxi_role_<role>`。这些是当前 Mumble session 的 runtime claims；客户端 `Authenticate.tokens` 绝不可取得 `fuxi_*` 声明。`SuperUser` 是保留协议身份，Seat 外部用户不得通过该名称取得它。

## 外部接口与故障策略

`/api/internal/mumble/v1/*` 仅接受 Seat“系统管理 → 基础配置”中保存的 Mumble → Seat 服务令牌，绝不接受普通 Seat JWT；`/api/internal` 仅作服务间路由分类，不代表天然安全。生产部署还必须置于私网 HTTPS，并推荐 mTLS。认证接口将未知用户名、错误密码和资格不足统一为 `INVALID_CREDENTIALS`，以避免枚举。Seat 不在认证路径访问 ESI。

新登录在 Seat 不可用时必须 fail closed。在线会话由 Mumble 定期调用 batch resolve 重新拉取真值；不再合格时断开，群组变更时替换 runtime claims 并失效 ACL 缓存。Seat 在凭据、角色、账号状态或主人物变化后，使用系统设置中的 Mumble 管理地址与 Seat → Mumble 重校验令牌调用 `POST /api/internal/identity/v1/revalidate`，发送最佳努力失效信号。推送只包含稳定 Mumble User ID，不能携带新的角色或群组事实；通知失败不回滚 Seat 写操作，周期重验负责最终收敛。

## 配置与用户入口

双向服务令牌、Mumble 管理地址和回调超时均持久化在 `system_config`，由超级管理员通过“系统管理 → 基础配置”维护，不属于 Seat YAML 配置。两个方向必须使用不同令牌，并与 go-mumble-server 自身进程配置中的对应值一致。

普通用户在“EVE 人物管理”页面创建、轮换或吊销 Mumble 凭据。登录用户名始终显示为当前主人物名；明文密码只在创建或轮换成功后显示一次。

## 对端实现

配套 go-mumble-server 已实现 external-only 认证、严格 HTTP timeout、runtime external groups、`fuxi_*` 保留命名空间保护、批量周期重验、Seat 失效回调和 `SuperUser` 显式授权。其 local 模式仍可独立运行，但 external 模式不得回退到本地注册用户。

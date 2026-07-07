---
status: active
doc_type: index
owner: engineering
last_reviewed: 2026-07-07
source_of_truth:
  - server/internal/router/router.go
---

# Feature Docs

## 说明

`docs/features/current/` 只描述当前仓库已经落地、可以从代码直接找到入口的模块行为。

如果一个想法还没有完整接入路由、页面、任务或服务，请写进 `docs/specs/draft/`，不要写进这里。

## 当前模块

- [auth-and-characters.md](current/auth-and-characters.md)
- [administration.md](current/administration.md)
- [audit-log.md](current/audit-log.md)
- [badge-counts.md](current/badge-counts.md)
- [commerce.md](current/commerce.md)
- [corporation-access-policy.md](current/corporation-access-policy.md)
- [corporation-structures.md](current/corporation-structures.md)
- [dashboard-galaxy-registry.md](current/dashboard-galaxy-registry.md)
- [fuxi-hall.md](current/fuxi-hall.md)
- [info-and-reporting.md](current/info-and-reporting.md)
- [mentor-system.md](current/mentor-system.md)
- [newbro-support.md](current/newbro-support.md)
- [npc-kills.md](current/npc-kills.md)
- [operation.md](current/operation.md)
- [pap-exchange.md](current/pap-exchange.md)
- [skill-planning.md](current/skill-planning.md)
- [sde.md](current/sde.md)
- [srp.md](current/srp.md)
- [task-manager.md](current/task-manager.md)
- [ticket-system.md](current/ticket-system.md)
- [welfare.md](current/welfare.md)

## Feature Doc 最少要回答的问题

- 这个模块当前对用户提供什么能力
- 入口页面和后端路由在哪里
- 需要什么职权 / 权限
- 哪些行为是必须保持的
- 真实代码文件在哪里

如果模块里存在容易被误解的高风险 caveat，也必须明确写出，例如：

- 真实权限边界与页面文案不一致时，以什么为准
- 自动职权 / 自动权限提升依赖的真实输入信号是什么
- 哪些“看起来像权限”的字段其实只是展示名称或兼容字段

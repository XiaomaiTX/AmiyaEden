---
status: active
doc_type: feature
owner: engineering
last_reviewed: 2026-04-12
source_of_truth:
  - server/internal/router/router.go
  - server/internal/service/eve_info.go
  - server/internal/service/fittings.go
  - server/internal/service/npc_kill.go
  - static/src/api/eve-info.ts
  - static/src/api/npc-kill.ts
  - static/src/views/info
---

# EVE 信息与报表

## 当前能力

- 钱包流水
  - 钱包流水数据通过 ESI 任务 `character_wallet` 定期刷新
  - 用户可通过页面顶部的"ESI 拉取"按钮手动触发钱包数据刷新（仅限自己绑定的角色）
  - 刷新为异步任务，提交后需等待任务完成并通过"刷新"按钮查看最新数据
- 技能列表
  - 技能列表与总技能点数据通过 ESI 任务 `character_skill` 定期刷新
  - 用户可通过页面顶部的"ESI 拉取"按钮手动触发技能数据刷新（仅限自己绑定的角色）
  - 刷新为异步任务，提交后需等待任务完成并通过"刷新"按钮查看最新数据
- 舰船列表
- 植入体
- 资产
- 合同列表与详情
- 装配列表与保存
- 个人 NPC 刷怪报表（详见 [npc-kills.md](npc-kills.md)）
- 全量 NPC 刷怪报表（详见 [npc-kills.md](npc-kills.md)）
- ESI 授权检查
  - 总览矩阵：横轴为绑定人物，纵轴为注册 scope，按模块分组，显示每人各 scope 的授权状态
  - 人物详情：下拉选择人物，展示 scope 列表含授权状态、用途说明、模块归属
  - Token 失效时标记警告，缺失 scope 提示需重新绑定
  - 数据来源：`GET /api/v1/sso/eve/scopes`（公开）+ `GET /api/v1/sso/eve/characters`（需 JWT）

## 前端金额展示

- 钱包余额、钱包流水、个人 NPC 刷怪报表、公司 NPC 刷怪报表中的 ISK 金额使用 plain ISK value style。
- 合同列表和合同详情中的 ISK 金额使用 smart abbreviation style。
- 这些页面的 ISK 显示统一复用 `@/utils/common` 中的共享 helper，不再定义页面内本地 formatter。

## 入口

### 前端页面

- `static/src/views/info/wallet`
- `static/src/views/info/skill`
- `static/src/views/info/ships`
- `static/src/views/info/implants`
- `static/src/views/info/assets`
- `static/src/views/info/contracts`
- `static/src/views/info/fittings`
- `static/src/views/info/esi-check`
- `static/src/views/info/npc-kills`
- `static/src/views/dashboard/npc-kills`

### 后端路由

- `/api/v1/info/wallet`
- `/api/v1/info/skills`
- `/api/v1/info/ships`
- `/api/v1/info/implants`
- `/api/v1/info/assets`
- `/api/v1/info/contracts`
- `/api/v1/info/contracts/detail`
- `/api/v1/info/esi-refresh` - 手动触发指定角色的 ESI 刷新（支持技能、钱包等任务，仅限自己的角色，需 `Login` 权限）
- `/api/v1/info/fittings`
- `/api/v1/info/fittings/save`
- `/api/v1/info/npc-kills`
- `/api/v1/info/npc-kills/all`
- `/api/v1/system/npc-kills`

## 权限边界

- 用户侧信息查询要求 `Login`，`guest` 不可访问
- 公司级 NPC 刷怪报表页面位于 `/dashboard/npc-kills`，仅 `admin` 或 `super_admin` 可见
- 公司级 NPC 刷怪报表接口仍为 `/api/v1/system/npc-kills`

## 关键不变量

- 此模块基于本地持久化的 ESI / SDE 数据与查询服务，不是页面直接调 CCP
- NPC 刷怪既有用户视角也有管理员视角，文档和实现都要区分清楚
- 装配功能属于 Info 模块，但也被舰队配置与自动 SRP 复用
- 技能相关表（`eve_character_skill`、`eve_character_skills`、`eve_character_skill_queue`）采用"整表重建"更新模式，不保留历史快照
- 钱包相关表（`eve_character_wallet_journal` 等）由 `character_wallet` ESI 任务定期更新，新增记录不覆盖历史

## 主要代码文件

- `server/internal/service/eve_info.go`
- `server/internal/service/fittings.go`
- `server/internal/service/npc_kill.go`
- `server/internal/router/router.go`
- `static/src/api/eve-info.ts`
- `static/src/api/npc-kill.ts`
- `static/src/views/info`

---
status: active
doc_type: feature
owner: engineering
last_reviewed: 2026-09-23
source_of_truth:
  - server/internal/router/router.go
  - server/internal/service/eve_info.go
  - server/internal/service/eve_wallet_analytics.go
  - server/internal/service/fittings.go
  - server/internal/service/npc_kill.go
  - static/src/api/eve-info.ts
  - static/src/api/npc-kill.ts
  - static/src/views/info
  - static-react/src/pages/info-wallet-page.tsx
---

# EVE 信息与报表

## 当前能力

- 钱包流水
  - 钱包流水数据通过 ESI 任务 `character_wallet` 定期刷新
  - 用户可通过页面顶部的"ESI 拉取"按钮手动触发钱包数据刷新（仅限自己绑定的角色）
  - 刷新为异步任务，提交后需等待任务完成并通过"刷新"按钮查看最新数据
- 钱包收支分析
  - 基于本地已同步的钱包流水按 EVE 时间（UTC）自然日聚合，展示每日余额、每日收支与交易类型构成
  - 分析范围仅由"人物 + 日期区间"决定，不跟随流水表格的类型（ref_type）筛选
  - 提供 1 / 7 / 30 / 90 天与"全部可得范围"预设，范围边界取自接口返回的 `available_from` / `available_to`
  - 日期区间可手动调整，超出可得范围时以后端实际数据为准
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
  - 未授权的可选 scope 可在对应行发起授权；授权请求仍包含系统全部必需 scope
  - Token 失效时标记警告，缺失 scope 提示需重新绑定
  - 数据来源：`GET /api/v1/sso/eve/scopes`（公开）+ `GET /api/v1/sso/eve/characters`（需 JWT）
- 常用工具网站
  - 登录用户可查看常用网站书签列表
  - 管理员可新增、编辑、删除书签
  - 后端在保存时自动尝试获取站点 logo（优先 HTML icon，回退 favicon）

## 前端金额展示

- 钱包余额、钱包流水、个人 NPC 刷怪报表、公司 NPC 刷怪报表中的 ISK 金额使用 plain ISK value style。
- 合同列表和合同详情中的 ISK 金额使用 smart abbreviation style。
- 钱包收支分析的汇总卡片主值、图表坐标轴与悬浮提示使用 smart abbreviation style，卡片明细行同时给出 plain 精确值。
- 这些页面的 ISK 显示统一复用 `@/utils/common` 中的共享 helper，不再定义页面内本地 formatter。

## 入口

### 前端页面

- `static/src/views/info/wallet`（含 `modules/wallet-analytics.vue` 收支分析区块）
- `static/src/views/info/skill`
- `static/src/views/info/ships`
- `static/src/views/info/implants`
- `static/src/views/info/assets`
- `static/src/views/info/contracts`
- `static/src/views/info/fittings`
- `static/src/views/info/esi-check`
- `static/src/views/info/tool-bookmarks`
- `static/src/views/info/npc-kills`
- `static/src/views/dashboard/npc-kills`

### 后端路由

- `/api/v1/info/wallet`
- `/api/v1/info/wallet/analytics` - 钱包收支分析（按 UTC 自然日聚合，仅查询本地已同步流水，需 `Login` 权限）
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
- `/api/v1/info/tool-bookmarks`
- `/api/v1/system/tool-bookmarks`
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

## 钱包收支分析的设计取舍

- 分析只读取本地已同步的 `eve_character_wallet_journal`，不额外调用 ESI，也不新增或收紧 scope。
- 分日聚合必须在 service 层显式按 UTC 完成，不得使用数据库会话时区的 `DATE()` 分组：部署环境的会话时区可能是本地时区（例如 Asia/Shanghai），否则北京时间 00:00–08:00 的流水会被归到前一天。
- 可得数据范围 = CCP 的 30 天回传窗口 ∩ 本地持续积累时长。`available_from` / `available_to` 取自本地表实际最早/最晚流水日期，前端日期选择边界必须使用接口返回值，不得写死 30 天。
- 不引入"每日余额快照"表：每笔流水自带该笔之后的余额，无流水日余额不变，余额曲线可由流水完整重建；快照只有预聚合省算力的价值，属于数据量足够大之后的性能优化项。
- 流水表格按浏览器本地时区展示时间，图表按 UTC 自然日分桶，因此北京时间 00:00–08:00 的流水在两者中的日期可能相差一天；页面以"按 EVE 时间（UTC）自然日统计"的说明提示这一差异。
- 当日无流水与当日同步失败在数据上无法区分，两者都按"余额延续上一日、收支为 0"处理，因此界面必须展示实际数据范围。
- 图表数值轴（每日余额、每日收支、交易类型构成）使用 ISK 智能缩写（K/M/B/T）渲染刻度，避免原始长数字在窄轴上互相重叠；悬浮提示中的数值使用同一格式化（不再显示全精度小数）。共享图表组件通过可选的 `valueFormatter` props 接收格式化函数，同时作用于数值轴刻度与提示框数值，未传入时保持 ECharts 默认渲染。
- 柱体厚度使用「百分比 + 像素上限」双约束：`barWidth` 用百分比以适配长序列，`barMaxWidth`（当前统一 14px）在类目较少时（如近 1 天）防止柱体过粗。类型构成图按金额降序、数值较大的在上（ECharts 通过 `inverse`、recharts 默认即为降序，两端不得混用同一开关）；数值轴为最右刻度预留右侧留白（`grid.right`），避免刻度标签被容器右缘裁切。
- 每日收支图使用语义色：收入绿（`#16a34a`）、支出红（`#ef4444`），与汇总卡同色系。Vue 端传入图表组件的 `colors`，React 端写入 `flowChartConfig`。注意 **ECharts 底层 zrender 的颜色解析器不支持 `oklch()`**，而 Vue 主题变量（`--art-success`/`--art-danger`）均为 oklch，故此处必须用十六进制常量（与 Tailwind green-600 / red-500 一致），不可传主题变量。

## 主要代码文件

- `server/internal/service/eve_info.go`
- `server/internal/service/eve_wallet_analytics.go`
- `server/internal/service/fittings.go`
- `server/internal/service/npc_kill.go`
- `server/internal/router/router.go`
- `server/internal/repository/eve_wallet.go`
- `static/src/api/eve-info.ts`
- `static/src/api/npc-kill.ts`
- `static/src/views/info`
- `static-react/src/pages/info-wallet-analytics`

## 前端实现映射（迁移期）

- Vue 当前实现位于 `static/src`。
- React 已承接钱包、技能、NPC 击杀、舰船、植入体、资产、合同、装配、ESI 检查和工具书签页面；对应 API/类型以 `static-react/src/api` 和 `static-react/src/types/api` 为准。
- `tool-bookmarks` 已在 React 落地：普通登录用户读取启用书签，`admin` / `super_admin` 使用管理接口读取全部书签并可维护书签；这属于范围漂移追赶完成项，不属于 Stage 0A capability/menu parity 基础设施。

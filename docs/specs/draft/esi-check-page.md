---
status: completed
doc_type: spec
owner: engineering
last_reviewed: 2026-04-12
---

# ESI 授权检查页面方案

## 当前状态

- 已实现：全部功能
- 完成日期：2026-04-12
- 验证：vue-tsc 0 新错误、eslint 0 错误、11/11 测试通过

## 背景

用户在使用平台功能时，需要通过 EVE SSO 授权一系列 ESI scope。当某个 scope 缺失或 token 失效时，对应的数据刷新任务会失败，但用户目前缺乏直观的手段了解：

1. 系统需要哪些 ESI scope 以及各自用途
2. 自己的人物当前已授权了哪些 scope
3. 哪些 scope 缺失需要重新授权

因此需要在 `/info` 菜单下新增 `/info/esi-check` 页面，向已登录用户展示完整的 ESI 授权状态矩阵。

## 可行性分析

### 可行性分析

### 后端基础设施（需一处 JSON tag 修复）

| 已有基础设施 | 来源 | 说明 |
|---|---|---|
| Scope 注册机制 | `service.RegisterScope` / `service.GetRegisteredScopes` | 所有模块在 `bootstrap/scopes.go` 启动时注册，包含 module、scope、description、required 字段 |
| ESI Task 定义 | `esi.RefreshTask.RequiredScopes()` | 每个任务声明所需 scope，含 Scope / Description / Optional |
| 人物 scope 数据 | `EveCharacter.Scopes` 字段 | 存储该人物登录时被授予的 scope（空格分隔字符串） |
| Scope 列表 API | `GET /api/v1/sso/eve/scopes` | 返回所有已注册 scope 列表（公开端点，无需认证） |
| 人物列表 API | `GET /api/v1/sso/eve/characters` | 返回当前用户绑定的人物列表，含 Scopes 字段（JWT 认证） |

**已发现问题**：`RegisteredScope` 结构体（`server/internal/service/eve_sso.go`）缺少 JSON tag，导致 `GET /api/v1/sso/eve/scopes` 返回 PascalCase 字段名（`Module`、`Scope`、`Required`），而前端 TypeScript 类型 `Api.Auth.RegisteredScope` 期望 lowercase（`module`、`scope`、`required`）。需补充 JSON tag 使 API 响应与前端类型对齐。

**结论：后端需一处 JSON tag 修复，无需新建 API 或修改数据模型。**

### 前端模式（已确立）

- 路由定义：`static/src/router/modules/info.ts`，所有 info 子页面使用 `component: '/info/xxx'` 模式
- 国际化：`zh.json` / `en.json` 中 `menus.info.*` 节点
- 菜单自动生成：路由 `meta.title` 对应 i18n key，由框架自动渲染菜单

## 提案内容

### 整体页面结构

页面分为两个层级，自上而下：

1. **Overview 区域** — 所有人物的 ESI 授权总览矩阵
2. **Detail 区域** — 选中人物的详细 scope 授权列表

### 第一层：Overview 总览区

#### 功能说明

- 以矩阵形式展示用户所有人物 × 系统所需 scope 的授权状态
- 每个人物一列，每个 scope 一行，交叉格显示 ✅ / ❌
- 人物列头显示头像 + 名称，Token 失效时列头标红警告
- 矩阵底部汇总每个人物的授权覆盖率（如 `12/15 已授权`）
- 点击人物列头可滚动/展开至下方 Detail 区域查看该人物详情

#### 数据流

```
1. 调用 GET /api/v1/sso/eve/scopes
   → 获取系统所需 scope 列表
   → RegisteredScope[]: [{module, scope, description, required}]

2. 调用 GET /api/v1/sso/eve/characters
   → 获取当前用户人物列表
   → EveCharacter[]: [{character_id, character_name, scopes, token_invalid, ...}]

3. 前端计算授权矩阵：
   - 将每个人物的 scopes（空格分隔字符串）→ Set<string>
   - 构建 scopes × characters 二维矩阵
   - 每个格子判断 scope 是否在人物 scope Set 中
```

#### 布局设计

```
┌──────────────────────────────────────────────────────────────────────────────┐
│  ESI 授权总览                                                                │
├────────────────────────────┬──────────┬──────────┬──────────┬────────────────┤
│  Scope (按模块分组)          │ [头像]    │ [头像]    │ [头像]    │               │
│                            │  人物A    │  人物B    │  人物C    │               │
│ ──── 钱包模块 ────────────────────────────────────────────────────────────── │
│  esi-wallet.read_charact.. │    ✅    │    ✅    │    ❌    │               │
│ ──── 击杀邮件模块 ────────────────────────────────────────────────────────── │
│  esi-killmails.read_kill.. │    ✅    │    ❌    │    ✅    │               │
│ ──── 舰队模块 ───────────────────────────────────────────────────────────── │
│  esi-fleets.read_fleet.v1  │    ❌    │    ❌    │    ❌    │               │
│  esi-fleets.write_fleet..  │    ❌    │    ❌    │    ❌    │               │
│ ──── SRP 模块 ────────────────────────────────────────────────────────────── │
│  esi-ui.open_window.v1     │    ✅    │    ❌    │    ❌    │               │
│ ──────────────────────────────────────────────────────────────────────────────│
│  覆盖率                     │  3/7     │  2/7⚠   │  2/7     │               │
└────────────────────────────┴──────────┴──────────┴──────────┴────────────────┘
  ⚠ = Token 已失效
```

当人物较多（>5）时，人物列水平滚动，Scope 列固定。

### 第二层：Detail 详情区

#### 功能说明

- 顶部人物选择器（与 wallet、skill 等页面一致，使用 ElSelect + Avatar）
- 展示选中人物的完整 scope 列表，包含：
  - scope 名称
  - 用途描述
  - 所属模块
  - 是否必需（required 标签）
  - 当前授权状态（✅ 已授权 / ❌ 缺失）
- Token 失效时顶部显示醒目警告横幅
- 缺失 scope 提供引导提示（重新绑定人物以获取授权）

#### 布局设计

```
┌─────────────────────────────────────────────────┐
│  ⚠ Token 已失效，部分数据可能无法刷新             │  ← 仅 token_invalid 时显示
├─────────────────────────────────────────────────┤
│  [人物选择器 ▼]  状态摘要: ✅ 3/7 已授权          │
├─────────────────────────────────────────────────┤
│  Scope                          │ 用途    │ 模块  │ 必需 │ 状态  │
│ ─────────────────────────────────────────────── │
│  esi-wallet.read_character_...  │ 读取钱包 │ 钱包  │  是  │  ✅   │
│  esi-killmails.read_killmail... │ 读取击杀 │ 击杀  │  是  │  ✅   │
│  esi-fleets.read_fleet.v1      │ 读取舰队 │ 舰队  │  否  │  ❌   │
│  esi-fleets.write_fleet.v1     │ 写入舰队 │ 舰队  │  否  │  ❌   │
│  esi-ui.open_window.v1         │ 打开窗口 │ SRP  │  否  │  ❌   │
│  ...                                            │
└─────────────────────────────────────────────────┘
```

### 组件拆分

```
static/src/views/info/esi-check/
├── index.vue                  # 页面入口，编排 Overview + Detail
└── modules/
    ├── overview-matrix.vue    # 总览矩阵组件（所有人物 × scope）
    └── character-detail.vue   # 单人物详情组件（scope 列表 + 状态）
```

#### index.vue

- 职责：调用两个 API，持有 `scopes` 和 `characters` 数据，传递给子组件
- 布局：Overview 在上，Detail 在下，自然滚动（不使用 `art-full-height`，页面内容随数据增长）

#### overview-matrix.vue

- Props：`scopes: RegisteredScope[]`、`characters: EveCharacter[]`
- 职责：渲染 scope × character 矩阵，计算每个格子的授权状态
- 交互：点击人物列头 → emit `select-character` 事件，通知父组件切换 Detail 区域的选中人物
- 模块分组：按 scope.module 分组，使用 ElDivider 或行分隔

#### character-detail.vue

- Props：`scopes: RegisteredScope[]`、`character: EveCharacter | null`
- 职责：渲染单人物的 scope 详细列表
- 内部包含人物选择器（ElSelect + Avatar），切换人物时 emit `update:character`
- Token 失效时显示 ElAlert 警告

### 变更清单

| 文件 | 变更类型 | 说明 |
|---|---|---|
| `server/internal/service/eve_sso.go` | 修改 | `RegisteredScope` 结构体补充 JSON tag（`json:"module"` 等），修复 API 返回 PascalCase 导致前端读取全部为 `undefined` 的问题 |
| `static/src/router/modules/info.ts` | 修改 | 新增 `esi-check` 子路由 |
| `static/src/locales/langs/zh.json` | 修改 | 新增 `menus.info.esiCheck` 及 18 个页面 i18n key |
| `static/src/locales/langs/en.json` | 修改 | 新增对应英文翻译 |
| `static/src/views/info/esi-check/index.vue` | **新建** | 页面入口组件 |
| `static/src/views/info/esi-check/modules/overview-matrix.vue` | **新建** | 总览矩阵组件 |
| `static/src/views/info/esi-check/modules/character-detail.vue` | **新建** | 单人物详情组件 |
| `static/src/views/info/esi-check/index.test.ts` | **新建** | 页面入口测试（3 案例） |
| `static/src/views/info/esi-check/modules/overview-matrix.test.ts` | **新建** | 矩阵组件测试（4 案例） |
| `static/src/views/info/esi-check/modules/character-detail.test.ts` | **新建** | 详情组件测试（4 案例） |
| `docs/features/current/info-and-reporting.md` | 修改 | 新增 ESI 授权检查能力描述与前端页面入口 |

> **实现偏差说明**：草案中计划修改 `static/src/api/eve-info.ts` 新增 API 封装，实际实现中发现 `static/src/api/auth.ts` 中已有 `fetchEveSSOScopes()` 和 `fetchMyCharacters()`，直接复用，无需新增 API 封装。`character-detail.vue` 的 Props 从草案的 `character: EveCharacter | null` 调整为 `characters + selectedCharacterId` 模式，与项目其他页面（如 wallet）保持一致。

### 路由注册

```typescript
{
  path: 'esi-check',
  name: 'EveInfoEsiCheck',
  component: '/info/esi-check',
  meta: { title: 'menus.info.esiCheck', keepAlive: true, login: true }
}
```

### 国际化 Key

```
menus.info.esiCheck          = "ESI 授权检查" / "ESI Authorization Check"
info.esiCheck.overview       = "授权总览" / "Authorization Overview"
info.esiCheck.detail         = "人物详情" / "Character Detail"
info.esiCheck.scope          = "Scope"
info.esiCheck.description    = "用途" / "Description"
info.esiCheck.module         = "模块" / "Module"
info.esiCheck.required       = "必需" / "Required"
info.esiCheck.optional       = "可选" / "Optional"
info.esiCheck.authorized     = "已授权" / "Authorized"
info.esiCheck.missing        = "缺失" / "Missing"
info.esiCheck.tokenInvalid   = "Token 已失效" / "Token Invalid"
info.esiCheck.tokenInvalidTip= "Token 已失效，部分数据可能无法刷新" / "Token is invalid, some data may not refresh"
info.esiCheck.selectChar     = "请选择人物" / "Select Character"
info.esiCheck.allGranted     = "所有授权均已就绪" / "All authorizations are ready"
info.esiCheck.someMissing    = "有 {n} 个授权缺失" / "{n} authorizations missing"
info.esiCheck.coverage       = "{granted}/{total} 已授权" / "{granted}/{total} authorized"
info.esiCheck.clickToDetail  = "点击查看详情" / "Click to view details"
info.esiCheck.reauthTip      = "缺失的 scope 需要重新绑定人物以获取授权" / "Missing scopes require re-binding the character to authorize"
info.esiCheck.noData         = "暂无 ESI 授权数据" / "No ESI authorization data"
info.esiCheck.noCharacters   = "暂无绑定人物" / "No bound characters"
```

## 实现时需同步更新的项目文档

根据 `docs/standards/pre-completion-checklist.md` 和 `docs/standards/documentation-governance.md` 的要求，实现时需同步更新以下文档：

### 必须更新

| 文档 | 更新内容 |
|---|---|
| `docs/features/current/info-and-reporting.md` | 1. 在"当前能力"中增加"ESI 授权检查"条目<br>2. 在"前端页面"入口列表中增加 `static/src/views/info/esi-check`<br>3. 说明此页面消费的 API 端点（`GET /sso/eve/scopes`、`GET /sso/eve/characters`）已在 `auth-and-characters.md` 中记录 |

### 无需更新（及原因）

| 文档 | 原因 |
|---|---|
| `docs/features/current/auth-and-characters.md` | 不改变认证/人物绑定行为，仅只读消费已有 API |
| `docs/api/route-index.md` | 不新增后端路由；消费的两个端点已记录在 Public § EVE SSO 和 Authenticated Base |
| `docs/architecture/routing-and-menus.md` | 不引入新路由模式，沿用 `info.ts` 模块 `login: true` 模式 |
| `docs/architecture/module-map.md` | 不引入新目录职责或模块结构模式 |
| `docs/standards/frontend-table-pages.md` | ESI Check 是矩阵视图，不是标准分页表格页 |

## 测试案例

### 测试策略说明

本方案为纯前端变更（不修改后端），遵循项目测试标准（`docs/standards/testing-and-verification.md`）和测试指南（`docs/guides/testing-guide.md`）：

- 核心逻辑（scope 集合计算、授权矩阵判定、覆盖率计算）是确定性纯函数，适合前端单元测试
- 项目现有 info 模块测试采用 `node:test` + 源码字符串匹配模式（如 `info/skill/index.test.ts`、`info/wallet/index.test.ts`），本功能沿用此模式
- 不需要后端测试（无后端改动）
- 不需要组件渲染测试（无复杂交互逻辑，UI 为纯展示 + 条件渲染）

### 建议新增测试文件

#### 1. `static/src/views/info/esi-check/index.test.ts`

页面入口源码断言，验证关键结构存在：

| 测试案例 | 验证内容 |
|---|---|
| 页面调用两个 API 获取数据 | 源码中引用 `fetchRegisteredScopes()` 和 `fetchMyCharacters()` |
| 页面将 scope 和 character 数据传递给子组件 | overview-matrix 和 character-detail 组件接收正确的 props |
| 页面响应 select-character 事件切换选中人物 | emit 处理函数正确更新选中人物 |

#### 2. `static/src/views/info/esi-check/modules/overview-matrix.test.ts`

Overview 矩阵的核心逻辑断言：

| 测试案例 | 验证内容 |
|---|---|
| 按模块分组 scope | 源码中使用 `scope.module` 进行分组逻辑 |
| Token 失效人物列头有警告标识 | `token_invalid` 为 true 时渲染警告样式/图标 |
| 覆盖率计算逻辑 | 授权数 / 总 scope 数的计算公式存在 |
| 点击人物列头触发 select 事件 | 存在 `emit('select-character', ...)` 调用 |

#### 3. `static/src/views/info/esi-check/modules/character-detail.test.ts`

单人物详情的核心逻辑断言：

| 测试案例 | 验证内容 |
|---|---|
| scope 列表渲染授权状态 | 通过 character.scopes 判断每个 scope 是否已授权 |
| Token 失效时显示警告横幅 | `token_invalid` 为 true 时渲染 ElAlert |
| 缺失 scope 显示重新授权提示 | 未授权 scope 旁显示引导文案 |
| 授权摘要计算 | 已授权数 / 总 scope 数 |

### 可选：纯函数提取测试

如果实现中将 scope 解析逻辑提取为独立纯函数（如 `parseScopes(scopes: string): Set<string>`、`buildAuthMatrix(scopes, characters)`），则应在同目录下新增 `.helpers.test.ts`，使用标准断言测试：

- `parseScopes('') → Set{}`
- `parseScopes('esi-wallet.v1 esi-skills.v1') → Set{'esi-wallet.v1', 'esi-skills.v1'}`
- `parseScopes` 处理连续空格、前后空格
- 矩阵构建：空人物列表 → 空矩阵；单人物单 scope → 正确判定

### 无需测试的部分

| 范围 | 原因 |
|---|---|
| 后端 service / handler | JSON tag 修改不影响 Go 编译或运行时行为，现有测试（`eve_sso_test.go`）通过 Go 字段名访问，不受影响 |
| API 封装层 `fetchRegisteredScopes()` | 纯 HTTP 调用包装，无逻辑分支 |
| i18n 翻译 | 纯文案，由 `pnpm lint` 和人工检查覆盖 |
| 路由注册 | 配置性代码，由 `vue-tsc --noEmit` 类型检查覆盖 |
| 组件渲染测试 | UI 为纯条件渲染，无复杂交互，源码断言已足够 |

## 不涉及的范围

- 不新建后端 API
- 不修改数据模型
- 不实现自动重新授权流程（仅提供引导提示）

### 后端 JSON tag 修复影响范围

修复仅影响 `GET /api/v1/sso/eve/scopes` 这一个 API 的 JSON 序列化输出。后端内部所有 Go 代码通过结构体字段名（`rs.Module`、`rs.Scope`、`rs.Required`）访问，不受 JSON tag 影响。前端已有 `Api.Auth.RegisteredScope` 类型定义（lowercase 字段名），修复后前后端契约对齐。

---
status: active
doc_type: standard
owner: engineering
last_reviewed: 2026-07-22
source_of_truth:
  - docs/README.md
  - docs/ai/repo-rules.md
---

# 文档治理规范

## 适用范围

本规范约束仓库根目录与 `docs/` 下的权威文档。这包括代理入口文件（`AGENTS.md`、`CLAUDE.md`），它们委托给 `docs/ai/repo-rules.md`。

## 核心规则

- 每份文档必须只有一项主要职责。
- 每类事实必须只有一处权威来源。
- 当前实现、工程规则与未来提案必须分别存放。
- 不得为同一主题维护第二套并行文档树。
- 仓库级权威文档仅允许存在于 `docs/` 与代理入口（`AGENTS.md`、`CLAUDE.md`），后者委托给 `docs/ai/repo-rules.md`。
- 根 `README.md` 可作为 onboarding 或面向产品的入口，但不定义工程规则。如存在冲突，以 `docs/ai/repo-rules.md` 与 `docs/` 为准。
- 子目录下的 `README.md` 仅为本地实现备注，不得重新定义仓库级规则、路由面或产品行为。

## 受众约定

目录到受众的映射见 `docs/README.md § 受众分类`。放置新文档时，选择与其主要受众匹配的目录。

### AI 中心内容准则

AI 中心文档中的每个词都会消耗代理上下文窗口。保持文档精简，避免可从代码派生的内容。

**应包含：**

- UI 布局与行为描述——这些是需求规格，没有它们代理可能无意中修改 UI 行为
- 业务逻辑、计算规则与资格条件——便于评审并防止与预期行为静默漂移
- 难以从代码直接看出的权限边界与关键不变量
- 约束未来实现的持久技术设计决策与理由，尤其是后端架构、安全、数据一致性、外部集成、队列、缓存与兼容行为
- 入口（路由、页面）与主要代码文件

**不应包含：**

- API 请求/响应 JSON 示例——可从 handler 代码与对应的 Vue/React 类型文件派生
- 例行实现选择、本地控制流或邻近代码已能直接看出的理由
- 仅重述本文档其他段落已写内容的不变量——总结小节应只引入真正新的信息
- 已在其他文档中权威化的内容——以引用代替复制（例如 feature 文档应引用 `docs/architecture/auth-and-permissions.md` 获取角色分配规则，而非重述权限矩阵）

**理由：** `docs/features/current/` 下的 feature 文档是需求规格。它们定义系统*应当*做什么，使代理能校验实现、让评审者捕获非预期变更。它们不是代码文档，不描述代码结构，也不重复代码已表达的内容。

## 文档类型

使用以下映射：

- `agent-rules` / `agent-guide` -> `docs/ai/`
  共享代理规则源，由 `AGENTS.md` 与 `CLAUDE.md` 使用，以及面向代理的说明性文档。
- `standard` -> `docs/standards/`
  强制规则、禁令、推荐实践与回归测试策略。
- `architecture` -> `docs/architecture/`
  当前系统如何工作。
- `api` -> `docs/api/`
  路由、鉴权与响应约定。
- `feature` -> `docs/features/current/`
  当前模块行为、入口、权限与不变量。
- `guide` -> `docs/guides/`
  面向人类工程师的分步操作说明。
- `reference` -> `docs/reference/`
  离线参考资产，不是当前实现的权威。
- `draft` -> `docs/specs/draft/`
  提案、增强与未实现设计。
- `template` -> `docs/templates/`
  用于创建新文档的模板。

## 技术设计决策

难以从代码看出、且约束未来变更的持久后端与架构决策必须文档化。

把这些决策记录在最近的权威文档中：

- 跨切后端架构、依赖方向、启动行为、后台任务、缓存、队列或集成策略 -> `docs/architecture/*.md`
- API 契约形状、路由边界、鉴权、授权或兼容行为 -> `docs/api/*.md`
- 模块级业务规则、资格逻辑、副作用、状态流转或运维警示 -> `docs/features/current/*.md`
- 可复用的工程规则、禁令或必备实践 -> `docs/standards/*.md`
- 提案或未实现设计 -> `docs/specs/draft/*.md`

当已有权威文档拥有该主题时，使用简短设计备注而非另起文档。仅当决策足够宽泛、能独立成立且没有现有文档具有正确的首要职责时，才创建新文档。

设计备注应只包含持久的部分：

- 决策
- 理由
- 未来变更必须保留的不变量
- 能防止日后走回头路的重要权衡或被否决的替代方案
- 主要代码文件

不得把持久设计理由仅留在聊天、issue 评论、PR 摘要、commit 消息或代理记忆中。这些来源可以解释某次变更的评审，但不是权威的仓库记忆。

## Front Matter 要求

所有新的权威文档必须包含 YAML front matter，至少声明以下字段：

- `status`
- `doc_type`
- `owner`
- `last_reviewed`
- `source_of_truth`

front matter 示例：

```yaml
status: active  
doc_type: feature  
owner: engineering  
last_reviewed: 2026-03-24  
source_of_truth:  
  - server/internal/router/router.go
```

推荐字段：

- `source_of_truth`
- `supersedes`
- `related_docs`

模板规则：

- `docs/templates/*` 下的文件必须使用 `status: template`
- 模板必须明确声明自身为模板，不描述当前实现

## 文件命名

- 使用 `kebab-case`
- 按范围而非临时结论命名
- 不得使用会很快过期的名字，如 `new-`、`final-`、`latest-`、`v2-`

推荐示例：

- `auth-and-permissions.md`
- `runtime-and-startup.md`
- `route-index.md`

## 按文档类型的最低结构

### standard

- 适用范围
- 核心规则
- 允许的例外
- 核对清单

### architecture

- 适用范围
- 当前实现
- 难以从代码看出的后端或系统选择的设计决策与理由
- 关键入口文件
- 不变量

### api

- base URL、鉴权与响应约定
- 路由索引或接口列表
- 相关处的显式权限边界
- 难以从代码看出的契约或兼容选择的设计理由
- 变更同步要求

### feature

- 模块目的
- 当前入口
- 权限边界
- 难以从代码看出的后端行为设计决策与理由
- 关键不变量
- 主要代码文件

### reference

- 资产目的
- 文件列表
- 非权威状态
- 使用限制或刷新指引

### draft

- 背景
- 当前状态
- 提案
- 未决问题
- 明确声明尚未实现

## 何时创建新文档

满足以下条件时创建新文档：

- 新功能模块足够大，能独立成立
- 新标准将在多个模块间复用
- 提案尚未实现但需要持续讨论
- 持久技术设计决策跨切足够广，需要独立的权威拥有者

满足以下条件时不要创建新文档：

- 仅从另一角度重复已有路由表
- 仅改写已有规则
- 仅记录临时讨论结论
- 属于应放在现有 architecture、API、standard 或 feature 文档中的设计备注
- 会创建一个与 `docs/` 已有权威文档重复的子目录 `README.md`

## 更新规则

- 行为变更与文档更新必须在同一次变更中完成。
- 难以从代码看出的、有意的后端设计决策必须在引入或实质性修改它的同一次变更中文档化。
- 更改文档状态或范围时，更新 `last_reviewed`。
- 当文档从 `draft` 升为活跃权威时，应移入正确目录而不是仅改标题。
- 删除或合并文档时，清除失效引用，不留影子入口。

## 权威来源

某些事实有指定的唯一来源。不要在其他文档中重新定义或复制，应以引用代替。

权威事实映射：

- 验证命令（`lint`、`test`、`build`）-> `docs/standards/testing-and-verification.md § 默认命令`
- 时间戳/日期时间展示格式 -> `docs/standards/timestamp-formatting.md`
- 页面级表格布局 / 账本默认 -> `docs/standards/frontend-table-pages.md`
- 记录卡片页溢出 / 页面扩展规则 -> `docs/standards/frontend-record-card-pages.md`

当新增一类会在多文档中出现的事实时，在此指定唯一权威来源，并将其他出现处转为引用。

## 反模式

避免以下做法：

- 在 README、guide 与 feature 文档之间重复同一份角色列表或规则
- 在 `docs/standards/testing-and-verification.md § 默认命令` 之外重新定义验证命令
- 把仓库级分页表格布局或账本默认重述在 `docs/features/current/*.md` 中，而不是放在 `docs/standards/`
- 把仓库级 UI 布局或溢出规则重述在 `docs/features/current/*.md` 中，而不是放在 `docs/standards/`
- 把根 `README.md` 做成与 `docs/ai/repo-rules.md` 和 `docs/` 并列的工程标准
- 把未来计划混入当前状态文档
- 维护第二套与权威文档冲突的并行文档树
- 维护大杂烩式后端决策日志，而该决策本应放在已有权威文档中
- 仅依赖聊天、issue 评论、PR 摘要、commit 消息或代理记忆作为持久设计理由的唯一记录
- 在 feature 文档中包含 API 请求/响应 JSON 示例（可从代码与类型定义派生）
- 在总结小节中重述文档正文已写的不变量
- 引用代码时过于模糊，读者无法定位真正的入口文件

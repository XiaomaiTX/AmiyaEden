---
status: active
doc_type: index
owner: engineering
last_reviewed: 2026-04-23
source_of_truth:
  - docs/ai/repo-rules.md
---

# AmiyaEden 文档索引

## 目的

`docs/` 是仓库规范化文档树，用来把”工程标准””当前架构””API 路由面””当前功能状态””未来提案”彻底分开，避免文档互相覆盖、过期后继续误导人或 AI。根目录的 `AGENTS.md` 和 `CLAUDE.md` 是代理入口文件，它们委托给 `docs/ai/repo-rules.md` 作为实际规则来源。

根目录 `README.md` 仍然保留为面向人类的 onboarding / 产品入口，但它不是 repo-level engineering rule 的裁决来源；如与 `docs/ai/repo-rules.md` 或 `docs/` 冲突，以后者为准。

## 信任顺序

权威性排序的规范定义见 `docs/ai/repo-rules.md`「Authority Order」一节。

简要概述（当多个文件描述同一件事时）：

1. `docs/ai/repo-rules.md`（通过 `AGENTS.md` / `CLAUDE.md` 加载）
2. `docs/standards/*.md`
3. `docs/architecture/*.md`
4. `docs/api/*.md`
5. `docs/features/current/*.md`
6. `docs/guides/*.md`
7. `docs/specs/draft/*.md`

说明：

- 第 7 层只表示”计划 / 草案 / 未完成设计”，不能覆盖当前实现。

## 受众分类

文档按主要受众分为两类：

### AI Agent 中心（agent-centric）

以下目录的文档主要服务于 AI 代理和自动化工具。人类工程师可能偶尔阅读（特别是 `docs/architecture/`），但内容以 agent 消费为主：

- `docs/ai/` — 代理规则、onboarding、Harness 原则
- `docs/standards/` — 约束性标准与回归测试策略
- `docs/architecture/` — 系统结构（人类也会少量阅读）
- `docs/api/` — 接口约定与路由索引
- `docs/features/current/` — 功能行为规格

### 人类工程师中心（human-centric）

以下目录的文档主要服务于人类开发者，提供日常开发所需的操作指南：

- `docs/guides/` — 本地开发、调试流程、测试编写、ESI 模块添加

### 通用（utility）

以下目录同时服务两类受众：

- `docs/reference/` — 离线参考资产
- `docs/specs/draft/` — 提案与未完成设计
- `docs/templates/` — 文档模板

放置新文档时，根据主要受众选择目录。如果某文档主要由 agent 消费，不应放在 `docs/guides/`；如果某文档主要面向人类操作流程，不应放在 `docs/standards/` 或 `docs/ai/`。

## 目录结构

| 路径 | 类型 | 受众 | 作用 |
| --- | --- | --- | --- |
| `docs/ai/` | agent guide | agent | 给 AI / 自动化代理的规则、onboarding、Harness 原则 |
| `docs/standards/` | standard | agent | 约束性标准，描述”必须 / 不得 / 推荐”（含依赖分层、回归测试策略、预完成检查清单） |
| `docs/architecture/` | architecture | agent（人类少量阅读） | 只描述当前已经存在的系统结构与运行方式 |
| `docs/api/` | api | agent | 接口约定、响应格式、路由索引 |
| `docs/features/current/` | feature | agent | 当前已落地功能的模块级说明 |
| `docs/guides/` | guide | human | 面向人类工程师的过程型指南（本地开发、调试、测试、ESI 模块添加） |
| `docs/reference/` | reference | both | 离线参考资产，不作为当前实现的 source of truth |
| `docs/specs/draft/` | draft | both | 提案、未来增强、未完成设计 |
| `docs/templates/` | template | both | 新建文档时复用的模板 |

## 状态字段

所有新的规范性文档都应包含 front matter，并至少声明：

- `status`: `active` / `draft` / `deprecated` / `template`
- `doc_type`: `standard` / `architecture` / `api` / `feature` / `guide` / `reference` / `draft` / `template` / `index`
- `owner`
- `last_reviewed`

约定：

- `docs/templates/*` 使用 `status: template`，不要写成 `active`
- `docs/specs/draft/*` 使用 `status: draft`

## 文档更新规则

- 当前行为变化时，优先更新对应的 `docs/architecture`、`docs/api`、`docs/features/current`。
- 持久的后端技术设计决策如果不能从代码直接看出，应在同一次变更中写入最近的权威文档，记录决策、理由、必须保留的不变量和主要代码文件。
- 新增工程约束时，更新 `docs/ai/repo-rules.md` 或 `docs/standards`，不要把规则写进 feature doc。
- 测试与验证规则优先维护在 `docs/ai/repo-rules.md` 与 `docs/standards/testing-and-verification.md`。
- 新增尚未落地的设计时，只放进 `docs/specs/draft`。
- 不要在多个文件里重复维护同一份职权定义、路由表、权限矩阵。
- 不要创建泛化的 backend decision dump；如果已有 architecture / api / feature / standard 文档拥有该主题，应把设计说明放在那里。
- 不要保留并行的“第二套文档入口”。
- 仓库内允许存在少量模块级 `README.md` 作为局部实现说明，但它们不是 repo-level canonical doc，不能覆盖 `docs/ai/repo-rules.md` 与 `docs/`。
- 根目录 `README.md` 应保持适合新开发者快速上手，但若涉及工程规则、当前架构边界、接口裁决，仍以 `docs/ai/repo-rules.md` 与 `docs/` 为准。

如果变更属于高风险行为边界，必须把 caveat 明确写出来，不能只靠上下文暗示。

典型场景：

- 认证 / 鉴权边界
- RBAC 职权提升规则
- 自动权限映射的特殊分支
- 兼容字段与当前权威字段的区别

这类 caveat 至少要同时落在：

- `docs/architecture/*.md`，说明系统规则
- `docs/features/current/*.md`，说明模块当前行为

当前如果变更数据库表、核心关系或历史兼容列，应同时更新：

- `docs/architecture/database-schema.md`
- 受影响的 architecture / api / feature 文档

## 推荐阅读顺序

### 对人类开发者

1. `README.md`
   把它当作 onboarding / 产品入口，而不是工程规则裁决文件
2. `docs/ai/repo-rules.md`
3. 本文件
4. 相关架构文档
   如果只是先找代码落点，优先补读 `docs/architecture/module-map.md`
5. 相关 feature doc
6. 相关 API / guide

### 对 AI Agent

1. `docs/ai/repo-rules.md`（通过代理入口文件自动加载）
2. `docs/ai/agent-onboarding.md`
3. `docs/architecture/overview.md`
4. `docs/architecture/module-map.md`
5. 任务对应的标准文档
   如果涉及测试、验证、回归保障，优先补读 `docs/standards/testing-and-verification.md`
   如果涉及层级依赖或架构合规，优先补读 `docs/standards/dependency-layering.md`
   完成任务前，参照 `docs/standards/pre-completion-checklist.md` 进行验证
6. 任务对应的 feature / API 文档
7. 只有在明确做规划工作时才读取 `docs/specs/draft/`
8. 如任务已明确落在某个子目录，再补读该目录下的局部 `README.md`，但只把它当作实现注释而不是规范裁决来源
9. 遇到问题时参照 `docs/guides/debugging-guide.md` 系统化排查

## 维护原则

`docs/ai/repo-rules.md` 与 `docs/` 是唯一维护中的 repo-level canonical Markdown 文档体系。`AGENTS.md` 和 `CLAUDE.md` 是代理入口文件，委托给 `docs/ai/repo-rules.md`。局部 `README.md` 可以存在，但只能补充子目录实现细节，不能重新建立影子规范树。

根目录 `README.md` 是例外中的入口文档：应持续维护，但定位是 onboarding / product-facing guide，而不是与 `docs/ai/repo-rules.md` 并列的工程规则源。

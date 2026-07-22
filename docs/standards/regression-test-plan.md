---
status: active
doc_type: standard
owner: engineering
last_reviewed: 2026-07-22
source_of_truth:
  - docs/ai/repo-rules.md
  - docs/standards/testing-and-verification.md
  - docs/guides/testing-guide.md
  - server/
  - static/
---

# 回归测试计划

## 目的

本文档不是新的强制性规则来源。它把已有测试标准翻译为增量实施计划，回答三个实际问题：

- 本仓库应优先补哪些回归测试？
- 为防止每类 bug 再次发生，最少需要什么测试？
- 如何在不进行一次性测试基础设施大改造的前提下，让回归覆盖增量提升？

目标受众：

- 修复 bug 的开发者
- 重构模块的开发者
- PR 评审者
- 文档与工程标准维护者

## 目标

- 通过本地测试捕获"修过一次又坏了"的问题
- 为高风险边界优先补覆盖：权限、兜底、查询 join、契约
- 让新测试贴合当前代码结构，无需先做大规模新基础设施
- 让每个模块逐步积累稳定的回归示例集

## 非目标

- 不要求一次性回填所有模块测试
- 不要求为单个 bug 修复搭建完整 e2e 框架
- 不把 build / lint / typecheck 当作行为级回归测试
- 现阶段不推送高维护成本的 UI 组件快照测试

## 核心策略

默认在"最贴近 bug 的那一层"测试：

1. 若 bug 在纯逻辑、归一化或权限校验中，优先 service / helper 单测。
2. 若 bug 在 repository 查询、join、兜底或字段映射中，优先 repository 回归测试。
3. 若 bug 在 API 契约、响应整形或分页外壳中，优先 handler / API 契约测试。
4. 若 bug 在前端纯 helper、过滤参数转换或名称兜底中，优先前端单测。
5. 若 bug 只在页面装配层显现，但根因是后端契约问题，先补后端测试，再验证前端构建。

不要默认选择最重的测试层。选择能锁定真实风险的最小、最稳定层。

## 风险层与推荐测试类型

| 风险类型 | 常见示例 | 最低推荐测试 |
| --- | --- | --- |
| 权限边界 | admin 编辑 admin、guest 访问 login 页 | service 单测 |
| 输入校验/归一化 | 昵称、QQ、Discord、时间范围、枚举纠错 | service 或 helper 单测 |
| Repository 查询 join | join 后列歧义、漏过滤、排序错 | repository SQL / 查询形态测试 |
| Repository 兜底/合并 | 昵称回退到角色名、角色列表回退到 guest | repository 行为测试 |
| API 契约 | 字段名变更、`roles[]` 与 `role` 差异、分页结构 | handler 或 API 契约测试 |
| 前端纯逻辑 | 过滤参数转换、兜底文案、表格 helper | `pnpm test:unit` |
| 本地化回归 | 缺键、页面显示原始键 | JSON 校验 + 页面变更时手工验证 |
| 页面装配错误 | 列映射错、字段绑定错、按钮条件错 | 优先 helper / 契约测试；必要时补轻量前端测试 |

## 当前仓库优先级

第一优先级模块：

- `operation`：fleets、fleet-detail、pap、fleet-configs

  Fleet-configs 缺陷修复应优先补 EFT 解析/重建往返与装备设置保留的回归覆盖。特别是：仅当 `flag + type_id + quantity` 不变时才保留设置，变更或移除的项应重置为默认值。
- `system`：user、role、auto-role、pap、webhook
- `auth-and-characters`：`/api/v1/me`、角色绑定、个人资料完整度

理由：

- 这些模块同时涉及权限、查询 join、前后端契约与展示兜底
- 近期已出现过 join 查询回归与展示字段兜底回归
- 这些模块日常使用影响大，且 bug 通常在编译期无法捕获

第二优先级模块：

- `srp`
- `commerce`
- `info-and-reporting`
- `skill-planning`

第三优先级模块：

- 文档、静态配置、低风险只读页面

## 分阶段实施

### 阶段 1：锁定新 bug

目标：从现在起，所有新 bug 修复必须包含最低限度的回归测试。

要求：

- 每次修复 bug 时，先问"根因最贴近哪一层？"
- 在可合理测试时，针对该 bug 的回归测试是必备的
- 若基础设施当前缺失，至少补查询形态 / helper / service 层测试

完成标准：

- 新 bug 修复不再仅依赖 `go build`、Vue `vue-tsc` 或 React `tsc -b`
- 近期回归点开始拥有对应测试

建议早期示例：

- Fleet 列表 FC 昵称兜底
- join 后列歧义（`deleted_at`、`status`、`id`）
- 用户列表角色兜底与排序
- admin / super_admin 保护逻辑（super_admin 仅通过配置文件管理；API 不能分配/修改/删除）
- `/api/v1/me` 个人资料完整度与联系方式唯一性

### 阶段 2：补齐模块级高频回归点

目标：为经常修改的模块构建稳定的"保护带"。

每个高优先级模块至少应包含：

- 2 到 5 条 service / helper 回归测试
- 1 到 3 条 repository 回归测试
- 1 条关键契约测试

模块建议：

### Operation

- `fleet list` 查询 join 与 FC 展示兜底
- PAP 日志展示字段兜底逻辑
- 自动 SRP 模式归一化
- 舰队权限校验：`fc` / `admin` / `super_admin`

### Administration

- 用户资料更新校验与唯一性
- 受保护账号无法被普通 admin 修改/删除
- super_admin 角色无法通过 API 授予、修改或删除
- super_admin 用户无法通过 API 删除
- super_admin 角色在登录时从配置文件自动同步
- 角色列表 `roles[]` 与遗留 `role` 兜底
- `GET /system/basic-config` 仅返回固定系统标识符，没有对应的写接口
- auto-role `Director -> admin` 规则仅接受来自伏羲军团（`98185110`）的军团角色信号
- `allow_corporations` 在保存和读取时始终保留 `98185110`

### Auth And Characters

- 个人资料完整度检查
- 角色绑定/主角色切换权限与输入校验
- `guest` 到 `user` 的边界行为

### 阶段 3：构建共享测试夹具

目标：降低每个新测试的重复环境搭建成本。

建议新增（不要求一次性完成）：

- 后端 repository dry-run GORM helper
- 后端 handler 测试 helper
- 前端 locale JSON 校验 helper
- 前端 API 契约 mock helper

注意事项：

- 本仓库当前已适合做 dry-run SQL / schema mapping 测试
- 若 repository 集成测试显著增长，后续可考虑统一的测试数据库夹具
- 不要为假设性的未来用途预先搭建复杂测试平台

## 具体测试模式

### 1. Repository 查询形态测试

适用场景：

- join 变更
- SQL select / where / order / fallback 变更
- 新增计算字段

目的：

- 确保关键 SQL 片段存在
- 确保不再次出现列歧义
- 确保保留兜底表达式

示例：

- `fleet.deleted_at IS NULL`
- `LEFT JOIN "user"`
- `COALESCE(NULLIF("user".nickname, ''), fleet.fc_character_name)`

此类测试特别适合本仓库，因为：

- 运行快
- 不依赖真实数据库
- 能捕获近期发生过的 join 回归

### 2. DTO / schema 映射测试

适用场景：

- 查询新增别名字段
- 某个 DTO 字段仅用于响应、不持久化
- GORM tag 拼写错误导致查询返回数据无法映射

目的：

- 确保查询别名能正确 scan 进 DTO
- 确保字段名与 JSON / DBName tag 对齐

### 3. Service 行为测试

适用场景：

- 权限校验
- 兜底规则
- 输入归一化
- 唯一性校验

目的：

- 锁定业务规则
- 防止策略散落在 handler 或页面中而没有保护

### 4. Handler / API 契约测试

适用场景：

- 分页结构变更
- 字段名变更
- 响应外壳变更
- 重要端点的权限边界变更

目的：

- 防止"后端能编译，但前端契约已坏"

### 5. 前端单测

适用场景：

- 纯 helper
- hook 中的纯计算
- 过滤参数转换
- 兜底文案选择

目的：

- 用最轻的方式保护前端行为

现阶段不优先：

- 标准列表页的重型组件挂载测试
- 针对简单文案变更的端到端浏览器测试

## 缺陷修复最低回归要求

修复 bug 时，直接使用下表：

| Bug 根因 | 最低要求 |
| --- | --- |
| Service 规则错 | 一条 service 测试 |
| Repository 查询错 | 一条 repository 回归测试 |
| 响应字段错 | 一条 handler / 契约测试 |
| 前端 helper 错 | 一条前端单测 |
| 多层涉及 | 根因层测试 + 另一层构建验证 |

若无法立即补测试，变更描述必须写明：

- 为什么现在没补
- 缺什么基础设施
- 后续应在何处补

## 评审清单

评审 bug 修复时，至少问：

1. bug 根因在 handler、service、repository 还是前端 helper？
2. 新测试是否真正锁定根因，而不只是表面行为？
3. 若日后有人修改同一逻辑，这条测试是否会立即失败？
4. 除了 build / lint / typecheck，是否有行为级保护？

## 建议的模块回归清单

以下不是一次性任务清单，而是增量覆盖的优先级队列。

### auth-and-characters

- `ProfileComplete()` 与前端个人资料完整度检查保持一致
- QQ / Discord 唯一性
- `/api/v1/me` 返回角色与权限上下文

### operation

- Fleet 列表查询与展示兜底
- Fleet 管理权限校验
- PAP 发放前置条件
- 自动 SRP 模式归一化与触发条件

### administration

- 用户列表 DTO 不再泄漏遗留 `role`
- 用户角色排序与兜底
- admin 无法操作受保护账号
- auto-role 内置快捷规则与 title 映射的区别
- super_admin 的配置驱动同步与 API 不可变性

### commerce

- 购买限制规则
- 订单状态流转
- 钱包交易类型与引用类型映射

### srp

- SRP 申请状态流转
- 舰队 / KM 关联兜底
- 自动审批与人工审批边界

## 命令参考

验证命令见 `docs/standards/testing-and-verification.md`。

## 文档维护

当某模块开始积累稳定的回归测试时，更新对应 feature 文档，至少写明：

- 该模块当前的关键不变量
- 近期新增的高风险保护点
- 哪个测试层在保护这些不变量

不要把具体测试文件清单复制到多份文档里重复维护。测试策略由以下文件统管：

- `docs/standards/testing-and-verification.md`
- `docs/guides/testing-guide.md`
- 本文档

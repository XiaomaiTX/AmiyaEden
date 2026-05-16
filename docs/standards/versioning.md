---
status: active
doc_type: standard
owner: engineering
last_reviewed: 2026-05-16
source_of_truth:
  - static/package.json
  - CHANGELOG.md
  - docs/standards/versioning.md
---

# Versioning Standard

## Scope

本规范适用于 AmiyaEden 项目的所有版本号管理，包括前端和后端的版本号递增、发布和记录。

## Version Format

项目使用 **Semantic Versioning (SemVer)** 格式：`MAJOR.MINOR.PATCH`

### 格式说明

```
MAJOR.MINOR.PATCH
  ^    ^     ^
  |    |     |
  |    |     +--- PATCH (c)
  |    +--------- MINOR (b)
  +-------------- MAJOR (a)
```

### 版本号级别定义

| 级别 | 版本号 | 更新场景 | 示例 |
|------|--------|----------|------|
| **MAJOR** | a | 大版本更新：包含很多内容、大量重构或大量破坏性更新 | `1.0.0` → `2.0.0` |
| **MINOR** | b | 功能更新：新增功能、模块或重要的功能改进 | `1.0.0` → `1.1.0` |
| **PATCH** | c | 修复小 bug、小视觉问题或安全推送 | `1.0.0` → `1.0.1` |

## Version Increment Rules

### MAJOR (a) 版本更新

**触发条件**：
- 大规模架构重构
- 破坏性 API 变更（不兼容的修改）
- 大量核心功能重写
- 数据库结构重大变更（迁移脚本无法自动处理）
- 移除重要的已弃用功能

**示例场景**：
- 重构整个认证系统
- 从单体架构迁移到微服务
- 重大数据库 schema 变更
- 移除已弃用超过 2 个版本的 API

**更新流程**：
1. 创建新分支：`feature/major-upgrade-{version}`
2. 更新版本号：`static/package.json` 中的 `version` 字段
3. 在 CHANGELOG 中记录所有破坏性变更
4. 编写迁移指南
5. 进行完整的回归测试
6. 发布新版本并打 Git tag

### MINOR (b) 版本更新

**触发条件**：
- 新增功能模块或页面
- 新增 API 端点
- 新增重要的功能特性
- 用户体验显著改进（不影响现有功能）
- 非破坏性的 API 扩展

**示例场景**：
- 新增福利审批模块
- 新增技能规划功能
- 新增 SRP 管理功能
- 优化用户界面交互
- 新增第三方集成

**更新流程**：
1. 创建新分支：`feature/{feature-name}`
2. 更新版本号：`static/package.json` 中的 `version` 字段
3. 在 CHANGELOG 中记录新增功能
4. 进行功能测试和回归测试
5. 发布新版本并打 Git tag

### PATCH (c) 版本更新

**触发条件**：
- 修复小 bug
- 修复视觉问题（UI/UX 小问题）
- 安全漏洞修复
- 性能优化（不影响功能）
- 文档错误修正
- 代码重构（不影响功能）

**示例场景**：
- 修复表格滚动失效
- 修复按钮点击无响应
- 修复样式显示问题
- 修复 API 响应错误
- 安全漏洞补丁
- 优化页面加载速度

**更新流程**：
1. 创建新分支：`fix/{bug-description}`
2. 更新版本号：`static/package.json` 中的 `version` 字段
3. 在 CHANGELOG 中记录修复内容
4. 进行相关功能测试
5. 发布新版本并打 Git tag

## Code Changes and Version Updates

### 强制更新版本号的场景

任何对以下文件的修改都必须更新版本号：

#### 前端文件

| 文件/目录 | 修改类型 | 最低版本号级别 |
|-----------|----------|----------------|
| `static/src/views/*` | 新增页面 | MINOR (b) |
| `static/src/views/*` | 修改现有页面 | PATCH (c) |
| `static/src/api/*` | 新增 API 端点 | MINOR (b) |
| `static/src/api/*` | 修改现有 API | PATCH (c) |
| `static/src/components/*` | 新增公共组件 | MINOR (b) |
| `static/src/components/*` | 修改现有组件 | PATCH (c) |
| `static/src/router/*` | 新增路由 | MINOR (b) |
| `static/src/router/*` | 修改路由 | PATCH (c) |
| `static/src/store/*` | 修改状态管理 | PATCH (c) |
| `static/src/styles/*` | 样式修改 | PATCH (c) |

#### 后端文件

| 文件/目录 | 修改类型 | 最低版本号级别 |
|-----------|----------|----------------|
| `server/internal/handler/*` | 新增 handler | MINOR (b) |
| `server/internal/handler/*` | 修改现有 handler | PATCH (c) |
| `server/internal/service/*` | 新增 service | MINOR (b) |
| `server/internal/service/*` | 修改现有 service | PATCH (c) |
| `server/internal/repository/*` | 新增 repository | MINOR (b) |
| `server/internal/repository/*` | 修改现有 repository | PATCH (c) |
| `server/internal/model/*` | 新增 model | MINOR (b) |
| `server/internal/model/*` | 修改现有 model | PATCH (c) |
| `server/internal/router/*` | 新增路由 | MINOR (b) |
| `server/internal/router/*` | 修改路由 | PATCH (c) |
| `server/migrations/*` | 新增迁移脚本 | PATCH (c) |
| `server/migrations/*` | 破坏性迁移 | MAJOR (a) |

### 版本号更新决策树

```
修改代码
   │
   ├─ 是否修改了数据库结构？
   │   ├─ 是 → 是否破坏现有数据？
   │   │   ├─ 是 → MAJOR (a)
   │   │   └─ 否 → PATCH (c)
   │   │
   │   └─ 否 → 继续
   │
   ├─ 是否新增了功能/页面/API？
   │   ├─ 是 → MINOR (b)
   │   │
   │   └─ 否 → 继续
   │
   ├─ 是否修复了 bug/视觉问题/安全漏洞？
   │   ├─ 是 → PATCH (c)
   │   │
   │   └─ 否 → 继续
   │
   └─ 是否进行了大规模重构？
       ├─ 是 → MAJOR (a)
       │
       └─ 否 → 不需要更新版本号
```

## Version Update Workflow

### 0. 版本核查

在执行任何版本号递增前，必须同时检查 `static/package.json` 和 `CHANGELOG.md` 的 Git 历史。

上一个版本的位置要以 `static/package.json` 的最近一次版本更新提交和 `CHANGELOG.md` 的最近一次版本段落共同确认，而不是只看 `static/package.json` 的提交记录、当前工作区状态或 tag。

通常使用下面的命令定位上一版本对应的提交：

```bash
git log --follow -- static/package.json
git log --follow -- CHANGELOG.md
```

检查时至少要确认以下内容：

1. `static/package.json` 最近一次 version 更新提交
2. `CHANGELOG.md` 最近一次版本段落提交
3. 两者之间是否只有依赖更新、文档整理或其他非发布提交
4. 该版本边界之后的全部代码、文档和配置变更是否都已纳入版本级别判断与 `CHANGELOG` 记录

不要只根据当前工作区的单个文件变更判断版本号，也不要跳过历史提交检查直接 bump version。特别是 `static/package.json` 中如果混有依赖更新提交，必须用 `CHANGELOG.md` 的历史交叉确认，不能把这些提交误认为版本边界。

### 0.1 提交范围复核

在确认版本边界后，必须先列出该边界到当前 `HEAD` 之间的全部提交，再开始整理 `CHANGELOG`。

过去出现漏写提交的根因，就是没有先生成完整提交清单，而是直接从当前 diff 或最近一次 changelog 标题整理发布说明，导致 `release-worthy` 提交可能被静默跳过。

建议使用下面的命令生成提交清单：

```bash
git log --reverse --oneline <上一个版本边界提交>..HEAD
```

复核时必须为提交清单中的每个提交分配一个明确分类：

- `release-worthy`：直接影响用户可见行为、接口、数据、权限、路由、文案、测试或发布说明的提交，必须进入 `CHANGELOG.md`
- `docs-only`：仅文档、注释、说明或非功能性排版改动的提交，可以不进入 `CHANGELOG.md`
- `dependency-only`：仅依赖或锁文件更新，且不改变产品行为的提交，可以不进入 `CHANGELOG.md`
- `non-release`：与本次发布无关、已被后续提交覆盖或撤销、或仅为流程性维护且不影响发布说明的提交，可以不进入 `CHANGELOG.md`

如果某个提交不写入 `CHANGELOG.md`，必须在发布核对清单、PR 描述或其他发布准备记录中写明该提交的 hash、分类和排除理由，避免“看漏了”和“故意不写”混在一起。

发布内容必须以这份提交范围清单为输入生成，不能只看当前 diff、最近一次 changelog 标题、单个文件的 Git 历史，或者凭记忆补写。

### 1. 开发阶段

```bash
# 确定版本号级别
# 根据修改内容判断是 MAJOR/MINOR/PATCH

# 拉取最新代码
git pull origin main

# 创建功能分支
git checkout -b feature/your-feature-name

# 进行开发...
```

### 2. 更新版本号

```bash
# 修改 static/package.json 中的 version 字段
# 例如：1.0.0 → 1.1.0 (MINOR 更新)

# 提交版本号更新
git add static/package.json
git commit -m "chore: bump version to 1.1.0"

# 或在 static 目录中使用版本命令自动更新
cd static
npm version minor  # 自动更新 static/package.json 并创建 commit
```

### 3. 更新 CHANGELOG

在项目根目录的 `CHANGELOG.md` 中记录变更：

```markdown
## [1.1.0] - 2026-04-10

### Added
- 新增福利审批模块
- 新增技能规划功能

### Fixed
- 修复表格滚动失效问题
- 修复按钮点击无响应

### Changed
- 优化用户界面交互
```

### 4. 测试和验证

```bash
# 完整前端检查
cd static
pnpm lint .
pnpm exec vue-tsc --noEmit
pnpm test:unit

# 构建项目
pnpm build
```

每次 bump up version 前都必须完成上述完整前端检查和 `pnpm build`。

### 5. 发布版本

```bash
# 合并到主分支
git checkout main
git merge feature/your-feature-name

# 打 Git tag
git tag -a v1.1.0 -m "Release version 1.1.0"

# 推送到远程仓库
git push origin main
git push origin v1.1.0
```

### 6. Docker 镜像构建

```bash
# 使用版本号构建 Docker 镜像
# 版本号会在前端构建时通过 sync-version.js 自动注入
docker build -t amiya-eden:latest .
```

## Common Scenarios

### 场景 1：新增一个功能模块

**示例**：新增福利审批模块

```bash
# 1. 创建分支
git checkout -b feature/welfare-approval

# 2. 开发功能
# - static/src/views/welfare/approval/index.vue
# - static/src/api/welfare.ts
# - server/internal/handler/welfare.go
# - server/internal/service/welfare.go
# - server/internal/repository/welfare.go

# 3. 更新版本号
cd static
npm version minor  # 1.0.0 → 1.1.0

# 4. 更新 CHANGELOG
## [1.1.0] - 2026-04-10
### Added
- 新增福利审批模块

# 5. 测试、合并、发布
```

**版本号更新**：MINOR (b)

### 场景 2：修复一个 bug

**示例**：修复表格滚动失效

```bash
# 1. 创建分支
git checkout -b fix/table-scroll

# 2. 修复问题
# - static/src/components/core/layouts/art-table-card/index.vue

# 3. 更新版本号
cd static
npm version patch  # 1.1.0 → 1.1.1

# 4. 更新 CHANGELOG
## [1.1.1] - 2026-04-10
### Fixed
- 修复表格滚动失效问题

# 5. 测试、合并、发布
```

**版本号更新**：PATCH (c)

### 场景 3：重构认证系统

**示例**：从 JWT 迁移到 OAuth2

```bash
# 1. 创建分支
git checkout -b refactor/auth-system

# 2. 重构代码
# - 大量修改认证相关代码
# - 修改 API 端点
# - 更新数据库结构

# 3. 更新版本号
cd static
npm version major  # 1.1.1 → 2.0.0

# 4. 更新 CHANGELOG
## [2.0.0] - 2026-04-10
### Changed
- 重构认证系统，从 JWT 迁移到 OAuth2
- 破坏性变更：所有 API 端点需要新的认证头

### Migration
- 查看迁移指南：docs/migration/v2.0.0.md

# 5. 编写迁移指南
# 6. 完整测试
# 7. 合并、发布
```

**版本号更新**：MAJOR (a)

### 场景 4：修复样式问题

**示例**：修复按钮颜色显示不正确

```bash
# 1. 创建分支
git checkout -b fix/button-color

# 2. 修复样式
# - static/src/styles/components/button.scss

# 3. 更新版本号
cd static
npm version patch  # 1.1.1 → 1.1.2

# 4. 更新 CHANGELOG
## [1.1.2] - 2026-04-10
### Fixed
- 修复按钮颜色显示不正确的问题

# 5. 测试、合并、发布
```

**版本号更新**：PATCH (c)

## Version Display

项目使用统一的版本号显示：

### 版本号显示

- **来源**：`static/package.json` 中的 `version` 字段
- **显示位置**：顶部栏右侧
- **显示格式**：`v{version}`

### 版本号同步

- 前端版本号在构建前通过 `static/scripts/sync-version.js` 自动同步到环境变量
- 构建时通过 Vite 的 `define` 功能注入到应用中

## Related Files

### Version Files

- `static/package.json` - 统一版本号
- `CHANGELOG.md` - 变更日志（建议创建）

### Build Files

- `static/scripts/sync-version.js` - 版本同步脚本
- `static/package.json` - 构建前钩子配置
- `static/.env.development` - 开发环境变量
- `static/.env.production` - 生产环境变量
- `static/vite.config.ts` - Vite 配置（环境变量注入）

### Documentation Files

- `docs/standards/versioning.md` - 本文档
- `docs/specs/archive/version-display-implementation.md` - 版本显示功能实施方案

## Best Practices

### 1. 版本号递增原则

- **从小到大**：优先使用 PATCH，其次是 MINOR，最后是 MAJOR
- **保守递增**：不确定时选择较小的版本号级别
- **及时更新**：每次发布都必须更新版本号
- **先清单后写作**：先完成提交范围复核和分类，再写 `CHANGELOG.md`
- **一条提交一条依据**：每个 `release-worthy` 提交都必须能在发布记录中找到对应说明
- **排除有理由**：任何未写入 `CHANGELOG.md` 的提交都必须有明确排除理由

### 2. CHANGELOG 维护

- **详细记录**：记录所有重要的变更
- **分类清晰**：按 Added/Changed/Fixed/Deprecated/Removed 分类
- **时间标记**：每个版本都标记发布日期

### 3. Git Tag 规范

- **格式**：`v{MAJOR.MINOR.PATCH}`，例如 `v1.2.3`
- **注释**：为每个 tag 添加描述信息
- **推送**：tag 必须推送到远程仓库

### 4. 分支命名规范

- **功能分支**：`feature/{feature-name}`
- **修复分支**：`fix/{bug-description}`
- **重构分支**：`refactor/{description}`
- **发布分支**：`release/{version}`

### 5. 发布前检查清单

- [ ] 版本号已更新
- [ ] CHANGELOG 已更新
- [ ] 所有测试通过
- [ ] Lint 检查通过
- [ ] 构建成功
- [ ] Git tag 已创建
- [ ] CHANGELOG 提交
- [ ] 版本显示正常工作

## FAQ

### Q: 小改动也需要更新版本号吗？

**A**: 不一定。如果是代码格式调整、注释修改、文档更新等不影响功能的改动，不需要更新版本号。只有涉及功能、bug 修复或用户体验的改动才需要更新版本号。

### Q: 如何判断是 MINOR 还是 PATCH？

**A**: 判断标准是是否新增了功能。如果是新增功能、页面、API 端点等，使用 MINOR。如果是修复现有问题，使用 PATCH。

### Q: 多个 bug 修复可以合并为一个 PATCH 版本吗？

**A**: 可以。多个小的 bug 修复可以合并为一个 PATCH 版本发布。

### Q: 版本号回滚怎么处理？

**A**: 版本号不应该回滚。如果新版本有问题，应该发布新的 PATCH 版本来修复问题，而不是回滚版本号。

### Q: 前端和后端版本号需要保持一致吗？

**A**: 项目使用统一的版本号，版本值记录在 `static/package.json` 中，发布边界再通过 `CHANGELOG.md` 交叉确认。前端和后端共享同一个版本号，简化版本管理。

## Examples

### Valid Version Sequences

```
1.0.0 → 1.0.1 → 1.0.2 → 1.1.0 → 1.1.1 → 1.2.0 → 2.0.0
```

### Invalid Version Sequences

```
1.0.0 → 2.0.0  (跳过了 MINOR 版本)
1.1.0 → 1.0.0  (版本号回滚)
1.0.0 → 1.0.0  (版本号未更新)
```

## References

- [Semantic Versioning 2.0.0](https://semver.org/)
- [Keep a Changelog](https://keepachangelog.com/)
- 项目版本显示功能文档：`docs/features/current/version-display.md`

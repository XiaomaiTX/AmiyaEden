---
status: active
doc_type: guide
owner: engineering
last_reviewed: 2026-03-23
source_of_truth:
  - AGENTS.md
  - docs/README.md
  - docs/ai/harness-principles.md
---

# AI Agent Onboarding

## 目标

本文件帮助未来的 AI agent 在最短路径内读到正确文档，并在代码与文档不一致时做出保守、可维护的判断。

## Harness Context

This repository uses harness engineering principles. Before starting any work:

1. **Load the harness**: `AGENTS.md` is the primary execution harness. It defines golden principles, architecture rules, dependency direction, pre-completion protocol, and behavioral guardrails.
2. **Understand the trust hierarchy**: `AGENTS.md` > `docs/standards/` > `docs/architecture/` > `docs/api/` > `docs/features/current/` > `docs/guides/` > `docs/specs/draft/`
3. **Context boundaries apply**: you can only reason about what's in the repository. External knowledge (chat threads, verbal decisions) is not context.

For the full harness engineering philosophy, see `docs/ai/harness-principles.md`.

## 最小阅读顺序

### 处理后端 / API 变更

1. `AGENTS.md`
2. `docs/README.md`
3. `docs/architecture/overview.md`
4. `docs/architecture/module-map.md`
5. `docs/architecture/auth-and-permissions.md`
6. `docs/api/conventions.md`
7. `docs/api/route-index.md`
8. 对应 feature doc

### 处理前端页面 / 路由 / 权限

1. `AGENTS.md`
2. `docs/README.md`
3. `docs/architecture/module-map.md`
4. `docs/architecture/routing-and-menus.md`
5. `docs/standards/frontend-table-pages.md`
6. 对应 feature doc

### 处理 ESI / SSO / CCP 数据同步

1. `AGENTS.md`
2. `docs/README.md`
3. `docs/architecture/overview.md`
4. `docs/architecture/module-map.md`
5. `docs/architecture/runtime-and-startup.md`
6. `docs/features/current/auth-and-characters.md`
7. `docs/features/current/esi-refresh.md`
8. `docs/guides/adding-esi-feature.md`
9. 只有在任务已经确定落在 `server/pkg/eve/esi/` 时，再读该目录下的局部 `README.md`

## 冲突处理规则

当文档之间互相冲突时：

1. 先信 `AGENTS.md`
2. 再信 `docs/` 中更高层级的 active 文档
3. `docs/templates/` 与局部目录 `README.md` 不作为规范裁决依据
4. 旧兼容文件不作为裁决依据

当代码与文档冲突时：

1. 把代码视为当前实现
2. 评估这是“代码漂移”还是“文档过时”
3. 如果任务允许，优先把 canonical 文档修正到当前实现
4. 不要为了迎合旧文档去回滚用户已有实现

## 修改前检查

- 阅读目标模块周边代码，而不是只看一个文件
- 找到对应 feature doc 与 API / architecture 文档
- 明确这次改动属于：标准、现状、接口、功能、提案中的哪一种
- 如果只是未来想法，不要改写 current-state 文档

## 修改后最少更新

- 行为变化：更新对应 feature doc
- 路由或权限边界变化：更新 `docs/api/route-index.md`
- 运行 / 启动方式变化：更新 `docs/architecture/runtime-and-startup.md`
- 规范变化：更新 `AGENTS.md` 或 `docs/standards`

## 行为守则

### 必须遵守

- 每次任务前加载 `AGENTS.md`
- 修改代码前先读对应模块的现有代码和 feature doc
- 遵循 pre-completion verification protocol (`AGENTS.md` "Pre-Completion Protocol")
- 被阻塞时停下来重新分析，不要反复重试同一种方法
- 发现文档与代码不一致时，评估是文档漂移还是代码漂移

### 不该做的事

- 重新建立第二套影子文档树
- 把”计划中的行为”写进 architecture / feature current
- 在多个文档里维护同一份角色、权限、路由清单
- 看到旧标题就假设旧内容仍然正确
- 把模板文件或模块局部 `README.md` 当成 repo-level source of truth
- 编辑同一个文件超过 3 次仍未解决问题（loop detection — see `AGENTS.md` "Agent Behavioral Guardrails"）
- 引入与现有仓库约定矛盾的新模式
- 给未修改的代码添加注释、docstring 或类型注解
- 为一次性操作创建工具函数或抽象

## 调试指引

遇到问题时，按照 `docs/guides/debugging-guide.md` 的系统化流程排查：

1. 分类问题（构建失败、类型错误、运行时错误、权限问题等）
2. 定位层级（handler / service / repository / view / api）
3. 最小范围复现
4. 在根因处修复
5. 补回归测试

## 验证参考

完成任务后，按变更类型使用 `docs/standards/pre-completion-checklist.md` 中的对应检查清单。

快速命令参考：

```bash
# 后端
cd server && go build ./...
cd server && go test ./...

# 前端
cd static && pnpm lint .
cd static && pnpm exec vue-tsc --noEmit
cd static && pnpm test:unit
```

---
status: active
doc_type: standard
owner: engineering
last_reviewed: 2026-11-24
source_of_truth:
  - docs/ai/repo-rules.md
  - docs/standards/regression-test-plan.md
  - server/go.mod
  - static/package.json
  - static-react/package.json
---

# 测试与验证规范

## 适用范围

适用于后端、前端、契约、repository、hook、handler 与 service 变更。

## 必需工具版本

| 工具 | 版本 | 安装方式 |
| --- | --- | --- |
| golangci-lint | v2.11.4 | `go install github.com/golangci/golangci-lint/cmd/golangci-lint@v2.11.4` |
| pnpm | 11.15.1 | `npm install -g pnpm@11.15.1` |
| Node.js | 24 | 见仓库根目录 `.nvmrc` |

- CI 在 `.github/workflows/verify-ci.yaml` 中锁定 `golangci-lint`。
- Vue 前端由 `static/pnpm-lock.yaml` 锁定，React 前端由 `static-react/pnpm-lock.yaml` 锁定；分别使用 `pnpm install --frozen-lockfile`。

## 默认命令

验证命令的唯一权威来源。

### 后端

- `cd server && golangci-lint run ./...`
- `cd server && go test ./...`
- `cd server && go build ./...`

### 前端

Vue：

- `cd static && pnpm lint .`
- `cd static && pnpm exec vue-tsc --noEmit`
- `cd static && pnpm test:unit`
- `cd static && pnpm build`

React：

- `cd static-react && pnpm lint`
- `cd static-react && pnpm exec tsc -b`（独立本地类型检查；React 构建已包含该检查）
- `cd static-react && pnpm test`
- `cd static-react && pnpm check:api-contract`
- `cd static-react && pnpm build`

## 规则

- `build`、`lint`、`typecheck` 不替代行为级覆盖。
- 测试必须真正执行被改动的逻辑，不得在测试里复刻生产逻辑。
- 新功能在可合理测试的前提下，必须新增或更新相关自动化覆盖。
- 已有功能变更若覆盖的行为或契约发生变化，必须审查并更新邻近测试。
- 缺陷修复在可合理测试的前提下，必须新增或更新回归覆盖。
- 后端逻辑变更应在同一 Go package 下补 `_test.go` 覆盖。
- Repository 的分支、过滤、合并、查询与兜底逻辑必须为关键分支补 Go 测试。
- 纯前端 helper 或 hook 逻辑应由对应前端的单测覆盖。
- API 契约变更必须同时校验后端与前端，并在至少一侧补行为级覆盖。
- 任何写入文档的测试命令必须能按字面运行。

## 测试选型

- 服务规则、归一化、权限校验、查询 helper、repository 分支与兜底逻辑优先使用后端 Go 测试。
- 纯 helper、纯 hook、确定性状态流转、合并逻辑、兜底逻辑与请求映射优先使用前端单测。
- 当轻量单元即可覆盖行为时，避免为小逻辑改动引入重型测试基础设施。

## 允许的例外

仅在写明原因且符合以下任一时才可省略测试：

- 纯文档变更
- 纯格式化变更
- 明确保持行为的重命名
- 基础设施缺失导致临时搭建成本与变更严重失衡
- 因外部依赖或运行条件限制，难以在仓库内可靠测试

## 最低验证要求

- 后端变更 -> 执行[后端 lint、测试与构建](#后端)
- 前端变更 -> 对每个受影响的前端执行 [lint 与 typecheck](#前端)；改动 helper、hook 等可单测逻辑时，执行对应前端的单测
- API 契约变更 -> 执行上述后端和实际消费该契约的前端检查；React 消费该契约时，执行 React API 契约隔离检查；涉及 OpenAPI 时，按[契约流程](../api/conventions.md#契约同步顺序)核对生成物与路由的一致性
- 新功能 -> 除非显式声明允许例外，必须为新增行为新增或更新相关自动化覆盖
- 缺陷修复 -> 除非显式声明允许例外，必须新增或更新回归覆盖
- 已有功能行为变更 -> 按需审查并更新现有测试，并为新增或变化的行为补覆盖，除非显式声明允许例外
- 纯文档变更 -> 无需代码级验证，除非命令或可执行示例发生变化

## 后端设置项规则

- 设置与配置结构体必须将 `0` 视为有效配置值，而不是缺失或未设置，除非该字段显式声明为指针或可选类型。
- 设置路径的回归测试必须包含零值用例，前提是业务逻辑可能基于 `== 0` 进行分支。

## Repository 注意事项

- 前端单测刻意保持轻量，最适合纯逻辑。
- Vue 侧保留 `static/src/types/import/auto-imports.d.ts` 与 `static/src/types/import/components.d.ts`，以使干净 checkout 能通过 lint 与 typecheck；React 侧以其 `tsconfig`、API 契约检查和本地类型目录为准。
- 不得要求 `static/.auto-import.json` 才能跑 CI lint。
- 测试放置与实现指导见 `docs/guides/testing-guide.md`。
- 增量回归计划见 `docs/standards/regression-test-plan.md`。

## 完成核对

- 新功能或变更的功能行为：是否新增或更新了相关覆盖？
- 缺陷修复、契约变更、兜底变更、非平凡分支变更：是否新增或更新了回归覆盖？
- 是否执行了最低要求的命令？若某项必要的检查或测试未执行，写明具体项目、原因和适用的后续补测位置。
- 若新增了写入文档的测试命令，是否已在本地实际运行？

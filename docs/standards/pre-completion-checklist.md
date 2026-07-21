---
status: active
doc_type: standard
owner: engineering
last_reviewed: 2026-04-23
source_of_truth:
  - docs/ai/repo-rules.md
  - docs/standards/dependency-layering.md
  - docs/standards/documentation-governance.md
---

# 预完成核对清单

## 适用范围

在把工作标记为完成之前，使用本清单核对。仅可跳过与当前改动无关的条目。

## 核心规则

- 本文件是完成门槛。
- `docs/standards/testing-and-verification.md` 是命令、覆盖规则与允许例外的唯一权威来源。
- 如跳过某项必要的检查或测试，必须明确写明原因。
- 除非存在显式迁移要求，不得引入任何推测性的向后兼容别名、隐藏重定向、兜底载荷、废弃路由、重复端点/契约或影子页面。
- 难以从代码直接看出的持久后端或架构决策，必须按 `docs/standards/documentation-governance.md` 写入最近的权威文档。

## 按变更类型核对

### 仅后端变更

- [ ] `cd server && golangci-lint run ./...`
- [ ] `cd server && go build ./...`
- [ ] `cd server && go test ./...`
- [ ] 范围保持聚焦，未引入无关重构
- [ ] 未在缺乏显式迁移要求的情况下引入推测性的向后兼容别名、兜底或重复契约
- [ ] 未引入层级违反
- [ ] 若为缺陷修复，已新增或更新回归测试
- [ ] 若 API 契约发生变化，已更新前端 API wrapper 与类型
- [ ] 若新增或修改路由，已更新 `docs/api/route-index.md`
- [ ] 若行为发生变化，已更新相关 feature 文档
- [ ] 若存在难以从代码看出的持久后端设计决策，最近的权威文档已记录决策、理由、不变量与主要代码文件

### 仅前端变更

- [ ] `cd static && pnpm lint .`
- [ ] `cd static && pnpm exec vue-tsc --noEmit`
- [ ] 若改动了纯 helper 或 hook，执行 `cd static && pnpm test:unit`
- [ ] 范围保持聚焦，未引入无关重构
- [ ] 未在缺乏显式迁移要求的情况下引入推测性的向后兼容别名、兜底、隐藏重定向或影子页面
- [ ] 未在 view 中直接发起 HTTP 调用
- [ ] 所有新增用户可见文案已同时写入 `zh.json` 和 `en.json`
- [ ] 若行为发生变化，已更新相关 feature 文档
- [ ] 通过 `@click` / `@change` 等绑定的事件处理器具有与模板调用点匹配的类型签名；vue-tsc 仅在处理器类型显式时才能捕获不匹配

### 跨契约变更

- [ ] `cd server && golangci-lint run ./...`
- [ ] `cd server && go build ./...`
- [ ] `cd server && go test ./...`
- [ ] `cd static && pnpm lint .`
- [ ] `cd static && pnpm exec vue-tsc --noEmit`
- [ ] 如相关，执行 `cd static && pnpm test:unit`
- [ ] 范围保持聚焦，未引入无关重构
- [ ] 未在缺乏显式迁移要求的情况下引入推测性的向后兼容别名、兜底、隐藏重定向或重复契约
- [ ] 已更新前端 API wrapper
- [ ] 已更新共享 TypeScript 类型
- [ ] 后端响应字段与前端类型字段一致
- [ ] 若路由面或权限边界发生变化，已更新 `docs/api/route-index.md`
- [ ] 若行为发生变化，已更新相关 feature 文档
- [ ] 若存在难以从代码看出的持久后端或契约设计决策，最近的权威文档已记录决策、理由、不变量与主要代码文件

### 权限或角色变更

- [ ] 已完成"跨契约变更"的所有适用项
- [ ] 已按需更新后端路由保护
- [ ] 已按需更新前端路由元信息
- [ ] 已对齐 `v-auth` 等按钮权限用法
- [ ] 若路由或页面访问模型发生变化，已更新 `docs/architecture/routing-and-menus.md`
- [ ] 若权限模型或行为发生变化，已更新 `docs/architecture/auth-and-permissions.md`
- [ ] `docs/features/current/badge-counts.md` 中的角标字段可见性仍与更新后的角色边界一致；若不一致需同步更新

### 仅文档变更

- [ ] 范围保持聚焦，未引入无关重构
- [ ] 已按需更新 front matter
- [ ] 未引入失效引用或断链
- [ ] 已按需更新索引文档
- [ ] 当文档描述当前实现时，已核对当前代码

### 新功能或新模块

- [ ] 已完成"跨契约变更"的所有适用项
- [ ] 若新功能具有持久行为，已在 `docs/features/current/` 下创建 feature 文档
- [ ] 已按需更新相关 feature 索引
- [ ] 已在 `zh.json` 和 `en.json` 中完成本地化
- [ ] 已按需新增前端路由与页面访问元信息
- [ ] 已按需注册后端与前端路由
- [ ] 改动遵循现有模块结构模式
- [ ] 难以从代码看出的持久后端设计决策已记录在拥有该主题的 feature、API、architecture 或 standard 文档中
- [ ] 除非另有明确说明，至少有一条回归测试覆盖关键行为

## 测试选型参考

实践层面的测试放置与选型指导见 `docs/guides/testing-guide.md`。

## 当某项测试被跳过时

当某项通常预期的测试被跳过时：

1. 写明被跳过的测试
2. 写明跳过的原因
3. 如适用，写明后续应在何处补测试

不得在未记录原因的情况下跳过通常预期的测试。

---
status: active
doc_type: standard
owner: engineering
last_reviewed: 2026-11-24
source_of_truth:
  - docs/ai/repo-rules.md
  - docs/standards/dependency-layering.md
  - docs/standards/documentation-governance.md
  - docs/standards/testing-and-verification.md
---

# 预完成核对清单

## 适用范围

完成前核对所有适用项。跨层变更同时核对涉及的各层；纯文档变更只核对通用项和文档项。

## 通用核对

- [ ] 改动聚焦于任务；替换契约或路由时，遵循[有意替换规则](../ai/repo-rules.md#non-negotiable-rules)处理旧入口。
- [ ] 按[测试与验证规范](testing-and-verification.md#最低验证要求)完成适用的验证和回归覆盖，照其[完成核对](testing-and-verification.md#完成核对)记录未执行的必要检查。
- [ ] 行为变化时更新相关 feature 文档；新增或修改路由、路由面或权限边界时更新[路由索引](../api/route-index.md)。
- [ ] 难以从代码看出的持久后端或架构决策按[文档治理规范](documentation-governance.md#技术设计决策)记录在最近的权威文档中。

## 按变更类型核对

### 后端变更

- [ ] 改动符合[依赖分层规范](dependency-layering.md)。

### 前端变更

- [ ] View 通过 API 层发出请求；新增用户可见文案已同步写入 `zh.json` 和 `en.json`。
- [ ] Vue 事件处理器签名与绑定匹配；React 事件和组件 props 已通过 TypeScript 检查。

### 跨契约变更

- [ ] 后端与前端 wrapper、类型和 UI 用法已同步；后端响应字段与前端类型一致。

### 权限或角色变更

- [ ] 已按需更新后端路由保护、前端路由元信息及 Vue `v-auth` / React permission gate 或 hook 的按钮权限。
- [ ] 路由或页面访问模型变化时更新[路由与菜单](../architecture/routing-and-menus.md)；权限模型或行为变化时更新[认证与权限](../architecture/auth-and-permissions.md)。
- [ ] 若角色边界影响角标字段可见性，已核对并按需更新[角标文档](../features/current/badge-counts.md)。

### 仅文档变更

- [ ] 已按需更新 front matter 和索引；新增或修改的引用可打开。
- [ ] 描述当前实现时，已核对当前代码。

### 新功能或新模块

- [ ] 持久行为已记录到对应 feature 文档，并按需更新索引。
- [ ] 已按需注册后端与前端路由，补充页面访问元信息，并沿用现有模块结构。

测试放置和选型见[测试指南](../guides/testing-guide.md)。

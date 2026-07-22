---
status: active
doc_type: standard
owner: engineering
last_reviewed: 2026-07-22
source_of_truth:
  - static/src/utils/common/time.ts
  - static/src/utils/common/index.ts
---

# 时间戳格式规范

## 适用范围

适用于前端 UI 中所有面向用户的时间戳与日期时间展示。

## 核心规则

- Vue 复用 `static/src/utils/common/time.ts`；React 应复用其共享时间 helper（当前若未建立等价 helper，属于迁移缺口），不得在 view/page 或 component 中定义本地变体。

## 允许的例外

- 产品特定的相对时间展示，仅当 UI 规格明确要求时，才允许使用独立 helper。
- 仅在底层字段确实是日历日期（而非时间戳）、且模块文档或功能规格写明这一点时，才允许仅展示日期。

## 核对清单

- [ ] 所有面向用户的时间戳字段都使用 `formatTime`
- [ ] UI 时间戳渲染处没有残留的内联 `new Date(...).toLocaleString()`
- [ ] 任何例外都已在对应的 feature 或 architecture 文档中写明

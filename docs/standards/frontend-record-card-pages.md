---
status: active
doc_type: standard
owner: frontend
last_reviewed: 2026-04-17
source_of_truth:
  - static/src/assets/styles/core/app.scss
  - static/src/views/newbro/select-mentor/index.vue
  - static/src/views/newbro/mentor-manage/index.vue
  - static/src/views/newbro/select-captain/index.vue
  - static/src/views/info/assets/index.vue
---

# 前端记录卡片页规范

## 适用范围

本规范适用于主要内容为"以卡片、堆叠编辑器或嵌入卡片的无分页编辑表格"形式渲染的重复记录列表的前端页面。

典型场景包括候选人目录、审批卡片列表、奖励阶段编辑器，以及其他"记录数量可能超过少量固定上限"的布局。

若主列表是分页的页面级表格，请改用 `docs/standards/frontend-table-pages.md`。本规范覆盖的是"在卡片内重复渲染的记录"的溢出行为，包括未使用 table-page 模式的可编辑行表格。

## 核心规则

- 每一处重复记录区域都必须明确声明增长策略：页面扩展或内部滚动二选一。
- 禁止隐式裁剪。页面根、卡片体、tab 面板或任意包装器不得仅通过隐藏溢出来限制高度而不提供滚动拥有者，否则用户数据将变得不可达。
- 无界卡片列表与分阶段编辑器默认采用页面扩展。
  - 让页面高度自然增长。
  - 当页面主内容是无界记录卡片列表时，除非某个后代元素显式接管滚动，否则不要使用 `art-full-height`。
  - 当列表只需随内容增长时，优先使用应用外壳的滚动路径。
- 仅当滚动拥有者显式时才允许内部滚动。
  - 滚动元素必须使用 `overflow: auto`、`overflow-y: auto` 或 `ElScrollbar`。
  - 若页面使用 `art-full-height`，则必须通过 `display: flex`、`flex-direction: column` 与 `min-height: 0` 在每个参与布局的中间包装器上把高度链补全。
- 可能容纳大量记录的卡片，要么随页面扩展，要么包含明确的内部滚动区域，绝不能仅依赖隐藏溢出作为唯一约束。
- 混合页面可以同时使用本规范与 `docs/standards/frontend-table-pages.md`，但每一区域都必须有一个明确的溢出拥有者。

## 允许的例外

- 固定高度的仪表盘或指标卡，其内容本身就有意限定边界。
- 由对话框或抽屉容器本身接管滚动的场景。
- 条目数量保证很少、不会因数据或配置增长的静态摘要卡。

## 核对清单

- 当记录数量翻倍时，内容是否仍然可达？
- 当页面使用 `art-full-height` 时，每个无界区域是否都有唯一的显式滚动拥有者？
- 当页面不需要内部滚动时，是否避免了 `art-full-height` 并改为页面扩展？
- 是否避免了在记录容器上单独使用 `overflow: hidden` 而没有嵌套滚动区域？
- 当页面同时存在卡片和表格时，是否对卡片使用卡片列表规则、对表格使用表格规则分别处理？

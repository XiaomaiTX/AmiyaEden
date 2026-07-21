---
status: active
doc_type: standard
owner: engineering
last_reviewed: 2026-04-03
source_of_truth:
  - static/src/utils/common/isk.ts
  - static/src/utils/common/index.ts
---

# ISK 格式规范

## 适用范围

适用于前端 UI 中所有面向用户、以 ISK 计价的数值。

不适用于伏羲币或其他非 ISK 货币的展示。

## 共享 Helper

- 从 `@/utils/common` 导入 ISK helper。
- 使用 `formatIskPlain(value)` 输出账本风格的精确金额。
- 使用 `formatIskSmart(value)` 输出紧凑摘要金额。
- 仅在编辑器明确以"百万"为单位暴露数字输入、但存储仍为 ISK 时，使用 `iskToMillionInput(value)` 与 `millionInputToIsk(value)`。
- 通用数字渲染组件在未引入额外 ISK 字符串格式化逻辑的前提下，可直接通过原始数字 prop 渲染精确数值；任何旁路的 ISK 摘要文案仍必须使用共享 helper。

## 标准展示样式

### 精确数值样式

- 输出完整数值，使用 `,` 千分位分组，并固定保留 `2` 位小数。
- 不得缩写单位。
- 示例：`711,103,702.38`

### 智能缩写样式

- 固定保留 `2` 位小数，并附大写单位后缀。
- 后缀前插入一个空格。
- 允许的后缀仅限 `K`、`M`、`B`、`T`。
- 当某单位下数值舍入到 `1000.00` 时，必须晋升到下一单位。
- 示例：`711.10 M`

## 展示位置规则

- 钱包余额、钱包流水、NPC 击杀报告金额、新人相关 ISK 展示使用 `formatIskPlain`。
- SRP 展示、合同列表/详情金额、仪表盘钱包说明等紧凑 ISK 摘要使用 `formatIskSmart`。
- 当同一处展示故意同时呈现两种样式时，精确数值保持 plain，辅助摘要文案使用 smart。

## 禁止模式

- 不得在 view、hook 或 component 中定义本地 `formatISK` helper。
- 不得在 UI 渲染逻辑中用内联 `Intl.NumberFormat('en-US', ...)` 或 `toLocaleString('en-US', ...)` 格式化 ISK。
- 不得用 `toFixed()` 手工拼接 `K` / `M` / `B` / `T` 字符串。

## 核对清单

- [ ] 所有显式的 ISK 字符串格式化逻辑都通过 `static/src/utils/common/isk.ts`
- [ ] 精确数值展示使用 `formatIskPlain`
- [ ] 紧凑摘要展示使用 `formatIskSmart`
- [ ] 以"百万"为单位的编辑器仅使用 `iskToMillionInput` 与 `millionInputToIsk`
- [ ] 伏羲币格式化保持独立，不混入 ISK helper

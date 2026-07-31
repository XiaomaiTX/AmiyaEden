---
status: draft
doc_type: draft
owner: engineering
last_reviewed: 2026-07-31
source_of_truth:
  - static-react/src/components/ui
  - static-react/src/pages
  - docs/standards/frontend-table-pages.md
---

# React shadcn/ui 规范收敛债务清单

## 规则

- `static-react/src/components/ui/` 中从 shadcn/ui 上游导入的组件必须保持上游实现原样；默认不得直接修改。确有必要偏离时，须先获得明确同意，并在同一变更中记录组件、原因、上游版本和后续重新导入策略。
- 对上游组件的视觉或业务定制应优先放在调用处、共享组合组件或全局主题令牌中；不得把本地业务逻辑、文案或页面状态混入上游组件文件。
- React UI 基元统一使用 shadcn/ui 的 React Aria（`aria-nova`）基座，以使用其状态属性和进入/退出动画；不得混用 Radix UI 或 Base UI 基元。
- 页面不得新建原生 `button`、`input`、`select`、`textarea`、`table` 或手写遮罩；使用 shadcn/ui 基元及共享 `DataTable`。
- 单选下拉使用 `Select`；可搜索或多选使用 `Combobox`。业务页面不得使用原生 `select` 或已移除的 `NativeSelect`。
- 分页表格使用 `DataTable`；详情子表、树形表和临时预览可使用 `Table` 基元，不直接书写 HTML table。
- 本轮不启用 CI UI 阻断；本清单是分批收敛和后续自动检查的输入。

## 2026-07-31 React Aria 迁移例外记录

本轮获准将既有 React UI 基元迁移至 shadcn/ui `aria-nova`。重新导入后仅保留以下兼容性补丁：

- `table.tsx`：将遗留的 `TableHeader > TableRow > TableHead` 静态写法适配为 React Aria 的直接列集合，并自动标记首列为行标题。原因是现有详情表数量较多，而 React Aria 表格要求列和单元格严格匹配；该补丁仍使用 React Aria `Table` 基元。重新导入时须保留此适配，或先将全部调用方改为直接 `TableHead` 列结构。
- `alert-dialog.tsx`：受控确认框由调用方显式解析结果，操作按钮不使用 React Aria 的自动关闭 slot，避免先触发 `onOpenChange(false)` 而把确认结果解析为取消。重新导入时须保留这一受控确认行为并回归测试删除确认流程。
- `combobox.tsx`、`sidebar.tsx`、`tabs.tsx` 与 `use-mobile.ts`：仅补齐本仓库 ESLint/Fast Refresh 约束及避免 effect 内同步状态更新，不改变上游视觉、状态或 API 行为。

上游基线为 2026 年 7 月 shadcn/ui React Aria (`aria-nova`) CLI 输出；后续升级应先重新导入，再逐项复核上述例外。

## 已完成的共享基元

- [x] `Table`、`Field`、`Label`、`Checkbox`
- [x] `DataTable` 使用 `Table`、`Button` 和 `Select` 组合渲染
- [x] `ShopDialog` 使用 shadcn `Dialog`，不再手写遮罩、焦点和关闭按钮
- [x] `operation-fleet-configs` 使用 `DataTable`、`Textarea` 和共享对话框

## 优先批次

- [ ] `dashboard-corporation-structures`：多表、筛选、写操作和分页
- [ ] `system-wallet`、`system-task-manager`、`system-user`：账本表、复杂筛选与配置表单
- [x] `operation-fleets`、`operation-fleet-detail`、`operation-fleet-configs`：分页表格、详情子表、筛选、编辑表单和共享对话框均已收敛

## 后续页面债务

- 仪表盘：`dashboard-console/srp-list`、`dashboard-npc-kills`。
- 伏羲与信息：`fuxi-hall-manage`、`info-assets`、`info-contracts`、`info-esi-check`、`info-fittings`、`info-implants`、`info-npc-kills`、`info-ships`、`info-skill`、`info-tool-bookmarks`、`info-wallet`。
- 新人与行动：`newbro-captain`、`newbro-manage`、`newbro-mentor`、`newbro-select-captain`、`operation-corporation-pap`、`operation-join`、`operation-pap`。
- 商店、技能、SRP：`shop-browse`、`shop-manage`、`shop-order-manage`、`shop-wallet`、`skill-plan-completion-check`、`skill-plan-management`、`srp-apply`、`srp-manage`、`srp-prices`。
- 系统：`system-audit`、`system-auto-role`、`system-basic-config`、`system-pap-exchange`、`system-pap`、`system-user-center`、`system-webhook`。
- 工单与福利：`ticket-admin-detail`、`ticket-categories`、`ticket-create`、`ticket-detail`、`ticket-management`、`ticket-my-tickets`、`welfare-approval`、`welfare-my`、`welfare-settings`。

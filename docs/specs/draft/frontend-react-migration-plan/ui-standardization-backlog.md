---
status: draft
doc_type: draft
owner: engineering
last_reviewed: 2026-07-30
source_of_truth:
  - static-react/src/components/ui
  - static-react/src/pages
  - docs/standards/frontend-table-pages.md
---

# React shadcn/ui 规范收敛债务清单

## 规则

- 页面不得新建原生 `button`、`input`、`select`、`textarea`、`table` 或手写遮罩；使用 shadcn/ui 基元及共享 `DataTable`。
- `NativeSelect` 用于原生浏览器、移动端或性能优先的下拉；自定义交互使用 `Select`，可搜索/多选使用 `Combobox`。
- 分页表格使用 `DataTable`；详情子表、树形表和临时预览可使用 `Table` 基元，不直接书写 HTML table。
- 本轮不启用 CI UI 阻断；本清单是分批收敛和后续自动检查的输入。

## 已完成的共享基元

- [x] `Table`、`Field`、`Label`、`Checkbox`
- [x] `DataTable` 使用 `Table`、`Button` 和 `NativeSelect` 组合渲染
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

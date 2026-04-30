---
status: completed
doc_type: completed
owner: engineering
last_reviewed: 2026-04-30
completed: 2026-04-30
source_of_truth:
  - static/src/views/welfare/approval/index.vue
  - static/src/views/welfare/my/index.vue
  - docs/standards/frontend-table-pages.md
---

# 福利模块表格页面规范修复计划

## 当前状态

这份草案已更新为“规范修复已完成，结构收敛可选”：

### 已完成的修复

- `welfare/approval/index.vue` 已补上 `ElCard / ElTabs` 的 flex 高度链 CSS
- `welfare/approval/index.vue` 的 `history` tab 已使用 `size: 200` 和 `visual-variant="ledger"`
- `welfare/approval/index.vue` 的 `pending` tab 已使用 `size: 200` 和 `visual-variant="ledger"`
- `welfare/approval/index.vue` 已移除页面内冗余 `ElEmpty`
- `welfare/my/index.vue` 已补上 flex 高度链 CSS
- `welfare/my/index.vue` 已移除页面内冗余 `ElEmpty`
- `welfare/my/index.vue` 的 `applications` tab 已使用 `size: 200` 和 `visual-variant="ledger"`

### 当前剩余事项（可选优化）

- `welfare/approval/index.vue` 仍为 `ElTabPane` 包裹完整表格内容，尚未做 SRP manage 共享表格重构
- `welfare/my/index.vue` 的双数据源结构（eligible + applications）保持 CSS-only 修复路径，暂不做强制模板合并

## 问题描述

福利模块多个页面存在表格规范违反，包括滚动失效、ledger 配置不完整，以及少量页面仍保留冗余空状态或未完成的结构收敛。

## 根因分析

### 结构问题：ElTabs 嵌套在 art-table-card 内破坏 Flex 高度链

当前结构：

```
.art-full-height
  └─ ElCard.art-table-card
       └─ .el-card__body (height:100%; overflow:hidden)
            └─ ElTabs                    ← 无 flex 属性，断链
                 └─ .el-tabs__content    ← 无 flex 属性，断链
                      └─ .el-tab-pane    ← 无 flex 属性，断链
                           └─ ArtTable   ← 无法获取约束高度，自然撑高
```

`art-table-card` 的 `.el-card__body` 设置了 `height: 100%; overflow: hidden`，但 ElTabs 的内部 DOM 元素（`el-tabs__content`、`el-tab-pane`）缺少 `flex: 1; min-height: 0`，导致高度链断裂。ArtTable 的 `useTableHeight` hook 无法计算出正确的容器高度，表格随内容自然撑高而非内部滚动。

### 规范违反项

| 项 | 当前 | 规范要求 |
|----|------|----------|
| 待发放 tab 页大小 | `size: 200` | 审批记录为无限增长，应使用 ledger 规则 `size: 200` |
| 待发放 tab variant | `visual-variant="ledger"` | 应加 `visual-variant="ledger"` |
| ElEmpty 组件 | 已移除 | ArtTable 内置空状态，冗余可移除 |
| ElTabs 使用方式 | TabPane 包裹完整表格内容 | 应使用 SRP manage 模式（Tab 仅作标签，表格共享） |

## 全项目表格审计结果

### 仍有可选结构优化的页面

| # | 页面文件 | 问题类型 | 严重度 |
|---|----------|----------|--------|
| 1 | `welfare/approval/index.vue` | SRP manage 共享表格结构尚未收敛 | 中 |
| 2 | `welfare/my/index.vue` | 双数据源 tab 结构可继续整理，但非规范阻塞项 | 低 |

### 已合规无需修复的页面

| 页面文件 | 说明 |
|----------|------|
| `system/wallet/index.vue` | CSS-only 修复已到位，子模块均 ledger + size:200 |
| `shop/browse/index.vue` | CSS-only 修复已到位 |
| `shop/order-manage/index.vue` | CSS-only 修复已到位 |
| `srp/manage/index.vue` | 金标准参考，ElTabs 共享模式 |
| `newbro/manage/index.vue` | ElTabs 在页面级（不在 art-table-card 内），三个表均 ledger + size:200 |
| `newbro/select-captain/index.vue` | ElTabs 在普通 ElCard 内，ArtTable 有 ledger + size:200 |
| `info/wallet/index.vue` | 无 tabs，标准表格页 |
| `welfare/settings/index.vue` | 无 tabs，标准布局；管理型表格数量有限，size:50 合理 |
| 其余 20+ 使用 ArtTable 的页面 | 无 ElTabs 嵌套，标准 art-table-card 布局 |

---

## 修复方案

两个福利页面已完成本轮规范性修复（高度链 + ledger + 空态清理）；后续仅保留结构收敛为可选优化项。

---

### 页面 1：welfare/approval/index.vue

#### 已完成

| 项 | 状态 | 说明 |
|----|------|------|
| `history` tab 页大小 | 已完成 | 已调整为 `size: 200` |
| `history` tab variant | 已完成 | 已添加 `visual-variant="ledger"` |
| `pending` tab 页大小 | 已完成 | 已调整为 `size: 200` |
| `pending` tab variant | 已完成 | 已添加 `visual-variant="ledger"` |
| `ElEmpty` 组件 | 已完成 | 已移除页面内冗余空状态 |
| Flex 高度链 | 已完成 | 已补上 `ElCard / ElTabs` 相关 CSS |

#### 当前保留项（可选）

| 项 | 当前 | 规范要求 |
|----|------|----------|
| ElTabs 使用方式 | TabPane 包裹完整表格内容 | 应使用 SRP manage 模式（Tab 仅作标签，表格共享） |
| Flex 高度链 | 已完成 | 已补上样式，非阻塞项 |

#### 后续计划

1. 评估是否继续收敛为 SRP manage 模式
2. 若收敛完成，减少 tab 内模板层级并统一表头行为

---

### 页面 2：welfare/my/index.vue

#### 已完成

| 项 | 状态 | 说明 |
|----|------|------|
| Flex 高度链 | 已完成 | 已补上 `ElCard / ElTabs` 相关 CSS |
| `ElEmpty` | 已完成 | 页面内冗余空状态已移除 |
| 已领取 tab 页大小 | 已完成 | 已调整为 `size: 200` |
| 已领取 tab variant | 已完成 | 已添加 `visual-variant="ledger"` |

#### 仍待完成

| 项 | 当前 | 规范要求 |
|----|------|----------|
| ElTabs 使用方式 | TabPane 包裹完整表格内容（与 approval 相同结构） | 视后续收敛情况决定是否改成共享模式 |
| 已领取 tab 页大小 | `size: 200` | 领取记录为无限增长，应 `size: 200` |
| 已领取 tab variant | `visual-variant="ledger"` | 应加 `visual-variant="ledger"` |

#### 特殊考虑

此页面两个 tab 的数据结构完全不同：
- **申请福利 tab**：`eligibleRows`（本地计算，无分页，无需 ledger）—— 可视为配置型表格
- **已领取福利 tab**：`applications`（API 分页，无限增长记录）—— 需要 ledger 规则

因此不适合直接合并为单个 ArtTable（数据来源不同）。当前已采用 **CSS-only 修复模式**（参照 `system/wallet/index.vue`）并完成高度链修复；后续如继续整理模板，再评估是否存在更合适的结构收敛方式。

#### 后续计划

1. 视后续维护成本，再决定是否继续整理 tab 模板结构

---

## 实施步骤

### 已落地

1. `welfare/approval/index.vue` 已完成 flex 高度链修复
2. `welfare/approval/index.vue` 的 `history` tab 已完成 ledger 调整
3. `welfare/approval/index.vue` 的 `pending` tab 已完成 ledger 调整
4. `welfare/approval/index.vue` 已清理页面内冗余 `ElEmpty`
5. `welfare/my/index.vue` 已完成 flex 高度链修复
6. `welfare/my/index.vue` 已完成冗余 `ElEmpty` 清理
7. `welfare/my/index.vue` 的 `applications` tab 已完成 ledger 调整

### 后续可选计划

1. `welfare/approval/index.vue` 评估是否要做 SRP manage 重构
2. 两个页面在结构继续整理时，再统一复核是否还需要保留当前模板层级

## 参考文件

- 正确模式参考：`static/src/views/srp/manage/index.vue`（ElTabs + ArtTable 共享模式）
- 简洁参考：`static/src/views/info/wallet/index.vue`（无 tabs 的标准表格页）
- 表格规范：`docs/standards/frontend-table-pages.md`
- 高度计算 hook：`static/src/hooks/core/useTableHeight.ts`
- 全局布局 CSS：`static/src/assets/styles/core/app.scss`

## 约束

- 不改动 `useTableHeight`、`useLayoutHeight`、`ArtTable` 等公共组件/hook
- 不引入新的依赖或工具函数
- 保持现有 i18n key 不变
- 保持现有 API 调用不变

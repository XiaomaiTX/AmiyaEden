---
status: draft
doc_type: draft
owner: engineering
last_reviewed: 2026-05-01
source_of_truth:
  - static/src
---

# 组件替换列表（首版）

## 布局与导航（P0）

- [x] `ArtSidebarMenu` -> React Sidebar + MenuTree（基线）
- [x] `ArtHeaderBar` -> React HeaderBar（基线）
- [x] `ArtPageContent` -> React PageContent（基线；KeepAlive 替代策略待补）
- [ ] `ArtWorkTab` -> React WorkTab（标签页、固定页、批量关闭）
- [x] `ArtGlobalComponent` -> React GlobalHost（基线占位）

参考：

- `static/src/views/index/index.vue`
- `static/src/components/core/layouts/*`

## 权限与路由基础设施（P0）

- [ ] `v-auth` 指令 -> `PermissionGate` / `usePermission`
- [ ] `v-roles` 指令 -> `RoleGate` / `useRole`
- [ ] `beforeEach/afterEach` 守卫逻辑 -> React Router 中间层封装
- [ ] `RouteRegistry/MenuProcessor/RouteTransformer` -> React 路由构建器

参考：

- `static/src/directives/core/auth.ts`
- `static/src/directives/core/roles.ts`
- `static/src/router/guards/*`
- `static/src/router/core/*`

## 状态管理模块（P0-P1）

- [ ] `userStore` -> `useUserStore`（Zustand）
- [ ] `menuStore` -> `useMenuStore`
- [ ] `worktabStore` -> `useWorktabStore`
- [ ] `settingStore` -> `useSettingStore`
- [ ] `badgeStore/sys-configStore/tableStore` -> 对应 Zustand slice

参考：

- `static/src/store/modules/*.ts`

## 基础 UI 替换（P1）

- [ ] Element Plus 基础组件（Button/Input/Dialog/Form/Table/Tabs/Dropdown）
- [ ] 消息与确认框（`ElMessage`/`ElMessageBox`）
- [ ] 表格与分页能力（对齐现有 `ArtTable` 使用规范）
- [ ] 日期、上传、富文本、图表、拖拽等高耦合组件

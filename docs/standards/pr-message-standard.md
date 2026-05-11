---

# 📝 PR Message Generation

## Source of Truth

生成 PR 标题和正文时，必须先检查 `preview` 到 `main` 之间的所有 commit。

### 必须收集的信息

- 每个 commit 的标题
- 每个 commit 的描述正文
- commit 之间的主题聚合关系
- 是否存在破坏性变更、迁移、回滚或依赖升级

### 生成规则

- PR 标题应概括 `preview..main` 之间所有 commit 的共同主题，避免只取单个 commit 标题
- PR 正文应基于这些 commit message 的标题和描述进行归纳，覆盖主要变更、影响范围和风险
- 如果多个 commit 属于同一功能线，应合并为一个更高层次的叙述，而不是逐条机械罗列
- 如果 commit message 已明确说明 Added / Changed / Fixed / Breaking Changes，应在 PR 正文中保持对应分类
- 如果 commit message 中包含迁移、配置、测试或回滚说明，PR 正文中也必须体现

### 不要这样做

- 只看当前工作区 diff 就直接编写 PR 信息
- 只引用最近一个 commit 的标题
- 忽略 commit 描述正文，只拼接标题
- 在没有检查 `preview` 和 `main` 之间完整 commit 历史的情况下输出最终 PR 标题和正文

---

<!-- Template Start -->

# 🧭 Overview

<!-- 一句话说明：做了什么 + 为什么做 -->
<!-- 示例：Migrate to frontend routing to simplify backend and unify permission control -->

---

# 🧩 TL;DR

<!-- 3~5条核心变更摘要（给 Reviewer 快速扫） -->
-
-
-

---

# 🚀 Key Changes

## 1. <Module / Feature Name>

- What changed:
- Why:
- Impact:

## 2. <Module / Feature Name>

- What changed:
- Why:
- Impact:

---

# 🏗 Architecture Impact

<!-- 是否涉及架构变更（必填） -->

- [ ] No
- [ ] Yes (describe below)

### Before
<!-- 旧架构简述 -->

### After
<!-- 新架构简述 -->

### Design Rationale (WHY)
<!-- 关键设计决策，必须写 -->
-

---

# ⚠️ Breaking Changes

<!-- 如果有破坏性变更，必须结构化列出 -->

| Type | Change | Impact | Action Required |
|------|--------|--------|----------------|
| API  |        |        |                |
| DB   |        |        |                |
| Auth |        |        |                |

---

# 🔄 Migration Guide

<!-- 有破坏性变更时必填 -->

## Database

```sql
-- migration scripts
````

## Config

-

## Frontend

-

---

# 🧪 Testing

- [ ] Unit tests added/updated
- [ ] Integration tests added/updated
- [ ] Manual test completed

### Coverage Scope

-

### Edge Cases

-

---

# 📚 Documentation

- [ ] Docs updated
- [ ] No docs needed

### Updated Files

-

---

# 🎯 Impact & Risks

## Impact

-

## Risks

-

## Rollback Plan

-

---

# 📦 Deployment Notes

<!-- 是否需要特殊部署步骤 -->

- [ ] No special steps
- [ ] Requires migration
- [ ] Requires config update

## Details

---

# 🔍 Review Focus

<!-- 告诉 Reviewer 应重点看什么 -->

-
-

---

# ✅ Checklist

- [ ] Code follows architecture layering (router → handler → service → repository → model)
- [ ] API / FE / Types are consistent
- [ ] Permissions enforced on server side
- [ ] No unintended breaking changes
- [ ] Migration path is provided (if needed)
- [ ] Tests cover critical paths
- [ ] No improper use of `any` / unsafe types
- [ ] Docs updated or explicitly not needed

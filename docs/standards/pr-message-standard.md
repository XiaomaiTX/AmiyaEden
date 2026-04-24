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

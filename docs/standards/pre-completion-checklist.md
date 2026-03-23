---
status: active
doc_type: standard
owner: engineering
last_reviewed: 2026-03-23
source_of_truth:
  - AGENTS.md
  - docs/standards/testing-and-verification.md
  - docs/standards/dependency-layering.md
---

# Pre-Completion Checklist

## Purpose

This checklist consolidates all verification steps that must be completed before any change is considered done. It is the reference implementation of `AGENTS.md` "Pre-Completion Protocol".

Use this checklist at the end of every task. Skip items that genuinely don't apply to your change, but do not skip items out of convenience.

## Checklist by Change Type

### Backend-Only Change

```
[ ] Code compiles:           cd server && go build ./...
[ ] Tests pass:              cd server && go test ./...
[ ] No layer violations:     handler doesn't import repository, repository has no business logic
[ ] If bug fix:              regression test added
[ ] If contract changed:     frontend API wrapper and types updated
[ ] If route added/changed:  docs/api/route-index.md updated
[ ] If behavior changed:     feature doc updated
```

### Frontend-Only Change

```
[ ] Lint passes:             cd static && pnpm lint .
[ ] Types check:             cd static && pnpm exec vue-tsc --noEmit
[ ] If helper/hook changed:  cd static && pnpm test:unit
[ ] No direct HTTP calls in views
[ ] All new strings in both zh.json and en.json
[ ] If behavior changed:     feature doc updated
```

### Cross-Contract Change (Backend + Frontend)

```
[ ] Backend compiles:        cd server && go build ./...
[ ] Backend tests:           cd server && go test ./...
[ ] Frontend lint:           cd static && pnpm lint .
[ ] Frontend types:          cd static && pnpm exec vue-tsc --noEmit
[ ] Frontend tests:          cd static && pnpm test:unit
[ ] API wrapper updated:     static/src/api/*.ts
[ ] TS types updated:        static/src/types/api/api.d.ts
[ ] Field names match:       backend JSON tags = frontend type fields
[ ] Route index updated:     docs/api/route-index.md
[ ] Feature doc updated:     docs/features/current/*.md
```

### Permission / Role Change

```
[ ] All of "Cross-Contract Change" above
[ ] Backend route protection updated: router.go middleware
[ ] Menu seeds updated:      server/internal/model/menu.go (if applicable)
[ ] Frontend route meta:     static/src/router/modules/*.ts
[ ] Button permissions:      v-auth usage aligned
[ ] Both menu modes work:    frontend and backend mode not broken
[ ] Auth doc updated:        docs/architecture/auth-and-permissions.md
```

### Documentation-Only Change

```
[ ] Front matter updated:    status, last_reviewed
[ ] No stale cross-references introduced
[ ] Index updated:           docs/README.md or docs/features/README.md if applicable
[ ] Content matches current code (verified by reading code, not assuming)
```

### New Feature / Module

```
[ ] All of "Cross-Contract Change" above
[ ] Feature doc created:     docs/features/current/<module>.md
[ ] Feature index updated:   docs/features/README.md
[ ] Localization complete:   both zh.json and en.json
[ ] Menu seeds added:        if menu item needed
[ ] Route registered:        both backend and frontend
[ ] Follows existing module structure pattern
[ ] At least one regression test covering key behavior
```

## Test Decision Matrix

| What Changed | Minimum Test Required |
| --- | --- |
| Service business logic | Go unit test in same package |
| Repository query / join / fallback | Go query-shape or behavior test |
| Handler response shape / contract | Go handler or contract test |
| Frontend pure helper / hook | `pnpm test:unit` |
| Bug fix (any layer) | Regression test at root cause layer |
| Localization only | Build verification only |
| Documentation only | No test required |

## When You Skip a Test

If you skip a test that would normally be required, you must:

1. State which test was skipped
2. Explain why (missing infrastructure, disproportionate effort, etc.)
3. Describe where the test should be added in the future

Never silently skip. The absence of a test should be a conscious, documented decision.

## Quick Reference Commands

```bash
# Backend
cd server && go build ./...
cd server && go test ./...

# Frontend
cd static && pnpm lint .
cd static && pnpm exec vue-tsc --noEmit
cd static && pnpm test:unit

# Full stack
cd server && go build ./... && go test ./... && cd ../static && pnpm lint . && pnpm exec vue-tsc --noEmit && pnpm test:unit
```

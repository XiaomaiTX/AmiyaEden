---
status: active
doc_type: standard
owner: engineering
last_reviewed: 2026-03-23
source_of_truth:
  - docs/ai/repo-rules.md
  - server/internal
  - static/src
---

# Dependency Layering Standard

## Scope

This standard governs import direction between layers in both backend and frontend. It applies to all code changes in the repository.

## Backend Dependency Direction

```text
model → repository → service → handler → router/middleware
  ↑                                            ↑
  pkg/* (shared infrastructure)                bootstrap/
```

### Backend Rules

- `model`
  may import: standard library, GORM tags
  must not import: `repository`, `service`, `handler`, `router`, `middleware`
- `repository`
  may import: `model`, standard library, GORM, `pkg/*`
  must not import: `service`, `handler`, `router`, `middleware`
- `service`
  may import: `model`, `repository`, `pkg/*`, other services
  must not import: `handler`, `router`, `middleware`
- `handler`
  may import: `service`, `model` for request or response types, `pkg/response`
  must not import: `repository` directly
- `router`
  may import: `handler`, `middleware`, `service` for DI
  must not import: `repository` directly
- `middleware`
  may import: `model` for role constants, `pkg/*`, `service` for auth
  must not import: `handler`, `repository` directly
- `pkg/*`
  may import: standard library, external packages
  must not import: `internal/*`

### Handler Input Parsing

- Handlers own request parsing and type conversion for path params, query params, and request bodies.
- When converting request-sourced numeric IDs from `uint64` to `uint`, handlers must reject values larger than `math.MaxUint32` before the cast.
- Do not write direct `uint(strconv.ParseUint(...))` style conversions without an explicit upper-bound check.
- Keep this validation in handlers rather than pushing request parsing into services.

Preferred pattern:

```go
id, err := strconv.ParseUint(c.Param("id"), 10, 64)
if err != nil || id > math.MaxUint32 {
    response.Fail(c, response.CodeParamError, "invalid id")
    return
}

typedID := uint(id)
```

### Backend Fast Rules

- `handler` must call `service`, not `repository`
- `repository` must stay data-access only; authorization and orchestration belong in `service`
- `model` must not depend on higher internal layers

## Frontend Dependency Direction

```text
types → api → hooks/store → components → views
```

### Frontend Rules

- `types/`
  may import: nothing; pure type definitions only
  must not import: `api/`, `hooks/`, `store/`, `components/`, `views/`
- `api/`
  may import: `types/`, HTTP client utilities
  must not import: `hooks/`, `store/`, `components/`, `views/`
- `hooks/`
  may import: `types/`, `api/`, `store/`, other hooks
  must not import: `views/`, feature-specific `components/`
- `store/`
  may import: `types/`, `api/`, `hooks/`
  must not import: `views/`, `components/`
- `components/`
  may import: `types/`, `hooks/`, `store/`, other components
  must not import: `views/`, `api/` directly
- `views/`
  may import: all layers above
  must not be imported by other application layers

### Frontend Fast Rules

- `views` must not make direct HTTP calls; use `api/`
- shared contracts must live in `types/`, not under `views/`
- `components` should reach backend data through hooks or store, not direct `api/` imports

## Cross-Boundary Rules

### Backend ↔ Frontend Contract

- change order:
  1. backend request or response shape
  2. backend service logic if needed
  3. frontend `static/src/api/*.ts`
  4. frontend `static/src/types/api/api.d.ts`
  5. consuming views or components
- backend JSON field names must match frontend type definitions exactly
- do not silently rename fields across the boundary

### Infrastructure Layer (`pkg/*`)

`pkg/*` is shared infrastructure for `internal/*`.

- `pkg/*` must never import from `internal/*`
- if shared code needs types used by `internal/*`, move the types into `pkg/*` or use interfaces for DI

## Enforcement

- enforce through code review, agent verification, and `docs/standards/pre-completion-checklist.md`
- if a violation is in code you are already touching, fix it in the same change
- if a violation is unrelated, note it but do not broaden scope
- never introduce a new layering violation

## Submission Check

- no new imports from a lower layer to a higher layer
- `handler` does not import `repository`
- `repository` does not contain business logic
- `views` do not call HTTP directly
- `types/` has no imports from application layers
- `pkg/*` does not import `internal/*`

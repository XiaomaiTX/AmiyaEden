# AGENTS.md

Status: Active
Scope: entire repository
Canonical copy: this file.
Harness version: 2.1
Last updated: 2026-03-23

## 0. Harness Identity

This file is the **primary agent execution harness** for the AmiyaEden repository. It defines the environment, constraints, and feedback loops that enable autonomous agent work.

- The highest-authority document in the repository
- A machine-readable specification that agents must load before any code change
- Designed as a **navigational map**, not an encyclopedia — detail lives in `docs/`

### Context Boundaries

- **Repository-local artifacts are authoritative**: code, markdown docs, schemas, config templates, test files
- **External knowledge is not context**: Slack, Google Docs, verbal decisions not committed to this repo do not exist for agent purposes
- **Code is current state**: when code and docs conflict, treat code as current implementation and evaluate whether the doc or the code drifted
- **Sub-directory READMEs are implementation notes**, not canonical rules

### Trust Hierarchy

When multiple docs describe the same thing, authority descends:

1. This file (`AGENTS.md`)
2. `docs/standards/*.md`
3. `docs/architecture/*.md`
4. `docs/api/*.md`
5. `docs/features/current/*.md`
6. `docs/guides/*.md`
7. `docs/specs/draft/*.md`

### Golden Principles

Non-negotiable rules. Every change must satisfy all ten:

1. **Layering is law** — backend: `router → middleware → handler → service → repository → model`. Frontend: `view → api → backend`. Never skip layers.
2. **Contracts are synchronized** — changing an API endpoint requires updating backend, frontend API wrapper, TS types, UI usage, and docs in the same change.
3. **All user-facing text is localized** — no hardcoded strings in views, dialogs, tables, buttons, or toast messages. Both `zh.json` and `en.json`.
4. **Type safety over convenience** — no `any` unless genuinely unavoidable. Prefer `Api.*` types.
5. **Business logic lives in service** — not in handlers, not in repositories, not in Vue views.
6. **Permissions are enforced server-side** — frontend authorization is UX, not security.
7. **Changes are scoped** — no opportunistic refactors. Fix what you came to fix.
8. **Tests lock regressions** — bug fixes ship with regression tests. No exceptions without explicit justification.
9. **Docs update with behavior** — if you changed what the system does, update what the docs say.
10. **Patterns over invention** — follow existing repo patterns before introducing new ones.

## 1. Project Intent

`AmiyaEden` is a full-stack EVE Online operations platform with:

- Go backend under `server/`
- Vue 3 + TypeScript frontend under `static/`
- RBAC roles, menus, and button permissions
- dynamic menu / routing support
- ESI / SSO integrations
- strongly typed frontend API contracts

The active product authentication flow is EVE SSO-based. Legacy template pages may still exist in `static/src/views/auth/`, but they are not the current supported login architecture and should not be treated as product requirements unless the user explicitly asks for them.

### Quick Start

```bash
# backend
cd server && go run main.go
cd server && go test ./...
cd server && go build ./...

# frontend
cd static && pnpm install
cd static && pnpm dev
cd static && pnpm lint .
cd static && pnpm exec vue-tsc --noEmit
cd static && pnpm test:unit
```

### Documentation Entry Points

- `docs/README.md` — documentation hierarchy and source-of-truth order
- `docs/architecture/module-map.md` — directory ownership and module lookup
- `docs/guides/local-development.md` — local runtime setup
- `docs/standards/pre-completion-checklist.md` — verification protocol

## 2. Architecture Rules

Detailed layering rules, import direction, and code examples: `docs/standards/dependency-layering.md`

### Layer Responsibilities (summary)

- **handler**: transport-only — parse request, read auth context, call service, return response
- **service**: business rules — authorization beyond route guards, cross-repo orchestration, ESI/SSO, response shaping
- **repository**: database access only — no business policy, no HTTP, no Gin types
- **model**: persistence and JSON contracts — keep naming explicit, keep field names aligned

### Routing and Menu Modes

The frontend supports both `frontend` mode (static route modules) and `backend` mode (`/api/v1/menu/list`). Changes to roles, menus, and button permissions must keep aligned:

- backend route protection
- menu seeds in `server/internal/model/menu.go`
- frontend route metadata
- button permission usage (`v-auth`)

## 3. API Contract Synchronization

When changing an endpoint, update in order:

1. backend response / request shape
2. frontend API wrapper in `static/src/api`
3. shared TS types in `static/src/types/api/api.d.ts`
4. UI usage
5. `docs/api/route-index.md` if route surface or permission boundary changed

Detailed conventions: `docs/api/conventions.md`

## 4. Localization

Preferred pattern — template: `$t('namespace.key')`, script: `const { t } = useI18n()` then `t('namespace.key')`. Prefer existing namespaces. Exceptions: developer comments, internal debug logs, clearly isolated demo content.

## 5. Backend Standards

- **Responses**: use `server/pkg/response` helpers. Do not invent per-handler envelopes.
- **Authorization**: coarse in router/middleware, fine-grained in service. Never frontend-only.
- **Persistence**: query only what's needed. Use DTOs for enriched rows. Keep joins explicit.
- **External integrations**: ESI/SSO calls in service or `pkg/eve`, not in handlers or repositories.

## 6. Frontend Standards

- **Pages**: keep thin. Extract repeated UI into components, repeated logic into hooks.
- **State**: page-local stays local. Cross-page goes to Pinia. No unnecessary global store.
- **Tables/Forms**: use `ArtTable`, `ArtTableHeader`, `useTable`. Standard: `docs/standards/frontend-table-pages.md`
- **Auth pages**: EVE SSO is the supported flow. Do not extend username/password unless explicitly requested.

## 7. Testing and Verification

Detailed testing standards: `docs/standards/testing-and-verification.md`
Practical patterns: `docs/guides/testing-guide.md`
Regression test plan: `docs/guides/regression-test-plan.md`

Summary: build/lint/typecheck do not replace regression tests. Bug fixes need tests. Contract changes need at least one-sided coverage. When you skip a test, document why.

## 8. Documentation Rules

Update docs when behavior changes materially:

- `README.md` — onboarding, setup, product workflow
- `docs/architecture/*` — current architecture or runtime changes
- `docs/api/route-index.md` — route / permission surface changes
- `docs/features/current/*` — module behavior changes
- `AGENTS.md` — engineering standards

Governance standard: `docs/standards/documentation-governance.md`

## 9. Anti-Patterns

Avoid these:

- hard-coded UI strings
- handlers with business logic
- repositories with authorization logic
- views with direct HTTP calls
- duplicated API types
- silently renamed fields across backend / frontend
- unrelated refactors mixed with feature fixes
- adding new patterns when an established repo pattern already exists
- N+1 database queries
- leaking internal errors to clients
- direct ESI calls from handlers
- business logic inside Vue views
- adding global store state unnecessarily

## 10. Preferred Change Pattern

For most feature work:

1. inspect the existing backend and frontend slice
2. identify the contract boundary
3. make the backend change
4. sync frontend API / types
5. update the UI
6. add localization entries
7. run verification (`docs/standards/pre-completion-checklist.md`)
8. update docs if routes, contracts, or supported behavior changed

## 11. Pre-Completion Protocol

Before considering any change complete, run the verification protocol in `docs/standards/pre-completion-checklist.md`. Minimum:

- backend changes: `go test ./...` and `go build ./...`
- frontend changes: `pnpm lint .` and `vue-tsc --noEmit`
- cross-contract changes: validate both sides
- bug fixes: add regression test or document why not

## 12. Agent Behavioral Guardrails

### Loop Detection

If you edit the same file 3+ times for the same issue, or retry a failing command without changing approach — **stop**. Re-read the relevant docs, re-examine the error, try a different approach. If blocked, surface the blocker to the user.

### Anti-Drift Rules

- Do not introduce patterns that contradict existing repo conventions
- Do not rename, comment, or annotate code you didn't change
- Do not "improve" error handling or create abstractions beyond what was requested
- Do not create utility functions for one-time operations

### Context Refresh

Before working on an unfamiliar module:

1. Read the feature doc in `docs/features/current/`
2. Read the module map in `docs/architecture/module-map.md`
3. Inspect actual code before proposing changes

### Conflict Resolution

1. Check the trust hierarchy (Section 0)
2. Treat code as current state
3. If a doc is stale, update it — don't change code to match a stale doc
4. When genuinely uncertain, ask the user

## 13. Entropy Management

- Update docs in the same change as code when behavior changes
- When you find a stale doc, fix it or flag it
- When you add a new feature, add its feature doc before the work is complete
- Do not leave commented-out code, placeholder files, or context-free TODOs
- When removing a feature, remove its docs, routes, menu entries, and localization keys
- When 3+ files follow a pattern and 1 diverges, the diverger is wrong (unless explicitly justified)

### Harness Maintenance

When engineering standards evolve: update this file for rule changes, `docs/standards/` for detail, and increment the harness version above.

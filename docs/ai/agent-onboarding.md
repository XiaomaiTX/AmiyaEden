---
status: active
doc_type: guide
owner: engineering
last_reviewed: 2026-03-26
source_of_truth:
  - docs/ai/repo-rules.md
  - docs/README.md
---

# AI Agent Onboarding

## Purpose

Fast routing for agents. Repository rules live in `docs/ai/repo-rules.md`.

## Startup

1. Read your agent entry point (`AGENTS.md` or `CLAUDE.md`).
2. Read `docs/README.md`.
3. Identify the change type.
4. Read only the docs needed for that change before editing code.

## Change Routing

### Backend or API

Read:

1. `docs/architecture/overview.md`
2. `docs/architecture/module-map.md`
3. `docs/architecture/auth-and-permissions.md`
4. `docs/api/conventions.md`
5. `docs/api/route-index.md`
6. the relevant feature doc

### Frontend Page, Route, or Permission

Read:

1. `docs/architecture/module-map.md`
2. `docs/architecture/routing-and-menus.md`
3. `docs/standards/frontend-table-pages.md` when relevant
4. the relevant feature doc

### ESI, SSO, or CCP Sync

Read:

1. `docs/architecture/overview.md`
2. `docs/architecture/module-map.md`
3. `docs/architecture/runtime-and-startup.md`
4. `docs/features/current/auth-and-characters.md`
5. `docs/features/current/esi-refresh.md`
6. `docs/guides/adding-esi-feature.md`

Read local `README.md` files under `server/pkg/eve/esi/` only when the task is clearly in that area.

## Agent Rules

Do:

- treat `docs/ai/repo-rules.md` as the primary authority
- reason from committed repository artifacts and user-provided session context
- read surrounding module code, not only the file being edited
- update relevant docs when behavior, routes, runtime behavior, or standards change
- stop and reassess when blocked or looping

Do not:

- treat `docs/templates/` or local directory `README.md` files as repository-wide authority
- write future or planned behavior into current-state docs
- revert working behavior only to satisfy stale docs
- create a shadow documentation tree
- keep editing without progress

## Conflict Handling

- Use the authority order and code-vs-docs rule in `docs/ai/repo-rules.md`.
- If code and docs disagree, determine whether code drifted or docs became stale before changing either.

## Finish

- For debugging workflow, use `docs/guides/debugging-guide.md`.
- For completion checks, use `docs/standards/pre-completion-checklist.md`.
- For verification commands, use `docs/standards/testing-and-verification.md`.

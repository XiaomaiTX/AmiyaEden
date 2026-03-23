---
status: active
doc_type: guide
owner: engineering
last_reviewed: 2026-03-23
source_of_truth:
  - AGENTS.md
  - docs/README.md
---

# Harness Engineering Principles

## What Is Harness Engineering

Harness engineering is the design of the environment, constraints, and feedback loops that enable AI coding agents to do reliable, consistent work. It is distinct from prompt engineering (what to ask) and context engineering (what to show). Harness engineering governs the entire agent execution system.

The relationship:

```
Prompt Engineering    → "What should be asked?"     → Instruction text
Context Engineering   → "What should be shown?"     → All tokens visible at reasoning time
Harness Engineering   → "How should the whole       → Constraints, feedback, lifecycle,
                         environment be designed?"      verification, entropy management
```

## Why This Matters for AmiyaEden

This repository is designed for productive human-agent collaboration. The harness ensures that:

- Agents produce code that follows existing patterns, not novel inventions
- Architecture boundaries are maintained mechanically, not by hope
- Contract changes propagate fully across backend and frontend
- Regressions are caught by tests, not by users
- Documentation stays synchronized with implementation

## Three Pillars

### 1. Context Engineering

Everything an agent needs to reason correctly must be accessible in-context within the repository.

AmiyaEden implements this through:

- `AGENTS.md` — golden principles, architecture rules, verification protocol
- `docs/ai/agent-onboarding.md` — reading orders per change type
- `docs/architecture/module-map.md` — directory ownership and cross-reference map
- `docs/features/current/*.md` — module behavior, invariants, code file pointers
- `docs/api/route-index.md` — complete route surface with permission boundaries
- `docs/standards/*.md` — enforceable engineering standards
- Front matter on all docs — status, type, owner, last_reviewed, source_of_truth

Information that lives only in people's heads, chat threads, or external wikis is invisible to agents. If it matters, it must be committed to the repository.

### 2. Architectural Constraints

Constraints reduce the solution space. Paradoxically, this makes agents more productive by eliminating dead-end exploration.

AmiyaEden enforces:

- **Dependency direction**: `model → repository → service → handler → router` (backend), `types → api → hooks/store → components → views` (frontend). See `docs/standards/dependency-layering.md`.
- **Layer responsibilities**: handlers are transport-only, services own business logic, repositories own data access only. See `AGENTS.md` "Architecture Rules".
- **Contract synchronization**: 5-step mandatory sync when changing endpoints. See `AGENTS.md` "API Contract Synchronization".
- **Localization**: no hardcoded strings. See `AGENTS.md` "Localization".
- **Type safety**: no `any`. See `AGENTS.md` Golden Principle 4.

### 3. Entropy Management

Without active maintenance, agent-generated code drifts from conventions. AmiyaEden manages entropy through:

- **Documentation consistency**: docs update in the same change as code. See `AGENTS.md` "Documentation Rules".
- **Pattern enforcement**: new modules must match existing module structure. See `AGENTS.md` "Entropy Management".
- **Dead code removal**: no commented-out code, no placeholder files. See `AGENTS.md` "Entropy Management".
- **Harness versioning**: `AGENTS.md` carries a version header so changes to the harness itself are trackable.

## Feedback Loops

Effective harness engineering requires feedback loops that tell agents whether their work is correct.

### Build-Time Feedback

```bash
cd server && go build ./...      # Does it compile?
cd server && go test ./...       # Do existing tests pass?
cd static && pnpm lint .         # Does it follow lint rules?
cd static && pnpm exec vue-tsc --noEmit  # Do types check?
cd static && pnpm test:unit      # Do unit tests pass?
```

### Structural Feedback

- `docs/standards/dependency-layering.md` — dependency direction rules
- `AGENTS.md` "Anti-Patterns" — anti-pattern checklist
- `docs/standards/pre-completion-checklist.md` — verification protocol

### Behavioral Feedback

- Regression tests lock specific behaviors
- Feature docs describe invariants that must be preserved
- Route index documents permission boundaries that must be enforced

## Agent Anti-Patterns

Common agent failure modes to watch for:

| Failure Mode | Symptom | Mitigation |
| --- | --- | --- |
| Over-abstraction | Creating helpers/utilities for one-time operations | AGENTS.md Golden Principle 10 |
| Under-testing | "Build passes" treated as sufficient verification | `docs/standards/pre-completion-checklist.md` |
| Documentation drift | Code changed but docs left stale | AGENTS.md Golden Principle 9 |
| Layer violations | Business logic in handler or view | `docs/standards/dependency-layering.md` |
| Scope creep | "Improving" unrelated code while fixing a bug | AGENTS.md Golden Principle 7 |
| Loop behavior | Editing same file repeatedly without progress | AGENTS.md "Agent Behavioral Guardrails" |

## How to Use This Document

- **For humans**: read this to understand the philosophy behind the repository's documentation structure
- **For agents**: your primary reference is `AGENTS.md`. This document explains the reasoning behind those rules. When in doubt about a rule, check `AGENTS.md` first, then this document for context.
- **For harness maintainers**: when adding new rules to `AGENTS.md`, check whether this document needs a corresponding update to explain the reasoning

## Related Documents

- `AGENTS.md` — the authoritative harness specification
- `docs/ai/agent-onboarding.md` — reading orders and conflict resolution
- `docs/standards/dependency-layering.md` — detailed dependency rules
- `docs/standards/pre-completion-checklist.md` — verification protocol reference
- `docs/standards/testing-and-verification.md` — testing standards
- `docs/guides/testing-guide.md` — practical testing patterns
- `docs/guides/regression-test-plan.md` — phased regression test rollout

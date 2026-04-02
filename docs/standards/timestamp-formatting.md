---
status: active
doc_type: standard
owner: engineering
last_reviewed: 2026-03-28
source_of_truth:
  - static/src/utils/common/time.ts
  - static/src/utils/common/index.ts
---

# Timestamp Formatting Standard

## Scope

Applies to all user-facing timestamp and date-time displays in the frontend UI.

## Core Rules

- Reuse the shared helper from `static/src/utils/common/time.ts` instead of defining local variants in views or components.

## Allowed Exceptions

- Product-specific relative-time displays may use a separate helper only when the UI spec explicitly calls for it.
- Date-only presentation is allowed only when the underlying field is truly a calendar date and not a timestamp, and the module doc or feature spec says so.

## Checklist

- [ ] All user-facing timestamp fields use `formatTime`
- [ ] No inline `new Date(...).toLocaleString()` remains in UI timestamp renderers
- [ ] Any exception is documented in the relevant feature or architecture doc

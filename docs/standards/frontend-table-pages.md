---
status: active
doc_type: standard
owner: frontend
last_reviewed: 2026-04-01
source_of_truth:
  - static/src/hooks/core/useTable
  - static/src/components/core
---

# Frontend Table Page Standard

## Use This By Default

- Use `useTable` for paginated page-level tables.
- Use `ArtTable` for the table block.
- Keep API calls in `static/src/api`.
- Localize all user-visible text.
- Keep page components thin; extract page-sized search areas, dialogs, or repeated column setup into `modules/` when helpful.

## Ledger Rule

Treat a table as ledger-style when rows grow unbounded over time: logs, histories, transactions, records, approvals.

For ledger tables:

- set default request size to `200`
- use `ArtTable` with `visual-variant="ledger"`
- do not repeat ledger page sizes or pager layout locally unless intentionally overriding the shared preset

For bounded management/config tables:

- use normal `ArtTable` defaults
- smaller page sizes are fine

## Layout Pattern

Preferred page structure:

- search area outside the table card
- `ElCard.art-table-card` as the table container
- `ArtTableHeader` above `ArtTable`
- dialogs as siblings outside the card

If the page is mixed layout or analytics-like, you may still use this pattern for the table section itself.

## Theme-Safe Styles

- Do not hardcode light backgrounds or gradients (`#fff`, very bright RGB values) for table rows, expandable panels, or selection states; they will appear as glaring white bands in dark mode.
- Prefer Element Plus theme tokens (`var(--el-bg-color)`, `var(--el-bg-color-overlay)`, `var(--el-fill-color)`, `var(--el-border-color-*)`) for backgrounds and borders so both light and dark themes stay consistent.
- For “selected” or “expanded” cards/rows, keep the background at most one brightness step lighter than the surrounding surface; use border and shadow emphasis instead of large areas of pure white.
- When adding custom table or list visuals, manually verify the page in both light and dark modes and adjust colors using theme tokens rather than fixed hex values.

## Exceptions

Use native `ElTable` only when the table itself does not fit `ArtTable`, for example:

- detail-page subtables
- tree tables
- highly customized expandable rows
- temporary import/preview tables
- Element Plus interactions that `ArtTable` does not expose cleanly

Using native `ElTable` does not relax the other rules in this document.

## AI Checklist

Before finishing:

- Is this paginated table using `useTable` unless there is a real exception?
- If it is ledger-style, is it using `visual-variant="ledger"`?
- Are API calls outside the view?
- Are visible strings localized?
- Is the implementation following existing page patterns instead of inventing a one-off abstraction?

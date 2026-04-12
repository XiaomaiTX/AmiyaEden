---
status: active
doc_type: standard
owner: frontend
last_reviewed: 2026-04-12
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

## Overflow And Height Ownership

- Paginated table pages must declare who owns overflow. Clipped table content is a layout bug.
- If the page should grow naturally with content, avoid `art-full-height` and let the application shell own scrolling.
- If the page uses `art-full-height`, the table section must complete the height chain through every intermediate wrapper that participates in layout.
  - Typical wrappers include `ElCard` body, `ElTabs`, `el-tabs__content`, and `el-tab-pane`.
  - Use `display: flex`, `flex-direction: column`, and `min-height: 0` on those wrappers so `ArtTable` can consume the remaining height.
- Do not rely on `overflow: hidden` alone on card bodies, tab panes, or page wrappers. Hidden overflow is only valid when a descendant table region clearly owns scrolling.
- Tabbed table pages must keep each tab pane height-aware. A table inside `ElTabs` is incomplete until the tab content area can either expand naturally or hand remaining height to the table.

Recommended full-height tabbed table pattern:

- page root uses `art-full-height`
- `ElCard` body uses flex column layout with `min-height: 0`
- `ElTabs` grows with `flex: 1` and `min-height: 0`
- `el-tabs__content` uses `flex: 1`, `min-height: 0`, and only hides overflow when the descendant table region owns scrolling
- `el-tab-pane` uses `display: flex`, `flex-direction: column`, `height: 100%`, and `min-height: 0`

## Theme-Safe Styles

- Do not hardcode light backgrounds or gradients (`#fff`, very bright RGB values) for table rows, expandable panels, or selection states; they will appear as glaring white bands in dark mode.
- Prefer Element Plus theme tokens (`var(--el-bg-color)`, `var(--el-bg-color-overlay)`, `var(--el-fill-color)`, `var(--el-border-color-*)`) for backgrounds and borders so both light and dark themes stay consistent.
- For “selected” or “expanded” cards/rows, keep the background at most one brightness step lighter than the surrounding surface; use border and shadow emphasis instead of large areas of pure white.
- When adding custom table or list visuals, manually verify the page in both light and dark modes and adjust colors using theme tokens rather than fixed hex values.

## Inline Copy Rule

- For compact inline copy actions attached to a text value inside a table cell, list row, or expanded record row, reuse the shared `ArtCopyButton`.
- Do not introduce page-local copy icon buttons, duplicate clipboard success/failure toast handling, or ad hoc inline copy markup for the same interaction shape.
- If the requirement is not a compact inline button beside a single displayed value, reuse the shared clipboard hook instead of forcing the button component into a mismatched flow.
- Feature docs may describe where copy is available, but the reuse rule itself is defined here and applies repository-wide.

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
- If it adds a compact inline copy action, is it reusing `ArtCopyButton`?
- If the page uses `art-full-height` or `ElTabs`, is the overflow owner explicit and is the height chain complete?
- Are API calls outside the view?
- Are visible strings localized?
- Is the implementation following existing page patterns instead of inventing a one-off abstraction?

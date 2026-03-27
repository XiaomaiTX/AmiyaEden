---
status: active
doc_type: standard
owner: frontend
last_reviewed: 2026-03-28
source_of_truth:
  - static/src/hooks/core/useTable
  - static/src/components/core
---

# Frontend Table Page Standard

## Scope

Applies to admin pages, paginated list pages, and standard CRUD table pages in the frontend.

## Core Rules

- For standard paginated management pages, use `useTable` by default.
- Do not make direct `axios` or `fetch` calls in view components.
- Localize all user-visible text, including column titles, buttons, empty states, and validation messages.
- Handle permissions through routes, `v-auth`, and shared store or hook patterns. Do not hardcode permission logic locally in the page when an existing repository pattern applies.
- Extract reusable or page-sized search areas, edit dialogs, and column definitions into `modules/`.

## Pagination Sizing

Choose page sizes based on the nature of the data:

**Ledger / high-volume views** — transaction logs, event histories, operation records, approval
histories, or any view whose row count grows unboundedly over time:

- `pageSizes`: `[200, 500, 1000]`
- default `size`: `200`
- Pass `:pagination-options="{ pageSizes: [200, 500, 1000] }"` to `ArtTable` and set `size: 200` in `apiParams`.

**Management / config views** — users, roles, products, settings, or any view with a bounded
dataset that admins manage rather than browse:

- Use the framework default page sizes.
- A lower default size (10–50) is appropriate.

When in doubt, ask: does this table accumulate records indefinitely? If yes, use ledger sizing.

## Default Page Pattern

When creating a standard table page, use this structure unless the page has a justified exception:

- place the search area outside the card
- use `ElCard.art-table-card` as the table container
- use `ArtTableHeader` for refresh, column settings, and primary actions
- use `ArtTable` for table rendering and pagination
- use `useTable` to manage `loading`, `data`, `pagination`, and `searchParams`
- place dialogs outside `ElCard` as sibling nodes

## Recommended Structure

```text
views/<module>/<page>/
├── index.vue
└── modules/
    ├── <page>-search.vue
    ├── <page>-dialog.vue
    └── columns.ts
```

## Allowed Exceptions

You may use native ElTable instead of ArtTable only when the page does not fit the standard management-page pattern, such as:

- a read-only subtable inside a detail page
- an analytics page or dashboard with mixed data blocks
- a tree table or highly customized expandable row that ArtTable does not express well
- a third-party import page or temporary preview page

When using native ElTable, the following rules still apply:

- keep API calls in static/src/api
- localize all user-visible text
- do not hardcode permission logic inside the page

## Pre-Completion Checklist

Before considering the page complete, verify:

- Does the page actually require pagination?
- If it is a standard paginated management page, does it use useTable?
- Were the search area and dialog extracted when they are reusable or page-sized?
- Are all user-visible strings localized?
- Does the page avoid direct HTTP client creation or direct HTTP calls?

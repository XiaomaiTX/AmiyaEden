---
status: active
doc_type: feature
owner: engineering
last_reviewed: 2026-04-12
---

# Fuxi Admin Directory

## Purpose

Provides a public-facing directory of Fuxi admin staff, organized by configurable tiers.
All users (including guests) can view. Admins manage content via in-page controls.

## Data Model

- `fuxi_admin_config` — singleton: `base_font_size` (8–32 px)
- `fuxi_admin_tier` — named tiers with sort_order; ordered by sort_order ASC, id ASC
- `fuxi_admin` — individual admin entries; fields: tier_id, name, title, contact_qq, contact_discord, character_id

Deleting a tier cascade-deletes all its admins.

## Page Access

Route: `/hall-of-fame/current-manage`

- View: all users including guests (no login required)
- Edit controls: visible only to users with `admin` or `super_admin` role

## Admin Capabilities

- Add / rename / delete tiers
- Add / edit / delete admin cards
- Set character_id on a card to show EVE character portrait
- Change global `base_font_size` (applies to all cards; title and contact scale relative to this in CSS)

## API

| Method | Path | Auth | Purpose |
|--------|------|------|---------|
| GET | /api/v1/fuxi-admins | none | Public directory (config + tiers + admins) |
| GET | /api/v1/system/fuxi-admins/config | admin | Get config |
| PUT | /api/v1/system/fuxi-admins/config | admin | Update base_font_size |
| GET | /api/v1/system/fuxi-admins/tiers | admin | List tiers |
| POST | /api/v1/system/fuxi-admins/tiers | admin | Create tier |
| PUT | /api/v1/system/fuxi-admins/tiers/:id | admin | Update tier |
| DELETE | /api/v1/system/fuxi-admins/tiers/:id | admin | Delete tier (cascades) |
| POST | /api/v1/system/fuxi-admins | admin | Create admin |
| PUT | /api/v1/system/fuxi-admins/:id | admin | Update admin |
| DELETE | /api/v1/system/fuxi-admins/:id | admin | Delete admin |

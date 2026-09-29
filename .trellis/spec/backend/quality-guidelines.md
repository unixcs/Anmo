# Quality Guidelines

> Code quality standards for backend development.

---

## Overview

<!--
Document your project's quality standards here.

Questions to answer:
- What patterns are forbidden?
- What linting rules do you enforce?
- What are your testing requirements?
- What code review standards apply?
-->

(To be filled by the team)

---

## Forbidden Patterns

<!-- Patterns that should never be used and why -->

(To be filled by the team)

---

## Required Patterns

<!-- Patterns that must always be used -->

(To be filled by the team)

---

## Testing Requirements

<!-- What level of testing is expected -->

(To be filled by the team)

---

## Code Review Checklist

<!-- What reviewers should check -->

(To be filled by the team)

---

## Customer-Visible Merchant Config Pattern (settings KV)

Merchant-editable, customer-readable configuration (e.g. `shop_phone`, `home_title`,
`home_service_limit`, `home_service_ids`) must follow the existing chain in
`server/internal/modules/content/repo.go` — do not invent parallel channels:

1. **Store**: `content_system_setting` KV via `SaveSetting` (admin `PUT /admin/settings`).
   No new tables/migrations for config-like data.
2. **Validate**: add a `validateSetting` case for keys with a domain (enum / JSON shape /
   range). Unknown keys pass through as free-form KV.
3. **Expose**: add the key to `PublicSettings` with an explicit downlink decision — sanitize
   stored garbage to `""` (e.g. `homeLimitDownlink`) so clients can fall back to defaults.
4. **Client contract**: customers receive strings; every consumer must tolerate missing/invalid
   values by falling back to a documented default (see `pickHomeServices` in H5
   `apps/customer/src/core/utils/home-services.ts` and its line-mirrored weapp twin in
   `apps/weapp/utils/format.js` — keep the two implementations byte-for-byte equivalent).

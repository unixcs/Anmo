# Component Guidelines

> How components are built in this project.

---

## Overview

<!--
Document your project's component conventions here.

Questions to answer:
- What component patterns do you use?
- How are props defined?
- How do you handle composition?
- What accessibility standards apply?
-->

(To be filled by the team)

---

## Component Structure

<!-- Standard structure of a component file -->

(To be filled by the team)

---

## Props Conventions

<!-- How props should be defined and typed -->

(To be filled by the team)

---

## Styling Patterns

<!-- How styles are applied (CSS modules, styled-components, Tailwind, etc.) -->

(To be filled by the team)

---

## Accessibility

<!-- A11y requirements and patterns -->

(To be filled by the team)

---

## Common Mistakes

<!-- Component-related mistakes your team has made -->

(To be filled by the team)

---

## WeChat Mini-Program (apps/weapp) Notes

### Native `input` requires explicit height (2026-09-29)

WeChat's native `input` component does not derive its height from padding + line-height the way
H5 inputs do. Styling `.input` with vertical padding only (`padding: 12px 14px`, no `height`)
causes the placeholder/text to be vertically clipped (top half visible, bottom half missing).

**Rule**: give global `.input` an explicit `height` (currently `height: 48px; padding: 0 14px;`
in `apps/weapp/app.wxss`) and put vertical padding on `.textarea` only. When touching input
styles, verify login (phone/code) and me (nickname) pages — they share the global class.

### H5 `pagehide` fires on `window`, not `document` (2026-09-29)

Navigation-away detection (`ShopCard` map-jump fallback) registers `pagehide` to cancel a
pending-jump timer. `pagehide` is dispatched on the global object (`window`); a listener on
`document` silently never fires in some engines, so the fallback UI would show even on
successful navigation. `visibilitychange` IS dispatched on `document` (and bubbles to
`window`), so listening for both events on `window` is the reliable pairing.

**Rule**: when watching for page-unload signals, attach `pagehide` to `window`;
`visibilitychange` may be attached to either, but keep both listeners on the same target and
remove them together.

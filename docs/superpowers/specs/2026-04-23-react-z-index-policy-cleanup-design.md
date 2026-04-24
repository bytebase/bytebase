# React Z-Index Policy Cleanup Design

## Summary

Clean up the drift from PR #20012 by making React global overlays obey the semantic layer policy:

- `overlay` for normal app dialogs, sheets, menus, popovers, selects, and tooltips
- `agent` for agent-owned surfaces
- `critical` only for forced auth/session recovery

This is a strict policy cleanup. It should not redesign page UX or broadly migrate unrelated custom controls.

## Problem

The semantic layer roots and shared primitives exist, but feature code still contains global stacking decisions such as:

- full-screen `fixed ... z-50` or `z-[60]` dialogs and drawers
- manual drawer backdrops with separate `z-40`/`z-50` values
- direct `document.body` portals for feature-owned menus
- freestanding dropdowns and popovers using `absolute z-50` to compete with page surfaces

Those patterns bypass the semantic roots, so normal app overlays can accidentally cover or disable the agent, and new migrations can reintroduce one-off z-index escalation.

## Goals

- Route normal React app overlays through the `overlay` family.
- Preserve the existing `agent` and `critical` behavior from PR #20012.
- Leave component-internal paint ordering alone when it is local to a component.
- Add a guardrail that catches new feature-owned global z-index overlays.
- Keep the cleanup reviewable by avoiding unrelated page redesigns.

## Non-Goals

- Do not standardize legacy Vue or Naive UI stacking.
- Do not replace every custom dropdown if it is only local paint order and does not compete globally.
- Do not introduce new semantic layer families.
- Do not change the product behavior of affected dialogs, drawers, or menus except for stacking and modality mechanics needed to follow the policy.

## Classification

### Global Modal Surfaces

Examples include full-screen dialogs, centered result modals, side drawers, and blocking confirmation surfaces implemented with raw `fixed` and `z-*` classes.

These should be replaced with existing app primitives:

- `Dialog` for short blocking dialogs and read-only result displays
- `AlertDialog` for destructive confirmations
- `Sheet` for right-side create/edit drawers

The primitives already portal into `getLayerRoot("overlay")` and preserve access to higher-priority layers.

### Freestanding Menus And Popovers

Examples include context menus portalled directly to `document.body`, dropdown lists that escape normal layout, and popovers using `absolute z-40` or `absolute z-50` to sit above other surfaces.

Prefer existing primitives:

- `DropdownMenu` for command menus
- `Popover` for richer anchored panels
- `Combobox` for searchable/selectable lists

If a surface is too specialized for a primitive but must escape clipping, it should portal into `getLayerRoot("overlay")` and use `LAYER_SURFACE_CLASS`. It should not choose a numeric global z-index.

### Local Paint Order

The cleanup should not touch local paint-order cases unless they are clearly competing globally.

Allowed examples:

- sticky table headers
- loading masks inside a single panel
- status dot rings
- resize handles
- badge or icon layering

These can keep local `z-0`, `z-1`, `z-10`, or similar values when the stacking context is local and does not try to outrank another page surface.

## Guardrail

Add a focused frontend policy check that scans `frontend/src/react` and fails on feature-owned global z-index patterns outside approved layer infrastructure.

The check should flag:

- `fixed` elements with raw `z-*` or arbitrary `z-[...]` classes outside approved files
- high arbitrary z-index values such as `z-[60]` or `z-[999...]`
- inline `zIndex` used for cross-surface ordering
- direct `createPortal(..., document.body)` in feature code

The allowlist should include:

- `frontend/src/react/components/ui/`
- `frontend/src/react/plugins/agent/`
- `frontend/src/react/components/auth/SessionExpiredSurface.tsx`
- narrow local paint-order exceptions only when documented in the check

The check should run with the existing frontend validation path, preferably through `pnpm --dir frontend check` or a test invoked by that command, so future migrations see the policy failure before review.

## Implementation Order

1. Add the guardrail in report-only form locally to produce the current violation list.
2. Convert the highest-risk global modal surfaces to `Dialog`, `AlertDialog`, or `Sheet`.
3. Convert direct `document.body` feature portals to `getLayerRoot("overlay")` or overlay primitives.
4. Convert freestanding global dropdowns/popovers that use raw `z-40`/`z-50`.
5. Lock the guardrail so the cleaned set must stay clean.

## Testing

Run the standard frontend gates after the cleanup:

- `pnpm --dir frontend fix`
- `pnpm --dir frontend check`
- `pnpm --dir frontend type-check`
- targeted frontend tests for touched components

Add or update tests for surfaces whose behavior changes from custom markup to primitives, especially:

- Escape dismissal
- outside click behavior
- rendering into the `overlay` root
- preserving the agent layer above normal app overlays

## Risks

- Some custom dialogs may currently rely on incomplete ARIA or manual keyboard handling. Moving them to primitives can expose behavior differences, so each conversion should keep visible behavior stable while adopting primitive modality.
- A broad regex guardrail can create false positives for local paint order. Keep the rule focused on global overlay patterns and make exceptions explicit.
- Converting too many bespoke controls in one PR can obscure regressions. Keep the first PR limited to clear policy violations.

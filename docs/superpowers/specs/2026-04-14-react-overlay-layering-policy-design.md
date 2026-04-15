# React Overlay Layering Policy Design

## Summary

Define a React-only global layering policy for Bytebase that replaces ad hoc `z-index` decisions with three semantic overlay families:

- `overlay` for standard app overlays
- `agent` for the page agent window and all agent-owned overlays
- `critical` for forced session-expired / re-login UI

The policy must preserve one intentional exception to ordinary modal behavior: the agent stays visible and interactive above normal app dialogs and sheets so users can supervise ongoing DOM manipulation. The only UI allowed above the agent is forced auth recovery.

## Goals

- Replace raw global `z-index` usage in React feature code with semantic layer ownership.
- Keep standard app overlays predictable during the Vue-to-React migration.
- Ensure the `AgentWindow` and minimized launcher remain above normal app overlays.
- Ensure forced session-expired / re-login UI is the only surface allowed above the agent.
- Define interaction precedence, not just paint order, for pointer events, focus, and `Escape`.
- Create a policy that can be enforced through design-system primitives and code review.

## Non-Goals

- Do not standardize legacy Vue / Naive UI stacking in this policy.
- Do not ban all `z-index` usage; component-internal paint order still needs it.
- Do not introduce many permanent top-level families beyond `overlay`, `agent`, and `critical`.
- Do not let individual features negotiate exceptions with local `z-index` values.

## Current State

Bytebase's React overlays are partially standardized today but not yet policy-driven:

- shared React overlay primitives currently coordinate around a common `z-50`
- the `AgentWindow` shell currently uses a lower layer (`z-40`)
- some React surfaces still use one-off values such as `z-[60]` or `z-[999]`

That model is insufficient for the intended product behavior because the user wants the agent to remain above normal app dialogs while the agent manipulates the page. A single shared `z-50` family cannot express that requirement cleanly.

## Policy Decision

Bytebase React will use a small fixed set of semantic global layers:

- `base`
- `overlay`
- `agent`
- `critical`

Precedence is strict:

- `critical` > `agent` > `overlay` > `base`

No additional permanent global layer may be introduced without an explicit policy change.

## Layer Model

### `base`

`base` is normal page rendering:

- page content
- layout chrome
- sticky headers and footers
- component-local paint order

`base` does not participate in global overlay arbitration.

### `overlay`

`overlay` is the default family for normal app overlays:

- `Dialog`
- `Sheet`
- `AlertDialog`
- `Select`
- `DropdownMenu`
- `Popover`
- `Tooltip`

These surfaces remain modal relative to the app, but they are not allowed to cover or disable the agent layer.

### `agent`

`agent` is a global supervisor family:

- full `AgentWindow`
- minimized agent launcher
- dialogs, menus, selects, tooltips, and other overlays opened from agent UI

This family remains above normal app overlays and stays interactive while app overlays are open underneath it.

### `critical`

`critical` is reserved for forced auth recovery:

- session-expired UI
- re-login UI required before further interaction
- child overlays opened from that UI

`critical` is the only family allowed to cover and disable the agent.

## Ownership Rules

Layer ownership is determined by surface owner, not by whichever `z-index` value a feature wants.

Rules:

- regular product pages use app overlay primitives only
- agent UI uses agent-owned primitives only
- auth recovery flows use critical-owned primitives only
- callers may not promote a surface into a higher family with custom classes or inline styles
- child overlays inherit the family of the parent surface that opened them

Examples:

- a `Select` opened inside a settings `Dialog` remains in `overlay`
- a confirm dialog opened from the `AgentWindow` belongs to `agent`
- a tooltip opened from the re-login panel belongs to `critical`

## Portal Architecture

Each family gets a dedicated portal root in `document.body`:

- app overlay root
- agent root
- critical root

These roots make cross-family precedence deterministic and inspectable. Product code should not choose portal roots directly. Design-system primitives choose the correct root automatically.

Within one family, later-mounted portals render above earlier-mounted portals. The policy relies on mount order only within a family, not across families.

## Primitive Model

The design system should expose separate primitive families or wrappers:

- app primitives for standard product overlays
- agent primitives for agent-owned overlays
- critical primitives for auth recovery overlays

Most teams should only use the app primitive family. Agent and critical primitives are intentionally narrow and used by a small number of owned surfaces.

Numeric layer values remain internal to the design system.

## Interaction Model

The policy must govern pointer events, focus, and keyboard dismissal in addition to paint order.

### Pointer Events

- pointer events go to the highest visible family under the pointer
- app backdrops and dialogs may not intercept clicks intended for visible agent UI
- when `critical` is visible, it intercepts interaction ahead of both `agent` and `overlay`

### Focus And Modality

- app overlays remain modal within the app overlay family
- the agent exists outside the app overlay family's modal domain
- the critical family is modal relative to everything

This means a normal app dialog can still trap focus for the app, while the user can interact with the agent as a separate higher-priority surface.

### Escape Handling

`Escape` handling follows active-surface and family precedence:

- the active topmost surface gets first refusal
- if a higher family handles `Escape`, lower families do not also act on it
- underlying app overlays may only receive `Escape` if the higher family declines it

This prevents agent interaction from accidentally dismissing an underlying app dialog.

## Allowed Local `z-index`

Local `z-index` remains allowed for component-internal paint order such as:

- sticky table headers
- resize handles
- icon or badge layering
- selection highlights

This is allowed only when the value does not attempt to outrank another page surface. If a `z-index` choice is intended to beat another surface elsewhere on the page, it belongs in the layer system and should not be local feature code.

## Guardrails

React feature code must not:

- use raw global `z-*` classes to beat other surfaces
- use inline `zIndex` for cross-surface ordering
- create feature-owned top-level portal roots
- add temporary escape hatches such as `z-[9999]`
- introduce additional global families without policy review

If a surface cannot be expressed in the policy, the design-system contract is incomplete and should be fixed centrally.

## Migration Strategy

Recommended rollout:

1. Introduce semantic layer tokens and dedicated portal roots.
2. Convert existing shared app overlay primitives to the `overlay` family.
3. Convert the `AgentWindow`, minimized launcher, and agent-owned child overlays to the `agent` family.
4. Build a shared session-expired / re-login surface in the `critical` family.
5. Add review and lint guardrails against ad hoc React global `z-index` usage.
6. Migrate existing React outliers to owned primitives and family-based portals.

## Testing Strategy

Minimum behavioral coverage:

- standard app overlay over app overlay within the `overlay` family
- agent shell above app dialog and sheet
- agent child dialog/menu above app overlay
- minimized agent launcher above app overlay
- critical session-expired UI above the agent
- critical child overlays above the agent
- pointer routing to agent over an underlying app backdrop
- `Escape` precedence across `overlay`, `agent`, and `critical`
- focus behavior when app modal and agent are both present

## Risks

- If family ownership is not enforced through primitives, features will route around the policy with local numbers.
- If portal roots are not explicit, cross-family ordering will become dependent on incidental mount behavior.
- If interaction precedence is under-specified, visual order will look correct while keyboard and pointer behavior remain broken.
- If the critical family is not tightly scoped, other teams will start requesting exceptions above the agent.

## Recommendation

Adopt a strict three-family React overlay policy with dedicated portal roots, semantic ownership, and explicit interaction precedence.

This is the smallest model that matches Bytebase's intended product behavior:

- standard app overlays remain coherent
- the agent stays on top while supervising page automation
- forced re-auth is the sole exception that can cover and disable the agent

Anything looser will regress into another round of ad hoc `z-index` escalation during migration.

This file provides additional guidance to AI coding assistants working under `./frontend/`.

## Inheritance

- Follow the repository-wide guidance in `../AGENTS.md`.
- Treat this file as frontend-specific additions, not a replacement for the root instructions.

## React

- All product UI code is React. Use React, Base UI, Tailwind CSS v4, and the shadcn-style component patterns below. The `pev2` adapter under `src/apps/explain-visualizer/` is the only Vue runtime exception.

## Localization

- All user-facing UI text belongs in `src/locales/` and is consumed through `useTranslation()` from `react-i18next`.
- Locale files must not contain empty JSON objects; remove them when encountered.

## UX contract

- Follow [`../docs/agents/frontend-ux.md`](../docs/agents/frontend-ux.md) as the canonical source for foundations, page and sheet forms, sticky actions, dialogs, wizards, tables, pagination, selection, and responsive behavior.
- New shared UI and every UI element directly modified by a change MUST follow the guideline. Do not copy an adjacent legacy pattern merely because it is already present.
- Existing feature violations in `scripts/ui-guideline-legacy-debt.json` are temporary incremental exceptions, not permission for new debt. A change may leave unrelated fingerprints in place, but MUST NOT add, mutate, or increase them.
- Repeated measurements belong in `src/components/ui/styles.stylex.ts`; semantic colors belong in `src/assets/css/tailwind.css`; variants belong in the shared primitive's CVA definition.
- Run `pnpm --dir frontend check` or `node frontend/scripts/check-ui-guideline.mjs` after UI work. When a change removes legacy debt, run `node frontend/scripts/check-ui-guideline.mjs --write-baseline`; the updater accepts reductions only.

## Source ownership

Use route ownership as the primary organization axis. Do not add a generic `features/` directory.

| Location | Owns |
| --- | --- |
| `src/app/` | Bootstrap, root providers, layouts, and React Router infrastructure |
| `src/app/router/` | Route definitions, navigation helpers, guards, and route handles |
| `src/routes/auth/` | Authentication route modules |
| `src/routes/workspace/` | Workspace-level route modules, including settings and resource lists |
| `src/routes/project/` | Project-scoped route modules; keep page-only components, hooks, and tests beside their route |
| `src/modules/` | Cohesive application subsystems reused by routes, such as SQL Editor, agent, AI, CEL, and schema tooling |
| `src/components/ui/` | Base UI/shadcn-style primitives |
| `src/components/` | Product components genuinely reused by multiple routes or modules |
| `src/stores/` | Cross-route Zustand state and compatibility helpers; route-local state stays with its owner |
| `src/api/` | ConnectRPC clients, transports, middleware, and API adapters |
| `src/hooks/` and `src/lib/` | Cross-cutting hooks and framework-neutral helpers only |
| `src/types/` and `src/utils/` | Existing cross-module contracts and compatibility utilities; prefer owner-local code for new work |
| `src/types/proto-es/` | Generated protobuf output; do not edit manually |
| `src/apps/explain-visualizer/` | Isolated secondary entrypoint; the only source subtree allowed to use Vue through `pev2` |

### Placement and dependency rules

- Start from the route in `src/app/router/routes/`, then open the matching subtree under `src/routes/`.
- Keep route-only code beside its route. Promote code to `components`, `hooks`, or `lib` only after it has multiple independent consumers.
- Colocate new types and helpers with their route or module. Do not grow the broad `types` and `utils` barrels without a cross-module need.
- Put a large reusable workflow in `src/modules/<name>/`; do not spread one subsystem across `components`, `stores`, and a migration-era `views` directory.
- Shared code and modules must not import from `src/routes/`. Move the shared implementation to its actual owner instead.
- Prefer direct owner imports such as `@/modules/sql-editor/store` over broad barrels when the owner is known.
- Historical migration plans under `docs/` describe old paths and are not current architecture guidance.
- `CLAUDE.md` files only import their adjacent `AGENTS.md`; update `AGENTS.md` as the source of truth.
- `pnpm --dir frontend check` runs the structure guard. Do not bypass failures by recreating retired framework, view, or singular-store namespaces.

## shadcn Skill

When a `shadcn` skill is available, use it before writing or modifying React UI. If it is unavailable, inspect `src/components/ui/` and follow the component rules below; do not block ordinary frontend work on a missing optional skill.

## shadcn Component Guidelines

React UI components live in `src/components/ui/` and follow shadcn-style patterns: Base UI primitives wrapped with `cva` variants and `cn()` for class merging.

### Rules

- **Use existing UI components first** — check `src/components/ui/` before writing custom markup. Use `Badge` not styled spans, `Alert` not custom callout divs, `Separator` not `<hr>` or `border-t` divs
- **Use `truncate` shorthand** — not `overflow-hidden text-ellipsis whitespace-nowrap`
- **Use `cn()` for conditional classes** — import from `@/lib/utils`, don't write manual template literal ternaries
- **Overlay layering policy** — React overlays use three semantic families: `overlay`, `agent`, and `critical`.
  - Standard app overlays mount into `overlay`.
  - The shared primitives in `src/components/ui/` are the `overlay` entry points; they are not for agent-owned or critical surfaces.
  - `AgentWindow`, the minimized launcher, and other agent-owned overlays mount into `agent` and stay above normal app overlays.
  - Agent-owned surfaces should use the wrappers in `src/modules/agent/components/ui/` or other agent-owned code that mounts into `getLayerRoot("agent")`.
  - Forced session-expired / re-login UI mounts into `critical` and is the only layer allowed above and disabling the agent.
  - `critical` is reserved for auth/session recovery surfaces such as `SessionExpiredSurface`; do not introduce new feature-level critical overlays without an explicit policy change.
  - Each family has a dedicated portal root; use `getLayerRoot(<family>)` to choose the family root, and use `LAYER_SURFACE_CLASS` / `LAYER_BACKDROP_CLASS` from `src/components/ui/layer.ts` for shared intra-family stacking where appropriate.
  - Children inherit the owning family. If a parent mounts into `agent` or `critical`, its descendants must not remount into a lower family.
  - Raw global `z-index` values are forbidden in React feature code for cross-surface stacking. Local component-internal `z-index` remains allowed when it only affects internal composition.
  - Consumers of overlay primitives must not set their own global `z-index`.
  - Dropdown-bearing controls inside `Sheet`, `Dialog`, or any clipped/stacked container must render their popup through the shared layer. For shared controls that expose a `portal` prop, such as `Combobox` and select wrappers built on it, pass `portal` instead of raising a local dropdown with raw `z-index`.
  - Menus, popovers, dropdowns, and custom floating panels should use shared `DropdownMenu`, `Popover`, `Combobox`, `Select`, `Dialog`, or `Sheet` primitives rather than ad hoc `absolute top-full z-*` markup.
  - Do not portal feature UI directly to `document.body` or a `document.body` alias. Use the shared overlay primitives, or explicitly mount into the correct semantic root with `getLayerRoot(<family>)`.
  - Do not hide raw global overlay classes in constants, imported helpers, `cn()` inputs, or interpolated template literals. A value like `fixed inset-0 z-50` is still forbidden even when it is not written directly in `className`.
  - When adding or changing React overlays, run `pnpm --dir frontend check` or `node frontend/scripts/check-react-layering.mjs` before handing off. The scanner is intended to catch raw high-z overlays, forbidden body portals, and policy drift in feature code.
  - The scanner is a guardrail, not proof of policy compliance. It intentionally avoids full static analysis, so imported, dynamic, shadowed, or complex expressions may be unresolved; passing the check does not permit raw global z-index overlays or body portals.

### Component Patterns

- **CVA for variants** — use `class-variance-authority` when a component has visual variants (see `button.tsx`, `badge.tsx`, `alert.tsx` for examples)
- **Wrap Base UI primitives** — import from `@base-ui/react/*`, wrap with styled components, export compound parts (Root, Trigger, Content, etc.)
- **Icons** — use `lucide-react`. No sizing classes on icons inside UI components that handle sizing. Prefer `size-*` shorthand when sizing icons manually
- **Dialog/Sheet must have a Title** — required for accessibility. Use `className="sr-only"` if visually hidden
- **Avatar must have a fallback** — for when the image fails to load
- **TabsTrigger must be inside TabsList** — never render triggers directly in Tabs

This file provides additional guidance to AI coding assistants working under `./frontend/`.

## Inheritance

- Follow the repository-wide guidance in `../AGENTS.md`.
- Treat this file as frontend-specific additions, not a replacement for the root instructions.

## React Migration

- For Vue-to-React migrations in `frontend/`, read and follow `../docs/plans/2026-04-08-react-migration-playbook.md`.
- Use that playbook to decide migration order, safe Vue deletions, state/data boundaries, testing expectations, and CI pitfalls.

## Frontend Reminder

- All new UI code should be written in React unless an existing Vue surface must be preserved temporarily for compatibility.
- Do not delete Vue counterparts until you verify they have no remaining live callers.

## shadcn Skill

When working on React UI, invoke the `shadcn` skill before writing or modifying components. The skill provides component selection guidance, critical rules, and best practices. Always check the skill when unsure which component to use.

## shadcn Component Guidelines

React UI components live in `src/react/components/ui/` and follow shadcn-style patterns: Base UI primitives wrapped with `cva` variants and `cn()` for class merging.

### Rules

- **Use existing UI components first** — check `src/react/components/ui/` before writing custom markup. Use `Badge` not styled spans, `Alert` not custom callout divs, `Separator` not `<hr>` or `border-t` divs
- **Use semantic color tokens** — `bg-accent`, `text-control`, `border-control-border`, `bg-error`, `text-warning`, etc. Never use raw color values like `bg-blue-500`, `text-gray-600`, or `bg-red-500`. Semantic tokens are defined as CSS custom properties in `src/assets/css/tailwind.css`
- **Use `gap-*` not `space-x-*` / `space-y-*`** — always use `flex gap-*` or `grid gap-*` for spacing between children
- **Use `size-*` for equal dimensions** — `size-4` not `w-4 h-4`
- **Use `truncate` shorthand** — not `overflow-hidden text-ellipsis whitespace-nowrap`
- **Use `cn()` for conditional classes** — import from `@/react/lib/utils`, don't write manual template literal ternaries
- **No manual `dark:` overrides** — use semantic tokens that handle theming
- **Overlay layering policy** — React overlays use three semantic families: `overlay`, `agent`, and `critical`.
  - Standard app overlays mount into `overlay`.
  - The shared primitives in `src/react/components/ui/` are the `overlay` entry points; they are not for agent-owned or critical surfaces.
  - `AgentWindow`, the minimized launcher, and other agent-owned overlays mount into `agent` and stay above normal app overlays.
  - Agent-owned surfaces should use the wrappers in `src/react/plugins/agent/components/ui/` or other agent-owned code that mounts into `getLayerRoot("agent")`.
  - Forced session-expired / re-login UI mounts into `critical` and is the only layer allowed above and disabling the agent.
  - `critical` is reserved for auth/session recovery surfaces such as `SessionExpiredSurface`; do not introduce new feature-level critical overlays without an explicit policy change.
  - Each family has a dedicated portal root; use `getLayerRoot(<family>)` to choose the family root, and use `LAYER_SURFACE_CLASS` / `LAYER_BACKDROP_CLASS` from `src/react/components/ui/layer.ts` for shared intra-family stacking where appropriate.
  - Children inherit the owning family. If a parent mounts into `agent` or `critical`, its descendants must not remount into a lower family.
  - Raw global `z-index` values are forbidden in React feature code for cross-surface stacking. Local component-internal `z-index` remains allowed when it only affects internal composition.
  - Consumers of overlay primitives must not set their own global `z-index`.
  - Dropdown-bearing controls inside `Sheet`, `Dialog`, or any clipped/stacked container must render their popup through the shared layer. For shared controls that expose a `portal` prop, such as `Combobox` and select wrappers built on it, pass `portal` instead of raising a local dropdown with raw `z-index`.
  - Menus, popovers, dropdowns, and custom floating panels should use shared `DropdownMenu`, `Popover`, `Combobox`, `Select`, `Dialog`, or `Sheet` primitives rather than ad hoc `absolute top-full z-*` markup.
  - Do not portal feature UI directly to `document.body` or a `document.body` alias. Use the shared overlay primitives, or explicitly mount into the correct semantic root with `getLayerRoot(<family>)`.
  - Do not hide raw global overlay classes in constants, imported helpers, `cn()` inputs, or interpolated template literals. A value like `fixed inset-0 z-50` is still forbidden even when it is not written directly in `className`.
  - When adding or changing React overlays, run `pnpm --dir frontend check` or `node frontend/scripts/check-react-layering.mjs` before handing off. The scanner is intended to catch raw high-z overlays, forbidden body portals, and policy drift in feature code.
  - The scanner is a guardrail, not proof of policy compliance. It intentionally avoids full static analysis, so imported, dynamic, shadowed, or complex expressions may be unresolved; passing the check does not permit raw global z-index overlays or body portals.
- **Dialog vs Sheet** — use `<Sheet>` (right-side drawer, in `src/react/components/ui/sheet.tsx`) for **creating or editing a resource**. Use `<Dialog>` for **confirmations, single-field prompts, critical interrupts, and read-only result displays**. The dividing line is whether the user is filling out a form with multiple fields — drawers keep the parent list/table visible behind a scrim and scale to multi-section forms, while dialogs are for short blocking interactions. `AlertDialog` is the right pick for destructive confirms that need an explicit acknowledgment.
- **Sheet width tiers** — `<SheetContent>` accepts a `width` variant. Pick the tier that matches the form complexity; don't inline ad-hoc widths. Add a new tier to `sheet.tsx` only if a genuinely new size is needed.
  - `narrow` (384px) — single-field pickers, short 2–3 field forms, environment/project selection, read-only display sheets
  - `standard` (704px, default) — 3–6 field forms, permission transfer lists, typical create/edit resources (role, user, group, service account, workload identity, request role)
  - `wide` (832px) — forms with CEL expression builders, nested tables, multi-tab layouts, multi-step wizards (custom approval rule, data export wizard)
- **Edit sheets must populate from props reliably** — when a Sheet is always-mounted via `<Sheet open={open}>` (the standard pattern), `useState` initializers only run on first mount, which means switching the entity being edited (e.g. clicking Edit on a different row) won't repopulate fields. Use the **outer wrapper + inner form + stable-entity ref + key** pattern. The ref freezes the last-open entity so the inner form stays visually stable through the Sheet's close animation (which is ~200ms), while the `key` forces a fresh mount when a new entity is opened. Example from `CreateUserSheet`:
  ```tsx
  function CreateUserSheet(props: Props) {
    const { open, user, onClose } = props;
    // Freeze the entity while open=false so the inner form stays visually
    // stable during the Sheet's close animation. Base UI's Dialog.Portal
    // unmounts after the animation, at which point the form unmounts with it.
    const openEntityRef = useRef(user);
    if (open) {
      openEntityRef.current = user;
    }
    const stableUser = openEntityRef.current;
    return (
      <Sheet open={open} onOpenChange={(next) => !next && onClose()}>
        <SheetContent width="standard">
          <UserForm
            key={stableUser?.name ?? "new"}
            user={stableUser}
            onClose={props.onClose}
            onCreated={props.onCreated}
            onUpdated={props.onUpdated}
          />
        </SheetContent>
      </Sheet>
    );
  }
  function UserForm({ user, ... }: InnerProps) {
    // useState initializers read directly from `user` — always fresh
    // because the inner component mounts fresh on every open.
    const [title, setTitle] = useState(user?.title ?? "");
    // ...
  }
  ```
  Do **not** guard the inner form with `{open && ...}` — that would unmount it at the start of the close animation, leaving a blank sheet sliding off-screen for ~200ms. Base UI's Dialog.Portal already handles the mount/unmount lifecycle around the animation.
- **Edit sheets must disable Update until dirty** — capture initial values at mount (inside the inner form component, so they reflect the just-mounted entity prop) and compute `isDirty` via `useMemo` comparing current state to captured initials. Gate the Update button on `isFormValid && isDirty`. Create mode is always "dirty" so Create is enabled as soon as required fields are valid.
- **Fetch the full entity before opening an edit sheet** — list APIs often return partial objects. Synchronous cache lookups like `store.getX(id)` can return a stub with only name/email/title fields, leaving nested fields (e.g. `workloadIdentityConfig.subjectPattern`) undefined. Use the async `getOrFetchX` form in row-click handlers so the Sheet receives a fully-hydrated entity — otherwise parsed/derived fields will be empty on first edit.

### Component Patterns

- **CVA for variants** — use `class-variance-authority` when a component has visual variants (see `button.tsx`, `badge.tsx`, `alert.tsx` for examples)
- **Wrap Base UI primitives** — import from `@base-ui/react/*`, wrap with styled components, export compound parts (Root, Trigger, Content, etc.)
- **Icons** — use `lucide-react`. No sizing classes on icons inside UI components that handle sizing. Prefer `size-*` shorthand when sizing icons manually
- **Dialog/Sheet must have a Title** — required for accessibility. Use `className="sr-only"` if visually hidden
- **Avatar must have a fallback** — for when the image fails to load
- **TabsTrigger must be inside TabsList** — never render triggers directly in Tabs

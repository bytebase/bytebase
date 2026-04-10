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
- **No manual `z-index` on overlay *consumers*** — callers of `Dialog`, `Sheet`, `Popover`, `Select`, `Tooltip` must not set their own `z-index`. The primitives in `src/react/components/ui/` already coordinate stacking (all overlays use `z-50`; within that layer, later-mounted portals win by DOM order). Do **not** strip `z-50` from `select.tsx`, `tooltip.tsx`, `dialog.tsx`, or `alert-dialog.tsx` — removing it makes Select/Tooltip render behind Dialog (BYT-9226, PR #19824)
- **Dialog vs Sheet** — use `<Sheet>` (right-side drawer, in `src/react/components/ui/sheet.tsx`) for **creating or editing a resource**. Use `<Dialog>` for **confirmations, single-field prompts, critical interrupts, and read-only result displays**. The dividing line is whether the user is filling out a form with multiple fields — drawers keep the parent list/table visible behind a scrim and scale to multi-section forms, while dialogs are for short blocking interactions. `AlertDialog` is the right pick for destructive confirms that need an explicit acknowledgment.
- **Sheet width tiers** — `<SheetContent>` accepts a `width` variant. Pick the tier that matches the form complexity; don't inline ad-hoc widths. Add a new tier to `sheet.tsx` only if a genuinely new size is needed.
  - `narrow` (384px) — single-field pickers, short 2–3 field forms, environment/project selection, read-only display sheets
  - `standard` (704px, default) — 3–6 field forms, permission transfer lists, typical create/edit resources (role, user, group, service account, workload identity, request role)
  - `wide` (832px) — forms with CEL expression builders, nested tables, multi-tab layouts, multi-step wizards (custom approval rule, data export wizard)
- **Edit sheets must populate from props reliably** — when a Sheet is always-mounted via `<Sheet open={open}>` (the standard pattern), `useState` initializers only run on first mount, which means switching the entity being edited (e.g. clicking Edit on a different row) won't repopulate fields. Use the **outer wrapper + inner form + key** pattern to force a clean remount on each open. Example from `CreateUserSheet`:
  ```tsx
  function CreateUserSheet(props: Props) {
    const { open, user, onClose } = props;
    return (
      <Sheet open={open} onOpenChange={(next) => !next && onClose()}>
        <SheetContent width="standard">
          {open && <UserForm key={user?.name ?? "new"} {...props} />}
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
  The `{open && ...}` guard ensures the inner form unmounts on close, and `key={entity?.name ?? "new"}` forces a remount when the edited entity changes. No reset `useEffect` needed.
- **Edit sheets must disable Update until dirty** — capture initial values at mount (inside the inner form component, so they reflect the just-mounted entity prop) and compute `isDirty` via `useMemo` comparing current state to captured initials. Gate the Update button on `isFormValid && isDirty`. Create mode is always "dirty" so Create is enabled as soon as required fields are valid.
- **Fetch the full entity before opening an edit sheet** — list APIs often return partial objects. Synchronous cache lookups like `store.getX(id)` can return a stub with only name/email/title fields, leaving nested fields (e.g. `workloadIdentityConfig.subjectPattern`) undefined. Use the async `getOrFetchX` form in row-click handlers so the Sheet receives a fully-hydrated entity — otherwise parsed/derived fields will be empty on first edit.

### Component Patterns

- **CVA for variants** — use `class-variance-authority` when a component has visual variants (see `button.tsx`, `badge.tsx`, `alert.tsx` for examples)
- **Wrap Base UI primitives** — import from `@base-ui/react/*`, wrap with styled components, export compound parts (Root, Trigger, Content, etc.)
- **Icons** — use `lucide-react`. No sizing classes on icons inside UI components that handle sizing. Prefer `size-*` shorthand when sizing icons manually
- **Dialog/Sheet must have a Title** — required for accessibility. Use `className="sr-only"` if visually hidden
- **Avatar must have a fallback** — for when the image fails to load
- **TabsTrigger must be inside TabsList** — never render triggers directly in Tabs

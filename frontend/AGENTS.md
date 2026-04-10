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
- **No manual `z-index` on overlays** — Dialog, Sheet, Popover handle their own stacking

### Component Patterns

- **CVA for variants** — use `class-variance-authority` when a component has visual variants (see `button.tsx`, `badge.tsx`, `alert.tsx` for examples)
- **Wrap Base UI primitives** — import from `@base-ui/react/*`, wrap with styled components, export compound parts (Root, Trigger, Content, etc.)
- **Icons** — use `lucide-react`. No sizing classes on icons inside UI components that handle sizing. Prefer `size-*` shorthand when sizing icons manually
- **Dialog/Sheet must have a Title** — required for accessibility. Use `className="sr-only"` if visually hidden
- **Avatar must have a fallback** — for when the image fails to load
- **TabsTrigger must be inside TabsList** — never render triggers directly in Tabs

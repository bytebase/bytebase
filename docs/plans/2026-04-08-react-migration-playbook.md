# React Migration Playbook

This playbook captures the repo-specific patterns that worked well while migrating `RevisionDetail` from Vue to React.

## Scope

Use this when migrating an existing frontend surface from Vue to React in `frontend/`.

The goal is not "remove all Vue in one PR". The goal is to move a bounded surface to React without breaking still-live Vue callers.

## Migration Order

1. Identify the route or feature boundary you want to migrate.
2. List every shared Vue dependency that boundary uses.
3. Decide which shared dependencies must move first because the target surface cannot be React without them.
4. Migrate only those required shared pieces.
5. Cut the route or feature over to React.
6. Delete only the Vue files that no longer have callers.

Do not rewrite an entire subsystem unless the target surface actually depends on that rewrite.

## Shared Component Rule

Migrate shared components first only when they are on the critical path for the target surface.

Good candidates:
- A Vue-only detail panel directly rendered by the route being migrated
- A shared viewer component that the new React surface must embed
- A small imperative utility that can become a stable React integration seam

Bad candidates:
- Large neighboring Vue subsystems with unrelated callers
- Shared components that still power multiple unchanged Vue screens
- Infrastructure rewrites justified only by "cleanup while we're here"

## Deletion Rule

Before deleting a Vue counterpart, check for live callers with `rg`.

Safe to delete:
- The route page that is no longer mounted
- A Vue component whose exports and imports have been fully removed

Not safe to delete:
- Shared Vue components still imported by other Vue screens
- Vue infrastructure that new React wrappers still reuse internally

If a React wrapper still depends on an existing TS utility from the Vue side, that is fine. Do not force a rewrite just to remove the old directory name.

## State and Data Guidance

Default to the existing store and Connect stack first.

Prefer:
- Pinia stores accessed directly from React
- `useVueState(getter)` when React needs to subscribe to Vue reactive state
- Existing router and utility modules

Only introduce `zustand` or `tanstack/query` when there is a concrete problem, such as:
- React-owned shared UI state that is awkward through the Vue store boundary
- Request caching, invalidation, or dedup that is materially cleaner with TanStack Query

Do not add either library by default just because the target is React.

## Route Migration Pattern

When migrating a route:

1. Keep route parsing at the router boundary.
2. Mount a React page through `ReactPageMount.vue`.
3. Pass normalized resource-style props into the React page.
4. Keep the React page self-contained: stores, router calls, derived values, and loading state should live there unless there is an obvious reusable boundary.

This keeps the React page focused on rendering and avoids scattering route parsing across components.

## Monaco and Other Imperative Libraries

For imperative libraries, prefer one stable integration seam over direct imports in React components.

Preferred pattern:
- Put the imperative entry points in a shared helper module
- Have React wrappers call that helper
- Mock the helper in tests

Avoid:
- Direct dynamic imports of third-party modules inside React effects when a shared helper can own that work

This mattered for Monaco. A direct async `import("monaco-editor")` inside a React effect passed locally but caused a CI-only failure under Node 24 because the late import resolved into a CSS-loading path outside the test's awaited boundary.

## Testing Guidance

For migrated React wrappers:

- Test the wrapper's contract, not the third-party library internals
- Mock repo-owned integration seams, not vendor modules when possible
- Expect CI to be stricter about async timing than local runs

Minimum verification for a migration PR:
- `pnpm --dir frontend fix`
- `pnpm --dir frontend check`
- `pnpm --dir frontend type-check`
- `pnpm --dir frontend test`

If you add a new shared React wrapper, add focused tests for it before relying only on page-level verification.

## i18n and Config

Treat these as part of the migration, not cleanup after it:

- Add all new user-facing strings to the React locale files under `frontend/src/react/locales/`
- Make any required `tsconfig` updates for new React barrels or entry points
- Run the normal frontend verification so `check-react-i18n` and type-checking catch migration drift early

## Practical Checklist

- Is the target boundary clearly defined?
- Have all required shared Vue dependencies been identified?
- Are you migrating only the dependencies the target truly needs?
- Is the React surface using existing stores/utilities unless a new state layer is justified?
- Are direct third-party async imports hidden behind a shared helper?
- Did you verify which Vue files are still live before deleting them?
- Did you add locale entries and any required TS config updates?
- Did you run `fix`, `check`, `type-check`, and `test`?

## RevisionDetail Takeaway

`RevisionDetail` was a good pattern for future migrations:

- Migrate the route-specific Vue files and the small set of shared components the route actually needed
- Keep the existing store/connect foundation
- Add React-native replacements for the embedded shared pieces
- Delete only the Vue files whose callers were gone
- Leave still-live Vue shared components alone until their remaining callers are migrated

---

## UX Patterns

These patterns must be followed in all React UI to maintain visual consistency.

### Border Radius

Use only these Tailwind classes:

- **`rounded-xs`** (2px) — inputs, buttons, tags, badges, alerts, checkboxes, small inline elements
- **`rounded-sm`** (4px) — modals, dialogs, dropdowns, popovers, tooltips, bordered card/section containers, list containers with overflow
- **`rounded-full`** — pills, avatars

**Never use**: `rounded`, `rounded-md`, `rounded-lg`, `rounded-xl`, or any other radius value.

### Input Component

Always use `<Input>` (`@/react/components/ui/input`) instead of raw `<input>` for `type="text"`, `type="number"`, `type="password"`, `type="email"`, `type="date"`, etc.

The only exception is when an input is **intentionally borderless** inside a custom wrapper (e.g., a search input inside a combo trigger, or an email prefix input inside a bordered div with a suffix).

```tsx
import { Input } from "@/react/components/ui/input";

<Input type="number" value={count} onChange={...} className="w-24" />
```

### SearchInput (`@/react/components/ui/search-input`)

Use `SearchInput` for all filter/search inputs. Do NOT build inline search inputs with `<Input>` + `<Search>` icon.

```tsx
import { SearchInput } from "@/react/components/ui/search-input";

<SearchInput
  placeholder={t("common.filter-by-name")}
  value={searchText}
  onChange={(e) => setSearchText(e.target.value)}
/>
```

- Icon is always on the left
- Default height `h-9`, default placeholder `t("common.type-to-search")`
- Default wrapper `flex-1` (full width); override with `wrapperClassName`
- Override input styles with `className` (e.g., `className="h-7"` inside dropdowns)

### PagedTableFooter (`@/react/hooks/usePagedData`)

Use `PagedTableFooter` for all paginated tables. Do NOT build inline pagination.

```tsx
import { PagedTableFooter } from "@/react/hooks/usePagedData";

<PagedTableFooter
  pageSize={pageSize}
  pageSizeOptions={pageSizeOptions}
  onPageSizeChange={setPageSize}
  hasMore={hasMore}
  isFetchingMore={isFetchingMore}
  onLoadMore={loadMore}
/>
```

### Combobox (`@/react/components/ui/combobox`)

Generic select supporting single-select, multi-select, grouped options, search, and portal rendering.

```tsx
import { Combobox } from "@/react/components/ui/combobox";

// Single-select
<Combobox value={selected} onChange={setSelected} options={options} />

// Multi-select
<Combobox multiple value={list} onChange={setList} options={options} />

// Inside modals
<Combobox value={val} onChange={setVal} options={options} portal />
```

### RoleSelect (`@/react/components/RoleSelect`)

Built on `Combobox`. Use for all role selection.

```tsx
import { RoleSelect } from "@/react/components/RoleSelect";

<RoleSelect value={roles} onChange={setRoles} />                    // multi
<RoleSelect value={[role]} onChange={(r) => set(r[0])} multiple={false} />  // single
<RoleSelect value={roles} onChange={setRoles} scope="project" />    // project only
```

### AccountMultiSelect (`@/react/components/AccountMultiSelect`)

Multi-select for users, groups, and special accounts with server-side search.

### UserAvatar (`@/react/components/UserAvatar`)

Renders a user avatar with color-coded initials.

### Permission & Feature Guards

| Component | Purpose | Usage |
|-----------|---------|-------|
| `FeatureBadge` | Sparkles icon + tooltip for plan-gated features | Next to labels; inside buttons with `clickable={false} className="mr-1 text-white inline-flex"` |
| `FeatureAttention` | Full-width banner for plan requirements | Top of page/section |
| `PermissionGuard` | Tooltip wrapper for missing permissions | Inline (buttons): default; Block (sections): `display="block"` |
| `ComponentPermissionGuard` | Error alert for gated components | Replaces content with permission error |

### Input Heights

All inputs and buttons in the same row: **`h-9`**.

### Focus Styles

- Custom selectors/dropdowns: `border-accent` for active state, NOT `ring-2 ring-accent`
- Inputs inside custom selectors: `outline-hidden border-none shadow-none`

### Scrollbars

Global thin scrollbars via CSS in `tailwind.css`. No per-component styling needed.

### Dropdowns in Modals

Use `createPortal` or pass `portal` prop to `Combobox` when inside `overflow: hidden/auto` containers.

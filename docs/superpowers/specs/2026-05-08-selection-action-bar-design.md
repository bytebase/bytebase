# Unified Selection Action Bar — Design

**Date:** 2026-05-08
**Scope:** Frontend, React (`frontend/src/react/`)
**Owner:** sl@bytebase.com

## Problem

Three near-identical batch-operation bars exist in the React codebase:

1. `BatchOperationsBar` in `frontend/src/react/pages/settings/ProjectsPage.tsx` —
   Archive / Restore / Delete on selected projects.
2. `DatabaseBatchOperationsBar` in
   `frontend/src/react/components/database/DatabaseBatchOperationsBar.tsx` — up
   to seven actions (Change DB, Export, Sync Schema, Edit Labels, Edit
   Environment, Transfer, Unassign), used on `DatabasesPage`,
   `InstanceDetailPage`, and `ProjectDatabasesPage`.
3. `BatchOperationsBar` in `frontend/src/react/pages/settings/InstancesPage.tsx`
   — Sync (with split dropdown: Sync All / Sync New), Edit Environment, Assign
   License.

All three render the same pattern: a full-width strip with `bg-blue-100`, a
plain count label, and a row of `variant="ghost"` buttons inserted between the
page header and the table.

Concrete issues:

- `bg-blue-100` is a **raw color**. `frontend/AGENTS.md` requires semantic
  tokens (`bg-accent`, `bg-control`, etc.) and forbids raw `bg-blue-*` /
  `text-gray-*`.
- No visual hierarchy: destructive actions (Delete, Archive) sit alongside
  neutral edits (Edit Labels, Sync Schema) as identical ghost buttons. Users
  cannot tell at a glance which action is dangerous.
- The strip occupies vertical space above the table, pushing rows down. With 7
  buttons (Database case) the strip can wrap onto a second line, doubling the
  intrusion.
- No "clear selection" affordance — the only way to dismiss the bar is to
  uncheck rows manually.
- No `Escape`-to-clear keyboard shortcut.
- The selection count is a plain `<span>`, no emphasis or noun styling.
- Three independent implementations means style fixes must be made three
  times, and the InstancesPage variant currently uses an ad-hoc
  `useClickOutside` dropdown instead of a shared `DropdownMenu`.

## Goals

- One shared `SelectionActionBar` component used by all three call sites.
- Floating bottom-center pill (Linear/Notion/Gmail style), not a full-width
  strip — keeps the table layout stable and is visually lighter.
- Uses semantic tokens only; passes the React layering scanner
  (`frontend/scripts/check-react-layering.mjs`).
- Visual hierarchy: ghost neutral actions, red-text destructive action.
- Built-in clear affordance (✕ button + `Escape` keypress).
- Accepts both declarative `actions[]` (covers Projects + Database cases) and
  custom node `children` (covers InstancesPage's `SyncDropdown`).
- All four database-bar call sites recompile unchanged
  (`DatabaseBatchOperationsBar` becomes a thin wrapper that builds the actions
  array).

## Non-goals

- Replacing the existing `ConfirmDialog` / `ProjectListPreview` flow on
  ProjectsPage. Those continue to handle final confirmation; the bar only
  triggers them.
- Introducing an overflow ("⋯ More") menu. v1 uses `flex-wrap` (Tier A) for
  many-action cases. Overflow can be added later if Database page feels
  crowded.
- Changing per-permission / per-state visibility logic. Each call site
  continues to compute `hidden` / `disabled` flags and pass them in.
- Building a generic toolbar component. `SelectionActionBar` is specifically
  for "N items selected → operate on them" toolbars, not for general page
  toolbars.

## Design

### Placement

`SelectionActionBar` mounts via React portal into `getLayerRoot("overlay")`
(per `frontend/AGENTS.md` overlay policy — never `document.body`). The bar is
absolutely positioned `bottom-6 left-1/2 -translate-x-1/2`, with
`max-w-[90vw]`. No raw `z-index` — relies on the overlay layer's stacking.

The bar is rendered conditionally on `count > 0`. Mount/unmount is animated
via Base UI's transition primitives (or a small CSS transition on
`data-state`): slide-up + fade-in on appear (~150ms), reverse on dismiss.

### Visual

- Pill: `rounded-full bg-control border border-control-border shadow-lg
  px-2 py-1.5 flex items-center gap-x-1`.
- `bg-control` is theme-aware (light: white-ish, dark: dark surface). Compare
  to current `bg-blue-100` which has no dark-mode counterpart.

### Internal layout (left → right)

```
[ ✕ ]  N selected  │  Edit  Sync  Transfer  🗑 Delete
```

1. **Clear button** — icon-only ghost `Button` with an `X` icon, `size="sm"`,
   `aria-label={t("common.clear-selection")}`. Calls `onClear`.
2. **Count label** — `<span>` with `text-sm font-medium text-control`. The
   noun is rendered via `t()` at the call site (`"2 selected"` /
   `"3 databases selected"`).
3. **Vertical separator** — `<Separator orientation="vertical"
   className="h-5 mx-1" />`. Single divider only between count and the action
   group.
4. **Action buttons** — each `variant="ghost" size="sm"` with optional
   `lucide-react` icon + label. No vertical divider between actions (per
   approved design).
5. **Destructive action** — same `variant="ghost" size="sm"` but with a tone
   override applied via `cn()`:
   `"text-error hover:bg-error/10 hover:text-error focus-visible:ring-error"`.
   Per design choice "all ghost, only Delete is red", we do not use the solid
   `variant="destructive"` here — that would clash with the pill's neutral
   surface.

Many-action layout (Tier A): the action group uses `flex flex-wrap
items-center gap-x-1 gap-y-1`. On wide viewports it stays one line; on narrow
viewports the buttons wrap inside the pill. The pill itself does not wrap
(count + separator + actions are siblings of a top-level `flex items-center`).

### Component API

```tsx
// frontend/src/react/components/SelectionActionBar.tsx

import type { LucideIcon } from "lucide-react";
import type { ReactNode } from "react";

export interface SelectionAction {
  /** Stable key for React reconciliation. */
  key: string;
  label: string;
  icon?: LucideIcon;
  onClick: () => void;
  disabled?: boolean;
  /** When true, the action is omitted from the bar entirely. */
  hidden?: boolean;
  /**
   * Visual tone. "destructive" applies the red-text override.
   * Default: "neutral".
   */
  tone?: "neutral" | "destructive";
}

export interface SelectionActionBarProps {
  /** Number of selected items. The bar renders only when count > 0. */
  count: number;
  /**
   * Pre-formatted label (e.g. "2 selected", "3 databases selected").
   * The call site owns i18n + pluralization.
   */
  label: string;
  /** Clears the selection. Also invoked on Escape. */
  onClear: () => void;
  /** Declarative actions. Rendered in order. */
  actions?: SelectionAction[];
  /**
   * Custom action nodes rendered after `actions`. Used for actions that
   * require richer UI (e.g. InstancesPage's split-dropdown Sync).
   */
  children?: ReactNode;
}

export function SelectionActionBar(props: SelectionActionBarProps): JSX.Element | null;
```

### Keyboard

While `count > 0`, attach a window-level `keydown` listener for `Escape` →
calls `onClear`. Removed when `count === 0` or on unmount. Listener is
attached with `{ capture: false }` and bails out if `event.defaultPrevented`
or if the active element is inside a modal — to avoid hijacking dialog
dismissal.

### Accessibility

- The pill has `role="toolbar"` and `aria-label={t("common.batch-actions")}`.
- The count label is inside an `<output aria-live="polite">` so screen readers
  announce changes (e.g. "3 selected" → "5 selected").
- Clear button has explicit `aria-label`.
- Action icons are `aria-hidden`; accessible names come from each action's
  `label`.
- Focus management: when the bar appears, focus is **not** auto-stolen (the
  user is mid-table-interaction). Tab order naturally reaches the bar after
  the table.

### Layering

Per `frontend/AGENTS.md`:

- Mount target: `getLayerRoot("overlay")`. Never `document.body`.
- No raw `z-index` in the component or its consumers.
- No agent / critical layer use — this is a normal app overlay.
- After implementation: run `pnpm --dir frontend check` (which includes
  `node frontend/scripts/check-react-layering.mjs`).

## Migration

### 1. ProjectsPage

- Delete the local `BatchOperationsBar` function (ProjectsPage.tsx:120–319).
- Build the actions array inline in `ProjectsPage`:

```tsx
const batchActions: SelectionAction[] = [
  {
    key: "archive",
    label: t("common.archive"),
    icon: Archive,
    onClick: () => setShowArchiveConfirm(true),
    hidden: !hasActiveProjects,
  },
  {
    key: "restore",
    label: t("common.restore"),
    onClick: () => setShowRestoreConfirm(true),
    hidden: !hasArchivedProjects,
  },
  {
    key: "delete",
    label: t("common.delete"),
    icon: Trash2,
    onClick: () => setShowDeleteConfirm(true),
    tone: "destructive",
  },
];
```

- Render:

```tsx
{canDelete && (
  <SelectionActionBar
    count={selectedProjectList.length}
    label={t("project.batch.selected", { count: selectedProjectList.length })}
    onClear={() => setSelectedNames(new Set())}
    actions={batchActions}
  />
)}
```

- Keep `ConfirmDialog`, `ProjectListPreview`, and the three `handleBatch*`
  callbacks unchanged. State (`showArchiveConfirm` etc.) stays at the page
  level.

### 2. DatabaseBatchOperationsBar

- The file becomes a thin wrapper that preserves the existing external API
  (`databases`, `project?`, `onSyncSchema`, `onEditLabels`, `onEditEnvironment`,
  `onTransferProject?`, `onUnassign?`, `onChangeDatabase?`, `onExportData?`).
- Internally, it builds an `actions[]` array from the props and renders
  `SelectionActionBar`. All four call sites compile unchanged.
- Permission checks (`hasPermission`) move from rendering branches into the
  `disabled` field of each action.

### 3. InstancesPage

- Delete the local `BatchOperationsBar` (InstancesPage.tsx:591–644).
- Keep `SyncDropdown` (lines 539–589). Pass it as `children`:

```tsx
<SelectionActionBar
  count={selectedInstances.length}
  label={t("instance.selected-n-instances", { count: selectedInstances.length })}
  onClear={() => setSelectedInstanceNames(new Set())}
  actions={[
    {
      key: "edit-env",
      label: t("database.edit-environment"),
      icon: SquareStack,
      onClick: onEditEnvironment,
      disabled: !canUpdate,
    },
    showAssignLicense && {
      key: "assign-license",
      label: t("subscription.instance-assignment.assign-license"),
      icon: GraduationCap,
      onClick: onAssignLicense,
      disabled: !canUpdate,
    },
  ].filter(Boolean) as SelectionAction[]}
>
  <SyncDropdown disabled={!canSync || syncing} onSync={onSync} />
</SelectionActionBar>
```

Note: `SyncDropdown` will need a `size="sm"` cosmetic tweak so it visually
aligns with the bar's `size="sm"` action buttons. Optionally migrate
`SyncDropdown` to use the shared `DropdownMenu` primitive — this is a small
cleanup that fits the same change.

### 4. Cleanup

After the three migrations:

- Remove the `bg-blue-100` strip pattern from the codebase
  (`grep "bg-blue-100" frontend/src/react` should return zero matches in batch
  bars).
- Verify no remaining call sites of the old local `BatchOperationsBar`
  functions.

## Testing

- **Component test** for `SelectionActionBar` (`SelectionActionBar.test.tsx`):
  - Renders nothing when `count === 0`.
  - Renders `label`, all visible actions, and the clear button when `count > 0`.
  - `hidden: true` actions are omitted; `disabled: true` actions render as
    `disabled`.
  - Clicking an action button invokes its `onClick`.
  - Clicking the clear button invokes `onClear`.
  - Pressing `Escape` while `count > 0` invokes `onClear`.
  - `Escape` listener is **not** attached when `count === 0`.
  - `tone: "destructive"` applies the red-text override class.
  - Custom `children` render after declarative actions.
- **Integration smoke**: ProjectsPage, DatabasesPage, and InstancesPage each
  render the new bar when items are selected. (Existing tests cover selection
  state; assert the new bar's presence.)
- **Layering**: `pnpm --dir frontend check` passes after migration. No new
  raw-`z-index` or `document.body`-portal violations.
- **Type check**: `pnpm --dir frontend type-check` passes.

## Open questions

None. Tier A (flex-wrap) chosen for v1; Tier B (overflow menu) deferred. File
location is `frontend/src/react/components/SelectionActionBar.tsx` (top-level
components folder, not `ui/`), per design approval.

## Out of scope / follow-ups

- Tier B overflow menu — add only if the Database page (with up to 7 actions)
  feels crowded in practice.
- Migration of `SyncDropdown` to the shared `DropdownMenu` primitive — small
  optional cleanup; can be a follow-up if it bloats this PR.
- Other future selection-bar consumers (e.g. RolesPage, MembersPage) — they
  don't currently have batch bars; if they get one, they should use this
  component.

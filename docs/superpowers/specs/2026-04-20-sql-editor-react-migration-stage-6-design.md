# SQL Editor React Migration — Stage 6 Design

**Date:** 2026-04-20
**Author:** d@bytebase.com
**Status:** Draft

## 1. Goal & non-goals

**Goal:** Migrate `EditorCommon/QueryContextSettingPopover.vue` (194 lines) to React. Along the way: (a) build a new React `Popover` primitive that future migrations will reuse; (b) port `MaxRowCountSelect.vue` as a React shared component (the Vue version is retained because `DataExportButton.vue` still uses it).

**Non-goals (Stage 6):**
- Migrating `SaveSheetModal.vue` (Stage 7 — requires Emittery + async actions bridge).
- Migrating `SharePopover.vue` (Stage 8 — multi-caller pattern).
- Migrating `OpenAIButton` or any other EditorAction child.
- Deleting `MaxRowCountSelect.vue` — it's still used by `DataExportButton.vue`. Vue and React versions co-exist.
- Building Vue-in-React mount infrastructure.
- Bridging the Emittery `events` bus.

## 2. Architecture

Three new React files, one swap, one Vue file deletion:

| Action | File | Why |
|---|---|---|
| Create | `react/components/ui/popover.tsx` | New shadcn-style primitive wrapping Base UI Popover. First React Popover in the codebase; needed here, reusable for Stages 7+. |
| Create | `react/components/MaxRowCountSelect.tsx` | React version of Vue `MaxRowCountSelect.vue`. Vue version retained for DataExportButton. |
| Create | `react/components/sql-editor/QueryContextSettingPopover.tsx` | The Stage 6 primary target — replaces the Vue popover. |
| Modify | `views/sql-editor/EditorCommon/EditorAction.vue:49-51` | Swap `<QueryContextSettingPopover>` → `<ReactPageMount page="QueryContextSettingPopover" :disabled="..." />`. |
| Delete | `views/sql-editor/EditorCommon/QueryContextSettingPopover.vue` | Single caller (EditorAction), now swapped — safe to delete after verification. |

Each new React file gets a `*.test.tsx` with focused tests.

## 3. New React `Popover` primitive

### Design

Follow existing shadcn primitives (see `react/components/ui/dialog.tsx`, `dropdown-menu.tsx`). Wraps `@base-ui/react/popover`. Exposes compound parts:

```tsx
import { Popover as BasePopover } from "@base-ui/react/popover";
// Export:
export const Popover = BasePopover.Root;
export const PopoverTrigger = BasePopover.Trigger;
// PopoverContent wraps BasePopover.Portal + Positioner + Popup with project
//   styling: rounded-sm, border, shadow, bg-main, layer-surface, etc.
// Mirror the existing Tooltip primitive's layer/portal/styling conventions.
```

Minimum API:

```tsx
<Popover>
  <PopoverTrigger>...</PopoverTrigger>
  <PopoverContent side="bottom-end" sideOffset={4}>
    ...
  </PopoverContent>
</Popover>
```

`side` supports `"top" | "bottom" | "left" | "right" | "bottom-end"` etc. via `side` + `align`. Rendered into the `overlay` layer via `getLayerRoot("overlay")` for proper z-index stacking (follows the AGENTS.md overlay layering policy).

### Test (`popover.test.tsx`)

Minimal: renders trigger; clicking trigger opens content; clicking outside closes content. 2-3 tests.

## 4. React `MaxRowCountSelect`

Mirrors Vue `MaxRowCountSelect.vue` (91 lines):

**Props:** `{ value: number; onChange: (value: number) => void; maximum: number; quaternary?: boolean; className?: string }`

**Shape:** button trigger with label `"{Result limit} {n rows}"` and chevron icon → Popover content with:
- List of predefined row-count options (1, 100, 500, 1000, 5000, 10000, 100000 filtered by `maximum`)
- Custom number input at the bottom for arbitrary values within range

Uses shadcn `Button` (variant controlled by `quaternary` prop — map to `ghost` when quaternary, else `outline`), new `Popover` primitive, shadcn `Input` (with `type="number"`), `ChevronRight` icon from `lucide-react`, i18n keys `sql-editor.result-limit.self`, `common.rows.n-rows`, `common.rows.self`.

**Test (`MaxRowCountSelect.test.tsx`):**
- Renders current value in trigger label
- Options list filtered by `maximum`
- Selecting a preset calls `onChange` with the number
- Custom number input respects min/max clamping via `minmax` util

## 5. React `QueryContextSettingPopover`

Mirrors Vue source (194 lines). Core responsibilities:

- **Visibility:** renders nothing when `currentTab.mode === "ADMIN"` (read via `useVueState`).
- **Trigger:** accent-bordered ghost button with `ChevronDown` icon.
- **Data source radio group:** reads instance's `dataSources`, renders "Automatic" + each data source. Admin-type data sources get tooltip-disabled behavior when `queryDataPolicy.allowAdminDataSource === false` AND read-only data sources exist. Selection writes `tab.connection.dataSourceId` (or unsets if "Automatic").
- **Redis cluster mode sub-group:** only when engine is Redis. Sub-group is disabled unless the selected data source's `redisType === CLUSTER`. Tooltip explains the disabled reason.
- **Max row count select:** bottom of popover, uses the new `MaxRowCountSelect` component, bound to `useSQLEditorStore().resultRowsLimit` via direct mutation.

**Props:** `{ disabled?: boolean }`

**Stores:** `useSQLEditorTabStore`, `useConnectionOfCurrentSQLEditorTab`, `useSQLEditorStore` — all at component top level.

**Utilities:** `getInstanceResource`, `readableDataSourceType` from `@/utils` (verify availability).

**Test (`QueryContextSettingPopover.test.tsx`):**
- Renders nothing when current tab is in ADMIN mode
- Renders nothing when `disabled=true` on the NPopover side — actually Vue renders the popover but disables it; React equivalent disables the trigger button when `disabled=true`
- Selecting a data source writes `tab.connection.dataSourceId`
- Selecting "Automatic" removes `dataSourceId`
- Redis sub-group only shown when engine is Redis
- `MaxRowCountSelect` selection updates `resultRowsLimit` on the store

## 6. EditorAction.vue swap

`EditorAction.vue:49-51`:

```vue
<QueryContextSettingPopover
  :disabled="!showQueryContextSettingPopover || !allowQuery"
/>
```

Becomes:

```vue
<ReactPageMount
  page="QueryContextSettingPopover"
  :disabled="!showQueryContextSettingPopover || !allowQuery"
/>
```

Remove the Vue import:
```ts
import QueryContextSettingPopover from "./QueryContextSettingPopover.vue";
```

`ReactPageMount` already imported (from Stage 4). The surrounding `NButtonGroup` (wrapping run-query button + this popover trigger) stays Vue.

**Visual concern:** the Vue `QueryContextSettingPopover` button is a direct sibling of the run-query button inside `NButtonGroup`, sharing borders. After the swap, the React trigger button is inside a ReactPageMount wrapper div, breaking the connected-border behavior (same issue that led to Stage 5's `ChooserGroup` decision).

**Decision:** accept the visual seam for Stage 6 — it's one button in isolation, less visually disruptive than the 3-chooser group. If the seam looks bad during manual UX, we can either (a) extend `ReactPageMount` to support `display: contents` wrapper, or (b) migrate the run-query button as well (expands Stage 6 scope — defer if possible).

## 7. Vue file deletion

After the swap:
- Delete `views/sql-editor/EditorCommon/QueryContextSettingPopover.vue` (single caller, now orphaned).
- **Retain** `views/sql-editor/AsidePanel/GutterBar/...` — already deleted in Stage 3.
- **Retain** `components/RoleGrantPanel/MaxRowCountSelect.vue` — still used by `DataExportButton.vue`.

Verify via `rg` before deletion.

## 8. i18n keys

Verify in React locales; add missing:
- `sql-editor.result-limit.self`
- `common.rows.n-rows` (pluralization key)
- `common.rows.self`
- `data-source.select-query-data-source`
- `data-source.automatic-query-data-source`
- `sql-editor.redis-command.self`
- `sql-editor.redis-command.single-node`
- `sql-editor.redis-command.all-nodes`
- `sql-editor.redis-command.only-for-cluster`
- `sql-editor.query-context.admin-data-source-is-disallowed-to-query`

## 9. Verification

- `pnpm fix && check && type-check && test` all green
- ~10 new tests
- Manual UX: open SQL Editor → see the chevron-down button next to run-query (no popover seams visible in toolbar)
- Click chevron-down → popover opens with data source radios + row count selector
- Select a non-default data source → query uses that data source (persists on tab.connection.dataSourceId)
- For Redis: sub-group appears; selecting cluster data source enables node-mode radios
- Change row limit → affects subsequent queries

## 10. Out of scope (deferred)

- Emittery bridge (Stage 7)
- Async actions bridge (Stage 7)
- SharePopover migration (Stage 8)
- OpenAIButton migration (Stage 9+)
- EditorAction.vue shell migration (final stage)

## 11. Practical checklist

- [ ] `react/components/ui/popover.tsx` + test created, exported from ui/
- [ ] `react/components/MaxRowCountSelect.tsx` + test
- [ ] `react/components/sql-editor/QueryContextSettingPopover.tsx` + test
- [ ] 10 i18n keys verified/added in 5 React locales
- [ ] `EditorAction.vue:49-51` swapped to `<ReactPageMount>`; QueryContextSettingPopover import removed
- [ ] `QueryContextSettingPopover.vue` deleted after `rg` confirms single caller
- [ ] `pnpm fix && check && type-check && test` all pass
- [ ] Manual UX parity verified

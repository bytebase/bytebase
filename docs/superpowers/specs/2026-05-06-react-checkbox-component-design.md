# React Checkbox Component — Design

**Date:** 2026-05-06
**Scope:** Add a unified `Checkbox` UI primitive at `frontend/src/react/components/ui/checkbox.tsx` and migrate the components that currently manipulate `indeterminate` imperatively.

## Background

The React side of `frontend/` has 123 `<input type="checkbox">` usages across ~50 files. Pain points:

- **Inconsistent styling.** Some use `accent-accent`, some use `size-3.5`, most use no class at all and inherit native browser rendering.
- **Imperative `indeterminate`.** ~14 places set `indeterminate` via `ref` callback — `ref={(el) => { if (el) el.indeterminate = ... }}`. There is no declarative way to express tri-state in the current code.
- **No checkbox in `ui/`.** Other interactive primitives (`Switch`, `RadioGroup`, `SegmentedControl`, `Tabs`, `Select`, `Tooltip`) all live under `components/ui/` as Base UI wrappers. Checkbox is the missing peer.

The Vue side uses `NCheckbox` from naive-ui. With the Vue→React migration ongoing, new React code needs a parallel primitive.

## Non-goals

- Not migrating all 123 usages. The ~40 form-only "boolean toggle" usages (sql-editor panels, DataSourceForm, InstanceFormBody, IDPDetailPage, etc.) are out of scope here — they migrate organically.
- Not adding a label / description slot. Form labels in this codebase have too much variation (description text, tooltip, hover-row highlighting) to fit a single API.
- Not adding a lint rule banning `<input type="checkbox">` — the remaining ~40 form usages are legitimate.

## Component API

File: `frontend/src/react/components/ui/checkbox.tsx`

```tsx
type CheckboxSize = "sm" | "md"; // sm = 14px, md = 16px (default)

interface CheckboxProps {
  checked: boolean | "indeterminate";
  onCheckedChange?: (checked: boolean) => void;
  disabled?: boolean;
  size?: CheckboxSize; // default "md"
  className?: string;
  id?: string;
  name?: string; // optional, for form integration
  "aria-label"?: string;
}
```

Design notes:

- **`checked: boolean | "indeterminate"`** rather than separate `indeterminate` prop. A checkbox cannot be simultaneously `checked` and `indeterminate`; the union type forbids the invalid state. This matches Base UI's `Checkbox.Root` and shadcn's official Checkbox.
- **`onCheckedChange` always emits `boolean`.** Base UI's behavior: clicking an indeterminate checkbox transitions to `true` (does not return to indeterminate). All 13 existing call sites that use `e.target.checked` are equivalent.
- **No `forwardRef`.** Callers needing a label wrap the component in `<label>` themselves — matches the `Switch` precedent and keeps the API minimal.

## Visual

Built on `@base-ui/react/checkbox`. Visual language matches the existing `Switch`:

### Root (the box)

| State | Classes |
|---|---|
| Size: `md` | `size-4` |
| Size: `sm` | `size-3.5` |
| Default | `rounded-sm border border-control-border bg-background` |
| Checked / indeterminate | `bg-accent border-accent` |
| Hover (unchecked) | `border-accent/60` |
| Focus | `focus-visible:ring-2 focus-visible:ring-accent focus-visible:ring-offset-2` |
| Disabled | `disabled:opacity-50 disabled:cursor-not-allowed` |
| Transition | `transition-colors` |

### Indicator (rendered inside `Checkbox.Indicator`)

| State | Icon (lucide-react) | Class |
|---|---|---|
| `checked` | `<Check />` | `size-3 text-background` (md) / `size-2.5 text-background` (sm) |
| `indeterminate` | `<Minus />` | same as above |

`text-background` is correct because the checked-state fill is `bg-accent` (dark) — the indicator needs the inverse for contrast.

No manual `dark:` overrides — `bg-background`, `bg-accent`, `border-control-border` are semantic tokens that handle theming.

## Migration scope (this PR)

13 files containing imperative `indeterminate` manipulation:

1. `components/IssueTable.tsx`
2. `components/DatabaseResourceSelector.tsx`
3. `components/ExprEditor.tsx`
4. `components/database/DatabaseTableView.tsx`
5. `components/sql-review/RuleTable.tsx`
6. `components/SchemaEditorLite/Aside/NodeCheckbox.tsx`
7. `pages/settings/InstancesPage.tsx`
8. `pages/project/ProjectPlanDashboardPage.tsx`
9. `pages/project/database-detail/revision/DatabaseRevisionTable.tsx`
10. `pages/project/database-detail/catalog/SensitiveColumnTable.tsx`
11. `pages/project/export-center/DataExportPrepSheet.tsx`
12. `pages/project/plan-detail/components/PlanDetailChangesBranch.tsx`
13. `pages/project/plan-detail/components/deploy/DeployTaskToolbar.tsx`

Plus 3 files with paired row-level checkboxes (so the visual stays consistent within the same surface):

14. `components/ProjectTable.tsx`
15. `components/release/ReleaseFileTable.tsx`
16. `pages/settings/MembersPage.tsx`

**Done state:** every `if (el) el.indeterminate = ...` ref callback in React is gone; all imperative call sites are declarative `checked={state.indeterminate ? "indeterminate" : state.checked}`.

### Explicitly out of scope

The ~40 form-only `<input type="checkbox">` usages migrate organically:

- `components/IAMRemindDialog.tsx`
- `components/IssueLabelSelect.tsx`
- `components/InstanceAssignmentSheet.tsx`
- `components/sql-editor/**` (SheetTree, AccessGrantRequestDrawer, ConnectionPane, CodeViewer, IndexesTable, ColumnsTable, SequencesTable, ViewDetail)
- `components/sql-review/Panels.tsx`
- `components/sql-review/RuleComponents.tsx`
- `components/instance/DataSourceForm.tsx`
- `components/instance/InstanceFormBody.tsx`
- `components/SchemaEditorLite/Panels/**` (TableColumnEditor, IndexesEditor)
- `pages/settings/IDPDetailPage.tsx`

## Testing

### New tests

`frontend/src/react/components/ui/checkbox.test.tsx`:

- Renders unchecked, checked, indeterminate
- Click triggers `onCheckedChange(boolean)`:
  - unchecked → `true`
  - checked → `false`
  - indeterminate → `true`
- `disabled` blocks `onCheckedChange`
- `size="sm"` / `size="md"` render the expected dimension class
- Keyboard `Space` toggles

### Regression risk in existing tests

Base UI's `Checkbox.Root` renders `<button role="checkbox">` by default. It emits an additional hidden `<input type="checkbox">` only when `name` is set or it's inside a `<form>`. This changes selector behavior in:

- `release/ReleaseFileTable.test.tsx` — uses `container.querySelectorAll('input[type="checkbox"]').length`. After migration this counts 0 unless rows pass `name`. **Action:** during ReleaseFileTable migration, change the selector to `getAllByRole('checkbox')` in the same commit.
- Audit `sql-editor/SheetTree.test.tsx` and `pages/project/database-detail/panels/DatabaseCatalogPanel.test.tsx` for the same pattern; update if found.

### Validation gates

- `pnpm --dir frontend fix`
- `pnpm --dir frontend check`
- `pnpm --dir frontend type-check`
- `pnpm --dir frontend test`

## Open questions

None blocking. The Base UI selector change is the only known regression risk and has a mechanical fix.

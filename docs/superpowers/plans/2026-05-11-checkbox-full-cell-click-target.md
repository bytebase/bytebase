# Checkbox Full-Cell Click Target Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Make the entire selection column the click target across 5 React selection tables, so clicking near (but not exactly on) the checkbox toggles selection instead of navigating away.

**Architecture:** Three local shape-specific fixes — same conceptual change ("the whole select cell is the click target"), implemented per table shape. Column-driven tables get a new `onCellClick`/`onHeaderClick` column field. Inline-JSX tables get `onClick` directly on their select `<TableCell>` / `<TableHead>`. Flex-div rows wrap the `Checkbox` in a click-target `<div>`. Design doc: `docs/superpowers/specs/2026-05-11-checkbox-full-cell-click-target-design.md`.

**Tech Stack:** React, `@base-ui/react`, Tailwind CSS v4. Linear: [BYT-9447](https://linear.app/bytebase/issue/BYT-9447).

**Testing note:** This is a click-handler behavior change. Existing React tests in this repo require heavy mocking to render these tables (stores, router, i18n, proto utilities). For this fix, the cost of test scaffolding outweighs the value — verification is **manual QA in the browser** per the repo's frontend convention (see `AGENTS.md`: "For UI or frontend changes, start the dev server and use the feature in a browser before reporting the task as complete"). Each task includes a manual QA step; Task 6 is the consolidated final QA pass.

---

## File Structure

Files modified, by responsibility:

- `frontend/src/react/components/database/DatabaseTableView.tsx` — column-driven Table; add `onCellClick`/`onHeaderClick` mechanism + set on select column.
- `frontend/src/react/pages/settings/InstancesPage.tsx` — column-driven Table with its own `InstanceColumn`; mirror the mechanism.
- `frontend/src/react/pages/project/database-detail/revision/DatabaseRevisionTable.tsx` — inline JSX Table; attach `onClick` directly to select cells.
- `frontend/src/react/components/ProjectTable.tsx` — inline JSX Table; upgrade existing `stopPropagation` to also toggle, add header handler.
- `frontend/src/react/components/IssueTable.tsx` — flex-div row; wrap `Checkbox` in click-target `<div>`.

Not modified: `frontend/src/react/components/ui/checkbox.tsx` (the `onClick={(e) => e.stopPropagation()}` it already exposes is load-bearing — see design doc "Why this avoids double-toggle").

---

### Task 1: DatabaseTableView — add `onCellClick` / `onHeaderClick` to column type and wire the select column

**Files:**
- Modify: `frontend/src/react/components/database/DatabaseTableView.tsx`

- [ ] **Step 1: Add `onCellClick` and `onHeaderClick` to `DatabaseColumn`**

At `frontend/src/react/components/database/DatabaseTableView.tsx:51-61`, extend the interface:

```ts
interface DatabaseColumn {
  key: string;
  title: React.ReactNode;
  defaultWidth: number;
  minWidth?: number;
  resizable?: boolean;
  sortable?: boolean;
  sortKey?: DatabaseTableSort["key"];
  cellClassName?: string;
  render: (database: Database) => React.ReactNode;
  /** Click handler for the entire body `<TableCell>` for this column.
   *  Set on the select column to make the whole cell toggle selection.
   *  Consumers should call `e.stopPropagation()` to prevent row navigation. */
  onCellClick?: (database: Database, e: React.MouseEvent) => void;
  /** Click handler for the entire header `<TableHead>` for this column.
   *  Mirror of `onCellClick` for the header row (e.g. select-all). */
  onHeaderClick?: (e: React.MouseEvent) => void;
}
```

- [ ] **Step 2: Wire `onHeaderClick` on the `<TableHead>` in the render loop**

At `frontend/src/react/components/database/DatabaseTableView.tsx:289-307`, modify the `<TableHead>` to accept the new handler and add `cursor-pointer` when present:

```tsx
return (
  <TableHead
    key={col.key}
    sortable={col.sortable && sortable}
    sortActive={sortActive}
    sortDir={sort?.order ?? "asc"}
    onSort={
      col.sortable && sortable && colSortKey
        ? () => toggleSort(colSortKey)
        : undefined
    }
    resizable={col.resizable}
    onResizeStart={
      col.resizable ? (e) => onResizeStart(colIdx, e) : undefined
    }
    className={cn(col.onHeaderClick && "cursor-pointer")}
    onClick={col.onHeaderClick}
  >
    {col.title}
  </TableHead>
);
```

The existing `TableHead` already forwards a consumer `onClick` before invoking `onSort` (verified at `frontend/src/react/components/ui/table.tsx:94-97`). Since the select column is not sortable, only `onHeaderClick` runs.

- [ ] **Step 3: Wire `onCellClick` on the body `<TableCell>` in the row loop**

At `frontend/src/react/components/database/DatabaseTableView.tsx:340-346`, modify the `<TableCell>` rendering inside `databases.map`:

```tsx
{columns.map((col) => (
  <TableCell
    key={col.key}
    className={cn(
      "overflow-hidden",
      col.cellClassName,
      col.onCellClick && "cursor-pointer"
    )}
    onClick={col.onCellClick ? (e) => col.onCellClick!(db, e) : undefined}
  >
    {col.render(db)}
  </TableCell>
))}
```

- [ ] **Step 4: Set `onCellClick` / `onHeaderClick` on the select column and add `onClick` to the header checkbox**

At `frontend/src/react/components/database/DatabaseTableView.tsx:132-150`, replace the select column definition:

```ts
if (showSelection) {
  cols.push({
    key: "select",
    title: (
      <Checkbox
        checked={someSelected ? "indeterminate" : allSelected}
        onCheckedChange={toggleSelectAll}
        onClick={(e) => e.stopPropagation()}
      />
    ),
    defaultWidth: 48,
    onCellClick: (db, e) => {
      e.stopPropagation();
      toggleSelection(db.name);
    },
    onHeaderClick: (e) => {
      e.stopPropagation();
      toggleSelectAll();
    },
    render: (db) => (
      <Checkbox
        checked={selectedNames?.has(db.name) ?? false}
        onCheckedChange={() => toggleSelection(db.name)}
        onClick={(e) => e.stopPropagation()}
      />
    ),
  });
}
```

The `onClick={(e) => e.stopPropagation()}` on both header and body checkboxes is critical: it consumes the click event before it bubbles to the `<TableHead>` / `<TableCell>` `onClick`, preventing a double-toggle (once from `onCheckedChange`, once from the cell handler) when the user clicks the checkbox itself.

- [ ] **Step 5: Run frontend fix + type-check**

```bash
pnpm --dir frontend fix
pnpm --dir frontend type-check
```

Expected: both succeed. If `type-check` fails on the new optional fields, the issue is likely in the `useMemo` dependency list at `DatabaseTableView.tsx:262-270` — `toggleSelection` is now closed over by `onCellClick`. Add `toggleSelection`'s deps (`selectedNames`, `onSelectedNamesChange`) if not already present (they are at the time of writing). No new deps should be needed.

- [ ] **Step 6: Manual QA in browser**

Start the dev server:

```bash
pnpm --dir frontend dev
```

Navigate to Settings → Databases (the page that mounts `DatabaseTable`). Verify all of:

1. Click directly on a row checkbox → row toggles selected, no navigation.
2. Click on cell padding ~20px to the right of a row checkbox (still inside the 48px cell) → row toggles selected, no navigation.
3. Click on cell padding above/below the checkbox (still inside the cell) → toggles, no navigation.
4. Click on the database name → navigates to the database page (existing behavior preserved).
5. Click on the header checkbox directly → toggles all on current page.
6. Click on header cell padding → toggles all (no sort triggered — select column isn't sortable).
7. With one row selected, click the header → all selected (indeterminate → all).
8. With all rows selected, click the header → all deselected.
9. Cursor over the select cell shows pointer (`cursor-pointer`).

- [ ] **Step 7: Commit**

```bash
git add frontend/src/react/components/database/DatabaseTableView.tsx
git commit -m "$(cat <<'EOF'
fix(react-databases): full-cell click target for selection column

Adds onCellClick/onHeaderClick to DatabaseColumn so the entire 48px
select cell toggles selection instead of just the 16px checkbox. Fixes
the regression where clicking near (but not on) the checkbox would
navigate to the database page.

BYT-9447
EOF
)"
```

---

### Task 2: InstancesPage — apply the same mechanism to `InstanceColumn`

**Files:**
- Modify: `frontend/src/react/pages/settings/InstancesPage.tsx`

- [ ] **Step 1: Add `onCellClick` / `onHeaderClick` to `InstanceColumn`**

At `frontend/src/react/pages/settings/InstancesPage.tsx:103-113`:

```ts
interface InstanceColumn {
  key: string;
  title: React.ReactNode;
  defaultWidth: number;
  minWidth?: number;
  resizable?: boolean;
  sortable?: boolean;
  sortKey?: string;
  cellClassName?: string;
  render: (instance: Instance) => React.ReactNode;
  /** Click handler for the body `<TableCell>` of this column. */
  onCellClick?: (instance: Instance, e: React.MouseEvent) => void;
  /** Click handler for the header `<TableHead>` of this column. */
  onHeaderClick?: (e: React.MouseEvent) => void;
}
```

- [ ] **Step 2: Wire `onHeaderClick` on the `<TableHead>`**

At `frontend/src/react/pages/settings/InstancesPage.tsx:1186-1204`:

```tsx
<TableHead
  key={col.key}
  sortable={col.sortable}
  sortActive={
    col.sortable && sortKey === (col.sortKey ?? col.key)
  }
  sortDir={sortOrder}
  onSort={
    col.sortable
      ? () => toggleSort(col.sortKey ?? col.key)
      : undefined
  }
  resizable={col.resizable}
  onResizeStart={
    col.resizable ? (e) => onResizeStart(colIdx, e) : undefined
  }
  className={cn(col.onHeaderClick && "cursor-pointer")}
  onClick={col.onHeaderClick}
>
  {col.title}
</TableHead>
```

- [ ] **Step 3: Wire `onCellClick` on the body `<TableCell>`**

At `frontend/src/react/pages/settings/InstancesPage.tsx:1237-1244`:

```tsx
{columns.map((col) => (
  <TableCell
    key={col.key}
    className={cn(
      "overflow-hidden",
      col.cellClassName,
      col.onCellClick && "cursor-pointer"
    )}
    onClick={col.onCellClick ? (e) => col.onCellClick!(instance, e) : undefined}
  >
    {col.render(instance)}
  </TableCell>
))}
```

- [ ] **Step 4: Update the select column definition**

At `frontend/src/react/pages/settings/InstancesPage.tsx:1034-1052`:

```ts
{
  key: "select",
  title: (
    <Checkbox
      checked={someSelected ? "indeterminate" : allSelected}
      onCheckedChange={toggleSelectAll}
      onClick={(e) => e.stopPropagation()}
    />
  ),
  defaultWidth: 48,
  cellClassName: "px-4 py-2",
  onCellClick: (instance, e) => {
    e.stopPropagation();
    toggleSelection(instance.name);
  },
  onHeaderClick: (e) => {
    e.stopPropagation();
    toggleSelectAll();
  },
  render: (instance) => (
    <Checkbox
      checked={selectedNames.has(instance.name)}
      onCheckedChange={() => toggleSelection(instance.name)}
      onClick={(e) => e.stopPropagation()}
    />
  ),
},
```

- [ ] **Step 5: Run frontend fix + type-check**

```bash
pnpm --dir frontend fix
pnpm --dir frontend type-check
```

Expected: both succeed.

- [ ] **Step 6: Manual QA in browser**

Navigate to Settings → Instances. Run the same 9-point checklist as Task 1 Step 6, substituting "instance" for "database" and "Settings → Instances" for "Settings → Databases".

- [ ] **Step 7: Commit**

```bash
git add frontend/src/react/pages/settings/InstancesPage.tsx
git commit -m "$(cat <<'EOF'
fix(react-instances): full-cell click target for selection column

Mirrors the DatabaseTableView fix — adds onCellClick/onHeaderClick
to InstanceColumn so the entire select cell toggles selection
instead of navigating to the instance page.

BYT-9447
EOF
)"
```

---

### Task 3: DatabaseRevisionTable — add `onClick` directly to select cells

**Files:**
- Modify: `frontend/src/react/pages/project/database-detail/revision/DatabaseRevisionTable.tsx`

- [ ] **Step 1: Upgrade the select `<TableHead>`**

At `frontend/src/react/pages/project/database-detail/revision/DatabaseRevisionTable.tsx:62-67`:

```tsx
<TableHead
  className="w-12 cursor-pointer"
  onClick={(e) => {
    e.stopPropagation();
    toggleSelectAll();
  }}
>
  <Checkbox
    checked={someSelected ? "indeterminate" : allSelected}
    onCheckedChange={toggleSelectAll}
    onClick={(e) => e.stopPropagation()}
  />
</TableHead>
```

Note: confirm that `someSelected` and `allSelected` are already in scope — they should be among the local variables in the surrounding component. If only `allSelected` exists, leave the indeterminate prop as-is and only update the wrapping `<TableHead>` and inner `onClick={(e) => e.stopPropagation()}`.

- [ ] **Step 2: Upgrade the select `<TableCell>` in the row body**

At `frontend/src/react/pages/project/database-detail/revision/DatabaseRevisionTable.tsx:83-89`:

```tsx
<TableCell
  className="w-12 cursor-pointer"
  onClick={(e) => {
    e.stopPropagation();
    toggleSelection(revision.name);
  }}
>
  <Checkbox
    checked={selectedNames.has(revision.name)}
    onCheckedChange={() => toggleSelection(revision.name)}
    onClick={(e) => e.stopPropagation()}
  />
</TableCell>
```

The inner `onClick={(e) => e.stopPropagation()}` already exists at line 87 — keep it.

- [ ] **Step 3: Run frontend fix + type-check**

```bash
pnpm --dir frontend fix
pnpm --dir frontend type-check
```

Expected: both succeed.

- [ ] **Step 4: Manual QA in browser**

Navigate to a database's Revisions tab (Database detail page → Revisions). Run the 9-point checklist from Task 1 Step 6, substituting "revision" for "database" and "the revision detail page" for "the database page" (Step 4 of the checklist).

- [ ] **Step 5: Commit**

```bash
git add frontend/src/react/pages/project/database-detail/revision/DatabaseRevisionTable.tsx
git commit -m "$(cat <<'EOF'
fix(react-revisions): full-cell click target for selection column

The select TableHead/TableCell now toggle selection on any click,
not just on the 16px checkbox itself.

BYT-9447
EOF
)"
```

---

### Task 4: ProjectTable — upgrade existing `stopPropagation` handler to also toggle

**Files:**
- Modify: `frontend/src/react/components/ProjectTable.tsx`

The existing select cell at `frontend/src/react/components/ProjectTable.tsx:208-211` has `onClick={(e) => e.stopPropagation()}` — clicks on the cell don't navigate, but they also don't toggle. Upgrade to toggle (respecting the `disabled` case for the default project). The header at lines 144-151 has no cell-level handler at all — add one.

- [ ] **Step 1: Upgrade the select `<TableCell>`**

At `frontend/src/react/components/ProjectTable.tsx:207-219`:

```tsx
{showSelection ? (
  <TableCell
    className={cn(
      "w-12",
      !isDefault && "cursor-pointer"
    )}
    onClick={(e) => {
      e.stopPropagation();
      if (isDefault) return;
      handleToggleRow(project.name);
    }}
  >
    <Checkbox
      checked={isSelected}
      aria-label={t("common.select")}
      disabled={isDefault}
      onCheckedChange={() => handleToggleRow(project.name)}
      onClick={(e) => e.stopPropagation()}
      className="disabled:opacity-50"
    />
  </TableCell>
) : ...
```

The `if (isDefault) return;` preserves the existing disabled-default behavior — clicking the cell of a default-project row stops the row's onClick (no navigation) but does not toggle. `cursor-pointer` is suppressed on default rows so the affordance matches.

The inner `onClick={(e) => e.stopPropagation()}` is added to the Checkbox for symmetry with the rest of the fix (prevents double-toggle when the checkbox itself is clicked, now that the cell has its own click handler).

If `cn` is not yet imported in this file, add it: `import { cn } from "@/react/lib/utils";` (check the existing import list first).

- [ ] **Step 2: Upgrade the header `<TableHead>`**

At `frontend/src/react/components/ProjectTable.tsx:144-151`:

```tsx
{showSelection ? (
  <TableHead
    className="w-12 cursor-pointer"
    onClick={(e) => {
      e.stopPropagation();
      handleSelectAll();
    }}
  >
    <Checkbox
      checked={allSelected}
      aria-label={t("common.select-all")}
      onCheckedChange={handleSelectAll}
      onClick={(e) => e.stopPropagation()}
    />
  </TableHead>
) : ...
```

- [ ] **Step 3: Run frontend fix + type-check**

```bash
pnpm --dir frontend fix
pnpm --dir frontend type-check
```

Expected: both succeed.

- [ ] **Step 4: Manual QA in browser**

Navigate to Settings → Projects. Run the 9-point checklist (Task 1 Step 6) substituting "project". Additionally:

10. Click on the select cell of the **default project** row → no toggle (the checkbox stays disabled), no navigation. Cursor is the default cursor (not pointer).
11. Click on the row content of the default project → navigates as before.

- [ ] **Step 5: Commit**

```bash
git add frontend/src/react/components/ProjectTable.tsx
git commit -m "$(cat <<'EOF'
fix(react-projects): full-cell click target for selection column

Upgrades the existing stopPropagation handler on the select cell to
also toggle selection on any click (still respecting the default
project's disabled state). Adds the matching handler on the header
for select-all.

BYT-9447
EOF
)"
```

---

### Task 5: IssueTable — wrap `Checkbox` in click-target `<div>`

**Files:**
- Modify: `frontend/src/react/components/IssueTable.tsx`

The row uses flex layout with `items-start py-3` — the checkbox sits at the top of the row with `mt-1`, leaving substantial empty padding below that currently navigates the row. Wrap it so the click target fills the row's vertical extent (but not horizontally, to keep the title cell navigable).

- [ ] **Step 1: Replace the Checkbox with a wrapper + Checkbox**

At `frontend/src/react/components/IssueTable.tsx:633-638`:

```tsx
<div
  className="shrink-0 self-stretch pt-1 cursor-pointer"
  onClick={(e) => {
    e.stopPropagation();
    onToggleSelection();
  }}
>
  <Checkbox
    checked={selected}
    onCheckedChange={() => onToggleSelection()}
    onClick={(e) => e.stopPropagation()}
  />
</div>
```

Notes:
- `self-stretch` overrides the parent's `items-start` for this child, so the wrapper fills the row's full vertical height regardless of where the row content ends. The clickable area is the full row height × 16px checkbox width.
- `pt-1` replaces the original `mt-1` on the checkbox — same visual top alignment, but the wrapper (not the checkbox) is responsible for it.
- `shrink-0` moves from the inner Checkbox to the wrapper so the wrapper itself doesn't shrink.
- The wrapper width is implicit (16px from the Checkbox content) — no `pr-*` is added, to preserve the existing `gap-x-2` between checkbox and issue title and to keep the title fully navigable.

- [ ] **Step 2: Run frontend fix + type-check**

```bash
pnpm --dir frontend fix
pnpm --dir frontend type-check
```

Expected: both succeed.

- [ ] **Step 3: Manual QA in browser**

Navigate to My Issues (`/my-issues`) and Project → Issues (`/project/<id>/issues`). For each:

1. Click directly on a row checkbox → toggles, no navigation.
2. Click on the empty vertical strip below/above the checkbox (still in the checkbox column, before the gap before the title) → toggles, no navigation.
3. Click on the issue title → navigates to issue page.
4. Click on the row body (status icon, labels, etc.) → navigates.
5. Verify the row's height and visual layout did not shift (no extra space introduced by the wrapper). Compare against `main` if needed.

- [ ] **Step 4: Commit**

```bash
git add frontend/src/react/components/IssueTable.tsx
git commit -m "$(cat <<'EOF'
fix(react-issues): full-height click target for selection checkbox

Wraps the row Checkbox in a self-stretch div so clicks anywhere in
the checkbox column's vertical extent toggle selection instead of
navigating to the issue page. Horizontal extent is unchanged to keep
the title cell navigable.

BYT-9447
EOF
)"
```

---

### Task 6: Final validation pass

- [ ] **Step 1: Run full frontend gate**

```bash
pnpm --dir frontend check
pnpm --dir frontend type-check
pnpm --dir frontend test
```

Expected: all three succeed. `check` is the CI-equivalent validation (lint + format + organize imports without modifying files). `test` should pass — no test files were added or modified, but the lint configuration may catch issues in the modified files.

If `check` fails on a "needs format" warning, run `pnpm --dir frontend fix` and re-run `check`.

- [ ] **Step 2: Cross-surface manual QA sweep**

Verify all five surfaces in one browser session:

| Surface | URL |
|---|---|
| Databases | `/setting/database` |
| Instances | `/setting/instance` |
| Database revisions | open any database, go to "Revisions" tab |
| Projects | `/setting/project` |
| Issues (workspace) | `/my-issues` |
| Issues (project) | open any project, go to "Issues" tab |

For each surface, perform the cell-padding click test (Step 2 of the per-task checklists): click ~20px from the checkbox, still in the select column. Confirm it toggles selection without navigation.

- [ ] **Step 3: Verify no `cursor-pointer`-only false positives**

Spot-check the layering scanner per `frontend/AGENTS.md`:

```bash
pnpm --dir frontend check
```

This runs the layering scanner (`check-react-layering.mjs`) as part of `check`. No new z-index or portal changes were introduced, so the scanner should pass clean. If it flags anything, the issue is unrelated to this PR — confirm before continuing.

- [ ] **Step 4: Commit the Linear reference (only if any final tweaks were made)**

If any of the previous QA passes surfaced a small correction (e.g. a missed `cursor-pointer`, a TypeScript narrowing), commit it separately:

```bash
git status
git add <fixed-files>
git commit -m "fix(react-tables): post-QA polish for full-cell click target

BYT-9447"
```

If no changes were made in this step, skip the commit.

---

## Self-review

**Spec coverage:**
- Shape A (column-driven): Tasks 1 (DatabaseTableView) + 2 (InstancesPage) ✓
- Shape B (inline JSX): Tasks 3 (DatabaseRevisionTable) + 4 (ProjectTable) ✓
- Shape C (flex-div): Task 5 (IssueTable) ✓
- "Why this avoids double-toggle" requirement (`onClick={(e) => e.stopPropagation()}` on every Checkbox, header and body, in all surfaces): present in every task's code block ✓
- Manual testing checklist (per design doc § Testing): Task 1 Step 6 defines the canonical 9-point list; later tasks reuse it ✓
- File-level change summary in design doc: matches Tasks 1–5 ✓
- Out of scope (no shared helper, no Checkbox primitive change): respected — no `<SelectCell>` introduction, `checkbox.tsx` is untouched ✓

**Placeholder scan:** None of "TBD/TODO/implement later/add appropriate error handling/similar to Task N" appear. All code blocks are complete.

**Type consistency:** `onCellClick`/`onHeaderClick` signatures are identical across `DatabaseColumn` (Task 1) and `InstanceColumn` (Task 2), modulo the entity type. `handleToggleRow` / `handleSelectAll` (Task 4) and `toggleSelection` / `toggleSelectAll` (Tasks 1, 2, 3) are the names already used in their respective files — no renames introduced.

**Known gap:** The `someSelected`/`allSelected`/`toggleSelectAll` names in `DatabaseRevisionTable.tsx` (Task 3) are referenced based on the visible code at line 64-65. If the actual local variable names differ, adapt them while keeping the same logic. This is the only place where verification on-the-fly may be needed.

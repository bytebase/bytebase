# Unify React list-page tables — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace four raw `<table>` callsites in React list pages with the unified `Table` component at `frontend/src/react/components/ui/table.tsx`, normalizing visuals to the unified component's defaults.

**Architecture:** Pure markup migration. Each callsite swaps raw `<table>` / `<thead>` / `<tbody>` / `<tr>` / `<th>` / `<td>` for the unified `Table` / `TableHeader` / `TableHead` / `TableBody` / `TableRow` / `TableCell`. Sortable headers and column resize use props the unified `Table` already exposes (`sortable`, `sortActive`, `sortDir`, `onSort`, `resizable`, `onResizeStart`). No new APIs are added to the unified `Table`. One PR per page, smallest first.

**Tech Stack:** React, TypeScript, Tailwind CSS v4, Base UI, `class-variance-authority`. Lint via `pnpm --dir frontend fix` / `check`. Type check via `pnpm --dir frontend type-check`. Manual visual verification in `pnpm --dir frontend dev`.

**Spec:** `docs/superpowers/specs/2026-05-06-unify-react-tables-design.md`

**Note on testing:** These are visual UI markup migrations. There are no existing unit tests for these table renderings, and the spec's validation strategy is type-check + lint + manual visual diff rather than TDD. Each task ends with the validation commands and a manual check pass.

**Reference — unified Table API surface (from `frontend/src/react/components/ui/table.tsx`):**

- `Table` — `<table>` wrapper, `w-full caption-bottom text-sm`. Accepts arbitrary children (e.g. `<colgroup>`) and `style` (inline `width` overrides `w-full`).
- `TableHeader` — `<thead>` wrapper, adds `[&_tr]:border-b`.
- `TableBody` — `<tbody>` wrapper. `striped?: boolean` (default `true`) toggles `[&_tr:nth-child(even)]:bg-control-bg/50`.
- `TableRow` — `<tr>` wrapper with hover and selected styles. `striped?: boolean` (default `true`); `striped={false}` adds `!bg-transparent`.
- `TableHead` — `<th>` wrapper, `h-10 px-4 py-2 text-left align-middle font-medium text-control-light`. Props: `sortable`, `sortActive`, `sortDir` (`"asc" | "desc"`), `onSort`, `resizable`, `onResizeStart`. When `sortable` it renders the children inside an inline-flex span with the unified `ArrowUp`/`ArrowDown`/`ArrowUpDown` indicator. When `resizable && onResizeStart`, it renders the shared `ColumnResizeHandle`.
- `TableCell` — `<td>` wrapper, `px-4 py-3 align-middle text-sm text-control`.

`TableHeadSortDirection` is exported from the same module.

---

## Task 1: Migrate inline ReleaseTable in ProjectReleaseDashboardPage

**Files:**
- Modify: `frontend/src/react/pages/project/ProjectReleaseDashboardPage.tsx` (the inline `ReleaseTable` and `ReleaseRow` functions; currently lines 186–281)

This is the smallest case: no sort, no resize, no selection, no pagination logic in the table itself. The page already uses `cn` from `@/react/lib/utils`, so no new imports needed besides the unified `Table` parts.

- [ ] **Step 1: Add the unified Table imports**

In the import block at the top of `ProjectReleaseDashboardPage.tsx`, add a new line in alphabetical order (between `Button` and `PagedTableFooter`):

```tsx
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/react/components/ui/table";
```

- [ ] **Step 2: Replace the `ReleaseTable` function body**

Replace the function body (currently lines 186–209) with:

```tsx
function ReleaseTable({ releases }: { releases: Release[] }) {
  const { t } = useTranslation();

  return (
    <div className="overflow-x-auto">
      <Table>
        <TableHeader>
          <TableRow>
            <TableHead className="w-75">{t("common.name")}</TableHead>
            <TableHead>{t("release.files")}</TableHead>
            <TableHead className="w-32">{t("common.created-at")}</TableHead>
          </TableRow>
        </TableHeader>
        <TableBody>
          {releases.map((release) => (
            <ReleaseRow key={release.name} release={release} />
          ))}
        </TableBody>
      </Table>
    </div>
  );
}
```

- [ ] **Step 3: Replace the `ReleaseRow` function's JSX**

In `ReleaseRow` (currently lines 215–281), replace the `<tr>...</tr>` block (currently lines 242–280) with:

```tsx
return (
  <TableRow className="cursor-pointer" onClick={onRowClick}>
    <TableCell>
      <span
        className={cn(
          "truncate",
          isDeleted && "text-control-light line-through"
        )}
      >
        {releaseName}
      </span>
    </TableCell>
    <TableCell>
      <div className="flex flex-col items-start gap-1">
        {showFiles.map((file, idx) => (
          <p key={idx} className="w-full truncate">
            {file.version && (
              <span className="mr-2 inline-flex items-center rounded-full bg-control-bg px-2 py-0.5 text-xs">
                {file.version}
              </span>
            )}
            {file.path}
          </p>
        ))}
        {release.files.length > MAX_SHOW_FILES_COUNT && (
          <p className="text-control-placeholder text-xs italic">
            {t("release.total-files", { count: release.files.length })}
          </p>
        )}
      </div>
    </TableCell>
    <TableCell>
      <HumanizeTs ts={createTimeTs} />
    </TableCell>
  </TableRow>
);
```

Notes on what changed:
- The custom `border-b cursor-pointer hover:bg-control-bg` becomes just `cursor-pointer` — `TableRow` already provides `border-b border-block-border` and `hover:bg-control-bg/60`. Per the spec, accept the unified hover (`/60` instead of solid).
- The custom `py-2 px-4` cell padding becomes the unified `px-4 py-3`.
- The header row's custom `border-b text-left text-control-light` is dropped — `TableHeader` adds the bottom border and `TableHead` has `text-control-light` and `text-left` baked in.

- [ ] **Step 4: Run the frontend fixer**

```bash
pnpm --dir frontend fix
```

Expected: imports get sorted; no errors.

- [ ] **Step 5: Type check**

```bash
pnpm --dir frontend type-check
```

Expected: PASS, no errors related to this file.

- [ ] **Step 6: Lint check**

```bash
pnpm --dir frontend check
```

Expected: PASS.

- [ ] **Step 7: Manual visual verification**

Start the dev server (`pnpm --dir frontend dev`) and open a project that has at least one release. Navigate to `/projects/<id>/releases`. Verify:

- The table renders with three columns (Name, Files, Created at).
- Rows are clickable and navigate to the release detail.
- The filter bar above still works (selecting a category filters the list).
- Striped rows alternate (this is new — ReleaseTable used to have no striping; the unified Table adds it by default).

Confirm visually that nothing looks broken. If you decide striping is undesirable here, pass `striped={false}` to `<TableBody>` and re-check; otherwise keep the default.

- [ ] **Step 8: Commit**

```bash
git add frontend/src/react/pages/project/ProjectReleaseDashboardPage.tsx
git commit -m "$(cat <<'EOF'
refactor(react): migrate ReleaseTable to unified Table component

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Task 2: Migrate ProjectTable

**Files:**
- Modify: `frontend/src/react/components/ProjectTable.tsx`

This is a focused component file. The migration drops the local `SortIndicator` (rotating ChevronDown) in favor of the unified Table's `ArrowUp`/`ArrowDown`/`ArrowUpDown`, drops the `bg-control-bg` header background, and uses `TableHead`'s `sortable` / `sortActive` / `sortDir` / `onSort` props.

- [ ] **Step 1: Update the imports**

Replace the existing import block (currently lines 1–10) with:

```tsx
import { Check } from "lucide-react";
import type { MouseEvent as ReactMouseEvent, ReactNode } from "react";
import { useTranslation } from "react-i18next";
import { HighlightLabelText } from "@/react/components/HighlightLabelText";
import { Badge } from "@/react/components/ui/badge";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/react/components/ui/table";
import { Tooltip } from "@/react/components/ui/tooltip";
import { getProjectName } from "@/react/lib/resourceName";
import { cn } from "@/react/lib/utils";
import { State } from "@/types/proto-es/v1/common_pb";
import type { Project } from "@/types/proto-es/v1/project_service_pb";
```

`ChevronDown` is removed because the unified `TableHead` provides its own sort indicator.

- [ ] **Step 2: Update the file-level doc comment on `ProjectTable`**

Replace the JSDoc above the `ProjectTable` function (currently lines 68–87) with:

```tsx
/**
 * React port of `frontend/src/components/v2/Model/ProjectV1Table.vue`.
 *
 * Renders the standard project listing — id / title / labels columns,
 * with optional leading current-project check, leading selection
 * checkboxes, and trailing action-dropdown slot. Matches the prop
 * shape of the Vue component so call sites read the same way.
 *
 * Two surfaces consume this today:
 *   - `ProjectsPage` (settings) — `showSelection` + `showActions` +
 *     server-side sort by title.
 *   - `ProjectSwitchPanel` (header popover) — `showLabels=false` +
 *     `currentProject` set to the active project.
 */
```

The note about "uses plain HTML table elements to preserve … bg-control-bg header / px-4 py-2 padding / rotating ChevronDown" is no longer accurate after this migration.

- [ ] **Step 3: Replace the `<table>` body with the unified primitives**

Replace the JSX returned by `ProjectTable` (currently lines 137–278) with:

```tsx
return (
  <Table className={className}>
    <TableHeader>
      <TableRow>
        {showSelection ? (
          <TableHead className="w-12">
            <input
              type="checkbox"
              aria-label={t("common.select-all")}
              checked={allSelected}
              onChange={handleSelectAll}
              className="rounded-xs border-control-border"
            />
          </TableHead>
        ) : showLeadingCheck ? (
          <TableHead className="w-8 px-2" />
        ) : null}
        <TableHead className="min-w-[128px]">{t("common.id")}</TableHead>
        <TableHead
          className="min-w-[200px]"
          sortable={!!onSortChange}
          sortActive={sortKey === "title"}
          sortDir={sortOrder}
          onSort={() => onSortChange?.("title")}
        >
          {t("project.table.name")}
        </TableHead>
        {showLabels ? (
          <TableHead className="min-w-[240px] hidden md:table-cell">
            {t("common.labels")}
          </TableHead>
        ) : null}
        {showActions ? <TableHead className="w-[50px]" /> : null}
      </TableRow>
    </TableHeader>
    <TableBody>
      {loading && projectList.length === 0 ? (
        <TableRow>
          <TableCell
            colSpan={totalColumns}
            className="px-4 py-8 text-center text-control-placeholder"
          >
            <div className="flex items-center justify-center gap-x-2">
              <div className="animate-spin size-4 border-2 border-accent border-t-transparent rounded-full" />
              {t("common.loading")}
            </div>
          </TableCell>
        </TableRow>
      ) : projectList.length === 0 ? (
        <TableRow>
          <TableCell
            colSpan={totalColumns}
            className="px-4 py-8 text-center text-control-placeholder"
          >
            {emptyContent ?? t("common.no-data")}
          </TableCell>
        </TableRow>
      ) : (
        projectList.map((project) => {
          const resourceId = getProjectName(project.name);
          const isDefault = resourceId === "default";
          const isCurrent = currentProject?.name === project.name;
          const isSelected = selectedProjectNames.includes(project.name);
          return (
            <TableRow
              key={project.name}
              className={cn(onRowClick && "cursor-pointer")}
              onClick={(event) => onRowClick?.(project, event)}
            >
              {showSelection ? (
                <TableCell
                  className="w-12"
                  onClick={(e) => e.stopPropagation()}
                >
                  <input
                    type="checkbox"
                    aria-label={t("common.select")}
                    checked={isSelected}
                    disabled={isDefault}
                    onChange={() => handleToggleRow(project.name)}
                    className="rounded-xs border-control-border disabled:opacity-50"
                  />
                </TableCell>
              ) : showLeadingCheck ? (
                <TableCell className="w-8 px-2">
                  {isCurrent ? (
                    <Check className="size-4 text-accent" />
                  ) : null}
                </TableCell>
              ) : null}
              <TableCell>
                <HighlightLabelText text={resourceId} keyword={keyword} />
              </TableCell>
              <TableCell>
                <div className="flex items-center gap-x-2">
                  <HighlightLabelText
                    text={project.title || resourceId}
                    keyword={keyword}
                  />
                  {project.state === State.DELETED ? (
                    <Badge variant="warning" className="text-xs">
                      {t("common.archived")}
                    </Badge>
                  ) : null}
                </div>
              </TableCell>
              {showLabels ? (
                <TableCell className="hidden md:table-cell">
                  <LabelsCell labels={project.labels ?? {}} />
                </TableCell>
              ) : null}
              {showActions ? (
                <TableCell
                  className="w-[50px]"
                  onClick={(e) => e.stopPropagation()}
                >
                  <div className="flex justify-end">
                    {renderActions?.(project)}
                  </div>
                </TableCell>
              ) : null}
            </TableRow>
          );
        })
      )}
    </TableBody>
  </Table>
);
```

Notes on what changed:
- The header `bg-control-bg` background is dropped (per spec — normalize to unified defaults).
- The local `SortIndicator` is no longer used; the unified `TableHead`'s `sortable` API renders the standard indicator.
- The custom `i % 2 === 1 && "bg-control-bg/50"` striping is dropped because `TableBody` provides striping by default. The current visual (every-other row tinted) stays.
- `onRowClick && "cursor-pointer hover:bg-control-bg"` becomes `onRowClick && "cursor-pointer"` because `TableRow` already provides hover (`hover:bg-control-bg/60`).
- Cell padding shifts from `px-4 py-2` to the unified `px-4 py-3`.

- [ ] **Step 4: Delete the now-unused local `SortIndicator`**

Remove the entire `SortIndicator` function (currently lines 281–305) including its JSDoc. Also remove the `ProjectTableSortDirection` type alias usage check — it remains used via the `sortOrder` prop type, so keep the `ProjectTableSortDirection` export. Only `SortIndicator` is deleted.

- [ ] **Step 5: Run the frontend fixer**

```bash
pnpm --dir frontend fix
```

Expected: imports get sorted; no errors.

- [ ] **Step 6: Type check**

```bash
pnpm --dir frontend type-check
```

Expected: PASS, no errors related to this file. The `sortDir` prop on `TableHead` accepts `"asc" | "desc" | undefined` and `ProjectTableSortDirection` is `"asc" | "desc"`, so passing `sortOrder` directly is type-safe.

- [ ] **Step 7: Lint check**

```bash
pnpm --dir frontend check
```

Expected: PASS.

- [ ] **Step 8: Manual visual verification**

Start the dev server (or keep it running). Verify both consumers of `ProjectTable`:

1. **Settings → Projects** (`/projects` settings list): table should render with the unified header style (no `bg-control-bg` background), unified arrow sort indicator on the Name column. Click the Name header to toggle sort. Selection checkboxes still work; bulk action bar still appears. Pagination still works.
2. **Project switcher** (header popover with `currentProject` set): leading column shows the check icon for the current project. No labels column. Clicking a row navigates.

Confirm both look right. The visual diff is intentional per the spec (Question 3, option A).

- [ ] **Step 9: Commit**

```bash
git add frontend/src/react/components/ProjectTable.tsx
git commit -m "$(cat <<'EOF'
refactor(react): migrate ProjectTable to unified Table component

Drop the local SortIndicator and bg-control-bg header background in
favor of the unified Table's defaults.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Task 3: Migrate AuditLogTable

**Files:**
- Modify: `frontend/src/react/components/AuditLogTable.tsx`

The page uses `useColumnWidths` + a `<colgroup>` for explicit per-column widths and `style={{ width: totalWidth }}` + `table-fixed` for layout. The unified `Table` accepts arbitrary children (so `<colgroup>` works) and forwards `style`/`className`. Sort indicators move from the local `renderSortIndicator` (rotating ChevronDown) to `TableHead`'s built-in indicator. Resize wires through `TableHead`'s `resizable` + `onResizeStart` props.

- [ ] **Step 1: Update the imports**

Replace the existing imports for `ChevronDown` (line 6) and the `ColumnResizeHandle` import (line 25). Specifically:

In the `lucide-react` import (currently line 5–11), drop `ChevronDown`:

```tsx
import { Download, ExternalLink, Maximize2, X } from "lucide-react";
```

Drop the `ColumnResizeHandle` import line (currently line 25) — `TableHead` renders the handle internally now.

Add the unified `Table` imports. In the alphabetical block of `@/react/components/ui/*` imports, add (between `LAYER_SURFACE_CLASS` and the next block — adjust insertion point so the file passes `pnpm --dir frontend fix`):

```tsx
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/react/components/ui/table";
```

- [ ] **Step 2: Delete the local `renderSortIndicator`**

Inside `AuditLogTable`, delete the `renderSortIndicator` constant (currently lines 692–703):

```tsx
const renderSortIndicator = (columnKey: string) => {
  if (sortKey !== columnKey)
    return <ChevronDown className="h-4 w-4 text-control-light" />;
  return (
    <ChevronDown
      className={cn(
        "h-4 w-4 text-accent transition-transform",
        sortOrder === "asc" && "rotate-180"
      )}
    />
  );
};
```

Sort indication will be handled by `TableHead`'s `sortable` API.

- [ ] **Step 3: Replace the `<table>` body with the unified primitives**

Replace the table block (currently lines 736–811) with:

```tsx
<Table
  className="border-t border-block-border table-fixed"
  style={{ width: `${totalWidth}px` }}
>
  <colgroup>
    {widths.map((w, i) => (
      <col key={columns[i].key} style={{ width: `${w}px` }} />
    ))}
  </colgroup>
  <TableHeader>
    <TableRow>
      {columns.map((col, colIdx) => (
        <TableHead
          key={col.key}
          className="text-sm text-main whitespace-nowrap"
          sortable={col.sortable}
          sortActive={col.sortable && sortKey === col.key}
          sortDir={sortOrder}
          onSort={
            col.sortable ? () => toggleSort(col.key) : undefined
          }
          resizable={col.resizable}
          onResizeStart={
            col.resizable ? (e) => onResizeStart(colIdx, e) : undefined
          }
        >
          {col.title}
        </TableHead>
      ))}
    </TableRow>
  </TableHeader>
  <TableBody striped={false}>
    {loading && auditLogs.length === 0 ? (
      <TableRow>
        <TableCell
          colSpan={columns.length}
          className="text-center py-8 text-control-placeholder"
        >
          {t("common.loading")}
        </TableCell>
      </TableRow>
    ) : auditLogs.length === 0 ? (
      <TableRow>
        <TableCell
          colSpan={columns.length}
          className="text-center py-8 text-control-placeholder"
        >
          {t("common.no-data")}
        </TableCell>
      </TableRow>
    ) : (
      auditLogs.map((log, idx) => (
        <TableRow key={log.name || idx}>
          {columns.map((col) => (
            <TableCell
              key={col.key}
              className="align-top overflow-hidden"
            >
              {col.render(log)}
            </TableCell>
          ))}
        </TableRow>
      ))
    )}
  </TableBody>
</Table>
```

Notes on what changed:
- `<table className="text-sm border-t border-block-border table-fixed">` becomes `<Table className="border-t border-block-border table-fixed">` (the unified `Table` already sets `text-sm`).
- `<colgroup>` is passed as a direct child of `<Table>` — the unified component spreads `...props` onto `<table>` and renders any children.
- The custom `text-main` and `whitespace-nowrap` per-header are kept via the `className` prop on `TableHead` (this is page-specific styling, not a deviation from the spec — the spec said header dropdown filters and arbitrary header content keep working).
- The page previously used manual `idx % 2 === 1 && "bg-control-bg/50"` striping. The unified `TableBody` provides the same effect by default — the snippet above does **not** pass `striped={false}`. The loading and empty states are also `<TableRow>` children of `<TableBody>` so striping parity is computed including them, but those states are short-lived; the visual diff is acceptable.
- `align-top` on cells is preserved (audit log rows can be tall). `overflow-hidden` is preserved.

- [ ] **Step 4: Run the frontend fixer**

```bash
pnpm --dir frontend fix
```

Expected: imports get sorted; no errors.

- [ ] **Step 5: Type check**

```bash
pnpm --dir frontend type-check
```

Expected: PASS, no errors related to this file. `sortDir` accepts `"asc" | "desc" | undefined`; `sortOrder` here is `"asc" | "desc"`. When `sortKey !== col.key`, `sortActive` is `false` so `sortDir` is ignored by the indicator.

- [ ] **Step 6: Lint check**

```bash
pnpm --dir frontend check
```

Expected: PASS.

- [ ] **Step 7: Manual visual verification**

Open the workspace audit log page (`/audit-log`) and the project audit log page (a project detail's audit log tab) — both consume `AuditLogTable`.

Verify:

- Columns render with their initial widths.
- Drag the resize handle on a resizable column header — it resizes.
- Click the **Created at** column header — sort toggles asc → desc → off, with the unified arrow indicator switching state.
- The export dropdown still opens and exports.
- The advanced search and time range picker still filter results.
- Pagination footer still works.

- [ ] **Step 8: Commit**

```bash
git add frontend/src/react/components/AuditLogTable.tsx
git commit -m "$(cat <<'EOF'
refactor(react): migrate AuditLogTable to unified Table component

Wire column resize and sort through the unified TableHead props
instead of the local renderSortIndicator and ColumnResizeHandle
imports.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Task 4: Migrate InstancesPage

**Files:**
- Modify: `frontend/src/react/pages/settings/InstancesPage.tsx`

This is the largest case but mechanically the same as AuditLogTable: `useColumnWidths` + `<colgroup>` + `style={{ width: totalWidth }}` + `table-fixed` pass through to the unified `Table`. Sort indicator and column resize move to `TableHead` props. Data source expansion stays in-cell (as in the existing code) — no special handling.

- [ ] **Step 1: Update the imports**

In the `lucide-react` import block (currently lines 3–11), drop `ChevronDown` if it is **only** used by the local `renderSortIndicator` and the `SyncDropdown` toggle and the in-cell data source toggle. Check those:

- `SyncDropdown` (line 559) uses `<ChevronDown className="h-3 w-3 ml-1" />` — keep.
- The address cell (line 1213) uses `<ChevronDown className="w-4 h-4" />` — keep.
- The local `renderSortIndicator` uses it — this gets deleted.

So **keep** `ChevronDown` in the import. Same for `ChevronUp` (used for the data source toggle).

Drop the `ColumnResizeHandle` import (currently line 37) — `TableHead` renders the handle internally.

Add the unified `Table` imports. After the existing `Sheet` imports block, add:

```tsx
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/react/components/ui/table";
```

- [ ] **Step 2: Delete the local `renderSortIndicator`**

Delete this constant inside `InstancesPage` (currently lines 1109–1121):

```tsx
const renderSortIndicator = (columnKey: string) => {
  if (sortKey !== columnKey) {
    return <ChevronDown className="size-3 text-control-border" />;
  }
  return (
    <ChevronDown
      className={cn(
        "h-3 w-3 text-accent transition-transform",
        sortOrder === "asc" && "rotate-180"
      )}
    />
  );
};
```

- [ ] **Step 3: Replace the `<table>` block with the unified primitives**

Replace the table block (currently lines 1314–1403, beginning with `<div className="border rounded-sm">` and ending with the closing `</div>` of that wrapper) with:

```tsx
{/* Table */}
<div className="border rounded-sm">
  <div className="overflow-x-auto">
    <Table
      className="table-fixed"
      style={{ width: `${totalWidth}px` }}
    >
      <colgroup>
        {widths.map((w, i) => (
          <col key={columns[i].key} style={{ width: `${w}px` }} />
        ))}
      </colgroup>
      <TableHeader>
        <TableRow>
          {columns.map((col, colIdx) => (
            <TableHead
              key={col.key}
              sortable={col.sortable}
              sortActive={col.sortable && sortKey === (col.sortKey ?? col.key)}
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
            >
              {col.title}
            </TableHead>
          ))}
        </TableRow>
      </TableHeader>
      <TableBody>
        {loading && instances.length === 0 ? (
          <TableRow>
            <TableCell
              colSpan={columns.length}
              className="px-4 py-8 text-center text-control-placeholder"
            >
              <div className="flex items-center justify-center gap-x-2">
                <div className="animate-spin size-4 border-2 border-accent border-t-transparent rounded-full" />
                {t("common.loading")}
              </div>
            </TableCell>
          </TableRow>
        ) : instances.length === 0 ? (
          <TableRow>
            <TableCell
              colSpan={columns.length}
              className="px-4 py-8 text-center text-control-placeholder"
            >
              {t("common.no-data")}
            </TableCell>
          </TableRow>
        ) : (
          instances.map((instance) => (
            <TableRow
              key={instance.name}
              className="cursor-pointer"
              onClick={(e) => handleRowClick(instance, e)}
            >
              {columns.map((col) => (
                <TableCell
                  key={col.key}
                  className={cn("overflow-hidden", col.cellClassName)}
                >
                  {col.render(instance)}
                </TableCell>
              ))}
            </TableRow>
          ))
        )}
      </TableBody>
    </Table>
  </div>
</div>
```

Notes on what changed:
- `<table className="text-sm table-fixed">` becomes `<Table className="table-fixed">` (the unified `Table` already sets `text-sm`).
- The header row's custom `bg-control-bg border-b border-control-border` is dropped — the unified `TableHeader` adds the bottom border, and per the spec we drop the custom header background.
- The local `renderSortIndicator` calls are replaced by `TableHead`'s built-in indicator via the `sortable` / `sortActive` / `sortDir` / `onSort` props.
- `<ColumnResizeHandle ... />` calls inside each `<th>` are removed; they are now rendered by `TableHead` itself when `resizable` is true.
- Per-row striping (`i % 2 === 1 && "bg-control-bg/50"`) is dropped because `TableBody` provides striping by default. Pass `cursor-pointer` to `TableRow` (hover is already provided by the unified component).
- Cells keep their `cellClassName` override pattern (used for the select column to set tighter padding). `align-middle` is dropped because `TableCell` already sets it. `overflow-hidden` is preserved.

- [ ] **Step 4: Run the frontend fixer**

```bash
pnpm --dir frontend fix
```

Expected: imports get sorted; no errors.

- [ ] **Step 5: Type check**

```bash
pnpm --dir frontend type-check
```

Expected: PASS, no errors related to this file. The `InstanceColumn` type's `title: React.ReactNode` is compatible with `TableHead`'s children. `sortable`, `sortActive`, `sortDir`, `onSort`, `resizable`, `onResizeStart` are all optional on `TableHead`.

- [ ] **Step 6: Lint check**

```bash
pnpm --dir frontend check
```

Expected: PASS.

- [ ] **Step 7: Manual visual verification — InstancesPage**

Open `/instance` (the workspace instances list). Verify:

- Header renders with no background fill (intentional — visual normalization).
- Sort indicators on Name and Environment use the unified `ArrowUp`/`ArrowDown`/`ArrowUpDown` icons. Clicking toggles asc → desc → off.
- Drag a resize handle on Name, Environment, Address, or Labels — column resizes.
- Click a row — navigates to instance detail.
- Click the row's checkbox without navigating — selection updates and the batch operations bar appears.
- Find an instance with multiple data sources (or temporarily add one). Click the chevron in the address cell. The cell expands vertically to show all data source `host:port` entries. Collapse — back to a single line.
- Sort while a row is expanded — the row stays in its position relative to the sort, with its expansion preserved (expansion state is keyed on `instance.name`).
- Pagination footer still works.
- Trigger an empty result (e.g. search for a string that matches nothing) — the empty placeholder spans all columns.
- Trigger the loading state (refresh) — the loading spinner spans all columns.

- [ ] **Step 8: Commit**

```bash
git add frontend/src/react/pages/settings/InstancesPage.tsx
git commit -m "$(cat <<'EOF'
refactor(react): migrate InstancesPage table to unified Table component

Wire column resize and sort through the unified TableHead props.
Data source expansion stays in-cell — no Table API change needed.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Final wrap-up

After all four tasks are committed:

- [ ] **Final step: Whole-frontend check**

Run the full frontend pipeline once more to catch any cross-file issues:

```bash
pnpm --dir frontend fix
pnpm --dir frontend check
pnpm --dir frontend type-check
pnpm --dir frontend test
```

Expected: all PASS. If `test` surfaces failures in unrelated areas, document them but do not attempt to fix them as part of this work.

- [ ] **Final step: Confirm no stragglers**

Run a quick grep to confirm there are no remaining raw `<table>` callsites under `frontend/src/react/`:

```bash
grep -rn '<table' frontend/src/react/ --include='*.tsx' | grep -v 'src/react/components/ui/table.tsx'
```

Expected: no results (other than possibly comments or string literals — eyeball them).

If new raw `<table>` callsites appear that were not in the original audit, do **not** silently expand the scope. Stop and report them; the spec covers exactly the four files above.

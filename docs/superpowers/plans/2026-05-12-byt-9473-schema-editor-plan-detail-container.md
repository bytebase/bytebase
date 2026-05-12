# Schema Editor — Plan Detail Container + BYT-9473 Polish Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace the fixed-width drawer hosting the schema editor in plan detail with a maximizable drawer, and bundle the four BYT-9473 polish fixes (column icon, "new table" action, add-column highlight, badge/empty-state polish) into one coherent change.

**Architecture:** Extend the shared `Sheet` primitive with one new width tier (`huge`) and an `actions` slot on `SheetHeader`. `SchemaEditorSheet` owns the maximize state locally — it resets on each open because the body is gated on `{open && ...}`. The four BYT-9473 items are scoped, low-risk swaps that keep the editor internals untouched.

**Tech Stack:** React, Base UI (`@base-ui/react`), Tailwind CSS v4, shadcn-style component patterns, `react-i18next`, `lucide-react`, `class-variance-authority`.

**Spec:** [`docs/superpowers/specs/2026-05-12-byt-9473-schema-editor-plan-detail-container-design.md`](../specs/2026-05-12-byt-9473-schema-editor-plan-detail-container-design.md)

**Linear:** [BYT-9473](https://linear.app/bytebase/issue/BYT-9473/schema-editor-display-not-work-properly)

---

## Task 1: Add `huge` width tier and `actions` slot to `SheetHeader`

**Files:**
- Modify: `frontend/src/react/components/ui/sheet.tsx`

- [ ] **Step 1: Add `huge` width variant**

  Open `frontend/src/react/components/ui/sheet.tsx`. Inside `sheetContentVariants` add a `huge` tier and a one-line comment so the convention is visible.

  Replace the `variants.width` block (currently lines 62-73) with:

  ```ts
  variants: {
    width: {
      narrow: "w-[24rem]",
      panel: "w-[31.25rem]",
      medium: "w-[40rem]",
      standard: "w-[44rem]",
      wide: "w-[52rem]",
      large: "w-[64rem]",
      xlarge: "w-[70rem]",
      // Maximized editor surfaces (e.g. plan-detail schema editor). Leaves a
      // 5vw strip on the left as a visual anchor; clicking the strip closes
      // the sheet like any other scrim click.
      huge: "w-[95vw]",
      workspace: "w-[calc(100vw-8rem)] lg:w-240 max-w-[calc(100vw-8rem)]",
    },
  },
  ```

- [ ] **Step 2: Add `actions` slot to `SheetHeader`**

  Replace the existing `SheetHeader` function (currently lines 116-138) with:

  ```tsx
  // Sticky top region with a bottom border, laid out as a row so the built-in
  // close button sits on the right. Typically contains a `SheetTitle` and an
  // optional `SheetDescription` — both are wrapped in a flex-col for the
  // vertical stack layout while the close button remains flush right.
  // `actions` renders secondary icon-buttons (maximize, settings, etc.)
  // immediately before the close button. Use the slot rather than rolling
  // a bespoke header so dismissal stays consistent across sheets.
  function SheetHeader({
    className,
    children,
    actions,
    ...props
  }: ComponentProps<"div"> & { actions?: React.ReactNode }) {
    const { t } = useTranslation();

    return (
      <div
        className={cn(
          "flex items-start justify-between gap-x-4 border-b border-control-border px-6 py-4",
          className
        )}
        {...props}
      >
        <div className="flex flex-col gap-y-1 min-w-0 flex-1">{children}</div>
        {actions ? (
          <div className="flex items-center gap-x-1 shrink-0">{actions}</div>
        ) : null}
        {/* Built-in close affordance. Callers should not render their own close
            button — Base UI's Close dismisses the Sheet via Root's onOpenChange. */}
        <BaseDialog.Close
          aria-label={t("common.close")}
          className="shrink-0 rounded-xs p-1 text-control hover:bg-control-bg focus:outline-hidden focus-visible:ring-2 focus-visible:ring-accent cursor-pointer"
        >
          <X className="size-4" />
        </BaseDialog.Close>
      </div>
    );
  }
  ```

  Note: `React` is not imported as a value anywhere else in this file. Add the type import at the top:

  ```ts
  import type { ComponentProps, ReactNode } from "react";
  ```

  …and use `ReactNode` instead of `React.ReactNode`:

  ```tsx
  }: ComponentProps<"div"> & { actions?: ReactNode }) {
  ```

- [ ] **Step 3: Type-check**

  Run: `pnpm --dir frontend type-check`
  Expected: PASS (no new errors). The `actions` prop is opt-in; no existing callers need to change.

- [ ] **Step 4: Commit**

  ```bash
  git add frontend/src/react/components/ui/sheet.tsx
  git commit -m "feat(ui): add huge width tier and actions slot to SheetHeader"
  ```

---

## Task 2: Wire maximize toggle into `SchemaEditorSheet`

**Files:**
- Modify: `frontend/src/react/pages/project/plan-detail/components/SchemaEditorSheet.tsx`
- Modify: `frontend/src/locales/en-US.json` (and any other locale files that mirror the same keys)

- [ ] **Step 1: Add `common.maximize` to locales**

  In `frontend/src/locales/en-US.json`, locate the `common` block (around line 200). Add a `"maximize": "Maximize"` entry alongside the existing `"restore": "Restore"` (already at line 377; reuse it for the unmaximized tooltip). Keep keys alphabetized — add immediately after `"manual"` / `"max"` style neighbors.

  If the project uses other locale files under `frontend/src/locales/` (e.g. `zh-CN.json`, `ja.json`), add the same key with the English fallback in each — the existing build typically copies missing keys; if not, use the existing language's word for "Maximize" if obvious, otherwise leave the English value.

  Run: `pnpm --dir frontend fix`
  Expected: PASS (formats JSON).

- [ ] **Step 2: Update `SchemaEditorSheet` imports**

  Open `frontend/src/react/pages/project/plan-detail/components/SchemaEditorSheet.tsx`. Replace the existing `lucide-react` import line (currently `import { Loader2 } from "lucide-react";`) with:

  ```ts
  import { Loader2, Maximize2, Minimize2 } from "lucide-react";
  ```

- [ ] **Step 3: Add maximize state and pass `actions` + `width` to the sheet**

  Replace the existing `SchemaEditorSheet` function (currently lines 37-62) with:

  ```tsx
  export function SchemaEditorSheet({
    open,
    onOpenChange,
    databaseNames,
    project,
    onInsert,
  }: Props) {
    const { t } = useTranslation();
    // Local state — resets on each open because the body below is gated on
    // {open && ...} and remounts when the sheet reopens.
    const [maximized, setMaximized] = useState(false);
    const MaximizeIcon = maximized ? Minimize2 : Maximize2;
    return (
      <Sheet
        open={open}
        onOpenChange={(next) => {
          if (!next) setMaximized(false);
          onOpenChange(next);
        }}
      >
        <SheetContent width={maximized ? "huge" : "xlarge"} className="flex flex-col">
          <SheetHeader
            actions={
              <button
                type="button"
                aria-label={
                  maximized ? t("common.restore") : t("common.maximize")
                }
                title={maximized ? t("common.restore") : t("common.maximize")}
                onClick={() => setMaximized((v) => !v)}
                className="rounded-xs p-1 text-control hover:bg-control-bg focus:outline-hidden focus-visible:ring-2 focus-visible:ring-accent cursor-pointer"
              >
                <MaximizeIcon className="size-4" />
              </button>
            }
          >
            <SheetTitle>{t("schema-editor.self")}</SheetTitle>
          </SheetHeader>
          {open && (
            <SchemaEditorSheetBody
              databaseNames={databaseNames}
              project={project}
              onInsert={onInsert}
              onCancel={() => onOpenChange(false)}
            />
          )}
        </SheetContent>
      </Sheet>
    );
  }
  ```

  Note: `useState` must be imported — the existing `import { useCallback, useEffect, useRef, useState } from "react";` line already has it.

- [ ] **Step 4: Type-check and lint**

  Run: `pnpm --dir frontend type-check`
  Expected: PASS.

  Run: `pnpm --dir frontend check`
  Expected: PASS. The layering scanner (`check-react-layering.mjs`) should not flag this change — no new portal targets, no raw `z-index`, no body portals.

- [ ] **Step 5: Manual sanity check**

  Run: `pnpm --dir frontend dev`
  Open a plan in the browser. Click "Schema editor" in the statement section toolbar.
  Expected:
  - Drawer opens at the same width as today (`xlarge`).
  - The ⤢ button appears immediately left of the X in the header. Tooltip reads "Maximize".
  - Click ⤢ — drawer expands to ~95% viewport, leaving a thin strip on the left. Icon flips to "Minimize" and tooltip reads "Restore".
  - Click the icon again — drawer collapses back to `xlarge`.
  - Click the left strip — sheet closes (matches scrim behavior).
  - Close + reopen — drawer is back at `xlarge` (state reset).

- [ ] **Step 6: Commit**

  ```bash
  git add frontend/src/react/pages/project/plan-detail/components/SchemaEditorSheet.tsx \
          frontend/src/locales/
  git commit -m "feat(react): maximize toggle on plan-detail schema editor sheet"
  ```

---

## Task 3: Shared `ColumnIcon` — fix the "C" mismatch (BYT-9473 item 1)

**Files:**
- Create: `frontend/src/react/components/schema/icons.tsx`
- Modify: `frontend/src/react/components/SchemaEditorLite/Aside/AsideTree.tsx`
- Modify: `frontend/src/react/components/sql-editor/SchemaPane/TreeNode/icons.tsx` (re-export from shared module)

- [ ] **Step 1: Create the shared icon module**

  Create `frontend/src/react/components/schema/icons.tsx` with:

  ```tsx
  import { cn } from "@/react/lib/utils";

  interface IconProps {
    className?: string;
  }

  /**
   * Single-column variant of `lucide:columns-3` (the second internal gap line
   * removed). Used in both Schema Editor and SQL Editor tree views so the two
   * surfaces stay visually identical.
   */
  export function ColumnIcon({ className }: IconProps) {
    return (
      <svg
        xmlns="http://www.w3.org/2000/svg"
        width="24"
        height="24"
        viewBox="0 0 24 24"
        fill="none"
        stroke="currentColor"
        strokeWidth="2"
        strokeLinecap="round"
        strokeLinejoin="round"
        className={cn("size-4 text-control-light", className)}
      >
        <rect width="18" height="18" x="3" y="3" rx="2" />
        <path d="M9 3v18" />
      </svg>
    );
  }
  ```

  Note: this uses the semantic `text-control-light` token, not `text-gray-500`. AGENTS.md forbids raw colors in React UI.

- [ ] **Step 2: Re-export from the SQL Editor's icons file**

  Open `frontend/src/react/components/sql-editor/SchemaPane/TreeNode/icons.tsx`. Replace the existing `ColumnIcon` definition (currently lines 129-151) with a re-export so the two surfaces share one source of truth:

  ```tsx
  // ColumnIcon is shared with SchemaEditorLite; both surfaces must render
  // identical icons so users don't see drift across editors.
  export { ColumnIcon } from "@/react/components/schema/icons";
  ```

  Remove the now-unused `cn` and `baseSize` references if no other icon in the file uses them. (`Check` is still in use for `CheckConstraintIcon` above — leave that.)

- [ ] **Step 3: Swap `<div>C</div>` for `<ColumnIcon />` in `AsideTree`**

  In `frontend/src/react/components/SchemaEditorLite/Aside/AsideTree.tsx`, add the import near the other component imports at the top:

  ```ts
  import { ColumnIcon } from "@/react/components/schema/icons";
  ```

  Then replace the `case "column"` branch in `NodeIcon` (currently lines 472-477):

  ```tsx
  case "column":
    return <ColumnIcon className={cls} />;
  ```

  `cls` is the existing `"size-4 shrink-0"` constant a few lines above — keep it so sizing stays consistent with the other Lucide icons in this switch.

- [ ] **Step 4: Type-check and lint**

  Run: `pnpm --dir frontend type-check`
  Expected: PASS.

  Run: `pnpm --dir frontend check`
  Expected: PASS.

- [ ] **Step 5: Manual sanity check**

  Reopen the schema editor in the dev server. Expand a table. Expected:
  - Each column row now shows the same single-column SVG used by SQL Editor (no more bold "C" text).
  - Status colors on the row label remain (`text-success` for created, etc.) — they apply to the label, not the icon.
  - Open the SQL Editor's schema pane in parallel and confirm column icons look identical.

- [ ] **Step 6: Commit**

  ```bash
  git add frontend/src/react/components/schema/icons.tsx \
          frontend/src/react/components/sql-editor/SchemaPane/TreeNode/icons.tsx \
          frontend/src/react/components/SchemaEditorLite/Aside/AsideTree.tsx
  git commit -m "fix(schema-editor): use shared ColumnIcon to match SQL Editor"
  ```

---

## Task 4: Convert tree status text into a Badge (BYT-9473 polish)

**Files:**
- Modify: `frontend/src/react/components/SchemaEditorLite/Aside/AsideTree.tsx`

- [ ] **Step 1: Add `Badge` import**

  In `AsideTree.tsx`, add:

  ```ts
  import { Badge } from "@/react/components/ui/badge";
  ```

- [ ] **Step 2: Add a small `StatusBadge` helper**

  Just below the existing `statusClassName` helper (around line 512), add:

  ```tsx
  // Single-letter badge to mark created / updated / dropped entries in the
  // tree. Sits to the right of the label so it doesn't crowd the icon column
  // and reads at a glance — text color alone was too quiet (BYT-9473).
  function StatusBadge({ status }: { status: EditStatus }) {
    if (status === "normal") return null;
    const variant =
      status === "created"
        ? "success"
        : status === "updated"
          ? "warning"
          : "error";
    const letter =
      status === "created" ? "+" : status === "updated" ? "~" : "−";
    return (
      <Badge variant={variant} className="ml-1 h-4 px-1 text-[10px] leading-none">
        {letter}
      </Badge>
    );
  }
  ```

  Note: confirm `Badge` exposes `success`, `warning`, `error` variants by opening `frontend/src/react/components/ui/badge.tsx`. If it only exposes `default`, `secondary`, `destructive`, `outline`, use semantic class overrides via `className` instead — e.g. `<Badge className="ml-1 bg-success/10 text-success ...">`. Pick whichever route lands without modifying `badge.tsx`.

- [ ] **Step 3: Render the badge in `NodeRenderer`**

  Replace the label span in `NodeRenderer` (currently lines 577-579):

  ```tsx
  <span className={cn("truncate", statusClassName(status))}>
    {treeNode.label || "(empty)"}
  </span>
  <StatusBadge status={status} />
  ```

  Keep `statusClassName(status)` on the label so the existing strike-through for `dropped` survives — the badge is additive, not a replacement.

- [ ] **Step 4: Type-check and lint**

  Run: `pnpm --dir frontend type-check`
  Expected: PASS.

  Run: `pnpm --dir frontend check`
  Expected: PASS.

- [ ] **Step 5: Manual sanity check**

  In the dev server, make any schema edit (add a table, drop a column). Expected:
  - Created entries show a `+` badge in the tree, updated `~`, dropped `−`.
  - Dropped entries still show the strike-through on the label.
  - Normal entries show no badge (no visual change vs today).

- [ ] **Step 6: Commit**

  ```bash
  git add frontend/src/react/components/SchemaEditorLite/Aside/AsideTree.tsx
  git commit -m "feat(schema-editor): badge created/updated/dropped tree entries"
  ```

---

## Task 5: Convert `TableNameDialog` to inline `TableNamePopover` (BYT-9473 item 2)

**Files:**
- Create: `frontend/src/react/components/SchemaEditorLite/Modals/TableNamePopover.tsx`
- Modify: `frontend/src/react/components/SchemaEditorLite/Aside/AsideTree.tsx` (swap call site)
- Delete: `frontend/src/react/components/SchemaEditorLite/Modals/TableNameDialog.tsx` (last task step, after wiring proves out)

- [ ] **Step 1: Investigate the failure (5 minutes)**

  Before writing code, reproduce the BYT-9473 "new table not work" symptom locally. Open the schema editor → right-click a schema → "New table". Note exactly what fails (modal doesn't appear, input doesn't focus, Enter does nothing, Esc closes the sheet instead of the dialog, etc.). Record observations in the PR description later.

  This step is information-gathering — no code change. It confirms the root cause matches the spec's hypothesis (nested Base UI Dialog inside Sheet in the same `overlay` layer family).

- [ ] **Step 2: Create the popover component**

  Create `frontend/src/react/components/SchemaEditorLite/Modals/TableNamePopover.tsx` with:

  ```tsx
  import { create } from "@bufbuild/protobuf";
  import { useCallback, useState } from "react";
  import { useTranslation } from "react-i18next";
  import { Button } from "@/react/components/ui/button";
  import { Input } from "@/react/components/ui/input";
  import {
    Popover,
    PopoverContent,
  } from "@/react/components/ui/popover";
  import { Engine } from "@/types/proto-es/v1/common_pb";
  import type {
    Database,
    DatabaseMetadata,
    SchemaMetadata,
    TableMetadata,
  } from "@/types/proto-es/v1/database_service_pb";
  import {
    ColumnMetadataSchema,
    TableMetadataSchema,
  } from "@/types/proto-es/v1/database_service_pb";
  import { getDatabaseEngine } from "@/utils";
  import { useSchemaEditorContext } from "../context";
  import { upsertColumnPrimaryKey } from "../core/edit";
  import { markUUID } from "../Panels/common";

  interface Props {
    open: boolean;
    onClose: () => void;
    /** Screen-space coordinates of the click that triggered the popover.
     *  The popover anchors to a virtual rect at this point so we don't depend
     *  on a DOM element that may have unmounted (e.g. the context menu item).
     */
    anchorPoint: { x: number; y: number };
    db: Database;
    database: DatabaseMetadata;
    schema: SchemaMetadata;
    table?: TableMetadata;
  }

  const TABLE_NAME_REGEX = /^\S[\S ]*\S?$/;

  export function TableNamePopover({
    open,
    onClose,
    anchorPoint,
    db,
    database,
    schema,
    table,
  }: Props) {
    const { t } = useTranslation();
    const { tabs, editStatus, rebuildTree, rebuildEditStatus, scrollStatus } =
      useSchemaEditorContext();
    const engine = getDatabaseEngine(db);

    const [tableName, setTableName] = useState(table?.name ?? "");
    const isCreateMode = !table;

    const isDuplicate =
      tableName !== table?.name &&
      schema.tables.some((tt) => tt.name === tableName);

    const isValid =
      tableName.length > 0 && TABLE_NAME_REGEX.test(tableName) && !isDuplicate;

    const handleConfirm = useCallback(() => {
      if (!isValid) return;

      if (isCreateMode) {
        const newTable = create(TableMetadataSchema, {
          name: tableName,
          columns: [],
          indexes: [],
          foreignKeys: [],
          partitions: [],
          comment: "",
        });
        schema.tables.push(newTable);
        editStatus.markEditStatus(db, { schema, table: newTable }, "created");

        const defaultType = engine === Engine.POSTGRES ? "integer" : "int";
        const idColumn = create(ColumnMetadataSchema, {
          name: "id",
          type: defaultType,
          nullable: false,
          hasDefault: false,
          default: "",
          comment: "",
        });
        markUUID(idColumn);
        newTable.columns.push(idColumn);
        editStatus.markEditStatus(
          db,
          { schema, table: newTable, column: idColumn },
          "created"
        );
        upsertColumnPrimaryKey(engine, newTable, "id");

        tabs.addTab({
          type: "table",
          database: db,
          metadata: { database, schema, table: newTable },
        });
        scrollStatus.queuePendingScrollToTable({
          db,
          metadata: { database, schema, table: newTable },
        });
        rebuildTree(false);
      } else {
        table.name = tableName;
        rebuildEditStatus(["tree"]);
      }

      onClose();
    }, [
      isValid,
      isCreateMode,
      tableName,
      schema,
      editStatus,
      db,
      database,
      engine,
      tabs,
      scrollStatus,
      rebuildTree,
      rebuildEditStatus,
      table,
      onClose,
    ]);

    // Virtual anchor at the click point. Base UI's Positioner accepts an
    // object with getBoundingClientRect(); we expose a 1×1 rect at (x, y).
    const anchor = {
      getBoundingClientRect: () =>
        ({
          width: 0,
          height: 0,
          x: anchorPoint.x,
          y: anchorPoint.y,
          top: anchorPoint.y,
          left: anchorPoint.x,
          right: anchorPoint.x,
          bottom: anchorPoint.y,
          toJSON() {
            return this;
          },
        }) as DOMRect,
    };

    return (
      <Popover open={open} onOpenChange={(next) => !next && onClose()}>
        <PopoverContent
          anchor={anchor}
          side="bottom"
          align="start"
          className="w-72 p-3"
        >
          <div className="flex flex-col gap-y-3">
            <div className="text-sm font-medium text-control">
              {isCreateMode
                ? t("schema-editor.actions.create-table")
                : t("schema-editor.actions.rename-table")}
            </div>
            <Input
              value={tableName}
              placeholder={t("schema-editor.table.name-placeholder")}
              autoFocus
              onChange={(e) => setTableName(e.target.value)}
              onKeyDown={(e) => {
                if (e.key === "Enter") handleConfirm();
                if (e.key === "Escape") onClose();
              }}
            />
            {isDuplicate && (
              <p className="text-xs text-error">
                {t("schema-editor.table.duplicate-name")}
              </p>
            )}
            <div className="flex items-center justify-end gap-x-2">
              <Button variant="outline" size="sm" onClick={onClose}>
                {t("common.cancel")}
              </Button>
              <Button size="sm" disabled={!isValid} onClick={handleConfirm}>
                {isCreateMode ? t("common.create") : t("common.save")}
              </Button>
            </div>
          </div>
        </PopoverContent>
      </Popover>
    );
  }
  ```

- [ ] **Step 3: Swap the call site in `AsideTree`**

  In `AsideTree.tsx`:

  - Replace the import:
    ```ts
    import { TableNamePopover } from "../Modals/TableNamePopover";
    ```
    (was `import { TableNameDialog } from "../Modals/TableNameDialog";`)

  - The existing `tableNameModalCtx` state captures `{ db, database, schema, table? }`. Extend the context to also carry the click coordinates. Locate the right-click handler that opens the menu (search for `setMenuState` / `handleMenuSelect`). Where the "create-table" / "rename-table" branch sets `tableNameModalCtx`, also include `anchorPoint: { x: menuState.x, y: menuState.y }`. Update the `tableNameModalCtx` type accordingly:

    ```ts
    type TableNameModalCtx = {
      db: Database;
      database: DatabaseMetadata;
      schema: SchemaMetadata;
      table?: TableMetadata;
      anchorPoint: { x: number; y: number };
    };
    ```

  - Replace the `<TableNameDialog ... />` block (currently lines 410-419) with:

    ```tsx
    {tableNameModalCtx && (
      <TableNamePopover
        open
        onClose={() => setTableNameModalCtx(null)}
        anchorPoint={tableNameModalCtx.anchorPoint}
        db={tableNameModalCtx.db}
        database={tableNameModalCtx.database}
        schema={tableNameModalCtx.schema}
        table={tableNameModalCtx.table}
      />
    )}
    ```

- [ ] **Step 4: Delete the old dialog**

  Delete `frontend/src/react/components/SchemaEditorLite/Modals/TableNameDialog.tsx`.

  ```bash
  git rm frontend/src/react/components/SchemaEditorLite/Modals/TableNameDialog.tsx
  ```

- [ ] **Step 5: Type-check and lint**

  Run: `pnpm --dir frontend type-check`
  Expected: PASS. If the layering scanner complains about the new file, double-check that `PopoverContent` is the only portal target — it already mounts into `getLayerRoot("overlay")` via the shared primitive.

  Run: `pnpm --dir frontend check`
  Expected: PASS.

- [ ] **Step 6: Manual sanity check**

  Reopen the schema editor:
  - Right-click a schema → "New table". Popover appears at the cursor; name input is autofocused; typing + Enter creates the table, opens its tab, scrolls to it.
  - Right-click a table → "Rename". Popover appears prefilled with the current name; edit + Enter renames; the tree refreshes.
  - Esc closes the popover but leaves the sheet open.
  - Clicking outside the popover (but inside the sheet) dismisses the popover only — sheet stays.

- [ ] **Step 7: Commit**

  ```bash
  git add frontend/src/react/components/SchemaEditorLite/Modals/TableNamePopover.tsx \
          frontend/src/react/components/SchemaEditorLite/Aside/AsideTree.tsx
  git commit -m "fix(schema-editor): convert table-name dialog to inline popover"
  ```

  Note: the `git rm` from Step 4 stages the deletion automatically; `git add` here covers the new file and the call-site change.

---

## Task 6: Highlight new column on add (BYT-9473 item 3)

**Files:**
- Modify: `frontend/src/assets/css/tailwind.css`
- Modify: `frontend/src/react/components/SchemaEditorLite/Panels/TableEditor.tsx`
- Modify: `frontend/src/react/components/SchemaEditorLite/Panels/TableColumnEditor.tsx` (apply the data attribute and focus the input — confirm exact filename when opening)

- [ ] **Step 1: Add the row-flash keyframe**

  Open `frontend/src/assets/css/tailwind.css`. Locate the `@theme` / `@utility` sections (search for `@utility` to find existing ones). Add:

  ```css
  @keyframes row-flash {
    0% {
      background-color: color-mix(in oklch, var(--color-success) 18%, transparent);
    }
    100% {
      background-color: transparent;
    }
  }

  @utility animate-row-flash {
    animation: row-flash 1.2s ease-out forwards;
  }
  ```

  Use `color-mix` against `--color-success` so the flash respects the theme; `bg-success/10` directly does not apply through animation in Tailwind v4 without a custom keyframe.

- [ ] **Step 2: Mark newly-added columns in `handleAddColumn`**

  In `frontend/src/react/components/SchemaEditorLite/Panels/TableEditor.tsx`, replace `handleAddColumn` (currently lines 69-86) with:

  ```tsx
  const handleAddColumn = useCallback(() => {
    const column = create(ColumnMetadataSchema, {
      name: "",
      type: "",
      nullable: true,
      hasDefault: false,
      default: "",
      comment: "",
    });
    markUUID(column);
    table.columns.push(column);
    editStatus.markEditStatus(db, { schema, table, column }, "created");
    rebuildTree(false);
    scrollStatus.queuePendingScrollToColumn({
      db,
      metadata: { database, schema, table, column },
    });
    // Surface the new row visually. The flash + autofocus run in
    // TableColumnEditor when it sees data-just-added on the row.
    scrollStatus.queuePendingFocusForColumn?.({
      db,
      metadata: { database, schema, table, column },
    });
  }, [db, database, schema, table, editStatus, rebuildTree, scrollStatus]);
  ```

  Note: if `queuePendingFocusForColumn` doesn't yet exist on `scrollStatus`, locate `scrollStatus` (likely in `frontend/src/react/components/SchemaEditorLite/context.ts` or a hook nearby) and add a small queue parallel to `queuePendingScrollToColumn`:

  ```ts
  // Inside the scrollStatus state object:
  pendingFocusColumnKey: null as string | null,

  queuePendingFocusForColumn(target: ColumnTarget) {
    this.pendingFocusColumnKey = columnKey(target);
  },

  consumePendingFocusForColumn(target: ColumnTarget): boolean {
    if (this.pendingFocusColumnKey !== columnKey(target)) return false;
    this.pendingFocusColumnKey = null;
    return true;
  },
  ```

  Use the existing `columnKey` helper that `queuePendingScrollToColumn` already uses (search the file for it). If the existing scrollStatus is a Zustand-like store with `set`, follow the same pattern.

- [ ] **Step 3: Consume the flag in the column editor row**

  Open `frontend/src/react/components/SchemaEditorLite/Panels/TableColumnEditor.tsx` (verify exact filename — referenced via `import { TableColumnEditor } from "./TableColumnEditor";` in `TableEditor.tsx`). For each rendered column row:

  - Add a `useRef<HTMLInputElement>(null)` for the Name input on each row (if not already present).
  - On mount of a row whose column matches the pending-focus key, call `consumePendingFocusForColumn`; if it returns `true`, focus the name input, set `data-just-added="true"` on the row, and add `animate-row-flash` to the row's `className`.
  - Remove the attribute / class on `onAnimationEnd`.

  Minimal sketch (drop into the row component):

  ```tsx
  const rowRef = useRef<HTMLTableRowElement>(null);
  const nameInputRef = useRef<HTMLInputElement>(null);
  const [justAdded, setJustAdded] = useState(false);

  useEffect(() => {
    if (scrollStatus.consumePendingFocusForColumn({ db, metadata: { database, schema, table, column } })) {
      setJustAdded(true);
      // Defer focus to the next frame so the row is in the DOM.
      requestAnimationFrame(() => nameInputRef.current?.focus());
    }
    // We only want this on mount per row instance.
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  return (
    <tr
      ref={rowRef}
      data-just-added={justAdded || undefined}
      className={cn(rowClassName, justAdded && "animate-row-flash")}
      onAnimationEnd={() => setJustAdded(false)}
    >
      {/* …existing cells, with nameInputRef passed to the Name <Input /> */}
    </tr>
  );
  ```

  If `TableColumnEditor` renders rows via a virtualized list, apply the same pattern at whichever element actually represents a row in the DOM.

- [ ] **Step 4: Type-check and lint**

  Run: `pnpm --dir frontend type-check`
  Expected: PASS.

  Run: `pnpm --dir frontend check`
  Expected: PASS.

- [ ] **Step 5: Manual sanity check**

  In the dev server, open a table and click "Add column". Expected:
  - The row appears, the editor scrolls to it (existing behavior).
  - The row pulses a faint green for ~1.2s.
  - The Name input is focused — you can immediately type the column name.
  - Adding a second column does the same; previously-added rows do not re-flash.

- [ ] **Step 6: Commit**

  ```bash
  git add frontend/src/assets/css/tailwind.css \
          frontend/src/react/components/SchemaEditorLite/Panels/TableEditor.tsx \
          frontend/src/react/components/SchemaEditorLite/Panels/TableColumnEditor.tsx \
          frontend/src/react/components/SchemaEditorLite/context.ts
  git commit -m "feat(schema-editor): flash and focus newly added columns"
  ```

---

## Task 7: Toolbar density and empty-state polish

**Files:**
- Modify: `frontend/src/react/pages/project/plan-detail/components/SchemaEditorSheet.tsx`
- Modify: `frontend/src/react/components/SchemaEditorLite/SchemaEditorLite.tsx` (or wherever the editor renders when no tables exist — verify on opening)

- [ ] **Step 1: Tighten the body spacing in `SchemaEditorSheet`**

  In `frontend/src/react/pages/project/plan-detail/components/SchemaEditorSheet.tsx`, locate the body wrapper (currently `<div className="flex flex-1 flex-col gap-y-3 overflow-hidden px-4 pb-2">` on line 212). Change:

  ```tsx
  <div className="flex flex-1 flex-col gap-y-2 overflow-hidden px-4 pb-2">
  ```

  (`gap-y-3` → `gap-y-2`). Align the combobox row above the editor by giving the combobox `<div>` a fixed height: `<div className="flex h-9 items-center gap-x-2">` — this keeps the baseline stable whether or not the row is present.

- [ ] **Step 2: Add an empty-state for the editor panel**

  Open `frontend/src/react/components/SchemaEditorLite/SchemaEditorLite.tsx` and locate where the right panel (`EditorPanel`) renders. Find the path where the selected database has no tables and no views yet — likely a render that returns nothing or a bare empty `<div>`. Replace whatever is rendered in that branch with:

  ```tsx
  <div className="flex h-full items-center justify-center px-6">
    <div className="flex flex-col items-center gap-y-2 text-center">
      <Table2 className="size-8 text-control-light" />
      <p className="text-sm text-control">
        {t("schema-editor.empty-state.no-tables")}
      </p>
      <p className="text-xs text-control-light">
        {t("schema-editor.empty-state.hint")}
      </p>
    </div>
  </div>
  ```

  Add the corresponding locale entries to `frontend/src/locales/en-US.json` under the existing `"schema-editor"` block (the block starts around line 1773):

  ```json
  "empty-state": {
    "no-tables": "No objects yet",
    "hint": "Right-click a schema in the tree to create your first table."
  }
  ```

  Import `Table2` from `lucide-react` and `useTranslation` from `react-i18next` if not already in this file.

- [ ] **Step 3: Type-check and lint**

  Run: `pnpm --dir frontend type-check`
  Expected: PASS.

  Run: `pnpm --dir frontend check`
  Expected: PASS.

- [ ] **Step 4: Manual sanity check**

  - Open a plan whose target database has at least one table — confirm there's no visual regression in the editor body; the combobox row should not jitter when switching databases.
  - Open a plan whose target database has no tables yet — confirm the empty-state appears with the correct icon, title, and hint.

- [ ] **Step 5: Commit**

  ```bash
  git add frontend/src/react/pages/project/plan-detail/components/SchemaEditorSheet.tsx \
          frontend/src/react/components/SchemaEditorLite/SchemaEditorLite.tsx \
          frontend/src/locales/
  git commit -m "feat(schema-editor): tighten toolbar spacing and add empty state"
  ```

---

## Task 8: Final verification

**Files:** none — verification only.

- [ ] **Step 1: Run the full frontend check suite**

  ```bash
  pnpm --dir frontend fix
  pnpm --dir frontend check
  pnpm --dir frontend type-check
  pnpm --dir frontend test
  ```

  Expected: all PASS. `fix` may auto-format a few files; if so, amend the most recent task's commit:

  ```bash
  git add -u
  git commit --amend --no-edit
  ```

- [ ] **Step 2: End-to-end manual walkthrough**

  In the dev server, walk the full BYT-9473 acceptance set in a single session:

  1. Open a plan, open the schema editor → drawer is `xlarge` width.
  2. Click ⤢ → drawer expands to ~95vw with the left strip visible. ⤢ again → back to `xlarge`.
  3. Click the left strip → sheet closes. Reopen → back at `xlarge` (state reset).
  4. Tree shows the new column SVG icon (matches SQL Editor); created/updated/dropped entries show their badges.
  5. Right-click a schema → "New table" → popover at cursor → name → Enter → table created, tab opened, tree updated.
  6. Click "Add column" → row scrolls into view, flashes green for ~1.2s, Name input is focused.
  7. Switch to a database with no tables → empty-state appears with hint.
  8. Esc closes the sheet from both `xlarge` and `huge` modes.

- [ ] **Step 3: Compare against the pre-PR checklist**

  Open `docs/pre-pr-checklist.md` and run through it. Notable items for this change:
  - No breaking changes (UI only).
  - No composite-PK queries touched.
  - No backend changes.
  - i18n: all new strings live in `frontend/src/locales/`.
  - SonarCloud properties: unchanged.

- [ ] **Step 4: Push and open PR**

  ```bash
  git push -u origin steven/byt-9473-schema-editor-display-not-work-properly
  gh pr create --title "fix(schema-editor): maximize toggle + BYT-9473 polish" --body "$(cat <<'EOF'
  ## Summary
  - Plan-detail schema editor drawer gains a maximize toggle (xlarge ↔ 95vw)
  - Replaces the bold "C" column icon with the shared SQL Editor `ColumnIcon`
  - Converts the nested "New table" dialog to an inline popover (fixes nested-modal focus issue)
  - Flashes + focuses newly added column rows so they're not silently appended
  - Adds Badge markers + empty-state polish in the schema editor tree

  Closes BYT-9473.

  ## Test plan
  - [ ] Drawer opens at xlarge; ⤢ toggles to 95vw; resets on each open
  - [ ] Strip click closes the sheet
  - [ ] Tree column rows render the shared `ColumnIcon`
  - [ ] "New table" popover works from the schema context menu
  - [ ] "Add column" — row scrolls in, flashes, Name input is focused
  - [ ] Empty-state appears for a database with no tables
  - [ ] `pnpm --dir frontend type-check && pnpm --dir frontend check && pnpm --dir frontend test` all pass

  🤖 Generated with [Claude Code](https://claude.com/claude-code)
  EOF
  )"
  ```

---

## Notes for the implementing engineer

- **TDD posture for this plan:** the spec deliberately calls for no new automated tests (the schema editor has none today, and adding them is out of scope). Verification is via type-check, lint, the layering scanner, and the manual walkthrough at the end of each task and in Task 8. If a step surfaces a unit-testable invariant cheaply (e.g. the maximize state reset), feel free to add a test in `frontend/src/react/components/ui/sheet.test.tsx`-style — but don't block the plan on it.
- **Layering policy:** every overlay surface in this plan portals into the shared `overlay` family via the existing primitives (`Sheet`, `Popover`). Do not introduce raw `z-index`, raw body portals, or new layer roots. Run `pnpm --dir frontend check` after each task to keep the layering scanner happy.
- **Locale files:** the project usually mirrors keys across all locales under `frontend/src/locales/`. If a `pnpm` build fails because a key is missing from a non-English locale, copy the English fallback into the missing file rather than silently dropping the key.
- **Worktree:** if executing in an isolated worktree, it should have been created via `superpowers:using-git-worktrees`. Otherwise work in the `steven/byt-9473-schema-editor-display-not-work-properly` branch already named on the Linear issue.

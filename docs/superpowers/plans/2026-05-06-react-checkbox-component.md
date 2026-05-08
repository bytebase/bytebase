# React Checkbox Component Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add a unified `Checkbox` UI primitive at `frontend/src/react/components/ui/checkbox.tsx` and migrate the ~16 components that currently manipulate `indeterminate` imperatively via ref callbacks or paired row checkboxes.

**Architecture:** Wrap `@base-ui/react/checkbox` in a styled component matching the existing `Switch` visual language. Expose `checked: boolean | "indeterminate"` so tri-state is declarative. Per-file migrations replace `<input type="checkbox" ...>` with `<Checkbox ...>` and delete `useRef`/`useEffect` indeterminate plumbing.

**Tech Stack:** React, Base UI (`@base-ui/react/checkbox`), Tailwind CSS v4 (semantic tokens), `class-variance-authority` style patterns (no cva needed for this component — only one orthogonal prop), `cn()` from `@/react/lib/utils`, `lucide-react` (Check, Minus icons), Vitest for unit tests.

**Spec:** [`docs/superpowers/specs/2026-05-06-react-checkbox-component-design.md`](../specs/2026-05-06-react-checkbox-component-design.md)

---

## Task 1: Build the Checkbox component (TDD)

**Files:**
- Create: `frontend/src/react/components/ui/checkbox.tsx`
- Create: `frontend/src/react/components/ui/checkbox.test.tsx`

- [ ] **Step 1: Write the failing test file**

Write `frontend/src/react/components/ui/checkbox.test.tsx`:

```tsx
import { fireEvent, render, screen } from "@testing-library/react";
import { useState } from "react";
import { describe, expect, test, vi } from "vitest";
import { Checkbox } from "./checkbox";

describe("Checkbox", () => {
  test("renders unchecked by default", () => {
    render(<Checkbox checked={false} aria-label="cb" />);
    const cb = screen.getByRole("checkbox");
    expect(cb).toHaveAttribute("aria-checked", "false");
  });

  test("renders checked", () => {
    render(<Checkbox checked={true} aria-label="cb" />);
    expect(screen.getByRole("checkbox")).toHaveAttribute(
      "aria-checked",
      "true"
    );
  });

  test("renders indeterminate", () => {
    render(<Checkbox checked="indeterminate" aria-label="cb" />);
    expect(screen.getByRole("checkbox")).toHaveAttribute(
      "aria-checked",
      "mixed"
    );
  });

  test("click on unchecked emits true", () => {
    const onCheckedChange = vi.fn();
    render(
      <Checkbox
        checked={false}
        onCheckedChange={onCheckedChange}
        aria-label="cb"
      />
    );
    fireEvent.click(screen.getByRole("checkbox"));
    expect(onCheckedChange).toHaveBeenCalledWith(true);
  });

  test("click on checked emits false", () => {
    const onCheckedChange = vi.fn();
    render(
      <Checkbox
        checked={true}
        onCheckedChange={onCheckedChange}
        aria-label="cb"
      />
    );
    fireEvent.click(screen.getByRole("checkbox"));
    expect(onCheckedChange).toHaveBeenCalledWith(false);
  });

  test("click on indeterminate emits true", () => {
    const onCheckedChange = vi.fn();
    render(
      <Checkbox
        checked="indeterminate"
        onCheckedChange={onCheckedChange}
        aria-label="cb"
      />
    );
    fireEvent.click(screen.getByRole("checkbox"));
    expect(onCheckedChange).toHaveBeenCalledWith(true);
  });

  test("disabled blocks onCheckedChange", () => {
    const onCheckedChange = vi.fn();
    render(
      <Checkbox
        checked={false}
        disabled
        onCheckedChange={onCheckedChange}
        aria-label="cb"
      />
    );
    fireEvent.click(screen.getByRole("checkbox"));
    expect(onCheckedChange).not.toHaveBeenCalled();
  });

  test("size md renders size-4 class", () => {
    render(<Checkbox checked={false} aria-label="cb" />);
    expect(screen.getByRole("checkbox").className).toContain("size-4");
  });

  test("size sm renders size-3.5 class", () => {
    render(<Checkbox checked={false} size="sm" aria-label="cb" />);
    expect(screen.getByRole("checkbox").className).toContain("size-3.5");
  });

  test("space key toggles", () => {
    function Harness() {
      const [v, setV] = useState(false);
      return <Checkbox checked={v} onCheckedChange={setV} aria-label="cb" />;
    }
    render(<Harness />);
    const cb = screen.getByRole("checkbox");
    cb.focus();
    fireEvent.keyDown(cb, { key: " ", code: "Space" });
    fireEvent.keyUp(cb, { key: " ", code: "Space" });
    expect(cb).toHaveAttribute("aria-checked", "true");
  });
});
```

- [ ] **Step 2: Run test, verify it fails**

Run: `pnpm --dir frontend test src/react/components/ui/checkbox.test.tsx`

Expected: FAIL — `Cannot find module './checkbox'` or "Checkbox is not defined".

- [ ] **Step 3: Implement the Checkbox component**

Write `frontend/src/react/components/ui/checkbox.tsx`:

```tsx
import { Checkbox as BaseCheckbox } from "@base-ui/react/checkbox";
import { Check, Minus } from "lucide-react";
import { cn } from "@/react/lib/utils";

type CheckboxSize = "sm" | "md";

const ROOT_SIZE: Record<CheckboxSize, string> = {
  sm: "size-3.5",
  md: "size-4",
};

const ICON_SIZE: Record<CheckboxSize, string> = {
  sm: "size-2.5",
  md: "size-3",
};

interface CheckboxProps {
  checked: boolean | "indeterminate";
  onCheckedChange?: (checked: boolean) => void;
  onClick?: React.MouseEventHandler<HTMLButtonElement>;
  disabled?: boolean;
  size?: CheckboxSize;
  className?: string;
  id?: string;
  name?: string;
  "aria-label"?: string;
}

function Checkbox({
  checked,
  onCheckedChange,
  onClick,
  disabled,
  size = "md",
  className,
  id,
  name,
  "aria-label": ariaLabel,
}: CheckboxProps) {
  const baseChecked = checked === "indeterminate" ? false : checked;
  const indeterminate = checked === "indeterminate";

  return (
    <BaseCheckbox.Root
      checked={baseChecked}
      indeterminate={indeterminate}
      onCheckedChange={(value) => onCheckedChange?.(value)}
      onClick={onClick}
      disabled={disabled}
      id={id}
      name={name}
      aria-label={ariaLabel}
      className={cn(
        "inline-flex shrink-0 items-center justify-center rounded-sm border bg-background transition-colors",
        ROOT_SIZE[size],
        "border-control-border hover:border-accent/60",
        "data-[checked]:bg-accent data-[checked]:border-accent",
        "data-[indeterminate]:bg-accent data-[indeterminate]:border-accent",
        "focus:outline-hidden focus-visible:ring-2 focus-visible:ring-accent focus-visible:ring-offset-2",
        "disabled:opacity-50 disabled:cursor-not-allowed disabled:hover:border-control-border",
        className
      )}
    >
      <BaseCheckbox.Indicator
        className="flex items-center justify-center text-background"
        keepMounted={false}
      >
        {indeterminate ? (
          <Minus className={ICON_SIZE[size]} />
        ) : (
          <Check className={ICON_SIZE[size]} />
        )}
      </BaseCheckbox.Indicator>
    </BaseCheckbox.Root>
  );
}

export { Checkbox };
export type { CheckboxProps, CheckboxSize };
```

- [ ] **Step 4: Run tests, verify they pass**

Run: `pnpm --dir frontend test src/react/components/ui/checkbox.test.tsx`

Expected: all 10 tests PASS.

- [ ] **Step 5: Type-check**

Run: `pnpm --dir frontend type-check`

Expected: no errors. If Base UI's `Checkbox.Root` rejects an `indeterminate` prop or the `onCheckedChange` value type mismatches, adjust per the actual `@base-ui/react/checkbox` types.

- [ ] **Step 6: Format and lint**

Run: `pnpm --dir frontend fix`

Expected: clean exit; if anything was reformatted, re-run tests/type-check.

- [ ] **Step 7: Commit**

```bash
git add frontend/src/react/components/ui/checkbox.tsx frontend/src/react/components/ui/checkbox.test.tsx
git commit -m "feat(react): add unified Checkbox UI component"
```

---

## Task 2: Migrate single-header indeterminate files

These 8 files all use the same pattern: a single header checkbox whose `indeterminate` state is set imperatively (either via `useRef` + `useEffect` or via inline `ref={(el) => { if (el) el.indeterminate = ... }}`). Sometimes they have row-level checkboxes nearby; migrate those too in the same file so visuals stay consistent.

**Files:**
- Modify: `frontend/src/react/pages/settings/InstancesPage.tsx` (header ~1133, row checkbox below)
- Modify: `frontend/src/react/components/database/DatabaseTableView.tsx` (header ~142, row ~152)
- Modify: `frontend/src/react/pages/project/database-detail/revision/DatabaseRevisionTable.tsx` (header ~37 useEffect + checkbox usage)
- Modify: `frontend/src/react/pages/project/database-detail/catalog/SensitiveColumnTable.tsx` (header ~134 useEffect + checkbox usage)
- Modify: `frontend/src/react/pages/project/ProjectPlanDashboardPage.tsx` (~1071 inline ref callback)
- Modify: `frontend/src/react/pages/project/export-center/DataExportPrepSheet.tsx` (~792 inline ref callback)
- Modify: `frontend/src/react/pages/project/plan-detail/components/PlanDetailChangesBranch.tsx` (~1438 inline ref callback)
- Modify: `frontend/src/react/pages/project/plan-detail/components/deploy/DeployTaskToolbar.tsx` (~111 useRef + indeterminate)

### Canonical migration patterns

**Pattern A — `useRef` + `useEffect` (e.g., InstancesPage, DatabaseTableView, DatabaseRevisionTable, SensitiveColumnTable, DeployTaskToolbar)**

Before:
```tsx
const headerCheckboxRef = useRef<HTMLInputElement>(null);
useEffect(() => {
  if (headerCheckboxRef.current) {
    headerCheckboxRef.current.indeterminate = someSelected;
  }
}, [someSelected]);
// ...
<input
  ref={headerCheckboxRef}
  type="checkbox"
  checked={allSelected}
  onChange={toggleSelectAll}
  className="rounded-xs border-control-border"
/>
```

After:
```tsx
// Delete the useRef + useEffect entirely.
<Checkbox
  checked={allSelected ? true : someSelected ? "indeterminate" : false}
  onCheckedChange={() => toggleSelectAll()}
/>
```

If the original `onChange` handler reads `e.target.checked`, replace with `onCheckedChange={(checked) => toggleSelectAll(checked)}`. If it ignores the event (most do — it's a pure toggle), use `onCheckedChange={() => toggleSelectAll()}`.

After deleting the ref/effect, also remove now-unused imports (`useRef`, `useEffect`) if no other code in the file uses them.

**Pattern B — inline ref callback (e.g., ProjectPlanDashboardPage, DataExportPrepSheet, PlanDetailChangesBranch)**

Before:
```tsx
<input
  type="checkbox"
  checked={allSelected}
  ref={(el) => {
    if (el) el.indeterminate = someSelected;
  }}
  onChange={toggleSelectAll}
  className="..."
/>
```

After:
```tsx
<Checkbox
  checked={allSelected ? true : someSelected ? "indeterminate" : false}
  onCheckedChange={() => toggleSelectAll()}
/>
```

**Pattern C — row-level checkbox (in same file, no indeterminate)**

Before:
```tsx
<input
  type="checkbox"
  checked={selectedNames?.has(db.name) ?? false}
  onChange={() => toggleSelection(db.name)}
  onClick={(e) => e.stopPropagation()}
  className="rounded-xs border-control-border"
/>
```

After:
```tsx
<Checkbox
  checked={selectedNames?.has(db.name) ?? false}
  onCheckedChange={() => toggleSelection(db.name)}
  onClick={(e) => e.stopPropagation()}
/>
```

Note: `Checkbox`'s `onClick` prop (added in Task 1) is forwarded to `BaseCheckbox.Root`, so `onClick={(e) => e.stopPropagation()}` works directly.

### Steps

- [ ] **Step 1: Migrate `InstancesPage.tsx`**

Open `frontend/src/react/pages/settings/InstancesPage.tsx`. Around lines 1118–1170:
- Delete `headerCheckboxRef`, the `useRef<HTMLInputElement>(null)` line, and the `useEffect` that sets `indeterminate`.
- Replace the header `<input type="checkbox" ... ref={headerCheckboxRef} ... />` with `<Checkbox checked={allSelected ? true : someSelected ? "indeterminate" : false} onCheckedChange={() => toggleSelectAll()} />`.
- Replace the row `<input type="checkbox" ... />` (in the same `columns` array `render`) with `<Checkbox checked={...} onCheckedChange={() => ...} onClick={(e) => e.stopPropagation()} />`.
- Add `import { Checkbox } from "@/react/components/ui/checkbox";`. Remove `useRef` and `useEffect` imports if no other code uses them.

- [ ] **Step 2: Migrate `DatabaseTableView.tsx`**

Open `frontend/src/react/components/database/DatabaseTableView.tsx`. Around lines 124–160:
- Delete `headerCheckboxRef` and its `useEffect`.
- Replace the header and row `<input type="checkbox" ...>` with `<Checkbox ...>` per Pattern A and Pattern C above.
- Add the `Checkbox` import; remove unused `useRef`/`useEffect` imports if applicable.

- [ ] **Step 3: Migrate `DatabaseRevisionTable.tsx`**

Open `frontend/src/react/pages/project/database-detail/revision/DatabaseRevisionTable.tsx`. Around line 37 and the corresponding header `<input type="checkbox">` site (search the file for `type="checkbox"`):
- Delete `headerCheckboxRef` and its `useEffect`.
- Replace `<input type="checkbox">` instances with `<Checkbox>` per Patterns A/C.
- Update imports.

- [ ] **Step 4: Migrate `SensitiveColumnTable.tsx`**

Open `frontend/src/react/pages/project/database-detail/catalog/SensitiveColumnTable.tsx`. Around lines 130–140 (note: the file has two `indeterminate` assignments — both are inside the same `useEffect` and become one declarative prop):
- Delete `headerCheckboxRef` and its `useEffect`.
- Replace `<input type="checkbox">` with `<Checkbox>` per Patterns A/C.
- Update imports.

- [ ] **Step 5: Migrate `ProjectPlanDashboardPage.tsx`**

Open `frontend/src/react/pages/project/ProjectPlanDashboardPage.tsx` line 1071 area. Pattern B (inline ref callback). Replace the header and any sibling row `<input type="checkbox">` per Patterns B/C. Add the `Checkbox` import.

- [ ] **Step 6: Migrate `DataExportPrepSheet.tsx`**

Open `frontend/src/react/pages/project/export-center/DataExportPrepSheet.tsx` line 792 area. Same as Step 6 — Pattern B + sibling row Pattern C.

- [ ] **Step 7: Migrate `PlanDetailChangesBranch.tsx`**

Open `frontend/src/react/pages/project/plan-detail/components/PlanDetailChangesBranch.tsx` line 1438 area. Same as Step 5.

- [ ] **Step 8: Migrate `DeployTaskToolbar.tsx`**

Open `frontend/src/react/pages/project/plan-detail/components/deploy/DeployTaskToolbar.tsx` around line 111. Pattern A (uses `checkboxRef.current.indeterminate = ...`). Same migration as Step 1.

- [ ] **Step 9: Validate the batch**

```
pnpm --dir frontend fix
pnpm --dir frontend type-check
pnpm --dir frontend test
```

All three must succeed. If `type-check` complains about an `onChange` callback signature mismatch on a row checkbox, the original handler took `(e: ChangeEvent)` — convert it to either `() => ...` or `(checked: boolean) => ...` to match `Checkbox`'s `onCheckedChange`.

- [ ] **Step 10: Confirm no `indeterminate` references remain in this batch**

```
grep -n "indeterminate" frontend/src/react/pages/settings/InstancesPage.tsx \
  frontend/src/react/components/database/DatabaseTableView.tsx \
  frontend/src/react/pages/project/database-detail/revision/DatabaseRevisionTable.tsx \
  frontend/src/react/pages/project/database-detail/catalog/SensitiveColumnTable.tsx \
  frontend/src/react/pages/project/ProjectPlanDashboardPage.tsx \
  frontend/src/react/pages/project/export-center/DataExportPrepSheet.tsx \
  frontend/src/react/pages/project/plan-detail/components/PlanDetailChangesBranch.tsx \
  frontend/src/react/pages/project/plan-detail/components/deploy/DeployTaskToolbar.tsx
```

Expected: empty output (or only matches inside string literals / unrelated code).

- [ ] **Step 11: Commit**

```bash
git add -u
git commit -m "refactor(react): migrate single-header indeterminate checkboxes to Checkbox component"
```

---

## Task 3: Migrate multi-checkbox files

These 5 files contain multiple checkbox sites — header + multiple row variants, or tree-style nested checkboxes. Migrations are still mechanical but each file deserves a focused review pass.

**Files:**
- Modify: `frontend/src/react/components/IssueTable.tsx` (lines 632 row, 924 sticky-bar header with `indeterminate = !allSelected`)
- Modify: `frontend/src/react/components/DatabaseResourceSelector.tsx` (lines 445, 532, 634 — three checkbox sites, two with `indeterminate`)
- Modify: `frontend/src/react/components/ExprEditor.tsx` (lines 625, 805, 826 — header at 805 has `indeterminate`)
- Modify: `frontend/src/react/components/sql-review/RuleTable.tsx` (lines 208, 413, 529 — header at 208 has `indeterminate`)
- Modify: `frontend/src/react/components/SchemaEditorLite/Aside/NodeCheckbox.tsx` (7 sites, 3 use `indeterminate`)

### Steps

- [ ] **Step 1: Migrate `IssueTable.tsx`**

Two sites:
- Line ~632 (row in `IssueTableRow`): plain row checkbox with `accent-accent`. Migrate per Pattern C.
- Line ~924 (sticky toolbar): the indeterminate logic is `el.indeterminate = !allSelected`, which encodes "if any but not all selected". The current code only renders this toolbar when there's at least one selection, so the `checked={allSelected}` + `indeterminate=!allSelected` translates to:
  ```tsx
  <Checkbox
    checked={allSelected ? true : "indeterminate"}
    onCheckedChange={() => onToggleSelectAll()}
  />
  ```

Both sites use `onClick={(e) => e.stopPropagation()}` — pass through via the new `onClick` prop.

Replace `className="accent-accent"` with `className="shrink-0 mt-1"` on the row site (preserving alignment), and drop the className entirely on the toolbar site unless layout breaks.

- [ ] **Step 2: Migrate `DatabaseResourceSelector.tsx`**

Three checkbox sites:
- Line ~445: db-level checkbox with `dbIndeterminate` — Pattern B (inline ref callback). Translate to `checked={dbAllSelected ? true : dbIndeterminate ? "indeterminate" : false}`.
- Line ~532: schema-level checkbox with `schemaIndeterminate` — same transform.
- Line ~634: third site (verify by reading) — likely a row-level Pattern C.

Read each call site to recover the exact source booleans (`dbAllSelected`, `dbIndeterminate`, etc.) — names may differ.

- [ ] **Step 3: Migrate `ExprEditor.tsx`**

Three sites:
- Line ~625: row-level Pattern C.
- Line ~805: header with `el.indeterminate = anySelected && !allSelected` — translate to `checked={allSelected ? true : anySelected ? "indeterminate" : false}`.
- Line ~826: row-level Pattern C.

- [ ] **Step 4: Migrate `RuleTable.tsx`**

Four sites:
- Line ~208: header — Pattern A or B. Read the surrounding code to extract the exact boolean expressions used in the indeterminate computation.
- Lines ~413, ~529: row-level Pattern C.

- [ ] **Step 5: Migrate `NodeCheckbox.tsx`**

This file is `frontend/src/react/components/SchemaEditorLite/Aside/NodeCheckbox.tsx` and contains 7 nearly-identical small components (`TableCheckbox`, `ColumnCheckbox`, `ViewCheckbox`, `ProcedureCheckbox`, `FunctionCheckbox`, `GroupCheckbox` table/view branches). Each takes a `state` object with `{ checked, indeterminate }` already computed.

For every `<input type="checkbox" ... />` in this file, replace with:

```tsx
<Checkbox
  checked={state.checked ? true : state.indeterminate ? "indeterminate" : false}
  size="sm"
  onCheckedChange={(checked) => selection.updateXxxSelection(node.db, node.metadata, checked)}
  onClick={(e) => e.stopPropagation()}
/>
```

Use `size="sm"` because the original used `className="size-3.5"` (14px). Drop the `className` prop. The `state.indeterminate` value already exists on every state object (see `state.indeterminate: boolean` in `types.ts`), so the conditional is straightforward.

For the components where `state.indeterminate` was previously not used at all (e.g., `ColumnCheckbox`), still use the same expression — `state.indeterminate` will be `false` and the result simplifies to `state.checked`. Keeping the expression uniform makes future edits safer.

- [ ] **Step 6: Validate the batch**

```
pnpm --dir frontend fix
pnpm --dir frontend type-check
pnpm --dir frontend test
```

If a test in `SchemaEditorLite` or sql-review queries `input[type="checkbox"]`, that test will need its selector updated to `[role="checkbox"]`. Search:

```
grep -rn 'input\[type="checkbox"\]' frontend/src/react/components/SchemaEditorLite frontend/src/react/components/sql-review frontend/src/react/components/IssueTable.test.tsx 2>/dev/null
```

Update any matches to `[role="checkbox"]`. Re-run tests.

- [ ] **Step 7: Confirm no `indeterminate` references remain in this batch**

```
grep -n "indeterminate" frontend/src/react/components/IssueTable.tsx \
  frontend/src/react/components/DatabaseResourceSelector.tsx \
  frontend/src/react/components/ExprEditor.tsx \
  frontend/src/react/components/sql-review/RuleTable.tsx \
  frontend/src/react/components/SchemaEditorLite/Aside/NodeCheckbox.tsx
```

Expected: empty (or only string-literal matches in `useSelection.ts` types, which is fine — that file is not in this batch).

- [ ] **Step 8: Commit**

```bash
git add -u
git commit -m "refactor(react): migrate multi-checkbox components to Checkbox component"
```

---

## Task 4: Migrate paired-row files and the test selector update

These 3 files have row-level checkboxes paired with a header (no `indeterminate` manipulation, but visually paired with checkboxes already migrated in earlier tasks or that appear in the same surface). Plus one test selector update.

**Files:**
- Modify: `frontend/src/react/components/ProjectTable.tsx` (lines 146 header, 214 row)
- Modify: `frontend/src/react/components/release/ReleaseFileTable.tsx` (line 123 row, plus header if present)
- Modify: `frontend/src/react/components/release/ReleaseFileTable.test.tsx` (lines 80, 92 selector update)
- Modify: `frontend/src/react/pages/settings/MembersPage.tsx` (lines 241 header, 258 row)

### Steps

- [ ] **Step 1: Migrate `ProjectTable.tsx`**

Two sites at lines ~146 (header) and ~214 (row). Read both and migrate per Pattern A/C. If the header has a `selectedAll` boolean and a `someSelected` boolean (typical), translate to `checked={selectedAll ? true : someSelected ? "indeterminate" : false}`. Otherwise, if it's a plain boolean header, translate to `checked={selectedAll}`.

- [ ] **Step 2: Migrate `ReleaseFileTable.tsx`**

Site at line ~123 (row). Search the file for any other `type="checkbox"` (header or all-select) and migrate them too:

```
grep -n 'type="checkbox"' frontend/src/react/components/release/ReleaseFileTable.tsx
```

- [ ] **Step 3: Update `ReleaseFileTable.test.tsx` selector**

Open `frontend/src/react/components/release/ReleaseFileTable.test.tsx`. Lines 80 and 92 currently read:

```tsx
hidden.container.querySelectorAll('input[type="checkbox"]').length
shown.container.querySelectorAll('input[type="checkbox"]').length
```

Change to:

```tsx
hidden.container.querySelectorAll('[role="checkbox"]').length
shown.container.querySelectorAll('[role="checkbox"]').length
```

This is necessary because Base UI's `Checkbox.Root` renders `<button role="checkbox">`, not `<input type="checkbox">`.

- [ ] **Step 4: Migrate `MembersPage.tsx`**

Two sites at lines ~241 (header) and ~258 (row). Read both. Header may or may not have `indeterminate` — search the file for `indeterminate`. If yes, Pattern A/B; if no, plain `checked={...}`. Row is Pattern C.

- [ ] **Step 5: Audit other tests for the same selector**

```
grep -rn 'input\[type="checkbox"\]' frontend/src/react --include="*.test.tsx" --include="*.test.ts"
```

For any match in a file whose corresponding component was migrated in Tasks 2/3/4, update the selector to `[role="checkbox"]`. (The files already covered: `ReleaseFileTable.test.tsx`. If the grep also flags `domTree.test.ts`, that file uses checkboxes as DOM-tree fixture data unrelated to our component — leave it alone.)

- [ ] **Step 6: Validate the batch**

```
pnpm --dir frontend fix
pnpm --dir frontend type-check
pnpm --dir frontend test
```

- [ ] **Step 7: Commit**

```bash
git add -u
git commit -m "refactor(react): migrate paired row checkboxes to Checkbox component"
```

---

## Task 5: Final verification

Verify the migration's done state matches the spec.

- [ ] **Step 1: Confirm no remaining `el.indeterminate = ...` patterns in React**

```
grep -rn "indeterminate = " frontend/src/react --include="*.tsx" --include="*.ts"
```

Expected: only matches in `frontend/src/react/components/SchemaEditorLite/types.ts` (`indeterminate: boolean` type field) and `frontend/src/react/components/SchemaEditorLite/useSelection.ts` (return-value `indeterminate: ...` keys). Both are computing the value, not assigning to a DOM property — they're fine.

If you see anything like `headerCheckboxRef.current.indeterminate = ...` or `if (el) el.indeterminate = ...`, those are leftovers — go back and migrate them.

- [ ] **Step 2: Confirm migrated files are using the new component**

```
grep -rln "from \"@/react/components/ui/checkbox\"" frontend/src/react
```

Expected: at least the ~16 migrated files plus the component's own test.

- [ ] **Step 3: Visually check the dev server**

Start the dev server:

```
pnpm --dir frontend dev
```

Open the browser, log in, and exercise the migrated surfaces:
- IssueTable: select an issue, verify the toolbar checkbox shows the partial-selected state.
- SchemaEditorLite (in a Schema Sync issue): open the tree, select some columns, verify the parent table/group checkboxes show indeterminate.
- DatabaseResourceSelector: in a project access flow, partially select databases, verify schema-level indeterminate.
- Settings → Instances: select some, verify header indeterminate.
- Project → Plan dashboard: select some, verify header indeterminate.
- Release files: open a release with multiple files, toggle selection.

For each surface, confirm: (a) checked state shows the white check icon on `bg-accent`, (b) indeterminate state shows the white minus icon on `bg-accent`, (c) clicking indeterminate transitions to fully checked, (d) clicking checked transitions to unchecked, (e) disabled state is greyed out.

Type-checking and unit tests verify code correctness, not visual correctness — this manual pass is required before considering the migration complete.

- [ ] **Step 4: Final lint/type/test gate**

```
pnpm --dir frontend fix
pnpm --dir frontend check
pnpm --dir frontend type-check
pnpm --dir frontend test
```

All four must pass clean.

- [ ] **Step 5: Document the remaining out-of-scope checkbox usages**

No commit needed — this is an awareness step. The ~40 form-only `<input type="checkbox">` usages listed in the spec's "Explicitly out of scope" section remain. They are non-blocking and will migrate organically. If the team wants to track them, the spec already enumerates them.

- [ ] **Step 6: Open the PR**

The branch already has commits from Tasks 1–4. Push and open a PR per `docs/pre-pr-checklist.md`. Title suggestion: `feat(react): unified Checkbox component + migrate indeterminate sites`.

---

## Self-Review

**Spec coverage:**
- §1 API → Task 1 Step 3 implements the exact `CheckboxProps` interface (including `onClick` for stopPropagation use cases in row-level migrations). One additive prop beyond the spec; not a deviation. ✓
- §2 Visual → Task 1 Step 3 implements every state row from the visual table. ✓
- §3 Migration scope → Tasks 2 (8 files), 3 (5 files), 4 (3 files + test) cover all 16 in-scope files. Task 4 Step 5 audits remaining test selectors. ✓
- §4 Testing → Task 1 Steps 1–4 cover unit tests; Task 4 Step 3 + Step 5 cover the `ReleaseFileTable.test.tsx` regression risk explicitly. ✓

**Placeholder scan:** No "TBD"/"TODO". Each migration step gives the file path, line number range, and the canonical before/after pattern. Files where the exact handler signature varies (DatabaseResourceSelector, RuleTable) instruct the engineer to read the call site to recover variable names — that's appropriate, not a placeholder.

**Type consistency:** `CheckboxProps`, `Checkbox`, `CheckboxSize` names used consistently. `onCheckedChange(checked: boolean)` signature consistent across all migration patterns.

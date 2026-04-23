# React Z-Index Policy Cleanup Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Make React global overlays obey the semantic `overlay` / `agent` / `critical` layer policy from PR #20012 and prevent new feature-owned z-index escalation.

**Architecture:** Reuse the existing semantic layer roots in `frontend/src/react/components/ui/layer.ts`. Convert feature-owned full-screen overlays to `Dialog`, `AlertDialog`, or `Sheet`, convert escaping menus/popovers to overlay-root portals or existing primitives, then wire a script-based guardrail into `pnpm --dir frontend check`.

**Tech Stack:** React, Base UI wrappers, Tailwind CSS v4, Node ESM scripts, pnpm, Vitest.

---

## File Map

- Create: `frontend/scripts/check-react-layering.mjs`
  - Scans React source for feature-owned global overlay z-index patterns.
- Modify: `frontend/package.json`
  - Adds `node scripts/check-react-layering.mjs` to the `check` script after cleanup.
- Modify: `frontend/src/react/components/ui/sheet.tsx`
  - Adds width tiers needed by existing large drawers so consumers do not inline global `fixed z-*` shells.
- Modify global modal/dialog files:
  - `frontend/src/react/pages/settings/IDPDetailPage.tsx`
  - `frontend/src/react/pages/settings/IDPsPage.tsx`
  - `frontend/src/react/pages/settings/RolesPage.tsx`
  - `frontend/src/react/pages/project/ProjectDatabasesPage.tsx`
  - `frontend/src/react/pages/settings/InstancesPage.tsx`
  - `frontend/src/react/pages/settings/ProjectsPage.tsx`
  - `frontend/src/react/pages/settings/general/GeneralSection.tsx`
  - `frontend/src/react/components/AuditLogTable.tsx`
- Modify global drawer files:
  - `frontend/src/react/pages/settings/MembersPage.tsx`
  - `frontend/src/react/pages/settings/SemanticTypesPage.tsx`
  - `frontend/src/react/components/sql-review/Panels.tsx`
  - `frontend/src/react/components/IssueTable.tsx`
  - `frontend/src/react/pages/settings/IDPsPage.tsx`
  - `frontend/src/react/pages/project/ProjectPlanDashboardPage.tsx`
  - `frontend/src/react/pages/project/ProjectSyncSchemaPage.tsx`
  - `frontend/src/react/components/instance/InfoPanel.tsx`
- Modify menu/popover files:
  - `frontend/src/react/components/SchemaEditorLite/Aside/AsideTree.tsx`
  - `frontend/src/react/pages/project/plan-detail/components/PlanDetailHeader.tsx`
  - `frontend/src/react/pages/project/plan-detail/components/PlanDetailMetadataSidebar.tsx`
  - `frontend/src/react/components/EnvironmentMultiSelect.tsx`
- Test/update:
  - Existing nearby tests for converted components.
  - Add focused tests only where a component has existing test coverage or the conversion changes Escape/outside-click behavior.

---

### Task 1: Add Report-Only React Layering Scanner

**Files:**
- Create: `frontend/scripts/check-react-layering.mjs`
- No package script wiring yet.

- [ ] **Step 1: Create the scanner**

Use this complete file:

```js
// frontend/scripts/check-react-layering.mjs
//
// Enforces the React semantic overlay layering policy introduced by PR #20012.
// Feature code must not create global z-index overlays directly. Use shared UI
// primitives or portal into the semantic layer roots instead.

import { readFileSync, readdirSync } from "fs";
import { relative, resolve } from "path";
import { fileURLToPath } from "url";
import { dirname } from "path";

const __dirname = dirname(fileURLToPath(import.meta.url));
const ROOT = resolve(__dirname, "..");
const REACT_DIR = resolve(ROOT, "src/react");
const REPORT_ONLY = process.argv.includes("--report-only");

const APPROVED_PREFIXES = [
  "src/react/components/ui/",
  "src/react/plugins/agent/",
];

const APPROVED_FILES = new Set([
  "src/react/components/auth/SessionExpiredSurface.tsx",
]);

const LOCAL_PAINT_ORDER_EXCEPTIONS = new Map([
  [
    "src/react/components/monaco/MonacoEditor.tsx",
    "Monaco action buttons are local to the editor surface.",
  ],
  [
    "src/react/plugins/agent/components/AgentInput.tsx",
    "Approved agent-owned layer family.",
  ],
  [
    "src/react/plugins/agent/components/AgentWindow.tsx",
    "Approved agent-owned layer family.",
  ],
]);

const findFiles = (dir) => {
  const files = [];
  for (const entry of readdirSync(dir, { withFileTypes: true })) {
    const full = resolve(dir, entry.name);
    if (entry.isDirectory()) {
      files.push(...findFiles(full));
    } else if (/\.(ts|tsx)$/.test(entry.name)) {
      files.push(full);
    }
  }
  return files;
};

const isApprovedPath = (path) =>
  APPROVED_FILES.has(path) ||
  APPROVED_PREFIXES.some((prefix) => path.startsWith(prefix));

const hasRawZClass = (line) =>
  /\bz-\d+\b/.test(line) || /\bz-\[[^\]]+\]/.test(line);

const hasGlobalFixedZ = (line) => /\bfixed\b/.test(line) && hasRawZClass(line);

const hasHighAbsoluteZ = (line) =>
  /\babsolute\b/.test(line) && /\b(?:z-4\d|z-5\d|z-\[[^\]]+\])\b/.test(line);

const hasInlineZIndex = (line) => /\bzIndex\s*:/.test(line);

const scanFile = (file) => {
  const rel = relative(ROOT, file);
  if (isApprovedPath(rel)) {
    return [];
  }
  if (LOCAL_PAINT_ORDER_EXCEPTIONS.has(rel)) {
    return [];
  }

  const source = readFileSync(file, "utf-8");
  const violations = [];
  const lines = source.split("\n");

  lines.forEach((line, index) => {
    const lineNumber = index + 1;
    if (hasGlobalFixedZ(line)) {
      violations.push({
        rel,
        lineNumber,
        reason: "feature-owned fixed overlay uses raw z-index",
        line,
      });
    }
    if (hasHighAbsoluteZ(line)) {
      violations.push({
        rel,
        lineNumber,
        reason: "feature-owned absolute overlay uses high raw z-index",
        line,
      });
    }
    if (hasInlineZIndex(line)) {
      violations.push({
        rel,
        lineNumber,
        reason: "inline zIndex bypasses semantic layer ownership",
        line,
      });
    }
  });

  const portalToBody = /createPortal\([\s\S]*?document\.body/g;
  let match;
  while ((match = portalToBody.exec(source))) {
    const lineNumber = source.slice(0, match.index).split("\n").length;
    violations.push({
      rel,
      lineNumber,
      reason: "feature-owned portal targets document.body directly",
      line: lines[lineNumber - 1] ?? "",
    });
  }

  return violations;
};

const violations = findFiles(REACT_DIR).flatMap(scanFile);

if (violations.length > 0) {
  console.error(
    `React layering policy violations (${violations.length}). ` +
      "Use shared overlay primitives or getLayerRoot(\"overlay\").\n"
  );
  for (const violation of violations) {
    console.error(
      `${violation.rel}:${violation.lineNumber}: ${violation.reason}\n` +
        `  ${violation.line.trim()}\n`
    );
  }
  if (!REPORT_ONLY) {
    process.exit(1);
  }
}

console.log(
  violations.length === 0
    ? "React layering policy: all checks passed."
    : "React layering policy: report-only mode completed with violations."
);
```

- [ ] **Step 2: Run in report-only mode**

Run:

```bash
pnpm --dir frontend exec node scripts/check-react-layering.mjs --report-only
```

Expected: exit 0 and prints current violations.

- [ ] **Step 3: Commit**

```bash
git add frontend/scripts/check-react-layering.mjs
git commit -m "chore(frontend): add react layering policy scanner"
```

---

### Task 2: Convert Centered Modal Surfaces To Dialog Primitives

**Files:**
- Modify: `frontend/src/react/pages/settings/IDPDetailPage.tsx`
- Modify: `frontend/src/react/pages/settings/IDPsPage.tsx`
- Modify: `frontend/src/react/pages/settings/RolesPage.tsx`
- Modify: `frontend/src/react/pages/project/ProjectDatabasesPage.tsx`
- Modify: `frontend/src/react/pages/settings/InstancesPage.tsx`
- Modify: `frontend/src/react/pages/settings/ProjectsPage.tsx`
- Modify: `frontend/src/react/pages/settings/general/GeneralSection.tsx`
- Modify: `frontend/src/react/components/AuditLogTable.tsx`

- [ ] **Step 1: Import primitives in each touched file**

Use `Dialog` for result/details/import modals and `AlertDialog` for destructive or confirmation modals:

```ts
import {
  Dialog,
  DialogContent,
  DialogTitle,
} from "@/react/components/ui/dialog";
import {
  AlertDialog,
  AlertDialogContent,
  AlertDialogTitle,
  AlertDialogDescription,
} from "@/react/components/ui/alert-dialog";
```

Only import the primitives each file actually uses.

- [ ] **Step 2: Replace manual centered result/detail modals**

For result/detail modals such as `TestConnectionResultDialog`, `ImportPermissionModal`, and `AuditLogTable` JSON detail view, use this shape:

```tsx
return (
  <Dialog open onOpenChange={(nextOpen) => !nextOpen && onClose()}>
    <DialogContent className="w-[32rem] max-w-[calc(100vw-2rem)] p-6">
      <DialogTitle>{title}</DialogTitle>
      <div className="mt-4 flex flex-col gap-y-4">{children}</div>
    </DialogContent>
  </Dialog>
);
```

For the `AuditLogTable` JSON detail view, preserve the large viewport sizing:

```tsx
<DialogContent className="flex h-[calc(100vh-12rem)] w-[calc(100vw-12rem)] max-w-none flex-col p-0">
```

- [ ] **Step 3: Replace manual confirmation modals**

For destructive confirmations such as role delete, IDP delete, and database unassign, use this shape:

```tsx
return (
  <AlertDialog open onOpenChange={(nextOpen) => !nextOpen && onCancel()}>
    <AlertDialogContent>
      <AlertDialogTitle>{title}</AlertDialogTitle>
      <AlertDialogDescription className="mt-2">
        {description}
      </AlertDialogDescription>
      <div className="mt-6 flex justify-end gap-x-2">
        <Button variant="outline" onClick={onCancel}>
          {t("common.cancel")}
        </Button>
        <Button variant="destructive" onClick={onConfirm}>
          {confirmText}
        </Button>
      </div>
    </AlertDialogContent>
  </AlertDialog>
);
```

If an existing confirmation uses warning styling instead of destructive styling, keep the current button variant/classes but keep the `AlertDialog` shell.

- [ ] **Step 4: Remove manual Escape handlers that duplicate primitives**

Remove `useEscapeKey` calls or `document.addEventListener("keydown", handler)` blocks from components converted to `Dialog` or `AlertDialog`. Base UI handles Escape through `onOpenChange`.

- [ ] **Step 5: Run targeted checks**

Run:

```bash
pnpm --dir frontend exec node scripts/check-react-layering.mjs --report-only
pnpm --dir frontend test -- --run frontend/src/react/components/ui/dialog.test.tsx frontend/src/react/components/ui/layer.test.tsx
```

Expected: the converted files no longer report fixed elements with raw `z-*`; the targeted tests pass.

- [ ] **Step 6: Commit**

```bash
git add frontend/src/react/pages/settings/IDPDetailPage.tsx \
  frontend/src/react/pages/settings/IDPsPage.tsx \
  frontend/src/react/pages/settings/RolesPage.tsx \
  frontend/src/react/pages/project/ProjectDatabasesPage.tsx \
  frontend/src/react/pages/settings/InstancesPage.tsx \
  frontend/src/react/pages/settings/ProjectsPage.tsx \
  frontend/src/react/pages/settings/general/GeneralSection.tsx \
  frontend/src/react/components/AuditLogTable.tsx
git commit -m "fix(frontend): route modal surfaces through overlay layer"
```

---

### Task 3: Convert Manual Drawers To Sheet

**Files:**
- Modify: `frontend/src/react/components/ui/sheet.tsx`
- Modify: `frontend/src/react/pages/settings/MembersPage.tsx`
- Modify: `frontend/src/react/pages/settings/SemanticTypesPage.tsx`
- Modify: `frontend/src/react/components/sql-review/Panels.tsx`
- Modify: `frontend/src/react/components/IssueTable.tsx`
- Modify: `frontend/src/react/pages/settings/IDPsPage.tsx`
- Modify: `frontend/src/react/pages/project/ProjectPlanDashboardPage.tsx`
- Modify: `frontend/src/react/pages/project/ProjectSyncSchemaPage.tsx`
- Modify: `frontend/src/react/components/instance/InfoPanel.tsx`

- [ ] **Step 1: Add the required `SheetContent` width tiers**

In `frontend/src/react/components/ui/sheet.tsx`, extend `sheetContentVariants`:

```ts
width: {
  narrow: "w-[24rem]",
  standard: "w-[44rem]",
  wide: "w-[52rem]",
  large: "w-[64rem]",
  xlarge: "w-[70rem]",
  workspace: "w-[calc(100vw-8rem)] lg:w-240 max-w-[calc(100vw-8rem)]",
},
```

- [ ] **Step 2: Import sheet primitives in drawer files**

```ts
import {
  Sheet,
  SheetContent,
  SheetHeader,
  SheetTitle,
} from "@/react/components/ui/sheet";
```

Only import `SheetDescription` or `SheetFooter` if the existing component already has equivalent content.

- [ ] **Step 3: Replace each manual drawer shell**

Use this pattern:

```tsx
return (
  <Sheet open={open} onOpenChange={(nextOpen) => !nextOpen && onClose()}>
    <SheetContent width="large">
      <SheetHeader>
        <SheetTitle>{title}</SheetTitle>
      </SheetHeader>
      <div className="flex-1 overflow-y-auto p-6">{body}</div>
    </SheetContent>
  </Sheet>
);
```

Map existing widths to tiers:

- `w-[40rem]` -> `width="standard"`
- `w-[64rem]` -> `width="large"`
- `w-[70rem]` -> `width="xlarge"`
- `w-[calc(100vw-8rem)] lg:w-160` -> `width="large"` unless visible content is clipped
- `w-[calc(100vw-8rem)] lg:w-240` -> `width="workspace"`

- [ ] **Step 4: Preserve close behavior**

Move existing close buttons into `SheetHeader` only when they provide custom behavior. Otherwise use the built-in close button from `SheetHeader` and keep `onOpenChange` as the single close path.

- [ ] **Step 5: Run targeted checks**

Run:

```bash
pnpm --dir frontend exec node scripts/check-react-layering.mjs --report-only
pnpm --dir frontend test -- --run frontend/src/react/components/ui/dialog.test.tsx frontend/src/react/components/ui/layer.test.tsx
```

Expected: converted drawer files no longer report fixed elements with raw `z-*`; tests pass.

- [ ] **Step 6: Commit**

```bash
git add frontend/src/react/components/ui/sheet.tsx \
  frontend/src/react/pages/settings/MembersPage.tsx \
  frontend/src/react/pages/settings/SemanticTypesPage.tsx \
  frontend/src/react/components/sql-review/Panels.tsx \
  frontend/src/react/components/IssueTable.tsx \
  frontend/src/react/pages/settings/IDPsPage.tsx \
  frontend/src/react/pages/project/ProjectPlanDashboardPage.tsx \
  frontend/src/react/pages/project/ProjectSyncSchemaPage.tsx \
  frontend/src/react/components/instance/InfoPanel.tsx
git commit -m "fix(frontend): route drawer surfaces through sheet layer"
```

---

### Task 4: Convert Escaping Menus And Popovers To Overlay Layer

**Files:**
- Modify: `frontend/src/react/components/SchemaEditorLite/Aside/AsideTree.tsx`
- Modify: `frontend/src/react/pages/project/plan-detail/components/PlanDetailHeader.tsx`
- Modify: `frontend/src/react/pages/project/plan-detail/components/PlanDetailMetadataSidebar.tsx`
- Modify: `frontend/src/react/components/EnvironmentMultiSelect.tsx`

- [ ] **Step 1: Replace direct `document.body` portal in `AsideTree.tsx`**

Import:

```ts
import { getLayerRoot, LAYER_SURFACE_CLASS } from "@/react/components/ui/layer";
```

Change the context menu portal target and classes:

```tsx
createPortal(
  <div
    className="fixed inset-0"
    onClick={hideMenu}
    onContextMenu={(e) => {
      e.preventDefault();
      hideMenu();
    }}
  >
    <div
      className={`absolute rounded-sm border border-control-border bg-white py-1 shadow-md ${LAYER_SURFACE_CLASS}`}
      style={{ left: menuState.x, top: menuState.y }}
      onClick={(e) => e.stopPropagation()}
    >
      {menuOptions.map((opt) => (
        <button
          key={opt.key}
          type="button"
          className="flex w-full items-center px-3 py-1.5 text-left text-sm hover:bg-control-bg-hover"
          onClick={() => handleMenuSelect(opt.key)}
        >
          {opt.label}
        </button>
      ))}
    </div>
  </div>,
  getLayerRoot("overlay")
)
```

- [ ] **Step 2: Convert ready-for-review popover to `Popover`**

In `PlanDetailHeader.tsx`, replace the raw `absolute right-0 top-full z-40` panel with `Popover`, `PopoverTrigger`, and `PopoverContent` from `@/react/components/ui/popover`.

The content class should preserve width and spacing:

```tsx
<Popover open={showReviewPopover} onOpenChange={handleReviewPopoverOpenChange}>
  <PopoverTrigger render={<Button disabled={submitDisabled} title={submitDisabledReason} />}>
    {t("plan.ready-for-review")}
  </PopoverTrigger>
  <PopoverContent
    align="end"
    className="w-[min(28rem,calc(100vw-2rem))] px-4 py-4"
  >
    <ReadyForReviewPopoverContent
      checksWarningAcknowledged={checksWarningAcknowledged}
      confirmErrors={createIssueConfirmErrors}
      forceIssueLabels={project.forceIssueLabels}
      issueLabels={project.issueLabels ?? []}
      onCancel={() => handleReviewPopoverOpenChange(false)}
      onChecksWarningAcknowledgedChange={setChecksWarningAcknowledged}
      onConfirm={() => void handleCreateIssue()}
      onSelectedLabelsChange={setSelectedLabels}
      selectedLabels={selectedLabels}
      showChecksWarning={showChecksWarning}
      submitting={submittingReview}
    />
  </PopoverContent>
</Popover>
```

If the current Base UI `PopoverTrigger` API does not support `render`, keep the existing button and pass its ref as `anchor` to `PopoverContent`.

- [ ] **Step 3: Convert custom select dropdowns that must escape clipping**

For `PlanDetailMetadataSidebar.tsx` and `EnvironmentMultiSelect.tsx`, either use `Combobox` with `portal` when the existing behavior maps directly, or portal the existing dropdown content into `getLayerRoot("overlay")` with `LAYER_SURFACE_CLASS`.

The portal fallback should follow this shape:

```tsx
{open &&
  createPortal(
    <div
      ref={dropdownRef}
      className={`fixed max-h-60 overflow-auto rounded-sm border border-control-border bg-background shadow-lg ${LAYER_SURFACE_CLASS}`}
      style={dropdownStyle}
    >
      {options}
    </div>,
    getLayerRoot("overlay")
  )}
```

Use the existing `getPortalDropdownStyle` helpers from `frontend/src/react/components/ui/combobox-position.ts` when positioning against a trigger.

- [ ] **Step 4: Run targeted checks**

Run:

```bash
pnpm --dir frontend exec node scripts/check-react-layering.mjs --report-only
pnpm --dir frontend test -- --run frontend/src/react/components/ui/popover.test.tsx frontend/src/react/components/ui/combobox-position.test.ts
```

Expected: converted menu/popover files no longer report direct `document.body` portals or high raw `z-*`; tests pass.

- [ ] **Step 5: Commit**

```bash
git add frontend/src/react/components/SchemaEditorLite/Aside/AsideTree.tsx \
  frontend/src/react/pages/project/plan-detail/components/PlanDetailHeader.tsx \
  frontend/src/react/pages/project/plan-detail/components/PlanDetailMetadataSidebar.tsx \
  frontend/src/react/components/EnvironmentMultiSelect.tsx
git commit -m "fix(frontend): route popover surfaces through overlay layer"
```

---

### Task 5: Lock The Guardrail Into Frontend Check

**Files:**
- Modify: `frontend/package.json`
- Modify: `frontend/scripts/check-react-layering.mjs`

- [ ] **Step 1: Run the scanner without report-only**

Run:

```bash
pnpm --dir frontend exec node scripts/check-react-layering.mjs
```

Expected: exit 0 and `React layering policy: all checks passed.`

If violations remain, classify each as either:

- a true global overlay violation to fix before continuing, or
- a local paint-order exception to document in `LOCAL_PAINT_ORDER_EXCEPTIONS` with a file-specific reason.

- [ ] **Step 2: Wire the scanner into `pnpm check`**

Update `frontend/package.json`:

```json
"check": "eslint --cache src && biome ci . && node scripts/check-react-i18n.mjs && node scripts/check-react-layering.mjs && node scripts/sort_i18n_keys.mjs --check"
```

- [ ] **Step 3: Run full frontend validation**

Run:

```bash
pnpm --dir frontend fix
pnpm --dir frontend check
pnpm --dir frontend type-check
pnpm --dir frontend test
```

Expected: all commands exit 0.

- [ ] **Step 4: Commit**

```bash
git add frontend/package.json frontend/scripts/check-react-layering.mjs
git commit -m "chore(frontend): enforce react layering policy"
```

---

### Task 6: Final Review Pass

**Files:**
- Review all modified files from Tasks 1-5.

- [ ] **Step 1: Verify no feature-owned global z-index overlays remain**

Run:

```bash
rg -n '(fixed[^\n]*(z-\[[^\]]+\]|\bz-[0-9]+\b)|createPortal\([^)]*document\.body|zIndex\s*:)' frontend/src/react \
  --glob '*.tsx' --glob '*.ts' \
  --glob '!frontend/src/react/components/ui/**' \
  --glob '!frontend/src/react/plugins/agent/**' \
  --glob '!frontend/src/react/components/auth/SessionExpiredSurface.tsx'
```

Expected: no true global overlay violations. Local paint-order matches must be documented in `frontend/scripts/check-react-layering.mjs`.

- [ ] **Step 2: Review policy docs against implementation**

Check:

```bash
sed -n '35,46p' frontend/AGENTS.md
sed -n '1,120p' frontend/src/react/components/ui/layer.ts
```

Expected: implementation still matches the documented `overlay` > `agent` > `critical` policy and no new layer family was introduced.

- [ ] **Step 3: Commit any final review changes**

```bash
git status --short
```

If `git status --short` prints modified files, inspect them with `git diff`, then stage the listed files explicitly and commit:

```bash
git diff
git add frontend/package.json frontend/scripts/check-react-layering.mjs frontend/src/react
git commit -m "fix(frontend): finish react layering cleanup"
```

Skip the commit when `git status --short` prints no files.

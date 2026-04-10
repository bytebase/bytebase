# shadcn Design Token & Pattern Cleanup

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Align all React code with shadcn best practices — semantic color tokens, proper spacing utilities, and consistent component patterns.

**Architecture:** Add missing semantic tokens to the CSS variable system, fix the shared UI components first (highest leverage), then sweep page-level code area by area. Each task is independently mergeable.

**Tech Stack:** Tailwind CSS v4, CSS custom properties, React + Base UI + cva

---

## Phase 1: Foundation — Add Missing Semantic Tokens

### Task 1: Add `background` and `overlay` semantic tokens

**Files:**
- Modify: `frontend/src/assets/css/tailwind.css` (`:root` block, lines 106-168)

**Step 1: Add the tokens**

In the `:root` block inside `@layer base`, add after the `--color-control-border` line:

```css
    /* white - page/card/panel background */
    --color-background: 255 255 255; /* #ffffff */
    /* black/50 - overlay backdrop */
    --color-overlay: 0 0 0; /* #000000, use as bg-overlay/50 */
```

**Step 2: Verify Tailwind picks them up**

Run: `pnpm --dir frontend type-check`
Expected: PASS (CSS vars are auto-available in Tailwind v4 via `@config`)

**Step 3: Commit**

```bash
git add frontend/src/assets/css/tailwind.css
git commit -m "feat(frontend): add background and overlay semantic color tokens"
```

---

## Phase 2: Fix Shared UI Components

### Task 2: Fix `dialog.tsx` — replace hardcoded colors

**Files:**
- Modify: `frontend/src/react/components/ui/dialog.tsx`

**Step 1: Replace `bg-black/50` with `bg-overlay/50`**

Line 20: `"fixed inset-0 z-50 bg-black/50"` → `"fixed inset-0 z-50 bg-overlay/50"`

**Step 2: Replace `bg-white` with `bg-background`**

Line 42: `"rounded-sm bg-white shadow-lg"` → `"rounded-sm bg-background shadow-lg"`

**Step 3: Replace `text-gray-500` with `text-control-light`**

Line 77: `"text-sm text-gray-500"` → `"text-sm text-control-light"`

**Step 4: Verify**

Run: `pnpm --dir frontend type-check`

**Step 5: Commit**

```bash
git add frontend/src/react/components/ui/dialog.tsx
git commit -m "fix(frontend): use semantic tokens in Dialog component"
```

### Task 3: Fix `select.tsx` — replace hardcoded colors and use `size-*`

**Files:**
- Modify: `frontend/src/react/components/ui/select.tsx`

**Step 1: Replace `bg-white` with `bg-background`**

Line 20 (`SelectTrigger`): `bg-white` → `bg-background`
Line 51 (`SelectContent`): `bg-white` → `bg-background`

**Step 2: Replace `w-3.5 h-3.5` with `size-3.5`**

Line 29: `className="w-3.5 h-3.5 opacity-50 shrink-0"` → `className="size-3.5 opacity-50 shrink-0"`
Line 82: `className="w-3.5 h-3.5"` → `className="size-3.5"`

**Step 3: Verify**

Run: `pnpm --dir frontend type-check`

**Step 4: Commit**

```bash
git add frontend/src/react/components/ui/select.tsx
git commit -m "fix(frontend): use semantic tokens and size shorthand in Select"
```

### Task 4: Fix `combobox.tsx` — replace hardcoded colors

**Files:**
- Modify: `frontend/src/react/components/ui/combobox.tsx`

**Step 1: Replace all `bg-white` with `bg-background`**

Lines 328, 365: `bg-white` → `bg-background`

**Step 2: Replace any `w-N h-N` pairs with `size-N`**

Scan for matching w/h pairs and replace.

**Step 3: Verify**

Run: `pnpm --dir frontend type-check`

**Step 4: Commit**

```bash
git add frontend/src/react/components/ui/combobox.tsx
git commit -m "fix(frontend): use semantic tokens in Combobox"
```

### Task 5: Fix `switch.tsx` — replace hardcoded colors

**Files:**
- Modify: `frontend/src/react/components/ui/switch.tsx`

**Step 1: Replace `bg-white` with `bg-background`**

Line 37: `bg-white` → `bg-background`

**Step 2: Verify & commit**

```bash
git add frontend/src/react/components/ui/switch.tsx
git commit -m "fix(frontend): use semantic tokens in Switch"
```

### Task 6: Fix `alert.tsx` — use `size-*` shorthand

**Files:**
- Modify: `frontend/src/react/components/ui/alert.tsx`

**Step 1: Replace `h-5 w-5` with `size-5`**

Line 52: `className="h-5 w-5 shrink-0 mt-0.5"` → `className="size-5 shrink-0 mt-0.5"`

**Step 2: Verify & commit**

```bash
git add frontend/src/react/components/ui/alert.tsx
git commit -m "fix(frontend): use size shorthand in Alert icon"
```

---

## Phase 3: Fix `space-x/y` → `gap` (23 instances, 7 files)

### Task 7: Fix spacing in `DatabaseObjectExplorer.tsx`

**Files:**
- Modify: `frontend/src/react/pages/project/database-detail/overview/DatabaseObjectExplorer.tsx`

**Step 1: Replace all `space-y-*` with `flex flex-col gap-*`**

For each occurrence (lines 357, 386, 414, 421, 441, 451, 460, 470, 476, 486):
- If element already has `flex flex-col`: just replace `space-y-N` with `gap-N`
- If element is a plain `div`/`section`: add `flex flex-col` and replace `space-y-N` with `gap-N`

Example: `<section className="space-y-4">` → `<section className="flex flex-col gap-4">`

**Step 2: Verify**

Run: `pnpm --dir frontend type-check`

**Step 3: Commit**

```bash
git add frontend/src/react/pages/project/database-detail/overview/DatabaseObjectExplorer.tsx
git commit -m "fix(frontend): replace space-y with flex gap in DatabaseObjectExplorer"
```

### Task 8: Fix spacing in remaining 6 files

**Files:**
- Modify: `frontend/src/react/plugins/agent/components/ToolCallCard.tsx` (lines 225, 306)
- Modify: `frontend/src/react/plugins/agent/components/AgentChat.tsx` (line 95)
- Modify: `frontend/src/react/pages/settings/PurchaseSection.tsx` (lines 366, 608)
- Modify: `frontend/src/react/pages/settings/RiskAssessmentPage.tsx` (lines 17, 31)
- Modify: `frontend/src/react/pages/settings/DataClassificationPage.tsx` (lines 333, 606)
- Modify: `frontend/src/react/pages/project/database-detail/revision/CreateRevisionDialog.tsx` (lines 109, 110, 140, 163)

**Step 1: Apply the same pattern as Task 7**

For each `space-y-N` or `space-x-N`:
- Replace with `flex flex-col gap-N` (for space-y) or `flex gap-N` (for space-x)
- If element already has `flex`: just swap the spacing utility

**Step 2: Verify**

Run: `pnpm --dir frontend type-check`

**Step 3: Commit**

```bash
git add frontend/src/react/plugins/agent/components/ToolCallCard.tsx \
       frontend/src/react/plugins/agent/components/AgentChat.tsx \
       frontend/src/react/pages/settings/PurchaseSection.tsx \
       frontend/src/react/pages/settings/RiskAssessmentPage.tsx \
       frontend/src/react/pages/settings/DataClassificationPage.tsx \
       frontend/src/react/pages/project/database-detail/revision/CreateRevisionDialog.tsx
git commit -m "fix(frontend): replace space-x/y with flex gap across React pages"
```

---

## Phase 4: Fix Raw Colors in Page-Level Code

**Strategy:** Work area by area. For each file, replace raw Tailwind colors with the closest semantic token:

| Raw Color | Semantic Token |
|-----------|---------------|
| `bg-white` | `bg-background` |
| `bg-black/50`, `bg-black/40`, `bg-black/30` | `bg-overlay/50`, `bg-overlay/40`, `bg-overlay/30` |
| `bg-gray-50`, `bg-gray-100` | `bg-control-bg` |
| `bg-gray-200` | `bg-control-bg-hover` |
| `text-gray-400` | `text-control-placeholder` |
| `text-gray-500`, `text-gray-600` | `text-control-light` |
| `text-gray-700`, `text-gray-800`, `text-gray-900` | `text-control` or `text-main` |
| `border-gray-200` | `border-block-border` |
| `border-gray-300` | `border-control-border` |
| `bg-blue-500`, `bg-blue-600`, `text-blue-600` | `bg-info` / `text-info` or `bg-accent` / `text-accent` (context-dependent) |
| `bg-red-500`, `bg-red-600`, `text-red-500`, `text-red-600` | `bg-error` / `text-error` |
| `bg-green-500`, `bg-green-600`, `text-green-600` | `bg-success` / `text-success` |
| `bg-yellow-500`, `text-yellow-600` | `bg-warning` / `text-warning` |
| `bg-indigo-*`, `text-indigo-*` | `bg-accent` / `text-accent` variants |

**Important:** Some raw colors are intentional (e.g., syntax highlighting, status indicators with specific shades). Use judgment — if a color conveys semantic meaning (error, warning, info, success, accent), use the token. If it's decorative or specific to a visualization, leave it.

### Task 9: Fix raw colors in Agent plugin components

**Files:**
- Modify: `frontend/src/react/plugins/agent/components/AgentWindow.tsx`
- Modify: `frontend/src/react/plugins/agent/components/AgentChat.tsx`
- Modify: `frontend/src/react/plugins/agent/components/AgentInput.tsx`
- Modify: `frontend/src/react/plugins/agent/components/ToolCallCard.tsx`

**Step 1: Read each file and identify all raw color classes**

**Step 2: Replace with semantic tokens per the mapping table above**

**Step 3: Verify**

Run: `pnpm --dir frontend type-check`

**Step 4: Commit**

```bash
git add frontend/src/react/plugins/agent/
git commit -m "fix(frontend): use semantic color tokens in Agent plugin components"
```

### Task 10: Fix raw colors in Drawer/Dialog components

**Files:**
- Modify: `frontend/src/react/components/database/CreateDatabaseDrawer.tsx`
- Modify: `frontend/src/react/components/database/TransferProjectDrawer.tsx`
- Modify: `frontend/src/react/components/database/LabelEditorDrawer.tsx`
- Modify: `frontend/src/react/components/EditEnvironmentDrawer.tsx`
- Modify: `frontend/src/react/components/CreateWorkloadIdentityDrawer.tsx`

**Step 1: Replace `bg-black/50` and `bg-black/30` with `bg-overlay/50` and `bg-overlay/30`**

**Step 2: Replace `bg-white` with `bg-background`**

**Step 3: Verify & commit**

```bash
git add frontend/src/react/components/database/CreateDatabaseDrawer.tsx \
       frontend/src/react/components/database/TransferProjectDrawer.tsx \
       frontend/src/react/components/database/LabelEditorDrawer.tsx \
       frontend/src/react/components/EditEnvironmentDrawer.tsx \
       frontend/src/react/components/CreateWorkloadIdentityDrawer.tsx
git commit -m "fix(frontend): use semantic tokens in Drawer/Dialog components"
```

### Task 11: Fix raw colors in shared components

**Files:**
- Modify: `frontend/src/react/components/AuditLogTable.tsx`
- Modify: `frontend/src/react/components/IssueTable.tsx`
- Modify: `frontend/src/react/components/ExprEditor.tsx`
- Modify: `frontend/src/react/components/AdvancedSearch.tsx`
- Modify: `frontend/src/react/components/AccountMultiSelect.tsx`
- Modify: `frontend/src/react/components/DatabaseGroupForm.tsx`
- Modify: `frontend/src/react/components/WorkspaceSwitcher.tsx`
- Modify: `frontend/src/react/components/TimeRangePicker.tsx`
- Modify: `frontend/src/react/components/MatchedDatabaseView.tsx`
- Modify: `frontend/src/react/components/sql-review/Panels.tsx`
- Modify: `frontend/src/react/components/sql-review/ReviewCreation.tsx`
- Modify: `frontend/src/react/components/CustomApproval/ApprovalStepsTable.tsx`
- Modify: `frontend/src/react/components/task-run-log/SectionHeader.tsx`
- Modify: `frontend/src/react/components/instance/DataSourceForm.tsx`
- Modify: `frontend/src/react/components/instance/InstanceSyncButton.tsx`
- Modify: `frontend/src/react/components/instance/InstanceFormButtons.tsx`
- Modify: `frontend/src/react/components/instance/InstanceActionDropdown.tsx`
- Modify: `frontend/src/react/components/instance/InfoPanel.tsx`
- Modify: `frontend/src/react/components/instance/InstanceFormBody.tsx`
- Modify: `frontend/src/react/components/instance/SslCertificateForm.tsx`

**Step 1: Read each file and replace raw colors per mapping table**

Focus on: `bg-white` → `bg-background`, `bg-black/*` → `bg-overlay/*`, gray scale → semantic tokens, status colors → `error`/`warning`/`success`/`info` tokens.

**Step 2: Verify**

Run: `pnpm --dir frontend type-check`

**Step 3: Commit**

```bash
git add frontend/src/react/components/
git commit -m "fix(frontend): use semantic color tokens in shared React components"
```

### Task 12: Fix raw colors in Settings pages

**Files:**
- Modify: `frontend/src/react/pages/settings/MembersPage.tsx`
- Modify: `frontend/src/react/pages/settings/IDPDetailPage.tsx`
- Modify: `frontend/src/react/pages/settings/ServiceAccountsPage.tsx`
- Modify: `frontend/src/react/pages/settings/ProjectsPage.tsx`
- Modify: `frontend/src/react/pages/settings/InstancesPage.tsx`
- Modify: `frontend/src/react/pages/settings/SemanticTypesPage.tsx`
- Modify: `frontend/src/react/pages/settings/CreateInstancePage.tsx`
- Modify: `frontend/src/react/pages/settings/SQLReviewDetailPage.tsx`
- Modify: `frontend/src/react/pages/settings/PurchaseSection.tsx`
- Modify: `frontend/src/react/pages/settings/DataClassificationPage.tsx`
- Modify: `frontend/src/react/pages/settings/general/GeneralSection.tsx`
- Modify: `frontend/src/react/pages/settings/general/AIAugmentationSection.tsx`
- Modify: `frontend/src/react/pages/settings/general/GeneralPage.tsx`

**Step 1: Read each file and replace raw colors per mapping table**

**Step 2: Verify**

Run: `pnpm --dir frontend type-check`

**Step 3: Commit**

```bash
git add frontend/src/react/pages/settings/
git commit -m "fix(frontend): use semantic color tokens in Settings pages"
```

### Task 13: Fix raw colors in Project pages

**Files:**
- Modify: `frontend/src/react/pages/project/ProjectDataExportPage.tsx`
- Modify: `frontend/src/react/pages/project/ProjectSettingsPage.tsx`
- Modify: `frontend/src/react/pages/project/ProjectWebhooksPage.tsx`
- Modify: `frontend/src/react/pages/project/ProjectReleaseDashboardPage.tsx`
- Modify: `frontend/src/react/pages/project/ProjectMaskingExemptionCreatePage.tsx`
- Modify: `frontend/src/react/pages/project/ProjectPlanDashboardPage.tsx`
- Modify: `frontend/src/react/pages/project/ProjectWebhookForm.tsx`
- Modify: `frontend/src/react/pages/project/ProjectSyncSchemaPage.tsx`
- Modify: `frontend/src/react/pages/project/DatabaseChangelogDetailPage.tsx`
- Modify: `frontend/src/react/pages/project/ProjectDatabasesPage.tsx`
- Modify: `frontend/src/react/pages/project/ProjectMaskingExemptionPage.tsx`
- Modify: `frontend/src/react/pages/project/ProjectDatabaseGroupDetailPage.tsx`
- Modify: `frontend/src/react/pages/project/ProjectDatabaseGroupsPage.tsx`
- Modify: `frontend/src/react/pages/project/export-center/DataExportPrepDrawer.tsx`
- Modify: `frontend/src/react/pages/project/database-detail/changelog/DatabaseChangelogTable.tsx`
- Modify: `frontend/src/react/pages/project/database-detail/DatabaseExportSchemaButton.tsx`
- Modify: `frontend/src/react/pages/project/database-detail/overview/TableDetailDialog.tsx`
- Modify: `frontend/src/react/pages/project/database-detail/overview/ObjectSectionTable.tsx`
- Modify: `frontend/src/react/pages/project/database-detail/overview/DatabaseObjectExplorer.tsx`
- Modify: `frontend/src/react/pages/project/database-detail/overview/TableMetadataTable.tsx`
- Modify: `frontend/src/react/pages/project/database-detail/revision/DatabaseRevisionTable.tsx`

**Step 1: Read each file and replace raw colors per mapping table**

**Step 2: Verify**

Run: `pnpm --dir frontend type-check`

**Step 3: Commit**

```bash
git add frontend/src/react/pages/project/
git commit -m "fix(frontend): use semantic color tokens in Project pages"
```

---

## Phase 5: Fix Legacy CSS Utility Classes

### Task 14: Fix raw colors in `tailwind.css` utility classes

**Files:**
- Modify: `frontend/src/assets/css/tailwind.css`

**Step 1: Replace raw colors in legacy utility classes**

- Line 238: `.textlabeltip` — `text-red-500` → `text-error`
- Line 242: `.textfield` — `bg-gray-50` → `bg-control-bg`
- Line 266: `.radio .label` — `text-gray-700` → `text-control`
- Line 298: `.normal-link` — `text-blue-600` → `text-info`, `text-blue-800` → `text-info-hover`
- Line 302: `.light-link` — `text-blue-400` / `text-blue-200` → keep (dark theme specific)

**Step 2: Verify & commit**

```bash
git add frontend/src/assets/css/tailwind.css
git commit -m "fix(frontend): use semantic tokens in legacy CSS utility classes"
```

---

## Verification Checklist

After all tasks are complete:

1. `pnpm --dir frontend type-check` — passes
2. `pnpm --dir frontend check` — no lint errors
3. `pnpm --dir frontend dev` — visual spot-check that colors haven't changed
4. Search for remaining violations:
   - `rg 'space-[xy]-' frontend/src/react/ --glob '*.tsx'` — should return 0 results
   - `rg 'bg-white|bg-black' frontend/src/react/components/ui/` — should return 0 results
   - `rg 'bg-(gray|blue|red|green|yellow|orange)-\d' frontend/src/react/ --glob '*.tsx'` — should be minimal (only intentional decorative uses)

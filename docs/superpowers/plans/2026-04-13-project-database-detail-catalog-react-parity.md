# Project Database Detail Catalog React Parity Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Remove the Vue bridge from the project database detail catalog tab and restore full catalog parity with a React-native implementation built from the repo's shadcn-style React primitives.

**Architecture:** Reintroduce a React-owned `DatabaseCatalogPanel` that orchestrates catalog flattening, permissions, feature gating, row selection, inline edits, delete cleanup, and grant-access dialog state. Keep the UI split into a page-specific panel shell, a presentational `SensitiveColumnTable`, and a focused `GrantAccessDialog`, all backed by existing Vue/Pinia stores through `useVueState`.

**Tech Stack:** React, TypeScript, `useVueState`, Pinia stores, protobuf-generated types, shadcn-style UI components on `@base-ui/react`, Tailwind CSS v4, Vitest

---

### Task 1: Restore the React catalog panel shell

**Files:**
- Modify: `frontend/src/react/pages/project/database-detail/panels/DatabaseCatalogPanel.tsx`
- Reference: `frontend/src/components/Database/DatabaseSensitiveDataPanel.vue`
- Reference: `frontend/src/components/SensitiveData/types.ts`
- Reference: `frontend/src/components/SensitiveData/utils.ts`

- [ ] **Step 1: Replace the Vue bridge with a React implementation**

Rework `DatabaseCatalogPanel.tsx` so it no longer imports `createApp`, `NConfigProvider`, `OverlayStackManager`, or `DatabaseSensitiveDataPanel.vue`. Use the earlier React panel from commit `79dd009d2c` as the behavioral baseline, but keep the current branch's import style and file structure.

The panel should own:

- catalog reads via `useDatabaseCatalog(database.name, false)`
- reactive store reads via `useVueState`
- flattening of relational columns, object-schema fields, and table classifications into `MaskData[]`
- local state for search text, checked rows, feature dialog, and grant-access dialog
- permission checks for catalog updates and masking exemption creation
- setting fetches for semantic types and data classification
- handlers for semantic type edits, classification edits, delete cleanup, and grant-access flow opening

Keep the helper logic inside this file unless a helper becomes reused by both tests and the table component.

- [ ] **Step 2: Preserve the current mutation semantics**

Implement the same write paths used by the Vue panel:

- semantic type changes mutate the row target and persist with `updateDatabaseCatalog`
- classification changes mutate the row target and persist with `updateDatabaseCatalog`
- delete clears semantic type and/or classification, persists the catalog, removes the row from the local checked selection, then removes matching masking exemptions from the project policy
- grant access opens the React dialog only when rows are selected and permissions/feature gates allow it

Continue using:

- `useDatabaseCatalogV1Store()`
- `usePolicyV1Store()`
- `useSettingV1Store()`
- `featureToRef(PlanFeature.FEATURE_DATA_MASKING)`
- `instanceV1MaskingForNoSQL(...)`

- [ ] **Step 3: Keep parity for NoSQL and feature gating**

Carry forward the Vue panel's NoSQL branching:

- object-schema flattening must still surface nested fields
- row selection must be disabled when `instanceV1MaskingForNoSQL(...)` says bulk grant access is unavailable
- rows with unsupported semantic type or classification targets must stay read-only
- missing masking subscription should open the feature dialog instead of the grant-access dialog

- [ ] **Step 4: Run focused type-check for the panel**

Run:

```bash
pnpm --dir frontend type-check
```

- [ ] **Step 5: Commit**

```bash
git add frontend/src/react/pages/project/database-detail/panels/DatabaseCatalogPanel.tsx
git commit -m "feat(frontend): restore react database catalog panel"
```

---

### Task 2: Recreate the catalog UI components with shadcn-style primitives

**Files:**
- Create: `frontend/src/react/pages/project/database-detail/catalog/SensitiveColumnTable.tsx`
- Create: `frontend/src/react/pages/project/database-detail/catalog/GrantAccessDialog.tsx`
- Reference: `frontend/src/components/SensitiveData/components/SensitiveColumnTable.vue`
- Reference: `frontend/src/components/SensitiveData/exemptionDataUtils.ts`
- Reference: `frontend/src/components/SensitiveData/utils.ts`

- [ ] **Step 1: Create `SensitiveColumnTable.tsx`**

Port the old React `SensitiveColumnTable` back into the tree, but align it with the repo's current React UI conventions instead of copying raw markup unchanged.

Requirements:

- use the shared React `Table` primitives if they fit cleanly; otherwise keep a narrow custom table wrapper only for what the shared table cannot express
- use existing `Button` and `Select` components
- preserve checkbox row selection, inline semantic type editing, inline classification editing, remove action, and empty state
- show schema-qualified table names and `-` placeholders consistently with the Vue panel
- avoid Naive UI and Vue cell components

Keep the component presentational:

- no direct store calls
- all mutations bubble up through callbacks
- selected rows are controlled by props

- [ ] **Step 2: Create `GrantAccessDialog.tsx`**

Restore the earlier React grant-access dialog and keep its policy-writing logic intact.

Requirements:

- preview selected database resources
- capture reason, expiration, and members
- generate the masking exemption expression with `rewriteResourceDatabase(...)`
- append the new exemption to the existing project masking exemption policy
- use the project's dialog, input, button, and form-layout patterns

Do not reintroduce the Vue drawer or a bridge wrapper.

- [ ] **Step 3: Wire both components into the panel**

Update `DatabaseCatalogPanel.tsx` to use the new components and verify:

- `SensitiveColumnTable` receives checked rows, filtered rows, option lists, and callbacks
- `GrantAccessDialog` receives the selected rows converted to the `SensitiveColumn` shape expected by the shared utilities
- closing the dialog clears checked rows only when the current Vue behavior does so

- [ ] **Step 4: Run fixers and type-check**

Run:

```bash
pnpm --dir frontend fix
pnpm --dir frontend type-check
```

- [ ] **Step 5: Commit**

```bash
git add frontend/src/react/pages/project/database-detail/catalog/SensitiveColumnTable.tsx frontend/src/react/pages/project/database-detail/catalog/GrantAccessDialog.tsx frontend/src/react/pages/project/database-detail/panels/DatabaseCatalogPanel.tsx
git commit -m "feat(frontend): restore react catalog interactions"
```

---

### Task 3: Rebuild and expand the React test coverage

**Files:**
- Create: `frontend/src/react/pages/project/database-detail/panels/DatabaseCatalogPanel.test.tsx`
- Modify: `frontend/src/react/pages/project/ProjectDatabaseDetailPage.test.tsx`

- [ ] **Step 1: Restore panel-level tests**

Recreate `DatabaseCatalogPanel.test.tsx` using the earlier React test file from commit `79dd009d2c` as the baseline. Mock the same stores and utilities, then keep or add coverage for:

- rendering flattened relational rows
- selection enabling the grant-access button
- missing masking feature opening the feature dialog
- permission inputs passed to the guard
- deleting a selected row clearing it from the checked selection
- inline semantic type and classification updates
- undefined catalog state before the store has loaded

- [ ] **Step 2: Add parity tests that were easy to miss in the old version**

Add explicit tests for:

- NoSQL mode disables row selection and still renders flattened object-schema rows
- delete path removes matching masking exemptions from the policy store
- grant-access submit appends to existing exemptions instead of replacing them
- action controls are disabled or hidden when `bb.databaseCatalogs.update` is missing

These are the most likely parity regressions after restoring older React code.

- [ ] **Step 3: Keep the page-level integration test minimal**

Do not move detailed workflow assertions into `ProjectDatabaseDetailPage.test.tsx`. Keep that file focused on:

- tab routing
- permission guard behavior for the catalog tab
- rendering the catalog panel when the tab is selected

Only update it if the import path, test id expectations, or mounting assumptions change.

- [ ] **Step 4: Run the targeted tests**

Run:

```bash
pnpm --dir frontend test -- --run frontend/src/react/pages/project/database-detail/panels/DatabaseCatalogPanel.test.tsx frontend/src/react/pages/project/ProjectDatabaseDetailPage.test.tsx
```

- [ ] **Step 5: Commit**

```bash
git add frontend/src/react/pages/project/database-detail/panels/DatabaseCatalogPanel.test.tsx frontend/src/react/pages/project/ProjectDatabaseDetailPage.test.tsx
git commit -m "test(frontend): cover react database catalog parity"
```

---

### Task 4: Verify, remove bridge assumptions, and finish

**Files:**
- Modify: `frontend/src/react/pages/project/database-detail/panels/DatabaseCatalogPanel.tsx`
- Modify: `frontend/src/react/pages/project/database-detail/catalog/SensitiveColumnTable.tsx`
- Modify: `frontend/src/react/pages/project/database-detail/catalog/GrantAccessDialog.tsx`
- Modify: `frontend/src/react/pages/project/database-detail/panels/DatabaseCatalogPanel.test.tsx`

- [ ] **Step 1: Remove all page-local Vue bridge dependencies**

Confirm the final catalog implementation no longer imports or depends on:

- `createApp`
- `NConfigProvider`
- `themeOverrides`
- `OverlayStackManager.vue`
- `DatabaseSensitiveDataPanel.vue`
- `NaiveUI`

The catalog tab should now be fully React-owned even though it still reads from Vue-backed stores.

- [ ] **Step 2: Run full frontend verification**

Run:

```bash
pnpm --dir frontend fix
pnpm --dir frontend check
pnpm --dir frontend type-check
pnpm --dir frontend test -- --run frontend/src/react/pages/project/database-detail/panels/DatabaseCatalogPanel.test.tsx frontend/src/react/pages/project/ProjectDatabaseDetailPage.test.tsx
```

- [ ] **Step 3: Sanity-check the diff**

Run:

```bash
git diff -- frontend/src/react/pages/project/database-detail/panels/DatabaseCatalogPanel.tsx frontend/src/react/pages/project/database-detail/catalog/SensitiveColumnTable.tsx frontend/src/react/pages/project/database-detail/catalog/GrantAccessDialog.tsx frontend/src/react/pages/project/database-detail/panels/DatabaseCatalogPanel.test.tsx frontend/src/react/pages/project/ProjectDatabaseDetailPage.test.tsx
```

Verify the final diff shows a React-native catalog flow and no accidental changes outside the catalog scope.

- [ ] **Step 4: Commit final polish**

```bash
git add frontend/src/react/pages/project/database-detail/panels/DatabaseCatalogPanel.tsx frontend/src/react/pages/project/database-detail/catalog/SensitiveColumnTable.tsx frontend/src/react/pages/project/database-detail/catalog/GrantAccessDialog.tsx frontend/src/react/pages/project/database-detail/panels/DatabaseCatalogPanel.test.tsx frontend/src/react/pages/project/ProjectDatabaseDetailPage.test.tsx
git commit -m "fix(frontend): finish react database catalog parity"
```

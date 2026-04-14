# Project Database Detail Catalog React Parity Design

## Summary

Remove the Vue bridge from the project database detail page's catalog tab and replace it with a fully React-owned implementation that preserves the current catalog workflow.

The finished state must not mount `frontend/src/components/Database/DatabaseSensitiveDataPanel.vue` from React. React should own the panel, table interactions, dialogs, and state orchestration while continuing to use the existing shared stores and protobuf types.

## Goals

- Remove the `createApp(...)` bridge in `frontend/src/react/pages/project/database-detail/panels/DatabaseCatalogPanel.tsx`.
- Keep feature parity with the current Vue-backed catalog panel on the project database detail page.
- Preserve existing permission, feature-gating, and policy-update behavior.
- Keep the page route, hash tab behavior, and surrounding React page shell unchanged.
- Reuse existing store modules and shared sensitive-data utilities instead of inventing a parallel data layer.
- Use the repo's existing React UI system: shadcn-style components built on `@base-ui/react` and Tailwind v4 tokens.

## Non-Goals

- Do not migrate unrelated database detail tabs.
- Do not rewrite Vue stores or policy services into React-only state.
- Do not broaden this work into a generic React replacement for every sensitive-data surface in the app.
- Do not redesign the catalog workflow.

## Current State

The catalog tab in `frontend/src/react/pages/project/ProjectDatabaseDetailPage.tsx` renders `DatabaseCatalogPanel`, but that component currently mounts a separate Vue app:

- `frontend/src/react/pages/project/database-detail/panels/DatabaseCatalogPanel.tsx`
- mounts `frontend/src/components/Database/DatabaseSensitiveDataPanel.vue`
- wraps it in `NConfigProvider` and `OverlayStackManager`

That bridge keeps the route shell in React but leaves the catalog workflow owned by Vue. It carries the same integration risks seen elsewhere in the migration:

- duplicated provider trees
- bridge-only lifecycle management
- stale props risk on database navigation
- reduced React test coverage for the actual workflow

Recent history shows this page already had a React-native catalog implementation and later regressed to the Vue bridge to restore parity quickly. That earlier React implementation is useful source material, but it needs to be treated as a baseline to restore and harden rather than copied blindly.

## Required Parity

The React catalog panel must preserve all of these user-facing behaviors from `frontend/src/components/Database/DatabaseSensitiveDataPanel.vue`:

- feature attention banner for masking
- search/filter across schema, table, and column names
- row selection for bulk grant access
- grant-access button gating based on permissions and subscription state
- feature paywall dialog when masking is unavailable
- inline semantic type editing where catalog update is allowed
- inline classification editing where catalog update is allowed
- deletion/removal of sensitive markings
- masking exemption cleanup when a sensitive entry is removed
- NoSQL-specific behavior differences, especially around row selection and editing
- i18n-backed display strings only

## Architecture

The catalog migration should stay split into three React units:

### 1. `DatabaseCatalogPanel.tsx`

This is the orchestration layer for the catalog tab.

Responsibilities:

- read the database catalog through the existing store APIs
- flatten catalog data into table rows
- own local UI state:
  - search text
  - checked rows
  - feature dialog open state
  - grant-access dialog open state
- fetch and read semantic type and data classification settings
- evaluate permission and feature gates
- apply mutations for:
  - semantic type changes
  - classification changes
  - removal of sensitive markings
  - masking exemption cleanup

This component should remain page-specific. It does not need to become a shared sensitive-data framework.

### 2. `catalog/SensitiveColumnTable.tsx`

This is the presentational table for the flattened sensitive-data rows.

Responsibilities:

- render the rows
- render selection controls when row selection is enabled
- render inline semantic type and classification controls
- render remove actions
- render empty state
- emit callbacks for edits, delete, and selection changes

The table should be React-native and should not depend on Naive UI or Vue cells. It should be composed from the existing React UI primitives already used elsewhere in this migration, such as the shared `Table`, `Button`, `Select`, `Badge`, and empty-state patterns, rather than bespoke styled markup. It may keep a simpler internal structure than the Vue `NDataTable` implementation as long as behavior parity is preserved.

### 3. `catalog/GrantAccessDialog.tsx`

This owns the masking exemption creation flow for the selected rows.

Responsibilities:

- preview selected database resources
- collect reason, expiration, and members
- build the masking exemption expression
- upsert the project masking exemption policy
- report success/failure and dismiss cleanly

This dialog can reuse the earlier React implementation structure because it is already close to the existing Vue behavior and has a clear boundary. The form layout should follow the project's shadcn-style composition patterns instead of ad hoc spacing wrappers where practical.

## Data Flow

The React panel should continue to use the existing shared data sources rather than introducing a new fetch layer:

- `useDatabaseCatalog(database.name, false)`
- `useDatabaseCatalogV1Store()`
- `usePolicyV1Store()`
- `useSettingV1Store()`
- `featureToRef(...)`
- `useVueState(...)` for reactive store reads

The panel should derive a flattened row list from the database catalog:

- relational table columns with semantic type/classification
- object-schema leaves for NoSQL/object-based catalogs
- table-level classifications

That flattening logic should live in React-side helpers close to the panel so the display model is explicit and testable.

## Mutation Rules

The React implementation must preserve the current write semantics:

- semantic type changes mutate the selected target and persist through `updateDatabaseCatalog`
- classification changes mutate the selected target and persist through `updateDatabaseCatalog`
- removing a sensitive entry clears semantic type and/or classification on the target, persists the catalog update, then removes matching masking exemptions from the project policy
- grant access appends new exemptions to the project masking exemption policy rather than replacing unrelated entries

The plan should explicitly test these mutation paths, because this is where parity regressions are most likely.

## NoSQL Handling

The Vue implementation has special handling for NoSQL instances and object-schema catalogs. The React version must keep those branch behaviors:

- detect NoSQL masking mode with `instanceV1MaskingForNoSQL(...)`
- disable row selection when bulk grant access should not be available
- preserve object-schema flattening for nested fields
- prevent unsupported classification or semantic-type edits on targets that do not support them

This is a required part of parity, not a follow-up enhancement.

## File Plan

Primary files to modify or restore:

- `frontend/src/react/pages/project/database-detail/panels/DatabaseCatalogPanel.tsx`
- `frontend/src/react/pages/project/database-detail/panels/DatabaseCatalogPanel.test.tsx`
- `frontend/src/react/pages/project/database-detail/catalog/SensitiveColumnTable.tsx`
- `frontend/src/react/pages/project/database-detail/catalog/GrantAccessDialog.tsx`

Reference files to verify behavior against:

- `frontend/src/components/Database/DatabaseSensitiveDataPanel.vue`
- `frontend/src/components/SensitiveData/components/SensitiveColumnTable.vue`
- `frontend/src/components/SensitiveData/types.ts`
- `frontend/src/components/SensitiveData/utils.ts`
- `frontend/src/components/SensitiveData/exemptionDataUtils.ts`

## Testing Strategy

The React migration should be validated with focused React tests rather than relying on the page shell test alone.

Minimum coverage:

- panel renders flattened rows from catalog data
- search filters by schema/table/column
- missing masking feature opens the feature dialog instead of the grant-access flow
- row selection enables and clears grant-access state correctly
- semantic type edit persists catalog updates
- classification edit persists catalog updates
- delete clears catalog data and removes masking exemptions
- NoSQL mode disables selection and preserves object-schema rows
- permission gating disables or hides actions as expected

The existing page-level test in `frontend/src/react/pages/project/ProjectDatabaseDetailPage.test.tsx` should continue to verify that the catalog tab is routed and permission-guarded correctly, but the detailed workflow tests belong in the panel test file.

## Risks

- The old React implementation may be functionally close but visually or behaviorally behind the current Vue panel in small ways.
- Inline editing and delete cleanup can regress silently if tests only assert rendered text.
- Classification and semantic type settings are loaded through Vue-backed stores, so tests need to mock those stores carefully.
- NoSQL behavior is easy to miss if tests only cover relational catalogs.

## Recommendation

Restore the previous React catalog implementation as the starting point, then reconcile it against the current Vue behavior and today's shared UI patterns. This is the fastest route to a bridge-free panel without accepting a greenfield rewrite's regression risk.

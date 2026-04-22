# Project Plan Detail React Migration

## Goal

Migrate `planRoutes` / `planDetailComponent` from Vue to React with Vue parity as the default target:

- layout parity
- interaction parity
- style parity
- no product-behavior changes without explicit confirmation

Vue reference implementation stays in place during the migration.

## Route Scope

- `PROJECT_V1_ROUTE_PLAN_DETAIL`
- `PROJECT_V1_ROUTE_PLAN_DETAIL_SPECS`
- `PROJECT_V1_ROUTE_PLAN_DETAIL_SPEC_DETAIL`
- legacy rollout redirects that resolve into the routes above

## Inventory

### Route Entry

- `frontend/src/router/dashboard/projectV1.ts`
- `frontend/src/router/dashboard/projectV1RouteHelpers.ts`

### Vue Page Entry

- `frontend/src/components/Plan/components/PlanDetailPage/PlanDetailLayout.vue`
- `frontend/src/components/Plan/components/PlanDetailPage/PlanDetailPage.vue`
- `frontend/src/components/Plan/components/PlanDetailPage/PlanDetailHeader.vue`

### Page-Level Vue Logic

- `frontend/src/components/Plan/logic/context.ts`
- `frontend/src/components/Plan/logic/base.ts`
- `frontend/src/components/Plan/logic/sidebar.ts`
- `frontend/src/components/Plan/logic/useEditorState.ts`
- `frontend/src/components/Plan/logic/useNavigationGuard.ts`
- `frontend/src/components/Plan/PollerProvider.vue`
- `frontend/src/components/Plan/logic/poller/poller.ts`
- `frontend/src/components/Plan/logic/poller/utils.ts`
- `frontend/src/components/Plan/logic/initialize/index.ts`
- `frontend/src/components/Plan/logic/initialize/create.ts`
- `frontend/src/components/Plan/logic/initialize/util.ts`

### Page Structure / Layout

- `frontend/src/components/Plan/components/PlanDetailPage/usePlanDetailViewportVars.ts`
- `frontend/src/components/Plan/components/PlanDetailPage/usePlanDetailDesktopLayout.ts`
- `frontend/src/components/Plan/components/PlanDetailPage/useActivePhase.ts`
- `frontend/src/components/Plan/components/PlanDetailPage/usePhaseState.ts`
- `frontend/src/components/Plan/components/PlanDetailPage/usePhaseSummaries.ts`
- `frontend/src/components/Plan/components/PlanDetailPage/PlanDetailDesktopColumn.vue`
- `frontend/src/components/Plan/components/PlanDetailPage/StickySidePanel.vue`
- `frontend/src/layouts/common.ts`

### Changes Branch

- `frontend/src/components/Plan/components/PlanDetailPage/PlanSectionChanges.vue`
- `frontend/src/components/Plan/components/SpecDetailView/*.vue`
- `frontend/src/components/Plan/components/SpecDetailView/context.ts`
- `frontend/src/components/Plan/components/StatementSection/*.vue`
- `frontend/src/components/Plan/components/StatementSection/*.ts`
- `frontend/src/components/Plan/components/PlanCheckSection/PlanCheckSection.vue`
- `frontend/src/components/Plan/components/ChecksView/*.vue`
- `frontend/src/components/Plan/components/Configuration/**/*.vue`
- `frontend/src/components/Plan/components/Configuration/**/*.ts`
- `frontend/src/components/Plan/components/AddSpecDrawer.vue`
- `frontend/src/components/Plan/components/common/*.vue`
- `frontend/src/components/Plan/components/common/validateSpec.ts`

### Review Branch

- `frontend/src/components/Plan/components/PlanDetailPage/PlanSectionReview.vue`
- `frontend/src/components/Plan/components/IssueReviewView/Sidebar/ApprovalFlowSection/*.vue`
- `frontend/src/components/Plan/components/IssueReviewView/Sidebar/ApprovalFlowSection/*.ts`
- `frontend/src/store/modules/v1/issueComment.ts`

### Deploy Branch

- `frontend/src/components/Plan/components/PlanDetailPage/PlanSectionDeploy.vue`
- `frontend/src/components/Plan/components/PlanDetailPage/DeployFutureAction.vue`
- `frontend/src/components/Plan/components/PlanDetailPage/TaskDetailPanel.vue`
- `frontend/src/components/RolloutV1/components/Stage/*.vue`
- `frontend/src/components/RolloutV1/components/Stage/*.ts`
- `frontend/src/components/RolloutV1/components/Task/*.vue`
- `frontend/src/components/RolloutV1/components/Task/*.ts`
- `frontend/src/components/RolloutV1/components/TaskRun/**/*.vue`
- `frontend/src/components/RolloutV1/components/TaskRun/**/*.ts`
- `frontend/src/store/modules/v1/taskRunLog.ts`

### Sidebar / Metadata

- `frontend/src/components/Plan/components/PlanDetailPage/PlanMetadataSidebar.vue`
- `frontend/src/components/Plan/components/PlanCheckStatusCount.vue`
- `frontend/src/components/Plan/components/RefreshIndicator.vue`

### Shared High-Risk Capabilities

- Monaco editor and LSP:
  - `frontend/src/components/MonacoEditor/**`
  - `frontend/src/react/components/monaco/**`
- Markdown editor:
  - `frontend/src/components/MarkdownEditor/**`
- Schema editor:
  - `frontend/src/components/SchemaEditorLite/**`
- Drawer / table / button:
  - `frontend/src/components/v2/Container/**`
  - `frontend/src/react/components/ui/button.tsx`
  - `frontend/src/react/components/ui/sheet.tsx`
  - `frontend/src/react/components/ui/table.tsx`

## Migration Order

1. React route entry
2. Page skeleton
3. Content branches
4. Leaf components
5. Shared capability wiring
6. Visual parity pass

## Execution Checklist

- [x] Inventory Vue route/page tree to leaf nodes
- [x] Record non-goals and parity constraints
- [x] Switch route entry to React and keep Vue files as reference
- [x] Build React page skeleton: header/content/sidebar/detail-panel/scroll model
- [x] Recreate route-driven phase expansion and task detail panel state
- [x] Recreate initialize/load state for existing plan and create-plan flow
- [x] Recreate polling and manual refresh without assuming rollout/plan always exist
- [x] Migrate changes branch shell
- [x] Migrate review branch shell
- [x] Migrate deploy branch shell
- [ ] Migrate changes leaf components
- [ ] Migrate deploy/task detail leaf components
- [ ] Migrate sidebar metadata
- [ ] Migrate high-risk editor integrations deliberately
- [x] Run `pnpm --dir frontend fix`
- [x] Run `pnpm --dir frontend check`
- [x] Run `pnpm --dir frontend type-check`
- [x] Run `pnpm --dir frontend test`

## Explicit Non-Goals

- Delete Vue reference implementation
- Change global shared component behavior for this page migration
- Add new product capabilities outside Vue parity
- Optimize workflows that are not already present on the Vue page
- Treat Monaco / task-run-log / schema-editor as ordinary UI leaves

## Current Batch

Batch 1 target:

- switch route entry to React
- implement React skeleton and route/query-driven shell
- keep deeper content leaves as follow-up work

Batch 2 progress:

- changes branch now renders React spec tabs, targets, statement editor/release view, checks, and readonly option summary
- changes branch tabs/content container/all-targets drawer now follow Vue `SpecListSection` and `SpecDetailView` layout more closely
- create-plan statement editing now works against local sheet state
- navigation guard now respects editing scopes in the React page
- sidebar now shows created-by/time, status, checks summary, issue labels, rollout stages, and refresh state
- deploy task detail now shows stage/environment, target database, latest task run log, and task run history
- sidebar approval flow is now wired in React, including re-request review and candidate display
- task detail now supports task-level RUN / SKIP / CANCEL actions in React
- changes branch now supports add spec, delete spec, and target selector sheets in React
- changes branch readonly option summary is now partially editable in React
- deploy branch now supports stage-level RUN / SKIP / CANCEL actions in React
- task detail now includes a session tab and single-task rollback flow in React
- header now supports title/description editing, create plan, ready-for-review, and close/reopen in React
- create-plan change specs now support draft SQL checks in React
- review branch now uses split approval-flow composition so page review section and sidebar no longer fight over layout
- deploy branch has been extracted into `frontend/src/react/pages/project/plan-detail/components/deploy/`
- deploy stage/task shell now follows Vue structure more closely:
  - stage navigation bar
  - stage content view
  - task list
  - task toolbar
  - task item
  - stage content sidebar
- deploy desktop inline detail panel and mobile drawer now follow Vue desktop/drawer layout more closely
- pending tasks preview / create rollout flow is now wired in React
- task run history/session/rollback leaves now match Vue behavior more closely:
  - Postgres session tab visibility matches Vue
  - task run comment troubleshoot link is wired
  - rollback sheet supports multi-task preview/create flow
  - rollback create permission is enforced
- rollout/task status stringification is now safe to call from React without Vue `useI18n()` runtime errors

Batch 3 behavior parity pass:

- future review/deploy phase content now renders instead of only summary text
- phase collapsed summaries now use Vue-equivalent i18n summaries for changes, review, and deploy
- ready-for-review now matches Vue warning acknowledgement and post-create navigation behavior
- sidebar status mapping now follows Vue review/rollout status rules
- task row/detail quick actions now respect rollout permissions before displaying
- task row/detail statements now fetch missing sheets and show truncation hints

Changes branch parity pass:

- SQL advice markers are now wired into the React statement editor for existing-plan checks and create-plan draft checks
- draft SQL checks now expose formatted plan check runs back to the statement editor
- checks summary/grouping/draft transform logic has been extracted into tested React helpers
- target database-group inline/overflow behavior and options visibility logic have tested helpers
- options row spacing/label styling and target chip density have been brought closer to Vue
- selected spec fallback, all-target filtering, and database-group route params now use tested React helpers
- database-group target display now follows Vue's plain-name plus external-link behavior more closely
- persisted and draft checks now share the same tested grouping/summary helpers
- excluded by current instruction:
  - Schema Editor drawer
  - expanded editor modal

Batch 3 remaining focus:

- deploy visual parity pass against Vue tailwind
- remaining `StageNavigationBar` / `StageContentView` spacing rhythm
- remaining `TaskToolbar` / `TaskItem` / `StageTimeline` row-level parity
- final residual-gap sweep before calling deploy branch done

## Residual Gaps After Batch 1

- full editable header parity:
  - major interactions exist
  - visual/detail parity still pending
- full changes branch parity:
- create-plan SQL check parity is now present
- full checks/configuration parity
- deploy action parity
- deploy action parity:
  - task-level run/skip/cancel exists
  - stage-level run/skip/cancel exists
  - pending task preview/create rollout now exists
  - broader rollout actions still need final parity sweep
- task detail log/session/rollback parity:
  - session tab exists
  - single-task rollback exists
  - multi-task rollback flow now exists
  - task history/session tables still need final visual parity sweep
- metadata sidebar parity:
  - approval flow present, but still not pixel/interaction identical to Vue
- visual spacing/sticky behavior parity
- no Vue-to-React leaf replacement yet for Monaco / markdown / schema editor / task-run log

## Focused Plan: `PlanDetailChangesBranch.tsx`

This phase is scoped to the React changes branch only:

- `frontend/src/react/pages/project/plan-detail/components/PlanDetailChangesBranch.tsx`
- `frontend/src/react/pages/project/plan-detail/components/PlanDetailStatementSection.tsx`
- `frontend/src/react/pages/project/plan-detail/components/PlanDetailChecks.tsx`
- `frontend/src/react/pages/project/plan-detail/components/PlanDetailDraftChecks.tsx`

The goal is to finish this branch to Vue parity before moving back to header/sidebar/deploy cleanup.

### Vue References To Match

- `frontend/src/components/Plan/components/PlanDetailPage/PlanSectionChanges.vue`
- `frontend/src/components/Plan/components/SpecDetailView/SpecDetailView.vue`
- `frontend/src/components/Plan/components/SpecDetailView/SpecListSection.vue`
- `frontend/src/components/Plan/components/SpecDetailView/TargetListSection.vue`
- `frontend/src/components/Plan/components/SpecDetailView/TargetsSelectorDrawer.vue`
- `frontend/src/components/Plan/components/SpecDetailView/AllTargetsDrawer.vue`
- `frontend/src/components/Plan/components/SpecDetailView/DatabaseGroupTargetDisplay.vue`
- `frontend/src/components/Plan/components/StatementSection/StatementSection.vue`
- `frontend/src/components/Plan/components/StatementSection/EditorView/EditorView.vue`
- `frontend/src/components/Plan/components/StatementSection/ReleaseView/ReleaseView.vue`
- `frontend/src/components/Plan/components/PlanCheckSection/PlanCheckSection.vue`
- `frontend/src/components/Plan/components/SQLCheckV1Section/SQLCheckV1Section.vue`
- `frontend/src/components/Plan/components/IssueReviewView/DatabaseChangeView/OptionsSection.vue`

### Execution Order

1. Container parity
2. Targets parity
3. SQL parity
4. Checks parity
5. Options parity
6. Drawer/modal parity
7. Edge-case parity
8. Visual tailwind pass

### Detailed Checklist

- [ ] Tabs and container parity
  - match `SpecListSection.vue` line-tab rhythm exactly
  - match suffix action placement for `Add Change`
  - match content container padding/scrolling from `SpecDetailView.vue`
  - keep route-driven selected spec behavior identical

- [ ] Targets section parity
  - match title row typography and button size
  - match warning alert spacing and child row density
  - match target chip height, padding, font size, icon spacing, environment text color
  - finish database-group target parity:
    - header row
    - flat child chip layout
    - `n more` behavior
    - external link behavior
  - match empty/loading/view-all states

- [ ] SQL section parity
  - match title row typography, button sizing, and wrapping
  - match oversized-sheet warning spacing and action placement
  - match Monaco container border radius, height, and surrounding whitespace
  - match readonly/edit mode toolbar behavior
  - match release-view card spacing and typography
  - SQL advice markers are wired
  - excluded for now:
    - Schema Editor drawer
    - expanded editor modal
  - do not change Monaco/LSP behavior beyond parity needs

- [ ] Checks section parity
  - match title row spacing and run button size
  - match status badge density, icons, labels, and selected state
  - match affected-rows badge styling
  - match draft checks and persisted checks presentation separately
  - match checks drawer/sheet grouping, card spacing, and empty state

- [ ] Options section parity
  - match title style exactly
  - match per-item label color and baseline
  - match switch/select height and alignment
  - match horizontal/vertical spacing across wrap points
  - preserve current behavior for planless/create-plan/read-only/oversized-sheet cases

- [ ] Selector/drawer parity
  - match target selector tabs, table spacing, and footer button rhythm
  - match all-targets drawer width, header spacing, search spacing, and chip list density
  - avoid changing global `sheet`, `select`, or `button` behavior

- [ ] Edge-case verification
  - verify Vue behavior first for each suspected bug
  - verify create-plan vs existing-plan behavior separately
  - verify release-backed spec vs sheet-backed spec separately
  - verify database-group targets and plain database targets separately
  - verify no-plan / no-rollout / no-check-result states remain reachable and correct

### Acceptance Criteria

- Every section in the React changes branch maps 1:1 to the Vue section it replaces
- Section titles use the same hierarchy and color intent as Vue
- Control sizes and vertical rhythm are consistent across `targets`, `sql`, `checks`, and `options`
- Database-group targets use the same layout model as Vue, not an improvised tree UI
- Draft checks and persisted checks preserve their distinct Vue behavior
- No new product capability is added in this branch
- `pnpm --dir frontend fix`
- `pnpm --dir frontend check`
- `pnpm --dir frontend type-check`
- `pnpm --dir frontend test`

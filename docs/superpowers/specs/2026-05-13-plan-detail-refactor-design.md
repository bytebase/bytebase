# Plan Detail Refactor — Design

Date: 2026-05-13
Status: design (pre-implementation)
Companion: [docs/plans/2026-05-13-plan-detail-current-state.md](../../plans/2026-05-13-plan-detail-current-state.md) — behavioral contract this refactor preserves

## Update — 2026-05-13 (post-design)

**Single-PR delivery chosen.** Despite the "Implementation sequencing" section below recommending six independently revertable PRs, the user chose to land the entire refactor in **one PR** at plan-writing time. The phased structure (PR 1 Foundation → PR 6 Header/Sidebar/cleanup) survives as internal commit groupings on a single branch, but there is no inter-PR merge gate.

Consequences:
- Intermediate compatibility shims are dropped — every consumer is updated in the same diff, so re-exports at old paths are not needed at any step.
- The CI guard for Pinia reads inside `plan-detail/` is strict from the first commit (no warning-mode flag).
- "Manual gates" become commit-time checks; no merge-to-main pauses.
- Rollback is all-or-nothing — there is no PR-N revert option.

The "Implementation sequencing" section below remains as a record of the original recommendation. The implementation plan at [docs/superpowers/plans/2026-05-13-plan-detail-refactor.md](../plans/2026-05-13-plan-detail-refactor.md) reflects the one-PR decision.

## Context

The Plan Detail page (`/projects/:projectId/plans/:planId` and child routes) is a standalone React workflow page. Its current React tree under `frontend/src/react/pages/project/plan-detail/` is 63 files / ~14,300 lines. Six files exceed 700 lines; the largest is `PlanDetailChangesBranch.tsx` at 1,969 lines and packs seven components plus a SQL parser.

The page still reads server state through Pinia stores via `useVueState`, leaving the React migration incomplete. The data hook `usePlanDetailPage` is 741 lines and owns twelve pieces of state plus four refs.

Upcoming feature work on Plan Detail is paused so we can land a maintainability pass first. This document is the design for that pass.

## Goals

1. No file in `plan-detail/` exceeds ~400 lines after the refactor. Anything larger is split into co-located children with a clear name.
2. Folder layout maps 1:1 to the page's product structure: shell / changes / review / deploy / header / sidebar / shared. A new contributor can find the right file by reading the URL or the page screenshot.
3. `usePlanDetailPage` becomes a thin composition of phase-scoped hooks. Each hook is independently testable.
4. Pinia reads inside plan-detail go through new Zustand slices in `frontend/src/react/stores/app/` (database, dbGroup, sheet, instanceRole). The page itself has no `useVueState` calls into Pinia for data — only into the Vue router, where it is unavoidable.
5. `/simplify` pass on each split unit: drop dead code, collapse duplicate Tailwind / JSX patterns, tighten naming, replace native controls with `react/components/ui` wrappers where supported.
6. Behavior is preserved. All existing tests pass; no UI/UX changes ship in this refactor.

## Non-goals

- No new product features. The current-state doc is the behavioral contract.
- No Connect-query infrastructure. Server reads stay imperative through service clients, cached by the new Zustand slices (mirroring the existing app-store pattern).
- No migration of unrelated Pinia stores. Only the four plan-detail uses get slices.
- No changes to the Vue router or `ReactPageMount.vue` bridge.
- No proto / schema changes.
- No CSS-token overhaul.
- No removal or rewriting of `useMemo` / `useCallback` calls. Leave existing memoization as-is, even where it looks unnecessary. A separate, measurement-backed pass can revisit later.
- No new abstractions added "for future use." Two duplications stay duplicated; three earns a helper.
- No prop-shape changes to `react/components/ui/` components.
- No Tailwind theme / token changes.
- No proto field reshaping or selector rewrites.
- No reordering of effects unless a bug is being fixed (and we are not fixing bugs in this refactor).
- No batch-rename of i18n keys.
- No visual-regression / screenshot tests added.

## Target folder layout

```
frontend/src/react/pages/project/plan-detail/
├── ProjectPlanDetailPage.tsx          (moved here from /pages/project/; thin shell)
├── index.ts                           (re-exports the page; nothing else)
│
├── shell/
│   ├── PlanDetailLayout.tsx           (three-phase timeline + responsive frame)
│   ├── PlanDetailContext.tsx          (provider only; no logic)
│   ├── types.ts                       (PlanDetailPageSnapshot, PlanDetailPhase, breakpoints)
│   ├── constants.ts                   (POLLER_INTERVAL, breakpoint pxs, route-query keys)
│   ├── leaveGuard.ts                  (pure; navigation-cancel/re-issue logic)
│   └── hooks/
│       ├── usePlanDetailPage.ts       (composition root, ≤120 lines)
│       ├── usePlanSnapshot.ts         (fetch + poll + refresh)
│       ├── usePhaseState.ts           (activePhases, toggle, expand, collapse)
│       ├── useEditingScopes.ts        (editingScopes, leave-guard wiring)
│       ├── useRouteSelection.ts       (route → selected phase/stage/task/spec)
│       ├── useSidebarMode.ts          (responsive sidebar mode)
│       └── useDerivedPlanState.ts     (selectors → PlanDetailPageState)
│
├── header/
│   ├── PlanDetailHeader.tsx           ≤150  composition + layout
│   ├── TitleEditor.tsx                ≤130  inline edit + draft
│   ├── DescriptionEditor.tsx          ≤130  inline edit + collapsed-when-long
│   ├── ReadyForReviewPopover.tsx      ≤220  popover, label pick, warn-ack
│   ├── CreatePlanButton.tsx           ≤80   create mode only
│   ├── CloseReopenActions.tsx         ≤90   draft-only actions
│   └── MobileDetailsButton.tsx        ≤50
│
├── changes/
│   ├── ChangesBranch.tsx              ≤200
│   ├── SpecTabStrip.tsx               ≤150
│   ├── StatementSection/
│   │   ├── StatementSection.tsx       ≤200
│   │   ├── StatementEditor.tsx        ≤220
│   │   ├── ReleaseFileSummary.tsx     ≤140
│   │   ├── OversizedSheetNotice.tsx   ≤60
│   │   └── parseStatement.ts          ≤80   pure (was inline at line 1917)
│   ├── TargetsSection/
│   │   ├── TargetsSection.tsx         ≤180
│   │   ├── TargetSelectorSheet.tsx    ≤180
│   │   ├── DatabaseAndGroupSelector.tsx ≤120
│   │   ├── DatabaseSelector.tsx       ≤260
│   │   ├── DatabaseGroupSelector.tsx  ≤200
│   │   ├── DatabaseTarget.tsx         ≤80
│   │   └── DatabaseGroupTarget.tsx    ≤80
│   ├── OptionsSection/
│   │   ├── OptionsSection.tsx         ≤180
│   │   ├── RoleDirective.tsx          ≤120
│   │   ├── TransactionModeDirective.tsx ≤100
│   │   ├── IsolationLevelDirective.tsx  ≤100
│   │   ├── PriorBackupToggle.tsx      ≤90
│   │   └── GhostMigrationDirective.tsx ≤110
│   ├── ChecksSection.tsx              ≤180  was PlanDetailChecks
│   ├── DraftChecks.tsx                ≤120  was PlanDetailDraftChecks
│   ├── SchemaEditorSheet.tsx          ≤300  /simplify pass only
│   └── hooks/
│       ├── usePlanCheckActions.tsx
│       └── useSpecValidation.ts       (was usePlanDetailSpecValidation)
│
├── review/
│   ├── ReviewBranch.tsx               ≤150
│   ├── ApprovalFlow.tsx               ≤200
│   ├── ApprovalStepItem.tsx           ≤180
│   ├── ApproverList.tsx               ≤140
│   ├── RejectionBanner.tsx            ≤180
│   ├── ReviewStatusTag.tsx            ≤90
│   ├── ReviewActions.tsx              ≤140
│   ├── IssueLinkButton.tsx            ≤60
│   └── FutureReviewNotice.tsx         ≤80
│
├── deploy/
│   ├── DeployBranch.tsx               (already exists; trimmed)
│   ├── DeployFuture.tsx               ≤180
│   ├── RolloutRequirementsList.tsx    ≤180  extracted from DeployFuture
│   ├── CreateRolloutSheet.tsx         ≤220  extracted from DeployFuture
│   ├── StageNav.tsx                   (was inline in DeployBranch)
│   ├── StageCard.tsx                  (already exists)
│   ├── StageContentView.tsx           (already exists; trimmed)
│   ├── StageContentSidebar.tsx        (already exists; trimmed)
│   ├── StageActionSection.tsx         (already exists)
│   ├── ReleaseInfoCard.tsx            (renamed from DeployReleaseInfoCard)
│   ├── PendingTasksSection.tsx        (already exists)
│   ├── tasks/
│   │   ├── TaskList.tsx
│   │   ├── TaskRow.tsx
│   │   ├── TaskItem.tsx
│   │   ├── TaskFilter.tsx
│   │   ├── TaskToolbar.tsx
│   │   ├── TaskStatus.tsx
│   │   ├── TaskDetailPanel.tsx
│   │   ├── TaskRunTable.tsx
│   │   ├── TaskRunSession.tsx
│   │   ├── TaskRunDetail.tsx
│   │   ├── LatestTaskRunInfo.tsx
│   │   └── RollbackSheet.tsx
│   ├── actions/
│   │   ├── TaskRolloutActionPanel.tsx ≤220  panel shell
│   │   ├── RunTaskAction.tsx          ≤140
│   │   ├── RetryTaskAction.tsx        ≤120
│   │   ├── SkipTaskAction.tsx         ≤160
│   │   ├── CancelTaskAction.tsx       ≤120
│   │   ├── ScheduleTaskAction.tsx     ≤160
│   │   ├── SkipPriorBackupAction.tsx  ≤120
│   │   ├── taskActions.tsx            (keep)
│   │   ├── taskActionState.ts         (keep)
│   │   └── useDeployTaskStatement.tsx (keep)
│   └── utils/
│       ├── taskRunUtils.ts
│       └── types.ts
│
├── sidebar/
│   ├── MetadataSidebar.tsx            ≤120  composition
│   ├── CreatedBySection.tsx           ≤60
│   ├── StatusSection.tsx              ≤80
│   ├── PlanCheckSummary.tsx           ≤100
│   ├── ApprovalSummary.tsx            ≤80
│   ├── IssueLabelsSection.tsx         ≤90
│   ├── RolloutStageProgress.tsx       ≤100
│   ├── RefreshControl.tsx             ≤80
│   └── FutureSectionHint.tsx          ≤50
│
└── shared/
    ├── stores/                        (page-scoped Zustand store)
    │   ├── usePlanDetailStore.ts
    │   ├── snapshotSlice.ts
    │   ├── phaseSlice.ts
    │   ├── editingSlice.ts
    │   ├── selectionSlice.ts
    │   ├── pollingSlice.ts
    │   └── types.ts
    └── utils/
        ├── createPlan.ts
        ├── directiveUtils.ts
        ├── header.ts
        ├── invalidate.ts               (cross-store mutation invalidation map)
        ├── localSheet.ts
        ├── options.ts
        ├── phaseSummary.ts
        ├── planCheck.ts
        ├── rolloutPreview.ts
        ├── sidebarStatus.ts
        ├── spec.ts
        ├── specMutation.ts
        ├── sqlAdvice.ts
        ├── targets.ts
        └── __tests__/                 (existing *.test.ts files moved here)
```

### Naming conventions

- Drop the redundant `PlanDetail` prefix inside `plan-detail/`. The folder already namespaces.
- Drop the redundant `Deploy` prefix inside `deploy/`. Same reason.
- `useXxxStore` is reserved for Zustand stores. Local React hooks use `useXxx`.
- `Xxx.tsx` for components; `xxx.ts` for pure utilities; `useXxx.ts(x)` for hooks.

### Compatibility shims

PRs 3–5 leave one-line re-exports at the old paths so any straggling import keeps working:

```ts
// plan-detail/components/PlanDetailHeader.tsx (shim during migration)
export { PlanDetailHeader } from "../header/PlanDetailHeader";
```

PR 6 deletes all shims.

## State architecture

Three layers, each with one job.

```
┌─────────────────────────────────────────────────────────────┐
│ Layer 1: Global app-store slices  (react/stores/app/)       │
│   Pinia replacements; cross-page caches; long-lived         │
└─────────────────────────────────────────────────────────────┘
                              ▲
                              │ read-only / loader calls
                              │
┌─────────────────────────────────────────────────────────────┐
│ Layer 2: Page-scoped Zustand store  (plan-detail/shared/)   │
│   snapshot · phase · editing · selection · polling          │
│   lifetime = one ProjectPlanDetailPage instance             │
└─────────────────────────────────────────────────────────────┘
                              ▲
                              │ selectors via shallow equality
                              │
┌─────────────────────────────────────────────────────────────┐
│ Layer 3: Component-local state  (useState / useReducer)     │
│   transient UI: input drafts, popover open, modal step      │
└─────────────────────────────────────────────────────────────┘
```

### Layer 1 — new global app-store slices

Add four slices in `frontend/src/react/stores/app/`, following the existing pattern (state + request-promise dedup + loader fns):

```ts
// database.ts
export type DatabaseSlice = {
  databasesByName: Record<string, Database>;
  databaseRequests: Record<string, Promise<Database | undefined>>;
  databaseErrorsByName: Record<string, Error | undefined>;
  fetchDatabase: (name: string) => Promise<Database | undefined>;
  batchFetchDatabases: (names: string[]) => Promise<Database[]>;
  searchDatabases: (params: DatabaseSearchParams) => Promise<{
    databases: Database[];
    nextPageToken?: string;
  }>;
};

// dbGroup.ts
export type DBGroupSlice = {
  dbGroupsByName: Record<string, DatabaseGroup>;
  dbGroupRequests: Record<string, Promise<DatabaseGroup | undefined>>;
  fetchDBGroup: (name: string) => Promise<DatabaseGroup | undefined>;
  listDBGroupsForProject: (project: string) => Promise<DatabaseGroup[]>;
};

// sheet.ts
export type SheetSlice = {
  sheetsByName: Record<string, Sheet>;
  sheetRequests: Record<string, Promise<Sheet | undefined>>;
  fetchSheet: (name: string, raw?: boolean) => Promise<Sheet | undefined>;
  createSheet: (parent: string, sheet: Sheet) => Promise<Sheet>;
};

// instanceRole.ts
export type InstanceRoleSlice = {
  rolesByInstance: Record<string, InstanceRole[]>;
  roleRequests: Record<string, Promise<InstanceRole[]>>;
  fetchInstanceRoles: (instance: string) => Promise<InstanceRole[]>;
};
```

Only port the methods plan-detail actually calls; do not migrate the full Pinia surface. Each new slice gets a unit test that proves request dedup works (the property the Pinia stores guarantee today).

**Cache coexistence during migration.** Pinia stores remain the source of truth for Vue pages. The new Zustand slices fetch independently. We accept a duplicate cache for the migration window. Mutations from plan-detail invalidate both: the slice's own cache and, where the Vue layer cares, a call into the Pinia store. The page has a tiny `shared/utils/invalidate.ts` helper that lists the cross-store invalidations needed by each mutation, so the surface is auditable.

### Layer 2 — page-scoped Zustand store

Replaces the `useState` cluster inside `usePlanDetailPage` (snapshot, activePhases, editingScopes, isRefreshing, isRunningChecks, lastRefreshTime, pendingLeaveConfirm, plus the four refs). Created fresh per page mount via React context, so navigating away and back gives a clean store.

```ts
// shared/stores/usePlanDetailStore.ts
const createPlanDetailStore = () =>
  create<PlanDetailStore>()((...a) => ({
    ...createSnapshotSlice(...a),
    ...createPhaseSlice(...a),
    ...createEditingSlice(...a),
    ...createSelectionSlice(...a),
    ...createPollingSlice(...a),
  }));

const PlanDetailStoreContext =
  createContext<ReturnType<typeof createPlanDetailStore> | null>(null);

export const PlanDetailStoreProvider = ({ children }: { children: ReactNode }) => {
  const [store] = useState(createPlanDetailStore);
  return (
    <PlanDetailStoreContext.Provider value={store}>
      {children}
    </PlanDetailStoreContext.Provider>
  );
};

export function usePlanDetailStore<T>(selector: (s: PlanDetailStore) => T): T {
  const store = useContext(PlanDetailStoreContext);
  if (!store) throw new Error("PlanDetailStoreProvider missing");
  return useStore(store, selector);
}
```

Slices:

- `SnapshotSlice` — `snapshot`, `setSnapshot`, `patchSnapshot`, `refresh()`
- `PhaseSlice` — `activePhases: Set<Phase>`, `togglePhase`, `expandPhase`, `collapsePhase`
- `EditingSlice` — `editingScopes: Record<string, true>`, `setEditing`, `isEditing` selector, `bypassLeaveGuardOnce`, `pendingLeaveConfirm`, `resolveLeaveConfirm`, `pendingLeaveTarget`
- `SelectionSlice` — mirrors URL-derived selection so components don't re-parse route query every render: `selectedSpecId`, `selectedStageId`, `selectedTaskName`, `routePhase`
- `PollingSlice` — `isRefreshing`, `isRunningChecks`, `lastRefreshTime`, `pollTimerId`, `startPolling`, `stopPolling`

Why a Zustand store instead of more React state: ~12 pieces of state read by many leaves (context re-render storm if pushed through React context); stable getters needed by the polling timer and leave guard (uniform with `getState()` instead of refs); reducer-like actions live next to the state they touch.

### Layer 3 — component-local state

Anything that does not outlive a single subtree stays local: search-input drafts, modal open/closed booleans, the wizard step cursor inside `CreateRolloutSheet`, draft strings for `TitleEditor` / `DescriptionEditor`. The rule: if only one component reads it and it dies with the component, it stays local.

### What `usePlanDetailPage` becomes

```ts
// shell/hooks/usePlanDetailPage.ts  (≤120 lines target)
export function usePlanDetailPage(params: PageParams): PlanDetailPageState {
  useInitialFetch(params);          // populates SnapshotSlice once
  usePolling();                     // owns the timer; respects done-state
  useRouteSelection(params);        // syncs SelectionSlice from route
  useRedirects(params);             // NotFound / PermissionDenied / linked-issue redirect
  useLeaveGuard();                  // wires editing → router beforeEach
  return useDerivedPlanState();     // selectors only; returns PlanDetailPageState
}
```

Each `useXxx` lives in its own file under `shell/hooks/`, ≤120 lines, with its own test.

### Vue router exception

The Vue router and its `route.query` reactive object stay; we keep `useVueState` for those. Rule enforced by CI: `useVueState` may only wrap Vue router/route inside `plan-detail/`. Any other Pinia read is a CI failure. We add a grep-based guard mirroring the existing cross-framework guard (commit `2a806ce64d`).

## Per-area splits

Line budgets below are caps, not minimums. If a unit comes in well under, that is fine. If a unit balloons during simplification, split further.

### Header

Today: `PlanDetailHeader.tsx` (761 lines) holds title editor, description editor, the Ready-for-Review popover with its validation + label-pick + warning-ack tree, the Create button, Close/Reopen actions, mobile-details trigger, and a composing wrapper.

After: 7 files in `header/`. Each editor owns its own draft state and "saving" boolean. The popover lifts out unchanged in this pass; its validation chain is simplified in Section 5 cleanup.

### Changes

Today: `PlanDetailChangesBranch.tsx` (1969) + `PlanDetailStatementSection.tsx` (748). The biggest payoff.

After: ~17 files split four ways — Statement / Targets / Options / Checks. Each section reads `selectedSpec` from the store and pushes mutations back through `patchSnapshot`; no cross-section prop drilling. Per-directive splits in Options because each directive has its own visibility rules and the future-feature path almost certainly adds directives. `parseStatement` extracted as a pure file with its first unit tests.

### Review

Today: `PlanDetailApprovalFlow.tsx` (864). Approval steps, approver list, rejection banner with re-request action, status tag, issue link button, future-state guidance for plans without an issue.

After: 8 files in `review/`. The 17 `useVueState` calls cluster in the approver-list and rejection-banner subtrees, so splitting those lets us migrate them piecewise.

### Deploy

Today: mostly already nested under `deploy/`. Two big files remain: `PlanDetailDeployFuture.tsx` (437) and `PlanDetailTaskRolloutActionPanel.tsx` (743).

After: `DeployFuture` splits 3 ways (composition / requirements list / create-rollout sheet). `TaskRolloutActionPanel` splits into a panel shell + one file per action type (run / retry / skip / cancel / schedule / skip-prior-backup); each action owns its own confirmation sheet. Other deploy files get renamed (drop `Deploy` prefix) and a `/simplify` pass; no further structural splits.

### Sidebar

Today: `PlanDetailMetadataSidebar.tsx` (371) — under the 400-line cap, but every section is independently useful and the future-feature path almost certainly adds sidebar sections.

After: 8 files in `sidebar/`. Adding a section becomes a one-file change.

### Shell + page hook

`usePlanDetailPage` splits 7 ways per Section 3. Leave-guard logic extracted as `shell/leaveGuard.ts` (pure functions over the editing slice) with a unit test for the cancel-then-confirm sequence.

## `/simplify` pass scope

### What `/simplify` runs over

Only the units that were split or moved in the per-area work above. A file untouched in the structural pass does not get a simplification pass in this refactor. The `shared/utils/` folder is mostly already pure and tested; run `/simplify` over it only where its tests stay green without modification.

### In-scope cleanups

1. Replace native controls with `react/components/ui/` wrappers, where the shared component already supports the needed behavior.
2. Collapse duplicated Tailwind class strings via `cva` variant or `cn(...)` helper inside the same file.
3. Drop dead branches the type system already proves unreachable.
4. Drop dead state — `useState` whose setter is never called, refs no component reads, props no caller passes.
5. Tighten naming where the new name is unambiguously better and the rename stays inside the file.
6. Inline single-use one-line helpers that obscure rather than clarify.
7. Replace `space-x-*` button rows with `gap-x-2` per AGENTS.md.
8. Drop empty-object i18n keys per AGENTS.md.

### Out-of-scope cleanups

(Listed alongside the global non-goals above.) Notably: no `useMemo` / `useCallback` removal; no new abstractions added "for future use"; no prop-shape changes to shared UI components.

### Per-file checklist

```
[ ] Imports: organized, no unused, no deep relative paths
[ ] Native controls replaced with react/components/ui where supported
[ ] Tailwind: gap-x for button rows; no redundant arbitrary values; no
    duplicated long class strings (collapse with cn / cva inside file)
[ ] State: every useState / useRef is read; every prop is consumed
[ ] Effects: dependencies honest; no missing deps; no spurious deps
[ ] Branches: switch defaults / unreachable else removed
[ ] Naming: file-local renames only, where the new name is clearly better
[ ] Comments: drop comments that restate the code; keep WHY comments
[ ] i18n: no hardcoded user-facing strings; no empty-object keys
[ ] Tests: existing tests still pass without modification
```

Same list per file; no judgment beyond it.

### Ordering inside a commit

For each split unit:

1. Move/split first — copy code to its new file, leave behavior identical, get tests green.
2. Simplify second — run the checklist over the new file only.

Doing simplification before splitting produces edits the split erases.

## Testing strategy

The refactor's success criterion is "behavior preserved," so tests exist to detect regressions, not verify new behavior.

### Layer A — existing tests, unchanged

Existing tests in `plan-detail/`:

```
utils/sqlAdvice.test.ts
utils/header.test.ts
utils/phaseSummary.test.ts
utils/targets.test.ts
utils/sidebarStatus.test.ts
utils/options.test.ts
utils/planCheck.test.ts
utils/spec.test.ts
components/PlanDetailChangesBranch.test.tsx
components/PlanDetailApprovalFlow.test.tsx
components/deploy/taskActions.test.tsx
```

These tests move alongside their subjects; their import paths are fixed. **No assertion or setup code changes.** If a test starts failing after a move, the refactor introduced a regression — fix the regression, not the test. The two component tests continue to test the composed parent shells (`ChangesBranch.tsx`, `ApprovalFlow.tsx`).

### Layer B — new unit tests for newly-exposed pure logic

1. `changes/StatementSection/parseStatement.ts` — first unit tests (empty, single, multiple, with comments, with semicolons in strings).
2. `shell/leaveGuard.ts` — tests for editing→navigate→cancel, editing→navigate→confirm, no-edit→navigate→passthrough, bypass-once flag.
3. `shell/hooks/useDerivedPlanState.ts` — priority ladder `creating → closed/deleted → canceled issue → rollout status → review status → draft` from the current-state doc.
4. New Zustand app-store slices — each gets a request-dedup test.

Cap: roughly 8–12 new unit-test files. More than that means we are testing plumbing instead of behavior.

### Layer C — one end-to-end smoke test

One Playwright-style smoke test walking the page's golden path:

```
open empty draft plan → edit a SQL statement → run checks →
Ready for Review (creates issue) → approve the issue →
create rollout → run one task → observe task done
```

If no plan-detail e2e exists today, it is added in PR 1 and stays after the refactor — not just scaffolding.

### Per-slice merge gate

```
[ ] Layer A: all moved tests pass with no assertion changes
[ ] Layer B: any newly-extracted pure logic has unit tests
[ ] Layer C: the e2e smoke test passes
[ ] pnpm --dir frontend type-check passes
[ ] pnpm --dir frontend fix produces no diff
[ ] Manual: open the page in dev, walk the journey for any visibly-touched area
```

The manual step is non-negotiable per AGENTS.md. Each PR description lists which areas were manually exercised, on desktop wide / desktop narrow / mobile.

### Not included

- No visual-regression / screenshot tests.
- No new test framework.
- No tests for Zustand selectors directly; they are exercised through components.
- No tests for file moves themselves; the compiler catches broken imports.

## Implementation sequencing

One spec; six PRs. Each PR is independently revertable.

### PR 1: Foundation

Add the new global Zustand slices (database, dbGroup, sheet, instanceRole) and the page-scoped store skeleton. No call-site changes yet.

- New: `react/stores/app/{database,dbGroup,sheet,instanceRole}.ts` + slice tests
- New: `plan-detail/shared/stores/*` (empty slices, provider, hook)
- New: `plan-detail/shell/leaveGuard.ts` + tests
- New: e2e smoke test
- CI guard updated to forbid Pinia reads inside `plan-detail/` (lands here so later PRs cannot backslide)

Risk: ~zero — no existing code paths changed.

### PR 2: Shell + page-hook restructure

Break `usePlanDetailPage` into `shell/hooks/*`; route data through the page-scoped store. Move `ProjectPlanDetailPage.tsx` into `plan-detail/`.

- Touched: `usePlanDetailPage`, `ProjectPlanDetailPage`, `PlanDetailContext`
- Behavior: identical
- Risk: high — page's data spine. PR 1's groundwork makes it tractable.

### PR 3: Changes phase

Split the 1969-line file. Switch its data reads to new app-store slices. `/simplify` per the checklist.

- Touched: `PlanDetailChangesBranch`, `PlanDetailStatementSection`, related utils
- New: ~17 files under `changes/`
- Risk: high — biggest payoff and biggest diff. Existing component tests are the safety net.

### PR 4: Review phase

Split `PlanDetailApprovalFlow`. Switch its 17 Pinia reads to slices.

- Touched: `PlanDetailApprovalFlow` → `review/*`
- Risk: medium — clear seams, existing tests.

### PR 5: Deploy phase

Split `PlanDetailDeployFuture` + `PlanDetailTaskRolloutActionPanel`. Rename `Deploy`-prefixed files inside `deploy/`. `/simplify` pass.

- Touched: `deploy/*` (already partly nested)
- Risk: medium.

### PR 6: Header + Sidebar + cleanup

Split `PlanDetailHeader` and `PlanDetailMetadataSidebar`. Remove compatibility shim re-exports. Drop the `PlanDetail` prefix on the renamed-but-not-yet-cleaned files.

- Risk: low — header/sidebar are the most self-contained.

### Ordering rationale

- PR 1 is foundation: nothing else compiles cleanly without it. Safe to sit on `main` for days.
- PR 2 before any phase PR: phase splits read from the page-scoped store; without PR 2, every phase is rewritten twice.
- Changes before Review before Deploy by payoff and risk. Header / Sidebar last because they are most self-contained.

### Concurrent feature work

Feature work on plan-detail is paused for the duration. If a feature must land mid-refactor:

- It lands on top of the most-recently-merged refactor PR.
- It uses the new structure if its area is already refactored; old structure otherwise.
- No rebasing the in-flight refactor PR onto the new feature.

### Rough timeline

- PR 1: 0.5 day
- PR 2: 2 days
- PR 3: 2–3 days
- PR 4: 1–1.5 days
- PR 5: 1–1.5 days
- PR 6: 1 day

Total ~8–10 working days. Longer if PR 2 or PR 3 surfaces a snag.

### Rollback

Each PR is revertable on its own. `git revert` of PR N restores prior behavior; PRs N+1+ rebase or also revert. No feature-flag rollbacks.

## Definition of done

- Every file in `plan-detail/` ≤ ~400 lines (documented exceptions if any).
- Folder layout matches the target tree above.
- `usePlanDetailPage.ts` ≤ 120 lines.
- Zero `useVueState` calls into Pinia inside `plan-detail/` (router uses allowed). CI guard enforces.
- All pre-refactor tests pass unchanged. New unit + e2e tests pass.
- Manual walk of the journey in the current-state doc: golden path + at least one branch per phase, on desktop wide / desktop narrow / mobile.
- The current-state doc (`docs/plans/2026-05-13-plan-detail-current-state.md`) gets a short addendum noting the refactor landed and pointing at the new file locations.

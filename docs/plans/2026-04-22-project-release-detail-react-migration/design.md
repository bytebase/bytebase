## References

- **React — Passing Data Deeply with Context** (https://react.dev/learn/passing-data-deeply-with-context, verified). Context "should not be your first choice"; prefer passing props directly or extracting components. Valid context use cases: theming, current account, routing, global state from distant parts of the tree.
- **React — You Might Not Need an Effect** (https://react.dev/learn/you-might-not-need-an-effect, verified). Data fetching in Effects is a legitimate use, but requires a cleanup flag (`let ignore = false; … return () => { ignore = true; }`) to handle race conditions. Extracting into a custom hook is recommended for ergonomics.
- **Martin Fowler — StranglerFigApplication** (https://martinfowler.com/bliki/StranglerFigApplication.html, verified). Incremental replacement with "seams" between old and new, coexistence through the transition, no big-bang rewrite. Maps to the playbook rule: migrate only what the target needs, leave shared dependencies live until their remaining callers are also migrated.
- **React — Add React to an Existing Project** (https://react.dev/learn/add-react-to-an-existing-project, verified). For partial integration, mount React into host-framework-owned DOM nodes via `createRoot`. This is the primitive behind `frontend/src/react/mount.ts:120` / `ReactPageMount.vue`.
- **Repo playbook** (`docs/plans/2026-04-08-react-migration-playbook.md`, in-repo primary source). Route migration pattern: keep route parsing at router boundary, mount through `ReactPageMount.vue`, pass normalized resource-style props, keep the React page self-contained. Shared component rule: migrate only dependencies on the critical path. Deletion rule: only delete Vue counterparts with no remaining callers.
- **Repo precedent — Plan Detail Migration** (`docs/plans/2026-04-13-project-plan-detail-react-migration.md`, `frontend/src/react/pages/project/ProjectPlanDetailPage.tsx`, in-repo primary source). Scoped subtree at `react/pages/project/plan-detail/{components,context,hooks,utils}`, route re-registered with `component: () => import("@/react/ReactPageMount.vue")`, Vue originals kept during the migration and deleted leaf-by-leaf.

## Industry Baseline

**Strangler Fig modernization** (Fowler) is the generally accepted approach for replacing a legacy UI framework in place of a big-bang rewrite: identify a seam, replace one piece, leave the rest coexisting, iterate. The "seam" for a Vue-to-React migration of a single route is the Vue router entry and the page's data inputs.

**React's own guidance** reinforces two choices that narrow the design:

1. *Context is a last resort* (React — Passing Data Deeply with Context). With a single consumer subtree of three values, direct props or locals outrank `createContext`.
2. *Effect-based data fetching is legitimate at page scope* when paired with a race-guard and cleanup (React — You Might Not Need an Effect). There is no need to introduce TanStack Query for a one-release-plus-one-project fetch. This also matches the playbook's "do not add TanStack Query by default" rule.

The plan-detail precedent (#20081) is the closest in-house baseline: it routes via `ReactPageMount.vue`, fetches from Pinia stores directly from React via `useReleaseStore` / `useProjectV1Store`, uses `useEffect` + `useVueState` where Vue reactivity must be observed, and keeps the Vue tree alive until every leaf's callers are gone. Its trade-off — a ~600-LOC page file plus a `plan-detail/` subtree — is appropriate for its ~4000-LOC surface but heavier than the ~730-LOC release detail surface needs.

## Research Summary

Three patterns from the references directly shape this migration:

- **Route-level seam, page-level swap.** The Strangler Fig "seam" here is exactly the `projectV1.ts:512-520` route entry. Swapping its `component` from `@/components/Release/ReleaseDetail/` to `@/react/ReactPageMount.vue` with `page: "ProjectReleaseDetailPage"` replaces the page atomically while the file-table leaf continues to serve `CreateRevisionDrawer`. This matches how `PROJECT_V1_ROUTE_RELEASES` was cut over (`projectV1.ts:506`).
- **Reject Context; use self-contained page fetch.** The Vue `provideReleaseDetailContext` exists to share three values (`release`, `project`, `allowApply`) across four components. React's context guidance is explicit that this is prop-passing territory, not context. The React page fetches release and project itself, passes them to `NavBar`, `BasicInfo`, `ReleaseFileTable`, and the apply/detail sheets as props.
- **Effect-based fetch with cleanup.** The release fetch keyed on `projects/{projectId}/releases/{releaseId}` is a single resource per route-param change. `useEffect` with an `ignore` flag is sufficient and matches the plan-detail precedent, avoiding a new query library.

The open question on file-table sharing is resolved by the playbook's deletion rule combined with Strangler Fig coexistence: the Vue `ReleaseFileTable.vue` stays live for `CreateRevisionDrawer`, and a new React `ReleaseFileTable.tsx` lives at a location that allows reuse when the revision drawer migrates.

## Design Goals

Ordered by priority. Each is verifiable.

1. **Behavioral parity with the Vue page** at `PROJECT_V1_ROUTE_RELEASE_DETAIL`. Verifiable by manual walkthrough against the Vue tree before and after: archive banner on deleted, apply drawer creates plan+rollout and redirects to `buildPlanDeployRoute`, abandon confirm deletes release, restore undeletes, file row click opens the detail panel showing the Monaco-rendered sheet, document title resets on route match.
2. **No live Vue caller of the swapped Vue files after merge**, except `ReleaseFileTable.vue` (preserved for `CreateRevisionDrawer.vue`). Verifiable with `rg` for imports of `ReleaseDetail.vue`, `BasicInfo.vue`, `NavBar.vue`, `ApplyToDatabaseButton.vue`, `ApplyToDatabasePanel.vue`, `ReleaseFileDetailPanel.vue`, and `context.ts` returning zero hits across the repo.
3. **Route-entry parity with the sibling `PROJECT_V1_ROUTE_RELEASES`.** Verifiable by `grep` showing `PROJECT_V1_ROUTE_RELEASE_DETAIL` registered with `component: () => import("@/react/ReactPageMount.vue")` and `props: (route) => ({ page: "ProjectReleaseDetailPage", ...route.params })`.
4. **No new shared React state abstraction.** Verifiable by the absence of `createContext` / new zustand store / new tanstack/query entry point tied to the release detail surface. Pinia stores are the only release I/O.
5. **Full i18n coverage in React locale files.** Verifiable by `pnpm --dir frontend check` and `check-react-i18n` passing, and by the React page passing all locale keys through `useTranslation()`.
6. **Frontend CI gates pass.** Verifiable by `pnpm --dir frontend fix`, `pnpm --dir frontend check`, `pnpm --dir frontend type-check`, and `pnpm --dir frontend test` all returning zero.

## Non-Goals

Inherited from definition:

- Do not delete `ReleaseFileTable.vue` (still imported by `CreateRevisionDrawer.vue:414`).
- Do not migrate `CreateRevisionDrawer.vue` or any revision surface.
- Do not port `ArchiveBanner.vue`, `VCSIcon.vue`, `HumanizeDate.vue`, `EllipsisText.vue`, or `@/components/v2` `Drawer` / `CopyButton` / `DrawerContent` as reusable React primitives beyond the minimum needed here.
- Do not extract a shared React `DatabaseAndGroupSelector`; reuse an existing inline copy.
- Do not change the apply RPC sequence or `buildPlanDeployRoute` call.
- Do not redesign the page layout, copy, or flow.
- Do not gate abandon/restore behind new permission checks.
- Do not change `PROJECT_V1_ROUTE_RELEASE_DETAIL` name constant or helper call sites.

Discovered during research:

- Do not introduce TanStack Query or zustand for this surface (Playbook §State and Data Guidance; React — You Might Not Need an Effect recommends custom hooks over libraries for single-resource fetches).
- Do not wrap the React page in a `React.createContext` provider for `release` / `project` / `allowApply` (React — Passing Data Deeply with Context).

## Proposed Design

### 1. Route rewrite

`frontend/src/router/dashboard/projectV1.ts:512-520` becomes:

```ts
{
  path: ":releaseId",
  name: PROJECT_V1_ROUTE_RELEASE_DETAIL,
  meta: {
    requiredPermissionList: () => ["bb.releases.get"],
  },
  component: () => import("@/react/ReactPageMount.vue"),
  props: (route: RouteLocationNormalized) => ({
    page: "ProjectReleaseDetailPage",
    ...route.params,
  }),
},
```

This matches the sibling `PROJECT_V1_ROUTE_RELEASES` registration at lines 500-511 and satisfies Design Goal 3.

### 2. React page entry

New file: `frontend/src/react/pages/project/ProjectReleaseDetailPage.tsx`.

Exported function signature:

```ts
export function ProjectReleaseDetailPage({
  projectId,
  releaseId,
}: {
  projectId: string;
  releaseId: string;
}) { ... }
```

`mount.ts:59-77` resolves `page: "ProjectReleaseDetailPage"` against `./pages/project/ProjectReleaseDetailPage.tsx` by convention — no registry edit required.

Responsibilities inside the page:

- Build `releaseName = \`${projectNamePrefix}${projectId}/releases/${releaseId}\`` and `projectName = \`${projectNamePrefix}${projectId}\`` inline (copied verbatim from `context.ts:42-44`).
- Fetch project via `useProjectV1Store().getOrFetchProjectByName(projectName)` inside a `useEffect` with an `ignore` flag (React — You Might Not Need an Effect).
- Fetch release via `useReleaseStore().fetchReleaseByName(releaseName)` in the same effect.
- Observe reactive Vue state via `useVueState(() => releaseStore.getReleaseByName(releaseName) ?? unknownRelease())` and the analogous project getter, so Pinia updates flow into React (matches plan-detail usage).
- Compute `allowApply` from `hasPermissionToCreateChangeDatabaseIssueInProject(project)` — the helper is already pure TS and callable from React.
- Reset the document title exactly when the mounted route name matches `PROJECT_V1_ROUTE_RELEASE_DETAIL`, via `useEffect` observing `project.title` (matches `ReleaseDetail.vue:65-73`).
- Own the `selectedReleaseFile` state with `useState<Release_File | undefined>` to drive the file detail sheet, mirroring `ReleaseDetail.vue:46-53`.

No `React.createContext`. Values are passed to children as props (Design Goal 4, React — Passing Data Deeply with Context).

### 3. Component decomposition inside the page

Keep the file single-file until it crosses ~500 LOC, per the definition's default answer to Open Question 2. Decompose in-file into local components:

| React local component | Replaces | Notes |
|---|---|---|
| `<ReleaseHeader>` | `NavBar.vue` + `ArchiveBanner` conditional | Title, `ApplyToDatabaseButton`, `DropdownMenu` with abandon/restore. Uses `@/react/components/ui/dropdown-menu`, `@/react/components/ui/alert-dialog`, `@/react/components/ui/button`. Abandon flow is an `AlertDialog` (frontend/AGENTS.md §Dialog vs Sheet — destructive confirm). Renders a local archive banner inline (`bg-warning/10 text-warning` etc.), not a new shared component (Non-Goal). |
| `<ReleaseBasicInfo>` | `BasicInfo.vue` | Uses `HumanizeTs` from `@/react/components/HumanizeTs`. VCS source row renders a local `<VCSLinkRow>` that reuses the Vue `VCSIcon.vue` via `<VueComponent>` *only if no React VCS icon equivalent exists*; otherwise port the tiny switch inline. Current state shows no React VCS icon, so inline `<Link2>` from lucide with a label is sufficient for parity at this surface. |
| `<ApplyToDatabaseSheet>` | `ApplyToDatabaseButton.vue` + `ApplyToDatabasePanel.vue` | `<Sheet width="wide">` per Open Question 3 default. Reuses `DatabaseAndGroupSelector` by inlining a fourth copy in this page file (Non-Goal: do not extract a shared one). RPC sequence copied verbatim from `ApplyToDatabasePanel.vue:103-154`. Footer button group uses `gap-x-2` per frontend AGENTS.md. |

### 4. Shared leaf: React ReleaseFileTable

New file: `frontend/src/react/components/release/ReleaseFileTable.tsx`.

Why under `react/components/release/` and not inline in the page file: Open Question 5 default — it will be reused when `CreateRevisionDrawer` migrates; having a React home now avoids reinlining. Matches the plan-detail convention of scoped `components/` directories.

Props mirror the Vue emits model:

```ts
type Props = {
  files: Release_File[];
  releaseType: Release_Type;
  showSelection?: boolean;          // default false
  rowClickable?: boolean;           // default true
  selectedFiles?: Release_File[];   // default []
  onRowClick?: (file: Release_File) => void;
  onSelectedFilesChange?: (files: Release_File[]) => void;
};
```

Implementation uses `@/react/components/ui/table`. Selection column renders a shadcn `Checkbox`; row click toggles selection in selection mode or calls `onRowClick` otherwise, matching `ReleaseFileTable.vue:104-137`. Type-text helper (`getReleaseFileTypeText`) is ported verbatim as a local function.

The Vue `ReleaseFileTable.vue` is **not deleted** (Non-Goal, playbook §Deletion Rule — `CreateRevisionDrawer.vue:414` still imports it). Two implementations coexist; this is the same coexistence tolerated in the playbook's `BytebaseLogo` / `UserPassword` examples.

### 5. File detail panel

New file: `frontend/src/react/components/release/ReleaseFileDetailPanel.tsx`.

- Matches `ReleaseFileDetailPanel.vue` 1:1.
- Monaco usage: `ReadonlyMonaco` from `@/react/components/monaco/ReadonlyMonaco` (already used elsewhere in React pages; covers `readonly`, `auto-height`).
- Sheet content fetching: `sheetServiceClientConnect.getSheet({ name, raw: true })` inside a `useEffect` with `ignore` cleanup, race-guard semantics per React — You Might Not Need an Effect.
- Copy button: reuse the inline `CopyButton` pattern already present in `frontend/src/react/components/revision/RevisionDetailPanel.tsx:46` (Non-Goal: not extracting a shared React `CopyButton`).

Rendered inside a `<Sheet width="wide">` driven by the page's `selectedReleaseFile` state, per Open Question 4 default.

### 6. Data flow summary

```
Vue router
  └─ ReactPageMount.vue (page="ProjectReleaseDetailPage")
       └─ ProjectReleaseDetailPage (fetch, useState, useVueState)
            ├─ ReleaseHeader             (release, project, allowApply, onArchive, onRestore)
            │    ├─ AlertDialog (abandon confirm)
            │    └─ ApplyToDatabaseSheet (release, project)
            ├─ ReleaseBasicInfo          (release)
            ├─ ReleaseFileTable          (files, releaseType, onRowClick)
            └─ Sheet                      (selectedReleaseFile)
                 └─ ReleaseFileDetailPanel (release, releaseFile)
```

No Context. All values pass through props. Pinia is the single source of truth for release and project state; `useVueState` bridges updates.

### 7. i18n

All keys already exist in `frontend/src/locales/`. Copy the subset consumed by the migrated surface into `frontend/src/react/locales/{en-US,es-ES,ja-JP,vi-VN,zh-CN}.json`:

- `release.releases`
- `common.version`, `common.type`, `common.apply-to-database`, `common.abandon`, `common.restore`, `common.confirm`, `common.cancel`, `common.create`, `common.statement`
- `database.revision.filename`, `database.selected-n-databases`
- `issue.title.change-database`
- `bbkit.confirm-button.sure-to-abandon`, `bbkit.confirm-button.can-undo`

`check-react-i18n` verifies completeness (Design Goal 5).

### 8. Testing

Page-level test at `frontend/src/react/pages/project/ProjectReleaseDetailPage.test.tsx`:

- Render with `projectId` + `releaseId` props, release state `ACTIVE`: assert title, apply button, abandon dropdown entry, no archive banner.
- State `DELETED`: archive banner present, dropdown shows only restore.
- Row click on a file opens the sheet with the release file name.
- Apply flow: mock `planServiceClientConnect.createPlan` + `rolloutServiceClientConnect.createRollout` + `router.push`; assert they are called in order with the expected payloads.

Leaf tests:

- `ReleaseFileTable.test.tsx` — selection, row click, hidden selection column when `showSelection=false`.
- `ReleaseFileDetailPanel.test.tsx` — mock `sheetServiceClientConnect.getSheet`, assert Monaco receives the decoded content, copy button reflects the same content.

Follows playbook §Testing Guidance: test the wrapper's contract, mock repo-owned seams (`*ServiceClientConnect`), not Monaco internals.

### 9. Deletion plan

After the route switches to React and all callers are verified with `rg`, delete in the same PR:

- `frontend/src/components/Release/ReleaseDetail/index.ts`
- `frontend/src/components/Release/ReleaseDetail/ReleaseDetail.vue`
- `frontend/src/components/Release/ReleaseDetail/context.ts`
- `frontend/src/components/Release/ReleaseDetail/BasicInfo.vue`
- `frontend/src/components/Release/ReleaseDetail/NavBar/` (whole directory)
- `frontend/src/components/Release/ReleaseDetail/ReleaseFileTable/ReleaseFileDetailPanel.vue`

Keep:

- `frontend/src/components/Release/ReleaseDetail/ReleaseFileTable/ReleaseFileTable.vue`
- `frontend/src/components/Release/ReleaseDetail/ReleaseFileTable/index.ts`

Verify with `rg "Release/ReleaseDetail" frontend/src` — only expected hits are the ReleaseFileTable live callers.

### 10. Rollout

One PR, single commit chain, behind no feature flag — consistent with plan-detail (#20081) and release-dashboard precedents. Rollback is a route-entry revert plus leaf restoration, so blast radius stays at the route level.

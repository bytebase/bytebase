## Background & Context

The frontend is migrating from Vue to React per `docs/plans/2026-04-08-react-migration-playbook.md`. The release surface has been partially migrated: `ProjectReleaseDashboardPage` (`frontend/src/react/pages/project/ProjectReleaseDashboardPage.tsx`) replaces the old Vue releases dashboard and is mounted via `frontend/src/react/ReactPageMount.vue` (see `PROJECT_V1_ROUTE_RELEASES` at `frontend/src/router/dashboard/projectV1.ts:501-511`). The detail page reached by clicking a row on that dashboard is still the Vue implementation rooted at `frontend/src/components/Release/ReleaseDetail/ReleaseDetail.vue`.

Two recent migrations establish the patterns to follow:

- `feat(frontend): migrate project plan detail to react` (#20081, commit `b001f2b9d7`, `docs/plans/2026-04-13-project-plan-detail-react-migration.md`) — full-page, route-mounted React detail page with a scoped `react/pages/project/plan-detail/{components,context,hooks,utils}` subtree. `ReactPageMount.vue` serves as the route component via `props: (route) => ({ page: "ProjectPlanDetailPage", ...route.params })`.
- `refactor(react): migrate auth/setup pages from vue` (#20069, `docs/plans/2026-04-21-auth-setup-react-migration/`) — reinforces the playbook's deletion rule: only drop Vue files whose callers are fully gone; shared components with non-auth Vue callers stay.

One subcomponent of `ReleaseDetail` is imported from outside the subtree: `frontend/src/components/Revision/CreateRevisionDrawer.vue:414` imports `ReleaseFileTable.vue`. Vue `CreateRevisionDrawer.vue` has not been migrated, so `ReleaseFileTable.vue` cannot be deleted by this migration.

## Issue Statement

The Vue component tree at `frontend/src/components/Release/ReleaseDetail/**` is still the live implementation for `PROJECT_V1_ROUTE_RELEASE_DETAIL` (`frontend/src/router/dashboard/projectV1.ts:512-520`). The route mounts `@/components/Release/ReleaseDetail/` directly instead of routing through `ReactPageMount.vue`, which diverges from the convention established by `PROJECT_V1_ROUTE_RELEASES` and `PROJECT_V1_ROUTE_PLAN_DETAIL`. One leaf (`ReleaseFileTable.vue`) has a live Vue caller outside the subtree, coupling the subtree to `CreateRevisionDrawer.vue`.

## Current State

### Route registration

- `frontend/src/router/dashboard/projectV1.ts:34` — `PROJECT_V1_ROUTE_RELEASE_DETAIL` name constant.
- `frontend/src/router/dashboard/projectV1.ts:512-520` — route definition; `component: () => import("@/components/Release/ReleaseDetail/")`, `props: true` (passes `releaseId` and `projectId` route params).
- `frontend/src/router/dashboard/projectV1.ts:500-511` — sibling route `PROJECT_V1_ROUTE_RELEASES` already uses `ReactPageMount.vue` with `page: "ProjectReleaseDashboardPage"`.

### Vue page tree (8 files, ~730 LOC)

| File | LOC | Role |
|---|---|---|
| `frontend/src/components/Release/ReleaseDetail/index.ts` | 3 | Default export of `ReleaseDetail.vue` |
| `frontend/src/components/Release/ReleaseDetail/ReleaseDetail.vue` | 74 | Page root, provides `ReleaseDetailContext`, owns document title + file drawer state |
| `frontend/src/components/Release/ReleaseDetail/context.ts` | 68 | `provideReleaseDetailContext` — fetches project + release via `useProjectV1Store`/`useReleaseStore`, derives `allowApply` from `hasPermissionToCreateChangeDatabaseIssueInProject` |
| `frontend/src/components/Release/ReleaseDetail/BasicInfo.vue` | 50 | Create-time + VCS source row |
| `frontend/src/components/Release/ReleaseDetail/NavBar/index.ts` | 3 | Barrel |
| `frontend/src/components/Release/ReleaseDetail/NavBar/NavBar.vue` | 82 | Title, apply button, abandon/restore dropdown, `ArchiveBanner` when deleted |
| `frontend/src/components/Release/ReleaseDetail/NavBar/ApplyToDatabaseButton.vue` | 25 | Opens the apply panel |
| `frontend/src/components/Release/ReleaseDetail/NavBar/ApplyToDatabasePanel.vue` | 155 | Drawer with `DatabaseAndGroupSelector`, creates plan + rollout, redirects to `buildPlanDeployRoute` |
| `frontend/src/components/Release/ReleaseDetail/ReleaseFileTable/index.ts` | - | Barrel |
| `frontend/src/components/Release/ReleaseDetail/ReleaseFileTable/ReleaseFileTable.vue` | 155 | `NDataTable` of `Release_File`, emits `row-click` / `update:selected-files` |
| `frontend/src/components/Release/ReleaseDetail/ReleaseFileTable/ReleaseFileDetailPanel.vue` | 80 | Fetches sheet via `sheetServiceClientConnect.getSheet`, renders in Monaco |

### Page behavior

- Document title reset on route match (`ReleaseDetail.vue:65-73`) — resets via `setDocumentTitle(t("release.releases"), project.title)` only when `route.name === PROJECT_V1_ROUTE_RELEASE_DETAIL`.
- Release fetch is keyed on `projects/{projectId}/releases/{releaseId}` and triggered via `watchEffect` on the name (`context.ts:42-48`).
- NavBar dropdown (`NavBar.vue:47-80`): when `state === ACTIVE`, shows `abandon` → naive-ui `useDialog().warning` confirm → `releaseStore.deleteRelease`; when `state === DELETED`, shows `restore` → `releaseStore.undeleteRelease`. Deleted state also renders `ArchiveBanner` at the top.
- File row click (`ReleaseDetail.vue:8-27`): selected file opens the Vue `Drawer`/`DrawerContent` (`@/components/v2`) at fixed width `75vw`/`max-w-[calc(100vw-8rem)]` with `ReleaseFileDetailPanel`.
- Apply flow (`ApplyToDatabasePanel.vue:103-154`): creates a `Plan` with a single `Plan_ChangeDatabaseConfigSchema` spec wrapping the release, immediately creates a `Rollout` for that plan, then routes via `router.push(buildPlanDeployRoute(...))`.

### Cross-subtree consumers of the Vue tree

- `frontend/src/components/Revision/CreateRevisionDrawer.vue:299,328,414` — imports and renders `ReleaseFileTable.vue` twice (selection + readonly modes). This file is not being migrated in this issue.

### React infrastructure already available

- Page registry — `frontend/src/react/mount.ts:59-77` resolves `page: "ProjectReleaseDetailPage"` by convention from `./pages/project/ProjectReleaseDetailPage.tsx`.
- Stores — `useProjectV1Store`, `useReleaseStore` (`frontend/src/store/modules/release.ts` — `fetchReleaseByName`, `getReleaseByName`, `deleteRelease`, `undeleteRelease`) are Pinia and consumable from React directly per playbook §State and Data Guidance.
- Monaco — `@/react/components/monaco/ReadonlyMonaco` + `MonacoEditor` cover the readonly sheet render used by `ReleaseFileDetailPanel`.
- Sheet / AlertDialog / Button / Table — `frontend/src/react/components/ui/{sheet,alert-dialog,button,table}.tsx`.
- Humanize — `frontend/src/react/components/HumanizeTs.tsx` covers the create-time row in `BasicInfo.vue`.
- `DatabaseAndGroupSelector` — has been re-implemented inline by three callers already: `ProjectPlanDashboardPage.tsx:878-…`, `plan-detail/components/PlanDetailChangesBranch.tsx:1272-…`, `export-center/DataExportPrepSheet.tsx:604-…`. No shared React wrapper exists yet.
- `ArchiveBanner.vue`, `VCSIcon.vue`, `HumanizeDate.vue`, and `@/components/v2` `CopyButton` have no React equivalents; `CopyButton` inline exists in `react/components/revision/RevisionDetailPanel.tsx:46` and `react/components/instance/InfoPanel.tsx:233`.

### i18n

All user-facing strings in the Vue tree resolve against existing keys (`common.version`, `common.type`, `common.apply-to-database`, `common.abandon`, `common.restore`, `common.confirm`, `common.cancel`, `common.create`, `common.statement`, `database.revision.filename`, `database.selected-n-databases`, `issue.title.change-database`, `bbkit.confirm-button.sure-to-abandon`, `bbkit.confirm-button.can-undo`, `release.releases`). React locale files under `frontend/src/react/locales/` currently hold only the keys used by existing React surfaces and do not yet contain all of these.

## Non-Goals

- Deleting `frontend/src/components/Release/ReleaseDetail/ReleaseFileTable/ReleaseFileTable.vue`. `CreateRevisionDrawer.vue:414` still imports it and is not being migrated here; playbook §Deletion Rule applies.
- Migrating `frontend/src/components/Revision/CreateRevisionDrawer.vue` or any revision surface.
- Porting `ArchiveBanner.vue`, `VCSIcon.vue`, `HumanizeDate.vue`, `EllipsisText.vue`, or `@/components/v2` `CopyButton` / `Drawer` / `DrawerContent` as reusable React primitives for other surfaces. Only the minimum surface consumed by the release detail page is created/reused.
- Extracting a shared React `DatabaseAndGroupSelector` component. The three existing inlined copies stay; this migration may use a fourth inline copy or reuse one of the existing ones, but does not introduce a shared abstraction.
- Changing the apply-to-database RPC sequence (`planServiceClientConnect.createPlan` → `rolloutServiceClientConnect.createRollout` → `buildPlanDeployRoute`).
- Changing the release fetch/store contract. `useReleaseStore()` methods remain the only release I/O.
- Redesigning the page layout, copy, or flow. This is a port, not a redesign.
- Adding new product capabilities beyond Vue parity (file download, diff, etc.).
- Migrating the `release.detail` route name constant or its route-helper call sites.
- Touching `frontend/src/utils/sso.ts`, auth stores, or anything in `frontend/src/router/index.ts`.

## Open Questions

1. Should the route's `component` continue to be a scoped Vue file (an `index.ts` that re-exports a `ReactPageMount.vue`-mounting SFC) or switch to the sibling convention used by `PROJECT_V1_ROUTE_RELEASES` (inline `() => import("@/react/ReactPageMount.vue")` with `page: "ProjectReleaseDetailPage"`)? (default: switch to the sibling convention — it is the pattern used by every migrated route in `projectV1.ts` including `PROJECT_V1_ROUTE_RELEASES` at line 506 and `PROJECT_V1_ROUTE_PLAN_DETAIL`, and lets the `frontend/src/components/Release/ReleaseDetail/index.ts` shim go away in the same PR.)
2. Where should the React page live — `frontend/src/react/pages/project/ProjectReleaseDetailPage.tsx` as a single file, or a `release-detail/` subtree mirroring `plan-detail/`? (default: start single-file given the surface is ~730 LOC with four natural subcomponents; split into a `release-detail/components/` subtree only if the single file crosses ~500 LOC. `revision/RevisionDetailPanel.tsx` is precedent for keeping a similar-size detail page mostly single-file.)
3. Should the apply-to-database drawer use `<Sheet width="wide">` (matches Vue's 75vw) or `standard` (704px, default)? (default: `wide` (832px) — the Vue drawer is explicitly `w-240 lg:w-240 max-w-[calc(100vw-8rem)]`, which is closer to the 832px tier; `DataExportPrepSheet` sets the precedent for wide apply-style sheets.)
4. Should the file detail sheet use `Sheet` or `Dialog`? (default: `Sheet` with `width="wide"` to match the Vue `75vw` drawer — the content is a readonly statement viewer, not a form, but the Vue parity target is drawer-shaped and frontend/AGENTS.md permits `Sheet` for "read-only display sheets" at the `narrow` tier; `wide` is used here because the Monaco viewer dominates the panel.)
5. Should `ReleaseFileTable` be re-implemented inline in React (since the Vue file must stay alive for `CreateRevisionDrawer`), or should the React version live at `frontend/src/react/components/release/ReleaseFileTable.tsx` so it can be reused later? (default: implement at `frontend/src/react/components/release/ReleaseFileTable.tsx` — it will be reused when `CreateRevisionDrawer` migrates, matches the plan-detail convention of extracting leaves under a scoped directory, and keeps the page file focused on layout.)
6. Should the naive-ui `useDialog().warning` abandon-confirm flow become a React `AlertDialog`? (default: yes — `AlertDialog` is the right pick for destructive confirms per frontend/AGENTS.md; the same pattern is used by `ProjectReleaseDashboardPage` for archival actions on siblings.)
7. Should the React page fetch release + project itself, or route through a React context provider analogous to `provideReleaseDetailContext`? (default: fetch directly in the page component — the Vue context only holds three values (`release`, `project`, `allowApply`) and has one consumer tree; React Context is unnecessary at this size, playbook §Route Migration Pattern explicitly calls for self-contained pages.)
8. Should the abandon / restore actions gate behind a permission check (`bb.releases.delete` / `bb.releases.undelete`)? (default: match Vue behavior — Vue does not gate these buttons, so React mirrors that; any permission tightening is out of scope for a parity port.)

## Scope

**L** — eight Vue files totaling ~730 LOC span a route page, a provider pattern, a naive-ui-dialog-driven destructive flow, a MonacoEditor-backed file panel, and a cross-subtree leaf (`ReleaseFileTable.vue`) that must stay alive for `CreateRevisionDrawer.vue`. Multiple viable approaches exist for the file-table sharing strategy, drawer-sheet mapping, and whether to introduce a React context provider. The surface is not novel — the plan detail and release dashboard migrations establish the core patterns — but the deletion-rule coupling and RPC-heavy apply flow justify a design pass before execution.

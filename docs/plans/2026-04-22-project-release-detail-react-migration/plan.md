## Task List

**Task Index**: T1: React `ReleaseFileTable.tsx` [M] — T2: React `ReleaseFileDetailPanel.tsx` [M] — T3: `ProjectReleaseDetailPage.tsx` [L] — T4: Add missing React i18n keys [S] — T5: Switch route entry to React [S] — T6: Delete obsolete Vue files [S] — T7: `ReleaseFileTable.test.tsx` leaf test [M] — T8: Frontend CI gates [S]

---

### T1: Create React `ReleaseFileTable.tsx` [M]

**Objective**: Mirror `ReleaseFileTable.vue` as a React leaf so the React page can render it and `CreateRevisionDrawer.vue` can later switch to it. Traces to design §4.

**Size**: M (one new file, ~160 LOC, moderate logic: selection sync, row-click modes).

**Files**:
- Create: `frontend/src/react/components/release/ReleaseFileTable.tsx`

**Implementation**:
1. Port the Vue component verbatim to React using `Table`, `TableHeader`, `TableBody`, `TableRow`, `TableCell`, `TableHead` from `@/react/components/ui/table`.
2. Props contract (matches design §4):
   ```ts
   export interface ReleaseFileTableProps {
     files: Release_File[];
     releaseType: Release_Type;
     showSelection?: boolean;       // default false
     rowClickable?: boolean;        // default true
     selectedFiles?: Release_File[];// default []
     onRowClick?: (file: Release_File, e: React.MouseEvent) => void;
     onSelectedFilesChange?: (files: Release_File[]) => void;
   }
   ```
3. Selection state: local `useState<Set<string>>` of selected paths, synced from `selectedFiles` prop via `useEffect`.
4. Columns: selection (when `showSelection`), version, type (via ported `getReleaseFileTypeText`), filename. Use `w-*` utility widths to approximate Vue widths (160/64/128).
5. Selection column uses a native `<input type="checkbox">` (no React `Checkbox` primitive exists in `ui/`); wrap in `<td className="px-4 py-3">` with `onClick` stopping propagation. (Per frontend/AGENTS.md: only `Input` is required for text-style inputs; checkboxes are OK as native.)
6. Row click mirrors `ReleaseFileTable.vue:104-137` — in selection mode, toggle; otherwise call `onRowClick`. Cursor style when `rowClickable || showSelection`.
7. `i18n`: use `useTranslation()` for `common.version`, `common.type`, `database.revision.filename`, `issue.title.change-database`.

**Boundaries**:
- Do not delete `ReleaseFileTable.vue`. It still has a live Vue caller at `CreateRevisionDrawer.vue:414`.
- Do not export from a barrel file; import by path. No `index.ts` for the directory.
- Do not re-implement `Release_Type` switch logic differently — port the exact 4-case switch from the Vue file.

**Dependencies**: None.

**Expected Outcome**: New file exists, exports `ReleaseFileTable`, compiles under the React tsconfig.

**Validation**: `pnpm --dir frontend type-check` — zero errors.

---

### T2: Create React `ReleaseFileDetailPanel.tsx` [M]

**Objective**: Mirror `ReleaseFileDetailPanel.vue` for use inside the page's file detail sheet. Traces to design §5.

**Size**: M (one new file, ~90 LOC, moderate logic: async sheet fetch with race-guard).

**Files**:
- Create: `frontend/src/react/components/release/ReleaseFileDetailPanel.tsx`

**Implementation**:
1. Props: `{ release: Release; releaseFile: Release_File }`.
2. State: `statement: string`, `loading: boolean`, both `useState`.
3. Effect: on `releaseFile` change, set `let cancelled = false` guard, call `sheetServiceClientConnect.getSheet({ name: releaseFile.sheet, raw: true })`, `new TextDecoder().decode(sheet.content)` into `statement` if not cancelled. Cleanup sets `cancelled = true`. Matches `RevisionDetailPanel.tsx:92-135` pattern.
4. Render structure (ported from `ReleaseFileDetailPanel.vue`):
   - Header row: `common.version`: `{releaseFile.version}` (bold).
   - Meta row: `database.revision.filename: {releaseFile.path}` + `Hash: {releaseFile.sheetSha256.slice(0,8)}`.
   - Separator (use `<Separator />` from `@/react/components/ui/separator`).
   - Statement section: label + inline `CopyButton` (port the inline `CopyButton` from `revision/RevisionDetailPanel.tsx:46-75` verbatim into this file — do not share).
   - `ReadonlyMonaco` from `@/react/components/monaco`, `className="relative h-auto max-h-[480px] min-h-[120px]"`.
5. Loading indicator: render `Loader2` from lucide with `animate-spin` above Monaco while `loading`.
6. `i18n`: `common.version`, `common.statement`, `database.revision.filename`.

**Boundaries**:
- Do not delete `ReleaseFileDetailPanel.vue` yet — deletion happens in T6.
- Do not export a shared React `CopyButton`. Inline it per the design non-goal.
- Do not call any Vue Drawer/DrawerContent — the sheet wrapper is owned by the page.

**Dependencies**: None.

**Expected Outcome**: New file exists, renders, async fetches on release file changes with cleanup.

**Validation**: `pnpm --dir frontend type-check` — zero errors.

---

### T3: Create `ProjectReleaseDetailPage.tsx` [L]

**Objective**: Self-contained React page that replaces `ReleaseDetail.vue`, `BasicInfo.vue`, `NavBar/*.vue`, and `context.ts`. Traces to design §2, §3, §6.

**Size**: L (one new file, ~500 LOC: page state + header + basic-info + apply-sheet + file detail sheet + inline DatabaseAndGroupSelector).

**Files**:
- Create: `frontend/src/react/pages/project/ProjectReleaseDetailPage.tsx`

**Implementation**:
1. Exported signature:
   ```ts
   export function ProjectReleaseDetailPage({
     projectId,
     releaseId,
   }: {
     projectId: string;
     releaseId: string;
   })
   ```
2. Page-level state:
   - `releaseName = \`${projectNamePrefix}${projectId}/releases/${releaseId}\``
   - `projectName = \`${projectNamePrefix}${projectId}\``
   - `const release = useVueState(() => releaseStore.getReleaseByName(releaseName) ?? unknownRelease())`
   - `const project = useVueState(() => projectV1Store.getProjectByName(projectName) ?? unknownProject())`
   - `useEffect` with `ignore` guard: `projectV1Store.getOrFetchProjectByName(projectName)` and `releaseStore.fetchReleaseByName(releaseName)`.
   - `allowApply = hasPermissionToCreateChangeDatabaseIssueInProject(project)` (computed inline each render).
   - Document title `useEffect` on `project.title`: call `setDocumentTitle(t("release.releases"), project.title)` — the route is always the detail route when this page is mounted, so skip the Vue route-name check.
   - `const [selectedReleaseFile, setSelectedReleaseFile] = useState<Release_File | undefined>()` for the file detail sheet.
   - `const [applyOpen, setApplyOpen] = useState(false)` for the apply-to-database sheet.
   - `const [abandonOpen, setAbandonOpen] = useState(false)` for the abandon confirm.
3. Page layout (mirrors `ReleaseDetail.vue:1-11`):
   - Outer `<div className="flex flex-col items-start gap-y-4 p-4 relative">`.
   - `<ReleaseHeader>` (local), `<ReleaseBasicInfo>` (local), `<ReleaseFileTable>` (from T1) with `onRowClick={(f) => setSelectedReleaseFile(f)}`.
4. `<ReleaseHeader>` local component:
   - If `release.state === State.DELETED`, render a sticky `<div className="h-8 w-full text-base font-medium bg-gray-700 text-white flex justify-center items-center">{t("common.archived")}</div>` (port of `ArchiveBanner.vue`).
   - Title row: `<h1 className="text-xl font-medium truncate p-0.5">{releaseName.split('/').pop()}</h1>`.
   - Right side: `<Button>{t("common.apply-to-database")}</Button>` when `release.state === State.ACTIVE` (opens apply sheet) + `<DropdownMenu>` with `abandon`/`restore` depending on state. Use `@/react/components/ui/dropdown-menu`.
   - Abandon selection opens `AlertDialog` with title `t("bbkit.confirm-button.sure-to-abandon")`, description `t("bbkit.confirm-button.can-undo")`, Cancel / Confirm footer. On confirm: `releaseStore.deleteRelease(release.name)`.
   - Restore selection: directly `releaseStore.undeleteRelease(release.name)` (no confirm, mirrors Vue).
5. `<ReleaseBasicInfo>` local component:
   - Row with `Clock4` icon + `HumanizeTs ts={getTimeForPbTimestampProtoEs(release.createTime)/1000}`.
   - If `release.vcsSource?.vcsType !== VCSType.VCS_TYPE_UNSPECIFIED`: render a small row with `Link2` icon and `<a href={release.vcsSource.url} target="_blank">{beautifyUrl(release.vcsSource.url)}</a>`. Port `beautifyUrl` inline as a local function.
6. `<ApplyToDatabaseSheet>` local component:
   - `<Sheet open={applyOpen} onOpenChange={(o) => !o && setApplyOpen(false)}>` + `<SheetContent width="wide">`.
   - Header: `SheetTitle` `t("common.apply-to-database")`.
   - Body: inline `<DatabaseAndGroupSelector>` ported from `ProjectPlanDashboardPage.tsx:881-949` (copy `DatabaseAndGroupSelector` + `DatabaseSelector` + `DatabaseGroupSelector` into this file, bound to local state). Footer status text uses `t("database.selected-n-databases", { n })`.
   - Footer: Cancel + Create buttons, disabled unless selection is valid.
   - On Create: port the RPC sequence from `ApplyToDatabasePanel.vue:103-154` verbatim — `create(PlanSchema, { title, description, specs: [changeDatabaseConfig with release]})`, `planServiceClientConnect.createPlan`, `rolloutServiceClientConnect.createRollout`, then `router.push(buildPlanDeployRoute({ projectId, planId }))`.
7. File detail sheet:
   - `<Sheet open={!!selectedReleaseFile} onOpenChange={(o) => !o && setSelectedReleaseFile(undefined)}>` + `<SheetContent width="wide">`.
   - `<SheetHeader><SheetTitle>{t("release.file") ?? "Release File"}</SheetTitle></SheetHeader>`. Use the existing key `release.file` if present, otherwise inline literal "Release File" (Vue did literal text).
   - Body: `<ReleaseFileDetailPanel release={release} releaseFile={selectedReleaseFile!} />` from T2.
8. Imports:
   ```ts
   import { create } from "@bufbuild/protobuf";
   import { Clock4, Link2, MoreVertical } from "lucide-react";
   import { v4 as uuidv4 } from "uuid";
   import { useEffect, useMemo, useState } from "react";
   import { useTranslation } from "react-i18next";
   import { planServiceClientConnect, rolloutServiceClientConnect } from "@/connect";
   import { HumanizeTs } from "@/react/components/HumanizeTs";
   import { ReleaseFileDetailPanel } from "@/react/components/release/ReleaseFileDetailPanel";
   import { ReleaseFileTable } from "@/react/components/release/ReleaseFileTable";
   import { AlertDialog, AlertDialogContent, AlertDialogDescription, AlertDialogFooter, AlertDialogTitle } from "@/react/components/ui/alert-dialog";
   import { Button } from "@/react/components/ui/button";
   import { DropdownMenu, DropdownMenuContent, DropdownMenuItem, DropdownMenuTrigger } from "@/react/components/ui/dropdown-menu";
   import { Sheet, SheetBody, SheetContent, SheetFooter, SheetHeader, SheetTitle } from "@/react/components/ui/sheet";
   import { useVueState } from "@/react/hooks/useVueState";
   import { cn } from "@/react/lib/utils";
   import { router } from "@/router";
   import { buildPlanDeployRoute } from "@/router/dashboard/projectV1RouteHelpers";
   import { useProjectV1Store, useReleaseStore, useDatabaseV1Store, useDBGroupStore } from "@/store";
   import { projectNamePrefix } from "@/store/modules/v1/common";
   import { getTimeForPbTimestampProtoEs, unknownProject, unknownRelease } from "@/types";
   import { State, VCSType } from "@/types/proto-es/v1/common_pb";
   import { CreatePlanRequestSchema, Plan_ChangeDatabaseConfigSchema, PlanSchema } from "@/types/proto-es/v1/plan_service_pb";
   import type { Release, Release_File } from "@/types/proto-es/v1/release_service_pb";
   import { CreateRolloutRequestSchema } from "@/types/proto-es/v1/rollout_service_pb";
   import { hasPermissionToCreateChangeDatabaseIssueInProject, setDocumentTitle } from "@/utils";
   ```
   (Imports actually needed for `DatabaseAndGroupSelector` inline copy will be expanded to match the source in `ProjectPlanDashboardPage.tsx`.)

**Boundaries**:
- Do not introduce `React.createContext` or any new store. Pinia is the only data source.
- Do not extract a shared `DatabaseAndGroupSelector`. Copy the three-function block (`DatabaseAndGroupSelector`, `DatabaseSelector`, `DatabaseGroupSelector`) from `ProjectPlanDashboardPage.tsx` inline.
- Do not introduce `tanstack/query`, `zustand`, or any new state library.
- Do not add a permission gate around abandon/restore beyond what Vue had.
- Do not touch Monaco internals.

**Dependencies**: T1, T2.

**Expected Outcome**: `frontend/src/react/mount.ts` can resolve `page: "ProjectReleaseDetailPage"` automatically via the `./pages/project/*.tsx` glob.

**Validation**: `pnpm --dir frontend type-check` — zero errors.

---

### T4: Add missing React i18n keys [S]

**Objective**: Make the keys consumed by T1/T2/T3 resolve in all five React locale files. Traces to design §7, design goal 5.

**Files**:
- Modify: `frontend/src/react/locales/en-US.json`
- Modify: `frontend/src/react/locales/es-ES.json`
- Modify: `frontend/src/react/locales/ja-JP.json`
- Modify: `frontend/src/react/locales/vi-VN.json`
- Modify: `frontend/src/react/locales/zh-CN.json`

**Implementation**: For each locale file, add the missing keys (copying values verbatim from `frontend/src/locales/<lang>.json`):

| Key | en-US | es-ES | ja-JP | vi-VN | zh-CN |
|---|---|---|---|---|---|
| `common.version` | Version | Versión | バージョン | Phiên bản | 版本 |
| `common.apply-to-database` | Apply to database | Aplicar a la base de datos | データベースに適用 | Áp dụng cho cơ sở dữ liệu | 应用到数据库 |
| `common.abandon` | Abandon | Abandonar | 破棄 | Hủy bỏ | 废弃 |
| `database.revision.filename` | Filename | Nombre del archivo | ファイル名 | Tên tệp | 文件名 |
| `issue.title.change-database` | Change database | Modificar base de datos | データベースの変更 | Thay đổi cơ sở dữ liệu | 变更数据库 |
| `bbkit.confirm-button.sure-to-abandon` | Are you sure to abandon? | ¿Estás seguro de abandonar? | 本当に破棄しますか? | Bạn có chắc chắn muốn hủy bỏ? | 确定要废弃吗？ |
| `bbkit.confirm-button.can-undo` | You can undo this action later. | Podrás deshacer esta acción más tarde. | この操作は後で元に戻すことができます。 | Bạn có thể hoàn tác hành động này sau. | 此操作可以撤销。 |

Keep existing alphabetical/group ordering. `common.restore`, `common.cancel`, `common.confirm`, `common.create`, `common.type`, `common.statement`, `common.copy`, `common.copied`, `common.close`, `common.archived`, `release.releases`, `release.files`, `database.selected-n-databases` — already present, do not duplicate.

**Validation**: `pnpm --dir frontend check` — zero errors (runs `check-react-i18n`).

---

### T5: Switch route entry to React [S]

**Objective**: Make `PROJECT_V1_ROUTE_RELEASE_DETAIL` mount the new React page. Traces to design §1, design goal 3.

**Files**:
- Modify: `frontend/src/router/dashboard/projectV1.ts`

**Implementation**: Replace lines 512-520:

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

This matches the sibling `PROJECT_V1_ROUTE_RELEASES` pattern at lines 500-511.

**Validation**: `pnpm --dir frontend type-check` — zero errors. `rg "Release/ReleaseDetail/\"" frontend/src/router` — no output (route no longer imports the Vue tree).

---

### T6: Delete obsolete Vue files [S]

**Objective**: Remove Vue files with no remaining callers. Preserve `ReleaseFileTable.vue` because `CreateRevisionDrawer.vue:414` still imports it. Traces to design §9, design goal 2.

**Files**:
- Delete: `frontend/src/components/Release/ReleaseDetail/index.ts`
- Delete: `frontend/src/components/Release/ReleaseDetail/ReleaseDetail.vue`
- Delete: `frontend/src/components/Release/ReleaseDetail/context.ts`
- Delete: `frontend/src/components/Release/ReleaseDetail/BasicInfo.vue`
- Delete: `frontend/src/components/Release/ReleaseDetail/NavBar/index.ts`
- Delete: `frontend/src/components/Release/ReleaseDetail/NavBar/NavBar.vue`
- Delete: `frontend/src/components/Release/ReleaseDetail/NavBar/ApplyToDatabaseButton.vue`
- Delete: `frontend/src/components/Release/ReleaseDetail/NavBar/ApplyToDatabasePanel.vue`
- Delete: `frontend/src/components/Release/ReleaseDetail/ReleaseFileTable/ReleaseFileDetailPanel.vue`

Keep:
- `frontend/src/components/Release/ReleaseDetail/ReleaseFileTable/index.ts`
- `frontend/src/components/Release/ReleaseDetail/ReleaseFileTable/ReleaseFileTable.vue`

**Implementation**: `rm` each file above. Use `rmdir` on `NavBar/` if it empties out (it will).

**Validation**:
- `rg "Release/ReleaseDetail/index" frontend/src` — no hits.
- `rg "Release/ReleaseDetail/ReleaseDetail" frontend/src` — no hits.
- `rg "Release/ReleaseDetail/BasicInfo" frontend/src` — no hits.
- `rg "Release/ReleaseDetail/NavBar" frontend/src` — no hits.
- `rg "Release/ReleaseDetail/context" frontend/src` — no hits.
- `rg "ReleaseFileDetailPanel.vue" frontend/src` — no hits.
- `rg "ReleaseFileTable" frontend/src` — only hits are `CreateRevisionDrawer.vue:414` (Vue) + T1's new React file + (if present) a test file.

---

### T7: Leaf test — `ReleaseFileTable.test.tsx` [M]

**Objective**: Contract test for the React `ReleaseFileTable`. Traces to design §8 (primary leaf test; page-level and detail-panel tests are out of scope per the plan tail).

**Size**: M (one new file, ~120 LOC, isolated contract test with no RPC mocking).

**Files**:
- Create: `frontend/src/react/components/release/ReleaseFileTable.test.tsx`

**Implementation**:
1. Setup: `IS_REACT_ACT_ENVIRONMENT = true`; import `createElement`, `createRoot`, `act`.
2. Mock `react-i18next` so `useTranslation` returns `{ t: (k) => k }` — no real locale loading.
3. Three tests (each < 30 LOC):
   - renders version/type/filename cells for each `Release_File`.
   - hides the selection column by default; shows when `showSelection=true`.
   - clicking a row in non-selection mode fires `onRowClick` with the file.
4. Use minimal test fixtures: two `Release_File` objects with only `path`, `version`, `sheetSha256`, `sheet` populated; `releaseType = Release_Type.VERSIONED`.

**Boundaries**:
- Do not mock `ui/table` internals — render the real components.
- Do not test selection toggling edge cases beyond hide/show (the Vue version did not have tests either — this is parity plus a minimum guard).

**Dependencies**: T1.

**Expected Outcome**: Test file exists, passes with `pnpm --dir frontend test ReleaseFileTable`.

**Validation**: `pnpm --dir frontend test ReleaseFileTable.test` — 3/3 tests pass.

---

### T8: Frontend CI gates [S]

**Objective**: Confirm `fix`, `check`, `type-check`, `test` all pass. Traces to design goal 6.

**Files**: none modified (fix may auto-edit).

**Implementation**: Run in order:
1. `pnpm --dir frontend fix` — format + lint + organize imports, auto-fixes only.
2. `pnpm --dir frontend check` — CI-mode validation (no modifications).
3. `pnpm --dir frontend type-check` — TypeScript/vue-tsc.
4. `pnpm --dir frontend test` — full vitest suite.

If `fix` edits files, re-run `check` and `type-check` once more.

**Validation**: All four commands exit 0. No new lint errors beyond any pre-existing ones on `main`.

---

## Out-of-Scope Tasks

- **Page-level test (`ProjectReleaseDetailPage.test.tsx`)** — design §8 listed it, but precedent `ProjectReleaseDashboardPage` and `ProjectPlanDashboardPage` ship without page-level tests, and adding one requires mocking `planServiceClientConnect`, `rolloutServiceClientConnect`, `releaseServiceClientConnect`, `useReleaseStore`, `useProjectV1Store`, `router.push`, `Monaco`, and five Vue-reactive getters. Design goal 1 calls for manual walkthrough, and design goal 6 covers CI gates, so the page is verified at those two levels instead. Defer to a follow-up if needed.
- **`ReleaseFileDetailPanel.test.tsx`** — same reasoning; requires mocking `sheetServiceClientConnect.getSheet` and `ReadonlyMonaco`. Covered by design goal 1 walkthrough.
- **Extracting a shared React `DatabaseAndGroupSelector`** — explicit Non-Goal in both definition and design.
- **Extracting a shared React `CopyButton`** — explicit Non-Goal; each page that needs one ports the inline 30 LOC version.
- **Porting `VCSIcon.vue`, `ArchiveBanner.vue`, `HumanizeDate.vue`, `EllipsisText.vue`, `@/components/v2` `Drawer`/`CopyButton`** — explicit Non-Goal.
- **Migrating `frontend/src/components/Revision/CreateRevisionDrawer.vue`** — explicit Non-Goal. This is the reason `ReleaseFileTable.vue` is kept alive in T6.
- **Changing the apply RPC sequence, adding permission gates on abandon/restore, redesigning layout** — explicit Non-Goals.

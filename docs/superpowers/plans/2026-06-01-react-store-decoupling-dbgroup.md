# DBGroup Store Decoupling Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Remove every React dependency on the Pinia `useDBGroupStore` by porting its API to the existing Zustand `createDBGroupSlice`, then cutting all 17 React consumers over to `useAppStore`.

**Architecture:** The Zustand `useAppStore` already has a partial `DBGroupSlice` (`fetchDBGroup`, `listDBGroupsForProject`). This plan extends that slice to full parity with the Pinia store's public method surface (`getDBGroupByName`, `getOrFetchDBGroupByName`, `fetchDBGroupListByProjectName`, `createDatabaseGroup`, `updateDatabaseGroup`, `deleteDatabaseGroup`, `fetchDatabaseGroupMatchList`), then mechanically replaces `const dbGroupStore = useDBGroupStore()` + `dbGroupStore.X(...)` with `useAppStore.getState().X(...)`. The Pinia `dbGroup.ts` file stays — it still has Vue-side consumers (deleted later, in the routing phase).

**Tech Stack:** Zustand 5 (slice pattern), Connect-ES clients, Vitest, TypeScript.

**Scope note:** This is PR #1 of the React store decoupling effort described in `docs/superpowers/specs/2026-06-01-react-store-decoupling-design.md`. The Vue composables `useDBGroupListByProject` / `useDatabaseGroupByName` exported from `store/modules/dbGroup.ts` have **no React consumers** (verified) and are out of scope.

---

## File Structure

**Modified:**
- `frontend/src/react/stores/app/types.ts` — extend `DBGroupSlice` type (one responsibility: app store type surface).
- `frontend/src/react/stores/app/dbGroup.ts` — implement the new slice methods (one responsibility: dbGroup data access).
- `frontend/src/react/stores/app/index.test.ts` — add tests for the new methods.
- 11 React **source** consumers — swap Pinia store for `useAppStore`.
- 4 React **test** consumers — relocate the dbGroup mock from `@/store` to `@/react/stores/app`.

**Not modified:**
- `frontend/src/store/modules/dbGroup.ts` — Pinia store stays (Vue consumers remain).

**Consumer inventory (source):**

| File | Methods used |
|---|---|
| `react/components/MatchedDatabaseView.tsx` | `fetchDatabaseGroupMatchList` |
| `react/components/DatabaseGroupTable.tsx` | (declares store; verify usage) |
| `react/components/DatabaseGroupForm.tsx` | `getOrFetchDBGroupByName`, `createDatabaseGroup`, `updateDatabaseGroup` |
| `react/lib/plan/issue.ts` | `getDBGroupByName` |
| `react/pages/project/ProjectGitOpsPage.tsx` | (declares store; verify usage) |
| `react/pages/project/ProjectPlanDashboardPage.tsx` | (declares store; verify usage) |
| `react/pages/project/ProjectDatabaseGroupDetailPage.tsx` | `getDBGroupByName`, `getOrFetchDBGroupByName`, `deleteDatabaseGroup` |
| `react/pages/project/ProjectDatabaseGroupsPage.tsx` | `fetchDBGroupListByProjectName`, `deleteDatabaseGroup` |
| `react/pages/project/issue-detail/components/IssueDetailDatabaseChangeView.tsx` | `getDBGroupByName`, `getOrFetchDBGroupByName` (+ threaded param) |
| `react/pages/project/issue-detail/components/IssueDetailDatabaseExportView.tsx` | `getDBGroupByName`, `getOrFetchDBGroupByName` |
| `react/pages/project/plan-detail/utils/rolloutPreview.ts` | `getOrFetchDBGroupByName` |
| `react/pages/project/plan-detail/components/PlanDetailChangesBranch.tsx` | `getDBGroupByName`, `getOrFetchDBGroupByName` (+ threaded param) |
| `react/pages/project/export-center/DataExportPrepSheet.tsx` | `getOrFetchDBGroupByName` |

**Consumer inventory (test):** `ConnectionPane.test.tsx`, `IssueDetailDatabaseChangeView.test.tsx`, `PlanDetailChangesBranch.test.tsx`, `DataExportPrepSheet.test.tsx`.

---

## Task 1: Extend the DBGroupSlice type

**Files:**
- Modify: `frontend/src/react/stores/app/types.ts:296-309` (the `DBGroupSlice` type)

- [ ] **Step 1: Add `ConditionGroupExpr` import**

At the top of `types.ts`, add (grouped with other `@/plugins` / type imports, sorted by path):

```typescript
import type { ConditionGroupExpr } from "@/plugins/cel";
```

- [ ] **Step 2: Extend the `DBGroupSlice` type**

Replace the existing `DBGroupSlice` type body (currently ending after `listDBGroupsForProject`) with:

```typescript
export type DBGroupSlice = {
  dbGroupsByName: Record<string, DatabaseGroup>;
  // Tracks which view (BASIC/FULL) the cached entry was fetched with, so a
  // FULL request (needs `matchedDatabases`) refetches when only BASIC is
  // cached.
  dbGroupViewByName: Record<string, DatabaseGroupView>;
  dbGroupRequests: Record<string, Promise<DatabaseGroup | undefined>>;
  dbGroupErrorsByName: Record<string, Error | undefined>;
  fetchDBGroup: (
    name: string,
    view?: DatabaseGroupView
  ) => Promise<DatabaseGroup | undefined>;
  listDBGroupsForProject: (project: string) => Promise<DatabaseGroup[]>;
  // Synchronous cache read. Returns `unknownDatabaseGroup()` when absent or
  // when a FULL view is requested but only BASIC is cached.
  getDBGroupByName: (name: string, view?: DatabaseGroupView) => DatabaseGroup;
  getOrFetchDBGroupByName: (
    name: string,
    options?: {
      skipCache?: boolean;
      silent?: boolean;
      view?: DatabaseGroupView;
    }
  ) => Promise<DatabaseGroup>;
  fetchDBGroupListByProjectName: (
    projectName: string,
    view: DatabaseGroupView
  ) => Promise<DatabaseGroup[]>;
  createDatabaseGroup: (params: {
    projectName: string;
    databaseGroup: Pick<
      DatabaseGroup,
      "$typeName" | "name" | "title" | "databaseExpr"
    >;
    databaseGroupId: string;
    validateOnly?: boolean;
  }) => Promise<DatabaseGroup>;
  updateDatabaseGroup: (
    databaseGroup: DatabaseGroup,
    updateMask: string[]
  ) => Promise<DatabaseGroup>;
  deleteDatabaseGroup: (name: string) => Promise<void>;
  fetchDatabaseGroupMatchList: (params: {
    projectName: string;
    expr: ConditionGroupExpr;
  }) => Promise<string[]>;
};
```

- [ ] **Step 3: Verify it type-checks (will fail on missing impls — expected)**

Run: `pnpm --dir frontend type-check`
Expected: errors in `dbGroup.ts` — "Property 'getDBGroupByName' is missing in type". This confirms the type is wired; Task 2 implements the methods.

- [ ] **Step 4: Commit**

```bash
git add frontend/src/react/stores/app/types.ts
git commit -m "feat(react/stores): extend DBGroupSlice type to Pinia parity"
```

---

## Task 2: Implement the new slice methods

**Files:**
- Modify: `frontend/src/react/stores/app/dbGroup.ts`
- Test: `frontend/src/react/stores/app/index.test.ts`

- [ ] **Step 1: Write failing tests**

Append to `frontend/src/react/stores/app/index.test.ts` (inside the existing top-level `describe` block, alongside the existing `fetchDBGroup` tests near line 1734). These exercise the sync getter's view-fallback and the cache-on-fetch behavior:

```typescript
describe("dbGroup parity methods", () => {
  it("getDBGroupByName returns unknownDatabaseGroup when absent", () => {
    const store = createAppStore();
    const group = store.getState().getDBGroupByName("projects/p/databaseGroups/x");
    expect(isValidDatabaseGroupName(group.name)).toBe(false);
  });

  it("getDBGroupByName with FULL view misses a BASIC-only cache entry", () => {
    const store = createAppStore();
    const name = "projects/p/databaseGroups/g";
    store.setState({
      dbGroupsByName: {
        [name]: create(DatabaseGroupSchema, { name, title: "g" }),
      },
      dbGroupViewByName: { [name]: DatabaseGroupView.BASIC },
    });
    // BASIC request is satisfied:
    expect(store.getState().getDBGroupByName(name).name).toBe(name);
    // FULL request is not satisfied by a BASIC entry:
    const full = store
      .getState()
      .getDBGroupByName(name, DatabaseGroupView.FULL);
    expect(isValidDatabaseGroupName(full.name)).toBe(false);
  });

  it("getOrFetchDBGroupByName returns cached entry without a request", async () => {
    const store = createAppStore();
    const name = "projects/p/databaseGroups/g";
    store.setState({
      dbGroupsByName: {
        [name]: create(DatabaseGroupSchema, { name, title: "g" }),
      },
      dbGroupViewByName: { [name]: DatabaseGroupView.BASIC },
    });
    const group = await store.getState().getOrFetchDBGroupByName(name);
    expect(group.name).toBe(name);
  });
});
```

Ensure the test file imports `create` from `@bufbuild/protobuf`, `DatabaseGroupSchema` + `DatabaseGroupView` from `@/types/proto-es/v1/database_group_service_pb`, and `isValidDatabaseGroupName` from `@/types/dbGroup` (add any that are missing to the existing import block).

- [ ] **Step 2: Run tests to verify they fail**

Run: `pnpm --dir frontend test -- src/react/stores/app/index.test.ts`
Expected: FAIL — `getDBGroupByName is not a function` / `getOrFetchDBGroupByName is not a function`.

- [ ] **Step 3: Implement the methods**

Replace the entire body of `frontend/src/react/stores/app/dbGroup.ts` with:

```typescript
import { create as createProto } from "@bufbuild/protobuf";
import { createContextValues } from "@connectrpc/connect";
import { head } from "lodash-es";
import { databaseGroupServiceClientConnect } from "@/connect";
import { silentContextKey } from "@/connect/context-key";
import { buildCELExpr } from "@/plugins/cel";
import { isValidDatabaseGroupName, unknownDatabaseGroup } from "@/types/dbGroup";
import { ExprSchema } from "@/types/proto-es/google/type/expr_pb";
import {
  type DatabaseGroup,
  CreateDatabaseGroupRequestSchema,
  DatabaseGroupSchema,
  DatabaseGroupView,
  DeleteDatabaseGroupRequestSchema,
  GetDatabaseGroupRequestSchema,
  ListDatabaseGroupsRequestSchema,
  UpdateDatabaseGroupRequestSchema,
} from "@/types/proto-es/v1/database_group_service_pb";
import { batchConvertParsedExprToCELString } from "@/utils";
import { databaseGroupNamePrefix } from "@/store/modules/v1/common";
import type { AppSliceCreator, DBGroupSlice } from "./types";
import { toError } from "./utils";

export const createDBGroupSlice: AppSliceCreator<DBGroupSlice> = (set, get) => {
  // Immutable cache write. A name maps to a single group + the view it was
  // fetched with; overwriting is sufficient (no per-view keying needed).
  const setCache = (group: DatabaseGroup, view: DatabaseGroupView) => {
    set((state) => ({
      dbGroupsByName: { ...state.dbGroupsByName, [group.name]: group },
      dbGroupViewByName: { ...state.dbGroupViewByName, [group.name]: view },
    }));
  };
  const removeCache = (name: string) => {
    set((state) => {
      const { [name]: _g, ...dbGroupsByName } = state.dbGroupsByName;
      const { [name]: _v, ...dbGroupViewByName } = state.dbGroupViewByName;
      return { dbGroupsByName, dbGroupViewByName };
    });
  };

  return {
    dbGroupsByName: {},
    dbGroupViewByName: {},
    dbGroupRequests: {},
    dbGroupErrorsByName: {},

    fetchDBGroup: async (name, view = DatabaseGroupView.BASIC) => {
      if (!isValidDatabaseGroupName(name)) return undefined;
      const existing = get().dbGroupsByName[name];
      if (
        existing &&
        (view !== DatabaseGroupView.FULL ||
          get().dbGroupViewByName[name] === DatabaseGroupView.FULL)
      ) {
        return existing;
      }
      const pending = get().dbGroupRequests[name];
      if (pending) return pending;

      const request = databaseGroupServiceClientConnect
        .getDatabaseGroup(
          createProto(GetDatabaseGroupRequestSchema, { name, view })
        )
        .then((group: DatabaseGroup) => {
          set((state) => {
            const { [name]: _, ...dbGroupRequests } = state.dbGroupRequests;
            return {
              dbGroupsByName: { ...state.dbGroupsByName, [group.name]: group },
              dbGroupViewByName: {
                ...state.dbGroupViewByName,
                [group.name]: view,
              },
              dbGroupErrorsByName: {
                ...state.dbGroupErrorsByName,
                [name]: undefined,
              },
              dbGroupRequests,
            };
          });
          return group;
        })
        .catch((error) => {
          set((state) => {
            const { [name]: _, ...dbGroupRequests } = state.dbGroupRequests;
            return {
              dbGroupErrorsByName: {
                ...state.dbGroupErrorsByName,
                [name]: toError(error),
              },
              dbGroupRequests,
            };
          });
          return undefined;
        });
      set((state) => ({
        dbGroupRequests: { ...state.dbGroupRequests, [name]: request },
      }));
      return request;
    },

    listDBGroupsForProject: async (project) => {
      const response =
        await databaseGroupServiceClientConnect.listDatabaseGroups(
          createProto(ListDatabaseGroupsRequestSchema, { parent: project })
        );
      for (const group of response.databaseGroups) {
        setCache(group, DatabaseGroupView.BASIC);
      }
      return response.databaseGroups;
    },

    getDBGroupByName: (name, view = DatabaseGroupView.UNSPECIFIED) => {
      const group = get().dbGroupsByName[name];
      if (group) {
        const cachedView = get().dbGroupViewByName[name];
        const satisfied =
          view === DatabaseGroupView.FULL
            ? cachedView === DatabaseGroupView.FULL
            : true; // UNSPECIFIED / BASIC are satisfied by either view
        if (satisfied) return group;
      }
      return unknownDatabaseGroup();
    },

    getOrFetchDBGroupByName: async (name, options) => {
      const {
        skipCache = false,
        silent = false,
        view = DatabaseGroupView.BASIC,
      } = options ?? {};
      if (!skipCache) {
        const cached = get().getDBGroupByName(name, view);
        if (isValidDatabaseGroupName(cached.name)) {
          return cached;
        }
      }
      const group = await databaseGroupServiceClientConnect.getDatabaseGroup(
        createProto(GetDatabaseGroupRequestSchema, { name, view }),
        { contextValues: createContextValues().set(silentContextKey, silent) }
      );
      setCache(group, view);
      return group;
    },

    fetchDBGroupListByProjectName: async (projectName, view) => {
      const response =
        await databaseGroupServiceClientConnect.listDatabaseGroups(
          createProto(ListDatabaseGroupsRequestSchema, {
            parent: projectName,
            view,
          })
        );
      for (const group of response.databaseGroups) {
        setCache(group, view);
      }
      return response.databaseGroups;
    },

    createDatabaseGroup: async ({
      projectName,
      databaseGroup,
      databaseGroupId,
      validateOnly = false,
    }) => {
      const newDatabaseGroup = createProto(DatabaseGroupSchema, {
        name: databaseGroup.name,
        title: databaseGroup.title,
        databaseExpr: databaseGroup.databaseExpr,
        matchedDatabases: [],
      });
      const created =
        await databaseGroupServiceClientConnect.createDatabaseGroup(
          createProto(CreateDatabaseGroupRequestSchema, {
            parent: projectName,
            databaseGroup: newDatabaseGroup,
            databaseGroupId,
            validateOnly,
          }),
          {
            contextValues: createContextValues().set(
              silentContextKey,
              validateOnly
            ),
          }
        );
      if (!validateOnly) {
        setCache(created, DatabaseGroupView.FULL);
      }
      return created;
    },

    updateDatabaseGroup: async (databaseGroup, updateMask) => {
      const updated =
        await databaseGroupServiceClientConnect.updateDatabaseGroup(
          createProto(UpdateDatabaseGroupRequestSchema, {
            databaseGroup,
            updateMask: { paths: updateMask },
          })
        );
      setCache(updated, DatabaseGroupView.FULL);
      return updated;
    },

    deleteDatabaseGroup: async (name) => {
      await databaseGroupServiceClientConnect.deleteDatabaseGroup(
        createProto(DeleteDatabaseGroupRequestSchema, { name })
      );
      removeCache(name);
    },

    fetchDatabaseGroupMatchList: async ({ projectName, expr }) => {
      const celexpr = await buildCELExpr(expr);
      if (!celexpr) {
        return [];
      }
      const celStrings = await batchConvertParsedExprToCELString([celexpr]);
      const expression = head(celStrings) || "true";
      const validateOnlyResourceId = `creating-database-group-${Date.now()}`;
      const result = await get().createDatabaseGroup({
        projectName,
        databaseGroup: createProto(DatabaseGroupSchema, {
          name: `${projectName}/${databaseGroupNamePrefix}${validateOnlyResourceId}`,
          title: validateOnlyResourceId,
          databaseExpr: createProto(ExprSchema, { expression }),
        }),
        databaseGroupId: validateOnlyResourceId,
        validateOnly: true,
      });
      return result.matchedDatabases.map((item) => item.name);
    },
  };
};
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `pnpm --dir frontend test -- src/react/stores/app/index.test.ts`
Expected: PASS (new `dbGroup parity methods` block + the pre-existing `fetchDBGroup` tests).

- [ ] **Step 5: Lint + type-check the store layer**

Run: `pnpm --dir frontend type-check`
Expected: PASS for `dbGroup.ts` and `types.ts` (consumer files still reference Pinia — they are migrated in later tasks and remain valid until then).

- [ ] **Step 6: Commit**

```bash
git add frontend/src/react/stores/app/dbGroup.ts frontend/src/react/stores/app/index.test.ts
git commit -m "feat(react/stores): port dbGroup methods to Zustand slice"
```

---

## Consumer migration — transformation rule (applies to Tasks 3–6)

For every **source** consumer:

1. Remove `useDBGroupStore` from the `@/store` (or `@/store/modules` / `@/store/modules/dbGroup`) import. If that import only provided `useDBGroupStore`, delete the whole import line; otherwise keep the other names.
2. Add (or extend an existing) `import { useAppStore } from "@/react/stores/app";`.
3. Delete the `const dbGroupStore = useDBGroupStore();` line.
4. Replace each `dbGroupStore.METHOD(...)` with `useAppStore.getState().METHOD(...)`. Signatures are identical, so arguments are unchanged.
5. Remove `dbGroupStore` from any `useCallback` / `useMemo` / `useEffect` dependency arrays — `useAppStore.getState()` is a stable module-level reference and must not be a dep.
6. For helpers that receive the store as a parameter typed `ReturnType<typeof useDBGroupStore>` (see Task 5): delete that parameter and call `useAppStore.getState()` inside the helper instead; update all call sites.

**Concrete example** (from `ProjectDatabaseGroupsPage.tsx`):

```diff
-import { useVueState } from "@/react/hooks/useVueState";
-import { useDBGroupStore, useProjectV1Store } from "@/store";
+import { useVueState } from "@/react/hooks/useVueState";
+import { useProjectV1Store } from "@/store";
+import { useAppStore } from "@/react/stores/app";
 ...
-  const dbGroupStore = useDBGroupStore();
 ...
-    dbGroupStore
-      .fetchDBGroupListByProjectName(projectName, DatabaseGroupView.BASIC)
+    useAppStore
+      .getState()
+      .fetchDBGroupListByProjectName(projectName, DatabaseGroupView.BASIC)
       .then(...)
-  }, [projectName, dbGroupStore]);
+  }, [projectName]);
 ...
-    await dbGroupStore.deleteDatabaseGroup(deleteTarget.name);
+    await useAppStore.getState().deleteDatabaseGroup(deleteTarget.name);
-  }, [deleteTarget, dbGroupStore]);
+  }, [deleteTarget]);
```

After each task, verify the touched files no longer reference the Pinia store:

Run: `grep -rn "useDBGroupStore\|dbGroupStore" <files in this task>`
Expected: no matches.

---

## Task 3: Migrate straightforward source consumers

**Files (read each, apply the transformation rule):**
- Modify: `frontend/src/react/components/MatchedDatabaseView.tsx` — `fetchDatabaseGroupMatchList` (line ~148)
- Modify: `frontend/src/react/components/DatabaseGroupForm.tsx` — `getOrFetchDBGroupByName`, `createDatabaseGroup`, `updateDatabaseGroup`
- Modify: `frontend/src/react/pages/project/ProjectDatabaseGroupsPage.tsx` — `fetchDBGroupListByProjectName`, `deleteDatabaseGroup`
- Modify: `frontend/src/react/pages/project/ProjectDatabaseGroupDetailPage.tsx` — `getDBGroupByName`, `getOrFetchDBGroupByName`, `deleteDatabaseGroup`
- Modify: `frontend/src/react/pages/project/export-center/DataExportPrepSheet.tsx` — `getOrFetchDBGroupByName` (lines ~99, ~181, ~869)
- Modify: `frontend/src/react/lib/plan/issue.ts` — `getDBGroupByName` (line ~26)
- Modify: `frontend/src/react/pages/project/plan-detail/utils/rolloutPreview.ts` — `getOrFetchDBGroupByName` (line ~106)

- [ ] **Step 1: Apply the transformation rule to each file above.** For `issue.ts` and `rolloutPreview.ts` (plain `.ts` helpers, not components), the same rule applies — `useAppStore.getState()` works outside React render.

- [ ] **Step 2: Verify no Pinia references remain in these files**

Run:
```bash
grep -rn "useDBGroupStore\|dbGroupStore" frontend/src/react/components/MatchedDatabaseView.tsx frontend/src/react/components/DatabaseGroupForm.tsx frontend/src/react/pages/project/ProjectDatabaseGroupsPage.tsx frontend/src/react/pages/project/ProjectDatabaseGroupDetailPage.tsx frontend/src/react/pages/project/export-center/DataExportPrepSheet.tsx frontend/src/react/lib/plan/issue.ts frontend/src/react/pages/project/plan-detail/utils/rolloutPreview.ts
```
Expected: no matches.

- [ ] **Step 3: Type-check**

Run: `pnpm --dir frontend type-check`
Expected: PASS for these files (other unmigrated consumers and their tests may still reference Pinia — that is fine).

- [ ] **Step 4: Commit**

```bash
git add frontend/src/react/components/MatchedDatabaseView.tsx frontend/src/react/components/DatabaseGroupForm.tsx frontend/src/react/pages/project/ProjectDatabaseGroupsPage.tsx frontend/src/react/pages/project/ProjectDatabaseGroupDetailPage.tsx frontend/src/react/pages/project/export-center/DataExportPrepSheet.tsx frontend/src/react/lib/plan/issue.ts frontend/src/react/pages/project/plan-detail/utils/rolloutPreview.ts
git commit -m "refactor(react): migrate straightforward dbGroup consumers to useAppStore"
```

---

## Task 4: Migrate IssueDetailDatabaseExportView

**Files:**
- Modify: `frontend/src/react/pages/project/issue-detail/components/IssueDetailDatabaseExportView.tsx`

This file has two `const dbGroupStore = useDBGroupStore()` declarations (lines ~591, ~681) and several `getDBGroupByName` / `getOrFetchDBGroupByName` calls (lines ~598, ~684, ~696). No threaded param.

- [ ] **Step 1: Apply the transformation rule** to both blocks. Remove both `const dbGroupStore = ...` lines; replace each call with `useAppStore.getState().METHOD(...)`; drop `dbGroupStore` from any dep arrays.

- [ ] **Step 2: Verify no Pinia references remain**

Run: `grep -rn "useDBGroupStore\|dbGroupStore" frontend/src/react/pages/project/issue-detail/components/IssueDetailDatabaseExportView.tsx`
Expected: no matches.

- [ ] **Step 3: Type-check**

Run: `pnpm --dir frontend type-check`
Expected: PASS for this file.

- [ ] **Step 4: Commit**

```bash
git add frontend/src/react/pages/project/issue-detail/components/IssueDetailDatabaseExportView.tsx
git commit -m "refactor(react): migrate IssueDetailDatabaseExportView dbGroup usage"
```

---

## Task 5: Migrate consumers that thread the store as a parameter

**Files:**
- Modify: `frontend/src/react/pages/project/issue-detail/components/IssueDetailDatabaseChangeView.tsx`
- Modify: `frontend/src/react/pages/project/plan-detail/components/PlanDetailChangesBranch.tsx`

These thread the store into a non-component helper typed `dbGroupStore: ReturnType<typeof useDBGroupStore>` (`IssueDetailDatabaseChangeView.tsx:916`, `PlanDetailChangesBranch.tsx:2041`). Each also has plain in-component usage (`getDBGroupByName`, `getOrFetchDBGroupByName`).

- [ ] **Step 1: Refactor the threaded helpers.** In each helper signature, delete the `dbGroupStore: ReturnType<typeof useDBGroupStore>` parameter. Inside the helper body, replace `dbGroupStore.METHOD(...)` with `useAppStore.getState().METHOD(...)`. Update the call sites to stop passing the store argument.

- [ ] **Step 2: Apply the transformation rule** to the in-component `const dbGroupStore = useDBGroupStore()` declarations and their `.METHOD(...)` calls (including `PlanDetailChangesBranch.tsx:1870` which passes `DatabaseGroupView.FULL`).

- [ ] **Step 3: Verify no Pinia references remain**

Run: `grep -rn "useDBGroupStore\|dbGroupStore" frontend/src/react/pages/project/issue-detail/components/IssueDetailDatabaseChangeView.tsx frontend/src/react/pages/project/plan-detail/components/PlanDetailChangesBranch.tsx`
Expected: no matches.

- [ ] **Step 4: Type-check**

Run: `pnpm --dir frontend type-check`
Expected: PASS for these files.

- [ ] **Step 5: Commit**

```bash
git add frontend/src/react/pages/project/issue-detail/components/IssueDetailDatabaseChangeView.tsx frontend/src/react/pages/project/plan-detail/components/PlanDetailChangesBranch.tsx
git commit -m "refactor(react): migrate threaded dbGroup-store helpers to useAppStore"
```

---

## Task 6: Migrate remaining "declared but verify" source consumers

**Files:**
- Modify: `frontend/src/react/components/DatabaseGroupTable.tsx:65`
- Modify: `frontend/src/react/pages/project/ProjectGitOpsPage.tsx:45`
- Modify: `frontend/src/react/pages/project/ProjectPlanDashboardPage.tsx:1122`

Each declares `const dbGroupStore = useDBGroupStore()`. The earlier inventory grep did not surface a `dbGroupStore.METHOD` call in these — the store may be threaded into JSX/helpers or be dead.

- [ ] **Step 1: For each file, locate every `dbGroupStore` reference.**

Run (per file): `grep -n "dbGroupStore" <file>`

- [ ] **Step 2: Apply the transformation rule.** If the declaration is unused, delete it and the `useDBGroupStore` import (dead code). If used, replace per the rule. If threaded into a helper, follow Task 5's helper-refactor approach.

- [ ] **Step 3: Verify no Pinia references remain**

Run: `grep -rn "useDBGroupStore\|dbGroupStore" frontend/src/react/components/DatabaseGroupTable.tsx frontend/src/react/pages/project/ProjectGitOpsPage.tsx frontend/src/react/pages/project/ProjectPlanDashboardPage.tsx`
Expected: no matches.

- [ ] **Step 4: Type-check**

Run: `pnpm --dir frontend type-check`
Expected: PASS for these files.

- [ ] **Step 5: Commit**

```bash
git add frontend/src/react/components/DatabaseGroupTable.tsx frontend/src/react/pages/project/ProjectGitOpsPage.tsx frontend/src/react/pages/project/ProjectPlanDashboardPage.tsx
git commit -m "refactor(react): migrate remaining dbGroup source consumers to useAppStore"
```

---

## Task 7: Relocate dbGroup mocks in test consumers

**Files:**
- Modify: `frontend/src/react/components/sql-editor/ConnectionPane/ConnectionPane.test.tsx` (lines ~147, ~218)
- Modify: `frontend/src/react/pages/project/issue-detail/components/IssueDetailDatabaseChangeView.test.tsx` (line ~232)
- Modify: `frontend/src/react/pages/project/plan-detail/components/PlanDetailChangesBranch.test.tsx` (lines ~270, ~462, ~472, ~691, ~697)
- Modify: `frontend/src/react/pages/project/export-center/DataExportPrepSheet.test.tsx` (line ~229)

Each currently mocks `useDBGroupStore` inside a `vi.mock("@/store", ...)` factory. The source files now import `useAppStore` from `@/react/stores/app`, so the dbGroup methods must be exposed through a `useAppStore` mock.

- [ ] **Step 1: For each test, move the dbGroup mock to `@/react/stores/app`.**

Remove `useDBGroupStore` from the `@/store` mock factory. Add or extend a `@/react/stores/app` mock that exposes the dbGroup methods via `getState()`. Follow the established pattern in `react/stores/sqlEditor/worksheet.test.ts:85`:

```typescript
vi.mock("@/react/stores/app", () => ({
  useAppStore: { getState: () => mocks.dbGroupStore },
}));
```

If the test ALSO needs the hook-call form (`useAppStore((s) => s.x)`) for other state, make the mock both callable and carry `getState` — e.g.:

```typescript
const useAppStore = Object.assign(
  (selector: (s: AppStoreShape) => unknown) => selector(mocks.appState),
  { getState: () => mocks.appState }
);
vi.mock("@/react/stores/app", () => ({ useAppStore }));
```

Reuse the existing `mocks.dbGroupStore` object (with `getDBGroupByName`, `getOrFetchDBGroupByName`, `fetchDBGroupListByProjectName`, etc.) so the per-test `mockResolvedValue` / `mockReturnValue` setup (e.g. `PlanDetailChangesBranch.test.tsx:462-697`) keeps working — only the mocked module path changes.

- [ ] **Step 2: Run each migrated test file**

Run:
```bash
pnpm --dir frontend test -- src/react/components/sql-editor/ConnectionPane/ConnectionPane.test.tsx src/react/pages/project/issue-detail/components/IssueDetailDatabaseChangeView.test.tsx src/react/pages/project/plan-detail/components/PlanDetailChangesBranch.test.tsx src/react/pages/project/export-center/DataExportPrepSheet.test.tsx
```
Expected: PASS.

- [ ] **Step 3: Commit**

```bash
git add frontend/src/react/components/sql-editor/ConnectionPane/ConnectionPane.test.tsx frontend/src/react/pages/project/issue-detail/components/IssueDetailDatabaseChangeView.test.tsx frontend/src/react/pages/project/plan-detail/components/PlanDetailChangesBranch.test.tsx frontend/src/react/pages/project/export-center/DataExportPrepSheet.test.tsx
git commit -m "test(react): point dbGroup mocks at useAppStore"
```

---

## Task 8: Final verification — React is dbGroup-Pinia-free

- [ ] **Step 1: Assert zero Pinia dbGroup references under react/**

Run:
```bash
grep -rn "useDBGroupStore" frontend/src/react
```
Expected: **no matches.** (If any remain, return to the relevant task.)

- [ ] **Step 2: Confirm the Pinia store still has its Vue consumers (must NOT be deleted)**

Run:
```bash
grep -rln "useDBGroupStore\|useDBGroupListByProject\|useDatabaseGroupByName" frontend/src --include="*.vue" --include="*.ts" | grep -v "/react/"
```
Expected: matches outside `react/` (Vue side). The file `frontend/src/store/modules/dbGroup.ts` is unchanged.

- [ ] **Step 3: Full frontend verification suite**

Run:
```bash
pnpm --dir frontend fix
pnpm --dir frontend check
pnpm --dir frontend type-check
pnpm --dir frontend test
```
Expected: all PASS. (Note: the background-task exit code can be misleading — read the actual log output before claiming a pass.)

- [ ] **Step 4: Final commit (if `fix` changed formatting)**

```bash
git add -A
git commit -m "chore(react): format after dbGroup decoupling"
```

---

## Self-Review Notes

- **Spec coverage:** This plan covers the `dbGroup` row of the spec's scope table (PR #1). The parity step (spec recipe step 1) is Tasks 1–2; consumer cutover (step 2) is Tasks 3–7; verification (step 3) is Task 8. The guard script (spec PR #13) is intentionally out of scope for this PR.
- **Out of scope, confirmed:** Pinia `dbGroup.ts` is not deleted (Task 8 Step 2 guards this); Vue composables `useDBGroupListByProject` / `useDatabaseGroupByName` have no React consumers.
- **Type consistency:** Method names are identical between the Pinia store, the new slice (Task 2), the type (Task 1), and the consumer calls — enabling the mechanical 1:1 rewrite. `getDBGroupByName` default view is `UNSPECIFIED` (sync getter), matching the Pinia store.

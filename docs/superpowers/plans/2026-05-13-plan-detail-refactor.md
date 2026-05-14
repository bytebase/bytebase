# Plan Detail Refactor Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Refactor `frontend/src/react/pages/project/plan-detail/` for maintainability without changing behavior — split oversized files, restructure the page hook, replace Pinia reads with Zustand slices, run `/simplify` per a bounded checklist.

**Architecture:** Single PR carrying the entire refactor. Internally the work is grouped into six phases (Foundation → Shell → Changes → Review → Deploy → Header/Sidebar/cleanup), but everything lands on one feature branch as a single review surface. No intermediate compatibility shims — every consumer updates in the same diff. The CI guard for Pinia reads inside `plan-detail/` is strict from the first commit.

**Tech Stack:** React 18, TypeScript, Zustand v5 (slice pattern), Vitest + React Testing Library, Playwright (e2e), Tailwind v4, Vue 3 Pinia (existing, kept in coexistence for other pages), Connect-RPC service clients.

**Behavioral contract:** `docs/plans/2026-05-13-plan-detail-current-state.md`. If a test fails after a refactor step → fix the regression, not the test.

**Design spec:** `docs/superpowers/specs/2026-05-13-plan-detail-refactor-design.md` (see the 2026-05-13 update noting the one-PR delivery decision).

**Branch:** `refactor/plan-detail-maintainability`

---

## Conventions used in every task

- **Format gate**: After any frontend code change, run `pnpm --dir frontend fix` and `pnpm --dir frontend type-check`. They must produce no diff and zero errors before commit.
- **Test gate**: Run the affected test file(s) with `pnpm --dir frontend test -- <path>`. Existing tests must pass unchanged.
- **No assertion edits**: When a test moves with its subject, only the import path may change.
- **Commit cadence**: Commit after every passing test or every cohesive file move. Small commits keep `git bisect` and per-commit review useful.
- **No compat shims**: Update every consumer in the same diff that moves the file. No one-line re-export files at old paths.
- **Pre-PR**: Walk `docs/pre-pr-checklist.md` before opening the PR (per AGENTS.md).

---

# Phase 1 — Foundation

**Goal:** Land all infrastructure plan-detail will depend on. No existing plan-detail call sites change in this phase.

## Task 1.1: Add `DatabaseSlice` to the app store

**Files:**
- Create: `frontend/src/react/stores/app/database.ts`
- Modify: `frontend/src/react/stores/app/types.ts`
- Modify: `frontend/src/react/stores/app/index.ts`
- Test: extend `frontend/src/react/stores/app/index.test.ts`

The slice mirrors the existing `InstanceSlice` pattern (per-name entity cache + request-promise dedup + error map). Only port methods plan-detail actually calls.

**Important proto naming gotcha (verified 2026-05-13):** the resource-`Database` schema generator is exported as `DatabaseSchema$` in `frontend/src/types/proto-es/v1/database_service_pb.d.ts`; `DatabaseSchema` is a different message (DDL metadata). Use `DatabaseSchema$` with `create(...)` in this slice's tests. Also: the proto only defines `getDatabase`, `listDatabases`, `batchGetDatabases` — there is no `searchDatabases` RPC. The slice should expose `fetchDatabase`, `batchFetchDatabases`, and `fetchDatabases` (wrapping `listDatabases`) with shape `{ parent, pageSize, pageToken?, filter?: string, orderBy? }` — the `filter` is the proto's raw CEL string (Pinia's `DatabaseFilter` is a Vue-layer wrapper that compiles to that string via `getListDatabaseFilter`).

- [ ] **Step 1: Write the failing test**

Add to `frontend/src/react/stores/app/index.test.ts` near the existing instance test:

```ts
test("deduplicates database fetches and caches the result", async () => {
  const dbName = "instances/i1/databases/db1";
  const database = createProto(DatabaseSchema$, { name: dbName });
  mocks.getDatabase.mockResolvedValueOnce(database);
  const store = createAppStore();

  const [first, second, third] = await Promise.all([
    store.getState().fetchDatabase(dbName),
    store.getState().fetchDatabase(dbName),
    store.getState().fetchDatabase(dbName),
  ]);

  expect(first).toEqual(database);
  expect(second).toEqual(database);
  expect(third).toEqual(database);
  expect(mocks.getDatabase).toHaveBeenCalledTimes(1);
  expect(store.getState().databasesByName[dbName]).toEqual(database);
});
```

Add to the `mocks` block:
```ts
  getDatabase: vi.fn(),
  batchGetDatabases: vi.fn(),
  searchDatabases: vi.fn(),
```

Add to the `vi.mock("@/connect", ...)` block:
```ts
  databaseServiceClientConnect: {
    getDatabase: mocks.getDatabase,
    batchGetDatabases: mocks.batchGetDatabases,
    searchDatabases: mocks.searchDatabases,
  },
```

Add to top imports:
```ts
import { DatabaseSchema } from "@/types/proto-es/v1/database_service_pb";
```

- [ ] **Step 2: Run test and verify it fails**

```
pnpm --dir frontend test -- src/react/stores/app/index.test.ts -t "deduplicates database fetches"
```

Expected: FAIL — `fetchDatabase` is not a function.

- [ ] **Step 3: Add `DatabaseSlice` type to `types.ts`**

Insert after `InstanceSlice`:

```ts
import type {
  Database,
} from "@/types/proto-es/v1/database_service_pb";

export type DatabaseSlice = {
  databasesByName: Record<string, Database>;
  databaseRequests: Record<string, Promise<Database | undefined>>;
  databaseErrorsByName: Record<string, Error | undefined>;
  fetchDatabase: (name: string) => Promise<Database | undefined>;
  batchFetchDatabases: (names: string[]) => Promise<Database[]>;
  searchDatabases: (params: {
    parent: string;
    filter?: string;
    pageSize?: number;
    pageToken?: string;
  }) => Promise<{ databases: Database[]; nextPageToken?: string }>;
};
```

Append `DatabaseSlice` to the `AppStoreState` union at the bottom.

- [ ] **Step 4: Implement the slice (`database.ts`)**

```ts
import { create as createProto } from "@bufbuild/protobuf";
import { databaseServiceClientConnect } from "@/connect";
import { isValidDatabaseName } from "@/react/lib/resourceName";
import {
  BatchGetDatabasesRequestSchema,
  GetDatabaseRequestSchema,
  SearchDatabasesRequestSchema,
} from "@/types/proto-es/v1/database_service_pb";
import type { AppSliceCreator, DatabaseSlice } from "./types";

function toError(error: unknown): Error {
  return error instanceof Error ? error : new Error(String(error));
}

export const createDatabaseSlice: AppSliceCreator<DatabaseSlice> = (set, get) => ({
  databasesByName: {},
  databaseRequests: {},
  databaseErrorsByName: {},

  fetchDatabase: async (name) => {
    if (!isValidDatabaseName(name)) return undefined;
    const existing = get().databasesByName[name];
    if (existing) return existing;
    const pending = get().databaseRequests[name];
    if (pending) return pending;

    const request = databaseServiceClientConnect
      .getDatabase(createProto(GetDatabaseRequestSchema, { name }))
      .then((database) => {
        set((state) => {
          const { [name]: _omit, ...rest } = state.databaseRequests;
          return {
            databasesByName: { ...state.databasesByName, [database.name]: database },
            databaseErrorsByName: { ...state.databaseErrorsByName, [name]: undefined },
            databaseRequests: rest,
          };
        });
        return database;
      })
      .catch((error) => {
        set((state) => {
          const { [name]: _omit, ...rest } = state.databaseRequests;
          return {
            databaseErrorsByName: { ...state.databaseErrorsByName, [name]: toError(error) },
            databaseRequests: rest,
          };
        });
        return undefined;
      });

    set((state) => ({
      databaseRequests: { ...state.databaseRequests, [name]: request },
    }));
    return request;
  },

  batchFetchDatabases: async (names) => {
    if (!names.length) return [];
    const response = await databaseServiceClientConnect.batchGetDatabases(
      createProto(BatchGetDatabasesRequestSchema, { names })
    );
    set((state) => {
      const next = { ...state.databasesByName };
      for (const db of response.databases) next[db.name] = db;
      return { databasesByName: next };
    });
    return response.databases;
  },

  searchDatabases: async ({ parent, filter, pageSize, pageToken }) => {
    const response = await databaseServiceClientConnect.searchDatabases(
      createProto(SearchDatabasesRequestSchema, { parent, filter, pageSize, pageToken })
    );
    set((state) => {
      const next = { ...state.databasesByName };
      for (const db of response.databases) next[db.name] = db;
      return { databasesByName: next };
    });
    return { databases: response.databases, nextPageToken: response.nextPageToken };
  },
});
```

- [ ] **Step 5: Wire the slice into `createAppStore`**

In `frontend/src/react/stores/app/index.ts`:

```ts
import { createDatabaseSlice } from "./database";
// ...
export const createAppStore = () =>
  create<AppStoreState>()((...args) => ({
    ...createAuthSlice(...args),
    ...createWorkspaceSlice(...args),
    ...createIamSlice(...args),
    ...createProjectSlice(...args),
    ...createInstanceSlice(...args),
    ...createDatabaseSlice(...args),
    ...createNotificationSlice(...args),
    ...createPreferencesSlice(...args),
  }));
```

- [ ] **Step 6: Run test → PASS. Type-check + format.**

```
pnpm --dir frontend test -- src/react/stores/app/index.test.ts -t "deduplicates database fetches"
pnpm --dir frontend type-check
pnpm --dir frontend fix
```

- [ ] **Step 7: Commit**

```
git add frontend/src/react/stores/app/database.ts frontend/src/react/stores/app/types.ts frontend/src/react/stores/app/index.ts frontend/src/react/stores/app/index.test.ts
git commit -m "feat(react): add DatabaseSlice to app store"
```

## Task 1.2: Add `DBGroupSlice`

**Files:**
- Create: `frontend/src/react/stores/app/dbGroup.ts`
- Modify: `types.ts`, `index.ts`
- Test: extend `index.test.ts`

- [ ] **Step 1: Write the failing test**

```ts
test("deduplicates db group fetches and caches the result", async () => {
  const name = "projects/p1/databaseGroups/g1";
  const group = createProto(DatabaseGroupSchema, { name });
  mocks.getDatabaseGroup.mockResolvedValueOnce(group);
  const store = createAppStore();

  const [first, second] = await Promise.all([
    store.getState().fetchDBGroup(name),
    store.getState().fetchDBGroup(name),
  ]);

  expect(first).toEqual(group);
  expect(second).toEqual(group);
  expect(mocks.getDatabaseGroup).toHaveBeenCalledTimes(1);
});
```

Add `getDatabaseGroup` and `listDatabaseGroups` to `mocks`. Wire `databaseGroupServiceClientConnect`. Import `DatabaseGroupSchema`.

- [ ] **Step 2: Run test → FAIL.**

- [ ] **Step 3: Add `DBGroupSlice` type**

```ts
import type { DatabaseGroup } from "@/types/proto-es/v1/database_group_service_pb";

export type DBGroupSlice = {
  dbGroupsByName: Record<string, DatabaseGroup>;
  dbGroupRequests: Record<string, Promise<DatabaseGroup | undefined>>;
  dbGroupErrorsByName: Record<string, Error | undefined>;
  fetchDBGroup: (name: string) => Promise<DatabaseGroup | undefined>;
  listDBGroupsForProject: (project: string) => Promise<DatabaseGroup[]>;
};
```

Append to `AppStoreState`.

- [ ] **Step 4: Implement `dbGroup.ts`**

```ts
import { create as createProto } from "@bufbuild/protobuf";
import { databaseGroupServiceClientConnect } from "@/connect";
import { isValidDatabaseGroupName } from "@/react/lib/resourceName";
import {
  GetDatabaseGroupRequestSchema,
  ListDatabaseGroupsRequestSchema,
} from "@/types/proto-es/v1/database_group_service_pb";
import type { AppSliceCreator, DBGroupSlice } from "./types";

function toError(e: unknown): Error {
  return e instanceof Error ? e : new Error(String(e));
}

export const createDBGroupSlice: AppSliceCreator<DBGroupSlice> = (set, get) => ({
  dbGroupsByName: {},
  dbGroupRequests: {},
  dbGroupErrorsByName: {},

  fetchDBGroup: async (name) => {
    if (!isValidDatabaseGroupName(name)) return undefined;
    const existing = get().dbGroupsByName[name];
    if (existing) return existing;
    const pending = get().dbGroupRequests[name];
    if (pending) return pending;

    const request = databaseGroupServiceClientConnect
      .getDatabaseGroup(createProto(GetDatabaseGroupRequestSchema, { name }))
      .then((group) => {
        set((state) => {
          const { [name]: _omit, ...rest } = state.dbGroupRequests;
          return {
            dbGroupsByName: { ...state.dbGroupsByName, [group.name]: group },
            dbGroupErrorsByName: { ...state.dbGroupErrorsByName, [name]: undefined },
            dbGroupRequests: rest,
          };
        });
        return group;
      })
      .catch((error) => {
        set((state) => {
          const { [name]: _omit, ...rest } = state.dbGroupRequests;
          return {
            dbGroupErrorsByName: { ...state.dbGroupErrorsByName, [name]: toError(error) },
            dbGroupRequests: rest,
          };
        });
        return undefined;
      });

    set((state) => ({ dbGroupRequests: { ...state.dbGroupRequests, [name]: request } }));
    return request;
  },

  listDBGroupsForProject: async (project) => {
    const response = await databaseGroupServiceClientConnect.listDatabaseGroups(
      createProto(ListDatabaseGroupsRequestSchema, { parent: project })
    );
    set((state) => {
      const next = { ...state.dbGroupsByName };
      for (const g of response.databaseGroups) next[g.name] = g;
      return { dbGroupsByName: next };
    });
    return response.databaseGroups;
  },
});
```

- [ ] **Step 5: Wire into `createAppStore`** (after `createDatabaseSlice`).
- [ ] **Step 6: Test → PASS. Type-check + fix.**
- [ ] **Step 7: Commit**

```
git commit -m "feat(react): add DBGroupSlice to app store"
```

## Task 1.3: Add `SheetSlice`

**Files:** mirrors Task 1.1.

- [ ] **Step 1: Write the failing test**

```ts
test("deduplicates sheet fetches and caches the result", async () => {
  const name = "projects/p1/sheets/s1";
  const sheet = createProto(SheetSchema, { name, content: new Uint8Array() });
  mocks.getSheet.mockResolvedValueOnce(sheet);
  const store = createAppStore();

  const [first, second] = await Promise.all([
    store.getState().fetchSheet(name),
    store.getState().fetchSheet(name),
  ]);

  expect(first).toEqual(sheet);
  expect(second).toEqual(sheet);
  expect(mocks.getSheet).toHaveBeenCalledTimes(1);
});
```

Add `getSheet` and `createSheet` to mocks; wire `sheetServiceClientConnect`; import `SheetSchema`.

- [ ] **Step 2: Run test → FAIL.**

- [ ] **Step 3: Add `SheetSlice` type**

```ts
import type { Sheet } from "@/types/proto-es/v1/sheet_service_pb";

export type SheetSlice = {
  sheetsByName: Record<string, Sheet>;
  sheetRequests: Record<string, Promise<Sheet | undefined>>;
  sheetErrorsByName: Record<string, Error | undefined>;
  fetchSheet: (name: string, raw?: boolean) => Promise<Sheet | undefined>;
  createSheet: (parent: string, sheet: Sheet) => Promise<Sheet>;
};
```

Append to `AppStoreState`.

- [ ] **Step 4: Implement `sheet.ts`**

```ts
import { create as createProto } from "@bufbuild/protobuf";
import { sheetServiceClientConnect } from "@/connect";
import { isValidSheetName } from "@/react/lib/resourceName";
import {
  CreateSheetRequestSchema,
  GetSheetRequestSchema,
} from "@/types/proto-es/v1/sheet_service_pb";
import type { AppSliceCreator, SheetSlice } from "./types";

function toError(e: unknown): Error {
  return e instanceof Error ? e : new Error(String(e));
}

export const createSheetSlice: AppSliceCreator<SheetSlice> = (set, get) => ({
  sheetsByName: {},
  sheetRequests: {},
  sheetErrorsByName: {},

  fetchSheet: async (name, raw = false) => {
    if (!isValidSheetName(name)) return undefined;
    const existing = get().sheetsByName[name];
    if (existing && !raw) return existing;
    const pending = get().sheetRequests[name];
    if (pending) return pending;

    const request = sheetServiceClientConnect
      .getSheet(createProto(GetSheetRequestSchema, { name, raw }))
      .then((sheet) => {
        set((state) => {
          const { [name]: _omit, ...rest } = state.sheetRequests;
          return {
            sheetsByName: { ...state.sheetsByName, [sheet.name]: sheet },
            sheetErrorsByName: { ...state.sheetErrorsByName, [name]: undefined },
            sheetRequests: rest,
          };
        });
        return sheet;
      })
      .catch((error) => {
        set((state) => {
          const { [name]: _omit, ...rest } = state.sheetRequests;
          return {
            sheetErrorsByName: { ...state.sheetErrorsByName, [name]: toError(error) },
            sheetRequests: rest,
          };
        });
        return undefined;
      });

    set((state) => ({ sheetRequests: { ...state.sheetRequests, [name]: request } }));
    return request;
  },

  createSheet: async (parent, sheet) => {
    const created = await sheetServiceClientConnect.createSheet(
      createProto(CreateSheetRequestSchema, { parent, sheet })
    );
    set((state) => ({
      sheetsByName: { ...state.sheetsByName, [created.name]: created },
    }));
    return created;
  },
});
```

- [ ] **Step 5: Wire into `createAppStore`.**
- [ ] **Step 6: Test → PASS. Type-check + fix.**
- [ ] **Step 7: Commit**

```
git commit -m "feat(react): add SheetSlice to app store"
```

## Task 1.4: Add `InstanceRoleSlice`

- [ ] **Step 1: Write the failing test**

```ts
test("deduplicates instance role fetches per instance", async () => {
  const instance = "instances/i1";
  const roles = [createProto(InstanceRoleSchema, { name: `${instance}/roles/admin` })];
  mocks.listInstanceRoles.mockResolvedValueOnce({ roles });
  const store = createAppStore();

  const [first, second] = await Promise.all([
    store.getState().fetchInstanceRoles(instance),
    store.getState().fetchInstanceRoles(instance),
  ]);

  expect(first).toEqual(roles);
  expect(second).toEqual(roles);
  expect(mocks.listInstanceRoles).toHaveBeenCalledTimes(1);
});
```

Add `listInstanceRoles` mock; wire `instanceRoleServiceClientConnect`; import `InstanceRoleSchema`.

- [ ] **Step 2: Run test → FAIL.**

- [ ] **Step 3: Add `InstanceRoleSlice` type**

```ts
import type { InstanceRole } from "@/types/proto-es/v1/instance_role_service_pb";

export type InstanceRoleSlice = {
  rolesByInstance: Record<string, InstanceRole[]>;
  roleRequests: Record<string, Promise<InstanceRole[]>>;
  fetchInstanceRoles: (instance: string) => Promise<InstanceRole[]>;
};
```

Append to `AppStoreState`.

- [ ] **Step 4: Implement `instanceRole.ts`**

```ts
import { create as createProto } from "@bufbuild/protobuf";
import { instanceRoleServiceClientConnect } from "@/connect";
import { ListInstanceRolesRequestSchema } from "@/types/proto-es/v1/instance_role_service_pb";
import type { AppSliceCreator, InstanceRoleSlice } from "./types";

export const createInstanceRoleSlice: AppSliceCreator<InstanceRoleSlice> = (set, get) => ({
  rolesByInstance: {},
  roleRequests: {},

  fetchInstanceRoles: async (instance) => {
    const cached = get().rolesByInstance[instance];
    if (cached) return cached;
    const pending = get().roleRequests[instance];
    if (pending) return pending;

    const request = instanceRoleServiceClientConnect
      .listInstanceRoles(createProto(ListInstanceRolesRequestSchema, { parent: instance }))
      .then((response) => {
        set((state) => {
          const { [instance]: _omit, ...rest } = state.roleRequests;
          return {
            rolesByInstance: { ...state.rolesByInstance, [instance]: response.roles },
            roleRequests: rest,
          };
        });
        return response.roles;
      })
      .catch(() => {
        set((state) => {
          const { [instance]: _omit, ...rest } = state.roleRequests;
          return { roleRequests: rest };
        });
        return [];
      });

    set((state) => ({ roleRequests: { ...state.roleRequests, [instance]: request } }));
    return request;
  },
});
```

- [ ] **Step 5: Wire into `createAppStore`.**
- [ ] **Step 6: Test → PASS. Type-check + fix.**
- [ ] **Step 7: Commit**

```
git commit -m "feat(react): add InstanceRoleSlice to app store"
```

## Task 1.5: Page-scoped Zustand store skeleton

**Files:**
- Create: `frontend/src/react/pages/project/plan-detail/shared/stores/types.ts`
- Create: `.../shared/stores/snapshotSlice.ts`
- Create: `.../shared/stores/phaseSlice.ts`
- Create: `.../shared/stores/editingSlice.ts`
- Create: `.../shared/stores/selectionSlice.ts`
- Create: `.../shared/stores/pollingSlice.ts`
- Create: `.../shared/stores/usePlanDetailStore.tsx`
- Test: `.../shared/stores/__tests__/usePlanDetailStore.test.tsx`

The store is built but not yet wired into the page — wiring happens in Phase 2.

- [ ] **Step 1: Write the failing test**

`shared/stores/__tests__/usePlanDetailStore.test.tsx`:

```tsx
import { act, renderHook } from "@testing-library/react";
import { describe, expect, test } from "vitest";
import type { ReactNode } from "react";
import {
  PlanDetailStoreProvider,
  usePlanDetailStore,
} from "../usePlanDetailStore";

const wrapper = ({ children }: { children: ReactNode }) => (
  <PlanDetailStoreProvider>{children}</PlanDetailStoreProvider>
);

describe("usePlanDetailStore", () => {
  test("phase slice toggles active phases", () => {
    const { result } = renderHook(
      () => ({
        active: usePlanDetailStore((s) => s.activePhases),
        toggle: usePlanDetailStore((s) => s.togglePhase),
      }),
      { wrapper }
    );

    expect(result.current.active.has("changes")).toBe(true);
    act(() => result.current.toggle("changes"));
    expect(result.current.active.has("changes")).toBe(false);
  });

  test("editing slice tracks scopes and isEditing", () => {
    const { result } = renderHook(
      () => ({
        scopes: usePlanDetailStore((s) => s.editingScopes),
        setEditing: usePlanDetailStore((s) => s.setEditing),
      }),
      { wrapper }
    );

    expect(Object.keys(result.current.scopes)).toHaveLength(0);
    act(() => result.current.setEditing("title", true));
    expect(result.current.scopes.title).toBe(true);
    act(() => result.current.setEditing("title", false));
    expect(result.current.scopes.title).toBeUndefined();
  });

  test("each provider mount creates an isolated store", () => {
    const { result: a } = renderHook(() => usePlanDetailStore((s) => s.togglePhase), {
      wrapper,
    });
    const { result: b } = renderHook(() => usePlanDetailStore((s) => s.togglePhase), {
      wrapper,
    });
    expect(a.current).not.toBe(b.current);
  });
});
```

- [ ] **Step 2: Run test → FAIL (module not found).**

```
pnpm --dir frontend test -- src/react/pages/project/plan-detail/shared/stores/__tests__/usePlanDetailStore.test.tsx
```

- [ ] **Step 3: Create `types.ts`**

```ts
import type { Plan, PlanCheckRun } from "@/types/proto-es/v1/plan_service_pb";
import type { Issue } from "@/types/proto-es/v1/issue_service_pb";
import type { Rollout, TaskRun } from "@/types/proto-es/v1/rollout_service_pb";
import type { StateCreator } from "zustand";

export type PlanDetailPhase = "changes" | "review" | "deploy";

export interface PlanDetailPageSnapshot {
  plan?: Plan;
  issue?: Issue;
  rollout?: Rollout;
  taskRuns: TaskRun[];
  planCheckRuns: PlanCheckRun[];
  isInitializing: boolean;
  isNotFound: boolean;
  isPermissionDenied: boolean;
}

export type SnapshotSlice = {
  snapshot: PlanDetailPageSnapshot;
  setSnapshot: (snapshot: PlanDetailPageSnapshot) => void;
  patchSnapshot: (patch: Partial<PlanDetailPageSnapshot>) => void;
};

export type PhaseSlice = {
  activePhases: Set<PlanDetailPhase>;
  togglePhase: (phase: PlanDetailPhase) => void;
  expandPhase: (phase: PlanDetailPhase) => void;
  collapsePhase: (phase: PlanDetailPhase) => void;
};

export type EditingSlice = {
  editingScopes: Record<string, true>;
  setEditing: (scope: string, editing: boolean) => void;
  bypassLeaveGuardOnce: () => void;
  isLeaveGuardBypassed: () => boolean;
  pendingLeaveTarget: string | null;
  setPendingLeaveTarget: (target: string | null) => void;
  pendingLeaveConfirm: boolean;
  setPendingLeaveConfirm: (open: boolean) => void;
};

export type SelectionSlice = {
  routePhase: PlanDetailPhase | undefined;
  selectedSpecId: string | undefined;
  selectedStageId: string | undefined;
  selectedTaskName: string | undefined;
  setRouteSelection: (selection: {
    phase?: PlanDetailPhase;
    specId?: string;
    stageId?: string;
    taskName?: string;
  }) => void;
};

export type PollingSlice = {
  isRefreshing: boolean;
  isRunningChecks: boolean;
  lastRefreshTime: number;
  pollTimerId: number | undefined;
  setRefreshing: (v: boolean) => void;
  setRunningChecks: (v: boolean) => void;
  setLastRefreshTime: (t: number) => void;
  setPollTimerId: (id: number | undefined) => void;
};

export type PlanDetailStore = SnapshotSlice & PhaseSlice & EditingSlice & SelectionSlice & PollingSlice;

export type PlanDetailSliceCreator<Slice> = StateCreator<PlanDetailStore, [], [], Slice>;
```

- [ ] **Step 4: Create `snapshotSlice.ts`**

```ts
import type { PlanDetailPageSnapshot, PlanDetailSliceCreator, SnapshotSlice } from "./types";

const buildDefaultSnapshot = (): PlanDetailPageSnapshot => ({
  taskRuns: [],
  planCheckRuns: [],
  isInitializing: true,
  isNotFound: false,
  isPermissionDenied: false,
});

export const createSnapshotSlice: PlanDetailSliceCreator<SnapshotSlice> = (set) => ({
  snapshot: buildDefaultSnapshot(),
  setSnapshot: (snapshot) => set({ snapshot }),
  patchSnapshot: (patch) =>
    set((state) => ({ snapshot: { ...state.snapshot, ...patch } })),
});
```

- [ ] **Step 5: Create `phaseSlice.ts`**

```ts
import type { PhaseSlice, PlanDetailPhase, PlanDetailSliceCreator } from "./types";

const defaultActivePhases = (): Set<PlanDetailPhase> => new Set(["changes"]);

export const createPhaseSlice: PlanDetailSliceCreator<PhaseSlice> = (set) => ({
  activePhases: defaultActivePhases(),
  togglePhase: (phase) =>
    set((state) => {
      const next = new Set(state.activePhases);
      if (next.has(phase)) next.delete(phase);
      else next.add(phase);
      return { activePhases: next };
    }),
  expandPhase: (phase) =>
    set((state) => {
      if (state.activePhases.has(phase)) return state;
      const next = new Set(state.activePhases);
      next.add(phase);
      return { activePhases: next };
    }),
  collapsePhase: (phase) =>
    set((state) => {
      if (!state.activePhases.has(phase)) return state;
      const next = new Set(state.activePhases);
      next.delete(phase);
      return { activePhases: next };
    }),
});
```

- [ ] **Step 6: Create `editingSlice.ts`**

```ts
import type { EditingSlice, PlanDetailSliceCreator } from "./types";

export const createEditingSlice: PlanDetailSliceCreator<EditingSlice> = (set) => {
  let bypassOnce = false;
  return {
    editingScopes: {},
    setEditing: (scope, editing) =>
      set((state) => {
        const next = { ...state.editingScopes };
        if (editing) next[scope] = true;
        else delete next[scope];
        return { editingScopes: next };
      }),
    bypassLeaveGuardOnce: () => {
      bypassOnce = true;
    },
    isLeaveGuardBypassed: () => {
      if (bypassOnce) {
        bypassOnce = false;
        return true;
      }
      return false;
    },
    pendingLeaveTarget: null,
    setPendingLeaveTarget: (target) => set({ pendingLeaveTarget: target }),
    pendingLeaveConfirm: false,
    setPendingLeaveConfirm: (open) => set({ pendingLeaveConfirm: open }),
  };
};
```

- [ ] **Step 7: Create `selectionSlice.ts`**

```ts
import type { PlanDetailSliceCreator, SelectionSlice } from "./types";

export const createSelectionSlice: PlanDetailSliceCreator<SelectionSlice> = (set) => ({
  routePhase: undefined,
  selectedSpecId: undefined,
  selectedStageId: undefined,
  selectedTaskName: undefined,
  setRouteSelection: (selection) =>
    set({
      routePhase: selection.phase,
      selectedSpecId: selection.specId,
      selectedStageId: selection.stageId,
      selectedTaskName: selection.taskName,
    }),
});
```

- [ ] **Step 8: Create `pollingSlice.ts`**

```ts
import type { PlanDetailSliceCreator, PollingSlice } from "./types";

export const createPollingSlice: PlanDetailSliceCreator<PollingSlice> = (set) => ({
  isRefreshing: false,
  isRunningChecks: false,
  lastRefreshTime: 0,
  pollTimerId: undefined,
  setRefreshing: (v) => set({ isRefreshing: v }),
  setRunningChecks: (v) => set({ isRunningChecks: v }),
  setLastRefreshTime: (t) => set({ lastRefreshTime: t }),
  setPollTimerId: (id) => set({ pollTimerId: id }),
});
```

- [ ] **Step 9: Create `usePlanDetailStore.tsx`**

```tsx
import { createContext, useContext, useState, type ReactNode } from "react";
import { create, useStore } from "zustand";
import { createEditingSlice } from "./editingSlice";
import { createPhaseSlice } from "./phaseSlice";
import { createPollingSlice } from "./pollingSlice";
import { createSelectionSlice } from "./selectionSlice";
import { createSnapshotSlice } from "./snapshotSlice";
import type { PlanDetailStore } from "./types";

const createPlanDetailStore = () =>
  create<PlanDetailStore>()((...args) => ({
    ...createSnapshotSlice(...args),
    ...createPhaseSlice(...args),
    ...createEditingSlice(...args),
    ...createSelectionSlice(...args),
    ...createPollingSlice(...args),
  }));

type PlanDetailStoreApi = ReturnType<typeof createPlanDetailStore>;

const PlanDetailStoreContext = createContext<PlanDetailStoreApi | null>(null);

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

export function usePlanDetailStoreApi(): PlanDetailStoreApi {
  const store = useContext(PlanDetailStoreContext);
  if (!store) throw new Error("PlanDetailStoreProvider missing");
  return store;
}
```

- [ ] **Step 10: Run test → PASS. Type-check + fix.**

- [ ] **Step 11: Commit**

```
git commit -m "feat(react): page-scoped plan-detail Zustand store skeleton"
```

## Task 1.6: Extract `shell/leaveGuard.ts` as pure functions

**Files:**
- Create: `frontend/src/react/pages/project/plan-detail/shell/leaveGuard.ts`
- Test: `frontend/src/react/pages/project/plan-detail/shell/__tests__/leaveGuard.test.ts`

- [ ] **Step 1: Write the failing test**

```ts
import { describe, expect, test } from "vitest";
import { decideLeaveAction } from "../leaveGuard";

describe("decideLeaveAction", () => {
  test("allows navigation when no scopes are editing", () => {
    expect(
      decideLeaveAction({ editingScopes: {}, isBypassed: false, targetPath: "/x" })
    ).toEqual({ action: "allow" });
  });

  test("allows navigation when bypass flag is set", () => {
    expect(
      decideLeaveAction({ editingScopes: { title: true }, isBypassed: true, targetPath: "/x" })
    ).toEqual({ action: "allow" });
  });

  test("intercepts when an edit scope is open", () => {
    expect(
      decideLeaveAction({ editingScopes: { title: true }, isBypassed: false, targetPath: "/x" })
    ).toEqual({ action: "intercept", pendingTarget: "/x" });
  });

  test("intercepts when multiple scopes are open", () => {
    expect(
      decideLeaveAction({
        editingScopes: { title: true, description: true },
        isBypassed: false,
        targetPath: "/y",
      })
    ).toEqual({ action: "intercept", pendingTarget: "/y" });
  });
});
```

- [ ] **Step 2: Run test → FAIL.**

- [ ] **Step 3: Implement `leaveGuard.ts`**

```ts
export type LeaveAction =
  | { action: "allow" }
  | { action: "intercept"; pendingTarget: string };

export function decideLeaveAction(input: {
  editingScopes: Record<string, true>;
  isBypassed: boolean;
  targetPath: string;
}): LeaveAction {
  if (input.isBypassed) return { action: "allow" };
  const hasEditing = Object.keys(input.editingScopes).length > 0;
  if (!hasEditing) return { action: "allow" };
  return { action: "intercept", pendingTarget: input.targetPath };
}
```

- [ ] **Step 4: Run test → PASS. Type-check + fix. Commit.**

```
git commit -m "feat(react): extract leave-guard decision as pure function"
```

## Task 1.7: CI guard against Pinia reads inside plan-detail/ (strict)

**Files:** the existing cross-framework CI guard.

- [ ] **Step 1: Locate the existing guard**

```
git log --oneline | grep -iE "cross-framework|cross framework" | head -5
git show <commit> --stat
```

Identify the script (likely `frontend/scripts/check-react-imports.*` or similar).

- [ ] **Step 2: Add the strict rule**

Append to the existing guard (or create a sibling check) a rule that fails CI when `useVueState` inside `plan-detail/` references a Pinia store hook (`use[A-Z][A-Za-z]+Store`), unless the read is for the Vue router (`useRouter()` / `useRoute()`).

Minimal grep-based form:

```bash
PLAN_DETAIL_DIR="frontend/src/react/pages/project/plan-detail"
VIOLATIONS=$(
  grep -rnE "useVueState\(\s*\(\)\s*=>\s*use[A-Z][A-Za-z]+Store" \
    "$PLAN_DETAIL_DIR" \
    --include="*.ts" --include="*.tsx" \
    | grep -v "useRouter\|useRoute" || true
)
if [ -n "$VIOLATIONS" ]; then
  echo "Pinia reads via useVueState are forbidden in plan-detail/. Use Zustand slices instead:"
  echo "$VIOLATIONS"
  exit 1
fi
```

If the existing guard is a Node script, port the same logic to JS. No warning-mode flag.

- [ ] **Step 3: Verify locally**

Running the guard NOW (before later phases land) will report violations — they exist throughout the existing code. **Do not commit the guard yet.** Leave the file edited locally; we will commit it only after Phase 5 makes the violation count zero.

Track the edit:

```
git stash -u -- frontend/<guard-script-path>
```

We will pop this stash near the end of the work (Task 6.4).

- [ ] **Step 4: No commit yet for the guard.**

## Task 1.8: E2E smoke test for plan-detail (hard gate)

**Files:**
- Create: `frontend/e2e/plan-detail-smoke.spec.ts` (location depends on existing e2e setup)

- [ ] **Step 1: Find the existing e2e setup**

```
find frontend -name "playwright.config*" -o -name "*.spec.ts" -path "*e2e*" | head -10
```

If nothing comes back, **stop and ask the user before continuing** — adding e2e infrastructure is outside this refactor's scope. The user requested the e2e smoke test as a hard gate, so we cannot proceed without it.

- [ ] **Step 2: Add the smoke spec**

`plan-detail-smoke.spec.ts` exercises:
- open `/projects/<seeded>/plans/<seeded>` for a seeded plan in the e2e environment
- assert page title renders
- assert Changes phase is expanded
- assert at least one spec tab is visible

Keep it minimal — the full golden-path journey remains a manual gate per spec Section 6.

- [ ] **Step 3: Run the spec locally; iterate until it passes.**

- [ ] **Step 4: Commit**

```
git add frontend/e2e/plan-detail-smoke.spec.ts
git commit -m "test(e2e): plan detail page renders smoke test"
```

---

# Phase 2 — Shell + page-hook restructure

**Goal:** Break `usePlanDetailPage` (741 lines) into composed phase-scoped hooks; route data through the page-scoped Zustand store. Move `ProjectPlanDetailPage` into `plan-detail/`. **Behavior is byte-identical.**

**Critical context:** The current `usePlanDetailPage.ts` is the entire page's spine. Read it end-to-end before starting Phase 2. The composition root we produce must call the extracted hooks in the same effect order so React's effect-flush semantics are preserved.

## Task 2.1: Move `ProjectPlanDetailPage.tsx` into `plan-detail/`

**Files:**
- Move: `frontend/src/react/pages/project/ProjectPlanDetailPage.tsx` → `frontend/src/react/pages/project/plan-detail/ProjectPlanDetailPage.tsx`
- Create: `frontend/src/react/pages/project/plan-detail/index.ts`
- Modify: every import site of `ProjectPlanDetailPage`

- [ ] **Step 1: Find all import sites**

```
grep -rn "ProjectPlanDetailPage" frontend/src --include="*.ts" --include="*.tsx" --include="*.vue"
```

- [ ] **Step 2: Move the file**

```
git mv frontend/src/react/pages/project/ProjectPlanDetailPage.tsx \
  frontend/src/react/pages/project/plan-detail/ProjectPlanDetailPage.tsx
```

- [ ] **Step 3: Create barrel `index.ts`**

```ts
export { ProjectPlanDetailPage } from "./ProjectPlanDetailPage";
```

- [ ] **Step 4: Update internal imports inside the moved file**

Paths like `./plan-detail/<x>` become `./<x>`; paths like `../<x>` become `../../<x>`.

- [ ] **Step 5: Update all external import sites**

Each call site changes from `@/react/pages/project/ProjectPlanDetailPage` to `@/react/pages/project/plan-detail`.

- [ ] **Step 6: Type-check + fix + test + manual smoke. Commit.**

```
git commit -m "refactor(plan-detail): move ProjectPlanDetailPage into plan-detail/"
```

## Task 2.2: Wrap page with `PlanDetailStoreProvider`

**File:** `frontend/src/react/pages/project/plan-detail/ProjectPlanDetailPage.tsx`

- [ ] **Step 1: Wrap the JSX**

```tsx
import { PlanDetailStoreProvider } from "./shared/stores/usePlanDetailStore";
// ...
return (
  <PlanDetailStoreProvider>
    {/* existing content */}
  </PlanDetailStoreProvider>
);
```

NotFound / PermissionDenied early returns do not need the provider.

- [ ] **Step 2: Type-check + test + manual smoke. Commit.**

```
git commit -m "refactor(plan-detail): wrap page in PlanDetailStoreProvider"
```

## Task 2.3: Extract `usePhaseState` hook

**Files:**
- Create: `shell/hooks/usePhaseState.ts`
- Modify: `hooks/usePlanDetailPage.ts`

- [ ] **Step 1: Write the hook**

```ts
import { useCallback } from "react";
import { usePlanDetailStore } from "../../shared/stores/usePlanDetailStore";
import type { PlanDetailPhase } from "../../shared/stores/types";

export function usePhaseState() {
  const activePhases = usePlanDetailStore((s) => s.activePhases);
  const togglePhase = usePlanDetailStore((s) => s.togglePhase);
  const expandPhase = usePlanDetailStore((s) => s.expandPhase);
  const collapsePhase = usePlanDetailStore((s) => s.collapsePhase);

  const isActive = useCallback(
    (phase: PlanDetailPhase) => activePhases.has(phase),
    [activePhases]
  );

  return { activePhases, isActive, togglePhase, expandPhase, collapsePhase };
}
```

- [ ] **Step 2: Replace usage in `usePlanDetailPage`** — delete local `activePhases` `useState` + `togglePhase` + `expandPhase`; call `usePhaseState()` instead.

- [ ] **Step 3: Type-check, test, manual smoke. Commit.**

```
git commit -m "refactor(plan-detail): extract usePhaseState into shell/hooks"
```

## Task 2.4: Extract `useEditingScopes` hook

**Files:**
- Create: `shell/hooks/useEditingScopes.ts`
- Modify: `hooks/usePlanDetailPage.ts`

- [ ] **Step 1: Write the hook**

```ts
import { useCallback, useMemo } from "react";
import { usePlanDetailStore } from "../../shared/stores/usePlanDetailStore";

export function useEditingScopes() {
  const editingScopes = usePlanDetailStore((s) => s.editingScopes);
  const setEditing = usePlanDetailStore((s) => s.setEditing);
  const bypassLeaveGuardOnce = usePlanDetailStore((s) => s.bypassLeaveGuardOnce);
  const pendingLeaveConfirm = usePlanDetailStore((s) => s.pendingLeaveConfirm);
  const setPendingLeaveConfirm = usePlanDetailStore((s) => s.setPendingLeaveConfirm);

  const isEditing = useMemo(
    () => Object.keys(editingScopes).length > 0,
    [editingScopes]
  );

  const setScopeEditing = useCallback(
    (scope: string, editing: boolean) => setEditing(scope, editing),
    [setEditing]
  );

  return {
    editingScopes,
    isEditing,
    setEditing: setScopeEditing,
    bypassLeaveGuardOnce,
    pendingLeaveConfirm,
    setPendingLeaveConfirm,
  };
}
```

- [ ] **Step 2: Replace usage in `usePlanDetailPage`** — delete `editingScopes` state + `setEditing` callback + `isEditing` derivation + `bypassLeaveGuardOnce` callback + `pendingLeaveConfirm` state. Call `useEditingScopes()`.

- [ ] **Step 3: Type-check, test, manual smoke. Commit.**

```
git commit -m "refactor(plan-detail): extract useEditingScopes into shell/hooks"
```

## Task 2.5: Extract `useSidebarMode` hook + `shell/constants.ts`

**Files:**
- Create: `shell/hooks/useSidebarMode.ts`
- Create: `shell/constants.ts`
- Modify: `hooks/usePlanDetailPage.ts`

- [ ] **Step 1: Create `shell/constants.ts`** with the breakpoint and poller constants currently at the top of `usePlanDetailPage.ts` (lines 61-77):

```ts
export const POLLER_INTERVAL = {
  // copy the exact values currently in usePlanDetailPage lines 61-67
} as const;

export const PROJECT_NAME_PREFIX = "projects/";

export const MOBILE_BREAKPOINT_PX = 780;
export const WIDE_SIDEBAR_BREAKPOINT_PX = 1280;
export const SIDEBAR_WIDTH_NARROW_PX = 240;
export const SIDEBAR_WIDTH_WIDE_PX = 336;
```

- [ ] **Step 2: Create `useSidebarMode.ts`** that owns the container-width effect:

```ts
import { useEffect, useState } from "react";
import {
  MOBILE_BREAKPOINT_PX,
  SIDEBAR_WIDTH_NARROW_PX,
  SIDEBAR_WIDTH_WIDE_PX,
  WIDE_SIDEBAR_BREAKPOINT_PX,
} from "../constants";

export type PlanDetailSidebarMode = "NONE" | "DESKTOP" | "MOBILE";

export function useSidebarMode(containerRef: React.RefObject<HTMLElement>) {
  const [sidebarMode, setSidebarMode] = useState<PlanDetailSidebarMode>("NONE");
  const [isMobileSidebarOpen, setMobileSidebarOpen] = useState(false);
  const [sidebarWidth, setSidebarWidth] = useState<number>(SIDEBAR_WIDTH_NARROW_PX);

  useEffect(() => {
    // Move the existing ResizeObserver / width-tracking effect from usePlanDetailPage here verbatim,
    // calling setSidebarMode / setSidebarWidth based on container width vs the imported breakpoints.
  }, [containerRef]);

  return { sidebarMode, sidebarWidth, isMobileSidebarOpen, setMobileSidebarOpen };
}
```

- [ ] **Step 3: Replace usage in `usePlanDetailPage`.** Update internal callers (`PlanDetailLayout` etc.) to import the constants from `shell/constants` instead of from `hooks/usePlanDetailPage`.

- [ ] **Step 4: Type-check, test, manual smoke (resize window; sidebar mode toggles). Commit.**

```
git commit -m "refactor(plan-detail): extract useSidebarMode and shell/constants"
```

## Task 2.6: Extract `useRouteSelection` hook

**Files:**
- Create: `shell/hooks/useRouteSelection.ts`
- Modify: `hooks/usePlanDetailPage.ts`

- [ ] **Step 1: Write the hook**

```ts
import { useEffect } from "react";
import { usePlanDetailStore } from "../../shared/stores/usePlanDetailStore";
import type { PlanDetailPhase } from "../../shared/stores/types";

function getRouteQueryString(value: unknown): string | undefined {
  if (typeof value === "string") return value || undefined;
  if (Array.isArray(value)) return getRouteQueryString(value[0]);
  return undefined;
}

export function useRouteSelection(params: {
  routeQuery: Record<string, unknown>;
  specId?: string;
}) {
  const setRouteSelection = usePlanDetailStore((s) => s.setRouteSelection);

  const phase = getRouteQueryString(params.routeQuery.phase) as PlanDetailPhase | undefined;
  const stageId = getRouteQueryString(params.routeQuery.stageId);
  const taskId = getRouteQueryString(params.routeQuery.taskId);

  useEffect(() => {
    setRouteSelection({ phase, stageId, taskName: taskId, specId: params.specId });
  }, [setRouteSelection, phase, stageId, taskId, params.specId]);

  return { phase, stageId, taskId };
}
```

- [ ] **Step 2: Replace usage in `usePlanDetailPage`** — remove `convertRouteQuery`, the route-query refs, and the derived `routePhase/Stage/Task`. Call `useRouteSelection({ routeQuery, specId })`.

- [ ] **Step 3: Type-check, test, manual smoke (navigate with `?phase=deploy&taskId=...`). Commit.**

```
git commit -m "refactor(plan-detail): extract useRouteSelection into shell/hooks"
```

## Task 2.7: Extract `fetchPlanSnapshot` + `useInitialFetch` + `usePolling`

**Files:**
- Create: `shell/hooks/fetchPlanSnapshot.ts`
- Create: `shell/hooks/useInitialFetch.ts`
- Create: `shell/hooks/usePolling.ts`
- Modify: `hooks/usePlanDetailPage.ts`

The snapshot fetch + polling lifecycle lives across `usePlanDetailPage.ts` lines 188-376. Riskiest extraction in Phase 2 — effect ordering matters.

- [ ] **Step 1: Re-read source lines 188-376** of `hooks/usePlanDetailPage.ts` end-to-end. Note `fetchPlanDetailSnapshot` (188-285) is a pure async function; the init-fetch and polling effects are coupled by `latestSnapshotRef`.

- [ ] **Step 2: Move `fetchPlanDetailSnapshot` verbatim into `fetchPlanSnapshot.ts`** — rename only if needed. Inside, replace Pinia reads with new app-store calls:
  - `useDatabaseV1Store.getDatabaseByName(...)` → `useAppStore.getState().fetchDatabase(...)`
  - `useDBGroupStore.getDBGroupByName(...)` → `useAppStore.getState().fetchDBGroup(...)`
  - `useSheetV1Store.getSheetByName(...)` → `useAppStore.getState().fetchSheet(...)`
  - `useProjectV1Store.getProjectByName(...)` → `useAppStore.getState().fetchProject(...)`
  - `useEnvironmentV1Store` reads → existing `WorkspaceSlice.environmentList`

List every replacement in the commit message.

- [ ] **Step 3: Write `useInitialFetch.ts`**

```ts
import { useEffect, useRef } from "react";
import { usePlanDetailStore } from "../../shared/stores/usePlanDetailStore";
import { fetchPlanSnapshot } from "./fetchPlanSnapshot";

export function useInitialFetch(params: {
  projectId: string;
  planId: string;
  routeQuery: Record<string, unknown>;
}) {
  const setSnapshot = usePlanDetailStore((s) => s.setSnapshot);
  const setLastRefreshTime = usePlanDetailStore((s) => s.setLastRefreshTime);
  const didRunRef = useRef(false);

  useEffect(() => {
    if (didRunRef.current) return;
    didRunRef.current = true;
    let cancelled = false;
    fetchPlanSnapshot(params).then((snapshot) => {
      if (cancelled) return;
      setSnapshot(snapshot);
      setLastRefreshTime(Date.now());
    });
    return () => {
      cancelled = true;
    };
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [params.projectId, params.planId]);
}
```

- [ ] **Step 4: Write `usePolling.ts`**

Port the predicates `snapshotIsEqual` and `allTasksDoneOrSkipped` from the body of the current `usePlanDetailPage.ts` verbatim — they exist today and define the polling stop condition.

```ts
import { useEffect } from "react";
import { POLLER_INTERVAL } from "../constants";
import {
  usePlanDetailStore,
  usePlanDetailStoreApi,
} from "../../shared/stores/usePlanDetailStore";
import type { PlanDetailPageSnapshot } from "../../shared/stores/types";
import { fetchPlanSnapshot } from "./fetchPlanSnapshot";

function snapshotIsEqual(a: PlanDetailPageSnapshot, b: PlanDetailPageSnapshot): boolean {
  // Lift the existing equality check from the current usePlanDetailPage polling
  // effect (it compared plan/issue/rollout/taskRuns identity). Reproduce
  // exactly the comparison used today.
  return false; // placeholder — replace with ported body
}

function allTasksDoneOrSkipped(s: PlanDetailPageSnapshot): boolean {
  // Port the existing "every task done or skipped" predicate. Today it iterates
  // s.taskRuns and s.rollout?.stages and returns true when no task is in a
  // pending/running state.
  return false; // placeholder — replace with ported body
}

export function usePolling(params: {
  projectId: string;
  planId: string;
  routeQuery: Record<string, unknown>;
  enabled: boolean;
}) {
  const storeApi = usePlanDetailStoreApi();
  const setSnapshot = usePlanDetailStore((s) => s.setSnapshot);
  const setLastRefreshTime = usePlanDetailStore((s) => s.setLastRefreshTime);

  useEffect(() => {
    if (!params.enabled) return;

    let cancelled = false;
    let timerId: number | undefined;

    const tick = async () => {
      if (cancelled) return;
      const snapshot = await fetchPlanSnapshot(params);
      if (cancelled) return;
      const current = storeApi.getState().snapshot;
      if (snapshotIsEqual(current, snapshot)) {
        timerId = window.setTimeout(tick, POLLER_INTERVAL.default);
        return;
      }
      setSnapshot(snapshot);
      setLastRefreshTime(Date.now());
      if (allTasksDoneOrSkipped(snapshot)) return;
      timerId = window.setTimeout(tick, POLLER_INTERVAL.default);
    };

    timerId = window.setTimeout(tick, POLLER_INTERVAL.default);

    return () => {
      cancelled = true;
      if (timerId !== undefined) window.clearTimeout(timerId);
    };
  }, [params.enabled, params.projectId, params.planId, storeApi, setSnapshot, setLastRefreshTime]);
}
```

When porting the two predicates, copy the exact comparison and iteration the current `usePlanDetailPage` uses — do not redesign them.

- [ ] **Step 5: Replace usage in `usePlanDetailPage`** — delete the init + polling effects. Call `useInitialFetch(...)` and `usePolling({ ..., enabled: !isCreateMode })`.

- [ ] **Step 6: Type-check, test, manual smoke**: open a plan with a running rollout (page refetches); open a plan with all tasks done (polling stops).

- [ ] **Step 7: Commit**

```
git commit -m "refactor(plan-detail): extract fetchPlanSnapshot/useInitialFetch/usePolling + slice migration"
```

## Task 2.8: Extract `useRedirects`

**Files:** create `shell/hooks/useRedirects.ts`. Move the redirect effect from `usePlanDetailPage.ts` (NotFound / PermissionDenied / linked-issue → issue-detail, around lines 396-485) into the hook. Call it from the page hook.

- [ ] Type-check, test, commit.

```
git commit -m "refactor(plan-detail): extract useRedirects into shell/hooks"
```

## Task 2.9: Extract `useLeaveGuard`

**Files:** create `shell/hooks/useLeaveGuard.ts` wired to `shell/leaveGuard.ts` (from Task 1.6).

- [ ] **Step 1: Write the hook**

```ts
import { useEffect } from "react";
import { router } from "@/router";
import {
  usePlanDetailStore,
  usePlanDetailStoreApi,
} from "../../shared/stores/usePlanDetailStore";
import { decideLeaveAction } from "../leaveGuard";

export function useLeaveGuard() {
  const storeApi = usePlanDetailStoreApi();
  const setPendingLeaveTarget = usePlanDetailStore((s) => s.setPendingLeaveTarget);
  const setPendingLeaveConfirm = usePlanDetailStore((s) => s.setPendingLeaveConfirm);

  useEffect(() => {
    const off = router.beforeEach((to, _from, next) => {
      const state = storeApi.getState();
      const decision = decideLeaveAction({
        editingScopes: state.editingScopes,
        isBypassed: state.isLeaveGuardBypassed(),
        targetPath: to.fullPath,
      });

      if (decision.action === "allow") {
        next();
        return;
      }
      setPendingLeaveTarget(decision.pendingTarget);
      setPendingLeaveConfirm(true);
      next(false);
    });
    return () => off();
  }, [storeApi, setPendingLeaveTarget, setPendingLeaveConfirm]);
}
```

- [ ] **Step 2: Replace usage in `usePlanDetailPage`** — delete the `beforeEach` registration and the ref-based dance.

- [ ] **Step 3: Manual smoke**: open title editor, attempt navigation → confirm dialog appears; confirm → navigation proceeds. Commit.

```
git commit -m "refactor(plan-detail): extract useLeaveGuard wired to pure decideLeaveAction"
```

## Task 2.10: Extract `useDerivedPlanState` selectors

**Files:**
- Create: `shell/hooks/useDerivedPlanState.ts`
- Create: `shell/hooks/types.ts` (move `PlanDetailPageState` shape here)

- [ ] **Step 1: Write the hook** — move the final big `useMemo` of `usePlanDetailPage.ts` (around line 692+) into the hook. Read every store field the returned shape needs via `usePlanDetailStore` selectors.

```ts
import { useMemo } from "react";
import { usePlanDetailStore } from "../../shared/stores/usePlanDetailStore";
import type { PlanDetailPageState } from "./types";

export function useDerivedPlanState(): PlanDetailPageState {
  const snapshot = usePlanDetailStore((s) => s.snapshot);
  const activePhases = usePlanDetailStore((s) => s.activePhases);
  // ... read every store field the returned shape needs

  return useMemo(() => ({
    // derive readonly, ready, isPlanDone, selectedTaskName, sidebarStatus, etc.
    // EXACTLY as the current final useMemo of usePlanDetailPage does
  }), [snapshot, activePhases /* ... */]);
}
```

- [ ] **Step 2: Replace the final `useMemo` in `usePlanDetailPage` with `return useDerivedPlanState();`.**

- [ ] **Step 3: Type-check, test, commit.**

```
git commit -m "refactor(plan-detail): extract useDerivedPlanState selectors"
```

## Task 2.11: Reduce `usePlanDetailPage` to composition root and move

**Files:**
- Move: `frontend/src/react/pages/project/plan-detail/hooks/usePlanDetailPage.ts` → `frontend/src/react/pages/project/plan-detail/shell/hooks/usePlanDetailPage.ts`
- Update every import site of `usePlanDetailPage`, `MOBILE_BREAKPOINT_PX`, `WIDE_SIDEBAR_BREAKPOINT_PX`, `SIDEBAR_WIDTH_NARROW_PX`, `SIDEBAR_WIDTH_WIDE_PX`, `PlanDetailSidebarMode`, `PlanDetailPhase`, `PlanDetailPageSnapshot`, `PlanDetailPageState`

- [ ] **Step 1: Verify size**

```
wc -l frontend/src/react/pages/project/plan-detail/hooks/usePlanDetailPage.ts
```

Expected: under 120 lines. The body should be a thin composition:

```ts
export function usePlanDetailPage(params: PageParams): PlanDetailPageState {
  useInitialFetch(params);
  usePolling({ ...params, enabled: !isCreateMode(params) });
  useRouteSelection(params);
  useRedirects(params);
  useLeaveGuard();
  return useDerivedPlanState();
}
```

If still bloated, extract another hook.

- [ ] **Step 2: Move the file**

```
git mv frontend/src/react/pages/project/plan-detail/hooks/usePlanDetailPage.ts \
  frontend/src/react/pages/project/plan-detail/shell/hooks/usePlanDetailPage.ts
```

Fix imports inside.

- [ ] **Step 3: Update every external import site**

```
grep -rn "from \"@/react/pages/project/plan-detail/hooks/usePlanDetailPage\"" frontend/src
```

Repoint each match to `@/react/pages/project/plan-detail/shell/hooks/usePlanDetailPage`.

For exports that lived at the old path but now logically live in `shell/constants.ts` or `shared/stores/types.ts`, repoint those imports to their new canonical homes. The hook file should only re-export from itself.

- [ ] **Step 4: Delete the old `hooks/` folder** if empty (`usePlanCheckActions.tsx` and `usePlanDetailSpecValidation.ts` move in Phase 3 — leave the folder until then if they're still there).

- [ ] **Step 5: Type-check, test, manual smoke. Commit.**

```
git commit -m "refactor(plan-detail): usePlanDetailPage as composition root in shell/hooks"
```

## Task 2.12: Move `PlanDetailContext`

**Files:**
- Move: `context/PlanDetailContext.tsx` → `shell/PlanDetailContext.tsx`
- Update every importer

- [ ] **Step 1: Find importers**

```
grep -rn "from \"@/react/pages/project/plan-detail/context/PlanDetailContext\"" frontend/src
grep -rn "from \"../../context/PlanDetailContext\"" frontend/src/react/pages/project/plan-detail
```

- [ ] **Step 2: Move the file**

```
git mv frontend/src/react/pages/project/plan-detail/context/PlanDetailContext.tsx \
  frontend/src/react/pages/project/plan-detail/shell/PlanDetailContext.tsx
```

- [ ] **Step 3: Update every import to the new path.** Delete the now-empty `context/` folder if applicable.

- [ ] **Step 4: Type-check, test, manual smoke. Commit.**

```
git commit -m "refactor(plan-detail): move PlanDetailContext into shell/"
```

---

# Phase 3 — Changes phase

**Goal:** Split `PlanDetailChangesBranch.tsx` (1969) and `PlanDetailStatementSection.tsx` (748) into ~17 focused files under `changes/`. Switch the 10 Pinia reads to the new app-store slices. Apply `/simplify`.

**Reference source line ranges** (from `git show HEAD:frontend/src/react/pages/project/plan-detail/components/PlanDetailChangesBranch.tsx` at Phase 3 start — verify before each extraction):
- `pushSpecDetailRoute`: 143
- `IsolationLevel` type: 153-167
- `PlanDetailChangesBranch` main: 168-651
- `OptionsSection`: 652-1121
- `TargetsSection`: 1122-1369
- `TargetSelectorSheet`: 1370-1459
- `DatabaseAndGroupSelector`: 1460-1530
- `DatabaseSelector`: 1531-1706
- `DatabaseGroupSelector`: 1707-1790
- `DatabaseTarget`: 1791-1822
- `DatabaseGroupTarget`: 1823-1916
- `parseStatement`: 1917-1969

## Task 3.1: Extract `parseStatement` with new tests

**Files:**
- Create: `changes/StatementSection/parseStatement.ts`
- Create: `changes/StatementSection/__tests__/parseStatement.test.ts`
- Modify: `components/PlanDetailChangesBranch.tsx`

Pure function; no existing tests — start with TDD.

- [ ] **Step 1: Write the failing test**

Fill these stubs by reading lines 1917-1969 of the source and writing the expected output for each input case:

```ts
import { describe, expect, test } from "vitest";
import { parseStatement } from "../parseStatement";

describe("parseStatement", () => {
  test("returns empty result for empty input", () => {
    expect(parseStatement("")).toEqual(/* shape per current impl */);
  });

  test("handles a single statement without trailing semicolon", () => { /* ... */ });
  test("splits multiple statements by semicolon", () => { /* ... */ });
  test("preserves semicolons inside string literals", () => { /* ... */ });
  test("strips line comments", () => { /* ... */ });
});
```

- [ ] **Step 2: Run test → FAIL.**

- [ ] **Step 3: Create `parseStatement.ts`** — function body lifted verbatim from lines 1917-1969 of the source.

- [ ] **Step 4: Run test → PASS.**

- [ ] **Step 5: Replace the inline `parseStatement` in `PlanDetailChangesBranch.tsx` with an import. Delete the original lines 1917-1969.**

- [ ] **Step 6: Type-check + existing `PlanDetailChangesBranch.test.tsx`. Commit.**

```
git commit -m "refactor(plan-detail): extract changes/parseStatement with tests"
```

## Task 3.2: Extract `DatabaseTarget` and `DatabaseGroupTarget`

**Files:**
- Create: `changes/TargetsSection/DatabaseTarget.tsx`
- Create: `changes/TargetsSection/DatabaseGroupTarget.tsx`
- Modify: `components/PlanDetailChangesBranch.tsx`
- Update: every external importer of these names

Both are exported today (`export function DatabaseTarget`, `export function DatabaseGroupTarget`).

- [ ] **Step 1: Read the current bodies**

```
sed -n '1791,1916p' frontend/src/react/pages/project/plan-detail/components/PlanDetailChangesBranch.tsx
```

- [ ] **Step 2: Create the two files** with the lifted function + required imports.

- [ ] **Step 3: Remove the inline definitions from `PlanDetailChangesBranch.tsx`.** Do not add re-exports at the old path.

- [ ] **Step 4: Update every external importer**

```
grep -rn "DatabaseTarget\|DatabaseGroupTarget" frontend/src --include="*.ts" --include="*.tsx"
```

Repoint each match to the new path.

- [ ] **Step 5: `/simplify` checklist on the new files.** Type-check + tests + commit.

```
git commit -m "refactor(plan-detail): extract DatabaseTarget and DatabaseGroupTarget"
```

## Task 3.3: Extract `DatabaseGroupSelector` (with slice migration)

**Files:** Create `changes/TargetsSection/DatabaseGroupSelector.tsx`.

- [ ] **Step 1: Lift function body** (source lines 1707-1790) into the new file.

- [ ] **Step 2: Replace Pinia reads with app-store calls.**

Before:
```ts
const dbGroupStore = useVueState(() => useDBGroupStore());
const groups = dbGroupStore.dbGroupsByProject(projectName);
```

After:
```ts
import { useAppStore } from "@/react/stores/app";
// ...
const listDBGroups = useAppStore((s) => s.listDBGroupsForProject);
const groupsByName = useAppStore((s) => s.dbGroupsByName);
useEffect(() => { listDBGroups(projectName); }, [listDBGroups, projectName]);
const groups = useMemo(
  () => Object.values(groupsByName).filter((g) => g.name.startsWith(`${projectName}/`)),
  [groupsByName, projectName]
);
```

Verify the resulting list matches what the Pinia store produces today.

- [ ] **Step 3: `/simplify` checklist.**

- [ ] **Step 4: Remove inline definition; import the new file in `PlanDetailChangesBranch.tsx`.**

- [ ] **Step 5: Type-check + test + manual smoke (target selector → switch to group mode → list renders). Commit.**

```
git commit -m "refactor(plan-detail): extract DatabaseGroupSelector + slice migration"
```

## Task 3.4: Extract `DatabaseSelector` (with slice migration)

**Files:** Create `changes/TargetsSection/DatabaseSelector.tsx`.

- [ ] **Step 1: Lift source lines 1531-1706.**

- [ ] **Step 2: Replace `useDatabaseV1Store` with `useAppStore`.**

**Plan correction (2026-05-13):** the proto has `listDatabases` (not `searchDatabases`). Task 1.1 added `fetchDatabases` to the slice with shape `{ parent, pageSize, pageToken?, filter?: string, orderBy? }`. The Pinia store's `fetchDatabases` accepts a structured `DatabaseFilter` and compiles it down to the proto's CEL string `filter`; the React slice takes the CEL string directly. If `DatabaseSelector` currently builds a `DatabaseFilter` and passes it to the Pinia store, compile it to a string via the existing `getListDatabaseFilter` helper before calling the slice.

Before:
```ts
const dbStore = useVueState(() => useDatabaseV1Store());
const result = await dbStore.fetchDatabases({ parent, filter, pageSize, pageToken });
```

After:
```ts
import { getListDatabaseFilter } from "@/store/modules/v1/database"; // or a React-local copy
// ...
const fetchDatabases = useAppStore((s) => s.fetchDatabases);
const result = await fetchDatabases({
  parent,
  filter: getListDatabaseFilter(filter ?? {}),
  pageSize,
  pageToken,
});
```

- [ ] **Step 3: `/simplify` checklist.**

- [ ] **Step 4: Remove inline; import; type-check + test + manual smoke (search, select all, load more). Commit.**

```
git commit -m "refactor(plan-detail): extract DatabaseSelector + slice migration"
```

## Task 3.5: Extract `DatabaseAndGroupSelector`, `TargetSelectorSheet`, `TargetsSection`

**Files:**
- Create `changes/TargetsSection/DatabaseAndGroupSelector.tsx` (source 1460-1530)
- Create `changes/TargetsSection/TargetSelectorSheet.tsx` (source 1370-1459)
- Create `changes/TargetsSection/TargetsSection.tsx` (source 1122-1369)

Bottom-up extraction order. Each file: lift verbatim, replace Pinia reads with app-store calls, `/simplify`, remove inline definition, type-check + test + manual smoke, commit.

```
git commit -m "refactor(plan-detail): extract DatabaseAndGroupSelector"
git commit -m "refactor(plan-detail): extract TargetSelectorSheet"
git commit -m "refactor(plan-detail): extract TargetsSection"
```

## Task 3.6: Split `OptionsSection` per directive

**Files:**
- Create: `changes/OptionsSection/OptionsSection.tsx`
- Create: `changes/OptionsSection/RoleDirective.tsx`
- Create: `changes/OptionsSection/TransactionModeDirective.tsx`
- Create: `changes/OptionsSection/IsolationLevelDirective.tsx`
- Create: `changes/OptionsSection/PriorBackupToggle.tsx`
- Create: `changes/OptionsSection/GhostMigrationDirective.tsx`

Source: lines 652-1121.

- [ ] **Step 1: Read the current `OptionsSection` end-to-end.** Identify the five branches (role / transaction-mode / isolation-level / prior-backup / ghost). Each has its own visibility predicate + form UI + onChange wiring.

- [ ] **Step 2: For each directive, create one file** that takes `{ selectedSpec }` and returns `null` when its visibility predicate fails:

```tsx
// e.g. RoleDirective.tsx
import type { Plan_Spec } from "@/types/proto-es/v1/plan_service_pb";

export function RoleDirective({ selectedSpec }: { selectedSpec: Plan_Spec }) {
  if (!shouldShowRoleDirective(selectedSpec)) return null;
  // UI + onChange wiring lifted from current OptionsSection's role branch
  return (/* JSX */);
}

function shouldShowRoleDirective(spec: Plan_Spec): boolean {
  // port the predicate currently inline in OptionsSection
}
```

Repeat for the other four directives.

- [ ] **Step 3: Rewrite `OptionsSection.tsx` as composition**

```tsx
import type { Plan_Spec } from "@/types/proto-es/v1/plan_service_pb";
import { GhostMigrationDirective } from "./GhostMigrationDirective";
import { IsolationLevelDirective } from "./IsolationLevelDirective";
import { PriorBackupToggle } from "./PriorBackupToggle";
import { RoleDirective } from "./RoleDirective";
import { TransactionModeDirective } from "./TransactionModeDirective";

export function OptionsSection({ selectedSpec }: { selectedSpec: Plan_Spec }) {
  return (
    <div className="space-y-2">
      <RoleDirective selectedSpec={selectedSpec} />
      <TransactionModeDirective selectedSpec={selectedSpec} />
      <IsolationLevelDirective selectedSpec={selectedSpec} />
      <PriorBackupToggle selectedSpec={selectedSpec} />
      <GhostMigrationDirective selectedSpec={selectedSpec} />
    </div>
  );
}
```

- [ ] **Step 4: Remove the inline `OptionsSection`** from `PlanDetailChangesBranch.tsx`; import the new one.

- [ ] **Step 5: `/simplify` each new file (six total).**

- [ ] **Step 6: Type-check + run `utils/options.test.ts` (visibility logic) + run `PlanDetailChangesBranch.test.tsx`. Manual smoke**: open each spec type (DML, DDL, ghost-eligible) and verify the correct directives render.

- [ ] **Step 7: Commit**

```
git commit -m "refactor(plan-detail): split OptionsSection per directive"
```

## Task 3.7: Extract `SpecTabStrip`

**Files:** Create `changes/SpecTabStrip.tsx`.

- [ ] **Step 1: Identify the spec-tabs JSX** in `PlanDetailChangesBranch.tsx` (lines 168-651) — typically the first JSX block: numbered tabs, empty-statement markers, add/delete controls.

- [ ] **Step 2: Lift into `SpecTabStrip.tsx`** with props `{ specs, selectedSpecId, onSelect, onAdd, onDelete, canAdd, canDelete }`.

- [ ] **Step 3: Replace inline JSX with `<SpecTabStrip ... />`.**

- [ ] **Step 4: `/simplify`. Type-check + test + manual smoke (add a spec, switch tabs, delete a spec). Commit.**

```
git commit -m "refactor(plan-detail): extract SpecTabStrip"
```

## Task 3.8: Split `PlanDetailStatementSection.tsx`

**Files:**
- Create: `changes/StatementSection/StatementSection.tsx` (composition)
- Create: `changes/StatementSection/StatementEditor.tsx`
- Create: `changes/StatementSection/ReleaseFileSummary.tsx`
- Create: `changes/StatementSection/OversizedSheetNotice.tsx`

Source: `components/PlanDetailStatementSection.tsx` (748 lines).

- [ ] **Step 1: Read end-to-end. Identify the three render branches**: oversized notice / release-backed view / editable editor.

- [ ] **Step 2: Extract `OversizedSheetNotice.tsx`** (smallest first).

- [ ] **Step 3: Extract `ReleaseFileSummary.tsx`** (release-backed read-only view).

- [ ] **Step 4: Extract `StatementEditor.tsx`** (Monaco-backed editor + upload). Replace `useSheetV1Store` with `useAppStore.fetchSheet` / `useAppStore.createSheet`.

- [ ] **Step 5: Rewrite source as `StatementSection.tsx`** composition selecting one of the three children based on spec/sheet state.

- [ ] **Step 6: Move + update importers (no shim)**

```
git mv frontend/src/react/pages/project/plan-detail/components/PlanDetailStatementSection.tsx \
  frontend/src/react/pages/project/plan-detail/changes/StatementSection/StatementSection.tsx
```

```
grep -rn "PlanDetailStatementSection" frontend/src --include="*.ts" --include="*.tsx"
```

Repoint each import to the new path, renaming `PlanDetailStatementSection` → `StatementSection` at every call site.

- [ ] **Step 7: Type-check + tests + manual smoke (small SQL, large SQL → oversized notice, release-backed plan). Commit.**

```
git commit -m "refactor(plan-detail): split StatementSection 4 ways + slice migration"
```

## Task 3.9: Move `ChangesBranch` and its test

**Files:**
- Move: `components/PlanDetailChangesBranch.tsx` → `changes/ChangesBranch.tsx`
- Move: `components/PlanDetailChangesBranch.test.tsx` → `changes/__tests__/ChangesBranch.test.tsx`
- Update: every external importer

- [ ] **Step 1: Verify size of the source file** — after all earlier extractions it should be ≤400 lines.

```
wc -l frontend/src/react/pages/project/plan-detail/components/PlanDetailChangesBranch.tsx
```

- [ ] **Step 2: Move**

```
git mv frontend/src/react/pages/project/plan-detail/components/PlanDetailChangesBranch.tsx \
  frontend/src/react/pages/project/plan-detail/changes/ChangesBranch.tsx
git mv frontend/src/react/pages/project/plan-detail/components/PlanDetailChangesBranch.test.tsx \
  frontend/src/react/pages/project/plan-detail/changes/__tests__/ChangesBranch.test.tsx
```

Rename the export inside from `PlanDetailChangesBranch` to `ChangesBranch`. Update test imports — **no assertion changes**.

- [ ] **Step 3: Update every external importer**

```
grep -rn "PlanDetailChangesBranch" frontend/src --include="*.ts" --include="*.tsx"
```

Repoint each match.

- [ ] **Step 4: `/simplify` checklist on `ChangesBranch.tsx`.**

- [ ] **Step 5: Type-check + run `ChangesBranch.test.tsx` (must pass with no assertion changes) + manual full walk of Changes phase. Commit.**

```
git commit -m "refactor(plan-detail): move ChangesBranch and tests into changes/"
```

## Task 3.10: Move `PlanDetailChecks`, `PlanDetailDraftChecks`, `SchemaEditorSheet`, Changes-only hooks

For each, `git mv` to its new home, update internal imports, update external importers, apply `/simplify`, no shims.

```
git mv components/PlanDetailChecks.tsx changes/ChecksSection.tsx
git mv components/PlanDetailDraftChecks.tsx changes/DraftChecks.tsx
git mv components/SchemaEditorSheet.tsx changes/SchemaEditorSheet.tsx
git mv hooks/usePlanCheckActions.tsx changes/hooks/usePlanCheckActions.tsx
git mv hooks/usePlanDetailSpecValidation.ts changes/hooks/useSpecValidation.ts
```

After each move, rename internal exports (`PlanDetailChecks` → `ChecksSection`, etc.) and update every importer.

- [ ] **Step 1: Run each `git mv` above.**

- [ ] **Step 2: Update internal imports inside each moved file.**

- [ ] **Step 3: Update every external importer**

```
grep -rn "PlanDetailChecks\|PlanDetailDraftChecks\|SchemaEditorSheet\|usePlanCheckActions\|usePlanDetailSpecValidation" frontend/src --include="*.ts" --include="*.tsx"
```

- [ ] **Step 4: `/simplify` each.**

- [ ] **Step 5: Type-check + tests + manual smoke. Commit (one cohesive commit is fine).**

```
git commit -m "refactor(plan-detail): move Checks/SchemaEditor/hooks into changes/"
```

---

# Phase 4 — Review phase

**Goal:** Split `PlanDetailApprovalFlow.tsx` (864) into 8 files under `review/`. Migrate the 17 `useVueState` calls to app-store slices and direct Connect calls. Apply `/simplify`.

## Task 4.1: Extract small leaves (`IssueLinkButton`, `ReviewStatusTag`)

For each: identify the function inside `components/PlanDetailApprovalFlow.tsx`, copy to a new file (`review/IssueLinkButton.tsx`, `review/ReviewStatusTag.tsx`), remove inline definition, update any external importer. `/simplify` checklist. Type-check + test + commit per file.

```
git commit -m "refactor(plan-detail): extract IssueLinkButton"
git commit -m "refactor(plan-detail): extract ReviewStatusTag"
```

## Task 4.2: Extract `FutureReviewNotice`

Same procedure → `review/FutureReviewNotice.tsx`.

```
git commit -m "refactor(plan-detail): extract FutureReviewNotice"
```

## Task 4.3: Extract `ApproverList` (with slice migration)

**File:** `review/ApproverList.tsx`.

Contains Pinia reads (likely `useUserStore` for user lookups; `useCurrentUserV1` for self).

- [ ] **Step 1: Inspect the subtree's Pinia usages.**

- [ ] **Step 2: If a user-store read is needed and no app-store slice covers it**, stop and add a `UserSlice` (mirror `ProjectSlice` shape) as a new sub-task before continuing. Otherwise read users from data the page snapshot already loaded.

- [ ] **Step 3: Replace `useCurrentUserV1` reads with `useAppStore((s) => s.currentUser)`.**

- [ ] **Step 4: Lift the subtree. `/simplify`. Type-check + test + commit.**

```
git commit -m "refactor(plan-detail): extract ApproverList + slice migration"
```

## Task 4.4: Extract `RejectionBanner` (with slice migration)

**File:** `review/RejectionBanner.tsx`. Contains the re-request-review action — keep direct Connect service-client calls.

- [ ] Lift. Replace Pinia reads. `/simplify`. Manual smoke (open a rejected issue → banner renders → re-request flow works). Commit.

```
git commit -m "refactor(plan-detail): extract RejectionBanner + slice migration"
```

## Task 4.5: Extract `ApprovalStepItem` (with slice migration)

**File:** `review/ApprovalStepItem.tsx`. Renders one step (avatars, role names — likely Pinia-heavy).

- [ ] Lift. Replace Pinia. `/simplify`. Commit.

```
git commit -m "refactor(plan-detail): extract ApprovalStepItem + slice migration"
```

## Task 4.6: Extract `ReviewActions`

**File:** `review/ReviewActions.tsx`. Re-request + label-update actions. Service-client calls stay; replace Pinia reads.

```
git commit -m "refactor(plan-detail): extract ReviewActions + slice migration"
```

## Task 4.7: Move `ApprovalFlow` composition and test

- [ ] **Step 1: Move**

```
git mv frontend/src/react/pages/project/plan-detail/components/PlanDetailApprovalFlow.tsx \
  frontend/src/react/pages/project/plan-detail/review/ApprovalFlow.tsx
git mv frontend/src/react/pages/project/plan-detail/components/PlanDetailApprovalFlow.test.tsx \
  frontend/src/react/pages/project/plan-detail/review/__tests__/ApprovalFlow.test.tsx
```

Rename export `PlanDetailApprovalFlow` → `ApprovalFlow`. Update test imports — no assertion changes.

- [ ] **Step 2: Update every external importer**

```
grep -rn "PlanDetailApprovalFlow" frontend/src --include="*.ts" --include="*.tsx"
```

- [ ] **Step 3: `/simplify` on the composition.**

- [ ] **Step 4: Type-check + tests + manual smoke. Commit.**

```
git commit -m "refactor(plan-detail): move ApprovalFlow and tests into review/"
```

## Task 4.8: Create `ReviewBranch.tsx`

**File:** `review/ReviewBranch.tsx`. New file — the Review phase's outer JSX is currently inlined in `ProjectPlanDetailPage`.

- [ ] **Step 1: Identify the Review JSX in `ProjectPlanDetailPage.tsx`** (the block rendering `<ApprovalFlow />` vs. `<FutureReviewNotice />`).

- [ ] **Step 2: Lift into `review/ReviewBranch.tsx`. Replace the page's inline JSX with `<ReviewBranch />`.**

- [ ] **Step 3: Type-check + test + manual smoke. Commit.**

```
git commit -m "refactor(plan-detail): introduce ReviewBranch composition"
```

---

# Phase 5 — Deploy phase

**Goal:** Split `PlanDetailDeployFuture.tsx` (437) into 3 files. Split `PlanDetailTaskRolloutActionPanel.tsx` (743) into 7 action-specific files + a panel shell. Rename `Deploy`-prefixed files inside `deploy/`. Apply `/simplify` to touched files.

## Task 5.1: Split `PlanDetailDeployFuture.tsx`

**Files:**
- Create: `deploy/DeployFuture.tsx` (composition; ~180)
- Create: `deploy/RolloutRequirementsList.tsx`
- Create: `deploy/CreateRolloutSheet.tsx`

- [ ] **Step 1: Read source `components/PlanDetailDeployFuture.tsx`.** Three subtrees: requirements list, manual-rollout hint + create button, create-rollout sheet body.

- [ ] **Step 2: Extract `CreateRolloutSheet.tsx`** (sheet body, warning bypass, submit).

- [ ] **Step 3: Extract `RolloutRequirementsList.tsx`** (per-row done/pending/failed/optional state).

- [ ] **Step 4: Rewrite as `deploy/DeployFuture.tsx`** composition.

- [ ] **Step 5: Move + update importers (no shim)**

```
git mv frontend/src/react/pages/project/plan-detail/components/PlanDetailDeployFuture.tsx \
  frontend/src/react/pages/project/plan-detail/deploy/DeployFuture.tsx
grep -rn "PlanDetailDeployFuture" frontend/src --include="*.ts" --include="*.tsx"
```

Repoint each match; rename the export `PlanDetailDeployFuture` → `DeployFuture`.

- [ ] **Step 6: `/simplify` each. Type-check + test + manual smoke (open a plan needing rollout → Create Rollout → warnings listed). Commit.**

```
git commit -m "refactor(plan-detail): split DeployFuture into requirements + sheet"
```

## Task 5.2: Split `PlanDetailTaskRolloutActionPanel.tsx`

**Files:**
- Create: `deploy/actions/TaskRolloutActionPanel.tsx`
- Create: `deploy/actions/RunTaskAction.tsx`
- Create: `deploy/actions/RetryTaskAction.tsx`
- Create: `deploy/actions/SkipTaskAction.tsx`
- Create: `deploy/actions/CancelTaskAction.tsx`
- Create: `deploy/actions/ScheduleTaskAction.tsx`
- Create: `deploy/actions/SkipPriorBackupAction.tsx`

Source: 743 lines.

- [ ] **Step 1: Read source.** Identify each action's predicate + confirmation sheet + onClick.

- [ ] **Step 2: Move existing co-located action helpers under `deploy/actions/`**

```
git mv frontend/src/react/pages/project/plan-detail/components/deploy/taskActions.tsx \
  frontend/src/react/pages/project/plan-detail/deploy/actions/taskActions.tsx
git mv frontend/src/react/pages/project/plan-detail/components/deploy/taskActions.test.tsx \
  frontend/src/react/pages/project/plan-detail/deploy/actions/__tests__/taskActions.test.tsx
git mv frontend/src/react/pages/project/plan-detail/components/deploy/taskActionState.ts \
  frontend/src/react/pages/project/plan-detail/deploy/actions/taskActionState.ts
git mv frontend/src/react/pages/project/plan-detail/components/deploy/useDeployTaskStatement.tsx \
  frontend/src/react/pages/project/plan-detail/deploy/actions/useDeployTaskStatement.tsx
```

Update test imports — no assertion changes.

- [ ] **Step 3: Create each action file (six files).** Each takes `{ task, taskRun, onConfirm }` and renders `null` if its predicate fails (predicates reuse `taskActionState.ts`).

```tsx
// deploy/actions/RunTaskAction.tsx
export function RunTaskAction({ task, taskRun, onConfirm }: ActionProps) {
  if (!canRun({ task, taskRun, perms })) return null;
  // confirm sheet + button
}
```

- [ ] **Step 4: Create `TaskRolloutActionPanel.tsx`** as the composing shell:

```tsx
export function TaskRolloutActionPanel({ task, taskRun }: PanelProps) {
  const onConfirm = useTaskActionConfirm();
  return (
    <div className="flex gap-x-2">
      <RunTaskAction task={task} taskRun={taskRun} onConfirm={onConfirm} />
      <RetryTaskAction task={task} taskRun={taskRun} onConfirm={onConfirm} />
      <SkipTaskAction task={task} taskRun={taskRun} onConfirm={onConfirm} />
      <CancelTaskAction task={task} taskRun={taskRun} onConfirm={onConfirm} />
      <ScheduleTaskAction task={task} taskRun={taskRun} onConfirm={onConfirm} />
      <SkipPriorBackupAction task={task} taskRun={taskRun} onConfirm={onConfirm} />
    </div>
  );
}
```

- [ ] **Step 5: Move the original file + update importers**

```
git mv frontend/src/react/pages/project/plan-detail/components/PlanDetailTaskRolloutActionPanel.tsx \
  frontend/src/react/pages/project/plan-detail/deploy/actions/TaskRolloutActionPanel.tsx
grep -rn "PlanDetailTaskRolloutActionPanel" frontend/src --include="*.ts" --include="*.tsx"
```

Rename export to `TaskRolloutActionPanel`; repoint every importer.

- [ ] **Step 6: `/simplify` each new file.**

- [ ] **Step 7: Type-check + run `taskActions.test.tsx` (no assertion changes) + manual smoke for each action (run / retry / skip with reason / cancel / schedule / skip prior backup). Commit.**

```
git commit -m "refactor(plan-detail): split TaskRolloutActionPanel per action"
```

## Task 5.3: Rename remaining `Deploy*` and `PlanDetailTaskRun*` files

For each pair below, `git mv` to the new location, rename the export (drop `Deploy` / `PlanDetail` prefix), update every importer, apply `/simplify`. No shims.

```
git mv components/deploy/DeployBranch.tsx          deploy/DeployBranch.tsx           # keep prefix on the branch entry
git mv components/deploy/DeployTaskList.tsx        deploy/tasks/TaskList.tsx
git mv components/deploy/DeployTaskDetailPanel.tsx deploy/tasks/TaskDetailPanel.tsx
git mv components/deploy/DeployTaskRow.tsx         deploy/tasks/TaskRow.tsx
git mv components/deploy/DeployTaskItem.tsx        deploy/tasks/TaskItem.tsx
git mv components/deploy/DeployTaskFilter.tsx      deploy/tasks/TaskFilter.tsx
git mv components/deploy/DeployTaskToolbar.tsx     deploy/tasks/TaskToolbar.tsx
git mv components/deploy/DeployTaskStatus.tsx      deploy/tasks/TaskStatus.tsx
git mv components/deploy/DeployLatestTaskRunInfo.tsx deploy/tasks/LatestTaskRunInfo.tsx
git mv components/deploy/DeployStageCard.tsx        deploy/StageCard.tsx
git mv components/deploy/DeployStageContentView.tsx deploy/StageContentView.tsx
git mv components/deploy/DeployStageContentSidebar.tsx deploy/StageContentSidebar.tsx
git mv components/deploy/DeployStageActionSection.tsx  deploy/StageActionSection.tsx
git mv components/deploy/DeployPendingTasksSection.tsx deploy/PendingTasksSection.tsx
git mv components/deploy/DeployReleaseInfoCard.tsx     deploy/ReleaseInfoCard.tsx
git mv components/deploy/taskRunUtils.ts deploy/utils/taskRunUtils.ts
git mv components/deploy/types.ts        deploy/utils/types.ts
git mv components/PlanDetailTaskRunTable.tsx   deploy/tasks/TaskRunTable.tsx
git mv components/PlanDetailTaskRunSession.tsx deploy/tasks/TaskRunSession.tsx
git mv components/PlanDetailTaskRunDetail.tsx  deploy/tasks/TaskRunDetail.tsx
git mv components/PlanDetailRollbackSheet.tsx  deploy/tasks/RollbackSheet.tsx
```

- [ ] **Step 1: Run each move above.** Commit per logical group:

```
git commit -m "refactor(plan-detail): rename deploy/tasks/* (drop Deploy prefix)"
git commit -m "refactor(plan-detail): rename deploy/Stage*.tsx (drop Deploy prefix)"
git commit -m "refactor(plan-detail): rename deploy/{Release,Pending}* and utils"
git commit -m "refactor(plan-detail): move PlanDetailTaskRun*/RollbackSheet into deploy/tasks/"
```

- [ ] **Step 2: For each moved file**: update its internal imports, update its exported name, update every external importer. `/simplify` checklist.

- [ ] **Step 3: After all moves**, verify line budgets:

```
wc -l frontend/src/react/pages/project/plan-detail/deploy/**/*.{ts,tsx} | sort -rn
```

No file over ~400.

- [ ] **Step 4: Full type-check + test + manual smoke of deploy phase (stage select, run task, skip with reason, cancel, schedule, rollback).**

---

# Phase 6 — Header + Sidebar + cleanup

**Goal:** Split `PlanDetailHeader.tsx` (761) and `PlanDetailMetadataSidebar.tsx` (371). Land the CI guard from Task 1.7 (now that violation count is zero). Add the addendum to the current-state doc. Final verification + open the PR.

## Task 6.1: Split `PlanDetailHeader.tsx`

**Files:**
- Create: `header/PlanDetailHeader.tsx` (composition)
- Create: `header/TitleEditor.tsx`
- Create: `header/DescriptionEditor.tsx`
- Create: `header/ReadyForReviewPopover.tsx`
- Create: `header/CreatePlanButton.tsx`
- Create: `header/CloseReopenActions.tsx`
- Create: `header/MobileDetailsButton.tsx`

Source: `components/PlanDetailHeader.tsx` (761 lines).

- [ ] **Step 1: Extract leaves bottom-up** (smallest first):
  - `MobileDetailsButton`
  - `CreatePlanButton`
  - `CloseReopenActions`
  - `TitleEditor` (owns draft + save state)
  - `DescriptionEditor` (owns draft + collapsed-when-long)
  - `ReadyForReviewPopover` (largest leaf)

For each: lift, `/simplify`, replace inline with import, type-check + test + commit.

- [ ] **Step 2: Move the composition**

```
git mv frontend/src/react/pages/project/plan-detail/components/PlanDetailHeader.tsx \
  frontend/src/react/pages/project/plan-detail/header/PlanDetailHeader.tsx
grep -rn "PlanDetailHeader" frontend/src --include="*.ts" --include="*.tsx"
```

Update every importer (the exported name `PlanDetailHeader` is kept — Header is the area's public entry).

- [ ] **Step 3: Manual smoke**: edit title, edit description (long + short), Ready-for-Review with required label, close+reopen draft, mobile details button on mobile viewport. Commit.

```
git commit -m "refactor(plan-detail): split PlanDetailHeader into 7 focused files"
```

## Task 6.2: Split `PlanDetailMetadataSidebar.tsx`

**Files:**
- Create: `sidebar/MetadataSidebar.tsx` (composition)
- Create: `sidebar/CreatedBySection.tsx`
- Create: `sidebar/StatusSection.tsx`
- Create: `sidebar/PlanCheckSummary.tsx`
- Create: `sidebar/ApprovalSummary.tsx`
- Create: `sidebar/IssueLabelsSection.tsx`
- Create: `sidebar/RolloutStageProgress.tsx`
- Create: `sidebar/RefreshControl.tsx`
- Create: `sidebar/FutureSectionHint.tsx`

Same pattern: lift each section into its own file, rewrite parent as composition.

- [ ] **Step 1: Extract each section file by file.** Per section: lift, `/simplify`, replace inline.

- [ ] **Step 2: Move and rename**

```
git mv frontend/src/react/pages/project/plan-detail/components/PlanDetailMetadataSidebar.tsx \
  frontend/src/react/pages/project/plan-detail/sidebar/MetadataSidebar.tsx
grep -rn "PlanDetailMetadataSidebar" frontend/src --include="*.ts" --include="*.tsx"
```

Repoint each importer; rename export `PlanDetailMetadataSidebar` → `MetadataSidebar`.

- [ ] **Step 3: Type-check + test + manual smoke. Commit.**

```
git commit -m "refactor(plan-detail): split MetadataSidebar per section"
```

## Task 6.3: Delete empty legacy folders

- [ ] **Step 1: Verify they are empty**

```
find frontend/src/react/pages/project/plan-detail/components -type f
find frontend/src/react/pages/project/plan-detail/hooks -type f
find frontend/src/react/pages/project/plan-detail/context -type f
```

Each command must return no files (everything has been moved).

- [ ] **Step 2: Remove the directories**

```
git rm -r frontend/src/react/pages/project/plan-detail/components
git rm -r frontend/src/react/pages/project/plan-detail/hooks
git rm -r frontend/src/react/pages/project/plan-detail/context
```

- [ ] **Step 3: Commit**

```
git commit -m "refactor(plan-detail): remove emptied legacy folders"
```

## Task 6.4: Land the strict CI guard

Recover the stashed guard from Task 1.7.

- [ ] **Step 1: Verify no Pinia reads remain in plan-detail/**

```
PLAN_DETAIL_DIR=frontend/src/react/pages/project/plan-detail
grep -rnE "useVueState\(\s*\(\)\s*=>\s*use[A-Z][A-Za-z]+Store" "$PLAN_DETAIL_DIR" \
  --include="*.ts" --include="*.tsx" \
  | grep -v "useRouter\|useRoute"
```

Must return empty. If anything appears, stop and clean it up before continuing.

- [ ] **Step 2: Pop the stash from Task 1.7**

```
git stash list   # find the index of the stash with the guard edit
git stash pop <index>
```

- [ ] **Step 3: Run the guard locally**

It must exit cleanly (zero violations).

- [ ] **Step 4: Commit**

```
git add <guard-script-path>
git commit -m "ci: forbid Pinia reads inside plan-detail/ via useVueState"
```

## Task 6.5: Addendum to current-state doc

**File:** `docs/plans/2026-05-13-plan-detail-current-state.md`

- [ ] **Step 1: Append an "Update — 2026-XX-XX" section** listing:
  - the refactor landed (link to the PR)
  - the new file locations replacing the "Implementation Pointers" section
  - the legacy `components/`, `hooks/`, `context/` folders no longer exist

- [ ] **Step 2: Commit**

```
git commit -m "docs(plan-detail): addendum noting refactor landed and new paths"
```

## Task 6.6: Final verification

- [ ] **Step 1: Verify line budgets**

```
find frontend/src/react/pages/project/plan-detail -type f \( -name "*.ts" -o -name "*.tsx" \) -exec wc -l {} + | sort -rn | head -10
```

Top file ≤ ~400 lines. `usePlanDetailPage.ts` ≤ 120 lines.

- [ ] **Step 2: Verify folder layout matches spec**

```
ls -la frontend/src/react/pages/project/plan-detail
```

Expected top-level: `ProjectPlanDetailPage.tsx`, `index.ts`, `shell/`, `header/`, `changes/`, `review/`, `deploy/`, `sidebar/`, `shared/`. No `components/`, `hooks/`, `context/`.

- [ ] **Step 3: Full type-check + fix + test**

```
pnpm --dir frontend type-check
pnpm --dir frontend fix
pnpm --dir frontend test
```

- [ ] **Step 4: Run the e2e smoke test from Task 1.8**

Must pass.

- [ ] **Step 5: Manual full-journey walk** (per spec Section 6):
- open empty draft plan → edit a SQL statement → run checks → Ready for Review (creates issue) → approve the issue → create rollout → run one task → observe task done
- on desktop wide / desktop narrow / mobile

- [ ] **Step 6: Walk `docs/pre-pr-checklist.md`.**

## Task 6.7: Open the PR

- [ ] **Step 1: Push**

```
git push -u origin refactor/plan-detail-maintainability
```

- [ ] **Step 2: Open the PR**

```
gh pr create --title "refactor(plan-detail): maintainability pass" --body "$(cat <<'EOF'
## Summary

Maintainability refactor for `frontend/src/react/pages/project/plan-detail/` before resuming feature work.

- Splits 6 oversized files (largest 1969 lines) into ~60 focused files; no file >400 lines after.
- New folder layout: `shell/ header/ changes/ review/ deploy/ sidebar/ shared/`. Legacy `components/`, `hooks/`, `context/` folders removed.
- `usePlanDetailPage` reduced from 741 lines to a ~120-line composition of phase-scoped hooks under `shell/hooks/`.
- Plan-detail's Pinia reads replaced with new app-store Zustand slices (`DatabaseSlice`, `DBGroupSlice`, `SheetSlice`, `InstanceRoleSlice`); a page-scoped Zustand store under `shared/stores/` holds page state (snapshot, phases, editing, selection, polling).
- CI guard added: `useVueState` inside `plan-detail/` may not reference Pinia store hooks (router excepted).
- `/simplify` checklist applied per split file (excluding `useMemo`/`useCallback` rewrites per design).
- New unit tests for `parseStatement`, `decideLeaveAction`, each new app-store slice, and the page-scoped store.
- E2E smoke test for plan-detail page load.

Design spec: `docs/superpowers/specs/2026-05-13-plan-detail-refactor-design.md`.

Behavior is preserved. All pre-refactor tests pass unchanged.

## Test plan
- [ ] pnpm --dir frontend type-check
- [ ] pnpm --dir frontend fix (no diff)
- [ ] pnpm --dir frontend test
- [ ] CI guard for Pinia reads passes
- [ ] E2E smoke test passes
- [ ] Manual: full golden-path journey on desktop wide / narrow / mobile

🤖 Generated with [Claude Code](https://claude.com/claude-code)
EOF
)"
```

Return the PR URL.

---

# After merge

- [ ] Verify the current-state doc addendum reflects the merged state.
- [ ] Verify CI runs the strict guard on `main`.
- [ ] Unblock feature work on plan-detail.

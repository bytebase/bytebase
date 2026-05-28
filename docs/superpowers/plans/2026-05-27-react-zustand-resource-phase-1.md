# React Zustand Resource Migration Phase 1 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Migrate the phase 1 access/account API resources from legacy Pinia stores into React `useAppStore` Zustand slices and switch React consumers to those slices.

**Architecture:** Add one focused Zustand slice per resource under `frontend/src/react/stores/app/`, composed by `createAppStore()`. Keep legacy Pinia stores in place for Vue or not-yet-migrated callers. Migrate React consumers only after the slice behavior is covered by tests.

**Tech Stack:** React, TypeScript, Zustand, Vitest, Connect RPC, proto-es, Pinia compatibility during migration.

---

## File Structure

Create:

- `frontend/src/react/stores/app/group.ts`
- `frontend/src/react/stores/app/serviceAccount.ts`
- `frontend/src/react/stores/app/workloadIdentity.ts`
- `frontend/src/react/stores/app/identityProvider.ts`
- `frontend/src/react/stores/app/accessGrant.ts`

Modify:

- `frontend/src/react/stores/app/types.ts`
- `frontend/src/react/stores/app/index.ts`
- `frontend/src/react/stores/app/index.test.ts`
- `frontend/src/react/no-legacy-vue-deps.test.ts`
- React consumers listed in Tasks 7 and 8

Keep unchanged in this phase:

- `frontend/src/store/modules/v1/group.ts`
- `frontend/src/store/modules/serviceAccount.ts`
- `frontend/src/store/modules/workloadIdentity.ts`
- `frontend/src/store/modules/idp.ts`
- `frontend/src/store/modules/accessGrant.ts`

Legacy stores are deleted only after a later audit confirms no Vue or React consumers remain.

---

### Task 1: Add App Store Type Contracts

**Files:**
- Modify: `frontend/src/react/stores/app/types.ts`

- [ ] **Step 1: Add imports for phase 1 resource types**

Add these imports near the existing proto resource imports:

```ts
import type { AccessGrant } from "@/types/proto-es/v1/access_grant_service_pb";
import type { Group } from "@/types/proto-es/v1/group_service_pb";
import type { IdentityProvider } from "@/types/proto-es/v1/idp_service_pb";
import type { ServiceAccount } from "@/types/proto-es/v1/service_account_service_pb";
import type { WorkloadIdentity } from "@/types/proto-es/v1/workload_identity_service_pb";
import type { State } from "@/types/proto-es/v1/common_pb";
```

- [ ] **Step 2: Add shared filter and params types**

Add these types after `ProjectListParams`:

```ts
export type GroupFilter = {
  query?: string;
  project?: string;
};

export type AccountFilter = {
  query?: string;
  state?: State;
};

export type ListServiceAccountsParams = {
  parent: string;
  pageSize: number;
  pageToken?: string;
  showDeleted: boolean;
  filter?: AccountFilter;
};

export type ListWorkloadIdentitiesParams = {
  parent: string;
  pageSize: number;
  pageToken?: string;
  showDeleted: boolean;
  filter?: AccountFilter;
};

export type AccessGrantFilter = {
  name?: string;
  statement?: string;
  creator?: string;
  status?: import("@/utils").AccessGrantFilterStatus[];
  issue?: string;
  target?: string;
  createdTsAfter?: number;
  createdTsBefore?: number;
};

export type ListAccessGrantsParams = {
  parent: string;
  filter?: AccessGrantFilter;
  pageSize?: number;
  pageToken?: string;
  orderBy?: string;
};
```

- [ ] **Step 3: Add slice type definitions**

Add these slice definitions after `InstanceRoleSlice`:

```ts
export type GroupSlice = {
  groupsByName: Record<string, Group>;
  groupRequests: Record<string, Promise<Group | undefined>>;
  groupErrorsByName: Record<string, Error | undefined>;
  listGroups: (params: {
    pageSize: number;
    pageToken?: string;
    filter?: GroupFilter;
  }) => Promise<{ groups: Group[]; nextPageToken: string }>;
  batchFetchGroups: (names: string[]) => Promise<Group[]>;
  batchGetOrFetchGroups: (names: string[]) => Promise<(Group | undefined)[]>;
  fetchGroup: (id: string) => Promise<Group | undefined>;
  getGroupByIdentifier: (id: string) => Group | undefined;
  createGroup: (group: Group) => Promise<Group>;
  updateGroup: (group: Group) => Promise<Group>;
  deleteGroup: (name: string) => Promise<void>;
};

export type ServiceAccountSlice = {
  serviceAccountsByName: Record<string, ServiceAccount>;
  serviceAccountRequests: Record<
    string,
    Promise<ServiceAccount | undefined>
  >;
  listServiceAccounts: (
    params: ListServiceAccountsParams
  ) => Promise<{
    serviceAccounts: ServiceAccount[];
    nextPageToken: string;
  }>;
  fetchServiceAccount: (
    name: string,
    silent?: boolean
  ) => Promise<ServiceAccount | undefined>;
  getServiceAccount: (name: string) => ServiceAccount;
  createServiceAccount: (
    serviceAccountId: string,
    serviceAccount: Partial<ServiceAccount>,
    parent: string
  ) => Promise<ServiceAccount>;
  updateServiceAccount: (
    serviceAccount: Partial<ServiceAccount>,
    updateMask: { paths: string[] }
  ) => Promise<ServiceAccount>;
  deleteServiceAccount: (name: string) => Promise<void>;
  undeleteServiceAccount: (name: string) => Promise<ServiceAccount>;
};

export type WorkloadIdentitySlice = {
  workloadIdentitiesByName: Record<string, WorkloadIdentity>;
  workloadIdentityRequests: Record<
    string,
    Promise<WorkloadIdentity | undefined>
  >;
  listWorkloadIdentities: (
    params: ListWorkloadIdentitiesParams
  ) => Promise<{
    workloadIdentities: WorkloadIdentity[];
    nextPageToken: string;
  }>;
  fetchWorkloadIdentity: (
    name: string,
    silent?: boolean
  ) => Promise<WorkloadIdentity | undefined>;
  getWorkloadIdentity: (name: string) => WorkloadIdentity;
  createWorkloadIdentity: (
    workloadIdentityId: string,
    workloadIdentity: Partial<WorkloadIdentity>,
    parent: string
  ) => Promise<WorkloadIdentity>;
  updateWorkloadIdentity: (
    workloadIdentity: Partial<WorkloadIdentity>,
    updateMask: { paths: string[] }
  ) => Promise<WorkloadIdentity>;
  deleteWorkloadIdentity: (name: string) => Promise<void>;
  undeleteWorkloadIdentity: (name: string) => Promise<WorkloadIdentity>;
};

export type IdentityProviderSlice = {
  identityProvidersByName: Record<string, IdentityProvider>;
  identityProviderRequests: Record<
    string,
    Promise<IdentityProvider | undefined>
  >;
  identityProviderList: () => IdentityProvider[];
  listIdentityProviders: (parent?: string) => Promise<IdentityProvider[]>;
  fetchIdentityProvider: (
    name: string,
    silent?: boolean
  ) => Promise<IdentityProvider | undefined>;
  getIdentityProvider: (name: string) => IdentityProvider | undefined;
  createIdentityProvider: (
    identityProvider: IdentityProvider
  ) => Promise<IdentityProvider>;
  updateIdentityProvider: (
    update: Partial<IdentityProvider>
  ) => Promise<IdentityProvider>;
  deleteIdentityProvider: (name: string) => Promise<void>;
};

export type AccessGrantSlice = {
  accessGrantsByName: Record<string, AccessGrant>;
  accessGrantRequests: Record<string, Promise<AccessGrant | undefined>>;
  fetchAccessGrant: (name: string) => Promise<AccessGrant | undefined>;
  searchMyAccessGrants: (
    params: ListAccessGrantsParams
  ) => Promise<{
    accessGrants: AccessGrant[];
    nextPageToken: string;
  }>;
  listAccessGrants: (
    params: ListAccessGrantsParams
  ) => Promise<{
    accessGrants: AccessGrant[];
    nextPageToken: string;
  }>;
  createAccessGrant: (
    parent: string,
    accessGrant: AccessGrant
  ) => Promise<AccessGrant>;
  activateAccessGrant: (name: string) => Promise<AccessGrant>;
  revokeAccessGrant: (name: string) => Promise<AccessGrant>;
};
```

- [ ] **Step 4: Extend `AppStoreState`**

Add the five slice types into the `AppStoreState` intersection:

```ts
  InstanceRoleSlice &
  GroupSlice &
  ServiceAccountSlice &
  WorkloadIdentitySlice &
  IdentityProviderSlice &
  AccessGrantSlice &
  NotificationSlice &
```

- [ ] **Step 5: Run focused type-check**

Run:

```bash
pnpm --dir frontend type-check
```

Expected: Type-check fails because the new slice types are not yet implemented in `createAppStore()`. This is the intended TDD failure.

---

### Task 2: Add Group Slice

**Files:**
- Create: `frontend/src/react/stores/app/group.ts`
- Modify: `frontend/src/react/stores/app/index.ts`
- Test: `frontend/src/react/stores/app/index.test.ts`

- [ ] **Step 1: Add connect mocks and test data**

In `index.test.ts`, extend the hoisted mocks:

```ts
listGroups: vi.fn(),
batchGetGroups: vi.fn(),
getGroup: vi.fn(),
createGroup: vi.fn(),
updateGroup: vi.fn(),
deleteGroup: vi.fn(),
```

Add the mocked connect client:

```ts
groupServiceClientConnect: {
  listGroups: mocks.listGroups,
  batchGetGroups: mocks.batchGetGroups,
  getGroup: mocks.getGroup,
  createGroup: mocks.createGroup,
  updateGroup: mocks.updateGroup,
  deleteGroup: mocks.deleteGroup,
},
```

Add imports:

```ts
import { GroupSchema } from "@/types/proto-es/v1/group_service_pb";
```

Add fixtures after `projectB`:

```ts
const groupA = createProto(GroupSchema, {
  name: "groups/dba@example.com",
  email: "dba@example.com",
  title: "DBA",
});

const groupB = createProto(GroupSchema, {
  name: "groups/dev@example.com",
  email: "dev@example.com",
  title: "Dev",
});
```

- [ ] **Step 2: Add failing group tests**

Add these tests inside `describe("useAppStore", ...)`:

```ts
test("lists groups and populates the group cache", async () => {
  mocks.listGroups.mockResolvedValue({
    groups: [groupA, groupB],
    nextPageToken: "next",
  });
  const store = createAppStore();
  store.setState({
    currentUser: user,
    roles: [createProto(RoleSchema, {
      name: "roles/group-viewer",
      permissions: ["bb.groups.list"],
    })],
    workspacePolicy: createProto(IamPolicySchema, {
      bindings: [createProto(BindingSchema, {
        role: "roles/group-viewer",
        members: [`user:${user.email}`],
      })],
    }),
  });

  const result = await store.getState().listGroups({ pageSize: 50 });

  expect(result.groups).toEqual([groupA, groupB]);
  expect(result.nextPageToken).toBe("next");
  expect(store.getState().groupsByName[groupA.name]).toBe(groupA);
  expect(store.getState().groupsByName[groupB.name]).toBe(groupB);
});

test("batchGetOrFetchGroups skips cached groups and preserves order", async () => {
  mocks.batchGetGroups.mockResolvedValue({ groups: [groupB] });
  const store = createAppStore();
  store.setState({ groupsByName: { [groupA.name]: groupA } });

  const result = await store
    .getState()
    .batchGetOrFetchGroups([groupA.name, groupB.name, groupA.name]);

  expect(mocks.batchGetGroups).toHaveBeenCalledTimes(1);
  expect(result).toEqual([groupA, groupB]);
  expect(store.getState().groupsByName[groupB.name]).toBe(groupB);
});
```

Run:

```bash
pnpm --dir frontend exec vitest run frontend/src/react/stores/app/index.test.ts
```

Expected: FAIL because `listGroups` and `batchGetOrFetchGroups` are not implemented.

- [ ] **Step 3: Create `group.ts`**

Create the slice with this structure:

```ts
import { create as createProto } from "@bufbuild/protobuf";
import { createContextValues } from "@connectrpc/connect";
import { uniq } from "lodash-es";
import { groupServiceClientConnect } from "@/connect";
import { silentContextKey } from "@/connect/context-key";
import { isValidProjectName } from "@/types";
import type { Group } from "@/types/proto-es/v1/group_service_pb";
import {
  BatchGetGroupsRequestSchema,
  CreateGroupRequestSchema,
  DeleteGroupRequestSchema,
  GetGroupRequestSchema,
  ListGroupsRequestSchema,
  UpdateGroupRequestSchema,
} from "@/types/proto-es/v1/group_service_pb";
import { extractGroupEmail, groupNamePrefix } from "@/store/modules/v1/common";
import type { AppSliceCreator, GroupFilter, GroupSlice } from "./types";

export const buildGroupFilter = (params: GroupFilter) => {
  const filter = [];
  const search = params.query?.trim()?.toLowerCase();
  if (search) {
    filter.push(`(title.contains("${search}") || email.contains("${search}"))`);
  }
  if (isValidProjectName(params.project)) {
    filter.push(`project == "${params.project}"`);
  }
  return filter.join(" && ");
};

export const ensureGroupIdentifier = (id: string) => {
  const email = extractGroupEmail(id);
  return `${groupNamePrefix}${email}`;
};

export const createGroupSlice: AppSliceCreator<GroupSlice> = (set, get) => ({
  groupsByName: {},
  groupRequests: {},
  groupErrorsByName: {},

  listGroups: async ({ pageSize, pageToken, filter }) => {
    if (!get().hasWorkspacePermission("bb.groups.list")) {
      return { groups: [], nextPageToken: "" };
    }
    const response = await groupServiceClientConnect.listGroups(
      createProto(ListGroupsRequestSchema, {
        pageSize,
        pageToken,
        filter: buildGroupFilter(filter ?? {}),
      }),
      { contextValues: createContextValues().set(silentContextKey, true) }
    );
    set((state) => ({
      groupsByName: {
        ...state.groupsByName,
        ...Object.fromEntries(
          response.groups.map((group) => [group.name, group])
        ),
      },
    }));
    return {
      groups: response.groups,
      nextPageToken: response.nextPageToken,
    };
  },

  batchFetchGroups: async (names) => {
    const validNames = uniq(names).filter(Boolean).map(ensureGroupIdentifier);
    if (validNames.length === 0) return [];
    const response = await groupServiceClientConnect.batchGetGroups(
      createProto(BatchGetGroupsRequestSchema, { names: validNames }),
      { contextValues: createContextValues().set(silentContextKey, true) }
    );
    set((state) => ({
      groupsByName: {
        ...state.groupsByName,
        ...Object.fromEntries(
          response.groups.map((group) => [group.name, group])
        ),
      },
    }));
    return response.groups;
  },

  batchGetOrFetchGroups: async (names) => {
    const validNames = uniq(names).filter(Boolean).map(ensureGroupIdentifier);
    const missing = validNames.filter((name) => !get().groupsByName[name]);
    if (missing.length > 0) {
      await get().batchFetchGroups(missing);
    }
    return validNames.map((name) => get().groupsByName[name]);
  },

  fetchGroup: async (id) => {
    if (!get().hasWorkspacePermission("bb.groups.get")) return undefined;
    const name = ensureGroupIdentifier(id);
    const existing = get().groupsByName[name];
    if (existing) return existing;
    const pending = get().groupRequests[name];
    if (pending) return pending;
    const request = groupServiceClientConnect
      .getGroup(createProto(GetGroupRequestSchema, { name }), {
        contextValues: createContextValues().set(silentContextKey, true),
      })
      .then((group) => {
        set((state) => {
          const { [name]: _, ...groupRequests } = state.groupRequests;
          return {
            groupsByName: { ...state.groupsByName, [group.name]: group },
            groupErrorsByName: { ...state.groupErrorsByName, [name]: undefined },
            groupRequests,
          };
        });
        return group;
      })
      .catch((error) => {
        set((state) => {
          const { [name]: _, ...groupRequests } = state.groupRequests;
          return {
            groupErrorsByName: {
              ...state.groupErrorsByName,
              [name]: error instanceof Error ? error : new Error(String(error)),
            },
            groupRequests,
          };
        });
        return undefined;
      });
    set((state) => ({
      groupRequests: { ...state.groupRequests, [name]: request },
    }));
    return request;
  },

  getGroupByIdentifier: (id) => get().groupsByName[ensureGroupIdentifier(id)],

  createGroup: async (group) => {
    const response = await groupServiceClientConnect.createGroup(
      createProto(CreateGroupRequestSchema, {
        group,
        groupEmail: extractGroupEmail(group.name),
      })
    );
    set((state) => ({
      groupsByName: { ...state.groupsByName, [response.name]: response },
    }));
    return response;
  },

  updateGroup: async (group) => {
    const response = await groupServiceClientConnect.updateGroup(
      createProto(UpdateGroupRequestSchema, {
        group,
        updateMask: { paths: ["title", "description", "members"] },
        allowMissing: false,
      })
    );
    set((state) => ({
      groupsByName: { ...state.groupsByName, [response.name]: response },
    }));
    return response;
  },

  deleteGroup: async (name) => {
    await groupServiceClientConnect.deleteGroup(
      createProto(DeleteGroupRequestSchema, { name })
    );
    set((state) => {
      const { [name]: _, ...groupsByName } = state.groupsByName;
      return { groupsByName };
    });
  },
});
```

- [ ] **Step 4: Compose the group slice**

In `index.ts`, import and spread the slice:

```ts
import { createGroupSlice } from "./group";
```

```ts
    ...createInstanceRoleSlice(...args),
    ...createGroupSlice(...args),
    ...createNotificationSlice(...args),
```

- [ ] **Step 5: Run group tests**

Run:

```bash
pnpm --dir frontend exec vitest run frontend/src/react/stores/app/index.test.ts
pnpm --dir frontend type-check
```

Expected: PASS for the new group tests. Type-check may still fail if later task type imports are already declared but slices are not implemented; if Task 1 added only Group types so far, it should pass.

- [ ] **Step 6: Commit group slice**

```bash
git add frontend/src/react/stores/app/types.ts frontend/src/react/stores/app/index.ts frontend/src/react/stores/app/group.ts frontend/src/react/stores/app/index.test.ts
git commit -m "feat(frontend): add group resource to app store"
```

---

### Task 3: Add Service Account and Workload Identity Slices

**Files:**
- Create: `frontend/src/react/stores/app/serviceAccount.ts`
- Create: `frontend/src/react/stores/app/workloadIdentity.ts`
- Modify: `frontend/src/react/stores/app/index.ts`
- Test: `frontend/src/react/stores/app/index.test.ts`

- [ ] **Step 1: Add connect mocks and fixtures**

Extend the hoisted mocks:

```ts
listServiceAccounts: vi.fn(),
getServiceAccount: vi.fn(),
createServiceAccount: vi.fn(),
updateServiceAccount: vi.fn(),
deleteServiceAccount: vi.fn(),
undeleteServiceAccount: vi.fn(),
listWorkloadIdentities: vi.fn(),
getWorkloadIdentity: vi.fn(),
createWorkloadIdentity: vi.fn(),
updateWorkloadIdentity: vi.fn(),
deleteWorkloadIdentity: vi.fn(),
undeleteWorkloadIdentity: vi.fn(),
```

Add mocked connect clients:

```ts
serviceAccountServiceClientConnect: {
  listServiceAccounts: mocks.listServiceAccounts,
  getServiceAccount: mocks.getServiceAccount,
  createServiceAccount: mocks.createServiceAccount,
  updateServiceAccount: mocks.updateServiceAccount,
  deleteServiceAccount: mocks.deleteServiceAccount,
  undeleteServiceAccount: mocks.undeleteServiceAccount,
},
workloadIdentityServiceClientConnect: {
  listWorkloadIdentities: mocks.listWorkloadIdentities,
  getWorkloadIdentity: mocks.getWorkloadIdentity,
  createWorkloadIdentity: mocks.createWorkloadIdentity,
  updateWorkloadIdentity: mocks.updateWorkloadIdentity,
  deleteWorkloadIdentity: mocks.deleteWorkloadIdentity,
  undeleteWorkloadIdentity: mocks.undeleteWorkloadIdentity,
},
```

Add imports:

```ts
import { ServiceAccountSchema } from "@/types/proto-es/v1/service_account_service_pb";
import { WorkloadIdentitySchema } from "@/types/proto-es/v1/workload_identity_service_pb";
```

Add fixtures:

```ts
const serviceAccountA = createProto(ServiceAccountSchema, {
  name: "serviceAccounts/robot@example.com",
  email: "robot@example.com",
  title: "Robot",
  state: State.ACTIVE,
});

const workloadIdentityA = createProto(WorkloadIdentitySchema, {
  name: "workloadIdentities/deploy@example.com",
  email: "deploy@example.com",
  title: "Deploy",
  state: State.ACTIVE,
});
```

- [ ] **Step 2: Add failing account tests**

Add tests:

```ts
test("lists service accounts and writes them into cache", async () => {
  mocks.listServiceAccounts.mockResolvedValue({
    serviceAccounts: [serviceAccountA],
    nextPageToken: "next-sa",
  });
  const store = createAppStore();

  const result = await store.getState().listServiceAccounts({
    parent: "workspaces/default",
    pageSize: 20,
    showDeleted: false,
  });

  expect(result.serviceAccounts).toEqual([serviceAccountA]);
  expect(result.nextPageToken).toBe("next-sa");
  expect(store.getState().serviceAccountsByName[serviceAccountA.name]).toBe(
    serviceAccountA
  );
});

test("marks cached service account deleted after delete", async () => {
  mocks.deleteServiceAccount.mockResolvedValue({});
  const store = createAppStore();
  store.setState({
    serviceAccountsByName: { [serviceAccountA.name]: serviceAccountA },
  });

  await store.getState().deleteServiceAccount(serviceAccountA.name);

  expect(
    store.getState().serviceAccountsByName[serviceAccountA.name].state
  ).toBe(State.DELETED);
});

test("lists workload identities and writes them into cache", async () => {
  mocks.listWorkloadIdentities.mockResolvedValue({
    workloadIdentities: [workloadIdentityA],
    nextPageToken: "next-wi",
  });
  const store = createAppStore();

  const result = await store.getState().listWorkloadIdentities({
    parent: "workspaces/default",
    pageSize: 20,
    showDeleted: false,
  });

  expect(result.workloadIdentities).toEqual([workloadIdentityA]);
  expect(result.nextPageToken).toBe("next-wi");
  expect(
    store.getState().workloadIdentitiesByName[workloadIdentityA.name]
  ).toBe(workloadIdentityA);
});
```

Run:

```bash
pnpm --dir frontend exec vitest run frontend/src/react/stores/app/index.test.ts
```

Expected: FAIL because the account slices are not implemented.

- [ ] **Step 3: Create `serviceAccount.ts`**

Create the file with this structure:

```ts
import { create as createProto } from "@bufbuild/protobuf";
import { createContextValues } from "@connectrpc/connect";
import { serviceAccountServiceClientConnect } from "@/connect";
import { silentContextKey } from "@/connect/context-key";
import { State } from "@/types/proto-es/v1/common_pb";
import type { ServiceAccount } from "@/types/proto-es/v1/service_account_service_pb";
import {
  CreateServiceAccountRequestSchema,
  DeleteServiceAccountRequestSchema,
  GetServiceAccountRequestSchema,
  ListServiceAccountsRequestSchema,
  ServiceAccountSchema,
  UndeleteServiceAccountRequestSchema,
  UpdateServiceAccountRequestSchema,
} from "@/types/proto-es/v1/service_account_service_pb";
import { type User, UserSchema } from "@/types/proto-es/v1/user_service_pb";
import {
  extractServiceAccountId,
  serviceAccountNamePrefix,
} from "@/store/modules/v1/common";
import type {
  AccountFilter,
  AppSliceCreator,
  ServiceAccountSlice,
} from "./types";

export const buildAccountListFilter = (params: AccountFilter) => {
  const filter = [];
  const search = params.query?.trim()?.toLowerCase();
  if (search) {
    filter.push(`(name.contains("${search}") || email.contains("${search}"))`);
  }
  if (params.state === State.DELETED) {
    filter.push(`state == "${State[params.state]}"`);
  }
  return filter.join(" && ");
};

export const ensureServiceAccountFullName = (identifier: string) => {
  const id = extractServiceAccountId(identifier);
  return `${serviceAccountNamePrefix}${id}`;
};

export const serviceAccountToUser = (sa: ServiceAccount): User =>
  createProto(UserSchema, {
    name: `users/${sa.email}`,
    email: sa.email,
    title: sa.title,
    state: sa.state,
    serviceKey: sa.serviceKey,
  });

export const createServiceAccountSlice: AppSliceCreator<
  ServiceAccountSlice
> = (set, get) => ({
  serviceAccountsByName: {},
  serviceAccountRequests: {},

  listServiceAccounts: async (params) => {
    const response =
      await serviceAccountServiceClientConnect.listServiceAccounts(
        createProto(ListServiceAccountsRequestSchema, {
          parent: params.parent,
          pageSize: params.pageSize,
          pageToken: params.pageToken,
          showDeleted: params.showDeleted,
          filter: buildAccountListFilter(params.filter ?? {}),
        })
      );
    set((state) => ({
      serviceAccountsByName: {
        ...state.serviceAccountsByName,
        ...Object.fromEntries(
          response.serviceAccounts.map((sa) => [sa.name, sa])
        ),
      },
    }));
    return {
      serviceAccounts: response.serviceAccounts,
      nextPageToken: response.nextPageToken,
    };
  },

  fetchServiceAccount: async (name, silent = false) => {
    const validName = ensureServiceAccountFullName(name);
    const existing = get().serviceAccountsByName[validName];
    if (existing) return existing;
    const pending = get().serviceAccountRequests[validName];
    if (pending) return pending;
    const request = serviceAccountServiceClientConnect
      .getServiceAccount(
        createProto(GetServiceAccountRequestSchema, { name: validName }),
        { contextValues: createContextValues().set(silentContextKey, silent) }
      )
      .then((sa) => {
        set((state) => {
          const { [validName]: _, ...serviceAccountRequests } =
            state.serviceAccountRequests;
          return {
            serviceAccountsByName: {
              ...state.serviceAccountsByName,
              [sa.name]: sa,
            },
            serviceAccountRequests,
          };
        });
        return sa;
      })
      .catch(() => undefined);
    set((state) => ({
      serviceAccountRequests: {
        ...state.serviceAccountRequests,
        [validName]: request,
      },
    }));
    return request;
  },

  getServiceAccount: (name) => {
    const validName = ensureServiceAccountFullName(name);
    const email = extractServiceAccountId(validName);
    return (
      get().serviceAccountsByName[validName] ??
      createProto(ServiceAccountSchema, {
        name,
        email,
        state: State.ACTIVE,
        title: email.split("@")[0],
      })
    );
  },

  createServiceAccount: async (serviceAccountId, serviceAccount, parent) => {
    const sa = await serviceAccountServiceClientConnect.createServiceAccount(
      createProto(CreateServiceAccountRequestSchema, {
        parent,
        serviceAccountId,
        serviceAccount: createProto(
          ServiceAccountSchema,
          serviceAccount as ServiceAccount
        ),
      })
    );
    set((state) => ({
      serviceAccountsByName: { ...state.serviceAccountsByName, [sa.name]: sa },
    }));
    return sa;
  },

  updateServiceAccount: async (serviceAccount, updateMask) => {
    const sa = await serviceAccountServiceClientConnect.updateServiceAccount(
      createProto(UpdateServiceAccountRequestSchema, {
        serviceAccount: createProto(
          ServiceAccountSchema,
          serviceAccount as ServiceAccount
        ),
        updateMask,
      })
    );
    set((state) => ({
      serviceAccountsByName: { ...state.serviceAccountsByName, [sa.name]: sa },
    }));
    return sa;
  },

  deleteServiceAccount: async (name) => {
    await serviceAccountServiceClientConnect.deleteServiceAccount(
      createProto(DeleteServiceAccountRequestSchema, { name })
    );
    set((state) => {
      const cached = state.serviceAccountsByName[name];
      if (!cached) return {};
      return {
        serviceAccountsByName: {
          ...state.serviceAccountsByName,
          [name]: { ...cached, state: State.DELETED },
        },
      };
    });
  },

  undeleteServiceAccount: async (name) => {
    const sa = await serviceAccountServiceClientConnect.undeleteServiceAccount(
      createProto(UndeleteServiceAccountRequestSchema, { name })
    );
    set((state) => ({
      serviceAccountsByName: { ...state.serviceAccountsByName, [sa.name]: sa },
    }));
    return sa;
  },
});
```

- [ ] **Step 4: Create `workloadIdentity.ts`**

Create the workload identity slice by following `serviceAccount.ts` with these substitutions:

```ts
import { workloadIdentityServiceClientConnect } from "@/connect";
import type { WorkloadIdentity } from "@/types/proto-es/v1/workload_identity_service_pb";
import {
  CreateWorkloadIdentityRequestSchema,
  DeleteWorkloadIdentityRequestSchema,
  GetWorkloadIdentityRequestSchema,
  ListWorkloadIdentitiesRequestSchema,
  UndeleteWorkloadIdentityRequestSchema,
  UpdateWorkloadIdentityRequestSchema,
  WorkloadIdentitySchema,
} from "@/types/proto-es/v1/workload_identity_service_pb";
import {
  extractWorkloadIdentityId,
  workloadIdentityNamePrefix,
} from "@/store/modules/v1/common";
import { buildAccountListFilter } from "./serviceAccount";
import type { AppSliceCreator, WorkloadIdentitySlice } from "./types";
```

Use these helper and conversion functions:

```ts
export const ensureWorkloadIdentityFullName = (identifier: string) => {
  const id = extractWorkloadIdentityId(identifier);
  return `${workloadIdentityNamePrefix}${id}`;
};

export const workloadIdentityToUser = (wi: WorkloadIdentity): User =>
  createProto(UserSchema, {
    name: `users/${wi.email}`,
    email: wi.email,
    title: wi.title,
    state: wi.state,
  });
```

The action names and cache keys must match `WorkloadIdentitySlice` from Task 1:

```ts
workloadIdentitiesByName
workloadIdentityRequests
listWorkloadIdentities
fetchWorkloadIdentity
getWorkloadIdentity
createWorkloadIdentity
updateWorkloadIdentity
deleteWorkloadIdentity
undeleteWorkloadIdentity
```

Use the same delete behavior as service accounts: if a cached resource exists,
replace it with a copy whose `state` is `State.DELETED`.

- [ ] **Step 5: Compose account slices**

In `index.ts`:

```ts
import { createServiceAccountSlice } from "./serviceAccount";
import { createWorkloadIdentitySlice } from "./workloadIdentity";
```

Add to `createAppStore()` after `createGroupSlice`:

```ts
    ...createServiceAccountSlice(...args),
    ...createWorkloadIdentitySlice(...args),
```

- [ ] **Step 6: Run account tests**

Run:

```bash
pnpm --dir frontend exec vitest run frontend/src/react/stores/app/index.test.ts
pnpm --dir frontend type-check
```

Expected: PASS.

- [ ] **Step 7: Commit account slices**

```bash
git add frontend/src/react/stores/app/types.ts frontend/src/react/stores/app/index.ts frontend/src/react/stores/app/serviceAccount.ts frontend/src/react/stores/app/workloadIdentity.ts frontend/src/react/stores/app/index.test.ts
git commit -m "feat(frontend): add account resources to app store"
```

---

### Task 4: Add Identity Provider Slice

**Files:**
- Create: `frontend/src/react/stores/app/identityProvider.ts`
- Modify: `frontend/src/react/stores/app/index.ts`
- Test: `frontend/src/react/stores/app/index.test.ts`

- [ ] **Step 1: Add IDP mocks and fixture**

Extend mocks:

```ts
listIdentityProviders: vi.fn(),
getIdentityProvider: vi.fn(),
createIdentityProvider: vi.fn(),
updateIdentityProvider: vi.fn(),
deleteIdentityProvider: vi.fn(),
```

Add connect client:

```ts
identityProviderServiceClientConnect: {
  listIdentityProviders: mocks.listIdentityProviders,
  getIdentityProvider: mocks.getIdentityProvider,
  createIdentityProvider: mocks.createIdentityProvider,
  updateIdentityProvider: mocks.updateIdentityProvider,
  deleteIdentityProvider: mocks.deleteIdentityProvider,
},
```

Add import and fixture:

```ts
import { IdentityProviderSchema } from "@/types/proto-es/v1/idp_service_pb";

const identityProviderA = createProto(IdentityProviderSchema, {
  name: "idps/google",
  title: "Google",
  domain: "example.com",
});
```

- [ ] **Step 2: Add failing IDP tests**

```ts
test("lists identity providers and replaces the identity provider cache", async () => {
  mocks.listIdentityProviders.mockResolvedValue({
    identityProviders: [identityProviderA],
  });
  const store = createAppStore();
  store.setState({
    identityProvidersByName: {
      "idps/stale": createProto(IdentityProviderSchema, {
        name: "idps/stale",
      }),
    },
  });

  const providers = await store.getState().listIdentityProviders();

  expect(providers).toEqual([identityProviderA]);
  expect(store.getState().identityProviderList()).toEqual([identityProviderA]);
  expect(store.getState().identityProvidersByName).toEqual({
    [identityProviderA.name]: identityProviderA,
  });
});

test("updates identity provider with a field mask based on changed fields", async () => {
  mocks.getIdentityProvider.mockResolvedValue(identityProviderA);
  mocks.updateIdentityProvider.mockResolvedValue({
    ...identityProviderA,
    title: "Google Workspace",
  });
  const store = createAppStore();

  const updated = await store.getState().updateIdentityProvider({
    name: identityProviderA.name,
    title: "Google Workspace",
  });

  expect(updated.title).toBe("Google Workspace");
  expect(mocks.updateIdentityProvider).toHaveBeenCalledWith(
    expect.objectContaining({
      updateMask: { paths: ["title"] },
    })
  );
});
```

Run:

```bash
pnpm --dir frontend exec vitest run frontend/src/react/stores/app/index.test.ts
```

Expected: FAIL because the IDP slice is not implemented.

- [ ] **Step 3: Create `identityProvider.ts`**

Create the slice:

```ts
import { clone, create as createProto } from "@bufbuild/protobuf";
import { createContextValues } from "@connectrpc/connect";
import { isEqual, isUndefined } from "lodash-es";
import { identityProviderServiceClientConnect } from "@/connect";
import { silentContextKey } from "@/connect/context-key";
import type { IdentityProvider } from "@/types/proto-es/v1/idp_service_pb";
import {
  CreateIdentityProviderRequestSchema,
  DeleteIdentityProviderRequestSchema,
  GetIdentityProviderRequestSchema,
  IdentityProviderSchema,
  ListIdentityProvidersRequestSchema,
  UpdateIdentityProviderRequestSchema,
} from "@/types/proto-es/v1/idp_service_pb";
import type { AppSliceCreator, IdentityProviderSlice } from "./types";

const getUpdateMaskFromIdentityProvider = (
  origin: IdentityProvider,
  update: Partial<IdentityProvider>
): string[] => {
  const updateMask: string[] = [];
  if (!isUndefined(update.title) && !isEqual(origin.title, update.title)) {
    updateMask.push("title");
  }
  if (!isUndefined(update.domain) && !isEqual(origin.domain, update.domain)) {
    updateMask.push("domain");
  }
  if (!isUndefined(update.config) && !isEqual(origin.config, update.config)) {
    updateMask.push("config");
  }
  return updateMask;
};

export const createIdentityProviderSlice: AppSliceCreator<
  IdentityProviderSlice
> = (set, get) => ({
  identityProvidersByName: {},
  identityProviderRequests: {},

  identityProviderList: () =>
    Object.values(get().identityProvidersByName),

  listIdentityProviders: async (parent) => {
    const response =
      await identityProviderServiceClientConnect.listIdentityProviders(
        createProto(ListIdentityProvidersRequestSchema, {
          parent: parent ?? "",
        })
      );
    set({
      identityProvidersByName: Object.fromEntries(
        response.identityProviders.map((idp) => [idp.name, idp])
      ),
    });
    return response.identityProviders;
  },

  fetchIdentityProvider: async (name, silent = false) => {
    const existing = get().identityProvidersByName[name];
    if (existing) return existing;
    const pending = get().identityProviderRequests[name];
    if (pending) return pending;
    const request = identityProviderServiceClientConnect
      .getIdentityProvider(
        createProto(GetIdentityProviderRequestSchema, { name }),
        { contextValues: createContextValues().set(silentContextKey, silent) }
      )
      .then((idp) => {
        set((state) => {
          const { [name]: _, ...identityProviderRequests } =
            state.identityProviderRequests;
          return {
            identityProvidersByName: {
              ...state.identityProvidersByName,
              [idp.name]: idp,
            },
            identityProviderRequests,
          };
        });
        return idp;
      })
      .catch(() => undefined);
    set((state) => ({
      identityProviderRequests: {
        ...state.identityProviderRequests,
        [name]: request,
      },
    }));
    return request;
  },

  getIdentityProvider: (name) => get().identityProvidersByName[name],

  createIdentityProvider: async (identityProvider) => {
    const idp =
      await identityProviderServiceClientConnect.createIdentityProvider(
        createProto(CreateIdentityProviderRequestSchema, {
          identityProvider,
          identityProviderId: identityProvider.name,
        })
      );
    set((state) => ({
      identityProvidersByName: {
        ...state.identityProvidersByName,
        [idp.name]: idp,
      },
    }));
    return idp;
  },

  updateIdentityProvider: async (update) => {
    const origin = await get().fetchIdentityProvider(update.name || "");
    if (!origin) {
      throw new Error(`identity provider with name ${update.name} not found`);
    }
    const fullProvider = clone(IdentityProviderSchema, origin);
    if (update.title !== undefined) fullProvider.title = update.title;
    if (update.domain !== undefined) fullProvider.domain = update.domain;
    if (update.type !== undefined) fullProvider.type = update.type;
    if (update.config !== undefined) fullProvider.config = update.config;
    const idp =
      await identityProviderServiceClientConnect.updateIdentityProvider(
        createProto(UpdateIdentityProviderRequestSchema, {
          identityProvider: fullProvider,
          updateMask: {
            paths: getUpdateMaskFromIdentityProvider(origin, update),
          },
        })
      );
    set((state) => ({
      identityProvidersByName: {
        ...state.identityProvidersByName,
        [idp.name]: idp,
      },
    }));
    return idp;
  },

  deleteIdentityProvider: async (name) => {
    await identityProviderServiceClientConnect.deleteIdentityProvider(
      createProto(DeleteIdentityProviderRequestSchema, { name })
    );
    set((state) => {
      const { [name]: _, ...identityProvidersByName } =
        state.identityProvidersByName;
      return { identityProvidersByName };
    });
  },
});
```

- [ ] **Step 4: Compose IDP slice**

In `index.ts`:

```ts
import { createIdentityProviderSlice } from "./identityProvider";
```

Add to `createAppStore()`:

```ts
    ...createIdentityProviderSlice(...args),
```

- [ ] **Step 5: Run IDP tests**

```bash
pnpm --dir frontend exec vitest run frontend/src/react/stores/app/index.test.ts
pnpm --dir frontend type-check
```

Expected: PASS.

- [ ] **Step 6: Commit IDP slice**

```bash
git add frontend/src/react/stores/app/types.ts frontend/src/react/stores/app/index.ts frontend/src/react/stores/app/identityProvider.ts frontend/src/react/stores/app/index.test.ts
git commit -m "feat(frontend): add identity provider resource to app store"
```

---

### Task 5: Add Access Grant Slice

**Files:**
- Create: `frontend/src/react/stores/app/accessGrant.ts`
- Modify: `frontend/src/react/stores/app/index.ts`
- Test: `frontend/src/react/stores/app/index.test.ts`

- [ ] **Step 1: Add access grant mocks and fixture**

Extend mocks:

```ts
getAccessGrant: vi.fn(),
searchMyAccessGrants: vi.fn(),
createAccessGrant: vi.fn(),
listAccessGrants: vi.fn(),
activateAccessGrant: vi.fn(),
revokeAccessGrant: vi.fn(),
```

Add connect client:

```ts
accessGrantServiceClientConnect: {
  getAccessGrant: mocks.getAccessGrant,
  searchMyAccessGrants: mocks.searchMyAccessGrants,
  createAccessGrant: mocks.createAccessGrant,
  listAccessGrants: mocks.listAccessGrants,
  activateAccessGrant: mocks.activateAccessGrant,
  revokeAccessGrant: mocks.revokeAccessGrant,
},
```

Add import and fixture:

```ts
import {
  AccessGrantSchema,
  AccessGrant_Status,
} from "@/types/proto-es/v1/access_grant_service_pb";

const accessGrantA = createProto(AccessGrantSchema, {
  name: "projects/a/accessGrants/ag1",
  title: "Query prod",
  status: AccessGrant_Status.ACTIVE,
});
```

- [ ] **Step 2: Add failing access grant tests**

```ts
test("lists access grants and writes them into cache", async () => {
  mocks.listAccessGrants.mockResolvedValue({
    accessGrants: [accessGrantA],
    nextPageToken: "next-ag",
  });
  const store = createAppStore();

  const result = await store.getState().listAccessGrants({
    parent: "projects/a",
    pageSize: 20,
    filter: { status: ["ACTIVE"] },
  });

  expect(result.accessGrants).toEqual([accessGrantA]);
  expect(result.nextPageToken).toBe("next-ag");
  expect(store.getState().accessGrantsByName[accessGrantA.name]).toBe(
    accessGrantA
  );
});

test("revokeAccessGrant writes returned grant into cache", async () => {
  const revoked = {
    ...accessGrantA,
    status: AccessGrant_Status.REVOKED,
  };
  mocks.revokeAccessGrant.mockResolvedValue(revoked);
  const store = createAppStore();

  const result = await store.getState().revokeAccessGrant(accessGrantA.name);

  expect(result).toBe(revoked);
  expect(store.getState().accessGrantsByName[accessGrantA.name]).toBe(revoked);
});
```

Run:

```bash
pnpm --dir frontend exec vitest run frontend/src/react/stores/app/index.test.ts
```

Expected: FAIL because the access grant slice is not implemented.

- [ ] **Step 3: Create `accessGrant.ts`**

Create the slice:

```ts
import { create as createProto } from "@bufbuild/protobuf";
import { accessGrantServiceClientConnect } from "@/connect";
import type { AccessGrant } from "@/types/proto-es/v1/access_grant_service_pb";
import {
  AccessGrant_Status,
  ActivateAccessGrantRequestSchema,
  CreateAccessGrantRequestSchema,
  GetAccessGrantRequestSchema,
  ListAccessGrantsRequestSchema,
  RevokeAccessGrantRequestSchema,
  SearchMyAccessGrantsRequestSchema,
} from "@/types/proto-es/v1/access_grant_service_pb";
import type {
  AccessGrantFilter,
  AccessGrantSlice,
  AppSliceCreator,
  ListAccessGrantsParams,
} from "./types";

export const buildAccessGrantFilter = (
  filter: AccessGrantFilter | undefined,
  now = new Date()
): string => {
  if (!filter) return "";
  const parts: string[] = [];
  if (filter.name) parts.push(`name == "${filter.name}"`);
  if (filter.status !== undefined && filter.status.length > 0) {
    const statusFilter: string[] = [];
    for (const status of filter.status) {
      switch (status) {
        case "ACTIVE":
          statusFilter.push(
            `(status == "${AccessGrant_Status[AccessGrant_Status.ACTIVE]}" && expire_time > "${now.toISOString()}")`
          );
          break;
        case "EXPIRED":
          statusFilter.push(`expire_time <= "${now.toISOString()}"`);
          break;
        default:
          statusFilter.push(
            `status == "${status as keyof typeof AccessGrant_Status}"`
          );
      }
    }
    parts.push(`(${statusFilter.join(" || ")})`);
  }
  if (filter.statement) {
    parts.push(`query.contains("${filter.statement.trim()}")`);
  }
  if (filter.creator) parts.push(`creator == "${filter.creator}"`);
  if (filter.issue) parts.push(`issue == "${filter.issue}"`);
  if (filter.target) parts.push(`target == "${filter.target}"`);
  if (filter.createdTsAfter !== undefined) {
    parts.push(
      `create_time >= "${new Date(filter.createdTsAfter).toISOString()}"`
    );
  }
  if (filter.createdTsBefore !== undefined) {
    parts.push(
      `create_time <= "${new Date(filter.createdTsBefore).toISOString()}"`
    );
  }
  return parts.join(" && ");
};

const upsertAccessGrants = (
  set: Parameters<AppSliceCreator<AccessGrantSlice>>[0],
  accessGrants: AccessGrant[]
) => {
  set((state) => ({
    accessGrantsByName: {
      ...state.accessGrantsByName,
      ...Object.fromEntries(accessGrants.map((grant) => [grant.name, grant])),
    },
  }));
};

export const createAccessGrantSlice: AppSliceCreator<AccessGrantSlice> = (
  set,
  get
) => ({
  accessGrantsByName: {},
  accessGrantRequests: {},

  fetchAccessGrant: async (name) => {
    const existing = get().accessGrantsByName[name];
    if (existing) return existing;
    const pending = get().accessGrantRequests[name];
    if (pending) return pending;
    const request = accessGrantServiceClientConnect
      .getAccessGrant(createProto(GetAccessGrantRequestSchema, { name }))
      .then((grant) => {
        set((state) => {
          const { [name]: _, ...accessGrantRequests } =
            state.accessGrantRequests;
          return {
            accessGrantsByName: {
              ...state.accessGrantsByName,
              [grant.name]: grant,
            },
            accessGrantRequests,
          };
        });
        return grant;
      })
      .catch(() => undefined);
    set((state) => ({
      accessGrantRequests: { ...state.accessGrantRequests, [name]: request },
    }));
    return request;
  },

  searchMyAccessGrants: async (params) => {
    const response =
      await accessGrantServiceClientConnect.searchMyAccessGrants(
        createAccessGrantListRequest(SearchMyAccessGrantsRequestSchema, params)
      );
    upsertAccessGrants(set, response.accessGrants);
    return {
      accessGrants: response.accessGrants,
      nextPageToken: response.nextPageToken,
    };
  },

  listAccessGrants: async (params) => {
    const response = await accessGrantServiceClientConnect.listAccessGrants(
      createAccessGrantListRequest(ListAccessGrantsRequestSchema, params)
    );
    upsertAccessGrants(set, response.accessGrants);
    return {
      accessGrants: response.accessGrants,
      nextPageToken: response.nextPageToken,
    };
  },

  createAccessGrant: async (parent, accessGrant) => {
    const grant = await accessGrantServiceClientConnect.createAccessGrant(
      createProto(CreateAccessGrantRequestSchema, { parent, accessGrant })
    );
    upsertAccessGrants(set, [grant]);
    return grant;
  },

  activateAccessGrant: async (name) => {
    const grant = await accessGrantServiceClientConnect.activateAccessGrant(
      createProto(ActivateAccessGrantRequestSchema, { name })
    );
    upsertAccessGrants(set, [grant]);
    return grant;
  },

  revokeAccessGrant: async (name) => {
    const grant = await accessGrantServiceClientConnect.revokeAccessGrant(
      createProto(RevokeAccessGrantRequestSchema, { name })
    );
    upsertAccessGrants(set, [grant]);
    return grant;
  },
});

function createAccessGrantListRequest<T>(
  schema: T,
  params: ListAccessGrantsParams
) {
  return createProto(schema as never, {
    parent: params.parent,
    filter: buildAccessGrantFilter(params.filter),
    pageSize: params.pageSize ?? 0,
    pageToken: params.pageToken ?? "",
    orderBy: params.orderBy ?? "",
  } as never);
}
```

If TypeScript rejects the generic helper because `createProto()` needs a concrete schema type, replace `createAccessGrantListRequest()` with two explicit request constructions, one for `SearchMyAccessGrantsRequestSchema` and one for `ListAccessGrantsRequestSchema`.

- [ ] **Step 4: Compose access grant slice**

In `index.ts`:

```ts
import { createAccessGrantSlice } from "./accessGrant";
```

Add to `createAppStore()`:

```ts
    ...createAccessGrantSlice(...args),
```

- [ ] **Step 5: Run access grant tests**

```bash
pnpm --dir frontend exec vitest run frontend/src/react/stores/app/index.test.ts
pnpm --dir frontend type-check
```

Expected: PASS. If the generic request helper fails type-check, apply the explicit-request fallback described in Step 3 and rerun.

- [ ] **Step 6: Commit access grant slice**

```bash
git add frontend/src/react/stores/app/types.ts frontend/src/react/stores/app/index.ts frontend/src/react/stores/app/accessGrant.ts frontend/src/react/stores/app/index.test.ts
git commit -m "feat(frontend): add access grant resource to app store"
```

---

### Task 6: Add React Hook Facades for Migrated Resources

**Files:**
- Modify: `frontend/src/react/hooks/useAppState.ts`

- [ ] **Step 1: Add resource hooks**

Add these exports near the existing project/instance resource hooks:

```ts
export function useGroups() {
  const groupsByName = useAppStore((state) => state.groupsByName);
  return Object.values(groupsByName).sort((a, b) =>
    a.name.localeCompare(b.name)
  );
}

export function useGroupByIdentifier(id: string) {
  return useAppStore((state) => state.getGroupByIdentifier(id));
}

export function useServiceAccount(name: string) {
  return useAppStore((state) => state.getServiceAccount(name));
}

export function useWorkloadIdentity(name: string) {
  return useAppStore((state) => state.getWorkloadIdentity(name));
}

export function useIdentityProviderList() {
  return useAppStore((state) => state.identityProviderList());
}

export function useIdentityProvider(name: string) {
  return useAppStore((state) => state.identityProvidersByName[name]);
}

export function useAccessGrant(name: string) {
  return useAppStore((state) => state.accessGrantsByName[name]);
}
```

- [ ] **Step 2: Run type-check**

```bash
pnpm --dir frontend type-check
```

Expected: PASS.

- [ ] **Step 3: Commit hooks**

```bash
git add frontend/src/react/hooks/useAppState.ts
git commit -m "feat(frontend): expose phase one app resource hooks"
```

---

### Task 7: Migrate Settings and Account React Consumers

**Files:**
- Modify: `frontend/src/react/pages/settings/GroupsPage.tsx`
- Modify: `frontend/src/react/pages/settings/UsersPage.tsx`
- Modify: `frontend/src/react/pages/settings/ServiceAccountsPage.tsx`
- Modify: `frontend/src/react/pages/settings/WorkloadIdentitiesPage.tsx`
- Modify: `frontend/src/react/components/AccountMultiSelect.tsx`
- Modify: `frontend/src/react/components/CreateWorkloadIdentitySheet.tsx`

- [ ] **Step 1: Replace store imports**

Replace imports from `@/store` or `@/store/modules/...` for these symbols:

```ts
useGroupStore
useServiceAccountStore
useWorkloadIdentityStore
```

Use:

```ts
import { useAppStore } from "@/react/stores/app";
```

If a file only needs conversion helpers, import those helpers directly from the new app slice:

```ts
import {
  serviceAccountToUser,
} from "@/react/stores/app/serviceAccount";
import {
  workloadIdentityToUser,
} from "@/react/stores/app/workloadIdentity";
```

- [ ] **Step 2: Convert component-local store reads**

Convert this pattern:

```ts
const groupStore = useGroupStore();
```

to selector-based actions:

```ts
const listGroups = useAppStore((state) => state.listGroups);
const createGroup = useAppStore((state) => state.createGroup);
const updateGroup = useAppStore((state) => state.updateGroup);
const deleteGroup = useAppStore((state) => state.deleteGroup);
const groups = useAppStore((state) =>
  Object.values(state.groupsByName).sort((a, b) => a.name.localeCompare(b.name))
);
```

Convert service account actions:

```ts
const listServiceAccounts = useAppStore(
  (state) => state.listServiceAccounts
);
const createServiceAccount = useAppStore(
  (state) => state.createServiceAccount
);
const updateServiceAccount = useAppStore(
  (state) => state.updateServiceAccount
);
const deleteServiceAccount = useAppStore(
  (state) => state.deleteServiceAccount
);
const undeleteServiceAccount = useAppStore(
  (state) => state.undeleteServiceAccount
);
```

Convert workload identity actions:

```ts
const listWorkloadIdentities = useAppStore(
  (state) => state.listWorkloadIdentities
);
const createWorkloadIdentity = useAppStore(
  (state) => state.createWorkloadIdentity
);
const updateWorkloadIdentity = useAppStore(
  (state) => state.updateWorkloadIdentity
);
const deleteWorkloadIdentity = useAppStore(
  (state) => state.deleteWorkloadIdentity
);
const undeleteWorkloadIdentity = useAppStore(
  (state) => state.undeleteWorkloadIdentity
);
```

- [ ] **Step 3: Preserve method names at call sites**

When a page currently calls:

```ts
groupStore.fetchGroupList(params)
```

replace with:

```ts
listGroups(params)
```

When it calls:

```ts
serviceAccountStore.listServiceAccounts(params)
workloadIdentityStore.listWorkloadIdentities(params)
```

keep the same params object and call the selected Zustand action:

```ts
listServiceAccounts(params)
listWorkloadIdentities(params)
```

When it calls:

```ts
serviceAccountStore.getServiceAccount(name)
workloadIdentityStore.getWorkloadIdentity(name)
groupStore.getGroupByIdentifier(name)
```

use either a selector in render code or `useAppStore.getState()` in non-render utility callbacks:

```ts
const account = useAppStore((state) => state.getServiceAccount(name));
```

```ts
const account = useAppStore.getState().getServiceAccount(name);
```

- [ ] **Step 4: Run focused type-check**

```bash
pnpm --dir frontend type-check
```

Expected: PASS.

- [ ] **Step 5: Run focused tests for touched settings pages**

Run existing tests if present:

```bash
pnpm --dir frontend exec vitest run frontend/src/react/pages/settings
pnpm --dir frontend exec vitest run frontend/src/react/components/AccountMultiSelect.test.tsx frontend/src/react/components/CreateWorkloadIdentitySheet.test.tsx
```

Expected: PASS for existing files. If a listed test file does not exist, rerun `pnpm --dir frontend test` in Task 10 instead of creating unrelated tests in this task.

- [ ] **Step 6: Commit settings/account consumer migration**

```bash
git add frontend/src/react/pages/settings/GroupsPage.tsx frontend/src/react/pages/settings/UsersPage.tsx frontend/src/react/pages/settings/ServiceAccountsPage.tsx frontend/src/react/pages/settings/WorkloadIdentitiesPage.tsx frontend/src/react/components/AccountMultiSelect.tsx frontend/src/react/components/CreateWorkloadIdentitySheet.tsx
git commit -m "refactor(frontend): use app store for account resources"
```

---

### Task 8: Migrate IDP and Access Grant React Consumers

**Files:**
- Modify: `frontend/src/react/pages/auth/SigninPage.tsx`
- Modify: `frontend/src/react/pages/settings/IDPsPage.tsx`
- Modify: `frontend/src/react/pages/settings/IDPDetailPage.tsx`
- Modify: `frontend/src/react/pages/settings/general/AccountSection.tsx`
- Modify: `frontend/src/react/pages/project/ProjectAccessGrantsPage.tsx`
- Modify: `frontend/src/react/pages/project/issue-detail/components/IssueDetailAccessGrantDetails.tsx`
- Modify: `frontend/src/react/components/sql-editor/AccessPane.tsx`

- [ ] **Step 1: Replace IDP store imports**

Replace:

```ts
import { useIdentityProviderStore } from "@/store";
import { useIdentityProviderStore } from "@/store/modules/idp";
```

with:

```ts
import { useAppStore } from "@/react/stores/app";
```

Select actions and state:

```ts
const identityProviders = useAppStore((state) =>
  state.identityProviderList()
);
const listIdentityProviders = useAppStore(
  (state) => state.listIdentityProviders
);
const fetchIdentityProvider = useAppStore(
  (state) => state.fetchIdentityProvider
);
const createIdentityProvider = useAppStore(
  (state) => state.createIdentityProvider
);
const updateIdentityProvider = useAppStore(
  (state) => state.updateIdentityProvider
);
const deleteIdentityProvider = useAppStore(
  (state) => state.deleteIdentityProvider
);
```

Map old calls:

```ts
identityProviderStore.fetchIdentityProviderList(parent)
```

to:

```ts
listIdentityProviders(parent)
```

Map:

```ts
identityProviderStore.patchIdentityProvider(update)
```

to:

```ts
updateIdentityProvider(update)
```

- [ ] **Step 2: Replace access grant store imports**

Replace:

```ts
import { useAccessGrantStore } from "@/store";
```

with:

```ts
import { useAppStore } from "@/react/stores/app";
```

Select actions:

```ts
const fetchAccessGrant = useAppStore((state) => state.fetchAccessGrant);
const searchMyAccessGrants = useAppStore(
  (state) => state.searchMyAccessGrants
);
const listAccessGrants = useAppStore((state) => state.listAccessGrants);
const createAccessGrant = useAppStore((state) => state.createAccessGrant);
const activateAccessGrant = useAppStore((state) => state.activateAccessGrant);
const revokeAccessGrant = useAppStore((state) => state.revokeAccessGrant);
```

Map old calls:

```ts
accessGrantStore.getAccessGrant(name)
```

to:

```ts
fetchAccessGrant(name)
```

Map the list/search/create/activate/revoke calls to the same action names on
`useAppStore`.

- [ ] **Step 3: Run focused type-check**

```bash
pnpm --dir frontend type-check
```

Expected: PASS.

- [ ] **Step 4: Run focused tests for touched surfaces**

```bash
pnpm --dir frontend exec vitest run frontend/src/react/pages/auth/SigninPage.test.tsx frontend/src/react/components/sql-editor/AccessPane.test.tsx
pnpm --dir frontend exec vitest run frontend/src/react/pages/settings frontend/src/react/pages/project
```

Expected: PASS for existing tests. If the broad project/settings command is too slow, run the specific test files touched by this task plus full `pnpm --dir frontend test` in Task 10.

- [ ] **Step 5: Commit IDP and access consumer migration**

```bash
git add frontend/src/react/pages/auth/SigninPage.tsx frontend/src/react/pages/settings/IDPsPage.tsx frontend/src/react/pages/settings/IDPDetailPage.tsx frontend/src/react/pages/settings/general/AccountSection.tsx frontend/src/react/pages/project/ProjectAccessGrantsPage.tsx frontend/src/react/pages/project/issue-detail/components/IssueDetailAccessGrantDetails.tsx frontend/src/react/components/sql-editor/AccessPane.tsx
git commit -m "refactor(frontend): use app store for idp and access grants"
```

---

### Task 9: Add No-Legacy Store Guard

**Files:**
- Modify: `frontend/src/react/no-legacy-vue-deps.test.ts`

- [ ] **Step 1: Add phase 1 resource store guard**

Append this test inside the existing `describe` block:

```ts
  test("phase one resource consumers do not import migrated Pinia stores", () => {
    const phaseOneFiles = [
      "./pages/settings/GroupsPage.tsx",
      "./pages/settings/UsersPage.tsx",
      "./pages/settings/ServiceAccountsPage.tsx",
      "./pages/settings/WorkloadIdentitiesPage.tsx",
      "./pages/auth/SigninPage.tsx",
      "./pages/settings/IDPsPage.tsx",
      "./pages/settings/IDPDetailPage.tsx",
      "./pages/settings/general/AccountSection.tsx",
      "./pages/project/ProjectAccessGrantsPage.tsx",
      "./pages/project/issue-detail/components/IssueDetailAccessGrantDetails.tsx",
      "./components/sql-editor/AccessPane.tsx",
      "./components/AccountMultiSelect.tsx",
      "./components/CreateWorkloadIdentitySheet.tsx",
    ];
    const bannedImports = [
      "useGroupStore",
      "useServiceAccountStore",
      "useWorkloadIdentityStore",
      "useIdentityProviderStore",
      "useAccessGrantStore",
      "@/store/modules/v1/group",
      "@/store/modules/serviceAccount",
      "@/store/modules/workloadIdentity",
      "@/store/modules/idp",
      "@/store/modules/accessGrant",
    ];
    const violations: string[] = [];
    for (const file of phaseOneFiles) {
      const source = sources[file];
      if (!source) {
        violations.push(`${file}: missing from no-legacy source glob`);
        continue;
      }
      for (const bannedImport of bannedImports) {
        if (source.includes(bannedImport)) {
          violations.push(`${file}: ${bannedImport}`);
        }
      }
    }
    expect(violations).toEqual([]);
  });
```

If any listed file is not covered by the existing `import.meta.glob`, extend the
glob at the top of the file to include it.

- [ ] **Step 2: Run the guard**

```bash
pnpm --dir frontend exec vitest run frontend/src/react/no-legacy-vue-deps.test.ts
```

Expected: PASS.

- [ ] **Step 3: Commit guard**

```bash
git add frontend/src/react/no-legacy-vue-deps.test.ts
git commit -m "test(frontend): guard phase one resource store migration"
```

---

### Task 10: Full Verification and Cleanup

**Files:**
- Any files changed by `pnpm --dir frontend fix`

- [ ] **Step 1: Run frontend fix**

```bash
pnpm --dir frontend fix
```

Expected: command exits 0. It may format or organize imports.

- [ ] **Step 2: Run frontend static check**

```bash
pnpm --dir frontend check
```

Expected: command exits 0.

- [ ] **Step 3: Run React type-check**

```bash
pnpm --dir frontend type-check
```

Expected: command exits 0.

- [ ] **Step 4: Run focused store and guard tests**

```bash
pnpm --dir frontend exec vitest run frontend/src/react/stores/app/index.test.ts frontend/src/react/no-legacy-vue-deps.test.ts
```

Expected: command exits 0.

- [ ] **Step 5: Run full frontend tests**

```bash
pnpm --dir frontend test
```

Expected: command exits 0. Known Vite dynamic-import warnings from SQL Editor store tests may appear; failures must be fixed.

- [ ] **Step 6: Run whitespace check**

```bash
git diff --check
```

Expected: command exits 0.

- [ ] **Step 7: Inspect remaining migrated-store imports**

Run:

```bash
rg -n "use(Group|ServiceAccount|WorkloadIdentity|IdentityProvider|AccessGrant)Store|@/store/modules/(v1/group|serviceAccount|workloadIdentity|idp|accessGrant)" frontend/src/react
```

Expected: no matches in the phase 1 migrated files. Matches in tests that mock legacy stores or in out-of-scope files must be reviewed individually and documented in the PR description.

- [ ] **Step 8: Commit final fixes**

If `frontend fix` or verification changes files:

```bash
git add frontend/src
git commit -m "fix(frontend): clean up phase one resource migration"
```

If there are no additional changes, do not create an empty commit.

---

## Plan Self-Review

- Spec coverage: This plan implements the five approved resources, switches React consumers, adds no-legacy guards, and leaves Pinia deletion out of scope.
- Placeholder scan: No implementation task relies on unspecified future work. Where a TypeScript generic helper may not type-check, the fallback is explicit and local to that step.
- Type consistency: Slice method names match the `types.ts` contracts and the consumer migration mapping.
- Scope check: `ProjectWebhook`, `AuditLog`, lifecycle resources, governance resources, and database metadata resources are excluded from this phase.

# React Zustand Resource Store Migration Phase 2 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Move React consumers of User, Role, Release, Revision, Changelog, and ProjectWebhook protobuf resources from legacy Pinia stores to the React Zustand app store.

**Architecture:** Add one focused app-store slice per resource under `frontend/src/react/stores/app/`, register them in `createAppStore`, expose typed actions/selectors in `types.ts`, and migrate React consumers to `useAppStore(selector)`. Keep legacy Pinia stores for Vue consumers, and add a guard test so React cannot re-import the migrated stores.

**Tech Stack:** React, TypeScript, Zustand, Connect RPC generated clients, Vitest, Biome/ESLint.

---

## File Structure

Create:

- `frontend/src/react/stores/app/user.ts`: user cache, list/get/batch/mutation actions, identifier helpers.
- `frontend/src/react/stores/app/role.ts`: role list cache and role mutation actions.
- `frontend/src/react/stores/app/release.ts`: release cache, project list, get/update/delete/undelete actions.
- `frontend/src/react/stores/app/revision.ts`: revision cache, database list/all/get/delete actions.
- `frontend/src/react/stores/app/changelog.ts`: changelog cache keyed by name and view, database list cache, previous changelog lookup.
- `frontend/src/react/stores/app/projectWebhook.ts`: project webhook helper and mutation actions.

Modify:

- `frontend/src/react/stores/app/types.ts`: add resource filter/params and slice types.
- `frontend/src/react/stores/app/index.ts`: import and compose the new slices.
- `frontend/src/react/stores/app/index.test.ts`: add client mocks and focused slice tests.
- `frontend/src/react/hooks/useAppState.ts`: expose small hook facades for migrated resources.
- `frontend/src/react/no-legacy-vue-deps.test.ts`: add the Phase 2 legacy-store guard.
- React consumers currently importing `useUserStore`, `useRoleStore`, `useReleaseStore`, `useRevisionStore`, `useChangelogStore`, and `useProjectWebhookV1Store`.
- Tests for touched consumers that mock the legacy stores.

---

### Task 1: Add User and Role App Store Slices

**Files:**
- Create: `frontend/src/react/stores/app/user.ts`
- Create: `frontend/src/react/stores/app/role.ts`
- Modify: `frontend/src/react/stores/app/types.ts`
- Modify: `frontend/src/react/stores/app/index.ts`
- Modify: `frontend/src/react/stores/app/index.test.ts`

- [ ] **Step 1: Extend app-store types for User and Role**

Add imports in `frontend/src/react/stores/app/types.ts` if they are not already present:

```ts
import type { UpdateUserRequest, User } from "@/types/proto-es/v1/user_service_pb";
import type { Role } from "@/types/proto-es/v1/role_service_pb";
```

Add these types near the existing resource filter types:

```ts
export type UserFilter = {
  query?: string;
  project?: string;
  state?: State;
};

export type ListUsersParams = {
  pageSize: number;
  pageToken?: string;
  filter?: UserFilter;
  showDeleted?: boolean;
};
```

Add these slice types before `NotificationSlice`:

```ts
export type UserSlice = {
  usersByName: Record<string, User>;
  userRequests: Record<string, Promise<User | undefined>>;
  listUsers: (
    params: ListUsersParams
  ) => Promise<{ users: User[]; nextPageToken: string }>;
  fetchUser: (name: string, silent?: boolean) => Promise<User | undefined>;
  batchGetOrFetchUsers: (names: string[]) => Promise<User[]>;
  getOrFetchUserByIdentifier: (params: {
    identifier: string;
    silent?: boolean;
    fallback?: boolean;
  }) => Promise<User>;
  getUserByIdentifier: (identifier: string) => User | undefined;
  createUser: (user: User) => Promise<User>;
  updateUser: (request: UpdateUserRequest) => Promise<User>;
  updateEmail: (oldEmail: string, newEmail: string) => Promise<User>;
  archiveUser: (name: string) => Promise<void>;
  restoreUser: (name: string) => Promise<User>;
};

export type RoleSlice = {
  roleList: Role[];
  listRoles: () => Promise<Role[]>;
  getRoleByName: (name: string) => Role | undefined;
  upsertRole: (role: Role) => Promise<Role>;
  deleteRole: (role: Role) => Promise<void>;
};
```

Extend `AppStoreState`:

```ts
  AccessGrantSlice &
  UserSlice &
  RoleSlice &
  NotificationSlice &
```

- [ ] **Step 2: Create the user slice**

Create `frontend/src/react/stores/app/user.ts`:

```ts
import { create as createProto } from "@bufbuild/protobuf";
import { createContextValues } from "@connectrpc/connect";
import { uniq } from "lodash-es";
import { userServiceClientConnect } from "@/connect";
import { silentContextKey } from "@/connect/context-key";
import {
  allUsersUser,
  isValidProjectName,
  isValidUserName,
  unknownUser,
  userBindingPrefix,
} from "@/types";
import { State } from "@/types/proto-es/v1/common_pb";
import {
  BatchGetUsersRequestSchema,
  CreateUserRequestSchema,
  DeleteUserRequestSchema,
  GetUserRequestSchema,
  ListUsersRequestSchema,
  UndeleteUserRequestSchema,
  UpdateUserRequestSchema,
  type User,
} from "@/types/proto-es/v1/user_service_pb";
import { ensureUserFullName } from "@/utils";
import { isValidProjectName as isValidReactProjectName } from "@/react/lib/resourceName";
import { type AppSliceCreator, type UserFilter, type UserSlice } from "./types";

export const extractUserEmail = (identifier: string) => {
  const matches = identifier.match(/^(?:user:|users\/)(.+)$/);
  return matches?.[1] ?? identifier;
};

export const buildUserFilter = (params: UserFilter) => {
  const filter: string[] = [];
  const search = params.query?.trim()?.toLowerCase();
  if (search) {
    filter.push(`(name.contains("${search}") || email.contains("${search}"))`);
  }
  const project = params.project;
  if (isValidProjectName(project) || isValidReactProjectName(project)) {
    filter.push(`project == "${project}"`);
  }
  if (params.state === State.DELETED) {
    filter.push(`state == "${State[params.state]}"`);
  }
  return filter.join(" && ");
};

const upsertUsers = (
  set: Parameters<AppSliceCreator<UserSlice>>[0],
  users: User[]
) => {
  set((state) => ({
    usersByName: {
      ...state.usersByName,
      ...Object.fromEntries(users.map((user) => [user.name, user])),
    },
  }));
};

export const createUserSlice: AppSliceCreator<UserSlice> = (set, get) => ({
  usersByName: { [allUsersUser().name]: allUsersUser() },
  userRequests: {},

  listUsers: async (params) => {
    if (!get().hasWorkspacePermission("bb.users.list")) {
      return { users: [], nextPageToken: "" };
    }
    const showDeleted =
      params.showDeleted ?? params.filter?.state === State.DELETED;
    const response = await userServiceClientConnect.listUsers(
      createProto(ListUsersRequestSchema, {
        pageSize: params.pageSize,
        pageToken: params.pageToken,
        filter: buildUserFilter(params.filter ?? {}),
        showDeleted,
      })
    );
    upsertUsers(set, response.users);
    return { users: response.users, nextPageToken: response.nextPageToken };
  },

  fetchUser: async (name, silent = false) => {
    const validName = ensureUserFullName(name);
    const existing = get().usersByName[validName];
    if (existing) return existing;
    const pending = get().userRequests[validName];
    if (pending) return pending;
    const request = userServiceClientConnect
      .getUser(createProto(GetUserRequestSchema, { name: validName }), {
        contextValues: createContextValues().set(silentContextKey, silent),
      })
      .then((user) => {
        set((state) => {
          const { [validName]: _, ...userRequests } = state.userRequests;
          return {
            usersByName: { ...state.usersByName, [user.name]: user },
            userRequests,
          };
        });
        return user;
      })
      .catch(() => {
        set((state) => {
          const { [validName]: _, ...userRequests } = state.userRequests;
          return { userRequests };
        });
        return undefined;
      });
    set((state) => ({
      userRequests: { ...state.userRequests, [validName]: request },
    }));
    return request;
  },

  batchGetOrFetchUsers: async (names) => {
    const validNames = uniq(names).filter(
      (name) =>
        Boolean(name) &&
        (name.startsWith("users/") || name.startsWith(userBindingPrefix))
    );
    const missing = validNames
      .map(ensureUserFullName)
      .filter((name) => !get().usersByName[name]);
    if (missing.length > 0) {
      const response = await userServiceClientConnect.batchGetUsers(
        createProto(BatchGetUsersRequestSchema, { names: missing }),
        { contextValues: createContextValues().set(silentContextKey, true) }
      );
      upsertUsers(set, response.users);
    }
    return validNames.map((name) => get().getUserByIdentifier(name) ?? unknownUser(name));
  },

  getOrFetchUserByIdentifier: async ({
    identifier,
    silent = true,
    fallback = true,
  }) => {
    const cached = get().getUserByIdentifier(identifier);
    if (cached) return cached;
    const fullname = ensureUserFullName(identifier);
    if (!isValidUserName(fullname)) {
      return unknownUser();
    }
    const user = await get().fetchUser(fullname, silent);
    if (user) return user;
    return unknownUser(fallback ? fullname : "");
  },

  getUserByIdentifier: (identifier) => {
    const email = extractUserEmail(identifier);
    if (Number.isNaN(Number(email))) {
      return Object.values(get().usersByName).find((user) => user.email === email);
    }
    return get().usersByName[ensureUserFullName(identifier)];
  },

  createUser: async (user) => {
    const response = await userServiceClientConnect.createUser(
      createProto(CreateUserRequestSchema, { user })
    );
    upsertUsers(set, [response]);
    return response;
  },

  updateUser: async (request) => {
    const name = request.user?.name || "";
    const origin = await get().getOrFetchUserByIdentifier({
      identifier: name,
      fallback: false,
    });
    if (!isValidUserName(origin.name)) {
      throw new Error(`user with name ${name} not found`);
    }
    const response = await userServiceClientConnect.updateUser(
      createProto(UpdateUserRequestSchema, {
        user: request.user,
        updateMask: request.updateMask,
        otpCode: request.otpCode,
        regenerateTempMfaSecret: request.regenerateTempMfaSecret,
        regenerateRecoveryCodes: request.regenerateRecoveryCodes,
      })
    );
    upsertUsers(set, [response]);
    return response;
  },

  updateEmail: async (oldEmail, newEmail) => {
    const origin = await get().getOrFetchUserByIdentifier({
      identifier: oldEmail,
      fallback: false,
    });
    if (!isValidUserName(origin.name)) {
      throw new Error(`user with email ${oldEmail} not found`);
    }
    const response = await userServiceClientConnect.updateEmail({
      name: `users/${oldEmail}`,
      email: newEmail,
    });
    set((state) => {
      const { [origin.name]: _, ...usersByName } = state.usersByName;
      return { usersByName: { ...usersByName, [response.name]: response } };
    });
    return response;
  },

  archiveUser: async (name) => {
    const validName = ensureUserFullName(name);
    await userServiceClientConnect.deleteUser(
      createProto(DeleteUserRequestSchema, { name: validName })
    );
    set((state) => {
      const cached = state.usersByName[validName];
      if (!cached) return {};
      return {
        usersByName: {
          ...state.usersByName,
          [validName]: { ...cached, state: State.DELETED },
        },
      };
    });
  },

  restoreUser: async (name) => {
    const response = await userServiceClientConnect.undeleteUser(
      createProto(UndeleteUserRequestSchema, { name: ensureUserFullName(name) })
    );
    upsertUsers(set, [response]);
    return response;
  },
});
```

- [ ] **Step 3: Create the role slice**

Create `frontend/src/react/stores/app/role.ts`:

```ts
import { create as createProto } from "@bufbuild/protobuf";
import { roleServiceClientConnect } from "@/connect";
import type { Role } from "@/types/proto-es/v1/role_service_pb";
import {
  DeleteRoleRequestSchema,
  ListRolesRequestSchema,
  UpdateRoleRequestSchema,
} from "@/types/proto-es/v1/role_service_pb";
import type { AppSliceCreator, RoleSlice } from "./types";

export const createRoleSlice: AppSliceCreator<RoleSlice> = (set, get) => ({
  roleList: [],

  listRoles: async () => {
    const response = await roleServiceClientConnect.listRoles(
      createProto(ListRolesRequestSchema, {})
    );
    set({ roleList: response.roles });
    return response.roles;
  },

  getRoleByName: (name) => get().roleList.find((role) => role.name === name),

  upsertRole: async (role) => {
    const response = await roleServiceClientConnect.updateRole(
      createProto(UpdateRoleRequestSchema, {
        role,
        updateMask: { paths: ["title", "description", "permissions"] },
        allowMissing: true,
      })
    );
    set((state) => {
      const roleList = [...state.roleList];
      const index = roleList.findIndex((item: Role) => item.name === response.name);
      if (index >= 0) {
        roleList.splice(index, 1, response);
      } else {
        roleList.push(response);
      }
      return { roleList };
    });
    return response;
  },

  deleteRole: async (role) => {
    await roleServiceClientConnect.deleteRole(
      createProto(DeleteRoleRequestSchema, { name: role.name })
    );
    set((state) => ({
      roleList: state.roleList.filter((item) => item.name !== role.name),
    }));
  },
});
```

- [ ] **Step 4: Register user and role slices**

Modify `frontend/src/react/stores/app/index.ts`:

```ts
import { createRoleSlice } from "./role";
import { createUserSlice } from "./user";
```

Add them before notification/preferences:

```ts
    ...createAccessGrantSlice(...args),
    ...createUserSlice(...args),
    ...createRoleSlice(...args),
    ...createNotificationSlice(...args),
```

- [ ] **Step 5: Add focused app-store tests for User and Role**

Modify `frontend/src/react/stores/app/index.test.ts` mocks:

```ts
  listUsers: vi.fn(),
  getUser: vi.fn(),
  batchGetUsers: vi.fn(),
  createUser: vi.fn(),
  updateUser: vi.fn(),
  updateEmail: vi.fn(),
  deleteUser: vi.fn(),
  undeleteUser: vi.fn(),
  updateRole: vi.fn(),
  deleteRole: vi.fn(),
```

Extend the connect mock:

```ts
  roleServiceClientConnect: {
    listRoles: mocks.listRoles,
    updateRole: mocks.updateRole,
    deleteRole: mocks.deleteRole,
  },
  userServiceClientConnect: {
    getCurrentUser: mocks.getCurrentUser,
    listUsers: mocks.listUsers,
    getUser: mocks.getUser,
    batchGetUsers: mocks.batchGetUsers,
    createUser: mocks.createUser,
    updateUser: mocks.updateUser,
    updateEmail: mocks.updateEmail,
    deleteUser: mocks.deleteUser,
    undeleteUser: mocks.undeleteUser,
  },
```

Add tests:

```ts
test("lists and caches users", async () => {
  const store = createAppStore();
  mocks.listUsers.mockResolvedValue({ users: [user], nextPageToken: "next" });
  const result = await store.getState().listUsers({ pageSize: 10 });
  expect(result).toEqual({ users: [user], nextPageToken: "next" });
  expect(store.getState().getUserByIdentifier(user.name)).toEqual(user);
});

test("batch fetches missing users and returns unknown fallback", async () => {
  const store = createAppStore();
  mocks.batchGetUsers.mockResolvedValue({ users: [user] });
  const result = await store
    .getState()
    .batchGetOrFetchUsers([user.name, "users/missing@example.com"]);
  expect(result[0]).toEqual(user);
  expect(result[1].name).toBe("users/missing@example.com");
});

test("updates user email and removes old cache key", async () => {
  const store = createAppStore();
  mocks.getUser.mockResolvedValue(user);
  const updated = createProto(UserSchema, {
    ...user,
    name: "users/new@example.com",
    email: "new@example.com",
  });
  mocks.updateEmail.mockResolvedValue(updated);
  await store.getState().fetchUser(user.name);
  await store.getState().updateEmail(user.email, updated.email);
  expect(store.getState().usersByName[user.name]).toBeUndefined();
  expect(store.getState().usersByName[updated.name]).toEqual(updated);
});

test("lists, upserts, and deletes roles", async () => {
  const store = createAppStore();
  const role = createProto(RoleSchema, {
    name: "roles/PROJECT_OWNER",
    title: "Project Owner",
  });
  mocks.listRoles.mockResolvedValue({ roles: [role] });
  await store.getState().listRoles();
  expect(store.getState().getRoleByName(role.name)).toEqual(role);
  const updated = createProto(RoleSchema, { ...role, title: "Owner" });
  mocks.updateRole.mockResolvedValue(updated);
  await store.getState().upsertRole(updated);
  expect(store.getState().getRoleByName(role.name)?.title).toBe("Owner");
  mocks.deleteRole.mockResolvedValue({});
  await store.getState().deleteRole(updated);
  expect(store.getState().getRoleByName(role.name)).toBeUndefined();
});
```

- [ ] **Step 6: Run focused app-store tests**

Run:

```bash
pnpm --dir frontend exec vitest run src/react/stores/app/index.test.ts
```

Expected: tests compile and pass. If `User` fallback typing fails, apply the fallback correction from Step 2 and rerun.

- [ ] **Step 7: Commit user and role slices**

```bash
git add frontend/src/react/stores/app/user.ts frontend/src/react/stores/app/role.ts frontend/src/react/stores/app/types.ts frontend/src/react/stores/app/index.ts frontend/src/react/stores/app/index.test.ts
git commit -m "feat(frontend): add user and role app stores"
```

---

### Task 2: Add Release, Revision, Changelog, and ProjectWebhook Slices

**Files:**
- Create: `frontend/src/react/stores/app/release.ts`
- Create: `frontend/src/react/stores/app/revision.ts`
- Create: `frontend/src/react/stores/app/changelog.ts`
- Create: `frontend/src/react/stores/app/projectWebhook.ts`
- Modify: `frontend/src/react/stores/app/types.ts`
- Modify: `frontend/src/react/stores/app/index.ts`
- Modify: `frontend/src/react/stores/app/index.test.ts`

- [ ] **Step 1: Extend app-store types for the four smaller resources**

Add imports to `types.ts`:

```ts
import type {
  Changelog,
  ChangelogView,
  ListChangelogsRequest,
  GetChangelogRequest,
} from "@/types/proto-es/v1/database_service_pb";
import type { Release } from "@/types/proto-es/v1/release_service_pb";
import type { Revision } from "@/types/proto-es/v1/revision_service_pb";
import type { Webhook } from "@/types/proto-es/v1/project_service_pb";
```

Add these slice types before `NotificationSlice`:

```ts
export type ReleaseSlice = {
  releasesByName: Record<string, Release>;
  releaseRequests: Record<string, Promise<Release | undefined>>;
  listReleasesByProject: (
    project: string,
    pagination?: { pageSize?: number; pageToken?: string },
    showDeleted?: boolean,
    filter?: string
  ) => Promise<{ releases: Release[]; nextPageToken: string }>;
  fetchRelease: (name: string, silent?: boolean) => Promise<Release | undefined>;
  getReleasesByProject: (project: string) => Release[];
  getReleaseByName: (name: string) => Release;
  updateRelease: (
    release: Partial<Release>,
    updateMask: string[]
  ) => Promise<Release>;
  deleteRelease: (name: string) => Promise<void>;
  undeleteRelease: (name: string) => Promise<Release>;
};

export type RevisionSlice = {
  revisionsByName: Record<string, Revision>;
  listRevisionsByDatabase: (
    database: string,
    pagination?: { pageSize?: number; pageToken?: string }
  ) => Promise<{ revisions: Revision[]; nextPageToken: string }>;
  listAllRevisionsByDatabase: (
    database: string,
    pagination?: { pageSize?: number }
  ) => Promise<Revision[]>;
  fetchRevision: (name: string) => Promise<Revision | undefined>;
  getRevisionsByDatabase: (database: string) => Revision[];
  getRevisionByName: (name: string) => Revision | undefined;
  deleteRevision: (name: string) => Promise<void>;
};

export type ChangelogSlice = {
  changelogsByCacheKey: Record<string, Changelog>;
  changelogsByDatabase: Record<string, Changelog[]>;
  changelogRequests: Record<string, Promise<Changelog | undefined>>;
  clearChangelogCache: (parent: string) => void;
  listChangelogs: (
    params: Partial<ListChangelogsRequest>
  ) => Promise<{ changelogs: Changelog[]; nextPageToken: string }>;
  getOrFetchChangelogListOfDatabase: (
    database: string,
    pageSize: number,
    view?: ChangelogView
  ) => Promise<Changelog[]>;
  changelogListByDatabase: (database: string) => Changelog[];
  fetchChangelog: (
    params: Partial<GetChangelogRequest>
  ) => Promise<Changelog | undefined>;
  getOrFetchChangelogByName: (
    name: string,
    view?: ChangelogView
  ) => Promise<Changelog | undefined>;
  getChangelogByName: (
    name: string,
    view?: ChangelogView
  ) => Changelog | undefined;
  fetchPreviousChangelog: (name: string) => Promise<Changelog | undefined>;
};

export type ProjectWebhookSlice = {
  getProjectWebhookFromProjectById: (
    project: Project,
    webhookId: string
  ) => Webhook | undefined;
  createProjectWebhook: (project: string, webhook: Webhook) => Promise<Project>;
  updateProjectWebhook: (
    webhook: Webhook,
    updateMask: string[]
  ) => Promise<Project>;
  deleteProjectWebhook: (webhook: Webhook) => Promise<Project>;
  testProjectWebhook: (
    project: Project,
    webhook: Webhook
  ) => Promise<{ error: string }>;
};
```

Extend `AppStoreState`:

```ts
  UserSlice &
  RoleSlice &
  ReleaseSlice &
  RevisionSlice &
  ChangelogSlice &
  ProjectWebhookSlice &
  NotificationSlice &
```

- [ ] **Step 2: Create release slice**

Create `frontend/src/react/stores/app/release.ts`:

```ts
import { create as createProto } from "@bufbuild/protobuf";
import { createContextValues } from "@connectrpc/connect";
import { releaseServiceClientConnect } from "@/connect";
import { silentContextKey } from "@/connect/context-key";
import { isValidReleaseName, unknownRelease } from "@/types";
import { State } from "@/types/proto-es/v1/common_pb";
import type { Release } from "@/types/proto-es/v1/release_service_pb";
import {
  DeleteReleaseRequestSchema,
  GetReleaseRequestSchema,
  ListReleasesRequestSchema,
  ReleaseSchema,
  UndeleteReleaseRequestSchema,
  UpdateReleaseRequestSchema,
} from "@/types/proto-es/v1/release_service_pb";
import type { AppSliceCreator, ReleaseSlice } from "./types";

const upsertReleases = (
  set: Parameters<AppSliceCreator<ReleaseSlice>>[0],
  releases: Release[]
) => {
  set((state) => ({
    releasesByName: {
      ...state.releasesByName,
      ...Object.fromEntries(releases.map((release) => [release.name, release])),
    },
  }));
};

export const createReleaseSlice: AppSliceCreator<ReleaseSlice> = (set, get) => ({
  releasesByName: {},
  releaseRequests: {},

  listReleasesByProject: async (project, pagination, showDeleted, filter) => {
    const response = await releaseServiceClientConnect.listReleases(
      createProto(ListReleasesRequestSchema, {
        parent: project,
        pageSize: pagination?.pageSize,
        pageToken: pagination?.pageToken || "",
        showDeleted: Boolean(showDeleted),
        filter: filter || "",
      })
    );
    upsertReleases(set, response.releases);
    return { releases: response.releases, nextPageToken: response.nextPageToken };
  },

  fetchRelease: async (name, silent = false) => {
    if (!isValidReleaseName(name)) return undefined;
    const existing = get().releasesByName[name];
    if (existing) return existing;
    const pending = get().releaseRequests[name];
    if (pending) return pending;
    const request = releaseServiceClientConnect
      .getRelease(createProto(GetReleaseRequestSchema, { name }), {
        contextValues: createContextValues().set(silentContextKey, silent),
      })
      .then((release) => {
        set((state) => {
          const { [name]: _, ...releaseRequests } = state.releaseRequests;
          return {
            releasesByName: { ...state.releasesByName, [release.name]: release },
            releaseRequests,
          };
        });
        return release;
      })
      .catch(() => undefined);
    set((state) => ({
      releaseRequests: { ...state.releaseRequests, [name]: request },
    }));
    return request;
  },

  getReleasesByProject: (project) =>
    Object.values(get().releasesByName).filter((release) =>
      release.name.startsWith(`${project}/releases/`)
    ),

  getReleaseByName: (name) => get().releasesByName[name] ?? unknownRelease(),

  updateRelease: async (release, updateMask) => {
    const response = await releaseServiceClientConnect.updateRelease(
      createProto(UpdateReleaseRequestSchema, {
        release: { ...createProto(ReleaseSchema, {}), ...release },
        updateMask: { paths: updateMask },
      })
    );
    upsertReleases(set, [response]);
    return response;
  },

  deleteRelease: async (name) => {
    await releaseServiceClientConnect.deleteRelease(
      createProto(DeleteReleaseRequestSchema, { name })
    );
    set((state) => {
      const cached = state.releasesByName[name];
      if (!cached) return {};
      return {
        releasesByName: {
          ...state.releasesByName,
          [name]: { ...cached, state: State.DELETED },
        },
      };
    });
  },

  undeleteRelease: async (name) => {
    const response = await releaseServiceClientConnect.undeleteRelease(
      createProto(UndeleteReleaseRequestSchema, { name })
    );
    upsertReleases(set, [response]);
    return response;
  },
});
```

- [ ] **Step 3: Create revision slice**

Create `frontend/src/react/stores/app/revision.ts`:

```ts
import { create as createProto } from "@bufbuild/protobuf";
import { revisionServiceClientConnect } from "@/connect";
import type { Revision } from "@/types/proto-es/v1/revision_service_pb";
import {
  DeleteRevisionRequestSchema,
  GetRevisionRequestSchema,
  ListRevisionsRequestSchema,
} from "@/types/proto-es/v1/revision_service_pb";
import { revisionNamePrefix } from "@/store/modules/v1/common";
import type { AppSliceCreator, RevisionSlice } from "./types";

const upsertRevisions = (
  set: Parameters<AppSliceCreator<RevisionSlice>>[0],
  revisions: Revision[]
) => {
  set((state) => ({
    revisionsByName: {
      ...state.revisionsByName,
      ...Object.fromEntries(revisions.map((revision) => [revision.name, revision])),
    },
  }));
};

export const createRevisionSlice: AppSliceCreator<RevisionSlice> = (set, get) => ({
  revisionsByName: {},

  listRevisionsByDatabase: async (database, pagination) => {
    const response = await revisionServiceClientConnect.listRevisions(
      createProto(ListRevisionsRequestSchema, {
        parent: database,
        pageSize: pagination?.pageSize,
        pageToken: pagination?.pageToken,
      })
    );
    upsertRevisions(set, response.revisions);
    return { revisions: response.revisions, nextPageToken: response.nextPageToken };
  },

  listAllRevisionsByDatabase: async (database, pagination) => {
    const revisions: Revision[] = [];
    let pageToken = "";
    do {
      const response = await get().listRevisionsByDatabase(database, {
        pageSize: pagination?.pageSize,
        pageToken,
      });
      revisions.push(...response.revisions);
      pageToken = response.nextPageToken;
    } while (pageToken);
    return revisions;
  },

  fetchRevision: async (name) => {
    const existing = get().revisionsByName[name];
    if (existing) return existing;
    const revision = await revisionServiceClientConnect.getRevision(
      createProto(GetRevisionRequestSchema, { name })
    );
    upsertRevisions(set, [revision]);
    return revision;
  },

  getRevisionsByDatabase: (database) =>
    Object.values(get().revisionsByName).filter((revision) =>
      revision.name.startsWith(`${database}/${revisionNamePrefix}`)
    ),

  getRevisionByName: (name) => get().revisionsByName[name],

  deleteRevision: async (name) => {
    await revisionServiceClientConnect.deleteRevision(
      createProto(DeleteRevisionRequestSchema, { name })
    );
    set((state) => {
      const { [name]: _, ...revisionsByName } = state.revisionsByName;
      return { revisionsByName };
    });
  },
});
```

- [ ] **Step 4: Create changelog slice**

Create `frontend/src/react/stores/app/changelog.ts`:

```ts
import { create as createProto } from "@bufbuild/protobuf";
import { databaseServiceClientConnect } from "@/connect";
import { UNKNOWN_ID } from "@/types";
import {
  ChangelogView,
  GetChangelogRequestSchema,
  ListChangelogsRequestSchema,
  type Changelog,
} from "@/types/proto-es/v1/database_service_pb";
import { extractChangelogUID } from "@/utils/v1/changelog";
import type { AppSliceCreator, ChangelogSlice } from "./types";

const cacheKey = (name: string, view: ChangelogView) => `${name}\u0000${view}`;

const upsertChangelogs = (
  set: Parameters<AppSliceCreator<ChangelogSlice>>[0],
  changelogs: Changelog[],
  view: ChangelogView
) => {
  set((state) => ({
    changelogsByCacheKey: {
      ...state.changelogsByCacheKey,
      ...Object.fromEntries(
        changelogs.map((changelog) => [cacheKey(changelog.name, view), changelog])
      ),
    },
  }));
};

export const createChangelogSlice: AppSliceCreator<ChangelogSlice> = (
  set,
  get
) => ({
  changelogsByCacheKey: {},
  changelogsByDatabase: {},
  changelogRequests: {},

  clearChangelogCache: (parent) => {
    set((state) => {
      const { [parent]: _, ...changelogsByDatabase } = state.changelogsByDatabase;
      return { changelogsByDatabase };
    });
  },

  listChangelogs: async (params) => {
    const parent = params.parent;
    if (!parent) throw new Error('"parent" field is required');
    const view = params.view ?? ChangelogView.BASIC;
    const response = await databaseServiceClientConnect.listChangelogs(
      createProto(ListChangelogsRequestSchema, {
        parent,
        pageSize: params.pageSize,
        pageToken: params.pageToken,
        view,
        filter: params.filter,
      })
    );
    set((state) => ({
      changelogsByDatabase: {
        ...state.changelogsByDatabase,
        [parent]: response.changelogs,
      },
    }));
    upsertChangelogs(set, response.changelogs, view);
    return { changelogs: response.changelogs, nextPageToken: response.nextPageToken };
  },

  getOrFetchChangelogListOfDatabase: async (
    database,
    pageSize,
    view = ChangelogView.BASIC
  ) => {
    const existing = get().changelogsByDatabase[database];
    if (existing) return existing;
    const { changelogs } = await get().listChangelogs({
      parent: database,
      pageSize,
      view,
    });
    return changelogs;
  },

  changelogListByDatabase: (database) => get().changelogsByDatabase[database] ?? [],

  fetchChangelog: async (params) => {
    if (!params.name) return undefined;
    const view = params.view ?? ChangelogView.BASIC;
    const changelog = await databaseServiceClientConnect.getChangelog(
      createProto(GetChangelogRequestSchema, {
        name: params.name,
        view,
      })
    );
    upsertChangelogs(set, [changelog], view);
    return changelog;
  },

  getOrFetchChangelogByName: async (name, view = ChangelogView.BASIC) => {
    const uid = extractChangelogUID(name);
    if (!uid || uid === String(UNKNOWN_ID)) return undefined;
    const existing = get().changelogsByCacheKey[cacheKey(name, view)];
    if (existing) return existing;
    const requestKey = cacheKey(name, view);
    const pending = get().changelogRequests[requestKey];
    if (pending) return pending;
    const request = get().fetchChangelog({ name, view });
    set((state) => ({
      changelogRequests: { ...state.changelogRequests, [requestKey]: request },
    }));
    return request;
  },

  getChangelogByName: (name, view) => {
    if (view !== undefined) return get().changelogsByCacheKey[cacheKey(name, view)];
    return (
      get().changelogsByCacheKey[cacheKey(name, ChangelogView.FULL)] ??
      get().changelogsByCacheKey[cacheKey(name, ChangelogView.BASIC)]
    );
  },

  fetchPreviousChangelog: async (name) => {
    const parts = name.split("/changelogs/");
    if (parts.length !== 2) return undefined;
    const database = parts[0];
    const currentUID = extractChangelogUID(name);
    if (!currentUID || currentUID === String(UNKNOWN_ID)) return undefined;
    const { changelogs } = await get().listChangelogs({
      parent: database,
      pageSize: 1000,
      view: ChangelogView.BASIC,
    });
    const index = changelogs.findIndex(
      (changelog) => extractChangelogUID(changelog.name) === currentUID
    );
    if (index === -1 || index === changelogs.length - 1) return undefined;
    return get().getOrFetchChangelogByName(
      changelogs[index + 1].name,
      ChangelogView.FULL
    );
  },
});
```

- [ ] **Step 5: Create project webhook slice**

Create `frontend/src/react/stores/app/projectWebhook.ts`:

```ts
import { create as createProto } from "@bufbuild/protobuf";
import { projectServiceClientConnect } from "@/connect";
import {
  AddWebhookRequestSchema,
  RemoveWebhookRequestSchema,
  TestWebhookRequestSchema,
  UpdateWebhookRequestSchema,
  type Project,
  type Webhook,
} from "@/types/proto-es/v1/project_service_pb";
import { extractProjectWebhookID } from "@/utils";
import type { AppSliceCreator, ProjectWebhookSlice } from "./types";

export const createProjectWebhookSlice: AppSliceCreator<ProjectWebhookSlice> = (
  _set,
  _get
) => ({
  getProjectWebhookFromProjectById: (project: Project, webhookId: string) =>
    project.webhooks.find(
      (webhook: Webhook) => extractProjectWebhookID(webhook.name) === webhookId
    ),

  createProjectWebhook: async (project, webhook) =>
    projectServiceClientConnect.addWebhook(
      createProto(AddWebhookRequestSchema, { project, webhook })
    ),

  updateProjectWebhook: async (webhook, updateMask) =>
    projectServiceClientConnect.updateWebhook(
      createProto(UpdateWebhookRequestSchema, {
        webhook,
        updateMask: { paths: updateMask },
      })
    ),

  deleteProjectWebhook: async (webhook) =>
    projectServiceClientConnect.removeWebhook(
      createProto(RemoveWebhookRequestSchema, { webhook })
    ),

  testProjectWebhook: async (project, webhook) => {
    const response = await projectServiceClientConnect.testWebhook(
      createProto(TestWebhookRequestSchema, {
        project: project.name,
        webhook,
      })
    );
    return { error: response.error };
  },
});
```

- [ ] **Step 6: Register the four smaller slices**

Modify `frontend/src/react/stores/app/index.ts`:

```ts
import { createChangelogSlice } from "./changelog";
import { createProjectWebhookSlice } from "./projectWebhook";
import { createReleaseSlice } from "./release";
import { createRevisionSlice } from "./revision";
```

Add them after role:

```ts
    ...createRoleSlice(...args),
    ...createReleaseSlice(...args),
    ...createRevisionSlice(...args),
    ...createChangelogSlice(...args),
    ...createProjectWebhookSlice(...args),
    ...createNotificationSlice(...args),
```

- [ ] **Step 7: Add app-store tests for the four smaller slices**

Extend `index.test.ts` mocks and connect clients for:

```ts
  listReleases: vi.fn(),
  getRelease: vi.fn(),
  updateRelease: vi.fn(),
  deleteRelease: vi.fn(),
  undeleteRelease: vi.fn(),
  listRevisions: vi.fn(),
  getRevision: vi.fn(),
  deleteRevision: vi.fn(),
  listChangelogs: vi.fn(),
  getChangelog: vi.fn(),
  addWebhook: vi.fn(),
  updateWebhook: vi.fn(),
  removeWebhook: vi.fn(),
  testWebhook: vi.fn(),
```

Add tests:

```ts
test("lists and mutates releases", async () => {
  const store = createAppStore();
  const release = createProto(ReleaseSchema, {
    name: "projects/a/releases/r1",
    title: "R1",
    state: State.ACTIVE,
  });
  mocks.listReleases.mockResolvedValue({ releases: [release], nextPageToken: "" });
  await store.getState().listReleasesByProject("projects/a");
  expect(store.getState().getReleaseByName(release.name)).toEqual(release);
  mocks.deleteRelease.mockResolvedValue({});
  await store.getState().deleteRelease(release.name);
  expect(store.getState().getReleaseByName(release.name).state).toBe(State.DELETED);
});

test("lists all revisions by database", async () => {
  const store = createAppStore();
  const revision = createProto(RevisionSchema, {
    name: "instances/i/databases/d/revisions/1",
  });
  mocks.listRevisions.mockResolvedValueOnce({
    revisions: [revision],
    nextPageToken: "",
  });
  const revisions = await store
    .getState()
    .listAllRevisionsByDatabase("instances/i/databases/d");
  expect(revisions).toEqual([revision]);
  expect(store.getState().getRevisionByName(revision.name)).toEqual(revision);
});

test("caches changelogs by database and view", async () => {
  const store = createAppStore();
  const changelog = createProto(ChangelogSchema, {
    name: "instances/i/databases/d/changelogs/1",
  });
  mocks.listChangelogs.mockResolvedValue({
    changelogs: [changelog],
    nextPageToken: "",
  });
  await store.getState().getOrFetchChangelogListOfDatabase(
    "instances/i/databases/d",
    100
  );
  expect(
    store.getState().getChangelogByName(changelog.name, ChangelogView.BASIC)
  ).toEqual(changelog);
});

test("runs project webhook operations", async () => {
  const store = createAppStore();
  const webhook = createProto(WebhookSchema, {
    name: "projects/a/webhooks/w1",
    title: "Webhook",
  });
  const project = createProto(ProjectSchema, {
    name: "projects/a",
    webhooks: [webhook],
  });
  expect(store.getState().getProjectWebhookFromProjectById(project, "w1")).toEqual(
    webhook
  );
  mocks.testWebhook.mockResolvedValue({ error: "" });
  await expect(
    store.getState().testProjectWebhook(project, webhook)
  ).resolves.toEqual({ error: "" });
});
```

- [ ] **Step 8: Run app-store tests**

Run:

```bash
pnpm --dir frontend exec vitest run src/react/stores/app/index.test.ts
```

Expected: PASS.

- [ ] **Step 9: Commit smaller resource slices**

```bash
git add frontend/src/react/stores/app/release.ts frontend/src/react/stores/app/revision.ts frontend/src/react/stores/app/changelog.ts frontend/src/react/stores/app/projectWebhook.ts frontend/src/react/stores/app/types.ts frontend/src/react/stores/app/index.ts frontend/src/react/stores/app/index.test.ts
git commit -m "feat(frontend): add release revision changelog webhook app stores"
```

---

### Task 3: Add Hook Facades

**Files:**
- Modify: `frontend/src/react/hooks/useAppState.ts`

- [ ] **Step 1: Add hook facades**

Add these exports near the existing resource hooks in `frontend/src/react/hooks/useAppState.ts`:

```ts
export function useUserByIdentifier(identifier: string | undefined) {
  return useAppStore((state) =>
    identifier ? state.getUserByIdentifier(identifier) : undefined
  );
}

export function useRoleByName(name: string | undefined) {
  return useAppStore((state) =>
    name ? state.getRoleByName(name) : undefined
  );
}

export function useReleaseByName(name: string | undefined) {
  return useAppStore((state) =>
    name ? state.getReleaseByName(name) : undefined
  );
}

export function useRevisionByName(name: string | undefined) {
  return useAppStore((state) =>
    name ? state.getRevisionByName(name) : undefined
  );
}

export function useChangelogByName(name: string | undefined) {
  return useAppStore((state) =>
    name ? state.getChangelogByName(name) : undefined
  );
}
```

- [ ] **Step 2: Run focused hook type coverage**

Run:

```bash
pnpm --dir frontend exec vitest run src/react/stores/app/index.test.ts
```

Expected: PASS.

- [ ] **Step 3: Commit hook facades**

```bash
git add frontend/src/react/hooks/useAppState.ts
git commit -m "feat(frontend): add resource app store hooks"
```

---

### Task 4: Add Legacy Guard and Observe Current Failures

**Files:**
- Modify: `frontend/src/react/no-legacy-vue-deps.test.ts`

- [ ] **Step 1: Add Phase 2 guard**

Append a new test in `frontend/src/react/no-legacy-vue-deps.test.ts`:

```ts
  test("Phase 2 protobuf resource consumers use the React app store", () => {
    const bannedImports = [
      "useUserStore",
      "useRoleStore",
      "useReleaseStore",
      "useRevisionStore",
      "useChangelogStore",
      "useProjectWebhookV1Store",
      "@/store/modules/user",
      "@/store/modules/role",
      "@/store/modules/release",
      "@/store/modules/revision",
      "@/store/modules/v1/changelog",
      "@/store/modules/v1/projectWebhook",
    ];
    const violations: string[] = [];
    for (const [file, source] of Object.entries(sources)) {
      for (const bannedImport of bannedImports) {
        if (source.includes(bannedImport)) {
          violations.push(`${file}: ${bannedImport}`);
        }
      }
    }
    expect(violations).toEqual([]);
  });
```

- [ ] **Step 2: Run guard test and observe expected failure**

Run:

```bash
pnpm --dir frontend exec vitest run src/react/no-legacy-vue-deps.test.ts
```

Expected: FAIL listing current React consumers of the Phase 2 legacy stores. Keep this failing state until Tasks 5-9 migrate the consumers.

- [ ] **Step 3: Keep the guard change uncommitted**

```bash
git diff -- frontend/src/react/no-legacy-vue-deps.test.ts
```

Expected: the diff contains only the Phase 2 guard test. Do not commit this failing guard yet.

---

### Task 5: Migrate User and Role React Consumers

**Files:**
- Modify all non-test React files returned by:
  - `rg -l "useUserStore" frontend/src/react --glob '*.{ts,tsx}' --glob '!**/*.test.ts' --glob '!**/*.test.tsx'`
  - `rg -l "useRoleStore" frontend/src/react --glob '*.{ts,tsx}' --glob '!**/*.test.ts' --glob '!**/*.test.tsx'`
- Modify affected tests that mock `useUserStore` or `useRoleStore`.

- [ ] **Step 1: Replace User store imports**

For each React source importing `useUserStore`, add:

```ts
import { useAppStore } from "@/react/stores/app";
```

Replace:

```ts
const userStore = useUserStore();
```

with only the actions/selectors used in that file. Example for list and cache lookups:

```ts
const listUsers = useAppStore((state) => state.listUsers);
const getUserByIdentifier = useAppStore((state) => state.getUserByIdentifier);
const batchGetOrFetchUsers = useAppStore(
  (state) => state.batchGetOrFetchUsers
);
```

Use these method mappings:

```ts
userStore.fetchUserList(...) -> listUsers(...)
userStore.getUserByIdentifier(...) -> getUserByIdentifier(...)
userStore.getOrFetchUserByIdentifier(...) -> getOrFetchUserByIdentifier(...)
userStore.batchGetOrFetchUsers(...) -> batchGetOrFetchUsers(...)
userStore.createUser(...) -> createUser(...)
userStore.updateUser(...) -> updateUser(...)
userStore.updateEmail(...) -> updateEmail(...)
userStore.archiveUser(...) -> archiveUser(...)
userStore.restoreUser(...) -> restoreUser(...)
```

For reactive display reads, do not use `useVueState`. Replace patterns like:

```ts
const user = useVueState(() => userStore.getUserByIdentifier(name));
```

with:

```ts
const user = useAppStore((state) => state.getUserByIdentifier(name));
```

- [ ] **Step 2: Replace Role store imports**

For each React source importing `useRoleStore`, add:

```ts
import { useAppStore } from "@/react/stores/app";
```

Replace:

```ts
const roleStore = useRoleStore();
```

with only the actions/selectors used in that file:

```ts
const roleList = useAppStore((state) => state.roleList);
const listRoles = useAppStore((state) => state.listRoles);
const getRoleByName = useAppStore((state) => state.getRoleByName);
const upsertRole = useAppStore((state) => state.upsertRole);
const deleteRole = useAppStore((state) => state.deleteRole);
```

Use these method mappings:

```ts
roleStore.roleList -> roleList
roleStore.fetchRoleList() -> listRoles()
roleStore.getRoleByName(name) -> getRoleByName(name)
roleStore.upsertRole(role) -> upsertRole(role)
roleStore.deleteRole(role) -> deleteRole(role)
```

- [ ] **Step 3: Update tests for User and Role consumers**

For tests mocking `@/store` with `useUserStore` or `useRoleStore`, remove those keys and add:

```ts
vi.mock("@/react/stores/app", () => ({
  useAppStore: (selector: (state: unknown) => unknown) =>
    selector({
      listUsers: mocks.listUsers,
      fetchUser: mocks.fetchUser,
      batchGetOrFetchUsers: mocks.batchGetOrFetchUsers,
      getOrFetchUserByIdentifier: mocks.getOrFetchUserByIdentifier,
      getUserByIdentifier: mocks.getUserByIdentifier,
      createUser: mocks.createUser,
      updateUser: mocks.updateUser,
      updateEmail: mocks.updateEmail,
      archiveUser: mocks.archiveUser,
      restoreUser: mocks.restoreUser,
      roleList: mocks.roleList,
      listRoles: mocks.listRoles,
      getRoleByName: mocks.getRoleByName,
      upsertRole: mocks.upsertRole,
      deleteRole: mocks.deleteRole,
    }),
}));
```

Keep only the functions each test actually needs in its mock object.

- [ ] **Step 4: Verify User and Role imports are gone**

Run:

```bash
rg -n "useUserStore|useRoleStore|@/store/modules/user|@/store/modules/role" frontend/src/react --glob '*.{ts,tsx}' --glob '!no-legacy-vue-deps.test.ts'
```

Expected: no output.

- [ ] **Step 5: Run focused tests**

Run:

```bash
pnpm --dir frontend exec vitest run src/react/stores/app/index.test.ts src/react/no-legacy-vue-deps.test.ts src/react/components/UserSelect.test.tsx src/react/components/RoleSelect.test.tsx src/react/pages/settings/MembersPage.test.tsx src/react/pages/project/plan-detail/components/PlanDetailApprovalFlow.test.tsx
```

Expected: tests for already-migrated resources pass; the guard may still fail for Release/Revision/Changelog/ProjectWebhook until Tasks 6-9 complete.

- [ ] **Step 6: Commit User and Role consumers**

```bash
git add frontend/src/react
git commit -m "refactor(frontend): migrate user and role consumers to app store"
```

---

### Task 6: Migrate Release Consumers

**Files:**
- Modify: `frontend/src/react/pages/project/ProjectReleaseDashboardPage.tsx`
- Modify: `frontend/src/react/pages/project/ProjectReleaseDetailPage.tsx`
- Modify: `frontend/src/react/pages/project/database-detail/revision/ImportRevisionSheet.tsx`
- Modify: `frontend/src/react/pages/project/issue-detail/components/IssueDetailStatementSection.tsx`
- Modify: `frontend/src/react/components/task-run-log/useTaskRunLogData.ts`
- Modify affected tests under the same directories.

- [ ] **Step 1: Replace release store imports**

In each release consumer, add:

```ts
import { useAppStore } from "@/react/stores/app";
```

Use these mappings:

```ts
releaseStore.fetchReleasesByProject(...) -> listReleasesByProject(...)
releaseStore.fetchReleaseByName(...) -> fetchRelease(...)
releaseStore.getReleasesByProject(...) -> getReleasesByProject(...)
releaseStore.getReleaseByName(...) -> getReleaseByName(...)
releaseStore.updateRelase(...) -> updateRelease(...)
releaseStore.deleteRelease(...) -> deleteRelease(...)
releaseStore.undeleteRelease(...) -> undeleteRelease(...)
```

Example:

```ts
const listReleasesByProject = useAppStore(
  (state) => state.listReleasesByProject
);
const getReleaseByName = useAppStore((state) => state.getReleaseByName);
```

- [ ] **Step 2: Replace release reactive reads**

Replace:

```ts
const release = useVueState(() => releaseStore.getReleaseByName(name));
```

with:

```ts
const release = useAppStore((state) => state.getReleaseByName(name));
```

- [ ] **Step 3: Update release tests**

For tests mocking `useReleaseStore`, use:

```ts
vi.mock("@/react/stores/app", () => ({
  useAppStore: (selector: (state: unknown) => unknown) =>
    selector({
      listReleasesByProject: mocks.listReleasesByProject,
      fetchRelease: mocks.fetchRelease,
      getReleasesByProject: mocks.getReleasesByProject,
      getReleaseByName: mocks.getReleaseByName,
      updateRelease: mocks.updateRelease,
      deleteRelease: mocks.deleteRelease,
      undeleteRelease: mocks.undeleteRelease,
    }),
}));
```

- [ ] **Step 4: Verify release imports are gone**

Run:

```bash
rg -n "useReleaseStore|@/store/modules/release" frontend/src/react --glob '*.{ts,tsx}' --glob '!no-legacy-vue-deps.test.ts'
```

Expected: no output.

- [ ] **Step 5: Commit release consumers**

```bash
git add frontend/src/react
git commit -m "refactor(frontend): migrate release consumers to app store"
```

---

### Task 7: Migrate Revision Consumers

**Files:**
- Modify: `frontend/src/react/components/revision/RevisionDetailPanel.tsx`
- Modify: `frontend/src/react/pages/project/database-detail/revision/ImportRevisionSheet.tsx`
- Modify: `frontend/src/react/pages/project/database-detail/panels/DatabaseRevisionPanel.tsx`
- Modify affected tests under the same directories.

- [ ] **Step 1: Replace revision store imports**

Add:

```ts
import { useAppStore } from "@/react/stores/app";
```

Use these mappings:

```ts
revisionStore.fetchRevisionsByDatabase(...) -> listRevisionsByDatabase(...)
revisionStore.fetchAllRevisionsByDatabase(...) -> listAllRevisionsByDatabase(...)
revisionStore.getRevisionsByDatabase(...) -> getRevisionsByDatabase(...)
revisionStore.getOrFetchRevisionByName(...) -> fetchRevision(...)
revisionStore.getRevisionByName(...) -> getRevisionByName(...)
revisionStore.deleteRevision(...) -> deleteRevision(...)
```

- [ ] **Step 2: Replace revision reactive reads**

Replace:

```ts
const revision = useVueState(() => revisionStore.getRevisionByName(name));
```

with:

```ts
const revision = useAppStore((state) => state.getRevisionByName(name));
```

- [ ] **Step 3: Verify revision imports are gone**

Run:

```bash
rg -n "useRevisionStore|@/store/modules/revision" frontend/src/react --glob '*.{ts,tsx}' --glob '!no-legacy-vue-deps.test.ts'
```

Expected: no output.

- [ ] **Step 4: Commit revision consumers**

```bash
git add frontend/src/react
git commit -m "refactor(frontend): migrate revision consumers to app store"
```

---

### Task 8: Migrate Changelog Consumers

**Files:**
- Modify: `frontend/src/react/pages/project/DatabaseChangelogDetailPage.tsx`
- Modify: `frontend/src/react/pages/project/ProjectSyncSchemaPage.tsx`
- Modify: `frontend/src/react/pages/project/database-detail/panels/DatabaseChangelogPanel.tsx`
- Modify affected tests under the same directories.

- [ ] **Step 1: Replace changelog store imports**

Add:

```ts
import { useAppStore } from "@/react/stores/app";
```

Use these mappings:

```ts
changelogStore.clearCache(...) -> clearChangelogCache(...)
changelogStore.fetchChangelogList(...) -> listChangelogs(...)
changelogStore.getOrFetchChangelogListOfDatabase(...) -> getOrFetchChangelogListOfDatabase(...)
changelogStore.changelogListByDatabase(...) -> changelogListByDatabase(...)
changelogStore.fetchChangelog(...) -> fetchChangelog(...)
changelogStore.getOrFetchChangelogByName(...) -> getOrFetchChangelogByName(...)
changelogStore.getChangelogByName(...) -> getChangelogByName(...)
changelogStore.fetchPreviousChangelog(...) -> fetchPreviousChangelog(...)
```

- [ ] **Step 2: Replace changelog reactive reads**

Replace:

```ts
const changelog = useVueState(() => changelogStore.getChangelogByName(name));
```

with:

```ts
const changelog = useAppStore((state) => state.getChangelogByName(name));
```

- [ ] **Step 3: Verify changelog imports are gone**

Run:

```bash
rg -n "useChangelogStore|@/store/modules/v1/changelog" frontend/src/react --glob '*.{ts,tsx}' --glob '!no-legacy-vue-deps.test.ts'
```

Expected: no output.

- [ ] **Step 4: Commit changelog consumers**

```bash
git add frontend/src/react
git commit -m "refactor(frontend): migrate changelog consumers to app store"
```

---

### Task 9: Migrate Project Webhook Consumers

**Files:**
- Modify: `frontend/src/react/pages/project/ProjectWebhookForm.tsx`
- Modify: `frontend/src/react/pages/project/ProjectWebhooksPage.tsx`
- Modify: `frontend/src/react/pages/project/ProjectWebhookDetailPage.tsx`
- Modify affected tests under the same directories.

- [ ] **Step 1: Replace project webhook store imports**

Add:

```ts
import { useAppStore } from "@/react/stores/app";
```

Use these mappings:

```ts
projectWebhookStore.getProjectWebhookFromProjectById(...) -> getProjectWebhookFromProjectById(...)
projectWebhookStore.createProjectWebhook(...) -> createProjectWebhook(...)
projectWebhookStore.updateProjectWebhook(...) -> updateProjectWebhook(...)
projectWebhookStore.deleteProjectWebhook(...) -> deleteProjectWebhook(...)
projectWebhookStore.testProjectWebhook(...) -> testProjectWebhook(...)
```

- [ ] **Step 2: Preserve project refresh behavior**

Where callers currently receive an updated `Project` from webhook mutations, keep the existing follow-up behavior. For example:

```ts
const project = await updateProjectWebhook(webhook, updateMask);
projectStore.updateProjectCache?.(project);
```

Use the refresh/update function already present in the current file. For example, if the file currently calls `fetchProject()` after a webhook mutation, keep that call and replace only the project webhook store call.

- [ ] **Step 3: Verify project webhook imports are gone**

Run:

```bash
rg -n "useProjectWebhookV1Store|@/store/modules/v1/projectWebhook" frontend/src/react --glob '*.{ts,tsx}' --glob '!no-legacy-vue-deps.test.ts'
```

Expected: no output.

- [ ] **Step 4: Commit project webhook consumers**

```bash
git add frontend/src/react
git commit -m "refactor(frontend): migrate project webhook consumers to app store"
```

---

### Task 10: Final Guard, Type, and Full Frontend Verification

**Files:**
- Modify only files needed to fix verification failures found in this task.

- [ ] **Step 1: Run legacy import greps**

Run:

```bash
rg -n "useUserStore|useRoleStore|useReleaseStore|useRevisionStore|useChangelogStore|useProjectWebhookV1Store|@/store/modules/user|@/store/modules/role|@/store/modules/release|@/store/modules/revision|@/store/modules/v1/changelog|@/store/modules/v1/projectWebhook" frontend/src/react --glob '*.{ts,tsx}' --glob '!no-legacy-vue-deps.test.ts'
```

Expected: no output.

- [ ] **Step 2: Run focused guard and app-store tests**

Run:

```bash
pnpm --dir frontend exec vitest run src/react/stores/app/index.test.ts src/react/no-legacy-vue-deps.test.ts
```

Expected: PASS.

- [ ] **Step 3: Run formatter/linter fixer**

Run:

```bash
pnpm --dir frontend fix
```

Expected: command exits 0.

- [ ] **Step 4: Run CI-style frontend check**

Run:

```bash
pnpm --dir frontend check
```

Expected: command exits 0.

- [ ] **Step 5: Run React/Vue type check**

Run:

```bash
pnpm --dir frontend type-check
```

Expected: command exits 0.

- [ ] **Step 6: Run full frontend tests**

Run:

```bash
pnpm --dir frontend test
```

Expected: all frontend tests pass.

- [ ] **Step 7: Run whitespace diff check**

Run:

```bash
git diff --check
```

Expected: no output and exit 0.

- [ ] **Step 8: Commit final guard and verification fixes**

When the guard and verification commands pass, commit the guard plus any verification fixes:

```bash
git add frontend/src/react frontend/src/react/stores/app
git commit -m "fix(frontend): complete phase two app store migration"
```

Expected: creates a commit containing `frontend/src/react/no-legacy-vue-deps.test.ts` and any source/test files adjusted during final verification.

---

### Task 11: Prepare Review Summary

**Files:**
- Read-only unless a PR description file is requested separately.

- [ ] **Step 1: Summarize commits**

Run:

```bash
git log --oneline main..HEAD
```

Expected: shows the Phase 2 migration commits on the working branch.

- [ ] **Step 2: Summarize final changed files**

Run:

```bash
git diff --stat main...HEAD
```

Expected: shows app-store slices, migrated consumers, tests, and guard updates.

- [ ] **Step 3: Prepare final handoff**

Report:

```md
Implemented Phase 2 resource store migration.

Resources migrated:
- User
- Role
- Release
- Revision
- Changelog
- ProjectWebhook

Verification:
- pnpm --dir frontend exec vitest run src/react/stores/app/index.test.ts src/react/no-legacy-vue-deps.test.ts
- pnpm --dir frontend fix
- pnpm --dir frontend check
- pnpm --dir frontend type-check
- pnpm --dir frontend test
- git diff --check
```

Use the actual command outcomes from Task 10.

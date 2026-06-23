import { create as createProto } from "@bufbuild/protobuf";
import { beforeEach, describe, expect, test, vi } from "vitest";
import { ReactShellBridgeEvent } from "@/react/shell-bridge";
import { isValidDatabaseGroupName, UNKNOWN_PROJECT_NAME } from "@/types";
import { ExprSchema } from "@/types/proto-es/google/type/expr_pb";
import {
  AccessGrant_Status,
  AccessGrantSchema,
} from "@/types/proto-es/v1/access_grant_service_pb";
import { ActuatorInfoSchema } from "@/types/proto-es/v1/actuator_service_pb";
import { State } from "@/types/proto-es/v1/common_pb";
import {
  DatabaseGroupSchema,
  DatabaseGroupView,
} from "@/types/proto-es/v1/database_group_service_pb";
import {
  ChangelogSchema,
  ChangelogView,
  DatabaseSchema$,
} from "@/types/proto-es/v1/database_service_pb";
import { GroupSchema } from "@/types/proto-es/v1/group_service_pb";
import {
  BindingSchema,
  IamPolicySchema,
} from "@/types/proto-es/v1/iam_policy_pb";
import { IdentityProviderSchema } from "@/types/proto-es/v1/idp_service_pb";
import { InstanceRoleSchema } from "@/types/proto-es/v1/instance_role_service_pb";
import { InstanceSchema } from "@/types/proto-es/v1/instance_service_pb";
import { IssueCommentSchema } from "@/types/proto-es/v1/issue_service_pb";
import {
  ProjectSchema,
  WebhookSchema,
} from "@/types/proto-es/v1/project_service_pb";
import { ReleaseSchema } from "@/types/proto-es/v1/release_service_pb";
import { RevisionSchema } from "@/types/proto-es/v1/revision_service_pb";
import { RoleSchema } from "@/types/proto-es/v1/role_service_pb";
import { ServiceAccountSchema } from "@/types/proto-es/v1/service_account_service_pb";
import {
  DatabaseChangeMode,
  EnvironmentSetting_EnvironmentSchema,
  EnvironmentSettingSchema,
  SettingSchema,
  SettingValueSchema,
  WorkspaceProfileSettingSchema,
} from "@/types/proto-es/v1/setting_service_pb";
import { SheetSchema } from "@/types/proto-es/v1/sheet_service_pb";
import {
  PlanFeature,
  PlanType,
  SubscriptionSchema,
} from "@/types/proto-es/v1/subscription_service_pb";
import {
  UpdateUserRequestSchema,
  UserSchema,
} from "@/types/proto-es/v1/user_service_pb";
import { WorkloadIdentitySchema } from "@/types/proto-es/v1/workload_identity_service_pb";
import { WorkspaceSchema } from "@/types/proto-es/v1/workspace_service_pb";
import {
  storageKeyIntroState,
  storageKeyRecentProjects,
  storageKeyRecentVisit,
} from "@/utils/storage-keys";
import { createAppStore } from ".";

const mocks = vi.hoisted(() => ({
  localStorage: (() => {
    const storage = new Map<string, string>();
    const localStorage = {
      clear: vi.fn(() => storage.clear()),
      getItem: vi.fn((key: string) => storage.get(key) ?? null),
      removeItem: vi.fn((key: string) => storage.delete(key)),
      setItem: vi.fn((key: string, value: string) => {
        storage.set(key, value);
      }),
    };
    vi.stubGlobal("localStorage", localStorage);
    return localStorage;
  })(),
  getCurrentUser: vi.fn(),
  logout: vi.fn(),
  getActuatorInfo: vi.fn(),
  getWorkspace: vi.fn(),
  updateWorkspace: vi.fn(),
  getIamPolicy: vi.fn(),
  setIamPolicy: vi.fn(),
  listRoles: vi.fn(),
  updateRole: vi.fn(),
  deleteRole: vi.fn(),
  getSubscription: vi.fn(),
  uploadLicense: vi.fn(),
  getSetting: vi.fn(),
  getProject: vi.fn(),
  getProjectIamPolicy: vi.fn(),
  batchGetProjects: vi.fn(),
  searchProjects: vi.fn(),
  createProject: vi.fn(),
  addWebhook: vi.fn(),
  updateWebhook: vi.fn(),
  removeWebhook: vi.fn(),
  testWebhook: vi.fn(),
  getInstance: vi.fn(),
  listReleases: vi.fn(),
  getRelease: vi.fn(),
  updateRelease: vi.fn(),
  deleteRelease: vi.fn(),
  undeleteRelease: vi.fn(),
  listRevisions: vi.fn(),
  getRevision: vi.fn(),
  deleteRevision: vi.fn(),
  getDatabase: vi.fn(),
  batchGetDatabases: vi.fn(),
  listDatabases: vi.fn(),
  listChangelogs: vi.fn(),
  getChangelog: vi.fn(),
  getDatabaseMetadata: vi.fn(),
  getDatabaseGroup: vi.fn(),
  listDatabaseGroups: vi.fn(),
  getSheet: vi.fn(),
  createSheet: vi.fn(),
  listInstanceRoles: vi.fn(),
  listGroups: vi.fn(),
  batchGetGroups: vi.fn(),
  getGroup: vi.fn(),
  createGroup: vi.fn(),
  updateGroup: vi.fn(),
  deleteGroup: vi.fn(),
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
  listIdentityProviders: vi.fn(),
  getIdentityProvider: vi.fn(),
  createIdentityProvider: vi.fn(),
  updateIdentityProvider: vi.fn(),
  deleteIdentityProvider: vi.fn(),
  listUsers: vi.fn(),
  getUser: vi.fn(),
  batchGetUsers: vi.fn(),
  createUser: vi.fn(),
  updateUser: vi.fn(),
  updateEmail: vi.fn(),
  deleteUser: vi.fn(),
  undeleteUser: vi.fn(),
  getAccessGrant: vi.fn(),
  searchMyAccessGrants: vi.fn(),
  createAccessGrant: vi.fn(),
  listAccessGrants: vi.fn(),
  activateAccessGrant: vi.fn(),
  revokeAccessGrant: vi.fn(),
  listIssueComments: vi.fn(),
  createIssueComment: vi.fn(),
  updateIssueComment: vi.fn(),
}));

vi.mock("@/connect", () => ({
  actuatorServiceClientConnect: {
    getActuatorInfo: mocks.getActuatorInfo,
  },
  authServiceClientConnect: {
    logout: mocks.logout,
  },
  projectServiceClientConnect: {
    getProject: mocks.getProject,
    getIamPolicy: mocks.getProjectIamPolicy,
    batchGetProjects: mocks.batchGetProjects,
    searchProjects: mocks.searchProjects,
    createProject: mocks.createProject,
    addWebhook: mocks.addWebhook,
    updateWebhook: mocks.updateWebhook,
    removeWebhook: mocks.removeWebhook,
    testWebhook: mocks.testWebhook,
  },
  releaseServiceClientConnect: {
    listReleases: mocks.listReleases,
    getRelease: mocks.getRelease,
    updateRelease: mocks.updateRelease,
    deleteRelease: mocks.deleteRelease,
    undeleteRelease: mocks.undeleteRelease,
  },
  revisionServiceClientConnect: {
    listRevisions: mocks.listRevisions,
    getRevision: mocks.getRevision,
    deleteRevision: mocks.deleteRevision,
  },
  instanceServiceClientConnect: {
    getInstance: mocks.getInstance,
  },
  databaseServiceClientConnect: {
    getDatabase: mocks.getDatabase,
    batchGetDatabases: mocks.batchGetDatabases,
    listDatabases: mocks.listDatabases,
    listChangelogs: mocks.listChangelogs,
    getChangelog: mocks.getChangelog,
    getDatabaseMetadata: mocks.getDatabaseMetadata,
  },
  databaseGroupServiceClientConnect: {
    getDatabaseGroup: mocks.getDatabaseGroup,
    listDatabaseGroups: mocks.listDatabaseGroups,
  },
  sheetServiceClientConnect: {
    getSheet: mocks.getSheet,
    createSheet: mocks.createSheet,
  },
  instanceRoleServiceClientConnect: {
    listInstanceRoles: mocks.listInstanceRoles,
  },
  groupServiceClientConnect: {
    listGroups: mocks.listGroups,
    batchGetGroups: mocks.batchGetGroups,
    getGroup: mocks.getGroup,
    createGroup: mocks.createGroup,
    updateGroup: mocks.updateGroup,
    deleteGroup: mocks.deleteGroup,
  },
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
  identityProviderServiceClientConnect: {
    listIdentityProviders: mocks.listIdentityProviders,
    getIdentityProvider: mocks.getIdentityProvider,
    createIdentityProvider: mocks.createIdentityProvider,
    updateIdentityProvider: mocks.updateIdentityProvider,
    deleteIdentityProvider: mocks.deleteIdentityProvider,
  },
  accessGrantServiceClientConnect: {
    getAccessGrant: mocks.getAccessGrant,
    searchMyAccessGrants: mocks.searchMyAccessGrants,
    createAccessGrant: mocks.createAccessGrant,
    listAccessGrants: mocks.listAccessGrants,
    activateAccessGrant: mocks.activateAccessGrant,
    revokeAccessGrant: mocks.revokeAccessGrant,
  },
  roleServiceClientConnect: {
    listRoles: mocks.listRoles,
    updateRole: mocks.updateRole,
    deleteRole: mocks.deleteRole,
  },
  settingServiceClientConnect: {
    getSetting: mocks.getSetting,
  },
  subscriptionServiceClientConnect: {
    getSubscription: mocks.getSubscription,
    uploadLicense: mocks.uploadLicense,
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
  workspaceServiceClientConnect: {
    getWorkspace: mocks.getWorkspace,
    updateWorkspace: mocks.updateWorkspace,
    getIamPolicy: mocks.getIamPolicy,
    setIamPolicy: mocks.setIamPolicy,
  },
  issueServiceClientConnect: {
    listIssueComments: mocks.listIssueComments,
    createIssueComment: mocks.createIssueComment,
    updateIssueComment: mocks.updateIssueComment,
  },
}));

const user = createProto(UserSchema, {
  name: "users/alice@example.com",
  email: "alice@example.com",
  groups: ["groups/dba"],
  workspace: "workspaces/default",
});

const projectA = createProto(ProjectSchema, {
  name: "projects/a",
  title: "A",
  state: State.ACTIVE,
});

const projectB = createProto(ProjectSchema, {
  name: "projects/b",
  title: "B",
  state: State.ACTIVE,
});

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

const userA = createProto(UserSchema, {
  name: "users/bob@example.com",
  email: "bob@example.com",
  title: "Bob",
  state: State.ACTIVE,
});

const userB = createProto(UserSchema, {
  name: "users/carol@example.com",
  email: "carol@example.com",
  title: "Carol",
  state: State.ACTIVE,
});

const roleA = createProto(RoleSchema, {
  name: "roles/sql-reviewer",
  title: "SQL Reviewer",
  permissions: ["bb.plans.get"],
});

const roleB = createProto(RoleSchema, {
  name: "roles/sql-admin",
  title: "SQL Admin",
  permissions: ["bb.plans.get", "bb.plans.update"],
});

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

const identityProviderA = createProto(IdentityProviderSchema, {
  name: "idps/google",
  title: "Google",
  domain: "example.com",
});

const accessGrantA = createProto(AccessGrantSchema, {
  name: "projects/a/accessGrants/ag1",
  status: AccessGrant_Status.ACTIVE,
});

const releaseA = createProto(ReleaseSchema, {
  name: "projects/a/releases/rel-a",
  state: State.ACTIVE,
});

const releaseB = createProto(ReleaseSchema, {
  name: "projects/a/releases/rel-b",
  state: State.ACTIVE,
});

const revisionA = createProto(RevisionSchema, {
  name: "instances/i1/databases/db1/revisions/1",
});

const revisionB = createProto(RevisionSchema, {
  name: "instances/i1/databases/db1/revisions/2",
});

const changelogA = createProto(ChangelogSchema, {
  name: "instances/i1/databases/db1/changelogs/1",
});

const changelogB = createProto(ChangelogSchema, {
  name: "instances/i1/databases/db1/changelogs/2",
  schema: "full",
});

const webhookA = createProto(WebhookSchema, {
  name: "projects/a/webhooks/hook-a",
  title: "Hook A",
  url: "https://example.com/hook-a",
});

const timestampSeconds = (seconds: number) => ({
  seconds: BigInt(seconds),
  nanos: 0,
});

beforeEach(() => {
  vi.clearAllMocks();
  localStorage.clear();
});

describe("useAppStore", () => {
  test("combines app state slices behind one bounded store", () => {
    const store = createAppStore();
    const state = store.getState();

    expect(state.loadCurrentUser).toBeTypeOf("function");
    expect(state.loadWorkspace).toBeTypeOf("function");
    expect(state.hasWorkspacePermission).toBeTypeOf("function");
    expect(state.fetchProject).toBeTypeOf("function");
    expect(state.fetchInstance).toBeTypeOf("function");
    expect(state.notify).toBeTypeOf("function");
    expect(state.recordRecentVisit).toBeTypeOf("function");
    expect(state.removeRecentVisit).toBeTypeOf("function");
  });

  test("keeps the signed-in identity when marking the session expired", () => {
    const store = createAppStore();
    store.setState({
      currentUser: user,
      currentUserName: user.name,
    });

    store.getState().setUnauthenticatedOccurred(true);

    expect(store.getState().unauthenticatedOccurred).toBe(true);
    expect(store.getState().currentUser).toBe(user);
    expect(store.getState().currentUserName).toBe(user.name);
    expect(store.getState().isLoggedIn()).toBe(true);
  });

  test("keeps the signed-in identity when current-user refresh has a transient failure", async () => {
    mocks.getCurrentUser.mockRejectedValue(new Error("network unavailable"));
    const store = createAppStore();
    store.setState({
      currentUser: user,
      currentUserName: user.name,
    });

    const result = await store.getState().fetchCurrentUser();

    expect(result).toBeUndefined();
    expect(store.getState().currentUser).toBe(user);
    expect(store.getState().currentUserName).toBe(user.name);
    expect(store.getState().isLoggedIn()).toBe(true);
  });

  test("lists groups and populates the group cache", async () => {
    mocks.listGroups.mockResolvedValue({
      groups: [groupA, groupB],
      nextPageToken: "next",
    });
    const store = createAppStore();
    store.setState({
      currentUser: user,
      roles: [
        createProto(RoleSchema, {
          name: "roles/group-viewer",
          permissions: ["bb.groups.list"],
        }),
      ],
      workspacePolicy: createProto(IamPolicySchema, {
        bindings: [
          createProto(BindingSchema, {
            role: "roles/group-viewer",
            members: [`user:${user.email}`],
          }),
        ],
      }),
    });

    const result = await store.getState().listGroups({ pageSize: 50 });

    expect(result.groups).toEqual([groupA, groupB]);
    expect(result.nextPageToken).toBe("next");
    expect(store.getState().groupsByName[groupA.name]).toBe(groupA);
    expect(store.getState().groupsByName[groupB.name]).toBe(groupB);
  });

  test("lists users and populates the user cache", async () => {
    mocks.listUsers.mockResolvedValue({
      users: [userA, userB],
      nextPageToken: "next-user",
    });
    const store = createAppStore();
    store.setState({
      currentUser: user,
      roles: [
        createProto(RoleSchema, {
          name: "roles/user-viewer",
          permissions: ["bb.users.list"],
        }),
      ],
      workspacePolicy: createProto(IamPolicySchema, {
        bindings: [
          createProto(BindingSchema, {
            role: "roles/user-viewer",
            members: [`user:${user.email}`],
          }),
        ],
      }),
    });

    const result = await store.getState().listUsers({
      pageSize: 50,
      filter: { query: "BO", state: State.DELETED },
    });

    expect(result.users).toEqual([userA, userB]);
    expect(result.nextPageToken).toBe("next-user");
    expect(store.getState().usersByName[userA.name]).toBe(userA);
    expect(store.getState().usersByName[userB.name]).toBe(userB);
    expect(mocks.listUsers).toHaveBeenCalledWith(
      expect.objectContaining({
        filter:
          '(name.contains("bo") || email.contains("bo")) && state == "DELETED"',
        showDeleted: true,
      })
    );
  });

  test("listUsers omits the project filter for unknown project sentinels", async () => {
    mocks.listUsers.mockResolvedValue({
      users: [userA],
      nextPageToken: "",
    });
    const store = createAppStore();
    store.setState({
      currentUser: user,
      roles: [
        createProto(RoleSchema, {
          name: "roles/user-viewer",
          permissions: ["bb.users.list"],
        }),
      ],
      workspacePolicy: createProto(IamPolicySchema, {
        bindings: [
          createProto(BindingSchema, {
            role: "roles/user-viewer",
            members: [`user:${user.email}`],
          }),
        ],
      }),
    });

    for (const project of [UNKNOWN_PROJECT_NAME, "projects/-"]) {
      await store.getState().listUsers({
        pageSize: 50,
        filter: { project, query: "BO" },
      });
    }

    expect(
      mocks.listUsers.mock.calls.map(([request]) => request.filter)
    ).toEqual([
      '(name.contains("bo") || email.contains("bo"))',
      '(name.contains("bo") || email.contains("bo"))',
    ]);
  });

  test("batchGetOrFetchUsers fetches missing users and returns unknown fallback", async () => {
    mocks.batchGetUsers.mockResolvedValue({ users: [userB] });
    const store = createAppStore();
    store.setState({ usersByName: { [userA.name]: userA } });

    const result = await store
      .getState()
      .batchGetOrFetchUsers([userA.name, "user:carol@example.com"]);

    expect(mocks.batchGetUsers).toHaveBeenCalledWith(
      expect.objectContaining({ names: [userB.name] }),
      expect.anything()
    );
    expect(result.map((u) => u.name)).toEqual([userA.name, userB.name]);

    mocks.batchGetUsers.mockResolvedValueOnce({ users: [] });
    const missing = await store
      .getState()
      .batchGetOrFetchUsers(["users/missing@example.com"]);

    expect(missing[0]).toMatchObject({
      name: "users/missing@example.com",
      email: "missing@example.com",
    });
  });

  test("updateEmail removes the old user cache key and stores the updated user", async () => {
    const updated = createProto(UserSchema, {
      ...userA,
      name: "users/robert@example.com",
      email: "robert@example.com",
    });
    mocks.updateEmail.mockResolvedValue(updated);
    const store = createAppStore();
    store.setState({ usersByName: { [userA.name]: userA } });

    const result = await store
      .getState()
      .updateEmail("bob@example.com", "robert@example.com");

    expect(result).toBe(updated);
    expect(store.getState().usersByName[userA.name]).toBeUndefined();
    expect(store.getState().usersByName[updated.name]).toBe(updated);
  });

  test("updateEmail refreshes currentUser when the signed-in user changes email", async () => {
    const updated = createProto(UserSchema, {
      ...user,
      name: "users/alice-updated@example.com",
      email: "alice-updated@example.com",
    });
    mocks.updateEmail.mockResolvedValue(updated);
    const store = createAppStore();
    store.setState({
      currentUser: user,
      usersByName: { [user.name]: user },
    });

    await store.getState().updateEmail(user.email, updated.email);

    expect(store.getState().currentUser).toBe(updated);
  });

  test("updateUser passes allowMissing through without requiring an existing user", async () => {
    mocks.updateUser.mockResolvedValue(userA);
    const store = createAppStore();

    const result = await store.getState().updateUser(
      createProto(UpdateUserRequestSchema, {
        user: userA,
        allowMissing: true,
      })
    );

    expect(result).toBe(userA);
    expect(mocks.getUser).not.toHaveBeenCalled();
    expect(mocks.updateUser).toHaveBeenCalledWith(
      expect.objectContaining({
        user: userA,
        allowMissing: true,
      })
    );
    expect(store.getState().usersByName[userA.name]).toBe(userA);
  });

  test("updateUser refreshes currentUser when the signed-in user is updated", async () => {
    const updated = createProto(UserSchema, {
      ...user,
      title: "Alice Updated",
    });
    mocks.updateUser.mockResolvedValue(updated);
    const store = createAppStore();
    store.setState({
      currentUser: user,
      usersByName: { [user.name]: user },
    });

    await store.getState().updateUser(
      createProto(UpdateUserRequestSchema, {
        user: updated,
      })
    );

    expect(store.getState().currentUser).toBe(updated);
  });

  test("updateWorkspace refreshes the current workspace and list entry", async () => {
    const existing = createProto(WorkspaceSchema, {
      name: "workspaces/default",
      title: "Old",
      logo: "old-logo",
    });
    const updated = createProto(WorkspaceSchema, {
      name: "workspaces/default",
      title: "Old",
      logo: "new-logo",
    });
    mocks.updateWorkspace.mockResolvedValue(updated);
    const store = createAppStore();
    store.setState({ workspace: existing, workspaceList: [existing] });

    const result = await store.getState().updateWorkspace(updated, ["logo"]);

    expect(result).toBe(updated);
    expect(store.getState().workspace).toBe(updated);
    expect(store.getState().workspaceList[0]).toBe(updated);
  });

  test("updateWorkspace leaves workspace untouched when a different workspace is active", async () => {
    const active = createProto(WorkspaceSchema, {
      name: "workspaces/other",
      title: "Other",
    });
    const updated = createProto(WorkspaceSchema, {
      name: "workspaces/default",
      title: "Default",
      logo: "new-logo",
    });
    mocks.updateWorkspace.mockResolvedValue(updated);
    const store = createAppStore();
    store.setState({ workspace: active, workspaceList: [active] });

    await store.getState().updateWorkspace(updated, ["logo"]);

    expect(store.getState().workspace).toBe(active);
  });

  test("patchWorkspaceIamPolicy adds the member to the requested role bindings", async () => {
    const existing = createProto(IamPolicySchema, {
      bindings: [
        createProto(BindingSchema, {
          role: "roles/workspaceMember",
          members: ["user:alice@example.com"],
        }),
      ],
    });
    const setPolicy = createProto(IamPolicySchema, {
      bindings: [
        createProto(BindingSchema, {
          role: "roles/workspaceMember",
          members: ["user:alice@example.com", "user:bob@example.com"],
        }),
        createProto(BindingSchema, {
          role: "roles/workspaceAdmin",
          members: ["user:bob@example.com"],
        }),
      ],
    });
    mocks.setIamPolicy.mockResolvedValue(setPolicy);
    const store = createAppStore();
    store.setState({ workspacePolicy: existing });

    await store.getState().patchWorkspaceIamPolicy([
      {
        member: "user:bob@example.com",
        roles: ["roles/workspaceMember", "roles/workspaceAdmin"],
      },
    ]);

    expect(mocks.setIamPolicy).toHaveBeenCalledTimes(1);
    const request = mocks.setIamPolicy.mock.calls[0][0] as {
      policy: { bindings: { role: string; members: string[] }[] };
    };
    // Existing member binding gains bob, and a new admin binding is appended.
    expect(
      request.policy.bindings.map(({ role, members }) => ({ role, members }))
    ).toEqual([
      {
        role: "roles/workspaceMember",
        members: ["user:alice@example.com", "user:bob@example.com"],
      },
      {
        role: "roles/workspaceAdmin",
        members: ["user:bob@example.com"],
      },
    ]);
    expect(store.getState().workspacePolicy).toBe(setPolicy);
  });

  test("workspaceUserMapToRoles inverts policy bindings to member -> roles", () => {
    const policy = createProto(IamPolicySchema, {
      bindings: [
        createProto(BindingSchema, {
          role: "roles/workspaceMember",
          members: ["user:alice@example.com", "user:bob@example.com"],
        }),
        createProto(BindingSchema, {
          role: "roles/workspaceAdmin",
          members: ["user:alice@example.com"],
        }),
      ],
    });
    const store = createAppStore();
    store.setState({ workspacePolicy: policy });

    const map = store.getState().workspaceUserMapToRoles();
    expect([...(map.get("users/alice@example.com") ?? [])].sort()).toEqual([
      "roles/workspaceAdmin",
      "roles/workspaceMember",
    ]);
    expect([...(map.get("users/bob@example.com") ?? [])]).toEqual([
      "roles/workspaceMember",
    ]);
  });

  test("create archive and restore update activated user count when server info is loaded", async () => {
    mocks.createUser.mockResolvedValue(userA);
    mocks.deleteUser.mockResolvedValue({});
    mocks.undeleteUser.mockResolvedValue(userA);
    const store = createAppStore();
    store.setState({
      serverInfo: createProto(ActuatorInfoSchema, {
        activatedUserCount: 2,
      }),
    });

    await store.getState().createUser(userA);
    expect(store.getState().serverInfo?.activatedUserCount).toBe(3);

    await store.getState().archiveUser(userA.name);
    expect(store.getState().serverInfo?.activatedUserCount).toBe(2);

    await store.getState().restoreUser(userA.name);
    expect(store.getState().serverInfo?.activatedUserCount).toBe(3);

    store.setState({
      serverInfo: createProto(ActuatorInfoSchema, {
        activatedUserCount: 0,
      }),
    });

    await store.getState().archiveUser(userA.name);
    expect(store.getState().serverInfo?.activatedUserCount).toBe(0);
  });

  test("lists, upserts, and deletes roles", async () => {
    mocks.listRoles.mockResolvedValue({ roles: [roleA] });
    mocks.updateRole.mockResolvedValue(roleB);
    mocks.deleteRole.mockResolvedValue({});
    const store = createAppStore();

    const roles = await store.getState().listRoles();
    expect(roles).toEqual([roleA]);
    expect(store.getState().getRoleByName(roleA.name)).toBe(roleA);

    const upserted = await store.getState().upsertRole(roleB);
    expect(upserted).toBe(roleB);
    expect(store.getState().roleList).toEqual([roleA, roleB]);
    expect(mocks.updateRole).toHaveBeenCalledWith(
      expect.objectContaining({
        role: roleB,
        updateMask: expect.objectContaining({
          paths: ["title", "description", "permissions"],
        }),
        allowMissing: true,
      })
    );

    await store.getState().deleteRole(roleA);
    expect(store.getState().roleList).toEqual([roleB]);
    expect(mocks.deleteRole).toHaveBeenCalledWith(
      expect.objectContaining({ name: roleA.name })
    );
  });

  test("loads workspace permission roles into the role display cache", async () => {
    mocks.getCurrentUser.mockResolvedValue(user);
    mocks.getActuatorInfo.mockResolvedValue({
      workspace: "workspaces/default",
    });
    mocks.getWorkspace.mockResolvedValue({ name: "workspaces/default" });
    mocks.listRoles.mockResolvedValue({ roles: [roleA, roleB] });
    mocks.getIamPolicy.mockResolvedValue(createProto(IamPolicySchema, {}));
    const store = createAppStore();

    await store.getState().loadWorkspacePermissionState();

    expect(store.getState().roles).toEqual([roleA, roleB]);
    expect(store.getState().roleList).toEqual([roleA, roleB]);
  });

  test("prefetches user members referenced by the workspace policy", async () => {
    mocks.getCurrentUser.mockResolvedValue(user);
    mocks.getActuatorInfo.mockResolvedValue({
      workspace: "workspaces/default",
    });
    mocks.getWorkspace.mockResolvedValue({ name: "workspaces/default" });
    mocks.listRoles.mockResolvedValue({ roles: [roleA] });
    mocks.getIamPolicy.mockResolvedValue(
      createProto(IamPolicySchema, {
        bindings: [
          createProto(BindingSchema, {
            role: roleA.name,
            members: [`user:${userA.email}`],
          }),
        ],
      })
    );
    mocks.batchGetUsers.mockResolvedValue({ users: [userA] });
    const store = createAppStore();

    await store.getState().loadWorkspacePermissionState();

    expect(mocks.batchGetUsers).toHaveBeenCalledWith(
      expect.objectContaining({ names: [userA.name] }),
      expect.anything()
    );
    expect(store.getState().usersByName[userA.name]).toBe(userA);
  });

  test("lists releases and marks cached release deleted", async () => {
    mocks.listReleases.mockResolvedValue({
      releases: [releaseA, releaseB],
      nextPageToken: "next-release",
    });
    mocks.deleteRelease.mockResolvedValue({});
    const store = createAppStore();

    const result = await store
      .getState()
      .listReleasesByProject("projects/a", { pageSize: 20 }, true, "x");

    expect(result.releases).toEqual([releaseA, releaseB]);
    expect(result.nextPageToken).toBe("next-release");
    expect(store.getState().releasesByName[releaseA.name]).toBe(releaseA);
    expect(store.getState().getReleasesByProject("projects/a")).toEqual([
      releaseA,
      releaseB,
    ]);
    expect(mocks.listReleases).toHaveBeenCalledWith(
      expect.objectContaining({
        parent: "projects/a",
        pageSize: 20,
        showDeleted: true,
        filter: "x",
      })
    );

    await store.getState().deleteRelease(releaseA.name);

    expect(store.getState().releasesByName[releaseA.name].state).toBe(
      State.DELETED
    );
  });

  test("returns undefined for an unknown release", () => {
    const store = createAppStore();

    // getReleaseByName backs the useReleaseByName Zustand selector. Returning a
    // freshly-constructed sentinel on every call made the selector snapshot
    // change each render, triggering an infinite re-render loop on cache miss.
    // A stable undefined avoids that, matching the other cache-miss getters.
    expect(
      store.getState().getReleaseByName("projects/a/releases/miss")
    ).toBeUndefined();
  });

  test("reuses stable fallback resources on cache misses", () => {
    const store = createAppStore();

    // These getters are tempting to use inside Zustand selectors. Returning a
    // fresh proto on every cache miss makes the selector snapshot unstable in
    // Zustand v5 / React useSyncExternalStore and can render-loop.
    expect(store.getState().getProjectByName("projects/missing")).toBe(
      store.getState().getProjectByName("projects/missing")
    );
    expect(store.getState().getProjectByName("projects/-")).toBe(
      store.getState().getProjectByName("projects/-")
    );
    expect(
      store.getState().getDatabaseByName("instances/i/databases/miss")
    ).toBe(store.getState().getDatabaseByName("instances/i/databases/miss"));
    expect(store.getState().getInstanceByName("instances/missing")).toBe(
      store.getState().getInstanceByName("instances/missing")
    );
    expect(
      store.getState().getDBGroupByName("projects/p/databaseGroups/miss")
    ).toBe(store.getState().getDBGroupByName("projects/p/databaseGroups/miss"));
    expect(store.getState().getRolloutByName("projects/p/rollouts/miss")).toBe(
      store.getState().getRolloutByName("projects/p/rollouts/miss")
    );
    expect(store.getState().getEnvironmentByName("environments/missing")).toBe(
      store.getState().getEnvironmentByName("environments/missing")
    );
    expect(
      store.getState().getDatabaseCatalog("instances/i/databases/miss")
    ).toBe(store.getState().getDatabaseCatalog("instances/i/databases/miss"));
    expect(
      store.getState().getDatabaseMetadata("instances/i/databases/miss")
    ).toBe(store.getState().getDatabaseMetadata("instances/i/databases/miss"));
  });

  test("deduplicates release fetches and clears failed requests", async () => {
    mocks.getRelease.mockResolvedValueOnce(releaseA);
    const store = createAppStore();

    const [first, second] = await Promise.all([
      store.getState().fetchRelease(releaseA.name),
      store.getState().fetchRelease(releaseA.name),
    ]);

    expect(first).toBe(releaseA);
    expect(second).toBe(releaseA);
    expect(mocks.getRelease).toHaveBeenCalledTimes(1);
    expect(store.getState().releaseRequests[releaseA.name]).toBeUndefined();

    const failedName = "projects/a/releases/fail";
    mocks.getRelease.mockRejectedValueOnce(new Error("failed"));
    await expect(store.getState().fetchRelease(failedName)).rejects.toThrow(
      "failed"
    );
    expect(store.getState().releaseRequests[failedName]).toBeUndefined();
  });

  test("paginates all revisions by database and caches results", async () => {
    mocks.listRevisions
      .mockResolvedValueOnce({
        revisions: [revisionA],
        nextPageToken: "next-revision",
      })
      .mockResolvedValueOnce({
        revisions: [revisionB],
        nextPageToken: "",
      });
    const store = createAppStore();

    const result = await store
      .getState()
      .listAllRevisionsByDatabase("instances/i1/databases/db1", {
        pageSize: 1,
      });

    expect(result).toEqual([revisionA, revisionB]);
    expect(mocks.listRevisions.mock.calls.map(([request]) => request)).toEqual([
      expect.objectContaining({
        parent: "instances/i1/databases/db1",
        pageSize: 1,
        pageToken: "",
      }),
      expect.objectContaining({
        parent: "instances/i1/databases/db1",
        pageSize: 1,
        pageToken: "next-revision",
      }),
    ]);
    expect(
      store.getState().getRevisionsByDatabase("instances/i1/databases/db1")
    ).toEqual([revisionA, revisionB]);
  });

  test("deleteRevision removes the cached revision", async () => {
    mocks.deleteRevision.mockResolvedValue({});
    const store = createAppStore();
    store.setState({ revisionsByName: { [revisionA.name]: revisionA } });

    await store.getState().deleteRevision(revisionA.name);

    expect(store.getState().revisionsByName[revisionA.name]).toBeUndefined();
  });

  test("caches changelog list by database and view preference", async () => {
    mocks.listChangelogs.mockResolvedValue({ changelogs: [changelogA] });
    const fullChangelog = createProto(ChangelogSchema, {
      ...changelogA,
      schema: "full",
    });
    const store = createAppStore();

    const changelogs = await store.getState().listChangelogs({
      parent: "instances/i1/databases/db1",
      pageSize: 20,
      view: ChangelogView.BASIC,
    });

    expect(changelogs.changelogs).toEqual([changelogA]);
    expect(
      store.getState().changelogListByDatabase("instances/i1/databases/db1")
    ).toEqual([changelogA]);
    expect(store.getState().getChangelogByName(changelogA.name)).toBe(
      changelogA
    );

    store.setState({
      changelogsByCacheKey: {
        ...store.getState().changelogsByCacheKey,
        [`${changelogA.name}|${ChangelogView.FULL}`]: fullChangelog,
      },
    });

    expect(store.getState().getChangelogByName(changelogA.name)).toBe(
      fullChangelog
    );
  });

  test("keeps changelog list cache isolated by request dimensions", async () => {
    mocks.listChangelogs
      .mockResolvedValueOnce({ changelogs: [changelogA] })
      .mockResolvedValueOnce({ changelogs: [changelogB] });
    const store = createAppStore();

    await store.getState().listChangelogs({
      parent: "instances/i1/databases/db1",
      pageSize: 20,
      view: ChangelogView.BASIC,
    });
    const full = await store
      .getState()
      .getOrFetchChangelogListOfDatabase(
        "instances/i1/databases/db1",
        20,
        ChangelogView.FULL
      );

    expect(full).toEqual([changelogB]);
    expect(mocks.listChangelogs).toHaveBeenCalledTimes(2);
    expect(
      store.getState().changelogListByDatabase("instances/i1/databases/db1")
    ).toEqual([changelogA]);

    mocks.listChangelogs.mockReset();
    mocks.listChangelogs
      .mockResolvedValueOnce({ changelogs: [changelogB] })
      .mockResolvedValueOnce({ changelogs: [changelogA, changelogB] });
    const filteredStore = createAppStore();

    await filteredStore.getState().listChangelogs({
      parent: "instances/i1/databases/db1",
      pageSize: 20,
      view: ChangelogView.BASIC,
      filter: 'status == "DONE"',
    });
    const unfiltered = await filteredStore
      .getState()
      .getOrFetchChangelogListOfDatabase("instances/i1/databases/db1", 20);

    expect(unfiltered).toEqual([changelogA, changelogB]);
    expect(mocks.listChangelogs).toHaveBeenCalledTimes(2);
    expect(
      filteredStore
        .getState()
        .changelogListByDatabase("instances/i1/databases/db1")
    ).toEqual([changelogA, changelogB]);
  });

  test("deduplicates changelog fetches and skips unknown changelog names", async () => {
    mocks.getChangelog.mockResolvedValue(changelogA);
    const store = createAppStore();

    const missing = await store
      .getState()
      .getOrFetchChangelogByName("instances/i1/databases/db1/changelogs/-1");
    expect(missing).toBeUndefined();

    const [first, second] = await Promise.all([
      store.getState().getOrFetchChangelogByName(changelogA.name),
      store.getState().getOrFetchChangelogByName(changelogA.name),
    ]);

    expect(first).toBe(changelogA);
    expect(second).toBe(changelogA);
    expect(mocks.getChangelog).toHaveBeenCalledTimes(1);
    expect(
      store.getState().changelogRequests[
        `${changelogA.name}|${ChangelogView.BASIC}`
      ]
    ).toBeUndefined();
  });

  test("project webhook operations call through and lookup by id", async () => {
    const projectWithWebhook = createProto(ProjectSchema, {
      ...projectA,
      webhooks: [webhookA],
    });
    mocks.addWebhook.mockResolvedValue(projectWithWebhook);
    mocks.updateWebhook.mockResolvedValue(projectWithWebhook);
    mocks.removeWebhook.mockResolvedValue(projectA);
    mocks.testWebhook.mockResolvedValue({ error: "no route" });
    const store = createAppStore();

    expect(
      store
        .getState()
        .getProjectWebhookFromProjectById(projectWithWebhook, "hook-a")
    ).toBe(webhookA);

    await expect(
      store.getState().createProjectWebhook(projectA.name, webhookA)
    ).resolves.toBe(projectWithWebhook);
    expect(store.getState().projectsByName[projectWithWebhook.name]).toBe(
      projectWithWebhook
    );

    await expect(
      store.getState().updateProjectWebhook(webhookA, ["title"])
    ).resolves.toBe(projectWithWebhook);
    expect(store.getState().projectsByName[projectWithWebhook.name]).toBe(
      projectWithWebhook
    );

    await expect(store.getState().deleteProjectWebhook(webhookA)).resolves.toBe(
      projectA
    );
    expect(store.getState().projectsByName[projectA.name]).toBe(projectA);

    await expect(
      store.getState().testProjectWebhook(projectWithWebhook, webhookA)
    ).resolves.toEqual({ error: "no route" });

    expect(mocks.addWebhook).toHaveBeenCalledWith(
      expect.objectContaining({ project: projectA.name, webhook: webhookA })
    );
    expect(mocks.updateWebhook).toHaveBeenCalledWith(
      expect.objectContaining({
        webhook: webhookA,
        updateMask: expect.objectContaining({ paths: ["title"] }),
      })
    );
    expect(mocks.removeWebhook).toHaveBeenCalledWith(
      expect.objectContaining({ webhook: webhookA })
    );
    expect(mocks.testWebhook).toHaveBeenCalledWith(
      expect.objectContaining({ project: projectA.name, webhook: webhookA })
    );
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
    expect(store.getState().identityProviderList()).toEqual([
      identityProviderA,
    ]);
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
        updateMask: expect.objectContaining({ paths: ["title"] }),
      })
    );
  });

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
    expect(store.getState().accessGrantsByName[accessGrantA.name]).toBe(
      revoked
    );
  });

  test("updateIssueComment writes the server response so the edited marker shows in place (BYT-9746)", async () => {
    const issueName = "projects/a/issues/1";
    const commentName = `${issueName}/issueComments/100`;
    const original = createProto(IssueCommentSchema, {
      name: commentName,
      comment: "before",
      createTime: timestampSeconds(1000),
      updateTime: timestampSeconds(1000),
    });
    // The server bumps updated_at on edit and returns the refreshed comment.
    const serverUpdated = createProto(IssueCommentSchema, {
      name: commentName,
      comment: "after",
      createTime: timestampSeconds(1000),
      updateTime: timestampSeconds(2000),
    });
    mocks.updateIssueComment.mockResolvedValue(serverUpdated);
    const store = createAppStore();
    store.setState({ issueCommentsByIssue: { [issueName]: [original] } });

    await store.getState().updateIssueComment({
      issueCommentName: commentName,
      comment: "after",
    });

    const cached = store.getState().getIssueComments(issueName)[0];
    expect(cached.comment).toBe("after");
    // Regression guard: the optimistic patch used to discard the RPC response
    // and keep the stale updateTime, so IssueCommentActivity's
    // `isEdited = createTime !== updateTime` stayed false until a full refetch.
    expect(cached.updateTime).toEqual(serverUpdated.updateTime);
    expect(cached.updateTime).not.toEqual(cached.createTime);
  });

  test("deduplicates project fetches and caches the result", async () => {
    mocks.getProject.mockResolvedValue(projectA);
    const store = createAppStore();

    const [first, second] = await Promise.all([
      store.getState().fetchProject(projectA.name),
      store.getState().fetchProject(projectA.name),
    ]);

    expect(first).toBe(projectA);
    expect(second).toBe(projectA);
    expect(mocks.getProject).toHaveBeenCalledTimes(1);
    expect(store.getState().projectsByName[projectA.name]).toBe(projectA);
  });

  test("falls back to individual project fetches when batch fetch fails", async () => {
    mocks.batchGetProjects.mockRejectedValue(new Error("batch failed"));
    mocks.getProject.mockImplementation(({ name }: { name: string }) => {
      return Promise.resolve(name === projectA.name ? projectA : projectB);
    });
    const store = createAppStore();

    const projects = await store
      .getState()
      .batchFetchProjects([projectA.name, projectB.name]);

    expect(projects).toEqual([projectA, projectB]);
    expect(mocks.batchGetProjects).toHaveBeenCalledTimes(1);
    expect(mocks.getProject).toHaveBeenCalledTimes(2);
  });

  test("writes created projects into the entity cache", async () => {
    mocks.createProject.mockResolvedValue(projectA);
    const store = createAppStore();

    const created = await store.getState().createProject("A", "a");

    expect(created).toBe(projectA);
    expect(store.getState().projectsByName[projectA.name]).toBe(projectA);
  });

  test("checks workspace permission through user, groups, and roles", () => {
    const store = createAppStore();
    store.setState({
      currentUser: user,
      roles: [
        createProto(RoleSchema, {
          name: "roles/admin",
          permissions: ["bb.projects.create"],
        }),
        createProto(RoleSchema, {
          name: "roles/viewer",
          permissions: ["bb.projects.get"],
        }),
      ],
      workspacePolicy: createProto(IamPolicySchema, {
        bindings: [
          createProto(BindingSchema, {
            role: "roles/admin",
            members: ["group:dba"],
          }),
          createProto(BindingSchema, {
            role: "roles/viewer",
            members: ["user:allUsers"],
          }),
        ],
      }),
    });

    expect(store.getState().hasWorkspacePermission("bb.projects.create")).toBe(
      true
    );
    expect(store.getState().hasWorkspacePermission("bb.settings.set")).toBe(
      false
    );
  });

  test("ignores expired IAM bindings when checking workspace permission", () => {
    const store = createAppStore();
    const yesterday = new Date(Date.now() - 24 * 60 * 60 * 1000).toISOString();
    store.setState({
      currentUser: user,
      roles: [
        createProto(RoleSchema, {
          name: "roles/admin",
          permissions: ["bb.projects.create"],
        }),
      ],
      workspacePolicy: createProto(IamPolicySchema, {
        bindings: [
          createProto(BindingSchema, {
            role: "roles/admin",
            members: ["group:dba"],
            condition: createProto(ExprSchema, {
              expression: `request.time < timestamp("${yesterday}")`,
            }),
          }),
        ],
      }),
    });

    expect(store.getState().hasWorkspacePermission("bb.projects.create")).toBe(
      false
    );
  });

  test("derives app features from the workspace profile setting", async () => {
    mocks.getSetting.mockResolvedValue(
      createProto(SettingSchema, {
        value: createProto(SettingValueSchema, {
          value: {
            case: "workspaceProfile",
            value: createProto(WorkspaceProfileSettingSchema, {
              databaseChangeMode: DatabaseChangeMode.EDITOR,
            }),
          },
        }),
      })
    );
    const store = createAppStore();

    await store.getState().loadWorkspaceProfile();

    expect(store.getState().appFeatures["bb.feature.hide-quick-start"]).toBe(
      true
    );
  });

  test("uses pipeline app feature defaults before profile overrides", () => {
    const store = createAppStore();

    expect(
      store.getState().appFeatures["bb.feature.database-change-mode"]
    ).toBe(DatabaseChangeMode.PIPELINE);
    expect(store.getState().appFeatures["bb.feature.hide-quick-start"]).toBe(
      false
    );
    expect(store.getState().appFeatures["bb.feature.hide-trial"]).toBe(false);
  });

  test("derives actuator state for React consumers", () => {
    const store = createAppStore();
    store.setState({
      serverInfo: createProto(ActuatorInfoSchema, {
        workspace: "workspaces/default",
        version: "3.10.2",
        externalUrl: "",
        saas: true,
        activatedInstanceCount: 2,
        totalInstanceCount: 3,
        userCountInIam: 7,
      }),
    });

    expect(store.getState().isSaaSMode()).toBe(true);
    expect(store.getState().workspaceResourceName()).toBe("workspaces/default");
    expect(store.getState().needConfigureExternalUrl()).toBe(true);
    expect(store.getState().changelogURL()).toBe(
      "https://docs.bytebase.com/changelog/bytebase-3-10-2/"
    );
    expect(store.getState().activatedInstanceCount()).toBe(2);
    expect(store.getState().totalInstanceCount()).toBe(3);
    expect(store.getState().userCountInIam()).toBe(7);
  });

  test("derives subscription limits and feature state", () => {
    const store = createAppStore();
    store.setState({
      subscription: createProto(SubscriptionSchema, {
        plan: PlanType.ENTERPRISE,
        seats: -1,
        instances: -1,
        activeInstances: -1,
        expiresTime: timestampSeconds(
          Math.floor(Date.now() / 1000) + 60 * 60 * 24 * 30
        ),
      }),
    });

    expect(store.getState().currentPlan()).toBe(PlanType.ENTERPRISE);
    expect(store.getState().isFreePlan()).toBe(false);
    expect(store.getState().isExpired()).toBe(false);
    expect(store.getState().showTrial()).toBe(false);
    expect(store.getState().instanceCountLimit()).toBe(Number.MAX_VALUE);
    expect(store.getState().userCountLimit()).toBe(Number.MAX_VALUE);
    expect(store.getState().instanceLicenseCount()).toBe(Number.MAX_VALUE);
    expect(store.getState().hasUnifiedInstanceLicense()).toBe(true);
    expect(store.getState().hasFeature(PlanFeature.FEATURE_DATA_MASKING)).toBe(
      true
    );
    expect(
      store.getState().hasInstanceFeature(
        PlanFeature.FEATURE_DATA_MASKING,
        createProto(InstanceSchema, {
          name: "instances/prod",
          activation: false,
        })
      )
    ).toBe(true);
    expect(
      store.getState().instanceMissingLicense(
        PlanFeature.FEATURE_DATA_MASKING,
        createProto(InstanceSchema, {
          name: "instances/prod",
          activation: false,
        })
      )
    ).toBe(false);

    store.setState({
      subscription: createProto(SubscriptionSchema, {
        plan: PlanType.ENTERPRISE,
        instances: 50,
        activeInstances: 20,
        expiresTime: timestampSeconds(
          Math.floor(Date.now() / 1000) + 60 * 60 * 24 * 30
        ),
      }),
    });
    expect(store.getState().hasUnifiedInstanceLicense()).toBe(false);
    expect(
      store.getState().hasInstanceFeature(
        PlanFeature.FEATURE_DATA_MASKING,
        createProto(InstanceSchema, {
          name: "instances/prod",
          activation: false,
        })
      )
    ).toBe(false);
    expect(
      store.getState().instanceMissingLicense(
        PlanFeature.FEATURE_DATA_MASKING,
        createProto(InstanceSchema, {
          name: "instances/prod",
          activation: false,
        })
      )
    ).toBe(true);
  });

  test("treats expired subscriptions as feature unavailable", () => {
    const store = createAppStore();
    store.setState({
      subscription: createProto(SubscriptionSchema, {
        plan: PlanType.ENTERPRISE,
        expiresTime: timestampSeconds(
          Math.floor(Date.now() / 1000) - 60 * 60 * 24
        ),
      }),
    });

    expect(store.getState().isExpired()).toBe(true);
    expect(store.getState().hasFeature(PlanFeature.FEATURE_DATA_MASKING)).toBe(
      false
    );
  });

  test("refreshes subscription state after external mutations", async () => {
    mocks.getSubscription.mockResolvedValue(
      createProto(SubscriptionSchema, {
        plan: PlanType.TEAM,
        seats: 12,
      })
    );
    const store = createAppStore();
    store.setState({
      subscription: createProto(SubscriptionSchema, {
        plan: PlanType.FREE,
        seats: 0,
      }),
    });

    const subscription = await store.getState().refreshSubscription();

    expect(subscription?.plan).toBe(PlanType.TEAM);
    expect(store.getState().currentPlan()).toBe(PlanType.TEAM);
    expect(store.getState().userCountLimit()).toBe(12);
  });

  test("loads environment settings into React state", async () => {
    mocks.getSetting.mockResolvedValue(
      createProto(SettingSchema, {
        value: createProto(SettingValueSchema, {
          value: {
            case: "environment",
            value: createProto(EnvironmentSettingSchema, {
              environments: [
                createProto(EnvironmentSetting_EnvironmentSchema, {
                  id: "dev",
                  title: "Development",
                  color: "#00aa00",
                }),
                createProto(EnvironmentSetting_EnvironmentSchema, {
                  id: "prod",
                  title: "Production",
                  tags: { protected: "protected" },
                }),
              ],
            }),
          },
        }),
      })
    );
    const store = createAppStore();

    const environments = await store.getState().loadEnvironmentList();

    expect(environments.map((env) => env.name)).toEqual([
      "environments/dev",
      "environments/prod",
    ]);
    expect(store.getState().environmentList[1]).toMatchObject({
      id: "prod",
      order: 1,
      tags: { protected: "protected" },
    });
  });

  test("refreshes environment settings after cache warm-up", async () => {
    mocks.getSetting
      .mockResolvedValueOnce(
        createProto(SettingSchema, {
          value: createProto(SettingValueSchema, {
            value: {
              case: "environment",
              value: createProto(EnvironmentSettingSchema, {
                environments: [
                  createProto(EnvironmentSetting_EnvironmentSchema, {
                    id: "dev",
                    title: "Development",
                  }),
                ],
              }),
            },
          }),
        })
      )
      .mockResolvedValueOnce(
        createProto(SettingSchema, {
          value: createProto(SettingValueSchema, {
            value: {
              case: "environment",
              value: createProto(EnvironmentSettingSchema, {
                environments: [
                  createProto(EnvironmentSetting_EnvironmentSchema, {
                    id: "prod",
                    title: "Production",
                  }),
                ],
              }),
            },
          }),
        })
      );
    const store = createAppStore();

    await store.getState().loadEnvironmentList();
    await store.getState().loadEnvironmentList();
    expect(mocks.getSetting).toHaveBeenCalledTimes(1);

    const environments = await store.getState().refreshEnvironmentList();

    expect(mocks.getSetting).toHaveBeenCalledTimes(2);
    expect(environments.map((env) => env.name)).toEqual(["environments/prod"]);
  });

  test("loads project IAM policy and checks project permissions", async () => {
    mocks.getCurrentUser.mockResolvedValue(user);
    mocks.getActuatorInfo.mockResolvedValue({
      workspace: "workspaces/default",
    });
    mocks.getWorkspace.mockResolvedValue({ name: "workspaces/default" });
    mocks.listRoles.mockResolvedValue({
      roles: [
        createProto(RoleSchema, {
          name: "roles/projectDeveloper",
          permissions: ["bb.projects.update"],
        }),
      ],
    });
    mocks.getIamPolicy.mockResolvedValue(createProto(IamPolicySchema, {}));
    mocks.getProjectIamPolicy.mockResolvedValue(
      createProto(IamPolicySchema, {
        bindings: [
          createProto(BindingSchema, {
            role: "roles/projectDeveloper",
            members: ["user:alice@example.com"],
          }),
        ],
      })
    );
    const store = createAppStore();

    await store.getState().loadProjectIamPolicy(projectA.name);

    expect(mocks.getProjectIamPolicy).toHaveBeenCalledTimes(1);
    expect(
      store.getState().hasProjectPermission(projectA, "bb.projects.update")
    ).toBe(true);
    expect(
      store.getState().hasProjectPermission(projectA, "bb.projects.delete")
    ).toBe(false);
  });

  test("caches instances by resource name", async () => {
    const instance = createProto(InstanceSchema, {
      name: "instances/prod",
      title: "Prod",
    });
    mocks.getInstance.mockResolvedValue(instance);
    const store = createAppStore();

    const [first, second] = await Promise.all([
      store.getState().fetchInstance(instance.name),
      store.getState().fetchInstance(instance.name),
    ]);

    expect(first).toBe(instance);
    expect(second).toBe(instance);
    expect(mocks.getInstance).toHaveBeenCalledTimes(1);
    expect(store.getState().instancesByName[instance.name]).toBe(instance);
  });

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

  test("fetchDatabases populates databasesByName from the list response", async () => {
    const dbA = createProto(DatabaseSchema$, {
      name: "instances/i1/databases/db1",
    });
    const dbB = createProto(DatabaseSchema$, {
      name: "instances/i1/databases/db2",
    });
    mocks.listDatabases.mockResolvedValue({
      databases: [dbA, dbB],
      nextPageToken: "next",
    });
    const store = createAppStore();

    const result = await store.getState().fetchDatabases({
      parent: "projects/a",
      pageSize: 50,
      filter: 'project == "projects/a"',
    });

    expect(result.databases).toEqual([dbA, dbB]);
    expect(result.nextPageToken).toBe("next");
    expect(store.getState().databasesByName[dbA.name]).toEqual(dbA);
    expect(store.getState().databasesByName[dbB.name]).toEqual(dbB);
    expect(mocks.listDatabases).toHaveBeenCalledTimes(1);
  });

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

  test("deduplicates sheet fetches and caches the result", async () => {
    const name = "projects/p1/sheets/s1";
    const sheet = createProto(SheetSchema, {
      name,
      content: new Uint8Array(),
    });
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

  test("deduplicates instance role fetches per instance", async () => {
    const instance = "instances/i1";
    const roles = [
      createProto(InstanceRoleSchema, { name: `${instance}/roles/admin` }),
    ];
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

  test("notify() routes to the React toast manager", async () => {
    const { toastManager } = await import("@/react/lib/toast");
    const addSpy = vi.spyOn(toastManager, "add");
    const store = createAppStore();

    store.getState().notify({
      module: "bytebase",
      style: "SUCCESS",
      title: "Saved",
    });

    expect(addSpy).toHaveBeenCalledTimes(1);
    expect(addSpy.mock.calls[0][0]).toMatchObject({
      title: "Saved",
      type: "success",
    });
    addSpy.mockRestore();
  });

  test("keeps user-scoped preferences in localStorage", () => {
    const store = createAppStore();
    const listener = vi.fn();
    store.setState({ currentUser: user });
    window.addEventListener(ReactShellBridgeEvent.quickstartReset, listener);

    store.getState().setRecentProject(projectA.name);
    store.getState().setRecentProject(projectB.name);
    store.getState().recordRecentVisit("/projects/a?tab=1");
    store.getState().recordRecentVisit("/projects/a?tab=2");
    store.getState().removeRecentVisit("/missing");
    store.getState().resetQuickstartProgress();

    // Not SaaS in this test, so keys are workspace-agnostic (scope "").
    expect(
      JSON.parse(
        localStorage.getItem(storageKeyRecentProjects("", user.email))!
      )
    ).toEqual([projectB.name, projectA.name]);
    expect(
      JSON.parse(localStorage.getItem(storageKeyRecentVisit("", user.email))!)
    ).toEqual(["/projects/a?tab=2"]);
    expect(
      JSON.parse(localStorage.getItem(storageKeyIntroState(user.email))!)
    ).toMatchObject({
      hidden: false,
      "project.visit": false,
      "data.query": false,
    });
    expect(listener).toHaveBeenCalledWith(
      expect.objectContaining({
        detail: expect.objectContaining({
          keys: expect.arrayContaining(["hidden", "data.query"]),
        }),
      })
    );
    window.removeEventListener(ReactShellBridgeEvent.quickstartReset, listener);
  });

  test("scopes recent projects by workspace in SaaS mode", () => {
    const store = createAppStore();
    // SaaS mode scopes cache keys by workspace.
    store.setState({
      serverInfo: createProto(ActuatorInfoSchema, { saas: true }),
    });
    store.setState({ currentUser: { ...user, workspace: "workspaces/a" } });
    store.getState().setRecentProject(projectA.name);
    store.getState().recordRecentVisit("/projects/a");

    // Same user, different workspace (e.g. after a SaaS workspace switch).
    store.setState({ currentUser: { ...user, workspace: "workspaces/b" } });
    store.getState().setRecentProject(projectB.name);
    store.getState().recordRecentVisit("/projects/b");

    expect(
      JSON.parse(
        localStorage.getItem(
          storageKeyRecentProjects("workspaces/a", user.email)
        )!
      )
    ).toEqual([projectA.name]);
    expect(
      JSON.parse(
        localStorage.getItem(
          storageKeyRecentProjects("workspaces/b", user.email)
        )!
      )
    ).toEqual([projectB.name]);
    // Recent-visit is isolated per workspace too.
    expect(
      JSON.parse(
        localStorage.getItem(storageKeyRecentVisit("workspaces/a", user.email))!
      )
    ).toEqual(["/projects/a"]);
    expect(
      JSON.parse(
        localStorage.getItem(storageKeyRecentVisit("workspaces/b", user.email))!
      )
    ).toEqual(["/projects/b"]);
  });

  test("caches database metadata and reuses inflight request", async () => {
    const {
      DatabaseMetadataSchema,
      SchemaMetadataSchema,
      TableMetadataSchema,
    } = await import("@/types/proto-es/v1/database_service_pb");
    const metadata = createProto(DatabaseMetadataSchema, {
      name: "instances/i1/databases/db1/metadata",
      schemas: [
        createProto(SchemaMetadataSchema, {
          name: "public",
          tables: [createProto(TableMetadataSchema, { name: "users" })],
        }),
      ],
    });
    mocks.getDatabaseMetadata.mockResolvedValue(metadata);
    const store = createAppStore();

    const [first, second] = await Promise.all([
      store
        .getState()
        .getOrFetchDatabaseMetadata({ database: "instances/i1/databases/db1" }),
      store
        .getState()
        .getOrFetchDatabaseMetadata({ database: "instances/i1/databases/db1" }),
    ]);

    expect(first).toBe(metadata);
    expect(second).toBe(metadata);
    // Concurrent calls dedupe through `metadataRequests`.
    expect(mocks.getDatabaseMetadata).toHaveBeenCalledTimes(1);

    // Sync derived getters resolve through the cached metadata.
    expect(
      store.getState().getTableMetadata({
        database: "instances/i1/databases/db1",
        schema: "public",
        table: "users",
      }).name
    ).toBe("users");
    // Unknown table falls back to an empty TableMetadata placeholder
    // (mirrors the legacy Pinia store's behavior).
    expect(
      store.getState().getTableMetadata({
        database: "instances/i1/databases/db1",
        schema: "public",
        table: "missing",
      }).name
    ).toBe("");
  });

  test("keeps metadata cache isolated by filter and limit", async () => {
    const { DatabaseMetadataSchema, SchemaMetadataSchema } = await import(
      "@/types/proto-es/v1/database_service_pb"
    );
    const fullMetadata = createProto(DatabaseMetadataSchema, {
      name: "instances/i1/databases/db1/metadata",
      schemas: [createProto(SchemaMetadataSchema, { name: "public" })],
    });
    const filteredMetadata = createProto(DatabaseMetadataSchema, {
      name: "instances/i1/databases/db1/metadata",
      schemas: [createProto(SchemaMetadataSchema, { name: "filtered" })],
    });
    mocks.getDatabaseMetadata
      .mockResolvedValueOnce(fullMetadata)
      .mockResolvedValueOnce(filteredMetadata);
    const store = createAppStore();

    await store
      .getState()
      .getOrFetchDatabaseMetadata({ database: "instances/i1/databases/db1" });
    await store.getState().getOrFetchDatabaseMetadata({
      database: "instances/i1/databases/db1",
      filter: `schema == "filtered"`,
    });

    expect(mocks.getDatabaseMetadata).toHaveBeenCalledTimes(2);
  });

  describe("getDBGroupByName / getOrFetchDBGroupByName cache semantics", () => {
    test("getDBGroupByName returns unknownDatabaseGroup when absent", () => {
      const store = createAppStore();
      const group = store
        .getState()
        .getDBGroupByName("projects/p/databaseGroups/x");
      expect(isValidDatabaseGroupName(group.name)).toBe(false);
    });

    test("getDBGroupByName with FULL view misses a BASIC-only cache entry", () => {
      const store = createAppStore();
      const name = "projects/p/databaseGroups/g";
      store.setState({
        dbGroupsByName: {
          [name]: createProto(DatabaseGroupSchema, { name, title: "g" }),
        },
        dbGroupViewByName: { [name]: DatabaseGroupView.BASIC },
      });
      expect(store.getState().getDBGroupByName(name).name).toBe(name);
      const full = store
        .getState()
        .getDBGroupByName(name, DatabaseGroupView.FULL);
      expect(isValidDatabaseGroupName(full.name)).toBe(false);
    });

    test("getOrFetchDBGroupByName returns cached entry without a request", async () => {
      const store = createAppStore();
      const name = "projects/p/databaseGroups/g";
      store.setState({
        dbGroupsByName: {
          [name]: createProto(DatabaseGroupSchema, { name, title: "g" }),
        },
        dbGroupViewByName: { [name]: DatabaseGroupView.BASIC },
      });
      const group = await store.getState().getOrFetchDBGroupByName(name);
      expect(group.name).toBe(name);
    });
  });
});

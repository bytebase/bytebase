// @vitest-environment node
import { create } from "@bufbuild/protobuf";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { createStore, type StoreApi } from "zustand/vanilla";
import { silentContextKey } from "@/api/context-key";
import { createIamSlice } from "@/stores/app/iam";
import type { AppStoreState } from "@/stores/app/types";
import { DatabaseSchema$ } from "@/types/proto-es/v1/database_service_pb";
import { IamPolicySchema } from "@/types/proto-es/v1/iam_policy_pb";
import { RoleSchema } from "@/types/proto-es/v1/role_service_pb";
import { UserSchema } from "@/types/proto-es/v1/user_service_pb";

const api = vi.hoisted(() => ({
  getDatabase: vi.fn(),
  getCurrentUser: vi.fn(),
  listRoles: vi.fn(),
  getWorkspacePolicy: vi.fn(),
  getProjectPolicy: vi.fn(),
}));
const app = vi.hoisted(() => ({ getState: vi.fn(), setState: vi.fn() }));
vi.mock("@/stores/app", () => ({ useAppStore: app }));
vi.mock("@/api", () => ({
  databaseServiceClientConnect: { getDatabase: api.getDatabase },
  userServiceClientConnect: { getCurrentUser: api.getCurrentUser },
  roleServiceClientConnect: { listRoles: api.listRoles },
  workspaceServiceClientConnect: { getIamPolicy: api.getWorkspacePolicy },
  projectServiceClientConnect: { getIamPolicy: api.getProjectPolicy },
}));

import { checkPreparedApiPermission } from "./apiPermissionPreflight";

const call = (service: string, method: string, body: object = {}) => ({
  operationId: `bytebase.v1.${service}.${method}`,
  path: `/bytebase.v1.${service}/${method}`,
  body: JSON.stringify(body),
});
const projectCall = call("ProjectService", "UpdateProject", {
  project: { name: "projects/example" },
});
const workspaceCall = call("InstanceService", "ListInstances");
const catalogCall = call("DatabaseCatalogService", "UpdateDatabaseCatalog", {
  catalog: { name: "instances/i/databases/db/catalog" },
});
const user = create(UserSchema, {
  name: "users/member@example.com",
  workspace: "workspaces/example",
});
const role = create(RoleSchema, { name: "roles/developer" });
const emptyPolicy = () => create(IamPolicySchema);
const memberPolicy = (condition = "", member = "user:member@example.com") =>
  create(IamPolicySchema, {
    bindings: [
      {
        role: role.name,
        members: [member],
        condition: { expression: condition },
      },
    ],
  });
const permittedRole = (permission: string) =>
  create(RoleSchema, { ...role, permissions: [permission] });
let store: StoreApi<AppStoreState>;

describe("checkPreparedApiPermission", () => {
  beforeEach(() => {
    vi.resetAllMocks();
    store = createStore<AppStoreState>()(
      (...args) =>
        ({
          ...createIamSlice(...args),
          currentUser: user,
          serverInfo: { workspace: "workspaces/example" },
          workspacePolicy: emptyPolicy(),
          roles: [role],
          projectPoliciesByName: { "projects/example": emptyPolicy() },
          projectsByName: { "projects/example": { name: "projects/example" } },
          databasesByName: {},
        }) as unknown as AppStoreState
    );
    app.getState.mockImplementation(store.getState);
    app.setState.mockImplementation(store.setState);
    api.getCurrentUser.mockResolvedValue(user);
    api.listRoles.mockResolvedValue({ roles: [role] });
    api.getWorkspacePolicy.mockImplementation(async () => emptyPolicy());
    api.getProjectPolicy.mockImplementation(async () => emptyPolicy());
    api.getDatabase.mockResolvedValue({
      name: "instances/i/databases/db",
      project: "projects/example",
    });
  });

  it("refreshes a complete cached denial before blocking a workspace API", async () => {
    const result = await checkPreparedApiPermission(workspaceCall);
    expect(result.kind).toBe("denied");
    if (result.kind !== "denied") throw new Error("Expected denial");
    expect(JSON.parse(result.result)).toMatchObject({
      status: 403,
      response: {
        requiredPermissions: ["bb.instances.list"],
        resources: ["workspaces/example"],
      },
    });
    expect(api.getCurrentUser).toHaveBeenCalledOnce();
    expect(api.listRoles).toHaveBeenCalledOnce();
    expect(api.getWorkspacePolicy).toHaveBeenCalledWith(
      expect.objectContaining({ resource: "workspaces/example" }),
      expect.anything()
    );
  });

  it("blocks a project API only after refreshing its IAM policy", async () => {
    const result = await checkPreparedApiPermission(projectCall);
    expect(result.kind).toBe("denied");
    if (result.kind !== "denied") throw new Error("Expected denial");
    expect(JSON.parse(result.result).response).toEqual({
      code: "permission_denied",
      requiredPermissions: ["bb.projects.update"],
      resources: ["projects/example"],
    });
    expect(api.getProjectPolicy).toHaveBeenCalledWith(
      expect.objectContaining({ resource: "projects/example" }),
      expect.anything()
    );
  });

  it("does not refresh an existing grant because the backend still authorizes the call", async () => {
    store.setState({
      roles: [permittedRole("bb.projects.update")],
      projectPoliciesByName: { "projects/example": memberPolicy() },
    });
    expect(await checkPreparedApiPermission(projectCall)).toEqual({
      kind: "allowed",
    });
    expect(api.getCurrentUser).not.toHaveBeenCalled();
  });

  it("recognizes a project permission granted after the cached denial", async () => {
    api.listRoles.mockResolvedValue({
      roles: [permittedRole("bb.projects.update")],
    });
    api.getProjectPolicy.mockResolvedValue(memberPolicy());
    expect(await checkPreparedApiPermission(projectCall)).toEqual({
      kind: "allowed",
    });
  });

  it("recognizes a workspace permission granted after the cached denial", async () => {
    api.listRoles.mockResolvedValue({
      roles: [permittedRole("bb.instances.list")],
    });
    api.getWorkspacePolicy.mockResolvedValue(memberPolicy());
    expect(await checkPreparedApiPermission(workspaceCall)).toEqual({
      kind: "allowed",
    });
  });

  it("refreshes user groups as well as policies and role definitions", async () => {
    api.getCurrentUser.mockResolvedValue(
      create(UserSchema, { ...user, groups: ["groups/engineering"] })
    );
    api.listRoles.mockResolvedValue({
      roles: [permittedRole("bb.projects.update")],
    });
    api.getProjectPolicy.mockResolvedValue(
      memberPolicy("", "group:engineering")
    );
    expect(await checkPreparedApiPermission(projectCall)).toEqual({
      kind: "allowed",
    });
  });

  it("does not reject a valid disjunctive CEL grant", async () => {
    api.listRoles.mockResolvedValue({
      roles: [permittedRole("bb.projects.update")],
    });
    api.getProjectPolicy.mockResolvedValue(
      memberPolicy(
        'request.time < timestamp("2000-01-01T00:00:00Z") || request.time < timestamp("2099-01-01T00:00:00Z")'
      )
    );
    expect(await checkPreparedApiPermission(projectCall)).toEqual({
      kind: "allowed",
    });
  });

  it.each([
    "getCurrentUser",
    "listRoles",
    "getWorkspacePolicy",
    "getProjectPolicy",
  ] as const)(
    "defers to backend when %s cannot refresh negative evidence",
    async (method) => {
      api[method].mockRejectedValueOnce(new Error("unavailable"));
      expect(await checkPreparedApiPermission(projectCall)).toEqual({
        kind: "allowed",
      });
    }
  );

  it("does not publish refreshed IAM into a different workspace", async () => {
    api.getProjectPolicy.mockImplementation(async () => {
      store.setState({
        serverInfo: {
          workspace: "workspaces/other",
        } as AppStoreState["serverInfo"],
      });
      return memberPolicy();
    });
    expect(await checkPreparedApiPermission(projectCall)).toEqual({
      kind: "allowed",
    });
    expect(
      store.getState().projectPoliciesByName["projects/example"]?.bindings
    ).toEqual([]);
  });

  it("does not publish refreshed IAM after the signed-in user changes", async () => {
    api.getProjectPolicy.mockImplementation(async () => {
      store.setState({
        currentUser: create(UserSchema, {
          ...user,
          name: "users/other@example.com",
        }),
      });
      return memberPolicy();
    });
    expect(await checkPreparedApiPermission(projectCall)).toEqual({
      kind: "allowed",
    });
    expect(store.getState().currentUser?.name).toBe("users/other@example.com");
    expect(
      store.getState().projectPoliciesByName["projects/example"]?.bindings
    ).toEqual([]);
  });

  it("defers to backend when the authenticated response identifies another user", async () => {
    api.getCurrentUser.mockResolvedValue(
      create(UserSchema, { ...user, name: "users/other@example.com" })
    );
    expect(await checkPreparedApiPermission(projectCall)).toEqual({
      kind: "allowed",
    });
    expect(store.getState().currentUser?.name).toBe(user.name);
  });

  it("does not infer a permission for a custom-auth API", async () => {
    expect(
      await checkPreparedApiPermission(
        call("DatabaseService", "ListDatabases", { parent: "projects/example" })
      )
    ).toEqual({ kind: "allowed" });
    expect(api.getCurrentUser).not.toHaveBeenCalled();
  });

  it.each([
    call("DatabaseService", "UpdateDatabase", {
      database: { name: "instances/i/databases/db" },
    }),
    call("DatabaseService", "BatchUpdateDatabases", {
      parent: "instances/i",
      requests: [{ database: { name: "instances/i/databases/db" } }],
    }),
    call("DatabaseService", "BatchSyncDatabases", {
      parent: "instances/i",
      names: ["instances/i/databases/db"],
    }),
    call("IssueService", "BatchUpdateIssuesStatus", {
      parent: "projects/-",
      issues: ["projects/example/issues/1"],
      status: "OPEN",
    }),
  ])(
    "defers $operationId dynamic authorization to backend",
    async (prepared) => {
      expect(await checkPreparedApiPermission(prepared)).toEqual({
        kind: "allowed",
      });
      expect(app.getState).not.toHaveBeenCalled();
    }
  );

  it("does not block when missing IAM evidence cannot be read", async () => {
    store.setState({ workspacePolicy: undefined });
    api.getWorkspacePolicy.mockRejectedValueOnce(new Error("unavailable"));
    expect(await checkPreparedApiPermission(workspaceCall)).toEqual({
      kind: "allowed",
    });
  });

  it("defers authorization when database ownership cannot be read", async () => {
    api.getDatabase.mockRejectedValueOnce(new Error("unavailable"));
    expect(await checkPreparedApiPermission(catalogCall)).toEqual({
      kind: "allowed",
    });
    expect(api.getCurrentUser).not.toHaveBeenCalled();
  });

  it("uses current database ownership rather than a stale cache", async () => {
    store.setState({
      databasesByName: {
        "instances/i/databases/db": create(DatabaseSchema$, {
          name: "instances/i/databases/db",
          project: "projects/old",
        }),
      },
    });
    const result = await checkPreparedApiPermission(catalogCall);
    expect(result.kind).toBe("denied");
    if (result.kind !== "denied") throw new Error("Expected denial");
    expect(JSON.parse(result.result).response.resources).toEqual([
      "projects/example",
    ]);
    expect(
      api.getDatabase.mock.calls[0][1].contextValues.get(silentContextKey)
    ).toBe(true);
  });

  it("does not deny when database ownership returns a different resource", async () => {
    api.getDatabase.mockResolvedValueOnce({
      name: "instances/i/databases/other",
      project: "projects/example",
    });
    expect(await checkPreparedApiPermission(catalogCall)).toEqual({
      kind: "allowed",
    });
    expect(api.getCurrentUser).not.toHaveBeenCalled();
  });

  it("checks update and create permission for allowMissing", async () => {
    const result = await checkPreparedApiPermission(
      call("InstanceService", "UpdateInstance", {
        instance: { name: "instances/i" },
        allowMissing: true,
      })
    );
    expect(result.kind).toBe("denied");
    if (result.kind !== "denied") throw new Error("Expected denial");
    expect(JSON.parse(result.result).response.requiredPermissions).toEqual([
      "bb.instances.update",
      "bb.instances.create",
    ]);
  });

  it("ignores project-looking connection parameters outside the authorization target", async () => {
    store.setState({
      roles: [permittedRole("bb.instances.update")],
      projectPoliciesByName: {
        "projects/example": memberPolicy(),
        "projects/other": emptyPolicy(),
      },
    });
    expect(
      await checkPreparedApiPermission(
        call("InstanceService", "UpdateInstance", {
          instance: {
            name: "projects/example/instances/i",
            title: "Renamed instance",
            dataSources: [
              { extraConnectionParameters: { project: "projects/other" } },
            ],
          },
          updateMask: "title",
        })
      )
    ).toEqual({ kind: "allowed" });
    expect(api.getProjectPolicy).not.toHaveBeenCalled();
  });

  it("does not guess a dynamic permission for inline instance database listing", async () => {
    expect(
      await checkPreparedApiPermission(
        call("InstanceService", "ListInstanceDatabase", {
          name: "instances/i",
          instance: { name: "instances/i" },
        })
      )
    ).toEqual({ kind: "allowed" });
    expect(app.getState).not.toHaveBeenCalled();
  });
});

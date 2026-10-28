// @vitest-environment node
import { beforeEach, describe, expect, it, vi } from "vitest";
import { silentContextKey } from "@/api/context-key";

const getState = vi.hoisted(() => vi.fn());
const getDatabase = vi.hoisted(() => vi.fn());
vi.mock("@/stores/app", () => ({ useAppStore: { getState } }));
vi.mock("@/api", () => ({ databaseServiceClientConnect: { getDatabase } }));

import { checkPreparedApiPermission } from "./apiPermissionPreflight";

const projectCall = {
  operationId: "bytebase.v1.ProjectService.UpdateProject",
  path: "/bytebase.v1.ProjectService/UpdateProject",
  body: '{"project":{"name":"projects/example"}}',
};
const workspaceCall = {
  operationId: "bytebase.v1.InstanceService.ListInstances",
  path: "/bytebase.v1.InstanceService/ListInstances",
  body: "{}",
};

const state = {
  currentUser: { name: "users/1" },
  workspacePolicy: { bindings: [] },
  roles: [{ name: "roles/developer", permissions: [] }],
  projectPoliciesByName: { "projects/example": { bindings: [] } },
  projectsByName: { "projects/example": { name: "projects/example" } },
  databasesByName: {},
  loadWorkspacePermissionState: vi.fn().mockResolvedValue(undefined),
  loadProjectIamPolicy: vi.fn().mockResolvedValue(undefined),
  hasWorkspacePermission: vi.fn().mockReturnValue(false),
  hasProjectPermission: vi.fn().mockReturnValue(false),
};

describe("checkPreparedApiPermission", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    getState.mockReturnValue(state);
    getDatabase.mockResolvedValue({
      name: "instances/i/databases/db",
      project: "projects/example",
    });
    state.hasWorkspacePermission.mockReturnValue(false);
    state.hasProjectPermission.mockReturnValue(false);
  });

  it("denies a workspace API when the cached IAM state is complete", async () => {
    const result = await checkPreparedApiPermission(workspaceCall);
    expect(state.hasWorkspacePermission).toHaveBeenCalledWith(
      "bb.instances.list"
    );
    expect(result).toMatchObject({
      kind: "denied",
      result: expect.stringContaining(
        "Not called: You don't have the required permission bb.instances.list."
      ),
    });
    if (result.kind !== "denied") throw new Error("Expected denial");
    expect(JSON.parse(result.result)).toMatchObject({
      response: {
        requiredPermissions: ["bb.instances.list"],
        resources: ["workspace"],
      },
    });
  });

  it("checks a project-scoped API using the frontend permission checker", async () => {
    const result = await checkPreparedApiPermission(projectCall);
    expect(state.hasProjectPermission).toHaveBeenCalledWith(
      expect.objectContaining({ name: "projects/example" }),
      "bb.projects.update"
    );
    expect(result).toMatchObject({
      kind: "denied",
      result: expect.stringContaining(
        "Not called: You don't have the required permission bb.projects.update."
      ),
    });
    if (result.kind !== "denied") throw new Error("Expected denial");
    expect(JSON.parse(result.result).response.resources).toEqual([
      "projects/example",
    ]);
  });

  it("allows an API when the frontend checker grants its permission", async () => {
    state.hasProjectPermission.mockReturnValue(true);
    expect(await checkPreparedApiPermission(projectCall)).toEqual({
      kind: "allowed",
    });
  });

  it("does not infer a permission for a custom-auth API", async () => {
    const result = await checkPreparedApiPermission({
      operationId: "bytebase.v1.DatabaseService.ListDatabases",
      path: "/bytebase.v1.DatabaseService/ListDatabases",
      body: '{"parent":"projects/example"}',
    });
    expect(result).toEqual({ kind: "allowed" });
    expect(state.loadWorkspacePermissionState).not.toHaveBeenCalled();
  });

  it.each([
    {
      operationId: "bytebase.v1.DatabaseService.UpdateDatabase",
      path: "/bytebase.v1.DatabaseService/UpdateDatabase",
      body: '{"database":{"name":"instances/i/databases/db"}}',
    },
    {
      operationId: "bytebase.v1.DatabaseService.BatchUpdateDatabases",
      path: "/bytebase.v1.DatabaseService/BatchUpdateDatabases",
      body: '{"parent":"instances/i","requests":[{"database":{"name":"instances/i/databases/db"}}]}',
    },
    {
      operationId: "bytebase.v1.DatabaseService.BatchSyncDatabases",
      path: "/bytebase.v1.DatabaseService/BatchSyncDatabases",
      body: '{"parent":"instances/i","names":["instances/i/databases/db"]}',
    },
    {
      operationId: "bytebase.v1.IssueService.BatchUpdateIssuesStatus",
      path: "/bytebase.v1.IssueService/BatchUpdateIssuesStatus",
      body: '{"parent":"projects/-","issues":["projects/example/issues/1"],"status":"OPEN"}',
    },
  ])("defers $operationId permission checks to the backend", async (call) => {
    expect(await checkPreparedApiPermission(call)).toEqual({ kind: "allowed" });
    expect(getDatabase).not.toHaveBeenCalled();
    expect(getState).not.toHaveBeenCalled();
  });

  it("leaves authorization to the target API if IAM state is unavailable", async () => {
    getState.mockReturnValue({ ...state, workspacePolicy: undefined });
    expect(await checkPreparedApiPermission(workspaceCall)).toEqual({
      kind: "allowed",
    });
  });

  it("leaves authorization to the API when database ownership cannot be read", async () => {
    getDatabase.mockRejectedValueOnce(new Error("unavailable"));
    expect(
      await checkPreparedApiPermission({
        operationId: "bytebase.v1.DatabaseCatalogService.UpdateDatabaseCatalog",
        path: "/bytebase.v1.DatabaseCatalogService/UpdateDatabaseCatalog",
        body: '{"catalog":{"name":"instances/i/databases/db/catalog"}}',
      })
    ).toEqual({ kind: "allowed" });
    expect(state.hasProjectPermission).not.toHaveBeenCalled();
  });

  it("uses current database ownership rather than a stale cache", async () => {
    getState.mockReturnValue({
      ...state,
      databasesByName: {
        "instances/i/databases/db": { project: "projects/old" },
      },
    });
    const result = await checkPreparedApiPermission({
      operationId: "bytebase.v1.DatabaseCatalogService.UpdateDatabaseCatalog",
      path: "/bytebase.v1.DatabaseCatalogService/UpdateDatabaseCatalog",
      body: '{"catalog":{"name":"instances/i/databases/db/catalog"}}',
    });
    expect(state.hasProjectPermission).toHaveBeenCalledWith(
      expect.objectContaining({ name: "projects/example" }),
      "bb.databaseCatalogs.update"
    );
    expect(result.kind).toBe("denied");
  });

  it("checks database subresources against their current owning project", async () => {
    const result = await checkPreparedApiPermission({
      operationId: "bytebase.v1.DatabaseCatalogService.UpdateDatabaseCatalog",
      path: "/bytebase.v1.DatabaseCatalogService/UpdateDatabaseCatalog",
      body: '{"catalog":{"name":"instances/i/databases/db/catalog"}}',
    });
    expect(getDatabase).toHaveBeenCalledWith(
      expect.objectContaining({ name: "instances/i/databases/db" }),
      expect.anything()
    );
    expect(
      getDatabase.mock.calls[0][1].contextValues.get(silentContextKey)
    ).toBe(true);
    expect(state.hasProjectPermission).toHaveBeenCalledWith(
      expect.objectContaining({ name: "projects/example" }),
      "bb.databaseCatalogs.update"
    );
    expect(result.kind).toBe("denied");
  });

  it("does not deny when the database lookup returns a different resource", async () => {
    getDatabase.mockResolvedValueOnce({
      name: "instances/i/databases/other",
      project: "projects/example",
    });
    expect(
      await checkPreparedApiPermission({
        operationId: "bytebase.v1.DatabaseCatalogService.UpdateDatabaseCatalog",
        path: "/bytebase.v1.DatabaseCatalogService/UpdateDatabaseCatalog",
        body: '{"catalog":{"name":"instances/i/databases/db/catalog"}}',
      })
    ).toEqual({ kind: "allowed" });
    expect(state.hasProjectPermission).not.toHaveBeenCalled();
  });

  it("checks both update and create permission for allowMissing", async () => {
    const result = await checkPreparedApiPermission({
      operationId: "bytebase.v1.InstanceService.UpdateInstance",
      path: "/bytebase.v1.InstanceService/UpdateInstance",
      body: '{"instance":{"name":"instances/i"},"allowMissing":true}',
    });
    expect(state.hasWorkspacePermission).toHaveBeenCalledWith(
      "bb.instances.create"
    );
    expect(result).toMatchObject({
      kind: "denied",
      result: expect.stringContaining(
        "Not called: You don't have the required permissions bb.instances.update and bb.instances.create."
      ),
    });
  });

  it("does not block a project call when its IAM policy cannot be read", async () => {
    getState.mockReturnValue({
      ...state,
      projectPoliciesByName: {},
    });
    expect(await checkPreparedApiPermission(projectCall)).toEqual({
      kind: "allowed",
    });
    expect(state.loadProjectIamPolicy).toHaveBeenCalledWith(
      "projects/example",
      true
    );
  });

  it("leaves a failed IAM state refresh to the target API", async () => {
    getState.mockReturnValue({ ...state, workspacePolicy: undefined });
    state.loadWorkspacePermissionState.mockRejectedValueOnce(
      new Error("unavailable")
    );
    expect(await checkPreparedApiPermission(workspaceCall)).toEqual({
      kind: "allowed",
    });
  });

  it("does not guess a dynamic permission for inline instance database listing", async () => {
    expect(
      await checkPreparedApiPermission({
        operationId: "bytebase.v1.InstanceService.ListInstanceDatabase",
        path: "/bytebase.v1.InstanceService/ListInstanceDatabase",
        body: '{"name":"instances/i","instance":{"name":"instances/i"}}',
      })
    ).toEqual({ kind: "allowed" });
    expect(state.hasWorkspacePermission).not.toHaveBeenCalled();
  });
});

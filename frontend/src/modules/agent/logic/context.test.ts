// @vitest-environment node
import { beforeEach, describe, expect, test, vi } from "vitest";
import type { ReactRoute } from "@/app/router";
import {
  INSTANCE_ROUTE_DETAIL,
  PROJECT_V1_ROUTE_INSTANCE_DETAIL,
} from "@/app/router/handles";
import { Engine, State } from "@/types/proto-es/v1/common_pb";
import type { AgentSqlContextCapture } from "./sqlEditorContext";

const mocks = vi.hoisted(() => ({
  sqlCapture: vi.fn<() => AgentSqlContextCapture | undefined>(),
  getOrFetchDatabaseByName: vi.fn(),
  getOrFetchInstanceByName: vi.fn(),
}));
vi.mock("./sqlEditorHost", () => ({
  getCurrentAgentSqlEditorContext: mocks.sqlCapture,
}));

vi.mock("@/stores/app", () => ({
  useAppStore: {
    getState: () => ({
      currentUser: undefined,
      loadCurrentUser: vi.fn(async () => undefined),
      getProjectByName: vi.fn(() => undefined),
      getOrFetchDatabaseByName: mocks.getOrFetchDatabaseByName,
      getOrFetchInstanceByName: mocks.getOrFetchInstanceByName,
      fetchIssueByName: vi.fn(async () => undefined),
    }),
  },
}));

import { extractRouteContext } from "./context";

const databaseRoute = {
  params: {
    projectId: "project1",
    instanceId: "instance1",
    databaseName: "database1",
  },
  query: {},
} as unknown as ReactRoute;

const unknownDatabase = {
  name: "instances/-1/databases/-1",
  effectiveEnvironment: "environments/-1",
};

describe("extractRouteContext", () => {
  beforeEach(() => {
    mocks.sqlCapture.mockReset();
    mocks.getOrFetchDatabaseByName.mockReset();
    mocks.getOrFetchInstanceByName.mockReset();
  });

  test("does not fetch a stale SQL URL database even while the selected Project is pending", async () => {
    mocks.sqlCapture.mockReturnValue({ kind: "pending" });
    expect((await extractRouteContext(databaseRoute)).database).toBeUndefined();
    expect(mocks.getOrFetchDatabaseByName).not.toHaveBeenCalled();
    mocks.sqlCapture.mockReturnValue({
      kind: "ready",
      snapshot: {
        version: 1,
        project: "projects/new",
        connection: {
          database: "instances/main/databases/new",
          project: "projects/new",
          engine: Engine.MYSQL,
        },
      },
    });
    const context = await extractRouteContext(databaseRoute);
    expect(context.database).toMatchObject({
      name: "instances/main/databases/new",
      engine: "MYSQL",
    });
    expect(context.project).toMatchObject({ name: "projects/new" });
    expect(mocks.getOrFetchDatabaseByName).not.toHaveBeenCalled();
  });

  test("resolves a workspace instance database", async () => {
    mocks.getOrFetchDatabaseByName.mockResolvedValueOnce({
      name: "instances/instance1/databases/database1",
      instanceResource: { engine: Engine.POSTGRES },
      effectiveEnvironment: "environments/test",
    });

    const context = await extractRouteContext(databaseRoute);

    expect(mocks.getOrFetchDatabaseByName).toHaveBeenCalledOnce();
    expect(mocks.getOrFetchDatabaseByName).toHaveBeenCalledWith(
      "instances/instance1/databases/database1"
    );
    expect(context.database).toEqual({
      name: "instances/instance1/databases/database1",
      engine: "POSTGRES",
      environment: "environments/test",
    });
  });

  test("falls back to a project instance database", async () => {
    mocks.getOrFetchDatabaseByName
      .mockResolvedValueOnce(unknownDatabase)
      .mockResolvedValueOnce({
        name: "projects/project1/instances/instance1/databases/database1",
        instanceResource: { engine: Engine.MYSQL },
        effectiveEnvironment: "environments/prod",
      });

    const context = await extractRouteContext(databaseRoute);

    expect(mocks.getOrFetchDatabaseByName).toHaveBeenNthCalledWith(
      1,
      "instances/instance1/databases/database1"
    );
    expect(mocks.getOrFetchDatabaseByName).toHaveBeenNthCalledWith(
      2,
      "projects/project1/instances/instance1/databases/database1"
    );
    expect(context.database).toEqual({
      name: "projects/project1/instances/instance1/databases/database1",
      engine: "MYSQL",
      environment: "environments/prod",
    });
  });

  test("omits database context when neither resource exists", async () => {
    mocks.getOrFetchDatabaseByName.mockResolvedValue(unknownDatabase);

    const context = await extractRouteContext(databaseRoute);

    expect(mocks.getOrFetchDatabaseByName).toHaveBeenCalledTimes(2);
    expect(context.database).toBeUndefined();
  });

  test.each([
    {
      route: {
        name: INSTANCE_ROUTE_DETAIL,
        params: { instanceId: "primary" },
      },
      name: "instances/primary",
    },
    {
      route: {
        name: PROJECT_V1_ROUTE_INSTANCE_DETAIL,
        params: { projectId: "project1", instanceId: "primary" },
      },
      name: "projects/project1/instances/primary",
    },
  ])("adds a safe instance summary for $name", async ({ route, name }) => {
    mocks.getOrFetchInstanceByName.mockResolvedValue({
      name,
      title: "Primary database",
      engine: Engine.POSTGRES,
      engineVersion: "16",
      state: State.ACTIVE,
      environment: "environments/prod",
      dataSources: [{ host: "private.example.com", password: "secret" }],
    });

    const context = await extractRouteContext({
      ...route,
      query: {},
    } as unknown as ReactRoute);

    expect(mocks.getOrFetchInstanceByName).toHaveBeenCalledWith(name, true);
    expect(context.instance).toEqual({
      name,
      title: "Primary database",
      engine: "POSTGRES",
      engineVersion: "16",
      state: "ACTIVE",
      environment: "environments/prod",
    });
  });
});

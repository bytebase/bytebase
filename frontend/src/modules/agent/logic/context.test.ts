import { beforeEach, describe, expect, test, vi } from "vitest";
import type { ReactRoute } from "@/app/router";
import { Engine } from "@/types/proto-es/v1/common_pb";

const mocks = vi.hoisted(() => ({
  getOrFetchDatabaseByName: vi.fn(),
}));

vi.mock("@/stores/app", () => ({
  useAppStore: {
    getState: () => ({
      currentUser: undefined,
      loadCurrentUser: vi.fn(async () => undefined),
      getProjectByName: vi.fn(() => undefined),
      getOrFetchDatabaseByName: mocks.getOrFetchDatabaseByName,
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
    mocks.getOrFetchDatabaseByName.mockReset();
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
});

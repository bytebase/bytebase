import { create } from "@bufbuild/protobuf";
import { beforeEach, describe, expect, test, vi } from "vitest";
import {
  type BatchSyncDatabasesRequest,
  DatabaseSchema$,
} from "@/types/proto-es/v1/database_service_pb";
import { InstanceSchema } from "@/types/proto-es/v1/instance_service_pb";
import { createDatabaseSlice } from "./database";

const mocks = vi.hoisted(() => ({
  batchSyncDatabases: vi.fn(),
  batchFetchProjects: vi.fn(),
  removeDatabaseMetadataCache: vi.fn(),
}));

vi.mock("@/api", () => ({
  databaseServiceClientConnect: {
    batchSyncDatabases: mocks.batchSyncDatabases,
  },
}));

const createStore = () => {
  const state: Record<string, unknown> = {
    batchFetchProjects: mocks.batchFetchProjects,
    removeDatabaseMetadataCache: mocks.removeDatabaseMetadataCache,
  };
  const set = (updater: unknown) => {
    const patch =
      typeof updater === "function"
        ? (updater as (value: typeof state) => object)(state)
        : updater;
    Object.assign(state, patch);
  };
  const get = () => state;
  Object.assign(
    state,
    createDatabaseSlice(set as never, get as never, {} as never)
  );
  return state as ReturnType<typeof createDatabaseSlice>;
};

describe("database store project instance scope", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    mocks.batchSyncDatabases.mockResolvedValue({});
  });

  test("uses the cross-parent wildcard for batch schema sync", async () => {
    const store = createStore();

    await store.batchSyncDatabases([
      "projects/app/instances/prod/databases/main",
    ]);

    const request = mocks.batchSyncDatabases.mock
      .calls[0][0] as BatchSyncDatabasesRequest;
    expect(request.parent).toBe("-");
  });

  test("updates only databases under the canonical project instance", () => {
    const store = createStore();
    store.databasesByName = {
      "projects/app/instances/prod/databases/main": create(DatabaseSchema$, {
        name: "projects/app/instances/prod/databases/main",
      }),
      "instances/prod/databases/main": create(DatabaseSchema$, {
        name: "instances/prod/databases/main",
      }),
    };

    store.updateDatabaseInstance(
      create(InstanceSchema, {
        name: "projects/app/instances/prod",
        title: "Project production",
      })
    );

    expect(
      store.databasesByName["projects/app/instances/prod/databases/main"]
        .instanceResource?.title
    ).toBe("Project production");
    expect(
      store.databasesByName["instances/prod/databases/main"].instanceResource
    ).toBeUndefined();
  });
});

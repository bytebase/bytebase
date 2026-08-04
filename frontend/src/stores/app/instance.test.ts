import { create } from "@bufbuild/protobuf";
import { beforeEach, describe, expect, test, vi } from "vitest";
import {
  InstanceSchema,
  type ListInstancesRequest,
} from "@/types/proto-es/v1/instance_service_pb";
import { createInstanceSlice } from "./instance";

const mocks = vi.hoisted(() => ({
  listInstances: vi.fn(),
  createInstance: vi.fn(),
  batchSyncInstances: vi.fn(),
  batchUpdateInstances: vi.fn(),
}));

vi.mock("@/api", () => ({
  instanceServiceClientConnect: {
    listInstances: mocks.listInstances,
    createInstance: mocks.createInstance,
    batchSyncInstances: mocks.batchSyncInstances,
    batchUpdateInstances: mocks.batchUpdateInstances,
  },
}));

const createStore = () => {
  const state: Record<string, unknown> = {};
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
    createInstanceSlice(set as never, get as never, {} as never)
  );
  return state as ReturnType<typeof createInstanceSlice>;
};

describe("instance store project parent", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    mocks.listInstances.mockResolvedValue({
      instances: [],
      nextPageToken: "",
    });
    mocks.createInstance.mockResolvedValue(
      create(InstanceSchema, {
        name: "projects/app/instances/prod",
        title: "Prod",
      })
    );
    mocks.batchSyncInstances.mockResolvedValue({});
    mocks.batchUpdateInstances.mockResolvedValue({ instances: [] });
  });

  test("forwards the project parent when listing instances", async () => {
    const store = createStore();

    await store.fetchInstanceList({
      parent: "projects/app",
      pageSize: 10,
    } as Parameters<typeof store.fetchInstanceList>[0]);

    const request = mocks.listInstances.mock
      .calls[0][0] as ListInstancesRequest;
    expect(request.parent).toBe("projects/app");
    expect(request.pageSize).toBe(10);
  });

  test("leaves the parent unset for workspace lists", async () => {
    const store = createStore();

    await store.fetchInstanceList({ pageSize: 10 });

    const request = mocks.listInstances.mock
      .calls[0][0] as ListInstancesRequest;
    expect(request.parent).toBeUndefined();
  });

  test("forwards the project parent when creating an instance", async () => {
    const store = createStore();
    const instance = create(InstanceSchema, {
      name: "projects/app/instances/prod",
      title: "Prod",
    });

    await store.createInstance(instance, false, {
      parent: "projects/app",
    } as Parameters<typeof store.createInstance>[2]);

    expect(mocks.createInstance.mock.calls[0][0]).toMatchObject({
      parent: "projects/app",
    });
  });

  test("forwards the project parent for batch operations", async () => {
    const store = createStore();

    await store.batchSyncInstances(
      ["projects/app/instances/prod"],
      true,
      "projects/app"
    );
    await store.batchUpdateInstances([], "projects/app");

    expect(mocks.batchSyncInstances.mock.calls[0][0]).toMatchObject({
      parent: "projects/app",
    });
    expect(mocks.batchUpdateInstances.mock.calls[0][0]).toMatchObject({
      parent: "projects/app",
    });
  });
});

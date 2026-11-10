// @vitest-environment node
import { create } from "@bufbuild/protobuf";
import { beforeEach, describe, expect, test, vi } from "vitest";
import { State } from "@/types/proto-es/v1/common_pb";
import { ProjectSchema } from "@/types/proto-es/v1/project_service_pb";
import { createProjectSlice, getListProjectFilter } from "./project";

const QUOTED = 'SELECT * FROM "users"';
const ESCAPED = 'SELECT * FROM \\"users\\"';

describe("getListProjectFilter", () => {
  test("escapes a quote in the free-text query", () => {
    const filter = getListProjectFilter({ query: QUOTED }).toLowerCase();
    expect(filter).not.toContain(QUOTED.toLowerCase());
    expect(filter).toContain(ESCAPED.toLowerCase());
  });
});

const mocks = vi.hoisted(() => ({
  listProjects: vi.fn(),
  searchProjects: vi.fn(),
  hasWorkspacePermissionV2: vi.fn(() => false),
}));

vi.mock("@/api", () => ({
  projectServiceClientConnect: {
    listProjects: mocks.listProjects,
    searchProjects: mocks.searchProjects,
  },
}));

vi.mock("@/utils", async (importOriginal) => ({
  ...(await importOriginal<typeof import("@/utils")>()),
  hasWorkspacePermissionV2: mocks.hasWorkspacePermissionV2,
}));

const createStore = () => {
  const state: Record<string, unknown> = {};
  const set = (updater: unknown) => {
    Object.assign(
      state,
      typeof updater === "function" ? updater(state) : updater
    );
  };
  Object.assign(
    state,
    createProjectSlice(set as never, (() => state) as never, {} as never)
  );
  return state as ReturnType<typeof createProjectSlice>;
};

const project = (id: string) =>
  create(ProjectSchema, { name: `projects/${id}` });

describe("fetchProjectList", () => {
  beforeEach(() => {
    vi.resetAllMocks();
    mocks.hasWorkspacePermissionV2.mockReturnValue(false);
  });

  test("fills short and empty pages and resumes without losing projects", async () => {
    const a = project("a");
    const b = project("b");
    const c = project("c");
    const d = project("d");
    mocks.searchProjects
      .mockResolvedValueOnce({ projects: [a], nextPageToken: "scan-3" })
      .mockResolvedValueOnce({ projects: [], nextPageToken: "scan-6" })
      .mockResolvedValueOnce({ projects: [b, c, d], nextPageToken: "scan-9" })
      .mockResolvedValueOnce({ projects: [project("e")], nextPageToken: "" });
    const store = createStore();
    const params = {
      pageSize: 3,
      filter: { state: State.ACTIVE, query: "app" },
      orderBy: "title asc",
      cache: true,
    };
    const first = await store.fetchProjectList(params);
    expect(first).toEqual({ projects: [a, b, c, d], nextPageToken: "scan-9" });
    expect(Object.values(store.projectsByName)).toEqual([a, b, c, d]);
    expect(
      mocks.searchProjects.mock.calls.map(([request]) => ({
        pageSize: request.pageSize,
        pageToken: request.pageToken,
      }))
    ).toEqual([
      { pageSize: 3, pageToken: "" },
      { pageSize: 3, pageToken: "scan-3" },
      { pageSize: 3, pageToken: "scan-6" },
    ]);
    for (const [request] of mocks.searchProjects.mock.calls) {
      expect(request).toMatchObject({
        filter: getListProjectFilter(params.filter),
        orderBy: "title asc",
        showDeleted: false,
      });
    }
    const second = await store.fetchProjectList({
      ...params,
      pageToken: first.nextPageToken,
    });
    expect(second).toEqual({ projects: [project("e")], nextPageToken: "" });
    expect(mocks.searchProjects.mock.calls[3][0]).toMatchObject({
      pageSize: 3,
      pageToken: "scan-9",
    });
  });

  test.each([undefined, 0, 5000])(
    "uses the API page-size bounds for %s",
    async (pageSize) => {
      mocks.searchProjects.mockResolvedValue({
        projects: [],
        nextPageToken: "",
      });
      await createStore().fetchProjectList({ pageSize });
      expect(mocks.searchProjects.mock.calls[0][0].pageSize).toBe(
        pageSize === 5000 ? 1000 : 10
      );
    }
  );

  test("returns all accumulated rows when the last page is short", async () => {
    mocks.searchProjects
      .mockResolvedValueOnce({
        projects: [project("a")],
        nextPageToken: "next",
      })
      .mockResolvedValueOnce({ projects: [project("b")], nextPageToken: "" });
    expect(await createStore().fetchProjectList({ pageSize: 3 })).toEqual({
      projects: [project("a"), project("b")],
      nextPageToken: "",
    });
  });

  test("stops at exhaustion when no projects are visible", async () => {
    mocks.searchProjects
      .mockResolvedValueOnce({ projects: [], nextPageToken: "next" })
      .mockResolvedValueOnce({ projects: [], nextPageToken: "" });
    expect(await createStore().fetchProjectList({ pageSize: 2 })).toEqual({
      projects: [],
      nextPageToken: "",
    });
    expect(mocks.searchProjects).toHaveBeenCalledTimes(2);
  });

  test("keeps a full workspace list page to one request", async () => {
    mocks.hasWorkspacePermissionV2.mockReturnValue(true);
    mocks.listProjects.mockResolvedValue({
      projects: [project("a"), project("b")],
      nextPageToken: "next",
    });
    const store = createStore();
    expect(await store.fetchProjectList({ pageSize: 2 })).toEqual({
      projects: [project("a"), project("b")],
      nextPageToken: "next",
    });
    expect(mocks.listProjects).toHaveBeenCalledTimes(1);
    expect(mocks.searchProjects).not.toHaveBeenCalled();
    expect(store.projectsByName).toEqual({});
  });

  test("does not start an already canceled fetch", async () => {
    const controller = new AbortController();
    controller.abort();
    await expect(
      createStore().fetchProjectList({ pageSize: 2, signal: controller.signal })
    ).rejects.toMatchObject({ name: "AbortError" });
    expect(mocks.searchProjects).not.toHaveBeenCalled();
  });

  test.each([false, true])(
    "passes cancellation to the RPC and stops an in-flight fetch (canList=%s)",
    async (canList) => {
      mocks.hasWorkspacePermissionV2.mockReturnValue(canList);
      const rpc = canList ? mocks.listProjects : mocks.searchProjects;
      const controller = new AbortController();
      rpc.mockImplementationOnce((_request, options) => {
        expect(options.signal).toBe(controller.signal);
        return new Promise((_resolve, reject) => {
          options.signal.addEventListener(
            "abort",
            () => reject(options.signal.reason),
            { once: true }
          );
        });
      });
      const store = createStore();
      const pending = store.fetchProjectList({
        pageSize: 2,
        signal: controller.signal,
        cache: true,
      });
      controller.abort();
      await expect(pending).rejects.toMatchObject({ name: "AbortError" });
      expect(rpc).toHaveBeenCalledTimes(1);
      expect(store.projectsByName).toEqual({});
    }
  );

  test.each(["next", ""])(
    "discards a response that arrives after cancellation (token=%s)",
    async (nextPageToken) => {
      const controller = new AbortController();
      mocks.searchProjects.mockImplementationOnce(async () => {
        controller.abort();
        return { projects: [project("a")], nextPageToken };
      });
      const store = createStore();
      await expect(
        store.fetchProjectList({
          pageSize: 2,
          signal: controller.signal,
          cache: true,
        })
      ).rejects.toMatchObject({ name: "AbortError" });
      expect(mocks.searchProjects).toHaveBeenCalledTimes(1);
      expect(store.projectsByName).toEqual({});
    }
  );

  test("propagates a refill failure without caching a partial page", async () => {
    mocks.searchProjects
      .mockResolvedValueOnce({
        projects: [project("a")],
        nextPageToken: "next",
      })
      .mockRejectedValueOnce(new Error("fetch failed"));
    const store = createStore();
    await expect(
      store.fetchProjectList({ pageSize: 2, cache: true })
    ).rejects.toThrow("fetch failed");
    expect(store.projectsByName).toEqual({});
  });
});

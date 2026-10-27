// @vitest-environment node
import { create } from "@bufbuild/protobuf";
import { beforeEach, describe, expect, test, vi } from "vitest";
import { databaseServiceClientConnect } from "@/api";
import { silentContextKey } from "@/api/context-key";
import { State } from "@/types/proto-es/v1/common_pb";
import {
  DatabaseSchema$,
  ListDatabasesRequestSchema,
} from "@/types/proto-es/v1/database_service_pb";
import { ProjectSchema } from "@/types/proto-es/v1/project_service_pb";

const mocks = vi.hoisted(() => ({
  fetchProjectList: vi.fn(),
  listDatabases: vi.fn(),
}));

vi.mock("@/api", () => ({
  databaseServiceClientConnect: { listDatabases: mocks.listDatabases },
}));

vi.mock("@/stores/app", () => ({
  useAppStore: {
    getState: () => ({ fetchProjectList: mocks.fetchProjectList }),
  },
}));

import { searchAgentDatabases, searchAgentProjects } from "./resourceSearch";

const project = create(ProjectSchema, {
  name: "projects/project-sample",
  title: "Sample project",
});

describe("searchAgentProjects", () => {
  beforeEach(() => {
    mocks.fetchProjectList.mockReset();
  });

  test("searches accessible active projects by title or ID in pages of 20", async () => {
    const controller = new AbortController();
    mocks.fetchProjectList.mockResolvedValueOnce({
      projects: [project],
      nextPageToken: "next-page",
    });

    await expect(
      searchAgentProjects(" sample ", "", controller.signal)
    ).resolves.toEqual({ items: [project], nextPageToken: "next-page" });
    expect(mocks.fetchProjectList).toHaveBeenCalledExactlyOnceWith({
      filter: {
        query: "sample",
        state: State.ACTIVE,
        excludeDefault: true,
      },
      pageSize: 20,
      pageToken: "",
      signal: controller.signal,
      silent: true,
      orderBy: "title",
      cache: true,
    });
  });

  test("forwards the continuation token and normalizes an exhausted result", async () => {
    const controller = new AbortController();
    mocks.fetchProjectList.mockResolvedValueOnce({
      projects: [],
    });

    await expect(
      searchAgentProjects("", "continuation", controller.signal)
    ).resolves.toEqual({ items: [], nextPageToken: "" });
    expect(mocks.fetchProjectList).toHaveBeenCalledWith(
      expect.objectContaining({
        pageToken: "continuation",
        filter: expect.objectContaining({ query: "" }),
      })
    );
  });

  test("includes the default project when choosing a database owner", async () => {
    const controller = new AbortController();
    mocks.fetchProjectList.mockResolvedValueOnce({ projects: [] });

    await searchAgentProjects("", "", controller.signal, true);

    expect(mocks.fetchProjectList).toHaveBeenCalledWith(
      expect.objectContaining({
        filter: expect.objectContaining({ excludeDefault: false }),
      })
    );
  });

  test("propagates fetch failures so the picker can offer Retry", async () => {
    const controller = new AbortController();
    const failure = new Error("projects unavailable");
    mocks.fetchProjectList.mockRejectedValueOnce(failure);

    await expect(
      searchAgentProjects("sample", "", controller.signal)
    ).rejects.toBe(failure);
  });
});

describe("searchAgentDatabases", () => {
  const database = create(DatabaseSchema$, {
    name: "instances/production/databases/orders",
    project: "projects/orders",
    state: State.ACTIVE,
  });

  beforeEach(() => {
    mocks.listDatabases.mockReset();
  });

  test("searches a project page by database name with an escaped CEL query", async () => {
    const controller = new AbortController();
    mocks.listDatabases.mockResolvedValueOnce({
      databases: [database],
      nextPageToken: "next-page",
    });

    await expect(
      searchAgentDatabases(
        "projects/orders",
        ' orders"main ',
        "previous-page",
        controller.signal
      )
    ).resolves.toEqual({ items: [database], nextPageToken: "next-page" });
    expect(
      databaseServiceClientConnect.listDatabases
    ).toHaveBeenCalledExactlyOnceWith(
      create(ListDatabasesRequestSchema, {
        parent: "projects/orders",
        pageSize: 20,
        pageToken: "previous-page",
        filter: 'name.contains("orders\\"main")',
        orderBy: "name",
        showDeleted: false,
      }),
      expect.objectContaining({
        signal: controller.signal,
        contextValues: expect.anything(),
      })
    );
    expect(
      mocks.listDatabases.mock.calls[0][1]?.contextValues.get(silentContextKey)
    ).toBe(true);
  });

  test("browses the first page without a query and normalizes an exhausted result", async () => {
    const controller = new AbortController();
    mocks.listDatabases.mockResolvedValueOnce({ databases: [] });

    await expect(
      searchAgentDatabases("projects/orders", "  ", "", controller.signal)
    ).resolves.toEqual({ items: [], nextPageToken: "" });
    expect(mocks.listDatabases).toHaveBeenCalledWith(
      create(ListDatabasesRequestSchema, {
        parent: "projects/orders",
        pageSize: 20,
        pageToken: "",
        filter: "",
        orderBy: "name",
        showDeleted: false,
      }),
      expect.anything()
    );
  });

  test("searches a workspace page without requiring a project", async () => {
    const controller = new AbortController();
    mocks.listDatabases.mockResolvedValueOnce({
      databases: [database],
      nextPageToken: "next-page",
    });

    await expect(
      searchAgentDatabases(
        "workspaces/default",
        "orders",
        "previous-page",
        controller.signal
      )
    ).resolves.toEqual({ items: [database], nextPageToken: "next-page" });
    expect(mocks.listDatabases).toHaveBeenCalledWith(
      create(ListDatabasesRequestSchema, {
        parent: "workspaces/default",
        pageSize: 20,
        pageToken: "previous-page",
        filter: 'name.contains("orders")',
        orderBy: "name",
        showDeleted: false,
      }),
      expect.anything()
    );
  });

  test("rejects non-project and non-workspace parents without listing databases", async () => {
    await expect(
      searchAgentDatabases(
        "instances/main",
        "",
        "",
        new AbortController().signal
      )
    ).rejects.toThrow("project or workspace");
    expect(mocks.listDatabases).not.toHaveBeenCalled();
  });

  test("does not start a search after the request was canceled", async () => {
    const controller = new AbortController();
    controller.abort();

    await expect(
      searchAgentDatabases("projects/orders", "", "", controller.signal)
    ).rejects.toMatchObject({ name: "AbortError" });
    expect(mocks.listDatabases).not.toHaveBeenCalled();
  });

  test("propagates API failures for the picker Retry state", async () => {
    const failure = new Error("databases unavailable");
    mocks.listDatabases.mockRejectedValueOnce(failure);

    await expect(
      searchAgentDatabases(
        "projects/orders",
        "orders",
        "",
        new AbortController().signal
      )
    ).rejects.toBe(failure);
  });
});

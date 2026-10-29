// @vitest-environment node
import { create } from "@bufbuild/protobuf";
import { beforeEach, describe, expect, test, vi } from "vitest";
import { databaseServiceClientConnect, issueServiceClientConnect } from "@/api";
import { silentContextKey } from "@/api/context-key";
import { State } from "@/types/proto-es/v1/common_pb";
import {
  DatabaseSchema$,
  SearchDatabasesRequestSchema,
} from "@/types/proto-es/v1/database_service_pb";
import {
  IssueSchema,
  SearchIssuesRequestSchema,
} from "@/types/proto-es/v1/issue_service_pb";
import { ProjectSchema } from "@/types/proto-es/v1/project_service_pb";

const mocks = vi.hoisted(() => ({
  fetchProjectList: vi.fn(),
  listDatabases: vi.fn(),
  searchDatabases: vi.fn(),
  searchIssues: vi.fn(),
  batchGetOrFetchUsers: vi.fn(),
}));

vi.mock("@/api", () => ({
  databaseServiceClientConnect: {
    listDatabases: mocks.listDatabases,
    searchDatabases: mocks.searchDatabases,
  },
  issueServiceClientConnect: { searchIssues: mocks.searchIssues },
}));

vi.mock("@/stores/app", () => ({
  useAppStore: {
    getState: () => ({
      fetchProjectList: mocks.fetchProjectList,
      batchGetOrFetchUsers: mocks.batchGetOrFetchUsers,
    }),
  },
}));

import {
  searchAgentDatabases,
  searchAgentIssues,
  searchAgentProjects,
} from "./resourceSearch";

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
    mocks.searchDatabases.mockReset();
  });

  test("searches a project page by database name with an escaped CEL query", async () => {
    const controller = new AbortController();
    mocks.searchDatabases.mockResolvedValueOnce({
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
      databaseServiceClientConnect.searchDatabases
    ).toHaveBeenCalledExactlyOnceWith(
      create(SearchDatabasesRequestSchema, {
        pageSize: 20,
        pageToken: "previous-page",
        filter:
          'project == "projects/orders" && name.contains("orders\\"main")',
      }),
      expect.objectContaining({
        signal: controller.signal,
        contextValues: expect.anything(),
      })
    );
    expect(
      mocks.searchDatabases.mock.calls[0][1]?.contextValues.get(
        silentContextKey
      )
    ).toBe(true);
    expect(mocks.listDatabases).not.toHaveBeenCalled();
  });

  test("browses the first page without a query and normalizes an exhausted result", async () => {
    const controller = new AbortController();
    mocks.searchDatabases.mockResolvedValueOnce({ databases: [] });

    await expect(
      searchAgentDatabases("projects/orders", "  ", "", controller.signal)
    ).resolves.toEqual({ items: [], nextPageToken: "" });
    expect(mocks.searchDatabases).toHaveBeenCalledWith(
      create(SearchDatabasesRequestSchema, {
        pageSize: 20,
        pageToken: "",
        filter: 'project == "projects/orders"',
      }),
      expect.anything()
    );
  });

  test("uses cross-project search for a workspace database page", async () => {
    const controller = new AbortController();
    mocks.searchDatabases.mockResolvedValueOnce({
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
    expect(mocks.searchDatabases).toHaveBeenCalledWith(
      create(SearchDatabasesRequestSchema, {
        pageSize: 20,
        pageToken: "previous-page",
        filter: 'name.contains("orders")',
      }),
      expect.objectContaining({ signal: controller.signal })
    );
    expect(mocks.listDatabases).not.toHaveBeenCalled();
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
    mocks.searchDatabases.mockRejectedValueOnce(failure);

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

describe("searchAgentIssues", () => {
  const issue = create(IssueSchema, {
    name: "projects/orders/issues/123",
    title: "Add an orders index",
  });

  beforeEach(() => {
    mocks.searchIssues.mockReset();
    mocks.batchGetOrFetchUsers.mockReset();
  });

  test("hydrates issue creators for the shared issue summary", async () => {
    const controller = new AbortController();
    const issueWithCreator = create(IssueSchema, {
      ...issue,
      creator: "users/author@example.com",
    });
    mocks.searchIssues.mockResolvedValueOnce({ issues: [issueWithCreator] });
    mocks.batchGetOrFetchUsers.mockResolvedValueOnce([]);

    await expect(
      searchAgentIssues("projects/-", "", "", controller.signal)
    ).resolves.toEqual({ items: [issueWithCreator], nextPageToken: "" });
    expect(mocks.batchGetOrFetchUsers).toHaveBeenCalledExactlyOnceWith([
      "users/author@example.com",
    ]);
  });

  test("keeps issue search results when creator lookup fails", async () => {
    const issueWithCreator = create(IssueSchema, {
      ...issue,
      creator: "users/author@example.com",
    });
    mocks.searchIssues.mockResolvedValueOnce({ issues: [issueWithCreator] });
    mocks.batchGetOrFetchUsers.mockRejectedValueOnce(
      new Error("users unavailable")
    );

    await expect(
      searchAgentIssues("projects/orders", "", "", new AbortController().signal)
    ).resolves.toEqual({ items: [issueWithCreator], nextPageToken: "" });
  });

  test("does not return a stale result if canceled during creator lookup", async () => {
    const controller = new AbortController();
    const issueWithCreator = create(IssueSchema, {
      ...issue,
      creator: "users/author@example.com",
    });
    let resolveLookup: (() => void) | undefined;
    mocks.searchIssues.mockResolvedValueOnce({ issues: [issueWithCreator] });
    mocks.batchGetOrFetchUsers.mockImplementationOnce(
      () =>
        new Promise<void>((resolve) => {
          resolveLookup = resolve;
        })
    );

    const search = searchAgentIssues(
      "projects/orders",
      "",
      "",
      controller.signal
    );
    await vi.waitFor(() => {
      expect(mocks.batchGetOrFetchUsers).toHaveBeenCalledOnce();
    });
    controller.abort();
    resolveLookup?.();

    await expect(search).rejects.toMatchObject({ name: "AbortError" });
  });

  test("searches only the selected project with remote query and pagination", async () => {
    const controller = new AbortController();
    mocks.searchIssues.mockResolvedValueOnce({
      issues: [issue],
      nextPageToken: "next-page",
    });

    await expect(
      searchAgentIssues(
        "projects/orders",
        " index ",
        "previous-page",
        controller.signal
      )
    ).resolves.toEqual({ items: [issue], nextPageToken: "next-page" });
    expect(
      issueServiceClientConnect.searchIssues
    ).toHaveBeenCalledExactlyOnceWith(
      create(SearchIssuesRequestSchema, {
        parent: "projects/orders",
        pageSize: 20,
        pageToken: "previous-page",
        query: "index",
        orderBy: "update_time desc",
      }),
      expect.objectContaining({
        signal: controller.signal,
        contextValues: expect.anything(),
      })
    );
    expect(
      mocks.searchIssues.mock.calls[0][1]?.contextValues.get(silentContextKey)
    ).toBe(true);
  });

  test("normalizes an exhausted page and propagates failures", async () => {
    const controller = new AbortController();
    mocks.searchIssues.mockResolvedValueOnce({ issues: [] });
    await expect(
      searchAgentIssues("projects/orders", "", "", controller.signal)
    ).resolves.toEqual({
      items: [],
      nextPageToken: "",
    });
    const failure = new Error("issues unavailable");
    mocks.searchIssues.mockRejectedValueOnce(failure);
    await expect(
      searchAgentIssues("projects/orders", "", "", controller.signal)
    ).rejects.toBe(failure);
  });

  test("searches issues across accessible projects from workspace context", async () => {
    const controller = new AbortController();
    mocks.searchIssues.mockResolvedValueOnce({
      issues: [issue],
      nextPageToken: "next-page",
    });

    await expect(
      searchAgentIssues(
        "projects/-",
        " index ",
        "previous-page",
        controller.signal
      )
    ).resolves.toEqual({ items: [issue], nextPageToken: "next-page" });
    expect(mocks.searchIssues).toHaveBeenCalledExactlyOnceWith(
      create(SearchIssuesRequestSchema, {
        parent: "projects/-",
        pageSize: 20,
        pageToken: "previous-page",
        query: "index",
        orderBy: "update_time desc",
      }),
      expect.objectContaining({ signal: controller.signal })
    );
  });

  test("rejects non-project parents before requesting issues", async () => {
    for (const parent of ["workspaces/default", "projects/orders/issues/123"]) {
      await expect(
        searchAgentIssues(parent, "", "", new AbortController().signal)
      ).rejects.toThrow("canonical project");
    }
    expect(mocks.searchIssues).not.toHaveBeenCalled();
  });

  test("does not start an aborted search", async () => {
    const controller = new AbortController();
    controller.abort();
    await expect(
      searchAgentIssues("projects/orders", "", "", controller.signal)
    ).rejects.toMatchObject({ name: "AbortError" });
    expect(mocks.searchIssues).not.toHaveBeenCalled();
  });
});

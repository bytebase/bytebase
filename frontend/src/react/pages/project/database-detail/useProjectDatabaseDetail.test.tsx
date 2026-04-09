import { act, createElement } from "react";
import { createRoot } from "react-dom/client";
import { afterEach, beforeEach, describe, expect, test, vi } from "vitest";
import type { UseProjectDatabaseDetailOptions } from "./useProjectDatabaseDetail";
import { useProjectDatabaseDetail } from "./useProjectDatabaseDetail";

(
  globalThis as { IS_REACT_ACT_ENVIRONMENT?: boolean }
).IS_REACT_ACT_ENVIRONMENT = true;

const originalConsoleError = console.error;

const mocks = vi.hoisted(() => {
  const localStorage = {
    clear: vi.fn(),
    getItem: vi.fn(() => null),
    removeItem: vi.fn(),
    setItem: vi.fn(),
  };
  vi.stubGlobal("localStorage", localStorage);

  return {
    localStorage,
    replace: vi.fn(),
    getOrFetchDatabaseByName: vi.fn(),
    getOrFetchDatabaseMetadata: vi.fn(),
    getDatabaseByName: vi.fn(),
    getInstanceResource: vi.fn((database: { instanceResource?: unknown }) => {
      return database.instanceResource ?? {};
    }),
    instanceV1HasAlterSchema: vi.fn(() => true),
    extractProjectResourceName: vi.fn((name: string) => {
      return name.split("/").at(-1) ?? "";
    }),
    isDefaultProject: vi.fn(() => false),
    routeNames: {
      databaseDetail: "workspace.project.database.detail",
      databaseChangelogDetail: "workspace.project.database.changelog.detail",
      databaseRevisionDetail: "workspace.project.database.revision.detail",
    },
  };
});

vi.mock("@/router", () => ({
  router: { replace: mocks.replace },
}));

vi.mock("@/router/dashboard/projectV1", () => ({
  PROJECT_V1_ROUTE_DATABASE_DETAIL: mocks.routeNames.databaseDetail,
  PROJECT_V1_ROUTE_DATABASE_CHANGELOG_DETAIL:
    mocks.routeNames.databaseChangelogDetail,
  PROJECT_V1_ROUTE_DATABASE_REVISION_DETAIL:
    mocks.routeNames.databaseRevisionDetail,
}));

vi.mock("@/store", () => ({
  useDatabaseV1Store: () => ({
    getOrFetchDatabaseByName: mocks.getOrFetchDatabaseByName,
    getDatabaseByName: mocks.getDatabaseByName,
  }),
  useDBSchemaV1Store: () => ({
    getOrFetchDatabaseMetadata: mocks.getOrFetchDatabaseMetadata,
  }),
}));

vi.mock("@/utils", () => ({
  getInstanceResource: mocks.getInstanceResource,
  instanceV1HasAlterSchema: mocks.instanceV1HasAlterSchema,
}));

vi.mock("@/utils/v1/project", () => ({
  extractProjectResourceName: mocks.extractProjectResourceName,
}));

vi.mock("@/types/v1/project", () => {
  const UNKNOWN_PROJECT_NAME = "projects/-";
  return {
    UNKNOWN_PROJECT_NAME,
    isDefaultProject: mocks.isDefaultProject,
    unknownProject: () => ({
      name: UNKNOWN_PROJECT_NAME,
      title: "",
    }),
    defaultProject: (name: string) => ({
      name,
      title: "Default project",
    }),
    isValidProjectName: (name: string | undefined) =>
      !!name && name.startsWith("projects/") && name !== UNKNOWN_PROJECT_NAME,
  };
});

function HookProbe(
  props: UseProjectDatabaseDetailOptions & {
    onValue: (value: ReturnType<typeof useProjectDatabaseDetail>) => void;
  }
) {
  props.onValue(useProjectDatabaseDetail(props));
  return null;
}

const waitFor = async (condition: () => boolean, timeoutMs = 1000) => {
  const start = Date.now();
  while (!condition()) {
    if (Date.now() - start >= timeoutMs) {
      throw new Error("Timed out waiting for condition");
    }
    await Promise.resolve();
    await new Promise<void>((resolve) => {
      setTimeout(resolve, 0);
    });
  }
};

const renderIntoContainer = (element: ReturnType<typeof createElement>) => {
  const container = document.createElement("div");
  const root = createRoot(container);

  return {
    render: (nextElement = element) => {
      root.render(nextElement);
    },
    unmount: () =>
      act(() => {
        root.unmount();
      }),
  };
};

beforeEach(() => {
  vi.spyOn(console, "error").mockImplementation((message, ...args) => {
    const text = String(message);
    if (text.includes("not wrapped in act")) {
      return;
    }
    originalConsoleError(message, ...args);
  });
  mocks.localStorage.getItem.mockReset();
  mocks.localStorage.getItem.mockReturnValue(null);
  mocks.localStorage.setItem.mockReset();
  mocks.localStorage.removeItem.mockReset();
  mocks.localStorage.clear.mockReset();
  mocks.replace.mockReset();
  mocks.getOrFetchDatabaseByName.mockReset();
  mocks.getOrFetchDatabaseMetadata.mockReset();
  mocks.getDatabaseByName.mockReset();
  mocks.getInstanceResource.mockReset();
  mocks.getInstanceResource.mockImplementation(
    (database: { instanceResource?: unknown }) =>
      database.instanceResource ?? {}
  );
  mocks.instanceV1HasAlterSchema.mockReset();
  mocks.instanceV1HasAlterSchema.mockReturnValue(true);
  mocks.extractProjectResourceName.mockReset();
  mocks.extractProjectResourceName.mockImplementation(
    (name: string) => name.split("/").at(-1) ?? ""
  );
  mocks.isDefaultProject.mockReset();
  mocks.isDefaultProject.mockReturnValue(false);
});

afterEach(() => {
  vi.restoreAllMocks();
});

describe("useProjectDatabaseDetail", () => {
  test("loads the database and warms metadata", async () => {
    const database = {
      name: "instances/inst1/databases/db1",
      project: "projects/proj1",
    };
    mocks.getDatabaseByName.mockReturnValue(database);
    mocks.getOrFetchDatabaseByName.mockResolvedValue(database);
    mocks.getOrFetchDatabaseMetadata.mockResolvedValue({});

    let latest: ReturnType<typeof useProjectDatabaseDetail> | undefined;

    const { render, unmount } = renderIntoContainer(
      createElement(HookProbe, {
        projectId: "proj1",
        instanceId: "inst1",
        databaseName: "db1",
        onValue: (value) => {
          latest = value;
        },
      })
    );

    act(() => {
      render();
    });
    await waitFor(() => latest?.ready === true);

    expect(mocks.getOrFetchDatabaseByName).toHaveBeenCalledWith(
      "instances/inst1/databases/db1"
    );
    expect(mocks.getOrFetchDatabaseMetadata).toHaveBeenCalledWith({
      database: "instances/inst1/databases/db1",
      silent: true,
    });
    expect(latest?.databaseName).toBe("instances/inst1/databases/db1");
    expect(latest?.ready).toBe(true);
    expect(latest?.allowAlterSchema).toBe(true);
    expect(latest?.isDefaultProject).toBe(false);

    unmount();
  });

  test("ignores metadata permission failures", async () => {
    const database = {
      name: "instances/inst1/databases/db1",
      project: "projects/proj1",
    };
    mocks.getDatabaseByName.mockReturnValue(database);
    mocks.getOrFetchDatabaseByName.mockResolvedValue(database);
    mocks.getOrFetchDatabaseMetadata.mockRejectedValue(
      new Error("permission denied")
    );

    let latest: ReturnType<typeof useProjectDatabaseDetail> | undefined;

    const { render, unmount } = renderIntoContainer(
      createElement(HookProbe, {
        projectId: "proj1",
        instanceId: "inst1",
        databaseName: "db1",
        onValue: (value) => {
          latest = value;
        },
      })
    );

    act(() => {
      render();
    });
    await waitFor(() => latest?.loading === false);

    expect(latest?.databaseName).toBe("instances/inst1/databases/db1");
    expect(mocks.replace).not.toHaveBeenCalled();

    unmount();
  });

  test("redirects to the canonical project route", async () => {
    const database = {
      name: "instances/inst1/databases/db1",
      project: "projects/proj2",
    };
    mocks.getDatabaseByName.mockReturnValue(database);
    mocks.getOrFetchDatabaseByName.mockResolvedValue(database);
    mocks.getOrFetchDatabaseMetadata.mockResolvedValue({});

    const { render, unmount } = renderIntoContainer(
      createElement(HookProbe, {
        projectId: "proj1",
        instanceId: "inst1",
        databaseName: "db1",
        routeName: mocks.routeNames.databaseRevisionDetail,
        revisionId: "123",
        hash: "#revision",
        query: { foo: "bar" },
        onValue: () => {},
      })
    );

    act(() => {
      render();
    });
    await waitFor(() => mocks.replace.mock.calls.length > 0);

    expect(mocks.replace).toHaveBeenCalledWith({
      name: mocks.routeNames.databaseRevisionDetail,
      params: {
        projectId: "proj2",
        instanceId: "inst1",
        databaseName: "db1",
        revisionId: "123",
      },
      hash: "#revision",
      query: { foo: "bar" },
    });

    unmount();
  });

  test("does not refetch database detail when only hash or query changes", async () => {
    const database = {
      name: "instances/inst1/databases/db1",
      project: "projects/proj1",
    };
    mocks.getDatabaseByName.mockReturnValue(database);
    mocks.getOrFetchDatabaseByName.mockResolvedValue(database);
    mocks.getOrFetchDatabaseMetadata.mockResolvedValue({});

    let latest: ReturnType<typeof useProjectDatabaseDetail> | undefined;

    const initialElement = createElement(HookProbe, {
      projectId: "proj1",
      instanceId: "inst1",
      databaseName: "db1",
      hash: "#overview",
      query: { schema: "public" },
      onValue: (value) => {
        latest = value;
      },
    });
    const { render, unmount } = renderIntoContainer(initialElement);

    act(() => {
      render();
    });
    await waitFor(() => latest?.ready === true);

    const initialDatabaseFetchCount =
      mocks.getOrFetchDatabaseByName.mock.calls.length;
    const initialMetadataFetchCount =
      mocks.getOrFetchDatabaseMetadata.mock.calls.length;

    act(() => {
      render(
        createElement(HookProbe, {
          projectId: "proj1",
          instanceId: "inst1",
          databaseName: "db1",
          hash: "#revision",
          query: { schema: "archive" },
          onValue: (value) => {
            latest = value;
          },
        })
      );
    });

    await act(async () => {
      await Promise.resolve();
      await Promise.resolve();
    });

    expect(mocks.getOrFetchDatabaseByName).toHaveBeenCalledTimes(
      initialDatabaseFetchCount
    );
    expect(mocks.getOrFetchDatabaseMetadata).toHaveBeenCalledTimes(
      initialMetadataFetchCount
    );
    expect(latest?.ready).toBe(true);

    unmount();
  });
});

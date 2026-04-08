import { act, createElement } from "react";
import { createRoot } from "react-dom/client";
import { beforeEach, describe, expect, test, vi } from "vitest";
import type { UseProjectDatabaseDetailOptions } from "./useProjectDatabaseDetail";

(
  globalThis as { IS_REACT_ACT_ENVIRONMENT?: boolean }
).IS_REACT_ACT_ENVIRONMENT = true;

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
    extractProjectResourceName: vi.fn((name: string) =>
      name.split("/").at(-1) ?? ""
    ),
    getInstanceResource: vi.fn((database: { instanceResource?: unknown }) =>
      database.instanceResource ?? {}
    ),
    replace: vi.fn(),
    getOrFetchDatabaseByName: vi.fn(),
    getOrFetchDatabaseMetadata: vi.fn(),
    getDatabaseByName: vi.fn(),
    isDefaultProject: vi.fn(() => false),
    instanceV1HasAlterSchema: vi.fn(() => true),
    routeNames: {
      databaseDetail: "workspace.project.database.detail",
      databaseChangelogDetail: "workspace.project.database.changelog.detail",
      databaseRevisionDetail: "workspace.project.database.revision.detail",
    },
  };
});

const databaseStore = {
  getOrFetchDatabaseByName: mocks.getOrFetchDatabaseByName,
  getDatabaseByName: mocks.getDatabaseByName,
};

const dbSchemaStore = {
  getOrFetchDatabaseMetadata: mocks.getOrFetchDatabaseMetadata,
};

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
  useDatabaseV1Store: () => databaseStore,
  useDBSchemaV1Store: () => dbSchemaStore,
}));

vi.mock("@/utils", () => ({
  getInstanceResource: mocks.getInstanceResource,
  instanceV1HasAlterSchema: mocks.instanceV1HasAlterSchema,
  isDev: () => true,
}));

vi.mock("@/utils/v1/project", () => ({
  extractProjectResourceName: mocks.extractProjectResourceName,
}));

vi.mock("@/types", () => ({
  isDefaultProject: mocks.isDefaultProject,
}));

vi.mock("@/types/v1/project", () => ({
  isDefaultProject: mocks.isDefaultProject,
}));

type UseProjectDatabaseDetail = typeof import("./useProjectDatabaseDetail").useProjectDatabaseDetail;

let useProjectDatabaseDetail: UseProjectDatabaseDetail;

function HookProbe(
  props: UseProjectDatabaseDetailOptions & {
    onValue: (value: ReturnType<UseProjectDatabaseDetail>) => void;
  }
) {
  props.onValue(useProjectDatabaseDetail(props));
  return null;
}

const renderIntoContainer = (element: ReturnType<typeof createElement>) => {
  const container = document.createElement("div");
  const root = createRoot(container);

  return {
    container,
    root,
    render: () => {
      root.render(element);
    },
    unmount: () =>
      act(() => {
        root.unmount();
      }),
  };
};

const createDeferred = <T,>() => {
  let resolve!: (value: T) => void;
  let reject!: (reason?: unknown) => void;
  const promise = new Promise<T>((res, rej) => {
    resolve = res;
    reject = rej;
  });
  return { promise, resolve, reject };
};

beforeEach(() => {
  mocks.replace.mockReset();
  mocks.extractProjectResourceName.mockReset();
  mocks.extractProjectResourceName.mockImplementation((name: string) =>
    name.split("/").at(-1) ?? ""
  );
  mocks.getInstanceResource.mockReset();
  mocks.getInstanceResource.mockImplementation(
    (database: { instanceResource?: unknown }) => database.instanceResource ?? {}
  );
  mocks.getOrFetchDatabaseByName.mockReset();
  mocks.getOrFetchDatabaseMetadata.mockReset();
  mocks.getDatabaseByName.mockReset();
  mocks.isDefaultProject.mockReset();
  mocks.isDefaultProject.mockReturnValue(false);
  mocks.instanceV1HasAlterSchema.mockReset();
  mocks.instanceV1HasAlterSchema.mockReturnValue(true);
  mocks.localStorage.getItem.mockReset();
  mocks.localStorage.getItem.mockReturnValue(null);
  mocks.localStorage.setItem.mockReset();
  mocks.localStorage.removeItem.mockReset();
  mocks.localStorage.clear.mockReset();
});

beforeEach(async () => {
  vi.resetModules();
  ({ useProjectDatabaseDetail } = await import("./useProjectDatabaseDetail"));
});

describe("useProjectDatabaseDetail", () => {
  test("loads the database and warms metadata", async () => {
    const database = {
      name: "instances/inst1/databases/db1",
      project: "projects/proj1",
    };
    const databaseDeferred = createDeferred<typeof database>();
    const metadataDeferred = createDeferred<Record<string, never>>();
    mocks.getDatabaseByName.mockReturnValue(database);
    mocks.getOrFetchDatabaseByName.mockReturnValue(databaseDeferred.promise);
    mocks.getOrFetchDatabaseMetadata.mockReturnValue(metadataDeferred.promise);

    let latest:
      | ReturnType<UseProjectDatabaseDetail>
      | undefined;

    const { unmount, render } = renderIntoContainer(
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
    await act(async () => {
      databaseDeferred.resolve(database);
      metadataDeferred.resolve({});
    });

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
    const databaseDeferred = createDeferred<typeof database>();
    const metadataDeferred = createDeferred<never>();
    mocks.getDatabaseByName.mockReturnValue(database);
    mocks.getOrFetchDatabaseByName.mockReturnValue(databaseDeferred.promise);
    mocks.getOrFetchDatabaseMetadata.mockReturnValue(metadataDeferred.promise);

    let latest:
      | ReturnType<UseProjectDatabaseDetail>
      | undefined;

    const { unmount, render } = renderIntoContainer(
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
    void metadataDeferred.promise.catch(() => {
      // The hook swallows metadata permission failures.
    });
    await act(async () => {
      databaseDeferred.resolve(database);
      metadataDeferred.reject(new Error("permission denied"));
    });

    expect(latest?.databaseName).toBe("instances/inst1/databases/db1");
    expect(mocks.replace).not.toHaveBeenCalled();

    unmount();
  });

  test("redirects to the canonical project route", async () => {
    const database = {
      name: "instances/inst1/databases/db1",
      project: "projects/proj2",
    };
    const databaseDeferred = createDeferred<typeof database>();
    const metadataDeferred = createDeferred<Record<string, never>>();
    mocks.getDatabaseByName.mockReturnValue(database);
    mocks.getOrFetchDatabaseByName.mockReturnValue(databaseDeferred.promise);
    mocks.getOrFetchDatabaseMetadata.mockReturnValue(metadataDeferred.promise);

    const { unmount, render } = renderIntoContainer(
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
    await act(async () => {
      databaseDeferred.resolve(database);
      metadataDeferred.resolve({});
    });

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
});

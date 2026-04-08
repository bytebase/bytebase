import { act, createElement } from "react";
import { createRoot } from "react-dom/client";
import { beforeEach, describe, expect, test, vi } from "vitest";
import { PROJECT_V1_ROUTE_DATABASE_REVISION_DETAIL } from "@/router/dashboard/projectV1";
import {
  type UseProjectDatabaseDetailOptions,
  useProjectDatabaseDetail,
} from "./useProjectDatabaseDetail";

(
  globalThis as { IS_REACT_ACT_ENVIRONMENT?: boolean }
).IS_REACT_ACT_ENVIRONMENT = true;

const mocks = vi.hoisted(() => ({
  replace: vi.fn(),
  getOrFetchDatabaseByName: vi.fn(),
  getOrFetchDatabaseMetadata: vi.fn(),
  getDatabaseByName: vi.fn(),
}));

vi.mock("@/router", () => ({
  router: { replace: mocks.replace },
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

function HookProbe(
  props: UseProjectDatabaseDetailOptions & {
    onValue: (value: ReturnType<typeof useProjectDatabaseDetail>) => void;
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

const flushAsync = () =>
  new Promise<void>((resolve) => {
    setTimeout(resolve, 0);
  });

beforeEach(() => {
  mocks.replace.mockReset();
  mocks.getOrFetchDatabaseByName.mockReset();
  mocks.getOrFetchDatabaseMetadata.mockReset();
  mocks.getDatabaseByName.mockReset();
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

    let latest:
      | ReturnType<typeof useProjectDatabaseDetail>
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

    await act(async () => {
      render();
      await flushAsync();
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
    mocks.getDatabaseByName.mockReturnValue(database);
    mocks.getOrFetchDatabaseByName.mockResolvedValue(database);
    mocks.getOrFetchDatabaseMetadata.mockRejectedValue(
      new Error("permission denied")
    );

    let latest:
      | ReturnType<typeof useProjectDatabaseDetail>
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

    await act(async () => {
      render();
      await flushAsync();
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
    mocks.getDatabaseByName.mockReturnValue(database);
    mocks.getOrFetchDatabaseByName.mockResolvedValue(database);
    mocks.getOrFetchDatabaseMetadata.mockResolvedValue({});

    const { unmount, render } = renderIntoContainer(
      createElement(HookProbe, {
        projectId: "proj1",
        instanceId: "inst1",
        databaseName: "db1",
        routeName: PROJECT_V1_ROUTE_DATABASE_REVISION_DETAIL,
        revisionId: "123",
        hash: "#revision",
        query: { foo: "bar" },
        onValue: () => {},
      })
    );

    await act(async () => {
      render();
      await flushAsync();
    });

    expect(mocks.replace).toHaveBeenCalledWith({
      name: PROJECT_V1_ROUTE_DATABASE_REVISION_DETAIL,
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

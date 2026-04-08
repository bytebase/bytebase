import { act, createElement } from "react";
import { createRoot } from "react-dom/client";
import { beforeEach, describe, expect, test, vi } from "vitest";

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
    routerPush: vi.fn(),
    useProjectDatabaseDetail: vi.fn(),
    useDatabaseV1Store: vi.fn(),
    getOrFetchDatabaseByName: vi.fn(),
    extractDatabaseResourceName: vi.fn((name: string) => ({
      instance: "instances/inst1",
      instanceName: "inst1",
      database: name,
      databaseName: name.split("/databases/").at(-1) ?? "",
    })),
    extractInstanceResourceName: vi.fn(
      (name: string) => name.split("/").at(-1) ?? ""
    ),
    extractProjectResourceName: vi.fn(
      (name: string) => name.split("/").at(-1) ?? ""
    ),
    useTranslation: vi.fn(() => ({
      t: (key: string) => key,
    })),
    LoaderCircle: vi.fn(() => <div data-testid="spinner" />),
    RevisionDetailPanel: vi.fn(({ revisionName }: { revisionName: string }) => (
      <div data-testid="revision-detail-panel">{revisionName}</div>
    )),
    routeNames: {
      databases: "workspace.project.databases",
      databaseDetail: "workspace.project.database.detail",
      databaseRevisionDetail: "workspace.project.database.revision.detail",
    },
  };
});

type DatabaseRevisionDetailPageComponent =
  typeof import("./DatabaseRevisionDetailPage").DatabaseRevisionDetailPage;

vi.mock("lucide-react", () => ({
  LoaderCircle: mocks.LoaderCircle,
}));

vi.mock("react-i18next", () => ({
  useTranslation: mocks.useTranslation,
}));

vi.mock("@/react/components/revision", () => ({
  RevisionDetailPanel: mocks.RevisionDetailPanel,
}));

vi.mock("@/router", () => ({
  router: {
    push: mocks.routerPush,
  },
}));

vi.mock("@/router/dashboard/projectV1", () => ({
  PROJECT_V1_ROUTE_DATABASES: mocks.routeNames.databases,
  PROJECT_V1_ROUTE_DATABASE_DETAIL: mocks.routeNames.databaseDetail,
  PROJECT_V1_ROUTE_DATABASE_REVISION_DETAIL:
    mocks.routeNames.databaseRevisionDetail,
}));

vi.mock("./database-detail/useProjectDatabaseDetail", () => ({
  useProjectDatabaseDetail: mocks.useProjectDatabaseDetail,
}));

vi.mock("@/store", () => ({
  useDatabaseV1Store: mocks.useDatabaseV1Store,
}));

vi.mock("@/utils/v1/database", () => ({
  extractDatabaseResourceName: mocks.extractDatabaseResourceName,
}));

vi.mock("@/utils/v1/instance", () => ({
  extractInstanceResourceName: mocks.extractInstanceResourceName,
}));

vi.mock("@/utils/v1/project", () => ({
  extractProjectResourceName: mocks.extractProjectResourceName,
}));

const {
  DatabaseRevisionDetailPage,
}: {
  DatabaseRevisionDetailPage: DatabaseRevisionDetailPageComponent;
} = await import("./DatabaseRevisionDetailPage");

const renderIntoContainer = (element: ReturnType<typeof createElement>) => {
  const container = document.createElement("div");
  const root = createRoot(container);

  return {
    container,
    render: (nextElement = element) => {
      act(() => {
        root.render(nextElement);
      });
    },
    unmount: () =>
      act(() => {
        root.unmount();
      }),
  };
};

beforeEach(() => {
  mocks.localStorage.getItem.mockReset();
  mocks.localStorage.getItem.mockReturnValue(null);
  mocks.localStorage.setItem.mockReset();
  mocks.localStorage.removeItem.mockReset();
  mocks.localStorage.clear.mockReset();
  mocks.routerPush.mockReset();
  mocks.useProjectDatabaseDetail.mockReset();
  mocks.useDatabaseV1Store.mockReset();
  mocks.getOrFetchDatabaseByName.mockReset();
  mocks.extractDatabaseResourceName.mockReset();
  mocks.extractDatabaseResourceName.mockImplementation((name: string) => ({
    instance: "instances/inst1",
    instanceName: "inst1",
    database: name,
    databaseName: name.split("/databases/").at(-1) ?? "",
  }));
  mocks.extractInstanceResourceName.mockReset();
  mocks.extractInstanceResourceName.mockImplementation(
    (name: string) => name.split("/").at(-1) ?? ""
  );
  mocks.extractProjectResourceName.mockReset();
  mocks.extractProjectResourceName.mockImplementation(
    (name: string) => name.split("/").at(-1) ?? ""
  );
  mocks.useTranslation.mockReset();
  mocks.useTranslation.mockReturnValue({
    t: (key: string) => key,
  });
  mocks.LoaderCircle.mockClear();
  mocks.RevisionDetailPanel.mockClear();
});

describe("DatabaseRevisionDetailPage", () => {
  test("shows a spinner while the shared database hook is loading", async () => {
    mocks.useProjectDatabaseDetail.mockReturnValue({
      database: {
        name: "instances/inst1/databases/db-from-hook",
        project: "projects/proj1",
      },
      databaseName: "instances/inst1/databases/db-from-hook",
      loading: true,
      ready: false,
      allowAlterSchema: true,
      isDefaultProject: false,
    });
    mocks.useDatabaseV1Store.mockReturnValue({
      getOrFetchDatabaseByName: mocks.getOrFetchDatabaseByName,
    });
    mocks.getOrFetchDatabaseByName.mockResolvedValue({
      name: "instances/inst1/databases/db-from-prop",
      project: "projects/proj1",
    });

    const { container, render, unmount } = renderIntoContainer(
      createElement(DatabaseRevisionDetailPage, {
        project: "projects/proj1",
        instance: "instances/inst1",
        database: "instances/inst1/databases/db-from-prop",
        revisionId: "7",
      })
    );

    render();

    await act(async () => {
      await Promise.resolve();
      await Promise.resolve();
    });

    expect(container.querySelector('[data-testid="spinner"]')).not.toBeNull();
    expect(mocks.RevisionDetailPanel).not.toHaveBeenCalled();

    unmount();
  });

  test("builds the revision name from the resolved database detail", async () => {
    mocks.useProjectDatabaseDetail.mockReturnValue({
      database: {
        name: "instances/inst1/databases/db-from-hook",
        project: "projects/proj1",
      },
      databaseName: "instances/inst1/databases/db-from-hook",
      loading: false,
      ready: true,
      allowAlterSchema: true,
      isDefaultProject: false,
    });
    mocks.useDatabaseV1Store.mockReturnValue({
      getOrFetchDatabaseByName: mocks.getOrFetchDatabaseByName,
    });
    mocks.getOrFetchDatabaseByName.mockResolvedValue({
      name: "instances/inst1/databases/db-from-prop",
      project: "projects/proj1",
    });

    const { container, render, unmount } = renderIntoContainer(
      createElement(DatabaseRevisionDetailPage, {
        project: "projects/proj1",
        instance: "instances/inst1",
        database: "instances/inst1/databases/db-from-prop",
        revisionId: "7",
      })
    );

    render();

    await act(async () => {
      await Promise.resolve();
      await Promise.resolve();
    });

    expect(mocks.useProjectDatabaseDetail).toHaveBeenCalledWith({
      projectId: "proj1",
      instanceId: "inst1",
      databaseName: "db-from-prop",
      routeName: mocks.routeNames.databaseRevisionDetail,
      revisionId: "7",
    });
    expect(mocks.RevisionDetailPanel).toHaveBeenCalled();
    expect(mocks.RevisionDetailPanel.mock.calls[0]?.[0]).toEqual({
      revisionName: "instances/inst1/databases/db-from-hook/revisions/7",
    });

    expect(container.textContent).toContain(
      "instances/inst1/databases/db-from-hook/revisions/7"
    );

    const clickButton = async (label: string) => {
      const button = Array.from(container.querySelectorAll("button")).find(
        (candidate) => candidate.textContent === label
      );

      await act(async () => {
        button?.dispatchEvent(new MouseEvent("click", { bubbles: true }));
      });
    };

    await clickButton("common.databases");
    expect(mocks.routerPush).toHaveBeenCalledWith({
      name: mocks.routeNames.databases,
      params: {
        projectId: "proj1",
      },
    });

    await clickButton("db-from-prop");
    expect(mocks.routerPush).toHaveBeenCalledWith({
      name: mocks.routeNames.databaseDetail,
      params: {
        projectId: "proj1",
        instanceId: "inst1",
        databaseName: "db-from-prop",
      },
    });

    await clickButton("database.revision.self");
    expect(mocks.routerPush).toHaveBeenCalledWith({
      name: mocks.routeNames.databaseDetail,
      params: {
        projectId: "proj1",
        instanceId: "inst1",
        databaseName: "db-from-prop",
      },
      hash: "#revision",
    });

    const databaseBreadcrumb = Array.from(
      container.querySelectorAll("button")
    ).find((button) => button.textContent === "db-from-prop");

    expect(databaseBreadcrumb).not.toBeNull();

    unmount();
  });
});

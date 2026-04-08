import type { ReactNode } from "react";
import { act, createElement } from "react";
import { createRoot } from "react-dom/client";
import { beforeEach, describe, expect, test, vi } from "vitest";
import {
  PROJECT_DATABASE_DETAIL_TAB_OVERVIEW,
  PROJECT_DATABASE_DETAIL_TAB_REVISION,
} from "./database-detail/tabs";

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
    routerReplace: vi.fn(),
    useProjectDatabaseDetail: vi.fn(),
    LoaderCircle: vi.fn(() => <div data-testid="spinner" />),
    Tabs: vi.fn(
      ({ value, children }: { value: string; children: ReactNode }) => (
        <div data-testid="tabs" data-value={value}>
          {children}
        </div>
      )
    ),
    TabsList: vi.fn(({ children }: { children: ReactNode }) => (
      <div data-testid="tabs-list">{children}</div>
    )),
    TabsTrigger: vi.fn(
      ({ value, children }: { value: string; children: ReactNode }) => (
        <button type="button" data-testid={`tab-${value}`}>
          {children}
        </button>
      )
    ),
    useTranslation: vi.fn(() => ({
      t: (key: string) => key,
    })),
    routeNames: {
      databaseDetail: "workspace.project.database.detail",
    },
  };
});

let latestTabsOnValueChange:
  | ((tab: string | number | null) => void)
  | undefined;

type ProjectDatabaseDetailPageComponent =
  typeof import("./ProjectDatabaseDetailPage").ProjectDatabaseDetailPage;
let ProjectDatabaseDetailPage: ProjectDatabaseDetailPageComponent;

vi.mock("lucide-react", () => ({
  LoaderCircle: mocks.LoaderCircle,
}));

vi.mock("react-i18next", () => ({
  useTranslation: mocks.useTranslation,
}));

vi.mock("@/router", () => ({
  router: {
    replace: mocks.routerReplace,
  },
}));

vi.mock("@/router/dashboard/projectV1", () => ({
  PROJECT_V1_ROUTE_DATABASE_DETAIL: mocks.routeNames.databaseDetail,
}));

vi.mock("@/react/components/ui/tabs", () => ({
  Tabs: ({
    value,
    children,
    onValueChange,
  }: {
    value: string;
    children: ReactNode;
    onValueChange?: (tab: string | number | null) => void;
  }) => {
    latestTabsOnValueChange = onValueChange;
    return (
      <div data-testid="tabs" data-value={value}>
        {children}
      </div>
    );
  },
  TabsList: mocks.TabsList,
  TabsTrigger: mocks.TabsTrigger,
}));

vi.mock("./database-detail/useProjectDatabaseDetail", () => ({
  useProjectDatabaseDetail: mocks.useProjectDatabaseDetail,
}));

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
  mocks.routerReplace.mockReset();
  mocks.useProjectDatabaseDetail.mockReset();
  mocks.LoaderCircle.mockClear();
  mocks.TabsList.mockClear();
  mocks.TabsTrigger.mockClear();
  mocks.useTranslation.mockReset();
  mocks.useTranslation.mockReturnValue({
    t: (key: string) => key,
  });
  latestTabsOnValueChange = undefined;
});

beforeEach(async () => {
  vi.resetModules();
  ({ ProjectDatabaseDetailPage } = await import("./ProjectDatabaseDetailPage"));
});

describe("ProjectDatabaseDetailPage", () => {
  test("shows a spinner while the shared database hook is loading", async () => {
    mocks.useProjectDatabaseDetail.mockReturnValue({
      database: undefined,
      databaseName: "instances/inst1/databases/db1",
      loading: true,
      ready: false,
      allowAlterSchema: false,
      isDefaultProject: false,
    });

    const { container, render, unmount } = renderIntoContainer(
      createElement(ProjectDatabaseDetailPage, {
        projectId: "proj1",
        instanceId: "inst1",
        databaseName: "db1",
      })
    );

    render();

    await act(async () => {
      await Promise.resolve();
      await Promise.resolve();
    });

    expect(container.querySelector('[data-testid="spinner"]')).not.toBeNull();
    expect(container.querySelector('[data-testid="tabs"]')).toBeNull();

    unmount();
  });

  test("selects the tab from a valid hash and keeps the query when the tab changes", async () => {
    mocks.useProjectDatabaseDetail.mockReturnValue({
      database: {
        name: "instances/inst1/databases/db1",
        project: "projects/proj1",
      },
      databaseName: "instances/inst1/databases/db1",
      loading: false,
      ready: true,
      allowAlterSchema: true,
      isDefaultProject: false,
    });

    const { container, render, unmount } = renderIntoContainer(
      createElement(ProjectDatabaseDetailPage, {
        projectId: "proj1",
        instanceId: "inst1",
        databaseName: "db1",
        hash: `#${PROJECT_DATABASE_DETAIL_TAB_REVISION}`,
        query: { foo: "bar", page: "2" },
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
      databaseName: "db1",
      hash: `#${PROJECT_DATABASE_DETAIL_TAB_REVISION}`,
      query: { foo: "bar", page: "2" },
    });
    expect(container.querySelector('[data-testid="tabs"]')).not.toBeNull();
    expect(
      container
        .querySelector('[data-testid="tabs"]')
        ?.getAttribute("data-value")
    ).toBe(PROJECT_DATABASE_DETAIL_TAB_REVISION);
    expect(
      Array.from(container.querySelectorAll("button")).map(
        (button) => button.textContent
      )
    ).toEqual([
      "common.overview",
      "common.changelog",
      "database.revision.self",
      "common.catalog",
      "common.settings",
    ]);

    await act(async () => {
      latestTabsOnValueChange?.("changelog");
      await Promise.resolve();
    });

    expect(mocks.routerReplace).toHaveBeenCalledWith({
      name: mocks.routeNames.databaseDetail,
      params: {
        projectId: "proj1",
        instanceId: "inst1",
        databaseName: "db1",
      },
      hash: "#changelog",
      query: { foo: "bar", page: "2" },
    });
    expect(
      container
        .querySelector('[data-testid="tabs"]')
        ?.getAttribute("data-value")
    ).toBe("changelog");

    unmount();
  });

  test("falls back to overview for an invalid hash", async () => {
    mocks.useProjectDatabaseDetail.mockReturnValue({
      database: {
        name: "instances/inst1/databases/db1",
        project: "projects/proj1",
      },
      databaseName: "instances/inst1/databases/db1",
      loading: false,
      ready: true,
      allowAlterSchema: true,
      isDefaultProject: false,
    });

    const { container, render, unmount } = renderIntoContainer(
      createElement(ProjectDatabaseDetailPage, {
        projectId: "proj1",
        instanceId: "inst1",
        databaseName: "db1",
        hash: "#not-a-real-tab",
      })
    );

    render();

    await act(async () => {
      await Promise.resolve();
      await Promise.resolve();
    });

    expect(
      container
        .querySelector('[data-testid="tabs"]')
        ?.getAttribute("data-value")
    ).toBe(PROJECT_DATABASE_DETAIL_TAB_OVERVIEW);
    expect(
      Array.from(container.querySelectorAll("button")).map(
        (button) => button.textContent
      )
    ).toEqual([
      "common.overview",
      "common.changelog",
      "database.revision.self",
      "common.catalog",
      "common.settings",
    ]);

    unmount();
  });
});

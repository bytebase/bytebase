import type { ReactNode } from "react";
import { act, createElement } from "react";
import { createRoot } from "react-dom/client";
import { beforeEach, describe, expect, test, vi } from "vitest";
import {
  PROJECT_DATABASE_DETAIL_TAB_OVERVIEW,
  PROJECT_DATABASE_DETAIL_TAB_REVISION,
  PROJECT_DATABASE_DETAIL_TAB_SETTING,
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
    windowOpen: vi.fn(),
    routerReplace: vi.fn(),
    routerPush: vi.fn(),
    routerResolve: vi.fn(() => ({ fullPath: "/sql-editor/projects/proj1" })),
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
    EnvironmentSelect: vi.fn(() => <div data-testid="environment-select" />),
    useTranslation: vi.fn(() => ({
      t: (key: string) => key,
    })),
    pushNotification: vi.fn(),
    transferProjectDrawer: vi.fn(
      ({
        open,
        onTransfer,
      }: {
        open: boolean;
        onTransfer: (projectName: string) => Promise<void>;
      }) => (
        <div
          data-testid="transfer-project-drawer"
          data-open={open ? "true" : "false"}
        >
          {open && (
            <button
              type="button"
              data-testid="transfer-project-confirm"
              onClick={() => void onTransfer("projects/proj2")}
            >
              transfer-project-confirm
            </button>
          )}
        </div>
      )
    ),
    DatabaseChangelogPanel: vi.fn(() => (
      <div data-testid="database-changelog-panel" />
    )),
    DatabaseRevisionPanel: vi.fn(() => (
      <div data-testid="database-revision-panel" />
    )),
    useProjectV1Store: vi.fn(() => ({
      getProjectByName: vi.fn(() => ({
        name: "projects/proj1",
        title: "Project 1",
      })),
    })),
    useDatabaseV1Store: vi.fn(() => ({
      syncDatabase: vi.fn(),
      batchUpdateDatabases: vi.fn(),
      getOrFetchDatabaseByName: vi.fn(async (name: string) => ({
        name,
        project: "projects/proj1",
      })),
    })),
    useDBSchemaV1Store: vi.fn(() => ({
      getOrFetchDatabaseMetadata: vi.fn(),
      getDatabaseMetadata: vi.fn(() => undefined),
    })),
    usePermissionStore: vi.fn(() => ({
      currentPermissions: new Set([
        "bb.databases.sync",
        "bb.databases.getSchema",
        "bb.databases.update",
        "bb.plans.create",
        "bb.sheets.create",
      ]),
      currentPermissionsInProjectV1: vi.fn(
        () =>
          new Set([
            "bb.databases.sync",
            "bb.databases.getSchema",
            "bb.databases.update",
            "bb.plans.create",
            "bb.sheets.create",
          ])
      ),
    })),
    getDatabaseSDLSchema: vi.fn(),
    schemaDiagram: vi.fn(() => <div data-testid="schema-diagram-vue" />),
    preCreateIssue: vi.fn(),
    useActuatorV1Store: vi.fn(() => ({
      serverInfo: {
        defaultProject: "projects/default",
      },
    })),
    pinia: {
      install: vi.fn(),
    },
    highlightPlugin: {
      install: vi.fn(),
    },
    i18nPlugin: {
      install: vi.fn(),
    },
    naivePlugin: {
      install: vi.fn(),
    },
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

vi.mock("lucide-react", async (importOriginal) => {
  const actual = await importOriginal<typeof import("lucide-react")>();
  return {
    ...actual,
    LoaderCircle: mocks.LoaderCircle,
  };
});

vi.mock("react-i18next", () => ({
  useTranslation: mocks.useTranslation,
}));

vi.mock("@/router", () => ({
  router: {
    replace: mocks.routerReplace,
    push: mocks.routerPush,
    resolve: mocks.routerResolve,
    currentRoute: { value: { name: mocks.routeNames.databaseDetail } },
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

vi.mock("@/react/components/EnvironmentSelect", () => ({
  EnvironmentSelect: mocks.EnvironmentSelect,
}));

vi.mock("./database-detail/useProjectDatabaseDetail", () => ({
  useProjectDatabaseDetail: mocks.useProjectDatabaseDetail,
}));

vi.mock("./database-detail/panels/DatabaseChangelogPanel", () => ({
  DatabaseChangelogPanel: mocks.DatabaseChangelogPanel,
}));

vi.mock("./database-detail/panels/DatabaseRevisionPanel", () => ({
  DatabaseRevisionPanel: mocks.DatabaseRevisionPanel,
}));

vi.mock("@/store", () => ({
  pinia: mocks.pinia,
  pushNotification: mocks.pushNotification,
  useProjectV1Store: mocks.useProjectV1Store,
  useDatabaseV1Store: mocks.useDatabaseV1Store,
  useDBSchemaV1Store: mocks.useDBSchemaV1Store,
  usePermissionStore: mocks.usePermissionStore,
}));

vi.mock("@/react/components/database", () => ({
  TransferProjectDrawer: mocks.transferProjectDrawer,
}));

vi.mock("@/connect", () => ({
  databaseServiceClientConnect: {
    getDatabaseSDLSchema: mocks.getDatabaseSDLSchema,
  },
}));

vi.mock("@/components/SchemaDiagram", () => ({
  default: mocks.schemaDiagram,
}));

vi.mock("@/plugins/highlight", () => ({
  default: mocks.highlightPlugin,
}));

vi.mock("@/plugins/i18n", () => ({
  default: mocks.i18nPlugin,
}));

vi.mock("@/plugins/naive-ui", () => ({
  default: mocks.naivePlugin,
}));

vi.mock("@/store/modules/v1/actuator", () => ({
  useActuatorV1Store: mocks.useActuatorV1Store,
}));

vi.mock("@/components/Plan/logic/issue", () => ({
  preCreateIssue: mocks.preCreateIssue,
}));

vi.mock("@/components/MonacoEditor/editor", () => ({
  createMonacoDiffEditor: vi.fn(),
  createMonacoEditor: vi.fn(),
}));

vi.mock("@/react/components/monaco", () => ({
  ReadonlyMonaco: vi.fn(() => null),
}));

vi.mock("@/components/MonacoEditor/services", () => ({
  initializeMonacoServices: vi.fn(),
}));

vi.mock("monaco-editor", () => ({}));

vi.mock(
  "@codingame/monaco-vscode-editor-api/vscode/src/vs/editor/standalone/browser/standalone-tokens.css",
  () => ({})
);

vi.mock("pouchdb", () => {
  class MockPouchDB {
    static plugin = vi.fn();
  }
  return {
    default: MockPouchDB,
  };
});

vi.mock("pouchdb-find", () => ({
  default: {},
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
  vi.stubGlobal("open", mocks.windowOpen);
  mocks.localStorage.getItem.mockReset();
  mocks.localStorage.getItem.mockReturnValue(null);
  mocks.localStorage.setItem.mockReset();
  mocks.localStorage.removeItem.mockReset();
  mocks.localStorage.clear.mockReset();
  mocks.routerReplace.mockReset();
  mocks.routerPush.mockReset();
  mocks.routerResolve.mockReset();
  mocks.routerResolve.mockReturnValue({
    fullPath: "/sql-editor/projects/proj1",
  });
  mocks.useProjectDatabaseDetail.mockReset();
  mocks.LoaderCircle.mockClear();
  mocks.TabsList.mockClear();
  mocks.TabsTrigger.mockClear();
  mocks.EnvironmentSelect.mockClear();
  mocks.useTranslation.mockReset();
  mocks.useTranslation.mockReturnValue({
    t: (key: string) => key,
  });
  mocks.pushNotification.mockReset();
  mocks.transferProjectDrawer.mockClear();
  mocks.DatabaseChangelogPanel.mockClear();
  mocks.DatabaseRevisionPanel.mockClear();
  mocks.useProjectV1Store.mockReset();
  mocks.useProjectV1Store.mockReturnValue({
    getProjectByName: vi.fn(() => ({
      name: "projects/proj1",
      title: "Project 1",
    })),
  });
  mocks.useDatabaseV1Store.mockReset();
  mocks.useDatabaseV1Store.mockReturnValue({
    syncDatabase: vi.fn(),
    batchUpdateDatabases: vi.fn(),
    getOrFetchDatabaseByName: vi.fn(async (name: string) => ({
      name,
      project: "projects/proj1",
    })),
  });
  mocks.useDBSchemaV1Store.mockReset();
  mocks.useDBSchemaV1Store.mockReturnValue({
    getOrFetchDatabaseMetadata: vi.fn(),
    getDatabaseMetadata: vi.fn(() => undefined),
  });
  mocks.usePermissionStore.mockReset();
  mocks.usePermissionStore.mockReturnValue({
    currentPermissions: new Set([
      "bb.databases.sync",
      "bb.databases.getSchema",
      "bb.databases.update",
      "bb.plans.create",
      "bb.sheets.create",
    ]),
    currentPermissionsInProjectV1: vi.fn(
      () =>
        new Set([
          "bb.databases.sync",
          "bb.databases.getSchema",
          "bb.databases.update",
          "bb.plans.create",
          "bb.sheets.create",
        ])
    ),
  });
  mocks.getDatabaseSDLSchema.mockReset();
  mocks.schemaDiagram.mockClear();
  mocks.preCreateIssue.mockReset();
  mocks.pinia.install.mockReset();
  mocks.highlightPlugin.install.mockReset();
  mocks.i18nPlugin.install.mockReset();
  mocks.naivePlugin.install.mockReset();
  mocks.useActuatorV1Store.mockReset();
  mocks.useActuatorV1Store.mockReturnValue({
    serverInfo: {
      defaultProject: "projects/default",
    },
  });
  mocks.windowOpen.mockReset();
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
    mocks.usePermissionStore.mockReturnValue({
      currentPermissions: new Set([
        "bb.databases.sync",
        "bb.databases.getSchema",
        "bb.databases.update",
        "bb.changelogs.list",
        "bb.revisions.list",
      ]),
      currentPermissionsInProjectV1: vi.fn(
        () =>
          new Set([
            "bb.databases.sync",
            "bb.databases.getSchema",
            "bb.databases.update",
            "bb.changelogs.list",
            "bb.revisions.list",
          ])
      ),
    });
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
    expect(mocks.DatabaseRevisionPanel).toHaveBeenCalledWith(
      expect.objectContaining({
        database: expect.objectContaining({
          name: "instances/inst1/databases/db1",
        }),
      }),
      undefined
    );
    expect(
      Array.from(container.querySelectorAll('[data-testid^="tab-"]')).map(
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
    expect(mocks.DatabaseChangelogPanel).toHaveBeenCalledWith(
      expect.objectContaining({
        database: expect.objectContaining({
          name: "instances/inst1/databases/db1",
        }),
      }),
      undefined
    );

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
      Array.from(container.querySelectorAll('[data-testid^="tab-"]')).map(
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

  test("shows the no-environment warning and routes to settings when its action is clicked", async () => {
    mocks.useProjectDatabaseDetail.mockReturnValue({
      database: {
        name: "instances/inst1/databases/db1",
        project: "projects/proj1",
        effectiveEnvironment: "",
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
        query: { foo: "bar" },
      })
    );

    render();

    await act(async () => {
      await Promise.resolve();
      await Promise.resolve();
    });

    expect(container.textContent).toContain("instance.no-environment");

    const warningAction = Array.from(container.querySelectorAll("button")).find(
      (button) => button.textContent === "database.edit-environment"
    );
    expect(warningAction).toBeDefined();

    await act(async () => {
      warningAction?.dispatchEvent(
        new MouseEvent("click", { bubbles: true, cancelable: true })
      );
      await Promise.resolve();
    });

    expect(
      container
        .querySelector('[data-testid="tabs"]')
        ?.getAttribute("data-value")
    ).toBe(PROJECT_DATABASE_DETAIL_TAB_SETTING);
    expect(mocks.routerReplace).toHaveBeenCalledWith({
      name: mocks.routeNames.databaseDetail,
      params: {
        projectId: "proj1",
        instanceId: "inst1",
        databaseName: "db1",
      },
      hash: `#${PROJECT_DATABASE_DETAIL_TAB_SETTING}`,
      query: { foo: "bar" },
    });

    unmount();
  });

  test("renders the header metadata and top-level action cluster", async () => {
    mocks.useProjectDatabaseDetail.mockReturnValue({
      database: {
        name: "instances/inst1/databases/db1",
        project: "projects/proj1",
        effectiveEnvironment: "environments/prod",
        release: "projects/proj1/releases/20260408",
        instanceResource: {
          name: "instances/inst1",
          title: "Instance 1",
        },
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
      })
    );

    render();

    await act(async () => {
      await Promise.resolve();
      await Promise.resolve();
    });

    expect(container.textContent).toContain("db1");
    expect(container.textContent).toContain("instances/inst1/databases/db1");
    expect(container.textContent).toContain("environments/prod");
    expect(container.textContent).toContain("Instance 1");
    expect(container.textContent).toContain("20260408");
    expect(
      Array.from(container.querySelectorAll("button")).map(
        (button) => button.textContent
      )
    ).toEqual(
      expect.arrayContaining([
        "sql-editor.self",
        "schema-diagram.self",
        "database.sync-database",
        "database.export-schema",
        "database.transfer-project",
        "database.change-database",
      ])
    );

    const schemaDiagramButton = Array.from(
      container.querySelectorAll("button")
    ).find((button) => button.textContent === "schema-diagram.self");
    expect(schemaDiagramButton).toBeDefined();

    await act(async () => {
      schemaDiagramButton?.dispatchEvent(
        new MouseEvent("click", { bubbles: true, cancelable: true })
      );
      await Promise.resolve();
      await Promise.resolve();
    });

    expect(
      mocks.useDBSchemaV1Store.mock.results.at(-1)?.value
        .getOrFetchDatabaseMetadata
    ).toHaveBeenCalledWith({
      database: "instances/inst1/databases/db1",
      skipCache: false,
    });
    expect(mocks.pinia.install).toHaveBeenCalled();
    expect(mocks.highlightPlugin.install).toHaveBeenCalled();
    expect(mocks.i18nPlugin.install).toHaveBeenCalled();
    expect(mocks.naivePlugin.install).toHaveBeenCalled();

    unmount();
  });

  test("renders the tab panels only when the matching list permissions are present", async () => {
    mocks.usePermissionStore.mockReturnValue({
      currentPermissions: new Set([
        "bb.databases.sync",
        "bb.databases.getSchema",
        "bb.databases.update",
        "bb.changelogs.list",
      ]),
      currentPermissionsInProjectV1: vi.fn(
        () =>
          new Set([
            "bb.databases.sync",
            "bb.databases.getSchema",
            "bb.databases.update",
            "bb.changelogs.list",
          ])
      ),
    });
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
        hash: "#changelog",
      })
    );

    render();

    await act(async () => {
      await Promise.resolve();
      await Promise.resolve();
    });

    expect(
      container.querySelector('[data-testid="database-changelog-panel"]')
    ).not.toBeNull();
    expect(
      container.querySelector('[data-testid="database-revision-panel"]')
    ).toBeNull();

    unmount();
  });

  test("shows the SQL editor fallback dialog for default-project denial and opens transfer flow", async () => {
    mocks.usePermissionStore.mockReturnValue({
      currentPermissions: new Set(),
      currentPermissionsInProjectV1: vi.fn(() => new Set()),
    });
    mocks.useProjectDatabaseDetail.mockReturnValue({
      database: {
        name: "instances/inst1/databases/db1",
        project: "projects/default",
        effectiveEnvironment: "environments/prod",
      },
      databaseName: "instances/inst1/databases/db1",
      loading: false,
      ready: true,
      allowAlterSchema: false,
      isDefaultProject: true,
    });

    const { container, render, unmount } = renderIntoContainer(
      createElement(ProjectDatabaseDetailPage, {
        projectId: "default",
        instanceId: "inst1",
        databaseName: "db1",
      })
    );

    render();

    await act(async () => {
      await Promise.resolve();
      await Promise.resolve();
    });

    expect(container.textContent).not.toContain("database.transfer-project");

    const sqlEditorButton = Array.from(
      container.querySelectorAll("button")
    ).find((button) => button.textContent === "sql-editor.self");
    expect(sqlEditorButton).toBeDefined();

    await act(async () => {
      sqlEditorButton?.dispatchEvent(
        new MouseEvent("click", { bubbles: true, cancelable: true })
      );
      await Promise.resolve();
    });

    expect(document.body.textContent).toContain("common.warning");
    expect(document.body.textContent).toContain(
      "common.missing-required-permission"
    );
    const transferButton = Array.from(
      document.body.querySelectorAll("button")
    ).find((button) => button.textContent === "database.transfer-project");
    expect(transferButton).toBeDefined();

    await act(async () => {
      transferButton?.dispatchEvent(
        new MouseEvent("click", { bubbles: true, cancelable: true })
      );
      await Promise.resolve();
    });

    expect(
      container
        .querySelector('[data-testid="transfer-project-drawer"]')
        ?.getAttribute("data-open")
    ).toBe("true");

    unmount();
  });

  test("disables gated action buttons when project permissions are missing", async () => {
    mocks.usePermissionStore.mockReturnValue({
      currentPermissions: new Set(),
      currentPermissionsInProjectV1: vi.fn(() => new Set()),
    });
    mocks.useProjectDatabaseDetail.mockReturnValue({
      database: {
        name: "instances/inst1/databases/db1",
        project: "projects/proj1",
        effectiveEnvironment: "environments/prod",
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
      })
    );

    render();

    await act(async () => {
      await Promise.resolve();
      await Promise.resolve();
    });

    const syncButton = Array.from(container.querySelectorAll("button")).find(
      (button) => button.textContent === "database.sync-database"
    );
    const exportButton = Array.from(container.querySelectorAll("button")).find(
      (button) => button.textContent === "database.export-schema"
    );
    const transferButton = Array.from(
      container.querySelectorAll("button")
    ).find((button) => button.textContent === "database.transfer-project");
    const changeButton = Array.from(container.querySelectorAll("button")).find(
      (button) => button.textContent === "database.change-database"
    );

    expect(syncButton?.hasAttribute("disabled")).toBe(true);
    expect(exportButton?.hasAttribute("disabled")).toBe(true);
    expect(transferButton?.hasAttribute("disabled")).toBe(true);
    expect(changeButton?.hasAttribute("disabled")).toBe(true);

    unmount();
  });

  test("preserves hash and query when transfer project succeeds", async () => {
    const batchUpdateDatabases = vi.fn(async () => undefined);
    const getOrFetchDatabaseByName = vi.fn(async () => ({
      name: "instances/inst1/databases/db1",
      project: "projects/proj2",
    }));
    mocks.useDatabaseV1Store.mockReturnValue({
      syncDatabase: vi.fn(),
      batchUpdateDatabases,
      getOrFetchDatabaseByName,
    });
    mocks.useProjectDatabaseDetail.mockReturnValue({
      database: {
        name: "instances/inst1/databases/db1",
        project: "projects/proj1",
        effectiveEnvironment: "environments/prod",
      },
      databaseName: "instances/inst1/databases/db1",
      loading: false,
      ready: true,
      allowAlterSchema: false,
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

    mocks.routerReplace.mockClear();

    const transferButton = Array.from(
      container.querySelectorAll("button")
    ).find((button) => button.textContent === "database.transfer-project");
    expect(transferButton).toBeDefined();

    await act(async () => {
      transferButton?.dispatchEvent(
        new MouseEvent("click", { bubbles: true, cancelable: true })
      );
      await Promise.resolve();
    });

    const confirmButton = container.querySelector(
      '[data-testid="transfer-project-confirm"]'
    );
    expect(confirmButton).not.toBeNull();

    await act(async () => {
      confirmButton?.dispatchEvent(
        new MouseEvent("click", { bubbles: true, cancelable: true })
      );
      await Promise.resolve();
      await Promise.resolve();
    });

    expect(batchUpdateDatabases).toHaveBeenCalled();
    expect(getOrFetchDatabaseByName).toHaveBeenCalledWith(
      "instances/inst1/databases/db1"
    );
    expect(mocks.routerReplace).toHaveBeenCalledWith({
      name: mocks.routeNames.databaseDetail,
      params: {
        projectId: "proj2",
        instanceId: "inst1",
        databaseName: "db1",
      },
      hash: `#${PROJECT_DATABASE_DETAIL_TAB_REVISION}`,
      query: { foo: "bar", page: "2" },
    });

    unmount();
  });
});

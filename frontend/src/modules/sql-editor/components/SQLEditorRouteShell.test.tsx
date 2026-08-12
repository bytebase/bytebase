import { act } from "react";
import { createRoot } from "react-dom/client";
import { beforeEach, describe, expect, test, vi } from "vitest";
import type { ReactRoute } from "@/app/router";
import { sqlEditorEvents } from "@/modules/sql-editor/model/events";
import type { SQLEditorTab } from "@/types";
import { SQLEditorRouteShell } from "./SQLEditorRouteShell";

(
  globalThis as { IS_REACT_ACT_ENVIRONMENT?: boolean }
).IS_REACT_ACT_ENVIRONMENT = true;

const mocks = vi.hoisted(() => {
  const tabsState = {
    tabsById: new Map(),
    openTmpTabList: [],
    currentTabId: "",
    addTab: vi.fn((payload: Partial<SQLEditorTab> = {}) => {
      const tab = {
        id: "tab-1",
        savedQuery: "",
        connection: {
          instance: "",
          database: "",
        },
        ...payload,
      } as SQLEditorTab;
      tabsState.tabsById.set(tab.id, tab);
      tabsState.currentTabId = tab.id;
      return tab;
    }),
    initProject: vi.fn(async () => undefined),
    updateCurrentTab: vi.fn(),
  };
  const editorState = {
    project: "projects/proj1",
    projectContextReady: true,
    setProjectContextReady: vi.fn(),
    setProject: vi.fn(),
  };
  return {
    tabsState,
    editorState,
    maybeSwitchProject: vi.fn<(project: string) => Promise<string | undefined>>(
      async (project: string) => project
    ),
    setAsidePanelTab: vi.fn(),
    getOrFetchDatabaseByName: vi.fn(async (name: string) => ({
      name,
      project: "projects/proj1",
    })),
    fetchInstance: vi.fn<
      (name: string) => Promise<{ name: string } | undefined>
    >(async (name: string) => ({ name })),
    getSavedQueryByName: vi.fn(),
    getOrFetchSavedQueryByName: vi.fn(),
    searchProjects: vi.fn(),
    beforeEach: vi.fn(() => vi.fn()),
    navigateReplace: vi.fn(),
    renderRoute: {
      name: "sql-editor.database",
      fullPath:
        "/sql-editor/projects/proj1/instances/inst1/databases/db1?schema=public&table=users",
      hash: "",
      params: {
        project: "proj1",
        instance: "inst1",
        database: "db1",
      },
      query: {
        schema: "public",
        table: "users",
      },
      requiredPermissions: [],
      overrideDocumentTitle: false,
      meta: {},
    } as ReactRoute,
    currentRoute: {
      name: "sql-editor.database",
      fullPath:
        "/sql-editor/projects/proj1/instances/inst1/databases/db1?schema=public&table=users",
      hash: "",
      params: {
        project: "proj1",
        instance: "inst1",
        database: "db1",
      },
      query: {
        schema: "public",
        table: "users",
      },
      requiredPermissions: [],
      overrideDocumentTitle: false,
      meta: {},
    } as ReactRoute,
  };
});

vi.mock("react-i18next", () => ({
  initReactI18next: { type: "3rdParty", init: () => undefined },
  useTranslation: () => ({
    t: (key: string) => key,
  }),
}));

vi.mock("@/components/ComponentPermissionGuard", () => ({
  PermissionDeniedFallback: () => <div data-testid="denied" />,
  useComponentPermissionState: () => ({
    missedBasicPermissions: [],
    missedPermissions: [],
    permitted: true,
  }),
  usePermissionDataReady: () => true,
}));

vi.mock("@/hooks/useAppProject", () => ({
  useAppProject: () => ({
    name: "projects/proj1",
  }),
}));

vi.mock("@/modules/sql-editor/hooks/useSQLEditorState", () => ({
  useClampResultRowsLimitToPolicy: vi.fn(),
}));

vi.mock("@/app/router", async (importOriginal) => ({
  ...(await importOriginal<typeof import("@/app/router")>()),
  router: {
    currentRoute: {
      get value() {
        return mocks.currentRoute;
      },
    },
    beforeEach: mocks.beforeEach,
  },
  useCurrentRoute: () => mocks.renderRoute,
  useNavigate: () => ({
    replace: mocks.navigateReplace,
  }),
}));

vi.mock("@/stores/app", () => ({
  useAppStore: {
    getState: () => ({
      getOrFetchDatabaseByName: mocks.getOrFetchDatabaseByName,
      fetchInstance: mocks.fetchInstance,
      getSavedQueryByName: mocks.getSavedQueryByName,
      getOrFetchSavedQueryByName: mocks.getOrFetchSavedQueryByName,
      searchProjects: mocks.searchProjects,
      serverInfo: {
        defaultProject: "projects/proj1",
      },
    }),
  },
}));

vi.mock("@/modules/sql-editor/store", () => ({
  useSQLEditorStore: Object.assign(
    (selector: (state: unknown) => unknown) =>
      selector({
        maybeSwitchProject: mocks.maybeSwitchProject,
        setAsidePanelTab: mocks.setAsidePanelTab,
        asidePanelTab: "SCHEMA",
      }),
    {
      getState: () => ({
        fetchQueryHistory: vi.fn(),
        setLinkedQueryHistory: vi.fn(),
      }),
    }
  ),
}));

vi.mock("@/utils", async (importOriginal) => ({
  ...(await importOriginal<typeof import("@/utils")>()),
  isSavedQueryReadableV1: vi.fn(() => true),
}));

vi.mock("@/modules/sql-editor/store/editor", () => ({
  getSQLEditorEditorState: () => mocks.editorState,
  useSQLEditorEditorState: (selector: (state: unknown) => unknown) =>
    selector(mocks.editorState),
}));

vi.mock("@/modules/sql-editor/store/tab", () => ({
  getSQLEditorTabsState: () => mocks.tabsState,
  useSQLEditorTabState: (selector: (state: unknown) => unknown) =>
    selector(mocks.tabsState),
}));

vi.mock("@/modules/sql-editor/legacy/migration", () => ({
  migrateLegacyCache: vi.fn(async () => undefined),
}));

vi.mock("./SQLEditorHomePage", () => ({
  SQLEditorHomePage: () => <div data-testid="sql-editor-home" />,
}));

const renderShell = () => {
  const container = document.createElement("div");
  document.body.appendChild(container);
  const root = createRoot(container);
  act(() => {
    root.render(<SQLEditorRouteShell />);
  });
  return {
    unmount: () =>
      act(() => {
        root.unmount();
        container.remove();
      }),
  };
};

beforeEach(() => {
  vi.clearAllMocks();
  localStorage.clear();
  mocks.maybeSwitchProject.mockImplementation(
    async (project: string) => project
  );
  mocks.tabsState.addTab.mockImplementation(
    (payload: Partial<SQLEditorTab> = {}) => {
      const tab = {
        id: "tab-1",
        savedQuery: "",
        connection: {
          instance: "",
          database: "",
        },
        ...payload,
      } as SQLEditorTab;
      mocks.tabsState.tabsById.set(tab.id, tab);
      mocks.tabsState.currentTabId = tab.id;
      return tab;
    }
  );
  mocks.getOrFetchDatabaseByName.mockImplementation(async (name: string) => ({
    name,
    project: "projects/proj1",
  }));
  mocks.fetchInstance.mockImplementation(async (name: string) => ({ name }));
  mocks.getSavedQueryByName.mockImplementation((name: string) => ({
    name,
    project: "projects/proj1",
  }));
  mocks.getOrFetchSavedQueryByName.mockImplementation(async (name: string) => ({
    name,
    project: "projects/proj1",
    database: "instances/inst1/databases/db1",
    content: new TextEncoder().encode("select 1"),
    contentSize: BigInt(new TextEncoder().encode("select 1").length),
  }));
  mocks.editorState.project = "projects/proj1";
  mocks.tabsState.tabsById = new Map();
  mocks.tabsState.openTmpTabList = [];
  mocks.tabsState.currentTabId = "";
  mocks.renderRoute = {
    ...mocks.renderRoute,
    name: "sql-editor.database",
    params: {
      project: "proj1",
      instance: "inst1",
      database: "db1",
    },
    query: {
      schema: "public",
      table: "users",
    },
    requiredPermissions: [],
  };
  mocks.currentRoute = {
    ...mocks.currentRoute,
    name: "sql-editor.database",
    params: {
      project: "proj1",
      instance: "inst1",
      database: "db1",
    },
    query: {
      schema: "public",
      table: "users",
    },
    requiredPermissions: [],
  };
});

describe("SQLEditorRouteShell", () => {
  test("seeds database route tabs with schema and table from the URL", async () => {
    const { unmount } = renderShell();

    await act(async () => {
      await Promise.resolve();
      await Promise.resolve();
      await Promise.resolve();
    });

    expect(mocks.tabsState.addTab).toHaveBeenCalledWith({
      connection: {
        instance: "instances/inst1",
        database: "instances/inst1/databases/db1",
        schema: "public",
        table: "users",
      },
      mode: "SAVED_QUERY",
    });
    expect(mocks.navigateReplace).toHaveBeenCalledWith({
      name: "sql-editor.database",
      params: {
        project: "proj1",
        instance: "inst1",
        database: "db1",
      },
      query: {
        table: "users",
        schema: "public",
      },
    });

    unmount();
  });

  test("uses the render route when the imperative router snapshot is still on the parent route", async () => {
    mocks.currentRoute = {
      ...mocks.currentRoute,
      name: "sql-editor",
      params: {},
      query: {},
      requiredPermissions: [],
    };

    const { unmount } = renderShell();

    await act(async () => {
      await Promise.resolve();
      await Promise.resolve();
      await Promise.resolve();
    });

    expect(mocks.tabsState.addTab).toHaveBeenCalledWith({
      connection: {
        instance: "instances/inst1",
        database: "instances/inst1/databases/db1",
        schema: "public",
        table: "users",
      },
      mode: "SAVED_QUERY",
    });
    expect(mocks.navigateReplace).toHaveBeenCalledWith({
      name: "sql-editor.database",
      params: {
        project: "proj1",
        instance: "inst1",
        database: "db1",
      },
      query: {
        table: "users",
        schema: "public",
      },
    });

    unmount();
  });

  test("opens the database route when the editor is already on that project", async () => {
    mocks.editorState.project = "projects/proj1";
    let switchProjectCallCount = 0;
    mocks.maybeSwitchProject.mockImplementation(async (project: string) => {
      switchProjectCallCount++;
      return switchProjectCallCount === 1 ? project : undefined;
    });

    const { unmount } = renderShell();

    await act(async () => {
      await Promise.resolve();
      await Promise.resolve();
      await Promise.resolve();
    });

    expect(mocks.tabsState.addTab).toHaveBeenCalledWith({
      connection: {
        instance: "instances/inst1",
        database: "instances/inst1/databases/db1",
        schema: "public",
        table: "users",
      },
      mode: "SAVED_QUERY",
    });

    unmount();
  });

  test("resolves a project instance database without a parent query", async () => {
    const parent = "projects/proj1/instances/inst1";
    mocks.getOrFetchDatabaseByName.mockImplementation(async (name: string) =>
      name.startsWith("projects/")
        ? { name, project: "projects/proj1" }
        : {
            name: "instances/-1/databases/-1",
            project: "projects/-1",
          }
    );

    const { unmount } = renderShell();

    await act(async () => {
      await Promise.resolve();
      await Promise.resolve();
      await Promise.resolve();
    });

    expect(mocks.getOrFetchDatabaseByName).toHaveBeenNthCalledWith(
      1,
      "instances/inst1/databases/db1"
    );
    expect(mocks.getOrFetchDatabaseByName).toHaveBeenNthCalledWith(
      2,
      `${parent}/databases/db1`
    );
    expect(mocks.tabsState.addTab).toHaveBeenCalledWith({
      connection: {
        instance: parent,
        database: `${parent}/databases/db1`,
        schema: "public",
        table: "users",
      },
      mode: "SAVED_QUERY",
    });
    expect(mocks.navigateReplace).toHaveBeenCalledWith({
      name: "sql-editor.database",
      params: {
        project: "proj1",
        instance: "inst1",
        database: "db1",
      },
      query: {
        table: "users",
        schema: "public",
      },
    });

    unmount();
  });

  test("restores a project instance connection without a parent query", async () => {
    const parent = "projects/proj1/instances/inst1";
    mocks.fetchInstance.mockImplementation(async (name: string) =>
      name.startsWith("projects/") ? { name } : undefined
    );
    mocks.renderRoute = {
      ...mocks.renderRoute,
      name: "sql-editor.instance",
      params: {
        project: "proj1",
        instance: "inst1",
      },
      query: {},
    };
    mocks.currentRoute = {
      ...mocks.currentRoute,
      name: "sql-editor.instance",
      params: {
        project: "proj1",
        instance: "inst1",
      },
      query: {},
    };

    const { unmount } = renderShell();

    await act(async () => {
      await Promise.resolve();
      await Promise.resolve();
      await Promise.resolve();
    });

    expect(mocks.fetchInstance).toHaveBeenNthCalledWith(
      1,
      "instances/inst1"
    );
    expect(mocks.fetchInstance).toHaveBeenNthCalledWith(2, parent);
    expect(mocks.tabsState.addTab).toHaveBeenCalledWith({
      connection: {
        instance: parent,
        database: "",
      },
      mode: "SAVED_QUERY",
    });
    expect(mocks.navigateReplace).toHaveBeenCalledWith({
      name: "sql-editor.instance",
      params: {
        project: "proj1",
        instance: "inst1",
      },
      query: {},
    });

    unmount();
  });

  test("opens the database route when project context enrichment fails after database fetch", async () => {
    mocks.editorState.project = "projects/other";
    let switchProjectCallCount = 0;
    mocks.maybeSwitchProject.mockImplementation(async (project: string) => {
      switchProjectCallCount++;
      return switchProjectCallCount === 1 ? project : undefined;
    });

    const { unmount } = renderShell();

    await act(async () => {
      await Promise.resolve();
      await Promise.resolve();
      await Promise.resolve();
    });

    expect(mocks.editorState.setProject).toHaveBeenCalledWith("projects/proj1");
    expect(mocks.tabsState.addTab).toHaveBeenCalledWith({
      connection: {
        instance: "instances/inst1",
        database: "instances/inst1/databases/db1",
        schema: "public",
        table: "users",
      },
      mode: "SAVED_QUERY",
    });

    unmount();
  });

  test("does not downgrade a database deep link while the tab connection is pending", async () => {
    mocks.tabsState.addTab.mockImplementation(
      (payload: Partial<SQLEditorTab> = {}) =>
        ({
          id: "tab-pending",
          savedQuery: "",
          connection: {
            instance: "",
            database: "",
          },
          ...payload,
        }) as SQLEditorTab
    );

    const { unmount } = renderShell();

    await act(async () => {
      await Promise.resolve();
      await Promise.resolve();
      await Promise.resolve();
    });

    expect(mocks.tabsState.addTab).toHaveBeenCalledWith({
      connection: {
        instance: "instances/inst1",
        database: "instances/inst1/databases/db1",
        schema: "public",
        table: "users",
      },
      mode: "SAVED_QUERY",
    });
    expect(mocks.navigateReplace).not.toHaveBeenCalledWith({
      name: "sql-editor.project",
      params: {
        project: "proj1",
      },
      query: {
        table: "users",
      },
    });

    unmount();
  });

  test("restores the sidebar tab from project-scoped localStorage", async () => {
    localStorage.setItem(
      "bb.sql-editor.sidebar.last-visited-tab.projects/proj1",
      JSON.stringify("SCHEMA")
    );

    const { unmount } = renderShell();

    await act(async () => {
      await sqlEditorEvents.emit("project-context-ready", {
        project: "projects/proj1",
      });
      await new Promise((resolve) => requestAnimationFrame(resolve));
    });

    expect(mocks.setAsidePanelTab).toHaveBeenCalledWith("SCHEMA");

    unmount();
  });

  test("updates a stale database URL when the active tab is disconnected", async () => {
    mocks.tabsState.tabsById.set("tab-blank", {
      id: "tab-blank",
      savedQuery: "",
      connection: {
        instance: "",
        database: "",
      },
    } as SQLEditorTab);
    mocks.tabsState.currentTabId = "tab-blank";
    mocks.tabsState.addTab.mockImplementation(
      (payload: Partial<SQLEditorTab> = {}) => {
        const tab = {
          id: "tab-from-route",
          savedQuery: "",
          connection: {
            instance: "",
            database: "",
          },
          ...payload,
        } as SQLEditorTab;
        mocks.tabsState.tabsById.set(tab.id, tab);
        return tab;
      }
    );

    const { unmount } = renderShell();

    await act(async () => {
      await Promise.resolve();
      await Promise.resolve();
      await Promise.resolve();
    });

    expect(mocks.navigateReplace).toHaveBeenCalledWith({
      name: "sql-editor.project",
      params: {
        project: "proj1",
      },
      query: {},
    });

    unmount();
  });

  test("seeds saved query route tabs with schema and table from the URL", async () => {
    mocks.renderRoute = {
      ...mocks.renderRoute,
      name: "sql-editor.saved-query",
      params: {
        project: "proj1",
        savedQuery: "sheet1",
      },
      query: {
        schema: "public",
        table: "SUPPORDERS_VIS.items",
      },
      requiredPermissions: [],
    };
    mocks.currentRoute = {
      ...mocks.currentRoute,
      name: "sql-editor.saved-query",
      params: {
        project: "proj1",
        savedQuery: "sheet1",
      },
      query: {
        schema: "public",
        table: "SUPPORDERS_VIS.items",
      },
      requiredPermissions: [],
    };

    const { unmount } = renderShell();

    await act(async () => {
      await Promise.resolve();
      await Promise.resolve();
      await Promise.resolve();
    });

    expect(mocks.tabsState.addTab).toHaveBeenCalledWith({
      id: undefined,
      connection: {
        instance: "instances/inst1",
        database: "instances/inst1/databases/db1",
        schema: "public",
        table: "SUPPORDERS_VIS.items",
      },
      savedQuery: "projects/proj1/savedQueries/sheet1",
      title: undefined,
      statement: "select 1",
      status: "CLEAN",
    });

    unmount();
  });

  test("keeps schema and table query parameters when syncing saved query tabs", async () => {
    mocks.renderRoute = {
      ...mocks.renderRoute,
      name: "sql-editor.project",
      params: {
        project: "proj1",
      },
      query: {},
      requiredPermissions: [],
    };
    mocks.currentRoute = {
      ...mocks.currentRoute,
      name: "sql-editor.project",
      params: {
        project: "proj1",
      },
      query: {},
      requiredPermissions: [],
    };
    mocks.tabsState.tabsById.set("tab-savedQuery", {
      id: "tab-savedQuery",
      savedQuery: "projects/proj1/savedQueries/sheet1",
      connection: {
        instance: "instances/inst1",
        database: "instances/inst1/databases/db1",
        schema: "public",
        table: "SUPPORDERS_VIS.items",
      },
    } as SQLEditorTab);
    mocks.tabsState.currentTabId = "tab-savedQuery";

    const { unmount } = renderShell();

    await act(async () => {
      await Promise.resolve();
      await Promise.resolve();
      await Promise.resolve();
    });

    expect(mocks.navigateReplace).toHaveBeenCalledWith({
      name: "sql-editor.saved-query",
      params: {
        project: "proj1",
        savedQuery: "sheet1",
      },
      query: {
        table: "SUPPORDERS_VIS.items",
        schema: "public",
      },
    });

    unmount();
  });
});

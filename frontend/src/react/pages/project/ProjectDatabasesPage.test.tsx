import { act } from "react";
import { createRoot } from "react-dom/client";
import { beforeEach, describe, expect, test, vi } from "vitest";
import { preCreateIssue } from "@/react/lib/plan/issue";

(
  globalThis as { IS_REACT_ACT_ENVIRONMENT?: boolean }
).IS_REACT_ACT_ENVIRONMENT = true;

const mocks = vi.hoisted(() => ({
  visibleDatabases: [] as { name: string }[],
  routerCurrentQuery: {} as Record<string, unknown>,
  routerPush: vi.fn(),
  useProductIntro: vi.fn(),
  removeDatabaseMetadataCache: vi.fn(),
  fetchInstanceList: vi.fn(async () => ({
    instances: [] as { name: string; title: string }[],
  })),
  workspacePermissions: new Set<string>([
    "bb.instances.create",
    "bb.instances.list",
  ]),
}));

let ProjectDatabasesPage: typeof import("./ProjectDatabasesPage").ProjectDatabasesPage;

vi.mock("react-i18next", () => ({
  initReactI18next: { type: "3rdParty", init: () => {} },
  useTranslation: () => ({ t: (key: string) => key }),
}));

vi.mock("@/react/router", () => ({
  router: {
    push: mocks.routerPush,
    currentRoute: {
      get value() {
        return { query: mocks.routerCurrentQuery };
      },
    },
  },
}));

vi.mock("@/react/components/AdvancedSearch", () => ({
  AdvancedSearch: () => <div data-testid="advanced-search" />,
  getValueFromScopes: () => undefined,
}));

vi.mock("@/react/components/EditEnvironmentSheet", () => ({
  EditEnvironmentSheet: () => null,
}));

vi.mock("@/react/components/PermissionGuard", () => ({
  PermissionGuard: ({ children }: { children: React.ReactNode }) => (
    <>{children}</>
  ),
}));

vi.mock("@/react/components/database", async () => {
  const React = await import("react");
  return {
    CreateDatabaseSheet: ({ open }: { open: boolean }) =>
      open
        ? React.createElement("div", { "data-testid": "create-database-sheet" })
        : null,
    DatabaseBatchOperationsBar: () => null,
    DatabaseTable: ({
      emptyPlaceholder,
      onDatabasesChange,
    }: {
      emptyPlaceholder?: React.ReactNode;
      onDatabasesChange: (databases: { name: string }[]) => void;
    }) => {
      React.useEffect(() => {
        onDatabasesChange(mocks.visibleDatabases);
      }, [onDatabasesChange]);
      return React.createElement(
        "div",
        { "data-testid": "database-table" },
        emptyPlaceholder
      );
    },
    LabelEditorSheet: () => null,
  };
});

vi.mock("@/react/hooks/useProjectByName", () => ({
  useProjectByName: (name: string) => ({ name, title: "Demo Project" }),
}));

vi.mock("@/react/lib/plan/issue", () => ({
  preCreateIssue: vi.fn(),
}));

vi.mock("@/react/lib/productIntro", () => ({
  CONNECT_DATABASE_PRODUCT_INTRO: "connect-database",
  PRODUCT_INTRO_QUERY_KEY: "intro",
  useProductIntro: mocks.useProductIntro,
}));

vi.mock("@/react/stores/app", () => {
  const appState = {
    removeDatabaseMetadataCache: mocks.removeDatabaseMetadataCache,
    databasesByName: {},
    projectsByName: {},
    environmentList: [],
  };
  const useAppStore = (selector: (state: typeof appState) => unknown) =>
    selector(appState);
  useAppStore.getState = () => ({
    fetchInstanceList: mocks.fetchInstanceList,
    batchSyncDatabases: vi.fn(),
    batchUpdateDatabases: vi.fn(),
    serverInfo: { defaultProject: "projects/default" },
  });
  return { useAppStore };
});

vi.mock("@/store", () => ({
  pushNotification: vi.fn(),
}));

vi.mock("@/utils", async () => {
  const actual = await vi.importActual<typeof import("@/utils")>("@/utils");
  return {
    ...actual,
    engineNameV1: () => "PostgreSQL",
    extractInstanceResourceName: (name: string) =>
      name.replace(/^instances\//, ""),
    getDefaultPagination: () => 1000,
    hasProjectPermissionV2: () => true,
    hasWorkspacePermissionV2: (permission: string) =>
      mocks.workspacePermissions.has(permission),
    PERMISSIONS_FOR_DATABASE_CREATE_ISSUE: [],
    supportedEngineV1List: () => [],
  };
});

beforeEach(async () => {
  vi.clearAllMocks();
  mocks.visibleDatabases = [];
  mocks.routerCurrentQuery = {};
  mocks.workspacePermissions = new Set([
    "bb.instances.create",
    "bb.instances.list",
  ]);
  ({ ProjectDatabasesPage } = await import("./ProjectDatabasesPage"));
});

describe("ProjectDatabasesPage", () => {
  test("shows a connect database action when the project has no databases", async () => {
    const container = document.createElement("div");
    const root = createRoot(container);

    await act(async () => {
      root.render(<ProjectDatabasesPage projectId="demo" />);
    });

    const toolbarButton = container.querySelector(
      'button[data-product-intro-target="connect-database"]'
    );
    expect(toolbarButton?.textContent).toContain("project.connect-database");
    expect(
      container.querySelector("[data-testid='database-table'] button")
    ).toBe(null);
    expect(container.textContent).toContain(
      "project.connect-database-empty-placeholder"
    );

    act(() => {
      root.unmount();
    });
  });

  test("starts the connect database intro when requested after project creation", async () => {
    mocks.routerCurrentQuery = { intro: "connect-database" };
    const container = document.createElement("div");
    const root = createRoot(container);

    await act(async () => {
      root.render(<ProjectDatabasesPage projectId="demo" />);
      await Promise.resolve();
    });

    expect(mocks.useProductIntro).toHaveBeenCalledWith({
      id: "connect-database",
      title: "project.connect-database-intro-title",
      description: "project.connect-database-intro-description",
      disabled: false,
    });

    act(() => {
      root.unmount();
    });
  });

  test("routes the empty project connect action to instance creation with project context", async () => {
    const container = document.createElement("div");
    const root = createRoot(container);

    await act(async () => {
      root.render(<ProjectDatabasesPage projectId="demo" />);
    });

    const button = container.querySelector(
      'button[data-product-intro-target="connect-database"]'
    ) as HTMLButtonElement;
    act(() => {
      button.click();
    });

    expect(mocks.routerPush).toHaveBeenCalledWith({
      name: "workspace.instance.create",
      query: { project: "demo" },
    });

    act(() => {
      root.unmount();
    });
  });

  test("opens the add database sheet when the project is empty but the workspace has instances", async () => {
    mocks.fetchInstanceList.mockResolvedValueOnce({
      instances: [{ name: "instances/prod", title: "Prod" }],
    });
    const container = document.createElement("div");
    const root = createRoot(container);

    await act(async () => {
      root.render(<ProjectDatabasesPage projectId="demo" />);
      await Promise.resolve();
    });

    const button = container.querySelector(
      "button:not([data-product-intro-target])"
    ) as HTMLButtonElement;
    expect(button.textContent?.trim()).toContain("project.add-database");
    expect(container.textContent).toContain(
      "project.add-database-empty-placeholder"
    );
    expect(container.textContent).not.toContain("project.connect-database");

    await act(async () => {
      button.click();
    });

    expect(mocks.routerPush).not.toHaveBeenCalled();
    expect(
      container.querySelector("[data-testid='create-database-sheet']")
    ).not.toBeNull();

    act(() => {
      root.unmount();
    });
  });

  test("disables the empty project connect action without workspace instance creation permission", async () => {
    mocks.workspacePermissions.delete("bb.instances.create");
    const container = document.createElement("div");
    const root = createRoot(container);

    await act(async () => {
      root.render(<ProjectDatabasesPage projectId="demo" />);
    });

    const button = container.querySelector(
      'button[data-product-intro-target="connect-database"]'
    ) as HTMLButtonElement;
    expect(button.textContent?.trim()).toContain("project.connect-database");
    expect(button.disabled).toBe(true);
    expect(
      container.querySelector("[data-testid='database-table'] button")
    ).toBe(null);

    act(() => {
      button.click();
    });
    expect(mocks.routerPush).not.toHaveBeenCalled();

    act(() => {
      root.unmount();
    });
  });

  test("shows syncing guidance when redirected from project-aware instance creation", async () => {
    mocks.routerCurrentQuery = { syncingInstance: "prod" };
    const container = document.createElement("div");
    const root = createRoot(container);

    await act(async () => {
      root.render(<ProjectDatabasesPage projectId="demo" />);
    });

    expect(container.textContent).toContain(
      "db.project-instance-syncing-title"
    );
    expect(container.textContent).toContain(
      "db.project-instance-syncing-description"
    );
    expect(container.textContent).toContain("common.refresh");
    expect(container.textContent).toContain(
      "db.project-instance-syncing-empty"
    );
    expect(container.textContent).not.toContain("project.connect-database");

    const refreshButton = Array.from(container.querySelectorAll("button")).find(
      (button) => button.textContent?.includes("common.refresh")
    ) as HTMLButtonElement;
    await act(async () => {
      refreshButton.click();
    });

    act(() => {
      root.unmount();
    });
  });

  test("shows next action after redirected instance databases finish syncing", async () => {
    mocks.routerCurrentQuery = { syncingInstance: "prod" };
    mocks.visibleDatabases = [{ name: "instances/prod/databases/app" }];
    const container = document.createElement("div");
    const root = createRoot(container);

    await act(async () => {
      root.render(<ProjectDatabasesPage projectId="demo" />);
    });

    expect(container.textContent).toContain("db.project-instance-synced-title");
    expect(container.textContent).toContain(
      "db.project-instance-synced-description"
    );
    expect(container.textContent).toContain(
      "db.project-instance-synced-action"
    );
    expect(container.textContent).toContain(
      "db.project-instance-synced-sql-editor-action"
    );
    expect(container.textContent).not.toContain(
      "db.project-instance-syncing-title"
    );

    const nextActionButton = Array.from(
      container.querySelectorAll("button")
    ).find((button) =>
      button.textContent?.includes("db.project-instance-synced-action")
    ) as HTMLButtonElement;
    await act(async () => {
      nextActionButton.click();
    });

    expect(preCreateIssue).toHaveBeenCalledWith("projects/demo", [
      "instances/prod/databases/app",
    ]);

    const sqlEditorButton = Array.from(
      container.querySelectorAll("button")
    ).find((button) =>
      button.textContent?.includes(
        "db.project-instance-synced-sql-editor-action"
      )
    ) as HTMLButtonElement;
    await act(async () => {
      sqlEditorButton.click();
    });

    expect(mocks.routerPush).toHaveBeenCalledWith({
      name: "sql-editor.database",
      params: {
        project: "demo",
        instance: "prod",
        database: "app",
      },
    });

    act(() => {
      root.unmount();
    });
  });

  test("recognizes syncInstance query alias for manually opened URLs", async () => {
    mocks.routerCurrentQuery = {
      syncInstance: "prod",
      intro: "connect-database",
    };
    mocks.visibleDatabases = [{ name: "instances/prod/databases/app" }];
    const container = document.createElement("div");
    const root = createRoot(container);

    await act(async () => {
      root.render(<ProjectDatabasesPage projectId="demo" />);
    });

    expect(container.textContent).toContain("db.project-instance-synced-title");
    expect(container.textContent).toContain(
      "db.project-instance-synced-action"
    );

    act(() => {
      root.unmount();
    });
  });
});

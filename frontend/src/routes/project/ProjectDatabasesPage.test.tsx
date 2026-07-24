import { act } from "react";
import { createRoot } from "react-dom/client";
import { beforeEach, describe, expect, test, vi } from "vitest";
import { preCreateIssue } from "@/lib/plan/issue";

(
  globalThis as { IS_REACT_ACT_ENVIRONMENT?: boolean }
).IS_REACT_ACT_ENVIRONMENT = true;

const mocks = vi.hoisted(() => ({
  visibleDatabases: [] as { name: string }[],
  databasesByName: {} as Record<string, { name: string }>,
  instancesByName: {} as Record<string, { name: string; title: string }>,
  routerCurrentName: "workspace.project.database",
  routerCurrentQuery: {} as Record<string, unknown>,
  routerPush: vi.fn(),
  useProductIntro: vi.fn(),
  captureMetric: vi.fn(),
  removeDatabaseMetadataCache: vi.fn(),
  fetchInstance: vi.fn(),
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
  Trans: ({
    i18nKey,
    components,
  }: {
    i18nKey: string;
    components?: { instance?: React.ReactNode };
  }) => (
    <>
      {i18nKey}
      {components?.instance}
    </>
  ),
  useTranslation: () => ({
    t: (key: string, options?: { instance?: string }) =>
      options?.instance ? `${key}:${options.instance}` : key,
  }),
}));

vi.mock("@/app/router", () => ({
  router: {
    push: mocks.routerPush,
    resolve: ({ name }: { name?: string }) => ({ href: `/${name ?? ""}` }),
    currentRoute: {
      get value() {
        return {
          name: mocks.routerCurrentName,
          query: mocks.routerCurrentQuery,
        };
      },
    },
  },
}));

vi.mock("@/app/analytics/provider", () => ({
  behaviorAnalytics: {
    captureMetric: mocks.captureMetric,
  },
}));

vi.mock("@/components/AdvancedSearch", () => ({
  AdvancedSearch: () => <div data-testid="advanced-search" />,
  getValueFromScopes: () => undefined,
}));

vi.mock("@/components/EditEnvironmentSheet", () => ({
  EditEnvironmentSheet: () => null,
}));

vi.mock("@/components/PermissionGuard", () => ({
  PermissionGuard: ({ children }: { children: React.ReactNode }) => (
    <>{children}</>
  ),
}));

vi.mock("@/components/database", async () => {
  const React = await import("react");
  return {
    CreateDatabaseSheet: ({ open }: { open: boolean }) =>
      open
        ? React.createElement("div", { "data-testid": "create-database-sheet" })
        : null,
    DatabaseBatchOperationsBar: ({
      databases,
      onTransferProject,
      onToggleSelectAll,
    }: {
      databases: { name: string }[];
      onTransferProject?: () => void;
      onToggleSelectAll: () => void;
    }) =>
      React.createElement(
        "div",
        {
          "data-testid": "batch-operations-bar",
          "data-selected-count": String(databases.length),
          "data-has-transfer": String(!!onTransferProject),
        },
        React.createElement(
          "button",
          {
            "data-testid": "batch-select-all",
            onClick: onToggleSelectAll,
          },
          "select all"
        ),
        databases.length > 0 && onTransferProject
          ? React.createElement(
              "button",
              {
                "data-testid": "batch-transfer-project",
                onClick: onTransferProject,
              },
              "database.transfer-project"
            )
          : null
      ),
    DatabaseTable: ({
      emptyPlaceholder,
      onSelectedNamesChange,
      onDatabasesChange,
    }: {
      emptyPlaceholder?: React.ReactNode;
      onSelectedNamesChange: (selectedNames: Set<string>) => void;
      onDatabasesChange: (databases: { name: string }[]) => void;
    }) => {
      React.useEffect(() => {
        onDatabasesChange(mocks.visibleDatabases);
        onSelectedNamesChange(
          new Set(mocks.visibleDatabases.map((d) => d.name))
        );
      }, [onDatabasesChange, onSelectedNamesChange]);
      return React.createElement(
        "div",
        { "data-testid": "database-table" },
        emptyPlaceholder
      );
    },
    LabelEditorSheet: () => null,
    TransferProjectSheet: ({ open }: { open: boolean }) =>
      open
        ? React.createElement("div", { "data-testid": "transfer-project-sheet" })
        : null,
  };
});

vi.mock("@/hooks/useProjectByName", () => ({
  useProjectByName: (name: string) => ({ name, title: "Demo Project" }),
}));

vi.mock("@/lib/plan/issue", () => ({
  preCreateIssue: vi.fn(),
}));

vi.mock("@/lib/productIntro", () => ({
  CONNECT_DATABASE_PRODUCT_INTRO: "connect-database",
  PROJECT_INSTANCE_SYNCED_PRODUCT_INTRO: "project-instance-synced",
  PRODUCT_INTRO_QUERY_KEY: "intro",
  useProductIntro: mocks.useProductIntro,
}));

vi.mock("@/types", async () => {
  const actual = await vi.importActual<typeof import("@/types")>("@/types");
  return {
    ...actual,
    isDefaultProject: (name: string) => name === "projects/default",
  };
});

vi.mock("@/stores/app", () => {
  const appState = {
    removeDatabaseMetadataCache: mocks.removeDatabaseMetadataCache,
    get databasesByName() {
      return mocks.databasesByName;
    },
    get instancesByName() {
      return mocks.instancesByName;
    },
    projectsByName: {},
    environmentList: [],
  };
  const useAppStore = (selector: (state: typeof appState) => unknown) =>
    selector(appState);
  useAppStore.getState = () => ({
    fetchInstance: mocks.fetchInstance,
    fetchInstanceList: mocks.fetchInstanceList,
    batchSyncDatabases: vi.fn(),
    batchUpdateDatabases: vi.fn(),
    serverInfo: { defaultProject: "projects/default" },
  });
  return { useAppStore };
});

vi.mock("@/stores", () => ({
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
  mocks.databasesByName = {};
  mocks.instancesByName = {};
  mocks.routerCurrentName = "workspace.project.database";
  mocks.routerCurrentQuery = {};
  mocks.workspacePermissions = new Set([
    "bb.instances.create",
    "bb.instances.get",
    "bb.instances.list",
  ]);
  ({ ProjectDatabasesPage } = await import("./ProjectDatabasesPage"));
});

describe("ProjectDatabasesPage", () => {
  test("shows a connect instance action when the project has no databases and no workspace instances", async () => {
    const container = document.createElement("div");
    const root = createRoot(container);

    await act(async () => {
      root.render(<ProjectDatabasesPage projectId="demo" />);
    });

    const toolbarButton = container.querySelector(
      'button[data-product-intro-target="connect-database"]'
    );
    expect(toolbarButton?.textContent).toContain("project.connect-instance");
    expect(
      container.querySelector("[data-testid='database-table'] button")
    ).toBe(null);
    expect(container.textContent).toContain(
      "project.connect-instance-empty-placeholder"
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
      title: "project.connect-instance-intro-title",
      description: "project.connect-instance-intro-description",
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
    expect(mocks.captureMetric).toHaveBeenCalledWith({
      event: "connect database clicked",
      properties: {
        route_id: "workspace.project.database",
        resource: "projects/demo",
      },
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
    expect(button.textContent?.trim()).toContain("project.connect-instance");
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
    mocks.instancesByName = {
      "instances/prod": { name: "instances/prod", title: "Prod Instance" },
    };
    const container = document.createElement("div");
    const root = createRoot(container);

    await act(async () => {
      root.render(<ProjectDatabasesPage projectId="demo" />);
    });

    expect(container.textContent).toContain(
      "db.project-instance-syncing-titleProd Instance"
    );
    expect(container.textContent).not.toContain(
      "db.project-instance-syncing-titleprod"
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

  test("falls back to create database after redirected instance sync finds no databases", async () => {
    vi.useFakeTimers();
    mocks.routerCurrentQuery = { syncingInstance: "prod" };
    mocks.fetchInstanceList.mockResolvedValue({
      instances: [{ name: "instances/prod", title: "Prod" }],
    });
    const container = document.createElement("div");
    const root = createRoot(container);

    await act(async () => {
      root.render(<ProjectDatabasesPage projectId="demo" />);
      await Promise.resolve();
    });

    expect(container.textContent).toContain(
      "db.project-instance-syncing-title"
    );

    await act(async () => {
      vi.advanceTimersByTime(60000);
      await Promise.resolve();
    });

    expect(container.textContent).not.toContain(
      "db.project-instance-syncing-title"
    );
    const button = container.querySelector("button") as HTMLButtonElement;
    expect(button.textContent?.trim()).toContain("project.add-database");

    await act(async () => {
      button.click();
    });

    expect(
      container.querySelector("[data-testid='create-database-sheet']")
    ).not.toBeNull();

    act(() => {
      root.unmount();
    });
    vi.useRealTimers();
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
    expect(
      container
        .querySelector("[role='alert']")
        ?.getAttribute("data-product-intro-target")
    ).toBe("project-instance-synced");
    expect(mocks.useProductIntro).toHaveBeenCalledWith({
      id: "project-instance-synced",
      title: "db.project-instance-synced-title",
      description: "db.project-instance-synced-description",
      disabled: false,
    });

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
    expect(sqlEditorButton.className).toContain("bg-accent");
    expect(nextActionButton.className).toContain("border-control-border");
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

  test("shows transfer project batch action for the default project", async () => {
    mocks.visibleDatabases = [{ name: "instances/prod/databases/app" }];
    mocks.databasesByName = {
      "instances/prod/databases/app": { name: "instances/prod/databases/app" },
    };
    const container = document.createElement("div");
    const root = createRoot(container);

    await act(async () => {
      root.render(<ProjectDatabasesPage projectId="default" />);
      await Promise.resolve();
    });
    await act(async () => {
      await Promise.resolve();
    });

    const selectAllButton = container.querySelector(
      '[data-testid="batch-select-all"]'
    ) as HTMLButtonElement;
    await act(async () => {
      selectAllButton.click();
    });

    const batchBar = container.querySelector(
      '[data-testid="batch-operations-bar"]'
    ) as HTMLElement;
    expect(batchBar.dataset.hasTransfer).toBe("true");
    expect(batchBar.dataset.selectedCount).toBe("1");

    const transferButton = container.querySelector(
      '[data-testid="batch-transfer-project"]'
    ) as HTMLButtonElement;
    expect(transferButton?.textContent).toContain("database.transfer-project");

    await act(async () => {
      transferButton.click();
    });

    expect(
      container.querySelector('[data-testid="transfer-project-sheet"]')
    ).not.toBeNull();

    act(() => {
      root.unmount();
    });
  });

});

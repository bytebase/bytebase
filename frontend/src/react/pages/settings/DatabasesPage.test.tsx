import { act, useEffect } from "react";
import { createRoot } from "react-dom/client";
import { MemoryRouter } from "react-router";
import { beforeEach, describe, expect, test, vi } from "vitest";

(
  globalThis as { IS_REACT_ACT_ENVIRONMENT?: boolean }
).IS_REACT_ACT_ENVIRONMENT = true;

const mocks = vi.hoisted(() => ({
  routerPush: vi.fn(),
  routerReplace: vi.fn(),
  currentRoute: { query: {} as Record<string, string> },
  batchSyncDatabases: vi.fn(),
  batchUpdateDatabases: vi.fn(),
  removeDatabaseMetadataCache: vi.fn(),
  useProductIntro: vi.fn(),
  hasWorkspacePermission: true,
  databaseTableRows: [] as unknown[],
  databaseTableLoading: false,
}));

let DatabasesPage: typeof import("./DatabasesPage").DatabasesPage;

vi.mock("react-i18next", () => ({
  initReactI18next: { type: "3rdParty", init: () => {} },
  useTranslation: () => ({ t: (key: string) => key }),
}));

vi.mock("@/react/router", () => ({
  router: {
    currentRoute: {
      get value() {
        return mocks.currentRoute;
      },
    },
    push: mocks.routerPush,
    replace: mocks.routerReplace,
  },
  useCurrentRoute: () => mocks.currentRoute,
}));

vi.mock("@/react/components/AdvancedSearch", () => ({
  AdvancedSearch: () => <div data-testid="advanced-search" />,
  getValueFromScopes: () => undefined,
}));

vi.mock("@/react/components/EditEnvironmentSheet", () => ({
  EditEnvironmentSheet: () => null,
}));

vi.mock("@/react/components/EngineIcon", () => ({
  EngineIcon: () => <span data-testid="engine-icon" />,
}));

vi.mock("@/react/components/EnvironmentLabel", () => ({
  EnvironmentLabel: () => <span data-testid="environment-label" />,
}));

vi.mock("@/react/components/PermissionGuard", () => ({
  PermissionGuard: ({ children }: { children: React.ReactNode }) => (
    <>{children}</>
  ),
}));

vi.mock("@/react/components/WorkspacePageLayout", () => ({
  WorkspacePageLayout: ({ children }: { children: React.ReactNode }) => (
    <div>{children}</div>
  ),
  WorkspacePageToolbar: ({ children }: { children: React.ReactNode }) => (
    <div>{children}</div>
  ),
}));

vi.mock("@/react/lib/productIntro", () => ({
  PREPARE_DATABASE_PRODUCT_INTRO: "prepare-database",
  PREPARE_DATABASE_TRANSFER_TIP: "transfer-databases-to-project",
  PRODUCT_INTRO_TIP_QUERY_KEY: "tip",
  useProductIntro: mocks.useProductIntro,
}));

vi.mock("@/react/components/database", () => ({
  CreateDatabaseSheet: ({ open }: { open: boolean }) => (
    <div data-testid="create-database-sheet" data-open={String(open)} />
  ),
  DatabaseBatchOperationsBar: () => null,
  DatabaseTable: ({
    emptyPlaceholder,
    selectionColumnIntroTarget,
    onDatabasesChange,
    onLoadingChange,
  }: {
    emptyPlaceholder?: React.ReactNode;
    selectionColumnIntroTarget?: string;
    onDatabasesChange?: (databases: unknown[]) => void;
    onLoadingChange?: (loading: boolean) => void;
  }) => (
    <DatabaseTableMock
      emptyPlaceholder={emptyPlaceholder}
      selectionColumnIntroTarget={selectionColumnIntroTarget}
      onDatabasesChange={onDatabasesChange}
      onLoadingChange={onLoadingChange}
    />
  ),
  LabelEditorSheet: () => null,
  TransferProjectSheet: () => null,
}));

const DatabaseTableMock = ({
  emptyPlaceholder,
  selectionColumnIntroTarget,
  onDatabasesChange,
  onLoadingChange,
}: {
  emptyPlaceholder?: React.ReactNode;
  selectionColumnIntroTarget?: string;
  onDatabasesChange?: (databases: unknown[]) => void;
  onLoadingChange?: (loading: boolean) => void;
}) => {
  useEffect(() => {
    onDatabasesChange?.(mocks.databaseTableRows);
  }, [onDatabasesChange]);
  useEffect(() => {
    onLoadingChange?.(mocks.databaseTableLoading);
  }, [onLoadingChange]);
  return (
    <div
      data-testid="database-table"
      data-has-empty-placeholder={String(!!emptyPlaceholder)}
      data-selection-column-intro-target={selectionColumnIntroTarget ?? ""}
    >
      {emptyPlaceholder}
    </div>
  );
};

vi.mock("@/react/stores/app", () => {
  const appState = {
    databasesByName: {},
    environmentList: [],
    getDatabaseByName: (name: string) => ({ name }),
    removeDatabaseMetadataCache: mocks.removeDatabaseMetadataCache,
    serverInfo: { defaultProject: "projects/default" },
  };
  const useAppStore = (selector: (state: typeof appState) => unknown) =>
    selector(appState);
  useAppStore.getState = () => ({
    ...appState,
    batchSyncDatabases: mocks.batchSyncDatabases,
    batchUpdateDatabases: mocks.batchUpdateDatabases,
    fetchInstanceList: vi.fn(async () => ({ instances: [] })),
    fetchProjectList: vi.fn(async () => ({ projects: [] })),
    getIntroStateByKey: vi.fn(() => true),
    saveIntroStateByKey: vi.fn(),
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
    extractProjectResourceName: (name: string) =>
      name.replace(/^projects\//, ""),
    getDefaultPagination: () => 1000,
    hasWorkspacePermissionV2: () => mocks.hasWorkspacePermission,
    supportedEngineV1List: () => [],
  };
});

beforeEach(async () => {
  vi.clearAllMocks();
  mocks.hasWorkspacePermission = true;
  mocks.currentRoute = { query: {} };
  mocks.databaseTableRows = [];
  mocks.databaseTableLoading = false;
  ({ DatabasesPage } = await import("./DatabasesPage"));
});

describe("DatabasesPage", () => {
  test("does not pass an action as the empty table placeholder", async () => {
    const container = document.createElement("div");
    const root = createRoot(container);

    await act(async () => {
      root.render(
        <MemoryRouter>
          <DatabasesPage />
        </MemoryRouter>
      );
      await Promise.resolve();
    });

    expect(
      container
        .querySelector("[data-testid='database-table']")
        ?.getAttribute("data-has-empty-placeholder")
    ).toBe("false");
    expect(
      container.querySelector("[data-testid='database-table'] button")
    ).toBeNull();
    expect(
      container
        .querySelector("[data-testid='create-database-sheet']")
        ?.getAttribute("data-open")
    ).toBe("false");
    expect(mocks.useProductIntro).toHaveBeenCalledWith({
      id: "prepare-database",
      title: "workspace-setup-guide.intro.database-title",
      description: "workspace-setup-guide.intro.database-description",
    });

    act(() => {
      root.unmount();
    });
  });

  test("opens the create database sheet from the toolbar action", async () => {
    const container = document.createElement("div");
    const root = createRoot(container);

    await act(async () => {
      root.render(
        <MemoryRouter>
          <DatabasesPage />
        </MemoryRouter>
      );
      await Promise.resolve();
    });

    const createButton = Array.from(container.querySelectorAll("button")).find(
      (button) => button.textContent?.includes("database.create-database")
    ) as HTMLButtonElement;
    expect(createButton).toBeTruthy();
    expect(createButton.getAttribute("data-product-intro-target")).toBe(
      "prepare-database"
    );

    await act(async () => {
      createButton.click();
      await Promise.resolve();
    });

    expect(mocks.routerPush).not.toHaveBeenCalled();
    expect(
      container
        .querySelector("[data-testid='create-database-sheet']")
        ?.getAttribute("data-open")
    ).toBe("true");

    act(() => {
      root.unmount();
    });
  });

  test("shows a transfer tip from the prepare database query", async () => {
    mocks.currentRoute = {
      query: {
        intro: "prepare-database",
        tip: "transfer-databases-to-project",
      },
    };
    mocks.databaseTableRows = [{ name: "instances/i/databases/db" }];
    const container = document.createElement("div");
    const root = createRoot(container);

    await act(async () => {
      root.render(
        <MemoryRouter>
          <DatabasesPage />
        </MemoryRouter>
      );
      await Promise.resolve();
    });

    expect(container.textContent).toContain(
      "workspace-setup-guide.prepare-database-tip"
    );
    const alert = container.querySelector("[role='alert']");
    expect(alert?.className).toContain("w-auto");
    expect(alert?.hasAttribute("data-product-intro-target")).toBe(false);
    expect(
      container
        .querySelector("[data-testid='database-table']")
        ?.getAttribute("data-selection-column-intro-target")
    ).toBe("prepare-database");
    expect(mocks.useProductIntro).toHaveBeenCalledWith({
      id: "prepare-database",
      title: "workspace-setup-guide.intro.transfer-title",
      description: "workspace-setup-guide.intro.transfer-description",
    });
    expect(
      Array.from(container.querySelectorAll("button"))
        .find((button) =>
          button.textContent?.includes("database.create-database")
        )
        ?.hasAttribute("data-product-intro-target")
    ).toBe(false);

    act(() => {
      root.unmount();
    });
  });

  test("waits for the table to load before highlighting the transfer selection column", async () => {
    mocks.currentRoute = {
      query: {
        intro: "prepare-database",
        tip: "transfer-databases-to-project",
      },
    };
    mocks.databaseTableLoading = true;
    mocks.databaseTableRows = [{ name: "instances/i/databases/db" }];
    const container = document.createElement("div");
    const root = createRoot(container);

    await act(async () => {
      root.render(
        <MemoryRouter>
          <DatabasesPage />
        </MemoryRouter>
      );
      await Promise.resolve();
    });

    expect(mocks.useProductIntro).toHaveBeenCalledWith(
      expect.objectContaining({
        id: "prepare-database",
        disabled: true,
      })
    );
    expect(
      container
        .querySelector("[data-testid='database-table']")
        ?.getAttribute("data-selection-column-intro-target")
    ).toBe("");
    expect(
      Array.from(container.querySelectorAll("button"))
        .find((button) =>
          button.textContent?.includes("database.create-database")
        )
        ?.hasAttribute("data-product-intro-target")
    ).toBe(false);

    act(() => {
      root.unmount();
    });
  });

  test("falls back to highlighting create database after transfer context loads with no rows", async () => {
    mocks.currentRoute = {
      query: {
        intro: "prepare-database",
        tip: "transfer-databases-to-project",
      },
    };
    mocks.databaseTableRows = [];
    const container = document.createElement("div");
    const root = createRoot(container);

    await act(async () => {
      root.render(
        <MemoryRouter>
          <DatabasesPage />
        </MemoryRouter>
      );
      await Promise.resolve();
    });

    expect(container.textContent).not.toContain(
      "workspace-setup-guide.prepare-database-tip"
    );
    expect(
      container
        .querySelector("[data-testid='database-table']")
        ?.getAttribute("data-selection-column-intro-target")
    ).toBe("");
    expect(
      Array.from(container.querySelectorAll("button"))
        .find((button) =>
          button.textContent?.includes("database.create-database")
        )
        ?.getAttribute("data-product-intro-target")
    ).toBe("prepare-database");

    act(() => {
      root.unmount();
    });
  });
});

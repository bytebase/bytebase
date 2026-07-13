import { act } from "react";
import { createRoot } from "react-dom/client";
import { beforeEach, describe, expect, test, vi } from "vitest";

(
  globalThis as { IS_REACT_ACT_ENVIRONMENT?: boolean }
).IS_REACT_ACT_ENVIRONMENT = true;

const mocks = vi.hoisted(() => ({
  routerPush: vi.fn(),
  routerReplace: vi.fn(),
  batchSyncDatabases: vi.fn(),
  batchUpdateDatabases: vi.fn(),
  removeDatabaseMetadataCache: vi.fn(),
  hasWorkspacePermission: true,
}));

let DatabasesPage: typeof import("./DatabasesPage").DatabasesPage;

vi.mock("react-i18next", () => ({
  initReactI18next: { type: "3rdParty", init: () => {} },
  useTranslation: () => ({ t: (key: string) => key }),
}));

vi.mock("@/react/router", () => ({
  router: {
    currentRoute: {
      value: { query: {} },
    },
    push: mocks.routerPush,
    replace: mocks.routerReplace,
  },
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

vi.mock("@/react/components/database", () => ({
  CreateDatabaseSheet: ({ open }: { open: boolean }) => (
    <div data-testid="create-database-sheet" data-open={String(open)} />
  ),
  DatabaseBatchOperationsBar: () => null,
  DatabaseTable: ({
    emptyPlaceholder,
  }: {
    emptyPlaceholder?: React.ReactNode;
  }) => <div data-testid="database-table">{emptyPlaceholder}</div>,
  LabelEditorSheet: () => null,
  TransferProjectSheet: () => null,
}));

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
  ({ DatabasesPage } = await import("./DatabasesPage"));
});

describe("DatabasesPage", () => {
  test("passes a connect databases action as the empty table placeholder", async () => {
    const container = document.createElement("div");
    const root = createRoot(container);

    await act(async () => {
      root.render(<DatabasesPage />);
      await Promise.resolve();
    });

    const emptyStateButton = container.querySelector(
      "[data-testid='database-table'] button"
    ) as HTMLButtonElement;
    expect(emptyStateButton?.textContent).toContain(
      "database.connect-databases"
    );
    expect(
      container
        .querySelector("[data-testid='create-database-sheet']")
        ?.getAttribute("data-open")
    ).toBe("false");

    await act(async () => {
      emptyStateButton.click();
      await Promise.resolve();
    });

    expect(mocks.routerPush).toHaveBeenCalledWith({
      name: "workspace.instance.create",
    });
    expect(
      container
        .querySelector("[data-testid='create-database-sheet']")
        ?.getAttribute("data-open")
    ).toBe("false");

    act(() => {
      root.unmount();
    });
  });
});

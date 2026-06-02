import { act, type ReactElement } from "react";
import { createRoot } from "react-dom/client";
import { beforeEach, describe, expect, test, vi } from "vitest";

(
  globalThis as { IS_REACT_ACT_ENVIRONMENT?: boolean }
).IS_REACT_ACT_ENVIRONMENT = true;

globalThis.ResizeObserver = class ResizeObserver {
  observe() {}
  unobserve() {}
  disconnect() {}
};

const mocks = vi.hoisted(() => {
  const tabStore = {
    supportBatchMode: true,
    isInBatchMode: false,
    currentTab: {
      id: "t-1",
      mode: "WORKSHEET" as const,
      title: "Untitled",
      connection: { database: "", instance: "" },
      batchQueryContext: {
        databases: [] as string[],
        databaseGroups: [] as string[],
      },
    } as {
      id: string;
      mode: string;
      title: string;
      connection: { database: string; instance: string };
      batchQueryContext: { databases: string[]; databaseGroups: string[] };
    },
    updateBatchQueryContext: vi.fn(),
    setCurrentTabId: vi.fn(),
    updateTab: vi.fn(),
  };
  const editorStore = {
    project: "projects/p",
    projectContextReady: true,
    allowAdmin: false,
  };
  // Plan-feature flags consumed via `useSQLEditorFeature`. Default ON for
  // render-path tests; individual tests flip to false to exercise gating.
  const features = { batchQuery: true, databaseGroups: true };
  const uiStore = { showConnectionPanel: true, asidePanelTab: "SCHEMA" };
  const databaseStore = {
    batchGetOrFetchDatabases: vi.fn().mockResolvedValue([]),
    getDatabaseByName: vi.fn(() => undefined),
    getOrFetchDatabaseByName: vi.fn().mockResolvedValue(undefined),
  };
  const project = {
    name: "projects/p",
    title: "Project One",
  };
  const treeStore = {
    state: "READY" as "LOADING" | "READY" | "UNSET",
    nodeKeysByTarget: vi.fn(() => []),
  };
  const appStore = {
    fetchInstance: vi.fn(async () => undefined),
    fetchDBGroup: vi.fn(async () => undefined),
    listDBGroupsForProject: vi.fn(async () => []),
    dbGroupsByName: {} as Record<string, unknown>,
    environmentList: [
      {
        name: "environments/prod",
        title: "Prod",
        color: "",
        tags: {},
      },
    ],
    getEnvironmentByName: vi.fn((name: string) => ({
      name,
      title: name.split("/").pop() ?? name,
      color: "",
      tags: {},
    })),
    getDatabaseByName: vi.fn((name: string) => ({
      name,
      instanceResource: { engine: 0, title: "", name: "" },
    })),
    getOrFetchDatabaseByName: vi.fn(async (name: string) => ({ name })),
    batchGetOrFetchDatabases: vi.fn(async () => []),
    fetchDatabases: vi
      .fn()
      .mockResolvedValue({ databases: [], nextPageToken: "" }),
    databasesByName: {} as Record<string, unknown>,
  };
  const currentUser = { email: "u@b.com" };
  return {
    tabStore,
    editorStore,
    features,
    uiStore,
    databaseStore,
    project,
    treeStore,
    appStore,
    currentUser,
    usePiniaBridge: vi.fn<(getter: () => unknown) => unknown>(),
    pushNotification: vi.fn(),
  };
});

vi.mock("react-i18next", () => ({
  useTranslation: () => ({ t: (key: string) => key }),
}));

vi.mock("@/react/hooks/usePiniaBridge", () => ({
  usePiniaBridge: mocks.usePiniaBridge,
}));

vi.mock("@/react/hooks/useSQLEditorBridge", () => ({
  // FEATURE_BATCH_QUERY === 1, FEATURE_DATABASE_GROUPS === 2 (see the
  // subscription_service_pb mock below).
  useSQLEditorFeature: (feature: number) =>
    feature === 1 ? mocks.features.batchQuery : mocks.features.databaseGroups,
}));

vi.mock("@/react/hooks/useAppState", () => ({
  useCurrentUser: () => mocks.currentUser,
}));

vi.mock("@/store", () => ({
  pushNotification: mocks.pushNotification,
}));

vi.mock("@/react/hooks/useAppProject", () => ({
  useAppProject: () => mocks.project,
}));

vi.mock("@/react/hooks/useAppDatabase", () => ({
  useAppDatabase: (name: string) => ({
    name,
    instanceResource: { engine: 0, title: "", name: "" },
  }),
}));

vi.mock("@/react/stores/app", () => ({
  useAppStore: Object.assign(
    (selector: (state: unknown) => unknown) => selector(mocks.appStore),
    { getState: () => mocks.appStore, subscribe: () => () => {} }
  ),
}));

vi.mock("@/react/stores/sqlEditor/tab", () => ({
  useSupportBatchMode: () => mocks.tabStore.supportBatchMode,
  useIsInBatchMode: () => mocks.tabStore.isInBatchMode,
  useCurrentSQLEditorTab: () => mocks.tabStore.currentTab,
  getSQLEditorTabsState: () => ({
    updateBatchQueryContext: mocks.tabStore.updateBatchQueryContext,
    setCurrentTabId: mocks.tabStore.setCurrentTabId,
    updateTab: mocks.tabStore.updateTab,
    currentTabId: mocks.tabStore.currentTab?.id ?? "",
    tabsById: new Map(
      mocks.tabStore.currentTab
        ? [[mocks.tabStore.currentTab.id, mocks.tabStore.currentTab]]
        : []
    ),
  }),
}));

vi.mock("@/react/stores/sqlEditor/editor", () => ({
  useSQLEditorEditorState: (
    selector: (s: { project: string; projectContextReady: boolean }) => unknown
  ) =>
    selector({
      project: mocks.editorStore.project,
      projectContextReady: mocks.editorStore.projectContextReady,
    }),
}));

vi.mock("@/react/stores/sqlEditor", () => ({
  useSQLEditorStore: (
    selector: (s: {
      setShowConnectionPanel: (v: boolean) => void;
      treeState: string;
      setTreeState: (state: string) => void;
      treeNodeKeysByTarget: () => string[];
    }) => unknown
  ) =>
    selector({
      setShowConnectionPanel: (next: boolean) => {
        mocks.uiStore.showConnectionPanel = next;
      },
      treeState: mocks.treeStore.state,
      setTreeState: (state: string) => {
        mocks.treeStore.state = state as typeof mocks.treeStore.state;
      },
      treeNodeKeysByTarget: mocks.treeStore.nodeKeysByTarget,
    }),
}));

vi.mock("@/react/lib/resourceName", () => ({
  instanceNamePrefix: "instances/",
}));

vi.mock("@/views/sql-editor/events", () => ({
  sqlEditorEvents: {
    emit: vi.fn(),
    on: vi.fn(() => () => {}),
  },
}));

vi.mock("@/types", () => ({
  isValidDatabaseGroupName: () => true,
  isValidDatabaseName: (name: string) => !!name,
  isValidProjectName: () => true,
  UNKNOWN_ENVIRONMENT_NAME: "environments/-1",
  NULL_ENVIRONMENT_NAME: "environments/-",
  unknownEnvironment: () => ({
    name: "environments/-1",
    title: "Unknown",
    color: "",
    tags: {},
  }),
}));

vi.mock("@/types/proto-es/v1/instance_service_pb", () => ({
  DataSourceType: { ADMIN: 1, READ_ONLY: 2 },
}));

vi.mock("@/types/proto-es/v1/subscription_service_pb", () => ({
  PlanFeature: {
    FEATURE_BATCH_QUERY: 1,
    FEATURE_DATABASE_GROUPS: 2,
    FEATURE_ENVIRONMENT_TIERS: 3,
  },
  PlanType: { FREE: 0, TEAM: 1, ENTERPRISE: 3 },
}));

vi.mock("@/types/proto-es/v1/database_group_service_pb", () => ({
  DatabaseGroupView: { FULL: 2 },
}));

vi.mock("@/utils", () => ({
  extractDatabaseResourceName: (n: string) => ({
    instance: "instances/x",
    databaseName: n.split("/").pop() ?? "",
  }),
  getConnectionForSQLEditorTab: () => ({ database: undefined }),
  getInstanceResource: () => ({ engine: "MYSQL" }),
  getValueFromSearchParams: () => "",
  getValuesFromSearchParams: () => [],
  instanceV1Name: (i: { title: string }) => i.title,
}));

vi.mock("@/components/InstanceForm/constants", () => ({
  EngineIconPath: { MYSQL: "/mysql.svg" },
}));

vi.mock("@/react/components/FeatureBadge", () => ({
  FeatureBadge: () => null,
}));

vi.mock("@/react/components/ui/feature-modal", () => ({
  FeatureModal: ({ open, feature }: { open: boolean; feature: unknown }) =>
    open && feature ? <div data-testid="feature-modal" /> : null,
}));

vi.mock("@/react/components/ui/radio-group", () => ({
  RadioGroup: ({ children }: { children: React.ReactNode }) => (
    <div data-testid="radio-group">{children}</div>
  ),
  RadioGroupItem: ({ children }: { children?: React.ReactNode }) => (
    <label>{children}</label>
  ),
}));

vi.mock("@/react/components/ui/separator", () => ({
  Separator: () => <hr data-testid="separator" />,
}));

vi.mock("@/react/components/AdvancedSearch", () => ({
  AdvancedSearch: () => <div data-testid="advanced-search" />,
  emptySearchParams: () => ({ query: "", scopes: [] }),
}));

vi.mock("@/react/components/useCommonSearchScopeOptions", () => ({
  useCommonSearchScopeOptions: () => [],
}));

vi.mock("@/react/components/ui/tabs", () => ({
  Tabs: ({
    value,
    onValueChange,
    children,
  }: {
    value: string;
    onValueChange: (v: string) => void;
    children: React.ReactNode;
  }) => (
    <div
      data-testid="tabs"
      data-value={value}
      onClick={(e) => {
        const t = (e.target as HTMLElement).dataset.tabValue;
        if (t) onValueChange(t);
      }}
    >
      {children}
    </div>
  ),
  TabsList: ({ children }: { children: React.ReactNode }) => (
    <div>{children}</div>
  ),
  TabsTrigger: ({
    value,
    children,
    disabled,
  }: {
    value: string;
    children: React.ReactNode;
    disabled?: boolean;
  }) => (
    <button
      type="button"
      data-testid={`tab-trigger-${value}`}
      data-tab-value={value}
      disabled={disabled}
    >
      {children}
    </button>
  ),
  TabsPanel: ({
    value,
    children,
  }: {
    value: string;
    children: React.ReactNode;
  }) => <div data-testid={`tab-panel-${value}`}>{children}</div>,
}));

vi.mock("@/react/components/ui/tooltip", () => ({
  Tooltip: ({ children }: { children: React.ReactNode }) => <>{children}</>,
}));

vi.mock("@/react/components/ui/tree", () => ({
  Tree: ({ data }: { data: unknown[] }) => (
    <div data-testid="tree" data-len={data.length} />
  ),
}));

vi.mock("./actions", () => ({
  setConnection: vi.fn(),
}));

vi.mock("./ConnectionContextMenu", () => ({
  ConnectionContextMenu: () => <div data-testid="context-menu" />,
}));

vi.mock("@/react/components/DatabaseGroupTable", () => ({
  DatabaseGroupTable: () => <div data-testid="db-group-table" />,
}));

vi.mock("./DatabaseGroupTag", () => ({
  DatabaseGroupTag: () => <div data-testid="db-group-tag" />,
}));

vi.mock("./DatabaseHoverPanel/DatabaseHoverPanel", () => ({
  DatabaseHoverPanel: () => <div data-testid="hover-panel" />,
}));

vi.mock("./DatabaseHoverPanel/hover-state", () => ({
  HoverStateProvider: ({ children }: { children: React.ReactNode }) => (
    <>{children}</>
  ),
  useHoverState: () => ({
    state: undefined,
    position: { x: 0, y: 0 },
    setPosition: vi.fn(),
    update: vi.fn(),
  }),
  useProvideHoverState: () => ({
    state: undefined,
    position: { x: 0, y: 0 },
    setPosition: vi.fn(),
    update: vi.fn(),
  }),
}));

vi.mock("./TreeNode/Label", () => ({
  Label: () => <div data-testid="tree-node-label" />,
}));

vi.mock("./tree", () => ({
  useSQLEditorTreeByEnvironment: () => ({
    tree: [],
    expandedState: { initialized: true, expandedKeys: [] },
    setExpandedKeys: vi.fn(),
    buildTree: vi.fn(),
    prepareDatabases: vi.fn().mockResolvedValue(undefined),
    fetchDatabases: vi.fn().mockResolvedValue(undefined),
    fetchDataState: { loading: false },
  }),
}));

let ConnectionPane: typeof import("./ConnectionPane").ConnectionPane;

const renderIntoContainer = (element: ReactElement) => {
  const container = document.createElement("div");
  const root = createRoot(container);
  document.body.appendChild(container);
  return {
    container,
    render: () => {
      act(() => {
        root.render(element);
      });
    },
    unmount: () => {
      act(() => {
        root.unmount();
      });
      container.remove();
    },
  };
};

beforeEach(async () => {
  vi.clearAllMocks();
  mocks.usePiniaBridge.mockImplementation((getter) => getter());
  // Reset feature flags default to true for render-path tests.
  mocks.features.batchQuery = true;
  mocks.features.databaseGroups = true;
  mocks.tabStore.supportBatchMode = true;
  mocks.tabStore.isInBatchMode = false;
  mocks.tabStore.currentTab = {
    id: "t-1",
    mode: "WORKSHEET",
    title: "Untitled",
    connection: { database: "", instance: "" },
    batchQueryContext: { databases: [], databaseGroups: [] },
  };
  ({ ConnectionPane } = await import("./ConnectionPane"));
});

describe("ConnectionPane", () => {
  test("renders DATABASE tab content when show=true and no selection", () => {
    const { container, render, unmount } = renderIntoContainer(
      <ConnectionPane show={true} onMissingFeature={() => {}} />
    );
    render();
    expect(
      container.querySelector("[data-testid='advanced-search']")
    ).not.toBeNull();
    expect(
      container
        .querySelector("[data-testid='tabs']")
        ?.getAttribute("data-value")
    ).toBe("DATABASE");
    unmount();
  });

  test("renders the batch-mode header when supportBatchMode is true", () => {
    const { container, render, unmount } = renderIntoContainer(
      <ConnectionPane show={true} onMissingFeature={() => {}} />
    );
    render();
    expect(container.textContent).toContain(
      "sql-editor.batch-query.description"
    );
    unmount();
  });

  test("mounts the context menu + hover panel + modal harness", () => {
    const { container, render, unmount } = renderIntoContainer(
      <ConnectionPane show={true} onMissingFeature={() => {}} />
    );
    render();
    expect(
      container.querySelector("[data-testid='context-menu']")
    ).not.toBeNull();
    expect(
      container.querySelector("[data-testid='hover-panel']")
    ).not.toBeNull();
    unmount();
  });

  test("switches to DATABASE-GROUP tab when only groups are selected on open", () => {
    mocks.tabStore.currentTab = {
      ...mocks.tabStore.currentTab,
      batchQueryContext: {
        databases: [],
        databaseGroups: ["projects/p/databaseGroups/g"],
      },
    };
    const { container, render, unmount } = renderIntoContainer(
      <ConnectionPane show={true} onMissingFeature={() => {}} />
    );
    render();
    expect(
      container
        .querySelector("[data-testid='tabs']")
        ?.getAttribute("data-value")
    ).toBe("DATABASE-GROUP");
    unmount();
  });

  test("disables DATABASE-GROUP tab when batch-query or database-group features missing", () => {
    mocks.features.batchQuery = false;
    mocks.features.databaseGroups = false;
    const { container, render, unmount } = renderIntoContainer(
      <ConnectionPane show={true} onMissingFeature={() => {}} />
    );
    render();
    const trigger = container.querySelector(
      "[data-testid='tab-trigger-DATABASE-GROUP']"
    ) as HTMLButtonElement;
    expect(trigger.disabled).toBe(true);
    unmount();
  });
});

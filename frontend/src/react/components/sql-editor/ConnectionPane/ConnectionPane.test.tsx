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
  const uiStore = { showConnectionPanel: true, asidePanelTab: "SCHEMA" };
  const databaseStore = {
    batchGetOrFetchDatabases: vi.fn().mockResolvedValue([]),
    getDatabaseByName: vi.fn(() => undefined),
    getOrFetchDatabaseByName: vi.fn().mockResolvedValue(undefined),
  };
  const dbGroupStore = {
    fetchDBGroupListByProjectName: vi.fn().mockResolvedValue([]),
    // The component reads `.name` and `.matchedDatabases` off the result
    // and validates `.name` via `isValidDatabaseGroupName`. Returning an
    // unknown-name placeholder preserves the "skip invalid" branch
    // without leaking real groups into tests that don't set them up.
    getDBGroupByName: vi.fn(() => ({
      name: "",
      matchedDatabases: [],
      title: "",
    })),
    getOrFetchDBGroupByName: vi.fn().mockResolvedValue(undefined),
  };
  const environmentStore = {
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
  };
  const projectStore = {
    getProjectByName: vi.fn(() => ({
      name: "projects/p",
      title: "Project One",
    })),
  };
  const treeStore = {
    state: "READY" as "LOADING" | "READY" | "UNSET",
    nodeKeysByTarget: vi.fn(() => []),
  };
  const instanceStore = {
    getInstanceByName: vi.fn(),
  };
  const currentUser = { value: { email: "u@b.com" } };
  return {
    tabStore,
    editorStore,
    uiStore,
    databaseStore,
    dbGroupStore,
    environmentStore,
    projectStore,
    treeStore,
    instanceStore,
    currentUser,
    useVueState: vi.fn<(getter: () => unknown) => unknown>(),
    featureToRef: vi.fn(() => ({ value: true })),
    pushNotification: vi.fn(),
  };
});

vi.mock("react-i18next", () => ({
  useTranslation: () => ({ t: (key: string) => key }),
}));

vi.mock("@/react/hooks/useVueState", () => ({
  useVueState: mocks.useVueState,
}));

vi.mock("@/store", () => ({
  featureToRef: mocks.featureToRef,
  pushNotification: mocks.pushNotification,
  useCurrentUserV1: () => mocks.currentUser,
  useDatabaseV1Store: () => mocks.databaseStore,
  useDBGroupStore: () => mocks.dbGroupStore,
  useEnvironmentV1Store: () => mocks.environmentStore,
  useInstanceV1Store: () => mocks.instanceStore,
  useProjectV1Store: () => mocks.projectStore,
}));

vi.mock("@/react/stores/sqlEditor/tab-vue-state", () => ({
  useSQLEditorTabStore: () => mocks.tabStore,
}));

vi.mock("@/react/stores/sqlEditor/editor-vue-state", () => ({
  useSQLEditorVueState: () => mocks.editorStore,
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

vi.mock("@/store/modules", () => ({
  useDBGroupStore: () => mocks.dbGroupStore,
}));

vi.mock("@/store/modules/v1/common", () => ({
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
  mocks.useVueState.mockImplementation((getter) => getter());
  // Reset feature flags default to true for render-path tests.
  mocks.featureToRef.mockReturnValue({ value: true });
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
    mocks.featureToRef.mockImplementation(() => ({ value: false }));
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

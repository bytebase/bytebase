import { act, type ReactElement } from "react";
import { createRoot } from "react-dom/client";
import { afterEach, beforeEach, describe, expect, test, vi } from "vitest";

(
  globalThis as { IS_REACT_ACT_ENVIRONMENT?: boolean }
).IS_REACT_ACT_ENVIRONMENT = true;

class StubResizeObserver implements ResizeObserver {
  constructor(_cb: ResizeObserverCallback) {}
  observe(): void {}
  unobserve(): void {}
  disconnect(): void {}
}
(
  globalThis as unknown as { ResizeObserver: typeof ResizeObserver }
).ResizeObserver = StubResizeObserver as unknown as typeof ResizeObserver;

const mocks = vi.hoisted(() => ({
  databaseRefValue: { name: "" } as {
    name: string;
    instanceResource?: unknown;
  },
  instanceRefValue: { engine: 1 },
  currentTab: null as unknown,
  getOrFetchDatabaseMetadata: vi
    .fn()
    .mockResolvedValue({ name: "instances/i/databases/db", schemas: [] }),
  isValidDatabaseName: vi.fn(() => false),
}));

vi.mock("react-i18next", () => ({
  useTranslation: () => ({ t: (k: string) => k, i18n: { language: "en-US" } }),
  Trans: ({ i18nKey }: { i18nKey: string }) => <span>{i18nKey}</span>,
  initReactI18next: { type: "3rdParty", init: () => {} },
}));

vi.mock("@/lib/i18n", () => ({
  default: { t: (k: string) => k },
}));

vi.mock("@/stores", () => ({
  pushNotification: vi.fn(),
}));

vi.mock("@/hooks/useAppDatabase", () => ({
  useAppDatabase: (name: string) => ({ name }),
}));

vi.mock("@/stores/app", () => {
  const state = {
    getDatabaseByName: (name: string) => ({ name }),
    getOrFetchDatabaseByName: vi.fn(async (name: string) => ({ name })),
    batchGetOrFetchDatabases: vi.fn(async () => []),
    syncDatabase: vi.fn(),
    fetchDatabases: vi.fn(async () => ({ databases: [], nextPageToken: "" })),
    databasesByName: {} as Record<string, unknown>,
    // SchemaPane reads cached metadata reactively via this getter; return
    // undefined so the tree stays empty unless a test seeds the cache.
    getCachedDatabaseMetadata: () => undefined,
    getOrFetchDatabaseMetadata: mocks.getOrFetchDatabaseMetadata,
  };
  return {
    useAppStore: Object.assign(
      (selector: (s: unknown) => unknown) => selector(state),
      { getState: () => state }
    ),
  };
});

vi.mock("@/modules/sql-editor/hooks/useSQLEditorState", () => ({
  // Returns plain values now — `database` is the unwrapped object.
  useConnectionOfCurrentSQLEditorTab: () => ({
    database: mocks.databaseRefValue,
    instance: mocks.instanceRefValue,
  }),
}));

// Minimal Zustand tabs-state stub. The selector hook runs against a
// state built from `mocks.currentTab`; the imperative getter exposes the
// same shape plus the mutators SchemaPane invokes.
const tabsState = () => ({
  currentTabId: mocks.currentTab ? "t1" : "",
  tabsById: new Map<string, unknown>(
    mocks.currentTab ? [["t1", mocks.currentTab]] : []
  ),
  openTmpTabList: [],
  addTab: vi.fn(),
  setCurrentTabId: vi.fn(),
  updateCurrentTab: vi.fn(),
  updateTab: vi.fn(),
});

vi.mock("@/modules/sql-editor/store/tab", () => ({
  useSQLEditorTabState: (selector: (s: unknown) => unknown) =>
    selector(tabsState()),
  getSQLEditorTabsState: () => tabsState(),
}));

vi.mock("@/types", () => ({
  isValidDatabaseName: mocks.isValidDatabaseName,
  DEFAULT_SQL_EDITOR_TAB_MODE: "WORKSHEET",
  dialectOfEngineV1: () => "MYSQL",
  languageOfEngineV1: () => "sql",
  typeToView: (type: string) => type.toUpperCase(),
  getDateForPbTimestampProtoEs: () => undefined,
}));

vi.mock("@/types/proto-es/v1/common_pb", () => ({
  Engine: { REDIS: 7, MYSQL: 1 },
}));

vi.mock("@/types/proto-es/v1/database_service_pb", () => ({
  GetSchemaStringRequest_ObjectType: { TABLE: 1, VIEW: 2 },
}));

vi.mock("@/types/sqlEditor/tabViewState", () => ({
  defaultViewState: () => ({ view: "CODE" }),
}));

vi.mock("@/utils", () => ({
  defaultSQLEditorTab: () => ({
    id: "n",
    title: "",
    connection: { instance: "", database: "" },
    viewState: { view: "CODE" },
    treeState: { database: "", keys: [] },
    editorState: { selection: null },
    mode: "WORKSHEET",
    status: "NEW",
  }),
  extractDatabaseResourceName: (n: string) => ({
    instance: "instances/i",
    databaseName: n.split("/").pop() ?? "",
  }),
  extractInstanceResourceName: (n: string) => n,
  extractProjectResourceName: (n: string) => n,
  generateSimpleDeleteStatement: () => "",
  generateSimpleInsertStatement: () => "",
  generateSimpleSelectAllStatement: () => "",
  generateSimpleUpdateStatement: () => "",
  getInstanceResource: () => ({ engine: 1 }),
  instanceV1HasAlterSchema: () => false,
  isSameSQLEditorConnection: () => false,
  sortByDictionary: () => {},
  supportGetStringSchema: () => false,
  isDev: () => false,
  bytesToString: (n: number) => `${n} B`,
  minmax: (v: number, lo: number, hi: number) => Math.max(lo, Math.min(hi, v)),
  instanceV1Name: (i: { title?: string }) => i.title ?? "",
}));

vi.mock("@/utils/dom", () => ({
  findAncestor: () => null,
}));

vi.mock("@/utils/v1/instance", () => ({
  instanceV1SupportsExternalTable: () => false,
  instanceV1SupportsPackage: () => false,
  instanceV1SupportsSequence: () => false,
}));
vi.mock("@/components/instance/constants", () => ({
  EngineIconPath: { MYSQL: "/mysql.svg" } as Record<string, string>,
}));

vi.mock("@/app/router", async (importOriginal) => ({
  ...(await importOriginal<typeof import("@/app/router")>()),
  router: { resolve: () => ({ href: "/" }) },
}));
vi.mock("@/lib/keyWithPosition", () => ({
  keyWithPosition: (k: string, p: number) => `${k}###${p}`,
}));
vi.mock("@/modules/sql-editor/model/events", () => ({
  sqlEditorEvents: { emit: vi.fn(), on: vi.fn(() => () => {}) },
}));
vi.mock("@/hooks/useExecuteSQL", () => ({
  useExecuteSQL: () => ({ execute: vi.fn() }),
}));
vi.mock("@/components/monaco/sqlFormatter", () => ({
  formatSQL: async (sql: string) => ({ data: sql, error: null }),
}));

vi.mock("@/components/HumanizeTs", () => ({
  HumanizeTs: () => <span />,
}));

vi.mock("@/components/ui/input", () => ({
  Input: ({
    size,
    ...props
  }: React.InputHTMLAttributes<HTMLInputElement> & { size?: string }) => (
    <input data-testid="search-input" data-size={size} {...props} />
  ),
}));

vi.mock("@/components/ui/button", () => ({
  Button: ({
    children,
    onClick,
    disabled,
    size,
  }: {
    children: React.ReactNode;
    onClick?: () => void;
    disabled?: boolean;
    size?: string;
  }) => (
    <button
      type="button"
      data-size={size}
      disabled={disabled}
      onClick={onClick}
    >
      {children}
    </button>
  ),
}));

vi.mock("@/components/ui/tooltip", () => ({
  Tooltip: ({ children }: { children: React.ReactNode }) => <>{children}</>,
}));

vi.mock("@/components/ui/dropdown-menu", () => ({
  DropdownMenu: () => null,
  DropdownMenuContent: () => null,
  DropdownMenuItem: () => null,
  DropdownMenuSubmenu: () => null,
  DropdownMenuSubmenuContent: () => null,
  DropdownMenuSubmenuTrigger: () => null,
}));

vi.mock("@/components/ui/dialog", () => ({
  Dialog: ({ children }: { children: React.ReactNode }) => <>{children}</>,
  DialogContent: ({ children }: { children: React.ReactNode }) => (
    <div>{children}</div>
  ),
  DialogTitle: ({ children }: { children: React.ReactNode }) => (
    <div>{children}</div>
  ),
}));

vi.mock("@/components/TableSchemaViewer", () => ({
  TableSchemaViewer: () => <div data-testid="table-schema-viewer" />,
}));

vi.mock("@base-ui/react/menu", () => ({
  Menu: { Trigger: () => null },
}));

vi.mock("@/components/ui/layer", () => ({
  LAYER_SURFACE_CLASS: "",
  getLayerRoot: () => document.body,
}));

vi.mock("@/components/ui/tree", () => ({
  Tree: () => <div data-testid="tree" />,
}));

vi.mock("@/components/ui/tree-utils", () => ({
  countVisibleRows: () => 0,
}));

let SchemaPane: typeof import("./SchemaPane").SchemaPane;

const renderInto = (element: ReactElement) => {
  const container = document.createElement("div");
  document.body.appendChild(container);
  const root = createRoot(container);
  return {
    container,
    render: () => act(() => root.render(element)),
    unmount: () => {
      act(() => root.unmount());
      container.remove();
    },
  };
};

beforeEach(async () => {
  vi.clearAllMocks();
  mocks.databaseRefValue = { name: "" };
  mocks.currentTab = null;
  mocks.isValidDatabaseName.mockReturnValue(false);
  ({ SchemaPane } = await import("./SchemaPane"));
});

afterEach(() => {
  document.body.innerHTML = "";
});

describe("SchemaPane shell", () => {
  test("renders search input + sync button without a connection", () => {
    const { container, render, unmount } = renderInto(<SchemaPane />);
    render();
    expect(
      container.querySelector("[data-testid='search-input']")
    ).not.toBeNull();
    expect(container.querySelector("button")).not.toBeNull();
    unmount();
  });

  test("aligns search input and sync button control sizes", () => {
    const { container, render, unmount } = renderInto(<SchemaPane />);
    render();

    const input = container.querySelector("[data-testid='search-input']");
    const button = container.querySelector("button");
    expect(input?.getAttribute("data-size")).toBe("sm");
    expect(button?.getAttribute("data-size")).toBe("sm");
    expect(input?.className).not.toContain("h-8");

    unmount();
  });

  test("does not call getOrFetchDatabaseMetadata when database name is invalid", () => {
    const { render, unmount } = renderInto(<SchemaPane />);
    render();
    expect(mocks.getOrFetchDatabaseMetadata).not.toHaveBeenCalled();
    unmount();
  });

  test("fetches metadata when database name becomes valid", () => {
    mocks.isValidDatabaseName.mockReturnValue(true);
    mocks.databaseRefValue = { name: "instances/i/databases/db" };
    const { render, unmount } = renderInto(<SchemaPane />);
    render();
    expect(mocks.getOrFetchDatabaseMetadata).toHaveBeenCalledWith({
      database: "instances/i/databases/db",
    });
    unmount();
  });
});

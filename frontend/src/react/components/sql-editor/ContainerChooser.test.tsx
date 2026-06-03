import type { ReactElement } from "react";
import { act } from "react";
import { createRoot } from "react-dom/client";
import { beforeEach, describe, expect, test, vi } from "vitest";

(
  globalThis as { IS_REACT_ACT_ENVIRONMENT?: boolean }
).IS_REACT_ACT_ENVIRONMENT = true;

const mocks = vi.hoisted(() => ({
  useTranslation: vi.fn(() => ({ t: (key: string) => key })),
  tabTable: undefined as string | undefined,
  currentTabId: "tab1",
  getSQLEditorTabsState: vi.fn(),
  // Default schemas shape — tests override `mocks.metadata` to drive
  // different option lists.
  metadata: {
    schemas: [] as Array<{ name: string; tables: { name: string }[] }>,
  },
  useConnectionOfCurrentSQLEditorTab: vi.fn(),
  router: {
    currentRoute: { value: { query: {} } },
    afterEach: () => () => {},
  },
}));

vi.mock("react-i18next", () => ({
  useTranslation: mocks.useTranslation,
}));

vi.mock("@/react/hooks/useAppDatabaseMetadata", () => ({
  useAppDatabaseMetadata: () => mocks.metadata,
}));

vi.mock("@/react/hooks/useSQLEditorBridge", () => ({
  useConnectionOfCurrentSQLEditorTab: mocks.useConnectionOfCurrentSQLEditorTab,
}));

vi.mock("@/react/stores/sqlEditor/tab", () => ({
  useSQLEditorTabState: (
    selector: (s: {
      currentTabId: string;
      tabsById: Map<string, { connection: { table?: string } }>;
    }) => unknown
  ) =>
    selector({
      currentTabId: mocks.currentTabId,
      tabsById: new Map([
        [mocks.currentTabId, { connection: { table: mocks.tabTable } }],
      ]),
    }),
  getSQLEditorTabsState: mocks.getSQLEditorTabsState,
}));

vi.mock("@/react/router", async (importOriginal) => ({
  ...(await importOriginal<typeof import("@/react/router")>()),
  router: mocks.router,
}));

// Mock Engine enum
vi.mock("@/types/proto-es/v1/common_pb", () => ({
  Engine: { COSMOSDB: "COSMOSDB", MYSQL: "MYSQL" },
}));

vi.mock("./ConnectChooser", () => ({
  ConnectChooser: ({
    placeholder,
    options,
    isChosen,
    value,
  }: {
    placeholder: string;
    options: { value: string; label: string }[];
    isChosen: boolean;
    value: string;
  }) => (
    <div data-testid="connect-chooser">
      <span data-testid="placeholder">{placeholder}</span>
      <span data-testid="value">{value}</span>
      <span data-testid="is-chosen">{String(isChosen)}</span>
      {options.map((o) => (
        <span key={o.value} data-testid={`option-${o.value}`}>
          {o.label}
        </span>
      ))}
    </div>
  ),
}));

let ContainerChooser: typeof import("./ContainerChooser").ContainerChooser;

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

const mockCosmosConnection = {
  instance: { engine: "COSMOSDB" },
  database: { name: "instances/inst1/databases/cosmosdb" },
};

const mockMySQLConnection = {
  instance: { engine: "MYSQL" },
  database: { name: "instances/inst1/databases/db1" },
};

beforeEach(async () => {
  vi.clearAllMocks();
  mocks.useConnectionOfCurrentSQLEditorTab.mockReturnValue(
    mockCosmosConnection
  );
  mocks.tabTable = undefined;
  mocks.currentTabId = "tab1";
  mocks.getSQLEditorTabsState.mockReturnValue({
    currentTabId: "tab1",
    tabsById: new Map([["tab1", { connection: {} }]]),
    updateCurrentTab: vi.fn(),
  });
  mocks.metadata = {
    schemas: [
      {
        name: "default",
        tables: [{ name: "container1" }, { name: "container2" }],
      },
    ],
  };
  ({ ContainerChooser } = await import("./ContainerChooser"));
});

describe("ContainerChooser", () => {
  test("renders nothing when engine is not COSMOSDB", () => {
    mocks.useConnectionOfCurrentSQLEditorTab.mockReturnValue(
      mockMySQLConnection
    );
    const { container, render, unmount } = renderIntoContainer(
      <ContainerChooser />
    );
    render();
    expect(
      container.querySelector("[data-testid='connect-chooser']")
    ).toBeNull();
    unmount();
  });

  test("renders ConnectChooser when engine is COSMOSDB", () => {
    const { container, render, unmount } = renderIntoContainer(
      <ContainerChooser />
    );
    render();
    expect(
      container.querySelector("[data-testid='connect-chooser']")
    ).not.toBeNull();
    unmount();
  });

  test("options include unspecified entry and all table names", () => {
    const { container, render, unmount } = renderIntoContainer(
      <ContainerChooser />
    );
    render();
    // Unspecified option
    expect(container.querySelector("[data-testid='option--1']")).not.toBeNull();
    // Table names from mock
    expect(
      container.querySelector("[data-testid='option-container1']")
    ).not.toBeNull();
    expect(
      container.querySelector("[data-testid='option-container2']")
    ).not.toBeNull();
    unmount();
  });

  test("is not chosen when no table is selected", () => {
    const { container, render, unmount } = renderIntoContainer(
      <ContainerChooser />
    );
    render();
    expect(
      container.querySelector("[data-testid='is-chosen']")?.textContent
    ).toBe("false");
    unmount();
  });

  test("is chosen when a table is selected", () => {
    mocks.tabTable = "container1";
    const { container, render, unmount } = renderIntoContainer(
      <ContainerChooser />
    );
    render();
    expect(
      container.querySelector("[data-testid='is-chosen']")?.textContent
    ).toBe("true");
    unmount();
  });
});

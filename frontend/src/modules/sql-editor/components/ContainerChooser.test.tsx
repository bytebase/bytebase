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

vi.mock("@/hooks/useAppDatabaseMetadata", () => ({
  useAppDatabaseMetadata: () => mocks.metadata,
}));

vi.mock("@/modules/sql-editor/hooks/useSQLEditorState", () => ({
  useConnectionOfCurrentSQLEditorTab: mocks.useConnectionOfCurrentSQLEditorTab,
}));

vi.mock("@/modules/sql-editor/store/tab", () => ({
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

vi.mock("@/app/router", async (importOriginal) => ({
  ...(await importOriginal<typeof import("@/app/router")>()),
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
    onChange,
    dropdownClassName,
    dropdownMinWidth,
    triggerClassName,
    triggerVariant,
  }: {
    placeholder: string;
    options: { value: string; label: string }[];
    isChosen: boolean;
    value: string;
    onChange: (value: string) => void;
    dropdownClassName?: string;
    dropdownMinWidth?: number;
    triggerClassName?: string;
    triggerVariant?: string;
  }) => (
    <div
      data-testid="connect-chooser"
      data-dropdown-class-name={dropdownClassName}
      data-dropdown-min-width={dropdownMinWidth}
      data-trigger-class-name={triggerClassName}
      data-trigger-variant={triggerVariant}
    >
      <span data-testid="placeholder">{placeholder}</span>
      <span data-testid="value">{value}</span>
      <span data-testid="is-chosen">{String(isChosen)}</span>
      {options.map((o) => (
        <button
          key={o.value}
          type="button"
          data-testid={`option-${o.value}`}
          onClick={() => onChange(o.value)}
        >
          {o.label}
        </button>
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
  mocks.router.currentRoute.value.query = {};
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
    expect(container.querySelector("[data-testid='placeholder']")?.textContent).toBe(
      "database.container.select"
    );
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
    expect(
      container
        .querySelector("[data-testid='connect-chooser']")
        ?.getAttribute("data-trigger-class-name")
    ).toBeNull();
    expect(
      container
        .querySelector("[data-testid='connect-chooser']")
        ?.getAttribute("data-dropdown-min-width")
    ).toBe("192");
    unmount();
  });

  test("renders run variant when mounted for the Run button", () => {
    const { container, render, unmount } = renderIntoContainer(
      <ContainerChooser variant="run" />
    );
    render();

    expect(
      container
        .querySelector("[data-testid='connect-chooser']")
        ?.getAttribute("data-trigger-variant")
    ).toBe("run");
    unmount();
  });

  test("does not auto-select the only known CosmosDB container", () => {
    const updateCurrentTab = vi.fn();
    mocks.getSQLEditorTabsState.mockReturnValue({
      currentTabId: "tab1",
      tabsById: new Map([["tab1", { connection: { database: "db" } }]]),
      updateCurrentTab,
    });
    mocks.metadata = {
      schemas: [
        {
          name: "default",
          tables: [{ name: "only-container" }],
        },
      ],
    };

    const { render, unmount } = renderIntoContainer(<ContainerChooser />);
    render();

    expect(updateCurrentTab).not.toHaveBeenCalled();
    unmount();
  });

  test("allows clearing the selected container even when it is the only container", () => {
    const updateCurrentTab = vi.fn((payload) => {
      mocks.tabTable = payload.connection.table;
    });
    mocks.tabTable = "only-container";
    mocks.getSQLEditorTabsState.mockImplementation(() => ({
      currentTabId: "tab1",
      tabsById: new Map([
        [
          "tab1",
          {
            connection: {
              database: "db",
              table: mocks.tabTable,
            },
          },
        ],
      ]),
      updateCurrentTab,
    }));
    mocks.metadata = {
      schemas: [
        {
          name: "default",
          tables: [{ name: "only-container" }],
        },
      ],
    };

    const { container, render, unmount } = renderIntoContainer(
      <ContainerChooser />
    );
    render();
    act(() => {
      container
        .querySelector<HTMLButtonElement>("[data-testid='option--1']")
        ?.click();
    });
    render();

    expect(updateCurrentTab).toHaveBeenCalledTimes(1);
    expect(updateCurrentTab).toHaveBeenCalledWith({
      connection: {
        database: "db",
        table: undefined,
      },
    });
    unmount();
  });

  test("does not restore a cleared container from a stale route query", () => {
    const updateCurrentTab = vi.fn((payload) => {
      mocks.tabTable = payload.connection.table;
    });
    mocks.tabTable = "ED";
    mocks.router.currentRoute.value.query = { table: "ED" };
    mocks.getSQLEditorTabsState.mockImplementation(() => ({
      currentTabId: "tab1",
      tabsById: new Map([
        [
          "tab1",
          {
            connection: {
              database: "db",
              table: mocks.tabTable,
            },
          },
        ],
      ]),
      updateCurrentTab,
    }));
    mocks.metadata = {
      schemas: [
        {
          name: "default",
          tables: [{ name: "WorldCities" }, { name: "ED" }],
        },
      ],
    };

    const first = renderIntoContainer(<ContainerChooser />);
    first.render();
    updateCurrentTab.mockClear();

    act(() => {
      first.container
        .querySelector<HTMLButtonElement>("[data-testid='option--1']")
        ?.click();
    });
    first.unmount();

    const second = renderIntoContainer(<ContainerChooser />);
    second.render();

    expect(updateCurrentTab).toHaveBeenCalledTimes(1);
    expect(updateCurrentTab).toHaveBeenCalledWith({
      connection: {
        database: "db",
        table: undefined,
      },
    });
    second.unmount();
  });
});

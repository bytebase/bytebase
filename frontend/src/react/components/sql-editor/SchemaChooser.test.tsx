import type { ReactElement } from "react";
import { act } from "react";
import { createRoot } from "react-dom/client";
import { beforeEach, describe, expect, test, vi } from "vitest";

(
  globalThis as { IS_REACT_ACT_ENVIRONMENT?: boolean }
).IS_REACT_ACT_ENVIRONMENT = true;

const mocks = vi.hoisted(() => ({
  useTranslation: vi.fn(() => ({ t: (key: string) => key })),
  // Selector hook over the Zustand tabs store; returns the current tab's
  // `connection.schema` (and `currentTabId` for the seed effect).
  tabSchema: undefined as string | undefined,
  currentTabId: "tab1",
  getSQLEditorTabsState: vi.fn(),
  metadata: { schemas: [] as Array<{ name: string }> },
  useConnectionOfCurrentSQLEditorTab: vi.fn(),
  instanceAllowsSchemaScopedQuery: vi.fn(),
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
      tabsById: Map<string, { connection: { schema?: string } }>;
    }) => unknown
  ) =>
    selector({
      currentTabId: mocks.currentTabId,
      tabsById: new Map([
        [mocks.currentTabId, { connection: { schema: mocks.tabSchema } }],
      ]),
    }),
  getSQLEditorTabsState: mocks.getSQLEditorTabsState,
}));

vi.mock("@/utils", () => ({
  instanceAllowsSchemaScopedQuery: mocks.instanceAllowsSchemaScopedQuery,
}));

vi.mock("@/router", () => ({
  router: mocks.router,
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

let SchemaChooser: typeof import("./SchemaChooser").SchemaChooser;

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

const mockConnection = {
  instance: { engine: "MYSQL" },
  database: { name: "instances/inst1/databases/db1" },
};

beforeEach(async () => {
  vi.clearAllMocks();
  mocks.useConnectionOfCurrentSQLEditorTab.mockReturnValue(mockConnection);
  mocks.tabSchema = undefined;
  mocks.currentTabId = "tab1";
  mocks.getSQLEditorTabsState.mockReturnValue({
    currentTabId: "tab1",
    tabsById: new Map([["tab1", { connection: {} }]]),
    updateCurrentTab: vi.fn(),
  });
  mocks.metadata = {
    schemas: [{ name: "public" }, { name: "private" }],
  };
  mocks.instanceAllowsSchemaScopedQuery.mockReturnValue(true);
  ({ SchemaChooser } = await import("./SchemaChooser"));
});

describe("SchemaChooser", () => {
  test("renders nothing when engine does not allow schema-scoped queries", () => {
    mocks.instanceAllowsSchemaScopedQuery.mockReturnValue(false);
    const { container, render, unmount } = renderIntoContainer(
      <SchemaChooser />
    );
    render();
    expect(
      container.querySelector("[data-testid='connect-chooser']")
    ).toBeNull();
    unmount();
  });

  test("renders ConnectChooser when engine supports schemas", () => {
    mocks.instanceAllowsSchemaScopedQuery.mockReturnValue(true);
    const { container, render, unmount } = renderIntoContainer(
      <SchemaChooser />
    );
    render();
    expect(
      container.querySelector("[data-testid='connect-chooser']")
    ).not.toBeNull();
    unmount();
  });

  test("options include unspecified entry and each schema name", () => {
    const { container, render, unmount } = renderIntoContainer(
      <SchemaChooser />
    );
    render();
    // Unspecified option
    expect(container.querySelector("[data-testid='option--1']")).not.toBeNull();
    // Schema names from mock
    expect(
      container.querySelector("[data-testid='option-public']")
    ).not.toBeNull();
    expect(
      container.querySelector("[data-testid='option-private']")
    ).not.toBeNull();
    unmount();
  });

  test("is chosen when a schema is selected on the tab", () => {
    mocks.tabSchema = "public";

    const { container, render, unmount } = renderIntoContainer(
      <SchemaChooser />
    );
    render();
    // Value should be "public" (non-unspecified)
    expect(
      container.querySelector("[data-testid='is-chosen']")?.textContent
    ).toBe("true");
    unmount();
  });
});

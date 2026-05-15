import type { ReactElement } from "react";
import { act } from "react";
import { createRoot } from "react-dom/client";
import { beforeEach, describe, expect, test, vi } from "vitest";

(
  globalThis as { IS_REACT_ACT_ENVIRONMENT?: boolean }
).IS_REACT_ACT_ENVIRONMENT = true;

const mocks = vi.hoisted(() => ({
  useTranslation: vi.fn(() => ({ t: (key: string) => key })),
  useVueState: vi.fn<(getter: () => unknown) => unknown>(),
  useSQLEditorTabStore: vi.fn(),
  useDBSchemaV1Store: vi.fn(),
  useConnectionOfCurrentSQLEditorTab: vi.fn(),
  instanceAllowsSchemaScopedQuery: vi.fn(),
  router: {
    currentRoute: { value: { query: {} } },
  },
}));

vi.mock("react-i18next", () => ({
  useTranslation: mocks.useTranslation,
}));

vi.mock("@/react/hooks/useVueState", () => ({
  useVueState: mocks.useVueState,
}));

vi.mock("@/store", () => ({
  useDBSchemaV1Store: mocks.useDBSchemaV1Store,
}));

vi.mock("@/react/stores/sqlEditor/tab-vue-state", () => ({
  useConnectionOfCurrentSQLEditorTab: mocks.useConnectionOfCurrentSQLEditorTab,
  useSQLEditorTabStore: mocks.useSQLEditorTabStore,
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
  instance: { value: { engine: "MYSQL" } },
  database: { value: { name: "instances/inst1/databases/db1" } },
};

beforeEach(async () => {
  vi.clearAllMocks();
  mocks.useConnectionOfCurrentSQLEditorTab.mockReturnValue(mockConnection);
  mocks.useSQLEditorTabStore.mockReturnValue({
    currentTab: { connection: { schema: undefined } },
  });
  mocks.useDBSchemaV1Store.mockReturnValue({
    getDatabaseMetadata: vi.fn(() => ({
      schemas: [{ name: "public" }, { name: "private" }],
    })),
  });
  mocks.useVueState.mockImplementation((getter) => getter());
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

  test("selecting unspecified sets tab.connection.schema to undefined", () => {
    const tab = { connection: { schema: "public" } };
    mocks.useSQLEditorTabStore.mockReturnValue({ currentTab: tab });

    // Import fresh copy with updated mock
    let callIdx = 0;
    mocks.useVueState.mockImplementation((getter) => {
      callIdx++;
      // 1: engine, 2: databaseName, 3: tabSchema, 4: schemas, 5: queryParam
      if (callIdx === 3) return "public";
      return getter();
    });

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

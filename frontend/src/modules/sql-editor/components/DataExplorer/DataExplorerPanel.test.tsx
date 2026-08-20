import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { beforeEach, describe, expect, test, vi } from "vitest";
import { Engine } from "@/types/proto-es/v1/common_pb";
import { DataExplorerPanel } from "./DataExplorerPanel";

const mocks = vi.hoisted(() => {
  const tab = {
    id: "explorer",
    mode: "DATA_EXPLORER",
    connection: {
      instance: "instances/i",
      database: "instances/i/databases/db",
      schema: "",
      table: "users",
    },
    editorState: { selection: null },
    dataExplorer: { filter: "", initialized: false },
    databaseQueryContexts: new Map(),
  };
  return {
    tab,
    engine: 23 as Engine,
    resultRowsLimit: 500,
    execute: vi.fn().mockResolvedValue(undefined),
    updateTab: vi.fn((_: string, patch: Record<string, unknown>) => {
      Object.assign(tab, patch);
      return tab;
    }),
    deleteDatabaseQueryContext: vi.fn(),
  };
});

vi.mock("react-i18next", () => ({
  initReactI18next: { type: "3rdParty", init: () => undefined },
  useTranslation: () => ({ t: (key: string) => key }),
}));

vi.mock("@/hooks/useExecuteSQL", () => ({
  useExecuteSQL: () => ({ execute: mocks.execute }),
}));

vi.mock("@/modules/sql-editor/hooks/useSQLEditorState", () => ({
  useConnectionOfCurrentSQLEditorTab: () => ({
    connection: mocks.tab.connection,
    database: { name: mocks.tab.connection.database },
    instance: { engine: mocks.engine },
  }),
}));

vi.mock("@/modules/sql-editor/store/editor", () => ({
  useSQLEditorEditorState: (selector: (value: unknown) => unknown) =>
    selector({ resultRowsLimit: mocks.resultRowsLimit }),
}));

vi.mock("@/modules/sql-editor/store/tab", () => {
  const state = {
    currentTabId: "explorer",
    tabsById: new Map([["explorer", mocks.tab]]),
  };
  return {
    useSQLEditorTabState: (selector: (value: typeof state) => unknown) =>
      selector(state),
    getSQLEditorTabsState: () => ({
      ...state,
      updateTab: mocks.updateTab,
      deleteDatabaseQueryContext: mocks.deleteDatabaseQueryContext,
    }),
  };
});

vi.mock(
  "@/modules/sql-editor/components/ResultPanel/DatabaseQueryContext",
  () => ({ DatabaseQueryContext: () => null })
);

vi.mock("@/modules/sql-editor/components/QueryContextSettingPopover", () => ({
  QueryContextSettingPopover: ({ disabled }: { disabled?: boolean }) => (
    <div
      data-testid="query-context-setting-popover"
      data-disabled={String(disabled)}
    />
  ),
}));

beforeEach(() => {
  vi.clearAllMocks();
  mocks.tab.dataExplorer = { filter: "", initialized: false };
  mocks.tab.databaseQueryContexts = new Map();
  mocks.tab.connection.schema = "";
  mocks.tab.connection.table = "users";
  mocks.engine = Engine.COSMOSDB;
  mocks.resultRowsLimit = 500;
});

describe("DataExplorerPanel", () => {
  test("runs the default CosmosDB query once on first open", async () => {
    render(<DataExplorerPanel />);

    await waitFor(() => {
      expect(mocks.execute).toHaveBeenCalledWith({
        connection: mocks.tab.connection,
        statement: "SELECT * FROM c",
        engine: Engine.COSMOSDB,
        explain: false,
        limit: 500,
        selection: null,
      });
    });
    expect(mocks.tab.dataExplorer.initialized).toBe(true);
  });

  test("joins the filter and runs it from the shared Run button", async () => {
    mocks.tab.dataExplorer = { filter: "", initialized: true };
    render(<DataExplorerPanel />);

    fireEvent.change(
      screen.getByLabelText("sql-editor.data-explorer-filter-placeholder"),
      { target: { value: "WHERE c.active = true" } }
    );
    fireEvent.click(
      screen.getByRole("button", { name: /\(limit\s*500\)/ })
    );

    await waitFor(() => {
      expect(mocks.execute).toHaveBeenCalledWith(
        expect.objectContaining({
          statement: "SELECT * FROM c WHERE c.active = true",
          limit: 500,
        })
      );
    });
  });

  test("builds a qualified relational preview query", async () => {
    mocks.engine = Engine.POSTGRES;
    mocks.tab.connection.schema = "public";
    mocks.tab.connection.table = "users";

    render(<DataExplorerPanel />);

    await waitFor(() => {
      expect(mocks.execute).toHaveBeenCalledWith(
        expect.objectContaining({
          statement: 'SELECT * FROM "public"."users";',
          engine: Engine.POSTGRES,
          limit: 500,
        })
      );
    });
    expect(
      screen.getByLabelText(
        "sql-editor.data-explorer-sql-filter-placeholder"
      )
    ).toBeInTheDocument();
  });

  test("disables the filter controls while the query is running", () => {
    mocks.tab.dataExplorer = { filter: "", initialized: true };
    mocks.tab.databaseQueryContexts = new Map([
      [mocks.tab.connection.database, [{ status: "EXECUTING" }]],
    ]);

    render(<DataExplorerPanel />);

    expect(
      screen.getByLabelText("sql-editor.data-explorer-filter-placeholder")
    ).toBeDisabled();
    expect(
      screen.getByRole("button", { name: /\(limit\s*500\)/ })
    ).toBeDisabled();
    expect(screen.getByTestId("query-context-setting-popover")).toHaveAttribute(
      "data-disabled",
      "true"
    );
  });
});

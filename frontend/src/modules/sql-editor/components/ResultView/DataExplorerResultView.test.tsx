import { create } from "@bufbuild/protobuf";
import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { beforeEach, describe, expect, test, vi } from "vitest";
import { Engine } from "@/types/proto-es/v1/common_pb";
import type { Database } from "@/types/proto-es/v1/database_service_pb";
import {
  QueryResultSchema,
  QueryRowSchema,
  RowValueSchema,
} from "@/types/proto-es/v1/sql_service_pb";
import { SQLResultViewProvider, useSQLResultViewContext } from "./context";
import { DataExplorerResultView } from "./DataExplorerResultView";
import type { ResultTableColumn, ResultTableRow } from "./types";

const mocks = vi.hoisted(() => {
  const tab = {
    id: "explorer",
    dataExplorer: {
      filter: "",
      initialized: true,
      selectedRowKey: undefined as number | undefined,
    },
  };
  return {
    tab,
    t: (key: string) => key,
    writeTextToClipboard: vi.fn().mockResolvedValue(true),
    updateTab: vi.fn(
      (_: string, patch: { dataExplorer?: typeof tab.dataExplorer }) => {
        if (patch.dataExplorer) tab.dataExplorer = patch.dataExplorer;
      }
    ),
  };
});

vi.mock("react-i18next", () => ({
  initReactI18next: { type: "3rdParty", init: () => undefined },
  useTranslation: () => ({ t: mocks.t }),
}));

vi.mock("react-resizable-panels", () => ({
  Group: ({ children }: { children: React.ReactNode }) => <div>{children}</div>,
  Panel: ({ children }: { children: React.ReactNode }) => <div>{children}</div>,
  Separator: () => <hr />,
}));

vi.mock("@/components/AdvancedSearch", () => ({
  AdvancedSearch: ({
    onParamsChange,
    placeholder,
  }: {
    onParamsChange: (params: { query: string; scopes: [] }) => void;
    placeholder: string;
  }) => (
    <div>
      <span data-testid="search-placeholder">{placeholder}</span>
      <button
        type="button"
        onClick={() => onParamsChange({ query: "2", scopes: [] })}
      >
        search second row
      </button>
    </div>
  ),
}));

vi.mock("@/lib/clipboard", () => ({
  writeTextToClipboard: mocks.writeTextToClipboard,
}));

vi.mock("@/modules/schema-editor/resize", () => ({
  resizeHandleClass: () => "resize-handle",
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
    }),
  };
});

vi.mock("@/stores/app", () => {
  const state = { notify: vi.fn() };
  return {
    useAppStore: Object.assign(
      (selector: (value: typeof state) => unknown) => selector(state),
      { getState: () => state }
    ),
  };
});

vi.mock("./VirtualDataTable", () => ({
  VirtualDataTable: ({
    activeRowIndex,
    onRowClick,
  }: {
    activeRowIndex: number;
    onRowClick: (row: number) => void;
  }) => (
    <div>
      <span data-testid="active-row">{activeRowIndex}</span>
      <button type="button" onClick={() => onRowClick(1)}>
        second row
      </button>
    </div>
  ),
}));

vi.mock("./DetailPanel", async () => {
  const context = await import("./context");
  return {
    DetailPanel: () => {
      const { detail } = context.useSQLResultViewContext();
      return (
        <div data-testid="detail">
          {detail ? `${detail.row}:${detail.view}` : "empty"}
        </div>
      );
    },
  };
});

vi.mock("./ResultStatusBar", () => ({
  formatQueryTime: () => "25 ms",
  ResultStatusBar: ({
    statement,
    queryTime,
  }: {
    statement: string;
    queryTime: string;
  }) => <div data-testid="status-bar">{`${statement} ${queryTime}`}</div>,
}));

const columns: ResultTableColumn[] = [
  { id: "id", name: "id", columnType: "TEXT" },
];
const rows: ResultTableRow[] = [0, 1].map((key) => ({
  key,
  item: create(QueryRowSchema, {
    values: [
      create(RowValueSchema, {
        kind: { case: "stringValue", value: String(key + 1) },
      }),
    ],
  }),
}));
const database = {
  name: "instances/i/databases/db",
  instanceResource: { engine: Engine.COSMOSDB },
} as Database;
const result = create(QueryResultSchema, {
  statement: "SELECT * FROM c",
  columnNames: ["document"],
  columnTypeNames: ["JSON"],
  rows: rows.map((row) => row.item),
});

function DetailProbe() {
  const { detail } = useSQLResultViewContext();
  return <span>{detail?.row}</span>;
}

beforeEach(() => {
  vi.clearAllMocks();
  mocks.tab.dataExplorer.selectedRowKey = undefined;
});

describe("DataExplorerResultView", () => {
  test("selects the first row, then follows a clicked row", async () => {
    render(
      <SQLResultViewProvider
        engine={Engine.COSMOSDB}
        rows={rows}
        columns={columns}
      >
        <DataExplorerResultView
          rows={rows}
          columns={columns}
          database={database}
          result={result}
          onToggleSort={() => undefined}
        />
        <DetailProbe />
      </SQLResultViewProvider>
    );

    await waitFor(() => {
      expect(screen.getByTestId("detail")).toHaveTextContent("0:row");
    });
    expect(screen.getByTestId("status-bar")).toHaveTextContent(
      "SELECT * FROM c 25 ms"
    );
    expect(screen.getByTestId("search-placeholder")).toHaveTextContent(
      "common.search-results"
    );
    expect(
      screen.getByRole("button", { name: "common.copy-all" })
    ).toBeVisible();
    fireEvent.click(screen.getByRole("button", { name: "common.copy-all" }));
    await waitFor(() => {
      expect(mocks.writeTextToClipboard).toHaveBeenCalledWith(
        "index\tid\n0\t1\n1\t2"
      );
    });

    fireEvent.click(screen.getByRole("button", { name: "second row" }));

    await waitFor(() => {
      expect(screen.getByTestId("detail")).toHaveTextContent("1:row");
      expect(screen.getByTestId("active-row")).toHaveTextContent("1");
    });
    expect(mocks.tab.dataExplorer.selectedRowKey).toBe(1);
  });

  test("selects the active search result in the table and detail panel", async () => {
    render(
      <SQLResultViewProvider
        engine={Engine.COSMOSDB}
        rows={rows}
        columns={columns}
      >
        <DataExplorerResultView
          rows={rows}
          columns={columns}
          database={database}
          result={result}
          onToggleSort={() => undefined}
        />
      </SQLResultViewProvider>
    );

    await waitFor(() => {
      expect(screen.getByTestId("detail")).toHaveTextContent("0:row");
    });

    fireEvent.click(
      screen.getByRole("button", { name: "search second row" })
    );

    await waitFor(() => {
      expect(screen.getByTestId("detail")).toHaveTextContent("1:row");
      expect(screen.getByTestId("active-row")).toHaveTextContent("1");
    });
  });

  test("preserves the selected row when sorting changes its index", async () => {
    const renderView = (viewRows: ResultTableRow[]) => (
      <SQLResultViewProvider
        engine={Engine.COSMOSDB}
        rows={viewRows}
        columns={columns}
      >
        <DataExplorerResultView
          rows={viewRows}
          columns={columns}
          database={database}
          result={result}
          onToggleSort={() => undefined}
        />
      </SQLResultViewProvider>
    );
    const { rerender } = render(renderView(rows));

    fireEvent.click(screen.getByRole("button", { name: "second row" }));
    await waitFor(() => {
      expect(screen.getByTestId("detail")).toHaveTextContent("1:row");
      expect(mocks.tab.dataExplorer.selectedRowKey).toBe(1);
    });

    rerender(renderView([rows[1], rows[0]]));

    await waitFor(() => {
      expect(screen.getByTestId("detail")).toHaveTextContent("0:row");
      expect(mocks.tab.dataExplorer.selectedRowKey).toBe(1);
    });
  });
});

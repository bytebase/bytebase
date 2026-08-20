import { create } from "@bufbuild/protobuf";
import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { useEffect } from "react";
import { beforeEach, describe, expect, test, vi } from "vitest";
import type { SQLEditorQueryParams } from "@/types";
import { Engine } from "@/types/proto-es/v1/common_pb";
import type { Database } from "@/types/proto-es/v1/database_service_pb";
import {
  QueryResultSchema,
  QueryRowSchema,
  RowValueSchema,
} from "@/types/proto-es/v1/sql_service_pb";
import { SingleResultView } from "./SingleResultView";

const {
  deltaDecorations,
  findMatches,
  monacoEditor,
  revealRangeInCenter,
  translate,
  writeTextToClipboard,
} = vi.hoisted(() => {
  const ranges = [
    {
      startLineNumber: 2,
      startColumn: 6,
      endLineNumber: 2,
      endColumn: 9,
    },
    {
      startLineNumber: 4,
      startColumn: 6,
      endLineNumber: 4,
      endColumn: 9,
    },
  ];
  const findMatches = vi.fn((query: string) =>
    query ? ranges.map((range) => ({ range })) : []
  );
  const deltaDecorations = vi.fn(() => ["match-1", "match-2"]);
  const revealRangeInCenter = vi.fn();
  return {
    deltaDecorations,
    findMatches,
    monacoEditor: {
      getModel: () => ({ deltaDecorations, findMatches }),
      getLayoutInfo: () => ({
        contentLeft: 48,
        contentWidth: 600,
        glyphMarginLeft: 0,
        glyphMarginWidth: 32,
      }),
      getScrollTop: () => 0,
      getTopForLineNumber: (lineNumber: number) => (lineNumber - 1) * 24,
      onDidChangeCursorPosition: vi.fn(() => ({ dispose: vi.fn() })),
      onDidScrollChange: vi.fn(() => ({ dispose: vi.fn() })),
      onMouseMove: vi.fn(() => ({ dispose: vi.fn() })),
      revealRangeInCenter,
    },
    revealRangeInCenter,
    translate: (key: string) => key,
    writeTextToClipboard: vi.fn(),
  };
});

vi.mock("react-i18next", () => ({
  initReactI18next: {
    init: vi.fn(),
    type: "3rdParty",
  },
  useTranslation: () => ({ t: translate }),
}));

vi.mock("@/components/AdvancedSearch", () => ({
  AdvancedSearch: () => <div data-testid="result-search" />,
}));

vi.mock("@/components/DataExportButton", () => ({
  DataExportButton: () => null,
}));

vi.mock("@/components/DatabaseTargetDisplay", () => ({
  DatabaseTargetDisplay: () => null,
}));

vi.mock("@/components/monaco/MonacoEditor", () => ({
  MonacoEditor: ({
    content,
    onReady,
  }: {
    content: string;
    onReady?: (monaco: unknown, editor: unknown) => void;
  }) => {
    useEffect(() => {
      onReady?.({}, monacoEditor);
    }, [onReady]);
    return <pre data-testid="json-editor">{content}</pre>;
  },
}));

vi.mock("@/hooks/useExecuteSQL", () => ({
  useExecuteSQL: () => ({ runQuery: vi.fn() }),
}));

vi.mock("@/lib/clipboard", () => ({
  writeTextToClipboard,
}));

vi.mock("@/components/ui/alert", () => ({
  Alert: ({ children }: { children: React.ReactNode }) => <div>{children}</div>,
}));

vi.mock("@/components/ui/button", () => ({
  Button: ({
    children,
    onClick,
    disabled,
    "aria-label": ariaLabel,
  }: {
    children: React.ReactNode;
    onClick?: () => void;
    disabled?: boolean;
    "aria-label"?: string;
  }) => (
    <button
      type="button"
      aria-label={ariaLabel}
      disabled={disabled}
      onClick={onClick}
    >
      {children}
    </button>
  ),
}));

vi.mock("@/components/ui/dropdown-menu", () => ({
  DropdownMenu: ({ children }: { children: React.ReactNode }) => <>{children}</>,
  DropdownMenuContent: ({ children }: { children: React.ReactNode }) => (
    <>{children}</>
  ),
  DropdownMenuItem: ({
    children,
    onClick,
  }: {
    children: React.ReactNode;
    onClick?: () => void;
  }) => (
    <button type="button" onClick={onClick}>
      {children}
    </button>
  ),
  DropdownMenuTrigger: ({ render }: { render: React.ReactNode }) => render,
}));

vi.mock("@/components/ui/switch", () => ({
  Switch: ({
    checked,
    onCheckedChange,
  }: {
    checked: boolean;
    onCheckedChange: (checked: boolean) => void;
  }) => (
    <input
      type="checkbox"
      checked={checked}
      onChange={(event) => onCheckedChange(event.target.checked)}
    />
  ),
}));

vi.mock("@/components/ui/tooltip", () => ({
  Tooltip: ({ children }: { children: React.ReactNode }) => <>{children}</>,
}));

vi.mock("@/modules/sql-editor/hooks/useSQLEditorState", () => ({
  useSQLEditorQueryDataPolicy: () => ({ maximumResultRows: 1000 }),
}));

vi.mock("@/modules/sql-editor/store/editor", () => ({
  useSQLEditorEditorState: (
    selector: (state: { project: string; resultRowsLimit: number }) => unknown
  ) => selector({ project: "projects/prod", resultRowsLimit: 100 }),
}));

vi.mock("@/modules/sql-editor/store/tab", () => ({
  useSQLEditorTabState: (
    selector: (state: {
      currentTabId: string;
      tabsById: Map<string, { mode: string }>;
    }) => unknown
  ) =>
    selector({
      currentTabId: "tab-1",
      tabsById: new Map([["tab-1", { mode: "READ_ONLY" }]]),
    }),
}));

vi.mock("@/stores/app", () => ({
  useAppStore: {
    getState: () => ({ notify: vi.fn() }),
  },
}));

vi.mock("@/utils/pev2", () => ({
  createExplainToken: vi.fn(),
}));

vi.mock("@/utils/util", () => ({
  isNullOrUndefined: (value: unknown) => value === null || value === undefined,
}));

vi.mock("@/utils/v1/database", () => ({
  getInstanceResource: (database: Database) => database.instanceResource,
}));

vi.mock("@/utils/v1/sql", () => ({
  compareQueryRowValues: () => 0,
  extractSQLRowValuePlain: (value?: {
    kind?: { case: string; value?: unknown };
  }) => value?.kind?.value,
}));

vi.mock("./DetailPanel", () => ({
  DetailPanel: ({ result }: { result: { rows: unknown[] } }) => (
    <div
      data-testid="detail-panel"
      data-result-row-count={result.rows.length}
    />
  ),
}));

vi.mock("./context", () => ({
  SQLResultViewProvider: ({ children }: { children: React.ReactNode }) => (
    <>{children}</>
  ),
  useSelectionContext: () => ({
    canCopyAsInsert: false,
    copy: vi.fn(),
  }),
}));

vi.mock("./EmptyView", () => ({
  EmptyView: () => <div data-testid="empty-view" />,
}));

vi.mock("./ErrorView", () => ({
  ErrorView: () => <div data-testid="error-view" />,
}));

vi.mock("./ResultStatusBar", () => ({
  ResultStatusBar: () => <div data-testid="result-status" />,
}));

vi.mock("./SelectionCopyTooltips", () => ({
  SelectionCopyTooltips: () => <div data-testid="selection-tooltips" />,
}));

vi.mock("./VirtualDataBlock", () => ({
  VirtualDataBlock: () => <div data-testid="result-block" />,
}));

vi.mock("./VirtualDataTable", () => ({
  VirtualDataTable: ({
    rows,
    columns,
  }: {
    rows: Array<{
      item: {
        values: Array<{ kind: { case: string; value?: unknown } }>;
      };
    }>;
    columns: Array<{ name: string }>;
  }) => (
    <div data-testid="result-table">
      {columns.map((column) => column.name).join(",")}
      {rows
        .flatMap((row) => row.item.values)
        .map((value) => String(value.kind.value ?? ""))
        .join(",")}
    </div>
  ),
}));

const params: SQLEditorQueryParams = {
  connection: {
    database: "instances/prod/databases/main",
    instance: "instances/prod",
  },
  engine: Engine.COSMOSDB,
  explain: false,
  selection: null,
  statement: "SELECT * FROM c",
};

const databaseForEngine = (engine: Engine) =>
  ({
    name: "instances/prod/databases/main",
    project: "projects/prod",
    instanceResource: {
      name: "instances/prod",
      engine,
    },
  }) as Database;

const documentResult = () =>
  create(QueryResultSchema, {
    columnNames: ["result"],
    columnTypeNames: ["TEXT"],
    rows: [
      create(QueryRowSchema, {
        values: [
          create(RowValueSchema, {
            kind: {
              case: "stringValue",
              value:
                '{"id":"one","profile":{"tags":["a","b"]},"ssn":"******"}',
            },
          }),
        ],
      }),
    ],
  });

describe("SingleResultView document view", () => {
  beforeEach(() => {
    localStorage.clear();
    deltaDecorations.mockClear();
    findMatches.mockClear();
    revealRangeInCenter.mockClear();
    writeTextToClipboard.mockClear();
  });

  test.each([Engine.COSMOSDB, Engine.MONGODB])(
    "defaults %s results to native JSON and switches to the table",
    (engine) => {
      render(
        <SingleResultView
          disallowCopyingData
          params={{ ...params, engine }}
          database={databaseForEngine(engine)}
          result={documentResult()}
          showExport={false}
        />
      );

      expect(screen.getByText("sql-editor.table-view")).toBeInTheDocument();
      expect(screen.getByText("sql-editor.json-view")).toBeInTheDocument();
      expect(
        screen.getByRole("radiogroup", {
          name: "sql-editor.result-view-mode",
        })
      ).toHaveClass("h-7");
      expect(
        screen.getByText("sql-editor.json-view").closest("label")
      ).toHaveClass("bg-accent/10", "text-accent");
      expect(
        screen.getByText("sql-editor.json-view").closest("label")
      ).not.toHaveClass("bg-accent", "text-accent-text");
      expect(document.querySelector(".lucide-table-2")).toBeInTheDocument();
      expect(document.querySelector(".lucide-braces")).toBeInTheDocument();
      expect(screen.getByText("sql-editor.table-view")).toHaveClass("sr-only");
      expect(screen.getByText("sql-editor.json-view")).toHaveClass("sr-only");
      expect(
        screen.getByRole("region", { name: "sql-editor.json-view" })
      ).toHaveTextContent('"tags": [');
      expect(screen.getByRole("region")).toHaveTextContent('"ssn": "******"');
      expect(screen.queryByTestId("result-table")).not.toBeInTheDocument();
      expect(screen.queryByTestId("result-search")).not.toBeInTheDocument();
      expect(
        screen.queryByText("sql-editor.vertical-display")
      ).not.toBeInTheDocument();
      expect(screen.queryByText("common.copy-all")).not.toBeInTheDocument();
      expect(screen.queryByTestId("selection-tooltips")).not.toBeInTheDocument();

      fireEvent.click(screen.getByText("sql-editor.table-view"));
      expect(screen.getByTestId("result-table")).toHaveTextContent(
        "id,profile,ssn"
      );
      expect(screen.getByTestId("result-table")).toHaveTextContent("******");
      expect(screen.getByTestId("detail-panel")).toHaveAttribute(
        "data-result-row-count",
        "1"
      );
    }
  );

  test("does not persist the document view selection", () => {
    const props = {
      disallowCopyingData: true,
      params,
      database: databaseForEngine(Engine.COSMOSDB),
      result: documentResult(),
      showExport: false,
    };
    const view = render(<SingleResultView {...props} />);

    fireEvent.click(screen.getByText("sql-editor.table-view"));
    expect(screen.getByTestId("result-table")).toBeInTheDocument();

    view.unmount();
    render(<SingleResultView {...props} />);

    expect(
      screen.getByRole("region", { name: "sql-editor.json-view" })
    ).toBeInTheDocument();
  });

  test("shows an empty document result in JSON view", () => {
    render(
      <SingleResultView
        disallowCopyingData
        params={params}
        database={databaseForEngine(Engine.COSMOSDB)}
        result={
          create(QueryResultSchema, {
            columnNames: ["result"],
            columnTypeNames: ["JSON"],
          })
        }
        showExport={false}
      />
    );

    expect(
      screen.getByRole("region", { name: "sql-editor.json-view" })
    ).toHaveTextContent("[]");
  });

  test("does not offer JSON mode for relational or malformed results", () => {
    const relationalResult = create(QueryResultSchema, {
      columnNames: ["name"],
      columnTypeNames: ["TEXT"],
      rows: [
        create(QueryRowSchema, {
          values: [
            create(RowValueSchema, {
              kind: { case: "stringValue", value: "Ada" },
            }),
          ],
        }),
      ],
    });
    const { rerender } = render(
      <SingleResultView
        disallowCopyingData
        params={{ ...params, engine: Engine.POSTGRES }}
        database={databaseForEngine(Engine.POSTGRES)}
        result={relationalResult}
        showExport={false}
      />
    );

    expect(screen.queryByText("sql-editor.json-view")).not.toBeInTheDocument();

    rerender(
      <SingleResultView
        disallowCopyingData
        params={params}
        database={databaseForEngine(Engine.COSMOSDB)}
        result={
          create(QueryResultSchema, {
            columnNames: ["result"],
            rows: [
              create(QueryRowSchema, {
                values: [
                  create(RowValueSchema, {
                    kind: { case: "stringValue", value: "{" },
                  }),
                ],
              }),
            ],
          })
        }
        showExport={false}
      />
    );

    expect(screen.queryByText("sql-editor.json-view")).not.toBeInTheDocument();
  });

  test("copies JSON", () => {
    render(
      <SingleResultView
        disallowCopyingData={false}
        params={params}
        database={databaseForEngine(Engine.COSMOSDB)}
        result={documentResult()}
        showExport={false}
      />
    );

    fireEvent.click(screen.getByText("common.copy-all"));

    expect(writeTextToClipboard).toHaveBeenCalledWith(
      expect.stringContaining('"ssn": "******"')
    );
  });

  test("searches JSON matches and moves the active match", async () => {
    render(
      <SingleResultView
        disallowCopyingData
        params={params}
        database={databaseForEngine(Engine.COSMOSDB)}
        result={documentResult()}
        showExport={false}
      />
    );

    fireEvent.click(screen.getByText("sql-editor.json-view"));
    const search = screen.getByRole("textbox", { name: "common.search" });
    expect(search.parentElement).toHaveClass("flex-1");
    fireEvent.change(search, { target: { value: "ssn" } });

    await waitFor(() => {
      expect(screen.getByText("1 / 2")).toBeInTheDocument();
    });
    expect(findMatches).toHaveBeenLastCalledWith(
      "ssn",
      false,
      false,
      false,
      null,
      false
    );
    expect(revealRangeInCenter).toHaveBeenLastCalledWith(
      expect.objectContaining({ startLineNumber: 2 })
    );

    fireEvent.click(
      screen.getByRole("button", {
        name: "sql-editor.result-detail.next-match",
      })
    );

    await waitFor(() => {
      expect(screen.getByText("2 / 2")).toBeInTheDocument();
    });
    expect(revealRangeInCenter).toHaveBeenLastCalledWith(
      expect.objectContaining({ startLineNumber: 4 })
    );
  });

  test("keeps the existing Elasticsearch table toggle", () => {
    const result = create(QueryResultSchema, {
      columnNames: ["hits"],
      columnTypeNames: ["JSON"],
      rows: [
        create(QueryRowSchema, {
          values: [
            create(RowValueSchema, {
              kind: {
                case: "stringValue",
                value:
                  '{"hits":[{"_id":"one","_score":1,"_source":{"name":"Ada"}}]}',
              },
            }),
          ],
        }),
      ],
    });

    render(
      <SingleResultView
        disallowCopyingData
        params={{ ...params, engine: Engine.ELASTICSEARCH }}
        database={databaseForEngine(Engine.ELASTICSEARCH)}
        result={result}
        showExport={false}
      />
    );

    expect(screen.queryByText("sql-editor.json-view")).not.toBeInTheDocument();
    expect(screen.getByTestId("result-table")).toHaveTextContent(
      "_id,_score,name"
    );

    const [tableViewToggle] = screen.getAllByRole("checkbox");
    fireEvent.click(tableViewToggle);
    expect(screen.getByTestId("result-table")).toHaveTextContent("hits");
  });
});

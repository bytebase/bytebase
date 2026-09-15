import { create } from "@bufbuild/protobuf";
import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { useEffect } from "react";
import { beforeEach, describe, expect, test, vi } from "vitest";
import type { SQLEditorQueryParams } from "@/types";
import { Engine } from "@/types/proto-es/v1/common_pb";
import type { Database } from "@/types/proto-es/v1/database_service_pb";
import {
  QueryOption_ExplainFormat,
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

const { createExplainToken, notify, runQuery } = vi.hoisted(() => ({
  createExplainToken: vi.fn(() => "explain-token"),
  notify: vi.fn(),
  runQuery: vi.fn(),
}));

vi.mock("@/hooks/useExecuteSQL", () => ({
  useExecuteSQL: () => ({ runQuery }),
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
    getState: () => ({ notify }),
  },
}));

vi.mock("@/utils/explainToken", async (importOriginal) => ({
  ...(await importOriginal<typeof import("@/utils/explainToken")>()),
  createExplainToken,
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
  formatQueryTime: () => "-",
  ResultStatusBar: ({
    showVisualizeButton,
    onVisualizeExplain,
  }: {
    showVisualizeButton?: boolean;
    onVisualizeExplain?: () => void;
  }) => (
    <div data-testid="result-status">
      {showVisualizeButton ? (
        <button type="button" onClick={onVisualizeExplain}>
          visualize-explain
        </button>
      ) : null}
    </div>
  ),
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

describe("SingleResultView explain visualizer", () => {
  beforeEach(() => {
    createExplainToken.mockClear();
    notify.mockClear();
    runQuery.mockReset();
  });

  test("PostgreSQL asks for the JSON plan instead of reusing the text plan", async () => {
    const openSpy = vi.spyOn(window, "open").mockReturnValue(null);
    const planJSON = '[{"Plan": {"Node Type": "Seq Scan"}}]';
    runQuery.mockImplementation(
      async (
        _database: unknown,
        context: {
          resultSet?: { results: unknown[] };
        }
      ) => {
        context.resultSet = {
          results: [
            create(QueryResultSchema, {
              statement: "SELECT 1",
              rows: [
                create(QueryRowSchema, {
                  values: [
                    create(RowValueSchema, {
                      kind: { case: "stringValue", value: planJSON },
                    }),
                  ],
                }),
              ],
            }),
          ],
        };
      }
    );

    render(
      <SingleResultView
        disallowCopyingData={false}
        params={{ ...params, engine: Engine.POSTGRES, explain: true }}
        database={databaseForEngine(Engine.POSTGRES)}
        result={create(QueryResultSchema, {
          columnNames: ["QUERY PLAN"],
          columnTypeNames: ["TEXT"],
          statement: "SELECT 1",
          rows: [
            create(QueryRowSchema, {
              values: [
                create(RowValueSchema, {
                  kind: { case: "stringValue", value: "Seq Scan on t" },
                }),
              ],
            }),
          ],
        })}
        showExport={false}
      />
    );

    fireEvent.click(screen.getByText("visualize-explain"));

    await waitFor(() => expect(openSpy).toHaveBeenCalled());
    expect(runQuery).toHaveBeenCalledTimes(1);
    expect(
      runQuery.mock.calls[0][1].params.queryOption?.explainFormat
    ).toBe(QueryOption_ExplainFormat.JSON);
    expect(createExplainToken).toHaveBeenCalledWith({
      statement: "SELECT 1",
      explain: planJSON,
      engine: Engine.POSTGRES,
    });
    expect(openSpy).toHaveBeenCalledWith(
      "/explain-visualizer.html?token=explain-token",
      "_blank"
    );
    openSpy.mockRestore();
  });

  test("re-runs for the tab the user clicked, not the first result", async () => {
    const openSpy = vi.spyOn(window, "open").mockReturnValue(null);
    runQuery.mockImplementation(
      async (_database: unknown, context: { resultSet?: { results: unknown[] } }) => {
        const plan = (name: string) =>
          create(QueryResultSchema, {
            statement: name,
            rows: [
              create(QueryRowSchema, {
                values: [
                  create(RowValueSchema, {
                    kind: { case: "stringValue", value: `plan of ${name}` },
                  }),
                ],
              }),
            ],
          });
        context.resultSet = { results: [plan("first"), plan("second")] };
      }
    );

    render(
      <SingleResultView
        disallowCopyingData={false}
        params={{ ...params, engine: Engine.POSTGRES, explain: true }}
        database={databaseForEngine(Engine.POSTGRES)}
        result={create(QueryResultSchema, {
          columnNames: ["QUERY PLAN"],
          columnTypeNames: ["TEXT"],
          statement: "second",
          rows: [
            create(QueryRowSchema, {
              values: [
                create(RowValueSchema, {
                  kind: { case: "stringValue", value: "Seq Scan on t" },
                }),
              ],
            }),
          ],
        })}
        resultIndex={1}
        showExport={false}
      />
    );

    fireEvent.click(screen.getByText("visualize-explain"));

    await waitFor(() => expect(createExplainToken).toHaveBeenCalled());
    expect(createExplainToken).toHaveBeenCalledWith({
      statement: "second",
      explain: "plan of second",
      engine: Engine.POSTGRES,
    });
    openSpy.mockRestore();
  });

  test("reports a blocked pop-up instead of doing nothing", async () => {
    const openSpy = vi.spyOn(window, "open").mockReturnValue(null);
    runQuery.mockImplementation(
      async (_database: unknown, context: { resultSet?: { results: unknown[] } }) => {
        context.resultSet = {
          results: [
            create(QueryResultSchema, {
              statement: "SELECT 1",
              rows: [
                create(QueryRowSchema, {
                  values: [
                    create(RowValueSchema, {
                      kind: { case: "stringValue", value: "[]" },
                    }),
                  ],
                }),
              ],
            }),
          ],
        };
      }
    );

    render(
      <SingleResultView
        disallowCopyingData={false}
        params={{ ...params, engine: Engine.POSTGRES, explain: true }}
        database={databaseForEngine(Engine.POSTGRES)}
        result={create(QueryResultSchema, {
          columnNames: ["QUERY PLAN"],
          columnTypeNames: ["TEXT"],
          statement: "SELECT 1",
          rows: [
            create(QueryRowSchema, {
              values: [
                create(RowValueSchema, {
                  kind: { case: "stringValue", value: "Seq Scan on t" },
                }),
              ],
            }),
          ],
        })}
        showExport={false}
      />
    );

    fireEvent.click(screen.getByText("visualize-explain"));

    await waitFor(() => expect(notify).toHaveBeenCalled());
    expect(notify.mock.calls[0][0].title).toBe(
      "sql-editor.visualize-explain-blocked"
    );
    openSpy.mockRestore();
  });

  test.each([[Engine.MSSQL, QueryOption_ExplainFormat.XML, "<ShowPlanXML/>"]])(
    "asks engine %s for the plan in the format its visualizer reads",
    async (engine, explainFormat, plan) => {
      const openSpy = vi
        .spyOn(window, "open")
        .mockReturnValue({} as Window);
      runQuery.mockImplementation(
        async (
          _database: unknown,
          context: { resultSet?: { results: unknown[] } }
        ) => {
          context.resultSet = {
            results: [
              create(QueryResultSchema, {
                statement: "SELECT 1",
                rows: [
                  create(QueryRowSchema, {
                    values: [
                      create(RowValueSchema, {
                        kind: { case: "stringValue", value: plan },
                      }),
                    ],
                  }),
                ],
              }),
            ],
          };
        }
      );

      render(
        <SingleResultView
          disallowCopyingData={false}
          params={{ ...params, engine, explain: true }}
          database={databaseForEngine(engine)}
          result={create(QueryResultSchema, {
            columnNames: ["QUERY PLAN"],
            statement: "SELECT 1",
            rows: [
              create(QueryRowSchema, {
                values: [
                  create(RowValueSchema, {
                    kind: { case: "stringValue", value: "a readable plan" },
                  }),
                ],
              }),
            ],
          })}
          showExport={false}
        />
      );

      fireEvent.click(screen.getByText("visualize-explain"));

      await waitFor(() => expect(openSpy).toHaveBeenCalled());
      expect(
        runQuery.mock.calls[0][1].params.queryOption?.explainFormat
      ).toBe(explainFormat);
      expect(createExplainToken).toHaveBeenCalledWith({
        statement: "SELECT 1",
        explain: plan,
        engine,
      });
      expect(notify).not.toHaveBeenCalled();
      openSpy.mockRestore();
    }
  );

  test("Spanner hands over the plan already in the result", async () => {
    const openSpy = vi.spyOn(window, "open").mockReturnValue({} as Window);
    const plan = '{"planNodes":[]}';

    render(
      <SingleResultView
        disallowCopyingData={false}
        params={{ ...params, engine: Engine.SPANNER, explain: true }}
        database={databaseForEngine(Engine.SPANNER)}
        result={create(QueryResultSchema, {
          columnNames: ["QUERY PLAN"],
          columnTypeNames: ["JSON"],
          statement: "SELECT 1",
          rows: [
            create(QueryRowSchema, {
              values: [
                create(RowValueSchema, {
                  kind: { case: "stringValue", value: plan },
                }),
              ],
            }),
          ],
        })}
        showExport={false}
      />
    );

    fireEvent.click(screen.getByText("visualize-explain"));

    await waitFor(() => expect(openSpy).toHaveBeenCalled());
    expect(runQuery).not.toHaveBeenCalled();
    expect(createExplainToken).toHaveBeenCalledWith({
      statement: "SELECT 1",
      explain: plan,
      engine: Engine.SPANNER,
    });
    openSpy.mockRestore();
  });

  test("offers no visualizer for an engine it cannot draw", () => {
    render(
      <SingleResultView
        disallowCopyingData={false}
        params={{ ...params, engine: Engine.MYSQL, explain: true }}
        database={databaseForEngine(Engine.MYSQL)}
        result={create(QueryResultSchema, {
          columnNames: ["EXPLAIN"],
          statement: "SELECT 1",
          rows: [
            create(QueryRowSchema, {
              values: [
                create(RowValueSchema, {
                  kind: { case: "stringValue", value: "-> Table scan on t" },
                }),
              ],
            }),
          ],
        })}
        showExport={false}
      />
    );

    expect(screen.queryByText("visualize-explain")).toBeNull();
  });
});

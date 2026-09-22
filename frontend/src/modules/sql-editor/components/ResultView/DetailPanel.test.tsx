import { create } from "@bufbuild/protobuf";
import { NullValue } from "@bufbuild/protobuf/wkt";
import type { ReactElement } from "react";
import { act, useEffect } from "react";
import { createRoot } from "react-dom/client";
import { afterEach, beforeEach, describe, expect, test, vi } from "vitest";
import { Engine } from "@/types/proto-es/v1/common_pb";
import type { Database } from "@/types/proto-es/v1/database_service_pb";
import {
  type QueryResult,
  QueryResultSchema,
  QueryRowSchema,
  type RowValue,
  RowValueSchema,
} from "@/types/proto-es/v1/sql_service_pb";
import {
  type ResultViewDetail,
  SQLResultViewProvider,
  useSelectionContext,
  useSQLResultViewContext,
} from "./context";
import { DetailPanel } from "./DetailPanel";
import type { ResultTableColumn, ResultTableRow } from "./types";

(
  globalThis as { IS_REACT_ACT_ENVIRONMENT?: boolean }
).IS_REACT_ACT_ENVIRONMENT = true;

vi.mock("react-i18next", () => ({
  initReactI18next: {
    init: vi.fn(),
    type: "3rdParty",
  },
  useTranslation: () => ({ t: (key: string) => key }),
}));

vi.mock("@/stores/app", () => {
  const state = () => ({
    notify: vi.fn(),
  });
  return {
    useAppStore: Object.assign(
      (selector: (s: ReturnType<typeof state>) => unknown) => selector(state()),
      { getState: state }
    ),
  };
});

vi.mock("@/components/ui/tooltip", () => ({
  Tooltip: ({ children }: { children: React.ReactNode }) => <>{children}</>,
}));

vi.mock("@/modules/sql-editor/components/MaskingReasonPopover", () => ({
  MaskingReasonPopover: () => null,
}));

vi.mock("@/utils/v1/database", () => ({
  getInstanceResource: (database: Database) => database.instanceResource,
}));

const textValue = (value: string): RowValue =>
  create(RowValueSchema, {
    kind: { case: "stringValue", value },
  });

const columns: ResultTableColumn[] = [
  {
    id: "payload",
    name: "payload",
    columnType: "json",
  },
];

const detailContent = '{"customer":"longbridge","allowed":true}';
const rows: ResultTableRow[] = [
  {
    key: 0,
    item: create(QueryRowSchema, {
      values: [textValue(detailContent)],
    }),
  },
];

const databaseForEngine = (engine: Engine) =>
  ({
    name: "instances/prod/databases/main",
    project: "projects/prod",
    instanceResource: {
      name: "instances/prod",
      engine,
    },
  }) as Database;

const resultForRows = (
  panelRows: ResultTableRow[],
  panelColumns: ResultTableColumn[]
) =>
  create(QueryResultSchema, {
    columnNames: panelColumns.map((column) => column.name),
    columnTypeNames: panelColumns.map((column) => column.columnType),
    rows: panelRows.map((row) => row.item),
  });

function OpenDetailOnMount({ view }: Pick<ResultViewDetail, "view">) {
  const { setDetail } = useSQLResultViewContext();
  useEffect(() => {
    setDetail({ row: 0, col: 0, view });
  }, [setDetail, view]);
  return null;
}

function SelectCellOnMount() {
  const { toggleSelectCell } = useSelectionContext();
  useEffect(() => {
    toggleSelectCell(0, 0);
  }, [toggleSelectCell]);
  return null;
}

function TestDetailPanel({
  disallowCopyingData = false,
  panelRows = rows,
  panelColumns = columns,
  engine = Engine.POSTGRES,
  sourceResult,
  detailView = "cell",
}: {
  disallowCopyingData?: boolean;
  panelRows?: ResultTableRow[];
  panelColumns?: ResultTableColumn[];
  engine?: Engine;
  sourceResult?: QueryResult;
  detailView?: ResultViewDetail["view"];
}) {
  const result = sourceResult ?? resultForRows(panelRows, panelColumns);
  return (
    <SQLResultViewProvider
      engine={engine}
      rows={panelRows}
      columns={panelColumns}
      disallowCopyingData={disallowCopyingData}
    >
      <OpenDetailOnMount view={detailView} />
      <DetailPanel
        rows={panelRows}
        columns={panelColumns}
        database={databaseForEngine(engine)}
        result={result}
      />
    </SQLResultViewProvider>
  );
}

function TestProviderWithGridSelection() {
  return (
    <SQLResultViewProvider
      engine={Engine.POSTGRES}
      rows={rows}
      columns={columns}
    >
      <SelectCellOnMount />
    </SQLResultViewProvider>
  );
}

const renderIntoContainer = (element: ReactElement) => {
  const container = document.createElement("div");
  const root = createRoot(container);
  document.body.appendChild(container);
  return {
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

const setInputValue = (input: HTMLInputElement, value: string) => {
  const valueSetter = Object.getOwnPropertyDescriptor(
    HTMLInputElement.prototype,
    "value"
  )?.set;
  valueSetter?.call(input, value);
  input.dispatchEvent(new Event("input", { bubbles: true }));
};

const flushAsyncRender = async () => {
  await act(async () => {
    await new Promise((resolve) => window.setTimeout(resolve, 0));
  });
};

const waitForAssertion = async (assertion: () => void) => {
  let lastError: unknown;
  const deadline = Date.now() + 5000;
  while (Date.now() < deadline) {
    await flushAsyncRender();
    try {
      assertion();
      return;
    } catch (err) {
      lastError = err;
      await new Promise((resolve) => window.setTimeout(resolve, 10));
    }
  }
  throw lastError;
};

const getDetailSearchControl = (input: HTMLInputElement) => {
  const searchControl = input.parentElement;
  expect(searchControl).toBeInstanceOf(HTMLDivElement);
  return searchControl as HTMLDivElement;
};

const getDetailContentRegion = (expectedText = "longbridge") => {
  const candidates = Array.from(document.body.querySelectorAll("div"));
  const contentRegion = candidates.find(
    (element) =>
      element.textContent?.includes(expectedText) &&
      element.className.includes("overflow-auto") &&
      element.className.includes("font-mono")
  );
  expect(contentRegion).toBeInstanceOf(HTMLDivElement);
  return contentRegion as HTMLDivElement;
};

beforeEach(() => {
  localStorage.clear();
  Object.defineProperty(navigator, "clipboard", {
    value: { writeText: vi.fn().mockResolvedValue(undefined) },
    configurable: true,
    writable: true,
  });
});

afterEach(() => {
  document.body.innerHTML = "";
});

describe("DetailPanel", () => {
  test("keeps allowed drawer text selectable", () => {
    const { render, unmount } = renderIntoContainer(<TestDetailPanel />);
    render();

    expect(document.body.textContent).toContain(
      "sql-editor.result-detail.cell"
    );
    expect(document.body.textContent).toContain("common.column:payload");
    expect(getDetailContentRegion().className).toContain("select-text");

    unmount();
  });

  test("keeps copy-restricted drawer text non-selectable", () => {
    const { render, unmount } = renderIntoContainer(
      <TestDetailPanel disallowCopyingData />
    );
    render();

    expect(getDetailContentRegion().className).toContain("select-none");

    unmount();
  });

  test("does not bubble a click that finishes native drawer text selection", () => {
    const onDocumentClick = vi.fn();
    document.addEventListener("click", onDocumentClick);
    const { render, unmount } = renderIntoContainer(<TestDetailPanel />);
    render();

    const getSelectionSpy = vi.spyOn(window, "getSelection").mockReturnValue({
      toString: () => "selected drawer text",
    } as Selection);

    act(() => {
      getDetailContentRegion().dispatchEvent(
        new MouseEvent("click", { bubbles: true })
      );
    });

    expect(onDocumentClick).not.toHaveBeenCalled();

    getSelectionSpy.mockRestore();
    document.removeEventListener("click", onDocumentClick);
    unmount();
  });

  test("lets native text selection handle copy even when grid selection exists", () => {
    const { render, unmount } = renderIntoContainer(
      <TestProviderWithGridSelection />
    );
    render();

    const getSelectionSpy = vi.spyOn(window, "getSelection").mockReturnValue({
      toString: () => "selected drawer text",
    } as Selection);

    const event = new KeyboardEvent("keydown", {
      key: "c",
      metaKey: true,
      bubbles: true,
      cancelable: true,
    });
    act(() => {
      document.dispatchEvent(event);
    });

    expect(event.defaultPrevented).toBe(false);
    expect(navigator.clipboard.writeText).not.toHaveBeenCalled();

    getSelectionSpy.mockRestore();
    unmount();
  });

  test("highlights plain text search matches and jumps between them", () => {
    const textRows: ResultTableRow[] = [
      {
        key: 0,
        item: create(QueryRowSchema, {
          values: [textValue("CREATE TABLE users;\nCREATE INDEX users_name;")],
        }),
      },
    ];
    const { render, unmount } = renderIntoContainer(
      <TestDetailPanel panelRows={textRows} />
    );
    render();

    const input = document.body.querySelector(
      "input[aria-label='sql-editor.result-detail.search']"
    ) as HTMLInputElement | null;
    expect(input).toBeInstanceOf(HTMLInputElement);

    act(() => {
      setInputValue(input!, "create");
    });

    const marks = Array.from(
      getDetailContentRegion("CREATE TABLE").querySelectorAll("mark")
    );
    expect(marks.map((mark) => mark.textContent)).toEqual(["CREATE", "CREATE"]);
    expect(document.body.textContent).toContain("1 / 2");
    expect(marks[0]?.className).toContain("bg-accent");
    const searchControl = getDetailSearchControl(input!);
    expect(searchControl).toBeInstanceOf(HTMLDivElement);
    expect(searchControl?.textContent).toContain("1 / 2");
    expect(searchControl?.querySelectorAll("button")).toHaveLength(3);
    expect(searchControl?.className).toContain("rounded-xs");
    expect(searchControl?.className).toContain("bg-transparent");
    expect(searchControl?.className).not.toContain("rounded-md");
    expect(searchControl?.className).not.toContain("shadow");
    expect(searchControl?.parentElement?.className).not.toContain("flex-1");

    act(() => {
      input!.dispatchEvent(
        new KeyboardEvent("keydown", {
          key: "Enter",
          bubbles: true,
          cancelable: true,
        })
      );
    });

    const nextMarks = Array.from(
      getDetailContentRegion("CREATE TABLE").querySelectorAll("mark")
    );
    expect(nextMarks[1]?.className).toContain("bg-accent");

    unmount();
  });

  test("scrolls the active match when the search target changes but match count is unchanged", () => {
    const scrollIntoView = vi.fn();
    const originalScrollIntoView = HTMLElement.prototype.scrollIntoView;
    HTMLElement.prototype.scrollIntoView = scrollIntoView;
    const textRows: ResultTableRow[] = [
      {
        key: 0,
        item: create(QueryRowSchema, {
          values: [textValue("CREATE TABLE users;\nCREATE INDEX users_name;")],
        }),
      },
    ];
    const { render, unmount } = renderIntoContainer(
      <TestDetailPanel panelRows={textRows} />
    );
    render();

    const input = document.body.querySelector(
      "input[aria-label='sql-editor.result-detail.search']"
    ) as HTMLInputElement | null;
    expect(input).toBeInstanceOf(HTMLInputElement);

    act(() => {
      setInputValue(input!, "create");
    });
    expect(scrollIntoView).toHaveBeenCalled();
    scrollIntoView.mockClear();

    act(() => {
      setInputValue(input!, "users");
    });

    expect(document.body.textContent).toContain("1 / 2");
    expect(scrollIntoView).toHaveBeenCalled();

    HTMLElement.prototype.scrollIntoView = originalScrollIntoView;
    unmount();
  });

  test("scrolls formatted JSON matches after the async highlighted render", async () => {
    localStorage.setItem("bb.sql-editor.detail-panel.format", "true");
    const scrolledContent: string[] = [];
    const originalScrollIntoView = HTMLElement.prototype.scrollIntoView;
    HTMLElement.prototype.scrollIntoView = function () {
      scrolledContent.push(
        this.closest(".overflow-auto")?.textContent?.trim() ?? ""
      );
    };
    const jsonRows: ResultTableRow[] = [
      {
        key: 0,
        item: create(QueryRowSchema, {
          values: [
            textValue('{"name":"alpha","first":"needle","second":"needle"}'),
          ],
        }),
      },
      {
        key: 1,
        item: create(QueryRowSchema, {
          values: [
            textValue('{"name":"beta","first":"needle","second":"needle"}'),
          ],
        }),
      },
    ];
    const { render, unmount } = renderIntoContainer(
      <TestDetailPanel panelRows={jsonRows} />
    );
    render();

    const input = document.body.querySelector(
      "input[aria-label='sql-editor.result-detail.search']"
    ) as HTMLInputElement | null;
    expect(input).toBeInstanceOf(HTMLInputElement);
    const searchControl = getDetailSearchControl(input!);

    act(() => {
      setInputValue(input!, "needle");
    });
    await waitForAssertion(() => {
      expect(searchControl.textContent).toContain("1 / 2");
    });
    scrolledContent.length = 0;

    act(() => {
      document.dispatchEvent(
        new KeyboardEvent("keydown", {
          key: "ArrowDown",
          bubbles: true,
          cancelable: true,
        })
      );
    });
    await waitForAssertion(() => {
      expect(scrolledContent.some((text) => text.includes("beta"))).toBe(true);
    });

    HTMLElement.prototype.scrollIntoView = originalScrollIntoView;
    unmount();
  });

  test("shows the original document as formatted JSON for a document table row", async () => {
    const flattenedRows: ResultTableRow[] = [
      {
        key: 1,
        item: create(QueryRowSchema, {
          values: [textValue("two")],
        }),
      },
    ];
    const sourceResult = create(QueryResultSchema, {
      columnNames: ["result"],
      columnTypeNames: ["JSON"],
      rows: [
        create(QueryRowSchema, {
          values: [
            textValue('{"id":"one","profile":{"tags":["a","b"]}}'),
          ],
        }),
        create(QueryRowSchema, {
          values: [
            textValue('{"id":"two","settings":{"theme":"dark"}}'),
          ],
        }),
      ],
    });
    const { render, unmount } = renderIntoContainer(
      <TestDetailPanel
        panelRows={flattenedRows}
        engine={Engine.COSMOSDB}
        sourceResult={sourceResult}
        detailView="row"
      />
    );
    render();

    await waitForAssertion(() => {
      const contentRegion = getDetailContentRegion("settings");
      expect(contentRegion.textContent).toContain("theme");
      expect(contentRegion.textContent).toContain("dark");
    });
    expect(document.body.querySelector(".lucide-braces")).toBeNull();

    unmount();
  });

  test("shows the clicked cell instead of the source document for cell detail", () => {
    const flattenedRows: ResultTableRow[] = [
      {
        key: 0,
        item: create(QueryRowSchema, {
          values: [textValue('{"cell":"value"}')],
        }),
      },
    ];
    const sourceResult = create(QueryResultSchema, {
      columnNames: ["result"],
      columnTypeNames: ["JSON"],
      rows: [
        create(QueryRowSchema, {
          values: [textValue('{"document":"source"}')],
        }),
      ],
    });
    const { render, unmount } = renderIntoContainer(
      <TestDetailPanel
        panelRows={flattenedRows}
        engine={Engine.COSMOSDB}
        sourceResult={sourceResult}
      />
    );
    render();

    const contentRegion = getDetailContentRegion("cell");
    expect(contentRegion.textContent).toContain('{"cell":"value"}');
    expect(contentRegion.textContent).not.toContain("source");

    unmount();
  });

  test("shows every column and value for a relational table row", () => {
    const rowColumns: ResultTableColumn[] = [
      { id: "id", name: "id", columnType: "TEXT" },
      { id: "name-1", name: "name", columnType: "TEXT" },
      { id: "name-2", name: "name", columnType: "TEXT" },
      { id: "environment", name: "environment", columnType: "TEXT" },
      { id: "optional", name: "optional", columnType: "TEXT" },
    ];
    const rowRows: ResultTableRow[] = [
      {
        key: 0,
        item: create(QueryRowSchema, {
          values: [
            textValue("1"),
            textValue("Ada"),
            textValue("Lovelace"),
            create(RowValueSchema, {
              kind: { case: "nullValue", value: NullValue.NULL_VALUE },
            }),
            create(RowValueSchema),
          ],
        }),
      },
    ];
    const { render, unmount } = renderIntoContainer(
      <TestDetailPanel
        panelRows={rowRows}
        panelColumns={rowColumns}
        detailView="row"
      />
    );
    render();

    const contentRegion = getDetailContentRegion("id: 1");
    expect(
      contentRegion.querySelector('[data-testid="row-data-block"]')
    ).toBeInstanceOf(HTMLDivElement);
    expect(contentRegion.textContent).toContain("name: Ada");
    expect(contentRegion.textContent).toContain("name: Lovelace");
    expect(contentRegion.textContent).toContain("environment: NULL");
    expect(contentRegion.textContent).toContain("optional: UNSET");
    const placeholders = Array.from(contentRegion.querySelectorAll("span"))
      .filter((element) => ["NULL", "UNSET"].includes(element.textContent ?? ""));
    expect(placeholders).toHaveLength(2);
    for (const placeholder of placeholders) {
      expect(placeholder).toHaveClass("text-control-placeholder", "italic");
    }

    const input = document.body.querySelector(
      "input[aria-label='sql-editor.result-detail.search']"
    ) as HTMLInputElement;
    act(() => {
      setInputValue(input, "name");
    });

    let marks = Array.from(contentRegion.querySelectorAll("mark"));
    expect(marks.map((mark) => mark.textContent)).toEqual(["name", "name"]);
    expect(getDetailSearchControl(input).textContent).toContain("1 / 2");

    act(() => {
      input.dispatchEvent(
        new KeyboardEvent("keydown", {
          key: "Enter",
          bubbles: true,
          cancelable: true,
        })
      );
    });
    marks = Array.from(contentRegion.querySelectorAll("mark"));
    expect(marks[1]?.dataset.detailSearchActiveMatch).toBe("true");

    unmount();
  });

  test("focuses detail search from the native find shortcut", () => {
    const { render, unmount } = renderIntoContainer(<TestDetailPanel />);
    render();

    const event = new KeyboardEvent("keydown", {
      key: "f",
      metaKey: true,
      bubbles: true,
      cancelable: true,
    });
    act(() => {
      document.dispatchEvent(event);
    });

    expect(event.defaultPrevented).toBe(true);
    expect(document.activeElement?.getAttribute("aria-label")).toBe(
      "sql-editor.result-detail.search"
    );

    unmount();
  });
});

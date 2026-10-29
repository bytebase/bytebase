import { act } from "react";
import { createRoot } from "react-dom/client";
import { describe, expect, test, vi } from "vitest";
import { ToolCallCard } from "./ToolCallCard";

(globalThis as { IS_REACT_ACT_ENVIRONMENT?: boolean }).IS_REACT_ACT_ENVIRONMENT =
  true;

vi.mock("react-i18next", () => ({
  initReactI18next: { type: "3rdParty", init: () => {} },
  useTranslation: () => ({ t: (key: string) => key }),
}));

vi.mock("@tanstack/react-virtual", () => ({
  useVirtualizer: (options: {
    count: number;
    estimateSize: (index: number) => number;
  }) => ({
    getVirtualItems: () =>
      Array.from({ length: options.count }, (_, index) => ({
        index,
        key: index,
        start: index * options.estimateSize(index),
      })),
    getTotalSize: () => options.count * options.estimateSize(0),
    measureElement: () => undefined,
    scrollToIndex: vi.fn(),
  }),
}));

class ResizeObserverStub implements ResizeObserver {
  observe() {}
  unobserve() {}
  disconnect() {}
}

globalThis.ResizeObserver = ResizeObserverStub;
Object.defineProperty(HTMLCanvasElement.prototype, "getContext", {
  configurable: true,
  value: () => ({ measureText: (text: string) => ({ width: text.length * 8 }) }),
});

const renderCard = (operationId: string, result: string) => {
  const container = document.createElement("div");
  const root = createRoot(container);
  act(() => {
    root.render(
      <ToolCallCard
        toolCall={{
          id: "query-1",
          name: "call_api",
          arguments: JSON.stringify({ operationId }),
        }}
        result={result}
      />
    );
  });
  return {
    container,
    unmount: () => act(() => root.unmount()),
  };
};

describe("ToolCallCard", () => {
  test("shows a canceled ask without claiming the user submitted an answer", () => {
    const container = document.createElement("div");
    const root = createRoot(container);
    act(() => {
      root.render(
        <ToolCallCard
          toolCall={{
            id: "ask-1",
            name: "ask_user",
            arguments: JSON.stringify({ kind: "input", prompt: "Which project?" }),
          }}
          result={JSON.stringify({ kind: "cancelled", reason: "user" })}
        />
      );
    });

    expect(container.textContent).toContain("agent.tool-request-canceled");
    expect(container.textContent).not.toContain("agent.tool-response-submitted");
    act(() => root.unmount());
  });

  const queryResult = JSON.stringify({
    status: 200,
    response: {
      results: [
        {
          columnNames: ["id", "name"],
          rows: [
            { values: [{ int32Value: 7 }, { stringValue: "Ada" }] },
            { values: [{ int32Value: 8 }, { stringValue: "Grace" }] },
          ],
          rowsCount: "2",
        },
      ],
    },
  });

  test("shows SQL query rows without expanding raw tool details", () => {
    const { container, unmount } = renderCard("SQLService/Query", queryResult);

    const grid = container.querySelector("[data-query-result-table-grid]");
    expect(grid).not.toBeNull();
    expect(grid?.querySelectorAll('[role="columnheader"]')).toHaveLength(3);
    expect(grid?.textContent).toContain("Ada");
    expect(grid?.textContent).toContain("Grace");
    expect(container.querySelector("pre")).toBeNull();

    unmount();
  });

  test("switches a SQL query result between table and vertical views", () => {
    const { container, unmount } = renderCard("SQLService/Query", queryResult);

    const viewChoices = container.querySelectorAll('[role="radio"]');
    expect(viewChoices).toHaveLength(2);
    expect(container.querySelector("[data-query-result-table-grid]")).not.toBeNull();

    act(() => {
      (viewChoices[1] as HTMLElement).click();
    });

    expect(container.querySelector("[data-query-result-table-grid]")).toBeNull();
    expect(container.querySelectorAll("[data-agent-query-result] dl")).toHaveLength(2);
    expect(container.querySelector("[data-agent-query-result] dl")?.textContent).toContain(
      "Ada"
    );

    act(() => {
      (viewChoices[0] as HTMLElement).click();
    });
    expect(container.querySelector("[data-query-result-table-grid]")).not.toBeNull();

    unmount();
  });

  test("presents SQL query rows in the indexed, horizontally scrollable result grid", () => {
    const { container, unmount } = renderCard("SQLService/Query", queryResult);

    const firstRow = container.querySelector(
      '[data-agent-query-result] [data-row-index="0"]'
    );
    expect(firstRow).not.toBeNull();
    expect(firstRow?.querySelector('[data-col-index="0"]')).toHaveTextContent(
      "1"
    );
    expect(firstRow?.querySelector('[data-col-index="1"]')).toHaveTextContent(
      "7"
    );
    expect(firstRow?.querySelector('[data-col-index="2"]')).toHaveTextContent(
      "Ada"
    );
    expect(firstRow?.closest(".overflow-auto")).toHaveClass("w-full");

    unmount();
  });

  test("presents vertical results in the shared numbered row frame", () => {
    const { container, unmount } = renderCard("SQLService/Query", queryResult);

    act(() => {
      (container.querySelectorAll('[role="radio"]')[1] as HTMLElement).click();
    });

    const rows = container.querySelectorAll(
      "[data-agent-query-result] [data-query-result-vertical-row]"
    );
    expect(rows).toHaveLength(2);
    expect(rows[0]).toHaveTextContent("1. row");
    expect(rows[0]).toHaveTextContent("Ada");
    expect(rows[1]).toHaveTextContent("2. row");
    expect(rows[1]).toHaveTextContent("Grace");

    unmount();
  });

  test("does not show a query table for another API operation", () => {
    const { container, unmount } = renderCard(
      "DatabaseService/ListDatabases",
      queryResult
    );

    expect(container.querySelector("[data-agent-query-result]")).toBeNull();

    unmount();
  });

  test("marks denied and failed API results as failed instead of completed", () => {
    for (const result of [
      { error: "Permission request denied." },
      { status: 403, error: "Permission denied.", response: null },
    ]) {
      const { container, unmount } = renderCard(
        "ProjectService/UpdateProject",
        JSON.stringify(result)
      );

      const status = container.querySelector(".text-error");
      expect(status).toHaveTextContent("agent.tool-failed");
      expect(container.querySelector(".text-success")).toBeNull();

      unmount();
    }
  });

  test("keeps masked, null, zero, and false values as table data", () => {
    const { container, unmount } = renderCard(
      "bytebase.v1.SQLService.Query",
      JSON.stringify({
        status: 200,
        response: {
          results: [
            {
              columnNames: ["mask", "null", "number", "flag"],
              rows: [
                {
                  values: [
                    { stringValue: "**rn**" },
                    { nullValue: null },
                    { int32Value: 0 },
                    { boolValue: false },
                  ],
                },
              ],
            },
          ],
        },
      })
    );

    const cells = Array.from(
      container.querySelectorAll(
        '[data-agent-query-result] [data-row-index="0"] [data-col-index]:not([data-col-index="0"])'
      )
    ).map((cell) => cell.textContent);
    expect(cells).toEqual(["**rn**", "NULL", "0", "false"]);

    act(() => {
      (container.querySelectorAll('[role="radio"]')[1] as HTMLElement).click();
    });
    const verticalValues = Array.from(
      container.querySelectorAll("[data-agent-query-result] dd")
    ).map((cell) => cell.textContent);
    expect(verticalValues).toEqual(["**rn**", "NULL", "0", "false"]);

    unmount();
  });

  test("does not turn a missing row value into SQL NULL", () => {
    const { container, unmount } = renderCard(
      "SQLService/Query",
      JSON.stringify({
        status: 200,
        response: {
          results: [
            {
              columnNames: ["id", "name"],
              rows: [{ values: [{ int32Value: 7 }] }],
            },
          ],
        },
      })
    );

    expect(
      container.querySelector('[data-row-index="0"] [data-col-index="2"]')
        ?.textContent
    ).toBe("");
    act(() => {
      (container.querySelectorAll('[role="radio"]')[1] as HTMLElement).click();
    });
    expect(container.querySelectorAll("dd")[1]?.textContent).toBe("");

    unmount();
  });

  test("formats binary values using the SQL Editor column type", () => {
    const { container, unmount } = renderCard(
      "SQLService/Query",
      JSON.stringify({
        status: 200,
        response: {
          results: [{
            columnNames: ["flags"],
            columnTypeNames: ["BIT"],
            rows: [{ values: [{ bytesValue: "pQ==" }] }],
          }],
        },
      })
    );

    expect(
      container.querySelector('[data-row-index="0"] [data-col-index="1"]')
        ?.textContent
    ).toBe("10100101");
    act(() => {
      (container.querySelectorAll('[role="radio"]')[1] as HTMLElement).click();
    });
    expect(container.querySelector("dd")?.textContent).toBe("10100101");

    unmount();
  });

  test("shows empty and failed SQL result sets without raw JSON", () => {
    const { container, unmount } = renderCard(
      "SQLService/Query",
      JSON.stringify({
        status: 200,
        response: {
          results: [
            { columnNames: ["name"], rows: [] },
            { error: "query failed", columnNames: [], rows: [] },
          ],
        },
      })
    );

    expect(container.querySelectorAll("[data-agent-query-result]")).toHaveLength(
      1
    );
    expect(container.querySelector("[data-agent-query-result]")?.textContent).toContain(
      "common.no-data"
    );
    expect(container.querySelector('[role="alert"]')?.textContent).toBe(
      "query failed"
    );
    expect(container.querySelector("pre")).toBeNull();

    unmount();
  });

  test("does not offer view modes when a query only returned an error", () => {
    const { container, unmount } = renderCard(
      "SQLService/Query",
      JSON.stringify({
        status: 200,
        response: { results: [{ error: "query failed" }] },
      })
    );

    expect(container.querySelector('[role="radio"]')).toBeNull();
    expect(container.querySelector('[role="alert"]')?.textContent).toBe(
      "query failed"
    );
    expect(container.querySelector("button .text-error")).toHaveTextContent(
      "agent.tool-failed"
    );

    unmount();
  });

  test("marks a query failed when an error is beyond the displayed result sets", () => {
    const { container, unmount } = renderCard(
      "SQLService/Query",
      JSON.stringify({
        status: 200,
        response: {
          results: [
            { columnNames: ["value"], rows: [] },
            { columnNames: ["value"], rows: [] },
            { columnNames: ["value"], rows: [] },
            { error: "fourth statement failed" },
          ],
        },
      })
    );

    expect(container.querySelector("button .text-error")).toHaveTextContent(
      "agent.tool-failed"
    );
    unmount();
  });

  test("shows the effective failed done result rather than the model's success claim", () => {
    const container = document.createElement("div");
    const root = createRoot(container);
    act(() => {
      root.render(
        <ToolCallCard
          toolCall={{
            id: "done-1",
            name: "done",
            arguments: JSON.stringify({ text: "Query succeeded", success: true }),
          }}
          result={JSON.stringify({ text: "query failed", success: false })}
        />
      );
    });

    expect(container.querySelector(".text-error")).toHaveTextContent(
      "agent.tool-failed"
    );
    act(() => {
      (container.querySelector("button") as HTMLElement).click();
    });
    expect(container.querySelector("pre")).toHaveTextContent("query failed");
    expect(container.querySelector("pre")).not.toHaveTextContent(
      "Query succeeded"
    );
    act(() => root.unmount());
  });

  test("keeps malformed SQL tool results behind the raw disclosure", () => {
    const { container, unmount } = renderCard(
      "SQLService/Query",
      "invalid JSON"
    );

    expect(container.querySelector("[data-agent-query-result]")).toBeNull();

    unmount();
  });

  test("labels row, column, and result-set previews when truncated", () => {
    const columns = Array.from({ length: 21 }, (_, i) => `column_${i}`);
    const resultSet = {
      columnNames: columns,
      rows: Array.from({ length: 101 }, () => ({
        values: columns.map((column) => ({ stringValue: column })),
      })),
    };
    const { container, unmount } = renderCard(
      "SQLService/Query",
      JSON.stringify({
        status: 200,
        response: { results: Array.from({ length: 4 }, () => resultSet) },
      })
    );

    expect(container.querySelectorAll("[data-query-result-table-grid]")).toHaveLength(
      3
    );
    expect(container.textContent).toContain("agent.query-result.showing-rows");
    expect(container.textContent).toContain("agent.query-result.showing-columns");
    expect(container.textContent).toContain("agent.query-result.showing-results");

    unmount();
  });
});

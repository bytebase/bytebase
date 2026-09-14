import { fireEvent, render, screen, within } from "@testing-library/react";
import { describe, expect, test, vi } from "vitest";
import indexSeekKeyLookup from "./test-data/mssql/index-seek-key-lookup.xml?raw";
import cteNestedLoopInitplan from "./test-data/postgres/cte-nested-loop-initplan.json";
import spannerHashJoin from "./test-data/spanner/hash-join.json";
import { parseMssqlPlan } from "./mssql-plan";
import type { PlanParseResult, PlanTree } from "./plan-model";
import { parsePostgresPlan } from "./postgres-plan";
import { QueryPlanGrid } from "./QueryPlanGrid";
import { parseSpannerPlan } from "./spanner-plan";

const treeOf = (result: PlanParseResult): PlanTree => {
  if (!result.ok) throw new Error(result.message);
  return result.tree;
};

const treeFrom = (plan: unknown): PlanTree =>
  treeOf(parsePostgresPlan(JSON.stringify(plan)));

const headers = () =>
  screen.getAllByRole("columnheader").map((cell) => cell.textContent);

const tree = () => treeFrom(cteNestedLoopInitplan);

const renderGrid = (
  overrides: Partial<Parameters<typeof QueryPlanGrid>[0]> = {}
) => {
  const onSelect = vi.fn();
  const props = {
    tree: tree(),
    selectedId: "0",
    onSelect,
    ...overrides,
  };
  return { onSelect, ...render(<QueryPlanGrid {...props} />) };
};

const rows = () => screen.getAllByTestId("plan-grid-row");
const nodeTypes = () =>
  screen.getAllByTestId("plan-grid-node-type").map((cell) => cell.textContent);
const cellsOf = (row: HTMLElement) =>
  within(row)
    .getAllByRole("cell")
    .map((cell) => cell.textContent);
const barWidth = (index: number) =>
  Number.parseFloat(
    screen.getAllByTestId("plan-cost-share-bar")[index].style.width
  );

describe("QueryPlanGrid", () => {
  test("lists every plan node once, in depth-first order", () => {
    renderGrid();

    expect(rows()).toHaveLength(6);
    // Six plan nodes plus the header row.
    expect(screen.getAllByRole("row")).toHaveLength(7);
    expect(nodeTypes()).toEqual([
      "Limit",
      "Aggregate",
      "Seq Scan",
      "Nested Loop",
      "Seq Scan",
      "Index Scan",
    ]);
  });

  test("has a column for every estimate PostgreSQL reports", () => {
    renderGrid();
    expect(headers()).toEqual([
      "#",
      "Node",
      "Added cost",
      "Total cost",
      "Rows",
      "Width",
    ]);
  });

  test("marks an estimate one node lacks and the rest of the plan has", () => {
    renderGrid({ tree: treeOf(parseMssqlPlan(indexSeekKeyLookup)) });

    expect(headers()).toEqual([
      "#",
      "Node",
      "Added cost",
      "Total cost",
      "Rows",
      "Width",
    ]);
    // A SQL Server statement has a cost and rows but no row width.
    expect(cellsOf(rows()[0]).slice(2)).toEqual(["0", "0.0325", "10", "—"]);
  });

  test("leaves out the estimate columns of a plan without estimates", () => {
    renderGrid({
      tree: treeOf(parseSpannerPlan(JSON.stringify(spannerHashJoin))),
    });

    expect(headers()).toEqual(["#", "Node"]);
    expect(cellsOf(rows()[4])).toEqual(["5", "Table ScanAlbums"]);
    expect(screen.queryAllByTestId("plan-cost-share-bar")).toHaveLength(0);
  });

  test("numbers rows from one and reports each node's estimates", () => {
    renderGrid();

    expect(cellsOf(rows()[0])).toEqual([
      "1",
      "Limit",
      "0",
      "979.51",
      "10",
      "9",
    ]);
    expect(cellsOf(rows()[4])).toEqual([
      "5",
      "Seq Scanorders",
      "1,319",
      "1,319",
      "464",
      "8",
    ]);
  });

  test("indents each node by its depth and exposes that level", () => {
    renderGrid();

    expect(rows().map((row) => row.getAttribute("aria-level"))).toEqual([
      "1",
      "2",
      "3",
      "2",
      "3",
      "3",
    ]);
    expect(
      screen
        .getAllByTestId("plan-grid-node")
        .map((cell) => cell.style.paddingLeft)
    ).toEqual(["0px", "12px", "24px", "12px", "24px", "24px"]);
  });

  test("sizes each bar by the node's share of the plan's cost", () => {
    renderGrid();
    // The same denominator the diagram and the summary use: two bars that look
    // alike have to mean the same thing.
    const basis = tree().costBasis;

    expect(barWidth(4)).toBeCloseTo((1319 / basis) * 100, 4);
    expect(barWidth(2)).toBeCloseTo((819 / basis) * 100, 4);
    expect(barWidth(0)).toBe(0);
  });

  test("marks the selected row and reports a click on another", () => {
    const { onSelect } = renderGrid();

    expect(rows()[0]).toHaveAttribute("aria-selected", "true");
    expect(rows()[3]).toHaveAttribute("aria-selected", "false");

    fireEvent.click(rows()[3]);
    expect(onSelect).toHaveBeenCalledWith("0.1");
  });

  test("keeps one tab stop and moves it with the selection", () => {
    const { rerender } = renderGrid();

    expect(rows().map((row) => row.tabIndex)).toEqual([0, -1, -1, -1, -1, -1]);

    rerender(
      <QueryPlanGrid
        tree={tree()}
        selectedId="0.1.1"
        onSelect={() => undefined}
      />
    );
    expect(rows().map((row) => row.tabIndex)).toEqual([-1, -1, -1, -1, -1, 0]);
  });

  test("walks the plan with the arrow, home, and end keys", () => {
    const { onSelect } = renderGrid();

    fireEvent.keyDown(rows()[0], { key: "ArrowDown" });
    expect(onSelect).toHaveBeenLastCalledWith("0.0");
    expect(document.activeElement).toBe(rows()[1]);

    fireEvent.keyDown(rows()[1], { key: "ArrowUp" });
    expect(onSelect).toHaveBeenLastCalledWith("0");

    fireEvent.keyDown(rows()[0], { key: "End" });
    expect(onSelect).toHaveBeenLastCalledWith("0.1.1");

    fireEvent.keyDown(rows()[5], { key: "Home" });
    expect(onSelect).toHaveBeenLastCalledWith("0");
  });

  test("stays at the ends of the plan instead of wrapping", () => {
    const { onSelect } = renderGrid();

    fireEvent.keyDown(rows()[0], { key: "ArrowUp" });
    fireEvent.keyDown(rows()[5], { key: "ArrowDown" });
    expect(onSelect).not.toHaveBeenCalled();
  });

  test("activates the focused row with enter and space", () => {
    const { onSelect } = renderGrid();

    fireEvent.keyDown(rows()[2], { key: "Enter" });
    fireEvent.keyDown(rows()[2], { key: " " });
    expect(onSelect).toHaveBeenNthCalledWith(1, "0.0.0");
    expect(onSelect).toHaveBeenNthCalledWith(2, "0.0.0");
  });

  test("renders a single-node plan without a scrollable body", () => {
    renderGrid({
      tree: treeFrom([
        {
          Plan: {
            "Node Type": "Seq Scan",
            "Relation Name": "customers",
            "Startup Cost": 0,
            "Total Cost": 90.5,
            "Plan Rows": 1667,
            "Plan Width": 12,
          },
        },
      ]),
      selectedId: "0",
    });

    expect(rows()).toHaveLength(1);
    expect(barWidth(0)).toBe(100);
  });

  test("draws a connector guide for every level a row is nested under", () => {
    renderGrid();
    const guideCells = (index: number) =>
      Array.from(
        screen.getAllByTestId("plan-grid-guides")[index]?.children ?? []
      );

    // The root is not nested under anything, so it has no guides at all.
    expect(screen.queryAllByTestId("plan-grid-guides")).toHaveLength(5);
    expect(rows()[0].querySelector("[data-testid='plan-grid-guides']")).toBeNull();
    // The scan at depth two draws one pass-through level and one elbow.
    expect(guideCells(1)).toHaveLength(2);
    expect(guideCells(0)).toHaveLength(1);
  });

  test("keeps the guides out of the accessibility tree", () => {
    renderGrid();

    for (const guides of screen.getAllByTestId("plan-grid-guides")) {
      expect(guides).toHaveAttribute("aria-hidden", "true");
    }
    // The nesting is stated once, by the row itself.
    expect(rows()[2]).toHaveAttribute("aria-level", "3");
  });

  test("leaves the row's own indent and content where they were", () => {
    renderGrid();

    expect(
      screen.getAllByTestId("plan-grid-node").map((cell) => cell.style.paddingLeft)
    ).toEqual(["0px", "12px", "24px", "12px", "24px", "24px"]);
    expect(cellsOf(rows()[0])).toHaveLength(6);
  });

  test("renders a subject containing markup as text, not as markup", () => {
    const injected = "<img src=x onerror=alert(1)>";
    const { container } = renderGrid({
      tree: treeFrom([
        {
          Plan: {
            "Node Type": "Seq Scan",
            "Startup Cost": 0,
            "Total Cost": 1,
            "Plan Rows": 1,
            "Plan Width": 1,
            Filter: injected,
          },
        },
      ]),
    });

    expect(screen.getByTestId("plan-grid-node-subject")).toHaveTextContent(
      injected
    );
    expect(container.querySelector("img")).toBeNull();
  });
});

import { fireEvent, render, screen, within } from "@testing-library/react";
import { describe, expect, test, vi } from "vitest";
import mssqlHashJoinAggregateSort from "./test-data/mssql/hash-join-aggregate-sort.xml?raw";
import twoStatementBatch from "./test-data/mssql/two-statement-batch.xml?raw";
import cteNestedLoopInitplan from "./test-data/postgres/cte-nested-loop-initplan.json";
import hashJoinAggregateSort from "./test-data/postgres/hash-join-aggregate-sort.json";
import seqScanFilter from "./test-data/postgres/seq-scan-filter.json";
import { parseMssqlPlan } from "./mssql-plan";
import type { PlanParseResult, PlanTree } from "./plan-model";
import { parsePostgresPlan } from "./postgres-plan";
import { QueryPlanSummary } from "./QueryPlanSummary";

const treeOf = (result: PlanParseResult): PlanTree => {
  if (!result.ok) throw new Error(result.message);
  return result.tree;
};

const treeFrom = (plan: unknown): PlanTree =>
  treeOf(parsePostgresPlan(JSON.stringify(plan)));

/** A plan the planner costs at nothing, as a `VALUES`-only statement is. */
const zeroCostPlan = [
  {
    Plan: {
      "Node Type": "Limit",
      "Startup Cost": 0,
      "Total Cost": 0,
      "Plan Rows": 1,
      "Plan Width": 4,
      Plans: [
        {
          "Node Type": "Result",
          "Parent Relationship": "Outer",
          "Startup Cost": 0,
          "Total Cost": 0,
          "Plan Rows": 1,
          "Plan Width": 4,
        },
      ],
    },
  },
];

const renderSummary = (
  overrides: Partial<Parameters<typeof QueryPlanSummary>[0]> = {}
) => {
  const onSelect = vi.fn();
  const props = {
    tree: treeFrom(hashJoinAggregateSort),
    selectedId: "0",
    onSelect,
    ...overrides,
  };
  return { onSelect, ...render(<QueryPlanSummary {...props} />) };
};

const timelineRows = () => screen.getAllByTestId("plan-timeline-row");
const spans = () => screen.getAllByTestId("plan-timeline-span");
const percentOf = (value: string) => Number.parseFloat(value);

describe("QueryPlanSummary", () => {
  test("reports the plan's totals and says they are estimates", () => {
    renderSummary();

    expect(screen.getByText("Nodes").nextSibling).toHaveTextContent("6");
    expect(
      screen.getByText("Total estimated cost").nextSibling
    ).toHaveTextContent("1,465.93");
    expect(
      screen.getByText("Estimated rows returned").nextSibling
    ).toHaveTextContent("3");
    expect(screen.getByText(/optimizer estimate rather than a measurement/))
      .toBeVisible();
    expect(
      screen.getByText(/comparable only within this plan/)
    ).toBeInTheDocument();
  });

  test("plots one timeline row per node, in plan order", () => {
    renderSummary();

    expect(
      screen
        .getAllByTestId("plan-timeline-node-type")
        .map((cell) => cell.textContent)
    ).toEqual([
      "Sort",
      "HashAggregate",
      "Hash Join",
      "Seq Scan",
      "Hash",
      "Seq Scan",
    ]);
  });

  test("spans each bar from the startup cost to the total cost", () => {
    renderSummary();

    // The hash join starts at 140.5 and finishes at 1,090.86 of the plan's
    // 1,465.93 cost units.
    const join = spans()[2];
    expect(percentOf(join.style.left)).toBeCloseTo((140.5 / 1465.93) * 100, 4);
    expect(percentOf(join.style.width)).toBeCloseTo(
      ((1090.86 - 140.5) / 1465.93) * 100,
      4
    );

  });

  test("keeps a span too short to draw visible and inside the axis", () => {
    renderSummary();

    // The root sort spans 1,465.92 to 1,465.93 — a thousandth of a percent,
    // so it is widened to the floor and pushed back off the axis end.
    const sort = spans()[0];
    expect(percentOf(sort.style.width)).toBe(0.5);
    expect(percentOf(sort.style.left)).toBe(99.5);
  });

  test("shades a span by how much of the plan the node itself costs", () => {
    renderSummary();
    const opacity = (index: number) => Number(spans()[index].style.opacity);

    // The scan of `orders` owns the largest self cost, so it saturates, and
    // the Hash that adds nothing stays at the floor that keeps it visible.
    expect(opacity(3)).toBe(1);
    expect(opacity(4)).toBe(0.3);
    expect(opacity(2)).toBeGreaterThan(0.3);
    expect(opacity(2)).toBeLessThan(1);
  });

  test("labels the axis from zero to the plan's cost", () => {
    renderSummary();
    expect(screen.getByTestId("plan-timeline-axis")).toHaveTextContent(
      "01,465.93"
    );
  });

  test("stretches the axis over the self costs a Limit hides", () => {
    renderSummary({ tree: treeFrom(cteNestedLoopInitplan) });
    // The root Limit reports 979.51, but its children account for 2,578.21.
    expect(screen.getByTestId("plan-timeline-axis")).toHaveTextContent(
      "02,578.21"
    );
  });

  test("indents a timeline row by the depth the planner nested it at", () => {
    renderSummary({ tree: treeFrom(cteNestedLoopInitplan) });

    expect(
      screen
        .getAllByTestId("plan-timeline-node-type")
        .map((cell) => (cell.parentElement as HTMLElement).style.paddingLeft)
    ).toEqual(["0px", "12px", "24px", "12px", "24px", "24px"]);
  });

  test("selects a node from its timeline row", () => {
    const { onSelect } = renderSummary();

    expect(timelineRows()[0]).toHaveAttribute("aria-pressed", "true");
    expect(timelineRows()[3]).toHaveAttribute("aria-pressed", "false");

    fireEvent.click(timelineRows()[3]);
    expect(onSelect).toHaveBeenCalledWith("0.0.0.0");
  });

  test("describes a timeline row without relying on its bar", () => {
    renderSummary();

    expect(timelineRows()[3]).toHaveAccessibleName(
      "Seq Scan, orders, cost 0 to 819, 55.9% of plan cost"
    );
  });

  test("ranks the costliest operators with their share of the plan", () => {
    renderSummary();
    const rows = screen.getAllByTestId("plan-costliest-row");

    expect(rows).toHaveLength(5);
    expect(rows[0]).toHaveTextContent("Seq Scan");
    expect(rows[0]).toHaveTextContent("orders");
    expect(rows[0]).toHaveTextContent("819");
    expect(rows[0]).toHaveTextContent("55.9%");
    expect(rows[1]).toHaveTextContent("HashAggregate");
    // The Hash node adds nothing on top of its child, so it is not ranked.
    expect(
      rows.some((row) => row.textContent?.startsWith("5Hash"))
    ).toBe(false);
  });

  test("selects a node from the costliest list", () => {
    const { onSelect } = renderSummary();

    fireEvent.click(screen.getAllByTestId("plan-costliest-row")[0]);
    expect(onSelect).toHaveBeenCalledWith("0.0.0.0");
  });

  test("totals the cost of every operation kind in the plan", () => {
    renderSummary();
    const cells = screen
      .getAllByTestId("plan-operation-row")
      .map((row) =>
        within(row)
          .getAllByRole("cell")
          .map((cell) => cell.textContent)
      );

    expect(cells[0]).toEqual(["Seq Scan", "2", "897", "61.2%"]);
    expect(cells.map((row) => row[0])).toEqual([
      "Seq Scan",
      "HashAggregate",
      "Hash Join",
      "Sort",
      "Hash",
    ]);
    expect(
      Number.parseFloat(
        screen.getAllByTestId("plan-cost-share-bar")[0].style.width
      )
    ).toBeCloseTo((897 / 1465.93) * 100, 4);
  });

  test("keeps every share inside the axis when a Limit truncates the root", () => {
    renderSummary({ tree: treeFrom(cteNestedLoopInitplan) });

    for (const span of spans()) {
      expect(percentOf(span.style.left)).toBeGreaterThanOrEqual(0);
      expect(
        percentOf(span.style.left) + percentOf(span.style.width)
      ).toBeLessThanOrEqual(100.0001);
    }
    // 1,319 of the 2,578.21 cost units the plan's nodes account for.
    expect(screen.getAllByTestId("plan-costliest-row")[0]).toHaveTextContent(
      "51.2%"
    );
  });

  test("gives a single-node plan the whole timeline", () => {
    renderSummary({ tree: treeFrom(seqScanFilter) });

    expect(timelineRows()).toHaveLength(1);
    expect(percentOf(spans()[0].style.left)).toBe(0);
    expect(percentOf(spans()[0].style.width)).toBe(100);
    expect(screen.getAllByTestId("plan-costliest-row")[0]).toHaveTextContent(
      "100%"
    );
    expect(
      within(screen.getAllByTestId("plan-operation-row")[0])
        .getAllByRole("cell")
        .map((cell) => cell.textContent)
    ).toEqual(["Seq Scan", "1", "90.5", "100%"]);
  });

  test("reports a plan the planner costs at nothing without dividing by it", () => {
    renderSummary({ tree: treeFrom(zeroCostPlan) });

    expect(spans().map((span) => span.style.left)).toEqual(["0%", "0%"]);
    expect(spans().map((span) => span.style.width)).toEqual(["0.5%", "0.5%"]);
    expect(timelineRows()[0]).toHaveTextContent("0%");
    expect(screen.queryAllByTestId("plan-costliest-row")).toHaveLength(0);
    expect(
      screen.getByText("No node in this plan adds an estimated cost of its own.")
    ).toBeVisible();
    expect(
      screen
        .getAllByTestId("plan-operation-row")
        .map((row) => row.textContent)
    ).toEqual(["Limit100%", "Result100%"]);
  });

  test("leaves the cost ranges out of a plan without startup costs", () => {
    renderSummary({ tree: treeOf(parseMssqlPlan(mssqlHashJoinAggregateSort)) });

    expect(screen.queryByText("Cost ranges")).toBeNull();
    expect(screen.queryByTestId("plan-timeline")).toBeNull();
    // The rest of the summary reads SQL Server's costs like any other.
    expect(
      screen.getByText("Total estimated cost").nextSibling
    ).toHaveTextContent("0.277");
    expect(screen.getAllByTestId("plan-costliest-row")[0]).toHaveTextContent(
      "Index Scan"
    );
    expect(screen.getAllByTestId("plan-operation-row").length).toBeGreaterThan(
      0
    );
  });

  test("leaves out the rows returned when the root estimates none", () => {
    renderSummary({ tree: treeOf(parseMssqlPlan(twoStatementBatch)) });

    expect(screen.getByText("Nodes").nextSibling).toHaveTextContent("9");
    expect(screen.queryByText("Estimated rows returned")).toBeNull();
  });

  test("renders a subject containing markup as text, not as markup", () => {
    const injected = "<img src=x onerror=alert(1)>";
    const { container } = renderSummary({
      tree: treeFrom([
        {
          Plan: {
            "Node Type": "Seq Scan",
            "Startup Cost": 0,
            "Total Cost": 500,
            "Plan Rows": 1,
            "Plan Width": 1,
            Filter: injected,
          },
        },
      ]),
    });

    expect(timelineRows()[0]).toHaveTextContent(injected);
    expect(container.querySelector("img")).toBeNull();
  });
});

import { fireEvent, render, screen } from "@testing-library/react";
import { describe, expect, test, vi } from "vitest";
import mssqlHashJoinAggregateSort from "./test-data/mssql/hash-join-aggregate-sort.xml?raw";
import hashJoinAggregateSort from "./test-data/postgres/hash-join-aggregate-sort.json";
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

const rows = () => screen.getAllByTestId("plan-costliest-row");

describe("QueryPlanSummary", () => {
  test("ranks the costliest operators with their share of the plan", () => {
    renderSummary();

    expect(screen.getByText("Costliest by estimated cost")).toBeVisible();
    expect(rows()).toHaveLength(3);
    expect(rows()[0]).toHaveTextContent("Seq Scan");
    expect(rows()[0]).toHaveTextContent("orders");
    expect(rows()[0]).toHaveTextContent("55.9%");
    expect(rows()[1]).toHaveTextContent("HashAggregate");
  });

  test("marks the selected operator", () => {
    renderSummary({ selectedId: "0.0.0.0" });

    expect(rows()[0]).toHaveAttribute("aria-pressed", "true");
    expect(rows()[1]).toHaveAttribute("aria-pressed", "false");
  });

  test("selects a node from the ranking", () => {
    const { onSelect } = renderSummary();

    fireEvent.click(rows()[0]);
    expect(onSelect).toHaveBeenCalledWith("0.0.0.0");
  });

  test("ranks SQL Server's costs like any other engine's", () => {
    renderSummary({ tree: treeOf(parseMssqlPlan(mssqlHashJoinAggregateSort)) });

    expect(rows()[0]).toHaveTextContent("Index Scan");
  });

  test("says so when no operator adds a cost of its own", () => {
    renderSummary({ tree: treeFrom(zeroCostPlan) });

    expect(screen.queryAllByTestId("plan-costliest-row")).toHaveLength(0);
    expect(
      screen.getByText("No node in this plan adds an estimated cost of its own.")
    ).toBeVisible();
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

    expect(rows()[0]).toHaveTextContent("Seq Scan");
    expect(container.querySelector("img")).toBeNull();
  });
});

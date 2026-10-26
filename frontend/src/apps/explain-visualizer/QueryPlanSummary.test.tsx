import { fireEvent, render, screen } from "@testing-library/react";
import { describe, expect, test, vi } from "vitest";
import mssqlHashJoinAggregateSort from "./test-data/mssql/hash-join-aggregate-sort.xml?raw";
import hashJoinAggregateSort from "./test-data/postgres/hash-join-aggregate-sort.json";
import { parseMssqlPlan } from "./mssql-plan";
import {
  buildPlanTree,
  type PlanParseResult,
  type PlanTree,
} from "./plan-model";
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
const warnings = () => screen.getAllByTestId("plan-summary-warning");

/**
 * The same shape as the estimated fixture, measured: the planner expected the
 * index scan to be the cheap side and it took almost all of the time.
 */
const measuredPlan = [
  {
    Plan: {
      "Node Type": "Nested Loop",
      "Total Cost": 100,
      "Plan Rows": 10,
      "Actual Total Time": 50,
      "Actual Rows": 10,
      "Actual Loops": 1,
      Plans: [
        {
          "Node Type": "Seq Scan",
          "Parent Relationship": "Outer",
          "Relation Name": "customers",
          "Total Cost": 90,
          "Plan Rows": 5,
          "Actual Total Time": 2,
          "Actual Rows": 5,
          "Actual Loops": 1,
        },
        {
          "Node Type": "Index Scan",
          "Parent Relationship": "Inner",
          "Relation Name": "orders",
          "Total Cost": 5,
          "Plan Rows": 2,
          "Actual Total Time": 9,
          "Actual Rows": 2,
          "Actual Loops": 5,
        },
      ],
    },
  },
];

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

describe("QueryPlanSummary on a measured plan", () => {
  test("ranks by the time each operator took, and says that is what it ranked", () => {
    renderSummary({ tree: treeFrom(measuredPlan) });

    expect(screen.getByText("Slowest by measured time")).toBeVisible();
    // The planner's costliest node is the fast one; 45 ms of the plan's 50 go
    // to the inner scan it costed at 5.
    expect(rows()[0]).toHaveTextContent("Index Scan");
    expect(rows()[0]).toHaveTextContent("90%");
  });

  test("says so when no operator took measurable time of its own", () => {
    renderSummary({
      tree: treeFrom([
        {
          Plan: {
            "Node Type": "Result",
            "Total Cost": 0,
            "Actual Total Time": 0,
            "Actual Rows": 1,
            "Actual Loops": 1,
          },
        },
      ]),
    });

    expect(
      screen.getByText("No node in this plan took measurable time of its own.")
    ).toBeVisible();
  });
});

describe("QueryPlanSummary warnings", () => {
  /** A 5,000-unit plan whose full scan of `orders` costs `scanCost` of it. */
  const scanPlan = (scanCost: number) => [
    {
      Plan: {
        "Node Type": "Nested Loop",
        "Total Cost": 5000,
        "Plan Rows": 1,
        Plans: [
          {
            "Node Type": "Seq Scan",
            "Parent Relationship": "Outer",
            "Relation Name": "orders",
            "Total Cost": scanCost,
            "Plan Rows": 1,
          },
        ],
      },
    },
  ];

  test("names a scan worth a reader's time, and what it reads", () => {
    renderSummary({ tree: treeFrom(scanPlan(4000)) });

    expect(screen.getByText("Warnings")).toBeVisible();
    expect(warnings()).toHaveLength(1);
    expect(warnings()[0]).toHaveTextContent("Full table scan");
    expect(warnings()[0]).toHaveTextContent("orders");
  });

  test("leaves out a scan that is a rounding error in its plan", () => {
    // Over the parser's own threshold, so the node carries the warning, but
    // 100 cost units of a 10,100-unit plan is not where to start reading.
    renderSummary({ tree: treeFrom(scanPlan(100)) });

    expect(screen.queryAllByTestId("plan-summary-warning")).toHaveLength(0);
    expect(screen.queryByText("Warnings")).toBeNull();
  });

  test("selects the node a warning is about", () => {
    const { onSelect } = renderSummary({ tree: treeFrom(scanPlan(4000)) });

    fireEvent.click(warnings()[0]);
    expect(onSelect).toHaveBeenCalledWith("0.0");
  });

  test("counts the operators when one warning covers several", () => {
    renderSummary({
      tree: treeFrom([
        {
          Plan: {
            "Node Type": "Hash Join",
            "Total Cost": 2000,
            "Plan Rows": 1,
            Plans: [
              {
                "Node Type": "Seq Scan",
                "Parent Relationship": "Outer",
                "Relation Name": "orders",
                "Total Cost": 1200,
                "Plan Rows": 1,
              },
              {
                "Node Type": "Seq Scan",
                "Parent Relationship": "Inner",
                "Relation Name": "customers",
                "Total Cost": 800,
                "Plan Rows": 1,
              },
            ],
          },
        },
      ]),
    });

    expect(warnings()).toHaveLength(1);
    expect(warnings()[0]).toHaveTextContent("2 operators");
  });

  test("says why it ranks nothing when the engine reports no cost or time", () => {
    // What an engine that reports no estimates leaves the panel, as Spanner's
    // plans do.
    renderSummary({
      tree: buildPlanTree({
        id: "0",
        nodeType: "Scan",
        properties: [],
        warnings: [],
        children: [],
      }),
    });

    expect(
      screen.getByText(
        "This plan reports no cost or time, so its operators cannot be ranked."
      )
    ).toBeVisible();
    expect(screen.queryAllByTestId("plan-costliest-row")).toHaveLength(0);
  });
});

import { create } from "@bufbuild/protobuf";
import { describe, expect, test } from "vitest";
import { Engine } from "@/types/proto-es/v1/common_pb";
import {
  QueryResult_QueryPlanSchema,
  QueryResultSchema,
  QueryRowSchema,
  RowValueSchema,
} from "@/types/proto-es/v1/sql_service_pb";
import { inlineQueryPlan, planCostShares } from "./resultPlans";

/** A result holding one column of text, the way an explain comes back. */
const textResult = (lines: string[], columnNames = ["QUERY PLAN"]) =>
  create(QueryResultSchema, {
    statement: "SELECT 1",
    columnNames,
    rows: lines.map((line) =>
      create(QueryRowSchema, {
        values: [
          create(RowValueSchema, {
            kind: { case: "stringValue", value: line },
          }),
        ],
      })
    ),
  });

/** A PostgreSQL plan result whose root costs `totalCost`. */
const planResult = (totalCost: number) => {
  const result = textResult([
    JSON.stringify([
      { Plan: { "Node Type": "Seq Scan", "Total Cost": totalCost } },
    ]),
  ]);
  result.queryPlan = create(QueryResult_QueryPlanSchema, {});
  return result;
};

describe("inlineQueryPlan", () => {
  test("joins a single-column plan into its source text", () => {
    expect(
      inlineQueryPlan(textResult(["Seq Scan on t", "  Filter: (a = 1)"]))
        ?.source
    ).toBe("Seq Scan on t\n  Filter: (a = 1)");
  });

  test("is undefined for a result with no rows to read", () => {
    expect(inlineQueryPlan(textResult([]))).toBeUndefined();
  });
});

describe("planCostShares", () => {
  test("gives each statement its share of the script's cost", () => {
    expect(
      planCostShares([planResult(75), planResult(25)], Engine.POSTGRES)
    ).toEqual([0.75, 0.25]);
  });

  test("withholds every share when one statement is not a plan", () => {
    expect(
      planCostShares(
        [planResult(75), textResult(["a"], ["a"])],
        Engine.POSTGRES
      )
    ).toBeUndefined();
  });

  test("withholds every share when a plan cannot be read", () => {
    const unreadable = textResult(["Seq Scan on t"]);
    unreadable.queryPlan = create(QueryResult_QueryPlanSchema, {});

    expect(
      planCostShares([planResult(75), unreadable], Engine.POSTGRES)
    ).toBeUndefined();
  });

  test("invents no order for a script the planner costs at nothing", () => {
    expect(
      planCostShares([planResult(0), planResult(0)], Engine.POSTGRES)
    ).toBeUndefined();
  });

  test("has nothing to say about an engine whose plans it cannot read", () => {
    expect(
      planCostShares([planResult(75), planResult(25)], Engine.MYSQL)
    ).toBeUndefined();
    expect(planCostShares([], Engine.POSTGRES)).toBeUndefined();
  });
});

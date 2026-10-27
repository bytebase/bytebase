import { create } from "@bufbuild/protobuf";
import { describe, expect, test } from "vitest";
import {
  QueryOption_ExplainFormat,
  QueryResult_QueryPlanSchema,
  QueryResultSchema,
} from "@/types/proto-es/v1/sql_service_pb";
import { isTextPlanFormat, planAvailability } from "./planAvailability";

const rows = (statement = "SELECT 1") =>
  create(QueryResultSchema, { statement, columnNames: ["a"] });
const affectedRows = () =>
  create(QueryResultSchema, {
    statement: "INSERT INTO t VALUES (1)",
    columnNames: ["Affected Rows"],
  });
const noRows = () =>
  create(QueryResultSchema, { statement: "SET search_path = tenant" });
const setConfig = () =>
  rows("SELECT set_config('search_path', 'tenant,public', false)");
const plan = (executed: boolean) =>
  create(QueryResultSchema, {
    statement: "EXPLAIN SELECT 1",
    columnNames: ["QUERY PLAN"],
    queryPlan: create(QueryResult_QueryPlanSchema, { executed }),
  });

const offered = (overrides: Partial<Parameters<typeof planAvailability>[0]>) =>
  planAvailability({
    result: rows(),
    results: [rows()],
    resultIndex: 0,
    engineExplains: true,
    canExplain: true,
    ...overrides,
  });

/** A result whose rows are data and whose plan came back beside them. */
const carriedPlan = () =>
  create(QueryResultSchema, {
    statement: "SELECT city, count(*) FROM customers GROUP BY city",
    columnNames: ["city", "f0_"],
    queryPlan: create(QueryResult_QueryPlanSchema, {
      format: QueryOption_ExplainFormat.JSON,
      executed: true,
      content: '[{"id":0,"name":"S00: Input"}]',
    }),
  });

describe("planAvailability", () => {
  test("offers the plan an engine stated beside the rows", () => {
    // BigQuery reports the stages its job ran, so the plan is in hand: the view
    // draws it without an explain request, which is why neither the engine's
    // explain support nor the permission for one gates it.
    expect(
      offered({
        result: carriedPlan(),
        results: [carriedPlan()],
        engineExplains: false,
        canExplain: false,
      })
    ).toBe("offered");
  });

  test("offers a plan for an ordinary read", () => {
    expect(offered({})).toBe("offered");
  });

  test("is disabled, not absent, without the explain permission", () => {
    expect(offered({ canExplain: false })).toBe("needs-permission");
  });

  test("is absent when the result already is a plan", () => {
    const result = create(QueryResultSchema, {
      statement: "EXPLAIN SELECT 1",
      columnNames: ["QUERY PLAN"],
      queryPlan: create(QueryResult_QueryPlanSchema, {}),
    });
    expect(offered({ result, results: [result] })).toBe("absent");
  });

  test("is absent on an engine that cannot explain", () => {
    expect(offered({ engineExplains: false })).toBe("absent");
  });

  test("is absent when the run recorded no executed statement", () => {
    const result = rows("");
    expect(offered({ result, results: [result] })).toBe("absent");
  });

  test("is absent after any earlier statement that executed", () => {
    for (const earlier of [noRows(), affectedRows(), setConfig(), rows()]) {
      expect(offered({ results: [earlier, rows()], resultIndex: 1 })).toBe(
        "absent"
      );
    }
    expect(offered({ results: [plan(true), rows()], resultIndex: 1 })).toBe(
      "absent"
    );
  });

  test("stays offered after earlier plans that did not execute", () => {
    expect(offered({ results: [plan(false), rows()], resultIndex: 1 })).toBe(
      "offered"
    );
  });
});

describe("isTextPlanFormat", () => {
  test("counts an unknown format as text, so its plan can be fetched again", () => {
    // MySQL's omitted FORMAT follows the session's explain_format, so a plain
    // EXPLAIN comes back without the result being able to say in what.
    expect(
      isTextPlanFormat(QueryOption_ExplainFormat.EXPLAIN_FORMAT_UNSPECIFIED)
    ).toBe(true);
    expect(isTextPlanFormat(QueryOption_ExplainFormat.TEXT)).toBe(true);
  });

  test("leaves out a format the viewer draws, which needs no second call", () => {
    expect(isTextPlanFormat(QueryOption_ExplainFormat.JSON)).toBe(false);
    expect(isTextPlanFormat(QueryOption_ExplainFormat.XML)).toBe(false);
    expect(isTextPlanFormat(undefined)).toBe(false);
  });
});

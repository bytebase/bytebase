import { create } from "@bufbuild/protobuf";
import { describe, expect, test } from "vitest";
import {
  QueryResult_QueryPlanSchema,
  QueryResultSchema,
} from "@/types/proto-es/v1/sql_service_pb";
import { planAvailability } from "./planAvailability";

const rows = (statement = "SELECT 1") =>
  create(QueryResultSchema, { statement, columnNames: ["a"] });
const affectedRows = () =>
  create(QueryResultSchema, {
    statement: "INSERT INTO t VALUES (1)",
    columnNames: ["Affected Rows"],
  });
const noRows = () =>
  create(QueryResultSchema, { statement: "SET search_path = tenant" });

const offered = (overrides: Partial<Parameters<typeof planAvailability>[0]>) =>
  planAvailability({
    result: rows(),
    results: [rows()],
    resultIndex: 0,
    engineExplains: true,
    canExplain: true,
    ...overrides,
  });

describe("planAvailability", () => {
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

  test("is absent after an earlier statement that returned no rows", () => {
    expect(offered({ results: [noRows(), rows()], resultIndex: 1 })).toBe(
      "absent"
    );
    expect(offered({ results: [affectedRows(), rows()], resultIndex: 1 })).toBe(
      "absent"
    );
  });

  test("stays offered after earlier statements that returned rows", () => {
    expect(offered({ results: [rows(), rows()], resultIndex: 1 })).toBe(
      "offered"
    );
  });
});

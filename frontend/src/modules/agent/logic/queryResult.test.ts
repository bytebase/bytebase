import { describe, expect, test } from "vitest";
import type { QueryRow } from "@/types/proto-es/v1/sql_service_pb";
import { extractSQLRowValuePlain } from "@/utils/v1/sql";
import { parseAgentQueryResult } from "./queryResult";

const apiResult = (results: unknown[], status = 200) =>
  JSON.stringify({ status, response: { results } });

const plainRows = (rows: QueryRow[]) =>
  rows.map((row) => row.values.map(extractSQLRowValuePlain));

describe("parseAgentQueryResult", () => {
  test("returns row values, including masked strings and nulls, from a successful query", () => {
    const preview = parseAgentQueryResult(
      apiResult([
        {
          columnNames: ["id", "secret", "missing"],
          rows: [
            {
              values: [
                { int64Value: "42" },
                { stringValue: "**rn**" },
                { nullValue: null },
              ],
            },
          ],
        },
      ])
    );

    expect(preview).toMatchObject({
      resultSets: [
        {
          columns: ["id", "secret", "missing"],
          returnedRowCount: 1,
          displayedRowCount: 1,
          returnedColumnCount: 3,
          displayedColumnCount: 3,
          rowsTruncated: false,
          columnsTruncated: false,
        },
      ],
      totalResultSets: 1,
      resultSetsTruncated: false,
    });
    expect(plainRows(preview?.resultSets[0].rows ?? [])).toEqual([
      ["42", "**rn**", null],
    ]);
  });

  test("retains typed row values for result presentation", () => {
    const preview = parseAgentQueryResult(
      apiResult([
        {
          columnNames: ["count", "payload"],
          columnTypeNames: ["BIGINT", "BIT"],
          rows: [
            {
              values: [{ int64Value: "42" }, { bytesValue: "AQID" }],
            },
          ],
        },
      ])
    );

    expect(preview?.resultSets[0].rows[0].values[0].kind).toEqual({
      case: "int64Value",
      value: 42n,
    });
    expect(preview?.resultSets[0].rows[0].values[1].kind).toEqual({
      case: "bytesValue",
      value: new Uint8Array([1, 2, 3]),
    });
    expect(preview?.resultSets[0].columnTypes).toEqual(["BIGINT", "BIT"]);
  });

  test("caps rows, columns, and result sets while retaining returned counts", () => {
    const columns = Array.from({ length: 21 }, (_, i) => `column_${i}`);
    const row = {
      values: columns.map((_, i) => ({ stringValue: `value_${i}` })),
    };
    const result = {
      columnNames: columns,
      columnTypeNames: columns.map(() => "TEXT"),
      rows: Array.from({ length: 101 }, () => row),
    };
    const preview = parseAgentQueryResult(
      apiResult([result, result, result, result])
    );

    expect(preview?.totalResultSets).toBe(4);
    expect(preview?.resultSetsTruncated).toBe(true);
    expect(preview?.resultSets).toHaveLength(3);
    expect(preview?.resultSets[0]).toMatchObject({
      returnedRowCount: 101,
      displayedRowCount: 100,
      returnedColumnCount: 21,
      displayedColumnCount: 20,
      rowsTruncated: true,
      columnsTruncated: true,
    });
    expect(preview?.resultSets[0].rows).toHaveLength(100);
    expect(preview?.resultSets[0].rows[0].values).toHaveLength(20);
    expect(preview?.resultSets[0].columns).toHaveLength(20);
    expect(preview?.resultSets[0].columnTypes).toHaveLength(20);
  });

  test("preserves an empty result and a per-result query error", () => {
    const preview = parseAgentQueryResult(
      apiResult([
        { columnNames: ["name"], rows: [] },
        { error: "permission denied", columnNames: [], rows: [] },
      ])
    );

    expect(preview?.resultSets).toMatchObject([
      {
        columns: ["name"],
        rows: [],
        returnedRowCount: 0,
        displayedRowCount: 0,
        returnedColumnCount: 1,
        displayedColumnCount: 1,
        rowsTruncated: false,
        columnsTruncated: false,
      },
      {
        columns: [],
        rows: [],
        returnedRowCount: 0,
        displayedRowCount: 0,
        returnedColumnCount: 0,
        displayedColumnCount: 0,
        rowsTruncated: false,
        columnsTruncated: false,
        error: "permission denied",
      },
    ]);
  });

  test("finds a tabular result after non-tabular statements", () => {
    const preview = parseAgentQueryResult(
      apiResult([
        { statement: "SET search_path TO public", rowsCount: "0" },
        { statement: "UPDATE settings SET enabled = true", rowsCount: "1" },
        { statement: "SET statement_timeout = 0", rowsCount: "0" },
        {
          columnNames: ["answer"],
          rows: [{ values: [{ int32Value: 42 }] }],
        },
      ])
    );

    expect(preview?.resultSets).toHaveLength(1);
    expect(plainRows(preview?.resultSets[0].rows ?? [])).toEqual([[42]]);
    expect(preview?.resultSetsTruncated).toBe(false);
  });

  test.each([
    "",
    "not json",
    JSON.stringify({ status: 403, response: { results: [] } }),
    JSON.stringify({ status: 200, response: {} }),
    JSON.stringify({ status: 200, response: { results: "not an array" } }),
    JSON.stringify({ status: 200, response: { results: [{ rows: {} }] } }),
  ])("ignores malformed or unsuccessful API results: %s", (raw) => {
    expect(parseAgentQueryResult(raw)).toBeUndefined();
  });
});

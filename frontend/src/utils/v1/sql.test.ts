import { create } from "@bufbuild/protobuf";
import { TimestampSchema } from "@bufbuild/protobuf/wkt";
import { describe, expect, it, vi } from "vitest";
import { Engine } from "@/types/proto-es/v1/common_pb";

// `sql.ts` imports `getDateForPbTimestampProtoEs` from the heavy `@/types`
// barrel, which transitively pulls in the React router. Mock it with a
// minimal real implementation so the unit under test stays isolated.
vi.mock("@/types", () => ({
  getDateForPbTimestampProtoEs: (ts: { seconds: bigint; nanos: number }) =>
    new Date(Number(ts.seconds) * 1000 + Math.floor((ts.nanos ?? 0) / 1e6)),
}));

import type { RowValue } from "@/types/proto-es/v1/sql_service_pb";
import {
  RowValue_TimestampSchema,
  RowValueSchema,
} from "@/types/proto-es/v1/sql_service_pb";
import { generateInsertStatementFromRows, rowValueToSQLLiteral } from "./sql";

const rv = (kind: RowValue["kind"]): RowValue =>
  create(RowValueSchema, { kind });

describe("rowValueToSQLLiteral", () => {
  it("renders NULL for undefined and nullValue", () => {
    expect(rowValueToSQLLiteral(undefined, Engine.POSTGRES)).toBe("NULL");
    expect(
      rowValueToSQLLiteral(rv({ case: "nullValue", value: 0 }), Engine.POSTGRES)
    ).toBe("NULL");
  });

  it("renders booleans as TRUE/FALSE by default", () => {
    expect(
      rowValueToSQLLiteral(
        rv({ case: "boolValue", value: true }),
        Engine.POSTGRES
      )
    ).toBe("TRUE");
    expect(
      rowValueToSQLLiteral(
        rv({ case: "boolValue", value: false }),
        Engine.POSTGRES
      )
    ).toBe("FALSE");
  });

  it("renders booleans as 1/0 for MySQL-family engines", () => {
    expect(
      rowValueToSQLLiteral(rv({ case: "boolValue", value: true }), Engine.MYSQL)
    ).toBe("1");
    expect(
      rowValueToSQLLiteral(
        rv({ case: "boolValue", value: false }),
        Engine.MYSQL
      )
    ).toBe("0");
  });

  it("renders numbers as bare literals", () => {
    expect(
      rowValueToSQLLiteral(rv({ case: "int32Value", value: 42 }), Engine.MYSQL)
    ).toBe("42");
    expect(
      rowValueToSQLLiteral(
        rv({ case: "int64Value", value: 9999999999n }),
        Engine.MYSQL
      )
    ).toBe("9999999999");
    expect(
      rowValueToSQLLiteral(
        rv({ case: "doubleValue", value: 3.14 }),
        Engine.MYSQL
      )
    ).toBe("3.14");
  });

  it("single-quotes strings and doubles embedded single quotes", () => {
    expect(
      rowValueToSQLLiteral(
        rv({ case: "stringValue", value: "hello" }),
        Engine.POSTGRES
      )
    ).toBe("'hello'");
    expect(
      rowValueToSQLLiteral(
        rv({ case: "stringValue", value: "O'Brien" }),
        Engine.POSTGRES
      )
    ).toBe("'O''Brien'");
  });

  it("escapes backslashes only for MySQL-family engines", () => {
    const value = rv({ case: "stringValue", value: "a\\b" });
    // MySQL treats backslash as an escape character → must double it.
    expect(rowValueToSQLLiteral(value, Engine.MYSQL)).toBe("'a\\\\b'");
    // Postgres (standard_conforming_strings) treats backslash literally.
    expect(rowValueToSQLLiteral(value, Engine.POSTGRES)).toBe("'a\\b'");
  });

  it("renders bytes as engine-specific binary literals", () => {
    const value = rv({
      case: "bytesValue",
      value: new Uint8Array([0x01, 0x02, 0xab]),
    });
    expect(rowValueToSQLLiteral(value, Engine.MYSQL)).toBe("0x0102AB");
    expect(rowValueToSQLLiteral(value, Engine.MSSQL)).toBe("0x0102AB");
    expect(rowValueToSQLLiteral(value, Engine.POSTGRES)).toBe("'\\x0102AB'");
  });

  it("single-quotes timestamps", () => {
    const seconds = BigInt(Date.UTC(2026, 3, 20, 13, 56, 0) / 1000);
    const value = rv({
      case: "timestampValue",
      value: create(RowValue_TimestampSchema, {
        googleTimestamp: create(TimestampSchema, { seconds, nanos: 0 }),
        accuracy: 0,
      }),
    });
    expect(rowValueToSQLLiteral(value, Engine.POSTGRES)).toBe(
      "'2026-04-20 13:56:00'"
    );
  });
});

describe("generateInsertStatementFromRows", () => {
  it("builds a single batched INSERT with engine-aware quoting", () => {
    const rows: RowValue[][] = [
      [
        rv({ case: "stringValue", value: "new-instance" }),
        rv({ case: "stringValue", value: "new_db" }),
        rv({ case: "boolValue", value: false }),
        rv({ case: "nullValue", value: 0 }),
      ],
      [
        rv({ case: "stringValue", value: "other" }),
        rv({ case: "stringValue", value: "db2" }),
        rv({ case: "boolValue", value: true }),
        rv({ case: "stringValue", value: "sample" }),
      ],
    ];
    const sql = generateInsertStatementFromRows({
      engine: Engine.POSTGRES,
      schema: "public",
      table: "<table_name>",
      columns: ["instance", "name", "deleted", "environment"],
      rows,
    });
    expect(sql).toBe(
      `INSERT INTO "public"."<table_name>" ("instance", "name", "deleted", "environment") VALUES\n` +
        `  ('new-instance', 'new_db', FALSE, NULL),\n` +
        `  ('other', 'db2', TRUE, 'sample');`
    );
  });

  it("omits the schema prefix when schema is empty", () => {
    const rows: RowValue[][] = [[rv({ case: "int32Value", value: 1 })]];
    const sql = generateInsertStatementFromRows({
      engine: Engine.MYSQL,
      schema: undefined,
      table: "<table_name>",
      columns: ["id"],
      rows,
    });
    expect(sql).toBe("INSERT INTO `<table_name>` (`id`) VALUES\n  (1);");
  });
});

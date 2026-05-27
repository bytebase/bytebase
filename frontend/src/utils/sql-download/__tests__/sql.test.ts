import { create } from "@bufbuild/protobuf";
import { StructSchema, type Value, ValueSchema } from "@bufbuild/protobuf/wkt";
import { describe, expect, it } from "vitest";
import { Engine } from "@/types/proto-es/v1/common_pb";
import {
  type QueryResult,
  QueryResultSchema,
  QueryRowSchema,
  type RowValue,
  RowValueSchema,
} from "@/types/proto-es/v1/sql_service_pb";
import { serializeSQL } from "../formats/sql";

const TEXT_DECODER = new TextDecoder("utf-8");

const rowOf = (...values: RowValue[]) => create(QueryRowSchema, { values });
const intRow = (n: bigint): RowValue =>
  create(RowValueSchema, { kind: { case: "int64Value", value: n } });
const strRow = (s: string): RowValue =>
  create(RowValueSchema, { kind: { case: "stringValue", value: s } });
const valueRow = (v: Value): RowValue =>
  create(RowValueSchema, { kind: { case: "valueValue", value: v } });

const sql = (result: QueryResult, engine: Engine): string =>
  TEXT_DECODER.decode(serializeSQL(result, engine));

describe("serializeSQL", () => {
  it("MySQL uses backtick identifiers and single-quoted string literals with `'` doubled", () => {
    const r = create(QueryResultSchema, {
      columnNames: ["id", "name"],
      rows: [rowOf(intRow(1n), strRow("it's"))],
    });
    expect(sql(r, Engine.MYSQL)).toBe(
      "INSERT INTO `<table_name>` (`id`,`name`) VALUES (1,'it''s');"
    );
  });

  it("Postgres uses double-quote identifiers and pq.QuoteLiteral for strings", () => {
    const r = create(QueryResultSchema, {
      columnNames: ["id", "name"],
      rows: [
        rowOf(intRow(1n), strRow("plain")),
        // Backslash in PG triggers the `E'...'` escape-string syntax.
        rowOf(intRow(2n), strRow("a\\b")),
      ],
    });
    expect(sql(r, Engine.POSTGRES)).toBe(
      `INSERT INTO "<table_name>" ("id","name") VALUES (1,'plain');\nINSERT INTO "<table_name>" ("id","name") VALUES (2, E'a\\\\b');`
    );
  });

  it("structpb cell uses engine-aware SQL string quoting (single-quote, embedded `'` doubled), not CSV-style", () => {
    const struct = create(ValueSchema, {
      kind: {
        case: "structValue",
        value: create(StructSchema, {
          fields: {
            a: create(ValueSchema, { kind: { case: "numberValue", value: 1 } }),
            b: create(ValueSchema, {
              kind: { case: "stringValue", value: "x" },
            }),
          },
        }),
      },
    });
    const r = create(QueryResultSchema, {
      columnNames: ["v"],
      rows: [rowOf(valueRow(struct))],
    });
    // Postgres: pq.QuoteLiteral wraps in `'...'`. No backslash in payload, so no `E'...'`.
    expect(sql(r, Engine.POSTGRES)).toBe(
      `INSERT INTO "<table_name>" ("v") VALUES ('{"a":1,"b":"x"}');`
    );
    // MySQL: non-PG engines use Go's strconv.Quote-style escape inside the
    // single-quoted literal, which escapes embedded `"` as `\"`. Payload has
    // no `'`, so the only escapes come from the JSON-as-string `"` chars.
    expect(sql(r, Engine.MYSQL)).toBe(
      'INSERT INTO `<table_name>` (`v`) VALUES (\'{\\"a\\":1,\\"b\\":\\"x\\"}\');'
    );
  });

  it("throws UnsupportedFormat for engines without a quote-character mapping", () => {
    const r = create(QueryResultSchema, {
      columnNames: ["a"],
      rows: [rowOf(intRow(1n))],
    });
    expect(() => serializeSQL(r, Engine.ENGINE_UNSPECIFIED)).toThrow(
      /UnsupportedFormat|engine/i
    );
  });
});

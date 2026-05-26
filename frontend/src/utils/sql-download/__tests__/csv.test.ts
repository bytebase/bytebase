import { create } from "@bufbuild/protobuf";
import { StructSchema, type Value, ValueSchema } from "@bufbuild/protobuf/wkt";
import { describe, expect, it } from "vitest";
import {
  type QueryResult,
  QueryResultSchema,
  QueryRowSchema,
  type RowValue,
  RowValueSchema,
} from "@/types/proto-es/v1/sql_service_pb";
import { serializeCSV } from "../formats/csv";

const TEXT_DECODER = new TextDecoder("utf-8");

const rowOf = (...values: RowValue[]) => create(QueryRowSchema, { values });
const intRow = (n: bigint): RowValue =>
  create(RowValueSchema, { kind: { case: "int64Value", value: n } });
const strRow = (s: string): RowValue =>
  create(RowValueSchema, { kind: { case: "stringValue", value: s } });
const f32Row = (n: number): RowValue =>
  create(RowValueSchema, { kind: { case: "floatValue", value: n } });
const f64Row = (n: number): RowValue =>
  create(RowValueSchema, { kind: { case: "doubleValue", value: n } });
const valueRow = (v: Value): RowValue =>
  create(RowValueSchema, { kind: { case: "valueValue", value: v } });

const csv = (result: QueryResult): string =>
  TEXT_DECODER.decode(serializeCSV(result));

describe("serializeCSV", () => {
  it("emits header-only output for an empty result", () => {
    const r = create(QueryResultSchema, { columnNames: ["id", "name"] });
    expect(csv(r)).toBe("id,name\n");
  });

  it("emits basic ASCII rows (every string cell unconditionally CSV-quoted)", () => {
    const r = create(QueryResultSchema, {
      columnNames: ["id", "name"],
      rows: [
        rowOf(intRow(1n), strRow("Alice")),
        rowOf(intRow(2n), strRow("Bob")),
      ],
    });
    expect(csv(r)).toBe('id,name\n1,"Alice"\n2,"Bob"');
  });

  it('doubles embedded `"` inside CSV-quoted strings and preserves newline/comma literals', () => {
    const r = create(QueryResultSchema, {
      columnNames: ["s"],
      rows: [rowOf(strRow(`a"b`)), rowOf(strRow("a,b")), rowOf(strRow("a\nb"))],
    });
    expect(csv(r)).toBe('s\n"a""b"\n"a,b"\n"a\nb"');
  });

  it("emits float32 via formatFloat32 (never exponential) and float64 native", () => {
    const r = create(QueryResultSchema, {
      columnNames: ["f32", "f64"],
      rows: [rowOf(f32Row(1.5), f64Row(1e21))],
    });
    // f32: shortest-roundtrip 'f' verb; f64: JS native exponential at >= 1e21.
    expect(csv(r)).toBe("f32,f64\n1.5,1e+21");
  });

  it("emits structpb cells as CSV-quoted JSON (Tier 2)", () => {
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
    // JSON.stringify preserves insertion order; CSV escapes embedded `"` to `""`.
    expect(csv(r)).toBe(`v\n"{""a"":1,""b"":""x""}"`);
  });
});

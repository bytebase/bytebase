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
import { serializeJSON } from "../formats/json";

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

const json = (result: QueryResult): string =>
  TEXT_DECODER.decode(serializeJSON(result));

describe("serializeJSON", () => {
  it("emits []  for an empty result", () => {
    expect(json(create(QueryResultSchema, { columnNames: ["a"] }))).toBe("[]");
  });

  it("emits basic ASCII rows with column-declaration key order", () => {
    const r = create(QueryResultSchema, {
      columnNames: ["id", "name"],
      rows: [rowOf(intRow(1n), strRow("Alice"))],
    });
    expect(json(r)).toBe(`[
  {
    "id": 1,
    "name": "Alice"
  }
]`);
  });

  it("uses JS-native JSON string escapes (no HTML escape)", () => {
    const r = create(QueryResultSchema, {
      columnNames: ["s"],
      rows: [rowOf(strRow(`<a>&\n"`))],
    });
    expect(json(r)).toBe(`[
  {
    "s": "<a>&\\n\\""
  }
]`);
  });

  it("emits float32 via formatFloat32 and float64 via JS native", () => {
    const r = create(QueryResultSchema, {
      columnNames: ["f32", "f64"],
      rows: [rowOf(f32Row(1.5), f64Row(1e-7))],
    });
    expect(json(r)).toBe(`[
  {
    "f32": 1.5,
    "f64": 1e-7
  }
]`);
  });

  it("emits structpb cells as JSON-as-string (Tier 2)", () => {
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
    expect(json(r)).toBe(`[
  {
    "v": "{\\"a\\":1,\\"b\\":\\"x\\"}"
  }
]`);
  });

  it("emits NaN / +Inf / -Inf as null (matches JSON.stringify)", () => {
    const r = create(QueryResultSchema, {
      columnNames: ["nan", "posinf", "neginf"],
      rows: [
        rowOf(
          f64Row(Number.NaN),
          f64Row(Number.POSITIVE_INFINITY),
          f64Row(Number.NEGATIVE_INFINITY)
        ),
      ],
    });
    expect(json(r)).toBe(`[
  {
    "nan": null,
    "posinf": null,
    "neginf": null
  }
]`);
  });

  it("emits int64 / uint64 as decimal integer tokens via bigint.toString()", () => {
    const r = create(QueryResultSchema, {
      columnNames: ["i"],
      rows: [
        rowOf(intRow(-9223372036854775808n)),
        rowOf(intRow(9223372036854775807n)),
      ],
    });
    expect(json(r)).toBe(`[
  {
    "i": -9223372036854775808
  },
  {
    "i": 9223372036854775807
  }
]`);
  });
});

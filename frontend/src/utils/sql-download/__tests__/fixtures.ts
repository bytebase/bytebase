import { create } from "@bufbuild/protobuf";
import {
  ListValueSchema,
  StructSchema,
  TimestampSchema,
  type Value,
  ValueSchema,
} from "@bufbuild/protobuf/wkt";
import {
  type QueryResult,
  QueryResultSchema,
  QueryRowSchema,
  type RowValue,
  RowValue_TimestampSchema,
  RowValue_TimestampTZSchema,
  RowValueSchema,
} from "@/types/proto-es/v1/sql_service_pb";

const intRow = (n: bigint) =>
  create(RowValueSchema, { kind: { case: "int64Value", value: n } });
const i32Row = (n: number) =>
  create(RowValueSchema, { kind: { case: "int32Value", value: n } });
const u32Row = (n: number) =>
  create(RowValueSchema, { kind: { case: "uint32Value", value: n } });
const u64Row = (n: bigint) =>
  create(RowValueSchema, { kind: { case: "uint64Value", value: n } });
const f32Row = (n: number) =>
  create(RowValueSchema, { kind: { case: "floatValue", value: n } });
const f64Row = (n: number) =>
  create(RowValueSchema, { kind: { case: "doubleValue", value: n } });
const strRow = (s: string) =>
  create(RowValueSchema, { kind: { case: "stringValue", value: s } });
const bytesRow = (b: Uint8Array) =>
  create(RowValueSchema, { kind: { case: "bytesValue", value: b } });
const nullRow = () =>
  create(RowValueSchema, { kind: { case: "nullValue", value: 0 } });
const tsRow = (seconds: bigint, nanos: number) =>
  create(RowValueSchema, {
    kind: {
      case: "timestampValue",
      value: create(RowValue_TimestampSchema, {
        googleTimestamp: create(TimestampSchema, { seconds, nanos }),
      }),
    },
  });
const tstzRow = (
  seconds: bigint,
  nanos: number,
  zone: string,
  offset: number
) =>
  create(RowValueSchema, {
    kind: {
      case: "timestampTzValue",
      value: create(RowValue_TimestampTZSchema, {
        googleTimestamp: create(TimestampSchema, { seconds, nanos }),
        zone,
        offset,
      }),
    },
  });
const valueRow = (v: Value) =>
  create(RowValueSchema, { kind: { case: "valueValue", value: v } });

const allBytes = () => {
  const out = new Uint8Array(256);
  for (let i = 0; i < 256; i++) out[i] = i;
  return out;
};

const row = (...values: RowValue[]) => create(QueryRowSchema, { values });

/**
 * Mirror of `downloadFixtures()` in
 * backend/component/export/download_goldens_fixtures.go. Same ids, same data.
 * If you add a fixture here, add it there too (and re-run `-update`).
 */
export const FIXTURES: Record<string, QueryResult> = {
  empty_no_columns_no_rows: create(QueryResultSchema, {}),
  empty_columns_no_rows: create(QueryResultSchema, {
    columnNames: ["a", "b"],
    columnTypeNames: ["INT", "TEXT"],
  }),
  ascii_basic: create(QueryResultSchema, {
    columnNames: ["id", "name"],
    columnTypeNames: ["INT", "TEXT"],
    rows: [row(intRow(1n), strRow("Alice")), row(intRow(2n), strRow("Bob"))],
  }),
  string_escapes: create(QueryResultSchema, {
    columnNames: ["s"],
    columnTypeNames: ["TEXT"],
    rows: [
      row(strRow("it's")),
      row(strRow(`a"b`)),
      row(strRow("a\\b")),
      row(strRow("a\nb")),
      row(strRow("a\r\nb")),
      row(strRow("a\tb")),
      row(strRow("a\x00b")),
      row(strRow("a\x1bb")),
      row(strRow("中文 👍")),
    ],
  }),
  bytes_full_range: create(QueryResultSchema, {
    columnNames: ["b"],
    columnTypeNames: ["BLOB"],
    rows: [
      row(bytesRow(new Uint8Array())),
      row(bytesRow(new Uint8Array([0]))),
      row(bytesRow(allBytes())),
    ],
  }),
  ints_edges: create(QueryResultSchema, {
    columnNames: ["i32", "i64", "u32", "u64"],
    columnTypeNames: ["INT", "BIGINT", "INT UNSIGNED", "BIGINT UNSIGNED"],
    rows: [
      row(i32Row(0), intRow(0n), u32Row(0), u64Row(0n)),
      row(
        i32Row(-2147483648),
        intRow(-9223372036854775808n),
        u32Row(0),
        u64Row(0n)
      ),
      row(
        i32Row(2147483647),
        intRow(9223372036854775807n),
        u32Row(4294967295),
        u64Row(18446744073709551615n)
      ),
    ],
  }),
  // B14: pin extreme float magnitudes through goldens.
  floats_extreme_magnitudes: create(QueryResultSchema, {
    columnNames: ["f64", "f32"],
    columnTypeNames: ["DOUBLE", "FLOAT"],
    rows: [
      row(f64Row(Number.MAX_VALUE), f32Row(3.4028234663852886e38)), // math.MaxFloat32
      row(
        f64Row(Number.MIN_VALUE),
        f32Row(1.401298464324817e-45) // math.SmallestNonzeroFloat32
      ),
      row(f64Row(1e21), f32Row(1e10)),
      row(f64Row(1e-7), f32Row(1e-7)),
      // The two literals below are deliberate boundary values for the JSON
      // 'f' / 'e' threshold — float64 round-trips them and the goldens lock
      // the exact decimal output. Go's parser sees the same rounded float64.
      // biome-ignore lint/correctness/noPrecisionLoss: deliberate boundary value
      row(f64Row(9.999999999999998e20), f32Row(0)),
      row(f64Row(1.0000000000000002e-6), f32Row(0)),
    ],
  }),
  floats_finite_edges: create(QueryResultSchema, {
    columnNames: ["f32", "f64"],
    columnTypeNames: ["FLOAT", "DOUBLE"],
    rows: [
      row(f32Row(0), f64Row(0)),
      row(f32Row(-0), f64Row(-0)),
      row(f32Row(1.5), f64Row(1.5)),
      row(f32Row(1e10), f64Row(1e21)),
      row(f32Row(1e-7), f64Row(1e-7)),
    ],
  }),
  floats_special_skip_json: create(QueryResultSchema, {
    columnNames: ["f32", "f64"],
    columnTypeNames: ["FLOAT", "DOUBLE"],
    rows: [
      row(f32Row(Number.NaN), f64Row(Number.NaN)),
      row(f32Row(Infinity), f64Row(Infinity)),
      row(f32Row(-Infinity), f64Row(-Infinity)),
    ],
  }),
  timestamps: create(QueryResultSchema, {
    columnNames: ["ts", "tstz"],
    columnTypeNames: ["TIMESTAMP", "TIMESTAMPTZ"],
    rows: [
      row(tsRow(0n, 0), tstzRow(0n, 0, "UTC", 0)),
      row(
        tsRow(1700000000n, 123456789),
        tstzRow(1700000000n, 500000000, "JST", 9 * 3600)
      ),
      row(
        tsRow(1700000000n, 123456000),
        tstzRow(1700000000n, 0, "PDT", -7 * 3600)
      ),
    ],
  }),
  structpb_kinds: create(QueryResultSchema, {
    columnNames: ["v"],
    columnTypeNames: ["VARIANT"],
    rows: [
      row(
        valueRow(create(ValueSchema, { kind: { case: "nullValue", value: 0 } }))
      ),
      row(
        valueRow(
          create(ValueSchema, { kind: { case: "stringValue", value: "ab" } })
        )
      ),
      row(
        valueRow(
          create(ValueSchema, { kind: { case: "numberValue", value: 1.5 } })
        )
      ),
      row(
        valueRow(
          create(ValueSchema, { kind: { case: "boolValue", value: true } })
        )
      ),
      row(
        valueRow(
          create(ValueSchema, {
            kind: {
              case: "listValue",
              value: create(ListValueSchema, {
                values: [
                  create(ValueSchema, {
                    kind: { case: "numberValue", value: 1 },
                  }),
                  create(ValueSchema, {
                    kind: { case: "stringValue", value: "x" },
                  }),
                ],
              }),
            },
          })
        )
      ),
      row(
        valueRow(
          create(ValueSchema, {
            kind: {
              case: "structValue",
              value: create(StructSchema, {
                fields: {
                  b: create(ValueSchema, {
                    kind: { case: "numberValue", value: 2 },
                  }),
                  a: create(ValueSchema, {
                    kind: { case: "numberValue", value: 1 },
                  }),
                },
              }),
            },
          })
        )
      ),
    ],
  }),
  nulls_only: create(QueryResultSchema, {
    columnNames: ["x", "y"],
    columnTypeNames: ["TEXT", "INT"],
    rows: [row(nullRow(), nullRow())],
  }),
  // TODO(B2/B3): Captures CURRENT (non-escaping) behavior for column names
  // containing identifier/header metacharacters. Both backend and TS produce
  // the same byte-for-byte broken output today. Fixing the underlying gap
  // requires a coordinated change with regenerated goldens.
  column_name_quotes_my: create(QueryResultSchema, {
    columnNames: ["a`b", "c,d", 'e"f'],
    columnTypeNames: ["INT", "TEXT", "TEXT"],
    rows: [row(intRow(1n), strRow("x"), strRow("y"))],
  }),
  column_name_quotes_pg: create(QueryResultSchema, {
    columnNames: ['a"b', "c,d", "e\nf"],
    columnTypeNames: ["INT", "TEXT", "TEXT"],
    rows: [row(intRow(1n), strRow("x"), strRow("y"))],
  }),
  // Exercises B1 (prototext escaping for structpb in XLSX/JSON).
  structpb_string_with_quotes: create(QueryResultSchema, {
    columnNames: ["v"],
    columnTypeNames: ["VARIANT"],
    rows: [
      row(
        valueRow(
          create(ValueSchema, { kind: { case: "stringValue", value: `a"b` } })
        )
      ),
      row(
        valueRow(
          create(ValueSchema, { kind: { case: "stringValue", value: "a\\b" } })
        )
      ),
      row(
        valueRow(
          create(ValueSchema, { kind: { case: "stringValue", value: "a\nb" } })
        )
      ),
      row(
        valueRow(
          create(ValueSchema, { kind: { case: "stringValue", value: "a\tb" } })
        )
      ),
      row(
        valueRow(
          create(ValueSchema, {
            kind: { case: "stringValue", value: "a\x00b" },
          })
        )
      ),
      // C1 controls — Go prototext emits \uHHHH (4-digit lowercase hex).
      row(
        valueRow(
          create(ValueSchema, {
            kind: { case: "stringValue", value: "a\x80b" },
          })
        )
      ),
      row(
        valueRow(
          create(ValueSchema, {
            kind: { case: "stringValue", value: "a\x9fb" },
          })
        )
      ),
      row(
        valueRow(
          create(ValueSchema, {
            kind: {
              case: "structValue",
              value: create(StructSchema, {
                fields: {
                  [`k"y`]: create(ValueSchema, {
                    kind: { case: "stringValue", value: `a"b` },
                  }),
                },
              }),
            },
          })
        )
      ),
    ],
  }),
};

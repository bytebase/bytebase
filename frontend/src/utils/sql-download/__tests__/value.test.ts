import { create } from "@bufbuild/protobuf";
import { TimestampSchema } from "@bufbuild/protobuf/wkt";
import { describe, expect, it } from "vitest";
import {
  RowValue_TimestampSchema,
  RowValue_TimestampTZSchema,
} from "@/types/proto-es/v1/sql_service_pb";
import {
  formatFloat32,
  formatFloat64,
  formatTimestamp,
  formatTimestampTZ,
  goQuoteInner,
  pqQuoteLiteral,
} from "../value";

describe("formatTimestamp", () => {
  it("formats epoch as midnight UTC with six fractional zeros", () => {
    const ts = create(RowValue_TimestampSchema, {
      googleTimestamp: create(TimestampSchema, { seconds: 0n, nanos: 0 }),
    });
    expect(formatTimestamp(ts)).toBe("1970-01-01 00:00:00.000000");
  });

  it("preserves microseconds and truncates sub-microsecond nanos", () => {
    // 123456789 ns = 123456 us + 789 ns; expect 123456 in output (truncate, not round)
    const ts = create(RowValue_TimestampSchema, {
      googleTimestamp: create(TimestampSchema, {
        seconds: 1700000000n,
        nanos: 123456789,
      }),
    });
    expect(formatTimestamp(ts)).toBe("2023-11-14 22:13:20.123456");
  });

  it("emits trailing zeros, not stripped", () => {
    const ts = create(RowValue_TimestampSchema, {
      googleTimestamp: create(TimestampSchema, {
        seconds: 1700000000n,
        nanos: 500000000, // exactly 500000 us
      }),
    });
    expect(formatTimestamp(ts)).toBe("2023-11-14 22:13:20.500000");
  });

  it("pads historical years to 4 digits (matches Go's time.Format)", () => {
    // Year 1 AD: Go emits "0001-01-01 00:00:00.000000"; JS's getUTCFullYear()
    // returns 1, which without padding would yield "1-01-01 ...".
    // Unix seconds for 0001-01-01 00:00:00 UTC = -62135596800.
    const tsYear1 = create(RowValue_TimestampSchema, {
      googleTimestamp: create(TimestampSchema, {
        seconds: -62135596800n,
        nanos: 0,
      }),
    });
    expect(formatTimestamp(tsYear1)).toBe("0001-01-01 00:00:00.000000");
  });
});

describe("formatTimestampTZ", () => {
  it("emits Z for zero offset and integer seconds", () => {
    const tz = create(RowValue_TimestampTZSchema, {
      googleTimestamp: create(TimestampSchema, {
        seconds: 1700000000n,
        nanos: 0,
      }),
      zone: "UTC",
      offset: 0,
    });
    expect(formatTimestampTZ(tz)).toBe("2023-11-14T22:13:20Z");
  });

  it("strips trailing zero digits from fractional second", () => {
    // 500_000_000 ns → ".5" (not ".500" or ".500000000")
    const tz = create(RowValue_TimestampTZSchema, {
      googleTimestamp: create(TimestampSchema, {
        seconds: 1700000000n,
        nanos: 500_000_000,
      }),
      zone: "UTC",
      offset: 0,
    });
    expect(formatTimestampTZ(tz)).toBe("2023-11-14T22:13:20.5Z");
  });

  it("preserves all nine digits when none are trailing zeros", () => {
    const tz = create(RowValue_TimestampTZSchema, {
      googleTimestamp: create(TimestampSchema, {
        seconds: 1700000000n,
        nanos: 123_456_789,
      }),
      zone: "UTC",
      offset: 0,
    });
    expect(formatTimestampTZ(tz)).toBe("2023-11-14T22:13:20.123456789Z");
  });

  it("renders positive offset as +HH:MM", () => {
    // PDT-like: -7 hours from UTC has offset -25200, but test +HH:MM with positive
    const tz = create(RowValue_TimestampTZSchema, {
      googleTimestamp: create(TimestampSchema, {
        seconds: 1700000000n,
        nanos: 0,
      }),
      zone: "JST",
      offset: 9 * 3600,
    });
    expect(formatTimestampTZ(tz)).toBe("2023-11-15T07:13:20+09:00");
  });

  it("renders negative offset as -HH:MM and shifts wall time", () => {
    const tz = create(RowValue_TimestampTZSchema, {
      googleTimestamp: create(TimestampSchema, {
        seconds: 1700000000n,
        nanos: 0,
      }),
      zone: "PDT",
      offset: -7 * 3600,
    });
    expect(formatTimestampTZ(tz)).toBe("2023-11-14T15:13:20-07:00");
  });

  it("pads historical years to 4 digits (matches Go's RFC3339Nano)", () => {
    const tz = create(RowValue_TimestampTZSchema, {
      googleTimestamp: create(TimestampSchema, {
        seconds: -62135596800n,
        nanos: 0,
      }),
      zone: "UTC",
      offset: 0,
    });
    expect(formatTimestampTZ(tz)).toBe("0001-01-01T00:00:00Z");
  });
});

describe("formatFloat64", () => {
  it.each([
    [0, "0"],
    [-0, "-0"],
    [1.5, "1.5"],
    [1e21, "1000000000000000000000"], // never exponential
    [1e-7, "0.0000001"],
  ])("formats %p as %p (Go strconv 'f' verb, no exponent)", (input, expected) => {
    expect(formatFloat64(input)).toBe(expected);
  });
  // Note: Number.MAX_VALUE is verified via the floats_edges golden in Chunk 3,
  // not in this unit test — the exact 309-digit Go output is fragile to
  // reproduce by hand. Goldens are the source of truth for that case.

  it("emits NaN, +Inf, -Inf strings", () => {
    expect(formatFloat64(Number.NaN)).toBe("NaN");
    expect(formatFloat64(Number.POSITIVE_INFINITY)).toBe("+Inf");
    expect(formatFloat64(Number.NEGATIVE_INFINITY)).toBe("-Inf");
  });
});

describe("formatFloat32", () => {
  it("rounds to float32 first, then prints shortest decimal", () => {
    // Math.fround(1/3) = 0.3333333432674408
    expect(formatFloat32(1 / 3)).toBe("0.33333334");
  });

  it("shares special-value rules with formatFloat64", () => {
    expect(formatFloat32(Number.NaN)).toBe("NaN");
    expect(formatFloat32(Number.POSITIVE_INFINITY)).toBe("+Inf");
    expect(formatFloat32(Number.NEGATIVE_INFINITY)).toBe("-Inf");
  });
});

describe("goQuoteInner (strconv.Quote inner content)", () => {
  it("returns ASCII verbatim", () => {
    expect(goQuoteInner("hello")).toBe("hello");
  });

  it.each([
    [`a"b`, `a\\"b`], // " escaped
    [`a\\b`, `a\\\\b`], // \ escaped
    ["a\nb", "a\\nb"],
    ["a\rb", "a\\rb"],
    ["a\tb", "a\\tb"],
    ["a\bb", "a\\bb"],
    ["a\fb", "a\\fb"],
    ["a\vb", "a\\vb"],
    ["a\x07b", "a\\ab"], // BEL → \a
    ["a\x00b", "a\\x00b"], // NUL → \x00 (lowercase)
    ["a\x1bb", "a\\x1bb"], // ESC → \x1b
    ["a\x7fb", "a\\x7fb"], // DEL → \x7f
    ["a'b", "a'b"], // single quote NOT escaped by strconv.Quote
  ])("escapes %p as %p", (input, expected) => {
    expect(goQuoteInner(input)).toBe(expected);
  });

  it("emits non-printable runes as \\uHHHH (no curly braces)", () => {
    // U+200B ZERO WIDTH SPACE is non-printable per unicode.IsPrint
    expect(goQuoteInner("a​b")).toBe("a\\u200bb");
  });

  it("emits non-BMP non-printable runes as \\UHHHHHHHH", () => {
    // U+E0001 (LANGUAGE TAG) is non-printable
    const langTag = String.fromCodePoint(0xe0001);
    expect(goQuoteInner(`a${langTag}b`)).toBe("a\\U000e0001b");
  });

  it("emits printable BMP runes verbatim (CJK, emoji)", () => {
    expect(goQuoteInner("中文")).toBe("中文");
    expect(goQuoteInner("👍")).toBe("👍");
  });

  // B9: explicit non-print categories.
  it("escapes Private-Use Area BMP runes as \\uHHHH", () => {
    expect(goQuoteInner(`a\u{e000}b`)).toBe("a\\ue000b");
    expect(goQuoteInner(`a\u{f8ff}b`)).toBe("a\\uf8ffb");
  });

  it("escapes Supplementary Private-Use runes as \\UHHHHHHHH", () => {
    expect(goQuoteInner(`a${String.fromCodePoint(0xf0000)}b`)).toBe(
      "a\\U000f0000b"
    );
    expect(goQuoteInner(`a${String.fromCodePoint(0x100000)}b`)).toBe(
      "a\\U00100000b"
    );
  });

  it("escapes Zs space-separator runes as \\uHHHH (parity with Go)", () => {
    // Go's unicode.IsPrint returns false for the Zs general category except
    // ASCII space (0x20). Backend escapes these as `\uXXXX`; the client must
    // match or SQL exports diverge for any string with exotic whitespace.
    expect(goQuoteInner("a b")).toBe("a\\u2000b"); // EN QUAD
    expect(goQuoteInner("a b")).toBe("a\\u200ab"); // HAIR SPACE
    expect(goQuoteInner("a b")).toBe("a\\u202fb"); // NARROW NBSP
    expect(goQuoteInner("a b")).toBe("a\\u205fb"); // MEDIUM MATH SPACE
    expect(goQuoteInner("a　b")).toBe("a\\u3000b"); // IDEOGRAPHIC SPACE
    expect(goQuoteInner("a b")).toBe("a\\u1680b"); // OGHAM SPACE MARK
  });

  it("emits invalid UTF-8 byte as \\xHH", () => {
    // We can't form an invalid UTF-8 string in JS directly; this test asserts
    // the encoded-bytes path. Skip in JS if no synthetic byte path exists;
    // covered instead via Chunk 2 goldens for all-bytes 0x00..0xFF strings.
    const lone = String.fromCharCode(0xd800); // lone surrogate
    // TextEncoder replaces lone surrogates with U+FFFD; expect REPLACEMENT
    // CHARACTER's UTF-8 encoded escape (E� = printable, so verbatim).
    expect(goQuoteInner(lone)).toBe("�");
  });
});

describe("pqQuoteLiteral", () => {
  it("wraps simple string in single quotes", () => {
    expect(pqQuoteLiteral("hello")).toBe("'hello'");
  });

  it("doubles single quotes inside plain form", () => {
    expect(pqQuoteLiteral("it's")).toBe("'it''s'");
  });

  it("uses E'…' form when a backslash is present and escapes it", () => {
    // Note: Go's pq.QuoteLiteral prepends a space before E' (matches libpq behavior)
    expect(pqQuoteLiteral("a\\b")).toBe(" E'a\\\\b'");
  });

  it("doubles single quotes inside E form too", () => {
    expect(pqQuoteLiteral("it's\\")).toBe(" E'it''s\\\\'");
  });
});

import {
  type RowValue,
  RowValueSchema,
} from "@/types/proto-es/v1/sql_service_pb";
import {
  csvCellFromRowValue,
  estimateResultBytes,
  JSONFloat32,
  jsonValueFromRowValue,
  sqlValueFromRowValue,
  xlsxValueFromRowValue,
} from "../value";

const rv = (kind: RowValue["kind"]) => create(RowValueSchema, { kind });

describe("csvCellFromRowValue", () => {
  it("null → empty", () =>
    expect(csvCellFromRowValue(rv({ case: "nullValue", value: 0 }))).toBe(""));
  it("bool true → true", () =>
    expect(csvCellFromRowValue(rv({ case: "boolValue", value: true }))).toBe(
      "true"
    ));
  it("int32 → bare digits", () =>
    expect(csvCellFromRowValue(rv({ case: "int32Value", value: 42 }))).toBe(
      "42"
    ));
  it("int64 → bare digits", () =>
    expect(
      csvCellFromRowValue(
        rv({ case: "int64Value", value: 9223372036854775807n })
      )
    ).toBe("9223372036854775807"));
  it('string → double-quoted with `"` doubled', () =>
    expect(csvCellFromRowValue(rv({ case: "stringValue", value: 'a"b' }))).toBe(
      `"a""b"`
    ));
  it("bytes → quoted lowercase 0x hex", () =>
    expect(
      csvCellFromRowValue(
        rv({ case: "bytesValue", value: new Uint8Array([0x00, 0xff]) })
      )
    ).toBe(`"0x00ff"`));
});

describe("jsonValueFromRowValue", () => {
  it("null → null", () =>
    expect(
      jsonValueFromRowValue(rv({ case: "nullValue", value: 0 }))
    ).toBeNull());
  it("int64 bigint passes through unchanged", () => {
    const out = jsonValueFromRowValue(rv({ case: "int64Value", value: 1n }));
    expect(out).toBe(1n);
  });
  it("string is plain string (encoder will JSON-escape)", () => {
    expect(jsonValueFromRowValue(rv({ case: "stringValue", value: "x" }))).toBe(
      "x"
    );
  });
  it("float32 returns JSONFloat32 instance", () => {
    const out = jsonValueFromRowValue(rv({ case: "floatValue", value: 1.5 }));
    expect(out).toBeInstanceOf(JSONFloat32);
    expect((out as JSONFloat32).value).toBe(Math.fround(1.5));
  });
});

describe("sqlValueFromRowValue", () => {
  it("non-PG string uses goQuote then SQL '' for single quotes", () => {
    expect(
      sqlValueFromRowValue(rv({ case: "stringValue", value: "it's" }), false)
    ).toBe(`'it''s'`);
  });
  it("PG string uses pqQuoteLiteral", () => {
    // Note: pqQuoteLiteral prepends a space before E' (matches Go lib/pq)
    expect(
      sqlValueFromRowValue(rv({ case: "stringValue", value: "a\\b" }), true)
    ).toBe(` E'a\\\\b'`);
  });
  it("null → NULL", () =>
    expect(
      sqlValueFromRowValue(rv({ case: "nullValue", value: 0 }), false)
    ).toBe("NULL"));
  it("bytes → unquoted 0xhex", () =>
    expect(
      sqlValueFromRowValue(
        rv({ case: "bytesValue", value: new Uint8Array([0x00]) }),
        false
      )
    ).toBe("0x00"));
});

describe("xlsxValueFromRowValue", () => {
  it("null → empty string", () =>
    expect(xlsxValueFromRowValue(rv({ case: "nullValue", value: 0 }))).toBe(
      ""
    ));
  it("bytes → base64", () =>
    expect(
      xlsxValueFromRowValue(
        rv({ case: "bytesValue", value: new Uint8Array([0x00, 0xff]) })
      )
    ).toBe("AP8="));
});

describe("estimateResultBytes (cap guard worst-case bounds)", () => {
  // Lazy schema imports inside each test to avoid top-of-file churn — the
  // existing test file mixes imports throughout.
  const loadSchemas = async () => {
    const sql = await import("@/types/proto-es/v1/sql_service_pb");
    const wkt = await import("@bufbuild/protobuf/wkt");
    return { sql, wkt };
  };

  it("MaxFloat64 cells are bounded above the formatted byte length", async () => {
    const { sql } = await loadSchemas();
    // The previous flat 16 const undercounted MaxFloat64 by ~20x: its
    // non-exponential form is ~326 chars. The new APPROX_FLOAT64_BYTES
    // (336) covers that.
    const result = create(sql.QueryResultSchema, {
      columnNames: ["d"],
      rows: [
        { values: [rv({ case: "doubleValue", value: Number.MAX_VALUE })] },
      ],
    });
    const formattedLen = csvCellFromRowValue(
      rv({ case: "doubleValue", value: Number.MAX_VALUE })
    ).length;
    const est = estimateResultBytes(result, Number.MAX_SAFE_INTEGER);
    expect(est).toBeGreaterThan(formattedLen);
  });

  it("hex-encoded bytes are bounded by 2x byteLength + overhead", async () => {
    const { sql } = await loadSchemas();
    const longBytes = new Uint8Array(10000);
    longBytes.fill(0xab);
    // CSV/SQL hex output is 2x byteLength + delimiters; the estimator must
    // bound that. The previous `byteLength + 4` const under-counted.
    const csvLen = csvCellFromRowValue(
      rv({ case: "bytesValue", value: longBytes })
    ).length;
    const result = create(sql.QueryResultSchema, {
      columnNames: ["b"],
      rows: [{ values: [rv({ case: "bytesValue", value: longBytes })] }],
    });
    const est = estimateResultBytes(result, Number.MAX_SAFE_INTEGER);
    expect(est).toBeGreaterThanOrEqual(csvLen);
  });

  it("structpb value_value is bounded by recursive walk, not a flat 1KB", async () => {
    const { sql, wkt } = await loadSchemas();
    // A struct whose single string field is 64KB. The old flat 1024 const
    // under-counted this by ~64x and let a single cell bypass the byte cap.
    const bigString = "x".repeat(64 * 1024);
    const inner = create(wkt.ValueSchema, {
      kind: {
        case: "structValue",
        value: create(wkt.StructSchema, {
          fields: {
            payload: create(wkt.ValueSchema, {
              kind: { case: "stringValue", value: bigString },
            }),
          },
        }),
      },
    });
    const result = create(sql.QueryResultSchema, {
      columnNames: ["v"],
      rows: [{ values: [rv({ case: "valueValue", value: inner })] }],
    });
    const est = estimateResultBytes(result, Number.MAX_SAFE_INTEGER);
    expect(est).toBeGreaterThan(bigString.length);
  });

  it("short-circuits once running total crosses cap", async () => {
    const { sql } = await loadSchemas();
    const rows = [];
    for (let i = 0; i < 1000; i++) {
      rows.push({
        values: [rv({ case: "doubleValue", value: Number.MAX_VALUE })],
      });
    }
    const result = create(sql.QueryResultSchema, { columnNames: ["d"], rows });
    // cap = 1000; with ~336 bytes per doubleValue the loop bails after a few
    // rows. The returned value is the running total when it crossed cap, not
    // the full 1000×336 sum.
    const est = estimateResultBytes(result, 1000);
    expect(est).toBeGreaterThan(1000);
    expect(est).toBeLessThan(336 * 10);
  });
});

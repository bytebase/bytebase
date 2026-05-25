import { describe, expect, it } from "vitest";
import { encodeIndentedJSON } from "../formats/json";

describe("encodeIndentedJSON (Go json.MarshalIndent parity)", () => {
  it("encodes empty array as []", () => {
    expect(encodeIndentedJSON([])).toBe("[]");
  });

  it("encodes empty object as {}", () => {
    expect(encodeIndentedJSON({})).toBe("{}");
  });

  it("encodes scalars matching Go", () => {
    expect(encodeIndentedJSON(null)).toBe("null");
    expect(encodeIndentedJSON(true)).toBe("true");
    expect(encodeIndentedJSON("a")).toBe('"a"');
    expect(encodeIndentedJSON(42)).toBe("42");
    expect(encodeIndentedJSON(1.5)).toBe("1.5");
  });

  it("encodes bigint as raw integer token", () => {
    expect(encodeIndentedJSON(9223372036854775807n)).toBe(
      "9223372036854775807"
    );
  });

  it('indents arrays with two spaces and "\\n"', () => {
    expect(encodeIndentedJSON([1, 2])).toBe("[\n  1,\n  2\n]");
  });

  it("indents nested objects, key-value separated by ': '", () => {
    expect(encodeIndentedJSON({ a: 1, b: { c: 2 } })).toBe(
      '{\n  "a": 1,\n  "b": {\n    "c": 2\n  }\n}'
    );
  });

  it("sorts object keys lexicographically (Go json.MarshalIndent of map)", () => {
    expect(encodeIndentedJSON({ b: 1, a: 2 })).toBe(
      '{\n  "a": 2,\n  "b": 1\n}'
    );
  });

  it("emits float64 in 'f' form below 1e21", () => {
    expect(encodeIndentedJSON(1e10)).toBe("10000000000");
  });

  it("emits float64 in 'e' form at/above 1e21", () => {
    expect(encodeIndentedJSON(1e21)).toBe("1e+21");
  });

  it("emits float64 in 'e' form below 1e-6 (with leading-zero strip)", () => {
    expect(encodeIndentedJSON(1e-7)).toBe("1e-7");
  });

  it("throws SerializationFailed on NaN/Inf", () => {
    expect(() => encodeIndentedJSON(Number.NaN)).toThrow();
    expect(() => encodeIndentedJSON(Number.POSITIVE_INFINITY)).toThrow();
    expect(() => encodeIndentedJSON(Number.NEGATIVE_INFINITY)).toThrow();
  });

  it("escapes string special chars per Go encoding/json (HTML-safe)", () => {
    // Go encoding/json escapes <, >, & by default.
    // Inputs for U+2028, U+2029, and NUL are written as `\uXXXX` escape
    // sequences so the source file stays pure ASCII and diffable.
    expect(encodeIndentedJSON("<a>")).toBe('"\\u003ca\\u003e"');
    expect(encodeIndentedJSON("a&b")).toBe('"a\\u0026b"');
    expect(encodeIndentedJSON("\u2028")).toBe('"\\u2028"');
    expect(encodeIndentedJSON("\u2029")).toBe('"\\u2029"');
    expect(encodeIndentedJSON('a"b')).toBe('"a\\"b"');
    expect(encodeIndentedJSON("a\\b")).toBe('"a\\\\b"');
    expect(encodeIndentedJSON("a\u0000b")).toBe('"a\\u0000b"');
  });
});

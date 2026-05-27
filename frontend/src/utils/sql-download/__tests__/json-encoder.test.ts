import { describe, expect, it } from "vitest";
import { encodeIndentedJSON } from "../formats/json";
import { float32 } from "../value";

// Tier 2 relaxation: the encoder no longer mirrors Go json.MarshalIndent
// (alphabetical key sort, HTML-safe escape, U+2028/2029 escape, throw on
// NaN/Inf). It now uses native JSON.stringify escape rules + JS Object
// insertion order + float32 cells via formatFloat32. The tests below cover
// the new contract.
describe("encodeIndentedJSON", () => {
  it("encodes empty containers", () => {
    expect(encodeIndentedJSON([])).toBe("[]");
    expect(encodeIndentedJSON({})).toBe("{}");
  });

  it("encodes scalars", () => {
    expect(encodeIndentedJSON(null)).toBe("null");
    expect(encodeIndentedJSON(true)).toBe("true");
    expect(encodeIndentedJSON("a")).toBe('"a"');
    expect(encodeIndentedJSON(42)).toBe("42");
    expect(encodeIndentedJSON(1.5)).toBe("1.5");
  });

  it("encodes bigint via .toString() — JSON.stringify can't take bigint natively", () => {
    expect(encodeIndentedJSON(9223372036854775807n)).toBe(
      "9223372036854775807"
    );
  });

  it("indents arrays and nested objects with two spaces", () => {
    expect(encodeIndentedJSON([1, 2])).toBe("[\n  1,\n  2\n]");
    expect(encodeIndentedJSON({ a: 1, b: { c: 2 } })).toBe(
      '{\n  "a": 1,\n  "b": {\n    "c": 2\n  }\n}'
    );
  });

  it("preserves object insertion order (NOT alphabetical sort)", () => {
    expect(encodeIndentedJSON({ b: 1, a: 2 })).toBe(
      '{\n  "b": 1,\n  "a": 2\n}'
    );
  });

  it("emits float64 via JS-native Number.toString (exponential at JS thresholds)", () => {
    expect(encodeIndentedJSON(1e10)).toBe("10000000000");
    expect(encodeIndentedJSON(1e21)).toBe("1e+21");
    expect(encodeIndentedJSON(1e-7)).toBe("1e-7");
  });

  it("emits float32 cells via formatFloat32 (always 'f' verb, byte-equal to backend CSV/SQL/XLSX)", () => {
    expect(encodeIndentedJSON(float32(1.5))).toBe("1.5");
    // MaxFloat32 stays in 'f' verb (no exponential). Backend JSON's encoding/json
    // would emit "3.4028235e+38" for the same value — diverged under Tier 2.
    expect(encodeIndentedJSON(float32(3.4028235e38))).toBe(
      "340282350000000000000000000000000000000"
    );
  });

  it("maps NaN / +Inf / -Inf to null (matches JSON.stringify)", () => {
    expect(encodeIndentedJSON(Number.NaN)).toBe("null");
    expect(encodeIndentedJSON(Number.POSITIVE_INFINITY)).toBe("null");
    expect(encodeIndentedJSON(Number.NEGATIVE_INFINITY)).toBe("null");
  });

  it("delegates string escapes to JSON.stringify — no HTML-safe `<>&` escape", () => {
    expect(encodeIndentedJSON("<a>&")).toBe('"<a>&"');
    expect(encodeIndentedJSON('a"b')).toBe('"a\\"b"');
    expect(encodeIndentedJSON("a\\b")).toBe('"a\\\\b"');
    expect(encodeIndentedJSON("a\x00b")).toBe('"a\\u0000b"');
  });
});

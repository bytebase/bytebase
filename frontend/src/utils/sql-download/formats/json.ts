import type { QueryResult } from "@/types/proto-es/v1/sql_service_pb";
import {
  formatFloat32,
  JSONFloat32,
  type JSONInput,
  jsonValueFromRowValue,
} from "../value";

const TEXT_ENCODER = new TextEncoder();

/**
 * Encode a JSONInput tree as indented JSON text. Tier-2 relaxed encoder:
 *
 *  - String escaping delegates to `JSON.stringify`, inheriting ES2019+ rules
 *    (no HTML escape; U+2028/2029 escaped as ` ` / ` `).
 *  - Object keys are emitted in insertion order — typically column order
 *    from `serializeJSON` below. (Pre-Tier-2 encoder sorted alphabetically
 *    to match Go map[string]any iteration.)
 *  - float32 cells still emit via `formatFloat32` to preserve cross-side
 *    byte-equality (the contract for float32 listed in goldens/formatters/).
 *  - float64 / int / string / bool emit via JS-native rules.
 *  - bigint (int64 / uint64 from proto) emit via `.toString()` since
 *    `JSON.stringify` throws on bigint.
 *  - NaN / ±Inf → "null" (mirrors `JSON.stringify` and JSON-spec rejection).
 */
function encode(v: JSONInput, indent: string, prefix: string): string {
  if (v === null) return "null";
  if (typeof v === "boolean") return v ? "true" : "false";
  if (typeof v === "string") return JSON.stringify(v);
  if (typeof v === "bigint") return v.toString();
  if (v instanceof JSONFloat32) {
    // `formatFloat32` emits "NaN" / "+Inf" / "-Inf" for non-finite values —
    // valid output for CSV/SQL/XLSX text cells but NOT valid JSON, so the
    // download would be a parse error. Match the bare-number guard just
    // below and emit `null` instead.
    if (!Number.isFinite(v.value)) return "null";
    return formatFloat32(v.value);
  }
  if (typeof v === "number") {
    if (!Number.isFinite(v)) return "null";
    return String(v);
  }
  if (Array.isArray(v)) {
    if (v.length === 0) return "[]";
    const inner = prefix + indent;
    return `[\n${v.map((x) => inner + encode(x, indent, inner)).join(",\n")}\n${prefix}]`;
  }
  const keys = Object.keys(v);
  if (keys.length === 0) return "{}";
  const inner = prefix + indent;
  return `{\n${keys
    .map((k) => `${inner}${JSON.stringify(k)}: ${encode(v[k], indent, inner)}`)
    .join(",\n")}\n${prefix}}`;
}

export function encodeIndentedJSON(value: JSONInput, indent = "  "): string {
  return encode(value, indent, "");
}

export function serializeJSON(result: QueryResult): Uint8Array {
  const records = result.rows.map((r) => {
    // Object.create(null) so a column literally named `__proto__` (legal SQL:
    // `SELECT 1 AS "__proto__"`) becomes an own enumerable property instead
    // of triggering Object.prototype's setter and getting silently dropped.
    const obj = Object.create(null) as { [k: string]: JSONInput };
    for (let i = 0; i < result.columnNames.length; i++) {
      obj[result.columnNames[i]] = jsonValueFromRowValue(r.values[i]);
    }
    return obj;
  });
  return TEXT_ENCODER.encode(encode(records, "  ", ""));
}

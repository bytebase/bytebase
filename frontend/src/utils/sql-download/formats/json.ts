import type { QueryResult } from "@/types/proto-es/v1/sql_service_pb";
import {
  formatJSONNumber,
  JSONFloat32,
  type JSONInput,
  jsonValueFromRowValue,
} from "../value";

const TEXT_ENCODER = new TextEncoder();
const HEX = "0123456789abcdef";

function escapeJSONString(s: string): string {
  // Mirror Go encoding/json default behavior (HTML-safe encoder used by
  // MarshalIndent): \", \\, \b, \f, \n, \r, \t for named escapes; \u00HH
  // for other ASCII control bytes; HTML-sensitive < > & escape; U+2028/9
  // also escape. Other Unicode emitted verbatim.
  let out = '"';
  for (let i = 0; i < s.length; i++) {
    const c = s.charCodeAt(i);
    if (c === 0x22) {
      out += '\\"';
      continue;
    }
    if (c === 0x5c) {
      out += "\\\\";
      continue;
    }
    if (c === 0x08) {
      out += "\\b";
      continue;
    }
    if (c === 0x0c) {
      out += "\\f";
      continue;
    }
    if (c === 0x0a) {
      out += "\\n";
      continue;
    }
    if (c === 0x0d) {
      out += "\\r";
      continue;
    }
    if (c === 0x09) {
      out += "\\t";
      continue;
    }
    if (
      c < 0x20 ||
      c === 0x3c ||
      c === 0x3e ||
      c === 0x26 ||
      c === 0x2028 ||
      c === 0x2029
    ) {
      out +=
        "\\u" +
        HEX[(c >> 12) & 0xf] +
        HEX[(c >> 8) & 0xf] +
        HEX[(c >> 4) & 0xf] +
        HEX[c & 0xf];
      continue;
    }
    out += s[i];
  }
  out += '"';
  return out;
}

export function encodeIndentedJSON(value: JSONInput, indent = "  "): string {
  return encodeNode(value, indent, "");
}

function encodeNode(v: JSONInput, indent: string, prefix: string): string {
  if (v === null) return "null";
  if (typeof v === "boolean") return v ? "true" : "false";
  if (typeof v === "string") return escapeJSONString(v);
  if (typeof v === "bigint") return v.toString();
  if (v instanceof JSONFloat32) return formatJSONNumber(v.value, 32);
  if (typeof v === "number") return formatJSONNumber(v, 64);
  if (Array.isArray(v)) {
    if (v.length === 0) return "[]";
    const inner = prefix + indent;
    const items = v.map((x) => inner + encodeNode(x, indent, inner));
    return `[\n${items.join(",\n")}\n${prefix}]`;
  }
  // Plain object — sort keys lexicographically to match Go's
  // json.MarshalIndent of map[string]any. Explicit `< / >` comparator
  // locks JS UTF-16 code-unit order (same as the default `.sort()`).
  // Known parity gap: Go uses UTF-8 byte order, which diverges from
  // UTF-16 for supplementary (non-BMP) code points. Realistic column
  // names are ASCII so this doesn't fire in practice; documented in the
  // round-9 review findings as a defer-listed limitation.
  const keys = Object.keys(v).sort((a, b) => {
    if (a < b) return -1;
    if (a > b) return 1;
    return 0;
  });
  if (keys.length === 0) return "{}";
  const inner = prefix + indent;
  const items = keys.map(
    (k) => `${inner}${escapeJSONString(k)}: ${encodeNode(v[k], indent, inner)}`
  );
  return `{\n${items.join(",\n")}\n${prefix}}`;
}

export function serializeJSON(result: QueryResult): Uint8Array {
  const records = result.rows.map((r) => {
    // Use a null-prototype object so a column literally named `__proto__`
    // (legal SQL: `SELECT 1 AS "__proto__"`) becomes an own enumerable
    // property rather than triggering Object.prototype's setter. With a
    // plain `{}`, the column is silently dropped from Object.keys() and
    // the downloaded JSON loses data.
    const obj = Object.create(null) as { [k: string]: JSONInput };
    for (let i = 0; i < result.columnNames.length; i++) {
      obj[result.columnNames[i]] = jsonValueFromRowValue(r.values[i]);
    }
    return obj;
  });
  return TEXT_ENCODER.encode(encodeIndentedJSON(records));
}

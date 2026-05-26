import type { Value as StructpbValue } from "@bufbuild/protobuf/wkt";
import { Engine } from "@/types/proto-es/v1/common_pb";
import type {
  QueryResult,
  RowValue,
  RowValue_Timestamp,
  RowValue_TimestampTZ,
} from "@/types/proto-es/v1/sql_service_pb";

const pad = (n: number, width: number): string =>
  n.toString().padStart(width, "0");

/** Format a Timestamp like Go's "2006-01-02 15:04:05.000000". UTC. Truncates sub-microsecond nanos. */
export function formatTimestamp(ts: RowValue_Timestamp): string {
  const seconds = Number(ts.googleTimestamp?.seconds ?? 0n);
  const nanos = ts.googleTimestamp?.nanos ?? 0;
  const date = new Date(seconds * 1000); // UTC ms
  // Pad years to 4 digits — Go's time.Format("2006-...") emits "0001-01-01"
  // for year 1, while JS's getUTCFullYear() returns 1. Without padding,
  // historical-date exports come out as "1-01-01" and break parity.
  const yyyy = pad(date.getUTCFullYear(), 4);
  const mm = pad(date.getUTCMonth() + 1, 2);
  const dd = pad(date.getUTCDate(), 2);
  const hh = pad(date.getUTCHours(), 2);
  const mi = pad(date.getUTCMinutes(), 2);
  const ss = pad(date.getUTCSeconds(), 2);
  const micros = pad(Math.floor(nanos / 1000), 6);
  return `${yyyy}-${mm}-${dd} ${hh}:${mi}:${ss}.${micros}`;
}

/** Format a TimestampTZ like Go's RFC3339Nano in the proto's FixedZone. */
export function formatTimestampTZ(tz: RowValue_TimestampTZ): string {
  const seconds = Number(tz.googleTimestamp?.seconds ?? 0n);
  const nanos = tz.googleTimestamp?.nanos ?? 0;
  const offset = tz.offset ?? 0; // seconds east of UTC

  // Apply the fixed offset to get wall-clock components in that zone.
  const shifted = new Date((seconds + offset) * 1000);
  // See formatTimestamp — same year-padding rationale (Go's RFC3339Nano
  // produces "0001-01-01T00:00:00Z" for year 1).
  const yyyy = pad(shifted.getUTCFullYear(), 4);
  const mm = pad(shifted.getUTCMonth() + 1, 2);
  const dd = pad(shifted.getUTCDate(), 2);
  const hh = pad(shifted.getUTCHours(), 2);
  const mi = pad(shifted.getUTCMinutes(), 2);
  const ss = pad(shifted.getUTCSeconds(), 2);

  let frac = "";
  if (nanos > 0) {
    const padded = pad(nanos, 9).replace(/0+$/, "");
    frac = padded.length > 0 ? `.${padded}` : "";
  }

  let zoneStr: string;
  if (offset === 0) {
    zoneStr = "Z";
  } else {
    const sign = offset > 0 ? "+" : "-";
    const abs = Math.abs(offset);
    const oh = pad(Math.floor(abs / 3600), 2);
    const om = pad(Math.floor((abs % 3600) / 60), 2);
    zoneStr = `${sign}${oh}:${om}`;
  }

  return `${yyyy}-${mm}-${dd}T${hh}:${mi}:${ss}${frac}${zoneStr}`;
}

/**
 * Convert a number to Go's strconv.FormatFloat(v, 'f', -1, 64) form:
 * shortest round-trip decimal, NEVER exponential, no trailing zeros except
 * "0" / "-0".
 */
export function formatFloat64(v: number): string {
  if (Number.isNaN(v)) return "NaN";
  if (v === Number.POSITIVE_INFINITY) return "+Inf";
  if (v === Number.NEGATIVE_INFINITY) return "-Inf";
  if (Object.is(v, -0)) return "-0";
  if (v === 0) return "0";

  const s = v.toString();
  if (!/[eE]/.test(s)) return s; // already non-exponential

  // Expand "1.234e21" / "1.234e-7" into fixed decimal manually.
  return expandExponential(s);
}

/**
 * Mirrors Go's strconv.FormatFloat(v, 'f', -1, 32): shortest round-trip
 * decimal for float32 precision, never exponential.
 * Uses toPrecision to find the shortest significand that round-trips as
 * float32, then expands exponential notation if needed.
 */
export function formatFloat32(v: number): string {
  if (Number.isNaN(v)) return "NaN";
  if (v === Number.POSITIVE_INFINITY) return "+Inf";
  if (v === Number.NEGATIVE_INFINITY) return "-Inf";
  const f32 = Math.fround(v);
  if (Object.is(f32, -0)) return "-0";
  if (f32 === 0) return "0";
  // Find shortest significant-digit count that round-trips through float32.
  for (let p = 1; p <= 17; p++) {
    const s = f32.toPrecision(p);
    if (Math.fround(Number.parseFloat(s)) === f32) {
      if (/[eE]/.test(s)) return expandExponential(s);
      return s.includes(".") ? s.replace(/\.?0+$/, "") : s;
    }
  }
  return expandExponential(f32.toString());
}

// Hot-path TextEncoder/TextDecoder — `goQuoteInner` runs per SQL string cell,
// so allocating a fresh encoder per call shows up in profiles on wide string
// columns. Both are safe to share (no per-call state).
const TEXT_ENCODER = new TextEncoder();
const FATAL_UTF8_DECODER = new TextDecoder("utf-8", { fatal: true });

const HEX = "0123456789abcdef";
const hex2 = (n: number): string => HEX[(n >> 4) & 0xf] + HEX[n & 0xf];
const hex4 = (n: number): string =>
  HEX[(n >> 12) & 0xf] +
  HEX[(n >> 8) & 0xf] +
  HEX[(n >> 4) & 0xf] +
  HEX[n & 0xf];
const hex8 = (n: number): string =>
  hex4((n >>> 16) & 0xffff) + hex4(n & 0xffff);

// Rough port of Go's unicode.IsPrint: letters, marks, numbers, punct, symbol,
// and ASCII space (0x20) only. We use Intl-free heuristics good enough for
// goldens-driven validation; final correctness anchored to byte-equal goldens.
//
// B9: explicitly reject surrogates (Cs) and Private-Use Area (Co) ranges.
// `unicode.IsPrint` rejects all Cc/Cf/Co/Cs/Cn categories. A full lookup
// table would be excessive — we cover the surrogate range and the three
// PUA blocks (BMP + supplementary). New runes that fall outside these
// known-non-print ranges still go through the heuristic; goldens are the
// source of truth for any specific rune we ship into customer output.
function isPrintRune(cp: number): boolean {
  if (cp === 0x20) return true;
  if (cp < 0x20 || cp === 0x7f) return false;
  // Format / control / unassigned ranges deemed non-print:
  if (cp >= 0x80 && cp <= 0xa0) return false;
  if (cp === 0xad) return false; // soft hyphen
  if (cp === 0x200b || cp === 0x200c || cp === 0x200d) return false;
  if (cp === 0x2028 || cp === 0x2029) return false;
  // Go's unicode.IsPrint returns false for the Zs (Space_Separator)
  // general category except ASCII space; matching that ensures SQL
  // exports containing en quads / em spaces / ideographic space round-
  // trip through `\uXXXX` escapes the same as the Go backend would.
  if (cp === 0x1680) return false; // Ogham space mark
  if (cp >= 0x2000 && cp <= 0x200a) return false; // en quad..hair space
  if (cp === 0x202f) return false; // narrow no-break space
  if (cp === 0x205f) return false; // medium mathematical space
  if (cp === 0x3000) return false; // ideographic space
  if (cp === 0xfeff) return false;
  if (cp >= 0xfff0 && cp <= 0xfffc) return false; // specials except U+FFFD (replacement char is printable)
  if (cp === 0xfffe || cp === 0xffff) return false; // BOM and non-char
  // Surrogates (Cs). TextEncoder normalizes lone surrogates to U+FFFD before
  // we ever see one here, but defense in depth is cheap.
  if (cp >= 0xd800 && cp <= 0xdfff) return false;
  // Private-Use Areas (Co):
  if (cp >= 0xe000 && cp <= 0xf8ff) return false; // BMP PUA
  if (cp >= 0xf0000 && cp <= 0xffffd) return false; // Supplementary PUA-A
  if (cp >= 0x100000 && cp <= 0x10fffd) return false; // Supplementary PUA-B
  if (cp >= 0xe0000 && cp <= 0xe007f) return false;
  return true;
}

/**
 * Reproduce the inner contents of Go's strconv.Quote(s):
 * the double-quote-stripped middle of `"…"`. Caller wraps in single quotes
 * and replaces ' with '' for SQL literals.
 *
 * Substrate: encode JS string to UTF-8, decode rune-by-rune; isolated bytes
 * that fail UTF-8 decode become \xHH (matches Go's per-byte behavior).
 */
export function goQuoteInner(s: string): string {
  const bytes = TEXT_ENCODER.encode(s);
  const decoder = FATAL_UTF8_DECODER;
  let out = "";
  let i = 0;
  while (i < bytes.length) {
    // Determine UTF-8 sequence length from leading byte.
    const b = bytes[i];
    let size: number;
    if (b < 0x80) size = 1;
    else if ((b & 0xe0) === 0xc0) size = 2;
    else if ((b & 0xf0) === 0xe0) size = 3;
    else if ((b & 0xf8) === 0xf0) size = 4;
    else size = 1; // invalid leading byte → single
    let rune: number | null = null;
    if (size > 1 && i + size <= bytes.length) {
      try {
        const seg = bytes.subarray(i, i + size);
        const text = decoder.decode(seg);
        if (text.length > 0) {
          rune = text.codePointAt(0)!;
        }
      } catch {
        rune = null;
      }
    } else if (size === 1) {
      rune = b;
    }
    if (rune === null) {
      out += "\\x" + hex2(b);
      i += 1;
      continue;
    }
    out += escapeRune(rune);
    i += size;
  }
  return out;
}

function escapeRune(cp: number): string {
  switch (cp) {
    case 0x07:
      return "\\a";
    case 0x08:
      return "\\b";
    case 0x09:
      return "\\t";
    case 0x0a:
      return "\\n";
    case 0x0b:
      return "\\v";
    case 0x0c:
      return "\\f";
    case 0x0d:
      return "\\r";
    case 0x22:
      return '\\"';
    case 0x5c:
      return "\\\\";
  }
  if (cp < 0x20 || cp === 0x7f) {
    return "\\x" + hex2(cp);
  }
  if (isPrintRune(cp)) {
    return String.fromCodePoint(cp);
  }
  if (cp <= 0xffff) {
    return "\\u" + hex4(cp);
  }
  return "\\U" + hex8(cp);
}

function expandExponential(s: string): string {
  const m = /^(-?)(\d+)(?:\.(\d+))?[eE]([+-]?\d+)$/.exec(s);
  if (!m) {
    // Defensive — should never happen for finite numbers.
    return s;
  }
  const [, sign, intPart, fracPart = "", expPart] = m;
  const exp = Number.parseInt(expPart, 10);
  const digits = intPart + fracPart;
  const decimalPos = intPart.length + exp;

  if (decimalPos <= 0) {
    return (
      `${sign}0.${"0".repeat(-decimalPos)}${digits}`.replace(/0+$/, "") ||
      `${sign}0`
    );
  }
  if (decimalPos >= digits.length) {
    return `${sign}${digits}${"0".repeat(decimalPos - digits.length)}`;
  }
  const left = digits.slice(0, decimalPos);
  const right = digits.slice(decimalPos).replace(/0+$/, "");
  return right.length === 0 ? `${sign}${left}` : `${sign}${left}.${right}`;
}

/**
 * Mirror lib/pq.QuoteLiteral — note the leading space before E' when
 * backslashes are present. This matches Go's pq.QuoteLiteral exactly:
 * https://github.com/lib/pq/blob/v1.11.1/quote.go#L75
 *
 * B12: NUL bytes (\\x00) are preserved literally inside the single quotes
 * for byte-for-byte parity with Go lib/pq. Postgres itself rejects NUL on
 * INSERT, so a customer who imports the resulting SQL will get an error
 * downstream — this is consistent with the backend behavior and the spec
 * accepts it as the contract. We do NOT silently strip or escape NUL here.
 */
export function pqQuoteLiteral(s: string): string {
  if (s.includes("\\")) {
    const escaped = s.replaceAll("\\", "\\\\").replaceAll("'", "''");
    return ` E'${escaped}'`;
  }
  return `'${s.replaceAll("'", "''")}'`;
}

/**
 * Compact JSON encoding of a structpb.Value — the cell-content string used
 * for VARIANT-style columns across every export format under the Tier 2
 * relaxation. Each side emits canonical JSON via JSON.stringify on the
 * unwrapped tree; the cross-side byte-equal contract no longer covers
 * structpb cells (each side picks its language-natural JSON form, which
 * happens to agree for ASCII content and may diverge on HTML-unsafe chars
 * or struct field order).
 * Inf / NaN → null to mirror JSON.stringify and JSON-spec restrictions.
 */
export function structpbValueAsJSON(v: StructpbValue | undefined): string {
  return JSON.stringify(unwrapStructpbValue(v));
}

function unwrapStructpbValue(v: StructpbValue | undefined): unknown {
  if (!v?.kind) return null;
  switch (v.kind.case) {
    case "nullValue":
      return null;
    case "boolValue":
      return v.kind.value;
    case "numberValue": {
      const n = v.kind.value;
      return Number.isFinite(n) ? n : null;
    }
    case "stringValue":
      return v.kind.value;
    case "listValue":
      return v.kind.value.values.map(unwrapStructpbValue);
    case "structValue": {
      // Object.create(null) so a column literally named "__proto__" inside
      // a structpb cell can't pollute the output via Object.prototype.
      const out: Record<string, unknown> = Object.create(null);
      for (const [k, val] of Object.entries(v.kind.value.fields ?? {})) {
        out[k] = unwrapStructpbValue(val);
      }
      return out;
    }
    default:
      return null;
  }
}

/** CSV cell rendering of a structpb value: JSON content surrounded by CSV
 *  quotes with embedded quotes doubled. Kept exported for compatibility
 *  with the `value.test.ts` import path; previously emitted the bracket
 *  "a:1,b:2" form (Tier 1) — now emits JSON-as-string (Tier 2). */
export function csvCellFromStructpbValue(v: StructpbValue | undefined): string {
  return csvQuoteString(structpbValueAsJSON(v));
}

/** XLSX cell rendering of a structpb value: raw JSON string (no CSV quoting,
 *  no surrounding quotes). Also used by the JSON encoder as the scalar
 *  string value for VARIANT columns. */
export function xlsxStringFromStructpbValue(
  v: StructpbValue | undefined
): string {
  return structpbValueAsJSON(v);
}

const ENGINES_PG_LIKE: ReadonlySet<Engine> = new Set([
  Engine.POSTGRES,
  Engine.REDSHIFT,
]);

const bytesToHex = (b: Uint8Array): string => {
  let s = "";
  for (const byte of b) s += hex2(byte);
  return s;
};

const bytesToBase64 = (b: Uint8Array): string => {
  let s = "";
  for (const byte of b) s += String.fromCharCode(byte);
  return btoa(s);
};

const csvQuoteString = (s: string): string => `"${s.replaceAll('"', '""')}"`;

/** Engine-aware single-quoted SQL literal. PG/Redshift use pq-style; others
 *  use Go's strconv.Quote inner with single-quote doubling. Hoist the
 *  PG-vs-non-PG branch once per call to avoid repeating the ternary in
 *  every type case below. */
export function quoteSqlString(s: string, isPg: boolean): string {
  return isPg
    ? pqQuoteLiteral(s)
    : `'${goQuoteInner(s).replaceAll("'", "''")}'`;
}

export function csvCellFromRowValue(v: RowValue | undefined): string {
  if (!v?.kind) return "";
  switch (v.kind.case) {
    case "nullValue":
      return "";
    case "stringValue":
      return csvQuoteString(v.kind.value);
    case "boolValue":
      return v.kind.value ? "true" : "false";
    case "int32Value":
      return v.kind.value.toString();
    case "int64Value":
      return v.kind.value.toString();
    case "uint32Value":
      return v.kind.value.toString();
    case "uint64Value":
      return v.kind.value.toString();
    case "floatValue":
      return formatFloat32(v.kind.value);
    case "doubleValue": {
      // Tier 2 relaxation: language-natural number emission for CSV/JSON/XLSX
      // float64 when finite. SQL float64 still uses formatFloat64 (engine-
      // literal compat). NaN / ±Inf cannot be expressed as a JS-native number
      // token (`String(NaN)` is "NaN" but `String(Infinity)` is "Infinity",
      // diverging from Go's "+Inf"/"-Inf"), AND emitting "" would lose the
      // distinction between a non-finite value and a NULL cell — so fall
      // through to formatFloat64 for non-finite, which produces the same
      // "NaN" / "+Inf" / "-Inf" tokens as the backend.
      const n = v.kind.value;
      return Number.isFinite(n) ? String(n) : formatFloat64(n);
    }
    case "bytesValue":
      return csvQuoteString("0x" + bytesToHex(v.kind.value));
    case "timestampValue":
      return csvQuoteString(formatTimestamp(v.kind.value));
    case "timestampTzValue":
      return csvQuoteString(formatTimestampTZ(v.kind.value));
    case "valueValue":
      return csvCellFromStructpbValue(v.kind.value);
    default:
      return "";
  }
}

/** What the JSON encoder accepts. `bigint` enables raw int64/uint64 tokens
 *  without going through JSON.stringify (which throws on bigint).
 *
 *  `JSONFloat32` is a class instance — NOT a plain object with a magic key —
 *  so a column literally named "__jsonFloat32" can never collide with the
 *  discriminator. The encoder uses `instanceof JSONFloat32` for routing. */
export class JSONFloat32 {
  constructor(public readonly value: number) {}
}
export const float32 = (n: number): JSONFloat32 =>
  new JSONFloat32(Math.fround(n));

export type JSONInput =
  | null
  | boolean
  | number
  | bigint
  | string
  | JSONFloat32
  | JSONInput[]
  | { [key: string]: JSONInput };

/** Map a single proto RowValue to the encoder's input tree directly — no
 *  intermediate atom shape, no round-trip. Throws SerializationFailed on
 *  NaN/Inf to mirror Go's encoding/json error. */
export function jsonValueFromRowValue(v: RowValue | undefined): JSONInput {
  if (!v?.kind) return null;
  switch (v.kind.case) {
    case "nullValue":
      return null;
    case "stringValue":
      return v.kind.value;
    case "boolValue":
      return v.kind.value;
    case "int32Value":
      return v.kind.value;
    case "uint32Value":
      return v.kind.value;
    case "int64Value":
      return v.kind.value; // bigint
    case "uint64Value":
      return v.kind.value; // bigint
    case "floatValue":
      return float32(v.kind.value);
    case "doubleValue":
      return v.kind.value;
    case "bytesValue":
      return bytesToBase64(v.kind.value);
    case "timestampValue":
      return formatTimestamp(v.kind.value);
    case "timestampTzValue":
      return formatTimestampTZ(v.kind.value);
    case "valueValue":
      return xlsxStringFromStructpbValue(v.kind.value);
    default:
      return null;
  }
}

/** Engine is reduced to a single boolean upstream (in serializeSQL) for the hot loop. */
export function sqlValueFromRowValue(
  v: RowValue | undefined,
  isPg: boolean
): string {
  if (!v?.kind) return "";
  switch (v.kind.case) {
    case "nullValue":
      return "NULL";
    case "stringValue":
      return quoteSqlString(v.kind.value, isPg);
    case "boolValue":
      return v.kind.value ? "true" : "false";
    case "int32Value":
      return v.kind.value.toString();
    case "int64Value":
      return v.kind.value.toString();
    case "uint32Value":
      return v.kind.value.toString();
    case "uint64Value":
      return v.kind.value.toString();
    case "floatValue":
      return formatFloat32(v.kind.value);
    case "doubleValue":
      return formatFloat64(v.kind.value);
    case "bytesValue":
      return "0x" + bytesToHex(v.kind.value);
    case "timestampValue":
      return quoteSqlString(formatTimestamp(v.kind.value), isPg);
    case "timestampTzValue":
      return quoteSqlString(formatTimestampTZ(v.kind.value), isPg);
    case "valueValue":
      // Structpb cells use proper engine-aware SQL string quoting (single-
      // quote literals with embedded `'` doubled, or `pq.QuoteLiteral` for
      // PG/Redshift). Backend currently still emits CSV-style double-quoted
      // strings here (a long-standing quirk noted in HANDOFF.md's "Known
      // SQL identifier / CSV header parity gap"); fixing the backend would
      // also require regenerating its tests and is intentionally deferred.
      // Tier 2 already drops cross-side byte equality for structpb cells,
      // so the frontend gets to be correct on its own.
      return quoteSqlString(structpbValueAsJSON(v.kind.value), isPg);
    default:
      return "";
  }
}

export const isPgLike = (engine: Engine): boolean =>
  ENGINES_PG_LIKE.has(engine);

/**
 * Estimate serialized byte size of `result`'s cells, walking RowValues once
 * without allocating any serialized output. Returns the running total
 * accumulated up to the moment it crosses `cap` (or the full sum if under).
 * Caller compares to the cap to decide whether to bail. Includes column-name
 * overhead so wide-schema-with-no-rows results aren't free.
 *
 * Per-cell upper bound bytes, conservative across CSV / SQL / JSON / XLSX
 * serialization. The estimator must NEVER under-count: the cap exists to
 * prevent browser OOM, so over-estimating is acceptable but under-estimating
 * defeats the guard.
 *
 *   string   → length * 6 + 4
 *              JSON `\u00XX` escape is the worst case: a string of pure
 *              control bytes / `<`,`>`,`&` (Go encoding/json escapes
 *              these by default) inflates to 6 ASCII bytes per source
 *              code unit. UTF-8 expansion alone bounds at 3x; the JSON
 *              escape dominates. CSV/SQL/XLSX worst cases stay under 6x.
 *   bytes    → byteLength * 2 + 4
 *              CSV/SQL hex-encode (each byte → two hex chars). JSON and
 *              XLSX base64-encode (4/3x). 2x bounds both.
 *   float32  → 48
 *              Max float32 (3.4e+38) in non-exponential form ≈ 40 chars.
 *   float64  → 336
 *              MaxFloat64 in non-exponential form ≈ 326 chars (309 integer
 *              digits + dot + 17 mantissa digits). The previous flat 16
 *              under-counted by 20x on dense float64 columns.
 *   int/uint → 24 (covers -9223372036854775808 worst case)
 *   bool     → 8
 *   timestamp→ 64 (RFC3339Nano with sub-second precision + zone)
 *   value_value → recursive walk over the structpb tree. The previous
 *              flat 1024 const let a single huge nested string/list/struct
 *              bypass the cap.
 *   null     → 4
 */
const APPROX_NULL_BYTES = 4;
const APPROX_BOOL_BYTES = 8;
const APPROX_INT_BYTES = 24;
const APPROX_FLOAT32_BYTES = 48;
const APPROX_FLOAT64_BYTES = 336;
const APPROX_TIMESTAMP_BYTES = 64;
const APPROX_BYTES_OVERHEAD = 4;
const APPROX_STRUCT_OVERHEAD = 16;

export function estimateResultBytes(result: QueryResult, cap: number): number {
  // JSON re-emits every column name per row (`{"col": v, ...}`), and SQL
  // re-emits the INSERT prefix's column list per row. Charging column-name
  // overhead only for the header would let a 5M-row × 1-col × 64-byte
  // result bypass the cap when serialized as JSON/SQL. Pre-compute the
  // per-row column-name cost once and apply it to header + each data row.
  // Column names use the SAME 6x string multiplier as cell strings (see
  // approxCellBytes) — a JSON object key like `"<col>"` can inflate
  // 6x via `<` escaping the same way value strings can.
  let columnOverhead = 0;
  for (const c of result.columnNames) columnOverhead += c.length * 6 + 4;
  let total = columnOverhead;
  for (const r of result.rows) {
    total += columnOverhead;
    for (const v of r.values) {
      total += approxCellBytes(v);
      if (total > cap) return total;
    }
  }
  return total;
}

function approxCellBytes(v: RowValue | undefined): number {
  if (!v?.kind) return APPROX_NULL_BYTES;
  switch (v.kind.case) {
    case "nullValue":
      return APPROX_NULL_BYTES;
    case "boolValue":
      return APPROX_BOOL_BYTES;
    case "int32Value":
    case "int64Value":
    case "uint32Value":
    case "uint64Value":
      return APPROX_INT_BYTES;
    case "floatValue":
      return APPROX_FLOAT32_BYTES;
    case "doubleValue":
      return APPROX_FLOAT64_BYTES;
    case "stringValue":
      return v.kind.value.length * 6 + 4;
    case "bytesValue":
      return v.kind.value.byteLength * 2 + APPROX_BYTES_OVERHEAD;
    case "timestampValue":
    case "timestampTzValue":
      return APPROX_TIMESTAMP_BYTES;
    case "valueValue":
      return approxStructpbBytes(v.kind.value);
    default:
      return APPROX_INT_BYTES;
  }
}

function approxStructpbBytes(v: StructpbValue | undefined): number {
  if (!v?.kind) return APPROX_NULL_BYTES;
  switch (v.kind.case) {
    case "nullValue":
      return APPROX_NULL_BYTES;
    case "boolValue":
      return APPROX_BOOL_BYTES;
    case "numberValue":
      return APPROX_FLOAT64_BYTES;
    case "stringValue":
      return v.kind.value.length * 6 + 4;
    case "listValue": {
      let total = APPROX_STRUCT_OVERHEAD;
      for (const item of v.kind.value.values ?? []) {
        total += approxStructpbBytes(item);
      }
      return total;
    }
    case "structValue": {
      let total = APPROX_STRUCT_OVERHEAD;
      for (const [k, val] of Object.entries(v.kind.value.fields ?? {})) {
        total += k.length * 6 + 4;
        total += approxStructpbBytes(val);
      }
      return total;
    }
    default:
      return APPROX_INT_BYTES;
  }
}

export function xlsxValueFromRowValue(v: RowValue | undefined): string {
  if (!v?.kind) return "";
  switch (v.kind.case) {
    case "nullValue":
      return "";
    case "stringValue":
      return v.kind.value;
    case "boolValue":
      return v.kind.value ? "true" : "false";
    case "int32Value":
      return v.kind.value.toString();
    case "int64Value":
      return v.kind.value.toString();
    case "uint32Value":
      return v.kind.value.toString();
    case "uint64Value":
      return v.kind.value.toString();
    case "floatValue":
      return formatFloat32(v.kind.value);
    case "doubleValue": {
      // See csvCellFromRowValue's doubleValue branch for the rationale.
      // Native String() for finite; formatFloat64 for NaN / ±Inf so we emit
      // textual tokens matching the backend ("NaN" / "+Inf" / "-Inf") instead
      // of "" (which would be indistinguishable from a NULL cell).
      const n = v.kind.value;
      return Number.isFinite(n) ? String(n) : formatFloat64(n);
    }
    case "bytesValue":
      return bytesToBase64(v.kind.value);
    case "timestampValue":
      return formatTimestamp(v.kind.value);
    case "timestampTzValue":
      return formatTimestampTZ(v.kind.value);
    case "valueValue":
      return xlsxStringFromStructpbValue(v.kind.value);
    default:
      return "";
  }
}

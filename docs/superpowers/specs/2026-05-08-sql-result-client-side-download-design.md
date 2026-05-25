# SQL Result Client-Side Download — Design

**Date:** 2026-05-08

---

## Implementation deltas (read this first — what shipped differs from the body below)

This design doc was written before implementation; review rounds and product feedback drove four substantive changes from the body below. Treat the deltas list here as authoritative when there's a conflict.

1. **Gated behind `isDev()`** — the client-side path is dev-build only. Production builds still call the backend `Export` RPC via the restored legacy components (`DataExportButtonLegacy.vue` / `DataExportButtonLegacy.tsx`). Driven by audit-log / access-check / abort-on-error gaps the client-side path can't yet match. Toggle: `import.meta.env.DEV`.
2. **Unified groups-shape public API** — replaces the originally-planned three separate helpers (`buildDownloadBlob` / `buildBatchDownloadBlob` / `buildMultiResultDownloadBlob`). Single signature `buildDownloadBlob({ groups, format, baseFilename, password? })` where each group is `{ instanceId, databaseName, engine, statements: [{ result, statement }] }`. Covers single-statement, multi-statement, and batch in one shape.
3. **Always-ZIP output with backend-parity directory layout** — `<baseFilename>.zip` containing `<instanceId>/<databaseName>/statement-N.{sql,result.<ext>}`. The `.sql` sidecar (per-statement SQL text) sits next to its result file. Mirrors `exportResultToZip` in `backend/api/v1/sql_service.go`. Earlier intermediate versions emitted bare-file (no ZIP) for single-result downloads — that's gone.
4. **Encryption is AES-256, not ZipCrypto** — the body originally said ZipCrypto. Backend uses `alexmullins/zip` which writes WinZip AES-256 (extra-field 0x9901, vendor "AE", strength=3). `zip.js` with default options matches. `TestZipFrontendParity` verifies bidirectional read.

## Top-priority follow-ups (recorded in `…-followups.md`)

- **Server-side audit log** — restore via non-blocking `RecordDownloadEvent` RPC. Compliance gap.
- **(resolved)** Multi-statement download — `buildDownloadBlob` groups shape handles this.

---

**Scope:** Move SQL editor result-panel "download as file" from the backend `Export` RPC to a framework-neutral, in-browser helper that serializes already-fetched rows to CSV/JSON/SQL/XLSX and optionally wraps them in a password-protected ZIP. Both the Vue (`frontend/src/components/DataExportButton.vue`) and React (`frontend/src/react/components/DataExportButton.tsx`) wrappers are migrated in this PR — they consume the same helper. The framework-neutral helper exists precisely because both wrappers exist; migrating only one would leave the React result panel still calling the backend RPC and still importing `JSZip`, defeating the cleanup.

**`BatchQuerySelect.tsx` is also in scope.** This React surface lets a user pick N databases and download N per-database results as a single multi-file ZIP. Today it iterates `selectedDatabaseNames`, calls `useSQLStore().exportData()` once per database, and bundles the returned `DownloadContent[]` via `JSZip` (the only "multi-file" path on the frontend). The plan migrates this by introducing a sibling `buildBatchDownloadBlob({ entries, password })` helper that takes `Array<{ result, baseFilename }>` and produces one multi-entry ZIP via `@zip.js/zip.js` (with WinZip AES-256 encryption when a password is set). All entries share the same password — matches today's behavior. After the migration, `JSZip` has zero remaining frontend callers and is removed.

## Background

Today, when a user clicks the result-panel download button in the SQL editor:

1. `frontend/src/components/DataExportButton.vue` collects format + optional password + row limit into an `ExportOption`.
2. The parent `frontend/src/views/sql-editor/EditorCommon/ResultView/ResultViewV1.vue` builds an `ExportRequest` and calls `useSQLStore().exportData()`.
3. `frontend/src/store/modules/v1/sql.ts::exportData` calls the backend `SQLService.Export` RPC.
4. The backend re-runs the SQL, applies masking, formats via `backend/component/export/{csv,json,sql,xlsx}.go`, and (if a password is set) wraps the bytes in a WinZip AES-256-encrypted ZIP via `github.com/alexmullins/zip`.
5. The client receives `Uint8Array` and saves it via `file-saver`.

The result panel already has the rows in memory as a `QueryResult` proto. Round-tripping to the backend to format them is wasted work. The data-export-redesign (`docs/design/data-export-redesign.md`) splits this into:

- **Download** — instant, client-side, operates on visible/in-memory rows. (This spec.)
- **Export** — server-run, approval/audit-gated, fetches fresh rows. (Not changed.)

The Vue→React migration is also live, and the React result-panel doesn't exist yet. A framework-neutral helper lets the same code serve Vue today and React tomorrow.

## Non-goals

- Not changing the backend `SQLService.Export` RPC. It stays available for any non-result-panel callers (admin Export Center, audit-log download, etc., when those land per the redesign).
- Not implementing the data-export-redesign UI split (`DownloadButton` vs `ExportButton`). That is a separate UX change.
- Not changing the ZIP encryption scheme. Both backend and frontend now produce WinZip AES-256 (extra-field 0x9901, vendor "AE", strength=3, deflate). Earlier drafts of this spec referred to this as ZipCrypto — that was incorrect.
- Not extending row limits or admin-mode (mask-bypass) on the client. The Download path operates strictly on what the panel already has.

## Decisions

These were locked during brainstorming:

| Decision | Choice | Why |
|---|---|---|
| Encryption | **WinZip AES-256** | Parity with backend (`alexmullins/zip` writes WinZip AES-256 by default: extra-field 0x9901, vendor "AE", strength=3, actual-method=deflate). Standard 7-Zip / OS unzippers handle it. |
| Scope | **Download path only** | Backend `Export` RPC stays for approval-gated/admin paths. Result-panel is the only frontend caller today. |
| SQL dialect | **Engine-aware (parity)** | Backend picks backticks vs double-quotes per engine and uses `pq.QuoteLiteral` for PG strings; frontend mirrors this exactly. |
| Limit field | **Drop it** | Client-side has only what's been fetched. `maximumResultRows` policy already caps this at the query stage. |
| Table name in SQL output | **Always `<table_name>`** | Matches backend's multi-resource fallback (`backend/component/export/sql.go:74`). No SQL parsing on the client. |

## Architecture

### Module layout

```
frontend/src/utils/sql-download/
├── index.ts               # public API: buildDownloadBlob()
├── types.ts               # DownloadInput, DownloadOutput, downloadError() factory
├── engines.ts             # SQL_DOWNLOADABLE_ENGINES, isSqlDownloadSupported(engine)
├── value.ts               # RowValue → cell helpers (shared by all formats)
├── formats/
│   ├── csv.ts
│   ├── json.ts
│   ├── sql.ts
│   └── xlsx.ts            # dynamic-imports exceljs
├── zip.ts                 # dynamic-imports @zip.js/zip.js, WinZip AES-256 wrap
└── __tests__/             # vitest fixtures + parity assertions vs goldens
```

**Why `engines.ts` is a separate file.** The Vue and React wrappers need to disable the SQL format option in the dropdown when the connection's engine is unsupported — answered by `isSqlDownloadSupported(engine)`. If that predicate lived in `value.ts` alongside the cell helpers, importing it would pull `value.ts` and its proto-WKT/timestamp/float dependencies into the eager component bundle, defeating the lazy-load story. `engines.ts` is small (an `Engine`-keyed `Set` plus a one-line predicate); only the predicate is exported to consumers, and the underlying set is reused by `formats/sql.ts` to look up the dialect quote character. Net eager bundle cost: a 12-entry Set.

**Engine coverage** (must mirror backend `SQLStatementPrefix` in `backend/component/export/sql.go:55-64` exactly):

- Backtick (`` ` ``): MYSQL, MARIADB, TIDB, OCEANBASE, SPANNER
- Double-quote (`"`): CLICKHOUSE, MSSQL, ORACLE, POSTGRES, REDSHIFT, SQLITE, SNOWFLAKE
- 14 engines (including all non-SQL stores like MongoDB / Redis / Cassandra / DynamoDB / Cosmos DB / Elasticsearch, plus SQL-flavored gaps like StarRocks / Doris / Hive / BigQuery / CockroachDB / Databricks / Trino) are not in the map; the SQL format option is disabled for those connections. **CSV / JSON / XLSX remain available for all engines** — the gating is SQL-format-specific.

Adding a new engine to the frontend map without adding it to `SQLStatementPrefix` on the backend would break goldens. Any expansion is a coordinated backend + frontend change, with new SQL goldens generated for the added engine.

### Public API

```typescript
import type { QueryResult } from "@/types/proto-es/v1/sql_service_pb";
import { ExportFormat, Engine } from "@/types/proto-es/v1/common_pb";

export interface DownloadInput {
  result: QueryResult;          // proto, has column_names/types/rows
  format: ExportFormat;         // CSV | JSON | SQL | XLSX
  engine: Engine;               // for SQL dialect quoting; ignored otherwise
  baseFilename: string;         // e.g. "mydb.2026-05-08T10-22-31"
  password?: string;            // empty/undefined → no encryption
}

export interface DownloadOutput {
  blob: Blob;
  filename: string;             // {baseFilename}.{ext} OR {baseFilename}.zip if password
  mimeType: string;
}

export async function buildDownloadBlob(input: DownloadInput): Promise<DownloadOutput>;
```

### Naming convention

The new client-side path is consistently called **download** (`utils/sql-download`, `buildDownloadBlob`, `downloadError()` factory with `code` field, locale keys `sql-editor.download-as-file`). The proto `ExportFormat` enum and the backend `SQLService.Export` RPC keep their names; we are not renaming wire protocol. This makes "download" (client, in-memory rows) unambiguously distinct from the backend's "Export" (server-run, approval-gated).

## Format serializers (parity with backend)

Each format module takes a `QueryResult` and produces `Uint8Array` bytes. The shared `value.ts` exposes per-format converters for a single `RowValue` plus the timestamp helpers:

```typescript
formatTimestamp(ts)    // "YYYY-MM-DD HH:mm:ss.SSSSSS" (6-digit microseconds, UTC)
formatTimestampTZ(ts)  // RFC3339Nano in {zone, offset}
```

These mirror `backend/component/export/export.go:30-38`.

### Per-type rules (mirrored from backend)

| Type | CSV | JSON | SQL | XLSX |
|---|---|---|---|---|
| `null` | empty (bare `,`) | `null` | `NULL` | empty cell |
| `string` | `"…"` (`"` → `""`) | Go `encoding/json` string escaping (see "JSON string escape" below) | engine-quoted (see "SQL string escape" below) | string |
| `int32/uint32` | bare digits | bare number | bare digits | string repr (text cell, parity) |
| `int64/uint64` | bare digits | bare-number JSON token (custom encoder, see "JSON int64" below) | bare digits | string repr |
| `float32` | shortest non-exponential decimal (see "Float formatting (CSV/SQL)" below) | Go `encoding/json`-style float (see "JSON float formatting" below) | bare (non-exponential `'f'` form) | string repr |
| `float64` | shortest non-exponential decimal | Go `encoding/json`-style float | bare | string repr |
| `bool` | `true`/`false` | boolean | `true`/`false` | `"true"`/`"false"` |
| `bytes` | `"0x{hex}"` (quoted, **lowercase hex**) | base64 (std) | `0x{hex}` (unquoted, lowercase) | base64 |
| `timestamp` | quoted formatted | string (formatted, not RFC3339) | engine-quoted formatted | formatted |
| `timestamptz` | quoted formatted | string (formatted) | engine-quoted formatted | formatted |
| `structpb.Value` (Spanner / ClickHouse TUPLE/ARRAY/MAP via `value_value`) | recursive helper, wrapped in `"…"` | proto `.String()` text repr (parity quirk) | reuses CSV helper (parity quirk) | proto `.String()` |

**Float32 JSON edge note.** Go encodes float32 via `strconv.AppendFloat(_, _, -1, 32)` — shortest decimal that round-trips through float32. The TS encoder applies `Math.fround(v)` first to canonicalize the float64 to its float32 image, then formats. For most practical values this matches Go byte-for-byte, but exotic float32 values may exhibit rounding-precision divergence. Treat float32 JSON as a goldens-iterated quirk: tune the formatter when a golden surfaces a mismatch.

**Encoder discriminator must be collision-proof.** The custom JSON encoder distinguishes float32 values via a wrapper. Using a magic property name (e.g. `__jsonFloat32`) is unsafe because a result column literally named that would create a row object whose `hasOwnProperty(__jsonFloat32)` is true and would be misclassified as a wrapper. Use a unique JS `Symbol` (or a dedicated class with `instanceof`) — both are unforgeable from data flowing through the encoder.

**JSON-specific output rules** (Go `encoding/json` divergences from CSV/SQL/XLSX):

- **Object keys are emitted in lexicographic order**, not column order. `json.MarshalIndent` of a `map[string]any` always sorts keys. The TS encoder must sort the per-row object keys before emitting them; matching column order would diverge from Go.
- **Float verb is `'g'` with thresholds**, not `'f'`: for `bits == 64`, abs values in `[1e-6, 1e21)` use fixed-decimal, otherwise exponential. Go also strips a leading zero from a two-digit negative exponent (`e-09` → `e-9`). For `bits == 32`, the threshold is computed against `float32(abs)`. The plan's `formatFloat64` (always `'f'`) is the wrong helper for JSON — use a dedicated `formatJSONNumber` formatter.
- **NaN, +Inf, -Inf are unrepresentable.** `json.MarshalIndent` returns `*json.UnsupportedValueError` for any of these. The TS encoder must throw `SerializationFailed` on the same inputs. The `floats_edges` fixture is therefore JSON-excluded (split into `floats_finite_edges` + `floats_special_edges` — only the former gets a JSON golden).
- **HTML escapes default-on:** `<` → `<`, `>` → `>`, `&` → `&`, plus ` ` / ` `. Standard control chars use `\b`, `\f`, `\n`, `\r`, `\t`, `\"`, `\\`. All other control bytes < 0x20 use `\u00HH`.

**CSV-specific output rule:** the header line is **always** followed by a trailing `\n`, even when there are zero rows. Backend (`csv.go:21-26`) writes `Join(columnNames, ",")` then `'\n'` unconditionally before the row loop. A naive `[header, ...rowLines].join("\n")` produces `"header"` for zero rows and diverges from Go by one byte. The serializer must construct as `header + "\n" + rowLines.join("\n")`.

**Quirks intentionally preserved.** JSON `value_value` and SQL `value_value` use different helpers in the backend, and XLSX writes everything as text cells. We mirror these for byte-equality with backend output. They are filed as separate follow-ups; fixing them belongs in a coordinated backend+frontend change.

### SQL string escape (port `strconv.Quote` semantics, not "Go-escape")

Backend (`backend/component/export/sql.go:120-134`):

```go
// non-PG path
result := []byte("'")
s := strconv.Quote(string(v))   // produces "…" with Go double-quoted-literal escapes
s = s[1 : len(s)-1]              // strip the surrounding double-quotes
s = strings.ReplaceAll(s, `'`, `''`)   // double single-quotes for SQL
result = append(result, []byte(s)...)
result = append(result, '\'')
```

The TS port must reproduce `strconv.Quote` byte-exactly. **Substrate:** Go iterates the input as a byte string, peeling off either a valid UTF-8 rune (via `utf8.DecodeRuneInString`) or a single invalid byte at a time (RuneError, size 1). The TS implementation receives a JS string (UTF-16). Encode to UTF-8 bytes first via `TextEncoder().encode(s)`, then iterate the resulting `Uint8Array` rune-by-rune (re-decoding via `TextDecoder({ fatal: false })` per code point, falling back to single-byte handling on decode failure). This matches Go's behavior for lone surrogates and invalid sequences.

Escape rules (output uses Go's hex/unicode notation, **no curly braces**):
- `\` → `\\`, `"` → `\"`, `\t` → `\t`, `\n` → `\n`, `\r` → `\r`, BEL/BS/FF/VT → corresponding `\a`/`\b`/`\f`/`\v`.
- Other ASCII control bytes < 0x20, plus DEL (0x7F) → `\xHH` (lowercase hex, exactly 2 digits).
- Single bytes that fail UTF-8 decode (e.g., a stray 0xFF) → `\xHH`.
- Valid UTF-8 runes that are **not printable** per Go's `unicode.IsPrint` → `\uHHHH` for code points ≤ 0xFFFF, `\UHHHHHHHH` for higher (lowercase hex, fixed width — 4 or 8 digits).
- **`isPrintRune` must be allow-list, not deny-list.** `unicode.IsPrint` returns true only for runes in categories `L` (letters), `M` (marks), `N` (numbers), `P` (punctuation), `S` (symbols), plus ASCII space `0x20`. Everything else — `Cc`, `Cf`, `Co`, `Cs`, `Cn`, line separators (`Zl`), paragraph separators (`Zp`), and non-ASCII space (`Zs`) — is non-printable. The TS heuristic must default `false` for any rune not explicitly classified into one of those categories. A permissive default-true version emits raw bytes where Go emits `\uHHHH`/`\UHHHHHHHH`, breaking SQL goldens for any string containing a private-use, format, or unassigned rune.
- Printable runes (per the allow-list above) → emitted as their UTF-8 bytes verbatim.
- Single quotes `'` are **not** escaped by `strconv.Quote` (they don't need to be inside a Go double-quoted literal). They survive into the post-strip string and then get doubled to `''` in the next step.
- The leading `\"` from any embedded `"` survives into the SQL output as a literal `\"`.

Reference table for the implementer; goldens are the source of truth.

Implement as a dedicated `goQuoteInner(s: string): string` helper in `formats/sql.ts`. Goldens cover `'`, `"`, `\`, `\n`, `\r\n`, `\t`, `\x00` (NUL), `\x1b` (ESC), and a non-BMP rune.

PG/Redshift path uses `pq.QuoteLiteral` semantics (`backend/component/export/sql.go:122-124`):
- If string contains `\` → emit `E'…'` form, escape `\` → `\\` and `'` → `''`.
- Otherwise → emit `'…'` with `'` → `''`.
- Postgres rejects NUL bytes in literals; this is a backend-shared known-bad case (see "Known parity quirks" below).

### Float formatting (CSV / SQL / XLSX) — Go `strconv.FormatFloat(v, 'f', -1, bitSize)`

CSV, SQL, and XLSX all share this format. Critical: `'f'` verb is **never** scientific notation — pure decimal. JS `Number.prototype.toString()` switches to exponential at large/small magnitudes (≥ 1e21, ≤ 1e-7). The bitSize controls shortest-round-trip precision (32 vs 64).

Implementation:
- `formatFloat64(v: number): string` — convert via `Number.toString()`, then if result contains `e`/`E` expand back to fixed decimal. ~30 lines.
- `formatFloat32(v: number): string` — apply `Math.fround(v)` first to canonicalize to float32 representation, then format. (Proto `FloatValue` arrives as JS `number` already round-tripped through float32 by proto-es decoder; `Math.fround` is belt-and-suspenders.)
- Goldens cover: `0`, `-0`, `1.5`, `1e21`, `1e-7`, `Number.MAX_VALUE`, denormals, NaN, ±Infinity.

Note: backend `'f' -1` for NaN emits `"NaN"`, for `+Inf` emits `"+Inf"`, for `-Inf` emits `"-Inf"`. JS toString emits `NaN`/`Infinity`/`-Infinity`. **Mirror Go's strings**, not JS's.

### JSON float formatting — Go `encoding/json` (verb `'g'` with thresholds)

`encoding/json` does **not** use `'f'`. Its `floatString` picks the verb based on magnitude (see Go stdlib `encoding/json/encode.go::floatEncoder`):

```
abs := math.Abs(f)
fmt := byte('f')
if abs != 0 {
  if (bits == 64 && (abs < 1e-6 || abs >= 1e21)) ||
     (bits == 32 && (float32(abs) < 1e-6 || float32(abs) >= 1e21)) {
    fmt = 'e'
  }
}
out := strconv.AppendFloat(buf, f, fmt, -1, bits)
// post-process: strip leading zero in two-digit negative exponent: "e-09" → "e-9"
```

The TS port (`formatJSONNumber(v: number, bits: 32 | 64): string`) must replicate:
- For abs values in `[1e-6, 1e21)` (using float32 boundary when bits=32): non-exponential decimal (effectively `formatFloat64`/`formatFloat32`).
- Otherwise: scientific notation with Go's specific format (`-?\d(?:\.\d+)?e[+-]\d+`), single leading digit before the `.`, signed exponent without leading zeros except for `e+0` / `e-0`.
- Two-digit negative exponent has its leading zero stripped: `1e-07` → `1e-7`. (Go stdlib does this for `e-0X` only; positive `e+09` is **not** stripped.)
- Special values cause `SerializationFailed`: `encoding/json` returns `*json.UnsupportedValueError` for NaN/+Inf/-Inf. TS encoder throws `downloadError("SerializationFailed", ...)` on the same inputs.

Goldens cover: `0`, `1.5`, `1e10` (still `'f'` form for bits=64 since `< 1e21`), `1e-7` (`< 1e-6` → `'e'` form), `1e21` (`>= 1e21` → `'e'` form), `Number.MAX_VALUE` (`'e'` form), denormals.

### JSON string escape — Go `encoding/json` (HTML-safe by default)

Default behavior of `json.MarshalIndent` (HTML escaping enabled):
- `"` → `\"`, `\` → `\\`.
- `\b`, `\f`, `\n`, `\r`, `\t` keep their named escapes.
- Other control bytes < 0x20 → `\u00HH` (lowercase hex).
- HTML-significant: `<` → `<`, `>` → `>`, `&` → `&`.
- Line/paragraph separators: U+2028 → ` `, U+2029 → ` `.
- All other Unicode is emitted verbatim as UTF-8 (Go does **not** escape non-printable runes outside the above list in the default encoder; `IsPrint` does not gate JSON output).

Test fixtures must include each special character. Confirm against Go output via the goldens generator before committing.

### JSON object key ordering

`json.MarshalIndent` of a `map[string]T` emits keys in **lexicographic order**, regardless of insertion order. The TS encoder must sort keys when emitting plain objects so frontend output matches Go output for any column-name ordering. The generator-side row representation in `serializeJSON` constructs `{ [columnName]: value }` in column order; the encoder sorts at emit time.

### JSON int64 (custom encoder, not `JSON.stringify`)

Go `encoding/json` emits int64 as a bare numeric token. JS `JSON.stringify(BigInt)` throws; coercing bigint to `Number` loses precision past 2^53. Solution: write a minimal indented-JSON encoder (~50 lines) that:
- Accepts a tree of `null` | `boolean` | `number` | `bigint` | `string` | array | plain object.
- Emits `bigint` as `value.toString()` directly (raw token).
- Mirrors `json.MarshalIndent(records, "", "  ")` whitespace exactly: 2-space indent, `,\n` separator inside containers, no trailing commas, key-value separated by `": "`.

Proto-es `int64Value` arrives as `bigint` by default — pass it through unchanged.

Empty result (zero rows) → backend Go emits `[]` (literal two bytes). Match.

### Timestamp format (truncation, not rounding)

`formatTimestamp(ts)` mirrors Go's `time.Format("2006-01-02 15:04:05.000000")`:
- Always 6 digits in the fractional part (trailing zeros preserved).
- **Truncates** sub-microsecond precision from the proto's `nanos` (Go's behavior, not rounding).
- Implementation: read `seconds` and `nanos` from `google.protobuf.Timestamp`, truncate `nanos` to `Math.floor(nanos / 1000)` for microseconds.

`formatTimestampTZ(ts)` mirrors Go `RFC3339Nano`:
- Strips **trailing zero digits** from the fractional second, including dropping the `.` entirely if all-zero. JS `Date.toISOString()` always emits 3 digits — wrong.
- Renders the offset as `Z` for zero-offset or `±HH:MM`. The proto provides `{zone, offset}` (offset in seconds east of UTC); render the offset directly without consulting browser timezone.
- Implementation: ~40 lines, no library dependency on dayjs's tz plugin needed.

### structpb.Value StructValue iteration (non-deterministic in backend)

Backend (`backend/component/export/csv.go:128-143`) iterates `value.GetStructValue().Fields` — a Go map with **non-deterministic iteration order**. Backend output for struct cells is therefore non-deterministic per process.

Resolution: the goldens generator wraps struct iteration with a deterministic key sort (lexicographic) before serializing. We commit sorted-key goldens; the TS implementation also sorts struct keys. This means:
- Frontend output is deterministic (good for tests).
- Backend output in production is non-deterministic; frontend output diverges from backend in struct-cell ordering.
- This divergence is ACCEPTED as a "fix forward" (the deterministic side is correct; backend should sort too).

Document this in "Known parity quirks accepted" below.

### Format module shapes

```typescript
// formats/csv.ts
export function serializeCSV(result: QueryResult): Uint8Array
//   header: columnNames.join(",") + "\n"
//   rows: cells.join(",") with "\n" between rows (NO trailing newline — backend csv.go:39-43)
//   zero rows: ends after the header newline (one trailing \n)
//   zero columns: header is empty string + "\n", rows are empty lines

// formats/json.ts
export function serializeJSON(result: QueryResult): Uint8Array
//   custom encoder (see "JSON int64" above) — NOT JSON.stringify
//   zero rows: literal "[]"
//   produces 2-space indent matching Go's json.MarshalIndent(_, "", "  ")

// formats/sql.ts
export function serializeSQL(result: QueryResult, engine: Engine): Uint8Array
//   prefix: INSERT INTO `<table_name>` (cols...) VALUES (
//     MySQL/MariaDB/TiDB/OceanBase/Spanner → backticks
//     ClickHouse/MSSQL/Oracle/Postgres/Redshift/SQLite/Snowflake → double-quotes
//     Other → throw with code: "UnsupportedFormat"
//   per row: prefix + values + ");"  with "\n" between rows (no trailing \n)
//   zero rows: empty output (matches backend sql.go:27-50)
//   empty column_names: emits "INSERT INTO `<table_name>` () VALUES (...);" — known bad
//                       case shared with backend; not fixed here

// formats/xlsx.ts
export async function serializeXLSX(result: QueryResult): Promise<Uint8Array>
//   dynamic-imports exceljs; one sheet "Sheet1"; row 1 = headers; data starts row 2
//   columns A..ZZZ via excelColumnName(index) (≥ 18278 → throw "UnsupportedFormat")
```

## ZIP wrapper (WinZip AES-256 via zip.js)

Library: `@zip.js/zip.js` (~120 KB gzipped, dynamic-imported only when `password` is set). `jszip` (existing dep) does not support encryption.

```typescript
// frontend/src/utils/sql-download/zip.ts
export async function wrapWithEncryptedZip(
  inner: { bytes: Uint8Array; filename: string },
  password: string,
  outerBaseFilename: string,
): Promise<DownloadOutput> {
  const { ZipWriter, BlobWriter, Uint8ArrayReader } = await import("@zip.js/zip.js");

  const zipWriter = new ZipWriter(new BlobWriter("application/zip"), {
    password,
    // No zipCrypto flag — zip.js defaults to WinZip AES-256, byte-equivalent
    // to backend (alexmullins/zip): extra-field 0x9901, vendor "AE",
    // strength=3, actual-method=deflate.
    keepOrder: true,
  });

  await zipWriter.add(inner.filename, new Uint8ArrayReader(inner.bytes));
  const blob = await zipWriter.close();

  return {
    blob,
    filename: `${outerBaseFilename}.zip`,
    mimeType: "application/zip",
  };
}
```

**Inner filename:** `{baseFilename}.{ext}` (e.g. `mydb.2026-05-08T10-22-31.csv`). Outer ZIP: `{baseFilename}.zip`. Matches backend output.

**Memory:** Single-entry ZIP, all in memory. For 100K-row CSV (~10–50 MB raw) the ZIP is ~3–10 MB. `maximumResultRows` policy already caps the row count at the query stage.

**Security note (in this doc, not in code comments):** Both backend and frontend produce WinZip AES-256-encrypted ZIPs. This is the modern, strong-encryption flavor; the legacy weak ZipCrypto stream cipher is NOT used by either side.

## Integration into the SQL editor

Two files change.

### `DataExportButton.vue`

**Remove:**
- The `limit` row in the form ("Export rows" input).
- The `limit` field on the option payload.
- `maximum-export-count` prop (already capped upstream at query time).
- The internal multi-file `JSZip` fallback path. Client-side download always produces exactly one `DownloadContent`, so the wrap-multiple-into-zip code is dead. Removing it lets `jszip` come out of the dep list (verified to have no other frontend callers).

**Keep:**
- Format selector (CSV/JSON/SQL/XLSX). When `database.instanceResource?.engine` is missing or not in the SQL dialect map, the SQL option is **disabled** in the dropdown with a tooltip ("SQL download is unavailable for this connection"). This is preferable to throwing `UnsupportedFormat` mid-download.
- Optional password input + the password-info modal.
- The emit-event contract, renamed `@export` → `@download` (also rename in `SingleResultViewV1.vue:89-91` which only re-emits up to `ResultViewV1.vue`).
- Trigger label changes to "Download as file" via i18n key (`sql-editor.export-as-file` → `sql-editor.download-as-file`).

**Option payload (kept as a small type for the emit signature):**

```typescript
interface DownloadOption {
  format: ExportFormat;
  password: string;
}
```

**`DownloadContent` shape:** change to `{ blob: Blob; filename: string }` (was `{ content: Uint8Array; filename: string }`). Eliminates a useless ArrayBuffer→Uint8Array→Blob round-trip in the result-panel handler. `file-saver` accepts `Blob` directly.

### `ResultViewV1.vue::handleExportBtnClick` rewrite

```typescript
const handleDownloadBtnClick = async ({
  options,
  resolve,
  reject,
  result,                       // passed from each <DataExportButton> in the multi-result loop
}: {
  options: DownloadOption;
  resolve: (content: DownloadContent[]) => void;
  reject: (reason?: unknown) => void;
  result: QueryResult;
}) => {
  try {
    const databaseName = extractDatabaseResourceName(props.database.name).databaseName;
    const baseFilename = `${databaseName}.${dayjs(new Date()).format("YYYY-MM-DDTHH-mm-ss")}`;

    const { blob, filename } = await buildDownloadBlob({
      result,
      format: options.format,
      engine: props.database.instanceResource?.engine ?? Engine.ENGINE_UNSPECIFIED,
      baseFilename,
      password: options.password,
    });

    resolve([{ blob, filename }]);
  } catch (e) {
    reject(e);
  }
};
```

**Removed from request:** `statement`, `limit`, `admin`, `dataSourceId`, `schema`, `name`. No backend round-trip.
**Added:** engine from `props.database.instanceResource?.engine`. SQL format is gated upstream in the dropdown when engine is missing/unsupported, so this branch should not see `ENGINE_UNSPECIFIED` — but the fallback is defensive.

The downstream `DownloadContent[]` → `saveAs()` flow inside `DataExportButton.vue` calls `saveAs(blob, filename)` directly (no `Uint8Array` conversion).

### What we do NOT touch

- `useSQLStore().exportData()` — stays in the store. No frontend caller after this change; left in place for future use.
- `proto/v1/v1/sql_service.proto::ExportRequest` and the backend `SQLService.Export` RPC — unchanged.
- `DataExportButton.vue` callers outside `ResultViewV1.vue` / `SingleResultViewV1.vue` — there are none (verified).

### Locale changes

In `frontend/src/locales/*.json`:

- Rename `sql-editor.export-as-file` → `sql-editor.download-as-file`.
- Drop `export-data.export-rows`.
- Add `sql-editor.sql-download-unavailable` for the disabled-SQL-option tooltip.
- Keep `export-data.password-optional`, `export-data.password-info`, `export-data.export-format`.

Mechanical; same set of keys across all locale files.

## Behavior changes (user-visible)

These differ from today's backend Export path. Document in PR description.

- **Limit field removed.** Download exports exactly the rows already in the panel (capped at query time by `maximumResultRows` policy).
- **Admin / mask-bypass no longer applies.** The backend `Export` RPC re-runs the query and, when `admin=true`, skips masking. Client-side Download serializes the rows already in memory, which are post-masking from the original Query RPC. Admin users wanting unmasked output must use the Query path with admin mode (which already skips masking) or a future server-side Export Center per data-export-redesign.
- **No server-side audit log entry for downloads.** Today's `Export` RPC logs an audit event. Download is purely client-side; nothing reaches the server. If audit is required for the redesign's Export path, that's a separate (server-run) flow.
- **SQL format hidden for unsupported engines.** The format dropdown disables the SQL option when the connection's engine is unknown or not in the dialect map. Today the request would reach the backend and fail server-side.

## Memory soft cap

A hostile or accidental large query can OOM the browser before any size check fires. CSV / JSON / SQL build an intermediate string array of bytes ≈ N×M cells in size; XLSX materializes the whole sheet via `exceljs` (~5–10× the raw text); the optional ZIP step adds a final Blob. A 500 MB raw result becomes multi-GB transient memory.

Two-level guard, applied before serialization in `buildDownloadBlob`:

```typescript
const MAX_DOWNLOADABLE_CELLS = 5_000_000;          // cheap structural guard
const MAX_ESTIMATED_BYTES   = 200 * 1024 * 1024;   // 200 MB raw cell text

// (1) Cheap O(1) structural cap.
const cellCount = result.rows.length * result.columnNames.length;
if (cellCount > MAX_DOWNLOADABLE_CELLS) {
  throw downloadError("ResultTooLarge", ...);
}

// (2) O(N×M) byte estimate over the actual cells. Walks each RowValue,
// summing string-length / bytes-length / a small constant for scalars.
// Bails as soon as the running sum crosses MAX_ESTIMATED_BYTES — does NOT
// scan the whole result if the limit is hit early. Catches the
// "small grid, huge cells" case the cell cap alone misses.
const overByteCap = estimateRowBytes(result, MAX_ESTIMATED_BYTES);
if (overByteCap) {
  throw downloadError("ResultTooLarge", ...);
}
```

`ResultTooLarge` is a fourth `DownloadErrorCode`. Surfaced as a clear toast in the UI (locale key `sql-editor.download-too-large`). The `maximumResultRows` policy already caps row count at the query stage, so this check is a belt-and-suspenders guard; the byte estimate matters more in practice for results with large text/JSON cells.

`estimateRowBytes` lives in `value.ts` next to the cell helpers, walks `RowValue` once, treats scalars as 16 bytes, strings/bytes as their length plus framing overhead. **Crucially, it must not allocate any serialized output to compute the estimate** — that would defeat the purpose. For `value_value` (structpb), use a fixed conservative constant (e.g., 1024 bytes) rather than `JSON.stringify`-ing the structpb tree; the cap is approximate and a single dense structpb cell rarely dominates a result. Short-circuits as soon as the running sum exceeds the cap.

**Cap semantics for batch downloads.** `buildBatchDownloadBlob` (used by the multi-database `BatchQuerySelect.tsx` flow) applies the cell+byte caps **globally across all entries**, not per-entry. A user picking 10 databases must not be able to ZIP 10× the single-export budget. The pre-check sums `rows × columns` and the byte estimate across all entries before kicking off any inner serialization.

**Error reporting.** When `ResultTooLarge` is thrown, the error includes the estimated and cap values (in MB). The UI surfaces these via i18n interpolation so the user sees how much they're over (e.g. "Result is ~520 MB; client-side download limit is 200 MB"). For the cell-cap branch, the message includes the actual cell count.

## Error handling

Use a plain `Error` with a `code` field — no dedicated class.

```typescript
type DownloadErrorCode = "SerializationFailed" | "EncryptionFailed" | "UnsupportedFormat";
function downloadError(code: DownloadErrorCode, message: string, cause?: unknown): Error {
  const e = new Error(message);
  (e as Error & { code: DownloadErrorCode; cause?: unknown }).code = code;
  if (cause !== undefined) (e as Error & { cause?: unknown }).cause = cause;
  return e;
}
```

- `UnsupportedFormat` — engine outside SQL dialect map, or column count ≥ 18278 for XLSX. UI surfaces the engine case via the disabled dropdown option (rare to throw at runtime).
- `SerializationFailed` — wraps unexpected errors from format serializers (XLSX runtime, exceljs failures, JSON NaN/Inf rejection).
- `EncryptionFailed` — wraps zip.js errors.
- `ResultTooLarge` — row × column count exceeds the soft cap. Surfaced via a dedicated locale key with a clear remediation hint.

`ResultViewV1.vue`'s try/reject routes errors to the existing BBNotification toast.

## Testing

### Unit tests (`frontend/src/utils/sql-download/__tests__/`)

- `csv.test.ts`, `json.test.ts`, `sql.test.ts` — byte-equal assertions against goldens.
- `xlsx.test.ts` — load both XLSX outputs through `exceljs`, compare cell-by-cell (XLSX bytes are non-deterministic due to creation timestamp inside the zip).
- `value.test.ts` — RowValue → cell rules; all 12 oneof variants and edge cases.
- `zip.test.ts` — round-trip encrypt+decrypt a known payload with zip.js (skip byte-equality), and a fixture-based decrypt of a Go-generated ZIP from the backend (proves the reverse direction works).
- `index.test.ts` — top-level: `password=undefined` returns raw bytes with right MIME; `password="x"` returns ZIP.

### Standard escape fixture set

Used across `csv.test.ts` / `json.test.ts` / `sql.test.ts`:

- Empty result: zero rows.
- Empty columns: zero columns, zero rows. Zero columns with rows.
- Plain ASCII row.
- Strings containing each of: `'`, `"`, `\`, `\n`, `\r\n`, `\t`, `\x00` (NUL), `\x1b` (ESC).
- Non-BMP rune (emoji + CJK).
- Bytes: empty, one byte, 256 bytes covering 0x00–0xFF.
- Ints: `0`, `±1`, `int32` min/max, `int64` min/max (~`9223372036854775807`).
- Floats: `0`, `-0`, `1.5`, `1e21`, `1e-7`, `Number.MAX_VALUE`, `Number.EPSILON`, denormal, NaN, `+Inf`, `-Inf`, float32-only values (`Math.fround(1/3)`).
- Timestamps: epoch, far past (year 1900), far future, with non-UTC offset, with sub-microsecond nanos (verify truncation).
- structpb.Value: scalar, nested list, nested struct (verify deterministic key sort).

For SQL, run the escape fixtures against each engine in the dialect map (MySQL, Postgres at minimum; spot-check ClickHouse, Snowflake, Oracle, MSSQL, SQLite, MariaDB, TiDB, OceanBase, Spanner, Redshift).

### Known parity quirks accepted

These produce non-ideal output but match (or knowingly diverge from) backend. Filed as follow-ups, not fixed here.

- **JSON `value_value` → proto text format**, not structured JSON (backend `json.go:70`).
- **SQL `value_value` → CSV-style helper**, not engine-aware quoting (backend `sql.go:114`).
- **XLSX writes everything as text cells** (numbers included), parity with `excelize.SetCellValue(string)` calls.
- **Empty `column_names`** produces invalid SQL (`INSERT INTO ... () VALUES (...)`), shared with backend.
- **Postgres NUL byte** in string literals — `pq.QuoteLiteral` doesn't reject; Postgres does on insert. Backend-shared.
- **structpb StructValue field order — divergence from backend.** Frontend sorts keys; backend iterates Go map (non-deterministic). Frontend is the deterministic side and should be considered correct. Backend should sort too in a follow-up.
- **JSON int64 precision past 2^53.** Even backend's bare-number emit loses precision when the consumer is JS. Mirror exactly.

### Goldens generator (`go test -update` pattern)

In `backend/component/export/`, add a test that, when run with `-update`, writes goldens to `frontend/src/utils/sql-download/__tests__/goldens/{format}/{fixture}.{ext}`. Standard Go pattern; no separate `cmd/` binary.

```bash
# regenerate goldens after a backend serializer change
go test ./backend/component/export -run TestDownloadGoldens -update
```

Document this command in the test file's package comment and in this design doc. The non-`-update` mode of the same test verifies backend output still matches committed goldens (catches drift from backend side).

If frontend goldens-tests fail in CI, the failure message points the developer to that command.

### ZIP wire-compat parity tests (both directions)

- **Forward:** `backend/component/export/zip_frontend_parity_test.go` checks in a frontend-generated encrypted ZIP fixture and verifies `github.com/alexmullins/zip` opens it. Proves a customer's existing scripts using the backend's library will still read frontend output.
- **Reverse:** `zip.test.ts` checks in a backend-generated encrypted ZIP fixture and verifies zip.js decrypts it. Proves users receiving backend-Export output can also be served by frontend tooling (matters for any future "preview encrypted file" feature).

### Manual QA checklist

- Download CSV/JSON/SQL/XLSX from a small result, no password.
- Same with password — verify opens in macOS Archive Utility, Windows built-in Compressed Folders, and 7-Zip.
- Verify SQL format option is disabled for a connection without engine info, enabled for MySQL/Postgres.
- Verify SQL output for MySQL (backticks) and Postgres (`pq.QuoteLiteral` style).
- Verify masked columns appear as their masked value (no leakage).
- Verify XLSX click shows a brief loading indicator (first-click async chunk fetch is ~600KB gzipped).

## Dependencies

| Package | Change | Where |
|---|---|---|
| `@zip.js/zip.js` | **add** (~120 KB gzipped) | dynamic-import in `zip.ts` |
| `exceljs` | **add** (~600 KB gzipped) | dynamic-import in `formats/xlsx.ts` |
| `jszip` | **remove** | only frontend caller is `DataExportButton.vue`, deleted with the multi-file fallback |

Eager bundle additions on the SQL editor route: only `engines.ts` (a 12-entry `Set` and a predicate, ~200 bytes minified). `value.ts`, `formats/*`, `zip.ts`, `exceljs`, `@zip.js/zip.js` are all dynamic-imported and pull as async chunks on first Download click. The XLSX click shows a brief loading state to mask the chunk fetch.

## Rollout

Single PR. No feature flag.

- Change touches the Vue and React `DataExportButton` wrappers and the `ResultViewV1.vue` / `BatchQuerySelect.tsx` consumers — no other UI surfaces.
- Format parity verified by goldens before merge.
- ZIP wire compat verified by Go round-trip test.
- Backend `Export` RPC unchanged; zero risk to non-SQL-editor surfaces.

### CI gate

Goldens parity is enforceable only if the drift test runs whenever **either** side changes. Because Bytebase CI uses path-based triggers, a frontend-only PR that doesn't touch `backend/component/export/` could otherwise miss the drift check. The CI configuration (`.github/workflows/*` or equivalent) must trigger `go test ./backend/component/export -run TestDownloadGoldens` on PRs that modify any of:

- `backend/component/export/**`
- `frontend/src/utils/sql-download/__tests__/goldens/**`
- `frontend/src/utils/sql-download/**`

Without this rule, a developer can change the TS implementation and the committed goldens together (regenerating them with `-update`) without ever exercising backend output, making the goldens a stale artifact rather than a contract.

### Fixture sync between Go and TS

Two parallel fixture maps (`downloadFixtures()` in Go and `FIXTURES` in TS) introduce drift risk. To make drift loud:

- The Go goldens generator emits a `frontend/src/utils/sql-download/__tests__/goldens/fixture_ids.txt` manifest (one ID per line, sorted) on `-update`.
- A TS test asserts `Object.keys(FIXTURES).sort().join("\n") + "\n"` matches the manifest. Adding a fixture in Go and forgetting to add it in TS (or vice versa) fails this test loudly with a clear message.

## Out of scope (filed as follow-ups)

1. ~~Migrate backend + frontend ZIP encryption from ZipCrypto to AES-256.~~ Done — both already produce WinZip AES-256.
2. Fix the JSON `value_value` → `.String()` and SQL `value_value` → CSV-helper quirks (both should produce structured JSON when fixed; backend + frontend together).
3. Make XLSX numeric cells actually numeric (currently text-typed for parity).
4. Sort `structpb.StructValue` field keys in backend's CSV/SQL helpers so backend matches the new frontend behavior (deterministic).
5. Reject empty `column_names` in SQL export instead of producing invalid SQL (backend + frontend).
6. Implement the data-export-redesign UI split (`DownloadButton` vs `ExportButton`).
7. Server-side audit log for the Download path if/when product wants Download events recorded.

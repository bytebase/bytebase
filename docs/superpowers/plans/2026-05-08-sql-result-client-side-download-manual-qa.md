# SQL Result Client-Side Download — Manual QA Plan

Companion to `2026-05-08-sql-result-client-side-download.md`. Use this as the
gate before merging the `feat/download` branch. Tick each box as you go; the
plan is grouped so you can stop after Section 5 if time is short.

## Pre-flight

```bash
# Backend (one terminal, port 8080)
PG_URL=postgresql://bbdev@localhost/bbdev go run ./backend/bin/server/main.go --port 8080 --data . --debug

# Frontend (another terminal)
pnpm --dir frontend dev
```

Open http://localhost:3000, log in as admin (default seed credentials).

**Dev gate:** `pnpm --dir frontend dev` sets `import.meta.env.DEV = true`, so
`isDev()` returns `true` and the React result panel handlers
(`ResultView.tsx::handleExport`, `BatchQuerySelect.tsx::handleExport`)
short-circuit into `buildDownloadBlob`. Sections 1–13 below all exercise
the **client-side path**. Section 13a (added at the end) tests the
**production fallback path** (`!isDev()`) — run that section against a
production build (`pnpm --dir frontend build && pnpm --dir frontend
preview`) to verify main's existing React backend Export RPC code still
works. No `DataExportButtonLegacy.*` sibling files exist; the existing
React code IS the legacy path.

**Browser:** Test in Chromium first. Spot-check Safari + Firefox for the XLSX
async-chunk path.

**Tools to keep handy:**
- `unzip -P <password> -p file.zip <inner>` — POSIX zip decrypt
- macOS Archive Utility (drag-drop a .zip)
- 7-Zip CLI on Linux: `7z l -p<password> file.zip`
- File watcher: `ls -lhS ~/Downloads | head` between clicks
- DevTools Network/Performance panels

---

## 1. Smoke (5 min — must pass before continuing)

**Note: every download is now a ZIP** (matches backend layout). The outer file
is `<db>.<ts>.zip`; inside is `<instance>/<db>/statement-1.sql` (the running
SQL) plus `statement-1.result.<ext>` (the formatted result).

Create one query in any connected database (Postgres or MySQL is fine):

```sql
SELECT 1 AS id, 'hello' AS name;
```

- [ ] CSV download: file `<db>.<ts>.zip` lands in Downloads. Inside: `<instance>/<db>/statement-1.sql` (containing the SQL text) + `<instance>/<db>/statement-1.result.csv` (header `id,name`, row `1,"hello"`).
- [ ] JSON download: `…statement-1.result.json` shape `[{"id":1,"name":"hello"}]`, two-space indent.
- [ ] SQL download: `…statement-1.result.sql` containing `INSERT INTO …`. The `…statement-1.sql` file contains the original `SELECT 1 AS id, 'hello' AS name;` query.
- [ ] XLSX download: `…statement-1.result.xlsx` opens in Numbers/Excel/LibreOffice; one sheet, two rows, header in row 1.
- [ ] Password CSV: enter password `qa-test-2026`. ZIP entries are encrypted. `unzip -P qa-test-2026 file.zip` produces both `.sql` + `.result.csv` files in their `<instance>/<db>/` subdirectories.

If any of the above fails — STOP and file before continuing. Most code-path coverage flows through here.

---

## 2. Per-engine SQL parity (15 min — pick 3 engines you have access to)

For each engine in your environment, run a tiny query and verify the SQL output uses the right identifier quote and string-literal style:

| Engine    | Identifier quote | String literal     | Check |
|---|---|---|---|
| MySQL/MariaDB/TiDB/OceanBase/Spanner | `` ` `` (backtick) | `'…'`, `\` not special | [ ] |
| Postgres  | `"`              | `'…'`, `' E'…\\…'` when contains `\` | [ ] |
| Redshift  | `"`              | same as Postgres | [ ] |
| MSSQL     | `"`              | `'…'`             | [ ] |
| Oracle    | `"`              | `'…'`             | [ ] |
| ClickHouse| `"`              | `'…'`             | [ ] |
| Snowflake | `"`              | `'…'`             | [ ] |
| SQLite    | `"`              | `'…'`             | [ ] |

Test query:

```sql
SELECT 'it''s a test' AS message, 'a\b' AS escaped;
```

For Postgres, the escaped row should produce `' E''it''''s a test''',E''a\\b''` (note the leading space + `E'` form because of `\`). For MySQL it should produce `'it''s a test','a\b'`.

- [ ] Connect each engine you have, run the query, download SQL, eyeball.

For engines NOT in your environment, the unit-test goldens cover them — see `frontend/src/utils/sql-download/__tests__/goldens/sql/<id>.<engine>.sql`.

---

## 3. Data-type coverage (20 min — Postgres recommended)

Postgres covers the widest type set. Create a test table:

```sql
CREATE TABLE qa_download (
  id            INT PRIMARY KEY,
  name          TEXT,
  bigint_col    BIGINT,
  uint_col      BIGINT,           -- Postgres has no unsigned, use bigint
  float_col     REAL,
  double_col    DOUBLE PRECISION,
  decimal_col   NUMERIC(20,5),
  bool_col      BOOLEAN,
  bytes_col     BYTEA,
  ts_col        TIMESTAMP,
  tstz_col      TIMESTAMPTZ,
  json_col      JSONB,
  null_text     TEXT,
  null_int      INT
);

INSERT INTO qa_download VALUES
  (1, 'simple ASCII',
   9223372036854775807, 0,
   1.5, 1.5e21, 12345.67890,
   true,
   '\xdeadbeef'::bytea,
   '2023-11-14 22:13:20'::timestamp,
   '2023-11-14 22:13:20+09:00'::timestamptz,
   '{"a":1,"b":[1,2,3]}'::jsonb,
   NULL, NULL),

  -- Edge case strings
  (2, 'with "quotes" and ''apostrophes''',
   -9223372036854775808, 18446744073709551615,
   0, -0.0, -99999.99999,
   false,
   '\x00ff80'::bytea,
   '1970-01-01 00:00:00.000001'::timestamp,
   '2023-11-14 22:13:20.5-07:00'::timestamptz,
   '{"k\"y":"v\nl","nest":{"n":null}}'::jsonb,
   NULL, NULL),

  -- Multibyte / control / NUL
  (3, E'multibyte 中文 日本 emoji 😀 NUL\x00 newline\nESC\x1B BOM\xEF\xBB\xBF',
   0, 0, 0, 0, 0,
   true,
   ''::bytea,
   '2024-02-29 03:30:45.123456'::timestamp,
   '2024-02-29 03:30:45.123456+05:45'::timestamptz,  -- non-hour offset
   '"a\"b\nc"'::jsonb,
   NULL, NULL);
```

Run `SELECT * FROM qa_download ORDER BY id;` and download as each format.

### Verifications

- [ ] **CSV**:
  - Row 2 column `name` shows `"with ""quotes"" and ''apostrophes''"` (CSV-escaped via doubled `"`, unchanged apostrophes).
  - Row 3 `name` shows the multibyte chars verbatim, NUL byte rendered visibly (consumer-defined; opening in Excel may show a control box).
  - `bytes_col` values render as `0xdeadbeef`, `0x00ff80`, empty (the 3 rows).
  - `null_text` and `null_int` cells are empty (no quotes, no `NULL`).
  - Row 1 `tstz_col` reads `2023-11-14T13:13:20Z` (note Postgres normalizes to UTC if your session TZ is UTC; otherwise the offset varies — golden parity holds either way).
  - `decimal_col` rounds-trips with full precision (Postgres returns NUMERIC as a string, so this is a stringValue path — verify no scientific notation).

- [ ] **JSON**:
  - Indented 2 spaces, lexicographic key order in objects (Go's `json.MarshalIndent` of `map[string]any`).
  - `bigint_col` row 1 is the literal token `9223372036854775807` (NOT `9.223372036854776e+18` — bigint precision retained).
  - `bool_col` is the JSON `true`/`false` literal.
  - `bytes_col` is base64 (`3q2+7w==`, `AP+A`, `""`).
  - `null_text`/`null_int` rows: JSON `null` value.
  - `json_col` is a STRING (proto-text shape) NOT a nested JSON value — that's by design (structpb passes through `xlsxStringFromStructpbValue`).
  - HTML-special chars `<`, `>`, `&` are escaped to `<` etc. in any string cell that contains them. Try adding a row with `'<script>alert(1)</script>'`.

- [ ] **SQL** (Postgres):
  - Multi-line `INSERT` statements separated by `\n`, no trailing newline.
  - Row 2 `name` becomes `'with "quotes" and ''apostrophes'''` (single quotes doubled; double quotes pass through).
  - Row 3 `name` uses the `E'…'` form because of the embedded backslash (from `\x00` representation? No — embedded raw NUL doesn't trigger E mode. But `\xEF\xBB\xBF` BOM is a UTF-8 byte sequence, not a `\` literal). Actually verify: only literal `\` triggers E mode, not Unicode bytes that happen to be 0x5C-adjacent. Row 1 `bytes_col` becomes `0xdeadbeef` (un-quoted hex literal — note this is NOT valid Postgres SQL, but matches backend behavior).
  - `null_*` cells become bare `NULL` (no quotes).
  - `bool_col` becomes `true`/`false`.
  - `tstz_col` becomes a quoted ISO-8601 string.

- [ ] **XLSX**: open in Excel/Numbers/LibreOffice.
  - Row 1 = headers, rows 2-4 = data.
  - All cells are stored as text strings (note: opening in Excel will see numeric strings; this is intentional parity with backend).
  - `null_*` cells appear empty.
  - `bytes_col` shows base64.
  - Multibyte text renders correctly.

---

## 4. Column-name edge cases (10 min)

This is the area where backend has known gaps (see follow-ups doc) — verify TS matches backend behavior, NOT that the output is well-formed.

```sql
-- Postgres
SELECT 1 AS "a,b", 2 AS "c""d", 3 AS "e
f";
```

- [ ] CSV header should literally read `a,b,c"d,e<newline>f` (broken — 5 columns by comma, embedded newline). This matches backend's current bug; do NOT report as new.
- [ ] SQL Postgres: `INSERT INTO "<table_name>" ("a,b","c"d","e<newline>f") VALUES ...` — broken identifier quoting. Same.

If you have customer impact concerns on this, flag in the follow-ups doc — TS cannot fix it without backend matching.

---

## 5. Encryption / ZIP correctness (15 min)

### Round-trip with multiple unzippers

Same file (the row-2 multibyte query above), CSV format, password `Test123!@#`.

- [ ] `unzip -P 'Test123!@#' -p file.zip` returns the CSV bytes unchanged
- [ ] macOS Archive Utility prompts for password, extracts to `<base>.csv`
- [ ] 7-Zip CLI: `7z x -p'Test123!@#' file.zip` succeeds
- [ ] Windows File Explorer's built-in ZIP support — it does NOT support encrypted ZIPs natively in older versions, but modern Windows 11 should prompt. If it doesn't, that's expected (older Windows behavior).

### Adversarial passwords

- [ ] Empty password (just hit "OK" with blank field) — file should download as plain `.csv`, not `.zip` (the `password && password.length > 0` guard).
- [ ] Single-char password `x` — encrypts; unzip with `-P x` succeeds.
- [ ] Long password (200 chars random ASCII): encrypts; correct password decrypts; wrong password fails decoder cleanly.
- [ ] Multibyte password `密码-2026-😀`: encrypts; same exact UTF-8 bytes decrypt successfully. (zip.js & alexmullins/zip both handle UTF-8 passwords.)
- [ ] Spaces in password ` lead trail `: round-trips.

### Encryption format

```bash
# After a download with password set, verify the actual encryption header
xxd file.zip | head -3
```

- [ ] Look for method=99 (extra-field 0x9901) — confirms WinZip AES-256, not classic ZipCrypto.

---

## 6. Batch query (15 min)

Use the Batch Query feature: pick 2-4 databases.

### Setup

Pick 2 databases on the SAME engine (e.g. two MySQL DBs):

```sql
-- Run on each
CREATE TABLE qa_batch (id INT, name TEXT);
INSERT INTO qa_batch VALUES (1, 'a'), (2, 'b');
SELECT * FROM qa_batch;
```

### Functional tests

- [ ] Batch download CSV (no password): produces `batch-download-<ts>.zip`. Layout:
      ```
      batch-download-<ts>.zip
      ├── <instance-A>/
      │   └── <db-1>/
      │       ├── statement-1.sql
      │       └── statement-1.result.csv
      └── <instance-A>/
          └── <db-2>/
              ├── statement-1.sql
              └── statement-1.result.csv
      ```
- [ ] Inner files have correct content (per-database).
- [ ] **SQL format enabled** when all selected DBs are on supported engines (MYSQL/POSTGRES/etc.).
- [ ] **SQL format DISABLED** when at least one selected DB is on an unsupported engine (MongoDB, Redis, etc.). Hover over the disabled radio: tooltip says "SQL download is unavailable for this connection".
- [ ] Batch with password: every inner entry encrypted with the same password.
- [ ] Two databases with the same name on different instances (e.g. `prod` on instance-A and `prod` on instance-B): natural directory separation by `<instanceId>`, no collision.

### Cross-engine batch

Pick one MySQL DB and one Postgres DB.

- [ ] Batch CSV: works (CSV is engine-agnostic). Each `<instance>/<db>/statement-1.result.csv` reflects its own data.
- [ ] Batch JSON: works.
- [ ] Batch SQL: each `<instance>/<db>/statement-1.result.sql` uses its own engine's quote style (MySQL backticks, Postgres double quotes). Verify by extracting and inspecting both.
- [ ] Batch XLSX: each `<instance>/<db>/statement-1.result.xlsx` is a valid `.xlsx`.

### Cross-engine batch with multi-statement

Run a batch query where multiple databases each return multiple statements (some test SQL like `SELECT 1; SELECT 2;` per database).

- [ ] Each `<instance>/<db>/` subdirectory contains `statement-1.sql`, `statement-1.result.<ext>`, `statement-2.sql`, `statement-2.result.<ext>`, ... matching that database's statements.

---

## 7. Multi-statement download — backend-parity layout (5 min)

Run a multi-statement query:

```sql
SELECT 1 AS first;
SELECT 2 AS second;
SELECT 3 AS third;
```

- [ ] Result panel shows 3 tabs (Query #1, Query #2, Query #3).
- [ ] Click "Download" on the tab-strip suffix → CSV format → file `<db>.<ts>.zip` lands. Extract: structure mirrors the screenshot from the spec —
      ```
      <db>.<ts>.zip
      └── <instanceId>/
          └── <databaseName>/
              ├── statement-1.sql        (text: "SELECT 1 AS first")
              ├── statement-1.result.csv (header + 1 row)
              ├── statement-2.sql        (text: "SELECT 2 AS second")
              ├── statement-2.result.csv
              ├── statement-3.sql
              └── statement-3.result.csv
      ```
- [ ] Each `statement-N.sql` contains exactly that statement's text (verify by `cat` or quick-look).
- [ ] Repeat with JSON, SQL, XLSX → only `statement-N.result.<ext>` extension changes; the `.sql` files stay `.sql`.
- [ ] Password set → every entry inside the ZIP is encrypted; structure unchanged.
- [ ] Single visible result: run `SET search_path = public; SELECT 1;` so only `SELECT 1` makes it past the SET filter. Output is still a ZIP with `<instance>/<db>/statement-1.sql` + `statement-1.result.csv` (NOT a bare CSV).
- [ ] Active-tab independence: switch to Query #2, click Download. ZIP still contains all 3 statements — the active tab does not change which statements are bundled.

---

## 8. Soft-cap / size limits (10 min)

Two caps in `frontend/src/utils/sql-download/index.ts`:

- `MAX_DOWNLOADABLE_CELLS = 5_000_000` (rows × cols)
- `MAX_ESTIMATED_BYTES = 200 MiB` (rough UTF-8 byte estimate)

### Cell cap

Construct a query that exceeds 5M cells. Easiest: a wide query result.

```sql
-- Postgres
SELECT generate_series(1, 60000) AS r,
       1, 2, 3, 4, 5, 6, 7, 8, 9, 10,
       11, 12, 13, 14, 15, 16, 17, 18, 19, 20,
       -- … 100 columns total …
       91, 92, 93, 94, 95, 96, 97, 98, 99, 100;
```

60_000 × 101 ≈ 6.06M cells > cap.

- [ ] Click any download format. Notification appears: "Result has 6,060,000 cells; client-side download limit is 5,000,000. Reduce row count or use server-side export."

### Byte cap

```sql
SELECT repeat('x', 25 * 1024 * 1024) AS big_string FROM generate_series(1,10);
```

10 rows × 1 col × 25 MiB ≈ 250 MiB > 200 MiB cap.

- [ ] Notification: "Result is ~750 MB; client-side download limit is 200 MB."

(The `*3` UTF-8 upper bound makes the estimate conservative — pure ASCII shows about 3× true size, which is fine for a soft cap.)

### Boundary

```sql
-- Just under the cap: 49,999 rows × 100 cols = 4,999,900 cells
SELECT generate_series(1, 49999) AS r, ... [99 more columns]
```

- [ ] Just-under works (download succeeds).
- [ ] Just-over (50_001 × 100) fails with the cap notification.

---

## 9. Engine gating (5 min)

For an unsupported-SQL engine connection:
- MongoDB, Redis, Cassandra, BigQuery, CockroachDB, Databricks, Trino — pick whichever you have.

Run any query and try downloading.

- [ ] CSV / JSON / XLSX downloads fine.
- [ ] SQL radio in the drawer is **disabled**, with tooltip "SQL download is unavailable for this connection".
- [ ] In dropdown mode (if used), the SQL menu item is disabled with the same tooltip.
- [ ] Programmatically forcing SQL via DOM (DevTools, set `formData.format = 2 // SQL`): clicking Confirm does nothing (button stays disabled) — defense-in-depth gate.

---

## 10. Performance (20 min — record numbers)

These are *guidelines*, not hard SLAs. Modern laptop, no other heavy tabs open.

### 10a. Wide-narrow

```sql
-- 10K rows × 10 cols, all int — should be fast
SELECT g, 1, 2, 3, 4, 5, 6, 7, 8, 9 FROM generate_series(1, 10000) AS g;
```

| Format | Expected wall-clock | Recorded |
|---|---|---|
| CSV    | <300ms | _____ |
| JSON   | <500ms | _____ |
| SQL    | <500ms (Postgres) | _____ |
| XLSX   | 2-5s (incl. exceljs lazy chunk fetch on first click) | _____ |

- [ ] All four under threshold.

### 10b. Tall

```sql
-- 100K rows × 5 cols
SELECT g, 'cell' || g, g * 1.5, true, NOW() FROM generate_series(1, 100000) AS g;
```

| Format | Expected wall-clock | Recorded |
|---|---|---|
| CSV    | <1s | _____ |
| JSON   | <2s | _____ |
| SQL    | <3s | _____ |
| XLSX   | <10s | _____ |

- [ ] All under threshold.
- [ ] UI does not lock up; cancel button (if shown) responds during XLSX.

### 10c. Wide

```sql
-- 5K rows × 50 cols of mixed types
SELECT g, 'short string', g * 3.14, true, NOW(),
       g + 1, 'another', g * 2.71, false, NOW(),
       -- ... 10 col group repeated 5 times ...
FROM generate_series(1, 5000) AS g;
```

| Format | Expected wall-clock | Recorded |
|---|---|---|
| CSV    | <500ms | _____ |
| XLSX   | <8s | _____ |

### 10d. Big strings

```sql
-- 100 rows × 1 col, each 1 MiB
SELECT repeat('a', 1024 * 1024) FROM generate_series(1, 100);
```

100 rows × 1 col × 1 MiB ≈ 100 MiB. Under the 200 MiB cap.

| Format | Expected wall-clock | Recorded |
|---|---|---|
| CSV    | <2s | _____ |
| JSON   | <5s (escape every byte) | _____ |
| ZIP-encrypted CSV | <8s (AES-256 + compress) | _____ |

- [ ] Output file size sane: CSV ≈ 100 MB; ZIP-CSV ≈ 70-90 MB after deflate.

### 10e. ZIP encryption overhead

Same 100K-row query as 10b, with password.

- [ ] Time delta vs unencrypted (CSV → CSV.zip): under 2× slowdown.
- [ ] DevTools Performance tab: main thread frame >16ms during encryption is OK (we use `useWebWorkers: false` because workers had CSP issues — captured as a follow-up to revisit). User-perceivable freeze should be <2s for 100K rows.

### 10f. Memory

DevTools → Memory tab → "Take heap snapshot" before download, then click download, snapshot during, then after.

- [ ] During serialization, heap should peak around 2-3× the result-set in-memory size (TS string allocations during serialize). Expect 100K-row test (~30 MB result) to peak around 100-150 MB.
- [ ] After download completes, heap returns to baseline (Blob is held only by the `<a download>` click).

---

## 11. Cross-tool extraction matrix (10 min)

Generate one encrypted ZIP (any small query, any password):

| Extractor | Decrypt | Notes |
|---|---|---|
| `unzip -P` (POSIX)  | [ ] |   |
| macOS Archive Utility (drag-drop) | [ ] | Prompts for password |
| 7-Zip CLI 7z 22+    | [ ] |   |
| 7-Zip Windows GUI   | [ ] |   |
| WinRAR 6+           | [ ] |   |
| PeaZip              | [ ] |   |
| Windows File Explorer (Win 11 23H2+) | [ ] | Older Windows lacks AES support |

If any tool fails: note its version and the specific error. Some niche
tools only support classic ZipCrypto and can't read AES-256; that's expected
since we pick AES for parity with backend's `alexmullins/zip` library.

---

## 12. Failure paths (5 min)

- [ ] Download with no rows (`SELECT 1 WHERE FALSE`): empty CSV `header\n`, empty JSON `[]`, empty SQL (zero bytes), empty XLSX (header-only sheet). All four download successfully.
- [ ] Download while a query is still running: download button should be disabled (no result available). Verify.
- [ ] Network offline + first XLSX click: `await import("exceljs")` fails. Notification: "Failed to serialize result" or similar. Re-online and retry: works.
- [ ] Concurrent clicks: rapid double-click on "Download CSV". Should not produce two files (button enters loading state on first click).

---

## 13a. Production-path fallback — `!isDev()` (10 min)

The client-side path is `isDev()`-gated; production builds run main's
existing React backend-Export-RPC path verbatim. There are **no
`DataExportButtonLegacy.*` sibling files** — the existing React handlers
in `ResultView.tsx` and `ResultPanel/BatchQuerySelect.tsx` IS the legacy
path. The `isDev()` branch is an early-return inside each handler that
short-circuits to `buildDownloadBlob`; falling through executes main's
verbatim `useSQLStore().exportData()` code.

```bash
# Build a production bundle and serve it locally
pnpm --dir frontend build
pnpm --dir frontend preview
```

The preview server runs on a different port (typically `:4173`).

- [ ] Click the download drawer "Confirm" → server-side `Export` RPC fires (DevTools Network shows `/v1/sql.SQLService/Export`).
- [ ] Output file is a `.zip` matching backend's structure (`<instance>/<db>/statement-N.{sql,result.<ext>}`).
- [ ] `query_history` table on the backend has a new Export row after the download (verify via `psql -U bbdev bbdev -c "SELECT type, created_at FROM query_history ORDER BY created_at DESC LIMIT 3"`).
- [ ] Admin mode (`tab.mode === "ADMIN"`) → `request.admin = true` → mask-bypass applies on the backend.
- [ ] Multi-database batch download → server returns one ZIP per database; JSZip wraps them into the outer ZIP (legacy path uses `jszip`, not `@zip.js/zip.js`).
- [ ] SQL format for an unsupported engine (MongoDB, etc.) → backend returns a runtime error, surfaced as a toast notification (no client-side gating in the legacy path).

This section is a smoke test, not exhaustive. If any item fails, main's
existing React export path is broken — file the issue against this PR.

## 13. Regression sweep (5 min)

Re-run the smoke tests from Section 1 immediately before merging. Catches anything that drifted between the start of QA and now.

- [ ] CSV  [ ] JSON  [ ] SQL  [ ] XLSX  [ ] Password CSV

---

## QA report template

When done, paste this into the PR description as the "QA results":

```
Manual QA: <date>, <name>, <platform/browser>

Sections completed: 1 ☐  2 ☐  3 ☐  4 ☐  5 ☐  6 ☐  7 ☐  8 ☐  9 ☐  10 ☐  11 ☐  12 ☐  13 ☐

Performance numbers (10b 100K rows):
  CSV:    ___ ms
  JSON:   ___ ms
  SQL:    ___ ms
  XLSX:   ___ s

Cross-tool extraction (Section 11):
  Pass: ___
  Fail: ___ (with notes)

Issues found: [link to filed bugs]

Followups confirmed in scope:
  - server-side audit log (top priority)
  - multi-statement download (top priority)
```

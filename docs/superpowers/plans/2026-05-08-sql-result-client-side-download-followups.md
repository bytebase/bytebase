# SQL Result Client-Side Download — Follow-Ups

Companion to `2026-05-08-sql-result-client-side-download.md`. The main PR ships
client-side download for the SQL editor result panel **behind `isDev()`**; the
items below are known scope cuts that the team agreed to track separately.

## Ship gate

**Client-side download is `isDev()`-only.** `import.meta.env.DEV === true`
(i.e. `pnpm dev`) routes to `buildDownloadBlob` inside the React result
panel's existing handlers (`ResultView.tsx::handleExport`,
`BatchQuerySelect.tsx::handleExport`). Production builds fall through the
same handler to main's verbatim `useSQLStore().exportData()` backend Export
RPC path — **no `DataExportButtonLegacy.*` sibling files exist; the
existing React code IS the legacy path**. Each handler is structured as:

```ts
if (isDev()) { /* buildDownloadBlob branch */ return; }
/* untouched main-branch Export RPC code */
```

Flipping the switch when client-side download ships GA is therefore a pure
deletion: remove the `isDev()` branches and the `import { isDev } from
"@/utils/util"` line in `ResultView.tsx` and
`ResultPanel/BatchQuerySelect.tsx`. No file moves, no rename dance.

Flip-the-switch checklist before promoting:

- [ ] Server-side audit-log restoration (top-priority item #1 below)
- [ ] Compliance review of the access-check model change ("can see ≡ can download")
- [ ] Customer-visible behavior callouts in release notes (row-limit field removed, mask-bypass dropped, in-tab session dependency)
- [ ] Confirm AES-256 ZIP UX across the target customer's unzip tools (Section 11 of the manual QA plan)

## PR breaking-changes (concise — paste into the PR body under "Breaking changes")

Behind `isDev()`. Production behavior is verbatim main: same React handlers,
same `useSQLStore().exportData()` calls, same `DataExportButton` UI, same
output shape. **When the flag flips on, customers see these differences vs the backend Export RPC:**

- **Audit log:** server no longer sees the download; no `query_history` Export row. *(Top-priority follow-up #1.)*
- **Row-limit field removed:** download exports exactly what's in the result panel, capped at query time by `maximumResultRows` policy.
- **Admin / mask-bypass dropped:** rows are post-masking; no separate elevated-export path.
- **Error abort changed:** multi-statement with one failing statement now downloads the successful statements anyway (backend used to abort and return nothing).
- **SQL gating UX:** unsupported engines show a disabled radio + tooltip instead of a runtime server error.
- **Result size caps:** ~5M cells / ~200 MiB (in-RAM) instead of server-side streaming; very large exports must keep using the server Export RPC path (issue/rollout export, untouched here).
- **Session-bound:** closing the tab cancels the in-flight serialization. (Server export survives tab close.)
- **Telemetry:** zero server-side metrics for downloads.

Output layout unchanged: `<baseFilename>.zip` containing `<instanceId>/<databaseName>/statement-N.sql` + `statement-N.result.<ext>`. WinZip AES-256 when password set. Verified by `TestZipFrontendParity` (Go reads frontend ZIP).

## Top priority

### 1. Restore server-side audit log entry for downloads

**Status this PR:** Dropped. The client-side path never round-trips to the
backend, so `Export` RPC's audit trail (`activity_log`, `slow_query` etc.) is
gone for this surface.

**Why it matters:** Audit logs may be required for SOC 2 / ISO 27001 / customer
compliance — "who downloaded what database's data when". Today, a customer
running a Bytebase tenancy can no longer answer that question for SQL editor
result-panel downloads.

**Approach options:**
- (A) Fire a non-blocking telemetry RPC (`SQLService.RecordDownloadEvent`) from
  `buildDownloadBlob` callers. Backend writes the audit row with a synthetic
  "client-side download" event type. Failure of the RPC must not block the
  download.
- (B) Keep the server-side `Export` RPC as the single source of audit, but
  have it return the same `QueryResult` proto (already in memory on the
  server during query execution) and let the client serialize. This loses
  the roundtrip benefit.
- (A) is preferred: keeps the client-side ergonomics, restores audit.

**Test plan when implemented:** Existing audit log integration tests must
catch the new event type; add a test asserting one row per download even on
batch (one row per inner result).

### 2. Restore multi-statement download — RESOLVED (with backend-parity layout)

**Status:** Fixed in this PR. Then upgraded to match backend's exact ZIP
structure (commits after the manual-QA doc landed).

**What ships:** Every download — single statement, multi-statement, or
multi-database batch — produces the same ZIP layout backend's
`exportResultToZip` produces:

```
<baseFilename>.zip
└── <instanceId>/
    └── <databaseName>/
        ├── statement-1.sql        (the SQL text)
        ├── statement-1.result.csv (the formatted result)
        ├── statement-2.sql
        ├── statement-2.result.csv
        └── ...
```

The single public API `buildDownloadBlob({ groups, format, baseFilename,
password? })` handles all three cases via the `groups` shape:

- 1 group, 1 statement → single SQL editor query
- 1 group, N statements → multi-statement query (3 statements → 3 sql + 3 result files)
- N groups → batch query (each (instance, database) becomes its own subtree)

Path segments are sanitized via `sanitizeBasename` so a malicious database
name can't escape the intended directory tree.

## Accepted scope cuts (documented, not on the priority list)

- **Export rows limit field removed.** Exports the rows already in the panel
  (capped at query time by `maximumResultRows` policy). Less flexible than
  the old form but matches the in-memory data model. Re-adding requires a
  server-side re-query, defeating the client-side path.
- **Admin / mask-bypass no longer applies.** Downloads are post-masking
  because the client only has post-masking rows. Admin still has the bypass
  via the result panel itself; this only affects download-as-file.
- **SQL format option disabled when engine is unknown.** Backend's old
  behavior was to attempt SQL serialization and fail server-side; new
  behavior is friendlier — option is greyed out with a tooltip.

## Other deferred items (from the review-loop docs)

- Backend `SQLStatementPrefix` and `csv.go` header-write don't escape column
  names containing `` ` ``, `"`, `,`, or `\n`. The TS port intentionally
  mirrors the backend bugs (captured by `column_name_quotes_my/pg` goldens)
  so the wire-format contract holds. Fix requires backend + TS + goldens
  regen in lockstep.
- `prototext.Format` whitespace is officially non-deterministic across
  protobuf-go versions. The TS port pins double-space (`PROTOTEXT_SEP`) to
  match protobuf-go ≥ v1.36's current output. A protobuf-go version bump can
  silently flip this; the comment in `value.ts::xlsxStringFromStructpbValue`
  documents the lockstep update.
- Refactor opportunities surfaced by `/simplify`: unify the three
  `*ValueFromRowValue` switches behind a strategy table; consolidate
  `wrapWithEncryptedZip` + `wrapWithMultiEntryZip`; share the `HEX` lookup
  table between `formats/json.ts` and `value.ts`. None affect output;
  separate cleanup PR.

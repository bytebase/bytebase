# Bytebase schema plugin

This file provides additional guidance to AI coding assistants working under
`./backend/plugin/schema/`, the per-engine conversions between database metadata
and DDL.

## Inheritance

- Follow the repository-wide guidance in `../../../AGENTS.md`.
- Treat this file as schema-plugin-specific additions, not a replacement for the
  root instructions.

## What these packages test

Each engine package owns three conversions, and a test belongs to exactly one of
them:

- `GetDatabaseMetadata` — schema text to metadata.
- `GetDatabaseDefinition` — metadata to DDL.
- `schema.GetDatabaseSchemaDiff` plus the package's `generateMigration` —
  a metadata pair to migration DDL.

Test that code and nothing behind it. Do not call `driver.SyncDBSchema` here, and
never use a live server's answer as the expected value: whether our output matches
what an engine stores is engine conformance, and it belongs in omni, which runs
its own container oracles. No test in these packages starts a container.

Chaining the conversions costs coverage rather than adding it. A definition test
fed from `GetDatabaseMetadata` reaches only as far as the parser extracts, and
silently skips every generator branch nothing feeds. So a metadata-to-DDL test
takes metadata as its input, written out in the fixture, including fields no
parser in this repo produces. Separating them this way is what surfaced the TiDB
parser dropping `primary_key_type`, per-column `character_set`/`collation`,
`CHECK` constraints and bare `AUTO_RANDOM` — none of which the chained form
would have missed loudly.

## Golden fixtures and the record process

Fixtures live in `testdata/` as a YAML list of cases, one output field per case:
one file per test, named after the source file it covers, with no per-case files
and no splitting a test's cases across several fixtures. Each test carries the
fixture path and a `record` constant:

```go
const (
	record   = false
	filepath = "testdata/get_database_definition.yaml"
)
```

To regenerate, set `record` to `true`, run that one test, and set it back:

```bash
go test ./backend/plugin/schema/tidb/ -run '^TestGetDatabaseDefinition$' -count=1
```

Record mode assigns each produced value into its case and writes the file through
`yamltest.Record`, which pins 2-space indentation so that re-recording unchanged
output is a no-op.

- Commit the fixture and the restored `record = false` together. A `record = true`
  left in the tree overwrites the goldens on every run, so the test always passes.
- Read the fixture diff before committing it. A re-record captures current behavior,
  bugs included, and a golden that moved because output regressed is indistinguishable
  from one that moved because output improved.
- Seed a new case by adding it with an empty output field and recording once.
- Two fixtures deviate, and the field names say which is which. Oracle's migration
  inputs are `oldMetadata`/`newMetadata`, protojson captured from a real Oracle,
  because that corpus predates the fixtures and no recorder survives to refresh it;
  every other engine's are `oldSchema`/`newSchema`, schema text this package parses.
  Converting Oracle's to text would rewrite the input of all 34 cases — a
  metadata/DDL/metadata round trip through this package is lossless for 0 of its 68
  blobs — so the difference stays until the corpus can be recaptured.
- PostgreSQL registers neither `GenerateMigration` nor `GetDatabaseMetadata`, so it
  has no fixture for either. Its migration generation is covered instead by
  `metadata_migration_test.go`, 70 hand-written `schema.DiffMigration` assertions
  whose names carry the intent a byte-for-byte golden cannot. Do not convert those.
- YAML block scalars cannot carry tabs or trailing whitespace, and dropping to a
  quoted one-line scalar makes fixture SQL unreadable. Indent fixture SQL with spaces.
- A block scalar also cannot carry a *leading* newline: yaml.v3 writes the value
  without it and the next read comes back one line short. Trim it before comparing,
  as the PostgreSQL definition test does, or the golden can never match.
- Trailing whitespace in a golden has the same effect, and it reaches the golden
  from the fixture rather than the generator: a view body or PL/SQL source keeps
  whatever the DDL that created it had, and the generator reproduces it verbatim
  because reformatting a stored definition risks changing it. Strip trailing
  whitespace from fixture SQL and from `definition`/`body` fields in fixture
  metadata, not from the generator's output.
- yaml.v3's `is_printable` stops at 3-byte UTF-8, so any astral-plane character —
  an emoji in a comment, say — is escaped, which forces the whole scalar onto one
  quoted line. Three goldens are in that state and should stay: they are the
  deliberate 4-byte-UTF-8 cases, which is exactly the utf8/utf8mb4 distinction
  worth covering. Read those by running the test, not by reading the fixture.
- Generated DDL must be byte-stable before it can be pinned. Map iteration order
  is the usual culprit; walk maps in sorted key order in the generators.

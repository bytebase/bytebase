# Bytebase schema plugin

Additional guidance for `./backend/plugin/schema/`, the per-engine conversions
between database metadata and DDL. Follow `../../../AGENTS.md` first; this file
adds to it.

## What these packages test

Each engine package owns three conversions, and a test covers exactly one:

- `GetDatabaseMetadata` — schema text to metadata.
- `GetDatabaseDefinition` — metadata to DDL.
- `schema.GetDatabaseSchemaDiff` plus the package's `generateMigration`.

Do not call `driver.SyncDBSchema` here, and never use a live server's answer as
the expected value: whether our output matches what an engine stores is engine
conformance, and it belongs in omni. No test in these packages starts a
container.

A metadata-to-DDL test takes metadata as its input, not a schema text. Deriving
it by parsing caps the test at whatever the parser extracts, so any generator
branch nothing feeds goes untested.

## The record process

Fixtures live in `testdata/` as a YAML list of cases, one file per test, named
after the source file it covers. Each test carries the path and a `record`
constant:

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

- Commit the fixture and the restored `record = false` together. A `record = true`
  left in the tree overwrites the goldens on every run, so the test always passes.
- Read the fixture diff. A re-record captures current behavior, bugs included.
- Recording only rewrites the output field. Changing a fixture's input means
  editing the metadata blob, or re-seeding it.

## Fixture gotchas

- YAML block scalars cannot carry tabs, trailing whitespace, or a leading
  newline. A value with any of them is stored as one unreadable quoted line.
  Indent fixture SQL with spaces; trim a leading newline before comparing, as
  the PostgreSQL definition test does. Trailing whitespace usually arrives from
  a `definition`/`body` field, not the generator — strip it in the fixture.
- yaml.v3 escapes astral-plane characters, so a golden containing an emoji is
  always a quoted line. Two are, deliberately: they cover 4-byte UTF-8.
- Generated DDL must be byte-stable before it can be pinned. Walk maps in sorted
  key order in the generators.

## Two deviations

Oracle's migration inputs are `oldMetadata`/`newMetadata`, protojson captured
from a real Oracle, where every other engine's are `oldSchema`/`newSchema`
schema text. Converting them would rewrite all 34 cases: a metadata/DDL/metadata
round trip is lossless for 0 of its 68 blobs.

PostgreSQL registers neither `GenerateMigration` nor `GetDatabaseMetadata`, so it
has no fixture for either. Its migration generation is covered by
`metadata_migration_test.go`, whose hand-written assertions carry intent a golden
cannot. Do not convert those.

# Bytebase schema plugin

Additional guidance for `./backend/plugin/schema/`, the per-engine conversions
between database metadata and DDL. Follow `../../../AGENTS.md` first; this file
adds to it.

## What these packages test

Each engine package owns two conversions, and a test covers exactly one:

- `GetDatabaseMetadata` — schema text to metadata.
- `GetDatabaseDefinition` — metadata to DDL.

A metadata diff is not one of them. `schema.DiffMigration` dispatches to the
engine's registered metadata diff, and every engine that supports one runs on its
own omni catalog; Bytebase has no shared differ to test.

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
go test ./backend/plugin/schema/mssql/ -run '^TestGetDatabaseDefinition$' -count=1
```

- Commit the fixture and the restored `record = false` together. A `record = true`
  left in the tree overwrites the goldens on every run, so the test always passes.
- Read the fixture diff. A re-record captures current behavior, bugs included.
- Recording only rewrites the output field. Changing a fixture's input means
  editing the metadata blob, or re-seeding it.

## Fixture gotchas

- YAML block scalars cannot carry tabs, trailing whitespace, or a leading
  newline. A value with any of them is stored as one unreadable quoted line.
  Indent fixture SQL with spaces; trim a leading newline before comparing when
  the header opens with a blank line. Trailing whitespace usually arrives from
  a `definition`/`body` field, not the generator — strip it in the fixture.
- yaml.v3 escapes astral-plane characters, so a golden containing an emoji is
  always a quoted line. Two are, deliberately: they cover 4-byte UTF-8.
- Generated DDL must be byte-stable before it can be pinned. Walk maps in sorted
  key order in the generators.

## Deviations

Oracle, PostgreSQL (including CockroachDB), MySQL, OceanBase, and TiDB run
their metadata extraction, definition rendering, simulation, and metadata
migrations on their omni catalogs, so their golden YAML corpora live in
`omni/{oracle,pg,mysql,oceanbase,tidb}/catalog/testdata/` under omni's own
gates. Bytebase keeps focused seam tests in `backend/plugin/schema/<engine>/`
for registry wiring, backup/scratch filtering, `stripUnrenderedFacts` /
`SkipDump`, `legacy_normalize.go`, `omnicatalog` prechecks, and error-to-advice
mapping.

`mariadb/testdata/cutover_parity.yaml` records what the deleted Bytebase
generator wrote for MariaDB and OceanBase and why each group of cases moved.

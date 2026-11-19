# Adding a database object type to declarative schema

This is the current guide for adding a database object type (for example a
collation, a foreign server, or a range type) to the declarative schema path
used by rollouts. Bytebase no longer owns an SDL chunk extractor, a shared
metadata differ, or a SQL migration generator. The object model and the engine
catalogs live in omni; the packages under `backend/plugin/schema/` register the
catalog entry points and keep the engine-specific deviations.

## Where the pieces live

| Layer | Location | Responsibility |
|---|---|---|
| Metadata contract | `github.com/bytebase/omni/metadata` (`proto/omni/metadata/database.proto` in omni) | `DatabaseSchemaMetadata`, `SchemaMetadata`, and the nested object messages. Bytebase persists this as protojson and discards unknown fields, so a field that is not here is dropped before any engine sees it. |
| Engine catalog | `github.com/bytebase/omni/<engine>/catalog` | Loads SDL, SQL, or synced metadata; renders database and per-object definitions; diffs two catalogs and emits a migration plan. |
| Engine shim | `backend/plugin/schema/<engine>/` | Registers the catalog functions, filters or repairs metadata where Bytebase has a compatibility rule, and supplies engine-specific walk-throughs. |
| Registry and dispatch | `backend/plugin/schema/schema.go` | Maps an engine to its registered functions. The main entry points are `GetDatabaseDefinition`, `GetDatabaseMetadata`, `SDLMigration`, `DiffSDLMigration`, and `DiffMigration`. |
| MySQL-family metadata wrapper | `backend/plugin/schema/internal/omnicatalog/omnicatalog.go` | `Engine`, `DiffMetadata`, and `StripDatabaseContext` share the metadata-diff flow across MySQL, MariaDB, OceanBase, and TiDB. |

## 1. Add the object to omni first

1. Add the message and its field to omni's metadata proto, under
   `DatabaseSchemaMetadata` or `SchemaMetadata` as appropriate. Regenerate the
   Go protobuf and release omni, then bump `github.com/bytebase/omni` in this
   repo's `go.mod` and run `go mod tidy`.
2. Add the object to the engine's omni catalog: parse and install it during
   SDL/SQL/metadata load, render it in the catalog's database and per-object
   definitions, and include it in catalog diff and migration generation. The
   catalog tests that check this behavior against a real engine belong in omni,
   not here.
3. If the object can exist in metadata that the catalog cannot install as
   defined, expose that state in the catalog's load report. The Bytebase diff
   refuses a migration only when such an object changed, so it needs to be able
   to find the object in both snapshots.

## 2. Wire the engine shim

Each engine package registers its catalog entry points from `init()`. For an
object that is only part of the whole-database definition, there is usually no
new registration: the existing `RegisterGetDatabaseDefinition` function already
delegates to the catalog's `DatabaseDefinition`, which now renders the object.

The registrations used by the schema path are:

- `RegisterGetDatabaseDefinition` and `GetDatabaseDefinition` for dump and
  definition output.
- `RegisterGetDatabaseMetadata` and `GetDatabaseMetadata` for parsing schema
  text back into metadata.
- `RegisterDiffMetadataMigration` and `DiffMigration` for metadata-to-metadata
  migration, with the engine's catalog loaded through `LoadMetadata` or
  `LoadMetadataInto`.
- `RegisterDiffSDLMigration`, `DiffSDLMigration`, and `SDLMigration` for the
  declarative path. `SDLMigration` converts the current metadata to SDL with
  `MetadataToSDL`, then diffs it against the user's SDL.
- `RegisterSDLDropAdvices` for destructive-operation warnings that the engine
  emits from the migration plan.
- `RegisterGetMultiFileDatabaseDefinition` for engines that support a
  multi-file definition.
- `RegisterGetSchemaDefinition`, `RegisterGetTableDefinition`,
  `RegisterGetViewDefinition`, `RegisterGetMaterializedViewDefinition`,
  `RegisterGetFunctionDefinition`, `RegisterGetProcedureDefinition`, and
  `RegisterGetSequenceDefinition` for the per-object definition endpoints that
  already exist.

If the new object needs a new per-object definition endpoint, add a function
type, a registry map, and a `Register...`/`Get...` pair to
`backend/plugin/schema/schema.go`, then register the engine's implementation in
its `init()`. Do not add a Bytebase-side differ, chunk extractor, or generator;
those responsibilities are in the catalog.

For the MySQL family, the metadata diff also has to understand the object when
the catalog reports it as unloadable. Add the new kind to
`omnicatalog.snapshotObject` in `backend/plugin/schema/internal/omnicatalog/omnicatalog.go`.
PostgreSQL has its own `snapshotObjects` in
`backend/plugin/schema/pg/catalog_migration.go`.

If the object's DDL needs policy checks or destructive-operation advice, extend
the engine's `walk_through.go` or its `SDLDropAdvices` implementation. That is a
separate concern from making the object round-trip through metadata and DDL.

## 3. Test the dump, definition, and diff paths

Use the existing test files as the template:

- Definition output: `backend/plugin/schema/<engine>/get_database_definition_test.go`,
  with fixtures in `testdata/get_database_definition.yaml`. The input is
  metadata, not schema text, so the test covers generator branches the parser
  does not feed. The test re-parses or reloads the generated DDL into the
  catalog and pins the output with the `record` constant.
- Metadata extraction: `backend/plugin/schema/<engine>/get_database_metadata_test.go`,
  with fixtures in `testdata/get_database_metadata.yaml`. This pins what this
  package extracts; engine conformance belongs in omni.
- Metadata diff: `backend/plugin/schema/<engine>/metadata_migration_test.go`
  (for example `TestDiffMetadataEvent` for OceanBase) and the catalog's own
  tests in omni.
- SDL diff: `backend/plugin/schema/<engine>/sdl_migration_test.go`. Names such
  as `TestSDLMigration_CreateTable` and `TestMySQLSDLDropAdvices` show the
  expected create, alter, drop, and advice cases.
- Driver dump: when the driver delegates `Dump` to the catalog, cover it with
  `TestDumpRendersTheSyncedMetadata` in `backend/plugin/db/<engine>/dump_test.go`
  (see the OceanBase and TiDB tests).

Read [the schema package guidance](../backend/plugin/schema/AGENTS.md) before
recording fixtures. Keep `record = false` in the committed test, and do not start
a database container in `backend/plugin/schema/`; the schema tests use fixtures
and catalog round trips.

## Checklist

1. Add the metadata message and field in omni; bump the omni dependency here.
2. Teach the omni engine catalog to load, define, and diff the object.
3. Update the engine shim's `get_database_definition.go`,
   `get_database_metadata.go`, `metadata_migration.go` or `catalog_migration.go`,
   and `sdl_migration.go` only where the wiring or a documented deviation
   requires it.
4. Add the refusal lookup for an unloadable object if the catalog reports one.
5. Add definition, metadata, metadata-diff, and SDL-diff coverage; add a driver
   dump test when the driver owns that path.
6. Run the package tests and the repository gates from `AGENTS.md`.

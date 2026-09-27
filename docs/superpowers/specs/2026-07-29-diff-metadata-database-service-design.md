# DiffMetadata Move to DatabaseService + Request Reshape

**Date**: 2026-07-29
**Status**: Approved (Danny: move to DatabaseService; single metadata in the
request; new IAM permission granted to schema-change-authoring roles only;
clean cut — no compatibility alias; separate PR based on main, independent of
the QueryHistoryService extraction PR)
**Ships in**: 3.21 as a labeled breaking change

## Summary

Move `DiffMetadata` from `SQLService` to `DatabaseService` and reshape its
request: instead of the caller uploading both schemas, the request carries the
database resource `name` plus the **target** `DatabaseMetadata` only. The
server reads the source (current) schema from the store, which also derives
the engine — the old `engine` field disappears.

The reshape changes the security class of the RPC: the old form was a pure
function over caller-supplied data (`allow_without_credential = true`); the
new form reads stored schema contents, so it is IAM-gated with a **new
permission `bb.databases.diffMetadata`**.

The old `SQLService.DiffMetadata` is **removed outright** in the same release
(decision 2026-07-29, superseding this doc's first revision, which kept it as
a deprecated anonymous alias): the only known caller is our own schema
editor, and the request shape changed anyway, so the alias would preserve a
dead contract. Anonymous or external callers of the old method get
unimplemented/404 and must adopt the new RPC.

## Who calls it today

- Frontend: exactly one call site — the schema editor's `generateDiffDDL`
  (used by `SchemaEditorSheet`), which already holds the `Database` object and
  passes the pristine store-fetched metadata as source. Server-side source
  reading is semantically equivalent (and fresher).
- No bytebase-action, no Terraform, no other backend callers. The
  `backend/plugin/schema` "DiffMetadata" hits are the underlying differ
  registry, not the RPC.
- The old RPC was anonymous, so unknown external callers cannot be ruled out
  via auth logs; the clean cut accepts that they break at upgrade and is
  called out in release notes.

## Design

### 1. Proto

**`database_service.proto`** gains, next to `DiffSchema`:

- `rpc DiffMetadata(DiffMetadataRequest) returns (DiffMetadataResponse)`
- Binding: `post: "/v1/{name=instances/*/databases/*}:diffMetadata"` (mirrors
  `DiffSchema`)
- `permission = "bb.databases.diffMetadata"`, `auth_method = IAM` — the ACL
  interceptor resolves the database name to its project with no custom code
- `DiffMetadataRequest{ name, target_metadata }`,
  `DiffMetadataResponse{ diff }`

**`sql_service.proto`** loses the RPC and both messages entirely, along with
its `import "v1/database_service.proto"` — `DatabaseMetadata` was the only
database-service type it used, so the SQL execution service no longer depends
on database types at all.

### 2. IAM

- New permission `bb.databases.diffMetadata` in
  `backend/common/permission/permission.go` + `permission.yaml`
  (frontend copy synced via `copy_config_files.sh` → generated TS)
- Granted to the schema-change-authoring roles only: **workspace admin,
  workspace DBA, project owner, project developer**. Deliberately absent from
  project viewer, project releaser, and both SQL editor roles (decision
  2026-07-29): generating migration DDL is change authoring, not schema
  reading, even though the diff output itself reveals nothing beyond
  `bb.databases.getSchema`
- Custom roles must add the permission explicitly — called out in release
  notes

### 3. Backend

`DatabaseService.DiffMetadata` (new, in `database_service.go` next to
`DiffSchema`):

1. Parse `name` → instance + database; load instance (workspace-scoped)
2. Gate on the same engine set the old RPC accepted (MYSQL, POSTGRES, TIDB,
   ORACLE, MSSQL) using the instance's engine — `InvalidArgument` otherwise
3. Source = `store.GetDBSchema` (already a `model.DatabaseMetadata`; no
   conversion) — `NotFound` if the database has never been synced
4. Target = `convertV1DatabaseMetadata(target_metadata)` →
   `model.NewDatabaseMetadata(..., store.IsObjectCaseSensitive(instance))` —
   the old handler hardcoded case sensitivity to `true`; the new one uses the
   instance's actual collation behavior, matching `DiffSchema`
5. `schema.DiffMigration(engine, source, target)` → `{ diff }`

`SQLService.DiffMetadata` and its implementation are deleted; `sql_service.go`
drops its `plugin/schema` import with it.

### 4. Frontend

`generateDiffDDL` keeps its signature (database + source + target): the local
`isEqual(source, target)` no-op short-circuit and validation stay client-side;
only the wire call changes to
`databaseServiceClientConnect.diffMetadata({ name, targetMetadata })`.

The target must be complete: `SchemaEditorSheet` previously fetched its
baseline with `limit: 200` (a windowed-editor perf guard from #17514), which
was safe when both diff sides shared the truncation but would read as DROPs
for every omitted table against the full server-side source (caught in review,
PR #21068). The sheet now fetches unlimited metadata — matching every other
metadata consumer, which already defaults to no limit — and the request proto
documents the completeness requirement.

## Compatibility (3.21)

| Surface | Impact |
|---|---|
| gRPC/Connect `bytebase.v1.SQLService/DiffMetadata` | **Removed — breaking.** Callers must switch to `bytebase.v1.DatabaseService/DiffMetadata` with the new request shape |
| REST `POST /v1/schemaDesign:diffMetadata` | **Removed — breaking.** Replaced by `POST /v1/{name=instances/*/databases/*}:diffMetadata` (authenticated) |
| Anonymous access to schema diffing | Gone by design — the new RPC reads stored schemas and requires `bb.databases.diffMetadata` |
| Cached pre-3.21 frontend bundles during rolling upgrade | Schema editor DDL preview fails until refresh — accepted with the clean cut |
| Custom roles | Need `bb.databases.diffMetadata` to use the new RPC (four predefined roles updated in-release) |

Ships with `--label breaking` and a `## Breaking Changes` section covering
the method removal, the REST path change, and the new permission.

## Testing

- New e2e `TestDiffMetadata`: sync an empty database, diff against a
  one-table target → expect `CREATE TABLE` DDL; missing target →
  InvalidArgument; workspace member without a project role → PermissionDenied;
  **project viewer → still PermissionDenied** (pins the role boundary);
  project developer → succeeds with the same diff as the owner
- Existing `TestDiffMetadataPreservesSRIDInvisible` (converter + differ unit
  test) is unaffected
- Standard gates: buf, golangci-lint, go build, e2e, pnpm suite

## Alternatives considered

1. **Deprecated anonymous alias with the old two-metadata shape** —
   implemented first (with a `Legacy*` message rename to free the canonical
   names), then deliberately removed: the alias would carry a dead contract
   whose only known caller is our own frontend, and the anonymous surface is
   exactly what this change retires. Clean cut chosen (Danny, 2026-07-29).
2. **Fold into `DiffSchema` as a `target_metadata` oneof member** — tempting
   end-state (one diff surface per database), but a larger contract change
   than asked for and `DiffSchema` carries its own TODO ("secure it", still
   on `bb.databases.get`); revisit when that TODO is addressed.
3. **Reuse `bb.databases.getSchema` / grant the new permission to every
   getSchema role** — rejected: per-method permissions keep role design
   explicit, and DDL generation belongs to authoring roles, not read roles.

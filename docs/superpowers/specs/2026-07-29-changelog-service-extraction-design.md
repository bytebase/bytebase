# ChangelogService Extraction Design

**Date**: 2026-07-29
**Status**: Implemented
**Ships in**: 3.21 (current release 3.20.1) as a clean cut — no compatibility aliases

## Summary

Extract `ListChangelogs` and `GetChangelog` from `DatabaseService` into a new
standalone `ChangelogService`, following the established per-resource service
pattern (`RevisionService` is the exact structural analog). The old
`DatabaseService` method names are **removed in the same release**: an earlier
revision of this design kept them for one release as deprecated delegating
aliases, but after reviewing the blast radius (external gRPC/Connect callers
only — REST, Terraform, bytebase-action, IAM, and page tokens are all
unaffected) we chose the clean cut and deleted the aliases outright.

## Goals

- `ChangelogService` owns the changelog resource
  (`instances/{instance}/databases/{database}/changelogs/{changelog}`).
- Zero behavior change on the surviving surface: same permissions, same REST
  paths, same wire format, same pagination semantics, same errors.

## Non-goals

- No store-layer changes (`backend/store/changelog.go` untouched).
- No new RPCs (cross-database listing, export, etc. — future work that this
  service gives a home to).
- `DiffSchema` stays in `DatabaseService`: it accepts changelog names as diff
  targets but is a schema operation, not changelog CRUD.
- No permission or resource-name changes (`bb.changelogs.*`,
  `bytebase.com/DatabaseChangelog` stay as-is).
- No compatibility aliases (decision 2026-07-29, superseding the earlier
  one-release-alias plan in this doc's first revision).

## Background

- `DatabaseService` had 14 RPCs; changelogs were the last child-resource
  collection inside it. After extraction it holds the database resource plus
  schema projections only.
- The changelog messages (`Changelog`, `ChangelogView`, `ListChangelogsRequest`,
  `ListChangelogsResponse`, `GetChangelogRequest`) were referenced by no other
  proto file. Moving them between files does not change fully-qualified names
  (`bytebase.v1.*`), so binary/JSON wire format is unaffected.
- Authorization is annotation-driven: `backend/api/auth/auth.go` resolves the
  method descriptor from the invoked full method name via
  `protoregistry.GlobalFiles` and reads the `bytebase.v1.permission` extension,
  so the moved RPCs are enforced with no auth code changes.

## Design

### 1. Proto: new `proto/v1/v1/changelog_service.proto`

Modeled on `revision_service.proto`: `ChangelogService` with the two RPCs
(HTTP bindings, `method_signature`, `bytebase.v1.permission`, and
`auth_method = IAM` annotations carried over unchanged), plus the five
changelog messages moved verbatim (field numbers, comments, resource
annotations unchanged). `database_service.proto` loses the two RPCs and the
message definitions entirely.

### 2. Backend

- `backend/api/v1/changelog_service.go`: `ChangelogService` struct (embeds
  `v1connect.UnimplementedChangelogServiceHandler`, holds `store`),
  `NewChangelogService(store)`. The implementation moved wholesale from the
  old `database_service_changelog.go` (deleted): `ListChangelogs`,
  `GetChangelog`, `parseChangelogFilter`, and the `convertToChangelog*`
  helpers (converters became plain functions; unused ctx/receivers dropped).
- Registration in `backend/server/grpc_routes.go` mirrors `RevisionService`,
  four touchpoints: construct, Connect handler → `connectHandlers`, static
  reflector list, and `v1pb.RegisterChangelogServiceHandler` for the gateway —
  which is what serves the unchanged REST route.

### 3. Frontend

- `changelogServiceClientConnect` added in `frontend/src/api/index.ts`; the
  changelog store (`frontend/src/stores/app/changelog.ts`) uses it.
- Changelog type imports repointed from `proto-es/v1/database_service_pb` to
  `proto-es/v1/changelog_service_pb` across the store, `stores/app/types.ts`,
  `utils/v1/changelog.ts`, and the changelog route components/tests.

### 4. Generated artifacts, MCP, and docs

- `buf generate` regenerates `backend/generated-go`,
  `frontend/src/types/proto-es`, `proto/gen/grpc-doc`, and
  `backend/api/mcp/gen/openapi.yaml` (the MCP spec is a second
  connect-openapi plugin output in `proto/buf.gen.yaml`).
- The agent's API index generator
  (`frontend/scripts/generate_openapi_index.js`) now skips
  `deprecated: true` operations generically — kept even though the aliases are
  gone, so any future deprecation is automatically hidden from agents. The
  manually maintained `serviceDirectory` in
  `frontend/src/modules/agent/logic/prompt.ts` gained a ChangelogService
  entry.
- Resource-pattern table updated in
  `docs/plans/saas/07.api-resource-patterns.md`.

## Compatibility (3.21)

| Surface | Impact |
|---|---|
| REST `GET /v1/.../changelogs[/*]` | None — identical path, payload, auth; now served by ChangelogService's gateway registration |
| gRPC/Connect `bytebase.v1.DatabaseService/ListChangelogs`, `GetChangelog` | **Removed — breaking.** External gRPC/Connect callers get unimplemented; they must switch to `bytebase.v1.ChangelogService/*` |
| Terraform provider, bytebase-action | Unaffected (REST-only / no changelog usage, verified) |
| IAM (`bb.changelogs.list/get`), resource names, page tokens | Unchanged |
| Published OpenAPI operationIds | `DatabaseService_*` → `ChangelogService_*` — breaking for OpenAPI-generated SDKs; paths unchanged |
| Audit log `method` field | New service name going forward; historical rows keep the old name |

`buf breaking` (FILE rules) flags the message moves and RPC deletions. No CI
job enforces it; accepted as part of the labeled breaking change.

## Rollout

Single phase: ships in 3.21 with `--label breaking` and a
`## Breaking Changes` section covering the gRPC method-name removal and the
OpenAPI operationId rename. Release notes call out the migration
(`DatabaseService.ListChangelogs` → `ChangelogService.ListChangelogs`; REST
users need no changes).

## Testing

- E2E tests migrated to the new `changelogServiceClient`
  (`schema_update_test.go`, `sync_schema_test.go`, `action_test.go`,
  client wiring in `tests.go`), including a `GetChangelog` e2e assertion that
  did not previously exist.
- Frontend store tests mock the new client; full suite green.
- Standard gates: `buf lint`, `golangci-lint`, `go build`, targeted e2e,
  `pnpm fix/check/type-check/test`.

## Alternatives considered

1. **One-release deprecated aliases + delegation** — implemented first, then
   deliberately removed: the only protected population is external
   gRPC/Connect callers of two read-only methods, and the REST surface (which
   external integrations actually use) is unaffected. Not worth carrying the
   shim, the duplicate docs entries, and a second breaking PR in 3.22.
2. **Runtime route rewrite, no proto aliases** — rejected: per-protocol
   plumbing, invisible in generated docs/SDKs, more code than the thing it
   replaces.
3. **Clean cut (chosen)** — one labeled breaking release; smallest permanent
   surface.

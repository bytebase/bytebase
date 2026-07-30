# QueryHistoryService Extraction Design

**Date**: 2026-07-29
**Status**: Approved (amended: keep SQLService aliases as deprecated proxies)
**Ships in**: 3.21 (current release 3.20.1); legacy alias removal in a future
release, date TBD

## Summary

Extract `SearchQueryHistories`, `ListQueryHistories`, and `GetQueryHistory`
from `SQLService` into a new standalone `QueryHistoryService`, following the
per-resource service pattern (`ChangelogService` and `RevisionService` are the
structural analogs). Unlike the ChangelogService extraction's clean cut, the
old `bytebase.v1.SQLService` methods are **kept as deprecated delegating
proxies** for a safe upgrade transition, and removed in a future release
(date TBD).

Rationale for diverging from the changelog decision: the query history RPCs
are called continuously by the SQL Editor over the Connect protocol
(`/bytebase.v1.SQLService/SearchQueryHistories` etc.). During a rolling
upgrade, browsers holding a cached pre-3.21 frontend bundle keep issuing the
old method names against a new backend; without aliases the History pane
breaks until a hard refresh. Changelogs had no such hot browser-side caller.

After the move, `SQLService`'s non-deprecated surface holds only execution
and utility RPCs: `Query`, `AdminExecute`, `Export`, `DiffMetadata`,
`AICompletion`.

## Goals

- `QueryHistoryService` owns the query history resource
  (`projects/{project}/queryHistories/{id}`) and serves the REST routes.
- Zero behavior change: same permissions, same REST paths, same wire format,
  same pagination semantics, same caller-scoping and existence-hiding errors —
  on both the new methods and the deprecated aliases.

## Non-goals

- No store-layer changes (`backend/store/query_history.go` untouched).
- The write path stays where it is: `createQueryHistory` is an internal store
  write invoked by `Query`/`AdminExecute`/`Export` inside `SQLService`, not an
  RPC. `QueryHistoryService` is read-only, like `ChangelogService` (whose rows
  are written by task runs).
- No new RPCs, no permission changes (`bb.queryHistories.list` stays), no
  resource-name changes, no new `google.api.resource` annotation for
  QueryHistory (it has none today).
- No commitment to a specific alias-removal release in this design; removal is
  a future labeled-breaking change, date TBD.

## Background

- `SQLService` mixes two concerns: SQL execution (Query/AdminExecute/Export
  plus utilities) and the query history child resource of Project. The history
  RPCs are the only project-parented resource CRUD in the service.
- The six messages to move (`SearchQueryHistoriesRequest/Response`,
  `ListQueryHistoriesRequest/Response`, `GetQueryHistoryRequest`,
  `QueryHistory` with its `Type` enum) are referenced by no other proto file.
  Package stays `bytebase.v1`, so fully-qualified names and binary/JSON wire
  format are unchanged. The alias RPCs reference the moved messages via
  `import "v1/query_history_service.proto"` — same types, so alias and new
  method are wire-identical.
- Authorization is annotation-driven: `backend/api/auth/auth.go` resolves the
  method descriptor from the invoked full method name via
  `protoregistry.GlobalFiles` and reads the extensions. The aliases keep their
  `bytebase.v1.permission` / `auth_method` annotations, so both paths enforce
  identical auth — `ListQueryHistories` (IAM, `bb.queryHistories.list`) and
  the two CUSTOM caller-scoped RPCs — with no auth code changes.
- Recent hardening carries over verbatim on both paths: SearchQueryHistories
  rejects the AIP-159 wildcard parent (#21064); ListQueryHistories supports
  "projects/-" gated on workspace-level permission.
- `buf.yaml` lints with the BASIC category; `RPC_REQUEST_RESPONSE_UNIQUE`
  (DEFAULT category) is not enforced, so the alias RPCs may share
  request/response types with the new service.

## Design

### 1. Proto

**New `proto/v1/v1/query_history_service.proto`**, modeled on
`changelog_service.proto`: `QueryHistoryService` with the three RPCs — HTTP
bindings under `/v1/{parent=projects/*}/queryHistories`, `method_signature`,
`bytebase.v1.permission`, and `auth_method` annotations carried over
unchanged — plus the six messages moved verbatim (field numbers, comments,
resource references unchanged).

**`sql_service.proto`** keeps the three RPCs as aliases with:

- `option deprecated = true;` and a comment pointing at
  `QueryHistoryService` and the TBD removal;
- **no `google.api.http` bindings** — the REST routes are served solely by
  `QueryHistoryService`'s gateway registration, so the gateway mux never sees
  duplicate paths and the OpenAPI spec lists each REST operation once
  (operationIds become `QueryHistoryService_*`);
- `method_signature` dropped with the bindings (the
  `google/api/client.proto` import had no other users in this file);
- `bytebase.v1.permission` / `auth_method` annotations kept identical.

The six message definitions leave `sql_service.proto`; it imports
`v1/query_history_service.proto` for the alias signatures.

### 2. Backend

- `backend/api/v1/query_history_service.go`: `QueryHistoryService` struct
  (embeds `v1connect.UnimplementedQueryHistoryServiceHandler`, holds `store` —
  the only dependency the moved code uses), `NewQueryHistoryService(store)`.
  Moves wholesale from `sql_service.go`: `SearchQueryHistories`,
  `ListQueryHistories`, `GetQueryHistory`, `paginatedQueryHistories`,
  `resolveQueryHistoryParent`, plus `convertToV1QueryHistory` from
  `sql_service_converter.go` (stays a method — it needs the store for the
  creator lookup). `createQueryHistory` stays in `sql_service.go`.
- `SQLService` gains a `queryHistoryService *QueryHistoryService` field
  (constructed first, injected via `NewSQLService`) and keeps three one-line
  deprecated methods that delegate:
  `func (s *SQLService) SearchQueryHistories(ctx, req) { return s.queryHistoryService.SearchQueryHistories(ctx, req) }`
  — identical semantics by construction, no duplicated logic.
- Registration in `backend/server/grpc_routes.go` mirrors `ChangelogService`,
  four touchpoints: construct, Connect handler → `connectHandlers`, static
  reflector list, and `v1pb.RegisterQueryHistoryServiceHandler` for the
  gateway (sole server of the REST routes). The SQLService Connect handler
  continues to serve the alias method paths.

### 3. Frontend

- `queryHistoryServiceClientConnect` added in `frontend/src/api/index.ts`;
  the 3.21 bundle uses only the new service (aliases exist for *older* cached
  bundles, not for new code).
- Call sites repointed: `modules/sql-editor/store/queryHistory.ts`
  (search + get) and `components/WorkspaceSetupGuide.tsx` (search).
- Type imports repointed from `proto-es/v1/sql_service_pb` to
  `proto-es/v1/query_history_service_pb` across `utils/v1/queryHistory.ts`,
  the sql-editor store types/`HistoryPane`, and their tests.

### 4. Generated artifacts, MCP, and docs

- `buf generate` regenerates `backend/generated-go`,
  `frontend/src/types/proto-es`, `proto/gen/grpc-doc`, and
  `backend/api/mcp/gen/openapi.yaml`. The deprecated aliases carry
  `deprecated: true` in generated specs; `generate_openapi_index.js` already
  skips deprecated operations generically (built for exactly this case during
  the changelog work), so agents only see `QueryHistoryService`.
- The agent `serviceDirectory` in `frontend/src/modules/agent/logic/prompt.ts`
  gains a QueryHistoryService entry.
- Resource-pattern table in `docs/plans/saas/07.api-resource-patterns.md`
  updated: query histories move out of the "SQLService workspace-level
  utility" bucket (stale since #21064 anyway) into their own project-parented
  service entry.

## Compatibility (3.21)

| Surface | Impact |
|---|---|
| REST `POST /v1/projects/*/queryHistories:search`, `GET .../queryHistories[/*]` | None — identical paths, payloads, auth; served by QueryHistoryService's gateway registration |
| gRPC/Connect `bytebase.v1.SQLService/SearchQueryHistories`, `ListQueryHistories`, `GetQueryHistory` | **Kept — deprecated delegating proxies**, wire-identical behavior; removal in a future release, date TBD |
| Cached pre-3.21 frontend bundles during rolling upgrade | Keep working via the aliases (the motivating case) |
| Terraform provider, bytebase-action | Unaffected (REST-only / no query history usage) |
| IAM (`bb.queryHistories.list`), caller-scoping, resource names, page tokens | Unchanged |
| Published OpenAPI operationIds | `SQLService_*` → `QueryHistoryService_*` for the REST operations — SDK regen concern only; paths unchanged |
| Audit log | No change — none of the three RPCs carry `audit = true` |

Not a breaking release: nothing is removed in 3.21. `buf breaking` (FILE
rules) still flags the message moves between files and the dropped HTTP
options on the aliases; no CI job enforces it, and the wire surface is
compatible. The breaking label belongs to the future alias-removal release.

## Rollout

- **3.21**: ship extraction + aliases; release notes announce the
  deprecation and point gRPC/Connect callers at
  `bytebase.v1.QueryHistoryService` (REST users need no changes).
- **Future release (TBD)**: remove the aliases with `--label breaking`,
  mirroring the ChangelogService removal notes.

## Testing

- E2E: `backend/tests/query_history_test.go` migrates to a new
  `queryHistoryServiceClient` wired in `tests.go`; it already covers all three
  RPCs, IAM gating, wildcard-parent behavior, and existence hiding. Added
  alias assertions pin the delegation: the deprecated
  `sqlServiceClient.SearchQueryHistories` / `GetQueryHistory` /
  `ListQueryHistories` return the same results as the new client until the
  aliases are removed.
- Frontend: sql-editor store tests mock the new client; full suite green.
- Standard gates: `buf format/lint`, `golangci-lint`, `go build`, targeted
  e2e, `pnpm fix/check/type-check/test`.

## Alternatives considered

1. **Clean cut, no aliases (the ChangelogService choice)** — rejected here:
   unlike changelogs, query history has a hot browser-side Connect caller
   (the SQL Editor History pane), so removal without a transition window
   breaks cached bundles mid-upgrade. Decision 2026-07-29 (Danny).
2. **Aliases with duplicated HTTP bindings on both services** — rejected:
   duplicate gateway routes make the serving handler registration-order
   dependent and double every REST operation in generated specs.
3. **Runtime route rewrite instead of proto aliases** — rejected for the same
   reasons as in the changelog design: per-protocol plumbing, invisible in
   generated docs/SDKs, more code than the thing it replaces.
4. **Deprecated delegating proxies, REST on the new service only (chosen)** —
   smallest transition shim, wire-identical by construction, visible as
   deprecated in every generated artifact, and removable in one future PR.

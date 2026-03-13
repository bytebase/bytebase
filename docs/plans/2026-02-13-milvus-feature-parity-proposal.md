# Milvus Full-Parity Support Proposal

## Goal

Define the work needed to move Milvus support from current MVP wiring to production-grade Bytebase support with clear scope, milestones, and test gates.

## Current Status (as of 2026-02-13)

Milvus support in this branch currently includes:

- Engine enum and backend/frontend wiring.
- Read-only query path for limited SQL-like statements (`SHOW COLLECTIONS`, `DESCRIBE COLLECTION`, simple `SELECT` mapped to query API).
- Basic metadata sync (`SyncInstance`, `SyncDBSchema`) using collection list/describe APIs.
- Unit tests and Docker-backed integration tests for query/sync and ACL user ping.

Major gaps remain:

- `Execute` is unimplemented in driver.
- `Dump` is unimplemented.
- Parser/validator only supports a narrow read-only subset.
- No vector-aware migration workflow parity (review/deploy/rollback policies) yet.
- No first-class mapping for vector-specific metadata (index params, metric type, load/release state, aliases, partitions) in schema model.

## Does Bytebase Support Vector DB Today?

Not as full parity.

- Upstream Bytebase documentation lists supported engines but does not currently list Milvus as a fully supported production engine.
- This branch adds early Milvus capability, but it is not yet equivalent to mature relational engines in Bytebase.

## What Must Differ for Vector DB vs Row DB

Milvus is API-first and vector-search oriented. Bytebase must adapt in these areas:

1. Change model
- Row DB: SQL DDL migrations are the core unit.
- Milvus: collection/index/partition/load/release/user/role operations are API operations.

2. Query model
- Row DB: broad SQL semantics.
- Milvus: vector search, hybrid search, rerank, and expression filter query APIs.

3. Schema model
- Row DB tables/columns/indexes are insufficient by themselves.
- Need vector-specific fields: embedding dimension, vector field type, metric type, index algorithm params, load state, aliases, partitions.

4. Governance model
- Existing SQL advisors/checks do not cover vector-specific risks (dimension mismatch, metric/index incompatibility, destructive load/index/collection ops).

## Milvus Capability Mapping Needed for Parity

Based on Milvus v2 API docs, Bytebase parity should cover:

- Database APIs (list/create/drop/use).
- Collection APIs (list/create/describe/rename/drop/load/release/stats/has/get-load-state/refresh-load).
- Partition APIs (list/create/drop/load/release/stats/has).
- Index APIs (create/list/describe/drop/alter/rebuild progress).
- Entity APIs (insert/upsert/delete/get/query/search/hybrid-search).
- Alias APIs (create/list/describe/alter/drop).
- RBAC APIs (users/roles/privileges + grant/revoke flows).
- Import job APIs and long-running operation tracking.

## Proposed Architecture Changes

1. Driver layer (`backend/plugin/db/milvus`)
- Implement `Execute` as a typed operation router (not generic SQL executor):
  - Collection/partition/index/alias lifecycle operations.
  - User/role grant/revoke operations.
  - Controlled data mutation (`insert/upsert/delete`) for admin flows where allowed.
- Extend query path:
  - Add `SEARCH` and `HYBRID SEARCH` grammar translation to Milvus entity APIs.
  - Add pagination, output field controls, consistency level options.
- Implement `Dump` for logical schema export:
  - Export collection definitions, fields, index definitions, aliases, partitions, and RBAC metadata (optionally redacted).

2. Parser layer (`backend/plugin/parser/milvus`)
- Add Milvus-specific statement grammar/AST for:
  - `CREATE/DROP/ALTER COLLECTION`
  - `CREATE/DROP INDEX`
  - `CREATE/DROP PARTITION`
  - `LOAD/RELEASE COLLECTION|PARTITION`
  - `CREATE USER/ROLE`, grant/revoke privileges
  - `SEARCH` / `HYBRID SEARCH`
- Splitter must handle JSON payload blocks reliably for API-like statements.
- Validation modes:
  - Read-only query mode.
  - Migration-safe mode.
  - Admin-only mode.

3. Metadata model changes
- Extend internal metadata conversion so Milvus schema sync retains:
  - Vector field params (dimension, data type).
  - Index type + metric + index params.
  - Collection properties and consistency level.
  - Partition metadata.
  - Alias mapping.
  - Load state and status fields.

4. Product workflow integration
- Migration/review/deployment:
  - Introduce “Milvus operation plans” as first-class migration artifacts.
  - Add dry-run simulation on target instance APIs.
  - Add rollback strategy where possible (e.g. drop newly created resources, restore aliases).
- Risk checks:
  - Dimension changes blocking.
  - Unsafe destructive operations policy gates.
  - Index rebuild impact advisory.

5. Security/governance integration
- RBAC sync into instance metadata for review and audit.
- Audit logging for API operations with operation type and payload summary.
- Masking/classification strategy for non-vector scalar fields and metadata payloads.

## Implementation Phases with Guardrails

Each phase is mergeable only if all guardrails pass. If any required guardrail fails, the phase is not complete.

### Phase 0: Baseline Freeze and Guardrails Bootstrap

Deliverables:
- Freeze current Milvus MVP behavior with explicit contract tests.
- Introduce phase-gate scripts/targets for repeatable local and CI execution.
- Single local gate command:
  - `./scripts/test_milvus_phase0_gate.sh`

Required guardrails:
- Contract tests:
  - `backend/api/v1/milvus_wiring_contract_test.go`
  - `backend/api/v1/common_engine_mapping_test.go`
  - `backend/server/ultimate_registration_test.go`
- Milvus unit and integration baseline:
  - `go test -count=1 github.com/bytebase/bytebase/backend/plugin/db/milvus github.com/bytebase/bytebase/backend/plugin/parser/milvus`
- Non-Milvus regression sentinel:
  - `go test -count=1 github.com/bytebase/bytebase/backend/plugin/db/mongodb`
  - `go test -count=1 github.com/bytebase/bytebase/backend/plugin/db/elasticsearch`

Exit criteria:
- Baseline guardrails run green in CI and locally from one command.

### Phase 1: Operational Completeness (Execute + Lifecycle APIs)

Deliverables:
- Implement `Execute` operation routing for Milvus lifecycle ops:
  - collection/partition/index/alias create/drop/alter/load/release.
  - user/role grant/revoke and selected data mutation ops.
- Async operation handling with retry/backoff + timeout semantics.

Required guardrails:
- Unit tests:
  - request mapping per operation type.
  - error mapping (HTTP, Milvus code, timeout, partial failure).
  - idempotent retries and cancellation behavior.
- Integration tests (Docker Milvus):
  - lifecycle round-trips for collection/partition/index/alias.
  - RBAC grant/revoke and permission checks.
  - failure-path checks (non-existent object, duplicated creation, unsupported transitions).
- No E2E break checks:
  - current backend test workflow green (`backend-tests.yml`).
  - Milvus CI job green (`milvus-integration-tests`).

Exit criteria:
- `Execute` is implemented and operation coverage matrix passes in integration tests.

### Phase 2: Query Completeness (Search + Hybrid Search)

Deliverables:
- Add parser and driver support for:
  - `SEARCH`
  - `HYBRID SEARCH`
  - query options: topK, metric/search params, rerank.
- Return distance/score in result schema consistently.

Required guardrails:
- Parser tests:
  - syntax acceptance/rejection matrix for query/search/hybrid-search statements.
  - statement splitting with JSON-like parameter blocks.
- Driver tests:
  - payload conversion and response normalization for score/distance fields.
  - limit enforcement and unsafe query rejection.
- Integration tests:
  - seeded vector data, deterministic search results, hybrid query behavior.
  - latency and timeout behavior under bounded dataset.
- No E2E break checks:
  - all Phase 1 checks plus frontend type-check/tests for Milvus query UI wiring.

Exit criteria:
- search and hybrid-search are usable end-to-end in query editor with deterministic integration coverage.

### Phase 3: Metadata and Dump Completeness

Deliverables:
- Extend sync metadata model to preserve Milvus-specific schema:
  - vector field params, index params/metric, partitions, aliases, load state.
- Implement `Dump` for logical schema export (collection/index/partition/alias and optional RBAC snapshot).

Required guardrails:
- Unit tests:
  - metadata conversion correctness for every Milvus schema facet.
  - dump deterministic output formatting and redaction behavior.
- Integration tests:
  - compare synced metadata with live Milvus API describe/list outputs.
  - dump/import replay smoke verification (where feasible).
- No E2E break checks:
  - schema sync and schema review flows for existing engines unchanged (targeted sentinel suites).

Exit criteria:
- sync and dump output contain vector-specific metadata with stable deterministic tests.

### Phase 4: Governance Parity

Deliverables:
- Milvus-native migration plan type for review/deploy (not forced relational SQL semantics).
- Vector-specific advisor/risk checks (dimension/metric/index/destructive-operation policies).
- Precheck permissions validation and RBAC sync in review flows.

Required guardrails:
- Policy/advisor unit tests:
  - explicit allow/deny matrices for destructive and incompatible operations.
- Workflow integration tests:
  - validate-only, review approval path, deployment path, and rollback/fallback path.
- No E2E break checks:
  - existing migration flows for MySQL/Postgres/MongoDB remain green.

Exit criteria:
- governance checks actively enforce Milvus safety rules without regressing existing engine flows.

### Phase 5: Production Hardening and Multi-Version Support

Deliverables:
- CI matrix for supported Milvus versions.
- Reliability hardening for async operations and transient failures.
- Operational runbooks and user docs.

Required guardrails:
- Multi-version integration matrix pass rate threshold (e.g. 100% required on protected branches).
- Flake budget: zero known flaky tests on protected branch.
- Performance guardrails:
  - sync and query-path budget checks on fixed dataset.

Exit criteria:
- sustained stable CI across Milvus versions with documented support policy.

## Phase Gate Template (apply to every PR in a phase)

- Scope guard:
  - PR only contains files relevant to current phase deliverables.
- Test guard:
  - required unit + integration + contract suites listed in phase are green.
- Regression guard:
  - non-Milvus sentinel suites are green.
- Observability guard:
  - errors include actionable operation context and Milvus error code/message.
- Rollback guard:
  - destructive operations have clear compensation or explicit non-rollback documentation.

## Required Test Gates

1. Unit
- Parser: split/parse/validate for all supported Milvus statements.
- Driver: request mapping and response conversion for all operation classes.

2. Integration (Docker Milvus)
- Lifecycle: create/list/describe/load/release/drop for collection/partition/index.
- Data: insert/upsert/delete/query/search/hybrid-search.
- Security: user/role/privilege grant/revoke and auth checks.
- Sync: assert metadata fidelity for fields/indexes/partitions/aliases/load state.

3. Contract
- Enum/mapping/registration wiring tests.
- API compatibility tests for engine conversions and frontend capability gates.

## Risks and Mitigations

1. Milvus API evolution risk
- Mitigation: adapter layer with version-aware endpoint handling + version CI matrix.

2. Long-running async operations
- Mitigation: operation polling abstraction with timeout/retry and idempotency checks.

3. Non-SQL migration mismatch
- Mitigation: introduce Milvus-native operation plan model instead of forcing relational SQL semantics.

## Deliverables

- Milvus operation parser + executor.
- Enhanced metadata sync model.
- Governance/risk checks for vector operations.
- CI workflow job for Milvus integration tests.
- User-facing docs for supported Milvus operations and known limitations.

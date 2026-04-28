# Milvus Support Test Plan

## Goal

Add Milvus support with clear pre-merge test gates so we do not ship partially wired engine support to production.

This plan focuses on **testability first**: fail fast on missing enum wiring, keep driver behavior isolated by unit tests, then add integration coverage once the basic plugin exists.

## Reference Engine Commits in This Repo

Use these as implementation and testing baselines:

- `d924f98715` `feat: sync cassandra (#15688)`
  - Added proto enum + backend driver + sync + frontend engine wiring.
- `935c2105ba` `feat(cosmosdb): support cosmosdb connection and sync (#14813)`
  - Added new NoSQL engine end-to-end wiring.
- `14f9df394b` `chore: update common.proto with trino (#15654)`
  - Shows initial enum wiring step before full capability rollout.

## What Existing Engines Test Today

- Driver unit tests for complex conversion/logic:
  - `backend/plugin/db/cassandra/cassandra_test.go`
  - `backend/plugin/db/elasticsearch/elasticsearch_test.go`
- Parser unit tests:
  - `backend/plugin/parser/cassandra/cassandra_test.go`
  - `backend/plugin/parser/standard/query_test.go`
- Cross-layer API logic tests (filtering/conversion patterns):
  - `backend/api/v1/database_service_test.go`

Milvus should follow this pattern: start with deterministic unit tests, then add integration tests when the runtime dependency is stable.

## Required Pre-Merge Gates for Milvus

1. Enum wiring completeness
- `proto/store/store/common.proto` includes `MILVUS`.
- `proto/v1/v1/common.proto` includes `MILVUS`.
- `backend/api/v1/common.go` maps both directions.
- Guardrail test must pass:
  - `backend/api/v1/common_engine_mapping_test.go`

2. Driver plugin minimum behavior
- New package `backend/plugin/db/milvus`.
- Tests for:
  - Connection config building and authentication options.
  - Ping behavior and error translation.
  - Query/execute error handling boundaries.
  - Sync fallback behavior (if metadata sync is minimal at MVP stage).

3. Parser/query validation behavior
- If SQL-like editor/query is supported:
  - splitter/validator/query-span tests in `backend/plugin/parser/milvus`.
- If non-SQL API mode:
  - explicit tests that SQL editor restrictions and request validation are correct.

4. Frontend engine wiring checks
- Engine listed in:
  - `frontend/src/utils/v1/instance.ts` (`supportedEngineV1List`, `engineNameV1`, feature gates).
  - `frontend/src/components/InstanceForm/constants.ts` (default port/icon).
- Type-check and frontend tests must pass after enum regen.

5. Runtime registration check
- `backend/server/ultimate.go` imports Milvus driver/parser packages.
- Build succeeds in non-minidemo mode.

## Suggested Test Matrix for Milvus

1. API and enum wiring
- Create instance with `engine=MILVUS` and `validate_only=true` reaches driver Open/Ping path.
- Round-trip conversion `store <-> v1` for Milvus.

2. Connection/auth variants
- Basic username/password.
- TLS on/off and certificate verification behavior.
- Invalid host/port and timeout behavior.

3. Query path
- Allowed query command shape passes validation.
- Disallowed mutating command in readonly context fails with clear error.
- Result conversion preserves core scalar and structured payload types.

4. Schema sync path
- `SyncInstance` returns version and collection/database list.
- `SyncDBSchema` handles empty and populated metadata safely.
- System/internal namespaces are filtered if needed.

5. Failure-path regression
- Unknown/unsupported operation returns deterministic error.
- No panic with empty optional fields.

## Command Checklist

Run at minimum before merge:

```bash
go test ./backend/api/v1 -run Engine
go test ./backend/plugin/db/... ./backend/plugin/parser/...
go test ./backend/runner/...
pnpm --dir frontend type-check
```

After proto edits:

```bash
buf lint proto
cd proto && buf generate
```

## Rollout Strategy

1. Phase 1 (safe skeleton): enum + conversion + registration + connection test path.
2. Phase 2 (query path): validator/splitter/query execution + unit tests.
3. Phase 3 (sync path): metadata sync and schema persistence coverage.
4. Phase 4 (hardening): integration test with a real Milvus instance (containerized in CI if stable).


# Retire the Export-Data Workflow

- Status: design for review — decision: full purge
- Date: 2026-07-31
- Target: main (3.22 dev cycle) — not `release/3.21.0`

## Background

The export center was deprecated in #20693. The SQL Editor "Request Export"
button now opens the JIT access-grant drawer, and JIT-approved exports stream
synchronously through `SQLService.Export` — they never touch `export_archive`.
The plan UI blocks export plans with "Data export issue creation is no longer
supported".

The machinery is still fully wired, reachable only via direct v1 API calls:

- `PlanService.CreatePlan` accepts `export_data_config` specs
  (`backend/api/v1/plan_service.go:650`); rollout creation turns them into
  `DATABASE_EXPORT` tasks (`backend/api/v1/rollout_service_task.go:300`).
- The executor is registered (`backend/server/server.go:222`) and writes
  `export_archive` rows (`backend/runner/taskrun/data_export_executor.go`).
- `SQLService.Export` with a rollout name downloads the archive
  (`doExportFromIssue`, `backend/api/v1/sql_service.go:1038`).
- The data cleaner purges archives older than 24h, so the table is in practice
  always near-empty.

## Decision: full purge

Delete legacy `DATABASE_EXPORT` issues and their plans/tasks/runs in a
migration, drop `export_archive`, and remove every proto value — no
deprecated read surface, no legacy UI.

Considered alternatives: tombstone read-only (keep ~40 deprecated proto lines
so history stays legible) and block-creation-only. Rejected in favor of a
clean end-state: export-center usage was near zero (already deprecated), and
the durable audit trail lives in `audit_log` — issue/plan rows are redundant
with it for compliance purposes.

What is irreversibly lost: the issue-level records of past export requests
(title, approval chain rendering, statement sheet linkage). Note that the
legacy executor wrote no `query_history` rows — the only
`QueryHistoryTypeExport` writer is the synchronous database-target
`SQLService.Export` path (`sql_service.go:1022`) — so workflow exports have
no per-query history row to fall back on; this purge accepts that. What
remains: `audit_log` rows for the create/approve RPCs and for archive
downloads (`SQLService.Export` is audited), plus `query_history` rows for
synchronous SQL Editor / JIT exports, which are unaffected.

## Migration (`backend/migrator/migration/3.22/xxxx##retire_export_data.sql`)

`issue.type` and `task.type` are text columns. Plan specs are protojson JSONB
(camelCase keys — the spec key is `exportDataConfig`). All tables involved use
composite PKs `(project, id)`; every join below carries the project column.

Export-plan set: plans referenced by a `DATABASE_EXPORT` issue, UNION plans
whose `config->'specs'` contains an `exportDataConfig` key (covers API-created
plans that never got an issue).

Delete order (children before parents, per the FK graph):

```sql
-- 1. task_run_log → task_run → task(type DATABASE_EXPORT)
DELETE FROM task_run_log l USING task_run r, task t
  WHERE l.project = r.project AND l.task_run_id = r.id
    AND r.project = t.project AND r.task_id = t.id
    AND t.type = 'DATABASE_EXPORT';
-- 2. task_run
DELETE FROM task_run r USING task t
  WHERE r.project = t.project AND r.task_id = t.id
    AND t.type = 'DATABASE_EXPORT';
-- 3. task
DELETE FROM task WHERE type = 'DATABASE_EXPORT';
-- 4-8. with export_plan AS (<set above>):
--   plan_check_run, issue_comment, issue, plan_webhook_delivery, plan.
--   issue_comment and issue are deleted for every issue attached to an
--   export plan — (project, plan_id) IN export_plan OR type =
--   'DATABASE_EXPORT' — NOT only for issues typed DATABASE_EXPORT.
-- 9. DROP TABLE export_archive;
```

The issue predicate matters: `CreateIssue` only rejects export specs for
non-draft issues (`backend/api/v1/issue_service.go:617`), so a draft
`DATABASE_CHANGE` issue can legally point at an export plan. Deleting issues
by type alone would leave such a row behind and the subsequent plan delete
would abort the upgrade on the `issue(project, plan_id) → plan(project, id)`
FK. (The 3.21.1 draft backfill excluded `exportDataConfig` plans, so it did
not mass-create these; the API path still can.)

Also: update `LATEST.sql` (drop the `export_archive` table, update the
`issue.type` comment), and bump `TestLatestVersion` in `migrator_test.go`.

Notes:

- The whole file runs in one transaction; DML + `DROP TABLE` is fine in PG.
- The `jsonb_array_elements` scan over `plan` is a one-time full-table scan;
  acceptable at migration time.
- No new store methods → no new `TestCollision_*` obligations.
- Historical migration files (3.4, 3.17) that mention `export_archive` are
  immutable and stay untouched.

## Proto removal (all tags `reserved`, number and name)

| File | Change |
| --- | --- |
| `store/export_archive.proto` | delete file (+ delete stale generated `export_archive*.pb.go`; `buf generate` does not remove them) |
| `store/issue.proto` | remove `Issue.Type.DATABASE_EXPORT = 3` |
| `store/task.proto` | remove `Task.Type.DATABASE_EXPORT = 3` |
| `store/plan.proto` | remove `Spec.export_data_config = 7` oneof arm + `ExportDataConfig` message |
| `store/task_run.proto` | remove `export_archive_id = 9` |
| `v1/issue_service.proto` | remove `Issue.Type.DATABASE_EXPORT = 3` |
| `v1/plan_service.proto` | remove `Spec.export_data_config = 4` + `ExportDataConfig` message; fix spec_type filter doc (`:139`) |
| `v1/rollout_service.proto` | remove `Task.Type.DATABASE_EXPORT = 4`, `database_data_export = 9` + `DatabaseDataExport` message, `ExportArchiveStatus` enum + field 10; fix task_type filter doc (`:229`) |
| `v1/sql_service.proto` | remove the `Export` rollout REST bindings — `additional_bindings` for `/v1/{name=projects/*/plans/*/rollout}:export` and `.../rollout/stages/*}:export` (`:76-83`) — and the two rollout formats from the `ExportRequest.name` doc (`:408-409`), so generated OpenAPI/clients stop advertising the retired route |

Then `buf format -w proto && buf lint proto && cd proto && buf generate`
(regenerates backend `generated-go` and frontend `types/proto-es`). If CI runs
`buf breaking`, this PR is intentionally breaking — handle per the pre-PR
breaking-change review.

## Backend removal

Compile-driven after proto regen; the enum removals surface every remaining
reference.

- Delete `runner/taskrun/data_export_executor.go`; unregister in
  `server.go:222`
- Delete `store/export_archive.go`; remove `DeleteExpiredExportArchivesAll`
  (`store/runner_queries.go`) and `cleanupExportArchives` + retention const
  (`runner/cleaner/data_cleaner.go`)
- `api/v1/sql_service.go`: remove `doExportFromIssue` and the rollout-name
  branch of `Export` (RPC then accepts only database targets); remove the zip
  re-encrypt helpers if unused elsewhere (`doEncrypt` is shared with the
  synchronous path — verify before removing). In the same step, change
  `prepareRelatedMessage` to wrap `GetInstanceDatabaseID` parse failures as
  `CodeInvalidArgument` instead of `CodeInternal` (`sql_service.go:2014`):
  once the rollout branch is gone, retired-route names fall through to the
  database path, and a malformed name must surface as `InvalidArgument`, not
  a 500 — this is also what the rejection test below asserts
- `api/v1/rollout_service_task.go`: remove `getTaskCreatesFromExportDataConfig`
  + its spec-switch case
- `api/v1/plan_service.go`: `CreatePlan`/`UpdatePlan` reject
  `export_data_config` specs with an explicit error; remove both converter
  directions
- `api/v1/issue_service.go`: `CreateIssue` rejects `DATABASE_EXPORT`;
  remove converter cases (`issue_service_converter.go`)
- `api/v1/issue_hook.go`: remove the `Issue_DATABASE_EXPORT` branches from
  approval finding
- `api/v1/rollout_service.go:289,1330`: remove creator-only export special
  cases
- `api/v1/rollout_service_converter.go`: remove `ExportArchiveStatus`
  population, `convertToTaskFromDatabaseDataExport`, task-type mapping
- `component/review/evaluator.go`: remove `Issue_DATABASE_EXPORT` branches
  (`:335,1053`); KEEP `buildCELVariablesForDataExport` and the JIT
  access-grant path (`:864` comment documents the split)
- `runner/taskrun/running_scheduler.go:296`,
  `runner/taskrun/database_migrate_executor.go:1042`: prune from
  `exhaustive:enforce` switches
- Tests: prune export cases in `plan_service_test.go`,
  `issue_service_test.go`; add rejection tests (below)

## Frontend removal

Type-check-driven after proto regen.

- Delete `IssueDetailDatabaseExportView.tsx` (~800 lines) and its route wiring
  (`useIssueDetailType.ts`)
- `actionRegistry.ts`: `exportArchiveReady`, `computeExportArchiveReady`,
  download action; `IssueDetailActionBar.tsx`: download/expiry UI
- Export gating in `IssueDetailTaskRolloutActionPanel`,
  `PlanDetailTaskRolloutActionPanel`, `DeployTaskToolbar`
  (`data-export-creator-only`)
- `advanceState.ts` export blocker (backend rejection replaces it)
- `IssueTable.tsx:537`: remove the `DATABASE_EXPORT` filter option
- Prune export branches across `utils/v1/issue/*`, `lib/plan/*`,
  `plan-detail/utils/*` (`createPlan.ts` `targetsForSpec`, `changeReference`,
  `phaseSummary`, `rolloutPreview`, `diffPlanSpecs`, `check`, `workflow`),
  `useRedirects.ts`, `useIssueDetailSpecValidation.ts`,
  `PlanDetailStatementSection.tsx`, `IssueDetailStatementSection.tsx`,
  `types/v1/issue/issue.ts`, `issueDetailRedirect.test.ts`
- Locales: remove `issue.data-export.*`, `task.data-export-creator-only`, and
  any keys that become unused — across all locale files. KEEP `export-data.*`
  keys used by `DataExportButton` and `sql-editor.request-export` (JIT)

## Explicitly kept (shared with live features)

- JIT access-grant flow: `RequestExportButton`, `AccessGrantRequestDrawer`,
  `bb.accessGrants.*`
- Synchronous SQL Editor export: `DataExportButton`, `SQLService.Export` with
  database targets, `doExport`, export masking/permissions
- Risk source `DATA_EXPORT` / CEL namespace `request.data_export` and
  `buildCELVariablesForDataExport` (used by JIT grants and risk rules)
- `query_history` export rows; `audit_log` — untouched
- Historical migration files referencing `export_archive`

## Compatibility

- Breaking v1 API change: `CreatePlan`/`CreateIssue` reject export specs;
  `Issue.Type.DATABASE_EXPORT`, `Task.Type.DATABASE_EXPORT`,
  `Task.DatabaseDataExport`, `TaskRun.export_archive_status`, and
  `Plan.ExportDataConfig` leave the wire. Goes through the pre-PR
  breaking-change review.
- Terraform provider does not manage plans/issues; `bytebase-action` creates
  migration rollouts only — both unaffected.
- In-flight export issues at upgrade time are deleted with the rest; there is
  no cancel-and-keep state under full purge.
- Downgrade: none (standard for migrations).

## Testing & verification

1. Migrator: `TestLatestVersion`; migration applies on a DB seeded with a DONE
   export issue (plan + task + task_run + comment + archive row) and leaves no
   orphans — verify FK integrity and that unrelated issues/plans survive.
2. API rejection tests: `CreatePlan` with `export_data_config` →
   InvalidArgument; `CreateIssue` type `DATABASE_EXPORT` → InvalidArgument;
   `SQLService.Export` with a rollout name → InvalidArgument.
3. Full gates: `gofmt`, `golangci-lint` (repeat until clean), backend build,
   `buf format/lint/generate`, `pnpm fix/check/type-check/test`.
4. Residual grep: `ExportArchive|DATABASE_EXPORT|exportDataConfig|DatabaseDataExport`
   returns nothing outside historical migrations and the kept JIT/risk
   surfaces.

## Implementation order

1. Branch off `origin/main` (currently on `release/3.21.0`)
2. Migration + `LATEST.sql` + migrator test
3. Proto edits + regen (backend and frontend generated code)
4. Backend removal, compile-driven; backend tests
5. Frontend removal, type-check-driven; frontend tests
6. Verification gates + residual grep; PR with breaking-change checklist

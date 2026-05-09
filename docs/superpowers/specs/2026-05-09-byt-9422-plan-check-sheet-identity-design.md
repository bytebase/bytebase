# BYT-9422 Plan Check Sheet Identity Design

## Problem

Approval rule evaluation for database-change issues can become spec-order dependent when multiple `ChangeDatabaseConfig` specs target the same database.

Plan check target derivation keeps the SQL sheet identity in `CheckTarget.SheetSha256`, but plan-check results persist only the database target and check type. When approval generation later reads `STATEMENT_SUMMARY_REPORT` results, it can key them only by `(instance_id, database_name)`. Multiple specs for the same database overwrite each other, so only one spec's statement summary contributes `resource.table_name`, `statement.affected_rows`, `statement.table_rows`, and `statement.sql_type` to CEL rule evaluation.

This can silently route an issue through a weaker approval rule when the protected spec is not the surviving summary.

## Goals

- Persist enough identity on each plan-check result to match it back to the spec sheet that produced it.
- Preserve per-spec approval evaluation semantics.
- Avoid guessing when existing legacy plan-check results do not contain sheet identity.
- Keep the change scoped to plan-check result identity and approval CEL variable generation.

## Non-Goals

- Do not aggregate statement-summary data across all specs for a database.
- Do not introduce a new persisted plan spec identifier.
- Do not change approval rule matching order or CEL expression behavior.
- Do not add compatibility fallback that maps an ambiguous legacy summary to multiple specs.

## Data Model

Add `sheet_sha256` to `PlanCheckRunResult.Result` in `proto/store/store/plan_check_run.proto`.

`CheckTarget` already carries `SheetSha256`, so `CombinedExecutor.RunForTarget` will copy it to each result while tagging `target` and `type`.

Each plan-check result will then be identifiable by:

```text
target + type + sheet_sha256
```

For non-sheet checks or future checks where sheet identity is irrelevant, the field can remain empty.

## Approval Evaluation

`buildCELVariablesForDatabaseChange` will continue unfolding plan specs into approval targets. For each sheet-backed target, statement-summary lookup will use:

```text
instance_id + database_name + sheet_sha256
```

The matched summary will enrich only that target's CEL variables:

- `statement.text`: the target sheet's SQL.
- `statement.sql_type`: the target sheet's statement types.
- `resource.table_name`: the target sheet's changed tables.
- `statement.affected_rows`: the target sheet's affected rows.
- `statement.table_rows`: the target sheet's table row total.

`expandCELVars` and rule matching stay unchanged. The first approval rule that matches any CEL variable map still wins. The behavioral change is that earlier specs targeting the same database are no longer hidden by later statement-summary results.

## Legacy Plan Check Results

Existing stored plan-check results will have an empty `sheet_sha256`. Approval generation must not silently use those results for sheet-backed database-change specs, because that would preserve the ambiguous behavior BYT-9422 fixes.

When approval generation sees a `DONE` plan-check run for a database-change plan and finds statement-summary results for sheet-backed targets without `sheet_sha256`, it will treat the run as stale. The approval runner will recreate the plan-check run, tickle the plan-check scheduler, and return `done=false`.

The normal async flow then resumes:

```text
approval runner detects legacy result
-> creates a fresh AVAILABLE plan_check_run
-> scheduler runs checks and persists sheet_sha256
-> scheduler triggers approval check
-> approval runner evaluates exact per-sheet summaries
```

Single-spec legacy plans will also rerun. That is intentional: it keeps the compatibility path simple and ensures all approval decisions after this change are based on unambiguous plan-check results.

## Error Handling

If the plan-check run is already `RUNNING`, approval generation keeps the current behavior and returns `done=false`.

If recreating a stale plan-check run fails, approval generation returns an error. The caller already logs approval-generation errors without persisting a partial approval result, and the user can retry by rerunning plan checks.

If a fresh `DONE` plan-check run lacks an exact result for a sheet-backed target, approval generation should not fall back to a database-only result. The target receives only base CEL variables, matching the existing behavior for missing summary data but without consuming ambiguous data.

## Testing

Add coverage for three behaviors:

1. Plan-check result identity propagation: `CombinedExecutor.RunForTarget` copies `CheckTarget.SheetSha256` into tagged results.
2. Approval summary matching: two sheet-backed specs targeting the same database produce CEL variables containing both specs' statement-summary data instead of only one surviving database-level summary.
3. Legacy auto-rerun: a completed plan-check run with statement-summary results missing `sheet_sha256` causes approval generation to recreate/tickle plan checks and return `done=false`, rather than generating an approval template from ambiguous data.

These tests should check behavior through the narrowest practical boundary. Unit-level tests are preferred for result tagging and summary matching; integration-level coverage is acceptable where store and scheduler interactions are required for the rerun path.

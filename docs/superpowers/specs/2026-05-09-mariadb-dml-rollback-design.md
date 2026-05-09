# MariaDB DML Rollback Support Design

## Context

BYT-9420 asks for official MariaDB rollback support. The customer observed that rollback worked when a MariaDB instance was accidentally configured as MySQL, then learned MariaDB was not officially supported.

Bytebase already has DML rollback through the prior-backup flow for MySQL. The current MariaDB gap is mostly explicit gating and registration:

- `EngineSupportPriorBackup` excludes `MARIADB`.
- The MySQL DML-to-backup transformer is registered only for `MYSQL`.
- The MySQL rollback SQL generator is registered only for `MYSQL`.
- The built-in prior-backup SQL review rule is registered only for `MYSQL`.
- Schema sync checks backup availability for MySQL, TiDB, and MSSQL, but not MariaDB.
- SQL review schema YAML does not expose `BUILTIN_PRIOR_BACKUP_CHECK` for MariaDB.

Many adjacent MySQL parser and advisor paths already register MariaDB, including parsing, splitting, validation, query span, and most SQL review rules.

## Goal

Support DML rollback for MariaDB by making MariaDB a first-class consumer of the existing MySQL prior-backup and rollback implementation.

This design intentionally covers DML rollback only. It does not promise schema rollback or broader rollback semantics.

## Non-Goals

- No schema rollback support.
- No MariaDB-specific SQL dialect expansion.
- No new frontend rollback workflow.
- No version gate for MariaDB unless implementation or verification exposes a concrete incompatibility.
- No separate MariaDB backup or restore generator fork.

## Recommended Approach

Enable MariaDB as an alias of the existing MySQL DML rollback path.

The engine should remain `MARIADB` in task, instance, and API state. The parser and advisor registries should route MariaDB to the same functions MySQL uses for prior-backup validation, backup SQL generation, and restore SQL generation.

This matches the customer evidence: the feature already worked against MariaDB when the instance was mislabeled as MySQL. It also matches existing Bytebase patterns where MariaDB shares MySQL parser and advisor implementations.

## Alternatives Considered

### Conservative Version or Feature Gate

This would enable the same wiring behind a MariaDB version gate or feature flag. It reduces perceived rollout risk but adds configuration and product complexity without a known incompatibility. This should only be used if verification finds version-specific behavior that breaks backup or restore SQL.

### Separate MariaDB Implementation

This would fork backup and restore behavior into MariaDB-specific functions. It provides maximum future flexibility but duplicates MySQL logic today. It is not justified unless MariaDB-specific syntax differences require separate handling.

## Architecture

The core implementation should update existing gates and registries:

- `common.EngineSupportPriorBackup(storepb.Engine_MARIADB)` returns true.
- `mysql.TransformDMLToSelect` is registered for `storepb.Engine_MARIADB`.
- `mysql.GenerateRestoreSQL` is registered for `storepb.Engine_MARIADB`.
- `StatementPriorBackupCheckAdvisor` is registered for `storepb.Engine_MARIADB`.
- Schema sync treats `MARIADB` like MySQL/TiDB/MSSQL for `backup_available`, checking for `bbdataarchive`.
- SQL review schema YAML includes `BUILTIN_PRIOR_BACKUP_CHECK` for MariaDB.

No frontend routing change is expected. Existing React rollback UI is driven by `taskRun.hasPriorBackup`, so once the backend creates prior backup details for MariaDB and rollback preview works, the current UI should show rollback actions naturally.

## Behavior and Data Flow

For a MariaDB database change task with prior backup enabled:

1. Plan checks run `BUILTIN_PRIOR_BACKUP_CHECK` for MariaDB.
2. The prior-backup advisor uses the existing MySQL AST-based validation.
3. Validation rejects unsupported prior-backup shapes such as mixed DDL/DML and mixed DML types on the same table.
4. Validation requires the backup database `bbdataarchive`.
5. During task execution, `EngineSupportPriorBackup(MARIADB)` allows `DatabaseMigrateExecutor.backupData` to run.
6. `backupData` calls `parserbase.TransformDMLToSelect` with engine `MARIADB`.
7. The MariaDB registration routes to the existing MySQL backup SQL generation.
8. Backup tables are created and populated in `bbdataarchive` before the DML executes.
9. The task run result sets `has_prior_backup` when backup detail contains items.
10. Rollback preview calls `parserbase.GenerateRestoreSQL` with engine `MARIADB`.
11. The MariaDB registration routes to the existing MySQL restore SQL generation.
12. The frontend sees `hasPriorBackup` and uses the existing rollback sheet to preview and create a rollback plan.

The supported statement surface should match MySQL's current prior-backup behavior for DML rollback, primarily `UPDATE` and `DELETE` paths covered by the existing MySQL implementation.

## Error Handling

MariaDB should inherit MySQL's current validation and runtime failures. Parse failures, unsupported DML shapes, mixed DML on the same table, missing `bbdataarchive`, missing metadata, and missing disjoint unique keys for update restore should return the same error classes and messages as MySQL unless a message explicitly names MySQL.

One consistency cleanup should be included: task execution currently sets backup table comments only for `Engine_MYSQL`. MariaDB should be included in that case so backup table source metadata is preserved.

## Testing

Tests should focus on the gates that currently block MariaDB rather than duplicating the full MySQL fixture corpus.

Required coverage:

- Parser registration: `parserbase.TransformDMLToSelect` works with `Engine_MARIADB` for representative DML.
- Restore registration: `parserbase.GenerateRestoreSQL` works with `Engine_MARIADB` for representative backup detail.
- Advisor registration/config: `BUILTIN_PRIOR_BACKUP_CHECK` is available for MariaDB and flags a representative invalid prior-backup case.
- Engine support: `EngineSupportPriorBackup(MARIADB)` returns true.
- Backup availability: MariaDB schema sync or the relevant lower-level path reports `backup_available` when `bbdataarchive` exists.
- Regression: existing MySQL prior-backup and restore tests remain unchanged and passing.

Optional manual verification:

- Run a MariaDB-backed DML change with prior backup enabled.
- Confirm the task run has `has_prior_backup`.
- Preview rollback and confirm generated SQL uses the existing MySQL-style rollback syntax.

## Rollout and Risk

This change should ship without a feature flag. The behavior is gated by existing prior-backup enablement and by the presence of `bbdataarchive`.

Primary risks:

- MariaDB syntax or version behavior differs from MySQL for a statement shape already accepted by the MySQL implementation.
- Existing error messages may mention MySQL in a MariaDB context.
- Backup availability could remain false if schema sync is not updated consistently.

Mitigation:

- Keep scope limited to the existing MySQL statement surface.
- Add focused MariaDB routing and availability tests.
- Reuse existing validation so unsupported shapes fail before execution.
- Avoid broad UI or API changes.

Rollback plan:

- Revert the registration and gating changes. No database migration is needed.

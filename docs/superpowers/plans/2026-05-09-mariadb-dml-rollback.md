# MariaDB DML Rollback Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Enable MariaDB DML rollback by routing MariaDB through the existing MySQL prior-backup and rollback implementation.

**Architecture:** Keep the engine as `MARIADB` in task/API state, but register MariaDB against the existing MySQL prior-backup validator, DML-to-backup transformer, and restore SQL generator. Update the engine gates and availability checks that currently exclude MariaDB, then expose the built-in prior-backup rule in frontend SQL review schema data.

**Tech Stack:** Go backend, Bytebase parser/advisor registries, store protobuf types, React/Vite frontend YAML type data, Vitest.

---

## File Structure

- Modify `backend/common/engine.go`: add MariaDB to `EngineSupportPriorBackup`.
- Create `backend/common/engine_test.go`: focused regression test for MariaDB prior-backup support.
- Modify `backend/plugin/parser/mysql/backup.go`: register `TransformDMLToSelect` for `Engine_MARIADB`.
- Modify `backend/plugin/parser/mysql/restore.go`: register `GenerateRestoreSQL` for `Engine_MARIADB`.
- Modify `backend/plugin/parser/mysql/backup_test.go`: parserbase registration test for MariaDB backup SQL generation.
- Modify `backend/plugin/parser/mysql/restore_test.go`: parserbase registration test for MariaDB rollback SQL generation.
- Modify `backend/plugin/advisor/builtin_rules.go`: include MariaDB in built-in prior-backup rule selection.
- Modify `backend/plugin/advisor/mysql/rule_builtin_prior_backup_check.go`: register the advisor for MariaDB.
- Modify `backend/plugin/advisor/mysql/mysql_rules_test.go`: exercise MariaDB through `SQLReviewCheck` with appended built-in prior-backup rules.
- Modify `backend/runner/schemasync/syncer.go`: include MariaDB in backup availability lookup.
- Modify `backend/runner/schemasync/syncer_test.go`: unit-test the engine grouping used by backup availability.
- Modify `backend/runner/taskrun/database_migrate_executor.go`: include MariaDB in the MySQL-family backup table comment path.
- Modify `backend/runner/taskrun/database_migrate_executor_test.go`: unit-test the generated MySQL-family backup comment statement.
- Modify `frontend/src/types/sql-review-schema.yaml`: expose `BUILTIN_PRIOR_BACKUP_CHECK` for MariaDB.
- Modify `frontend/src/types/sqlReview.test.ts`: assert SQL review schema includes MariaDB prior-backup rule.

---

### Task 1: Enable the MariaDB Prior-Backup Engine Gate

**Files:**
- Create: `backend/common/engine_test.go`
- Modify: `backend/common/engine.go`

- [ ] **Step 1: Write the failing engine support test**

Create `backend/common/engine_test.go`:

```go
package common

import (
	"testing"

	"github.com/stretchr/testify/require"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
)

func TestEngineSupportPriorBackupMariaDB(t *testing.T) {
	require.True(t, EngineSupportPriorBackup(storepb.Engine_MARIADB))
}
```

- [ ] **Step 2: Run the focused test and verify it fails**

Run:

```bash
go test -v -count=1 ./backend/common -run '^TestEngineSupportPriorBackupMariaDB$'
```

Expected: FAIL because `EngineSupportPriorBackup(storepb.Engine_MARIADB)` currently returns false.

- [ ] **Step 3: Add MariaDB to the supported prior-backup engines**

In `backend/common/engine.go`, update `EngineSupportPriorBackup`:

```go
	case
		storepb.Engine_MYSQL,
		storepb.Engine_MARIADB,
		storepb.Engine_TIDB,
		storepb.Engine_MSSQL,
		storepb.Engine_ORACLE,
		storepb.Engine_POSTGRES:
		return true
```

Remove `storepb.Engine_MARIADB` from the false branch of the same switch.

- [ ] **Step 4: Format and run the focused test**

Run:

```bash
gofmt -w backend/common/engine.go backend/common/engine_test.go
go test -v -count=1 ./backend/common -run '^TestEngineSupportPriorBackupMariaDB$'
```

Expected: PASS.

- [ ] **Step 5: Commit Task 1**

```bash
git add backend/common/engine.go backend/common/engine_test.go
git commit -m "feat: enable prior backup for mariadb"
```

---

### Task 2: Register MariaDB Parser Backup and Restore Functions

**Files:**
- Modify: `backend/plugin/parser/mysql/backup.go`
- Modify: `backend/plugin/parser/mysql/restore.go`
- Modify: `backend/plugin/parser/mysql/backup_test.go`
- Modify: `backend/plugin/parser/mysql/restore_test.go`

- [ ] **Step 1: Write the failing MariaDB backup registry test**

Append this test to `backend/plugin/parser/mysql/backup_test.go`:

```go
func TestMariaDBTransformDMLToSelectRegistration(t *testing.T) {
	getter, lister := buildFixedMockDatabaseMetadataGetterAndLister()

	result, err := base.TransformDMLToSelect(context.Background(), store.Engine_MARIADB, base.TransformContext{
		GetDatabaseMetadataFunc: getter,
		ListDatabaseNamesFunc:   lister,
		IsCaseSensitive:         false,
	}, "DELETE FROM test WHERE b1 = 1;", "db", "backupDB", "_rollback")

	require.NoError(t, err)
	require.Len(t, result, 1)
	require.Equal(t, "test", result[0].SourceTableName)
	require.Equal(t, "_rollback_test_db", result[0].TargetTableName)
	require.Contains(t, result[0].Statement, "CREATE TABLE `backupDB`.`_rollback_test_db` LIKE `db`.`test`;")
}
```

- [ ] **Step 2: Write the failing MariaDB restore registry test**

Append this test to `backend/plugin/parser/mysql/restore_test.go`:

```go
func TestMariaDBGenerateRestoreSQLRegistration(t *testing.T) {
	getter, lister := buildFixedMockDatabaseMetadataGetterAndLister()

	result, err := base.GenerateRestoreSQL(context.Background(), store.Engine_MARIADB, base.RestoreContext{
		GetDatabaseMetadataFunc: getter,
		ListDatabaseNamesFunc:   lister,
		IsCaseSensitive:         false,
	}, "DELETE FROM test WHERE b1 = 1;", &store.PriorBackupDetail_Item{
		SourceTable: &store.PriorBackupDetail_Item_Table{
			Database: "instances/i1/databases/db",
			Table:    "test",
		},
		TargetTable: &store.PriorBackupDetail_Item_Table{
			Database: "instances/i1/databases/bbarchive",
			Table:    "prefix_test",
		},
		StartPosition: &store.Position{
			Line:   0,
			Column: 0,
		},
		EndPosition: &store.Position{
			Line:   math.MaxInt32,
			Column: 0,
		},
	})

	require.NoError(t, err)
	require.Equal(t, "/*\nOriginal SQL:\nDELETE FROM test WHERE b1 = 1;\n*/\nINSERT INTO `db`.`test` SELECT * FROM `bbarchive`.`prefix_test`;", result)
}
```

- [ ] **Step 3: Run the focused parser tests and verify they fail**

Run:

```bash
go test -v -count=1 ./backend/plugin/parser/mysql -run '^(TestMariaDBTransformDMLToSelectRegistration|TestMariaDBGenerateRestoreSQLRegistration)$'
```

Expected: FAIL with `engine MARIADB is not supported` from parserbase registration lookups.

- [ ] **Step 4: Register MariaDB in the backup transformer**

In `backend/plugin/parser/mysql/backup.go`, update `init`:

```go
func init() {
	base.RegisterTransformDMLToSelect(storepb.Engine_MYSQL, TransformDMLToSelect)
	base.RegisterTransformDMLToSelect(storepb.Engine_MARIADB, TransformDMLToSelect)
}
```

- [ ] **Step 5: Register MariaDB in the restore generator**

In `backend/plugin/parser/mysql/restore.go`, update `init`:

```go
func init() {
	base.RegisterGenerateRestoreSQL(storepb.Engine_MYSQL, GenerateRestoreSQL)
	base.RegisterGenerateRestoreSQL(storepb.Engine_MARIADB, GenerateRestoreSQL)
}
```

- [ ] **Step 6: Format and run parser tests**

Run:

```bash
gofmt -w backend/plugin/parser/mysql/backup.go backend/plugin/parser/mysql/restore.go backend/plugin/parser/mysql/backup_test.go backend/plugin/parser/mysql/restore_test.go
go test -v -count=1 ./backend/plugin/parser/mysql -run '^(TestMariaDBTransformDMLToSelectRegistration|TestMariaDBGenerateRestoreSQLRegistration|TestBackup|TestRestore)$'
```

Expected: PASS.

- [ ] **Step 7: Commit Task 2**

```bash
git add backend/plugin/parser/mysql/backup.go backend/plugin/parser/mysql/restore.go backend/plugin/parser/mysql/backup_test.go backend/plugin/parser/mysql/restore_test.go
git commit -m "feat: register mariadb rollback parser paths"
```

---

### Task 3: Enable MariaDB Built-In Prior-Backup Review

**Files:**
- Modify: `backend/plugin/advisor/builtin_rules.go`
- Modify: `backend/plugin/advisor/mysql/rule_builtin_prior_backup_check.go`
- Modify: `backend/plugin/advisor/mysql/mysql_rules_test.go`

- [ ] **Step 1: Write the failing MariaDB prior-backup advisor test**

Update the imports in `backend/plugin/advisor/mysql/mysql_rules_test.go`:

```go
import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/parser/sheet"
)
```

Append this test to the same file:

```go
func TestMariaDBPriorBackupCheckAdvisor(t *testing.T) {
	sm := sheet.NewManager()

	adviceList, err := advisor.SQLReviewCheck(context.Background(), sm, "CREATE TABLE t(id INT);\nUPDATE test SET c1 = 1 WHERE b1 = 1;", nil, advisor.Context{
		DBType:            storepb.Engine_MARIADB,
		DBSchema:          advisor.MockMySQLDatabase,
		EnablePriorBackup: true,
		InstanceID:        "instance",
		ListDatabaseNamesFunc: func(context.Context, string) ([]string, error) {
			return []string{"bbdataarchive"}, nil
		},
	})

	require.NoError(t, err)
	require.NotEmpty(t, adviceList)
	require.Equal(t, storepb.SQLReviewRule_BUILTIN_PRIOR_BACKUP_CHECK.String(), adviceList[0].Title)
	require.Contains(t, adviceList[0].Content, "mixed DDL and DML")
}
```

- [ ] **Step 2: Run the focused advisor test and verify it fails**

Run:

```bash
go test -v -count=1 ./backend/plugin/advisor/mysql -run '^TestMariaDBPriorBackupCheckAdvisor$'
```

Expected: FAIL because MariaDB does not receive the built-in prior-backup rule and the MySQL advisor is not registered for `Engine_MARIADB`.

- [ ] **Step 3: Include MariaDB in built-in prior-backup rules**

In `backend/plugin/advisor/builtin_rules.go`, update the first switch:

```go
	case storepb.Engine_MYSQL, storepb.Engine_MARIADB, storepb.Engine_POSTGRES, storepb.Engine_TIDB, storepb.Engine_MSSQL, storepb.Engine_ORACLE:
		rules = append(rules, &storepb.SQLReviewRule{
			Type:   storepb.SQLReviewRule_BUILTIN_PRIOR_BACKUP_CHECK,
			Level:  storepb.SQLReviewRule_WARNING,
			Engine: engine,
		})
```

- [ ] **Step 4: Register the MySQL prior-backup advisor for MariaDB**

In `backend/plugin/advisor/mysql/rule_builtin_prior_backup_check.go`, update `init`:

```go
func init() {
	advisor.Register(storepb.Engine_MYSQL, storepb.SQLReviewRule_BUILTIN_PRIOR_BACKUP_CHECK, &StatementPriorBackupCheckAdvisor{})
	advisor.Register(storepb.Engine_MARIADB, storepb.SQLReviewRule_BUILTIN_PRIOR_BACKUP_CHECK, &StatementPriorBackupCheckAdvisor{})
}
```

- [ ] **Step 5: Format and run advisor tests**

Run:

```bash
gofmt -w backend/plugin/advisor/builtin_rules.go backend/plugin/advisor/mysql/rule_builtin_prior_backup_check.go backend/plugin/advisor/mysql/mysql_rules_test.go
go test -v -count=1 ./backend/plugin/advisor/mysql -run '^TestMariaDBPriorBackupCheckAdvisor$'
go test -v -count=1 ./backend/plugin/advisor -run '^Test'
```

Expected: PASS.

- [ ] **Step 6: Commit Task 3**

```bash
git add backend/plugin/advisor/builtin_rules.go backend/plugin/advisor/mysql/rule_builtin_prior_backup_check.go backend/plugin/advisor/mysql/mysql_rules_test.go
git commit -m "feat: enable mariadb prior backup review"
```

---

### Task 4: Mark MariaDB Backup Availability and Backup Comments

**Files:**
- Modify: `backend/runner/schemasync/syncer.go`
- Modify: `backend/runner/schemasync/syncer_test.go`
- Modify: `backend/runner/taskrun/database_migrate_executor.go`
- Modify: `backend/runner/taskrun/database_migrate_executor_test.go`

- [ ] **Step 1: Write a failing backup availability engine grouping test**

Append this test to `backend/runner/schemasync/syncer_test.go`:

```go
func TestEngineUsesBackupDatabaseLookupForPriorBackup(t *testing.T) {
	require.True(t, engineUsesBackupDatabaseLookupForPriorBackup(storepb.Engine_MYSQL))
	require.True(t, engineUsesBackupDatabaseLookupForPriorBackup(storepb.Engine_MARIADB))
	require.True(t, engineUsesBackupDatabaseLookupForPriorBackup(storepb.Engine_TIDB))
	require.True(t, engineUsesBackupDatabaseLookupForPriorBackup(storepb.Engine_MSSQL))
	require.False(t, engineUsesBackupDatabaseLookupForPriorBackup(storepb.Engine_POSTGRES))
	require.False(t, engineUsesBackupDatabaseLookupForPriorBackup(storepb.Engine_ORACLE))
}
```

- [ ] **Step 2: Write a failing MySQL-family backup comment test**

Append this test to `backend/runner/taskrun/database_migrate_executor_test.go`:

```go
func TestBuildMySQLFamilyBackupTableCommentStatement(t *testing.T) {
	require.Equal(
		t,
		"ALTER TABLE `bbdataarchive`.`_rollback_test_db` COMMENT = 'task 101, source table (app, test)'",
		buildMySQLFamilyBackupTableCommentStatement("bbdataarchive", "_rollback_test_db", "task 101", "app", "test"),
	)
}
```

- [ ] **Step 3: Run focused tests and verify they fail**

Run:

```bash
go test -v -count=1 ./backend/runner/schemasync -run '^TestEngineUsesBackupDatabaseLookupForPriorBackup$'
go test -v -count=1 ./backend/runner/taskrun -run '^TestBuildMySQLFamilyBackupTableCommentStatement$'
```

Expected: FAIL because both helper functions do not exist yet.

- [ ] **Step 4: Add a pure backup availability engine helper**

In `backend/runner/schemasync/syncer.go`, add this helper near `databaseBackupAvailable`:

```go
func engineUsesBackupDatabaseLookupForPriorBackup(engine storepb.Engine) bool {
	switch engine {
	case storepb.Engine_MYSQL, storepb.Engine_MARIADB, storepb.Engine_MSSQL, storepb.Engine_TIDB:
		return true
	default:
		return false
	}
}
```

Then update `databaseBackupAvailable`:

```go
	case storepb.Engine_POSTGRES:
		if dbMetadata == nil {
			return false
		}
		for _, schema := range dbMetadata.Schemas {
			if schema.GetName() == common.BackupDatabaseNameOfEngine(storepb.Engine_POSTGRES) {
				return true
			}
		}
	case storepb.Engine_ORACLE:
		dbName := common.BackupDatabaseNameOfEngine(storepb.Engine_ORACLE)
		backupDB, err := s.store.GetDatabase(ctx, &store.FindDatabaseMessage{
			InstanceID:   &instance.ResourceID,
			DatabaseName: &dbName,
		})
		if err != nil {
			slog.Debug("Failed to get backup database", "err", err)
			return false
		}
		return backupDB != nil
	default:
		if engineUsesBackupDatabaseLookupForPriorBackup(instance.Metadata.GetEngine()) {
			dbName := common.BackupDatabaseNameOfEngine(instance.Metadata.GetEngine())
			backupDB, err := s.store.GetDatabase(ctx, &store.FindDatabaseMessage{
				InstanceID:   &instance.ResourceID,
				DatabaseName: &dbName,
			})
			if err != nil {
				slog.Debug("Failed to get backup database", "err", err)
				return false
			}
			return backupDB != nil
		}
		// Unsupported database engine for backup
		slog.Debug("Unsupported database engine for backup", "engine", instance.Metadata.GetEngine())
		return false
```

- [ ] **Step 5: Add the MySQL-family backup comment helper**

In `backend/runner/taskrun/database_migrate_executor.go`, add this helper near `backupData`:

```go
func buildMySQLFamilyBackupTableCommentStatement(backupDatabaseName, targetTableName, bbSource, databaseName, sourceTableName string) string {
	return fmt.Sprintf("ALTER TABLE `%s`.`%s` COMMENT = '%s, source table (%s, %s)'", backupDatabaseName, targetTableName, bbSource, databaseName, sourceTableName)
}
```

Then replace the duplicate TiDB/MySQL switch cases inside `backupData` with:

```go
			case storepb.Engine_TIDB, storepb.Engine_MYSQL, storepb.Engine_MARIADB:
				if _, err := driver.Execute(driverCtx, buildMySQLFamilyBackupTableCommentStatement(backupDatabaseName, statement.TargetTableName, bbSource, database.DatabaseName, statement.SourceTableName), db.ExecuteOptions{}); err != nil {
					return nil, errors.Wrap(err, "failed to set table comment")
				}
```

- [ ] **Step 6: Format and run focused runner tests**

Run:

```bash
gofmt -w backend/runner/schemasync/syncer.go backend/runner/schemasync/syncer_test.go backend/runner/taskrun/database_migrate_executor.go backend/runner/taskrun/database_migrate_executor_test.go
go test -v -count=1 ./backend/runner/schemasync -run '^TestEngineUsesBackupDatabaseLookupForPriorBackup$'
go test -v -count=1 ./backend/runner/taskrun -run '^TestBuildMySQLFamilyBackupTableCommentStatement$'
```

Expected: PASS.

- [ ] **Step 7: Commit Task 4**

```bash
git add backend/runner/schemasync/syncer.go backend/runner/schemasync/syncer_test.go backend/runner/taskrun/database_migrate_executor.go backend/runner/taskrun/database_migrate_executor_test.go
git commit -m "feat: mark mariadb rollback backup availability"
```

---

### Task 5: Expose the Prior-Backup Rule for MariaDB in Frontend SQL Review Schema

**Files:**
- Modify: `frontend/src/types/sql-review-schema.yaml`
- Modify: `frontend/src/types/sqlReview.test.ts`

- [ ] **Step 1: Write the failing frontend schema test**

Inside the `"Schema validation"` describe block in `frontend/src/types/sqlReview.test.ts`, add:

```ts
    test("schema exposes built-in prior backup check for MariaDB", () => {
      expect(
        schemaRules.some(
          (rule) =>
            rule.engine === "MARIADB" &&
            rule.type === "BUILTIN_PRIOR_BACKUP_CHECK" &&
            rule.category === "BUILTIN"
        )
      ).toBe(true);
    });
```

- [ ] **Step 2: Run the focused frontend test and verify it fails**

Run:

```bash
pnpm --dir frontend test -- src/types/sqlReview.test.ts
```

Expected: FAIL because `sql-review-schema.yaml` does not contain `MARIADB:BUILTIN_PRIOR_BACKUP_CHECK`.

- [ ] **Step 3: Add MariaDB to SQL review schema YAML**

In `frontend/src/types/sql-review-schema.yaml`, add this entry after the Oracle prior-backup entry and before `BUILTIN_WALK_THROUGH_CHECK`:

```yaml
- type: BUILTIN_PRIOR_BACKUP_CHECK
  category: BUILTIN
  engine: MARIADB
```

- [ ] **Step 4: Run frontend fix and focused test**

Run:

```bash
pnpm --dir frontend fix
pnpm --dir frontend test -- src/types/sqlReview.test.ts
```

Expected: PASS.

- [ ] **Step 5: Commit Task 5**

```bash
git add frontend/src/types/sql-review-schema.yaml frontend/src/types/sqlReview.test.ts
git commit -m "feat: expose mariadb prior backup review rule"
```

---

### Task 6: Final Verification

**Files:**
- Verify all modified files.

- [ ] **Step 1: Run Go formatting**

Run:

```bash
gofmt -w backend/common/engine.go backend/common/engine_test.go backend/plugin/parser/mysql/backup.go backend/plugin/parser/mysql/restore.go backend/plugin/parser/mysql/backup_test.go backend/plugin/parser/mysql/restore_test.go backend/plugin/advisor/builtin_rules.go backend/plugin/advisor/mysql/rule_builtin_prior_backup_check.go backend/plugin/advisor/mysql/mysql_rules_test.go backend/runner/schemasync/syncer.go backend/runner/schemasync/syncer_test.go backend/runner/taskrun/database_migrate_executor.go backend/runner/taskrun/database_migrate_executor_test.go
```

Expected: command exits 0.

- [ ] **Step 2: Run focused Go tests**

Run:

```bash
go test -v -count=1 ./backend/common -run '^TestEngineSupportPriorBackupMariaDB$'
go test -v -count=1 ./backend/plugin/parser/mysql -run '^(TestMariaDBTransformDMLToSelectRegistration|TestMariaDBGenerateRestoreSQLRegistration|TestBackup|TestRestore)$'
go test -v -count=1 ./backend/plugin/advisor/mysql -run '^TestMariaDBPriorBackupCheckAdvisor$'
go test -v -count=1 ./backend/runner/schemasync -run '^TestEngineUsesBackupDatabaseLookupForPriorBackup$'
go test -v -count=1 ./backend/runner/taskrun -run '^TestBuildMySQLFamilyBackupTableCommentStatement$'
```

Expected: all PASS.

- [ ] **Step 3: Run frontend checks for touched files**

Run:

```bash
pnpm --dir frontend fix
pnpm --dir frontend check
pnpm --dir frontend type-check
pnpm --dir frontend test -- src/types/sqlReview.test.ts
```

Expected: all PASS.

- [ ] **Step 4: Run backend lint repeatedly until clean**

Run:

```bash
golangci-lint run --allow-parallel-runners
```

Expected: PASS. If it reports issues, run:

```bash
golangci-lint run --fix --allow-parallel-runners
golangci-lint run --allow-parallel-runners
```

Repeat until `golangci-lint run --allow-parallel-runners` exits 0.

- [ ] **Step 5: Build backend**

Run:

```bash
go build -ldflags "-w -s" -p=16 -o ./bytebase-build/bytebase ./backend/bin/server/main.go
```

Expected: command exits 0 and produces `./bytebase-build/bytebase`.

- [ ] **Step 6: Review diff for scope**

Run:

```bash
git diff --stat
git diff -- backend/common/engine.go backend/plugin/parser/mysql/backup.go backend/plugin/parser/mysql/restore.go backend/plugin/advisor/builtin_rules.go backend/plugin/advisor/mysql/rule_builtin_prior_backup_check.go backend/runner/schemasync/syncer.go backend/runner/taskrun/database_migrate_executor.go frontend/src/types/sql-review-schema.yaml
```

Expected: diff only contains MariaDB prior-backup/rollback enablement, tests, and the frontend schema rule.

- [ ] **Step 7: Final commit if verification changed files**

If final verification commands modified files, commit them:

```bash
git add backend frontend
git commit -m "chore: apply mariadb rollback verification fixes"
```

If no files changed after Task 5, skip this commit.

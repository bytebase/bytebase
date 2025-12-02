# DDL/DML Plan Check Consolidation Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Remove DDL/DML distinction from plan checks and SQL review, consolidating to `CHANGE_DATABASE` and `SDL` types.

**Architecture:** Update proto enum, add `EnableGhost` to advisor context, remove mix rules, update remaining rules to run unconditionally, migrate stored data.

**Tech Stack:** Go, Protocol Buffers, PostgreSQL migrations

---

## Task 1: Update Proto Enum

**Files:**
- Modify: `proto/store/store/plan_check_run.proto:27-33`

**Step 1: Update the enum definition**

```proto
  // ChangeDatabaseType extends MigrationType with additional execution contexts.
  enum ChangeDatabaseType {
    CHANGE_DATABASE_TYPE_UNSPECIFIED = 0;
    CHANGE_DATABASE = 1;
    SDL = 2;
  }
```

**Step 2: Remove outdated comment**

Remove lines 25-26:
```proto
  // Note: DDL, DML, and DDL_GHOST values align with MigrationType enum values.
```

**Step 3: Generate proto code**

Run: `cd proto && buf generate`

**Step 4: Verify generation succeeded**

Run: `grep -n "CHANGE_DATABASE\|SDL" backend/generated-go/store/plan_check_run.pb.go | head -10`

Expected: See `CHANGE_DATABASE = 1` and `SDL = 2`

**Step 5: Commit**

```
feat(advisor): consolidate ChangeDatabaseType enum to CHANGE_DATABASE and SDL
```

---

## Task 2: Add EnableGhost to Advisor Context

**Files:**
- Modify: `backend/plugin/advisor/advisor.go:38-71`

**Step 1: Add EnableGhost field to Context struct**

After line 41 (`EnablePriorBackup bool`), add:

```go
	EnableGhost           bool
```

**Step 2: Verify no syntax errors**

Run: `go build ./backend/plugin/advisor/...`

Expected: Build succeeds

**Step 3: Commit**

```
feat(advisor): add EnableGhost field to advisor Context
```

---

## Task 3: Update Statement Advise Executor to Pass EnableGhost

**Files:**
- Modify: `backend/runner/plancheck/statement_advise_executor.go:47-100`
- Modify: `backend/runner/plancheck/statement_advise_executor.go:119-186`

**Step 1: Extract EnableGhost from config in Run method**

After line 72 (`enablePriorBackup := config.EnablePriorBackup`), add:

```go
	enableGhost := config.EnableGhost
```

**Step 2: Update runReview call to pass enableGhost**

Change line 100:
```go
	results, err := e.runReview(ctx, instance, database, changeType, statement, enablePriorBackup)
```
To:
```go
	results, err := e.runReview(ctx, instance, database, changeType, statement, enablePriorBackup, enableGhost)
```

**Step 3: Update runReview function signature**

Change lines 119-126:
```go
func (e *StatementAdviseExecutor) runReview(
	ctx context.Context,
	instance *store.InstanceMessage,
	database *store.DatabaseMessage,
	changeType storepb.PlanCheckRunConfig_ChangeDatabaseType,
	statement string,
	enablePriorBackup bool,
) ([]*storepb.PlanCheckRunResult_Result, error) {
```
To:
```go
func (e *StatementAdviseExecutor) runReview(
	ctx context.Context,
	instance *store.InstanceMessage,
	database *store.DatabaseMessage,
	changeType storepb.PlanCheckRunConfig_ChangeDatabaseType,
	statement string,
	enablePriorBackup bool,
	enableGhost bool,
) ([]*storepb.PlanCheckRunResult_Result, error) {
```

**Step 4: Add EnableGhost to advisor.Context**

In the advisor.Context struct literal (around line 172-186), after `EnablePriorBackup: enablePriorBackup,` add:

```go
		EnableGhost:              enableGhost,
```

**Step 5: Verify build**

Run: `go build ./backend/runner/plancheck/...`

Expected: Build succeeds

**Step 6: Commit**

```
feat(advisor): pass EnableGhost to advisor context
```

---

## Task 4: Update Type Conversion Function

**Files:**
- Modify: `backend/api/v1/plan_service_plan_check.go:246-258`

**Step 1: Simplify convertToChangeDatabaseType function**

Replace the entire function:

```go
func convertToChangeDatabaseType(t storepb.PlanConfig_ChangeDatabaseConfig_Type) storepb.PlanCheckRunConfig_ChangeDatabaseType {
	switch t {
	case storepb.PlanConfig_ChangeDatabaseConfig_MIGRATE:
		return storepb.PlanCheckRunConfig_CHANGE_DATABASE
	case storepb.PlanConfig_ChangeDatabaseConfig_SDL:
		return storepb.PlanCheckRunConfig_SDL
	default:
		return storepb.PlanCheckRunConfig_CHANGE_DATABASE_TYPE_UNSPECIFIED
	}
}
```

**Step 2: Find and update all callers**

Run: `grep -rn "convertToChangeDatabaseType" backend/`

Update each call site to remove the `enableGhost` parameter (should be passed separately now).

**Step 3: Verify build**

Run: `go build ./backend/api/v1/...`

**Step 4: Commit**

```
refactor(api): simplify convertToChangeDatabaseType, remove ghost handling
```

---

## Task 5: Remove Disallow-Mix Rules

**Files to delete:**
- `backend/plugin/advisor/mysql/rule_statement_disallow_mix_in_ddl.go`
- `backend/plugin/advisor/mysql/rule_statement_disallow_mix_in_dml.go`
- `backend/plugin/advisor/pg/advisor_statement_disallow_mix_in_ddl.go`
- `backend/plugin/advisor/pg/advisor_statement_disallow_mix_in_dml.go`
- `backend/plugin/advisor/mssql/rule_statement_disallow_mix_in_ddl.go`
- `backend/plugin/advisor/mssql/rule_statement_disallow_mix_in_dml.go`
- `backend/plugin/advisor/oracle/rule_statement_disallow_mix_in_ddl.go`
- `backend/plugin/advisor/oracle/rule_statement_disallow_mix_in_dml.go`
- `backend/plugin/advisor/tidb/advisor_statement_disallow_mix_in_ddl.go`
- `backend/plugin/advisor/tidb/advisor_statement_disallow_mix_in_dml.go`

**Step 1: Delete all mix rule files**

Run:
```bash
rm backend/plugin/advisor/mysql/rule_statement_disallow_mix_in_ddl.go
rm backend/plugin/advisor/mysql/rule_statement_disallow_mix_in_dml.go
rm backend/plugin/advisor/pg/advisor_statement_disallow_mix_in_ddl.go
rm backend/plugin/advisor/pg/advisor_statement_disallow_mix_in_dml.go
rm backend/plugin/advisor/mssql/rule_statement_disallow_mix_in_ddl.go
rm backend/plugin/advisor/mssql/rule_statement_disallow_mix_in_dml.go
rm backend/plugin/advisor/oracle/rule_statement_disallow_mix_in_ddl.go
rm backend/plugin/advisor/oracle/rule_statement_disallow_mix_in_dml.go
rm backend/plugin/advisor/tidb/advisor_statement_disallow_mix_in_ddl.go
rm backend/plugin/advisor/tidb/advisor_statement_disallow_mix_in_dml.go
```

**Step 2: Remove rule type constants from sql_review.go**

In `backend/plugin/advisor/sql_review.go`, remove lines 121-123:
```go
	// SchemaRuleStatementDisallowMixInDDL disallows DML statements in DDL statements.
	SchemaRuleStatementDisallowMixInDDL SQLReviewRuleType = "statement.disallow-mix-in-ddl"
	// SchemaRuleStatementDisallowMixInDML disallows DDL statements in DML statements.
	SchemaRuleStatementDisallowMixInDML SQLReviewRuleType = "statement.disallow-mix-in-dml"
```

**Step 3: Verify build**

Run: `go build ./backend/plugin/advisor/...`

Expected: Build succeeds (rules are self-registering via init())

**Step 4: Commit**

```
refactor(advisor): remove disallow-mix-in-ddl and disallow-mix-in-dml rules
```

---

## Task 6: Update Prior Backup Check Rules

**Files:**
- Modify: `backend/plugin/advisor/mysql/rule_builtin_prior_backup_check.go:29-32`
- Modify: `backend/plugin/advisor/pg/advisor_builtin_prior_backup_check.go:37-40`
- Modify: `backend/plugin/advisor/mssql/rule_builtin_prior_backup_check.go:36-39`
- Modify: `backend/plugin/advisor/oracle/rule_builtin_prior_backup_check.go:29-33`
- Modify: `backend/plugin/advisor/tidb/advisor_builtin_prior_backup_check.go:36-40`

**Step 1: Update MySQL prior backup check**

In `backend/plugin/advisor/mysql/rule_builtin_prior_backup_check.go`, change line 30:
```go
	if !checkCtx.EnablePriorBackup || checkCtx.ChangeType != storepb.PlanCheckRunConfig_DML {
```
To:
```go
	if !checkCtx.EnablePriorBackup {
```

**Step 2: Update PostgreSQL prior backup check**

In `backend/plugin/advisor/pg/advisor_builtin_prior_backup_check.go`, change line 39:
```go
	if !checkCtx.EnablePriorBackup || checkCtx.ChangeType != storepb.PlanCheckRunConfig_DML {
```
To:
```go
	if !checkCtx.EnablePriorBackup {
```

**Step 3: Update MSSQL prior backup check**

In `backend/plugin/advisor/mssql/rule_builtin_prior_backup_check.go`, change line 37:
```go
	if !checkCtx.EnablePriorBackup || checkCtx.ChangeType != storepb.PlanCheckRunConfig_DML {
```
To:
```go
	if !checkCtx.EnablePriorBackup {
```

**Step 4: Update Oracle prior backup check**

In `backend/plugin/advisor/oracle/rule_builtin_prior_backup_check.go`, change line 31:
```go
	if !checkCtx.EnablePriorBackup || checkCtx.ChangeType != storepb.PlanCheckRunConfig_DML {
```
To:
```go
	if !checkCtx.EnablePriorBackup {
```

**Step 5: Update TiDB prior backup check**

In `backend/plugin/advisor/tidb/advisor_builtin_prior_backup_check.go`, change line 38:
```go
	if !checkCtx.EnablePriorBackup || checkCtx.ChangeType != storepb.PlanCheckRunConfig_DML {
```
To:
```go
	if !checkCtx.EnablePriorBackup {
```

**Step 6: Verify build**

Run: `go build ./backend/plugin/advisor/...`

**Step 7: Commit**

```
refactor(advisor): remove ChangeType check from prior backup rules
```

---

## Task 7: Update Online Migration Rule

**Files:**
- Modify: `backend/plugin/advisor/mysql/rule_online_migration.go:56-70,100-120`

**Step 1: Update gh-ost database check**

Change lines 56-70:
```go
	// Check gh-ost database existence first if the change type is gh-ost.
	if checkCtx.ChangeType == storepb.PlanCheckRunConfig_DDL_GHOST {
```
To:
```go
	// Check gh-ost database existence first if gh-ost is enabled.
	if checkCtx.EnableGhost {
```

**Step 2: Update single statement check**

Change lines 100-103:
```go
	if len(adviceList) == 1 && len(stmtList) == 1 {
		if checkCtx.ChangeType == storepb.PlanCheckRunConfig_DDL_GHOST {
			return nil, nil
		}
```
To:
```go
	if len(adviceList) == 1 && len(stmtList) == 1 {
		if checkCtx.EnableGhost {
			return nil, nil
		}
```

**Step 3: Update no-statement-needs-migration check**

Change lines 111-119:
```go
	if len(adviceList) == 0 {
		if checkCtx.ChangeType == storepb.PlanCheckRunConfig_DDL_GHOST {
			return []*storepb.Advice{{
```
To:
```go
	if len(adviceList) == 0 {
		if checkCtx.EnableGhost {
			return []*storepb.Advice{{
```

**Step 4: Verify build**

Run: `go build ./backend/plugin/advisor/mysql/...`

**Step 5: Commit**

```
refactor(advisor): update online migration rule to use EnableGhost
```

---

## Task 8: Create Database Migration

**Files:**
- Create: `backend/migrator/migration/3.13/0007##plan_check_ddl_dml_consolidation.sql`

**Step 1: Create migration file**

```sql
-- Consolidate ChangeDatabaseType in plan_check_run configs.
-- DDL(1), DML(2), DDL_GHOST(4) -> CHANGE_DATABASE
-- SDL(3) -> SDL

-- Update DDL, DML, DDL_GHOST to CHANGE_DATABASE
UPDATE plan_check_run
SET config = jsonb_set(config, '{changeDatabaseType}', '"CHANGE_DATABASE"')
WHERE config->>'changeDatabaseType' IN ('DDL', 'DML', 'DDL_GHOST');

-- Update SDL to new value (stays as SDL but enum value changes from 3 to 2)
-- Note: protojson serializes as string, so we update the string value
UPDATE plan_check_run
SET config = jsonb_set(config, '{changeDatabaseType}', '"SDL"')
WHERE config->>'changeDatabaseType' = 'SDL';

-- Remove disallow-mix rules from review_config if they exist
UPDATE review_config
SET payload = jsonb_set(
  payload,
  '{rules}',
  COALESCE(
    (SELECT jsonb_agg(r)
     FROM jsonb_array_elements(payload->'rules') r
     WHERE r->>'type' NOT IN (
       'statement.disallow-mix-in-ddl',
       'statement.disallow-mix-in-dml'
     )),
    '[]'::jsonb
  )
)
WHERE payload->'rules' IS NOT NULL
  AND jsonb_array_length(payload->'rules') > 0;
```

**Step 2: Verify migration syntax**

Run against dev database:
```bash
psql "postgres://bbdev@127.0.0.1:5983/bbdev" -c "\i backend/migrator/migration/3.13/0007##plan_check_ddl_dml_consolidation.sql"
```

**Step 3: Commit**

```
chore(migration): add DDL/DML consolidation migration for plan checks
```

---

## Task 9: Run Linter and Fix Issues

**Step 1: Run golangci-lint**

Run: `golangci-lint run --allow-parallel-runners`

**Step 2: Fix any issues**

Run repeatedly until clean: `golangci-lint run --fix --allow-parallel-runners`

**Step 3: Commit fixes if any**

```
fix: address linter issues from DDL/DML consolidation
```

---

## Task 10: Build and Verify

**Step 1: Full backend build**

Run: `go build -ldflags "-w -s" -p=16 -o ./bytebase-build/bytebase ./backend/bin/server/main.go`

Expected: Build succeeds

**Step 2: Run relevant tests**

Run: `go test -v ./backend/plugin/advisor/...`

**Step 3: Commit final state**

```
chore: complete DDL/DML plan check consolidation
```

---

## Summary Checklist

- [ ] Task 1: Update proto enum
- [ ] Task 2: Add EnableGhost to advisor Context
- [ ] Task 3: Update statement advise executor
- [ ] Task 4: Update type conversion function
- [ ] Task 5: Remove disallow-mix rules (10 files)
- [ ] Task 6: Update prior backup check rules (5 files)
- [ ] Task 7: Update online migration rule
- [ ] Task 8: Create database migration
- [ ] Task 9: Run linter and fix issues
- [ ] Task 10: Build and verify

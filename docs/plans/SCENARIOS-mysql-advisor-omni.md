# MySQL Advisor Omni Migration Scenarios

> Goal: Migrate all 87 MySQL advisor rules from ANTLR tree walking to omni AST type switches
> Verification: `go test -count=1 ./backend/plugin/advisor/mysql/...` passes with identical advice output; `grep -r "GetANTLRAST" backend/plugin/advisor/mysql/` returns zero matches after completion
> Reference sources: PG omni migration (`backend/plugin/advisor/pg/generic_checker_omni.go`, `utils_omni.go`), omni MySQL AST (`github.com/bytebase/omni/mysql/ast`)

Status: [ ] pending, [x] passing, [~] partial

---

## Phase 1: Framework & Proof

Build the omni rule infrastructure and prove it works with one representative rule from each major category.

### 1.1 Core Framework

- [x] `OmniRule` interface with `OnStatement(node ast.Node)`, `Name()`, `GetAdviceList()`
- [x] `OmniBaseRule` struct with `Level`, `Title`, `Advice`, `BaseLine`, `StmtText` fields
- [x] `SetStatement(baseLine int, stmtText string)` sets context for position calculations
- [x] `AddAdvice()` adds advice with BaseLine offset adjustment
- [x] `LocToLine(loc ast.Loc)` converts omni byte offset to 1-based line number
- [x] `RunOmniRules()` dispatcher iterates statements, extracts omni node, dispatches to rules
- [x] `FindLineByName(name string)` searches identifier in statement text, returns line

### 1.2 Utility Functions

- [x] `omniTableName(ref *ast.TableRef)` extracts table name
- [x] `omniColumnNames(constraint *ast.Constraint)` extracts column name list from constraint
- [x] `omniIndexColumns(cols []*ast.IndexColumn)` extracts column names from index column list
- [x] `omniDataTypeName(dt *ast.DataType)` extracts normalized type name string
- [x] `omniIsNullable(col *ast.ColumnDef)` checks if column allows NULL
- [x] `omniHasDefault(col *ast.ColumnDef)` checks if column has DEFAULT
- [x] `omniIsAutoIncrement(col *ast.ColumnDef)` checks if column is AUTO_INCREMENT
- [x] `omniColumnComment(col *ast.ColumnDef)` extracts column COMMENT string
- [x] `omniTableOptionValue(opts []*ast.TableOption, name string)` extracts table option by name
- [x] `omniConstraintsByType(constraints []*ast.Constraint, typ ast.ConstraintType)` filters constraints

### 1.3 Proof-of-Concept Rules (one per category)

- [x] `rule_table_require_pk` migrated — DDL-table representative (CreateTable + AlterTable + DropTable)
- [x] `rule_column_no_null` migrated — DDL-column representative (CreateTable + AlterTable column inspection)
- [x] `rule_index_no_duplicate_column` migrated — DDL-index representative (CreateTable + AlterTable + CreateIndex)
- [x] `rule_naming_table` migrated — naming representative (CreateTable + AlterTable + RenameTable)
- [x] `rule_stmt_no_select_all` migrated — DML representative (SelectStmt target list inspection)
- [x] All 5 proof rules produce identical advice output to ANTLR versions

---

## Phase 2: DDL-Column Rules (21 rules)

### 2.1 Column Type & Attribute Rules

- [x] `rule_column_auto_increment_initial_value` — AUTO_INCREMENT initial value check
- [x] `rule_column_auto_increment_must_integer` — AUTO_INCREMENT must be integer type
- [x] `rule_column_auto_increment_must_unsigned` — AUTO_INCREMENT must be UNSIGNED
- [x] `rule_column_current_time_count_limit` — limit CURRENT_TIMESTAMP columns
- [x] `rule_column_maximum_character_length` — CHAR length limit
- [x] `rule_column_maximum_varchar_length` — VARCHAR length limit
- [x] `rule_column_type_disallow_list` — disallowed column types
- [x] `rule_table_text_fields_total_length` — total text field length limit

### 2.2 Column Constraint & Default Rules

- [x] `rule_column_no_null` — (already migrated in Phase 1, skip)
- [x] `rule_column_require_default` — columns must have DEFAULT
- [x] `rule_column_set_default_for_not_null` — NOT NULL columns must have DEFAULT
- [x] `rule_column_required` — required columns must exist
- [x] `rule_column_comment_convention` — column COMMENT convention

### 2.3 Column Modification Rules

- [x] `rule_column_disallow_changing` — disallow column changes
- [x] `rule_column_disallow_changing_order` — disallow column reordering
- [x] `rule_column_disallow_changing_type` — disallow column type changes
- [x] `rule_column_disallow_drop` — disallow column drops
- [x] `rule_column_disallow_drop_in_index` — disallow dropping indexed columns
- [x] `rule_column_disallow_set_charset` — disallow column-level charset
- [x] `rule_column_require_charset` — require column charset
- [x] `rule_column_require_collation` — require column collation

---

## Phase 3: DDL-Table Rules (11 rules)

### 3.1 Table Structure Rules

- [x] `rule_table_require_pk` — (already migrated in Phase 1, skip)
- [x] `rule_table_comment_convention` — table COMMENT convention
- [x] `rule_table_disallow_partition` — disallow partitioning
- [x] `rule_table_disallow_set_charset` — disallow table-level charset
- [x] `rule_table_require_charset` — require table charset
- [x] `rule_table_require_collation` — require table collation
- [x] `rule_table_limit_size` — table size limit check

### 3.2 Table DDL Policy Rules

- [x] `rule_table_disallow_ddl` — disallow DDL on specific tables
- [x] `rule_table_disallow_dml` — disallow DML on specific tables
- [x] `rule_table_drop_naming_convention` — drop table naming convention
- [x] `rule_use_innodb` — require InnoDB engine

---

## Phase 4: DDL-Index Rules (8 rules)

### 4.1 Index Rules

- [x] `rule_index_no_duplicate_column` — (already migrated in Phase 1, skip)
- [x] `rule_index_key_number_limit` — max columns per index
- [x] `rule_index_pk_type` — primary key type restrictions
- [x] `rule_index_primary_key_type_allowlist` — PK type allowlist
- [x] `rule_index_total_number_limit` — max indexes per table
- [x] `rule_index_type_allow_list` — index type allowlist
- [x] `rule_index_type_no_blob` — disallow BLOB in index
- [x] `rule_table_no_duplicate_index` — no duplicate indexes

---

## Phase 5: DDL-Constraint & Database & View & Misc (12 rules)

### 5.1 Constraint & Charset Rules

- [x] `rule_table_no_fk` — disallow foreign keys
- [x] `rule_charset_allowlist` — charset allowlist
- [x] `rule_collation_allowlist` — collation allowlist

### 5.2 Database, View & System Object Rules

- [x] `rule_database_drop_empty_db` — only drop empty databases
- [x] `rule_view_disallow_create` — disallow view creation
- [x] `rule_disallow_procedure` — disallow procedure creation
- [x] `rule_event_disallow_create` — disallow event creation
- [x] `rule_function_disallow_create` — disallow function creation
- [x] `rule_function_disallowed_list` — disallowed function list
- [x] `rule_table_disallow_trigger` — disallow trigger creation
- [x] `rule_migration_compatibility` — backward compatibility checks
- [x] `rule_online_migration` — online migration checks

---

## Phase 6: Naming Rules (7 rules)

### 6.1 Naming Convention Rules

- [x] `rule_naming_table` — (already migrated in Phase 1, skip)
- [x] `rule_naming_column` — column naming convention
- [x] `rule_naming_auto_increment_column` — auto-increment column naming
- [x] `rule_naming_identifier_no_keyword` — no reserved keywords as identifiers
- [x] `rule_naming_index_convention` — index naming convention
- [x] `rule_naming_foreign_key_convention` — foreign key naming convention
- [x] `rule_naming_unique_key_convention` — unique key naming convention

---

## Phase 7: Statement Quality Rules (18 rules)

### 7.1 WHERE Clause Rules

- [x] `rule_stmt_where_requirement_for_select` — SELECT requires WHERE
- [x] `rule_stmt_where_requirement_for_update_delete` — UPDATE/DELETE requires WHERE
- [x] `rule_stmt_no_leading_wildcard_like` — no leading wildcard in LIKE
- [x] `rule_statement_where_no_equal_null` — no `= NULL` (use IS NULL)
- [x] `rule_statement_where_disallow_using_function` — no functions in WHERE
- [x] `rule_statement_where_maximum_logical_operator_count` — max logical operators in WHERE

### 7.2 Statement Structure Rules

- [x] `rule_stmt_no_select_all` — (already migrated in Phase 1, skip)
- [x] `rule_stmt_disallow_limit` — disallow LIMIT in DML
- [x] `rule_stmt_disallow_order_by` — disallow ORDER BY in DML
- [x] `rule_stmt_disallow_commit` — disallow COMMIT statement
- [x] `rule_statement_merge_alter_table` — merge multiple ALTER TABLE
- [x] `rule_statement_maximum_limit_value` — max LIMIT value
- [x] `rule_statement_maximum_join_table_count` — max JOIN table count
- [x] `rule_statement_maximum_statements_in_transaction` — max statements in transaction

### 7.3 Performance & Execution Rules

- [x] `rule_stmt_max_execution_time` — max execution time hint
- [x] `rule_stmt_require_algorithm_or_lock_options` — require ALGORITHM/LOCK in ALTER
- [x] `rule_statement_add_column_without_position` — ADD COLUMN without FIRST/AFTER
- [x] `rule_statement_join_strict_column_attrs` — strict JOIN column attribute matching

---

## Phase 8: DML Rules (8 rules)

### 8.1 INSERT Rules

- [x] `rule_insert_must_specify_column` — INSERT must specify columns
- [x] `rule_insert_row_limit` — INSERT row count limit
- [x] `rule_insert_disallow_order_by_rand` — no ORDER BY RAND() in INSERT

### 8.2 DML Safety Rules

- [x] `rule_statement_affected_row_limit` — affected row limit
- [x] `rule_statement_dml_dry_run` — DML dry run check
- [x] `rule_statement_select_full_table_scan` — detect full table scans
- [x] `rule_statement_disallow_using_filesort` — detect filesort usage
- [x] `rule_statement_disallow_using_temporary` — detect temporary table usage

---

## Phase 9: System Rule & Cleanup

### 9.1 System Rule

- [x] `rule_builtin_prior_backup_check` migrated — backup check before DDL
- [x] `rule_statement_query_minimum_plan_level` migrated — query plan level check

### 9.2 ANTLR Removal

- [ ] Zero `GetANTLRAST` calls remain in `backend/plugin/advisor/mysql/`
- [ ] `GenericChecker` ANTLR dispatcher removed or deprecated
- [ ] `AsANTLRAST()` fallback removed from `OmniAST`
- [ ] `parseSingleStatementLenient()` removed
- [ ] `ParseMySQL()` callers migrated or removed
- [ ] Build passes with no ANTLR imports in MySQL advisor package
- [x] All tests pass: advisor, schema, parser, integration

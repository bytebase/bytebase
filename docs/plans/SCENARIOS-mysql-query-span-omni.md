# MySQL Query Span Omni Scenarios

> Goal: Prove the omni MySQL query-span extractor preserves legacy MySQL query-span behavior without runtime fallback.
> Verification: each scenario must have an automated assertion against legacy-compatible query-span output, query type, access-table set, or error semantics.
> Reference sources: legacy ANTLR MySQL query-span extractor, legacy MySQL query type listener, current YAML query-span fixtures, omni MySQL AST node definitions, MySQL query-span migration plan.

Status: [ ] pending, [x] passing, [~] partial.

---

## Phase 1: Statement Type And Root Selection

### 1.1 Select-Family Roots

- [x] Plain `SELECT` returns `Select` and extracts target-list results.
- [x] `TABLE t` returns `Select` and expands to the referenced table columns.
- [x] `VALUES ROW(...)` returns `Select` and derives result columns from the first row.
- [ ] `VALUES ROW(DEFAULT)` returns a constant `DEFAULT` result with empty source columns.
- [ ] `SELECT ... INTO` preserves the legacy unsupported/error behavior.
- [x] `EXPLAIN SELECT ...` returns `Explain` without result columns.
- [x] `EXPLAIN ANALYZE SELECT ...` returns `Select` and preserves select access tables.
- [ ] `EXPLAIN ANALYZE TABLE t` returns `Select` and preserves table access.
- [ ] `EXPLAIN ANALYZE VALUES ROW(...)` returns `Select` and preserves values results.
- [ ] `EXPLAIN ANALYZE INSERT ...` returns `DML` and no result columns.
- [ ] Multiple statements in one query return the legacy single-statement error.

### 1.2 Query Type Buckets

- [x] `SHOW ...` returns `SelectInfoSchema`.
- [x] non-password `SET ...` returns `Select`.
- [ ] password-changing `SET PASSWORD ...` preserves legacy query type behavior.
- [x] `CREATE TABLE ...` returns `DDL`.
- [ ] `CREATE DATABASE ...` returns `DDL`.
- [ ] `CREATE VIEW ...` returns `DDL`.
- [ ] `ALTER TABLE ...` returns `DDL`.
- [ ] `DROP TABLE ...` returns `DDL`.
- [ ] `RENAME TABLE ...` returns `DDL`.
- [ ] `TRUNCATE TABLE ...` returns `DDL`.
- [ ] `IMPORT TABLE ...` returns `DDL`.
- [x] `INSERT ...` returns `DML`.
- [ ] `REPLACE ...` returns `DML`.
- [ ] `UPDATE ...` returns `DML`.
- [ ] `DELETE ...` returns `DML`.
- [ ] `LOAD DATA ...` returns `DML`.
- [x] `CALL ...` returns `DML`.
- [x] `DO ...` returns `DML`.
- [x] `HANDLER ... OPEN` returns `DML`.
- [x] `HANDLER ... READ` returns `DML`.
- [x] `HANDLER ... CLOSE` returns `DML`.
- [ ] transaction and locking statements return `DML`.
- [ ] replication statements return `DML`.
- [ ] prepared statements return `DML`.
- [ ] unsupported utility statements fall back to the legacy query type.

## Phase 2: Target List And Expression Lineage

### 2.1 Target List Basics

- [x] Literal target `SELECT 1` returns a literal result with empty source columns.
- [x] Bare column target returns that column as a plain field.
- [x] qualified column target resolves table and database qualifiers.
- [x] target alias overrides expression-derived result name.
- [x] unaliased expression uses the legacy expression text as result name.
- [x] `SELECT * FROM t` expands all source columns in table order.
- [x] `SELECT t.* FROM t` expands only the qualified table columns.
- [x] `SELECT *, a FROM t` preserves star expansion plus explicit target.
- [ ] duplicate output names preserve legacy ordering and duplicate result entries.
- [ ] reserved-word quoted identifiers resolve like legacy identifiers.

### 2.2 Expression Node Coverage

- [x] binary arithmetic expression merges both operand source columns.
- [x] binary comparison expression merges both operand source columns.
- [ ] JSON extraction binary operators merge object and path expression sources.
- [x] unary expression preserves operand source columns and is not a plain field.
- [x] function call merges argument source columns.
- [x] aggregate function merges argument source columns.
- [x] window function merges argument, partition, and order source columns.
- [ ] function separator expression contributes lineage where applicable.
- [x] scalar subquery result merges selected result source columns.
- [x] `EXISTS (SELECT ...)` returns subquery result lineage and is not a plain field.
- [x] `CASE` expression merges operand, condition, result, and default sources.
- [x] `BETWEEN` expression merges expression, lower, and upper sources.
- [x] `IN` value-list expression merges left expression and list item sources.
- [x] `IN (SELECT ...)` merges left expression and subquery result sources.
- [x] `LIKE` expression merges expression, pattern, and escape sources.
- [x] `IS NULL` and related `IS` tests preserve expression sources.
- [x] `CAST` expression preserves inner expression lineage.
- [x] `EXTRACT` expression preserves inner expression lineage.
- [x] `INTERVAL` expression preserves value expression lineage.
- [x] `COLLATE` expression preserves inner expression lineage.
- [x] `MATCH ... AGAINST ...` merges matched columns and search expression sources.
- [x] `CONVERT(expr USING charset)` preserves inner expression lineage.
- [x] `DEFAULT` expression returns empty source columns.
- [x] `ROW(...)` expression merges item sources.
- [x] `MEMBER OF` expression merges value and JSON-array sources.
- [ ] user variable reference preserves legacy empty-lineage behavior.
- [ ] system variable reference preserves legacy empty-lineage behavior.
- [ ] unknown future expression nodes fail closed or get explicit coverage before release.

## Phase 3: FROM And Table Sources

### 3.1 Tables, Aliases, And Stars

- [x] single table source resolves physical table columns.
- [x] table alias changes the visible table name for column resolution.
- [ ] database-qualified table source resolves against the qualified database.
- [ ] cluster-qualified StarRocks table source filters cluster name as before.
- [ ] parenthesized single table source resolves like the unparenthesized table.
- [ ] comma-separated table sources expose all tables to target resolution.
- [ ] `DUAL` produces no table source and allows literal targets.
- [x] missing table maps to `NotFoundError` fail-open behavior.
- [x] missing column maps to `NotFoundError` fail-open behavior.

### 3.2 Joins

- [x] inner join exposes left and right table columns.
- [x] cross join exposes left and right table columns.
- [ ] straight join exposes left and right table columns.
- [ ] left outer join exposes left and right table columns.
- [ ] right outer join exposes left and right table columns.
- [x] `JOIN ... USING(col)` merges duplicate `USING` columns.
- [x] natural join merges common columns.
- [ ] natural left join merges common columns.
- [ ] natural right join merges common columns.
- [ ] `JOIN ... ON` expression contributes access-table dependencies.
- [ ] nested join tree preserves visible source ordering.
- [ ] parenthesized table reference list behaves like legacy cross-join expansion.

### 3.3 Derived Tables And Table Functions

- [x] derived table body contributes access-table dependencies.
- [x] derived table alias controls visible table qualifier.
- [x] derived table column alias list renames output columns.
- [ ] derived table column alias count mismatch returns legacy error.
- [ ] nested derived tables preserve lineage through each layer.
- [ ] lateral derived table preserves correlation behavior.
- [x] `JSON_TABLE` exposes declared columns as pseudo-table columns.
- [x] `JSON_TABLE` column lineage comes from the JSON document expression.
- [ ] nested `JSON_TABLE` columns flatten in legacy order.
- [ ] `JSON_TABLE` with alias omitted uses the legacy generated table name.

## Phase 4: Subqueries, CTEs, And Set Operations

### 4.1 Subquery Scope

- [x] scalar subquery in target list contributes selected source columns.
- [x] correlated scalar subquery can resolve outer query columns.
- [ ] subquery in `WHERE` contributes access-table dependencies.
- [ ] subquery in `HAVING` contributes access-table dependencies.
- [ ] subquery in `ORDER BY` contributes access-table dependencies.
- [ ] nested subqueries preserve nearest-scope alias shadowing.
- [ ] unqualified inner subquery columns follow legacy resolution order.
- [ ] outer table alias shadowed by inner alias follows legacy resolution.

### 4.2 Common Table Expressions

- [x] non-recursive CTE exposes selected columns.
- [x] non-recursive CTE explicit column list renames output columns.
- [ ] non-recursive CTE column count mismatch returns legacy error.
- [ ] nested CTE references resolve in legacy visibility order.
- [x] recursive CTE merges anchor and recursive branch source columns.
- [ ] recursive CTE explicit column list count mismatch returns legacy error.
- [ ] recursive CTE reaches a stable source-column closure.
- [ ] CTE name shadows physical table name according to legacy behavior.
- [ ] later CTE cannot be referenced by earlier CTE unless legacy allowed it.

### 4.3 Set Operations

- [x] `UNION` merges left and right source columns by position.
- [x] `UNION ALL` merges left and right source columns by position.
- [ ] `INTERSECT` preserves legacy support or error behavior.
- [ ] `EXCEPT` preserves legacy support or error behavior.
- [x] set operation result names come from the left side.
- [ ] set operation column count mismatch returns legacy error.
- [ ] set operation inside derived table preserves derived output lineage.
- [ ] set operation inside CTE preserves CTE output lineage.
- [ ] parenthesized set operation preserves legacy grouping behavior.

## Phase 5: Access Tables, System Tables, And Errors

### 5.1 Access Tables

- [x] top-level `FROM` table appears in `SourceColumns` as table-level access.
- [x] joined tables appear in `SourceColumns` as table-level access.
- [x] derived-table body tables appear in access tables.
- [x] CTE body tables appear in access tables.
- [x] scalar subquery tables appear in access tables.
- [x] correlated subquery inner tables appear in access tables.
- [ ] function arguments containing subqueries appear in access tables.
- [ ] `VALUES` expressions containing subqueries appear in access tables.
- [ ] `CALL` arguments containing subqueries appear in access tables where legacy did.
- [x] `HANDLER` table appears in access tables where legacy did.

### 5.2 System Tables

- [x] all-system select returns `SelectInfoSchema`.
- [x] all-system select suppresses returned access tables.
- [x] mixed user/system table query returns `MixUserSystemTablesError`.
- [ ] uppercase system schema follows case-sensitive legacy behavior.
- [ ] uppercase system schema follows case-insensitive legacy behavior.
- [ ] `information_schema` access behaves like legacy system resource detection.
- [ ] `performance_schema` access behaves like legacy system resource detection.
- [ ] `mysql` schema access behaves like legacy system resource detection.

### 5.3 Error Semantics

- [x] missing database maps to fail-open `NotFoundError`.
- [x] missing table maps to fail-open `NotFoundError`.
- [x] missing column maps to fail-open `NotFoundError`.
- [ ] unsupported select-with-into returns the legacy hard error.
- [ ] unsupported table source returns a hard error instead of silent empty lineage.
- [x] unsupported expression node cannot silently return empty source columns.
- [ ] parser failure returns the parser error and does not fabricate a query span.
- [ ] nil or empty statement list returns the legacy empty select span.

## Phase 6: Metadata, Views, Engines, And Case Sensitivity

### 6.1 Metadata And Views

- [ ] view column lineage is derived from the view definition.
- [ ] view output column names are applied to view-derived results.
- [ ] view recursion through nested views preserves source table lineage.
- [ ] missing view dependency maps to legacy `NotFoundError` behavior.
- [ ] duplicate database names with case differences follow `ignoreCaseSensitive`.
- [ ] default database is used when table reference omits database.
- [ ] explicit database overrides default database.

### 6.2 Case Sensitivity

- [x] case-sensitive column lookup preserves exact table/database matching rules.
- [x] case-insensitive column lookup resolves case variants.
- [ ] table alias matching follows legacy case-sensitivity behavior.
- [ ] CTE name matching follows legacy case-sensitivity behavior.
- [ ] derived table alias matching follows legacy case-sensitivity behavior.
- [ ] quoted identifier case is preserved in result names where legacy preserves it.

### 6.3 Engine-Specific Behavior

- [x] current StarRocks fixtures keep matching golden output.
- [ ] StarRocks cluster-qualified database names are normalized before metadata lookup.
- [ ] StarRocks system/user table mixing follows legacy behavior.
- [ ] MySQL and StarRocks use the same lineage rules unless a fixture proves otherwise.


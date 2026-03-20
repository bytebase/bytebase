# PG Completion Scenarios

> Goal: Comprehensive test coverage for PostgreSQL autocompletion across all SQL contexts
> Verification: `go test -v -count=1 github.com/bytebase/bytebase/backend/plugin/parser/pg -run ^TestCompletion$`
> Reference sources: PostgreSQL documentation, omni parser grammar rules, completion.go source code

Status: [ ] pending, [x] passing, [~] partial (needs upstream change)

---

## Phase 1: Foundation Gaps

Fill missing coverage for features already partially tested.

### 1.1 Join Variants

- [x] `SELECT | FROM t1 RIGHT JOIN t2 ON t1.c1 = t2.c1` — columns from both tables visible in RIGHT JOIN
- [x] `SELECT | FROM t1 FULL OUTER JOIN t2 ON t1.c1 = t2.c1` — columns from both tables visible in FULL OUTER JOIN
- [x] `SELECT | FROM t1 CROSS JOIN t2` — columns from both tables visible in CROSS JOIN (no ON clause)
- [x] `SELECT | FROM t1 NATURAL JOIN t2` — columns from both tables visible in NATURAL JOIN
- [x] `SELECT * FROM t1 RIGHT JOIN t2 ON t1.c1 = t2.| ` — qualified column in RIGHT JOIN ON condition
- [x] `SELECT * FROM t1 FULL OUTER JOIN t2 ON t1.c1 = t2.|` — qualified column in FULL OUTER JOIN ON condition
- [x] `SELECT * FROM t1 CROSS JOIN t2 WHERE t1.|` — qualified column in WHERE after CROSS JOIN
- [~] `SELECT * FROM t1 JOIN t2 USING (|)` — columns common to both tables in USING clause (no candidates returned; USING clause not supported by completion engine)
- [x] `SELECT | FROM t1 JOIN t2 ON t1.c1 = t2.c1 LEFT JOIN test.auto ON t2.c1 = test.auto.id` — mixed join types with cross-schema
- [x] `SELECT a.| FROM t1 a NATURAL JOIN t2 b` — alias-qualified columns after NATURAL JOIN

### 1.2 Advanced WHERE Expressions

- [x] `SELECT * FROM t2 WHERE c1 BETWEEN | AND c2` — completion after BETWEEN
- [x] `SELECT * FROM t2 WHERE c1 BETWEEN c2 AND |` — completion after AND in BETWEEN
- [x] `SELECT * FROM test.auto WHERE name LIKE |` — completion after LIKE
- [x] `SELECT * FROM t2 WHERE c1 IS NOT NULL AND |` — completion after IS NOT NULL condition
- [x] `SELECT * FROM t2 WHERE c1 IS NULL OR |` — completion after IS NULL condition
- [x] `SELECT * FROM t2 WHERE NOT |` — completion after NOT
- [x] `SELECT * FROM t2 WHERE c1 > | ` — completion after comparison operator
- [x] `SELECT * FROM t2 WHERE c1 + | > 0` — completion in arithmetic expression
- [x] `SELECT * FROM t2 WHERE c1 IN (1, 2) AND |` — completion after IN list
- [x] `SELECT * FROM t1 WHERE c1 = (SELECT MAX(|) FROM t2)` — completion inside aggregate in scalar subquery
- [x] `SELECT * FROM t1 x WHERE x.c1 IN (SELECT c1 FROM t2 WHERE |)` — completion in correlated subquery WHERE
- [x] `SELECT * FROM t2 WHERE CASE WHEN c1 > 0 THEN | END = 1` — completion in CASE WHEN THEN

### 1.3 Advanced SELECT Features

- [x] `SELECT DISTINCT | FROM t2` — columns after DISTINCT
- [x] `SELECT DISTINCT ON (|) c1, c2 FROM t2` — columns in DISTINCT ON
- [x] `SELECT | FROM t2 LIMIT 10` — columns in SELECT with LIMIT present
- [x] `SELECT * FROM t2 LIMIT |` — completion in LIMIT expression
- [x] `SELECT * FROM t2 OFFSET |` — completion in OFFSET expression
- [x] `SELECT * FROM t2 ORDER BY c1 LIMIT |` — completion in LIMIT after ORDER BY
- [x] `SELECT c1, | FROM t2` — completion for second column in select list
- [x] `DELETE FROM t2 WHERE c1 > 0 RETURNING |` — completion in RETURNING clause (DML)
- [x] `UPDATE t2 SET c1 = 1 RETURNING |` — completion in UPDATE RETURNING clause
- [x] `SELECT c1, SUM(c2) OVER (PARTITION BY |) FROM t2` — column in window PARTITION BY
- [x] `SELECT c1, SUM(c2) OVER (ORDER BY |) FROM t2` — column in window ORDER BY
- [x] `SELECT c1, ROW_NUMBER() OVER (PARTITION BY | ORDER BY c1) FROM t2` — column in window function
- [x] `SELECT COALESCE(|, 0) FROM t2` — completion in COALESCE first argument
- [x] `SELECT COALESCE(c1, |) FROM t2` — completion in COALESCE second argument
- [x] `SELECT NULLIF(|, 0) FROM t2` — completion in NULLIF
- [x] `SELECT CAST(| AS int) FROM t2` — completion in CAST expression
- [x] `SELECT | FROM t2 FOR UPDATE` — columns with FOR UPDATE locking clause

### 1.4 Object Type Coverage

- [x] `SELECT * FROM |` with foreign tables in metadata — foreign tables appear in FROM completion
- [x] `SELECT * FROM |` with materialized views in metadata — materialized views appear in FROM completion
- [x] `SELECT * FROM public.|` with foreign tables — foreign tables in schema-qualified FROM
- [x] `SELECT * FROM public.|` with materialized views — materialized views in schema-qualified FROM
- [x] `SELECT mv1.| FROM mv1` — columns from materialized view (qualified)
- [x] `SELECT | FROM mv1` — columns from materialized view (unqualified)
- [x] `SELECT ft1.| FROM ft1` — columns from foreign table (qualified)
- [x] `SELECT | FROM ft1` — columns from foreign table (unqualified)
- [x] `INSERT INTO |` with foreign tables — foreign tables in INSERT target
- [~] `UPDATE |` with materialized views — materialized views should NOT appear in UPDATE target (they are read-only; completion engine uses same relation_expr rule for SELECT/UPDATE/DELETE and doesn't distinguish context)

---

## Phase 2: DDL Contexts

Completion in DDL statements. Independent of Phase 1 — can run in parallel. Many scenarios likely need `completion.go` changes or omni parser grammar support; mark `[~]` if blocked.

### 2.1 CREATE Statements

- [x] `CREATE TABLE test_tbl (id int, FOREIGN KEY (id) REFERENCES |)` — table completion in FK reference
- [x] `CREATE TABLE test_tbl (id int, FOREIGN KEY (id) REFERENCES public.|)` — schema-qualified FK reference
- [x] `CREATE TABLE test_tbl (id int REFERENCES |)` — inline FK reference
- [~] `CREATE TABLE test_tbl (id int REFERENCES t1(|))` — column completion in FK reference (no candidates returned; parser does not emit columnref rule inside FK column parentheses)
- [x] `CREATE INDEX idx ON |` — table completion in CREATE INDEX
- [x] `CREATE INDEX idx ON public.|` — schema-qualified table in CREATE INDEX
- [~] `CREATE INDEX idx ON t1 (|)` — column completion in index expression (no candidates returned; parser does not emit columnref rule inside index column parentheses)
- [~] `CREATE INDEX idx ON t2 (c1, |)` — second column in composite index (no candidates returned; same parser limitation as above)
- [x] `CREATE VIEW v2 AS SELECT | FROM t1` — completion in CREATE VIEW body
- [x] `CREATE VIEW v2 AS SELECT * FROM |` — FROM completion in CREATE VIEW body
- [x] `CREATE MATERIALIZED VIEW mv AS SELECT | FROM t1` — completion in CREATE MATERIALIZED VIEW body
- [x] `CREATE TABLE test_tbl AS SELECT | FROM t1` — completion in CREATE TABLE AS SELECT
- [x] `CREATE TABLE test_tbl (LIKE |)` — table completion in LIKE clause
- [x] `CREATE TRIGGER trig AFTER INSERT ON |` — table completion in CREATE TRIGGER
- [x] `CREATE TRIGGER trig AFTER INSERT ON public.|` — schema-qualified table in CREATE TRIGGER

### 2.2 ALTER Statements

- [x] `ALTER TABLE |` — table completion in ALTER TABLE
- [x] `ALTER TABLE public.|` — schema-qualified table in ALTER TABLE
- [x] `ALTER TABLE t1 ADD COLUMN c2 int REFERENCES |` — FK reference in ALTER ADD COLUMN
- [~] `ALTER TABLE t1 DROP COLUMN |` — column completion for existing columns (no candidates returned; parser does not emit columnref rule in ALTER TABLE DROP COLUMN context)
- [~] `ALTER TABLE t1 RENAME COLUMN | TO new_name` — column completion in RENAME (no candidates returned; parser does not emit columnref rule in ALTER TABLE RENAME COLUMN context)
- [~] `ALTER TABLE t1 ALTER COLUMN |` — column completion in ALTER COLUMN (no candidates returned; parser does not emit columnref rule in ALTER TABLE ALTER COLUMN context)
- [~] `ALTER TABLE t1 ALTER COLUMN | SET NOT NULL` — column completion with SET NOT NULL (no candidates returned; parser does not emit columnref rule in ALTER TABLE ALTER COLUMN context)
- [~] `ALTER TABLE t1 ADD CONSTRAINT fk FOREIGN KEY (|) REFERENCES t2` — column in FK constraint (returns relation_expr candidates instead of columns; parser does not emit columnref rule inside FK column parentheses)
- [~] `ALTER TABLE t1 ADD CONSTRAINT fk FOREIGN KEY (c1) REFERENCES t2(|)` — referenced column (no candidates returned; parser does not emit columnref rule inside FK referenced column parentheses)
- [x] `ALTER INDEX | RENAME TO new_name` — index completion in ALTER INDEX
- [x] `ALTER VIEW |` — view completion in ALTER VIEW
- [x] `ALTER VIEW public.|` — schema-qualified view in ALTER VIEW
- [x] `ALTER SEQUENCE |` — sequence completion in ALTER SEQUENCE
- [x] `ALTER SEQUENCE public.|` — schema-qualified sequence
- [x] `ALTER MATERIALIZED VIEW |` — materialized view completion

### 2.3 DROP and TRUNCATE Statements

- [x] `DROP TABLE |` — table completion in DROP TABLE
- [x] `DROP TABLE public.|` — schema-qualified table in DROP
- [x] `DROP TABLE IF EXISTS |` — table completion after IF EXISTS
- [x] `DROP VIEW |` — view completion in DROP VIEW
- [x] `DROP VIEW public.|` — schema-qualified view in DROP VIEW
- [x] `DROP INDEX |` — index completion in DROP INDEX
- [x] `DROP SEQUENCE |` — sequence completion in DROP SEQUENCE
- [x] `DROP MATERIALIZED VIEW |` — materialized view in DROP
- [x] `TRUNCATE |` — table completion in TRUNCATE
- [x] `TRUNCATE public.|` — schema-qualified table in TRUNCATE

### 2.4 COMMENT and GRANT Statements

- [x] `COMMENT ON TABLE |` — table completion in COMMENT ON
- [x] `COMMENT ON TABLE public.|` — schema-qualified table in COMMENT ON
- [~] `COMMENT ON COLUMN t1.|` — column completion in COMMENT ON COLUMN (no candidates returned; parser does not emit columnref rule in COMMENT ON COLUMN context)
- [~] `COMMENT ON COLUMN public.t1.|` — schema-qualified column in COMMENT ON COLUMN (no candidates returned; parser does not emit columnref rule in COMMENT ON COLUMN context)
- [x] `GRANT SELECT ON |` — table completion in GRANT
- [x] `GRANT SELECT ON public.|` — schema-qualified table in GRANT
- [~] `GRANT ALL ON ALL TABLES IN SCHEMA |` — schema completion in GRANT (no candidates returned; parser does not emit schema_name rule in GRANT ... IN SCHEMA context)
- [x] `REVOKE SELECT ON |` — table completion in REVOKE
- [x] `REVOKE SELECT ON public.|` — schema-qualified table in REVOKE
- [x] `GRANT USAGE ON SEQUENCE |` — sequence completion in GRANT

---

## Phase 3: Advanced Query Patterns

Complex query structures and nesting. Independent of Phase 2. Sections 3.1 and 3.2 may need `completion.go` scope-tracking changes.

### 3.1 Recursive CTEs and CTEs in DML

- [~] `WITH RECURSIVE x AS (SELECT c1 FROM t1 UNION ALL SELECT | FROM x JOIN t1 ON x.c1 = t1.c1) SELECT * FROM x` — completion in recursive branch (only t1 columns resolve; recursive self-reference x does not contribute its own columns)
- [~] `WITH RECURSIVE x AS (SELECT c1 FROM t1 UNION ALL SELECT x.| FROM x JOIN t1 ON x.c1 = t1.c1) SELECT * FROM x` — qualified columns from recursive CTE reference (no candidates returned; recursive self-reference columns not resolvable)
- [x] `WITH x AS (SELECT * FROM t2) INSERT INTO t1 SELECT | FROM x` — CTE used in INSERT...SELECT
- [x] `WITH x AS (SELECT c1, c2 FROM t2) UPDATE t1 SET c1 = (SELECT | FROM x)` — CTE in UPDATE scalar subquery
- [x] `WITH x AS (SELECT * FROM t2) DELETE FROM t1 WHERE c1 IN (SELECT | FROM x)` — CTE in DELETE subquery
- [~] `WITH a AS (SELECT c1 FROM t1), b AS (SELECT * FROM a) SELECT | FROM b` — chained CTEs (CTE referencing another CTE) (b's columns not resolved; query span cannot resolve SELECT * FROM a where a is a CTE)
- [x] `WITH a AS (SELECT c1 FROM t1), b AS (SELECT a.| FROM a) SELECT * FROM b` — qualified column in chained CTE
- [x] `WITH x AS (SELECT c1 FROM t1) SELECT | FROM x, t2` — CTE mixed with regular table in FROM
- [x] `WITH x AS (SELECT c1 FROM t1) SELECT x.| FROM x JOIN t2 ON x.c1 = t2.c1` — CTE qualified columns in JOIN
- [x] `WITH x(a, b) AS (SELECT c1, c2 FROM t2) SELECT | FROM x JOIN t1 ON x.a = t1.c1` — named CTE columns in JOIN

### 3.2 LATERAL Joins and Table Functions

- [ ] `SELECT * FROM t1, LATERAL (SELECT | FROM t2 WHERE t2.c1 = t1.c1) sub` — LATERAL subquery referencing outer table
- [ ] `SELECT sub.| FROM t1, LATERAL (SELECT c1, c2 FROM t2 WHERE t2.c1 = t1.c1) sub` — columns from LATERAL subquery alias
- [ ] `SELECT | FROM t1 LEFT JOIN LATERAL (SELECT * FROM t2 WHERE t2.c1 = t1.c1) sub ON true` — LATERAL with LEFT JOIN
- [ ] `SELECT * FROM t1, LATERAL (SELECT t1.| FROM t2) sub` — outer table column access in LATERAL
- [ ] `SELECT * FROM t2 x, LATERAL (SELECT x.| FROM t1) sub` — alias-qualified outer reference in LATERAL
- [ ] `SELECT | FROM generate_series(1, 10) g` — table function in FROM clause
- [ ] `SELECT g.| FROM generate_series(1, 10) g` — qualified column from table function alias
- [ ] `SELECT | FROM t1, generate_series(1, 10) g` — table function with regular table
- [ ] `SELECT | FROM t2 x, LATERAL (SELECT * FROM t1 WHERE t1.c1 = x.c1) sub` — complex LATERAL with alias

### 3.3 Nested Subqueries and Expressions

- [ ] `SELECT (SELECT | FROM t1) FROM t2` — scalar subquery in SELECT list
- [ ] `SELECT (SELECT t1.| FROM t1) FROM t2` — qualified column in scalar subquery
- [ ] `SELECT * FROM t1 WHERE c1 = ANY(SELECT | FROM t2)` — ANY subquery
- [ ] `SELECT * FROM t1 WHERE c1 = ALL(SELECT | FROM t2)` — ALL subquery
- [ ] `SELECT * FROM t1 WHERE c1 > (SELECT MAX(|) FROM t2)` — aggregate in scalar subquery
- [ ] `SELECT * FROM (SELECT * FROM (SELECT | FROM t1) a) b` — triple-nested subquery
- [ ] `SELECT b.| FROM (SELECT * FROM (SELECT c1 FROM t1) a) b` — qualified column from triple-nested
- [ ] `SELECT * FROM t1 WHERE EXISTS (SELECT 1 FROM t2 WHERE t2.c1 = t1.|)` — correlated subquery referencing outer column
- [ ] `SELECT | FROM (SELECT c1, c2 FROM t2 UNION SELECT c1, c2 FROM t2) sub` — subquery with UNION
- [ ] `SELECT sub.| FROM (SELECT c1, c2 FROM t2 UNION SELECT c1, c2 FROM t2) sub` — qualified columns from UNION subquery

### 3.4 Set Operations

- [ ] `SELECT c1 FROM t1 INTERSECT SELECT | FROM t2` — SELECT list in INTERSECT
- [ ] `SELECT c1 FROM t1 EXCEPT SELECT | FROM t2` — SELECT list in EXCEPT
- [ ] `SELECT c1 FROM t1 UNION SELECT c1 FROM t1 UNION SELECT | FROM t2` — triple UNION
- [ ] `(SELECT c1 FROM t1) UNION (SELECT | FROM t2)` — parenthesized set operation
- [ ] `SELECT c1 FROM t1 UNION ALL SELECT c1 FROM t2 ORDER BY |` — ORDER BY on UNION result
- [ ] `SELECT * FROM (SELECT c1 FROM t1 UNION SELECT c1 FROM t2) x WHERE x.|` — WHERE on UNION subquery
- [ ] `WITH x AS (SELECT c1 FROM t1 UNION SELECT c1 FROM t2) SELECT x.| FROM x` — CTE with UNION body
- [ ] `SELECT c1 FROM t1 EXCEPT ALL SELECT | FROM t2` — EXCEPT ALL variant
- [ ] `SELECT c1 FROM t1 INTERSECT ALL SELECT | FROM t2` — INTERSECT ALL variant
- [ ] `SELECT | FROM t2 UNION SELECT * FROM t1` — first branch of UNION

---

## Phase 4: Edge Cases & Robustness

Identifier handling, error recovery, and special scenarios.

### 4.1 Quoted Identifiers and Reserved Keywords

- [ ] `SELECT | FROM "t1"` — quoted table name in FROM (should resolve t1 columns)
- [ ] `SELECT "t1".| FROM t1` — quoted table qualifier
- [ ] `SELECT * FROM "public".|` — quoted schema name
- [ ] `SELECT * FROM |` with table named using reserved keyword (metadata has table named "order") — reserved keyword table appears quoted
- [ ] `SELECT | FROM "order"` — columns from reserved-keyword-named table
- [ ] `SELECT "order".| FROM "order"` — qualified columns from reserved-keyword table
- [ ] `SELECT * FROM |` with mixed-case table name (metadata has "MyTable") — mixed-case name appears quoted
- [ ] `SELECT | FROM "MyTable"` — columns from mixed-case table
- [ ] `SELECT * FROM |` with table containing special chars (metadata has "my-table") — special char name appears quoted
- [ ] `SELECT * FROM public."t1" WHERE |` — columns after quoted schema-qualified table

### 4.2 Partial Prefix Completion

- [ ] `SELECT * FROM t|` — partial table name prefix "t" matches t1, t2
- [ ] `SELECT * FROM public.t|` — partial schema-qualified prefix
- [ ] `SELECT t1.c|` — partial column name prefix
- [ ] `SELECT * FROM test.a|` — partial prefix in non-default schema
- [ ] `SELECT * FROM tes|` — partial schema name prefix
- [ ] `SELECT | FROM t1 WHERE t1.c|` — partial column in WHERE
- [ ] `INSERT INTO t|` — partial table name in INSERT
- [ ] `UPDATE t|` — partial table name in UPDATE
- [ ] `DELETE FROM t|` — partial table name in DELETE
- [ ] `SELECT * FROM t1 JOIN t|` — partial table name in JOIN

### 4.3 Multi-Statement and Error Recovery

- [ ] `SELECT 1; INSERT INTO |` — completion in second statement (INSERT)
- [ ] `SELECT 1; UPDATE |` — completion in second statement (UPDATE)
- [ ] `SELECT 1; DELETE FROM |` — completion in second statement (DELETE)
- [ ] `INVALID SQL; SELECT * FROM |` — recovery after invalid first statement
- [ ] `SELECT * FROM t1; SELECT * FROM t2 WHERE |` — columns from correct table in second statement
- [ ] `SELECT; SELECT | FROM t1` — recovery after incomplete SELECT
- [ ] `SELECT * FROM t1 WHERE; SELECT | FROM t2` — recovery after incomplete WHERE
- [ ] `CREATE TABLE x (id int); SELECT | FROM t1` — DDL then DML multi-statement

### 4.4 Whitespace and Formatting Variations

- [ ] Multi-line SELECT: `SELECT\n  |\nFROM t1` — completion works across line breaks
- [ ] Multi-line FROM: `SELECT *\nFROM\n  |` — FROM on separate line
- [ ] Tab-indented: `SELECT\t|\tFROM t1` — tab characters in SQL
- [ ] Extra whitespace: `SELECT   *   FROM   |` — multiple spaces between tokens
- [ ] Trailing whitespace: `SELECT * FROM t1 WHERE | ` — space after cursor
- [ ] Comment before cursor: `SELECT * FROM /* comment */ |` — completion after block comment
- [ ] Line comment: `SELECT * FROM t1 -- comment\nWHERE |` — completion after line comment

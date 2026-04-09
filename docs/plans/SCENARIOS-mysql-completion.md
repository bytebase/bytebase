# MySQL Completion Test Coverage Alignment

> Goal: Align MySQL completion test coverage with PG's 235 test cases
> Verification: `go test -count=1 ./backend/plugin/parser/mysql/ -run ^TestCompletion$`
> Reference: PG completion tests at `backend/plugin/parser/pg/test-data/test_completion.yaml`
> Test file: `backend/plugin/parser/mysql/test-data/test_completion.yaml`
> Test runner: `backend/plugin/parser/mysql/completion_test.go`

Status: [ ] pending, [x] passing, [~] partial

---

## Phase 1: Core SQL Completion

### 1.1 Basic SELECT & FROM (14 scenarios)

- [x] `SELECT * FROM |` — tables, databases, views
- [x] `SELECT * FROM db.|` — tables and views in specific database
- [x] `SELECT | FROM t1` — columns of t1, plus tables/databases/views
- [x] `SELECT | FROM t2 x` — columns of t2 (aliased as x), tables including alias
- [ ] `SELECT DISTINCT | FROM t2` — same as SELECT columns
- [ ] `SELECT | FROM t2 LIMIT 10` — columns despite trailing LIMIT
- [ ] `SELECT c1, | FROM t2` — columns after comma in select list
- [x] `SELECT * FROM t1, |` — second table after comma
- [ ] `SELECT\n  |\nFROM t1` — multiline with cursor on empty line before FROM
- [ ] `SELECT *\nFROM\n  |` — multiline FROM on separate line
- [ ] `SELECT   *   FROM   |` — extra whitespace
- [ ] `SELECT * FROM /* comment */ |` — inline block comment before cursor
- [ ] `SELECT * FROM t1 -- comment\nWHERE |` — line comment before WHERE
- [ ] `SELECT | FROM t2 FOR UPDATE` — columns despite trailing FOR UPDATE

### 1.2 Table.Column Qualification (10 scenarios)

- [ ] `SELECT t1.| FROM t1` — columns of t1 via direct table name
- [ ] `SELECT cc1.| FROM t2 cc1` — columns of t2 via alias
- [ ] `SELECT cc1.| FROM t1 cc1 JOIN t2 ON NOT cc1.c1 = t2.c1` — alias columns in JOIN context
- [ ] `SELECT db.t2.| FROM t2` — fully qualified database.table.column
- [x] `SELECT * FROM t2 x ORDER BY x.|` — alias columns in ORDER BY
- [x] `SELECT * FROM t2 x GROUP BY x.|` — alias columns in GROUP BY
- [x] `SELECT MAX(cc1.|) FROM t2 cc1` — alias columns inside function call
- [ ] `SELECT a.| FROM t1 a JOIN t1 b ON a.c1 = b.c1` — disambiguate self-join aliases (a)
- [ ] `SELECT b.| FROM t1 a JOIN t1 b ON a.c1 = b.c1` — disambiguate self-join aliases (b)
- [ ] `SELECT COUNT(|) FROM t1` — columns inside aggregate function

### 1.3 JOIN Variants (18 scenarios)

- [ ] `SELECT | FROM t1 JOIN t2 ON t1.c1 = t2.c1` — columns from both joined tables
- [x] `SELECT * FROM t1 cc1 JOIN t2 ON cc1.|` — alias columns in ON clause (left)
- [ ] `SELECT * FROM t1 cc1 JOIN t2 ON cc1.c1 = t2.|` — table columns in ON clause (right)
- [ ] `SELECT | FROM t1 LEFT JOIN t2 ON t1.c1 = t2.c1` — LEFT JOIN
- [ ] `SELECT | FROM t1 RIGHT JOIN t2 ON t1.c1 = t2.c1` — RIGHT JOIN
- [ ] `SELECT | FROM t1 CROSS JOIN t2` — CROSS JOIN (no ON)
- [ ] `SELECT | FROM t1 NATURAL JOIN t2` — NATURAL JOIN
- [ ] `SELECT | FROM t1 a JOIN t2 b ON a.c1 = b.c1 JOIN t2 c ON b.c1 = c.c1` — 3-way JOIN
- [ ] `SELECT c.| FROM t1 a JOIN t2 b ON a.c1 = b.c1 JOIN t2 c ON b.c1 = c.c1` — 3rd alias columns
- [ ] `SELECT * FROM t1 RIGHT JOIN t2 ON t1.c1 = t2.|` — columns in RIGHT JOIN ON
- [ ] `SELECT * FROM t1 CROSS JOIN t2 WHERE t1.|` — columns after CROSS JOIN in WHERE
- [ ] `SELECT * FROM t1 JOIN t2 ON t1.c1 = t2.c1 LEFT JOIN t2 t3 ON t2.c1 = t3.c1 WHERE |` — multi-JOIN WHERE
- [ ] `SELECT a.| FROM t1 a NATURAL JOIN t2 b` — alias after NATURAL JOIN
- [x] `SELECT * FROM t1 JOIN |` — table candidates after JOIN keyword
- [ ] `SELECT * FROM t1 LEFT JOIN |` — table candidates after LEFT JOIN
- [x] `SELECT FROM t1 JOIN |` — skeleton: table candidates after JOIN (no select expr)
- [ ] `SELECT | FROM t1 JOIN t2 ON t1.c1 = t2.c1 LEFT JOIN t2 t3 ON t2.c1 = t3.c1` — columns from all 3 tables
- [ ] `SELECT * FROM t1 JOIN t2 USING (|)` — columns for USING clause

### 1.4 Subquery & Derived Tables (14 scenarios)

- [x] `SELECT | FROM (SELECT c1 FROM t1) cc1` — derived table columns (inferred)
- [x] `SELECT | FROM (SELECT c1 FROM t1) cc1(cc1c1)` — derived table columns (explicit alias)
- [ ] `SELECT x.| FROM (SELECT * FROM t2) x` — derived table qualified columns
- [ ] `SELECT | FROM (SELECT * FROM (SELECT c1 FROM t1) inner_q) outer_q` — nested derived tables
- [ ] `SELECT | FROM (SELECT c1 FROM t1) sub1 JOIN t2 ON sub1.c1 = t2.c1` — derived table JOIN
- [ ] `SELECT | FROM (SELECT c1 FROM (SELECT c1 FROM (SELECT c1 FROM t1) a) b) c` — triple nested
- [ ] `SELECT b.| FROM (SELECT * FROM (SELECT c1 FROM t1) a) b` — nested qualified access
- [ ] `(SELECT c1 FROM t1) UNION (SELECT | FROM t2)` — subquery in parenthesized UNION
- [ ] `SELECT * FROM t1 WHERE c1 IN (SELECT | FROM t2)` — subquery in WHERE IN
- [ ] `SELECT * FROM t1 WHERE EXISTS (SELECT | FROM t2)` — subquery in WHERE EXISTS
- [ ] `SELECT * FROM t1 WHERE c1 = (SELECT MAX(|) FROM t2)` — scalar subquery
- [ ] `SELECT * FROM t1 x WHERE x.c1 IN (SELECT c1 FROM t2 WHERE |)` — correlated subquery WHERE
- [ ] `SELECT * FROM t1 WHERE EXISTS (SELECT 1 FROM t2 WHERE t2.c1 = t1.|)` — correlated column ref
- [ ] `SELECT (SELECT | FROM t1) FROM t2` — scalar subquery in SELECT list

### 1.5 CTE (Common Table Expressions) (16 scenarios)

- [x] `WITH x AS (SELECT * FROM t2) SELECT x.| FROM x;` — CTE qualified columns (inferred)
- [x] `WITH x(x1, x2) AS (SELECT * FROM t2) SELECT x.| FROM x;` — CTE with explicit columns
- [x] `WITH x(x1, x2) AS (SELECT * FROM t2) SELECT | FROM x` — CTE unqualified columns
- [ ] `WITH a AS (SELECT c1 FROM t1), b AS (SELECT c1, c2 FROM t2) SELECT | FROM a JOIN b ON a.c1 = b.c1` — multiple CTEs
- [ ] `WITH a AS (SELECT c1 FROM t1), b AS (SELECT c1, c2 FROM t2) SELECT b.| FROM a JOIN b ON a.c1 = b.c1` — multiple CTE qualified
- [ ] `WITH x AS (SELECT c1 FROM t1) SELECT | FROM (SELECT * FROM x) sub1` — CTE used in derived table
- [ ] `WITH a AS (SELECT c1 FROM t1), b AS (SELECT * FROM a) SELECT | FROM b` — CTE referencing CTE
- [ ] `WITH a AS (SELECT c1 FROM t1), b AS (SELECT a.| FROM a) SELECT * FROM b` — CTE qualified in CTE definition
- [ ] `WITH x AS (SELECT c1 FROM t1) SELECT | FROM x, t2` — CTE mixed with regular table
- [ ] `WITH x AS (SELECT c1 FROM t1) SELECT x.| FROM x JOIN t2 ON x.c1 = t2.c1` — CTE in JOIN
- [ ] `WITH x(a, b) AS (SELECT c1, c2 FROM t2) SELECT | FROM x JOIN t1 ON x.a = t1.c1` — CTE explicit cols in JOIN
- [ ] `WITH x AS (SELECT * FROM t2) INSERT INTO t1 SELECT | FROM x` — CTE in INSERT SELECT
- [ ] `WITH x AS (SELECT c1, c2 FROM t2) UPDATE t1 SET c1 = (SELECT | FROM x)` — CTE in UPDATE subquery
- [ ] `WITH x AS (SELECT * FROM t2) DELETE FROM t1 WHERE c1 IN (SELECT | FROM x)` — CTE in DELETE subquery
- [ ] `WITH RECURSIVE x AS (SELECT c1 FROM t1 UNION ALL SELECT c1 FROM x) SELECT | FROM x` — recursive CTE
- [ ] `WITH x AS (SELECT c1 FROM t1 UNION SELECT c1 FROM t2) SELECT x.| FROM x` — CTE with UNION body

### 1.6 WHERE Clause Variants (18 scenarios)

- [x] `SELECT * FROM t1 WHERE |` — basic WHERE
- [ ] `SELECT * FROM t2 WHERE c1 = 1 AND |` — after AND
- [ ] `SELECT * FROM t2 WHERE (c1 = 1 AND |)` — inside parenthesized AND
- [ ] `SELECT * FROM t2 WHERE c1 IS NOT NULL AND |` — after IS NOT NULL AND
- [ ] `SELECT * FROM t2 WHERE c1 IS NULL OR |` — after IS NULL OR
- [ ] `SELECT * FROM t2 WHERE NOT |` — after NOT
- [ ] `SELECT * FROM t2 WHERE c1 > |` — after comparison operator
- [ ] `SELECT * FROM t2 WHERE c1 + | > 0` — in arithmetic expression
- [ ] `SELECT * FROM t2 WHERE c1 IN (1, 2) AND |` — after IN list AND
- [ ] `SELECT * FROM t2 WHERE c1 BETWEEN | AND c2` — BETWEEN left
- [ ] `SELECT * FROM t2 WHERE c1 BETWEEN c2 AND |` — BETWEEN right
- [ ] `SELECT * FROM t2 WHERE c1 LIKE |` — after LIKE
- [ ] `SELECT * FROM t2 WHERE CASE WHEN c1 > 0 THEN | END = 1` — inside CASE THEN
- [ ] `SELECT * FROM t1 a JOIN t2 b ON a.c1 = b.c1 WHERE |` — WHERE after JOIN
- [ ] `SELECT * FROM t1 a JOIN t2 b ON a.c1 = b.c1 WHERE a.|` — qualified column in WHERE after JOIN
- [x] `SELECT FROM t1 WHERE |` — skeleton: WHERE without select expressions
- [ ] `SELECT\nFROM t1\nWHERE |` — skeleton multiline WHERE
- [ ] `SELECT c1 as eid FROM t1 WHERE |` — WHERE with select alias (alias NOT valid in WHERE)

---

## Phase 2: DML & DDL Completion

### 2.1 ORDER BY / GROUP BY / HAVING (10 scenarios)

- [x] `SELECT c1 as eid FROM t1 ORDER BY |` — ORDER BY with alias
- [x] `SELECT c1 as eid, c2 as xid FROM t2 ORDER BY |` — ORDER BY with multiple aliases
- [ ] `SELECT c1 as eid FROM t1 GROUP BY |` — GROUP BY with alias
- [ ] `SELECT c1 as eid, c2 as xid FROM t2 HAVING |` — HAVING with aliases
- [x] `SELECT FROM t1 ORDER BY |` — skeleton ORDER BY
- [x] `SELECT FROM t1 GROUP BY |` — skeleton GROUP BY
- [ ] `SELECT c1 FROM t1 UNION SELECT c1 FROM t2 ORDER BY |` — ORDER BY on UNION result
- [ ] `SELECT c1 FROM t1 UNION ALL SELECT c1 FROM t2 ORDER BY |` — ORDER BY on UNION ALL result
- [ ] `SELECT * FROM t2 LIMIT |` — keywords only in LIMIT position
- [ ] `SELECT * FROM t2 ORDER BY c1 LIMIT |` — LIMIT after ORDER BY

### 2.2 INSERT (8 scenarios)

- [x] `INSERT INTO |` — table candidates
- [ ] `INSERT INTO db.|` — database-qualified table candidates
- [x] `INSERT INTO t1(|);` — column candidates for target table
- [x] `INSERT INTO t2(c1, |);` — remaining columns after comma
- [ ] `INSERT INTO t1 SELECT | FROM t2` — INSERT SELECT columns
- [ ] `INSERT INTO t1 VALUES (|)` — inside VALUES (columns/functions)
- [ ] `SELECT 1; INSERT INTO |` — INSERT after prior statement
- [ ] `UPDATE | SET c1 = 1` — UPDATE with partial context

### 2.3 UPDATE & DELETE (12 scenarios)

- [x] `UPDATE |` — table candidates
- [ ] `UPDATE db.|` — database-qualified table candidates
- [x] `UPDATE t1 SET |` — SET column candidates
- [ ] `UPDATE t1 SET c1 = |` — value position (columns/functions)
- [ ] `UPDATE t1 SET c1 = 1 WHERE |` — UPDATE WHERE
- [x] `DELETE FROM |` — table candidates
- [ ] `DELETE FROM db.|` — database-qualified table candidates
- [ ] `DELETE FROM t1 WHERE |` — DELETE WHERE
- [ ] `SELECT 1; UPDATE |` — UPDATE after prior statement
- [ ] `SELECT 1; DELETE FROM |` — DELETE after prior statement
- [x] `SELECT FROM t1 WHERE |` — skeleton WHERE (already in 1.6, cross-ref)
- [ ] `DELETE FROM t1 WHERE c1 > 0 AND |` — compound DELETE WHERE

### 2.4 ALTER / DROP / CREATE (18 scenarios)

- [x] `ALTER TABLE |` — table candidates
- [ ] `ALTER TABLE db.|` — database-qualified
- [ ] `ALTER TABLE t1 ADD COLUMN c2 int REFERENCES |` — FK reference table
- [ ] `ALTER TABLE t1 DROP COLUMN |` — column candidates
- [ ] `ALTER TABLE t1 RENAME COLUMN | TO new_name` — column candidates
- [ ] `ALTER TABLE t1 MODIFY COLUMN |` — MySQL-specific MODIFY
- [ ] `ALTER TABLE t1 ADD CONSTRAINT fk FOREIGN KEY (|) REFERENCES t2` — FK source columns
- [ ] `ALTER TABLE t1 ADD CONSTRAINT fk FOREIGN KEY (c1) REFERENCES t2(|)` — FK target columns
- [x] `DROP TABLE |` — table candidates
- [ ] `DROP TABLE db.|` — database-qualified
- [ ] `DROP TABLE IF EXISTS |` — table candidates after IF EXISTS
- [ ] `DROP VIEW |` — view candidates
- [ ] `DROP INDEX |` — index candidates (table context)
- [ ] `TRUNCATE TABLE |` — table candidates (MySQL requires TABLE keyword)
- [ ] `CREATE TABLE t3 (id int, FOREIGN KEY (id) REFERENCES |)` — FK reference table
- [ ] `CREATE TABLE t3 (id int REFERENCES t1(|))` — FK reference columns
- [ ] `CREATE INDEX idx ON |` — table candidates
- [ ] `CREATE INDEX idx ON t1 (|)` — column candidates for index
- [ ] `CREATE INDEX idx ON t2 (c1, |)` — second column in compound index
- [ ] `CREATE VIEW v2 AS SELECT | FROM t1` — columns in CREATE VIEW
- [ ] `CREATE VIEW v2 AS SELECT * FROM |` — tables in CREATE VIEW FROM
- [ ] `CREATE TABLE t3 AS SELECT | FROM t1` — CREATE TABLE AS SELECT

---

## Phase 3: Advanced Completion

### 3.1 UNION / INTERSECT / EXCEPT (12 scenarios)

- [ ] `SELECT c1 FROM t1 UNION SELECT | FROM t2` — after UNION SELECT
- [ ] `SELECT c1 FROM t1 UNION ALL SELECT | FROM t2` — after UNION ALL SELECT
- [ ] `SELECT c1 FROM t1 INTERSECT SELECT | FROM t2` — INTERSECT
- [ ] `SELECT c1 FROM t1 EXCEPT SELECT | FROM t2` — EXCEPT
- [ ] `SELECT c1 FROM t1 UNION SELECT c1 FROM t1 UNION SELECT | FROM t2` — triple UNION
- [ ] `SELECT c1 FROM t1 EXCEPT ALL SELECT | FROM t2` — EXCEPT ALL
- [ ] `SELECT c1 FROM t1 INTERSECT ALL SELECT | FROM t2` — INTERSECT ALL
- [ ] `SELECT | FROM t2 UNION SELECT * FROM t1` — first SELECT in UNION
- [ ] `SELECT * FROM (SELECT c1 FROM t1 UNION SELECT c1 FROM t2) x WHERE x.|` — qualified from UNION subquery
- [ ] `WITH x AS (SELECT c1 FROM t1 UNION SELECT c1 FROM t2) SELECT x.| FROM x` — CTE with UNION (dup of 1.5, cross-ref)
- [ ] `SELECT | FROM (SELECT c1, c2 FROM t2 UNION SELECT c1, c2 FROM t2) sub` — derived table from UNION
- [ ] `SELECT c1 FROM t1 UNION ALL SELECT c1 FROM t2 ORDER BY |` — ORDER BY on UNION (dup of 2.1, cross-ref)

### 3.2 Multi-Statement & Error Recovery (14 scenarios)

- [x] `SELECT 1; SELECT * FROM |` — after valid statement (complex variant)
- [x] `SELECT FROM basdkfjasldf; SELECT | FROM t1` — after invalid statement
- [x] `select count(1) from t1 where id 'asdfsadf'; SELECT * FROM |` — after complex invalid (multiline)
- [ ] `INVALID SQL; SELECT * FROM |` — after completely invalid SQL
- [ ] `SELECT * FROM t1; SELECT * FROM t2 WHERE |` — WHERE in second statement
- [ ] `SELECT; SELECT | FROM t1` — after empty SELECT
- [ ] `SELECT * FROM t1 WHERE; SELECT | FROM t2` — after incomplete WHERE
- [ ] `CREATE TABLE x (id int); SELECT | FROM t1` — after DDL
- [x] `SELECT\nFROM |` — skeleton (no select expressions)
- [x] `SELECT\n\nFROM |` — skeleton with blank line
- [x] `SELECT FROM |` — skeleton single line
- [ ] `SELECT\nFROM t1\nWHERE |` — skeleton multiline WHERE (dup of 1.6)
- [ ] `SELECT 1; UPDATE |` — UPDATE after prior statement
- [ ] `SELECT 1; DELETE FROM |` — DELETE after prior statement

### 3.3 Partial Identifiers (10 scenarios)

- [ ] `SELECT * FROM t|` — partial table name
- [ ] `SELECT * FROM db.t|` — partial after database dot
- [ ] `SELECT t1.c|` — partial column name
- [ ] `INSERT INTO t|` — partial in INSERT
- [ ] `UPDATE t|` — partial in UPDATE
- [ ] `DELETE FROM t|` — partial in DELETE
- [ ] `SELECT * FROM t1 JOIN t|` — partial after JOIN
- [ ] `SELECT * FROM t1 WHERE t1.c|` — partial qualified column in WHERE
- [ ] `ALTER TABLE t|` — partial in ALTER
- [ ] `DROP TABLE t|` — partial in DROP

### 3.4 Quoted Identifiers & Formatting (10 scenarios)

- [ ] `` SELECT | FROM `t1` `` — backtick-quoted table
- [ ] `` SELECT `t1`.| FROM t1 `` — backtick-quoted qualifier
- [ ] `` SELECT * FROM `db`.| `` — backtick-quoted database
- [ ] `` SELECT * FROM `db`.`t1` WHERE | `` — fully backtick-quoted table in WHERE
- [ ] `SELECT\t|\tFROM t1` — tab whitespace
- [ ] `SELECT * FROM t1 WHERE | ` — trailing space after cursor
- [ ] `SELECT\n  *\n  FROM\n  t1\n  WHERE |` — fully expanded multiline
- [ ] `/* leading comment */ SELECT * FROM |` — leading block comment
- [ ] `SELECT * FROM t1 WHERE c1 = /* inline */ |` — inline comment in expression
- [ ] `` INSERT INTO `t1`(|) `` — backtick-quoted table in INSERT

### 3.5 Window Functions & Expressions (10 scenarios)

- [ ] `SELECT c1, SUM(c2) OVER (PARTITION BY |) FROM t2` — PARTITION BY columns
- [ ] `SELECT c1, SUM(c2) OVER (ORDER BY |) FROM t2` — window ORDER BY columns
- [ ] `SELECT c1, ROW_NUMBER() OVER (PARTITION BY | ORDER BY c1) FROM t2` — PARTITION BY in compound window
- [ ] `SELECT COALESCE(|, 0) FROM t2` — first arg of COALESCE
- [ ] `SELECT COALESCE(c1, |) FROM t2` — second arg of COALESCE
- [ ] `SELECT NULLIF(|, 0) FROM t2` — first arg of NULLIF
- [ ] `SELECT CAST(| AS int) FROM t2` — CAST argument
- [ ] `SELECT IF(|, 1, 0) FROM t2` — MySQL IF function
- [ ] `SELECT IFNULL(|, 0) FROM t2` — MySQL IFNULL function
- [ ] `SELECT * FROM t1 WHERE c1 > (SELECT MAX(|) FROM t2)` — aggregate in scalar subquery (dup of 1.4)

---

## Test Infrastructure

### Metadata Expansion

The current test metadata has: db with t1(c1), t2(c1,c2), v1.

To support database-qualified tests (MySQL equivalent of PG's schema-qualified tests), the test metadata getter needs to handle `db` as database name and return tables/views. No second database needed — MySQL's `db.|` tests verify the database-qualified path works with the default database.

### Verification

Each section is verified by:
1. Adding test cases to `test_completion.yaml`
2. Running `go test -count=1 ./backend/plugin/parser/mysql/ -run ^TestCompletion$`
3. For new scenarios, first run with `record = true` to capture actual output, then verify output is sensible before committing as expected

### Summary

| Phase | Sections | Total | Already Passing | New |
|-------|----------|-------|-----------------|-----|
| Phase 1 | 6 | 90 | 15 | 75 |
| Phase 2 | 4 | 48 | 12 | 36 |
| Phase 3 | 5 | 56 | 4 | 52 |
| **Total** | **15** | **194** | **31** | **163** |

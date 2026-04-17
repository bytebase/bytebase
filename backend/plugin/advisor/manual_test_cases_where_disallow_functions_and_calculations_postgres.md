# Manual UI Test Cases: `STATEMENT_WHERE_DISALLOW_FUNCTIONS_AND_CALCULATIONS` — PostgreSQL

Rule title: `Do not apply functions or perform calculations on indexed fields in the WHERE clause`

This document is intended for manual testing in the UI SQL editor with SQL review enabled against a PostgreSQL instance.

## Pre-requisites

Enable the SQL review rule `STATEMENT_WHERE_DISALLOW_FUNCTIONS_AND_CALCULATIONS` in the project SQL review policy and set the level to `Warning`. Target engine: **PostgreSQL**.

## Supported statements

| Statement | PostgreSQL |
|-----------|-------|
| `SELECT` | Yes |
| `INSERT ... SELECT` | Yes |
| `INSERT ... ON CONFLICT ... WHERE` | Yes (target-table scope) |
| `CREATE TABLE ... AS SELECT` | Yes |
| `CREATE [OR REPLACE] VIEW ... AS SELECT` | Yes |
| `CREATE MATERIALIZED VIEW ... AS SELECT` | Yes |
| `UPDATE` (incl. `UPDATE ... FROM`) | Yes |
| `DELETE` (incl. `DELETE ... USING`) | Yes |
| `MERGE` (ON condition + WHEN conditions) | Yes |
| Data-modifying CTE (`WITH ... UPDATE/DELETE/INSERT RETURNING ...`) | Yes |

Plus CTEs, `UNION` / `INTERSECT` / `EXCEPT` (including `ALL`), and nested / correlated subqueries within any of the above.

`HAVING` predicates are **never** flagged — a common post-aggregate predicate shape (`HAVING COUNT(name) > 5`) is intentional, not an index-abuse bug.

## Setup SQL

```sql
DROP TABLE IF EXISTS new_table CASCADE;
DROP TABLE IF EXISTS insert_target CASCADE;
DROP TABLE IF EXISTS no_index_table CASCADE;
DROP TABLE IF EXISTS products CASCADE;
DROP TABLE IF EXISTS orders CASCADE;
DROP TABLE IF EXISTS tech_book CASCADE;

CREATE TABLE tech_book (
  id       integer NOT NULL,
  name     text    NOT NULL,
  creator  text,
  PRIMARY KEY (id, name)
);
CREATE INDEX idx_tech_book_name ON tech_book (id, name);

CREATE TABLE orders (
  order_id      integer NOT NULL PRIMARY KEY,
  customer_name text,
  amount        numeric,
  note          text,
  name          text
);
CREATE INDEX idx_orders_customer ON orders (customer_name);
CREATE INDEX idx_orders_amount   ON orders (amount);

CREATE TABLE products (
  product_id integer NOT NULL PRIMARY KEY,
  title      text,
  price      numeric
);
CREATE INDEX idx_products_price ON products (price);

CREATE TABLE no_index_table (
  col_a integer,
  col_b text
);

CREATE TABLE insert_target (
  id   integer,
  name text
);
```

## Cleanup SQL

```sql
DROP TABLE IF EXISTS new_table CASCADE;
DROP TABLE IF EXISTS insert_target CASCADE;
DROP TABLE IF EXISTS no_index_table CASCADE;
DROP TABLE IF EXISTS products CASCADE;
DROP TABLE IF EXISTS orders CASCADE;
DROP TABLE IF EXISTS tech_book CASCADE;
```

## Test Cases

Expected labels:
- **Expected: WARN** — the rule should emit one or more advice entries.
- **Expected: PASS** — zero advice entries.

### Basic WARN

1. `SELECT * FROM tech_book WHERE UPPER(name) = 'test';` — Expected: WARN (function on indexed `name`).
2. `SELECT * FROM tech_book WHERE id + 1 > 5;` — Expected: WARN (calculation on indexed `id`).
3. `SELECT * FROM tech_book WHERE id * 2 = 10;` — Expected: WARN.
4. `SELECT * FROM tech_book WHERE id - 1 > 0;` — Expected: WARN.
5. `SELECT * FROM tech_book WHERE id / 2 > 5;` — Expected: WARN.
6. `SELECT * FROM tech_book WHERE -id > -5;` — Expected: WARN (unary minus).
7. `SELECT * FROM tech_book WHERE ABS(id) > 5 AND UPPER(name) = 'test';` — Expected: WARN (two advice entries).

### Basic PASS

8. `SELECT * FROM tech_book WHERE id = ABS(-5);` — Expected: PASS (function on value side).
9. `SELECT * FROM tech_book WHERE id = RANDOM();` — Expected: PASS.
10. `SELECT * FROM tech_book WHERE UPPER(creator) = 'test';` — Expected: PASS (`creator` not indexed).
11. `SELECT * FROM tech_book WHERE id = 1 AND name = 'test';` — Expected: PASS.
12. `SELECT * FROM unknown_table WHERE UPPER(col) = 'test';` — Expected: PASS (unknown table).
13. `SELECT * FROM no_index_table WHERE -col_a > 5;` — Expected: PASS.

### UPDATE / DELETE

14. `UPDATE tech_book SET creator = 'new' WHERE ABS(id) > 5;` — Expected: WARN.
15. `DELETE FROM tech_book WHERE UPPER(name) = 'test';` — Expected: WARN.
16. `UPDATE tech_book SET creator = 'new' WHERE id = 1;` — Expected: PASS.
17. `DELETE FROM no_index_table WHERE -col_a > 5;` — Expected: PASS.

### JOIN

18. `SELECT * FROM tech_book t JOIN orders o ON t.id = o.order_id WHERE UPPER(t.name) = 'X';` — Expected: WARN.
19. `SELECT * FROM tech_book t JOIN orders o ON UPPER(o.customer_name) = 'X' WHERE t.id = 1;` — Expected: WARN (ON predicate is checked).
20. `SELECT * FROM tech_book t, orders o WHERE t.id = o.order_id AND UPPER(t.name) = 'X';` — Expected: WARN (implicit JOIN).
21. `SELECT * FROM tech_book t JOIN no_index_table n ON t.id = n.col_a WHERE UPPER(n.col_b) = 'Y';` — Expected: PASS (non-indexed column).

### INSERT ... SELECT / CTAS

22. `INSERT INTO insert_target (id, name) SELECT id, name FROM tech_book WHERE UPPER(name) = 'test';` — Expected: WARN.
23. `CREATE TABLE new_table AS SELECT * FROM tech_book WHERE UPPER(name) = 'test';` — Expected: WARN.
24. `INSERT INTO insert_target (id, name) SELECT id, name FROM tech_book WHERE id = 1;` — Expected: PASS.

### CREATE VIEW / MATERIALIZED VIEW

25. `CREATE VIEW v AS SELECT * FROM tech_book WHERE UPPER(name) = 'test';` — Expected: WARN.
26. `CREATE MATERIALIZED VIEW mv AS SELECT * FROM tech_book WHERE id + 1 > 5;` — Expected: WARN.
27. `CREATE VIEW v2 AS SELECT * FROM tech_book WHERE id = 1;` — Expected: PASS.
28. `CREATE MATERIALIZED VIEW mv2 AS SELECT * FROM tech_book WHERE id = 1;` — Expected: PASS.

### Subqueries

29. `SELECT * FROM tech_book WHERE id IN (SELECT order_id FROM orders WHERE UPPER(customer_name) = 'JOHN');` — Expected: WARN (inner-scope violation).
30. `SELECT * FROM tech_book WHERE EXISTS (SELECT 1 FROM orders WHERE UPPER(customer_name) = 'X');` — Expected: WARN.
31. `SELECT * FROM tech_book WHERE NOT EXISTS (SELECT 1 FROM orders WHERE amount + 10 > 100);` — Expected: WARN.
32. `SELECT * FROM tech_book WHERE id NOT IN (SELECT order_id FROM orders WHERE ABS(amount) > 100);` — Expected: WARN.
33. `SELECT * FROM tech_book WHERE id > (SELECT COUNT(*) FROM orders WHERE ABS(amount) > 0);` — Expected: WARN (scalar subquery).
34. `SELECT * FROM tech_book WHERE id IN (SELECT order_id FROM orders WHERE amount > (SELECT AVG(amount) FROM orders WHERE LOWER(customer_name) = 'john'));` — Expected: WARN (3-level nested).
35. `SELECT * FROM tech_book WHERE UPPER(name) = 'A' AND id IN (SELECT order_id FROM orders WHERE ABS(amount) > 0);` — Expected: WARN (two advice entries).
36. `SELECT * FROM tech_book WHERE id IN (SELECT order_id FROM orders WHERE UPPER(note) = 'X');` — Expected: PASS (inner column not indexed).

### Set operations

37. `SELECT id FROM tech_book WHERE UPPER(name) = 'A' UNION SELECT order_id FROM orders;` — Expected: WARN (first branch).
38. `SELECT id FROM tech_book WHERE UPPER(name) = 'A' UNION ALL SELECT order_id FROM orders WHERE ABS(amount) > 0;` — Expected: WARN (both branches).
39. `SELECT id FROM tech_book WHERE id + 1 > 5 INTERSECT SELECT order_id FROM orders;` — Expected: WARN.
40. `SELECT id FROM tech_book WHERE UPPER(name) = 'A' EXCEPT SELECT order_id FROM orders;` — Expected: WARN.
41. `SELECT id FROM tech_book WHERE id = 1 UNION SELECT order_id FROM orders WHERE order_id = 1;` — Expected: PASS.

### CTE

42. `WITH bad AS (SELECT * FROM tech_book WHERE UPPER(name) = 'test') SELECT * FROM bad;` — Expected: WARN (CTE body).
43. `WITH ok AS (SELECT * FROM tech_book WHERE id = 1) SELECT * FROM ok WHERE UPPER(name) = 'X';` — Expected: WARN (outer query only; CTE body is PASS; CTE reference is opaque).
44. `WITH ok AS (SELECT * FROM tech_book WHERE id = 1) SELECT * FROM ok;` — Expected: PASS.

### MERGE

45. `MERGE INTO orders t USING products p ON (UPPER(t.customer_name) = p.title) WHEN MATCHED THEN UPDATE SET note = 'x';` — Expected: WARN (ON condition).
46. `MERGE INTO orders t USING products p ON (t.order_id = p.product_id) WHEN MATCHED AND t.amount * 2 > 100 THEN UPDATE SET note = 'x';` — Expected: WARN (WHEN condition).
47. `MERGE INTO orders t USING products p ON (t.order_id = p.product_id) WHEN MATCHED THEN UPDATE SET note = 'x';` — Expected: PASS.

### HAVING exclusion

48. `SELECT name, COUNT(*) FROM tech_book GROUP BY name HAVING COUNT(name) > 5;` — Expected: PASS.
49. `SELECT id, AVG(id) FROM tech_book GROUP BY id HAVING AVG(id) * 3 + 1 > 100;` — Expected: PASS.

### Function wrapping calculation (edge)

50. `SELECT * FROM tech_book WHERE ABS(id + 1) > 5;` — Expected: WARN (inner calc on indexed column).
51. `SELECT * FROM tech_book WHERE UPPER(CONCAT(name, 'x')) = 'TESTx';` — Expected: WARN.
52. `SELECT * FROM no_index_table WHERE ABS(col_a + 1) > 5;` — Expected: PASS.

### PostgreSQL-specific

53. `UPDATE orders o SET note = 'x' FROM products p WHERE UPPER(o.customer_name) = p.title;` — Expected: WARN (`UPDATE ... FROM`).
54. `DELETE FROM orders o USING products p WHERE ABS(o.amount) > 5;` — Expected: WARN (`DELETE ... USING`).
55. `INSERT INTO orders (order_id, customer_name) VALUES (1, 'a') ON CONFLICT (order_id) DO UPDATE SET note = 'x' WHERE UPPER(orders.customer_name) = 'A';` — Expected: WARN (`ON CONFLICT` WHERE on indexed target column).
56. `INSERT INTO orders (order_id, customer_name) VALUES (1, 'a') ON CONFLICT (order_id) DO UPDATE SET note = 'x' WHERE UPPER(orders.note) = 'A';` — Expected: PASS (non-indexed target column).
57. `WITH upd AS (UPDATE orders SET note = 'x' WHERE ABS(order_id) > 5 RETURNING *) SELECT * FROM upd;` — Expected: WARN (data-modifying CTE).

## Suggested Smoke Test Order

Run in sequence to cover the common categories quickly:

1. #1, #2, #6 (basic function, calculation, unary minus)
2. #14, #15 (UPDATE, DELETE)
3. #18, #19 (JOIN, ON predicate)
4. #22, #25, #26 (INSERT..SELECT, VIEW, MATVIEW)
5. #29, #33 (IN subquery, scalar subquery)
6. #37, #38 (UNION, UNION ALL)
7. #42 (CTE body)
8. #45, #46 (MERGE ON + WHEN)
9. #48 (HAVING — must be PASS)
10. #53, #54, #55 (PG extras)

## Behavior Notes

- Advice text is emitted at statement level with column names as stored in PostgreSQL catalog (lower-case for unquoted identifiers).
- The rule uses schema metadata loaded by Bytebase; run the "sync schema" action after DDL changes before expecting metadata-dependent advice.
- Qualified column references (`t.name`) resolve through FROM-clause aliases and fall back to outer scope only when no local alias matches (standard SQL name resolution).

## Known Limitations

These limitations apply to **all four engines** (MySQL, MSSQL, PostgreSQL, Oracle) that this rule supports. They are tracked for a future cross-engine enhancement PR.

1. **Expression / functional indexes are invisible.**
   `CREATE INDEX ON t (LOWER(name))` (PG), Oracle function-based indexes, MySQL 8 functional indexes, and MSSQL computed-column indexes all serve `WHERE LOWER(name)='x'` efficiently — but the rule still flags such queries. False-positive shape.

2. **Composite index trailing columns are treated as leading.**
   Given `INDEX (a, b)`, the rule marks both `a` and `b` as "indexed." A query like `WHERE UPPER(b) = 'x'` is flagged even though the optimizer likely would not have used this index for a bare-`b` predicate. False-positive shape.

3. **Partial / filtered indexes** — PG partial, MSSQL filtered, and Oracle FBIs with predicates are not distinguished from unconditional indexes; the rule flags based on column name alone.

4. **`CAST` / `TypeCast` is transparent.** `WHERE amount::text = '100'` (PG) or `WHERE CAST(amount AS VARCHAR2) = '100'` (Oracle) passes the rule even though the cast prevents b-tree use on `amount`. **False-negative shape** — the more dangerous direction. Prefer explicit range predicates on the cast source column until this is fixed cross-engine.

5. **Stale metadata fails open.** An index added after the last Bytebase metadata sync is invisible to the rule. Refresh advisor metadata after DDL changes.

6. **Advice text is diagnostic, not prescriptive.** `Function "UPPER" is applied to indexed column "name" …` tells you what, not how to fix. Common remediations: rewrite as a range predicate (`name BETWEEN 'a' AND 'b'`), add a functional index, or store a pre-computed column.

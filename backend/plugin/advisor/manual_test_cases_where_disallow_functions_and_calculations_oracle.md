# Manual UI Test Cases: `STATEMENT_WHERE_DISALLOW_FUNCTIONS_AND_CALCULATIONS` — Oracle

Rule title: `Do not apply functions or perform calculations on indexed fields in the WHERE clause`

This document is intended for manual testing in the UI SQL editor with SQL review enabled against an Oracle instance.

## Pre-requisites

Enable the SQL review rule `STATEMENT_WHERE_DISALLOW_FUNCTIONS_AND_CALCULATIONS` in the project SQL review policy and set the level to `Warning`. Target engine: **Oracle**.

## Supported statements

| Statement | Oracle |
|-----------|-------|
| `SELECT` | Yes |
| `INSERT ... SELECT` | Yes |
| `CREATE TABLE ... AS SELECT` | Yes |
| `CREATE [OR REPLACE] VIEW ... AS SELECT` | Yes |
| `CREATE MATERIALIZED VIEW ... AS SELECT` | Yes |
| `UPDATE` | Yes |
| `DELETE` | Yes |
| `MERGE` (ON + WHEN MATCHED … [WHERE] + DELETE WHERE + WHEN NOT MATCHED … WHERE) | Yes |
| `CONNECT BY` hierarchical query (START WITH + CONNECT BY predicates) | Yes |
| `(+)` outer join syntax | Yes |

Plus CTEs (`WITH` factoring clause), `UNION` / `UNION ALL` / `INTERSECT` / `MINUS`, and nested / correlated subqueries within any of the above.

`HAVING` predicates are **never** flagged — a common post-aggregate predicate shape is intentional, not an index-abuse bug.

DML embedded inside PL/SQL anonymous blocks (`BEGIN … END;`) is also checked — the walker reaches each top-level DML context independently of its surrounding block. DML inside PL/SQL procedure bodies and triggers is checked on the same basis when submitted as a top-level DDL.

## Setup SQL

Run as the schema owner (schema = your Bytebase default for this database — examples below use `TEST_DB`).

```sql
BEGIN EXECUTE IMMEDIATE 'DROP TABLE new_table PURGE'; EXCEPTION WHEN OTHERS THEN NULL; END;
/
BEGIN EXECUTE IMMEDIATE 'DROP TABLE insert_target PURGE'; EXCEPTION WHEN OTHERS THEN NULL; END;
/
BEGIN EXECUTE IMMEDIATE 'DROP TABLE no_index_table PURGE'; EXCEPTION WHEN OTHERS THEN NULL; END;
/
BEGIN EXECUTE IMMEDIATE 'DROP TABLE products PURGE'; EXCEPTION WHEN OTHERS THEN NULL; END;
/
BEGIN EXECUTE IMMEDIATE 'DROP TABLE orders PURGE'; EXCEPTION WHEN OTHERS THEN NULL; END;
/
BEGIN EXECUTE IMMEDIATE 'DROP TABLE tech_book PURGE'; EXCEPTION WHEN OTHERS THEN NULL; END;
/

CREATE TABLE tech_book (
  id       NUMBER NOT NULL,
  name     VARCHAR2(255) NOT NULL,
  creator  VARCHAR2(255),
  CONSTRAINT tech_book_pk PRIMARY KEY (id, name)
);
CREATE INDEX idx_tech_book_name ON tech_book (id, name);

CREATE TABLE orders (
  order_id      NUMBER NOT NULL,
  customer_name VARCHAR2(255),
  amount        NUMBER,
  note          VARCHAR2(255),
  name          VARCHAR2(255),
  CONSTRAINT orders_pk PRIMARY KEY (order_id)
);
CREATE INDEX idx_orders_customer ON orders (customer_name);
CREATE INDEX idx_orders_amount   ON orders (amount);

CREATE TABLE products (
  product_id NUMBER NOT NULL,
  title      VARCHAR2(255),
  price      NUMBER,
  CONSTRAINT products_pk PRIMARY KEY (product_id)
);
CREATE INDEX idx_products_price ON products (price);

CREATE TABLE no_index_table (
  col_a NUMBER,
  col_b VARCHAR2(255)
);

CREATE TABLE insert_target (
  id   NUMBER,
  name VARCHAR2(255)
);
```

## Cleanup SQL

```sql
DROP TABLE new_table PURGE;
DROP TABLE insert_target PURGE;
DROP TABLE no_index_table PURGE;
DROP TABLE products PURGE;
DROP TABLE orders PURGE;
DROP TABLE tech_book PURGE;
```

## Test Cases

Expected labels:
- **Expected: WARN** — the rule should emit one or more advice entries.
- **Expected: PASS** — zero advice entries.

### Basic WARN

1. `SELECT * FROM tech_book WHERE UPPER(name) = 'test'` — Expected: WARN.
2. `SELECT * FROM tech_book WHERE MOD(id, 2) = 0` — Expected: WARN (`MOD` treated as function on indexed col).
3. `SELECT * FROM tech_book WHERE id + 1 = 2` — Expected: WARN.
4. `SELECT * FROM tech_book WHERE id - 1 = 0` — Expected: WARN.
5. `SELECT * FROM tech_book WHERE id * 2 = 10` — Expected: WARN.
6. `SELECT * FROM tech_book WHERE id / 2 = 5` — Expected: WARN.
7. `SELECT * FROM tech_book WHERE -id > -5` — Expected: WARN (unary minus).

### Basic PASS

8. `SELECT * FROM tech_book WHERE id = ABS(-5)` — Expected: PASS (function on value side).
9. `SELECT * FROM tech_book WHERE id = DBMS_RANDOM.VALUE` — Expected: PASS.
10. `SELECT * FROM tech_book WHERE UPPER(creator) = 'X'` — Expected: PASS (`creator` not indexed).
11. `SELECT * FROM tech_book WHERE id = 1` — Expected: PASS.
12. `SELECT * FROM unknown WHERE UPPER(col) = 'X'` — Expected: PASS (unknown table).
13. `SELECT * FROM no_index_table WHERE -col_a > 5` — Expected: PASS.

### UPDATE / DELETE

14. `UPDATE tech_book SET creator = 'x' WHERE UPPER(name) = 'test'` — Expected: WARN.
15. `UPDATE tech_book SET creator = 'x' WHERE id * 2 = 10` — Expected: WARN.
16. `DELETE FROM tech_book WHERE UPPER(name) = 'test'` — Expected: WARN.
17. `DELETE FROM tech_book WHERE id + 1 > 10` — Expected: WARN.
18. `UPDATE tech_book SET creator = 'x' WHERE id = 1` — Expected: PASS.
19. `DELETE FROM tech_book WHERE id = 1` — Expected: PASS.

### JOIN (ANSI and implicit)

20. `SELECT * FROM tech_book t JOIN orders o ON t.id = o.order_id WHERE UPPER(t.name) = 'X'` — Expected: WARN.
21. `SELECT * FROM tech_book t JOIN orders o ON UPPER(o.customer_name) = 'X' WHERE t.id = 1` — Expected: WARN (ANSI ON predicate).
22. `SELECT * FROM tech_book t, orders o WHERE t.id = o.order_id AND UPPER(t.name) = 'X'` — Expected: WARN (implicit join).
23. `SELECT * FROM tech_book t JOIN orders o ON t.id = o.order_id WHERE t.id = 1 AND o.order_id = 1` — Expected: PASS.

### INSERT … SELECT / CTAS / VIEW / MATERIALIZED VIEW

24. `INSERT INTO insert_target (id, name) SELECT id, name FROM tech_book WHERE UPPER(name) = 'test'` — Expected: WARN.
25. `CREATE TABLE new_table AS SELECT * FROM tech_book WHERE UPPER(name) = 'test'` — Expected: WARN.
26. `CREATE OR REPLACE VIEW v AS SELECT * FROM tech_book WHERE id + 1 > 5` — Expected: WARN.
27. `CREATE MATERIALIZED VIEW mv AS SELECT * FROM orders WHERE ABS(amount) > 10` — Expected: WARN.
28. `CREATE TABLE new_table AS SELECT * FROM tech_book WHERE id = 1` — Expected: PASS.
29. `CREATE OR REPLACE VIEW v_ok AS SELECT * FROM tech_book WHERE id = 1` — Expected: PASS.

### Subqueries (IN / NOT IN / EXISTS / NOT EXISTS / scalar / correlated / nested)

30. `SELECT * FROM tech_book WHERE id IN (SELECT order_id FROM orders WHERE UPPER(customer_name) = 'JOHN')` — Expected: WARN.
31. `SELECT * FROM tech_book WHERE id NOT IN (SELECT order_id FROM orders WHERE ABS(amount) > 100)` — Expected: WARN.
32. `SELECT * FROM tech_book WHERE EXISTS (SELECT 1 FROM orders WHERE UPPER(customer_name) = 'X')` — Expected: WARN.
33. `SELECT * FROM tech_book WHERE NOT EXISTS (SELECT 1 FROM orders WHERE amount + 10 > 100)` — Expected: WARN.
34. `SELECT * FROM tech_book WHERE id > (SELECT COUNT(*) FROM orders WHERE ABS(amount) > 0)` — Expected: WARN.
35. `SELECT * FROM tech_book t WHERE EXISTS (SELECT 1 FROM orders o WHERE o.order_id = t.id AND UPPER(o.customer_name) = 'X')` — Expected: WARN (correlated).
36. `SELECT * FROM tech_book WHERE id IN (SELECT order_id FROM orders WHERE amount > (SELECT AVG(amount) FROM orders WHERE LOWER(customer_name) = 'john'))` — Expected: WARN (3-level nested).
37. `SELECT * FROM tech_book WHERE UPPER(name) = 'A' AND id IN (SELECT order_id FROM orders WHERE ABS(amount) > 0)` — Expected: WARN (two advice entries).
38. `SELECT * FROM tech_book WHERE id IN (SELECT order_id FROM orders WHERE UPPER(note) = 'X')` — Expected: PASS.

### Set operations (UNION / INTERSECT / MINUS)

39. `SELECT id FROM tech_book WHERE UPPER(name) = 'A' UNION SELECT order_id FROM orders` — Expected: WARN.
40. `SELECT id FROM tech_book WHERE UPPER(name) = 'A' UNION ALL SELECT order_id FROM orders WHERE ABS(amount) > 0` — Expected: WARN (two advice).
41. `SELECT id FROM tech_book WHERE id + 1 > 5 INTERSECT SELECT order_id FROM orders` — Expected: WARN.
42. `SELECT id FROM tech_book WHERE UPPER(name) = 'A' MINUS SELECT order_id FROM orders` — Expected: WARN.

### CTE

43. `WITH bad AS (SELECT * FROM tech_book WHERE UPPER(name) = 'test') SELECT * FROM bad` — Expected: WARN.
44. `WITH ok AS (SELECT * FROM tech_book WHERE id = 1) SELECT * FROM ok` — Expected: PASS.

### MERGE

45. `MERGE INTO orders t USING products p ON (UPPER(t.customer_name) = p.title) WHEN MATCHED THEN UPDATE SET note = 'x'` — Expected: WARN (ON condition).
46. `MERGE INTO orders t USING products p ON (t.order_id = p.product_id) WHEN MATCHED THEN UPDATE SET note = 'x' WHERE t.amount * 2 > 100` — Expected: WARN (MATCHED WHERE).
47. `MERGE INTO orders t USING products p ON (t.order_id = p.product_id) WHEN MATCHED THEN UPDATE SET note = 'x' DELETE WHERE ABS(t.amount) > 100` — Expected: WARN (Oracle-only DELETE WHERE).
48. `MERGE INTO orders t USING products p ON (t.order_id = p.product_id) WHEN MATCHED THEN UPDATE SET note = 'x'` — Expected: PASS.

### HAVING exclusion

49. `SELECT name, COUNT(*) FROM tech_book GROUP BY name HAVING COUNT(name) > 5` — Expected: PASS.
50. `SELECT id, AVG(id) FROM tech_book GROUP BY id HAVING AVG(id) * 3 + 1 > 100` — Expected: PASS.

### Function wrapping calculation / CAST / NVL (edge)

51. `SELECT * FROM tech_book WHERE ABS(id + 1) > 5` — Expected: WARN (inner calculation).
52. `SELECT * FROM tech_book WHERE UPPER(NVL(name, 'x')) = 'TESTx'` — Expected: WARN.
53. `SELECT * FROM no_index_table WHERE ABS(col_a + 1) > 5` — Expected: PASS.

### Oracle-specific

54. `SELECT * FROM tech_book t, orders o WHERE t.id = o.order_id(+) AND UPPER(t.name) = 'X'` — Expected: WARN (`(+)` outer join).
55. `SELECT id, name FROM tech_book START WITH id = 1 CONNECT BY PRIOR id = id AND UPPER(name) = 'X'` — Expected: WARN (hierarchical query).
56. `SELECT id, name FROM tech_book START WITH id = 1 CONNECT BY PRIOR id = id` — Expected: PASS (PRIOR is not arithmetic).
57. `SELECT * FROM tech_book t WHERE t.id IN (SELECT id FROM tech_book CONNECT BY PRIOR id = id AND UPPER(name) = 'X')` — Expected: WARN (CONNECT BY inside nested subquery).

### Multi-statement depth-reset

58. Submit this as two statements in one request:
    ```
    SELECT * FROM tech_book WHERE UPPER(name) = 'A';
    UPDATE tech_book SET creator = 'x' WHERE ABS(id) > 5
    ```
    Expected: WARN (two advice entries, one per statement). Regression case — confirms the `depth` counter resets between top-level statements.

## Suggested Smoke Test Order

Run in sequence to cover the common categories quickly:

1. #1, #3, #7 (basic function, calculation, unary minus)
2. #14, #16 (UPDATE, DELETE)
3. #20, #21 (JOIN, ANSI ON predicate)
4. #24, #26, #27 (INSERT..SELECT, VIEW, MATVIEW)
5. #30, #34, #35 (IN, scalar, correlated)
6. #39, #42 (UNION, MINUS)
7. #43 (CTE)
8. #45, #46, #47 (MERGE ON, WHEN MATCHED WHERE, DELETE WHERE)
9. #49 (HAVING — must be PASS)
10. #54, #55 (Oracle specifics)
11. #58 (multi-statement regression)

## Behavior Notes

- Advice text reports the column name as stored in Oracle metadata. With case-insensitive metadata (the Bytebase default for Oracle), unquoted names round-trip as lower-case in the advice text for cross-engine parity. Quoted (`"Foo"`) identifiers preserve case.
- The rule uses schema metadata loaded by Bytebase; run the "sync schema" action after DDL changes before expecting metadata-dependent advice.
- `(+)` outer-join syntax is a marker on the column reference; scope resolution behaves the same as the equivalent ANSI join.
- `PRIOR` in `CONNECT BY` is its own unary operator and is **not** conflated with arithmetic unary minus.

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


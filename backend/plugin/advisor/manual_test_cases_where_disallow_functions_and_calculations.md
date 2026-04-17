# Manual UI Test Cases: `STATEMENT_WHERE_DISALLOW_FUNCTIONS_AND_CALCULATIONS`

Rule title: `Do not apply functions or perform calculations on indexed fields in the WHERE clause`

This document is intended for manual testing in the UI SQL editor with SQL review enabled.

## Pre-requisites

Enable the SQL review rule `STATEMENT_WHERE_DISALLOW_FUNCTIONS_AND_CALCULATIONS` in the project SQL review policy and set the level to `Warning`.

The current implementation covers: **MySQL** and **MSSQL**.

## Supported statements

| Statement | MySQL | MSSQL |
|-----------|-------|-------|
| `SELECT` | Yes | Yes |
| `INSERT...SELECT` | Yes | Yes |
| `CREATE TABLE...AS SELECT` | Yes | Synapse/Fabric only (not standard SQL Server) |
| `UPDATE` | Yes | Yes (incl. T-SQL `FROM` clause) |
| `DELETE` | Yes | Yes (incl. T-SQL `FROM` clause) |
| `MERGE` | — | Yes (ON condition + WHEN conditions) |

Plus CTEs, UNION, and nested subqueries (including derived tables in FROM, scalar
subqueries in SELECT list / HAVING / GROUP BY / ORDER BY) within any of the above.

## MySQL

### Setup SQL

```sql
DROP TABLE IF EXISTS new_table;
DROP TABLE IF EXISTS insert_target;
DROP TABLE IF EXISTS no_index_table;
DROP TABLE IF EXISTS products;
DROP TABLE IF EXISTS orders;
DROP TABLE IF EXISTS tech_book;

CREATE TABLE tech_book (
  id INT NOT NULL,
  name VARCHAR(255) NOT NULL,
  creator VARCHAR(255),
  PRIMARY KEY (id, name),
  INDEX idx_name (id, name)
);

CREATE TABLE orders (
  order_id INT NOT NULL,
  customer_name VARCHAR(255),
  amount DECIMAL(10,2),
  note VARCHAR(255),
  created_at DATETIME,
  PRIMARY KEY (order_id),
  INDEX idx_customer (customer_name),
  INDEX idx_amount (amount)
);

CREATE TABLE products (
  product_id INT NOT NULL,
  title VARCHAR(255),
  price DECIMAL(10,2),
  PRIMARY KEY (product_id),
  INDEX idx_price (price)
);

CREATE TABLE no_index_table (
  col_a INT,
  col_b VARCHAR(255)
);

CREATE TABLE insert_target (
  id INT,
  name VARCHAR(255)
);
```

### Cleanup SQL

```sql
DROP TABLE IF EXISTS new_table;
DROP TABLE IF EXISTS insert_target;
DROP TABLE IF EXISTS no_index_table;
DROP TABLE IF EXISTS products;
DROP TABLE IF EXISTS orders;
DROP TABLE IF EXISTS tech_book;
```

### Test Cases

#### Basic — WARN

**MySQL 1. Function on indexed column**

```sql
SELECT * FROM tech_book WHERE UPPER(name) = 'TEST';
```

Expected: `WARN`

**MySQL 2. Addition on indexed column**

```sql
SELECT * FROM tech_book WHERE id + 1 > 5;
```

Expected: `WARN`

**MySQL 3. Multiplication on indexed column**

```sql
SELECT * FROM tech_book WHERE id * 2 = 10;
```

Expected: `WARN`

**MySQL 4. Subtraction on indexed column**

```sql
SELECT * FROM orders WHERE amount - 50 > 0;
```

Expected: `WARN`

**MySQL 5. Division on indexed column**

```sql
SELECT * FROM tech_book WHERE id / 2 > 3;
```

Expected: `WARN`

**MySQL 6. Modulo on indexed column**

```sql
SELECT * FROM tech_book WHERE id % 2 = 0;
```

Expected: `WARN`

**MySQL 7. Bitwise operation on indexed column**

```sql
SELECT * FROM tech_book WHERE id & 255 = 1;
```

Expected: `WARN`

**MySQL 8. Unary minus on indexed column**

```sql
SELECT * FROM tech_book WHERE -id > 5;
```

Expected: `WARN`

#### Basic — PASS

**MySQL 9. Function on value side only**

```sql
SELECT * FROM tech_book WHERE id = ABS(-5);
```

Expected: `PASS`

**MySQL 10. RAND on value side**

```sql
SELECT * FROM tech_book WHERE id = RAND();
```

Expected: `PASS`

**MySQL 11. Function on non-indexed column**

```sql
SELECT * FROM tech_book WHERE UPPER(creator) = 'ADMIN';
```

Expected: `PASS`

**MySQL 12. Calculation on non-indexed column**

```sql
SELECT * FROM tech_book WHERE CHAR_LENGTH(creator) > 10;
```

Expected: `PASS`

**MySQL 13. Simple comparison on indexed columns**

```sql
SELECT * FROM tech_book WHERE id = 1 AND name = 'test';
```

Expected: `PASS`

**MySQL 14. Table not found in schema metadata**

```sql
SELECT * FROM unknown_table WHERE UPPER(col) = 'X';
```

Expected: `PASS`

**MySQL 15. Unary minus on non-indexed column**

```sql
SELECT * FROM no_index_table WHERE -col_a > 5;
```

Expected: `PASS`

#### Medium — WARN

**MySQL 16. Multiple violations in the same WHERE**

```sql
SELECT * FROM tech_book WHERE ABS(id) > 5 AND UPPER(name) = 'TEST';
```

Expected: `WARN x2`

**MySQL 17. Indexed and non-indexed mixed — only flag indexed**

```sql
SELECT * FROM tech_book WHERE UPPER(name) = 'TEST' AND LOWER(creator) = 'admin';
```

Expected: `WARN x1`

**MySQL 18. UPDATE with function on indexed column**

```sql
UPDATE tech_book SET creator = 'new' WHERE ABS(id) > 5;
```

Expected: `WARN`

**MySQL 19. DELETE with function on indexed column**

```sql
DELETE FROM tech_book WHERE UPPER(name) = 'TEST';
```

Expected: `WARN`

**MySQL 20. UPDATE with calculation on indexed column**

```sql
UPDATE orders SET amount = 0 WHERE amount - 10 > 100;
```

Expected: `WARN`

**MySQL 21. Alias with function on indexed column**

```sql
SELECT * FROM tech_book t WHERE UPPER(t.name) = 'TEST';
```

Expected: `WARN`

**MySQL 22. Alias with calculation on indexed column**

```sql
SELECT * FROM orders o WHERE o.amount * 1.1 > 500;
```

Expected: `WARN`

#### Medium — PASS

**MySQL 23. DELETE with plain indexed comparison**

```sql
DELETE FROM tech_book WHERE id = 5;
```

Expected: `PASS`

**MySQL 24. UPDATE with plain indexed comparison**

```sql
UPDATE orders SET note = 'x' WHERE order_id = 1;
```

Expected: `PASS`

#### INSERT / CREATE TABLE — WARN

**MySQL 25. INSERT...SELECT with function on indexed column**

```sql
INSERT INTO insert_target (id, name)
SELECT id, name
FROM tech_book
WHERE UPPER(name) = 'TEST';
```

Expected: `WARN`

**MySQL 26. INSERT...SELECT with calculation on indexed column**

```sql
INSERT INTO insert_target (id, name)
SELECT id, name
FROM tech_book
WHERE id + 1 > 5;
```

Expected: `WARN`

**MySQL 27. CREATE TABLE ... AS SELECT with function on indexed column**

```sql
DROP TABLE IF EXISTS new_table;
CREATE TABLE new_table AS SELECT * FROM tech_book WHERE UPPER(name) = 'TEST';
```

Expected: `WARN`

**MySQL 28. CREATE TABLE ... AS SELECT with calculation on indexed column**

```sql
DROP TABLE IF EXISTS new_table;
CREATE TABLE new_table AS SELECT * FROM orders WHERE amount * 2 > 100;
```

Expected: `WARN`

#### INSERT / CREATE TABLE — PASS

**MySQL 29. INSERT...SELECT without violation**

```sql
INSERT INTO insert_target (id, name)
SELECT id, name
FROM tech_book
WHERE id = 1;
```

Expected: `PASS`

**MySQL 30. CREATE TABLE ... AS SELECT without violation**

```sql
DROP TABLE IF EXISTS new_table;
CREATE TABLE new_table AS SELECT * FROM tech_book WHERE id = 1;
```

Expected: `PASS`

#### JOIN — WARN

**MySQL 31. JOIN with one indexed-column violation**

```sql
SELECT *
FROM tech_book t
JOIN orders o ON t.id = o.order_id
WHERE UPPER(t.name) = 'TEST';
```

Expected: `WARN`

**MySQL 32. JOIN with two indexed-column violations**

```sql
SELECT *
FROM tech_book t
JOIN orders o ON t.id = o.order_id
WHERE ABS(t.id) > 5 AND LOWER(o.customer_name) = 'john';
```

Expected: `WARN x2`

#### JOIN — PASS

**MySQL 33. JOIN with function on non-indexed column only**

```sql
SELECT *
FROM tech_book t
JOIN no_index_table n ON t.id = n.col_a
WHERE UPPER(n.col_b) = 'X';
```

Expected: `PASS`

#### Subquery — WARN

**MySQL 34. Violation inside IN subquery (different table)**

```sql
SELECT * FROM tech_book
WHERE id IN (
  SELECT order_id FROM orders WHERE UPPER(customer_name) = 'JOHN'
);
```

Expected: `WARN`

**MySQL 35. Outer query violates, inner subquery is clean**

```sql
SELECT * FROM tech_book
WHERE UPPER(name) = 'TEST'
  AND id IN (SELECT col_a FROM no_index_table WHERE col_a = 1);
```

Expected: `WARN`

**MySQL 36. Outer query violates with scalar subquery on value side**

```sql
SELECT * FROM orders o
WHERE ABS(o.amount) > (
  SELECT AVG(price) FROM products
);
```

Expected: `WARN`

**MySQL 37. Three-level nested subquery — innermost violates**

```sql
SELECT * FROM tech_book
WHERE id IN (
  SELECT order_id FROM orders
  WHERE amount > (
    SELECT AVG(price) FROM products WHERE ABS(price) > 10
  )
);
```

Expected: `WARN`

**MySQL 38. Four-level nested subquery — deepest violates**

```sql
SELECT * FROM tech_book WHERE id IN (
  SELECT order_id FROM orders WHERE amount IN (
    SELECT amount FROM orders WHERE order_id IN (
      SELECT order_id FROM orders WHERE UPPER(customer_name) = 'X'
    )
  )
);
```

Expected: `WARN`

**MySQL 39. NOT IN subquery with function on inner indexed column**

```sql
SELECT * FROM tech_book WHERE id NOT IN (
  SELECT order_id FROM orders WHERE ABS(amount) > 100
);
```

Expected: `WARN`

**MySQL 40. NOT EXISTS subquery with calculation on inner indexed column**

```sql
SELECT * FROM tech_book WHERE NOT EXISTS (
  SELECT 1 FROM orders WHERE amount + 10 > 100
);
```

Expected: `WARN`

**MySQL 41. EXISTS subquery with function on inner indexed column**

```sql
SELECT * FROM tech_book WHERE EXISTS (
  SELECT 1 FROM orders WHERE UPPER(customer_name) = 'X'
);
```

Expected: `WARN`

**MySQL 42. Correlated subquery**

```sql
SELECT * FROM tech_book t
WHERE EXISTS (
  SELECT 1 FROM orders o
  WHERE o.order_id = t.id AND UPPER(o.customer_name) = 'X'
);
```

Expected: `WARN`

**MySQL 43. Violations at both outer and inner**

```sql
SELECT * FROM tech_book
WHERE UPPER(name) = 'A'
  AND id IN (SELECT order_id FROM orders WHERE ABS(amount) > 0);
```

Expected: `WARN x2`

**MySQL 44. Violations at three levels (outer, middle, inner)**

```sql
SELECT * FROM tech_book WHERE UPPER(name) = 'A'
  AND id IN (
    SELECT order_id FROM orders WHERE LOWER(customer_name) = 'x'
      AND amount > (
        SELECT AVG(amount) FROM orders WHERE ABS(amount) > 0
      )
  );
```

Expected: `WARN x3`

**MySQL 45. Multiple independent subqueries — flag each**

```sql
SELECT * FROM tech_book
WHERE id IN (SELECT order_id FROM orders WHERE ABS(amount) > 0)
  AND name IN (SELECT customer_name FROM orders WHERE UPPER(customer_name) = 'A');
```

Expected: `WARN x2`

**MySQL 46. Subquery referencing same table as outer**

```sql
SELECT * FROM tech_book WHERE id IN (
  SELECT id FROM tech_book WHERE UPPER(name) = 'X'
);
```

Expected: `WARN`

**MySQL 47. UPDATE with subquery in WHERE — inner violation**

```sql
UPDATE tech_book SET creator = 'x'
WHERE id IN (
  SELECT order_id FROM orders WHERE UPPER(customer_name) = 'JOHN'
);
```

Expected: `WARN`

**MySQL 48. DELETE with subquery in WHERE — inner violation**

```sql
DELETE FROM tech_book
WHERE id IN (
  SELECT order_id FROM orders WHERE ABS(amount) > 10
);
```

Expected: `WARN`

#### Subquery — PASS

**MySQL 49. Inner subquery on table with no matching index**

```sql
SELECT * FROM tech_book WHERE id IN (
  SELECT order_id FROM orders WHERE UPPER(note) = 'x' AND LOWER(note) = 'y'
);
```

Expected: `PASS` — `note` is not indexed on `orders`.

**MySQL 50. Subquery with function on non-indexed column of non-indexed table**

```sql
SELECT * FROM tech_book WHERE id IN (
  SELECT col_a FROM no_index_table WHERE UPPER(col_b) = 'X'
);
```

Expected: `PASS`

#### UNION — WARN

**MySQL 51. UNION with violation in first branch**

```sql
SELECT * FROM tech_book WHERE UPPER(name) = 'A'
UNION
SELECT * FROM tech_book WHERE id = 1;
```

Expected: `WARN`

**MySQL 52. UNION ALL with violations in both branches**

```sql
SELECT * FROM tech_book WHERE ABS(id) > 0
UNION ALL
SELECT * FROM orders WHERE UPPER(customer_name) = 'X';
```

Expected: `WARN x2`

**MySQL 53. UNION where one branch has a violating subquery**

```sql
SELECT * FROM tech_book WHERE id = 1
UNION
SELECT * FROM tech_book WHERE id IN (
  SELECT order_id FROM orders WHERE UPPER(customer_name) = 'A'
);
```

Expected: `WARN`

#### CTE — WARN

**MySQL 54. CTE body violation**

```sql
WITH filtered AS (
  SELECT * FROM tech_book WHERE UPPER(name) = 'TEST'
)
SELECT * FROM filtered WHERE id = 1;
```

Expected: `WARN`

**MySQL 55. Outer query violation after CTE**

```sql
WITH summary AS (
  SELECT order_id, amount FROM orders WHERE amount > 100
)
SELECT * FROM tech_book
WHERE ABS(id) > (SELECT MIN(order_id) FROM summary);
```

Expected: `WARN`

**MySQL 56. CTE body with calculation violation**

```sql
WITH calc AS (
  SELECT * FROM orders WHERE amount * 2 > 100
)
SELECT * FROM calc;
```

Expected: `WARN`

#### CTE — PASS

**MySQL 57. CTE body without violation**

```sql
WITH clean AS (
  SELECT * FROM tech_book WHERE id = 1
)
SELECT * FROM clean;
```

Expected: `PASS`

#### Edge — Function wrapping expression

**MySQL 58. ABS(indexed_col + 1) — inner calculation on indexed column**

```sql
SELECT * FROM tech_book WHERE ABS(id + 1) > 5;
```

Expected: `WARN` — the inner `id + 1` calculation is detected by the walker.

**MySQL 59. UPPER(CONCAT(indexed_col, 'x')) — nested function with indexed column as inner arg**

```sql
SELECT * FROM tech_book WHERE UPPER(CONCAT(name, 'x')) = 'TESTx';
```

Expected: `WARN` — the walker descends into function args and finds `name`.

**MySQL 60. ABS(non_indexed_col + 1) — no indexed column involved**

```sql
SELECT * FROM no_index_table WHERE ABS(col_a + 1) > 5;
```

Expected: `PASS`

#### Edge — HAVING

**MySQL 61. HAVING with aggregate function — not WHERE**

```sql
SELECT name, COUNT(*) AS cnt
FROM tech_book
GROUP BY name
HAVING COUNT(name) > 5;
```

Expected: `PASS`

**MySQL 62. HAVING with arithmetic expression — not WHERE**

```sql
SELECT id, AVG(id) AS avg_id
FROM tech_book
GROUP BY id
HAVING AVG(id) * 3 + 1 > 100;
```

Expected: `PASS`

---

## MSSQL

### Setup SQL

```sql
DROP TABLE IF EXISTS dbo.new_table_ms;
DROP TABLE IF EXISTS dbo.insert_target_ms;
DROP TABLE IF EXISTS dbo.orders_ms;
DROP TABLE IF EXISTS dbo.pokes2;
DROP TABLE IF EXISTS dbo.pokes;

CREATE TABLE dbo.pokes (
  c1 INT,
  c2 INT,
  c3 INT,
  c10 INT,
  c20 INT,
  foo INT,
  bar INT
);
CREATE INDEX idx_0 ON dbo.pokes (c1, c2, c3);
CREATE INDEX idx_1 ON dbo.pokes (c10, c20);

CREATE TABLE dbo.pokes2 (
  foo INT,
  bar INT
);

CREATE TABLE dbo.orders_ms (
  order_id INT NOT NULL PRIMARY KEY,
  customer_name NVARCHAR(255),
  amount DECIMAL(10,2),
  note NVARCHAR(255)
);
CREATE INDEX idx_cust ON dbo.orders_ms (customer_name);
CREATE INDEX idx_amt ON dbo.orders_ms (amount);

CREATE TABLE dbo.insert_target_ms (
  c1 INT,
  c2 INT
);
```

### Cleanup SQL

```sql
DROP TABLE IF EXISTS dbo.new_table_ms;
DROP TABLE IF EXISTS dbo.insert_target_ms;
DROP TABLE IF EXISTS dbo.orders_ms;
DROP TABLE IF EXISTS dbo.pokes2;
DROP TABLE IF EXISTS dbo.pokes;
```

### Test Cases

#### Basic — WARN

**MSSQL 1. Function on indexed column**

```sql
SELECT c1 FROM dbo.pokes WHERE ABS(c1) > 5;
```

Expected: `WARN`

**MSSQL 2. Calculation on indexed column**

```sql
SELECT c1 FROM dbo.pokes WHERE c1 + 1 > 5;
```

Expected: `WARN`

**MSSQL 3. Unary bitwise NOT on indexed column**

```sql
SELECT c1 FROM dbo.pokes WHERE ~c1 > 0;
```

Expected: `WARN`

**MSSQL 4. Unary minus on indexed column**

```sql
SELECT c1 FROM dbo.pokes WHERE -c1 > 0;
```

Expected: `WARN`

**MSSQL 5. Multiplication on indexed column**

```sql
SELECT c1 FROM dbo.pokes WHERE c1 * 2 = 10;
```

Expected: `WARN`

#### Basic — PASS

**MSSQL 6. Unary operator on non-indexed column**

```sql
SELECT foo FROM dbo.pokes WHERE ~foo > 0;
```

Expected: `PASS`

**MSSQL 7. Function on value side only**

```sql
SELECT c1 FROM dbo.pokes WHERE c1 = ABS(-5);
```

Expected: `PASS`

**MSSQL 8. Simple comparison on indexed column**

```sql
SELECT c1 FROM dbo.pokes WHERE c1 > 1;
```

Expected: `PASS`

**MSSQL 9. Function on non-indexed column**

```sql
SELECT bar FROM dbo.pokes WHERE ABS(bar) > 0;
```

Expected: `PASS`

**MSSQL 10. Calculation on non-indexed column**

```sql
SELECT foo FROM dbo.pokes WHERE (foo + 1) * 2 > 0;
```

Expected: `PASS`

**MSSQL 11. Table with no indexes**

```sql
SELECT foo FROM dbo.pokes2 WHERE ABS(foo) > 5;
```

Expected: `PASS`

**MSSQL 12. Bitwise expression on non-indexed column**

```sql
SELECT foo FROM dbo.pokes WHERE foo | -foo > 0;
```

Expected: `PASS`

#### Medium — WARN

**MSSQL 13. Multiple indexed-column violations**

```sql
SELECT c1, c10 FROM dbo.pokes WHERE ABS(c1) > 5 AND c10 + 1 > 10;
```

Expected: `WARN x2`

**MSSQL 14. UPDATE with function on indexed column**

```sql
UPDATE dbo.pokes SET foo = 0 WHERE ABS(c1) > 5;
```

Expected: `WARN`

**MSSQL 15. DELETE with function on indexed column**

```sql
DELETE FROM dbo.pokes WHERE ABS(c1) > 10;
```

Expected: `WARN`

#### Medium — PASS

**MSSQL 16. UPDATE with plain comparison**

```sql
UPDATE dbo.pokes SET foo = 0 WHERE c1 = 1;
```

Expected: `PASS`

**MSSQL 17. DELETE with plain comparison**

```sql
DELETE FROM dbo.pokes WHERE c1 = 5;
```

Expected: `PASS`

#### INSERT / CREATE TABLE — WARN

**MSSQL 18. INSERT...SELECT with function on indexed column**

```sql
INSERT INTO dbo.insert_target_ms (c1, c2)
SELECT c1, c2
FROM dbo.pokes
WHERE ABS(c1) > 5;
```

Expected: `WARN`

**MSSQL 19. INSERT...SELECT with calculation on indexed column**

```sql
INSERT INTO dbo.insert_target_ms (c1, c2)
SELECT c1, c2
FROM dbo.pokes
WHERE c1 + 1 > 5;
```

Expected: `WARN`

#### INSERT / CREATE TABLE — PASS

**MSSQL 20. INSERT...SELECT without violation**

```sql
INSERT INTO dbo.insert_target_ms (c1, c2)
SELECT c1, c2
FROM dbo.pokes
WHERE c1 = 1;
```

Expected: `PASS`

#### Subquery — WARN

**MSSQL 21. Violation inside IN subquery (different table)**

```sql
SELECT c1 FROM dbo.pokes
WHERE c1 IN (
  SELECT order_id FROM dbo.orders_ms WHERE UPPER(customer_name) = 'JOHN'
);
```

Expected: `WARN`

**MSSQL 22. Outer on non-indexed table, inner on indexed — inner flags**

```sql
SELECT foo FROM dbo.pokes2
WHERE foo IN (SELECT c1 FROM dbo.pokes WHERE ABS(c1) > 5);
```

Expected: `WARN`

**MSSQL 23. Three-level nested subquery — innermost violates**

```sql
SELECT foo FROM dbo.pokes2
WHERE foo IN (
  SELECT foo FROM dbo.pokes2
  WHERE foo IN (SELECT c1 FROM dbo.pokes WHERE ABS(c1) > 0)
);
```

Expected: `WARN`

**MSSQL 24. EXISTS subquery with calculation**

```sql
SELECT foo FROM dbo.pokes2
WHERE EXISTS (SELECT 1 FROM dbo.pokes WHERE c1 + 1 > 5);
```

Expected: `WARN`

**MSSQL 25. Violations at both outer and inner**

```sql
SELECT c1 FROM dbo.pokes
WHERE ABS(c1) > 5
  AND c10 IN (SELECT c1 FROM dbo.pokes WHERE c1 * 2 > 0);
```

Expected: `WARN x2`

**MSSQL 26. Scalar subquery — only flag inner WHERE, not SELECT list**

```sql
SELECT foo FROM dbo.pokes2
WHERE foo > (SELECT MAX(c1) FROM dbo.pokes WHERE ABS(c1) > 0);
```

Expected: `WARN` — only `ABS(c1)` in inner WHERE is flagged. `MAX(c1)` in inner SELECT list is not.

#### Subquery — PASS

**MSSQL 27. IN subquery on non-indexed table — inner should not flag**

```sql
SELECT c1 FROM dbo.pokes
WHERE c1 IN (SELECT foo FROM dbo.pokes2 WHERE ABS(foo) > 0);
```

Expected: `PASS` — `pokes2` has no indexes.

**MSSQL 28. Inner subquery on non-indexed column**

```sql
SELECT c1 FROM dbo.pokes
WHERE c1 IN (SELECT order_id FROM dbo.orders_ms WHERE UPPER(note) = 'X');
```

Expected: `PASS` — `note` is not indexed on `orders_ms`.

#### CTE — WARN

**MSSQL 29. CTE body violation**

```sql
WITH filtered AS (
  SELECT c1, c2 FROM dbo.pokes WHERE ABS(c1) > 5
)
SELECT * FROM filtered WHERE c1 = 1;
```

Expected: `WARN`

**MSSQL 30. CTE body with calculation violation**

```sql
WITH calc AS (
  SELECT c1 FROM dbo.pokes WHERE c1 + 1 > 10
)
SELECT * FROM calc;
```

Expected: `WARN`

#### CTE — PASS

**MSSQL 31. CTE body without violation**

```sql
WITH clean AS (
  SELECT c1, c2 FROM dbo.pokes WHERE c1 = 1
)
SELECT * FROM clean;
```

Expected: `PASS`

#### UNION — WARN

**MSSQL 32. UNION with one violating branch**

```sql
SELECT c1 FROM dbo.pokes WHERE ABS(c1) > 5
UNION
SELECT c1 FROM dbo.pokes WHERE c1 = 1;
```

Expected: `WARN`

#### Edge — Function wrapping expression

**MSSQL 33. ABS(indexed_col + 1) — inner calculation detected**

```sql
SELECT c1 FROM dbo.pokes WHERE ABS(c1 + 1) > 5;
```

Expected: `WARN` — the walker descends into function args and finds the `c1 + 1` calculation.

**MSSQL 34. ABS(non_indexed_col + 1) — no indexed column**

```sql
SELECT foo FROM dbo.pokes WHERE ABS(foo + 1) > 5;
```

Expected: `PASS`

#### Edge — HAVING

**MSSQL 35. HAVING with aggregate arithmetic — not WHERE**

```sql
SELECT c1, AVG(c2) AS avg_c2
FROM dbo.pokes
GROUP BY c1
HAVING AVG(c2) * 3 + 1 > 50000;
```

Expected: `PASS`

---

## Behavior Notes

### Both engines

1. **Subquery scoping**: Each nested SELECT is checked against its own FROM-clause tables. Functions/calculations in inner subqueries are evaluated against the inner table's indexes, not the outer's.

2. **HAVING excluded**: Both implementations exclude HAVING from this rule. Aggregate functions in HAVING operate on grouped results, not base table rows, so indexes are irrelevant.

3. **CTE support**: CTE bodies are recursively checked with their own table scope.

4. **INSERT...SELECT / CREATE TABLE...AS SELECT**: The source SELECT is checked just like a standalone SELECT.

### MySQL-specific

5. **Function argument checking**: MySQL checks direct function arguments for indexed column references. For nested expressions like `ABS(id + 1)`, the walker descends into function args and the inner `id + 1` BinaryExpr triggers the calculation check.

### MSSQL-specific

6. **Scalar subquery scope isolation**: Functions in a subquery's SELECT list (e.g., `MAX(c1)` in `WHERE x > (SELECT MAX(c1) FROM t WHERE ...)`) are NOT flagged because the subquery enters a fresh WHERE/HAVING scope.

## Suggested Smoke Test Order

Quick sanity check covering all categories:

- MySQL 1, 9, 16, 25, 27, 31, 34, 37, 47, 54, 58, 61
- MSSQL 1, 7, 13, 18, 21, 26, 29, 33, 35

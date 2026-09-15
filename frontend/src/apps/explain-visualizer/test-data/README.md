# Plan test data

Real query plans, one file per plan shape, in the exact form Bytebase hands the
visualizer. The parser tests read these instead of hand-written plans so each
parser is pinned to what the engine actually emits — including the fields
nobody thinks to invent.

Keep them verbatim. Editing one to make a test pass defeats the point of
capturing it.

## `postgres/`

`EXPLAIN (FORMAT JSON)` output captured from PostgreSQL 17 against a seeded
schema: `customers` (5,000 rows) and `orders` (50,000 rows).

| File | Shape it covers |
| --- | --- |
| `seq-scan-filter.json` | A single node with a filter — the smallest real plan |
| `bitmap-index-scan.json` | An index path: bitmap heap scan over a bitmap index scan |
| `hash-join-aggregate-sort.json` | Hash join under a hashed aggregate under a sort |
| `cte-nested-loop-initplan.json` | A CTE, a nested loop, an InitPlan subquery and a limit |

To add one:

```bash
psql -At -c "EXPLAIN (FORMAT JSON) <query>" > postgres/<shape>.json
```

## `mssql/`

`SHOWPLAN_XML` output captured from SQL Server 2022 against the same schema,
with `orders_customer_id_idx` on `orders(customer_id)`. Each file is one
statement run with `SET SHOWPLAN_XML ON`, which is how the driver runs them.

The exception is `split-statements-batch.xml`, a SQL Server 2016 plan kept as
published: `Cursors/cursor2.sqlplan` from
[html-query-plan's test plans](https://github.com/JustinPealing/html-query-plan/tree/master/test_plans)
(MIT License). SQL Server 2016 gives each statement of a batch its own
`<Statements>` block, which SQL Server 2022 no longer does.

| File | Shape it covers |
| --- | --- |
| `index-seek-key-lookup.xml` | Nested loops over an index seek and a key lookup |
| `hash-join-aggregate-sort.xml` | Merge join, stream and hash aggregates, and a sort |
| `missing-index.xml` | A missing-index suggestion and a full scan of `orders` |
| `implicit-conversion-warning.xml` | Statement-level type conversion warnings |
| `no-join-predicate.xml` | An operator warning on a join without a condition |
| `if-else.xml` | An `IF` with its condition, `THEN` and `ELSE` |
| `procedure-two-statements.xml` | A procedure call holding two statements |
| `two-statement-batch.xml` | Two statements in one plan, under a batch |
| `static-cursor.xml` | A cursor's population and fetch queries |
| `scalar-udf.xml` | A call to a scalar function SQL Server doesn't inline, with the function's own statements |
| `unmatched-filtered-index.xml` | A procedure whose parameter keeps SQL Server from using a filtered index |
| `split-statements-batch.xml` | A batch of cursor statements, each in its own `<Statements>` block |

To add one, run the statement in `sqlcmd` and keep the `<ShowPlanXML>` document
from its output:

```bash
printf 'SET SHOWPLAN_XML ON;\nGO\n<statement>\nGO\n' | sqlcmd -S localhost -U sa -C -d plandb -y 0
```

## `spanner/`

The Spanner emulator does not plan queries, so these are production Spanner
plans from [spanner-cli's test data](https://github.com/cloudspannerecosystem/spanner-cli/tree/master/testdata/plans)
(Apache License 2.0), passed through the driver's `convertQueryPlanToJSON` so
each file holds the JSON Bytebase returns for that plan. Biome re-indents it,
as it does every JSON file here.

| File | Source | Shape it covers |
| --- | --- | --- |
| `hash-join.json` | `hash_join.input.json` | Build and probe inputs, full table and index scans |
| `filter-limit.json` | `filter.input.json` | Global and local limits, a seek condition |
| `scalar-subquery.json` | `scalar_subquery_with_filter_scan.input.json` | A scalar subquery with its own aggregates |
| `nested-array-subqueries.json` | `array_subqueries_with_compute_struct.input.json` | An array subquery inside another |

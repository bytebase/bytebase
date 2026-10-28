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

## `cockroachdb/`

Not an EXPLAIN format: CockroachDB has none that returns a document. These are
the payloads behind the diagram link `EXPLAIN (DISTSQL)` prints beside its text
plan, decoded by the driver, captured from CockroachDB 24.3. Processors with
the streams between them, and no estimate of any kind — the row counts are in
the text plan, not in the link.

| File | Shape it covers |
| --- | --- |
| `scan-filter-group.json` | A read, a filter and an aggregator in one chain |
| `index-join.json` | A bounded index read feeding a lookup join, each naming its own table |
| `sort-limit.json` | A top-K sorter over a read whose spans cover everything |

To add one, run `EXPLAIN (DISTSQL) <query>` through the driver with
`QueryContext{Explain: true}` and `ExplainFormat_JSON`, which is what decodes
the link.

## `mongodb/`

`explain()` output from MongoDB 5, captured through the driver so each file is
exactly what Bytebase hands the viewer. The database name in a namespace is the
one the test container generated, which is why it reads `test_1` rather than
anything meaningful — these are verbatim.

MongoDB's planner publishes no estimate at all, so the plans carry shape and
index usage; only `executionStats` adds numbers, and only by running the query.

| File | Shape it covers |
| --- | --- |
| `collection-scan.json` | A `COLLSCAN` with a filter — a query with no index to use |
| `index-scan-sort.json` | `SORT` over `FETCH` over `IXSCAN`, with the index bounds |
| `aggregate-group.json` | An aggregation's `stages` pipeline, whose `$cursor` holds the read's own plan |
| `measured-collection-scan.json` | `explain("executionStats")`: the same tree with rows returned and documents examined |
| `measured-aggregate-group.json` | A measured pipeline, where each stage states its rows and its spill counters beside itself rather than in its body |
| `slot-based-index-scan.json` | The slot-based engine's shape: the tree under `winningPlan.queryPlan`, beside the plan it compiled for itself |
| `sharded-index-scan.json` | A sharded collection: each shard's own plan under the `SHARD_MERGE` that reads them, each naming its shard and namespace |
| `measured-sharded-index-scan.json` | The same query run, where a shard's entry holds `executionStages` and states no namespace |
| `measured-slot-based-join.json` | A measured slot-based plan, whose `nlj` holds its two sides under `outerStage` and `innerStage` |
| `sharded-aggregate-group.json` | A pipeline split across shards: no `stages` of its own, each shard's part under `shards` and the merger's under `splitPipeline` |

The slot-based one is from MongoDB 7, which runs the classic engine by
default; `db.adminCommand({ setParameter: 1, internalQueryFrameworkControl:
"trySbeEngine" })` is what makes it choose the other one. The sharded pair is
from a cluster of two single-node shards behind a `mongos`, explained through
the router on a collection sharded by a hashed key.

To add one, run the statement through the driver with
`QueryContext{Explain: true}`, or in `mongosh`:

```
db.orders.find({ total: { $gt: 880 } }).explain()
```

## `mysql/`

`EXPLAIN FORMAT=JSON` output against the same schema, with
`orders_customer_id_idx` on `orders(customer_id)` and both tables analyzed.
MySQL 8.0 and MariaDB 11.8 put the same shape under different keys, so one
parser reads both and the `mariadb-` files are the second dialect's.

| File | Server | Shape it covers |
| --- | --- | --- |
| `full-scan-filter.json` | MySQL | A query block over one scan — the smallest real plan |
| `group-sort-join.json` | MySQL | An ordering over a grouping over a two-table join |
| `derived-table.json` | MySQL | A derived table materialized from its own query block |
| `attached-subquery.json` | MySQL | A real table's scan carrying a subquery's operators under `attached_subqueries`, which a derived table's `materialized_from_subquery` is not |
| `ordering-filesort.json` | MySQL | A filesort, whose `sort_cost` is the sort's alone where the block's `query_cost` is the whole |
| `union-distinct.json` | MySQL | A union result over two query specifications |
| `mariadb-sort-temporary-join.json` | MariaDB | `filesort` and `temporary_table` where MySQL says `ordering_operation` and `grouping_operation` |
| `mariadb-block-nl-join.json` | MariaDB | `block-nl-join`, a join wrapper holding the table it buffers |

To add one:

```bash
docker exec <container> mysql -uroot -p<password> <database> -N -B -r \
  -e "EXPLAIN FORMAT=JSON <query>" > mysql/<shape>.json
```

## `starrocks/`

`EXPLAIN` output from StarRocks 3.4 against the same schema. StarRocks rejects
every JSON spelling, so its text plan is what there is. A plan is a list of
fragments joined where one sends to another's exchange, and within a fragment
the operators are drawn as a tree, a second input introduced by `|----` and
drawn in a branch behind a bar.

`EXPLAIN COSTS` would state a cardinality on every operator rather than on the
scans alone, but it is a different plan from the one a plain EXPLAIN returns,
and the editor asks for the plain one.

| File | Shape it covers |
| --- | --- |
| `single-scan.txt` | Two fragments: a scan sending to the exchange that reads it |
| `scan-aggregate.txt` | A partial and a final aggregate either side of an exchange |
| `join-exchange.txt` | A join whose two inputs each arrive over their own exchange |
| `nested-join-branch.txt` | A join drawn inside another's `|----` branch, bars and all |

To add one:

```bash
docker exec <container> mysql -h127.0.0.1 -P9030 -uroot <database> -N -B -r \
  -e "EXPLAIN <query>" > starrocks/<shape>.txt
```

## `doris/`

`EXPLAIN` output from Apache Doris 3.1 against the same schema, read by the
StarRocks parser: StarRocks forked from Doris and their plans still agree on
the fragments and the tree. Doris rejects every JSON spelling too, so its text
plan is what there is.

What differs is how an operator spells itself. Doris prefixes the vectorized
operators with `V`, prints its own plan node id after the name, packs an
operator's numbers onto one line, qualifies a scanned table with its database,
and prints the statistics it planned from under a rule of equals signs.

| File | Shape it covers |
| --- | --- |
| `single-scan.txt` | Two fragments, and the `avgRowSize=0.0` Doris prints with no column statistics |
| `analyzed-join.txt` | The same tables once Doris had analyzed them, so it states rows and width — and groups a standalone cardinality by thousands |
| `scan-aggregate.txt` | A partial and a final aggregate either side of an exchange |
| `join-exchange.txt` | A broadcast join, and the statistics that follow the operator printed last |

To add one, the same client as above against a Doris frontend.

## `tidb/`

`EXPLAIN FORMAT=tidb_json` output from TiDB 8.5 against the same schema. TiDB
reports no cost in this format or any other the editor can ask for, so these
plans carry rows alone. It has printed these same keys since it added the
format in v6.5.0, so one capture stands for every release that answers at all.

| File | Shape it covers |
| --- | --- |
| `table-reader-selection.json` | A reader over a coprocessor selection over a full scan |
| `index-hash-join.json` | A join's build and probe sides, and an index lookup's own two reads |
| `union-hash-agg.json` | A union under a hash aggregate |
| `measured-hash-join.json` | `EXPLAIN ANALYZE`: actual rows, and the `executeInfo` string the parser does not read timings from |

To add one, `EXPLAIN FORMAT=tidb_json` through the same client as above.

## `oracle/`

Oracle states a plan as rows of `PLAN_TABLE`, so these are what the driver
renders those rows as — a JSON array, one object per operator, lower-cased
column names, columns the optimizer said nothing about left out. Captured from
Oracle Free 23ai against the same schema, with `orders_customer_id_idx` on
`orders(customer_id)` and the schema analyzed. Numbers arrive as strings
because that is how the Oracle driver returns `NUMBER`.

| File | Shape it covers |
| --- | --- |
| `full-scan-filter.json` | A statement over one full scan — the smallest real plan |
| `hash-join-group-sort.json` | An adaptive join: a hash join beside the nested loops and statistics collector Oracle keeps as the alternative |
| `index-range-scan.json` | A table read by rowid from an index range scan |
| `union-distinct.json` | `UNION-ALL` under the `HASH UNIQUE` that makes it distinct |

To add one, `EXPLAIN PLAN SET STATEMENT_ID = '<id>' FOR <query>` and then select
the columns `planStepsQuery` lists (see `backend/plugin/db/oracle/oracle.go`)
from `PLAN_TABLE` where `statement_id = '<id>'`, ordered by `id`.

## `clickhouse/`

`EXPLAIN json = 1, indexes = 1` output from ClickHouse 24.8 against an `orders`
table of 50,000 rows. ClickHouse publishes no cost or row estimate in any
EXPLAIN it offers, so these plans carry shape, descriptions and index
accounting alone.

| File | Shape it covers |
| --- | --- |
| `read-filter-aggregate.json` | The pipeline from a read up through an aggregate, a sort and a limit |
| `primary-key-range.json` | A read whose primary-key index entry reports the parts and granules it kept |
| `join-two-tables.json` | A join over a table read and a table function, which names no table |

To add one:

```bash
curl -s -u default:<password> --data-binary \
  "EXPLAIN json = 1, indexes = 1 <query> FORMAT TSVRaw" http://localhost:8123/ \
  > clickhouse/<shape>.json
```

`FORMAT TSVRaw` matters: the HTTP interface otherwise escapes the plan's
newlines into a TSV cell, which is not what the driver hands the viewer.

## `trino/`

`EXPLAIN (FORMAT JSON)` output from Trino 482 against the `tpch` connector's
`sf1` schema, which has statistics behind it. Trino returns the fragments it
would distribute, keyed by number, so these files are objects rather than
trees; the parser splices them at each `RemoteSource`.

| File | Shape it covers |
| --- | --- |
| `scan-filter-project.json` | A single fragment whose one operator fuses a scan, a filter and a projection, each with its own estimate |
| `join-aggregate-fragments.json` | Three fragments: a partial and a final aggregate either side of a join |
| `sort-limit.json` | Two fragments of partial top-N under a final one |

To add one:

```bash
docker exec <container> trino --server localhost:8080 --catalog tpch \
  --schema sf1 --output-format TSV --execute "EXPLAIN (FORMAT JSON) <query>"
```

The client returns it as a quoted, escaped TSV cell; unescape it before saving.

## `oceanbase/`

`EXPLAIN FORMAT=JSON` output from OceanBase CE 4.3 against the same schema,
with `orders_customer_id_idx` on `orders(customer_id)`. Each operator names its
children under `CHILD_1`, `CHILD_2`, and `EST.TIME(us)` covers an operator and
everything below it.

| File | Shape it covers |
| --- | --- |
| `full-scan-filter.json` | One operator — the smallest real plan |
| `hash-join-group-sort.json` | A sort over a group-by over a two-table hash join |
| `index-lookup.json` | A range scan, which is not a read of a whole table |
| `union-distinct.json` | A merge union with its two branches |

To add one, `EXPLAIN FORMAT=JSON <query>` through `obclient -N -B -r`.

## `redshift/`

Redshift's text plan, which is the only plan it publishes. Unlike every other
directory here these were not captured by this repository: Redshift runs
nowhere but a cluster, so the four `XN` plans are AWS's own documented output
over the TICKIT sample schema, which the affected-rows estimator in
`backend/plugin/db/redshift/stats.go` already pins against.

`postgres-same-grammar.txt` is the exception, and the point of it. Redshift
inherited PostgreSQL's text format and prefixed its operators with `XN`, so a
plan captured from a real PostgreSQL 17 server exercises the same indentation,
the same cost tuple and the same attribute lines — which is what checks the
grammar against output a server actually produced.

| File | Server | Shape it covers |
| --- | --- | --- |
| `hash-join-broadcast.txt` | Redshift (AWS docs) | A broadcast join over two scans, the whole plan indented one space |
| `three-way-join.txt` | Redshift (AWS docs) | A merge join under a hash join, with both distribution strategies |
| `hash-aggregate.txt` | Redshift (AWS docs) | An aggregate over a scan |
| `scan-filter.txt` | Redshift (AWS docs) | One scan and its filter |
| `postgres-same-grammar.txt` | PostgreSQL 17 | The same grammar without the `XN` prefix, captured here |

To add a PostgreSQL one:

```bash
psql -At -c "EXPLAIN <query>" > redshift/<shape>.txt
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

## `snowflake/`

`EXPLAIN USING JSON` output as Snowflake publishes it in
[its EXPLAIN documentation](https://docs.snowflake.com/en/sql-reference/sql/explain)
— the one fixture here not captured through the driver, because Snowflake runs
nowhere but Snowflake. What the driver hands over is the same single JSON value,
the one cell the statement returns. Snowflake estimates neither cost nor rows;
what it states is how much of each table the query would read.

| File | Shape it covers |
| --- | --- |
| `inner-join.json` | A join and its filter over two scans, each reporting the partitions and bytes assigned to it, with the query's totals in `GlobalStats` |

To add one, run `EXPLAIN USING JSON <statement>` on a Snowflake warehouse and
keep the cell verbatim. The cases a published plan does not show — a scan
assigned every partition of its table, a statement compiled into several steps —
are written inline in the test, which is where a plan no server emitted belongs.

## `hive/`

`EXPLAIN FORMATTED` output from Hive 4.0.0, HiveServer2 with its embedded
metastore, against `customers` and `orders`. The repository's formatter indents
the outer object; the plan the parser reads stays one escaped line within it,
the `CBOPlan` *string*, which is how Hive answers. It states that string only
where its cost-based optimizer planned the statement.

| File | Shape it covers |
| --- | --- |
| `single-scan.json` | A scan under a filter under a project — the chain Calcite writes by leaving `inputs` out |
| `join-aggregate.json` | A join naming both of its inputs, under an aggregate, on tables nothing had analyzed |
| `analyzed-join.json` | The same query once the tables held rows and `ANALYZE TABLE … COMPUTE STATISTICS FOR COLUMNS` had run, so every operator states rows and the scans state a row width |
| `union-all.json` | A union reading two branches |
| `order-limit.json` | A sort with a fetch |
| `no-cbo-plan.json` | `hive.cbo.enable=false`: the Tez stages alone, with no `CBOPlan`, which the viewer shows as text |

To add one, `EXPLAIN FORMATTED` through the same client.

## `bigquery/`

The stages of a BigQuery job as the driver encodes them: BigQuery's own REST
names for `ExplainQueryStage`, with its durations as milliseconds. BigQuery is
reachable from neither a dev box nor CI, so this is the driver's own output for
the stage list in `backend/plugin/db/bigquery/query_plan_test.go`, which holds
the shape a job reports — regenerate it from there rather than editing it. Every
plan here is measured, being the record of a run; BigQuery states none for a
statement it has not run.

| File | Shape it covers |
| --- | --- |
| `filtered-aggregate.json` | Two stages: the input reads a table, aggregates and writes a shuffle, and the output reads that shuffle and writes the query's rows |

## `databricks/`

`EXPLAIN FORMATTED` output from Spark 3.5.3, which is what Databricks SQL runs,
against `customers`, `orders` and a partitioned `events`. Databricks itself is
reachable from neither a dev box nor CI; the statement is identical and the
layout is Spark's own, though a Databricks runtime may name operators these
fixtures do not show. The plan states neither cost nor rows: `EXPLAIN COST`
reports those, and only for the logical plan.

| File | Shape it covers |
| --- | --- |
| `single-scan.txt` | A filter over a read, with the predicate pushed into it |
| `broadcast-join.txt` | A broadcast hash join under two aggregates and an exchange, with the build side's own exchange |
| `union-all.txt` | A union of two reads, the one tree with two branches |
| `sort-limit.txt` | A sort with its fetch, as `TakeOrderedAndProject` |
| `partition-filter.txt` | A read of a partitioned table, stating the partitions it skipped |
| `exists-subquery.txt` | An EXISTS turned into a left semi join |

To add one, `EXPLAIN FORMATTED <statement>` through the same client. Spark
suffixes each column with the expression id of the session that planned it, so
a re-capture renumbers them; the tests match the shape of those values rather
than the numbers.

## `elasticsearch/`

What a profiled search answers with, from Elasticsearch 8.15.0: the response's
own `took` beside its `profile`, which is what the driver hands over once it has
taken the profile out of the rows. Captured against an index of 60 documents and
a three-shard index of 30, both created for the capture.

| File | Shape it covers |
| --- | --- |
| `bool-aggregation.json` | One shard: a boolean query over a term and a range, the collectors that gathered it, a terms aggregation with an average nested inside, and the fetch's three phases |
| `multi-shard.json` | The same search across three shards, each profiled on its own |

To add one, send a search with `"profile": true` and keep the response's `took`
and `profile`. Every node is timed in nanoseconds and covers its own
subtree, so a re-capture renumbers every timing, and the tests that assert them
against this capture would need updating with it.


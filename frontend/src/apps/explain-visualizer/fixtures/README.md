# Plan fixtures

Real `EXPLAIN (FORMAT JSON)` output, captured from PostgreSQL 17 against a seeded
schema, one file per plan shape. The tests in this directory parse these instead
of hand-written JSON so the parser is pinned to what the server actually emits —
including the fields nobody thinks to invent, such as `Parent Relationship`,
`Subplan Name`, `Strategy` and `Partial Mode`.

| File | Shape it covers |
| --- | --- |
| `seq-scan-filter.json` | A single node with a filter — the smallest real plan |
| `bitmap-index-scan.json` | An index path: bitmap heap scan over a bitmap index scan |
| `hash-join-aggregate-sort.json` | Hash join under a hashed aggregate under a sort |
| `cte-nested-loop-initplan.json` | A CTE, a nested loop, an InitPlan subquery and a limit |

To add one, run the query against PostgreSQL and save the output verbatim:

```bash
psql -At -c "EXPLAIN (FORMAT JSON) <query>" > <shape>.json
```

Keep them verbatim. Editing one to make a test pass defeats the point of
capturing it.

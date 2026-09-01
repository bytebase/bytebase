# Bytebase storage package

This file provides additional guidance to AI coding assistants working under
`./backend/store/`, the mapping between Go and the metadata database.

## Inheritance

- Follow the repository-wide guidance in `../../AGENTS.md`.
- Treat this file as store-specific additions, not a replacement for the root instructions.

For schema update, please follow [Bytebase Schema Update Guide](https://github.com/bytebase/bytebase/blob/main/docs/schema-update-guide.md)

## Composite primary keys

Several tables use composite primary keys (e.g., `(project, id)`). Check
`backend/migrator/migration/LATEST.sql` for the full list — any table with a
multi-column PRIMARY KEY. `task_run_log` deliberately has no primary key (it is
an append-only log whose entries can share a `created_at` microsecond,
BYT-10035) but is equally project-scoped, so the same predicate rules apply to
it.

When writing or modifying queries on these tables:
- Every WHERE, JOIN, USING, DELETE, and UPDATE predicate must include every
  project/tenant scope column. Identify rows with either the full primary key or
  a full declared non-partial UNIQUE key that contains the same scope columns;
  verify alternate keys in `LATEST.sql`. Never filter by `id` or another locally
  unique identifier alone
- When adding a new store method touching a composite-PK table, add a
  corresponding `TestCollision_*` test in `backend/tests/`, built on
  `setupCollidingProjects` + `fixture.completeRolloutB` for setup and
  `snapshotProject` / `assertProjectUnchanged` for assertions — all going through
  the public gRPC API, no store access. Run with:
  `go test -v -count=1 ./backend/tests/ -run "^(TestClaim|TestCollision)" -timeout 5m`
- Which tables the shared snapshot already covers, and the two (`plan_webhook_delivery`,
  `sheet_blob_ref`) that need table-specific handling because they have no public
  read API, are listed in [`docs/pre-pr-checklist.md`](../../docs/pre-pr-checklist.md)
  step 3c. That is the copy kept in step with the fixture — read it there rather
  than trusting a restatement here

## Pagination ordering

Every paginated v1 list reads its pages with `LIMIT`/`OFFSET`: the page token
carries `{limit, offset}` (`backend/api/v1/common.go`), so each page is a
separate query against a table that other transactions are still writing.

PostgreSQL only guarantees the order the `ORDER BY` specifies. When the sort key
is not unique under the query's scope, tied rows may come back in a different
order in each of those queries — different effective `LIMIT` values pick top-N
heapsort or a full sort, and parallel gather-merge interleaves workers
independently. A tied row then crosses the page boundary between two reads, and
the caller skips it or receives it twice. This is not a concurrency artifact: it
happens on completely static data. Measured on 10,000 issues across 40 projects
by issuing the 100 page queries separately, exactly as the API does: ordering by
`issue.id` alone returned 10,000 rows of which only 9,764 were distinct — 236
duplicated and 236 never returned at all. Adding `issue.project` makes it
10,000 of 10,000.

**Every offset-paginated query must sort on a total order.** Write the clause
out — the tiebreak columns are part of the SQL, not something to build:

```go
// created_at defaults to now() and is not unique; resource_id is the primary key.
q.Space("ORDER BY changelog.created_at DESC, changelog.resource_id DESC")
```

Where a caller-supplied `order_by` leads, append the same columns after it:

```go
orderBy := []string{}
for _, v := range find.OrderByKeys {
    orderBy = append(orderBy, fmt.Sprintf("%s %s", v.Key, v.SortOrder.String()))
}
if len(orderBy) == 0 {
    orderBy = append(orderBy, "access_grant.created_at DESC")
}
orderBy = append(orderBy, "access_grant.id DESC")
q.Space("ORDER BY " + strings.Join(orderBy, ", "))
```

There is deliberately no shared helper for this. One existed and was removed in
review: it put four lines of ceremony on the ten lists whose clause is a
constant, and the SQL reads better written out. If a caller's key repeats a
tiebreak column, leave it — PostgreSQL ignores a redundant sort key.

Rules for choosing the tiebreak:

1. Take it from the table's primary key or a declared **non-partial** unique key
   in `LATEST.sql`. A partial unique index (`... WHERE deleted_at IS NULL`)
   constrains nothing outside its predicate, so it does not qualify whenever the
   query can return rows outside it.
2. Include **every** column of a composite key. `id` alone does not identify a
   row in a `(project, id)` table, and IDs are allocated per project by
   `nextProjectID` — every project's first row is `101` — so cross-project lists
   collide constantly. `(project, train, iteration)` needs all three.
3. A column is only a valid tiebreak if it is `NOT NULL`. `user_group.email` is
   nullable, so groups without one all tie.
4. `created_at` is never a tiebreak. It defaults to `now()`, which is the
   *transaction* timestamp, so every row written by one batch insert shares it
   exactly.
5. A mandatory `WHERE` predicate that pins a scope column to one value already
   supplies that column, so the rest of the key is total on its own and needs no
   tiebreak. `ListPlans` is total under `WHERE plan.project = ?` with nothing but
   `ORDER BY id DESC`, because `(project, id)` is the primary key. Naming the
   pinned column anyway is free in the plan — PostgreSQL folds a
   constant-equality column out of the pathkeys — but it is not a fix, and adding
   it to a query that is already total is churn. What matters is the reverse: if
   you ever relax such a predicate into a cross-scope list, the ordering stops
   being total that moment and the tiebreak becomes mandatory.
6. A user-supplied `order_by` replaces the default *sort key* — that is correct
   AIP-132 behavior — but it must never replace the *tiebreak*. Append the
   tiebreak after the caller's keys, as in the snippet above. `ListDatabases`,
   `ListInstances` and `ListProjects` each used to drop the whole clause,
   tiebreak included, the moment a caller passed `order_by`.
7. Check what the clause costs. Appending a tiebreak can turn an ordered index
   scan into an incremental sort, so `EXPLAIN` against the query's *real*
   predicate shape, not an idealized one, and keep the tiebreak's direction
   matching any index you rely on — `ListSavedQueries` does, so that
   `order_by: update_time desc` still reads
   `idx_saved_query_creator_updated_at_resource_id` in order. Then decide
   deliberately: widen the index with a migration, or accept the cost and say
   so. Accepted here: `SearchAuditLogs` costs ~16 ms more per 5000-row export
   page because `resource_id` falls outside
   `idx_audit_log_workspace_created_at`; judged not worth a migration.

### What is and is not enforced

`TestPaginationStabilityAcrossProjects` pages through a deliberately tied
cross-project issue list against a real PostgreSQL and asserts every row comes
back exactly once; `TestIssueCommentBatchKeepsInsertionOrder` pins the batch
ordering. Both fail against the pre-fix behavior. Add a case when a new list's
tiebreak is not obviously total.

Nothing checks this statically. An earlier revision routed all 17 lists through
a shared helper so an AST test could require it; review judged that ceremony,
the helper is gone, and the guard had no chokepoint left to police. So the rules above are
enforced by the pre-PR checklist and by review, not by a test — which means a
new paginated list added without a tiebreak will not fail CI. Step 4 of
[`docs/pre-pr-checklist.md`](../../docs/pre-pr-checklist.md) is the gate.

A total order fixes misordering, not drift: rows inserted or deleted between two
page reads still shift the offset window. Fixing that needs keyset pagination
and a page-token format change, which is out of scope here.

## Transaction row-lock ordering

PostgreSQL holds row locks until a transaction ends. Transactions that acquire the same locks in different orders can deadlock, so every store transaction must follow these rules:

1. Acquire transaction-scoped advisory locks before row locks.
2. Lock existing related rows from the deepest child to its parents. The project workflow chains are:
   - `issue_comment -> issue -> plan -> project`
   - `review_run -> issue -> plan -> project`
   - `plan_webhook_delivery -> plan -> project`
   - `plan_check_run -> plan -> project`
   - `task_run_log -> task_run -> task -> plan -> project`
   - `saved_query_organizer -> saved_query -> project`
   - `changelog -> sync_history -> db -> instance -> project`
   - `revision -> db -> instance -> project`
   - `db_schema -> db -> instance -> project`

   These workflow chains are writer-to-writer correctness invariants. The
   best-effort lifecycle policy below does not relax them; in particular,
   preserve `issue -> plan` and `plan_check_run -> plan` ordering.
3. Identify project-scoped rows with every scope column plus either the remaining primary-key columns or every remaining column of a declared non-partial unique key. Verify alternate keys in `LATEST.sql`. Lock batches in full primary-key order; project-scoped `(project, id)` batches therefore use that order, not `id` alone.
4. Treat locks acquired by `UPDATE`, `DELETE`, foreign-key checks, and `INSERT ... ON CONFLICT DO UPDATE` as part of the order. An upsert that can update an existing row is not a new-row-only insert.
5. `nextProjectID` locks `project` and requires it to be active before allocating an ID. Call it after locking any existing descendants, and do not lock an existing descendant afterward. Creation is rejected when the project is missing or deleted.

Row ordering prevents wait-for cycles on existing rows. Keep transactions short
and preserve this order whether locks are acquired explicitly or by `UPDATE` and
`DELETE` statements.

Project and instance archive and physical purge are point-in-time lifecycle
operations. Writers validate the current state required for their own work, but
do not coordinate with archive or purge. A rare concurrent writer may fail with
a PostgreSQL conflict or commit based on state it observed before the lifecycle
transition; these are accepted best-effort outcomes and the user may retry.

Examples of ordinary transaction ordering:

- Pending Task Run creation: existing `task` rows ordered by `(project, id)`, then `project`, then new `task_run` rows.
- Plan Check Run refresh: existing `plan_check_run`, then `plan`, then `project`, then the upsert.
- Issue creation: existing `plan`, then `project`, then the new `issue` row.
- Task skipping: existing `task` rows ordered by `(project, id)`; it does not lock `task_run` rows.

When adding or changing a transaction that coordinates multiple rows or tables,
add deterministic real-PostgreSQL regression tests for its ordinary contention
paths and assert the terminal outcomes.

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
- When adding a new store method touching a composite-PK table, add a corresponding
  `TestCollision_*` test in `backend/tests/`. The existing `setupCollidingProjects`
  fixture and `assertProjectUnchanged` helper cover `plan`, `issue`, `task`, `task_run`,
  `plan_check_run`, `task_run_log`, `db_group`, `release`, and `sheet_blob_ref` (the
  snapshot reads the last four through public APIs where one exists). `plan_webhook_delivery`
  has no public read API and uses a table-specific raw metadata-DB read; its rows are
  claimed asynchronously after rollout completion, so raw-read collision tests
  must stabilize before snapshotting and compare the table separately (see
  `backend/tests/README.md`). `sheet_blob_ref` also has no public read API and is
  read raw, but its rows are written synchronously, so `assertProjectUnchanged`
  compares them directly. For any future composite-PK table outside that set,
  write table-specific seed and assertion helpers — or extend the shared helper
  first
- Collision tests use `setupCollidingProjects` + `fixture.completeRolloutB` for setup
  and `snapshotProject` / `assertProjectUnchanged` for assertions — all going through
  the public gRPC API, no store access. Run with:
  `go test -v -count=1 ./backend/tests/ -run "^(TestClaim|TestCollision)" -timeout 5m`

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
happens on completely static data. Measured on 10,030 issues across 40 projects,
paging 100 at a time, ordering by `issue.id` alone duplicated 195 rows and
missed another 195.

**Every offset-paginated query must sort on a total order.** Build the clause
with `buildStableOrderBy` (`common.go`), passing the caller's sort keys plus
tiebreak columns that are unique under that query's scope:

```go
// created_at defaults to now() and is not unique; resource_id is the primary key.
q.Space(buildStableOrderBy(
    []*OrderByKey{{Key: "changelog.created_at", SortOrder: DESC}},
    "changelog.resource_id",
))
```

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
5. A scope column pinned to a single value by a mandatory `WHERE` predicate is
   already satisfied and need not be repeated — say so in a comment, as
   `ListPlans` does.
6. Sorting a user-supplied `order_by` must **add** to the default ordering, not
   replace it. Pass the caller's keys to `buildStableOrderBy` so the tiebreak is
   still appended; several lists previously dropped a correct default the moment
   a caller passed `order_by`.

`TestPaginatedListsUseStableOrderBy` fails any store function that applies
`OFFSET` without reaching `buildStableOrderBy`, directly or through a named
helper. It cannot check that the tiebreak columns you chose are actually unique
— that judgment is yours, against `LATEST.sql`.

Note that a total order fixes misordering, not drift: rows inserted or deleted
between two page reads still shift the offset window. Fixing that needs keyset
pagination and a page-token format change, which is out of scope here.

## Transaction row-lock ordering

PostgreSQL holds row locks until a transaction ends. Transactions that acquire the same locks in different orders can deadlock, so every store transaction must follow these rules:

1. Acquire transaction-scoped advisory locks before row locks.
2. Lock existing related rows from the deepest child to its parents. The project workflow chains are:
   - `issue_comment -> issue -> plan -> project`
   - `plan_webhook_delivery -> plan -> project`
   - `plan_check_run -> plan -> project`
   - `task_run_log -> task_run -> task -> plan -> project`
   - `saved_query_organizer -> saved_query -> project`
   - `changelog -> sync_history -> db -> instance -> project`
   - `revision -> db -> instance -> project`
   - `db_schema -> db -> instance -> project`
3. Identify project-scoped rows with every scope column plus either the remaining primary-key columns or every remaining column of a declared non-partial unique key. Verify alternate keys in `LATEST.sql`. Lock batches in full primary-key order; project-scoped `(project, id)` batches therefore use that order, not `id` alone.
4. Treat locks acquired by `UPDATE`, `DELETE`, foreign-key checks, and `INSERT ... ON CONFLICT DO UPDATE` as part of the order. An upsert that can update an existing row is not a new-row-only insert.
5. `nextProjectID` locks `project` and requires it to be active before allocating an ID. Call it after locking any existing descendants, and do not lock an existing descendant afterward. Creation is rejected when the project is missing or deleted.

Row ordering prevents wait-for cycles on existing rows. It cannot protect an
absent child row because there is no row to lock before a concurrent purge passes
that branch. The active-project check in `nextProjectID` covers this case only for
writers that call it; it is not a repository-wide purge fence because other
writers bypass `nextProjectID`.

Database creation, database-sync, batch database updates, task-run creation,
sheet creation, and Query History writers, together with direct instance
archive and direct project/instance purge, additionally take the matching
transaction-scoped purge fence before any row lock. This closes absent-descendant gaps; writers then
retain the normal child-to-parent row-lock order. Database sync may continue for
an archived project while its row exists, but never through a soft-deleted
instance. Direct instance archive and restore fail while any targeting task run
is pending, available, or running.

Every new or modified writer of purge-managed data must define its project
lifecycle policy: require an active project for new resources, or require only an
existing project when deleted-project continuation is intentional. Serialize and
validate that policy against project deletion before writing the managed data.

Transactions spanning project- or instance-owned sibling branches follow this canonical order:

```text
query_history -> policy -> saved_query_organizer -> saved_query
-> issue_comment -> issue -> plan_webhook_delivery -> plan_check_run
-> task_run_log -> task_run -> task -> plan -> access_grant -> release
-> db_group -> changelog -> sync_history -> revision -> db_schema -> db
-> sheet_blob_ref -> project_webhook -> service_account -> workload_identity -> instance -> project
```

Update this list, `DeleteProject`, and `DeleteInstance` together. A transaction that needs another sibling branch must establish its position here before implementation. When one table is touched by multiple predicates, keep those mutations contiguous at that table's position. Keep transactions short and preserve this order whether locks are acquired explicitly or by `UPDATE` and `DELETE` statements.

Examples:

- Pending Task Run creation: existing `task` rows ordered by `(project, id)`, then `project`, then new `task_run` rows.
- Plan Check Run refresh: existing `plan_check_run`, then `plan`, then `project`, then the upsert.
- Issue creation: existing `plan`, then `project`, then the new `issue` row.
- Task skipping: existing `task` rows ordered by `(project, id)`; it does not lock `task_run` rows.

When adding or changing a transaction that coordinates multiple rows or tables,
add deterministic real-PostgreSQL regression tests for both lock-acquisition
directions. Assert the terminal outcomes, including that neither direction ends
in a foreign-key failure; merely checking for the absence of SQLSTATE `40P01` is
insufficient.

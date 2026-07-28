# Bytebase storage package

For schema update, please follow [Bytebase Schema Update Guide](https://github.com/bytebase/bytebase/blob/main/docs/schema-update-guide.md)

## Transaction row-lock ordering

PostgreSQL holds row locks until a transaction ends. Transactions that acquire the same locks in different orders can deadlock, so every store transaction must follow these rules:

1. Acquire transaction-scoped advisory locks before row locks.
2. Lock existing related rows from the deepest child to its parents. The project workflow chains are:
   - `issue_comment -> issue -> plan -> project`
   - `plan_webhook_delivery -> plan -> project`
   - `plan_check_run -> plan -> project`
   - `task_run_log -> task_run -> task -> plan -> project -> instance`
   - `worksheet_organizer -> worksheet -> project`
   - `changelog -> sync_history -> db -> instance`
   - `revision -> db -> instance`
   - `db_schema -> db -> instance`
3. Identify project-scoped rows with every scope column plus either the remaining primary-key columns or every remaining column of a declared non-partial unique key. Verify alternate keys in `LATEST.sql`. Lock batches in full primary-key order; project-scoped `(project, id)` batches therefore use that order, not `id` alone.
4. Treat locks acquired by `UPDATE`, `DELETE`, foreign-key checks, and `INSERT ... ON CONFLICT DO UPDATE` as part of the order. An upsert that can update an existing row is not a new-row-only insert.
5. `nextProjectID` locks `project` and requires it to be active before allocating an ID. Call it after locking any existing descendants, and do not lock an existing descendant afterward. Creation is rejected when the project is missing or deleted.

Row ordering prevents wait-for cycles on existing rows. It cannot protect an
absent child row because there is no row to lock before a concurrent purge passes
that branch. The active-project check in `nextProjectID` covers this case only for
writers that call it; it is not a repository-wide purge fence because other
writers bypass `nextProjectID`.

Every new or modified writer of purge-managed data must define its project
lifecycle policy: require an active project for new resources, or require only an
existing project when deleted-project continuation is intentional. Serialize and
validate that policy against project deletion before writing the managed data.

Transactions spanning project- or instance-owned sibling branches follow this canonical order:

```text
query_history -> policy -> worksheet_organizer -> worksheet
-> issue_comment -> issue -> plan_webhook_delivery -> plan_check_run
-> task_run_log -> task_run -> task -> plan -> access_grant -> release
-> db_group -> changelog -> sync_history -> revision -> db_schema -> db
-> project_webhook -> service_account -> workload_identity -> project -> instance
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

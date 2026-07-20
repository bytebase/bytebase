# Bytebase storage package

For schema update, please follow [Bytebase Schema Update Guide](https://github.com/bytebase/bytebase/blob/main/docs/schema-update-guide.md)

## Transaction row-lock ordering

PostgreSQL holds row locks until a transaction ends. Transactions that acquire the same locks in different orders can deadlock, so every store transaction must follow these rules:

1. Acquire transaction-scoped advisory locks before row locks.
2. Lock existing related rows from the deepest child to its parents. The project workflow chains are:
   - `issue_comment -> issue -> plan -> project`
   - `plan_webhook_delivery -> plan -> project`
   - `plan_check_run -> plan -> project`
   - `task_run_log -> task_run -> task -> plan -> project`
3. Identify project-scoped rows with every scope column plus either the remaining primary-key columns or every remaining column of a declared non-partial unique key. Verify alternate keys in `LATEST.sql`. Lock batches in full primary-key order; project-scoped `(project, id)` batches therefore use that order, not `id` alone.
4. Treat locks acquired by `UPDATE`, `DELETE`, foreign-key checks, and `INSERT ... ON CONFLICT DO UPDATE` as part of the order. An upsert that can update an existing row is not a new-row-only insert.
5. `nextProjectID` locks `project`. Call it after locking any existing descendants, and do not lock an existing descendant afterward.

Transactions spanning project-owned sibling branches follow the order used by `DeleteProject`:

```text
query_history -> policy -> worksheet (project reassignment)
-> issue_comment -> issue -> plan_webhook_delivery -> plan_check_run
-> task_run_log -> task_run -> task -> plan -> access_grant -> release
-> db_group -> db (project reassignment) -> project_webhook
-> worksheet_organizer -> worksheet (creator cleanup) -> revision
-> service_account -> workload_identity -> project
```

Update this list and `DeleteProject` together. A transaction that needs another sibling branch must establish its position here before implementation. Keep transactions short and acquire every required lock before performing work that depends on the protected state.

Examples:

- Pending Task Run creation: existing `task` rows ordered by `(project, id)`, then `project`, then new `task_run` rows.
- Plan Check Run refresh: existing `plan_check_run`, then `plan`, then `project`, then the upsert.
- Issue creation: existing `plan`, then `project`, then the new `issue` row.
- Task skipping: existing `task` rows ordered by `(project, id)`; it does not lock `task_run` rows.

When adding or changing a transaction that coordinates multiple rows or tables, add a deterministic real-PostgreSQL regression test that exercises its competing transaction path and fails on a deadlock.

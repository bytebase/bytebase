# Project and instance purge concurrency

## Decision

Treat physical project and instance purge as **best-effort maintenance**: if a
normal write is already operating on the same lifecycle root, purge should fail
fast with a retryable error. Do not, however, remove all common-path locking and
hope that foreign keys will make the rare operation lose. PostgreSQL does not
guarantee which overlapping transaction wins, and Bytebase has lifecycle rules
and soft references that foreign keys cannot enforce.

The recommended design is one shared/exclusive lifecycle gate per project or
instance:

- A normal transaction takes a transaction-scoped **shared advisory try-lock**
  once, then validates the required lifecycle state (active, or merely
  existing).
- Archive, restore, and purge take the corresponding **exclusive try-lock**.
  Either side returns `common.Conflict` with `resource is busy; retry` without
  changing anything when its try-lock cannot be acquired. RPCs expose this as
  Connect `Aborted`.
- Once every purge-managed writer uses that gate, remove explicit row locks whose
  only purpose was coordination with purge. Keep locks needed for ordinary
  common/common concurrency, allocation, state transitions, or queue claiming.

Acquire and deduplicate all project gates in sorted order before all instance
gates in sorted order. A project-instance transition therefore holds its owning
project's shared gate before its own exclusive gate.

This concentrates the defensive policy in transaction helpers while preserving
per-project and per-instance concurrency. It also gives the intended bias: an
already-running normal write holding the shared gate makes the rare purge lose.
If purge acquires the gate first, later writes fail fast and retry after purge;
no mechanism can guarantee that a transaction which
has not yet announced itself always beats one already performing the purge.

## Starting point

Purge is already an archive-then-hard-delete operation. The API rejects project
and instance purge unless the root is soft-deleted
([project service](../../backend/api/v1/project_service.go#L399-L417),
[instance service](../../backend/api/v1/instance_service.go#L973-L983)). The store
then performs a large atomic transaction. Project purge deletes some descendants,
reassigns workspace-instance databases to the default project, deletes project
instances, and finally deletes the project
([project purge](../../backend/store/project.go#L448-L464),
[database handling](../../backend/store/project.go#L689-L735),
[root deletion](../../backend/store/project.go#L779-L856)). Instance purge has a
similar descendants-first transaction
([instance purge](../../backend/store/instance.go#L850-L870),
[root deletion](../../backend/store/instance.go#L1003-L1053)).

Before this change, the protocol was deliberately repository-wide. The store documentation
defines child-to-parent row ordering, a total order across sibling branches, and
requires two-direction real-PostgreSQL tests for each coordinated path
([store locking rules](../../backend/store/AGENTS.md#transaction-row-lock-ordering)). Explicit purge
fences exist because an absent descendant cannot be row-locked before a writer
inserts it. Database writers therefore acquire project and instance advisory
locks, lock descendants and roots, revalidate ownership, and only then write
([database purge fence](../../backend/store/database.go#L753-L820)). Sheet and
Task Run creation show the same policy distributed through otherwise ordinary
write code
([sheet creation](../../backend/store/sheet.go#L283-L294),
[Task Run creation](../../backend/store/task_run.go#L353-L427)).

Not every `SELECT ... FOR UPDATE` in the package is purge defense. For example,
`nextProjectID` locks the project and then calculates `MAX(id) + 1`
([allocator](../../backend/store/id.go#L18-L65)); removing that lock without
replacing the allocator would allow concurrent creators to choose the same ID.
Likewise scheduler claims, read-modify-write updates, and idempotent upserts need
their own concurrency analysis. The removable set must be identified by purpose,
not syntax. A separate allocator migration could use a database sequence (if
globally increasing IDs are acceptable) or a counter row keyed by project and
table. The latter still locks, but it localizes an ordinary allocation invariant
instead of using the project lifecycle row for two unrelated jobs.

## Assessment of “let purge fail and ask the user to retry”

The idea is sound as the **maintenance contract**, with three qualifications.

### 1. Foreign keys preserve physical integrity, but do not choose purge as loser

Most project- and instance-owned rows have foreign keys to their roots or an
ancestor. For example, `plan.project`, `db.project`, `db.instance`, and
`task.instance` are constrained
([schema](../../backend/migrator/migration/LATEST.sql#L280-L311),
[database and instance schema](../../backend/migrator/migration/LATEST.sql#L496-L537),
[task schema](../../backend/migrator/migration/LATEST.sql#L600-L638)). A foreign
key guarantees that a referencing value matches a referenced row; the default
`NO ACTION` deletion normally fails if the relationship is left broken, while
`CASCADE` deletes components automatically
([PostgreSQL constraints](https://www.postgresql.org/docs/current/ddl-constraints.html#DDL-CONSTRAINTS-FK)).

PostgreSQL implements a foreign-key reference check by locking the referenced row
`FOR KEY SHARE`
([PostgreSQL `ri_triggers.c`](https://github.com/postgres/postgres/blob/master/src/backend/utils/adt/ri_triggers.c#L550-L624)).
That conflicts with the `FOR UPDATE` lock acquired by `DELETE`; row locks last to
transaction end, and `DELETE` already acquires `FOR UPDATE`
([PostgreSQL row-level locks](https://www.postgresql.org/docs/current/explicit-locking.html#LOCKING-ROWS)).
Thus a child insert and parent delete cannot both commit with a dangling foreign
key. But ordering still matters: an insert that reaches the reference first can
make delete wait or fail, while a delete that reaches the root first can make the
insert wait and then fail. Foreign keys provide arbitration, not “purge always
loses.”

For the current manual descendants-first purge, a child inserted after that
child table was swept can cause the final parent delete to fail with
`foreign_key_violation`; because purge is one transaction, the cleanup rolls
back, and a later attempt can see and delete the new child. That is a valid
retry model for *fully constrained physical relationships*.

### 2. Lifecycle state and soft references are outside that protection

A foreign key checks existence, not Bytebase's `deleted` flag. PostgreSQL's
`FOR KEY SHARE` does not conflict with an ordinary non-key update, which takes
`FOR NO KEY UPDATE`
([lock conflict table](https://www.postgresql.org/docs/current/explicit-locking.html#LOCKING-ROWS)).
Consequently an application that reads `deleted = false` without a conflicting
gate can commit a child after another transaction archives the root. That may be
referentially valid but violate the product lifecycle rule. If the product
explicitly accepts optimistic semantics, an ordinary write that actually commits
before archive can be linearized before it; a write that reaches the lifecycle
gate after archive must be rejected. The gate defines that boundary without
requiring every child row to participate in the purge's lock order.

Bytebase also has references that are intentionally not foreign keys. A policy
stores a project resource name as text
([policy schema](../../backend/migrator/migration/LATEST.sql#L108-L131)); Query
History has an FK to its project but stores the database resource name as text
([Query History schema](../../backend/migrator/migration/LATEST.sql#L471-L484)).
`revision.deleter` is another text reference: it has no FK to a principal
([revision schema](../../backend/migrator/migration/LATEST.sql#L540-L551)), and
project purge explicitly nulls values belonging to principals it will delete
([cleanup](../../backend/store/project.go#L692-L703)). Revision creation is behind
the database lifecycle gate, and `DeleteRevision` now uses the same gate before updating `deleter`
([revision writers](../../backend/store/revision.go#L150-L192)). This does not
prove the latter is a bug—the intended retention semantics need a decision—but it
does prove that an FK inventory alone is insufficient.
Instance purge cleans Query History with a string predicate
([instance purge](../../backend/store/instance.go#L864-L870)), while Query History
creation participates in the database lifecycle gate
([writer](../../backend/store/query_history.go#L104-L113)). If that gate were
removed, a concurrent history insert could survive instance purge without
violating any FK. JSON payload references and cache invalidation create similar
application-level obligations. Retrying only on FK errors cannot detect them.

### 3. The API must expose a retryable result, not `Internal`

Project and instance purge previously translated every store failure to Connect
`CodeInternal`
([project](../../backend/api/v1/project_service.go#L414-L417),
[instance](../../backend/api/v1/instance_service.go#L980-L983)). A deliberate
maintenance collision should instead be a stable conflict/aborted result with a
message `resource is busy; retry`. PostgreSQL recommends testing
SQLSTATE rather than localized error text. Relevant codes are `40001`
`serialization_failure`, `40P01` `deadlock_detected`, `55P03`
`lock_not_available`, and `23503` `foreign_key_violation`
([PostgreSQL error codes](https://www.postgresql.org/docs/current/errcodes-appendix.html)).

Only the first three are generically transient in this context. PostgreSQL says
serialization failures require retrying the **complete transaction**, including
the logic that chose its SQL and values
([serialization failure handling](https://www.postgresql.org/docs/current/mvcc-serialization-failure-handling.html)).
Treat `23503` as retryable only when the failing constraint and operation prove
it is the known purge race; otherwise it is persistent bad data or a purge bug.

## Options

| Option | Normal-path cost | Scope of coordination | Assessment |
| --- | --- | --- | --- |
| Keep current row-order protocol | Multiple locks, ordering rules, and tests in normal writers | Target rows, project, and instance | Correct, but high cognitive cost; retains locks that also serve unrelated purposes. |
| Rely only on FKs and retry purge | Lowest | Enforced physical relationships only | Acceptable only for a small, audited subset. Does not enforce active/archived state or clean soft references; either transaction can lose. |
| Shared/exclusive advisory lifecycle gate | One shared gate plus lifecycle validation per transaction | One project or instance | Recommended. Centralizable, targeted, and explicitly biases an already-running normal write over purge. |
| Rare-side table locks with `NOWAIT` | No common-path protocol | Every writer to every locked table | Technically simple but too broad by default: one project purge can stall unrelated projects. Useful only if global maintenance quiescence is acceptable. |
| Tombstone plus asynchronous garbage collection | Lifecycle check on normal writes; final sweep is off the request path | One archived root, but finalization still needs a gate | Good UX and naturally retryable. A grace period reduces overlap but cannot prove quiescence; the final collector still needs a gate or broad table locks, especially for intentional archived-root continuation. |
| `SERIALIZABLE` transactions | Dependency tracking and whole-transaction retry on every participant | Every participating serializable transaction | Not a drop-in. Making only purge serializable does not establish the lifecycle contract with Read Committed writers. |
| More `ON DELETE CASCADE` | Less manual physical cleanup | FK ownership graph | Worth considering for true components, but cannot express reparenting, soft references, lifecycle policy, or cache publication. |

PostgreSQL advisory locks have application-defined meaning, so all relevant paths
must cooperate. Transaction-level variants are released automatically at
transaction end
([PostgreSQL advisory locks](https://www.postgresql.org/docs/current/explicit-locking.html#ADVISORY-LOCKS)).
The `pg_try_advisory_xact_lock` family returns immediately instead of waiting,
and shared variants allow normal writers to coexist while conflicting with an
exclusive purge gate
([advisory lock functions](https://www.postgresql.org/docs/current/functions-admin.html#FUNCTIONS-ADVISORY-LOCKS)).
This is a direct refinement of Bytebase's existing project/instance advisory
namespaces
([current definitions](../../backend/store/advisory_lock.go#L29-L49)), rather
than a new coordination mechanism.

A rare-side table-lock design can use `LOCK TABLE ... IN SHARE ROW EXCLUSIVE
MODE NOWAIT` in a fixed order before cleanup. That mode conflicts with ordinary
writers but permits plain readers; `NOWAIT` makes existing contention fail
immediately
([PostgreSQL table locks](https://www.postgresql.org/docs/current/explicit-locking.html#LOCKING-TABLES)).
After it succeeds, however, later writes to any locked table wait until purge
ends, including writes for unrelated projects and instances. This trades code
complexity for a potentially large availability blast radius.

`SET LOCAL lock_timeout` is a useful purge-only safety net because it aborts a
statement that waits too long for explicit or implicit locks
([PostgreSQL `lock_timeout`](https://www.postgresql.org/docs/current/runtime-config-client.html#GUC-LOCK-TIMEOUT)).
It is not the primary correctness mechanism: timeout is per lock acquisition,
does not identify the business resource causing contention, and cannot detect a
concurrent soft-reference insert that takes no conflicting lock.

## Recommended migration

1. **Write down lifecycle policy by writer.** Classify every project/instance
   writer as requiring an active root, requiring an existing root (for deliberate
   archived-project continuation), or independent of purge. Include soft
   references, not only FK descendants.
2. **Introduce transaction wrappers.** At transaction start, acquire shared
   project gates in sorted order, then shared instance gates in sorted order, and
   validate lifecycle state after acquisition. Keep the ordering already used by
   the store. Normal writers for one root no longer exclude one another merely
   because purge is possible.
3. **Make lifecycle transitions try-exclusive and user-retryable.** Archive and
   restore need the exclusive gate too; otherwise a shared normal writer can
   still cross the `deleted` state change. Purge acquires it before any row work.
   Failure maps to Connect `Aborted`, not `Internal`. Optionally add a short
   transaction-local `lock_timeout` as defense against unmodeled locks. Purge
   remains non-idempotent: an already-missing target returns `NotFound`.
4. **Delete locks by proven purpose.** Remove explicit child/root locks only when
   they exist solely for purge serialization and every relevant writer is behind
   the lifecycle gate. Retain `nextProjectID` until its `MAX(id) + 1` allocator is
   replaced, and retain locks for common/common atomicity.
5. **Simplify physical ownership separately.** Consider `ON DELETE CASCADE` only
   for tables that are true components and have no reparenting or retention
   semantics. Keep explicit handling for saved queries, workspace-instance
   databases, principals, policies, Query History, JSON references, and caches.
6. **Test the contract, not the implementation.** For both acquisition orders,
   assert: an already-running normal write makes purge return retryable conflict;
   purge-first prevents resurrection; retry eventually succeeds; no orphan or
   soft reference survives; and unrelated projects/instances continue. Keep
   separate concurrency tests for allocator and state-transition locks.

Keep the synchronous RPC shape. `BatchDeleteProjects(purge=true)` validates
existence, permissions, and archived state for every target before deleting any,
then purges sequentially. A runtime failure may leave earlier projects deleted;
the caller retries only the remaining targets.

The lifecycle module owns transaction cleanup, advisory-key construction,
sorting, deduplication, stricter state merging, state validation, and conflict
normalization. Callers supply only their lifecycle roots and transactional work.
The cutover is coordinated across replicas; mixed old/new replicas are not
supported. It requires no schema migration, feature flag, background job, or
automatic server retry.

This approach accepts the user's core premise—the rare operation should pay and
should be retried—without weakening Bytebase's governed lifecycle invariants or
turning expected contention into an opaque internal error.

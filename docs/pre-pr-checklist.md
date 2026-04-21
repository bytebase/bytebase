# Pre-PR Checklist

Run through every section below before creating a pull request. Each section has a
skip condition — check it first to avoid unnecessary work.

## 1. Determine What Changed

```bash
git diff main...HEAD --stat
git diff main...HEAD
```

Use the diff output to decide which sections below apply.

## 2. Breaking Change Check

**Skip if:** diff only touches test files, docs, or comments.

Review the diff against each category. A "breaking change" is anything that could
cause existing users, integrations, or deployments to fail after upgrading.

| # | Category | What to look for |
|---|----------|-----------------|
| 1 | API | Removed/renamed endpoints, changed request/response formats, removed query params |
| 2 | Database schema | Dropped columns/tables, non-backward-compatible migrations |
| 3 | Proto | Removed/renamed fields, changed field numbers/types, removed RPCs |
| 4 | Configuration | Removed flags, changed defaults, renamed env vars |
| 5 | Behavior | Changed default values, altered workflows, modified permission logic |
| 6 | Webhooks/events | Renamed/removed events, changed payload formats |
| 7 | UI workflows | Redesigned user-facing flows that change how users perform existing tasks |
| 8 | Composite-PK migration | Adding a composite PK to an EXISTING table, or changing the PK columns of an existing table. Does NOT apply to: new queries (those are bugs — block via section 3), or new tables with composite PKs (those are additive). |

**If ANY apply:**
- Add `--label breaking` to the `gh pr create` command
- Include a `## Breaking Changes` section in the PR body

**Common NOT-breaking scenarios** (do not add the breaking label):
- A new method/query on a composite-PK table (even if buggy — block via section 3 instead)
- A new table added via migration (even with composite PK — it's additive)
- A bug fix that adds missing PK columns to an existing query (it restores correctness)

## 3. Composite-PK Query Safety

**Skip if:** diff does not touch `backend/store/` or `backend/migrator/`.

Composite primary keys (e.g., `(project, id)`) mean that `id` alone is NOT unique.
Filtering by `id` without `project` causes cross-project data corruption — the exact
bug class behind BYT-9259 (customer data loss from silent task re-execution).

### Step 3a: Identify composite-PK tables in the diff

Read `backend/migrator/migration/LATEST.sql` and find every table with a multi-column
`PRIMARY KEY`. The known project-scoped set (as of April 2026) includes:

- `plan (project, id)`
- `plan_check_run (project, id)`
- `plan_webhook_delivery (project, plan_id)`
- `issue (project, id)`
- `task (project, id)`
- `task_run (project, id)`
- `task_run_log (project, task_run_id, created_at)`
- `db_group (project, resource_id)`
- `release (project, train, iteration)`

Always verify against LATEST.sql — tables may have been added since this list was
last updated. Cross-reference with the tables your diff touches.

### Step 3b: Verify predicates

For every query in the diff that touches a composite-PK table, verify:

- Every `WHERE` clause includes ALL primary key columns
- Every `JOIN ... ON` includes ALL primary key columns
- Every `DELETE ... USING` includes ALL primary key columns
- Every `UPDATE ... FROM` includes ALL primary key columns

**Red flag pattern:** `WHERE id = ?` without `AND project = ?` on any of these tables.

**STOP — do not proceed to PR creation if any predicate is missing a PK column.**
This is the exact bug pattern that caused BYT-9259. Fix the query first.

### Step 3c: Verify collision test coverage

If the diff adds or modifies a store method that touches a composite-PK table:

1. Check if a corresponding `TestCollision*` or `TestClaim*` test exists in `backend/tests/`
2. If not, add one using `setupCollidingProjects` and `assertProjectUnchanged` from
   `backend/tests/collision_helper_test.go`. Note: the shared snapshot currently
   covers `plan`, `issue`, `task`, `task_run`, and `plan_check_run`. For methods
   touching `task_run_log`, `plan_webhook_delivery`, `db_group`, or `release`,
   add table-specific assertions inline — the shared helper is not sufficient
3. If your test needs project B rolled out (to create task/task_run/plan_check_run
   rows that could collide with project A's), call `fixture.completeRolloutB(ctx, t, ctl)`
   — this is the ONLY supported rollout path and it proves `task` and `task_run`
   id collisions automatically. (Plan-check-run id collision is NOT proven —
   the v1 API uses a UID-less name for PCRs, so the collision can't be observed
   from public gRPC. The PCR claim test is belt-and-suspenders coverage; the
   load-bearing regression lock is the task_run claim test.) Do NOT hand-roll
   `CreateRollout` + `waitRollout` — the collision invariant must not be a
   per-test responsibility
4. If testing delete cascades across projects where both projects share an
   instance, also consider adding a variant using `setupCollidingProjectsSeparateInstances`
   to catch cross-project over-delete bugs that shared-instance tests cannot detect
5. **Every cross-project / isolation test must have a positive precondition.**
   `assert(rows belong to project A)` is vacuously true when the list is
   empty. Add `Greater(len, 0, ...)` for the list under test before iterating
   so that an over-filtering regression cannot pass silently.

If the diff adds a NEW composite-PK table via migration:

1. Add collision test coverage for every store method touching the new table
2. Update the table list in step 3a above

**STOP — do not proceed to PR creation until collision tests exist for every new or
modified store method on a composite-PK table.** Write the tests before continuing.

### Step 3d: Run collision tests

Only run after steps 3b and 3c are resolved.

```bash
go test -v -count=1 ./backend/tests/ -run "^(TestClaim|TestCollision)" -timeout 5m
```

All must pass before proceeding.

### Step 3e: Doc-code drift check

**Skip if:** the diff didn't remove or rename any exported symbol or any
documented helper.

If you removed `Server.StoreForTest()`, renamed `assertFooCollide`, etc.,
grep for stale references that would now lie to the reader:

```bash
git diff main...HEAD --name-only -- '*.go' | xargs -I{} grep -l 'OldSymbolName' AGENTS.md docs/ backend/ 2>/dev/null
```

For each match, either delete the prose or update it to reference the
current API. AGENTS.md and `docs/pre-pr-checklist.md` are the highest-leak
spots — they document what helpers exist and what guarantees they make.
Stale references don't break the build but they actively mislead future
contributors and AI agents that read these docs at session start.

## 4. Lint and Format Gate

**Skip if:** no code changes (docs-only PR).

Run the checks relevant to the files you changed:

**Go changes:**
```bash
gofmt -w <changed .go files>
golangci-lint run --allow-parallel-runners
```
Run golangci-lint repeatedly until zero issues (the linter has a max-issues limit).

**Frontend changes:**
```bash
pnpm --dir frontend check
pnpm --dir frontend type-check
```

**Proto changes:**
```bash
buf lint proto
```

## 5. Test Gate

**Skip if:** no code changes.

- Run tests for every changed package
- For store changes touching composite-PK tables, run the collision tests (section 3d)
- For Go changes: `go build -ldflags "-w -s" -p=16 -o ./bytebase-build/bytebase ./backend/bin/server/main.go`
- For new migration files: update `TestLatestVersion` in `backend/migrator/migrator_test.go`

## 6. SonarCloud Properties

**Skip if:** no new files or directories added.

Update `.sonarcloud.properties` to reflect the latest file structure:
- `sonar.exclusions` for generated code, build artifacts, dependencies (directory paths)
- `sonar.test.inclusions` for test file patterns (e.g., `**/*_test.go`)
- `sonar.cpd.exclusions` to skip copy-paste detection on test files

## 7. Final Verification

Before running `gh pr create`:

- [ ] All checks above passed or were correctly skipped
- [ ] PR description clearly describes what changed and why
- [ ] Breaking changes are labeled and documented (if applicable)
- [ ] No unrelated files are staged

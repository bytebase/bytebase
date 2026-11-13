# What a rollout job reports

Status: the action change is implemented in this change, and the generator change follows ·
2026-11-12 ·
[BYT-10246](https://linear.app/bytebase/issue/BYT-10246/return-rollout-execution-results-and-affected-row-counts-from-bytebase)

A deploy job runs `bytebase-action rollout` and tells the developer one thing: pass or fail. What
ran, what it changed and why it failed are in Bytebase.
**`bytebase-action rollout` builds one result from the rollout and its task-run logs, and renders
it as the `--output` file, a block in the job log and the CI's summary.**

## Problem

Verified on `main` at `f4937cda3a`:

- **A failure names nothing.** The job ends with `found failed tasks. view on Bytebase`
  (`waitForRollout` in `action/command/rollout.go`). The database, the file, the statement and the
  error are missing.
- **`--output` holds three names.** `writeOutputJSON` in `action/command/output/output.go` writes
  the release, the plan and the rollout.
- **The GitHub summary stops at the stage.** `BuildSummaryMarkdown` in `action/github/summary.go`
  prints one status per stage. A stage with nine done tasks and one skipped reads `Skipped`.
- **GitLab gets nothing.** Only `check` writes a report there.
- **The server already has the answer.** Each task run's log records every command, its error and
  its affected rows (`TaskRunLogEntry.CommandExecute` in `proto/v1/v1/rollout_service.proto`). The
  action never reads it, so a user who wants it in the pipeline writes an API polling script.
- **The log alone does not name the statement.** For a release file it records a byte range of the
  file, not the text (`LogCommandExecute` in `backend/plugin/db/driver.go`).

## Goals

- **G1** The job page alone says which databases changed, which did not, and for a failure the
  file, the statement and the database's error.
- **G2** Affected rows are the counts the database returned. Unavailable is not zero, and work
  that was rolled back or superseded is not counted.
- **G3** The file, the log and the summary are the same result. A script reads the file and calls
  no API.
- **G4** A failed job still produces the result. No exit code changes.
- **G5** GitHub and GitLab get the same content, and any other CI gets the log and the file.
- **G6** Every consumer of today's `--output` keeps working.

### Non-goals

- **Exit codes and skip policy.** A missing stage that passes is BYT-10426.
- **A comment on the pull request or merge request.** See
  [Why the job and not the pull request](#why-the-job-and-not-the-pull-request).
- **Totals by insert, update and delete.** See [Rows](#rows).
- **Counts of changed tables and columns**, tracked in BYT-8367.

## Decision

The action reads the logs and builds the result. Four approaches were weighed, and
[Alternatives](#alternatives) gives each one's case.

| | In the action (chosen) | Result RPC on the server | Server posts to the VCS | API script |
|---|---|---|---|---|
| Result on the job page | Yes | Yes | No, on the commit or MR | Yes |
| Works on any CI | Yes | Yes | One integration per VCS | Yes |
| Server or proto change | None | New RPC | New integration | None |
| Calls when the job ends | One per database | One | None | The user's |
| Console can share it | No | Yes | No | No |
| Code the user maintains | None | None | None | The script |

Why the action:

- **The data exists and the action may already read it.** `GetTaskRunLog` needs
  `bb.taskRuns.list`, the permission `cancelRollout` uses today through `ListTaskRuns`.
  `GetRelease` needs `bb.releases.get`, which the GitOps Service Agent role holds.
- **The job is where the developer is looking.** The process that waited on the rollout holds its
  final state and is still running when the answer is needed.
- **It ships in the image.** No server release, migration or new permission gates it.
- **It closes no door.** A result RPC can fill the same structure later, and the three renderers
  do not change.

What it costs:

- **One log call per database** when the job ends. See [Open items](#open-items).
- **The action repeats rules the console has**, such as which retry counts. The two must agree.
  See [Attempts](#attempts).
- **The log's shape becomes a contract.** `checkVersionCompatibility` lets an action run against
  a server up to two minor versions newer, so log entries must stay readable that long.

## Design

### The result

The result is a tree. Each level is read from an existing response, with the API's own names and
status values.

| Level | Read from | Carries |
|---|---|---|
| Stage | `Rollout.stages` | Environment, task counts by status |
| Task | `Stage.tasks` | Database, status, skip reason and skipper, link |
| Task run | `ListTaskRuns` | Status, detail, start and update time, attempt number |
| File | `RELEASE_FILE_EXECUTE` log entries | Version, path, status, affected rows |
| Command | `COMMAND_EXECUTE` log entries, and the file for the text | Statement, affected rows, error |

- **Detail covers one stage**, the one the run ended on. That is the target stage on success and
  the failing stage otherwise. Other stages carry task counts only.
- **It describes the stage, not the process.** A job rerun after someone retried the task in
  Bytebase reports the same result as the job that first succeeded.
- **The file level is optional.** A `--plan` whose tasks run a sheet has no file entries, and its
  commands sit under the task.

### Statement text

The log gives a release file's commands as byte ranges, so the action needs the file to show a
statement. It tries two places, and the count and the error never depend on either.

1. **The job's checkout.** `GetRelease` returns each file's path and SHA-256. The action reads the
   path from the working directory and uses it only when the hash matches. This needs no
   permission beyond reading the release.
2. **The sheet**, with `GetSheet`. It needs `bb.sheets.get`, which the GitOps Service Agent role
   does not hold. The first refusal ends the attempts and logs one warning.

- **The ranges count from the file without its transaction directives.** A driver removes the
  `-- txn-mode` and `-- txn-isolation` lines at the top before it splits and logs commands
  (`ParseTransactionConfig` in `backend/plugin/parser/base`). The action removes them the same
  way before it cuts, and a test holds the two together.
- **Otherwise the command is named by position**, as `statement 3`.
- **Generated deploy jobs take the first path.** They check out the repository, and the release
  was created from the same commit.
- **A declarative rollout needs neither.** Its statements are generated, and the server logs their
  text.

### Outcome

One field says how the run ended. It restates today's behavior and changes none of it.

| Outcome | When | Exit code |
|---|---|---|
| `DONE` | Every task in the target stage is `DONE` or `SKIPPED` | 0 |
| `FAILED` | A task is `FAILED` | 1 |
| `CANCELED` | A task is `CANCELED` | 1 |
| `NOT_WAITED` | No `--target-stage`. The rollout was created and nothing was awaited | 0 |
| `STAGE_NOT_FOUND` | The rollout has no stage for `--target-stage` | 0 until BYT-10426 |
| `INTERRUPTED` | The action stopped first: a signal, a lost sign-in or an API error | 1 |

- **`DONE` is not "deployed".** The headline counts changed, unchanged and skipped databases
  apart. A task is unchanged when it applied no file, or when a declarative release computed an
  empty diff.
- **`NOT_WAITED` and `STAGE_NOT_FOUND` never read as success.** Their headline says nothing was
  deployed by this job.
- **`INTERRUPTED` reports the last state seen** and says it is not final.

### Rows

- **A count is what the driver returned** for a command. A file's count is the sum of its
  commands, and a task's is the sum of its files.
- **Unavailable is absent, not zero.** Fourteen drivers log a command response. The rest, such as
  Spanner, Doris and Databricks, log none, and their tasks carry no count.
- **Zero is real.** Most DDL reports zero.
- **Counts are actual.** `check` reports an estimate before execution, and the summary labels the
  two differently.
- **Each command shows its statement beside its count**, so the reader sees which statement
  changed how many rows.
- **No totals by insert, update and delete.** The log has one count per command and no statement
  kind. The first keyword is wrong for `MERGE`, an upsert, a CTE or a procedure call.

### Attempts

- **The latest task run is the task's result.** Earlier runs are listed with their status, and
  their rows are never added.
- **Inside one run, only the final attempt is the result.** An attempt before a `RETRY_INFO`
  marker was retried, so its failure is the reason for the retry and not the outcome, the rule the
  console follows in [task-run-log-previous-attempts.md](task-run-log-previous-attempts.md).
- **What a retried attempt committed still counts.** A script run with `-- txn-mode = off` commits
  each command, and the retry starts it again from the top. Those commands stay in the result,
  marked as from an earlier attempt, and add to the totals. Anything rolled back or never settled
  in a retried attempt is dropped.
- **Rolled-back commands add nothing to totals.** When the log shows `ROLLBACK` for a
  transaction, its commands keep their counts and are marked rolled back, as is their file. The
  file and task totals leave them out.
- **Without a logged rollback the result claims nothing** about what a failed file left behind.
  A rollback that itself failed is not a logged rollback.
- **Files after a failure are counted as not run**: the pending files after the last one the run
  reached, in the release's order. The task's `release_state` is a snapshot from when the rollout
  was created, and a retry skips what an earlier run applied, so its pending count alone is right
  only for a first run.

### Where it shows

| Surface | CI | Content |
|---|---|---|
| `--output` file | Any | The whole result |
| Job log | Any | The result block |
| Step summary | GitHub | The result block as Markdown |
| Error annotation | GitHub | One line per failed task |
| Annotations report | GitLab | A link to the rollout on the job page |

- **No new flag.** The block and the summary are always written, and `--output` gains keys.
- **A link goes where the reader runs out of answers or has to act, never above the result.**
  - A finished stage carries no rollout link. Each database links to its task.
  - A failed or canceled task links to itself beside its error, because retry and skip exist only
    in Bytebase.
  - A result with no finished stage to show links the rollout under the headline.
  - The job log has no hyperlinks, so it ends with the rollout's URL.
- **The plan and the release are never linked.** The plan link opens the page the rollout's does,
  and the release has no page of its own. Both stay in the file as names, which is what a script
  passes on.
- **Reporting never changes the exit code.** A log that cannot be fetched leaves that task with
  its status only and logs a warning.

### Why the job and not the pull request

A review belongs to a pull request, and a deploy belongs to a commit. Each reports where it
belongs.

| | Review | Deploy |
|---|---|---|
| Runs for | A pull request | A commit on the default branch |
| Command | `check` | `rollout` |
| Answers | Is this change safe to merge? | What did this commit do to each database? |
| Reader | The reviewer, before the merge | Whoever ships it, after the merge |
| Reports on | The pull request | The job |

- **A commit is not a pull request.** The commit a deploy runs for can come from one pull request,
  from several merged together, or from none: a direct push, a revert or a re-run.
- **A deploy is not one event.** One commit deploys to each environment in turn, sometimes days
  apart, and a failed stage is retried. The job page keeps each run and each attempt. A comment
  thread would grow a list on a pull request that is already closed.
- **The job is where a deploy is watched.** The CI already puts the job's status on the commit,
  notifies on its failure and offers the re-run there.
- **A comment needs more access.** A generated deploy job holds `id-token: write` and nothing
  else. A comment needs write access to pull requests.

A comment can still be rendered from the same result. See
[Not in this change](#not-in-this-change).

### The file

`release`, `plan` and `rollout` stay as they are. `result` is new.

```json
{
  "release": "projects/example/releases/release_20261112-RC00",
  "plan": "projects/example/plans/101",
  "rollout": "projects/example/plans/101/rollout",
  "result": {
    "outcome": "FAILED",
    "url": "https://bytebase.example.com/projects/example/plans/101/rollout",
    "targetStage": "environments/prod",
    "stages": [
      { "environment": "environments/test", "taskCounts": { "DONE": 2 } },
      {
        "environment": "environments/prod",
        "taskCounts": { "DONE": 1, "FAILED": 1, "SKIPPED": 1 },
        "tasks": [
          {
            "target": "instances/prod/databases/hr",
            "status": "DONE",
            "attempt": 1,
            "affectedRows": 1204,
            "files": [
              {
                "version": "0003",
                "path": "migrations/0003_backfill_grade.sql",
                "status": "DONE",
                "affectedRows": 1204,
                "commands": [
                  { "statement": "UPDATE employee SET grade = 'L1' WHERE grade IS NULL", "affectedRows": 1200 },
                  { "statement": "DELETE FROM employee_draft WHERE stale", "affectedRows": 4 }
                ]
              }
            ]
          },
          {
            "target": "instances/prod/databases/crm",
            "status": "FAILED",
            "attempt": 2,
            "affectedRows": 0,
            "filesNotRun": 1,
            "files": [
              {
                "version": "0004",
                "path": "migrations/0004_add_risk_score.sql",
                "status": "FAILED",
                "rolledBack": true,
                "affectedRows": 0,
                "commands": [
                  { "statement": "UPDATE account SET tier = 'basic' WHERE tier IS NULL", "affectedRows": 40, "rolledBack": true },
                  {
                    "statement": "ALTER TABLE account ADD COLUMN risk_score int",
                    "error": "ERROR: column \"risk_score\" of relation \"account\" already exists (SQLSTATE 42701)"
                  }
                ]
              }
            ]
          },
          {
            "target": "instances/prod/databases/legacy",
            "status": "SKIPPED",
            "skippedReason": "decommissioned",
            "skipper": "users/dba@example.com"
          }
        ]
      }
    ]
  }
}
```

- **Keys are camelCase and enum values are the API's**, as `checkResults` already is.
- **Counts are JSON numbers.** `checkResults` keeps the strings protojson writes for `int64`.
- **The file is complete.** Only the log and the summary shorten what they show.
- **Statement text is included.** It comes from the repository, so the file exposes no SQL the
  job could not already read.

### The job log

The block prints once, after the last log line, on every CI.

```text
Rollout to prod: FAILED
1 changed · 1 failed · 1 skipped
Other stages: test (2 done)

FAILED   instances/prod/databases/crm     attempt 2 · 1 file not run
         migrations/0004_add_risk_score.sql  failed, rolled back
           ALTER TABLE account ADD COLUMN risk_score int
           ERROR: column "risk_score" of relation "account" already exists (SQLSTATE 42701)
         Retry or skip: https://bytebase.example.com/projects/example/plans/101/rollout/stages/prod/tasks/2
DONE     instances/prod/databases/hr      1 file · 2 statements · 1,204 rows · 3.2s
         migrations/0003_backfill_grade.sql  1,204 rows
           1,200 rows  UPDATE employee SET grade = 'L1' WHERE grade IS NULL
               4 rows  DELETE FROM employee_draft WHERE stale
SKIPPED  instances/prod/databases/legacy  by dba@example.com: decommissioned

Rollout: https://bytebase.example.com/projects/example/plans/101/rollout
```

- **Failed tasks come first**: the file, the failed statement and the error.
- **A failed statement and its error are cut to 30 lines of 200 characters**, with a note that the
  full text is in the output file. The error is always kept, so a statement written on one long
  line cannot push it out.
- **Other tasks show one line per file**, then the first ten commands that changed rows and a
  count of the rest.
- **The failure line names the cause.** `found failed tasks. view on Bytebase` becomes the first
  failed database, its file and its error.
- **Each finished database folds on GitLab**, with section markers. A failure is never folded.
- **On GitHub the block is printed with workflow commands stopped.** A statement or a database
  error is printed as it is, and a line of either could otherwise be read as a command.

### GitHub

- **The step summary is the block as Markdown**: the headline, one table row per database, each
  failed statement with its error and a link to retry or skip it, and each changed database's
  statements in a collapsed section. It replaces the stage table and the Release and Plan links.
- **The table names the database and the instance** in two columns, not as a resource path.
- **It stays under 1 MiB.** GitHub drops a larger summary. Failed tasks are written first, then
  rows until the budget ends, then a count of the rest and the link. Failures that do not fit are
  counted too.
- **One `::error::` annotation per failed task**, through `github.ErrorCommand`, up to GitHub's
  ten per step.

### GitLab

GitLab has no job summary, so the log block is the result and the job keeps the file.

```yaml
bytebase-deploy-to-prod:
  script:
    - PLAN=$(jq -r .plan bytebase-metadata.json)
    - bytebase-action rollout --url=$BYTEBASE_URL --project=$BYTEBASE_PROJECT --target-stage=environments/prod --plan=$PLAN --output=bytebase-rollout.json
  artifacts:
    when: always
    paths: [bytebase-rollout.json]
    reports:
      annotations: bytebase_annotations.json
```

- **`when: always` is required.** GitLab uploads `paths` on success only by default, and the
  failed job is the one that needs the file (G4).
- **The action writes `bytebase_annotations.json`** on GitLab, as `check` writes
  `bytebase_codequality.json`. It puts a link to the rollout on the job page.
- **The generator adds these lines** to each deploy job, in the change that follows. It waits on
  the generator rewrite in BYT-10423.

### In the action

- **Three client calls.** `getTaskRunLog`, `getRelease` and `getSheetContent` join
  `listAllTaskRuns` in `action/command/api.go`.
- **A pure builder.** `action/result` turns a `Rollout`, its task runs, their logs and the files'
  text into the result. It makes no calls.
- **Three renderers** over the result: JSON, text and Markdown.
- **Collection runs on the way out.** `runRollout` already defers `output.WriteOutput`, so it runs
  after a failure. It uses a fresh context when the command's is canceled, as `cancelRollout`
  does.
- **Log calls are bounded in parallel**, and one that fails does not stop the rest.
- **Collection is bounded in time.** It has five minutes, or fifteen seconds once the job is
  canceled. When the rollout itself cannot be read again, the result stops at the last state seen
  and no call is made per database.
- **A checked-out file is read only when it is an ordinary file** of at most 100 MiB, since its
  path comes from the release.
- **`action/README.md`** documents the result, and its `--output` entry changes.

### Compatibility

- **`--output` only gains a key.** The generated workflows read `.plan` with `jq`. A consumer
  that decodes the file as a map of strings breaks, as five of our own tests did.
- **No server, proto, schema or permission change.**
- **No fallback for an older server.** The action refuses a server older than itself.
- **The GitHub summary changes content.** Nothing parses it.
- **Public docs** get a GitLab example that needs no API script.

## Testing

- **The builder**, on logs written as fixtures:
  - sums per file and task;
  - a count that is absent against one that is zero;
  - a rolled-back file left out of totals;
  - entries before a `RETRY_INFO` marker superseded;
  - the latest of two task runs;
  - a skipped task;
  - a sheet task with no file entries;
  - a statement cut from its file by byte range.
- **The renderers**: golden text and Markdown, and a summary under 1 MiB for twenty thousand
  tasks.
- **Statement text**, against a fake server: a checked-out file needs no sheet read, and an
  identity that cannot read sheets is refused once.
- **`backend/tests/action_test.go`**, against PostgreSQL. It needs a real rollout, and it is the
  builder's contract test against the server's log.
  - A DML file reports its rows in the file.
  - A failing statement leaves the file written, the statement and error in it, and a non-zero
    exit.
- **`gitopsWorkflowYaml.test.ts`**, with the generator change: the GitLab deploy job keeps the
  file with `when: always`.
- **Manual**: one passing and one failing rollout on GitHub and on GitLab.

## Open items

- **`bb.sheets.get` for the GitOps Service Agent role.** A job with no checkout shows `statement 3`
  in place of the text. Granting the permission fixes that and lets the role read every sheet in
  the project.
- **Totals by insert, update and delete.** The requester asked for them. They need the server to
  record each command's kind. Ask whether a count beside each statement is enough.
- **Cost at scale.** A thousand databases is a thousand log calls. Measure it. A row count on
  `TaskRun` would let the action fetch logs for failed tasks only.
- **MySQL commits DDL implicitly.** A file marked rolled back can still leave a schema change.
  The mark says the transaction rolled back, not that the database is unchanged.
- **A log too large for one response.** `GetTaskRunLog` has no paging. Such a task falls back to
  its status.
- **Database errors can quote row values**, such as a duplicate key. They would now reach the CI
  log, which more people may read than read Bytebase.

## Alternatives

**A result RPC on the server.** The server aggregates, and the action renders.

- **For:** one call for any number of databases. The console and BYT-8367 could show the same
  numbers.
- **Against:** a new API surface to design with the console's needs, and a server release before
  any job improves. It can follow, behind the same structure.

**The server posts to the VCS**, as a commit status or a merge request note.

- **For:** the result reaches the merge request with no job involved, and outlives the job.
- **Against:** the GitOps setup gives the server no VCS credential, and each VCS needs its own
  integration. The developer watching the job still sees pass or fail.

**A comment on the pull request or merge request, from the action.**

- **For:** it is where the change was reviewed, and `check` already comments there.
- **Against:** a deploy runs for a commit, not for a pull request, so the comment has no natural
  home. [Why the job and not the pull request](#why-the-job-and-not-the-pull-request) gives the
  case. Atlas, Liquibase and Flyway all leave deploy results on the job.

**A separate `report` command.**

- **For:** it can run in a later step, or for a rollout that finished outside the job.
- **Against:** it adds a step to every job, and the waiting process already holds the state. It
  can reuse the builder when that case appears.

**An HTML report as an artifact**, as Liquibase and Flyway produce.

- **For:** it is rich, and it works on any CI that keeps artifacts.
- **Against:** it is a download away from the job page, and a script cannot read it. JSON and the
  log cover both readers.

## Not in this change

- **The exit code for a missing stage**, BYT-10426.
- **A pull request or merge request comment** with the deploy result.
- **Totals by statement kind**, pending the first open item.
- **Counts of changed schema objects**, BYT-8367.
- **`check` output.** It keeps its file, comment and code quality report.

## Reference

How three migration tools report a deploy to CI. Surveyed 2026-11-12 from their docs and source.

| Element | Same as |
|---|---|
| One result rendered as a file, a log and a summary | Atlas's report struct, Flyway's actions over the CLI's JSON |
| The result is written on failure, with the file, the statement and the error | Atlas `Error{Stmt,Text}`, Flyway's error object with path and line |
| A job summary table per unit on GitHub | Atlas `migrate/apply`, Flyway `migrations/deploy` |
| No pull request comment for a deploy | Atlas, Liquibase, Flyway |
| A reporting failure does not fail the job | Liquibase's report, Atlas's cloud report, GitHub's summary upload |

| Element | Differs from |
|---|---|
| Affected rows per statement in the result | Atlas discards the count, Liquibase keeps one total per run, Flyway logs it at debug level only |
| A summary on a failed deploy | Flyway's action writes none |
| The run's link inside the structured result | Atlas prints its link only when the output is not JSON |
| A rendered result on GitLab | Atlas writes variables and log lines, Liquibase and Flyway leave the log and artifacts |
| Rolled-back work marked as such | Atlas's report is the same shape in every transaction mode |

Sources: [Atlas report structs](https://github.com/ariga/atlas/blob/master/cmd/atlas/internal/cmdlog/cmdlog.go) ·
[ariga/atlas-action](https://github.com/ariga/atlas-action) ·
[Atlas GitLab CI components](https://atlasgo.io/integrations/gitlab-ci-components) ·
[Liquibase structured logging keys](https://docs.liquibase.com/user-guide/what-are-structured-logging-keys) ·
[Liquibase update report](https://docs.liquibase.com/secure/user-guide-5-0/what-is-the-update-report) ·
[GitLab's Liquibase template](https://gitlab.com/gitlab-org/gitlab/-/blob/master/lib/gitlab/ci/templates/liquibase.gitlab-ci.yml) ·
[flyway/flyway](https://github.com/flyway/flyway) ·
[red-gate/flyway-actions](https://github.com/red-gate/flyway-actions) ·
[GitHub: job summaries](https://docs.github.com/en/actions/reference/workflows-and-actions/workflow-commands) ·
[GitLab: artifacts reports](https://docs.gitlab.com/ci/yaml/artifacts_reports/) ·
[GitLab: job artifacts](https://docs.gitlab.com/ci/jobs/job_artifacts/).

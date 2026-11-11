# One review on the plan page

Status: proposal · 2026-11-11 · [BYT-10505](https://linear.app/bytebase/issue/BYT-10505)

Checked on `main` at `309dd443b2`.

## Problem

The plan page splits the review into two reviewers, Standard rules and AI review. Each has its own
row, its own status, and its own findings. A finding is a comment thread a reviewer posts on the
SQL. Telling the two reviewers apart brings the author of a database change no benefit. The
author's job is to clear every open finding so the database change can roll out. Any open finding
holds the rollout, whichever reviewer posted it, and the author handles both kinds the same way:
change the SQL or resolve the finding.

## Goals

1. The plan page and the API show one review. The author never tells the two reviewers apart. An
   API caller starts or reads the review without naming a reviewer, and the calls that name one are
   removed.
2. The review is one action with one outcome. It posts every finding, or on an error it posts none.
3. Every on and off combination of the two reviewers works. The page says what was reviewed and
   never shows a pass for SQL that nobody checked.

## Non-goals

- Merging an AI finding with the standard rule finding it repeats. Today the AI review leaves such
  a finding out. Merging is better, because the AI review explains why the SQL is a problem where a
  rule only says it is not allowed. A separate document will cover it.

## Lifecycle of a review

One review with both reviewers on.

### 1. Trigger

Three calls start the review: `IssueService.CreateIssue`, `PlanService.UpdatePlan` when the SQL
changes, and `IssueService.RunReview` on Rerun. Today `RunReview` names one reviewer. In this
design it names none.

The rollout creator restarts the review when its job is missing or outdated.
`PlanService.RunPlanChecks` no longer restarts it, because a review can cost model calls.

### 2. Job

A trigger leaves one job for the issue in `review_run`. Today it leaves one job for each reviewer.
A new job supersedes a run still in flight.

### 3. Runner

One executor runs the job. It runs the standard rules first, then the AI review, and skips a
reviewer that is off. An error in either one ends the job, so a failure of the rules costs no
model call.

Both reviewers cover the CI sample of the project, or every database when the project sets no
sample size. Today the standard rules cover every database.

### 4. Result

The job ends in one transaction.

- Success: the job posts the findings of both reviewers as open threads and resolves its earlier
  findings that it no longer reports.
- Failure: the job stores its error and changes no finding.

### 5. Rollout

The rollout is created only when the review has succeeded on the current SQL and every thread is
resolved. Today the rollout waits for the rule review and never for the AI review.

## Reviewers on and off

- One reviewer off: the job skips it and posts the findings of the other.
- Both off: the backend behaves as today. The job still runs and succeeds with no finding, which
  resolves the earlier findings. The page says the review is off and shows no pass. It reads the
  two settings, as it does today.
- A reviewer turned off while a plan is open: its findings stay until the next run, which resolves
  them.
- The old SQL review is unchanged. It runs as a plan check where every standard rule is off or
  the engine is not covered, and it keeps its own row.
- An issue with no review run returns `NOT_FOUND` from `GetReviewRun`, as today.

## API changes

Today two calls on `IssueService` each name a reviewer, `rule` or `ai`:

- `GetReviewRun`: `GET /v1/projects/*/issues/*/reviewRuns/{reviewer}`
- `RunReview`: `POST /v1/projects/*/issues/*/reviewRuns/{reviewer}:run`

The design keeps both calls and the `ReviewRun` resource. An issue has one review run, so the name
drops the reviewer:

- `GetReviewRun`: `GET /v1/projects/*/issues/*/reviewRun`
- `RunReview`: `POST /v1/projects/*/issues/*/reviewRun:run`

```proto
message ReviewRun {
  // Format: projects/{project}/issues/{issue}/reviewRun
  string name = 1;

  enum State {
    STATE_UNSPECIFIED = 0;
    // The job is queued or running.
    RUNNING = 1;
    // Every reviewer that is on finished. The findings are complete.
    SUCCEEDED = 2;
    // A reviewer failed. The job posted no finding.
    FAILED = 3;
  }
  State state = 2;

  google.protobuf.Timestamp create_time = 3;
  google.protobuf.Timestamp end_time = 4;

  // Set only when the state is FAILED.
  google.rpc.Status error = 5;

  bool outdated = 6;
}
```

Four things change on `ReviewRun`:

- The name has no reviewer.
- `type` and the `Type` enum are removed.
- `status` becomes `state`, as AIP-216 asks, and `DONE` becomes `SUCCEEDED`.
- `error` changes from a string to `google.rpc.Status`.

`run_type` is removed from a finding and from the completion event in the API, because both use
the `Type` enum. A rule finding is the one that carries `rule_type`.

Two things stay as they are:

- The permissions `bb.reviewRuns.get` and `bb.reviewRuns.run`. Custom roles already hold them.
- The MCP classes, READ for the read call and WRITE for the run call.

The review run calls are not a formal API yet, because SQL Review V2 runs on dev builds only. So
fields are renamed or deleted in place, with no reserved numbers. Three callers move with the
change: the plan page, the issue overview tool of the in-console agent, and the generated MCP spec.

## Errors

A failed review is data. It is never an error of the call that reads it.

- `RunReview` fails only when the review cannot start: `NOT_FOUND` for a missing issue,
  `PERMISSION_DENIED` for a caller without the permission, and `FAILED_PRECONDITION` for a closed
  issue, a plan that has a rollout, or a plan with no SQL to review. A reviewer that is off is
  skipped without an error.
- `GetReviewRun` returns OK for a failed review. `state` and `error` carry the outcome.
- An error in either reviewer fails the whole review, and the job posts no finding. The findings
  of the last successful review stay as they were.
- A failed review holds the rollout, as a failed rule review does today.
- Nothing retries a failed review. `RunReview` starts it again.

Example: the AI provider keeps limiting the rate. Each model call is retried up to 3 times. Then
the review fails, and `GetReviewRun` returns:

```
state: FAILED
error: { code: RESOURCE_EXHAUSTED }
```

The page shows no finding of that run. It shows the reason and Retry.

One rule of AIP-151 is not followed. It asks a long running method to return
`google.longrunning.Operation`. No call in the product does. A review run is a resource with a
state, as `TaskRun` and `PlanCheckRun` are.

## Storage

- `review_run` holds one row for each issue. Today it holds one row for each issue and reviewer. A
  migration leaves one row for each issue and drops the reviewer type from the key.
- A stored finding keeps its reviewer type. The API does not show it. It lets an engineer trace
  which reviewer found a finding, and the AI review uses it to find its own earlier findings.

## Frontend changes

The plan page shows one review, named SQL review. The
[prototype](https://claude.ai/artifact/NfGEsz9jxQPiNo26FWuuX7) shows every state. The link is
private until it is shared.

- Review section: one row replaces the Standard rules row and the AI review row. It shows the open
  findings by priority and one Rerun button.
- Findings: every finding names SQL review as its author. A rule finding keeps its rule name and
  code.
- Activity: one entry for each review replaces one entry for each reviewer.
- Deploy: one line says what the rollout waits on.
- Running: the row says the review is running. No finding of that run shows until it ends.
- Failed: the row shows the reason and Retry. No finding of that run shows.
- Both reviewers off: the row says the review is off. It never shows a pass.

## Alternatives considered

- Run the reviewers as two jobs and report a partial success. Rejected. The author then has to
  tell a partial review from a complete one, which brings the two reviewers back. One job has a
  cost: the rule findings wait for the AI review, and an AI error leaves no finding.
- Run the standard rules as a tool call of the AI review. Rejected. The model decides when to call
  a tool, so it can skip the rules.
- Keep the calls that name a reviewer beside the new ones. Rejected. One API is enough, and every
  caller is in this repository and moves with the change.

## Build plan

Three PRs, in this order. Each carries its tests.

1. API and plan page: the new `ReviewRun`, its generated code, and the one row on the plan page.
   The page moves with the proto, because the new shape cannot fill two rows. Until the second PR
   lands, the two calls read the two existing jobs as one run, and the rollout waits for both.
2. One job: the executor and the sample. The job uses the existing table with one fixed reviewer
   type.
3. Migration: one row for each issue in `review_run`. It lands last, because the code before the
   second PR still reads the reviewer type in the key.

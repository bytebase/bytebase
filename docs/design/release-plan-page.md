# Release-backed plan page

Status: proposal · 2026-10-27

A GitOps release is minted by bytebase-action inside the customer's CI: one run creates the release
(a snapshot of the migration folder at that commit), a plan whose title is the release id, and the
rollout, then waits for the target stage. Nobody reviews in Bytebase; the pull request did that. The
plan page is the one page these users open, usually from the CI job summary and usually because a
migration failed, and today it shows the release artifact twice and the per-database outcome never.
The rule this doc lands on: **Changes is the artifact and Deploy is the outcome: the Changes rail
lists the release's files by version with no status, and each task shows what the release did, or
will do, to its database as one spine of file sections in the task-run log's own rows.** Every
task is folded on load to one line that says what happened. A details block under the title names
the pull request or commit, the release and the progress per stage, and gives a UIOps plan the same
block. No action is added. The release detail page goes. The backend adds one task payload field,
two fields on a skipped task and a few optional proto fields; the executor does not change.

## Problem

Checked on `main` at `1057d6cf19`. The plan page for a release-backed plan is the UIOps plan page
with the release artifact dropped into two slots:

- **Changes shows the manifest, not the change.** `PlanDetailStatementSection.tsx` renders
  `ReleaseInfoCard` for a release spec: the first four files in stored order, which is ascending by
  version (`validateAndSanitizeReleaseFiles` in `backend/api/v1/release_service.go` sorts them
  that way), a "View all files" link to the release page, the commit URL and the creation time. On a
  release with hundreds of carried-over files the four shown are the oldest; the one that changed
  is behind "and N more files", and the chips do nothing when clicked.
- **The task repeats the manifest and hides the outcome.** `DeployTaskBody.tsx` renders the same
  `ReleaseInfoCard` through `DeployReleaseInfoCard.tsx`, then `DeployLatestTaskRunInfo.tsx`
  renders the run's error in an alert and the task-run log under it. The executor
  (`runVersionedRelease` in `backend/runner/taskrun/database_migrate_executor.go`) lists the
  database's revisions, skips every file whose version has a live `VERSIONED` revision, runs each
  remaining file with one `driver.Execute`, creates the file's revision only after the whole file
  succeeded, and stops at the first failure. The skips reach only the server log
  (`slog.InfoContext`, line 537); the task-run log records a `RELEASE_FILE_EXECUTE` marker per
  file that ran. So a task shows the files that ran and nothing about the ones that were skipped or
  not reached, and a pending task shows nothing about what will happen on its database.
- **The failure cannot be read or taken away.** The error appears twice (alert and log row). The
  failed statement is in the log, under the fold of the failed row, but the file it belongs to
  cannot be opened from the page. To recover statements from their byte ranges, the log fetches the
  sheet of every file in the release (`useTaskRunLogData.ts`), several hundred requests on the
  largest releases, for the one to three files a run touches. The log groups the entries
  after each file marker under that file (`groupEntriesByReleaseFile` in
  `frontend/src/components/task-run-log/model.ts`), so the database sync that runs after the loop
  renders under the last file. In the replays, users select the error text to copy it, click the
  file name expecting it to open, and leave for the repository or for the database's revision
  page to read the SQL.
- **Nothing relates the plan to the change in git.** The title is the release id, the creator is
  the service account, and the only link to the source is the commit URL that `getVCSSource` in
  `action/command/rollout.go` builds from `GITHUB_SHA` and its equivalents. `Release.VCSSource`
  holds `vcs_type` and `url` and nothing else. A user looking for "the release with my PR" has
  only dates and candidate numbers to go by.
- **A done task does not lead to the database.** Verifying that a migration reached the database
  goes through the Databases navigation; the task has no link to the revisions its run created.
- **Skip leaves no honest trace.** `PlanDetailTaskRolloutActionPanel.tsx` shows the reason field
  only when the plan has an issue (`action === "SKIP" && page.issue`), and a release plan has
  none, so a skipped migration is recorded with "No reason provided"
  (`DeployTaskSkippedReason.tsx`) and the plan reads Deployed. The task payload records a skip as
  `skipped` and `skipped_reason` only (`proto/store/store/task.proto`): who skipped, and when,
  exist only in the audit log.

The evidence is 21 archived Cloud session replays from the four accounts that run this workflow,
read against the plan, rollout, task runs, task-run log, revisions and release of the plans they
opened. The heaviest account runs several releases a day from a service account, with 197 to 535
files per release and one to three new files each; the others carry 20 to 120 files. Of the
failed-release visits, most end with no action in Bytebase: the user reads the error and fixes the
file in git. The rest rerun, skip, or import a revision on the database page after a long search.
Every visit that wanted to verify a deploy went through the Databases navigation.

## Principle

> In GitOps the team already has an authoring tool and a review tool. Bytebase is the thing that
> runs the migration and the only place that knows what happened on each database. The page keeps
> the plan page's frame and makes the per-database outcome its content; the release artifact is
> context, shown once and collapsed.

What follows from it:

- Changes never carries status: applied is a per-database fact, and one stage can hold a database
  that has every version but one, a new one that gets them all, and one that failed halfway.
- The failure is a row, not a banner.
- Facts, no classification, no new actions.
- Stay close to today's plan page: the same frame, task rows, log rows and actions as a UIOps plan;
  what is new is only what a release needs.
- Every bypass leaves a trace: who, when, why.
- Phone width is a layout, not a degradation; the console is opened from phones in the replays and
  clips there today.

## Decisions

**D1 · Changes lists every file by parsed version, newest first, grouped by the version's shape.**
The store already holds a release's files ascending by parsed version
(`validateAndSanitizeReleaseFiles`, see Problem; `model.NewVersion` takes digits and dots, and the
action strips an optional `v` or `V` prefix in `extractVersion`), rejects duplicate versions, and
the executor runs them in that order. The rail shows the same list reversed, so the newest file is
on top whether versions are timestamps such as `202610051500` or counters such as `001`. One
grouping level, derived from the version string: a timestamp-like version (a first segment of 8 to
14 digits starting with a plausible year) groups by day for the newest group and by month for the
rest, with the newest group open and the others collapsed with a count; counters and anything else
get no date groups: flat up to the row limit, then buckets of a hundred. An open group shows at most
12 rows and then "N more", which extends the group in place. A 535-file release therefore opens as
one day group with its files and about eight month headers. A row shows the version and the file's
path as stored in the release. Collapsed and declarative forms are in States. Rejected: naming the
files added since the previous release first. It needs a lookup of an earlier release and a
definition of "previous" (by creation time, of the same type, in the same project) that the executor
does not share; descending version order puts the new files on top without any of that, and the one
case it gets wrong, a backfilled migration given a low version on purpose, is the case the executor
applies in that same low position, so the rail and the run agree. Also rejected: the current
four-chip grid, and a sheet listing the files (one click too many for the most common visit).

**D2 · Changes never shows per-database status.** No chips, no "applied" marks, no counts of what
ran. The aggregate is the Progress row (D10) and the stage chips; the detail is in each task.
Rejected: status chips on the files in Changes, which would have to pick one database to be true
for.

**D3 · Each task is one spine: the per-database file state in the task-run log's own rows, from the
stored state before the first run and from the run's log after it.** The expanded task card drops
the release card and the separate error alert and keeps the waiting note, the skipped line and the
latest-run line (time, who triggered the run, duration, history), which already names the run's
executor. Under it, one slot, whose sources are:

- Before the first run, the state stored on the task at rollout creation (D4): the versions the task
  will apply, and the count already applied.
- Once a run has started, the run's log and the release's file list: one file section per
  `RELEASE_FILE_EXECUTE` marker, the already-applied versions derived from them (D5), and task-level
  steps (prior backup, schema dump, database sync) as sibling sections in the order they ran, never
  nested under a file. A gh-ost migration runs per file inside the executor's file loop, so its
  entries belong to that file's section.

While a run is in progress, the files it has not reached show below it as "Will apply", from the
stored state. The log decides every version up to its newest marker and the stored state covers the
rest, because the executor writes a marker before each file it runs and writes nothing for a file it
passes over: a version the stored state listed as "Will apply" that has no marker below a newer one
was found applied, and moves into the already-applied fold; a version it counted as applied that
gets a marker (its revision was deleted after the rollout was created) leaves the fold and becomes a
section in its place. Once the run ends, done, failed or canceled, the spine shows only what ran:
the executor syncs the database after every run, and a file after the failure did not run in it. The
release's full file list stays in Changes. The stored state also keeps the "revision deleted
`<date>` by `<who>`" note on its version's section, so a re-run version still says why. Which source
each task state reads, and what it renders, is in States. The log is append-only and untouched; the
list is a view. Earlier runs stay under history as today.

Every task is folded on load, release and UIOps alike. A folded card is one line: the status dot,
the database, one short description, and today's buttons. The description takes a soft tint of the
task's state (green done, blue running, red failed, gray otherwise) while the dot keeps the full
color, and below 640 px it moves under the database. Per state, with the full table in States:

- Failed: the error's first line in red, after the failed version for a release task. It is the
  run's `detail` with the executor's prefix naming the file and version removed, so a stage with
  several failed tasks says why each failed without opening any.
- Done: "All 493 applied · 4.1 s" for a release task: the executor ends a run Done only after every
  version in the release has a live revision on the database, so the release's file count is exact
  without counting anything, and it answers whether the database reached the release. For a UIOps
  task the statement kind and the duration ("DML · 4.1 s", "DDL · 1.2 s", "Mixed · 3.4 s"). The
  duration is the run's start and end time, as today's collapsed card already shows it. Which
  files this run applied, and the exact rows each statement affected, are in the opened task.
- Running: "Applying 3 files · started 15:14", or "Running · started 15:14" for a UIOps task. The
  count is the stored to-apply count, fixed when the run starts, and the time is absolute, so the
  line does not go stale between polls; no file name, which would.
- Not started: "3 to apply · 490 applied earlier" from the stored state (D4), or "DML · about 1,180
  rows" from the plan check's summary for a UIOps task, then the scheduled time or the waiting cause
  when the run has one.
- Skipped: "Skipped by `<user>` · `<reason>`". Canceled: "Canceled · `<time>`".

*Applied earlier* counts the release's versions that were on the database before this plan; the
statement kind is DDL, DML or Mixed from the plan check's `statement_types`, whose enum is grouped
that way. A database sync that failed after the files applied leaves the task Done (the executor
logs the error and goes on); its section carries the error as today, and the done line adds "·
schema sync failed". The word "skipped" is reserved for the user action (D7); versions the executor
passes over are "applied earlier" on the card and "already applied" in the fold.

A file section header carries the version, the file's path as a link to the viewer (D8) with a copy
control for the path, a status chip and the duration; the chevron toggles, the path opens the file.
A failed section that is collapsed also shows the error's first line in red, as the folded task
card does. Inside a file the rows are exactly the UIOps rows (`SectionContent.tsx`): begin, one row
per command, commit or rollback, retries as attempts. A succeeded command's line is its statement's
first line; a failed command's line is the error, and the row toggles to the statement block with
its copy control; below 640 px the failed row opens with the block folded. For a driver that never
writes a command response (the `TaskRunErrorAlert.tsx` comment lists them), and for a run that
failed in a task-level step or before any file, the failed section's header line carries the run's
error (`TaskRun.detail`) and says that no statement was identified, so the cause is never absent.

On load, a URL that names a stage or task wins and opens that task; otherwise the page selects the
stage holding the latest failed task, else the running one, else the last stage, and opens no task.
Opening a failed task opens its failed section and failed row, the others collapsed.

The spine shows at most 24 file sections. Beyond that, the oldest applied sections fold into one "N
applied" row and the planned tail folds into "N more, will apply"; both expand in place, and the
running section and a failed section are always shown and open. The already-applied fold ("N
already applied", with the version range) is folded by default and opens newest first, five rows
at a time with "Show more"; each row is the path, opening the viewer. Inside a section the log's
existing "Load more" paging stays. A done task ends with a "Revisions on `<database>`" link to the
database page's Revisions tab. A UIOps task is the same component with one implicit section for its
statement. The revision page, which embeds the same viewer for the revision's task run
(`RevisionDetailPanel.tsx`), renders the log-only form (States).

Rejected: a ledger table of files and statuses above the log, which repeats every file the log
already names and leaves two places to read the same error; a separate error alert; a stepper of
files with the log as its leaves, which is the spine drawn with more chrome; a copy control on the
error line, since the error is selectable text and the statement block beside it already has one;
listing the files after a failure as "Will apply" or "Not run", which describes a run that has
ended as if it continued; a line under the failed section naming its transaction record, which
repeated the commit or rollback row above it; the failed statement's position in the header
("statement k of n" or a line number), since the open row shows the statement and the viewer
highlights it; a Revisions link on a task that has not finished, where it leads to no new revision;
a sentence under the Deploy header naming each failed database, file and error, which repeated what
the folded cards already show; counts as icon-and-number columns on each card (applied, failed, to
apply, already applied), too much for a folded row, and a progress bar on a running card; the files
a run applied and the rows it affected on the folded card, which exist only in the run's log: saving
them as totals on the run's result adds fields to keep, counting them when task runs are listed
scans logs on every poll, and reading each card's log costs a request per task on every load; and
expanding the latest failed task on load, which buried the other failures below it.

**D4 · The per-database file state is computed once, at rollout creation, and stored on the task.**
`getTaskCreatesFromChangeDatabaseConfig` in `backend/api/v1/rollout_service_task.go` builds each
release-backed task's payload today with only the release name. It gains a field holding the
versions this task will apply (the small set: one to three for a train, all of them for a new
database), each with the deleted revision's time and deleter when one existed, the count of versions
already applied, and the time of the computation; the already-applied set is release files minus
to-apply, so the large set is never stored. The rule is the executor's: a version is applied when
the database has a live `VERSIONED` revision with that version; deleted revisions do not count. One
indexed query per target, through the store's existing `Versions` filter (`version = ANY(?)` in
`backend/store/revision.go`, which `checkReleaseVersioned` in `release_service_check.go` already
uses per target), so the cost is bounded by the release's file count, not the database's history.
`GetRollout` and `ListTasks` already return the task, so the folded card and the planned spine read
the field with no second request. What a run applies is decided only when it runs: the executor
reads live revisions at its start, so the field is a preview that can go stale (another plan applies
a version first, a revision is imported or deleted). The caption on the planned spine says so: "As
of plan creation, `<time ago>`; the run re-checks the database." Once a run starts, its log corrects
the list version by version (D3). A declarative release gets no stored list: `runDeclarativeRelease`
computes its diff inside the run, so its pending task shows only its status. Rerun creates a new
task run, not a new task, so the field is refreshed only when a new rollout re-creates the task,
which is what the action does on every push. Rejected: a read-only preview RPC (release plus targets
to per-file state), which the page would call on every load for every pending task; computing the
intersection in the frontend, which has no version filter (`ListRevisions` in the v1 API exposes
none; the store has one) and would read a database's whole history; reusing `CheckRelease`, which
runs SQL review and returns only the files it will apply; recomputing when a card is expanded, which
is the preview RPC again; and storing nothing, which keeps finished tasks as they are (they read the
log) but leaves a task that has not run with no answer to what it will apply: no planned list, no
"Will apply" zone while running, no "revision deleted" note, and folded cards that say only "Not
started" or "Running".

**D5 · No new log entry: the already-applied list is derived from the markers the log already has.**
The executor writes a `RELEASE_FILE_EXECUTE` marker before each file it runs, in version order, and
stops at the first failure. With the release's file list, a version below the run's newest marker
that has no marker was found applied: that part is exact. Above the newest marker the log says
nothing, because a run can fail before the next file's marker (loading that file's statement fails)
or before any marker at all (the driver cannot connect, the revisions cannot be read). So the
boundary is the newest marker, not the failure. A done run's unmarked versions are all applied,
since Done means every version has a live revision, marker or not. A run that ended any other way
takes the versions above its newest marker, or all of them when it wrote none, from the stored state
(D4): its applied versions join the already-applied fold and its to-apply versions are not listed,
as nothing after a stop is (D3). A task without a stored state, whose run predates this change,
shows only the versions below its newest marker. While a run is in progress the stored state gives
the files not yet reached in the same way. The executor does not change: its skips stay in the
server log (`slog`, line 537 of `database_migrate_executor.go`). Rejected: a `RELEASE_FILE_SKIP` log
entry listing the versions the executor passes over with the revision that satisfied each, which
would let each fold row name the release that applied it but adds an executor write and a proto type
for detail the Revisions link already reaches; reconstructing the set from the database's revisions
after the fact, which drifts the moment a revision is deleted or imported.

**D6 · No new actions; Rerun, Skip and Cancel stay as today.** They render as today, for a user
without the permission too. The commit is one click away in the source row (D9); the database is
one click away in the task header. Rejected: an "Open file" button (the path already opens the
viewer), a "Query `<database>`" button (the database page has the SQL editor entry), a "Mark as
applied" button that would move Import Revision from the database page into the task, which stays
where it is; a hint under the spine stating what a rerun does, which repeated the card.

**D7 · Skip shows why each task failed, requires a reason, and records who and when.** The dialog
keeps today's shape: the selected tasks and the reason field, which becomes required for a
release-backed task and is written on each selected task. Each task line keeps today's database
display, as the plan targets show it (engine icon, instance / environment badge, database), and
ends with the task's status chip, which replaces today's leading status icon. A failed task's line
also carries its error's first line in red below it, the same text its folded card shows, so the
person skipping sees why each task failed before deciding. It is the plan page's panel, so UIOps
plans get the chips and error lines too. Skip runs nothing and reverts
nothing. Each release file is its own transaction, so the files the last run applied stay applied
with their revisions, and the failed file's own transaction was already rolled back by the run
(engines that commit DDL on their own, such as MySQL, TiDB and Oracle, keep what ran of it).
`BatchSkipTasks` already takes several tasks and a `reason` (up to 1000 characters) and the store
already keeps `skipped_reason`; the store's `BatchSkipTasks` additionally writes the skipper and
the skip time into the task payload, returned as two output-only fields on the v1 `Task`, and the
task then shows "Skipped by `<user>` · `<time>` · `<reason>`" above its last run. The panel stops
gating the reason field on the plan having an issue. Rejected: making the reason required in the
API, which would change UIOps skips too; reading the skipper from the audit log, which the plan
page does not load and a project Developer cannot read; spelling out the consequence per database
and per later stage, and pointing at Import Revision, in the dialog, since the task list is enough;
recording a revision on Skip, which would claim the version applied, which is what Import Revision
on the database page is for.

**D8 · A file opens in a read-only viewer: a side panel on wide screens, a sheet below.** Opened
from any file path: in Changes, in a spine section header or the already-applied fold. The body is
`ReleaseFileDetailPanel.tsx`'s content: version, path with a copy control, size, the sheet's
statement in the read-only editor with its copy control, and a link to the file at the commit, "Open
in `<platform>` at `<short sha>`", built from the D9 template when the platform is known. On a
failed file the link is the header's main action: the next step after reading the error on its
statement is to fix the file in git. Opened from a failed section, it scrolls to the failed command
and draws the error on that statement as an inline diagnostic, from the command's byte range
(`CommandExecute.range`, byte offsets into the sheet) and the error in its response, placed once the
log arrives; the header says "Failed on `<stage>`". With no command range (the drivers above) it
opens at the top with the error above the editor. Opened from an applied section or the
already-applied fold, one line under the header says what this database did with the version
("applied by this run", "already applied; this run passed over it"). Opened from Changes there is no
database and no line. Drivers log one command per statement, so the diagnostic lands on the
statement; a PL/SQL block is one command, so there the diagnostic covers the block. The container
follows the viewport through the `useMediaQuery` hook (`frontend/src/hooks/useMediaQuery.ts`), as
`ProjectPlanDashboardPage.tsx` does with its `MOBILE_MEDIA_QUERY` constant; the breakpoints are in
States. One body component either way. Rejected: a modal dialog (reading a statement next to the row
that failed, or clicking through files in Changes, should not open and close a modal each time);
keeping the release page as the place to read a file (D12).

**D9 · The source row lives on `Release.VCSSource`, filled by the action from the CI environment and
explicit flags.** The details block's Source row (D10) renders the fields it has in a fixed order:
`title · author · repo@sha · branch`, missing fields leaving no gap. The title is the pull request
title when the run knows one, else the first line of the commit message, else `repo@sha`; the PR or
the commit is the link. Text only: trimmed, control characters removed, no markdown, 72 characters
then an ellipsis with the full text as the tooltip. Repository and SHA are parsed from the commit
URL the action already sends, so a release with only `url` still gets a row (States). On a phone the
title takes its own line and the rest a second muted line.

`VCSSource` gains `commit_sha`, `branch`, `commit_title`, `author_name`, `author_login`,
and `pull_request { url, number, title }` in both the v1 and the store proto, all
optional, `url` unchanged as the commit link; the backend converts them through. The action image
is Alpine with jq, bash and curl (`scripts/Dockerfile.action`; the Debian variant has the same
tools), so `getVCSSource` reads only environment variables and, on GitHub, the event payload file
that `vcs_user.go` already parses for pull-request events:

| Field | GitHub Actions | GitLab CI | Bitbucket Pipelines | Azure Pipelines |
|---|---|---|---|---|
| Branch | `GITHUB_REF_NAME` | `CI_COMMIT_REF_NAME` | `BITBUCKET_BRANCH` | `BUILD_SOURCEBRANCHNAME` |
| Commit title | event payload `head_commit.message`, first line | `CI_COMMIT_TITLE` | not in the environment | `BUILD_SOURCEVERSIONMESSAGE`, first line |
| Author | `head_commit.author.username`, or `pull_request.user.login` | `CI_COMMIT_AUTHOR` | not in the environment | `BUILD_REQUESTEDFOR` |
| Pull request | `pull_request` event: number, title, URL. On push, the merge marker in the head commit's message: `(#N)` at the end of a squash merge, `Merge pull request #N` on a merge commit; a rebase merge carries none | `CI_MERGE_REQUEST_IID` and `CI_MERGE_REQUEST_TITLE` in MR pipelines; on push, `See merge request …!N` | `BITBUCKET_PR_ID` in PR pipelines; on push, `(pull request #N)` | `SYSTEM_PULLREQUEST_PULLREQUESTID` in PR builds; on push, `Merged PR N:` |
| File at commit (frontend, from `repo@sha` and the file's path) | `<repo>/blob/<sha>/<path>` | `<repo>/-/blob/<sha>/<path>` | `<repo>/src/<sha>/<path>` | `<repo>?path=/<path>&version=GC<sha>` |

The file's path is the action's glob match, relative to the pipeline's working directory; a pipeline
that runs in a subdirectory gets a link that misses, and an unknown platform gets none. A marker
miss is silent: the row shows the commit. When the title is the commit's first line and the PR
number came from its marker, the marker is removed from the title, so a squash merge reads `<title>
#N` once. The PR URL is the repository URL plus the platform's path and the number. On a platform
the action does not detect (Jenkins, CircleCI, Buildkite, a shell) nothing is derived; four new
flags let the pipeline pass what it knows: `--source-title`, `--source-url`, `--source-author`,
`--source-ref`, and flags win over detection everywhere. A new action against an old server sends
the fields and the server ignores what it does not know; an old action against a new server gives
the commit-URL row. Rejected: writing the title into the plan's description, a free-text field a
user can edit, which loses a system-written line on the first edit, cannot be linked reliably and
cannot feed the plan list later; shelling out to git in the action, which has no git binary and runs
inside the pipeline's container on every platform; an API lookup of the PR by SHA, which needs a
token the rollout job does not always have; a link to the CI run: no visit in the replays went back
to the pipeline, the PR's checks already lead there, and it would cost a URL template per platform
and a proto field.

**D10 · A details block under the title, for release and UIOps plans; the title stays the release
id.** Short facts about the plan sit as labeled rows in one block in the page's flow, between the
title row and the description:

| Row | Release-backed plan | UIOps plan |
|---|---|---|
| Source | D9's row: PR or commit title, author, `repo@sha`, branch | none |
| Triggered / Created | "shop-ci (service account) · Oct 5, 15:02 · 12 minutes ago": the plan's creator by display name, not email, and `create_time` | The creator and time, today's "Created by" line |
| Labels | none (no issue) | Today's inline issue labels, editable in place, required when the project forces them; absent when the plan has no issue |
| Release / Changes | "Migration-based · 493 files · newest `202610051530` Oct 5", or "State-based · `<schema version>`", from the release's type and files | "DDL", "DML" or "Mixed" on N databases, from the plan check's `statement_types` |
| Review | none | The issue's approval state, "✓ Approved by alex (DBA) · Oct 5, 14:45", or who is next; absent when the plan has no issue |
| Progress | The plan list's stage cell (`ProjectPlanDashboardPage.tsx`): each stage's status icon and name, joined by arrows, wrapping when there are many | Same |

The description stays below the block, rendered and clipped as today (`PlanDetailHeaderDetails.tsx`:
150 characters or 3 lines, then Show more), so the rows keep their place whatever its length. A
release plan's collapsed Changes line then shows only the targets per stage. "Migration-based" and
"State-based" are the docs' names for the two GitOps workflows; the console's "Versioned" and
"Declarative" elsewhere are a separate copy change. Rejected: the PR title as the plan title, which
breaks the link from the CI summary and the plan list, and is absent on a rebase merge or an
undetected platform; a side panel for the block, which competes with the file viewer's side panel
(D8) and takes width the folded cards need; the description as the block's last row, which turns
the block into a document container under a second label and gives every CI-created plan an empty
row; the release facts in the Changes summary instead of a block, which leaves Progress no place and
splits plan facts between two spots; a statement count (it needs a parse the page does not have)
and a summed row estimate (each database differs; its card has its own); counts in the Progress
row, which the stage tabs and chips already carry.

**D11 · No error classification.** The page states the file, the statement, the error and what
applied, and stops. Rejected: recognizing "already exists", lock timeouts and constraint
violations and proposing a next step per class. Every class needs its own detector and its own
advice, both wrong on the next engine or driver, and the replays show users reading the error
themselves and fixing the file in git.

**D12 · The release detail page is removed; release URLs redirect to the owning plan.** Reading a
file's SQL moves into Changes and the viewer, and the plan already shows everything else the page
showed. The CI job summary prints a Release link (`action/github/summary.go`), so the URL keeps
working: `/projects/<p>/releases/<r>` redirects to the newest plan whose spec references the
release. A release no plan references (created through the API, or a CI run that died between the
release and the plan) shows the not-found page; that is rare and the API still serves it. The page
is also the only place that archives and restores a release (`deleteRelease` and `undeleteRelease`,
called only from `ProjectReleaseDetailPage.tsx`), so both move to the release list before the page
goes: each row gets Archive with the page's confirmation, and a "Show archived" switch lists
archived releases (`ListReleases` already takes `show_deleted`) with Restore on each row. Rejected:
keeping both pages, which leaves two places to open a file and the manifest shown twice; putting
Archive on the plan page, which is about deploying the release, not managing it.

## States

The spine, per task status:

| Task | Sources | Sections, top to bottom |
|---|---|---|
| Not started, pending | Stored state, caption "As of plan creation, `<time ago>`; the run re-checks the database." | "N already applied" folded with the version range; one "Will apply" section per file to apply, in version order, no children and no caret; "Will apply again · revision deleted `<date>` by `<who>`" where a revision was deleted |
| Running | The live log, the release's file list, and the stored state for files not reached | Already-applied fold; reached files (applied collapsed, the current one open with a running indicator); unreached files below as "Will apply"; the boundary moves down |
| Done | The log and the release's file list | Already-applied fold (the files without a marker); every file section Applied; task-level steps as siblings; "Revisions on `<database>`" at the end |
| Failed | The log and the release's file list | Already-applied fold; applied sections collapsed; the failed section open, its failed row open with the error on the line and the statement block under it (folded below 640 px); task-level steps as siblings; nothing for the files after the failed one |
| Canceled | The log and the release's file list | As Failed, with the section the cancel interrupted marked Canceled |
| Skipped, after a run | As Failed | "Skipped by `<user>` · `<time>` · `<reason>`", then the last run as Failed |
| Skipped, never run | Stored state | "Skipped by `<user>` · `<time>` · `<reason>`", then the planned list, collapsed |
| Revision page, any run | The log and the release's file list | The sections of that run with the open and closed defaults of the run's status; no stored state, no Revisions link |

The folded card's description, per task state (D3):

| Task | Release-backed | UIOps | Tint |
|---|---|---|---|
| Not started | "3 to apply · 490 applied earlier", then "· scheduled 18:00" or the waiting cause; a declarative release shows the status only | "DML · about 1,180 rows", then the schedule or waiting cause | gray |
| Running | "Applying 3 files · started 15:14" | "Running · started 15:14" | blue |
| Done | "All 493 applied · 4.1 s"; "· schema sync failed" when the sync failed | "DML · 4.1 s", "DDL · 1.2 s" or "Mixed · 3.4 s" | green |
| Failed | `<failed version>` then the error's first line | The error's first line | red |
| Skipped | "Skipped by `<user>` · `<reason>`" | Same | gray |
| Canceled | "Canceled · `<time>`" | Same | gray |

The title row's status is today's; with a skipped task it reads "Deployed, 1 task skipped".

Changes:

| Rollout | Rail |
|---|---|
| Not started | Open: manifest line, groups, 12 rows then "N more" |
| Started or finished | Collapsed: the targets per stage (the release facts are in the details block, D10); opens on click to the same content |
| Declarative release | One line: the schema folder at its version, with Open |

The viewer, by viewport:

| Viewport | Container | Behavior |
|---|---|---|
| 1280 px and wider | Side panel, page keeps about 38% | Page stays interactive; another file swaps the content; Esc closes |
| 640 to 1279 px | `Sheet` over a scrim | Standard width, same body |
| Below 640 px | Full-width sheet | Lines wrap |

The details block's Source row:

| Data the release carries | Row |
|---|---|
| PR known | `<PR title> #N` as the link · by `<author>` · `repo@sha` · `branch` |
| Commit title only | `<commit title>` as the link · by `<author>` · `repo@sha` · `branch` |
| Platform without title or author in the environment (Bitbucket push) | `repo@sha` · `branch` |
| Only `url` (old action, or a release created through the API) | Source · `repo@sha` |

## Scope

| Area | File | Change |
|---|---|---|
| Frontend | `frontend/src/routes/project/plan-detail/components/PlanDetailStatementSection.tsx` | The release branch renders the new Changes rail (D1, D2) instead of `ReleaseInfoCard`. |
| Frontend | `frontend/src/routes/project/plan-detail/components/changes/` (new) | The release Changes rail: version parsing and grouping, the collapsed line, the open groups with the row limit, the viewer trigger. Beside the route that is its only consumer. |
| Frontend | `frontend/src/routes/project/plan-detail/components/deploy/DeployTaskBody.tsx`, `DeployLatestTaskRunInfo.tsx`, `DeployReleaseInfoCard.tsx`, `DeployTaskSkippedReason.tsx` | The body drops the release card and, for release tasks, the error alert, and renders the spine (D3); the run line gains the Revisions link; the skipped line gains who and when (D7); `DeployReleaseInfoCard.tsx` goes. |
| Frontend | `frontend/src/components/release/ReleaseInfoCard.tsx`, `ReleaseInfoCard.test.tsx` | Removed with its two consumers. |
| Frontend | `frontend/src/components/task-run-log/model.ts`, `types.ts`, `useTaskRunLogSections.ts`, `TaskRunLogViewer.tsx` | A file group ends at the next file marker or at a task-level entry; task-level entries become sibling sections; the already-applied fold derived from the release's files, the markers and the stored state; the list of a task that has run built from its log and the release's files, with the stored state only for files a running task has not reached, the log deciding every version up to its newest marker; the already-applied fold's paging; the spine's row limits; the log-only form the revision page renders (D3, D5). |
| Frontend | `frontend/src/components/task-run-log/useTaskRunLogData.ts` | Fetch the sheets of the files the run reached (its `RELEASE_FILE_EXECUTE` markers), not every file in the release; the viewer fetches the one file it opens. |
| Frontend | `frontend/src/components/task-run-log/SectionHeader.tsx` | Status chip, duration, the path as a link with a copy control beside the toggle; the failed section open by default, and the error's first line on it when collapsed; the run's error as the header line when no row carries one. |
| Frontend | `frontend/src/routes/project/plan-detail/components/deploy/DeployTaskHeader.tsx`, `DeployTaskItem.tsx`, `DeployBranch.tsx` | Every task folded on load; the folded card's one-line description per state for release and UIOps tasks, with its tint, under the database below 640 px; the stage selected on load (D3). |
| Frontend | `frontend/src/components/release/ReleaseFileViewer.tsx` (new), `ReleaseFileDetailPanel.tsx` | The viewer: one body with the diagnostic drawn from the command's byte range and error, the path's copy control and the file-at-commit link; side panel or sheet by `useMediaQuery` (D8, D9). |
| Frontend | `frontend/src/routes/project/plan-detail/components/PlanDetailTaskRolloutActionPanel.tsx` | Each task's status chip after its database, replacing the status icon, and a failed task's error line, for UIOps and release tasks; the reason field shown and required for release-backed tasks (D7). |
| Frontend | `frontend/src/routes/project/plan-detail/components/PlanDetailHeaderDetails.tsx`, `PlanDetailMeta.tsx` | The details block for release and UIOps plans, with the Source row and the plan list's stage cell for Progress, above the description, which keeps today's clipping (D9, D10). |
| Frontend | `frontend/src/routes/project/ProjectReleaseDashboardPage.tsx` | Archive on each row, a "Show archived" switch, and Restore on archived rows, before the detail page goes (D12). |
| Frontend | `frontend/src/routes/project/ProjectReleaseDetailPage.tsx`, `frontend/src/app/router/routes/dashboard.tsx` | The page goes; the route redirects to the owning plan (D12). |
| Frontend | `frontend/src/locales/*.json` | New strings; the retired release-page strings removed. |
| Backend | `proto/store/store/task.proto`, `proto/v1/v1/rollout_service.proto`, `backend/api/v1/rollout_service_task.go`, `rollout_service_converter.go` | The release state field on the task payload, computed at creation with the store's `Versions` filter, returned on the v1 task (D4). |
| Backend | `proto/store/store/task.proto`, `proto/v1/v1/rollout_service.proto`, `backend/store/task.go`, `backend/api/v1/rollout_service.go` | The skipper and skip time written by `BatchSkipTasks`, output-only on the v1 task (D7). |
| Backend | `proto/store/store/release.proto`, `proto/v1/v1/release_service.proto`, `backend/api/v1/release_service.go` and its converter | The optional `VCSSource` fields, converted through (D9). |
| Action | `action/command/rollout.go`, `action/command/vcs_user.go`, `action/command/root.go` | `getVCSSource` per platform from the table in D9; the GitHub event payload read shared with the VCS user code; the four `--source-*` flags. |

Every element on the page renders with the project Developer role, which reads plans, rollouts,
task runs, logs, revisions and the release's sheets today; the read path adds no permission.

Tests to write, per the placement rules in `AGENTS.md`:

- `model.test.ts`: a file group ending at a database-sync entry (today's fixture puts the sync under
  the last file; the new one proves it is a sibling); the already-applied fold derived for a done
  task (every file without a marker) and for a failed task (the files below the newest marker
  without a marker), and its paging of five; a running task listing the files it has not reached as
  "Will apply" from the stored state, the boundary at the current file; a stored "Will apply"
  version the run passed over moving into the fold, and a stored applied version that gets a marker
  leaving it; a failed task listing nothing after the failed file; a failed run with no marker at
  all (the driver cannot connect) and one that failed loading the second file's statement after one
  marker, each taking the versions above its newest marker from the stored state, and the same run
  without a stored state listing only the versions below it; a done run with no marker, all applied;
  a collapsed failed section carrying the error's first line; the 24-section fold with a failed
  section past the limit still shown and the fold expanding in place; a prior backup as a sibling
  and a gh-ost migration inside its file's section; a run with no command response carrying the
  run's error on the header; the log-only form without a task keeping the run's defaults.
- `useTaskRunLogData.test.tsx`: a run that reached two files of a 500-file release fetches two
  sheets; a re-run version keeps the stored "revision deleted" note on its section.
- Folded card tests: each cell of the card table in States from the task, its latest run, the
  stored state and the plan check, before any log loads; the executor prefix removed from the
  error's first line; the release's file count on a done release task and "· schema sync failed"
  after a failed sync; DDL, DML and Mixed from the statement types, with the run's duration;
  the description under the database below 640 px; every task folded on load, the stage selected for
  a failed, a running and a deployed plan, and a URL naming a task opening it.
- Details block tests: each row for a release plan and a UIOps plan, the rows a plan kind lacks
  absent; Migration-based and State-based; the Progress row's stage cells wrapping at six stages;
  the labels editable in place and required when the project forces them; the description below the
  block clipped at 150 characters or 3 lines.
- Release list tests: Archive on an active row with its confirmation, "Show archived" listing
  archived releases through `show_deleted`, and Restore on an archived row.
- Changes rail tests: descending order over timestamp, counter and dotted versions; the grouping per
  shape with the newest group open; 12 rows then "N more"; a backfilled low version landing in its
  old group; the collapsed line; the declarative one-liner.
- Viewer tests: the diagnostic placed from a byte range on a multi-byte statement, opened from the
  failed section; opening at the top with the error when there
  is no range; the per-database line when opened from a section and none from Changes; the
  file-at-commit link per platform and none on an unknown one; the container chosen per viewport
  through a mocked `matchMedia`; swapping files in the side panel; Esc.
- Source row tests: each row of the States table from a `VCSSource` with that subset; a squash-merge
  title ending in `(#N)` rendered with the number once; the 72-character truncation and tooltip; a
  title with markdown and control characters rendered as text; the parse of `repo@sha` from each
  platform's commit URL.
- Action panel tests: each task line with its status chip, and a failed task's error first line, for
  a UIOps and a release task; the reason field shown and required for a release task, single and
  multi-select, and the skipper and time shown on the task afterwards; the UIOps path unchanged.
- `backend/api/v1`: rollout creation stores the state for a release-backed task against PostgreSQL
  through `testcontainer`: a target with two of three versions applied, a new target, and a target
  with a deleted revision carrying the deleter and time. Collision coverage, since the lookup keys
  on instance, database and version: the same version with a live revision on a sibling database of
  the same instance must not count for the target, and the target's own deleted revision of that
  version must not count either; the sibling's state is asserted unchanged. `BatchSkipTasks` writes
  the skipper and the time on each selected task and on no other task of the stage, and the v1 task
  returns them.
- `action/command`: `getVCSSource` per platform from environment fixtures and a GitHub payload file:
  PR event, squash-merge push, merge-commit push, rebase push (commit title, no PR), Bitbucket push
  (no title), undetected platform with and without the flags, flags overriding detection.
- `frontend/tests/e2e/plan-detail`: a release-backed plan created through the API against the shared
  server with a target that already carries one version, read as a project Developer: the Changes
  rail newest first with the viewer opening from a file path; the planned spine before the run;
  after the run the already-applied fold with its row and the applied section with the Revisions
  link; a failing second of three files showing the error on the row and in the viewer, nothing
  listed for the third, and the error on the folded card; as a project Owner, Skip refusing an
  empty reason, the task then showing the skip line over the failed run, and the header reading
  "Deployed, 1 task skipped"; the release URL redirecting to the plan.

## Not in this PR

- The plan list for release plans (the failing file in the row, the source title as a column) and
  the Releases navigation entry, which the replays show as a decoy. Confirming that several
  candidates failed on the same file therefore costs one page open each, and an old failed plan does
  not name the newer candidate that superseded it, nor say whether one exists.
- The database page: the Instances permission wall, GitOps setup, Import Revision's own dialog
  (including importing several versions at once after a replayed baseline), and the Revisions tab as
  an entry for failures; a failed file leaves no revision, so the tab cannot lead to one, and the
  revision page's existing link to its task run is the only change-free path back.
- A files-by-database grid for stages with dozens of databases, which can be added without changing
  the task, and a refresh of a pending task's stored state without a new rollout: a small refresh
  control beside the "As of plan creation" caption that recomputes the list read-only and does not
  store it (an option on `CheckRelease` to skip review and return the applied files is the cheapest
  form), if stale previews on long-gated stages turn out to matter. The actions stay enabled while
  it runs, since a run re-checks the database anyway.
- A live check of a failed version against the database's current revisions when an old failed plan
  is opened after a later candidate fixed it (D6), and carrying a user Skip forward onto the next
  release's planned list for that database.
- Revisions on the database that the release does not know (a version outside the release, a
  baseline imported under another version); see the counts in D3.
- A revision link on each applied section, a finish time on each section header, a line number on a
  section header, a reason on Rerun, a sum of applied files across a task's runs, and who imported a
  revision (revisions record no creator); the Revisions link, the rows' timestamps, the viewer, the
  run's executor and the run history cover them.
- One copy control on the failed section header that copies the path, the statement and the error
  together; the path and the statement each have their own control.
- Naming, on each already-applied row, the release that applied it, and flagging applied content
  that differs from the release's file; both need the log entry D5 rejects, and the Revisions link
  reaches them.
- Error classification (D11) and any new action (D6).
- The issue page's own action panel (`IssueDetailTaskRolloutActionPanel.tsx`), which keeps today's
  task list.
- A phone layout for the task-run log rows themselves (the index and time columns ahead of the text,
  the section's fixed scroll height); they are the UIOps component's rows and change with it, for
  every plan.
- A PR lookup by SHA in the action, for a rebase merge or for a job that holds a token.

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
block. No action is added. Main already retired the release detail page (D12). The backend adds one task payload field,
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
rest, with the newest group open and the others collapsed with a count; counters, dotted versions
such as `3.10.0008` and anything else are one flat list. A group exists only where its label names
what its files share, a day or a month; a boundary drawn by row count names nothing, and the load
row below already bounds the list. An open group shows its
first 12 rows; one row below them offers "Load 12 more" and "Load all (N)", which load in place, and
once more than 12 rows show it also offers "Show fewer", which returns the group to its first 12
and keeps the row under the pointer. "Load all" is one click, so an accidental "Show fewer" costs
one click to undo. A 535-file release therefore opens as
one day group with its files and about eight month headers. A row reads like a repository's file
list, not like code: a file icon, the file name in body text, its folder muted after it, and the
version muted at the row's end, in the body font at body size, with a divider between rows. The
whole row opens the viewer, and on hover the name takes the link color and an underline. The path
is as stored in the release; a path with no folder shows the name alone. Opened, the section keeps today's spec tab and Targets on top; only
the release card under them is replaced by this list. Collapsed and declarative forms are in States.
Rejected: naming the files added since the previous release first. It needs a lookup of an earlier
release and a definition of "previous" (by creation time, of the same type, in the same project)
that the executor does not share; descending version order puts the new files on top without any of
that, and the one case it gets wrong, a backfilled migration given a low version on purpose, is the
case the executor applies in that same low position, so the rail and the run agree. Also rejected:
buckets of a hundred files labeled by their version range ("2.23.0–3.3.0005 19"), whose edge marks
nothing about the files and reads as a category such as "applied before", which Changes cannot say
(D2); the current four-chip grid, and a sheet listing the files (one click too many for the most common
visit); each row's full path in monospace link color, which reads as source code and makes the folder,
repeated on every row, as loud as the name.

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
was found applied, and joins the files already applied; a version it counted as applied that
gets a marker (its revision was deleted after the rollout was created) leaves them and becomes a
section in its place. Once the run ends, done, failed or canceled, the spine shows only what ran:
the executor syncs the database after every run, and a file after the failure did not run in it. The
release's full file list stays in Changes. The stored state also keeps who deleted the version's earlier revision
and when, so a re-run version still says why: its applied section reads "Applied again" instead of
"Applied", and every status of it carries an info icon whose tooltip, also reached by keyboard, reads
"Earlier revision deleted `<date>` by `<who>`". A version is re-run when its section exists and the
stored state either noted a deleted revision for it or counted it applied: a revision deleted after
the rollout was created leaves no note, but the run still applied the version a second time, and its
tooltip reads "Earlier revision deleted after this rollout was created". A task without a stored
state cannot tell, and reads "Applied". The icon sits beside the status chip, outside the
section's toggle. "Earlier" keeps the note from reading as this run's revision. Rejected: the note as
a line under the section, which under "Applied" reads as if this run's revision was deleted. Which source
each task state reads, and what it renders, is in States. The log is append-only and untouched; the
list is a view. Earlier runs stay under history as today.

Every task is folded on load, release and UIOps alike. A folded card is one line: the status dot,
the database, one short description, and today's buttons. The description takes a soft tint of the
task's state (green done, blue running, red failed, gray otherwise) while the dot keeps the full
color, and below 640 px it moves under the database. A click anywhere on the header line, the card's
padding around it included, folds or unfolds the card, so a reader never has to find the live part.
The checkbox, the buttons and a margin around each keep their own click, and a click that ends a
text selection does nothing. The task name stays the keyboard toggle. Per state, with the full
table in States:

- Failed: the error's first line in red, after the failed version for a release task whose error
  names a file. It is the
  run's `detail` with the executor's prefix naming the file and version removed, so a stage with
  several failed tasks says why each failed without opening any.
- Done: "All 493 applied · 4.1 s" for a release task: the executor ends a run Done only when every
  version in the release had a live revision when the run read them at its start or was applied by
  the run, so the release's file count is exact without counting anything, and it answers whether
  the run brought the database to the release. It is the run's result, not a live check: a revision
  deleted during or after the run leaves the line as it was, as it leaves today's Done status. For a UIOps
  task the statement kind and the duration ("DML · 4.1 s", "DDL · 1.2 s", "Mixed · 3.4 s"). The
  duration is the run's start and end time, as today's collapsed card already shows it. Which
  files this run applied, and the exact rows each statement affected, are in the opened task.
- Running: "Applying 3 files · started 15:14", or "Running · started 15:14" for a UIOps task. The
  count is the stored to-apply count from rollout creation (D4), a preview like the not-started
  line, and the time is absolute, so the line does not change between polls; no file name, which
  would. The opened task's spine shows what the run actually reaches.
- Not started: "3 to apply · 490 already applied" from the stored state (D4), or "DML · about 1,180
  rows" from the plan check's summary for a UIOps task, then the scheduled time or the waiting cause
  when the run has one.
- Skipped: "Skipped by `<user>` · `<reason>`". Canceled: "Canceled · `<time>`".

A versioned release task whose stored to-apply list is empty says so before it runs: "Nothing to
apply · 493 already applied" not started, and "Running · started 15:14" running. A declarative
release has no list, so its task shows only its status in every state. A done versioned release task
reads "All 493 applied · 0.3 s" whatever its stored list held: that is
a statement about the database (D5), true after any successful run, while the stored list is a
preview that a revision deleted between rollout creation and the run makes wrong, so "Nothing new"
there could contradict the opened task's "Applied again".

*Already applied* counts the release's versions that were on the database when the rollout was created; the
statement kind is DDL, DML or Mixed from the plan check's `statement_types`, whose enum is grouped
that way. A database sync that failed after the files applied leaves the task Done (the executor
logs the error and goes on); the error reaches only the log, so it shows in the opened task's
Database sync section as today, not on the folded card. The word "skipped" is reserved for the user
action (D7); versions the executor passes over are "already applied" on the card and in the
spine.

A file section header carries the version, the file's path as a link to the viewer (D8) with a copy
control for the path, a status chip and the duration; the chevron toggles, the path opens the file.
A failed section that is collapsed also shows the error's first line in red, as the folded task
card does. Inside a file the rows are exactly the UIOps rows (`SectionContent.tsx`): begin, one row
per command, commit or rollback, retries as attempts. A succeeded command's line is its statement's
first line; a failed command's line is the error, and the row toggles to the statement block with
its copy control; below 640 px the failed row opens with the block folded.

A failed run's error (`TaskRun.detail`) always has a section. An error with the executor's file
prefix ("failed to execute release file `<path>` (version `<v>`)") belongs to that file's section,
the newest one. The path ends at the prefix's first " (version ", since the driver's message after
it can quote the user's SQL (a MySQL syntax error does), which may contain the same text; one from a task-level step (a prior backup) belongs to that step's section. Every
other error has no section of its own: the revisions could not be read, the driver could not
connect, a file's statement could not be loaded before its marker, or a revision could not be
written after the file ran. For those the spine ends with an open task-level "Run failed" section
holding the full error. A section carrying the error with no failed command row, as with a driver
that never writes a command response (the `TaskRunErrorAlert.tsx` comment lists them), says that no
statement was identified, so the cause is never absent.

On load, a URL that names a stage or task wins and opens that task; otherwise the page selects the
stage holding the latest failed task, else the running one, else the last stage, and opens no task.
Opening a failed task opens its failed section and failed row, the others collapsed.

A long spine follows the UIOps log (`SectionContent.tsx`): rows keep their order, a cap bounds
what renders, a row loads what is hidden in place, and the row that explains the outcome is never
hidden. The spine opens with a count line naming the states present, "60 files · 29 applied · 1
running · 30 pending" (applied, running, failed, canceled, pending, already applied). It then shows
the first 20 file sections in version order, and the running, failed or canceled section wherever
it falls, open. Each hidden stretch is one row that names what it loads: "Load 9 more applied",
"Load 30 pending". "More" appears only when a section of the same state
shows above the row; there is no version range, because the rows around it show where the stretch
starts and ends. A stretch longer than 20 offers "Load 20 more …" and "Load all (N)". Loading is
one way, as in the UIOps log: there is no "Show fewer", which would sit below the task-level steps.
The files already applied are not part of the run: they are the database's state before it, so
they are no stretch and get no load row. They are one fold line at the top, before the first section,
"60 files already applied to this database", closed on load and outside the cap. Opened, it lists
every one of them in version order as a nested tree: a recessed gray panel, a guide line down from
the caret, the rows indented right of it and their versions faded. The panel's edge ends the list,
so the first file to apply never reads as one more row of it, and no added heading repeats the
count line. Each row has its version and its path, which opens the viewer, and no chip, since the
line names their state. Rejected: only a heavier divider after the list, which leaves the two kinds
of row alike; a "N files to apply" heading, which repeats the count line. Inside a section the log's existing "Load more" paging stays. A task ends with a
"Revisions on `<database>`" link to the database page's Revisions tab once its latest run applied at
least one file, whether the run is done, failed, canceled or still running, and on every done task:
each file writes its revision when it commits, so a failed run's applied files are already on the
Revisions tab. A UIOps task is the same component with one implicit section for its
statement. The revision page, which embeds the same viewer for the revision's task run
(`RevisionDetailPanel.tsx`), renders the log-only form (States). Rejected: folding the stretches
into "N applied" and "N more, will apply" rows, which read as rows of the list and unfold their
files at the level of their own header; loading the earliest files at the top of a run, which
reverses the reading order; the UIOps cap of 50, whose rows sit in a box ten rows tall, while file
sections open their own log boxes, so they sit in the page and the cap bounds the page height;
"to apply" on a load row, which reads as an instruction ("Load 30 to apply"); a load row for the
files already applied, which pages the database's prior state as if it were the run's progress and
lists those files at the level of the run's own; "applied in a previous release" for them, since a
revision can come from any release, another plan or an import, and a database added to the group
later has none of them, which the per-database line states and a release-level label cannot.

Rejected: a ledger table of files and statuses above the log, which repeats every file the log
already names and leaves two places to read the same error; a separate error alert; a stepper of
files with the log as its leaves, which is the spine drawn with more chrome; a copy control on the
error line, since the error is selectable text and the statement block beside it already has one;
listing the files after a failure as "Will apply" or "Not run", which describes a run that has
ended as if it continued; a line under the failed section naming its transaction record, which
repeated the commit or rollback row above it; the failed statement's position in the header
("statement k of n" or a line number), since the open row shows the statement and the viewer
highlights it; a Revisions link on a task whose latest run applied no file and that is not done,
where it leads to no new revision;
a sentence under the Deploy header naming each failed database, file and error, which repeated what
the folded cards already show; counts as icon-and-number columns on each card (applied, failed, to
apply, already applied), too much for a folded row, and a progress bar on a running card; the files
a run applied and the rows it affected on the folded card, which exist only in the run's log: saving
them as totals on the run's result adds fields to keep, counting them when task runs are listed
scans logs on every poll, and reading each card's log costs a request per task on every load; and
expanding the latest failed task on load, which buried the other failures below it.

**D4 · The per-database file state is computed once, at rollout creation, and stored on the task.**
`getTaskCreatesFromChangeDatabaseConfig` in `backend/api/v1/rollout_service_task.go` builds each
release-backed task's payload today with only the release name. It gains a field holding which of
the release's files this task will apply, as half-open ranges of positions in the release's file
list (a release's files never change, so the positions stay valid): one range for a train or a new
database, another only where a deleted revision left a gap. Beside the ranges it holds the count of
versions already applied, each to-apply version whose revision was deleted with the newest
deletion's time and deleter, and the time of the computation; the already-applied set is release
files minus to-apply. The field stays small whatever the release's size: a 1,000-database group of
new databases on a 500-file release stores one range per task, where a list of versions would store
500,000 entries and return them on every rollout poll. The v1 task returns the ranges, their total
as `pending_count`, and the deleted-revision notes; the page expands the ranges against the
release's files, which it already loads. The rule is the executor's: a version is applied when
the database has a live `VERSIONED` revision with that version; deleted revisions do not count. One
query per instance for all of the rollout's targets on it (`ListPendingReleaseVersions` in
`backend/store/revision.go`) returns only the pairs of target database and release version that
have no live `VERSIONED` revision, each with its newest deleted revision of that version if one
exists; the applied count is the release's file count minus the pending ones, since a versioned
release holds each version once. A release carries the whole migrations folder, so on a train
nearly every pair is applied: returning the pending pairs instead of the matching revisions keeps
the rows to about one per database for the new file, where reading the revisions would return one
per applied file per database, about 5 million for a 500-file release on 10,000 databases. Postgres
still checks every pair, through the revision index on instance, database, type and version, but
nothing it rejects leaves the database. The rule therefore lives twice, in the executor's Go and in
this query's SQL, and the store test pins it to the executor's: a deleted or declarative revision
is not applied. Rejected: one query per target, which `checkReleaseVersioned` in
`release_service_check.go` uses in the background plan check but which would put one uncached round
trip per database into the synchronous rollout request; reading the matching revisions per instance
and filtering in Go, which costs one row per applied pair.
`GetRollout` and `ListTasks` already return the task, so the folded card and the planned spine read
the field with no second request. What a run applies is decided only when it runs: the executor
reads live revisions at its start, so the field is a preview that can go stale (another plan applies
a version first, a revision is imported or deleted). The caption on the planned spine says so: "As
of rollout creation, `<time ago>`; the run re-checks the database.", its age from the stored
computation time, not the plan's: a plan can wait before its rollout is created. Once a run starts, its log corrects
the list version by version (D3). A declarative release gets no stored list: `runDeclarativeRelease`
computes its diff inside the run, so its pending task shows only its status. A release that is
gone when the rollout is created (deleted) gets no stored state either: the task is still created
and its run fails on the page, as it did before the field existed, rather than rollout creation
failing in the background for an approved issue. The deleter is stored by email, so a user's email
change rewrites it with the user's other references (`UpdateUserEmail`). Rerun creates a new
task run, not a new task, so the field is refreshed only when a new rollout re-creates the task,
which is what the action does on every push. Rejected: a read-only preview RPC (release plus targets
to per-file state), which the page would call on every load for every pending task; computing the
intersection in the frontend, which has no version filter (`ListRevisions` in the v1 API exposes
none; the store has one) and would read a database's whole history; reusing `CheckRelease`, which
runs SQL review and returns only the files it will apply; recomputing when a card is expanded, which
is the preview RPC again; a list of the to-apply versions, which on a new database holds every file
of the release, per task; and storing nothing, which keeps finished tasks as they are (they read the
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
since Done means every version was found applied at the run's start or applied by it, marker or not. A run that ended any other way
takes the versions above its newest marker, or all of them when it wrote none, from the stored state
(D4): its applied versions join the files already applied and its to-apply versions are not listed,
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
the skip time into the task payload, returned as two output-only fields on the v1 `Task` (a user's
email change rewrites the skipper with the user's other references), and the
task then shows "Skipped by `<user>` · `<time>` · `<reason>`" above its last run. The panel stops
gating the reason field on the plan having an issue. Rejected: requiring the reason in the API,
even only when a selected task is release-backed, which would refuse the skips API clients and
scripts send today without one; the dialog is where a person decides; reading the skipper from the audit log, which the plan
page does not load and a project Developer cannot read; spelling out the consequence per database
and per later stage, and pointing at Import Revision, in the dialog, since the task list is enough;
recording a revision on Skip, which would claim the version applied, which is what Import Revision
on the database page is for.

**D8 · A file opens in a read-only viewer: a side panel on wide screens, a sheet below.** Opened
from any file path: in Changes, in a spine section header or a file already applied. The body is
`ReleaseFileDetailPanel.tsx`'s content: the version as the title, the path as plain text with a copy
control, size, the sheet's
statement in the read-only editor with its copy control, and a link to the file at the commit, "Open
in `<platform>` at `<short sha>`", built from the D9 template when the platform is known. On a
failed file the link is the header's main action: the next step after reading the error on its
statement is to fix the file in git. Opened from a failed section, it scrolls to the failed command
and draws the error on that statement as an inline diagnostic, from the command's byte range
(`CommandExecute.range`, byte offsets into the sheet) and the error in its response, placed once the
log arrives; the header says "Failed on `<stage>`". With no command range (the drivers above) it
opens at the top with the error above the editor. Opened from an applied section or the
files already applied, one line under the header says what this database did with the version
("applied by this run", "already applied; this run passed over it"). Opened from Changes there is no
database and no line. Drivers log one command per statement, so the diagnostic lands on the
statement; a PL/SQL block is one command, so there the diagnostic covers the block. When the page
is at least 1280 px wide, the viewer is a column docked on the page's right: the page body becomes a
two-column grid, the plan on the left half and the column on the right half, sticky at the top, the window's height below the header, and scrolling on its own, with Close in its
header. Opening, swapping or closing a file changes only the grid's columns: the plan stays mounted,
so loaded rows, open tasks and sections, and fetched logs keep their state. This is the layout the task detail panel had until #20785 replaced it with inline task
cards; that change removed the panel because it repeated the stage's task list, not because of the
column. The width is the page's, measured with a `ResizeObserver` as that panel did, since the
sidebar takes part of the window. Narrower, the viewer is the standard-width sheet the task's run
history uses, over a scrim. One body component either way. Rejected: a modal dialog (reading a
statement next to the row that failed, or clicking through files in Changes, should not open and
close a modal each time); keeping the release page as the place to read a file (D12); a sheet over
the page on wide screens, even without a scrim, which covers the plan it is read beside; the old
task panel's 1024 px threshold, which leaves the task cards 512 px; the path as the title, whose
length outweighs the version.

**D9 · The source row lives on `Release.VCSSource`, filled by the action from the CI environment and
explicit flags.** The details block's Source row (D10) renders the fields it has in a fixed order:
`title · author · repo@sha · branch`, missing fields leaving no gap. The title is the pull request
title when the run knows one, else the first line of the commit message, else `repo@sha`; the PR or
the commit is the link. Text only: trimmed, control and bidirectional-override characters removed
(U+202A–202E, U+2066–2069, which could make a title read as another), no markdown, 72 characters
then an ellipsis with the full text as the tooltip. Repository and SHA are parsed from the commit
URL the action already sends, so a release with only `url` still gets a row (States). Any client
that can create a release writes these fields, so the row and the viewer's file link (D8) use only
an `http` or `https` URL; any other scheme leaves the title as text and the file unlinked. On a phone the
title takes its own line and the rest a second muted line.

`VCSSource` gains `commit_sha`, `branch`, `commit_title`, `author_name`, `author_login`,
and `pull_request { url, number, title }` in both the v1 and the store proto, all
optional, `url` unchanged as the commit link; the backend converts them through. The action image
is Alpine with jq, bash and curl (`scripts/Dockerfile.action`; the Debian variant has the same
tools), so `getVCSSource` reads only environment variables and, on GitHub, the event payload file
that `vcs_user.go` already parses for pull-request events:

| Field | GitHub Actions | GitLab CI | Bitbucket Pipelines | Azure Pipelines |
|---|---|---|---|---|
| Branch | `GITHUB_HEAD_REF` in pull-request runs, else `GITHUB_REF_NAME` when `GITHUB_REF_TYPE` is `branch`; none in a `merge_group` run | `CI_MERGE_REQUEST_SOURCE_BRANCH_NAME` in MR pipelines, else `CI_COMMIT_BRANCH` (unset in tag pipelines) | `BITBUCKET_BRANCH`, the source branch in PR pipelines too | `SYSTEM_PULLREQUEST_SOURCEBRANCH` in PR builds, else `BUILD_SOURCEBRANCH` when it starts with `refs/heads/`, without that prefix |
| Commit title | event payload `head_commit.message`, first line | `CI_COMMIT_TITLE` | not in the environment | `BUILD_SOURCEVERSIONMESSAGE`, first line |
| Author | `head_commit.author.username`, or `pull_request.user.login` (the PR's author) | `CI_COMMIT_AUTHOR` | not in the environment | `BUILD_REQUESTEDFOR` (who queued the build; Azure has no commit author in the environment) |
| Pull request | `pull_request` event: number, title, URL. On push, the merge marker in the head commit's message: `(#N)` at the end of a squash merge, `Merge pull request #N` on a merge commit; a rebase merge carries none | `CI_MERGE_REQUEST_IID` and `CI_MERGE_REQUEST_TITLE` in MR pipelines; on push, `See merge request …!N` | `BITBUCKET_PR_ID` in PR pipelines; on push, `(pull request #N)` | `SYSTEM_PULLREQUEST_PULLREQUESTID` in PR builds; on push, `Merged PR N:` |
| File at commit (frontend, from `repo@sha` and the file's path) | `<repo>/blob/<sha>/<path>` | `<repo>/-/blob/<sha>/<path>` | `<repo>/src/<sha>/<path>` | `<repo>?path=/<path>&version=GC<sha>` |

In pull-request builds the platform's own ref is synthetic (`<N>/merge` on GitHub, `merge` on
Azure), so the branch comes from the source-branch variable; Azure's `BUILD_SOURCEBRANCHNAME` also
keeps only the last segment of `feature/x`. A run on a tag or on a merge queue's temporary branch
(`gh-readonly-queue/…`) has no source branch, so `branch` stays empty rather than naming the tag or
the queue. The SHA stays the checked-out commit, a merge commit in
those builds, because the release's files came from it and D8 links them there.

The file's path is the action's glob match, relative to the pipeline's working directory; a pipeline
that runs in a subdirectory gets a link that misses, and an unknown platform gets none. A
declarative rollout concatenates every match into one file and records the `--file-pattern` itself as
its path (`getReleaseFiles` in `action/command/file.go`), so on a declarative release a path with a
doublestar metacharacter (`*`, `?`, `[`, `{`) names no file and gets no link; a literal pattern is
the file and keeps it. A versioned release's path is a real file name, which may contain `?` or
`[`, so it always keeps its link. A marker
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
title row and the description. The rows are body text (14 px), the size of the Progress row's stage
cells, with muted labels; the caption size (12 px) is for secondary text, and these are the page's
main facts. Every row has one height, 24 px, the shared `xs` control height that the label chips,
their remove button and the add-label button keep, with its content centered, so a row of chips
lines up with a row of text; only a row that wraps grows:

| Row | Release-backed plan | UIOps plan |
|---|---|---|
| Source | D9's row: PR or commit title, author, `repo@sha`, branch | none |
| Triggered / Created | "shop-ci (service account) · Oct 5, 15:02 · 12 minutes ago": the plan's creator by display name, not email, and `create_time` | The creator and time, today's "Created by" line |
| Labels | As UIOps: a release plan can get an issue like any plan, and a project that forces labels then needs them here | Today's inline issue labels, editable in place, required when the project forces them; absent when the plan has no issue |
| Release / Changes | "Migration-based · 493 files · update to version `202610051530`", the release's highest version in D1's order, or "State-based · `<schema version>`", from the release's type and files | "DDL", "DML" or "Mixed" on N databases, from the plan check's `statement_types`; database groups are counted beside the databases, and a kind of target the plan does not have is left out: "DML on 1 database group", "DDL on 2 databases · 1 database group" |
| Review | As UIOps | The issue's approval state, "✓ Approved by alex (DBA) · Oct 5, 14:45", or who is next; absent when the plan has no issue |
| Progress | The plan list's stage cell (`ProjectPlanDashboardPage.tsx`): each stage's status icon and name, joined by arrows, wrapping when there are many | Same |

The block folds. It opens unfolded every time the page loads, with "Hide details" at its top right,
the plan page's existing link text for its sections. Folded, it keeps one line in the same box: the
Progress cell, after the Source title on a release plan, with "Show details" at the right. The fold
lasts while the page is open and is not remembered, so the next plan opened, or this one reloaded,
starts unfolded. The description is not part of the block and keeps its own Show more.

The description stays below the block, rendered and clipped as today (`PlanDetailHeaderDetails.tsx`:
150 characters or 3 lines, then Show more), so the rows keep their place whatever its length. A
release plan's collapsed Changes line then shows only the targets per stage. "Migration-based" and
"State-based" are the docs' names for the two GitOps workflows; the console's "Versioned" and
"Declarative" elsewhere are a separate copy change. Rejected: the PR title as the plan title, which
breaks the link from the CI summary and the plan list, and is absent on a rebase merge or an
undetected platform; a side panel for the block, which competes with the file viewer's side panel
(D8) and takes width the folded cards need; the description as the block's last row, which turns
the block into a document container under a second label and gives every CI-created plan an empty
row; remembering the fold across plans, which would hide Source and Review on the next plan without
the reader choosing it; folding by default, which hides the facts a reader opens the page for; the release facts in the Changes summary instead of a block, which leaves Progress no place and
splits plan facts between two spots; a statement count (it needs a parse the page does not have)
and a summed row estimate (each database differs; its card has its own); counts in the Progress
row, which the stage tabs and chips already carry.

**D11 · No error classification.** The page states the file, the statement, the error and what
applied, and stops. Rejected: recognizing "already exists", lock timeouts and constraint
violations and proposing a next step per class. Every class needs its own detector and its own
advice, both wrong on the next engine or driver, and the replays show users reading the error
themselves and fixing the file in git.

**D12 · No release detail page; reading a file's SQL lives on the plan.** Changes and the viewer
take over reading a file's SQL, and the plan already shows everything else the release page showed.
Main retired that page before this PR (#21637): `releaseDetailRedirectLoader`
(`frontend/src/app/router/releaseDetailRedirect.ts`) keeps the CI summary's Release link
(`action/github/summary.go`) working by opening the plan when exactly one active plan references the
release, and the Releases list filtered to the release otherwise, so a release no plan references
still lands somewhere. This PR relies on that redirect and adds nothing to it. The retired page was
also the only place that archived and restored a release (`deleteRelease` and `undeleteRelease`), and
#21637 gave neither a new home, so today only the API archives or restores one; that gap is BYT-10419,
not this PR. Rejected: restoring a release page as the place to read a file, which leaves two places
to open a file and the manifest shown twice.

## States

The spine, per task status:

| Task | Sources | Sections, top to bottom |
|---|---|---|
| Not started, pending | Stored state, caption "As of rollout creation, `<time ago>`; the run re-checks the database.", the age from the stored computation time | "N files already applied to this database", folded; one "Will apply" section per file to apply, in version order, no children and no caret; "Will apply again · revision deleted" where a revision was deleted, one short line so the path keeps its width, with an info icon whose tooltip, also reached by keyboard, reads "Earlier revision deleted `<date>` by `<who>`" |
| Running | The live log, the release's file list, and the stored state for files not reached | "N files already applied to this database", folded; reached files (applied collapsed, the current one open with a running indicator); unreached files below as "Will apply"; the boundary moves down; "Revisions on `<database>`" at the end once a file is applied |
| Done | The log and the release's file list | "N files already applied to this database", folded (the files without a marker); every file section Applied; task-level steps as siblings; "Revisions on `<database>`" at the end |
| Failed | The log and the release's file list | "N files already applied to this database", folded; applied sections collapsed; the failed section open, its failed row open with the error on the line and the statement block under it (folded below 640 px); task-level steps as siblings; nothing for the files after the failed one. An error that names no file or step (D3) ends the spine with an open "Run failed" section instead, after the files already applied and any applied sections; "Revisions on `<database>`" at the end when a file was applied |
| Canceled | The log and the release's file list | As Failed, with the section the cancel interrupted marked Canceled |
| Skipped, after a run | As Failed | "Skipped by `<user>` · `<time>` · `<reason>`", then the last run as Failed |
| Skipped, never run | Stored state | "Skipped by `<user>` · `<time>` · `<reason>`", then the planned list behind a closed "Planned at rollout creation · N files to apply" toggle; opened, it has no "re-checks" caption and marks each file to apply "Not applied" instead of "Will apply", since the task will not run |
| Revision page, any run | The log and the release's file list | The sections of that run with the open and closed defaults of the run's status; no stored state, no Revisions link |

Every row starts with the count line and follows the long-list rule (D3): at most 20 file sections
plus the running, failed or canceled one, the rest behind load rows.

The folded card's description, per task state (D3):

| Task | Release-backed | UIOps | Tint |
|---|---|---|---|
| Not started | "3 to apply · 490 already applied", or "Nothing to apply · 493 already applied", then "· scheduled 18:00" or the waiting cause; a declarative release shows the status only | "DML · about 1,180 rows", then the schedule or waiting cause | gray |
| Running | "Applying 3 files · started 15:14", or "Running · started 15:14" with nothing to apply | "Running · started 15:14" | blue |
| Done | "All 493 applied · 4.1 s", whatever the stored list held; a declarative release shows the status and duration | "DML · 4.1 s", "DDL · 1.2 s" or "Mixed · 3.4 s" | green |
| Failed | `<failed version>` then the error's first line, or the first line alone when the error names no file | The error's first line | red |
| Skipped | "Skipped by `<user>` · `<reason>`" | Same | gray |
| Canceled | "Canceled · `<time>`" | Same | gray |

The title row's status is today's; with a skipped task it reads "Deployed, 1 task skipped".

Changes:

| Rollout | Rail |
|---|---|
| Not started | Open: today's spec tab and Targets, then the manifest line, groups, 12 rows then "Load 12 more" and "Load all (N)", "Show fewer" once extended |
| Started or finished | Collapsed: the targets per stage (the release facts are in the details block, D10); opens on click to the same content |
| Declarative release | One line: the schema folder at its version, with Open |

The viewer, by the page's width:

| Page width | Container | Behavior |
|---|---|---|
| 1280 px and wider | Column docked on the right half of the page | The plan narrows to the left half and stays interactive; another file swaps the content; Close or Esc closes |
| 640 to 1279 px | Standard-width sheet over a scrim | Same body; on close the sheet keeps showing its file until it has slid away, as the UX contract's sheets do, so no empty panel slides off |
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
| Frontend | `frontend/src/routes/project/plan-detail/components/PlanDetailHeaderDetails.tsx`, `PlanDetailMeta.tsx` | The details block for release and UIOps plans, with the Source row and the plan list's stage cell for Progress, its fold (unfolded on every load, held in component state), above the description, which keeps today's clipping (D9, D10). |
| Frontend | `frontend/src/locales/*.json` | New strings; the retired release-page strings removed. |
| Backend | `proto/store/store/task.proto`, `proto/v1/v1/rollout_service.proto`, `backend/api/v1/rollout_service_task.go`, `rollout_service_converter.go` | The release state field on the task payload, computed at creation with the store's `Versions` filter, returned on the v1 task (D4). |
| Backend | `proto/store/store/task.proto`, `proto/v1/v1/rollout_service.proto`, `backend/store/task.go`, `backend/api/v1/rollout_service.go` | The skipper and skip time written by `BatchSkipTasks`, output-only on the v1 task (D7). |
| Backend | `proto/store/store/release.proto`, `proto/v1/v1/release_service.proto`, `backend/api/v1/release_service.go` and its converter | The optional `VCSSource` fields, converted through (D9). |
| Action | `action/command/rollout.go`, `action/command/vcs_user.go`, `action/command/root.go` | `getVCSSource` per platform from the table in D9; the GitHub event payload read shared with the VCS user code; the four `--source-*` flags. |

Every element on the page renders with the project Developer role, which reads plans, rollouts,
task runs, logs, revisions and the release's sheets today; the read path adds no permission.

Tests to write, per the placement rules in `AGENTS.md`:

- `model.test.ts`: a file group ending at a database-sync entry (today's fixture puts the sync under
  the last file; the new one proves it is a sibling); the files already applied derived for a done
  task (every file without a marker) and for a failed task (the files below the newest marker
  without a marker); a running task listing the files it has not reached as
  "Will apply" from the stored state, the boundary at the current file; a stored "Will apply"
  version the run passed over joining the files already applied, and a stored applied version that
  gets a marker leaving them; a failed task listing nothing after the failed file; a failed run with no marker at
  all (the driver cannot connect) and one that failed loading the second file's statement after one
  marker, each taking the versions above its newest marker from the stored state and ending with an
  open "Run failed" section holding the full error, and the same run without a stored state listing
  only the versions below it; a revision-write failure after a file ran, the file's section applied
  and the error in "Run failed"; a file-prefixed error on the newest file's section, with "no
  statement identified" when it has no failed command; a done run with no marker, all applied;
  a collapsed failed section carrying the error's first line; the count line per task state; the
  cap of 20 with the running and a failed section past it still shown, each hidden stretch's load
  row named by its state with "more" only after a shown section of that state, and "Load 20 more"
  and "Load all" on a stretch over 20; the files already applied passed through as one fold, never
  windowed; a prior backup as a sibling
  and a gh-ost migration inside its file's section; a run with no command response carrying the
  run's error on the header; the log-only form without a task keeping the run's defaults.
- `useTaskRunLogData.test.tsx`: a run that reached two files of a 500-file release fetches two
  sheets; a re-run version keeps the stored "revision deleted" note on its section.
- Planned spine tests: a version whose revision was deleted reads "Will apply again · revision
  deleted" on its chip, without the date or the deleter, and its info icon names both.
- Run spine tests: a re-run version's applied section reads "Applied again" with the info icon
  naming the earlier revision's deleter and time, and no line under the section; a version the
  stored state counted applied that the run applied reads "Applied again" with the
  after-the-rollout tooltip; without a stored state, "Applied". Stored ranges expand to the
  release's files, a range past the list stopping at its end. A task skipped before it ran shows
  the planned list behind the closed "Planned at rollout creation" toggle, its files "Not applied",
  with no caption.
- Folded card tests: each cell of the card table in States from the task, its latest run, the
  stored state and the plan check, before any log loads; the executor prefix removed from the
  error's first line; the release's file count on a done release task; DDL, DML and Mixed from the
  statement types, with the run's duration; an empty stored list reading "Nothing to apply" and
  "Running" by state, and "All N applied" once done, as does a done task whose list was not empty; the description under the database below 640 px; every task folded on load, the stage selected for
  a failed, a running and a deployed plan, and a URL naming a task opening it.
- Header click tests, folded and open: a click on the line's empty space and on the card padding
  toggles once; the task name and the caret toggle once, not twice; the checkbox, an action button
  and the margin around each do not toggle; a click ending a text selection does not toggle.
- Details block tests: each row for a release plan and a UIOps plan, the rows a plan kind lacks
  absent, and a release plan with an issue keeping Labels and Review; Migration-based and
  State-based; the Progress row's stage cells wrapping at six stages;
  the labels editable in place and required when the project forces them, and in
  `plan-detail-lifecycle.spec.ts` every row the same height once a label is set; the description below the
  block clipped at 150 characters or 3 lines; the block unfolded on load, folding to the Progress
  line (after the Source title on a release plan) and back, and unfolded again after a reload or on
  another plan.
- Changes rail tests: descending order over timestamp, counter and dotted versions; the grouping per
  shape with the newest group open, and over a hundred counter or dotted versions as one flat list; 12 rows, "Load 12 more", "Load all" and "Show fewer"; a backfilled low version landing in its
  old group; a row's name, folder and version as separate parts, a path with no folder showing the
  name alone, and a click anywhere on the row opening the viewer; the collapsed line; the
  declarative one-liner.
- Viewer tests: the diagnostic placed from a byte range on a multi-byte statement, opened from the
  failed section; opening at the top with the error when there
  is no range; the per-database line when opened from a section and none from Changes; the
  file-at-commit link per platform and none on an unknown one or for a declarative pattern path
  with `*`, `?`, `[` or `{`, while a versioned file named with `?` or `[` keeps it; the container chosen
  by the page's width; swapping files in the docked column; Close and Esc; the plan's state, such
  as loaded rows, kept when the column opens and closes.
- Source row tests: each row of the States table from a `VCSSource` with that subset; a squash-merge
  title ending in `(#N)` rendered with the number once; the 72-character truncation and tooltip; a
  title with markdown, control and bidirectional-override characters rendered as text; no link,
  in the row or the viewer, from a URL that is not `http` or `https`; the parse of `repo@sha` from each
  platform's commit URL.
- Action panel tests: each task line with its status chip, and a failed task's error first line, for
  a UIOps and a release task; the reason field shown and required for a release task, single and
  multi-select, and the skipper and time shown on the task afterwards; the UIOps path unchanged.
- `backend/api/v1`: rollout creation stores the state for a release-backed task against PostgreSQL
  through `testcontainer`: a target with two of three versions applied, a new target, and a target
  with a deleted revision carrying the deleter and time. Collision coverage, since the lookup keys
  on instance, database and version: the same version with a live revision on a sibling database of
  the same instance must not count for the target, and the target's own deleted revision of that
  version must not count either; the sibling's state is asserted unchanged. The pending files
  stored as ranges, one range for a new database on a 500-file release; a deleted release creating
  its task with no state. In `backend/store`, `ListPendingReleaseVersions` returning only the pairs a database lacks: a live `VERSIONED` revision applied, a deleted or declarative one not, the newest deletion chosen, and a sibling database's or another instance's revision not counted; a user's email change rewriting the skipper and the
  deleted-revision deleters, and not another user's email that contains it. `BatchSkipTasks` writes
  the skipper and the time on each selected task and on no other task of the stage, and the v1 task
  returns them.
- `action/command`: `getVCSSource` per platform from environment fixtures and a GitHub payload file:
  PR event, squash-merge push, merge-commit push, rebase push (commit title, no PR), Bitbucket push
  (no title), undetected platform with and without the flags, flags overriding detection; the
  source branch, not the synthetic ref, in a GitHub pull-request run, a GitLab MR pipeline and an
  Azure PR build, and a slashed Azure branch kept whole; no branch on a GitHub tag push, a GitHub
  merge queue run, a GitLab tag pipeline or an Azure tag build.
- `frontend/tests/e2e/plan-detail`: a release-backed plan created through the API against the shared
  server with a target that already carries one version, read as a project Developer: the Changes
  rail newest first with the viewer opening from a file path; the planned spine before the run;
  after the run the already-applied fold opening to its indented row, and the applied section with
  the Revisions link; a failing second of three files showing the error on the row and in the viewer, nothing
  listed for the third, and the error on the folded card; as a project Owner, Skip refusing an
  empty reason, the task then showing the skip line over the failed run, and the header reading
  "Deployed, 1 task skipped"; the release URL redirecting to the plan. Three journeys that need the
  real executor's history: a re-run, where one version's revision is deleted before the rollout is
  created and another's after it, reading "Will apply again · revision deleted" with the deleter on
  its icon before the run, then "Applied again" on both with their two tooltips and the untouched
  version still in the fold; a 25-file release failing at its 22nd file, showing the count line, a
  load row, the failed file and its error past the row limit, and the Revisions link, then a release
  fixing the file whose fold holds the 21 and whose run finishes; and the header line in a real
  browser, where a click on its empty space folds and unfolds the card and a click on the margin
  around the checkbox or an action button does not.

## Not in this PR

- Archive and restore of a release in the UI (BYT-10419). Main retired the release page that held
  them (#21637, D12), so only the API archives or restores a release today.
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
  control beside the "As of rollout creation" caption that recomputes the list read-only and does not
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

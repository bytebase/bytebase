# What ran, in the rollout job

Status: proposal · 2026-11-27 ·
[BYT-10580](https://linear.app/bytebase/issue/BYT-10580/show-the-migration-script-in-the-sdl-deploy-job)

A deploy job's summary says which databases changed, but for a state-based (SDL) rollout it shows
none of the SQL that ran. **The rollout result groups databases by the script they ran and shows
each script as the SQL that ran, in order, the same way for SDL and migration rollouts. Fixed
counts bound the summary, and every cut says how much is left and links to the rollout in
Bytebase.**

This builds on [bytebase-action-rollout-result.md](bytebase-action-rollout-result.md), which
defines the result, and changes how its job log and step summary show it.

## Problem

Verified on `main` at `c84fbc9ba6`.

- **An SDL database lists no statements.** The summary lists only commands that changed rows
  (`commandLines` in `action/result/render.go`). DDL changes none, so an SDL database reads
  "3 statements" over a block that holds only the file's version.
- **The text is already in the result.** The deploy logs each generated statement's text
  (`runDeclarativeRelease` in `backend/runner/taskrun/database_migrate_executor.go`), and the
  action keeps the first 100 per file.
- **The same script repeats per database.** Each changed database gets its own block, so 300
  databases that ran one file make 300 identical blocks.
- **Nothing says whether databases differed.** A reader compares the Statements column, a count
  that leads to no SQL.
- **The headline is an enum.** "Rollout to test: DONE" does not say what was deployed where.
- **Names need decoding.** In "crm on prod failed", the reader works out which name is the
  database and which the instance.
- **Statement blocks are cut silently.** They stop when the 900 KiB budget runs out, with no word
  about the rest.
- **The rollout link is rare above the result.** It shows there only when no stage finished or the
  job was interrupted.

## Goals

- **G1** An SDL job shows the SQL that ran on each database, in the order it ran.
- **G2** The reader sees at once whether databases ran different SQL, and what each group ran.
- **G3** SDL and migration rollouts share one layout.
- **G4** Fixed counts bound what is shown. Every cut says how much is left and links to the
  rollout.
- **G5** The `--output` file and the exit code do not change.

### Non-goals

- **The SQL preview in the review comment.** That is BYT-10578, deferred. It can reuse this
  grouping.
- **A pull request comment for the deploy.** The result reports on the job, as the result design
  explains.
- **Grouping in the `--output` file.** The file keeps each database's own commands.
- **A server or proto change.**

## Proposed approach

### What a reader asks

The summary answers these in order.

1. Did my change deploy?
2. Did any database run something different?
3. What exactly ran on a database?
4. For a failure: which database, which statement, and what do I do now?

### The layout

```text
✅ Deployed to test                                               headline
9 changed · 4 different scripts · Open the rollout in Bytebase    counts and link
3 databases were retried; their earlier runs are in Bytebase.     notes, when any
❌ Database tenant_043 failed                                     each failure, when any
Script A · 5 databases                                            the scripts, A to E
All 9 databases                                                   the table, folded
```

- **The headline is the answer.**
  - "✅ Deployed to test" when the stage finished and at least one database changed.
  - "✅ No changes to test" when it finished and none changed. The counts line says why, as "9
    already up to date" or "4 skipped". A finished stage is not a deploy, as the result design
    holds.
  - "❌ Deploy to prod failed on 2 of 300 databases" on a failure.
  - "⛔ Deploy to prod canceled on 1 of 300 databases" when a task was canceled.
  - An interrupted job, a missing stage and no `--target-stage` keep today's headlines, which
    already say that nothing finished or nothing was deployed.
- **The counts line says whether databases differed**: "4 different scripts", left out when every
  database ran the same one. It ends with the link to the rollout.
- **Notes sit under the counts line**: the databases that were retried, whose logs could not be
  read, or whose SQL is past its cap. Each is defined below.
- **Failures come before the scripts**, as in a migration rollout. Each one is a heading and
  labeled lines:

  ```text
  ❌ Database tenant_043 failed
  Instance: prod-pg-1
  File: migrations/0004_add_risk_score.sql
  Before it stopped, it applied migrations/0003_backfill_grade.sql (4,812 rows).
  ```

  Then the statement, the error and the link to retry or skip it, as today.
- **An SDL file is named by its release's path**, the file pattern the deploy read, such as
  `schema/*.sql`. The log records no path for it, and its version is a timestamp.
- **The table is folded**, under "All 300 databases", failed first. Its Script column names the
  script each database ran. The Instance column shows only when the stage spans more than one
  instance, and Rows only when a database changed rows.
- **Databases that ran nothing are counted**, as "2 already up to date" in the counts line.

### Scripts

A script is everything one database ran, in order. Databases that ran the same script share it.

- **A migration script is files.** A file that finishes has run all its statements, so two
  databases ran the same script when they finished the same files in the same order. A sheet
  task's script is its sheet. A `--plan` that runs sheets has no release files.
- **An SDL script is statements, matched by text.** The deploy generates the SQL for each
  database, so two databases ran the same script only when every statement matches.
- **Script A is the one most databases ran.** The others follow by how many databases ran them.
  Ties, and the databases in a script, follow the stage's task order.
- **Each script is the SQL that ran, in full and in order**, in a `sql` block. Nothing in it
  marks how it differs from another script. The counts line says how many scripts ran, and each
  script lists its databases.
- **Rows are in the script.** A statement that changed rows gets a comment above it: `-- 25 rows`
  on one database, `-- 25 rows on each database` when the script's databases agree, and
  `-- 36 rows across 3 databases` when they differ. A bare sum would read as one statement's
  count. A count shows only when every database reported one. A file's counts line up by
  position, so they show only when every database ran the same number of its statements. The
  table's totals tell zero from no count.
- **Only finished databases that ran statements have a script.** A failed database is in the
  failures. A skipped one, and one that logged no statement, are in the table.
- **A script covers each database's latest run.** A retry skips the files an earlier run applied,
  so a retried database's script is what its last run applied. The summary says so once: "3
  databases were retried; their earlier runs are in Bytebase."
- **A file whose text cannot be read shows no SQL.** The log gives a migration statement as a
  byte range, and the job may have neither the checkout nor `bb.sheets.get`. The script then shows
  the file's name, its statement count and its rows.
- **A finished database whose log could not be read is counted**: "2 databases' statements could
  not be read; they are in Bytebase."

### Names

- **A database is named alone**, as `tenant_043`. Its instance is added, labeled, only when another
  database in the stage has the same name: `hr (instance prod-pg-2)`.
- **A script names its first five databases** under its heading, and lists up to 50 in a folded
  list when there are more. The log cannot fold, so it lists them once, up to 50.
- **In the log, a database is named as the log's lines name it**, as
  `instances/prod/databases/hr`, so a search for a database finds its script.

### Limits

| What | Shown | Past the limit | Applies to |
|---|---|---|---|
| Failures | 10 | "and 5 more failed databases, in Bytebase" | Summary |
| Table rows | 50 | "and 250 more databases, in Bytebase" | Summary |
| Scripts | 5 | "and scripts F–H on 3 databases, in Bytebase" | Summary and log |
| Statements in a script | 50 | "and 450 more statements, in Bytebase" | Summary and log |
| Databases named under a script | 5 | "and 295 more" | Summary |
| Databases listed in a script | 50 | "and 250 more databases" | Summary and log |
| Lines of one statement | 30 lines of 200 characters | `-- 14 more lines`, then "(shortened; the full text is in Bytebase)" | Summary and log |

- **"in Bytebase" links to the rollout** on every line.
- **The scripts left out are named by letter**, since the table's Script column shows those
  letters too.
- **A statement's lines left out are a SQL comment**, so the block still reads as SQL.
- **A script longer than 12 lines starts folded**, with its statement count in the title, in the
  summary and on GitLab.
- **The 900 KiB budget stays as a backstop** under GitHub's 1 MiB limit. A script that does not fit
  ends the section, and the rest are counted. A later, smaller script is never shown in its place.
- **The statement limit is the one failures use today.** A failed statement keeps today's note,
  since the output file always holds its full text. A script's statement may be past the file's
  first 100, so its note names Bytebase only.

### Many statements

- **Comparing SDL scripts needs every statement**, not the first 100. An SDL file's statements are
  all kept, and each distinct text is held once for the whole result.
- **Both caps give the same answer in any order.** Logs are read in parallel, so a cap that
  depends on which log arrived first would change the summary between runs.
- **One database keeps at most 1 MiB of SDL text.** A database past it has no script and is
  counted: "Statements run on 4 databases are not shown; they are in Bytebase."
- **The result holds at most 16 MiB of distinct text.** Past it, the scripts are replaced by one
  line linking to the rollout. The total does not depend on order, so collection can stop holding text
  as soon as it is reached.
- **Migration files keep no extra text.** Their scripts compare files, and the text is read from
  the file as today.
- **The output file keeps its bound**: the first 100 commands per file and any that failed.

### The job log

- **The log opens with the summary's headline, counts and notes**, without ✅, ❌ or ⛔ and
  without the link.
- **Every database and every failure stays in the log**, one line per database with its script's
  letter. The failure and table limits are for the summary only. A log has no size limit, and
  readers search it for a database.
- **The scripts follow the database lines**, with the script limits, so the log and the summary
  read alike.
- **A long script folds on GitLab**, with section markers. A failure is never folded.
- **The log still ends with the rollout's URL.**

### Example

The real run behind this design: an SDL rollout to nine test databases, which had started from
four different schemas.

````markdown
### ✅ Deployed to test

9 changed · 4 different scripts · [Open the rollout in Bytebase](https://bytebase.example.com/projects/example/plans/101/rollout)

#### Script A · 5 databases

tenant_01, tenant_02, tenant_03, tenant_04, tenant_05

```sql
CREATE TABLE "public"."department" (
    "id" integer NOT NULL,
    "name" text NOT NULL,
    CONSTRAINT "department_pkey" PRIMARY KEY ("id")
);
ALTER TABLE public.employee ADD COLUMN job_title text;
CREATE INDEX idx_employee_job_title ON public.employee (job_title);
COMMENT ON SCHEMA "public" IS NULL;
```

#### Script B · 2 databases

tenant_06, tenant_07

```sql
DROP INDEX public.idx_employee_name;
CREATE TABLE "public"."department" (
    "id" integer NOT NULL,
    "name" text NOT NULL,
    CONSTRAINT "department_pkey" PRIMARY KEY ("id")
);
ALTER TABLE public.employee ADD COLUMN job_title text;
CREATE INDEX idx_employee_job_title ON public.employee (job_title);
COMMENT ON SCHEMA "public" IS NULL;
```

#### Script C · 1 database

tenant_08

```sql
COMMENT ON SCHEMA "public" IS NULL;
```

#### Script D · 1 database

tenant_09

```sql
CREATE TABLE "public"."department" (
    "id" integer NOT NULL,
    "name" text NOT NULL,
    CONSTRAINT "department_pkey" PRIMARY KEY ("id")
);
CREATE TABLE "public"."employee" (
    "id" integer NOT NULL,
    "name" text NOT NULL,
    "job_title" text,
    CONSTRAINT "employee_pkey" PRIMARY KEY ("id")
);
CREATE INDEX idx_employee_job_title ON public.employee (job_title);
COMMENT ON SCHEMA "public" IS NULL;
```

<details><summary>All 9 databases</summary>

| Database | Status | Script | Duration |
|---|---|---|---|
| tenant_01 | ✅ Changed | A | 0.1s |
| … | | | |

</details>
````

A migration rollout looks the same, with each file's path as a `--` line above its statements, and
a rows comment, such as `-- 1,200 rows on each database`, above each statement that changed rows.

#### A failed SDL rollout

A later real run made `employee.email` required, from the same four starting schemas. tenant_03
and tenant_07 had employees without an email, so adding the column failed there. The other seven
had none and changed.

````markdown
### ❌ Deploy to test failed on 2 of 9 databases

7 changed · 2 failed · 4 different scripts · [Open the rollout in Bytebase](https://bytebase.example.com/projects/example/plans/112/rollout)

#### ❌ Database tenant_03 failed

Instance: `test-pg-1`<br>
File: `schema/*.sql` (rolled back)

```sql
ALTER TABLE public.employee ADD COLUMN email text NOT NULL;
```

```text
ERROR: column "email" of relation "employee" contains null values (SQLSTATE 23502)
```

[Retry or skip this task in Bytebase](https://bytebase.example.com/projects/example/plans/112/rollout/stages/test/tasks/294)

#### ❌ Database tenant_07 failed

…the same lines, statement, error and link…

#### Script A · 4 databases

tenant_01, tenant_02, tenant_04, tenant_05

```sql
CREATE TABLE "public"."department" (
    "id" integer NOT NULL,
    "name" text NOT NULL,
    CONSTRAINT "department_pkey" PRIMARY KEY ("id")
);
ALTER TABLE public.employee ADD COLUMN email text NOT NULL;
ALTER TABLE public.employee ADD COLUMN job_title text;
CREATE INDEX idx_employee_job_title ON public.employee (job_title);
COMMENT ON SCHEMA "public" IS NULL;
```

…Scripts B to D…

<details><summary>All 9 databases</summary>

| Database | Status | Script | Duration |
|---|---|---|---|
| tenant_03 | ❌ Failed | – | 0.0s |
| tenant_07 | ❌ Failed | – | 0.1s |
| tenant_01 | ✅ Changed | A | 0.0s |
| … | | | |

</details>
````

An SDL failure reads like a migration one. Its statement is the SQL the deploy generated for that
database.

Its job log:

```text
Deploy to test failed on 2 of 9 databases
7 changed · 2 failed · 4 different scripts

FAILED  instances/test-pg-1/databases/tenant_03
        schema/*.sql  failed, rolled back
          ALTER TABLE public.employee ADD COLUMN email text NOT NULL;
          ERROR: column "email" of relation "employee" contains null values (SQLSTATE 23502)
        Retry or skip: https://bytebase.example.com/projects/example/plans/112/rollout/stages/test/tasks/294
FAILED  instances/test-pg-1/databases/tenant_07
        schema/*.sql  failed, rolled back
          …
DONE    instances/test-pg-1/databases/tenant_01  script A · 1 file · 5 statements · 0 rows · 0.0s
…

Script A · 4 databases
  instances/test-pg-1/databases/tenant_01, instances/test-pg-1/databases/tenant_02, …
  CREATE TABLE "public"."department" (
  …

Rollout: https://bytebase.example.com/projects/example/plans/112/rollout
```

### What changes from the result design

- **"Rollout to test: DONE"** becomes "✅ Deployed to test", or "✅ No changes to test" when no
  database changed, with a counts line that says whether databases differed.
- **"Each changed database's statements in a collapsed section"** becomes one script per group of
  databases that ran the same SQL.
- **"The first ten commands that changed rows"** becomes every statement, up to the script limit.
- **"One table row per database" at the top** becomes a folded table, after the scripts.
- **"A finished stage carries no rollout link"** becomes a link in the counts line, so the
  rollout is one click away even when nothing is cut.
- **"crm on prod failed"** becomes "Database crm failed", with the instance on its own labeled
  line.
- **"Each command shows its statement beside its count"** stays true in the output file. The log
  and the summary show one count per statement for a script's databases: the count on each
  database when they agree, and the sum when they differ.
- **"Rows until the budget ends"** becomes fixed counts, with the budget as a backstop.
- **"Each finished database folds on GitLab"** becomes each long script folding, since a finished
  database is one line.

The result design is updated to match, and points here.

### In the code

- **`action/result`** builds the scripts from the result, applies the limits and renders both the
  log and the summary.
- **An SDL file is one whose log computed a diff**, which only the declarative path logs.
- **`action/README.md`** updates its job log and GitHub entries.

### Testing

- Databases with identical SDL statements share one script, and a database with one more or one
  fewer statement gets its own.
- Each script is the SQL its databases ran, in order, with nothing marked.
- One migration file on many databases is one script. Databases that did not run one of the files
  get their own script.
- A count reads per database when the script's databases agree, and is summed across them when
  they differ. It shows only when every database reported one.
- A sheet task's script is its sheet.
- A file whose text cannot be read shows its name, count and rows.
- A retried database is under its last run's script, and the retry line is written.
- A script past five databases lists them, up to the limit.
- Scripts run by the same number of databases keep the stage's task order.
- Each limit writes its line with the link, and the scripts left out are named by letter.
- An SDL failure is named by its release's file pattern, and by its version without the release.
- A long statement is shortened.
- The budget ends the section at a script.
- Unchanged databases and unreadable logs are counted.
- A database past 1 MiB is counted, and a result past 16 MiB shows one line.
- Logs read in a different order give the same summary.
- A failure names the database and labels its instance and file.
- A finished database that logged no statement has no script.
- A finished stage where no database changed reads "No changes", and a canceled one "canceled".
- The notes sit under the counts line.
- A database's instance is added only when another database has its name.
- A script of 12 lines or fewer is open, and a longer one folds, in the summary and on GitLab.

## Considered approaches

**How to group**

A first version grouped by statement and ran on nine real databases. Its blocks nested as 9, 8,
7, 2 and 1 databases, and the reader had to rebuild each database's script from them.

| | Per database (today) | Per statement (first version) | Statement matrix | Per script (chosen) |
|---|---|---|---|---|
| Did any database differ? | Compare counts | Compare block sizes | Matrix columns | "4 different scripts" |
| One database's SQL, in order | Yes | No | Order of rows only | Yes |
| 300 databases, one script | 300 identical blocks | One block | One column | One script |
| 290 with A, B and 10 with A, B, C | 300 blocks | Two blocks, "A, B on 300" and "C on 10" | Two columns | Two scripts, each in full |
| Long statements | Full | Full | One line, full text folded | Full |
| On a phone | Long | Fine | Wide with many scripts | Fine |

**How to show a script that differs from script A**

| | A diff against A | The SQL, differences marked green | Only the difference | The SQL that ran (chosen) |
|---|---|---|---|---|
| What the block is | `+` and `-` lines, unchanged runs folded | The script, with a marker column | Extra statements, and a list of A's it skipped | The script |
| Reads as | A diff to work out | A diff, though it is the script | A patch on A | What ran |
| Copies and runs as SQL | No | No | No | Yes |
| Finding why it is separate | Easy | Easy | Easy | The reader compares it with A |

Marks on top of the SQL made the block read as a change instead of as what ran. The cost of
leaving them out is in long scripts that differ by one statement, where the reader compares the
two by eye. The counts line and the database lists still say that they differ.

**How to bound it**

| | Bytes only (today) | Fixed counts and a byte backstop (chosen) |
|---|---|---|
| Reader can predict what shows | No | Yes |
| Says what was cut | Table rows and failures | Every part |
| Fits GitHub's limit | Yes | Yes |

**SDL only, or both kinds**

- **SDL only** shows the SQL with the smallest change. Migration rollouts keep 300 identical
  blocks, and the two kinds read differently.
- **Both kinds (chosen)** cost a second rule for what makes two scripts the same, by file.

**Keep every statement for every file**

- It would treat both kinds alike, but migration scripts need no text to compare, since they
  compare files. A file of fifty thousand inserts on a thousand databases would hold all of it,
  which the 100-command bound exists to prevent. Only SDL files keep every statement.

## Open items

- **Commands from an attempt retried inside one run** stay in the output file and in Bytebase,
  not in the scripts. Today the summary marks them beside rows.
- **The limits are first numbers.** Tune them after real rollouts.
- **An unchanged SDL database can read as changed.** On PostgreSQL the differ adds
  `COMMENT ON SCHEMA "public" IS NULL` to a database whose schema already matches, as tenant_08
  shows in the example. That is the differ's to fix.

## Reference

How other tools show deploy SQL, from BYT-10580 and the research for BYT-10578.

| Tool | What the deploy job shows |
|---|---|
| Atlas | The SQL of each applied file, folded in the job summary. One run is one database |
| pgschema | The plan it applies, in the job log, unless `--quiet` is set |
| Flyway | The prepared script runs. Its output is not documented |
| Terraform | Results are on the run page, not the job |

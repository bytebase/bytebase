# GitOps setup for state-based (SDL)

Status: implemented in this change · 2026-11-25 · [BYT-10568](https://linear.app/bytebase/issue/BYT-10568) · builds on [gitops-setup-page.md](gitops-setup-page.md)

The project's GitOps page sets up migration-based workflows only. A team that wants state-based
(SDL) exports its schema and rewrites the workflow by hand.
**Setup gains a workflow type. For state-based it writes the workflow files and exports the first
schema files, so the repository gets everything it needs with nothing to edit. The workflow type is
a fourth choice that shapes every file, so the three steps become one page.** Frontend only: no
proto, API, store or migration change.

The mock-ups are kept outside the repo: the page layouts, and the options for the part that shows
while a choice is open.

## Problem

Verified on `main` at `745cdae1c`.

**Setup has no state-based.** `generateWorkflowFiles` in
`frontend/src/routes/project/gitopsWorkflowYaml.ts` writes migration-based files only. An owner who
wants SDL does four things by hand, and nothing checks them:

1. Exports the schema on the database page (`DatabaseExportSchemaButton.tsx`).
2. Unzips it into a folder in the repository.
3. Adds `--declarative` to the workflow.
4. Changes the file pattern from `migrations/*.sql` to one that reaches the nested export.

**The three steps do not fit one more choice.**

- The workflow type shapes every file, so it must come before the steps the page has today.
- Setup saves nothing, so Continue stores and checks nothing. It is three screens between the owner
  and the files, and it would be four.
- A choice and its result are on different screens. The databases and the identity are picked in
  steps 1 and 2. The files they shape are in step 3.

## Principle

> For state-based, setup hands over the workflow files and the first schema files, ready to commit.
> Every choice and its result are on one page.

## References

**State-based setup in other tools.** Each one exports the first files with a command:
[Atlas](https://atlasgo.io/inspect/database-to-code), [pgschema](https://www.pgschema.com/cli/dump),
[Skeema](https://www.skeema.io/docs/commands/init/),
[Flyway](https://documentation.red-gate.com/flyway/reference/commands/model) and
[Supabase](https://supabase.com/docs/guides/local-development/declarative-database-schemas). The
workflow is a separate step in each, and none shows at setup how other databases differ from the
exported one. They leave that to a plan or a dry run against each database.

- **Taken:** the first files are an export of one existing database.
- **Left out:** a separate step for the workflow. Here one page gives both (D4, D5).

**Two pages that turn a few choices into text to copy**, and save nothing. Screenshots of both are
in BYT-10568.

| | Supabase, "Connect to your project" | GitHub, "Add new self-hosted runner" | This design |
|---|---|---|---|
| The choices | Under the page intro. Cards, then rows with the label on the left | Under the page intro. Cards, then a select | The same (D7, D8) |
| A title over the choices | None | None | None (D7) |
| The result | Numbered steps, all visible | Headed blocks in order, all visible | Numbered steps, all visible (D12) |
| Heading over the result | "Follow these steps" | "Download", "Configure" | "Add the files to your repository" (D12) |
| Copy | A button on each block | On each block | A button on each file (D12) |
| First view | Every choice already picked | Every choice already picked | Only the workflow type is picked (D9) |

- **Taken:** no title over the choices, the result as steps that are all on the page, and a copy
  button on each block.
- **Left out:** a first view with every choice picked (D9), and Supabase's tinted background behind
  the steps. A rule separates the two parts here.

**What state-based cannot do** is the docs'
[Limitations](https://docs.bytebase.com/gitops/state-based-workflow/limitations) page. Setup shows
three of its points where the workflow type is picked (D1).

## Decisions

### State-based setup

**D1 · A workflow type choice comes first.** Two cards, named as in the
[GitOps docs](https://docs.bytebase.com/gitops/overview). Migration-based is picked at the start.

- *Migration-based.* "You write incremental migration files. They run in order of their version
  numbers."
- *State-based (SDL).* "You maintain the complete schema definition. Bytebase generates the
  migration DDL that brings each database to that state."

While State-based is picked, an info alert under the cards, "Before you choose state-based", gives
the limits that decide whether it fits, each with its point in bold:

- **Schema changes only.** Data changes need migration-based.
- **PostgreSQL recommended.** MySQL 8.0 or later is also supported.
- **Limited statements.** On PostgreSQL, the list from the docs.

A link opens the docs' full list. The server also deploys state-based to CockroachDB, but the page's
text names PostgreSQL and MySQL only.

**D2 · The folder follows the workflow type.** "Migration folder" starts as `migrations`, "Schema
folder" as `schema`. `schema` is the folder the
[docs](https://docs.bytebase.com/gitops/state-based-workflow/develop) tell users to unzip the export
into. A typed name is kept when the workflow type changes. Only an untouched default follows it.

**D3 · "Deploy to" warns about databases state-based cannot deploy to.** A database on another
engine than PostgreSQL, CockroachDB or MySQL fails at deploy, and the review does not say so. In
state-based, a warning under the deploy order lists these databases. It is the same kind of warning
as the one for databases with no environment, and like it, it does not block the files.

**D4 · State-based workflow files.** The same files as migration-based, with two differences: the
file pattern is `<folder>/**/*.sql`, and `check` and the `rollout` that creates the release pass
`--declarative`. `**` is needed because the export nests its files, and the action's glob supports
it (`action/command/file.go`).

**D5 · The schema files are a step of their own, and it can wait.** "Add the schema files" is
step 2 of the result (D12), for state-based only.

- *The export database is picked in the step.* "Export schema from" and its picker are one row, with
  the help text under it. The pick decides the schema files and nothing else, so it sits next to
  them, and the workflow files do not wait for it.
- *Which databases it lists.* The ones in "Deploy to" that have an environment and are on an engine
  state-based supports. The first deploy makes every database in "Deploy to" match the export, so a
  database from outside is not offered. The others are left out, because "Deploy to" already names
  them (D3).
- *It can be done later.* A repository with the workflow files and no schema files does nothing,
  because the workflow runs only when the schema folder changes. A repository that already holds its
  schema files skips the step.
- *It is not called optional.* State-based reads the folder as the complete schema. A first change
  in a folder that does not hold everything shows every missing object as a drop, and the review
  only warns. The step's description says the folder must be complete before the first schema
  change.

Setup does not compute how the other databases differ from the export database. That is one diff per
database (`diffSchemaViaSDL` in `backend/api/v1/database_service.go`, one request each, as the Sync
Schema page does), and a group can hold thousands. The review comment on the pull request shows the
SQL for each database ([BYT-10578](https://linear.app/bytebase/issue/BYT-10578)).

**D6 · The schema file box: one export, one download, a preview that scales.**

- *Format.* Always the multi-file export: `GetDatabaseSDLSchema` with `MULTI_FILE`, the ZIP the
  database page already offers. Both formats hold the same SDL, and the pattern in D4 also matches a
  single file for a team that wants one.
- *Paths.* The ZIP's own root is `schemas/`, so with the default folder a table is at
  `schema/schemas/public/tables/orders.sql`. This is the layout the docs describe.
- *Download.* One button gives the ZIP. The export is fetched once per database and kept while the
  page is open.
- *Preview.* The box names the folder, the file count and the database. Below it is one row per
  folder with its file count, and a row shows its path from that folder. A folder opens to its
  files, at most 50 at a time with "Show all N files", and a file opens to its SQL. A folder that
  holds one file lists that file directly. A schema with 12 files or fewer starts with its folders
  open. Above 20 files a filter field narrows the list by path and shows "13 of 241 files".
- *Empty.* A database with no schema objects shows a line that says so, and Download is off.

### One page

**D7 · One page, top to bottom.** Under the page intro are the choices, then a rule, then "Add the
files to your repository". There is no step row, no Back or Continue, and no title over the choices:
the page saves nothing, so a title such as "Configuration" would promise a saved setting. The two
parts share one column, at most 896px wide, so the fields and the file boxes end on the same edge.
The intro line is the same for both workflow types.

**D8 · The choices, in this order.** Both workflow types show the same fields, so switching the type
moves none of them.

| Field | Control |
|---|---|
| Workflow type | Two cards (D1) |
| Deploy to | Database group or Databases, as today |
| Sign in as | Workload identity picker, with Create workload identity |
| Migration folder, or Schema folder | Text (D2) |

The fields use the shared form layout: each label is on the left while the form is at least 560px
wide, and above its control when it is narrower. Help text sits under the control.

**D9 · Only the workflow type has a default.** The database group, the databases, the workload
identity and the export database start unpicked, also when the project has only one. Each of them
decides which databases change or who may deploy, so the owner picks it. This replaces the earlier
rule that a lone database and a lone identity are already picked.

**D10 · Deploy to and Sign in as keep their content in less space.**

- *Deploy to.* The tabs and pickers are unchanged, without the "Recommended" label on the Database
  group tab. The deploy order is one line, "84 databases in 3 environments: Test (4) → Staging (12)
  → Prod (68)", with the full list behind "Deploy order".
- *Sign in as.* Under the picker is the line that says why an identity is used. Then two lines: the
  git provider with its logo and the repositories that can sign in, then "Role in this project:
  GitOps Service Agent". The existing warning shows when the identity has no role.

**D11 · While a form choice is open, the steps are locked.** The result part shows no file. It shows
the steps to come, dimmed, with dashed numbers and one line each, so the owner sees what the choices
lead to. Above them a "Waiting for" line lists the open choices as buttons. A button brings its
field into view and opens its picker. A choice that needs a change says what it needs: "Needs a
GitOps role", "Needs a database with an environment". The open choices are "Deploy to" and "Sign in
as". The export database is not one of them (D5).

**D12 · The result is numbered steps, with every file on the page.** The heading is "Add the files
to your repository", as today, with the repository link under it. There are no tabs. Each box has
one solid button, its main action: Copy on a file, Download on the schema files.

1. **Add the workflow files.** "Create each file at the path shown." Each file is a box with its
   path, a copy icon for the path, the Copy button for the content, its first 12 lines and "Show
   all N lines". On GitLab the `.gitlab-ci.yml` box says "If your repository already has
   `.gitlab-ci.yml`, add these lines to it."
2. **Add the schema files.** State-based only (D5, D6).
3. **Open a pull request.** "Commit the files and open a pull request into the default branch."
   Then what happens next, as today. For state-based: "the next pull request that changes the schema
   files gets a Bytebase review comment with the migration DDL for each database."

Migration-based has two steps.

## States

| Choices | The result part shows |
|---|---|
| "Deploy to" or "Sign in as" open | "Waiting for" with the open choices, then the locked steps (D11) |
| Picked identity has no role | The same, with "Sign in as" marked "Needs a GitOps role" |
| Migration-based, both picked | Steps 1 and 3 |
| State-based, both picked | Steps 1, 2 and 3. Step 2 shows the picker |
| State-based, export database picked | Step 2 also shows the schema file box |
| GitLab identity | GitLab files, the `.gitlab-ci.yml` note, and "merge request" in step 3 |

| Schema | The schema file box shows |
|---|---|
| Still exporting | One "Loading" row |
| No schema objects | A line that says so. Download is off |
| 12 files or fewer | Folders open, no filter |
| More than 20 files | Folders closed, with the filter field |
| A folder with more than 50 files | Its first 50, then "Show all N files" |

| "Deploy to", state-based | The page shows |
|---|---|
| Some databases on an unsupported engine | The warning that lists them (D3). The files still show |
| Every database on an unsupported engine | The warning, and an "Export schema from" picker with nothing to pick |

The preconditions are unchanged: no database, no workload identity, and an unset External URL show
as they do today.

## Alternatives considered

**The schema files**

- **"Export schema from" as a form field.** Rejected. The workflow files would wait for a choice
  they do not depend on, the form would differ by workflow type, and the pick would sit far from the
  files it produces.
- **Holding "Open a pull request" back until a database is picked.** Rejected. A pull request with
  the workflow files alone is harmless (D5), and a team with its own schema files never picks one.
- **Calling the step optional.** Rejected (D5).
- **The shared database picker as it is, listing every database in the project.** Rejected. A
  database from outside "Deploy to" picked by mistake would make the first deploy drop objects on
  every target. The picker draws each database the way the shared one does.
- **Listing databases on unsupported engines as disabled rows.** Rejected. "Deploy to" already names
  them (D3).
- **Blocking the files while "Deploy to" holds an unsupported database.** Rejected. A group's
  members are not the owner's to change on this page, and the no-environment case is a warning too.
- **Showing how the other databases differ from the export database.** Rejected (D5).
- **Offering the single-file export.** Rejected (D6).
- **The picker row as the header of the schema file box.** Rejected in review of the mockup.
- **A separate page or entry for state-based setup.** Rejected. Every form choice is the same, and
  the two workflow types share their files apart from two lines (D4).

**The limits of state-based**

- **A warning, not an info alert.** Rejected. It is guidance at the moment of choosing, and "Deploy
  to" warns when a picked database does not fit (D3).
- **Every limit on the page.** Rejected. The docs page is long. Three points and a link fit under
  the cards.

**The layout**

- **A step for each choice, with the workflow type first.** Rejected. Four screens of Continue for a
  page that saves nothing.
- **Workflow type, then databases and identity on one screen, then the files.** Rejected. It keeps
  the steps and saves one click.
- **Every section open on one page, each with its full result.** Rejected. The files start below the
  first window on the smallest project.
- **The choices on the left and the files on the right.** Rejected in review of the mockup.
- **A tab per file.** Rejected. A state-based owner can copy two workflow files and never open the
  schema files tab.
- **A title over the choices.** Rejected (D7).
- **Picking a lone database or identity for the owner.** Rejected (D9).
- **A copy icon alone on each file, and no solid button.** Rejected. Each box has one thing to do
  with it, and a button with its name says so.

**While a choice is open**

- **Showing each file as soon as its own choices are made.** Rejected. A half-made setup offered
  schema files to download while "Sign in as" was still empty.
- **"Your files appear here" over a list of the choices.** Rejected. It lists what is missing, not
  what the choices lead to.
- **A checklist card with a count and a progress bar.** Rejected in review of the mockup, for the
  same reason.
- **One sentence that names the open choices.** Rejected. It hides the steps as well.
- **Locked steps (D11).** Taken. It is the result part itself, before its files exist.

## Scope

| File | Change |
|---|---|
| `frontend/src/routes/project/ProjectGitOpsPage.tsx`, with its test | One page: the fields, the rule, the result part. No `StepIndicator`, no `StickyActionFooter`, and nothing picked for the owner (D7, D8, D9, D11) |
| `frontend/src/routes/project/gitopsWorkflowYaml.ts`, with its test | Takes the workflow type. State-based adds `--declarative` and the `**` pattern (D4). Its test cases stay out of `gitops-workflow-cases/`, which the Go test of the MCP tool's generator also reads |
| `frontend/src/routes/project/gitops-setup/WorkflowTypeField.tsx` | New: the two cards, the limits alert, and the list of engines state-based supports (D1, D3) |
| `frontend/src/routes/project/gitops-setup/FolderField.tsx` | New: the folder, split out of the workflow files step (D2) |
| `frontend/src/routes/project/gitops-setup/{DeployToField,SignInField}.tsx` | Renamed from the first two step components (D10) |
| `frontend/src/routes/project/gitops-setup/DeployOrder.tsx`, with its test | The one-line summary with the list behind "Deploy order", and the unsupported-engine warning (D3, D10) |
| `frontend/src/routes/project/gitops-setup/FilesSection.tsx` | Renamed from the workflow files step: the locked steps and the numbered steps (D11, D12) |
| `frontend/src/routes/project/gitops-setup/setupField.ts` | New: brings an open choice's field into view and opens its picker (D11) |
| `frontend/src/routes/project/gitops-setup/ExportSchemaField.tsx` | New: the "Export schema from" row (D5) |
| `frontend/src/routes/project/gitops-setup/useSchemaExport.ts` | New: fetches the multi-file export once per database and reads the ZIP with `jszip`, which the frontend already depends on (D6) |
| `frontend/src/routes/project/gitops-setup/SchemaFiles.tsx` | New: the schema file box, with the preview and Download (D6) |
| `frontend/src/routes/project/gitops-setup/schemaFileList.ts`, with its test | New: groups the export's paths by folder and filters them (D6) |
| `frontend/src/components/DatabaseSelectOption.tsx`, `DatabaseSelect.tsx` | The option row of `DatabaseSelect`, moved out so the export picker draws a database the same way (D5) |
| `frontend/src/components/ui/copy-button.tsx`, with its test | An optional text beside the icon, for the Copy button (D12) |
| `frontend/src/components/ui/combobox.tsx` | A `data-slot` on the trigger, so the page can find it (D11) |
| `frontend/src/locales/{en-US,es-ES,ja-JP,vi-VN,zh-CN}.json` | New and changed `gitops` setup strings |
| `docs/design/gitops-setup-state-based.md` | This document |

Tests: the generated state-based files carry `--declarative` and the pattern. The schema file list
groups by folder and filters by path. Nothing is picked on first view except the workflow type. No
file shows while a form choice is open, and an open choice opens its field's picker. The state-based
steps show before an export database is picked, and the export picker leaves out unsupported
engines. A typed folder survives a workflow type switch.

## Not in this change

- Setup over MCP for state-based ([BYT-10581](https://linear.app/bytebase/issue/BYT-10581)). That
  includes the Go copy of the workflow generator, `backend/api/mcp/gitops_workflow.go`, which stays
  migration-based until then.
- The SQL in the review comment ([BYT-10578](https://linear.app/bytebase/issue/BYT-10578)) and in
  the deploy job ([BYT-10580](https://linear.app/bytebase/issue/BYT-10580)), and a review that
  passes when the SQL cannot be generated
  ([BYT-10579](https://linear.app/bytebase/issue/BYT-10579)).
- Supabase-managed schemas in the export
  ([BYT-10571](https://linear.app/bytebase/issue/BYT-10571)).
- A warning for a database in "Deploy to" whose engine differs from the export database's. Its
  deploy fails too.
- Keyboard access to the shared picker. Its trigger cannot take focus, so the page opens it with a
  click (D11).
- The docs' Limitations page, which lists MySQL 5.7 and leaves out CockroachDB. The server rejects
  MySQL older than 8.0 and deploys to CockroachDB.
- One action that downloads every generated file.
- Saving the setup in Bytebase.

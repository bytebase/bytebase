# GitOps setup page

Status: implemented in this PR · 2026-11-12 · [BYT-10423](https://linear.app/bytebase/issue/BYT-10423) · builds on the workload identity dialog, [#21942], and on sign-in inside the action, [#21974]

The project's GitOps page sets up database changes by pull request: a workflow in the team's
repository asks Bytebase to review each pull request and to deploy on merge.
**The page becomes a three-step setup. The owner picks the databases, picks or creates a workload
identity, and gets workflow files in which every value is already filled in. Each step shows the
result of its decision before any file exists.**
Frontend only: no proto, API, store or migration change. The exact text of the generated workflow
files is tracked in [BYT-10424](https://linear.app/bytebase/issue/BYT-10424).

The mock-up (v15, with desktop, large-project and 375px phone frames) and the research pages are
kept outside the repo. The research is summarized in this document with its public sources.

## Problem

The setup owner owns a Bytebase project and can add files to the team's repository. This is often a
DBA or a platform engineer. They want database changes to travel the same way as code changes: a
developer adds a SQL file to a pull request, Bytebase reviews it there, and merging the pull request
deploys it to the databases, one environment after another. Developers do all of this in the git
provider and do not open Bytebase.

To get there, a pipeline in the repository must act in Bytebase on the right databases. Before
setup, the pipeline and Bytebase know nothing about each other. The knowledge needed to connect them
is split between the owner and Bytebase, and nothing tells the owner when the two sides do not
agree. The problem has four parts.

| # | Part of the problem | Why it is hard for the owner | What a mistake causes |
|---|---|---|---|
| 1 | The pipeline does not know what to change. | Which databases belong to the repository is the owner's knowledge. Their names and their environment order are Bytebase's knowledge. The pipeline needs both. | A deploy for an environment that none of the databases is in passes and deploys nothing. A database with no environment is skipped. |
| 2 | Bytebase does not know the pipeline. | The pipeline needs an account in Bytebase that it can use with no stored secret. The rule that says which repository may use the account is written in the git provider's token terms. The account also needs a role in the project. | A rule that is too narrow refuses the pipeline. A rule that is too wide lets other repositories change the databases. An account with no role cannot review or deploy. |
| 3 | The repository has no workflow that calls Bytebase. | The workflow is written in the git provider's language and carries about ten values that must match Bytebase. The owner knows some of them, and Bytebase knows the rest. | One wrong value fails the job, or makes it do nothing. |
| 4 | A mistake is found late, somewhere else, by someone else. | The result of setup shows only when a pipeline runs. That can be days later, in the git provider, in front of a developer who did not do setup. Some mistakes give a passing job. | The team believes a change is deployed when it is not, or a developer is stopped by an error they cannot fix. |

The behavior behind the "mistake" column, verified on `main` at `b203c3f`:

- `action/command/rollout.go`: when the stage named by `--target-stage` is not in the rollout, the
  deploy command logs it and ends without an error.
- `backend/api/v1/rollout_service_task.go`: a target database with no environment gets no deploy
  task.
- `action/command/file.go`: a migration file with no version at the start of its name is skipped
  with a warning. When no file matches the pattern, the command stops with an error.
- `backend/api/v1/release_service_check.go`: the review call stores nothing, so Bytebase holds no
  record that a review ran.

## Principle

> The owner decides only what only the owner knows. Each decision shows its result at once. The
> files are written from the decisions, so they agree with Bytebase with nothing to edit.

Setup is the one place where both sides are known. The owner brings two decisions: which databases,
and which repository may deploy. Bytebase brings everything else.

| Part | Resolved by | The owner decides | Setup does | Result |
|---|---|---|---|---|
| 1 | Step 1, Databases | Which databases this repository's changes belong to | Shows the deploy order, built from the databases' environments. Warns about a database with no environment | The owner sees what a merge will change. The files get the right databases and their environments in order |
| 2 | Step 2, Workload identity | Which repository may deploy, by picking or creating an identity | Shows which repositories the identity trusts and which role it has. Warns when it has no role | No secret is stored. Only the named repositories can sign in, and the account may review and deploy |
| 3 | Step 3, Workflow files | Nothing. The owner adds the files to the repository | Writes the files in full from steps 1 and 2, with every value filled in | The pipeline and Bytebase agree, because the pipeline's values come from Bytebase's own data |
| 4 | Every step | Nothing extra | Checks what Bytebase can see: the environments in step 1, the git provider and the role in step 2, and the Bytebase address before the steps | A mistake that Bytebase can see is caught where the decision is made. What only the repository can show is out of scope (see "Not in this PR") |

## Research

Four questions were researched. Each part below gives the evidence, then what this design takes and
what it leaves out. Nothing is taken because another product does it. A pattern is taken only where
it serves the owner's problem above.

### R1 · How Atlas, Liquibase and Flyway set up a pipeline

Atlas, Liquibase and Flyway each have a first-party path from a migration folder to a CI/CD
pipeline. For each one, the setup path was read in the vendor's docs and, where the code is public,
in its source at a pinned commit, on October 30 and on November 10 and 11, 2026. Flyway Desktop's
wizard was read from Redgate's tutorial and its two screenshots. No pipeline was run end to end. Two
things were not read: the Flyway Pipelines page, which did not load, and the signed-in Atlas Cloud
app, which no public page shows.

**Atlas.** Setup is a terminal command or a docs guide. No public page shows a setup screen in Atlas
Cloud.

- *The command.* `gh atlas init-action` is an extension of the GitHub CLI ([ariga/gh-atlas],
  `main.go` and `prompt.go` at `e861b64`). It asks for the Atlas Cloud token, the Atlas Cloud
  repository, whether a config file is used, the migration directory, the database driver, and
  whether one schema or several are managed. It skips a question when it can find the answer itself.
  Then it stores the token as a repository secret, commits `.github/workflows/ci-atlas.yml` on a new
  branch, and opens a pull request. It can do this because it runs with the user's own GitHub login,
  which needs the `workflow` permission. It supports GitHub only.
- *The guide.* [Setup CI/CD for versioned migrations][atlas-setup] starts with a choice of six CI
  platforms. On the [GitHub Actions page][atlas-github] the user signs in with the CLI, creates a
  bot token, adds two repository secrets (`ATLAS_CLOUD_TOKEN` and `DB_URL`), pushes the migration
  directory, and creates the workflow file from the YAML on the page. There is one YAML per database
  type.
- *Sign-in.* The pipeline holds a long-lived token and connects to the database itself, with the
  address from a secret ([`setup-atlas`][setup-atlas] at `4060c70`).

**Liquibase.** Setup is docs only. [Set up your Liquibase GitHub Action][liquibase-setup] (updated
August 11, 2026) has nine steps, all in GitHub's own screens, with five screenshots of GitHub. The
result is a workflow that installs Liquibase and prints its version. The database address, the
credentials and the commands to run are on other pages. The first-party action only installs the
CLI, and every command is written by hand as a `run:` step ([`setup-liquibase`][setup-liquibase] at
`09a8af1`). For GitLab the docs say to fork an [example repository][liquibase-gitlab]. The pipeline
holds a license key for the paid edition and a database password for each target.

**Flyway.** Flyway Desktop is a desktop app, and it has the only in-product setup that was found:
the "Automated Deployment" page of the Enterprise edition ([tutorial][flyway-tutorial], updated
August 18, 2026).

- A dialog first asks for the CI/CD platform, once per project: GitHub Actions, Bash or PowerShell.
  Only GitHub Actions gets a workflow file. A team on another platform picks a script.
- Four steps follow, named in a row across the top: Workflow, Environments, Review and Checklist.
  Back is at the top left and Continue at the top right.
- In Environments the user builds the deploy order: a row of cards that starts at the repository,
  one card per deployment step, each edited in a side panel, with an optional manual review.
- Review shows the whole file, which the user can edit and copy.
- Checklist lists what is left to do in GitHub: create a personal access token, configure GitHub
  Environments when a step needs a manual review, and add the secrets `FLYWAY_EMAIL`,
  `FLYWAY_TOKEN`, and a user and a password for each environment.
- Finish saves `deployment.yml` into the local repository folder and reminds the user to commit, to
  push and to add the secrets. The generated workflow starts on a manual run.
- The pipeline signs in with an email and a personal access token, and connects to each database
  with its own user and password ([`setup-flyway`][setup-flyway] at `ebb315e`,
  [`flyway-github-actions`][flyway-actions] at `1da52b6`).

The three side by side, with this design:

| | Atlas | Liquibase | Flyway | This design |
|---|---|---|---|---|
| Where setup happens | A terminal command, or a docs guide | A docs page | A wizard in a desktop app, Enterprise edition | The project's GitOps page |
| What the user goes through | About six prompts | Nine steps in GitHub's screens | One dialog and four steps | Three steps |
| CI platform | The command: GitHub only. The guide: asked first, six platforms | One docs page per platform | Asked first. GitHub Actions gets a workflow file, the rest get a script | Not asked. Read from the workload identity. GitHub Actions and GitLab CI |
| Deploy targets and their order | The guide's workflow takes one database address, from a secret | Written by hand | Built by the user, one card per deployment step | Read from the environments of the picked databases |
| Who writes the workflow file | The command, in a pull request. In the guide, the user | The user | The app, into the local folder. The user commits it | The page generates it. The owner adds it to the repository |
| Secrets the user stores | 2 in the guide. The command stores the token itself | A license key, and a password per target | 2, and 2 more per environment | None |
| How the pipeline signs in | A long-lived token | A license key. No hosted service is in the deploy path | An email and a personal access token | The job's own OIDC token |
| Explains the git provider's screens | Partly. One sentence on where the secrets go | Yes. Most of the page | Partly. One checklist line per item | No |
| Confirms that setup worked | No | Partly. The user reads the workflow log | No | No (see "Not in this PR") |

What the research shows:

1. **A generated file asks less of the user than an assembled one.** The two paths that generate the
   file, Atlas's command and Flyway's wizard, ask a handful of questions. The docs-only path takes
   nine steps and still leaves the database address, the credentials and the commands to other
   pages. This design generates the files (D5).
2. **Each tool asks for what it cannot know.** Atlas asks for a token, a directory and a driver.
   Flyway asks for the platform and the environments, because a desktop app has no record of either.
   Bytebase already holds the databases, their environments and the identity, so setup asks two
   questions and reads the rest (D6).
3. **Stored secrets are what make the other setups long.** Each of the three keeps long-lived
   secrets in the CI platform's secret store: 2 in Atlas's guide, and 8 in Flyway for a test, a
   production and a build database. Flyway's Checklist step and half of the steps in Atlas's guide
   exist to put them there. A workload identity stores none, so setup has no such step (D11).
4. **The deploy order is the user's work in the other tools.** Flyway has the user build it card by
   card. Atlas's guide and Liquibase leave it to the YAML. Here the environments of the picked
   databases already define it, so step 1 shows the order and asks nothing (D3).
5. **Only a tool that holds the user's git login can put the file in the repository.** Atlas's
   command opens the pull request because it runs as the user in a terminal. Flyway writes to a
   local folder because it is a desktop app. Bytebase is a server with no access to the repository,
   so the owner adds the files (see "Alternatives considered").
6. **None of the three uses the pipeline's own OIDC token to sign in to its service, and none
   confirms from inside the product that setup worked.** For both, there is no pattern in these
   tools to weigh. Sign-in follows the workload identity dialog of [#21942]. A check that setup
   worked is left out (see "Not in this PR").

- **Taken:** the files are generated from the user's answers (Atlas's command, Flyway). Every step
  is named in one row, and each step asks one thing (Flyway). Each file has a copy button (Flyway).
  The last step ends on what is still to do, in one sentence (Flyway's reminder after Finish).
- **Left out:** asking for the platform (Flyway, Atlas's guide), because the workload identity
  already records it. Building the deploy order by hand (Flyway), because the environments define
  it. A Review step and a Checklist step (Flyway), because each step here already shows the result
  of its decision and there is no secret to add. Editing the file inside the product (Flyway),
  because the owner edits it in the repository. Opening the pull request for the user (Atlas),
  because Bytebase has no access to the repository. Screenshots of the git provider's screens
  (Liquibase), because the owner knows their git provider.

### R2 · What makes an in-product setup guide work

Sources: NN/g on wizards, GOV.UK service patterns, PatternFly wizard, Shopify setup guide, Carbon
progress indicator, and the setup flows of Sentry
([getsentry/sentry](https://github.com/getsentry/sentry) at `9e85b69`), Codecov
([codecov/gazebo](https://github.com/codecov/gazebo) at `8633d95`), PostHog
([PostHog/posthog](https://github.com/PostHog/posthog) at `61817ef`), and the Claude Platform
"Connect workload" dialog.

| # | Rule found | In this design |
|---|---|---|
| 1 | Few steps, one question each | Yes. Three steps, two questions and one task (D2) |
| 2 | Show all steps, and name each by its result | Yes. The step row shows the three titles (D7) |
| 3 | Ask only what the product cannot know, once | Yes. Two questions. Every other value is read (D6) |
| 4 | Help stays in the step | Yes. One line under each question. No links to setup docs |
| 5 | A blocked step is a task to do, not an error | Yes. Preconditions and failed checks show as tasks with a link (D4, D9) |
| 6 | A generated file carries the user's values | Yes (D5, D6) |
| 7 | Guide the step outside the product with exact handholds | Partly. Paths, copy buttons, the repository link and "what happens next" are given. Ordinary git actions are not explained, because the owner knows their git provider |
| 8 | End on what happens next, with no congratulation page | Yes (D1, D5) |
| 9 | Let the user leave and come back | Not used. Setup saves nothing, and going through it again takes a few clicks (D1) |

### R3 · Where the step list goes

Three layouts were compared: a rail on the left, a row on top, and all steps stacked on one page.

| Evidence | Finding |
|---|---|
| Bytebase pages, `main` at `b203c3f` | Every page-level step flow uses the horizontal `StepIndicator` on top: `routes/project/ProjectSyncSchemaPage.tsx`, `components/sql-review/ReviewCreation.tsx`, `routes/auth/workspace-setup/SetupScreenFrame.tsx`. A vertical rail exists only in the workload identity dialog of [#21942] |
| `docs/agents/frontend-ux.md` | A long multi-step workflow that deserves a route is a full-page wizard with the shared step indicator |
| Design systems | Side: Carbon, PatternFly, Cloudscape. Top: Material, SAP Fiori, Atlassian, Ant Design, USWDS. No source gives a rule that fits every case |
| Products, by kind of setup | Choices that shape a result are shown one step at a time (Flyway Desktop, Claude Platform). Instructions that are carried out elsewhere are stacked (Sentry, Codecov, Liquibase docs). GitOps setup is the first kind |
| Width, measured on the mock-up | The project navigation is 208px (`components/DashboardBodyShell.tsx`). A 232px rail beside it leaves 760px for the workflow files. In a 1024px window, 5 and 7 lines of the two GitHub files do not fit. With the row on top the content is 992px wide, and 1 and 1 lines do not fit |

- **Taken:** the row on top (D7). It is what Bytebase pages already use, and it gives the workflow
  files the most width.
- **Left out:** the rail's one-line description per step. The row has no room for it, so each step
  states its question and one line of help in its own body.

### R4 · Where Back and Continue go when the steps are on top

| Where the buttons are | Who does it | In view on a tall step |
|---|---|---|
| In the header, beside the step row | Flyway Desktop (Back top left, Continue top right) | Only if the header stays on screen |
| In a footer bar at the bottom | [SAP Fiori wizard](https://experience.sap.com/fiori-design-web/wizard/) in tab-bar mode, the Azure portal create flow, Bytebase `ReviewCreation.tsx` | Bytebase: yes, verified in source. The others do not state it |
| At the end of the step content | Demos of [MUI](https://mui.com/material-ui/react-stepper/), [Mantine](https://mantine.dev/core/stepper/), [Chakra UI](https://chakra-ui.com/docs/components/steps), [Ark UI](https://ark-ui.com/docs/components/steps), [PrimeReact](https://primereact.org/stepper/), SAP Fiori in anchor-bar mode, PostHog onboarding | No |

[USWDS](https://designsystem.digital.gov/components/step-indicator/) and
[Carbon](https://carbondesignsystem.com/components/progress-indicator/usage/) say the step indicator
is not navigation and that back and next must be provided separately, but not where. shadcn/ui, Base
UI, Radix and HeroUI have no stepper component.

- **Taken:** the buttons sit under the step content, where every library puts them, in a footer that
  stays at the bottom of the window (D8). Bytebase already has this part:
  `components/ui/sticky-action-footer.tsx`, used by 11 pages and forms.
- **Left out:** buttons in the header. See "Alternatives considered".

## Decisions

**D1 · The GitOps page is the setup.** There is no start screen, no Start button, no finish screen
and no Done button. The page opens with the outcome in one sentence: "Every pull request that
changes the migration folder gets a Bytebase review. Merging it deploys." A second line states which
CI/CD platforms setup writes files for (D11). Under it are the three steps, with step 1 open.
Bytebase saves nothing: the files in the repository are the setup, and a saved copy would go stale
as soon as someone edits them. Opening the page again starts at step 1. A lone database and a lone
identity are already picked, so going through the steps again takes a few clicks.

**D2 · Three steps: Databases, Workload identity, Workflow files.** Two decisions that only the
owner can make, and one hand-over. Databases comes first because it is the goal the owner came with,
and the identity is the means. Workflow files comes last because the files are built from both. Each
title names what the step settles. Steps 1 and 2 open with a question, and step 3 opens with a task.

**D3 · Step 1, Databases: "Which databases should GitOps deploy to?"** Resolves part 1.

- *The owner decides:* one database group (the first tab, marked Recommended), or one or more
  databases. A group is recommended because a database that joins the group later is deployed too,
  with no change to the workflow files. A list of databases is written into the files, so adding one
  means editing them. Databases are picked with `DatabaseSelect`, which searches on the server and
  loads more results on demand, so a project with hundreds of databases stays usable. If the project
  has no database group, the Databases tab opens, and a lone database is already picked.
- *Setup shows:* the deploy order, as one bordered list with one section per environment, in the
  project's environment order. A vertical line with one dot per environment joins the sections.
  There are no order numbers, because the setup steps on the same page are already numbered. A
  section has the environment name and the first three databases, each as a chip written as instance
  / database (`DatabaseTargetDisplay`), because several instances often hold a database with the
  same name. A section with more than three databases ends the chips with an inline "+81 more"
  action, placed and styled as on the plan page, that opens the list of the other databases, with
  search when the list is long. The list has a total when the pick has more than three databases. On
  the Database group tab, a View group link opens the group. Hiding provider-reserved databases,
  such as Neon's `postgres`, is tracked in [BYT-10254](https://linear.app/bytebase/issue/BYT-10254)
  and is not part of this change.
- *Setup checks:* databases with no environment. A group that matches some shows a warning under the
  deploy order. The warning says how many will not be deployed, names the first three with the same
  "+N more" action, and says to assign an environment to include them. On the Databases tab they are
  listed but cannot be picked, and each one says "No environment". If no database in the project has
  an environment, the tab shows a warning instead.
- *Goes into the files:* the group, or the list of databases, and their environments. Nothing is
  created in Bytebase.
- *Not asked:* the environment. It comes from each database.
- Continue is disabled until at least one database or a group is picked.

**D4 · Step 2, Workload identity: "Which workload identity does your workflow sign in as?"**
Resolves part 2.

- *The owner decides:* one workload identity of this project. If the project has only one, it is
  already picked. If it has none, the step shows only a Create workload identity button, which opens
  the dialog of [#21942]. Opened from setup, the dialog offers GitHub Actions and GitLab CI only,
  closes after Create, and the new identity comes back picked (D10).
- *Setup shows:* what it reads from the picked identity, in a read-only box: Git provider,
  Repositories that can sign in (one repository, or every repository of an owner), and Role in this
  project. The git provider's logo shows before its name, so the owner sees which platform the
  identity belongs to before reading the row. It is the logo on the Create dialog's cards.
- *Setup checks,* shown only when one fails, as a task with a link:
  - The git provider is GitHub Actions or GitLab CI. An identity for another OIDC provider is listed
    but cannot be picked, because setup has no files for it. The Create dialog cannot return one,
    because opened from setup it does not offer Other OIDC (D10).
  - The identity has a role that allows GitOps in this project, such as GitOps Service Agent. An
    identity with no role cannot review or deploy, so Continue is disabled and a warning links to
    the Members page. A role granted to `allUsers` counts, as in the server's permission check
    (`backend/component/iam/manager.go`): always in the project, and in the workspace except on
    Bytebase Cloud.
- *Goes into the files:* the identity's email and its git provider.
- *Not asked:* git provider, owner, repository, owner ID, email, role, issuer and audience. The
  identity already holds them.

**D5 · Step 3, Workflow files: "Add the files to your repository".** Resolves part 3. The step asks
no question. It does not explain how to commit a file or open a pull request. It shows, in this
order:

1. The migration folder, `migrations/`, read-only with an Edit action. Changing it updates the
   files.
2. The repository, as a link, when the identity names one. A workload identity does not require a
   repository: it can trust every repository of an owner. In that case this line is left out. The
   git provider's logo shows before the link, so the owner sees where the link leads.
3. The generated files for the identity's git provider, each with its path in a header bar and a
   copy button, one under the other, with no tabs. The header bar holds the path and the copy
   button, and nothing else. A long file starts short (see below). GitHub Actions:
   `.github/workflows/bytebase-review.yml` reviews pull requests that change migration files, and
   `.github/workflows/bytebase-deploy.yml` deploys on each merge. GitLab CI: one
   `bytebase.gitlab-ci.yml` with both jobs, plus the one `include` line to add to `.gitlab-ci.yml`.
4. What happens next: "Once the files are on the default branch, the next pull request that changes
   the migration folder gets a Bytebase review comment. Merging it deploys." GitLab says "merge
   request".
5. For developers: "Add a file such as `migrations/0001_create_orders.sql` in a pull request. File
   names start with a version number. The workflow skips a file without one."

Each job signs in with the identity's email and the job's own token. There is no sign-in step, no
secret and no build job. The only placeholder is the Bytebase URL while the External URL is not set
(D9). The owner commits the files. Bytebase does not.

Sign-in happens inside `bytebase-action` ([#21974], `docs/design/bytebase-action-sign-in.md`). A job
names the identity in `BYTEBASE_WORKLOAD_IDENTITY`, and the action exchanges the job's ID token
itself. So the files hold no token exchange script and pass no `--access-token`:

- *GitHub Actions:* the job has the `id-token: write` permission. The action requests the token.
- *GitLab CI:* the job has an `id_tokens` entry named `BYTEBASE_ID_TOKEN`, with the audience
  `bytebase`.

Without the script a GitHub review file is 28 lines, and each line is a setting the owner can check.
A refused sign-in fails the job with one line that carries the server's reason.

The files need an action image that has [#21974]. A self-hosted server pins the image to its own
version, so the files and the image always match. Bytebase Cloud uses the `:cloud` image, so this
change ships after [#21974] has reached Cloud.

**A long file starts short.** The owner's task in step 3 is to copy each file into the repository.
Shown whole, the two GitHub files for three environments are 113 lines, and the owner scrolls past
them to find the second file and "What happens next". So a file with more than 12 lines shows its
first 12 lines, with a fade over the last one. Under them, inside the file's box, a "Show all 28
lines" action opens the whole file, and "Show less" closes it again. The fade and the small link
action are the ones the plan page uses for a long description.

- The copy button always takes the whole file, open or not.
- A file with 12 lines or fewer shows whole and has no action. The GitLab `include` lines are such a
  file.
- Each file opens on its own. Leaving the step closes them again.
- The rule is the same while the External URL is not set (D9). The placeholder can then be below the
  cut, and the line above the files names it.

**D6 · Every value in the files has one source, and the owner is asked for two.**

| Value | Used for | Source |
|---|---|---|
| Bytebase URL and project | Every call to Bytebase | Known by the page. On self-hosted Bytebase the URL is the workspace External URL (D9) |
| Action version | The action must fit the server's version | Known by the page (`bytebaseActionImage` in `gitopsWorkflowYaml.ts`) |
| Databases to deploy to | What to review and deploy | **Asked in step 1** |
| Deploy environments | Which stages the deploy runs, in order | The databases' environments, in the project's environment order |
| Workload identity | The account the workflow signs in as | **Picked or created in step 2** |
| Email, audience, git provider | Signing in, file format and triggers | Read from the workload identity |
| Token settings | Letting the job get its own token | Written by the page for the identity's git provider: the `id-token: write` permission on GitHub Actions, an `id_tokens` entry named `BYTEBASE_ID_TOKEN` on GitLab CI |
| Repository | Where the owner commits the files | Read from the workload identity when it names one. The files do not contain it |
| Deploy branch | When the deploy job runs | The repository's default branch, read by the workflow at run time |
| Migration folder | Which files to review and deploy | Prefilled `migrations/`, editable in step 3 |

**D7 · The step list is a row on top.** It is the shared horizontal `StepIndicator`: a 24px circle
and a title per step, joined by lines. A finished step shows a check. A reached step can be clicked
to go back to it (`onStepClick`, added by [#21942]). On a phone the row shows only the open step's
title. The row scrolls with the page. Step headings, fields, buttons, code blocks and warnings use
the same parts as the workload identity dialog (`WorkloadIdentityFlow/StepParts.tsx`), so the two
step flows look and behave the same. Setup has no close button, no Cancel button and no fixed
height.

**D8 · Back and Continue sit in a footer that stays at the bottom of the window.** It is the shared
`StickyActionFooter`: Back on the left from step 2, Continue on the right, a line on top and the
page background, so the step content scrolls under it. A step can be taller than the window, for
example with a long deploy order, a warning, or the workflow files. The owner then still sees
Continue without scrolling to the end. Step 3 has only Back. The footer is the same on a phone.

**D9 · Preconditions show only when they are not met.** Setup needs three things: the project has at
least one database, the project has a workload identity or the user may create one, and on
self-hosted Bytebase the workspace External URL is set. A precondition that is not met shows as a
task at the top of the page: what is missing, who can fix it, and a link.

Without a database, or without an identity and the permission to create one, the steps cannot be
used. An unset External URL does not block the steps. Only a workspace admin can set it, the owner
is often not one, and steps 1 and 2 do not need it. The task stays on top of every step. Step 3
shows the files with a marked placeholder in place of the URL. Its copy buttons are disabled, with
the reason beside them, so a file that cannot reach Bytebase is not committed by accident.

**D10 · Opened from setup, the workload identity dialog offers only what setup can use.** Two things
differ, and both are opt-in. Opened anywhere else, the dialog is unchanged: it offers all three
token sources and keeps its last step.

- *It offers GitHub Actions and GitLab CI, not Other OIDC.* Setup has workflow files for those two
  only (D11). An identity for another OIDC provider could not be used in step 3, so the owner would
  create it and then be told to create another.
- *It closes after Create.* Its last step shows the email and a test workflow file. Setup already
  shows the identity, and step 3 gives the real files, so a test file shown first would give the
  owner two different files to add.

**D11 · Workload identity only, with GitHub Actions and GitLab CI only, and the page says so.**
Setup offers no service account and no other platform. The git provider is not asked: the owner's
repository is already on one, and the workload identity records it (D4). An owner on another
platform must learn this before picking anything. So a line under the outcome sentence, on every
step, says: "Setup writes workflow files for GitHub Actions and GitLab CI. For other CI/CD
platforms, see the GitOps docs." The line names the two platforms setup supports and no others, and
links to the GitOps docs. Service accounts stay available elsewhere in Bytebase, and hand-written
workflows keep working.

**D12 · The deploy branch is not asked.** The deploy job runs on the repository's default branch,
which GitHub and GitLab both expose to the workflow. A team that deploys from another branch changes
one line.

## States

| Project | Step 1 opens with |
|---|---|
| Has a database group | Database group tab, the group picked, deploy order shown |
| No database group, one database | Databases tab, the database picked |
| No database group, several databases | Databases tab, nothing picked, Continue disabled |
| Some databases have no environment | A warning under the deploy order. On the Databases tab they cannot be picked |
| No database has an environment | A warning in place of the picker. Continue disabled |

| Project's workload identities | Step 2 shows |
|---|---|
| None | Only the Create workload identity button |
| One, for GitHub Actions or GitLab CI | That identity picked, with the read-only box |
| Several | A picker, nothing picked, and a Create workload identity button beside it |
| One for another OIDC provider | Listed, cannot be picked, with the reason |
| Picked identity has no role | The read-only box, a warning with a link to Members, Continue disabled |

| Picked identity | Step 3 shows |
|---|---|
| GitHub Actions, names one repository | The repository link and two files |
| GitHub Actions, trusts every repository of an owner | Two files, no repository link |
| GitLab CI | One file and the `include` line, and "merge request" in the text |

| File in step 3 | Shows |
|---|---|
| More than 12 lines | Its first 12 lines with a fade, and "Show all N lines" under them. Opened: every line, and "Show less" |
| 12 lines or fewer | Every line, no action |
| Any length | The copy button takes the whole file |

| Workspace | Page shows |
|---|---|
| External URL set, or Bytebase Cloud | The three steps |
| Self-hosted, External URL not set | A task on top of every step with "Configure now". The steps can be used. Step 3 shows the files with a placeholder for the URL, and its copy buttons are disabled |

| Window | Layout |
|---|---|
| Desktop | Content up to 1040px wide, step row with three titles, footer at the bottom of the window |
| Phone, 375px | One column, step row with the open step's title only, the same footer |

## Alternatives considered

**The setup approach**

These come from the other tools' setups (R1).

- **A docs guide with YAML to copy, and no setup page (Liquibase, Atlas's guide).** Rejected. A page
  of docs cannot hold the owner's values. The owner would type the project, the targets, the
  identity and the Bytebase address into the YAML by hand, which is part 3 of the problem.
- **A command in the terminal that asks the questions and opens the pull request (Atlas's `gh atlas
  init-action`).** Rejected for this change. It needs a client on the owner's machine that holds
  their git login, and one client per git provider. The two decisions are about things that live in
  Bytebase, the databases and the identity, so the owner would still come to Bytebase to make them.
  Setup from outside the page is a separate question
  ([BYT-10425](https://linear.app/bytebase/issue/BYT-10425)).
- **Bytebase adds the files to the repository itself.** Rejected. It needs write access to the
  repository, so Bytebase would have to store a git provider credential or have an app installed
  there. A workload identity lets the repository sign in to Bytebase with no stored secret, and this
  would add a stored secret in the other direction.
- **Building the deploy order in setup, one card per environment (Flyway).** Rejected. The
  environments and their order already exist in Bytebase, and the rollout follows them. A second
  order made in setup could disagree with the one the rollout uses.
- **A Review step and a Checklist step after the files (Flyway).** Rejected. Flyway needs them
  because its file can be edited in the wizard and its workflow needs secrets and GitHub
  Environments. Here each step already shows its result, and nothing is left to set up in the git
  provider except adding the files.
- **Generating a script for other CI platforms (Flyway's Bash and PowerShell).** Rejected for this
  change (D11). Each platform issues its job token in its own way, and the workload identity dialog
  has presets for GitHub Actions and GitLab CI only. The scope line sends an owner on another
  platform to the docs.

**The step set**

- **Sign-in first (Workload identity, then Databases).** Rejected. The databases are what the owner
  came for. Neither decision depends on the other, so the order follows the owner's goal.
- **Two steps (Databases and Workload identity on one screen, then the files).** Rejected. Each
  decision has its own result to show and its own check, and two results on one screen bury both.
- **A fourth step, "First migration", with a sample file.** Rejected. Owners will not run a sample
  change against real databases, and Bytebase cannot see whether they did. After the hand-over the
  owner is in the git provider, so a step on the setup page would speak in the wrong place.
- **A repository step.** Rejected. The workload identity already says which repositories may sign
  in, and the files do not contain the repository.
- **Offering a service account as well.** Rejected for setup (D11). It needs a stored secret, which
  is part 2 of the problem.

**Knowing that setup worked**

- **Run the review job on the pull request that adds the workflow files.** The review file would
  also trigger on a change to itself, and a passing job would show that sign-in works. Rejected for
  now: the review command stops with an error when the migration folder has no file
  (`action/command/file.go`), so a repository that is new to GitOps would get a failed job although
  sign-in worked. It needs a change in the action first (BYT-10424).
- **A live check on the setup page.** Not in this PR. Since [#21971] the audit log records each
  sign-in with its repository, so a later change could read it. That change needs its own design
  (see "Not in this PR").

**The layout**

- **A rail on the left, as in the workload identity dialog.** Rejected (R3). On a page it adds a
  second left column beside the project navigation and takes 232px from the workflow files.
- **All steps stacked on one page.** Rejected (R3). Stacked steps fit instructions that are carried
  out elsewhere. Here step 3 cannot be shown until steps 1 and 2 are answered, because the identity
  decides the file format and the databases decide the deploy stages.
- **Back and Continue in the header, beside the step row.** Rejected (R4). The owner finishes
  reading a step at the bottom, where a header button is furthest away. A 375px row has no room for
  the step and two buttons. Only one product was found that does this.
- **Back and Continue at the end of the step content, not fixed.** Rejected. It is the libraries'
  default and works for short steps, but a long deploy order or the workflow files push Continue
  below the window.
- **Keeping the step row fixed at the top as well.** Rejected. A fixed row and a fixed footer
  together take too much of a phone screen.

**Smaller choices**

- **Saving the setup in Bytebase.** Rejected (D1).
- **Asking for the deploy branch.** Rejected (D12).
- **Showing every file whole.** Rejected (D5). The owner comes to step 3 to copy the files, and the
  copy button does not need the lines on screen. Whole files push the second file and "What happens
  next" far down, most of all on a phone.
- **A box of a fixed height that scrolls, for each file.** Rejected. A box that scrolls inside a
  page that scrolls takes the wheel or the finger when the owner means to move the page.
- **Showing only each file's header bar until it is opened.** Rejected. The first lines say what the
  file does: its name, its trigger and its paths.
- **A label after each path ("new file", "add these lines").** Rejected. Every generated file is
  new, so "new file" tells the owner nothing. The path `.gitlab-ci.yml` and the two `include` lines
  already say where those lines go, and adding lines to that file is ordinary GitLab work.
- **Keeping the token exchange script in the files.** Rejected (D5). The script is 18 lines in every
  GitHub job and 10 in every GitLab job, so it is most of what the owner reads in step 3. It also
  prints the server's raw answer when sign-in is refused, and it passes the token from one step to
  the next.
- **Asking which runner the jobs use.** Rejected. The files use the git provider's hosted runner. A
  team with its own runners needs its own labels or runner group, which only the team knows, and it
  changes one line.
- **A git provider step before Databases.** Rejected (D11). The workload identity already records
  the provider, so the step would ask the same fact twice and add a case where the two disagree. The
  scope line tells the owner whether setup can serve their repository without a step.
- **Offering Other OIDC in the Create dialog and warning in step 2.** Rejected (D10). The owner
  would finish the dialog and only then learn that setup cannot use the identity.
- **Blocking every step until the External URL is set.** Rejected (D9). The owner often cannot set
  it, and would see nothing of the setup while waiting for an admin.
- **Deploy order as counts only, as plain names with a "Show all" link, with a "View all" action in
  a header strip, or with order numbers.** Rejected in favor of the plan page's own pattern: chips
  and an inline "+N more" (D3).

## Scope

Files in this PR.

| File | Change |
|---|---|
| `frontend/src/routes/project/ProjectGitOpsPage.tsx`, with a new test | Becomes the three-step setup: outcome sentence, preconditions, step row, step body, footer (D1, D2, D7, D8, D9). |
| `frontend/src/routes/project/gitops-setup/{DatabasesStep,WorkloadIdentityStep,WorkflowFilesStep}.tsx` | New: one component per step (D3, D4, D5). |
| `frontend/src/routes/project/gitops-setup/ProviderLogo.tsx` | New: the GitHub or GitLab logo at 16px, from the files in `frontend/src/assets/git/` that the Create dialog and the release page already use (D4, D5). |
| `frontend/src/routes/project/gitops-setup/DeployOrder.tsx` and its test | New: the deploy order list, the "+N more" list and the no-environment warning (D3). |
| `frontend/src/routes/project/gitopsWorkflowYaml.ts` and its test | Takes the picked targets, their environments, the identity and the folder, and returns the files for the identity's git provider (D5, D6). The files no longer carry the token exchange script, which is the generator change of [#21974]. The exact YAML is BYT-10424. |
| `frontend/src/routes/project/gitops-setup/useDeployTargets.ts` and its test | New: loads the picked group or databases and puts them in deploy order by environment, with the databases that have no environment apart (D3). |
| `frontend/src/routes/project/gitops-setup/identity.ts` | New: what setup reads from a workload identity: its git provider, its repositories, the repository address and its roles in the project (D4). |
| `frontend/src/routes/project/WorkloadIdentitySelect.tsx` and its test | Two opt-in props: the caller sets each identity's description and can refuse an identity with a reason. Setup uses them to disable identities for another OIDC provider (D4). |
| `frontend/src/components/WorkloadIdentityFlow/{WorkloadIdentityDialog,TokenSourceStep}.tsx` and the dialog's test | Two opt-in props: the token sources to offer, and close after Create and return the new identity (D10). |
| `frontend/src/components/ui/step-indicator.tsx` | Reused as it is after [#21942]. |
| `frontend/src/components/ui/sticky-action-footer.tsx` | Reused as it is. |
| `frontend/src/components/DatabaseSelect.tsx` and its test | One opt-in prop: a database with no environment is listed as disabled and marked "No environment" (D3). |
| `frontend/src/components/DatabaseTargetDisplay.tsx` | Reused as it is. |
| `frontend/src/assets/gitops-workflow.svg` | Removed. Nothing shows it any more. |
| `frontend/src/locales/{en-US,es-ES,ja-JP,vi-VN,zh-CN}.json` | New `gitops` setup strings in all five locales. Strings that are no longer used are removed. |
| `docs/design/gitops-setup-page.md` | This document. |

Tests:

- `gitopsWorkflowYaml.test.ts`: the files parse as YAML and carry the URL, project, targets, email,
  audience, action version and folder. A group gives one target. Environments appear in the
  project's order. GitHub gives two files and GitLab one. No secret is referenced. No job has a
  sign-in step or passes `--access-token`. Every GitHub job has the `id-token: write` permission,
  and every GitLab job names its ID token `BYTEBASE_ID_TOKEN`.
- `DeployOrder.test.tsx`: sections follow the environment order. Three chips, then "+N more". The
  total shows above three databases. The warning counts databases with no environment.
- `useDeployTargets.test.ts`: stages follow the workspace's environment order, and an environment
  missing from that order comes first, as in the rollout. Databases with no environment are left out
  of the stages. A group expands to the databases it matches.
- `DatabaseSelect.test.tsx`: a database with no environment can be picked by default, and is
  disabled and marked when the caller requires an environment.
- `WorkloadIdentitySelect.test.tsx`: the caller's description shows on each identity, and an
  identity the caller refuses is disabled.
- `WorkloadIdentityDialog.test.tsx`: limited to GitHub Actions and GitLab CI, the dialog shows two
  token sources and no Other OIDC. With no limit it shows all three. Editing an identity always
  shows all three.
- `ProjectGitOpsPage.test.tsx`:
  - A project with one database and no identity opens on the Databases tab with the database picked.
  - Continue is disabled with nothing picked, and with an identity that has no role.
  - A role granted to `allUsers` counts for the identity: in the project always, in the workspace
    except on Bytebase Cloud.
  - The scope line shows on every step.
  - An identity for another OIDC provider cannot be picked.
  - The Create dialog opens with GitHub Actions and GitLab CI only.
  - Creating an identity from step 2 returns to step 2 with it picked.
  - The repository link shows only when the identity names a repository.
  - The git provider's logo shows before the provider name in step 2 and before the repository link
    in step 3, for GitHub Actions and for GitLab CI.
  - Editing the folder changes the paths in the files and in the developers' line.
  - A file with more than 12 lines shows 12 until "Show all", and the copy button holds the whole
    file before and after. The GitLab `include` lines have no action.
  - An unset External URL shows the task on every step, a placeholder in the files and disabled copy
    buttons.
  - A file's header bar names its path and nothing else.
  - Every picker opens its menu above the sticky footer.
  - The footer holds Continue on steps 1 and 2 and only Back on step 3.
- Manual: walk the three steps on desktop and at 375px wide, in a project with one database and in
  one with more than a hundred. In a window shorter than the step, check that Continue stays in
  view. Add the generated files to a GitHub repository and to a GitLab project, open a pull request
  with a migration file, and merge it.

## Not in this PR

- The exact workflow YAML and the action's behavior
  ([BYT-10424](https://linear.app/bytebase/issue/BYT-10424)), including a deploy job that passes
  without deploying ([BYT-10426](https://linear.app/bytebase/issue/BYT-10426)) and a review that
  ends without an error when there are no files.
- Checking the repository side: whether the files were added, whether a job may request a token, and
  whether the runner can reach Bytebase. The first pull request with a migration file is the check.
- A live sign-in status for the identity on the setup page. Since [#21971], audit rows name the
  repository behind each sign-in ([BYT-10494](https://linear.app/bytebase/issue/BYT-10494)). Setup
  does not read them.
- The one-off test workflow in the workload identity dialog
  (`WorkloadIdentityFlow/testWorkflow.ts`). It runs without the action image, so it keeps its own
  exchange, and its GitLab job still names the token `GITLAB_OIDC_TOKEN`. Setup does not show it
  (D10).
- Setup by coding agents ([BYT-10425](https://linear.app/bytebase/issue/BYT-10425)).
- Service accounts, and CI systems other than GitHub Actions and GitLab CI.
- Review rules, approvals and environments. They have their own places in Bytebase.
- The workload identity and service account list pages
  ([BYT-10511](https://linear.app/bytebase/issue/BYT-10511)).

[ariga/gh-atlas]: https://github.com/ariga/gh-atlas
[atlas-setup]: https://atlasgo.io/versioned/setup-cicd
[atlas-github]: https://atlasgo.io/guides/ci-platforms/github-versioned
[setup-atlas]: https://github.com/ariga/setup-atlas/blob/4060c70f266b5c63d372dc0c707ea54303139494/action.yml
[liquibase-setup]: https://docs.liquibase.com/secure/user-guide-5-0/set-up-your-liquibase-github-action
[setup-liquibase]: https://github.com/liquibase/setup-liquibase/blob/09a8af19c0c87ccbbe9a154dce0e716b3add0dcd/README.md
[liquibase-gitlab]: https://contribute.liquibase.com/extensions-integrations/directory/integration-docs/gitlab-ci-cd/
[flyway-tutorial]: https://documentation.red-gate.com/fd/tutorial-generate-a-github-actions-deployment-workflow-with-flyway-desktop-342852947.html
[setup-flyway]: https://github.com/red-gate/setup-flyway/blob/ebb315ee386e853d5e6e58d2529f1665b61b31a5/action.yml
[flyway-actions]: https://github.com/red-gate/flyway-github-actions/tree/1da52b65e2864a99fc48ab02ce0e6b5f28749636
[#21942]: https://github.com/bytebase/bytebase/pull/21942
[#21971]: https://github.com/bytebase/bytebase/pull/21971
[#21974]: https://github.com/bytebase/bytebase/pull/21974

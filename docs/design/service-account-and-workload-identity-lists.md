# Service account and workload identity lists and create flows

**Status:** Proposed design for review. No implementation changes.

This doc covers the two list pages and the surfaces that create and edit an account. For workload
identities it replaces the surface decision (D1) and the phone layout (D8) in
[workload-identity-creation-flow.md](workload-identity-creation-flow.md). The steps, fields and
wording in that doc stand.

## Problem

**The lists say too little.** Both show a name and an email that repeats the name. To learn what
an account can do or which CI jobs a workload identity trusts, a reviewer opens every row and then
the Members page.

Each list should answer its reader's questions without opening a row:

| List | Questions |
|---|---|
| Service accounts | What can it do? |
| Workload identities | Which repositories can sign in? What can it do? |

"Is it still used?" and "Has it ever worked?" are left to [Future work](#future-work).

**The two pages create differently.** Create on Service accounts slides a sheet in from the right.
Create on Workload identities opens a centered dialog. The pages are siblings, the button sits in
the same place, and Edit follows each.

## Changes from what ships

Shipped behavior is in `frontend/src/routes/workspace/ServiceAccountsPage.tsx`,
`WorkloadIdentitiesPage.tsx` and `frontend/src/components/WorkloadIdentityFlow/`. Both pages are
used at workspace and project level.

| Area | Shipped | Proposed |
|---|---|---|
| Page intro | None | One-sentence description in the page info note, as Issues, Plans and Members have. No page title |
| Toolbar | Create | Search, state filter, Create |
| Columns | Account, Operations | Service accounts: Name, Roles, Created. Workload identities: Name, Trusts, Roles, Created |
| Avatar | Colored letter circle | None. Workload identities show a provider icon in Trusts |
| Type badge | "Service Account" on every row | Removed |
| Row actions | Reset Service Key button in the Account cell, trash icon in Operations | One row menu |
| Deactivate | Browser confirm that does not name the account | `AlertDialog` that names the account |
| Inactive accounts | Checkbox under the table reveals a second table | State filter on the same list |
| Empty list | "No data" | What the account type is for, with the Create action |
| Create and edit surface | Service accounts: sheet. Workload identities: centered dialog | A sheet on both pages |

Unchanged: the service account form, the workload identity steps with their fields and wording,
the paged load-more footer, permissions, and the deactivate and restore semantics.

## Shared layout

Order follows the table anatomy in `docs/agents/frontend-ux.md`: page info, toolbar, table, footer.

### Page info

Use `ProjectPageInfo` or `WorkspacePageInfo`. There is no Learn more link until the docs have a
page for these account types.

- **Service accounts:** "Service accounts sign in with a key. Use them for scripts, Terraform and
  API calls. For CI on GitHub or GitLab, a workload identity needs no stored secret."
- **Workload identities:** "Workload identities let CI jobs sign in with their own short-lived
  token, so there is no secret to store. Each one names the jobs it trusts and the role they get."

### Toolbar

- **Search** filters by name and email on the server. Both list APIs already accept `name` and
  `email` with `.contains()`.
- **State filter** has Active, Inactive and All. Active is the default.
- **Create** stays at the end.

### Name cell

- Line 1 is the title, and it is the control that opens Edit. The row itself is not a button, so
  the controls inside it stay reachable by keyboard and screen reader.
- Line 2 is the email with a copy button.
- Both lines stay on one line. A long title or email truncates with the full value in a tooltip.
- An inactive row adds an "Inactive" label after the title. The label never truncates.

### Row menu

One overflow menu per row. The destructive item is last, after a separator.

| List | Active row | Inactive row |
|---|---|---|
| Service accounts | Reset service key, Edit, Deactivate | Restore |
| Workload identities | Edit, Deactivate | Restore |

- Deactivate asks in an `AlertDialog` that names the account and says it can be restored.
- Reset service key keeps its confirmation and its wording.
- While a new or reset key is waiting to be copied, the row shows **Copy service key** beside the
  menu, as today. The key is returned once.

### States

- **Loading:** a table skeleton with the final column widths. Never the empty state.
- **Empty, no filter:** one line, such as "No service accounts". The page info above already says
  what the type is for, and the toolbar already has Create. The paging footer is hidden.
- **Empty, filtered:** "Nothing matches" with a clear-filter action.
- **Narrow screens:** Name gives way first. It takes the width the other columns leave and
  truncates, so Trusts, Roles and the row menu stay in view. Below a minimum Name width, the table
  scrolls horizontally inside its frame.

## Service accounts list

| Name | Roles | Created | |
|---|---|---|---|
| **terraform-apply**<br>`terraform-apply@billing.service.bytebase.com` | Project Owner | Mar 4, 2025 | ⋯ |
| **schema-migrator**<br>`schema-migrator@billing.service.bytebase.com` | Project Developer, SQL Editor User | Jan 20, 2026 | ⋯ |
| **legacy-deploy-script**<br>`legacy-deploy-script@billing.service.bytebase.com` | Project Owner | Jun 18, 2023 | ⋯ |
| **backfill-2024**<br>`backfill-2024@billing.service.bytebase.com` | No role | Jan 8, 2024 | ⋯ |

- **Roles** lists the roles bound to the account in the page's scope: the project IAM policy on a
  project page, the workspace policy on the workspace page. Every role shows as a chip, as on the
  Members page, and the chips wrap.
- **No role** is muted text, not a warning. A new account without a role is a normal intermediate
  state.
- **Created** uses the shared timestamp (`HumanizeTs`): relative time up to 30 days, then a date,
  with the full time on hover.

## Workload identities list

| Name | Trusts | Roles | Created | |
|---|---|---|---|---|
| **deploy-ci**<br>`deploy-ci@billing.workload.bytebase.com` | (GitHub) example-org/billing-api<br>Only jobs with `environment: prod` | GitOps Service Agent | Jun 2, 2026 | ⋯ |
| **schema-check**<br>`schema-check@billing.workload.bytebase.com` | (GitHub) example-org/schema | GitOps Service Agent | Mar 14, 2026 | ⋯ |
| **platform-shared-ci**<br>`platform-shared-ci@billing.workload.bytebase.com` | (GitHub) Any repository of example-org | Project Owner | Aug 30, 2025 | ⋯ |
| **mobile-ci**<br>`mobile-ci@billing.workload.bytebase.com` | (GitLab) example-group/mobile/app<br>`ref_type:branch:ref:main` | Project Developer | Nov 8, 2026 | ⋯ |
| **nightly-loader**<br>`nightly-loader@billing.workload.bytebase.com` | (key) oidc.ci.example.com<br>`pipeline:nightly:*` | Project Developer | Oct 1, 2026 | ⋯ |

In the table above, (GitHub), (GitLab) and (key) stand for icons.

### Trusts

Trusts restates what the create form asked, in the form's own words. The form
(`frontend/src/components/WorkloadIdentityFlow/`) has a token source, an owner, an optional
repository, and a subject pattern it builds from them. The cell reads the same parts back with
`parseSubjectPattern`, from the provider type and subject pattern the list response already
carries.

| What the form holds | Line 1 | Line 2 |
|---|---|---|
| Owner and repository | owner/repository | None |
| Owner, repository left blank | Any repository of *owner* | None |
| GitLab group and project | group/project | None |
| GitLab group, project left blank | Any project in *group* | None |
| A pattern that pins an environment | owner/repository | Only jobs with `environment: name`, or "Only jobs that run in an environment" for any |
| A pattern with any other remainder | owner/repository | The remainder, monospace |
| A pattern the fields cannot read | The subject pattern, monospace | None |
| Other OIDC | Issuer host | The subject pattern, monospace |

- **Line 1 is the form's own summary.** The Done step says "Jobs in *target* can now sign in".
  Line 1 is that target in the same words, such as "any repository of *owner*".
- **The wording is the form's.** The cell and the form share their strings, so a later change to
  the form's wording changes the cell with it.
- **The default rule has no second line.** The form's default lets every job in the repository
  sign in. The cell adds nothing to it and does not mark it as wide.
- **No branch or environment vocabulary of its own.** The form has no branch field. An environment
  shows in the words of the form's own warning. Anything else after the repository shows as
  written.
- **Each line stays on one line.** The cell has a maximum width. A longer line truncates with the
  full value in a tooltip.
- **Provider icon.** The form's token source icon leads line 1: the GitHub or GitLab asset the
  form uses, or the key icon for Other OIDC. Its tooltip and accessible name are the form's
  titles: GitHub Actions, GitLab CI, Other OIDC.
- **Owner and repository IDs are not shown.** A GitHub subject may carry them
  (`repo:OWNER@ID/REPO@ID:...`). The cell shows the names.
- **Roles** and **Created** follow the service account rules.
- **Line 1 is regular weight.** Only the name is emphasized, so each row has one headline.

## Create and edit

Create and Edit open a sheet on both pages. This follows the surface table in
`docs/agents/frontend-ux.md`: a resource form goes in a sheet, and a short multi-step workflow
goes in a wide sheet.

| Page | Surface | Width | Content |
|---|---|---|---|
| Service accounts | Sheet | `standard`, 704px | The shipped form: Name, Email, Roles |
| Workload identities | Sheet | `wide`, 832px | The shipped four steps |

### Service account sheet

Unchanged. It is `CreateServiceAccountSheet` in `ServiceAccountsPage.tsx`.

- Create closes the sheet and shows a "Created" toast. The new row shows **Copy service key**.
- Edit opens the same sheet with the email locked.

### Workload identity sheet

The four steps move from the dialog into a sheet. Their fields, validation, wording and the edit
rules are the shipped ones. Only the container changes.

- **Header:** "Create workload identity" or "Edit workload identity", with the sheet's close
  button. Edit shows the "Editing *email*" banner under the title.
- **Step row:** the four steps run in one row under the header. It is the shared `StepIndicator`
  in its default horizontal style with `fitOneRow`, the one the GitOps setup page uses.
- **Step titles:** every title shows. When they do not fit one row, only the current step keeps
  its title.
- **Going back:** a finished step can be clicked. Done is reached only by saving.
- **No step descriptions.** The dialog's rail printed a line under each title, such as "Select the
  CI provider". Each step already opens with its own question and description.
- **Body:** only the step body scrolls, and each step opens scrolled to its top. The fields get
  about 780px, against about 620px beside the dialog's rail.
- **Footer:** stays in place. Back is on the left from step 2. Cancel and Continue, Create or
  Update are on the right. The last step shows only Done.
- **Height:** the sheet is full height, so the footer does not move between steps. The dialog's
  fixed 680px height is no longer needed.
- **Phone:** the shared sheet takes the full width. Fields go to one column and code blocks
  scroll sideways, as today.
- **Callers:** the Workload identities page and the GitOps setup page open the same sheet. GitOps
  setup keeps closing after Create and offering GitHub Actions and GitLab CI only.

## Implementation

### Backend

The layout, Trusts, Roles and the create surface need no API change. Roles are joined on the client
from the project or workspace IAM policy.

Created needs a small server change. Both resources declare `create_time`, but
`convertToServiceAccount` and `convertToWorkloadIdentity` never set it. The `service_account` and
`workload_identity` tables already have `created_at`. The store reads it into its messages on
every read path and returns it from create, and both converters set `create_time`.

### Lists

- `frontend/src/routes/workspace/MachineAccountList.tsx` holds what both pages share: the
  toolbar, the Name, Roles and Created cells, the row menu, the deactivate dialog, and the empty
  and loading rows.
- `workloadIdentityTrust.ts` splits a workload identity's config into the Trusts cell's parts.
- Both pages load one list, filtered by state on the server, in place of the second table.

### Workload identity sheet

- `WorkloadIdentityDialog.tsx` and its test become `WorkloadIdentitySheet.tsx`, built on `Sheet`
  with `width="wide"`. State, validation, submit and the four step components are untouched.
- `WorkloadIdentitiesPage.tsx` and `ProjectGitOpsPage.tsx` open the renamed component.
- The five rail description strings are removed from all five locale files.

## Alternatives considered

Each was mocked on the same example accounts.

### Service accounts list

| Alternative | Shows | Why not |
|---|---|---|
| Identity only | Name, description, status, created. The rest on a detail page | A review means opening every row. Nothing shows what an account can do |
| Key-led | Key hint, last used, key created | Says nothing about what the account can do. It models the key, not the account, and needs sign-in recording |
| Status-led | One health verdict per row, with count cards as filters | Every threshold is an opinion. The CIS AWS benchmark uses 45 days unused, Google and Microsoft use 90, and NIST leaves it to the organization |

### Workload identities list

| Alternative | Shows | Why not |
|---|---|---|
| Trust tuple | Provider, then repository, branch and environment as labeled fields | No name or role. It fits a list nested under the thing being accessed, where the grant is obvious |
| Raw claims | Issuer URL, subject, audience | The reader parses the subject by eye and must recognize an issuer URL. Long values truncate |
| Health-led | Working or failing, last result, breadth | Hides which repository is trusted. Needs sign-in and failure recording, which this proposal leaves out |

### Create surface

| Alternative | What opens | Why not |
|---|---|---|
| Keep the dialog | The wizard in an 896 by 680px dialog with a 224px step rail | Two sibling pages answer the same button with two surfaces. The dialog was chosen for field width beside the rail, and the sheet with steps on top gives the fields more |
| Sheet with the rail kept | A wide sheet with the same side rail | The fields drop to about 555px, less than in the dialog. Two-column rows get tight |
| Dialogs on both pages | The service account form in a small dialog | A resource form in a dialog breaks the surface table and departs from users and the other sheet forms |
| Keep both and state a rule | A form in a sheet, a guided setup in a dialog | The two pages still open different surfaces, and the surface table needs a new line |
| Full page wizard | A route for the four steps | The flow is short and is also opened from GitOps setup, which a route would leave |

### Smaller decisions

| Decision | Rejected | Why |
|---|---|---|
| No page title | A 24px title above the description | The sidebar names the page and no sibling list page has a title |
| Description as the page info note | Plain gray text | Matches the pages that already carry a description |
| Provider icon, no word | Icon plus word, or word only | The mark is faster to scan and the word repeats it |
| "Any repository" in words | A warning badge | The form does not warn about it. PyPI prints "(Any)" in place of the missing value |
| Remainder as written | "branch main" and similar labels | The create form has no branch field, so the list would name things the form never does |
| No role as muted text | A warning | It would fire on every new account |
| No key age column | Key created date | Most products flag non-use or expiry, not age |
| State filter | A second table for inactive accounts | No reviewed product stacks a second table. Grafana, GitLab and Datadog filter one list |
| Two pages | One list with a type filter | Trusts has no meaning for a service account, and the row menus differ |
| The form's wording in Trusts | New wording, such as "All repositories under *owner*" | The list should match the form users see |
| Column named Trusts | Repositories, the form's step name | Other OIDC rows have a subject, not a repository |
| No owner ID warning in the list | Repeating the create form's warning on each row | It would mark most existing rows. The edit form keeps the warning |
| Roles omit expired bindings | Showing them marked as expired | The cell answers what the account can do now. The Members page keeps showing them |
| Trust fields stay editable | Delete and recreate, as five reviewed trust UIs do | Recreating changes the identity's email and drops its role bindings. Edit reuses the create steps |
| Step row in the default style | The rail look with every title and green finished steps | The shared component's rail look shows only the current title. The default style needs no change and matches GitOps setup |

## Evidence

A review of other products' machine identity lists backs Trusts as readable fields and is split on
Role. The create surface rests on this product's own UX contract, not on other products.

- **Trust as readable fields.** PyPI, crates.io and RubyGems show a trusted publisher as provider
  plus labeled repository, workflow and environment, never a raw subject
  ([PyPI template](https://github.com/pypi/warehouse/blob/main/warehouse/templates/manage/manage_base.html),
  [crates.io](https://github.com/rust-lang/crates.io/blob/main/svelte/src/routes/crates/%5Bcrate_id%5D/settings/%2Bpage.svelte)).
- **Role in the row.** Grafana's service account list has a Roles column
  ([source](https://raw.githubusercontent.com/grafana/grafana/main/public/app/features/serviceaccounts/ServiceAccountTable.tsx)).
  The AWS Roles list has none.
- **Two subject formats.** GitHub repositories created, renamed or transferred after July 15, 2026
  emit subjects with numeric IDs
  ([changelog](https://github.blog/changelog/2026-04-23-immutable-subject-claims-for-github-actions-oidc-tokens/)).

Limits of the evidence:

- Most vendor docs do not name list columns and the dashboards need a login. Only a few lists were
  read from product UI source: Grafana, GitLab, PyPI, crates.io, RubyGems, Argo CD and OpenShift.
- No reviewed list was confirmed to show a readable trust rule beside a separately granted role.
  The registries show the rule with no name or role. The AWS Roles list shows trusted entities,
  but there the row is the role.
- A provider icon in a row has no confirmed precedent. It is new, not shown wrong.
- The field widths for the create surface were measured in a mockup, not in the product.

## Future work

### Last used and Last sign-in

Out of this proposal. It is the only part that needs a schema change and a write on the sign-in
path. Until it ships, the lists do not show whether an account is still used or whether an
identity has ever signed in.

- **Intent:** a Last used column on service accounts and a Last sign-in column on workload
  identities. Both show the last successful sign-in, or "Never".
- **Display:** freshness readings follow the work-queue rule in
  [timestamp-display.md](timestamp-display.md): relative time up to 30 days, then an absolute
  date.
- **Evidence:** the AWS IAM Roles list has Trusted entities and Last activity columns
  ([AWS docs](https://docs.aws.amazon.com/IAM/latest/UserGuide/id_roles_manage_delete.html)).
  GitLab's token table prints "Last used:" with a relative time or "Never"
  ([source](https://gitlab.com/gitlab-org/gitlab/-/raw/master/app/assets/javascripts/vue_shared/access_tokens/components/access_token_table.vue)).
  Grafana's token table has "Last used at"
  ([source](https://raw.githubusercontent.com/grafana/grafana/main/public/app/features/serviceaccounts/components/ServiceAccountTokensTable.tsx)).
  In Cloudflare's November 2023 incident, four credentials went unrotated because they were
  believed unused
  ([post-mortem](https://blog.cloudflare.com/thanksgiving-2023-security-incident/)).

Implementation notes, from reading the sign-in code:

- Both account types live in their own tables, not in `principal`, so the end-user
  `last_login_time` in the principal profile does not cover them.
- **Store:** a nullable `last_login_time` column on `service_account` and `workload_identity`.
- **Write:** set it on a successful sign-in. Service accounts sign in through `Login`
  (`finalizeLogin` in `backend/api/v1/auth_service.go`) and workload identities through
  `ExchangeToken` in the same file. A failed write is logged and does not fail the sign-in.
- **Read:** an output-only `last_login_time` on `ServiceAccount` and `WorkloadIdentity`.
- **Sign-in, not every request:** both token types expire after one hour, so an account in use
  signs in at least hourly. The request path stays free of writes.
- **Not read from the audit log:** the list would aggregate every sign-in row for each account on
  every page load.

Questions to settle then:

- **Accounts older than the column.** One that has not signed in since shows "No use recorded",
  not "Never", unless the migration backfills from the audit log.
- **Failed sign-ins.** Recording successes only leaves a never-working identity at "Never".
- **Unused flags.** A fixed "unused for 90 days" flag needs a number, and the standards disagree
  on it.

## Non-goals

- Sorting and column configuration.
- A health or posture dashboard.
- Key expiry, or more than one key per service account.
- Changing how roles are granted.
- Merging the two account types.
- Changing the workload identity steps, their fields or their wording.
- Changing the service account form.

## Assumptions

- Workspace and project pages use the same layout, with roles read from their own scope.
- The list response keeps carrying the workload identity configuration.
- `workload-identity-creation-flow.md` gets its D1 and D8 updated when the sheet ships.

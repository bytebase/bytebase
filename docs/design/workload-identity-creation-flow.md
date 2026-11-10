# Workload Identity creation flow

Status: implemented in this PR · 2026-11-09 · [BYT-10429](https://linear.app/bytebase/issue/BYT-10429) · builds on the BYT-10349 server fix, [#21931] (merged)

A workload identity lets a CI job sign in to Bytebase with the job's OIDC token instead of a stored
secret. Today it is created in one sheet with up to eleven fields: the provider is a dropdown, the
issuer and subject pattern hide under "Advanced", and nothing explains what any field binds to.
The role is optional, so an identity can be saved that signs in and can do nothing. After Create
the sheet closes, and nothing says what to do next.
**The sheet becomes a four-step dialog that asks one question per step: where the tokens come
from, which repositories may sign in, and who the identity is in Bytebase. It then ends on the
email the workflow uses. Every value the old sheet stored is still stored, in the same fields, and
every identity that signs in today keeps signing in.** Frontend only: no proto, API, store or
migration change.

The step layout follows the Claude Platform's "Connect workload" wizard (left rail, one question
and one sentence per step, choice cards). The mock-up (v47, with desktop, phone and variant
screens) is kept outside the repo.

## Problem

Verified on `main` at `f52ecbc`, in `CreateWorkloadIdentitySheet`
(`frontend/src/components/CreateWorkloadIdentitySheet.tsx`):

- **One long form, in an order that doesn't match the task.** The fields are: Name, Email, Roles,
  Platform, Owner, Repository, Allowed branches/tags (GitLab), Branch, Audience. Behind
  "Advanced" are Issuer URL and Subject pattern. Other OIDC shows Issuer URL, JWKS URL and Subject
  pattern instead. The account's name and email come first, before the user has said what will
  sign in.
- **The provider is a dropdown.** The UX contract (`docs/agents/frontend-ux.md`, Choice Controls)
  asks for visible choices with descriptions when choosing an identity-provider type.
- **What signs in is hidden.** The subject pattern decides which jobs may sign in, and it sits
  under Advanced. Editing Owner, Repository or Branch recomputes it from scratch. That widens a
  hand-written pattern such as `repo:o/r:environment:prod` to `repo:o/r:*` (comment at lines 252 to 259).
- **No owner ID.** GitHub moves repositories created, renamed or transferred after 2026-07-15 to
  ID-based subjects (`repo:owner@123/name@456:...`). #21931 makes the server accept both formats
  for a name pattern, and it also accepts `repo:owner@ID/...`, which survives the owner name being
  freed and taken over. The form has no way to set the ID.
- **No role is allowed.** Roles are optional, and hidden entirely when the user cannot set IAM
  policy. The identity is created, signs in, and has no permission.
- **No next step.** Create shows a "Created" toast and closes the sheet. The email the workflow
  needs is not shown, and there is no way to check that the identity works.

## Principle

> Ask for what the user knows, in the order they know it. Prefill what Bytebase knows, and show
> it read-only until the user chooses to change it. Store exactly what the old form stored, so one
> path serves every identity, old or new.

## Decisions

**D1 · Four steps in a centered dialog.** Title "Create workload identity" with a close button.
Steps: Token source, Repositories, Account, Done. A 224px left rail shows each step's title and
one-line description ("Select the CI provider", "Match owner and repository", "Email and role",
"Use it in your workflow"). Finished steps turn green and the line between them follows; a step can
be clicked to go back to it, or forward past steps that are complete; Done is reached only by
saving. The footer: Back (bordered, with a left arrow, from step 2) on the left; Cancel and
Continue / Create (primary) on the right; on Done, only Done. Each step is validated before Continue, with errors beside the fields, and
Back keeps what was entered. Every step opens scrolled to its top. The dialog has a fixed height:
680px, or the window height minus 64px when the window is smaller. The buttons stay in the same
place on every step; a short step leaves space above them, and a long one scrolls inside the step
body. The scroll area leaves 4px of room for focus rings, so no field border is cut. Fields use the
shared label, with a grey "(optional)" on optional fields, and their hint below the input. This
is a `Dialog`, not the wide sheet that the UX contract suggests for short wizards. The rail with
descriptions and the side-by-side layout follow the Claude Platform wizard that the flow is
modelled on, and a side sheet leaves the form too little width beside the rail. All other wizard
rules in the contract apply. Rejected: one page with sections; it keeps today's problem of a long
form with no order.

**D2 · Step 1, Token source: "Where do your CI jobs get their token?"** Three choice cards
(`RadioGroup` with the radio hidden; each card shows an icon, a title and a subtitle): GitHub
Actions, GitLab CI, Other OIDC. GitHub Actions is selected when the dialog opens for a new
identity. Below a separator, for the chosen provider:

- **GitHub and GitLab.** One sentence, then Issuer URL and Audiences, prefilled
  (`https://token.actions.githubusercontent.com` or `https://gitlab.com`, and audience
  `GENERATED_WORKFLOW_AUDIENCE`, `bytebase`). Issuer URL is shown read-only in monospace and
  Audiences as chips, each with an Edit link. Issuer URL always has the hint "Change it only for
  GitHub Enterprise Server, GHE.com, or an enterprise custom issuer." (GitLab: "Change it only for
  a self-managed GitLab instance, to that instance's URL."). Edit turns Issuer URL into an input
  and Audiences into the shared `TagInput`: Enter adds a value, × removes one, and at least one is
  required. After Edit the link reads Reset, which puts the preset back.
- **Other OIDC.** Issuer URL (required), Audiences (required, starts with `bytebase`) and JWKS URL
  (optional, last). All are editable from the start. JWKS URL hint: "Where Bytebase downloads the
  issuer's public keys. Leave blank if the issuer publishes `/.well-known/openid-configuration`."
  GitHub and GitLab never show JWKS URL, because both publish discovery documents.

Changing the provider resets the issuer, audiences, JWKS URL, fields and subject pattern, as today.

**D3 · Step 2, Repositories: the fields fold into `subject_pattern`.** No new stored field. The
fields are a view of the one stored pattern.

- **GitHub.** Owner (required), Repository (optional, blank means any repository of the owner),
  Owner ID (optional, digits only). Owner ID hint: "Get it with `gh api users/OWNER --jq .id`. It
  never changes, even when the account is renamed.", with the typed owner in place of OWNER.
- **GitLab.** Group (required), Project (optional). There is no group ID: GitLab's subject has no
  place for one.
- **Other OIDC.** The step is titled "Which tokens can sign in?" and has only Subject pattern
  (required, editable), with the hint "Must equal the token's `sub`. End with `*` to match a
  prefix."

The pattern is built from the fields:

| Fields | Pattern |
|---|---|
| Owner `acme-corp` | `repo:acme-corp/*` |
| Owner, Repository `deploy` | `repo:acme-corp/deploy:*` |
| Owner, Owner ID `111`, Repository | `repo:acme-corp@111/deploy:*` |
| Group `acme`, Project `deploy` | `project_path:acme/deploy:*` |

It follows the grammar the server parses in `backend/plugin/idp/wif/github.go`
(`repo:OWNER[@OWNER_ID]/*` or `repo:OWNER[@OWNER_ID]/REPO[@REPO_ID]:REST`). Since #21931, every
GitHub pattern in the table matches tokens in both subject formats.

Below the fields, Subject pattern is shown read-only with an Edit link and the hint "Built from the fields above. Edit it only for a rule the fields can't express." After Edit:

- Typing in the pattern updates the fields from it.
- Changing a field afterwards replaces only that field's part and keeps the rest of the pattern
  (for example `:environment:prod`). It never widens the pattern.
- Clearing Repository while the pattern names more than the repository is refused, with "The
  pattern names more than the repository. Reset it before clearing the repository."
- Reset rebuilds the pattern from the fields, and makes it read-only again.
- A pattern the fields cannot read, for example one that doesn't start with `repo:OWNER/`,
  disables the fields. The pattern is then matched by prefix only, as today.
- A pattern that names one environment shows a warning: "Only jobs with `environment: prod` can
  sign in. Jobs without an environment, such as the SQL review job, will fail."

For GitHub, a box below Owner ID explains the owner ID. It is always shown, from the moment the
step opens, with the same text, so typing Owner or Owner ID never changes the step's height. While
Owner ID is empty it is a warning; once an ID is typed, the same box turns neutral (info). Title:
"The owner ID protects this rule from name reuse". Body: "Without it, the rule trusts the name alone. If the account is renamed or deleted, anyone can register the name and sign in as this identity." The body ends
with the inline link "GitHub changelog", which opens GitHub's announcement in a new tab:
https://github.blog/changelog/2026-04-23-immutable-subject-claims-for-github-actions-oidc-tokens/. It is the one external link in the dialog, because the risk is
GitHub's behavior and GitHub's post explains it. The step 2 wording is kept short so that the step
fits the fixed height without scrolling, also with the edit banner.
There is no field for the branch, tag, event or environment. Today's Branch and GitLab
"Allowed branches/tags" fields are dropped; a pattern edit still expresses them (D9).

**D4 · Step 3, Account: "Who is it in Bytebase?"**

- **Email** (required). The name is prefilled from the repository name plus `-ci` (for example
  `deploy-ci`), or from the owner name plus `-ci` when Repository is blank. The fixed suffix is shown beside it (`@<project>.workload.bytebase.com`, or
  `@workload.bytebase.com` at workspace level). A debounced `fetchWorkloadIdentity` lookup shows
  "Already used by another workload identity." before Create. A deleted identity also counts as
  used, because the server refuses the email. Hint: "Your workflow signs in with this email. It
  cannot be changed later."
- **Display name** (optional). Hint: "Shown in member lists. Blank uses the email name." Its
  placeholder is the email name.
- **Role** (required, last). Prefilled with GitOps Service Agent (`roles/gitopsServiceAgent`) and
  shown read-only with an Edit link. It comes last for the same reason Subject pattern does in
  step 2: it is a prefilled value that the user rarely changes. Edit turns it into the shared
  `RoleSelect` in single-select mode (`multiple={false}`, `clearable={false}`, with
  `scope="project"` in a project), and Reset puts the default back. The default is the same at
  project and workspace level. Hints: "What the account can do in this project. GitOps Service
  Agent is the role GitOps workflows need." and, at workspace level, "What the account can do in
  every project of this workspace." A user without
  `bb.projects.setIamPolicy` (`bb.workspaces.setIamPolicy` at workspace level) sees a warning in
  place of the value, with no Edit link. Create stays allowed and no role is granted. Today the
  field is hidden.

Create runs on leaving step 3. It makes the same two calls as today: create the identity, then set
the IAM policy with the role. If the role call fails, the identity already exists, so the flow
still moves to Done, which shows the error and says the role was not granted.

**D5 · Step 4, Done: the email, then an optional check.** "Workload identity created. Jobs in
acme-corp/deploy can now sign in to project acme-deploy as this identity." (the project's title;
"any repository of acme-corp" when Repository is blank). The email follows in a grey box with a
Copy link and the hint "Set it as `BYTEBASE_WORKLOAD_IDENTITY` in your workflow. Workflows
generated on the GitOps page already use it."

Below a separator, the section "Check that it signs in" (with a grey "(optional)") and "Run this
workflow once to confirm the identity works before you rely on it." show one complete test file.
The code block has a header bar: the file name on the left, in monospace, and the shared Copy
button on the right. There is only one file, so the name is a plain label, not a tab, and it wraps
on a narrow screen instead of being cut off. Other OIDC has no test section. For GitHub,
`.github/workflows/bytebase-wif-test.yml`:

```yaml
name: bytebase-wif-test
on:
  push:
    paths:
      - .github/workflows/bytebase-wif-test.yml
permissions:
  id-token: write
  contents: read
jobs:
  sign-in:
    runs-on: ubuntu-latest
    steps:
      - name: Fetch GitHub OIDC token
        run: |
          OIDC_TOKEN=$(curl -sS --fail -H "Authorization: bearer $ACTIONS_ID_TOKEN_REQUEST_TOKEN" \
            "$ACTIONS_ID_TOKEN_REQUEST_URL&audience=bytebase" | jq -r .value)
          echo "::add-mask::$OIDC_TOKEN"
          echo "OIDC_TOKEN=$OIDC_TOKEN" >> $GITHUB_ENV
      - name: Exchange for a Bytebase access token
        run: |
          RESPONSE=$(curl -sS --fail-with-body <EXTERNAL_URL>/v1/auth:exchangeToken \
            -H "Content-Type: application/json" \
            -d '{"token":"'"$OIDC_TOKEN"'","email":"<EMAIL>"}') \
            || { echo "Sign-in refused: $RESPONSE"; exit 1; }
          echo "$RESPONSE" | jq '.accessToken = "<redacted>"'
```

For GitLab, a `bytebase-wif-test` job for `.gitlab-ci.yml`, with the header ".gitlab-ci.yml add
this job" (the last words in grey): `image: alpine:3.20`, `when: manual`,
`id_tokens` with `aud: bytebase`, `apk add --no-cache curl jq`, the same exchange, and the `jq`
redact line as its own script item.

- The test runs on a push of its own file, on any branch. It needs no pull request to a protected
  main, and it does not run on other pushes. `workflow_dispatch` was rejected because it only
  works from the default branch.
- `--fail-with-body` makes a refusal fail the job and print the server's reason. Plain `curl -sS`
  exits 0 on HTTP 401.
- `<EXTERNAL_URL>` is the workspace's External URL. When `needConfigureExternalUrl()` is true,
  the test file is still shown, with the placeholder `https://YOUR-BYTEBASE-URL`, and a compact
  warning above it: "The External URL is not set, so this file uses a placeholder. Replace
  https://YOUR-BYTEBASE-URL with the address your CI jobs can reach, or set the External URL." and,
  for a user who can set it, a "Configure now" link to the workspace settings. A run with the
  placeholder fails loudly ("Could not resolve host").
- The audience is the identity's first audience.

Rejected: a live "waiting for the first sign-in" status in the dialog, which needs new backend
state; the job log already shows the result and the refusal reason. Also rejected: an "Add to an
existing workflow" tab, which repeated the generated GitOps workflow.

**D6 · Edit reuses all four steps.** The title is "Edit workload identity", and a compact one-line
info banner says the user is editing an existing identity: "Editing
deploy-ci@acme-deploy.workload.bytebase.com. Changes apply from its next sign-in." The rail has
four steps. Steps 1 to 3 are reachable from it; step 4 only through Update. Steps 1 and 2 show
Continue; step 3 shows Update. Update saves and moves to step 4, which shows "Changes saved" and
"Jobs in acme-corp/deploy can sign in to project acme-deploy as this identity from their next
run." There is no email row, because the banner shows the email. Below a separator is the same
optional "Check that it signs in" section as on create, so the user can re-check sign-in after
changing the trust rule; an existing identity has no other place to get the test file. Done closes
the dialog. The steps load what is stored: the issuer, the audiences, the JWKS URL, and the
pattern, with the fields read from it. There is no separate path for identities made by the old
form.

- **Email** is locked, as today.
- **Provider** is locked when one is stored, so a stored pattern is never rebuilt in another
  provider's format. When none is stored (`PROVIDER_TYPE_UNSPECIFIED`), the cards start empty and
  required, as on create, because the server refuses a save without one
  (`validateWorkloadIdentityConfig`). Today the sheet opens such an identity on GitHub.
- **Roles** are not edited here, as today. They are managed on the Members page.
- **Update** is enabled only when something changed. It sends `title`, and sends
  `workload_identity_config` only when the config changed, as today. The config always carries
  the stored `jwks_url`, also for GitHub and GitLab where the field is not shown.

**D7 · A server error goes to its step.** An `InvalidArgument` whose message names the provider,
the JWKS URL, the subject, the audience, the issuer or the email (checked in that order) moves the
user to the step with that field and shows "Bytebase cannot accept this configuration:" and the
message beside it. Any other `InvalidArgument` is shown beside Subject pattern. Today the sheet
opens Advanced instead.

**D8 · Phone layout.** Below 640px:

- The rail becomes one row at the top: numbered circles joined by lines, with only the current
  step's title and no descriptions. This is the shared `StepIndicator` with `variant="rail"`.
- Fields go to one column.
- The dialog fills the screen height minus 16px top and bottom. The title, the step row and the
  footer stay in place; only the step body scrolls. The footer gets a top border.
- The email suffix moves under the name.
- Code blocks scroll sideways.
- The dialog sets its own width (full width minus 16px each side), because the shared
  `DialogContent` is `100vw - 8rem` wide, which leaves 247px on a 375px phone.

**D9 · Nothing that works today breaks.**

- The API and stored data are unchanged.
- Every stored pattern, audience, issuer and JWKS URL loads and saves unchanged, unless the user
  edits it.
- Patterns the old form wrote, including branch patterns such as `repo:o/r:ref:refs/heads/main` and
  GitLab `ref_type` patterns, load with their remainder kept.
- Hand-written workflows and API clients are untouched.
- The API still accepts an identity with no role; the role is required only in the UI.
- The GitOps page reads Owner and Repository from the pattern with
  `parseWorkloadIdentitySubjectPattern` (`frontend/src/utils/workloadIdentity.ts`). It is taught
  to drop `@ID` parts, so `repo:acme-corp@111/deploy:*` gives owner `acme-corp`, not `acme-corp@111`,
  and its repository link stays right.

## States

| Provider | Step 1 shows | Step 2 shows | Step 4 test |
|---|---|---|---|
| GitHub Actions | Issuer, Audiences (read-only, Edit) | Owner, Repository, Owner ID, owner ID box, Subject pattern (read-only, Edit) | `.github/workflows/bytebase-wif-test.yml` |
| GitLab CI | Issuer, Audiences (read-only, Edit) | Group, Project, Subject pattern (read-only, Edit) | `bytebase-wif-test` job |
| Other OIDC | Issuer, Audiences, JWKS URL (editable) | Subject pattern (editable) | None |

| Edit: what is stored | Dialog shows |
|---|---|
| Provider set, pattern the fields can read | Provider locked, fields filled, pattern read-only with Edit |
| Provider set, pattern the fields cannot read | Provider locked, fields disabled, pattern editable |
| No provider | Provider cards empty and required |
| Audiences empty (not repaired by migration 3.23.4) | Audiences tag input empty; at least one value required |

| Role permission | Step 3 Role |
|---|---|
| Can set IAM policy | GitOps Service Agent, read-only with Edit |
| Cannot | A warning in place of the value, no Edit; Create allowed, no IAM call |

| Workspace | Step 4 test section |
|---|---|
| External URL set | The test file with that URL |
| External URL not set | The test file with the placeholder, and a compact warning with "Configure now" |

## Scope

| File | Change |
|---|---|
| `frontend/src/utils/workloadIdentity.ts` | Drop `@ID` from GitHub owner and repository (D9). Ships first, on its own. |
| `frontend/src/utils/workloadIdentity.test.ts` | New: tests for that parser. |
| `frontend/src/components/ui/step-indicator.tsx` and its test | `orientation="vertical"` with an optional `description` per step, an opt-in `variant="rail"` row for phones, and an optional `onStepClick` (D1, D8). The horizontal default is unchanged. |
| `frontend/src/components/ui/tag-input.tsx` and its test | New: a free-text tag input with a read-only chip mode; `Combobox` does not take typed values (D2). |
| `frontend/src/components/RoleSelect.tsx` | An opt-in `clearable` prop for single-select mode; other callers are unchanged (D4). |
| `frontend/src/components/WorkloadIdentityFlow/subjectPattern.ts` and its test | New: build, parse and replace-one-part for GitHub and GitLab patterns (D3). |
| `frontend/src/components/WorkloadIdentityFlow/testWorkflow.ts` and its test | New: the GitHub test file and GitLab job (D5). |
| `frontend/src/components/WorkloadIdentityFlow/StepParts.tsx` | New: the step heading, the field with its label, Edit or Reset link and hint, and inline code for hints. |
| `frontend/src/components/WorkloadIdentityFlow/WorkloadIdentityDialog.tsx` and its test | New: the dialog, keeping the stable-entity ref and keyed inner form; state, validation and submit (D1, D6, D7, D8). |
| `frontend/src/components/WorkloadIdentityFlow/{TokenSource,Repositories,Account,Done}Step.tsx` | New: one component per step (D2–D5). |
| `frontend/src/routes/workspace/WorkloadIdentitiesPage.tsx` | Opens the new dialog. |
| `frontend/src/routes/project/ProjectGitOpsPage.tsx` | Opens the new dialog. |
| `frontend/src/components/CreateWorkloadIdentitySheet.tsx` and its test | Deleted. |
| `frontend/src/locales/{en-US,es-ES,ja-JP,vi-VN,zh-CN}.json` | New `workload-identity` strings in all five locales; the old sheet's strings that are no longer used are removed. |
| `docs/design/workload-identity-creation-flow.md` | This document. |

Tests:

- `workloadIdentity.test.ts`: `@ID` dropped from owner and repository; GitLab patterns; another
  provider's pattern.
- `subjectPattern.test.ts`: parse, build and replace for both providers; round trips; ID forms;
  unparsable patterns; a field change never widens; field validation; the environment a pattern
  names. The GitHub parse cases mirror `backend/plugin/idp/wif/github_test.go`, so the form and
  the server read a pattern the same way.
- `testWorkflow.test.ts`: the GitHub file and the GitLab job parse as YAML and carry the URL,
  email and audience; the GitLab redact line is its own script item.
- `tag-input.test.tsx`: Enter adds, blanks and duplicates are ignored, × and Backspace remove, and
  the read-only mode has no input or remove buttons.
- `step-indicator.test.tsx`: the vertical rail with descriptions, the `rail` row and a step click.
- `WorkloadIdentityDialog.test.tsx`:
  - GitHub create sends `repo:acme-corp/deploy:*`, audiences `[bytebase]`, and the default role,
    GitOps Service Agent; Done shows the test file with its Copy button.
  - The test file uses the External URL, or, while it is unset, the placeholder under a warning.
  - A new identity opens on GitHub Actions with its issuer and audience.
  - The email is prefilled from the owner plus `-ci` when Repository is blank.
  - Every step opens scrolled to its top.
  - The owner ID box is always shown for GitHub, as a warning and then as info, with its link;
    GitLab shows none.
  - Owner ID gives `repo:acme-corp@111/deploy:*`.
  - Role is read-only until Edit; a role picked after Edit is granted; Reset restores the default;
    a cleared role blocks Create.
  - A user without IAM permission creates with no IAM call.
  - A failed role call shows the warning on Done.
  - An email another identity uses blocks Create.
  - Other OIDC sends its JWKS URL.
  - Edit keeps the stored `jwks_url` and the pattern's remainder when only Owner changes, and
    Update stays disabled until something changes.
  - Update ends on step 4 with "Changes saved", the test file header with Copy and no email row;
    Done closes the dialog.
  - Edit of an identity with no provider requires picking one, and keeps the stored pattern.
  - Edit of an identity with no audiences requires one.
  - An `InvalidArgument` about the subject pattern moves the user to step 2.
- Manual: walk the flow on desktop and at 375px wide, as a project owner and as a workspace admin.
  Run the generated test file from a GitHub repository against a server with a public External
  URL.

## Not in this PR

- Which job used an identity, in the audit log ([BYT-10494](https://linear.app/bytebase/issue/BYT-10494)).
- A live sign-in status in the dialog.
- Branch, tag, event or environment fields; the pattern edit covers them.
- A GitLab group ID.
- "Read more" or setup-guide links in the dialog (the GitHub link in the owner warning is the one
  exception), and the Bytebase docs page for workload identity.

[#21931]: https://github.com/bytebase/bytebase/pull/21931

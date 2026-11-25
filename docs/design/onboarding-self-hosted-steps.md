# Onboarding: three self-hosted steps, one name, words about the choice

Status: proposal · 2026-11-24 · amends [onboarding-flow.md](onboarding-flow.md) (BYT-10591)

The setup wizard from [onboarding-flow.md](onboarding-flow.md) serves Cloud and self-hosted with
the same four screens. Walked as a new self-hosted admin, it asks for a name the sign-up page has
just taken, a team size that reaches nothing, and a project name it calls optional beside a skip
button that does something else. Its words also describe the setup itself ("We suggest it first
once setup ends", "Finish line: …") rather than the choice on screen. The rule this doc lands on:
**setup asks each thing once, only where its answer is used, and its words describe the choice on
screen, not the setup around it.** Self-hosted gets three steps and finishes on the project step;
Cloud keeps four. Frontend only.

Mockups of the screens for both deployments are in [#22220](https://github.com/bytebase/bytebase/pull/22220).

## Problem

On a new self-hosted server, Linear BYT-10591 found five things wrong:

- **The name is asked twice.** The sign-up page requires "Username", and step 1 then asks for
  "Your name", optional, prefilled with the same value. Both write the user's title.
- **The team step reaches nothing.** Its one question is the team size. On self-hosted the
  console sends no analytics, invitations are Cloud's (onboarding-flow D7), and the answer's only
  effect is an "add member" step in the guide.
- **"Optional" and "I'll do this later" are the same choice by two routes.** An empty project name
  goes on to the team step and then leaves without a project; the skip button leaves at once. Both
  end on the workspace landing.
- **No way out until step 3.** The wizard offers its skip only on step 3, though the browser's
  Back on step 1 already leaves setup, since nothing holds a self-hosted account in it.
- **The words narrate the setup.** Finish lines, "We suggest it first once setup ends", "One
  question before you finish", "Bytebase needs a database to do anything" and "Bytebase is ready
  once it has a workspace, a project and a database" promise or describe a process instead of the
  choice on screen, and the last two are false on a server that skips the project.

Review of the same screens found three more, on both deployments: completed steps are drawn
exactly like the current one; the form column is centered vertically, so it moves when a screen's
height changes; and notes such as "Pick one to continue" vanish on a selection and move the button
the user is reaching for.

## Given

Verified against the code; none of it changes here.

- **Cloud has no password sign-up.** `getAccountRestriction` sets `DisallowSignup` and
  `DisallowPasswordSignin` on SaaS, so an account arrives by email code, named with its address's
  local part (`auth_service_email_code.go`), or through an identity provider, named with the
  provider's display name, which may be empty (`auth_service_idp.go`). Nothing asks a Cloud user
  for a name before the wizard.
- **The self-hosted first admin signs up on the sign-up page**, which requires a name and prefills
  it from the email address (`SignupPage.tsx`).
- **Analytics is Cloud-only.** `RouteBehaviorRecorder` runs only when `isSaaSMode()` and the
  workspace allows metric collection.
- **The team size keys the guide's "add member" step and nothing else in the console**
  (`getGuideJourney` in `scenarios.ts`, through `readGuideWorkspaceUsage`). Without it the guide
  has no team step, and shows only while the admin is the sole member.
- **The server refuses an empty workspace title** (`UpdateWorkspace`, "title cannot be empty").
- **Masking is Enterprise** (`plan.yaml`); a new workspace is on the free plan.
- **Finish lands, as built:** with the sample, on the project's Databases page with its "database
  ready" prompt; with the own-instance choice, on the project's Instances page with its "connect an
  instance" prompt; without a project, on the workspace landing (`WorkspaceSetupPage.tsx`).

## Principle

> **Setup asks each thing once, only where its answer is used, and its words describe the choice
> on screen, not the setup around it.** A question whose answer reaches nothing on a deployment is
> not asked there. A value another page already asked for is not asked again. A way out is a
> button, offered where a user may want it, and it says what it leaves undone. A heading or label
> asks the question and nothing else on the screen repeats it. A choice may say what it creates or
> leaves for later; nothing narrates the setup or says what the product will suggest after it.

## What this replaces in onboarding-flow

- D2: "Your name" optional, and left blank for local parts that name no person (now D5).
- D3: the goal cards' finish lines (now D6).
- D4: the project name optional on self-hosted (now D3).
- D6, D7: on self-hosted, Finish on screen 4 and the team screen (now D1, D2). On Cloud both stand.
- D9: the query goal's masking-free picture (now D8), and screen 3's dashed drawing of a project
  left without a name (gone with D3).
- D11: screen 3 answered without a project name, and a reason under every disabled button (now D3,
  D7).
- D12: "I'll do this later", the optional name and the self-hosted screen 4 (now D1, D3, D4). Its
  sample wording and its reason for letting a server finish without a project stand.
- States: on self-hosted, step 3's rows read Finish where they say Continue, and the step-4 rows
  are Cloud's.

Where the code has moved on since onboarding-flow merged, the code stands and this doc changes
neither: Finish lands on the pages in Given since #21594 (not D8's project home), and invitees get
SQL Editor Exporter since #21690 (not D7's SQL Editor Read User).

## Decisions

**D1 · Self-hosted has three steps; the team step is Cloud's.** Workspace, goal, project. On
self-hosted the team size reaches only the guide's "add member" step (Given), and the owner
accepted losing that step there: the guide then has no team step on self-hosted and shows while the
admin is the sole member. Cloud keeps its fourth step, where the size is the qualification question
and invitations are sent (onboarding-flow D7). Rejected: keeping a one-question step whose answer
nothing on the server or in analytics receives.

**D2 · On self-hosted, Finish is on the project step.** It is the same one commit (onboarding-flow
D6) minus the invitees: save the workspace name, create the project, prepare the sample when it was
chosen. Everything that belonged to the commit's screen moves with it: the progress line, the
failure naming its step, Try again, "Finish without the sample" when the sample failed, and the
"project exists" state after a failed attempt. A failure is reported for the answers it ran with,
so here its report goes when the name, the ID or the database choice changes. Back still works on
every step, because nothing exists on the server before Finish. Cloud's project step keeps
Continue, and its Finish stays on the team step.

**D3 · Creating a project needs a name; finishing without one is a button.** The project name is
required on both deployments, so the label carries no "optional" and the database choice shows on
arrival. On self-hosted the project step offers "Finish without a project" beside Finish. It saves
the workspace name and the goal, creates nothing, and lands on the workspace landing. It replaces
both the optional name and "I'll do this later", which expressed the same choice twice. An enterprise admin who configures the server through Terraform still finishes
without a placeholder project (onboarding-flow D12). Rejected: an optional name, where an empty field
decides whether a project exists.

**D4 · Self-hosted can skip setup from the first step.** Step 1 offers "Skip setup" beside
Continue. It saves the workspace name when the field holds one, leaves it as it is when the field is
empty (the server refuses an empty title), and lands on the workspace landing. If either way out,
this one or D3's, cannot save the workspace name, its step stays with the error and the button can
be pressed again. Both are offered until a Finish attempt has created the project or has sent a
create whose outcome is still unknown (onboarding-flow D11); a create the server refused leaves no
project, so they are offered again. After that the way out is Finish. Cloud offers neither, because
setup is mandatory there. Rejected: leaving
the browser's Back as the only way out of step 1.

**D5 · The name is asked once.** On self-hosted the sign-up page asks it: its "Username" field
becomes "Your name", which is what it writes, since sign-in is by email. Step 1 then asks only the
workspace name, and the wizard never writes the user's title. On Cloud nothing asked before the
wizard (Given), so step 1 keeps "Your name", now required and always prefilled: the account's
name when it is neither its address nor the address's local part (an identity provider supplied
it), otherwise the local part before any "+", split at dots, dashes and underscores into
capitalized words. That replaces
onboarding-flow D2's blank prefill for local parts that name no person, such as admin or a phone
number: a field that starts filled keeps the one-click arrival, and a required name means every
account has a name of its own to show beside its address. Clearing it disables Continue with
"Enter your name" under the field. On self-hosted, an account that may not rename the workspace has
nothing to answer on step 1 and starts on step 2. Rejected: an optional name, which left the account
named by its address.

**D6 · The words describe the choice on screen.** A screen's heading names its question, its
description says what the thing being chosen is, and a choice may say what it creates or leaves
for later; nothing narrates the setup or says what the product will suggest after it. Removed: the goal cards' finish lines and "We suggest it first once setup
ends" (the goal screen's description becomes "Pick what brought you here."); "We'll suggest a quick
query to start" under "just exploring"; step 1's "Bytebase is ready once it has a workspace, a
project and a database"; the project step's "Bytebase needs a database to do anything"; the team
step's "Your team" and "One question before you finish", the question becoming the heading; and
"Everyone you add joins this workspace", which repeats step 1's description. This withdraws
onboarding-flow D3's finish lines: they described a guide step the screen never names.

**D7 · An unanswered question explains itself; a removed or refused value gets its reason.** While
the question on screen is unanswered (no goal, no team size, no project name on arrival), the primary
button is disabled and nothing is added under the question: its heading or label already asks for
the answer. A value the user cleared (workspace name, Cloud's "Your name") or the server refused
(the project ID) gets its reason under the field, as the UX contract requires for invalid input
(`docs/agents/frontend-ux.md`, Shared Form Anatomy). This narrows onboarding-flow D11's "no Continue
or Finish is disabled without its reason on screen" to values the user changed. Rejected: a note
under every unanswered question, which repeated the heading and moved the button when it vanished.

**D8 · The query goal's picture shows the salary column masked.** The owner chose this knowing
masking is Enterprise (Given): the picture shows what guardrails look like, and the card's words
promise only what the free plan has, a query recorded in query history. It is drawn in the page like
the project and workspace pictures, replacing the still image of onboarding-flow D9.

**D9 · The steps hold still, and only the current step is filled.** The shared step indicator
draws a completed step as a check on a pale accent tint with a plain title, and fills only the
current step with the accent color, so the eye finds where it is; this changes every wizard that
uses it. The setup route's form column starts at a fixed distance from the top instead of being
centered, so the steps and the heading stay in place from step to step and while fields appear.
Rejected: a setup-only variant of the indicator, which would make the same steps look different in
other wizards.

## States

Only the rows that change from onboarding-flow; the rest stand, read as "What this replaces" says.

| Step | State | What is on screen | Button |
|---|---|---|---|
| 1 Workspace | self-hosted | the workspace name only; "Skip setup" beside Continue while D4 offers it | Continue enabled; disabled if the name is cleared, with "Enter a workspace name" under the field |
| 1 Workspace | Cloud | "Your name" (required, prefilled) and the workspace name | Continue enabled; disabled if either is cleared, with its reason under that field |
| 2 Goal | none chosen | two cards and the lighter "Not sure yet, just exploring"; nothing under them | Continue disabled |
| 3 Project | no name | name field empty; the database choice shown, the sample preselected when the server offers it | self-hosted: Finish disabled, "Finish without a project" while D4 offers it; Cloud: Continue disabled |
| 3 Project | self-hosted, named | the project ID under the name; the database choice | Finish enabled; "Finish without a project" while D4 offers it |
| 3 · finishing | self-hosted, Finish pressed | the progress line; the form read-only | n/a |
| 3 · finish failed | self-hosted, a step failed | the error naming the step, gone once an answer it ran with changes; Try again; "Finish without the sample" when the sample failed; name and ID read-only once the project exists | Try again enabled; "Finish without a project" while D4 offers it |
| 3 · left without a project | self-hosted, "Finish without a project" | saves the workspace name and goal; lands on the workspace landing | n/a |
| 1 or 3 · way out failed | self-hosted, "Skip setup" or "Finish without a project" could not save the workspace name | stays on its step with the error | the same button, enabled again |
| 4 Team | Cloud, size unset | the question as the heading, five sizes, nothing under them | Finish disabled |

Removed: self-hosted step 4, and the self-hosted step-3 states "no project" and "skipped".

## Scope

Two slices, shipped in one pull request as separate commits; they share no files.

1. **The wizard: D1 to D8.** Delivers every row of the States table. Files:
   `frontend/src/routes/auth/WorkspaceSetupPage.tsx`; under `frontend/src/routes/auth/workspace-setup/`
   the four screens, `SetupScreenFrame.tsx`, `SetupPanel.tsx` (the drawn query picture, and screen 3
   without its dashed no-project drawing), `draft.ts` (three self-hosted steps, the ways out),
   `finish.ts` (no user title written on self-hosted) and `prefill.ts`; `frontend/src/routes/auth/SignupPage.tsx`; the locales, with the retired strings
   removed; `frontend/src/assets/illustration/setup-query.webp`, deleted. Tests: the existing
   `WorkspaceSetupPage.test.tsx` and `SignupPage.test.tsx`, by the rows they assert, and the existing
   e2e spec `frontend/tests/e2e/workspace-setup/workspace-setup.spec.ts` walking self-hosted through
   three steps: "Skip setup" from step 1, "Finish without a project", and the sample finishing on
   step 3. Depends on nothing.
2. **Shared layout: D9.** `frontend/src/components/ui/step-indicator.tsx` and
   `frontend/src/app/layouts/SplashLayout.tsx`, with their existing tests
   (`step-indicator.test.tsx`, `SplashLayout.test.tsx`) asserting the completed and current step's
   look and the setup column's top alignment. Depends on nothing.

No backend, proto or event change: the events of onboarding-flow D10 stay as they are, and
self-hosted sends none (Given).

## Not in this PR

Making setup mandatory on self-hosted, or a way back into it after leaving. Changing where Finish
lands. The guide beyond losing its
team step on self-hosted. Storing the goal or the team size on the server. The Cloud sign-in page,
which asks for no name. The Rive player of onboarding-flow D9.

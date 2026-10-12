# Onboarding flow, the first four screens

Status: proposal · 2026-10-12

A new Cloud workspace opens on a setup wizard that asks for a goal and "who will use Bytebase
with you", then for your name, a workspace name, a first project and a sample-database checkbox,
and lands on the project's databases page. Since setup became mandatory on Sep 20, 89% of
signups see it and 70% of those finish, so 70% of all signups end with a database. Only 24% go on
to do first work, and the own-database path fails three connection tests in four. The wizard
collects six things, explains none of them, and the one step the product cannot work without is
the one it used to let people skip. The rule this doc lands on: **a screen asks only for what the
next screen uses by name, and the one thing Bytebase cannot work without is never skippable.**
Four screens: workspace, goal, project with its databases, team. The project is explained where
the databases that fill it are added, the own-instance path is the real instance form, and the
last button opens the goal's first action instead of a landing page. Frontend only, with one
small actuator field.

The interactive mockups, the four alternatives this was chosen from, and the funnel evidence are
in the design page kept outside the repo
(`~/Development/design-and-test-docs/2026-09-16-onboarding-flow-designs.html`, rev 4). The
numbers below come from the "Onboarding Wizard Funnel to Oct 10" edition: 396 Cloud signups from
Aug 25 to Oct 10 joined to the metadata database, plus 318 first-session analyst verdicts.

## Problem

The wizard is `frontend/src/routes/auth/WorkspaceSetupPage.tsx` with
`WorkspaceSetupQuestionnaireStep.tsx` as its first step, shown by the router guard while
`isFirstLogin` holds and the account is the sole IAM member, and mandatory on Cloud since Sep 20.
Four things are wrong with what it asks, in what order, and what it does afterwards.

**Information in the wrong order, or not needed.** Step 1 asks the goal and "who will use
Bytebase with you". Step 2 asks your name, the workspace name, a project name with a resource id,
and whether to enable sample databases. The name and workspace name are prefillable and were
asked of everyone though the identity provider supplied them. The team question decided only
whether the guide shows an "add teammate" step: of 41 who said "my team", 4 added a teammate. Of
the 42 who left the mandatory wizard, 11 reached step 2 and left without pressing the button, 4 of
them after pressing Back; the screen asks for things nobody cares about a minute after signup.

**No explanation where one is needed.** The project field is the first time a new user meets
the concept, and the form says nothing about what a project is or why it exists. Analysts tagged
the project/instance/database concept gap in 22 of 140 first sessions, and "form friction" in 47.

**The gate was a skip.** Until Sep 20 every step could be skipped, leaving an empty workspace
whose Databases and Instances pages said "No data". Setup is now mandatory, which fixed the
instance step (24% to 70% of signups with a database), but the mechanism is still a checkbox:
the wizard reports a finish even when the sample request fails, and the own-database path has
no gate at all. Since Sep 20, 62 own-instance connection tests failed and 21 succeeded. The
failures were private addresses (20), wrong credentials (17) and timeouts (12). The console names
the category, tells people to allowlist the Cloud IP without showing it, does not detect a private
range before the test, and offers "Continue anyway", which creates an instance that cannot connect
and a project that says "syncing" forever.

**The landing is a dead end.** Of 100 finishers in the mandatory state, 73 did nothing
afterwards; their median time in the product is under two minutes. The analysts' recurring
description: wizard finished in about twenty seconds with the sample on, the sample opened, then
idling on a page with three prompts and no dominant one, Issues opening on "No data", the SQL
Editor opening blank. The goal answer predicts first work weakly (query choosers run a query 33%
of the time, change choosers create an issue 19%) because it changes only the guide bar, not what
opens.

Two measurement defects shape the design too. Step 1 to step 2 has never emitted an event, so
leavers cannot be placed on a screen. And 17 finishes produced no finish event because the
console's analytics went silent after the wizard mounted; "finished" has to come from the backend.

## Principle

> **A screen asks only for what the next screen uses by name, and the one thing Bytebase cannot
> work without is never skippable.** Every answer is spent on the following screen: the workspace
> name is what invitees join, the goal decides what the database step prepares and what opens at
> the end, the project is the container the databases go into, and the team size keys the guide.
> Nothing is collected for its own sake except one qualification question, labeled as such. The
> database step is a gate: a sample that is verified ready, or an own instance that passed its
> test, with the facts a failing test needs on the screen where it fails.

## Decisions

**D1 · Four screens, in this order: workspace, goal, project with databases, team.** Identity
first because it is known and cheap, and every later screen names the workspace it confirms.
Goal before database because the goal decides which sample state and which first action to
prepare. Database before team because invitees should find something when they click the invite;
Linear and Neon invite early because their product works before any setup, ours works after a
database. Team last, always offered, skippable. Rejected: today's order (goal and team first,
then everything else on one screen), and team as the second screen, which moves the gate later and
invites people into an empty workspace.

**D2 · Screen 1 is a confirmation, not a form.** Your name (optional, prefilled from the identity
provider or the email local part, left empty for alias-like local parts such as admin or info) and
the workspace name (required, prefilled from the email domain, "Peter's workspace" for public mail
domains). Continue is enabled on arrival, so the first click is the only click. Nine of the 42
mandatory-state leavers never clicked anything, consistent with an OAuth redirect returning to a
tab nobody is looking at; a screen with nothing to decide is the cheapest first screen.

**D3 · Two goals.** "Review and roll out a schema change" and "Query data with guardrails", each
with a one-line finish line under the card. "Mark sensitive data" is removed as a goal: masking is
a guardrail on querying and is shown inside the query finish line, and the goal led straight into
an Enterprise gate (1 of 9 choosers acted). Both finish lines are reachable on the free plan.

**D4 · The project is named where the databases are added.** Screen 3 opens with the project name
(required, empty with example placeholder; the resource id derived from it, editable, collapsed)
and one line of explanation, then "Databases for <project>" with the two cards. The project exists
to hold databases, so naming it beside them teaches by adjacency and removes the forward reference
("the databases you add next...") that the workspace screen needed. Rejected: prefilling the name,
which removes the one keystroke burst and the lesson with it; and keeping it on screen 1.

**D5 · The database step is a gate with two cards.** "Continue with sample databases" is the
primary, preselected card: two PostgreSQL databases, hr_test and hr_prod, hosted by Bytebase Cloud,
assigned to the project. "Connect my own instance" must pass Test connection before Continue
enables. There is no skip and no "Continue anyway". 93 of 100 finishers keep the sample on and it
has not failed in 143 attempts since Sep 8; the 7 who turn it off are the own-database path.
Rejected: the checkbox, and the skip.

**D6 · The own-instance path renders the real instance form, in sync by construction.** The
wizard mounts the page's own `InstanceFormBody` and `DataSourceForm` under `InstanceFormContext`,
never a copy and never a list of fields, so a field or control added to the Instances page appears
in the wizard without anyone remembering to add it. The wizard contributes layout only: the
compact field density the form already has, the existing hide-advanced flag turned into one
"Advanced" disclosure (SSL/TLS, SSH tunnel, extra connection parameters, read-only replica), and
where the info panel goes. Engine comes first because it decides which fields exist; environment
defaults to Test; authentication method stays a select (password, AWS RDS IAM, Cloud SQL IAM,
Azure IAM, external secret). A contract test renders both and asserts that the set of field labels
in wizard mode equals the page's. Rejected: a wizard-specific mini form. The 62 failures were not
missing fields; they were missing facts on screen, and a second form would drift the day the page
changed.

**D7 · Width: the form column widens in place when the own-instance card is chosen.** The page's
form is laid out for a 1024px column, but it is already single-column stacked sections with an
info panel that docks beside the form on wide screens and opens as an overlay sheet on narrow ones;
it was built to collapse. Choosing "Connect my own instance" switches the setup layout from 42/58
to 30/70, so at a 1280px viewport the form column is about 900px, inside the range the page serves,
and two-field rows stay two fields. The narrowed left panel becomes the docked info panel: the
engine's connection facts (egress address, the privileges the user needs, SSH tunnel for private
networks, the guides) above the connection checklist that is the gate. Below 1024px the left panel
hides, as `SplashLayout` does today, and the info panel falls back to its overlay. Escape hatch:
if an engine's form is still too tall in place (Oracle wallet, AWS IAM's six fields), the wizard
opens the real page in a full-width drawer and waits for the created instance; same component
either way. In place first; the drawer only if a real engine proves it.

**D8 · Reachability facts before the test, recovery at the failure.** Before a test runs, a host
that is localhost, a private range (10/8, 172.16/12, 192.168/16) or an internal hostname is named as
unreachable from Bytebase Cloud, with the three ways out (public endpoint, SSH tunnel under
Advanced, the sample for now), and the test button stays disabled. The Cloud egress address is shown
inline with a copy action, next to the allowlist instruction, instead of a "Learn more" link. A
failed test names the cause (credentials, timeout, DNS, TLS, unknown), gives the one fact the fix
needs, and offers "Test again" and "Use the sample for now" in place. The draft survives Back and
refresh. Rejected: the current static banner, and the raw driver message as the whole explanation.

**D9 · The sample provisions during screen 4, and Finish waits for it.** Screen 3's Continue
creates the project and starts sample provisioning in the background; screen 4's footer reports
"Preparing hr_test and hr_prod..." and then ready. Finish is enabled only when the sample is ready,
and a provisioning failure shows "Try the sample again" instead of a silent success. Today the
wizard reports a finish even when the sample request fails and offers no retry; the path is
untested only because provisioning has not failed since Sep 8. The minute the sample takes is spent
on the team screen rather than on a spinner or, as today, on a landing page polling 12 times.

**D10 · Team: size required, invites optional, no role.** "How many people will use Bytebase?"
with five options (Just me, 2–10, 11–50, 51–200, 200+) is one required click, the flow's only
qualification question. It keys the guide's team step (only above "Just me"), the plan hint, and
lead scoring; team accounts do first work at 34% against 22%. Emails are chips: type one, press
Enter, type the next; a pasted list splits; an invalid entry is shown struck through and not sent.
There is no role picker. Today's "Role in <project>" selector names Bytebase's project roles,
which a user ten seconds into the product cannot judge; invitees join as members who can query
and propose changes in the project, one sentence says so, and roles are changed later in Members,
where the names mean something. Greptile asks a default role (Admin or Member); Linear, Neon and
Metabase ask for emails only. Rejected: the "Just me / My team" question (it only decided whether
this screen appears, and the screen is now always offered and one click to pass), the role
selector, and the free-text email box.

**D11 · The last button opens the goal's first action.** For the schema-change goal, the
new-change dialog targeting hr_test; for the query goal, the SQL Editor on hr_test with a generated
SELECT. No landing page in between. The landing page and the guide bar are separate work; the
wizard passes them goal, team size and source so the guide starts at the goal step with the
automatic "explore database" step already done. Rejected: landing on the project databases page,
where 73 of 100 finishers idle.

**D12 · The left panel is a story per screen, from bytebase.com's Rive files.** Screen 1 plays the
homepage hero (requests flowing into review); screen 2 swaps to the chosen goal's feature animation
(Change Review, or Dynamic Data Masking) the moment a card is pressed; screen 3 draws a diagram
from what the user typed (sample instance, hr_test and hr_prod, flowing into the project name) or,
for an own instance, the connection checklist with the info panel; screen 4 plays Access Controls,
which is why no role is asked. The files (`masthead`, `dcm-review`, `data-masking`, `iam`) and the
`RivePlayer` wrapper exist in the bytebase.com repository; the frontend is React, so the wrapper
ports as is, with `rive.wasm` self-hosted and the files lazy-loaded on the setup route only. The
files have no state-machine inputs and play autonomously; under reduced-motion the panel shows one
frame. The rocket illustration stays on sign-in and sign-up. Rejected for now: a constant stage (the
panel teaches nothing) and a live preview of the product being configured (strongest teaching,
highest drift; planned for screen 3 later).

**D13 · Instrumentation.** Each Continue emits `workspace setup screen continued` with the
screen number and the answers so far (goal, project named, source, own engine and host class, team
size, invite count). Each connection test emits `instance connection test` with engine, result,
cause and attempt number. `workspace setup finish clicked` fires before the backend calls on
screen 3, so a closed tab cannot lose the finish. `workspace setup submitted` gains
`result=failed` with `failure_stage` (project, sample, invite) and `sample_succeeded`. The last
button emits `first action opened`. The weekly exporter treats the backend signature (project and
sample instance, or a successful `CreateInstance`, within three minutes of the page-entered event)
as the truth for "finished", whatever the events say.

**D14 · The gate and the drafts.** The gate is unchanged: first login, sole admin, mandatory on
Cloud. Answers from screens 1 and 2 and the own-instance draft are held locally until screen 3
completes, so Back, refresh and a re-login resume with nothing retyped. Every disabled Continue
says why, in words, beside the button; two of the 42 leavers stalled on a disabled button with no
explanation.

## States

| Screen | State | What is on screen | Continue |
|---|---|---|---|
| 1 Workspace | default | name and workspace name prefilled; hero animation | enabled; "Continue needs a workspace name" if cleared |
| 2 Goal | none chosen | two cards, hero animation | disabled |
| 2 Goal | chosen | card pressed, panel swaps to the goal's animation | enabled |
| 3 Project | no name | project field empty; panel shows "New project" placeholder | disabled, "Continue needs a project name" |
| 3 Sample | named | sample card preselected; diagram of instance, two databases, project | enabled, "Continue with sample" |
| 3 Own · idle | form shown | engine, environment, essentials, authentication, Advanced collapsed, egress address; layout 30/70; panel = info facts + checklist | disabled, "No skip: pass the test or use the sample" |
| 3 Own · unreachable host | localhost, private range or internal hostname | amber notice naming the address and the three ways out; test button disabled | disabled |
| 3 Own · testing | after Test connection | checklist rows fill in: host reachable, credentials accepted, databases discovered, assigned to project | disabled |
| 3 Own · failed | credentials, timeout, DNS, TLS, unknown | red box: cause, the fact the fix needs, Test again, Use the sample for now, the engine's guide | disabled |
| 3 Own · connected | test passed | "Connected · N databases found · assigned to <project>" | enabled |
| 4 Team | size unset | five size options, chips input, one line about membership; footer shows provisioning progress | disabled, "Pick a team size to finish" |
| 4 Team | size set | same; "Finish without inviting" or "Send N invites and finish" | enabled |
| 4 Team | sample still preparing | footer "Preparing hr_test and hr_prod in <project>..."; Finish waits | disabled until ready |
| Finish | ready | one button: open the goal's first action | n/a |
| Finish | sample failed | "Try the sample again" and "Connect my own instance instead" | n/a |

Narrow viewports (below 1024px): the left panel hides on every screen, as the sign-in shell does
today; the own-instance info panel falls back to its overlay sheet.

## Scope

| File | Change |
|---|---|
| `frontend/src/routes/auth/WorkspaceSetupPage.tsx` | Rewrite as four screens with local draft state, per-screen gating messages, background sample provisioning, and the first-action hand-off. |
| `frontend/src/routes/auth/WorkspaceSetupQuestionnaireStep.tsx` | Retire; the goal moves to screen 2 and the team question is removed. |
| `frontend/src/app/layouts/SplashLayout.tsx` | Setup variant: a panel slot instead of the fixed illustration, and a 30/70 mode while the own-instance form is open. |
| `frontend/src/components/workspace-setup/` (new) | Screen components, the Rive panel (port of bytebase.com `RivePlayer.tsx` with self-hosted `rive.wasm` and the four files), the topology and checklist panel, the chips input. |
| `frontend/src/components/instance/InstanceFormBody.tsx`, `DataSourceForm.tsx`, `InstanceFormContext.tsx`, `CreateInstanceView.tsx` | A wizard layout mode: compact density, the advanced sections under one disclosure, the info panel rendered into a slot. No field changes. |
| `frontend/src/components/instance/InfoPanel.tsx` | Accept a docked target outside the form so the wizard's left panel can host it. |
| `frontend/src/modules/workspace-setup-guide/` | Take goal, team size and source from the wizard; mark "explore database" done on arrival; show the team step only above "Just me". |
| `frontend/src/locales/en-US.json` (and the other locales) | New strings for the four screens; retire `setup-scenario.mark-sensitive-data` and `setup-scenario.workspace-usage`. |
| `frontend/src/stores/app/workspace.ts`, `frontend/src/app/router/guard.ts` | Unchanged gate; the guard's "setup unfinished" flag now clears only when screen 3 completes. |
| `backend/api/v1/actuator_service.go` | SaaS only: expose the Cloud egress addresses (from the environment) so the form can show them inline. |

Tests to write: vitest for each screen's validation and gating message, for the host classifier
(localhost, private ranges, internal hostnames, public) and for the chips parser; the instance-form
contract test (wizard-mode labels equal page labels); an e2e spec under `frontend/tests/e2e` that
walks the sample path, the own path through a failed and then a passing test, the team screen with
chips and with "Finish without inviting", and asserts the first action opens; and event tests that
each Continue, the finish click and the submitted result carry the properties in D13.

## Not in this PR

The implementation, which follows separately. The landing page and the guide bar beyond the
hand-off in D11; a project home page is its own design. The live-preview panel for screen 3 (the
alternative kept for later). Provider presets for the own-instance form (Neon pooler parameter,
RDS IAM, Cloud SQL), which come from the one-click instance research. Persisting team size on the
workspace; it travels as an event property and in the guide's local state until a consumer needs
it server-side. The weekly exporter change that reads "finished" from the backend signature lives
in the hub repository.

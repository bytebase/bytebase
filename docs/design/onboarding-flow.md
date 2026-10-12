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
console's analytics went silent after the wizard mounted; whether a workspace is set up has to
come from the backend.

## Given

Two existing contracts bound the design. Neither changes here.

**The sample is ADR-0004's Sample Project Instance.** One per workspace, prepared for one project
by a single-use entitlement: one PostgreSQL database (`bb_sample_<id>`) on Bytebase's shared Cloud
SQL instance, seeded with the employee dataset, available for seven days.
`PrepareSampleProjectInstance` is synchronous: it returns once the database is seeded, discovered
and assigned to the project, within a three-minute bound. A repeated call for the same project
returns the same instance, so a retry is safe.

**A new workspace is on the free plan.** Changes with SQL review and rollout, the SQL Editor and
query history work there. Data masking and custom approval are Enterprise, so nothing the wizard
promises can depend on either.

## Principle

> **A screen asks only for what the next screen uses by name, and the one thing Bytebase cannot
> work without is never skippable.** Every answer is spent on the following screen: the workspace
> name is what invitees join, the goal decides what the database step prepares and what opens at
> the end, the project is the container the databases go into, and the team size keys the guide.
> Nothing is collected for its own sake except one qualification question, labeled as such. The
> database step is a gate: a sample that is verified ready, or an own instance that passed its
> test and yielded a database, with the facts a failing test needs on the screen where it fails.

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
with a one-line finish line under the card: a change that passed SQL review and rolled out to the
database, and a query result from it. "Mark sensitive data" is removed as a goal: masking guards
querying rather than being a destination, and as an Enterprise feature it led its choosers into a
gate (1 of 9 acted). Both remaining finish lines are reachable on the free plan.

**D4 · The project is named where the databases are added.** Screen 3 opens with the project name
(required, empty with example placeholder; the resource id derived from it, editable, collapsed)
and one line of explanation, then "Databases for <project>" with the two cards. The project exists
to hold databases, so naming it beside them teaches by adjacency and removes the forward reference
("the databases you add next...") that the workspace screen needed. Rejected: prefilling the name,
which removes the one keystroke burst and the lesson with it; and keeping it on screen 1.

**D5 · The database step is a gate with two cards.** "Continue with the sample database" is the
primary, preselected card: the Sample Project Instance, one PostgreSQL database with the employee
data, hosted by Bytebase Cloud, assigned to the project, available for seven days. It is offered
when the server reports the sample available and unused; otherwise the own-instance card is the
only path. "Connect my own instance" must pass Test connection before Continue enables, and a
passing result belongs to the values it tested: editing a connection field clears it. There is no
skip and no "Continue anyway". 93 of 100 finishers keep the sample on and it has not failed in 143 attempts
since Sep 8; the 7 who turn it off are the own-database path. Rejected: the checkbox, and the skip.

**D6 · The own-instance path renders the real instance form, in sync by construction.** The
wizard mounts the page's own `InstanceFormBody` and `DataSourceForm` under `InstanceFormContext`,
never a copy and never a list of fields, so a field or control added to the Instances page appears
in the wizard without anyone remembering to add it. The wizard contributes layout only: the
compact field density the form already has, the existing hide-advanced flag turned into one
"Advanced" disclosure (SSL/TLS, SSH tunnel, extra connection parameters, read-only replica), and
where the info panel goes. Engine comes first because it decides which fields exist; environment
defaults to Test; authentication offers whatever the page offers for that engine. A contract test
renders both and asserts that the set of field labels
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

**D8 · Reachability facts before the test, recovery at the failure.** Before a test runs, the
wizard classifies the address Bytebase Cloud itself would dial: the database host, or the SSH host
once a tunnel is configured, because the database host is then resolved from the bastion. When that
address is localhost, a private range (10/8, 172.16/12, 192.168/16) or an internal hostname, it is
named as unreachable from Bytebase Cloud with the three ways out (public endpoint, SSH tunnel under
Advanced, the sample for now), and the test button stays disabled until the address changes or a
tunnel is configured. The Cloud egress address is shown inline with a copy action, next to the
allowlist instruction, instead of a "Learn more" link. A failed test names the cause (credentials,
timeout, DNS, TLS, unknown), gives the one fact the fix needs, and offers "Test again" and "Use the
sample for now" in place. Rejected: the current static banner, and the raw driver message as the
whole explanation.

**D9 · Screen 3's Continue is the commit, and it waits for the database.** Until then nothing
exists on the server. Continue creates the project, then either prepares the sample or creates the
tested instance in that project, and the button shows progress until a database exists. The sample
request returns with the database ready (Given). `CreateInstance` does not: it returns the instance
even when its first sync fails or finds nothing, so on that path the commit also requires at least
one discovered database in the project, and otherwise stays on screen 3 saying the instance
connected but no database was found, with "Sync again" and the sample card. Passing screen 3
therefore means a database exists on both paths. A failure stays on screen 3 with the error, "Try again" and the other card, where today
the wizard reports a finish even when the sample request fails and offers no retry. After the
commit the project and the source are fixed, because the project exists and the sample entitlement
is spent on it: the project name is read-only on a retry, and screen 4 has no Back. Rejected:
issuing the sample request in the background and letting Finish wait for it. That adds a pending
and a failed state to screen 4 and to Finish in order to hide a wait that today's finishers
complete, with the whole wizard around it, in a median of one minute.

**D10 · Team: size required, invites optional, no role.** "How many people will use Bytebase?"
with five options (Just me, 2–10, 11–50, 51–200, 200+) is one required click, the flow's only
qualification question. It keys the guide's team step (only above "Just me"), the plan hint, and
lead scoring; team accounts do first work at 34% against 22%. Emails are chips: type one, press
Enter, type the next; a pasted list splits; an invalid entry is shown struck through and not sent.
There is no role picker. An earlier draft offered a "Role in <project>" selector, which names
Bytebase's project roles to a user ten seconds into the product. Invitees are added as Workspace
Member and as Project Developer on the new project, one sentence says they can query and propose
changes there, and roles are changed later in Members, where the names mean something. Greptile asks a default role (Admin or Member); Linear, Neon and
Metabase ask for emails only. Rejected: the "Just me / My team" question (it only decided whether
this screen appears, and the screen is now always offered and one click to pass), the role
selector, and the free-text email box.

**D11 · The last button opens the goal's first action.** For the schema-change goal, the
new-change dialog targeting the sample database; for the query goal, the SQL Editor on it with a
generated SELECT. On the own-instance path the same action opens in the project and the user picks
the database. No landing page in between. The landing page and the guide bar are separate work; the
wizard passes them goal, team size and source so the guide starts at the goal step with the
automatic "explore database" step already done. Rejected: landing on the project databases page,
where 73 of 100 finishers idle.

**D12 · The left panel is a story per screen, from bytebase.com's Rive files.** Screen 1 plays the
homepage hero (requests flowing into review); screen 2 swaps to the chosen goal's feature animation
(Change Review, or Dynamic Data Masking as the picture of a guardrail; the card's finish line does
not depend on it) the moment a card is pressed; screen 3 draws a diagram from what the user typed
(the sample instance and its database flowing into the project name) or, for an own instance, the connection checklist with the info panel; screen 4 plays Access Controls,
which is why no role is asked. The files (`masthead`, `dcm-review`, `data-masking`, `iam`) and the
`RivePlayer` wrapper exist in the bytebase.com repository; the frontend is React, so the wrapper
ports as is, with `rive.wasm` self-hosted and the files lazy-loaded on the setup route only. The
files have no state-machine inputs and play autonomously; under reduced-motion the panel shows one
frame. The rocket illustration stays on sign-in and sign-up. Rejected for now: a constant stage (the
panel teaches nothing) and a live preview of the product being configured (strongest teaching,
highest drift; planned for screen 3 later).

**D13 · Instrumentation.** Each Continue emits `workspace setup screen continued` on the click,
before any backend call, with the screen number and the answers so far (goal, project named,
source, own engine and address class, team size, invite count). No event carries a host, a
username, an email address or a credential. Each connection test emits `instance connection test`
with engine, result, cause and attempt number. `workspace setup submitted` keeps its meaning, the
workspace has a project and a database, so it fires when screen 3's commit returns, and it gains
`result=failed` with `failure_stage` (project, sample, instance). The last button emits
`first action opened`. No client event can be the record, though: the 17 lost finishes lost every
custom event after the wizard mounted, including one that fires before the backend calls, and 13
accounts since Sep 11 were never tracked at all. The weekly exporter therefore reads "set up" from
the backend alone: the workspace has a project with a database, timed from the workspace's
creation.

**D14 · Gate, draft and resume.** The gate is unchanged: first login, sole admin, mandatory on
Cloud. Until the commit, the wizard's answers live in a local draft, so Back, a refresh and a
re-login in the same browser resume where the user left. The draft holds the answers and the
non-secret connection fields only. Passwords, tokens, private keys and certificates, the SSH
tunnel's included, stay in memory and are re-entered after a reload; the wizard has no safe place
to keep them. The draft also records the commit, and the flag that sends a Cloud account back to
the wizard clears on Finish, not at the commit. A return after the commit therefore resumes on
screen 4, so reloading skips neither the team-size question nor the hand-off to the first action.
Every disabled Continue says why, in words, beside the button; two of the 42 leavers stalled on a
disabled button with no explanation.

**D15 · Self-hosted keeps its differences.** The same four screens serve self-hosted, where setup
is offered once and screen 3 keeps "I'll do this later". That action leaves the wizard for the
workspace landing as it does today, with no screen 4 and no first action, since nothing was
created. D8's classification and the egress address do not apply there, because Bytebase dials
from the customer's own network.

## States

| Screen | State | What is on screen | Continue |
|---|---|---|---|
| 1 Workspace | default | name and workspace name prefilled; hero animation | enabled; "Continue needs a workspace name" if cleared |
| 2 Goal | none chosen | two cards, hero animation | disabled |
| 2 Goal | chosen | card pressed, panel swaps to the goal's animation | enabled |
| 3 Project | no name | project field empty; panel shows "New project" placeholder | disabled, "Continue needs a project name" |
| 3 Sample | named | sample card preselected; diagram of the sample instance, its database, the project | enabled, "Continue with sample" |
| 3 Own · idle | form shown | engine, environment, essentials, authentication, Advanced collapsed, egress address; layout 30/70; panel = info facts + checklist | disabled, "No skip: pass the test or use the sample" |
| 3 Own · unreachable address | the dialed address is localhost, a private range or an internal hostname | amber notice naming the address and the three ways out; test button disabled until the address changes or an SSH tunnel is configured | disabled |
| 3 Own · failed | credentials, timeout, DNS, TLS, unknown | red box: cause, the fact the fix needs, Test again, Use the sample for now, the engine's guide | disabled |
| 3 Own · connected | test passed | "Connected"; the checklist's host and credentials rows are done, the databases rows wait for Continue | enabled |
| 3 · committing | Continue pressed | the button shows progress while the project and the database are created; the form is read-only | n/a |
| 3 · commit failed | project, sample or instance creation failed | the error, Try again, and the other card; the project name is read-only once the project exists | enabled for a retry |
| 3 · skipped | "I'll do this later", self-hosted only | leaves for the workspace landing; nothing is created | n/a |
| 3 Own · no database | the instance was created but its first sync failed or found nothing | "Connected, but no database was found", the sync error if there is one, Sync again, Use the sample for now | disabled |
| 4 Team | size unset | five size options, chips input, one line about membership; no Back | disabled, "Pick a team size to finish" |
| 4 Team | size set | same; "Finish without inviting" or "Send N invites and finish" | enabled |
| Finish | after screen 4 | one button: open the goal's first action | n/a |

A reload before the commit returns to the same screen with the draft; a reload after it returns
to screen 4.

Narrow viewports (below 1024px): the left panel hides on every screen, as the sign-in shell does
today; the own-instance info panel falls back to its overlay sheet.

## Scope

| File | Change |
|---|---|
| `frontend/src/routes/auth/WorkspaceSetupPage.tsx` | Rewrite as four screens with the local draft, per-screen gating messages, the commit on screen 3, and the first-action hand-off. |
| `frontend/src/routes/auth/WorkspaceSetupQuestionnaireStep.tsx` | Retire; the goal moves to screen 2 and the team question is removed. |
| `frontend/src/app/layouts/SplashLayout.tsx` | Setup variant: a panel slot instead of the fixed illustration, and a 30/70 mode while the own-instance form is open. |
| `frontend/src/components/workspace-setup/` (new) | Screen components, the Rive panel (port of bytebase.com `RivePlayer.tsx` with self-hosted `rive.wasm` and the four files), the topology and checklist panel, the chips input. |
| `frontend/src/components/instance/InstanceFormBody.tsx`, `DataSourceForm.tsx`, `InstanceFormContext.tsx`, `CreateInstanceView.tsx` | A wizard layout mode: compact density, the advanced sections under one disclosure, the info panel rendered into a slot. No field changes. |
| `frontend/src/components/instance/InfoPanel.tsx` | Accept a docked target outside the form so the wizard's left panel can host it. |
| `frontend/src/modules/workspace-setup-guide/` | Take goal, team size and source from the wizard; mark "explore database" done on arrival; show the team step only above "Just me". |
| `frontend/src/locales/en-US.json` (and the other locales) | New strings for the four screens; retire `setup-scenario.mark-sensitive-data` and `setup-scenario.workspace-usage`. |
| `frontend/src/modules/workspace-setup-guide/setup.ts`, `frontend/src/app/router/guard.ts`, `frontend/src/stores/app/auth.ts` | Gate unchanged. The setup-finished flag is saved on Finish instead of on submit, and the draft's commit marker resumes a returning account on screen 4. |
| `proto/v1/v1/actuator_service.proto`, `backend/api/v1/actuator_service.go` | SaaS only: expose the Cloud egress addresses (from the environment) so the form can show them inline. |

Tests to write: vitest for each screen's validation and gating message, for the address classifier
(localhost, private ranges, internal hostnames, public; the SSH host once a tunnel is configured;
off on self-hosted) and for the chips parser; a draft test asserting that no credential field
reaches storage; the instance-form contract test (wizard-mode labels equal page labels); an e2e
spec under `frontend/tests/e2e` that walks the sample path, the own path through a failed and then
a passing test and through an instance that yields no database, a reload on screen 4 returning to
screen 4, the self-hosted skip, the team screen with chips and with
"Finish without inviting", and asserts the first action opens; and event tests that each Continue
and the submitted result carry the properties in D13.

## Not in this PR

The implementation, which follows separately. The landing page and the guide bar beyond the
hand-off in D11; a project home page is its own design. The live-preview panel for screen 3 (the
alternative kept for later). Provider presets for the own-instance form (Neon pooler parameter,
RDS IAM, Cloud SQL), which come from the one-click instance research. Persisting team size on the
workspace; it travels as an event property and in the guide's local state until a consumer needs
it server-side. The weekly exporter change that reads "set up" from the backend lives
in the hub repository. Any change to the sample itself, such as a test and prod pair, which would
reopen ADR-0004. Making setup mandatory on self-hosted.

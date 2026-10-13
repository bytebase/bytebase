# Onboarding flow, the first four screens

Status: proposal · 2026-10-12 (revised 2026-10-13) · lands after the project home, [#21572]

A new Cloud workspace opens on a setup wizard that asks for a goal and "who will use Bytebase with
you", then for your name, a workspace name, a first project and a sample-database checkbox, and
lands on the project's databases page. Since setup became mandatory on Sep 20, 89% of signups see it
and 70% of those finish, so 70% of all signups end with a database. Only 24% go on to do first work.
The wizard collects six things, explains none of them, and ends on a page with nothing to do. The
rule this doc lands on: **a screen asks only for what a later step uses and explains it where it
asks, and setup always ends with a project, a decision about its database, and one next step.** Four
screens: workspace, goal, project with its database, team. The project is explained where its
database is chosen, everything is created in one step at Finish, and connecting your own instance
happens after setup, not inside the wizard. Finish lands on the project home on both paths, so this
design depends on the project home ([#21572]) and merges after it. Frontend only.

The interactive mockups, the alternatives this was chosen from, and the funnel evidence are in the
design page kept outside the repo
(`~/Development/design-and-test-docs/2026-09-16-onboarding-flow-designs.html`, rev 6). The
numbers below come from the "Onboarding Wizard Funnel to Oct 10" edition: 396 Cloud signups from
Aug 25 to Oct 10 joined to the metadata database, plus 318 first-session analyst verdicts.

## Problem

The wizard is `frontend/src/routes/auth/WorkspaceSetupPage.tsx` with
`WorkspaceSetupQuestionnaireStep.tsx` as its first step, shown by the router guard while
`isFirstLogin` holds and the account is the sole IAM member, and mandatory on Cloud since Sep 20.
Three things are wrong with what it asks and in what order, and a fourth with where it lands.

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

**The database choice is a checkbox.** Until Sep 20 every step could be skipped, leaving an empty
workspace whose Databases and Instances pages said "No data". Setup is now mandatory, which fixed
the instance step (24% to 70% of signups with a database), but the choice that produced that
result is one unexplained checkbox, and the wizard reports a finish even when the sample request
fails. The 7 of 100 finishers who turn the sample off go to the instance form, where 62
connection tests failed and 21 passed since Sep 20.

**The landing is a dead end.** Of 100 finishers in the mandatory state, 73 did nothing afterwards;
their median time in the product is under two minutes. The analysts' recurring description: wizard
finished in about twenty seconds with the sample on, the sample opened, then idling on a page with
three prompts and no dominant one, Issues opening on "No data", the SQL Editor opening blank. The
wizard does not fix this by itself. The project home does, and this design lands on it (D8).

Two measurement defects shape the design too. Step 1 to step 2 has never emitted an event, so
leavers cannot be placed on a screen. And 17 finishes produced no finish event because the
console's analytics went silent after the wizard mounted; whether a workspace is set up has to
come from the backend.

## Given

Three existing contracts bound the design. None changes here.

**The sample is ADR-0004's Sample Project Instance.** One per workspace, prepared for one project
by a single-use entitlement: one PostgreSQL database (`bb_sample_<id>`) on Bytebase's shared Cloud
SQL instance, seeded with the employee dataset, available for seven days.
`PrepareSampleProjectInstance` is synchronous: it returns once the database is seeded, discovered
and assigned to the project, within a three-minute bound. A repeated call for the same project
returns the same instance, so a retry is safe. Self-hosted answers the same request with a
different sample (`backend/component/sample/selfhost`): one database on the server's own embedded
PostgreSQL, with no expiry, offered only when the server runs on embedded PostgreSQL.

**Invitation emails need a mail setting.** Adding a member to the workspace sends an invitation
only when the workspace has one. Cloud workspaces are created with one; a new self-hosted server
has none until an admin configures it.

**A new workspace is on the free plan.** Changes with SQL review and rollout, the SQL Editor and
query history work there. Data masking and custom approval are Enterprise, so nothing the wizard
promises can depend on either.

## Principle

> **A screen asks only for what a later step uses and explains it where it asks, and setup always
> ends with a project, a decision about its database, and one next step.** Every answer is spent
> downstream: the workspace name is what invitees join, the goal tells the guide what to suggest
> first, the project is the container the database goes into, and the team size keys the guide.
> Nothing is collected for its own sake except one qualification question, labeled as such. The
> wizard does not carry the connection form: connecting an instance is a task with its own page,
> its own failure modes and its own recovery. And it does not build a landing of its own: it ends
> on the project home, which shows the next step for the project as setup left it.

## Decisions

**D1 · Four screens, in this order: workspace, goal, project with its database, team.** Identity
first because it is known and cheap, and every later screen names the workspace it confirms. Goal
before database because it is what the user came for, and it tells the guide what to suggest first.
Database before team because invitees should find something when they click the invite; Linear and
Neon invite early because their product works before any setup, ours works after a database. Team
last, always offered, skippable. Rejected: today's order (goal and team first, then everything else
on one screen), and team as the second screen, which invites people into an empty workspace.

**D2 · Screen 1 is a confirmation, not a form.** Your name (optional, prefilled from the identity
provider or the email local part, left empty for alias-like local parts such as admin or info) and
the workspace name (required, prefilled from the email domain, "Peter's workspace" for public mail
domains). Continue is enabled on arrival, so the first click is the only click. Nine of the 42
mandatory-state leavers never clicked anything, consistent with an OAuth redirect returning to a
tab nobody is looking at; a screen with nothing to decide is the cheapest first screen.

**D3 · Two goals, and a quieter third answer.** "Review and roll out a schema change" and "Query
data with guardrails" are the two cards, each with a one-line finish line under it: a change that
passed SQL review and rolled out to the database, and a query result from it. Under them sits a
lighter third answer, "Not sure yet, just exploring", which commits to neither; the guide then
suggests a query first, the quickest first result. Every product in the study that asks an intent
question lets the user decline it: PostHog has "Continue without choosing a goal", Metabase "I'm not
sure yet", and n8n and Greptile make the question optional. Here, 3 of 20 finishers left the goal
blank while it was optional, 2 of the leavers since it became required stalled on it with Continue
disabled, and the forced answer predicts the first action only weakly. An honest "not sure" also
makes the other two answers worth more as signals. It is drawn lighter than the cards so it does not
read as their peer, and its share is a number to watch. "Mark sensitive data" is removed as a goal:
masking guards querying rather than being a destination, and as an Enterprise feature it led its
choosers into a gate (1 of 9 acted). Every finish line is reachable on the free plan. Rejected: a
third equal card, and keeping the forced choice between two.

**D4 · The project is named where its database is chosen.** Screen 3 opens with the project name
(required, empty with example placeholder; the resource id derived from it, editable, collapsed)
and one line of explanation, then "Database for <project>" with the two choices. The project exists
to hold databases, so naming it beside them teaches by adjacency and removes the forward reference
("the databases you add next...") that the workspace screen needed. Rejected: prefilling the name,
which removes the one keystroke burst and the lesson with it; and keeping it on screen 1.

**D5 · Screen 3 offers two choices and no form.** "Start with the sample database" is the primary,
preselected choice: the Sample Project Instance, one PostgreSQL database with the employee data,
hosted by Bytebase Cloud, assigned to the project, available for seven days (Cloud's wording; D12
has self-hosted's). It is offered when the server reports the sample available and unused; otherwise
the other choice is the only one. "Connect my own instance after setup" defers the connection: setup
finishes with an empty project, and the project home's next step is to connect an instance (D8). One
sentence under it says what an instance is. The wizard holds no connection form, no test and no
credentials. 93 of 100 finishers keep the sample on, and it has not failed in 143 attempts since Sep
8. Rejected: the instance form inside the wizard, as an earlier draft had it. It made screen 3 the
heaviest screen of the flow, put a form laid out for a wide page into half a screen, and pulled
connection testing, failure recovery and credential handling into setup. Also rejected: the
checkbox, and a skip.

**D6 · Finish is the one commit, so Back works on every screen.** Continue on screens 1 to 3 only
records answers; nothing exists on the server until Finish on screen 4. Finish saves the name and
workspace name, creates the project, prepares the sample when it was chosen, and adds the invitees
to the project and then to the workspace, and its button shows progress until the last call returns.
The sample request returns with the database ready (Given), so invitees arrive to a project that has
one; on the own-instance path they arrive before the instance is connected (D7). Adding them is the
part Finish can verify; the server then sends their invitation emails in the background, as it does
from the Members page. The workspace write is what triggers those emails, so it goes last: nobody is
invited to a project they cannot open. A failure stays on screen 4 naming the step that failed, with
"Try again", which runs Finish again without creating the project twice (D11); when the sample is
what failed it also offers "Finish without the sample", which finishes as the own-instance choice
does. Today the wizard reports a finish even when the sample request fails and offers no retry.
Rejected: committing on screen 3, as an earlier draft had it. It made the team screen a point of no
return with no Back, and needed a resume rule for an account that reloaded after the commit.

**D7 · Team: size required, invites optional, no role.** "How many people will use Bytebase?" with
five options (Just me, 2–10, 11–50, 51–200, 200+) is one required click, the flow's only
qualification question. It keys the guide's team step (only above "Just me"), the plan hint, and
lead scoring; team accounts do first work at 34% against 22%. The invite field appears once a size
above "Just me" is chosen, so the two answers cannot disagree and a solo account, the majority, sees
one question and Finish. Emails are chips: type one, press Enter, type the next; a pasted list
splits; an invalid entry is shown struck through and not sent. There is no role picker. An earlier
draft offered a "Role in <project>" selector, which names Bytebase's project roles to a user ten
seconds into the product. Invitees are added as Workspace Member, and on the new project as Project
Developer and SQL Editor Read User, which together let them propose changes and run read-only
queries. One sentence says so, and roles are changed later in Members, where the names mean
something. On the own-instance path that sentence also says the project's databases appear once the
instance is connected, because those invitees arrive first. Holding their invitations until then
would need somewhere to hold them, which the 7 of 100 finishers on that path do not justify.
Greptile asks a default role (Admin or Member); Linear, Neon and Metabase ask for emails only.
Rejected: the "Just me / My team" question (it only decided whether this screen appears, and the
screen is now always offered and one click to pass), the role selector, and the free-text email box.

**D8 · Finish lands on the project home, on both paths.** The project home ([#21572]) makes the
project root a page of its own: a next-step band chosen from the project's state, the user's open
issues, and the project's databases. Finish navigates there whichever database choice was made. With
the sample, the project has a database and no change yet, so the next step shown is "Change a
database". With "Connect my own instance after setup", the project is empty, so the next step is
"Connect an instance", which opens the instance form for that project; the form still offers the
sample, so a connection that will not work is one step from a working database. An account that
invited nobody also sees "Manage members" as an optional step. The wizard builds no landing of its
own. An earlier draft opened the goal's first action straight from Finish and highlighted "Connect
instance" on the databases page; both are withdrawn, because the project home is where the next step
is chosen. It chooses from the project's state and does not read the goal, so the goal reaches the
guide bar only, as it does today; making the home's next step follow the goal would be a change to
that page. This design merges after [#21572]: 73 of 100 finishers idle on the project's databases
page, and a new wizard should not ship pointing at it.

**D9 · The left panel is a story per screen, from bytebase.com's Rive files.** Screen 1 plays the
homepage hero (requests flowing into review); screen 2 swaps to the chosen goal's feature animation
(Change Review, or Dynamic Data Masking as the picture of a guardrail; the card's finish line does
not depend on it) the moment a card is pressed, and stays on the hero for "just exploring"; screen 3
draws a diagram from what the user typed
(the sample instance and its database flowing into the project name, or the user's own instance
joining after setup); screen 4 plays Access Controls, which is why no role is asked. The files
(`masthead`, `dcm-review`, `data-masking`, `iam`) and the `RivePlayer` wrapper exist in the
bytebase.com repository; the frontend is React, so the wrapper ports as is, with `rive.wasm`
self-hosted and the files lazy-loaded on the setup route only. The files have no state-machine
inputs and play autonomously; under reduced-motion the panel shows one frame. The rocket
illustration stays on sign-in and sign-up. Rejected for now: a constant stage (the panel teaches
nothing) and a live preview of the product being configured (strongest teaching, highest drift;
planned for screen 3 later).

**D10 · Instrumentation.** Each Continue emits `workspace setup screen continued` on the click, with
the screen number and the answers so far (goal, project named, source). No event carries a name, an
email address or a project name. `workspace setup submitted` still fires when Finish succeeds, now
carrying the goal, the source, the team size and the invite count, and gains `result=failed` with
`failure_stage` (names, project, sample, invitees). No client event can be the record, though: the
17 lost finishes lost every custom event after the wizard mounted, and 13 accounts since Sep 11 were
never tracked at all. The record stays with the backend, as the Cloud usage dashboard's exporter
already reads it: "finished setup" from the workspace's first project, and "connected an instance"
from its first instance, sample or own. An account that defers its instance therefore shows up as
finished but not yet connected, with no new backend work.

**D11 · Gate, draft and resume.** The gate is unchanged: first login, sole admin, mandatory on
Cloud, and the flag that sends an unfinished Cloud account back to the wizard clears when Finish
succeeds, as it does on submit today. Until then the answers live in a local draft, so Back, a
reload and a re-login in the same browser return to the same screen with nothing retyped. The draft
also records that the project was created, the one step of Finish that cannot be repeated, so a
retry, after a reload too, goes past it instead of failing on a project that already exists. From
then on screen 3 shows the project's name and ID read-only, and the database choice too once the
sample has been prepared: both exist, and are changed in the product, not in the wizard. Every other
step is an overwrite or safe to repeat (saving the names, preparing the sample for the same project,
writing the invitees to the project and then the workspace), so each retry runs them again from the
answers on screen. A checkpoint therefore never outlives the answers it was made with: an invitee
removed after a failed attempt is removed from the project as well. The wizard collects no
credentials, so the draft holds none. Every disabled button says why, in words, beside it; two of
the 42 leavers stalled on a disabled button with no explanation.

**D12 · Self-hosted keeps its differences.** The same four screens serve self-hosted, where setup is
offered once and screen 3 keeps "I'll do this later". That action saves the two names from screen 1
and leaves for the workspace landing as it does today. No project is created, so nothing else is
left behind. Two lines of copy follow the deployment (Given) instead of repeating Cloud's. The
sample choice says the database runs on this server and names no expiry, as the product's existing
self-hosted sample strings do. And where the workspace has no mail setting, screen 4 says teammates
are added without an email being sent, and its button reads "Add N teammates and finish".

## States

| Screen | State | What is on screen | Button |
|---|---|---|---|
| 1 Workspace | default | name and workspace name prefilled; hero animation | Continue enabled; "Continue needs a workspace name" if cleared |
| 2 Goal | none chosen | two cards and the lighter "Not sure yet, just exploring"; hero animation | Continue disabled |
| 2 Goal | a card chosen | card pressed, panel swaps to the goal's animation | Continue enabled |
| 2 Goal | just exploring | the third answer pressed, panel stays on the hero | Continue enabled |
| 3 Project | no name | project field empty; the sample choice preselected | Continue disabled, "Continue needs a project name" |
| 3 Sample | named, sample chosen | diagram of the sample instance, its database, the project | Continue enabled |
| 3 Own | named, own instance chosen | one line on what happens after setup; diagram of the user's instance joining the project later | Continue enabled |
| 3 · no sample | the server reports the sample unavailable or used | only the own-instance choice, with the reason | Continue enabled once named |
| 3 · skipped | "I'll do this later", self-hosted only | saves the screen-1 names and leaves for the workspace landing; no project is created | n/a |
| 3 · project exists | Back after a Finish attempt that created the project | project name and ID read-only, with a line saying the project exists and is renamed in its settings; the database choice read-only once the sample is prepared | Continue enabled |
| 4 Team | size unset | five size options, nothing else | Finish disabled, "Pick a team size to finish"; Back enabled |
| 4 Team | "Just me" | the size options only; no invite field | Finish enabled |
| 4 Team | a size above "Just me" | the chips input appears with one line about membership; the button reads "Finish" or "Send N invites and finish" | Finish enabled |
| 4 · finishing | Finish pressed | the button shows progress; the form is read-only | n/a |
| 4 · finish failed | a step of the commit failed | the error naming the step, Try again; "Finish without the sample" when the sample failed | Try again enabled |

After Finish the wizard is gone and the account is on the project home (D8). A reload before Finish
returns to the same screen with the draft.

## Scope

| File | Change |
|---|---|
| `frontend/src/routes/auth/WorkspaceSetupPage.tsx` | Rewrite as four screens with the local draft, per-screen gating messages, and the single commit at Finish. Finish navigates to the project home, the redirect [#21572] introduces. |
| `frontend/src/routes/auth/WorkspaceSetupQuestionnaireStep.tsx` | Retire; the goal moves to screen 2 and the team question is removed. |
| `frontend/src/app/layouts/SplashLayout.tsx` | Setup variant: a panel slot instead of the fixed illustration. |
| `frontend/src/routes/auth/workspace-setup/` (new) | Beside the route that is their only consumer: screen components, the Rive panel (port of bytebase.com `RivePlayer.tsx` with self-hosted `rive.wasm` and the four files), the diagram panel, the chips input. |
| `frontend/src/modules/workspace-setup-guide/` | Take goal, team size and source from the wizard; mark "explore database" done on arrival with the sample; start at "Connect instance" on the own-instance path; use the query journey for "just exploring"; show the team step only above "Just me". |
| `frontend/src/locales/en-US.json` (and the other locales) | New strings for the four screens; retire `setup-scenario.mark-sensitive-data` and `setup-scenario.workspace-usage`. |

No backend, proto or instance-form change.

Tests to write: vitest for each screen's validation and gating message, for the chips parser, and
for the draft restoring every screen; an e2e spec under `frontend/tests/e2e` that walks the sample
path onto the project home with "Change a database" as its next step, the own-instance path onto it
with "Connect an instance", Back from the team screen keeping every answer, a failed Finish retried
after the invite list changed, and the self-hosted skip with its sample and invite wording; and
event tests that each Continue and the submitted result carry the properties in D10.

## Not in this PR

The implementation, which follows separately, after the project home ([#21572]) is on main. The
project home itself, including whether its next step should follow the goal the wizard collected.
The guide bar beyond the answers the wizard hands it. Connection-failure recovery on the instance
page (naming a private address before the test, showing the Cloud egress address, cause-specific
fixes): that is where the 62 failed tests happen, and it is separate work on that page. The
live-preview panel for screen 3. Persisting team size on the workspace; it travels as an event
property and in the guide's local state until a consumer needs it server-side. Any change to the
sample itself, such as a test and prod pair, which would reopen ADR-0004. Making setup mandatory on
self-hosted. Enforcing mandatory setup across browsers: the flag is per-browser today, so an
unfinished account can still enter from another one, as 3 accounts have since Sep 20, and closing
that needs server-side state.

[#21572]: https://github.com/bytebase/bytebase/pull/21572

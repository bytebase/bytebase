# Onboarding flow, the first four screens

Status: implemented in this PR, with stills in place of D9's animations · 2026-10-12 (revised 2026-10-14) · builds on the project home, [#21572] (merged)

A new Cloud workspace opens on a setup wizard that asks for a goal and "who will use Bytebase with
you", then for your name, a workspace name, a first project and a sample-database checkbox. Until
Oct 13 it ended on the project's databases page; since the project home ([#21572]) it ends there.
Since setup became mandatory on Sep 20, 89% of signups see it and 70% of those finish, so 70% of all
signups end with a database. Only 24% go on to do first work. The wizard collects six things and
explains none of them, and the page it ended on while those numbers were measured had nothing to do.
The rule this doc lands on: **a screen asks only for what a later step uses and explains it where it
asks, and setup always ends with a project, a decision about its database, and one next step.** Four
screens: workspace, goal, project with its database, team. The project is explained where its
database is chosen, everything is created in one step at Finish, and connecting your own instance
happens after setup, not inside the wizard. Finish lands on the project home on both paths, and the
project home puts the goal's step first. Frontend only.

The interactive mockups, the alternatives this was chosen from, and the funnel evidence are in the
design page kept outside the repo
(`~/Development/design-and-test-docs/2026-09-16-onboarding-flow-designs.html`, rev 6). The
numbers below come from the "Onboarding Wizard Funnel to Oct 10" edition: 396 Cloud signups from
Aug 25 to Oct 10 joined to the metadata database, plus 318 first-session analyst verdicts.

## Problem

The wizard is `frontend/src/routes/auth/WorkspaceSetupPage.tsx` with
`WorkspaceSetupQuestionnaireStep.tsx` as its first step, shown by the router guard while
`isFirstLogin` holds and the account is the sole IAM member, and mandatory on Cloud since Sep 20.
Three things are wrong with what it asks and in what order, and a fourth with where it landed.

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

**The landing was a dead end.** Of 100 finishers in the mandatory state, 73 did nothing afterwards;
their median time in the product is under two minutes. The analysts' recurring description: wizard
finished in about twenty seconds with the sample on, the sample opened, then idling on a page with
three prompts and no dominant one, Issues opening on "No data", the SQL Editor opening blank. That
page was the project's databases page. The project home replaced it on Oct 13 with one "Next step"
band above the user's issues and the project's databases. The band lists its steps in a fixed order,
though, whatever the user said they came for, and the goal answer predicts first work only weakly
today (query choosers run a query 33% of the time, change choosers create an issue 19%) because it
changes the guide bar and nothing else. This design lands on the project home and gives it the goal
(D8).

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
> downstream: the workspace name is what invitees join, the goal decides which step the project home
> and the guide put first, the project is the container the database goes into, and the team size
> keys the guide. Nothing is collected for its own sake except one qualification question, labeled
> as such. The wizard does not carry the connection form: connecting an instance is a task with its
> own page, its own failure modes and its own recovery. And it does not build a landing of its own:
> it ends on the project home, which shows the next steps for the project as setup left it, the
> goal's first.

## Decisions

**D1 · Four screens, in this order: workspace, goal, project with its database, team.** Identity
first because it is known and cheap, and every later screen names the workspace it confirms. Goal
before database because it is what the user came for, and it decides which step the project home and
the guide put first. Database before team because invitees should find something when they click the
invite; Linear and Neon invite early because their product works before any setup, ours works after
a database. Team last and always offered: one required click for its size, invitations optional.
Rejected: today's order (goal and team first, then everything else on one screen), and team as the
second screen, which invites people into an empty workspace.

**D2 · Screen 1 is a confirmation, not a form.** Your name (optional, prefilled from the identity
provider or the email local part, left empty for local parts that name no person, such as admin,
it-support or a phone number) and the workspace name (required, prefilled from the email domain,
"Peter's workspace" for public mail domains, "My workspace" when the address names nobody either).
A public mail provider is recognized by its name, so its country domains count (yahoo.co.jp,
hotmail.co.uk). Continue is enabled on arrival, so the first click is the only click. Nine of the 42
mandatory-state leavers never clicked anything, consistent with an OAuth redirect returning to a
tab nobody is looking at; a screen with nothing to decide is the cheapest first screen.

**D3 · Two goals, and a quieter third answer.** "Plan, review and roll out a database change" (a
schema change or a data fix) and "Query data with guardrails" are the two cards, each with a
one-line finish line under it: a change that passed SQL review and rolled out to the database, and a
query result from it. Under them sits a lighter third answer, "Not sure yet, just exploring", which
commits to neither; the project home and the guide then put a query first, the quickest first
result. Every product in the study that asks an intent question lets the user decline it: PostHog
has "Continue without choosing a goal", Metabase "I'm not sure yet", and n8n and Greptile make the
question optional. Here, 3 of 20 finishers left the goal blank while it was optional, 2 of the
leavers since it became required stalled on it with Continue disabled, and the forced answer
predicts the first action only weakly. An honest "not sure" also makes the other two answers worth
more as signals. It is drawn lighter than the cards so it does not read as their peer, and its share
is a number to watch. "Mark sensitive data" is removed as a goal: masking guards querying rather
than being a destination, and as an Enterprise feature it led its choosers into a gate (1 of 9
acted). Every finish line is reachable on the free plan. Rejected: a third equal card, and keeping
the forced choice between two.

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
does. The failure is reported there and nowhere else: Finish's calls do not also raise the product's
error toast. One failure leaves screen 4: a project ID the server will not take goes back to the
field it is about (D11). Today the wizard reports a finish even when the sample request fails and
offers no retry.
Rejected: committing on screen 3, as an earlier draft had it. It made the team screen a point of no
return with no Back, and needed a resume rule for an account that reloaded after the commit.

**D7 · Team: size required, invites optional, no role.** "How many people will use Bytebase?" with
five options (Just me, 2–10, 11–50, 51–200, 200+) is one required click, the flow's only
qualification question. It is a radio group, in a row that wraps on narrow screens, not a segmented
control, which the UX contract keeps for view switches. It keys the guide's team step (only above
"Just me"), the plan hint, and lead scoring; team accounts do first work at 34% against 22%. The
invite field appears once a size above "Just me" is chosen, so "Just me" never comes with invitees
and a solo account, the majority, sees one question and Finish. The size is how many people will use
Bytebase, not how many are invited today, so the two numbers are not reconciled further. Emails are
chips: type one, press Enter, type the next; a pasted list splits at commas, semicolons and line
breaks, the full-width ones an input method types included, and repeats are dropped; a recipient
pasted from a mail client as `Ann Lee <ann@example.com>` is its address, not three entries; a pasted
list takes the place of the text selected in the field, and text typed around it stays an entry of
its own; an invalid entry, or the owner's own address, is shown struck through, is not sent and is
not counted on the button. Text still in the field when the button is pressed is entered first, and
if it is not an address that press stops there, so the struck entry is seen before anything is sent.
Inviting is Cloud's: an invitation is an email, and only Cloud has an email service at first run
(Given). On Cloud the field reads "Invite teammates" and the button "Send N invites and finish". On
self-hosted the same field reads "Add members": Finish adds those addresses as workspace members
with the same roles and sends nothing, the line under the field says so and that each person gets
access once they have an account with that address, and the button reads "Add N members and finish".
Calling it an invitation there would promise an email that never arrives. There is no role picker.
An earlier draft offered a "Role in <project>" selector, which names Bytebase's project roles to a
user ten seconds into the product. Invitees are added as Workspace Member, and on the new project as
Project Developer and SQL Editor Read User, which together let them propose changes and run
read-only queries. One sentence says so, and roles are changed later in Members, where the names
mean something. On the own-instance path that sentence also says the project's databases appear once
the instance is connected, because those invitees arrive first. Holding their invitations until then
would need somewhere to hold them, which the 7 of 100 finishers on that path do not justify.
Greptile asks a default role (Admin or Member); Linear, Neon and Metabase ask for emails only.
Rejected: the "Just me / My team" question (it only decided whether this screen appears, and the
screen is now always offered and one click to pass), the role selector, and the free-text email box.

**D8 · Finish lands on the project home, and the home puts the goal's step first.** The project home
([#21572], on main since Oct 13) makes the project root a page of its own: a "Next step" band chosen
from the project's state, above the user's open issues and the project's databases. Finish navigates
there whichever database choice was made, as today's wizard already does. With "Connect my own
instance after setup" the project is empty, so the band's first step is "Connect an instance", which
opens the instance form for that project; the form still offers the sample, so a connection that
will not work is one step from a working database. With the sample the project has a database and no
change yet, and the band lists "Change a database" and then "Query a database", in that fixed order,
the first as the primary button. This design makes the order follow the goal: "Query a database"
leads for the query goal and for "just exploring", and "Change a database" for the change goal and
when no goal is known. An account that invited nobody also sees "Manage members" as an optional last
step. The goal is read from where it is already kept: the browser's local storage, per workspace and
user, written by the wizard and read by the guide
(`frontend/src/modules/workspace-setup-guide/selection.ts`). That is enough here, because the order
matters in the session that follows setup, which runs in the browser that ran the wizard; another
browser, or an invited teammate, has no stored goal and sees the order as built. The wizard builds
no landing of its own. An earlier draft opened the goal's first action straight from Finish and
highlighted "Connect instance" on the databases page; both are withdrawn, because the project home
is where the next step is chosen.

**D9 · The left panel shows what each screen's answer is for.** Screen 1 previews the workspace
being named: a small Bytebase window with the workspace name in its header and the person setting it
up as its Workspace Admin, drawn from the answers as they are typed, so both names have a visible
use. Screen 2 keeps that preview until a goal is chosen, then shows the goal the moment its card is
pressed: the homepage hero (requests flowing through SQL lint, agent review and a reviewer) for the
change goal, a SQL statement above its result for the query goal, and for "just exploring" the
homepage's after-Bytebase diagram (people and agents reaching the databases through Bytebase),
redrawn narrower to fit the panel. The card's finish line does not depend on the picture. Screen 3
draws a diagram from what the user typed (the sample instance and its database flowing into the
project name, or the user's own instance joining after setup); screen 4 shows Access Controls on the
homepage's green stage, which is why no role is asked. The goal pictures sit on the hero's light
background, and the query picture is a frame of bytebase.com's Dynamic Data Masking file with its
masking rows replaced by the statement, so it shows a query rather than masking. The hero and Access
Controls are still frames of bytebase.com's Rive files (`masthead`, `iam`), which a follow-up may
play with the site's `RivePlayer` (`rive.wasm` self-hosted, the files lazy-loaded on the setup route
only, one frame under reduced motion); the query and exploring pictures are composed for the panel
and stay still. The rocket illustration stays on sign-in and sign-up. Rejected: a constant stage
(the panel teaches nothing); the website's closing slogan, or its brand art alone, on screen 1
(decoration beside a form that already has its heading); the homepage's Change Review file for the
change goal (its dark parts are made for the blue stage, and the hero tells the same story); and a
live preview of the whole product being configured (highest drift; screen 1's preview draws only the
two names it shows).

**D10 · Instrumentation.** Each Continue emits `workspace setup screen continued` on the click, with
the screen number and the answers so far (goal, project named, source). No event carries a name, an
email address or a project name. `workspace setup submitted` still fires when Finish succeeds, now
carrying the goal, the source, the team size and the invite count, and gains `result=failed` with
`failure_stage` (names, project, sample, invitees). `sample_enabled` says whether the attempt asked
for the sample, whatever its result. An attempt that stops (D11) sends neither; the
Finish that completes setup reports. No client event can be the record, though: the 17 lost finishes
lost every custom event after the wizard mounted, and 13 accounts since Sep 11 were never tracked at
all. The record stays with the backend, as the Cloud usage dashboard's exporter already reads it:
"finished setup" from the workspace's first project, and "connected an instance" from its first
instance, sample or own. An account that defers its instance therefore shows up as finished but not
yet connected, with no new backend work.

**D11 · Gate, draft and resume.** The gate is unchanged: first login, sole admin, mandatory on
Cloud, and the flag that sends an unfinished Cloud account back to the wizard clears when Finish
succeeds, as it does on submit today. One addition: a workspace admin whose draft is in this browser
may come back to it, as Finish adds the invitees before it ends and the workspace then has other
members. Until then the answers live in a local draft, keyed by workspace and user as the guide's
stored answers are and removed when Finish succeeds, so Back and a reload return to the same screen
with nothing retyped. On Cloud so does a new login in the same browser, because that flag sends the
account back; on self-hosted, where setup is offered on the first login only, a new login opens the
workspace as it does today. The draft also records that the project was created, the one step of
Finish that cannot be repeated, so a retry, after a reload too, goes past it instead of failing on a
project that already exists. One tab works on setup at a time: the one that saved the draft last.
When another tab saves the draft, this one stops and says setup continued in another tab, with
Continue here, which takes the saved draft up with every screen mounted afresh; when another tab
ends setup, this one says so, with a way to the workspace. Continue here and Finish save the draft
before anything else, and every save is marked as new even when no answer changed, so the other tabs
always hear of it. Work this tab had already started, whether a Finish, an ID check or a skip,
writes nothing more to the server or the draft once another tab saves the draft, setup ends, here or
in another tab, or the wizard is left. A Finish stops before its next write, reports no failure, and
shows the notice once the request it is waiting on returns. A request already sent may still take
effect. The tab that now holds setup finds a created project through the three states below and asks
for the sample again; an invitee's roles may land after its read, so its own roles write makes sure
the earlier one cannot land after it (Project members below). So a stale tab never writes over the
draft or over another tab's answers, a Finish record never meets answers it was not made with, and a
tab whose saves fail keeps its own progress, since without storage no other tab can hear of it.

A create can also be sent and never answered, so the draft holds one of three states for the
project: none, a create sent for an ID with its outcome unknown (recorded before the call), and
created. The unknown state is settled from the server, and only from the server. The project that
holds the ID is the draft's own when it carries the draft's name and its access is still what the
server gives a new project, its creator as owner and nobody else; a project anyone else has been
given access to is never taken over, whatever its ID and name. No such project in this workspace
settles nothing: the create may still be on its way. Two moments settle it. At Finish the create is
sent again, and when it reports that the ID already exists, the draft's own project is taken as
created and anything else is a clash. On screen 3, when it is shown with the outcome still unknown:
the draft's own project puts the screen in its "project exists" state, and another's clears the
record; an answer that arrives after the name or ID was changed is about neither and is dropped. No
project found, or a server that cannot be read, leaves the state unknown, the attempt retryable, and
"I'll do this later" withheld, because a project may exist or still be made; a name or ID changed in
the meantime leaves such a project, if it is made, unused in the workspace. An ID that exists on a
first attempt belongs to someone else, whatever its name. The rule cannot tell the draft's project
from an older one of the same ID and name that the account owns and has shared with nobody; the
wizard takes that one too, as the account named it and nobody else's access is in it, and an ID can
only collide that way if it was typed in by hand.

A clash, and an ID the server refuses for another reason (a reserved one, such as an ID starting
with `default-`), goes back to screen 3 and asks for a different ID under the field, with the
server's reason. The refusal settles that create, and is kept in the draft with its ID, so a reload
still shows the reason until the ID changes. Once the project exists screen 3 shows its name and ID
read-only, and the database choice too once the sample's request has returned: both exist, and are
changed in the product, not in the wizard.

The other steps run again on each retry from the answers on screen, so a checkpoint never outlives
the answers it was made with. Each is written to be safe on state the wizard shares with the rest of
the product:

- **Names.** An overwrite.
- **Sample.** Requested again until its request has returned once, which is safe: a repeated request
  for the same project answers with the same instance (Given). After a reload that cut the request
  short the server already lists the sample under this project, on self-hosted before it is ready.
  That is not a sample another project used, so the sample stays on offer and the stored choice
  decides: the wizard asks again, unless the account chose "Finish without the sample".
- **Project members.** The wizard changes only what it added. For each of the two invitee roles the
  draft records the members a write gives it, which leaves out anyone who held the role already. The
  record is made before the write, because its answer may never arrive, keeps the earlier additions
  until a write is known to have taken them back, and is narrowed once one has. A write the server
  refused changed nothing, so its additions leave the record again; a failure that may have come
  after the write was applied keeps them. A retry gives the two roles to the listed addresses and
  takes them only from recorded members who are no longer listed, so an invitee removed after a
  failed attempt leaves the project too, and a role anyone else gave keeps its members, in their
  order. A draft that never added anyone writes nothing to the project, however long it was left.
  The write replaces the project's whole access policy, the owner's binding included, so it is built
  on the policy read from the server at that moment, with its etag, and is not made when that read
  fails. The copy the page keeps is not used: it would miss an earlier attempt's write whose answer
  was lost. Every attempt that has invitees or recorded members makes the write, even when the
  policy already is what it would make it: an earlier write may still be on its way, after a lost
  answer or from a tab whose Finish stopped (D11), and no read can see it. Both carry an etag, so
  whichever lands second is refused; when the earlier one lands first, the retry that follows reads
  what it did.
- **Workspace members.** Additive: each address gains the member role and keeps any role it already
  holds. It is built on the workspace policy read from the server at that moment, for the same
  reason.

One outcome is accepted, not prevented. An address taken off the list after an attempt whose
workspace write was applied, but whose answer was lost or was still out when its Finish stopped
(D11), stays a workspace member, and on Cloud has its invitation; undoing that would have the wizard
remove workspace members, which it never does.

A failed attempt is reported for the answers it ran with, so its report goes when its screen is
left, and a failure adding the people listed goes when the team size or the addresses change: Back
from screen 4 and a changed answer do not return to the earlier attempt's error. Any other step's
failure still stands after a team answer changes, with "Finish without the sample" when the sample
is what failed. The
wizard collects no credentials, so the draft holds none. No Continue or Finish is disabled without
its reason on screen, and the reason sits with the field or question it is about, as the UX
contract requires (`docs/agents/frontend-ux.md`): under the workspace name, under the goal choices,
under the project name, under the team-size question. That includes the project ID: Continue waits
only on an ID the field has found wrong and says so, not on a check still in flight, because Finish
has the server's answer for an ID that got past it. Two of the 42 leavers stalled on a disabled
button with no explanation.

**D12 · Self-hosted keeps its differences.** The same four screens serve self-hosted, where setup is
offered once and screen 3 keeps "I'll do this later". That action saves the two names from screen 1
and then leaves for the workspace landing as it does today; if the save fails it stays on screen 3
with the error and can be pressed again. No project is created, so nothing else is left behind. The
skip is offered only while that holds: once a Finish attempt has created the project, or has sent a
create whose outcome is unknown (D11), screen 3 no longer shows it, and the way out is Finish. Two
lines of copy follow the deployment (Given) instead of repeating Cloud's. The sample choice says the
database runs on this server and names no expiry, as the product's existing self-hosted sample
strings do. And screen 4 adds members where Cloud invites them (D7).

## States

| Screen | State | What is on screen | Button |
|---|---|---|---|
| 1 Workspace | default | name and workspace name prefilled; the panel previews the workspace with both, updating as they are typed | Continue enabled; disabled if the workspace name is cleared, with "Enter a workspace name" under the field |
| 2 Goal | none chosen | two cards and the lighter "Not sure yet, just exploring"; the workspace preview stays | Continue disabled, with "Pick one to continue" under the choices |
| 2 Goal | a card chosen | card pressed; the panel shows the hero for the change goal, and a statement above its result for the query goal | Continue enabled |
| 2 Goal | just exploring | the third answer pressed; the panel shows people and agents reaching the databases through Bytebase | Continue enabled |
| 3 Project | no name | project field empty; the sample choice preselected | Continue disabled, with "Enter a project name" under the field |
| 3 Sample | named, sample chosen | diagram of the sample instance, its database, the project | Continue enabled |
| 3 Own | named, own instance chosen | one line on what happens after setup; diagram of the user's instance joining the project later | Continue enabled |
| 3 · no sample | the server reports the sample unavailable, or used by another project | only the own-instance choice, with the reason | Continue enabled once named |
| 3 · skipped | "I'll do this later", self-hosted only | saves the screen-1 names and leaves for the workspace landing; no project is created; a failed save stays here with the error | n/a |
| 3 · project exists | Back after a Finish attempt that created the project, its answer lost included | project name and ID read-only, with a line saying the project exists and is renamed in its settings; the database choice read-only once the sample is prepared; no "I'll do this later" on self-hosted | Continue enabled |
| 4 Team | size unset | five size options, nothing else | Finish disabled, with "Pick a team size to finish" under the question; Back enabled |
| 4 Team | "Just me" | the size options only; no invite field | Finish enabled |
| 4 Team | a size above "Just me" | the chips input appears with one line about membership, titled "Invite teammates" on Cloud and "Add members" on self-hosted; the button reads "Finish", or with addresses entered "Send N invites and finish" on Cloud and "Add N members and finish" on self-hosted | Finish enabled |
| 4 · finishing | Finish pressed | the button shows progress; the form is read-only | n/a |
| 4 · finish failed | a step of the commit failed | the error naming the step, Try again; "Finish without the sample" when the sample failed | Try again enabled |
| any · another tab | another tab saved the draft, or ended setup | in place of the screen: "This setup continued in another tab" with Continue here, or "This setup ended in another tab" with Go to the workspace; a running Finish shows its progress until the request it waits on returns, then sends nothing more | Continue here, or Go to the workspace |
| 3 · ID refused | Finish found the project ID taken, or the server refused it | back on screen 3 with the reason at the ID field, after a reload too | Continue disabled until the ID changes |

After Finish the wizard is gone and the account is on the project home (D8). A reload before Finish
returns to the same screen with the draft.

## Scope

| File | Change |
|---|---|
| `frontend/src/routes/auth/WorkspaceSetupPage.tsx` | Rewrite as four screens with the local draft, validation under each field, and the single commit at Finish. Finish navigates to the project home, as the wizard has since [#21572]. |
| `frontend/src/routes/auth/WorkspaceSetupQuestionnaireStep.tsx` | Retire; the goal moves to screen 2 and the team question is removed. |
| `frontend/src/app/layouts/SplashLayout.tsx` | Setup variant: a panel slot instead of the fixed illustration. |
| `frontend/src/routes/auth/workspace-setup/` (new) | Beside the route that is their only consumer: screen components, the left panel (the workspace preview, still frames from two of bytebase.com's Rive files and two pictures composed for the panel), the diagram panel, the chips input, whose text field keeps the shared `Input`'s `xs` size, the chips' text size, and only drops its border. The screens use the shared `StepIndicator`, `RadioGroup` and form primitives, with each validation message under its field. |
| `frontend/src/modules/workspace-setup-guide/` | Take the goal and the team size from the wizard; store "just exploring" as a goal of its own, which uses the query journey; mark "explore database" done on arrival with the sample; start at "Connect instance" on the own-instance path; show the team step only above "Just me". |
| `frontend/src/routes/project/projectHomeNextSteps.ts` | Take the goal as an input. With a database present, "Query a database" comes before "Change a database" for the query goal and for "just exploring"; the order is unchanged otherwise. |
| `frontend/src/routes/project/ProjectHomePage.tsx`, `ProjectDatabasesPage.tsx` | Read the stored goal through the guide module: the home passes it to the next-step function, and the databases page leads with a query for the same two goals. |
| `frontend/src/app/router/guard.ts` | The setup route also admits a workspace admin whose unfinished draft is in this browser, after the wizard's own invitees have joined (D11). |
| `frontend/src/components/ui/step-indicator.tsx` | Shared step indicator: an opt-in `fitOneRow`, so when the titles do not fit one row (a narrow screen, or a long locale such as Spanish) only the current step keeps its title in sight and the others stay for screen readers. Without the prop it behaves as before. |
| `frontend/src/components/ResourceIdField.tsx` | Shared ID field: an opt-in `keepInitialValue`, so the wizard's field keeps a settled ID when screen 3 is shown again. Without the prop it behaves as before. |
| `frontend/src/stores/app/` | An optional argument on five writes (`updateUser`, `updateWorkspace`, `createProject`, `updateProjectIamPolicy`, `patchWorkspaceIamPolicy`) and on `fetchWorkspaceIamPolicy` for a caller that reports the failure itself, so Finish's errors are not also raised as the product's toast. Other callers are unchanged. |
| `frontend/src/utils/storage-keys.ts`, `frontend/src/app/analytics/behavior.ts`, `frontend/scripts/check-i18n.mjs` | The draft's and the team size's storage keys, the per-screen event's name, and the retired string prefixes. |
| `frontend/tests/e2e/` | A second, first-run server for the wizard's spec, started through the same functions as the shared one. |
| `frontend/src/locales/en-US.json` (and the other locales) | New strings for the four screens; retire `setup-scenario.mark-sensitive-data` and `setup-scenario.workspace-usage`. |

No backend, proto or instance-form change.

Tests to write: vitest for each screen's validation message and its place under the field, for the
chips parser, for the draft restoring every screen and staying with its workspace and user, and for
the project home's step order under each goal and with none stored ("Change a database" first for
the change goal, "Query a database" first for the query goal and for "just exploring"); an e2e spec
under `frontend/tests/e2e`, on a server of its own because the sample is single-use, that walks the
own-instance path onto the project home with "Connect an instance", Back from the team screen
keeping every answer, the sample path once, with the query goal, onto the project home with "Query a
database" first, a Finish retried after answers of the server were lost and after the invite list
changed, and the self-hosted skip, sample wording and "Add members" field; vitest for a second tab
taking setup up while a Finish step, an ID check or a skip is out, which then writes nothing more,
for a roles write still on its way, after a lost answer or from a stopped tab, which cannot land
after the next attempt's write, and for a create still on its way when screen 3 finds no project,
which the retry then takes as the draft's own; and event tests that each Continue and the submitted
result carry the properties in D10.

## Not in this PR

The Rive player of D9, which follows separately. The project home beyond the order of those two
steps: when a step leaves the band, and everything else on the page. The guide bar beyond the
answers the wizard hands it. Connection-failure recovery on the instance page (naming a private
address before the test, showing the Cloud egress address, cause-specific fixes): that is where the
62 failed tests happen, and it is separate work on that page. The live-preview panel for screen 3.
Persisting the goal or the team size on the server; both travel as event properties and in the
browser's local storage until a consumer needs them elsewhere. Any change to the sample itself, such
as a test and prod pair, which would reopen ADR-0004. Making setup mandatory on self-hosted.
Enforcing mandatory setup across browsers: the flag is per-browser today, so an unfinished account
can still enter from another one, as 3 accounts have since Sep 20, and closing that needs
server-side state.

[#21572]: https://github.com/bytebase/bytebase/pull/21572

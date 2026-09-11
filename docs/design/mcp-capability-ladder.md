# MCP access policy — show what a mode allows

Status: accepted · 2026-09-10

The MCP settings page offers three modes (Disabled, Read-only, Read-write) and describes each in
one sentence. Since [#21324](https://github.com/bytebase/bytebase/pull/21324) removed the per-mode
method drawer, nothing on the page says what a mode actually serves, and an admin choosing between
Read-only and Read-write has nothing to compare beyond two sentences. The rule this doc lands on:
**show capabilities, not methods or permissions — one ordered list of eight capability rows in
which each mode is a prefix, collapsed by default inside the existing Access policy section, and
following the picked mode while editing.** The page also gets shorter: the mode cards shrink to an
icon, a label and a three-word caption, with the selected mode's "Best for" under them; the
disclosure line is the mode's description, and row details sit behind one toggle. The product change
is frontend-only, with one comment added to the `mcp_method_class` annotation and its generated
output; keeping the row wording true to the method classification is a rule in `AGENTS.md`
rather than a lint (D11). A custom access policy is out of scope, but the list is shaped so that it
becomes that editor later without a redesign.

## Problem

The page today (`frontend/src/routes/workspace/mcp/MCPAccessPolicySection.tsx`) shows, in view
state, an "In force" line with the mode chip, the mode's description, and two notes; in edit
state, three mode cards, the masking toggle, and a footer. The consent page
(`frontend/src/routes/auth/MCPConsentCeiling.tsx`) renders three prose lines for the same policy.
None of these says what an agent can do.

The drawer that used to answer this listed every served method grouped by service, with the
permission each declares, plus per-engine read-only depth and masking. It went with the
`GetMCPInfo` fields it read, because a large API whose only reader was one drawer was not worth
keeping, and because a list served by the backend and rendered by the frontend drifts. It also
answered the wrong question. An admin does not think in `DatabaseService/GetDatabaseSDLSchema`,
and comparing two modes meant opening two drawers.

The gap, as raised in review: some admins need to understand what each mode really allows. The
word "permissions" is itself the trap here. A mode never grants or removes a permission; the
ceiling gate (`backend/api/v1/mcp_gate.go`) removes *methods* by class before the permission check
runs, and under Read-only the SQL clamp removes *statements*. So the page has to say what an agent
can do, in the admin's words, and say it in a form that stays true after the next release.

A second problem surfaced once the list existed: the edit state carried about 375 words, most of
them duplicated. The three mode cards described what the list already showed, and every row's
detail line was visible whether or not anyone wanted it.

## Principle

> **A capability is a verb phrase an admin recognizes, standing for an exact set of served methods. The three modes are nested, so one ordered list with two dividers shows all three at once,
> and the mode in force — or the mode being picked — is a highlighted prefix of that list.**

What follows from it:

- The unit shown is a capability row, never a method, a permission, or a count.
- The list is the same in view and in edit. Editing changes which prefix is served, nothing else.
- Rows a mode does not serve stay visible and muted, so comparing modes never needs a second
  surface.
- What no mode serves is one line under the list, never a row.
- Nothing on the section is said twice. The mode name appears in the chip or the selector; the
  description in the disclosure line; the audience in the "Best for" line; the detail behind one
  toggle.

## The rows

Eight rows: three read, five write. Each has a title and a one-line list of sub-items in plain
words. The right column records which classified methods the row's wording has to cover; it is
documentation for whoever rereads these rows after a classification change, and never appears in
the product.

| Tier | Row | Sub-items shown | Classified methods the wording must cover |
|---|---|---|---|
| read | Read schemas and metadata | Schemas · Databases and instances · Projects and database groups · Catalogs, changelogs and revisions · SQL review configs · Your own session and workspace facts | 31 READ methods: `DatabaseService` reads, projects, instances, database groups, catalogs, changelogs, revisions, review configs, session facts |
| read | Read data by running queries | Run read-only queries · Under Read-only, a request is refused whole unless every statement is shown to be a read that returns data; how deeply that can be checked varies by engine, and on some engines no statement can be shown to be a read at all · Your own query history · Saved queries and sheets you have access to | 9 READ methods: `SQLService/Query`, query history (4), saved-query reads (3), `GetSheet` |
| read | Read the change workflow | Issues and comments · Plans and plan checks · Rollouts, task runs and logs · Releases · Rollback previews | 16 READ methods: issue, plan, rollout and release reads |
| — | *Read-only stops here* | | 56 methods |
| write | Propose changes | Create sheets · Create and edit plans and issues · Run plan checks and reviews · Create, delete and restore releases and revisions · Generate schema diffs. An agent never approves its own change; the project's approval policy decides whether a human must | 22 WRITE methods: sheet, plan, issue, release and revision writes, `RequestIssue`, `RunReview`, `DiffSchema`, `DiffMetadata` |
| write | Run rollouts and tasks | Create a rollout · Run, skip or cancel its tasks, under the project's approval policy | 4 WRITE methods: `CreateRollout`, `BatchRunTasks`, `BatchSkipTasks`, `BatchCancelTaskRuns` |
| write | Run DML and DDL statements | INSERT, UPDATE, DELETE, CREATE, ALTER, DROP through queries, as far as the engine and the user's own permissions allow | Not a method: the statement clamp in `mcp_sql_clamp.go`, which Read-write lifts |
| write | Export query results | Download results as a file. Data leaves Bytebase | 1 WRITE method: `SQLService/Export` |
| write | Manage database housekeeping | Sync instances and databases · Database settings and labels · Move databases between projects · Database groups · Saved queries | 14 WRITE methods: sync, `UpdateDatabase`, database groups, saved-query writes |
| — | *Read-write stops here* | | 97 methods |
| floor | Never, in any mode: approve issues, administer the workspace, handle credentials, open an Admin mode session, or read anyone else's query history. | | 121 methods: 35 FORBIDDEN, 86 EXCLUDED |

Every row is displayed under every mode — served, or muted with a `—` — and against every engine a
workspace happens to hold, so **every line on this card must be true on both axes: mode and engine.**
That covers the eight rows, and equally the collapsed summary, the card captions and the "Best for"
lines, which are not rows and were missed the first time the rule was applied. The summary is the
line most admins read and often the only one. A
claim whose truth depends on the mode names the mode as a condition; a claim whose truth depends on
the engine says so without naming engines, because per-engine depth is out of scope for this card
(see below) and an enumeration goes stale the release a driver changes.

Both axes have caught lines here, and the engine axis caught the same line twice. On the mode axis,
the read-only clamp holds only under Read-only (`mcpReadOnlyClampApplies`), so without its qualifier
row 2 promised a whole-request refusal under Read-write while row 6 on the same screen offered INSERT
and DROP.

On the engine axis, three drafts failed in three different ways, and the shape of the failure is the
lesson. Naming engines (PostgreSQL, CockroachDB, Redshift) went stale against the Redshift datashare
carve-out and was outright false where an engine has no query validator: `refuseNonReadOnlyStatement`
bails before classification and refuses *every* statement, so Read-only serves no query there rather
than a shallowly-checked one. Replacing that with a universal — "queries stay read-only where
Bytebase does not gate per statement" — was false in the permissive direction, because the real
predicate is a conjunction (`HasQueryValidator && !EngineSupportQueryNewACL`) that the sentence
collapsed: on Databricks, outside the ACL set and with no validator, `ValidateSQLForEditor` returns
its permissive default and the write executes. `mcp_sql_clamp.go` carries a comment warning against
exactly that call, eight lines from where the clamp reads it.

So: **state the bound, never the behavior.** Where a claim's truth varies along an axis the card
cannot enumerate, say what limits it — "as far as the engine and the user's own permissions allow",
"how deeply that can be checked varies by engine" — and let the reader learn the specifics from the
docs. An enumeration is wrong the release a driver changes; a universal is wrong the release an
engine is added.

Two further axes were found in review, each after it had already shipped a false line.

**Effect.** A refusal is a claim about what a session can cause, not about which methods it can
call, and the two come apart wherever a permitted method has an effect no annotation names.
`CreateIssue` is an ordinary WRITE; `startIssueWorkflow` calls `emitIssueCreated`, and the webhook
manager posts the issue title and description to whatever endpoint the project configured
(`backend/component/webhook/manager.go`). A Read-write session can put anything it has read into a
description, so the floor's "never … send data to a third party" was false while every method it
reasoned about was classified correctly. Reasoning from the annotation set cannot find this; only
following the effect can. **Check a claim about effects against effects, including the ones no
annotation names.**

The same axis caught a second verb in the same sentence one round later, and the tell was
vocabulary. "Never … open an admin connection" reads as a denial-reason
(`OPENS_AN_ADMIN_CONNECTION`, carried only by `AdminExecute` and `GetTaskRunSession`) and is false as
plumbing: `resolveDataSourceID` falls through to the ADMIN data source for any instance with no
read-only one, in either mode, and `sql_service.go` opens it with
`ReadOnly: clamped || type == READ_ONLY` — so under Read-write an ordinary MCP query runs on the
admin data source, not read-only. **A verb that names a mechanism is a claim about the plumbing and
has to be checked against the plumbing; name the feature the reader knows instead** — here Admin
mode, which really is refused.

**Complement.** A conditional line also asserts something about its complement, and a reader takes
the reassuring half. "Where Bytebase cannot check a statement, it is not verified before it runs" is
true as written and implies that where Bytebase can check, it does — but Read-write leaves
`mcpReadOnlyClampApplies` false and skips `validateQueryRequest` for every engine in
`EngineSupportQueryNewACL`, so on PostgreSQL, the engine with the most complete parser, nothing
checks the statement at all. The conditional pointed its reassurance exactly where the product is
weakest. **If negating the condition yields a promise the product does not keep, drop the condition
or state the bound unconditionally.**

Dropping it unconditionally was the next round's defect, and it is the more useful half of this
entry. "This mode does not require a statement to be a read" is what the MCP ceiling does — the
clamp is off — but not what the product does: `Query` runs `validateQueryRequest` under every mode
for engines outside `EngineSupportQueryNewACL`, and 13 of those 14 have a registered validator, so a
Read-write session there is still refused a write. The repair for the complement axis had removed
the engine axis from the one line that carried it, on a screen that renders no row details. **A line
being fixed on one axis is still a line, and has to be rechecked against every axis — the one being
repaired crowds out the rest.** The wording that holds on both states the bound and the consequence:
"Capped by your own Bytebase permissions, and by what Bytebase supports on each database engine — this mode lifts the read-only limit where it can, and refuses the write outright where it cannot".

Two choices in the wording are deliberate. Row 2 says *Read data by running queries* rather than
"Run queries" so the verb stays Read and the sub-item carries the rule that keeps it true under
Read-only. Row 6 is not a method at all: it names the statement clamp being lifted, because that is
a real difference between the modes that no method name shows, and a future custom policy will want
it as its own switch.

Every sub-item names only what the served methods can do. Sheets are created, never edited: there
is no update RPC. Releases and revisions are created and deleted: `ReleaseService/UpdateRelease`
is classified WRITE but answers Unimplemented, and Revision has no update RPC, so "edit" would
advertise an operation that does not exist. Moving a database between projects is named under
housekeeping because `DatabaseService/UpdateDatabase` accepts the `project` mask path and
`BatchUpdateDatabases` exists for it; it changes a governance boundary and should not hide behind
"settings and labels". Row 2 states the depth of the read-only check because only PostgreSQL,
CockroachDB and Redshift open the database session read-only; elsewhere the clamp is statement
classification alone, and a statement that classifies as a read can still call a function that
writes. The proto calls the ceiling "classifier-enforced, not proven", and the row says so in the
admin's words.

Approval is stated as the project's policy, never as a promise. The backend requires an approved
issue before a rollout only when the project has `require_issue_approval` on and an issue is linked
(`backend/api/v1/rollout_service.go`); the MCP-origin guard there refuses an issueless rollout only
under that same flag, and an issue whose approval finding produced no template counts as approved
with no human acting. The console turns the flag on for new projects; the API default is off. A row
that said "of an approved change" would therefore promise more than the gate enforces, and the
wording an admin reads while choosing Read-write is the wrong place to be generous. What holds
unconditionally is that an agent never approves its own change, because the three approval methods
are FORBIDDEN, and that is the half the rows state. Whether MCP-originated rollouts should require
approval regardless of the project flag is a product question tracked separately (BOT-71), not one
this doc decides.

### The verb rule

The verb names the row's effect on the customer's databases. *Read*: none. *Propose*: none until a
human approves. *Run*: direct. *Export*: data leaves the product. *Manage*: configuration only, no
data path. Within a tier the verb repeats; it changes only where the effect changes.

One verb for the whole write tier was tried and rejected:

- **Write** for all five: "Write query results" (export writes nothing to a database), "Write
  change proposals" (nothing reaches a database until approval), "Write rollouts" (not English),
  "Write database housekeeping" (reads as a data change).
- **Change** for all five: fails for exports and rollouts, and makes "Change proposals" and
  "Change data with SQL" read as the same thing — the distinction the FORBIDDEN approval methods
  exist to protect.
- **Manage** for all five: hides the hazard on exports and on SQL.

Per row: *Propose* over Create (Create fits the method names but hides that the project's approval
policy may still require a human); *Export* over Read or Download (Read hides the egress hazard,
which is why Export is WRITE-class in the first place); *Manage* over Update or Maintain (Update
reads as a data change, Maintain is vague). Rollouts and DML/DDL both take *Run* and stay two rows,
because a custom policy will want "run approved rollouts, but no ad-hoc SQL" — the workflow the
product exists for.

The tier is named once, by the dividers, so no row repeats it in its title.

## Decisions

**D1 — Placement: a disclosure inside the Access policy section, collapsed by default.**
Not a drawer (a second surface, one mode at a time) and not a separate section (it belongs to the
policy it describes). Collapsed, the trigger line *is* the mode's description: "▸ Read schemas, data
and the change workflow; propose, run, export and manage". The read half is the same phrase in
both modes, because Read-write serves exactly the read rows Read-only serves; the EXCLUDED reads
(audit logs, users, roles, IAM policies, other people's query history, task-run sessions) are
refused in every mode, so no summary may say "read everything". The line never repeats the mode
name, which the chip (view) or the
selector (edit) already shows. Expanded, it becomes the list heading "Read-write allows", with a
"Show details" control on the right. In view state it sits under the chip line; in edit state it
sits under the selector and its "Best for" line, above the masking toggle, so the cause and its
effect are adjacent. The open state persists per browser and carries across the view-to-edit
transition within a page visit.

**D2 — Content: rows first, sub-items behind one toggle.** Expanded, each row shows its title and
its tier tag, nothing else. "Show details" reveals the sub-item line for every row at once, so
there is one second-level control rather than eight; per-row expanders were rejected as fiddly. No
method names, no permission names, no counts, no link to a method list. The floor is kept as one
short line because it is what a security-minded admin reads first, and it is what keeps the list
honest about the 121 methods no mode reaches. There is no constants line under the list; the facts
it would carry (capped by the user's own permissions, refusals audited) move into the section
description.

**D3 — The mode chip is the subject; the "In force" line and the mode sentence go.** The
three-stripe icon and the words "In force" are removed. View state reads: the mode chip (success
for Read-only, warning for Read-write, destructive for Disabled), the "Masking exemptions ignored"
chip when set, Edit policy on the right; then the disclosure line, which carries the description.
The mode chip carries the mode's icon before its name — the same Lucide glyph as the card in edit
state (`Unplug`, `Eye`, `PencilLine`) — so the identity an admin picked is the identity shown in
force. The chip is the only place the mode's color appears; the cards stay neutral, because a
control must not look like a status badge and the accent must remain the one selected-state color.
The separate sentence under the chip is retired for Read-only and Read-write, since the disclosure
line says the same thing; Disabled keeps its sentence, "No MCP session can connect to this
workspace.", because it has no list. "Active" was considered and rejected as the label: "Active ·
Disabled" contradicts itself, and a chip under a section titled Access policy needs no label. The
chip announces "Current policy: Read-only" through visually hidden text, never through
`aria-label`: a chip is a bare `span`, whose implicit `generic` role ARIA forbids naming, so a
label put there is dropped and the deleted "In force" text is replaced by nothing. Tests assert
the rendered name rather than the attribute, which satisfies a DOM query while naming nothing.

**D4 — The two notes under the card move and shrink.** "A ceiling change applies to the next
request…" shows only while editing, as "Applies to every running session's next request.", and
names the change when the form is dirty: "Read-only → Read-write applies to every running
session's next request." "MCP policy denials are recorded in the audit log" is a fact about the
feature, not about the current state, so it joins the section description, which becomes: "The
most any MCP session may do here. Sessions are also capped by each user's permissions, and policy
refusals are audited." The word "policy" is load-bearing: the audit interceptor writes a row for a
refusal only when the RPC opts into auditing or the gate marked a policy denial, so a permission
denial on an unannotated method is silent, and "every refusal" would promise more than the backend
records.

**D5 — Edit state: icon cards, the selected mode's "Best for", and the pick's ladder.** The
three mode cards lose their descriptions; they duplicated the ladder. What remains is the shape of
the instance engine selector: three bordered cards in a row, each with a Lucide icon, the mode name,
and a three-word caption — Disabled "No sessions connect", Read-only "Explore and query",
Read-write "Change data and schemas". The icons are `Unplug`, `Eye` and `PencilLine` from
`lucide-react`, which the frontend already ships: unplug says nothing connects, in the same
vocabulary as the Connect a client section below; eye is view only; pencil-line is write. Icons
are grey and turn accent on the selected card, which also carries the accent border, ring and
tint; no red, green or amber on the cards, so the mode's color keeps meaning one thing on the chip.
A label-only card was tried and read empty; the shared segmented control was tried and read as a
toggle. The caption costs nine words and gives back the at-a-glance comparison the old cards
provided. Under the cards, one line: the "Best for" of the selected mode, shown once instead of
three times. Under that, the disclosure for the picked mode, rendering exactly what the view would
show after saving. No "adds N over Read-only", no added or removed marks, no comparison against the
stored mode: the muted rows already show what a pick does not serve. The edit state is about 125
words collapsed and 175 expanded, down from about 215 and 375 in the first draft of this design.

**D6 — Marks: ✓ or —, plus a tier tag on served rows only.** A served row shows ✓, a `success`
"read" badge or a `warning` "write" badge after its title, and full-contrast text. An unserved row
shows —, muted text, and no badge. Under Read-only the write rows are unserved, so a "write" badge
never appears there.

**D7 — Disabled.** In view, the chip and the sentence "No MCP session can connect to this
workspace." are the whole answer; no disclosure. In edit with Disabled picked, the Disabled card is
selected, the "Best for" line reads "keeping MCP off until you are ready to turn it on", and the
disclosure slot holds a static line in the floor's soft error tone, "Nothing is allowed; no MCP
session can connect.", so red means "no capability" everywhere on the card. It is not a button and
does not repeat the mode name.

The rule extends past color: **a control that governs only a serving session is withheld while
Disabled is picked, and the save leaves its stored value alone.** The masking toggle is the one
such control today. `mcpIgnoresMaskingExemptions` (`backend/api/v1/mcp_masking.go`) answers on the
delegated grant an MCP request carries, and Disabled admits no MCP session, so the stored flag is
never read there — a live toggle under a red line saying no session can connect would be the card
asserting two things that cannot both hold. Withholding the control is not a reason to discard what the admin set with it, and two attempts to
make it one both lost an explicit choice. Gating the write dropped the edit on a save under Disabled;
resetting the draft on the pick dropped it earlier and even when the admin returned to a serving
mode. The draft is therefore kept and saved whatever the pick: clicking through the modes to read
their descriptions must not silently undo an unrelated edit, and under Disabled the stored flag is
inert rather than wrong, so writing it costs nothing now and honors the choice when MCP is turned
back on.

The "Masking exemptions ignored" chip belongs to the **view**. It reports the stored flag in the
present tense and says which of three things that flag is doing: in effect, stored but unlicensed,
or stored with MCP off. Withholding it was tried and is wrong in both inert cases — the flag is
storable under Disabled, which preserving the draft makes reachable by design, and hiding it leaves
a set flag with nowhere to see it. The reasons it might be inert live in the editor and in the
Disabled sentence, and the editor is a different branch behind `bb.settings.set`, so a reader of
this page may never reach them; the chip therefore carries the reason itself. Its inventory is
therefore Disabled, then licensed or not — three arms, because a Disabled policy is reported as off
whether or not the workspace holds a masking license. It renders only where a stored mode exists to
qualify it, and takes that mode as an argument so the branch that proved there is one passes the
proof in. A predicate would not: `!isServingMode(storedMode)` also catches the ceiling this build
cannot parse, and would label a policy nobody turned off as "MCP is off". That ceiling gets no chip
at all, which is the one place this decision accepts a set flag with nowhere to see it: the repair
card carries a single instruction — pick a mode — and a second chip beside it competes with the only
action that resolves the state.

The editor does not render it, and an attempt to do so failed three ways at once. A present-tense
chip under a pick that admits no session states a live restriction directly beneath the red line
saying nothing can connect — the contradiction this decision withholds the toggle to prevent,
restated as a chip. Its text came from the stored pair while Save writes the draft, so it could
assert the opposite of what Save was about to do, in both directions. And gating it on the stored
flag left the "turned it on, then picked Disabled" case showing nothing at all — the very gap it was
added to close. **A surface discloses the state it owns: the view reports what is stored, the editor
reports what Save will write.** Under a pick that withholds the toggle the editor says so in the
footer, in the future tense, naming the direction — and whenever the saved flag will be set, not
only when this edit changed it, because an admin who cannot see the control cannot see the value
either.

Two bounds on that line, both found by making it say too much. It is gated on the form being
saveable: an editor opened and not touched has no save to describe, and under no pick at all the card
would otherwise say "pick a mode to save this policy" and "this policy will be saved" at once. And it
says only what will be written, never when that takes effect — "takes effect when MCP is enabled" is
false on a workspace with no masking license, and the caveat that would fix it lives in the toggle
this pick withholds. **State what is written, never when it takes effect**: the write is a fact this
card owns, the effect depends on axes it cannot see.

The consent page still withholds its masking line without a license, because its reader sees neither
the setting nor any caveat and would read the line as "my data is covered".
(Mock E draws the toggle under Disabled; the implementation does not.)

**D8 — Copy that shrinks, but keeps the coverage limit.** The masking toggle's two paragraphs
become: "If enabled, masked data stays masked in MCP sessions even for users with exemptions or
unmask grants. Coverage depends on the engine: where Bytebase does not mask, this changes nothing.
The console is unaffected." The middle sentence stays in the product on purpose. Masking runs only
on the engines `common.EngineSupportMasking` lists; on the others the query masker falls back to a
no-op, so on Snowflake or ClickHouse the toggle keeps nothing masked, and the setting's own proto
comment says it "is not a confidentiality boundary". A toggle that promised more would mislead the
admin it exists to protect. The three "Best for" lines keep their current wording. The Read-only description sentence ("Sessions can explore schemas and run
read-only queries…") is retired everywhere; its content lives in the Read-only summary and in row
2's sub-item.

**D9 — The Authentication Required alert on the MCP page is removed.** Its two sentences already
exist on the page: "approve access in the browser" in the Connect a client description, and
"appropriate permissions" in the section description. Connect a client gains the one clause that
was useful: "Add Bytebase to your AI client and start asking. On first connection you sign in and
approve access in the browser."

**D10 — The consent page uses the row titles, and bounds them once.** "This session may" lists the served rows with ✓,
one ✕ line for the unserved tier under Read-only ("No changes, rollouts or exports"), then the
existing capped, masking and audit lines. Read-write keeps its caution. Its mode chip carries the
same icon as the settings page's. One wording table serves both surfaces.

Titles alone carry no caveats, and the caveats live in the row details this screen does not render —
so on a workspace whose engine refuses every statement, an unqualified "Read data by running queries
✓" promises a capability the session does not have. The screen cannot carry eight detail lines and
stay an approval screen, so it bounds the whole list once instead, on the line that already limits it
by the reader's own permissions. That bound is per mode, because the statement clamp it describes
runs only under Read-only: "Capped by your own Bytebase permissions, and by what Bytebase can check on each database engine — where it cannot show a statement is a read, no query runs at all" against
"Capped by your own Bytebase permissions. This mode does not require a statement to be a read". The Read-only line carries the engine axis
because the clamp genuinely consults `HasQueryValidator`; the Read-write line carries it because
`validateQueryRequest` still refuses a write on the engines outside `EngineSupportQueryNewACL`. What
it must not carry is a *condition* — that points its reassurance at the engines Bytebase parses best
and checks least (see the complement axis in The rows). One line either way, and the ✓ marks
read as what the policy admits rather than what will succeed. It carries a neutral glyph and no
"Allowed" mark, so a bound is not counted as a further grant.

**D11 — The wording is bound to the classification by instruction, not by a lint.** The eight
titles claim to cover every READ and WRITE method, and a method annotated into either class is
served the moment it is annotated, whether or not a row names it. The first draft closed that with
a row table in `backend/api/v1/mcp_gate.go` and a lint in `mcp_gate_test.go`, so a class change
failed CI until the table — and therefore the row wording — was reviewed. That was dropped: it
bought a mechanical check at the cost of a second table to keep in step, on a set that changes
rarely and only in commits already about MCP classification. A later review pointed out that a
cheaper check exists and needs no second table: `TestMCPClassificationInventory` already renders the
annotations into `backend/api/v1/testdata/mcp_method_classification.md`, whose header carries the
same per-class counts this row table sums to, so asserting those integers would fail on exactly the
change that requires the reread. That is a real option and it is not the one taken; what it buys is
a prompt to reread, not proof that the wording is right, and the decision here is to spend the
reread on the instruction rather than on a gate. The cost of the trade is unchanged and stated
below. The rule instead lives under Metadata
and API conventions in the root `AGENTS.md`: annotating an RPC READ or WRITE means rereading the
rows and rewording one, or adding one, when none describes it. The cost of the trade is that a
reclassification which skips that reread is silent — the page keeps its old sentences and an admin
picks a ceiling on them. No new API and no field the frontend reads: the product shows no counts, so
it needs only the static tier of each row. The one code change is a comment on `MCPMethodClass` in
`proto/v1/v1/annotation.proto`, stating that the class is disclosed to admins as capability rows —
put where the annotation is typed, since that is where the reread has to happen. It is worded as a
fact rather than a chore, and names no repo path, because proto comments ship into the published API
reference.

## States

| State | What the section shows |
|---|---|
| View · Read-only or Read-write | Chip line with Edit policy, plus the masking chip when the flag is stored (D7); the disclosure line as the description, collapsed by default. The open state persists per browser (D1), so neither the product nor a test may treat collapsed as an invariant. |
| View · Disabled | Chip line, with the masking chip naming MCP as off when the flag is stored (D7); "No MCP session can connect to this workspace." No disclosure. |
| View · unreadable, unserved, read failed | The existing warning or error, unchanged. No disclosure. |
| Edit · Read-only or Read-write picked | Icon cards with the pick selected; the pick's "Best for" line; the disclosure for the pick, collapsed by default, rendering the post-save view, with "Show details" once expanded; masking toggle; separator; footer sentence (naming the change when dirty), Cancel, Save (enabled only when dirty). No masking chip: in edit the toggle is the flag's disclosure (D7). |
| Edit · Disabled picked | Icon cards with Disabled selected; its "Best for" line; the static soft-error line in the disclosure slot; NO masking toggle and NO masking chip (D7); separator; footer — the mode sentence, plus, once the form is saveable, the line naming what Save writes for the masking flag — Cancel, Save. |
| Edit · nothing picked | Only reachable from an unreadable or unserved ceiling: icon cards with no selection; "Pick a mode to save this policy." in the disclosure slot; no masking toggle, no masking chip, and no pending line, because nothing can be saved yet; Cancel, Save disabled. |
| Consent page | Served row titles with ✓, the ✕ line under Read-only, then the existing constants and caution. |

## Copy

The strings as decided, with the reasoning that picked them. The locale files are what ships and
what to edit; this section is the record of why, and a copy edit is expected to update both. Keys
under
`settings.mcp.ladder.*` are new; the rest replace existing `settings.mcp.*` and
`oauth2.consent.mcp.*` values.

- Section description: "The most any MCP session may do here. Sessions are also capped by each
  user's permissions, and policy refusals are audited."
- Disclosure line, collapsed — Read-only: "Read schemas, data and the change workflow; a request carrying anything that cannot be shown to be a read is refused, and nothing is exported".
  Read-write: "Read schemas, data and the change workflow; propose, run, export and manage".
  Expanded heading: "{mode} allows". Details control: "Show details" / "Hide details".
- Disabled — view sentence: "No MCP session can connect to this workspace." Edit static line:
  "Nothing is allowed; no MCP session can connect."
- Card captions: Disabled "No sessions connect"; Read-only "Explore and query"; Read-write "Change
  data and schemas".
- "Best for" lines, unchanged: Disabled "keeping MCP off until you are ready to turn it on";
  Read-only "querying and exploring data, including by people who do not write SQL"; Read-write
  "making database changes through an AI agent, still capped by each user's own permissions".
- Row titles and sub-items: the table above, verbatim.
- Dividers: "Read-only stops here", "Read-write stops here".
- Floor: "Never, in any mode: approve issues, administer the workspace, handle credentials, open an Admin mode session, or read anyone else's query history."
  The verbs cover every denial reason, and claim only what is refused **whatever the caller's
  permissions** and by **whatever effect** a served method has. That second rule cost three drafts. MCP's ceiling removes methods by class; it
  never adds per-row privacy, so an ownership claim — "never read other people's SQL", then
  "never browse everyone's saved SQL" — is false at whatever permission level makes it true in the
  console: `GetSavedQuery` reaches SQL shared by a binding, and `searchScope` drops all scoping for
  a caller holding project-wide `bb.savedQueries.get`. Query history is the one that qualifies,
  because `SearchQueryHistories` pins the creator and nothing widens it. A fourth draft added "or
  send data to a third party" and was false on the effect axis: `CreateIssue` is served under
  Read-write and its webhook posts the title and description outward. It is not replaced by a row
  caveat — the webhook fires identically for an issue a human files in the console, so it is a
  property of the workspace's notification config rather than of this ceiling, and the "Propose
  changes" row already says the session can file issues. The fifth draft read "open an admin
  connection" and was false on the same axis for the opposite reason — it named the plumbing rather
  than the feature, and MCP does reach the admin data source. It names Admin mode now.
- Tier badges: "read", "write".
- Masking toggle: the three sentences in D8, including the engine-coverage limit.
- Masking chip, by what the stored flag is doing: "Masking exemptions ignored",
  "Masking exemptions ignored — masking not licensed",
  "Masking exemptions ignored — MCP is off".
- Footer, once the form is saveable under a Disabled pick, naming what Save writes and not when it
  takes effect:
  "This policy will be saved with masking exemptions ignored."
  and "This policy will be saved with masking exemptions applied.".
- Footer, clean: "Applies to every running session's next request." Dirty: "{from} → {to} applies
  to every running session's next request."
- Connect a client: the sentence in D9.
- Consent: row titles; "No changes, rollouts or exports"; the two capped lines above.

## Implementation

### Frontend

- New `MCPCapabilityLadder` beside `MCPAccessPolicySection.tsx`, props `mode`, `expanded`,
  `details`, `onExpandedChange`, `onDetailsChange`. It derives the served set from the mode by tier:
  READ_ONLY serves the read rows, READ_WRITE both tiers, DISABLED none. No comparison logic.
- Disclosure behavior belongs in a shared primitive per the UX contract. There is no
  `Collapsible` in `frontend/src/components/ui/` today; add one wrapping Base UI's Collapsible
  (trigger with `aria-expanded`, panel region) rather than hand-rolling it in the feature.
- The mode selector keeps the `RadioGroup` and `RadioGroupItem` semantics with the radio hidden
  (`radioClassName="sr-only"`), and renders each item the way `InstanceEngineRadioGrid` in
  `InstanceFormBody.tsx` renders an engine: `rounded-sm border px-3 py-2`, selected
  `border-accent bg-accent/5 ring-1 ring-accent`, hover `border-accent/50`. Each item holds the
  Lucide icon at `size-5` (`text-control-light`, `text-accent` when selected), the label at
  body size, and the caption at caption size. Verify the item shows the shared focus ring with the
  radio hidden.
- The ladder is an unframed region: the trigger row and the rows are separated by
  `border-block-border` hairlines, with no outer frame, so the change adds no card inside the
  section's existing framed card. (Design mocks draw a light border around the ladder for
  legibility; the implementation does not.)
- Marks use the shared `Badge` (`success` for read, `warning` for write). The floor line and the
  Disabled static line use the `error` semantic tokens at low opacity; no raw palette colors, no
  `dark:` variants.
- `MCPAccessPolicySection.tsx`: remove the `Rows3` icon, the in-force string, the mode cards and
  the mode sentence; render the chip through the shared `MCPModeBadge`, which the consent page
  also uses, so one component carries the glyph and the accessible name for both surfaces;
  strip the mode cards to icon, label and caption and add the "Best for" line
  under them;
  move the audit sentence to the section description; show the footer sentence only while editing
  and interpolate both modes when dirty; replace the masking copy.
- `MCPPage.tsx`: remove the Authentication Required alert; extend the Connect a client description.
- `MCPConsentCeiling.tsx`: replace the read, write and workflow lines with the row titles from the
  same table.
- Locale: keys under `settings.mcp.ladder.*` in all five locale files. The retired mode-description
  keys are removed, not left empty. `frontend/scripts/check-react-i18n.mjs` enforces cross-locale
  parity over all of them; the four template-keyed families (`ladder.row.`, `.stops.`, `.summary.`,
  `.tier.`) are registered in its `DYNAMIC_PREFIXES`, which exempts them from the unused-key check
  as well, so their coverage comes from `mcpCapabilityRows.i18n.test.ts` instead.
- Tests: the served set per mode; the disclosure collapsed by default, opens, persists, and follows
  the pick; the details toggle reveals sub-items on every row and persists with the open state; the
  "Best for" line follows the selection; Disabled renders the static line in edit and the sentence in
  view; the consent page renders the row titles; one e2e case that edits Read-only to Read-write,
  saves, and sees the chip change.
- Run `node frontend/scripts/check-ui-guideline.mjs`; the change must not add to the legacy
  baseline.

### Backend

No behavior change. One comment on `MCPMethodClass` in `proto/v1/v1/annotation.proto` and its
regenerated output; the rule that keeps the row wording true is an instruction under Metadata and
API conventions in the root `AGENTS.md` (D11), not code.

## Keeping the rows true

Nothing enforces the wording, so annotating an RPC `mcp_method_class = READ` or `WRITE` — a new RPC,
or a reclassified one — is also a change to what the Access policy page and the OAuth consent screen
promise. What to do:

1. Read the row table above and judge whether a row still describes the method. The right column
   records the method families each row stands for, not an exhaustive list, so this is a judgment
   about the wording rather than a lookup. `backend/api/v1/testdata/mcp_method_classification.md` is
   the generated list of every method and its class, and CI forces a diff on it for the same change.
2. If none covers it, reword a row or add one. The copy is `settings.mcp.ladder.row.*` in
   `frontend/src/locales/` — all five files — and the order and tier are in
   `frontend/src/components/mcp/mcpCapabilityRows.ts`. Update the table above in the same change.
3. Check the new wording on all four axes in The rows — mode, engine, effect and complement —
   against their rules: state the bound and never the behavior, name the feature and never the
   mechanism, and claim only what holds whatever the caller's permissions. A line you are changing
   to fix one axis still has to hold on the other three; every regression this page has shipped came
   from checking only the axis that prompted the edit.

The consent screen renders row titles only, so a caveat that belongs to one mode or one engine has
to live in the bound line there (D10), not in a row's sub-items.


## Out of scope

- A custom access policy. When it comes, the rows become checkboxes, the cards become preset
  buttons that select a prefix, and a policy matching no prefix shows a Custom chip; the backend
  row table is what the gate would read. The ladder needs no redesign. One precondition before
  rows become independently selectable: `SQLService/Export` must be clamped to read statements.
  On MySQL it skips statement validation and the driver executes non-query statements, and the
  clamp today lives only in `SQLService/Query`. Under the presets Export is served only alongside
  the DML/DDL row, so there is no exposure until then. That clamp is an MCP implementation change,
  tracked separately from this doc.
- A docs page listing the methods per row, generated from the inventory. Worth doing; not linked
  from the card until it exists.
- Per-engine read-only depth and masking coverage. The removed drawer showed both; they belong in
  docs, not on the policy card.
- The ceiling gate itself, the classification, and the consent flow's mechanics.

## Open questions

- The Access policy section is a framed card, which predates the "no cards inside cards" rule.
  This change adds no nesting and removes the nested mode cards, so unframing the section is a
  natural follow-up.
- Whether the consent page should also show the sub-items. The proposal shows titles only, to keep
  the approval screen short.

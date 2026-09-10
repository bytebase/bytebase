# MCP access policy — show what a mode allows

Status: proposal · 2026-09-10

The MCP settings page offers three modes (Disabled, Read-only, Read-write) and describes each in
one sentence. Since [#21324](https://github.com/bytebase/bytebase/pull/21324) removed the per-mode
method drawer, nothing on the page says what a mode actually serves, and an admin choosing between
Read-only and Read-write has nothing to compare beyond two sentences. The rule this doc lands on:
**show capabilities, not methods or permissions — one ordered list of eight capability rows in
which each mode is a prefix, collapsed by default inside the existing Access policy section, and
following the picked mode while editing.** The page also gets shorter: the mode cards shrink to an
icon, a label and a three-word caption, with the selected mode's "Best for" under them; the
disclosure line is the mode's description, and row details sit behind one toggle. The change is
frontend-only; keeping the row wording true to the method classification is a rule in `AGENTS.md`
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

> **A capability is a verb phrase an admin recognizes, backed by an exact set of served methods in
> code. The three modes are nested, so one ordered list with two dividers shows all three at once,
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

| Tier | Row | Sub-items shown | Backed by (code only) |
|---|---|---|---|
| read | Read schemas and metadata | Schemas · Databases and instances · Projects and database groups · Catalogs, changelogs and revisions · SQL review configs · Your own session and workspace facts | 31 READ methods: `DatabaseService` reads, projects, instances, database groups, catalogs, changelogs, revisions, review configs, session facts |
| read | Read data by running queries | Run read-only queries; a request is refused whole if any statement is not a read, and on engines other than PostgreSQL, CockroachDB and Redshift the check is by statement shape only · Query history · Saved queries and sheets | 9 READ methods: `SQLService/Query`, query history (4), saved-query reads (3), `GetSheet` |
| read | Read the change workflow | Issues and comments · Plans and plan checks · Rollouts, task runs and logs · Releases · Rollback previews | 16 READ methods: issue, plan, rollout and release reads |
| — | *Read-only stops here* | | 56 methods |
| write | Propose changes | Create sheets · Create and edit plans and issues · Run plan checks and reviews · Create and delete releases and revisions · Generate schema diffs. An agent never approves its own change; the project's approval policy decides whether a human must | 22 WRITE methods: sheet, plan, issue, release and revision writes, `RequestIssue`, `RunReview`, `DiffSchema`, `DiffMetadata` |
| write | Run rollouts and tasks | Create a rollout · Run, skip or cancel its tasks, under the project's approval policy | 4 WRITE methods: `CreateRollout`, `BatchRunTasks`, `BatchSkipTasks`, `BatchCancelTaskRuns` |
| write | Run DML and DDL statements | INSERT, UPDATE, DELETE, CREATE, ALTER, DROP through queries, where the engine checks each statement and the user may run it | Not a method: the statement clamp in `mcp_sql_clamp.go`, which Read-write lifts |
| write | Export query results | Download results as a file. Data leaves Bytebase | 1 WRITE method: `SQLService/Export` |
| write | Manage database housekeeping | Sync instances and databases · Database settings and labels · Move databases between projects · Database groups · Saved queries | 14 WRITE methods: sync, `UpdateDatabase`, database groups, saved-query writes |
| — | *Read-write stops here* | | 97 methods |
| floor | Never, in any mode: approve issues, administer the workspace, or handle credentials. | | 121 methods: 35 FORBIDDEN, 86 EXCLUDED |

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
chip carries `aria-label="Current policy: Read-only"`.

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
asserting two things that cannot both hold. Withholding it costs nothing: picking a serving mode in
the same editor brings it straight back with the draft intact, and the update mask simply omits
`value.mcp.ignore_masking_exemptions`. Omitting it from the mask is what closes a hazard the visible
version carries — toggling masking under Read-write and then picking Disabled would otherwise write
a masking change through a control that is no longer on screen. For the same reason the "Masking
exemptions ignored" chip is withheld from a Disabled policy in view: it asserts a restriction on
sessions that do not exist. (Mock E draws the toggle under Disabled; the implementation does not.)

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

**D10 — The consent page uses the row titles.** "This session may" lists the served rows with ✓,
one ✕ line for the unserved tier under Read-only ("No changes, rollouts or exports"), then the
existing capped, masking and audit lines. Read-write keeps its caution. Its mode chip carries the
same icon as the settings page's. One wording table serves both surfaces.

**D11 — The wording is bound to the classification by instruction, not by a lint.** The eight
titles claim to cover every READ and WRITE method, and a method annotated into either class is
served the moment it is annotated, whether or not a row names it. The first draft closed that with
a row table in `backend/api/v1/mcp_gate.go` and a lint in `mcp_gate_test.go`, so a class change
failed CI until the table — and therefore the row wording — was reviewed. That was dropped: it
bought a mechanical check at the cost of a second table to keep in step, on a set that changes
rarely and only in commits already about MCP classification. The rule instead lives under Metadata
and API conventions in the root `AGENTS.md`: annotating an RPC READ or WRITE means rereading the
rows and rewording one, or adding one, when none describes it. The cost of the trade is that a
reclassification which skips that reread is silent — the page keeps its old sentences and an admin
picks a ceiling on them. No proto change, no new API, no generated file: the product shows no
counts, so the frontend needs only the static tier of each row.

## States

| State | What the section shows |
|---|---|
| View · Read-only or Read-write | Chip line with Edit policy; the disclosure line as the description, collapsed. Nothing below it. |
| View · Disabled | Chip line, without the masking chip (D7); "No MCP session can connect to this workspace." No disclosure. |
| View · unreadable, unserved, read failed | The existing warning or error, unchanged. No disclosure. |
| Edit · Read-only or Read-write picked | Icon cards with the pick selected; the pick's "Best for" line; the disclosure for the pick, collapsed by default, rendering the post-save view, with "Show details" once expanded; masking toggle; separator; footer sentence (naming the change when dirty), Cancel, Save (enabled only when dirty). |
| Edit · Disabled picked | Icon cards with Disabled selected; its "Best for" line; the static soft-error line in the disclosure slot; NO masking toggle (D7); separator; footer, Cancel, Save. |
| Consent page | Served row titles with ✓, the ✕ line under Read-only, then the existing constants and caution. |

## Copy

All strings, so the change and the locale files have one source. Keys under
`settings.mcp.ladder.*` are new; the rest replace existing `settings.mcp.*` and
`oauth2.consent.mcp.*` values.

- Section description: "The most any MCP session may do here. Sessions are also capped by each
  user's permissions, and policy refusals are audited."
- Disclosure line, collapsed — Read-only: "Read schemas, data and the change workflow; statements
  that write are refused, nothing is exported". Read-write: "Read schemas, data and the change
  workflow; propose, run, export and manage".
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
- Floor: "Never, in any mode: approve issues, administer the workspace, or handle credentials."
- Tier badges: "read", "write".
- Masking toggle: the three sentences in D8, including the engine-coverage limit.
- Footer, clean: "Applies to every running session's next request." Dirty: "{from} → {to} applies
  to every running session's next request."
- Connect a client: the sentence in D9.
- Consent: row titles; "No changes, rollouts or exports".

## Implementation

### Frontend

- New `MCPCapabilityLadder` beside `MCPAccessPolicySection.tsx`, props `mode`, `expanded`,
  `details`, `onToggle`, `onToggleDetails`. It derives the served set from the mode by tier:
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
  the mode sentence; render the chip with its aria-label and the mode's Lucide icon as its first
  child at `size-3.5`; strip the mode cards to icon, label and caption and add the "Best for" line
  under them;
  move the audit sentence to the section description; show the footer sentence only while editing
  and interpolate both modes when dirty; replace the masking copy.
- `MCPPage.tsx`: remove the Authentication Required alert; extend the Connect a client description.
- `MCPConsentCeiling.tsx`: replace the read, write and workflow lines with the row titles from the
  same table.
- Locale: keys under `settings.mcp.ladder.*` in all five locale files;
  `frontend/scripts/check-react-i18n.mjs` guards them. The retired mode-description keys are
  removed, not left empty.
- Tests: the served set per mode; the disclosure collapsed by default, opens, persists, and follows
  the pick; the details toggle reveals sub-items on every row and persists with the open state; the
  "Best for" line follows the selection; Disabled renders the static line in edit and the sentence in
  view; the consent page renders the row titles; one e2e case that edits Read-only to Read-write,
  saves, and sees the chip change.
- Run `node frontend/scripts/check-ui-guideline.mjs`; the change must not add to the legacy
  baseline.

### Backend

None. The rule that keeps the row wording true is an instruction under Metadata and API conventions
in the root `AGENTS.md` (D11), not code.

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

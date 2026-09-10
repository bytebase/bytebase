# MCP access policy — show what a mode allows

Status: proposal · 2026-09-09

The MCP settings page offers three modes (Disabled, Read-only, Read-write) and describes each in
one sentence. Since [#21324](https://github.com/bytebase/bytebase/pull/21324) removed the per-mode
method drawer, nothing on the page says what a mode actually serves, and an admin choosing between
Read-only and Read-write has nothing to compare beyond two sentences. The rule this doc lands on:
**show capabilities, not methods or permissions — one ordered list of eight capability rows in
which each mode is a prefix, collapsed by default inside the existing Access policy section, and
following the picked mode while editing.** The product change is frontend-only. One backend lint
binds the row wording to the method classification, so the list cannot drift from what the gate
serves. A custom access policy is out of scope, but the list is shaped so that it becomes that
editor later without a redesign.

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

## The rows

Eight rows: three read, five write. Each has a title, a one-line list of sub-items in plain words,
and — in code only — the set of served methods it stands for. The right column here is the code
mapping; it never appears in the product.

| Tier | Row | Sub-items shown | Backed by (code only) |
|---|---|---|---|
| read | Read schemas and metadata | Schemas · Databases and instances · Projects and database groups · Catalogs, changelogs and revisions · SQL review configs · Your own session and workspace facts | 31 READ methods: `DatabaseService` reads, projects, instances, database groups, catalogs, changelogs, revisions, review configs, session facts |
| read | Read data by running queries | Run read-only queries; a request is refused whole if any statement is not a read · Query history · Saved queries and sheets | 9 READ methods: `SQLService/Query`, query history (4), saved-query reads (3), `GetSheet` |
| read | Read the change workflow | Issues and comments · Plans and plan checks · Rollouts, task runs and logs · Releases · Rollback previews | 16 READ methods: issue, plan, rollout and release reads |
| — | *Read-only stops here* | | 56 methods |
| write | Propose changes | Create and edit sheets, plans and issues · Run plan checks and reviews · Create, edit and delete releases and revisions · Generate schema diffs. An agent never approves its own change; the project's approval policy decides whether a human must | 22 WRITE methods: sheet, plan, issue, release and revision writes, `RequestIssue`, `RunReview`, `DiffSchema`, `DiffMetadata` |
| write | Run rollouts and tasks | Create a rollout · Run, skip or cancel its tasks, under the project's approval policy | 4 WRITE methods: `CreateRollout`, `BatchRunTasks`, `BatchSkipTasks`, `BatchCancelTaskRuns` |
| write | Run DML and DDL statements | INSERT, UPDATE, DELETE, CREATE, ALTER, DROP through queries, where the engine checks each statement and the user may run it | Not a method: the statement clamp in `mcp_sql_clamp.go`, which Read-write lifts |
| write | Export query results | Download results as a file. Data leaves Bytebase | 1 WRITE method: `SQLService/Export` |
| write | Manage database housekeeping | Sync instances and databases · Database settings and labels · Database groups · Saved queries | 14 WRITE methods: sync, `UpdateDatabase`, database groups, saved-query writes |
| — | *Read-write stops here* | | 97 methods |
| floor | Never, in any mode: approve issues, administer the workspace, or handle credentials. | | 121 methods: 35 FORBIDDEN, 86 EXCLUDED |

Two choices in the wording are deliberate. Row 2 says *Read data by running queries* rather than
"Run queries" so the verb stays Read and the sub-item carries the rule that keeps it true under
Read-only. Row 6 is not a method at all: it names the statement clamp being lifted, because that is
a real difference between the modes that no method name shows, and a future custom policy will want
it as its own switch.

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

Per row: *Propose* over Create (Create fits the method names but hides that a human still
approves); *Export* over Read or Download (Read hides the egress hazard, which is why Export is
WRITE-class in the first place); *Manage* over Update or Maintain (Update reads as a data change,
Maintain is vague). Rollouts and DML/DDL both take *Run* and stay two rows, because a custom policy
will want "run approved rollouts, but no ad-hoc SQL" — the workflow the product exists for.

The tier is named once, by the dividers, so no row repeats it in its title.

## Decisions

**D1 — Placement: a disclosure inside the Access policy section, collapsed by default.**
Not a drawer (a second surface, one mode at a time) and not a separate section (it belongs to the
policy it describes). One trigger line, "What Read-only allows · Read schemas, data and the change
workflow; nothing is written or exported", expands inline into the list. In view state it sits
under the mode sentence; in edit state it sits under the three mode cards, above the masking
toggle, so the cause and its effect are adjacent. The open state persists per browser and carries
across the view-to-edit transition within a page visit.

**D2 — Content: rows and sub-items only.** No method names, no permission names, no counts, no
link to a method list. The floor is kept as one short line because it is what a security-minded
admin reads first, and it is what keeps the list honest about the 121 methods no mode reaches.
There is no constants line under the list; the facts it would carry (capped by the user's own
permissions, refusals audited) move into the section description.

**D3 — The mode chip is the subject; the "In force" line goes.** The three-stripe icon and the
words "In force" are removed. View state reads: the mode chip (success for Read-only, warning for
Read-write, destructive for Disabled), the "Masking exemptions ignored" chip when set, Edit policy
on the right; then the mode's one sentence; then the disclosure. "Active" was considered and
rejected as the label: "Active · Disabled" contradicts itself, and a chip under a section titled
Access policy needs no label. The chip carries `aria-label="Current policy: Read-only"`. The icon
also leaves the mode cards, so color appears in exactly two places: the chip in view, the selected
border in edit.

**D4 — The two notes under the card move.** "A ceiling change applies to the next request…" shows
only while editing, and it names the change when the form is dirty: "Changing Read-only to
Read-write applies to the next request of every running session; work already admitted runs to
completion." "MCP policy denials are recorded in the audit log" is a fact about the feature, not
about the current state, so it joins the section description: "The most any MCP session may do in
this workspace. Every session is also capped by the connecting user's own permissions, and every
refusal is recorded in the audit log."

**D5 — Edit state renders the pick, and only the pick.** The disclosure under the cards shows
exactly what the view would show after saving. No "adds N over Read-only", no added or removed
marks, no comparison against the stored mode: the muted rows already show what a pick does not
serve. The mode cards lose their visible radio; the card is the control, with the selected card
carrying the accent border and a faint accent tint, hover lightening the border, and keyboard focus
drawing the shared ring on the card. The radio group semantics stay.

**D6 — Marks: ✓ or —, plus a tier tag on served rows only.** A served row shows ✓, a `success`
"read" badge or a `warning` "write" badge after its title, and full-contrast text. An unserved row
shows —, muted text, and no badge. Under Read-only the write rows are unserved, so a "write" badge
never appears there.

**D7 — Disabled.** In view, the chip and the sentence "No MCP session can connect to this
workspace" are the whole answer; no disclosure. In edit with Disabled picked, the disclosure slot
holds a static line in the floor's soft error tone, "Disabled · nothing, no MCP session can
connect", so red means "no capability" everywhere on the card. It is not a button.

**D8 — Copy that shrinks.** The Read-only card drops to two sentences plus its "Best for" line:
"Sessions can explore schemas and run read-only queries. Every statement is checked before it
runs." The classifier and driver-depth detail it carried moves to docs; the "refused whole" rule
lives in row 2's sub-item. The masking toggle's two paragraphs become one sentence: "If enabled, a
user's masking exemptions and access-grant unmasks do not apply in MCP sessions; masked data stays
masked. The console is unaffected."

**D9 — The Authentication Required alert on the MCP page is removed.** Its two sentences already
exist on the page: "approve access in the browser" in the Connect a client description, and
"appropriate permissions" in the section description. Connect a client gains the one clause that
was useful: "Add Bytebase to your AI client and start asking. On first connection you sign in and
approve access in the browser."

**D10 — The consent page uses the row titles.** "This session may" lists the served rows with ✓,
one ✕ line for the unserved tier under Read-only ("No changes, rollouts or exports"), then the
existing capped, masking and audit lines. Read-write keeps its caution. One wording table serves
both surfaces.

**D11 — A backend lint binds the wording to the classification.** A table in
`backend/api/v1/mcp_gate.go` assigns every served method to exactly one row, and a lint in
`mcp_gate_test.go` holds it: every READ or WRITE method is in a row, no FORBIDDEN or EXCLUDED
method is, the read rows are exactly the READ set and the write rows exactly the WRITE set. A
class change that moves a method across tiers then fails CI until the table — and therefore the row
wording — is reviewed. No proto change, no new API, no generated file for the frontend: the product
shows no counts, so the frontend needs only the static tier of each row.

## States

| State | What the section shows |
|---|---|
| View · Read-only or Read-write | Chip line with Edit policy; the mode sentence; the disclosure, collapsed. Nothing below it. |
| View · Disabled | Chip line; "No MCP session can connect to this workspace." No disclosure. |
| View · unreadable, unserved, read failed | The existing warning or error, unchanged. No disclosure. |
| Edit · Read-only or Read-write picked | Three selectable cards; the disclosure for the picked mode, collapsed by default, rendering the post-save view; masking toggle; separator; footer sentence (naming the change when dirty), Cancel, Save (enabled only when dirty). |
| Edit · Disabled picked | As above, with the disclosure slot holding the static soft-error line. |
| Consent page | Served row titles with ✓, the ✕ line under Read-only, then the existing constants and caution. |

## Copy

All strings, so the change and the locale files have one source. Keys under
`settings.mcp.ladder.*` are new; the rest replace existing `settings.mcp.*` and
`oauth2.consent.mcp.*` values.

- Section description: "The most any MCP session may do in this workspace. Every session is also
  capped by the connecting user's own permissions, and every refusal is recorded in the audit log."
- Trigger: "What {mode} allows". Summaries — Read-only: "Read schemas, data and the change
  workflow; nothing is written or exported". Read-write: "Read everything, and propose, run, export
  and manage". Disabled (static): "Disabled · nothing, no MCP session can connect".
- Row titles and sub-items: the table above, verbatim.
- Dividers: "Read-only stops here", "Read-write stops here".
- Floor: "Never, in any mode: approve issues, administer the workspace, or handle credentials."
- Tier badges: "read", "write".
- Read-only card: "Sessions can explore schemas and run read-only queries. Every statement is
  checked before it runs." Best-for line unchanged.
- Masking toggle: the one sentence in D8.
- Footer, clean: "Changes apply to the next request of every running session; work already
  admitted runs to completion." Dirty: "Changing {from} to {to} applies to the next request of every
  running session; work already admitted runs to completion."
- Connect a client: the sentence in D9.
- Consent: row titles; "No changes, rollouts or exports".

## Implementation

### Frontend

- New `MCPCapabilityLadder` beside `MCPAccessPolicySection.tsx`, props `mode`, `expanded`,
  `onToggle`. It derives the served set from the mode by tier: READ_ONLY serves the read rows,
  READ_WRITE both tiers, DISABLED none. No comparison logic.
- Disclosure behavior belongs in a shared primitive per the UX contract. There is no
  `Collapsible` in `frontend/src/components/ui/` today; add one wrapping Base UI's Collapsible
  (trigger with `aria-expanded`, panel region) rather than hand-rolling it in the feature.
- The ladder is an unframed region: the trigger row and the rows are separated by
  `border-block-border` hairlines, with no outer frame, so the change adds no card inside the
  section's existing framed card. (Design mocks draw a light border around the ladder for
  legibility; the implementation does not.)
- Marks use the shared `Badge` (`success` for read, `warning` for write). The floor line uses the
  `error` semantic tokens at low opacity; no raw palette colors, no `dark:` variants.
- `MCPAccessPolicySection.tsx`: remove the `Rows3` icon and the in-force string; render the chip as
  the sentence subject with the aria-label; move the audit sentence to the section description;
  show the tightening sentence only while editing and interpolate both modes when dirty; replace the
  masking copy; pass `radioClassName="sr-only"` to `RadioGroupItem` and verify the item shows the
  shared focus ring with the radio hidden.
- `MCPPage.tsx`: remove the Authentication Required alert; extend the Connect a client description.
- `MCPConsentCeiling.tsx`: replace the read, write and workflow lines with the row titles from the
  same table.
- Locale: keys under `settings.mcp.ladder.*` in all five locale files;
  `frontend/scripts/check-react-i18n.mjs` guards them.
- Tests: the served set per mode; the disclosure collapsed by default, opens, persists, and follows
  the pick; Disabled renders the static line in edit and nothing in view; the consent page renders
  the row titles; one e2e case that edits Read-only to Read-write, saves, and sees the chip change.
- Run `node frontend/scripts/check-ui-guideline.mjs`; the change must not add to the legacy
  baseline.

### Backend

- `mcpCapabilityRows` in `backend/api/v1/mcp_gate.go`: a map from row id to procedure names, next
  to `mcpRequestShapeRefusals`, which it resembles in shape and intent.
- Lint clauses in `mcp_gate_test.go`, each with a RED test that breaks one input: every served
  method in exactly one row; no refused method in any row; read rows equal the READ set; write rows
  equal the WRITE set.
- `TestMCPClassificationInventory` gains a Row column in `testdata/mcp_method_classification.md`,
  so a class or row change shows up as a reviewable diff.

## Out of scope

- A custom access policy. When it comes, the rows become checkboxes, the three cards become preset
  buttons that select a prefix, and a policy matching no prefix shows a Custom chip; the backend
  row table is what the gate would read. The ladder needs no redesign.
- A docs page listing the methods per row, generated from the inventory. Worth doing; not linked
  from the card until it exists.
- Per-engine read-only depth and masking coverage. The removed drawer showed both; they belong in
  docs, not on the policy card.
- The ceiling gate itself, the classification, and the consent flow's mechanics.

## Open questions

- The Access policy section is a framed card containing the mode cards, which predates the "no
  cards inside cards" rule. This change does not add nesting, but unframing the section is a
  natural follow-up.
- Whether the consent page should also show the sub-items. The proposal shows titles only, to keep
  the approval screen short.

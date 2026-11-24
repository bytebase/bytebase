# MCP access policy — show what a mode allows

Status: accepted · 2026-09-10 · revised 2026-11-24 for BYT-10597 (D1, D2, D4, D6, D10, D14, D15); the revision's frontend is implemented in this PR

The MCP settings page offers three modes (Disabled, Read-only, Read-write) and describes each in
one sentence. Since [#21324](https://github.com/bytebase/bytebase/pull/21324) removed the per-mode
method drawer, nothing on the page says what a mode actually serves, and an admin choosing between
Read-only and Read-write has nothing to compare beyond two sentences. The rule this doc lands on:
**show capabilities, not methods or permissions — for the mode in force or being picked, what it
allows and then what it does not allow, as row titles alone, collapsed by default inside the
existing Access policy section.** The page also gets shorter: the mode cards shrink to an
icon, a label and a three-word caption, with the selected mode's "Best for" under them; the
disclosure line is the mode's description, and what each row covers is in the docs. The product change
is frontend-only, with one comment added to the `mcp_method_class` annotation and its generated
output; keeping the row wording true to the method classification is a rule in `AGENTS.md`
rather than a lint (D11). A custom access policy is out of scope, but the list is shaped so that it
becomes that editor later without a redesign.

## Problem

The page today (`frontend/src/routes/workspace/mcp/MCPAccessPolicySection.tsx`) shows, in view
state, an "In force" line with the mode chip, the mode's description, and two notes; in edit
state, three mode cards and a footer. The consent page
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

A third came from review once the list had grown to sixteen rows (BYT-10597). Two dividers, "Read-only
stops here" and "Read-write stops here", split one list, and a separate red line said what no mode
does. A reader has to know the modes are nested to parse that, and under Read-only the boundary of
the mode in force sits mid-list, followed by a divider naming a mode that is not in force. Consent
screens elsewhere read the other way: what is allowed, then what is not. Under Read-write the list
was sixteen ticked rows, and "Show details" opened a long reference line under every one of them.

## Principle

> **A capability is a verb phrase an admin recognizes, standing for an exact set of served methods.
> A mode is shown as two groups of rows: what it allows, then what it does not allow. The rows keep
> one order, and because the modes are nested, each mode's allowed group is a prefix of it.**

What follows from it:

- The unit shown is a capability row, never a method, a permission, or a count.
- The rows are the same in view and in edit. Editing changes which rows fall in which group,
  nothing else.
- Rows a mode does not serve stay visible, muted in the second group, so comparing modes never
  needs a second surface.
- What no mode serves is one more row, always the last of the second group, so under Read-write it
  is the whole group.
- Nothing on the section is said twice. The mode name appears in the chip or the selector, so the
  group headings say "This policy"; the description is in the disclosure line; the audience in the
  "Best for" line; what each row covers, in the docs.

## The rows

Twelve rows: eleven capability rows, four read and seven write, and a twelfth that no mode serves,
called the floor below. Each has a title, which is all the
product shows, and a list of sub-items in plain words, which the MCP docs page publishes under the
row's title (D2). A write row merged from two earlier rows (D15) lists each part's sub-items under
the part's own verb. The right column records which classified methods the row's wording has to
cover; it is documentation for whoever rereads these rows after a classification change, and never
appears in the product.

| Tier | Row | Sub-items, published in the docs | Classified methods the wording must cover |
|---|---|---|---|
| read | Read schemas and metadata | Schemas · Databases and instances · Projects and database groups · Catalogs, changelogs and revisions · SQL review configs · Your own session and workspace facts | 31 READ methods: `DatabaseService` reads, projects, instances, database groups, catalogs, changelogs, revisions, review configs, session facts |
| read | Read data by running queries | Run read-only queries · Under Read-only, a request is refused whole unless every statement is shown to be a read that returns data; how deeply that can be checked varies by engine, and on some engines no statement can be shown to be a read at all · Your own query history · Saved queries and sheets you have access to | 9 READ methods: `SQLService/Query`, query history (4), saved-query reads (3), `GetSheet` |
| read | Read the change workflow | Issues and comments · Plans and plan checks · Rollouts, task runs and logs · Releases · Rollback previews | 17 READ methods: issue, plan, rollout and release reads |
| read | Read members, roles and policies | Who holds which role in the workspace and its projects · Groups and their members · Roles and their permissions · Masking rules and exemptions · Rollout, data query, tag and review policies · Approval rules, environments, semantic types and data classification · Access grants, including pending requests · CI identities and who can sign in as them | 15 READ methods: `GetIamPolicy` on projects and the workspace, group reads (3), role reads (2), `GetPolicy`, `ListPolicies`, access-grant reads (3, with `SearchMyAccessGrants`), workload identity reads (2), and `GetSetting`, which the gate serves for those four settings only |
| — | *Read-only serves the rows above* | | 72 methods |
| write | Propose and roll out database changes | *Propose:* Create sheets · Create and edit plans and issues · Run plan checks and reviews · Create, delete and restore releases · Create and delete revisions · Generate schema diffs. The project's approval policy decides whether a change needs approval · *Roll out:* Create a rollout · Run, skip or cancel its tasks, under the project's approval policy | 26 WRITE methods. Propose: 22 WRITE methods: sheet, plan, issue, release and revision writes, `RequestIssue`, `RunReview`, `DiffSchema`, `DiffMetadata`. Roll out: 4 WRITE methods: `CreateRollout`, `BatchRunTasks`, `BatchSkipTasks`, `BatchCancelTaskRuns` |
| write | Run DML and DDL statements | INSERT, UPDATE, DELETE, CREATE, ALTER, DROP through queries, as far as the engine and the user's own permissions allow · Where the user's MCP client supports confirmations, a query that is not a read as far as the engine can check it runs only after the user confirms it · In other clients such a query can run without asking | Not a method: the statement clamp in `mcp_sql_clamp.go`, which Read-write lifts. A statement Bytebase cannot show is a read runs only through the `query_database` tool, which asks the user to confirm it when the client can show the form (`mcp_sql_write.go`) |
| write | Request access grants and export query results | *Request:* Ask for a time-boxed grant to run a specific statement, optionally with results unmasked or exported; the statement must be a read as far as the engine can check it · The grant starts only after it is approved, and a request that would need no one's approval is refused · A grant applies only where the project allows access grants · Revoke an active grant · Turn a project's "Allow access grants" on or off, after the user confirms · *Export:* Export results as files. Data leaves Bytebase | 4 WRITE methods. Request: 3 WRITE methods: `AccessGrantService/CreateAccessGrant` and `RevokeAccessGrant`, served like any WRITE method; and `ProjectService/UpdateProject`, served only through the `set_just_in_time_access` tool, which asks the user to confirm and changes only that setting. Export: 1 WRITE method: `SQLService/Export`, which requires the user's own `bb.sql.export` on the data it reads, from a role or a matching export grant, and runs only statements Bytebase can show are reads |
| write | Approve or reject issues and manage approval rules for database changes | *Approve:* Each decision is confirmed in the user's MCP client before it is made · Role requests are decided in the console · The approval rules and the project's self-approval setting apply, as in the console · *Approval rules:* Add, remove, reorder or change the rules that decide who approves a database change · Each change is confirmed in the user's MCP client before it is made · The fallback rules and the rules for other kinds of issues stay as they are | 3 WRITE methods. Approve: 2 WRITE methods: `ApproveIssue`, `RejectIssue`, served only through the `decide_issue` tool, which asks the user to confirm each decision and refuses role requests. Approval rules: 1 WRITE method: `SettingService/UpdateSetting`, which the gate serves only with a confirmation that `administer_workspace` attaches, for the approval rules alone (D13) |
| write | Set masking rules and grant or revoke exemptions | *Masking rules:* Add, remove, reorder or change the rules that decide which columns are masked and how · Each change is confirmed in the user's MCP client before it is made · Semantic types stay as they are · *Exemptions:* Exempt one member from masking for a database, a schema, a table or a column, or for every database in a project, with an optional expiry · Each exemption and each removal is confirmed in the user's MCP client before it is made · Exempted data shows unmasked in query and export results | 1 WRITE method, `OrgPolicyService/UpdatePolicy`, served two ways. Exemptions: served only through the `grant_masking_exemption` and `revoke_masking_exemption` tools, which ask the user to confirm, then add one exemption entry or remove one member from the entries whose condition matches exactly; other policy types and `CreatePolicy` and `DeletePolicy` stay refused. Masking rules: served for the workspace masking rule policy only through `administer_workspace`, which asks the user to confirm. On a workspace with no rules stored, `allowMissing` makes the handler create the policy; `CreatePolicy` itself stays refused |
| write | Add instances to projects and create CI identities | *Instances:* Add a PostgreSQL instance to a project, signed in with a user name and password · Each instance is confirmed in the user's MCP client before it is added · Its databases join that project for good · *CI identities:* Create a workload identity that GitHub Actions or GitLab CI jobs in one repository sign in as, with the GitOps Service Agent role in one project, so it can roll out without approval · Each identity is confirmed in the user's MCP client before it is created · This policy does not limit what it does, and it lasts until it is removed | 3 WRITE methods. Instances: 1 WRITE method: `InstanceService/CreateInstance`, which the gate serves only with a confirmation that `administer_workspace` attaches for the new instance; the tool confirms only a PostgreSQL instance in a project that signs in with a password (D13). `UpdateInstance`, whose `allow_missing` also creates an instance, stays EXCLUDED with the other instance writes. CI identities: 2 WRITE methods: `WorkloadIdentityService/CreateWorkloadIdentity` and `ProjectService/SetIamPolicy`, which the gate serves only with the confirmations `administer_workspace` attaches: one for the new identity, and one for the project whose policy then gains it, in the GitOps Service Agent role (D13). The tool takes the provider's own issuer and a pattern that admits every job of one repository. `UpdateWorkloadIdentity` stays FORBIDDEN, and the other identity writes and `WorkspaceService/SetIamPolicy` stay EXCLUDED |
| write | Sync instances and databases, and manage saved queries | Sync instances and databases · Saved queries | 9 WRITE methods: sync, saved-query writes |
| — | *Read-write serves the rows above* | | 118 methods |
| floor | Raise this policy, create a service account key, change a person's password, email or MFA, show a stored secret, open an Admin mode session, or read anyone else's query history | | 104 methods: 32 FORBIDDEN, 72 EXCLUDED |

Every row is displayed under every mode — served, or muted with a `—` — and against every engine a
workspace happens to hold, so **every line on this card must be true on both axes: mode and engine.**
That covers every row, and equally the collapsed summary, the card captions and the "Best for"
lines, which are not rows and were missed the first time the rule was applied. The summary is the
line most admins read and often the only one. A
claim whose truth depends on the mode names the mode as a condition; a claim whose truth depends on
the engine says so without naming engines, because per-engine depth is out of scope for this card
(see below) and an enumeration goes stale the release a driver changes.

Both axes have caught lines here, and the engine axis caught the same line twice. On the mode axis,
the read-only clamp holds only under Read-only (`mcpReadOnlyClampApplies`), so without its qualifier
row 2 promised a whole-request refusal under Read-write while the DML and DDL row on the same screen
offered INSERT and DROP.

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

Dropping it unconditionally was the next round's defect, and the two rounds after that are the more
useful half of this entry. "This mode does not require a statement to be a read" is what the MCP
ceiling does — the clamp is off — but not what the product does: `Query` runs `validateQueryRequest`
under every mode for engines outside `EngineSupportQueryNewACL`, and 13 of those 14 have a
registered validator, so a Read-write session there is still refused a write. The repair for the
complement axis had removed the engine axis from the one line that carried it, on a screen that
renders no row details. **A line being fixed on one axis is still a line, and has to be rechecked
against every axis — the one being repaired crowds out the rest.**

**Method.** The replacement kept the engine axis and named a consequence — lifted where Bytebase can
check, refused where it cannot — and that is still one variable too few. `Query` and `Export` do not
share a gate: `Query` skips the check for the ten `EngineSupportQueryNewACL` engines, while `Export`
skips it only for MySQL. So on PostgreSQL under Read-write the same write runs through `Query` and is
refused through `Export`, and Export is a served row on the same card. The fourteenth engine breaks
it the other way: Databricks is outside the ACL set *and* registers no validator, so
`ValidateSQLForEditor` returns its permissive default, `GetQuerySpan` returns no spans, and the
access check falls back to `bb.sql.select` — a `DELETE` that runs unrefused, unclassified, and
authorized by a read permission. **The same statement, mode and engine can get different answers from
different RPCs; a claim about what runs has to name the operation as well as the engine.**

Four wordings later, the Read-write consent line stated the bound and stopped: "Capped by your own Bytebase permissions, and by what Bytebase supports for each database engine and each operation".
Both modes now share one bound line, which states the bound and names no consequence (D10).

Two choices in the wording are deliberate. Row 2 says *Read data by running queries* rather than
"Run queries" so the verb stays Read and the sub-item carries the rule that keeps it true under
Read-only. The DML and DDL row is not a method at all: it names the statement clamp being lifted,
because that is a real difference between the modes that no method name shows, and a future custom
policy will want it as its own switch.

Every sub-item names only what the served methods can do. Sheets are created, never edited: there
is no update RPC. Releases and revisions are created and deleted: `ReleaseService/UpdateRelease` is
classified WRITE but answers Unimplemented, and Revision has no update RPC, so "edit" would
advertise an operation that does not exist. The sync and saved-queries row, once titled "Manage
database housekeeping", also named database settings and
labels, moving a database between projects, and database groups. Those five methods
(`UpdateDatabase`, `BatchUpdateDatabases` and the three database-group writes) are EXCLUDED now,
for `REDRAWS_GOVERNANCE_BOUNDARY`: a database's project and environment decide which approval, SQL
review and access rules apply to a change, and which masking rules decide what a query returns; its
environment, its labels and the group expressions decide which databases a change that targets a
group runs on. An agent that could edit them would choose the checks on its own changes. Row 2
states the depth of the read-only check because only PostgreSQL, CockroachDB and Redshift open the
database session read-only; elsewhere the clamp is statement classification alone, and a statement
that classifies as a read can still call a function that writes. The proto calls the ceiling
"classifier-enforced, not proven", and the row says so in the admin's words.

Approval is stated as the project's policy, never as a promise. The backend requires an approved
issue before a rollout only when the project has `require_issue_approval` on and an issue is linked
(`backend/api/v1/rollout_service.go`); the MCP-origin guard there refuses an issueless rollout only
under that same flag, and an issue whose approval finding produced no template counts as approved
with no human acting. The console turns the flag on for new projects; the API default is off. A row
that said "of an approved change" would therefore promise more than the gate enforces, and the
wording an admin reads while choosing Read-write is the wrong place to be generous. What holds
unconditionally is that no approval decision is made over MCP without a confirmation in the client.
A Read-write session approves or rejects an issue only through the `decide_issue` tool. The tool
decides database changes and access requests. It refuses role requests, because approving one grants
a project role; they are decided in the console with the rest of access administration. It asks
the user through MCP form elicitation, inside the same tool call, and acts only on an accepted form
with the box checked. A client that cannot show the form gets the issue's console link. The gate
serves `ApproveIssue` and `RejectIssue` only to a request that carries the confirmation, so
`call_api` is refused. The workflow then decides as it does for a person: the step's role, and the
project's self-approval and last-editor settings. Bytebase records the decision as confirmed in the
client, not as proof that a person clicked: a hook in the user's own client can answer the form, and
that is the user's choice. Re-checking an issue's approval and activating an access grant directly
stay FORBIDDEN, because each can let a request pass with no approver deciding it. That is the half
the rows state. Whether MCP-originated rollouts should require approval regardless of the project
flag is a product question tracked separately (BOT-71), not one this doc decides.

Access requests are the exception, and the request row states the approval unconditionally. An
approved access grant is active at once, so an MCP session's request that would count as approved
on creation, with no approval template or one that names no approver, is refused
before any row is written (`refuseUnattendedMCPAccessGrant` in
`backend/api/v1/access_grant_service.go`). The finding is saved on the request's issue at
creation, so an approval-rule edit afterwards cannot change it. Every other way a request becomes
active goes through an approver: in the console, or through `decide_issue` with a confirmation in
the client. Activating a grant directly stays FORBIDDEN to the session. The approver can be the
requester where the project allows self-approval; the rows promise an approval, not a second
person.

The DML and DDL row states its confirmation with a condition, because it works unlike the confirmed
tools: `query_database` asks before a query that may write only in a client that can show the form,
and runs it at once in one that cannot, where `decide_issue` and the other confirmed tools refuse.
The confirmation covers queries only. A statement filed as an issue runs as a rollout task, under
the project's approval policy, as the rollouts row says.

The members row shows who holds which role, the groups and their members, each role's permissions
and every policy type, so an agent can tell which people an approval step waits on. Two choices in
its wording are deliberate. It says *members*, not users: `UserService` reads stay EXCLUDED, so a
member is the email in a binding, not an account record. And it names what it serves rather than
"other policies" or "settings". `GetSetting` serves four settings by name: the approval rules behind
the project's approval policy, which the propose and rollout rows speak of, and the environments,
semantic types and data classification. The gate refuses every other setting name, and
`ListSettings` stays EXCLUDED because it returns them all. An MCP read of the semantic types leaves
out the MD5 masking salt.

The row states no bound of its own; the section's shared line limits every session by the user's
Bytebase permissions. That line has to say permissions rather than what the console shows, because
the two differ at one edge: the console reads a policy only through `GetPolicy`, which asks for
`bb.policies.getMaskingExemptionPolicy` or `bb.policies.getMaskingRulePolicy` on those two types,
while `ListPolicies` asks only for `bb.policies.list` and returns every type. Every built-in role
that holds `bb.policies.list` also holds both masking permissions, so the edge reaches custom roles
only. An access grant carries the statement its requester asked to run, which is SQL another person
wrote; the floor's query-history promise still holds, because it rests on `SearchQueryHistories`
pinning the creator, which this row does not touch.

### The verb rule

The verb names the row's effect on the customer's databases. *Read*: none. *Propose*: none until it
is approved. *Request*: none until it is approved, and then access to one statement's results,
unmasked or exported if asked for. *Approve*: none directly; the issue proceeds: a change rolls out
as the project's rollout policy allows, and an access request takes effect. Role requests stay in
the console. *Grant*: direct and standing: an exempted member sees the covered data unmasked in
query and export results until the exemption ends or is revoked. *Run* and *Roll out*: direct. *Export*: data leaves the product. *Manage*:
configuration only, no data path. Within a tier the verb repeats; it changes only where the effect
changes. A row merged from two parts (D15) keeps each part's verb, so "Set masking rules and grant
or revoke exemptions" says *Grant* where "Manage masking rules and exemptions" would hide the
standing unmask behind a configuration verb.

One verb for the whole write tier was tried and rejected:

- **Write** for every write row: "Write query results" (export writes nothing to a database), "Write
  change proposals" (nothing reaches a database until approval), "Write rollouts" (not English),
  "Write database housekeeping" (reads as a data change).
- **Change** for every write row: fails for exports and rollouts, and makes "Change proposals" and
  "Change data with SQL" read as the same thing — the distinction the approval step exists to
  protect.
- **Manage** for every write row: hides the hazard on exports and on SQL.

Per row: *Propose* over Create (Create fits the method names but hides that the project's approval
policy may still require approval); *Export* over Read or Download (Read hides the egress hazard,
which is why Export is WRITE-class in the first place); *Manage* over Update or Maintain (Update
reads as a data change, Maintain is vague); *Request* over Propose (a proposal waits for approval
only when the project's approval policy says so, a session's request always does, and what an approved request
gives is access to data, not a change); *Approve* stays out of the propose row,
because deciding someone's proposal or request is a different effect from filing one; it shares a
row with the approval rules, which decide who may approve. Rollouts join proposals in one row and
DML/DDL keeps a row of its own, because a custom policy will want "run approved rollouts, but no
ad-hoc SQL" — the workflow the product exists for — and that line falls between two rows.

The tier is named by the tag on each allowed row, so no row repeats it in its title.

## Decisions

**D1 — Placement: a disclosure inside the Access policy section, collapsed by default.**
Not a drawer (a second surface, one mode at a time) and not a separate section (it belongs to the
policy it describes). Collapsed, the trigger line *is* the mode's description, the summary under
Copy. The read half is the same phrase in both modes, because Read-write serves exactly the read
rows Read-only serves; the EXCLUDED reads
(audit logs, users, other people's query history, task-run sessions) are
refused in every mode, so no summary may say "read everything". The line never repeats the mode
name, which the chip (view) or the
selector (edit) already shows. Expanded, it becomes the first group's heading, "This policy
allows" (D14). In view state it sits under the chip line; in edit state it
sits under the selector and its "Best for" line, so the cause and its effect are adjacent. The open
state carries across the view-to-edit transition within a page
visit, and starts collapsed on the next one. Remembering it per browser was cut in review as more
machinery than a settings page visited this rarely earns.

**D2 — Content: row titles only; what each row covers is in the docs.** Expanded, each row shows
its title, and an allowed row its tier tag, nothing else. Under the groups, one link, "What each
capability covers", opens the MCP docs page at its "What each mode allows" heading
(`https://docs.bytebase.com/integrations/mcp#what-each-mode-allows`), whose table lists each row's
sub-items (The rows, middle column). No method names, no permission names, no counts. The row no mode serves is
the last row of the second group (D14), because it is what a security-minded admin reads first and
what keeps the list honest about the 104 methods no mode reaches. There is no constants line under
the list; the facts it would carry (capped by the user's own permissions, what the audit log
records) are the section description (D4).

The first version put the sub-items behind one "Show details" toggle for every row at once
(per-row expanders were rejected as fiddly). BYT-10597 removed it: opened, it put a long reference
line under each of sixteen rows, and an admin choosing a mode needs the titles, while the
sub-items are reference for the few who audit a row. Their caveats do not disappear from the
product's promises: the titles are bounded once by the section description (and on the consent
page by the same line, D10), and the docs carry the rest. Three sub-items are caveats rather than
reference — a write query runs without asking in a client that cannot show confirmations, a CI
identity is not limited by this policy and lasts until removed, an added instance's databases join
the project for good. The first goes on the consent page, in the Read-write caution (D10), because
it is what the person approving most needs: a write can run with no prompt and no review. The
other two move to the docs with the rest. On the settings card they are one link away, and the card
does not say that a CI identity or an instance's project membership outlasts a later downgrade;
that is accepted with the move.

**D3 — The mode chip is the subject; the "In force" line and the mode sentence go.** The
three-stripe icon and the words "In force" are removed. View state reads: the mode chip (success
for Read-only, warning for Read-write, destructive for Disabled), Edit policy on the right; then
the disclosure line, which carries the description.
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
session's next request." The note on what the audit log records is a fact about the
feature, not about the current state, so it joins the section description. It claims only what
the database stores per [audit-what-produces-a-row.md](audit-what-produces-a-row.md): calls to
audited methods that reach their handler. Schema and metadata reads are unaudited, and a call that
the ceiling gate or a permission check refuses before its handler is only streamed, when audit
logging to stdout is on. So the line
names what is stored, never an unqualified "MCP operations are recorded": "Queries, exports,
proposed changes and rollouts are recorded in the audit log." Where refusals go is in the docs. The section description is the bound line followed by this line, and the
consent page shows the same two (D10).

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
stored mode: the muted rows already show what a pick does not serve. The edit state is about 90
words collapsed and 140 expanded, down from about 215 and 375 in the first draft of this design.

**D6 — Marks: ✓ or —, plus a tier tag on served rows only.** A served row, in the first group,
shows ✓, a `success` "read" badge or a `warning` "write" badge after its title, and full-contrast
text. An unserved row, in the second group, shows —, muted text, and no badge; the row no mode
serves is marked like any other. Under Read-only the write rows are unserved, so a "write" badge
never appears there.

**D7 — Disabled.** In view, the chip and the sentence "No MCP session can connect to this
workspace." are the whole answer; no disclosure. In edit with Disabled picked, the Disabled card is
selected, the "Best for" line reads "keeping MCP off until you are ready to turn it on", and the
disclosure slot holds a static line in the soft error tone of the — marks, "Nothing is allowed; no MCP
session can connect.", so red means "no capability" everywhere on the card. It is not a button and
does not repeat the mode name.

**D9 — The Authentication Required alert on the MCP page is removed.** Its two sentences already
exist on the page: "approve access in the browser" in the Connect a client description, and
"appropriate permissions" in the section description. Connect a client gains the one clause that
was useful: "Add Bytebase to your AI client and start asking. On first connection you sign in and
approve access in the browser."

**D10 — The consent page uses the same rows in the same two groups, and bounds them once.** The
card's label above the mode chip is "Workspace policy", as on the Disabled and unreadable consent
cards, since the chip shows the workspace's policy as it is now. Under the chip, "This session may" lists the served rows with ✓ and "This session may not" the rest,
ending with the row no mode serves, exactly as the settings page groups them (D14). Then the bound
line and the audit line, the same two sentences the settings section description is built from.
Read-write keeps its caution, with one clause added (Peter Zhu, 2026-11-24): "An AI session with
write access makes real changes. In clients that can't ask for confirmation, write queries run
without asking. Approve only on a machine you control." Its mode chip carries the same icon as the settings page's. One
wording table serves both surfaces. The headings name the session rather than the policy, because
the person approving is deciding about their own session; there is no docs link, so the approval
screen stays one decision.

Titles alone carry no caveats, so on a workspace whose engine refuses every statement, an
unqualified "Read data by running queries ✓" promises a capability the session does not have.
Neither surface carries a detail line per row, so the whole list is bounded once, on the line that
already limits it by the reader's own permissions: "Every MCP session is limited by the connected
user's Bytebase permissions and the target database engine's supported operations." It states the
bound and stops, in both modes. A first version gave each mode its own bound and named a consequence
for Read-only; the consent page and the settings page were unified on this one line in
[#21517](https://github.com/bytebase/bytebase/pull/21517), and every consequence clause written for
Read-write had been false once (see the complement and method axes in The rows). The ✓ marks read as
what the policy admits rather than what will succeed. The bound line carries a neutral glyph and no
"Allowed" mark, so it is not counted as a further grant.

**D11 — The wording is bound to the classification by instruction, not by a lint.** The row
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

**D12 — MCP sessions follow the caller's own masking provisioning.** On 2026-09-23, Vincent Huang
and Peter Zhu decided to retire the "Ignore masking exemptions" toggle (`ignore_masking_exemptions`,
field 2 of `MCPSetting`) before the 3.23.0 tag. The toggle came from the MCP design work in
[#21216](https://github.com/bytebase/bytebase/pull/21216), and no customer stated a requirement for
it. The former D8 set the toggle's copy and is removed; its note on the Read-only description
sentence moved to Copy. An MCP session now applies the caller's own masking provisioning exactly as
the console does, including masking exemptions and the unmask carried by an access grant. There is
deliberately no setting that forces masking for MCP sessions. Field 2 is deleted without a
`reserved` entry, because no other field is expected to take the number. Stored rows need no
migration, because the store's unmarshaler discards the unknown key and the next save rewrites the
row without it. The masked-write guard stays, because it never depended on the toggle.

**D13 — Read-write changes workspace administration after a confirmation.** Under Read-write, the
agent can change the approval rules for database changes and the masking rules, add a PostgreSQL
instance to a project, and create a workload identity for CI with the GitOps Service Agent role in
its project, through `administer_workspace`, after the user confirms each change in the MCP client.
Approval and masking rules bound what a user's agent can change and read, an instance's databases
join its project for good, and a workload identity signs in outside this policy until it is removed
and can roll out without approval, so this is a trade: the confirmation checks
intent and is not a security boundary, since a hook in the user's own client can answer the form. An
admin who wants no agent to change a setting chooses Read-only. Vincent Huang decided on 2026-11-02
that admin changes are no more dangerous than the DDL and DML that Read-write already runs, and that
a user who does not trust the agent with a setting changes it in the console.

**D14 — Two groups, allowed then not allowed, instead of one list with dividers.** The rows split
by whether the mode serves them: "This policy allows" holds the served rows, "This policy does not
allow" holds the rest, and the row no mode serves is always the last row of the second group. Under
Read-only that is four rows, then seven write rows and the last row; under Read-write, eleven rows,
then the last row alone. Each group keeps the table's order. The headings say "This policy" because
the chip or the selector directly above already names the mode, and in edit state they describe
the mode being picked.

The first version drew one list of all sixteen rows with "Read-only stops here" and "Read-write
stops here" dividers, plus a red "Never, in any mode:" line under it. Rejected in BYT-10597: the
dividers ask the reader to know the modes are nested, put the boundary of the mode in force
mid-list, and under Read-only name a mode that is not in force. The red line was a third kind of
"not allowed" beside the — rows; once a group says what the policy does not allow, the line is one
more such row, and its label is dropped. Its text is unchanged, and it holds in every mode, so
switching modes in edit state shows it staying put. Comparing modes still needs no second surface:
picking the other mode moves rows between the groups.

**D15 — Write rows merge by the object they act on, from twelve to seven.** Two write rows merge
when they act on the same object or the same flow: proposing and rolling out a change; requesting an
access grant, which can ask for exportable results, and exporting results; deciding issues and
setting who decides them; masking rules and masking exemptions; adding an instance and creating a CI
identity, the two parts that connect a system outside Bytebase to a project. A row stays alone when an admin chooses or refuses
Read-write over it: running DML and DDL directly, outside any review. Housekeeping is renamed for
what it does, since "Manage database housekeeping" said nothing. A merged title names every part's
verb and object (the verb rule), so it covers each part's methods and "Keeping the rows true" applies
to it unchanged; the table keeps the parts apart, so a custom policy that needs one as its own
switch can split the row again.

The four read rows do not merge. They are short and parallel ("Read …"), and the one candidate pair
— the change workflow with members, roles and policies — makes a title that reads as two rows.

## States

| State | What the section shows |
|---|---|
| View · Read-only or Read-write | Chip line with Edit policy; the disclosure line as the description, collapsed on every visit (D1). |
| View · Disabled | Chip line; "No MCP session can connect to this workspace." No disclosure. |
| View · unreadable, unserved, read failed | The existing warning or error, unchanged. No disclosure. |
| Edit · Read-only or Read-write picked | Icon cards with the pick selected; the pick's "Best for" line; the disclosure for the pick, collapsed by default, rendering the post-save view, expanded as the two groups (D14) and the docs link; separator; footer sentence (naming the change when dirty), Cancel, Save (enabled only when dirty). |
| Edit · Disabled picked | Icon cards with Disabled selected; its "Best for" line; the static soft-error line in the disclosure slot (D7); separator; footer sentence (naming the change when dirty), Cancel, Save. |
| Edit · nothing picked | Only reachable from an unreadable or unserved ceiling: icon cards with no selection; "Pick a mode to save this policy." in the disclosure slot; Cancel, Save disabled. |
| Consent page | The mode chip; "This session may" with the served rows; "This session may not" with the rest, ending with the row no mode serves; the bound and audit lines; the caution under Read-write (D10). |

## Copy

The strings as decided, with the reasoning that picked them. The locale files are what ships and
what to edit; this section is the record of why, and a copy edit is expected to update both. Keys
under
`settings.mcp.ladder.*` are new; the rest replace existing `settings.mcp.*` and
`oauth2.consent.mcp.*` values.

- Section description: the bound line and the audit line, "Every MCP session is limited by the
  connected user's Bytebase permissions and the target database engine's supported operations.
  Queries, exports, proposed changes and rollouts are recorded in the audit log." The consent page shows the same two sentences (D10). The audit line replaces the shipped
  "MCP operations are recorded in Bytebase audit logs.", which claims more than is stored (D4).
- Disclosure line, collapsed — Read-only: "Read schemas, data, the change workflow, members, roles and policies; a request carrying anything that cannot be shown to be a read is refused, and nothing is exported".
  Read-write: "Read schemas, data, the change workflow, members, roles and policies; propose, request and revoke access, grant masking exemptions, approve, run, export and manage".
  Group headings: "This policy allows", "This policy does not allow". Docs link: "What each
  capability covers".
- Disabled — view sentence: "No MCP session can connect to this workspace." Edit static line:
  "Nothing is allowed; no MCP session can connect."
- Card captions: Disabled "No sessions connect"; Read-only "Explore and query"; Read-write "Change
  data and schemas".
- "Best for" lines, unchanged: Disabled "keeping MCP off until you are ready to turn it on";
  Read-only "querying and exploring data, including by people who do not write SQL"; Read-write
  "making database changes through an AI agent, still capped by each user's own permissions".
- The Read-only description sentence ("Sessions can explore schemas and run read-only queries…")
  is retired everywhere; its content lives in the Read-only summary and in row 2's sub-item.
- Row titles: the table above, verbatim. Sub-items: the docs page, from the same table.
- The row no mode serves: "Raise this policy, create a service account key, change a person's
  password, email or MFA, show a stored secret, open an Admin mode session, or read anyone else's
  query history". Until BYT-10597 it was a separate line led by "Never, in any mode:" (D14).
  "Approve issues" left the floor when approving and rejecting moved to the confirmed tool.
  "Administer the workspace" and "handle credentials" left it when `administer_workspace` began
  changing workspace administration after a confirmation (D13). The line no longer covers every
  denial reason: it names, item by item, what no mode does. Raising this policy is one of them,
  because the gate serves `UpdateSetting` only with a confirmation, and no tool attaches one for the
  MCP setting. The verbs claim only
  what is refused **whatever the caller's permissions** and by **whatever effect** a served method
  has.
  That second rule cost three drafts. MCP's ceiling removes methods by class; it never adds
  per-row privacy, so an ownership claim — "never read other people's SQL", then
  "never browse everyone's saved SQL" — is false at whatever permission level makes it true in the
  console: `GetSavedQuery` reaches SQL shared by a binding, and `searchScope` drops all scoping for
  a caller holding project-wide `bb.savedQueries.get`. Query history is the one that qualifies,
  because `SearchQueryHistories` pins the creator and nothing widens it. A fourth draft added "or
  send data to a third party" and was false on the effect axis: `CreateIssue` is served under
  Read-write and its webhook posts the title and description outward. It is not replaced by a row
  caveat — the webhook fires identically for an issue a human files in the console, so it is a
  property of the workspace's notification config rather than of this ceiling, and the propose
  row already says the session can file issues. The fifth draft read "open an admin
  connection" and was false on the same axis for the opposite reason — it named the plumbing rather
  than the feature, and MCP does reach the admin data source. It names Admin mode now.
- Tier badges: "read", "write".
- Footer, clean: "Applies to every running session's next request." Dirty: "{from} → {to} applies
  to every running session's next request."
- Connect a client: the sentence in D9.
- Consent: group headings "This session may", "This session may not"; row titles; the bound and
  audit lines of the section description; the Read-write caution with its added clause (D10).

## Implementation

The first version's implementation, then the BYT-10597 revision, which supersedes it where the two
differ: the details control, the dividers and the floor line are gone.

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
  and interpolate both modes when dirty.
- `MCPPage.tsx`: remove the Authentication Required alert; extend the Connect a client description.
- `MCPConsentCeiling.tsx`: replace the read, write and workflow lines with the row titles from the
  same table.
- Locale: keys under `settings.mcp.ladder.*` in all five locale files. The retired mode-description
  keys are removed, not left empty. `frontend/scripts/check-i18n.mjs` enforces cross-locale
  parity over all of them; the four template-keyed families (`ladder.row.`, `.stops.`, `.summary.`,
  `.tier.`) are registered in its `DYNAMIC_PREFIXES`, which exempts them from both the missing-key
  and unused-key checks, as it does for every other template-keyed family in the product. A row id
  with no copy renders its raw key on the page, and the e2e spec, which reads every row's title from
  `en-US.json`, fails to load; an orphaned line after a rename is dead weight in the locale files. Both need a row renamed, which is rare, so neither
  earns a check of its own.
- Tests: the served set per mode; the disclosure collapsed by default, opens, follows the pick, and
  starts over on the next visit; the details toggle reveals sub-items on every row; the
  "Best for" line follows the selection; Disabled renders the static line in edit and the sentence in
  view; the consent page renders the row titles; one e2e case that edits Read-only to Read-write,
  saves, and sees the chip change.
- Run `node frontend/scripts/check-ui-guideline.mjs`; the change must not add to the legacy
  baseline.

### Backend

No behavior change. One comment on `MCPMethodClass` in `proto/v1/v1/annotation.proto` and its
regenerated output; the rule that keeps the row wording true is an instruction under Metadata and
API conventions in the root `AGENTS.md` (D11), not code.

### Revision for BYT-10597

Two PRs, in this order, because the card links to a docs anchor that has to exist when the
release ships.

1. **Docs (bytebase.com), `docs/integrations/mcp.mdx`.** Under Access policy, a "What each mode
   allows" table: one entry per row in The rows, titled as in the product, with its sub-items, and
   a column for the modes that serve it; the row no mode serves closes the table. The settings card
   links to that heading (D2). The table
   replaces the page's mode table. The "What no mode allows" list stays, and every line that a row
   contradicts, in either mode, is cut back to what is still refused. Read-write contradicts the
   lines on changing approval and masking rules, adding instances to projects, and creating CI
   identities with a project role; every mode contradicts the line on reading workspace settings,
   since the members row serves four of them. What those lines still refuse, such as workspace IAM
   and other workspace settings, stays. The same PR corrects the rest of the page where it
   trails the product: it has no lines for adding instances, creating CI
   identities, or changing approval or masking rules, and it says an agent cannot read the approval
   rules, which the members row now serves.
2. **This repo: this doc, then the frontend, in one PR.** It delivers D1, D2, D4, D6, D10, D14 and D15.
   - Rows: `mcpCapabilityRows.ts` lists the eleven capability ids of The rows in order, with their
     tiers. The floor is not one of them: `MCPCapabilityList` renders it from its own key,
     `settings.mcp.ladder.floor.text`, as the last row of the second group in every mode (D14), so
     the tier type stays read and write, and the rule that Read-write serves every listed row stays
     true.
   - `MCPCapabilityList` renders the two groups from the mode, with no `details` or divider
     options; `MCPCapabilityLadder` drops its details control and renders the docs link, and since its trigger
     is the first group's heading (D1), it passes only the second;
     `MCPConsentCeiling` drops its details control, labels its card "Workspace policy" and passes
     its two headings.
   - Locale, all five files: reword `ladder.heading` to "This policy allows"; add the "does not
     allow" heading, the consent "may not" heading and the docs link; reword `floor.text` to the
     floor row; reword the titles of The rows, the audit line (D4) and the Read-write caution
     (`oauth2.consent.mcp.write-caution`, D10); remove the `stops.*` family and its
     `DYNAMIC_PREFIXES` entry, `floor.label`, `show-details`, `hide-details` and every
     `row.*.details`.
   - Tests, named by what they assert: each mode's two groups (D14 counts, Read-only and
     Read-write); the row no mode serves is last in the second group in both modes; no details
     control on either surface; the consent page renders both headings; the settings card renders
     the docs link. `frontend/tests/e2e/workspace/workspace-mcp-policy.spec.ts` reads row titles,
     a row's details and the audit sentence from `en-US.json`, so it moves to the new ids and the new
     audit line, and asserts the groups instead of the details and dividers.

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
   `frontend/src/components/mcp/mcpCapabilityRows.ts`. Update the table above in the same change,
   and the docs table that publishes the sub-items (D2) in the docs repo.
3. Check the new wording on all five axes in The rows — mode, engine, effect, complement and
   method — against their rules: state the bound and never the behavior, name the feature and never
   the mechanism, and claim only what holds whatever the caller's permissions. A line you are
   changing to fix one axis still has to hold on the other four; every regression this page has
   shipped came from checking only the axis that prompted the edit.
4. For any claim of the form "refused where X", check `HasQueryValidator` and
   `EngineSupportQueryNewACL` as a conjunction rather than either alone, and check each RPC that
   executes statements separately — `SQLService/Query` and `SQLService/Export` gate on different
   conditions, so a sentence true of one is not thereby true of the other.
5. On every reclassification, even one an existing row already covers, check the floor row and the
   docs page's "What no mode allows" list against every row's sub-items in both modes, and cut back
   any line a served method now contradicts.

Both surfaces render row titles only, so a caveat a title needs in order to be true, one that holds
only for one mode or one engine, has to live in the bound line (D10) or in the title itself, never
only in a row's sub-items. A note on how a true capability behaves, such as when a confirmation is
asked, is not that kind of caveat; it lives in the docs (D2).


## Out of scope

- A custom access policy. When it comes, the rows become checkboxes, the cards become preset
  buttons that select a prefix, and a policy matching no prefix shows a Custom chip; the backend
  row table is what the gate would read. The ladder needs no redesign. The one precondition before
  rows become independently selectable is met: `SQLService/Export` runs only statements Bytebase
  can show are reads for an MCP session (`mcp_sql_write.go`). Export needed its own check because
  it skips statement validation on MySQL, and the driver executes non-query statements. The same
  split is why the bound line names no consequence: a cap stays true however each operation
  refuses (the method axis in The rows). A merged row splits again where a custom policy needs its parts as separate
  switches (D15).
- A docs page listing the methods per row, generated from the inventory. The card links to the
  sub-item table (D2), which names capabilities, not methods.
- Making the collapsed list findable by the browser's find-in-page. Base UI unmounts a closed panel,
  so the rows are absent from the DOM until the disclosure is opened, and
  `hiddenUntilFound` would keep them mounted instead. Declined: collapsing by default is D1's
  decision, the summary line states what the mode allows and the trigger is beside it, and the
  attribute's browser support is uneven — the unmount also keeps the rows out of the accessibility
  tree while they are not shown. Revisit if the list stops being a disclosure.
- Per-engine read-only depth and masking coverage. The removed drawer showed both; they belong in
  docs, not on the policy card.
- The ceiling gate itself, the classification, and the consent flow's mechanics.

## Open questions

- The Access policy section is a framed card, which predates the "no cards inside cards" rule.
  This change adds no nesting and removes the nested mode cards, so unframing the section is a
  natural follow-up.

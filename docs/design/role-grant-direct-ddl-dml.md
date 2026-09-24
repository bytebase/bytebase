# Role grant — the environment field is a permission, not a scope

Status: proposal · 2026-09-24 (revised after review the same day)

When a project owner grants a role that carries `bb.sql.ddl` or `bb.sql.dml`, the grant form shows
a field titled **Environments** between *Databases* and *Expiration*. Every neighbour narrows the
grant, so owners read this one the same way — "which environments does the role apply to" — and
tick all of them to make the grant complete. What the field actually does is the opposite: each
environment ticked lets the member run DDL and DML in SQL Editor there **immediately, with no
change plan and no approval**, and the empty field is the strict choice. A customer whose policy is
"every change goes through a ticket a DBA approves" found members with standing direct-DDL access
to production, granted by owners who thought they were scoping access. The amber warning between
the title and the picker says the truth and is not read, because people act on the control, not on
the paragraph above it. The rule this doc lands on: **the field is a permission you switch on,
never a scope you fill in.** Off is the default and the safe state; on names environments and says,
in the reader's words, what happens there. The change is frontend-only and touches the two grant
forms, the workspace member sheet, the member list, and the approver's issue page.

Two things called "approval" appear below and are not the same. A *change plan's* approval is what
a DDL/DML statement goes through when SQL Editor refuses to run it directly. A *grant's* approval is
what a role request goes through before the binding exists. The switch decides the first; the
approver's page shows the second.

## Problem

The two forms render the same field. In the members page
(`frontend/src/routes/workspace/MembersPage.tsx:1243`) and the self-service request sheet
(`frontend/src/modules/access-control/RequestRoleSheet.tsx:588`):

```tsx
{envKind && (
  <FormField title={<>{t("common.environments")}</>}>
    <DDLWarningCallout type="drawer" kind={envKind} />
    <EnvironmentSelect multiple portal value={environments} onChange={setEnvironments} />
  </FormField>
)}
```

`envKind` is `"DDL"`, `"DML"` or `"DDL/DML"` when the role carries the matching permission
(`frontend/src/lib/project-member/utils.ts:33`), so the field appears for Project Owner, SQL Editor
User, and any custom role with either permission (`backend/store/predefined_roles.go:346,505`).
The callout reads *"In the selected environments, DDL/DML statements can be directly run in SQL
Editor without approval."* The picker's placeholder is *"Select environment"*
(`frontend/src/locales/en-US.json:905`).

**The semantics are inverted relative to the neighbours.** *Databases* restricts where the role's
permissions apply. *Expiration* restricts when. This field adds a capability. An empty picker is
serialized as `resource.environment_id in []` — `RequestRoleSheet.tsx:342` passes the empty array
through, and `frontend/src/utils/issue/cel.ts:246` writes the clause whenever the array is defined.
The SQL service reads the clause's contents only when it checks `bb.sql.ddl` or `bb.sql.dml`, and
strips it before checking any other permission (`backend/api/v1/sql_service.go:2275-2298`), so the
empty field changes nothing about querying or requesting changes. (One other reader exists: the IAM
manager's project-wide check treats any binding whose condition names an environment — empty list
or not — as a scoped binding and skips it for project-wide saved-query permissions,
`backend/component/iam/manager.go:131-140`, `condition.go:21-35`. On and off are identical there;
only a clause-less binding differs.) Nothing on the form says that empty is safe, so the reader who
is unsure ticks everything.

**What "refused" looks like depends on the role.** When SQL Editor refuses a DDL/DML statement,
the change-plan hint (*"To execute DDL/DML statements in Prod, please submit a Data Change Plan for
approval."*, `frontend/src/modules/sql-editor/components/ExecuteHint.tsx:101`) appears only for a
member who also holds `bb.plans.create` and `bb.sheets.create`
(`frontend/src/hooks/useExecuteSQL.ts:390-401`, `frontend/src/utils/iam/permission-utils.ts:5-8`).
SQL Editor User carries neither (`predefined_roles.go:505-527`), and the result view suppresses the
permission panel for this error (`frontend/src/modules/sql-editor/components/ResultView/ResultView.tsx:98`),
so a member holding only that role sees a bare error with no path forward. Any copy that says
"goes through a change plan" is therefore true for Project Developer and above and false for the
role the field most often appears on.

**The gate is engine-conditional.** Only engines in `EngineSupportQueryNewACL`
(`backend/common/engine.go:87-101`) route DDL/DML to `bb.sql.ddl` / `bb.sql.dml`; on the others the
statement is checked as `bb.sql.select` (`sql_service.go:1634-1657`) and the clause is stripped. On
those engines the field is inert, on and off alike. This doc does not change that; the form does
not know the engine and should not pretend to.

**The role hides the power.** SQL Editor User is described as *"Permissions for querying database
data."* (`en-US.json:2278`; zh-CN: 可查询指定范围的数据), which never mentions DDL or DML. Its
read-only sibling says *"…without DDL or DML access"*, which by contrast makes the write role sound
read-only. Project Owner is *"All permissions within the project"*. A granter who has just chosen a
role that sounds like read access has no reason to slow down at a field called Environments.

**The decision used to belong to admins.** Until 3.15.0 the control was an environment-level
policy (`disallow_ddl` / `disallow_dml` on the environment's query-data policy) set by workspace
admins. [#19239](https://github.com/bytebase/bytebase/pull/19239) replaced it with the per-binding
condition and dropped the policy; the migration
(`backend/migrator/migration/3.15/0009##migrate_iam_binding_environment_condition.sql`) rewrote
existing bindings to carry the environments the policy had allowed, and left bindings alone in
workspaces where every environment allowed both. Since then, whoever can grant a role decides, per
grant, where direct DDL/DML is allowed — and the product tells them so only in the paragraph they
skip.

**The same field appears where someone asks for the role, and neither the approver nor the
approval rules can see what is being approved.** The request sheet is the same component, so a
requester who ticks every environment is asking for direct production DDL without knowing it. The
approver's page (`frontend/src/routes/project/issue-detail/components/IssueDetailRoleGrantDetails.tsx:120-134`)
shows the role, a box of raw permission ids, the same generic callout, and the environment titles
joined with commas; for an empty list it shows nothing at all, under a comment that calls the state
"degenerate" — the exact misreading this doc corrects, checked in. It never shows the grantee:
`roleGrant.user` is read nowhere in the frontend. The request sheet always sets it to the current
user (`RequestRoleSheet.tsx:409`), but the API accepts any existing user and never compares them to
the caller (`backend/api/v1/issue_service.go:692-705`), and approval writes that principal into the
binding (`backend/utils/utils.go:111-122`). The grantee is one principal — a user, a service
account, or a workload identity — never a group. The approval router is blind in the same way:
`buildCELVariablesForRoleGrant` (`backend/component/review/evaluator.go:794-860`) exposes the role,
the expiration and the *database* scope, and derives `resource.environment_id` from that scope or
from every workspace environment; the direct-execution clause never reaches a rule. "Direct DDL/DML
in Prod requires a DBA" cannot be written today; the coarse "every SQL Editor User request requires
a DBA" can, through `request.role`. A trap follows: `request.role == "roles/sqlEditorUser" &&
resource.environment_id == "prod"` compiles and fires on every such request in any project that
has a Prod environment, switch on or off, because that variable names the *database* scope, not
the switch.

## Principle

> **The field is a permission you switch on, never a scope you fill in.**
> Off is the default and the safe state, and the form says what off means — and what it does not
> mean. On reveals the environment picker, and every surface that shows the result — the form, the
> request, the approval, the member list — says in the reader's words what will happen, naming the
> environments it happens in.

What follows from it:

- A switch, not a picker, is the first thing the reader touches. Turning it on is the commitment;
  nobody ticks a switch "to be complete".
- The warning is a consequence of state, rendered after a choice, not a preface rendered before
  one. It names what was chosen.
- Every surface uses the same vocabulary and the same component, so the granter, the requester,
  the approver, and whoever reads the member list later see the same sentence.
- Copy describes this grant, never the member's effective access. Grants are additive: a binding
  that adds nothing cannot take anything away.
- Nothing about the stored binding changes. `resource.environment_id in []` stays the storage form
  of *off*; a list stays the storage form of *on*.

## Decisions

**D1 · The field is a Switch, off by default.** Title *Direct DDL/DML execution*; switch label
*Allow running DDL/DML in SQL Editor without approval*, with `{{kind}}` substituted as today (DDL,
DML, or DDL/DML). This is the UX contract's case for a `Switch`: one binary setting whose label
names the enabled behavior and whose off state has a meaningful result
(`docs/agents/frontend-ux.md:204-208`). Off renders one caption under the switch:

> Off: this grant adds no direct DDL/DML access. SQL Editor refuses DDL/DML unless another grant
> allows it; members who can create change plans are pointed to one.

The caption describes the grant, not the member: bindings are additive, the SQL service accepts
any matching binding (`sql_service.go:2261-2285`), and a group or workspace grant that already
allows direct execution is untouched by an off grant. Off submits exactly what an empty picker
submits today on both forms — the request sheet passes the empty list through
(`RequestRoleSheet.tsx:342`), and the members page does too, since an empty list still counts as a
condition there (`MembersPage.tsx:1556-1565`) — so no binding, migration, or backend path changes.

**D2 · On reveals the picker, labeled for what it is.** Sub-label *In these environments*, the
existing `EnvironmentSelect`, placeholder *Select environments*. The word *Environments* no longer
titles the field; a title has to name the effect, and this one is the dimension.

**D3 · On with nothing picked is an error, not a silent off — and the switch is form state.** The
field shows *"Pick at least one environment, or turn this off."* beside the picker and the form
cannot submit. On + empty would write the same binding as off, and the reader's intent is
ambiguous; asking is cheaper than guessing. Validation sits with the field, per
`frontend-ux.md:192`. Because on + empty and off are the same `string[]`, the switch cannot be
derived from the list: both forms hold `directExecution: boolean` beside `environments` in their
form state, and a role change resets both together (`MembersPage.tsx:1103-1111` and
`RequestRoleSheet.tsx:509-517` already reset the list). `DirectExecutionField` is controlled on
both values and owns no state, so a parent reset can never leave it on with an empty list.

**D4 · The callout follows the state and names the environments.** It renders only when the
switch is on and at least one environment is picked. Its content is a static lead, one
`EnvironmentLabel` chip per picked environment, and — when any picked environment carries the
protected tag — a second line naming those environments as protected. Chips, not prose: joining
titles into a sentence is fragile across locales (English *and*, Chinese 、), and the chip is what
the member list already renders. The protected line is gated by the environment-tier feature the
way `EnvironmentLabel` already gates its shield (`frontend/src/components/EnvironmentLabel.tsx:52`);
without the feature, neither appears. Plural forms use the locale files' existing `_one` / `_other`
keys.

**D4a · Chips summarize only conditions the forms could have written.** The decoder walks `||`
exactly as it walks `&&` (`cel.ts:398-401`), so a condition crafted through the API — say
`resource.environment_id in ["staging"] || true` — decodes to a Staging chip while the SQL service,
which evaluates the whole expression, allows every environment. The forms never emit `||`,
negation, or a raw string; a condition that contains any of them, or that fails to parse, is
*unrecognized*. `convertFromExpr` reports it as such, and every surface that would show chips shows
*Custom condition* with the raw expression instead — no chips, no protected line, no claim. This
matters most on the approver's page, where the new callout is what the approver relies on.

**D5 · One component, four leads.** `DDLWarningCallout` becomes `DirectExecutionCallout`, taking
`kind`, `environments`, and a `lead` variant:

| Surface | Lead |
|---|---|
| Grant form (members page) | Members with this role run DDL/DML straight from SQL Editor, with no change plan and no approval, in: |
| Request sheet | Once approved, you run DDL/DML straight from SQL Editor, with no change plan, in: |
| Approver's issue page | Approving lets *{grantee}* run DDL/DML straight from SQL Editor, with no change plan and no approval, in: |
| Member list, edit drawer | Runs DDL/DML in SQL Editor without approval in: |

The member list keeps its three binding states with the same vocabulary. *some* uses the lead
above with chips. *none* is an info alert — *"This binding adds no direct DDL/DML access"* — since
it is the benign state and describes the binding, not the member. *all* — a binding with no
environment clause — stays amber: *"Runs DDL/DML in SQL Editor without approval in every
environment; this binding has no environment scope."* Such bindings come from the workspace member
sheet, from the API, and from pre-3.15 workspaces the migration left alone. The workspace member
sheet (`MembersPage.tsx:1985-1993`) offers the same roles with no picker; when a selected role
carries the permission it renders the *all* callout, so the granter there is told what they are
handing out.

**D6 · The role says what it can do, on every surface it is shown.** SQL Editor User's description
becomes *"Query data in SQL Editor. Can also run DDL/DML there without approval, in the environments
the grant opens."*, and Project Owner's gains the same clause after *"All permissions within the
project"*. The sentence names no control, because the description also renders in the role
dropdown's option rows (`frontend/src/components/RoleSelect.tsx:59-75`), on the roles settings
page, and in the workspace member sheet, where there is no switch. Both project grant forms render
the selected role's description under the select — today neither form shows it once a role is
chosen, and the open dropdown omits it for custom roles, whose custom row renderer replaces the
description block (`RoleSelect.tsx:77-90`) — and
the pointer to the field ("…in the environments you open below") is a form-owned caption there,
not part of the description. Custom roles keep their author's description, which this change makes visible on the forms for
the first time; the presence of the switch is what tells the granter the role carries the
permission.

**D7 · Protected environments are flagged, not blocked.** Every picker row already renders
`EnvironmentLabel`, so the shield shows today (`frontend/src/components/EnvironmentSelect.tsx:44`).
The *Protected environment* text tag beside it needs `renderSuffix`, which the multi-select branch
does not expose (`EnvironmentSelect.tsx:14-15`); lifting it is the one change to that file. In the
callout the environment gets D4's second line. It stays selectable. There is no policy that closes
an environment to direct execution, and this doc does not invent one; a granter who opens
production sees that they did, on the form, in the approval, and in the member list afterwards.

**D8 · The approver sees the person and the power first, in every state.** The details card puts
a *Grantee* row under *Role* — display name, email, principal type — and a *Direct DDL/DML
execution* row right after it. That row is never empty: with a list it renders D5's approver lead
and chips; with `in []` it renders the *none* sentence, replacing today's hidden state and the
comment that calls it degenerate; with no clause it renders *all*; with an unrecognized condition
it renders D4a's raw expression. *Databases* and *Expires* follow, and the permission list moves to
the bottom. The grantee is read from `roleGrant.user`, never from the issue creator. When the two
differ — possible only through the API — the row gains one line, *"Requested by {creator}"*; when
they match, nothing. That line is visibility, not a control: the self-approval guard keys on the
creator (`backend/component/review/workflow.go:352`), so a request opened by a service account for
a human leaves that human free to approve their own grant until the guard also excludes the
grantee ([BYT-10275](https://linear.app/bytebase/issue/BYT-10275)).

**D9 · The issue title carries the scope.** The sheet generates *Request "SQL Editor User" role*
today, or *[Request role] {reason}* when the project enforces issue titles
(`RequestRoleSheet.tsx:426-433`); with the switch on, both branches get the suffix *· direct DDL/DML in
Staging, Prod*, placed after the database names the generated title already carries, so the issue
list shows the request's shape without opening it. The title is
client-supplied text stored verbatim (`issue_service.go:711`) and an environment rename does not
update it, so the details card is authoritative and the title is a convenience.

**D10 · Locales move together.** New keys land in all five locale files: `check-i18n.mjs` fails
the frontend gate for a key missing from any locale (Check 3, `frontend/scripts/check-i18n.mjs:292`)
and for a locale key no code references (Check 2, `:271`), which is what forces
`project.members.ddl-warning` and the three `ddl-current-*` keys out rather than leaving them
behind.

## States

Mockups A–J are in the PR description. Product typography, spacing and semantic colors are taken
from the live forms; the copy is the copy the decisions above specify. K has no mockup: it is one
sentence in the row J already shows.

| | State | What it settles |
|---|---|---|
| A | Grant form, today | The field between Databases and Expiration, the skipped callout, the role description that omits DDL/DML |
| B | Grant form, switch off | D1, D6 — the default, its caption, the description under the role |
| C | Grant form, on, nothing picked | D2, D3 — the picker, the adjacent error, submit blocked |
| D | Grant form, on, Staging + Prod | D4, D5 — chips in the callout, the protected line |
| E | Picker open | D7 — the protected row, flagged and selectable |
| F | Grant form, zh-CN | D1–D5 in the customer's locale; the chip layout survives the change of list punctuation |
| G | Request sheet, today | The same field in the requester's hands |
| H | Request sheet, proposed | D1–D5 with the requester's lead |
| I | Approver's page, today | Raw permission ids, a generic callout, comma-joined titles, no grantee, nothing at all for an empty list |
| J | Approver's page, proposed | D8, D9 — Grantee row, the callout naming the person, the title suffix |
| K | Approver's page, switch off | D8 — the *none* sentence in the row, never hidden |

## Scope

Frontend only. No proto, store, or migration change; the stored condition is byte-for-byte what
the current forms produce for the same choices.

| File | Change |
|---|---|
| `components/role-grant/DDLWarningCallout.tsx` → `DirectExecutionCallout.tsx` | `environments`, `lead`, and unrecognized-condition props; chips via `EnvironmentLabel`; the protected line; the `none` and `all` states (D4, D4a, D5) |
| `components/role-grant/DirectExecutionField.tsx` (new) | Controlled switch + caption + picker + error; renders `enabled` and `environments`, owns no state (D1–D3) |
| `components/EnvironmentSelect.tsx` | `renderSuffix` on the multi-select branch (D7) |
| `utils/issue/cel.ts` | `convertFromExpr` reports whether the condition is structurally one the forms emit (D4a) |
| `routes/workspace/MembersPage.tsx` | `RoleBindingFormState` gains `directExecution`; `ProjectRoleBindingForm` uses the field and shows the role description; role change resets both values; the workspace sheet renders the `all` callout; `canSubmit` gains D3 (D1–D3, D5, D6) |
| `modules/access-control/RequestRoleSheet.tsx` | Same state, field, description, and reset; `canSubmit` gains D3; the title suffix in both branches (D9) |
| `routes/workspace/MemberBindingEnvironmentBanner.tsx` | Renders the new callout's `some` / `none` / `all` / unrecognized states (D5) |
| `routes/project/issue-detail/components/IssueDetailRoleGrantDetails.tsx` | Grantee row, *Requested by* line, the always-present execution row with all four states, row order; the "degenerate" comment goes (D8) |
| `locales/*.json` (five files) | New keys; role descriptions; the removed `ddl-*` keys (D6, D10) |

Tests. `DDLWarningCallout.test.tsx`, `MemberBindingEnvironmentBanner.test.tsx`,
`RequestRoleSheet.test.tsx` and `IssueDetailRoleGrantDetails.test.tsx` exist and lock the current
copy keys; they are rewritten rather than extended.

- `DirectExecutionCallout.test.tsx`: renders one chip per environment, in the order given; the
  protected line appears only when a picked environment is protected *and* the feature is on;
  each lead variant renders its own text; the approver lead interpolates the grantee's display
  name; an unrecognized condition renders the raw expression and no chip.
- `DirectExecutionField.test.tsx`: off renders the caption and no picker; on reveals the picker;
  on with nothing picked shows the error; the component is controlled — it renders what it is
  given and reports changes through callbacks, nothing more.
- `cel.test.ts`: the forms' own output (`&&` of database, environment, and expiry clauses) is
  recognized; an expression containing `||`, a negation, or a raw string is not; `in []` decodes to
  an empty list and is recognized (regression: byte-for-byte today's empty-picker output).
- `RequestRoleSheet.test.tsx`: submit is disabled on + empty and enabled off; the condition built
  for off is `resource.environment_id in []`; for on it lists the picked ids; a role change while on
  turns the switch off and clears the error; the generated title carries the suffix only when on,
  in both the enforced-title and generated-title branches.
- `MembersPage.test.tsx`: the same submit and reset cases for `ProjectRoleBindingForm`; the
  selected role's description renders under the select; the workspace sheet shows the `all`
  callout when a selected role carries the permission and nothing when none does.
- `IssueDetailRoleGrantDetails.test.tsx`: the Grantee row shows `roleGrant.user`, not the creator;
  *Requested by* renders only when they differ; the callout names the grantee; `in []` renders the
  *none* sentence rather than nothing; no clause renders `all`; `… || true` renders the raw
  expression and no chip.
- `frontend/tests/e2e/project-members/project-members.spec.ts` (new, following the harness's
  `<feature>/<feature>.spec.ts` rule): grant Project Developer plus SQL Editor User with the switch
  off, open SQL Editor on a staging database, run a `CREATE TABLE`, and see the change-plan hint;
  re-grant with the switch on and Staging picked, and the same statement runs; the member list
  shows the Staging chip under the binding. No browser test covers this form today.

## Not in this PR

The implementation, which follows separately. Also deliberately out:

- **Approval rules keyed on the direct-execution scope.** The evaluator never reads the
  environment clause, so "direct DDL/DML in Prod requires a DBA" cannot be routed; only the coarse
  `request.role` rule can. Exposing the clause to role-grant rules is a backend change, tracked as
  [BYT-10274](https://linear.app/bytebase/issue/BYT-10274).
- **Denying approval when the actor is the grantee.** One guard beside the creator check in
  `workflow.go:352`, effective when self-approval is off. Backend, tracked as
  [BYT-10275](https://linear.app/bytebase/issue/BYT-10275); until it lands, D8's *Requested by* line is
  visibility only.
- **A workspace-level ceiling.** An environment setting that closes an environment to direct
  execution for everyone, which the picker would show as disabled with a reason, would turn "do
  not open production" from discipline into policy. It reintroduces the control 3.15.0 removed and
  is a backend design of its own. Recorded here as the option not taken; D7 flags protected
  environments and stops there.
- **Separate DDL and DML scopes.** A role carrying both gets one environment list
  ([BYT-8891](https://linear.app/bytebase/issue/BYT-8891)); two roles is the workaround.
- **Scoping workspace-level grants**, which carry no environment clause and so allow direct
  DDL/DML everywhere ([BYT-9356](https://linear.app/bytebase/issue/BYT-9356)). D5 tells the
  granter and the member list what such a binding does; it does not change what it does.
- **Engines outside the SQL Editor ACL gate**, where DDL/DML is checked as `bb.sql.select` and the
  clause is inert. The form does not know the engine.
- **Editing a binding's scope in place.** The edit drawer shows the binding's environments but a
  change still means revoking and re-granting.
- **The documentation page** (*Security › Database permission › Overview*), which describes the
  field in its current words.

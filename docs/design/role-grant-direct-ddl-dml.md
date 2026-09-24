# Role grant — the environment field is a permission, not a scope

Status: proposal · 2026-09-24

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
forms, the member list, and the approver's issue page.

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
through, and `frontend/src/utils/issue/cel.ts:246` writes the clause whenever the array is defined —
which the SQL service evaluates only for `bb.sql.ddl` and `bb.sql.dml` and strips for every other
permission (`backend/api/v1/sql_service.go:2275-2298`). So the empty field changes nothing about
querying, viewing, or requesting changes; it only keeps DDL/DML on the change workflow, which is
what the SQL Editor then tells the user: *"To execute DDL/DML statements in Prod, please submit a
Data Change Plan for approval."* (`frontend/src/modules/sql-editor/components/ExecuteHint.tsx:101`).
Nothing on the form says that empty is safe, so the reader who is unsure ticks everything.

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
existing bindings to carry the environments the policy had allowed. Since then, whoever can grant a
role decides, per grant, where direct DDL/DML is allowed — and the product tells them so only in the
paragraph they skip.

**The same field appears where someone asks for the role, and the approver cannot see what they
are approving.** The request sheet is the same component, so a requester who ticks every
environment is asking for direct production DDL without knowing it. The approver's page
(`frontend/src/routes/project/issue-detail/components/IssueDetailRoleGrantDetails.tsx:120-134`)
shows the role, a box of raw permission ids, the same generic callout, and the environment titles
joined with commas. It never shows the grantee: `roleGrant.user` is read nowhere in the frontend.
The request sheet always sets it to the current user (`RequestRoleSheet.tsx:409`), but the API
accepts any existing user and never compares them to the caller
(`backend/api/v1/issue_service.go:692-705`), and approval writes that principal into the binding
(`backend/utils/utils.go:111-122`). The grantee is one principal — a user, a service account, or a
workload identity — never a group.

## Principle

> **The field is a permission you switch on, never a scope you fill in.**
> Off is the default and the safe state, and the form says what off means. On reveals the
> environment picker, and every surface that shows the result — the form, the request, the
> approval, the member list — says in the reader's words what will happen, naming the environments
> it happens in.

What follows from it:

- A switch, not a picker, is the first thing the reader touches. Turning it on is the commitment;
  nobody ticks a switch "to be complete".
- The warning is a consequence of state, rendered after a choice, not a preface rendered before
  one. It names what was chosen.
- Every surface uses the same vocabulary and the same component, so the granter, the requester,
  the approver, and whoever reads the member list later see the same sentence.
- Nothing about the stored binding changes. `resource.environment_id in []` stays the storage form
  of *off*; a list stays the storage form of *on*.

## Decisions

**D1 · The field is a Switch, off by default.** Title *Direct DDL/DML execution*; switch label
*Allow running DDL/DML in SQL Editor without approval*, with `{{kind}}` substituted as today (DDL,
DML, or DDL/DML). This is the UX contract's case for a `Switch`: one binary setting whose label
names the enabled behavior and whose off state has a meaningful result
(`docs/agents/frontend-ux.md:204-208`). Off renders one caption under the switch: *"Off: every
DDL/DML from SQL Editor goes through a change plan and its approval flow, in every environment."*
Off submits exactly what an empty picker submits today, so no binding, migration, or backend path
changes.

**D2 · On reveals the picker, labeled for what it is.** Sub-label *In these environments*, the
existing `EnvironmentSelect`, placeholder *Select environments*. The word *Environments* no longer
titles the field; a title has to name the effect, and this one is the dimension.

**D3 · On with nothing picked is an error, not a silent off.** The field shows *"Pick at least one
environment, or turn this off."* beside the picker and the form cannot submit. On + empty would
write the same binding as off, and the reader's intent is ambiguous; asking is cheaper than
guessing. Validation sits with the field, per `frontend-ux.md:192`.

**D4 · The callout follows the state and names the environments.** It renders only when the
switch is on and at least one environment is picked. Its content is a static lead, one
`EnvironmentLabel` chip per picked environment, and — when any picked environment carries the
protected tag — a second line naming those environments as protected. Chips, not prose: joining
titles into a sentence is fragile across locales (English *and*, Chinese 、), and the chip is what
the member list already renders. The protected line is gated by the environment-tier feature the
way `EnvironmentLabel` already gates its shield (`frontend/src/components/EnvironmentLabel.tsx:52`);
without the feature, neither appears. Plural forms use the locale files' existing `_one` / `_other`
keys.

**D5 · One component, four leads.** `DDLWarningCallout` becomes `DirectExecutionCallout`, taking
`kind`, `environments`, and a `lead` variant:

| Surface | Lead |
|---|---|
| Grant form (members page) | Members with this role run DDL/DML straight from SQL Editor, with no change plan and no approval, in: |
| Request sheet | Once approved, you run DDL/DML straight from SQL Editor, with no change plan, in: |
| Approver's issue page | Approving lets *{grantee}* run DDL/DML straight from SQL Editor, with no change plan and no approval, in: |
| Member list, edit drawer | Runs DDL/DML in SQL Editor without approval in: |

The member list keeps its three binding states with the same vocabulary: *some* uses the lead
above with chips; *none* stays an info alert (*"Requires a change plan for DDL/DML in every
environment"*), since it is the benign state; *all* — a binding with no environment clause, which
the workspace-level members page can still produce — stays amber (*"…in every environment; this
binding has no environment scope"*).

**D6 · The role says what it can do.** SQL Editor User's description becomes *"Query data in SQL
Editor. Can also run DDL/DML there without approval, but only in environments you open below."*,
and Project Owner's gains the same clause after *"All permissions within the project"*. Both forms
render the selected role's description under the select, which the members form does not do today
(it shows a box of permission ids) and the request sheet does not do at all. Custom roles keep
their author's description; the presence of the switch is what tells the granter the role carries
the permission.

**D7 · Protected environments are flagged, not blocked.** In the open picker a protected
environment's row carries the shield and a *Protected environment* tag; in the callout it gets the
second line. It stays selectable. There is no policy that closes an environment to direct
execution, and this doc does not invent one; a granter who opens production sees that they did, on
the form, in the approval, and in the member list afterwards.

**D8 · The approver sees the person and the power first.** The details card puts a *Grantee* row
under *Role* — display name, email, principal type — and a *Direct DDL/DML execution* row with the
callout of D5 right after it; *Databases* and *Expires* follow, and the permission list moves to
the bottom. The grantee is read from `roleGrant.user`, never from the issue creator. When the two
differ — possible only through the API — the row gains one line, *"Requested by {creator}"*; when
they match, nothing. That line is the only place the mismatch could ever be seen, and it matters:
the self-approval guard keys on the creator (`backend/component/review/workflow.go`), so a request
opened by a service account for a human leaves that human free to approve their own grant.

**D9 · The issue title carries the scope.** The sheet already generates *Request "SQL Editor User"
role*; with the switch on it appends *· direct DDL/DML in Staging, Prod*, so the issue list shows
what the request is before anyone opens it. The title is stored text, so a later environment
rename does not update it; the details card is authoritative.

**D10 · Locales move together.** New keys land in all five locale files; `check-i18n.mjs` fails
the frontend gate for a key used in code and missing from any of them
(`frontend/scripts/check-i18n.mjs:115,255-263`). `project.members.ddl-warning` and the three
`ddl-current-*` keys are removed, not left behind.

## States

Mockups A–J are in the PR description. Product typography, spacing and semantic colors are taken
from the live forms; the copy is the copy the decisions above specify.

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
| I | Approver's page, today | Raw permission ids, a generic callout, comma-joined titles, no grantee |
| J | Approver's page, proposed | D8, D9 — Grantee row, the callout naming the person, the title suffix |

## Scope

Frontend only. No proto, store, or migration change; the stored condition is byte-for-byte what
the current forms produce for the same choices.

| File | Change |
|---|---|
| `components/role-grant/DDLWarningCallout.tsx` → `DirectExecutionCallout.tsx` | `environments` and `lead` props; chips via `EnvironmentLabel`; the protected line; the `none` and `all` states (D4, D5) |
| `components/role-grant/DirectExecutionField.tsx` (new) | Switch, caption, picker, error; owns the on/off ↔ `string[]` mapping so both forms submit what they do today (D1–D3) |
| `routes/workspace/MembersPage.tsx` | `ProjectRoleBindingForm` uses the field; role description under the select; `canSubmit` gains D3 (D1–D3, D6) |
| `modules/access-control/RequestRoleSheet.tsx` | Same field and description; `canSubmit` gains D3; the title suffix (D9) |
| `routes/workspace/MemberBindingEnvironmentBanner.tsx` | Renders the new callout's `some` / `none` / `all` states (D5) |
| `routes/project/issue-detail/components/IssueDetailRoleGrantDetails.tsx` | Grantee row, *Requested by* line, callout with the grantee's name, row order (D8) |
| `locales/*.json` (five files) | New keys; role descriptions; the removed `ddl-*` keys (D6, D10) |

Tests. `DDLWarningCallout.test.tsx`, `MemberBindingEnvironmentBanner.test.tsx`,
`RequestRoleSheet.test.tsx` and `IssueDetailRoleGrantDetails.test.tsx` exist and lock the current
copy keys; they are rewritten rather than extended.

- `DirectExecutionCallout.test.tsx`: renders one chip per environment, in the order given; the
  protected line appears only when a picked environment is protected *and* the feature is on;
  each lead variant renders its own text; the approver lead interpolates the grantee's display
  name.
- `DirectExecutionField.test.tsx`: off reports `[]`; turning on reveals the picker and reports the
  picked list; on with nothing picked shows the error and reports invalid; turning off clears the
  error and reports `[]` again while remembering nothing.
- `RequestRoleSheet.test.tsx`: submit is disabled on + empty and enabled off; the condition built
  for off is `resource.environment_id in []` (regression: byte-for-byte today's empty-picker
  output); for on it lists the picked ids; the generated title carries the suffix only when on.
- `MembersPage.test.tsx`: the same three submit cases for `ProjectRoleBindingForm`; the selected
  role's description renders under the select.
- `IssueDetailRoleGrantDetails.test.tsx`: the Grantee row shows `roleGrant.user`, not the creator;
  *Requested by* renders only when they differ; the callout names the grantee; the `all` state (no
  environment clause) still renders amber.
- `frontend/tests/e2e/project-members/project-members.spec.ts` (new): grant SQL Editor User with the
  switch off, open SQL Editor on a staging database, run a `CREATE TABLE`, and see the change-plan
  hint; re-grant with the switch on and Staging picked, and the same statement runs; the member
  list shows the Staging chip under the binding. No browser test covers this form today.

## Not in this PR

The implementation, which follows separately. Also deliberately out:

- **A workspace-level ceiling.** An environment setting that closes an environment to direct
  execution for everyone, which the picker would show as disabled with a reason, would turn "do
  not open production" from discipline into policy. It reintroduces the control 3.15.0 removed and
  is a backend design of its own. Recorded here as the option not taken; D7 flags protected
  environments and stops there.
- **Enforcing grantee = caller on `CreateIssue`.** The API's on-behalf shape has a customer asking
  for it; D8 makes it visible rather than closing it.
- **Separate DDL and DML scopes.** A role carrying both gets one environment list
  ([BYT-8891](https://linear.app/bytebase/issue/BYT-8891)); two roles is the workaround.
- **Workspace-level grants of these roles**, which carry no environment clause and so allow direct
  DDL/DML everywhere ([BYT-9356](https://linear.app/bytebase/issue/BYT-9356)). D5's `all` state
  keeps them visible in the member list.
- **Editing a binding's scope in place.** The edit drawer shows the binding's environments but a
  change still means revoking and re-granting.
- **The documentation page** (*Security › Database permission › Overview*), which describes the
  field in its current words.

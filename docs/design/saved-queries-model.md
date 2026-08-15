# Saved queries: sharing, permissions, and API

2026-08-09. Redesign of the SQL Editor's saved-query model — sharing,
permissions, API, storage — and a rename from "worksheet" (see Naming).
Replaces the three-state visibility flag with the industry-consensus model:
per-object graduated grants to users and groups, private by default, with
per-project discovery.

Amended 2026-08-14: the single admin backstop `bb.savedQueries.manage` is
split into per-verb permissions — `get`, `update`, `delete`, and the policy
pair `getIamPolicy`/`setIamPolicy` — so saved-query access is standard IAM
evaluation: the saved query is a sub-project policy attachment point, the
creator its owner holding full per-object permissions, the sharing levels
fixed permission bundles, and a project- or workspace-level role grant
reaching every saved query in scope. No predefined role carries
`setIamPolicy`, so sharing stays the creator's by default. See The
permissions, and Alternatives: A single `manage` backstop.

## Background

Saved queries (renamed from worksheets) are personal SQL scratchpads in the
SQL Editor: drafted under a project, optionally shared with teammates,
autosaved, high-churn. They are not change-management artifacts
(sheets/releases carry deployed SQL). Every SQL Editor user touches them
daily.

## Problem

1. **The authorization model has holes and no statable rule** (2026-08 v1
   API audit, [v1-api-audit-2026-08.md](v1-api-audit-2026-08.md), T7/T8):
   `CreateWorksheet` had **no permission check** — any authenticated
   principal could plant a worksheet into any project; the creator
   short-circuit **outlives project membership implicitly** — flag-derived
   access with no grant to see, audit, or revoke; the `PROJECT_READ` write
   path requires a permission no predefined role grants;
   `bb.worksheets.list` is granted by no role and called by no UI;
   `bb.worksheets.manage` is an unsurfaced god-mode that reads PRIVATE
   content. Holes recur because there is no model to check a change against.
2. **Sharing is all-or-nothing at project scope.** One mode flag per
   worksheet: no per-person or per-group grants, no read/write split per
   audience. "Share" means granting the whole project write — the easy path
   is the unsafe one. Users overshare or don't share.
3. **The current shape is the industry's deprecated generation.** A
   project-scoped permission family plus a per-object visibility enum is
   classic BigQuery — the design its own vendor is retiring. All current
   peers use per-object grants; even the name "worksheet" is the retiring
   term.

## Goals

Cited throughout as G1–G8.

1. **One access rule**, statable in eight lines, enforced server-side;
   every RPC gate derives from it.
2. **Private by default**; per-object **VIEWER / EDITOR** grants to `user:`
   and `group:` principals. No grantable owner tier.
3. **BigQuery-layered gates**: a per-object grant carries content access;
   *discovery* (Search, always per-project, returning what the caller can
   read) is gated
   by a dedicated `bb.savedQueries.search` permission carried by the human
   project roles — automation roles stay off the surface;
   *running* is gated by the SQL Editor's own database permissions. Leaving
   a project ends discovery and run; explicit grants persist until
   revoked.
4. The **creator is the fixed owner** (no transfer), holding every
   permission on their own saved query; roles granting per-verb
   `bb.savedQueries.*` permissions reach any saved query in scope, private
   included.
5. **Creation is gated** by `bb.savedQueries.create` on the parent project.
6. A cross-project **auditor** (`bb.savedQueries.list`) enumerates/filters
   saved queries (e.g. by creator) independent of grants — offboarding
   review, incident audit.
7. **Listing scales**: My / Shared / Starred are separate indexed queries
   on the platform's standard pagination; group sharing never expands group
   memberships.
8. **Rename** worksheet → saved query, bundled with the already-breaking
   API change.

The frontend mirrors these rules for control visibility only; enforcement is
server-side. Where the mirror cannot reproduce an evaluation — a binding
condition beyond the canonical expiry form — it hides the control rather than
showing one the server denies.

### Non-goals

- Anonymous public links — never; an *authenticated* link-grant is deferred,
  not rejected (see Link sharing).
- Cross-project "shared with me" — no peer has one.
- Ownership transfer / co-ownership (see Alternatives).
- Shared folders — Phase 2 (see Folders and stars).
- A resource-level IAM store — the policy is purpose-built; an IAM-native
  store can absorb it later (see Alternatives).

## Naming: "worksheet" vs "saved query"

| Term | Products |
|---|---|
| **Saved query** | BigQuery (our reference model) |
| **Query** | Databricks, PopSQL, Redash |
| **Question** | Metabase (BI framing) |
| **Worksheet** | Snowflake — and Bytebase (current) |

The industry standardized on **query / saved query**; "worksheet" is
Snowflake's term, and Snowflake is retiring it (Worksheets → Workspaces,
June 2026). **Decision: rename to `SavedQuery`** — resource type
`bytebase.com/SavedQuery`, `SavedQueryService`,
`projects/{p}/savedQueries/{id}`, `bb.savedQueries.*`, table `saved_query`,
UI copy "Saved queries". The rename is breaking (proto, permissions, URLs,
i18n, data) and cheapest paid now, bundled with the redesign's other
breaking changes. In this document, "worksheet" / `bb.worksheets.*` / the
`worksheet` table always denote the **current** system this design
replaces — plus Snowflake's product term in the survey.

## Industry survey

| Product | Private by default | Sharing mechanism | Grant levels |
|---|---|---|---|
| BigQuery modern (Dataform-backed) | Yes | **Per-query IAM bindings**, to users/groups/`allAuthenticatedUsers` | Code Viewer / Editor / Owner |
| Snowflake (Snowsight) | Yes | Per-recipient share on the worksheet | View / View+Run / Edit |
| Databricks SQL | Yes (creator + admins) | Per-object ACL | CAN VIEW / RUN / EDIT / MANAGE |
| PopSQL | Yes (personal folder); shared folders team-visible | Per-query / per-folder | Viewer / Editor |
| BigQuery classic (deprecated) | Yes (Personal) | Visibility tier + project-scoped permission family | Personal / Project / Public-link |
| Bytebase (current) | Yes (PRIVATE) | Per-worksheet project-wide mode flag | PROJECT_READ / PROJECT_WRITE |

Classic BigQuery — the project-scoped permission family plus visibility
enum, the model closest to Bytebase's current shape — is **deprecated**
(proximity is not a direction); modern Dataform-backed saved queries with
**resource-level IAM** (`dataform.codeViewer/Editor/Owner` on the query) are
the only path forward. The modern model has no project-members principal, so
Google's own migration turns Project-visibility queries owner-private for
manual re-grant. This design makes the same call at cutover — existing
shared worksheets migrate owner-private and owners re-share (see
Compatibility) — rather than inventing a project-derived principal.

### Consensus

All four current-generation products (modern BigQuery, Snowsight,
Databricks, PopSQL) agree: **private by default** with the creator as
implicit owner; **sharing = graduated per-object grants**; **shared editing
standard**, fork as complement; grants ride the platform's own grant system
where one exists. Team-wide sharing always means **groups** — explicit,
separately-maintained user lists (Snowflake roles, Google groups, Databricks
groups) — **never a principal derived from a container's access policy**.
This design follows: sharing targets `user:`/`group:` only, and accepts
losing the live project-wide flag deliberately (see Alternatives:
projectMembers).

## Current Bytebase behavior

The evidence for Problem 1, in pre-rename terms:

- Three visibility states (`PRIVATE`/`PROJECT_READ`/`PROJECT_WRITE`): one
  project-wide mode flag per worksheet, no per-user grants.
- Permissions: `bb.worksheets.create` (the interim audit fix),
  `bb.worksheets.get` (read shared), `bb.worksheets.list` (granted by no
  role, called by no UI), `bb.worksheets.manage` (workspace god-mode incl.
  reading PRIVATE, surfaced by no UI).
- Creator short-circuits before membership: a user removed from a project
  keeps editing their shared worksheets there.
- `PROJECT_READ` write expects a project-scoped `manage` no predefined role
  grants, contradicting the frontend's own comment.

## Design

### Key decisions

Cited throughout as "decision N"; the sections below carry the mechanics,
Alternatives the rejected options.

1. **Rename worksheet → saved query** (Naming). The industry-standard term;
   Snowflake is retiring "worksheet"; and the API is already breaking, so
   the one-time rename cost (proto, URLs, permissions, i18n, table) is at
   its cheapest.
2. **Per-object VIEWER/EDITOR grants to `user:`/`group:`.** The consensus
   model of all four current-generation peers; the sharer scopes both
   audience and level — this fixes Problem 2. The real cost: "everyone in
   the project" is no longer a live flag — it is a group, or a
   share-with-project snapshot of the project's current principals.
3. **BigQuery-layered gates: the grant carries content, a search
   permission gates discovery, running needs database permissions** (G3).
   Exactly new BigQuery's enforcement — `codeViewer` reads the asset while
   `dataform.repositories.list` gates listing and `bigquery.jobUser` gates
   running; here, a VIEWER/EDITOR binding reads/edits,
   `bb.savedQueries.search` gates `SearchSavedQueries` (per-project), and
   running always passes the SQL Editor's database grants. The real cost:
   removing a user from a project ends discovery and run but not
   explicitly granted content access — the grants themselves are the
   revocation surface (see Model: Offboarding). The stricter
   membership-AND-grant conjunction was adopted in an earlier revision and
   dropped as stricter than any peer (see Alternatives).
4. **Creator = fixed owner, no transfer** (G4). Scratchpad-grade
   simplicity: no extra tier, no orphan-guard machinery; the admin backstop
   and fork-a-copy cover the departed-creator case.
5. **Per-verb role permissions are the admin backstop** (amended
   2026-08-14; originally a single `bb.savedQueries.manage`). A project- or
   workspace-level grant of `get`, `getIamPolicy`, `update`, or `delete`
   reaches any saved query in scope, private included — peer parity
   (`dataform.admin`, Databricks workspace admins), and what makes incident
   response and orphan cleanup possible. **Sharing** has a verb too,
   `setIamPolicy`, but no predefined role carries it, so nobody re-grants a
   saved query out from under its creator unless a custom role says so. One
   thing sits outside the verbs: the bulk
   **`MoveMySavedQueries`**, which re-files only the caller's own folder
   tree, though a single `UpdateSavedQuery` still moves one saved query for
   anyone holding `update`. **Starring** follows `read` — a personal marker on anything
   you can reach, granting nothing, yours alone. The real tradeoff:
   role-`get` holders read private drafts, and reads are not audited — the
   audit log records changes, not lookups (see API and authorization: Audit
   events). The control on admin access is therefore who holds the verbs
   (never a default member role), not an after-the-fact read trail; an
   admin *write* to someone else's query is audited like any other write.
6. **A purpose-built level policy, not the role-based `IamPolicy` proto.**
   Per-object access is a *capability* (view/edit), as every peer models
   it; reusing `IamPolicy` would force per-object pseudo-roles and a
   resource-level role store. The cost — a second policy shape in the API —
   is softened by both shapes reducing to *(members, capability)*, so the
   bindings can be re-expressed as `IamPolicy` later without a data change.
   The levels are fixed permission bundles — VIEWER = `{get, getIamPolicy}`,
   EDITOR = VIEWER + `{update}` — so that later re-expression is a rename,
   not a re-evaluation.
7. **A `bb.savedQueries.list` auditor permission** (G6). Governance needs
   grant-independent, cross-project enumeration; `list` is the bulk
   counterpart to per-object `get`. It can read all content in scope — that is its
   purpose — so it is granted deliberately, never in default member roles.
8. **Hard cutover, no legacy shim.** The feature is UI-owned and the
   frontend ships in-repo the same release; after the rename, a shim is an
   entire duplicate service to maintain and secure. The cost: external
   scripts on `/v1/**/worksheets` break loudly at upgrade, with changelog
   guidance.
9. **Deep links only — no anonymous links; authenticated link-grant
   deferred** (Link sharing). The governance-adjacent peers ship exactly
   this for queries: the URL carries location, IAM/ACL carries access; an
   anonymous URL is unattributable, ungoverned egress. Given up: the
   PopSQL/Metabase-style "anyone with the link" convenience; the
   Snowflake-style authenticated link-grant waits for demand and slots in
   later as one more binding principal.

### Model and access rules

Three things grant access to a saved query, and nothing else does:

- **Creator.** Whoever created it owns it, permanently. Ownership never
  transfers and cannot be granted away (G4).
- **Binding.** A VIEWER or EDITOR grant to a `user:` or `group:` principal,
  stored on the saved query itself. A saved query with no bindings is private.
- **Role permission.** A project- or workspace-level grant of a per-verb
  `bb.savedQueries.*` permission reaches any saved query in scope, private
  ones included — matching `dataform.admin` and Databricks workspace
  admins, and what makes incident response and orphan cleanup possible
  (decision 5). The saved query is thus a sub-project policy attachment
  point: owner, then object bindings, then the project's own IAM.

```
Levels:      VIEWER = {get, getIamPolicy}   EDITOR = VIEWER + {update}
Principals:  user:{email} | group:{email}
```

The creator may be any principal type — a user, a service account, a workload
identity — since whoever holds `create` owns what they create. Only *sharing*
is narrower: binding members are `user:` and `group:` only, so a service
account can own and run its own automation queries but is never a grantee.

Member format follows `bytebase.v1.IamPolicy` exactly, on both sides of the
boundary it draws. The API takes the binding form (`user:`, `group:`), minus
the types above; the store holds the resource-name form the IAM policy
payloads hold, `users/{email}` and `groups/{email}`, converted once on write
as `convertToStoreIamPolicyMember` does for a project policy.

The read path then converts nothing: the caller's members resolve in stored
form — their own typed name plus `iam.GetUserGroups` — so the access clause
compares them to `bindings` directly. Principal typing stays load-bearing: a
service account is only ever named `serviceAccounts/{email}`, so a binding
naming one under a user prefix matches nobody. That is what lets the write
path validate members by prefix alone, exactly as project IAM does.

#### The rules

Every RPC gate derives from these (G1). `u` is the caller, `P` a project, `s`
a saved query; `proj(u, P, v)` means any of `u`'s role bindings —
project-level, direct or via a group, or workspace-level — grants
`bb.savedQueries.<v>` on `P`.

```
bind(s, u, v)  = a binding in s.bindings names principals(u) and its level
                 carries v: VIEWER carries get, getIamPolicy; EDITOR adds update

create(u, P)   = proj(u, P, create)
discover(u, P) = proj(u, P, search)              -- gates Search, not content
read(s, u)     = u == s.creator OR bind(s, u, get) OR proj(u, s.project, get)
write(s, u)    = u == s.creator OR bind(s, u, update) OR proj(u, s.project, update)
star(s, u)     = read(s, u)                      -- personal, grants nothing
move(s, u)     = u == s.creator                  -- MoveMySavedQueries
readPolicy(s, u) = u == s.creator OR bind(s, u, getIamPolicy) OR proj(u, s.project, getIamPolicy)
share(s, u)    = u == s.creator OR proj(u, s.project, setIamPolicy)
delete(s, u)   = u == s.creator OR proj(u, s.project, delete)
```

Every per-object RPC answers NotFound, never PermissionDenied, when its one
permission check fails — reads and writes alike — so a name stays
unprobeable. The blunt edge is deliberate: a VIEWER whose update is denied
gets the same NotFound as a stranger; one rule, no per-capability
distinction. A role granting `update` or `delete` without `get` can act on
names it already knows but discover nothing. Write verbs read what they
touch: `update` alone returns the updated saved query, content included,
and `setIamPolicy` alone can grant its holder the rest. Confidentiality
against a role means withholding the verb, not a pairing rule.

#### The permissions

| Permission | Meaning |
|---|---|
| `bb.savedQueries.create` | Create saved queries in the project |
| `bb.savedQueries.search` | Discover: Search and the folder list, results still filtered to `read(s,u)` |
| `bb.savedQueries.get` | Read any saved query in scope: Get, star eligibility |
| `bb.savedQueries.update` | Write any saved query in scope, the `folder` field included |
| `bb.savedQueries.delete` | Delete any saved query in scope |
| `bb.savedQueries.getIamPolicy` | Read any saved query's sharing policy in scope |
| `bb.savedQueries.setIamPolicy` | Replace any saved query's sharing policy in scope; no predefined role carries it |
| `bb.savedQueries.list` | Audit: enumerate in scope, full content, ignoring bindings |

One permission per verb, named for its RPC (`list` for `ListSavedQueries`,
`delete` for `DeleteSavedQuery`, …), evaluated exactly like the rest of the
permission vocabulary. The binding levels are fixed bundles of the same
verbs — VIEWER = `{get, getIamPolicy}`, EDITOR = VIEWER + `{update}` — and
the owner holds full per-object permissions. `setIamPolicy` sits in no
bundle and no predefined role, so sharing is the creator's unless a custom
role grants it deliberately.

`bb.savedQueries.get` is a role-scope verb, not the grantee read path —
bindings decide grantee reads, so the old `bb.worksheets.get` ("read
shared") is still dropped rather than renamed. `search` is dedicated rather
than riding on `bb.projects.get` (see Alternatives) so a custom role can
control the saved-query surface independently of project visibility.

Two deliberate asymmetries:

- **The policy pair has its own permissions**, `getIamPolicy` and
  `setIamPolicy`, mirroring `bb.projects.get/setIamPolicy`. The VIEWER and
  EDITOR bundles carry `getIamPolicy`, so a grantee can always learn their
  own level; `setIamPolicy` sits in no bundle and no predefined role, so
  sharing is the creator's by default and admin re-sharing is a deliberate
  custom-role grant.
- **`list` is strictly stronger than `get` in information terms** — whole
  statements, bindings ignored — yet does not satisfy `GetSavedQuery` by
  name: per-method permissions stay per-method, and roles holding `list`
  should hold `get` (every predefined one does).

| Role | create | search | get | getIamPolicy | update | delete | setIamPolicy | list |
|---|:--:|:--:|:--:|:--:|:--:|:--:|:--:|:--:|
| `workspaceAdmin`, `workspaceDBA` | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ | | ✓ |
| `projectOwner` | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ | | ✓ |
| `projectDeveloper`, `sqlEditorUser`, `sqlEditorReadUser`, `projectViewer` | ✓ | ✓ | | | | | | |
| `projectReleaser`, `gitopsServiceAgent` | | | | | | | | |

Two things this table encodes. Every verb except `create` and `search`
reaches other people's private content by design — that is what a
project-wide grant means on a private-by-default resource — so none of them
belongs in a default member role (decision 7): ordinary SQL Editor use
needs only `create + search`, with reads and writes flowing from creator
and bindings. And the automation roles carry nothing at all, which keeps CI
principals off the saved-query surface entirely — the property that makes
the share-with-project role filter meaningful.

The discovery gate is `search` alone: without it the search family is
denied outright, whatever the caller could read. With it, results carry
every saved query the caller holds `get` on — own, shared, or role-granted
— the same access rule as `GetSavedQuery`.

#### Layered gates

Access is three independent layers, exactly as new BigQuery enforces them (G3):

| Layer | Gate | BigQuery analog |
|---|---|---|
| Content | the binding on the saved query, or a role-granted verb | `dataform.codeViewer` / `codeEditor` |
| Discovery | `bb.savedQueries.search` on the project | `dataform.repositories.list` |
| Running | the SQL Editor's own database permissions | `bigquery.jobUser` |

So a grantee outside the project opens the saved query by link — BigQuery's own
"grant the role, then share the link" flow — but cannot browse the project or
run anything in it. Discovery is always per-project; there is no cross-project
"shared with me", and no peer has one.

**Conditioned role bindings.** Expiry (`request.time`) is honored. A binding
whose condition scopes *resources* — databases, environments — confers **no**
`bb.savedQueries.*` permission at all: these surfaces are project-wide, and a
data-slice grant must not silently widen to them. The generic interceptor
cannot enforce this (it evaluates `request.time` only and passes residual
resource conditions), so the rule lives in saved-query code: the CUSTOM gates,
plus an in-handler source re-check on both IAM methods, `create` and `list`.

#### Offboarding

Removing a user from a project ends discovery and running immediately
(workspace-wide roles aside — those reach every project by design). Explicit
grants, and the creator's access to their own queries, persist until revoked.
Three levers, blunt to precise:

1. **Workspace deactivation** ends everything at once.
2. **Group membership** revokes every group-held grant in one edit, since a
   group binding names the live group rather than a frozen expansion.
3. **`SetSavedQueryPolicy`** (the creator, or a custom role granting
   `setIamPolicy`) strips a direct grant on one saved query.

Users can always see what they hold: `SearchSavedQueries` with `shared == true`
returns everything shared with the caller.

What v1 does not ship is the third-party direction — a bulk "what is still
shared with *this other person*" sweep. Search cannot answer it, since it
scopes to the caller's own readable set. `ListSavedQueries` filters by
`creator`, which finds what the departing user owns (no grant-based filter
could find those anyway; an admin deletes them, or deliberately leaves them
their drafts), but there is no `member` filter and bindings are not on the
`SavedQuery` resource, so enumerating someone's residual grants means walking
`GetSavedQueryPolicy` per saved query as an admin.

That is a scope cut, not a model limitation: the grants exist, are readable,
and are revocable per query — by its creator; an admin's blunt lever is
deleting the query — only the sweep over them is missing, and a
`member ==` filter slots in later as a purely additive change. The audit's
Problem-1 complaint was never retention itself but *implicit* retention:
flag-derived access with no grant to see, audit, or revoke. Grants are now
explicit objects with a read and a write API, which answers it.

#### Share with project

There is deliberately no derived "everyone in the project" principal (see
Alternatives). The share dialog's *Share with project* shortcut instead
snapshots the project's current principals into the policy at the chosen
level, as `user:` and `group:` bindings — never an expansion of group members.

The source is the project's **effective** role bindings, not just its own
policy row: a project role granted at the workspace level applies to every
project, so those fold in too. Otherwise a workspace-level project member
would pass discovery yet be silently missing from the share.

Four things are skipped:

| Skipped | Why |
|---|---|
| Roles without `search` | A CI releaser is deliberately off this surface; sweeping it in would put it back on |
| Roles with `get` | Their holders reach the query through the role grant, so a binding is redundant *and* would outlive the role, leaving a former operator deep-link access |
| `allUsers` members | The policy accepts only `user:`/`group:`, and freezing a workspace-wide audience into per-user grants is the member explosion the snapshot exists to avoid. The dialog surfaces the skip so the sharer can grant a group instead |
| Conditioned bindings | A time-bounded grant cannot be faithfully frozen into an unconditional one; under-granting beats silently making expiring access permanent |

What remains is the pure project audience — `projectDeveloper`,
`sqlEditorUser`, `sqlEditorReadUser`, `projectViewer` — which is exactly who
"share with everyone here" means.

A snapshot keeps group dynamism (a group gaining a member propagates, since the
binding stores the group reference) but not membership dynamism: a principal
added to the project later is re-shared by the owner, as in every group-based
peer.


### API and authorization

Sharing is a policy on the resource, read and written by the
`GetSavedQueryPolicy`/`SetSavedQueryPolicy` pair — new BigQuery's shape, and
Databricks and Snowflake likewise keep permissions on a dedicated surface. The
pair is named for the resource rather than reusing `GetIamPolicy`/`SetIamPolicy`,
which already exist on `ProjectService` and `WorkspaceService` returning
`bytebase.v1.IamPolicy`: one method name with two return types across the same
API is a trap on the generated SDK and OpenAPI surfaces.

Because Bytebase has no resource-level role store below the project, the policy
carries capability **levels** grouped by member list, evaluated by the
saved-query handlers rather than the central `CheckPermission` (decision 6).

Two properties matter:

- **`SetSavedQueryPolicy` is compare-and-swap.** `GetSavedQueryPolicy` returns
  an `etag`; the write must present it, and a mismatch aborts for refetch. A
  full-replacement write may never silently overwrite a concurrent revocation
  — the offboarding race.
- **`GetSavedQueryPolicy` is gated on `readPolicy(s,u)`** — the bundles
  carry `getIamPolicy` — and returns the whole policy. That is what lets a
  grantee discover whether they may edit, which is the only way to know now
  that nothing caller-relative rides on the resource. Writing the policy
  still needs `share(s,u)`.

  An earlier revision filtered the response so a grantee saw only their own
  bindings. Dropped as unnecessary: any authenticated user can already list
  every user in the workspace, so all the filtering concealed was which of
  those colleagues hold a grant on this query, from someone who holds one
  themselves. It also had no peer — GCP's `getIamPolicy` is all-or-nothing —
  and it bought a second response shape plus an invariant about not writing a
  partial view back. Seeing who else can reach a query you can reach is
  arguably the better governance property anyway.

**Database is project-scoped.** `database` is an ordinary `write(s,u)` field
with one invariant, carried over from today's `getWorksheetDatabase`: a set
database must belong to the saved query's own project, stored canonically. A
project-A editor cannot pin a project-B database onto a project-A query —
otherwise Get/Search responses and the deep link become a cross-project
metadata surface, even though running still needs that database's own grants.
Empty is allowed, and the reference stays soft (nullable, no FK), so it may
dangle after the database is deleted or transferred and degrade to "no
database" in the UI.

**Per-RPC access.** `CreateSavedQuery` and `ListSavedQueries` are IAM — one
family permission at the interceptor — and both re-check the permission source
in-handler to reject resource-conditioned bindings (see Conditioned role
bindings). The search family is CUSTOM so the same source re-check runs
beside its read-scoped access clause; its permission is `search` alone.
Every object method is a CUSTOM per-row predicate.

| RPC | Auth | Gate | Content / audit |
|---|---|---|---|
| `CreateSavedQuery` | IAM `create` + source re-check | creator becomes owner | writes own; audited |
| `SearchSavedQueries` | CUSTOM | `discover` gate; returns rows where `read(s,u)` holds | previews; unaudited |
| `ListSavedQueries` | IAM `list` + source re-check | all rows matching `filter`; `projects/-` allowed | full content; unaudited |
| `GetSavedQuery` | CUSTOM | `read(s,u)`; NotFound when unreadable | full content; unaudited |
| `UpdateSavedQuery` | CUSTOM | `write(s,u)` | returns content; audited |
| `DeleteSavedQuery` | CUSTOM | `delete(s,u)` | — ; audited |
| `GetSavedQueryPolicy` | CUSTOM | `readPolicy(s,u)`; NotFound when unreadable | policy only; unaudited |
| `SetSavedQueryPolicy` | CUSTOM | `share(s,u)` + etag CAS | policy only; audited |
| `UpdateSavedQueryStar` | CUSTOM | `read(s,u)` — star any readable query | — ; unaudited |
| `MoveMySavedQueries` | CUSTOM | the caller's own rows under `source_folder` (descendants included); single re-files ride `UpdateSavedQuery` | count only; audited |
| `SearchSavedQueryFolders` | CUSTOM | `discover` → folder paths of readable rows, same access clause as Search | paths only; unaudited |

Duplicate and fork need no RPC: read what you can already see, then
`CreateSavedQuery`. The copy is yours, and private.

#### Search versus List

Two enumeration surfaces for two use cases, not redundant:

| | `SearchSavedQueries` | `ListSavedQueries` |
|---|---|---|
| For | The SQL Editor | Governance |
| Scope | One project | One project, or `projects/-` |
| Bindings | Respected; project-level `get` adds the rest | Ignored |
| Content | Previews | Whole statements |
| Views | `creator == me`, `shared == true`, `starred == true` | `creator ==` |

For a role-`get` holder the two overlap in reach within one project; they
stay two surfaces because Search serves the SQL Editor (previews, filter
views) while List serves governance (whole statements, `projects/-`).
One boundary is deliberately left out: a member has no
cross-project "all my own queries" view, since discovery is per-project
(peer-consistent with BigQuery and Databricks) and the only `projects/-` path is
the privileged audit List. A self-scoped cross-project view is safe to add later
without `list` — `creator == me` exposes only the caller's own rows.

The SQL Editor's Shared view is `shared == true`, not `creator != me`. The two
diverge for an admin, for whom `creator != me` also matches saved queries nobody
shared with them.

#### Audit events

Today's `WorksheetService` carries no `bytebase.v1.audit` annotations at all,
though sheet and database services do. This design closes that gap under one
invariant: **the audit log records changes, not lookups.** Every RPC that
mutates a saved query or its policy is annotated — `CreateSavedQuery`,
`UpdateSavedQuery`, `MoveMySavedQueries`, `DeleteSavedQuery`,
`SetSavedQueryPolicy` — and no read RPC is. That keeps the annotation set
derivable from the method rather than from who the caller happened to be, and
draws the same line as the rest of the v1 API.

Privileged reads are left unaudited deliberately: `GetSavedQuery` and
`SearchSavedQueries` under a role-granted `get`, and the binding-bypassing
`ListSavedQueries` — all gated by permissions no default member role
carries (decision 7), so the control is *who holds `get`/`list`*, not a
trail behind them — and a read trail on the SQL Editor's hot path would be
mostly people reading their own drafts.

Two costs, stated plainly:

- **Autosave writes are audit events.** The editor saves on a 2-second debounce,
  so an active session emits a row every couple of seconds and saved-query
  writes will dominate the log by volume. That is the price of a rule that does
  not special-case the hot path.
- **A shared query keeps no per-editor trail** in the resource. The fixed
  `creator` names the owner, not whoever last edited, so multi-editor changes
  are last-write-wins and the updater's identity lives only in the audit event.
  Add version history if that ever matters.

`UpdateSavedQueryStar` is a write but stays unaudited — a per-user marker
carrying no content and no access. `GetSavedQueryPolicy` and the folder listing
return no content and are unaudited.

#### Compatibility

**Hard cutover, no shim** (decision 8). `WorksheetService` and the `visibility`
field are removed in the release that ships `SavedQueryService`, and the in-repo
frontend ships against the new API in the same release. After the rename a shim
would not be a deprecated field but an entire second service to maintain and
secure, for a UI-owned feature. External worksheet scripts break loudly at
upgrade, documented in the changelog.

**Existing data migrates owner-private** — a deliberate, one-time breaking
change, and BigQuery's own classic→modern path. Every row keeps its `creator`,
`title`, `content`, and connected database (carried into
`SavedQueryPayload.database` as a canonical resource name, so run context
survives; cleared if that database no longer belongs to the query's project).
`visibility` is dropped and `bindings` starts empty, so `PRIVATE`,
`PROJECT_READ`, and `PROJECT_WRITE` alike become creator-only.

Owner organization is preserved: the creator's own `worksheet_organizer` row
migrates, its folder placement into `saved_query.folder` and its `starred` flag
into a `saved_query_star` row, so owners keep their tree and favorites. Only
*sharing* resets. Non-creator organizer rows are dropped, since their owners
lose access to the now-private query anyway.

Owners re-share from the new dialog, where Share with project reproduces the old
project-wide audience in one click. There is deliberately no automatic mapping:
the old flag carries no per-user grant to preserve, and snapshotting project IAM
into every shared row would widen or persist access differently — the
project-derived-principal hack rejected in Alternatives. The blast radius is
small, since sharing was a coarse and little-used flag. Called out in the
release notes; historical audit-log entries keep the old names as records.

The 2026-08-14 permission split rides the same upgrade train as the rename:
custom roles reach the split migration carrying `bb.savedQueries.manage`
either from a pre-release build or — the shipped population — from
`bb.worksheets.manage` renamed by the earlier migration in the same run. The
split expands it to `get` + `getIamPolicy` + `update` + `delete`;
`setIamPolicy` is deliberately withheld, so re-share retires for migrated
custom roles exactly as it does for the predefined admin roles, and granting
it back is an explicit custom-role edit called out in the release notes.


### Storage and query scalability

The policy lives in `saved_query.bindings jsonb` — no separate access
table. The **stored shape is pinned**: the protojson *array* of
`Binding` messages, e.g.
`[{"level": "EDITOR", "members": ["groups/eng@corp.com", "users/a@corp.com"]}]`
— the array must sit at the jsonb root for the `@>` probes below (a wrapped
`{"bindings": [...]}` would force expression indexes). This deviates from
the store-a-whole-message payload convention, deliberately; the store layer
gets its own `SavedQueryBinding` message (store protos never import v1).

**Creator storage.** `saved_query.creator` stores the raw principal email
(exactly as `worksheet.creator` does today), and the API projects it to the
typed resource name — `users/{email}` or `serviceAccounts/{email}` — by the
principal's type. So the rename carries the email over unchanged (no owner
rewrite, no lost access), and `u == s.creator` stays an email comparison
that holds for any principal type.

**Principal renames.** Principals are email-keyed by repo convention, so
a user- or group-email change must, in the same transaction, rewrite
`saved_query.creator`, `saved_query_star.principal`, and the
`user:`/`group:` members inside `saved_query.bindings` — exactly as
`UpdateUserEmail` rewrites `worksheet.creator` today. Without the jsonb
rewrite, grants, ownership, and stars silently orphan on rename. Stable
numeric principal IDs would avoid rewrites entirely but are a
platform-wide convention change outside this design's scope.

The caller's principal set expands cheaply — groups are flat (members are
users, one hop) and `GetUserGroupsSnapshot` is cached in
`memberGroupsCache`: `principals(u) = {users/u} ∪ {groups/g …}`, typically a
handful. Bindings carry `user:`/`group:` members (the IamPolicy invariant
above), so the only format boundary is group *membership*, stored
resource-name-style (`GroupMember.member = users/{email}`):
`GetUserGroupsSnapshot`'s reverse index matches `users/{u}` to find the
caller's groups, then emits each as `group:{email}` for the probe — exactly
the conversion the existing IAM group-expansion already performs. (`creator`
is a typed **principal** resource name — `users/{email}` or
`serviceAccounts/{email}` — not a binding member, so the `creator ==` filter
matches that principal form while the bindings themselves carry the
`user:`/`group:` form. Only the read path crosses between them; there is no
member-based filter in v1, see Offboarding.)
The index is `gin (bindings jsonb_path_ops)` — smaller and faster than the
default opclass, and containment is all these queries do. The list query
filters the table directly, one probe per principal:

```sql
-- The search gate already passed. A caller with project-level get drops the
-- whole access clause, as List does. Paging is the platform's LIMIT/OFFSET
-- page token.
SELECT s.* FROM saved_query s
WHERE s.project = $P
  AND ( s.creator = $me                                  -- own (incl. private)
        OR s.bindings @> '[{"members":["users/me"]}]'
        OR s.bindings @> '[{"members":["groups/g1"]}]'
        OR ... )               -- one @> per principals(u), each GIN-indexed
ORDER BY s.name, s.resource_id                           -- name is the title
LIMIT $n OFFSET $k;
-- "My" adds creator = $me and drops the bindings OR; "Shared" adds creator <> $me.
```

Honest per-tab costs (G7):

- **My** (`creator = me`): the `(project, creator, folder)` and
  `(creator, project)` btrees answer the equality; the title sort is
  uncovered, so each page sorts the matched rows. That set is one person's
  scratchpad in one project — tens to hundreds — so the sort is noise. The
  `(creator, project)` index also serves the auditor's cross-project
  `creator` filter (no binding test needed there), which is bounded and
  cold enough to need no index of its own.
- **Starred**: my rows in `saved_query_star` via its `principal` btree —
  row existence is the star. Fetch the (naturally small) set, join, drop
  rows I can no longer read from the view — the star row itself persists,
  hidden, so re-granted access restores the bookmark; rows are removed only
  with the saved query. `O(my stars)` — bounded by my own behavior.
- **Shared with me**: the only tab touching groups — `K+1` GIN probes for
  `K` groups, BitmapOr'd into my accessible set `S`, then sorted per page:
  `O(S log S)` — the honest cost of the no-extra-table choice, acceptable
  because `S` is *my* reachable set (tens to hundreds), not the project's
  total.

**Pagination follows the platform, not a bespoke cursor.** Every v1
List/Search paginates with `storepb.PageToken`'s `{limit, offset}` and a
`LIMIT/OFFSET` tail — ~22 call sites across the API — and the token
carries no cursor field to hold a keyset. Saved queries use the same
mechanism; adopting keyset pagination here would be a platform-wide change,
out of scope for this design. Two properties worth naming:

- **The sort key is the title** (`saved_query.name`), matching the SQL
  Editor's folder tree rather than a recency feed. This matters more than
  it looks: titles change only on explicit create, rename, or delete, so
  the row set under a paging caller is nearly static, and the window in
  which offset paging can skip or repeat a row is correspondingly narrow.
  Ordering by `updated_at` would instead have the 2-second autosave
  debounce reshuffling the list continuously — a worse fit for the same
  pagination mechanism.
- **The offset re-scan is real but small**: page *k* walks and discards
  *k × page* rows. Scratchpad volumes keep this far from mattering; if a
  project's list ever grows enough to feel it, the first fix is an index
  covering the sort key, not a new pagination scheme.

And the expensive move — expanding a *group's members* — never happens on
any path: policies store group references, so a 1,000-member group costs
the same as a 3-member one and membership changes rewrite nothing.

**GIN churn, stated plainly**: the bindings index sits on the same row as
the autosaved content; a non-HOT content save re-touches it, and churn can
bloat it between vacuums. Bounded in practice (large content is TOASTed,
GIN fastupdate batches, bindings change rarely, saved queries are
human-edited) — fine at scratchpad scale. **Escape hatch** if ever measured
otherwise: normalize into `saved_query_access(principal, project,
saved_query)` — still group references — indexed
`(principal, project, updated_at DESC)`, turning Shared-with-me into
per-principal *ordered* seeks (`O(K · page)`) and isolating the ACL index
from content churn. A pure storage change behind the same API.

**Lifecycle vs. project deletion.** `saved_query.project` is an FK with no
cascade, so the two directions are fenced explicitly (AGENTS.md transaction
lock-ordering):

- **Create requires an *active* project.** `CreateSavedQuery` does not go
  through `nextProjectID` (IDs are `gen_random_uuid()`, not a project-scoped
  sequence), so it must itself, in the insert transaction, lock the project
  row and reject the write unless the project is active — a create racing a
  purge fails cleanly (`FAILED_PRECONDITION`), never as an FK violation and
  never leaving a row in a project mid-deletion.
- **Star writes are child-before-parent; the parent fence covers only the
  first star.** The lock rule treats an upsert as an existing-row lock, so
  toggling or removing an **existing** `saved_query_star` row locks that
  child row directly — no parent lock. Only the **first** star for a
  (query, caller) inserts a child that cannot be locked in advance; that
  case alone takes the parent fence (lock the `saved_query` row, reject as
  `NotFound` if gone), the same missing-child carve-out `CreateSavedQuery`
  uses for `project`. Its inserted key is novel, so it never contends with
  purge's existing-child locks — no cross-order deadlock.
- **Batch folder moves lock rows in primary-key order.**
  `MoveMySavedQueries` only *updates* existing `saved_query` rows (no
  new child), but "existing" is not enough on its own: two overlapping
  batches could grab their target rows in different scan orders and
  deadlock. So it locks its selected rows in full primary-key order
  (`resource_id`) — the AGENTS.md batch rule — and a folder move touching a
  row mid-purge simply updates zero rows.
- **Delete is child-before-parent, explicitly.** `DeleteSavedQuery` removes
  the query's `saved_query_star` rows before the `saved_query` row itself —
  **not** via the FK cascade, which would lock the parent first — matching
  the star and purge order so a delete racing a star toggle or a purge
  cannot deadlock.
- **Purge is child-before-parent, and re-parents human rows.** The
  hard-delete path deletes the stars and saved queries of project service
  accounts and workload identities (stars first), then reassigns the
  remaining, human-created saved queries to the default project — locking
  existing child rows ahead of their parents, per the rule. Because a
  surviving row changes project mid-purge, **every non-purge writer scopes
  its predicate by `(resource_id, project)`** — patch, delete, and the
  first-star parent fence — so a write authorized in the purged project
  cannot land on the reassigned row; it updates zero rows and surfaces as
  NotFound (the fenced delete rolls back its star cleanup). The declared
  lifecycle policy: writers on existing rows require the row in the
  authorized project, not an active project — archived projects are already
  unreachable through every read path, so a write racing an archive is an
  ordinary serialization of concurrent requests, and restore does not
  promise to resurrect a row whose authorized delete won that race. Only
  creation requires an active project.

Two invariants cover every writer. **(1) Child before parent** for
existing-row writes/deletes; the *only* parent-first step is a new-child
*insert* (create → lock `project`; first star → lock `saved_query`), safe
because its key cannot be locked in advance — the AGENTS.md missing-child
carve-out. **(2) Any multi-row lock/update/delete acquires its rows in full
primary-key order** — `saved_query` by `resource_id`, `saved_query_star` by
`(saved_query, principal)` — so two operations over an overlapping set
(delete vs. purge deleting the same query's stars; two batch folder moves;
purge's own `saved_query` batch) can never lock in opposing orders. A plain
`DELETE … WHERE` does not guarantee that order, so these paths take their
row locks through an ordered `… ORDER BY <pk> FOR UPDATE` (or an ordered
delete) before mutating. Required before
implementation: deterministic real-PostgreSQL regression tests for **both**
acquisition orders of each contending pair — create↔purge,
first-star↔purge, delete↔star-toggle, delete↔purge over a query with
multiple stars, and two overlapping `MoveMySavedQueries` over
the same folder — asserting the terminal outcomes — project (or query) deleted, no orphaned
saved query or star, and **no** FK failure or deadlock (`40P01`) in either
direction (absence of `40P01` alone is insufficient).

### Sharing and organization UX

- Views, all within the current project, all served by `SearchSavedQueries`:
  **My** (`creator == me`, organized in my folder tree), **Shared**
  (`shared == true` — a binding grants me VIEWER or EDITOR — as a flat
  searchable list), **Starred** (`starred == true`, own or shared). Nobody
  fetches a saved query they have no access to.
- Edit-vs-view is resolved when a saved query is opened, not per row in the
  list — BigQuery's behavior, and the reason nothing caller-relative beyond
  `starred` sits on the resource. The editor calls `GetSavedQueryPolicy`
  alongside `GetSavedQuery` and reads its own binding from the response;
  creator status and the caller's role-granted verbs the UI already knows.
  In **My** everything is
  writable anyway, so only **Shared** ever resolves to view-only.
- Share dialog (creator, or a custom role granting `setIamPolicy`): add
  `user:`/`group:` at VIEWER/EDITOR;
  the picker suggests project members/groups (a non-member grantee opens
  via link but cannot browse the project or run — see Model). **Share with
  project** = the snapshot shortcut (Model). Role holders see per-verb
  affordances on any query — edit (`update`), delete (`delete`), share
  (`setIamPolicy`, no predefined role) — and sharing stays the creator's by
  default. Duplicate is always available.
  Later dialog polish (search, suggested principals, per-row revoke) needs
  no schema or API change — the bindings already carry it.

#### Folders and stars

| | Folders | Star / favorite |
|---|---|---|
| BigQuery Studio | Personal root **and** shareable folders; folder permissions **propagate** to contents | — |
| Databricks | Personal `/Users/<me>` **and** `/Shared`; folder ACL, objects **inherit** it | **Star any asset**, incl. shared-with-you; per-user |
| PopSQL | Private and shared folders; moving into a shared folder auto-shares | Per-user favorite |
| Snowflake | Folders private by default, shareable as a whole | Recent / My / Shared views |

Two conclusions: a **star is a per-user marker on anything you can reach**
(shared items included), and every peer's **shared folders grant access by
inheritance** — a second ACL surface, which is exactly why they are
deferred.

- **Stars**: a row in `saved_query_star(saved_query, principal)` — row
  existence is the star. Star any readable query; my star is invisible to
  you. Powers the Starred view.
- **Folders (Phase 1 — personal)**: in every peer a query lives in exactly
  one folder. Storage follows the semantic: the location is the
  `saved_query.folder` **path column on the object** ("a/b/c", '' =
  unfiled), set via `UpdateSavedQuery` as an ordinary `write` field, bulk
  rename/move via `MoveMySavedQueries`. Folders never
  grant access; a folder is a path on rows, so empty folders cannot exist.
  Today's per-user organizer re-foldering (everyone re-files everything) is
  dropped as the outlier. A query shared with you appears in
  **Shared**/**Starred**, never in your tree.
- **Folders (Phase 2, when demanded — shared)**: a folder becomes a shared
  resource carrying the same VIEWER/EDITOR policy shape, its contents
  inheriting `grant(s) ⊔ grant(folder)` — the peers' team library.
  Deferred: it adds a folder ACL, inheritance in the rules, and a heavier
  list query, while per-object sharing already delivers the outcome.

### Link sharing

Three different features hide under "share a link" (survey verified against
vendor docs; sources below):

| Product | Deep link (URL + normal authz) | Authenticated link-grant ("anyone *here* with the link") | Anonymous public link (secret URL, no sign-in) |
|---|---|---|---|
| BigQuery modern | Yes — grant IAM first, *then* share the link | No — "public" is an `allAuthenticatedUsers` IAM **binding**, not a secret URL | No (classic's Public-link tier dies with classic) |
| Databricks (queries) | Yes — Share dialog copies a link; ACL still decides | No | No for queries (dashboards publish/embed separately — a reporting artifact) |
| Snowflake (Snowsight) | Yes | **Yes, opt-in**: **account** users with the link get View / View+Run — weaker tiers; running still needs the role | No — sharing is to "users in your account" |
| PopSQL | Yes | — | **Yes** — public presentation links, results embedded, viewers re-run with variables |
| Metabase | Yes | — | **Yes** — admin-minted; workspace toggle; admin page lists all links |
| Redash | Yes | — | **Yes** — secret dashboard URLs on revocable API keys |

The pattern: the governance-adjacent platforms (modern BigQuery,
Databricks) offer **no access-granting link for queries**; Snowflake's
link-grant is account-scoped and authenticated; anonymous secret URLs exist
only in collaboration/BI tools whose artifact is a *result for outward
consumption*.

**What Bytebase provides (decision 9):**

1. **Deep links — yes.** Stable URL; opening runs `read(s,u)`;
   NotFound for non-viewers; "Copy link" in the editor. The URL carries
   *location*, never *access* — exactly the BigQuery/Databricks behavior.
2. **Authenticated link-grant — deferred behind a workspace policy.**
   Snowflake's pattern maps onto one more binding principal (an opaque
   link-token member) — it substitutes only for the *grant*, so a leaked
   link still reaches only authenticated workspace users, and running
   still requires database permissions. Off by default; add on demand.
3. **Anonymous public links — never.** Unattributable access (no identity
   in the audit log), no revocation short of URL rotation, ungoverned
   egress of SQL embedding schema and business logic — against the
   product's first principle. If outward *result*-sharing is ever wanted,
   it is a separately designed egress surface with its own audit events.

Sources: [Snowflake worksheets](https://docs.snowflake.com/en/user-guide/ui-snowsight-worksheets),
[Databricks saved queries](https://docs.databricks.com/aws/en/sql/user/queries/),
[BigQuery manage saved queries](https://docs.cloud.google.com/bigquery/docs/manage-saved-queries),
[Metabase public links](https://www.metabase.com/docs/latest/embedding/public-links),
[Redash sharing dashboards](https://redash.io/help/user-guide/dashboards/sharing-dashboards/),
[PopSQL sharing links](https://docs.popsql.com/docs/sharing-a-link-to-your-query-and-results).

## Alternatives

- **A project-derived `projectMembers` principal** — would re-express the
  visibility flag exactly, keeping today's project-wide semantic alive.
  Rejected as a hack:
  no peer derives a sharing audience from an access policy; it couples the
  audience to the project IAM graph (adding a project viewer silently
  widens every project-visible query) and is coarse. The
  share-with-project snapshot is the honest cost.
- **Classic-BigQuery-style permission family** (`get`/`update` on roles
  deciding read/edit of all shared queries, *instead of* per-object grants)
  — the deprecated generation of the reference product; the sharer cannot
  scope who edits. An earlier revision recommended it for proximity to
  current RBAC; proximity is not a direction. The 2026-08-14 split is not
  this: sharing stays per-object; the per-verb permissions are the admin
  backstop's vocabulary, not the sharing mechanism.
- **A single `manage` backstop** (the 2026-08-09 revision as first
  implemented) — one coarse admin permission covering read, write, delete,
  and policy administration, honestly named and easy to audit. Replaced
  2026-08-14: it was a lump the rest of the permission vocabulary does not
  have, it hid the sub-project IAM shape (owner → bindings → project
  roles), and it blocked custom-role granularity — read-only support
  (`get` alone), cleanup-only (`delete` alone). What is given
  up: the single scary knob; mitigated by the roles table (only admin roles
  hold the verbs, none holds `setIamPolicy`) and the proto comment stating
  that every verb except `create`/`search` reaches private content.
- **No policy permissions** (policy read riding on `get`, policy write
  creator-only with no permission — an interim revision of this amendment)
  — dropped for uniformity: a method whose permission line names another
  method's verb reads wrong, and a permissionless write is invisible to
  IAM. The grantee-must-see-their-own-level requirement is met by the
  bundles carrying `getIamPolicy` instead, and the creator-only default by
  granting `setIamPolicy` to no predefined role (see The permissions).
- **Keep the mode flag** (status quo) — no path to per-user/per-group
  sharing; it is Problem 2.
- **Reuse the `bytebase.v1.IamPolicy` proto per query** — role-based, would
  force per-object pseudo-roles and invite `CheckPermission` expectations.
  Both shapes reduce to *(members, capability)*, so bindings can be
  re-expressed as `IamPolicy` later without a data change.
- **Full per-object IAM** (grants in the central IAM store, resolved by
  `CheckPermission`) — the ideal end state; Bytebase has no resource-level
  binding storage below the project, and building one for scratchpads is
  disproportionate. The bindings shape lets an IAM-native store absorb it.
- **Ungated PRIVATE creation** (BigQuery leaves Personal permission-free) —
  rejected narrowly: principals include service accounts; an explicit
  `create` keeps the surface auditable at zero UX cost.
- **Gating discovery on `bb.projects.get`** instead of a dedicated
  `search` permission — an earlier revision's choice, on the argument that
  discovery is a reachability check (the handler filters to `read(s,u)`
  anyway) and a dedicated permission would be granted everywhere
  `projects.get` is. Reversed: that argument proves too much — it applies
  equally to `create`, which is kept precisely so custom roles control it
  independently — and the proxy couples the saved-query surface to a
  permission customers grant for unrelated reasons, making both
  "discover without project visibility" and "project visibility without
  the saved-query surface" inexpressible. A dedicated `search` is also
  the exact `dataform.repositories.list` analog.
- **A strict membership conjunction on every access** (`member(u, P) AND
  grant` — an earlier revision's model): project removal would sever even
  explicitly granted content access, and a grant to a non-member would sit
  inert. Dropped as stricter than any peer — BigQuery's object grant
  carries content (membership gates listing and running), and
  Databricks/Snowflake need no conjunction only because their principal
  universe *is* the container. The layered model keeps the offboarding
  outcome where it matters (discovery and run end with membership) while
  making residual access explicit and revocable instead of impossible. Its
  opposite extreme — a global cross-project "shared with me" — was also
  rejected: discovery stays per-project, as in every peer.
- **Privacy-preserving admin** (delete-without-read) — used in an earlier
  revision, dropped for peer parity and workable incident response
  (decision 5). The per-verb split makes it expressible after all: a role
  granting `delete` without `get` deletes names it learns out of band while
  discovering nothing — the single-check NotFound mask no longer re-checks
  read on writes.
- **Grantable OWNER / transfer** — briefly adopted, removed as overkill: a
  tier plus orphan-guard for cases the admin backstop and fork already
  cover. Re-adding an OWNER level later is additive.
- **A legacy `WorksheetService` shim** (derive `visibility` best-effort,
  translate legacy writes to a snapshot policy write) — after the rename the
  shim is an entire second service to maintain, secure, and audit, for a
  feature whose real consumer ships in lockstep. Hard cutover instead.

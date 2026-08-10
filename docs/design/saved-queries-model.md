# Saved queries: sharing, permissions, and API

2026-08-09. Redesign of the SQL Editor's saved-query model — sharing,
permissions, API, storage — and a rename from "worksheet" (see Naming).
Replaces the three-state visibility flag with the industry-consensus model:
per-object graduated grants to users and groups, private by default, with
per-project discovery.

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
   *discovery* (Search, always per-project) is gated by a dedicated
   `bb.savedQueries.search` permission carried by the human project roles —
   automation roles stay off the surface (`manage` passes too — the admin
   backstop);
   *running* is gated by the SQL Editor's own database permissions. Leaving
   a project ends discovery and run; explicit grants persist until
   revoked.
4. The **creator is the fixed owner** (no transfer); admins
   (`bb.savedQueries.manage`) manage any saved query in scope, private
   included.
5. **Creation is gated** by `bb.savedQueries.create` on the parent project.
6. A cross-project **auditor** (`bb.savedQueries.list`) enumerates/filters
   saved queries (e.g. by creator) independent of grants — offboarding
   review, incident audit.
7. **Listing scales**: My / Shared / Starred are separate indexed queries
   with keyset pagination; group sharing never expands group memberships.
8. **Rename** worksheet → saved query, bundled with the already-breaking
   API change.

The frontend mirrors these rules for control visibility only; enforcement is
server-side.

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
manual re-grant. This design accepts the same class of tradeoff — the
share-with-project snapshot (see Model) — rather than inventing a
project-derived principal.

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
   `bb.savedQueries.search` — or `manage`, the admin backstop — gates
   `SearchSavedQueries` (per-project), and running always passes the SQL
   Editor's database grants. The real cost:
   removing a user from a project ends discovery and run but not
   explicitly granted content access — the grants themselves are the
   revocation surface (see Model: Offboarding). The stricter
   membership-AND-grant conjunction was adopted in an earlier revision and
   dropped as stricter than any peer (see Alternatives).
4. **Creator = fixed owner, no transfer** (G4). Scratchpad-grade
   simplicity: no extra tier, no orphan-guard machinery; the admin backstop
   and fork-a-copy cover the departed-creator case.
5. **Admin (`manage`) reads private.** Peer parity (`dataform.admin`,
   Databricks workspace admins), and what makes incident response and
   orphan cleanup possible. The real tradeoff: admins can read private
   drafts — accepted, and made accountable by auditing content reads (see
   API and authorization: Audit events).
6. **A purpose-built level policy, not the role-based `IamPolicy` proto.**
   Per-object access is a *capability* (view/edit), as every peer models
   it; reusing `IamPolicy` would force per-object pseudo-roles and a
   resource-level role store. The cost — a second policy shape in the API —
   is softened by both shapes reducing to *(members, capability)*, so the
   bindings can be re-expressed as `IamPolicy` later without a data change.
7. **A `bb.savedQueries.list` auditor permission** (G6). Governance needs
   grant-independent, cross-project enumeration; `list` is the read-only
   counterpart to `manage`. It can read all content in scope — that is its
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

A saved query carries a policy of per-object grants; the `creator` is the
immutable owner; a private saved query has no bindings.

```
Grant levels:  VIEWER (open, read)  <  EDITOR (+ write content/title/database)
Principals:    user:{email} | group:{email}
```

The whole model (G1); every RPC gate derives from these. Notation: `u` is
the caller, `P` a project, `s` a saved query, `s.project` its owning
project; "holds" means any of `u`'s role bindings — project-level (direct
or via a group) or workspace-level — grants the permission. Binding
conditions are handled precisely: expiry (`request.time`) is honored, but
a binding whose condition scopes *resources* (databases, environments)
confers **no** `bb.savedQueries.*` permission at all — saved-query
surfaces are project-wide, and a data-slice grant must not silently widen
to them. The generic interceptor cannot enforce this (it evaluates
`request.time` only and passes residual resource conditions), so the rule
lives where saved-query code evaluates permissions: the CUSTOM gates and
`list`'s handler re-check — `create` alone rides the generic evaluation,
exposing no content (see Per-RPC access). Same narrowing stance as the
snapshot's condition skip:

```
admin(u, P)    = u holds bb.savedQueries.manage on P          -- the dataform.admin backstop
grant(s, u)    = max level (VIEWER|EDITOR) among s.bindings whose principal is in principals(u)

create(u, P)   = u holds bb.savedQueries.create on P
discover(u, P) = u holds bb.savedQueries.search on P          -- gates Search, not content
read(s, u)     = admin(u, s.project) OR u == s.creator OR grant(s, u) >= VIEWER
write(s, u)    = admin(u, s.project) OR u == s.creator OR grant(s, u) >= EDITOR
share(s, u)    = admin(u, s.project) OR u == s.creator        -- manage the policy
delete(s, u)   = admin(u, s.project) OR u == s.creator
```

The four role permissions the rules reference (G3–G6) — everything between
them (who reads, edits, re-shares a *specific* saved query) is that query's
own bindings:

| Permission | Meaning |
|---|---|
| `bb.savedQueries.create` | Create saved queries in the project |
| `bb.savedQueries.search` | Discover: the per-member Search and folder list (results still filtered to `read(s,u)`) |
| `bb.savedQueries.list` | Auditor: enumerate/filter in scope (project or `projects/-`), read-only, ignoring grants |
| `bb.savedQueries.manage` | Admin: read/write/re-share/delete any saved query in scope, incl. private |

There is no `get` permission — bindings decide reads (`bb.worksheets.get`
is dropped, not renamed). `search` is a dedicated permission rather than a
ride on `bb.projects.get` (see Alternatives) so custom roles can control
the saved-query surface independently of project visibility; the decoupled
combos are API-level shapes — the UI needs `bb.projects.get` to render
project context, and every search-carrying role also holds it. `list` and
`manage` can read content in scope — that is their purpose — so neither
belongs in default member roles (decision 7). Role mapping: the human project roles — `projectOwner`,
`projectDeveloper`, `sqlEditorUser`, `sqlEditorReadUser`, `projectViewer` —
carry `create` and `search`; the automation roles (`projectReleaser`,
`gitopsServiceAgent`) carry **neither**, keeping CI principals off the
saved-query surface entirely — which is what makes the share-with-project
snapshot's role filter meaningful for them. `projectOwner` carries
project-scoped `list` + `manage`; `workspaceAdmin`/`workspaceDBA` carry
**all four** workspace-scoped — they are human operator roles, and an
admin who could manage everyone's queries yet not draft their own would
be absurd. `manage` implies the Search gate — Search evaluates
`discover ∨ admin` in the handler — so a manage-only custom role can
still enumerate what it manages; there is no hidden permission
coupling.

The gates are **layered, exactly as new BigQuery enforces them** (G3): the
per-object grant carries content access (`codeViewer/Editor` on the asset);
a dedicated permission gates **discovery** — `SearchSavedQueries`
requires `bb.savedQueries.search` or `manage` (the backstop must be able
to enumerate what it manages), the exact `dataform.repositories.list`
analog, always per-project (no cross-project "shared with me"; no peer
has one); and
**running** always passes the SQL Editor's own database grants, the
`bigquery.jobUser` analog. A grantee who is not a project member opens the
saved query by its link — BigQuery's own sharing flow, "grant the role,
then share the link" — but cannot browse the project or run anything in
it. Databricks and Snowflake enforce their container structurally (an ACL
can only name a workspace/account user); BigQuery and this design let a
grant name any workspace identity, expressing the container through the
discovery and run gates instead.

**Offboarding.** Removing a user from a project ends discovery and run
immediately (workspace-wide roles aside — those reach every project by
design). Explicitly granted content access — and the creator's access to
their own queries — persists until revoked, and the revocation surface is
explicit: an offboarding review is **two `ListSavedQueries` filters** over
the target. `creator == target` lists the rows they own — no binding
exists for these, so no member filter can find them (remediation: an
admin deletes them, or deliberately leaves the departed user their own
drafts). `member == target`, given a `user:` principal, **expands the
target's group memberships server-side** — the same `principals(u)`
expansion the read path uses — listing rows granted directly *and* via
any of the user's groups (a `group:` principal matches that group's
rows); direct grants strip via `SetIamPolicy`, while group-held access is
revoked at the group (leave or edit it), not per-query — a group grant's
audience is deliberately the live group. Content reads are audited, and
workspace deactivation ends everything. This is precisely BigQuery's
behavior. The audit's Problem-1 complaint was never retention itself but
*implicit* retention — flag-derived access with no grant to see, audit,
or revoke.

**Share-with-project is a snapshot.** There is deliberately no derived
"everyone in the project" principal (see Alternatives); the share dialog's
*Share with project* shortcut writes the project's current IAM principals —
its `group:` and `user:` bindings, not an expansion of members — into the
policy at the chosen level. Only bindings **whose role carries
`bb.savedQueries.search`** are included: a role deliberately kept off the
saved-query surface (a CI releaser, say) must not be swept into content
grants that outlive its project role. Snapshots keep group dynamism (a group
gaining a member propagates, since the entry stores the group reference) but
not membership dynamism (a principal added to the project later is re-shared
by the owner, as in every group-based peer). Project bindings carrying a CEL
`condition` (expiring or scoped access) are **skipped**: a time-bounded
grant cannot be faithfully frozen into an unconditional one, and for a
governance product under-granting (the owner re-shares) beats silently
converting expiring access into permanent access.

**Ownership does not transfer** (G4): the creator alone (plus admins)
re-shares or deletes. Transfer/co-ownership would add a tier and an
orphan-guard for a case the admin backstop and fork-a-copy already cover.
**The admin backstop** (`bb.savedQueries.manage`) matches `dataform.admin`
and Databricks workspace admins: read/write/re-share/delete anything in
scope, private included — a deliberate privacy reversal (decision 5) that
makes orphan cleanup and incident response possible.

### API and authorization

Sharing is a policy on the resource, managed via a
`GetIamPolicy`/`SetIamPolicy` pair — new BigQuery's shape; Databricks and
Snowflake likewise keep permissions on a dedicated surface. Because Bytebase
has no resource-level role store below the project, the policy carries
capability **levels** grouped by member list, evaluated by the saved-query
handlers, not central `CheckPermission` (decision 6). `SetIamPolicy` is
**compare-and-swap**: `GetIamPolicy` returns an `etag`, `SetIamPolicy`
must present it, and a mismatch aborts for refetch — a full-replacement
write may never silently overwrite a concurrent revocation (the
offboarding race).

**Per-RPC access.** `CreateSavedQuery` and `ListSavedQueries` are IAM — a
single family permission checked at the interceptor. The platform's
generic check evaluates conditions with `request.time` only, which is
acceptable for `create` (it exposes no one's content) but not for the
content-bearing `list`: its handler **re-checks the permission source and
rejects resource-conditioned bindings**, enforcing the family condition
rule. The search family (`SearchSavedQueries`, `ListSavedQueryFolders`)
is CUSTOM — its gate is `discover(u, P) ∨ admin(u, P)`, an OR the
single-permission interceptor cannot express — and every object method is
a CUSTOM per-row predicate. `SearchSavedQueries` is the per-member view
(concrete project only); `ListSavedQueries` is the auditor view, which is
why only it may wildcard `projects/-`.

| RPC | Auth | Gate | Content / audit |
|---|---|---|---|
| `CreateSavedQuery` | IAM `create` | creator becomes owner | writes own; audited |
| `SearchSavedQueries` | CUSTOM | `discover ∨ admin` → rows where `read(s,u)`; admin: all | previews, metadata-only on override rows; unaudited |
| `ListSavedQueries` | IAM `list` + handler source re-check | all rows matching `filter`; `projects/-` allowed | full content (FULL); audited |
| `GetSavedQuery` | CUSTOM | `read(s,u)`; NotFound when unreadable | full content; audited |
| `UpdateSavedQuery` | CUSTOM | title/content/database `write(s,u)`; `folder` creator/admin | returns content; override write emits audit event |
| `DeleteSavedQuery` | CUSTOM | `delete(s,u)` | — ; audited |
| `GetIamPolicy` | CUSTOM | `read(s,u)` | policy only; unaudited |
| `SetIamPolicy` | CUSTOM | `share(s,u)` + etag CAS | policy only; audited |
| `UpdateSavedQueryStar` | CUSTOM | `read(s,u)` — star any readable query | — ; unaudited |
| `BatchUpdateSavedQueries` | CUSTOM | rows the caller may re-file (creator; admin: any); mask limited to `folder` | count only; audited |
| `ListSavedQueryFolders` | CUSTOM | `discover ∨ admin` → the caller's own folder paths | paths only; unaudited |

Duplicate/fork needs no RPC: read what you can already see, then
`CreateSavedQuery` — the copy is yours and private.

**Audit events.** Today's `WorksheetService` carries **no**
`bytebase.v1.audit` annotations at all (sheet and database services do) —
this design closes that gap, under one invariant: **every path to another
user's private content is audited.** By annotation: `CreateSavedQuery`,
`GetSavedQuery` (content reads — what makes admin access attributable, the
accountability half of decision 5), `DeleteSavedQuery`, `SetIamPolicy`
(the share event), `BatchUpdateSavedQueries`, and the grant-bypassing
`ListSavedQueries`. The two unaudited RPCs that could return content are
closed differently: `SearchSavedQueries` — the hot list path — returns
**metadata only, no content preview, for rows the caller reaches solely
via the admin override** (neither creator nor grantee), funneling their
content through the audited `GetSavedQuery`; `UpdateSavedQuery` — the
autosave path, where per-keystroke audit rows drown signal — carries no
annotation, but the handler **emits an audit event when the write
exercises the admin override**, which also covers reading content via a
no-op update's response. Self and granted traffic through Search and
Update stays unaudited; the accepted cost is that a grantee's edits are
attributed only via `creator`/`update_time` — revisit with debounced audit
or version history if that ever matters. Star, folder listing, and
`GetIamPolicy` return no content and are unaudited.

**Compatibility: hard cutover, no shim** (decision 8). `WorksheetService`
and the `visibility` field are removed in the release that ships
`SavedQueryService`; the in-repo frontend ships against the new API in the
same release. Post-rename,
a shim is not a deprecated field but an entire second service to maintain
and secure — for a UI-owned feature. Cost: external worksheet scripts break
loudly at upgrade, documented in the changelog. Historical audit-log
entries keep old names as records.

### Storage and query scalability

The policy lives in `saved_query.bindings jsonb` — no separate access
table. The **stored shape is pinned**: the protojson *array* of
`Binding` messages, e.g.
`[{"level": "EDITOR", "members": ["group:eng@corp.com", "user:a@corp.com"]}]`
— the array must sit at the jsonb root for the `@>` probes below (a wrapped
`{"bindings": [...]}` would force expression indexes). This deviates from
the store-a-whole-message payload convention, deliberately; the store layer
gets its own `SavedQueryBinding` message (store protos never import v1).

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
`memberGroupsCache`: `principals(u) = {user:u} ∪ {group:g …}`, typically a
handful. The list query filters the table directly, one GIN probe per
principal:

```sql
-- The search gate already passed at the interceptor. An admin drops the
-- whole access clause. KEYSET cursor: the last row's (updated_at,
-- resource_id); the cursor predicate is omitted on the first page.
SELECT s.* FROM saved_query s
WHERE s.project = $P
  AND ( s.creator = $me                                  -- own (incl. private)
        OR s.bindings @> '[{"members":["user:me"]}]'
        OR s.bindings @> '[{"members":["group:g1"]}]'
        OR ... )               -- one @> per principals(u), each GIN-indexed
  AND (s.updated_at, s.resource_id) < ($cursor_at, $cursor_id)
ORDER BY s.updated_at DESC, s.resource_id DESC
LIMIT $n;
-- "My" adds creator = $me and drops the bindings OR; "Shared" adds creator <> $me.
```

Honest per-tab costs (G7):

- **My** (`creator = me`): btree `(creator, project, updated_at DESC,
  resource_id DESC)` — equality on the leading columns plus an
  index-ordered keyset scan → true `O(page)` at any scale. The index's
  `creator` prefix also serves the auditor's cross-project `creator`
  filter (no binding test needed there) — but not its ordering: with
  `projects/-` there is no `project` equality, so the audit list top-N
  sorts that one creator's rows. Bounded and cold, so no dedicated index;
  and because completeness matters more than recency on a governance
  surface, the audit list keysets on an immutable key (`created_at`,
  `resource_id`), immune to the autosave skips below.
- **Starred**: my rows in `saved_query_star` via its `principal` btree —
  row existence is the star. Fetch the (naturally small) set, join, drop
  rows I can no longer read (lazily deleting stale stars), sort.
  `O(my stars)` — bounded by my own behavior.
- **Shared with me**: the only tab touching groups — `K+1` GIN probes for
  `K` groups, BitmapOr'd into my accessible set `S`. Bitmap output is
  unordered, so each page top-N sorts `S`: `O(S log page)`, **not** pure
  keyset — the honest cost of the no-extra-table choice, acceptable because `S` is *my*
  reachable set (tens to hundreds), not the project's total.

Pagination is **keyset, never OFFSET**, on every tab: where the plan is
index-ordered (My) pages are `O(page)`; where it is bitmap-based (Shared)
the cursor still avoids OFFSET's re-scan and caps the sort at top-N. One
honest limit of the mutable sort key: `updated_at` only moves forward, so
already-returned rows never repeat, but a row autosaved mid-scroll jumps
ahead of the cursor and drops out of the remaining pages — it surfaces at
the top of the next refresh. Acceptable for a scratchpad list; the auditor
view keysets on an immutable key instead (above). And the expensive move —
expanding a *group's members* — never happens on any path: policies store
group references, so a 1,000-member group costs the same as a 3-member one
and membership changes rewrite nothing.

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

### Sharing and organization UX

- Views, all within the current project: **My** (`creator == me`, organized
  in my folder tree), **Shared** (a binding grants me VIEWER+, flat
  searchable list), **Starred** (my favorites, own or shared). Nobody
  fetches a query they have no grant on.
- The edit-vs-view affordance reads `effective_level` off each row
  (server-computed, on the resource) — no per-row `GetIamPolicy` fan-out;
  creator and admin status the UI already knows.
- Share dialog (creator or admin): add `user:`/`group:` at VIEWER/EDITOR;
  the picker suggests project members/groups (a non-member grantee opens
  via link but cannot browse the project or run — see Model). **Share with
  project** = the snapshot shortcut (Model). Admins
  get a manage affordance on any query. Duplicate is always available.
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
  one folder, set by its owner; you never re-file someone else's item.
  Storage follows the semantic: the location is the `saved_query.folder`
  **path column on the object** ("a/b/c", '' = unfiled), set via
  `UpdateSavedQuery` (creator/admin only — re-filing is organization, not
  editing), bulk rename/move via `BatchUpdateSavedQueries`. Folders never
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
  deciding read/edit of all shared queries) — the deprecated generation of
  the reference product; the sharer cannot scope who edits. An earlier
  revision recommended it for proximity to current RBAC; proximity is not a
  direction.
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
  (decision 5); narrow `manage` back if the privacy guarantee matters more.
- **Grantable OWNER / transfer** — briefly adopted, removed as overkill: a
  tier plus orphan-guard for cases the admin backstop and fork already
  cover. Re-adding an OWNER level later is additive.
- **A legacy `WorksheetService` shim** (derive `visibility` best-effort,
  translate legacy writes to snapshot SetIamPolicy) — after the rename the
  shim is an entire second service to maintain, secure, and audit, for a
  feature whose real consumer ships in lockstep. Hard cutover instead.

# Bytebase v1 API — audit

What's wrong with the v1 API, in plain terms.

Audited against `93671b00b0` (2026-08-04); every finding below re-verified against `9e117d42e2`
(2026-08-27), with HEAD line numbers. Findings are code-traced, not runtime-proven, except where
noted. Fixed findings are removed and listed at the end.

## The short version

| | Problem | Severity |
|---|---|---|
| T15 | Rotating a leaked cloud credential can silently do nothing | HIGH |
| T13 | "Waiting for my approval" can show an empty page with a live Load More | HIGH |
| T18a | Test-environment rights can cancel a running production migration | MED |
| T12 | Anonymous callers can enumerate emails, relay mail, and read your LDAP config | MED |
| T16 | Four `BatchGet` RPCs behave four different ways when an item is missing | MED |
| T18b | Concurrent permission edits silently lose one side, including a revoke | MED |
| T11 | A create-permission check is skipped on `allow_missing` updates | MED |
| T17 | Typing `"` in a search box breaks search | MED |
| T18c | Three smaller ones: broken filter, unpaginated list, half-finished delete | MED |

---

## Doing more than you're allowed

### The one-resource ACL rule, and what it still costs

`getResourceFromSingleRequest` authorizes exactly one resource name per request, picked by field-name
convention: `parent`, else `name`, else `resource`, else `project`, else — for `Create`/`Update`/
`Remove`/`Test` — the nested `<snake_case(resource)>.name`. Nothing else in the body is visible to it.
`acl.go:735-795`

A sweep of all 216 v1 RPCs found 24 whose body names a resource the interceptor never authorizes.
Twenty-two are closed downstream: `CreatePlan`/`UpdatePlan`, the three `Release` RPCs,
`BatchCreateRevisions`, `CreateAccessGrant`, the `SavedQuery` writes, `CreateIssue`,
`UpdateIssueComment`, `TestWebhook` and `GetQueryHistory` each compare the second resource's project
to the authorized one; `BatchRunTasks`, `BatchSkipTasks` and `CreateRollout` re-derive it from the
parent instead of trusting the name; `UpdateDatabase`/`BatchUpdateDatabases` are special-cased in the
interceptor itself. `DiffSchema` was the one live gap and is now closed the same way.

The convention can also miss the *only* resource: `UpdateDatabaseCatalog` names its target in
`catalog`, not the `database_catalog` the method suffix derives, so it resolved nothing and every
call was authorized against the workspace. Both are fixed and listed at the end. Neither fix
generalizes — the next request field that departs from the convention arrives unchecked, silently,
and only a sweep like this one finds it.

### T18a · Test-environment rights can cancel a production migration — MED

`BatchCancelTaskRuns` checks your permission against the stage *you claim in the URL*, then cancels
task runs by ID without ever confirming those IDs belong to that stage. So permission to cancel in
test is enough to kill an in-flight production migration in the same project. Sibling handlers get
this right by reading the environment off the actual rows. `rollout_service.go:1040-1154`

### T11 · A create check is skipped on `allow_missing` updates — MED

`UpdateGroup(allow_missing=true)` creates a group when none exists, but the create-permission check
is short-circuited for CUSTOM-auth RPCs, so nothing is verified. Harmless today only because every
member can create groups anyway — but the code comment right above it claims the check happens,
which is how this survives review.

Same root cause, wider: 15 of the 45 CUSTOM RPCs declare a `permission` annotation that is never
enforced. It's decorative, and it reads as protection. `acl.go:251-253`, `group_service.go:167-177`

---

## The API says OK and does nothing

### T15 · `update_mask` is ignored in four places — HIGH

The worst one: **rotating a leaked cloud credential can silently no-op.** `UpdateDataSource` decides
what to write based on the `authentication_type` in the request rather than the mask or the stored
value, so a request carrying only a new GCP credential falls through an empty branch and writes
nothing. You get HTTP 200 and the compromised key stays live. `instance_service.go:1405-1447`

The other three lose data rather than fail to save it:

- **`UpdateSetting` replaces whole settings.** Updating your Slack config wipes stored Feishu, WeCom,
  Lark, DingTalk and Teams secrets — and it *validates* the mask path first, then ignores it. For
  environments the replace cascades into every instance and database. `setting_service.go:329-391`,
  `:505-525`
- **`UpdateDatabaseCatalog` never reads a mask at all** and replaces the whole config column, wiping
  other schemas' classification and masking. Its proto comment refers to an `update_mask` field that
  doesn't exist. `database_catalog_service.go:96-139`
- **`UpdateIssue` implements four fields** and silently drops the rest, so `update_mask=["status"]`
  returns 200 and does nothing. `issue_service.go:922-948`

Underneath: some Update RPCs reject unknown mask paths, others ignore them, in four different error
wordings. There's no shared validator.

### T18b · Concurrent permission edits silently lose one side — MED

`IamPolicy` has an etag for exactly this, and the docs promise it triggers ABORTED — but the
handlers only check the etag on the *request wrapper*, never the one inside the policy. A normal
read-modify-write round-trips the policy etag and has it silently discarded, so two admins editing
permissions at once means one edit vanishes. If the lost edit was a revoke, access stays granted.

The saved-query path already implements this correctly, so the pattern is in-tree.
`project_service.go:601`, `workspace_service.go:437`

### T18c-i · Deleting several projects can half-finish — MED

`BatchDeleteProjects` purges in a loop with no transaction. A failure partway leaves some projects
irreversibly purged, and the error names only the one that failed — not the ones already gone.
`project_service.go:495-504`

---

## Lists that return the wrong rows

### T13 · Issue filters run after the page is cut — HIGH

`approval_status` and `current_approver` never reach SQL. The database returns a full page, the page
token is minted, and *then* non-matching issues are dropped in Go. So filtering by "waiting for my
approval" in a busy workspace shows an empty list with a live "Load more" button underneath.

This is the default My Issues view, and there's no empty-page workaround on the frontend.
`issue_service_converter.go:29-41`, `issue_service.go:134-148`

(Same shape as the `SearchProjects` bug that parked patch `03` fixes — also still unmerged.)

### T16 · Four `BatchGet` RPCs, four behaviors — MED

Ask for five databases and get three back, with no way to tell which two are missing. Ask for five
projects with one missing and the entire call 404s. `BatchGetUsers` returns store order, not request
order. `BatchGetGroups` swallows every error, so a database failure is indistinguishable from "that
group doesn't exist."

None of this is documented, and the responses are bare repeated lists — so any client matching
`names[i]` to `results[i]` silently mis-attributes data. Contradicts AIP-231.

### T18c-ii · Two smaller list bugs — MED

- **The query-history `instance ==` filter always returns nothing.** It compares an instance name
  against a longer stored value using `LIKE` with no wildcard. The proto documents the broken form.
  `store/query_history.go:257`
- **`ListTaskRuns` has no pagination and no limit** — and its documented wildcard parent pulls an
  entire rollout in one call, on the table that grows fastest. `store/task_run.go:64-130`

### T17 · Typing a quote breaks search — MED

The frontend builds CEL filter strings by interpolating raw user input. There's an escaping helper
that exists precisely for this, used at 6 sites; 11 others still interpolate raw. Type a `"` in
those search boxes and search fails with `InvalidArgument`. It also allows filter-semantics
injection — not a privilege boundary, since the filter only narrows an already-authorized list.

Worst site: `accessGrant.ts:35` interpolates SQL statement text raw, while lines 37-40 right next to
it do it safely.

---

## What anonymous callers can learn

### T12 · The unauthenticated surface — MED

12 RPCs allow calls with no credentials. Four are worth attention:

- **Email enumeration.** `Signup` answers "already registered" *before* it checks whether signup is
  even allowed, so anyone can test whether an email has an account — on a workspace with signup
  disabled. `auth_service.go:298-304` (existence), `:327-329` (restriction)
- **Open mail relay.** `SendEmailLoginCode` will send to any address over the customer's SMTP.
  [#21177](https://github.com/bytebase/bytebase/pull/21177) added a domain check, but it only
  applies if the domain-restriction license feature is on, enforcement is enabled, *and* a domain
  list is set — no default deployment qualifies. The 60s cooldown is per recipient, so volume scales
  with the address list. `auth_service_email_code.go:137-179`, `user_service.go:1443-1473`
- **LDAP config disclosure.** `ListIdentityProviders` hands any caller the SSO config for any
  workspace ID they guess: LDAP host, port, bind DN, base DN, user filter, plus OAuth/OIDC endpoints
  and client IDs. Passwords and secrets are redacted. `idp_service.go:55-85`, `:461-541`
- **Workspace-existence oracle.** `GetAuthenticationRestriction` (which replaced the actuator leak,
  now fixed) is anonymous by design and doesn't need membership. Naming a real workspace returns
  200, a fake one returns `InvalidArgument`. The pre-login page genuinely needs most of these
  fields; the oracle is the part worth closing. `auth_service.go:107-152`

### T18c-iii · `CheckRelease` error codes leak existence — MED · ✅ read myself

Point it at a database in someone else's project and you get `InvalidArgument`; point it at one that
doesn't exist and you get `NotFound`. So permission to check releases on any single project lets you
map which databases exist workspace-wide. [#21102](https://github.com/bytebase/bytebase/pull/21102)
added a third distinguishable error that also leaks instance ownership.

**Fix:** return `NotFound` for both, matching the convention `BatchCreateRevisions` already adopted,
with a regression test. `release_service_check.go:77-110`

---

## Structural / AIP (not re-run)

Numbers below are the 2026-08-04 baseline; the worksheet→saved-query rename has since moved some of
this surface. Re-measure before acting.

An api-linter run produced 3808 raw findings, of which **567 are substantive** — the rest are opted-out
field-behavior rules, missing comments, and Google-internal package rules. They concentrate in
`database_service`, `rollout_service`, `subscription_service`, and `instance_service`. It confirms
the hand audit rather than contradicting it: the same 12 unpaginated List RPCs, the same 7 `state`
fields missing `OUTPUT_ONLY`, the same enum zero-value naming. Also still open: 28 RPCs lack the
mandated `// Permissions required:` comment, and 22 files have string fields with no protovalidate
constraints.

---

## What I'd do, in order

1. **Make credential rotation fail loudly (T15).** A security operation that returns 200 and does
   nothing is worse than one that errors.
2. **Make the extractor's misses loud.** A `Create`/`Update` request whose conventional field is
   absent still degrades silently to a workspace check — `UpdateDatabaseCatalog` sat that way until
   it was found by sweep, not by failure. Rejecting it at startup, the same shape as the
   `AUTH_METHOD_UNSPECIFIED` gate below, turns the next `catalog`-style field name into a build
   failure instead of a silent scope change.
3. **Fail closed on auth annotations, and put api-linter in CI.** Reject
   `AUTH_METHOD_UNSPECIFIED` at startup; make a `permission` on a CUSTOM RPC either enforced or a
   build error, since 15 are decorative. None of Tier 5 was visible to `buf lint`'s BASIC profile,
   which is why it accumulated.

---

## Already resolved

| Finding | Resolution |
|---|---|
| T1, T3 | Fixed earlier in the session |
| T2 (INPUT_ONLY) | Read-path fix |
| T2 `DiffSchema` | `DiffSchema` rejects a changelog target owned by another project before any read, matching how every other second-resource handler closes it. Not the general interceptor change: the resource the caller names must simply belong to the project they were authorized on, which is also the only thing a cross-project diff could have meant. A missing target and a foreign one return the same error, so neither confirms what lives in the other project. Accepted with it: the class stays latent — a future second-resource field still arrives unchecked |
| T19 `UpdateDatabaseCatalog` | The interceptor now reads `catalog.name`, so the permission is checked against the named database's project instead of the workspace — the route resolver already drops the trailing `/catalog`. Strictly additive: workspace holders are unaffected and Project Owner, whose role carries `bb.databaseCatalogs.update`, is no longer denied. Separate and still open: `bb.databaseCatalogs.create` does not exist, so the `allow_missing` secondary check denies every caller. Dead rather than harmful — no UI sets the flag and the RPC is MCP-`EXCLUDED` |
| T5, T6 | [#21143](https://github.com/bytebase/bytebase/pull/21143) — `sheet_blob_ref` gives sheets per-project ownership; `BatchCreateRevisions` rejects foreign-project provenance without echoing the hash |
| T7 | [#21102](https://github.com/bytebase/bytebase/pull/21102) — every `CheckRelease` target validated against the parent project before any schema read (its error codes still leak existence — T18c-iii) |
| T8 `CreateWorksheet` | [#21169](https://github.com/bytebase/bytebase/pull/21169) — `CreateSavedQuery` is now IAM-enforced; Workspace Member holds no saved-query permission, and new queries start creator-private |
| T12 `GetActuatorInfo` | [#21184](https://github.com/bytebase/bytebase/pull/21184) — anonymous access and the `name` field removed; workspace now comes from the token (the narrower surface that replaced it is T12) |
| T13 `SearchWorksheets` | [#21160](https://github.com/bytebase/bytebase/pull/21160) + [#21178](https://github.com/bytebase/bytebase/pull/21178) — visibility predicate pushed into SQL before `LIMIT` |
| T18 worksheet write/delete | [#21169](https://github.com/bytebase/bytebase/pull/21169) + [#21181](https://github.com/bytebase/bytebase/pull/21181) — per-verb permissions; SQL Editor Read User can no longer rewrite or delete |
| T9 | [#21189](https://github.com/bytebase/bytebase/pull/21189) (self-hosted audit rows) + [#21234](https://github.com/bytebase/bytebase/pull/21234) — one `login_attempt` table bounds password, email-code, and MFA guessing per identity on both deployments, replacing the audit-log counter and the per-code attempt column (which bypassed the resend cooldown). Accepted with it: no per-tenant failed-login record on Cloud, and lockout-as-denial-of-service — see [`login-attempt-lockout.md`](login-attempt-lockout.md) |
| T14 | [#21267](https://github.com/bytebase/bytebase/pull/21267) — filed as issue paging; the sweep it prompted found the same defect across the class, so it was fixed as a class. Counting endpoints, **17 of the 22** offset-paginated v1 list RPCs were affected; counting the store functions behind them, **14 of 17**. The three already sorting on a total order were `ListQueryHistories` and `ListSavedQueries` (both from one commit, [#21203](https://github.com/bytebase/bytebase/pull/21203), that was never swept across the other lists) and `ListPlans`, which is total only because a mandatory `WHERE plan.project = ?` pins its scope column. Every store list now names tiebreak columns that are unique under its own scope — written into the SQL, with the same columns appended after the caller's keys at the seven lists that accept an `order_by`. Nothing enforces this statically: an earlier revision routed all seventeen through a `buildStableOrderBy` helper so an AST test could require it, but review judged that ceremony and both were removed, so a new paginated list added without a tiebreak will not fail CI. Enforcement is step 4 of `docs/pre-pr-checklist.md` plus review, with `TestPaginationStabilityAcrossProjects` and `TestIssueCommentBatchKeepsInsertionOrder` covering the behavior against a real PostgreSQL. The rules, and the five traps that produced the class — `id` alone in a `(project, id)` table, `created_at` (the *transaction* timestamp, identical across a batch insert), a nullable column, a partial unique index, and an `order_by` that replaces the default ordering rather than adding to it — are in [`backend/store/AGENTS.md`](../../backend/store/AGENTS.md#pagination-ordering). Two adjacent bugs fixed with it: issue search silently discarded the caller's `order_by`, and `ListDatabases`/`ListInstances`/`ListProjects` each threw away a correct default sort the moment a caller passed `order_by`. Accepted with them: offset paging still drifts under concurrent inserts and deletes (that needs keyset pagination and a page-token change). `ListIssueComment` needed a second fix: its `resource_id` tiebreak is a random UUID, which would have scrambled the activity feed of a multi-field `UpdateIssue`, so `CreateIssueComments` now offsets each row of a batch by its ordinal to keep `created_at` unique and in insertion order. Cross-project issue lists now lead with `created_at`, since `issue.id` is a per-project counter — the newest issue in a new project ranked 4962nd under the old default. Single-project lists keep `id`, where it is exactly creation order and is served from `issue_pkey` as an ordered index scan |
| T10 | [#21252](https://github.com/bytebase/bytebase/pull/21252) + [#21258](https://github.com/bytebase/bytebase/pull/21258) — password change and MFA lifecycle move off `UpdateUser` onto their own methods, each requiring a `CredentialProof`: current password, live OTP, recovery code, or a Cloud-only emailed re-auth code. Every proof claims a T9 login-attempt slot, so no proof channel is an unbounded guessing oracle, and a factor-touching method refuses the password while a live factor exists. Accepted with it: **the stolen access token still answers until it expires (≤1h)** — credential generation and the fenced transaction were cut or deferred, password change revokes only the account's web refresh tokens (best-effort, OAuth grants untouched), and an MFA change revokes nothing. What this closes is the credential being *spent* on its own replacement, not the session. Design, and the shipped-vs-designed delta: [`reauthenticate-credential-changes.md`](reauthenticate-credential-changes.md) |

**`AIService.Chat` — accepted, not fixed** (2026-08-18). `Chat` requires an authenticated workspace
principal and touches no database or sheet, so the only exposure is LLM spend. On Cloud the feature
is free on Bytebase's own key, so no customer budget is at risk and abuse is our own spend-management
problem. Self-hosted, the admin supplies the org's key when enabling it, so org-wide use by members
is the intended model. The missing per-user rate limit and audit are accepted with that disposition.

Two patches from earlier in the session (`SearchProjects` proto comment, `SearchProjects` pagination)
remain parked and still apply — neither landed independently, and both targets are still present at
HEAD. The third, `DiffSchema` ACL, is superseded by the same-project fix above.

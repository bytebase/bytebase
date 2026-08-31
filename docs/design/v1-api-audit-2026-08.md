# Bytebase v1 API — audit

What's wrong with the v1 API, in plain terms.

Audited against `93671b00b0` (2026-08-04); every finding below re-verified against `9e117d42e2`
(2026-08-27), with HEAD line numbers. Findings are code-traced, not runtime-proven, except where
noted. Fixed findings are removed and listed at the end.

## The short version

| | Problem | Severity |
|---|---|---|
| T12 | Anonymous callers can enumerate emails, relay mail, and read your LDAP config | MED |
| T16 | Four `BatchGet` RPCs behave four different ways when an item is missing | MED |
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

---

## The API says OK and does nothing

### T18c-i · Deleting several projects can half-finish — MED

`BatchDeleteProjects` purges in a loop with no transaction. A failure partway leaves some projects
irreversibly purged, and the error names only the one that failed — not the ones already gone.
`project_service.go:495-504`

---

## Lists that return the wrong rows

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

1. **Make the extractor's misses loud.** A `Create`/`Update` request whose conventional field is
   absent still degrades silently to a workspace check — `UpdateDatabaseCatalog` sat that way until
   it was found by sweep, not by failure. Rejecting it at startup, the same shape as the
   `AUTH_METHOD_UNSPECIFIED` gate below, turns the next `catalog`-style field name into a build
   failure instead of a silent scope change.
2. **Fail closed on auth annotations, and put api-linter in CI.** Reject
   `AUTH_METHOD_UNSPECIFIED` at startup; make a `permission` on a CUSTOM RPC either enforced or a
   build error, since 16 of the 52 are decorative — `TestAllowMissingCreatePermission`, added with
   T11, is the descriptor-walking shape to generalize. None of Tier 5 was visible to `buf lint`'s
   BASIC profile, which is why it accumulated.

---

## Already resolved

| Finding | Resolution |
|---|---|
| T1, T3 | Fixed earlier in the session |
| T2 (INPUT_ONLY) | Read-path fix |
| T2 `DiffSchema` | `DiffSchema` rejects a changelog target owned by another project before any read, matching how every other second-resource handler closes it. Not the general interceptor change: the resource the caller names must simply belong to the project they were authorized on, which is also the only thing a cross-project diff could have meant. A missing target and a foreign one return the same error, so neither confirms what lives in the other project. Accepted with it: the class stays latent — a future second-resource field still arrives unchecked |
| T19 `UpdateDatabaseCatalog` | The interceptor now reads `catalog.name`, so the permission is checked against the named database's project instead of the workspace — the route resolver already drops the trailing `/catalog`. Strictly additive: workspace holders are unaffected and Project Owner, whose role carries `bb.databaseCatalogs.update`, is no longer denied. Separate and still open: `bb.databaseCatalogs.create` does not exist, so the `allow_missing` secondary check denies every caller. Dead rather than harmful — no UI sets the flag and the RPC is MCP-`EXCLUDED` |
| T11 | `UpdateGroup(allow_missing=true)` now checks `bb.groups.create` itself, the shape `UpdatePlan` and `UpdateUser` already had. The interceptor never covered it: `doIAMPermissionCheck` returns true for every non-IAM auth method, so the `allow_missing` secondary block evaluated to nothing on a CUSTOM RPC, and the create path calls `CreateGroup` in-process, which bypasses the interceptor anyway — while the comment above it claimed both permissions were verified. The exposure was narrower and differently shaped than first written: Workspace Member does hold `bb.groups.create`, but Workspace DBA and every project role do not, so a DBA-only principal could create a group and name itself OWNER, which `checkPermission` then honors for that group. It stops there — a group confers nothing until someone with IAM-policy rights binds a role to it. The interceptor's block is now gated on `AuthMethod == IAM` so it no longer reads as protection where it verifies nothing (behavior-preserving: it already returned true there). Found tracing the same mechanism, and fixed with it: the `.update` → `.create` string rewrite is unvalidated, and `UpdateDatabase` derived `bb.databases.create`, which does not exist — `CheckPermission` never matches an unknown string, so `allow_missing=true` denied every caller including Workspace Admin, on a flag `database_service.go` never read and the proto documented as working. The field is removed and the proto now says never to add it back, since a database is not independently creatable. `TestAllowMissingCreatePermission` walks the descriptors and makes all four silent failure modes a build failure — an RPC that gains the flag with nothing authorizing its create path, a derived permission that does not exist, a declared permission the rewrite leaves unchanged (`bb.settings.set` is one, and vacuous if it ever went IAM), and a nested `allow_missing` sub-request the interceptor's hand-written type switch does not name, which it proves by building the request and asserting `hasAllowMissingEnabled` reads it. Still open, and the other half of the original finding: 16 of the 52 CUSTOM RPCs declare a `permission` annotation that nothing ties to enforcement, so it remains documentation that reads as protection |
| T5, T6 | [#21143](https://github.com/bytebase/bytebase/pull/21143) — `sheet_blob_ref` gives sheets per-project ownership; `BatchCreateRevisions` rejects foreign-project provenance without echoing the hash |
| T7 | [#21102](https://github.com/bytebase/bytebase/pull/21102) — every `CheckRelease` target validated against the parent project before any schema read (its error codes still leak existence — T18c-iii) |
| T8 `CreateWorksheet` | [#21169](https://github.com/bytebase/bytebase/pull/21169) — `CreateSavedQuery` is now IAM-enforced; Workspace Member holds no saved-query permission, and new queries start creator-private |
| T12 `GetActuatorInfo` | [#21184](https://github.com/bytebase/bytebase/pull/21184) — anonymous access and the `name` field removed; workspace now comes from the token (the narrower surface that replaced it is T12) |
| T13 `SearchWorksheets` | [#21160](https://github.com/bytebase/bytebase/pull/21160) + [#21178](https://github.com/bytebase/bytebase/pull/21178) — visibility predicate pushed into SQL before `LIMIT` |
| T13 issue filters | `approval_status` and `current_approver` are matched in SQL before `LIMIT`, so a filtered page can no longer come back empty under a live page token. The status is a `CASE` over `payload->'approval'` mirroring `computeApprovalStatus`; `current_approver` cannot be resolved in SQL alone (binding conditions and group expansion are evaluated in Go), so the caller passes down the `(project, role)` pairs the named user holds, the same shape as the saved-query `AccessMembers` predicate. Accepted with it: the predicates are not indexable, so a filtered list scans until it has a full page of matches — measured on 50,000 issues across 40 projects, that is ~9 ms against ~14 ms for the same unfiltered query, and ~3.5 ms in the zero-match case, because PostgreSQL drives the join from the small `(project, role)` set, so no expression index is warranted yet. Deliberately unchanged: the filter still ignores rejection and self-approval, so a rejected issue names the holder of the *next* role as its current approver — masked in the default view, and worth its own fix |
| T18 worksheet write/delete | [#21169](https://github.com/bytebase/bytebase/pull/21169) + [#21181](https://github.com/bytebase/bytebase/pull/21181) — per-verb permissions; SQL Editor Read User can no longer rewrite or delete |
| T17 | Every CEL string literal the frontend emits now comes from `celString`/`celStringList`/`celMapField` in a dependency-free `frontend/src/utils/v1/celLiteral.ts` — the helpers own the quotes, so the escape cannot be forgotten the way it was at 11 of 17 sites. Worst was the access-grant page, whose search box takes SQL: a quoted identifier made `InvalidArgument` routine. All 17 sites converted, the enum and resource-name ones included, because an absolute rule is the only one a machine can check — `src/architecture/cel-filter-literals.test.ts` scans `src/**` for the three raw shapes and fails CI on a new one, in the style of the existing Vue-boundary guards. The helpers are a leaf module because putting them beside the CEL RPC clients dragged `@/api` into `modules/cel/logic/stringify.ts` and broke that subtree's tests. Two adjacent defects the sweep found, neither of which escaping addresses, fixed with it. **A label key with a dash never reached the backend at all**: keys are validated `^[a-z][a-z0-9_-]{0,62}$`, and CEL parses `labels.cost-center` as subtraction, so the store answered `unsupport variable \"\"` — the frontend now emits `labels[\"key\"]` index syntax, `getVariableAndValueFromExpr` canonicalizes both forms, and the key is **bound** rather than interpolated into the JSONB path, since index syntax admits a quote the dotted form could not. **`%` and `_` reached `LIKE` as wildcards**, so a statement search containing either silently over-matched — `escapeLikePattern`, already in-tree for saved queries, plus `ESCAPE '\\'` now covers all 13 `contains` translations. The three copies of the label-filter closure collapsed into one `buildLabelFilterSQL`, which also drops an unchecked `raw.(string)` that panicked on `labels.k in [1]`. Verified against PostgreSQL 16: `->>$1::text` matches, an injected key is inert, and `\_` stops matching what `_` did. Accepted with it: the guard is line-based and skips test files, which should spell out expected filter strings literally; and `query_history`'s `instance ==`/`database ==` LIKE-without-wildcard filters are left alone — those are T18c-ii |
| T18a | [#21265](https://github.com/bytebase/bytebase/pull/21265) — `BatchCancelTaskRuns` confines its lookup to the plan and environment just authorized; the stage in a task run name is caller-supplied, so cancel rights in one environment reached a RUNNING row in another. An out-of-scope ID is rejected, not dropped — the cancel is driven by the requested IDs |
| T9 | [#21189](https://github.com/bytebase/bytebase/pull/21189) (self-hosted audit rows) + [#21234](https://github.com/bytebase/bytebase/pull/21234) — one `login_attempt` table bounds password, email-code, and MFA guessing per identity on both deployments, replacing the audit-log counter and the per-code attempt column (which bypassed the resend cooldown). Accepted with it: no per-tenant failed-login record on Cloud, and lockout-as-denial-of-service — see [`login-attempt-lockout.md`](login-attempt-lockout.md) |
| T15 | `update_mask` now decides what each of the four Update methods writes. **`UpdateDataSource`** dispatches the IAM credential on the mask path instead of the request's `authentication_type`, which was unset on any request that masked only the credential — the AIP-134 shape — so rotating a leaked key returned 200 and wrote nothing. The same branch, given a body type that disagreed with the mask, wrote the *other* credential and moved `authentication_type` with it though the mask named neither; the effective type is now resolved from the mask before the loop and a mismatched path is `InvalidArgument`. **`UpdateSetting(APP_IM)`** splices only the masked providers into the stored value rather than assigning the request wholesale, so saving Slack no longer erases the Feishu, WeCom, Lark, DingTalk and Teams secrets; the mask is now required, and a masked provider the payload omits is removed, which is how the console deletes one (it no longer merges the list client-side). **`UpdateIssue`** rejects paths it does not implement instead of returning 200 and dropping them — `status` belongs to `BatchUpdateIssuesStatus` and approvals to `ApproveIssue`/`RejectIssue`, so the supported set was right and only the silence was wrong. **`UpdateDatabaseCatalog`** keeps full-replace semantics and now says so in the proto; its `allow_missing` is removed and the number reserved, because a catalog is not independently creatable — it exists once its database is synced — so the flag had nothing to create, and the ACL's create-permission check on it denied every caller. Both merges were extracted into pure functions (`applyDataSourceUpdateMask`, `mergeAppIMSetting`), so `instance_service_test.go` and `setting_service_test.go` pin the behavior in milliseconds without a server or a metadata database. Accepted with it: the APP_IM mask paths still name no real proto field (`AppIMSetting` holds one repeated `settings`, and `value.app_im_setting_value.teams` is not a field path at all), so they are matched as a fixed vocabulary — making them true paths means restructuring the message one-field-per-provider, which needs a JSONB migration. Also left open, and the reason `UpdateDatabaseCatalog`'s hazard is documented rather than removed: a mask cannot express "this one table" while `schemas` is repeated, and read-modify-write on the catalog still loses one side of a concurrent edit, which needs an etag (AIP-154); the IAM-policy half of it is fixed, see T18b below |
| T14 | [#21267](https://github.com/bytebase/bytebase/pull/21267) — filed as issue paging; the sweep it prompted found the same defect across the class, so it was fixed as a class. Counting endpoints, **17 of the 22** offset-paginated v1 list RPCs were affected; counting the store functions behind them, **14 of 17**. The three already sorting on a total order were `ListQueryHistories` and `ListSavedQueries` (both from one commit, [#21203](https://github.com/bytebase/bytebase/pull/21203), that was never swept across the other lists) and `ListPlans`, which is total only because a mandatory `WHERE plan.project = ?` pins its scope column. Every store list now names tiebreak columns that are unique under its own scope — written into the SQL, and appended after the caller's keys at the six affected lists that accept an `order_by`. The three that were already total are left untouched. Nothing enforces this statically: an earlier revision routed all seventeen through a `buildStableOrderBy` helper so an AST test could require it, but review judged that ceremony and both were removed, so a new paginated list added without a tiebreak will not fail CI. Enforcement is step 4 of `docs/pre-pr-checklist.md` plus review, with `TestPaginationStabilityAcrossProjects` and `TestIssueCommentBatchKeepsInsertionOrder` covering the behavior against a real PostgreSQL. The rules, and the five traps that produced the class — `id` alone in a `(project, id)` table, `created_at` (the *transaction* timestamp, identical across a batch insert), a nullable column, a partial unique index, and an `order_by` that replaces the default ordering rather than adding to it — are in [`backend/store/AGENTS.md`](../../backend/store/AGENTS.md#pagination-ordering). One adjacent bug fixed with it: `ListDatabases`, `ListInstances` and `ListProjects` each threw away the whole clause, tiebreak included, the moment a caller passed `order_by` — which is what broke their paging. Accepted with them: offset paging still drifts under concurrent inserts and deletes (that needs keyset pagination and a page-token change). `ListIssueComment` needed a second fix: its `resource_id` tiebreak is a random UUID, which would have scrambled the activity feed of a multi-field `UpdateIssue`, so `CreateIssueComments` now offsets each row of a batch by its ordinal to keep `created_at` unique and in insertion order. Deliberately out of scope, and worth its own change: `issue.id` is a per-project counter, so ordering a cross-project list by it is not recency — seeded with a three-year-old project and one created today, the newest issue in the new project ranked 4962nd on the default list. Making `create_time` the cross-project default is a product decision. Also left alone: issue search still discards an explicit `order_by` outright when the caller supplied query text |
| T10 | [#21252](https://github.com/bytebase/bytebase/pull/21252) + [#21258](https://github.com/bytebase/bytebase/pull/21258) — password change and MFA lifecycle move off `UpdateUser` onto their own methods, each requiring a `CredentialProof`: current password, live OTP, recovery code, or a Cloud-only emailed re-auth code. Every proof claims a T9 login-attempt slot, so no proof channel is an unbounded guessing oracle, and a factor-touching method refuses the password while a live factor exists. Accepted with it: **the stolen access token still answers until it expires (≤1h)** — credential generation and the fenced transaction were cut or deferred, password change revokes only the account's web refresh tokens (best-effort, OAuth grants untouched), and an MFA change revokes nothing. What this closes is the credential being *spent* on its own replacement, not the session. Design, and the shipped-vs-designed delta: [`reauthenticate-credential-changes.md`](reauthenticate-credential-changes.md) |
| T18b | The etag is now read wherever the caller presents it: `policy.etag` — the only place `GetIamPolicy` returns one, and so the only place a read-modify-write round-trips it — as well as the request field, with two that disagree rejected. Both handlers write through `store.SetIamPolicy`, which compares under `SELECT ... FOR UPDATE` inside the write transaction. Comparing against the handler's own earlier read would not have been enough: that is a check-then-act, and two requests reading the same policy both pass it before either writes. (An earlier revision justified this by replica cache skew, which does not exist: `store.New(ctx, pgURL, !profile.HA)` disables read caching in the only mode that runs more than one process.) The same lock closed two merges that present no etag at all and so could never be compared — `PatchWorkspaceIamPolicy` (signup, SSO provisioning, workspace-leave) and the role-grant issue approval in `backend/utils` — each of which read the policy, merged one member in, and wrote the whole thing back, so either could resurrect a member an admin had just revoked; both now merge inside the locked transaction through `store.PatchIamPolicy`. Fixed with it: workspace `SetIamPolicy` wrote to `request.Resource` while every read and check used the workspace from the token, so a request naming anything else matched no row and returned 200 having written nothing — the write now targets the workspace the reads did, which is what `GetIamPolicy` already does by ignoring the field. The audit delta and the invite emails are now derived from the policy the write actually replaced rather than the pre-validation snapshot, so they describe what happened even for an etag-less caller, and the workspace seat guard counts against that same locked policy through a `ValidateReplaced` hook rather than against a read the write does not hold. Accepted with it: an etag-less caller still gets last-write-wins, which is what AIP-154 optional-etag semantics mean and what existing API clients rely on; and the etag is still `updated_at` in milliseconds, so a write landing in the same millisecond as the one before it leaves the etag unmoved and a stale write racing that pair is accepted — too narrow to justify the wire change to a content hash, unlike the saved-query policy, which has no timestamp to derive one from. Also left as is: `validateIAMPolicy` grandfathers bindings found in the policy the handler read before the write, so an etag-less caller racing a concurrent removal re-adds that binding without the `allUsers`, role-existence, CEL and maximum-expiration checks — pre-existing, and closing it means splitting the function into a fetch half and a pure half that runs under the lock |

**`AIService.Chat` — accepted, not fixed** (2026-08-18). `Chat` requires an authenticated workspace
principal and touches no database or sheet, so the only exposure is LLM spend. On Cloud the feature
is free on Bytebase's own key, so no customer budget is at risk and abuse is our own spend-management
problem. Self-hosted, the admin supplies the org's key when enabling it, so org-wide use by members
is the intended model. The missing per-user rate limit and audit are accepted with that disposition.

Two patches from earlier in the session (`SearchProjects` proto comment, `SearchProjects` pagination)
remain parked and still apply — neither landed independently, and both targets are still present at
HEAD. The third, `DiffSchema` ACL, is superseded by the same-project fix above.

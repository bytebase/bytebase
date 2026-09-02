# Bytebase v1 API — audit

What's wrong with the v1 API, in plain terms.

Audited against `93671b00b0` (2026-08-04); every finding below re-verified against `4a80dac023`
(2026-08-31), whose line numbers these are. Findings are code-traced, not runtime-proven, except
where noted. Fixed findings are removed and listed at the end.

## The short version

| | Problem | Severity |
|---|---|---|
| T12 | Anonymous callers can read your LDAP config | MED |
| T18c ii–iii | Broken filter, unpaginated list, existence-leaking error codes | MED |
| T18c-iv | `BatchDeleteProjects` checks its permission against the workspace, not the named projects — latent, no predefined role reaches it | LOW |

---

## Doing more than you're allowed

### The one-resource ACL rule, and what it still costs

`getResourceFromSingleRequest` authorizes exactly one resource name per request, picked by field-name
convention: `parent`, else `name`, else `resource`, else `project`, else — for `Create`/`Update`/
`Remove`/`Test` — the nested `<snake_case(resource)>.name`. Nothing else in the body is visible to it.
`acl.go:743-803`

A sweep of the v1 RPCs — 205 at the 2026-08-04 baseline, 217 now — found 24 whose body names a
resource the interceptor never authorizes. Twenty-two are closed downstream: `CreatePlan`/
`UpdatePlan`, the three `Release` RPCs, `BatchCreateRevisions`, `CreateAccessGrant`, the
`SavedQuery` writes, `CreateIssue`, `UpdateIssueComment`, `TestWebhook` and `GetQueryHistory` each
compare the second resource's project to the authorized one; `BatchRunTasks`, `BatchSkipTasks` and
`CreateRollout` re-derive it from the parent instead of trusting the name; `UpdateDatabase`/
`BatchUpdateDatabases` are special-cased in the interceptor itself. `DiffSchema` was the one live
gap and is now closed the same way.

The convention can also miss the *only* resource: `UpdateDatabaseCatalog` names its target in
`catalog`, not the `database_catalog` the method suffix derives, so it resolved nothing and every
call was authorized against the workspace. Both are fixed and listed at the end. Neither fix
generalizes — the next request field that departs from the convention arrives unchecked, silently,
and only a sweep like this one finds it.

### T18c-iv · `BatchDeleteProjects` authorizes at workspace scope — LOW

Found re-sweeping the 217 RPCs at HEAD. `names` carries a `resource_reference`, but nothing resolves
it: the RPC is not a `BatchGet` (so the `names` branch does not apply), has no `parent` and no
`requests`, and no `parent`/`name`/`resource`/`project` field — so `getResourceFromSingleRequest`
returns `""` and `bb.projects.delete` is checked against the workspace. `DeleteProject`, same
permission, resolves `name` to project scope.

Fail-closed, and latent: no predefined role reaches the mismatch. `bb.projects.delete` is held only
by Workspace Admin and Workspace DBA, both of which satisfy the workspace-scoped check anyway, and
Project Owner holds `bb.projects.update` but not `.delete` — so `DeleteProject` denies it too. It
takes a custom role containing `bb.projects.delete` bound at project scope, which nothing prevents:
`CreateRole` validates only that each permission exists. Such a principal deletes one project at a
time and is denied the batch form.

Worth recording anyway, because it is the same silent convention miss as `UpdateDatabaseCatalog`
pointing the other way — the scope is decided by which field names the request happens to use, not
by anything stated — and nothing in the proto says the batch form needs workspace-level rights.
`acl.go:708-739`, `project_service.proto:394-400`, `predefined_roles.go:97`, `:249`, `:406`

---

## Lists that return the wrong rows

### T18c-ii · Two smaller list bugs — MED

- **The query-history `instance ==` filter always returns nothing.** It compares an instance name
  against a longer stored value using `LIKE` with no wildcard. `SearchQueryHistories` documents the
  broken form. Its sibling `database ==` is plain equality and works. Two adjacent uses of the same
  operator are also unescaped: `statement ==` passes the raw value to `LIKE`, so `%` and `_`
  over-match on the `==` path that T17's `.contains()` fix did not cover; and `FindQueryHistoryMessage.Instance`
  repeats the missing wildcard, harmlessly — no caller sets it.
  `store/query_history.go:257`, `:262`, `:152-153`
- **`ListTaskRuns` has no pagination and no limit** — and its documented wildcard parent pulls an
  entire rollout in one call, on the table that grows fastest. Not just an unbounded store query:
  `ListTaskRunsRequest` has no `page_size` or `page_token` field to add a limit to.
  `store/task_run.go:65-191`, `rollout_service.proto:317-325`

---

## What anonymous callers can learn

### T12 · The unauthenticated surface — MED

12 RPCs allow calls with no credentials. Two are still open:

- **LDAP config disclosure.** `ListIdentityProviders` hands any caller the SSO config for the
  workspace they name: LDAP host, port, bind DN, base DN, user filter, plus OAuth/OIDC endpoints
  and client IDs. Passwords and secrets are redacted. No guessing is needed self-hosted —
  `GetWorkspace("workspaces/-")` is anonymous too and returns the singleton workspace ID, so the
  disclosure is a two-call chain; and self-hosted is where LDAP is configured. Omitting `parent`
  needs no chain at all, but reaches only the `workspace IS NULL` global IdPs — the SaaS shared
  login providers, whose client IDs are public by design.
  `idp_service.go:55-85`, `:461-541`, `workspace_service.go:63-74`

  The endpoint serves three journeys with three different needs — the anonymous login page (which
  needs the authorization-request fields and no LDAP config at all), the SSO admin console (which
  needs everything), and General settings (which needs `count > 0` to guard the
  disallow-password-signin toggle, and which every Workspace Member can reach). Redacting by caller
  closes it, but the durable fix is to give the login page its own message on `AuthenticationInfo`,
  following [#21184](https://github.com/bytebase/bytebase/pull/21184), so a new admin-side config
  field cannot become public by default. Tracked in
  [BYT-10156](https://linear.app/bytebase/issue/BYT-10156/listidentityproviders-leaks-ldap-config-to-anonymous-callers-t12).
- **Workspace-existence oracle.** `GetAuthenticationRestriction` (which replaced the actuator leak,
  now fixed) is anonymous by design and doesn't need membership. Naming a real workspace returns
  200, a fake one returns `InvalidArgument`. Left open on purpose: workspace IDs are
  `RandomString(16)` with no rename path, so this confirms an ID already held rather than
  enumerating one, and the pre-login page needs the workspace's real settings — so returning
  defaults for an unknown workspace trades a genuine login-page error message for an oracle that
  still distinguishes any workspace with non-default auth settings. `auth_service.go:107-157`,
  `auth_service.go:449` (ID generation)

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
   `AUTH_METHOD_UNSPECIFIED` at startup; tie a `permission` on a CUSTOM RPC to the handler that
   enforces it, or make it a build error. The interceptor skips CUSTOM, so the annotation is held
   true only by hand: 15 of the 52 declare one, 14 check that same permission in their handler, and
   `SearchMyAccessGrants` declares `bb.accessGrants.get` and checks nothing — it scopes to
   `Creator == caller` instead. So the standing defect is drift, not absence, plus that one live
   case. `TestAllowMissingCreatePermission`, added with T11, is the descriptor-walking shape to
   generalize. None of Tier 5 was visible to `buf lint`'s BASIC profile, which is why it accumulated.

---

## Already resolved

| Finding | Resolution |
|---|---|
| T1, T3 | Fixed earlier in the session |
| T2 (INPUT_ONLY) | Read-path fix |
| T2 `DiffSchema` | `DiffSchema` rejects a changelog target owned by another project before any read, matching how every other second-resource handler closes it. Not the general interceptor change: the resource the caller names must simply belong to the project they were authorized on, which is also the only thing a cross-project diff could have meant. A missing target and a foreign one return the same error, so neither confirms what lives in the other project. Accepted with it: the class stays latent — a future second-resource field still arrives unchecked |
| T19 `UpdateDatabaseCatalog` | The interceptor now reads `catalog.name`, so the permission is checked against the named database's project instead of the workspace — the route resolver already drops the trailing `/catalog`. Strictly additive: workspace holders are unaffected and Project Owner, whose role carries `bb.databaseCatalogs.update`, is no longer denied. Separate and still open: `bb.databaseCatalogs.create` does not exist, so the `allow_missing` secondary check denies every caller. Dead rather than harmful — no UI sets the flag and the RPC is MCP-`EXCLUDED` |
| T11 | `UpdateGroup(allow_missing=true)` now checks `bb.groups.create` itself, the shape `UpdatePlan` and `UpdateUser` already had. The interceptor never covered it: `doIAMPermissionCheck` returns true for every non-IAM auth method, so the `allow_missing` secondary block evaluated to nothing on a CUSTOM RPC, and the create path calls `CreateGroup` in-process, which bypasses the interceptor anyway — while the comment above it claimed both permissions were verified. The exposure was narrower and differently shaped than first written: Workspace Member does hold `bb.groups.create`, but Workspace DBA and every project role do not, so a DBA-only principal could create a group and name itself OWNER, which `checkPermission` then honors for that group. It stops there — a group confers nothing until someone with IAM-policy rights binds a role to it. The interceptor's block is now gated on `AuthMethod == IAM` so it no longer reads as protection where it verifies nothing (behavior-preserving: it already returned true there). Found tracing the same mechanism, and fixed with it: the `.update` → `.create` string rewrite is unvalidated, and `UpdateDatabase` derived `bb.databases.create`, which does not exist — `CheckPermission` never matches an unknown string, so `allow_missing=true` denied every caller including Workspace Admin, on a flag `database_service.go` never read and the proto documented as working. The field is removed and the proto now says never to add it back, since a database is not independently creatable. `TestAllowMissingCreatePermission` walks the descriptors and makes all four silent failure modes a build failure — an RPC that gains the flag with nothing authorizing its create path, a derived permission that does not exist, a declared permission the rewrite leaves unchanged (`bb.settings.set` is one, and vacuous if it ever went IAM), and a nested `allow_missing` sub-request the interceptor's hand-written type switch does not name, which it proves by building the request and asserting `hasAllowMissingEnabled` reads it. Still open, and the other half of the original finding: 15 of the 52 CUSTOM RPCs declare a `permission` annotation the interceptor does not enforce. Fourteen check that same permission by hand, so what is missing is anything binding the two — a handler can drift from its annotation and nothing fails. `SearchMyAccessGrants` has already drifted: it declares `bb.accessGrants.get` and checks nothing, scoping to `Creator == caller` instead |
| T5, T6 | [#21143](https://github.com/bytebase/bytebase/pull/21143) — `sheet_blob_ref` gives sheets per-project ownership; `BatchCreateRevisions` rejects foreign-project provenance without echoing the hash |
| T7 | [#21102](https://github.com/bytebase/bytebase/pull/21102) — every `CheckRelease` target validated against the parent project before any schema read (its error codes still leak existence — T18c-iii) |
| T8 `CreateWorksheet` | [#21169](https://github.com/bytebase/bytebase/pull/21169) — `CreateSavedQuery` is now IAM-enforced; Workspace Member holds no saved-query permission, and new queries start creator-private |
| T12 `GetActuatorInfo` | [#21184](https://github.com/bytebase/bytebase/pull/21184) — anonymous access and the `name` field removed; workspace now comes from the token (the narrower surface that replaced it is T12) |
| T12 email enumeration, mail relay | Two of the four anonymous surfaces; LDAP disclosure and the existence oracle stay open and are what T12 is now. **`Signup`** checks the restriction before it checks existence, so a workspace that refuses every signup — every SaaS workspace, since the override always sets `DisallowSignup` — no longer answers `AlreadyExists` for registered addresses and `PermissionDenied` for the rest. The duplicate is still reported where signup is allowed; only the order moved, and denied duplicates now reach the deferred `SetAuditWorkspaceID` they used to return ahead of. **`SendEmailLoginCode`** caps sign-in-code mail at 1000/hour for the whole deployment, whatever workspace the caller names: SaaS copies `EMAIL_CONFIG` into every workspace it creates, so naming one does not change the sender, and keying on the workspace would hand each tenant its own budget against a shared reputation — with a caller-supplied key besides. The 60s per-recipient cooldown already bounded one address; only a sender-wide cap bounds a campaign, which writes each address exactly once. Claimed before the code row is written, so a refusal leaves a pending code intact, and before the cooldown is consulted, so the answer never depends on whether that recipient has a recent code. Reuses `login_attempt` under a new `EMAIL_CODE_SEND` kind — no migration, `kind` is text — but not its semantics: a lockout resets only after a quiet window, which for a volume budget would turn "1000 per hour" into "1000 ever, until an hour of silence", so this kind reads `last_attempt_at` as the window's start. Accepted: spending the cap delays sign-in codes deployment-wide until the window passes, and one address can spend it as cheaply as many since the cost is the requests — bounded and recoverable, unlike the deliverability damage it prevents. Nothing here varies by recipient: a member exemption, a redeemability gate and a cooldown pre-check were each implemented and removed, because on an endpoint answering anonymous callers any recipient-dependent branch is an oracle |
| T13 `SearchWorksheets` | [#21160](https://github.com/bytebase/bytebase/pull/21160) + [#21178](https://github.com/bytebase/bytebase/pull/21178) — visibility predicate pushed into SQL before `LIMIT` |
| T13 issue filters | `approval_status` and `current_approver` are matched in SQL before `LIMIT`, so a filtered page can no longer come back empty under a live page token. The status is a `CASE` over `payload->'approval'` mirroring `computeApprovalStatus`; `current_approver` cannot be resolved in SQL alone (binding conditions and group expansion are evaluated in Go), so the caller passes down the `(project, role)` pairs the named user holds, the same shape as the saved-query `AccessMembers` predicate. Accepted with it: the predicates are not indexable, so a filtered list scans until it has a full page of matches — measured on 50,000 issues across 40 projects, that is ~9 ms against ~14 ms for the same unfiltered query, and ~3.5 ms in the zero-match case, because PostgreSQL drives the join from the small `(project, role)` set, so no expression index is warranted yet. Deliberately unchanged: the filter still ignores rejection and self-approval, so a rejected issue names the holder of the *next* role as its current approver — masked in the default view, and worth its own fix |
| T18 worksheet write/delete | [#21169](https://github.com/bytebase/bytebase/pull/21169) + [#21181](https://github.com/bytebase/bytebase/pull/21181) — per-verb permissions; SQL Editor Read User can no longer rewrite or delete |
| T17 | Every CEL string literal the frontend emits now comes from `celString`/`celStringList`/`celMapField` in a dependency-free `frontend/src/utils/v1/celLiteral.ts` — the helpers own the quotes, so the escape cannot be forgotten the way it was at 11 of 17 sites. Worst was the access-grant page, whose search box takes SQL: a quoted identifier made `InvalidArgument` routine. All 17 sites converted, the enum and resource-name ones included, because an absolute rule is the only one a machine can check — `src/architecture/cel-filter-literals.test.ts` scans `src/**` for the three raw shapes and fails CI on a new one, in the style of the existing Vue-boundary guards. The helpers are a leaf module because putting them beside the CEL RPC clients dragged `@/api` into `modules/cel/logic/stringify.ts` and broke that subtree's tests. Two adjacent defects the sweep found, neither of which escaping addresses, fixed with it. **A label key with a dash never reached the backend at all**: keys are validated `^[a-z][a-z0-9_-]{0,62}$`, and CEL parses `labels.cost-center` as subtraction, so the store answered `unsupport variable \"\"` — the frontend now emits `labels[\"key\"]` index syntax, `getVariableAndValueFromExpr` canonicalizes both forms, and the key is **bound** rather than interpolated into the JSONB path, since index syntax admits a quote the dotted form could not. **`%` and `_` reached `LIKE` as wildcards**, so a statement search containing either silently over-matched — `escapeLikePattern`, already in-tree for saved queries, plus `ESCAPE '\\'` now covers all 13 `contains` translations. The three copies of the label-filter closure collapsed into one `buildLabelFilterSQL`, which also drops an unchecked `raw.(string)` that panicked on `labels.k in [1]`. Verified against PostgreSQL 16: `->>$1::text` matches, an injected key is inert, and `\_` stops matching what `_` did. Accepted with it: the guard is line-based and skips test files, which should spell out expected filter strings literally; and `query_history`'s two remaining raw-`LIKE` filters are left alone — `instance ==` missing its wildcard and `statement ==` missing the escape, both T18c-ii. Its `database ==` is plain equality and was never affected |
| T18c-i | `BatchDeleteProjects` purges every named project in one transaction, so a failure partway rolls the whole batch back instead of leaving the projects ahead of it irreversibly gone with an error naming only the one that failed. `Store.DeleteProject` is now a one-name call into `DeleteProjects`, whose statements carry `project = ANY(?)` — one statement per table for the whole batch, in the same child-to-parent table order the loop already used, so the chains in [`backend/store/AGENTS.md`](../../backend/store/AGENTS.md#transaction-row-lock-ordering) are walked bottom-up for every project at once. Names are sorted and deduped, so two batches naming the same projects in different orders take the project row locks in the same order: the second finds nothing left to purge and fails rather than deadlocking, which `TestDeleteProjectsConcurrentBatchesLockOrder` pins with a barrier against a real PostgreSQL, alongside rollback tests for a name that is not archived and one that does not exist. Accepted with it: sample-instance cleanup still runs per project *before* the transaction, because it removes an embedded PostgreSQL data directory or an external database and role that no rollback restores — it writes no project row, so a failure there still purges nothing, and `sample_instance_setup` holds one live row per workspace, so at most one project in a batch has work to do. Unchanged: the default-project and archived-before-purge guards are still checked for every name before any purge begins, and the soft-delete path was already a single `UPDATE`. `TestBatchDeleteProjectsPurgeIsAtomic` drives the refusal through the gRPC API and `TestCollisionBatchDeleteProjectsCascade` is the batch counterpart of the existing purge collision test. Untouched, and still open as T18c-iv: the batch form is authorized at workspace scope |
| T18a | [#21265](https://github.com/bytebase/bytebase/pull/21265) — `BatchCancelTaskRuns` confines its lookup to the plan and environment just authorized; the stage in a task run name is caller-supplied, so cancel rights in one environment reached a RUNNING row in another. An out-of-scope ID is rejected, not dropped — the cancel is driven by the requested IDs |
| T9 | [#21189](https://github.com/bytebase/bytebase/pull/21189) (self-hosted audit rows) + [#21234](https://github.com/bytebase/bytebase/pull/21234) — one `login_attempt` table bounds password, email-code, and MFA guessing per identity on both deployments, replacing the audit-log counter and the per-code attempt column (which bypassed the resend cooldown). Accepted with it: no per-tenant failed-login record on Cloud, and lockout-as-denial-of-service — see [`login-attempt-lockout.md`](login-attempt-lockout.md) |
| T15 | `update_mask` now decides what each of the four Update methods writes. **`UpdateDataSource`** dispatches the IAM credential on the mask path instead of the request's `authentication_type`, which was unset on any request that masked only the credential — the AIP-134 shape — so rotating a leaked key returned 200 and wrote nothing. The same branch, given a body type that disagreed with the mask, wrote the *other* credential and moved `authentication_type` with it though the mask named neither; the effective type is now resolved from the mask before the loop and a mismatched path is `InvalidArgument`. **`UpdateSetting(APP_IM)`** splices only the masked providers into the stored value rather than assigning the request wholesale, so saving Slack no longer erases the Feishu, WeCom, Lark, DingTalk and Teams secrets; the mask is now required, and a masked provider the payload omits is removed, which is how the console deletes one (it no longer merges the list client-side). **`UpdateIssue`** rejects paths it does not implement instead of returning 200 and dropping them — `status` belongs to `BatchUpdateIssuesStatus` and approvals to `ApproveIssue`/`RejectIssue`, so the supported set was right and only the silence was wrong. **`UpdateDatabaseCatalog`** keeps full-replace semantics and now says so in the proto; its `allow_missing` is removed and the number reserved, because a catalog is not independently creatable — it exists once its database is synced — so the flag had nothing to create, and the ACL's create-permission check on it denied every caller. Both merges were extracted into pure functions (`applyDataSourceUpdateMask`, `mergeAppIMSetting`), so `instance_service_test.go` and `setting_service_test.go` pin the behavior in milliseconds without a server or a metadata database. Accepted with it: the APP_IM mask paths still name no real proto field (`AppIMSetting` holds one repeated `settings`, and `value.app_im_setting_value.teams` is not a field path at all), so they are matched as a fixed vocabulary — making them true paths means restructuring the message one-field-per-provider, which needs a JSONB migration. Also left open, and the reason `UpdateDatabaseCatalog`'s hazard is documented rather than removed: a mask cannot express "this one table" while `schemas` is repeated, and read-modify-write on the catalog still loses one side of a concurrent edit, which needs an etag (AIP-154); the IAM-policy half of it is fixed, see T18b below |
| T16 | The four `BatchGet` RPCs behaved four different ways when a name resolved to nothing, none of it documented, and the responses are bare repeated lists — so a client matching `names[i]` to `results[i]` silently mis-attributed data. They now answer one contract, stated in each proto and matching AIP-231: **one resource per requested name, in request order, and the first name that does not resolve fails the whole call.** `BatchGetX` is exactly `GetX` per name, so there is no partial response to reason about. What that fixed: `BatchGetDatabases` dropped a database that was gone or invisible and answered 200 with a short list; `BatchGetGroups` swallowed *every* error into a silent omission, so a store failure was indistinguishable from "no such group"; `BatchGetUsers` emitted the store's `created_at` order rather than request order, which is the mis-attribution hazard in its purest form; `BatchGetProjects` was already atomic but silently dropped a deleted project that `GetProject` returns. Partial success — reporting the names that did not resolve rather than failing — was tried and rejected as a second contract every client would have to handle; AIP-231 says to use a List method when you want one. Accepted with that: a caller resolving names held in stored references — IAM bindings, saved-query grants, SQL editor tab state — now loses the whole batch to one stale name, and on Cloud `BatchGetUsers` scopes to workspace IAM, so an issue created by a since-removed member fails the batch behind that page. Each of the four frontend stores therefore falls back to per-name fetches when the batch fails, which is the pattern `batchFetchProjects` already used for exactly this. `TestBatchGetUsers` in `backend/tests/user_test.go` pins request order and the all-or-nothing failure against a real PostgreSQL — on `BatchGetUsers` only, the one whose store order actually diverged; the other three were driven against a running server but have no regression test |
| T14 | [#21267](https://github.com/bytebase/bytebase/pull/21267) — filed as issue paging; the sweep it prompted found the same defect across the class, so it was fixed as a class. Counting endpoints, **17 of the 22** offset-paginated v1 list RPCs were affected; counting the store functions behind them, **14 of 17**. The three already sorting on a total order were `ListQueryHistories` and `ListSavedQueries` (both from one commit, [#21203](https://github.com/bytebase/bytebase/pull/21203), that was never swept across the other lists) and `ListPlans`, which is total only because a mandatory `WHERE plan.project = ?` pins its scope column. Every store list now names tiebreak columns that are unique under its own scope — written into the SQL, and appended after the caller's keys at the six affected lists that accept an `order_by`. The three that were already total are left untouched. Nothing enforces this statically: an earlier revision routed all seventeen through a `buildStableOrderBy` helper so an AST test could require it, but review judged that ceremony and both were removed, so a new paginated list added without a tiebreak will not fail CI. Enforcement is step 4 of `docs/pre-pr-checklist.md` plus review, with `TestPaginationStabilityAcrossProjects` and `TestIssueCommentBatchKeepsInsertionOrder` covering the behavior against a real PostgreSQL. The rules, and the five traps that produced the class — `id` alone in a `(project, id)` table, `created_at` (the *transaction* timestamp, identical across a batch insert), a nullable column, a partial unique index, and an `order_by` that replaces the default ordering rather than adding to it — are in [`backend/store/AGENTS.md`](../../backend/store/AGENTS.md#pagination-ordering). One adjacent bug fixed with it: `ListDatabases`, `ListInstances` and `ListProjects` each threw away the whole clause, tiebreak included, the moment a caller passed `order_by` — which is what broke their paging. Accepted with them: offset paging still drifts under concurrent inserts and deletes (that needs keyset pagination and a page-token change). `ListIssueComment` needed a second fix: its `resource_id` tiebreak is a random UUID, which would have scrambled the activity feed of a multi-field `UpdateIssue`, so `CreateIssueComments` now offsets each row of a batch by its ordinal to keep `created_at` unique and in insertion order. Deliberately out of scope, and worth its own change: `issue.id` is a per-project counter, so ordering a cross-project list by it is not recency — seeded with a three-year-old project and one created today, the newest issue in the new project ranked 4962nd on the default list. Making `create_time` the cross-project default is a product decision. Also left alone: issue search still discards an explicit `order_by` outright when the caller supplied query text |
| T10 | [#21252](https://github.com/bytebase/bytebase/pull/21252) + [#21258](https://github.com/bytebase/bytebase/pull/21258) — password change and MFA lifecycle move off `UpdateUser` onto their own methods, each requiring a `CredentialProof`: current password, live OTP, recovery code, or a Cloud-only emailed re-auth code. Every proof claims a T9 login-attempt slot, so no proof channel is an unbounded guessing oracle, and a factor-touching method refuses the password while a live factor exists. Accepted with it: **the stolen access token still answers until it expires (≤1h)** — credential generation and the fenced transaction were cut or deferred, password change revokes only the account's web refresh tokens (best-effort, OAuth grants untouched), and an MFA change revokes nothing. What this closes is the credential being *spent* on its own replacement, not the session. Design, and the shipped-vs-designed delta: [`reauthenticate-credential-changes.md`](reauthenticate-credential-changes.md) |
| T18b | The etag is now read wherever the caller presents it: `policy.etag` — the only place `GetIamPolicy` returns one, and so the only place a read-modify-write round-trips it — as well as the request field, with two that disagree rejected. Both handlers write through `store.SetIamPolicy`, which compares under `SELECT ... FOR UPDATE` inside the write transaction. Comparing against the handler's own earlier read would not have been enough: that is a check-then-act, and two requests reading the same policy both pass it before either writes. (An earlier revision justified this by replica cache skew, which does not exist: `store.New(ctx, pgURL, !profile.HA)` disables read caching in the only mode that runs more than one process.) The same lock closed two merges that present no etag at all and so could never be compared — `PatchWorkspaceIamPolicy` (signup, SSO provisioning, workspace-leave) and the role-grant issue approval in `backend/utils` — each of which read the policy, merged one member in, and wrote the whole thing back, so either could resurrect a member an admin had just revoked; both now merge inside the locked transaction through `store.PatchIamPolicy`. Fixed with it: workspace `SetIamPolicy` wrote to `request.Resource` while every read and check used the workspace from the token, so a request naming anything else matched no row and returned 200 having written nothing — the write now targets the workspace the reads did, which is what `GetIamPolicy` already does by ignoring the field. The audit delta and the invite emails are now derived from the policy the write actually replaced rather than the pre-validation snapshot, so they describe what happened even for an etag-less caller, and the workspace seat guard counts against that same locked policy through a `ValidateReplaced` hook rather than against a read the write does not hold. Accepted with it: an etag-less caller still gets last-write-wins, which is what AIP-154 optional-etag semantics mean and what existing API clients rely on; and the etag is still `updated_at` in milliseconds, so a write landing in the same millisecond as the one before it leaves the etag unmoved and a stale write racing that pair is accepted — too narrow to justify the wire change to a content hash, unlike the saved-query policy, which has no timestamp to derive one from. Also left as is: `validateIAMPolicy` grandfathers bindings found in the policy the handler read before the write, so an etag-less caller racing a concurrent removal re-adds that binding without the `allUsers`, role-existence, CEL and maximum-expiration checks — pre-existing, and closing it means splitting the function into a fetch half and a pure half that runs under the lock |

**`AIService.Chat` — accepted, not fixed** (2026-08-18). `Chat` requires an authenticated workspace
principal and touches no database or sheet, so the only exposure is LLM spend. On Cloud the feature
is free on Bytebase's own key, so no customer budget is at risk and abuse is our own spend-management
problem. Self-hosted, the admin supplies the org's key when enabling it, so org-wide use by members
is the intended model. The missing per-user rate limit and audit are accepted with that disposition.

Two patches from earlier in the session (`SearchProjects` proto comment, `SearchProjects` pagination)
are parked and unlanded, but do not assume they still apply: `SearchProjects` is fully paginated at
HEAD — `page_size`/`page_token` in the proto, `parseLimitAndOffset` and `nextPageToken` in the
handler — and was already so at the 2026-08-04 baseline, so whatever the pagination patch targeted,
it is not that. Re-read both against HEAD before landing either. The third, `DiffSchema` ACL, is
superseded by the same-project fix above.

# Bytebase v1 API — audit

What's wrong with the v1 API, in plain terms.

Audited against `93671b00b0` (2026-08-04); every finding below re-verified against `fd4ef828e6`
(2026-08-18), with HEAD line numbers. Findings are code-traced, not runtime-proven, except where
noted. Nothing here is patched yet. Fixed findings are removed and listed at the end.

## The short version

| | Problem | Severity |
|---|---|---|
| T9 | On Cloud, password guessing is unlimited — the lockout never fires | HIGH |
| T10 | A stolen session can change your password and switch off your 2FA | HIGH |
| T2 | Schema diff reads a second database without checking permission on it | HIGH |
| T15 | Rotating a leaked cloud credential can silently do nothing | HIGH |
| T13 | "Waiting for my approval" can show an empty page with a live Load More | HIGH |
| T14 | Paging through issues across projects skips some and repeats others | HIGH |
| T18a | Test-environment rights can cancel a running production migration | MED |
| T12 | Anonymous callers can enumerate emails, relay mail, and read your LDAP config | MED |
| T16 | Four `BatchGet` RPCs behave four different ways when an item is missing | MED |
| T18b | Concurrent permission edits silently lose one side, including a revoke | MED |
| T11 | A create-permission check is skipped on `allow_missing` updates | MED |
| T17 | Typing `"` in a search box breaks search | MED |
| T18c | Three smaller ones: broken filter, unpaginated list, half-finished delete | MED |

---

## Getting in, and staying in

### T9 · Password guessing is unlimited on Cloud — HIGH

The 10-strikes lockout works by counting failed-login rows in the audit log. On Cloud the sign-in
page doesn't send a workspace ID, and with no workspace the audit row is never written — so the
counter reads zero forever and neither the password (10/10min) nor the MFA (5/5min) limit ever
fires. There is no IP or global rate limiter anywhere in the stack, so this counter is the only
brake on brute force.

Self-hosted is fixed and tested ([#21189](https://github.com/bytebase/bytebase/pull/21189)); this
is the SaaS remainder. **Fix:** in SaaS, either reject a login that names no workspace, or resolve
it from the request host *before* checking the password. `auth_service.go:180-189`

### T10 · A borrowed session becomes permanent account takeover — HIGH

Changing your own password doesn't require the current password, and switching off your own 2FA
doesn't require a code. So anyone who gets a session — stolen laptop, shared browser, leaked token —
converts that temporary access into permanent ownership of the account. Refresh tokens are revoked
on password change, but the attacker's own access token keeps working.

The frontend work in [#21193](https://github.com/bytebase/bytebase/pull/21193) split these screens
apart, but the backend is unchanged: there is still no current-password or recent-auth field on the
API. **Fix:** require the current password (or a fresh OTP) on both paths.
`user_service.go:371-378` (password), `:395-411` (MFA)

---

## Doing more than you're allowed

### T2 · The permission check only looks at one resource per request — HIGH

The ACL interceptor authorizes exactly one resource name per request: `parent`, else `name`, else
`resource`, else `project`. Any *second* resource named in the body is invisible to it.

One live case: `DiffSchema` takes both a database and a changelog, and the changelog can belong to a
different project. Only the first is checked, so a caller can read schema out of a project they have
no access to. Every other known case is closed inside the handler; this is the last one, and the
parked patch `04` fixes it.

The real fix is the general one — teach the interceptor to collect *every* annotated resource field,
so the next second-resource field doesn't reopen the hole. `acl.go:708-726`,
`database_service.go:1047-1105`

### T18a · Test-environment rights can cancel a production migration — MED

`BatchCancelTaskRuns` checks your permission against the stage *you claim in the URL*, then cancels
task runs by ID without ever confirming those IDs belong to that stage. So permission to cancel in
test is enough to kill an in-flight production migration in the same project. Sibling handlers get
this right by reading the environment off the actual rows. `rollout_service.go:1034-1143`

### T11 · A create check is skipped on `allow_missing` updates — MED

`UpdateGroup(allow_missing=true)` creates a group when none exists, but the create-permission check
is short-circuited for CUSTOM-auth RPCs, so nothing is verified. Harmless today only because every
member can create groups anyway — but the code comment right above it claims the check happens,
which is how this survives review.

Same root cause, wider: 15 of the 45 CUSTOM RPCs declare a `permission` annotation that is never
enforced. It's decorative, and it reads as protection. `acl.go:252-254`, `group_service.go:172-177`

---

## The API says OK and does nothing

### T15 · `update_mask` is ignored in four places — HIGH

The worst one: **rotating a leaked cloud credential can silently no-op.** `UpdateDataSource` decides
what to write based on the `authentication_type` in the request rather than the mask or the stored
value, so a request carrying only a new GCP credential falls through an empty branch and writes
nothing. You get HTTP 200 and the compromised key stays live. `instance_service.go:1346-1388`

The other three lose data rather than fail to save it:

- **`UpdateSetting` replaces whole settings.** Updating your Slack config wipes stored Feishu, WeCom,
  Lark, DingTalk and Teams secrets — and it *validates* the mask path first, then ignores it. For
  environments the replace cascades into every instance and database. `setting_service.go:508-512`
- **`UpdateDatabaseCatalog` never reads a mask at all** and replaces the whole config column, wiping
  other schemas' classification and masking. Its proto comment refers to an `update_mask` field that
  doesn't exist. `database_catalog_service.go:96-139`
- **`UpdateIssue` implements four fields** and silently drops the rest, so `update_mask=["status"]`
  returns 200 and does nothing. `issue_service.go:883-910`

Underneath: some Update RPCs reject unknown mask paths, others ignore them, in four different error
wordings. There's no shared validator.

### T18b · Concurrent permission edits silently lose one side — MED

`IamPolicy` has an etag for exactly this, and the docs promise it triggers ABORTED — but the
handlers only check the etag on the *request wrapper*, never the one inside the policy. A normal
read-modify-write round-trips the policy etag and has it silently discarded, so two admins editing
permissions at once means one edit vanishes. If the lost edit was a revoke, access stays granted.

The saved-query path already implements this correctly, so the pattern is in-tree.
`project_service.go:586`, `workspace_service.go:437`

### T18c-i · Deleting several projects can half-finish — MED

`BatchDeleteProjects` purges in a loop with no transaction. A failure partway leaves some projects
irreversibly purged, and the error names only the one that failed — not the ones already gone.
`project_service.go:485-489`

---

## Lists that return the wrong rows

### T13 · Issue filters run after the page is cut — HIGH

`approval_status` and `current_approver` never reach SQL. The database returns a full page, the page
token is minted, and *then* non-matching issues are dropped in Go. So filtering by "waiting for my
approval" in a busy workspace shows an empty list with a live "Load more" button underneath.

This is the default My Issues view, and there's no empty-page workaround on the frontend.
`issue_service_converter.go:29-41`, `issue_service.go:139-148`

(Same shape as the `SearchProjects` bug that parked patch `03` fixes — also still unmerged.)

### T14 · Paging issues across projects skips and repeats rows — HIGH

Issue IDs restart from 1 in each project, but the cross-project list sorts by ID alone. Issues from
different projects constantly share an ID, and tied rows can land in a different order on each query
— so paging forward drops some issues entirely and shows others twice. My Issues always queries
across all projects, so this is the common path, not an edge case.

**Fix:** sort by `(project, id)`, which is the primary key. `store/issue.go:323`

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
  `store/query_history.go:255-256`
- **`ListTaskRuns` has no pagination and no limit** — and its documented wildcard parent pulls an
  entire rollout in one call, on the table that grows fastest. `store/task_run.go:65-131`

### T17 · Typing a quote breaks search — MED

The frontend builds CEL filter strings by interpolating raw user input. There's an escaping helper
that exists precisely for this, used at 6 sites; 11 others still interpolate raw. Type a `"` in
those search boxes and search fails with `InvalidArgument`. It also allows filter-semantics
injection — not a privilege boundary, since the filter only narrows an already-authorized list.

Worst site: `accessGrant.ts:35` interpolates SQL statement text raw, while lines 38-40 right next to
it do it safely.

---

## What anonymous callers can learn

### T12 · The unauthenticated surface — MED

12 RPCs allow calls with no credentials. Four are worth attention:

- **Email enumeration.** `Signup` answers "already registered" *before* it checks whether signup is
  even allowed, so anyone can test whether an email has an account — on a workspace with signup
  disabled. `auth_service.go:328-334`
- **Open mail relay.** `SendEmailLoginCode` will send to any address over the customer's SMTP.
  [#21177](https://github.com/bytebase/bytebase/pull/21177) added a domain check, but it only
  applies if the domain-restriction license feature is on, enforcement is enabled, *and* a domain
  list is set — no default deployment qualifies. The 60s cooldown is per recipient, so volume scales
  with the address list. `user_service.go:753-785`
- **LDAP config disclosure.** `ListIdentityProviders` hands any caller the SSO config for any
  workspace ID they guess: LDAP host, port, bind DN, base DN, user filter, plus OAuth/OIDC endpoints
  and client IDs. Passwords and secrets are redacted. `idp_service.go:55-85`, `:526-535`
- **Workspace-existence oracle.** `GetAuthenticationRestriction` (which replaced the actuator leak,
  now fixed) is anonymous by design and doesn't need membership. Naming a real workspace returns
  200, a fake one returns `InvalidArgument`. The pre-login page genuinely needs most of these
  fields; the oracle is the part worth closing. `auth_service.go:140-179`

### T18c-iii · `CheckRelease` error codes leak existence — MED · ✅ read myself

Point it at a database in someone else's project and you get `InvalidArgument`; point it at one that
doesn't exist and you get `NotFound`. So permission to check releases on any single project lets you
map which databases exist workspace-wide. [#21102](https://github.com/bytebase/bytebase/pull/21102)
added a third distinguishable error that also leaks instance ownership.

**Fix:** return `NotFound` for both, matching the convention `BatchCreateRevisions` already adopted,
with a regression test. `release_service_check.go:92-111`

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

1. **Close the SaaS half of T9.** It's the only brute-force control that exists, and it currently
   doesn't run on the hosted product.
2. **Fix the ACL class, not the instance.** Teach the interceptor to collect every annotated
   resource field, then annotate `DiffSchema`. Otherwise the next second-resource field reopens it.
3. **Fail closed on auth annotations.** Reject `AUTH_METHOD_UNSPECIFIED` at startup, and make a
   `permission` on a CUSTOM RPC either enforced or a build error — 15 are decorative today.
4. **Put api-linter in CI** at a freshly measured baseline. None of this was visible to `buf lint`'s
   BASIC profile, which is why it accumulated.

---

## Already resolved

| Finding | Resolution |
|---|---|
| T1, T3 | Fixed earlier in the session |
| T2 (INPUT_ONLY) | Read-path fix |
| T5, T6 | [#21143](https://github.com/bytebase/bytebase/pull/21143) — `sheet_blob_ref` gives sheets per-project ownership; `BatchCreateRevisions` rejects foreign-project provenance without echoing the hash |
| T7 | [#21102](https://github.com/bytebase/bytebase/pull/21102) — every `CheckRelease` target validated against the parent project before any schema read (its error codes still leak existence — T18c-iii) |
| T8 `CreateWorksheet` | [#21169](https://github.com/bytebase/bytebase/pull/21169) — `CreateSavedQuery` is now IAM-enforced; Workspace Member holds no saved-query permission, and new queries start creator-private |
| T12 `GetActuatorInfo` | [#21184](https://github.com/bytebase/bytebase/pull/21184) — anonymous access and the `name` field removed; workspace now comes from the token (the narrower surface that replaced it is T12) |
| T13 `SearchWorksheets` | [#21160](https://github.com/bytebase/bytebase/pull/21160) + [#21178](https://github.com/bytebase/bytebase/pull/21178) — visibility predicate pushed into SQL before `LIMIT` |
| T18 worksheet write/delete | [#21169](https://github.com/bytebase/bytebase/pull/21169) + [#21181](https://github.com/bytebase/bytebase/pull/21181) — per-verb permissions; SQL Editor Read User can no longer rewrite or delete |
| T9 (self-hosted) | [#21189](https://github.com/bytebase/bytebase/pull/21189) — failed logins are audited and the lockout is tested end-to-end (SaaS remainder above) |

**`AIService.Chat` — accepted, not fixed** (2026-08-18). `Chat` requires an authenticated workspace
principal and touches no database or sheet, so the only exposure is LLM spend. On Cloud the feature
is free on Bytebase's own key, so no customer budget is at risk and abuse is our own spend-management
problem. Self-hosted, the admin supplies the org's key when enabling it, so org-wide use by members
is the intended model. The missing per-user rate limit and audit are accepted with that disposition.

The three patches from earlier in the session (`SearchProjects` proto comment, `SearchProjects`
pagination, `DiffSchema` ACL) remain parked and still apply — none landed independently, and all
three targets are still present at HEAD.

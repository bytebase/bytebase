> [!CAUTION]
> **Tier 1–3 describe unpatched issues in shipped code. Treat the fix order below as the
> priority order.**
>
> Tier 1 is a credential-disclosure path and Tier 2 are cross-project data-access paths;
> neither is fixed as of this commit. Until they ship and customers have had a window to
> upgrade, do not amplify this outside the project — no blog posts, release-note call-outs,
> or advisories built from it. When an advisory is eventually written, trim the
> reproduction detail.
>
> Rotate every `directory_sync_token` (T1) regardless of fix timing — assume disclosure.

# Bytebase v1 API — consolidated audit

Against `main` @ `93671b00b0` (2026-08-04). 36 proto files, 205 RPCs.

**Method.** Four parallel implementation-focused audits (multi-resource authz, CUSTOM-auth
enforcement, list/pagination correctness, field-behavior vs. reality), plus my own mechanical
sweeps and an independent run of Google's [api-linter](https://github.com/googleapis/api-linter).
The first audit was proto-only; this pass compares **the declared contract against what the
backend actually does**, which is where everything serious lives.

**Verification legend.** ✅ = I read the code myself and confirmed the full chain.
◐ = agent-traced with exact citations, structurally consistent, not runtime-proven.

Two independent agents converged on T1 without seeing each other's work.

---

# TIER 1 — Credential disclosure

### T1 ✅ CRITICAL — SCIM admin token readable by every workspace member

| link | evidence |
|---|---|
| token auto-generated for every workspace | `backend/store/workspace.go:136` — `DirectorySyncToken: uuid.New().String()` |
| `GetSetting(WORKSPACE_PROFILE)` gates on a member-level permission | `backend/api/v1/setting_service.go:105` → `permission.WorkspaceProfileSettingsGet` |
| that permission is in the **default member role** | `backend/store/predefined_roles.go:320` (inside `WorkspaceMemberRole`, opens :307) |
| converter returns it **verbatim** | `backend/api/v1/setting_service_converter.go:348` |
| it is the sole SCIM credential | `backend/api/directory-sync/webhook.go:852` — `subtle.ConstantTimeCompare` |

Every *other* secret setting is redacted in that same converter — `ApiKey: ""` (:822), SMTP
`Password: ""` (:867). `directory_sync_token` is the one that was missed, and it sits in the one
setting readable by everybody.

SCIM exposes POST/PUT/PATCH/DELETE on `/Users` and `/Groups` (`webhook.go:109,255,281,315,441`),
entirely outside the v1 ACL interceptor and outside the audit log. Since IAM bindings name
`groups/{email}` members, rewriting the membership of a group bound to `roles/workspaceAdmin`
is workspace takeover.

Mitigation that limits *exploitation* but not *disclosure*: SCIM use requires
`FEATURE_DIRECTORY_SYNC` (`webhook.go:53`). A token leaked on a lower plan stays valid after
upgrade unless rotated. `UpdateSetting` rotates rather than clears on empty
(`setting_service.go:895`), so the credential cannot be disabled.

**Fix:** blank it in the converter like every sibling secret; expose it only through an explicit
admin-gated reveal/rotate RPC. Treat existing tokens as compromised and rotate.

### T2 ✅ HIGH — Kerberos keytab returned to the lowest project role

- Returned verbatim: `backend/api/v1/instance_service_converter.go:442` (`Keytab: m.KrbConfig.Keytab`)
- Sibling credentials are all blanked in the same file, with the comment *"We don't return the password and SSLs on reads."* (`:248-286`)
- The audit redactor **masks this exact field** — `backend/api/v1/audit.go:711` — proving it is classified sensitive; the read path simply doesn't.
- Reachable via `Database.instance_resource` (`proto/v1/v1/database_service.proto:528`, populated at `database_service.go:1192`), which needs only `bb.databases.get` — held by `ProjectViewerRole` (`predefined_roles.go:589`).

Any project viewer calls `GetDatabase` and extracts the keytab, then authenticates straight to
the database, bypassing Bytebase entirely.

Same read path, lower severity: AWS `role_arn` and `external_id` are marked `INPUT_ONLY` but
returned anyway (`instance_service.proto:756,759` → `instance_service_converter.go:302-303`).
The external ID is the confused-deputy guard shared with the customer's AWS account.

### T3 ✅ HIGH — live credentials written to the audit log in plaintext

- **Service-account API keys.** `backend/api/v1/service_account_service.go:117` (create) and `:270` (rotate) set `result.ServiceKey = serviceKey`. `audit.go` has redactors only for `ExchangeToken` (`:463`, `:512`) — none for `ServiceAccount` — so it falls through to `default: protojson.Marshal` and the live key lands in `audit_log.response`, and in stdout when `RuntimeEnableAuditLogStdout` is set (`audit.go:312`).
- The codebase already names this exact hazard for the sibling type: `redactExchangeTokenResponse` exists because *"Logging it would give anyone with audit-log read access a valid API token"* (`audit.go:612-616`).
- **◐ `UpdateSetting` payloads.** No `*v1pb.UpdateSettingRequest` case in `getRequestString` (`audit.go:446-495`), so SMTP password, AI `api_key`, all six IM secrets and the directory-sync token are logged unredacted. Note `getRequestResource` *does* have a case for this type (`audit.go:433`) — the type was handled for resource extraction but not for redaction.

Anyone with `bb.auditLogs.search`/`export`, or read access to the log pipeline, harvests these.

---

# TIER 2 — Cross-project authorization gaps

**Shared root cause.** The ACL authorizes exactly ONE resource per request — `parent`, else
`name`, else `resource`, else `project`, and only if that field carries a `resource_reference`
annotation (`backend/api/v1/acl.go:667-685`). Any *second* resource name in the body is invisible
to it. `DiffSchema` (fix already written, patch `04`) was one instance; these are the rest.

### T4 ✅ HIGH — `SearchIssues` performs no authorization for a concrete project

`backend/api/v1/issue_service.go:321-323`:

```go
var projectIDs []string
if projectID != "-" {
    projectIDs = append(projectIDs, projectID)   // no permission check
} else {
    projectIDsFilter, err := getProjectIDsSearchFilter(ctx, user, permission.IssuesGet, ...)
}
```

`auth_method = CUSTOM` makes `doIAMPermissionCheck` return early (`acl.go:246`), and `parent` has
no `resource_reference` (`issue_service.proto:270`) so the resolver yields only the workspace
fallback. `CheckPermission` appears in `issue_service.go` only at `:799` and `:1217` — neither on
this path. `workspaceMember` does not hold `bb.issues.get`.

Perverse detail: the **wildcard** `projects/-` branch is correctly filtered. The concrete-project
branch — the one that looks safe — is the hole. Any authenticated member reads every non-draft
issue in any project: titles, descriptions, creators, labels, risk levels, approval state.

**Fix:** call `CheckPermission(IssuesGet, user, workspaceID, projectID)` on the concrete branch.

### T5 ✅ HIGH — sheet content is globally readable by SHA256

`sheet_blob`'s primary key is `sha256` alone (`backend/migrator/migration/LATEST.sql:237-240`),
and `getSheet` selects `WHERE sha256 = decode(?, 'hex')` with no project or workspace predicate
(`backend/store/sheet.go:60-71`). `GetSheet` validates only that the **caller-named** project
exists and is readable, then fetches by hash (`backend/api/v1/sheet_service.go:111-147`).

The `projects/{project}/sheets/{sha}` prefix is decorative. Anyone who learns a SHA256 reads that
SQL by requesting it under a project they control. This is a boundary failure in its own right
and the amplifier for T6.

### T6 ✅ HIGH — `BatchCreateRevisions` reads another project's — and another *workspace's* — release, and leaks the sheet hash

`backend/api/v1/revision_service.go:203-210` takes `projectID` from the attacker-supplied
`revision.file` and uses it directly as the store key, never comparing it to the authorized
database's project. The mismatch error then echoes the hash (`:221-223`):

```go
fileSheet := common.FormatSheet(release.ProjectID, f.SheetSha256)
if fileSheet != revision.Sheet {
    return nil, ...errors.Errorf("The sheet in file %q is %q which is different from revision.sheet %q", ...)
}
```

Amplifiers, all verified: `FindReleaseMessage` has **no `Workspace` field**
(`backend/store/release.go:30-37`) and `ListReleases` emits no workspace predicate (`:150-154`),
so this crosses tenants; release IDs are deterministic and enumerable
(`fmt.Sprintf("%s%02d", train, nextIteration)`, `release.go:80`); and T5 turns the leaked hash
into content.

Chain: `bb.revisions.create` on one database → another team's (or another tenant's) migration SQL.

### T7 ✅ HIGH — `CheckRelease.targets` reaches arbitrary databases, including a live admin connection

`backend/api/v1/release_service_check.go:70-86` resolves database targets workspace-wide with no
comparison to the parent project. The databaseGroup branch (`:87-125`) is worse — it takes
`projectResourceID` straight from the target string and calls
`ListDatabases(ProjectID: projectResourceID)` on that other project, expanding to every match.
`grep CheckPermission release_service_check.go release_service.go` → **no matches**.

Those databases flow into stored-schema reads (`:257`, `:551`, `:747`) and
`plancheck.GetSQLSummaryReport` (`:366`), which opens a real admin driver to the target database
(`backend/runner/plancheck/statement_report_executor.go:175`) and runs EXPLAIN.

A user with only `bb.releases.check` on a throwaway project gets SQL-review advice derived from
production schemas they cannot read, plus affected-row counts computed over a live admin
connection to those databases.

### T8 ◐ HIGH — two RPCs enforce nothing at all

- **`CreateWorksheet`** (`worksheet_service.go:39-103`) — resolves the caller, checks the project exists, optionally checks the database belongs to it, then writes. No `CheckPermission`, no membership test; `parent` has no `resource_reference` so the interceptor contributes nothing. A bare member plants arbitrary SQL into any project's shared worksheet list, where legitimate members may open and run it.
- **`AIService.Chat`** (`ai_service.go:37-51`) — the `aiSetting.Enabled` toggle is the entire gate; the service holds no `iamManager`. Not a schema-exfiltration path (it dereferences no database or sheet), but it is unmetered use of the org's LLM endpoint on the org's key by any principal, with no rate limit and no audit.

---

# TIER 3 — Authentication mechanism failures

### T9 ◐ HIGH — password and MFA brute-force lockout is inert

`checkPasswordLockout`/`checkMFALockout` count **audit-log rows** for failed logins
(`auth_service.go:826-834`, filtering `payload->>'method' = '/bytebase.v1.AuthService/Login'`
with non-zero status).

✅ I confirmed the load-bearing half: a failed credential check returns at
`auth_service.go:145-148`, **before** `common.SetAuditWorkspaceID(ctx, workspaceID)` at `:169` —
the only such call on this path. The agent traced the rest: with no workspace in context, no
resources (the ACL interceptor returned early for `allow_without_credential`) and a nil response,
the audit interceptor's `parents` list stays empty and the write loop never runs.

If that holds, `passwordMaxFailedAttempts = 10/10min` and `mfaMaxFailedAttempts = 5/5min` never
fire — unlimited unauthenticated password and TOTP brute force against `/v1/auth/login`, with no
rate-limiting middleware anywhere in the stack (`backend/server/echo_routes.go:37-103`) and **no
failed-login record for detection**. No test covers the lockout.

**This one deserves a runtime check before anything else** — it is cheap to confirm (attempt 11
bad logins, see whether attempt 11 is refused) and it is the difference between "hardened" and
"no lockout at all".

### T10 ◐ HIGH — self password change and MFA disable require no re-authentication

`user_service.go:383-438`: on the self branch all checks are skipped by design. `case "password"`
calls `validatePassword`, which checks only the *policy* (`:282-295`), never the current password;
the sole guard is rejecting a new password identical to the old (`:435-438`). `case "mfa_enabled": false`
disables MFA with no OTP challenge unless workspace `Require_2Fa` is set. One borrowed session
becomes permanent account takeover. Contrast the email reset flow, which requires the code *and*
revokes all refresh tokens (`auth_service.go:1629`).

### T11 ◐ MED — the `allow_missing` create-permission guard is dead for every CUSTOM RPC

`acl.go:190-193` copies `AuthMethod` into the secondary create-check context, and `acl.go:246`
short-circuits it. So `UpdateGroup(allow_missing=true)` creates a group with no create check —
under a comment asserting the opposite (`group_service.go:167-177`: *"Permission check is now
handled by the ACL interceptor which verifies both bb.groups.update and bb.groups.create"*).
Masked today only because `bb.groups.create` is in the member role anyway.

Related: 20 of the 44 CUSTOM RPCs also declare `(bytebase.v1.permission)`, which is **dead
annotation** — it is never enforced for non-IAM auth methods.

**CUSTOM enforcement tally across all 44:** 27 ENFORCES · 14 PARTIAL · 2 DOES-NOT-ENFORCE.

### T12 ◐ MED — unauthenticated surface

12 RPCs are `allow_without_credential`. Worth attention:

- **`GetActuatorInfo`** takes `name` as-is and `acl.go:126` returns before the workspace-mismatch check at `:155`, so `GET /v1/workspaces/{anyID}/actuator/info` returns *that* workspace's default project ID, user/instance counts, external URL, and security posture (SSO configured, signup disallowed, 2FA required).
- **`Signup`** answers `AlreadyExists` for a registered email *before* the `DisallowSignup` check (`auth_service.go:272` vs `:297`) — a clean user-enumeration oracle in SaaS, contradicting the care taken in `RequestPasswordReset` and `SendEmailLoginCode`.
- **`SendEmailLoginCode`** deliberately skips the existence check for `LOGIN` (`:1759`), and the 60s cooldown is keyed per `(email, purpose)` — so it is an unauthenticated mail relay to arbitrary recipients over the customer's SMTP.
- **`ListIdentityProviders`** returns any named workspace's SSO config to anonymous callers — secrets redacted, but LDAP **host, port, bind DN, base DN, user filter** exposed. (Correction to a plausible worse reading: the empty-`parent` branch is *not* a cross-tenant dump; `store/idp.go:167-171` applies `WHERE workspace IS NULL`.)

---

# TIER 4 — Correctness bugs

### T13 ◐ HIGH — filter-after-pagination (the `SearchProjects` bug is not unique)

Same shape as the one already fixed in patch `03`:

- **`ListIssues` / `SearchIssues`** — `approval_status` and `current_approver` are applied in Go during conversion (`issue_service_converter.go:29-41`) *after* the page and token are computed (`issue_service.go:273-280`, `:347-354`). This is the "My Issues" page (`MyIssuesPage.tsx:91`), and unlike the project store, `stores/app/issue.ts` has **no** empty-page workaround. Filtering by "waiting for my approval" in a large workspace renders an empty list with a live "Load more" button.
- **`SearchWorksheets`** — worse: `store.ListWorkSheets` has *zero* visibility predicate (`store/worksheet.go:106-155`), so other users' PRIVATE worksheets are fetched, counted toward the page, then dropped (`worksheet_service.go:318-342`).

### T14 ◐ HIGH — cross-project issue pagination skips and duplicates rows

`issue.id` is per-project (`LATEST.sql:293-310`, `PRIMARY KEY (project, id)`), but the cross-project
sort is `ORDER BY issue.id DESC` (`store/issue.go:323`, `:400-406`). Ties are the dominant case,
not an edge case, so `LIMIT/OFFSET` may order a tie group differently between pages — rows are
silently omitted from one page and duplicated into another. "My Issues" always sends `projects/-`.
Needs a unique tiebreaker, e.g. `(issue.project, issue.id)`.

Softer variant (T-low): nine stores paginate on `created_at`, which is `DEFAULT now()` —
transaction time, so same-transaction rows tie exactly.

### T15 ◐ HIGH — `update_mask` silently ignored

- **`UpdateDataSource` credential rotation no-ops.** `instance_service.go:1304`/`:1343` switch on the **request's** `authentication_type`; a mask carrying only `gcp_credential` leaves it zero-valued, falls to an empty `default:`, and writes nothing — HTTP 200 while the compromised credential stays live.
- **`UpdateIssue`** implements only title/description/labels/draft (`issue_service.go:884`); `status`, `type`, `plan`, `risk_level`, `role_grant` are writable in proto with no case, so `update_mask=["status"]` returns 200 and does nothing.
- **`UpdateSetting`** ignores the mask entirely for 5 of 8 settings (full replace, `setting_service.go:509-513`). `update_mask=["value.app_im.slack"]` wipes stored Feishu/WeCom/Lark/DingTalk/Teams secrets. For `ENVIRONMENT` the replace cascades into `UpdateInstance` + `BatchUpdateDatabases` across every instance and database (`:430`, `:437`).
- **`UpdateDatabaseCatalog`** names an `update_mask` that doesn't exist in the request (already reported) and full-replaces the config column, wiping other schemas' classification and masking config.
- Four Update RPCs silently ignore unknown mask paths; four others reject them. Inconsistent contract.

### T16 ◐ MED — `BatchGet*`: four RPCs, four different behaviors, none documented

| RPC | missing entry | no-permission entry |
|---|---|---|
| `BatchGetDatabases` | silently skipped | silently skipped |
| `BatchGetProjects` | **whole batch 404s** | whole batch 403s |
| `BatchGetUsers` | silently skipped, **store order** | n/a |
| `BatchGetGroups` | silently skipped | **all errors swallowed** (a DB failure looks like "absent") |

Responses are bare `repeated X` with no ordering or omission contract, so any client zipping
`names[i]` to `results[i]` mis-attributes data. Contradicts AIP-231.

### T17 ◐ MED — frontend builds CEL filters without escaping

`frontend/src/utils/v1/cel.ts:12-22` exists precisely for this and documents the hazard — but only
2 of 14 call sites use it. The other 12 (`stores/app/project.ts:36`, `instance.ts:41,53,56`,
`user.ts:35`, `group.ts:35`, `serviceAccount.ts:41`, `plan.ts:19`, `accessGrant.ts:35`,
`utils.ts:65,136,142`) interpolate raw. Typing a `"` in those search boxes breaks search with
`InvalidArgument`; it also allows filter-semantics injection (not a privilege boundary — the
filter only narrows an already-authorized list).

### T18 ◐ MED — other

- **`BatchDeleteProjects` partial purge** (`project_service.go:504-508`): soft-delete is one statement, hard-delete is a loop; a mid-loop failure leaves some projects irreversibly purged and the rest soft-deleted, with no indication which.
- **`BatchCancelTaskRuns`** authorizes a caller-asserted stage but cancels by UID without `PlanUID`/`Environment` predicates (`rollout_service.go:1073-1110`), so test-environment rights can cancel in-flight production migrations. Sibling handlers derive the environment from real rows.
- **`SearchQueryHistories` `instance ==` filter always matches zero rows** — `store/query_history.go:247` uses `LIKE` with no `%` against a longer stored string; the proto documents the broken form (`query_history_service.proto:68,75`).
- **`ListTaskRuns` is unpaginated in the proto and unbounded in the store** (`rollout_service.proto:257`, `store/task_run.go:99-121`) — and `task_run` is the canonical unbounded-growth table, with a documented wildcard parent that fetches an entire rollout.
- **Worksheet write and delete are gated on `bb.worksheets.get`** (`worksheet_service.go:664-675`), which SQL Editor Read User holds — that role can rewrite and hard-delete any PROJECT_WRITE worksheet.
- **`IamPolicy.etag` is decorative**: handlers check `SetIamPolicyRequest.etag` (field 3) but never `IamPolicy.etag` (field 2), so the natural read-modify-write round-trip never triggers ABORTED and concurrent edits silently lose one side — including a role revocation.

---

# TIER 5 — Structural / AIP

Objective run of Google's api-linter: **3808 raw findings**, but 2060 are `field-behavior-required`
(the repo opts out via `FIELD_NOT_REQUIRED` in `buf.yaml`), 1027 are missing comments, and 144 are
Google-internal java/package rules. **567 substantive**, concentrated in `database_service` (63),
`rollout_service` (58), `subscription_service` (44), `instance_service` (42).

It independently confirms the first audit rather than contradicting it: the same 12 unpaginated
List RPCs, the same 7 `state` fields missing `OUTPUT_ONLY`, the same enum zero-value naming
(`FORMAT_UNSPECIFIED` → `EXPORT_FORMAT_UNSPECIFIED`), and the `schemaString`/`sdlSchema`/`changelogs`
HTTP patterns in `database_service` that match no declared resource pattern. It adds a cleaner
framing for 30 `name` fields: they want `IDENTIFIER`.

Still open from the first pass: only 4 files use `reserved` (7 enums and 13 messages carry
unreserved gaps — SQLite's `Engine = 5` is the proven case); 28 of 205 RPCs lack the mandated
`// Permissions required:` comment; 22 files have string fields and zero protovalidate constraints
despite `VALIDATION_STANDARDS.md`.

---

# What I'd do, in order

1. **Rotate every `directory_sync_token`** and blank it in the converter (T1). Assume disclosure.
2. **Runtime-confirm T9.** Eleven bad logins. If the eleventh is accepted, there is no lockout and that outranks everything else here.
3. **Redact the audit path** (T3) — `ServiceAccount` and `UpdateSettingRequest`. The `ExchangeToken` redactors are the template.
4. **Blank the keytab** (T2), and stop returning AWS `role_arn`/`external_id`.
5. **Close the multi-resource ACL class, not the instances** (T4–T7 + the pending `DiffSchema` patch): teach `getResourceFromRequest` to collect *every* `resource_reference`d field including oneof and repeated members, then annotate `DiffSchemaRequest.changelog`, `CheckReleaseRequest.targets`, `Revision.release/file/task_run`. `UpdateDatabase`'s project-transfer case (`acl.go:604-624`) is the in-repo precedent.
6. **Give sheets an ownership model** (T5), or at minimum scope the blob fetch by the owning project/workspace. Until then the sheet namespace is workspace-global by design, and should be documented as such.
7. **Fail closed**: reject `AUTH_METHOD_UNSPECIFIED` at startup; make `permission` on a CUSTOM RPC either enforced or a build error, since 20 of them are currently decorative.
8. **Wire api-linter into CI** at the 474-finding baseline. None of Tier 5 was visible to `buf lint`'s `BASIC` profile, which is why it accumulated.

**Note on scope.** Tier 1–3 are security issues in a shipped product. I have not filed anything or
touched any of it — this is a report. The three patches from earlier in the session
(`SearchProjects` proto comment, `SearchProjects` pagination, `DiffSchema` ACL) are still parked
and still apply cleanly.

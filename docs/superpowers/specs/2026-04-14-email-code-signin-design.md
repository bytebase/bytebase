# Email + 6-Digit Code Signin Design

**Date**: 2026-04-14
**Status**: Draft
**Scope**: Add passwordless signin/signup using email + 6-digit verification code. Also refactor password reset to use the same code-based mechanism for consistency.

---

## Goal

Let users sign in (and sign up) using only their email address and a one-time 6-digit code delivered via email. Mirror how Slack/Linear/Notion handle magic-code auth. Unify the password reset flow onto the same code mechanism.

## Decisions (from brainstorming)

| Decision | Choice | Rationale |
|----------|--------|-----------|
| Setting name | `allow_email_code_signin` | Clearest intent (not "2-step" since it replaces password, not augments it) |
| Login vs alternative | Alternative to password (not replacement) | Same login page, separate tab, coexists with password & SSO |
| Signup on unknown email | Yes, auto-create principal | Magic-code semantics; possession of inbox proves ownership |
| New principal shape | `name` defaults to email local-part; random bcrypt password | User can later set both via profile |
| Workspace assignment | Same as password Signup: join pre-invited workspace or create new | Symmetry with existing flow |
| Code expiry | 10 minutes | Balances email delivery latency with security |
| Resend cooldown | 60 seconds | Prevents mail bombing; both new login and password reset |
| Max attempts per code | 5 | Matches existing MFA lockout threshold |
| Per-email send rate | Effective ~60/hour via 60s cooldown | Simpler than audit-log counting, which fails for unauthenticated requests (audit middleware skips them when workspace is empty) |
| Code storage | Dedicated DB table (`email_verification_code`) | HA-safe; stateful semantics (attempts, cooldown) can't be done with JWT |
| Password reset | Migrate from JWT to code (shares table) | HA-correct cooldown/retry limits; unified UX |
| 2FA interaction | TOTP still required if user has it | Defense in depth; don't silently downgrade |
| RPC shape | Add `email_code` to existing `LoginRequest` | Reuses workspace resolution / MFA / token pipeline |
| SaaS toggleability | Read-only like `disallow_password_signin` | Auto-enabled on workspace creation when `EMAIL_CONFIG` is set |

## Non-goals

- Magic-link (clickable link) login — we're doing code-only.
- Per-workspace email template customization.
- Rate-limiting bypass for dev/test mode.
- Delivery receipts or bounce handling.
- Federated SSO changes.

---

## 1. Proto: New Workspace Setting Field

### Store proto (`proto/store/store/setting.proto`)

Add to `WorkspaceProfileSetting`:

```proto
// Allow signin/signup using email + a 6-digit one-time verification code.
// Requires the EMAIL setting to be configured on the workspace.
bool allow_email_code_signin = NEXT;
```

### V1 proto (`proto/v1/v1/setting_service.proto`)

Mirror the field in the v1 `WorkspaceProfileSetting`.

### V1 proto (`proto/v1/v1/actuator_service.proto`)

Add the flag to `Restriction` so the unauthenticated login page can decide whether to show the email-code tab (same plumbing as `disallow_password_signin`):

```proto
message Restriction {
  bool disallow_signup = 1 [(google.api.field_behavior) = OUTPUT_ONLY];
  bool disallow_password_signin = 2 [(google.api.field_behavior) = OUTPUT_ONLY];
  WorkspaceProfileSetting.PasswordRestriction password_restriction = 3 [(google.api.field_behavior) = OUTPUT_ONLY];
  // NEW — whether email + 6-digit code signin is enabled for this workspace.
  bool allow_email_code_signin = 4 [(google.api.field_behavior) = OUTPUT_ONLY];
}
```

Update `getAccountRestriction` in `auth_service.go` to populate the new field from `setting.AllowEmailCodeSignin`. License-gate it the same way as other restriction fields if a plan feature is required (TBD during implementation — likely no gate since email sending already requires the EMAIL setting).

### SaaS-specific behavior

- **On workspace creation**: if `EMAIL_CONFIG` env var is set, inject `allow_email_code_signin = true` via `getAdditionalWorkspaceSettings()`.
- **`UpdateSetting` validation**: in SaaS mode, this field is read-only (return `InvalidArgument` on any attempt to change).
- **Self-hosted**: workspace admin can toggle it. Setting `true` requires an existing EMAIL setting on the workspace; otherwise reject with `FailedPrecondition`.

---

## 2. Data Model

### New table `email_verification_code`

```sql
CREATE TABLE email_verification_code (
  email         text NOT NULL,
  -- Stored as EmailVerificationCodePurpose enum name (proto/store/store/email_verification_code.proto)
  purpose       text NOT NULL,
  code_hash     text NOT NULL,
  attempts      int  NOT NULL DEFAULT 0,
  expires_at    timestamptz NOT NULL,
  last_sent_at  timestamptz NOT NULL,
  PRIMARY KEY (email, purpose)
);

CREATE INDEX idx_email_verification_code_expires_at ON email_verification_code (expires_at);
```

- **PK `(email, purpose)`** — one active code per purpose per email. Resend upserts the row.
- **No workspace column** — codes are identity-scoped (matches `principal`, `web_refresh_token`).
- **`code_hash`** — HMAC-SHA256 of the 6-digit code, keyed with the server's `auth_secret`. Using HMAC with a server-side secret (vs. bare SHA-256) prevents offline brute force of the 10^6-size code space if the DB is ever compromised — the attacker would also need the auth secret to verify candidate codes.
- **`expires_at` index** — backs the background cleanup job.

### New proto (`proto/store/store/email_verification_code.proto`)

```proto
syntax = "proto3";
package bytebase.store;
option go_package = "generated-go/store";

enum EmailVerificationCodePurpose {
  EMAIL_VERIFICATION_CODE_PURPOSE_UNSPECIFIED = 0;
  LOGIN = 1;
  PASSWORD_RESET = 2;
}
```

The `purpose` column stores the enum's string name (`"LOGIN"`, `"PASSWORD_RESET"`), matching `policy.resource_type` and similar patterns.

### Store methods (`backend/store/email_verification_code.go`)

```go
type EmailVerificationCode struct {
    Email       string
    Purpose     storepb.EmailVerificationCodePurpose
    CodeHash    string
    Attempts    int
    ExpiresAt   time.Time
    LastSentAt  time.Time
}

// UpsertEmailVerificationCode — resets attempts to 0, overwrites code_hash, sets expires_at and last_sent_at.
func (s *Store) UpsertEmailVerificationCode(ctx context.Context, code *EmailVerificationCode) error

// GetEmailVerificationCode — returns (nil, nil) if no row exists.
func (s *Store) GetEmailVerificationCode(ctx context.Context, email string, purpose storepb.EmailVerificationCodePurpose) (*EmailVerificationCode, error)

// IncrementEmailVerificationCodeAttempts — atomic +1 to attempts.
func (s *Store) IncrementEmailVerificationCodeAttempts(ctx context.Context, email string, purpose storepb.EmailVerificationCodePurpose) error

// DeleteEmailVerificationCode — one-time invalidation on success OR when attempts exceeded.
func (s *Store) DeleteEmailVerificationCode(ctx context.Context, email string, purpose storepb.EmailVerificationCodePurpose) error

// DeleteExpiredEmailVerificationCodes — background cleanup.
func (s *Store) DeleteExpiredEmailVerificationCodes(ctx context.Context) error
```

Add `DeleteExpiredEmailVerificationCodes` to the existing background sweeper that runs `DeleteExpiredWebRefreshTokens`.

---

## 3. API Surface

### Modified: `LoginRequest` (`proto/v1/v1/auth_service.proto`)

Add one field:

```proto
message LoginRequest {
  // ... existing fields ...
  // 6-digit code from email for passwordless login/signup.
  // Pairs with `email`. Mutually exclusive with `password` and `idp_name`.
  optional string email_code = 9;
}
```

### New: `SendEmailLoginCode`

```proto
// Sends a 6-digit verification code to the email for login/signup.
// Always returns success (no email enumeration). Enforces 60-sec resend cooldown.
// Permissions required: None
rpc SendEmailLoginCode(SendEmailLoginCodeRequest) returns (google.protobuf.Empty) {
  option (google.api.http) = {
    post: "/v1/auth:sendEmailLoginCode"
    body: "*"
  };
  option (bytebase.v1.allow_without_credential) = true;
  option (bytebase.v1.audit) = true;
}

message SendEmailLoginCodeRequest {
  string email = 1;
}
```

### Changed: `ResetPassword` — JWT token → code

```proto
message ResetPasswordRequest {
  string email = 1;
  string code = 2;
  string new_password = 3;
}
```

### Unchanged: `RequestPasswordReset`

Signature unchanged, but behavior gains the 60-sec cooldown via the new DB row.

---

## 4. Backend Service Layer

### Constants (`auth_service.go`)

```go
const (
    emailCodeLength         = 6
    emailCodeExpiry         = 10 * time.Minute
    emailCodeMaxAttempts    = 5
    emailCodeResendCooldown = 60 * time.Second
)
```

Remove `passwordResetTokenDuration` (was 15 min JWT). Remove `GeneratePasswordResetToken` and `GetEmailFromPasswordResetToken` from `backend/api/auth/tokens.go`.

### `SendEmailLoginCode`

1. Validate `req.Msg.Email` format. If invalid → `InvalidArgument`.
2. Resend cooldown: `GetEmailVerificationCode(email, LOGIN)`. If row exists AND `now - last_sent_at < 60s` → return `Empty` silently.
3. Generate 6-digit numeric code via `crypto/rand`. Hash with HMAC-SHA256 keyed by the server's `auth_secret` (consistent with the data-model section — bare SHA-256 would be bruteforceable across the 10^6 code space).
4. `UpsertEmailVerificationCode` with `attempts=0`, `expires_at=now+10min`, `last_sent_at=now`.
5. Fire-and-forget goroutine (`context.WithoutCancel`): resolve workspace → load EMAIL setting → send code via mailer. If no workspace exists yet but `EMAIL_CONFIG` env var is set, use it.
6. Return `Empty` unconditionally.

### `Login` — new `email_code` branch in `authenticateLogin`

Dispatch order (in order of precedence):
1. `mfa_temp_token` set → MFA completion (existing)
2. `idp_name` set → IDP login (existing)
3. `email_code` set → **new email-code branch** (below)
4. else → password login (existing)

**Email-code branch:**
1. Reject if `password` or `idp_name` is also set → `InvalidArgument` "email_code is mutually exclusive with password and idp_name".
2. `verifyEmailCode(ctx, email, LOGIN, submittedCode)` (shared helper — see below).
3. Look up principal by email.
4. **Existing principal**: return user → downstream pipeline continues. The `allow_email_code_signin` check for existing users runs **later** in `validateLoginPermissions` against the **actually-resolved** login workspace (see post-auth section below). This matters for multi-workspace users: `resolveWorkspaceForLogin` prefers `LastLoginWorkspace`, which may differ from the first pre-invited workspace.
5. **New principal (signup)** — workspace-level gates run here, BEFORE user creation, to prevent orphan accounts:
   - Resolve pre-invited workspaces via `ListWorkspacesByEmail`.
   - If a pre-invited workspace exists AND `allow_email_code_signin = false` → `FailedPrecondition` "email code login is not enabled for this workspace".
   - If a pre-invited workspace exists AND `disallow_signup = true` → `Unauthenticated` "account not found" (generic, no enumeration).
   - Fail closed on store errors (return `Internal`) — a transient read failure must not bypass either gate.
   - If no resolvable workspace exists (SaaS first-time user): signup proceeds. `provisionWorkspaceForNewUser` creates a new workspace which inherits `allow_email_code_signin = true` via `getAdditionalWorkspaceSettings`.
   - Generate random 32-byte string, bcrypt-hash it.
   - Create principal: `email = email`, `name = email.split("@")[0]`, `type = END_USER`, `password_hash = hashedRandom`.
   - Workspace assignment via `provisionWorkspaceForNewUser` (shared with password Signup RPC).
   - Return new user → downstream pipeline continues.

**Post-auth: updates to `validateLoginPermissions`** (required for SaaS):
1. **Exempt email-code logins from `DisallowPasswordSignin`**: SaaS defaults to `DisallowPasswordSignin = true`, which blocks any non-IDP login. Add an `isEmailCodeLogin` branch that skips this check — email-code is a distinct auth method, not password.
2. **Enforce `allow_email_code_signin` on the resolved workspace**: for email-code logins, verify the resolved `workspaceID` (from `resolveWorkspaceForLogin`) has `allow_email_code_signin = true`. Fails closed on store errors. This covers existing-user logins correctly even when the user's `LastLoginWorkspace` differs from their first pre-invited workspace.

**Rate limiting:** `SendEmailLoginCode` has no workspace context and can't use audit-log-based counting (the audit middleware skips unauthenticated requests without workspace). The 60-second `last_sent_at` cooldown is the only rate limit, capping sends at ~60/hour/email. The cooldown check MUST be atomic with the row write to prevent a TOCTOU race: a conditional upsert (`INSERT ... ON CONFLICT DO UPDATE ... WHERE last_sent_at < now - cooldown RETURNING 1`) does the check-and-set in one statement. A separate read-then-write allows concurrent requests to both pass the cooldown and both send emails.

### Shared helper: `verifyEmailCode(ctx, email, purpose, submittedCode) error`

1. `GetEmailVerificationCode(email, purpose)`. If nil → `Unauthenticated` "invalid or expired code".
2. `expires_at > now`? If not → `DeleteEmailVerificationCode` → `Unauthenticated` "invalid or expired code".
3. `attempts < 5`? If at limit → `DeleteEmailVerificationCode` → `Unauthenticated` "too many attempts".
4. Constant-time compare HMAC-SHA256(submittedCode, auth_secret) with `code_hash`. If mismatch → `IncrementEmailVerificationCodeAttempts` → `Unauthenticated` "invalid or expired code".
5. On match: `DeleteEmailVerificationCode` (one-time use). Return nil.

### `ResetPassword` — switch to code

Body:
1. `verifyEmailCode(ctx, email, PASSWORD_RESET, req.Msg.Code)`.
2. Find user by email.
3. bcrypt-hash `new_password`, update via `UpdateUser`.
4. Revoke all refresh tokens for the user.

### `RequestPasswordReset` — add cooldown + existence check

The shared `sendEmailVerificationCode` helper enforces two safeguards for `PASSWORD_RESET`:
1. **Principal existence check**: look up the email via `store.GetAccountByEmail`. If no principal (or not an `END_USER`), return silently without upserting a row or sending an email. Prevents the endpoint from being abused to send reset emails to arbitrary addresses.
2. **Cooldown**: 60-sec `last_sent_at` check (same as LOGIN path).

Both checks are internal to the helper; the RPC itself always returns `Empty` success regardless (no enumeration).

The `LOGIN` purpose intentionally skips the existence check because it also handles signup (unknown-email → create principal).

---

## 5. Frontend

### Login page (`Signin.vue`)

Add a third tab next to "Standard" (password) and IDP tabs:

- **Tab**: `t("auth.sign-in.email-code-tab")` = "Email code"
- **Visibility**: `serverInfo?.allowEmailCodeSignin === true` (from actuator)

**Two-step UI inside the tab:**

```
Step 1 (initial):
  Email field
  [ Send code ] button

Step 2 (after Send code clicked):
  Email field (disabled, "change" link to go back to step 1)
  6-digit code input (NInputOtp, autofocus)
  [ Resend ] button (60-sec countdown)
  [ Verify ] button (disabled until 6 digits; auto-submits on complete)
```

### Frontend store (`auth.ts`)

- New: `sendEmailLoginCode(email)`
- Existing: `login()` — just pass `emailCode` field

### Code submission flow

1. `authStore.login({ email, emailCode, web: true })`
2. Same downstream as password login — if response has `mfaTempToken`, redirect to `MultiFactor.vue`. Otherwise `fetchCurrentUser()` + redirect to dashboard.

### Resend countdown

After "Send code" or "Resend", start a client-side 60-sec timer. Button shows `t("auth.sign-in.resend-in", { seconds: 45 })` while counting. Backend enforces the same window server-side regardless.

### Password reset pages — complete rewrite

**`PasswordForgot.vue`** (minor change):
- Same email-input form. Copy updated from "link" to "code". Same "Send code" + cooldown UX.
- On success, navigate to `password-reset?email=...`.

**`PasswordReset.vue`** (complete rewrite):
- No more `?token=` query param handling. No more token-based forced-reset-after-login flow (that flow is orthogonal; keep it via existing `UpdateUser` path).
- Form:
  - Email field (prefilled from query)
  - 6-digit code input
  - New password + confirm fields
  - [ Resend code ] button (60-sec cooldown)
  - [ Reset password ] button
- On submit: `authServiceClientConnect.resetPassword({ email, code, newPassword })` → navigate to signin.

### Pre-fill UX

When user clicks "Forgot password?" on signin, carry `email` through: `password-forgot?email=...` → `password-reset?email=...`.

### Removed from frontend

- `auth/password-reset?token=xxx` query handling.
- `GeneratePasswordResetToken` / reset-link construction in emails.

### i18n (en-US + mirror to zh-CN, ja-JP, es-ES, vi-VN)

```json
"auth.sign-in.email-code-tab": "Email code",
"auth.sign-in.send-code": "Send code",
"auth.sign-in.resend-code": "Resend code",
"auth.sign-in.resend-in": "Resend in {seconds}s",
"auth.sign-in.code-sent-hint": "We've sent a 6-digit code to {email}",
"auth.password-reset.code-label": "Verification code"
```

---

## 6. Error Handling

| Scenario | Response |
|----------|----------|
| `SendEmailLoginCode` with invalid email | `InvalidArgument` "invalid email" |
| `SendEmailLoginCode` within 60s cooldown | `Empty` success (silent) |
| `SendEmailLoginCode` no EMAIL setting + no `EMAIL_CONFIG` | `Empty` success, log warning |
| `SendEmailLoginCode` SMTP send fails | `Empty` success, log warning (background) |
| `Login` email_code + password or idp_name | `InvalidArgument` "mutually exclusive" |
| `Login` email_code: no row for email | `Unauthenticated` "invalid or expired code" |
| `Login` email_code: expired | Delete, `Unauthenticated` "invalid or expired code" |
| `Login` email_code: attempts >= 5 | Delete, `Unauthenticated` "too many attempts" |
| `Login` email_code: mismatch | Increment attempts, `Unauthenticated` "invalid or expired code" |
| `Login` email_code: email unknown AND a resolvable pre-invited workspace has `disallow_signup=true` | `Unauthenticated` "account not found" (generic — don't leak why) |
| `Login` email_code: pre-invited workspace has `allow_email_code_signin=false` | `FailedPrecondition` "email code login is not enabled for this workspace" (checked BEFORE any state mutation to prevent orphan accounts) |
| `ResetPassword` — all code errors mirror Login branch | Same semantics |
| `RequestPasswordReset` within 60s cooldown | `Empty` success (silent) |

**Principles:**
- No email enumeration: `SendEmailLoginCode` and `RequestPasswordReset` always return success.
- Generic error message for all verification failures: "invalid or expired code". Exception: "too many attempts" is surfaced so the user knows to request a new code.
- Constant-time compare for code hashes (matches existing `challengeRecoveryCode`).

---

## 7. Testing & Rollout

### Unit tests

- `backend/store/email_verification_code_test.go`:
  - Upsert → Get → all fields match
  - Upsert twice → second replaces first (attempts reset to 0)
  - Two purposes for same email → coexist independently
  - IncrementAttempts → atomic +1
  - Delete → row gone
  - DeleteExpired → only expired rows removed

- `backend/api/v1/auth_service_test.go` (new cases):
  - `verifyEmailCode`: happy, expired, attempts-exceeded, mismatch, missing row
  - `authenticateLogin` email-code: existing user, new user (signup), mutually-exclusive rejection

### Integration tests (`backend/tests/auth_test.go`)

- `SendEmailLoginCode` + `Login(email_code)` happy path → token returned, user created if new
- Resend within 60s → silently ignored, stored row unchanged
- 5 wrong codes → 6th request returns "too many attempts", row deleted
- Expired code (mock clock) → login fails
- Password reset: `RequestPasswordReset` → `ResetPassword(email, code, new_password)` → can login with new password; old refresh tokens revoked
- `allow_email_code_signin = false` on workspace → email-code login rejected
- `disallow_signup = true` on workspace + unknown email → login rejected with "account not found"; existing user unaffected
- 2FA interaction: TOTP-enabled user → email-code login returns `mfa_temp_token` → regular MFA flow completes

### Migration & rollout

- **Migration file**: `backend/migrator/migration/NNNN/0000_add_email_verification_code.sql`. Bump `TestLatestVersion`. Update `LATEST.sql`.
- **Proto generation**: `cd proto && buf format -w . && buf lint && buf generate`.
- **No feature flag**: gated by `allow_email_code_signin` workspace setting. Default `false` self-hosted. SaaS workspaces get `true` when `EMAIL_CONFIG` is set.
- **Background cleanup**: add `DeleteExpiredEmailVerificationCodes` to the existing sweeper alongside `DeleteExpiredWebRefreshTokens`.
- **Breaking change**: `ResetPassword` signature changes from `(token, new_password)` to `(email, code, new_password)`. Internal feature, 2 days old → acceptable. PR labeled `breaking`.
- **Audit logs**: `SendEmailLoginCode`, `Login` (existing), `RequestPasswordReset` (existing), `ResetPassword` all have `option (bytebase.v1.audit) = true`.

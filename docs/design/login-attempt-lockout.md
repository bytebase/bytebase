# Login attempt lockout

One table, `login_attempt`, gives every credential — password, emailed code, second factor — an
attempt limit keyed on the identity under attack, identical on Cloud and self-hosted. It replaces
the two counters that fail today: the audit-log count, which reads zero on Cloud, and the per-code
`attempts` column, which an attacker resets by exhausting it. Closes T9 in
[`v1-api-audit-2026-08.md`](v1-api-audit-2026-08.md).

## Background

Guessing is throttled at three points by three mechanisms:

| Credential | Today |
|---|---|
| Password | count failed-login rows in `audit_log`; 10 in 10 minutes → `ResourceExhausted` |
| Emailed code (Cloud's primary factor; password reset) | `attempts` column on the code row; 5 wrong guesses → row deleted |
| MFA (TOTP, recovery code) | the same audit-log count; 5 in 5 minutes → `ResourceExhausted` |

### Problem

- **The audit-log counter reads zero on Cloud.** A failed login's audit row needs a workspace. On
  SaaS the only source is the request's own `workspace` field, which an attacker omits, so the row
  is never written and the password and MFA lockouts never fire. One MFA temp token buys five
  minutes of unmetered TOTP guesses. Self-hosted was fixed in #21189 but still reads the audit
  log, so a retention purge or the BYT-10124 rework can switch it off again.
- **The per-code counter defeats the resend cooldown.** The 60-second cooldown is a predicate on
  the code row, and the row is deleted after the fifth wrong guess. Send, guess five times, trigger
  the delete, send again: five fresh guesses at a 10⁶-space code per seven requests, bounded only
  by throughput. This is the first factor of every MFA-less Cloud account. Even without the
  delete, five guesses per code at one code a minute is 7,200 a day.
- **The code row stores caller-supplied context.** `workspace` is whatever the send request
  named, kept so verify can re-apply that workspace's policies. It binds nothing — the membership
  check at verify is what stops a user choosing a weaker policy — and by verify time the server
  knows the user's workspaces itself.

## Goals

- **G1** One behavior on Cloud and self-hosted, keyed on the identity under attack — never on
  optional request context a caller can omit to make the record vanish.
- **G2** Guessing is bounded per identity across codes and tokens, independent of request rate.
- **G3** No existence oracle: unknown and known emails lock at the same attempt with the same error.
- **G4** Locks expire on their own; no admin unlock.
- **G5** Correct under replicas: state in the metadata database, one atomic statement per attempt.
- **G6** Password and MFA thresholds, error codes, and messages unchanged.

### Non-goals

- **Per-IP throttling and send caps.** Edge concerns: Cloudflare rules on Cloud, the reverse proxy
  on self-hosted. See [Security and performance](#security-and-performance).
- **A per-tenant failed-login record on Cloud.** Accepted in T9.
- **Single-use MFA temp tokens.** Unnecessary once guesses are bounded per identity.
- **The SaaS password oracle** (wrong password → `Unauthenticated`, right password →
  `PermissionDenied` after bcrypt). Now bounded by the `PASSWORD` lock; the reorder must keep
  service-account password login working, so it is its own change.

## Design

### Table

```sql
-- One row per (identity, kind): attempts since the last success, and when the latest was.
CREATE TABLE login_attempt (
    identity        text NOT NULL,   -- the identity under attack, server-resolved and globally unique (see Semantics); not a FK, so unknown ones count (G3)
    kind            text NOT NULL,   -- LoginAttemptKind: PASSWORD | EMAIL_CODE | MFA
    attempts        int NOT NULL,
    last_attempt_at timestamptz NOT NULL,
    PRIMARY KEY (identity, kind)
);

CREATE INDEX idx_login_attempt_last_attempt_at ON login_attempt (last_attempt_at);
```

No `workspace` column: the credential is per identity and opens every workspace that identity
belongs to. No `ip` column: the caller IP is forgeable, and per-source throttling is an edge concern.

### Semantics

| Kind | Attempts `N` | Duration `D` |
|---|---|---|
| `PASSWORD` | 10 | 10 min |
| `EMAIL_CODE` | 5 | 10 min |
| `MFA` | 5 | 5 min |

1. **An attempt claims a slot before the credential is checked.** `N` attempts are granted. If
   the identity already holds `N` and the latest was under `D` ago, the next gets no slot:
   `ResourceExhausted`, before any bcrypt, TOTP, or hash comparison.
2. **The counter forgets after `D` of quiet.** A claim more than `D` after the latest attempt
   restarts at one. Locked attempts do not claim, so a lock lasts exactly `D` from the `N`th.
3. **Success deletes the row.**

The `identity` is the globally-unique subject of the attack, resolved by the server so a caller
cannot merge two identities into one bucket or split one across buckets (G1):

- local password and emailed code: the normalized email;
- MFA: the email inside the signed temp token;
- LDAP: the resolved identity-provider ID joined with the submitted username. The same username in
  another directory is a different person and must be a different row — the point sharpened by
  success deleting the row: keyed by bare username, an attacker who controls that username in a
  directory of their own could clear a victim's counter by logging in there.

Every request field that becomes an identity is bounded at the proto edge (emails at 254
characters, `string.max_len` enforced by the validate interceptor), invalid email syntax is
rejected before the claim, and the store refuses structurally oversized keys — so garbage never
writes a row.

This is Vault's threshold / duration / counter-reset model with the last two collapsed, as Vault's
own defaults do. It is at least as strict as a sliding window everywhere, and stricter against an
attacker who spaces guesses just under `D` apart.

### One statement per attempt

The same guarded-upsert shape the store already uses for the resend cooldown. `$3` is `N`, `$4`
is `D` in seconds:

```sql
INSERT INTO login_attempt (identity, kind, attempts, last_attempt_at)
VALUES ($1, $2, 1, now())
ON CONFLICT (identity, kind) DO UPDATE SET
    attempts = CASE
        WHEN login_attempt.last_attempt_at < now() - make_interval(secs => $4) THEN 1
        ELSE login_attempt.attempts + 1
    END,
    last_attempt_at = now()
WHERE login_attempt.attempts < $3
   OR login_attempt.last_attempt_at < now() - make_interval(secs => $4)
RETURNING 1;
```

A row back is a slot; no row is a lockout. The row lock serializes concurrent claims, so the
`N`th slot goes to exactly one of them (G5). `now()` is database time, so replicas cannot
disagree. A slot spent on a non-credential error (database failure mid-verification) is not
refunded.

The store exposes exactly three operations: claim an attempt, clear on success, purge stale rows.

### Where the claims happen

Password login claims `PASSWORD` once, before the credential is checked — under the email for a
local password, under the provider-scoped identity for an LDAP bind — and clears it on success. Verifying an emailed code claims `EMAIL_CODE` before the code row is even loaded, for
login and password-reset codes alike, and clears it on a match. Completing MFA — during login or
when switching workspaces — claims `MFA` for the email inside the signed temp token and clears it
on success. A successful password reset also clears `PASSWORD`, as Grafana and Mattermost do, so
a user who locked themselves out is not still locked with the new password. The three
audit-counting checks they replace are deleted.

### `email_verification_code`

Becomes `(email, purpose, code_hash, expires_at, last_sent_at)`: a code, its expiry, and the
resend cooldown.

- **Loses `attempts`**, `IncrementEmailVerificationCodeAttempts`, and the delete on exhaustion.
  The row lives until it expires or is consumed, so the cooldown always has a row to evaluate: the
  bypass is closed structurally. Sending is unaffected by a lock — a new code can be requested
  (cooldown applies) but not verified until the lock expires, so the send path needs no check. The
  lockout error becomes `ResourceExhausted` / `too many attempts, please try again later` (was
  `Unauthenticated`); nothing depends on the old form. Deliberately, five wrong codes now lock the
  email for ten minutes, where today a fresh code was one request away — the price of a bound that
  survives code rotation.
- **Loses `workspace`.** Verify never reads a caller-supplied workspace; it uses what the server
  knows about the email. Password reset: once the code verifies, the user is known, so the
  password policy and the audit workspace come from the user's own memberships — the singleton on
  self-hosted. Email-code signup: the gates that run before creating a new user check the singleton on
  self-hosted, or on SaaS the workspace whose invitation the email holds — the same lookup
  provisioning already uses; a brand-new signup has neither and gets the SaaS defaults, as today.
  Existing users are checked after authentication against their resolved workspace, as today.

### Audit log and cleanup

The audit interceptor is untouched: self-hosted keeps writing the failed and locked-out login
rows it writes today, so the existing end-to-end lockout test holds. The hourly data cleaner also
purges rows older than an hour — the longest `D` is ten minutes, so they are dead — which caps the
table at one row per `(identity, kind)` attempted in the last hour.

### Security and performance

**Security: better everywhere except one regression.** Cloud gets working lockouts and the bypass
closes. The regression: the lockout itself becomes a silent denial of service — five requests per
ten minutes hold an email's `EMAIL_CODE` lock, no code pending, no email sent. No rate limit stops
one request every two minutes, keying on IP would hand a botnet unlimited guesses, and Clerk,
Stytch, and GitLab ship the same exposure — so this ships accepted. Remedy if a targeted lockout is
observed: OWASP's device-cookie bucket — a signed cookie set on web login routes a returning
browser's attempts to a trusted bucket an attacker cannot reach; the key grows to
`(identity, kind, trusted)`, a free change on a table purged hourly. One soft spot: an LDAP filter
that accepts several identifier forms gets one bucket per form; the directory's own lockout backs
it.

**Performance: net improvement.** An indexed single-row upsert replaces an audit-log scan per
attempt, locked attempts return before bcrypt, and one identifier's attempts serialize on its own
row, throttling only that attacker. A successful login holds its slot only for the verification's
duration, so more than `N` concurrent correct-credential logins for one identity can see transient
refusals until the first completes — accepted; a retry succeeds. The one new cost — a small row per distinct identifier
attempted, an unauthenticated write path Cloud did not have — is bounded by pre-claim validation,
the hourly purge, and edge rate caps.

**Edge rules (configuration, not code).** Cloudflare does nothing for these paths until rules
exist. `Login` / `ResetPassword`: ~30 per 10 min per IP, counting only `401`/`429` responses, so
service accounts and CI logging in successfully from shared IPs never trip it. For that filter to
catch LDAP spraying, an invalid-credential bind must surface as `Unauthenticated`, not the
`Internal` (500) it returns today — a small pre-existing fix, without which the edge misses failed
LDAP binds and the per-username rows they write. The two send paths, which always answer OK to
avoid enumeration: the same limit on all requests — tolerates an office NAT, caps single-IP
mail-bombing. No challenges on auth paths; CLI and CI cannot solve them.
Preconditions: origin locked to Cloudflare's ranges, `CF-Connecting-IP` as the audit caller IP.
Self-hosted: the reverse proxy in front, as Gitea and Kratos do.

## Alternatives

- **Announce the principal's workspace so the audit row is written on Cloud** (T9's original
  fix). Still reads the audit log; nothing for emails with no workspace; leaves the code path.
- **Centralize password and MFA, keep the per-code counter.** Two mechanisms; 7,200 guesses a
  day even with the bypass patched (fails G2).
- **Counter on the `principal` row** (Devise, ASP.NET). No row for unknown emails → existence
  oracle on the Nth attempt (fails G3).
- **Log-count table** (Grafana, Authelia). N× the rows, a purge that must never lag, no atomic slot.
- **Reuse `email_verification_code` with new purposes.** Saves one `CREATE TABLE`; costs
  placeholder `code_hash`/`last_sent_at` and a table whose name lies.
- **In-memory or Redis counters.** Wrong under replicas (fails G5); no Redis dependency to add.
- **Exact sliding window.** Needs per-failure rows; differs only against a drip-feeding attacker.
- **Lock until admin unlock** (Mattermost, pgAdmin, Zitadel). Fails G4; no admin on Cloud.
- **Keep `workspace` on the code row, or read it from the verify request.** Both keep a
  caller-supplied value in a policy decision, and the second needs a new `ResetPasswordRequest`
  field. The server knows the user's workspaces after the code verifies.
- **Claim only when a code is pending.** Makes the lockout denial noisy (the attacker must trigger
  a send every ten minutes) without preventing it, and adds a pending-code oracle.
- **Key LDAP by the bare username.** Since success deletes the row, an attacker who controls the
  same username in a directory of their own clears the victim's counter by logging in there
  (fails G1).

## Reference

Surveyed 29 products for lockout and 18 for emailed codes (2026-08-23). Every element here has
precedent:

| Element | Same as |
|---|---|
| Lockout state in dedicated state, not derived from an audit log | every product that has a lockout |
| Separate table keyed by the submitted identifier, unknown IDs counted | Grafana, Nextcloud, django-axes, Laravel, Authentik |
| One record for password and second factor | GitLab, Keycloak, Zitadel, Clerk, Stytch |
| Threshold + duration, counter forgets after idle | Vault, Sourcegraph, ASP.NET Identity |
| Temporary self-expiring lock, reset on success | GitLab 10/10 min, ASP.NET 5/5 min, Keycloak, Vault, Entra |
| Slot claimed atomically before verification | Mattermost |
| Emailed-code failures feed the per-identity lockout, no per-code cap | Zitadel, Keycloak (same user lock as passwords), Clerk (10 → 1 h), Stytch (10 → 1 h), Okta (5 → authenticator blocked), Cognito |
| 6 digits, ≤10-minute validity, single use, hashed | NIST 800-63B r4, Clerk, WorkOS, Twilio |

The other standard shapes for codes are a cap per code (Auth0: 3; Hanko: 3; Twilio Verify: 5
checks, 5 sends per 10 minutes) and a counter on the flow that survives resends (SuperTokens,
Kratos). A bound that survives code rotation is what the bypass needs; per identity is the simpler
place to keep it, and the one that also covers passwords and MFA.

Related: [`v1-api-audit-2026-08.md`](v1-api-audit-2026-08.md) T9 ·
[BYT-10068](https://linear.app/bytebase/issue/BYT-10068) / #21189 (self-hosted half) ·
[BYT-10124](https://linear.app/bytebase/issue/BYT-10124).

Sources: [Vault user lockout](https://developer.hashicorp.com/vault/docs/concepts/user-lockout) ·
[Keycloak `LOGIN_FAILURE`](https://github.com/keycloak/keycloak/blob/main/model/jpa/src/main/resources/META-INF/jpa-changelog-26.7.0.xml) ·
[GitLab devise config](https://gitlab.com/gitlab-org/gitlab/-/blob/master/config/initializers/8_devise.rb) ·
[Mattermost claim](https://raw.githubusercontent.com/mattermost/mattermost/master/server/channels/app/authentication.go) ·
[Grafana `login_attempt`](https://raw.githubusercontent.com/grafana/grafana/main/pkg/services/loginattempt/loginattemptimpl/store.go) ·
[django-axes](https://github.com/jazzband/django-axes/blob/master/axes/models.py) ·
[ASP.NET LockoutOptions](https://github.com/dotnet/aspnetcore/blob/main/src/Identity/Extensions.Core/src/LockoutOptions.cs) ·
[Zitadel `checkOTP`](https://github.com/zitadel/zitadel/blob/main/internal/command/user_human_otp.go) ·
[SuperTokens passwordless](https://github.com/supertokens/supertokens-core/blob/master/src/main/java/io/supertokens/passwordless/Passwordless.java) ·
[Clerk user lockout](https://clerk.com/docs/guides/secure/user-lockout) ·
[Stytch user locks](https://stytch.com/docs/resources/platform/user-locks) ·
[Okta OTP lockout](https://support.okta.com/help/s/article/configurable-lockout-settings-for-multifactor-authentication-failure-attempts) ·
[Auth0 email OTP](https://auth0.com/docs/authenticate/passwordless/authentication-methods/email-otp) ·
[Twilio Verify check](https://www.twilio.com/docs/verify/api/verification-check) ·
[OWASP device cookies](https://owasp.org/www-community/Slow_Down_Online_Guessing_Attacks_with_Device_Cookies) ·
[NIST SP 800-63B r4](https://pages.nist.gov/800-63-4/sp800-63b.html).

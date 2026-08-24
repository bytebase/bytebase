# Require re-authentication to change your own credentials

Status: proposal · 2026-08-24

`UpdateUser` lets an authenticated caller rewrite their own password, TOTP secret, and recovery
codes without proving they still control the credential being replaced. Closes T10 in
[`v1-api-audit-2026-08.md`](v1-api-audit-2026-08.md) entirely: password and MFA lifecycle both move
off `UpdateUser` onto their own methods, not just a field addition — see
[Resource design](#resource-design).

## Background

`UpdateUser` (`backend/api/v1/user_service.go:305`) serves two callers: a person editing their own
account (`callerUser.ID == user.ID`), and — self-hosted only — an admin with `bb.users.update`
editing someone else's, the recovery path [`split-profile-self-service-vs-admin.md`](split-profile-self-service-vs-admin.md)
covers. This doc is only about the first caller.

### Problem

The audit names two self-service paths: **password change** (`:371-378`) hashes and stores whatever
`password` the caller sends, and **2FA disable** (`:395-411`) clears `MFAConfig` on `mfa_enabled=false`
— neither checks that the caller still controls what's being replaced. Refresh tokens are revoked on
password change, but the attacker's access token keeps working for up to `access_token_duration`
(default 1h) after the account is already gone. A stolen laptop, shared browser, or leaked token is
enough to lock the real owner out permanently.

**Two more paths do the same damage without touching `password` or `mfa_enabled=false` at all**,
found while tracing what else writes `MFAConfig`:

- **`regenerate_recovery_codes=true`** (`:474-485`) promotes temp recovery codes to live ones,
  checked against nothing but an existing `OtpSecret`. The temp codes come from a prior
  `regenerate_temp_mfa_secret` call, itself unchecked. Two calls, no credential needed, and a
  stolen-session attacker holds working recovery codes.
- **`mfa_enabled=true`** (`:379-394`) promotes a temp TOTP secret over whatever is already live —
  reachable whether or not MFA was already on. Over an existing device, this **swaps it out without
  ever showing as "2FA disabled."** A victim checking their settings sees 2FA still on and has no
  reason to suspect the device isn't theirs anymore.

A fix scoped to literally the two audited lines leaves both of these open. All four live behind one
`update_mask` and three booleans on one RPC — itself part of why the gap existed (see
[Resource design](#resource-design)).

## Goals

- **G1** Every mutation that rewrites live authentication material — password hash, permanent TOTP
  secret, or permanent recovery codes — requires proof the caller currently holds a credential on the
  account: the current password, or, where the account has MFA, a live MFA code instead.
- **G2** Password change gets the same treatment as MFA: its own method, not a field plus a flag on
  a generic `Update` — matching this repo's own precedent. `UpdateEmail` already left `UpdateUser`
  for exactly this reason (it "changes the identity the person signs in with," per
  [`split-profile-self-service-vs-admin.md`](split-profile-self-service-vs-admin.md)); password is
  the same class of field.
- **G3** MFA enrollment, disablement, and recovery-code rotation are each their own method too, so a
  future MFA-adjacent change can't reopen this via a flag nobody thought to gate.
- **G4** No new elevated-session or token concept — proof is supplied inline, per request, from
  credentials the account already has.
- **G5** The check can't become a fresh guessing oracle — it shares the bound T9 already built for
  this exact problem shape, for both the password and MFA cases.
- **G6** Admin-assisted recovery (self-hosted only) is untouched: an admin resetting a locked-out
  user's password or MFA must not need the credential being replaced
  ([`split-profile-self-service-vs-admin.md`](split-profile-self-service-vs-admin.md)).
- **G7** All four state-changing methods revoke the account's other sessions on success, same as
  password change already does — not just closing the credential-rewrite path, but not leaving a
  second stolen token (a refresh token in particular, valid up to 7 days) sitting live after the
  account owner has acted.

### Non-goals

- A time-boxed elevated session (GitHub sudo mode) or the formal OAuth step-up challenge (RFC 9470)
  — real session-state infrastructure for a handful of settings screens. See Alternatives.
- A general multi-factor collection (`users/{email}/mfaFactors/{id}`). Bytebase has one factor type
  and one factor per account; that cardinality doesn't exist yet.
- Whether SSO-provisioned accounts should be able to set a local password at all — pre-existing,
  orthogonal (`auth_service_idp.go:118-134`).
- Session/device listing and revocation — a separate feature (distinct from G7's automatic revocation
  on these specific actions, which is in scope).
- **Notifying the account owner by email when their password or MFA changes.** Standard elsewhere —
  GitLab, Google, and GitHub all do it (Reference) — but no such mechanism exists anywhere in this
  codebase today; the only email path near this code is the verification-code send for
  `RequestPasswordReset`. Detection, not prevention: T10 is closed by `CredentialProof` regardless of
  whether the owner is notified. Worth a follow-up, not bundled into this fix.

## Resource design

MFA enrollment isn't a field edit — it's a two-step handshake (mint a secret, prove the caller
scanned it) plus two more state transitions that don't compose with a `FieldMask`. Every MFA-capable
API surveyed (Okta, Auth0, AWS IAM, WorkOS, Google Identity Platform, Zitadel, Keycloak — see
Reference) models it as its own resource or its own methods, never as flags on the account's generic
update call. None support the shape Bytebase has today: `mfa_enabled` doubling as both an output
status and a write trigger (disambiguated only by whether it's in `update_mask.paths`), and
`temp_otp_secret` / `temp_recovery_codes` — enrollment-flow-local scratch state — living as
permanent fields on `User` because they had nowhere else to go.

Password change is a plainer field, but the same argument applies once it needs a credential proof
rather than just a new value: AWS splits self-service `ChangePassword` from admin `UpdateLoginProfile`
outright, and this repo already draws that line for email — `UpdateEmail` is its own RPC because
identity-changing fields get their own call and their own confirmation, not a shared PATCH. Password
change moves to a new `ChangePassword` method for the same reason; `UpdateUser` keeps `password` only
for the admin-assisted path, where G6 means no proof is needed anyway.

## Design

**Verification.** `ChangePassword`, `DisableMfa`, and `RegenerateRecoveryCodes` each take a
`CredentialProof` — either the current password or, when the account has MFA, a live OTP or recovery
code — and check it before touching anything: claim the matching slot (`PASSWORD` or `MFA`) in the T9
`login_attempt` table for the account ([`login-attempt-lockout.md`](login-attempt-lockout.md)),
verify with `bcrypt.CompareHashAndPassword` or the existing `challengeMFACode`/`challengeRecoveryCode`
helpers, clear the slot on success. Reusing T9's table means an attacker with a stolen session but no
credential gets the same five-guesses-per-ten-minutes bound as at login — not a fresh oracle, and no
new lockout kind to build. `EnableMfa` takes the same `CredentialProof` as the other three — accepting
the old device's code (or a recovery code) as an alternative to password matters most exactly when
replacing a lost device, and password isn't always available as the fallback (see Cloud vs.
self-hosted). Because each method requires its proof in its own request message, whether it needs
re-authentication is visible in its schema — nothing to audit across a shared patch. On success, all
four call `DeleteWebRefreshTokensByUser` — today only `ChangePassword` inherits this from
`UpdateUser`'s existing password branch (`user_service.go:436`); `EnableMfa`/`DisableMfa`/
`RegenerateRecoveryCodes` need the same call added, forcing re-login everywhere, including the
caller's own current session (G7).

Sharing T9's lockout buckets has one accepted side effect worth naming, not just inheriting silently:
a `PASSWORD` or `MFA` claim from a failed `CredentialProof` counts against the same budget a login
attempt would. A user who fumbles their password a few times inside `ChangePassword` can lock
themselves out of *logging in* too, not just out of changing it — a narrower version of the
lockout-as-availability-cost trade [`login-attempt-lockout.md`](login-attempt-lockout.md) already
accepted for a different reason. Anyone locked out this way still has the request's own error message
telling them when it clears.

**What's exempt.** `title`/`phone` on `UpdateUser` aren't authentication material.
`StartMfaEnrollment` mints a secret but changes nothing live, so it needs no proof — same as AWS's
`CreateVirtualMFADevice` needing none while `EnableMFADevice` does. `UpdateUser`'s `password` field
becomes admin-only: a self-service call (`callerUser.ID == user.ID`) that sets it is rejected,
pointing at `ChangePassword`. `ChangePassword` has no admin path at all (see its `name` field comment
in API). Admin-assisted resets of another user's password or MFA stay on `UpdateUser`'s `password`
field and `DisableMfa` respectively, both exempt: the `bb.users.update` permission check and audit log are the
correct control there, and an admin recovering a locked-out user cannot know the credential being
replaced. `CreateUser`/`Signup` have no prior credential to prove. `ResetPassword` (emailed code)
already has its own proof channel — mailbox possession — including for SSO accounts whose
`PasswordHash` they never saw.

**Cloud vs. self-hosted.** Cloud users never know their own local password — email-code signup and
SSO both assign a random, unseen one (`auth_service_email_code.go:252-258`,
`auth_service_idp.go:119-127`), and `Login` itself refuses the password path via
`restriction.disallow_password_signin`, which SaaS forces on unconditionally. `ChangePassword`
therefore rejects any call when that restriction is set, reusing the flag `Login`/`Signup` already
check rather than a separate `s.profile.SaaS` branch — the same behavior falls out for a self-hosted
deployment that goes SSO-only. This also closes a gap in today's code: `UpdateUser`'s password branch
has no SaaS check at all, harmless only because `Login` independently blocks the result. MFA is
unaffected either way — it's a second factor on top of whichever primary method a workspace allows,
so `Require_2Fa` and all four MFA methods behave identically on both deployments, and `EnableMfa`
needing `CredentialProof` rather than password-only is exactly what makes device replacement possible
for a Cloud account that has no password to fall back to.

**Frontend.** The four call sites in `AccountSettingsPage.tsx`, `TwoFactorSetupPage`, and
`RegenerateRecoveryCodesView` move to their matching new method and gain a credential-proof field —
no new dialogs, since [`split-profile-self-service-vs-admin.md`](split-profile-self-service-vs-admin.md)
already gives password and 2FA changes their own confirmations. This is a breaking proto change;
frontend and backend ship as one rollout on both Cloud and self-hosted (a single `go:embed`'d image,
confirmed against `scripts/Dockerfile` and the Cloud deploy workflow — no separate frontend pipeline),
same as the other breaking changes already in flight this cycle (#21181, #21234). One residual edge,
inherent to any SPA rolling deploy rather than specific to this change: a browser tab already open
before the rollout is still running the old bundle and can call the old fields on `UpdateUser` against
an already-upgraded backend for the rest of that rollout window — they're reserved, so the call
silently no-ops rather than erroring. Self-correcting on the next page load; not worth engineering
around for a field that's disappearing anyway.

## API

Five new methods on `UserService` (same service `UpdateEmail` already lives in, rather than a new
service), one per credential state transition. Full proto below, then what each piece is doing.

### Service methods

```protobuf
service UserService {
  // ...existing RPCs...

  // Changes the caller's own password. Rejected if the workspace disallows
  // password sign-in (Cloud always does) — see Design → Cloud vs. self-hosted.
  // Permissions required: none beyond being signed in as `name`.
  rpc ChangePassword(ChangePasswordRequest) returns (User) {
    option (google.api.http) = {
      post: "/v1/{name=users/*}:changePassword"
      body: "*"
    };
    option (google.api.method_signature) = "name,new_password,credential";
    option (bytebase.v1.auth_method) = CUSTOM;
    option (bytebase.v1.audit) = true;
    option (bytebase.v1.mcp_method_class) = FORBIDDEN;
    option (bytebase.v1.mcp_denial_reason) = TAKES_OVER_ACCOUNT;
  }

  // Mints a new TOTP secret and recovery codes for the caller to confirm.
  // Inert until EnableMfa promotes it, so no proof is required here.
  rpc StartMfaEnrollment(StartMfaEnrollmentRequest) returns (MfaEnrollment) {
    option (google.api.http) = {
      post: "/v1/{name=users/*}:startMfaEnrollment"
      body: "*"
    };
    option (google.api.method_signature) = "name";
    option (bytebase.v1.auth_method) = CUSTOM;
    option (bytebase.v1.mcp_method_class) = FORBIDDEN;
    option (bytebase.v1.mcp_denial_reason) = MINTS_CREDENTIAL;
  }

  // Confirms the code from a started enrollment and promotes it to the
  // caller's live MFA factor, replacing any existing one.
  rpc EnableMfa(EnableMfaRequest) returns (User) {
    option (google.api.http) = {
      post: "/v1/{name=users/*}:enableMfa"
      body: "*"
    };
    option (google.api.method_signature) = "name,otp_code,credential";
    option (bytebase.v1.auth_method) = CUSTOM;
    option (bytebase.v1.audit) = true;
    option (bytebase.v1.mcp_method_class) = FORBIDDEN;
    option (bytebase.v1.mcp_denial_reason) = TAKES_OVER_ACCOUNT;
  }

  // Turns the caller's MFA off.
  rpc DisableMfa(DisableMfaRequest) returns (User) {
    option (google.api.http) = {
      post: "/v1/{name=users/*}:disableMfa"
      body: "*"
    };
    option (google.api.method_signature) = "name,credential";
    option (bytebase.v1.auth_method) = CUSTOM;
    option (bytebase.v1.audit) = true;
    option (bytebase.v1.mcp_method_class) = FORBIDDEN;
    option (bytebase.v1.mcp_denial_reason) = TAKES_OVER_ACCOUNT;
  }

  // Replaces the caller's recovery codes.
  rpc RegenerateRecoveryCodes(RegenerateRecoveryCodesRequest) returns (RegenerateRecoveryCodesResponse) {
    option (google.api.http) = {
      post: "/v1/{name=users/*}:regenerateRecoveryCodes"
      body: "*"
    };
    option (google.api.method_signature) = "name,credential";
    option (bytebase.v1.auth_method) = CUSTOM;
    option (bytebase.v1.audit) = true;
    option (bytebase.v1.mcp_method_class) = FORBIDDEN;
    option (bytebase.v1.mcp_denial_reason) = MINTS_CREDENTIAL;
  }
}
```

`mcp_denial_reason` isn't the same value on every method, even though all five are `FORBIDDEN`: the
enum distinguishes "rewrites credentials, response carries none" (`TAKES_OVER_ACCOUNT`) from "puts a
usable secret directly in the response body" (`MINTS_CREDENTIAL`). `StartMfaEnrollment` returns a
fresh TOTP secret; `RegenerateRecoveryCodes` returns fresh, immediately-live codes — both get
`MINTS_CREDENTIAL`. The other three return only `User`, so they get `TAKES_OVER_ACCOUNT`, matching
what `UpdateUser` already carries today for the same operations.

### Messages

```protobuf
// One proof that the caller currently controls a credential on the account.
// Exactly one field must be set; enforced in the handler, not via
// buf.validate.oneof.required — no other message in this repo uses that
// constraint, so this doesn't introduce a second convention for it.
message CredentialProof {
  oneof proof {
    // The account's current password.
    string current_password = 1 [
      (bytebase.v1.audit_behavior) = SENSITIVE,
      (buf.validate.field).string.max_bytes = 72
    ];

    // A live code from the account's enrolled TOTP authenticator.
    string otp_code = 2 [
      (bytebase.v1.audit_behavior) = SENSITIVE,
      (buf.validate.field).string.max_len = 64
    ];

    // A single-use MFA recovery code.
    string recovery_code = 3 [
      (bytebase.v1.audit_behavior) = SENSITIVE,
      (buf.validate.field).string.max_len = 64
    ];
  }
}

message ChangePasswordRequest {
  // Format: users/{email}. Must be the caller's own name — CredentialProof
  // proves control of *your* credential, which no one can supply for someone
  // else, so this is never valid on an admin-assisted call.
  string name = 1 [
    (google.api.field_behavior) = REQUIRED,
    (google.api.resource_reference) = {type: "bytebase.com/User"}
  ];

  string new_password = 2 [
    (google.api.field_behavior) = REQUIRED,
    (bytebase.v1.audit_behavior) = SENSITIVE,
    (buf.validate.field).string.max_bytes = 72
  ];

  CredentialProof credential = 3 [(google.api.field_behavior) = REQUIRED];
}

message StartMfaEnrollmentRequest {
  string name = 1 [
    (google.api.field_behavior) = REQUIRED,
    (google.api.resource_reference) = {type: "bytebase.com/User"}
  ];
}

// Enrollment-flow-local; never persisted as such. Returned only by
// StartMfaEnrollment, consumed only by EnableMfa. Replaces
// User.temp_otp_secret / temp_recovery_codes / temp_otp_secret_created_time,
// which had no reason to be permanent fields on the account resource.
message MfaEnrollment {
  string otp_secret = 1 [
    (google.api.field_behavior) = OUTPUT_ONLY,
    (bytebase.v1.audit_behavior) = SENSITIVE
  ];
  string provisioning_uri = 2 [
    (google.api.field_behavior) = OUTPUT_ONLY,
    (bytebase.v1.audit_behavior) = SENSITIVE
  ];
  repeated string recovery_codes = 3 [
    (google.api.field_behavior) = OUTPUT_ONLY,
    (bytebase.v1.audit_behavior) = SENSITIVE
  ];
  google.protobuf.Timestamp expire_time = 4 [(google.api.field_behavior) = OUTPUT_ONLY];
}

message EnableMfaRequest {
  string name = 1 [
    (google.api.field_behavior) = REQUIRED,
    (google.api.resource_reference) = {type: "bytebase.com/User"}
  ];

  // The code computed from the enrollment StartMfaEnrollment returned.
  string otp_code = 2 [
    (google.api.field_behavior) = REQUIRED,
    (bytebase.v1.audit_behavior) = SENSITIVE,
    (buf.validate.field).string.max_len = 64
  ];

  // Proof for the *existing* factor being replaced, if any. Not the code
  // above — that proves the new enrollment, this proves the caller still
  // owns the account before the swap.
  CredentialProof credential = 3 [(google.api.field_behavior) = REQUIRED];
}

message DisableMfaRequest {
  string name = 1 [
    (google.api.field_behavior) = REQUIRED,
    (google.api.resource_reference) = {type: "bytebase.com/User"}
  ];
  CredentialProof credential = 2 [(google.api.field_behavior) = REQUIRED];
}

message RegenerateRecoveryCodesRequest {
  string name = 1 [
    (google.api.field_behavior) = REQUIRED,
    (google.api.resource_reference) = {type: "bytebase.com/User"}
  ];
  CredentialProof credential = 2 [(google.api.field_behavior) = REQUIRED];
}

message RegenerateRecoveryCodesResponse {
  repeated string recovery_codes = 1 [
    (google.api.field_behavior) = OUTPUT_ONLY,
    (bytebase.v1.audit_behavior) = SENSITIVE
  ];
}
```

`current_password`/`new_password` are bounded at `max_bytes = 72` — bcrypt's real limit, matching
`User.password` — not `LoginRequest.password`'s `max_bytes = 512`, which is wider only because `Login`
also serves LDAP bind; nothing reachable from these five methods ever touches an LDAP-bound principal.
`otp_code`/`recovery_code` take `max_len = 64`, matching `LoginRequest`'s existing fields of the same
name.

`RegenerateRecoveryCodes` collapses today's mint-then-promote pair into one call: recovery codes have
no client-side proof step the way a TOTP code does (nothing to "verify you can compute"), so the
two-step shape in the current code was only ever an accident of sharing plumbing with TOTP enrollment.

### Existing messages

```protobuf
message UpdateUserRequest {
  reserved 3, 4, 5; // otp_code, regenerate_temp_mfa_secret, regenerate_recovery_codes — moved above.
  User user = 1 [(google.api.field_behavior) = REQUIRED];
  google.protobuf.FieldMask update_mask = 2;
  bool allow_missing = 6;
  // `password` in update_mask is now admin-assisted only (callerUser.ID != user.ID);
  // a self-service caller is rejected and pointed at ChangePassword.
}
```

`User.mfa_enabled` stays a bool — just gains `OUTPUT_ONLY`, since nothing sets it through `UpdateUser`
anymore. That's an annotation, not a wire change, so the field keeps its number; no new field, no
reservation needed for it. (An earlier pass here added a nested `MfaStatus` with a
`recovery_codes_remaining` count, reasoning that it would help a client prompt for regeneration before
a user runs out. Dropped: nothing in T10 needs it, no frontend surface consumes it today, and a
message wrapping one field is the same speculative-abstraction mistake at a different scale. Revisit
if a real reader shows up for it.)

```protobuf
message User {
  reserved 5, 7, 9, 10, 11, 15;
  // 9, 10, 11: temp_otp_secret / temp_recovery_codes / temp_otp_secret_created_time
  //    — superseded by MfaEnrollment, which is where that data actually belongs.

  // ...name, state, email, title, password, phone, profile, groups, workspace unchanged...

  // The mfa_enabled flag means if the user has enabled MFA.
  bool mfa_enabled = 8 [(google.api.field_behavior) = OUTPUT_ONLY];
}
```

### Naming

`StartMfaEnrollment` is the only method that produces an `MfaEnrollment`, so it's the only one named
after that object — once confirmed, there's no standing "enrollment" left to refer to, only the
account's live MFA state, which `EnableMfa`/`DisableMfa` name directly as a matched pair, matching the
existing `handleEnable2FA`/`handleDisable2FA` vocabulary in `AccountSettingsPage.tsx`. `ChangePassword`
and `RegenerateRecoveryCodes` need no qualifier — recovery codes only ever mean the MFA kind, matching
today's `regenerate_recovery_codes` field name and `RegenerateRecoveryCodesView` component.

## Alternatives

- **Password only, no `CredentialProof` alternative** — the doc's own starting point, for all four
  methods and then, briefly, for `EnableMfa` alone even after the others got `CredentialProof`.
  Dropped on both counts: it forces an MFA-enabled user to produce the one factor they might
  legitimately be rotating away from (a suspected-leaked password), while the bookkeeping to accept
  their second factor instead (T9's `MFA` lockout kind, `challengeMFACode`/`challengeRecoveryCode`)
  already exists — and for `EnableMfa` specifically, restricting to password broke Cloud entirely,
  since a Cloud account has no password to supply (see Cloud vs. self-hosted).
- **Boolean flags on `UpdateUser`, gated by a single check on the built patch** (this doc's own first
  draft). Smaller diff, but keeps `mfa_enabled` on status-and-trigger double duty and the scratch
  fields on `User`. Superseded once every comparable product turned out to reject this shape.
- **Time-boxed elevated session** (GitHub sudo mode: 2h, resets per action; Google Cloud console:
  15min). Real session-state infrastructure for a handful of rarely-touched screens, and the
  elevated flag would live in the same access token T10 says an attacker already holds. Revisit if
  the number of sensitive self-service endpoints grows enough to justify a shared mechanism.
- **RFC 9470 OAuth step-up** (`401` + `insufficient_user_authentication`, `max_age`). The formal
  version of the above — same objection, plus it needs an `auth_time` claim Bytebase's access token
  doesn't carry (only a per-refresh `IssuedAt`, unrelated to when the user last typed a credential).
- **Grace window off token freshness.** Rejected outright: `IssuedAt` resets on ordinary hourly
  refresh, so this would wave a stolen, recently-refreshed token straight through.
- **Overload the existing `otp_code` field for re-authentication.** Two different secrets
  (`TempOtpSecret` for enrollment vs. the account's real credential) behind one field name is the
  ambiguity this bug already came from.
- **Keep password on `UpdateUser` with a `current_password` field**, as in this doc's own earlier
  draft. Once the proof became a three-way `CredentialProof` rather than one string, it stopped
  reading as "one more field on a PATCH" and started reading as exactly the kind of dedicated,
  confirmed action `UpdateEmail` already models for a less sensitive field.
- **A general `users/{email}/mfaFactors/{id}` collection.** Correct once Bytebase supports more than
  one factor type; not yet.

## Reference

| Element | Same as |
|---|---|
| MFA as its own resource/methods, not flags on the account's update call | [Okta Factors](https://developer.okta.com/docs/reference/api/factors/), [Auth0 multifactor delete](https://auth0.com/docs/api/management/v2/users/delete-multifactor-by-provider), [AWS IAM MFA devices](https://docs.aws.amazon.com/IAM/latest/APIReference/API_CreateVirtualMFADevice.html), [WorkOS MFA](https://workos.com/docs/mfa), [Keycloak credentials](https://www.keycloak.org/docs-api/latest/rest-api/index.html) |
| Start / finalize / withdraw enrollment lifecycle, proto-first API (closest architectural analog) | [Google Identity Platform `mfaEnrollment`](https://docs.cloud.google.com/identity-platform/docs/reference/rest/v2/accounts.mfaEnrollment/start) |
| Distinct RPCs per MFA transition, gRPC-first API (other close analog, already cited in this repo for its lockout model) | Zitadel `AddTOTP` / `VerifyTOTP` / `RemoveTOTP` |
| Mint step needs no re-auth; promotion does | AWS `CreateVirtualMFADevice` (no re-auth) → `EnableMFADevice` (two consecutive codes) |
| Password *or* a live second factor accepted as interchangeable proof | GitHub sudo mode confirms with password, security key, or TOTP interchangeably |
| Current password required to change your own password | [GitLab](https://docs.gitlab.com/user/profile/user_passwords.html), OWASP ASVS 3.7.1 |
| Current password required to disable your own second factor | [GitLab](https://docs.gitlab.com/user/profile/account/two_factor_authentication.html) |
| Enrolling/replacing a second factor gated the same as disabling it | [Google](https://support.google.com/accounts/answer/7162782) lists "Turn on 2-Step Verification" at the same reauth tier as password change |
| Self-service credential change split into its own method from admin-assisted reset; one needs the old value, one doesn't | [AWS `ChangePassword` vs. `UpdateLoginProfile`](https://docs.aws.amazon.com/IAM/latest/APIReference/API_ChangePassword.html); this repo's own `UpdateEmail` split from `UpdateUser` |
| Re-auth attempts bounded per identity, not a free-standing oracle | [`login-attempt-lockout.md`](login-attempt-lockout.md) (T9, this repo) |
| Admin-assisted recovery exempt from proving the credential being replaced | [`split-profile-self-service-vs-admin.md`](split-profile-self-service-vs-admin.md) (this repo); GitLab, Okta, AWS IAM admin consoles all reset without the old value |
| Sensitive-attribute re-auth as a general requirement, not just password | OWASP ASVS re-auth for changes to email, phone, MFA config, or other recovery-relevant fields ([ASVS #2727](https://github.com/OWASP/ASVS/issues/2727)) |

Considered, not adopted — specifically GitHub's *time-boxed session*, not its factor-choice model
(adopted above): [GitHub sudo mode](https://docs.github.com/en/authentication/keeping-your-account-and-data-secure/sudo-mode) ·
[RFC 9470 explained](https://workos.com/blog/rfc-9470-step-up-authentication-challenge) ·
[Google Cloud console reauthentication](https://docs.cloud.google.com/docs/authentication/reauthentication) ·
[Okta account management policy](https://developer.okta.com/docs/concepts/policies/).

Related: [`v1-api-audit-2026-08.md`](v1-api-audit-2026-08.md) T10 ·
[`login-attempt-lockout.md`](login-attempt-lockout.md) (T9) ·
[`split-profile-self-service-vs-admin.md`](split-profile-self-service-vs-admin.md).

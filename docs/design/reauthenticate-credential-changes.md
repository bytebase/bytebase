# Require re-authentication to change your own credentials

Status: proposal · 2026-08-24

`UpdateUser` lets an authenticated caller rewrite their own password, TOTP secret, and recovery
codes without proving they still control the credential being replaced. Closes T10 in
[`v1-api-audit-2026-08.md`](v1-api-audit-2026-08.md) for every path `UpdateUser` and its neighbors
expose today: password and MFA lifecycle both move off `UpdateUser` onto their own methods, not just a
field addition — see [Resource design](#resource-design). One residual instance of the same threat
shape survives in OAuth-enabled workspaces — a still-valid stolen access token minting a brand-new,
indefinitely renewable OAuth grant at `/authorize` — and is carried forward as a named follow-up, not
closed here; see G7 and Non-goals.

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
  account: the current password, a live MFA code where the account has MFA, or a one-time email code
  where neither exists yet (Cloud, SSO, or a self-hosted account enrolling in MFA for the first time).
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
- **G7** All four state-changing methods revoke the account's other refresh tokens on success, same
  as password change already does, and — unlike password change today — atomically: the credential
  mutation and the revocation commit in one transaction, so a revocation failure fails the whole
  request instead of silently leaving refresh tokens live the way `UpdateUser`'s log-and-continue
  (`user_service.go:436-438`) does today. *Every* refresh token, not just web sessions — `web_refresh_token`
  and the OAuth2/MCP `oauth2_refresh_token` grants, plus outstanding `oauth2_authorization_code` rows
  that could still mint one, since G7's own claim is about what a stolen credential can do afterward,
  and an OAuth grant is exactly as persistent a credential as a web session is. Scoped precisely: this
  closes the ability to mint fresh access tokens afterward (the 7-day exposure window for web, 30 days
  for OAuth), not the access tokens already issued — those are self-contained JWTs `APIAuthInterceptor`
  accepts until they expire regardless of refresh-token state, so a stolen one already in use keeps
  working for up to `access_token_duration` (default 1h) after any of these four actions. For web
  sessions that hour is a hard ceiling: the token answers requests and then is worthless, since `Login`/
  `Refresh` both require the credential this design just changed. It is not an equal ceiling for
  OAuth-enabled workspaces — see Verification — where the same still-valid hour can instead be spent
  minting a brand-new, independently 30-day-lived grant; that gap is carried forward as a named
  follow-up, not closed by this design (see Non-goals). The atomicity, and the revocation coverage
  across all three OAuth/web token tables, are not carried forward unchanged — both are tightened here,
  not merely copied.

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
- **A `CredentialProof` channel that doesn't depend on SMTP** — most plausibly, re-authenticating
  against the account's own upstream IdP. Without it, a self-hosted, SSO-only workspace with
  `require_2fa` on and no mail server configured still has no way for a user to complete first-time
  MFA enrollment (see API → Messages, `email_code`'s own limits). Real, not closed by this design;
  needs its own integration with whatever OIDC/SAML flow already handles login, not a reuse of
  existing infrastructure the way `email_code` is.
- **Blocking a still-valid access token from authorizing a brand-new OAuth grant.**
  `handleAuthorizePost` (`backend/api/oauth2/authorize.go:126,168-171`) accepts any unexpired, correctly
  signed access-token JWT as sufficient proof to mint a fresh `oauth2_authorization_code` — it never
  checks the account's current credential state, only that the token verifies and the user still
  exists. G7's revocation clears every *existing* OAuth grant, but a stolen access token still in its
  ≤1h window can be used once, at `/authorize`, to mint a *new* one — and because refresh-token rotation
  has no absolute cap (each successful refresh reissues 30 days out, `token.go:503-523`), that single
  use converts a bounded hour into a credential that survives indefinitely, past every revocation this
  design performs, for as long as the client keeps refreshing it. Real, and worse than the caveat this
  design otherwise inherits from password change (see G7) — but closing it means checking JWTs against
  state that changes after they're signed, which this design's `CredentialProof` mechanism doesn't
  touch and self-contained JWTs don't carry today. The two directions worth a follow-up: reject
  `/authorize` unless the presented token is fresher than the account's last credential-revocation event
  (a generation counter in the JWT claims, checked at `parseSessionClaims`), or require the same
  `CredentialProof` re-authentication this design already built, at consent time, before any new grant
  is issued — closer to RFC 9470 step-up than to anything else in this doc, and scoped to one endpoint
  rather than session infrastructure generally.

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

**Verification.** `ChangePassword`, `EnableMfa`, `DisableMfa`, and `ConfirmRecoveryCodes` each take
a `CredentialProof` — the current password, a live OTP or recovery code where the account has MFA, or
an emailed code where neither exists — and check it before touching anything. "Check, then mutate"
has to be one transaction with the account row locked (`SELECT ... FOR UPDATE`) from before the check
to after the write, per this repo's own [row-lock ordering](../../backend/store/README.md#transaction-row-lock-ordering)
convention — not a read-verify-then-later-write sequence against a row nothing is holding. Without the
lock, two concurrent calls against the *same* still-current credential (an attacker who separately
obtained it, and the legitimate owner racing to rotate away from it) can both pass verification before
either commits; whichever write lands second overwrites the first regardless of who verified when —
so the owner's own password change could be silently undone by a request the attacker verified
*before* the owner's committed, using a credential that's supposed to already be dead by the time the
attacker's write lands. The lock removes the window: only one verify-and-write per account can be in
flight at a time, so a credential that stopped being current mid-flight fails the re-check the lock
forces before the write, not after.

That lock isn't only for the four proof-consuming methods, though — [a fifth
finding](https://github.com/bytebase/bytebase/pull/21235) caught that `StartMfaEnrollment` and
`RegenerateRecoveryCodes` need it too, for a reason that has nothing to do with proof freshness.
`Store.UpdateUser` writes the entire `mfa_config` JSONB column, not individual fields
(`principal.go:439-444`), so both mint methods have to read the account's current live `OtpSecret`/
`RecoveryCodes` and copy them into their own patch just to avoid clobbering them while only meaning to
touch the temp fields (`user_service.go:462-470` today). Without a lock, that read can happen *before*
a concurrent `ConfirmRecoveryCodes` promotes a rotation, and the mint's write can still land *after* —
resurrecting the exact live codes the owner just rotated away from, no credential needed to trigger it
at all. Locking and proof are answering different questions — is this row safe to read-then-write
concurrently, versus is the caller who they claim to be — and they only happen to overlap on four of
the six methods that touch this row. `StartMfaEnrollment`/`RegenerateRecoveryCodes` need the lock for
their own read-modify-write of `mfa_config`, same discipline, zero `CredentialProof` involved.

The lockout claim itself, though, has to stay *outside* that locked transaction, not inside it —
[a third finding](https://github.com/bytebase/bytebase/pull/21235) against an earlier version of this
paragraph, which said the opposite. A failed `CredentialProof` means the handler returns an error, and
an error inside the locked transaction rolls it back — if the claim were inside too, the rollback
would erase it right along with the abandoned mutation, so every wrong guess would vanish without a
trace and G5's five-attempt bound would never actually apply. T9's own design already has the right
shape for this, matching how login itself works: claim the matching slot (`PASSWORD`, `MFA`, or
`EMAIL_CODE`) in the `login_attempt` table for the account
([`login-attempt-lockout.md`](login-attempt-lockout.md)) as its own statement, committed unconditionally,
*before* the locked transaction opens — then verify with `bcrypt.CompareHashAndPassword` or the existing
`challengeMFACode` helper, both pure comparisons with no DB write and safe to call as-is, inside the lock,
clearing the already-committed slot on success. The `email_code` case is not a pure comparison, and
[not just a lookup either](https://github.com/bytebase/bytebase/pull/21235): a `REAUTH` code is a
one-time credential the same way a recovery code is, so verifying it has to *consume* it, atomically, in
the same transaction — an unconsumed match leaves the code valid for anyone who intercepts it again
before it expires. `verifyEmailCode` (`auth_service_email_code.go:420-446`) isn't reusable as-is for
exactly that: it reads the row (`:429`) and deletes it (`:443`) as two separate statements on the store's
own connection, and discards the delete's own success (`_ = s.store.DeleteEmailVerificationCodeIfMatch`)
— nothing stops two concurrent proofs from both reading the row before either deletes it, and both being
told they matched. `CredentialProof`'s `email_code` path needs the same shape as the recovery-code fix
below: a `DELETE ... WHERE email = ? AND purpose = ? AND code_hash = ?` against the open transaction,
required to affect a row, not read-then-delete-and-ignore. `challengeRecoveryCode`
(`auth_service.go:601-619`) is not reusable as-is the same
way — [a sixth finding](https://github.com/bytebase/bytebase/pull/21235): it consumes the code via
`s.store.UpdateUser`, a standalone statement on the store's own connection, not the transaction the lock
was just acquired on. Calling it while already holding `SELECT ... FOR UPDATE` on the same principal row
has that inner update wait on a lock its own outer call is holding — a session waiting on itself,
indistinguishable from a hang until `statement_timeout` fires, not a false positive to code around.
`CredentialProof`'s recovery-code path needs its own transaction-aware compare-and-consume: match and
delete the code, and write `mfa_config`, against the already-open transaction directly, instead of
routing through the top-level helper. This isn't only a `CredentialProof` problem, though — [a ninth
finding](https://github.com/bytebase/bytebase/pull/21235) against the claim just made: `Login`'s own
recovery-code completion (`completeMFALogin` → `challengeMFAAndClear` →
`s.challengeRecoveryCode`, `auth_service_lockout.go:78-88`) is not exempt either, because `Login`
already holds this same principal lock by the time it gets there — the session-issuance-fencing fix
above put it there deliberately ("the entire authenticate-or-consume-then-issue sequence runs while
holding it"), and `authenticateLogin` → `completeMFALogin` is exactly that authentication step for an
MFA-second-step login. A recovery-code login would hang the same way a `CredentialProof` recovery-code
check would. Both call sites need the same fix: the transaction-aware compare-and-consume replaces
`challengeRecoveryCode` everywhere it would otherwise run inside this lock, not just in the four new
methods — `challengeMFACode` is still fine everywhere, since it never writes. Reusing T9's table
for all three means an attacker with a stolen session but no credential gets the same
five-guesses-per-ten-minutes bound as at login on every channel — not a fresh oracle, and no new
lockout kind to build; `RequestReauthCode`'s send side reuses the same table's existing resend
cooldown, so it can't be used to mail-bomb an account either. Accepting the old device's code (or a
recovery code) as an alternative to password on `EnableMfa` matters most exactly when replacing a lost
device; `email_code` covers the case neither of those helps — first-time enrollment, where nothing
prior exists to prove control of at all (see Cloud vs. self-hosted). Because each method requires its
proof in its own request message, whether it needs re-authentication is visible in its schema —
nothing to audit across a shared patch. On success, all four state-changing methods call
`DeleteWebRefreshTokensByUser` in the same transaction as the credential mutation itself — today only
`ChangePassword` inherits a call to it at all, from `UpdateUser`'s existing password branch
(`user_service.go:436`), and even there it's best-effort (`:437-438` logs and continues on error, the
password change commits regardless); `EnableMfa`/`DisableMfa`/`ConfirmRecoveryCodes` need the call
added, and all four need it atomic rather than copying that log-and-continue pattern (G7). Precisely:
this forces re-login, including the caller's own current session, the next time a refresh token would
have minted a new access token — it doesn't revoke an access token already in someone's hands, which
keeps working until it expires regardless (see G7).

Ordered wrong, though, as first written — [a sixth finding](https://github.com/bytebase/bytebase/pull/21235):
locking the principal row and only then deleting its `web_refresh_token` rows is parent-then-child,
backwards from this repo's mandatory rule of locking existing related rows child-to-parent
(`backend/store/README.md#transaction-row-lock-ordering`), and `web_refresh_token.user_email` is a
real foreign key to `principal.email` (`LATEST.sql:715-720`) — a `DELETE` on those rows counts as
acquiring a lock on them, same as an explicit `SELECT ... FOR UPDATE` would. Corrected order: lock the
account's *existing* `web_refresh_token` rows first (in primary-key order, if there's more than one),
then the principal row, then verify/mutate/delete within that same transaction. This is deliberately
the opposite of how session *issuance* orders its lock (principal first, since it's about to insert a
child row that doesn't exist yet — the `nextProjectID` shape, not the child-before-parent one)
— existing rows and not-yet-existing rows follow different rules in this repo's own convention, and
revocation and issuance land on opposite sides of that split for exactly that reason, not
inconsistently.

One more consequence of the same fix: before this design, nothing in this codebase ever locked
`web_refresh_token` before `principal` — this new child-before-parent ordering is the first, so it has
to be checked against every *existing* code path that touches both tables, not just the ones this doc
otherwise redesigns. `UpdateUserEmail` (`UpdateEmail`, unrelated to T10, untouched everywhere else in
this doc) runs `UPDATE principal SET email = ?` (`principal.go:496`), and `web_refresh_token.user_email`
is declared `ON UPDATE CASCADE` (`LATEST.sql:715-720`) — changing the email locks the principal first,
then cascades into locking and rewriting every `web_refresh_token` row referencing it, parent-then-child,
opposite of what revocation now does. `Store.UpdateUser` — the generic patch method all four
state-changing methods in this doc actually use — never sets `email` (`principal.go:419-450`), so
nothing else in this doc's own surface writes it.

That first pass [checked the wrong thing](https://github.com/bytebase/bytebase/pull/21235), though:
whether any other *code path* writes `principal.email`, not which *tables* `ON UPDATE CASCADE` off of
it — and there are three, not one. `grep -n "REFERENCES principal(email)" LATEST.sql` turns up
`oauth2_authorization_code.user_email` and `oauth2_refresh_token.user_email`
(`LATEST.sql:687,698`) alongside `web_refresh_token.user_email` (`:717`) — the same OAuth2
authorization-server tables `SwitchWorkspace`/token-exchange machinery uses, and the *only* three
`REFERENCES principal` anywhere in the schema (confirmed by grepping for that alone, not just the
`(email)` form, so there's no fourth relationship hiding behind a differently-named column). All three
cascade off the exact same `UPDATE principal SET email = ?`, so `UpdateUserEmail` needs to lock
existing rows in all three — `oauth2_authorization_code`, `oauth2_refresh_token`, and
`web_refresh_token` — not just the one this doc had already been discussing, each internally in
primary-key order and the three tables themselves in one fixed order, before running the `UPDATE
principal` that would otherwise cascade-lock them afterward.

Finding all three tables here turned out to matter for more than lock ordering — [G7's revocation
scope was too narrow](https://github.com/bytebase/bytebase/pull/21235): `DeleteWebRefreshTokensByUser`
only ever touches `web_refresh_token`, so a stolen OAuth/MCP grant sails through every one of these
four methods untouched. `handleRefreshTokenGrant` consumes an `oauth2_refresh_token` row and issues a
replacement valid 30 days (`backend/api/oauth2/token.go:265-310,503-523`), and an unconsumed
`oauth2_authorization_code` can still be exchanged for a fresh one — neither participates in G7's
revocation at all today, so G7's own claim ("closes the ability to mint fresh access tokens
afterward") was false for exactly the credential class this table-discovery exercise was already
looking at. No bulk per-user delete exists yet for either table — `DeleteOAuth2RefreshTokensByUser`
needs writing (the existing `DeleteOAuth2RefreshTokensByUserAndClient` is scoped to one client, not
all of them) alongside an equivalent for `oauth2_authorization_code`. Revocation now locks and deletes
existing rows in all three tables — the same set, the same fixed cross-table order, as the
`UpdateEmail` fix above, so the two don't end up disagreeing with each other the way issuance and
revocation almost did.

Both new bulk-delete methods land on tables with [no index that serves
them](https://github.com/bytebase/bytebase/pull/21235): `oauth2_authorization_code` and
`oauth2_refresh_token` carry only their primary key and an `expires_at` index (`LATEST.sql:709-710`) —
no index on `user_email` for either. `web_refresh_token`, by contrast, already has
`idx_web_refresh_token_user_email` (`LATEST.sql:721`), which is exactly why
`DeleteWebRefreshTokensByUser` was cheap to add in the first place; `DeleteOAuth2RefreshTokensByUser`
and its authorization-code equivalent would be sequential scans without the same index, on tables an
active MCP integration can populate continuously. Add `idx_oauth2_authorization_code_user_email` and
`idx_oauth2_refresh_token_user_email` alongside the two new store methods, not as a follow-up — the
methods only exist because of this design, so there's no pre-existing caller to stay compatible with.

Locking the account row for the mutation closes the credential-vs-credential race above, but not a
[separate one](https://github.com/bytebase/bytebase/pull/21235) between revocation and session
issuance, which never acquires that lock today. First found against `Refresh` specifically (it
consumes the caller's old refresh token, does its own work, and only inserts the replacement at the
end, `auth_service.go:479` through `:543`) — the same gap exists everywhere else a session gets minted:
`Login`/`Signup` (`finalizeLogin`, `auth_service.go:1036`) and `switchWorkspaceInternal` (`:1005`) call
the identical `issueSessionCookies` (`:1071`) with no lock either, and all four are one root cause, not
four separate ones.

Putting the lock only around `issueSessionCookies`'s own insert — this doc's own first attempt at this
fix — [turned out not to actually close it](https://github.com/bytebase/bytebase/pull/21235):
authentication (`Login`) and old-token consumption (`Refresh`, `switchWorkspaceInternal`) all happen
*before* `issueSessionCookies` is ever called, unlocked, so by the time the lock is finally acquired
there's nothing left for it to protect — the proof that justified issuing a session was already
established against whatever the row looked like earlier, and locking the insert doesn't retroactively
re-check it. The earlier reasoning that `Refresh` "finds its old token already gone" if a revocation
won the race was wrong for the same reason: `Refresh` consumes its own old token unconditionally, as
the first thing it does, regardless of whether a revocation ran concurrently — that consumption isn't
a signal about revocation at all. The lock has to move to *before* authentication/consumption, not just
before the insert, so the entire authenticate-or-consume-then-issue sequence runs while holding it —
matching how `nextProjectID` locks the project and requires it active *before* allocating an ID, not
after.

That "lock the principal first" framing was itself only half right, and the half that was wrong
[deadlocks against the revocation fix above](https://github.com/bytebase/bytebase/pull/21235):
`Login`/`Signup` have no existing token to consume, so locking the principal first before inserting a
brand-new child row is the correct `nextProjectID` shape. `Refresh` and `switchWorkspaceInternal`
aren't in that shape at all — both call `GetAndDeleteWebRefreshToken` (`auth_service.go:479,990`),
which deletes an *existing* `web_refresh_token` row, the same child-before-parent case revocation is.
Locking the principal before that delete, as issuance originally specified, is backwards for exactly
the reason it was backwards for revocation — and now the two disagree with each other in a way that
can deadlock: issuance holding the principal while waiting for the old-token row a concurrent
revocation is holding, revocation holding that row while waiting for the principal issuance is
holding. Split by whether there's an existing row to touch, matching revocation's own split: `Login`/
`Signup` lock the principal first (no existing child, `nextProjectID` shape); `Refresh`/
`switchWorkspaceInternal` lock their existing old-token row first, *then* the principal, consume the
token, re-authenticate against the now-current credential, and only then insert the new token (child,
then parent, then a fresh child — consistent with revocation's ordering, not opposed to it). With that
correction, whichever side — issuance or revocation — reaches the shared row first now determines the
outcome correctly without either one blocking the other indefinitely: issuance first means it
authenticates or consumes against the still-current credential, issues a session, and commits — then
revocation runs and deletes that session along with everything else, since
`DeleteWebRefreshTokensByUser` deletes whatever exists when it runs, not a snapshot from when the
request started; revocation first means it changes the credential and clears all tokens first — then
issuance acquires whatever lock it needs next and either re-authenticates against the *now-changed*
credential or finds the token it means to consume already gone, and fails closed instead of silently
minting a session that outlives the revocation meant to end it.

Holding that lock through the rest of the flow surfaces [one more standalone write in the same
sequence](https://github.com/bytebase/bytebase/pull/21235): `finalizeLogin` (shared by `Login`/`Signup`,
`auth_service.go:1044-1054`) and `switchWorkspaceInternal` (`:970-980`) both bump
`LastLoginTime`/`LastLoginWorkspace` via their own `s.store.UpdateUser` call, on `Store.UpdateUser`'s own
freshly-opened transaction (`principal.go:465`) — a second connection, not the one the surrounding lock
was just acquired on, so once that lock is held throughout authentication this call blocks on it exactly
like `challengeRecoveryCode` did. Moving it to after commit doesn't fix it either: `Store.UpdateUser`
replaces the whole `profile` column with whatever `Profile` message it's given
(`principal.go:446-451`), and both call sites build that message from `user.Profile` — the copy read
*before* the lock — carrying `LastChangePasswordTime` forward from that stale snapshot. A `ChangePassword`
that commits while this login is in flight would have its `LastChangePasswordTime` silently reverted to
the pre-lock value the moment this call lands afterward, which is exactly the field
[`email_code` eligibility](#design) is keyed on — turning a completed password change back into an
`email_code`-eligible account without ever touching `PasswordHash` again. Both writes move inside the
already-open transaction, built from the locked, re-read profile rather than the pre-lock one, so they
either land before the same commit the session insert does or lose to a concurrent credential change the
same way the rest of this sequence now does.

That same root cause [reaches OAuth token exchange too](https://github.com/bytebase/bytebase/pull/21235),
unfixed by any of the above: `handleAuthorizationCodeGrant` and `handleRefreshTokenGrant`
(`backend/api/oauth2/token.go:129-145,271-286`) each consume their existing child row — an
`oauth2_authorization_code` or `oauth2_refresh_token` — via an atomic single-use delete with no lock on
the account row at all, then call `issueTokens` (`:490-523`) afterward to insert the replacement, on the
opposite side of the same gap `Refresh`/`switchWorkspaceInternal` had before the fix above: the code or
token was already consumed, and the new one already queued for insertion, by the time a concurrent
revocation could have locked anything. `refuseIssuanceByCeiling`, called just before each consume
(`:129,271`), guards a different thing — the workspace's MCP capability toggle, not a
per-account credential race — and doesn't lock the principal row either, so it provides no fencing here.
Same fix, same shape: lock the existing `oauth2_authorization_code`/`oauth2_refresh_token` row first,
then the principal row, consume, and only then call `issueTokens`'s store writes — inside the
transaction, not around it — so a concurrent G7 revocation and a concurrent token exchange land in the
same deterministic order this doc already established for web sessions, instead of a fifth, OAuth-shaped
version of the same race surviving because the fix stopped one call site short.

Putting `issueTokens` itself inside that transaction isn't quite right either — [a seventh
finding](https://github.com/bytebase/bytebase/pull/21235): `issueTokens` (`token.go:489-535`) writes the
HTTP response via `c.JSON` as its last step, in the same function that does the DB insert. Calling the
whole function from inside the open transaction means the response goes out with the transaction still
uncommitted; if the commit then fails, the insert rolls back but the client already has an access token
response describing a refresh token that no longer exists in the database — self-contained enough to
answer requests for up to an hour on a grant the server no longer believes it issued. `issueTokens` needs
splitting: the token generation and the `CreateOAuth2RefreshToken`/`UpdateOAuth2ClientLastActiveAt`
writes stay inside the transaction; the `c.JSON` response write moves to after it commits, using the
values the in-transaction part returns rather than writing the response itself.

That same transaction also has to be ordered against a table this design hasn't touched yet — [an eighth
finding](https://github.com/bytebase/bytebase/pull/21235): `DeleteOAuth2Client`
(`oauth2_client.go:107-120`) and `DeleteExpiredOAuth2Clients` (`:124-138`) both `DELETE FROM
oauth2_client` directly, and `oauth2_authorization_code.client_id`/`oauth2_refresh_token.client_id` are
both `REFERENCES oauth2_client(client_id) ON DELETE CASCADE` — so either delete locks the client row
first and cascades into locking and deleting its authorization-code and refresh-token rows afterward,
parent-then-child. The token-exchange fix above holds an existing authorization-code or refresh-token
row first, then reaches for the client row too — `CreateOAuth2RefreshToken`'s FK reference and
`UpdateOAuth2ClientLastActiveAt`'s own update both require a lock on `oauth2_client` — child-then-parent,
the opposite order. A token exchange racing either client-deletion path can deadlock: exchange holding
the grant row waiting on the client, deletion holding the client waiting on the grant row it's about to
cascade into. `DeleteOAuth2Client`/`DeleteExpiredOAuth2Clients` predate this doc and are otherwise
untouched by it, but the new transaction above is what turns a previously-harmless ordering into a live
deadlock surface, so both need the same correction: lock every existing
`oauth2_authorization_code`/`oauth2_refresh_token` row referencing the client (in primary-key order) —
and for the bulk expiry path, every such row across every client being deleted, clients themselves also
in primary-key order — before locking and deleting the client rows, matching the child-before-parent
rule this doc already applies everywhere else.

Naming that correction "child-before-parent" is necessary but
[not sufficient on its own](https://github.com/bytebase/bytebase/pull/21235): it fixes ordering between
each child table and the client, but says nothing about the order *between* the two child tables, and
a user/client pair with both an authorization-code and a refresh-token row outstanding can still deadlock
if one transaction locks them in the opposite order from another. Revocation and the `UpdateEmail` fix
above already committed to "the three tables themselves in one fixed order" without ever pinning down
what that order is; this is where it stops being safe to leave implicit. One shared order, named once
and reused everywhere rows from more than one of these tables are locked together:
`oauth2_authorization_code`, then `oauth2_refresh_token`, then `web_refresh_token` — the order every
mixed-table mention in this doc has already been listing them in, now load-bearing rather than
incidental. G7's revocation, the `UpdateEmail` fix, `DeleteOAuth2Client`, and `DeleteExpiredOAuth2Clients`
all lock rows from more than one of these tables and must all use this exact order — not just each
internally consistent in primary-key order within its own table, but agreeing with every other path on
which table goes first. Needs the deterministic real-PostgreSQL regression test this repo's
row-lock-ordering convention requires for new multi-row coordination paths, covering both the
client-vs-grant direction above and this table-vs-table one: a token exchange started first should let a
racing client deletion proceed after it commits, a client deletion started first should let a racing
token exchange fail closed (invalid_grant, its row already gone) rather than deadlock either way, and a
revocation racing a client deletion for a user/client pair holding both an authorization-code and a
refresh-token row must resolve the same way, not deadlock on the two child tables disagreeing about which
comes first.

That fix closes the race around *exchanging* an existing grant, but not [a separate gap one step
earlier](https://github.com/bytebase/bytebase/pull/21235): `handleAuthorizePost`
(`authorize.go:103-171`) mints a brand-new `oauth2_authorization_code` — a grant that didn't exist
before the request — off nothing but `resolveConsentingUser` (`:200-228`) verifying a bearer access
token's signature and expiry and confirming the user row still exists (`:212,221-227`). No lock, because
there's no existing row to lock: this is the `nextProjectID` shape, not the child-before-parent one, and
correctly so. The problem isn't the lock ordering, it's what "the user still exists" is being asked to
stand in for — proof the account's *credentials* haven't changed, which a still-valid JWT cannot carry,
since it was signed before whatever credential change just happened and self-contained JWTs are exactly
as valid the second after a revocation as the second before. This is the mechanism behind the G7 caveat
correction above, and it's out of scope for the same reason named there (see Non-goals): closing it
means teaching `/authorize` to check a credential-generation signal that doesn't exist in a JWT today,
which is a change to token issuance across this system, not a fifth method alongside the other four.

`email_code`'s eligibility is a server-side check, not a UI affordance — otherwise an attacker with a
stolen session and separate mailbox access could use it against an *already*-MFA-protected account,
weaker than the factor actually protecting it (a
[Codex finding](https://github.com/bytebase/bytebase/pull/21235) against an earlier draft that
accepted it unconditionally on all four methods). The rule — no live `MFAConfig.OtpSecret`, and either
`Profile.LastChangePasswordTime` is unset or the workspace's `restriction.disallow_password_signin` is
set (added by the fourth finding above, once the backfill made the account-level signal alone
insufficient for pre-existing accounts) — is checked once, in `CredentialProof` verification itself,
not per-method, so it can't be forgotten on a future caller of the same helper.
`restriction.disallow_password_signin` *alone*, unconditionally, is still wrong on its own — this doc's
own earlier version, [caught by Codex](https://github.com/bytebase/bytebase/pull/21235) as too coarse: a
self-hosted workspace can allow both SSO and local password login at once, and the workspace-wide flag
can't see that one specific SSO-provisioned user in that mix still has no usable password. The `or`
clause only adds the flag back in as an alternative path to eligibility, not a replacement for the
account-level check, and only fires when the flag is affirmatively true — a state where, unlike the
mixed case Codex caught, every account in the workspace genuinely is passwordless. The account-level
half leans entirely on `LastChangePasswordTime` meaning what it says, which took a second
[Codex finding](https://github.com/bytebase/bytebase/pull/21235) to get right: `CreateUser` and
`Signup` both construct a brand-new local account with an empty `Profile{}` even when the caller
supplies a real, self-chosen password (`auth_service.go:344-349`, `user_service.go:263-268`) — so
every local account starts out looking exactly like a passwordless one until its first subsequent
password change, which is backwards from what this rule needs. Both must set
`Profile.LastChangePasswordTime` at creation whenever the password came from the caller, not the
server — leaving it unset only for the genuinely passwordless creation paths, SSO auto-provisioning
(`auth_service_idp.go:119-127`) and Cloud email-code auto-provisioning
(`auth_service_email_code.go:252-258`), both of which generate the password themselves and never hand
it to anyone. `ResetPassword` never had this problem: it only ever touches the password, MFA stays
live regardless of how it's
reached.

Setting it going forward isn't enough on its own —
[a third finding](https://github.com/bytebase/bytebase/pull/21235): every local account that already
exists at rollout, including ones with a real password its owner has simply never changed since
signup, still has `LastChangePasswordTime` unset today, and this design would newly classify every one
of them as eligible for `email_code` alongside the accounts that actually are passwordless. This needs
a migration, not just a forward fix: backfill `LastChangePasswordTime` to a non-null value (the row's
own `created_at` is enough) for every existing `END_USER` row that doesn't already have one set. This
is deliberately the conservative direction — it can't distinguish, for any given pre-existing row,
"real password never changed" from "SSO/Cloud placeholder never disclosed," so it treats every unknown
row as *has* a real password. The alternative default — treat unknowns as passwordless — is the one
that actually matters here: it would reopen the downgrade this whole mechanism exists to prevent for
every real password that predates the migration.

Left there, the backfill is one-way and permanent for exactly the population `email_code` exists to
serve — [a fourth finding](https://github.com/bytebase/bytebase/pull/21235): once
`LastChangePasswordTime` is non-null, nothing ever unsets it again, and a pre-existing Cloud/SSO account
has no path back to eligibility — `ChangePassword` itself is rejected on Cloud (see Cloud
vs. self-hosted), so "until they change something that legitimately sets the field" was never an escape
for exactly the accounts most likely to need one. Combined with `require_2fa`, an existing passwordless
account with no MFA yet is redirected into an enrollment it now has no self-service way to complete. The
eligibility rule needs a second, durable clause, not just the account-level one: `LastChangePasswordTime`
unset **or** the workspace's `restriction.disallow_password_signin` is set. That flag was rejected
earlier in this doc as an eligibility signal on its own — correctly, since a self-hosted workspace that
allows *both* SSO and local password login can't use a workspace-wide flag to identify which specific
users are the passwordless ones. This is different: it only fires when the flag is affirmatively true,
which means the *entire* workspace has no local passwords — Cloud, unconditionally (SaaS forces the flag
on), and any self-hosted workspace deliberately configured SSO-only. In both cases every account in the
workspace genuinely is passwordless, so there's no mixed population left to misidentify, and the clause
restores eligibility for exactly the accounts the migration would otherwise lock out forever. The
residual gap narrows to pre-existing passwordless accounts in a *mixed* self-hosted workspace (SSO and
password both allowed, flag off) — those still have no self-service path back, but unlike Cloud they at
least have G6's admin-assisted reset as a recovery route, which Cloud has no equivalent of. That
narrower residual is the wider version of the already-accepted SMTP-less gap (see Non-goals), not a new
kind of gap.

Three checks `UpdateUser` enforces today have to move with their fields, not just the credential
proof — existing behavior this design must carry forward, not decisions it makes new. `ChangePassword`
still calls `s.validatePassword` (`user_service.go:282-295`) on `new_password` before hashing — the
proof gates *who* can set it, not *what's* an acceptable value, and workspace password-complexity
policy is orthogonal to this doc's threat model. It also still rejects `new_password` matching the
*current* password (`bcrypt.CompareHashAndPassword`, `user_service.go:423-426`) — not just a UX nicety:
`Store.UpdateUser` bumps `Profile.LastChangePasswordTime` on any write to `PasswordHash`
(`principal.go:429-433`), whether or not the resulting hash represents an actual change, so skipping
this check would let a workspace's password-rotation deadline be reset by "changing" to the same
password. `DisableMfa` still checks `Require_2Fa` (`user_service.go:399-408`): a non-admin caller,
self-service or admin-assisted, cannot turn MFA off for anyone while the workspace requires it — only
a workspace admin can, unchanged from today.

Sharing T9's lockout buckets has one accepted side effect worth naming, not just inheriting silently:
a `PASSWORD` or `MFA` claim from a failed `CredentialProof` counts against the same budget a login
attempt would. A user who fumbles their password a few times inside `ChangePassword` can lock
themselves out of *logging in* too, not just out of changing it — a narrower version of the
lockout-as-availability-cost trade [`login-attempt-lockout.md`](login-attempt-lockout.md) already
accepted for a different reason. Anyone locked out this way still has the request's own error message
telling them when it clears.

**What's exempt.** `title`/`phone` on `UpdateUser` aren't authentication material.
`StartMfaEnrollment` mints a secret but changes nothing live, so it needs no proof — same as AWS's
`CreateVirtualMFADevice` needing none while `EnableMFADevice` does. `RequestReauthCode` needs no proof
either, for a different reason: it's what *produces* a proof, not a consumer of one. It's still
bounded — by the resend cooldown on the send side, by the `EMAIL_CODE` lockout on redeeming it — same
split `RequestPasswordReset` already has between sending and verifying. `UpdateUser`'s `password` field
becomes admin-only: a self-service call (`callerUser.ID == user.ID`) that sets it is rejected,
pointing at `ChangePassword`. `ChangePassword` has no admin path at all (see its `name` field comment
in API). Admin-assisted resets of another user's password or MFA stay on `UpdateUser`'s `password`
field and `DisableMfa` respectively, both exempt: the `bb.users.update` permission check and audit log are the
correct control there, and an admin recovering a locked-out user cannot know the credential being
replaced. Both also carry forward the *other* half of `UpdateUser`'s existing cross-user gate, not just
the permission check —
[worth stating explicitly](https://github.com/bytebase/bytebase/pull/21235), since it's easy to
carry over only the part that reads as "the interesting check": `callerUser.ID != user.ID` is rejected
outright in SaaS mode, unconditionally, checked *before* `bb.users.update` is even evaluated
(`user_service.go:347-359`) — "the principal is global," per the comment already on that check today.
`DisableMfa`'s admin path is self-hosted only for the same reason `UpdateUser`'s already is, not a new
decision; a SaaS workspace admin gets exactly as little cross-user reach here as anywhere else.
`CreateUser`/`Signup` have no prior credential to prove. `ResetPassword` (emailed code)
already has its own proof channel — mailbox possession — including for SSO accounts whose
`PasswordHash` they never saw.

Ownership is enforced explicitly, not assumed. All seven methods are `auth_method = CUSTOM`, which
means the ACL interceptor's automatic IAM check is skipped entirely (`doIAMPermissionCheck` returns
true unconditionally for non-IAM methods, `acl.go:252-253`) — the same gap T11 in the original audit
flagged for 15 other CUSTOM RPCs whose declared `permission` is never enforced. `DisableMfa` is the
one method here with a real admin path, gated on `bb.users.update`; every other method —
`ChangePassword`, `StartMfaEnrollment`, `EnableMfa`, `RegenerateRecoveryCodes`, `ConfirmRecoveryCodes`,
`RequestReauthCode` — must independently reject any `name` other than the caller's own, since nothing
upstream of the handler does it for them. Without that check explicitly stated per method, a future
implementation reading only the proto could plausibly ship one of them missing it — silently letting
any authenticated workspace member read another account's freshly-minted MFA secret, or overwrite
their pending recovery codes.

**Cloud vs. self-hosted.** Cloud users never know their own local password — email-code signup and
SSO both assign a random, unseen one (`auth_service_email_code.go:252-258`,
`auth_service_idp.go:119-127`), and `Login` itself refuses the password path via
`restriction.disallow_password_signin`, which SaaS forces on unconditionally. `ChangePassword`
therefore rejects any call when that restriction is set, reusing the flag `Login`/`Signup` already
check rather than a separate `s.profile.SaaS` branch — the same behavior falls out for a self-hosted
deployment that goes SSO-only. This also closes a gap in today's code: `UpdateUser`'s password branch
has no SaaS check at all, harmless only because `Login` independently blocks the result. MFA is
unaffected either way — it's a second factor on top of whichever primary method a workspace allows, so
`Require_2Fa` and all four MFA methods behave identically on both deployments. `EnableMfa` needing
`CredentialProof` rather than password-only handles *replacing* a factor on a Cloud account, but not
enrolling the *first* one: a Cloud or SSO account has no password, and first-time enrollment by
definition has no existing OTP or recovery code either, so all three `CredentialProof` options are
simultaneously unsatisfiable — exactly the account `guard.ts:265-275` hard-redirects into MFA setup
with no escape route when the workspace requires it. `CredentialProof` gains a fourth option,
`email_code`, for this: a one-time code to the account's own registered email via a new
`RequestReauthCode`, reusing the `email_verification_code` table T9 already built for `LOGIN` and
`PASSWORD_RESET` — a third purpose, not new infrastructure.

**Frontend.** The four call sites in `AccountSettingsPage.tsx`, `TwoFactorSetupPage`, and
`RegenerateRecoveryCodesView` move to their matching new method and gain a credential-proof field —
no new dialogs, since [`split-profile-self-service-vs-admin.md`](split-profile-self-service-vs-admin.md)
already gives password and 2FA changes their own confirmations.
`RegenerateRecoveryCodesView`'s existing shape barely changes: it already calls one RPC on mount
(mint) and a second when the user confirms they've downloaded the codes (promote) — `credential` just
moves onto the second call (`ConfirmRecoveryCodes`), where the download-confirmation button already
lives. `TwoFactorSetupPage` gains that same second call: it already shows recovery codes and gates
advancing on a download confirmation (`:146-160,185-195,210-213`) — that confirmation now drives a real
`ConfirmRecoveryCodes` call instead of being purely client-side, since `EnableMfa` no longer promotes
recovery codes itself. That field needs a way to reach for
`email_code` when the account has no password and no MFA yet — a "email me a code" link inline with the
proof field, calling `RequestReauthCode`, rather than a fifth dialog. This is a breaking proto change;
frontend and backend ship as one rollout on both Cloud and self-hosted (a single `go:embed`'d image,
confirmed against `scripts/Dockerfile` and the Cloud deploy workflow — no separate frontend pipeline),
same as the other breaking changes already in flight this cycle (#21181, #21234). One residual edge,
inherent to any SPA rolling deploy rather than specific to this change: a browser tab already open
before the rollout is still running the old bundle and can call the old fields on `UpdateUser` against
an already-upgraded backend for the rest of that rollout window — they're reserved, so the call
silently no-ops rather than erroring. Self-correcting on the next page load; not worth engineering
around for a field that's disappearing anyway.

## API

Seven new methods on `UserService` (same service `UpdateEmail` already lives in, rather than a new
service): one per credential state transition, `RequestReauthCode` (the send side of the fourth
`CredentialProof` option), and `ConfirmRecoveryCodes` (recovery codes mint-then-confirm the same way
MFA enrollment does, for a different reason — see Alternatives). Full proto below, then what each
piece is doing.

### Service methods

```protobuf
service UserService {
  // ...existing RPCs...

  // Sends a one-time code to the caller's own registered email, usable as
  // CredentialProof.email_code. The one channel that works when neither a
  // usable password nor an existing MFA factor exists yet — Cloud, SSO, or
  // first-time MFA enrollment anywhere. name must be the caller's own — see
  // What's exempt.
  rpc RequestReauthCode(RequestReauthCodeRequest) returns (google.protobuf.Empty) {
    option (google.api.http) = {
      post: "/v1/{name=users/*}:requestReauthCode"
      body: "*"
    };
    option (google.api.method_signature) = "name";
    option (bytebase.v1.auth_method) = CUSTOM;
    option (bytebase.v1.audit) = true;
    option (bytebase.v1.mcp_method_class) = FORBIDDEN;
    option (bytebase.v1.mcp_denial_reason) = RESETS_CREDENTIAL;
  }

  // Changes the caller's own password. Rejected if the workspace disallows
  // password sign-in (Cloud always does) — see Design → Cloud vs. self-hosted.
  // new_password still runs through the workspace's password-complexity
  // policy before hashing, same as UpdateUser does today.
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

  // Mints a new TOTP secret and recovery codes, persists them as the
  // account's temporary MFA state (store-level only — see API → Messages),
  // and returns them for the caller to confirm. Not yet live, so no proof
  // is required here, but the account row is still locked for the read of
  // current live MFAConfig fields and the write that preserves them — see
  // Design → Verification. name must be the caller's own — see What's exempt.
  rpc StartMfaEnrollment(StartMfaEnrollmentRequest) returns (MfaEnrollment) {
    option (google.api.http) = {
      post: "/v1/{name=users/*}:startMfaEnrollment"
      body: "*"
    };
    option (google.api.method_signature) = "name";
    option (bytebase.v1.auth_method) = CUSTOM;
    option (bytebase.v1.audit) = true;
    option (bytebase.v1.mcp_method_class) = FORBIDDEN;
    option (bytebase.v1.mcp_denial_reason) = MINTS_CREDENTIAL;
  }

  // Validates otp_code and pending_version against the temporary state
  // StartMfaEnrollment persisted for this account (re-read from the store
  // by name, not carried in this request) and promotes *only* the secret to
  // the caller's live MFA factor, replacing any existing one — recovery
  // codes stay pending until a separate ConfirmRecoveryCodes call, same as
  // an ordinary rotation (see Design → Verification). Also rejects a
  // pending set past its expiry (isMFATempSecretExpired, checked today both
  // before OTP verification and before promotion — both carry forward,
  // inside the locked transaction) — pending_version alone only detects a
  // *replaced* pending set, not one that's simply gone stale with no
  // replacement ever minted. name must be the caller's own — no admin path,
  // same reason as ChangePassword.
  rpc EnableMfa(EnableMfaRequest) returns (User) {
    option (google.api.http) = {
      post: "/v1/{name=users/*}:enableMfa"
      body: "*"
    };
    option (google.api.method_signature) = "name,otp_code,credential,pending_version";
    option (bytebase.v1.auth_method) = CUSTOM;
    option (bytebase.v1.audit) = true;
    option (bytebase.v1.mcp_method_class) = FORBIDDEN;
    option (bytebase.v1.mcp_denial_reason) = TAKES_OVER_ACCOUNT;
  }

  // Turns MFA off for `name`. Rejected for any non-admin caller while the
  // workspace requires MFA (Require_2Fa), self-service or admin-assisted
  // alike, same as UpdateUser enforces today. The admin path itself
  // (name != caller) is self-hosted only: SaaS rejects any cross-user call
  // outright before bb.users.update is even checked, same as UpdateUser's
  // existing cross-user gate.
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

  // Mints a new set of recovery codes, persisted as the account's pending
  // set (storepb.MFAConfig.TempRecoveryCodes — reused, not new) alongside
  // the still-live old ones, and returns them, with a version token, for
  // the caller to save. Not yet live, so no proof is required here, same as
  // StartMfaEnrollment — but the account row is locked for the same reason
  // StartMfaEnrollment's is: this also reads and rewrites live MFAConfig
  // fields, not just temp ones (see Design → Verification). Requires a live
  // MFAConfig.OtpSecret already — recovery codes back up an existing
  // factor, so an account with none should enroll via StartMfaEnrollment
  // instead, same precondition the current regenerate_recovery_codes flag
  // enforces (user_service.go:475-477). name must be the caller's own — no
  // admin path.
  rpc RegenerateRecoveryCodes(RegenerateRecoveryCodesRequest) returns (RegenerateRecoveryCodesResponse) {
    option (google.api.http) = {
      post: "/v1/{name=users/*}:regenerateRecoveryCodes"
      body: "*"
    };
    option (google.api.method_signature) = "name";
    option (bytebase.v1.auth_method) = CUSTOM;
    option (bytebase.v1.audit) = true;
    option (bytebase.v1.mcp_method_class) = FORBIDDEN;
    option (bytebase.v1.mcp_denial_reason) = MINTS_CREDENTIAL;
  }

  // Promotes the pending set RegenerateRecoveryCodes minted to live,
  // invalidating the old set — but only the *exact* pending set named by
  // pending_version; a mismatch (superseded by a later RegenerateRecoveryCodes
  // call, own or someone else's) is rejected rather than silently promoting
  // whatever is currently pending. Also re-checks MFAConfig.OtpSecret is
  // still live, inside the same locked transaction — pending_version alone
  // doesn't catch DisableMfa running in between, since disabling clears
  // MFAConfig entirely without touching the recovery-code temp state or its
  // version, so a stale confirmation after MFA was turned off must still
  // fail rather than write live recovery codes onto an account with no
  // factor left for them to back up. The caller's acknowledgment that they
  // saved the new codes, not a proof step on its own — but promotion is
  // exactly the moment the old codes stop working, so it's gated like every
  // other promotion in this design. name must be the caller's own.
  rpc ConfirmRecoveryCodes(ConfirmRecoveryCodesRequest) returns (User) {
    option (google.api.http) = {
      post: "/v1/{name=users/*}:confirmRecoveryCodes"
      body: "*"
    };
    option (google.api.method_signature) = "name,credential,pending_version";
    option (bytebase.v1.auth_method) = CUSTOM;
    option (bytebase.v1.audit) = true;
    option (bytebase.v1.mcp_method_class) = FORBIDDEN;
    option (bytebase.v1.mcp_denial_reason) = TAKES_OVER_ACCOUNT;
  }
}
```

`mcp_denial_reason` isn't the same value across all seven methods, even though all seven are
`FORBIDDEN`: the enum distinguishes "rewrites credentials, response carries none"
(`TAKES_OVER_ACCOUNT`), "puts a usable secret directly in the response body" (`MINTS_CREDENTIAL`), and
"drives the out-of-band flow that delivers the secret a login accepts" (`RESETS_CREDENTIAL`).
`StartMfaEnrollment` returns a fresh TOTP secret; `RegenerateRecoveryCodes` returns fresh codes that
aren't live yet — both put a secret in the response body before anything is decided, so both get
`MINTS_CREDENTIAL`. `ChangePassword`/`EnableMfa`/`DisableMfa`/`ConfirmRecoveryCodes` return only
`User`, so they get `TAKES_OVER_ACCOUNT`, matching what `UpdateUser` already carries today for the
same operations. `RequestReauthCode` returns nothing but *sends* a credential-proving secret out of
band — the same shape `RequestPasswordReset` already carries this reason for — so it gets
`RESETS_CREDENTIAL`, the one value none of the other six use.

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

    // A one-time code from RequestReauthCode, sent to the account's own
    // registered email. Valid only when this account has no live MFA factor,
    // AND either its own Profile.LastChangePasswordTime is unset (no password
    // was ever legitimately written for it — see Design → Cloud vs.
    // self-hosted, and note CreateUser/Signup must set this field at creation
    // for it to mean that) OR the workspace's restriction.disallow_password_signin
    // is set — bootstrap proof for an account with nothing else yet, never a
    // substitute for a factor that already exists. The workspace-wide flag is
    // an alternative path to eligibility, not a replacement for the
    // account-level check: on its own, an unconditional workspace-wide flag is
    // too coarse (a self-hosted workspace can run SSO and local password login
    // side by side, and an individual SSO-provisioned user in that mix still
    // has no usable password even though the workspace itself allows one) —
    // but when the flag is affirmatively set, the whole workspace has no local
    // passwords, so that coarseness doesn't apply, and the flag is what
    // restores eligibility for pre-existing Cloud/SSO accounts the
    // LastChangePasswordTime backfill migration would otherwise lock out
    // permanently (see Design → Verification). The handler checks this
    // eligibility itself; it is not just a
    // frontend choice of which field to show. The condition is never true
    // for DisableMfa or ConfirmRecoveryCodes (both imply live MFA), so
    // email_code is only ever actually usable on ChangePassword or
    // first-time EnableMfa.
    string email_code = 4 [
      (bytebase.v1.audit_behavior) = SENSITIVE,
      (buf.validate.field).string.max_len = 64
    ];
  }
}

message RequestReauthCodeRequest {
  // Format: users/{email}. Must be the caller's own name — the code is
  // delivered to that account's own registered email, so requesting one for
  // someone else would only prove the target can read their own mail, not
  // anything about the caller.
  string name = 1 [
    (google.api.field_behavior) = REQUIRED,
    (google.api.resource_reference) = {type: "bytebase.com/User"}
  ];
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

// The API-facing shape of an in-progress enrollment. What it's NOT is a claim
// about storage: EnableMfa still needs something server-side to validate
// otp_code against between the two calls, and that's the same store-level
// storepb.MFAConfig.TempOtpSecret / TempRecoveryCodes / TempOtpSecretCreatedTime
// StartMfaEnrollment already writes today (user_service.go's
// regenerate_temp_mfa_secret branch) — untouched by this design, a different
// proto package from v1pb.User, never itself exposed over the API. What
// changes is only the *API-facing* surface: those three fields disappear from
// User (see Existing messages) and reappear here, returned once by
// StartMfaEnrollment and never carried back into EnableMfaRequest — EnableMfa
// re-reads them from the account's store row instead.
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

  // Same shared temp-state version RegenerateRecoveryCodesResponse.pending_version
  // identifies (see its comment) — StartMfaEnrollment and RegenerateRecoveryCodes
  // both mint into the same account's temp MFA state, so they share one version.
  // Echo to EnableMfa.
  google.protobuf.Timestamp pending_version = 5 [(google.api.field_behavior) = OUTPUT_ONLY];
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

  // Echoed from MfaEnrollment.pending_version. otp_code alone only proves
  // the caller knows the current TempOtpSecret; it says nothing about
  // whether TempRecoveryCodes — minted alongside it, promoted alongside it
  // — has since been overwritten by an intervening RegenerateRecoveryCodes
  // call. A mismatch here is rejected for the same reason a mismatched
  // ConfirmRecoveryCodesRequest.pending_version is.
  google.protobuf.Timestamp pending_version = 4 [(google.api.field_behavior) = REQUIRED];
}

message DisableMfaRequest {
  string name = 1 [
    (google.api.field_behavior) = REQUIRED,
    (google.api.resource_reference) = {type: "bytebase.com/User"}
  ];

  // Required only when name is the caller's own. On an admin-assisted call
  // (name is someone else's), unset and unchecked — this is the one method
  // in this file with a real admin path, so its requirement is conditional
  // on caller identity, which a blanket proto REQUIRED can't express; the
  // conditionality is enforced in the handler, same as every other
  // identity-dependent check in this design.
  CredentialProof credential = 2;
}

message RegenerateRecoveryCodesRequest {
  string name = 1 [
    (google.api.field_behavior) = REQUIRED,
    (google.api.resource_reference) = {type: "bytebase.com/User"}
  ];
}

message ConfirmRecoveryCodesRequest {
  string name = 1 [
    (google.api.field_behavior) = REQUIRED,
    (google.api.resource_reference) = {type: "bytebase.com/User"}
  ];
  CredentialProof credential = 2 [(google.api.field_behavior) = REQUIRED];

  // Echoed from RegenerateRecoveryCodesResponse.pending_version. Must match
  // the account's current pending set exactly, or this is rejected — a
  // mismatch means a later RegenerateRecoveryCodes call (the caller's own,
  // in a second tab, or someone else's) has already superseded the set this
  // request thinks it's confirming.
  google.protobuf.Timestamp pending_version = 3 [(google.api.field_behavior) = REQUIRED];
}

message RegenerateRecoveryCodesResponse {
  repeated string recovery_codes = 1 [
    (google.api.field_behavior) = OUTPUT_ONLY,
    (bytebase.v1.audit_behavior) = SENSITIVE
  ];

  // Identifies the account's current temp-MFA-state generation — the
  // store's own mint timestamp, not a newly invented token, shared with
  // MfaEnrollment.pending_version because StartMfaEnrollment and this
  // method mint into the same underlying temp state. Pass back to
  // ConfirmRecoveryCodes.
  google.protobuf.Timestamp pending_version = 2 [(google.api.field_behavior) = OUTPUT_ONLY];
}
```

`current_password`/`new_password` are bounded at `max_bytes = 72` — bcrypt's real limit, matching
`User.password` — not `LoginRequest.password`'s `max_bytes = 512`, which is wider only because `Login`
also serves LDAP bind; nothing reachable from these six methods ever touches an LDAP-bound principal.
`otp_code`/`recovery_code`/`email_code` take `max_len = 64`, matching `LoginRequest`'s existing fields
of the same name.

`RegenerateRecoveryCodes`/`ConfirmRecoveryCodes` keep today's mint-then-promote shape — an earlier
draft of this doc collapsed it into one call, reasoning that recovery codes have no client-side proof
step the way a TOTP code does (nothing to "verify you can compute"), so the two-step shape looked like
an accident of sharing plumbing with TOTP enrollment. Wrong, per a
[Codex finding](https://github.com/bytebase/bytebase/pull/21235): the two steps aren't about proving
computation, they're about confirming *receipt* — `RegenerateRecoveryCodesView.tsx:60-75` already
disables the promote button until the user reports downloading the codes, precisely so a lost
response, a closed tab, or a crash between mint and promote can't leave someone with live MFA and zero
saved recovery codes. Splitting the RPC preserves that property structurally: the old codes stay valid
through `RegenerateRecoveryCodes` (mint) and only stop working once `ConfirmRecoveryCodes` runs, so a
client that never gets a chance to call `ConfirmRecoveryCodes` — for any reason — has lost nothing.

That split alone still had a hole, caught the same review pass: since `RegenerateRecoveryCodes` needs
no proof (matching `StartMfaEnrollment`), and `TempRecoveryCodes` is one mutable slot per account, a
stolen-session caller with no credential could mint *after* the real owner, overwriting the pending
set with one only the attacker knows — the owner's own, correctly-proofed `ConfirmRecoveryCodes` call
would then promote the attacker's set, not theirs, without the attacker ever needing to produce a
credential at all. TOTP enrollment doesn't have this hole: `EnableMfa`'s `otp_code` is itself proof of
which secret the caller actually has, so an overwritten `TempOtpSecret` just makes the legitimate
call fail closed (wrong code) rather than silently promote the wrong thing. Recovery codes have
nothing equivalent to check computation against, so `pending_version` — the mint's own timestamp,
echoed back and matched exactly — plays that role instead: it doesn't prove the caller's identity
(`CredentialProof` still does that), it proves the caller is confirming the *specific* set they
actually received, so a set superseded by any later mint is rejected rather than silently promoted.

That "fails closed" claim about `otp_code` turned out to only cover half of what `EnableMfa` used to
promote — [caught in the same review round](https://github.com/bytebase/bytebase/pull/21235):
`StartMfaEnrollment` mints `TempOtpSecret` *and* `TempRecoveryCodes` together, and `EnableMfa`
originally promoted both together in one call, but only `otp_code` was bound to anything. A
stolen-session caller with no credential could call the still-proof-free `RegenerateRecoveryCodes`
mid-enrollment, overwriting `TempRecoveryCodes` with a set only they know, while `TempOtpSecret` stays
untouched — so the real owner's `otp_code` still validates fine, `EnableMfa` still succeeds, and it
would have promoted the attacker's recovery codes right alongside the owner's legitimate TOTP secret.

A [later finding](https://github.com/bytebase/bytebase/pull/21235) made the fix simpler than patching
that race in place: `EnableMfa` shouldn't have been promoting recovery codes at all. The existing
enrollment UI validates the OTP, shows the recovery codes, and only promotes them after the user
confirms downloading — the exact same receipt-confirmation shape `RegenerateRecoveryCodesView.tsx`
already has for rotation, which is what motivated `ConfirmRecoveryCodes`'s own `pending_version` gate
above. `EnableMfa` combining OTP validation with recovery-code promotion in one atomic call had the
identical lost-response problem: if the response never arrives or the tab closes right after, the
account has live MFA and recovery codes the owner never actually saw, no different from the rotation
bug already fixed. So `EnableMfa` now promotes *only* the secret; the recovery codes it minted
alongside stay pending until a separate `ConfirmRecoveryCodes{name, pending_version}` call — reusing
the method already built for rotation rather than inventing a second confirmation mechanism, since
enrollment and rotation turn out to be the same "confirm receipt before the old thing goes away"
shape once the OTP-specific part is factored out. (For enrollment there's no "old thing" — the check
`ConfirmRecoveryCodes` already has, requiring a live `OtpSecret`, is satisfied by the `EnableMfa` call
that necessarily already ran.) `EnableMfaRequest.pending_version` still guards `StartMfaEnrollment`
against being silently superseded before `EnableMfa` runs — general temp-state freshness, not
specifically a recovery-codes protection anymore, since `EnableMfa` no longer touches them at all.

`email_code` verifies against a new `REAUTH` purpose on `email_verification_code`
(`storepb.EmailVerificationCodePurpose`), alongside the existing `LOGIN` and `PASSWORD_RESET` —
that table was already built to hold more than one purpose, so this is a third row shape, not new
storage. Without it, `EnableMfaRequest.credential` had exactly the failure mode
[a Codex review on this doc's own PR](https://github.com/bytebase/bytebase/pull/21235) caught: a Cloud
or SSO account has no password, and first-time enrollment has no existing OTP or recovery code either,
so all three original options were simultaneously unsatisfiable for precisely the accounts
`guard.ts:265-275` force-redirects into MFA setup under a `require_2fa` policy — a hard lockout, not
an edge case. `email_code` closes it for every Cloud account, and for any self-hosted account with no
password of its own — including a mixed-mode SSO user in a workspace that still allows local password
login generally — with mail configured, by giving them a proof they can always produce.

It doesn't close it for every account, though — a second
[Codex finding](https://github.com/bytebase/bytebase/pull/21235) on the same review caught the
remainder: `resolvePreLoginEmailSetting` returns `nil` with no workspace `EMAIL` setting and no
`EMAIL_CONFIG` env var (`auth_service_email_code.go:319-342`), and `sendEmailVerificationCode` then
fails outright rather than degrading (`:366-374`) — so a self-hosted, SSO-only workspace with
`require_2fa` on and no SMTP configured leaves an enrolling user with no password, no existing MFA,
*and* no deliverable code. Narrower than the original gap (self-hosted, SSO-only, no SMTP, all
required together) but real, and not something this design closes. Two things worth doing regardless
of that gap, both applied here: `RequestReauthCode` surfaces the send failure rather than swallowing
it like `ResetPassword` does — there's no existence-oracle reason to hide it, since the caller is
already authenticated as the only account it could possibly be about — so the failure is at least
diagnosable ("no email configured, contact your workspace admin") instead of a silent dead end. The
full fix — a proof channel that doesn't depend on SMTP, most plausibly re-authenticating against the
account's own upstream IdP for SSO deployments — is real scope beyond this doc: it means integrating
with whatever OIDC/SAML flow already exists for login, not reusing something already built the way
`email_code` reuses T9's table. Tracked as a follow-up, not silently left for someone to discover
later.

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
  //    — API-facing removal only; the underlying storepb.MFAConfig columns
  //    these were read from are untouched (see MfaEnrollment's comment).
  //    Enrollment now returns this data via MfaEnrollment instead of here.

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
`ConfirmRecoveryCodes` isn't `EnableRecoveryCodes` — recovery codes aren't a feature with an on/off
state the way MFA is, they're always live once any exist, so "confirm" names what the caller is
actually doing (acknowledging receipt), not a state transition that doesn't exist.

## Alternatives

- **Collapse recovery-code regeneration into one call** (this doc's own earlier draft) rather than
  the `RegenerateRecoveryCodes`/`ConfirmRecoveryCodes` pair. Reasoned that recovery codes have no
  client-side proof step the way a TOTP code does, so the existing two-step shape looked accidental.
  Wrong: the two steps aren't about proof, they're about confirming receipt before the old codes stop
  working — `RegenerateRecoveryCodesView.tsx` already disables its promote button until the user
  reports downloading the codes, and one call would have silently dropped that safety property.
- **Skip the proof entirely for first-time MFA enrollment**, on the theory that there's nothing live
  yet to take over. Rejected: an attacker with a stolen session who enrolls their own device on a
  victim's account doesn't just add a factor — once `MFAConfig.OtpSecret` is non-empty, `Login`
  challenges it on *every* future login, including the real owner's, who never set it up and can't
  produce the attacker's code. Different shape of harm than a full impersonation, same severity: the
  real owner is locked out of their own account. Caught by
  [Codex's review](https://github.com/bytebase/bytebase/pull/21235) of this doc's own PR, which is what
  `email_code` closes.
- **Reuse `ResetPassword`'s existing send/verify flow directly for MFA enrollment**, instead of a new
  `RequestReauthCode` and a `REAUTH` purpose. Rejected: `ResetPassword`'s code is scoped to setting a
  new password, not to generic identity proof, and folding a second meaning into it is the same
  field-overload mistake `otp_code` already made once in the original `UpdateUser`. A third purpose on
  the same table costs one enum value; a second meaning on an existing purpose costs a code path that
  has to explain both.
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

# Require re-authentication to change your own credentials

Status: proposal · 2026-08-24

`UpdateUser` lets an authenticated caller rewrite their own password, TOTP secret, and recovery
codes without proving they still control the credential being replaced. Closes T10 in
[`v1-api-audit-2026-08.md`](v1-api-audit-2026-08.md) for every path `UpdateUser` and its neighbors
expose today: password and MFA lifecycle both move off `UpdateUser` onto their own methods, not just a
field addition — see [Resource design](#resource-design). A stolen access token may keep answering
requests until it expires (≤1h), but this design also stops it being *spent* on a replacement: every
path that mints a new token from an existing one is bound to the account's credential generation. See
G7 and Verification → Token-minting paths.

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
- **G7** Every mutation that actually changes live authentication material revokes the account's other
  refresh tokens (see Verification for why "actually changes" excludes the no-op enrollment check), same
  as password change already does, and — unlike password change today — atomically: the credential
  mutation and the revocation commit in one transaction, so a revocation failure fails the whole
  request instead of silently leaving refresh tokens live the way `UpdateUser`'s log-and-continue
  (`user_service.go:436-438`) does today. *Every* refresh token, not just web sessions — `web_refresh_token`
  and the OAuth2/MCP `oauth2_refresh_token` grants, plus outstanding `oauth2_authorization_code` rows
  that could still mint one, since G7's own claim is about what a stolen credential can do afterward,
  and an OAuth grant is exactly as persistent a credential as a web session is. Scoped precisely: this
  closes the ability to mint fresh access tokens afterward, not the access tokens already issued — those
  are self-contained JWTs `APIAuthInterceptor` accepts until they expire regardless of refresh-token
  state, so a stolen one already in use keeps answering requests for up to the workspace-configured
  access-token lifetime (default 1h, but see Verification → Token-minting paths — it has only a
  one-minute floor and no upper bound, so this window is not a fixed ceiling). What this design *does*
  bound is the *reuse* of that token to obtain a replacement: every path that mints a new token from an
  existing one — `SwitchWorkspace` and its two `workspace_service.go` callers, the MFA temp-token
  completion (`checkMFARequired`), and OAuth `/authorize` — is bound to `LastCredentialChangeTime` and
  refuses a token stamped before the change. Without that, the window is not an exposure ceiling but a
  renewal interval (unbounded for OAuth, which re-issues a full lifetime on every refresh), and G7's own
  sentence would be false. The atomicity, and the revocation coverage
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
- ~~**Blocking a still-valid access token from authorizing a brand-new OAuth grant.**~~ *Promoted into
  scope* — see Design → Verification, "Token-minting paths." This was a Non-goal on the grounds that
  closing it required a credential-generation claim in JWTs, which nothing else in the design needed.
  Two later findings removed that reasoning: `LastCredentialChangeTime` had to exist anyway for the MFA
  temp token, and `SwitchWorkspace` turned out to mint fresh access tokens off a stale JWT with *no*
  bound at all, which is strictly worse than the `/authorize` case and not deferrable. Once the claim
  has to be in access tokens for one, checking it in the other is a line of code, and leaving either
  open would keep G7's own sentence false.

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
an emailed code where neither exists — and check it before touching anything. One constraint on that
menu is not optional, and getting it wrong reopens the exact takeover `email_code`'s eligibility rule
exists to close: **when the account has a live MFA factor, a mutation that touches that factor must be
proven with the factor, not with `current_password`.** `DisableMfa`, and `EnableMfa`/`ConfirmRecoveryCodes`
when they replace an existing factor, therefore accept only `otp_code` or `recovery_code` while
`MFAConfig.OtpSecret` is live — never `current_password` or `email_code`. The reason is that
`current_password` is not a second, independent thing the caller knows: `ResetPassword` mints it from
mailbox possession alone, leaving MFA untouched (it "only ever touches the password," see below), so an
attacker with a stolen session and the account's mailbox could reset the password, then present that
fresh password as `CredentialProof` to strip the MFA the reset left standing. That is precisely the
mailbox-plus-session downgrade `email_code` refuses by requiring no-live-MFA; allowing
`current_password` against live MFA would leave the front door open beside the locked window. Requiring
the factor itself to change the factor closes both. (`ChangePassword` is unaffected — it accepts
`current_password` with or without MFA, since changing the password is not touching the factor, and a
password change already revokes sessions so it cannot be a step toward one.) "Check, then mutate"
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
below: a `DELETE ... WHERE email = ? AND purpose = ? AND code_hash = ? AND expires_at > now()` against
the open transaction, required to affect a row, not read-then-delete-and-ignore.
The `expires_at` clause isn't optional —
[caught in a later round](https://github.com/bytebase/bytebase/pull/21235): matching only on
email/purpose/hash would let an already-expired row still consume successfully for as long as it
survives until the hourly cleanup job removes it, `verifyEmailCode` itself checks `row.ExpiresAt` before
ever deleting (`:436-438`), and dropping that check here would leave an intercepted, expired code usable
against a credential change for up to that same hour. The atomic, expiry-checked, result-checked consume
is not `REAUTH`-only, either: `verifyEmailCode`'s read-then-delete-and-ignore shape is the *existing*
consume for the `LOGIN` and `PASSWORD_RESET` purposes too, so the same two-concurrent-proofs race lets an
attacker who intercepts a password-reset code race the legitimate owner and have *both* resets succeed.
The fix therefore replaces `verifyEmailCode` for all three purposes rather than adding a parallel path for
`REAUTH`, and it needs a store method that can report an affected-row count —
`DeleteEmailVerificationCodeIfMatch` (`email_verification_code.go:91-105`) uses `ExecContext` and discards
`RowsAffected` entirely, so a new `ConsumeEmailVerificationCode` returning the count (or a `RETURNING`
row) is a prerequisite for "required to affect a row" to be implementable at all. `challengeRecoveryCode`
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
check would. And `Login` is not the only such caller: `SwitchWorkspace`'s own MFA second step
(`auth_service.go:895-898`) reaches `challengeMFAAndClear` → `challengeRecoveryCode` too, *before*
`switchWorkspaceInternal` runs — so it consumes a recovery code outside every fence and lock this design
places on the switch path, with the same blind last-write-wins overwrite (two concurrent presentations of
the same recovery code each match their own `MFAConfig` snapshot, both write, the code is spent twice and
the later write resurrects whatever the earlier removed; a correct code clears the lockout, so T9 does not
bound it). All three call sites need the same fix: the transaction-aware compare-and-consume replaces
`challengeRecoveryCode` everywhere it would otherwise run inside — or, for `SwitchWorkspace`'s second
step, alongside — this lock, not just in the four new methods; and `SwitchWorkspace`'s MFA step joins the
fenced sequence rather than running ahead of it. `challengeMFACode` is still fine everywhere, since it
never writes.

Consuming a recovery code as `EnableMfa`'s own proof, specifically for a rotation, creates one more
place the whole-column-replace rule above already warned about —
[caught in a later round](https://github.com/bytebase/bytebase/pull/21235): a rotation's `CredentialProof`
can itself be `recovery_code` (see below), so the same request both consumes one live recovery code (the
proof) and promotes `TempOtpSecret` to `OtpSecret` (the mutation) — two writes to the same `mfa_config`
column, in the same transaction. Building the promotion's patch from the snapshot this transaction locked
at the *start* — before the proof consumed anything — would write back a `RecoveryCodes` list that still
contains the code the proof step just used, resurrecting a single-use credential the same request just
spent. The two writes have to be one: consume the code and promote the secret against a single patch
built from the post-consumption state (or folded into the same `UPDATE ... SET mfa_config = ?` as one
statement), never two sequential writes to a column `Store.UpdateUser` always replaces whole. Reusing T9's table
for all three means an attacker with a stolen session but no credential gets the same
five-guesses-per-ten-minutes bound as at login on every channel — not a fresh oracle, and no new
lockout kind to build; `RequestReauthCode`'s send side reuses the same table's existing resend
cooldown, so it can't be used to mail-bomb an account either. Accepting the old device's code (or a
recovery code) as an alternative to password on `EnableMfa` matters most exactly when replacing a lost
device; `email_code` covers the case neither of those helps — first-time enrollment, where nothing
prior exists to prove control of at all (see Cloud vs. self-hosted). Because each method requires its
proof in its own request message, whether it needs re-authentication is visible in its schema —
nothing to audit across a shared patch. On success — meaning a success that *actually changed live
credential material*, which is not every accepted call — the mutating methods call
`DeleteWebRefreshTokensByUser` and bump `LastCredentialChangeTime` in the same transaction as the
mutation itself. The qualifier is load-bearing for exactly one branch: `EnableMfa` on a first-time
passwordless enrollment requires no `CredentialProof` and promotes nothing (it only verifies the OTP and
leaves the account as unenrolled as it found it — see `EnableMfa` and the receipt-confirmation split), so
revoking sessions or bumping the generation there would let any stolen session run
`StartMfaEnrollment`+`EnableMfa` and knock the victim out of every session without holding any credential
at all — a no-credential griefing lever, and a direct contradiction of that branch's own "writes nothing,
harmless to retry" guarantee. Revocation and the bump follow the *mutation*, not the method name: they
fire in `EnableMfa` only on a rotation (a live secret is actually replaced), and for enrollment they
move to `ConfirmRecoveryCodes`, which is where the secret and codes actually go live and which does
require proof. `DisableMfa`, `ChangePassword`, and every rotation path change live material by
definition, so they always revoke — today only
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

That fixed order still has a gap when the email itself moves mid-sequence —
[a seventh finding](https://github.com/bytebase/bytebase/pull/21235): every child-row lock above runs
`WHERE user_email = ?` against whatever email the caller resolved *before* opening this locking
sequence — read once, from the account row a credential handler fetched to authorize the request, before
any lock is held. If a concurrent `UpdateEmail` commits between that read and these child-lock queries,
its cascade has already moved every matching row in `oauth2_authorization_code`, `oauth2_refresh_token`,
and `web_refresh_token` to the new email by the time `WHERE user_email = <old email>` runs — locking
nothing, not the rows this design needs held. Locking the principal row afterward by its stable ID still
finds the right account, so the credential mutation itself succeeds — but the child-table locks taken
moments earlier locked the wrong, now-vacated key, and any delete keyed off that same stale email (G7's
revocation included) deletes nothing: the tokens it exists to revoke survive, silently, on an account
that just proved it still controls its own credential. Re-deriving the email from *inside* the lock
doesn't avoid this, only moves it — the child rows still have to be located by some email before the
principal is locked, and that read is exactly what can go stale.

The fix is a check, not a different lock order: immediately after locking the principal row by ID,
compare its current `email` against the email the child-row locks were just taken under. A mismatch
means a concurrent `UpdateEmail` won the race and every child lock this sequence is holding is stale —
abort and retry the whole sequence from the top, re-reading the account's current email and re-locking
its child rows under it, rather than proceeding to mutate or delete against a key that no longer matches
any row. This is the same "whichever side reaches the shared row first determines the outcome" principle
this doc already applies to the issuance-vs-revocation race, with `UpdateEmail` as a third contender:
`UpdateEmail` first means it wins outright and the retry picks up the new email cleanly; one of these
four methods first means it holds the current email's child locks for the rest of its own transaction,
so `UpdateEmail`'s own child-lock step simply blocks behind them instead of racing past unlocked.

"Retry against the new email" is right for most proofs and [catastrophically wrong for
one](https://github.com/bytebase/bytebase/pull/21235), so the mismatch branch has to split by what the
proof was bound to. `current_password`, `otp_code`, and `recovery_code` are bound to the *account*: a
rename says nothing about whether the caller still holds them, so retrying against the account's new
address is correct and the caller is none the wiser. `email_code` is bound to a *mailbox address* — it
proves only that someone can read mail sent to the address it was mailed to. If the account has since
been renamed, that proof says nothing whatsoever about the account as it now exists, and retrying would
let control of the *former* address authorize a credential change on the renamed account. So: on
mismatch, account-bound proofs retry, `email_code` proofs **reject** — the caller must request a fresh
code, which will necessarily go to the account's current address.

The same split applies with more force to `ResetPassword`, which is `email_code`-shaped end to end and
otherwise exempt from this design. Its `PASSWORD_RESET` code proves control of one mailbox; if
`UpdateEmail` commits between that proof and the fenced write, proceeding by stable principal ID hands
control of the *old* mailbox a password reset on the renamed account. That is not a hypothetical ordering
nicety: renaming away from a compromised address is exactly why someone changes their email, and this
would let the attacker they were fleeing complete a reset afterward. Moving the write and revocation
inside the fence — the fix given for `ResetPassword` further below — does not address it at all, since
the problem is *which account* the transaction is entitled to act on, not whether its writes are atomic. After acquiring the
fence, `ResetPassword` re-reads the principal and proceeds only if its current email still equals the one
the code proved; otherwise it rejects, unconsumed work already done notwithstanding. Same rule as the
OAuth grant re-read below, with the opposite resolution — there a rename means "same grant, new address,
carry on," here it means "different account than the one that was proven, stop."

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
The *same two tables also lack a `client_id` index* — a gap that predates this design but that its new
client-fence locking turns from latent to load-bearing: `DeleteOAuth2Client`'s existing `ON DELETE
CASCADE` already fires an unindexed `DELETE … WHERE client_id = ?` per client (a locking sequential scan),
and the client-fence step this design adds ("lock every grant row referencing the client in primary-key
order") is a second unindexed locking scan, now run while holding the client fence — the worst place for
one. Add `idx_oauth2_authorization_code_client_id` and `idx_oauth2_refresh_token_client_id` in the same
migration.

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

That "whichever side reaches the shared row first" framing only holds for two parties —
[a genuine three-party gap found in a later round](https://github.com/bytebase/bytebase/pull/21235):
`Login`/`Signup` locks the principal, inserts its new `web_refresh_token` row, and commits, releasing
the principal lock — but revocation's own child-row lock step ran *earlier*, against whatever rows
existed *before* that insert, so the newly-committed session isn't among the rows revocation already
locked. Revocation still has to catch it — "deletes whatever exists when it runs, not a snapshot from
when the request started" is the whole point of the paragraph above — so its actual deletion, run after
it finally acquires the principal lock, re-derives the current row set and reaches for that new row too.
If a *third* transaction — an ordinary `Refresh` against that same brand-new session — locks it first
(the correct child-before-parent order for an *existing* row, same as any other `Refresh`), the two
sides now deadlock exactly like the two-party case above, just assembled by three transactions instead
of two: revocation holds the principal, waiting on the row `Refresh` holds; `Refresh` holds the row,
waiting on the principal revocation holds. Neither one used the wrong order — revocation locked every
row that existed at scan time before touching the principal, and `Refresh` locked its row before
reaching for the principal — the row that breaks it is one neither transaction had anything to order
against, because a still-uncommitted `Login` created it in between.

Ordering alone can't close this — the row genuinely didn't exist yet when revocation did its own
child-locking pass, so no fixed table order helps. Closing it needs a lock that every participant shares
*before* it goes near the account's rows at all: one account-scoped advisory lock (keyed on the
principal's stable ID), held for the rest of the transaction that acquires it and released automatically
on commit or rollback, no separate unlock to forget.

Where in the transaction it's acquired is the whole point, and [getting that wrong reintroduces the
deadlock it exists to remove](https://github.com/bytebase/bytebase/pull/21235): an earlier version of
this paragraph had the inserting paths take it "immediately before that insert," which is *after*
`Login`/`Signup` has already locked the principal and after `Refresh`/`switchWorkspaceInternal` has
locked its old token row and the principal. That's a plain lock-order inversion against revocation, which
takes the fence first and reaches for rows second — issuance holding the principal while waiting for the
fence, revocation holding the fence while waiting for the principal. The fence is only a fence if it is
strictly first: **every** participating transaction acquires it as its opening statement, before any
`SELECT ... FOR UPDATE`, any consume, any insert, and any authentication step. It sits above this doc's
entire child-before-parent ordering rather than inside it — that ordering governs which *row* locks come
in which sequence, and this governs that no participant touches rows at all until it holds the account.

Who participates is a **rule, not a list** — stated that way deliberately, because the enumerated version
of this paragraph was wrong twice in a row, each time by naming the paths that had come up so far and
[missing ones found the round after](https://github.com/bytebase/bytebase/pull/21235). The rule: *any
transaction that reads or scans this account's token rows and then writes based on what it found, and any
transaction that inserts one, takes the fence as its first statement.* Both halves matter — a scanner
without the fence can have rows appear underneath it, and an inserter without the fence is what makes
them appear. (The one case that cannot take a fence first is an account that doesn't exist yet; see
account creation below, where the uncommitted principal row does the same job.) A path that satisfies the ordered-lock contract but skips the fence is still broken: ordering
governs rows that exist, the fence governs rows that don't exist yet, and every finding in this section
has been one or the other.

By that rule the participants are: G7's revocation on all four credential-mutating methods;
`Login`/`Signup`, `Refresh`, and `switchWorkspaceInternal`; the OAuth mint (`handleAuthorizePost`) and
both exchange paths; `UpdateEmail`; `handleReauthorize`; the three retained password resets —
`ResetPassword`, `UpdateUser`'s admin-assisted path, and the `bytebase recovery` CLI
(`backend/component/recovery/service.go:361-368`, see Token-minting paths for why it was the easiest to
miss and the worst to have missed). Several of these were missed by earlier enumerations the same way —
each had already been given the ordered-lock fix in an earlier round, which reads like the whole
treatment but is only half of it.

**One account fence is not enough**, though — [the rule above is right and its
single scope is wrong](https://github.com/bytebase/bytebase/pull/21235). OAuth grants hang off two
parents, and client deletion is a scan-then-write over the *client's* children with the identical
absent-child gap: `DeleteOAuth2Client`/`DeleteExpiredOAuth2Clients` scan the client's existing grants,
then `DELETE FROM oauth2_client` cascades and re-derives locks over whatever grants exist at that moment
— including an authorization code `/authorize` committed in between. A token exchange holding that unseen
code and waiting on the client row deadlocks against the deletion holding the client and cascade-waiting
for the code. The account fence cannot close it, and not just because client deletion wasn't listed: a
client's grants span *many* accounts, and `DeleteExpiredOAuth2Clients` spans many clients, so no
principal-keyed lock serializes a deletion against a mint for a different account on the same client —
including a client that had no grants at all when the scan ran.

So there are two fences, one per parent: the account fence keyed on `principal.id`, and a **client fence**
keyed on `oauth2_client.client_id`. A transaction takes whichever the rule implicates — and a transaction
touching grants implicates both, since every grant row has both parents. `handleAuthorizePost` and both
exchange paths take both; `handleReauthorize` takes both (it deletes by user *and* client);
`DeleteOAuth2Client`/`DeleteExpiredOAuth2Clients` take the client fence, before their own scan; the
web-session-only paths (`Login`/`Signup`, `Refresh`, `switchWorkspaceInternal`, `ResetPassword`,
`UpdateEmail`, the credential-mutating four) take the account fence, and also the client fence when their
revocation reaches OAuth grants — which G7's does, so in practice all four credential-mutating methods
take both.

Two fences means an ordering problem *between the fences*, which is the same failure this whole section
keeps rediscovering one level up, so it gets a fixed order rather than being left to chance: **account
fence first, then client fence**, always, by every path that takes both. For the bulk paths, the same
rule that governs rows governs fences — `DeleteExpiredOAuth2Clients` takes its client fences in
`client_id` order, and a revocation spanning several of an account's clients takes those in `client_id`
order too, after the account fence. Arbitrary but fixed is the whole requirement; picking account-first
just matches the order the rest of this doc already reads in.

One participant *cannot* obey "as its first statement," and saying so is the fix rather than an exception
to wave at — [caught the round after](https://github.com/bytebase/bytebase/pull/21235): account creation.
`Signup` (`auth_service.go:345,361`) calls `Store.CreateUser` and then `finalizeLogin` as two separate
calls, and `CreateUser` runs a bare `QueryRowContext` on the store's own handle (`principal.go`,
`INSERT INTO principal ... RETURNING id`) — autocommit. The same shape covers the other two creation
paths, SSO auto-provisioning (`auth_service_idp.go:129`) and Cloud email-code auto-provisioning
(`auth_service_email_code.go:267`), both of which flow into the same `finalizeLogin` (`:223`). The account
fence is keyed on `principal.id`, which does not exist until that insert commits, so a newborn account
literally has nothing to take a fence on when its flow begins.

The gap that leaves is not theoretical: the principal is committed and visible for the whole window
between `CreateUser` returning and `finalizeLogin` inserting the session. An admin deactivation or
admin-assisted password reset landing in that window finds a real account with zero sessions, revokes
nothing, and reports success — and then the original signup inserts its refresh token afterward, leaving
a live session on an account that was just deactivated or had its password reset out from under it. No
fence ordering helps, because the fence didn't exist yet on one side.

Creation and session issuance become one transaction: insert the principal, take the account fence on the
id that insert returns, insert the session, commit. Inside it the uncommitted parent does the fence's job
by itself — no other transaction can see the row at all, so none can act on the account in the window,
and an admin path racing it fails to find the user rather than half-succeeding against it. That is the
correct outcome: creation either happens completely, session and all, or not at all. This also folds the
creation paths into the same `nextProjectID`-shaped discipline the rest of this section uses — lock (or
here, own) the parent before writing the child, rather than committing the parent and hoping nothing
happens next. Needs its own interleaving test: an admin deactivation or reset issued against an account
mid-signup must either lose cleanly (user not yet visible) or win cleanly (signup rolls back), never
observe the account and miss its session.

A fence per transaction still doesn't make MFA login *one* authenticate-and-issue sequence, which is what
this section has been assuming all along — [the round after](https://github.com/bytebase/bytebase/pull/21235).
An MFA login is two HTTP requests: the first verifies the first factor and returns a signed
`mfaTempToken` good for five minutes (`auth_service.go:36,731`), committing and releasing its fence; the
second presents that token plus an OTP or recovery code, and `completeMFALogin`
(`:645-672`) re-reads the account but verifies *only* the second factor — it never revisits the password
that got the temp token issued. So a `ChangePassword` or `ResetPassword` can commit in between, revoke
everything, and the holder of a temp token minted against the **old** password still completes step two
and walks away with a brand-new session. No fence closes this: the two requests are separate
transactions, and the fence is correctly released when the first commits.

This one is not a caveat to accept alongside the access-token window (see G7). That window is about
already-issued tokens continuing to answer for ≤1h; this *mints a new session after the revocation*, which
is precisely the thing G7 claims to close, so leaving it would make G7's own sentence false for a
five-minute window on every MFA account. It is also the cheapest place to introduce the generation stamp:
this token is short-lived, single-purpose, minted by `Login` itself, and already carries account-specific
claims, so one more is a local change to a flow this doc already modifies. (At the time, the `/authorize`
gap was still a Non-goal precisely because *it* would have needed the stamp threaded through system-wide
JWT issuance. Token-minting paths, below, is where that stopped being avoidable.)

The temp token carries the credential generation it was issued against, and step two rejects it on
mismatch, re-read inside its own fence. Generation means *any* of the mutations that revoke sessions, not
just the password: all four methods here, plus `ResetPassword` and the admin-assisted reset, already write
the principal row inside the fence, so they bump one monotonic `Profile.LastCredentialChangeTime` while
they're there. `LastChangePasswordTime` is not reusable for this — it deliberately means "has this account
ever had a caller-chosen password" for `email_code` eligibility (see below), and MFA mutations must not
move it. A stale temp token then fails closed at step two rather than completing. Note the second factor
alone does not cover this even incidentally: `EnableMfa` rotating the secret would fail the OTP check by
luck, but `ChangePassword` and `ResetPassword` leave `MFAConfig` untouched, so the OTP still validates
perfectly against a credential the account no longer authenticates with. Needs the interleaving in the
regression tests: a temp token issued, a credential mutation committed, and step two attempted after —
which must fail, not mint.

**Token-minting paths.** The temp-token fix above is one instance of a question this design had never
asked systematically: *what, besides `Login`, can produce a fresh access token from something an attacker
might already hold?* Asking it turned up
[the one that matters most](https://github.com/bytebase/bytebase/pull/21235). `SwitchWorkspace`
(`auth_service.go:889-910`) authenticates from nothing but the interceptor-validated JWT — `user, ok :=
GetUserFromContext(ctx)` — checks workspace membership, and calls `switchWorkspaceInternal`, whose
`web == false` branch returns `response.Token = token` (`:1008-1010`): a brand-new access token minted
by `generateLoginToken` with a full `GetAccessTokenDuration` (`:743-749`). No refresh token is consumed,
no credential is proven, and `rejectMCPOriginatedTokenMint` blocks only MCP-origin requests, not a stolen
ordinary API token. `workspace_service.go:317,426` reach the same mint by two more routes.

So a stolen non-web access token can be spent, before it expires, on a fresh one with a fresh full
lifetime — and then again, and again. The ≤1h ceiling G7 claims is not a ceiling at all for a non-web
token; it is a renewal interval. That is worse than the `/authorize` case that was a Non-goal: that one
needs an OAuth client and yields a grant this design's revocation can at least delete, whereas this needs
nothing, leaves no row anywhere to revoke, and renews indefinitely. The account fence does not touch it —
fencing serializes the issuance, it does not ask whether the JWT presenting itself is still entitled to
one.

Fixing it is now cheap for a reason that did not hold when `/authorize` was deferred:
`Profile.LastCredentialChangeTime` already has to exist for the MFA temp token, so the remaining work is
to stamp it into access tokens at issuance and compare it against the account row at every path that mints
a *new* token from an *existing* one — `SwitchWorkspace` and its two `workspace_service.go` callers,
plus `/authorize`, which the same check closes for free. A token whose stamp is older than the account's
current value is refused; the holder must authenticate again. Ordinary API calls are deliberately *not*
in this set: rejecting those would turn every credential change into an immediate hard cutoff for
in-flight work, which is the caveat G7 knowingly accepts. The line is minting, not use.

Two rollout details this needs and the temp-token version does not. The first is a distinction the
naive version of "refuse on a missing stamp" gets fatally wrong — [caught in
review](https://github.com/bytebase/bytebase/pull/21235): there are *two* different things that can be
absent, and only one of them may cause a refusal. A **token** with no generation claim is a pre-upgrade
token, and refusing to let it mint is correct. But the **account row** also has no generation until its
first credential mutation — `LastCredentialChangeTime` starts nil on every account that exists today and
every account created after the upgrade — so a token freshly and legitimately minted *after* the upgrade,
for an account that has simply never changed a credential, would carry whatever that nil stamps to. If a
nil account value produces a missing claim and a missing claim is refused, `SwitchWorkspace` and
`/authorize` reject brand-new, perfectly valid tokens until the user happens to change a credential —
breaking normal operation for the entire existing user base on day one. The account's generation must
therefore always be *defined*, never nil: the same migration that backfills `LastChangePasswordTime`
(see Cloud vs. self-hosted) backfills `LastCredentialChangeTime` to the row's `created_at` for every
existing `END_USER`, and account creation sets it going forward. With that, the rule is a clean
comparison — a token is refused only when its claim is *strictly older* than the account's current
generation (a real credential change has happened since it was minted), a fresh token carries the
account's current generation and matches, and the only truly-missing *claim* is a pre-upgrade token,
refused as its own explicit case. And `LastCredentialChangeTime` is now load-bearing
for both this and the temp token, so the seven mutations that bump it — the four here plus `ResetPassword`,
the admin-assisted reset, and the `bytebase recovery` CLI reset (below) — must bump it inside their fenced
transaction, not before or after, or a token minted mid-mutation could carry a stamp that never matches
anything.

The seventh is the one a `backend/api/`-scoped search cannot see —
[caught by Codex](https://github.com/bytebase/bytebase/pull/21235): the self-hosted operator recovery CLI
(`backend/component/recovery/service.go:361-368`) writes `PasswordHash` through the same `Store.UpdateUser`,
and today it revokes nothing at all — not even the best-effort web-token delete the other two retained
resets have. A break-glass reset is the *most* adversarial context this design serves: the operator is
resetting the password precisely because the account may be compromised, and today every session, OAuth
grant, and MFA temp token the attacker holds survives it. It joins the other resets fully: account fence
first, password write, all-three-table revocation, and the generation bump, in one fenced transaction.
(Its `LastChangePasswordTime` was already handled — `Store.UpdateUser` bumps that automatically on any
`PasswordHash` write, `principal.go:429-434` — which is exactly why the *other* stamp has to be verified
per-writer rather than assumed to tag along.)

Four corrections to the paragraphs above, from a systematic sweep of every token-minting path (not the
one-at-a-time discovery that found `SwitchWorkspace`):

- *The MFA temp token has two mint sites, not one.* The temp-token fix says it is "minted by `Login`
  itself"; it is not. `checkMFARequired` (`auth_service.go:718-739`) is called from `Login` (`:209`) **and
  from `SwitchWorkspace` (`:903`)**, both reaching `GenerateMFATempToken` at `:731`. The
  `SwitchWorkspace` route is worse: it needs only the stolen access token, no password, to obtain a temp
  token, which then completes a full `Login` after the victim's credential change. The stamp is written
  and checked inside `checkMFARequired`, not at the `Login` call site, or the second door stays open.
- *`SwitchWorkspace` doesn't even expiry-check the session it consumes.* `switchWorkspaceInternal`
  consumes the web refresh token via `GetAndDeleteWebRefreshToken`, which carries no `expires_at`
  predicate (`web_refresh_token.go:62-84`); `Refresh` compensates by checking the returned row's expiry
  (`auth_service.go:488-490`), but `switchWorkspaceInternal` never does (`:994` nil-check and `:997`
  email-check only) and reuses the already-past `ExpiresAt` for the new session (`:1000`). So a session
  past its 7-day absolute lifetime still mints a fresh full-lifetime token through
  `SwitchWorkspace`/`LeaveWorkspace`/`DeleteWorkspace` until the hourly sweep removes the row — the
  absolute web-session cap is bypassed independently of the credential stamp. Both paths must check
  expiry at consume time, the same `expires_at > now()`-at-consume rule the credential codes get below.
- *"≤1h" and "30 days" are not real ceilings.* `access_token_duration` has only a one-minute *floor*
  (`setting_service.go:769-775`); a workspace with `FEATURE_TOKEN_DURATION_CONTROL` can set it to 90
  days, and every minted replacement inherits that (`generateLoginToken` reads it live). And OAuth refresh
  writes a *fresh* full `refreshTokenExpiry` on every rotation (`token.go:522`) with no absolute cap, so
  30 days is the maximum *idle* gap, not a lifetime. Only the web refresh token has a true absolute cap
  (`Refresh`/`switchWorkspaceInternal` pass the original `ExpiresAt` through). G7's exposure-window
  numbers are restated as "up to the workspace-configured access-token lifetime, unbounded above" and
  "renewable indefinitely for OAuth," and closing the *renewal* is what the stamp does regardless of the
  window's size. An absolute cap on both is a worthwhile follow-up but not required for G7.
- *The stamp has nowhere to live for non-`END_USER` principals.* `SwitchWorkspace` itself guards
  `Type != END_USER` (`:850`), but its two `workspace_service.go` callers do not, and
  `generateLoginToken` has a live `SERVICE_ACCOUNT` branch (`:751`). Service accounts and workload
  identities are not in `principal` and have no `profile`, so `LastCredentialChangeTime` cannot exist for
  them and "refuse to mint on a missing stamp" would permanently break service-account-driven
  `LeaveWorkspace`/`DeleteWorkspace`. The stamp check applies to `END_USER` principals only, stated
  explicitly so it is a typed rule rather than a silent type-conditional bypass — non-`END_USER`
  credentials are not credential-changeable through any path this design touches, so there is nothing for
  the stamp to protect there anyway. (For the same reason, `RotateDirectorySyncToken`, service-account
  keys, and workload-identity config — the repo's `MINTS_CREDENTIAL_FOR_OTHERS` class — survive every
  revocation here and are correctly out of scope: they establish *other* principals' credentials, not a
  replacement for the acting account's.)

`UpdateEmail` earns membership without deleting anything: it scans and locks the child rows that exist,
then runs `UPDATE principal SET email = ?`, whose `ON UPDATE CASCADE` re-derives and locks whatever child
rows exist *at that moment*, including any a login committed in between. A third-party `Refresh` holding
one of those new rows and waiting on the principal deadlocks against the cascade holding the principal and
waiting on the row — the same three-party shape, with the email cascade where revocation's delete was.

For the three deleters, the fence closes a second failure the ordering fix leaves wide open, and it is
the more serious one: a row inserted after the scan is not in the deleting statement's snapshot, so it
simply *survives*. `handleReauthorize` returns success while a refresh grant minted mid-flight by a
concurrent OAuth exchange stays usable; `ResetPassword` and the admin password path report a completed
reset while a session created mid-flight stays live. Each is the exact guarantee the caller just told the
user it had made. So for those three the fence isn't only deadlock avoidance — and for the two password
paths it composes with the correctness fix above: fence first, then the password write and the revocation
in that same fenced transaction, so "the password changed but the sessions didn't" cannot be a reachable
state by either route.

With the fence held from the top by all of them, `Login` can't commit a new session while a revocation, an
email change, or a reset is anywhere between its scan and its commit, so there's no window left for a
third party to grab a row the scanning side hasn't accounted for, and no window for a row to slip past a
delete entirely; and none of the scanning paths can start while a new session or grant is mid-insert, so
none undercounts what it's about to delete or rewrite — and with the client fence, the same holds for a
client's grants across every account that has one, which the account fence alone could never cover. Needs
a deterministic real-PostgreSQL regression test for this three-transaction interleaving across the
variants that differ in *kind*, not merely in caller — a deleting one (revocation), the cascading one
(`UpdateEmail`), one whose failure mode is survival-not-deadlock (`handleReauthorize` or `ResetPassword`),
and one that turns on the client fence specifically: a client deletion racing an `/authorize` mint for an
account the deletion never touches, which no account-keyed fence can serialize. Plus one for the fence
order itself — a path taking both fences against a path taking them in the opposite order is precisely
the deadlock the fixed order exists to prevent, so the test has to prove the order is actually held, not
just that each fence works alone.

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

That is now the third separate finding of one pattern — `challengeRecoveryCode`, then these profile
writes, then [SSO auto-undelete](https://github.com/bytebase/bytebase/pull/21235) — so it is worth stating
as a rule and closing by audit rather than waiting for a fourth. The rule: **once a flow holds the
principal lock, every write to that principal inside it must go through the enclosing transaction.**
`Store.UpdateUser` opens its own (`principal.go:465`), so any call to it from inside a fenced flow is a
transaction waiting on a lock its own caller holds — a hang until `statement_timeout`, not an error, and
not something a retry helps.

The new instance: `getOrCreateUserWithIDP` reactivates a deactivated account on SSO login via a
standalone `Store.UpdateUser` (`auth_service_idp.go:179-186`), and that call *is* part of SSO's
authentication step, which this design now runs under the principal lock — so SSO login for a previously
deactivated user would stop working entirely. It moves into the enclosing transaction like the others.

That same SSO flow forces one bound on *how* the lock-before-authentication rule applies, or it becomes a
denial-of-service surface: `getOrCreateUserWithIDP` performs an outbound round-trip to the identity
provider (`auth_service_idp.go:55-64`, dispatching to `oauth2UserInfo`/`oidcUserInfo`/`ldapUserInfo`)
as part of "authentication," and the earlier rule ("move the lock before authentication") read literally
would hold the account's `FOR UPDATE` across that unbounded external call — a slow or hung IdP would pin
one account's row lock, blocking every concurrent login, refresh, and credential change for that account
until it times out. The resolution is that the IdP round-trip authenticates against the *provider*, not
against local mutable credential state, so it belongs *before* the lock, not inside it: resolve the
external identity first, unlocked, and take the fence and the principal lock only for the local
verify-consume-mutate-issue sequence that actually reads and writes rows. "Lock before authentication"
means before the part of authentication that inspects state the lock protects — never around a network
call to a third party. The same applies to `bcrypt` and `totp` verification, which are CPU-bound and
already pure, but the IdP call is the one that could hold the lock for seconds.

Auditing every `Store.UpdateUser`/`CreateUser`/`UpdateUserEmail` call site — across the whole `backend/`
tree, a scope [that itself had to be corrected](https://github.com/bytebase/bytebase/pull/21235): the
first version of this audit searched only `backend/api/`, and the `bytebase recovery` CLI's password
writer lives in `backend/component/recovery/`, which is precisely how it went unlisted — rather than
pattern-matching on the ones already found, the complete set reachable from a lock-holding flow is:
`challengeRecoveryCode` (`auth_service.go:606`), `switchWorkspaceInternal` (`:971`) and `finalizeLogin`
(`:1045`), the three account-creation paths (`auth_service.go:345`, `auth_service_idp.go:129`,
`auth_service_email_code.go:267`), `ResetPassword`'s own password write
(`auth_service_email_code.go:116`), the SSO undelete (`auth_service_idp.go:184`), `UpdateUser`/
`UpdateEmail` themselves (`user_service.go:487,703`), and the two SCIM profile writers
(`webhook.go:433,942`). All are addressed above. The remaining call sites — admin
`DeleteUser`/`UndeleteUser` (`user_service.go:558,637`), the SCIM create/delete pair
(`webhook.go:141,270`), and the recovery CLI's password write
(`backend/component/recovery/service.go:366`, a standalone process run while no server flow holds any
lock) — are operations that never run inside a login or credential flow, so they cannot self-deadlock;
they take the account fence under the rule above. Admin deactivation's separate problem (it revokes no
sessions at all today) is the one named in the account-creation paragraph, and the recovery CLI's
(it revokes nothing either, in the most adversarial context of all) is covered in Token-minting paths.

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

**Store-layer prerequisite.** Every "inside the transaction," "before the insert," and "in one fenced
transaction" instruction in this section assumes the writes involved *can* join a caller-held
transaction. Today they cannot: `CreateWebRefreshToken` (`web_refresh_token.go:19-33`),
`CreateOAuth2AuthorizationCode` (`oauth2_authorization_code.go:30-51`), and `CreateOAuth2RefreshToken`
(`oauth2_refresh_token.go:32-53`) each run on `s.GetDB().ExecContext` — an autocommit statement on a
pooled connection — with no `*sql.Tx`-accepting variant, and the same is true of the `principal` writers
`Store.UpdateUser`/`CreateUser` open their own `BeginTx`. So the design's own prescriptions are not just
unimplemented but *unimplementable as written*, and in the worst way: because `web_refresh_token.user_email`
is a foreign key to `principal`, the autocommit `INSERT` takes a `FOR KEY SHARE` on the principal row —
which **conflicts with the `FOR UPDATE` the same request's outer transaction is already holding, on a
different pooled connection.** PostgreSQL sees no cycle (the outer transaction is idle-in-transaction,
blocked on the application, not on the database), so it does not abort — the insert simply hangs until
`statement_timeout`, and every web login, signup, refresh, and workspace switch hangs with it. This is
the same self-deadlock shape the doc flags for `challengeRecoveryCode`, one layer down, and it is why the
fence and the ordering are necessary but not sufficient on their own. The prerequisite for this whole
design is a `Tx`-accepting variant of each of these store writers (`CreateWebRefreshTokenTx(ctx, tx, …)`
and the two OAuth equivalents, plus the transaction-aware `Store.UpdateUser` the profile-write fixes
already require), so a handler can open one transaction, take the fence and the row locks, and do the
consume, the mutation, the revocation, and the insert all on that one connection. Without it, the OAuth
mint's "lock `principal` then `oauth2_client` before the insert" is inert — the insert runs on another
connection and orders against nothing — and the web paths hang outright.

Inserting that principal-lock wait ahead of the consume opens a narrower gap of its own —
[caught in a later round](https://github.com/bytebase/bytebase/pull/21235): `validateAuthorizationCode`/
`validateRefreshTokenGrant` check `expires_at` before this fix's lock is ever acquired, and
`ConsumeOAuth2AuthorizationCode`/`ConsumeOAuth2RefreshToken` (`oauth2_authorization_code.go:97-117`,
similarly for refresh tokens) match only on `code`/`token_hash` and `client_id` — no `expires_at`
predicate at all. Before this fix, the gap between that validation and the consume was negligible; now
it's however long the principal lock stays contended, which a concurrent revocation or another exchange
can stretch arbitrarily. A code or token that validated as not-yet-expired can cross its expiry while
waiting on the lock and still get consumed and exchanged once it's finally acquired, since nothing
re-checks. Same shape as the REAUTH fix above: add `AND expires_at > now()` to both consume queries
directly, rather than a separate re-check step, so a grant that expired mid-wait fails the consume itself
instead of relying on a check that already ran before the wait began.

Expiry isn't the only thing that can move during that wait —
[the same shape, one field over](https://github.com/bytebase/bytebase/pull/21235): both handlers keep
using the `authCode`/`refreshToken` struct they read *before* the fence, and after consuming the row they
resolve the account with `GetUserByEmail(authCode.UserEmail)` / `GetUserByEmail(refreshToken.UserEmail)`
(`token.go:149,290`). If `UpdateEmail` wins the fence first, its cascade rewrites `user_email` on that very
grant row. The exchange then resumes, consumes successfully — the consume matches on `code`/`token_hash`
and `client_id`, none of which changed — and looks the user up by an email that no longer exists. The
grant is burned, the client is told `invalid_grant`, and the user is forced to re-authorize, all because
they renamed their account mid-refresh. Nothing is left insecure, but a valid credential is destroyed by a
race the user can't see, and the retry has nothing to retry with.

The pre-fence read is a snapshot, so once the fence is held the row has to be read again rather than
trusted: re-read the locked grant inside the fence and use *its* `user_email`, not the copy from before
the wait. That resolves correctly — a cascaded rename leaves the same grant belonging to the same account
under its new address, so the exchange should simply succeed. Detecting the mismatch and refusing without
consuming would also be safe, but it's the worse behavior: it turns a rename into a spurious re-authorize
for a grant that is still perfectly valid. This is the general form of the rule the rest of this section
keeps applying to rows — anything read before a lock must be re-read after it — and it applies to the
grant's own columns just as much as to the set of rows a scan returned.

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
rule this doc already applies everywhere else. Both also take the client fence before that scan (see the
fence rule above): the ordering fixes rows that exist when they look, and only the fence stops
`/authorize` from committing one they never saw.

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

Naming `DeleteExpiredOAuth2Clients` doesn't cover the token tables' own expiry cleaners —
[caught in a later round](https://github.com/bytebase/bytebase/pull/21235):
`DeleteExpiredOAuth2AuthorizationCodes`, `DeleteExpiredOAuth2RefreshTokens`, and
`DeleteExpiredWebRefreshTokens` (`backend/runner/cleaner/data_cleaner.go:137-160`) each run today as a
single unordered bulk `DELETE ... WHERE expires_at < ?` — harmless before this design, since nothing else
ever locked more than one row across these tables at a time. Now that revocation, `UpdateEmail`, and
client deletion all lock multiple existing rows per table in primary-key order, an unordered bulk delete
overlapping any of them on two or more expired rows can lock those same rows in whatever order its own
scan (an `expires_at` index, most plausibly) happens to visit them — the opposite order from the other
side often enough to deadlock, for exactly the reason every other multi-row fix in this doc locks in
primary-key order in the first place. These three cleaners predate this doc and are otherwise untouched
by it, but so were `DeleteOAuth2Client`/`DeleteExpiredOAuth2Clients` before the finding above — same root
cause, a previously-harmless bulk operation turned live by this design's own new locking. Fixed the same
way: each cleaner locks its matching expired rows in primary-key order before deleting, folding them into
the same deterministic-ordering contract and its two-direction regression tests as every other path that
touches these tables.

Two *more* cleaners of the same shape were missed even after that fix, for a mechanical reason worth
recording: the cited cleaner range (`data_cleaner.go:137-160`) ends exactly one function before
`cleanupLoginAttempts` (`:168`) and `cleanupEmailVerificationCodes` (`:176`), so `DeleteStaleLoginAttempts`
(`login_attempt.go:83-97`) and `DeleteExpiredEmailVerificationCodes` (`email_verification_code.go:109-124`)
— both unordered bulk `DELETE`s — fell just outside the window and went unlisted. `login_attempt` and
`email_verification_code` have no foreign keys, so nothing *cascades* them into the token-table order; on
their own they would be a stall, not a deadlock, since the design moves a single-row `email_verification_code`
consume and `ClearLoginAttempt` inside the fenced transaction and either can queue behind the hourly bulk
delete while the fence and every token-row lock are held. But `login_attempt` becomes a genuine deadlock
surface the moment one fenced transaction holds *two* of its rows — and `ResetPassword` does exactly that
once folded into a single transaction: it clears `(email, EMAIL_CODE)` inside `verifyEmailCode`
(`auth_service_email_code.go:444`) and then `(email, PASSWORD)` (`:129`) in the same request, and the login
side clears a third kind (`MFA`) through `challengeMFAAndClear`. So both cleaners join the ordered-lock
contract, and the in-transaction `login_attempt` clears take their rows in primary-key (`identity, kind`)
order like everything else.

Scheduled cleanup isn't the only unordered multi-row deleter left, either — [one more of the same
shape](https://github.com/bytebase/bytebase/pull/21235), reachable on demand rather than hourly: the MCP
`reauthorize` tool. `handleReauthorize` (`backend/api/mcp/tool_reauthorize.go:35`) calls
`DeleteOAuth2RefreshTokensByUserAndClient`, a plain `DELETE ... WHERE user_email = ? AND client_id = ?`
with no ordering and no lock step (`oauth2_refresh_token.go:138-152`). A user/client pair holding two or
more refresh grants — ordinary for a long-lived MCP integration that has refreshed since its last
cleanup — gives it two rows to lock in scan order, against a credential revocation or client deletion
holding those same rows in primary-key order. Identical root cause to the cleaners, and identical fix:
lock the matching rows in primary-key order before deleting, inside the same contract and the same
two-direction tests. Worth naming separately because it's a user-triggered path rather than a background
one, so its overlap with a credential change isn't a rare scheduling coincidence — a user reauthorizing
an MCP client *because* they just changed a credential is the expected sequence, not an unlucky one. It
also takes the account fence, per the rule above — ordered locking alone would still let an OAuth exchange
mint a replacement grant after the scan, outside the deleting statement's snapshot, so `reauthorize` would
report success on a grant that is still live.

The last unordered deleter is the one this design otherwise *keeps* —
[found the round after](https://github.com/bytebase/bytebase/pull/21235):
`DeleteWebRefreshTokensByUser` itself (`web_refresh_token.go:103-118`) is a plain
`DELETE ... WHERE user_email = ?`, and while G7 moves it inside the locked transaction for the four new
methods, two callers deliberately stay outside that: `ResetPassword`
(`auth_service_email_code.go:123`), exempt because mailbox possession is its own proof channel, and
`UpdateUser`'s surviving admin-assisted password path (`user_service.go:436`), exempt under G6. Both
still call it unordered, on the store's own connection, against an account that can hold several
sessions — so both can deadlock against the primary-key-ordered revocation this design introduces, the
same way the cleaners could.

Here the deadlock is not the worst part. Both callers already committed the password write before this
delete runs, and both swallow its error and continue (`slog.Warn` at
`auth_service_email_code.go:124`, `slog.Error` at `user_service.go:437` — the very log-and-continue
pattern G7 exists to replace). If PostgreSQL picks one of them as the deadlock victim, the password
change stands, the delete is rolled back, and the caller reports success with every pre-reset session
still live — a silent failure of exactly the recovery path a locked-out user reaches for, and one that
looks identical to a clean reset from the outside. Both get the same ordered-lock fix as every other
multi-row deleter here, and both should stop treating the revocation as best-effort: a reset that cannot
revoke is a reset that did not do what it told the user it did. That is the same argument G7 already
makes for the four new methods — it applies just as well to the two paths G7 leaves alone, which is why
this shows up as a lock-ordering finding and a correctness one at once.

Ordering and error-handling still aren't the whole fix here, for the same reason they weren't for
`handleReauthorize` — per the fence rule above, both paths take the account fence first, and run the
password write and the revocation inside that one fenced transaction. Without the fence, a login that
commits a new session after the reset's scan is invisible to the reset's own delete statement, so the
session outlives the password change that was supposed to end it — the failure survives even if the
delete succeeds and its error is checked, because there was never an error to check. Fenced, with both
writes in one transaction, "password changed, sessions didn't" stops being reachable by either the
deadlock route or the snapshot route.

"Stop treating revocation as best-effort" generalizes past the two resets, to three more swallowed-error
sites a sweep for the pattern turns up — each one a security-relevant write that logs its failure and
reports success:

- `Logout` (`auth_service.go:459-463`, also reached from `workspace_service.go:312,421`) logs a
  `DeleteWebRefreshToken` failure and returns `Empty` with the cookies cleared, so a real transient DB
  error leaves the server-side session live for its full remaining lifetime while the user believes they
  logged out — the one moment a user is explicitly asking for revocation. The fix is to propagate the SQL
  *error*, not to treat a zero-row delete as failure: for `Logout` an absent row is the *normal success*
  case (a credential change, the expiry cleaner, or an earlier logout already removed it), and the
  desired end state — no server-side session — is already met. So `Logout` diverges from the
  single-use-consume rule deliberately: a consume needs exactly one row (the row *is* the single-use
  gate), but an idempotent revoke-if-present is satisfied by zero. Keep an absent row successful,
  surface only genuine errors, and *always* expire the cookies (never return before that), so a user can
  clear a stale browser session even when the server row is already gone.
- The OAuth2 revoke endpoint (`revoke.go:60-65`) logs a `DeleteOAuth2RefreshToken` failure and returns
  `200`, telling an MCP client its compromised grant is dead while the row survives up to 30 days. RFC 7009
  wants `200` for an *unknown* token, not a *failed* revocation (`503` is allowed); the two are conflated.
  Same idempotency caveat as `Logout`: an already-absent token is a legitimate `200`, only a real
  storage error becomes `503`.
- The audit interceptor (`audit.go:94-119`) is the deeper of the three, and "return the error instead of
  swallowing it" is *not* a sufficient fix — because the interceptor runs the handler first (`:94-96`)
  and writes the audit row afterward (`:98-119`), the credential mutation has **already committed** by
  the time `createAuditLog` fails, so returning an error only hands the client a misleading failure for a
  change that did happen, still with no audit record. The ordering is the problem, not the error
  handling. For the security-sensitive mutations that lean on audit as their control (`DisableMfa`, the
  admin-assisted resets), the audit row has to be written *inside the handler's own fenced transaction*,
  so the mutation and its record commit or roll back together — the generic post-hoc interceptor cannot
  provide that, because it does not share the handler's transaction. This is a real departure from how
  auditing works for ordinary RPCs, and it is the price of the doc's claim that "the `bb.users.update`
  permission check and audit log are the correct control": a control that can silently no-op on a write
  failure after the fact is not one. (An outbox/precommit record is the alternative if threading the
  audit write through every such handler's transaction proves too invasive; either gives atomicity, which
  the interceptor alone does not.)

None of the three is introduced by this design, but all three sit on paths it now depends on for its own
guarantees, so it names them rather than inheriting them silently — and the audit one in particular is a
correctness change, not just a stricter error check.

That fix closes the race around *exchanging* an existing grant, but not [a separate gap one step
earlier](https://github.com/bytebase/bytebase/pull/21235): `handleAuthorizePost`
(`authorize.go:103-171`) mints a brand-new `oauth2_authorization_code` — a grant that didn't exist
before the request — off nothing but `resolveConsentingUser` (`:200-228`) verifying a bearer access
token's signature and expiry and confirming the user row still exists (`:212,221-227`). No lock on an
*existing* row, because there's no existing `oauth2_authorization_code` row to lock: this part of the
`nextProjectID` shape, not the child-before-parent one, is correct as far as it goes. The problem isn't
the lock ordering, it's what "the user still exists" is being asked to
stand in for — proof the account's *credentials* haven't changed, which a still-valid JWT cannot carry,
since it was signed before whatever credential change just happened and self-contained JWTs are exactly
as valid the second after a revocation as the second before. This was carried as a Non-goal for a while,
on the grounds that closing it meant teaching `/authorize` to check a credential-generation signal no JWT
carried. That reasoning expired: the signal now exists for the MFA temp token, and `SwitchWorkspace`
forced it into access tokens regardless. `/authorize` performs the same
`LastCredentialChangeTime` check as every other mint-from-token path (see Token-minting paths above) —
which is why the account fence it takes is necessary but never sufficient on its own: the fence decides
*when* the mint may run, the stamp decides *whether* it may run at all.

"No existing row to lock" wasn't the whole picture either —
[caught in a later round](https://github.com/bytebase/bytebase/pull/21235): the new
`oauth2_authorization_code` row this insert creates still has two *parents*, not zero, and
`CreateOAuth2AuthorizationCode` acquires both through its own foreign keys — `client_id` against
`oauth2_client`, `user_email` against `principal` — as part of the insert itself, the same way any FK
reference locks the row it points to. The `nextProjectID` pattern this doc keeps citing for "no existing
child" cases doesn't skip locking on that basis; it explicitly locks its one parent *first*. This insert
has two, and nothing says which one the FK checks reach first — if that's `oauth2_client` before
`principal`, it's the opposite of the order the token-exchange fix above already commits to (principal,
then `oauth2_client`, via `issueTokens`/`UpdateOAuth2ClientLastActiveAt`), and the two can deadlock: the
mint holding `oauth2_client` while waiting on `principal`, an in-flight exchange holding `principal` while
waiting on `oauth2_client`. The fix is the same discipline as every other multi-table lock in this doc:
`handleAuthorizePost` explicitly locks `principal` then `oauth2_client` (`SELECT ... FOR KEY SHARE`, the
weakest lock that still orders against the FK check, taken in that order before the insert) rather than
leaving the order to whatever the FK constraints happen to check first — folding the mint into the same
shared order this doc already names for every other path that touches more than one of these tables.
Needs the same deterministic real-PostgreSQL regression test as the other lock-ordering fixes, covering
both directions: a mint racing an in-flight token exchange for the same user/client, and a mint racing
`DeleteOAuth2Client`/`DeleteExpiredOAuth2Clients`. (Strictly there is a *third* parent —
`oauth2_authorization_code.workspace REFERENCES workspace(resource_id)`, `LATEST.sql:690` — so the insert
also takes `FOR KEY SHARE` on the workspace row. It is benign today, because the only `workspace` writers
(`UpdateWorkspace`, `DeleteWorkspace`) take `FOR NO KEY UPDATE`, which `FOR KEY SHARE` is compatible with,
and nothing anywhere takes `FOR UPDATE` on a workspace row. It stops being benign the instant any path
does, so the fixed order is stated as principal → workspace → `oauth2_client`, with the workspace lock
noted as currently uncontended rather than omitted — an omission is how the two-parent framing missed it
in the first place.)

`email_code`'s eligibility is a server-side check, not a UI affordance — otherwise an attacker with a
stolen session and separate mailbox access could use it against an *already*-MFA-protected account,
weaker than the factor actually protecting it (a
[Codex finding](https://github.com/bytebase/bytebase/pull/21235) against an earlier draft that
accepted it unconditionally on all four methods). The rule — no live `MFAConfig.OtpSecret`, and either
`Profile.LastChangePasswordTime` is unset or the deployment is SaaS (added by the fourth finding below,
once the backfill made the account-level signal alone insufficient for pre-existing accounts) — is
checked once, in `CredentialProof` verification itself, not per-method, so it can't be forgotten on a
future caller of the same helper. Both halves of that `or` are deliberately principal-scoped:
`restriction.disallow_password_signin` was tried in the second position twice and is wrong there both
times — first unconditionally
([caught by Codex](https://github.com/bytebase/bytebase/pull/21235) as too coarse: a self-hosted
workspace can allow SSO and local password login at once, and a workspace-wide flag can't see that one
specific SSO-provisioned user in that mix has no usable password), then as an affirmative-only clause,
[caught again](https://github.com/bytebase/bytebase/pull/21235) because a workspace flag cannot speak
for a global principal at all (see the fourth finding below). The account-level
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
every real password that predates the migration. The same migration backfills `LastCredentialChangeTime`
to `created_at` for every existing `END_USER` (for a different reason — see Token-minting paths, where a
nil generation would otherwise make every fresh post-upgrade token unmintable): two columns, one
backfill, both to `created_at`, both then maintained forward.

Left there, the backfill is one-way and permanent for exactly the population `email_code` exists to
serve — [a fourth finding](https://github.com/bytebase/bytebase/pull/21235): once
`LastChangePasswordTime` is non-null, nothing ever unsets it again, and a pre-existing Cloud/SSO account
has no path back to eligibility — `ChangePassword` itself is rejected on Cloud (see Cloud
vs. self-hosted), so "until they change something that legitimately sets the field" was never an escape
for exactly the accounts most likely to need one. Combined with `require_2fa`, an existing passwordless
account with no MFA yet is redirected into an enrollment it now has no self-service way to complete. The
eligibility rule needs a second, durable clause, not just the account-level one: `LastChangePasswordTime`
unset **or** the deployment is SaaS (`s.profile.SaaS`).

The obvious candidate for that second clause was the workspace's own `restriction.disallow_password_signin`,
and [that was unsound](https://github.com/bytebase/bytebase/pull/21235) — worth recording, because the
reasoning behind it looked airtight and was wrong by one scope level. The argument was: the flag only
fires when affirmatively set, which means the entire *workspace* has no local passwords, so there's no
mixed population left to misidentify. True about the workspace, and irrelevant to the principal.
`getAccountRestriction` is workspace-scoped (`auth_service.go:1147-1153`), while `principal` has no
workspace column at all — one global account row per person, across every workspace. So in a self-hosted
multi-workspace deployment, a user with a real, caller-chosen password they use in workspace A becomes
`email_code`-eligible merely by being evaluated through workspace B, which happens to be SSO-only. That
is the precise downgrade this rule exists to prevent: a stolen session plus mailbox access, standing in
for a password the account genuinely has, ending in a replaced MFA configuration.

`s.profile.SaaS` is the same intent at the right scope. It is deployment-wide, not per-workspace, so
there is no other workspace for a password to hide in; SaaS forces `disallow_password_signin` on every
workspace and `ChangePassword` refuses to run at all there, so no caller-chosen password can exist on the
deployment in the first place. That is a property of the whole principal, which is what the rule needs.
Note this doc uses the workspace flag *elsewhere*, in `ChangePassword`'s own rejection, and that stays —
the two directions are not symmetric. Refusing to set a password is a safety judgment where being
over-broad costs a user an operation they can retry; granting `email_code` eligibility is a permission
judgment where being over-broad is a vulnerability. Same flag, opposite consequences of imprecision.

Self-hosted therefore keeps only the per-principal clause, and its residual gap is pre-existing
passwordless accounts anywhere self-hosted (not merely in mixed workspaces): they have no self-service
path back until something legitimately sets the field. Unlike Cloud they do have G6's admin-assisted
reset as a recovery route, which is exactly the asymmetry that justified giving Cloud a clause of its
own. That residual is the wider version of the already-accepted SMTP-less gap (see Non-goals), not a new
kind of gap.

Everything above assumes `LastChangePasswordTime`, once set, stays set — and the SCIM directory-sync
surface can clear it [without ever meaning to](https://github.com/bytebase/bytebase/pull/21235):
`updateUserFromSCIM` (`backend/api/directory-sync/webhook.go:930-943`) wants to change one subfield,
`Profile.Source`, but passes the whole `user.Profile` it read earlier into `Store.UpdateUser`, which
replaces the entire `profile` JSONB column (`principal.go:446-451`) — the same whole-column-replace
hazard already found in `finalizeLogin`, in a writer this doc hadn't looked at because SCIM has nothing
to do with credentials. A SCIM write that reads a profile with `LastChangePasswordTime` unset, then
loses the race to a `ChangePassword` that sets it, writes the stale profile back afterward and clears it
— leaving an account that genuinely has a password looking passwordless, which is precisely the state
that makes `email_code` eligible. The consequence is worse here than the login-timestamp version: that
one reverted a field to a stale value, this one reverts it to *unset*, straight into the eligibility
condition.

Fixing `updateUserFromSCIM` alone doesn't cover the surface, though — [a second SCIM writer, found the
round after](https://github.com/bytebase/bytebase/pull/21235): the `PATCH` handler
(`webhook.go:335-348,433`) never calls `updateUserFromSCIM` at all. It builds its own
`UpdateUserMessage` from the same previously-read `user.Profile` (`:343-345`), applies its operations,
and calls `Store.UpdateUser` directly (`:433`) — the identical whole-column replace, reached by a
different path. Both handlers need the same treatment; fixing only the shared-looking helper would leave
the standalone one open, which is exactly the shape of mistake this doc keeps finding.

Two acceptable fixes, and this design doesn't need to pick between them: each writer can update only the
`source` subfield (a targeted `jsonb_set`, leaving every other field untouched by construction), or it
can join the same lock-and-re-read discipline every other profile writer in this doc now follows — read
the profile inside the locked transaction, patch `source` on *that* copy, write it back before commit.
The subfield update is the smaller change and doesn't put a directory-sync webhook behind a per-account
lock; either one closes it. What isn't acceptable is leaving a whole-profile writer — either of them —
outside the coordination that the entire `email_code` eligibility rule depends on.

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

Every `CredentialProof` channel must actually claim a slot, which is not automatic — today's
`UpdateUser` has an unbounded one to avoid inheriting: its `otp_code` branch runs
`totp.Validate(..., TempOtpSecret)` (`user_service.go:442-450`) with no `claimLoginAttempt`, an
unbounded guessing oracle against the pending TOTP seed (bounded in *impact* only because the minted
secret is returned solely to the same self-caller and cross-user reach needs `bb.users.update`, blocked
in SaaS — so it is an oracle, not a takeover). The design must not carry that forward: `EnableMfa` and
`ConfirmRecoveryCodes` verify `otp_code` too, and the enrollment branch of `EnableMfa` takes *no*
`CredentialProof` at all — so unless each `otp_code` verification claims the `MFA` bucket on its own
(independently of whether a `CredentialProof` is present), first-time enrollment would ship the same
unbounded oracle. `otp_code` verification claims `MFA`; the enrollment path claims it despite having no
`CredentialProof`, precisely so the oracle is closed there rather than reopened.

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
recovery codes itself. For first-time enrollment specifically, that confirmation screen also gains a
fresh OTP input: the code entered earlier for `EnableMfa` has long since expired by the time the user
finishes saving their codes, so `ConfirmRecoveryCodes` needs its own, not a resend of the first (see
Design → Verification). That field needs a way to reach for
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
  // by name, not carried in this request). Replacing an existing factor
  // (the account already has a live OtpSecret) requires credential and
  // promotes *only* the secret here — recovery codes stay pending until a
  // separate ConfirmRecoveryCodes call, safely, since the old set keeps
  // working as a fallback in the meantime (see Design → Verification).
  // First-time enrollment (no live OtpSecret yet) has no such fallback, so
  // this call verifies otp_code but does *not* promote anything — promotion
  // of the secret moves to ConfirmRecoveryCodes, atomically with the
  // recovery codes, so the account is never left MFA-required with zero
  // usable codes (see Design → Verification). credential is required here
  // too for that branch, *unless* the account has neither a password nor
  // live MFA — email_code would be its only option, and email_code is
  // single-use, so it's checked once, at ConfirmRecoveryCodes, where the
  // mutation actually happens (see Design → Verification and
  // EnableMfaRequest.credential). Also rejects a pending set past its expiry
  // (isMFATempSecretExpired, checked today both before OTP verification and
  // before promotion — both carry forward, inside the locked transaction)
  // — pending_version alone only detects a *replaced* pending set, not one
  // that's simply gone stale with no replacement ever minted. name must be
  // the caller's own — no admin path, same reason as ChangePassword.
  rpc EnableMfa(EnableMfaRequest) returns (User) {
    option (google.api.http) = {
      post: "/v1/{name=users/*}:enableMfa"
      body: "*"
    };
    option (google.api.method_signature) = "name,otp_code,pending_version";
    option (bytebase.v1.auth_method) = CUSTOM;
    option (bytebase.v1.audit) = true;
    option (bytebase.v1.mcp_method_class) = FORBIDDEN;
    option (bytebase.v1.mcp_denial_reason) = TAKES_OVER_ACCOUNT;
  }

  // Turns MFA off for `name` by clearing the ENTIRE MFAConfig — the live
  // secret and recovery codes, and equally TempOtpSecret/TempRecoveryCodes
  // and their timestamps. Any pending enrollment or rotation dies with the
  // disable, deliberately: "turn MFA off" moots whatever was in flight, and
  // preserving pending state would let a later ConfirmRecoveryCodes
  // re-enable MFA through its first-time-enrollment branch, silently undoing
  // this call (see ConfirmRecoveryCodes). Matches what UpdateUser's
  // mfa_enabled=false already does today (patch.MFAConfig = &MFAConfig{},
  // user_service.go:410) — stated here because it is now load-bearing, not
  // incidental. Rejected for any non-admin caller while the
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
  // whatever is currently pending. A DisableMfa in between is caught the same
  // way, because DisableMfa wipes the ENTIRE MFAConfig — pending temp state
  // included, see DisableMfa — so a stale confirmation finds no pending set
  // at all and fails on the pending_version check in either branch. That
  // wipe is load-bearing for the branch selector below: without it, an
  // account after a mid-rotation disable (no live OtpSecret, pending state
  // surviving) would be indistinguishable from a first-time enrollment
  // mid-flight, and this call would promote the pending secret down the
  // enrollment branch — silently re-enabling the MFA the owner just proved
  // they wanted off. The rotation branch still re-checks MFAConfig.OtpSecret
  // is live inside the locked transaction, as a structural guard: the branch
  // split's meaning depends on no path ever clearing the live secret while
  // leaving pending state behind, and this check is what turns a future
  // violation of that into a visible failure instead of a silent promote.
  // For first-time enrollment (no live
  // OtpSecret when StartMfaEnrollment minted this pending set — see
  // EnableMfa), that precondition doesn't apply; instead this call requires
  // otp_code and promotes the secret alongside the recovery codes, atomically,
  // completing what EnableMfa already verified but deliberately left
  // unpromoted (see Design → Verification). That branch also runs
  // isMFATempSecretExpired before promoting, the same check EnableMfa itself
  // runs — EnableMfa's own pass over it protects nothing here, since it
  // records no state ConfirmRecoveryCodes can see and this call may run
  // arbitrarily later, or without EnableMfa ever having run at all; a TOTP
  // code stays computable from a stale secret indefinitely, so otp_code
  // matching and pending_version matching together still say nothing about
  // whether the enrollment window has closed. The caller's acknowledgment that
  // they saved the new codes, not a proof step on its own — but promotion is
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
    // The account's current password. Accepted for ChangePassword always,
    // and for a factor-touching method (DisableMfa, or EnableMfa/
    // ConfirmRecoveryCodes when replacing an existing factor) ONLY when the
    // account has no live MFA — the handler rejects it while
    // MFAConfig.OtpSecret is set, because ResetPassword mints a password from
    // mailbox possession alone, so accepting it against live MFA would let a
    // stolen session plus mailbox strip the factor (see Design →
    // Verification).
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
    // for it to mean that) OR the deployment is SaaS — bootstrap proof for an
    // account with nothing else yet, never a substitute for a factor that
    // already exists. The SaaS clause is an alternative path to eligibility,
    // not a replacement for the account-level check, and restores eligibility
    // for pre-existing Cloud/SSO accounts the LastChangePasswordTime backfill
    // migration would otherwise lock out permanently. It is deliberately NOT
    // the workspace's restriction.disallow_password_signin: principal has no
    // workspace column, so a workspace-scoped flag cannot speak for a global
    // account, and in a self-hosted multi-workspace deployment a user with a
    // real password in one workspace would become eligible via another that
    // happens to be SSO-only. SaaS is deployment-wide, forces that flag on
    // every workspace, and rejects ChangePassword outright, so no
    // caller-chosen password can exist there at all (see Design →
    // Verification). The handler checks this
    // eligibility itself; it is not just a
    // frontend choice of which field to show. The condition is never true for
    // DisableMfa, or for a rotation's ConfirmRecoveryCodes (both imply live
    // MFA already). It also is not the credential path for first-time
    // EnableMfa: that call omits credential entirely when the account has
    // neither a password nor live MFA (see EnableMfaRequest.credential), so
    // email_code is never consumed there. First-time ConfirmRecoveryCodes is
    // exactly where it's needed and used instead — that call still has no
    // live MFA at the time it runs, since promotion of the secret is deferred
    // to it precisely for this account shape (see Design → Verification) —
    // so email_code is only ever actually usable on ChangePassword or
    // first-time ConfirmRecoveryCodes.
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
    (google.api.resource_reference) = {type: "bytebase.com/User"},
    (buf.validate.field).string.max_len = 260
  ];
}

message ChangePasswordRequest {
  // Format: users/{email}. Must be the caller's own name — CredentialProof
  // proves control of *your* credential, which no one can supply for someone
  // else, so this is never valid on an admin-assisted call.
  string name = 1 [
    (google.api.field_behavior) = REQUIRED,
    (google.api.resource_reference) = {type: "bytebase.com/User"},
    (buf.validate.field).string.max_len = 260
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
    (google.api.resource_reference) = {type: "bytebase.com/User"},
    (buf.validate.field).string.max_len = 260
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
    (google.api.resource_reference) = {type: "bytebase.com/User"},
    (buf.validate.field).string.max_len = 260
  ];

  // The code computed from the enrollment StartMfaEnrollment returned.
  string otp_code = 2 [
    (google.api.field_behavior) = REQUIRED,
    (bytebase.v1.audit_behavior) = SENSITIVE,
    (buf.validate.field).string.max_len = 64
  ];

  // Proof for the *existing* factor being replaced, if any. Not the code
  // above — that proves the new enrollment, this proves the caller still
  // owns the account before the swap. Required whenever a reusable proof
  // exists — a rotation (there's a live factor to prove), or a first-time
  // enrollment where the account already has a password. Omitted only when
  // `email_code` would be the sole option: a first-time enrollment on an
  // account with neither a password nor live MFA — see Design → Verification
  // for why this call must not consume that code itself.
  CredentialProof credential = 3 [(google.api.field_behavior) = OPTIONAL];

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
    (google.api.resource_reference) = {type: "bytebase.com/User"},
    (buf.validate.field).string.max_len = 260
  ];

  // Required only when name is the caller's own. On an admin-assisted call
  // (name is someone else's), unset and unchecked — this is the one method
  // in this file with a real admin path, so its requirement is conditional
  // on caller identity, which a blanket proto REQUIRED can't express; the
  // conditionality is enforced in the handler, same as every other
  // identity-dependent check in this design. Also constrained by factor: on
  // a self-service call against an account with live MFA, the handler
  // accepts only otp_code/recovery_code, never current_password/email_code
  // (see CredentialProof.current_password and Design → Verification).
  CredentialProof credential = 2 [(google.api.field_behavior) = OPTIONAL];
}

message RegenerateRecoveryCodesRequest {
  string name = 1 [
    (google.api.field_behavior) = REQUIRED,
    (google.api.resource_reference) = {type: "bytebase.com/User"},
    (buf.validate.field).string.max_len = 260
  ];
}

message ConfirmRecoveryCodesRequest {
  string name = 1 [
    (google.api.field_behavior) = REQUIRED,
    (google.api.resource_reference) = {type: "bytebase.com/User"},
    (buf.validate.field).string.max_len = 260
  ];
  CredentialProof credential = 2 [(google.api.field_behavior) = REQUIRED];

  // Echoed from RegenerateRecoveryCodesResponse.pending_version. Must match
  // the account's current pending set exactly, or this is rejected — a
  // mismatch means a later RegenerateRecoveryCodes call (the caller's own,
  // in a second tab, or someone else's) has already superseded the set this
  // request thinks it's confirming.
  google.protobuf.Timestamp pending_version = 3 [(google.api.field_behavior) = REQUIRED];

  // Required only for first-time MFA enrollment (the account has no live
  // OtpSecret yet) — rejected if set otherwise. A *fresh* code, not the one
  // already submitted to EnableMfa — TOTP codes are only valid for one ~30s
  // period, and the recovery-code download screen in between routinely takes
  // longer than that, so resending the earlier value would just fail (see
  // Design → Verification and Frontend). Verified against the same pending
  // TempOtpSecret EnableMfa already validated, and promoted alongside the
  // recovery codes in this call rather than in EnableMfa, so the account is
  // never left MFA-required with zero usable recovery codes if this call
  // never arrives. Left unset for an ordinary rotation, where EnableMfa
  // already promoted the secret and this call only confirms recovery-code
  // receipt.
  string otp_code = 4 [
    (google.api.field_behavior) = OPTIONAL,
    (bytebase.v1.audit_behavior) = SENSITIVE,
    (buf.validate.field).string.max_len = 64
  ];
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
also serves LDAP bind; nothing reachable from the password-bearing methods here ever touches an
LDAP-bound principal. `otp_code`/`recovery_code`/`email_code` take `max_len = 64`, matching
`LoginRequest`'s existing fields of the same name. Each `name` field takes `max_len = 260` (`"users/"` +
a 254-char email; the workspace-wide 256-byte resource-name bound would reject a legal maximum-length
email), because every one of these methods turns `name` into a T9 lockout identity — the validate
interceptor must refuse an oversized identity before any handler claims a slot, the rule
`TestAuthEmailFieldsAreLengthBounded` already pins for every `email` field in `auth_service.proto`. That
test's inventory, and the audit-payload inventory `TestLintAuditPayloadInventory` pins, both have to gain
the seven new `name` fields, and `TestForbiddenClassMembership` has to gain all seven new procedures with
their `mcp_denial_reason`s — three enumerated lints that fail closed on anything unlisted, so they are
part of this change, not a follow-up.

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
shape once the OTP-specific part is factored out.

That "no old thing" framing for enrollment was itself the bug —
[caught in a later review round](https://github.com/bytebase/bytebase/pull/21235): rotation's safety
net is precisely that old thing — the previous live recovery codes keep working right up until
`ConfirmRecoveryCodes` runs, so a lost response or a closed tab costs nothing. First-time enrollment has
no such fallback by construction: nothing was live before this flow started, so if `EnableMfa` promotes
the secret and `ConfirmRecoveryCodes` never runs, the account is left MFA-required with zero usable
recovery codes — worse than the two problems the split was built to fix, not equivalent to them. This
is also the point `email_code` eligibility can't rescue: it requires no live `OtpSecret`
(see Design → Verification), which `EnableMfa` promoting the secret already ruled out. Fixed by
splitting `EnableMfa` itself along the same line as before, one level deeper: for a rotation (the
account already had a live `OtpSecret`), it promotes the secret as already described, unaffected. For
first-time enrollment, it verifies `otp_code` (and `credential`, when the account has a password to
prove — see EnableMfaRequest.credential) but promotes nothing — the account stays exactly as unenrolled
as before the call — and the actual promotion of *both* `TempOtpSecret` and `TempRecoveryCodes` moves to
`ConfirmRecoveryCodes`, atomically, gated on a fresh `otp_code` verified there
(`ConfirmRecoveryCodesRequest.otp_code`, required only in the enrollment case — see API → Messages).
A client that never gets a chance to call `ConfirmRecoveryCodes` after enrolling has, once
again, lost nothing: no secret went live, no recovery codes exist to lose track of, and a retried
`EnableMfa` against the same still-pending `pending_version` is exactly as harmless as the first attempt,
since it never wrote anything either. `EnableMfaRequest.pending_version` still guards `StartMfaEnrollment`
against being silently superseded before `EnableMfa` runs — general temp-state freshness, not
specifically a recovery-codes protection, and for enrollment now doubles as the token that ties
`EnableMfa`'s verification to `ConfirmRecoveryCodes`'s promotion of the exact same pending set.

Deferring promotion to `ConfirmRecoveryCodes` also deferred the expiry check without saying so —
[caught in a later round](https://github.com/bytebase/bytebase/pull/21235): `EnableMfa` still runs
`isMFATempSecretExpired` before it verifies `otp_code`, but for enrollment that check now protects
nothing, since `EnableMfa` writes no state `ConfirmRecoveryCodes` can see and the two calls can be
separated by however long the user takes to save their codes — or `ConfirmRecoveryCodes` can be called
directly, `EnableMfa` never having run at all. A TOTP code stays computable from a stale secret
indefinitely; only `isMFATempSecretExpired` catches a pending set that's simply gone stale with no
replacement minted, the same gap `pending_version` alone was already established not to cover
(see `EnableMfa` above). `ConfirmRecoveryCodes`'s enrollment branch runs the identical
`isMFATempSecretExpired` check itself, immediately before promoting, rather than trusting whatever
`EnableMfa` may or may not have already checked.

Requiring `otp_code` again at `ConfirmRecoveryCodes` only works if it's a *different* code from the one
`EnableMfa` already validated — [caught in a later round](https://github.com/bytebase/bytebase/pull/21235):
a TOTP code is only valid for roughly one 30-second period, and the time between the two calls is
however long the user takes to read, download, and put away their recovery codes on
`TwoFactorSetupPage` — routinely longer than that. Resubmitting the value the client already has from
the `EnableMfa` step (`TwoFactorSetupPage.tsx:146-160` stores it before advancing to the recovery-code
screen) would fail almost every time, not occasionally. The frontend needs a real second input here, not
a replay: the confirm step prompts for a fresh code from the authenticator app immediately before
submitting `ConfirmRecoveryCodes`, the same way `EnableMfa`'s own step did — a second, deliberate proof
of live possession, not friction for its own sake, since it's standing in for the same `otp_code` gate
`EnableMfa` already carries for the accounts where a rotation's version of this call needs none.

Requiring `credential` on both `EnableMfa` and `ConfirmRecoveryCodes` is fine for a rotation, or for
enrollment on an account that already has a password — both proofs are pure comparisons or a spendable-
but-plentiful recovery code, reusable or numerous enough to supply twice in one flow. It breaks
[precisely the account this whole mechanism exists for](https://github.com/bytebase/bytebase/pull/21235):
a passwordless first-time enrollment, where `email_code` is the *only* available proof. `EnableMfa`
already requires and atomically consumes one `REAUTH` code there — even though, per the fix above, it
promotes nothing — so `ConfirmRecoveryCodes`'s own required `credential` demands a second code
immediately afterward, and `RequestReauthCode` silently no-ops for its full resend cooldown
(`auth_service_email_code.go:392-393`), stalling the flow this design built `email_code` to unblock in
the first place. The fix isn't a longer cooldown or a second exemption on the resend path — it's not
asking twice: `EnableMfaRequest.credential` becomes optional, required only when a reusable proof exists
(a rotation, or enrollment where the account has a password); omitted when the account has neither a
password nor live MFA, since `email_code` would be the only option and this call doesn't mutate anything
in that branch anyway. The actual gate moves entirely to `ConfirmRecoveryCodes`, which already requires
`credential` unconditionally and is the point where promotion — the thing worth gating — actually
happens. `otp_code` still runs in `EnableMfa` regardless of branch: it isn't the credential proof, it's
confirmation the caller correctly configured the new device, and skipping it would leave a typo'd
authenticator app undetected until the user is already locked out at their next login.

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

One store-level message changes too — `storepb.UserProfile` (`proto/store/store/user.proto:38-47`) gains
the credential-generation stamp the MFA temp token is validated against (see Design → Verification). Field
6, not the reserved 4:

```protobuf
message UserProfile {
  // ... last_login_time = 1, last_change_password_time = 2, source = 3,
  //     reserved 4, last_login_workspace = 5 ...

  // Bumped by every mutation that revokes this account's sessions:
  // ChangePassword, EnableMfa, DisableMfa, ConfirmRecoveryCodes, ResetPassword,
  // and the admin-assisted password reset. Distinct from
  // last_change_password_time, which means "this account has had a
  // caller-chosen password" for email_code eligibility and must not move on
  // MFA-only changes.
  google.protobuf.Timestamp last_credential_change_time = 6;
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

Four conformance points where the proposal above still has to be brought into line with the repo's own
pinned conventions, none of which changes the design, all of which a lint or a reviewer will otherwise
catch:

- **`StartMfaEnrollment` returns a bare `MfaEnrollment`, which AIP-136 doesn't allow.** A custom method
  returns `<Rpc>Response` unless it returns an actual resource, and `MfaEnrollment` has no
  `google.api.resource` and no `name` — it is a response object, not a resource (the Alternatives section
  deliberately declines to model MFA as a `users/{email}/mfaFactors/{id}` collection). Rename it
  `StartMfaEnrollmentResponse`, matching `LoginResponse`/`ExchangeTokenResponse` in the same package.
- **Acronym casing: `Mfa` vs `MFA`.** Every acronym in the v1 protos is uppercase
  (`OIDCIdentityProviderContext`, `LDAPIdentityProviderConfig`, `AIChatMessage`), and the store side is
  already `MFAConfig`. `Mfa`/`EnableMfa`/`DisableMfa` would be the only lowercase-acronym names in the
  tree. Recommendation is to uppercase to `EnableMFA`/`DisableMFA`/`StartMFAEnrollment`/`MFAEnrollment`
  for consistency; flagged as a deliberate call rather than silently drifting, since it touches every
  MFA-named symbol and the HTTP verbs (`:enableMFA`).
- **`RESETS_CREDENTIAL` on `RequestReauthCode` contradicts the enum's own text**, which defines it as
  driving "the out-of-band reset flow that sets or delivers the secret a login accepts" — and the
  `REAUTH` code is deliberately *not* login-accepted. The enum comment in `annotation.proto` must be
  widened ("the secret a login *or a credential change* accepts") in the same change, or the reason is
  self-contradictory against `TestLintReasonsMatchTheClass`.
- **The prose paraphrases of `MINTS_CREDENTIAL`/`TAKES_OVER_ACCOUNT` don't match `annotation.proto`.**
  The real definitions are "puts a *token* for the caller's own principal in the response" and "rewrites
  an account's own credentials, which would let the session log in as that account" — nothing about
  whether the response body carries a secret. A not-yet-live TOTP secret or pending recovery-code set is
  neither a token nor a live credential, so `StartMfaEnrollment`/`RegenerateRecoveryCodes` fit
  `MINTS_CREDENTIAL` only once the enum comment is widened to name pending/enrollment material
  explicitly; do that alongside, since the reason-per-method mapping is lint-pinned.

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

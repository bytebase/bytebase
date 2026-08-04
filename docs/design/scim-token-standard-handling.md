# SCIM directory-sync token: move to the standard SaaS pattern

Status: proposal · 2026-08-04 · addresses T1 of `v1-api-audit-2026-08.md`

## Problem

`directory_sync_token` is a bearer credential stored in plaintext inside
`WorkspaceProfileSetting` — a blob that is deliberately readable by every workspace member
(`bb.settings.getWorkspaceProfile` is in the default member role,
`backend/store/predefined_roles.go:320`). The converter returns it verbatim
(`backend/api/v1/setting_service_converter.go:348`), so it reaches every logged-in user's
browser store via the profile fetch that all clients make for watermark and password rules.

It is also minted for **every** workspace at creation (`backend/store/workspace.go:136`) whether
or not SCIM is ever used, so every workspace carries a dormant live credential.

The cryptographic mechanics are already correct — UUIDv4 from `crypto/rand` (122 bits, satisfies
RFC 7644's entropy requirement) and `subtle.ConstantTimeCompare` at
`backend/api/directory-sync/webhook.go:852`. What is missing is the lifecycle model.

## What the standard is

RFC 7644 is permissive: bearer tokens MAY be used with TLS, MUST carry sufficient entropy, and
validation is explicitly out of scope. The convention among SaaS SCIM service providers is
consistent:

1. Admin generates the token in the app's console.
2. Shown **exactly once**, copy-to-clipboard; not retrievable afterwards.
3. Stored **hashed** (or encrypted); never returned by any read API.
4. Regenerating invalidates the previous token immediately.
5. Constant-time comparison on validation.
6. Scoped to SCIM only.

Non-expiring is common and acceptable; TTLs are best practice but silently break sync when they
lapse. OAuth 2.0 client credentials is the direction of travel, but Okta and Entra generally
support it only for pre-built gallery apps, not custom SCIM connections — so static bearer
remains correct for us today.

## Proposal

### 1. Hash at rest

Add `directory_sync_token_hash` (hex SHA-256) as **field 27** — free on both sides — to
`WorkspaceProfileSetting` in `proto/store/store/setting.proto`. Reserve field 15 there once
migrated. The SCIM handler hashes the presented token and compares against the stored hash,
still with `subtle.ConstantTimeCompare`.

Plain SHA-256, deliberately — not bcrypt/argon2. Slow KDFs exist to defend low-entropy
human-chosen secrets. This is a 122-bit random token; a slow KDF buys nothing and costs latency
on every SCIM request. Same reasoning behind GitHub PAT storage.

The hash is not exposed by any API. It is not a credential, so its presence in the blob is
harmless by construction — which is the property that makes this bug class unable to recur here.

### 2. Remove the read path from the v1 API

Delete `directory_sync_token` (field 15) from `proto/v1/v1/setting_service.proto`; reserve 15.
Add `bool directory_sync_token_configured` (field 27) so the UI can render state without the
secret.

New custom method on `WorkspaceService`:

```proto
// Mints a new directory sync token, invalidating the previous one.
// The plaintext token is returned exactly once and cannot be retrieved later.
rpc RotateDirectorySyncToken(RotateDirectorySyncTokenRequest) returns (RotateDirectorySyncTokenResponse) {
  option (google.api.http) = {
    post: "/v1/{name=workspaces/*}:rotateDirectorySyncToken"
    body: "*"
  };
  option (bytebase.v1.permission) = "bb.workspaces.rotateDirectorySyncToken";
  option (bytebase.v1.auth_method) = IAM;
  option (bytebase.v1.audit) = true;
}
```

`WorkspaceService` rather than `SettingService`: the token is a workspace-scoped credential, not
a setting, and moving it out of the settings blob is the point of the change.

**Permission.** A dedicated `bb.workspaces.rotateDirectorySyncToken` granted to workspace admin
only, rather than reusing `bb.settings.set`. Minting a credential that bypasses the v1 ACL
entirely is a strictly larger capability than editing SMTP config, and the audit already found
that conflating permissions is how this class of problem starts. Costs a role-definition change
and a regenerated frontend permission list.

### 3. Lazy generation

Drop `DirectorySyncToken: uuid.New().String()` from `backend/store/workspace.go:136`. No token
exists until an admin generates one. This eliminates the dormant-credential-on-every-workspace
problem and closes the "scraped while on Free, used after upgrading to Enterprise" vector, which
redaction alone does not.

### 4. Migration — non-breaking for live SCIM

The key property: **hash the existing plaintext in place**. The token value does not change, so
every configured Okta/Entra integration keeps working untouched. Only readability goes away.
Customers do not re-paste anything.

`backend/migrator/migration/3.22/0003##hash_directory_sync_token.sql`:

```sql
UPDATE setting
SET value = jsonb_set(
      value #- '{workspaceProfile,directorySyncToken}',
      '{workspaceProfile,directorySyncTokenHash}',
      to_jsonb(encode(sha256(convert_to(value->'workspaceProfile'->>'directorySyncToken', 'UTF8')), 'hex'))
    )
WHERE name = 'WORKSPACE_PROFILE'
  AND value->'workspaceProfile' ? 'directorySyncToken'
  AND value->'workspaceProfile'->>'directorySyncToken' <> '';
```

JSONB keys are protojson camelCase per AGENTS.md. Postgres has native `sha256(bytea)` — no
pgcrypto dependency. `TestLatestVersion` in `backend/migrator/migrator_test.go:20-21` must be
updated to `3.22.3` / this path.

### 5. Frontend

`AADSyncSheet.tsx` becomes show-once:

- No token configured → "Generate token" button → calls the rotate RPC → renders the plaintext
  once with copy-to-clipboard and an explicit "you will not see this again" warning.
- Token configured → masked placeholder + "Regenerate", reusing the existing confirmation
  warning already wired to the reset flow.

The sheet's entry point is already gated on an admin permission
(`UsersPage.tsx:928`), so no new gating is needed — but note that gating was cosmetic before,
since the data arrived regardless. After this change the gate is real.

### 6. Guards against recurrence

- `audit = true` on the rotate RPC.
- Regression test: a member-role caller's `GetSetting(WORKSPACE_PROFILE)` response contains no
  token field and no hash.
- A test that walks `WorkspaceProfileSetting`'s fields for secret-shaped names
  (`*token`, `*secret`, `*key`, `*password`) and fails if any is populated for a non-admin
  caller — so the next secret added to this blob is caught at CI rather than in an audit.
- Separately (out of scope here, tracked in the audit): SCIM mutations bypass the audit log
  entirely, which is why we cannot tell whether this was ever exploited.

## Remediation, and what this does not fix

Hashing in place stops **future** disclosure. It does not invalidate tokens already scraped —
anyone who read one still holds a valid credential.

Full remediation requires rotation, which must be admin-initiated. **Do not auto-rotate on
upgrade**: that would silently break every live Azure AD / Okta sync and convert a disclosure
into an outage. Ship instead:

- an admin banner recommending rotation, and
- release-note guidance to rotate.

Possible refinement if a reliable "SCIM never configured" signal exists: delete rather than hash
those tokens during migration, since rotating a token no IdP holds breaks nothing. Combined with
lazy generation this would clear the dormant credential from the large majority of workspaces
that never use SCIM.

## Rollout

Backend and frontend ship together (monorepo), since removing field 15 breaks the current UI
read. This is a **breaking** v1 API change — `breaking` label plus a `## Breaking Changes`
section per the pre-PR checklist:

- removed: `WorkspaceProfileSetting.directory_sync_token`
- added: `WorkspaceProfileSetting.directory_sync_token_configured`,
  `WorkspaceService.RotateDirectorySyncToken`

No API client outside the web UI consumes this field (`bytebase-action` does not), so external
impact should be nil.

**Sequencing.** If this lands in the next release, ship it as one change. If it slips a cycle,
ship the tourniquet first — redact the token in `GetSetting` for callers lacking
`bb.settings.get`, roughly a dozen lines plus a test, safe to backport to a patch release — and
follow with this.

## Estimate

| Piece | Size |
|---|---|
| Store + v1 proto, regenerate | S |
| Rotate RPC + permission + role wiring | M |
| SCIM handler hash comparison | S |
| Migration + `TestLatestVersion` | S |
| Lazy generation | S |
| Frontend show-once UX | M |
| Tests (regression + secret-shaped-field guard) | M |

One focused PR, or two if the tourniquet ships separately.

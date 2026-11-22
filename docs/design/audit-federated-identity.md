# Federated identity on audit rows

Status: implemented in this change · 2026-11-11 ·
[BYT-10494](https://linear.app/bytebase/issue/BYT-10494/workload-identity-does-not-record-which-workload-signed-in)

A workload identity lets a CI job sign in with its OIDC token. Bytebase checks the token's subject
against the identity's pattern and then discards it, so the audit log names the identity and never
the job behind it. With a pattern such as `repo:my-org/*`, every repository in the organization
signs in under one actor and the log cannot tell them apart.
**Each audit row a federated workload produces now carries the issuer and subject of the token it
signed in with, on the exchange and on every call made with the issued token.**

## Problem

Verified on `main` at `95bc3a4bd9`:

- **The claims are dropped.** `AuthService.ExchangeToken` calls `wif.ValidateToken` and ignores the
  claims it returns.
- **The issued token cannot carry them.** It holds the identity's email and workspace only.
- **The exchange row cannot show them.** `ExchangeTokenRequest.token` is `SENSITIVE`, so the row
  redacts it. The row also had no actor.
- **Failures say more than successes.** The subject appears only in the error text of a pattern
  mismatch.

## Goals

- **G1** A row names the external workload on the exchange and on every later call.
- **G2** A row stands alone. Reading it needs no join and no lookup of the identity's config.
- **G3** The actor, the existing filters and the existing stdout keys do not change.
- **G4** The recorded identity is never read for authorization.

### Non-goals

- **Job-level claims** such as GitHub's `run_id`. A subject names the repository and ref, not the
  run.
- **A per-token correlator** tying later rows to one exchange.
- **Refused exchanges.** They keep today's error text.

## Design

### The row

```proto
message AuditLog {
  // The external workload behind a workload identity. Set on a successful
  // token exchange and on every call made with the token it issued.
  FederatedIdentity federated_identity = 13;
}

// The verified OIDC identity a workload identity token was exchanged for.
// Recorded for audit and never read for authorization.
message FederatedIdentity {
  string issuer = 1;  // the token's "iss" claim
  string subject = 2; // the token's "sub" claim
}
```

This is the store proto. The v1 `AuditLog` carries the same message as `OUTPUT_ONLY` field 14. Both
values are copied verbatim, so a GitHub ID-based subject (`repo:my-org@123/app@456:...`) is stored
as sent. Presence of the message marks the row as federated, the way `mcp_delegation` marks MCP
rows.

The issuer is stored although one identity pins one issuer. `UpdateWorkloadIdentity` replaces the
whole config, so the identity is not a stable place to look up the issuer of an old row (G2).

The payload is JSONB, so there is no migration. Rows written before this change lack the key.

### The token

The exchange issues a token whose `act` claim holds the external `iss` and `sub`
(`GenerateWorkloadIdentityToken` in `backend/api/auth/tokens.go`). This is the RFC 8693 actor
claim: the token's own `sub` stays the identity's email, and `act` names the party acting as it.

There is one mint site, the token lives one hour and has no refresh path, so the claim reaches
every call the job makes.

### From token to row

- The auth interceptor copies the claim to `AuthContext.FederatedIdentity` for unary and streaming
  calls.
- `buildAuditRows` writes it to the row.
- The exchange itself carries no Bytebase token, so `ExchangeToken` sets it for its own row.
- A successful exchange row now names its actor, `workloadIdentities/{email}`. A refused one names
  none, because only success proves the caller is the identity it claimed.

### Reading it

| Surface | What it shows |
|---|---|
| API | `AuditLog.federated_identity` |
| Search | `federated_subject == "repo:my-org/app:ref:refs/heads/main"` |
| Stdout | `federated_issuer` and `federated_subject` |
| Console | The subject as a second line under the actor, with subject and issuer in a tooltip |

The console adds no column. The field is empty on almost every row, and on a workload identity's
rows the subject is the one value that differs, so it sits in view under the actor.

### Limits

- **Subject length.** An exchange is refused when the subject exceeds 1024 bytes
  (`maxSubjectLength` in `backend/plugin/idp/wif/wif.go`). The subject rides in every request
  header and every row.
- **Rolling upgrade.** A replica on the old version accepts the new token and writes rows without
  the field until it is upgraded.
- **No index.** The subject filter scans within the workspace and time range, as the MCP filters
  do.

## Alternatives

- **Flat `principal_subject` only.** Smallest change, but a subject is unique only within its
  issuer, and the issuer could not be recovered later (fails G2).
- **Two flat fields.** Same data. One identifier becomes two independent fields that can be
  half-set, and each later fact adds another top-level field.
- **One composite string**, issuer and subject joined. GitHub subjects contain `:` and `/`, so it
  needs escaping, and the `==`-only filter could not search by subject.
- **Google-style `authentication_info`.** The actor and `mcp_delegation` would stay outside it.
  Google's message has no issuer and nests this case under `serviceAccountDelegationInfo[]`.
- **Detail on the exchange row, a token ID on later rows.** "What did this repository do" becomes
  a join, and audit search has none (fails G2).
- **Fold the subject into the actor.** Breaks `actor ==` filters and the actor's mapping to a
  principal (fails G3).
- **Reuse `service_data`.** It holds one value, and `UpdateSetting` and `SetIamPolicy` already
  fill it.
- **Record on the exchange row only.** Callers under one wildcard identity stay indistinguishable
  on every later call (fails G1).
- **Allow exact subjects only**, as Azure does. Breaks every identity that uses a pattern.

## Not in this change

- **Export.** `ExportAuditLogs` writes eight fixed columns, so the field does not reach CSV, JSON
  or XLSX exports.
- **MCP.** A workload identity token used at `/mcp` loses the identity when the internal
  credential is minted.
- **Refused exchanges** do not record the issuer and subject.

## Reference

Products that let one principal admit many external workloads record the workload on each row.
Products that bind one external subject per principal record nothing extra. Surveyed 2026-11-10.

| Element | Same as |
|---|---|
| Issuer and subject together are the identifier | OpenID Connect Core §5.7, RFC 9493 `iss_sub` |
| The acting party carried as a token claim | RFC 8693 `act` |
| The external caller on every later row | Google `serviceAccountDelegationInfo[].principalSubject`, Kubernetes issuer-prefixed username |
| Issuer recorded beside the subject | AWS `identityProvider` and `userName` on `AssumeRoleWithWebIdentity` |
| Later rows without the caller need a join | AWS logs "Role identity only (no user)" unless `sourceIdentity` is set |
| One subject per principal, nothing extra recorded | Azure federated credentials match exactly; the sign-in records `federatedCredentialId` (Graph beta) |
| Dropping the issuer makes callers ambiguous | Google warns that colliding subjects are "indistinguishable in Cloud Audit Logs" |

Sources: [OpenID Connect Core 1.0](https://openid.net/specs/openid-connect-core-1_0.html) ·
[RFC 9493](https://www.rfc-editor.org/rfc/rfc9493) ·
[RFC 8693](https://www.rfc-editor.org/rfc/rfc8693) ·
[Google: example logs for Workload Identity Federation](https://docs.cloud.google.com/iam/docs/audit-logging/examples-workload-identity) ·
[Google: Workload Identity Federation best practices](https://docs.cloud.google.com/iam/docs/best-practices-for-using-workload-identity-federation) ·
[AWS: logging IAM and STS calls with CloudTrail](https://docs.aws.amazon.com/IAM/latest/UserGuide/cloudtrail-integration.html) ·
[AWS: source identity](https://docs.aws.amazon.com/IAM/latest/UserGuide/id_credentials_temp_control-access_monitor.html) ·
[Azure: workload identity federation considerations](https://learn.microsoft.com/en-us/entra/workload-id/workload-identity-federation-considerations) ·
[Microsoft Graph: signIn (beta)](https://learn.microsoft.com/en-us/graph/api/resources/signin?view=graph-rest-beta) ·
[Kubernetes: authenticating](https://kubernetes.io/docs/reference/access-authn-authz/authentication/).

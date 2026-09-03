# Login identity provider metadata

`AuthenticationInfo` — the anonymous message the login page already fetches — grows a hand-written
list of the providers that page renders, and `GetAuthenticationRestriction` is renamed
`GetAuthenticationInfo` to match what it now returns. `ListIdentityProviders` becomes an ordinary
IAM method behind a new `bb.identityProviders.list` permission, so LDAP host, bind DN, base DN and
user filter stop being readable without a credential. Closes T12 in
[`v1-api-audit-2026-08.md`](v1-api-audit-2026-08.md) /
[BYT-10156](https://linear.app/bytebase/issue/BYT-10156/listidentityproviders-leaks-ldap-config-to-anonymous-callers-t12),
which carries the line-level citations.

## Background

`ListIdentityProviders` is anonymous and takes a caller-supplied `parent`. Three journeys call it,
and all three receive the same admin record — every provider field except `client_secret` and
`bind_password`, the complete LDAP config included.

| Journey | Reachable by | Needs |
|---|---|---|
| Login page | anyone | which providers exist, what to call them, where to send the browser |
| SSO admin console | `bb.identityProviders.get` | the whole record |
| General settings | every Workspace Member | whether any provider exists |

### Problem

- **Anonymous callers read the directory topology.** Self-hosted needs no guessing — `GetWorkspace`
  is anonymous too and returns the singleton workspace ID — and self-hosted is where LDAP is
  configured. The login page never needed any of it: it reads an authorization endpoint, a client ID
  and scopes, and the LDAP bind happens server-side.
- **Redaction is a denylist.** Two `SECURITY:` blanks stand between the admin record and an
  anonymous caller. Every field added to the config is public the day it ships unless someone
  remembers to blank it — the failure mode
  [#21184](https://github.com/bytebase/bytebase/pull/21184) removed from `GetActuatorInfo`.
- **The third journey is why it is still anonymous.** An IAM gate on today's method would break
  General settings, whose route asks only for `bb.settings.getWorkspaceProfile` and
  `bb.policies.get` — both held by `workspaceMember`, the role every user receives on joining a
  workspace.

## Goals

- **G1** An anonymous caller learns only what the login page renders.
- **G2** Provider configuration is readable only with `bb.identityProviders.get`.
- **G3** A field added to the admin record stays private until someone publishes it on purpose.
- **G4** No journey loses access on upgrade — including custom roles that already reached the SSO
  console.
- **G5** One anonymous request serves the pre-login page, and the server resolves workspace versus
  global providers.

### Non-goals

- **Hiding that a workspace has SSO.** The login page renders "Continue with Okta" to anonymous
  visitors. Provider existence, type and branding are public by construction here and everywhere
  surveyed; only the configuration behind them is not.
- **The workspace-existence oracle** on `GetAuthenticationRestriction`, accepted in T12. An unknown
  workspace is refused before any provider is read, so the list adds no signal.
- **Changing LDAP sign-in, or who may open General settings.**

## Design

### The pre-login list rides on `AuthenticationInfo`

```proto
// A provider as the login page sees it. Not a view of IdentityProvider:
// every field here is published by hand.
message LoginIdentityProvider {
  string name = 1; // idps/{idp}
  IdentityProviderType type = 2;
  string title = 3;
  // Set for OAUTH2 and OIDC, absent for LDAP.
  AuthorizationRequest authorization_request = 4;
}

// What the browser needs to start the redirect, and nothing else.
message AuthorizationRequest {
  // The OAuth2 auth_url, or the authorization endpoint from the OIDC
  // issuer's discovery document.
  string endpoint = 1;
  string client_id = 2;
  repeated string scopes = 3;
}
```

`AuthenticationInfo` gains `repeated LoginIdentityProvider identity_providers`. That is exactly what
the browser puts in an authorization URL or on a button label; an LDAP entry carries name, type and
title, and the message has nowhere to put a host.

A separate message rather than a mask over `IdentityProvider` is what makes G3 structural: a new
LDAP attribute or OIDC field reaches no one until someone writes a line for it here.

The server returns the named workspace's providers, falling back to the global ones when it has
none — the fallback the client codes today, at the cost of a second anonymous round trip.

### `ListIdentityProviders` becomes an admin method

Drop the anonymous option and require `bb.identityProviders.list` under `auth_method = IAM`. The
permission is new: `bb.identityProviders.get` reads one provider, and a collection read is its own
verb. Workspace admin gains it in code; custom roles that already carry `.get` — the same roles that
could already reach the SSO console — gain it in a data migration, so no existing grant silently
stops working (G4).

`parent` stays. Identity providers are workspace children and AIP-132 names the parent, which is
also how every other workspace-scoped list here reads. What made `parent` dangerous was the
anonymous method behind it, not the field: the gate is what stops a caller reading another
workspace. The permission is checked against the credential's workspace, so the handler refuses a
`parent` naming any other workspace rather than quietly serving the caller's own — a request the
check never authorized.

Dropping the anonymous option is breaking.

### The method is named for what it returns

`GetAuthenticationRestriction` now returns restrictions *and* providers, so it becomes
`GetAuthenticationInfo`, matching its long-standing `AuthenticationInfo` response and the
`/v1/auth/info` path. It stays anonymous and workspace-named.

### General settings reads the bit, not the list

The disallow-password-signin checkbox is an affordance only: `UpdateSetting` already refuses the
write when no provider exists, and the checkbox is disabled anyway without
`bb.settings.setWorkspaceProfile`. So the page takes the length of
`AuthenticationInfo.identity_providers` — no permission a signed-in user lacks, and nothing newly
disclosed. It is also the truer signal: the guard means "an SSO login path exists", which is what
the login list enumerates rather than what the IdP table holds. The fetch is gated on self-hosted to
match the toggle it feeds; today it runs in SaaS too, where the setting cannot be changed.

## Alternatives

- **Redact `IdentityProvider` by caller.** Smallest diff, closes it today, leaves the denylist
  standing for the next field (fails G3).
- **A `view` enum or field mask on the one method (AIP-157).** Two shapes behind one name, where the
  anonymous shape is the default a mistake widens (fails G3).
- **A second anonymous RPC.** The same content under a better name, at the cost of another
  unauthenticated endpoint and another round trip (fails G5).
- **Grant `bb.identityProviders.get` to `workspaceMember`.** Makes today's method gateable by handing
  every member the LDAP config (fails G2).
- **Reuse `bb.identityProviders.get` for the list.** Adds no permission and needs no backfill, but
  collapses "read one provider" and "enumerate the collection" into one grant, which no other
  resource here does.
- **Drop `parent` and take the workspace from the token,** as #21184 did to `GetActuatorInfo`'s
  `name`. Safe, but it removes the parent from a child collection for a danger the IAM gate already
  answers, and leaves the list shaped unlike every other workspace-scoped list.
- **`bool sso_configured` on the workspace profile read.** Serves General settings with a permission
  members already hold, but builds a second path to a bit `AuthenticationInfo` carries and that is
  anonymous either way.

## Reference

Every product fronting an SSO login page draws the line in the same place: a public document holding
exactly what a client needs to begin the flow, and a gated API holding the configuration. Surveyed
2026-09-02.

| Element | Same as |
|---|---|
| Public metadata document, contents fixed by a spec | OIDC Discovery `/.well-known/openid-configuration`, RFC 8414 |
| A public API for untrusted front-ends, separate from the configuration API | Auth0 Authentication API vs. Management API |
| Provider configuration behind an admin role or scoped token | Keycloak `manage-identity-providers`, Okta `/api/v1/idps`, Grafana SSO Settings under RBAC |
| Secrets withheld even from authorized readers | Grafana returns `clientSecret` as `*********` |
| Directory bind configuration never client-facing | LDAP and AD binds are server-side in all of the above |

Sources: [OpenID Connect Discovery 1.0](https://openid.net/specs/openid-connect-discovery-1_0.html) ·
[RFC 8414](https://datatracker.ietf.org/doc/html/rfc8414) ·
[Auth0 Management API](https://auth0.com/docs/api/management/v2) ·
[Keycloak Admin REST API](https://www.keycloak.org/docs-api/latest/rest-api/index.html) ·
[Okta Identity Providers API](https://developer.okta.com/docs/reference/api/idps/) ·
[Grafana SSO Settings API](https://grafana.com/docs/grafana/latest/developer-resources/api-reference/http-api/api-legacy/sso-settings/).

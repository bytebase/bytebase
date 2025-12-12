# OAuth2 Server Design for Bytebase

## Overview

Add an OAuth2 Authorization Server to Bytebase enabling third-party applications (starting with MCP servers) to authenticate users and access Bytebase APIs on their behalf.

**Key decisions:**

- Authorization Code + PKCE flow (OAuth 2.1 compliant)
- Dynamic Client Registration for automated onboarding
- Tokens inherit full user permissions (no scope filtering)
- JWT access tokens with existing revocation cache
- Standard HTTP endpoints (`/oauth2/*`)
- All state persisted in database
- Clients expire after 30 days of inactivity

## Architecture

```
┌─────────────────┐     ┌──────────────────────────────────────────┐
│  Third-Party    │     │              Bytebase                    │
│  App (MCP)      │     │                                          │
│                 │     │  ┌─────────────────────────────────────┐ │
│  1. Register ───────────▶│ POST /oauth2/register               │ │
│     client      │     │  │ (Dynamic Client Registration)       │ │
│                 │     │  └─────────────────────────────────────┘ │
│                 │     │                                          │
│  2. Redirect ───────────▶│ GET /oauth2/authorize               │ │
│     user        │     │  │ (Shows consent, redirects back)     │ │
│                 │     │  └─────────────────────────────────────┘ │
│                 │     │                                          │
│  3. Exchange ───────────▶│ POST /oauth2/token                  │ │
│     code        │     │  │ (Returns JWT access + refresh)      │ │
│                 │     │  └─────────────────────────────────────┘ │
│                 │     │                                          │
│  4. Call API ───────────▶│ Existing gRPC/REST APIs             │ │
│     Bearer token│     │  │ (Auth interceptor validates JWT)    │ │
└─────────────────┘     └──────────────────────────────────────────┘
```

## Endpoints

| Endpoint | Method | Purpose |
|----------|--------|---------|
| `/.well-known/oauth-authorization-server` | GET | Server metadata discovery (RFC 8414) |
| `/oauth2/register` | POST | Dynamic Client Registration (RFC 7591) |
| `/oauth2/authorize` | GET | Authorization endpoint - shows consent UI |
| `/oauth2/authorize` | POST | User grants/denies consent |
| `/oauth2/token` | POST | Token endpoint - code exchange, refresh |
| `/oauth2/revoke` | POST | Token revocation (RFC 7009) |

**Discovery metadata** (`/.well-known/oauth-authorization-server`):

```json
{
  "issuer": "https://bytebase.example.com",
  "authorization_endpoint": "https://bytebase.example.com/oauth2/authorize",
  "token_endpoint": "https://bytebase.example.com/oauth2/token",
  "registration_endpoint": "https://bytebase.example.com/oauth2/register",
  "revocation_endpoint": "https://bytebase.example.com/oauth2/revoke",
  "response_types_supported": ["code"],
  "grant_types_supported": ["authorization_code", "refresh_token"],
  "code_challenge_methods_supported": ["S256"],
  "token_endpoint_auth_methods_supported": ["client_secret_basic", "client_secret_post"]
}
```

**Not included initially** (can add later):

- `/oauth2/introspect` - Token introspection for external resource servers
- `/oauth2/userinfo` - OIDC userinfo endpoint
- `/oauth2/jwks` - Public keys (if switching to asymmetric signing)

## Authorization Flow (Authorization Code + PKCE)

**1. Client Registration** (one-time)

```
POST /oauth2/register
Content-Type: application/json

{
  "client_name": "My MCP Server",
  "redirect_uris": ["http://localhost:3000/callback"],
  "grant_types": ["authorization_code", "refresh_token"],
  "token_endpoint_auth_method": "client_secret_basic"
}

Response:
{
  "client_id": "bb_oauth_xxxxxxxx",
  "client_secret": "bb_secret_xxxxxxxx",
  "client_name": "My MCP Server",
  "redirect_uris": ["http://localhost:3000/callback"],
  ...
}
```

**2. Authorization Request**

```
GET /oauth2/authorize?
  response_type=code&
  client_id=bb_oauth_xxxxxxxx&
  redirect_uri=http://localhost:3000/callback&
  state=random_state&
  code_challenge=BASE64URL(SHA256(verifier))&
  code_challenge_method=S256
```

User sees consent screen, logs in if needed, approves.

**3. Authorization Response**

```
HTTP/1.1 302 Found
Location: http://localhost:3000/callback?code=AUTH_CODE&state=random_state
```

**4. Token Exchange**

```
POST /oauth2/token
Content-Type: application/x-www-form-urlencoded
Authorization: Basic base64(client_id:client_secret)

grant_type=authorization_code&
code=AUTH_CODE&
redirect_uri=http://localhost:3000/callback&
code_verifier=ORIGINAL_VERIFIER
```

**5. Token Response**

```json
{
  "access_token": "eyJhbG...",
  "token_type": "Bearer",
  "expires_in": 3600,
  "refresh_token": "bb_refresh_xxxxxxxx"
}
```

## Token Design

**Access Token (JWT)**

Same structure as existing Bytebase access tokens for consistency:

```json
{
  "iss": "bytebase",
  "sub": "users/user@example.com",
  "aud": "bb.oauth2.access",
  "exp": 1702300800,
  "iat": 1702297200,
  "jti": "unique-token-id",
  "client_id": "bb_oauth_xxxxxxxx"
}
```

Key points:

- Signed with HS256 using existing signing key
- `aud` distinguishes OAuth2 tokens from regular session tokens (`bb.user.access` vs `bb.oauth2.access`)
- `client_id` claim identifies which OAuth2 client issued the request
- `jti` enables revocation tracking in existing LRU cache
- Default expiry: 1 hour

**Refresh Token**

Opaque token stored in database:

- Format: `bb_refresh_xxxxxxxx` (random, not JWT)
- Linked to user and client
- Longer expiry: 30 days
- Single-use: rotated on each refresh

**Validation**

The existing auth interceptor in `backend/api/auth/auth.go` will be extended to:

1. Accept `bb.oauth2.access` audience
2. Look up user from `sub` claim (same as today)
3. Check revocation cache using `jti`
4. User's full permissions apply (no scope filtering)

## Database Schema

**`oauth2_client`**

```sql
CREATE TABLE oauth2_client (
    client_id TEXT PRIMARY KEY,
    client_secret_hash TEXT NOT NULL,
    config JSONB NOT NULL,
    last_active_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
```

**`oauth2_authorization_code`**

```sql
CREATE TABLE oauth2_authorization_code (
    code TEXT PRIMARY KEY,
    client_id TEXT NOT NULL REFERENCES oauth2_client(client_id) ON DELETE CASCADE,
    user_id INTEGER NOT NULL REFERENCES principal(id),
    config JSONB NOT NULL,
    expires_at TIMESTAMPTZ NOT NULL
);
```

**`oauth2_refresh_token`**

```sql
CREATE TABLE oauth2_refresh_token (
    token_hash TEXT PRIMARY KEY,
    client_id TEXT NOT NULL REFERENCES oauth2_client(client_id) ON DELETE CASCADE,
    user_id INTEGER NOT NULL REFERENCES principal(id),
    expires_at TIMESTAMPTZ NOT NULL
);
```

**Proto definitions:**

```protobuf
message OAuth2ClientConfig {
  string client_name = 1;
  repeated string redirect_uris = 2;
  repeated string grant_types = 3;
  string token_endpoint_auth_method = 4;
}

message OAuth2AuthorizationCodeConfig {
  string redirect_uri = 1;
  string code_challenge = 2;
  string code_challenge_method = 3;
}
```

**Cleanup:**

- Background job deletes clients where `last_active_at < now() - 30 days`
- `last_active_at` updated when authorization code is created or token is issued/refreshed
- Expired authorization codes and refresh tokens cleaned up based on `expires_at`

## Backend Structure

```
backend/
├── api/
│   └── oauth2/
│       ├── oauth2.go           # HTTP handlers for /oauth2/* endpoints
│       ├── discovery.go        # /.well-known/oauth-authorization-server
│       ├── register.go         # POST /oauth2/register (DCR)
│       ├── authorize.go        # GET/POST /oauth2/authorize
│       ├── token.go            # POST /oauth2/token
│       └── revoke.go           # POST /oauth2/revoke
├── store/
│   ├── oauth2_client.go        # CRUD for oauth2_client
│   ├── oauth2_authorization_code.go
│   └── oauth2_refresh_token.go
└── proto/
    └── store/
        └── oauth2.proto        # OAuth2ClientConfig, OAuth2AuthorizationCodeConfig
```

**Registration in server:**

- Mount OAuth2 handlers on the HTTP mux alongside existing routes
- Handlers use existing `store.Store` for database access
- Reuse existing auth interceptor logic for validating user sessions during `/oauth2/authorize`

## Security Considerations

**PKCE (required)**

- `code_challenge_method` must be `S256` (reject `plain`)
- Validate `code_verifier` matches `code_challenge` during token exchange

**Client secrets**

- Store bcrypt hash, never plaintext
- Generate with sufficient entropy (32+ bytes, base64 encoded)

**Redirect URI validation**

- Exact match against registered URIs (no wildcards)
- Must use HTTPS in production (allow `http://localhost` for development)

**Token security**

- Authorization codes: single-use, 10-minute expiry
- Refresh tokens: single-use (rotate on refresh), 30-day expiry
- Access tokens: 1-hour expiry

**Rate limiting**

- Apply existing rate limiting to `/oauth2/token` endpoint
- Prevent brute-force on authorization codes

**CSRF protection**

- `state` parameter required on authorization requests
- Validate `state` matches on callback

## Future Enhancements

- Admin UI for managing OAuth2 clients
- Token introspection endpoint for external resource servers
- OIDC userinfo endpoint
- Asymmetric signing (RS256) with JWKS endpoint
- Scope-based permission filtering

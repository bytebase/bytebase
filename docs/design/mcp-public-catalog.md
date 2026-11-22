# MCP public tool catalog

Status: proposal · 2026-11-22 ·
[BYT-10055](https://linear.app/bytebase/issue/BYT-10055) · rationale:
[`bytebase/bytebase-mcp` `docs/DESIGN.md`](https://github.com/bytebase/bytebase-mcp/blob/byt-10055-pure-proxy/docs/DESIGN.md)
(on its branch until it merges; `main` still describes the bundled tool list it replaces)

The npm package `@bytebase/bytebase-mcp` lets MCP directories and local-command clients use the MCP
server built into Bytebase. It keeps no copy of the tool list, so it needs to read the list before
anyone signs in. Its design doc owns the rationale and the alternatives; this doc covers only the
server change. **`GET /mcp/catalog` returns the tool list and the routing instructions to anyone.
It is built once from the live tool registry and reads nothing at request time. It is the only
path of the MCP package that answers without a token, and `/mcp` keeps its 401.**

## Problem

Verified on `main` at `47022b702e`:

- **Nothing about MCP is readable before sign-in.** `RegisterRoutes` registers one route,
  `e.Any("/mcp", …, s.authMiddleware)`, in `backend/api/mcp/server.go`. Every request without a
  valid bearer gets 401 with a `WWW-Authenticate` header carrying `resource_metadata`
  (`unauthorized`). The handler behind it is wrapped in `mcpauth.RequireBearerToken`.
- **The tool list is not private.** It is the same for every workspace and user: the comment on
  `registerTools` explains why it is not filtered by the workspace's policy (BOT-89). Tool
  registration reads no profile, setting or store. The source is public.
- **The instructions have a per-session part.** `serverInstructions` is constant.
  `nameSignedInAccount` prepends a line naming the session's account and workspace, and
  `askInChatWithoutForm` appends `askInChatInstructions` for a client that declared no form
  elicitation.
- **An unknown path returns the console page.** The frontend's static middleware
  (`registerFrontendRoutes`, `backend/server/server_frontend_routes.go`) serves `index.html` with
  `200` for any router-level 404 outside `/api`, `/v1`, `/bytebase.v1`, `/.well-known` and `/hook`.
  Bytebase Cloud answers `GET /mcp/catalog` that way today: `200 text/html`.

## Principle

> Publish what the source code already publishes, and nothing an instance knows about itself.

## Decisions

**D1 · Path: `GET <external URL>/mcp/catalog`.** The MCP package already owns `/mcp`, and the
route sits beside it. A reverse proxy that forwards `/mcp` by prefix forwards this path too, so
an operator who exposed MCP exposes the catalog without a new rule. The MCP Server Card extension
(SEP-2127, `modelcontextprotocol/ext-server-card`) reserves `<streamable-http-url>/server-card`
for its document, so the ecosystem is already putting discovery documents under the endpoint
path. Rejected:
- `/.well-known/…`: well-known names are meant to be registered, and an MCP name there is
  unsettled. The earlier Server Card draft (SEP-1649) used one and was superseded.
- `GET /mcp` without a token: clients rely on that 401 to start OAuth (D5).
- A path under `/api`: those are OAuth and Connect routes, with their own conventions.

**D2 · Response shape.** `200`, `Content-Type: application/json`:

```json
{
  "instructions": "<serverInstructions>",
  "instructionsWithoutForm": "<askInChatInstructions>",
  "tools": [ /* exactly the tools a session's tools/list returns */ ]
}
```

`tools` holds the same objects, in the same order, as a signed-in session's `tools/list` from this
binary. `instructionsWithoutForm` is the paragraph the server appends for a client without form
elicitation. A proxy whose client has no form shows it to the model before anyone signs in; under
Read-write it is what makes the model confirm writes. The shape only grows: a client ignores
fields it does not know, and a breaking change takes a new path. Rejected: a JSON-RPC envelope.
It would wrap one array in ceremony nobody parses.

**D3 · Built once, from the registry, at construction.** After `registerTools`, the server opens
a session to itself over the SDK's in-memory transport, lists its tools, and serializes the
document. The handler writes those bytes. Since the list comes from the same registry a session
reads, the catalog cannot drift from `tools/list`. A failure to build fails `NewServer`, so a
broken registry fails at startup. Rejected: a generated `tools.json` checked into the repository.
That is a copy, and keeping copies current is the problem this removes.

**D4 · No request-time input.** The handler ignores the query, headers, cookies and body, and calls
no store, setting, license or network. That makes the response the same on every instance running
the same binary, and makes an unauthenticated request cost about as much as `/healthz`. **The
response never contains** an account, user or workspace (ID or name), the external URL or any host
name, a setting or policy value (including the workspace's MCP access policy), license or plan
data, a session ID, or the server version. The version is left out because the RPC that reports
it, `ActuatorService.GetActuatorInfo`, requires a signed-in user (`actuator_service.go`).

**D5 · `/mcp` and its 401 handshake do not change.** MCP clients learn where to sign in from the
401's `resource_metadata` pointer (RFC 9728). Answering anything else on `/mcp` without a token
would break sign-in for every client. The new route is registered separately, does not pass
through `authMiddleware`, and never reaches the streamable handler, so it cannot open a session or
run a tool.

**D6 · Caching.** `Cache-Control: public, max-age=3600`. The content changes only when the binary
does, so an HTTP cache serves a pre-upgrade copy for at most an hour. A proxy that already
holds a catalog keeps it until a session opens; how it then catches up is the main doc's concern.
No ETag, compression or CORS of its own. No browser page reads the catalog, and adding no CORS
keeps another site's scripts from reading it with a visitor's cookies, even though it holds
nothing secret. In dev mode the global CORS middleware (`configureEchoRouters`) covers it as it
covers every route.

**D7 · SaaS and self-hosted behave the same.** The route needs no workspace, so Cloud, one host
serving every workspace, answers at its root exactly as a self-hosted instance does. It works
before an external URL is set; `/mcp` needs one for token audiences, the catalog does not. It
answers even when a workspace's MCP access policy is Disabled. The catalog describes the product,
not a workspace, and the policy's promise, "No MCP session can connect to this workspace"
(`mcp-capability-ladder.md`), stays true: the catalog opens no session. Rejected: an admin switch
to turn it off. It would add a store read to an unauthenticated path to hide what the public
source already shows.

**D8 · Server Card, later.** A Server Card leaves tool lists out on purpose, so it cannot replace
the catalog. It describes identity and transport: name, version, endpoint URLs, protocol
versions. If Bytebase serves one once the extension leaves Experimental (its repository still says
so, and its `$schema` URL returns 404 as of 2026-11-16), it would go at `/mcp/server-card`, built
the same way. Its required `version` field would reopen D4's decision on disclosing the version.

## Least privilege

**Rule: of the routes the MCP package registers, exactly one answers without a bearer token,
`GET /mcp/catalog`. This change adds that one and no other. Every other registered route gets 401
with the `resource_metadata` pointer, or, as today, 503 when no external URL is configured
(`unauthorized`; `TestMCPAuthFailsExplicitlyWithoutExternalURL`).**

| Request | Before | After |
| --- | --- | --- |
| Any method on `/mcp`, no token | 401 | 401 |
| `GET /mcp/catalog`, with or without a token | 200 console page (fallback) | 200 catalog |
| `HEAD`, `POST` or another method on `/mcp/catalog` | 200 console page (fallback) | 405 (`HEAD` empty, others a JSON error body) |
| `OPTIONS /mcp/catalog` | 200 console page (fallback) | 204 with `Allow`, Echo's automatic answer |
| Any method on `/mcp/<anything else>` | 200 console page (fallback) | 200 console page (fallback) |

Probed with Echo v5.4.4 and `registerFrontendRoutes` on an in-memory dist. The fallback rows are
the frontend's existing behavior for unknown paths; they expose no MCP data, and this change
leaves them alone. The rest of the server's unauthenticated surface,
including OAuth discovery and `/api/oauth2/*` (`backend/api/oauth2`), RPCs marked
`allow_without_credential`, `/healthz` and the console's static files, is governed elsewhere and
does not change.

A test enforces the rule by enumeration rather than by a list of paths. It registers the MCP
package's routes on a fresh Echo instance with an external URL configured, walks
`e.Router().Routes()`, expands Echo's any-method entry (how `e.Any` is listed) into every HTTP
method, sends each method and path without a token, and requires 401 with `resource_metadata`
everywhere except `GET /mcp/catalog`. A route added later without auth fails the test whatever its path.

## States

| Request to `/mcp/catalog` | Response |
| --- | --- |
| `GET`, on a release with the route | 200, the catalog (D2), identical for every caller |
| `GET`, on a release without the route | 200 `text/html`, the console page. Clients must check the content type and the `tools` array, not only the status. |
| Any other method | 405; `OPTIONS` gets 204 with `Allow` |
| A server whose registry fails to build | Does not start (D3) |

## Scope

One pull request, which carries this doc.

1. **Catalog route.** Delivers D1–D7 and the least-privilege rule. Files:
   `backend/api/mcp/catalog.go` (new: build and handler), `backend/api/mcp/server.go`
   (`newServerWithIdleTimeout` builds it, `RegisterRoutes` registers it),
   `backend/api/mcp/catalog_test.go` (new). Tests:
   - the catalog's `tools` equal a signed-in session's `tools/list` (`listToolsOverRoute`), and its
     two instruction fields equal `serverInstructions` and `askInChatInstructions`;
   - D4: built over a store fake that fails every call, the catalog is still served; responses
     with and without an `Authorization` header are byte-identical; the body contains neither the
     profile's version nor its external URL; a server with no external URL configured still
     serves it;
   - the least-privilege enumeration;
   - the frontend fallthrough: in `backend/server`, with `registerFrontendRoutes` on an in-memory
     dist, `GET /mcp/catalog` reaches the route, not `index.html`, and the other methods on it
     answer as the least-privilege table says.

   Depends on nothing. The npm package's first publish waits until Cloud serves this route.

## Not in this PR

- A Server Card at `/mcp/server-card` (D8).
- Tool annotations (`readOnlyHint` and the rest). Directory scores and vendor preflights read
  them, and the catalog will carry them once tools set them. Whether `query_database` gets static
  or mode-aware hints is still open.
- The npm snippet on the console's MCP page (`frontend/src/routes/workspace/MCPPage.tsx`).
- Returning a real 404 for unknown `/mcp/…` paths instead of the console page.

# Page agent module

Follow `../../../AGENTS.md`. This file adds rules for the in-product page agent under `src/modules/agent/`.

## Ownership map

```text
src/modules/agent/
├── components/       # React window, chat, input, tool cards, agent-owned overlays
├── dom/              # DOM snapshots and browser interaction primitives
├── logic/
│   ├── agentLoop.ts  # Tool-call loop and retry behavior
│   ├── context.ts    # Page and application context extraction
│   ├── prompt.ts     # System prompt
│   ├── skills/       # On-demand workflow guides exposed by get_skill
│   └── tools/        # API, navigation, page-state, and DOM tool implementations
├── store/            # Zustand state for the agent window and conversation
├── index.tsx         # Public module exports
└── window.ts         # Window-level integration
```

Keep agent-only UI and state inside this module. Shared application primitives still come from `@/components/ui`, but agent overlays must use the wrappers in `components/ui/` so they mount in the `agent` layer.

## Tool behavior

- `search_api` discovers endpoints and request/response shapes from the generated OpenAPI index.
- `call_api` executes Bytebase API requests.
- Protected API calls require their normal one-time permission. Never automatically repeat a protected POST after session refresh, cancellation, or response loss.
- Local receipts describe observed outcomes and can warn about a repeated payload. Missing receipts and earlier uncertainty do not authorize or block new work.
- `navigate` uses the React Router instance from `@/app/router` and can list registered route patterns.
- `get_page_state` returns semantic application context or a DOM snapshot.
- `dom_action` interacts with elements from the latest DOM snapshot.
- `get_skill` loads workflow guidance from `logic/skills/`.

### API response recovery

`logic/apiResponse.ts` interprets observed failures, not entitlement preflight.
Generic API dispatch and typed API error adapters use the same classifier.
Paid-feature advice requires an explicit recognized backend error; a 403 or
unavailable overview alone is not subscription evidence. Typed readers that need
recovery guidance should use `agentApiErrorResult` before losing their Connect
error details. Keep intentional unavailable facets local.

Typed failures keep fallback text and protocol code in `apiError`, not fabricated
HTTP status, raw diagnostic prose, or unrelated details. Client errors do not prove
that a server responded or a request was dispatched. Local validation and target
clarification stay domain-owned. `apiResponseGuidance.ts` supplies a common fallback
plus protocol-specific advice only for API failures or validated uncertain execution.
Execution metadata must satisfy the shared envelope validator before specialized
advice. A later matching request cannot resolve an earlier attempt's possible or
partial effects. Shared safety rules cover all prompt modes. Diagnostic chats are
read-by-default: protected APIs use the common exact-request one-time approval
flow after positive project-scope admission, with fresh scope and identity checks
at approval and dispatch. Direct SQL execution and export are unavailable until
their effective database targets can be verified within the diagnostic project.
Recovery advice never authorizes a write, grants access,
or permits automatic retries. Diagnostic DOM, navigation, and draft tools remain
unavailable.

Current-turn recovery guidance reuses the existing query Access Grant workflow,
including fresh eligibility and member consent. Recovery metadata never authorizes
an action, changes execution evidence, proves rollback, or permits automatic retry.

DOM refs such as `e1` are local to one `get_page_state(mode="dom")` snapshot. Refresh the snapshot after navigation or any DOM-changing action; never describe refs as durable identifiers.

### Registered DOM actions

`dom_action` has no raw click, input, select, keyboard, or event-dispatch fallback.
Only an exact element-and-verb binding registered by its mounted route owner is
eligible. Shared control types and ancestor elements do not grant authority.
Bindings remain scoped to the current actor, Workspace, Project, chat, run,
route generation, resource, and owner revision.

Inspect and local-draft bindings explicitly declare their effect. A local setter
must not trigger persistence, autosave, focus changes, or delayed side effects.
Persistent bindings prepare exactly one API request without executing it. That
exact request uses the common scope, IAM, and existing Allow once/Deny permission
path, with fresh owner checks at approval and the final dispatch boundary. These
guards use local execution identity and do not depend on a subscription preflight
service. Deny keeps the unsaved draft; a known successful write synchronizes the
matching owner without replaying the action.

Database labels and environment are the supported persistent route bindings.
Their permission preview uses the runtime-owned canonical resource and field
diff, never the model's description. Local Plan statement editing uses an
isolated, unsaved draft. Unsupported controls remain manual, including autosave
surfaces, SQL Editor, Plan Save/Create, multi-request workflows, and destructive
actions. Registration is explicit, not automatic support for the whole console.
Native human interactions are unchanged.

## OpenAPI index

Run:

```bash
pnpm --dir frontend run prepare
```

The generator reads `backend/api/mcp/gen/openapi.yaml` and writes `logic/tools/gen/openapi-index.ts`. Do not edit the generated file manually.

`search_api(query="...")` searches generated OpenAPI metadata and returns a
small ranked endpoint set. The generated index remains the endpoint source of
truth; do not duplicate a service catalogue in `logic/prompt.ts`.

## Extending the agent

- Add a workflow guide in `logic/skills/` and register it in `logic/skills/index.ts`.
- Add tool implementation and tests under `logic/tools/`, then register it in `logic/tools/index.ts`.
- Keep API discovery in `searchApi.ts`; update its semantic-search tests when
  changing generated endpoint metadata.
- Use the Zustand store in `store/agent.ts`; do not introduce a second state system.
- Run focused Vitest for the touched agent subtree, then the standard frontend verification sequence.

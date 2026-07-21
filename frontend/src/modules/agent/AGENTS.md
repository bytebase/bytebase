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
│   ├── prompt.ts     # System prompt and maintained service directory
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
- `navigate` uses the React Router instance from `@/app/router` and can list registered route patterns.
- `get_page_state` returns semantic application context or a DOM snapshot.
- `dom_action` interacts with elements from the latest DOM snapshot.
- `get_skill` loads workflow guidance from `logic/skills/`.

DOM refs such as `e1` are local to one `get_page_state(mode="dom")` snapshot. Refresh the snapshot after navigation or any DOM-changing action; never describe refs as durable identifiers.

## OpenAPI index

Run:

```bash
pnpm --dir frontend run generate:openapi-index
```

The generator reads `backend/api/mcp/gen/openapi.yaml` and writes `logic/tools/gen/openapi-index.ts`. Do not edit the generated file manually.

`logic/prompt.ts` contains a compact, manually maintained `serviceDirectory`. Update it when the generator reports uncovered services or when discovery terminology is insufficient. Keep descriptions short and user-facing; the generated index remains the endpoint source of truth.

## Extending the agent

- Add a workflow guide in `logic/skills/` and register it in `logic/skills/index.ts`.
- Add tool implementation and tests under `logic/tools/`, then register it in `logic/tools/index.ts`.
- Keep API discovery in `searchApi.ts`; do not hardcode broad endpoint inventories into the prompt.
- Use the Zustand store in `store/agent.ts`; do not introduce a second state system.
- Run focused Vitest for the touched agent subtree, then the standard frontend verification sequence.

# Page Agent — Maintenance Guide

This document explains how the page agent works and how to maintain it. It is intended for both humans and LLMs.

## Architecture

```
frontend/src/plugins/agent/
├── components/          # Vue UI (AgentWindow, message rendering)
├── dom/                 # DOM interaction layer
│   ├── actions.ts       # Execute DOM actions (click, input, select, scroll)
│   └── domTree.ts       # Extract and index interactive DOM elements
├── logic/
│   ├── agentLoop.ts     # Main agent loop: tool calls, retries, message history
│   ├── context.ts       # Extract page context from Pinia stores
│   ├── prompt.ts        # System prompt construction
│   ├── types.ts         # Shared types
│   ├── skills/          # On-demand workflow guides
│   │   ├── index.ts     # Skill registry and get_skill handler
│   │   ├── query.ts
│   │   ├── databaseChange.ts
│   │   └── grantPermission.ts
│   └── tools/           # Tool implementations
│       ├── index.ts     # Tool definitions and executor dispatcher
│       ├── callApi.ts   # Execute API endpoints
│       ├── searchApi.ts # API discovery with search + alias map
│       ├── navigate.ts  # Router navigation + route listing
│       ├── pageState.ts # Page context (semantic or DOM)
│       ├── domAction.ts # DOM action wrapper
│       └── gen/         # Generated API discovery artifacts
│           └── openapi-index.ts     # Generated API endpoint and schema index
└── store/               # Pinia store for agent state
```

## Tools Available to the Agent

| Tool | Purpose |
|------|---------|
| `search_api` | Discover API endpoints, browse services, get request/response schemas |
| `call_api` | Execute a Bytebase API endpoint |
| `navigate` | Navigate to a page or list available routes |
| `get_page_state` | Get current page context (route, stores, or DOM tree) |
| `dom_action` | Interact with DOM elements (click, input, select, read, scroll) |
| `get_skill` | Load step-by-step workflow guides for multi-step tasks |

## Tool Selection Philosophy

The agent chooses between DOM and API based on page context:
- **DOM-first** on form/preview/editor/creation pages — these have unsaved state that only exists in the UI
- **API-first** for data queries, bulk operations, or when the relevant page is not open
- **Either** for mutations on persisted resources — DOM if the user benefits from seeing it, API for speed

## Maintaining the Service Directory

**What:** A compact, feature-grouped summary of API services. It is embedded directly in `logic/prompt.ts` so the agent can choose the right service quickly before using `search_api` for exact endpoint details.

**Where:** `logic/prompt.ts` (`serviceDirectory` constant)

**Source of truth:**
- The `serviceDirectory` string in `logic/prompt.ts` is a **checked-in, manually maintained** prompt aid.
- `openapi-index.ts` and `search_api` remain the actual API discovery source of truth.

**When to update:**
- After adding or removing an API service
- After major endpoint additions that materially change what a service covers
- When `pnpm --dir frontend run generate:openapi-index` warns about uncovered services
- When the page agent consistently picks the wrong service because the descriptions are stale or missing key synonyms

**How to update:**
1. Run `pnpm --dir frontend run generate:openapi-index`.
2. If the script warns about missing services, edit the `serviceDirectory` block in `logic/prompt.ts` manually.
3. Keep each service description to one line.
4. Use user-facing feature names and useful synonyms, not raw method lists.
5. Keep the category structure compact and stable; do not try to enumerate every endpoint.
6. Preserve the surrounding prompt structure and the `serviceDirectory` constant name.
7. Review the resulting prompt text and commit the file.

## Maintaining Search Hints

When users repeatedly search for a feature using terms that do not map cleanly to the right service, update the search heuristics in `logic/tools/searchApi.ts` so the agent can discover the relevant APIs more reliably.

## Maintaining Skills

**What:** Step-by-step workflow guides for multi-step tasks (e.g., running a query, creating a schema change).

**Where:** `logic/skills/`

**How to add a new skill:**
1. Create `logic/skills/mySkill.ts` exporting `{ name, description, content }`
2. Register it in `logic/skills/index.ts`
3. Update the `get_skill` tool description in `logic/tools/index.ts` to list the new skill

## Maintaining Route List

**What:** The `navigate` tool supports `list=true` which returns all valid route patterns from Vue Router at runtime. No maintenance needed — it reads directly from the router instance.

## Regenerating the OpenAPI Index

```bash
pnpm --dir frontend run generate:openapi-index
```

This reads `backend/api/mcp/gen/openapi.yaml` and produces `gen/openapi-index.ts` with all endpoint definitions and schemas. Run after proto changes that affect the API.

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
│       └── gen/         # Generated files (do not edit by hand)
│           ├── openapi-index.ts     # API endpoint and schema index
│           └── service-directory.ts # LLM-generated service directory
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

**What:** A compact, feature-grouped summary of all API services. Injected into the system prompt so the agent knows which service to target without keyword-searching.

**Where:** `logic/tools/gen/service-directory.ts`

**When to update:**
- After adding or removing API services
- After significant endpoint additions that change what a service covers
- When the generator script warns about uncovered services

**How to update:**
1. Run `pnpm --dir frontend run generate:openapi-index` — check for service coverage warnings
2. Run `pnpm --dir frontend run generate:service-directory` — regenerates using an LLM
3. Review the output in `gen/service-directory.ts` and commit

The directory generation script (`scripts/generate_service_directory.js`) feeds the full endpoint list to an LLM and asks it to produce a grouped directory. The LLM prompt is in the script.

## Maintaining the Concept Alias Map

**What:** A mapping from user-facing feature names to API service names and keywords. Lives in `searchApi.ts` as `CONCEPT_ALIASES`.

**When to update:** When users report that searching for a feature by its common name returns no results.

**How to update:** Add an entry mapping the user's term to the relevant service(s) and keywords:
```ts
"feature name": ["ServiceName", "keyword1", "keyword2"],
```

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

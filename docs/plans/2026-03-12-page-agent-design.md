# Bytebase Page Agent Design

This document describes the page agent as currently implemented. It is the durable design reference for the in-page agent and intentionally omits earlier phase-by-phase implementation planning.

## Goal

Provide an app-wide AI assistant inside the Bytebase console that can:

- understand the current page,
- navigate the product,
- use Bytebase APIs,
- interact with the DOM when page-local UI state matters, and
- load reusable workflow guidance for common tasks.

The page agent complements the existing SQL-editor AI experience; it does not replace it.

## Non-goals

- controlling non-Bytebase pages,
- relying on a browser extension or external page controller,
- changing production behavior independently of the normal UI and API contracts.

## Architecture

The implementation has four main parts.

### 1. App-wide UI surface

The floating agent window is mounted from `frontend/src/layouts/BodyLayout.vue`, so it is available across the dashboard instead of being tied to a single page.

Entry points:

- `frontend/src/layouts/BodyLayout.vue` mounts `AgentWindow` and registers the keyboard shortcut.
- `frontend/src/views/DashboardHeader.vue` exposes the header toggle button.
- `frontend/src/plugins/agent/index.ts` defines the `Ctrl/Cmd + Shift + A` shortcut.
- `frontend/src/plugins/agent/store/agent.ts` stores UI state and conversation state.

The Pinia store persists the conversation history plus the window layout state used by the floating window (position and size) in `localStorage`.

### 2. Frontend agent runtime

The agent loop runs in the frontend in `frontend/src/plugins/agent/logic/agentLoop.ts`.

High-level flow:

1. Build the system prompt.
2. Send conversation messages plus tool definitions to `AIService.Chat`.
3. Receive assistant text and/or tool calls.
4. Execute tool calls locally in the browser.
5. Append tool results to the conversation.
6. Repeat until the model returns a final text response.

This keeps page-aware operations in the frontend while the backend remains the provider-facing proxy.

### 3. Tool layer

The implemented tool set is six tools:

1. `search_api`
2. `call_api`
3. `navigate`
4. `get_page_state`
5. `dom_action`
6. `get_skill`

Tool definitions live in `frontend/src/plugins/agent/logic/tools/index.ts`, and tool execution is local to the page agent runtime.

### 4. Backend AI proxy

The backend contract is defined in `proto/v1/v1/ai_service.proto`, with backend handling in `backend/api/v1/ai_service.go`.

The backend is the source of truth for provider integration and normalizes tool-calling responses from supported AI providers. The frontend never talks directly to model providers.

## Tool model

### `search_api`

`search_api` is a structured OpenAPI index browser, not a free-text keyword search tool.

Current modes:

- list available services,
- browse endpoints for a service,
- inspect an endpoint by `operationId`,
- inspect a schema by name.

The implementation is backed by the generated OpenAPI index used by the frontend tool code. The expected workflow is:

1. identify a service from the API directory in the prompt,
2. call `search_api(service="...")`,
3. call `search_api(operationId="...")`,
4. then call `call_api(...)`.

### `call_api`

`call_api` executes a Bytebase API operation by `operationId` with an optional JSON body. It is the direct bridge from the agent to Bytebase APIs already available to the current signed-in user.

### `navigate`

`navigate` uses Vue Router to either:

- navigate to a concrete path, or
- list known route patterns with `list=true`.

The prompt instructs the model to list routes first when unsure instead of guessing paths.

### `get_page_state`

`get_page_state` is the read tool for current-page context.

Current modes:

- default `semantic` mode returns route information plus structured context when available,
- `mode: "dom"` returns the same base page state plus indexed DOM information.

There is no standalone `get_dom_tree` tool. DOM inspection is part of `get_page_state(mode="dom")`.

In semantic mode the implementation currently extracts a narrow context set:

- `user`
- `project`
- `database`
- `issue`

Those values are populated from route-aware store lookups in `frontend/src/plugins/agent/logic/context.ts`.

### `dom_action`

`dom_action` is the browser-side UI interaction tool. It supports the implemented action types used by the agent runtime, including reading indexed DOM elements and performing interactions such as click, input, select, and scroll.

The intended workflow is:

1. call `get_page_state(mode="dom")`,
2. inspect element indices,
3. call `dom_action(...)`.

### `get_skill`

`get_skill` loads reusable workflow guidance shipped with the page agent.

Current skills are:

- `query`
- `database-change`
- `grant-permission`

This keeps step-by-step workflow guidance out of the main prompt until it is needed.

## Prompt and context model

Prompt construction lives in `frontend/src/plugins/agent/logic/prompt.ts`.

The current system prompt includes:

- assistant identity and safety/usage rules,
- an API service directory,
- compact Bytebase domain concepts,
- dynamic page information including current path, page title, and role when available.

Important current behavior:

- dynamic route/page context is folded into the system prompt,
- it is not appended as a separate user message,
- the prompt tells the model to call `get_page_state` first,
- the prompt tells the model to use `get_skill` before common multi-step workflows,
- tool choice is context-sensitive rather than universally API-first.

The implemented guidance prefers DOM interaction on pages with unsaved or in-progress UI state, and prefers API access for persisted data, cross-resource lookup, or bulk operations.

## AI service contract

`proto/v1/v1/ai_service.proto` is the source of truth.

At a high level:

- `AIService.Chat` accepts conversation `messages` and `tool_definitions`.
- Message roles use the `AIChatMessageRole` enum rather than raw role strings.
- Assistant messages can carry `tool_calls`.
- Tool result messages carry `tool_call_id`.
- Tool calls include opaque provider metadata, which the frontend preserves when echoing tool results back through the loop.

This contract is what the frontend agent loop serializes to in `agentLoop.ts`.

## Durable implementation map

Primary implementation files:

- `frontend/src/layouts/BodyLayout.vue`
- `frontend/src/views/DashboardHeader.vue`
- `frontend/src/plugins/agent/index.ts`
- `frontend/src/plugins/agent/store/agent.ts`
- `frontend/src/plugins/agent/logic/agentLoop.ts`
- `frontend/src/plugins/agent/logic/prompt.ts`
- `frontend/src/plugins/agent/logic/context.ts`
- `frontend/src/plugins/agent/logic/tools/index.ts`
- `frontend/src/plugins/agent/logic/tools/searchApi.ts`
- `frontend/src/plugins/agent/logic/tools/pageState.ts`
- `proto/v1/v1/ai_service.proto`

## Notes on scope

This document intentionally excludes older phase plans, handoff notes, and exploratory design alternatives that were useful during implementation but are no longer the durable source of truth.
